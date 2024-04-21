package auth

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"log"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/totegamma/concurrent/x/core"
	"github.com/totegamma/concurrent/x/util"

	"github.com/totegamma/concurrent/internal/testutil"
	"github.com/totegamma/concurrent/x/domain/mock"
	"github.com/totegamma/concurrent/x/entity/mock"
	"github.com/totegamma/concurrent/x/jwt"
	"github.com/totegamma/concurrent/x/key/mock"
)

const (
	User1ID   = "con1mu9xruulec4y6hd0d369sdf325l94z4770m33d"
	User1Priv = "3fcfac6c211b743975de2d7b3f622c12694b8125daf4013562c5a1aefa3253a5"

	SubKey1ID   = "cck1ydda2qj3nr32hulm65vj2g746f06hy36wzh9ke"
	SubKey1Priv = "1ca30329e8d35217b2328bacfc21c5e3d762713edab0252eead1f4c1ac0b4d81"

	RemoteDomainFQDN = "remote.example.com"
	RemoteDomainCCID = "con1er7kuzrw6vtv6nrq98d4jg7n2r0ayz772zvwxz"
	RemoteDomainPriv = "863183823d2c2a19101140eef0f905c872de1dae6470c9129a1547f3482cb612"
)

func createJwt(t *testing.T, priv string, claims jwt.Claims) string {
	jwt, err := jwt.Create(claims, priv)
	if !assert.NoError(t, err) {
		log.Fatal(err)
	}

	return jwt
}

var checker *tracetest.InMemoryExporter

func TestMain(m *testing.M) {
	checker = testutil.SetupMockTraceProvider()
	m.Run()
}

func TestLocalRootSuccess(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEntity := mock_entity.NewMockService(ctrl)
	mockEntity.EXPECT().Get(gomock.Any(), gomock.Any()).Return(core.Entity{
		ID:     User1ID,
		Domain: "local.example.com",
	}, nil).AnyTimes()
	mockDomain := mock_domain.NewMockService(ctrl)
	mockKey := mock_key.NewMockService(ctrl)

	config := util.Config{
		Concurrent: util.Concurrent{
			FQDN: "local.example.com",
		},
	}

	service := NewService(config, mockEntity, mockDomain, mockKey)

	c, req, rec, traceID := testutil.CreateHttpRequest()

	jwt := createJwt(t, User1Priv, jwt.Claims{
		Issuer:   User1ID,
		Subject:  "concrnt",
		Audience: "local.example.com",
	})

	req.Header.Set("Authorization", "Bearer "+jwt)

	h := service.IdentifyIdentity(func(c echo.Context) error {
		return nil
	})

	err := h(c)
	log.Println(rec.Body.String())
	if assert.NoError(t, err) {
		assert.Equal(t, core.LocalUser, c.Get(core.RequesterTypeCtxKey))
		assert.Equal(t, User1ID, c.Get(core.RequesterIdCtxKey))
		tags := c.Get(core.RequesterTagCtxKey).(core.Tags)
		tagString := tags.ToString()
		assert.Equal(t, "", tagString)
		assert.Equal(t, nil, c.Get(core.RequesterDomainCtxKey))
		assert.Equal(t, nil, c.Get(core.RequesterDomainTagsKey))
		assert.Equal(t, nil, c.Get(core.RequesterKeychainKey))
		assert.Equal(t, nil, c.Get(core.CaptchaVerifiedKey))
	} else {
		testutil.PrintSpans(checker.GetSpans(), traceID)
	}
}

func TestRemoteRootSuccess(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEntity := mock_entity.NewMockService(ctrl)
	mockEntity.EXPECT().Affiliation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(core.Entity{
		ID:     User1ID,
		Domain: RemoteDomainFQDN,
		Score:  0,
	}, nil)
	mockEntity.EXPECT().UpdateScore(gomock.Any(), User1ID, 100).Return(nil)

	mockDomain := mock_domain.NewMockService(ctrl)
	mockDomain.EXPECT().GetByFQDN(gomock.Any(), RemoteDomainFQDN).Return(core.Domain{
		ID:   RemoteDomainFQDN,
		CCID: RemoteDomainCCID,
	}, nil)

	mockKey := mock_key.NewMockService(ctrl)

	config := util.Config{
		Concurrent: util.Concurrent{
			FQDN: "local.example.com",
		},
	}

	service := NewService(config, mockEntity, mockDomain, mockKey)
	c, req, rec, traceID := testutil.CreateHttpRequest()

	fmt.Print("traceID: ", traceID, "\n")

	jwt := createJwt(t, User1Priv, jwt.Claims{
		Issuer:   User1ID,
		Subject:  "concrnt",
		Audience: "local.example.com",
	})

	req.Header.Set("Authorization", "Bearer "+jwt)

	passportDoc := core.PassportDocument{
		Domain: RemoteDomainFQDN,
		Entity: core.Entity{
			ID:     User1ID,
			Domain: RemoteDomainFQDN,
			Tag:    "_admin, _root",
			Score:  100,
		},
		Keys: []core.Key{},
	}

	passportDocJson, _ := json.Marshal(passportDoc)
	signatureBytes, _ := util.SignBytes(passportDocJson, RemoteDomainPriv)
	signature := hex.EncodeToString(signatureBytes)

	fmt.Println("signature: ", signature)

	passport := core.Passport{
		Document:  string(passportDocJson),
		Signature: string(signature),
	}

	passportJson, _ := json.Marshal(passport)

	req.Header.Set("passport", string(passportJson))

	h := service.IdentifyIdentity(func(c echo.Context) error {
		return nil
	})

	err := h(c)
	log.Println(rec.Body.String())

	testutil.PrintSpans(checker.GetSpans(), traceID)
	if assert.NoError(t, err) {
		assert.Equal(t, core.RemoteUser, c.Get(core.RequesterTypeCtxKey))
		assert.Equal(t, User1ID, c.Get(core.RequesterIdCtxKey))
		tags := c.Get(core.RequesterTagCtxKey).(core.Tags)
		tagString := tags.ToString()
		assert.Equal(t, "", tagString)
		assert.Equal(t, RemoteDomainFQDN, c.Get(core.RequesterDomainCtxKey))
		domainTags := c.Get(core.RequesterDomainTagsKey).(core.Tags)
		domainTagString := domainTags.ToString()
		assert.Equal(t, "", domainTagString)
		assert.Len(t, c.Get(core.RequesterKeychainKey).([]core.Key), 0)
		assert.Equal(t, nil, c.Get(core.CaptchaVerifiedKey))
	}

	log.Println(traceID)

}