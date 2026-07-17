package client

import (
	"errors"
	"fmt"
)

// ErrNotFound is returned when the target OMA-URI does not exist.
var ErrNotFound = errors.New("csp node not found")

// ErrDeferred is returned by transports that only queue operations (such as
// syncml.Recorder) when a read result is requested: the value is not known
// until the batch is delivered to a device.
var ErrDeferred = errors.New("csp read deferred: transport queues operations for later delivery")

// StatusError carries an OMA-DM (SyncML) status code returned by a device
// or management server, e.g. 404 (not found) or 405 (command not allowed).
type StatusError struct {
	Code int
	URI  string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("csp operation on %s failed with SyncML status %d", e.URI, e.Code)
}

// Unwrap maps well-known SyncML statuses onto sentinel errors, so
// errors.Is(err, ErrNotFound) works on 404 responses.
func (e *StatusError) Unwrap() error {
	if e.Code == 404 {
		return ErrNotFound
	}
	return nil
}
