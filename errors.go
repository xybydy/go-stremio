package stremio

import (
	"errors"
)

var (
	// ErrBadRequest signals that the client sent a bad request.
	// It leads to a "400 Bad Request" response.
	ErrBadRequest = errors.New("bad request")
	// ErrNotFound signals that the catalog/meta/stream was not found.
	// It leads to a "404 Not Found" response.
	ErrNotFound = errors.New("not found")

	ErrNoMeta = errors.New("no meta in context")
)
