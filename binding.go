package aero

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
)

var (
	ErrBindingTargetInvalid = errors.New("aero: binding target must be a pointer to a struct")
	ErrUnsupportedFieldType = errors.New("aero: unsupported field type")
)

// Bind parses the request into v by reading each struct field's tag to
// determine the source. Supported tags are "json", "xml", "form", "query",
// "param", and "header". A single struct can mix multiple sources; Bind
// fills each field from the appropriate location.
//
//	type CreateUserReq struct {
//		OrgID  string `param:"orgId"`
//		Token  string `header:"X-Auth-Token"`
//		Page   int    `query:"page"`
//		Name   string `json:"name"`
//	}
//
//	var req CreateUserReq
//	if err := c.Req.Bind(&req); err != nil {
//		return err
//	}
func (req *Req) Bind(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Struct {
		return ErrBindingTargetInvalid
	}

	rv = rv.Elem()
	rt := rv.Type()
	n := rt.NumField()

	ct := req.c.r.Header.Get(HeaderContentType)

	var (
		needsQuery     bool
		needsParam     bool
		needsHeader    bool
		needsForm      bool
		needsMultipart bool
	)

	for i := range n {
		f := rt.Field(i)
		if f.Tag.Get("query") != "" {
			needsQuery = true
		}
		if f.Tag.Get("param") != "" {
			needsParam = true
		}
		if f.Tag.Get("header") != "" {
			needsHeader = true
		}
		if f.Tag.Get("form") != "" {
			if isMultipart(ct) {
				needsMultipart = true
			} else {
				needsForm = true
			}
		}
	}

	switch {
	case isJSON(ct):
		if err := req.BindJSON(v); err != nil {
			return err
		}
	case isXML(ct):
		if err := req.BindXML(v); err != nil {
			return err
		}
	case needsMultipart:
		if err := req.parseMultipartForm(); err != nil {
			return err
		}
	case needsForm:
		if err := req.parseForm(); err != nil {
			return err
		}
	}

	if !needsQuery && !needsParam && !needsHeader && !needsForm && !needsMultipart {
		return nil
	}

	var queryVals url.Values
	if needsQuery {
		queryVals = req.QueryAll()
	}

	var formVals url.Values
	if (needsForm || needsMultipart) && req.c.r.Form != nil {
		formVals = req.c.r.Form
	}

	for i := range n {
		field := rt.Field(i)
		fv := rv.Field(i)

		if !fv.CanSet() {
			continue
		}

		if needsParam {
			if key := field.Tag.Get("param"); key != "" {
				if val := req.Param(key); val != "" {
					if err := setScalar(fv, val); err != nil {
						return fmt.Errorf("aero: field %s: %w", field.Name, err)
					}
				}

				continue
			}
		}

		if needsQuery {
			if key := field.Tag.Get("query"); key != "" {
				if vals := queryVals[key]; len(vals) > 0 {
					if err := setField(fv, field.Type, vals); err != nil {
						return fmt.Errorf("aero: field %s: %w", field.Name, err)
					}
				}

				continue
			}
		}

		if needsHeader {
			if key := field.Tag.Get("header"); key != "" {
				if val := req.c.r.Header.Get(key); val != "" {
					if err := setScalar(fv, val); err != nil {
						return fmt.Errorf("aero: field %s: %w", field.Name, err)
					}
				}

				continue
			}
		}

		if formVals != nil {
			if key := field.Tag.Get("form"); key != "" {
				if vals := formVals[key]; len(vals) > 0 {
					if err := setField(fv, field.Type, vals); err != nil {
						return fmt.Errorf("aero: field %s: %w", field.Name, err)
					}
				}
			}
		}
	}

	return nil
}

// BindJSON decodes the request body as JSON into v.
// Returns an error if the Content-Type is not application/json or
// if decoding fails.
func (req *Req) BindJSON(v any) error {
	if req.c.r.Body == nil {
		return ErrBodyAlreadyRead
	}

	defer func() {
		req.c.r.Body.Close() //nolint:errcheck
		req.c.r.Body = nil
	}()

	if req.c.app.config.MaxBodySize > 0 {
		req.c.r.Body = http.MaxBytesReader(req.c.w, req.c.r.Body, req.c.app.config.MaxBodySize)
	}

	return json.NewDecoder(req.c.r.Body).Decode(v)
}

// BindXML decodes the request body as XML into v.
// Returns an error if the Content-Type is not application/xml or
// if decoding fails.
func (req *Req) BindXML(v any) error {
	if req.c.r.Body == nil {
		return ErrBodyAlreadyRead
	}

	defer func() {
		req.c.r.Body.Close() //nolint:errcheck
		req.c.r.Body = nil
	}()

	if req.c.app.config.MaxBodySize > 0 {
		req.c.r.Body = http.MaxBytesReader(req.c.w, req.c.r.Body, req.c.app.config.MaxBodySize)
	}

	return xml.NewDecoder(req.c.r.Body).Decode(v)
}

// BindForm parses the request body as application/x-www-form-urlencoded or
// multipart/form-data and maps the values into v using the "form" struct tag.
func (req *Req) BindForm(v any) error {
	ct := req.c.r.Header.Get(HeaderContentType)

	var err error
	if isMultipart(ct) {
		err = req.parseMultipartForm()
	} else {
		err = req.parseForm()
	}
	if err != nil {
		return err
	}

	return mapValues(v, req.c.r.Form, "form")
}

// BindQuery maps URL query parameters into v using the "query" struct tag.
//
//	type SearchReq struct {
//		Q    string `query:"q"`
//		Page int    `query:"page"`
//	}
func (req *Req) BindQuery(v any) error {
	return mapValues(v, req.QueryAll(), "query")
}

// BindParams maps URL path parameters into v using the "param" struct tag.
//
//	type UserReq struct {
//		ID string `param:"id"`
//	}
func (req *Req) BindParams(v any) error {
	values := make(map[string][]string, req.c.paramsCount)
	for i := range req.c.paramsCount {
		values[req.c.params[i].Key] = []string{req.c.params[i].Value}
	}

	return mapValues(v, values, "param")
}

// BindHeaders maps request headers into v using the "header" struct tag.
// Header names are case-insensitive.
//
//	type AuthReq struct {
//		Token string `header:"X-Auth-Token"`
//	}
func (req *Req) BindHeaders(v any) error {
	return mapValues(v, map[string][]string(req.c.r.Header), "header")
}

func mapValues(v any, values map[string][]string, tag string) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Struct {
		return ErrBindingTargetInvalid
	}

	rv = rv.Elem()
	rt := rv.Type()

	for i := range rt.NumField() {
		field := rt.Field(i)
		fv := rv.Field(i)

		if !fv.CanSet() {
			continue
		}

		key := field.Tag.Get(tag)
		if key == "" {
			key = field.Name
		}

		vals, ok := values[key]
		if !ok || len(vals) == 0 {
			continue
		}

		if err := setField(fv, field.Type, vals); err != nil {
			return fmt.Errorf("aero: field %s: %w", field.Name, err)
		}
	}

	return nil
}

func setField(fv reflect.Value, ft reflect.Type, vals []string) error {
	if ft.Kind() != reflect.Slice {
		return setScalar(fv, vals[0])
	}

	slice := reflect.MakeSlice(ft, len(vals), len(vals))
	for i, val := range vals {
		if err := setScalar(slice.Index(i), val); err != nil {
			return err
		}
	}
	fv.Set(slice)

	return nil
}

func setScalar(fv reflect.Value, s string) error {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(s)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		fv.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		fv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		fv.SetFloat(n)
	default:
		return ErrUnsupportedFieldType
	}

	return nil
}

func isJSON(ct string) bool {
	return ct == MIMEApplicationJSON || hasPrefix(ct, MIMEApplicationJSON+";")
}

func isXML(ct string) bool {
	return ct == MIMEApplicationXML || hasPrefix(ct, MIMEApplicationXML+";") ||
		ct == MIMETextXML || hasPrefix(ct, MIMETextXML+";")
}

func isMultipart(ct string) bool {
	return hasPrefix(ct, MIMEMultipartForm)
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
