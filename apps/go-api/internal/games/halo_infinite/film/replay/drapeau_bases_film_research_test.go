//go:build research

package replay

// drapeau_bases_film_research_test.go — INSTRUMENT DE MESURE : LA BASE ET LE CAMP DE CHAQUE
// DRAPEAU SE LISENT-ILS DANS LE FILM ?
//
// # LA QUESTION
//
// Sur une carte absente du catalogue de socles, le calque du drapeau vivant n'a aucune base : tous
// les portages tombent dans un drapeau d'equipe -1, sans etat `home`, et le client n'y dessine pas
// la zone de retour. L'hypothese a eprouver : les VOLS (`flag_steals`) d'un camp tombent tous au
// meme point, qui est la base du camp ADVERSE ; la vie libre de l'objet drapeau renait AU POINT de
// ce socle ; sur une partie « drapeau neutre », les vols des deux camps tombent au meme point.
//
// # LES SEUILS SONT CEUX DE LA PRODUCTION
//
// Ils ont ete ecrits ici avant la mesure, puis adoptes par la regle de production
// (`flag_film_bases.go`) : l'instrument lit desormais ses constantes, et le verdict imprime par
// film est celui que la regle rend.
//
// # LECTURE SEULE
//
// Les vols, les captures, les pistes et l'equipe de chaque piste viennent de l'ARTEFACT publie ;
// les vies libres de l'objet drapeau viennent du FICHIER DE FAITS (aucun octet de film). Les
// socles du catalogue des temoins sont lus sur l'artefact (etat `home` publie par drapeau).
//
//	L1_DEPOT=<checkout qui porte data/cache> \
//	  go test -tags research -count=1 -run '^TestDrapeauBasesDuFilm$' -v ./internal/games/halo_infinite/film/replay/

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// LES SEUILS DE DECISION SONT CEUX DE LA REGLE DE PRODUCTION (flag_film_bases.go), lus et non
// copies : le verdict imprime est celui que la production rend.
const (
	// l1RayonBase : rayon (m) du groupe de vols d'un camp, de la recherche d'une renaissance
	// autour du centre des vols, et de l'appariement a un socle du catalogue. Trois metres : un
	// vol se declenche quand le porteur touche le socle, l'axe du rejeu avance par 100 ms (moins
	// d'un metre de course), et deux bases d'equipe sont a 8,46 m au plus pres (Isolation).
	l1RayonBase = flagFilmBaseRadiusM
	// l1PartMin : part minimale des vols LOCALISES d'un camp que son groupe principal doit tenir.
	// Une base se lit a un seul point ; un camp dont les vols se partagent entre deux points (camps
	// qui changent de cote, piste mal nommee) ne donne pas de base.
	l1PartMin = flagFilmBaseMinShare
	// l1VolsMin : nombre minimal de vols localises par camp. LES DEUX CAMPS doivent voler : un
	// seul camp qui vole ne distingue pas la variante neutre de la variante ordinaire.
	l1VolsMin = 1
	// l1SeparationMin : distance minimale (m) entre les deux bases d'equipe. Jusqu'a l1RayonBase
	// les deux camps volent au meme point : variante neutre. Entre les deux : rien.
	l1SeparationMin = flagFilmBaseMinSeparationM
	// l1RenaissancesMin : nombre minimal de naissances de l'objet a moins de flagHomeExactDist les
	// unes des autres pour qu'un point soit une RENAISSANCE (un lacher ne se repete pas au
	// centimetre ; le moteur repose le drapeau au point du socle).
	l1RenaissancesMin = flagFilmRebirthMin
)

type l1Film struct {
	id, groupe string
}

var l1Films = []l1Film{
	{"fd247c3f", "absent"}, {"f7b74a65", "absent"}, {"fa70c437", "absent"},
	{"92c950ee", "absent"}, {"f1db4a07", "absent"}, {"798d1ff4", "absent"},
	{"5da3e346", "absent"}, {"708abcd1", "absent"}, {"73c1df0b", "absent"}, {"be758198", "absent"},
	{"eba1e63f", "temoin"}, {"068fb1ac", "temoin"}, {"1e5e355e", "temoin"},
	{"a32ee8d2", "equipe-1"}, {"61614156", "equipe-1"}, {"81cc9952", "equipe-1"},
	{"084a804d", "equipe-1"}, {"db6dc73c", "equipe-1"}, {"390b1de5", "equipe-1"},
	{"e94163af", "neutre"}, {"a1995edc", "neutre"}, {"059b721f", "neutre"}, {"323ec1cf", "neutre"},
}

type l1Pt struct{ x, y float64 }

func l1Dist(a, b l1Pt) float64 { return math.Hypot(a.x-b.x, a.y-b.y) }

// l1Groupe rend le groupe principal d'un nuage : le point qui a le plus de voisins a moins du
// rayon, ses voisins, leur centre et leur dispersion (distance maximale au centre).
func l1Groupe(pts []l1Pt) (centre l1Pt, n int, dispersion float64) {
	best := -1
	for i := range pts {
		c := 0
		for j := range pts {
			if l1Dist(pts[i], pts[j]) <= l1RayonBase {
				c++
			}
		}
		if c > n {
			best, n = i, c
		}
	}
	if best < 0 {
		return l1Pt{}, 0, 0
	}
	var sx, sy float64
	var membres []l1Pt
	for _, p := range pts {
		if l1Dist(pts[best], p) <= l1RayonBase {
			membres = append(membres, p)
			sx, sy = sx+p.x, sy+p.y
		}
	}
	centre = l1Pt{sx / float64(len(membres)), sy / float64(len(membres))}
	for _, p := range membres {
		dispersion = math.Max(dispersion, l1Dist(centre, p))
	}
	return centre, n, dispersion
}

// l1Renaissances rend les points de RENAISSANCE : les groupes d'au moins l1RenaissancesMin
// naissances a moins de flagHomeExactDist les unes des autres, avec leur effectif.
func l1Renaissances(nais []l1Pt) map[l1Pt]int {
	out := map[l1Pt]int{}
	pris := make([]bool, len(nais))
	for i := range nais {
		if pris[i] {
			continue
		}
		var groupe []int
		for j := range nais {
			if !pris[j] && l1Dist(nais[i], nais[j]) <= flagHomeExactDist {
				groupe = append(groupe, j)
			}
		}
		if len(groupe) < l1RenaissancesMin {
			continue
		}
		var sx, sy float64
		for _, j := range groupe {
			pris[j] = true
			sx, sy = sx+nais[j].x, sy+nais[j].y
		}
		out[l1Pt{sx / float64(len(groupe)), sy / float64(len(groupe))}] = len(groupe)
	}
	return out
}

// l1PointA rend la position publiee d'un joueur a une frame (+-1), et l'equipe de la piste.
func l1PointA(doc *ReplayDocument, xuid string, t int) (l1Pt, int, bool) {
	for _, tr := range doc.Tracks {
		if tr.XUID != xuid {
			continue
		}
		for _, p := range tr.Points {
			if p.T >= t-1 && p.T <= t+1 {
				return l1Pt{float64(p.X), float64(p.Y)}, tr.Team, true
			}
		}
	}
	return l1Pt{}, 0, false
}

func TestDrapeauBasesDuFilm(t *testing.T) {
	depot := os.Getenv("L1_DEPOT")
	if depot == "" {
		t.Skip("instrument de mesure : L1_DEPOT requis")
	}
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(repoRootForTest(t), "data", "titles",
		"halo_infinite", "reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatal(err)
	}
	labels := goldenCatalog(t)
	for _, f := range l1Films {
		t.Run(f.groupe+"/"+f.id, func(t *testing.T) { l1Mesurer(t, depot, f, cat, labels) })
	}
}

func l1Mesurer(t *testing.T, depot string, f l1Film, cat *profile.MapQuantCatalog, labels LabelCatalog) {
	blob, err := os.ReadFile(filepath.Join(depot, "data", "cache", "replays", "halo_infinite", f.id+".json")) //nolint:gosec // mesure
	if err != nil {
		t.Skipf("artefact absent : %v", err)
	}
	var doc ReplayDocument
	if err := json.Unmarshal(blob, &doc); err != nil {
		t.Fatal(err)
	}
	vols, captures := map[int][]l1Pt{}, map[int][]l1Pt{}
	sansPos := 0
	for _, o := range doc.Objectives {
		if o.Stat != "flag_steals" && o.Stat != "flag_captures" {
			continue
		}
		p, team, ok := l1PointA(&doc, o.XUID, o.T)
		if !ok || team < 0 {
			sansPos++
			continue
		}
		if o.Stat == "flag_steals" {
			vols[team] = append(vols[team], p)
		} else {
			captures[team] = append(captures[team], p)
		}
	}
	renaissances := l1RenaissancesDesFaits(t, depot, f.id, cat, labels)
	t.Logf("%s [%s] : vols localises par camp %v, sans position %d, renaissances %d points",
		f.id, f.groupe, l1Comptes(vols), sansPos, len(renaissances))
	centres := map[int]l1Pt{}
	verdictCamps := true
	for _, team := range l1Cles(vols) {
		c, n, disp := l1Groupe(vols[team])
		part := float64(n) / float64(len(vols[team]))
		centres[team] = c
		rn, rd := l1RenaissanceProche(renaissances, c)
		cc, cn, _ := l1Groupe(captures[team])
		t.Logf("   camp %d : %d vols, groupe %d (part %.2f), centre (%.2f, %.2f), dispersion %.2f m ; "+
			"renaissance la plus proche %.3f m (%d naissances) ; captures %d, centre (%.2f, %.2f)",
			team, len(vols[team]), n, part, c.x, c.y, disp, rd, rn, cn, cc.x, cc.y)
		if n < l1VolsMin || part < l1PartMin {
			verdictCamps = false
		}
		l1Temoin(t, &doc, team, c, renaissances)
	}
	t.Logf("   VERDICT : %s", l1Verdict(centres, verdictCamps))
}

// l1Verdict rend ce que la regle deciderait.
func l1Verdict(centres map[int]l1Pt, campsOK bool) string {
	if len(centres) != 2 || !campsOK {
		return fmt.Sprintf("AUCUNE BASE (camps qui volent %d, groupes conformes %v)", len(centres), campsOK)
	}
	cles := l1Cles(centres)
	d := l1Dist(centres[cles[0]], centres[cles[1]])
	switch {
	case d <= l1RayonBase:
		return fmt.Sprintf("NEUTRE (centres a %.2f m)", d)
	case d >= l1SeparationMin:
		return fmt.Sprintf("DEUX BASES (separation %.1f m)", d)
	default:
		return fmt.Sprintf("AUCUNE BASE (separation ambigue %.1f m)", d)
	}
}

// l1Temoin compare le centre des vols d'un camp aux socles publies sur l'artefact (etat `home`).
func l1Temoin(t *testing.T, doc *ReplayDocument, team int, c l1Pt, r map[l1Pt]int) {
	for _, fc := range doc.FlagCarries {
		for _, s := range fc.Spans {
			if s.State != FlagStateHome {
				continue
			}
			t.Logf("      socle publie du drapeau d'equipe %d (%.2f, %.2f) : a %.3f m du centre des vols du camp %d",
				fc.Team, s.X, s.Y, l1Dist(c, l1Pt{float64(s.X), float64(s.Y)}), team)
			rn, rd := l1RenaissanceProche(r, l1Pt{float64(s.X), float64(s.Y)})
			t.Logf("         renaissance la plus proche de ce socle : %.4f m (%d naissances)", rd, rn)
			break
		}
	}
}

func l1RenaissanceProche(r map[l1Pt]int, c l1Pt) (int, float64) {
	n, d := 0, math.Inf(1)
	for p, k := range r {
		if dd := l1Dist(p, c); dd < d {
			n, d = k, dd
		}
	}
	return n, d
}

// l1RenaissancesDesFaits lit les vies libres de l'objet drapeau dans le fichier de faits.
func l1RenaissancesDesFaits(t *testing.T, depot, id string, cat *profile.MapQuantCatalog,
	labels LabelCatalog) map[l1Pt]int {
	blob, err := os.ReadFile(filepath.Join(depot, "data", "cache", "film_facts", "halo_infinite", id+".filmfacts.bin")) //nolint:gosec // mesure
	if err != nil {
		t.Logf("   faits absents : %v", err)
		return nil
	}
	head, err := DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Logf("   en-tete illisible : %v", err)
		return nil
	}
	var entry profile.MapQuantEntry
	for _, e := range cat.Maps {
		if e.Module == head.MapModule {
			entry = e
			break
		}
	}
	ff, err := DecodeFilmFactsFile(blob, entry)
	if err != nil {
		t.Logf("   faits illisibles : %v", err)
		return nil
	}
	vies := flagFreeLives(ff.Facts.FilmInputs.Pads.Weapons, labels.ObjectiveObjects)
	nais := make([]l1Pt, 0, len(vies))
	for _, v := range vies {
		x, y := v.First()
		nais = append(nais, l1Pt{float64(x), float64(y)})
	}
	return l1Renaissances(nais)
}

func l1Comptes(m map[int][]l1Pt) map[int]int {
	out := map[int]int{}
	for k, v := range m {
		out[k] = len(v)
	}
	return out
}

func l1Cles[V any](m map[int]V) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
