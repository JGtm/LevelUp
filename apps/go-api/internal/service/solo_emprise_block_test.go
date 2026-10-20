package service

// solo_emprise_block_test.go — L'ASSEMBLAGE DE L'EMPRISE avec une liste de joueurs fournie (Vue match :
// les joueurs de l'équipe, dans l'ordre de la page) : les fiches sont CETTE liste, dans cet ordre, et
// les prises de chaque objet se partagent entre ces joueurs et le reste de l'équipe. Sans liste,
// l'assemblage garde le joueur seul (Séries temporelles, Sessions : leurs propres tests).

import (
	"context"
	"fmt"
	"testing"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
)

func TestBuildSoloEmpriseBlock_JoueursFournis(t *testing.T) {
	usage := usageTestRepoMock()
	usage.padTiers = []sessionusage.PadTierRow{
		{MatchID: "m1", XUID: "A", Tier: domain.PadTierPower, WeaponFamily: "sniper", Pickups: 2, PadsTotal: 2, PadsConfirmed: 2},
		{MatchID: "m1", XUID: "P", Tier: domain.PadTierPower, WeaponFamily: "sniper", Pickups: 1, PadsTotal: 2, PadsConfirmed: 2},
	}
	joueurs := []domain.SessionUsageSquadPlayer{{XUID: "P", Gamertag: "Papa"}, {XUID: "A", Gamertag: "Alpha"}}
	b := buildSoloEmpriseBlock(context.Background(), soloEmpriseQuery{
		Page: "match_view", Player: "Papa", PlayerXUID: "P", Locale: "fr",
		Current: []squademprise.Match{{MatchID: "m1"}}, UsageRepo: usage, Players: joueurs,
	})
	if fmt.Sprint(b.Players) != fmt.Sprint(joueurs) {
		t.Fatalf("fiches = %+v, attendu %+v dans cet ordre", b.Players, joueurs)
	}
	var sniper *domain.SquadEmpriseObject
	for i := range b.Objects {
		if b.Objects[i].Key == "sniper" {
			sniper = &b.Objects[i]
		}
	}
	if sniper == nil || len(sniper.Squad) != 3 {
		t.Fatalf("objet sniper = %+v, attendu trois parts (P, A, reste de l'équipe)", sniper)
	}
	if sniper.Squad[0].XUID != "P" || sniper.Squad[0].Taken != 1 || sniper.Squad[1].XUID != "A" || sniper.Squad[1].Taken != 2 {
		t.Errorf("parts = %+v, attendu P 1, A 2", sniper.Squad)
	}
}

// pendingFixe — un résolveur d'analyse en cours à table fixe.
type pendingFixe struct{ table map[string][]string }

func (p pendingFixe) ResolvePending(_ context.Context, _ []string) map[string][]string {
	return p.table
}

// Le bloc dit, pour son périmètre, combien de matchs sans résumé d'usage ont une analyse EN COURS et combien
// n'en ont pas (Sessions et Séries temporelles passent par cet assemblage).
func TestBuildSoloEmpriseBlock_DitLesMatchsSansFilm(t *testing.T) {
	usage := usageTestRepoMock()
	usage.padTiers = []sessionusage.PadTierRow{
		{MatchID: "m1", XUID: "P", Tier: domain.PadTierPower, WeaponFamily: "sniper", Pickups: 1, PadsTotal: 2, PadsConfirmed: 2},
	}
	q := soloEmpriseQuery{
		Page: "sessions", Player: "Papa", PlayerXUID: "P", Locale: "fr",
		Current:   []squademprise.Match{{MatchID: "m1"}, {MatchID: "m2"}, {MatchID: "m3"}},
		UsageRepo: usage,
		Pending:   pendingFixe{table: map[string][]string{"m2": {domain.AnalysisPartKillSource}}},
	}
	b := buildSoloEmpriseBlock(context.Background(), q)
	if b.MatchesFilmPending != 1 || b.MatchesFilmMissing != 1 {
		t.Fatalf("en cours=%d manquant=%d, attendu 1 et 1 (m1 lu, m2 en cours, m3 ni lu ni en cours)",
			b.MatchesFilmPending, b.MatchesFilmMissing)
	}

	q.Pending = nil
	b = buildSoloEmpriseBlock(context.Background(), q)
	if b.MatchesFilmPending != 0 || b.MatchesFilmMissing != 2 {
		t.Fatalf("sans résolveur : en cours=%d manquant=%d, attendu 0 et 2", b.MatchesFilmPending, b.MatchesFilmMissing)
	}
}
