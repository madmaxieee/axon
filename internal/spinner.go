package internal

import (
	"fmt"
	"os"
	"time"
)

var frames = []string{"⠇", "⠋", "⠙", "⠸", "⢰", "⣠", "⣄", "⡆"}
var gradient = []string{
	"\033[38;2;255;0;0m",
	"\033[38;2;255;0;85m",
	"\033[38;2;255;0;170m",
	"\033[38;2;255;0;255m",
	"\033[38;2;212;0;255m",
	"\033[38;2;170;0;255m",
	"\033[38;2;128;0;255m",
	"\033[38;2;170;0;255m",
	"\033[38;2;212;0;255m",
	"\033[38;2;255;0;255m",
	"\033[38;2;255;0;170m",
	"\033[38;2;255;0;85m",
}

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
	// \033[?25l hides the terminal cursor
	fmt.Fprintf(os.Stderr, "\033[?25l")
	s.stopChan = make(chan struct{})
	s.doneChan = make(chan struct{})
	go func() {
		defer close(s.doneChan)
		colorIndex := 0
		for {
			color := gradient[colorIndex]
			colorIndex = (colorIndex + 1) % len(gradient)
			fmt.Fprintf(os.Stderr, "\r\033[K%s%s\033[0m \033[2m%s\033[0m", color, s.Next(), message)
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
		// \033[K clears the line
		// \033[?25h restores the cursor
		fmt.Fprintf(os.Stderr, "\r\033[K\033[?25h")
	}
}
