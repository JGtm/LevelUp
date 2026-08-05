// netguard_coverage_test.go — garde-rail : tout fichier qui émet une requête HTTP
// SORTANTE doit passer par netguard.Check, sinon le mode démo cesse d'être hermétique.
//
// POURQUOI CE RATCHET. Le coupe-circuit démo est posé à la frontière sortante de
// chaque client. Rien, dans le compilateur, n'empêche d'ajouter demain un
// `client.Do(req)` sans garde — et la fuite serait invisible (elle ne casse aucun
// test, elle ralentit juste la démo jusqu'à l'affamer, cf. l'incident qui a motivé
// le package). Ce test scanne donc `internal/` et échoue sur tout fichier
// contenant un appel sortant sans `netguard.Check`, hors allowlist justifiée.
package netguard_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// outboundCallMarkers : formes d'émission d'une requête HTTP sortante.
var outboundCallMarkers = []string{
	".Do(req)",
	".Do(httpReq)",
	"http.Get(",
	"http.Post(",
	"http.PostForm(",
}

// allowlist : fichiers émettant une requête sortante SANS garde netguard, avec
// la raison. Toute entrée ajoutée ici doit être justifiée — un fichier
// allowlisté est une fuite potentielle du mode démo.
//
// Établie le 2026-08-04 (chantier fixture démo). Principe retenu : on garde les
// surfaces DE DONNÉES (Halo, Xbox, assets, OAuth refresh) — celles qu'une page
// de la démo peut déclencher — et on laisse hors garde la POIGNÉE DE MAIN
// d'authentification interactive et les canaux désactivés en démo.
var allowlist = map[string]string{
	// Poignée de main d'auth INTERACTIVE (login SSO, capture de token en CLI).
	// Jamais atteinte en démo : LEVELUP_AUTH_MODE=none, aucun flux de login monté
	// (cf. docker-compose service levelup-demo). La porte d'entrée réellement
	// atteignable depuis un boot démo — le rafraîchissement OAuth — EST gardée
	// (oauth_refresh.go) : sans token frais, ces échanges ne partent pas.
	"platform/auth/auth_code.go":        "login interactif — non monté en démo",
	"platform/auth/device_token.go":     "device code flow — CLI uniquement",
	"platform/auth/xbox_device_code.go": "device code flow — CLI uniquement",
	"platform/auth/sisu_client.go":      "poignée de main SISU — dépend d'un token frais, coupé en amont",
	// halo_exchange.go porte aussi le helper `postJSON` par lequel passent
	// xsts.go et sisu_provider.go : ces deux-là n'émettent rien en propre et
	// n'ont donc pas à figurer ici.
	"platform/auth/halo_exchange.go":         "échange Spartan (+ helper postJSON) — dépend d'un token frais, coupé en amont",
	"platform/auth/peoplehub_resolver.go":    "résolution gamertag — dépend d'un token frais, coupé en amont",
	"platform/auth/xbox_profile_resolver.go": "résolution profil — dépend d'un token frais, coupé en amont",

	// Canaux sortants NON liés aux données de jeu, désactivés par la config démo
	// (app_settings émis par le seed : discord_notifications_enabled=false,
	// watcher_presence_enabled=false).
	"notify/discord.go":        "webhook Discord — désactivé par app_settings en démo",
	"presence/rest_client.go":  "présence Xbox — daemon watcher désactivé en démo",
	"presence/steam_poller.go": "présence Steam — daemon watcher désactivé en démo",
}

func TestOutboundCallsAreNetguarded(t *testing.T) {
	root := filepath.Join("..", "..", "..", "internal")
	var leaks []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, readErr := os.ReadFile(path) //nolint:gosec // chemin dérivé du walk du repo
		if readErr != nil {
			return readErr
		}
		src := string(content)

		hasOutbound := false
		for _, marker := range outboundCallMarkers {
			if strings.Contains(src, marker) {
				hasOutbound = true
				break
			}
		}
		if !hasOutbound || strings.Contains(src, "netguard.Check") {
			return nil
		}

		// Clé d'allowlist : chemin relatif à internal/, séparateurs normalisés.
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		key := filepath.ToSlash(rel)
		if _, ok := allowlist[key]; ok {
			return nil
		}
		leaks = append(leaks, key)
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}

	if len(leaks) > 0 {
		t.Errorf(
			"appel HTTP sortant sans netguard.Check (le mode démo fuiterait) dans :\n  %s\n\n"+
				"Corriger en ajoutant `if err := netguard.Check(ctx, \"<surface>\"); err != nil { ... }` "+
				"AVANT l'émission, et en traitant l'erreur par le chemin de dégradation existant. "+
				"Si le fichier ne peut réellement pas être atteint en mode démo, l'ajouter à "+
				"`allowlist` avec une justification.",
			strings.Join(leaks, "\n  "),
		)
	}
}

// TestAllowlistHasNoStaleEntry : une entrée d'allowlist qui ne correspond plus à
// un fichier sortant est du bruit — elle ferait croire à une exception encore
// active. On échoue pour forcer le nettoyage.
func TestAllowlistHasNoStaleEntry(t *testing.T) {
	root := filepath.Join("..", "..", "..", "internal")
	for key := range allowlist {
		path := filepath.Join(root, filepath.FromSlash(key))
		content, err := os.ReadFile(path) //nolint:gosec // chemin dérivé de l'allowlist du test
		if err != nil {
			t.Errorf("allowlist: %s n'existe plus — retirer l'entrée", key)
			continue
		}
		src := string(content)
		outbound := false
		for _, marker := range outboundCallMarkers {
			if strings.Contains(src, marker) {
				outbound = true
				break
			}
		}
		if !outbound {
			t.Errorf("allowlist: %s n'émet plus d'appel sortant — retirer l'entrée", key)
		}
	}
}
