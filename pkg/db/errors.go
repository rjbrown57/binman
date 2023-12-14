package binmandb

import "errors"

var (
	// ErrNilReadResponse is returned when a read operation  returns nil
	ErrNilReadResponse = errors.New("Read Operation returned nil")
	// ErrKeyExists is returned when trying to update a data key that already has data
	// This will only be returned if overwrite is set to false(the default)
	ErrKeyExists = errors.New("Refusing to overwrite exisitng data key")
)
