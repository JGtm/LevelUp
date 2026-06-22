// Package halo — auth_retry_guard_test.go : garde-fou C2 (statique).
//
// Empêche par construction qu'une entrée live token-gated per-player contourne le
// filet d'auto-réparation `retryOnAuth`. Si un nouveau fetch live per-player est ajouté
// SANS retryOnAuth, ce test casse le build — le contributeur doit soit l'enrober, soit
// l'ajouter explicitement à l'allowlist ci-dessous (avec justification), comme
// no_art_patterns_test.go pour les écritures append-only.
package halo

import (
	"os"
	"strings"
	"testing"
)

// tokenGatedEntryPoints : fonctions du package halo qui émettent un fetch live
// token-gated PER-PLAYER et DOIVENT donc être enrobées de retryOnAuth (filet 401).
// fichier → liste des signatures de fonctions.
var tokenGatedEntryPoints = map[string][]string{
	"provider.go": {
		"func (p *HaloProvider) GetBattlePassWithRaw(",
		"func (p *HaloProvider) GetChallengesWithRaw(",
	},
	"compare_provider.go": {
		"func (p *HaloProvider) FetchServiceRecord(",
		"func (p *HaloProvider) FetchSeasonServiceRecord(",
	},
}

func TestAuthRetryGuard_PerPlayerEntryPointsUseRetryOnAuth(t *testing.T) {
	for file, funcs := range tokenGatedEntryPoints {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("lecture %s: %v", file, err)
		}
		for _, sig := range funcs {
			body, ok := extractFuncBody(string(src), sig)
			if !ok {
				t.Errorf("%s: fonction introuvable %q — l'allowlist est-elle à jour ?", file, sig)
				continue
			}
			if !strings.Contains(body, "retryOnAuth(") {
				t.Errorf("%s: %q n'enrobe PAS son fetch live dans retryOnAuth — "+
					"un 401/403 ne se réparerait pas (token périmé servi tout le process). "+
					"Enrober via retryOnAuth, ou justifier dans l'allowlist.", file, sig)
			}
		}
	}
}

// Le sentinel d'échec auth doit rester wrappé dans doGet, sinon retryOnAuth ne peut
// plus détecter les 401/403 et le filet devient inopérant silencieusement.
func TestAuthRetryGuard_DoGetWrapsAuthSentinel(t *testing.T) {
	src, err := os.ReadFile("provider.go")
	if err != nil {
		t.Fatalf("lecture provider.go: %v", err)
	}
	body, ok := extractFuncBody(string(src), "func (p *HaloProvider) doGet(")
	if !ok {
		t.Fatal("doGet introuvable dans provider.go")
	}
	if !strings.Contains(body, "StatusUnauthorized") || !strings.Contains(body, "errHaloAuthFailure") {
		t.Error("doGet doit wrapper errHaloAuthFailure sur 401/403 (détection du filet retryOnAuth)")
	}
}

// extractFuncBody retourne le corps (accolades incluses) de la première fonction dont
// la déclaration contient sig, par comptage d'accolades. ok=false si introuvable.
func extractFuncBody(src, sig string) (string, bool) {
	start := strings.Index(src, sig)
	if start < 0 {
		return "", false
	}
	open := strings.IndexByte(src[start:], '{')
	if open < 0 {
		return "", false
	}
	open += start
	depth := 0
	for i := open; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[open : i+1], true
			}
		}
	}
	return "", false
}
