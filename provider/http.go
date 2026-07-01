package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// dataURI encodes image bytes as a base64 data URI accepted by Fal and Meshy.
func dataURI(image []byte, mime string) string {
	if mime == "" {
		mime = "image/png"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(image)
}

// pollInterval is how long to wait between job-status polls. It is a package
// variable so tests can shrink it; jobs that complete on the first poll never
// wait at all.
var pollInterval = 3 * time.Second

// pollTimeout bounds how long a single job is polled before giving up. It is
// generous by default: heavy models (e.g. Fal pro) can spend many minutes in
// cold-start/queue before compute even begins, and timing out would discard a
// job the provider is still running - and that you have already paid for.
var pollTimeout = 30 * time.Minute

// SetWaitTimeout overrides how long Generate/Fetch will wait for a job. A
// non-positive duration leaves the default unchanged.
func SetWaitTimeout(d time.Duration) {
	if d > 0 {
		pollTimeout = d
	}
}

// postJSON sends a JSON body and decodes the JSON response into out.
func postJSON(ctx context.Context, client *http.Client, url string, headers map[string]string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return doJSON(client, req, out)
}

// getJSON performs a GET and decodes the JSON response into out.
func getJSON(ctx context.Context, client *http.Client, url string, headers map[string]string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return doJSON(client, req, out)
}

func doJSON(client *http.Client, req *http.Request, out any) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed (status %d): %s", resp.StatusCode, string(respBody))
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

// download fetches a URL and returns its raw bytes.
func download(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating download request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading model: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("model download failed (status %d)", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// pollFunc checks a job once and reports whether it is done. It returns
// (done=true) when the job has finished successfully, an error if the job
// failed, or (done=false, nil) to keep polling.
type pollFunc func() (done bool, err error)

// poll calls check immediately, then every pollInterval until it is done,
// the context is cancelled, or pollTimeout elapses.
func poll(ctx context.Context, check pollFunc) error {
	deadline := time.Now().Add(pollTimeout)
	for {
		done, err := check()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("job timed out after %s", pollTimeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}
