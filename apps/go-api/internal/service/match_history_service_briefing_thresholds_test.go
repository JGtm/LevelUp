package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// briefingMapGroups construit un scope où la carte i porte counts[i] matchs. Mode et
// sélection sont uniques : seule la dimension « carte » peut être libre.
func briefingMapGroups(counts ...int) []domain.MatchHistoryRawRow {
	var rows []domain.MatchHistoryRawRow
	for g, n := range counts {
		mapID := fmt.Sprintf("map%d", g)
		label := fmt.Sprintf("Carte %d", g)
		for i := 0; i < n; i++ {
			rows = append(rows, briefingRaw(fmt.Sprintf("g%d-%d", g, i), g*30+i, domain.OutcomeWin, 10, 5, 2, 60, mapID, label, "Slayer", "Arène"))
		}
	}
	return rows
}

func briefingMapDimension(t *testing.T, scope []domain.MatchHistoryRawRow) *domain.ExplorerBriefingDimension {
	t.Helper()
	for _, d := range buildBriefingDimensions(scope, scope, domain.GroupesDeCartes{}) {
		if d.Dimension == "map" {
			d := d
			return &d
		}
	}
	return nil
}

// D1/D3 : deux groupes au seuil (5 et 5) → une vraie comparaison.
func TestBuildDimension_TwoGroupsAtThresholdAreEntries(t *testing.T) {
	d := briefingMapDimension(t, briefingMapGroups(MinDimensionGroupMatches, MinDimensionGroupMatches))
	if d == nil || d.Insufficient || len(d.Entries) != 2 {
		t.Fatalf("5 et 5 : 2 entrées attendues, got %+v", d)
	}
}

// D3 : groupes à 4 et 5 → un seul qualifié, la carte est servie en état vide.
func TestBuildDimension_OneQualifiedGroupIsInsufficient(t *testing.T) {
	d := briefingMapDimension(t, briefingMapGroups(MinDimensionGroupMatches-1, MinDimensionGroupMatches))
	if d == nil || !d.Insufficient {
		t.Fatalf("4 et 5 : Insufficient attendu, got %+v", d)
	}
	if d.Entries == nil || len(d.Entries) != 0 {
		t.Fatalf("Entries doit être une slice vide non nil, got %#v", d.Entries)
	}
}

// D3 : aucun groupe qualifié (trois cartes à 4) → Insufficient.
func TestBuildDimension_NoQualifiedGroupIsInsufficient(t *testing.T) {
	n := MinDimensionGroupMatches - 1
	d := briefingMapDimension(t, briefingMapGroups(n, n, n))
	if d == nil || !d.Insufficient || len(d.Entries) != 0 {
		t.Fatalf("aucun groupe qualifié : Insufficient attendu, got %+v", d)
	}
}

// D4 : une seule valeur dans le scope → carte absente (rien à comparer).
func TestBuildDimension_SingleValueIsNil(t *testing.T) {
	if d := briefingMapDimension(t, briefingMapGroups(20)); d != nil {
		t.Fatalf("une seule carte : dimension absente attendue, got %+v", d)
	}
}

// La carte en état vide se sérialise avec `entries: []` (jamais null) et `insufficient`.
func TestBuildDimension_InsufficientSerializesEmptyEntries(t *testing.T) {
	d := briefingMapDimension(t, briefingMapGroups(2, 2))
	if d == nil {
		t.Fatal("dimension attendue")
	}
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(raw)
	if !strings.Contains(got, `"entries":[]`) || !strings.Contains(got, `"insufficient":true`) {
		t.Fatalf("JSON = %s, attendu entries:[] et insufficient:true", got)
	}
}

// Une dimension normale ne porte pas `insufficient` (omitempty).
func TestBuildDimension_NormalOmitsInsufficient(t *testing.T) {
	d := briefingMapDimension(t, briefingMapGroups(6, 6))
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), "insufficient") {
		t.Fatalf("JSON = %s, insufficient ne doit pas être émis", raw)
	}
}

// D6 : sous 10 matchs notés, seuls les blocs qui comparent sont omis ; Classement,
// arme favorite, séries et moments forts restent servis.
func TestBuildExplorerBriefing_LowSampleKeepsNonComparativeModules(t *testing.T) {
	tiers := []string{"Bronze I", "Bronze II"}
	var filtered []domain.MatchHistoryRawRow
	for i := 0; i < 8; i++ {
		label := &tiers[0]
		if i == 0 {
			label = &tiers[1]
		}
		r := briefingRankedRaw(rankedRawOpts{
			id: fmt.Sprintf("m%d", i), daysAgo: i, ratingType: "CSR", group: "ranked",
			tierLabel: label, ratingValue: float64Ptr(float64(100 + 10*(8-i))),
		})
		r.DominanceFlag = analysis.DominanceFlagDomination
		filtered = append(filtered, r)
	}
	repo := &briefingWeaponsRepo{rows: []port.WeaponKillRow{{WeaponID: 1, Label: "Fusil de combat", Kills: 40}}}
	svc := &MatchHistoryService{rankedCapable: true, weaponKillsRepo: repo, weaponKillsXUID: "2533"}

	b := svc.buildExplorerBriefing(context.Background(), filtered, filtered)
	if b == nil || !b.LowSample {
		t.Fatalf("LowSample attendu, got %+v", b)
	}
	if b.Ranked == nil || b.Weapons == nil || b.Streaks == nil || b.Dominance == nil {
		t.Fatalf("Ranked/Weapons/Streaks/Dominance attendus non nil : ranked=%v weapons=%v streaks=%v dominance=%v",
			b.Ranked, b.Weapons, b.Streaks, b.Dominance)
	}
	if b.Baseline != nil || b.Dimensions != nil || b.ContextSplit != nil {
		t.Errorf("Baseline/Dimensions/ContextSplit doivent être nil : %+v %+v %+v", b.Baseline, b.Dimensions, b.ContextSplit)
	}
	if b.MinGroupMatches != 0 {
		t.Errorf("MinGroupMatches = %d, attendu 0 en échantillon faible", b.MinGroupMatches)
	}
}

// Hors échantillon faible, le seuil de groupe est servi au front.
func TestBuildExplorerBriefing_ServesMinGroupMatches(t *testing.T) {
	scope := briefingMapGroups(6, 6)
	b := svcWithRanked(false).buildExplorerBriefing(context.Background(), scope, scope)
	if b == nil || b.LowSample || b.MinGroupMatches != MinDimensionGroupMatches {
		t.Fatalf("MinGroupMatches = %+v, attendu %d", b, MinDimensionGroupMatches)
	}
}

// D7 : le seuil des chaînes du Classement reste 10, indépendamment du seuil de
// dimension (5) : une chaîne à 7 matchs à côté d'une chaîne à 12 est écartée.
func TestBuildBriefingRanked_ChainThresholdStaysTen(t *testing.T) {
	if MinRankedChainMatches != 10 {
		t.Fatalf("MinRankedChainMatches = %d, attendu 10 (D7)", MinRankedChainMatches)
	}
	var scope []domain.MatchHistoryRawRow
	for i := 0; i < 12; i++ {
		scope = append(scope, briefingRankedRaw(rankedRawOpts{
			id: fmt.Sprintf("c%d", i), daysAgo: i, ratingType: "CSR", group: "ranked", ratingValue: float64Ptr(100),
		}))
	}
	for i := 0; i < 7; i++ {
		scope = append(scope, briefingRankedRaw(rankedRawOpts{
			id: fmt.Sprintf("l%d", i), daysAgo: 50 + i, ratingType: "LUSR", group: "arena_slayer", ratingValue: float64Ptr(20),
		}))
	}
	r := buildBriefingRanked(context.Background(), scope)
	if r == nil || len(r.Kinds) != 1 {
		t.Fatalf("une seule chaîne attendue (celle à 12), got %+v", r)
	}
	if k := r.Kinds[0]; k.Kind != "csr" || k.Matches != 12 {
		t.Errorf("chaîne = %s/%d, attendu csr/12", k.Kind, k.Matches)
	}
}
