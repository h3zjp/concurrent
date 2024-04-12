package core

type Commit struct {
	Document  string `json:"document"`
	Signature string `json:"signature"`
	Option    string `json:"option"`
}

type ResponseBase[T any] struct {
	Status  string `json:"status"`
	Content T      `json:"content"`
	Error   string `json:"error,omitempty"`
}
