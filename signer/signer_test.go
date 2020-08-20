package signer

import (
	"errors"
	"github.com/sidmal/webmoney/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"testing"
)

const (
	TestWmId     = "405002833238"
	TestKey      = "gQABADCWZW2w1EMgHCYswfVPdf6MAAAAAAAAAEIADHN9yDTlBIQnJd4W/Rk+UDGhrYiYoC5yVGjSkV9GFSkLFKgMk2r2bJDnFUAub2sc9vjXbpkcUlS8QX60Ti83ECQXbomCybZS4zN/pO0IJU77H3FBeFOvjh32PLswJaEqKGCIgU7lydVsT7KBJd9vfNhYaRNVnbH5NQdF+nmDv373G+Ovt9Y="
	TestPassword = "FvGqPdAy8reVWw789"
)

type SignerTestSuite struct {
	suite.Suite
	signer         *Signer
	defaultOptions []Option
}

func Test_Signer(t *testing.T) {
	suite.Run(t, new(SignerTestSuite))
}

func (suite *SignerTestSuite) SetupTest() {
	suite.defaultOptions = []Option{
		WmId(TestWmId),
		Key(TestKey),
		Password(TestPassword),
	}
	signer, err := NewSigner(suite.defaultOptions...)

	if err != nil {
		suite.FailNow("Signer initialization failed", "%v", err)
	}

	assert.NotNil(suite.T(), signer)

	signerT, ok := signer.(*Signer)
	assert.True(suite.T(), ok)
	assert.NotNil(suite.T(), signerT.power)
	assert.NotNil(suite.T(), signerT.modulus)

	suite.signer = signerT
}

func (suite *SignerTestSuite) TestSigner_NewSigner_ExecuteOptions_ErrorWmIdNotConfigured_Error() {
	options := []Option{
		WmId("1234"),
		Key(TestKey),
		Password(TestPassword),
	}
	signer, err := NewSigner(options...)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrorWmIdIsIncorrect, err)
	assert.Nil(suite.T(), signer)
}

func (suite *SignerTestSuite) TestSigner_NewSigner_ExecuteOptions_ErrorKeyNotConfigured_Error() {
	options := []Option{
		WmId(TestWmId),
		Password(TestPassword),
	}
	signer, err := NewSigner(options...)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrorKeyNotConfigured, err)
	assert.Nil(suite.T(), signer)
}

func (suite *SignerTestSuite) TestSigner_NewSigner_ExecuteOptions_ErrorPasswordNotConfigured_Error() {
	options := []Option{
		WmId(TestWmId),
		Key(TestKey),
	}
	signer, err := NewSigner(options...)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrorPasswordNotConfigured, err)
	assert.Nil(suite.T(), signer)
}

func (suite *SignerTestSuite) TestSigner_NewSigner_ExecuteOptions_ErrorWmIdIsIncorrect_Error() {
	options := []Option{
		Key(TestKey),
		Password(TestPassword),
	}
	signer, err := NewSigner(options...)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrorWmIdNotConfigured, err)
	assert.Nil(suite.T(), signer)
}

func (suite *SignerTestSuite) TestSigner_NewSigner_Base64DecodeString_Error() {
	suite.defaultOptions = append(suite.defaultOptions, Key("0123456789"))
	signer, err := NewSigner(suite.defaultOptions...)
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "illegal base64 data at input byte 8")
	assert.Nil(suite.T(), signer)
}

func (suite *SignerTestSuite) TestSigner_NewSigner_DecodedKeyLengthInvalid_Error() {
	suite.defaultOptions = append(suite.defaultOptions, Key("MTIzNDU2Nzg5MA=="))
	signer, err := NewSigner(suite.defaultOptions...)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrorKeyFileBroken, err)
	assert.Nil(suite.T(), signer)
}

func (suite *SignerTestSuite) TestSigner_NewSigner_NewKeyContainerFn_Error() {
	newKeyContainerFn := func(_ []byte, _, _ string, _ ExternalInterface) (KeyContainerInterface, error) {
		return nil, errors.New("TestSigner_NewSigner_NewKeyContainerFn_Error")
	}
	options := []Option{
		WmId(TestWmId),
		Key(TestKey),
		Password(TestPassword),
		NewKeyContainerFn(newKeyContainerFn),
	}
	signer, err := NewSigner(options...)
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "TestSigner_NewSigner_NewKeyContainerFn_Error")
	assert.Nil(suite.T(), signer)
}

func (suite *SignerTestSuite) TestSigner_NewSigner_KeyContainer_Verify_Error() {
	keyContainerMock := &mocks.KeyContainerInterface{}
	keyContainerMock.On("Verify").Return(false)
	newKeyContainerFn := func(_ []byte, _, _ string, _ ExternalInterface) (KeyContainerInterface, error) {
		return keyContainerMock, nil
	}
	options := []Option{
		WmId(TestWmId),
		Key(TestKey),
		Password(TestPassword),
		NewKeyContainerFn(newKeyContainerFn),
	}
	signer, err := NewSigner(options...)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ErrorKeyFileBroken, err)
	assert.Nil(suite.T(), signer)
}

func (suite *SignerTestSuite) TestSigner_NewSigner_KeyContainer_Extract_Error() {
	keyContainerMock := &mocks.KeyContainerInterface{}
	keyContainerMock.On("Verify").Return(true)
	keyContainerMock.On("Extract").Return(nil, nil, errors.New("TestSigner_NewSigner_KeyContainer_Extract_Error"))
	newKeyContainerFn := func(_ []byte, _, _ string, _ ExternalInterface) (KeyContainerInterface, error) {
		return keyContainerMock, nil
	}
	options := []Option{
		WmId(TestWmId),
		Key(TestKey),
		Password(TestPassword),
		NewKeyContainerFn(newKeyContainerFn),
	}
	signer, err := NewSigner(options...)
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "TestSigner_NewSigner_KeyContainer_Extract_Error")
	assert.Nil(suite.T(), signer)
}

func (suite *SignerTestSuite) TestSigner_Sign_Ok() {
	encrypted, err := suite.signer.Sign("1234567890")
	assert.NoError(suite.T(), err)
	assert.NotZero(suite.T(), encrypted)
}

func (suite *SignerTestSuite) TestSigner_Sign_Rand_Read_Error() {
	mockExternal := &mocks.ExternalInterface{}
	mockExternal.On("RandRead", mock.Anything).Return(0, errors.New("TestSigner_Sign_Rand_Read_Error"))
	mockExternal.On("BinaryRead", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("TestSigner_Sign_Rand_Read_Error"))
	suite.signer.external = mockExternal
	encrypted, err := suite.signer.Sign("1234567890")
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "TestSigner_Sign_Rand_Read_Error")
	assert.Zero(suite.T(), encrypted)
}

func (suite *SignerTestSuite) TestSigner_Sign_ReverseBytesAsWords_Error() {
	mockExternal := &mocks.ExternalInterface{}
	mockExternal.On("RandRead", mock.Anything).Return(0, nil)
	mockExternal.On("BinaryRead", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("TestSigner_Sign_ReverseBytesAsWords_Error"))
	suite.signer.external = mockExternal
	encrypted, err := suite.signer.Sign("1234567890")
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "TestSigner_Sign_ReverseBytesAsWords_Error")
	assert.Zero(suite.T(), encrypted)
}

func (suite *SignerTestSuite) TestSigner_NewKeyContainer_Binary_Read_Error() {
	mockExternal := &mocks.ExternalInterface{}
	mockExternal.On("BinaryRead", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("TestSigner_NewKeyContainer_Binary_Read_Error"))
	keyContainer, err := newKeyContainer([]byte("1234567890"), TestWmId, TestPassword, mockExternal)
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "TestSigner_NewKeyContainer_Binary_Read_Error")
	assert.Nil(suite.T(), keyContainer)
}

func (suite *SignerTestSuite) TestSigner_ReverseBytesAsWords_DataNotModTwo_Ok() {
	result, err := suite.signer.reverseBytesAsWords([]byte("12345678901"))
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), result)
}

func (suite *SignerTestSuite) TestSigner_KeyContainer_Extract_BinaryRead_Error() {
	mockExternal := &mocks.ExternalInterface{}
	mockExternal.On("BinaryRead", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("TestSigner_KeyContainer_Extract_BinaryRead_Error"))
	keyContainer := &keyContainer{
		key:      &keyContainerKey{},
		external: mockExternal,
	}
	_, _, err := keyContainer.Extract()
	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "TestSigner_KeyContainer_Extract_BinaryRead_Error")
}

func (suite *SignerTestSuite) TestSigner_KeyContainer_Verify_BinaryWrite_Error() {
	mockExternal := &mocks.ExternalInterface{}
	mockExternal.On("BinaryWrite", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("TestSigner_KeyContainer_Verify_BinaryWrite_Error"))
	keyContainer := &keyContainer{
		key:      &keyContainerKey{},
		external: mockExternal,
	}
	result := keyContainer.Verify()
	assert.False(suite.T(), result)
}
