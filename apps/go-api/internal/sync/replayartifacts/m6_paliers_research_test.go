//go:build research

package replayartifacts

// m6_paliers_research_test.go — INSTRUMENT DU LOT M6 des retours du rejeu (revue adverse, constat
// R3, 2026-09-24) : ce que la sortie de la remise des mains nues hors de `pickups` et de
// `weaponChanges` change aux PALIERS DE SOCLE.
//
// `prisesPour` (et son jumeau web `prisesEnVie`) tient toute prise d arme de ces deux canaux,
// posterieure au premier point du slot, pour une PRISE EN VIE : elle disqualifie la vie de la
// mesure des armes de base (`weapontier.baseWeaponsOf`). Une remise datee apres ce premier point
// (reapparition sur un slot deja vu) disqualifiait donc une vie a tort. L instrument rejoue, sur
// deux reconstructions du parc (base, branche), les entrees de la mesure : prises retenues, vies
// retenues (meme regle que `baseWeaponsOf`), ensemble des armes de base (hors modes a departs
// aleatoires, que la mesure ignore — non connus ici : l ensemble est calcule pour tous).
//
//	M6_AVANT=<documents base> M6_APRES=<documents branche> \
//	  go test -tags research -count=1 -run '^TestM6Paliers$' -v ./internal/sync/replayartifacts/

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/weapontier"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

var m6DocRe = regexp.MustCompile(`^[0-9a-f]{8}\.json$`)

type m6Mesure struct {
	prises, vies int
	base         []string
}

func m6Mesurer(t *testing.T, chemin string) m6Mesure {
	t.Helper()
	blob, err := os.ReadFile(chemin) //nolint:gosec // instrument, chemin de l operateur
	if err != nil {
		t.Fatal(err)
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(blob, &doc); err != nil {
		t.Fatalf("%s : %v", chemin, err)
	}
	spawns, takes := departsPour(doc.Loadouts), prisesPour(&doc)
	m := weapontier.NewMatch(nil, nil, spawns, takes, false)
	armes := map[string]bool{}
	for _, s := range spawns {
		for _, w := range s.Weapons {
			if !armes[w] && m.TierOf(-1, w) == weapontier.TierBase {
				armes[w] = true
			}
		}
	}
	base := make([]string, 0, len(armes))
	for w := range armes {
		base = append(base, w)
	}
	sort.Strings(base)
	return m6Mesure{prises: len(takes), vies: m6ViesRetenues(spawns, takes), base: base}
}

// m6ViesRetenues : la regle de `baseWeaponsOf` (premiere emission par slot, ecartee si elle suit
// la premiere prise du slot), recomptee ici parce que le paquet n expose pas le nombre de vies.
func m6ViesRetenues(spawns []weapontier.Spawn, takes []weapontier.Take) int {
	prise := map[uint32]int{}
	for _, tk := range takes {
		if v, ok := prise[tk.Slot]; !ok || tk.T < v {
			prise[tk.Slot] = tk.T
		}
	}
	vu, ecarte, n := map[uint32]bool{}, map[uint32]bool{}, 0
	for _, s := range spawns {
		if vu[s.Slot] || ecarte[s.Slot] {
			continue
		}
		if p, ok := prise[s.Slot]; ok && s.T >= p {
			ecarte[s.Slot] = true
			continue
		}
		vu[s.Slot] = true
		n++
	}
	return n
}

func TestM6Paliers(t *testing.T) {
	avant, apres := os.Getenv("M6_AVANT"), os.Getenv("M6_APRES")
	if avant == "" || apres == "" {
		t.Skip("instrument : M6_AVANT et M6_APRES requis")
	}
	entrees, err := os.ReadDir(avant)
	if err != nil {
		t.Fatal(err)
	}
	var totA, totB m6Mesure
	docsPrises, docsVies, docsBase := 0, 0, 0
	for _, e := range entrees {
		if !m6DocRe.MatchString(e.Name()) {
			continue
		}
		a, b := m6Mesurer(t, filepath.Join(avant, e.Name())), m6Mesurer(t, filepath.Join(apres, e.Name()))
		totA.prises += a.prises
		totB.prises += b.prises
		totA.vies += a.vies
		totB.vies += b.vies
		if a.prises != b.prises {
			docsPrises++
		}
		if a.vies != b.vies {
			docsVies++
		}
		ba, bb := strings.Join(a.base, ","), strings.Join(b.base, ",")
		if a.vies != b.vies || ba != bb {
			t.Logf("   %s prises %d->%d vies %d->%d base [%s] -> [%s]", e.Name()[:8],
				a.prises, b.prises, a.vies, b.vies, ba, bb)
		}
		if ba != bb {
			docsBase++
		}
	}
	t.Logf("PALIERS : prises en vie %d -> %d (%d documents), vies retenues %d -> %d (%d documents), "+
		"ensemble des armes de base change sur %d documents", totA.prises, totB.prises, docsPrises,
		totA.vies, totB.vies, docsVies, docsBase)
}
