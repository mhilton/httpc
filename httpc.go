package httpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

type Outputter interface {
	Output(ctx context.Context, contentType string, body io.Reader) error
}

type Proc struct {
	Method    string
	URL       string
	Header    http.Header
	Client    *http.Client
	Verbose   bool
	Body      io.Reader
	Outputter Outputter
	Err       io.Writer
}

func (p *Proc) Run(ctx context.Context) (int, error) {
	if p.Header.Get("Content-Type") == "" {
		var contentType string
		p.Body, contentType = sniffContentType(p.Body)
		if contentType != "" {
			p.Header.Set("Content-Type", contentType)
		}
	}

	req, err := http.NewRequest(p.Method, p.URL, p.Body)
	if err != nil {
		return 0, fmt.Errorf("%s %s: %w", p.Method, p.URL, err)
	}
	req.Header = p.Header
	req = req.WithContext(ctx)

	if p.Verbose {
		fmt.Fprintf(p.Err, "> %s %s\n", req.Method, req.URL)
		for k, v := range req.Header {
			for _, v1 := range v {
				fmt.Fprintf(p.Err, "> %s: %s\n", k, v1)
			}
		}
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("%s %s: %w", p.Method, p.URL, err)
	}
	if p.Verbose {
		fmt.Fprintf(p.Err, "< %s %s\n", resp.Proto, resp.Status)
		for k, v := range resp.Header {
			for _, v1 := range v {
				fmt.Fprintf(os.Stderr, "< %s: %s\n", k, v1)
			}
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Fprintf(p.Err, "httpc: %s %s: %s", p.Method, p.URL, resp.Status)
	}

	return resp.StatusCode, p.Outputter.Output(ctx, resp.Header.Get("Content-Type"), resp.Body)
}

func sniffContentType(body io.Reader) (io.Reader, string) {
	buf := new(bytes.Buffer)
	n, _ := io.CopyN(buf, body, 512)
	if n > 0 {
		body = io.MultiReader(buf, body)
		p := buf.Bytes()
		if sniffJSON(p) {
			return body, "application/json"
		}
		return body, http.DetectContentType(p)
	}
	return body, ""
}

func sniffJSON(p []byte) bool {
	r := bytes.NewReader(p)
	dec := json.NewDecoder(r)
	var v any
	err := dec.Decode(&v)
	if err == io.ErrUnexpectedEOF && len(p) == 512 {
		return true
	}
	if err != nil {
		return false
	}
	return dec.Decode(&v) != nil
}

func RequestURL(base, s string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		return s
	}
	u, err := url.Parse(s)
	if err != nil {
		return s
	}
	return baseURL.ResolveReference(u).String()
}

type SimpleOutputter struct {
	Out io.Writer
}

func (o SimpleOutputter) Output(_ context.Context, _ string, r io.Reader) error {
	_, err := io.Copy(o.Out, r)
	return err
}
