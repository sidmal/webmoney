package webmoney

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestWebmoneyOptions_Setters(t *testing.T) {
	httpClient := &http.Client{}
	opts := []Option{
		WmId("123456789012"),
		Key("key"),
		Password("password"),
		HttpClient(httpClient),
	}

	options := &Options{}

	for _, opt := range opts {
		opt(options)
	}

	assert.EqualValues(t, "123456789012", options.wmId)
	assert.EqualValues(t, "key", options.key)
	assert.EqualValues(t, "password", options.password)
	assert.EqualValues(t, httpClient, options.httpClient)
}
