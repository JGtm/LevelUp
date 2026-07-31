// tmp_i22scan — trouver les comptes de grenades (i22) par SCAN DIRECT, et mesurer le resultat
// contre la verite de position issue de la capture CE.
//
// LA DOCTRINE, et d'ou elle vient. Le chantier armes a bute sur le meme mur que nous — la marche
// sequentielle des records decroche tot dans le paquet — et l'a contourne au lieu de le reparer,
// sur une idee de l'utilisateur : « le fichier ne se lit pas en continu, on n'a pas a l'approcher
// de maniere sequentielle ». On balaie TOUTES les positions de bit et on ne retient que ce qui
// passe des tests simultanes. Plafond casse : 88,7 % -> 97,6 % sur l'arme du kill.
//
// POURQUOI CA MARCHE ICI, chiffre AVANT de coder. i22 fait 35 bits : R(3) compteur puis 4xR(8).
// La capture CE donne 35 bits fixes sur 100 % des lectures, donc le compteur vaut TOUJOURS 4. Les
// bornes du jeu ajoutent : au plus deux types portes, au plus deux unites de chacun. Une position
// quelconque passe donc si :
//
//	compteur == 4                        1/8
//	4 octets tous dans {0,1,2}           (3/256)^4  ~ 1,9e-9
//
// soit ~2,4e-10 par position. Sur 21 chunks d'environ 8e6 bits, l'esperance de faux positifs est
// de 0,04 — moins d'UN sur tout le film. Ce n'est pas une heuristique, c'est un filtre dont le
// taux d'accident est calcule.
//
// LE JUGE. cmd/tmp_comptruth localise, par la signature de 16 octets de la capture, l'offset
// EXACT de chacune des 249 lectures d'i22 du moteur. On mesure donc RAPPEL (combien de vraies
// lectures le scan retrouve) et PRECISION (combien de ses candidats sont a une position connue),
// au lieu de les estimer. Un scanner sans juge est un generateur d'illusions.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

// i22Bits est la largeur mesuree du composant (capture CE : 35 bits, 100 % des lectures).
const i22Bits = 3 + 4*8

// grenadeMaxTypes / grenadeMaxEach sont les bornes du JEU, pas du format : on porte au plus deux
// types de grenade, deux unites de chacun. Source : utilisateur, consignee dans
// .ai/V7.5/film_re/GRAMMAIRE_RECORD_FILM.md.
const (
	grenadeMaxTypes = 2
	grenadeMaxEach  = 2
)

// truthTol est la tolerance d'appariement entre un candidat (position en bits) et une position de
// verite (offset en OCTETS). Le pointeur d'octet du moteur peut devancer le curseur de bits de
// quelques octets — on mesure la distribution reelle avant de conclure, cette tolerance ne sert
// qu'a cadrer la mesure.
const truthTol = 128

type cand struct {
	Chunk, Bit int
	Vals       [4]uint64
}

func main() {
	dir := flag.String("film", "", "dossier des chunks du film")
	truth := flag.String("truth", "", "fichier de positions de reference (tmp_comptruth -out)")
	showN := flag.Int("show", 10, "candidats a afficher")
	calib := flag.Bool("calib", false, "calibrer le decalage pointeur d octet -> debut des donnees")
	cursorMode := flag.Bool("cursor", false, "tester la voie directe : position = curseur moteur + amorce")
	flag.IntVar(&backWin, "back", 96, "fenetre de calibration en arriere, en bits")
	flag.BoolVar(&looseBounds, "loose", false, "ne garder que la contrainte structurelle (compteur == 4), sans les bornes de jeu")
	flag.Parse()
	if *dir == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_i22scan -film <dir> [-truth positions.tsv]")
		os.Exit(2)
	}

	nc := filmdec.CountFilmChunks(*dir)
	var cands []cand
	scanned := 0
	for c := 1; c <= nc; c++ {
		chunk, err := filmdec.ReadFilmChunk(*dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			pay := p.Payload(chunk)
			// L'offset des candidats est rendu RELATIF AU CHUNK, pas au payload : c'est dans
			// cette unite que la verite de position est exprimee.
			base := p.Start
			lim := len(pay)*8 - i22Bits
			scanned += lim
			for b := 0; b <= lim; b++ {
				v, ok := parseI22(pay, b)
				if !ok {
					continue
				}
				cands = append(cands, cand{Chunk: c, Bit: base*8 + b, Vals: v})
			}
		}
	}

	fmt.Printf("SCAN DIRECT i22 — %s\n\n", *dir)
	fmt.Printf("  %d positions de bit examinees\n", scanned)
	fmt.Printf("  %d candidats retenus par les bornes du jeu\n", len(cands))
	exp := float64(scanned) * (1.0 / 8.0) * pow(3.0/256.0, 4)
	fmt.Printf("  esperance de faux positifs par hasard : %.3f\n\n", exp)

	if *cursorMode && *truth != "" {
		tp, err := loadTruth(*truth)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		byCursor(*dir, tp)
		return
	}
	if *calib && *truth != "" {
		tp, err := loadTruth(*truth)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		calibrate(*dir, tp)
		return
	}
	if *truth == "" {
		showCands(cands, *showN)
		return
	}
	tp, err := loadTruth(*truth)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	measure(cands, tp, *showN)
}

// parseI22 lit un i22 candidat a la position b et applique les bornes. Renvoie ok=false des
// qu'une borne est violee — aucune tolerance : c'est la severite du filtre qui fait sa valeur.
// looseBounds relache les bornes de JEU et ne garde que la contrainte STRUCTURELLE (compteur
// == 4), celle qu'etablit la mesure CE : 35 bits fixes = 3 + 4x8.
//
// POURQUOI CE COMMUTATEUR EXISTE. Les bornes « au plus deux types, deux unites de chacun »
// viennent du jeu STANDARD. Le film mesure ici est un Super Fiesta, ou le loadout est tire au
// hasard a chaque reapparition — rien ne garantit que ces bornes y tiennent. Les appliquer
// sans les avoir verifiees sur CE mode reviendrait a rejeter de vraies lectures. Le
// commutateur permet de mesurer l'ecart entre les deux hypotheses au lieu d'en supposer une.
var looseBounds bool

// backWin est la fenetre de calibration. Elle est un DRAPEAU parce que sa valeur decide du
// resultat : a 96 bits on mesure 57 %% de rappel, et rien ne dit si le reste est hors fenetre
// ou hors bornes. Il faut donc la faire varier et regarder ou le rappel sature.
var backWin = 96

func parseI22(pay []byte, b int) ([4]uint64, bool) {
	var v [4]uint64
	if filmdec.PeekBits(pay, b, 3) != 4 {
		return v, false // compteur != 4 : la capture CE dit qu'il vaut TOUJOURS 4
	}
	nz := 0
	for i := 0; i < 4; i++ {
		x := filmdec.PeekBits(pay, b+3+i*8, 8)
		if !looseBounds && x > grenadeMaxEach {
			return v, false
		}
		if x > 0 {
			nz++
		}
		v[i] = x
	}
	if looseBounds {
		return v, true
	}
	if nz > grenadeMaxTypes {
		return v, false
	}
	// PIEGE CORRIGE LE 2026-07-27 : ce filtre rejetait `nz == 0` (« aucun type porte :
	// structurellement possible, mais sans valeur ici »). C'ETAIT FAUX, et cette seule ligne
	// jetait 107 des 249 lectures reelles — 43 %.
	//
	// La relecture aux positions EXACTES (voie curseur) le montre sans appel : sur les 249
	// lectures du moteur, le compteur vaut 4 dans 100 % des cas, la valeur maximale observee
	// est 2, et les seules valeurs vues sont 0, 1 et 2. Les bornes du jeu sont donc PARFAITEMENT
	// respectees — et 107 lectures portent legitimement zero grenade (juste apres un lancer,
	// cas frequent). Un joueur sans grenade est un etat normal, pas un parse invalide.
	//
	// J'avais conclu de ce rejet que « 43 % des lectures violent le critere absolu du projet ».
	// C'etait mon filtre qui etait faux, pas le critere.
	return v, true
}

type tpos struct{ Chunk, ByteOff, Cursor int }

// byCursor teste la voie DIRECTE : le curseur capture est-il relatif au debut du payload du
// paquet ? Si oui, `paquet.Start*8 + amorce + curseur` donne la position EXACTE du composant,
// sans scan et sans calibration — et donc la VALEUR.
//
// C'est la seule voie qui donnerait 249/249 : le scan plafonne a 142 parce que les bornes de jeu
// ecartent 43 % des lectures reelles, et aucune fenetre ne rattrape ca.
//
// ON BALAIE L'AMORCE plutot que de la fixer : elle vaut 2 bits d'apres le desassemblage plus une
// mesure, mais c'est precisement le genre de constante qu'il faut re-verifier quand on change de
// repere. Si un decalage sort a un taux tres superieur aux autres, il est etabli ; s'ils se
// valent, la voie est refutee et il faut le dire.
func byCursor(dir string, truth []tpos) {
	nc := filmdec.CountFilmChunks(dir)
	type pk struct{ start, size int }
	packets := make([][]pk, nc+1)
	chunks := make([][]byte, nc+1)
	for c := 1; c <= nc; c++ {
		b, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		chunks[c] = b
		for _, p := range filmdec.WalkPackets(b) {
			if p.Type == filmdec.PacketTypeDelta {
				packets[c] = append(packets[c], pk{p.Start, p.Size})
			}
		}
	}
	best := map[int]int{}
	noPacket := 0
	for _, t := range truth {
		if t.Chunk >= len(chunks) || chunks[t.Chunk] == nil {
			continue
		}
		// paquet contenant le pointeur d'octet localise
		var found *pk
		for i := range packets[t.Chunk] {
			p := packets[t.Chunk][i]
			if t.ByteOff >= p.start && t.ByteOff < p.start+p.size {
				found = &p
				break
			}
		}
		if found == nil {
			noPacket++
			continue
		}
		pay := chunks[t.Chunk][found.start : found.start+found.size]
		for pre := 0; pre <= 8; pre++ {
			b := t.Cursor + pre
			if b < 0 || b+i22Bits > len(pay)*8 {
				continue
			}
			if _, ok := parseI22(pay, b); ok {
				best[pre]++
			}
		}
	}
	fmt.Printf("  VOIE DIRECTE — position = curseur moteur + amorce, dans le payload du paquet\n\n")
	fmt.Printf("    %d / %d positions dont le pointeur d'octet tombe hors de tout paquet delta\n\n",
		noPacket, len(truth))
	type kv struct{ pre, n int }
	var rows []kv
	for p, n := range best {
		rows = append(rows, kv{p, n})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
	fmt.Println("    amorce testee -> lectures ou i22 parse validement —")
	for _, r := range rows {
		fmt.Printf("      +%d bits : %3d / %d  (%.1f %%)\n",
			r.pre, r.n, len(truth), 100*float64(r.n)/float64(max(1, len(truth))))
	}
	if len(rows) == 0 {
		fmt.Println("      AUCUNE : le curseur n'est pas relatif au payload du paquet. Voie refutee.")
		return
	}

	// LA VALEUR, enfin. La position etant exacte, on lit i22 aux 249 positions SANS filtre et on
	// publie la distribution reelle. C'est la seule facon de trancher si la borne « au plus deux
	// types, deux unites » est un fait ou une croyance : jusqu'ici on ne pouvait que constater
	// qu'elle rejetait 43 % des lectures, sans savoir CE qu'elle rejetait.
	fmt.Println("\n    VALEURS REELLES lues aux 249 positions exactes (aucun filtre) —")
	cnt := map[uint64]int{}
	valHist := map[uint64]int{}
	nzHist := map[int]int{}
	maxVal := uint64(0)
	read := 0
	for _, t := range truth {
		if t.Chunk >= len(chunks) || chunks[t.Chunk] == nil {
			continue
		}
		var found *pk
		for i := range packets[t.Chunk] {
			p := packets[t.Chunk][i]
			if t.ByteOff >= p.start && t.ByteOff < p.start+p.size {
				found = &p
				break
			}
		}
		if found == nil {
			continue
		}
		pay := chunks[t.Chunk][found.start : found.start+found.size]
		if t.Cursor < 0 || t.Cursor+i22Bits > len(pay)*8 {
			continue
		}
		read++
		cnt[filmdec.PeekBits(pay, t.Cursor, 3)]++
		nz := 0
		for i := 0; i < 4; i++ {
			x := filmdec.PeekBits(pay, t.Cursor+3+i*8, 8)
			valHist[x]++
			if x > 0 {
				nz++
			}
			if x > maxVal {
				maxVal = x
			}
		}
		nzHist[nz]++
	}
	fmt.Printf("      %d lectures relues\n", read)
	fmt.Printf("      compteur R(3) : %s\n", histLine(cnt, read))
	fmt.Printf("      types non nuls par lecture : %s\n", histLineInt(nzHist, read))
	fmt.Printf("      valeur maximale observee : %d\n", maxVal)
	fmt.Printf("      valeurs les plus frequentes : %s\n", topVals(valHist, 10))
}

func histLine(m map[uint64]int, tot int) string {
	type kv struct {
		k uint64
		n int
	}
	var rows []kv
	for k, n := range m {
		rows = append(rows, kv{k, n})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].k < rows[j].k })
	s := ""
	for _, r := range rows {
		s += fmt.Sprintf("%d:%d (%.0f%%)  ", r.k, r.n, 100*float64(r.n)/float64(max(1, tot)))
	}
	return s
}

func histLineInt(m map[int]int, tot int) string {
	c := map[uint64]int{}
	for k, v := range m {
		c[uint64(k)] = v
	}
	return histLine(c, tot)
}

func topVals(m map[uint64]int, n int) string {
	type kv struct {
		k uint64
		n int
	}
	var rows []kv
	for k, c := range m {
		rows = append(rows, kv{k, c})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
	s := ""
	for i, r := range rows {
		if i >= n {
			s += fmt.Sprintf("(+%d autres)", len(rows)-n)
			break
		}
		s += fmt.Sprintf("%d×%d  ", r.k, r.n)
	}
	return s
}

func loadTruth(path string) ([]tpos, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("lecture de la reference : %w", err)
	}
	defer f.Close()
	var out []tpos
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 2 {
			continue
		}
		c, e1 := strconv.Atoi(f[0])
		o, e2 := strconv.Atoi(f[1])
		if e1 != nil || e2 != nil {
			continue
		}
		cur := 0
		if len(f) >= 5 {
			cur, _ = strconv.Atoi(f[4])
		}
		out = append(out, tpos{c, o, cur})
	}
	return out, sc.Err()
}

// measure confronte les candidats a la verite. On publie la DISTRIBUTION des ecarts avant de
// conclure : si les vraies lectures sont a un decalage constant, cela se voit, et c'est une
// information — pas un echec.
func measure(cands []cand, truth []tpos, showN int) {
	byChunk := map[int][]cand{}
	for _, c := range cands {
		byChunk[c.Chunk] = append(byChunk[c.Chunk], c)
	}
	for k := range byChunk {
		sort.Slice(byChunk[k], func(i, j int) bool { return byChunk[k][i].Bit < byChunk[k][j].Bit })
	}

	hitDelta := map[int]int{}
	found := 0
	for _, t := range truth {
		best, bestD := -1, 1<<30
		for _, c := range byChunk[t.Chunk] {
			d := c.Bit - t.ByteOff*8
			if d < 0 {
				d = -d
			}
			if d < bestD {
				bestD, best = d, c.Bit
			}
		}
		if best >= 0 && bestD <= truthTol {
			found++
			hitDelta[best-t.ByteOff*8]++
		}
	}
	fmt.Printf("  CONFRONTATION a %d positions de reference (tolerance %d bits)\n\n", len(truth), truthTol)
	fmt.Printf("    RAPPEL   : %d / %d = %.1f %%\n", found, len(truth), 100*float64(found)/float64(len(truth)))
	if len(cands) > 0 {
		fmt.Printf("    candidats : %d — soit %.1f par lecture reelle\n",
			len(cands), float64(len(cands))/float64(max(1, len(truth))))
	}
	if len(hitDelta) > 0 {
		type kv struct{ d, n int }
		var rows []kv
		for d, n := range hitDelta {
			rows = append(rows, kv{d, n})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
		fmt.Println("\n    ecart candidat - reference (bits), les plus frequents —")
		for i, r := range rows {
			if i >= 8 {
				break
			}
			fmt.Printf("      %+5d bits : %d fois\n", r.d, r.n)
		}
		fmt.Println("\n    UN ECART DOMINANT ET CONSTANT signifierait que le pointeur d'octet du moteur")
		fmt.Println("    devance le curseur d'une quantite fixe — donc que le scan tombe JUSTE et que")
		fmt.Println("    seule la conversion de repere reste a poser.")
	}
	showCands(cands, showN)
}

// calibrate cherche le decalage entre le POINTEUR D'OCTET du moteur (ce que la signature
// localise) et le DEBUT DES DONNEES du composant (ce qu'il faut pour lire la valeur).
//
// POURQUOI CE DECALAGE EXISTE. Le bitreader consomme par mots : quand il rend la main, son
// pointeur d'octet a deja avance au-dela des bits qu'il vient de servir. La signature capture
// donc un point situe APRES le debut du champ — d'ou les ecarts negatifs mesures (-26/-28,
// -58/-61) entre les candidats du scan et les positions de reference.
//
// LA METHODE. Pour chaque position de reference, on balaie une fenetre en arriere et on note
// TOUS les decalages ou un i22 valide se presente. Si un decalage domine nettement, c'est la
// conversion cherchee, et elle rend les positions exactes — donc les VALEURS.
//
// CE QUI FERAIT ECHOUER CETTE MESURE, et qu'il faut regarder : si plusieurs decalages sortent
// a egalite, la fenetre contient plusieurs parses valides et le filtre ne suffit pas a
// trancher. Ce serait un resultat, pas un echec : il dirait que les bornes du jeu ne sont pas
// assez selectives et qu'il faut un second test.
func calibrate(dir string, truth []tpos) {
	back := backWin // fenetre de recherche en arriere, en bits (drapeau : mesurer, pas supposer)
	nc := filmdec.CountFilmChunks(dir)
	chunks := make([][]byte, nc+1)
	for c := 1; c <= nc; c++ {
		if b, err := filmdec.ReadFilmChunk(dir, c); err == nil {
			chunks[c] = b
		}
	}
	hist := map[int]int{}
	withAny, withOne := 0, 0
	for _, t := range truth {
		if t.Chunk >= len(chunks) || chunks[t.Chunk] == nil {
			continue
		}
		ch := chunks[t.Chunk]
		var ok []int
		for d := -back; d <= 8; d++ {
			b := t.ByteOff*8 + d
			if b < 0 || (b+i22Bits)/8 >= len(ch) {
				continue
			}
			if _, good := parseI22(ch, b); good {
				ok = append(ok, d)
			}
		}
		if len(ok) > 0 {
			withAny++
			for _, d := range ok {
				hist[d]++
			}
		}
		if len(ok) == 1 {
			withOne++
		}
	}
	fmt.Printf("  CALIBRATION du decalage (fenetre %d bits en arriere)\n\n", back)
	fmt.Printf("    %d / %d positions ont au moins un i22 valide dans la fenetre\n", withAny, len(truth))
	fmt.Printf("    %d / %d n'en ont EXACTEMENT UN (non ambigu)\n\n", withOne, len(truth))
	type kv struct{ d, n int }
	var rows []kv
	for d, n := range hist {
		rows = append(rows, kv{d, n})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
	fmt.Println("    decalages les plus frequents —")
	for i, r := range rows {
		if i >= 12 {
			break
		}
		fmt.Printf("      %+4d bits : %4d positions (%.1f %%)\n",
			r.d, r.n, 100*float64(r.n)/float64(max(1, len(truth))))
	}
}

func showCands(cands []cand, n int) {
	if len(cands) == 0 {
		return
	}
	fmt.Println("\n  echantillon de candidats (chunk, bit, comptes par type) —")
	for i, c := range cands {
		if i >= n {
			fmt.Printf("    … %d autres\n", len(cands)-n)
			break
		}
		fmt.Printf("    %02d  %10d   [%d %d %d %d]\n", c.Chunk, c.Bit, c.Vals[0], c.Vals[1], c.Vals[2], c.Vals[3])
	}
}

func pow(x float64, n int) float64 {
	r := 1.0
	for i := 0; i < n; i++ {
		r *= x
	}
	return r
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
