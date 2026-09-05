package filmdec

// vehicules_v5_balayage_test.go — LOT V5 : LE BALAYAGE SUPERVISÉ des records d'image-clé.
//
// LA QUESTION POSÉE AU FILM. Existe-t-il un DÉCALAGE EN BITS, stable d'un record à l'autre,
// tel que la valeur lue à ce décalage dans le record d'image-clé d'un bipède désigne le
// VÉHICULE qu'il occupe pendant les épisodes attestés — et rien du tout quand il est à pied ?
//
// AUCUNE GRAMMAIRE N'EST SUPPOSÉE. On lit des fenêtres de bits à tous les décalages, ancrées
// au DÉBUT du record puis à sa FIN (un champ de queue s'aligne sur la fin, pas sur le début).
// C'est la méthode déjà éprouvée par `ScanFilmKeyframeLoadouts` pour les armes portées.
//
// LE TÉMOIN EST INTÉGRÉ, pas optionnel : la MÊME lecture, aux MÊMES décalages, sur les
// bipèdes qui sont À PIED à ce même instant. Un décalage qui « désigne un véhicule » aussi
// souvent chez les piétons ne désigne rien.
//
//	CGO_ENABLED=0 V5_ROOT=<cache> V5_FILMS=... \
//	  go test ./internal/analysis/filmdec/ -run TestV5Balayage -v -timeout 120m

import (
	"fmt"
	"sort"
	"testing"
)

// v5MaxDecalage borne le balayage : au-delà, on sort du plus long record observé. Les records
// bipède d'image-clé mesurent ~2 800 bits (cf. keyframe_loadout.go) ; 6 000 laisse de la
// marge sans faire exploser le coût.
const v5MaxDecalage = 6000

// v5Extracteur décrit UNE façon de lire un slot à un décalage : la largeur de la fenêtre et
// la transformation qui en tire un slot candidat.
type v5Extracteur struct {
	Nom     string
	Largeur int
	Slot    func(uint64) uint32
}

// v5Extracteurs : les quatre formes de champ d'entité que le déserialiseur emploie ailleurs
// dans le film. Aucune n'est privilégiée — c'est la mesure qui tranche.
//
//	s13  : l'index brut de 13 bits (FUN_1406d3140 au range par défaut 0x1FFF).
//	g15h : 13 bits d'index puis 2 bits de génération (l'ordre lu par readVarWidthInt).
//	g15l : 2 bits de génération puis 13 bits d'index.
//	h32  : le handle complet `gen<<30 | slot` (FUN_14080d6f0).
var v5Extracteurs = []v5Extracteur{
	{"s13", 13, func(v uint64) uint32 { return uint32(v) }},
	{"g15h", 15, func(v uint64) uint32 { return uint32(v >> 2) }},
	{"g15l", 15, func(v uint64) uint32 { return uint32(v & 0x1FFF) }},
	{"h32", 32, func(v uint64) uint32 { return uint32(v) & 0x3FFFFFFF }},
}

// v5Ancre dit si le décalage se compte depuis le DÉBUT du record ou depuis sa FIN.
type v5Ancre int

const (
	v5AncreDebut v5Ancre = iota
	v5AncreFin
)

func (a v5Ancre) String() string {
	if a == v5AncreDebut {
		return "debut"
	}
	return "fin"
}

// v5Compteur agrège, pour UN (ancre, extracteur, décalage), les touches de la classe positive
// (le bipède est à bord) et celles du témoin (le bipède est à pied).
type v5Compteur struct {
	PosN, PosTouche int
	NegN, NegTouche int
}

// v5Cible est la clé d'un canal balayé.
type v5Cible struct {
	Ancre v5Ancre
	Extr  int
	Dec   int
}

// TestV5Balayage — LE BALAYAGE. Pour chaque film, chaque image-clé, chaque record bipède :
// la valeur lue à chaque décalage désigne-t-elle un véhicule VIVANT à cet instant ?
func TestV5Balayage(t *testing.T) {
	global := map[v5Cible]*v5Compteur{}
	for _, dir := range v5Films(t) {
		v5BalayageUnFilm(t, dir, global)
	}
	t.Logf("")
	t.Logf("V5 BALAYAGE — CUMUL TOUS FILMS")
	v5Verdict(t, global, "cumul")
}

func v5BalayageUnFilm(t *testing.T, dir string, global map[v5Cible]*v5Compteur) {
	t.Helper()
	eps, _, err := v5Episodes(dir)
	if err != nil {
		t.Logf("V5 BALAYAGE %s : %v", dir, err)
		return
	}
	kfs := v5Keyframes(dir)
	local := map[v5Cible]*v5Compteur{}
	instantsPos, instantsNeg := 0, 0
	for _, kf := range kfs {
		if len(kf) == 0 {
			continue
		}
		ts := kf[0].TS
		occupants := map[int]bool{}
		for _, e := range eps {
			if ts > e.DebutUS && ts < e.FinUS {
				occupants[int(e.Slot)] = true
			}
		}
		vehicules := map[uint32]bool{}
		for _, r := range kf {
			if r.TI == v5VehiculeTI {
				vehicules[uint32(r.Slot)] = true
			}
		}
		if len(vehicules) == 0 {
			continue
		}
		for _, r := range kf {
			if r.TI != v5BipedeTI {
				continue
			}
			pos := occupants[r.Slot]
			if pos {
				instantsPos++
			} else {
				instantsNeg++
			}
			v5BalayerRecord(r, vehicules, pos, local, global)
		}
	}
	t.Logf("V5 BALAYAGE %s — %d épisodes, %d images-clés ; lectures bipède : à bord=%d, "+
		"à pied (témoin)=%d", dir, len(eps), len(kfs), instantsPos, instantsNeg)
	v5Verdict(t, local, dir)
}

// v5BalayerRecord lit toutes les fenêtres d'UN record et incrémente les compteurs.
func v5BalayerRecord(
	r v5KfRec, cibles map[uint32]bool, positif bool,
	local, global map[v5Cible]*v5Compteur,
) {
	long := r.Fin - r.BitStart
	if long <= 0 {
		return
	}
	for ei, ex := range v5Extracteurs {
		maxDec := long - ex.Largeur
		if maxDec > v5MaxDecalage {
			maxDec = v5MaxDecalage
		}
		for d := 0; d <= maxDec; d++ {
			vDebut := ex.Slot(kfReadBits(r.Payload, r.BitStart+d, ex.Largeur))
			v5Marquer(local, global, v5Cible{v5AncreDebut, ei, d}, cibles[vDebut], positif)
			vFin := ex.Slot(kfReadBits(r.Payload, r.Fin-ex.Largeur-d, ex.Largeur))
			v5Marquer(local, global, v5Cible{v5AncreFin, ei, d}, cibles[vFin], positif)
		}
	}
}

func v5Marquer(local, global map[v5Cible]*v5Compteur, c v5Cible, touche, positif bool) {
	for _, m := range []map[v5Cible]*v5Compteur{local, global} {
		a := m[c]
		if a == nil {
			a = &v5Compteur{}
			m[c] = a
		}
		if positif {
			a.PosN++
			if touche {
				a.PosTouche++
			}
		} else {
			a.NegN++
			if touche {
				a.NegTouche++
			}
		}
	}
}

// v5SeuilPositif est la part minimale de touches côté « à bord » pour qu'un décalage soit
// seulement REGARDÉ. Écrit avant la mesure : un état d'occupation présent dans l'image-clé
// doit être là à CHAQUE image-clé de l'épisode, donc au moins 90 %.
const v5SeuilPositif = 0.90

// v5SeuilTemoin est la part maximale de touches côté témoin (à pied). Un champ qui désigne
// un véhicule chez un piéton n'est pas un champ d'occupation.
const v5SeuilTemoin = 0.10

// v5Verdict publie les décalages qui passent les deux seuils, et à défaut les dix meilleurs
// par écart positif - témoin (un négatif se lit sur ses chiffres, pas sur son silence).
func v5Verdict(t *testing.T, m map[v5Cible]*v5Compteur, quoi string) {
	t.Helper()
	type ligne struct {
		c    v5Cible
		a    *v5Compteur
		p, n float64
	}
	var toutes []ligne
	var retenues []ligne
	for c, a := range m {
		if a.PosN == 0 {
			continue
		}
		p := float64(a.PosTouche) / float64(a.PosN)
		n := 0.0
		if a.NegN > 0 {
			n = float64(a.NegTouche) / float64(a.NegN)
		}
		l := ligne{c, a, p, n}
		toutes = append(toutes, l)
		if p >= v5SeuilPositif && n <= v5SeuilTemoin {
			retenues = append(retenues, l)
		}
	}
	sort.Slice(toutes, func(i, j int) bool { return toutes[i].p-toutes[i].n > toutes[j].p-toutes[j].n })
	sort.Slice(retenues, func(i, j int) bool { return retenues[i].p-retenues[i].n > retenues[j].p-retenues[j].n })
	t.Logf("  [%s] canaux balayés=%d — décalages passant (à bord >= %.0f %%, témoin <= %.0f %%) : %d",
		quoi, len(toutes), v5SeuilPositif*100, v5SeuilTemoin*100, len(retenues))
	aff := retenues
	if len(aff) == 0 {
		aff = toutes
		t.Logf("  [%s] AUCUN décalage ne passe. Les dix MEILLEURS écarts (à bord - témoin) :", quoi)
	}
	for i, l := range aff {
		if i >= 10 {
			t.Logf("  [%s] ... et %d autres", quoi, len(aff)-10)
			break
		}
		t.Logf("    %-5s %-5s d=%-5d  à bord %4d/%-4d = %5.1f %%   témoin %5d/%-6d = %5.1f %%   écart %+6.1f pts",
			l.c.Ancre, v5Extracteurs[l.c.Extr].Nom, l.c.Dec,
			l.a.PosTouche, l.a.PosN, l.p*100, l.a.NegTouche, l.a.NegN, l.n*100, (l.p-l.n)*100)
	}
	_ = fmt.Sprint()
}
