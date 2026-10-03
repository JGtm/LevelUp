//go:build research

package grammar

// mouvement_5_15_cascade_research_test.go — LA FORME DU TROU DANS LE TEMPS (lot 5.15.1).
//
// L instrument de position (`mouvement_5_15_position_research_test.go`) a nomme l endroit : la
// vue B clot sa liste sur un REJET de table de vue, et le slot rejete n est LIE PAR RIEN dans le
// monde hors ligne (21 988 sur 22 112 sur `bfecd02b`). Reste a savoir si ces slots sont perdus
// UNE FOIS pour toutes (une cascade a premiere cause unique) ou paquet par paquet.
//
// Il ne conclut rien : il donne le RANG du paquet dans son chunk, l instant de la premiere
// faute, et le sort des slots rejetes (lies plus tard par une image-cle ? par un NEW ?).
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestTrou515Cascade$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"sort"
	"testing"
)

// TestTrou515Cascade dit OU dans la suite des paquets d un chunk le trou commence, et si les
// slots rejetes sont connus AILLEURS dans le film.
func TestTrou515Cascade(t *testing.T) {
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

	// TOUS LES SLOTS QUE LES IMAGES-CLES DU FILM DECLARENT, tous chunks confondus : c est la
	// reference contre laquelle un slot rejete se lit « connu du film » ou « inconnu ».
	imageCle := map[uint32]uint32{}
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				imageCle[uint32(r.Slot)] = uint32(r.TI) //nolint:gosec // bornes du walker
			}
		}
	}

	w := NewWorld(reg)
	rangFerme := map[int]int{}
	rangFautif := map[int]int{}
	premierFautif := map[int]int{}
	rejetesConnus, rejetesInconnus := 0, 0
	slotsRejetes := map[uint32]bool{}
	var localise, amorce int
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		rang := 0
		premier := -1
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
			rang++
			m := t515Marcher(pay, w, cfg, debut)
			reste := len(pay)*8 - m.FinVueC
			ferme := reste >= 0 && reste <= m5116GateOctet && c514ResteNul(pay, m.FinVueC)
			if ferme {
				rangFerme[t515Classe(rang)]++
				continue
			}
			rangFautif[t515Classe(rang)]++
			if debut == DefaultPacketPreambleBits {
				amorce++
			} else {
				localise++
			}
			if premier < 0 {
				premier = rang
				premierFautif[t515Classe(rang)]++
			}
			if !m.PorteA || !m.HitEndB || !m.PorteC {
				continue
			}
			r, okR := t515LireRejet(pay, w, cfg, m)
			if !okR || r.Lie {
				continue
			}
			slotsRejetes[r.Slot] = true
			if _, connu := imageCle[r.Slot]; connu {
				rejetesConnus++
			} else {
				rejetesInconnus++
			}
		}
	}
	t.Logf("DEPART DE LA MARCHE sur les paquets fautifs : amorce %d · debut LOCALISE %d",
		amorce, localise)
	t.Logf("RANG DU PAQUET DANS SON CHUNK (classes) :")
	t.Logf("  paquets FERMES  : %s", c514Hist(rangFerme))
	t.Logf("  paquets FAUTIFS : %s", c514Hist(rangFautif))
	t.Logf("  PREMIER fautif du chunk : %s", c514Hist(premierFautif))
	t.Logf("SLOT REJETE ET NON LIE : %d declare par une image-cle du film · %d declare par AUCUNE",
		rejetesConnus, rejetesInconnus)
	t.Logf("  slots distincts rejetes non lies : %d · dont dans une image-cle : %d",
		len(slotsRejetes), t515Intersection(slotsRejetes, imageCle))
	t.Logf("  slots declares par les images-cles du film : %d", len(imageCle))
}

// t515Classe regroupe un rang en classes lisibles (1, 2, 3, 5, 10, 20, 50, 100, 200, 500, 1000+).
func t515Classe(rang int) int {
	for _, b := range []int{1, 2, 3, 5, 10, 20, 50, 100, 200, 500, 1000} {
		if rang <= b {
			return b
		}
	}
	return 2000
}

// t515Intersection compte les slots de `a` presents dans `b`.
func t515Intersection(a map[uint32]bool, b map[uint32]uint32) int {
	n := 0
	for k := range a {
		if _, ok := b[k]; ok {
			n++
		}
	}
	return n
}

// TestTrou515Population confronte les populations de slots : ceux que les images-cles declarent,
// ceux que la marche LIE par un record NEW, ceux qu elle lit en DELTA, et ceux qu elle REJETTE.
// Elle ne conclut rien : elle dit d ou vient (ou ne vient pas) la liaison qui manque.
func TestTrou515Population(t *testing.T) {
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

	kf := map[uint32]bool{}
	nouveaux := map[uint32]bool{}
	deltas := map[uint32]bool{}
	rejets := map[uint32]bool{}
	var nNew, nDelta int
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				kf[uint32(r.Slot)] = true //nolint:gosec // bornes du walker
			}
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
			recs, _, _ := t515Records(pay, w, cfg, debut)
			for _, r := range recs {
				switch r.Type {
				case recNew:
					nNew++
					nouveaux[r.Slot] = true
				case recDelta:
					nDelta++
					deltas[r.Slot] = true
				}
			}
			if !m.PorteA || !m.HitEndB || !m.PorteC {
				continue
			}
			if r, okR := t515LireRejet(pay, w, cfg, m); okR && !r.Lie {
				rejets[r.Slot] = true
			}
		}
	}
	t.Logf("RECORDS LUS : %d NEW (%d slots distincts) · %d DELTA (%d slots distincts)",
		nNew, len(nouveaux), nDelta, len(deltas))
	t.Logf("SLOTS : images-cles %d · NEW lus %d · deltas lus %d · REJETS non lies %d",
		len(kf), len(nouveaux), len(deltas), len(rejets))
	t.Logf("ETENDUE DES SLOTS (min / max / mediane) :")
	t.Logf("  images-cles     : %s", t515Etendue(kf))
	t.Logf("  NEW lus         : %s", t515Etendue(nouveaux))
	t.Logf("  deltas lus      : %s", t515Etendue(deltas))
	t.Logf("  rejets non lies : %s", t515Etendue(rejets))
}

// t515Records rejoue la vue B d un paquet et rend SES records (la marche de [t515Marcher] ne les
// garde pas).
func t515Records(pay []byte, w *World, cfg FrameConfig, debut int) ([]FrameRecord, int, bool) {
	frameLen := len(pay) * 8
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	if debut == DefaultPacketPreambleBits && cfg.PacketPreambleBits >= 1 {
		br.Skip(cfg.PacketPreambleBits - 1)
		if a := consumeVueA(br, frameLen); !a.Porte {
			return nil, br.BitPos(), false
		}
	} else {
		br.Skip(debut)
	}
	w.PoserVueCourante(int(vueDeLImageCle))
	return decodeInferLoop(br, pay, w, cfg)
}

// t515Etendue rend « min N · max N · mediane N · N valeurs » d un ensemble de slots.
func t515Etendue(m map[uint32]bool) string {
	if len(m) == 0 {
		return "vide"
	}
	l := make([]int, 0, len(m))
	for k := range m {
		l = append(l, int(k))
	}
	sort.Ints(l)
	return fmt.Sprintf("min %d · max %d · mediane %d · %d valeurs · sous 1024 : %d",
		l[0], l[len(l)-1], l[len(l)/2], len(l), t515Sous(l, 1024))
}

// t515Sous compte les valeurs strictement inferieures a `n`.
func t515Sous(l []int, n int) int {
	return sort.SearchInts(l, n)
}
