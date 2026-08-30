package replay

// visee_larg114_research_test.go — LOT A4 : LES LARGEURS DU VAR-INT DE REFERENCE, PAR FERMETURE
// ARITHMETIQUE SUR LES FRONTIERES MESUREES.
//
// CE QU'ON CHERCHE. La grammaire du dispatcher FUN_14080a9d4 impose, pour un evenement :
//
//	R(7) type · { R(1) porte ; si 1 : [sonde R(1)] + R(W) + R(2) } x3 · payload du type · R(1)
//
// avec W = bitLen(plage du domaine) — pour le type 114, les domaines valent 2, 3 et 7 (case
// +0x58 du descripteur). Aucune de ces largeurs n'est connue hors ligne. Mais elles ne sont
// pas libres : elles doivent placer la FIN du var-int exactement la ou A1 a mesure une
// frontiere. C'est la fermeture arithmetique de la methode du depot (§5) — deux mesures
// independantes qui doivent coincider sans ajustement.
//
// L'INCONNUE REELLE EST UNE SEULE POSITION. Sonde presente + W et sonde absente + (W+1)
// consomment le meme nombre de bits et laissent la suite au meme endroit : elles sont
// INDISCERNABLES sur le flux. On ne balaye donc pas (sonde, W) mais la position de fin du
// var-int, pFin = 8 + sonde + W, et l'on publie les deux lectures equivalentes.
//
// AVERTISSEMENT — LE CRITERE C1 DE LA PREMIERE PASSE EST REFUTE, ET IL EST GARDE ICI EXPRES.
// La passe L1 ci-dessous suppose que les deux bits qui closent le var-int sont CONSTANTS (des
// bits de fin de champ). La note de rétro-ingénierie du lot B (`.ai/V7.5/film_re/
// NOTE_ENVELOPPE_EVENTS_2026-08-30.md`, lecture statique du binaire) etablit qu'ils sont les
// deux bits de GENERATION du handle 32 bits rendu par FUN_1406d3140 : de la donnee, qui n'a
// aucune raison d'etre constante. C1 elimine donc des hypotheses valides, et la « solution
// unique » de L1 n'en est pas une. L1 reste publie comme mesure de frontieres — ses colonnes
// sont justes — mais c'est la mesure L4, qui decode les portes DYNAMIQUEMENT, qui fait foi.
//
// CRITERES ECRITS AVANT LA MESURE, appliques a chaque pFin de 12 a 26, sur DEUX films (la
// regle d'ecriture du depot interdit de conclure sur un seul match) :
//
//	C1. les deux bits de queue du var-int, [pFin ; pFin+2), sont CONSTANTS sur tous les
//	    paquets des deux films (ce sont des bits de fin de champ, pas de la donnee) ;
//	C2. le bit de porte de la reference 1, en pFin+2, est constant ;
//	C3. si cette porte vaut 0, le bit de porte de la reference 2, en pFin+3, est constant ;
//	C4. le payload du type — R(6), lu en pFin+4 quand les deux portes sont fermees — a une
//	    cardinalite <= 8 sur chaque film (c'est un index de siege, pas un compteur) ;
//	C5. FERMETURE : le type d'evenement relu a la fin calculee de l'enveloppe tombe dans les
//	    types deja nommes par le depot. Ce critere n'est informatif que si le format enchaine
//	    les evenements dans un meme paquet ; la mesure L3 le dit, elle n'est pas supposee.
//
// Une hypothese qui passe C1..C4 sur les deux films est RETENUE ; s'il en reste plusieurs,
// l'ambiguite est publiee telle quelle — c'est un resultat, pas un echec.
//
// SOUS GARDE (LARG114_FILMS : repertoires separes par des virgules).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 LARG114_FILMS=<repo>/data/cache/film_chunks/00162144,<repo>/data/cache/film_chunks/00ba2e1c \
//	  go test ./internal/analysis/replay/ -run TestViseeLargeurs114 -v -timeout 30m

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	larg114FilmsEnv = "LARG114_FILMS"
	larg114PFinMin  = 12
	larg114PFinMax  = 26
	larg114Payload  = 6 // largeur du payload du type 114 (vtable +0x68 : un seul R(6))
	larg114CardMax  = 8 // cardinalite maximale acceptee pour le payload (critere C4)
)

// larg114Verdict porte le resultat des criteres pour une position de fin de var-int.
type larg114Verdict struct {
	pFin       int
	queueConst bool
	queue      string
	porte1     string
	porte2     string
	cardPay    int
	palette    string
}

// larg114ConstanteAt rend "0", "1" ou "" (variable) pour un bit sur un lot de paquets.
func larg114ConstanteAt(pk []env114Paquet, b int) string {
	var uns, total int
	for _, p := range pk {
		if b >= p.nBits {
			continue
		}
		uns += int(filmdec.ReadBitsAtForDiag(p.pay, b, 1))
		total++
	}
	switch {
	case total == 0:
		return ""
	case uns == 0:
		return "0"
	case uns == total:
		return "1"
	default:
		return ""
	}
}

// larg114Palette rend la cardinalite et le detail des valeurs d'une tranche.
func larg114Palette(pk []env114Paquet, d, w int) (int, string) {
	comptes := map[uint32]int{}
	for _, p := range pk {
		if d+w <= p.nBits {
			comptes[filmdec.ReadBitsAtForDiag(p.pay, d, w)]++
		}
	}
	var vs []int
	for v := range comptes {
		vs = append(vs, int(v))
	}
	sort.Ints(vs)
	var parts []string
	for i, v := range vs {
		if i == 10 {
			parts = append(parts, "...")
			break
		}
		parts = append(parts, fmt.Sprintf("%d:%d", v, comptes[uint32(v)]))
	}
	return len(vs), strings.Join(parts, " ")
}

// larg114Evalue applique C1..C4 a une position de fin de var-int, sur un film.
func larg114Evalue(pk []env114Paquet, pFin int) larg114Verdict {
	v := larg114Verdict{pFin: pFin}
	q0, q1 := larg114ConstanteAt(pk, pFin), larg114ConstanteAt(pk, pFin+1)
	v.queueConst = q0 != "" && q1 != ""
	v.queue = q0 + q1
	if !v.queueConst {
		v.queue = "variable"
	}
	v.porte1 = larg114ConstanteAt(pk, pFin+2)
	if v.porte1 == "" {
		v.porte1 = "variable"
	}
	v.porte2 = larg114ConstanteAt(pk, pFin+3)
	if v.porte2 == "" {
		v.porte2 = "variable"
	}
	v.cardPay, v.palette = larg114Palette(pk, pFin+4, larg114Payload)
	return v
}

// larg114Retient dit si un verdict passe C1..C4.
func larg114Retient(v larg114Verdict) bool {
	return v.queueConst && v.porte1 == "0" && v.porte2 == "0" && v.cardPay <= larg114CardMax
}

// larg114Tableau imprime le tableau des hypotheses pour un film et rend les positions retenues.
func larg114Tableau(t *testing.T, nom string, pk []env114Paquet) map[int]bool {
	t.Helper()
	retenus := map[int]bool{}
	t.Logf("L1. HYPOTHESES [%s] — %d paquets ; pFin = fin du var-int de la reference 0", nom, len(pk))
	for pFin := larg114PFinMin; pFin <= larg114PFinMax; pFin++ {
		v := larg114Evalue(pk, pFin)
		marque := " "
		if larg114Retient(v) {
			marque = "*"
			retenus[pFin] = true
		}
		t.Logf("   %s pFin=%2d (sonde+W=%d, soit W=%d avec sonde ou W=%d sans) : queue=%-8s"+
			" porte1=%-8s porte2=%-8s payload R(6) en %d : %d valeurs [%s]",
			marque, pFin, pFin-8, pFin-9, pFin-8, v.queue, v.porte1, v.porte2, pFin+4,
			v.cardPay, v.palette)
	}
	return retenus
}

// larg114Types recense les types d'evenement en tete de tous les paquets delta d'un film :
// c'est la reference a laquelle le critere C5 compare le type relu apres l'enveloppe.
func larg114Types(dir string) map[uint32]int {
	comptes := map[uint32]int{}
	n := filmdec.CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 1 {
				continue
			}
			comptes[uint32(p.Payload(chunk)[0]>>1)]++
		}
	}
	return comptes
}

// larg114Fermeture — critere C5 : le type relu apres l'enveloppe ressemble-t-il a un type
// d'evenement ? Compare la distribution des types relus a celle des types de tete du film.
func larg114Fermeture(t *testing.T, nom string, pk []env114Paquet, pFin int, ref map[uint32]int) {
	t.Helper()
	fin := pFin + 4 + larg114Payload + 1 // payload puis la queue R(1) de l'evenement
	relu := map[uint32]int{}
	for _, p := range pk {
		if fin+7 <= p.nBits {
			relu[filmdec.ReadBitsAtForDiag(p.pay, fin, 7)]++
		}
	}
	var connus, total int
	for v, n := range relu {
		total += n
		if ref[v] > 0 {
			connus += n
		}
	}
	var vs []int
	for v := range relu {
		vs = append(vs, int(v))
	}
	sort.Ints(vs)
	var parts []string
	for i, v := range vs {
		if i == 12 {
			parts = append(parts, "...")
			break
		}
		parts = append(parts, fmt.Sprintf("%d:%d", v, relu[uint32(v)]))
	}
	frac := 0.0
	if total > 0 {
		frac = float64(connus) / float64(total)
	}
	t.Logf("L2. FERMETURE [%s] pFin=%d — type relu au bit %d : %d valeurs distinctes, %.0f %%"+
		" dans les types vus en tete du film ; detail : %s", nom, pFin, fin, len(vs), 100*frac,
		strings.Join(parts, " "))
}

// larg114ReferenceTypes imprime les types de tete les plus frequents du film (mesure L3).
func larg114ReferenceTypes(t *testing.T, nom string, ref map[uint32]int) {
	t.Helper()
	var vs []uint32
	for v := range ref {
		vs = append(vs, v)
	}
	sort.Slice(vs, func(i, j int) bool { return ref[vs[i]] > ref[vs[j]] })
	var parts []string
	for i, v := range vs {
		if i == 12 {
			break
		}
		parts = append(parts, fmt.Sprintf("%d:%d", v, ref[v]))
	}
	t.Logf("L3. REFERENCE [%s] — %d types d'evenement distincts en tete de paquet delta ;"+
		" les plus frequents : %s", nom, len(vs), strings.Join(parts, " "))
}

// larg114Etat porte le resultat du decodage dynamique d'un paquet.
type larg114Etat struct {
	g0, g1, g2 uint32
	posPayload int
	payload    uint32
	ok         bool
}

// larg114Decode rejoue la grammaire du dispatcher sur un paquet, portes LUES et non supposees :
// R(7) type · { R(1) porte ; si 1 : R(W du domaine) + R(2) de generation } x3 · R(6) payload.
// Les largeurs sont fournies par l'appelant (c'est ce que la mesure balaye).
func larg114Decode(p env114Paquet, w2, w3, w7 int) larg114Etat {
	var e larg114Etat
	lire := func(pos, n int) (uint32, bool) {
		if pos+n > p.nBits {
			return 0, false
		}
		return filmdec.ReadBitsAtForDiag(p.pay, pos, n), true
	}
	pos := 7
	var ok bool
	if e.g0, ok = lire(pos, 1); !ok {
		return e
	}
	pos++
	if e.g0 == 1 {
		pos += w2 + 2
	}
	if e.g1, ok = lire(pos, 1); !ok {
		return e
	}
	pos++
	if e.g1 == 1 {
		pos += w3 + 2
	}
	if e.g2, ok = lire(pos, 1); !ok {
		return e
	}
	pos++
	if e.g2 == 1 {
		pos += w7 + 2
	}
	e.posPayload = pos
	if e.payload, ok = lire(pos, larg114Payload); !ok {
		return e
	}
	e.ok = true
	return e
}

// larg114Profil resume une combinaison de largeurs sur un film : palette du payload et portes.
type larg114Profil struct {
	card     int
	palette  map[uint32]int
	partG1   float64
	partG2   float64
	decodes  int
	dominant float64
}

// larg114Mesure applique un jeu de largeurs a un film.
func larg114Mesure(pk []env114Paquet, w2, w3, w7 int) larg114Profil {
	pr := larg114Profil{palette: map[uint32]int{}}
	var g1, g2 int
	for _, p := range pk {
		e := larg114Decode(p, w2, w3, w7)
		if !e.ok {
			continue
		}
		pr.decodes++
		pr.palette[e.payload]++
		g1 += int(e.g1)
		g2 += int(e.g2)
	}
	if pr.decodes == 0 {
		return pr
	}
	pr.card = len(pr.palette)
	pr.partG1 = float64(g1) / float64(pr.decodes)
	pr.partG2 = float64(g2) / float64(pr.decodes)
	max := 0
	for _, n := range pr.palette {
		if n > max {
			max = n
		}
	}
	pr.dominant = float64(max) / float64(pr.decodes)
	return pr
}

// larg114Recouvrement rend la part des paquets du film b dont la valeur de payload existe
// aussi dans la palette du film a — c'est le test de STABILITE INTER-FILMS de la decoupe.
func larg114Recouvrement(a, b larg114Profil) float64 {
	if b.decodes == 0 {
		return 0
	}
	var commun int
	for v, n := range b.palette {
		if a.palette[v] > 0 {
			commun += n
		}
	}
	return float64(commun) / float64(b.decodes)
}

// larg114Balayage — mesure L4 : LE DECODAGE DYNAMIQUE, QUI FAIT FOI.
//
// SEUILS ECRITS AVANT LA MESURE. Un triplet de largeurs (W2, W3, W7) est RETENU si, sur CHAQUE
// film : le payload a une cardinalite <= 8 (c'est un index de siege, pas un compteur) ET la
// palette recouvre >= 90 % des paquets des autres films (une decoupe fausse d'un bit disperse
// les valeurs, et le recouvrement s'effondre). Le balayage couvre W de 4 a 16 par domaine, ce
// qui contient les largeurs predites par la table du binaire (lot B : domaine 2 -> 8,
// domaine 3 -> 8, domaine 7 -> 13).
func larg114Balayage(t *testing.T, films map[string][]env114Paquet, noms []string) {
	t.Helper()
	type sol struct {
		w2, w3, w7 int
		resume     string
	}
	var sols []sol
	for w2 := 4; w2 <= 16; w2++ {
		for w3 := 4; w3 <= 16; w3++ {
			for w7 := 4; w7 <= 16; w7++ {
				profils := make([]larg114Profil, len(noms))
				bon := true
				for i, nom := range noms {
					profils[i] = larg114Mesure(films[nom], w2, w3, w7)
					if profils[i].decodes == 0 || profils[i].card > larg114CardMax {
						bon = false
						break
					}
				}
				if !bon || !larg114Stable(profils) {
					continue
				}
				sols = append(sols, sol{w2, w3, w7, larg114Resume(noms, profils)})
			}
		}
	}
	t.Logf("L4. DECODAGE DYNAMIQUE — %d triplets (W2, W3, W7) retenus sur %d films",
		len(sols), len(noms))
	for i, s := range sols {
		if i == 20 {
			t.Logf("    ... %d autres triplets retenus", len(sols)-20)
			break
		}
		t.Logf("    W2=%2d W3=%2d W7=%2d : %s", s.w2, s.w3, s.w7, s.resume)
	}
}

// larg114Stable applique le critere de recouvrement croise des palettes.
func larg114Stable(profils []larg114Profil) bool {
	for i := range profils {
		for j := range profils {
			if i != j && larg114Recouvrement(profils[i], profils[j]) < 0.90 {
				return false
			}
		}
	}
	return true
}

// larg114Resume met en forme les palettes et les taux d'ouverture des portes.
func larg114Resume(noms []string, profils []larg114Profil) string {
	var parts []string
	for i, nom := range noms {
		var vs []int
		for v := range profils[i].palette {
			vs = append(vs, int(v))
		}
		sort.Ints(vs)
		var pal []string
		for k, v := range vs {
			if k == 6 {
				pal = append(pal, "...")
				break
			}
			pal = append(pal, fmt.Sprintf("%d:%d", v, profils[i].palette[uint32(v)]))
		}
		parts = append(parts, fmt.Sprintf("[%s] payload {%s} portes g1=%.0f%% g2=%.0f%%",
			nom, strings.Join(pal, " "), 100*profils[i].partG1, 100*profils[i].partG2))
	}
	return strings.Join(parts, " · ")
}

// TestViseeLargeurs114 execute L1 a L3 sur chaque film, puis croise les positions retenues.
func TestViseeLargeurs114(t *testing.T) {
	liste := os.Getenv(larg114FilmsEnv)
	if liste == "" {
		t.Skipf("%s absent : instrument saute", larg114FilmsEnv)
	}
	var commun map[int]bool
	var films int
	lots := map[string][]env114Paquet{}
	var noms []string
	for _, dir := range strings.Split(liste, ",") {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		pk := env114Collecte(dir)
		nom := env114Nom(dir)
		if len(pk) == 0 {
			t.Logf("[%s] aucun paquet 114 — film ignore", nom)
			continue
		}
		films++
		lots[nom], noms = pk, append(noms, nom)
		ref := larg114Types(dir)
		larg114ReferenceTypes(t, nom, ref)
		retenus := larg114Tableau(t, nom, pk)
		for pFin := range retenus {
			larg114Fermeture(t, nom, pk, pFin, ref)
		}
		if commun == nil {
			commun = retenus
			continue
		}
		for pFin := range commun {
			if !retenus[pFin] {
				delete(commun, pFin)
			}
		}
	}
	if films < 2 {
		t.Logf("VERDICT — un seul film mesure : rien n'est publiable comme propriete du FORMAT" +
			" (regle d'ecriture du depot).")
		return
	}
	var vs []int
	for pFin := range commun {
		vs = append(vs, pFin)
	}
	sort.Ints(vs)
	switch len(vs) {
	case 0:
		t.Logf("A4. VERDICT — aucune position de fin de var-int ne passe C1..C4 sur les %d films.",
			films)
	case 1:
		t.Logf("A4. VERDICT — SOLUTION UNIQUE sur %d films : pFin=%d, soit sonde presente et"+
			" W(domaine 2)=%d, ou sonde absente et W(domaine 2)=%d ; payload R(6) au bit %d.",
			films, vs[0], vs[0]-9, vs[0]-8, vs[0]+4)
	default:
		t.Logf("A4. VERDICT — AMBIGUITE RESIDUELLE sur %d films : %d positions passent C1..C4"+
			" (%v). Les departager demande une mesure hors de ce lot.", films, len(vs), vs)
	}
	larg114Balayage(t, lots, noms)
}
