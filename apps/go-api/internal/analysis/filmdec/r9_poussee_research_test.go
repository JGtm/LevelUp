package filmdec

// r9_poussee_research_test.go — LA FACE VICTIME (par. 6 du RAPPORT_R9_REPULSEUR_2026-09-03) :
// le film porte-t-il l'EFFET du repulseur ?
//
// LA QUESTION, ET POURQUOI ELLE N'EST PAS LA MEME QUE LES PRECEDENTES. Les portes (a) et (b)
// cherchaient le GESTE : une marque, dans le flux, a l'instant ou le porteur appuie. Elles
// ont echoue. Reste une hypothese que l'utilisateur peut trancher d'un regard mais que la
// mesure doit etablir : **le film porte peut-etre l'EFFET sans porter le GESTE**. La victime
// d'un repulseur est projetee, et sa position EST repliquee — le visionneur du jeu la voit
// partir sans avoir besoin qu'on lui dise pourquoi.
//
// Cette hypothese n'est publiable que MESUREE, avec sa symetrie inverse comme controle.
//
// DEFINITIONS, ECRITES AVANT LA MESURE (pre-inscription, cf. RAPPORT_R9 par. 1.4) :
//
//	POUSSEE     un bipede dont la vitesse horizontale passe de <= 3,0 m/s a >= 6,0 m/s entre
//	            deux echantillons consecutifs distants d'au plus 250 ms. Les memes seuils que
//	            l'oracle physique de R8 (le grappin y sort a 6,04 m/s de pic median contre
//	            2,91 au hasard) : ils ne sont pas choisis pour cette mesure-ci.
//	ATTRIBUTION une poussee est attribuee a un PORTEUR si celui-ci est a <= 6,0 m au moment
//	            du pas ET si la nouvelle vitesse s'eloigne de lui (produit scalaire positif
//	            avec la direction radiale porteur -> victime). Le rayon est celui de R8.
//	EXPOSITION  le denominateur : le temps cumule pendant lequel la victime est a <= 6,0 m du
//	            porteur. Sans lui, un rang tres porte et un rang rare ne se comparent pas.
//	EPISODE     deux poussees du meme couple (victime, porteur) a moins d'1 s n'en font qu'une.
//
// CRITERES PRE-INSCRITS :
//
//  1. CIBLE — le taux de poussees SUBIES PAR LES VOISINS d'un porteur de REPULSEUR, par
//     minute d'exposition, est >= 2x le maximum des rangs temoins (detecteur de menaces,
//     mur de protection, champ de reparation, ecran occultant — des equipements sans effet
//     cinetique sur autrui).
//  2. TEMOIN POSITIF PAR SYMETRIE INVERSE — le taux de poussees PROPRES (le porteur lui-meme)
//     doit etre maximal sur le rang du PROPULSEUR et quasi nul sur les rangs temoins. Un
//     instrument qui ne retrouve pas la bouffee du propulseur, dont R8 a etabli les instants,
//     ne mesure rien ; son resultat sur le repulseur serait sans valeur.
//  3. NON-RETOUR — le porteur de repulseur ne doit PAS montrer d'exces de poussee propre :
//     s'il en montrait un, le geste serait lisible et on serait revenu au canal de R8.
//
// GARDES : `R8_FILMS`, `R8_BOUNDS`, `R8_IDS`.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R8_FILMS=<repo>/data/cache/film_chunks \
//	  R8_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  R8_IDS=00ba2e1c,06dfe6d9 go test ./internal/analysis/filmdec/ \
//	  -run '^TestR9Poussee$' -count=1 -timeout 120m -v

import (
	"math"
	"path/filepath"
	"sort"
	"testing"
)

const (
	// r9PousseeBasseMS : au-dessous, le bipede est « au repos cinetique » (marche/course
	// normale). Valeur de R8 : la mediane du temoin aleatoire y est de 2,7 a 3,5 m/s.
	r9PousseeBasseMS = 3.0
	// r9PousseeHauteMS : au-dessus, le bipede subit une impulsion. Valeur de R8 : le grappin
	// sort a 6,04 m/s de pic median sur la fenetre large.
	r9PousseeHauteMS = 6.0
	// r9RayonM : rayon d'attribution et d'exposition. Meme valeur que l'oracle du voisin de
	// R8, pour que les deux mesures se comparent.
	r9RayonM = 6.0
	// r9EpisodeUS : deux poussees du meme couple a moins d'1 s n'en font qu'une.
	r9EpisodeUS = 1_000_000
)

// r9Pas est UN pas de deplacement d'un bipede : ses deux echantillons, sa vitesse et sa
// direction horizontale normalisee.
type r9Pas struct {
	t0, t1     uint64
	x, y       float64 // position d'arrivee
	v          float64 // vitesse horizontale du pas
	dx, dy     float64 // direction normalisee du pas
	dtS        float64
	valide     bool
	vPrecedent float64
}

// r9Pistes construit, par slot, la suite des pas exploitables.
func r9Pistes(pos []BipedPosition) map[uint32][]r9Pas {
	bySlot := map[uint32][]BipedPosition{}
	for _, p := range pos {
		if p.HasWorld {
			bySlot[p.Slot] = append(bySlot[p.Slot], p)
		}
	}
	out := map[uint32][]r9Pas{}
	for slot, list := range bySlot {
		sort.Slice(list, func(i, j int) bool { return list[i].TimestampUS < list[j].TimestampUS })
		prev := 0.0
		for i := 1; i < len(list); i++ {
			a, b := list[i-1], list[i]
			dt := b.TimestampUS - a.TimestampUS
			if dt == 0 || dt > r8SpeedMaxDtUS {
				prev = 0
				continue
			}
			ddx, ddy := float64(b.X-a.X), float64(b.Y-a.Y)
			d := math.Hypot(ddx, ddy)
			dts := float64(dt) / 1e6
			p := r9Pas{t0: a.TimestampUS, t1: b.TimestampUS, x: float64(b.X), y: float64(b.Y),
				v: d / dts, dtS: dts, valide: true, vPrecedent: prev}
			if d > 0 {
				p.dx, p.dy = ddx/d, ddy/d
			}
			out[slot] = append(out[slot], p)
			prev = p.v
		}
	}
	return out
}

// r9Compteur agrege, pour un rang de capacite, l'exposition et les poussees.
type r9Compteur struct {
	exposS      float64 // secondes-victime passees a <= 6 m d'un porteur de ce rang
	poussees    int     // episodes de poussee attribues (voisin)
	viesS       float64 // secondes de vie cumulees des porteurs de ce rang
	pousseesSoi int     // episodes de poussee du PORTEUR lui-meme
}

func TestR9Poussee(t *testing.T) {
	for _, dir := range r8FilmDirs(t) {
		r9PousseeOneFilm(t, dir)
	}
}

func r9PousseeOneFilm(t *testing.T, dir string) {
	t.Helper()
	entry := r8MapEntry(t, dir)
	wr := entry.Range()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	SetWorldObjectPrecisionFromLayout(entry.Layout())
	defer func() { WorldObjectPrecision = saved }()

	opt := DefaultScanFilmOptions()
	opt.WorldRange = &wr
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("positions illisibles : %v", err)
	}
	speeds := r8BuildSpeeds(pos)
	lives := r8Lives(speeds)
	ranks, _, err := ScanFilmAbilityRanks(dir)
	if err != nil {
		t.Logf("rangs de capacite illisibles : %v", err)
	}
	pistes := r9Pistes(pos)
	cnt := r9Mesure(pistes, ranks, lives)
	r9LogPoussees(t, filepath.Base(dir), cnt)
}

// r9Mesure parcourt tous les pas de tous les bipedes et remplit les compteurs par rang.
// La boucle est O(pas x slots) : ~240 000 pas x ~12 slots par film, borne et sans allocation
// dans la boucle interne.
func r9Mesure(
	pistes map[uint32][]r9Pas, ranks []AbilityRank, lives map[uint32][]r8LifeSpan,
) map[int]*r9Compteur {
	out := map[int]*r9Compteur{}
	get := func(r int) *r9Compteur {
		c := out[r]
		if c == nil {
			c = &r9Compteur{}
			out[r] = c
		}
		return c
	}
	dernier := map[[2]uint32]uint64{}
	dernierSoi := map[uint32]uint64{}
	for victime, pas := range pistes {
		for _, p := range pas {
			estPoussee := p.vPrecedent > 0 && p.vPrecedent <= r9PousseeBasseMS &&
				p.v >= r9PousseeHauteMS
			// FACE PORTEUR (temoin positif par symetrie inverse) : la victime est ici son
			// propre porteur — on compte sa bouffee sur SON rang.
			if rs := r8RankInLife(ranks, lives, victime, p.t1); rs >= 0 {
				c := get(rs)
				c.viesS += p.dtS
				if estPoussee {
					if prev, ok := dernierSoi[victime]; !ok || p.t1-prev > r9EpisodeUS {
						c.pousseesSoi++
					}
					dernierSoi[victime] = p.t1
				}
			}
			// FACE VICTIME : tous les porteurs a <= 6 m recoivent l'exposition, et la poussee
			// si elle s'eloigne d'eux.
			for porteur, autre := range pistes {
				if porteur == victime {
					continue
				}
				q, ok := r9PasAt(autre, p.t1)
				if !ok {
					continue
				}
				rx, ry := p.x-q.x, p.y-q.y
				dist := math.Hypot(rx, ry)
				if dist > r9RayonM {
					continue
				}
				rang := r8RankInLife(ranks, lives, porteur, p.t1)
				c := get(rang)
				c.exposS += p.dtS
				if !estPoussee || dist == 0 || p.dx*rx/dist+p.dy*ry/dist <= 0 {
					continue
				}
				k := [2]uint32{victime, porteur}
				if prev, ok := dernier[k]; !ok || p.t1-prev > r9EpisodeUS {
					c.poussees++
				}
				dernier[k] = p.t1
			}
		}
	}
	return out
}

// r9PasAt rend le pas du slot le plus proche en temps de `at`, si son arrivee est a moins de
// 100 ms — une position vieillie n'est plus une position (meme regle que `r8NearestAt`).
func r9PasAt(list []r9Pas, at uint64) (r9Pas, bool) {
	i := sort.Search(len(list), func(k int) bool { return list[k].t1 >= at })
	best, ok := r9Pas{}, false
	for _, k := range []int{i - 1, i} {
		if k < 0 || k >= len(list) {
			continue
		}
		if d := r8AbsDiff(list[k].t1, at); d <= 100_000 {
			if !ok || d < r8AbsDiff(best.t1, at) {
				best, ok = list[k], true
			}
		}
	}
	return best, ok
}

// r9LogPoussees publie les deux faces cote a cote : ce que SUBISSENT les voisins d'un porteur,
// et ce que le porteur subit LUI-MEME. C'est la comparaison des deux colonnes qui tranche.
func r9LogPoussees(t *testing.T, film string, cnt map[int]*r9Compteur) {
	t.Helper()
	keys := make([]int, 0, len(cnt))
	for k := range cnt {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	t.Logf("%s : poussees (pas de <= %.1f a >= %.1f m/s, rayon %.1f m)",
		film, r9PousseeBasseMS, r9PousseeHauteMS, r9RayonM)
	t.Logf("  %-6s %10s %9s %12s   %10s %9s %12s", "rang", "exposMin", "poussVois",
		"parMinVois", "vieMin", "poussSoi", "parMinSoi")
	for _, k := range keys {
		c := cnt[k]
		t.Logf("  %-6d %10.2f %9d %12.3f   %10.2f %9d %12.3f",
			k, c.exposS/60, c.poussees, r9Taux(c.poussees, c.exposS),
			c.viesS/60, c.pousseesSoi, r9Taux(c.pousseesSoi, c.viesS))
	}
}

func r9Taux(n int, secondes float64) float64 {
	if secondes <= 0 {
		return 0
	}
	return float64(n) / (secondes / 60)
}
