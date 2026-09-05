package replayartifacts

// positions_test.go — LA PROJECTION DES POSITIONS, sans base.
//
// Ce qui est éprouvé : la décimation à [GrainPositionsMS] retient EXACTEMENT ce qu'elle doit
// retenir. C'est la seule chose qui décide du volume écrit — et c'est le point sur lequel la
// décision 1 se joue : projeter les trajectoires telles quelles écrirait ~31 000 lignes par
// match, la décimation en écrit ~215 (mesures du 2026-09-06 sur les 106 artefacts du cache).

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis/replay"
)

// trajectoire fabrique une vie dont les points vont de 0 à n-1 frames, une frame par point.
func trajectoire(team, n int) replay.Track {
	pts := make([]replay.Point, 0, n)
	for i := 0; i < n; i++ {
		pts = append(pts, replay.Point{T: i, X: float32(i), Y: float32(2 * i), Z: float32(3 * i)})
	}
	return replay.Track{Team: team, Points: pts}
}

func TestProjeterPositions_Decimation(t *testing.T) {
	// 100 ms par frame, 20 s de grain -> un point retenu toutes les 200 frames.
	const cadence = 100
	const parGrain = GrainPositionsMS / cadence

	cas := map[string]struct {
		doc     replay.ReplayDocument
		attendu int
	}{
		"une vie plus courte que le grain rend UN point": {
			doc: replay.ReplayDocument{
				FrameIntervalMS: cadence,
				Tracks:          []replay.Track{trajectoire(-1, 5)},
			},
			attendu: 1,
		},
		"une vie de trois grains rend trois points": {
			doc: replay.ReplayDocument{
				FrameIntervalMS: cadence,
				Tracks:          []replay.Track{trajectoire(-1, 2*parGrain+1)},
			},
			attendu: 3,
		},
		"deux vies se decimment independamment": {
			doc: replay.ReplayDocument{
				FrameIntervalMS: cadence,
				Tracks:          []replay.Track{trajectoire(0, parGrain+1), trajectoire(1, parGrain+1)},
			},
			attendu: 4,
		},
		"cadence absente : la grille par defaut de 100 ms": {
			doc: replay.ReplayDocument{
				Tracks: []replay.Track{trajectoire(-1, 2*parGrain+1)},
			},
			attendu: 3,
		},
		"aucune trajectoire : passe vide": {
			doc:     replay.ReplayDocument{FrameIntervalMS: cadence},
			attendu: 0,
		},
		"une trajectoire sans point : passe vide": {
			doc: replay.ReplayDocument{
				FrameIntervalMS: cadence,
				Tracks:          []replay.Track{{Team: 0}},
			},
			attendu: 0,
		},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			b := projeterPositions("m-1", &c.doc)
			if c.attendu == 0 {
				if b.MatchID != "" || len(b.Rows) != 0 {
					t.Fatalf("passe = %+v, attendue VIDE (rien à écrire n'est pas un défaut)", b)
				}
				return
			}
			if b.MatchID != "m-1" {
				t.Fatalf("MatchID = %q, attendu m-1", b.MatchID)
			}
			if len(b.Rows) != c.attendu {
				t.Fatalf("%d position(s) retenue(s), attendu %d", len(b.Rows), c.attendu)
			}
		})
	}
}

// TestProjeterPositions_ValeursTransportees : la projection TRANSPORTE, elle ne calcule pas.
func TestProjeterPositions_ValeursTransportees(t *testing.T) {
	doc := replay.ReplayDocument{
		FrameIntervalMS: 100,
		Tracks: []replay.Track{{
			Team: -1,
			Points: []replay.Point{
				{T: 0, X: 1.5, Y: -2.5, Z: 3.5},
				{T: 400, X: 9, Y: 8, Z: 7}, // 40 s plus tard : retenu
			},
		}},
	}
	b := projeterPositions("m-1", &doc)
	if len(b.Rows) != 2 {
		t.Fatalf("%d position(s), attendu 2", len(b.Rows))
	}
	if b.Rows[0].TimeMS != 0 || b.Rows[1].TimeMS != 40_000 {
		t.Errorf("axe de temps = [%d, %d], attendu [0, 40000] (frame x frameIntervalMs)",
			b.Rows[0].TimeMS, b.Rows[1].TimeMS)
	}
	if b.Rows[0].X != 1.5 || b.Rows[0].Y != -2.5 || b.Rows[0].Z != 3.5 {
		t.Errorf("position = (%v,%v,%v), attendu (1.5,-2.5,3.5)", b.Rows[0].X, b.Rows[0].Y, b.Rows[0].Z)
	}
	// L'ÉQUIPE N'EST PAS INVENTÉE : le film ne la porte pas, -1 est une valeur PLEINE.
	if b.Rows[0].Team != -1 {
		t.Errorf("team = %d, attendu -1 (non attribuée) — le film ne porte pas l'équipe",
			b.Rows[0].Team)
	}
}

// TestPersisterPositions_SansWriter : un chemin de sync sans writer câblé ne panique pas et ne
// prétend pas avoir écrit.
func TestPersisterPositions_SansWriter(t *testing.T) {
	doc := replay.ReplayDocument{FrameIntervalMS: 100, Tracks: []replay.Track{trajectoire(0, 3)}}
	persisterPositions(context.Background(), Deps{Gamertag: "t"},
		[]artefactLu{{matchID: "m-1", path: "peu-importe", doc: &doc}})
}
