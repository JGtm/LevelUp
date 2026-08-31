package replay

// visee_onde_research_test.go — LOT C : LA CORRELATION D'ONDE CARREE (moteur de mesure).
//
// POURQUOI CET INSTRUMENT. Tous les canaux « evenementiels » du chantier ont ete refutes :
// l'evenement dedie type 126 `unit_zoom` n'apparait JAMAIS dans les films (0 sur 41 M de paquets,
// phase 3) ; l'identification « type 114 = lunette » est tombee au lot A (controle par
// translation : 10/12 de couverture, mais 1,04 % des decalages font aussi bien) ; aucun composant
// du registre ECS ne porte la vue ; aucun bit de la tete du record de degat fatal ne separe
// zoome de non zoome. Or Theater PREMIERE PERSONNE affiche reellement le zoom, et la
// retro-ingenierie (phase 3) etablit que l'etat de zoom d'une unite n'a que DEUX sources
// d'ecriture pilotees par des donnees : l'evenement 126 (exclu) et L'OCTET 6 DE LA COMMANDE
// JOUEUR (`FUN_1406db688` : `unite+0x462 = commande[6]`, commande appliquee depuis la structure
// joueur +0x68 par `FUN_1404d5384`). Par elimination, les commandes sont rejouees depuis le film,
// et le lot de commandes d'un tick vit dans le FLUX D'ETAT du paquet delta — pas dans un
// evenement.
//
// CE QUE CHANGE L'ANGLE. Chercher un EVENEMENT, c'est disposer de 12 transitions : un oracle
// pauvre, ou 1,04 % du hasard suffit a imiter le signal. Chercher un ETAT a position fixe, c'est
// disposer d'UN ECHANTILLON PAR PAQUET — de l'ordre du millier sur la meme plage. C'est le
// meme relevé terrain, exploite avec deux ordres de grandeur d'echantillons en plus.
//
// L'ONDE. Relevé Theater manuscrit de l'utilisateur sur le film 00162144 (decalage feed->film
// +1 171 858 ms, etabli phase 6 par 91 fins de vie appariees, fige ici comme DEPENDANCE et non
// comme mesure) : Nilton410 est a la lunette sur les intervalles feed (secondes) {41 -> 46,3},
// {49 -> 52}, {61 -> 61,8}, {68 -> 68,8}, {71 -> 73}, {85 -> 86} et nulle part ailleurs dans
// [35 ; 95] ; Madina97294 sur {45 -> 46,3}. Ces episodes sont lus dans `chronoEpisodes` /
// `chronoEpisodeMadina` (source unique, partagee avec les instruments des phases 6 et A).
//
// LES BANDES DE GARDE, ET CE QU'ELLES COUTENT. Le relevé est precis a +/- 1 s. Toute la zone a
// +/- 1,2 s d'une transition est donc EXCLUE du score : on ne peut pas y dire si le joueur est
// zoome. Consequence assumee, chiffree ici pour qu'elle ne soit pas decouverte apres coup : les
// episodes de moins de 2,4 s disparaissent entierement, et l'episode {71 -> 73} aussi. Il ne
// reste, en classe « zoome », que [42,2 ; 45,1] et [50,2 ; 50,8], soit 3,5 s. C'est la raison
// d'etre de la variante a garde courte (0,5 s), declaree ci-dessous AVANT la mesure et publiee
// quel qu'en soit le resultat — jamais choisie apres coup.
//
// LA MESURE. Pour chaque paquet delta de la fenetre et chaque position de bit fixe b, la serie
// bit(b, t) est confrontee a l'onde s(t). Score = EXACTITUDE EQUILIBREE, moyenne du taux de vrais
// 1 et du taux de vrais 0 : une position constante vaut 0,5, l'aleatoire aussi, quelle que soit
// la proportion des classes. Les deux polarites sont comptees (un bit peut valoir 1 hors lunette).
// Domaine balaye : les 512 premiers bits du payload (le mandat), les 512 derniers, et une passe
// elargie a 1024 bits. Les positions sont ABSOLUES : l'hypothese est qu'un bloc de commandes de
// tick occupe un emplacement fixe, ce qui est plausible tant que le roster ne bouge pas — et il
// ne bouge pas sur les 60 s de la fenetre.
//
// SEUILS ECRITS AVANT TOUTE MESURE (regle absolue, METHODE_RETRO_INGENIERIE_FILM.md) :
//
//	S1. CANDIDAT      : exactitude equilibree >= 0,95 avec >= 200 echantillons de CHAQUE classe.
//	S2. A SUIVRE      : exactitude equilibree >= 0,85 (meme exigence d'echantillons).
//	S3. SOUS-DIMENSIONNE : si une classe compte moins de 200 echantillons, la variante est
//	    declaree telle et son resultat n'est PAS publiable comme candidat — le verdict repose
//	    alors exclusivement sur S4.
//	S4. CONTROLE PAR TRANSLATION (le juge, cf. O4 du lot A) : l'onde ENTIERE est translatee de
//	    delta, fenetre d'analyse comprise — ce qui preserve exactement la duree des creneaux, le
//	    nombre de transitions, les bandes de garde et la structure temporelle reelle des paquets.
//	    Deux parts sont publiees :
//	      - p(max) : part des decalages ou le MEILLEUR score, toutes positions et polarites
//	        confondues, atteint le score observe. C'est le controle severe : il corrige de lui-meme
//	        le balayage de ~500 positions x 2 polarites ;
//	      - p(pos) : part des decalages ou la POSITION candidate seule atteint son score observe
//	        (le controle du lot A, plus permissif, publie pour comparaison).
//	    VERDICT POSITIF EXIGE p(max) < 1 %. Aucune conclusion positive sans ce controle.
//
// SOUS GARDE (ONDE_FILM, qui doit pointer 00162144 — la chronologie est celle de CE film).
// Lecture de paquets pure : ni Scan*, ni LockProcessDecode, aucun etat global touche.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 ONDE_FILM=<repo>/data/cache/film_chunks/00162144 \
//	  go test ./internal/analysis/replay/ -run TestViseeOndeCarree -v -timeout 60m

import (
	"math/bits"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	ondeFilmEnv = "ONDE_FILM"
	// ondeOffsetMS : decalage feed -> film du film 00162144. Dependance figee (phase 6).
	ondeOffsetMS = sig114OffsetMS
	// Fenetre d'analyse, en secondes d'horloge du feed : [35 ; 95], la plage que le relevé
	// couvre de bout en bout (« pas a la lunette ailleurs dans cet intervalle »).
	ondeFeneDebutMS = 35_000 + ondeOffsetMS
	ondeFeneFinMS   = 95_000 + ondeOffsetMS
	ondeGardeMS     = 1200 // demi-largeur de la bande de garde autour d'une transition
	ondeGardeBrefMS = 500  // variante declaree d'avance (episodes courts)
	// ondeOctetMin : largeur de la tranche echantillonnee, en octets (512 bits, le domaine du
	// mandat). ondeOctetLarge : la meme mesure sur 1024 bits, qui n'accepte que les paquets d'au
	// moins 128 octets — le prix de l'elargissement est publie en paquets ecartes.
	ondeOctetMin    = 64
	ondeOctetLarge  = 128
	ondeBitMin      = 7 // les bits 0..6 portent le type de l'evenement : sans interet ici
	ondeEchMin      = 200
	ondeSeuilCand   = 0.95
	ondeSeuilSuivre = 0.85
	// Controle par translation : amplitude, pas, et zone exclue autour de la vraie position.
	ondeCtrlAmplMS  = 400_000
	ondeCtrlPasMS   = 250
	ondeCtrlGardeMS = 10_000
)

// ondeCarree est l'onde de reference : les episodes zoomes, leurs transitions et la garde.
type ondeCarree struct {
	nom   string
	eps   [][2]int64 // bornes des episodes, en ms d'horloge du film
	trans []int64    // instants de transition (2 par episode)
	garde int64
}

// ondeConstruit convertit des episodes exprimes en secondes de feed en onde de film.
func ondeConstruit(nom string, eps [][2]float64, garde int64) ondeCarree {
	bornes := chronoVersFilm(eps, ondeOffsetMS)
	return ondeCarree{nom: nom, eps: bornes, trans: chronoTransitions(bornes), garde: garde}
}

// classe rend 1 (zoome), 0 (pas zoome) ou -1 (dans une bande de garde : exclu du score).
func (o ondeCarree) classe(t int64) int {
	for _, tr := range o.trans {
		if t >= tr-o.garde && t <= tr+o.garde {
			return -1
		}
	}
	for _, e := range o.eps {
		if t >= e[0] && t <= e[1] {
			return 1
		}
	}
	return 0
}

// dureeClasse1 rend la duree totale, en ms, effectivement classee « zoome » (hors gardes).
func (o ondeCarree) dureeClasse1() int64 {
	var total int64
	for t := int64(ondeFeneDebutMS); t <= int64(ondeFeneFinMS); t += 10 {
		if o.classe(t) == 1 {
			total += 10
		}
	}
	return total
}

// ondePaquet est un paquet delta reduit a ce que la mesure consomme : son instant, la tete de
// son evenement, et deux tranches egales (debut et fin du payload).
type ondePaquet struct {
	tMS   int64
	tete  int
	debut []byte
	fin   []byte
}

// ondeCollecte rassemble les paquets delta du film dont l'horodatage tombe dans [t0 ; t1] et
// dont le payload porte au moins octets octets. Rend aussi le nombre de paquets ecartes
// pour cause de payload trop court (publie : un filtre silencieux est un biais cache).
func ondeCollecte(dir string, t0, t1 int64, octets int) ([]ondePaquet, int) {
	var out []ondePaquet
	var courts int
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
			tMS := int64(p.TimestampUS / 1000)
			if tMS < t0 || tMS > t1 {
				continue
			}
			pay := p.Payload(chunk)
			if len(pay) < octets {
				courts++
				continue
			}
			debut := make([]byte, octets)
			copy(debut, pay[:octets])
			fin := make([]byte, octets)
			copy(fin, pay[len(pay)-octets:])
			out = append(out, ondePaquet{tMS: tMS, tete: int(pay[0] >> 1), debut: debut, fin: fin})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].tMS < out[j].tMS })
	return out, courts
}

// ondeFiltre rend le sous-ensemble des paquets dont la tete figure dans tetes (nil = toutes).
func ondeFiltre(pk []ondePaquet, tetes map[int]bool) []ondePaquet {
	if tetes == nil {
		return pk
	}
	var out []ondePaquet
	for _, p := range pk {
		if tetes[p.tete] {
			out = append(out, p)
		}
	}
	return out
}

// ondeCol transpose un lot de paquets en COLONNES DE BITS : col[b] est un vecteur de bits
// indexe par paquet, portant la valeur du bit b de chaque paquet. Cette transposition est ce
// qui rend le controle par translation abordable : un score se calcule alors par ET binaire et
// popcount sur des mots de 64 bits, et non paquet par paquet.
type ondeCol struct {
	temps []int64
	col   [][]uint64
	mots  int
	nbits int
}

// ondeBatColonnes construit les colonnes ; queue = true echantillonne la FIN du payload.
func ondeBatColonnes(pk []ondePaquet, queue bool) *ondeCol {
	nbits := len(pk[0].debut) * 8
	c := &ondeCol{temps: make([]int64, len(pk)), mots: (len(pk) + 63) / 64, nbits: nbits}
	c.col = make([][]uint64, nbits)
	for b := range c.col {
		c.col[b] = make([]uint64, c.mots)
	}
	for i, p := range pk {
		c.temps[i] = p.tMS
		src := p.debut
		if queue {
			src = p.fin
		}
		mot, bit := i/64, uint(i%64)
		for b := 0; b < nbits; b++ {
			if src[b>>3]>>(7-uint(b&7))&1 == 1 {
				c.col[b][mot] |= 1 << bit
			}
		}
	}
	return c
}

// ondeMasque porte les deux classes d'un decalage donne, sous forme de vecteurs de bits.
type ondeMasque struct {
	un, zero []uint64
	n1, n0   int
	nGarde   int // paquets DANS la fenetre mais tombes dans une bande de garde
	nHors    int // paquets hors de la fenetre d'analyse : ni classes, ni exclus
	w0, w1   int // bornes des mots non nuls : le score ne parcourt qu'eux
}

// marque construit les masques de classe pour l'onde translatee de delta. La fenetre d'analyse
// se translate AVEC l'onde : le controle compare ainsi des situations rigoureusement de meme
// forme (meme duree, memes creneaux, memes gardes), seul le contenu du film changeant.
func (c *ondeCol) marque(o ondeCarree, delta int64) ondeMasque {
	m := ondeMasque{un: make([]uint64, c.mots), zero: make([]uint64, c.mots), w0: c.mots, w1: 0}
	n := len(c.temps)
	i0 := sort.Search(n, func(i int) bool { return c.temps[i] >= ondeFeneDebutMS+delta })
	i1 := sort.Search(n, func(i int) bool { return c.temps[i] > ondeFeneFinMS+delta })
	m.nHors = n - (i1 - i0)
	for i := i0; i < i1; i++ {
		mot, bit := i/64, uint(i%64)
		switch o.classe(c.temps[i] - delta) {
		case 1:
			m.un[mot] |= 1 << bit
			m.n1++
		case 0:
			m.zero[mot] |= 1 << bit
			m.n0++
		default:
			m.nGarde++
			continue
		}
		if mot < m.w0 {
			m.w0 = mot
		}
		if mot > m.w1 {
			m.w1 = mot
		}
	}
	return m
}

// ondeScore decrit la performance d'une position de bit.
type ondeScore struct {
	pos      int
	polarite int // +1 : bit a 1 pendant la lunette ; -1 : l'inverse
	score    float64
	tp, fp   int
}

// evalue rend l'exactitude equilibree BRUTE (polarite +1) de la position b.
func (c *ondeCol) evalue(b int, m ondeMasque) (float64, int, int) {
	var tp, fp int
	colonne := c.col[b]
	for w := m.w0; w <= m.w1; w++ {
		tp += bits.OnesCount64(colonne[w] & m.un[w])
		fp += bits.OnesCount64(colonne[w] & m.zero[w])
	}
	if m.n1 == 0 || m.n0 == 0 {
		return 0.5, tp, fp
	}
	ba := 0.5 * (float64(tp)/float64(m.n1) + float64(m.n0-fp)/float64(m.n0))
	return ba, tp, fp
}

// meilleur rend le score maximal sur toutes les positions et les deux polarites.
func (c *ondeCol) meilleur(m ondeMasque) ondeScore {
	best := ondeScore{pos: -1}
	for b := ondeBitMin; b < c.nbits; b++ {
		ba, tp, fp := c.evalue(b, m)
		s, pol := ba, 1
		if 1-ba > s {
			s, pol = 1-ba, -1
		}
		if s > best.score {
			best = ondeScore{pos: b, polarite: pol, score: s, tp: tp, fp: fp}
		}
	}
	return best
}

// classement rend toutes les positions triees par score decroissant.
func (c *ondeCol) classement(m ondeMasque) []ondeScore {
	out := make([]ondeScore, 0, c.nbits-ondeBitMin)
	for b := ondeBitMin; b < c.nbits; b++ {
		ba, tp, fp := c.evalue(b, m)
		s, pol := ba, 1
		if 1-ba > s {
			s, pol = 1-ba, -1
		}
		out = append(out, ondeScore{pos: b, polarite: pol, score: s, tp: tp, fp: fp})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].score > out[j].score })
	return out
}

// ondeScorePos rend le score (polarite libre) d'UNE position pour un masque donne.
func (c *ondeCol) ondeScorePos(b int, m ondeMasque) float64 {
	ba, _, _ := c.evalue(b, m)
	if 1-ba > ba {
		return 1 - ba
	}
	return ba
}

// ondeTetes construit un ensemble de tetes d'evenement pour ondeFiltre.
func ondeTetes(vals ...int) map[int]bool {
	m := map[int]bool{}
	for _, v := range vals {
		m[v] = true
	}
	return m
}

// ondeDebitParTete rend le nombre de paquets par tete d'evenement (mesure descriptive).
func ondeDebitParTete(pk []ondePaquet) map[int]int {
	out := map[int]int{}
	for _, p := range pk {
		out[p.tete]++
	}
	return out
}
