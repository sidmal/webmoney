package webmoney

import "net/http"

type Options struct {
	// The WebMoney's WMID identifier
	wmId string
	// The WebMoney Pro *.kvm key
	key string
	// The password to WebMoney Pro *.kvm key
	password string
	// The HTTP client to send requests
	httpClient *http.Client
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
