package replay

// ctf_bridge_research_test.go — INSTRUMENT DE RECHERCHE #2 (v7.5 voie B).
//
// L'instrument #1 (ctf_research_test.go) a montré que 90 % des tirs perdus en CTF tombent
// pendant une VIE NON NOMMÉE, et que la réplication des positions n'y est pour rien (pas
// médian 16,7 ms, 0,21 % des pas au-dessus de la tolérance). Celui-ci démonte le pont pour
// dire POURQUOI ces vies ne sont pas nommées. Trois causes possibles, mutuellement exclusives
// et toutes trois mesurables ici :
//
//	fil des morts incomplet   des morts du match n'apparaissent pas dans le chunk highlight
//	horloges qui dérivent     l'appariement vie <-> mort tient au début et lâche à la fin
//	plus de vies que de morts le découpage en vies coupe là où le joueur n'est pas mort
//
// COMME LE PREMIER, IL NE MODIFIE RIEN et se saute sans sa variable d'environnement.
//
//	CGO_ENABLED=0 FILM_CACHE_ROOT=<mainrepo>/data/cache CTF_RESEARCH_OUT=<dir> \
//	  CTF_BRIDGE_FILMS="64e8adfa:Catalyst" \
//	  go test ./internal/analysis/replay/ -run CTFBridge -timeout 60m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const ctfBridgeFilmsEnv = "CTF_BRIDGE_FILMS"

// ctfWidenedWindows : fenêtres d'appariement testées, en millisecondes. 150 est celle du code
// (deathMatchWindowMS) ; les suivantes disent si la fenêtre est la contrainte qui mord.
var ctfWidenedWindows = []int64{150, 500, 2000, 10000}

func TestCTFBridgeAnatomy(t *testing.T) {
	spec := os.Getenv(ctfBridgeFilmsEnv)
	if spec == "" {
		t.Skipf("recherche pont non demandée : %s vide", ctfBridgeFilmsEnv)
	}
	cache, outDir := os.Getenv(ctfCacheEnv), os.Getenv(ctfOutEnv)
	if cache == "" || outDir == "" {
		t.Fatalf("%s et %s sont requis", ctfCacheEnv, ctfOutEnv)
	}
	cat := loadCTFQuantCatalog(t)
	for _, item := range strings.Split(spec, ",") {
		short, mapName, ok := strings.Cut(strings.TrimSpace(item), ":")
		if !ok {
			t.Fatalf("entrée mal formée %q", item)
		}
		t.Run(short, func(t *testing.T) {
			b := ctfBridgeReport(t, cat, filepath.Join(cache, "film_chunks", short), short, mapName)
			path := filepath.Join(outDir, short+"_pont.txt")
			if err := os.WriteFile(path, []byte(b), 0o644); err != nil {
				t.Fatalf("écriture : %v", err)
			}
			t.Logf("\n%s", b)
		})
	}
}

func ctfBridgeReport(t *testing.T, cat *filmdec.MapQuantCatalog, dir, short, mapName string) string {
	t.Helper()
	entry, err := cat.Lookup(mapName)
	if err != nil {
		t.Fatalf("bornes de %s : %v", mapName, err)
	}
	world := entry.Range()
	scan := filmdec.DefaultScanFilmOptions()
	scan.WorldRange, scan.CaptureDirs = &world, true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions : %v", err)
	}
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("morts : %v", err)
	}
	fire, err := filmdec.ScanFilmFireEvents(dir)
	if err != nil {
		t.Fatalf("tirs : %v", err)
	}
	sort.SliceStable(pos, func(i, j int) bool { return pos[i].TimestampUS < pos[j].TimestampUS })
	tracks := indexBySlot(pos)
	lives := buildLifeSpans(tracks)
	off, matched := bestDeathOffset(lives, deaths)
	named := nameLivesByDeaths(lives, deaths, off)

	var b strings.Builder
	origin := pos[0].TimestampUS
	fmt.Fprintf(&b, "film\t%s\ncarte\t%s\nduree_s\t%.1f\n",
		short, mapName, float64(pos[len(pos)-1].TimestampUS-origin)/1e6)
	fmt.Fprintf(&b, "slots_distincts\t%d\nvies\t%d\nvies_nommees\t%d\nmorts_du_fil\t%d\nmorts_appariees\t%d\noffset_ms\t%d\n",
		len(tracks), len(lives), named, len(deaths), matched, off)
	ctfWriteWindowSweep(&b, tracks, deaths)
	ctfWriteUnnamedLives(&b, lives, fire, origin)
	ctfWriteDriftProbe(&b, lives, deaths, off, origin)
	return b.String()
}

// ctfWriteWindowSweep rejoue l'appariement avec des fenêtres de plus en plus larges. Si la
// fenêtre de 150 ms était la contrainte qui mord, l'élargir nommerait davantage de vies.
func ctfWriteWindowSweep(b *strings.Builder, tracks map[uint32]slotTrack, deaths []Death) {
	fmt.Fprintf(b, "\n# sensibilite a la fenetre d'appariement\n")
	for _, w := range ctfWidenedWindows {
		lv := buildLifeSpans(tracks) // vies neuves : nameLivesByDeaths écrit dedans
		o, _ := bestDeathOffset(lv, deaths)
		fmt.Fprintf(b, "fenetre_ms\t%d\tvies_nommees\t%d\tsur\t%d\n",
			w, ctfNameWithWindow(lv, deaths, o, w), len(lv))
	}
}

// ctfNameWithWindow est nameLivesByDeaths avec sa fenêtre en paramètre. C'est une COPIE DE
// RECHERCHE assumée : la constante du code reste intouchée, et une mesure de sensibilité qui
// modifierait le code mesuré ne vaudrait rien.
func ctfNameWithWindow(lives []lifeSpan, deaths []Death, off, windowMS int64) int {
	ends := lifeEndsMS(lives)
	type pair struct {
		di, li int
		d      int64
	}
	var ps []pair
	for di, d := range deaths {
		for li, e := range ends {
			if delta := absI64(e - (d.TimeMS + off)); delta <= windowMS {
				ps = append(ps, pair{di, li, delta})
			}
		}
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].d != ps[j].d {
			return ps[i].d < ps[j].d
		}
		return ps[i].li < ps[j].li
	})
	usedD, usedL, n := make([]bool, len(deaths)), make([]bool, len(lives)), 0
	for _, p := range ps {
		if usedD[p.di] || usedL[p.li] {
			continue
		}
		usedD[p.di], usedL[p.li] = true, true
		n++
	}
	return n
}

// ctfWriteUnnamedLives donne l'anatomie de chaque vie non nommée : quand elle commence, combien
// elle dure, et combien de tirs tombent dedans. C'est le compte qui relie le pont au calque.
func ctfWriteUnnamedLives(b *strings.Builder, lives []lifeSpan, fire []filmdec.FireEvent, origin uint64) {
	fmt.Fprintf(b, "\n# vies non nommees (t relatif au debut du film)\n")
	var unnamed []lifeSpan
	for _, l := range lives {
		if l.xuid == 0 {
			unnamed = append(unnamed, l)
		}
	}
	sort.Slice(unnamed, func(i, j int) bool { return unnamed[i].from < unnamed[j].from })
	for _, l := range unnamed {
		n := 0
		for _, e := range fire {
			if int64(e.TimestampUS) >= l.from && int64(e.TimestampUS) <= l.to {
				n++
			}
		}
		fmt.Fprintf(b, "vie\tslot\t%d\tdebut_s\t%.1f\tfin_s\t%.1f\tduree_s\t%.1f\ttirs_dans_la_fenetre\t%d\n",
			l.slot, float64(l.from-int64(origin))/1e6, float64(l.to-int64(origin))/1e6,
			float64(l.to-l.from)/1e6, n)
	}
}

// ctfWriteDriftProbe cherche une DÉRIVE entre l'horloge du fil des morts et celle du film :
// l'offset qui apparie le mieux la première moitié du film contre celui de la seconde. Deux
// valeurs éloignées signeraient une dérive ; deux valeurs proches l'écartent.
func ctfWriteDriftProbe(b *strings.Builder, lives []lifeSpan, deaths []Death, off int64, origin uint64) {
	mid := int64(origin) + (lives[len(lives)-1].to-int64(origin))/2
	var early, late []lifeSpan
	for _, l := range lives {
		if l.to < mid {
			early = append(early, l)
		} else {
			late = append(late, l)
		}
	}
	oe, ne := bestDeathOffset(early, deaths)
	ol, nl := bestDeathOffset(late, deaths)
	fmt.Fprintf(b, "\n# sonde de derive d'horloge\n")
	fmt.Fprintf(b, "offset_global_ms\t%d\noffset_premiere_moitie_ms\t%d\tapparies\t%d\noffset_seconde_moitie_ms\t%d\tapparies\t%d\n",
		off, oe, ne, ol, nl)
	// Résidus des vies nommées, par décile de film : une dérive les ferait croître.
	span := lives[len(lives)-1].to - int64(origin)
	if span <= 0 {
		return
	}
	sums, counts := make([]int64, 10), make([]int, 10)
	for _, l := range lives {
		if l.xuid == 0 {
			continue
		}
		best := int64(1 << 40)
		for _, d := range deaths {
			if d.XUID != l.xuid {
				continue
			}
			if r := (l.to / 1000) - (d.TimeMS + off); absI64(r) < absI64(best) {
				best = r
			}
		}
		i := int((l.to - int64(origin)) * 10 / span)
		if i > 9 {
			i = 9
		}
		sums[i] += best
		counts[i]++
	}
	fmt.Fprintf(b, "\n# residu median d'appariement par decile (ms)\n")
	for i := 0; i < 10; i++ {
		if counts[i] == 0 {
			continue
		}
		fmt.Fprintf(b, "decile\t%d\tvies_nommees\t%d\tresidu_moyen_ms\t%.1f\n",
			i+1, counts[i], float64(sums[i])/float64(counts[i]))
	}
}
