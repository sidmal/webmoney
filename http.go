package webmoney

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"time"
)

type httpTransport struct {
	transport      http.RoundTripper
	logger         *zap.Logger
	clearRequestFn func(req *http.Request) *http.Request
	getTransportFn func() http.RoundTripper
}

type httpContextKey struct {
	name string
}

func newHttpTransport(
	logger *zap.Logger,
	clearRequestFn func(req *http.Request) *http.Request,
	caCertPool *x509.CertPool,
) *httpTransport {
	return &httpTransport{
		logger:         logger,
		clearRequestFn: clearRequestFn,
		transport:      getTransport(caCertPool),
	}
}

func getTransport(_ *x509.CertPool) http.RoundTripper {
	transport := http.DefaultTransport
	transport.(*http.Transport).TLSClientConfig = &tls.Config{
		Renegotiation:      tls.RenegotiateOnceAsClient,
		InsecureSkipVerify: true,
	}
	transport.(*http.Transport).DisableCompression = true
	return transport
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

	rsp, err = m.transport.RoundTrip(req)

	if err != nil || m.logger == nil {
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
MIIGEzCCA/ugAwIBAgIQfVtRJrR2uhHbdBYLvFMNpzANBgkqhkiG9w0BAQwFADCB
iDELMAkGA1UEBhMCVVMxEzARBgNVBAgTCk5ldyBKZXJzZXkxFDASBgNVBAcTC0pl
cnNleSBDaXR5MR4wHAYDVQQKExVUaGUgVVNFUlRSVVNUIE5ldHdvcmsxLjAsBgNV
BAMTJVVTRVJUcnVzdCBSU0EgQ2VydGlmaWNhdGlvbiBBdXRob3JpdHkwHhcNMTgx
MTAyMDAwMDAwWhcNMzAxMjMxMjM1OTU5WjCBjzELMAkGA1UEBhMCR0IxGzAZBgNV
BAgTEkdyZWF0ZXIgTWFuY2hlc3RlcjEQMA4GA1UEBxMHU2FsZm9yZDEYMBYGA1UE
ChMPU2VjdGlnbyBMaW1pdGVkMTcwNQYDVQQDEy5TZWN0aWdvIFJTQSBEb21haW4g
VmFsaWRhdGlvbiBTZWN1cmUgU2VydmVyIENBMIIBIjANBgkqhkiG9w0BAQEFAAOC
AQ8AMIIBCgKCAQEA1nMz1tc8INAA0hdFuNY+B6I/x0HuMjDJsGz99J/LEpgPLT+N
TQEMgg8Xf2Iu6bhIefsWg06t1zIlk7cHv7lQP6lMw0Aq6Tn/2YHKHxYyQdqAJrkj
eocgHuP/IJo8lURvh3UGkEC0MpMWCRAIIz7S3YcPb11RFGoKacVPAXJpz9OTTG0E
oKMbgn6xmrntxZ7FN3ifmgg0+1YuWMQJDgZkW7w33PGfKGioVrCSo1yfu4iYCBsk
Haswha6vsC6eep3BwEIc4gLw6uBK0u+QDrTBQBbwb4VCSmT3pDCg/r8uoydajotY
uK3DGReEY+1vVv2Dy2A0xHS+5p3b4eTlygxfFQIDAQABo4IBbjCCAWowHwYDVR0j
BBgwFoAUU3m/WqorSs9UgOHYm8Cd8rIDZsswHQYDVR0OBBYEFI2MXsRUrYrhd+mb
+ZsF4bgBjWHhMA4GA1UdDwEB/wQEAwIBhjASBgNVHRMBAf8ECDAGAQH/AgEAMB0G
A1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAbBgNVHSAEFDASMAYGBFUdIAAw
CAYGZ4EMAQIBMFAGA1UdHwRJMEcwRaBDoEGGP2h0dHA6Ly9jcmwudXNlcnRydXN0
LmNvbS9VU0VSVHJ1c3RSU0FDZXJ0aWZpY2F0aW9uQXV0aG9yaXR5LmNybDB2Bggr
BgEFBQcBAQRqMGgwPwYIKwYBBQUHMAKGM2h0dHA6Ly9jcnQudXNlcnRydXN0LmNv
bS9VU0VSVHJ1c3RSU0FBZGRUcnVzdENBLmNydDAlBggrBgEFBQcwAYYZaHR0cDov
L29jc3AudXNlcnRydXN0LmNvbTANBgkqhkiG9w0BAQwFAAOCAgEAMr9hvQ5Iw0/H
ukdN+Jx4GQHcEx2Ab/zDcLRSmjEzmldS+zGea6TvVKqJjUAXaPgREHzSyrHxVYbH
7rM2kYb2OVG/Rr8PoLq0935JxCo2F57kaDl6r5ROVm+yezu/Coa9zcV3HAO4OLGi
H19+24rcRki2aArPsrW04jTkZ6k4Zgle0rj8nSg6F0AnwnJOKf0hPHzPE/uWLMUx
RP0T7dWbqWlod3zu4f+k+TY4CFM5ooQ0nBnzvg6s1SQ36yOoeNDT5++SR2RiOSLv
xvcRviKFxmZEJCaOEDKNyJOuB56DPi/Z+fVGjmO+wea03KbNIaiGCpXZLoUmGv38
sbZXQm2V0TP2ORQGgkE49Y9Y3IBbpNV9lXj9p5v//cWoaasm56ekBYdbqbe4oyAL
l6lFhd2zi+WJN44pDfwGF/Y4QA5C5BIG+3vzxhFoYt/jmPQT2BVPi7Fp2RBgvGQq
6jG35LWjOhSbJuMLe/0CjraZwTiXWTb2qHSihrZe68Zk6s+go/lunrotEbaGmAhY
LcmsJWTyXnW0OMGuf1pGg+pRyrbxmRE1a6Vqe8YAsOf4vmSyrcjC8azjUeqkk+B5
yOGBQMkKW+ESPMFgKuOXwIlCypTPRpgSabuY0MLTDXJLR27lk8QyKGOHQ+SwMj4K
00u/I5sUKUErmgQfky3xxzlIPK1aEn8=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIF3jCCA8agAwIBAgIQAf1tMPyjylGoG7xkDjUDLTANBgkqhkiG9w0BAQwFADCB
iDELMAkGA1UEBhMCVVMxEzARBgNVBAgTCk5ldyBKZXJzZXkxFDASBgNVBAcTC0pl
cnNleSBDaXR5MR4wHAYDVQQKExVUaGUgVVNFUlRSVVNUIE5ldHdvcmsxLjAsBgNV
BAMTJVVTRVJUcnVzdCBSU0EgQ2VydGlmaWNhdGlvbiBBdXRob3JpdHkwHhcNMTAw
MjAxMDAwMDAwWhcNMzgwMTE4MjM1OTU5WjCBiDELMAkGA1UEBhMCVVMxEzARBgNV
BAgTCk5ldyBKZXJzZXkxFDASBgNVBAcTC0plcnNleSBDaXR5MR4wHAYDVQQKExVU
aGUgVVNFUlRSVVNUIE5ldHdvcmsxLjAsBgNVBAMTJVVTRVJUcnVzdCBSU0EgQ2Vy
dGlmaWNhdGlvbiBBdXRob3JpdHkwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIK
AoICAQCAEmUXNg7D2wiz0KxXDXbtzSfTTK1Qg2HiqiBNCS1kCdzOiZ/MPans9s/B
3PHTsdZ7NygRK0faOca8Ohm0X6a9fZ2jY0K2dvKpOyuR+OJv0OwWIJAJPuLodMkY
tJHUYmTbf6MG8YgYapAiPLz+E/CHFHv25B+O1ORRxhFnRghRy4YUVD+8M/5+bJz/
Fp0YvVGONaanZshyZ9shZrHUm3gDwFA66Mzw3LyeTP6vBZY1H1dat//O+T23LLb2
VN3I5xI6Ta5MirdcmrS3ID3KfyI0rn47aGYBROcBTkZTmzNg95S+UzeQc0PzMsNT
79uq/nROacdrjGCT3sTHDN/hMq7MkztReJVni+49Vv4M0GkPGw/zJSZrM233bkf6
c0Plfg6lZrEpfDKEY1WJxA3Bk1QwGROs0303p+tdOmw1XNtB1xLaqUkL39iAigmT
Yo61Zs8liM2EuLE/pDkP2QKe6xJMlXzzawWpXhaDzLhn4ugTncxbgtNMs+1b/97l
c6wjOy0AvzVVdAlJ2ElYGn+SNuZRkg7zJn0cTRe8yexDJtC/QV9AqURE9JnnV4ee
UB9XVKg+/XRjL7FQZQnmWEIuQxpMtPAlR1n6BB6T1CZGSlCBst6+eLf8ZxXhyVeE
Hg9j1uliutZfVS7qXMYoCAQlObgOK6nyTJccBz8NUvXt7y+CDwIDAQABo0IwQDAd
BgNVHQ4EFgQUU3m/WqorSs9UgOHYm8Cd8rIDZsswDgYDVR0PAQH/BAQDAgEGMA8G
A1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQEMBQADggIBAFzUfA3P9wF9QZllDHPF
Up/L+M+ZBn8b2kMVn54CVVeWFPFSPCeHlCjtHzoBN6J2/FNQwISbxmtOuowhT6KO
VWKR82kV2LyI48SqC/3vqOlLVSoGIG1VeCkZ7l8wXEskEVX/JJpuXior7gtNn3/3
ATiUFJVDBwn7YKnuHKsSjKCaXqeYalltiz8I+8jRRa8YFWSQEg9zKC7F4iRO/Fjs
8PRF/iKz6y+O0tlFYQXBl2+odnKPi4w2r78NBc5xjeambx9spnFixdjQg3IM8WcR
iQycE0xyNN+81XHfqnHd4blsjDwSXWXavVcStkNr/+XeTWYRUc+ZruwXtuhxkYze
Sf7dNXGiFSeUHM9h4ya7b6NnJSFd5t0dCy5oGzuCr+yDZ4XUmFF0sbmZgIn/f3gZ
XHlKYC6SQK5MNyosycdiyA5d9zZbyuAlJQG03RoHnHcAP9Dc1ew91Pq7P8yF1m9/
qS3fuQL39ZeatTXaw2ewh0qpKJ4jjv9cJ2vhsE/zB+4ALtRZh8tSQZXq9EfX7mRB
VXyNWQKV3WKdwrnuWih0hKWbt5DHDAff9Yk2dDLWKMGwsAvgnEzDHNb842m1R0aB
L6KCq9NjRHDEjf8tM7qtj3u1cIiuPhnPQCjY/MiQu12ZIvVS5ljFH4gxQ+6IHdfG
jjxDah2nGN59PRbxYvnKkKj9
-----END CERTIFICATE-----
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
-----END CERTIFICATE-----`
