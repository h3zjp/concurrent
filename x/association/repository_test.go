package association

import (
	"context"
	"log"
	"testing"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/stretchr/testify/assert"
	"github.com/totegamma/concurrent/internal/testutil"
	"github.com/totegamma/concurrent/x/core"
	"github.com/totegamma/concurrent/x/schema"
	"gorm.io/gorm"
)

var ctx = context.Background()
var repo Repository
var db *gorm.DB
var mc *memcache.Client

func TestMain(m *testing.M) {
	log.Println("Test Start")

	var cleanup_db func()
	db, cleanup_db = testutil.CreateDB()
	defer cleanup_db()

	var cleanup_mc func()
	mc, cleanup_mc = testutil.CreateMC()
	defer cleanup_mc()

	schemaRepository := schema.NewRepository(db)
	schemaService := schema.NewService(schemaRepository)

	repo = NewRepository(db, mc, schemaService)

	m.Run()

	log.Println("Test End")
}

func TestRepository(t *testing.T) {

	// create dummy message
	message := core.Message{
		ID:        "D895NMA837R0C6B90676P2S1J4",
		Author:    "con18fyqn098jsf6cnw2r8hkjt7zeftfa0vqvjr6fe",
		Schema:    "https://gammalab.net/test-message-schema.json",
		Payload:   "{}",
		Signature: "DUMMY",
	}

	err := db.WithContext(ctx).Create(&message).Error
	assert.NoError(t, err)

	messageTID := "m" + message.ID

	// create association
	like := core.Association{
		ID:        "EQB2YB2Q529837710676PETFAR",
		Author:    "con1n42l2lektua69gvza8xhksq3t2we8nnlkmzct4",
		Schema:    "https://gammalab.net/test-like-schema.json",
		TargetTID: messageTID,
		Payload:   "{}",
		Variant:   "",
		Signature: "DUMMY",
	}
	_, err = repo.Create(ctx, like)
	assert.NoError(t, err)

	emoji1 := core.Association{
		ID:        "5GBDM539MCXKY2GJ0676PETFAR",
		Author:    "con1n42l2lektua69gvza8xhksq3t2we8nnlkmzct4",
		Schema:    "https://gammalab.net/test-emoji-schema.json",
		TargetTID: messageTID,
		Payload:   "{}",
		Variant:   "smile",
		Signature: "DUMMY",
	}
	_, err = repo.Create(ctx, emoji1)
	assert.NoError(t, err)

	emoji2 := core.Association{
		ID:        "1EQW1AEZ3WC1J42C0676PETFAR",
		Author:    "con1n42l2lektua69gvza8xhksq3t2we8nnlkmzct4",
		Schema:    "https://gammalab.net/test-emoji-schema.json",
		TargetTID: messageTID,
		Payload:   "{}",
		Variant:   "ultrafastpolar",
		Signature: "DUMMY",
	}
	_, err = repo.Create(ctx, emoji2)
	assert.NoError(t, err)

	emoji3 := core.Association{
		ID:        "KRE2MN45QXFE3AV20676PETFAR",
		Author:    "con1sh4vuw03nn20hn94tuk7h7u3ne5n20avfl5sjm",
		Schema:    "https://gammalab.net/test-emoji-schema.json",
		TargetTID: messageTID,
		Payload:   "{}",
		Variant:   "ultrafastpolar",
		Signature: "DUMMY",
	}
	_, err = repo.Create(ctx, emoji3)
	assert.NoError(t, err)

	// test GetCountsBySchema
	results, err := repo.GetCountsBySchema(ctx, messageTID)
	if assert.NoError(t, err) {
		assert.Equal(t, 2, len(results))
	}

	// test GetBySchema
	associations, err := repo.GetBySchema(ctx, messageTID, "https://gammalab.net/test-like-schema.json")
	if assert.NoError(t, err) {
		assert.Equal(t, 1, len(associations))
	}
	associations, err = repo.GetBySchema(ctx, messageTID, "https://gammalab.net/test-emoji-schema.json")
	if assert.NoError(t, err) {
		assert.Equal(t, 3, len(associations))
	}

	// test GetCountsBySchemaAndVariant
	results, err = repo.GetCountsBySchemaAndVariant(ctx, messageTID, "https://gammalab.net/test-emoji-schema.json")
	if assert.NoError(t, err) {
		assert.Equal(t, 2, len(results))
	}

	// test GetBySchemaAndVariant
	associations, err = repo.GetBySchemaAndVariant(ctx, messageTID, "https://gammalab.net/test-emoji-schema.json", "ultrafastpolar")
	if assert.NoError(t, err) {
		assert.Equal(t, 2, len(associations))
	}

}
