package service

// filters_map_variants_test.go — le pipeline de filtres devant des lignes dont les variantes de
// carte ont été rattachées à leur carte de base par le repo (platform/duckdb/map_variants.go) :
// une seule option par carte, comptes cumulés, filtrage qui ramène la base ET ses variantes.

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

// ligneCarte : une ligne de filtres telle que le repo la rend une fois les variantes rattachées
// (nom EN et FR = ceux de la carte de base, identifiant = celui de la carte jouée).
func ligneCarte(matchID, mapID, nom string) domain.FilterMatchRow {
	t := time.Now()
	return domain.FilterMatchRow{
		MatchID:   matchID,
		StartTime: &t,
		MapID:     strPtr(mapID),
		MapName:   i18n.FromColumns(nom, nom),
	}
}

func lignesAvecVariantes() []domain.FilterMatchRow {
	return []domain.FilterMatchRow{
		ligneCarte("m1", "thunderhead", "Thunderhead"),
		ligneCarte("m2", "thunderhead-heavies", "Thunderhead"), // variante rattachée
		ligneCarte("m3", "thunderhead-heavies", "Thunderhead"),
		ligneCarte("m4", "live-fire-ranked", "Live Fire"), // variante dont la base n'a aucune ligne
		ligneCarte("m5", "aquarius", "Aquarius"),
	}
}

func TestFiltresDeCartes_UneOptionParCarteDeBase(t *testing.T) {
	avail := buildAvailableOptions(lignesAvecVariantes(), domain.CascadeFilter{})

	if got := len(avail.Maps); got != 3 {
		t.Fatalf("3 cartes attendues (Aquarius, Live Fire, Thunderhead), reçu %d : %v", got, labelValues(avail.Maps))
	}
	for _, lv := range avail.Maps {
		if lv.Label == "Thunderhead" && lv.Count != 3 {
			t.Errorf("Thunderhead doit cumuler la base et sa variante : %d matchs, attendu 3", lv.Count)
		}
	}
}

func TestFiltresDeCartes_ChoisirLaBaseRameneSesVariantes(t *testing.T) {
	resolved := ResolveFiltersFromRows(lignesAvecVariantes(), domain.FilterContextInput{
		FilterMode: "period",
		Cascade:    domain.CascadeFilter{Maps: []string{"Thunderhead"}},
	})
	if got := resolved.Counts.TotalMatchesAfterFilters; got != 3 {
		t.Errorf("Thunderhead = base (1) + variante (2) : %d matchs après filtre, attendu 3", got)
	}
	if got := resolved.Counts.TotalMatchesBeforeFilters; got != 5 {
		t.Errorf("total avant filtres = %d, attendu 5", got)
	}
}

// Une variante dont la base n'a aucune partie reste proposée sous le nom de la base.
func TestFiltresDeCartes_VarianteSansPartieSurLaBase(t *testing.T) {
	resolved := ResolveFiltersFromRows(lignesAvecVariantes(), domain.FilterContextInput{
		FilterMode: "period",
		Cascade:    domain.CascadeFilter{Maps: []string{"Live Fire"}},
	})
	if got := resolved.Counts.TotalMatchesAfterFilters; got != 1 {
		t.Errorf("Live Fire : %d matchs après filtre, attendu 1", got)
	}
}
