//go:build research

package service

// tactical_sequence_store_research_test.go — LA MESURE DU LOT L2.9 : combien coute la lecture a la
// demande des pistes d'un artefact (decodage partiel, sans cache), sur le plus gros artefact du parc.
// Critere du plan : plus de 500 ms = a remonter au user.
//
// LECTURE SEULE d'un fichier d'artefact (aucune base ouverte) :
//
//	SEQUENCE_RUNTIME_ROOT=<racine contenant data/cache/replays> SEQUENCE_MATCH=<short8 de l'artefact> \
//	  go test -tags research -run TestMesureLectureDesPistes -v ./internal/service/

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
)

func TestMesureLectureDesPistes(t *testing.T) {
	racine, match := os.Getenv("SEQUENCE_RUNTIME_ROOT"), os.Getenv("SEQUENCE_MATCH")
	if racine == "" || match == "" {
		t.Skip("SEQUENCE_RUNTIME_ROOT / SEQUENCE_MATCH non poses : mesure sur le parc")
	}
	store := NewTacticalSequenceStore(racine, "halo_infinite")
	var pistes *domain.TacticalPistes
	for i := 0; i < 5; i++ {
		debut := time.Now()
		p, err := store.Pistes(context.Background(), match)
		if err != nil || p == nil {
			t.Fatalf("Pistes: %v / %v", p, err)
		}
		pistes = p
		t.Logf("lecture %d : %v", i+1, time.Since(debut))
	}
	points := 0
	for _, tr := range pistes.Tracks {
		points += len(tr.Points)
	}
	t.Logf("pistes = %d, points = %d, frameIntervalMs = %d, originMs = %v", len(pistes.Tracks), points,
		pistes.FrameIntervalMS, pistes.OriginMs)

	// une sequence sur la premiere piste nommee, a mi-vie : le cout du calcul pur et le poids du corps.
	for _, tr := range pistes.Tracks {
		if tr.XUID == "" || len(tr.Points) < 400 {
			continue
		}
		mort := int64(tr.Points[len(tr.Points)/2].T) * int64(pistes.FrameIntervalMS)
		debut := time.Now()
		seq, err := tactical.Sequence(tactical.EntreeSequence{
			Victime:         tactical.ActeurSequence{Role: domain.TacticalRoleVictime, XUID: tr.XUID, Vies: [][]domain.TacticalPistePoint{tr.Points}},
			FrameIntervalMs: pistes.FrameIntervalMS, MortFilmMs: mort, RayonM: 18,
		})
		if err != nil {
			continue
		}
		corps, _ := json.Marshal(seq)
		t.Logf("sequence (une piste) : %v, corps = %d octets", time.Since(debut), len(corps))
		return
	}
}
