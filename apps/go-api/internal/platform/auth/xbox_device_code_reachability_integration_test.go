//go:build integration

// xbox_device_code_reachability_integration_test.go — garde-rail réseau opt-in.
//
// POURQUOI (incident 2026-07-13, PLAN_AUTH_DEVICE_FLOW_SISU_404) : la cause
// racine du login SSO Xbox cassé était l'URL de démarrage device-code erronée
// (`/oauth20_connect/device` → HTTP 404), restée INVISIBLE parce que TOUS les
// tests unitaires du package injectent des URLs httptest mockées (conclusion D
// du plan) — aucun test ne touchait jamais l'endpoint Microsoft réel.
//
// Ce test exerce la constante RÉELLE `xboxDeviceCodeURL` via StartXboxDeviceCode
// et asserte que Microsoft répond 2xx + device_code. Si MS déplace/retire à
// nouveau l'endpoint, ou si quelqu'un régresse l'URL, il échoue immédiatement.
//
// Choix « test opt-in réseau » plutôt que « sonde de santé au boot » (justifié) :
//   - coût runtime NUL : inerte tant qu'on ne l'invoque pas explicitement ;
//   - zéro faux WARN en dev offline (une sonde au boot bruiterait chaque
//     démarrage sans réseau → fatigue d'alerte) et zéro dépendance réseau au
//     démarrage du serveur ;
//   - DOUBLE garde : tag `integration` (exclu du `go test ./...` par défaut) ET
//     variable d'env `LEVELUP_DEVICE_ENDPOINT_LIVE_CHECK` — ainsi le gate de
//     livraison `go test -tags=integration ./...` (suites anti-ART) SKIP ce test
//     sans réseau et ne peut PAS flaker le CI ;
//   - exerce la vraie constante : le blind spot documenté est refermé.
//
// Invocation manuelle (ou job planifié) :
//
//	LEVELUP_DEVICE_ENDPOINT_LIVE_CHECK=1 \
//	  go test -tags=integration -run TestXboxDeviceCodeEndpointReachable \
//	  ./internal/platform/auth/
package auth

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestXboxDeviceCodeEndpointReachable vérifie que l'endpoint MSA de démarrage du
// Device Code Flow natif Xbox (constante xboxDeviceCodeURL) répond réellement.
func TestXboxDeviceCodeEndpointReachable(t *testing.T) {
	if os.Getenv("LEVELUP_DEVICE_ENDPOINT_LIVE_CHECK") == "" {
		t.Skip("garde-rail réseau opt-in : poser LEVELUP_DEVICE_ENDPOINT_LIVE_CHECK=1 pour joindre login.live.com")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// StartXboxDeviceCode utilise la constante réelle xboxDeviceCodeURL + les
	// scopes Xbox réels — c'est exactement le chemin de production.
	res, err := StartXboxDeviceCode(ctx, SISUDefaultAppID)
	if err != nil {
		t.Fatalf("endpoint device-code Xbox injoignable via la constante xboxDeviceCodeURL (%s) — "+
			"Microsoft a-t-il déplacé/retiré l'endpoint, ou l'URL a-t-elle régressé ? err: %v",
			xboxDeviceCodeURL, err)
	}
	if res.DeviceCode == "" {
		t.Fatalf("réponse 2xx mais device_code absent (%s) — schéma de réponse Microsoft changé ?",
			xboxDeviceCodeURL)
	}
	if res.UserCode == "" {
		t.Errorf("user_code absent — l'UI ne pourrait afficher aucun code à saisir")
	}
	t.Logf("OK: endpoint %s joignable, user_code obtenu (expires_in=%d s)", xboxDeviceCodeURL, res.ExpiresIn)
}
