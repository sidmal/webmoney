package webmoney

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"testing"
)

func TestWebmoneyOptions_Setters(t *testing.T) {
	httpClient := &http.Client{}
	rootCaReader := strings.NewReader(``)
	logger, err := zap.NewProduction()
	assert.NoError(t, err)
	logClearFn := func(req *http.Request) *http.Request {
		return req
	}

	opts := []Option{
		WmId("123456789012"),
		Key("key"),
		Password("password"),
		HttpClient(httpClient),
		RootCaReader(rootCaReader),
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
	assert.EqualValues(t, httpClient, options.httpClient)
	assert.EqualValues(t, rootCaReader, options.rootCaReader)
	assert.EqualValues(t, logger, options.logger)
	assert.NotNil(t, options.logClearFn)
}
