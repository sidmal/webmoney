package webmoney

import (
	"errors"
	"github.com/sidmal/webmoney/mocks"
	"github.com/sidmal/webmoney/signer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"net/url"
	"testing"
	"time"
)

const (
	TestWmId     = "405002833238"
	TestKey      = "gQABADCWZW2w1EMgHCYswfVPdf6MAAAAAAAAAEIADHN9yDTlBIQnJd4W/Rk+UDGhrYiYoC5yVGjSkV9GFSkLFKgMk2r2bJDnFUAub2sc9vjXbpkcUlS8QX60Ti83ECQXbomCybZS4zN/pO0IJU77H3FBeFOvjh32PLswJaEqKGCIgU7lydVsT7KBJd9vfNhYaRNVnbH5NQdF+nmDv373G+Ovt9Y="
	TestPassword = "FvGqPdAy8reVWw789"
)

type WebmoneyTestSuite struct {
	suite.Suite
	webmoney       *WebMoney
	defaultOptions []Option
	logObserver    *zap.Logger
	zapRecorder    *observer.ObservedLogs
}

func Test_Webmoney(t *testing.T) {
	suite.Run(t, new(WebmoneyTestSuite))
}

func (suite *WebmoneyTestSuite) SetupTest() {
	suite.defaultOptions = []Option{
		WmId(TestWmId),
		Key(TestKey),
		Password(TestPassword),
		httpClient(mocks.NewTransportStatusOk()),
	}
	wm, err := NewWebMoney(suite.defaultOptions...)

	if err != nil {
		suite.FailNow("WebMoney handler initialization failed", "%v", err)
	}

	assert.NotNil(suite.T(), wm)

	wmt, ok := wm.(*WebMoney)
	assert.True(suite.T(), ok)
	assert.NotNil(suite.T(), wmt.signer)
	assert.NotNil(suite.T(), wmt.options)
	assert.NotNil(suite.T(), wmt.unMarshalFn)
	assert.NotNil(suite.T(), wmt.marshalFn)

	suite.webmoney = wmt

	var core zapcore.Core

	lvl := zap.NewAtomicLevel()
	core, suite.zapRecorder = observer.New(lvl)
	suite.logObserver = zap.New(core)
}

func (suite *WebmoneyTestSuite) TestWebMoney_NewWebMoney_HttpClientNotSet_Ok() {
	suite.defaultOptions = append(suite.defaultOptions, httpClient(nil))
	wm, err := NewWebMoney(suite.defaultOptions...)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), wm)
}

func (suite *WebmoneyTestSuite) TestWebMoney_NewWebMoney_ExecuteOptions_ErrorWmIdNotConfigured_Error() {
	wm, err := NewWebMoney([]Option{}...)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), signer.ErrorWmIdNotConfigured, err)
	assert.Nil(suite.T(), wm)
}

func (suite *WebmoneyTestSuite) TestWebMoney_NewWebMoney_ExecuteOptions_ErrorWmIdIsIncorrect_Error() {
	suite.defaultOptions = append(suite.defaultOptions, WmId("0123456789"))
	wm, err := NewWebMoney(suite.defaultOptions...)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), signer.ErrorWmIdIsIncorrect, err)
	assert.Nil(suite.T(), wm)
}

func (suite *WebmoneyTestSuite) TestWebMoney_NewWebMoney_ExecuteOptions_ErrorKeyNotConfigured_Error() {
	suite.defaultOptions = append(suite.defaultOptions, Key(""))
	wm, err := NewWebMoney(suite.defaultOptions...)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), signer.ErrorKeyNotConfigured, err)
	assert.Nil(suite.T(), wm)
}

func (suite *WebmoneyTestSuite) TestWebMoney_NewWebMoney_ExecuteOptions_ErrorPasswordNotConfigured_Error() {
	suite.defaultOptions = append(suite.defaultOptions, Password(""))
	wm, err := NewWebMoney(suite.defaultOptions...)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), signer.ErrorPasswordNotConfigured, err)
	assert.Nil(suite.T(), wm)
}

func (suite *WebmoneyTestSuite) TestWebMoney_NewWebMoney_NewSigner_Error() {
	suite.defaultOptions = append(suite.defaultOptions, Key("0123456789"))
	wm, err := NewWebMoney(suite.defaultOptions...)
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "illegal base64 data at input byte 8")
	assert.Nil(suite.T(), wm)
}

func (suite *WebmoneyTestSuite) TestWebMoney_NewWebMoney_CaCert_IoUtil_ReadAll_Error() {
	suite.defaultOptions = append(suite.defaultOptions, rootCaReader(&mocks.IoReaderError{}))
	suite.defaultOptions = append(suite.defaultOptions, httpClient(nil))
	wm, err := NewWebMoney(suite.defaultOptions...)
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "SomeError")
	assert.Nil(suite.T(), wm)
}

func (suite *WebmoneyTestSuite) TestWebMoney_TransferMoney_Ok() {
	in := &TransferMoneyRequest{
		TxnId:     1234567890,
		PurseSrc:  "Z123456789012",
		PurseDest: "Z0987654321098",
		Amount:    "10.00",
		Period:    0,
		Desc:      "Тестовая операция",
		PCode:     "",
		WmInvId:   0,
		OnlyAuth:  1,
	}
	result, err := suite.webmoney.TransferMoney(in)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.NotZero(suite.T(), result.Id)
	assert.NotZero(suite.T(), result.Ts)
	assert.NotZero(suite.T(), result.TxnId)
	assert.NotZero(suite.T(), result.PurseSrc)
	assert.NotZero(suite.T(), result.PurseDest)
	assert.NotZero(suite.T(), result.Amount)
	assert.NotZero(suite.T(), result.Commission)
	assert.NotZero(suite.T(), result.Desc)
	assert.NotZero(suite.T(), result.DateCrt)
	assert.NotZero(suite.T(), result.DateUpd)
}

func (suite *WebmoneyTestSuite) TestWebMoney_TransferMoney_Error() {
	suite.webmoney.httpClient = mocks.NewTransportStatusWmError()
	in := &TransferMoneyRequest{
		TxnId:     1234567890,
		PurseSrc:  "Z123456789012",
		PurseDest: "Z0987654321098",
		Amount:    "10.00",
		Period:    0,
		Desc:      "Тестовая операция",
		PCode:     "",
		WmInvId:   0,
		OnlyAuth:  1,
	}
	result, err := suite.webmoney.TransferMoney(in)
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "Mock error")
	assert.Nil(suite.T(), result)
}

func (suite *WebmoneyTestSuite) TestWebMoney_GetTransactionsHistory_Ok() {
	t := time.Now().Format("20060102 15:04:05")
	in := &GetTransactionsHistoryRequest{
		Purse:      "Z123456789012",
		TxnId:      1234567890,
		DateStart:  t,
		DateFinish: t,
	}
	result, err := suite.webmoney.GetTransactionsHistory(in)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.NotZero(suite.T(), result.Count)
	assert.NotEmpty(suite.T(), result.OperationList)
	assert.Len(suite.T(), result.OperationList, 1)
	assert.NotNil(suite.T(), result.OperationList[0])
	assert.NotZero(suite.T(), result.OperationList[0].Id)
	assert.NotZero(suite.T(), result.OperationList[0].Ts)
	assert.NotZero(suite.T(), result.OperationList[0].TxnId)
	assert.NotZero(suite.T(), result.OperationList[0].PurseSrc)
	assert.NotZero(suite.T(), result.OperationList[0].PurseDest)
	assert.NotZero(suite.T(), result.OperationList[0].Amount)
	assert.NotZero(suite.T(), result.OperationList[0].Commission)
	assert.NotZero(suite.T(), result.OperationList[0].Desc)
	assert.NotZero(suite.T(), result.OperationList[0].DateCrt)
	assert.NotZero(suite.T(), result.OperationList[0].DateUpd)
}

func (suite *WebmoneyTestSuite) TestWebMoney_GetTransactionsHistory_Error() {
	suite.webmoney.httpClient = mocks.NewTransportStatusWmError()
	t := time.Now().Format("20060102 15:04:05")
	in := &GetTransactionsHistoryRequest{
		Purse:      "Z123456789012",
		TxnId:      1234567890,
		DateStart:  t,
		DateFinish: t,
	}
	result, err := suite.webmoney.GetTransactionsHistory(in)
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "Mock error")
	assert.Nil(suite.T(), result)
}

func (suite *WebmoneyTestSuite) TestWebMoney_GetBalance_Ok() {
	in := &GetBalanceRequest{
		Wmid: TestWmId,
	}
	result, err := suite.webmoney.GetBalance(in)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.NotZero(suite.T(), result.Count)
	assert.NotEmpty(suite.T(), result.PurseList)
	assert.Len(suite.T(), result.PurseList, 1)
	assert.NotNil(suite.T(), result.PurseList[0])
	assert.NotNil(suite.T(), result.PurseList[0].PurseName)
	assert.NotNil(suite.T(), result.PurseList[0].Amount)
	assert.NotNil(suite.T(), result.PurseList[0].Desc)
	assert.NotNil(suite.T(), result.PurseList[0].OutsideOpen)
	assert.NotNil(suite.T(), result.PurseList[0].LastIncomeTxmId)
	assert.NotNil(suite.T(), result.PurseList[0].LastOutcomeTxnId)
}

func (suite *WebmoneyTestSuite) TestWebMoney_GetBalance_Error() {
	suite.webmoney.httpClient = mocks.NewTransportStatusWmError()
	in := &GetBalanceRequest{
		Wmid: TestWmId,
	}
	result, err := suite.webmoney.GetBalance(in)
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "Mock error")
	assert.Nil(suite.T(), result)
}

func (suite *WebmoneyTestSuite) TestWebMoney_GetBalance_SendRequest_Signer_Sign_Error() {
	mockSigner := &mocks.WebMoneySignerInterface{}
	mockSigner.On("Sign", mock.Anything).
		Return("", errors.New("TestWebMoney_GetBalance_SendRequest_Signer_Sign_Error"))
	suite.webmoney.signer = mockSigner
	in := &GetBalanceRequest{
		Wmid: TestWmId,
	}
	result, err := suite.webmoney.GetBalance(in)
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "TestWebMoney_GetBalance_SendRequest_Signer_Sign_Error")
	assert.Nil(suite.T(), result)
}

func (suite *WebmoneyTestSuite) TestWebMoney_GetBalance_SendRequest_XmlMarshal_Error() {
	suite.webmoney.marshalFn = func(v interface{}) ([]byte, error) {
		return nil, errors.New("TestWebMoney_GetBalance_SendRequest_XmlMarshal_Error")
	}
	in := &GetBalanceRequest{
		Wmid: TestWmId,
	}
	result, err := suite.webmoney.GetBalance(in)
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "TestWebMoney_GetBalance_SendRequest_XmlMarshal_Error")
	assert.Nil(suite.T(), result)
}

func (suite *WebmoneyTestSuite) TestWebMoney_GetBalance_SendRequest_Http_Do_Error() {
	suite.webmoney.httpClient = mocks.NewTransportStatusError()
	in := &GetBalanceRequest{
		Wmid: TestWmId,
	}
	result, err := suite.webmoney.GetBalance(in)
	assert.Error(suite.T(), err)

	errT, ok := err.(*url.Error)
	assert.True(suite.T(), ok)
	assert.EqualError(suite.T(), errT.Err, "TransportStatusError")
	assert.Nil(suite.T(), result)
}

func (suite *WebmoneyTestSuite) TestWebMoney_GetBalance_SendRequest_IoUtil_ReadAll_Error() {
	suite.webmoney.httpClient = mocks.NewTransportStatusErrorIoReader()
	in := &GetBalanceRequest{
		Wmid: TestWmId,
	}
	result, err := suite.webmoney.GetBalance(in)
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "SomeError")
	assert.Nil(suite.T(), result)
}

func (suite *WebmoneyTestSuite) TestWebMoney_GetBalance_SendRequest_XmlUnMarshal_Error() {
	suite.webmoney.unMarshalFn = func(_ []byte, _ interface{}) error {
		return errors.New("TestWebMoney_GetBalance_SendRequest_XmlUnMarshal_Error")
	}
	in := &GetBalanceRequest{
		Wmid: TestWmId,
	}
	result, err := suite.webmoney.GetBalance(in)
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "TestWebMoney_GetBalance_SendRequest_XmlUnMarshal_Error")
	assert.Nil(suite.T(), result)
}

func (suite *WebmoneyTestSuite) TestWebMoney_SendRequest_Http_NewRequest_Error() {
	result, err := suite.webmoney.sendRequest("\n", new(BaseRequest), new(GetBalanceResponse))
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
}
