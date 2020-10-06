package webmoney

import (
	"bytes"
	"crypto/x509"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/sidmal/webmoney/signer"
	"golang.org/x/net/html/charset"
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
	Server        string      `xml:"ser,omitempty"`
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
		options:   options,
		signer:    sig,
		marshalFn: xml.Marshal,
		unMarshalFn: func(data []byte, v interface{}) error {
			decoder := xml.NewDecoder(bytes.NewReader(data))
			decoder.CharsetReader = charset.NewReaderLabel
			err = decoder.Decode(v)
			return err
		},
		httpClient: options.httpClient,
	}

	if options.rootCaReader == nil {
		options.rootCaReader = strings.NewReader(rootCa)
	}

	if webmoney.httpClient == nil {
		caCert, err := ioutil.ReadAll(options.rootCaReader)

		if err != nil {
			return nil, err
		}

		rootCAs, _ := x509.SystemCertPool()

		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}

		rootCAs.AppendCertsFromPEM(caCert)

		webmoney.httpClient = &http.Client{
			Timeout:   10 * time.Second,
			Transport: newHttpTransport(options.logger, options.logClearFn, rootCAs),
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
