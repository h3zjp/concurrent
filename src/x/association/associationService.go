package association

import (
    "log"
    "encoding/json"
    "github.com/totegamma/concurrent/x/util"
    "github.com/totegamma/concurrent/x/stream"
    "github.com/totegamma/concurrent/x/socket"
)

// Service is association service
type Service struct {
    repo Repository
    stream stream.Service
    socket *socket.Service
}

// NewService is used for wire.go
func NewService(repo Repository, stream stream.Service, socket*socket.Service) Service {
    return Service{repo: repo, stream: stream, socket: socket}
}

// PostAssociation creates new association
func (s *Service) PostAssociation(association Association) {
    if err := util.VerifySignature(association.Payload, association.Author, association.Signature); err != nil {
        log.Println("verify signature err: ", err)
        return
    }

    s.repo.Create(&association)
    for _, stream := range association.Streams {
        s.stream.Post(stream, association.ID)
    }

    jsonstr, _ := json.Marshal(StreamEvent{
        Type: "association",
        Action: "create",
        Body: association,
    })
    s.socket.NotifyAllClients(jsonstr)
}

// Get returns an association by ID
func (s *Service) Get(id string) Association {
    return s.repo.Get(id)
}

// GetOwn returns associations by author
func (s *Service) GetOwn(author string) []Association {
    return s.repo.GetOwn(author)
}

// Delete deletes an association by ID
func (s *Service) Delete(id string) {
    deleted := s.repo.Delete(id)
    jsonstr, _ := json.Marshal(StreamEvent{
        Type: "association",
        Action: "delete",
        Body: deleted,
    })
    s.socket.NotifyAllClients(jsonstr)
}

