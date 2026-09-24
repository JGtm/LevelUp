//go:build research

package grammar

// m4b_cloture_research_test.go — LOT M4b : L EFFET DE LA LECTURE CORRIGEE SUR LA CLOTURE DES VUES B
// ET C, MESURE AVANT / APRES. Mesure seule.
//
// Il n emploie que ce que la base (`feat/rr-vague-d`) porte deja — `DecodeFrameViewsCurseur`, la
// liaison d image-cle de production, le localisateur strict, la table anticipee — pour pouvoir se
// jouer A L IDENTIQUE sur la base et sur la tete : seule la grammaire change entre les deux.
//
//	vue B close : la marche rend le rang de la vue B (terminateur OU rejet d en-tete)
//	vue C portee : la vue C se lit jusqu a son terminateur
//	vue C FERMEE : et le paquet se ferme (reste de 0 a 7 bits nuls) — l oracle de la sonde P1-S3
//
// Rejouable : memes variables que `m4b_tir_continu_research_test.go`.

import "testing"

// m4bFermeLocal est l oracle de cadrage, recopie ici pour que le fichier se joue sur la base.
func m4bFermeLocal(pay []byte, curseur int) bool {
	reste := len(pay)*8 - curseur
	if reste < 0 || reste > 7 {
		return false
	}
	for i := curseur; i < len(pay)*8; i++ {
		if pay[i/8]&(1<<uint(7-i%8)) != 0 { //nolint:gosec // i%8 in [0,7]
			return false
		}
	}
	return true
}

// TestM4bClotureDesVues publie la cloture des vues B et C sous la grammaire du binaire.
func TestM4bClotureDesVues(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	obs := NouvelleObservation()
	cfg := tc.cfg
	cfg.Obs = obs
	marche := tc.fc.MarcheDImageCle()
	var paquets, nonLoc, vueB, vueC, fermes, records, desyncs int
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		lierLeChunkAuMonde(w, marche, data, pks, obs)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			paquets++
			pay := pk.Payload(data)
			debut, rangsB := movementStateSkipLeadBits, 2
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					nonLoc++
					continue
				}
				rangsB = 1
			}
			recs, rangs, curseur := DecodeFrameViewsCurseur(pay, w, cfg, MovementStateViews, debut)
			for _, r := range recs {
				if r.TypeIndex == BipedTypeIndex {
					records++
					desyncs += s3B(r.DesyncAt >= 0)
				}
			}
			if rangs >= rangsB {
				vueB++
			}
			if rangs >= rangsB+1 {
				vueC++
				fermes += s3B(m4bFermeLocal(pay, curseur))
			}
		}
	}
	t.Logf("== CLOTURE : %d paquets · liste non localisee %d · vue B close %d · vue C portee %d · "+
		"vue C FERMEE %d (%.1f %%) · records ti=35 %d (desyncs %d)", paquets, nonLoc, vueB, vueC, fermes,
		100*float64(fermes)/float64(max(paquets, 1)), records, desyncs)
}
