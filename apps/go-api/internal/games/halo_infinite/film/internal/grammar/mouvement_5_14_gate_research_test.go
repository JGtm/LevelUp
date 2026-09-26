//go:build research

package grammar

// mouvement_5_14_gate_research_test.go — LE GATE DU LOT 5.14 ET SON REGISTRE DE RESTES.
//
// SORTI DE `mouvement_5_14_classes_research_test.go` PAR DEPLACEMENT PUR : ce fichier avait
// atteint 699 lignes pour un seuil de 500 (`archlint/film_file_size_test.go`, CLAUDE.md regle 5).
// Aucune ligne de logique ne change. La frontiere est celle du GATE — mesurer si un paquet est
// LU — face a la LECTURE des trois classes, qui reste dans le fichier d origine.
//
// # LE GATE, ET POURQUOI IL EST PLUS STRICT QUE CELUI DU LOT 5.11.6
//
// « Reste dans [0 ; 7] » ne prouve pas qu un paquet est lu : le bourrage d octet est ecrit A
// ZERO, donc un reste qui porte un 1 est de la grammaire MANQUANTE meme quand il tient dans sept
// bits. [TestClasses514Bourrage] exige que TOUS les bits du reste soient nuls.
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestClasses514(Bourrage|Restes|IdLow)$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"sort"
	"testing"
)

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
