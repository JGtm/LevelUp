package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

// TestPrepareMatchViewEvents_DureeDeJeu : la durée qui découpe la Dominance est LA durée de jeu de
// la chronologie du match (durée du registre moins le coup d'envoi), la même que celle de
// l'en-tête. Coup d'envoi inconnu (Halo 5, estimation refusée) : la chronologie part du début du
// match ; durée absente : 0, jamais une valeur de remplacement.
func TestPrepareMatchViewEvents_DureeDeJeu(t *testing.T) {
	t.Parallel()

	ptrI64 := func(v int64) *int64 { return &v }
	ptrF64 := func(v float64) *float64 { return &v }

	cases := []struct {
		name string
		meta *domain.MatchMetaRaw
		want int64
	}{
		{
			name: "coup d'envoi connu : durée moins coup d'envoi, à la milliseconde",
			meta: &domain.MatchMetaRaw{DurationSeconds: ptrF64(600), T0Ms: ptrI64(26304)},
			want: 600_000 - 26_304,
		},
		{
			name: "coup d'envoi inconnu : durée du match",
			meta: &domain.MatchMetaRaw{DurationSeconds: ptrF64(480)},
			want: 480_000,
		},
		{
			name: "coup d'envoi au-delà de la durée : borné à zéro",
			meta: &domain.MatchMetaRaw{DurationSeconds: ptrF64(20), T0Ms: ptrI64(28_000)},
			want: 0,
		},
		{
			name: "durée absente : zéro, pas de Dominance fabriquée",
			meta: &domain.MatchMetaRaw{T0Ms: ptrI64(15_000)},
			want: 0,
		},
		{name: "meta nil : zéro", meta: nil, want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := prepareMatchViewEvents(&matchViewData{}, "m1", tc.meta).GameplayDurationMs(); got != tc.want {
				t.Errorf("durée de jeu = %d ms, attendu %d", got, tc.want)
			}
		})
	}
}

// L'en-tête sert la durée de jeu de la même chronologie, en secondes tronquées, et le coup
// d'envoi tel quel ; sans durée au registre il ne sert aucune durée.
func TestBuildMatchHeader_DureeDeJeuDeLaChronologie(t *testing.T) {
	t.Parallel()
	d, t0 := 600.0, int64(26304)
	tl, ok := matchViewTimeline(&domain.MatchMetaRaw{DurationSeconds: &d, T0Ms: &t0})
	if !ok || tl.GameplayDurationSeconds() != 573 || tl.T0Ms != 26304 {
		t.Fatalf("chronologie = %+v ok=%v, attendu 573 s de jeu et un coup d'envoi de 26304 ms", tl, ok)
	}
	if _, ok := matchViewTimeline(&domain.MatchMetaRaw{T0Ms: &t0}); ok {
		t.Fatal("sans durée au registre, aucune durée de jeu ne doit être servie")
	}
}

// La Dominance bine des paires killer→victim restées sur l'horloge BRUTE du film : ses bins couvrent
// la durée du match, countdown compris. Découpée sur la durée de jeu, elle écraserait dans son dernier
// bin les frags des dernières secondes (le temps du coup d'envoi). Un frag à 500 s sur l'horloge du film,
// pour un match de 508 s dont le coup d'envoi tombe à 43 s, garde son bin.
func TestBuildCombatTabFull_DominanceCouvreLaDureeDuMatch(t *testing.T) {
	t.Parallel()
	me, enemy := "ME", "EN"
	kvPairs := []domain.KVPairRaw{
		{KillerXUID: me, VictimXUID: enemy, TimeMS: 100_000, KillCount: 1},
		{KillerXUID: me, VictimXUID: enemy, TimeMS: 500_000, KillCount: 1},
	}
	scoreboard := []domain.ScoreboardRaw{
		{XUID: me, Kills: 2, OutcomeCode: 2, TeamID: intPtr(0)},
		{XUID: enemy, Deaths: 2, OutcomeCode: 3, TeamID: intPtr(1)},
	}
	t0 := int64(43_166)
	tl := timeline.ForMatch(time.Time{}, 508_000, &t0)
	tab := buildCombatTabFull(combatTabScope{matchID: "m1", myXUID: me, timeline: tl, locale: i18n.Default}, nil, nil, kvPairs, scoreboard)
	last := tab.TugOfWar[len(tab.TugOfWar)-1]
	if last.BinEnd != 510 {
		t.Fatalf("dernier bin de la Dominance finit à %d s, attendu 510 (508 s de match)", last.BinEnd)
	}
	var in500 int
	for _, b := range tab.TugOfWar {
		if b.BinStart <= 500 && 500 < b.BinEnd {
			in500 = b.TeamKills
		}
	}
	if in500 != 1 {
		t.Fatalf("le frag à 500 s n'est pas dans son propre bin : %+v", tab.TugOfWar)
	}
}
