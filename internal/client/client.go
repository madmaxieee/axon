package client

import (
	"context"

	"github.com/madmaxieee/axon/internal/proto"
	"github.com/openai/openai-go/v3"
)

type Client struct {
	*openai.Client
}

func NewClient() *Client {
	client := openai.NewClient()
	return &Client{Client: &client}
}

func (c *Client) Request(ctx context.Context, request proto.Request) *Stream {
	params := openai.ChatCompletionNewParams{
		Messages: request.Messages,
		Seed:     openai.Int(0),
		Model:    openai.ChatModelGPT4o,
	}

	stream := c.Chat.Completions.NewStreaming(ctx, params)
	return NewStream(stream)
}
