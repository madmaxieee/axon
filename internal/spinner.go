package internal

import (
	"fmt"
	"os"
	"time"
)

var frames = []string{"⠇", "⠋", "⠙", "⠸", "⢰", "⣠", "⣄", "⡆"}

type Spinner struct {
	frames   []string
	pos      int
	stopChan chan struct{}
	doneChan chan struct{}
}

func NewSpinner() *Spinner {
	return &Spinner{
		frames: frames,
	}
}

func (s *Spinner) Next() string {
	if len(s.frames) == 0 {
		return ""
	}
	frame := s.frames[s.pos]
	s.pos = (s.pos + 1) % len(s.frames)
	return frame
}

func (s *Spinner) Start(message string) {
	if s.stopChan != nil {
		return
	}
	s.stopChan = make(chan struct{})
	s.doneChan = make(chan struct{})
	go func() {
		defer close(s.doneChan)
		for {
			fmt.Fprintf(os.Stderr, "\r\033[K%s %s", s.Next(), message)
			select {
			case <-s.stopChan:
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
	}()
}

func (s *Spinner) Stop() {
	if s.stopChan != nil {
		close(s.stopChan)
		<-s.doneChan
		s.stopChan = nil
		s.doneChan = nil
		fmt.Fprintf(os.Stderr, "\r\033[K")
	}
}
