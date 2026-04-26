package api

type Response struct {
	Data  interface{} `json:"data,omitempty"`
	Error *ErrorInfo  `json:"error,omitempty"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewSuccessResponse(data interface{}) Response {
	return Response{Data: data}
}

func NewErrorResponse(code, message string) Response {
	return Response{Error: &ErrorInfo{Code: code, Message: message}}
}
