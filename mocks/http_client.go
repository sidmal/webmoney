package mocks

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type TransportStatusOk struct {
	Transport http.RoundTripper
}

type TransportStatusWmError struct {
	Transport http.RoundTripper
}

type TransportStatusError struct {
	Transport http.RoundTripper
}

type TransportStatusErrorIoReader struct {
	Transport http.RoundTripper
}

type IoReaderError struct{}

func NewTransportStatusOk() *http.Client {
	return &http.Client{
		Transport: &TransportStatusOk{},
	}
}

func NewTransportStatusWmError() *http.Client {
	return &http.Client{
		Transport: &TransportStatusWmError{},
	}
}

func NewTransportStatusError() *http.Client {
	return &http.Client{
		Transport: &TransportStatusError{},
	}
}

func NewTransportStatusErrorIoReader() *http.Client {
	return &http.Client{
		Transport: &TransportStatusErrorIoReader{},
	}
}

func (m *TransportStatusOk) RoundTrip(req *http.Request) (*http.Response, error) {
	body := ""
	t := time.Now().Format("20060102 15:04:05")

	switch req.URL.Path {
	case "/asp/XMLTrans.asp":
		body = `<w3s.response><reqn>1234567890</reqn><retval>0</retval><retdesc>Ok</retdesc><operation id="123" ts="456"><tranid>1234567890</tranid><pursesrc>Z123456789012</pursesrc><pursedest>Z0987654321098</pursedest><amount>100.00</amount><comiss>0.8</comiss><opertype>0</opertype><period>0</period><wminvid>0</wminvid><orderid>0</orderid><desc>Mock test</desc><datecrt>` + t + `</datecrt><dateupd>` + t + `</dateupd></operation></w3s.response>`
		break
	case "/asp/XMLOperations.asp":
		body = `<w3s.response><reqn>1234567890</reqn><retval>0</retval><retdesc>Ok</retdesc><operations cnt="1"><operation id="123" ts="456"><tranid>1234567890</tranid><pursesrc>Z123456789012</pursesrc><pursedest>Z0987654321098</pursedest><amount>100.00</amount><comiss>0.8</comiss><opertype>0</opertype><period>0</period><wminvid>0</wminvid><orderid>0</orderid><desc>Mock test</desc><datecrt>` + t + `</datecrt><dateupd>` + t + `</dateupd></operation></operations></w3s.response>`
		break
	case "/asp/XMLPurses.asp":
		body = `<w3s.response><reqn>1234567890</reqn><retval>0</retval><retdesc>Ok</retdesc><purses cnt="1"><purse id="Z123456789012"><pursename>Mock purse</pursename><amount>112345.45</amount><desc>Тестовый кошелек</desc><outsideopen>0</outsideopen><lastintr>123</lastintr><lastouttr>321</lastouttr></purse></purses></w3s.response>`
		break
	default:
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       ioutil.NopCloser(strings.NewReader(`{}`)),
			Header:     make(http.Header),
		}, nil
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}, nil
}

func (m *TransportStatusWmError) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(strings.NewReader(`<w3s.response><reqn>1234567890</reqn><retval>-999</retval><retdesc>Mock error</retdesc></w3s.response>`)),
		Header:     make(http.Header),
	}, nil
}

func (m *TransportStatusError) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, errors.New("TransportStatusError")
}

func (h *TransportStatusErrorIoReader) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       ioutil.NopCloser(&IoReaderError{}),
		Header:     make(http.Header),
	}, nil
}

func (m *IoReaderError) Read(_ []byte) (int, error) {
	return 0, errors.New("SomeError")
}
