package client

import (
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
			break
		}

		if _, ok := acc.JustFinishedRefusal(); ok {
		}

		if _, ok := acc.JustFinishedToolCall(); ok {
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onChunk(chunk)
		}
	}

	if err := stream.Err(); err != nil {
		return openai.ChatCompletion{}, err
	}

	return acc.ChatCompletion, nil
}
