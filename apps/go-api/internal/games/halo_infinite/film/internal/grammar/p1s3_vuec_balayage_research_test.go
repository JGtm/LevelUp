//go:build research

package grammar

// p1s3_vuec_balayage_research_test.go — SONDE P1-S3, MESURE DE COUVERTURE : la vue C d un paquet
// dont la vue B ne clot pas est-elle retrouvable PAR LA FIN ? (en-tete et grammaire :
// `p1s3_vuec_tir_continu_research_test.go`.) Mesure seule : ce balayage n est PAS une grammaire,
// il n est ecrit que pour chiffrer ce qu un repli nomme rendrait et ce qu il couterait.
//
// PRINCIPE, ecrit avant la mesure. La vue C est le DERNIER rang du paquet. Un depart candidat `p`
// est RETENU quand la vue C lue depuis `p` sous la grammaire complete (1) va jusqu a son
// terminateur, (2) laisse un reste de 0 a 7 bits nuls, (3) porte au moins une entree, (4) des
// index STRICTEMENT croissants (l ecrivain `FUN_14076b0e8` parcourt ses 32 cases dans l ordre),
// tous inferieurs a S3_JOUEURS. ETALONNAGE : sur les paquets dont la vue B clot ET que l oracle
// valide, le depart vrai est connu ; le balayage doit le retrouver, et seul.
//
//	(memes variables que TestP1S3VueCTirContinu) S3_JOUEURS=8 \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestP1S3VueCBalayage$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"os"
	"sort"
	"strconv"
	"testing"
)

// s3PaquetBrut garde ce qu il faut pour balayer un paquet apres coup.
type s3PaquetBrut struct {
	pay   []byte
	ts    uint64
	trame int
	vrai  int // depart vrai de la vue C quand l oracle valide la marche, sinon -1
}

// s3Plafond borne la fenetre de balayage (bits depuis la fin du paquet).
const s3Plafond = 2048

// s3Candidats rend les departs retenus pour un paquet.
func s3Candidats(pay []byte, joueurs int) []int {
	n := len(pay) * 8
	var out []int
	for p := n - 1; p >= 0 && p >= n-s3Plafond; p-- {
		var b s3Paquet
		es, fin, porte := s3LireVueC(pay, p, s3Entree{}, &b, true)
		if !porte || len(es) == 0 || !s3Ferme(pay, fin) {
			continue
		}
		ok, prec := true, -1
		for _, e := range es {
			if e.index <= prec || e.index >= joueurs {
				ok = false
				break
			}
			prec = e.index
		}
		if ok {
			out = append(out, p)
		}
	}
	return out
}

// s3PasseBrute decode le film une fois et garde les paquets delta, avec leur depart vrai.
func s3PasseBrute(tc t516Temoin, cad s3Cadre) []s3PaquetBrut {
	var out []s3PaquetBrut
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		t525Lier(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			q := s3PaquetBrut{pay: pay, ts: pk.TimestampUS, trame: cad.trame(pk.TimestampUS), vrai: -1}
			debut := DefaultPacketPreambleBits
			if _, present := PacketHeadEventType(pay); present {
				debut = marchLocate(pay, w, tc.cfg)
			}
			if debut >= 0 {
				if mar := t519Marcher(pay, w, tc.cfg, debut); mar.m.HitEndB {
					var b s3Paquet
					es, fin, porte := s3LireVueC(pay, mar.m.FinVueB, s3Entree{}, &b, true)
					if porte && s3Ferme(pay, fin) && len(es) > 0 {
						q.vrai = mar.m.FinVueB
					}
				}
			}
			out = append(out, q)
		}
	}
	return out
}

// s3Chaine rend les bits de presence des entrees d une vue C lue depuis `p`.
func s3Chaine(pay []byte, p int) map[int]bool {
	var b s3Paquet
	es, _, _ := s3LireVueC(pay, p, s3Entree{}, &b, true)
	out := map[int]bool{}
	for _, e := range es {
		out[e.debut] = true
	}
	return out
}

// s3Retenu applique la regle mesuree : le candidat le plus a GAUCHE, a condition que tous les
// autres candidats soient des debuts d entree de SA chaine (suffixes). Rend -1 sinon.
func s3Retenu(pay []byte, cs []int) int {
	if len(cs) == 0 {
		return -1
	}
	gauche := cs[len(cs)-1] // balayage de la fin vers le debut : le dernier est le plus a gauche
	ch := s3Chaine(pay, gauche)
	for _, c := range cs[:len(cs)-1] {
		if !ch[c] {
			return -1
		}
	}
	return gauche
}

// TestP1S3VueCBalayage etalonne le balayage par la fin, puis mesure ce qu il rend au pilote.
func TestP1S3VueCBalayage(t *testing.T) {
	cad := s3LireCadre(t)
	joueurs := 8
	if v, err := strconv.Atoi(os.Getenv("S3_JOUEURS")); err == nil {
		joueurs = v
	}
	tc := t516Cadre(t)
	paqs := s3PasseBrute(tc, cad)
	var etal [3]int // regle -> depart vrai · regle -> autre depart · regle muette
	var neufs [2]int
	var ents, marche []s3Entree
	for _, q := range paqs {
		r := s3Retenu(q.pay, s3Candidats(q.pay, joueurs))
		if q.vrai >= 0 {
			etal[map[bool]int{true: 0, false: map[bool]int{true: 2, false: 1}[r < 0]}[r == q.vrai]]++
			var b s3Paquet
			es, _, _ := s3LireVueC(q.pay, q.vrai, s3Entree{ts: q.ts, trame: q.trame, valide: true}, &b, true)
			marche = append(marche, es...)
			continue
		}
		if r < 0 {
			neufs[0]++
			continue
		}
		neufs[1]++
		var b s3Paquet
		es, _, _ := s3LireVueC(q.pay, r, s3Entree{ts: q.ts, trame: q.trame, valide: true}, &b, true)
		ents = append(ents, es...)
	}
	t.Logf("== ETALONNAGE de la regle (paquets a depart vrai connu, %d) : depart vrai %d · AUTRE depart %d "+
		"· muette %d", etal[0]+etal[1]+etal[2], etal[0], etal[1], etal[2])
	t.Logf("== PAQUETS SANS DEPART VRAI : regle muette %d · depart retenu %d", neufs[0], neufs[1])
	var tir, lues int
	for _, e := range ents {
		if e.index == cad.index {
			lues++
			tir += s3B(e.tir())
		}
	}
	t.Logf("== ENTREES RENDUES PAR LA REGLE : %d · index %d : %d dont %d qui tirent", len(ents), cad.index,
		lues, tir)
	s3PublierGate(t, ents, nil, cad, true)
	s3PublierAutourDesFrags(t, ents, cad)
	t.Logf("== MARCHE VALIDEE + REGLE (%d + %d entrees) :", len(marche), len(ents))
	tout := append(append([]s3Entree{}, marche...), ents...)
	sort.SliceStable(tout, func(i, j int) bool { return tout[i].ts < tout[j].ts })
	s3PublierGate(t, tout, nil, cad, true)
	s3PublierRafales(t, tout, cad, true)
	s3PublierCouverture(t, tout, cad)
}
