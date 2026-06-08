// Package middleware — request_scheme_test.go : couverture de la détection de
// schéma HTTPS et de la policy de cookie Secure (fix onboarding 2026-06-08).
package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func reqWith(tlsOn bool, fwdProto string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if tlsOn {
		r.TLS = &tls.ConnectionState{}
	}
	if fwdProto != "" {
		r.Header.Set("X-Forwarded-Proto", fwdProto)
	}
	return r
}

func TestRequestIsHTTPS(t *testing.T) {
	cases := []struct {
		name       string
		tlsOn      bool
		fwdProto   string
		trustProxy bool
		want       bool
	}{
		{"http nu", false, "", false, false},
		{"tls natif", true, "", false, true},
		{"proxy https + trust", false, "https", true, true},
		{"proxy https sans trust (anti-spoof)", false, "https", false, false},
		{"proxy http + trust", false, "http", true, false},
		{"proxy casse HTTPS + trust", false, "HTTPS", true, true},
		{"proxy liste https,http + trust", false, "https, http", true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := RequestIsHTTPS(reqWith(c.tlsOn, c.fwdProto), c.trustProxy); got != c.want {
				t.Errorf("RequestIsHTTPS = %v, attendu %v", got, c.want)
			}
		})
	}
}

func TestSecureCookiePolicy_Secure(t *testing.T) {
	cases := []struct {
		name       string
		mode       string
		tlsOn      bool
		fwdProto   string
		trustProxy bool
		want       bool
	}{
		{"auto http nu", "auto", false, "", false, false},
		{"auto tls", "auto", true, "", false, true},
		{"auto proxy https trust", "auto", false, "https", true, true},
		{"vide => auto http nu", "", false, "", false, false},
		{"inconnu => auto http nu", "banane", false, "", false, false},
		{"force true sur http nu", "true", false, "", false, true},
		{"force false sur tls", "false", true, "", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := SecureCookiePolicy{Mode: c.mode, TrustProxy: c.trustProxy}
			if got := p.Secure(reqWith(c.tlsOn, c.fwdProto)); got != c.want {
				t.Errorf("policy{%q}.Secure = %v, attendu %v", c.mode, got, c.want)
			}
		})
	}
}
