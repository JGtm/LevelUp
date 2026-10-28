// Package archlint — i18n_tier_label_stored_not_served_test.go : la colonne `tier_label` de
// `match_skill_rank` n'est jamais servie (ADR 0040 D-2, plan i18n 8.2).
//
// Écrite dans la langue du moment de l'écriture (« Or VI » pour Halo Infinite, « Diamond 4 » pour
// Halo 5), elle donne un libellé dans la mauvaise langue à tout client dont la langue diffère : le
// libellé d'un palier se DÉDUIT du code (`tier` + `sub_tier`) dans la langue de la requête
// (`analysis.SkillTierDisplayLabel`). Les seules lectures SQL qui subsistent dans `internal/platform`
// servent à lire le nombre de matchs restants d'un placement (`domain.ParsePlacementRemaining`) ou à
// retrouver un code de palier manquant pour l'URL d'un badge : elles ne servent aucun texte. Le
// plafond par fichier ne monte pas ; une lecture de plus doit être justifiée ici avec sa date.
package archlint

import (
	"fmt"
	"strings"
	"testing"
)

// tierLabelReadsAllowed : nombre de lignes non commentées qui lisent `tier_label` dans un fichier de
// `internal/platform`, au 2026-10-28. queries_career.go : Q5 (valeur lue dans
// MatchHistoryRawRow.SkillStoredLabel). queries_home_citations.go : Q26 (enrichissement des matchs de
// l'Accueil, chemin legacy), Q26e (pic de rang : code du palier pour le badge) et Q26g (dernier rang
// par playlist : placement).
var tierLabelReadsAllowed = map[string]int{
	"internal/platform/duckdb/queries_career.go":         1,
	"internal/platform/duckdb/queries_home_citations.go": 4,
}

func TestStoredTierLabelIsNotServed(t *testing.T) {
	found := map[string]int{}
	for _, s := range goSourcesUnder(t, "internal/platform") {
		if strings.HasSuffix(s.Rel, "_test.go") {
			continue
		}
		for _, line := range strings.Split(string(s.Src), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "--") {
				continue
			}
			if strings.Contains(line, "tier_label") {
				found[s.Rel]++
			}
		}
	}
	var offenders []string
	for rel, n := range found {
		if n > tierLabelReadsAllowed[rel] {
			offenders = append(offenders, fmt.Sprintf("%s : %d lecture(s) de tier_label (plafond %d)", rel, n, tierLabelReadsAllowed[rel]))
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("`tier_label` ne se sert pas : déduire le libellé du code (analysis.SkillTierDisplayLabel) :\n%s",
			strings.Join(offenders, "\n"))
	}
}
