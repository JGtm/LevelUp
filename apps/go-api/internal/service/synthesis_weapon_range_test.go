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
//  4. les armes sous le seuil sont NOMMÉES et ventilées par côté, et quand elles le sont
//     TOUTES la section reste PRÉSENTE avec zéro arme (les deux médianes et les couvertures
//     restent l'information) ;
//  5. capability absente / repo nil / scope vide / erreur SQL : section absente, jamais de
//     panique et jamais une page cassée — et un match canonique SANS compteur ne fait
//     paniquer ni le scope ni la page ;
//  6. le régime de journalisation : capability absente en DEBUG, panne en WARN. Un WARN
//     permanent sur un titre sans décodeur noierait les vraies pannes.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
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

// wrCanonRowSansCompteur : un match canonique SANS compteur de frags ni de morts (les deux
// pointeurs nils). C'est un état réel : une ligne canonique dont l'API n'a pas rendu le
// scoreboard porte un Self vide, et le scope de la Synthèse la contient comme les autres.
func wrCanonRowSansCompteur(matchID string) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{Summary: canonical.MatchSummary{MatchID: matchID}}
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
	if block.Opening.Delta == nil {
		t.Fatal("sous-bloc delta nil alors que les 10 frags sont appariés")
	}
	if block.Opening.MeasuredKills != 10 || block.Opening.Delta.N != 10 {
		t.Errorf("entame : mesurées=%d appariées=%d, attendu 10 et 10",
			block.Opening.MeasuredKills, block.Opening.Delta.N)
	}
	if math.Abs(block.Opening.MedianM-18) > epsRange {
		t.Errorf("entame médiane = %v, attendu 18", block.Opening.MedianM)
	}
	if math.Abs(block.Opening.Delta.MedianM-(-6)) > epsRange {
		t.Errorf("delta médian = %v, attendu -6 (l'engagement se ferme)", block.Opening.Delta.MedianM)
	}
	if math.Abs(block.Opening.Delta.ClosingSharePct-100) > epsRange {
		t.Errorf("part de fermeture = %v, attendu 100", block.Opening.Delta.ClosingSharePct)
	}
}

// TestLoadWeaponRange_EntamesSansCoupFatalMesure_DeltaOmis — le sous-bloc `delta` est NIL
// quand aucun frag ne porte les DEUX mesures (constat F9, revue adversariale du lot 4,
// 2026-09-06).
//
// LE CAS EST ATTEIGNABLE : `kill_positions` et `kill_openings` s'écrivent sous deux leases
// indépendants, et un scope peut porter des entames dont aucun coup fatal n'est placé. À plat,
// il publiait `closing_share_pct: 0` — un champ requis, donc toujours présent — qui se lit
// « ce joueur ne ferme jamais la distance » alors qu'aucune mesure ne le dit. La couverture de
// l'entame, elle, reste publiée : c'est un fait mesuré.
func TestLoadWeaponRange_EntamesSansCoupFatalMesure_DeltaOmis(t *testing.T) {
	kills := wrKills("hinf_br75", analysis.SideKiller, 9, 12, 0)
	// Les entames portent des instants qu'AUCUN coup fatal mesuré ne porte : rien n'apparie.
	openings := wrKills("hinf_br75", analysis.SideKiller, 9, 30, 0)
	for i := range openings {
		openings[i].TimeMS += 500000
	}
	repo := &mockWeaponRangeRepo{kills: kills, openings: openings}

	block := wrService(repo).loadWeaponRange(context.Background(), wrCanonRows(1, 9, 5))
	if block == nil || block.Opening == nil {
		t.Fatalf("bloc d'entame absent : %+v — 9 entames sont pourtant mesurées", block)
	}
	if block.Opening.MeasuredKills != 9 || math.Abs(block.Opening.MedianM-30) > epsRange {
		t.Errorf("entame = %d mesures à %v m, attendu 9 à 30 m (la couverture reste publiée)",
			block.Opening.MeasuredKills, block.Opening.MedianM)
	}
	if block.Opening.Delta != nil {
		t.Errorf("sous-bloc delta = %+v, attendu nil : aucun frag ne porte les deux mesures, "+
			"et un zéro publié se lirait « la distance ne bouge jamais »", *block.Opening.Delta)
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

// TestLoadWeaponRange_MatchSansCompteur_NiPaniqueNiZeroCompte — les gardes `Self.Kills != nil`
// / `Self.Deaths != nil` de `weaponRangeScope`.
//
// POURQUOI CE TEST EXISTE (constat F2, revue adversariale du lot 4, 2026-09-06) : les deux
// gardes n'étaient épinglées par rien — les retirer laissait toute la suite verte, alors qu'un
// seul match canonique sans scoreboard ferait paniquer le chemin PRINCIPAL de
// `GetSynthesisPage`, sans recover, donc en 500 sur la page ENTIÈRE (pas seulement sur la
// section). Le match sans compteur reste dans le SCOPE (ses frags mesurés comptent, ils
// viennent de la table de positions) mais n'ajoute rien aux TOTAUX : l'absence de donnée n'est
// pas une performance nulle.
//
// MUTATION : retirer l'une des deux gardes -> panique de déréférencement, ce test rouge.
func TestLoadWeaponRange_MatchSansCompteur_NiPaniqueNiZeroCompte(t *testing.T) {
	rows := append(wrCanonRows(1, 9, 5), wrCanonRowSansCompteur("m_sans_scoreboard"))
	repo := &mockWeaponRangeRepo{kills: wrKills("hinf_br75", analysis.SideKiller, 9, 12, 0)}

	block := wrService(repo).loadWeaponRange(context.Background(), rows)
	if block == nil {
		t.Fatal("section nil : un match sans compteur ne doit pas emporter la section")
	}
	// Le match sans compteur est dans le scope lu...
	if len(repo.lastFilters.MatchIDs) != 2 {
		t.Errorf("scope = %d match(s), attendu 2 (le match sans compteur est lu comme les autres)",
			len(repo.lastFilters.MatchIDs))
	}
	// ...mais il ne pèse pas sur les dénominateurs de couverture.
	if block.TotalKills != 9 || block.TotalDeaths != 5 {
		t.Errorf("totaux = %d frags / %d morts, attendu 9 et 5 (le match sans compteur "+
			"n'ajoute rien, il n'ajoute pas non plus zéro)", block.TotalKills, block.TotalDeaths)
	}
	if block.MeasuredKills != 9 {
		t.Errorf("frags mesurés = %d, attendu 9", block.MeasuredKills)
	}
}

// TestLoadWeaponRange_ToutSousLeSeuil_SectionPresenteAvecZeroArme — le cas « des mesures, mais
// TOUTES sous le seuil » (constat F3, revue adversariale du lot 4 ; décision du pilote,
// 2026-09-06).
//
// LA SECTION EST PRÉSENTE, AVEC UNE LISTE D'ARMES VIDE. Ce n'est pas une section vide : les
// deux médianes, les deux couvertures et les listes NOMMÉES d'armes écartées portent toute
// l'information — « tu as tué à 10 m en médiane, sur 7 frags mesurés, répartis sur des armes
// trop peu jouées pour qu'un bâton p10-p90 veuille dire quelque chose ». La faire disparaître
// se lirait « aucune mesure », ce qui est faux.
func TestLoadWeaponRange_ToutSousLeSeuil_SectionPresenteAvecZeroArme(t *testing.T) {
	kills := wrKills("hinf_hydra", analysis.SideKiller, 4, 10, 0)
	kills = append(kills, wrKills("hinf_ravager", analysis.SideKiller, 3, 20, 0)...)
	kills = append(kills, wrKills("hinf_shotgun", analysis.SideVictim, 2, 3, 0)...)
	repo := &mockWeaponRangeRepo{kills: kills}

	block := wrService(repo).loadWeaponRange(context.Background(), wrCanonRows(1, 20, 9))
	if block == nil {
		t.Fatal("section nil alors que 9 frags sont mesurés : la couverture et les médianes " +
			"restent publiables, seul le graphe par arme est vide")
	}
	if block.Weapons == nil {
		t.Error("Weapons = nil, attendu une liste VIDE : le contrat sérialise `weapons: []`, " +
			"jamais `null` (le front itère sans garde)")
	}
	if len(block.Weapons) != 0 {
		t.Errorf("armes publiées = %+v, attendu aucune (toutes sont sous le seuil)", block.Weapons)
	}
	// Les médianes et la couverture sont calculées sur TOUS les frags mesurés, seuil compris.
	if math.Abs(block.MedianKillsM-10) > epsRange {
		t.Errorf("médiane des frags = %v, attendu 10 (médiane de 4x10 m et 3x20 m)", block.MedianKillsM)
	}
	if math.Abs(block.MedianDeathsM-3) > epsRange {
		t.Errorf("médiane des morts = %v, attendu 3", block.MedianDeathsM)
	}
	if block.MeasuredKills != 7 || block.TotalKills != 20 {
		t.Errorf("couverture frags = %d/%d, attendu 7/20", block.MeasuredKills, block.TotalKills)
	}
	if block.MeasuredDeaths != 2 || block.TotalDeaths != 9 {
		t.Errorf("couverture morts = %d/%d, attendu 2/9", block.MeasuredDeaths, block.TotalDeaths)
	}
	// Ce qui est écarté est NOMMÉ et ventilé par côté (D9) — c'est ce qui rend la section utile.
	if len(block.BelowThresholdKills) != 2 || len(block.BelowThresholdDeaths) != 1 {
		t.Errorf("sous le seuil = %d frags / %d morts, attendu 2 et 1",
			len(block.BelowThresholdKills), len(block.BelowThresholdDeaths))
	}
}

// TestMergeWeaponSides_LaMedianeDesFragsPrimeSurCelleDesMorts — la règle de tri de
// `mergeWeaponSides` (constat F4, revue adversariale du lot 4, 2026-09-06).
//
// Une arme mesurée DES DEUX CÔTÉS se range à la médiane de ses FRAGS, jamais à celle de ses
// morts : le graphe se lit « où je frague », les morts en sont le contrepoint. La règle
// n'était épinglée par rien — le témoin `if out[i].Kills == nil` remplacé par `if true`
// restait vert, et le tri basculait silencieusement sur le dernier côté rencontré.
//
// FIXTURE CONSTRUITE POUR QUE LA MUTATION SE VOIE : le BR75 frague à 12 m et tue son porteur à
// 20 m, l'Hydra ne frague qu'à 15 m. Par la médiane des frags -> BR75 (12) puis Hydra (15) ;
// par celle des morts -> Hydra (15) puis BR75 (20). L'ordre s'inverse.
func TestMergeWeaponSides_LaMedianeDesFragsPrimeSurCelleDesMorts(t *testing.T) {
	kills := wrKills("hinf_br75", analysis.SideKiller, 9, 12, 0)
	kills = append(kills, wrKills("hinf_br75", analysis.SideVictim, 9, 20, 0)...)
	kills = append(kills, wrKills("hinf_hydra", analysis.SideKiller, 9, 15, 0)...)
	repo := &mockWeaponRangeRepo{kills: kills}

	block := wrService(repo).loadWeaponRange(context.Background(), wrCanonRows(1, 30, 12))
	if block == nil || len(block.Weapons) != 2 {
		t.Fatalf("armes = %+v, attendu 2 lignes", block)
	}
	if block.Weapons[0].WeaponKey != "hinf_br75" || block.Weapons[1].WeaponKey != "hinf_hydra" {
		t.Fatalf("ordre = [%s %s], attendu [hinf_br75 hinf_hydra] : le BR75 se range à la "+
			"médiane de ses FRAGS (12 m), pas à celle de ses morts (20 m)",
			block.Weapons[0].WeaponKey, block.Weapons[1].WeaponKey)
	}
	if block.Weapons[0].Kills == nil || block.Weapons[0].Deaths == nil {
		t.Errorf("le BR75 doit porter ses deux côtés : %+v", block.Weapons[0])
	}
}

// TestLogWeaponRangeFailure_RegimeDesNiveaux — capability absente -> DEBUG, tout le reste ->
// WARN (constat F6, revue adversariale du lot 4, 2026-09-06).
//
// POURQUOI C'EST UN TEST ET PAS UN COMMENTAIRE : inverser la condition laissait toute la suite
// verte. Or le régime a une conséquence d'exploitation directe — un titre sans décodeur de
// film émettrait un WARN à CHAQUE lecture de Synthèse, et ce bruit permanent noierait les
// vraies pannes SQL, qui sont exactement ce que ce log doit faire remonter.
//
// Pas de `t.Parallel` : le test remplace le logger PAR DÉFAUT du process.
func TestLogWeaponRangeFailure_RegimeDesNiveaux(t *testing.T) {
	var buf threadSafeBuffer
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	cas := []struct {
		nom      string
		err      error
		niveau   string
		veutWarn bool
	}{
		{"capability absente -> DEBUG", games.ErrCapabilityNotSupported, "DEBUG", false},
		{"capability absente emballee -> DEBUG",
			fmt.Errorf("LoadWeaponRange: %w", games.ErrCapabilityNotSupported), "DEBUG", false},
		{"erreur SQL -> WARN", errors.New("Catalog Error: table does not exist"), "WARN", true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			buf.Reset()
			repo := &mockWeaponRangeRepo{killsErr: c.err}
			block := wrService(repo).loadWeaponRange(context.Background(), wrCanonRows(1, 9, 5))
			if block != nil {
				t.Fatalf("section = %+v, attendu nil", block)
			}
			out := buf.String()
			if !strings.Contains(out, `"level":"`+c.niveau+`"`) {
				t.Errorf("aucun log de niveau %s emis :\n%s", c.niveau, out)
			}
			if aWarn := strings.Contains(out, `"level":"WARN"`); aWarn != c.veutWarn {
				t.Errorf("presence d'un WARN = %v, attendu %v — un titre sans decodeur ne doit "+
					"pas alerter, une panne SQL doit alerter :\n%s", aWarn, c.veutWarn, out)
			}
		})
	}
}
