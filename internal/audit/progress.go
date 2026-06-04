package audit

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Spinner struct {
	out     io.Writer
	frames  []string
	enabled bool
	period  time.Duration

	mu      sync.Mutex
	done    int
	total   int
	running bool
	stop    chan struct{}
	stopped chan struct{}
}

func NewSpinner(out io.Writer, enabled bool) *Spinner {
	return &Spinner{
		out:     out,
		frames:  []string{"-", "\\", "|", "/"},
		enabled: enabled,
		period:  120 * time.Millisecond,
	}
}

func NewTerminalSpinner(out *os.File) *Spinner {
	return NewSpinner(out, isTerminal(out))
}

func (s *Spinner) Start(total int) {
	if !s.enabled || total == 0 {
		return
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.total = total
	s.stop = make(chan struct{})
	s.stopped = make(chan struct{})
	s.running = true
	s.mu.Unlock()

	go s.loop()
}

func (s *Spinner) Advance(done int) {
	if !s.enabled {
		return
	}

	s.mu.Lock()
	s.done = done
	s.mu.Unlock()
}

func (s *Spinner) Stop() {
	if !s.enabled {
		return
	}

	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	stop := s.stop
	stopped := s.stopped
	s.running = false
	s.mu.Unlock()

	close(stop)
	<-stopped
	clearLine(s.out)
}

func (s *Spinner) ProgressFunc() ProgressFunc {
	return func(done, total int) {
		if done == 0 {
			s.Start(total)
			return
		}
		s.Advance(done)
	}
}

func (s *Spinner) loop() {
	defer close(s.stopped)

	frameIndex := 0
	ticker := time.NewTicker(s.period)
	defer ticker.Stop()

	s.render(frameIndex)
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			frameIndex = (frameIndex + 1) % len(s.frames)
			s.render(frameIndex)
		}
	}
}

func (s *Spinner) render(frameIndex int) {
	s.mu.Lock()
	done := s.done
	total := s.total
	frame := s.frames[frameIndex]
	s.mu.Unlock()

	clearLine(s.out)
	fmt.Fprintf(s.out, "Checking policies %d/%d %s", done, total, frame)
}

func clearLine(out io.Writer) {
	fmt.Fprint(out, "\r\033[2K")
}

func isTerminal(file *os.File) bool {
	if file == nil {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func SpinnerPreview(done, total int, frame string) string {
	return strings.TrimSpace(fmt.Sprintf("Checking policies %d/%d %s", done, total, frame))
}
