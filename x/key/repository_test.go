package key

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/totegamma/concurrent/client"
	"github.com/totegamma/concurrent/core"
	"github.com/totegamma/concurrent/internal/testutil"
	"testing"
	"time"
)

func TestRepository(t *testing.T) {

	var ctx = context.Background()

	db, cleanup_db := testutil.CreateDB()
	defer cleanup_db()

	mc, cleanup_mc := testutil.CreateMC()
	defer cleanup_mc()

	client := client.NewClient()
	repo := NewRepository(db, mc, client)

	newkey := core.Key{
		ID:              "cck1v26je8uyhc9x6xgcw26d3cne20s44atr7a94em",
		Root:            "con1fk8zlkrfmens3sgj7dzcu3gsw8v9kkysrf8dt5",
		Parent:          "con1fk8zlkrfmens3sgj7dzcu3gsw8v9kkysrf8dt5",
		EnactDocument:   "{}",                                                                                                                                 //TODO: change to real payload
		EnactSignature:  "8c3e365f8b085d4823eb6c824d0eceeb5a2fc194b4055052260042a3a2d40f88002eb2ccbeac62169f4c579ae1831075e887d6e7a4ac9f0ce6a91306de54ba3301", //TODO: change to real signature
		RevokeDocument:  nil,
		RevokeSignature: nil,
	}

	created, err := repo.Enact(ctx, newkey)
	if assert.NoError(t, err) {
		assert.NotZero(t, created.EnactDocument)
		assert.NotZero(t, created.EnactSignature)
		assert.NotZero(t, created.ID)
	}

	found, err := repo.Get(ctx, created.ID)
	if assert.NoError(t, err) {
		assert.Equal(t, created.ID, found.ID)
	}

	modified, err := repo.Revoke(
		ctx,
		created.ID,
		"{}",
		"413d2b0eddf46846a0f5aa16d5cb94644877a4c17ceb76a7639166ea037166ce0fd16b0555ed9c99803a43ac2b8fa21fad5e66968bed9b10a4e709683abfe3c400",
		time.Now(),
	)

	if assert.NoError(t, err) {
		assert.NotZero(t, modified.RevokeDocument)
		assert.NotZero(t, modified.RevokeSignature)
	}

	found, err = repo.Get(ctx, modified.ID)
	if assert.NoError(t, err) {
		assert.Equal(t, modified.ID, found.ID)
	}
}
