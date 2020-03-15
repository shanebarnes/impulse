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

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s usage:\n\n", appName)
		flag.PrintDefaults()
	}
}

func main() {
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
			impulse.AddTransaction(false, "GET /index.html HTTP/1.1\r\n\r\n", "HTTP/1.1 200 OK\r\n\r\n")
			err = impulse.Send(nil)
		}

		if err != nil {
			logger.Printf("%s\n", err)
			os.Exit(1)
		}
	}
}
