// Package handlers_test — default_status_ratchet_test.go : ratchet « statut de
// succès écrit à un seul endroit » (V721-04).
//
// POURQUOI. Un handler Huma qui répond 201/202 peut le faire de deux façons :
//   - un champ `Status int` dans sa struct de sortie — que Huma lit AU RUNTIME
//     mais qu'il ne sait PAS documenter : le document annonce alors 200 ;
//   - `humacore.DefaultStatus(...)` au montage — que Huma applique au runtime ET
//     au document.
//
// Le premier a produit 18 routes dont le contrat publié mentait (11 rattrapées à
// la main dans api/openapi_manual_fragment.yaml, 7 fausses en silence). La
// correction canonique est le modificateur ; ce ratchet interdit le retour du
// littéral, faute de quoi la dette re-croîtrait exactement comme avant.
//
// MAINTENANCE. Statut de succès FIXE = retirer le champ `Status` de la struct de
// sortie et poser `humacore.DefaultStatus(...)` au montage. Statut réellement
// DYNAMIQUE (une même opération répond 202 OU 409 avec un corps différent) = le
// champ reste le mécanisme runtime → 1 ligne d'allowlist AVEC justification, ET
// `humacore.DefaultStatus` au montage pour le statut nominal du document.

package handlers_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// dynamicStatusAllowlist — sorties dont le code de succès est POSÉ AU RUNTIME par
// un champ `Status int` parce qu'il varie selon la branche. Entrée = "fichier:type".
// DÉCROISSANTE : n'ajouter qu'avec raison.
var dynamicStatusAllowlist = []string{
	// 202 (job lancé) OU 409 (cycle auto-sync déjà en vol) — Body any, corps
	// d'enveloppe différent ; 202 nominal déclaré par humacore.DefaultStatus.
	"admin_actions.go:runSyncCycleOutput",
	// 202 (drain lancé) OU 409 (drain déjà en cours pour ce titre) — idem.
	"admin_actions_catalog_drain.go:catalogDrainOutput",
}

// successStatusLiteral repère un `Status: http.StatusCreated|Accepted` dans un
// littéral de struct de sortie Huma.
var successStatusLiteral = regexp.MustCompile(`Status:\s*http\.Status(Created|Accepted)`)

// outputTypeOfLiteral extrait le nom du type d'un `&xxxOutput{Status: http.Status…`
// (forme systématique des retours de handler concernés).
var outputTypeOfLiteral = regexp.MustCompile(`&(\w+)\{[^}]*Status:\s*http\.Status(?:Created|Accepted)`)

// TestNoHardcodedSuccessStatusOutsideAllowlist : aucun handler ne pose 201/202 via
// un champ `Status` hors des sorties à statut réellement dynamique.
func TestNoHardcodedSuccessStatusOutsideAllowlist(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) < 50 {
		t.Fatalf("%d fichiers Go scannés (attendu >= 50) — ratchet vide, sans valeur", len(files))
	}

	allowed := map[string]bool{}
	for _, k := range dynamicStatusAllowlist {
		allowed[k] = false
	}

	var offenders []string
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, rerr := os.ReadFile(f)
		if rerr != nil {
			t.Fatalf("lecture %s: %v", f, rerr)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if !successStatusLiteral.MatchString(line) {
				continue
			}
			typeName := ""
			if m := outputTypeOfLiteral.FindStringSubmatch(line); len(m) > 1 {
				typeName = m[1]
			}
			key := f + ":" + typeName
			if _, ok := allowed[key]; ok {
				allowed[key] = true
				continue
			}
			offenders = append(offenders, f+":"+strconv.Itoa(i+1)+" ("+key+")")
		}
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("%d statut(s) de succès 201/202 posé(s) par un champ `Status` hors allowlist — "+
			"Huma documenterait 200. Retirer le champ et poser humacore.DefaultStatus(...) au "+
			"montage (source unique : runtime ET document) :\n  %s",
			len(offenders), strings.Join(offenders, "\n  "))
	}

	// Anti-caducité : une entrée d'allowlist qui ne correspond plus à rien signale
	// un statut devenu fixe — la retirer plutôt que de la laisser périmer.
	for key, hit := range allowed {
		if !hit {
			t.Errorf("entrée d'allowlist PÉRIMÉE : %q — plus aucun `Status: http.StatusCreated/Accepted` "+
				"correspondant, retirer la ligne", key)
		}
	}
}
