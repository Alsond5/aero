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

const (
	HeaderTransferEncoding    = "Transfer-Encoding"
	HeaderContentType         = "Content-Type"
	HeaderContentLength       = "Content-Length"
	HeaderLink                = "Link"
	HeaderContentDisposition  = "Content-Disposition"
	HeaderXContentTypeOptions = "X-Content-Type-Options"
	HeaderReferer             = "Referer"
	HeaderLocation            = "Location"
	HeaderVary                = "Vary"
	HeaderCacheControl        = "Cache-Control"
)

var (
	ErrFileNotFound    = errors.New("aero: file not found")
	ErrIsDirectory     = errors.New("aero: path is a directory")
	ErrPathRequired    = errors.New("aero: path is required")
	ErrPathNotAbsolute = errors.New("aero: path must be absolute or root must be specified")
)

type Res struct {
	c *Ctx
}

func (res *Res) Status(code int) *Res {
	if res.c.written || code < 100 || code > 999 {
		return res
	}

	res.c.status = code
	return res
}

func (res *Res) ResponseStatus() int {
	return res.c.status
}

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

func (res *Res) SendBytes(body []byte) error {
	if res.c.written {
		return nil
	}

	if res.c.w.Header().Get(HeaderContentType) == "" {
		res.c.w.Header().Set(HeaderContentType, "application/octet-stream")
	}

	return res.c.send(body)
}

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

func (res *Res) SendStatus(code int) error {
	if res.c.written {
		return nil
	}

	res.c.Status(code)
	res.c.w.WriteHeader(res.c.status)

	return nil
}

func (res *Res) SetHeader(key, value string) *Res {
	res.c.w.Header().Set(key, value)
	return res
}

func (res *Res) AddHeader(key, value string) *Res {
	res.c.w.Header().Add(key, value)
	return res
}

func (res *Res) GetHeader(key string) string {
	return res.c.w.Header().Get(key)
}

func (res *Res) DeleteHeader(key string) *Res {
	res.c.w.Header().Del(key)
	return res
}

type CookieOptions struct {
	MaxAge   int
	Path     string
	Domain   string
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

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

func (res *Res) ClearCookie(name string, opts ...CookieOptions) *Res {
	o := CookieOptions{Path: "/"}
	if len(opts) > 0 {
		o = opts[0]
		o.Path = "/"
	}

	o.MaxAge = -1
	return res.SetCookie(name, "", o)
}

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
	io.WriteString(w, "/**/ typeof ")
	io.WriteString(w, callback)
	io.WriteString(w, " === 'function' && ")
	io.WriteString(w, callback)
	io.WriteString(w, "(")

	json.NewEncoder(w).Encode(body)

	_, err := io.WriteString(w, ");")
	return err
}

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

type SendFileOptions struct {
	MaxAge  int
	Headers map[string]string
	Root    string
}

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
	defer f.Close()

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

func (res *Res) SendFileFS(fsys http.FileSystem, path string) error {
	f, err := fsys.Open(path)
	if err != nil {
		return ErrFileNotFound
	}
	defer f.Close()

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

func (res *Res) Download(path string, filename ...string) error {
	name := filepath.Base(path)
	if len(filename) > 0 && filename[0] != "" {
		name = filename[0]
	}

	res.c.w.Header().Set(HeaderContentDisposition, `attachment; filename="`+name+`"`)

	return res.SendFile(path)
}

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
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '_' || c == '$' || c == '.' ||
			c == '[' || c == ']') {
			return false
		}
	}
	return true
}
