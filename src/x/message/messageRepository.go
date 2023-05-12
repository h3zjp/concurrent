package message

import (
    "gorm.io/gorm"
    "github.com/totegamma/concurrent/x/association"
)


// Repository is message repository
type Repository struct {
    db *gorm.DB
}

// NewRepository is used for wire.go
func NewRepository(db *gorm.DB) Repository {
    return Repository{db: db}
}

// Create creates new message
func (r *Repository) Create(message *Message) string {
    r.db.Create(&message)
    return message.ID
}

// Get returns a message with associaiton data
func (r *Repository) Get(key string) Message {
    var message Message
    var associations []association.Association
    r.db.First(&message, "id = ?", key)
    r.db.Table("associations").
        Select("associations.*").
        Joins("JOIN messages ON messages.id = associations.target").
        Where("messages.id = ? AND associations.id = ANY(messages.associations)", message.ID).
        Find(&associations)
    message.AssociationsData = associations
    return message
}

// Delete deletes an message
func (r *Repository) Delete(id string) Message {
    var deleted Message
    r.db.Where("id = $1", id).Delete(&deleted)
    return deleted
}

