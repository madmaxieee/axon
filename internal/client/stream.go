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

		// When this fires, the current chunk value will not contain content data
		if _, ok := acc.JustFinishedContent(); ok {
			println()
			println("finish-event: Content stream finished")
		}

		if refusal, ok := acc.JustFinishedRefusal(); ok {
			println()
			println("finish-event: refusal stream finished:", refusal)
			println()
		}

		if tool, ok := acc.JustFinishedToolCall(); ok {
			println("finish-event: tool call stream finished:", tool.Index, tool.Name, tool.Arguments)
		}

		// It's best to use chunks after handling JustFinished events.
		// Here we print the delta of the content, if it exists.
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onChunk(chunk)
		}
	}

	if err := stream.Err(); err != nil {
		return openai.ChatCompletion{}, err
	}

	return acc.ChatCompletion, nil
}
