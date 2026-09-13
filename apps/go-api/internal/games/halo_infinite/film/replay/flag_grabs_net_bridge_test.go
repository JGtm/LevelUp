package replay

// flag_grabs_net_bridge_test.go — le pont, et LE GARDE-RAIL DES QUATRE ETATS.
//
// `objectiveevents` recopie les quatre chaines d'etat du drapeau pour rester independant du
// decodeur de film. Une recopie sans garde-rail derive : ce test echoue le jour ou l'une des
// quatre diverge de sa source (meme dispositif que bomb_stats_sentinels_test.go).

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

func TestFlagStates_SentinellesObjectiveEvents(t *testing.T) {
	t.Parallel()
	paires := []struct {
		nom             string
		source, recopie string
	}{
		{"carried", FlagStateCarried, objectiveevents.FlagSpanCarried},
		{"carried_open", FlagStateCarriedOpen, objectiveevents.FlagSpanCarriedOpen},
		{"dropped", FlagStateDropped, objectiveevents.FlagSpanDropped},
		{"home", FlagStateHome, objectiveevents.FlagSpanHome},
	}
	for _, p := range paires {
		if p.source != p.recopie {
			t.Errorf("%s : replay=%q mais objectiveevents=%q — la recopie a derive de sa source",
				p.nom, p.source, p.recopie)
		}
	}
}

func xuidPtr(s string) *string { return &s }

// docDrapeau construit un document minimal porteur d'un calque de drapeau.
func docDrapeau(flagFilm bool, intervalMS int, spans []FlagSpan) *ReplayDocument {
	return &ReplayDocument{
		FrameIntervalMS: intervalMS,
		FlagCarries:     []FlagCarry{{Team: 0, Spans: spans}},
		Coverage: &Coverage{FlagCarries: &FlagCarriesCoverage{
			FlagFilm: flagFilm, Openings: 7,
		}},
	}
}

func TestFlagTracksOf_ConvertitLesFramesEnMillisecondes(t *testing.T) {
	t.Parallel()
	doc := docDrapeau(true, 100, []FlagSpan{
		{State: FlagStateCarried, T0: 50, T1: 100, XUID: xuidPtr("A")},
		{State: FlagStateDropped, T0: 100, T1: 112},
	})
	tracks, openings, ok := FlagTracksOf(doc)
	if !ok {
		t.Fatal("document CTF exploitable refuse")
	}
	if openings != 7 {
		t.Errorf("openings=%d, want 7 (le compte de la couverture)", openings)
	}
	if len(tracks) != 1 || len(tracks[0].Spans) != 2 {
		t.Fatalf("pistes = %+v", tracks)
	}
	sp := tracks[0].Spans[0]
	if sp.StartMS != 5_000 || sp.EndMS != 10_000 || sp.XUID != "A" {
		t.Errorf("premier portage = %+v, want 5000..10000 pour A", sp)
	}
	if tracks[0].Spans[1].XUID != "" {
		t.Errorf("un etat non porte a recu un xuid : %+v", tracks[0].Spans[1])
	}
}

func TestFlagTracksOf_RefusLesTroisSilences(t *testing.T) {
	t.Parallel()
	cas := []struct {
		nom string
		doc *ReplayDocument
	}{
		{"document nil", nil},
		{"pas de couverture de drapeau", &ReplayDocument{FrameIntervalMS: 100}},
		{"film qui n est pas du CTF", docDrapeau(false, 100, nil)},
		{"axe sans echelle", docDrapeau(true, 0, nil)},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if _, _, ok := FlagTracksOf(c.doc); ok {
				t.Errorf("%s : accepte, attendu NON MESURE", c.nom)
			}
		})
	}
}
