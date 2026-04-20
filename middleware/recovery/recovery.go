package recovery

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"unsafe"

	"github.com/Alsond5/aero"
)

// New creates a new recovery middleware handler
func New(config ...Config) aero.HandlerFunc {
	cfg := getConfig(config...)

	return func(c *aero.Ctx) (err error) {
		if cfg.Skip != nil && cfg.Skip(c) {
			return c.Next()
		}

		defer func() {
			if r := recover(); r != nil {
				if cfg.EnableStackTrace {
					cfg.StackTraceHandler(c, r)
				}

				err = cfg.PanicHandler(c, r)
			}
		}()

		return c.Next()
	}
}
func DefaultPanicHandler(_ *aero.Ctx, r any) error {
	switch v := r.(type) {
	case error:
		return v
	case string:
		return errors.New(v)
	case []byte:
		return errors.New(unsafe.String(unsafe.SliceData(v), len(v)))
	case int:
		return errors.New(strconv.Itoa(v))
	case bool:
		return errors.New(strconv.FormatBool(v))
	default:
		return fmt.Errorf("unknown panic: %v", v)
	}
}

func defaultStackTraceHandler(_ *aero.Ctx, e any) {
	_, _ = os.Stderr.WriteString("panic: ")

	switch v := e.(type) {
	case error:
		_, _ = os.Stderr.WriteString(v.Error())
	case string:
		_, _ = os.Stderr.WriteString(v)
	case []byte:
		_, _ = os.Stderr.Write(v)
	case int:
		var buf [20]byte
		_, _ = os.Stderr.Write(strconv.AppendInt(buf[:0], int64(v), 10))
	default:
		_, _ = fmt.Fprintf(os.Stderr, "recovered from non-standard panic type: %v", v)
	}

	_, _ = os.Stderr.WriteString("\n\n")
	_, _ = os.Stderr.Write(debug.Stack())
	_, _ = os.Stderr.WriteString("\n")
}
