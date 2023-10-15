//go:generate go run github.com/google/wire/cmd/wire gen .
package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/totegamma/concurrent/x/auth"
	"github.com/totegamma/concurrent/x/core"
	"github.com/totegamma/concurrent/x/util"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"gorm.io/plugin/opentelemetry/tracing"
)

func main() {

	fmt.Print(concurrentBanner)

	e := echo.New()
	config := util.Config{}
	configPath := os.Getenv("CONCURRENT_CONFIG")
	if configPath == "" {
		configPath = "/etc/concurrent/config.yaml"
	}

	err := config.Load(configPath)
	if err != nil {
		e.Logger.Fatal(err)
	}

	log.Print("Concurrent ", util.GetFullVersion(), " starting...")
	log.Print("Config loaded! I am: ", config.Concurrent.CCID)

	logfile, err := os.OpenFile(filepath.Join(config.Server.LogPath, "api-access.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer logfile.Close()

	// e.Logger.SetOutput(logfile)

	e.HidePort = true
	e.HideBanner = true

	if config.Server.EnableTrace {
		cleanup, err := setupTraceProvider(config.Server.TraceEndpoint, config.Concurrent.FQDN+"/ccapi", util.GetFullVersion())
		if err != nil {
			panic(err)
		}
		defer cleanup()

		skipper := otelecho.WithSkipper(
			func(c echo.Context) bool {
				return c.Path() == "/metrics" || c.Path() == "/health"
			},
		)
		e.Use(otelecho.Middleware("api", skipper))
	}

	e.Use(echoprometheus.NewMiddleware("ccapi"))
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	db, err := gorm.Open(postgres.Open(config.Server.Dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	sqlDB, err := db.DB() // for pinging
	if err != nil {
		panic("failed to connect database")
	}
	defer sqlDB.Close()

	err = db.Use(tracing.NewPlugin(
		tracing.WithDBName("postgres"),
	))
	if err != nil {
		panic("failed to setup tracing plugin")
	}

	// Migrate the schema
	log.Println("start migrate")
	db.AutoMigrate(
		&core.Message{},
		&core.Character{},
		&core.Association{},
		&core.Stream{},
		&core.Domain{},
		&core.Entity{},
		&core.Collection{},
		&core.CollectionItem{},
		&core.Ack{},
	)

	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Server.RedisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	err = redisotel.InstrumentTracing(
		rdb,
		redisotel.WithAttributes(
			attribute.KeyValue{
				Key:   "db.name",
				Value: attribute.StringValue("redis"),
			},
		),
	)
	if err != nil {
		panic("failed to setup tracing plugin")
	}

	agent := SetupAgent(db, rdb, config)

	socketHandler := SetupSocketHandler(rdb, config)
	messageHandler := SetupMessageHandler(db, rdb, config)
	characterHandler := SetupCharacterHandler(db, config)
	associationHandler := SetupAssociationHandler(db, rdb, config)
	streamHandler := SetupStreamHandler(db, rdb, config)
	domainHandler := SetupDomainHandler(db, config)
	entityHandler := SetupEntityHandler(db, rdb, config)
	authHandler := SetupAuthHandler(db, config)
	userkvHandler := SetupUserkvHandler(db, rdb, config)
	collectionHandler := SetupCollectionHandler(db, rdb, config)

	authService := SetupAuthService(db, config)

	apiV1 := e.Group("")
	apiV1.GET("/message/:id", messageHandler.Get)
	apiV1.GET("/characters", characterHandler.Get)
	apiV1.GET("/association/:id", associationHandler.Get)
	apiV1.GET("/stream/:id", streamHandler.Get)
	apiV1.GET("/streams", streamHandler.List)
	apiV1.GET("/streams/recent", streamHandler.Recent)
	apiV1.GET("/streams/range", streamHandler.Range)
	apiV1.GET("/socket", socketHandler.Connect)
	apiV1.GET("/domain", domainHandler.Profile)
	apiV1.GET("/domain/:id", domainHandler.Get)
	apiV1.GET("/domains", domainHandler.List)
	apiV1.GET("/entity/:id", entityHandler.Get)
	apiV1.GET("/entities", entityHandler.List)
	apiV1.GET("/auth/claim", authHandler.Claim)
	apiV1.GET("/profile", func(c echo.Context) error {
		profile := config.Profile
		profile.Registration = config.Concurrent.Registration
		profile.Version = util.GetVersion()
		profile.Hash = util.GetGitHash()
		profile.SiteKey = config.Server.CaptchaSitekey
		return c.JSON(http.StatusOK, profile)
	})

	apiV1R := apiV1.Group("", auth.JWT)
	apiV1R.PUT("/domain", domainHandler.Upsert, authService.Restrict(auth.ISADMIN))
	apiV1R.DELETE("/domain/:id", domainHandler.Delete, authService.Restrict(auth.ISADMIN))
	apiV1R.POST("/domains/hello", domainHandler.Hello, authService.Restrict(auth.ISUNUNITED))
	apiV1R.GET("/admin/sayhello/:fqdn", domainHandler.SayHello, authService.Restrict(auth.ISADMIN))

	apiV1R.POST("/entity", entityHandler.Register, authService.Restrict(auth.ISUNKNOWN))
	apiV1R.DELETE("/entity/:id", entityHandler.Delete, authService.Restrict(auth.ISADMIN))
	apiV1R.PUT("/entity/:id", entityHandler.Update, authService.Restrict(auth.ISADMIN))
    apiV1R.POST("/ack", entityHandler.Ack, authService.Restrict(auth.ISLOCAL))
    apiV1R.DELETE("/ack", entityHandler.Unack, authService.Restrict(auth.ISLOCAL))
	apiV1R.POST("/admin/entity", entityHandler.Create, authService.Restrict(auth.ISADMIN))

	apiV1R.POST("/message", messageHandler.Post, authService.Restrict(auth.ISLOCAL))
	apiV1R.DELETE("/message/:id", messageHandler.Delete, authService.Restrict(auth.ISLOCAL))

	apiV1R.PUT("/character", characterHandler.Put, authService.Restrict(auth.ISLOCAL))

	apiV1R.POST("/association", associationHandler.Post, authService.Restrict(auth.ISKNOWN))
	apiV1R.DELETE("/association/:id", associationHandler.Delete, authService.Restrict(auth.ISKNOWN))

	apiV1R.POST("/stream", streamHandler.Create, authService.Restrict(auth.ISLOCAL))
	apiV1R.PUT("/stream/:id", streamHandler.Update, authService.Restrict(auth.ISLOCAL))
	apiV1R.POST("/streams/checkpoint", streamHandler.Checkpoint, authService.Restrict(auth.ISUNITED))
	apiV1R.DELETE("/stream/:id", streamHandler.Delete, authService.Restrict(auth.ISLOCAL))
	apiV1R.DELETE("/stream/:stream/:element", streamHandler.Remove, authService.Restrict(auth.ISLOCAL))
	apiV1.GET("/streams/mine", streamHandler.ListMine)

	apiV1R.GET("/kv/:key", userkvHandler.Get, authService.Restrict(auth.ISLOCAL))
	apiV1R.PUT("/kv/:key", userkvHandler.Upsert, authService.Restrict(auth.ISLOCAL))

	apiV1R.POST("/collection", collectionHandler.CreateCollection, authService.Restrict(auth.ISLOCAL))
	apiV1R.GET("/collection/:id", collectionHandler.GetCollection)
	apiV1R.PUT("/collection/:id", collectionHandler.UpdateCollection, authService.Restrict(auth.ISLOCAL))
	apiV1R.DELETE("/collection/:id", collectionHandler.DeleteCollection, authService.Restrict(auth.ISLOCAL))

	apiV1R.POST("/collection/:collection", collectionHandler.CreateItem, authService.Restrict(auth.ISLOCAL))
	apiV1R.GET("/collection/:collection/:item", collectionHandler.GetItem)
	apiV1R.PUT("/collection/:collection/:item", collectionHandler.UpdateItem, authService.Restrict(auth.ISLOCAL))
	apiV1R.DELETE("/collection/:collection/:item", collectionHandler.DeleteItem, authService.Restrict(auth.ISLOCAL))

	e.GET("/health", func(c echo.Context) (err error) {
		ctx := c.Request().Context()

		err = sqlDB.Ping()
		if err != nil {
			return c.String(http.StatusInternalServerError, "db error")
		}

		err = rdb.Ping(ctx).Err()
		if err != nil {
			return c.String(http.StatusInternalServerError, "redis error")
		}

		return c.String(http.StatusOK, "ok")
	})

	e.GET("/metrics", echoprometheus.NewHandler())

	agent.Boot()

	e.Logger.Fatal(e.Start(":8000"))
}

func setupTraceProvider(endpoint string, serviceName string, serviceVersion string) (func(), error) {

	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)

	if err != nil {
		return nil, err
	}

	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
		semconv.ServiceVersionKey.String(serviceVersion),
	)

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource),
	)
	otel.SetTracerProvider(tracerProvider)

	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagator)

	cleanup := func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := tracerProvider.Shutdown(ctx); err != nil {
			log.Printf("Failed to shutdown tracer provider: %v", err)
		}
	}
	return cleanup, nil
}
