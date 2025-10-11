package core

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
)

const (
	maxFeedDownloadBytes   = 8 << 20 // 8 MiB hard limit to guard against oversized feeds
	defaultFeedHTTPTimeout = 15 * time.Second
)

var errFeedBodyTooLarge = errors.New("feed body exceeds configured limit")

func newFeedParser() *gofeed.Parser {
	parser := gofeed.NewParser()
	parser.Client = &http.Client{
		Timeout:   defaultFeedHTTPTimeout,
		Transport: &limitedBodyTransport{limit: maxFeedDownloadBytes},
	}
	return parser
}

type limitedBodyTransport struct {
	base  http.RoundTripper
	limit int64
}

func (t *limitedBodyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport := t.base
	if transport == nil {
		transport = http.DefaultTransport
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if t.limit <= 0 {
		return resp, nil
	}

	if resp.ContentLength > 0 && resp.ContentLength > t.limit {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%w: content-length %d exceeds %d", errFeedBodyTooLarge, resp.ContentLength, t.limit)
	}

	resp.Body = newLimitedReadCloser(resp.Body, t.limit)
	return resp, nil
}

type limitedReadCloser struct {
	reader    io.ReadCloser
	remaining int64
	err       error
}

func newLimitedReadCloser(rc io.ReadCloser, limit int64) *limitedReadCloser {
	return &limitedReadCloser{
		reader:    rc,
		remaining: limit,
	}
}

func (l *limitedReadCloser) Read(p []byte) (int, error) {
	if l.err != nil {
		return 0, l.err
	}
	if l.remaining <= 0 {
		l.err = errFeedBodyTooLarge
		return 0, l.err
	}

	if int64(len(p)) > l.remaining+1 {
		p = p[:l.remaining+1]
	}

	n, err := l.reader.Read(p)
	if int64(n) <= l.remaining {
		l.remaining -= int64(n)
		l.err = err
		return n, err
	}

	n = int(l.remaining)
	l.remaining = 0
	l.err = errFeedBodyTooLarge
	return n, l.err
}

func (l *limitedReadCloser) Close() error {
	return l.reader.Close()
}
