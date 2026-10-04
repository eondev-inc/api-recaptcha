package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func clientIPRouter(t *testing.T, trustedProxies string) (*gin.Engine, error) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	r := gin.New()
	err := ConfigureTrustedProxies(r, trustedProxies)
	r.GET("/ip", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ip": c.ClientIP()}) })

	return r, err
}

func requestIP(t *testing.T, r http.Handler, remoteAddr, forwardedFor string) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var body struct {
		IP string `json:"ip"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal %q: %v", rec.Body.String(), err)
	}

	return body.IP
}

// Gin's own default is to trust every peer, so an unconfigured router lets any caller choose its
// own ClientIP. An empty TRUSTED_PROXIES must mean "trust nobody".
func TestConfigureTrustedProxies_TrustsNobodyByDefault(t *testing.T) {
	r, err := clientIPRouter(t, "")
	if err != nil {
		t.Fatalf("an empty TRUSTED_PROXIES must be valid, got %v", err)
	}

	got := requestIP(t, r, "192.0.2.1:1234", "203.0.113.9")
	if got != "192.0.2.1" {
		t.Errorf("a forged X-Forwarded-For must be ignored, ClientIP was %q", got)
	}
}

func TestConfigureTrustedProxies_TrustsOnlyListedPeers(t *testing.T) {
	cases := []struct {
		name           string
		trustedProxies string
		remoteAddr     string
		want           string
	}{
		{name: "listed IP", trustedProxies: "192.0.2.1", remoteAddr: "192.0.2.1:1234", want: "203.0.113.9"},
		{name: "listed CIDR", trustedProxies: "192.0.2.0/24", remoteAddr: "192.0.2.7:1234", want: "203.0.113.9"},
		{name: "peer outside the list", trustedProxies: "10.0.0.0/8", remoteAddr: "192.0.2.1:1234", want: "192.0.2.1"},
		{name: "one of several", trustedProxies: "10.0.0.0/8, 192.0.2.1", remoteAddr: "192.0.2.1:1234", want: "203.0.113.9"},
		{name: "blank entries ignored", trustedProxies: ",,", remoteAddr: "192.0.2.1:1234", want: "192.0.2.1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := clientIPRouter(t, tc.trustedProxies)
			if err != nil {
				t.Fatalf("TRUSTED_PROXIES=%q must be valid, got %v", tc.trustedProxies, err)
			}

			if got := requestIP(t, r, tc.remoteAddr, "203.0.113.9"); got != tc.want {
				t.Errorf("TRUSTED_PROXIES=%q with peer %s: got ClientIP %q, want %q",
					tc.trustedProxies, tc.remoteAddr, got, tc.want)
			}
		})
	}
}

// A typo in the proxy list must not silently fall back to trusting everyone.
func TestConfigureTrustedProxies_RejectsInvalidEntries(t *testing.T) {
	for _, raw := range []string{"not-an-ip", "192.0.2.1/33", "10.0.0.0/8, oops"} {
		r, err := clientIPRouter(t, raw)
		if err == nil {
			t.Errorf("TRUSTED_PROXIES=%q must be rejected at startup", raw)
			continue
		}
		if got := requestIP(t, r, "192.0.2.1:1234", "203.0.113.9"); got != "192.0.2.1" {
			t.Errorf("after a rejected TRUSTED_PROXIES=%q the router must trust nobody, got %q", raw, got)
		}
	}
}
