package grammar

// section3_slot_ordre_research_test.go — L'ORDRE DES SLOTS, L'EQUIPE, ET LE TEXTE DU 30/08.
//
// Ce fichier prolonge `section3_slot_grammar_research_test.go` (la grammaire et le decodeur) sur
// les questions laissees ouvertes par la phase 1.
//
//	G-IDX  L'ORDRE DES ENREGISTREMENTS EST-IL LE `player_index` DE LA PRODUCTION ? Aujourd'hui
//	       `killcollector.resolvePlayerIndices` reconstruit `indice -> xuid` en cherchant le
//	       motif du xuid dans le flux de replication et en lisant les 5 bits qui le precedent.
//	       Si le RANG d'un enregistrement dans la table de `chunk_00` vaut le `filmIndex`, une
//	       table explicite remplace cette inference. Reference : `roster[].filmIndex` des
//	       documents de rejeu deja produits.
//	G-EQP  OU EST L'EQUIPE ? Deux lentilles distinctes (methode, regle 7), et c'est voulu :
//	       (a) REFERENCE EXTERNE — un vecteur d'equipes dans l'ordre des slots, fourni par
//	           `CHUNK00_EQUIPES`, releve dans `match_participants` ; un champ n'est retenu que
//	           s'il coincide sur TOUS les slots de TOUS les films qui en ont un.
//	       (b) ORACLE INTERNE — sans aucune entree externe : un champ d'equipe doit PARTITIONNER
//	           chaque roster en deux moities de taille egale (une partie d'arene a 4 contre 4).
//	           Un champ a faible cardinal peut coincider par hasard sur un film ; il ne peut pas
//	           rendre un partage equilibre sur tous.
//	       Un champ CONSTANT par joueur d'un match a l'autre n'est PAS l'equipe : le meme joueur
//	       change d'equipe. Ce filtre est ecrit avant la mesure.
//	G-TXT  LA CONTRADICTION DU 30/08, TRANCHEE PAR UNE MESURE. Le 30/08 relevait des gamertags
//	       lisibles en UTF-16 aligne sur l'octet a des positions relatives tres differentes
//	       selon l'enregistrement — « soit plusieurs champs texte, soit une lecture fortuite »
//	       (question ouverte n°1 de la phase 1). Ce test REJOUE ce balayage et confronte CHAQUE
//	       touche aux DEUX champs de texte que la grammaire connait desormais : le champ de nom
//	       (`sub+0xc14`, unite par unite par `FUN_1407ece18`) et le bloc de queue (`sub+0x1400`,
//	       recopie brut). Une touche tombe dans l'un des deux, ou reste hors grammaire.
//	G-T44  LE BLOC DE QUEUE PORTE-T-IL LE NOM ? `sub+0x1400` fait 44 octets, soit 22 unites
//	       UTF-16 — la taille d'un gamertag. L'offset du nom dans le bloc est CHERCHE, pas
//	       suppose, et rendu par build.
//
// Garde `CHUNK00_FILMS` (+ `CHUNK00_REPLAYS` pour G-IDX, + `CHUNK00_EQUIPES` pour G-EQP (a)).
// Lecture seule, aucun code de production touche.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"
)

// s3oTexteMin : longueur minimale d'une suite UTF-16LE imprimable retenue par G-TXT. Le 30/08
// travaillait sur des gamertags ; quatre caracteres est le plancher le plus permissif possible,
// choisi AVANT la mesure pour ne pas rater de touche.
const s3oTexteMin = 4

// s3sJoueur : une ligne du `roster` d'un document de rejeu (la reference EXTERNE).
type s3sJoueur struct {
	XUID string `json:"xuid"`
	Nom  string `json:"name"`
	Idx  int    `json:"filmIndex"`
}

// s3sRef : la reference externe pour un XUID.
type s3sRef struct {
	nom    string
	idx    int
	equipe int
}

// s3sOracle rend `xuid -> (nom, filmIndex, equipe)` pour un film.
//
// `CHUNK00_REPLAYS` est une liste de repertoires separes par `;` : les documents de rejeu du
// depot vivent a plusieurs endroits (cache courant, sauvegardes), et un chantier de recherche
// n'a pas a les recopier pour les confronter.
//
// L'equipe ne vient PAS du document de rejeu : son champ `tracks[].team` vaut -1 sur tous les
// documents mesures (schema 2 et 53), c'est-a-dire non renseigne. Elle arrive par
// `CHUNK00_EQUIPES`, un vecteur d'equipes DANS L'ORDRE DES SLOTS releve dans
// `match_participants` — la meme source externe que les rosters de la phase 1.
func s3sOracle(dir string) map[uint64]s3sRef {
	roster := s3oRoster(dir)
	if len(roster) == 0 {
		return nil
	}
	eq := s3oEquipes(os.Getenv("CHUNK00_EQUIPES"))[filepath.Base(dir)]
	out := map[uint64]s3sRef{}
	for _, j := range roster {
		x, err := strconv.ParseUint(j.XUID, 10, 64)
		if err != nil {
			continue
		}
		e := -1
		if j.Idx >= 0 && j.Idx < len(eq) {
			e = eq[j.Idx]
		}
		out[x] = s3sRef{nom: j.Nom, idx: j.Idx, equipe: e}
	}
	return out
}

// s3oRoster lit le `roster` du document de rejeu du film, dans le premier repertoire de
// `CHUNK00_REPLAYS` qui le porte.
func s3oRoster(dir string) []s3sJoueur {
	for _, root := range strings.Split(os.Getenv("CHUNK00_REPLAYS"), ";") {
		if root = strings.TrimSpace(root); root == "" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, filepath.Base(dir)+".json"))
		if err != nil {
			continue
		}
		var doc struct {
			Roster []s3sJoueur `json:"roster"`
		}
		if json.Unmarshal(b, &doc) == nil && len(doc.Roster) > 0 {
			return doc.Roster
		}
	}
	return nil
}

// s3oEquipes decoupe `film=equipe,equipe,...;film=...` : un vecteur d'equipes par film, dans
// l'ordre des slots.
func s3oEquipes(v string) map[string][]int {
	out := map[string][]int{}
	for _, p := range strings.Split(v, ";") {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) != 2 {
			continue
		}
		var es []int
		for _, s := range strings.Split(kv[1], ",") {
			n, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil {
				n = -1
			}
			es = append(es, n)
		}
		if len(es) > 0 {
			out[strings.TrimSpace(kv[0])] = es
		}
	}
	return out
}

// TestSection3SlotIndex execute G-IDX : le rang d'un enregistrement vaut-il le `filmIndex` ?
//
// Le test mesure DEUX choses, et la seconde est la bonne question. L'egalite stricte
// `rang == filmIndex` echoue des que le lecteur rate le debut de la table (la grappe terminale
// peut perdre ses premiers enregistrements) : tous les rangs sont alors decales du meme nombre.
// Le test rend donc aussi l'ensemble des `filmIndex - rang` du film : s'il est REDUIT A UNE
// SEULE VALEUR, l'ORDRE est celui du `filmIndex`, a une troncature de tete pres — ce qui est un
// defaut de lecture, pas un desaccord de semantique. Une vraie permutation donnerait plusieurs
// valeurs.
func TestSection3SlotIndex(t *testing.T) {
	egaux, vus, films, filmsEgaux, filmsUnDecalage := 0, 0, 0, 0, 0
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		ref := s3sOracle(dir)
		if len(ref) == 0 {
			t.Logf("=== %s === pas d'oracle (CHUNK00_REPLAYS)", filepath.Base(dir))
			continue
		}
		films++
		bons, cptes, dec := s3oIndexFilm(t, dir, s3sChaine(d), ref)
		egaux += bons
		vus += cptes
		if cptes > 0 && bons == cptes {
			filmsEgaux++
		}
		if len(dec) == 1 {
			filmsUnDecalage++
		}
	}
	t.Logf("=== BILAN G-IDX === %d/%d slots ont rang == filmIndex ; %d/%d films ou la "+
		"coincidence est TOTALE ; %d/%d films ou `filmIndex - rang` est CONSTANT (donc ordre "+
		"correct a une troncature de tete pres)", egaux, vus, filmsEgaux, films,
		filmsUnDecalage, films)
}

// s3oIndexFilm confronte les rangs d'un film a l'oracle et rend (egalites, compares, decalages).
func s3oIndexFilm(t *testing.T, dir string, es []*s3sEnr, ref map[uint64]s3sRef) (int, int,
	map[int]int) {
	t.Helper()
	bons, cptes := 0, 0
	dec := map[int]int{}
	var lignes []string
	for i, e := range es {
		r, ok := ref[e.xuid]
		if !ok {
			lignes = append(lignes, "rang "+strconv.Itoa(i)+" -> HORS ORACLE")
			continue
		}
		cptes++
		if r.idx == i {
			bons++
		}
		dec[r.idx-i]++
		lignes = append(lignes, "rang "+strconv.Itoa(i)+" -> filmIndex "+
			strconv.Itoa(r.idx)+" ("+r.nom+")")
	}
	t.Logf("=== %s === %d/%d rangs egaux au filmIndex ; decalage(s) `filmIndex - rang` %v : %v",
		filepath.Base(dir), bons, cptes, s3pDist(dec), lignes)
	return bons, cptes, dec
}

// s3oNomsCourts : l'ordre des champs candidats de G-EQP, aligne sur s3oCourts.
func s3oNomsCourts() []string {
	return []string{"b2(2b)", "f10(10b)", "f14(14b)", "f6(6b,signe)", "f8(8b)", "f7(7b)",
		"f1(1b)", "repr(32b)", "u32Tete(32b)"}
}

// s3oCourts rend les valeurs des champs candidats, dans l'ordre de s3oNomsCourts.
func s3oCourts(e *s3sEnr) []int {
	return []int{int(e.b2), int(e.f10), int(e.f14), e.f6, int(e.f8), int(e.f7),
		int(e.f1), int(e.repr), int(e.u32Tete)}
}

// TestSection3SlotEquipe execute G-EQP : lequel des champs courts porte l'equipe ? Deux
// lentilles, la reference externe et le partage equilibre interne.
func TestSection3SlotEquipe(t *testing.T) {
	noms := s3oNomsCourts()
	bons, equil, partages := make([]int, len(noms)), make([]int, len(noms)), 0
	vus := 0
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		es := s3sChaine(d)
		if len(es) == 0 {
			continue
		}
		partages++
		vus += s3oEquipeExterne(t, dir, es, bons)
		s3oEquipeInterne(t, dir, es, equil)
	}
	for k, n := range noms {
		t.Logf("=== BILAN G-EQP === %-14s (a) coincide avec l'equipe de reference sur %d/%d "+
			"slots ; (b) partage le roster en deux moities egales sur %d/%d films",
			n, bons[k], vus, equil[k], partages)
	}
}

// s3oEquipeExterne execute la lentille (a) : confrontation au vecteur d'equipes de reference.
func s3oEquipeExterne(t *testing.T, dir string, es []*s3sEnr, bons []int) int {
	t.Helper()
	ref := s3sOracle(dir)
	vus := 0
	for i, e := range es {
		r, ok := ref[e.xuid]
		if !ok || r.equipe < 0 {
			continue
		}
		vus++
		for k, v := range s3oCourts(e) {
			if v == r.equipe {
				bons[k]++
			}
		}
		t.Logf("  %s slot %2d %-16s equipe de reference %d ; champs %v", filepath.Base(dir),
			i, r.nom, r.equipe, s3oCourts(e))
	}
	return vus
}

// s3oEquipeInterne execute la lentille (b) : le champ partage-t-il le roster en deux moities
// de taille egale ? Critere ecrit d'avance : exactement deux valeurs distinctes, en parts
// egales. Aucune entree externe.
func s3oEquipeInterne(t *testing.T, dir string, es []*s3sEnr, equil []int) {
	t.Helper()
	noms := s3oNomsCourts()
	var l []string
	for k := range noms {
		dist := map[int]int{}
		for _, e := range es {
			dist[s3oCourts(e)[k]]++
		}
		if len(dist) == 2 {
			var c []int
			for _, n := range dist {
				c = append(c, n)
			}
			if c[0] == c[1] {
				equil[k]++
				l = append(l, noms[k]+" "+s3pDist(dist)[0]+"/"+s3pDist(dist)[1])
			}
		}
	}
	t.Logf("  %s partages equilibres : %v", filepath.Base(dir), l)
}

// TestSection3SlotChampsCourts recense la DISTRIBUTION des champs courts, sans verdict : c'est
// le releve qui dira lesquels sont constants, lesquels varient par joueur, lesquels par match.
func TestSection3SlotChampsCourts(t *testing.T) {
	noms := s3oNomsCourts()
	dist := make([]map[int]int, len(noms))
	for k := range dist {
		dist[k] = map[int]int{}
	}
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		for _, e := range s3sChaine(d) {
			for k, v := range s3oCourts(e) {
				dist[k][v]++
			}
		}
	}
	for k, n := range noms {
		t.Logf("  %-14s %d valeur(s) observee(s) : %v", n, len(dist[k]), s3pDist(dist[k]))
	}
}

// s3oNomPredit rend la position en BITS, RELATIVE au debut de l'enregistrement, des DEUX champs
// de texte : le champ de nom (`sub+0xc14`) et le bloc de queue (`sub+0x1400`).
func s3oNomPredit(e *s3sEnr) (nom, queue int) {
	nom = s3rEnteteBits + 64 + s3sPrefixeMasque + e.compteMasque + s3sLargeurN + e.n*8 +
		s3sLargeurM + e.m*32 + s3sBloc104
	return nom, e.total - 32 - s3sBloc44
}

// s3oTouche : une suite UTF-16LE imprimable trouvee par le balayage du 30/08.
type s3oTouche struct {
	octet int
	texte string
}

// s3oBalayageUTF16 rejoue la lecture du 30/08 : balayage du tampon a la recherche de suites
// UTF-16LE imprimables ALIGNEES SUR L'OCTET, dans la plage [deb, fin[ en octets.
func s3oBalayageUTF16(d []byte, deb, fin int) []s3oTouche {
	var out []s3oTouche
	for i := deb; i+2*s3oTexteMin <= fin; {
		var u []uint16
		j := i
		for j+1 < fin {
			v := uint16(d[j]) | uint16(d[j+1])<<8
			if v < 0x20 || v > 0x7e {
				break
			}
			u = append(u, v)
			j += 2
		}
		if len(u) >= s3oTexteMin {
			out = append(out, s3oTouche{octet: i, texte: string(utf16.Decode(u))})
			i = j
			continue
		}
		i++
	}
	return out
}

// TestSection3SlotTexte execute G-TXT : tout texte UTF-16 aligne sur l'octet du corps est-il
// explique par l'un des deux champs de texte de la grammaire ?
func TestSection3SlotTexte(t *testing.T) {
	nom, queue, hors := 0, 0, 0
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		es := s3sChaine(d)
		if len(es) == 0 {
			continue
		}
		deb, fin := es[0].debut/8, dernierNonNul(d)+1
		t.Logf("=== %s === balayage UTF-16LE aligne sur l'octet de 0x%06x a 0x%06x, "+
			"%d enregistrement(s)", filepath.Base(dir), deb, fin, len(es))
		for _, tc := range s3oBalayageUTF16(d, deb, fin) {
			e, rel := s3oSitue(es, tc.octet*8)
			marque, cat := s3oExplique(e, rel)
			switch cat {
			case "nom":
				nom++
			case "queue":
				queue++
			default:
				hors++
			}
			t.Logf("  0x%06x %-18q %s", tc.octet, tc.texte, marque)
		}
	}
	t.Logf("=== BILAN G-TXT === %d touche(s) sur le champ de nom, %d sur le bloc de queue, "+
		"%d hors grammaire", nom, queue, hors)
}

// s3oSitue rend l'enregistrement qui contient `bit` et la position RELATIVE a son debut.
func s3oSitue(es []*s3sEnr, bit int) (*s3sEnr, int) {
	for i := len(es) - 1; i >= 0; i-- {
		if bit >= es[i].debut {
			return es[i], bit - es[i].debut
		}
	}
	return nil, -1
}

// s3oExplique dit sur quel champ de texte tombe une touche.
//
// Le test est une APPARTENANCE A UNE PLAGE, pas une tolerance : le balayage du 30/08 lit des
// unites UTF-16 PETIT-BOUTISTES alignees sur l'octet, alors que les deux champs de texte de
// l'enregistrement sont ecrits a une position quelconque du flux (l'un en unites
// GROS-BOUTISTES). Un tel balayage n'accroche donc pas le DEBUT d'un champ : il demarre au
// premier octet ou la donnee decalee tombe dans l'ASCII imprimable, c'est-a-dire quelque part
// A L'INTERIEUR du champ. La seule question qui se pose est « dans quel champ ? », et une plage
// y repond sans qu'on ait rien a regler.
func s3oExplique(e *s3sEnr, rel int) (string, string) {
	if e == nil {
		return "hors de tout enregistrement", "hors"
	}
	pn, pq := s3oNomPredit(e)
	fn := pn + (len(utf16.Encode([]rune(e.gamertag)))+1)*16
	base := "enr. xuid " + strconv.FormatUint(e.xuid, 10) + " (" + e.gamertag + "), " +
		strconv.Itoa(rel) + " bits du debut"
	switch {
	case rel >= pn && rel < fn:
		return base + " -> DANS LE CHAMP DE NOM [" + strconv.Itoa(pn) + ", " +
			strconv.Itoa(fn) + "[", "nom"
	case rel >= pq && rel < pq+s3sBloc44:
		return base + " -> DANS LE BLOC DE QUEUE [" + strconv.Itoa(pq) + ", " +
			strconv.Itoa(pq+s3sBloc44) + "[", "queue"
	}
	return base + " -> HORS GRAMMAIRE (champ de nom [" + strconv.Itoa(pn) + ", " +
		strconv.Itoa(fn) + "[, bloc de queue [" + strconv.Itoa(pq) + ", " +
		strconv.Itoa(pq+s3sBloc44) + "[)", "hors"
}

// TestSection3SlotBlocQueue execute G-T44 : le bloc de queue de 44 octets porte-t-il le nom du
// joueur, et A QUEL OFFSET ? L'offset est cherche, pas suppose : le test balaie les 22 positions
// paires du bloc et retient celle ou la relecture rend EXACTEMENT le gamertag du meme
// enregistrement. Rien a ajuster, et la reponse est chiffree par build.
func TestSection3SlotBlocQueue(t *testing.T) {
	parBuild := map[string]map[int]int{}
	vus, trouves := 0, 0
	for _, dir := range chunk00Films(t, "CHUNK00_FILMS") {
		_, d := readChunk00(t, dir)
		build := s3wChaine(d, s3wBuildOff, 32)
		if parBuild[build] == nil {
			parBuild[build] = map[int]int{}
		}
		for i, e := range s3sChaine(d) {
			vus++
			off := s3oOffsetNom(e.bloc44, e.gamertag)
			if off >= 0 {
				trouves++
			}
			parBuild[build][off]++
			t.Logf("  %s (%s) slot %2d nom %-16q ; bloc de queue : nom a l'offset %3d, "+
				"debut du bloc %s", filepath.Base(dir), build, i, e.gamertag, off,
				s3oHex(e.bloc44, 16))
		}
	}
	for b, dist := range parBuild {
		t.Logf("=== BILAN G-T44 === %-12s offsets du nom dans le bloc de queue : %v",
			b, s3pDist(dist))
	}
	t.Logf("=== BILAN G-T44 === %d/%d blocs de queue portent le gamertag de leur "+
		"enregistrement (-1 = absent)", trouves, vus)
}

// s3oOffsetNom cherche, aux 22 positions paires du bloc, celle ou une suite d'unites UTF-16
// PETIT-BOUTISTES terminee par NUL rend exactement `nom`. Rend -1 si aucune.
func s3oOffsetNom(b []byte, nom string) int {
	if nom == "" {
		return -1
	}
	for off := 0; off+2 <= len(b); off += 2 {
		if s3sTexte16(b[off:], true) == nom {
			return off
		}
	}
	return -1
}

// s3oHex rend les n premiers octets d'une zone en hexadecimal, pour le releve.
func s3oHex(b []byte, n int) string {
	if n > len(b) {
		n = len(b)
	}
	const chiffres = "0123456789abcdef"
	out := make([]byte, 0, n*3)
	for i := 0; i < n; i++ {
		if i > 0 {
			out = append(out, ' ')
		}
		out = append(out, chiffres[b[i]>>4], chiffres[b[i]&0xf])
	}
	return string(out)
}
