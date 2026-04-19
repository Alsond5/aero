package aero

import (
	"compress/flate"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const (
	RangeUnsatisfiable = -1
	RangeMalformed     = -2
)

var (
	ErrBodyAlreadyRead      = errors.New("aero: request body already read")
	ErrBodyTooLarge         = errors.New("aero: request body too large")
	ErrBodyEmpty            = errors.New("aero: request body is empty")
	ErrFormAlreadyParsed    = errors.New("aero: form already parsed")
	ErrNotMultipart         = errors.New("aero: content type is not multipart/form-data")
	ErrRangeNotPresent      = errors.New("aero: Range header not present")
	ErrRangeUnsatisfiable   = errors.New("aero: range not satisfiable")
	ErrRangeMalformed       = errors.New("aero: range header malformed")
	ErrUnsupportedMediaType = errors.New("aero: unsupported content encoding")
)

type Req struct {
	c *Ctx
}

func (req *Req) Header(name string) string {
	return req.c.r.Header.Get(name)
}

func (req *Req) Get(name string) string {
	return req.Header(name)
}

func (req *Req) Headers() http.Header {
	return req.c.r.Header
}

func (req *Req) Accepts(types ...string) string {
	header := req.c.r.Header.Get("Accepts")
	if header == "" {
		if len(types) > 0 {
			return types[0]
		}

		return ""
	}

	accepted := parseAcceptHeader(header)

	for _, a := range accepted {
		for _, t := range types {
			mime := resolveType(t)
			if matchMIME(a.value, mime) {
				return t
			}
		}
	}

	return ""
}

func (req *Req) AcceptsEncodings(encodings ...string) string {
	header := req.c.r.Header.Get("Accept-Encoding")
	if header == "" {
		return ""
	}

	accepted := parseAcceptHeader(header)

	for _, a := range accepted {
		for _, e := range encodings {
			if strings.EqualFold(a.value, e) || a.value == "*" {
				return e
			}
		}
	}

	return ""
}

func (req *Req) AcceptsCharsets(charsets ...string) string {
	header := req.c.r.Header.Get("Accept-Charset")
	if header == "" {
		for _, c := range charsets {
			if strings.EqualFold(c, "utf-8") {
				return c
			}
		}

		return ""
	}

	accepted := parseAcceptHeader(header)

	for _, a := range accepted {
		for _, c := range charsets {
			if strings.EqualFold(a.value, c) || a.value == "*" {
				return c
			}
		}
	}

	return ""
}

func (req *Req) AcceptsLanguages(langs ...string) string {
	header := req.c.r.Header.Get("Accept-Language")
	if header == "" {
		return ""
	}

	accepted := parseAcceptHeader(header)

	for _, a := range accepted {
		for _, l := range langs {
			if strings.EqualFold(a.value, l) || a.value == "*" {
				return l
			}
		}
	}

	return ""
}

func (r *Req) Body() ([]byte, error) {
	dst := make([]byte, 0, 512)
	err := r.AppendBody(&dst)

	return dst, err
}

func (r *Req) AppendBody(dst *[]byte) error {
	if r.c.r.Body == nil {
		return ErrBodyAlreadyRead
	}

	defer func() {
		r.c.r.Body.Close() //nolint:errcheck
		r.c.r.Body = nil
	}()

	if r.c.app.config.MaxBodySize > 0 {
		r.c.r.Body = http.MaxBytesReader(r.c.w, r.c.r.Body, r.c.app.config.MaxBodySize)
	}

	reader := io.Reader(r.c.r.Body)

	encoding := r.c.r.Header.Get("Content-Encoding")
	switch encoding {
	case "", "identity":
	case "gzip":
		gr, err := gzip.NewReader(reader)
		if err != nil {
			return err
		}
		defer gr.Close() //nolint:errcheck
		reader = gr
	case "deflate":
		reader = flate.NewReader(reader)
	default:
		return ErrUnsupportedMediaType
	}

	var err error
	*dst, err = appendReadAll(*dst, reader)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return ErrBodyTooLarge
		}

		return err
	}

	return nil
}

func (r *Req) BodyReader() (io.ReadCloser, error) {
	if r.c.r.Body == nil {
		return nil, ErrBodyAlreadyRead
	}
	rc := r.c.r.Body
	r.c.r.Body = nil

	return rc, nil
}

func (req *Req) Param(key string) string {
	for i := range req.c.paramsCount {
		if req.c.params[i].Key != key {
			continue
		}

		return req.c.params[i].Value
	}

	return ""
}

func (req *Req) Params() []Param {
	return req.c.params[:req.c.paramsCount]
}

func (req *Req) Range(size int64, combine ...bool) (*RangeResult, error) {
	header := req.c.r.Header.Get("Range")
	if header == "" {
		return nil, ErrRangeNotPresent
	}

	_combine := false
	if len(combine) > 0 {
		_combine = combine[0]
	}

	result, code := parseRange(header, size, _combine)
	switch code {
	case RangeUnsatisfiable:
		return nil, ErrRangeUnsatisfiable
	case RangeMalformed:
		return nil, ErrRangeMalformed
	}

	return result, nil
}

func (req *Req) Query(key string) string {
	return req.QueryAll().Get(key)
}

func (req *Req) QueryAll() url.Values {
	if !req.c.queryParsed {
		req.c.query = req.c.r.URL.Query()
		req.c.queryParsed = true
	}

	return req.c.query
}

func (req *Req) Protocol() string {
	if req.c.r.TLS != nil {
		return "https"
	}

	return "http"
}

func (req *Req) Secure() bool {
	return req.Protocol() == "https"
}

func (req *Req) IP() string {
	if req.c.app.config.TrustProxy {
		if xff := req.c.r.Header.Get("X-Forwarded-For"); xff != "" {
			if i := strings.IndexByte(xff, ','); i != -1 {
				return strings.TrimSpace(xff[:i])
			}

			return strings.TrimSpace(xff)
		}

		if xri := req.c.r.Header.Get("X-Real-IP"); xri != "" {
			return strings.TrimSpace(xri)
		}
	}

	ip, _, err := net.SplitHostPort(req.c.r.RemoteAddr)
	if err != nil {
		return req.c.r.RemoteAddr
	}

	return ip
}

func (req *Req) IPs() []string {
	xff := req.c.r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return nil
	}

	parts := strings.Split(xff, ",")
	ips := make([]string, 0, len(parts))
	for _, p := range parts {
		if ip := strings.TrimSpace(p); ip != "" {
			ips = append(ips, ip)
		}
	}

	return ips
}

func (req *Req) OriginalURL() string {
	return req.c.r.RequestURI
}

func (req *Req) BaseURL() string {
	return req.c.basePath
}

func (req *Req) Path() string {
	if req.c.path == "" {
		if req.c.basePath != "" {
			req.c.path = strings.TrimPrefix(req.c.r.URL.Path, req.c.basePath)
		} else {
			req.c.path = req.c.r.URL.Path
		}
	}

	return req.c.path
}

func (req *Req) Validate(i any) error {
	if req.c.app.validator == nil {
		return nil
	}

	return req.c.app.validator.Validate(i)
}

func (req *Req) Context() context.Context {
	return req.c.r.Context()
}

func (req *Req) Cookie(name string) (*http.Cookie, error) {
	return req.c.r.Cookie(name)
}

func (req *Req) Cookies() []*http.Cookie {
	return req.c.r.Cookies()
}

func (req *Req) Method() string {
	return req.c.r.Method
}

func (req *Req) Host() string {
	if req.c.app.config.TrustProxy {
		if val := req.c.r.Header.Get("X-Forwarded-Host"); val != "" {
			if before, _, ok := strings.Cut(val, ","); ok {
				return strings.TrimSpace(before)
			}

			return strings.TrimSpace(val)
		}
	}

	return req.c.r.Header.Get("Host")
}

func (req *Req) Hostname() string {
	host := req.Host()
	if host == "" {
		return ""
	}

	if host[0] == '[' {
		end := strings.IndexByte(host, ']')
		if end == -1 {
			return host
		}

		return host[1:end]
	}

	if i := strings.IndexByte(host, ':'); i != -1 {
		return host[:i]
	}

	return host
}

func (req *Req) Subdomains() []string {
	hostname := req.Hostname()
	if hostname == "" {
		return nil
	}

	if net.ParseIP(hostname) != nil {
		return nil
	}

	parts := strings.Split(hostname, ".")

	offset := req.c.app.config.SubdomainOffset
	if offset <= 0 {
		offset = 2
	}

	if offset >= len(parts) {
		return nil
	}

	sub := parts[:len(parts)-offset]
	for i, j := 0, len(sub)-1; i < j; i, j = i+1, j-1 {
		sub[i], sub[j] = sub[j], sub[i]
	}

	return sub
}

func (req *Req) Fresh() bool {
	method := req.Method()

	if method != http.MethodGet && method != http.MethodHead {
		return false
	}

	status := req.c.status

	if (status < 200 || status >= 300) && status != 304 {
		return false
	}

	etag := req.c.w.Header().Get("ETag")
	noneMatch := req.c.r.Header.Get("If-None-Match")

	if etag != "" && noneMatch != "" {
		return noneMatch == "*" || noneMatch == etag
	}

	lastModified := req.c.w.Header().Get("Last-Modified")
	modifiedSince := req.c.r.Header.Get("If-Modified-Since")

	if lastModified != "" && modifiedSince != "" {
		return lastModified == modifiedSince
	}

	return false
}

func (req *Req) Stale() bool {
	return !req.Fresh()
}

func (req *Req) XHR() bool {
	val := req.Header("X-Requested-With")
	return strings.ToLower(val) == "xmlhttprequest"
}

func (req *Req) FormValue(key string) string {
	if err := req.parseForm(); err != nil {
		return ""
	}

	return req.c.r.FormValue(key)
}

func (req *Req) FormValues() map[string][]string {
	if err := req.parseForm(); err != nil {
		return nil
	}

	return map[string][]string(req.c.r.Form)
}

func (req *Req) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	if err := req.parseMultipartForm(); err != nil {
		return nil, nil, err
	}

	return req.c.r.FormFile(key)
}

func (req *Req) FormFiles(key string) ([]*multipart.FileHeader, error) {
	if err := req.parseMultipartForm(); err != nil {
		return nil, err
	}

	if req.c.r.MultipartForm == nil {
		return nil, ErrNotMultipart
	}

	files, ok := req.c.r.MultipartForm.File[key]
	if !ok {
		return nil, nil
	}

	return files, nil
}

func (req *Req) MultipartReader() (*multipart.Reader, error) {
	if req.c.formParsed {
		return nil, ErrFormAlreadyParsed
	}

	if req.c.r.Body == nil {
		return nil, ErrBodyAlreadyRead
	}

	r, err := req.c.r.MultipartReader()
	if err != nil {
		if err == http.ErrNotMultipart {
			return nil, ErrNotMultipart
		}
		return nil, err
	}

	req.c.r.Body = nil
	req.c.formParsed = true

	return r, nil
}

func (req *Req) parseForm() error {
	if req.c.formParsed {
		return nil
	}

	if req.c.r.Body == nil {
		return ErrBodyAlreadyRead
	}

	if req.c.app.config.MaxBodySize > 0 {
		req.c.r.Body = http.MaxBytesReader(req.c.w, req.c.r.Body, req.c.app.config.MaxBodySize)
	}

	if err := req.c.r.ParseForm(); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return ErrBodyTooLarge
		}

		return err
	}

	req.c.formParsed = true
	return nil
}

func (req *Req) parseMultipartForm() error {
	if req.c.formParsed {
		return nil
	}

	if req.c.r.Body == nil {
		return ErrBodyAlreadyRead
	}

	if req.c.app.config.MaxBodySize > 0 {
		req.c.r.Body = http.MaxBytesReader(req.c.w, req.c.r.Body, req.c.app.config.MaxBodySize)
	}

	maxMemory := req.c.app.config.MaxMultipartMemory
	if maxMemory == 0 {
		maxMemory = 32 << 20
	}

	if err := req.c.r.ParseMultipartForm(maxMemory); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return ErrBodyTooLarge
		}
		if err == http.ErrNotMultipart {
			return ErrNotMultipart
		}

		return err
	}

	req.c.formParsed = true
	return nil
}

type acceptItem struct {
	value string
	q     float32
}

func parseAcceptHeader(header string) []acceptItem {
	parts := strings.Split(header, ",")
	items := make([]acceptItem, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		item := acceptItem{q: 1.0}

		semi := strings.IndexByte(part, ';')
		if semi == -1 {
			item.value = strings.ToLower(strings.TrimSpace(part))
		} else {
			item.value = strings.ToLower(strings.TrimSpace(part[:semi]))
			param := strings.TrimSpace(part[semi+1:])

			if strings.HasPrefix(param, "q=") {
				item.q = parseQ(param[2:])
			}
		}

		items = append(items, item)
	}

	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].q > items[j-1].q; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}

	return items
}

func parseQ(s string) float32 {
	if s == "1" || s == "1.0" {
		return 1.0
	}
	if len(s) == 0 {
		return 1.0
	}

	var q float32
	var frac float32 = 1.0
	afterDot := false

	for _, c := range s {
		if c == '.' {
			afterDot = true
			continue
		}
		if c < '0' || c > '9' {
			return 1.0
		}
		if afterDot {
			frac *= 0.1
			q += float32(c-'0') * frac
		} else {
			q = q*10 + float32(c-'0')
		}
	}

	return q
}

func resolveType(t string) string {
	if strings.ContainsRune(t, '/') {
		return strings.ToLower(t)
	}

	m := mime.TypeByExtension("." + t)
	if m == "" {
		return strings.ToLower(t)
	}

	semi := strings.IndexByte(m, ';')
	if semi != -1 {
		m = strings.TrimSpace(m[:semi])
	}

	return strings.ToLower(m)
}

func matchMIME(pattern, target string) bool {
	if pattern == "*/*" || pattern == "*" {
		return true
	}
	if pattern == target {
		return true
	}

	if strings.HasSuffix(pattern, "/*") {
		prefix := pattern[:len(pattern)-2]
		return strings.HasPrefix(target, prefix+"/")
	}

	return false
}

type Range struct {
	Start int64
	End   int64
}

type RangeResult struct {
	Type   string
	Ranges []Range
}

func parseRange(header string, size int64, combine bool) (*RangeResult, int) {
	eq := strings.IndexByte(header, '=')
	if eq == -1 {
		return nil, RangeMalformed
	}

	rangeType := strings.TrimSpace(header[:eq])
	if rangeType == "" {
		return nil, RangeMalformed
	}

	raw := header[eq+1:]
	parts := strings.Split(raw, ",")
	ranges := make([]Range, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, RangeMalformed
		}

		dash := strings.IndexByte(part, '-')
		if dash == -1 {
			return nil, RangeMalformed
		}

		startStr := strings.TrimSpace(part[:dash])
		endStr := strings.TrimSpace(part[dash+1:])

		var start, end int64

		switch {
		case startStr == "" && endStr == "":
			return nil, RangeMalformed

		case startStr == "":
			n, ok := parseInt(endStr)
			if !ok || n < 0 {
				return nil, RangeMalformed
			}

			start = size - n
			end = size - 1

			if start < 0 {
				start = 0
			}

		case endStr == "":
			n, ok := parseInt(startStr)
			if !ok || n < 0 {
				return nil, RangeMalformed
			}

			start = n
			end = size - 1

		default:
			s, ok1 := parseInt(startStr)
			e, ok2 := parseInt(endStr)
			if !ok1 || !ok2 || s < 0 || e < 0 {
				return nil, RangeMalformed
			}

			start = s
			end = e
		}

		if start > end {
			return nil, RangeMalformed
		}

		if start >= size {
			continue
		}

		if end >= size {
			end = size - 1
		}

		ranges = append(ranges, Range{Start: start, End: end})
	}

	if len(ranges) == 0 {
		return nil, RangeUnsatisfiable
	}

	result := &RangeResult{
		Type:   rangeType,
		Ranges: ranges,
	}

	if combine {
		result.Ranges = combineRanges(result.Ranges)
	}

	return result, 0
}

func combineRanges(ranges []Range) []Range {
	if len(ranges) <= 1 {
		return ranges
	}

	for i := 1; i < len(ranges); i++ {
		for j := i; j > 0 && ranges[j].Start < ranges[j-1].Start; j-- {
			ranges[j], ranges[j-1] = ranges[j-1], ranges[j]
		}
	}

	merged := make([]Range, 0, len(ranges))
	cur := ranges[0]

	for i := 1; i < len(ranges); i++ {
		r := ranges[i]

		if cur.End+1 >= r.Start {
			if r.End > cur.End {
				cur.End = r.End
			}
		} else {
			merged = append(merged, cur)
			cur = r
		}
	}

	merged = append(merged, cur)
	return merged
}

func parseInt(s string) (int64, bool) {
	if len(s) == 0 {
		return 0, false
	}

	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int64(c-'0')
	}

	return n, true
}

func appendReadAll(dst []byte, r io.Reader) ([]byte, error) {
	for {
		if len(dst) == cap(dst) {
			newCap := cap(dst) * 2
			if newCap == 0 {
				newCap = 512
			}

			tmp := make([]byte, len(dst), newCap)
			copy(tmp, dst)

			dst = tmp
		}

		n, err := r.Read(dst[len(dst):cap(dst)])
		dst = dst[:len(dst)+n]

		if err != nil {
			if err == io.EOF {
				return dst, nil
			}

			return dst, err
		}
	}
}
