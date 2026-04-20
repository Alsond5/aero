package recovery

import "github.com/Alsond5/aero"

// Config defines the settings for the Recovery middleware.
type Config struct {
	// Skip defines a function to skip this middleware when returned true.
	// Use it to exclude specific routes or conditions from panic recovery.
	//
	// Optional. Default: nil
	Skip func(c *aero.Ctx) bool

	// PanicHandler defines a function that will be executed when a panic occurs.
	// This allows you to define a custom response (JSON, XML, HTML, etc.)
	// when the server recovers from a panic.
	//
	// Optional. Default: sends a "500 Internal Server Error" response.
	PanicHandler func(c *aero.Ctx, r any) error

	// StackTraceHandler defines a function that will be executed to handle
	// the recovered stack trace. Useful for sending traces to external
	// monitoring tools like Sentry, Datadog, or custom loggers.
	//
	// Optional. Default: nil
	StackTraceHandler func(c *aero.Ctx, e any)

	// EnableStackTrace enables or disables stack trace collection.
	// While useful for debugging, disabling it can slightly improve performance
	// in high-throughput production environments.
	//
	// Optional. Default: false
	EnableStackTrace bool
}

// ConfigDefault is the default configuration for the Recovery middleware.
var ConfigDefault = Config{
	Skip:             nil,
	EnableStackTrace: false,
	PanicHandler:     DefaultPanicHandler,
}

func getConfig(config ...Config) Config {
	if len(config) == 0 {
		return ConfigDefault
	}

	cfg := config[0]

	if cfg.EnableStackTrace && cfg.StackTraceHandler == nil {
		cfg.StackTraceHandler = defaultStackTraceHandler
	}

	if cfg.PanicHandler == nil {
		cfg.PanicHandler = DefaultPanicHandler
	}

	return cfg
}
