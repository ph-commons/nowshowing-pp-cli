// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

// Package httpx is the shared outbound GET path for the hand-written source
// clients (ctc, popcorn, imdb). It sends a browser User-Agent (ClickTheCity
// returns {error} without one), paces requests with an adaptive limiter, and
// surfaces a typed *cliutil.RateLimitError on 429 so an empty result is never
// confused with a throttled one.
package httpx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/ph-commons/nowshowing-pp-cli/internal/cliutil"
)

// DefaultUserAgent mirrors the required_headers User-Agent in the spec. A
// browser UA is mandatory for ClickTheCity and harmless for the other sources.
const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

const maxResponseBytes = 8 << 20 // 8 MiB ceiling on any single source page

// Client is a rate-limited GET client shared by the source packages.
type Client struct {
	hc  *http.Client
	ua  string
	lim *cliutil.AdaptiveLimiter
}

// New returns a Client with the default browser User-Agent and an adaptive
// limiter that discovers each host's ceiling from 429s.
func New() *Client {
	return &Client{
		hc:  &http.Client{},
		ua:  DefaultUserAgent,
		lim: cliutil.NewAdaptiveLimiterAuto(8),
	}
}

// GetBytes fetches url and returns the response body. It honors ctx, paces via
// the adaptive limiter, returns a *cliutil.RateLimitError on HTTP 429, and a
// plain error on any other non-2xx status.
func (c *Client) GetBytes(ctx context.Context, url string) ([]byte, error) {
	c.lim.Wait()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.ua)
	req.Header.Set("Accept", "*/*")
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		c.lim.OnRateLimit()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, &cliutil.RateLimitError{
			URL:        url,
			RetryAfter: retryAfter(resp.Header.Get("Retry-After")),
			Body:       string(body),
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	c.lim.OnSuccess()
	return io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
}

func retryAfter(h string) time.Duration {
	if h == "" {
		return 0
	}
	if secs, err := strconv.Atoi(h); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}
