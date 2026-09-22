//go:build research

package grammar

// mouvement_5_15_recalage_research_test.go — LA POSITION EXACTE DU DECALAGE (lot 5.15.1).
//
// Les instruments precedents disent QUE la vue B quitte la piste et OU elle s arrete. Celui-ci
// dit DE COMBIEN DE BITS : pour chaque paquet fautif, il cherche la position de reprise la plus
// proche de la fin du dernier record lu a partir de laquelle la vue B va jusqu a son
// terminateur, la vue C jusqu au sien, et le paquet ferme A RESTE NUL.
//
// CE N EST PAS UN BALAYAGE DE GRAMMAIRE : aucune largeur n est choisie par cette mesure. Elle
// localise le decalage — l ecart entre la fin du dernier record LU et la fin du dernier record
// ECRIT — pour que la lecture de l ecrivain sache quel champ chercher et de quelle taille.
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestTrou515Recalage$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"testing"
)

// t515FenetreRecalage est l etendue, en bits, ou la reprise est cherchee autour de la fin du
// dernier record lu (assez large pour couvrir la queue d un composant, pas assez pour tomber sur
// un record entier par hasard).
const t515FenetreRecalage = 192

// TestTrou515Recalage publie l histogramme du decalage sur les paquets fautifs.
func TestTrou515Recalage(t *testing.T) {
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

	decalage := map[int]int{}
	parDernier := map[string]map[int]int{}
	var fautifs, recales int
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
			m := t515Marcher(pay, w, cfg, debut)
			reste := len(pay)*8 - m.FinVueC
			if reste >= 0 && reste <= m5116GateOctet && c514ResteNul(pay, m.FinVueC) {
				continue
			}
			if !m.PorteA || !m.HitEndB || !m.PorteC {
				continue
			}
			fautifs++
			enTete := 1 + cfg.IDLowBits + 2
			if cfg.HasExtraFields {
				enTete += 32
			}
			base := m.FinVueB - enTete // la fin du corps du dernier record LU
			d, trouve := t515ChercherRecalage(pay, w, cfg, base)
			if !trouve {
				continue
			}
			recales++
			decalage[d]++
			recs, _, _ := t515Records(pay, w, cfg, debut)
			cle := t515CleDernier(recs)
			if parDernier[cle] == nil {
				parDernier[cle] = map[int]int{}
			}
			parDernier[cle][d]++
		}
	}
	t.Logf("PAQUETS FAUTIFS A TROIS RANGS PORTES : %d · RECALES dans +-%d bits : %d (%.1f %%)",
		fautifs, t515FenetreRecalage, recales, m533bPart(recales, fautifs))
	t.Logf("DECALAGE (bits a ajouter a la fin du dernier record lu) : %s", c514Hist(decalage))
	t.Logf("DECALAGE PAR DERNIER RECORD LU (les dix classes les plus peuplees) :")
	t515TopDecalage(t, parDernier)
}

// t515ChercherRecalage cherche, par ordre d ecart croissant autour de `base`, la premiere
// position ou la vue B puis la vue C se lisent jusqu a leur terminateur et ou le paquet ferme a
// reste NUL. Rend l ecart signe et `true` quand une telle position existe.
func t515ChercherRecalage(pay []byte, w *World, cfg FrameConfig, base int) (int, bool) {
	frameLen := len(pay) * 8
	for ecart := 0; ecart <= t515FenetreRecalage; ecart++ {
		for _, d := range t515Signes(ecart) {
			p := base + d
			if p < 0 || p >= frameLen {
				continue
			}
			br := LecteurSur(pay)
			br.poserCadre(cfg)
			br.Skip(p)
			w.PoserVueCourante(int(vueDeLImageCle))
			if _, _, hitEnd := decodeInferLoop(br, pay, w, cfg); !hitEnd {
				continue
			}
			if cc := consumeVueC(br, frameLen); !cc.Porte {
				continue
			}
			r := frameLen - br.BitPos()
			if r >= 0 && r <= m5116GateOctet && c514ResteNul(pay, br.BitPos()) {
				return d, true
			}
		}
	}
	return 0, false
}

// t515Signes rend les ecarts a tester pour une distance donnee (0, puis +n et -n).
func t515Signes(n int) []int {
	if n == 0 {
		return []int{0}
	}
	return []int{n, -n}
}

// t515TopDecalage publie, par classe de dernier record, la distribution des ecarts.
func t515TopDecalage(t *testing.T, m map[string]map[int]int) {
	t.Helper()
	type kv struct {
		k string
		v int
	}
	var l []kv
	for k, h := range m {
		n := 0
		for _, v := range h {
			n += v
		}
		l = append(l, kv{k, n})
	}
	for i := 0; i < len(l); i++ {
		for j := i + 1; j < len(l); j++ {
			if l[j].v > l[i].v {
				l[i], l[j] = l[j], l[i]
			}
		}
	}
	for i := 0; i < 10 && i < len(l); i++ {
		t.Logf("  %6d  %s", l[i].v, l[i].k)
		t.Logf("          ecarts : %s", c514Hist(m[l[i].k]))
	}
	_ = fmt.Sprintf
}
