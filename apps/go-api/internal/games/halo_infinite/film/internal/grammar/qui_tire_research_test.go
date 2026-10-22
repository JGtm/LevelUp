//go:build research

package grammar

// qui_tire_research_test.go — INSTRUMENT : QUI TIRE depuis chaque vie publiee d'un artefact.
//
// LA QUESTION (controle n° 8 des controles de la v8, decision utilisateur du 2026-10-21 : « impossible
// d'avoir 5 joueurs en meme temps, meme s'il y a un bot »). Sur onze films du parc, la lecture montre
// cinq presences d'une equipe en 4v4 ; la cinquieme vie est nommee par la base (`tableau_api`). Le tir
// porte l'index de tireur (la PLACE, champ `d`) et son unite (reference 0, le corps qui tire) : pour
// chaque vie publiee, l'instrument compte les index de tireur des tirs dont l'unite est son corps.
//
// USAGE (lecture seule) :
//
//	QUITIRE_FILMS=<racine data/cache> QUITIRE_ID=3d58eb37 \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar -run TestQuiTire -v

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

type quiTireArtefact struct {
	FrameIntervalMs int `json:"frameIntervalMs"`
	Tracks          []struct {
		Slot   uint32  `json:"slot"`
		XUID   *string `json:"xuid"`
		Bot    string  `json:"bot"`
		Team   *int    `json:"team"`
		Points []struct {
			T int `json:"t"`
		} `json:"points"`
	} `json:"tracks"`
	Roster []struct {
		XUID      string `json:"xuid"`
		Name      string `json:"name"`
		FilmIndex int    `json:"filmIndex"`
		Seat      int    `json:"seat"`
	} `json:"roster"`
}

func TestQuiTire(t *testing.T) {
	racine, id := os.Getenv("QUITIRE_FILMS"), os.Getenv("QUITIRE_ID")
	if racine == "" || id == "" {
		t.Skip("QUITIRE_FILMS ou QUITIRE_ID absent : instrument saute")
	}
	raw, err := os.ReadFile(filepath.Join(racine, "replays", "halo_infinite", id+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var a quiTireArtefact
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	film, ok, err := filmcache.LoadFilm(racine, id)
	if err != nil || !ok {
		t.Fatalf("film : %v", err)
	}
	origine, err := ScanClockOrigin(film)
	if err != nil {
		t.Fatal(err)
	}
	tirs, err := ScanFireEvents(NewFilmContext(film))
	if err != nil {
		t.Fatal(err)
	}
	pas := uint64(max(a.FrameIntervalMs, 1)) * 1000
	noms := map[string]string{}
	for _, e := range a.Roster {
		noms[e.XUID] = fmt.Sprintf("%s(i%d,s%d)", e.Name, e.FilmIndex, e.Seat)
	}
	type vie struct {
		nom      string
		de, a    int
		parIndex map[int]int
	}
	var vies []*vie
	parSlot := map[uint32][]*vie{}
	for _, tr := range a.Tracks {
		if len(tr.Points) == 0 {
			continue
		}
		nom := tr.Bot
		if tr.XUID != nil && *tr.XUID != "" {
			nom = noms[*tr.XUID]
		}
		if tr.Team != nil {
			nom = fmt.Sprintf("e%d %s", *tr.Team, nom)
		}
		v := &vie{nom: nom, de: tr.Points[0].T, a: tr.Points[len(tr.Points)-1].T, parIndex: map[int]int{}}
		vies = append(vies, v)
		parSlot[tr.Slot] = append(parSlot[tr.Slot], v)
	}
	for _, e := range tirs {
		if !e.Unit.Present || !e.HasShooter || e.TimestampUS < origine {
			continue
		}
		f := int((e.TimestampUS - origine) / pas)
		for _, v := range parSlot[e.Unit.Slot] {
			if v.de <= f && f <= v.a {
				v.parIndex[e.FilmIndex]++
				break
			}
		}
	}
	// L'INDEX DE PARTICIPANT QUE LE RECORD DE CREATION DE CHAQUE CORPS ECRIT.
	creations, err := quiTireCreations(t, film)
	t.Logf("%d creations de corps", len(creations))
	if err != nil {
		t.Logf("creations : %v", err)
	}
	for _, c := range creations {
		if c.TimestampUS < origine {
			continue
		}
		f := int((c.TimestampUS - origine) / pas)
		for _, v := range parSlot[c.Slot] {
			if v.de-50 <= f && f <= v.de+5 {
				idx := -1
				if c.HasIndex {
					idx = int(c.ParticipantIndex)
				}
				v.nom += fmt.Sprintf(" [creation i%d]", idx)
			}
		}
	}
	sort.SliceStable(vies, func(i, j int) bool { return vies[i].nom < vies[j].nom })
	for _, v := range vies {
		var parts []string
		for idx, n := range v.parIndex {
			parts = append(parts, fmt.Sprintf("i%d:%d", idx, n))
		}
		sort.Strings(parts)
		t.Logf("%-40s [%5d,%5d] tirs par index %s", v.nom, v.de, v.a, strings.Join(parts, " "))
	}
}

// quiTireCreations lit les creations de corps dans le contexte de carte qui en rend le plus : le
// decoupage d'i0 depend de la carte, et deux cartes de memes largeurs rendent les memes records.
func quiTireCreations(t *testing.T, film *source.Film) ([]BipedCreation, error) {
	t.Helper()
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		return nil, err
	}
	var meilleures []BipedCreation
	carte := ""
	module := os.Getenv("QUITIRE_MODULE")
	for nom, entree := range cat.Maps {
		e := entree
		if module != "" && e.Module != module {
			continue
		}
		c, _, err := aj14fContexte(film, &e).CreationsDeBipede()
		if err == nil && len(c) > len(meilleures) {
			meilleures, carte = c, nom
		}
	}
	t.Logf("carte retenue pour les creations : %s", carte)
	return meilleures, nil
}
