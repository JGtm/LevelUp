//go:build research

package grammar

// mouvement_5_14_classes_research_test.go — LES TROIS CLASSES DE VUE, CHACUNE SOUS SA PROPRE
// GRAMMAIRE (lot 5.14). Instrument de LECTURE : il n infere rien, il applique ce que
// l ecrivain ecrit et compte ce qui reste.
//
// # L ORDRE DES RANGS EST PROUVE CHEZ L ECRIVAIN (et non suppose)
//
// `FUN_141f855b4` enregistre les trois vues du conteneur par `FUN_1409c9860(conteneur+8, rang,
// vue)` :
//
//	rang 0 -> conteneur + 0x3ce98 · vtable 0x1436a8700 · vtable[0x40] = FUN_14076a1c4   (vue A)
//	rang 1 -> conteneur + 0x21b70 · vtable 0x1436a87e0 · vtable[0x40] = FUN_1406cd128   (vue B)
//	rang 2 -> conteneur + 0x3d2d8 · vtable 0x1436a8770 · vtable[0x40] = FUN_1406cf548   (vue C)
//
// et `FUN_1409c9860` clot le rang dans `*(int *)(vue + 8)` — le champ que `FUN_142f2e174` ecrit
// dans les deux bits de tete d un identifiant d image-cle (lot 5.13.1). Le rang 1 mesure sur les
// images-cles des deux temoins EST donc la vue B, le gestionnaire d entites : la seule grammaire
// que le depot porte aujourd hui.
//
// # LES GRAMMAIRES, TELLES QUE L ECRIVAIN LES ECRIT
//
//	vue A  FUN_14076a1c4 : si `vue[0x11]` -> ZERO bit. Sinon boucle { R(1) ; 0 -> fin ;
//	                       corps FUN_14080a9d4 }. `*param_6 = 0` : elle ne rend JAMAIS un record.
//	                       corps = R(7) `genre` (< 0x7b = 123) puis la charge du genre par
//	                       `handler->vtable[0x68]`, puis `si HasExtraFields et R(1) : R(32)`.
//	vue B  FUN_1406cd128 : le gestionnaire d entites (deja porte : `decodeInferLoop`).
//	vue C  FUN_1406cf548 : prologue `FUN_142f2539c` — ZERO bit : il recopie le TAMPON
//	                       (`memcpy` de `reader+8` sur `reader+0x18` octets,
//	                       `replication_control_view.cpp:421`). Puis boucle { R(1) ; 0 -> fin ;
//	                       `kind` R(2) ; 0 -> FUN_1406d0388 · 1 -> FUN_142f29b38 ·
//	                       2 -> FUN_142f29e54 · 3 -> ZERO bit }.
//
// # CE QUE CET INSTRUMENT MESURE
//
// Il lit le bit de configuration (`FUN_1406cf008` en tete de `FUN_142987460`), puis la vue A a
// sa grammaire, puis la vue B, puis la vue C a sa grammaire — et il publie, PAR PAQUET : la vue
// A est-elle vide (un bit) ou non (et alors quel genre R(7)), combien d iterations la vue C
// fait, quel `kind` chacune porte, et combien de bits restent quand la vue C a lu son en-tete.
// Ce dernier chiffre BORNE la charge du handler : c est lui qui dit de combien de bits la
// grammaire manquante a besoin.
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestClasses514' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// c514AConfigBits est le bit de configuration du frame-processeur : `FUN_142987460` fait
// `DAT_144706104 = FUN_1406cf008(param_2)` AVANT sa boucle sur les trois vues. UN bit, etabli
// par le desassemblage (c est la moitie prouvee de [DefaultPacketPreambleBits]).
const c514AConfigBits = 1

// vueB514 est l index sous lequel le MONDE HORS LIGNE range les entites du gestionnaire
// d entites : `vueDeLImageCle` vaut 0, donc la marche de la vue B s annonce en 0. Le RANG du
// film pour cette meme vue est 1 (`FUN_141f855b4`) — les deux numerotations ne coincident pas
// encore, et c est le point 5.14.3 qui les aligne.
const vueB514 = 0

// vueC514 est l index d annonce de la vue C dans le monde hors ligne : aucun slot ne lui est
// attribue, donc sa garde rejette tout — ce qui est exactement ce que l ecrivain fait, la vue C
// n ayant pas de table d entites.
const vueC514 = 2

// c514VueAVide lit la vue A (rang 0) et rend le genre de son premier corps, ou -1 quand la vue
// est VIDE — le cas ou elle coute EXACTEMENT UN BIT.
//
// C EST LA MOITIE NON LOCALISEE DE [DefaultPacketPreambleBits] : le depot etablissait par la
// MESURE un second bit d amorce que le desassemblage ne montrait pas. Ce bit est le terminateur
// `R(1) = 0` de la vue A vide.
func c514VueAVide(br *Lecteur) (genre int, vide bool) {
	if !br.ReadBit() {
		return -1, true
	}
	return int(br.ReadBits(7)), false
}

// c514IterC est une iteration de la boucle de la vue C : son `kind` R(2), et les bits restants
// du payload au moment ou l en-tete est lu.
type c514IterC struct {
	kind  int
	reste int
}

// c514VueC lit la vue C (rang 2) SANS charge utile : elle s arrete au premier `kind` qui en
// porte une (0, 1, 2) et rend les bits restants. `kind == 3` ne coute rien et la boucle
// continue. `ferme` dit que la vue a lu son terminateur.
func c514VueC(br *Lecteur, frameLen int) (iters []c514IterC, ferme bool) {
	for i := 0; i < 64; i++ {
		if br.BitPos() >= frameLen {
			return iters, false
		}
		if !br.ReadBit() {
			return iters, true
		}
		if br.BitPos()+2 > frameLen {
			return iters, false
		}
		k := int(br.ReadBits(2))
		iters = append(iters, c514IterC{kind: k, reste: frameLen - br.BitPos()})
		if k != 3 { // 0/1/2 portent une charge que ce lot n a pas encore lue
			return iters, false
		}
	}
	return iters, false
}

// TestClasses514Marche applique les TROIS grammaires, une par rang, et publie ce que chaque
// classe porte.
func TestClasses514Marche(t *testing.T) {
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	cfg := fc.CadreDeBalayage()
	w := NewWorld(reg)

	var paquets, evenements, nonLocalise int
	var aVide, aPleine int
	genresA := map[int]int{}
	var bFerme int
	var cFermeVide, cAvecIter int
	kindsC := map[int]int{}
	resteC := map[int]int{}
	nbIterC := map[int]int{}

	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			frameLen := len(pay) * 8
			br := LecteurSur(pay)
			br.poserCadre(cfg)
			if _, present := PacketHeadEventType(pay); present {
				// Paquet a liste d evenements : la tete n est pas l amorce du
				// frame-processeur. On se cale la ou la vue B commence, comme le gate.
				d := marchLocateStrict(pay, w, cfg)
				if d < 0 {
					nonLocalise++
					continue
				}
				evenements++
				br.Skip(d)
			} else {
				br.Skip(c514AConfigBits)
				g, vide := c514VueAVide(br)
				if vide {
					aVide++
				} else {
					aPleine++
					genresA[g]++
					paquets++
					continue // charge du genre non lue : le paquet s arrete ici
				}
			}
			paquets++
			// La vue B se marche sous l index de vue du MONDE HORS LIGNE (0 : cf.
			// `vueDeLImageCle`), qui n est pas encore le rang du film (1). Le port du
			// point 5.14.3 aligne les deux ; l instrument, lui, mesure avec le monde tel
			// qu il est aujourd hui.
			w.PoserVueCourante(vueB514)
			_, _, hitEnd := decodeInferLoop(br, pay, w, cfg)
			if !hitEnd {
				continue
			}
			bFerme++
			w.PoserVueCourante(vueC514)
			iters, ferme := c514VueC(br, frameLen)
			nbIterC[len(iters)]++
			if ferme && len(iters) == 0 {
				cFermeVide++
				continue
			}
			cAvecIter++
			for _, it := range iters {
				kindsC[it.kind]++
				if it.kind != 3 {
					resteC[it.reste]++
				}
			}
		}
	}

	t.Logf("PAQUETS DELTA MARCHES : %d (dont %d a liste d evenements, %d non localises)",
		paquets, evenements, nonLocalise)
	t.Logf("VUE A (rang 0, FUN_14076a1c4) : VIDE (un bit) sur %d paquets · NON VIDE sur %d",
		aVide, aPleine)
	if len(genresA) > 0 {
		t.Logf("  genres R(7) de la vue A : %s", c514Hist(genresA))
	}
	t.Logf("VUE B (rang 1) : terminateur atteint sur %d paquets", bFerme)
	t.Logf("VUE C (rang 2, FUN_1406cf548) : VIDE (un bit) sur %d paquets · au moins une "+
		"iteration sur %d", cFermeVide, cAvecIter)
	t.Logf("  iterations par paquet : %s", c514Hist(nbIterC))
	t.Logf("  kind R(2) : %s", c514Hist(kindsC))
	t.Logf("  BITS RESTANTS quand l en-tete de la vue C est lu (borne de la charge) : %s",
		c514Hist(resteC))
}

// c514Hist rend un histogramme trie, tronque a 24 classes.
func c514Hist(m map[int]int) string {
	cles := make([]int, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	var parts []string
	for i, k := range cles {
		if i >= 24 {
			parts = append(parts, fmt.Sprintf("... (%d classes)", len(cles)))
			break
		}
		parts = append(parts, fmt.Sprintf("%d : %d", k, m[k]))
	}
	return strings.Join(parts, " · ")
}

// TestClasses514Bourrage TRANCHE la nature du reste d un paquet ferme : le bourrage d octet est
// ecrit A ZERO, donc un reste dont TOUS les bits sont nuls est du bourrage, et un reste qui
// porte un 1 est de la GRAMMAIRE MANQUANTE — meme quand il tient dans les sept bits que le gate
// du lot 5.11.6 tolere.
//
// C EST LE RAFFINEMENT QUE LE LOT 5.14 APPORTE AU GATE : « reste dans [0 ; 7] » ne prouve pas
// qu un paquet est lu ; « reste dans [0 ; 7] ET tous ses bits a zero » le prouve.
func TestClasses514Bourrage(t *testing.T) {
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	bal := fc.ProfilDeBalayage()
	bal.Grammaire.ClassesDeVue = true
	fc.PoserProfilDeBalayage(bal)
	cfg := fc.CadreDeBalayage()
	w := NewWorld(reg)

	var paquets, nonLocalise int
	var bourrage, residuNonNul, hors int
	parReste := map[int]int{}
	parResteNonNul := map[int]int{}
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := DefaultPacketPreambleBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					nonLocalise++
					continue
				}
			}
			paquets++
			_, _, curseur := DecodeFrameViewsCurseur(pay, w, cfg, 3, debut)
			reste := len(pay)*8 - curseur
			if reste < 0 || reste > m5116GateOctet {
				hors++
				continue
			}
			parReste[reste]++
			if c514ResteNul(pay, curseur) {
				bourrage++
				continue
			}
			residuNonNul++
			parResteNonNul[reste]++
		}
	}
	t.Logf("PAQUETS : %d (%d non localises) · %d hors de [0 ; 7]", paquets, nonLocalise, hors)
	t.Logf("RESTE DANS [0 ; 7] : %d dont %d A ZERO (bourrage prouve) et %d PORTANT UN 1 "+
		"(grammaire manquante)", bourrage+residuNonNul, bourrage, residuNonNul)
	t.Logf("  reste par largeur            : %s", c514Hist(parReste))
	t.Logf("  reste NON NUL par largeur    : %s", c514Hist(parResteNonNul))
}

// c514ResteNul dit si tous les bits du payload a partir de `curseur` valent zero.
func c514ResteNul(pay []byte, curseur int) bool {
	for i := curseur; i < len(pay)*8; i++ {
		if pay[i/8]&(1<<uint(7-i%8)) != 0 {
			return false
		}
	}
	return true
}

// TestClasses514Restes NOMME les paquets qui ne ferment pas : a quel RANG la marche s arrete, et
// pourquoi. C est le registre de ce qui reste a lire apres le lot 5.14.
func TestClasses514Restes(t *testing.T) {
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	bal := fc.ProfilDeBalayage()
	bal.Grammaire.ClassesDeVue = true
	fc.PoserProfilDeBalayage(bal)
	cfgC := fc.CadreDeBalayage()
	w := NewWorld(reg)

	causes := map[string]int{}
	resteParCause := map[string]int{}
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := DefaultPacketPreambleBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfgC); debut < 0 {
					continue
				}
			}
			_, _, curseur := DecodeFrameViewsCurseur(pay, w, cfgC, 3, debut)
			reste := len(pay)*8 - curseur
			if reste >= 0 && reste <= m5116GateOctet {
				continue
			}
			cause, r := c514Cause(pay, w, cfgC, debut)
			causes[cause]++
			resteParCause[cause] += r
		}
	}
	noms := make([]string, 0, len(causes))
	for k := range causes {
		noms = append(noms, k)
	}
	sort.Strings(noms)
	t.Logf("PAQUETS QUI NE FERMENT PAS — PAR CAUSE :")
	for _, n := range noms {
		t.Logf("  %-44s %5d paquets · %d bits de reste au total", n, causes[n], resteParCause[n])
	}
}

// c514Cause rejoue un paquet rang par rang et nomme l endroit ou la marche s arrete.
func c514Cause(pay []byte, w *World, cfg FrameConfig, debut int) (string, int) {
	frameLen := len(pay) * 8
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	if debut == DefaultPacketPreambleBits {
		br.Skip(DefaultPacketPreambleBits - 1)
		a := consumeVueA(br, frameLen)
		if !a.Porte {
			return fmt.Sprintf("rang 0 vue A : genre %v non porte", a.Genres),
				frameLen - br.BitPos()
		}
	} else {
		br.Skip(debut)
	}
	w.PoserVueCourante(int(vueDeLImageCle))
	_, _, hitEnd := decodeInferLoop(br, pay, w, cfg)
	if !hitEnd {
		return "rang 1 vue B : desynchronisation", frameLen - br.BitPos()
	}
	c := consumeVueC(br, frameLen)
	if !c.Porte {
		return fmt.Sprintf("rang 2 vue C : kinds %v non portes", c.Kinds), frameLen - br.BitPos()
	}
	return "les trois rangs portes, reste hors bourrage", frameLen - br.BitPos()
}

// TestClasses514IdLow BALAYE `IDLowBits` sous la grammaire des trois classes. Il existe parce
// que la fermeture d un paquet est le SEUL gate, et que `IDLowBits` est une valeur de RUNTIME
// qui differe d un film a l autre (11 sur `000d5950`, 14 sur le film de la capture live) : un
// reste de mille bits par paquet se lit comme une grammaire manquante alors qu il peut n etre
// qu un en-tete de record mal cadre. C est le piege 10 de la passation 5.11 — suspecter
// l instrument d abord.
func TestClasses514IdLow(t *testing.T) {
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	bal := fc.ProfilDeBalayage()
	bal.Grammaire.ClassesDeVue = true
	fc.PoserProfilDeBalayage(bal)
	base := fc.CadreDeBalayage()

	t.Logf("BALAYAGE DE `IDLowBits` SOUS LES TROIS CLASSES DE VUE :")
	for low := 10; low <= 15; low++ {
		cfg := base
		cfg.IDLowBits = low
		w := NewWorld(reg)
		var paquets, ferme, nul, ti35, desync int
		for _, c := range fc.ChunkNumbers() {
			data, pks, ok := fc.ChunkAt(c)
			if !ok {
				continue
			}
			m533bLierMonde(w, data, pks)
			for _, pk := range pks {
				if pk.Type != PacketTypeDelta || pk.Size < 1 {
					continue
				}
				pay := pk.Payload(data)
				debut := DefaultPacketPreambleBits
				if _, present := PacketHeadEventType(pay); present {
					if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
						continue
					}
				}
				paquets++
				recs, _, curseur := DecodeFrameViewsCurseur(pay, w, cfg, 3, debut)
				reste := len(pay)*8 - curseur
				if reste >= 0 && reste <= m5116GateOctet {
					ferme++
					if c514ResteNul(pay, curseur) {
						nul++
					}
				}
				for _, r := range recs {
					if r.TypeIndex != BipedTypeIndex {
						continue
					}
					ti35++
					if r.DesyncAt >= 0 {
						desync++
					}
				}
			}
		}
		t.Logf("  idLow %2d : %6d paquets · FERMES %6d (%5.2f %%) dont %6d a reste NUL · "+
			"ti=35 %7d (%d desynchronises)", low, paquets, ferme,
			m533bPart(ferme, paquets), nul, ti35, desync)
	}
}
