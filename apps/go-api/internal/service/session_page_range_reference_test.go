package service

// session_page_range_reference_test.go — la PÉRIODE DE RÉFÉRENCE de la portée (lot U,
// décision D23-4), au niveau service.
//
// Ce que ces tests verrouillent :
//
//  1. UN SEUL bloc pour les deux colonnes du drawer : la référence vient du FILTRE de la
//     page, pas de la session affichée — il n'y a pas de `compare_range_reference` ;
//  2. les matchs des sessions AFFICHÉES sont TOUJOURS dans la fenêtre (sinon la
//     surbrillance de la session n'aurait rien à mettre en valeur), et la fenêtre est
//     bornée à rangeReferenceWindow matchs ;
//  3. référence tautologique (elle se réduit aux matchs de la session) ⇒ bloc OMIS ;
//  4. les seuils de rôle sont ceux de la période, calculés sur les points pleins.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// srrRows fabrique n matchs horodatés d'heure en heure, du plus ancien au plus récent.
func srrRows(prefixe string, n int, debut time.Time) []legacymatch.StatsMatchRow {
	out := make([]legacymatch.StatsMatchRow, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, legacymatch.StatsMatchRow{
			MatchID:   fmt.Sprintf("%s_%02d", prefixe, i),
			StartTime: debut.Add(time.Duration(i) * time.Hour),
			MapName:   "Live Fire",
		})
	}
	return out
}

// srrRepo rend, pour CHAQUE match du filtre reçu, cinq frags du joueur consulté à `dist` et
// cinq frags d'un adversaire à 30 m — de quoi faire un point plein par match.
type srrRepo struct {
	dist float64
	vu   []port.WeaponRangeFilters
}

func (r *srrRepo) LoadMatchRangeKills(
	_ context.Context, _ string, f port.WeaponRangeFilters,
) (port.MatchRangeRead, error) {
	r.vu = append(r.vu, f)
	kills := make([]analysis.MeasuredKill, 0, len(f.MatchIDs)*10)
	for _, id := range f.MatchIDs {
		for i := 0; i < 5; i++ {
			kills = append(kills,
				analysis.MeasuredKill{MatchID: id, KillerXUID: sprMain, TimeMS: int64(i),
					Side: analysis.SideKiller, DistanceM: r.dist},
				analysis.MeasuredKill{MatchID: id, KillerXUID: sprAutre, TimeMS: int64(100 + i),
					Side: analysis.SideKiller, DistanceM: 30},
			)
		}
	}
	return port.MatchRangeRead{Kills: kills, KillsTotal: len(kills)}, nil
}

func srrScope(ref, session, compare []legacymatch.StatsMatchRow) sessionBlocksScope {
	return sessionBlocksScope{Matches: session, CompareMatches: compare, ReferenceMatches: ref}
}

// TestRangeReference_UnSeulBlocPartageParLesDeuxColonnes : la référence ne dépend pas de la
// session affichée, donc le drawer n'en produit qu'une.
func TestRangeReference_UnSeulBlocPartageParLesDeuxColonnes(t *testing.T) {
	ref := srrRows("m", 10, time.Date(2026, 9, 20, 18, 0, 0, 0, time.UTC))
	session, compare := ref[8:], ref[:2]
	repo := &srrRepo{dist: 10}
	svc := sprService(repo)
	var resp domain.SessionPageResponse
	svc.attachSessionRange(context.Background(), &resp, srrScope(ref, session, compare))

	if resp.RangeProfiles == nil || resp.CompareRangeProfiles == nil {
		t.Fatalf("blocs de session = %v / %v, want les deux", resp.RangeProfiles, resp.CompareRangeProfiles)
	}
	if resp.RangeReference == nil {
		t.Fatal("range_reference nil, want un bloc")
	}
	if len(resp.RangeReference.Profiles) != 10 || resp.RangeReference.MatchesTotal != 10 {
		t.Errorf("reference = %d profils / %d matchs, want 10 / 10",
			len(resp.RangeReference.Profiles), resp.RangeReference.MatchesTotal)
	}
	// Les matchs des DEUX sessions sont dans le nuage : la surbrillance a de quoi peindre.
	vus := map[string]bool{}
	for _, p := range resp.RangeReference.Profiles {
		vus[p.MatchID] = true
	}
	for _, lot := range [][]legacymatch.StatsMatchRow{session, compare} {
		for _, m := range lot {
			if !vus[m.MatchID] {
				t.Errorf("match %s absent de la reference", m.MatchID)
			}
		}
	}
	// La référence est ordonnée du plus ancien au plus récent — l'axe des x du nuage.
	for i := 1; i < len(resp.RangeReference.Profiles); i++ {
		if resp.RangeReference.Profiles[i].PlayedAt.Before(resp.RangeReference.Profiles[i-1].PlayedAt) {
			t.Fatalf("profils non ordonnes a l'index %d", i)
		}
	}
	// Trois lectures, une par scope : session, session comparée, référence.
	if len(repo.vu) != 3 {
		t.Errorf("lectures = %d, want 3 (session, comparee, reference)", len(repo.vu))
	}
	for i, f := range repo.vu {
		if !f.AllPlayers {
			t.Errorf("lecture %d sans AllPlayers : la mediane du lobby n'aurait plus de sens", i)
		}
	}
}

// TestRangeReference_FenetreBorneeMaisSessionsToujoursRetenues.
func TestRangeReference_FenetreBorneeMaisSessionsToujoursRetenues(t *testing.T) {
	ref := srrRows("m", 60, time.Date(2026, 8, 1, 18, 0, 0, 0, time.UTC))
	// La session affichée est la PLUS ANCIENNE : elle sortirait d'une simple coupe des 30
	// derniers, et le nuage n'aurait rien à mettre en surbrillance.
	session := ref[:2]
	var resp domain.SessionPageResponse
	sprService(&srrRepo{dist: 10}).attachSessionRange(
		context.Background(), &resp, srrScope(ref, session, nil))

	if resp.RangeReference == nil {
		t.Fatal("range_reference nil, want un bloc")
	}
	if got := resp.RangeReference.MatchesTotal; got != rangeReferenceWindow {
		t.Errorf("matchs de la fenetre = %d, want %d", got, rangeReferenceWindow)
	}
	vus := map[string]bool{}
	for _, p := range resp.RangeReference.Profiles {
		vus[p.MatchID] = true
	}
	for _, m := range session {
		if !vus[m.MatchID] {
			t.Errorf("match de la session %s hors de la fenetre de reference", m.MatchID)
		}
	}
}

// TestRangeReference_OmisQuandTautologique : filtre resserré sur la session affichée.
func TestRangeReference_OmisQuandTautologique(t *testing.T) {
	session := srrRows("m", 4, time.Date(2026, 9, 21, 18, 0, 0, 0, time.UTC))
	var resp domain.SessionPageResponse
	sprService(&srrRepo{dist: 10}).attachSessionRange(
		context.Background(), &resp, srrScope(session, session, nil))

	if resp.RangeReference != nil {
		t.Errorf("range_reference = %+v, want nil : la reference se reduit au scope mesure",
			resp.RangeReference)
	}
	if resp.RangeProfiles == nil {
		t.Error("le bloc de session, lui, reste servi")
	}
}

// TestRangeReference_SeuilsDeRoleDeLaPeriode : les bandes viennent de la période entière.
func TestRangeReference_SeuilsDeRoleDeLaPeriode(t *testing.T) {
	ref := srrRows("m", 6, time.Date(2026, 9, 20, 18, 0, 0, 0, time.UTC))
	var resp domain.SessionPageResponse
	// Tous les matchs du joueur à 10 m, lobby à 30 m ⇒ chaque écart vaut -10 ; les trois
	// repères se confondent, et c'est exactement ce qu'une période homogène doit rendre.
	sprService(&srrRepo{dist: 10}).attachSessionRange(
		context.Background(), &resp, srrScope(ref, ref[5:], nil))

	if resp.RangeReference == nil || resp.RangeReference.RoleLowM == nil {
		t.Fatalf("reference = %+v, want des seuils", resp.RangeReference)
	}
	r := resp.RangeReference
	if *r.RoleLowM != -10 || *r.RoleHighM != -10 || *r.PeriodMedianDeltaM != -10 {
		t.Errorf("seuils = %v/%v/%v, want -10 partout",
			*r.RoleLowM, *r.RoleHighM, *r.PeriodMedianDeltaM)
	}
	if r.MatchesMeasured != 6 || r.MatchesTotal != 6 {
		t.Errorf("couverture = %d/%d, want 6/6", r.MatchesMeasured, r.MatchesTotal)
	}
}
