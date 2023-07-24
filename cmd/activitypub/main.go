package main

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/totegamma/concurrent/x/activitypub"
	"github.com/totegamma/concurrent/x/util"
	"github.com/totegamma/concurrent/x/auth"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/plugin/opentelemetry/tracing"
)

func main() {
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
	log.Print("Config loaded! I am: ", config.Concurrent.CCAddr)

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
		e.Use(otelecho.Middleware("dev", skipper))

		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				span := trace.SpanFromContext(c.Request().Context())
				c.Response().Header().Set("trace-id", span.SpanContext().TraceID().String())
				return next(c)
			}
		})
	}

	e.Use(echoprometheus.NewMiddleware("ccapi"))
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Binder = &activitypub.Binder{}

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
		&activitypub.ApEntity{},
		&activitypub.ApPerson{},
		&activitypub.ApFollow{},
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

	authService := SetupAuthService(db, config)
	activitypubHandler := SetupActivitypubHandler(db, rdb, config)

	e.GET("/.well-known/webfinger", activitypubHandler.WebFinger)
	e.GET("/.well-known/nodeinfo", activitypubHandler.NodeInfoWellKnown)

	ap := e.Group("/ap")
	ap.GET("/nodeinfo/2.0", activitypubHandler.NodeInfo)
	ap.GET("/acct/:id", activitypubHandler.User)
	ap.POST("/acct/:id/inbox", activitypubHandler.Inbox)
	ap.POST("/acct/:id/outbox", activitypubHandler.PrintRequest)
	ap.GET("/note/:id", activitypubHandler.Note)

	ap.GET("/api/entity/:ccaddr", activitypubHandler.GetEntityID)
	ap.GET("/api/person/:id", activitypubHandler.GetPerson)

	// should be restricted
	apR := ap.Group("", auth.JWT)
	apR.POST("/api/entity", activitypubHandler.CreateEntity, authService.Restrict(auth.ISLOCAL)) // ISLOCAL
	apR.PUT("/api/person", activitypubHandler.UpdatePerson, authService.Restrict(auth.ISLOCAL)) // ISLOCAL

	go activitypubHandler.Boot()

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
