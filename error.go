package main

import (
	"errors"
	"fmt"
)

var (
	ErrResponseIsEmpty error = errors.New("response is empty")
)

type ErrorResponse struct {
	Code      PaymentStatusCode `json:"code"`
	Message   string            `json:"message"`
	Param     string            `json:"param"`
	PaymentID string            `json:"payment_id"`
	Type      string            `json:"type"`
}

func (e *ErrorResponse) ErrorCode() error {
	return fmt.Errorf(string(e.Code))
}
