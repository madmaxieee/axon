package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/madmaxieee/axon/internal/proto"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type Client struct {
	*openai.Client
	opts ClientOptions
}

type ClientOptions struct {
	ProviderName string
	ModelName    string
	BaseURL      string
	APIKey       string
}

var clientsMap = make(map[string]*Client)

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

func GetClient(opts ClientOptions) *Client {
	key := opts.ProviderName + "/" + opts.ModelName
	if client, ok := clientsMap[key]; ok {
		return client
	}
	client := NewClient(opts)
	clientsMap[key] = client
	return client
}

func (c *Client) Request(ctx context.Context, request proto.Request) *Stream {
	params := openai.ChatCompletionNewParams{
		Messages: request.Messages,
		Model:    c.opts.ModelName,
	}

	stream := c.Chat.Completions.NewStreaming(ctx, params)
	return NewStream(stream)
}

func ParseModelString(modelStr string) (string, string, error) {
	parts := strings.SplitN(modelStr, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid model string: %s", modelStr)
	}
	// provider, model
	return parts[0], parts[1], nil
}
