package webmoney

import (
	"github.com/sidmal/webmoney/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"net/http"
	"strings"
	"testing"
)

type HttpTestSuite struct {
	suite.Suite
	httpTransport *httpTransport
	logObserver   *zap.Logger
	zapRecorder   *observer.ObservedLogs
}

func Test_Http(t *testing.T) {
	suite.Run(t, new(HttpTestSuite))
}

func (suite *HttpTestSuite) SetupTest() {
	var core zapcore.Core

	lvl := zap.NewAtomicLevel()
	core, suite.zapRecorder = observer.New(lvl)
	suite.logObserver = zap.New(core)

	suite.httpTransport = newHttpTransport(
		suite.logObserver,
		func(req *http.Request) *http.Request {
			return req
		},
		nil,
	)
}

func (suite *HttpTestSuite) TestHttpTransport_RoundTrip_WithoutLog_Ok() {
	httpTransport := &httpTransport{
		transport: &mocks.TransportStatusOk{},
	}
	req, err := http.NewRequest(http.MethodPost, "http://localhost", strings.NewReader(`{}`))
	assert.NoError(suite.T(), err)

	assert.Empty(suite.T(), suite.zapRecorder.All())
	rsp, err := httpTransport.RoundTrip(req)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), rsp)
	assert.Empty(suite.T(), suite.zapRecorder.All())
}

func (suite *HttpTestSuite) TestHttpTransport_RoundTrip_WithLog_Ok() {
	req, err := http.NewRequest(http.MethodPost, "http://localhost", strings.NewReader(`{}`))
	assert.NoError(suite.T(), err)

	suite.httpTransport.transport = &mocks.TransportStatusOk{}

	assert.Empty(suite.T(), suite.zapRecorder.All())
	rsp, err := suite.httpTransport.RoundTrip(req)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), rsp)

	logs := suite.zapRecorder.All()
	assert.NotEmpty(suite.T(), logs)
	assert.Len(suite.T(), logs, 1)
}

func (suite *HttpTestSuite) TestHttpTransport_RoundTrip_ReadResponseBody_Error() {
	req, err := http.NewRequest(http.MethodPost, "http://localhost", strings.NewReader(`{}`))
	assert.NoError(suite.T(), err)

	suite.httpTransport.transport = &mocks.TransportStatusErrorIoReader{}

	assert.Empty(suite.T(), suite.zapRecorder.All())
	_, err = suite.httpTransport.RoundTrip(req)
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "SomeError")
	assert.Empty(suite.T(), suite.zapRecorder.All())
}
