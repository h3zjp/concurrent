package stream

import (
    "gorm.io/gorm"
    "golang.org/x/exp/slices"
    "github.com/totegamma/concurrent/x/core"
)


// Repository is stream repository
type Repository struct {
    db *gorm.DB
}

// NewRepository is for wire.go
func NewRepository(db *gorm.DB) *Repository {
    return &Repository{db: db}
}

// Get returns a stream by ID
func (r *Repository) Get(key string) (core.Stream, error) {
    var stream core.Stream
    err := r.db.First(&stream, "id = ?", key).Error
    return stream, err
}

// Upsert updates a stream
func (r *Repository) Upsert(stream *core.Stream) error {
    return r.db.Save(&stream).Error
}

// GetList returns list of schemas by schema
func (r *Repository) GetList(schema string) ([]core.Stream, error) {
    var streams []core.Stream
    err := r.db.Where("Schema = ?", schema).Find(&streams).Error
    return streams, err
}


// HasWriteAccess returns true if the user has write access
func (r *Repository) HasWriteAccess(streamID string, userAddress string) bool {
    var stream core.Stream
    r.db.First(&stream, "id = ?", streamID)
    if len(stream.Writer) == 0 {
        return true
    }
    return slices.Contains(stream.Writer, userAddress)
}

// HasReadAccess returns true if the user has read access
func (r *Repository) HasReadAccess(streamID string, userAddress string) bool {
    var stream core.Stream
    r.db.First(&stream, "id = ?", streamID)
    if len(stream.Reader) == 0 {
        return true
    }
    return slices.Contains(stream.Reader, userAddress)
}


