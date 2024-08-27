package core

import ()

// Event is websocket root packet model
type Event struct {
	Timeline  string       `json:"timeline"` // stream full id (ex: <streamID>@<domain>)
	Item      TimelineItem `json:"item,omitempty"`
	Resource  any          `json:"resource,omitempty"`
	Document  string       `json:"document"`
	Signature string       `json:"signature"`
}

type Chunk struct {
	Key   string         `json:"key"`
	Chunk string         `json:"chunk"`
	Items []TimelineItem `json:"items"`
}

type RequestContext struct {
	Requester       Entity
	RequesterDomain Domain
	Document        any
	Self            any
	Resource        any
	Params          map[string]any
}

type PolicyDocument struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Versions    map[string]Policy `json:"versions"`
}

type Policy struct {
	Statements map[string]Statement `json:"statements"`
	Defaults   map[string]bool      `json:"defaults"`
}

type Statement struct {
	Dominant       bool `json:"dominant"`
	DefaultOnTrue  bool `json:"defaultOnTrue"`
	DefaultOnFalse bool `json:"defaultOnFalse"`
	Condition      Expr `json:"condition"`
}

type Expr struct {
	Operator string `json:"op"`
	Args     []Expr `json:"args"`
	Constant any    `json:"const"`
}

type EvalResult struct {
	Operator string       `json:"op"`
	Args     []EvalResult `json:"args"`
	Result   any          `json:"result"`
	Error    string       `json:"error"`
}

type Config struct {
	FQDN         string `yaml:"fqdn"`
	PrivateKey   string `yaml:"privatekey"`
	Registration string `yaml:"registration"` // open, invite, close
	SiteKey      string `yaml:"sitekey"`
	Dimension    string `yaml:"dimension"`
	CCID         string `yaml:"ccid"`
}

type ConfigInput struct {
	FQDN         string `yaml:"fqdn"`
	PrivateKey   string `yaml:"privatekey"`
	Registration string `yaml:"registration"` // open, invite, close
	SiteKey      string `yaml:"sitekey"`
	Dimension    string `yaml:"dimension"`
}
