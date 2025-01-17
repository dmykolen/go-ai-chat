package models

import (
	"errors"
	"fmt"
)

var (
	RespInvalidCredentials = Response{Code: 401, Message: "Invalid credentials"}
	RespErrDB              = Response{Code: -2, Message: "Error while DB operation"}
	RespOK                 = Response{Code: 200, Message: "OK"}
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func Success(data interface{}) *Response {
	return &Response{Code: 0, Message: "success", Data: data}
}

func Error(error string) *Response {
	return &Response{Code: -1, Message: "error", Error: error}
}

func R(code int, message string, data ...interface{}) *Response {
	if len(data) > 0 {
		return &Response{Code: code, Message: message, Data: data[0]}
	}
	return &Response{Code: code, Message: message}
}

func RErr(err string, data error) *Response {
	return &Response{Code: -1, Error: err, Data: data.Error()}
}

func (r *Response) WithData(data interface{}) *Response {
	r.Data = data
	return r
}

// Sentinel error for a specific rendering error condition
var ErrTemplateNotFound = errors.New("template not found")

// Define RenderingError for errors during template rendering
type RenderingError struct{ OriginalError error }
type NotFoundError struct{ Message string }
type InternalError struct{ Message string }

func NewRenderingError(err error) error { return &RenderingError{OriginalError: err} }
func (e *RenderingError) Error() string { return fmt.Sprintf("rendering error: %v", e.OriginalError) }
func (e *NotFoundError) Error() string  { return e.Message }
func (e *InternalError) Error() string  { return e.Message }

// AIRequest represents the structure of the request body for user AI requests.
// @Description Request body for user AI requests
type AIRequest struct {
	Question string `json:"userRequest" example:"What is the weather today?"`
	ChatId   string `json:"chatId,omitempty" example:"123e4567-e89b-12d3-a456-426614174000"` // @Description Chat ID associated with the AI request, can be empty if creating a new chat
	TabId    string `json:"tabId,omitempty"`
}

// AIResponse represents the structure of the response body for user AI requests.
// @Description Response body for user AI requests
type AIResponse struct {
	Status string `json:"status" example:"success"`                              // @Description Status of the AI request
	ChatID string `json:"chatId" example:"123e4567-e89b-12d3-a456-426614174000"` // @Description Chat ID associated with the AI request
}
