package teammates

// teammates_squad_range_test.go — le bloc « roles de portee » de l'Escouade, au niveau
// service (mock de port.MatchRangeRepository).
//
// Ce que ces tests verrouillent :
//
//  1. le roster publie est celui de la page (joueur principal EN TETE, puis les coequipiers)
//     et rien d'autre : un adversaire du lobby sert de referentiel, jamais de ligne ;
//  2. le scope passe au repo porte les matchs filtres ET AllPlayers ;
//  3. l'abscisse suit l'ordre chronologique des matchs (c'est l'axe X du nuage) ;
//  4. repo absent, titre sans positions, lecture en echec ou scope sans mesure rendent nil.

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

const (
	sqrMain    = "xuid(11)"
	sqrCoequip = "xuid(12)"
	sqrAdverse = "xuid(13)"
)

type mockSquadRangeRepo struct {
	read port.MatchRangeRead
	err  error
	vu   port.WeaponRangeFilters
}

func (m *mockSquadRangeRepo) LoadMatchRangeKills(
	_ context.Context, _ string, f port.WeaponRangeFilters,
) (port.MatchRangeRead, error) {
	m.vu = f
	return m.read, m.err
}

func sqrRows() []domain.SquadMatchRow {
	return []domain.SquadMatchRow{
		{MatchID: "m_recent", StartTime: time.Date(2026, 9, 21, 22, 0, 0, 0, time.UTC), MapUI: "Recharge"},
		{MatchID: "m_ancien", StartTime: time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC), MapUI: "Live Fire"},
	}
}

func sqrTeammates() []domain.TeammateRow {
	x := sqrCoequip
	return []domain.TeammateRow{{Gamertag: "Kaya", XUID: &x}}
}

func sqrKill(matchID, killer string, timeMS int64, dist float64) analysis.MeasuredKill {
	return analysis.MeasuredKill{
		MatchID: matchID, KillerXUID: killer, TimeMS: timeMS,
		Side: analysis.SideKiller, DistanceM: dist,
	}
}

func sqrService(repo port.MatchRangeRepository) *TeammatesService {
	return (&TeammatesService{titleSlug: "halo_infinite", gamertag: "JGtm"}).WithMatchRange(repo)
}

// TestSquadRange_RosterPublieEtLobbyReferentiel : LE test du cablage.
func TestSquadRange_RosterPublieEtLobbyReferentiel(t *testing.T) {
	repo := &mockSquadRangeRepo{read: port.MatchRangeRead{
		Kills: []analysis.MeasuredKill{
			// m_ancien : main a 10 m, coequipier a 30 m, adversaire a 20 m -> lobby 20 m.
			sqrKill("m_ancien", sqrMain, 1, 10),
			sqrKill("m_ancien", sqrCoequip, 2, 30),
			sqrKill("m_ancien", sqrAdverse, 3, 20),
			// m_recent : le seul main.
			sqrKill("m_recent", sqrMain, 4, 15),
		},
		KillsTotal: 8,
	}}
	block := sqrService(repo).buildSquadRange(
		context.Background(), sqrRows(), "JGtm", sqrMain, sqrTeammates())
	if block == nil {
		t.Fatal("bloc nil, want un bloc")
	}
	if !repo.vu.AllPlayers || len(repo.vu.MatchIDs) != 2 {
		t.Fatalf("filtres = %+v, want les 2 matchs et AllPlayers", repo.vu)
	}
	if len(block.Profiles) != 2 {
		t.Fatalf("profils = %d, want 2", len(block.Profiles))
	}
	// L'abscisse du nuage : du plus ancien au plus recent.
	if block.Profiles[0].MatchID != "m_ancien" || block.Profiles[1].MatchID != "m_recent" {
		t.Fatalf("ordre = %s puis %s, want m_ancien puis m_recent",
			block.Profiles[0].MatchID, block.Profiles[1].MatchID)
	}
	if block.Profiles[0].MapName != "Live Fire" {
		t.Errorf("MapName = %q, want le libelle d'affichage", block.Profiles[0].MapName)
	}

	anc := block.Profiles[0]
	// Le referentiel porte les TROIS frags, adversaire compris.
	if anc.LobbyMeasured != 3 || math.Abs(anc.LobbyMedianM-20) > 1e-9 {
		t.Errorf("lobby = %d frags / %v m, want 3 / 20", anc.LobbyMeasured, anc.LobbyMedianM)
	}
	// Le roster publie : le joueur principal EN TETE, puis le coequipier. L'adversaire,
	// jamais.
	if len(anc.Players) != 2 {
		t.Fatalf("joueurs = %+v, want main + coequipier", anc.Players)
	}
	if anc.Players[0].XUID != sqrMain || anc.Players[0].Gamertag != "JGtm" {
		t.Errorf("joueur[0] = %+v, want le joueur principal en tete", anc.Players[0])
	}
	if anc.Players[1].XUID != sqrCoequip || math.Abs(anc.Players[1].LobbyDeltaM-10) > 1e-9 {
		t.Errorf("joueur[1] = %+v, want Kaya a +10 m du lobby", anc.Players[1])
	}
	// Le coequipier n'a aucun frag mesure sur m_recent : il y est ABSENT, jamais a zero.
	if len(block.Profiles[1].Players) != 1 {
		t.Errorf("m_recent joueurs = %+v, want le seul main", block.Profiles[1].Players)
	}
	if block.KillsMeasured != 4 || block.KillsTotal != 8 {
		t.Errorf("couverture = %d/%d, want 4/8", block.KillsMeasured, block.KillsTotal)
	}
}

// TestSquadRange_OmissionsSontNil : tous les chemins qui ne publient rien.
func TestSquadRange_OmissionsSontNil(t *testing.T) {
	cas := []struct {
		nom  string
		svc  *TeammatesService
		rows []domain.SquadMatchRow
	}{
		{"repo non cable", sqrService(nil), sqrRows()},
		{"perimetre vide", sqrService(&mockSquadRangeRepo{}), nil},
		{"titre sans positions", sqrService(
			&mockSquadRangeRepo{err: games.ErrCapabilityNotSupported}), sqrRows()},
		{"lecture en echec", sqrService(
			&mockSquadRangeRepo{err: errors.New("boom")}), sqrRows()},
		{"aucun frag mesure", sqrService(&mockSquadRangeRepo{}), sqrRows()},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got := c.svc.buildSquadRange(
				context.Background(), c.rows, "JGtm", sqrMain, sqrTeammates())
			if got != nil {
				t.Fatalf("bloc = %+v, want nil (omission, jamais un bloc vide)", got)
			}
		})
	}
}
