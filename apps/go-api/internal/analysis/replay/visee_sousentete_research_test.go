package replay

// visee_sousentete_research_test.go — LOT A4 (suite) : L'HYPOTHESE DU SOUS-EN-TETE, TESTEE SUR
// L'ANCRE 105 PUIS SUR LE 114.
//
// L'HYPOTHESE. L'ancre du type 105 a montre qu'aucune grammaire « [type][porte][var-int] » ne
// place l'index d'attaquant a son offset connu (bit 36) : il manque des bits entre le type et
// les references. L'hypothese proposee est un SOUS-EN-TETE DE LONGUEUR FIXE S, commun aux
// evenements, apres quoi viennent les trois references du dispatcher. Avec S = 16 et les
// largeurs lues dans le binaire (lot B), le type 105 fermerait ainsi :
//
//	[0..6 type][7..22 sous-en-tete 16 b][23 porte ref0][24 sonde du domaine 1]
//	[25..33 valeur 9 b][34..35 generation] -> le champ suivant commence au bit 36.
//
// DEUX MESURES, chacune falsifiable :
//
//	M1. ANCRE 105 — pour chaque longueur S de 0 a 24, on calcule ou tomberait le champ qui suit
//	    la reference 0 (7 + S + 1 porte + 1 sonde + 9 + 2 generation = 20 + S), et l'on ne
//	    retient que les S qui tombent sur 36. Pour ceux-la on PUBLIE les valeurs qui devraient
//	    verifier l'hypothese : la porte doit etre ouverte, la sonde majoritairement a 1, et
//	    l'index tenir sous le cardinal 0x200 du domaine 1 avec sonde.
//	    SEUILS ECRITS AVANT : porte ouverte sur >= 95 % des records longs ; sonde a 1 sur
//	    >= 60 % ; index sous 0x200 sur >= 95 %.
//	M2. TYPE 114 — balayage de S de 0 a 24 avec les portes LUES et les largeurs du binaire
//	    (domaine 2 : 8 bits, domaine 3 : 8, domaine 7 : 13 ; sonde pour le seul domaine 1, donc
//	    aucune ici). Un S est RETENU si, sur les trois films, la charge utile R(6) qui suit les
//	    trois portes a une cardinalite <= 8 et une palette recouvrant >= 90 % des paquets des
//	    autres films — les memes criteres que la fermeture precedente.
//
// SOUS GARDE (SOUSENTETE_FILMS).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 SOUSENTETE_FILMS=<repo>/data/cache/film_chunks/00162144,<repo>/data/cache/film_chunks/00ba2e1c,<repo>/data/cache/film_chunks/03ccbe42 \
//	  go test ./internal/analysis/replay/ -run TestViseeSousEntete -v -timeout 30m

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	sousEnteteFilmsEnv = "SOUSENTETE_FILMS"
	sousEnteteAncre105 = 36 // offset connu de l'index d'attaquant (filmdec/fire_events.go)
	sousEnteteWDom2    = 8  // largeurs lues dans le binaire par le lot B
	sousEnteteWDom3    = 8
	sousEnteteWDom7    = 13
	sousEnteteWDom1    = 9 // domaine 1 AVEC sonde (cardinal 0x200)
)

// sousEnteteTaux rend la part des paquets dont le bit a la position donnee vaut 1.
func sousEnteteTaux(pk []env114Paquet, pos int) (float64, int) {
	var uns, total int
	for _, p := range pk {
		if pos >= p.nBits {
			continue
		}
		total++
		uns += int(filmdec.ReadBitsAtForDiag(p.pay, pos, 1))
	}
	if total == 0 {
		return 0, 0
	}
	return float64(uns) / float64(total), total
}

// sousEnteteSousSeuil rend la part des paquets dont la valeur lue est strictement inferieure
// a la borne donnee.
func sousEnteteSousSeuil(pk []env114Paquet, pos, w int, borne uint32) float64 {
	var ok, total int
	for _, p := range pk {
		if pos+w > p.nBits {
			continue
		}
		total++
		if filmdec.ReadBitsAtForDiag(p.pay, pos, w) < borne {
			ok++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(ok) / float64(total)
}

// sousEnteteAncre — mesure M1 : l'hypothese ferme-t-elle sur le type 105 ?
func sousEnteteAncre(t *testing.T, nom string, pk []env114Paquet) {
	t.Helper()
	var longs []env114Paquet
	for _, p := range pk {
		if p.nBits > sousEnteteAncre105+5 && filmdec.ReadBitsAtForDiag(p.pay, 7, 1) == 0 {
			longs = append(longs, p)
		}
	}
	if len(longs) == 0 {
		return
	}
	for s := 0; s <= 24; s++ {
		if 20+s != sousEnteteAncre105 {
			continue
		}
		posPorte := 7 + s
		tauxPorte, n := sousEnteteTaux(longs, posPorte)
		tauxSonde, _ := sousEnteteTaux(longs, posPorte+1)
		partIdx := sousEnteteSousSeuil(longs, posPorte+2, sousEnteteWDom1, 0x200)
		verdict := "REFUTEE"
		if tauxPorte >= 0.95 && tauxSonde >= 0.60 && partIdx >= 0.95 {
			verdict = "COMPATIBLE"
		}
		t.Logf("M1. ANCRE 105 [%s] S=%d — %d records longs ; porte (bit %d) ouverte %.1f %% ·"+
			" sonde (bit %d) a 1 %.1f %% · index 9 b (bits %d..%d) sous 0x200 %.1f %% -> %s",
			nom, s, n, posPorte, 100*tauxPorte, posPorte+1, 100*tauxSonde, posPorte+2,
			posPorte+10, 100*partIdx, verdict)
	}
}

// sousEnteteDecode rejoue les trois references apres un sous-en-tete de S bits.
func sousEnteteDecode(p env114Paquet, s int) (payload uint32, g0, g1, g2 uint32, ok bool) {
	lire := func(pos, n int) (uint32, bool) {
		if pos+n > p.nBits {
			return 0, false
		}
		return filmdec.ReadBitsAtForDiag(p.pay, pos, n), true
	}
	pos := 7 + s
	if g0, ok = lire(pos, 1); !ok {
		return
	}
	pos++
	if g0 == 1 {
		pos += sousEnteteWDom2 + 2
	}
	if g1, ok = lire(pos, 1); !ok {
		return
	}
	pos++
	if g1 == 1 {
		pos += sousEnteteWDom3 + 2
	}
	if g2, ok = lire(pos, 1); !ok {
		return
	}
	pos++
	if g2 == 1 {
		pos += sousEnteteWDom7 + 2
	}
	payload, ok = lire(pos, larg114Payload)
	return
}

// sousEnteteProfil resume un S sur un film.
type sousEnteteProfil struct {
	palette        map[uint32]int
	n              int
	p0, p1, p2     float64
	posMoyPayload  float64
	cardinalitePay int
}

func sousEnteteMesure(pk []env114Paquet, s int) sousEnteteProfil {
	pr := sousEnteteProfil{palette: map[uint32]int{}}
	var a, b, c int
	for _, p := range pk {
		pay, g0, g1, g2, ok := sousEnteteDecode(p, s)
		if !ok {
			continue
		}
		pr.n++
		pr.palette[pay]++
		a, b, c = a+int(g0), b+int(g1), c+int(g2)
	}
	if pr.n == 0 {
		return pr
	}
	pr.p0, pr.p1, pr.p2 = float64(a)/float64(pr.n), float64(b)/float64(pr.n), float64(c)/float64(pr.n)
	pr.cardinalitePay = len(pr.palette)
	return pr
}

func sousEnteteRecouvre(a, b sousEnteteProfil) float64 {
	if b.n == 0 {
		return 0
	}
	var commun int
	for v, n := range b.palette {
		if a.palette[v] > 0 {
			commun += n
		}
	}
	return float64(commun) / float64(b.n)
}

// sousEntete114 — mesure M2 : quels S rendent une charge utile stable sur les trois films ?
func sousEntete114(t *testing.T, noms []string, lots map[string][]env114Paquet) {
	t.Helper()
	var retenus []int
	for s := 0; s <= 24; s++ {
		profils := make([]sousEnteteProfil, len(noms))
		bon := true
		for i, nom := range noms {
			profils[i] = sousEnteteMesure(lots[nom], s)
			if profils[i].n == 0 || profils[i].cardinalitePay > larg114CardMax {
				bon = false
			}
		}
		if bon {
			for i := range profils {
				for j := range profils {
					if i != j && sousEnteteRecouvre(profils[i], profils[j]) < 0.90 {
						bon = false
					}
				}
			}
		}
		if !bon {
			continue
		}
		retenus = append(retenus, s)
		var parts []string
		for i, nom := range noms {
			parts = append(parts, fmt.Sprintf("[%s] portes %.0f/%.0f/%.0f %% payload %s", nom,
				100*profils[i].p0, 100*profils[i].p1, 100*profils[i].p2,
				sousEntetePalette(profils[i].palette)))
		}
		t.Logf("M2. TYPE 114 — S=%d RETENU : %s", s, strings.Join(parts, " · "))
	}
	t.Logf("M2. BILAN — %d longueurs de sous-en-tete retenues sur 25 testees : %v", len(retenus),
		retenus)
}

func sousEntetePalette(m map[uint32]int) string {
	var vs []int
	for v := range m {
		vs = append(vs, int(v))
	}
	sort.Ints(vs)
	var parts []string
	for _, v := range vs {
		parts = append(parts, fmt.Sprintf("%d:%d", v, m[uint32(v)]))
	}
	return "{" + strings.Join(parts, " ") + "}"
}

// TestViseeSousEntete execute M1 puis M2.
func TestViseeSousEntete(t *testing.T) {
	liste := os.Getenv(sousEnteteFilmsEnv)
	if liste == "" {
		t.Skipf("%s absent : instrument saute", sousEnteteFilmsEnv)
	}
	lots := map[string][]env114Paquet{}
	var noms []string
	for _, dir := range strings.Split(liste, ",") {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		nom := env114Nom(dir)
		if pk := env114Collecte(dir); len(pk) > 0 {
			lots[nom], noms = pk, append(noms, nom)
		}
		sousEnteteAncre(t, nom, env114CollecteType(dir, 105))
	}
	if len(noms) == 0 {
		t.Fatalf("aucun film exploitable")
	}
	// Rappel de la mesure qui contraint deja l'hypothese sur le 114 : l'etat des bits 20..25.
	for _, nom := range noms {
		var etats []string
		for b := 20; b <= 25; b++ {
			taux, _ := sousEnteteTaux(lots[nom], b)
			etats = append(etats, fmt.Sprintf("%d:%.2f", b, taux))
		}
		t.Logf("CONTRAINTE [%s] — part de bits a 1 aux positions 20..25 : %s", nom,
			strings.Join(etats, " "))
	}
	sousEntete114(t, noms, lots)
}
