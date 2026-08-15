package replaylabels

// cadence_corpus_test.go — LES DEUX MESURES QUI DÉCIDENT DU MODE DE TIR.
//
// POURQUOI UN INSTRUMENT, ET PAS UNE SONDE JETABLE. Deux questions se posent au rendu du
// rejeu, et aucune ne se tranche de mémoire :
//
//  1. QUELLES ARMES TIRENT EN FAISCEAU TENU. « Continu » est une propriété de l'ARME, pas
//     du tir : la mesure est l'écart temporel entre deux dégâts consécutifs d'une même arme
//     tenue par le même tireur. Une arme continue frappe à cadence très serrée et régulière.
//  2. LE TIR CHARGÉ EST-IL QUALIFIABLE. La jauge de charge (`AmmoSlot.Gauge`) est lue aux
//     IMAGES-CLÉS, pas aux instants de tir : la question est de savoir de combien de temps
//     un tir est séparé de la lecture de jauge la plus proche.
//
// GARDE : `REPLAY_CORPUS` pointe le dossier d'artefacts (data/cache/replays/halo_infinite).
// Sans lui, les tests SAUTENT — le corpus n'est pas versionné, la CI ne l'a pas. Aucun
// décodage de film n'a lieu ici : on relit des artefacts JSON déjà construits.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/games/weapons"
)

// corpusDir rend le dossier d'artefacts, ou saute le test.
func corpusDir(t *testing.T) string {
	t.Helper()
	dir := strings.TrimSpace(os.Getenv("REPLAY_CORPUS"))
	if dir == "" {
		t.Skip("REPLAY_CORPUS non défini : corpus d'artefacts absent (non versionné)")
	}
	return dir
}

// artefacts liste les documents de rejeu du corpus, extension .json STRICTE (le dossier
// porte aussi des copies datées, qui ne sont pas des artefacts courants).
func artefacts(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture du corpus : %v", err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		out = append(out, filepath.Join(dir, e.Name()))
	}
	sort.Strings(out)
	return out
}

func lireDocument(t *testing.T, path string) replay.ReplayDocument {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // chemin de corpus fourni par l'opérateur
	if err != nil {
		t.Fatalf("lecture %s : %v", path, err)
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("décodage %s : %v", path, err)
	}
	return doc
}

// cleDArme rend le weapon_key d'un identifiant d'arme du document, ou "".
func cleDArme(registre map[uint32]string, id string) string {
	fam, ok := replay.FamilyOfWeaponID(id)
	if !ok {
		return ""
	}
	return registre[fam]
}

// intervalleMS rend la durée d'une frame du document (100 ms en production).
func intervalleMS(doc replay.ReplayDocument) int {
	if doc.FrameIntervalMS > 0 {
		return doc.FrameIntervalMS
	}
	return 100
}

type cadence struct {
	tirs   int
	deltas []int // écarts consécutifs, en ms, même tireur et même arme
	serres int   // écarts <= une frame
	// nuls : écarts NULS (deux dégâts dans la même frame). Ce n'est PAS une cadence — une
	// arme qui touche deux cibles d'un coup (chaîne électrique, gerbe) en produit autant.
	// Séparé des écarts d'une frame pour que la colonne « serrée » reste lisible.
	nuls int
}

func mediane(v []int) int {
	if len(v) == 0 {
		return -1
	}
	c := append([]int(nil), v...)
	sort.Ints(c)
	return c[len(c)/2]
}

func quantile(v []int, q float64) int {
	if len(v) == 0 {
		return -1
	}
	c := append([]int(nil), v...)
	sort.Ints(c)
	i := int(q * float64(len(c)-1))
	return c[i]
}

// TestCadenceDesArmesSurLeCorpus — LA MESURE QUI CLASSE LES ARMES À FAISCEAU TENU.
//
// Regroupement par (film, slot tireur, arme) : deux tirs consécutifs du MÊME tireur avec la
// MÊME arme. On publie l'écart médian, la part d'écarts d'au plus une frame, et la médiane
// des écarts « en rafale » (au plus 1 s) — c'est celle-là qui dit la cadence de l'arme, les
// grands écarts n'étant que du temps sans tirer.
func TestCadenceDesArmesSurLeCorpus(t *testing.T) {
	dir := corpusDir(t)
	registre := weapons.FilmshellWeaponKeysByFamily()
	cat, err := Load(repoRoot(t), "halo_infinite")
	if err != nil {
		t.Fatalf("catalogue du titre : %v", err)
	}
	par := map[string]*cadence{}
	films := 0
	// LE DÉNOMINATEUR EST TENU : un tir dont la famille n'a pas de weapon_key sort de la
	// table par arme. Sans ce compte, une arme absente du classement serait indistinguable
	// d'une arme absente du corpus — exactement la confusion que cette mesure doit éviter.
	sansCle := map[uint32]int{}
	totalTirs := 0
	// Les armes PORTÉES (loadouts) : une arme peut être au corpus sans y tirer.
	portees := map[string]int{}
	for _, path := range artefacts(t, dir) {
		doc := lireDocument(t, path)
		for _, l := range doc.Loadouts {
			for _, w := range l.W {
				if k := cleDArme(registre, w); k != "" {
					portees[k]++
				}
			}
		}
		if len(doc.Shots) == 0 {
			continue
		}
		films++
		pas := intervalleMS(doc)
		// suites[clé de serie] = frames triées
		suites := map[string][]int{}
		for _, s := range doc.Shots {
			totalTirs++
			key := cleDArme(registre, s.Weapon)
			if key == "" {
				fam, ok := replay.FamilyOfWeaponID(s.Weapon)
				if !ok {
					continue
				}
				sansCle[fam]++
				// LES FAMILLES SANS CLÉ ENTRENT QUAND MÊME dans la mesure de cadence, sous leur
				// hexadécimal : une source de dégât non nommée peut très bien être un faisceau
				// tenu, et l'écarter du classement reviendrait à décider avant de mesurer.
				key = fmt.Sprintf("0x%08X", fam)
			}
			id := fmt.Sprintf("%d|%s", s.Slot, key)
			suites[id] = append(suites[id], s.T)
			if par[key] == nil {
				par[key] = &cadence{}
			}
			par[key].tirs++
		}
		for id, frames := range suites {
			key := id[strings.IndexByte(id, '|')+1:]
			sort.Ints(frames)
			for i := 1; i < len(frames); i++ {
				d := (frames[i] - frames[i-1]) * pas
				c := par[key]
				c.deltas = append(c.deltas, d)
				if d <= pas {
					c.serres++
				}
				if d == 0 {
					c.nuls++
				}
			}
		}
	}
	if films == 0 {
		t.Fatal("aucun artefact porteur de tirs dans le corpus")
	}
	// Nom affichable par weapon_key : le catalogue nomme par FAMILLE, la mesure agrège par clé.
	nomParCle := map[string]string{}
	for fam, key := range cat.Keys {
		if w, ok := cat.Weapons[fam]; ok && w.Fr != "" {
			nomParCle[key] = w.Fr
		}
	}
	type ligne struct {
		key                                 string
		nom                                 string
		tirs, paires                        int
		med, medRafale, p90                 int
		partSerree, partEnRafale, partNulle float64
	}
	var lignes []ligne
	for key, c := range par {
		nom := key
		if n := nomParCle[key]; n != "" {
			nom = n
		}
		var enRafale []int
		for _, d := range c.deltas {
			if d <= 1000 {
				enRafale = append(enRafale, d)
			}
		}
		l := ligne{key: key, nom: nom, tirs: c.tirs, paires: len(c.deltas),
			med: mediane(c.deltas), medRafale: mediane(enRafale), p90: quantile(c.deltas, 0.9)}
		if len(c.deltas) > 0 {
			l.partSerree = 100 * float64(c.serres) / float64(len(c.deltas))
			l.partEnRafale = 100 * float64(len(enRafale)) / float64(len(c.deltas))
			l.partNulle = 100 * float64(c.nuls) / float64(len(c.deltas))
		}
		lignes = append(lignes, l)
	}
	sort.Slice(lignes, func(i, j int) bool { return lignes[i].tirs > lignes[j].tirs })
	t.Logf("CADENCE PAR ARME — %d films, %d tirs, écarts entre tirs consécutifs du même tireur", films, totalTirs)
	t.Logf("%-24s %7s %7s %9s %9s %9s %9s %9s %8s", "arme", "tirs", "paires", "med(ms)", "med<=1s", "p90(ms)", "<=1frame", "part<=1s", "nuls")
	classes := 0
	for _, l := range lignes {
		classes += l.tirs
		t.Logf("%-24s %7d %7d %9d %9d %9d %8.1f%% %8.1f%% %6.1f%%",
			l.nom, l.tirs, l.paires, l.med, l.medRafale, l.p90, l.partSerree, l.partEnRafale, l.partNulle)
	}
	t.Logf("tirs mesurés %d / %d — les familles ci-dessous n'ont PAS de weapon_key au registre "+
		"et figurent dans la table sous leur hexadécimal :", classes, totalTirs)
	fams := make([]uint32, 0, len(sansCle))
	for f := range sansCle {
		fams = append(fams, f)
	}
	sort.Slice(fams, func(i, j int) bool { return sansCle[fams[i]] > sansCle[fams[j]] })
	for _, f := range fams {
		t.Logf("  famille %08X : %d tir(s) SANS weapon_key", f, sansCle[f])
	}
	// UNE ARME PEUT ÊTRE AU CORPUS SANS Y TIRER : le loadout dit ce qui est porté. Une arme
	// absente des DEUX tables est absente du corpus, point — et aucune mesure de cadence ne
	// pourra jamais la classer.
	cles := make([]string, 0, len(portees))
	for k := range portees {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return portees[cles[i]] > portees[cles[j]] })
	t.Logf("ARMES PORTÉES (lectures de loadout), pour distinguer « absente » de « n'a pas tiré » :")
	for _, k := range cles {
		tirs := 0
		if c := par[k]; c != nil {
			tirs = c.tirs
		}
		t.Logf("  %-22s portée %6d fois, %6d tir(s)", k, portees[k], tirs)
	}
}

// jaugeParFamille — les familles d'arme pour lesquelles le film écrit une jauge de charge,
// et le nombre de lectures. La jointure emplacement -> arme passe par le Loadout du MÊME
// instant : `Inventory.Am[k]` suit l'ordre de `Loadout.W`.
type lectureJauge struct {
	frame int
	slot  uint32
	fam   string
	val   float32
}

func lecturesDeJauge(doc replay.ReplayDocument) []lectureJauge {
	// loadouts[slot][frame] = familles portées
	loadouts := map[uint32]map[int][]string{}
	for _, l := range doc.Loadouts {
		if loadouts[l.Slot] == nil {
			loadouts[l.Slot] = map[int][]string{}
		}
		loadouts[l.Slot][l.T] = l.W
	}
	var out []lectureJauge
	for _, inv := range doc.Inventory {
		portees := loadouts[inv.Slot][inv.T]
		for k, a := range inv.Am {
			if a.Gauge == nil || k >= len(portees) {
				continue
			}
			out = append(out, lectureJauge{frame: inv.T, slot: inv.Slot, fam: portees[k], val: *a.Gauge})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].frame < out[j].frame })
	return out
}

// TestEncadrementDeLaJaugeParLesTirs — LA MESURE QUI DÉCIDE DU TIR CHARGÉ.
//
// Pour chaque tir d'une arme QUI PORTE UNE JAUGE dans le corpus : distance temporelle à la
// lecture de jauge la plus proche (même tireur, même arme), puis part des tirs ENCADRÉS par
// deux lectures, et part de ceux dont la jauge bouge distinctement entre l'avant et l'après.
func TestEncadrementDeLaJaugeParLesTirs(t *testing.T) {
	dir := corpusDir(t)
	registre := weapons.FilmshellWeaponKeysByFamily()
	type acc struct {
		tirs     int
		ecarts   []int
		encadres int
		bougent  int
		lectures int
	}
	par := map[string]*acc{}
	totalTirs, totalLectures := 0, 0
	// QUI PORTE UNE JAUGE, MESURÉ : l'appariement emplacement -> arme passe par le Loadout du
	// même instant. S'il est faux, des armes à chargeur apparaîtront ici — c'est précisément
	// ce que ce compte rend visible au lieu de le supposer.
	jaugeParArme := map[string]int{}
	for _, path := range artefacts(t, dir) {
		doc := lireDocument(t, path)
		pas := intervalleMS(doc)
		lect := lecturesDeJauge(doc)
		totalLectures += len(lect)
		// index par (slot, famille)
		idx := map[string][]lectureJauge{}
		for _, l := range lect {
			id := fmt.Sprintf("%d|%s", l.slot, strings.ToUpper(strings.TrimPrefix(l.fam, "0x")))
			idx[id] = append(idx[id], l)
		}
		familles := map[string]bool{}
		for _, l := range lect {
			famHex := strings.ToUpper(strings.TrimPrefix(l.fam, "0x"))
			familles[famHex] = true
			nom := famHex
			var v uint64
			if _, err := fmt.Sscanf(famHex, "%X", &v); err == nil {
				if k := registre[uint32(v)]; k != "" {
					nom = k
				}
			}
			jaugeParArme[nom]++
		}
		for _, s := range doc.Shots {
			fam, ok := replay.FamilyOfWeaponID(s.Weapon)
			if !ok {
				continue
			}
			famHex := fmt.Sprintf("%08X", fam)
			if !familles[famHex] {
				continue
			}
			key := registre[fam]
			if key == "" {
				key = "0x" + famHex
			}
			if par[key] == nil {
				par[key] = &acc{}
			}
			a := par[key]
			a.tirs++
			totalTirs++
			serie := idx[fmt.Sprintf("%d|%s", s.Slot, famHex)]
			a.lectures = len(serie)
			var avant, apres *lectureJauge
			meilleur := -1
			for i := range serie {
				d := (serie[i].frame - s.T) * pas
				abs := d
				if abs < 0 {
					abs = -abs
				}
				if meilleur < 0 || abs < meilleur {
					meilleur = abs
				}
				if d <= 0 && (avant == nil || serie[i].frame > avant.frame) {
					avant = &serie[i]
				}
				if d > 0 && (apres == nil || serie[i].frame < apres.frame) {
					apres = &serie[i]
				}
			}
			if meilleur >= 0 {
				a.ecarts = append(a.ecarts, meilleur)
			}
			if avant != nil && apres != nil {
				a.encadres++
				if avant.val != apres.val {
					a.bougent++
				}
			}
		}
	}
	if totalTirs == 0 {
		t.Logf("AUCUN tir d'arme à jauge dans le corpus (%d lectures de jauge au total)", totalLectures)
		return
	}
	t.Logf("LECTURES DE JAUGE PAR ARME (appariement emplacement -> arme par le loadout) :")
	armes := make([]string, 0, len(jaugeParArme))
	for k := range jaugeParArme {
		armes = append(armes, k)
	}
	sort.Slice(armes, func(i, j int) bool { return jaugeParArme[armes[i]] > jaugeParArme[armes[j]] })
	for _, k := range armes {
		t.Logf("  %-24s %4d lecture(s) de jauge", k, jaugeParArme[k])
	}
	t.Logf("ENCADREMENT DE LA JAUGE — %d tirs d'armes à jauge, %d lectures de jauge", totalTirs, totalLectures)
	t.Logf("%-24s %7s %9s %9s %9s %9s %9s", "arme", "tirs", "med(ms)", "p90(ms)", "<500ms", "encadrés", "bougent")
	keys := make([]string, 0, len(par))
	var tousEcarts []int
	sousGlobal, encadresGlobal, bougentGlobal := 0, 0, 0
	for k, a := range par {
		keys = append(keys, k)
		tousEcarts = append(tousEcarts, a.ecarts...)
		for _, e := range a.ecarts {
			if e < 500 {
				sousGlobal++
			}
		}
		encadresGlobal += a.encadres
		bougentGlobal += a.bougent
	}
	sort.Strings(keys)
	for _, k := range keys {
		a := par[k]
		sous := 0
		for _, e := range a.ecarts {
			if e < 500 {
				sous++
			}
		}
		part := 0.0
		if len(a.ecarts) > 0 {
			part = 100 * float64(sous) / float64(len(a.ecarts))
		}
		t.Logf("%-24s %7d %9d %9d %8.1f%% %9d %9d",
			k, a.tirs, mediane(a.ecarts), quantile(a.ecarts, 0.9), part, a.encadres, a.bougent)
	}
	partGlobale := 0.0
	if len(tousEcarts) > 0 {
		partGlobale = 100 * float64(sousGlobal) / float64(len(tousEcarts))
	}
	t.Logf("TOTAL : %d tirs, écart médian %d ms, p90 %d ms, %.2f%% sous 500 ms",
		totalTirs, mediane(tousEcarts), quantile(tousEcarts, 0.9), partGlobale)
	t.Logf("TOTAL : %d tir(s) ENCADRÉS par deux lectures (%.2f%%), dont %d où la jauge BOUGE (%.2f%%)",
		encadresGlobal, 100*float64(encadresGlobal)/float64(totalTirs),
		bougentGlobal, 100*float64(bougentGlobal)/float64(totalTirs))
}
