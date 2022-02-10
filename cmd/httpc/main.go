package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/mhilton/httpc"
)

var proc = httpc.Proc{
	Header: make(http.Header),
	Client: new(http.Client),
	Body:   os.Stdin,
	Err:    os.Stderr,
	Outputter: httpc.SimpleOutputter{
		Out: os.Stdout,
	},
}

func main() {
	parseEnv()

	switch filepath.Base(arg(0)) {
	case "DELETE", "GET", "HEAD":
		proc.Method = filepath.Base(arg(0))
		proc.URL = httpc.RequestURL(os.Getenv("HTTPC_URL"), arg(1))
		proc.Body = strings.NewReader("")
	case "PATCH", "POST", "PUT":
		proc.Method = filepath.Base(arg(0))
		proc.URL = httpc.RequestURL(os.Getenv("HTTPC_URL"), arg(1))
	default:
		proc.Method = arg(1)
		proc.URL = httpc.RequestURL(os.Getenv("HTTPC_URL"), arg(2))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	code, err := proc.Run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "httpc: %s\n", err)
		os.Exit(1)
	}
	if code/100 != 2 {
		os.Exit(code / 100)
	}
}

func parseEnv() {
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "HTTPC_") {
			continue
		}
		var k, v string
		if n := strings.Index(e, "="); n > -1 {
			k, v = e[:n], e[n+1:]
		} else {
			k = e
		}
		switch k {
		case "HTTPC_DISPLAY_HELPER":
			proc.Outputter = httpc.DisplayHelperOutputter{
				Helper: v,
				Out:    os.Stdout,
				Err:    os.Stderr,
			}
		case "HTTPC_INSECURE_SKIP_VERIFY":
			if v != "" {
				proc.Client.Transport = &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				}
			}
		case "HTTPC_URL":
			continue
		case "HTTPC_VERBOSE":
			if v != "" {
				proc.Verbose = true
			}
		default:
			proc.Header.Set(http.CanonicalHeaderKey(strings.ReplaceAll(k[6:], "_", "-")), v)
		}
	}
}

func arg(i int) string {
	if i < len(os.Args) {
		return os.Args[i]
	}
	return ""
}
