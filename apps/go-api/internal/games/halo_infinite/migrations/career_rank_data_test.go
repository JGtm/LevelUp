package migrations

import (
	"fmt"
	"hash/fnv"
	"testing"

	"levelup/go-api/internal/migration"
)

// goldenCareerRankHash est le FNV-1a 64 de la sérialisation canonique des 544
// lignes produites par BuildHaloCareerRankTranslations AVANT le déplacement MT-07
// (capturé sur l'arbre pré-refactor). Garde-fou byte-identique : tout changement
// de la grille de rangs (grades, tiers, algorithme) casse ce test.
const (
	goldenCareerRankHash = 0x4c3eb7c01615f4eb
	goldenCareerRankLen  = 544
)

// hashCareerRows sérialise les lignes (ordre inclus) et retourne le FNV-1a 64.
func hashCareerRows(rows []migration.CareerRankTranslation) uint64 {
	h := fnv.New64a()
	for _, r := range rows {
		fmt.Fprintf(h, "%d|%s|%s|%s\n", r.RankID, r.Lang, r.Title, r.Tier)
	}
	return h.Sum64()
}

// TestCareerRankTranslations_GoldenParity garantit que le générateur déplacé
// (MT-07) produit EXACTEMENT les 544 lignes historiques (hash + longueur + bornes).
func TestCareerRankTranslations_GoldenParity(t *testing.T) {
	rows := CareerRankTranslations()

	if len(rows) != goldenCareerRankLen {
		t.Fatalf("len = %d, want %d", len(rows), goldenCareerRankLen)
	}
	if got := hashCareerRows(rows); got != goldenCareerRankHash {
		t.Fatalf("hash = %#x, want %#x (grille de rangs modifiée — parité rompue)", got, goldenCareerRankHash)
	}

	// Bornes explicites (lisibilité du diff si le hash casse).
	want := []migration.CareerRankTranslation{
		{RankID: 0, Lang: "fr", Title: "Recrue", Tier: "Bronze"},
		{RankID: 0, Lang: "en", Title: "Recruit", Tier: "Bronze"},
		{RankID: 1, Lang: "fr", Title: "Cadet 1", Tier: "Bronze"},
		{RankID: 1, Lang: "en", Title: "Cadet 1", Tier: "Bronze"},
	}
	for i, w := range want {
		if rows[i] != w {
			t.Errorf("row[%d] = %+v, want %+v", i, rows[i], w)
		}
	}
	if last := rows[len(rows)-1]; last != (migration.CareerRankTranslation{RankID: 271, Lang: "en", Title: "Hero", Tier: "Onyx"}) {
		t.Errorf("dernière ligne = %+v, want {271 en Hero Onyx}", last)
	}
	if penult := rows[len(rows)-2]; penult != (migration.CareerRankTranslation{RankID: 271, Lang: "fr", Title: "Héros", Tier: "Onyx"}) {
		t.Errorf("avant-dernière ligne = %+v, want {271 fr Héros Onyx}", penult)
	}
}
