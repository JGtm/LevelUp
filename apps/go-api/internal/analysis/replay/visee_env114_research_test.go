package replay

// visee_env114_research_test.go — LOT A1 : LES FRONTIERES DE CHAMPS DE L'ENVELOPPE DU TYPE 114,
// INFEREES PAR LA VARIANCE.
//
// CE QUI EST ACQUIS (phases 6 / 6bis, cf. .ai/thought_log.md du 2026-08-30) : la mise a la
// lunette est l'evenement de paquet delta de type 114 (« biped_board_vehicle » — la lunette est
// un siege de l'arme) ; 11 des 12 transitions relevees a la main par l'utilisateur sur le film
// 00162144 tombent a moins d'une seconde d'un paquet 114. La grammaire GENERALE d'un paquet
// d'evenement (dispatcher FUN_14080a9d4) est :
//
//	R(7) type
//	{ R(1) porte ; si 1 : var-int FUN_1406d3140 = [sonde R(1)] + R(W) + R(2) }  x3 references
//	payload du type (vtable +0x68)
//	R(1) queue [+ R(32)]
//
// Les largeurs W dependent du DOMAINE de chaque reference (case +0x58 du descripteur : pour le
// type 114, ref0 -> domaine 2, ref1 -> 3, ref2 -> 7) et ne sont PAS connues. Consequence de
// methode, payee cher : **aucun offset absolu apres le bit 7 n'est acquis** — les champs sont a
// largeur variable. Cet instrument ne suppose donc AUCUNE frontiere : il les fait ressortir de
// la seule variance observee, et ne conclut qu'aux seuils ecrits ci-dessous.
//
// TROIS MESURES :
//
//	A. TAILLES — distribution des longueurs de payload des paquets 114 (une longueur unique
//	   dirait un evenement a portes figees ; plusieurs, des portes qui bougent).
//	B. BIT A BIT — pour chaque position, la fraction de paquets a 1 et l'entropie. Verdict
//	   CONSTANT (H = 0) / VARIABLE.
//	C. FENETRE GLISSANTE DE CARDINALITE — pour chaque position b, le nombre de valeurs
//	   DISTINCTES de la tranche [b ; b+8). C'est la lentille qui separe un identifiant (peu de
//	   valeurs : il n'y a que 8 joueurs et une poignee d'entites) d'un compteur ou d'un hachage
//	   (presque autant de valeurs que de paquets).
//
// SEUILS ECRITS AVANT LA MESURE (fenetre de 8 bits, N paquets du film) :
//   - zone CONSTANTE      : cardinalite de la fenetre == 1 ;
//   - zone D'IDENTIFIANTS : cardinalite <= 24 (8 joueurs x 3 entites plausibles) ;
//   - zone DE HAUTE VARIANCE : cardinalite >= min(64, N/2) ;
//   - entre les deux : INDECIS, publie comme tel.
//
// STABILITE INTER-FILMS : l'ossature (positions constantes et leurs valeurs) est declaree
// STABLE si les memes positions sortent constantes, avec les memes valeurs, sur les trois
// films. Tout ecart est publie, pas lisse.
//
// SOUS GARDE (ENV114_FILM). Balayage de paquets purs : ni Scan* ni LockProcessDecode.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 ENV114_FILM=<repo>/data/cache/film_chunks/00162144 \
//	  ENV114_AUTRES=<repo>/data/cache/film_chunks/00ba2e1c,<repo>/data/cache/film_chunks/03ccbe42 \
//	  go test ./internal/analysis/replay/ -run TestViseeEnv114Frontieres -v -timeout 30m

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	env114FilmEnv   = "ENV114_FILM"
	env114AutresEnv = "ENV114_AUTRES"
	env114Type      = 114
	env114Fenetre   = 8  // largeur de la fenetre glissante de cardinalite
	env114SeuilIdnt = 24 // cardinalite <= : zone d'identifiants
	env114SeuilHaut = 64 // cardinalite >= : zone de haute variance
	env114MaxBits   = 112
)

// env114Paquet est un evenement 114 brut, garde tel quel (aucune decoupe supposee).
type env114Paquet struct {
	tMS   int64
	pay   []byte
	nBits int
}

// env114Collecte rassemble tous les paquets delta dont l'evenement de tete est un type 114.
func env114Collecte(dir string) []env114Paquet { return env114CollecteType(dir, env114Type) }

// env114CollecteType rassemble les paquets delta dont l'octet de tete porte le type demande.
func env114CollecteType(dir string, typ int) []env114Paquet {
	var out []env114Paquet
	n := filmdec.CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 2 {
				continue
			}
			pay := p.Payload(chunk)
			if int(pay[0]>>1) != typ {
				continue
			}
			cp := make([]byte, len(pay))
			copy(cp, pay)
			out = append(out, env114Paquet{
				tMS:   int64(p.TimestampUS / 1000),
				pay:   cp,
				nBits: len(cp) * 8,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].tMS < out[j].tMS })
	return out
}

// env114BitsCommuns rend le nombre de bits presents dans TOUS les paquets (borne d'analyse).
func env114BitsCommuns(pk []env114Paquet) int {
	min := env114MaxBits
	for _, p := range pk {
		if p.nBits < min {
			min = p.nBits
		}
	}
	return min
}

// env114Tailles — mesure A : distribution des longueurs de payload.
func env114Tailles(t *testing.T, nom string, pk []env114Paquet) {
	t.Helper()
	comptes := map[int]int{}
	for _, p := range pk {
		comptes[len(p.pay)]++
	}
	var tailles []int
	for o := range comptes {
		tailles = append(tailles, o)
	}
	sort.Ints(tailles)
	var parts []string
	for _, o := range tailles {
		parts = append(parts, fmt.Sprintf("%do:%d", o, comptes[o]))
	}
	t.Logf("A. TAILLES [%s] — %d paquets 114 ; longueurs de payload : %s", nom, len(pk),
		strings.Join(parts, " "))
}

// env114Entropies rend, par position de bit, la fraction de paquets a 1 (borne = bits communs).
func env114Entropies(pk []env114Paquet, nb int) []float64 {
	frac := make([]float64, nb)
	for b := 0; b < nb; b++ {
		var uns int
		for _, p := range pk {
			uns += int(filmdec.ReadBitsAtForDiag(p.pay, b, 1))
		}
		frac[b] = float64(uns) / float64(len(pk))
	}
	return frac
}

func env114Shannon(p float64) float64 {
	if p <= 0 || p >= 1 {
		return 0
	}
	return -(p*math.Log2(p) + (1-p)*math.Log2(1-p))
}

// env114Cardinalites rend, par position b, le nombre de valeurs distinctes de [b ; b+w).
func env114Cardinalites(pk []env114Paquet, nb, w int) []int {
	card := make([]int, 0, nb)
	for b := 0; b+w <= nb; b++ {
		vus := map[uint32]bool{}
		for _, p := range pk {
			vus[filmdec.ReadBitsAtForDiag(p.pay, b, w)] = true
		}
		card = append(card, len(vus))
	}
	return card
}

// env114Classe rend le verdict d'une position, aux seuils declares en tete de fichier.
func env114Classe(card, n int) string {
	seuilHaut := env114SeuilHaut
	if n/2 < seuilHaut {
		seuilHaut = n / 2
	}
	switch {
	case card == 1:
		return "CONSTANTE"
	case card <= env114SeuilIdnt:
		return "IDENTIFIANT"
	case card >= seuilHaut:
		return "HAUTE-VARIANCE"
	default:
		return "indecis"
	}
}

// env114TableBits — mesures B et C reunies : une ligne par position de bit.
func env114TableBits(t *testing.T, nom string, pk []env114Paquet, nb int) []int {
	t.Helper()
	frac := env114Entropies(pk, nb)
	card := env114Cardinalites(pk, nb, env114Fenetre)
	t.Logf("B+C. PROFIL [%s] — %d paquets, %d bits communs ; par position : p(1), entropie,"+
		" cardinalite de [b ; b+%d) et verdict", nom, len(pk), nb, env114Fenetre)
	for b := 0; b < nb; b++ {
		verdict, c := "-", -1
		if b < len(card) {
			c = card[b]
			verdict = env114Classe(c, len(pk))
		}
		cst := ""
		if frac[b] == 0 {
			cst = "  <- constant 0"
		} else if frac[b] == 1 {
			cst = "  <- constant 1"
		}
		t.Logf("    bit %3d : p1=%.3f H=%.2f card8=%3d %-14s%s", b, frac[b],
			env114Shannon(frac[b]), c, verdict, cst)
	}
	return card
}

// env114Ossature rend les positions constantes et leur valeur (signature inter-films).
func env114Ossature(pk []env114Paquet, nb int) map[int]int {
	out := map[int]int{}
	frac := env114Entropies(pk, nb)
	for b := 0; b < nb; b++ {
		if frac[b] == 0 {
			out[b] = 0
		} else if frac[b] == 1 {
			out[b] = 1
		}
	}
	return out
}

// env114Cumul — cardinalite de la tranche [depuis ; b] quand b croit : les paliers de cette
// courbe sont les champs, ses marches les frontieres. La signature exploitable en A2 s'arrete
// la ou la courbe rejoint N (chaque paquet devient unique : compteur ou horodatage).
func env114Cumul(t *testing.T, nom string, pk []env114Paquet, depuis, nb int) {
	t.Helper()
	prec := 0
	var marches []string
	t.Logf("D. CARDINALITE CUMULEE [%s] depuis le bit %d (N=%d) :", nom, depuis, len(pk))
	for b := depuis; b < nb; b++ {
		vus := map[string]bool{}
		for _, p := range pk {
			vus[env114Cle(p, depuis, b-depuis+1)] = true
		}
		c := len(vus)
		if c != prec {
			marches = append(marches, fmt.Sprintf("bit %d -> %d", b, c))
			prec = c
		}
	}
	t.Logf("    marches (position : cardinalite atteinte) : %s", strings.Join(marches, " · "))
}

// env114Cle rend la tranche [depuis ; depuis+long) d'un paquet sous forme de chaine de bits
// (chaine et non entier : la tranche depasse 32 bits des qu'on cumule).
func env114Cle(p env114Paquet, depuis, long int) string {
	var sb strings.Builder
	for i := 0; i < long; i++ {
		b := depuis + i
		if b >= p.nBits {
			sb.WriteByte('_')
			continue
		}
		sb.WriteByte('0' + byte(filmdec.ReadBitsAtForDiag(p.pay, b, 1)))
	}
	return sb.String()
}

// TestViseeEnv114Frontieres execute A1 sur le film principal puis confronte l'ossature aux
// films de confirmation.
func TestViseeEnv114Frontieres(t *testing.T) {
	dir := os.Getenv(env114FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", env114FilmEnv)
	}
	pk := env114Collecte(dir)
	if len(pk) == 0 {
		t.Fatalf("aucun paquet 114 dans %s", dir)
	}
	nom := env114Nom(dir)
	env114Tailles(t, nom, pk)
	nb := env114BitsCommuns(pk)
	if nb > env114MaxBits {
		nb = env114MaxBits
	}
	env114TableBits(t, nom, pk, nb)
	env114Cumul(t, nom, pk, 24, nb)
	env114Cumul(t, nom, pk, 8, nb)

	ossature := env114Ossature(pk, nb)
	t.Logf("E. OSSATURE [%s] — %d positions constantes sur %d : %s", nom, len(ossature), nb,
		env114Motif(ossature, nb))

	for _, autre := range strings.Split(os.Getenv(env114AutresEnv), ",") {
		autre = strings.TrimSpace(autre)
		if autre == "" {
			continue
		}
		env114Confronte(t, autre, ossature, nb)
	}
}

// env114Confronte mesure la stabilite de l'ossature sur un autre film.
func env114Confronte(t *testing.T, dir string, ossature map[int]int, nbRef int) {
	t.Helper()
	pk := env114Collecte(dir)
	nom := env114Nom(dir)
	if len(pk) == 0 {
		t.Logf("F. [%s] — aucun paquet 114 (film sans zoom ou chunks absents)", nom)
		return
	}
	nb := env114BitsCommuns(pk)
	if nb > nbRef {
		nb = nbRef
	}
	env114Tailles(t, nom, pk)
	frac := env114Entropies(pk, nb)
	var tenus, casses, hors []string
	for b := 0; b < nb; b++ {
		v, ok := ossature[b]
		if !ok {
			continue
		}
		switch {
		case frac[b] == float64(v):
			tenus = append(tenus, fmt.Sprint(b))
		case frac[b] == 0 || frac[b] == 1:
			casses = append(casses, fmt.Sprintf("%d(ref %d, ici %.0f)", b, v, frac[b]))
		default:
			hors = append(hors, fmt.Sprintf("%d(ref %d, ici p1=%.2f)", b, v, frac[b]))
		}
	}
	t.Logf("F. STABILITE [%s] — %d paquets ; ossature tenue sur %d/%d positions comparables",
		nom, len(pk), len(tenus), len(tenus)+len(casses)+len(hors))
	if len(casses) > 0 {
		t.Logf("    constantes de VALEUR DIFFERENTE : %s", strings.Join(casses, " "))
	}
	if len(hors) > 0 {
		t.Logf("    constantes du film de reference qui VARIENT ici : %s", strings.Join(hors, " "))
	}
	card := env114Cardinalites(pk, nb, env114Fenetre)
	var zones []string
	for b := 0; b < len(card); b++ {
		zones = append(zones, fmt.Sprintf("%d:%s", b, env114ZoneCourte(env114Classe(card[b], len(pk)))))
	}
	t.Logf("    profil de zones : %s", strings.Join(zones, " "))
}

func env114ZoneCourte(v string) string {
	switch v {
	case "CONSTANTE":
		return "C"
	case "IDENTIFIANT":
		return "I"
	case "HAUTE-VARIANCE":
		return "H"
	default:
		return "?"
	}
}

// env114Motif rend l'ossature sous forme lisible : 0/1 aux positions constantes, « . » ailleurs.
func env114Motif(ossature map[int]int, nb int) string {
	var sb strings.Builder
	for b := 0; b < nb; b++ {
		if b%8 == 0 && b > 0 {
			sb.WriteByte(' ')
		}
		if v, ok := ossature[b]; ok {
			sb.WriteByte('0' + byte(v))
		} else {
			sb.WriteByte('.')
		}
	}
	return sb.String()
}

func env114Nom(dir string) string {
	dir = strings.TrimRight(dir, "/\\")
	if i := strings.LastIndexAny(dir, "/\\"); i >= 0 {
		return dir[i+1:]
	}
	return dir
}
