package service

// timeseries_range_test.go — le nuage des rôles de portée de la page Séries temporelles
// (lot U, décision D23-a), au niveau service.
//
// Ce que ces tests verrouillent :
//
//  1. la lecture porte AllPlayers (le lobby entier fait la médiane de référence) mais SEUL
//     le joueur consulté est publié dans Players ;
//  2. le scope est ordonné du plus ancien au plus récent — l'axe des x du nuage ;
//  3. xuid inconnu ou repo non câblé ⇒ champ omis, jamais un bloc vide.

import (
	"context"
	"math"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

func tsrRow(id string, t0 time.Time) canonical.PlayerMatchRow {
	return canonical.PlayerMatchRow{Summary: canonical.MatchSummary{
		MatchID:      id,
		StartedAtUTC: t0,
		Map:          &canonical.AssetReference{DefaultLabel: "Live Fire"},
	}}
}

func tsrService(repo port.MatchRangeRepository, xuid string) *TimeseriesService {
	svc := &TimeseriesService{titleSlug: "halo_infinite", gamertag: "JGtm"}
	return svc.WithMatchRange(repo, xuid)
}

func TestTimeseriesRange_JoueurConsulteSeulMaisLobbyEntier(t *testing.T) {
	base := time.Date(2026, 9, 20, 18, 0, 0, 0, time.UTC)
	repo := &mockMatchRangeRepo{read: portMatchRangeReadDeuxJoueurs()}
	var resp domain.TimeseriesPageResponse
	// Rows donnés dans le DÉSORDRE : le scope doit les remettre en ordre.
	rows := []canonical.PlayerMatchRow{
		tsrRow("m_b", base.Add(time.Hour)),
		tsrRow("m_a", base),
	}
	tsrService(repo, sprMain).attachMatchRange(context.Background(), &resp, rows, "fr")

	if resp.RangeProfiles == nil {
		t.Fatal("range_profiles nil, want un bloc")
	}
	if !repo.vu.AllPlayers || len(repo.vu.MatchIDs) != 2 {
		t.Fatalf("filtres = %+v, want les deux matchs et AllPlayers", repo.vu)
	}
	if repo.vu.MatchIDs[0] != "m_a" || repo.vu.MatchIDs[1] != "m_b" {
		t.Errorf("scope = %v, want du plus ancien au plus recent", repo.vu.MatchIDs)
	}
	p := resp.RangeProfiles.Profiles[0]
	if len(p.Players) != 1 || p.Players[0].XUID != sprMain {
		t.Fatalf("joueurs = %+v, want le seul joueur consulte", p.Players)
	}
	if p.LobbyMeasured != 4 || math.Abs(p.LobbyMedianM-20) > 1e-9 {
		t.Errorf("lobby = %d frags / %v m, want 4 / 20", p.LobbyMeasured, p.LobbyMedianM)
	}
	if p.MapName != "Live Fire" {
		t.Errorf("MapName = %q, want le libelle du canonique", p.MapName)
	}
}

func TestTimeseriesRange_OmisSansJoueurNiRepo(t *testing.T) {
	rows := []canonical.PlayerMatchRow{tsrRow("m_a", time.Now().UTC())}
	for nom, svc := range map[string]*TimeseriesService{
		"xuid inconnu":   tsrService(&mockMatchRangeRepo{read: portMatchRangeReadDeuxJoueurs()}, ""),
		"repo non cable": tsrService(nil, sprMain),
	} {
		t.Run(nom, func(t *testing.T) {
			var resp domain.TimeseriesPageResponse
			svc.attachMatchRange(context.Background(), &resp, rows, "fr")
			if resp.RangeProfiles != nil {
				t.Fatalf("range_profiles = %+v, want nil", resp.RangeProfiles)
			}
		})
	}
}

// portMatchRangeReadDeuxJoueurs : deux frags du joueur consulté à 10 m et deux d'un
// adversaire à 30 m, sur CHAQUE match que le mock rend (m_a et m_b).
func portMatchRangeReadDeuxJoueurs() port.MatchRangeRead {
	kills := make([]analysis.MeasuredKill, 0, 8)
	for _, id := range []string{"m_a", "m_b"} {
		for i := 0; i < 2; i++ {
			kills = append(kills,
				analysis.MeasuredKill{MatchID: id, KillerXUID: sprMain, TimeMS: int64(i),
					Side: analysis.SideKiller, DistanceM: 10},
				analysis.MeasuredKill{MatchID: id, KillerXUID: sprAutre, TimeMS: int64(50 + i),
					Side: analysis.SideKiller, DistanceM: 30},
			)
		}
	}
	return port.MatchRangeRead{Kills: kills, KillsTotal: len(kills)}
}
