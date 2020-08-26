package webmoney

import (
	"go.uber.org/zap"
	"io"
	"net/http"
)

type Options struct {
	// The WebMoney's WMID identifier
	wmId string
	// The WebMoney Pro *.kvm key
	key string
	// The password to WebMoney Pro *.kvm key
	password string
	// The HTTP client to send requests
	httpClient *http.Client
	// The reader for WebMoney root certificate
	rootCaReader io.Reader
	// The logger
	logger *zap.Logger
	// The func to clear log before save
	logClearFn func(req *http.Request) *http.Request
}

type Option func(*Options)

func WmId(val string) Option {
	return func(opts *Options) {
		opts.wmId = val
	}
}

func Key(val string) Option {
	return func(opts *Options) {
		opts.key = val
	}
}

func Password(val string) Option {
	return func(opts *Options) {
		opts.password = val
	}
}

func HttpClient(val *http.Client) Option {
	return func(opts *Options) {
		opts.httpClient = val
	}
}

func RootCaReader(val io.Reader) Option {
	return func(opts *Options) {
		opts.rootCaReader = val
	}
}

func Logger(val *zap.Logger) Option {
	return func(opts *Options) {
		opts.logger = val
	}
}

func LogClearFn(val func(req *http.Request) *http.Request) Option {
	return func(opts *Options) {
		opts.logClearFn = val
	}
}
