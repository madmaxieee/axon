package client

import (
	"context"

	"github.com/madmaxieee/axon/internal/proto"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type Client struct {
	*openai.Client
	opts ClientOptions
}

// TODO: support more configurable client options
type ClientOptions struct {
	ProviderName string
	ModelName    string
	BaseURL      string
	APIKey       string
}

func NewClient(opts ClientOptions) *Client {
	var client openai.Client
	client = openai.NewClient(
		option.WithBaseURL(opts.BaseURL),
		option.WithAPIKey(opts.APIKey),
	)
	return &Client{
		Client: &client,
		opts:   opts,
	}
}

func (c *Client) Request(ctx context.Context, request proto.Request) *Stream {
	params := openai.ChatCompletionNewParams{
		Messages: request.Messages,
		Model:    c.opts.ModelName,
	}

	stream := c.Chat.Completions.NewStreaming(ctx, params)
	return NewStream(stream)
}
