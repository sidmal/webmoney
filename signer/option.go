package signer

type Options struct {
	// The WebMoney's WMID identifier
	wmId string
	// The WebMoney Pro *.kvm TestKey
	key string
	// The TestPassword to WebMoney Pro *.kvm TestKey
	password string
	// The handler to initialize WebMoney TestKey container
	newKeyContainerFn func(key []byte, wmid, keyPassword string, reader ExternalInterface) (KeyContainerInterface, error)
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

func NewKeyContainerFn(val func(key []byte, wmid, keyPassword string, reader ExternalInterface) (KeyContainerInterface, error)) Option {
	return func(opts *Options) {
		opts.newKeyContainerFn = val
	}
}
