package aero_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/Alsond5/aero"
)

var payload []byte
var healthResponse = []byte("ok")

func init() {
	payload, _ = json.Marshal(struct {
		Name string `json:"name"`
	}{Name: "Alsond5"})
}

func setupAero() *aero.App {
	app := aero.New()

	app.Use(func(c *aero.Ctx) error {
		c.Header("Content-Type")
		return c.Next()
	})

	app.GET("/api/users/:id", func(c *aero.Ctx) error {
		return c.JSON("hello")
	})

	app.GET("/health", func(c *aero.Ctx) error {
		return c.SendBytes(healthResponse)
	})

	app.POST("/api/profile", func(c *aero.Ctx) error {
		var s struct {
			Name string `json:"name"`
		}
		c.BodyJSON(&s)
		return c.SendString(s.Name)
	})

	return app
}

func BenchmarkAero(b *testing.B) {
	app := setupAero()

	req := httptest.NewRequest(http.MethodPost, "/api/profile", nil)
	req.Header.Set("Content-Type", "application/json")
	reader := bytes.NewReader(payload)

	requests := []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/users/42", nil),
		httptest.NewRequest(http.MethodGet, "/health", nil),
		req,
	}

	w := httptest.NewRecorder()

	runtime.GC()

	b.ResetTimer()
	b.ReportAllocs()

	i := 0
	for b.Loop() {
		if i%3 == 2 {
			reader.Seek(0, io.SeekStart)
			requests[2].Body = io.NopCloser(reader)
		}

		w.Body.Reset()
		app.ServeHTTP(w, requests[i%3])
		i++
	}
}

func BenchmarkAeroRouting(b *testing.B) {
	app := setupAero()

	requests := []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/users/42", nil),
		httptest.NewRequest(http.MethodGet, "/health", nil),
	}

	w := httptest.NewRecorder()
	runtime.GC()

	b.ResetTimer()
	b.ReportAllocs()

	i := 0
	for b.Loop() {
		w.Body.Reset()
		app.ServeHTTP(w, requests[i%2])
		i++
	}
}

func BenchmarkAeroParallel(b *testing.B) {
	app := setupAero()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		requests := []*http.Request{
			httptest.NewRequest(http.MethodGet, "/api/users/42", nil),
			httptest.NewRequest(http.MethodGet, "/health", nil),
		}
		w := httptest.NewRecorder()
		i := 0

		for pb.Next() {
			w.Body.Reset()
			app.ServeHTTP(w, requests[i%2])
			i++
		}
	})
}
