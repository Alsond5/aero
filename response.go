package aero

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrFileNotFound    = errors.New("aero: file not found")
	ErrIsDirectory     = errors.New("aero: path is a directory")
	ErrPathRequired    = errors.New("aero: path is required")
	ErrPathNotAbsolute = errors.New("aero: path must be absolute or root must be specified")
)

var (
	jsonpTypeof   = []byte("/**/ typeof ")
	jsonpFunction = []byte(" === 'function' && ")
	jsonpOpen     = []byte("(")
	jsonpClose    = []byte(");")
)

// Res provides methods for building and sending the HTTP response.
// It is embedded in [Ctx] and should not be instantiated directly.
type Res struct {
	c *Ctx
}

// Status sets the HTTP status code for the response. Returns *Res to allow
// method chaining before a final send call.
//
//	c.Res.Status(http.StatusCreated).JSON(body)
func (res *Res) Status(code int) *Res {
	if res.c.written || code < 100 || code > 999 {
		return res
	}

	res.c.status = code
	return res
}

// ResponseStatus returns the HTTP status code that will be (or has been)
// written for this response. Defaults to 200 if [Res.Status] was not called.
func (res *Res) ResponseStatus() int {
	return res.c.status
}

// Send writes body to the response. The Content-Type is inferred from the
// value type: []byte and string are sent as-is, all other types are JSON-encoded.
// For 204, 304 responses the body is automatically discarded and
// content-related headers are stripped. HEAD requests are handled correctly.
//
//	c.Res.Send("hello")
//	c.Res.Send(myStruct)
func (res *Res) Send(body any) error {
	if res.c.written {
		return nil
	}

	switch v := body.(type) {
	case string:
		return res.c.SendString(v)
	case []byte:
		return res.c.SendBytes(v)
	case nil:
		return res.c.SendBytes(nil)
	default:
		return res.c.JSON(v)
	}
}

// SendString writes a plain string body with Content-Type text/plain.
func (res *Res) SendString(body string) error {
	if res.c.written {
		return nil
	}

	if res.c.w.Header().Get(HeaderContentType) == "" {
		res.c.w.Header().Set(HeaderContentType, "text/html; charset=utf-8")
	}

	res.c.w.WriteHeader(res.c.status)
	res.c.written = true

	if res.c.r.Method != http.MethodHead {
		n, err := io.WriteString(res.c.w, body)
		res.c.size = int64(n)

		return err
	}

	return nil
}

// SendBytes writes a raw byte slice body.
// Content-Type defaults to application/octet-stream unless set beforehand.
func (res *Res) SendBytes(body []byte) error {
	if res.c.written {
		return nil
	}

	if res.c.w.Header().Get(HeaderContentType) == "" {
		res.c.w.Header().Set(HeaderContentType, "application/octet-stream")
	}

	return res.c.send(body)
}

// JSON serializes body to JSON and writes it with Content-Type application/json.
// Returns an error if serialization fails.
//
//	c.Res.JSON(map[string]any{"ok": true})
func (res *Res) JSON(body any) error {
	if res.c.written {
		return nil
	}

	if res.c.w.Header().Get(HeaderContentType) == "" {
		res.c.w.Header().Set(HeaderContentType, "application/json; charset=utf-8")
	}

	res.c.w.WriteHeader(res.c.status)
	res.c.written = true

	if res.c.r.Method == http.MethodHead {
		return nil
	}

	return json.NewEncoder(res.c.w).Encode(body)
}

// JSONP sends a JSONP response by wrapping the JSON-serialized body in the
// provided JavaScript callback. The callback name is validated against a
// safe identifier allowlist to prevent XSS.
//
//	c.Res.JSONP(data, "cb")
//	// → cb({"key":"value"});
func (res *Res) JSONP(body any, callback string) error {
	if res.c.written {
		return nil
	}

	if callback == "" {
		return res.JSON(body)
	}

	if !isSafeCallback(callback) {
		res.c.w.WriteHeader(400)
		res.c.written = true
		return nil
	}

	res.c.w.Header().Set(HeaderXContentTypeOptions, "nosniff")
	res.c.w.Header().Set(HeaderContentType, "text/javascript; charset=utf-8")
	res.c.w.WriteHeader(res.c.status)
	res.c.written = true

	if res.c.r.Method == http.MethodHead {
		return nil
	}

	w := res.c.w
	if _, err := w.Write(jsonpTypeof); err != nil {
		return err
	}
	if _, err := io.WriteString(w, callback); err != nil {
		return err
	}
	if _, err := w.Write(jsonpFunction); err != nil {
		return err
	}
	if _, err := io.WriteString(w, callback); err != nil {
		return err
	}
	if _, err := w.Write(jsonpOpen); err != nil {
		return err
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		return err
	}
	_, err := w.Write(jsonpClose)

	return err
}

// SendStatus writes the given HTTP status code and immediately flushes
// the response with no body. Useful for sending header-only responses
// such as 204 No Content or 401 Unauthorized.
//
//	return c.Res.SendStatus(http.StatusNoContent)
func (res *Res) SendStatus(code int) error {
	if res.c.written {
		return nil
	}

	res.c.Status(code)
	res.c.w.WriteHeader(res.c.status)

	return nil
}

// SetHeader sets a response header, replacing any existing value.
// Returns *Res for chaining.
func (res *Res) SetHeader(key, value string) *Res {
	res.c.w.Header().Set(key, value)
	return res
}

// AddHeader appends a value to a response header without replacing
// existing values. Returns *Res for chaining.
func (res *Res) AddHeader(key, value string) *Res {
	res.c.w.Header().Add(key, value)
	return res
}

// GetHeader returns the current value of the named response header.
// Returns an empty string if the header has not been set.
func (res *Res) GetHeader(key string) string {
	return res.c.w.Header().Get(key)
}

// DeleteHeader removes the named header from the response.
// Returns *Res for chaining.
func (res *Res) DeleteHeader(key string) *Res {
	res.c.w.Header().Del(key)
	return res
}

// CookieOptions configures the attributes of a cookie set via [Res.SetCookie]
// or cleared via [Res.ClearCookie].
type CookieOptions struct {
	// MaxAge sets the cookie's Max-Age attribute in seconds.
	// A negative value instructs the client to delete the cookie immediately.
	MaxAge int

	// Path restricts the cookie to the given URL path prefix.
	// Defaults to "/" if empty.
	Path string

	// Domain sets the cookie's Domain attribute, controlling which hosts
	// receive the cookie. Omit to default to the current host only.
	Domain string

	// Secure marks the cookie to be sent only over HTTPS connections.
	Secure bool

	// HttpOnly prevents the cookie from being accessed via JavaScript,
	// mitigating XSS exposure.
	HttpOnly bool

	// SameSite controls cross-site request behaviour for the cookie.
	// Accepts [http.SameSiteStrictMode], [http.SameSiteLaxMode], or
	// [http.SameSiteNoneMode].
	SameSite http.SameSite
}

// SetCookie adds a Set-Cookie header to the response with the given name,
// value, and optional [CookieOptions] (path, domain, expiry, flags, etc.).
// Returns *Res for chaining.
func (res *Res) SetCookie(name, value string, opts ...CookieOptions) *Res {
	cookie := &http.Cookie{
		Name:  name,
		Value: value,
		Path:  "/",
	}

	if len(opts) > 0 {
		o := opts[0]
		if o.Path != "" {
			cookie.Path = o.Path
		}
		if o.MaxAge != 0 {
			cookie.MaxAge = o.MaxAge
		}

		cookie.HttpOnly = o.HttpOnly
		cookie.Domain = o.Domain
		cookie.Secure = o.Secure
		cookie.SameSite = o.SameSite
	}

	http.SetCookie(res.c.w, cookie)
	return res
}

// ClearCookie expires the named cookie on the client by setting its Max-Age
// to -1. Optional [CookieOptions] can be provided to match the original cookie's
// path and domain. Returns *Res for chaining.
func (res *Res) ClearCookie(name string, opts ...CookieOptions) *Res {
	o := CookieOptions{Path: "/"}
	if len(opts) > 0 {
		o = opts[0]
		o.Path = "/"
	}

	o.MaxAge = -1
	return res.SetCookie(name, "", o)
}

// Links sets the Link response header from the provided map of relation to URL.
// Multiple relations are comma-separated per RFC 8288.
//
//	c.Res.Links(map[string]string{
//		"next": "/page/3",
//		"prev": "/page/1",
//	})
func (res *Res) Links(links map[string]string) *Res {
	existing := res.c.w.Header().Get(HeaderLink)

	var b strings.Builder
	if existing != "" {
		b.WriteString(existing)
		b.WriteString(", ")
	}

	first := true
	for rel, url := range links {
		if !first {
			b.WriteString(", ")
		}
		b.WriteByte('<')
		b.WriteString(url)
		b.WriteString(">; rel=\"")
		b.WriteString(rel)
		b.WriteByte('"')
		first = false
	}

	res.c.w.Header().Set(HeaderLink, b.String())
	return res
}

// Type sets the Content-Type response header. Short aliases such as "json",
// "html", "text", and "xml" are resolved to their full MIME types.
// Returns *Res for chaining.
//
//	c.Res.Type("html").SendString("<h1>Hello</h1>")
func (res *Res) Type(t string) *Res {
	var ct string
	if strings.ContainsRune(t, '/') {
		ct = t
	} else {
		ext := t
		if ext[0] != '.' {
			ext = "." + ext
		}

		ct = mime.TypeByExtension(ext)
		if ct == "" {
			ct = "application/octet-stream"
		}
	}

	res.c.w.Header().Set(HeaderContentType, ct)
	return res
}

// Attachment sets the Content-Disposition header to "attachment", prompting
// the browser to download the response rather than display it. An optional
// filename is included in the header when provided.
//
//	c.Res.Attachment("report.pdf")
func (res *Res) Attachment(filename ...string) *Res {
	if len(filename) > 0 && filename[0] != "" {
		f := filename[0]

		res.Type(filepath.Ext(f))
		res.c.w.Header().Set(
			HeaderContentDisposition,
			`attachment; filename="`+f+`"`,
		)
	} else {
		res.c.w.Header().Set(HeaderContentDisposition, "attachment")
	}

	return res
}

// Format performs content negotiation using the request's Accept header.
// handlers is a map of MIME type to handler function; the best matching
// handler is called. Returns an error if no match is found.
//
//	c.Res.Format(map[string]func(...){
//		"application/json": func(...) { c.Res.JSON(data) },
//		"text/html":        func(...) { c.Res.SendString("<p>data</p>") },
//	})
func (res *Res) Format(handlers map[string]func() error) error {
	accept := res.c.r.Header.Get("Accept")
	if accept == "" {
		for ct, h := range handlers {
			if ct == "default" {
				continue
			}

			res.c.w.Header().Set(HeaderContentType, ct)
			return h()
		}
	}

	for ct, h := range handlers {
		if ct == "default" {
			continue
		}
		if strings.Contains(accept, ct) || strings.Contains(accept, "*/*") {
			res.c.w.Header().Set(HeaderContentType, ct)
			return h()
		}
	}

	if h, ok := handlers["default"]; ok {
		return h()
	}

	res.c.w.WriteHeader(406)
	res.c.written = true

	return nil
}

// Location sets the Location response header to the given URL.
// Returns *Res for chaining.
func (res *Res) Location(url string) *Res {
	if url == "back" {
		url = res.c.r.Header.Get(HeaderReferer)
		if url == "" {
			url = "/"
		}
	}

	res.c.w.Header().Set(HeaderLocation, url)
	return res
}

// Redirect sends an HTTP redirect to the given URL. The status code defaults
// to 302 (Found) if not provided; any 3xx code is accepted.
//
//	c.Res.Redirect("/login")
//	c.Res.Redirect("/new-path", http.StatusMovedPermanently)
func (res *Res) Redirect(url string, code ...int) error {
	status := http.StatusFound
	if len(code) > 0 {
		status = code[0]
	}

	res.Location(url)

	http.Redirect(res.c.w, res.c.r, res.c.w.Header().Get(HeaderLocation), status)
	res.c.written = true
	return nil
}

// Vary appends the given field to the Vary response header, indicating that
// the response may differ based on that request header.
//
//	c.Res.Vary("Accept-Encoding")
func (res *Res) Vary(field string) *Res {
	vary := res.c.w.Header().Get(HeaderVary)

	if vary == "" {
		res.c.w.Header().Set(HeaderVary, field)
		return res
	}

	for v := range strings.SplitSeq(vary, ",") {
		if strings.EqualFold(strings.TrimSpace(v), field) {
			return res
		}
	}

	res.c.w.Header().Set(HeaderVary, vary+", "+field)
	return res
}

// SendFileOptions configures the behaviour of [Res.SendFile] and [Res.Download].
type SendFileOptions struct {
	// MaxAge sets the Cache-Control max-age directive in seconds for the
	// served file. A value of 0 disables caching headers.
	MaxAge int

	// Headers is a map of additional response headers to set when serving
	// the file (e.g. custom Cache-Control directives, CORS headers).
	Headers map[string]string

	// Root restricts file serving to the given directory. Any path that
	// resolves outside Root is rejected to prevent directory traversal attacks.
	// If empty, no restriction is applied.
	Root string
}

// SendFile serves a file from the local filesystem at the given path.
// It sets Content-Type based on the file extension, supports Range requests,
// and accepts optional [SendFileOptions] (e.g. root directory restriction).
// Path traversal outside the configured root is rejected.
func (res *Res) SendFile(path string, opts ...SendFileOptions) error {
	if path == "" {
		return ErrPathRequired
	}

	opt := SendFileOptions{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	fullPath := path
	if !filepath.IsAbs(path) {
		if opt.Root == "" {
			return ErrPathNotAbsolute
		}
		fullPath = filepath.Join(opt.Root, path)
	}

	if opt.Root != "" {
		if !isSubPath(opt.Root, fullPath) {
			return ErrFileNotFound
		}
	}

	f, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotFound
		}
		return err
	}
	defer f.Close() //nolint:errcheck

	stat, err := f.Stat()
	if err != nil {
		return err
	}

	if stat.IsDir() {
		return ErrIsDirectory
	}

	ext := filepath.Ext(fullPath)
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		ct = "application/octet-stream"
	}

	res.c.w.Header().Set(HeaderContentType, ct)

	if opt.MaxAge > 0 {
		res.c.w.Header().Set(HeaderCacheControl,
			"public, max-age="+itoa(opt.MaxAge),
		)
	}

	for k, v := range opt.Headers {
		res.c.w.Header().Set(k, v)
	}

	http.ServeContent(res.c.w, res.c.r, stat.Name(), stat.ModTime(), f)
	res.c.written = true

	return nil
}

// SendFileFS serves a file from the provided [http.FileSystem] at the given path.
// Useful for serving embedded files (e.g. via go:embed).
func (res *Res) SendFileFS(fsys http.FileSystem, path string) error {
	f, err := fsys.Open(path)
	if err != nil {
		return ErrFileNotFound
	}
	defer f.Close() //nolint:errcheck

	stat, err := f.Stat()
	if err != nil {
		return err
	}

	if stat.IsDir() {
		return ErrIsDirectory
	}

	ext := filepath.Ext(path)
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		ct = "application/octet-stream"
	}

	res.c.w.Header().Set(HeaderContentType, ct)

	rs, ok := f.(io.ReadSeeker)
	if !ok {
		return errors.New("aero: file does not support seeking")
	}

	http.ServeContent(res.c.w, res.c.r, stat.Name(), stat.ModTime(), rs)
	res.c.written = true

	return nil
}

// Download serves a file as a downloadable attachment. It behaves like
// [Res.SendFile] but also sets Content-Disposition to "attachment".
// An optional filename overrides the name shown in the browser's save dialog.
func (res *Res) Download(path string, filename ...string) error {
	name := filepath.Base(path)
	if len(filename) > 0 && filename[0] != "" {
		name = filename[0]
	}

	res.c.w.Header().Set(HeaderContentDisposition, `attachment; filename="`+name+`"`)

	return res.SendFile(path)
}

// DownloadFS serves a file from the provided [http.FileSystem] as a
// downloadable attachment. See [Res.Download] and [Res.SendFileFS].
func (res *Res) DownloadFS(fsys http.FileSystem, path string, filename ...string) error {
	name := filepath.Base(path)
	if len(filename) > 0 && filename[0] != "" {
		name = filename[0]
	}

	res.c.w.Header().Set(HeaderContentDisposition, `attachment; filename="`+name+`"`)

	return res.SendFileFS(fsys, path)
}

func (res *Res) send(body []byte) error {
	switch res.c.status {
	case http.StatusNoContent, http.StatusNotModified:
		res.c.w.Header().Del(HeaderContentType)
		res.c.w.Header().Del(HeaderContentLength)
		res.c.w.Header().Del(HeaderTransferEncoding)

		body = nil

	case http.StatusResetContent:
		res.c.w.Header().Set(HeaderContentLength, "0")
		res.c.w.Header().Del(HeaderTransferEncoding)

		body = nil
	}

	res.c.w.WriteHeader(res.c.status)
	res.c.written = true

	if res.c.Method() == http.MethodHead || body == nil {
		res.c.size = 0
		return nil
	}

	n, err := res.c.w.Write(body)
	res.c.size = int64(n)

	return err
}

func isSubPath(root, fullPath string) bool {
	root = filepath.Clean(root)
	fullPath = filepath.Clean(fullPath)

	rel, err := filepath.Rel(root, fullPath)
	if err != nil {
		return false
	}

	return rel != ".." && len(rel) >= 2 && rel[:2] != ".."
}

func itoa(n int) string {
	var buf [20]byte

	pos := len(buf)
	for n >= 10 {
		pos--
		buf[pos] = byte(n%10) + '0'
		n /= 10
	}
	pos--

	buf[pos] = byte(n) + '0'
	return string(buf[pos:])
}

func isSafeCallback(cb string) bool {
	for _, c := range cb {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') &&
			c != '_' && c != '$' && c != '.' && c != '[' && c != ']' {

			return false
		}
	}

	return true
}
