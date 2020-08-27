package webmoney

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"testing"
)

func TestWebmoneyOptions_Setters(t *testing.T) {
	httpCln := &http.Client{}
	caReader := strings.NewReader(``)
	logger, err := zap.NewProduction()
	assert.NoError(t, err)
	logClearFn := func(req *http.Request) *http.Request {
		return req
	}

	opts := []Option{
		WmId("123456789012"),
		Key("key"),
		Password("password"),
		httpClient(httpCln),
		rootCaReader(caReader),
		Logger(logger),
		LogClearFn(logClearFn),
	}

	options := &Options{}

	for _, opt := range opts {
		opt(options)
	}

	assert.EqualValues(t, "123456789012", options.wmId)
	assert.EqualValues(t, "key", options.key)
	assert.EqualValues(t, "password", options.password)
	assert.EqualValues(t, httpCln, options.httpClient)
	assert.EqualValues(t, caReader, options.rootCaReader)
	assert.EqualValues(t, logger, options.logger)
	assert.NotNil(t, options.logClearFn)
}
