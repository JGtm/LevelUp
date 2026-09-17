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

// TestEnrichEncounterFragRange : les deux bandes sont publiées, bornées aux matchs communs
// et à leur joueur, avec les rôles du registre comme clés de ligne.
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

	if stats.FragRangeSelf == nil || stats.FragRangeTarget == nil {
		t.Fatalf("les deux bandes attendues, got self=%v cible=%v", stats.FragRangeSelf, stats.FragRangeTarget)
	}
	// La clé de ligne est le RÔLE, jamais la clé d'arme (le front résout frags.role.*).
	if got := stats.FragRangeSelf.Weapons[0].WeaponKey; got != "precision" {
		t.Errorf("clé de ligne self = %q, want \"precision\"", got)
	}
	if got := stats.FragRangeTarget.Weapons[0].WeaponKey; got != "sniper" {
		t.Errorf("clé de ligne cible = %q, want \"sniper\"", got)
	}
	// Aucun libellé écrit côté Go (D8).
	if stats.FragRangeSelf.Weapons[0].Label != "" || stats.FragRangeSelf.Weapons[0].LabelEN != "" {
		t.Errorf("libellés attendus vides, got %q / %q",
			stats.FragRangeSelf.Weapons[0].Label, stats.FragRangeSelf.Weapons[0].LabelEN)
	}
	// Dénominateurs de couverture : ceux du scope, pas la carrière.
	if stats.FragRangeSelf.TotalKills != 40 || stats.FragRangeTarget.TotalKills != 33 {
		t.Errorf("totaux = self %d / cible %d, want 40 / 33",
			stats.FragRangeSelf.TotalKills, stats.FragRangeTarget.TotalKills)
	}
	// Bornage : deux lectures, chacune sur les matchs communs et UN joueur.
	if len(repo.vus) != 2 {
		t.Fatalf("%d lectures, want 2", len(repo.vus))
	}
	for i, f := range repo.vus {
		if len(f.MatchIDs) != 2 || len(f.XUIDs) != 1 {
			t.Errorf("lecture %d mal bornée : matchs=%v xuids=%v", i, f.MatchIDs, f.XUIDs)
		}
		if err := f.Validate(); err != nil {
			t.Errorf("lecture %d refusée par le port : %v", i, err)
		}
	}
}

// TestEnrichEncounterFragRange_UnSeulCote : un joueur sans frag mesuré laisse SA bande
// absente ; l'autre reste publiée. Les deux ne tombent pas ensemble.
func TestEnrichEncounterFragRange_UnSeulCote(t *testing.T) {
	t.Parallel()
	repo := &fakeWeaponRangeRepo{
		parXUID: map[string][]analysis.MeasuredKill{"self": mesures("hinf_br75", 10, 12)},
		dims:    map[string]port.WeaponDimensions{"hinf_br75": {Role: "precision"}},
	}
	svc := serviceAvecPortee(repo, &domain.ParticipantStatsAggregate{Kills: 40, Deaths: 30})
	stats := &domain.ExplorerEncounterStats{CountTogether: 4}

	svc.enrichEncounterFragRange(context.Background(), stats, "target-x", []string{"m1"}, nil)

	if stats.FragRangeSelf == nil {
		t.Error("bande du joueur courant attendue")
	}
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
			if stats.FragRangeSelf != nil || stats.FragRangeTarget != nil {
				t.Errorf("bandes attendues absentes, got self=%v cible=%v",
					stats.FragRangeSelf, stats.FragRangeTarget)
			}
		})
	}

	// Repo non câblé : no-op, et pas de panique.
	sansRepo := NewExplorerService(&mockExplorerRepo{}, "self")
	stats := &domain.ExplorerEncounterStats{CountTogether: 1}
	sansRepo.enrichEncounterFragRange(context.Background(), stats, "target-x", []string{"m1"}, nil)
	if stats.FragRangeSelf != nil || stats.FragRangeTarget != nil {
		t.Error("no-op attendu sans repo")
	}
	// stats nil : no-op, et pas de panique.
	serviceAvecPortee(&fakeWeaponRangeRepo{}, nil).
		enrichEncounterFragRange(context.Background(), nil, "target-x", []string{"m1"}, nil)
}

// TestEnrichEncounterFragRange_TotauxIllisibles : la lecture des totaux du joueur courant
// échoue → sa bande est QUAND MÊME publiée, couverture à zéro. Le bloc ne rend que les
// bandes ; perdre la portée parce qu'un dénominateur manque serait disproportionné.
func TestEnrichEncounterFragRange_TotauxIllisibles(t *testing.T) {
	t.Parallel()
	repo := &fakeWeaponRangeRepo{
		parXUID: map[string][]analysis.MeasuredKill{"self": mesures("hinf_br75", 10, 12)},
		dims:    map[string]port.WeaponDimensions{"hinf_br75": {Role: "precision"}},
	}
	svc := NewExplorerService(&mockExplorerRepo{participantsErr: errors.New("db")}, "self").
		WithWeaponRangeRepo(repo)
	stats := &domain.ExplorerEncounterStats{CountTogether: 2}

	svc.enrichEncounterFragRange(context.Background(), stats, "target-x", []string{"m1"}, nil)

	if stats.FragRangeSelf == nil {
		t.Fatal("bande du joueur courant attendue malgré les totaux illisibles")
	}
	if stats.FragRangeSelf.TotalKills != 0 {
		t.Errorf("TotalKills = %d, want 0 (couverture inconnue, jamais inventée)", stats.FragRangeSelf.TotalKills)
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
