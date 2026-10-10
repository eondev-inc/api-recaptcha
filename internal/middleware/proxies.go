package middleware

import (
	"fmt"
	"net"
	"strings"

	"github.com/gin-gonic/gin"

	"api-recaptcha/internal/logger"
)

// ConfigureTrustedProxies decides which peers may set the forwarded-IP headers that ClientIP
// reads.
//
// TRUSTED_PROXIES is a comma-separated list of IP addresses or CIDR blocks. When it is unset no
// peer is trusted, so ClientIP returns the real remote address: a caller cannot pick its own
// rate-limit bucket, or its own identity in the logs, by forging X-Forwarded-For. Gin's own
// default is the opposite — it trusts every peer — which is why this has to be called at startup.
//
// List only your own ingress, never a shared or public range. An entry that is neither an IP nor
// a CIDR block fails startup instead of silently degrading to a wider trust set.
func ConfigureTrustedProxies(engine *gin.Engine, trustedProxies string) error {
	proxies, err := parseTrustedProxies(trustedProxies)
	if err != nil {
		// Trusting nobody is the safe outcome of every failure path: never leave the engine on
		// Gin's trust-everyone default just because the configuration was rejected.
		_ = engine.SetTrustedProxies(nil)
		return err
	}

	if len(proxies) == 0 {
		logger.Log.Info("TRUSTED_PROXIES set to no usable entry, X-Forwarded-For is ignored from every peer")
		return engine.SetTrustedProxies(nil)
	}

	logger.Log.Info("trusting forwarded-IP headers", "proxies", proxies)

	return engine.SetTrustedProxies(proxies)
}

// parseTrustedProxies splits the configured list and rejects anything that is neither an IP
// address nor a CIDR block.
func parseTrustedProxies(trustedProxies string) ([]string, error) {
	raw := strings.TrimSpace(trustedProxies)
	if raw == "" {
		return nil, nil
	}

	proxies := make([]string, 0, strings.Count(raw, ",")+1)
	for _, proxy := range strings.Split(raw, ",") {
		proxy = strings.TrimSpace(proxy)
		if proxy == "" {
			continue
		}
		if net.ParseIP(proxy) == nil {
			if _, _, err := net.ParseCIDR(proxy); err != nil {
				return nil, fmt.Errorf("TRUSTED_PROXIES entry %q is neither an IP address nor a CIDR block", proxy)
			}
		}
		proxies = append(proxies, proxy)
	}

	return proxies, nil
}

// parseTrustedProxies splits the configured list and rejects anything that is neither an IP
// address nor a CIDR block. An entry that cannot be parsed fails startup instead of silently
// degrading to a wider trust set.
