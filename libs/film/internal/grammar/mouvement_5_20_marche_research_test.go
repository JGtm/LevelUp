//go:build research

package grammar

// mouvement_5_20_marche_research_test.go — LA MARCHE DETERMINISTE D IMAGE-CLE CONTRE LE
// BALAYEUR D ANCRES (lot 5.20.1).
//
// Le lot 5.19 a mesure que le BALAYEUR (`WalkKeyframeWorld`, fenetre de 120 000 bits) s arrete
// entre 45,7 % et 55,8 % du payload d image-cle sur un film dense, et que la marche
// deterministe (`WalkKeyframeRecords`) rendait UN seul record, arret « en-tete-invalide », sur
// les deux temoins. La lecture de `FUN_142e2bfd0` (5.20.1) dit pourquoi : l en-tete par entite
// fait 108 bits et le corps est un ETAT COMPLET (aucune porte, aucun masque), quand la marche
// repartait a +64 bits par le cadre du record NEW du chemin DELTA.
//
// Cet instrument publie, chunk par chunk et pour les DEUX lectures : le nombre de records, le
// bit du dernier record, la part du payload atteinte, le slot maximal declare, et — pour la
// marche seule — la cause d arret et le composant qui bloque.
//
// Rejouable (un film a la fois) :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestMarche520$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"sort"
	"testing"
)

// m520FenetreHistorique est la fenetre de recherche d ancre que le depot portait jusqu au lot
// 5.20.1 (`kfScanNext`, 120 000 bits) — une invention du port : le jeu n en a aucune.
const m520FenetreHistorique = 120000

// m520Lecture est ce qu UNE lecture d un payload d image-cle rend.
type m520Lecture struct {
	Records   int
	DernierAt int
	FinAt     int
	SlotMax   int
	Arret     string
	Bloquant  string
}

// m520Balayeur joue le BALAYEUR d ancres sur un payload, avec la fenetre demandee
// (`maxWin <= 0` = aucune).
func m520Balayeur(pay []byte, maxWin int) m520Lecture {
	out := m520Lecture{Arret: "fenetre"}
	for _, r := range walkKeyframeWorldFenetre(pay, maxWin) {
		out.Records++
		if r.Bit > out.DernierAt {
			out.DernierAt = r.Bit
		}
		if r.Slot > out.SlotMax {
			out.SlotMax = r.Slot
		}
	}
	out.FinAt = out.DernierAt
	return out
}

// m520Marche joue la MARCHE DETERMINISTE sur un payload et nomme le composant qui bloque.
func m520Marche(pay []byte, reg *Registry, ctx ContexteDeLecture) m520Lecture {
	recs, stop := WalkKeyframeRecords(pay, reg, ctx)
	out := m520Lecture{Records: len(recs), Arret: stop.String()}
	for _, r := range recs {
		if r.BitStart > out.DernierAt {
			out.DernierAt = r.BitStart
		}
		if r.BitEnd > out.FinAt {
			out.FinAt = r.BitEnd
		}
		if r.Slot > out.SlotMax {
			out.SlotMax = r.Slot
		}
	}
	if n := len(recs); n > 0 && recs[n-1].DesyncAt >= 0 {
		out.Bloquant = nomComposantBloquant(reg, recs[n-1].TI, recs[n-1].DesyncAt)
	}
	return out
}

// TestMarche520 compare, chunk par chunk, le balayeur d ancres et la marche deterministe.
func TestMarche520(t *testing.T) {
	tc := t516Cadre(t)
	ctx := tc.fc.ContexteDeLecture()
	var totBal, totMar int
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			bits := len(pay) * 8
			bal := m520Balayeur(pay, m520FenetreHistorique)
			plein := m520Balayeur(pay, 0)
			mar := m520Marche(pay, tc.reg, ctx)
			totBal += bal.Records
			totMar += plein.Records
			t.Logf("chunk %2d · %d bits | fenetre 120k : %4d records, dernier %7d (%.1f %%),"+
				" slot max %4d | SANS fenetre : %4d records, dernier %7d (%.1f %%), slot max %4d"+
				" | marche deterministe %4d records, fin %7d (%.1f %%), arret %s %s",
				c, bits, bal.Records, bal.DernierAt, m533bPart(bal.DernierAt, bits), bal.SlotMax,
				plein.Records, plein.DernierAt, m533bPart(plein.DernierAt, bits), plein.SlotMax,
				mar.Records, mar.FinAt, m533bPart(mar.FinAt, bits), mar.Arret, mar.Bloquant)
		}
	}
	t.Logf("TOTAL : fenetre 120k %d records · sans fenetre %d records", totBal, totMar)
}

// TestBloquants520 recense, sur TOUS les payloads d image-cle du film, les composants sur
// lesquels la marche deterministe s arrete — le classement des ports qui la debloqueraient.
func TestBloquants520(t *testing.T) {
	tc := t516Cadre(t)
	ctx := tc.fc.ContexteDeLecture()
	par := map[string]int{}
	arrets := map[string]int{}
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			mar := m520Marche(pk.Payload(data), tc.reg, ctx)
			arrets[mar.Arret]++
			if mar.Bloquant != "" {
				par[mar.Bloquant]++
			}
		}
	}
	for a, n := range arrets {
		t.Logf("arret %-20s %3d payloads", a, n)
	}
	noms := make([]string, 0, len(par))
	for n := range par {
		noms = append(noms, n)
	}
	sort.Slice(noms, func(i, j int) bool {
		if par[noms[i]] != par[noms[j]] {
			return par[noms[i]] > par[noms[j]]
		}
		return noms[i] < noms[j]
	})
	for _, n := range noms {
		t.Logf("  bloquant %-60s %3d payloads", n, par[n])
	}
}
