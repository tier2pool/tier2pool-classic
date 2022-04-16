package jsonrpc

import "encoding/json"

type Request struct {
	ID     int             `json:"id"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Worker string          `json:"worker,omitempty"`
}
