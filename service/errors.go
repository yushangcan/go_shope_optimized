package service

import "errors"

// Service 层用固定错误表达业务结果；router 会把它们转换为对应 HTTP 状态码。
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
