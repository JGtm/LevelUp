package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// ─── Double de test du lecteur de frags mesurés ──────────────────────────────

type fakeWeaponRangeRepo struct {
	// parXUID : les frags mesurés rendus pour chaque joueur. Un xuid absent rend zéro
	// ligne sans erreur — l'état nominal d'un joueur dont aucun match n'est décodé.
	parXUID map[string][]analysis.MeasuredKill
	err     error
	dims    map[string]port.WeaponDimensions
	dimsErr error
	// vus : les filtres reçus, pour vérifier le bornage (matchs + xuid).
	vus []port.WeaponRangeFilters
}

func (f *fakeWeaponRangeRepo) LoadWeaponRange(
	_ context.Context, _ string, filters port.WeaponRangeFilters,
) ([]analysis.MeasuredKill, error) {
	f.vus = append(f.vus, filters)
	if f.err != nil {
		return nil, f.err
	}
	if len(filters.XUIDs) == 0 {
		return nil, nil
	}
	return f.parXUID[filters.XUIDs[0]], nil
}

func (f *fakeWeaponRangeRepo) LoadWeaponOpening(
	_ context.Context, _ string, _ port.WeaponRangeFilters,
) ([]analysis.MeasuredKill, error) {
	return nil, nil
}

func (f *fakeWeaponRangeRepo) ResolveWeaponDimensions(
	_ context.Context, _ string, _ []string,
) (map[string]port.WeaponDimensions, error) {
	return f.dims, f.dimsErr
}

func (f *fakeWeaponRangeRepo) ResolveWeaponLabels(
	_ context.Context, _ []string,
) (map[string]port.WeaponLabel, error) {
	return nil, nil
}

// mesures fabrique n frags mesurés d'une arme, côté tueur, à distance croissante — assez
// pour passer le seuil de publication (analysis.WeaponRangeMinMeasured).
func mesures(weaponKey string, n int, base float64) []analysis.MeasuredKill {
	out := make([]analysis.MeasuredKill, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, analysis.MeasuredKill{
			MatchID:   fmt.Sprintf("m%d", i),
			WeaponKey: weaponKey,
			Side:      analysis.SideKiller,
			DistanceM: base + float64(i),
		})
	}
	return out
}

func serviceAvecPortee(repo *fakeWeaponRangeRepo, parts *domain.ParticipantStatsAggregate) *ExplorerService {
	return NewExplorerService(&mockExplorerRepo{participants: parts}, "self").
		WithWeaponRangeRepo(repo)
}

// ─── Tests ───────────────────────────────────────────────────────────────────

// TestEnrichEncounterFragRange : la bande de la CIBLE est publiée, bornée aux matchs
// communs et à son joueur, avec les rôles du registre comme clés de ligne. Depuis la
// décision 7 du plan d'ajustements pré-v7.5 (2026-09-19) le côté du joueur courant n'est
// plus publié : une seule lecture part au repo.
func TestEnrichEncounterFragRange(t *testing.T) {
	t.Parallel()
	repo := &fakeWeaponRangeRepo{
		parXUID: map[string][]analysis.MeasuredKill{
			"self":     mesures("hinf_br75", 10, 12),
			"target-x": mesures("hinf_sniper", 9, 40),
		},
		dims: map[string]port.WeaponDimensions{
			"hinf_br75":   {Class: "shoulder", Role: "precision"},
			"hinf_sniper": {Class: "heavy", Role: "sniper"},
		},
	}
	svc := serviceAvecPortee(repo, &domain.ParticipantStatsAggregate{Kills: 40, Deaths: 30})
	stats := &domain.ExplorerEncounterStats{CountTogether: 4}
	sample := &domain.ExplorerTargetSampleStats{Kills: 33, Deaths: 28}

	svc.enrichEncounterFragRange(context.Background(), stats, "target-x", []string{"m1", "m2"}, sample)

	if stats.FragRangeTarget == nil {
		t.Fatal("bande de la cible attendue")
	}
	// La clé de ligne est le RÔLE, jamais la clé d'arme (le front résout frags.role.*).
	if got := stats.FragRangeTarget.Weapons[0].WeaponKey; got != "sniper" {
		t.Errorf("clé de ligne cible = %q, want \"sniper\"", got)
	}
	// Aucun libellé écrit côté Go (D8).
	if stats.FragRangeTarget.Weapons[0].Label != "" || stats.FragRangeTarget.Weapons[0].LabelEN != "" {
		t.Errorf("libellés attendus vides, got %q / %q",
			stats.FragRangeTarget.Weapons[0].Label, stats.FragRangeTarget.Weapons[0].LabelEN)
	}
	// Dénominateurs de couverture : ceux du scope, pas la carrière.
	if stats.FragRangeTarget.TotalKills != 33 {
		t.Errorf("total cible = %d, want 33", stats.FragRangeTarget.TotalKills)
	}
	// Bornage : UNE seule lecture — celle de la cible — sur les matchs communs et un joueur.
	if len(repo.vus) != 1 {
		t.Fatalf("%d lectures, want 1 (le côté du joueur courant n'est plus lu)", len(repo.vus))
	}
	f := repo.vus[0]
	if len(f.MatchIDs) != 2 || len(f.XUIDs) != 1 || f.XUIDs[0] != "target-x" {
		t.Errorf("lecture mal bornée : matchs=%v xuids=%v", f.MatchIDs, f.XUIDs)
	}
	if err := f.Validate(); err != nil {
		t.Errorf("lecture refusée par le port : %v", err)
	}
}

// TestEnrichEncounterFragRange_CibleSansMesure : la cible n'a aucun frag mesuré → aucune
// bande, et surtout pas celle du joueur courant en remplacement.
func TestEnrichEncounterFragRange_CibleSansMesure(t *testing.T) {
	t.Parallel()
	repo := &fakeWeaponRangeRepo{
		parXUID: map[string][]analysis.MeasuredKill{"self": mesures("hinf_br75", 10, 12)},
		dims:    map[string]port.WeaponDimensions{"hinf_br75": {Role: "precision"}},
	}
	svc := serviceAvecPortee(repo, &domain.ParticipantStatsAggregate{Kills: 40, Deaths: 30})
	stats := &domain.ExplorerEncounterStats{CountTogether: 4}

	svc.enrichEncounterFragRange(context.Background(), stats, "target-x", []string{"m1"}, nil)

	if stats.FragRangeTarget != nil {
		t.Errorf("bande cible attendue absente (aucun frag mesuré), got %+v", stats.FragRangeTarget)
	}
}

// TestEnrichEncounterFragRange_Degradations : capability absente, repo en erreur, registre
// muet, repo non câblé, scope vide, stats nil → aucune bande, aucune panique.
func TestEnrichEncounterFragRange_Degradations(t *testing.T) {
	t.Parallel()
	mesuresOK := map[string][]analysis.MeasuredKill{
		"self": mesures("hinf_br75", 10, 12), "target-x": mesures("hinf_br75", 10, 12),
	}
	cas := []struct {
		nom      string
		repo     *fakeWeaponRangeRepo
		target   string
		matchIDs []string
	}{
		{"capability absente", &fakeWeaponRangeRepo{err: games.ErrCapabilityNotSupported}, "target-x", []string{"m1"}},
		{"lecture en echec", &fakeWeaponRangeRepo{err: errors.New("boom")}, "target-x", []string{"m1"}},
		{"registre muet", &fakeWeaponRangeRepo{parXUID: mesuresOK, dimsErr: errors.New("metadata")}, "target-x", []string{"m1"}},
		{"aucun role connu", &fakeWeaponRangeRepo{parXUID: mesuresOK, dims: map[string]port.WeaponDimensions{}}, "target-x", []string{"m1"}},
		{"cible inconnue", &fakeWeaponRangeRepo{parXUID: mesuresOK}, "", []string{"m1"}},
		{"aucun match commun", &fakeWeaponRangeRepo{parXUID: mesuresOK}, "target-x", nil},
	}
	for _, tc := range cas {
		t.Run(tc.nom, func(t *testing.T) {
			t.Parallel()
			svc := serviceAvecPortee(tc.repo, &domain.ParticipantStatsAggregate{Kills: 40})
			stats := &domain.ExplorerEncounterStats{CountTogether: 1}
			svc.enrichEncounterFragRange(context.Background(), stats, tc.target, tc.matchIDs, nil)
			if stats.FragRangeTarget != nil {
				t.Errorf("bande attendue absente, got cible=%v", stats.FragRangeTarget)
			}
		})
	}

	// Repo non câblé : no-op, et pas de panique.
	sansRepo := NewExplorerService(&mockExplorerRepo{}, "self")
	stats := &domain.ExplorerEncounterStats{CountTogether: 1}
	sansRepo.enrichEncounterFragRange(context.Background(), stats, "target-x", []string{"m1"}, nil)
	if stats.FragRangeTarget != nil {
		t.Error("no-op attendu sans repo")
	}
	// stats nil : no-op, et pas de panique.
	serviceAvecPortee(&fakeWeaponRangeRepo{}, nil).
		enrichEncounterFragRange(context.Background(), nil, "target-x", []string{"m1"}, nil)
}

// TestEnrichEncounterFragRange_SansAgregatDeCible : l'encart n'a pas calculé les totaux de
// la cible → la bande est QUAND MÊME publiée, couverture à zéro. Le bloc ne rend que les
// bandes ; perdre la portée parce qu'un dénominateur manque serait disproportionné.
func TestEnrichEncounterFragRange_SansAgregatDeCible(t *testing.T) {
	t.Parallel()
	repo := &fakeWeaponRangeRepo{
		parXUID: map[string][]analysis.MeasuredKill{"target-x": mesures("hinf_br75", 10, 12)},
		dims:    map[string]port.WeaponDimensions{"hinf_br75": {Role: "precision"}},
	}
	svc := serviceAvecPortee(repo, nil)
	stats := &domain.ExplorerEncounterStats{CountTogether: 2}

	svc.enrichEncounterFragRange(context.Background(), stats, "target-x", []string{"m1"}, nil)

	if stats.FragRangeTarget == nil {
		t.Fatal("bande de la cible attendue malgré l'absence d'agrégat")
	}
	if stats.FragRangeTarget.TotalKills != 0 {
		t.Errorf("TotalKills = %d, want 0 (couverture inconnue, jamais inventée)", stats.FragRangeTarget.TotalKills)
	}
}

// TestWeaponRangeByRoleChaineUnique — GARDE-RAIL : la chaîne « frags mesurés → rôles →
// regroupement » n'existe qu'une fois.
//
// Deux surfaces la rendent (Face-à-face, Explorer) et une troisième viendra. Une copie
// divergerait au premier réglage de seuil, et deux pages afficheraient alors deux portées
// des mêmes frags. `analysis.RegroupMeasuredKills` est la signature de la chaîne : il ne
// doit s'appeler que depuis `weapon_range_by_role.go`.
func TestWeaponRangeByRoleChaineUnique(t *testing.T) {
	t.Parallel()
	const autorise = "weapon_range_by_role.go"
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	var fautifs []string
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") || nom == autorise {
			continue
		}
		src, err := os.ReadFile(filepath.Clean(nom))
		if err != nil {
			t.Fatalf("lecture de %s : %v", nom, err)
		}
		if strings.Contains(string(src), "analysis.RegroupMeasuredKills(") {
			fautifs = append(fautifs, nom)
		}
	}
	if len(fautifs) > 0 {
		t.Errorf("la chaîne de portée par rôle est réécrite dans %v — appeler "+
			"buildWeaponRangeByRole (%s) plutôt que de la recopier", fautifs, autorise)
	}
}
