package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"
)

func TestHaloServicesUnreachable(t *testing.T) {
	for _, tc := range []struct {
		nom  string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"délai dépassé enveloppé", fmt.Errorf("sisu_provider: échange XBL: %w", fmt.Errorf("XSTS token: %w", context.DeadlineExceeded)), true},
		{"erreur réseau d'URL", fmt.Errorf("XSTS token: %w", &url.Error{Op: "Post", URL: "https://xsts.auth.xboxlive.com", Err: &net.DNSError{Err: "no such host", IsNotFound: true}}), true},
		{"panne 500", errors.New("spartan token: HTTP 500 depuis https://halostats/x: oups"), true},
		{"panne 503", errors.New("XSTS token: HTTP 503 depuis https://xsts/x: indisponible"), true},
		{"ralentissement 429", errors.New("user token XBL: HTTP 429 depuis https://user.auth/x: trop"), true},
		{"délai de requête 408", errors.New("clearance token HTTP 408"), true},
		{"pas de profil Xbox (401)", errors.New("XSTS token: HTTP 401 depuis https://xsts/x: {\"XErr\":2148916233}"), false},
		{"accès refusé (403)", errors.New("spartan token: HTTP 403 depuis https://halo/x: refusé"), false},
		{"jeton absent de la réponse", errors.New("XSTS token: token absent dans la réponse XSTS"), false},
		{"annulation du contexte", fmt.Errorf("échange: %w", context.Canceled), false},
	} {
		if got := HaloServicesUnreachable(tc.err); got != tc.want {
			t.Errorf("%s : HaloServicesUnreachable = %v, attendu %v", tc.nom, got, tc.want)
		}
	}
}
