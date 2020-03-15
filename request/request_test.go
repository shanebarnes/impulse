package request

import (
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequest_AddTransaction_EmptyRequestAndResponse(t *testing.T) {
	req, _ := New(nil, "https://127.0.0.1:443", true)
	assert.NotNil(t, req.AddTransaction(false, "", ""))
	assert.Zero(t, len(req.txnPreTls) + len(req.txnPostTls))
}

func TestRequest_AddTransaction_AfterTls(t *testing.T) {
	req, _ := New(nil, "http://127.0.0.1:80", false)
	assert.NotNil(t, req.AddTransaction(false, "CONNECT host:port HTTP/1.1\r\n\r\n", "HTTP/1.1 200 OK\r\n\r\n"))
	assert.Zero(t, len(req.txnPreTls) + len(req.txnPostTls))
}

func TestRequest_AddTransaction_RequestAndResponse(t *testing.T) {
	req, _ := New(nil, "https://127.0.0.1:443", true)
	assert.Nil(t, req.AddTransaction(true, "CONNECT host:port HTTP/1.1\r\n\r\n", "HTTP/1.1 200 OK\r\n\r\n"))
	assert.Zero(t, len(req.txnPostTls))
	assert.Equal(t, 1, len(req.txnPreTls))
}

func TestRequest_AddTransaction_RequestOnly(t *testing.T) {
	req, _ := New(nil, "tcp://127.0.0.1:12345", true)
	assert.Nil(t, req.AddTransaction(true, "send", ""))
	assert.Zero(t, len(req.txnPostTls))
	assert.Equal(t, 1, len(req.txnPreTls))
}

func TestRequest_AddTransaction_ResponseOnly(t *testing.T) {
	req, _ := New(nil, "tcp://127.0.0.1:12345", true)
	assert.Nil(t, req.AddTransaction(true, "", "recv"))
	assert.Zero(t, len(req.txnPostTls))
	assert.Equal(t, 1, len(req.txnPreTls))
}

func TestRequest_AddTransaction_X2(t *testing.T) {
	req, _ := New(nil, "tcp://127.0.0.1:443", true)
	assert.Nil(t, req.AddTransaction(true, "CONNECT host:port HTTP/1.1\r\n\r\n", "HTTP/1.1 200 OK\r\n\r\n"))
	assert.Equal(t, 1, len(req.txnPreTls))
	assert.Nil(t, req.AddTransaction(false, "GET /index.html HTTP/1.1\r\n\r\n", "HTTP/1.1 200 OK\r\n\r\n"))
	assert.Equal(t, 1, len(req.txnPostTls))
}

func TestRequest_NewRequest_UrlEmpty(t *testing.T) {
	_, err := New(nil, "", false)
	assert.NotNil(t, err)
}

func TestRequest_NewRequest_UrlInvalid(t *testing.T) {
	_, err := New(nil, " https://host:443", false)
	assert.NotNil(t, err)
}

func TestRequest_NewRequest_UrlMissingHost(t *testing.T) {
	_, err := New(nil, "tcp://:443", true)
	assert.NotNil(t, err)
}

func TestRequest_NewRequest_UrlMissingPort(t *testing.T) {
	_, err := New(nil, "tcp://host", true)
	assert.NotNil(t, err)
}

func TestRequest_NewRequest_UrlMissingProt(t *testing.T) {
	_, err := New(nil, "//host:443", true)
	assert.NotNil(t, err)
}

func TestRequest_NewRequest_UrlUnsupportedProt(t *testing.T) {
	_, err := New(nil, "smb://host:139", true)
	assert.NotNil(t, err)
}

func TestRequest_NewRequest_UrlTcp(t *testing.T) {
	logger := log.New(os.Stdout, "", 0)
	req, err := New(logger, "tcp://host:12345", true)
	assert.Nil(t, err)
	assert.True(t, reflect.DeepEqual(logger, req.logger))
	assert.Equal(t, "tcp", req.prot)
	assert.Equal(t, "host", req.host)
	assert.Equal(t, "12345", req.port)
	assert.True(t, req.useTls)
}
