//go:build research

package grammar

// deto_position_mesure_research_test.go — INSTRUMENT : la position et l'instant du message de
// detonation/impact de projectile (genre 5/6, [ScanProjectileDetonations]) compares a ce que le
// rejeu publie AUJOURD'HUI comme explosion : la DERNIERE POSITION REPLIQUEE de la piste de
// projectile liee au lancer ([ScanProjectiles]). Lot detonations du 2026-10-22, item 1 du cadrage
// (controle n° 24 des controles v8, 2026-10-21 : des lancers de grenade sans explosion au rejeu).
//
// LA QUESTION. [chargeDetonation]/[chargeImpact] lisaient deja cette position puis la jetaient ;
// ce lot la garde (grammar/vue_a_detonations.go, detonations.go, ZERO bit de plus consomme). Avant
// de la publier au rejeu, on COMPTE l'evenement dans le film reel : combien de detonations, s'
// apparient-elles aux pistes de projectile deja liees a un lancer, et l'instant/la position
// qu'elles portent AMELIORENT-ils le repli actuel (qui pose l'explosion au dernier point connu,
// 1 a 1,5 s avant la fin reelle de la meche pour une frag) ?
//
// METHODE, par film :
//  1. compte brut : messages lus, avec reference de projectile, avec position (ScanProjectileDetonations.stats) ;
//  2. lancer -> piste : la MEME regle que la production (naissance la plus proche dans la fenetre
//     de 200 ms, cf. replay/grenades.go), simplifiee ici (pas de pont bipede : le but est de
//     mesurer la position de detonation, pas l'attribution de l'auteur) ;
//  3. piste -> detonation : parmi les detonations A POSITION, celle la plus proche dans l'espace
//     du DERNIER point replique de la piste, dans une fenetre de temps qui laisse la meche courir
//     (jusqu'a 4 s apres la derniere position, une tolerance de 0,5 s avant) ;
//  4. TEMOIN : la meme recherche sur les detonations DECALEES d'un demi-tour de liste (meme jeu de
//     valeurs, decorrele de CETTE piste) — le plancher de coincidence spatiale au hasard ;
//  5. la reference de projectile (domaine 5), quand la piste->detonation s'apparie : coincide-t-
//     elle avec le SLOT de la piste, brute ou avec la base 0x400 du domaine (varwidth.go) ?
//
// USAGE (lecture seule, aucun film = saut propre) :
//
//	DETOPOS_FILMS=<racine data/cache> DETOPOS_IDS='d6918972=origin;d9781168=dredge;bfecd02b-9798-4407-aa55-05244f4c1fa0=snowbound;4f77afc1=flood gulch;92c950ee-9aad-4471-a076-98882738f869=catalyst' \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar -run TestDetoPositionMesure -v -timeout 1800s

import (
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/filmcache"
	"github.com/JGtm/LevelUp/libs/film/internal/profile"
	"github.com/JGtm/LevelUp/libs/film/types"
)

// detoPosDomaine5Base est la base que le jeu ajoute a la categorie 5 de `FUN_1406d3140`
// (varwidth.go, i==5 -> base 0x400) — NOTRE PORT ne l'ajoute PAS ([readVarWidthInt] rend la valeur
// brute) : la mesure 5 ci-dessus confronte donc m.Ref0 brut ET m.Ref0+cette base au slot de piste.
const detoPosDomaine5Base = 0x400

func TestDetoPositionMesure(t *testing.T) {
	racine, liste := os.Getenv("DETOPOS_FILMS"), os.Getenv("DETOPOS_IDS")
	if racine == "" || liste == "" {
		t.Skip("DETOPOS_FILMS ou DETOPOS_IDS absent : instrument saute")
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	for _, couple := range strings.Split(liste, ";") {
		id, carte, _ := strings.Cut(couple, "=")
		entree, err := cat.Lookup(carte)
		if err != nil {
			t.Fatalf("%s : carte %q : %v", id, carte, err)
		}
		film, ok, err := filmcache.LoadFilm(racine, id)
		if err != nil || !ok {
			t.Fatalf("film %s : %v (present %v)", id, err, ok)
		}
		detoPositionUnFilm(t, id, aj14fContexte(film, &entree), entree.Range())
	}
}

// detoPosLien est un lancer apparie a l'indice, dans `pistes`, de la piste nee dans sa fenetre.
type detoPosLien struct {
	throwTS uint64
	track   int
}

func detoPositionUnFilm(t *testing.T, id string, fc *FilmContext, wr profile.Vec3Range) {
	t.Helper()
	lancers, err := ScanGrenadeThrows(fc)
	if err != nil {
		t.Fatalf("%s : lancers : %v", id, err)
	}
	pistes, err := ScanProjectiles(fc, &wr)
	if err != nil {
		t.Fatalf("%s : pistes : %v", id, err)
	}
	detos, st := ScanProjectileDetonations(fc)
	t.Logf("=== %s : %d lancers · %d pistes de projectile · detonations lues %d (reference %d, position %d)",
		id, len(lancers), len(pistes), st.Read, st.WithProjectile, st.WithPosition)
	if st.Read == 0 {
		t.Logf("    AUCUN message de detonation/impact lu sur ce film")
		return
	}
	liens := detoPosLierLancers(lancers, pistes)
	t.Logf("    lancers->piste lies (naissance a 200 ms) : %d / %d", len(liens), len(lancers))
	detoPosMesureAppariement(t, liens, pistes, detos)
}

// detoPosLierLancers apparie chaque lancer a l'indice de la piste dont la NAISSANCE est la plus
// proche dans sa fenetre (meme fenetre que grenadeBirthWindowUS, replay/grenades.go) — la
// simplification mesuree au § 2 de l'en-tete : aucun pont bipede, seul le temps depart.
func detoPosLierLancers(lancers []GrenadeThrow, pistes []types.ProjectileTrack) []detoPosLien {
	const fenetreUS = 200_000
	var out []detoPosLien
	for _, g := range lancers {
		best, bestDT := -1, uint64(math.MaxUint64)
		for i, p := range pistes {
			if len(p.Pts) == 0 {
				continue
			}
			b := p.Pts[0].TimestampUS
			if b+50_000 < g.TimestampUS || b > g.TimestampUS+fenetreUS {
				continue
			}
			dt := detoPosEcartAbs(b, g.TimestampUS)
			if best < 0 || dt < bestDT {
				best, bestDT = i, dt
			}
		}
		if best >= 0 {
			out = append(out, detoPosLien{throwTS: g.TimestampUS, track: best})
		}
	}
	return out
}

func detoPosEcartAbs(a, b uint64) uint64 {
	if a >= b {
		return a - b
	}
	return b - a
}

// detoPosMesureAppariement fait les mesures 3, 4 et 5 de l'en-tete sur les lancers lies.
func detoPosMesureAppariement(t *testing.T, liens []detoPosLien, pistes []types.ProjectileTrack, detos []types.ProjectileDetonation) {
	t.Helper()
	var avecPosition []types.ProjectileDetonation
	for _, d := range detos {
		if d.HasPosition {
			avecPosition = append(avecPosition, d)
		}
	}
	if len(avecPosition) == 0 {
		t.Logf("    AUCUNE detonation a position lue : mesure 3/4/5 sans objet")
		return
	}
	sort.Slice(avecPosition, func(i, j int) bool { return avecPosition[i].TimestampUS < avecPosition[j].TimestampUS })

	type appariement struct {
		dtUS  float64
		distU float64
		idx   int
		piste int
	}
	chercher := func(last types.ProjectileSample, decalage int) (appariement, bool) {
		bestI, bestDist := -1, math.MaxFloat64
		for i := range avecPosition {
			d := avecPosition[(i+decalage)%len(avecPosition)]
			if d.TimestampUS+500_000 < last.TimestampUS || d.TimestampUS > last.TimestampUS+4_000_000 {
				continue
			}
			dist := detoPosDist3(d.X, d.Y, d.Z, last.X, last.Y, last.Z)
			if dist < bestDist {
				bestI, bestDist = i, dist
			}
		}
		if bestI < 0 {
			return appariement{}, false
		}
		d := avecPosition[(bestI+decalage)%len(avecPosition)]
		return appariement{dtUS: float64(d.TimestampUS) - float64(last.TimestampUS), distU: bestDist, idx: bestI}, true
	}

	var dtReel, distReel, dtTemoin, distTemoin []float64
	matched, matchedTemoin := 0, 0
	basesSlot := map[string]int{"brut": 0, "+base": 0, "aucun": 0}
	for _, l := range liens {
		p := pistes[l.track]
		last := p.Pts[len(p.Pts)-1]
		if a, ok := chercher(last, 0); ok {
			matched++
			dtReel = append(dtReel, a.dtUS)
			distReel = append(distReel, a.distU)
			d := avecPosition[a.idx]
			switch {
			case d.HasProjectile && d.Projectile.Slot == p.Slot:
				basesSlot["brut"]++
			case d.HasProjectile && d.Projectile.Slot == p.Slot+detoPosDomaine5Base:
				basesSlot["+base"]++
			default:
				basesSlot["aucun"]++
			}
		}
		// TEMOIN : meme recherche, detonations decalees d'un demi-tour de liste.
		if len(avecPosition) > 1 {
			if a, ok := chercher(last, len(avecPosition)/2); ok {
				matchedTemoin++
				dtTemoin = append(dtTemoin, a.dtUS)
				distTemoin = append(distTemoin, a.distU)
			}
		}
	}
	t.Logf("    piste->detonation a portee (<=4 s apres, 0,5 s de tolerance avant, %d pistes liees) : %d (%.1f %%) · TEMOIN decale : %d (%.1f %%)",
		len(liens), matched, lot1Pct(matched, len(liens)), matchedTemoin, lot1Pct(matchedTemoin, len(liens)))
	if len(dtReel) > 0 {
		t.Logf("    ecart temps detonation - derniere position repliquee : mediane %.0f ms (min %.0f max %.0f) · TEMOIN mediane %.0f ms",
			geoMedian(dtReel)/1000, attribMin(dtReel)/1000, attribMax(dtReel)/1000, geoMedian(dtTemoin)/1000)
		t.Logf("    distance detonation <-> derniere position repliquee : mediane %.2f u (min %.2f max %.2f) · TEMOIN mediane %.2f u",
			geoMedian(distReel), attribMin(distReel), attribMax(distReel), geoMedian(distTemoin))
	}
	t.Logf("    reference de projectile (domaine 5) de la detonation appariee == slot de la piste : brute %d · +base(0x400) %d · aucune des deux %d",
		basesSlot["brut"], basesSlot["+base"], basesSlot["aucun"])
	t.Logf("    LECTURE : appariement net (bien superieur au TEMOIN) ET ecart temps POSITIF (la detonation suit la derniere")
	t.Logf("    position repliquee, elle ne la precede pas) = la position de detonation est fiable et AFFINE le repli actuel.")
}

func detoPosDist3(ax, ay, az, bx, by, bz float32) float64 {
	dx, dy, dz := float64(bx-ax), float64(by-ay), float64(bz-az)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}
