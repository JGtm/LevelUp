package grammar

// f0_103_pieces_research_test.go — LOT F.0, QUESTION 4 : LA PIECE ENGENDREE.
//
// # LA QUESTION (D13)
//
// Le mur est aujourd hui le SEUL equipement dont le manifeste declare une piece engendree
// (`kind = "deployed"`, les panneaux) — et c est pour cela qu il est le seul dont le canal des
// poses voie le deploiement (rapport E0 du 2026-09-10, question 5 : 84 % pour le mur, ZERO sur
// 202 consommations pour toutes les autres familles). Reste l hypothese jamais eprouvee : le
// champ de reparation, le capteur et le traqueur engendrent peut-etre EUX AUSSI une piece, dont
// le GlobalID ne serait simplement pas au manifeste.
//
// # LA MESURE, ET SON TEMOIN POSITIF OBLIGATOIRE
//
// Pour chaque CONSOMMATION DE CHARGE (`equipmentChanges` `spent`, chaine intacte `gap = 0`,
// rang precedent nomme), on recense les creations `ti=37` TOUTES CATEGORIES — pas seulement
// les GlobalID du manifeste — dans une fenetre de +/- 2 s, et on les compare a la distribution
// du film entier. La grandeur publiee est donc un ENRICHISSEMENT, pas un compte brut : un
// identifiant tres frequent (les grenades) tombe pres de n importe quoi.
//
// LE TEMOIN POSITIF EST LE MUR : sa piece est connue (`0x528fce46`, `0x686b40c9`). Si la
// methode la retrouve sur le mur et ne trouve rien sur le champ de reparation, le negatif est
// ANCRE. Si elle ne retrouve meme pas celle du mur, elle ne prouve rien et le dit.
//
// TEMOIN NEGATIF : le meme nombre de fenetres, posees a des instants TIRES AU HASARD dans le
// film. C est lui qui donne l echelle du bruit.
//
// LECTURE SEULE. Gardes et commande : `f0_103_contexte_research_test.go`.

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

// f0FenetrePieceUS est le rayon de la fenetre autour d une consommation de charge : 2 s, le
// meme ordre que la fenetre de la mesure E0 (question 5), qui cherchait une POSE autour d un
// `spent`. La garder identique rend les deux mesures comparables.
const f0FenetrePieceUS = 2_000_000

// f0Hist est un histogramme de GlobalID.
type f0Hist map[uint32]int

func (h f0Hist) total() int {
	n := 0
	for _, v := range h {
		n += v
	}
	return n
}

// TestF0PieceEngendree : question 4 du lot F.0.
func TestF0PieceEngendree(t *testing.T) {
	root, ids := f0Films(t)
	cat := f0Catalogue(t)
	cartes := f0Cartes(t, cat)
	objets := f0ManifesteObjets(t)

	parcFond := f0Hist{}
	parcPres := map[string]f0Hist{}
	parcTemoin := map[string]f0Hist{}
	parcFenetres := map[string]int{}

	for _, id := range ids {
		e, ok := cartes[id]
		if !ok {
			continue
		}
		f := f0Charge(t, root, id, e)
		a, vu := f0LitArtefact(t, id)
		if !vu {
			continue
		}
		origin, step, ok := f0OrigineUS(f, a)
		if !ok {
			t.Logf("film %s : artefact sans `originMs` — hors de la question 4", id)
			continue
		}
		rangs, source := f0RangFamille(t, a)
		fond := f0Fond(f)
		for k, v := range fond {
			parcFond[k] += v
		}
		t.Logf("")
		t.Logf("######## FILM %s ########", id)
		f0LogCascade(t, f, objets, fond)
		t.Logf("  palette rang -> famille : %d rangs, source %s", len(rangs), source)

		fenetres := f0FenetresSpent(a, rangs, origin, step)
		rng := rand.New(rand.NewSource(20260913))
		derniere := f0DernierInstant(f)
		for _, fam := range f0FamillesTriees(fenetres) {
			ats := fenetres[fam]
			pres := f0RecenseAutour(f, ats)
			var faux []uint64
			for range ats {
				faux = append(faux, f.BaseUS+uint64(rng.Int63n(int64(max(1, int(derniere))))))
			}
			temoin := f0RecenseAutour(f, faux)
			t.Logf("  %-20s %3d consommations · %3d creations `ti=37` a +/- 2 s (temoin hasard %d) · %s",
				fam, len(ats), pres.total(), temoin.total(), f0HistTexte(pres, fond, objets))
			f0FusionneHist(parcPres, fam, pres)
			f0FusionneHist(parcTemoin, fam, temoin)
			parcFenetres[fam] += len(ats)
		}
	}

	t.Logf("")
	t.Logf("######## PARC — question 4 ########")
	t.Logf("  fond du parc (toutes creations `ti=37` acceptees) : %d records, %d identifiants",
		parcFond.total(), len(parcFond))
	for _, fam := range f0FamillesTrieesHist(parcPres) {
		t.Logf("  %-20s %3d consommations · pres %3d · temoin %3d · %s",
			fam, parcFenetres[fam], parcPres[fam].total(), parcTemoin[fam].total(),
			f0HistTexte(parcPres[fam], parcFond, objets))
	}
}

// f0Fond rend la distribution des GlobalID sur TOUTES les creations `ti=37` du film.
func f0Fond(f f0Film) f0Hist {
	out := f0Hist{}
	for _, c := range f.Creations {
		out[uint32(c.MPPVal[MPPWord32])]++
	}
	return out
}

// f0DernierInstant rend la duree couverte par les creations, en µs depuis le zero du film.
func f0DernierInstant(f f0Film) uint64 {
	var last uint64
	for _, c := range f.Creations {
		if c.TimestampUS > last {
			last = c.TimestampUS
		}
	}
	if last <= f.BaseUS {
		return 1
	}
	return last - f.BaseUS
}

// f0FenetresSpent rend, par FAMILLE, les instants MOTEUR des consommations de charge
// exploitables. Les regles sont celles de la production (`usage_summary_outcomes.go`) : le
// rang consomme est sur `from`, et une chaine trouee (`gap > 0`) rend `from` non fiable.
func f0FenetresSpent(a f0Art, rangs map[int]string, origin, step uint64) map[string][]uint64 {
	out := map[string][]uint64{}
	for _, c := range a.EquipmentChanges {
		if c.Kind != "spent" || c.Gap > 0 {
			continue
		}
		fam := rangs[c.From]
		if fam == "" {
			continue
		}
		out[fam] = append(out[fam], origin+uint64(c.T)*step)
	}
	return out
}

// f0RecenseAutour recense les creations `ti=37` a moins de `f0FenetrePieceUS` d un des
// instants donnes. Une creation comptee une seule fois par fenetre qui la voit.
func f0RecenseAutour(f f0Film, ats []uint64) f0Hist {
	out := f0Hist{}
	for _, at := range ats {
		for _, c := range f.Creations {
			d := int64(c.TimestampUS) - int64(at)
			if d < -int64(f0FenetrePieceUS) || d > int64(f0FenetrePieceUS) {
				continue
			}
			out[uint32(c.MPPVal[MPPWord32])]++
		}
	}
	return out
}

// f0HistTexte ecrit les cinq identifiants les plus enrichis : leur compte, leur part dans la
// fenetre, leur part dans le film, et leur famille de manifeste quand elle existe.
func f0HistTexte(h, fond f0Hist, objets map[uint32]string) string {
	type kv struct {
		id    uint32
		n     int
		enrch float64
	}
	tot, totF := h.total(), fond.total()
	if tot == 0 || totF == 0 {
		return "aucune creation dans les fenetres"
	}
	var l []kv
	for id, n := range h {
		part := float64(n) / float64(tot)
		partF := float64(fond[id]) / float64(totF)
		e := 0.0
		if partF > 0 {
			e = part / partF
		}
		l = append(l, kv{id, n, e})
	}
	// TRI PAR COMPTE, PAS PAR ENRICHISSEMENT. Un identifiant vu UNE seule fois dans tout le
	// film rend mecaniquement l enrichissement le plus haut possible (1/partF) des qu il
	// tombe dans une fenetre : classer dessus ferait remonter le bruit du balayage brut, qui
	// porte des milliers d identifiants a un seul record. Le compte porte le classement,
	// l enrichissement reste la colonne qui dit si ce compte vaut mieux que le hasard.
	sort.Slice(l, func(i, j int) bool {
		if l[i].n != l[j].n {
			return l[i].n > l[j].n
		}
		return l[i].enrch > l[j].enrch
	})
	var parts []string
	for i, e := range l {
		if i >= 6 {
			break
		}
		fam := objets[e.id]
		if fam == "" {
			fam = "HORS MANIFESTE"
		}
		parts = append(parts, fmt.Sprintf("0x%08x(%s) n=%d x%.2f", e.id, fam, e.n, e.enrch))
	}
	return "enrichissement : " + fmt.Sprint(parts)
}

// f0LogCascade ecrit la cascade du balayage `ti=37` — ce que le film voit contre ce que
// l artefact publie — et DIT ce que sont les creations non publiees.
func f0LogCascade(t *testing.T, f f0Film, objets map[uint32]string, fond f0Hist) {
	t.Helper()
	connus, inconnus := 0, 0
	histInconnus := f0Hist{}
	for id, n := range fond {
		if objets[id] != "" {
			connus += n
			continue
		}
		inconnus += n
		histInconnus[id] += n
	}
	t.Logf("  cascade `ti=37` : %d ancres -> %d records acceptes -> %d confirmes par l oracle "+
		"de vie -> %d poses publiees", f.CreStats.Anchors, f.CreStats.Accepted,
		f.PlaceStats.Confirmed, f.PlaceStats.Placements)
	t.Logf("  identite des %d records acceptes : %d au manifeste, %d hors manifeste "+
		"(%d identifiants distincts)", fond.total(), connus, inconnus, len(histInconnus))
	type kv struct {
		id uint32
		n  int
	}
	var l []kv
	for id, n := range histInconnus {
		l = append(l, kv{id, n})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
	var parts []string
	for i, e := range l {
		if i >= 8 {
			break
		}
		parts = append(parts, fmt.Sprintf("0x%08x:%d", e.id, e.n))
	}
	t.Logf("  hors manifeste, les plus frequents : %v", parts)
}

func f0FusionneHist(dst map[string]f0Hist, fam string, h f0Hist) {
	if dst[fam] == nil {
		dst[fam] = f0Hist{}
	}
	for k, v := range h {
		dst[fam][k] += v
	}
}

func f0FamillesTriees(m map[string][]uint64) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func f0FamillesTrieesHist(m map[string]f0Hist) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
