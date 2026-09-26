package service

// session_page_range_test.go — le bloc « portée des engagements » de la session, au niveau
// service (mock de port.MatchRangeRepository).
//
// Ce que ces tests verrouillent :
//
//  1. le scope passé au repo porte MatchIDs ET AllPlayers — sans AllPlayers, le lecteur
//     retomberait sur le seul joueur consulté et la médiane du lobby n'aurait plus de sens ;
//  2. SEUL le joueur consulté est publié dans Players, mais LobbyMedianM porte tout le lobby ;
//  3. une capability absente, une lecture en échec ou un scope sans mesure rendent nil —
//     une OMISSION, jamais un bloc vide qui se lirait comme une mesure à zéro.

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

const (
	sprMain  = "xuid(1)"
	sprAutre = "xuid(2)"
	sprMatch = "m_session_portee"
)

// mockMatchRangeRepo : le repo de portée, avec la mémoire du filtre reçu (c'est lui qu'on
// vérifie autant que le résultat).
type mockMatchRangeRepo struct {
	read port.MatchRangeRead
	err  error
	vu   port.WeaponRangeFilters
}

func (m *mockMatchRangeRepo) LoadMatchRangeKills(
	_ context.Context, _ string, f port.WeaponRangeFilters,
) (port.MatchRangeRead, error) {
	m.vu = f
	return m.read, m.err
}

func sprService(repo port.MatchRangeRepository) *SessionPageService {
	return (&SessionPageService{gamertag: "JGtm", titleSlug: "halo_infinite"}).
		WithMatchRange(repo, sprMain)
}

func sprMatches() []legacymatch.StatsMatchRow {
	return []legacymatch.StatsMatchRow{{
		MatchID:   sprMatch,
		StartTime: time.Date(2026, 9, 21, 21, 0, 0, 0, time.UTC),
		MapName:   "Live Fire",
		MapNameFR: "Tir réel",
	}}
}

func sprKill(killer string, timeMS int64, dist float64) analysis.MeasuredKill {
	return analysis.MeasuredKill{
		MatchID: sprMatch, KillerXUID: killer, TimeMS: timeMS,
		Side: analysis.SideKiller, DistanceM: dist,
	}
}

// TestSessionRange_JoueurConsulteSeulMaisLobbyEntier : LE test du câblage.
func TestSessionRange_JoueurConsulteSeulMaisLobbyEntier(t *testing.T) {
	repo := &mockMatchRangeRepo{read: port.MatchRangeRead{
		Kills: []analysis.MeasuredKill{
			sprKill(sprMain, 1, 10), sprKill(sprMain, 2, 10),
			sprKill(sprAutre, 3, 30), sprKill(sprAutre, 4, 30),
		},
		KillsTotal: 6,
	}}
	block := sprService(repo).buildSessionRange(context.Background(), sprMatches(), "session")
	if block == nil {
		t.Fatal("bloc nil, want un bloc")
	}
	// Le filtre : borné par les matchs ET sans désignant de joueur.
	if !repo.vu.AllPlayers || len(repo.vu.MatchIDs) != 1 || repo.vu.MatchIDs[0] != sprMatch {
		t.Fatalf("filtres = %+v, want MatchIDs=[%s] et AllPlayers", repo.vu, sprMatch)
	}
	if repo.vu.Gamertag != "" || len(repo.vu.XUIDs) != 0 {
		t.Errorf("filtres = %+v, want AUCUN designant de joueur", repo.vu)
	}
	if len(block.Profiles) != 1 {
		t.Fatalf("profils = %d, want 1", len(block.Profiles))
	}
	p := block.Profiles[0]
	// Un seul joueur publié...
	if len(p.Players) != 1 || p.Players[0].XUID != sprMain || p.Players[0].Gamertag != "JGtm" {
		t.Fatalf("joueurs = %+v, want le seul joueur consulte", p.Players)
	}
	// ... mais le référentiel porte les QUATRE frags du lobby (médiane 10,10,30,30 = 20).
	if p.LobbyMeasured != 4 || math.Abs(p.LobbyMedianM-20) > 1e-9 {
		t.Errorf("lobby = %d frags / %v m, want 4 / 20", p.LobbyMeasured, p.LobbyMedianM)
	}
	if math.Abs(p.Players[0].LobbyDeltaM-(-10)) > 1e-9 {
		t.Errorf("LobbyDeltaM = %v, want -10", p.Players[0].LobbyDeltaM)
	}
	if p.MapName != "Tir réel" {
		t.Errorf("MapName = %q, want la traduction FR", p.MapName)
	}
	if block.KillsMeasured != 4 || block.KillsTotal != 6 {
		t.Errorf("couverture = %d/%d, want 4/6", block.KillsMeasured, block.KillsTotal)
	}
}

// TestSessionRange_OmissionsSontNil : les quatre chemins qui ne publient rien.
func TestSessionRange_OmissionsSontNil(t *testing.T) {
	cas := []struct {
		nom  string
		svc  *SessionPageService
		rows []legacymatch.StatsMatchRow
	}{
		{"repo non cable", (&SessionPageService{}).WithMatchRange(nil, sprMain), sprMatches()},
		{"xuid inconnu", sprService(&mockMatchRangeRepo{}).WithMatchRange(&mockMatchRangeRepo{}, ""), sprMatches()},
		{"session sans match", sprService(&mockMatchRangeRepo{}), nil},
		{"titre sans positions", sprService(&mockMatchRangeRepo{err: games.ErrCapabilityNotSupported}), sprMatches()},
		{"lecture en echec", sprService(&mockMatchRangeRepo{err: errors.New("boom")}), sprMatches()},
		{"aucun frag mesure", sprService(&mockMatchRangeRepo{}), sprMatches()},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := c.svc.buildSessionRange(context.Background(), c.rows, "session"); got != nil {
				t.Fatalf("bloc = %+v, want nil (omission, jamais un bloc vide)", got)
			}
		})
	}
}
