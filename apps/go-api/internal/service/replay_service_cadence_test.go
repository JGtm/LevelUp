package service

// replay_service_cadence_test.go — la fraîcheur de cadence du document servi est décidée par le service
// et rendue avec le document : un document est périmé quand son pas n'est pas le pas de cuisson courant,
// un document sans pas se lisant à 100 ms.

import (
	"context"
	"strconv"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replaybuild"
)

func TestGetReplay_CadencePerimeeQuandLePasNEstPasLePasCourant(t *testing.T) {
	cas := []struct {
		nom     string
		courant int
		pas     int
		perime  bool
	}{
		{"100 ms face à 33 ms", 33, 100, true},
		{"100 ms face à 100 ms", 100, 100, false},
		{"pas absent face à 33 ms", 33, 0, true},
		{"pas absent face à 100 ms", 100, 0, false},
		{"33 ms face à 33 ms", 33, 33, false},
		{"33 ms face à 100 ms", 100, 33, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			defer replaybuild.ForceCookFrameIntervalMS(c.courant)()
			corps := `{"schemaVersion":115,"matchId":"0000ca01","frameCount":1`
			if c.pas != 0 {
				corps += `,"frameIntervalMs":` + strconv.Itoa(c.pas)
			}
			root := artefactAt(t, title.DefaultSlug, "0000ca01", corps+`}`)

			got, err := NewReplayService(title.DefaultSlug, root, nil, nil).GetReplay(context.Background(), "0000ca01")
			if err != nil {
				t.Fatalf("lecture du rejeu : %v", err)
			}
			if got.CadenceStale != c.perime {
				t.Errorf("CadenceStale = %v, attendu %v", got.CadenceStale, c.perime)
			}
			if got.FrameIntervalMS != c.pas {
				t.Errorf("le document doit garder son pas (%d), a %d", c.pas, got.FrameIntervalMS)
			}
		})
	}
}
