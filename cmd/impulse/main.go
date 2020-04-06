package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/shanebarnes/impulse/internal/version"
	"github.com/shanebarnes/impulse/request"
)

const appName = "impulse"

type stringSlice []string

func (s *stringSlice) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s usage:\n\n", appName)
		flag.PrintDefaults()
	}
}

func loadTransactions(impulse *request.Request, beforeTls bool, txns stringSlice) {
	for _, txn := range txns {
		err := impulse.ParseAndAddTransaction(beforeTls, txn)
		if err != nil {
			fmt.Fprint(os.Stderr, "Impulse transaction cannot be loaded: %s\n", err.Error())
			os.Exit(1)
		}
	}
}

func main() {
	exitCode := 0

	postTxn := stringSlice{}
	preTxn := stringSlice{}
	flag.Var(&postTxn, "post-tls", `Transactions to perform after TLS handshake (e.g., "{ \"request\": \"GET /index.html HTTP/1.1\r\n\r\n\", \"response\": \"HTTP/1.1 200 OK\r\n\" }")`)
	flag.Var(&preTxn, "pre-tls", `Transactions to perform before TLS handshake (e.g., "{ \"request\": \"CONNECT 127.0.0.1:443 HTTP/1.1\r\n\r\n\", \"response\": \"HTTP/1.1 200 OK\r\n\" }")`)
	tls := flag.Bool("tls", false, "Enable TLS")
	url := flag.String("url", "", "URL of server to receive impulse (e.g., tcp://127.0.0.1:443)")
	printVersion := flag.Bool("version", false, "print version information")
	flag.Parse()

	if *printVersion {
		fmt.Fprintf(os.Stdout, "%s version %s\n", appName, version.String())
	} else if *url == "" {
		flag.Usage()
	} else {
		logger := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds)
		impulse, err := request.New(logger, *url, *tls)
		if err == nil {
			loadTransactions(&impulse, true, preTxn)
			loadTransactions(&impulse, false, postTxn)
			err = impulse.Send(nil)
		}

		if err == nil {
			logger.Printf("Impulse response succeeded")
		} else {
			logger.Printf("Impulse response failed: %s\n", err)
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}
