package urit

import (
	"errors"
)

type HeadersOption interface {
	GetHeaders() (map[string]string, error)
}

type Headers interface {
	HeadersOption
	Set(key string, value interface{}) Headers
	Get(key string) (interface{}, bool)
	Has(key string) bool
	Del(key string) Headers
	Clone() Headers
}

func NewHeaders(namesAndValues ...interface{}) (Headers, error) {
	if len(namesAndValues)%2 != 0 {
		return nil, errors.New("must be a value for each name")
	}
	result := &headers{
		entries: map[string]interface{}{},
	}
	for i := 0; i < len(namesAndValues)-1; i += 2 {
		if k, ok := namesAndValues[i].(string); ok {
			result.entries[k] = namesAndValues[i+1]
		} else {
			return nil, errors.New("name must be a string")
		}
	}
	return result, nil
}

type headers struct {
	entries map[string]interface{}
}

func (h *headers) GetHeaders() (map[string]string, error) {
	result := map[string]string{}
	for k, v := range h.entries {
		if str, err := getValue(v); err == nil {
			result[k] = str
		} else {
			return result, err
		}
	}
	return result, nil
}

func (h *headers) Set(key string, value interface{}) Headers {
	h.entries[key] = value
	return h
}

func (h *headers) Get(key string) (interface{}, bool) {
	v, ok := h.entries[key]
	return v, ok
}

func (h *headers) Has(key string) bool {
	_, ok := h.entries[key]
	return ok
}

func (h *headers) Del(key string) Headers {
	delete(h.entries, key)
	return h
}

func (h *headers) Clone() Headers {
	result := &headers{
		entries: map[string]interface{}{},
	}
	for k, v := range h.entries {
		result.entries[k] = v
	}
	return result
}
