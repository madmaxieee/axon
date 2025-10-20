package proto

import "github.com/openai/openai-go/v3"

type Request struct {
	Messages       []openai.ChatCompletionMessageParamUnion
	ResponseFormat *string
	Temperature    *float64
	TopP           *float64
	TopK           *int64
	Stop           []string
	MaxTokens      *int64
}
