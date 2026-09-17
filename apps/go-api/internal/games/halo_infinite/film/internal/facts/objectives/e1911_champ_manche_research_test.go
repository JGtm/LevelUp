//go:build research

package objectives

// e1911_champ_manche_research_test.go — INSTRUMENT 1.9.11 / D1 : LE `2` EST-IL LU AU MEME
// DEPLACEMENT QUE LE `1` DES VRAIES MANCHES, OU EST-CE UNE LECTURE QUI DERAPE ?
//
// # LA QUESTION (precision demandee par le pilote, 2026-09-16)
//
// Le lot 1.9.11 a mesure que 24 films du cache portent un designateur de manche `2` MATERIEL
// avec la manche 1 absente, et que 23 des 24 ont fini DANS leur temps reglementaire sur un mode
// SANS manche. D1 le consigne comme un artefact SANS en nommer la cause. La question posee :
// l'enregistrement qui porte ce `2` lit-il le MEME champ de 5 bits (`FUN_140C18794`, deux
// en-tetes de 5 bits en tete du PREMIER composant) au MEME deplacement depuis l'en-tete
// d'enregistrement, ou bien la lecture DERAPE-t-elle — comme au lot 1.9.1 bis, ou une largeur
// dependante des donnees faisait derouler un record a `n2 == 0` sur du bruit ?
//
// # CE QUE L'INSTRUMENT RELEVE, ET POURQUOI CES GRANDEURS-LA
//
// Pour chaque enregistrement retenu par la production ([scanFrameForRecords]) :
//
//	chunk / pidx     le paquet, tel que `types.Packet` le nomme deja (Chunk, Index).
//	bit              le bit de l'en-tete d'enregistrement dans la charge du paquet.
//	at               le bit du CHAMP DE 5 BITS (le premier des deux en-tetes du composant).
//	at-bit           LE DEPLACEMENT. C'est la reponse directe a la question : la grammaire le
//	                 fixe a `14 + 2 + 4 + 6n` (liste creuse, n composants) ou `14 + 2 + 1 + 64`
//	                 (liste dense), soit 26, 32, 38, 44, 50, 56, 62 ou 81. Toute autre valeur
//	                 serait une lecture d'une autre forme.
//	dense / n        la forme de liste et le nombre de composants annonce.
//	h1 h2            les deux en-tetes de 5 bits. La production EXIGE h1 == h2 ; les imprimer
//	                 tous les deux dit si la contrainte est tenue.
//	bits             les 8 bits AVANT le champ, les 10 bits du champ, les 8 bits APRES.
//	fin              le dernier bit consomme par l'enregistrement (en-tete + composants).
//	chevauche        L'ENREGISTREMENT TOMBE-T-IL DANS LA PORTEE D'UN AUTRE ? C'est le
//	                 discriminant : [scanFrameForRecords] essaie TOUTES les positions de bit du
//	                 paquet. Un enregistrement vrai commence ou le precedent finit ; un ancrage
//	                 fortuit est trouve DANS les octets d'un autre, et ne peut donc pas etre une
//	                 emission du moteur.
//
// L'instrument ne conclut pas : il imprime, et le releve va en D1 tel quel.
//
// USAGE (gardes par environnement, saute sans elles) :
//
//	FILM_CACHE_ROOT=C:/.../data/cache E1911_CHAMP_FILMS=fb1a1a72,72b0a25e,64e8adfa \
//	E1911_CHAMP_DESIG=2,2,1 \
//	  go test -tags research ./internal/games/halo_infinite/film/internal/facts/objectives/ -run E1911Champ -v

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

const (
	// e1911ChampFilmsEnv : les films a relever (short8, virgules).
	e1911ChampFilmsEnv = "E1911_CHAMP_FILMS"
	// e1911ChampDesigEnv : le designateur a relever pour chaque film, dans le MEME ordre.
	e1911ChampDesigEnv = "E1911_CHAMP_DESIG"
	// e1911ChampMaxLignes borne le detail imprime par population (le releve reste lisible ; les
	// comptes, eux, portent sur la population entiere).
	e1911ChampMaxLignes = 12
)

// e1911Site est un enregistrement retenu, avec tout ce qui permet de le situer a l'octet.
type e1911Site struct {
	Chunk, Pidx int
	Bit, At     int
	Dense       bool
	NComp       int
	H1, H2      int
	Round       int
	Fin         int
	Chevauche   bool
	Avant       string
	Champ       string
	Apres       string
}

// Deplacement rend `at - bit` : le deplacement du champ depuis l'en-tete d'enregistrement.
func (s e1911Site) Deplacement() int { return s.At - s.Bit }

// Forme nomme la forme de liste de composants de l'enregistrement.
func (s e1911Site) Forme() string {
	if s.Dense {
		return "DENSE"
	}
	return fmt.Sprintf("creuse n=%d", s.NComp)
}

// TestE1911ChampDeManche imprime le releve, film par film.
func TestE1911ChampDeManche(t *testing.T) {
	if cacheRoot() == "" {
		t.Skipf("%s absent : instrument saute", filmCacheEnv)
	}
	films, desig := e1911ChampEntree(t)
	for i, id := range films {
		film, ok := newDiskFilm(t, id)
		if !ok {
			t.Logf("%s : ECARTE (absent du cache)", id)
			continue
		}
		e1911ImprimeChamp(t, id, desig[i], e1911Sites(film))
	}
}

// e1911ChampEntree lit les deux listes d'environnement et exige qu'elles s'apparient.
func e1911ChampEntree(t *testing.T) (films []string, desig []int) {
	t.Helper()
	fs := e1911Decoupe(os.Getenv(e1911ChampFilmsEnv))
	ds := e1911Decoupe(os.Getenv(e1911ChampDesigEnv))
	if len(fs) == 0 || len(ds) == 0 {
		t.Skipf("%s ou %s absent : instrument saute", e1911ChampFilmsEnv, e1911ChampDesigEnv)
	}
	if len(fs) != len(ds) {
		t.Fatalf("%d film(s) pour %d designateur(s) : les deux listes doivent s'apparier",
			len(fs), len(ds))
	}
	desig = make([]int, len(ds))
	for i, s := range ds {
		v, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("designateur %q illisible : %v", s, err)
		}
		desig[i] = v
	}
	return fs, desig
}

// e1911Decoupe rend les parties non vides d'une liste separee par des virgules.
func e1911Decoupe(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// e1911Sites rejoue la boucle de [StatRecordsCtx] et releve, pour chaque enregistrement RETENU,
// ou son champ de manche a ete lu.
//
// AUCUNE LIGNE DE PRODUCTION N'EST TOUCHEE : l'instrument appelle [matchRecordHeader] et
// [decodeComponents], exactement comme [scanFrameForRecords], et refait le meme filtre.
func e1911Sites(film *source.Film) []e1911Site {
	var out []e1911Site
	for _, c := range manifestChunks(film) {
		for _, f := range framesOf(film, c.pos) {
			out = append(out, e1911SitesDuPaquet(f)...)
		}
	}
	return out
}

// e1911SitesDuPaquet balaie UN paquet et rend ses enregistrements retenus, chevauchements
// calcules.
func e1911SitesDuPaquet(f types.Packet) []e1911Site {
	pay := f.Payload
	var out []e1911Site
	lim := len(pay)*8 - statTailBits
	for b := 1; b < lim; b++ {
		_, idx, at, ok := matchRecordHeader(pay, b)
		if !ok {
			continue
		}
		comps, round := decodeComponents(pay, at, idx)
		if len(comps) == 0 || !statCountersInDomain(comps) {
			continue
		}
		s := e1911SiteDe(f, pay, b, at, idx)
		s.Round = round
		out = append(out, s)
	}
	e1911MarqueChevauchements(out)
	return out
}

// e1911SiteDe assemble le releve d'UN enregistrement : sa position, sa forme, ses deux en-tetes
// et les bits autour du champ.
func e1911SiteDe(f types.Packet, pay []byte, b, at int, idx []int) e1911Site {
	return e1911Site{
		Chunk: f.Chunk, Pidx: f.Index, Bit: b, At: at,
		Dense: source.BitsTronques(pay, b+statIDBits+statGenBits, 1) == 1,
		NComp: len(idx),
		H1:    int(source.BitsTronques(pay, at, statHdrBits)),
		H2:    int(source.BitsTronques(pay, at+statHdrBits, statHdrBits)),
		Fin:   e1911FinDe(pay, at, idx),
		Avant: e1911Bits(pay, at-8, 8),
		Champ: e1911Bits(pay, at, 2*statHdrBits),
		Apres: e1911Bits(pay, at+2*statHdrBits, 8),
	}
}

// e1911FinDe rend le dernier bit consomme par les composants de l'enregistrement.
func e1911FinDe(pay []byte, at int, idx []int) int {
	q := at
	for range idx {
		_, w, ok := decodeStatComponent(pay, q)
		if !ok {
			break
		}
		q += w
	}
	return q
}

// e1911MarqueChevauchements dit, pour chaque enregistrement, s'il tombe dans la PORTEE d'un
// autre enregistrement retenu du meme paquet.
//
// LE DISCRIMINANT DU RELEVE : le balayage essaie toutes les positions de bit, donc un ancrage
// fortuit est trouve DANS les octets d'un enregistrement voisin. Une emission du moteur, elle,
// commence ou la precedente finit.
func e1911MarqueChevauchements(sites []e1911Site) {
	for i := range sites {
		for j := range sites {
			if i == j {
				continue
			}
			if sites[i].Bit > sites[j].Bit && sites[i].Bit < sites[j].Fin {
				sites[i].Chevauche = true
				break
			}
		}
	}
}

// e1911Bits rend n bits a partir de `p`, en clair, ou des points si la position sort du paquet.
func e1911Bits(pay []byte, p, n int) string {
	if p < 0 || p+n > len(pay)*8 {
		return strings.Repeat(".", n)
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "%d", source.BitsTronques(pay, p+i, 1))
	}
	return b.String()
}

// e1911ImprimeChamp imprime le releve d'un film : le designateur demande, puis la manche 0 du
// MEME film, qui sert de temoin.
func e1911ImprimeChamp(t *testing.T, id string, desig int, sites []e1911Site) {
	t.Helper()
	t.Logf("=== %s — designateur releve : %d (temoin du meme film : manche 0) ===", id, desig)
	for _, r := range []int{desig, 0} {
		pop := e1911Population(sites, r)
		t.Logf("  -- manche %d : %d enregistrement(s) --", r, len(pop))
		e1911ImprimeComptes(t, pop)
		e1911ImprimeLignes(t, pop)
	}
}

// e1911Population rend les enregistrements d'un designateur.
func e1911Population(sites []e1911Site, round int) []e1911Site {
	var out []e1911Site
	for _, s := range sites {
		if s.Round == round {
			out = append(out, s)
		}
	}
	return out
}

// e1911ImprimeComptes imprime les grandeurs qui repondent a la question, sur la population
// ENTIERE : la repartition des deplacements, la forme de liste, et le chevauchement.
func e1911ImprimeComptes(t *testing.T, pop []e1911Site) {
	t.Helper()
	if len(pop) == 0 {
		return
	}
	depl, forme := map[int]int{}, map[string]int{}
	chev, egaux := 0, 0
	for _, s := range pop {
		depl[s.Deplacement()]++
		forme[s.Forme()]++
		if s.Chevauche {
			chev++
		}
		if s.H1 == s.H2 {
			egaux++
		}
	}
	t.Logf("     deplacements (at-bit) : %s", e1911Compte(depl))
	t.Logf("     forme de liste        : %s", e1911CompteS(forme))
	t.Logf("     CHEVAUCHE un autre enregistrement : %d sur %d (%.0f %%)",
		chev, len(pop), 100*float64(chev)/float64(len(pop)))
	t.Logf("     h1 == h2 : %d sur %d (la production l'EXIGE, donc 100 %% attendu)", egaux, len(pop))
	e1911ImprimeRepetitions(t, pop)
}

// e1911ImprimeRepetitions dit si la population est VARIEE ou REPETEE : combien de motifs de bits
// distincts autour du champ, combien de positions de bit distinctes, combien de chunks.
//
// C'est la seconde moitie de la reponse a D1. Le deplacement dit OU le champ est lu ; ces trois
// comptes disent si ce qu'on lit est une population d'emissions differentes ou la meme suite
// d'octets retrouvee au meme endroit, paquet apres paquet.
func e1911ImprimeRepetitions(t *testing.T, pop []e1911Site) {
	t.Helper()
	motifs, bits, chunks := map[string]int{}, map[int]int{}, map[int]int{}
	for _, s := range pop {
		motifs[s.Avant+" "+s.Champ+" "+s.Apres]++
		bits[s.Bit]++
		chunks[s.Chunk]++
	}
	t.Logf("     motifs de bits DISTINCTS (8 avant + champ + 8 apres) : %d pour %d enregistrement(s) ; les 3 plus frequents : %s",
		len(motifs), len(pop), e1911Tete(motifs, 3))
	t.Logf("     positions de bit DISTINCTES dans le paquet : %d ; les 3 plus frequentes : %s",
		len(bits), e1911Tete(e1911EnChaines(bits), 3))
	t.Logf("     chunks : %s", e1911Compte(chunks))
}

// e1911EnChaines convertit une table entier -> compte en table chaine -> compte.
func e1911EnChaines(m map[int]int) map[string]int {
	out := make(map[string]int, len(m))
	for k, v := range m {
		out[strconv.Itoa(k)] = v
	}
	return out
}

// e1911Tete rend les n entrees les plus frequentes d'une table, la plus frequente d'abord.
func e1911Tete(m map[string]int, n int) string {
	type paire struct {
		cle string
		n   int
	}
	all := make([]paire, 0, len(m))
	for k, v := range m {
		all = append(all, paire{k, v})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].n != all[j].n {
			return all[i].n > all[j].n
		}
		return all[i].cle < all[j].cle
	})
	parts := make([]string, 0, n)
	for i, p := range all {
		if i >= n {
			break
		}
		parts = append(parts, fmt.Sprintf("%q x%d", p.cle, p.n))
	}
	return strings.Join(parts, " ")
}

// e1911Compte rend une table entier -> compte, triee par cle.
func e1911Compte(m map[int]int) string {
	cles := make([]int, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	parts := make([]string, 0, len(cles))
	for _, k := range cles {
		parts = append(parts, fmt.Sprintf("%d:%d", k, m[k]))
	}
	return strings.Join(parts, " ")
}

// e1911CompteS rend une table chaine -> compte, triee par cle.
func e1911CompteS(m map[string]int) string {
	cles := make([]string, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	parts := make([]string, 0, len(cles))
	for _, k := range cles {
		parts = append(parts, fmt.Sprintf("%s:%d", k, m[k]))
	}
	return strings.Join(parts, " ")
}

// e1911ImprimeLignes imprime le detail, enregistrement par enregistrement, borne a
// [e1911ChampMaxLignes].
func e1911ImprimeLignes(t *testing.T, pop []e1911Site) {
	t.Helper()
	if len(pop) == 0 {
		return
	}
	t.Logf("     %-6s %-6s %-8s %-7s %-12s %-3s %-3s %-9s %-11s %-9s %s",
		"chunk", "pidx", "bit", "at-bit", "forme", "h1", "h2",
		"8 avant", "champ 5+5", "8 apres", "chevauche")
	for i, s := range pop {
		if i >= e1911ChampMaxLignes {
			t.Logf("     ... (%d enregistrement(s) au total)", len(pop))
			break
		}
		t.Logf("     %-6d %-6d %-8d %-7d %-12s %-3d %-3d %-9s %-11s %-9s %v",
			s.Chunk, s.Pidx, s.Bit, s.Deplacement(), s.Forme(), s.H1, s.H2,
			s.Avant, s.Champ, s.Apres, s.Chevauche)
	}
}
