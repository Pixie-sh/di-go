package di

import (
	goctx "context"
	"github.com/pixie-sh/errors-go"
	"slices"
	"time"
)

type ConfigRawData = map[string]interface{}
type Configuration interface {
	LookupNode(lookupPath string) (any, error)
}

// Context extends the standard context.NewContext interface with additional
// functionality for configuration management and access to the underlying context
type Context interface {
	goctx.Context

	RawConfiguration() ConfigRawData
	Configuration() Configuration

	Inner() goctx.Context
	Clone() Context

	Breadcrumbs() []string
	AppendBreadcrumb(token InjectionToken)
}

// context implements the Context interface and wraps the standard context
// with additional configuration data
type context struct {
	ctx goctx.Context

	rawCfg                   ConfigRawData
	cfg                      Configuration
	injectionTokenBreadcrumb []string
}

func (s *context) AppendBreadcrumb(token InjectionToken) {
	if len(token) == 0 {
		return
	}

	s.injectionTokenBreadcrumb = append(s.injectionTokenBreadcrumb, token.String())
	return
}

func (s *context) RawConfiguration() ConfigRawData {
	return s.rawCfg
}
func (s *context) Configuration() Configuration {
	return s.cfg
}

func (s *context) Deadline() (deadline time.Time, ok bool) {
	return s.ctx.Deadline()
}

func (s *context) Done() <-chan struct{} {
	return s.ctx.Done()
}

func (s *context) Err() error {
	return s.ctx.Err()
}

func (s *context) Value(key any) any {
	return s.ctx.Value(key)
}

func (s *context) Inner() goctx.Context {
	return s.ctx
}

func (s *context) Breadcrumbs() []string {
	return s.injectionTokenBreadcrumb
}

func (s *context) Clone() Context {
	return &context{
		ctx:                      s.ctx,
		rawCfg:                   s.rawCfg,
		cfg:                      s.cfg,
		injectionTokenBreadcrumb: slices.Clone(s.injectionTokenBreadcrumb),
	}
}

// NewContext creates a new Context instance with optional context and configuration data.
// It accepts variable arguments that can be a context.NewContext, Context, ConfigRawData or Configuration.
// If no context is provided, it uses context.Background().
// New NewContext will inherit configuration from parent contexts unless explicitly overridden.
func NewContext(args ...any) Context {
	var ctx goctx.Context
	var parentDiCtx *context
	var rawData ConfigRawData
	var cfg Configuration
	var err error

	for i := 0; i < len(args); i++ {
		switch v := args[i].(type) {
		case Context:
			var ok bool
			parentDiCtx, ok = v.(*context)
			if ok {
				args = append(args[:i], args[i+1:]...)
				i--
			}
		case goctx.Context:
			ctx = v
			args = append(args[:i], args[i+1:]...)
			i--
		case ConfigRawData:
			rawData = v
			args = append(args[:i], args[i+1:]...)
			i--
		case Configuration:
			cfg = v
			args = append(args[:i], args[i+1:]...)
			i--
		}
	}

	if parentDiCtx != nil {
		if rawData == nil {
			rawData = parentDiCtx.RawConfiguration()
		}

		if cfg == nil {
			cfg = parentDiCtx.Configuration()
		}

		if ctx == nil {
			ctx = parentDiCtx.Inner()
		}
	}

	if ctx == nil {
		ctx = goctx.Background()
	}

	if cfg != nil {
		rawData, err = Decode[ConfigRawData](cfg)
		errors.Must(err)
	}

	if rawData == nil {
		rawData = make(ConfigRawData)
	}

	return &context{ctx, rawData, cfg, nil}
}
