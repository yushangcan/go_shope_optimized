package service

import "errors"

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrConflict            = errors.New("conflict")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrNotFound            = errors.New("not found")
	ErrActivityUnavailable = errors.New("activity unavailable")
	ErrOutOfStock          = errors.New("out of stock")
	ErrInvalidOrderStatus  = errors.New("invalid order status")
)
