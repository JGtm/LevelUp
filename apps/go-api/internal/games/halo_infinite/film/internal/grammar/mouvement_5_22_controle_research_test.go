//go:build research

package grammar

// mouvement_5_22_controle_research_test.go — LE CANAL D ENTREE, DATE (lot 5.22.1).
//
// D2 (5.14) : la garde des BITS D ACTION de la vue de controle (`FUN_1406d025c`) s ouvre 108 fois
// sur `bfecd02b` et JAMAIS sur `dad793c7`. Le saut est une ENTREE ; si le film porte un
// declencheur, c est le premier endroit ou le chercher.
//
// Cet instrument DATE ces ouvertures sur l horloge du film et les confronte aux decollages du
// temoin. Il lit la vue C a la main, comme `TestClasses514Contenu`, parce que le port
// (`consumeControleVueC`) s arrete sur la garde au lieu de publier ce qu il a lu.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// m522Entree est UNE entree de controle lue, datee.
type m522Entree struct {
	rel float64
	ctl c514Controle
}

// m522CausePaquet dit pourquoi un paquet delta n a rendu aucune entree de controle. LA COUVERTURE
// EST LA PREMIERE MESURE : « zero ouverture dans la fenetre » ne veut rien dire tant qu on n a pas
// dit combien de paquets de cette fenetre atteignent seulement la vue C (film DENSE, trou du
// rang 1 des lots 5.15-5.21).
type m522CausePaquet int

const (
	m522PaquetLu       m522CausePaquet = iota // la vue C a ete atteinte
	m522PaquetNonLocal                        // paquet a evenements dont le corps n est pas localise
	m522PaquetVueA                            // la vue A n est pas vide : le port s arrete
	m522PaquetRang1                           // le rang 1 ne ferme pas sa liste (trou du film dense)
)

// m522Couverture compte les paquets par cause.
type m522Couverture map[m522CausePaquet]int

// m522LireControles deroule le film sous la grammaire des trois classes de vue et rend toutes les
// entrees de controle datees, plus l origine de l horloge relative.
func m522LireControles(t *testing.T, couv map[float64]m522CausePaquet) ([]m522Entree, uint64) {
	t.Helper()
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
	var out []m522Entree
	var origine uint64
	cov := m522Couverture{}
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
			rel := float64(pk.TimestampUS-origine) / 1e6
			es, cause := m522ControlesDuPaquet(pk.Payload(data), w, cfg, rel)
			out = append(out, es...)
			cov[cause]++
			couv[rel] = cause
		}
	}
	t.Logf("COUVERTURE DES PAQUETS DELTA : %d lus jusqu a la vue C · %d non localises · "+
		"%d vue A non vide · %d rang 1 ouvert", cov[m522PaquetLu], cov[m522PaquetNonLocal],
		cov[m522PaquetVueA], cov[m522PaquetRang1])
	return out, origine
}

// m522ControlesDuPaquet lit les entrees de controle d UN paquet delta.
func m522ControlesDuPaquet(pay []byte, w *World, cfg FrameConfig,
	rel float64) ([]m522Entree, m522CausePaquet) {
	frameLen := len(pay) * 8
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	if _, present := PacketHeadEventType(pay); present {
		d := marchLocateStrict(pay, w, cfg)
		if d < 0 {
			return nil, m522PaquetNonLocal
		}
		br.Skip(d)
	} else {
		br.Skip(DefaultPacketPreambleBits - 1)
		if a := consumeVueA(br, frameLen); !a.Vide {
			return nil, m522PaquetVueA
		}
	}
	w.PoserVueCourante(int(vueDeLImageCle))
	if _, _, hitEnd := decodeInferLoop(br, pay, w, cfg); !hitEnd {
		return nil, m522PaquetRang1
	}
	var out []m522Entree
	for tour := 0; tour < plafondToursVueC; tour++ {
		if !placeDisponible(br, frameLen, 1) || !br.ReadBit() {
			return out, m522PaquetLu
		}
		if !placeDisponible(br, frameLen, LargeurKindVueC) {
			return out, m522PaquetLu
		}
		k := int(br.ReadBits(LargeurKindVueC))
		if k == kindVueCNeant {
			continue
		}
		if k != kindVueCControle {
			return out, m522PaquetLu
		}
		e := c514LireControle(br, frameLen)
		out = append(out, m522Entree{rel: rel, ctl: e})
		if !e.porte {
			return out, m522PaquetLu
		}
	}
	return out, m522PaquetLu
}

// TestMouvement522Controle DATE les ouvertures de la garde des bits d action et les confronte aux
// decollages du temoin.
func TestMouvement522Controle(t *testing.T) {
	entrees, origine := m522LireControles(t, map[float64]m522CausePaquet{})
	if len(entrees) == 0 {
		t.Fatalf("aucune entree de controle")
	}
	t.Logf("ENTREES DE CONTROLE : %d · origine %d us", len(entrees), origine)
	parIdx := map[int]int{}
	var actions, blocs, pile, longue, troisieme, seconds int
	var ouvertures []m522Entree
	for _, e := range entrees {
		parIdx[e.ctl.indexCtrl]++
		if !e.ctl.blocPresent {
			continue
		}
		blocs++
		switch {
		case e.ctl.troisieme:
			troisieme++
		case e.ctl.champPile:
			pile++
		case e.ctl.brancheLong:
			longue++
		case e.ctl.actionsPres:
			actions++
			ouvertures = append(ouvertures, e)
		}
		if e.ctl.secondBloc {
			seconds++
		}
	}
	t.Logf("  blocs 0x68 : %d · gardes non portees : troisieme %d · pile %d · longue %d · "+
		"BITS D ACTION %d · second bloc %d", blocs, troisieme, pile, longue, actions, seconds)
	idx := make([]int, 0, len(parIdx))
	for i := range parIdx {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	var parts []string
	for _, i := range idx {
		parts = append(parts, fmt.Sprintf("%d:%d", i, parIdx[i]))
	}
	t.Logf("  index de controle R(5) : %v", parts)
	t.Logf("OUVERTURES DE LA GARDE D ACTION, DATEES :")
	for _, e := range ouvertures {
		t.Logf("  t=%8.3f s · index %2d · second %2d · scalaires %2d / %2d", e.rel,
			e.ctl.indexCtrl, e.ctl.second, e.ctl.scalaireA, e.ctl.scalaireB)
	}
}

// TestMouvement522ControleFenetre publie TOUTES les entrees de controle d une fenetre
// (`MOUV511_T0` / `MOUV511_T1`) : c est la differentielle du canal d entree, en regard des
// decollages du temoin.
func TestMouvement522ControleFenetre(t *testing.T) {
	couv := map[float64]m522CausePaquet{}
	entrees, _ := m522LireControles(t, couv)
	t0, t1 := c514Fenetre()
	fen := m522Couverture{}
	for rel, cause := range couv {
		if rel >= t0 && rel <= t1 {
			fen[cause]++
		}
	}
	t.Logf("COUVERTURE DANS LA FENETRE : %d lus · %d non localises · %d vue A · %d rang 1 ouvert",
		fen[m522PaquetLu], fen[m522PaquetNonLocal], fen[m522PaquetVueA], fen[m522PaquetRang1])
	var n int
	for _, e := range entrees {
		if e.rel < t0 || e.rel > t1 {
			continue
		}
		n++
		t.Logf("t=%8.3f s · idx %2d · bloc %v · second %2d · scal %2d/%2d · 3e %v · pile %v · "+
			"longue %v · ACTIONS %v · 2e bloc %v · porte %v", e.rel, e.ctl.indexCtrl,
			e.ctl.blocPresent, e.ctl.second, e.ctl.scalaireA, e.ctl.scalaireB, e.ctl.troisieme,
			e.ctl.champPile, e.ctl.brancheLong, e.ctl.actionsPres, e.ctl.secondBloc, e.ctl.porte)
	}
	t.Logf("TOTAL FENETRE [%.3f ; %.3f] : %d entrees", t0, t1, n)
}

// TestMouvement522Fermeture EST LE CONTROLE DE VALIDITE DE LA MESURE PRECEDENTE : « aucune entree
// de controle dans la fenetre du saut » ne vaut que si les paquets de cette fenetre sont LUS
// JUSQU AU BOUT. Le gate du lot 5.14 s applique tel quel : reste dans [0 ; 7] ET tous ses bits a
// ZERO (le bourrage d octet est ecrit a zero, donc un reste qui porte un 1 est de la grammaire
// manquante, meme dans sept bits).
func TestMouvement522Fermeture(t *testing.T) {
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
	var origine uint64
	var paquets, fermes, fermesNuls, restants int
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
			rel := float64(pk.TimestampUS-origine) / 1e6
			pay := pk.Payload(data)
			fin, ok := m522FinDePaquet(pay, w, cfg)
			if rel < t0 || rel > t1 {
				continue
			}
			paquets++
			if !ok {
				continue
			}
			reste := len(pay)*8 - fin
			if reste < 0 || reste > 7 {
				restants++
				continue
			}
			fermes++
			if m511Bits(pay, fin, reste) == strings.Repeat("0", reste) || reste == 0 {
				fermesNuls++
			}
		}
	}
	t.Logf("FENETRE [%.3f ; %.3f] : %d paquets delta · %d fermes (reste dans [0;7]) dont %d a "+
		"BITS TOUS NULS · %d laissent du reste", t0, t1, paquets, fermes, fermesNuls, restants)
}

// m522FinDePaquet rend le bit de fin de la lecture des trois rangs d un paquet delta, et dit si
// les trois rangs ont ete lus jusqu a leur terminateur.
func m522FinDePaquet(pay []byte, w *World, cfg FrameConfig) (int, bool) {
	frameLen := len(pay) * 8
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	if _, present := PacketHeadEventType(pay); present {
		d := marchLocateStrict(pay, w, cfg)
		if d < 0 {
			return 0, false
		}
		br.Skip(d)
	} else {
		br.Skip(DefaultPacketPreambleBits - 1)
		if a := consumeVueA(br, frameLen); !a.Vide {
			return 0, false
		}
	}
	w.PoserVueCourante(int(vueDeLImageCle))
	if _, _, hitEnd := decodeInferLoop(br, pay, w, cfg); !hitEnd {
		return 0, false
	}
	c := consumeVueC(br, frameLen)
	return br.BitPos(), c.Porte
}
