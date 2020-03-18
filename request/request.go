package request

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"strings"

	"github.com/grantae/certinfo"
)

const (
	defaultBufSize = 1*1024*1024
)

type Request struct {
	conNet     net.Conn
	conTls     *tls.Conn
	host       string
	logger     *log.Logger
	port       string
	prot       string
	txnPreTls  []Transaction // Transactions before TLS handshake
	txnPostTls []Transaction // Transactions after TLS handshake
	url        string
	useTls     bool
}

type Transaction struct {
	Request   string
	Response  string
}

var protocols = map[string]int{
	//"http": 1,
	//"https": 1,
	"tcp": 1,
	"udp": 1,
}

type reqOp func(*Request) error

func (r *Request) ParseAndAddTransaction(beforeTls bool, jsonStr string) error {
	txn := Transaction{}
	err := json.Unmarshal([]byte(jsonStr), &txn)
	if err == nil {
		err = r.AddTransaction(beforeTls, txn.Request, txn.Response)
	} else {
		err = fmt.Errorf("Impulse transaction failed JSON parser: %s", err.Error())
	}
	return err
}

func (r *Request) AddTransaction(beforeTls bool, request, response string) error {
	var err error
	if request == "" && response == "" { // Transaction request or response can be empty but not both
		err = fmt.Errorf("Impulse transaction is empty")
	} else if beforeTls == false && r.useTls == false { // Cannot make transaction after TLS if TLS is disabled
		err = fmt.Errorf("Impulse transaction cannot be added to request: TLS is disabled")
	} else if beforeTls {
		r.txnPreTls = append(r.txnPreTls, Transaction{Request: request, Response: response})
	} else {
		r.txnPostTls = append(r.txnPostTls, Transaction{Request: request, Response: response})
	}
	return err
}

func New(logger *log.Logger, urlStr string, useTls bool) (Request, error) {
	var req Request
	u, err := url.Parse(urlStr)
	if err == nil {
		if urlStr == "" {
			err = fmt.Errorf("Impulse URL is empty")
		} else if u.Hostname() == "" {
			err = fmt.Errorf("Impulse URL, '%s', is missing a hostname", urlStr)
		} else if u.Port() == "" {
			err = fmt.Errorf("Impuse URL, '%s', is missing a port number", urlStr)
		} else if u.Scheme == "" {
			err = fmt.Errorf("Impulse URL, '%s', is missing a protocol", urlStr)
		} else if _, ok := protocols[u.Scheme]; !ok {
			err = fmt.Errorf("Impulse URL, '%s', contains an unsupported protocol", urlStr)
		} else {
			if logger == nil {
				req.logger = log.New(ioutil.Discard, "", 0)
			} else {
				req.logger = logger
			}
			req.host = u.Hostname()
			req.port = u.Port()
			req.prot = u.Scheme
			req.url = urlStr
			req.useTls = useTls
		}
	} else {
		err = fmt.Errorf("Impulse URL, '%s', cannot be parsed: %s", urlStr, err.Error())
	}
	return req, err
}

func (r *Request) opNetConnect() error {
	prefix := "Impulse Net Connect"
	r.logger.Printf("%s: connecting to %s\n", prefix, r.url)

	conn, err := net.Dial(r.prot, r.host+":"+r.port)
	if err == nil {
		r.logger.Printf("%s: connected to %s\n", prefix, r.url)
		r.conNet = conn
		//defer conn.Close()
		// Measure connect time? Get socket options?
	} else {
		r.logger.Printf("%s: failed to connect to %s: %s\n", prefix, r.url, err.Error())
	}

	return err
}

func (r *Request) opNetShutdown() error {
	prefix := "Impulse Shutdown"

	if r.conTls != nil {
		r.logger.Printf("%s: closing tls connection to %s\n", prefix, r.host)
		r.conTls.Close()
		r.conTls = nil
	}

	if r.conNet != nil {
		r.logger.Printf("%s: closing net connection to %s\n", prefix, r.url)
		r.conNet.Close()
		r.conNet = nil
	}
	return nil
}

func (r *Request) opPostTlsHandshake() error {
	var err error
	prefix := "Impulse Post-TLS Handshake"

	for i, txn := range r.txnPostTls {
		r.logger.Printf("%s: executing transaction %d\n", prefix, i+1)
		if r.useTls {
			err = r.executeTransaction(r.conTls, txn)
		} else {
			err = r.executeTransaction(r.conNet, txn)
		}
		if err != nil {
			break
		}
	}

	return err
}

func (r *Request) opPreTlsHandshake() error {
	var err error
	prefix := "Impulse Pre-TLS Handshake"

	for i, txn := range r.txnPreTls {
		r.logger.Printf("%s: executing transaction %d\n", prefix, i+1)
		err = r.executeTransaction(r.conNet, txn)
		if err != nil {
			break
		}
	}

	return err
}

func (r *Request) verifyPeerCertCb(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	for chainNum, chain := range verifiedChains {
		for certNum, cert := range chain {
			text, _ := certinfo.CertificateText(cert)
			r.logger.Printf("Chain %d, Certificate %d details:\n\n\n", chainNum+1, certNum+1)
			r.logger.Printf("%s\n\n\n", text)
		}
	}
	return nil
}

func (r *Request) opTlsHandshake() error {
	var err error
	prefix := "Impulse TLS Handshake"

	if r.useTls {
		r.logger.Printf("%s: handshaking with %s\n", prefix, r.host)
		var sysRoots *x509.CertPool
		sysRoots, err = x509.SystemCertPool()
		config := &tls.Config{
			InsecureSkipVerify: false,
			RootCAs: sysRoots,
			ServerName: r.host,
			VerifyPeerCertificate: r.verifyPeerCertCb,
		}
		r.conTls = tls.Client(r.conNet, config)

		r.logger.Printf("%s: found %d certificates in system certificate pool\n", prefix, len(sysRoots.Subjects()))
		rawSubjects := sysRoots.Subjects()
		for i, rawSubject := range rawSubjects {
			var subject pkix.RDNSequence
			if _, err = asn1.Unmarshal(rawSubject, &subject); err == nil {
				r.logger.Printf("%s: certificate %3d subject: %s\n", prefix, i+1, subject.String())
			} else {
				r.logger.Printf("%s: failed to retrieve certificate %3d subject: %s\n", prefix, i+1, err)
			}
		}

		err = r.conTls.Handshake()
		if err == nil {
			r.logger.Printf("%s: successfully completed handshake with %s\n", prefix, r.host)
		} else {
			r.logger.Printf("%s: failed to handshake with %s: %s\n", prefix, r.host, err.Error())
		}
	}

	return err
}

func (r *Request) Send(ctx context.Context) error { // Return response
	var err error
	ops := []reqOp{
		(*Request).opNetConnect,
		(*Request).opPreTlsHandshake,
		(*Request).opTlsHandshake,
		(*Request).opPostTlsHandshake,
		(*Request).opNetShutdown,
	}

	for _, op := range ops {
		//r.logger.Printf("Impulse Send: starting operation %d\n", i+1)
		err = op(r)
		//r.logger.Printf("Impulse Send: completed operation %d\n", i+1)
		if err != nil {
			r.opNetShutdown()
			break
		}
	}

	return err
}

func (r *Request) executeTransaction(conn net.Conn, txn Transaction) error {
	var err error
	var n int
	prefix := "Impulse Transaction"

	if err == nil && len(txn.Request) > 0 {
		r.logger.Printf("%s: sending %s\n", prefix, txn.Request)
		n, err = io.WriteString(conn, txn.Request)
		r.logger.Printf("%s: sent %d of %d bytes\n", prefix, n, len(txn.Request))

		if err == nil {
			if n != len(txn.Request) {
				err = fmt.Errorf("Impulse transaction failed: sent %d of %d bytes", n, len(txn.Request))
			}
		} else if err == io.EOF {
			if n != len(txn.Request) {
				err = fmt.Errorf("Impulse transaction failed: sent %d of %d bytes", n, len(txn.Request))
			} else if txn.Response != "" {
				err = fmt.Errorf("Impulse transaction failed: received 0 of %d bytes", len(txn.Response))
			} else {
				err = nil
			}
		} else {
			err = fmt.Errorf("Impulse transaction failed on request: %s", err.Error())
		}
	}

	if err == nil && len(txn.Response) > 0 {
		bufSize := defaultBufSize
		if len(txn.Response) > bufSize {
			bufSize = len(txn.Response)
		}
		buf := make([]byte, bufSize)
		r.logger.Printf("%s: receiving %d bytes\n", prefix, len(buf))
		n, err = conn.Read(buf)
		r.logger.Printf("%s: received %d of %d bytes\n", prefix, n, len(buf))

		response := ""
		if n > 0 {
			response = string(buf[:n])
			r.logger.Printf("%s: received %s\n", prefix, response)
		}
		if err == nil || err == io.EOF {
			if n < len(txn.Response) {
				err = fmt.Errorf("Impulse transaction failed: received %d of %d bytes", n, len(txn.Response))
			} else if !strings.HasPrefix(response, txn.Response) {
				err = fmt.Errorf("Impulse transaction failed: received response does not equal expected response")
			}
		} else {
			err = fmt.Errorf("Impulse transaction failed on response: %s", err.Error())
		}
	}

	return err
}
