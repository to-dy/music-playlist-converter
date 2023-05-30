package handlers

type APIResponse struct {
	Data interface{} `json:"data,omitempty"`
}

type ErrorObject struct {
	Status int          `json:"status,omitempty"`
	Title  string       `json:"title,omitempty"`
	Detail string       `json:"detail,omitempty"`
	Source *ErrorSource `json:"source,omitempty"`
	Meta   interface{}  `json:"meta,omitempty"`
}

type ErrorSource struct {
	Pointer   string `json:"pointer,omitempty"`
	Parameter string `json:"parameter,omitempty"`
}

type Errors []*ErrorObject

type ErrorResponse struct {
	Errors Errors `json:"errors"`
}
