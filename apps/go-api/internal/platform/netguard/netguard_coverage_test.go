// netguard_coverage_test.go — garde-rail : tout fichier qui émet une requête HTTP
// SORTANTE doit passer par netguard.Check, sinon le mode démo cesse d'être hermétique.
//
// POURQUOI CE RATCHET. Le coupe-circuit démo est posé à la frontière sortante de
// chaque client. Rien, dans le compilateur, n'empêche d'ajouter demain un
// `client.Do(req)` sans garde — et la fuite serait invisible (elle ne casse aucun
// test, elle ralentit juste la démo jusqu'à l'affamer, cf. l'incident qui a motivé
// le package). Ce test scanne donc `internal/` et échoue sur tout fichier
// contenant un appel sortant sans `netguard.Check`, hors allowlist justifiée.
//
// BRIQUE `internal/haloapi` : ses fichiers émettent par un `haloapi.Doer` reçu et n'ont jamais
// `netguard.Check` à côté de `.Do(req)` : ils sont exemptés par PRÉFIXE (`prefixeBriqueHaloapi`).
// Côté exploitation, ce ratchet VOIT les fichiers qui définissent un `Doer` gardé
// (`platform/auth/xbl_doer.go`, `games/halo_infinite/haloadapter/doer.go` : `netguard.Check` puis
// `next.Do(req)`) ; il ne voit PAS si chaque client de la brique reçoit bien ce `Doer`. Ce qui tient
// l'hermétisme de la brique : (1) elle ne construit ni n'atteint aucun client HTTP
// (`archlint/haloapi_sans_client_http_test.go`) ; (2) tout client de `haloapi/infinite` construit par
// go-api reçoit un littéral `netguardDoer{...}` écrit à l'appel (`archlint/haloapi_doer_garde_test.go`) ;
// (3) les huit surfaces de l'adaptateur sont prouvées muettes en mode démo
// (`haloadapter/netguard_surfaces_test.go`). Les jambes d'authentification de `haloapi/xblauth` à
// qui la façade `platform/auth` passe un client nu (échanges XBL/XSTS/Spartan, code d'autorisation,
// résolveurs de profil) ne sont pas gardées : elles exigent un jeton frais, que le rafraîchissement
// gardé ne produit pas en démo.
package netguard_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// prefixeBriqueHaloapi : la brique haloapi (chemin relatif à internal/), exemptée par préfixe :
// elle émet par un Doer reçu, dont la garde est vue chez l'exploitation (voir l'en-tête).
const prefixeBriqueHaloapi = "haloapi/"

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
	// Les fichiers de la brique `haloapi/` n'y figurent pas : ils sont exemptés par préfixe (voir
	// l'en-tête). La poignée de main d'auth INTERACTIVE (code d'autorisation, échanges XBL/XSTS/
	// Spartan, résolveurs de profil), à qui la façade passe un client nu, n'est jamais atteinte en
	// démo : LEVELUP_AUTH_MODE=none, aucun flux de login monté, et le rafraîchissement OAuth — seule
	// porte d'entrée atteignable depuis un boot démo — EST gardé (`platform/auth/xbl_oauth.go`, par
	// le Doer gardé de `xbl_doer.go`) : sans jeton frais, ces échanges ne partent pas.
	//
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
		if strings.HasPrefix(key, prefixeBriqueHaloapi) {
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
		if strings.HasPrefix(key, prefixeBriqueHaloapi) {
			t.Errorf("allowlist: %s est déjà exempté par le préfixe de la brique — retirer l'entrée", key)
			continue
		}
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
