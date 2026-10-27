//go:build research

package replayartifacts

// citations_bouclier_research_test.go — INSTRUMENT DE L'ÉTAPE E1 du plan des citations Halo 5
// (`.ai/archive/V7.5/PLAN_CITATIONS_FILM_2026-10-16.md`) : « Presque, ça ne compte pas » = un frag dont le
// tueur a un bouclier à 0 % à l'instant du frag, et qui survit.
//
// Mesure, sur les artefacts du parc croisés aux frags de `match_kill_events_latest` (COPIE de la
// base partagée, jamais la base tenue par le serveur) : la part des frags dont la vie du tueur
// porte une mesure de bouclier avant le frag, l'âge de cette dernière mesure, la part à 0 %, la
// survie du tueur, et le coût de la recherche.
//
//	CITBOUCLIER_ARTEFACTS=<dossier des artefacts> CITBOUCLIER_SHARED=<copie shared_matches_v2.duckdb> \
//	  go test -tags research -count=1 -run '^TestCitationsBouclier$' -v ./internal/sync/replayartifacts/

import (
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

var citBouclierDocRe = regexp.MustCompile(`^([0-9a-f]{8})\.json$`)

type citBouclierFrag struct {
	timeMS int
	killer string
}

type citBouclierBilan struct {
	docs, docsAvecBouclier, frags, sansVie, sansMesure int
	ages                                               []int // ms entre la derniere mesure et le frag
	aZero                                              map[int]int
	aZeroSurvit                                        map[int]int
	ecartsVictime                                      []int // ms, fin de vie de la victime - frag
	viesVictimes, viesVictimesAvecMesure               int
	duree                                              time.Duration
}

var citBouclierPlafonds = []int{1000, 2000, 3000, 5000, 1 << 30}

func TestCitationsBouclier(t *testing.T) {
	dir, shared := os.Getenv("CITBOUCLIER_ARTEFACTS"), os.Getenv("CITBOUCLIER_SHARED")
	if dir == "" || shared == "" {
		t.Skip("CITBOUCLIER_ARTEFACTS et CITBOUCLIER_SHARED requis")
	}
	db, err := sql.Open("duckdb", shared+"?access_mode=read_only")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	b := citBouclierBilan{aZero: map[int]int{}, aZeroSurvit: map[int]int{}}
	for _, e := range entries {
		m := citBouclierDocRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		doc, err := LireArtefactRange(filepath.Join(dir, e.Name()))
		if err != nil || doc.OriginMs == nil || doc.FrameIntervalMS <= 0 {
			continue
		}
		frags, victimes := citBouclierFrags(t, db, doc.MatchID, m[1])
		if len(frags) == 0 {
			continue
		}
		b.docs++
		debut := time.Now()
		citBouclierMesurer(doc, frags, victimes, &b)
		b.duree += time.Since(debut)
	}
	sort.Ints(b.ages)
	sort.Ints(b.ecartsVictime)
	t.Logf("artefacts croises : %d (dont avec au moins une mesure de bouclier : %d)", b.docs, b.docsAvecBouclier)
	t.Logf("frags de tueur nomme : %d ; vie du tueur introuvable : %d ; vie trouvee sans mesure avant le frag : %d",
		b.frags, b.sansVie, b.sansMesure)
	avec := len(b.ages)
	t.Logf("frags avec une mesure de bouclier avant : %d (%.1f %% des frags)", avec, pct(avec, b.frags))
	if avec > 0 {
		t.Logf("age de la derniere mesure (ms) : p25 %d, mediane %d, p75 %d, p90 %d",
			b.ages[avec/4], b.ages[avec/2], b.ages[avec*3/4], b.ages[avec*9/10])
	}
	for _, p := range citBouclierPlafonds {
		t.Logf("bouclier a 0 %% et mesure de moins de %s : %d frags (%.1f %%), dont tueur vivant 1 s apres : %d",
			plafondTexte(p), b.aZero[p], pct(b.aZero[p], b.frags), b.aZeroSurvit[p])
	}
	if n := len(b.ecartsVictime); n > 0 {
		t.Logf("controle d horloge, fin de vie de la victime - frag (ms) : p10 %d, mediane %d, p90 %d (n=%d)",
			b.ecartsVictime[n/10], b.ecartsVictime[n/2], b.ecartsVictime[n*9/10], n)
	}
	t.Logf("vies de victimes portant au moins une mesure de bouclier : %d sur %d (%.1f %%)",
		b.viesVictimesAvecMesure, b.viesVictimes, pct(b.viesVictimesAvecMesure, b.viesVictimes))
	t.Logf("cout de la recherche : %s pour %d artefacts (%s par artefact)",
		b.duree, b.docs, b.duree/time.Duration(maxInt(b.docs, 1)))
}

func plafondTexte(p int) string {
	if p >= 1<<30 {
		return "l infini"
	}
	return strconv.Itoa(p) + " ms"
}

func pct(a, n int) float64 {
	if n == 0 {
		return 0
	}
	return 100 * float64(a) / float64(n)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// citBouclierFrags rend les frags a tueur nomme du match et, par xuid, les instants de ses morts.
func citBouclierFrags(t *testing.T, db *sql.DB, matchID, court string) ([]citBouclierFrag, map[string][]int) {
	t.Helper()
	if matchID == "" {
		return nil, nil
	}
	rows, err := db.Query(`SELECT time_ms, COALESCE(feed_killer_xuid, ''), COALESCE(victim_xuid, '')
		FROM match_kill_events_latest WHERE match_id = ? ORDER BY time_ms`, matchID)
	if err != nil {
		t.Fatalf("%s : %v", court, err)
	}
	defer rows.Close()
	var out []citBouclierFrag
	morts := map[string][]int{}
	for rows.Next() {
		var ms int
		var killer, victime string
		if err := rows.Scan(&ms, &killer, &victime); err != nil {
			t.Fatal(err)
		}
		if killer != "" && killer != victime {
			out = append(out, citBouclierFrag{timeMS: ms, killer: strings.TrimSpace(killer)})
		}
		if victime != "" {
			morts[victime] = append(morts[victime], ms)
		}
	}
	return out, morts
}

func citBouclierMesurer(doc *replay.ReplayDocument, frags []citBouclierFrag, morts map[string][]int, b *citBouclierBilan) {
	pas := doc.FrameIntervalMS
	origine := int(*doc.OriginMs)
	vies := map[string][]*replay.Track{}
	avecBouclier := false
	for i := range doc.Tracks {
		tr := &doc.Tracks[i]
		if tr.XUID != "" {
			vies[tr.XUID] = append(vies[tr.XUID], tr)
		}
		for j := range tr.Points {
			if tr.Points[j].Sh != nil {
				avecBouclier = true
			}
		}
	}
	if avecBouclier {
		b.docsAvecBouclier++
	}
	for _, f := range frags {
		b.frags++
		frame := (f.timeMS - origine) / pas
		tr := citBouclierVieA(vies[f.killer], frame)
		if tr == nil {
			b.sansVie++
			continue
		}
		var derniere *replay.Point
		for j := range tr.Points {
			p := &tr.Points[j]
			if p.T > frame {
				break
			}
			if p.Sh != nil {
				derniere = p
			}
		}
		if derniere == nil {
			b.sansMesure++
			continue
		}
		age := (frame - derniere.T) * pas
		b.ages = append(b.ages, age)
		if *derniere.Sh == 0 {
			survit := true
			for _, ms := range morts[f.killer] {
				if ms >= f.timeMS && ms <= f.timeMS+1000 {
					survit = false
				}
			}
			for _, p := range citBouclierPlafonds {
				if age < p {
					b.aZero[p]++
					if survit {
						b.aZeroSurvit[p]++
					}
				}
			}
		}
	}
	// Controle d'horloge : la fin de vie d'une victime nommee doit tomber sur l'instant du frag.
	for xuid, ms := range morts {
		for _, m := range ms {
			frame := (m - origine) / pas
			if tr := citBouclierVieFinissantPres(vies[xuid], frame); tr != nil {
				b.ecartsVictime = append(b.ecartsVictime, (citBouclierFin(tr)-frame)*pas)
				b.viesVictimes++
				for j := range tr.Points {
					if tr.Points[j].Sh != nil {
						b.viesVictimesAvecMesure++
						break
					}
				}
			}
		}
	}
}

func citBouclierFin(tr *replay.Track) int {
	if tr.EndFrame > 0 {
		return tr.EndFrame
	}
	if n := len(tr.Points); n > 0 {
		return tr.Points[n-1].T
	}
	return 0
}

func citBouclierDebut(tr *replay.Track) int {
	if tr.StartFrame > 0 {
		return tr.StartFrame
	}
	if len(tr.Points) > 0 {
		return tr.Points[0].T
	}
	return 0
}

func citBouclierVieA(vies []*replay.Track, frame int) *replay.Track {
	for _, tr := range vies {
		if frame >= citBouclierDebut(tr) && frame <= citBouclierFin(tr) {
			return tr
		}
	}
	return nil
}

func citBouclierVieFinissantPres(vies []*replay.Track, frame int) *replay.Track {
	var best *replay.Track
	bestD := 1 << 30
	for _, tr := range vies {
		d := citBouclierFin(tr) - frame
		if d < 0 {
			d = -d
		}
		if d < bestD {
			best, bestD = tr, d
		}
	}
	return best
}
