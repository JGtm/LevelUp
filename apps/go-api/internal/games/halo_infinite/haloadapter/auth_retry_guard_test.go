// Package haloadapter — auth_retry_guard_test.go : garde-fou C2 (statique).
//
// Empêche par construction qu'une entrée live token-gated per-player contourne le
// filet d'auto-réparation `retryOnAuth`. Si un nouveau fetch live per-player est ajouté
// SANS retryOnAuth, ce test casse le build — le contributeur doit soit l'enrober, soit
// l'ajouter explicitement à l'allowlist ci-dessous (avec justification), comme
// no_art_patterns_test.go pour les écritures append-only.
package haloadapter

import (
	"os"
	"strings"
	"testing"
)

// tokenGatedEntryPoints : fonctions du paquet qui émettent un fetch live
// token-gated PER-PLAYER et DOIVENT donc être enrobées de retryOnAuth (filet 401).
// fichier → liste des signatures de fonctions.
var tokenGatedEntryPoints = map[string][]string{
	"provider_operations.go": {
		"func (p *Provider) GetBattlePassWithRaw(",
		"func (p *Provider) GetChallengesWithRaw(",
	},
	// Le calendrier de saisons est un fichier GameCMS commun au titre, mais l'appel est signé du
	// jeton du lecteur : même péremption possible, même filet (D5 du lot aj14-d, 2026-10-14).
	"season_provider.go": {
		"func (p *Provider) FetchSeasonCalendar(",
	},
	"compare_provider.go": {
		"func (p *Provider) FetchServiceRecord(",
		"func (p *Provider) FetchSeasonServiceRecord(",
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

// Le filet doit reconnaître le refus d'authentification par le verdict TYPÉ de la brique (statut
// 401/403 de `infinite.HTTPError`) : sans lui, retryOnAuth ne détecte plus les 401/403 et le filet
// devient inopérant silencieusement.
func TestAuthRetryGuard_DoGetWrapsAuthSentinel(t *testing.T) {
	src, err := os.ReadFile("auth_retry.go")
	if err != nil {
		t.Fatalf("lecture auth_retry.go: %v", err)
	}
	filetBody, ok := extractFuncBody(string(src), "func retryOnAuth[")
	if !ok {
		t.Fatal("retryOnAuth introuvable dans auth_retry.go")
	}
	predicat, ok := extractFuncBody(string(src), "func IsAuthError(")
	if !ok {
		t.Fatal("IsAuthError introuvable dans auth_retry.go")
	}
	if !strings.Contains(filetBody, "IsAuthError") || !strings.Contains(predicat, "infinite.IsAuthError") {
		t.Error("retryOnAuth doit reconnaître les 401/403 par infinite.IsAuthError (détection du filet)")
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
