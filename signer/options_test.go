package signer

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSignerOptions_Setters(t *testing.T) {
	opts := []Option{
		WmId("123456789012"),
		Key("TestKey"),
		Password("TestPassword"),
		NewKeyContainerFn(newKeyContainer),
	}

	options := &Options{}

	for _, opt := range opts {
		opt(options)
	}

	assert.Equal(t, "123456789012", options.wmId)
	assert.Equal(t, "TestKey", options.key)
	assert.Equal(t, "TestPassword", options.password)
	assert.NotNil(t, options.newKeyContainerFn)
}
