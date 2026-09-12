package filmdec

// event_list_board_report_test.go — la SORTIE JOURNALISEE de l'instrument d'embarquement
// (balayage de largeurs, gates B1 a B4, diagnostic de decalage de base). Separe de
// event_list_board_test.go, qui porte la mesure, pour tenir le seuil de 500 lignes par fichier.

import (
	"sort"
	"strconv"
	"strings"
	"testing"
)

// evbRapport publie le balayage de largeurs et verdit les trois gates.
func evbRapport(t *testing.T, tot *evbTotaux) {
	t.Helper()
	t.Logf("\n########## V3 item B — L'OCCUPANT DE L'EMBARQUEMENT (%d films) ##########", tot.films)
	t.Logf("  trous >= 3 s : %d · embarquements : %d (payload relu %d) · sorties : %d",
		tot.trous, tot.boardTotal, tot.boardAvecOccupantPorte, tot.exits)
	widths := make([][3]int, 0, len(tot.parLargeur))
	for w := range tot.parLargeur {
		widths = append(widths, w)
	}
	sort.Slice(widths, func(i, j int) bool {
		for k := 0; k < 3; k++ {
			if widths[i][k] != widths[j][k] {
				return widths[i][k] < widths[j][k]
			}
		}
		return false
	})
	for _, w := range widths {
		c := tot.parLargeur[w]
		marque := ""
		if w[0] == dom2RefWidth && w[1] == dom3RefWidth && w[2] == dom7RefWidth {
			marque = "  <= largeurs de la build de reference (exe)"
		}
		t.Logf("  (dom2=%2d dom3=%2d dom7=%2d) bande %3d/%3d=%5.1f%% · ouvre trou %3d=%5.1f%% · TEM %3d=%4.1f%% · appariees %3d=%5.1f%% · trajet %3d=%5.1f%% · siege=sortie %3d=%5.1f%% · sieges %s%s",
			w[0], w[1], w[2], c.enBande, c.board, 100*evbPart(c.enBande, c.board),
			c.recoupe, 100*evbPart(c.recoupe, c.enBande),
			c.temoin, 100*evbPart(c.temoin, c.enBande),
			c.paires, 100*evbPart(c.paires, c.enBande),
			c.trajet, 100*evbPart(c.trajet, c.enBande),
			c.seatAccord, 100*evbPart(c.seatAccord, c.paires),
			evbTopSieges(c.sieges), marque)
	}
	ref := tot.parLargeur[[3]int{dom2RefWidth, dom3RefWidth, dom7RefWidth}]
	if ref == nil {
		ref = &evbCompte{}
	}
	b1 := evbPart(ref.recoupe, ref.enBande) >= evbPartMin
	b2 := evbPart(ref.enBande, ref.board) >= evbPartMin
	t.Logf("  GATE B1 (recoupement ouverture de trou >= %.0f %%) : %.1f %% (temoin %.1f %%) — %s",
		100*evbPartMin, 100*evbPart(ref.recoupe, ref.enBande),
		100*evbPart(ref.temoin, ref.enBande), evbVerdict(b1))
	t.Logf("  GATE B2 (occupant en bande >= %.0f %%) : %.1f %% — %s",
		100*evbPartMin, 100*evbPart(ref.enBande, ref.board), evbVerdict(b2))
	b4 := evbPart(ref.paires, ref.enBande) >= evbPartMin
	t.Logf("  GATE B4 (embarquement suivi d'une SORTIE du MEME occupant >= %.0f %%) : %d/%d = %.1f %% · dont trajet complet (le trou ouvert a l'embarquement est ferme par CETTE sortie) %d = %.1f %% — %s",
		100*evbPartMin, ref.paires, ref.enBande, 100*evbPart(ref.paires, ref.enBande),
		ref.trajet, 100*evbPart(ref.trajet, ref.enBande), evbVerdict(b4))
	t.Logf("  ACCORD DES SIEGES (embarquement vs sortie appariee) : %d/%d = %.1f %% — depend des largeurs dom3/dom7, cf. balayage",
		ref.seatAccord, ref.paires, 100*evbPart(ref.seatAccord, ref.paires))
	evbRapportBase(t, tot)
	b3 := evbPart(tot.exitRecoupe, tot.exitEnBande) >= evbPartMin
	t.Logf("  GATE B3 (controle SORTIE, non-regression) : en bande %d/%d = %.1f %% · ferme un trou %d = %.1f %% (temoin %d = %.1f %%) — %s",
		tot.exitEnBande, tot.exits, 100*evbPart(tot.exitEnBande, tot.exits),
		tot.exitRecoupe, 100*evbPart(tot.exitRecoupe, tot.exitEnBande),
		tot.exitTemoin, 100*evbPart(tot.exitTemoin, tot.exitEnBande), evbVerdict(b3))
}

// evbTopSieges rend les trois valeurs de siège les plus fréquentes.
func evbTopSieges(h map[uint32]int) string {
	keys := make([]uint32, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if h[keys[i]] != h[keys[j]] {
			return h[keys[i]] > h[keys[j]]
		}
		return keys[i] < keys[j]
	})
	var b strings.Builder
	for i, k := range keys {
		if i == 3 {
			break
		}
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(strconv.Itoa(int(k)) + "x" + strconv.Itoa(h[k]))
	}
	if b.Len() == 0 {
		return "-"
	}
	return b.String()
}

// evbRapportBase publie le diagnostic de décalage de base entre le domaine 2 et le domaine 1.
func evbRapportBase(t *testing.T, tot *evbTotaux) {
	t.Helper()
	t.Logf("  DIAGNOSTIC DECALAGE DE BASE : %d/%d trous ouverts a un embarquement sont refermes par une SORTIE (quel que soit son occupant)",
		tot.trousFermesParSortie, tot.trousFermesParSortie0)
	keys := make([]int64, 0, len(tot.ecartsBase))
	for k := range tot.ecartsBase {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if tot.ecartsBase[keys[i]] != tot.ecartsBase[keys[j]] {
			return tot.ecartsBase[keys[i]] > tot.ecartsBase[keys[j]]
		}
		return keys[i] < keys[j]
	})
	var b strings.Builder
	for i, k := range keys {
		if i == 6 {
			break
		}
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(strconv.FormatInt(k, 10) + "x" + strconv.Itoa(tot.ecartsBase[k]))
	}
	if b.Len() == 0 {
		b.WriteString("-")
	}
	t.Logf("     ecarts (slot de la sortie) - (slot lu a l'embarquement) : %s  [0 = meme numerotation]", b.String())
}

func evbVerdict(ok bool) string {
	if ok {
		return "PASSE"
	}
	return "ECHOUE"
}
