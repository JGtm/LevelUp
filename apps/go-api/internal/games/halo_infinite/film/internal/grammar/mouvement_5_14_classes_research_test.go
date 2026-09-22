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
	"os"
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

// c514Controle est ce qu UNE entree de controle porte, valeurs lues.
type c514Controle struct {
	indexPresent bool
	indexCdc04   int // le R(7) de FUN_1406cdc04, -1 quand absent
	indexCtrl    int // le R(5) de FUN_1406d0388 : l index de controle, 0..31
	blocPresent  bool
	secondPres   bool
	second       int // le R(2) de FUN_1406cd860, -1 quand absent
	scalaireA    int // le premier code 6 bits de FUN_1406d6ef4
	scalaireB    int // le second
	troisieme    bool
	champPile    bool
	brancheLong  bool
	actionsPres  bool
	secondBloc   bool // le bit `b` de FUN_1406d0388 (FUN_141fdae44)
	porte        bool
}

// c514LireControle rejoue `FUN_1406d0388` EN RENDANT SES VALEURS. Elle emploie les MEMES
// constantes de largeur que le port (`frame_vue_controle.go`) : aucun litteral n est recopie.
func c514LireControle(br *Lecteur, frameLen int) c514Controle {
	out := c514Controle{indexCdc04: -1, second: -1, scalaireA: -1, scalaireB: -1}
	if !placeDisponible(br, frameLen, 1) {
		return out
	}
	if out.indexPresent = br.ReadBit(); out.indexPresent {
		if !placeDisponible(br, frameLen, largeurIndexCdc04) {
			return out
		}
		out.indexCdc04 = int(br.ReadBits(largeurIndexCdc04))
	}
	if !placeDisponible(br, frameLen, largeurIndexControle+1) {
		return out
	}
	out.indexCtrl = int(br.ReadBits(largeurIndexControle))
	if out.blocPresent = br.ReadBit(); out.blocPresent {
		if !placeDisponible(br, frameLen, 1) {
			return out
		}
		if out.secondPres = br.ReadBit(); out.secondPres {
			if !placeDisponible(br, frameLen, largeurCourteControle) {
				return out
			}
			out.second = int(br.ReadBits(largeurCourteControle))
		}
		if !placeDisponible(br, frameLen, 2*LargeurScalaireAnalogique+3) {
			return out
		}
		out.scalaireA = int(br.ReadBits(LargeurScalaireAnalogique))
		out.scalaireB = int(br.ReadBits(LargeurScalaireAnalogique))
		out.troisieme = br.ReadBit()
		out.champPile = br.ReadBit()
		out.brancheLong = br.ReadBit()
		if out.troisieme || out.champPile || out.brancheLong {
			return out
		}
		if !placeDisponible(br, frameLen, 1) {
			return out
		}
		out.actionsPres = br.ReadBit()
		if out.actionsPres {
			return out
		}
	}
	if !placeDisponible(br, frameLen, 1) {
		return out
	}
	out.secondBloc = br.ReadBit()
	out.porte = !out.secondBloc
	return out
}

// TestClasses514Contenu PUBLIE CE QUE LES VUES A ET C PORTENT, en clair. Fenetre optionnelle par
// `MOUV511_T0` / `MOUV511_T1` (en secondes depuis le premier paquet delta).
func TestClasses514Contenu(t *testing.T) {
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
	t0, t1 := c514Fenetre()

	var paquets, dansFenetre int
	genresA := map[int]int{}
	var aNonVide int
	idxCtrl, cdc04, seconds := map[int]int{}, map[int]int{}, map[int]int{}
	scalA, scalB := map[int]int{}, map[int]int{}
	var blocs, actions, secondsBlocs, troisiemes, champsPile, branchesLong int
	var origine uint64
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
			if origine == 0 {
				origine = pk.TimestampUS
			}
			sec := float64(pk.TimestampUS-origine) / 1e6
			paquets++
			if sec < t0 || sec > t1 {
				continue
			}
			dansFenetre++
			pay := pk.Payload(data)
			frameLen := len(pay) * 8
			br := LecteurSur(pay)
			br.poserCadre(cfg)
			if _, present := PacketHeadEventType(pay); present {
				d := marchLocateStrict(pay, w, cfg)
				if d < 0 {
					continue
				}
				br.Skip(d)
			} else {
				br.Skip(DefaultPacketPreambleBits - 1)
				a := consumeVueA(br, frameLen)
				if !a.Vide {
					aNonVide++
					for _, g := range a.Genres {
						genresA[g]++
					}
					continue
				}
			}
			w.PoserVueCourante(int(vueDeLImageCle))
			if _, _, hitEnd := decodeInferLoop(br, pay, w, cfg); !hitEnd {
				continue
			}
			// la vue C, entree par entree
			for tour := 0; tour < 64; tour++ {
				if !placeDisponible(br, frameLen, 1) || !br.ReadBit() {
					break
				}
				if !placeDisponible(br, frameLen, LargeurKindVueC) {
					break
				}
				k := int(br.ReadBits(LargeurKindVueC))
				if k == kindVueCNeant {
					continue
				}
				if k != kindVueCControle {
					break
				}
				e := c514LireControle(br, frameLen)
				if e.indexPresent {
					cdc04[e.indexCdc04]++
				}
				idxCtrl[e.indexCtrl]++
				if !e.blocPresent {
					if !e.porte {
						break
					}
					continue
				}
				blocs++
				if e.secondPres {
					seconds[e.second]++
				}
				scalA[e.scalaireA]++
				scalB[e.scalaireB]++
				if e.troisieme {
					troisiemes++
				}
				if e.champPile {
					champsPile++
				}
				if e.brancheLong {
					branchesLong++
				}
				if e.actionsPres {
					actions++
				}
				if e.secondBloc {
					secondsBlocs++
				}
				if !e.porte {
					break
				}
			}
		}
	}
	t.Logf("PAQUETS DELTA : %d, dont %d dans la fenetre [%.3f ; %.3f] s", paquets, dansFenetre,
		t0, t1)
	t.Logf("VUE A (rang 0) : non vide sur %d paquets · genres R(7) : %s", aNonVide,
		c514Hist(genresA))
	t.Logf("VUE C (rang 2) — L ENTREE DE CONTROLE :")
	t.Logf("  index de controle R(5)        : %s", c514Hist(idxCtrl))
	t.Logf("  index FUN_1406cdc04 R(7)      : %s", c514Hist(cdc04))
	t.Logf("  bloc de 0x68 present          : %d fois · second champ R(2) : %s", blocs,
		c514Hist(seconds))
	t.Logf("  scalaire A (6 bits)           : %s", c514Hist(scalA))
	t.Logf("  scalaire B (6 bits)           : %s", c514Hist(scalB))
	t.Logf("  gardes ouvertes NON PORTEES   : troisieme champ %d · champ de pile %d · "+
		"branche longue %d · BITS D ACTION %d · second bloc (0xbc) %d",
		troisiemes, champsPile, branchesLong, actions, secondsBlocs)
}

// c514Fenetre lit la fenetre temporelle de l instrument (`MOUV511_T0` / `MOUV511_T1`).
func c514Fenetre() (float64, float64) {
	t0, t1 := 0.0, 1e9
	if v := os.Getenv("MOUV511_T0"); v != "" {
		fmt.Sscanf(v, "%f", &t0)
	}
	if v := os.Getenv("MOUV511_T1"); v != "" {
		fmt.Sscanf(v, "%f", &t1)
	}
	return t0, t1
}
