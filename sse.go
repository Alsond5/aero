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

// SSEWriter writes Server-Sent Events to the client over an open HTTP
// connection. Obtain an instance via [Ctx.SSE].
type SSEWriter struct {
	c       *Ctx
	flusher http.Flusher
}

// SSE upgrades the current request to a Server-Sent Events stream and returns
// an [SSEWriter] for pushing events to the client. The second return value
// reports whether the upgrade succeeded; it is false if the underlying
// ResponseWriter does not support flushing (i.e. does not implement
// http.Flusher).
//
// The response Content-Type is automatically set to "text/event-stream" and
// caching is disabled. The connection is kept open until the handler returns.
//
//	app.GET("/events", func(c *aero.Ctx) error {
//		sse, ok := c.SSE()
//		if !ok {
//			return c.Res.SendStatus(http.StatusInternalServerError)
//		}
//		sse.Send("hello")
//		return nil
//	})
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

// Send writes a data-only SSE event to the client and flushes immediately.
//
//	sse.Send("ping")
//	// event stream: "data: ping\n\n"
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

// SendEvent writes a named SSE event with data to the client and flushes.
//
//	sse.SendEvent("update", `{"count":42}`)
//	// event stream: "event: update\ndata: {\"count\":42}\n\n"
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

// SendID writes a data event with an explicit event ID to the client and
// flushes. The ID allows clients to resume the stream from the last seen
// event via the Last-Event-ID header on reconnect.
//
//	sse.SendID("msg-42", "hello again")
//	// event stream: "id: msg-42\ndata: hello again\n\n"
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
