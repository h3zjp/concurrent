package core

import (
	"time"
)

// commons
type DocumentBase[T any] struct {
	ID           string    `json:"id,omitempty"`
	Signer       string    `json:"signer"`
	Owner        string    `json:"owner,omitempty"`
	Type         string    `json:"type"`
	Schema       string    `json:"schema,omitempty"`
	Policy       string    `json:"policy,omitempty"`
	PolicyParams string    `json:"policyParams,omitempty"`
	KeyID        string    `json:"keyID,omitempty"`
	Body         T         `json:"body,omitempty"`
	Meta         any       `json:"meta,omitempty"`
	SemanticID   string    `json:"semanticID,omitempty"`
	SignedAt     time.Time `json:"signedAt"`
}

// entity
type EntityAffiliation struct { // type: affiliation
	Domain string `json:"domain"`
	DocumentBase[any]
}

type EntityTombstone struct { // type: tombstone
	Reason string `json:"reason"`
	DocumentBase[any]
}

type ExtensionDocument[T any] struct { // type: extension
	DocumentBase[T]
}

// ack
type AckDocument struct { // type: ack
	DocumentBase[any]
	From string `json:"from"`
	To   string `json:"to"`
}

type UnackDocument struct { // type: unack
	DocumentBase[any]
	From string `json:"from"`
	To   string `json:"to"`
}

// message
type CreateMessage[T any] struct { // type: message
	DocumentBase[T]
	Timelines []string `json:"timelines"`
}

type DeleteDocument struct { // type: delete
	DocumentBase[any]
	Target string `json:"target"`
}

// association
type CreateAssociation[T any] struct { // type: association
	DocumentBase[T]
	Timelines []string `json:"timelines"`
	Variant   string   `json:"variant"`
	Target    string   `json:"target"`
}

// profile
type UpsertProfile[T any] struct { // type: profile
	DocumentBase[T]
}

// key
type EnactKey struct { // type: enact
	DocumentBase[any]
	Target string `json:"target"`
	Root   string `json:"root"`
	Parent string `json:"parent"`
}

type RevokeKey struct { // type: revoke
	DocumentBase[any]
	Target string `json:"target"`
}

// timeline
type TimelineDocument[T any] struct { // type: timeline
	DocumentBase[T]
	Indexable   bool `json:"indexable"`
	DomainOwned bool `json:"domainOwned"`
}

// subscription
type SubscriptionDocument[T any] struct { // type: subscription
	DocumentBase[T]
	Indexable   bool `json:"indexable"`
	DomainOwned bool `json:"domainOwned"`
}

type SubscribeDocument[T any] struct { // type: subscribe
	DocumentBase[T]
	Subscription string `json:"subscription"`
	Target       string `json:"target"`
}

type UnsubscribeDocument struct { // type: unsubscribe
	DocumentBase[any]
	Subscription string `json:"subscription"`
	Target       string `json:"target"`
}

type PassportDocument struct {
	DocumentBase[any]
	Domain string `json:"domain"`
	Entity Entity `json:"entity"`
	Keys   []Key  `json:"keys"`
}

type EventDocument struct { // type: event
	DocumentBase[any]
	Timeline  string       `json:"timeline"`
	Item      TimelineItem `json:"item"`
	Resource  any          `json:"resource"`
	Document  string       `json:"document"`
	Signature string       `json:"signature"`
}
