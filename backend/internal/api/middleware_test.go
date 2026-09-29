package api

import (
	"net/http"
	"testing"
)

func TestGetRealIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		realIP     string
		want       string
	}{
		{
			name:       "direct client ignores spoofed forwarded header",
			remoteAddr: "203.0.113.9:1234",
			xff:        "1.2.3.4",
			realIP:     "198.51.100.7",
			want:       "203.0.113.9",
		},
		{
			name:       "loopback proxy uses the hop it appended",
			remoteAddr: "127.0.0.1:1234",
			xff:        "1.2.3.4, 203.0.113.9",
			want:       "203.0.113.9",
		},
		{
			name:       "ipv6 loopback proxy uses the appended hop",
			remoteAddr: "[::1]:1234",
			xff:        "198.51.100.8",
			want:       "198.51.100.8",
		},
		{
			name:       "loopback without forwarding uses the peer",
			remoteAddr: "127.0.0.1:1234",
			realIP:     "198.51.100.7",
			want:       "127.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/api/health", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.realIP != "" {
				req.Header.Set("X-Real-Ip", tt.realIP)
			}
			if got := getRealIP(req); got != tt.want {
				t.Fatalf("getRealIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
