package validation

import (
	"context"
	"reflect"
)

type (
	GetErrorFieldNameFunc func(f *reflect.StructField) string
	ValuerFunc            func(any) (any, bool)

	Options interface {
		ValuerFunc() ValuerFunc
		GetErrorFieldNameFunc() GetErrorFieldNameFunc
	}

	options struct {
		valuerFunc            ValuerFunc
		getErrorFieldNameFunc GetErrorFieldNameFunc
	}

	Option func(*options)
)

var _ Options = (*options)(nil)

type optionsCtxKeyType struct{}

var optionsCtxKey = optionsCtxKeyType{}

var defaultOptions = &options{
	valuerFunc:            DefaultValuer,
	getErrorFieldNameFunc: DefaultGetErrorFieldName,
}

func (o *options) ValuerFunc() ValuerFunc                       { return o.valuerFunc }
func (o *options) GetErrorFieldNameFunc() GetErrorFieldNameFunc { return o.getErrorFieldNameFunc }

func DefaultOptions() Options {
	return defaultOptions
}

func WithValuerFunc(valuerFunc ValuerFunc) Option {
	return func(o *options) {
		o.valuerFunc = valuerFunc
	}
}

func WithGetErrorFieldNameFunc(f GetErrorFieldNameFunc) Option {
	return func(o *options) {
		o.getErrorFieldNameFunc = f
	}
}

func getOpts(ctx context.Context) *options {
	if ctx != nil {
		if opts, ok := ctx.Value(optionsCtxKey).(*options); ok {
			return opts
		}
	}

	return defaultOptions
}

func GetOptions(ctx context.Context) Options {
	if ctx != nil {
		if opts, ok := ctx.Value(optionsCtxKey).(*options); ok {
			return opts
		}
	}
	return getOpts(ctx)
}

func WithOptions(ctx context.Context, opts ...Option) context.Context {
	o := getOpts(ctx)

	o2 := new(options)
	*o2 = *o

	for _, opt := range opts {
		opt(o2)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	ctx = context.WithValue(ctx, optionsCtxKey, o2)
	return ctx
}
