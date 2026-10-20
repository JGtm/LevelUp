//go:build research

package grammar

// mouvement_5_3_2d_preambule_research_test.go — LE PREAMBULE DES PAQUETS, SANS HYPOTHESE.
//
// # LA QUESTION, POSEE SANS PRESUPPOSE
//
// 12 316 paquets a liste d evenements vide sont rejetes des leur PREMIER record, contre 10 512
// qui se decodent entierement. Avant de chercher un drapeau precis dans l ecrivain, une
// question plus simple : **ces deux populations sont-elles seulement distinguables a leur
// en-tete ?**
//
// Si une valeur d octet de tete ou une classe de taille est propre aux rejetes, c est le
// cadrage manquant, et il se lira chez l ecrivain. Si les deux distributions se superposent,
// la piste du preambule se ferme — et c est une reponse, pas un echec.
//
// # POURQUOI L OCTET DE TETE ET LA TAILLE, ET RIEN D AUTRE
//
// Ce sont les deux seules grandeurs disponibles AVANT toute lecture de grammaire : l octet 0
// porte le bit de configuration, le bit de continuation et les bits de tete du premier champ
// (`event_list.go` : « l octet 0 du payload vaut 0xC0 | (type >> 1) » quand la liste porte un
// evenement) ; la taille est celle du paquet. Tout le reste supposerait deja un cadrage.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// m532dPreambule confronte les deux populations sur leur en-tete.
func m532dPreambule(t *testing.T, st m532dStat) {
	t.Helper()
	nS, nR := 0, 0
	for _, v := range st.teteSains {
		nS += v
	}
	for _, v := range st.teteRejetes {
		nR += v
	}
	if nS == 0 || nR == 0 {
		t.Logf("PREAMBULE : une des deux populations est vide (sains %d, rejetes %d)", nS, nR)
		return
	}
	cles := map[byte]bool{}
	for k := range st.teteSains {
		cles[k] = true
	}
	for k := range st.teteRejetes {
		cles[k] = true
	}
	ord := make([]int, 0, len(cles))
	for k := range cles {
		ord = append(ord, int(k))
	}
	sort.Ints(ord)
	t.Logf("PREAMBULE — OCTET DE TETE, part dans chaque population (%d sains, %d rejetes) :", nS, nR)
	exclusifs := 0
	for _, k := range ord {
		b := byte(k) //nolint:gosec // clef d octet
		s, r := st.teteSains[b], st.teteRejetes[b]
		marque := ""
		switch {
		case s == 0 && r >= 100:
			marque = "   <-- PROPRE AUX REJETES"
			exclusifs++
		case r == 0 && s >= 100:
			marque = "   (propre aux sains)"
		}
		t.Logf("    0x%02X : sains %6d (%5.2f %%) · rejetes %6d (%5.2f %%)%s",
			b, s, m532Pct(s, nS), r, m532Pct(r, nR), marque)
	}
	t.Logf("PREAMBULE — TAILLE (classes de 64 octets) : sains %s · rejetes %s",
		m532dTailles(st.tailleSains), m532dTailles(st.tailleRejetes))
	if exclusifs == 0 {
		t.Logf("PREAMBULE : VERDICT NEGATIF — aucune valeur d octet de tete n est propre aux " +
			"rejetes. Les deux populations ne se distinguent pas a leur en-tete, et la piste du " +
			"preambule se ferme.")
		return
	}
	t.Logf("PREAMBULE : %d valeur(s) d octet de tete PROPRE(S) aux rejetes — a lire chez "+
		"l ecrivain avant toute correction.", exclusifs)
}

func m532dTailles(m map[int]int) string {
	ord := make([]int, 0, len(m))
	for k := range m {
		ord = append(ord, k)
	}
	sort.Ints(ord)
	var parts []string
	for i, k := range ord {
		if i >= 5 {
			parts = append(parts, fmt.Sprintf("+%d classes", len(ord)-5))
			break
		}
		parts = append(parts, fmt.Sprintf("%d-%do:%d", k*64, (k+1)*64-1, m[k]))
	}
	return strings.Join(parts, " ")
}
