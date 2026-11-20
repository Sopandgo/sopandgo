package errreport_test

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// ingest plays a Sentry/GlitchTip ingest endpoint and keeps every event
// payload it receives, raw and decoded.
type ingest struct {
	srv    *httptest.Server
	mu     sync.Mutex
	raw    []string
	events []map[string]any
	fail   bool
}

func newIngest(t *testing.T) *ingest {
	t.Helper()
	in := &ingest{}
	in.srv = httptest.NewServer(http.HandlerFunc(in.serve))
	t.Cleanup(in.srv.Close)
	return in
}

// dsn points at this endpoint, project 1.
func (in *ingest) dsn() string {
	return "http://public-key@" + strings.TrimPrefix(in.srv.URL, "http://") + "/1"
}

func (in *ingest) setFail(v bool) {
	in.mu.Lock()
	defer in.mu.Unlock()
	in.fail = v
}

func (in *ingest) serve(w http.ResponseWriter, r *http.Request) {
	in.mu.Lock()
	defer in.mu.Unlock()
	if in.fail {
		http.Error(w, "ingest down", http.StatusInternalServerError)
		return
	}
	var body io.Reader = r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		zr, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		body = zr
	}
	b, _ := io.ReadAll(body)
	// An envelope is newline-separated JSON: a header, then item header and
	// payload pairs.
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	var lines []string
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	for i := 1; i+1 < len(lines); i++ {
		var item struct {
			Type string `json:"type"`
		}
		if json.Unmarshal([]byte(lines[i]), &item) == nil && item.Type == "event" {
			var ev map[string]any
			if err := json.Unmarshal([]byte(lines[i+1]), &ev); err == nil {
				in.raw = append(in.raw, lines[i+1])
				in.events = append(in.events, ev)
			}
			i++
		}
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"id":"ok"}`))
}

func (in *ingest) received() ([]map[string]any, []string) {
	in.mu.Lock()
	defer in.mu.Unlock()
	return append([]map[string]any(nil), in.events...), append([]string(nil), in.raw...)
}
