//go:build research

package replay

// zone_56_camp_research_test.go — LOT 5.6, POINT 2 : UN CANAL DU FILM NOMME-T-IL LE CAMP QUI
// POUSSE UNE RAMPE ?
//
// # CE QUE LE POINT 1 A FERME, ET CE QU IL A OUVERT
//
// L archetype `zones` `ti=23` est REFUTE comme porteur (ecrivain lu, puis mesure : 0 slot
// recense aux 27 images-cles de `396cfc92` et aux 24 de `f75e7053`, quand la MEME marche rend
// 26 et 44 slots pour `ti=13`). Le seul porteur d etat de zone du film est donc `ti=13`
// (`managed-object-property`), deja porte : un slot = UNE propriete nommee, un tag = son type.
//
// La couche de publication tire aujourd hui DEUX canaux de ce sac (tag 3 = la jauge, tag 4 = le
// proprietaire) et DEDUIT `capturingTeam` de l issue de la rampe. La question de ce point est
// donc : parmi TOUS les slots que le film fait parler, y en a-t-il un dont la valeur PENDANT une
// rampe nomme le camp qui pousse — y compris quand la rampe AVORTE ?
//
// # LA METHODE : AUCUNE GRAMMAIRE NEUVE, UN BALAYAGE DE PRODUCTION ET UNE CORRELATION
//
// `grammar.ScanManagedProperties` est le lecteur de production. Cette passe :
//
//	1. range ses lectures scalaires par (slot, tag) sur l axe des MICROSECONDES moteur ;
//	2. decoupe les rampes de chaque slot de jauge avec `findZoneRamps` — LE MEME decoupage que
//	   la publication, donc les memes rampes ;
//	3. pour chaque rampe, lit la valeur de CHAQUE canal candidat pendant [t0, tPeak] ;
//	4. confronte, sur les rampes ABOUTIES (sommet >= `zoneGaugeRampComplete`), cette valeur au
//	   camp que la publication deduit (le proprietaire juste apres le sommet) ;
//	5. dit, sur les rampes AVORTEES, si le canal donne une valeur — c est le GAIN cherche.
//
// Un canal qui vaut le camp attendu sur toutes les rampes abouties ET qui parle sur les
// avortees est le canal cherche. Un canal qui ne parle que sur les abouties n apporte rien de
// plus que la deduction actuelle, et cela se dit.
//
// # LA TOLERANCE EST CELLE DE LA PUBLICATION, PAS UN REGLAGE
//
// La fenetre d appariement et le seuil d aboutissement sont ceux du code de production
// (`zoneGaugeRampComplete`, `zoneOwnerWindowFrames`) : mesurer sous d autres constantes
// mesurerait un autre calque.
//
// # REGIME
//
//	ZONE56P_FILM=<abs>/data/cache/film_chunks/396cfc92 ZONE56P_CARTE=Illusion \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	    -run '^TestZone56CampCandidats$' ./internal/games/halo_infinite/film/replay/
//
// UN SEUL FILM PAR INVOCATION, aucune base DuckDB, aucun artefact ecrit.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/testutil"
)

// zone56Canal identifie un canal candidat : le slot qui parle et le TAG sous lequel il parle.
type zone56Canal struct {
	slot uint32
	tag  int
}

func (c zone56Canal) String() string { return fmt.Sprintf("slot %d tag %d", c.slot, c.tag) }

// zone56Bilan est ce qu un canal candidat rend face aux rampes.
type zone56Bilan struct {
	// abouties : rampes abouties ou le canal porte une valeur ; accord / desaccord avec le camp
	// que la publication deduit.
	abouties, accord, desaccord int
	// avortees : rampes avortees ou le canal porte une valeur — le GAIN cherche.
	avortees int
	// constant : rampes ou la valeur du canal ne CHANGE PAS pendant [t0, tPeak]. Un camp qui
	// pousse est constant sur une rampe ; un canal qui varie n est pas un camp.
	constant int
	// valeurs distinctes vues pendant une rampe (au plus 6 collees au rapport).
	valeurs map[uint64]int
}

// TestZone56CampCandidats — LA CORRELATION.
func TestZone56CampCandidats(t *testing.T) {
	dir, carte := os.Getenv("ZONE56P_FILM"), os.Getenv("ZONE56P_CARTE")
	if dir == "" {
		t.Skip("instrument de mesure : ZONE56P_FILM requis")
	}
	fc := zone56Contexte(t, dir, carte)
	sc, err := grammar.ScanManagedProperties(fc)
	if err != nil {
		t.Fatalf("balayage ti=13 : %v", err)
	}
	t.Logf("%s (carte %s) : %d slots, %d records, %d marches, %d chainees, %d lectures",
		filepath.Base(dir), carte, sc.Slots, sc.Records, sc.Walked, sc.Chained, len(sc.Reads))
	chaine := os.Getenv("ZONE56P_CHAINE") == "1"
	if chaine {
		t.Log("LECTURES CHAINEES SEULEMENT : le temoin de fiabilite par lecture est arme")
	}
	canaux := zone56Canaux(sc.Reads, chaine)
	zone56LogInventaire(t, canaux)
	rampes := zone56Rampes(t, canaux)
	if len(rampes) == 0 {
		// UN MODE SANS JAUGE N EST PAS UNE PANNE : la colline de KOTH n en a pas (mesure du
		// 2026-09-20 : 0 point de jauge, 72 intervalles actifs). L inventaire ci-dessus est
		// alors TOUTE la reponse — il dit si un canal a valeurs de camp existe malgre tout.
		t.Log("AUCUNE RAMPE DE JAUGE : rien a correler, l inventaire des canaux est le resultat")
		t.Logf("canaux a valeurs de camp : %v", zone56CanauxDeCamp(canaux))
		return
	}
	zone56LogCandidats(t, canaux, rampes)
}

// zone56Contexte charge le film avec les LARGEURS D AXE DE LA CARTE installees par le geste de
// production (`poserProfilPuisCarte`), piege documente au lot 5.3.
//
// CARTE VIDE = MESURE SANS CARTE, ET C EST LICITE ICI, DEMONTRE PAR UN A/B. `ti=13` ne porte
// AUCUN composant de position : ses largeurs sont celles du variant (4 bits de tag + une table
// fixe), que la carte ne touche pas. L A/B est colle au journal du lot ; il autorise a mesurer
// un film a zones dont le nom de carte n est pas versionne (le catalogue de noms vit dans la
// base, qu un backfill tient).
func zone56Contexte(t *testing.T, dir, carte string) *grammar.FilmContext {
	t.Helper()
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("chargement %s : %v", dir, err)
	}
	if carte == "" {
		t.Log("SANS CARTE : largeurs d axe auto-detectees — licite pour ti=13 (aucune position)")
		fc := grammar.NewFilmContext(film)
		poserProfilPuisCarte(fc, "lot-5.6-sans-carte", Options{})
		return fc
	}
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	cat, err := profile.LoadMapQuantCatalog(
		title.NewPathResolver(root).MapQuantBoundsPath(title.DefaultSlug))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	entry, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("carte %q hors catalogue : %v", carte, err)
	}
	fc := grammar.NewFilmContextForMap(film, &entry, nil)
	poserProfilPuisCarte(fc, "lot-5.6", Options{})
	return fc
}

// zone56Canaux range les lectures SCALAIRES par (slot, tag), sur l axe des microsecondes moteur.
//
// L axe est celui du film et non la grille de frames : la correlation ne publie rien, elle
// compare des instants entre eux — et la grille exigerait un document assemble.
func zone56Canaux(reads []grammar.ManagedPropertyRead, chaine bool) map[zone56Canal][]zoneSample {
	out := map[zone56Canal][]zoneSample{}
	for _, r := range reads {
		if r.Field != grammar.ManagedPropertyScalar || !r.HasValue {
			continue
		}
		if chaine && !r.Chained {
			continue // TEMOIN DE FIABILITE : le record chaine (cf. ManagedPropertyRead.Chained)
		}
		c := zone56Canal{slot: r.Slot, tag: r.Tag}
		out[c] = append(out[c], zoneSample{t: int(r.TimestampUS / 1000), v: r.Value})
	}
	for c := range out {
		ss := out[c]
		sort.SliceStable(ss, func(i, j int) bool { return ss[i].t < ss[j].t })
		out[c] = ss
	}
	return out
}

// zone56LogInventaire colle l inventaire des canaux : ce que le film fait parler, et combien.
func zone56LogInventaire(t *testing.T, canaux map[zone56Canal][]zoneSample) {
	t.Helper()
	cles := zone56Cles(canaux)
	t.Logf("--- INVENTAIRE DES CANAUX ti=13 (%d couples slot x tag) ---", len(cles))
	t.Log("| slot | tag | emissions | valeurs distinctes | min | max |")
	t.Log("|---:|---:|---:|---:|---:|---:|")
	for _, c := range cles {
		ss := canaux[c]
		vus := map[uint64]bool{}
		mn, mx := ss[0].v, ss[0].v
		for _, s := range ss {
			vus[s.v] = true
			if s.v < mn {
				mn = s.v
			}
			if s.v > mx {
				mx = s.v
			}
		}
		t.Logf("| %d | %d | %d | %d | %d | %d |", c.slot, c.tag, len(ss), len(vus), mn, mx)
	}
}

// zone56JaugeReelle dit si un canal tag 3 est une VRAIE jauge de capture : la MAJORITE de ses
// valeurs vit sur l echelle du jeu, du zero quantifie a l unite (cf. l en-tete de l echelle dans
// zone_states.go).
//
// LA MAJORITE, ET PAS LA TOTALITE — CORRECTION D UNE PREMIERE PASSE TROP STRICTE. Le depot
// documente deja l accident : `7344d24f` porte, sur deux slots de jauge sur trois, UNE emission
// aberrante sous zero (-2,3 et -51,6 unites, la PREMIERE emission du slot). Un filtre qui exige
// toutes les valeurs dans la plage jetait ces deux jauges entieres — mesure du 2026-09-21 : 4
// canaux tag 3 ecartes sur 6, dont ceux de 974 et 858 emissions.
const zone56JaugePartMin = 0.80

func zone56JaugeReelle(ss []zoneSample) bool {
	if len(ss) < zoneRampMinSamples {
		return false
	}
	dans := 0
	for _, s := range ss {
		if s.v >= zoneGaugeQuantZero-zoneGaugeQuantUnit &&
			s.v <= zoneGaugeQuantZero+2*zoneGaugeQuantUnit {
			dans++
		}
	}
	return float64(dans)/float64(len(ss)) >= zone56JaugePartMin
}

// zone56CampLike dit si un canal porte des valeurs de CAMP : un identifiant d equipe court ou
// le neutre. C est la forme du canal de propriete que la publication lit (tag 4, valeurs 0, 1 et
// 0xFFFFFFFF sur les temoins du corpus) — et la garde qui empeche de prendre un identifiant de
// chaine de 32 bits pour un camp.
func zone56CampLike(ss []zoneSample) bool {
	for _, s := range ss {
		if s.v > 7 && s.v != zoneNeutralOwner {
			return false
		}
	}
	return len(ss) > 0
}

// zone56Rampes decoupe les rampes des slots de jauge REELS, AVEC LE DECOUPAGE DE PRODUCTION.
func zone56Rampes(t *testing.T, canaux map[zone56Canal][]zoneSample) []zoneRamp {
	t.Helper()
	var out []zoneRamp
	for _, c := range zone56Cles(canaux) {
		if c.tag != grammar.ManagedPropertyTagQuant {
			continue
		}
		if !zone56JaugeReelle(canaux[c]) {
			t.Logf("  slot %d tag 3 ECARTE : moins de %.0f %% de ses %d emissions sur l echelle de jauge",
				c.slot, 100*zone56JaugePartMin, len(canaux[c]))
			continue
		}
		out = append(out, findZoneRamps(c.slot, canaux[c])...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].tPeak < out[j].tPeak })
	nAbouties := 0
	for _, r := range out {
		if gaugeProgressOf(r.top) >= zoneGaugeRampComplete {
			nAbouties++
		}
	}
	t.Logf("--- %d RAMPES (%d abouties au seuil %.2f, %d avortees) ---",
		len(out), nAbouties, zoneGaugeRampComplete, len(out)-nAbouties)
	return out
}

// zone56LogCandidats cherche, JAUGE PAR JAUGE, le COUPLE (canal candidat, canal de reference)
// qui explique le mieux les rampes de cette jauge.
//
// AUCUN APPARIEMENT N EST SUPPOSE, ET C EST LA CORRECTION DE DEUX PASSES FAUSSES. La premiere
// confrontait un canal a TOUTES les rampes du film (une propriete de la zone A jugee sur les
// rampes de la zone B). La seconde appariait la jauge au canal qui BASCULE le plus apres ses
// rampes — et ce critere a choisi un canal d une AUTRE zone pour la jauge 1608, parce qu une
// zone tres disputee bascule souvent, quelle que soit la rampe regardee. Cette passe essaie
// donc TOUS les couples ordonnes de canaux a valeurs de camp, et laisse la mesure designer.
//
// LE CRITERE : le candidat, LU PENDANT la rampe, doit valoir ce que la reference prend JUSTE
// APRES le sommet — c est-a-dire exactement ce que la publication deduit aujourd hui. Zero
// desaccord, et une valeur sur les rampes AVORTEES : c est le gain cherche.
func zone56LogCandidats(t *testing.T, canaux map[zone56Canal][]zoneSample, rampes []zoneRamp) {
	t.Helper()
	camps := zone56CanauxDeCamp(canaux)
	t.Logf("--- %d CANAUX A VALEURS DE CAMP : %v ---", len(camps), camps)
	slots := map[uint32]bool{}
	for _, r := range rampes {
		slots[r.slot] = true
	}
	for _, sl := range sortedZoneSlots(slots) {
		zone56LogJauge(t, canaux, camps, sl, zone56RampesDe(rampes, sl))
	}
}

// zone56RampesDe rend les rampes d UNE jauge.
func zone56RampesDe(rampes []zoneRamp, slot uint32) []zoneRamp {
	var out []zoneRamp
	for _, r := range rampes {
		if r.slot == slot {
			out = append(out, r)
		}
	}
	return out
}

// zone56LogJauge essaie tous les couples ordonnes sur les rampes d UNE jauge et colle le
// classement.
func zone56LogJauge(t *testing.T, canaux map[zone56Canal][]zoneSample, camps []zone56Canal,
	sl uint32, mesRampes []zoneRamp,
) {
	t.Helper()
	nA := 0
	for _, r := range mesRampes {
		if gaugeProgressOf(r.top) >= zoneGaugeRampComplete {
			nA++
		}
	}
	t.Logf("--- JAUGE slot %d : %d rampes (%d abouties, %d avortees) ---",
		sl, len(mesRampes), nA, len(mesRampes)-nA)
	type ligne struct {
		cand, ref zone56Canal
		b         *zone56Bilan
	}
	var lignes []ligne
	for _, cand := range camps {
		for _, ref := range camps {
			if cand == ref {
				continue
			}
			b := zone56Bilan1(canaux[cand], canaux[ref], mesRampes)
			if b.abouties < 2 {
				continue
			}
			lignes = append(lignes, ligne{cand, ref, b})
		}
	}
	sort.Slice(lignes, func(i, j int) bool {
		a, b := lignes[i].b, lignes[j].b
		if (a.desaccord == 0) != (b.desaccord == 0) {
			return a.desaccord == 0
		}
		if a.accord != b.accord {
			return a.accord > b.accord
		}
		return a.avortees > b.avortees
	})
	t.Log("| candidat (pendant la rampe) | reference (apres le sommet) | abouties | accord | desaccord | avortees vues | constant | valeurs |")
	t.Log("|---|---|---:|---:|---:|---:|---:|---|")
	n := len(lignes)
	if n > 6 {
		n = 6
	}
	for _, l := range lignes[:n] {
		t.Logf("| %s | %s | %d | %d | %d | %d | %d | %s |",
			l.cand, l.ref, l.b.abouties, l.b.accord, l.b.desaccord, l.b.avortees,
			l.b.constant, zone56Valeurs(l.b.valeurs))
	}
}

// zone56CanauxDeCamp rend les canaux dont les valeurs sont des identifiants de CAMP.
func zone56CanauxDeCamp(canaux map[zone56Canal][]zoneSample) []zone56Canal {
	var out []zone56Canal
	for _, c := range zone56Cles(canaux) {
		if c.tag != grammar.ManagedPropertyTagU32 || !zone56CampLike(canaux[c]) {
			continue
		}
		out = append(out, c)
	}
	return out
}

// zone56Bilan1 confronte UN canal aux rampes d UNE jauge, contre UN canal de reference.
func zone56Bilan1(ss, ref []zoneSample, rampes []zoneRamp) *zone56Bilan {
	b := &zone56Bilan{valeurs: map[uint64]int{}}
	for _, r := range rampes {
		v, cst, ok := zone56ValeurPendant(ss, r)
		if !ok {
			continue
		}
		b.valeurs[v]++
		if cst {
			b.constant++
		}
		if gaugeProgressOf(r.top) < zoneGaugeRampComplete {
			b.avortees++
			continue
		}
		attendu, connu := zone56ValeurApres(ref, r.tPeak)
		if !connu {
			continue
		}
		b.abouties++
		if v == attendu {
			b.accord++
		} else {
			b.desaccord++
		}
	}
	return b
}

// zone56ValeurPendant rend la valeur du canal PENDANT la rampe, et si elle y est constante.
// La valeur retenue est la DERNIERE emission dans [t0, tPeak] ; sans emission dans la fenetre,
// le canal ne parle pas de cette rampe.
func zone56ValeurPendant(ss []zoneSample, r zoneRamp) (uint64, bool, bool) {
	var v uint64
	n, cst := 0, true
	for _, s := range ss {
		if s.t < r.t0 || s.t > r.tPeak {
			continue
		}
		if n > 0 && s.v != v {
			cst = false
		}
		v, n = s.v, n+1
	}
	return v, cst, n > 0
}

// zone56ValeurApres rend la premiere valeur a t ou apres.
func zone56ValeurApres(ss []zoneSample, t int) (uint64, bool) {
	for _, s := range ss {
		if s.t >= t {
			return s.v, true
		}
	}
	return 0, false
}

// zone56Cles rend les canaux tries — determinisme du rapport.
func zone56Cles(m map[zone56Canal][]zoneSample) []zone56Canal {
	out := make([]zone56Canal, 0, len(m))
	for c := range m {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].slot != out[j].slot {
			return out[i].slot < out[j].slot
		}
		return out[i].tag < out[j].tag
	})
	return out
}

// zone56Valeurs colle au plus six valeurs, les plus fréquentes d abord.
func zone56Valeurs(m map[uint64]int) string {
	vals := make([]uint64, 0, len(m))
	for v := range m {
		vals = append(vals, v)
	}
	sort.Slice(vals, func(i, j int) bool { return m[vals[i]] > m[vals[j]] })
	n := len(vals)
	if n > 6 {
		n = 6
	}
	var s string
	for _, v := range vals[:n] {
		s += fmt.Sprintf(" %d(x%d)", v, m[v])
	}
	if len(vals) > n {
		s += fmt.Sprintf(" +%d", len(vals)-n)
	}
	return s
}
