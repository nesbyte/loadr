package handlers

import "net/http"

type HeaderSetter interface {
	// Set sets the header entries associated with key to the single element value.
	// Follows the same semantics as w.Header().Set()
	Set(key, value string) HeaderSetter
	// Returns the middleware handler with the headers set
	Middleware() func(http.Handler) http.Handler
}

type headerHandler struct {
	headers []kv
}

type kv struct {
	Key   string
	Value string
}

func NewHeaders() HeaderSetter {
	return &headerHandler{}
}

func (h *headerHandler) Set(key string, value string) HeaderSetter {
	h.headers = append(h.headers, kv{key, value})
	return h
}

func (h *headerHandler) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, header := range h.headers {
				w.Header().Set(header.Key, header.Value)
			}
		})
	}
}
