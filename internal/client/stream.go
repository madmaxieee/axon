package client

import (
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/ssestream"
)

type Stream struct {
	ssestream *ssestream.Stream[openai.ChatCompletionChunk]
	acc       openai.ChatCompletionAccumulator
}

func NewStream(stream *ssestream.Stream[openai.ChatCompletionChunk]) *Stream {
	return &Stream{
		ssestream: stream,
		acc:       openai.ChatCompletionAccumulator{},
	}
}

func (s *Stream) Collect(onChunk func(chunk openai.ChatCompletionChunk)) (openai.ChatCompletion, error) {
	stream := s.ssestream
	acc := s.acc

	for stream.Next() {
		chunk := stream.Current()

		acc.AddChunk(chunk)

		if _, ok := acc.JustFinishedContent(); ok {
			_ = fmt.Errorf("finish-event: Content stream finished")
		}

		if refusal, ok := acc.JustFinishedRefusal(); ok {
			_ = fmt.Errorf("finish-event: refusal stream finished: %v", refusal)
		}

		if tool, ok := acc.JustFinishedToolCall(); ok {
			_ = fmt.Errorf("finish-event: tool call stream finished: %v", tool)
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onChunk(chunk)
		}
	}

	return acc.ChatCompletion, stream.Err()
}
