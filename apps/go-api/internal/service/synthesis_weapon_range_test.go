// Package service — synthesis_weapon_range_test.go : la section « Portée par arme » vue du
// service (lot 4 du plan .ai/PLAN_DUELS_PORTEE_2026-09-06.md).
//
// Ce que ces tests verrouillent, dans l'ordre d'importance :
//
//  1. LE BLOC D'ENTAME EST NIL, JAMAIS UN ZÉRO (D5). Un `SynthesisOpening{}` publié se lirait
//     « ce joueur engage au contact » alors que la vérité est « on ne sait pas » — et c'est
//     l'état NOMINAL tant que le backfill de kill_openings n'a pas tourné.
//  2. les deux côtés vivent sur la MÊME ligne d'arme, chacun optionnel ;
//  3. les totaux viennent du scope canonique, pas de la table de positions (sans quoi la
//     couverture afficherait toujours 100 %) ;
//  4. les armes sous le seuil sont NOMMÉES et ventilées par côté ;
//  5. capability absente / repo nil / scope vide / erreur SQL : section absente, jamais de
//     panique et jamais une page cassée.
package service

import (
	"context"
	"errors"
	"math"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

const epsRange = 1e-9

// mockWeaponRangeRepo — un port.WeaponRangeRepository scriptable.
type mockWeaponRangeRepo struct {
	kills     []analysis.MeasuredKill
	killsErr  error
	openings  []analysis.MeasuredKill
	openErr   error
	labels    map[string]port.WeaponLabel
	labelsErr error

	killCalls, openCalls, labelCalls int
	lastFilters                      port.WeaponRangeFilters
}

func (m *mockWeaponRangeRepo) LoadWeaponRange(
	_ context.Context, _ string, f port.WeaponRangeFilters,
) ([]analysis.MeasuredKill, error) {
	m.killCalls++
	m.lastFilters = f
	return m.kills, m.killsErr
}

func (m *mockWeaponRangeRepo) LoadWeaponOpening(
	_ context.Context, _ string, f port.WeaponRangeFilters,
) ([]analysis.MeasuredKill, error) {
	m.openCalls++
	m.lastFilters = f
	return m.openings, m.openErr
}

func (m *mockWeaponRangeRepo) ResolveWeaponLabels(
	_ context.Context, _ []string,
) (map[string]port.WeaponLabel, error) {
	m.labelCalls++
	return m.labels, m.labelsErr
}

// wrKill : un frag mesuré de fixture.
func wrKill(weapon string, side analysis.Side, timeMS int64, dist, dz float64) analysis.MeasuredKill {
	return analysis.MeasuredKill{
		MatchID: "m1", KillerXUID: "k" + string(side), TimeMS: timeMS,
		WeaponKey: weapon, Side: side, DistanceM: dist, DeltaZ: dz,
	}
}

// wrKills : `n` frags d'une arme et d'un côté, à distance fixe et dénivelé fixe. Les instants
// sont distincts (la clé du frag les distingue) et décalés par côté, pour que les deux
// populations ne se recouvrent jamais.
func wrKills(weapon string, side analysis.Side, n int, dist, dz float64) []analysis.MeasuredKill {
	base := int64(0)
	if side == analysis.SideVictim {
		base = 100000
	}
	out := make([]analysis.MeasuredKill, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, wrKill(weapon, side, base+int64(i), dist, dz))
	}
	return out
}

// wrCanonRows : un scope canonique de `matchs` matchs portant chacun kills/deaths.
func wrCanonRows(matchs, kills, deaths int) []canonical.PlayerMatchRow {
	out := make([]canonical.PlayerMatchRow, 0, matchs)
	for i := 0; i < matchs; i++ {
		k, d := kills, deaths
		out = append(out, canonical.PlayerMatchRow{
			Summary: canonical.MatchSummary{MatchID: "m" + string(rune('a'+i))},
			Self:    canonical.MatchParticipant{Kills: &k, Deaths: &d},
		})
	}
	return out
}

func wrService(repo port.WeaponRangeRepository) *SynthesisService {
	return NewSynthesisService(&mockSynthesisRepo{}).
		WithPlayerMatchesRepo(&mockSynthesisPlayerMatches{}, "halo_infinite", "GT").
		WithWeaponRangeRepo(repo)
}

// TestLoadWeaponRange_Nominal_DeuxCotesEtEntame — le chemin complet.
func TestLoadWeaponRange_Nominal_DeuxCotesEtEntame(t *testing.T) {
	kills := append(
		wrKills("hinf_br75", analysis.SideKiller, 10, 12, 2),   // je frague à 12 m, d'en haut
		wrKills("hinf_br75", analysis.SideVictim, 8, 20, 2)..., // je meurs à 20 m ; dz BRUT +2
	)
	// Une entame par frag du côté tueur : 18 m -> 12 m, l'engagement se ferme de 6 m.
	openings := make([]analysis.MeasuredKill, 0, 10)
	for _, k := range kills {
		if k.Side == analysis.SideKiller {
			o := k
			o.DistanceM = 18
			openings = append(openings, o)
		}
	}
	repo := &mockWeaponRangeRepo{
		kills: kills, openings: openings,
		labels: map[string]port.WeaponLabel{
			"hinf_br75": {Label: "Fusil de combat BR75", LabelEN: "BR75 Battle Rifle"},
		},
	}

	block := wrService(repo).loadWeaponRange(context.Background(), wrCanonRows(2, 9, 7))
	if block == nil {
		t.Fatal("section nil, attendue peuplée")
	}
	if repo.killCalls != 1 || repo.openCalls != 1 || repo.labelCalls != 1 {
		t.Errorf("appels repo = %d/%d/%d, attendu 1/1/1",
			repo.killCalls, repo.openCalls, repo.labelCalls)
	}
	// Le scope arrive par MatchIDs + Gamertag (D10), jamais par une période.
	if len(repo.lastFilters.MatchIDs) != 2 || repo.lastFilters.Gamertag != "GT" {
		t.Errorf("filtres = %+v, attendu 2 match_id et le gamertag", repo.lastFilters)
	}

	if len(block.Weapons) != 1 {
		t.Fatalf("%d ligne(s) d'arme, attendu 1 (les deux côtés fusionnent) : %+v",
			len(block.Weapons), block.Weapons)
	}
	w := block.Weapons[0]
	if w.WeaponKey != "hinf_br75" || w.Label != "Fusil de combat BR75" || w.LabelEN != "BR75 Battle Rifle" {
		t.Errorf("ligne = %+v, libellés non hydratés", w)
	}
	if w.Kills == nil || w.Deaths == nil {
		t.Fatalf("les deux côtés doivent être renseignés : %+v", w)
	}
	if w.Kills.Measured != 10 || math.Abs(w.Kills.Median-12) > epsRange {
		t.Errorf("côté frags = %+v, attendu 10 mesures à 12 m", w.Kills)
	}
	if w.Deaths.Measured != 8 || math.Abs(w.Deaths.Median-20) > epsRange {
		t.Errorf("côté morts = %+v, attendu 8 mesures à 20 m", w.Deaths)
	}
	// Dénivelé BRUT +2 des deux côtés : « d'en haut » quand je frague, « d'en bas » quand je
	// meurs. C'est l'inversion de point de vue de analysis.signedElevation, et elle doit
	// TRAVERSER le service sans être annulée.
	if math.Abs(w.Kills.AbovePct-100) > epsRange {
		t.Errorf("côté frags AbovePct = %v, attendu 100", w.Kills.AbovePct)
	}
	if math.Abs(w.Deaths.BelowPct-100) > epsRange {
		t.Errorf("côté morts BelowPct = %v, attendu 100 (le point de vue s'inverse)", w.Deaths.BelowPct)
	}

	// Couverture : mesurés depuis les positions, totaux depuis le scope canonique.
	if block.MeasuredKills != 10 || block.TotalKills != 18 {
		t.Errorf("frags = %d mesurés sur %d, attendu 10 sur 18", block.MeasuredKills, block.TotalKills)
	}
	if block.MeasuredDeaths != 8 || block.TotalDeaths != 14 {
		t.Errorf("morts = %d mesurées sur %d, attendu 8 sur 14", block.MeasuredDeaths, block.TotalDeaths)
	}
	if math.Abs(block.MedianKillsM-12) > epsRange || math.Abs(block.MedianDeathsM-20) > epsRange {
		t.Errorf("médianes globales = %v / %v, attendu 12 / 20", block.MedianKillsM, block.MedianDeathsM)
	}

	if block.Opening == nil {
		t.Fatal("bloc d'entame nil alors que 10 entames sont mesurées")
	}
	if block.Opening.MeasuredKills != 10 || block.Opening.N != 10 {
		t.Errorf("entame : mesurées=%d appariées=%d, attendu 10 et 10",
			block.Opening.MeasuredKills, block.Opening.N)
	}
	if math.Abs(block.Opening.MedianM-18) > epsRange {
		t.Errorf("entame médiane = %v, attendu 18", block.Opening.MedianM)
	}
	if math.Abs(block.Opening.DeltaMedianM-(-6)) > epsRange {
		t.Errorf("delta médian = %v, attendu -6 (l'engagement se ferme)", block.Opening.DeltaMedianM)
	}
	if math.Abs(block.Opening.ClosingSharePct-100) > epsRange {
		t.Errorf("part de fermeture = %v, attendu 100", block.Opening.ClosingSharePct)
	}
}

// TestLoadWeaponRange_SansEntame_BlocNilJamaisZero — D5. C'est l'état NOMINAL tant que le
// backfill de kill_openings n'a pas tourné : la portée se publie, l'entame ne s'invente pas.
func TestLoadWeaponRange_SansEntame_BlocNilJamaisZero(t *testing.T) {
	repo := &mockWeaponRangeRepo{kills: wrKills("hinf_br75", analysis.SideKiller, 9, 12, 0)}

	block := wrService(repo).loadWeaponRange(context.Background(), wrCanonRows(1, 9, 5))
	if block == nil {
		t.Fatal("section nil : l'absence d'entame ne doit PAS emporter la portée")
	}
	if block.Opening != nil {
		t.Errorf("bloc d'entame = %+v, attendu nil (un zéro se lirait « engage au contact »)",
			*block.Opening)
	}
}

// TestLoadWeaponRange_EntameEnEchec_LaPorteeSurvit — l'entame est un bonus par-dessus un
// enrichissement : son échec de lecture dégrade le bloc, il ne supprime pas la section.
func TestLoadWeaponRange_EntameEnEchec_LaPorteeSurvit(t *testing.T) {
	repo := &mockWeaponRangeRepo{
		kills:   wrKills("hinf_br75", analysis.SideKiller, 9, 12, 0),
		openErr: errors.New("boom SQL"),
	}

	block := wrService(repo).loadWeaponRange(context.Background(), wrCanonRows(1, 9, 5))
	if block == nil {
		t.Fatal("section nil alors que seule l'entame a échoué")
	}
	if block.Opening != nil {
		t.Errorf("bloc d'entame = %+v, attendu nil", *block.Opening)
	}
}

// TestLoadWeaponRange_SousLeSeuil_ArmesNommeesParCote — D9. Ce qui est écarté est NOMMÉ :
// « 3 armes sous le seuil » n'apprend rien, « Hydra (6) » se lit.
func TestLoadWeaponRange_SousLeSeuil_ArmesNommeesParCote(t *testing.T) {
	kills := wrKills("hinf_br75", analysis.SideKiller, analysis.WeaponRangeMinMeasured, 12, 0)
	kills = append(kills, wrKills("hinf_hydra", analysis.SideKiller, 6, 20, 0)...)
	kills = append(kills, wrKills("hinf_ravager", analysis.SideVictim, 5, 25, 0)...)
	repo := &mockWeaponRangeRepo{kills: kills, labels: map[string]port.WeaponLabel{
		"hinf_hydra":   {Label: "Hydra", LabelEN: "Hydra"},
		"hinf_ravager": {Label: "Ravageur", LabelEN: "Ravager"},
	}}

	block := wrService(repo).loadWeaponRange(context.Background(), wrCanonRows(1, 20, 10))
	if block == nil {
		t.Fatal("section nil")
	}
	if len(block.Weapons) != 1 || block.Weapons[0].WeaponKey != "hinf_br75" {
		t.Fatalf("armes publiées = %+v, attendu la seule au-dessus du seuil", block.Weapons)
	}
	if len(block.BelowThresholdKills) != 1 || block.BelowThresholdKills[0].Label != "Hydra" ||
		block.BelowThresholdKills[0].Measured != 6 {
		t.Errorf("sous le seuil (frags) = %+v, attendu Hydra (6)", block.BelowThresholdKills)
	}
	if len(block.BelowThresholdDeaths) != 1 || block.BelowThresholdDeaths[0].Label != "Ravageur" ||
		block.BelowThresholdDeaths[0].Measured != 5 {
		t.Errorf("sous le seuil (morts) = %+v, attendu Ravageur (5)", block.BelowThresholdDeaths)
	}
	// La médiane globale et la couverture comptent TOUS les frags mesurés, seuil compris :
	// elles décrivent le joueur, pas la sélection publiable.
	if block.MeasuredKills != analysis.WeaponRangeMinMeasured+6 {
		t.Errorf("frags mesurés = %d, attendu %d (l'arme sous le seuil compte)",
			block.MeasuredKills, analysis.WeaponRangeMinMeasured+6)
	}
}

// TestLoadWeaponRange_LibellesNonResolus_LaSectionSurvit — la metadata peut ne pas connaître
// une clé (registre non seedé, arme neuve). Le front retombe sur WeaponKey ; on n'invente
// jamais un nom côté Go.
func TestLoadWeaponRange_LibellesNonResolus_LaSectionSurvit(t *testing.T) {
	repo := &mockWeaponRangeRepo{
		kills:     wrKills("hinf_br75", analysis.SideKiller, 9, 12, 0),
		labelsErr: errors.New("metadata indisponible"),
	}

	block := wrService(repo).loadWeaponRange(context.Background(), wrCanonRows(1, 9, 5))
	if block == nil || len(block.Weapons) != 1 {
		t.Fatalf("section = %+v, attendue peuplée", block)
	}
	if block.Weapons[0].Label != "" || block.Weapons[0].WeaponKey != "hinf_br75" {
		t.Errorf("ligne = %+v, attendu un libellé vide et la clé conservée", block.Weapons[0])
	}
}

// TestLoadWeaponRange_DegradationsSansSection — les cinq chemins qui ne publient rien. Aucun
// ne doit paniquer ni rendre une section vide (une section vide se lit comme un résultat).
func TestLoadWeaponRange_DegradationsSansSection(t *testing.T) {
	cas := []struct {
		nom  string
		svc  *SynthesisService
		rows []canonical.PlayerMatchRow
	}{
		{
			nom: "repo non câblé",
			svc: NewSynthesisService(&mockSynthesisRepo{}).
				WithPlayerMatchesRepo(&mockSynthesisPlayerMatches{}, "halo_infinite", "GT"),
			rows: wrCanonRows(1, 9, 5),
		},
		{
			nom:  "scope vide",
			svc:  wrService(&mockWeaponRangeRepo{kills: wrKills("hinf_br75", analysis.SideKiller, 9, 12, 0)}),
			rows: nil,
		},
		{
			nom:  "capability absente",
			svc:  wrService(&mockWeaponRangeRepo{killsErr: games.ErrCapabilityNotSupported}),
			rows: wrCanonRows(1, 9, 5),
		},
		{
			nom:  "erreur SQL",
			svc:  wrService(&mockWeaponRangeRepo{killsErr: errors.New("boom")}),
			rows: wrCanonRows(1, 9, 5),
		},
		{
			nom:  "scope non décodé (zéro frag mesuré)",
			svc:  wrService(&mockWeaponRangeRepo{}),
			rows: wrCanonRows(1, 9, 5),
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if block := c.svc.loadWeaponRange(context.Background(), c.rows); block != nil {
				t.Errorf("section = %+v, attendu nil", block)
			}
		})
	}
}

// TestLoadWeaponRange_GamertagVide — le filtre exige un désignant de joueur (port.Validate).
// Le service coupe AVANT d'appeler le repo : une lecture rejetée serait un aller-retour et un
// log d'erreur pour rien.
func TestLoadWeaponRange_GamertagVide(t *testing.T) {
	repo := &mockWeaponRangeRepo{kills: wrKills("hinf_br75", analysis.SideKiller, 9, 12, 0)}
	svc := NewSynthesisService(&mockSynthesisRepo{}).
		WithPlayerMatchesRepo(&mockSynthesisPlayerMatches{}, "halo_infinite", "").
		WithWeaponRangeRepo(repo)

	if block := svc.loadWeaponRange(context.Background(), wrCanonRows(1, 9, 5)); block != nil {
		t.Errorf("section = %+v, attendu nil", block)
	}
	if repo.killCalls != 0 {
		t.Errorf("repo appelé %d fois sans gamertag, attendu 0", repo.killCalls)
	}
}

// TestMergeWeaponSides_TriEtCoteUnique — l'ordre de lecture du graphe (D6) : médiane des
// FRAGS croissante, et une arme sans frag publié se range à la médiane de ses MORTS.
func TestMergeWeaponSides_TriEtCoteUnique(t *testing.T) {
	kills := wrKills("hinf_sniper", analysis.SideKiller, 9, 40, 0)                  // frags à 40 m
	kills = append(kills, wrKills("hinf_br75", analysis.SideKiller, 9, 12, 0)...)   // frags à 12 m
	kills = append(kills, wrKills("hinf_shotgun", analysis.SideVictim, 9, 3, 0)...) // morts seules, 3 m
	repo := &mockWeaponRangeRepo{kills: kills}

	block := wrService(repo).loadWeaponRange(context.Background(), wrCanonRows(1, 30, 10))
	if block == nil || len(block.Weapons) != 3 {
		t.Fatalf("armes = %+v, attendu 3 lignes", block)
	}
	ordre := []string{block.Weapons[0].WeaponKey, block.Weapons[1].WeaponKey, block.Weapons[2].WeaponKey}
	attendu := []string{"hinf_shotgun", "hinf_br75", "hinf_sniper"}
	for i := range attendu {
		if ordre[i] != attendu[i] {
			t.Fatalf("ordre = %v, attendu %v (médiane croissante, morts seules incluses)", ordre, attendu)
		}
	}
	if block.Weapons[0].Kills != nil {
		t.Errorf("le fusil à pompe ne porte aucun frag : Kills = %+v, attendu nil", block.Weapons[0].Kills)
	}
	if block.Weapons[0].Deaths == nil {
		t.Error("le fusil à pompe doit porter son côté morts")
	}
}
