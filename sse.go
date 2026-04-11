package aero

import (
	"io"
	"net/http"
)

var (
	sseData        = []byte("data: ")
	sseEvent       = []byte("event: ")
	sseID          = []byte("id: ")
	sseNewlineData = []byte("\ndata: ")
	sseEnd         = []byte("\n\n")
)

type SSEWriter struct {
	c       *Ctx
	flusher http.Flusher
}

func (c *Ctx) SSE() (*SSEWriter, bool) {
	flusher, ok := c.w.(http.Flusher)
	if !ok {
		return nil, false
	}

	h := c.w.Header()
	h.Set(HeaderContentType, "text/event-stream")
	h.Set(HeaderCacheControl, "no-cache")
	h.Set(HeaderConnection, "keep-alive")
	c.w.WriteHeader(http.StatusOK)
	c.written = true

	return &SSEWriter{c: c, flusher: flusher}, true
}

func (s *SSEWriter) Send(data string) error {
	w := s.c.w

	if _, err := w.Write(sseData); err != nil {
		return err
	}
	if _, err := io.WriteString(w, data); err != nil {
		return err
	}
	if _, err := w.Write(sseEnd); err != nil {
		return err
	}

	s.flusher.Flush()
	return nil
}

func (s *SSEWriter) SendEvent(event, data string) error {
	w := s.c.w

	if _, err := w.Write(sseEvent); err != nil {
		return err
	}
	if _, err := io.WriteString(w, event); err != nil {
		return err
	}
	if _, err := w.Write(sseNewlineData); err != nil {
		return err
	}
	if _, err := io.WriteString(w, data); err != nil {
		return err
	}
	if _, err := w.Write(sseEnd); err != nil {
		return err
	}

	s.flusher.Flush()
	return nil
}

func (s *SSEWriter) SendID(id, data string) error {
	w := s.c.w

	if _, err := w.Write(sseID); err != nil {
		return err
	}
	if _, err := io.WriteString(w, id); err != nil {
		return err
	}
	if _, err := w.Write(sseNewlineData); err != nil {
		return err
	}
	if _, err := io.WriteString(w, data); err != nil {
		return err
	}
	if _, err := w.Write(sseEnd); err != nil {
		return err
	}

	s.flusher.Flush()
	return nil
}
