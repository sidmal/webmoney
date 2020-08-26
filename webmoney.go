package webmoney

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/sidmal/webmoney/signer"
	"go.uber.org/zap"
	"golang.org/x/text/encoding/charmap"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	operationTransferMoney          = "Trans"
	operationGetTransactionsHistory = "Operations"
	operationGetBalance             = "Purses"

	apiUrlMask = "https://w3s.webmoney.ru/asp/XML%s.asp"
)

type XMLInterface interface {
	TransferMoney(in *TransferMoneyRequest) (*TransferMoneyResponse, error)
	GetTransactionsHistory(in *GetTransactionsHistoryRequest) (*GetTransactionsHistoryResponse, error)
	GetBalance(in *GetBalanceRequest) (*GetBalanceResponse, error)
}

type WebMoney struct {
	options     *Options
	signer      signer.WebMoneySignerInterface
	marshalFn   func(v interface{}) ([]byte, error)
	unMarshalFn func(data []byte, v interface{}) error
	httpClient  *http.Client
}

type BaseRequest struct {
	XMLName         xml.Name    `xml:"w3s.request"`
	RequestNumber   string      `xml:"reqn"`
	WmId            string      `xml:"wmid"`
	Signature       string      `xml:"sign"`
	Request         interface{} `xml:",>"`
	SignatureString string      `xml:"-"`
}

type BaseResponse struct {
	XMLName       xml.Name    `xml:"w3s.response"`
	RequestNumber string      `xml:"reqn"`
	Code          int         `xml:"retval"`
	Reason        string      `xml:"retdesc"`
	Response      interface{} `xml:",any"`
}

type TransferMoneyRequest struct {
	XMLName   xml.Name `xml:"trans"`
	TxnId     int      `xml:"tranid"`
	PurseSrc  string   `xml:"pursesrc"`
	PurseDest string   `xml:"pursedest"`
	Amount    string   `xml:"amount"`
	Period    int      `xml:"period"`
	Desc      string   `xml:"desc"`
	PCode     string   `xml:"pcode"`
	WmInvId   int      `xml:"wminvid"`
	OnlyAuth  int      `xml:"onlyauth"`
}

type TransferMoneyResponse struct {
	XMLName       xml.Name `xml:"operation"`
	Id            string   `xml:"id,attr"`
	Ts            string   `xml:"ts,attr"`
	TxnId         int64    `xml:"tranid"`
	PurseSrc      string   `xml:"pursesrc"`
	PurseDest     string   `xml:"pursedest"`
	Amount        string   `xml:"amount"`
	Commission    string   `xml:"comiss"`
	OperationType string   `xml:"opertype"`
	Period        int      `xml:"period"`
	WmInvId       int      `xml:"wminvid"`
	Desc          string   `xml:"desc"`
	DateCrt       string   `xml:"datecrt"`
	DateUpd       string   `xml:"dateupd"`
	CorrWm        string   `xml:"corrwm"`
	Rest          string   `xml:"rest"`
	TimeLock      bool     `xml:"timelock"`
}

type GetTransactionsHistoryRequest struct {
	XMLName    xml.Name `xml:"getoperations"`
	Purse      string   `xml:"purse"`
	WmTranId   string   `xml:"wmtranid"`
	TxnId      int64    `xml:"tranid"`
	WmInvId    string   `xml:"wminvid"`
	OrderId    string   `xml:"orderid"`
	DateStart  string   `xml:"datestart"`
	DateFinish string   `xml:"datefinish"`
}

type GetTransactionsHistoryResponse struct {
	XMLName       xml.Name                 `xml:"operations"`
	Count         int64                    `xml:"cnt,attr"`
	OperationList []*TransferMoneyResponse `xml:"operation"`
}

type GetBalanceRequest struct {
	XMLName xml.Name `xml:"getpurses"`
	Wmid    string   `xml:"wmid"`
}

type GetBalanceResponse struct {
	XMLName   xml.Name                   `xml:"purses"`
	Count     string                     `xml:"cnt,attr"`
	PurseList []*GetBalanceResponsePurse `xml:"purse"`
}

type GetBalanceResponsePurse struct {
	XMLName          xml.Name `xml:"purse"`
	PurseName        string   `xml:"pursename"`
	Amount           float32  `xml:"amount"`
	Desc             string   `xml:"desc"`
	OutsideOpen      string   `xml:"outsideopen"`
	LastIncomeTxmId  string   `xml:"lastintr"`
	LastOutcomeTxnId string   `xml:"lastouttr"`
}

type httpTransport struct {
	transport      *http.Transport
	logger         *zap.Logger
	clearRequestFn func(req *http.Request) *http.Request
	caCertPool     *x509.CertPool
}

type httpContextKey struct {
	name string
}

func NewWebMoney(opts ...Option) (XMLInterface, error) {
	options, err := executeOptions(opts...)

	if err != nil {
		return nil, err
	}

	signerOpts := []signer.Option{
		signer.WmId(options.wmId),
		signer.Key(options.key),
		signer.Password(options.password),
	}
	sig, err := signer.NewSigner(signerOpts...)

	if err != nil {
		return nil, err
	}

	webmoney := &WebMoney{
		options:     options,
		signer:      sig,
		marshalFn:   xml.Marshal,
		unMarshalFn: xml.Unmarshal,
		httpClient:  options.httpClient,
	}

	if options.rootCaReader == nil {
		options.rootCaReader = strings.NewReader(rootCa)
	}

	if webmoney.httpClient == nil {
		caCert, err := ioutil.ReadAll(options.rootCaReader)

		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		webmoney.httpClient = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &httpTransport{
				logger:         options.logger,
				clearRequestFn: options.logClearFn,
				caCertPool:     caCertPool,
			},
		}
	}

	return webmoney, nil
}

func executeOptions(opts ...Option) (*Options, error) {
	options := &Options{}

	for _, opt := range opts {
		opt(options)
	}

	if options.wmId == "" {
		return nil, signer.ErrorWmIdNotConfigured
	}

	if !signer.WmIdRegex.MatchString(options.wmId) {
		return nil, signer.ErrorWmIdIsIncorrect
	}

	if options.key == "" {
		return nil, signer.ErrorKeyNotConfigured
	}

	if options.password == "" {
		return nil, signer.ErrorPasswordNotConfigured
	}

	return options, nil
}

func (m *WebMoney) TransferMoney(in *TransferMoneyRequest) (*TransferMoneyResponse, error) {
	if in.Desc != "" {
		in.Desc = m.Utf8ToWin(in.Desc)
	}

	req := &BaseRequest{
		RequestNumber: m.getRequestNumber(),
		WmId:          m.options.wmId,
		Request:       in,
	}
	req.SignatureString = req.RequestNumber + strconv.Itoa(in.TxnId) + in.PurseSrc + in.PurseDest + in.Amount +
		strconv.Itoa(in.Period) + in.PCode + in.Desc + strconv.Itoa(in.WmInvId)

	url := fmt.Sprintf(apiUrlMask, operationTransferMoney)
	result, err := m.sendRequest(url, req, new(TransferMoneyResponse))

	if err != nil {
		return nil, err
	}

	return result.Response.(*TransferMoneyResponse), nil
}

func (m *WebMoney) GetTransactionsHistory(in *GetTransactionsHistoryRequest) (*GetTransactionsHistoryResponse, error) {
	req := &BaseRequest{
		RequestNumber: m.getRequestNumber(),
		WmId:          m.options.wmId,
		Request:       in,
	}
	req.SignatureString = in.Purse + req.RequestNumber

	url := fmt.Sprintf(apiUrlMask, operationGetTransactionsHistory)
	result, err := m.sendRequest(url, req, new(GetTransactionsHistoryResponse))

	if err != nil {
		return nil, err
	}

	return result.Response.(*GetTransactionsHistoryResponse), nil
}

func (m *WebMoney) GetBalance(in *GetBalanceRequest) (*GetBalanceResponse, error) {
	req := &BaseRequest{
		RequestNumber: m.getRequestNumber(),
		WmId:          m.options.wmId,
		Request:       in,
	}
	req.SignatureString = in.Wmid + req.RequestNumber

	url := fmt.Sprintf(apiUrlMask, operationGetBalance)
	result, err := m.sendRequest(url, req, new(GetBalanceResponse))

	if err != nil {
		return nil, err
	}

	return result.Response.(*GetBalanceResponse), nil
}

func (m *WebMoney) getRequestNumber() string {
	nanoseconds := fmt.Sprintf("%03.f", float64(time.Now().Nanosecond()/1000000))
	return time.Now().Local().Format("20060102150405") + nanoseconds
}

func (m *WebMoney) sendRequest(url string, payload *BaseRequest, receiver interface{}) (*BaseResponse, error) {
	var err error
	payload.Signature, err = m.signer.Sign(payload.SignatureString)

	if err != nil {
		return nil, err
	}

	b, err := m.marshalFn(payload)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))

	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "text/xml")
	rsp, err := m.httpClient.Do(req)

	if err != nil {
		return nil, err
	}

	rspBody, err := ioutil.ReadAll(rsp.Body)

	if err != nil {
		return nil, err
	}

	_ = rsp.Body.Close()

	out := &BaseResponse{
		Response: receiver,
	}
	err = m.unMarshalFn(rspBody, out)

	if err != nil {
		return nil, err
	}

	if out.Code != 0 {
		return nil, errors.New(out.Reason)
	}

	return out, nil
}

func (m *WebMoney) Utf8ToWin(str string) string {
	enc := charmap.Windows1251.NewEncoder()
	out, _ := enc.String(str)
	return out
}

func (m *httpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := context.WithValue(req.Context(), &httpContextKey{name: "webmoneyRequestStart"}, time.Now())
	req = req.WithContext(ctx)

	var reqBody []byte

	if req.Body != nil {
		reqBody, _ = ioutil.ReadAll(req.Body)
	}

	req.Body = ioutil.NopCloser(bytes.NewBuffer(reqBody))

	var (
		rsp *http.Response
		err error
	)

	if m.transport == nil {
		m.transport = http.DefaultTransport.(*http.Transport)
		m.transport.TLSClientConfig = &tls.Config{
			RootCAs: m.caCertPool,
		}
		m.transport.DisableCompression = true
	}

	rsp, err = m.transport.RoundTrip(req)

	if err != nil {
		return rsp, err
	}

	var rspBody []byte

	if rsp.Body != nil {
		rspBody, err = ioutil.ReadAll(rsp.Body)
		if err != nil {
			return nil, err
		}
	}

	rsp.Body = ioutil.NopCloser(bytes.NewBuffer(rspBody))

	if m.clearRequestFn != nil {
		req = m.clearRequestFn(req)
	}

	m.logger.Info(
		req.URL.String(),
		zap.String("request_method", req.Method),
		zap.Any("request_headers", req.Header),
		zap.ByteString("request_body", reqBody),
		zap.Int("response_status", rsp.StatusCode),
		zap.Any("response_headers", rsp.Header),
		zap.ByteString("response_body", rspBody),
	)

	return rsp, err
}

// Root ca for webmoney requests
const rootCa = `
-----BEGIN CERTIFICATE-----
MIIFsTCCA5mgAwIBAgIQA7dHzSZ7uJdBxFycIWn+WjANBgkqhkiG9w0BAQUFADBr
MSswKQYDVQQLEyJXTSBUcmFuc2ZlciBDZXJ0aWZpY2F0aW9uIFNlcnZpY2VzMRgw
FgYDVQQKEw9XTSBUcmFuc2ZlciBMdGQxIjAgBgNVBAMTGVdlYk1vbmV5IFRyYW5z
ZmVyIFJvb3QgQ0EwHhcNMTAwMzEwMTczNDU2WhcNMzUwMzEwMTc0NDUxWjBrMSsw
KQYDVQQLEyJXTSBUcmFuc2ZlciBDZXJ0aWZpY2F0aW9uIFNlcnZpY2VzMRgwFgYD
VQQKEw9XTSBUcmFuc2ZlciBMdGQxIjAgBgNVBAMTGVdlYk1vbmV5IFRyYW5zZmVy
IFJvb3QgQ0EwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQDFLJXtzEkZ
xLj1HIj9EhGvajFJ7RCHzF+MK2ZrAgxmmOafiFP6QD/aVjIexBqRb8SVy38xH+wt
hqkZgLMOGn8uDNpFieEMoX3ZRfqtCcD76KDySTOX1QUwHAzBfGuhe2ZQULUIjxdP
Ra4NDyvmXh4pE/s1+/7dGbUZs/JpYYaD2xxAt5PDTjylsKOk4FMb5kv6jzORkXku
5UKFGUXEXbbf1xzgYHMIzoeJGn+iPgVFYAvkkQyvxEaVj0lNE+q/ua761krgCo47
BiH1zMFzkv4uNHEZfe/lyHaozzbsu6yaK3EdrURSLuWrlxKy9yo3xDe9TPkzkhPe
JPbV7YgvUUtWSeAJpksBU8GCALEhSgXOfHckuJFj9QB3YecHBvjdSiAUuntwM/iH
vtSOXEUHxqW75E2Gq/2L4vBcxArXVdbUrVQDF3klzYu17OFgfe1hHHMHzgr4HBML
ZiRCcvNLqghBCVxu1DM15YDfw+wnNV/5dUPx60tiocmCZpJKTwVl8gc85QCPyREu
jey8F0kgdgssQosPWTTWDg7X4Ifw20VkplHZDr29K5HdwLe56TvOI/4H24XJdqpA
xoLBx9PL6ZXxH52wU0bSluL8/joXGzavFrhsXH7jJocH6tsFVzBZrmnVswbUMHDN
L3xSnr5fAAXXZa7UwHd3pq/fsdG7s9PByQIDAQABo1EwTzALBgNVHQ8EBAMCAYYw
DwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUsTCnSwOZT4Q2HBN9V/TrafuIG8Mw
EAYJKwYBBAGCNxUBBAMCAQAwDQYJKoZIhvcNAQEFBQADggIBAAy5jHDFpVWtF209
N30I+LHsiqMaLUmYDeV6sUBJqmWAZav7pWnigiMLuJd9yRa/ow6yKlKPRi3sbKaB
wsAQ+xnz811nLFBBdS4PkwlHu1B7P4B2YbcqmF6k1QieJBZxOn7wledtnoBAkZ4d
6HEW1OM5cvCoyj8YAdJTZIBzn61aNt/viPvypIUQf6Ps6Q2daNEAj7DoxIY8crnO
aSIGdGmlT/y/edSqWv9Am5e9KXkJhQWMnGXh43wJYyHTetxVWPS43bW7gIUADYyc
KSH3isrBN5xQOFXMfL+lVHHSs7ap23DOo7xIDenm5PWz+QdDDFz3zLVeRovnkIdk
a/Wgk3f6rFfKB0y5POJ+BJvkorIYNZiN3dnmc6cDP840BUMv3BUrOe8iSy5lRr8m
R+daktbZfO8E/rAb3zEdN+KG/CNJfAnQvp6DT4LqY/J9pG+VusH5GpUwuXr7UqLw
End1LRp7qm28Cic7fegUnnUpkuF4ZFq8pWq8w59sOWlRuKBuWX46OghMrjgD0AN1
hlA2/d5ULImX70Q2te3xiS1vrQhu77mkb/jA4/9+YfeT7VMpbnC3OoHiZ2bjudKn
thlOs+AuUvzB4Tqo62VSF5+r0sYI593S+STmaZBAzsoaoEB7qxqKbEKCvXb9BlXk
L76xIOEkbSIdPIkGXM4aMo4mTVz7
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIENjCCAx6gAwIBAgIBATANBgkqhkiG9w0BAQUFADBvMQswCQYDVQQGEwJTRTEU
MBIGA1UEChMLQWRkVHJ1c3QgQUIxJjAkBgNVBAsTHUFkZFRydXN0IEV4dGVybmFs
IFRUUCBOZXR3b3JrMSIwIAYDVQQDExlBZGRUcnVzdCBFeHRlcm5hbCBDQSBSb290
MB4XDTAwMDUzMDEwNDgzOFoXDTIwMDUzMDEwNDgzOFowbzELMAkGA1UEBhMCU0Ux
FDASBgNVBAoTC0FkZFRydXN0IEFCMSYwJAYDVQQLEx1BZGRUcnVzdCBFeHRlcm5h
bCBUVFAgTmV0d29yazEiMCAGA1UEAxMZQWRkVHJ1c3QgRXh0ZXJuYWwgQ0EgUm9v
dDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALf3GjPm8gAELTngTlvt
H7xsD821+iO2zt6bETOXpClMfZOfvUq8k+0DGuOPz+VtUFrWlymUWoCwSXrbLpX9
uMq/NzgtHj6RQa1wVsfwTz/oMp50ysiQVOnGXw94nZpAPA6sYapeFI+eh6FqUNzX
mk6vBbOmcZSccbNQYArHE504B4YCqOmoaSYYkKtMsE8jqzpPhNjfzp/haW+710LX
a0Tkx63ubUFfclpxCDezeWWkWaCUN/cALw3CknLa0Dhy2xSoRcRdKn23tNbE7qzN
E0S3ySvdQwAl+mG5aWpYIxG3pzOPVnVZ9c0p10a3CitlttNCbxWyuHv77+ldU9U0
WicCAwEAAaOB3DCB2TAdBgNVHQ4EFgQUrb2YejS0Jvf6xCZU7wO94CTLVBowCwYD
VR0PBAQDAgEGMA8GA1UdEwEB/wQFMAMBAf8wgZkGA1UdIwSBkTCBjoAUrb2YejS0
Jvf6xCZU7wO94CTLVBqhc6RxMG8xCzAJBgNVBAYTAlNFMRQwEgYDVQQKEwtBZGRU
cnVzdCBBQjEmMCQGA1UECxMdQWRkVHJ1c3QgRXh0ZXJuYWwgVFRQIE5ldHdvcmsx
IjAgBgNVBAMTGUFkZFRydXN0IEV4dGVybmFsIENBIFJvb3SCAQEwDQYJKoZIhvcN
AQEFBQADggEBALCb4IUlwtYj4g+WBpKdQZic2YR5gdkeWxQHIzZlj7DYd7usQWxH
YINRsPkyPef89iYTx4AWpb9a/IfPeHmJIZriTAcKhjW88t5RxNKWt9x+Tu5w/Rw5
6wwCURQtjr0W4MHfRnXnJK3s9EK0hZNwEGe6nQY1ShjTK3rMUUKhemPR5ruhxSvC
Nr4TDea9Y355e6cJDUCrat2PisP29owaQgVR1EX1n6diIWgVIEM8med8vSTYqZEX
c4g/VhsxOBi0cQ+azcgOno4uG+GMmIPLHzHxREzGBHNJdmAPx/i9F4BrLunMTA5a
mnkPIAou1Z5jJh5VkpTYghdae9C8x49OhgQ=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIICPDCCAaUCEHC65B0Q2Sk0tjjKewPMur8wDQYJKoZIhvcNAQECBQAwXzELMAkG
A1UEBhMCVVMxFzAVBgNVBAoTDlZlcmlTaWduLCBJbmMuMTcwNQYDVQQLEy5DbGFz
cyAzIFB1YmxpYyBQcmltYXJ5IENlcnRpZmljYXRpb24gQXV0aG9yaXR5MB4XDTk2
MDEyOTAwMDAwMFoXDTI4MDgwMTIzNTk1OVowXzELMAkGA1UEBhMCVVMxFzAVBgNV
BAoTDlZlcmlTaWduLCBJbmMuMTcwNQYDVQQLEy5DbGFzcyAzIFB1YmxpYyBQcmlt
YXJ5IENlcnRpZmljYXRpb24gQXV0aG9yaXR5MIGfMA0GCSqGSIb3DQEBAQUAA4GN
ADCBiQKBgQDJXFme8huKARS0EN8EQNvjV69qRUCPhAwL0TPZ2RHP7gJYHyX3KqhE
BarsAx94f56TuZoAqiN91qyFomNFx3InzPRMxnVx0jnvT0Lwdd8KkMaOIG+YD/is
I19wKTakyYbnsZogy1Olhec9vn2a/iRFM9x2Fe0PonFkTGUugWhFpwIDAQABMA0G
CSqGSIb3DQEBAgUAA4GBALtMEivPLCYATxQT3ab7/AoRhIzzKBxnki98tsX63/Do
lbwdj2wsqFHMc9ikwFPwTtYmwHYBV4GSXiHx0bH/59AhWM1pF+NEHJwZRDmJXNyc
AA9WjQKZ7aKQRUzkuxCkPfAyAw7xzvjoyVGM5mKf5p/AfbdynMk2OmufTqj/ZA1k
-----END CERTIFICATE-----`
