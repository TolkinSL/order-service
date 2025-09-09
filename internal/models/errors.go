package models

import "errors"

var (
	ErrInvalidOrderUID    = errors.New("invalid order UID")
	ErrInvalidTrackNumber = errors.New("invalid track number")
	ErrEmptyItems         = errors.New("items list is empty")
	ErrOrderNotFound      = errors.New("order not found")
	ErrInvalidJSON        = errors.New("invalid JSON data")
)
