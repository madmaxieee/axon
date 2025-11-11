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

type TemplateArgs map[string]string

func (m TemplateArgs) Get(key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	return v
}

type Flags struct {
	ConfigFilePath string
	Pattern        string
	Explain        bool
	ShowLast       bool
	Model          string
	Replay         bool
	Quiet          bool
}
