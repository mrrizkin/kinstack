package bootstrap

import (
	"testing"

	"kinstack/config"
)

func TestTrustProxyConfigFollowsDeployment(t *testing.T) {
	if got := trustProxyConfig(config.AppConfig{Env: "development"}); got != nil {
		t.Fatalf("development must not trust forwarded headers: %+v", got)
	}

	production := trustProxyConfig(
		config.AppConfig{Env: "production", TrustedProxies: []string{"203.0.113.7"}},
	)
	if production == nil || !production.Loopback || !production.Private {
		t.Fatalf("production must trust loopback and private peers: %+v", production)
	}
	if len(production.Proxies) != 1 || production.Proxies[0] != "203.0.113.7" {
		t.Fatalf("configured proxy missing: %+v", production.Proxies)
	}

	staging := trustProxyConfig(
		config.AppConfig{Env: "staging", TrustedProxies: []string{"10.1.2.3"}},
	)
	if staging == nil {
		t.Fatal("an explicit trusted proxy must enable forwarded headers outside production")
	}
}
