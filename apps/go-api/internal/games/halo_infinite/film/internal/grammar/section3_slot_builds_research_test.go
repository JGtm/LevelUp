package grammar

// section3_slot_builds_research_test.go — LE PROFIL PAR BUILD DE LA TABLE DES SLOTS.
//
// ## POURQUOI CE FICHIER EXISTE
//
// La phase 1 a conclu « le cache local ne contient plus que deux builds » et n'a donc verifie
// la table des slots que sur `HI_1_12_0` et `HI_1_13_0`. C'EST FAUX : `TestD1Builds` classe les
// 1 351 films du cache en **13 groupes (build, version, cardinal de la table par type)** portant
// **7 builds distincts** — `HI_1_13_0`, `HI_1_12_0`, `HI_1_11_0`, `HI_1_10_0`, `HI_1_9_0`,
// `HI_1_8_0`, `HI_1_4_1` — plus 5 films sans section d'identification. Une premisse negative se
// re-teste a chaque reprise (methode, erreur A) ; celle-la ne l'avait pas ete.
//
// Ce fichier mesure donc, sur TOUT le cache et sans liste de films ecrite a la main, ce que la
// grammaire du slot devient d'un build a l'autre. C'est une donnee de PROFIL pour le chantier
// decodeur : elle dit ce qu'un decodeur doit parametrer et ce qu'il peut tenir pour acquis.
//
// ## LES CONTROLES, ECRITS AVANT LA MESURE
//
//	B-ENT  OU EST LA CHAINE DE BUILD DANS L'EN-TETE ? La carte de la phase 1 la place a
//	       `0x0CB414`, juste apres une table par type de 123 u32. Or `TestD1Builds` mesure des
//	       cardinaux differents selon le build (124, 123, 122, 117) : si le cardinal change, tout
//	       ce qui suit la table se DECALE de 4 octets par entree. Le test ne suppose rien : il
//	       CHERCHE le prefixe `HI_` dans la zone d'en-tete et rend l'offset trouve, par build.
//	       Critere : un seul offset par build, et l'ecart a `0x0CB414` doit valoir un multiple
//	       de 4 — sinon l'explication par le cardinal de la table est fausse.
//	B-ENR  DE COMBIEN LA LONGUEUR D'UN ENREGISTREMENT DE SLOT CHANGE-T-ELLE ? La grammaire lue
//	       dans l'exe courant (`HI_1_13_0`) predit une longueur a partir de quatre nombres lus
//	       dans le flux. Sur un autre build, la partie FIXE du sous-enregistrement peut avoir
//	       change de taille. Le test mesure, pour chaque film, l'ecart `mesure - predit` sur tous
//	       les enregistrements consecutifs, et verifie qu'UN SEUL ecart couvre tout le film.
//	       Critere ecrit d'avance : un changement de build change la partie FIXE, donc le meme
//	       ecart sur TOUS les enregistrements d'un film. Si l'ecart variait d'un joueur a
//	       l'autre, la lecture serait fausse et non « transposable a une constante pres ».
//	B-NOM  LE NOM ET LE XUID SURVIVENT-ILS ? Sur chaque build, le balayage doit continuer a
//	       rendre des XUID de la plage Xbox ET des gamertags imprimables. C'est le controle qui
//	       dit si la tete de l'enregistrement (85 bits + XUID) et le champ de nom sont stables
//	       a travers les builds, independamment du decalage de B-ENR.
//
// Garde `CHUNK00_CORPUS` = la racine de `film_chunks/` (+ `CHUNK00_CORPUS_MAX`, defaut 1400).
// Lecture seule, aucun code de production touche.

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// s3bEnteteFin : borne haute de la recherche de la chaine de build. Elle s'arrete AVANT le
// corps (`0x0CE68C`) : le corps est bit-packe et une occurrence de `HI_` y serait fortuite.
// La borne basse est l'octet 0x20, comme chez `lireEntete` — un `HI_` plus tot ne peut pas
// avoir de champ version devant lui.
const s3bEnteteFin = 0x0CE68C

// s3bProfil : ce qu'on mesure pour un build.
type s3bProfil struct {
	films          int
	offsetsEntete  map[int]int // offset de la chaine de build -> nombre de films
	ecarts         map[int]int // (mesure - predit) -> nombre d'enregistrements
	filmsUnEcart   int         // films ou UN SEUL ecart couvre tous les enregistrements
	enrs, nomsBons int
}

// s3bBuild cherche le prefixe `HI_` dans la zone d'en-tete et rend (chaine, offset). Rend
// ("", -1) si le film ne porte pas de section d'identification.
func s3bBuild(d []byte) (string, int) {
	if len(d) < s3bEnteteFin {
		return "", -1
	}
	for off := 0x20; off < s3bEnteteFin; off++ {
		if d[off] == 'H' && d[off+1] == 'I' && d[off+2] == '_' {
			return s3wChaine(d, off, 32), off
		}
	}
	return "", -1
}

// TestSection3SlotProfilBuilds execute B-ENT, B-ENR et B-NOM sur tout le cache.
func TestSection3SlotProfilBuilds(t *testing.T) {
	racine := os.Getenv("CHUNK00_CORPUS")
	if racine == "" {
		t.Skip("CHUNK00_CORPUS absent : profil par build saute")
	}
	max := 1400
	if v, err := strconv.Atoi(os.Getenv("CHUNK00_CORPUS_MAX")); err == nil && v > 0 {
		max = v
	}
	entrees, err := os.ReadDir(racine)
	if err != nil {
		t.Fatalf("lecture de %s : %v", racine, err)
	}
	profils, lus := map[string]*s3bProfil{}, 0
	for _, e := range entrees {
		if !e.IsDir() || lus >= max {
			continue
		}
		d, err := os.ReadFile(filepath.Join(racine, e.Name(), "chunk_00.bin"))
		if err != nil || len(d) < s3bEnteteFin {
			continue
		}
		lus++
		s3bMesureFilm(d, profils)
	}
	s3bRapport(t, profils, lus)
}

// s3bMesureFilm ajoute un film au profil de son build.
func s3bMesureFilm(d []byte, profils map[string]*s3bProfil) {
	build, off := s3bBuild(d)
	cle := build
	if off < 0 {
		cle = "(sans section d'identification)"
	}
	p := profils[cle]
	if p == nil {
		p = &s3bProfil{offsetsEntete: map[int]int{}, ecarts: map[int]int{}}
		profils[cle] = p
	}
	p.films++
	p.offsetsEntete[off]++
	// Le balayage rend, sur certains films, des positions PARASITES (motif d'en-tete fortuit a
	// l'interieur d'un vrai enregistrement, reconnaissable a son champ de nom illisible). Elles
	// faussent les DEUX ecarts qui les encadrent, donc on les ecarte AVANT de mesurer l'ecart
	// de build — sans quoi la mesure melange un effet de build et un artefact de detection.
	var es []*s3sEnr
	for _, en := range s3bEnrs(d) {
		p.enrs++
		if !s3sImprimable(en.gamertag) {
			continue
		}
		p.nomsBons++
		es = append(es, en)
	}
	vus := map[int]int{}
	for i, en := range es {
		if i+1 < len(es) {
			ec := (es[i+1].debut - en.debut) - s3sPredite(en)
			p.ecarts[ec]++
			vus[ec]++
		}
	}
	if len(vus) == 1 {
		p.filmsUnEcart++
	}
}

// s3bEnrs decode les enregistrements de la grappe terminale. Identique a `s3sEnrs` mais sans
// l'exigence d'un flux complet : sur les builds anciens la longueur predite est fausse d'une
// constante, ce qui est precisement ce que B-ENR mesure.
func s3bEnrs(d []byte) []*s3sEnr {
	fin := (dernierNonNul(d) + 1) * 8
	g := s3rGrappe(s3rBalayage(d, fin))
	out := make([]*s3sEnr, 0, len(g))
	for _, h := range g {
		if e := s3sDecode(d, h.bit-s3rEnteteBits, fin); e != nil {
			out = append(out, e)
		}
	}
	return out
}

// s3bRapport imprime les trois bilans, un build par ligne.
func s3bRapport(t *testing.T, profils map[string]*s3bProfil, lus int) {
	t.Helper()
	var cles []string
	for k := range profils {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool {
		return profils[cles[i]].films > profils[cles[j]].films
	})
	t.Logf("%d chunk_00 lus ; %d build(s) distinct(s)", lus, len(profils))
	for _, k := range cles {
		p := profils[k]
		t.Logf("=== BILAN B-ENT === %-32s %4d film(s) ; chaine de build aux offset(s) %v "+
			"(la carte de la phase 1 dit 0x%06x)", k, p.films, s3bOffsets(p.offsetsEntete),
			s3wBuildOff)
		t.Logf("=== BILAN B-ENR === %-32s ecart(s) `mesure - predit` %v ; %d/%d film(s) ou "+
			"UN SEUL ecart couvre tous les enregistrements", k, s3pDist(p.ecarts),
			p.filmsUnEcart, p.films)
		t.Logf("=== BILAN B-NOM === %-32s %d/%d enregistrement(s) rendent un gamertag "+
			"imprimable", k, p.nomsBons, p.enrs)
	}
}

// s3bOffsets rend les offsets trouves, en hexadecimal et avec leur ecart a la carte v1.
func s3bOffsets(m map[int]int) []string {
	var ks []int
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	out := make([]string, 0, len(ks))
	for _, k := range ks {
		if k < 0 {
			out = append(out, "absente x"+strconv.Itoa(m[k]))
			continue
		}
		out = append(out, "0x"+strconv.FormatInt(int64(k), 16)+" (ecart "+
			strconv.Itoa(k-s3wBuildOff)+") x"+strconv.Itoa(m[k]))
	}
	return out
}

// TestSection3SlotProfilExemples imprime, pour chaque build rencontre, le PREMIER film et ses
// enregistrements — le releve dont on part pour porter la grammaire sur un build donne.
func TestSection3SlotProfilExemples(t *testing.T) {
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		build, off := s3bBuild(d)
		es := s3bEnrs(d)
		var ecarts []string
		for i, e := range es {
			if i+1 < len(es) {
				ecarts = append(ecarts, strconv.Itoa((es[i+1].debut-e.debut)-s3sPredite(e)))
			}
		}
		t.Logf("=== %s === build %q a l'offset 0x%x ; %d enregistrement(s) ; "+
			"ecarts `mesure - predit` [%s]", filepath.Base(dir), build, off, len(es),
			strings.Join(ecarts, " "))
		for i, e := range es {
			t.Logf("  slot %2d xuid %19d gt %-16q masque %4d N %4d M %3d predite %6d",
				i, e.xuid, e.gamertag, e.compteMasque, e.n, e.m, s3sPredite(e))
		}
	}
}
