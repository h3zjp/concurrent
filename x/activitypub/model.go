package activitypub

// ApEntity is a db model of an ActivityPub entity.
type ApEntity struct {
    ID string `json:"id" gorm:"type:text"`
    CCAddr string `json:"ccaddr" gorm:"type:char(42)"`
    Publickey string `json:"publickey" gorm:"type:text"`
    Privatekey string `json:"privatekey" gorm:"type:text"`
}

// ApPerson is a db model of an ActivityPub entity.
type ApPerson struct {
    ID string `json:"id" gorm:"type:text"`
    Name string `json:"name" gorm:"type:text"`
    Summary string `json:"summary" gorm:"type:text"`
    ProfileURL string `json:"profile_url" gorm:"type:text"`
    IconURL string `json:"icon_url" gorm:"type:text"`
}

// WebFinger is a struct for a WebFinger response.
type WebFinger struct {
    Subject string `json:"subject"`
    Links []WebFingerLink `json:"links"`
}

// WebFingerLink is a struct for the links field of a WebFinger response.
type WebFingerLink struct {
    Rel string `json:"rel"`
    Type string `json:"type"`
    Href string `json:"href"`
}

// Person is a struct for an ActivityPub actor.
type Person struct {
    Context interface{} `json:"@context"`
    Type string `json:"type"`
    ID string `json:"id"`
    Inbox string `json:"inbox"`
    Outbox string `json:"outbox"`
    Followers string `json:"followers"`
    Following string `json:"following"`
    Liked string `json:"liked"`
    PreferredUsername string `json:"preferredUsername"`
    Name string `json:"name"`
    Summary string `json:"summary"`
    URL string `json:"url"`
    Icon Icon `json:"icon"`
    PublicKey Key `json:"publicKey"`
}

// Key is a struct for the publicKey field of an actor.
type Key struct {
    ID string `json:"id"`
    Type string `json:"type"`
    Owner string `json:"owner"`
    PublicKeyPem string `json:"publicKeyPem"`
}

// Icon is a struct for the icon field of an actor.
type Icon struct {
    Type string `json:"type"`
    MediaType string `json:"mediaType"`
    URL string `json:"url"`
}

// Create is a struct for an ActivityPub create activity.
type Create struct {
    Context interface{} `json:"@context"`
    Type string `json:"type"`
    ID string `json:"id"`
    Actor string `json:"actor"`
    To []string `json:"to"`
    CC []string `json:"cc"`
    Object Object `json:"object"`
    Published string `json:"published"`
    Summary string `json:"summary"`
    Content string `json:"content"`
}

// Object is a struct for an ActivityPub object.
type Object struct {
    Context interface{} `json:"@context"`
    Type string `json:"type"`
    ID string `json:"id"`
    Content string `json:"content"`
    Actor string `json:"actor"`
    Object interface{} `json:"object"`
}

// Accept is a struct for an ActivityPub accept activity.
type Accept struct {
    Context interface{} `json:"@context"`
    Type string `json:"type"`
    ID string `json:"id"`
    Actor string `json:"actor"`
    Object Object `json:"object"`
}


// CreateEntityRequest is a struct for a request to create an entity.
type CreateEntityRequest struct {
    ID string `json:"id"`
}

