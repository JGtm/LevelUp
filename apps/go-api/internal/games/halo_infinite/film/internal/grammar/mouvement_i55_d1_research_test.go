//go:build research

package grammar

// mouvement_i55_d1_research_test.go — LA MESURE DE D1 (lot 5.3), SUR LES MINI-BOBINES SEULES.
//
// # LA QUESTION
//
// `consumeBipedPosturePhysics` fait `br.Skip(2)` : il lit le tag de `ti=35 i55` et ne consomme
// AUCUNE des quatre charges que le jeu lit (NOTE_5_3 § 2.4). Si `i55` est parcouru avec un tag
// dont la charge coute des bits, la marche devrait deriver — elle ne le fait pas. Avant toute
// conclusion, il faut le COMPTE et la DISTRIBUTION DU TAG.
//
// # POURQUOI LES MINI-BOBINES, ET CE QU'ELLES PERMETTENT
//
// Un backfill decode le parc pendant ce lot (un decodage a la fois) : aucun film du cache n'est
// lu. Les mini-bobines versionnees portent, d'apres leur `PROVENANCE.txt`, le chunk de REGISTRE,
// douze paquets d'IMAGE-CLE reels et le PIED — et AUCUN paquet de replication. Le chemin DELTA
// du bipede (`dispatch_biped`) n'y est donc pas exercable ; mesure a l'appui,
// `ScanFilmBipedPositions` refuse les sept bobines (« aucun slot biped (ti=35) dans les
// keyframes »). Le chemin d'IMAGE-CLE, lui, marche les composants du bipede un par un par la
// MEME boucle de production (`WalkKeyframeFullState` -> `traverseComponentLoop` -> le meme
// `case "biped-posture-physics-component"`), et c'est la que la mesure se fait.
//
// # COMMENT LE TAG EST LU SANS TOUCHER UN OCTET DE PRODUCTION
//
// La marche publie, pour chaque composant, son `StartBit` ([CompResult]). Le tag est donc relu
// DIRECTEMENT dans le payload a cette position, par `kfReadBits(pay, StartBit, 2)` — la meme
// valeur que le `Skip(2)` a franchie, sans crochet d'observation, donc sans toucher
// `observateur.go` ni `components_probe.go` (qui feraient bouger `grammar.Rev` et les huit
// fixtures de contrat).
//
// # L'ORACLE QUI DONNE SON POIDS A LA MESURE
//
// `EntityTrace.EndBit == borne.Want` : le record FERME, c'est-a-dire que la marche atterrit
// exactement sur le premier bit du record suivant. Un record qui ferme APRES avoir franchi
// `i55` prouve que la charge sautee coute bien ZERO bit pour ce tag-la — c'est une preuve au
// bit pres, pas une impression.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

const (
	d1BipedTI    = 35
	d1NomPosture = "biped-posture-physics-component"
	d1NomCrouch  = "unit-crouch-component"
	d1NomSlide   = "biped-slide-component"
	d1NomMobil   = "biped-mobility-action-component"
)

// d1Compte porte ce qu'une bobine (ou le total) rend a la mesure.
type d1Compte struct {
	records    int            // records ti=35 BORNES examines
	atteintI55 int            // records dont la marche a franchi i55
	tags       map[uint64]int // distribution du tag de 2 bits
	fermes     int            // records qui ferment exactement
	fermesI55  int            // records qui ferment APRES avoir franchi i55
	presence   map[string]int
}

func d1Neuf() *d1Compte {
	return &d1Compte{tags: map[uint64]int{}, presence: map[string]int{}}
}

func (c *d1Compte) absorber(o *d1Compte) {
	c.records += o.records
	c.atteintI55 += o.atteintI55
	c.fermes += o.fermes
	c.fermesI55 += o.fermesI55
	for k, v := range o.tags {
		c.tags[k] += v
	}
	for k, v := range o.presence {
		c.presence[k] += v
	}
}

// TestMouvementI55D1 — LE COMPTE ET LA DISTRIBUTION DU TAG, bobine par bobine.
func TestMouvementI55D1(t *testing.T) {
	total := d1Neuf()
	bobines := 0
	for _, court := range closureMiniFilms() {
		dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court)
		c := d1Bobine(t, dir)
		if c == nil {
			continue
		}
		bobines++
		total.absorber(c)
		t.Logf("%s : %4d records ti=35 bornes · i55 franchi %4d · ferment %d (dont %d apres i55) · tags %s",
			court, c.records, c.atteintI55, c.fermes, c.fermesI55, d1TexteTags(c.tags))
	}
	if bobines == 0 {
		t.Fatalf("aucune mini-bobine lue : `film/replay/testdata/` a bouge")
	}
	t.Logf("PRESENCE DES QUATRE COMPOSANTS DE MOUVEMENT (records ti=35 qui les declarent) : %s",
		d1TextePresence(total.presence, total.records))
	t.Logf("TOTAL (%d bobines) : %d records · i55 franchi %d · ferment %d (dont %d apres i55) · tags %s",
		bobines, total.records, total.atteintI55, total.fermes, total.fermesI55, d1TexteTags(total.tags))
	d1Verdict(t, total)
}

// d1Verdict ecrit la reponse a D1 dans les termes de la question, sans l'arrondir.
func d1Verdict(t *testing.T, c *d1Compte) {
	t.Helper()
	if c.atteintI55 == 0 {
		t.Logf("D1 : NEGATIF MESURE — `i55` n'est franchi par AUCUN des %d records ti=35 bornes "+
			"des mini-bobines. Le `Skip(2)` n'y est jamais exerce : la question se reporte "+
			"telle quelle a la mesure sur film entier (5.3.2).", c.records)
		return
	}
	nonNuls := 0
	for tag, n := range c.tags {
		if tag != 0 {
			nonNuls += n
		}
	}
	if nonNuls == 0 {
		t.Logf("D1 : `i55` est franchi %d fois et son tag vaut TOUJOURS 0. Le `Skip(2)` n'a donc "+
			"jamais saute la charge d'un tag 1, 2 ou 3 sur ces bobines — ce qui explique que la "+
			"marche reste alignee, SANS prouver que la charge du tag 0 soit vide.", c.atteintI55)
	} else {
		t.Logf("D1 : `i55` porte un tag NON NUL %d fois sur %d franchissements. La charge de ce "+
			"tag est lue par le jeu et sautee par le port : la grammaire d'`i55` est incomplete.",
			nonNuls, c.atteintI55)
	}
	d1Uniformite(t, c)
	if c.fermesI55 > 0 {
		t.Logf("D1 (ORACLE) : %d record(s) FERMENT exactement APRES avoir franchi `i55` — la "+
			"largeur totale est juste au bit pres pour ces records, donc la charge sautee y "+
			"coute ZERO bit.", c.fermesI55)
		return
	}
	t.Logf("D1 (ORACLE) : AUCUN record ne ferme apres avoir franchi `i55` (la marche bute plus " +
		"loin, sur un composant non porte) — aucune preuve au bit pres n'est disponible ici.")
}

// d1Uniformite est LE CONTROLE INTERNE DE LA MESURE, et il est necessaire : si le curseur avait
// derive AVANT `i55`, les deux bits relus a `StartBit` seraient du bruit — et du bruit se
// repartit uniformement sur quatre valeurs. Une distribution franchement NON uniforme est donc
// la preuve que ces deux bits sont bien un champ structure, lu au bon endroit.
func d1Uniformite(t *testing.T, c *d1Compte) {
	t.Helper()
	if c.atteintI55 == 0 {
		return
	}
	attendu := float64(c.atteintI55) / 4
	ecart := 0.0
	for tag := uint64(0); tag < 4; tag++ {
		d := float64(c.tags[tag]) - attendu
		ecart += d * d / attendu
	}
	t.Logf("D1 (CONTROLE) : ecart a l'uniforme = %.0f sur %d lectures (un bruit de curseur derive "+
		"rendrait ~4 valeurs equiprobables, donc un ecart proche de 3). Au-dela, les deux bits "+
		"sont un champ structure lu au bon endroit.", ecart, c.atteintI55)
}

// d1Bobine marche les records d'image-cle d'UNE bobine et rend ses comptes.
func d1Bobine(t *testing.T, dir string) *d1Compte {
	t.Helper()
	if _, err := os.Stat(dir); err != nil {
		t.Logf("bobine absente (%s) : %v", dir, err)
		return nil
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Logf("LoadDir %s : %v", dir, err)
		return nil
	}
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Logf("%s : registre illisible : %v", dir, err)
		return nil
	}
	// MEME INSTALLATION QUE LA PRODUCTION : les largeurs du bloc MPP varient par build, et sans
	// elles la marche du bipede est mesuree au decoupage d'un AUTRE build.
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	c := d1Neuf()
	ctx := fc.ContexteDeLecture()
	for _, num := range fc.ChunkNumbers() {
		data, packets, ok := fc.ChunkAt(num)
		if !ok {
			continue
		}
		for _, pk := range packets {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			d1Payload(pk.Payload(data), reg, ctx, c)
		}
	}
	return c
}

// d1Payload marche les records BORNES de ti=35 d'un payload d'image-cle.
func d1Payload(pay []byte, reg *Registry, ctx ContexteDeLecture, c *d1Compte) {
	for _, b := range keyframeBornes(pay) {
		if b.TI != d1BipedTI {
			continue
		}
		c.records++
		tr := WalkKeyframeFullState(pay, b.Bit, reg, ctx)
		ferme := tr.DesyncAt < 0 && tr.EndBit == b.Want
		if ferme {
			c.fermes++
		}
		vuI55 := false
		for _, comp := range tr.Comps {
			switch comp.Name {
			case d1NomCrouch, d1NomSlide, d1NomMobil:
				c.presence[comp.Name]++
			case d1NomPosture:
				c.presence[comp.Name]++
				vuI55 = true
				c.atteintI55++
				c.tags[kfReadBits(pay, comp.StartBit, 2)]++
			}
		}
		if ferme && vuI55 {
			c.fermesI55++
		}
	}
}

// d1TexteTags rend la distribution du tag, tag croissant.
func d1TexteTags(tags map[uint64]int) string {
	if len(tags) == 0 {
		return "(aucun)"
	}
	cles := make([]int, 0, len(tags))
	for k := range tags {
		cles = append(cles, int(k)) //nolint:gosec // le tag tient sur 2 bits
	}
	sort.Ints(cles)
	out := ""
	for _, k := range cles {
		if out != "" {
			out += " "
		}
		out += fmt.Sprintf("t%d=%d", k, tags[uint64(k)]) //nolint:gosec // cle issue d'un uint64
	}
	return out
}

// d1TextePresence rend, par composant, le nombre de records qui le declarent.
func d1TextePresence(p map[string]int, records int) string {
	noms := []string{d1NomCrouch, d1NomMobil, d1NomPosture, d1NomSlide}
	out := ""
	for _, n := range noms {
		if out != "" {
			out += " · "
		}
		pct := 0.0
		if records > 0 {
			pct = 100 * float64(p[n]) / float64(records)
		}
		out += fmt.Sprintf("%s %d (%.1f %%)", n, p[n], pct)
	}
	return out
}
