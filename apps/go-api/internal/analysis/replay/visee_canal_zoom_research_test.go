package replay

// visee_canal_zoom_research_test.go — LE CANAL DU ZOOM DANS LES PAQUETS DELTA. Les deux
// recensements qui ont clos la campagne visee : l'inventaire des types d'event d'un film, et la
// chasse au type 126 (`unit_zoom`) sur tout le corpus. L'instrument principal (differentiel du
// tir fatal) est dans `visee_tir_research_test.go` ; ce fichier vit a part pour le seuil de
// 500 lignes du depot.
//
// CE QUE LA TABLE DES HANDLERS A DONNE (Ghidra, 2026-08-30). Chaque paquet delta commence par un
// type d'event sur 7 bits (dispatcher @0x14080AADE) ; la table @0x144724A90 associe chaque type
// a un descripteur, et le getName de chaque descripteur nomme le type. Types observes dans les
// films : 105 = action_weapon_fire, 64 = weapon_overheat, 99 = weapon_empty_click,
// 115 = projectile_detonate, 114 = biped_board_vehicle, 68 = PowerUpApplied, ... et
// type 126 = unit_zoom — le descripteur 143d0da50, celui-la meme dont l'applicateur
// (+0x78 -> FUN_14110ec20) ecrit le niveau de zoom desire (unite+0x462).
//
// SOUS GARDE D'ENVIRONNEMENT, saute partout ailleurs, CI comprise. Parallelisme sans verrou :
// memes raisons que l'instrument principal (aucun global de decodage touche).

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/filmdec"
)

// TestViseeTypesDePaquet recense les TYPES D'EVENT des paquets delta d'un film (payload[0]>>1,
// le champ 7 bits du dispatcher @0x14080AADE) : c'est le seul flux du film jamais inventorie.
// Si les COMMANDES des joueurs voyagent dans la bobine, elles sont sous un de ces types — et le
// niveau de zoom desire est l'octet 6 de la commande (FUN_1406db688, ecrit unite+0x462).
// Garde : TIR_TYPES_FILM porte le repertoire d'UN film.
func TestViseeTypesDePaquet(t *testing.T) {
	dir := os.Getenv("TIR_TYPES_FILM")
	if dir == "" {
		t.Skipf("TIR_TYPES_FILM absent : recensement saute")
	}
	n := filmdec.CountFilmChunks(dir)
	compte := map[int]int{}
	tailles := map[int]int{}
	octets := map[int]int{}
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 1 {
				continue
			}
			pay := p.Payload(chunk)
			ty := int(pay[0] >> 1)
			compte[ty]++
			octets[ty] += len(pay)
			if len(pay) > tailles[ty] {
				tailles[ty] = len(pay)
			}
		}
	}
	var types []int
	for ty := range compte {
		types = append(types, ty)
	}
	sort.Slice(types, func(i, j int) bool { return compte[types[i]] > compte[types[j]] })
	t.Logf("TYPES — %d types distincts dans les paquets delta de %s", len(types), filepath.Base(dir))
	for _, ty := range types {
		t.Logf("  type %3d : %7d paquets · taille max %5d o · moyenne %7.1f o",
			ty, compte[ty], tailles[ty], float64(octets[ty])/float64(compte[ty]))
	}
}

// TestViseeCensusTypesCorpus recense les types d'event des paquets delta sur TOUT le corpus, et
// capture les eventuels paquets du type 126 : `unit_zoom` — nomme par la table des handlers du
// dispatcher (@0x144724A90, entree 126 -> descripteur 143d0da50, getName « unit_zoom »).
// C'est LA question binaire de la campagne visee : si un seul film porte un paquet 126, l'etat
// de lunette voyage dans la bobine. Garde : TIR_FILMS_DIR.
func TestViseeCensusTypesCorpus(t *testing.T) {
	root := os.Getenv(tirFilmsDirEnv)
	if root == "" {
		t.Skipf("%s absent : recensement saute", tirFilmsDirEnv)
	}
	dirs := adsListeFilms(t, root)
	if cap, _ := strconv.Atoi(os.Getenv(tirMaxFilmsEnv)); cap > 0 && cap < len(dirs) {
		dirs = dirs[:cap]
	}
	type bilan struct {
		types     [128]int
		films     [128]int
		zoomHex   []string
		zoomFilms []string
	}
	agg := bilan{}
	var mu sync.Mutex
	in := make(chan string, len(dirs))
	for _, d := range dirs {
		in <- d
	}
	close(in)
	var wg sync.WaitGroup
	workers := runtime.NumCPU()
	if workers > 8 {
		workers = 8
	}
	debut := time.Now()
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for d := range in {
				dir := filepath.Join(root, d)
				var loc [128]int
				var zoomHex []string
				n := filmdec.CountFilmChunks(dir)
				for c := 1; c <= n; c++ {
					chunk, err := filmdec.ReadFilmChunk(dir, c)
					if err != nil {
						continue
					}
					for _, p := range filmdec.WalkPackets(chunk) {
						if p.Type != filmdec.PacketTypeDelta || p.Size < 1 {
							continue
						}
						pay := p.Payload(chunk)
						ty := int(pay[0] >> 1)
						loc[ty]++
						if ty == 126 && len(zoomHex) < 3 {
							dump := pay
							if len(dump) > 40 {
								dump = dump[:40]
							}
							zoomHex = append(zoomHex,
								fmt.Sprintf("%s chunk %d paquet %d t=%dus : %x",
									d, c, p.Index, p.TimestampUS, dump))
						}
					}
				}
				mu.Lock()
				for ty, v := range loc {
					agg.types[ty] += v
					if v > 0 {
						agg.films[ty]++
					}
				}
				if len(zoomHex) > 0 {
					agg.zoomHex = append(agg.zoomHex, zoomHex...)
					agg.zoomFilms = append(agg.zoomFilms, d)
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	t.Logf("COUT — %d films en %s", len(dirs), time.Since(debut).Round(time.Second))
	for ty, v := range agg.types {
		if v > 0 {
			t.Logf("  type %3d : %9d paquets sur %4d films", ty, v, agg.films[ty])
		}
	}
	if len(agg.zoomFilms) == 0 {
		t.Logf("VERDICT — AUCUN paquet de type 126 (unit_zoom) dans les %d films du corpus :"+
			" le canal existe dans le dispatcher du jeu, la bobine ne l'emprunte jamais.", len(dirs))
		return
	}
	sort.Strings(agg.zoomFilms)
	t.Logf("VERDICT — unit_zoom PRESENT dans %d films : %s", len(agg.zoomFilms),
		strings.Join(agg.zoomFilms, ", "))
	for _, h := range agg.zoomHex {
		t.Logf("  %s", h)
	}
}

// TestViseeCanalFenetres — LE ZOOM SOUS N'IMPORTE QUEL NOM. Pour chaque type d'event, la
// presence de paquets dans la fenetre [t-8s, t+2s] d'un kill est comparee entre les kills
// ZOOMES (Counter-snipe) et NON ZOOMES (No Scope). Un event emis a la mise en lunette (ou au
// descope) separerait massivement, quel que soit son nom — c'est le filet qui attrape un zoom
// « code autrement », l'hypothese de l'utilisateur.
//
// SEUILS ECRITS AVANT LA MESURE : CANDIDAT si ecart de presence >= 0,35 ET ratio >= 2 ;
// « a suivre » si ecart >= 0,20. Population minimale : 30 fenetres par classe.
// LIMITE DITE D'AVANCE : la fenetre est centree sur le TUEUR dans le temps, pas dans l'espace —
// les paquets des autres joueurs y sont aussi. Un event de zoom PAR JOUEUR y restera visible
// (le tueur zoome a coup sur dans la classe zoomee), mais l'ecart mesure est dilue d'autant.
func TestViseeCanalFenetres(t *testing.T) {
	root := os.Getenv(tirFilmsDirEnv)
	if root == "" {
		t.Skipf("%s absent : instrument saute", tirFilmsDirEnv)
	}
	dirs := adsListeFilms(t, root)
	if cap, _ := strconv.Atoi(os.Getenv(tirMaxFilmsEnv)); cap > 0 && cap < len(dirs) {
		dirs = dirs[:cap]
	}
	const avantMS, apresMS = 8000, 2000
	var zoome, nonZoome, autresKills canalClasse
	var mu sync.Mutex
	in := make(chan string, len(dirs))
	for _, d := range dirs {
		in <- d
	}
	close(in)
	var wg sync.WaitGroup
	workers := runtime.NumCPU()
	if workers > 8 {
		workers = 8
	}
	debut := time.Now()
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for d := range in {
				dir := filepath.Join(root, d)
				feed, ok := tirLitFeed(dir)
				if !ok || (len(feed.noScope) == 0 && len(feed.counter) == 0) || len(feed.kills) == 0 {
					continue
				}
				types, records := canalLitTypes(dir)
				if len(records) == 0 {
					continue
				}
				off, ok := tirRecale(feed.kills, records)
				if !ok {
					continue
				}
				var zl, nl, al canalClasse
				canalCompte(&zl, feed.counter, off, types, avantMS, apresMS)
				canalCompte(&nl, feed.noScope, off, types, avantMS, apresMS)
				canalCompte(&al, canalAutresKills(feed, 8), off, types, avantMS, apresMS)
				mu.Lock()
				canalFusionne(&zoome, zl)
				canalFusionne(&nonZoome, nl)
				canalFusionne(&autresKills, al)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	t.Logf("COUT — %d films en %s ; fenetres : %d zoomees, %d non zoomees, %d autres kills"+
		" (temoin de contexte)", len(dirs), time.Since(debut).Round(time.Second),
		zoome.fenetres, nonZoome.fenetres, autresKills.fenetres)
	if zoome.fenetres < tirMinParClasse || nonZoome.fenetres < tirMinParClasse {
		t.Logf("VERDICT — POPULATION INSUFFISANTE (%d / %d, seuil %d). RIEN N'EST CONCLU.",
			zoome.fenetres, nonZoome.fenetres, tirMinParClasse)
		return
	}
	candidats := 0
	for ty := 0; ty < 128; ty++ {
		if zoome.avecType[ty] == 0 && nonZoome.avecType[ty] == 0 {
			continue
		}
		pz := float64(zoome.avecType[ty]) / float64(zoome.fenetres)
		pn := float64(nonZoome.avecType[ty]) / float64(nonZoome.fenetres)
		ecart, ratio := pz-pn, 0.0
		if ecart < 0 {
			ecart = -ecart
		}
		lo, hi := pz, pn
		if lo > hi {
			lo, hi = hi, lo
		}
		if lo > 0 {
			ratio = hi / lo
		} else {
			ratio = 999
		}
		mz := float64(zoome.paquets[ty]) / float64(zoome.fenetres)
		mn := float64(nonZoome.paquets[ty]) / float64(nonZoome.fenetres)
		pa := 0.0
		if autresKills.fenetres > 0 {
			pa = float64(autresKills.avecType[ty]) / float64(autresKills.fenetres)
		}
		switch {
		case ecart >= 0.35 && ratio >= 2:
			candidats++
			t.Logf("  type %3d : CANDIDAT — presence %5.1f %% vs %5.1f %% · temoin autres kills"+
				" %5.1f %% (moy %0.2f vs %0.2f paquets/fenetre)", ty, 100*pz, 100*pn, 100*pa, mz, mn)
		case ecart >= 0.20:
			t.Logf("  type %3d : a suivre — presence %5.1f %% vs %5.1f %% · temoin %5.1f %%"+
				" (moy %0.2f vs %0.2f)", ty, 100*pz, 100*pn, 100*pa, mz, mn)
		}
	}
	if candidats == 0 {
		t.Log("VERDICT — aucun type de paquet ne separe les fenetres zoomees des fenetres non" +
			" zoomees au seuil declare : pas d'event de mise en lunette dans la bobine, meme" +
			" sous un autre nom.")
	} else {
		t.Logf("VERDICT — %d type(s) CANDIDAT(S) : decoder leur grammaire (case +0x68 du"+
			" descripteur) et confronter au registre avant toute publication.", candidats)
	}
}

// canalAutresKills rend jusqu'a `max` instants de kill NON etiquetes (ni No Scope ni
// Counter-snipe a moins d'une seconde) : le temoin de CONTEXTE — memes films, memes combats,
// arme quelconque. Deterministe : les premiers du film dans l'ordre du feed.
func canalAutresKills(f tirFeed, max int) []int64 {
	var out []int64
	for _, k := range f.kills {
		if len(out) >= max {
			break
		}
		pres := false
		for _, l := range append(append([]int64{}, f.noScope...), f.counter...) {
			if k-l < 1000 && l-k < 1000 {
				pres = true
				break
			}
		}
		if !pres {
			out = append(out, k)
		}
	}
	return out
}

// canalLitTypes rend (type, instant) de tous les paquets delta, et la sous-liste des records
// 105 longs au format attendu par tirRecale.
func canalLitTypes(dir string) ([][2]int64, []tirRecord) {
	n := filmdec.CountFilmChunks(dir)
	var types [][2]int64
	var records []tirRecord
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 1 {
				continue
			}
			pay := p.Payload(chunk)
			ty := int64(pay[0] >> 1)
			tMS := int64(p.TimestampUS / 1000)
			types = append(types, [2]int64{ty, tMS})
			if ty == int64(filmdec.FireEventType) && int(pay[0])&1 == 0 {
				records = append(records, tirRecord{tMS: tMS})
			}
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].tMS < records[j].tMS })
	return types, records
}

// canalClasse accumule, par type d'event, la presence et le volume dans les fenetres d'une
// classe de kills.
type canalClasse struct {
	fenetres int
	avecType [128]int
	paquets  [128]int
}

// canalCompte verse les fenetres d'une liste d'instants etiquetes (horloge du feed) dans une
// classe. `types` est la liste (type, instant film) triee par construction du balayage.
func canalCompte(c *canalClasse, instants []int64, off int64, types [][2]int64, avantMS, apresMS int64) {
	for _, tms := range instants {
		debut, fin := tms+off-avantMS, tms+off+apresMS
		var vus [128]int
		for _, tp := range types {
			if tp[1] >= debut && tp[1] <= fin {
				vus[tp[0]]++
			}
		}
		c.fenetres++
		for ty, n := range vus {
			if n > 0 {
				c.avecType[ty]++
				c.paquets[ty] += n
			}
		}
	}
}

// canalFusionne verse une classe de film dans l'agregat corpus.
func canalFusionne(dst *canalClasse, src canalClasse) {
	dst.fenetres += src.fenetres
	for ty := 0; ty < 128; ty++ {
		dst.avecType[ty] += src.avecType[ty]
		dst.paquets[ty] += src.paquets[ty]
	}
}

// TestViseeCanal97Enveloppe — l'enveloppe du type 97 porte-t-elle un index de joueur au meme
// endroit que le record de degat (bits 36..40, valeur = index x 2) ? Histogramme des valeurs
// sur UN film + les tailles de payload : c'est le prealable au conditionnement par tueur.
// Garde : TIR_TYPES_FILM.
func TestViseeCanal97Enveloppe(t *testing.T) {
	dir := os.Getenv("TIR_TYPES_FILM")
	if dir == "" {
		t.Skipf("TIR_TYPES_FILM absent : saute")
	}
	n := filmdec.CountFilmChunks(dir)
	hist := map[int]int{}
	tailles := map[int]int{}
	total := 0
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 6 {
				continue
			}
			pay := p.Payload(chunk)
			if int(pay[0]>>1) != 97 {
				continue
			}
			total++
			hist[filmdec.ReadAttackerIndex(pay)]++
			tailles[len(pay)]++
		}
	}
	t.Logf("TYPE 97 — %d paquets", total)
	var idx []int
	for v := range hist {
		idx = append(idx, v)
	}
	sort.Ints(idx)
	for _, v := range idx {
		t.Logf("  bits36..40>>1 = %2d : %d paquets", v, hist[v])
	}
	var ts []int
	for v := range tailles {
		ts = append(ts, v)
	}
	sort.Ints(ts)
	aff := 0
	for _, v := range ts {
		if aff < 12 {
			t.Logf("  taille %4d o : %d paquets", v, tailles[v])
			aff++
		}
	}
}
