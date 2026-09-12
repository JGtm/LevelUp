package replay

// grenade_ecart_research_test.go — BANC DE MESURE : un lancer de grenade est-il posé sur SON
// lanceur ?
//
// LA QUESTION. `locateThrow` situe un lancer par la naissance du projectile la plus proche DANS
// LE TEMPS (fenêtre ±200 ms). Deux joueurs qui lancent dans la même fenêtre — banal — et rien,
// dans ce choix, ne dit lequel des deux projectiles appartient à quel lancer : le départage
// vient du tri, donc de X. Si le défaut est réel, la position publiée pour un lancer n'est pas
// à portée de main de son auteur.
//
// LA GRANDEUR MESURÉE, ET POURQUOI ELLE TRANCHE. Le lancer PORTE son auteur (`FilmIndex`, écrit
// par le film). Le pont d'identité donne le corps de cet auteur à cet instant. La distance entre
// la position PUBLIÉE du lancer et ce corps est donc une erreur mesurable, sans rien supposer :
// un lancer sort de la main de son lanceur. La mesure qui fonde la source projectile donne
// 0,77 unité entre une naissance et le biped de son auteur ; une médiane de l'ordre du mètre ou
// plus est le régime du contrôle négatif, pas celui du signal.
//
// CE QUE LE BANC PUBLIE, par film :
//
//	M1 population   lancers publiés, part situés par PROJECTILE, part dont l'auteur est résolu
//	                (le dénominateur : un lancer sans auteur résolu n'est pas mesurable).
//	M2 écart        médiane, part > `grenadeAuthorRadiusM`, pire cas de la distance lancer ->
//	                lanceur, sur la seule branche PROJECTILE.
//	M3 ambiguïté    nombre de fenêtres ±200 ms portant DEUX naissances ou plus — la population
//	                exacte où le départage par le temps ne décide rien.
//	M4 témoin       le même écart sur la branche BIPED, qui lit la position de l'auteur : il
//	                doit valoir zéro. S'il ne le vaut pas, c'est la mesure qui est fausse, pas
//	                le décodeur.
//
// CE QU'IL NE MESURE PAS : les lancers dont l'auteur n'est pas résolu par le pont. Ils sont
// COMPTÉS (M1) et exclus de M2 — les inclure mélangerait un défaut d'attribution avec un trou
// du pont d'identité, qui est un autre chantier.
//
// LE PONT D'IDENTITÉ EST RECONSTRUIT ICI à partir des lectures que l'observateur rend
// (positions, créations, morts, index de joueur, tirs), avec les MÊMES entrées que `build.go`
// hormis celles qui viennent de la BASE (bots, roster, tableau, statborg) : ce banc est hors
// ligne, comme `cmd/replay-build` sans faits. La conséquence est écrite dans M1 — la part
// d'auteurs résolus y est un peu plus basse qu'en production.
//
// USAGE (un film par process, verrou de décodage pris par BuildFromFilm) :
//
//	CGO_ENABLED=0 \
//	GRENADE_ECART_PARC=<racine portant data/> \
//	GRENADE_ECART_FILMS='000d5950=Cliffhanger,0797ce72=Live Fire,21ece4d8=Live Fire' \
//	  go test ./internal/games/halo_infinite/film/replay -run TestBancEcartLancerLanceur -v -timeout 1800s

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

const (
	grenEcartParcEnv  = "GRENADE_ECART_PARC"
	grenEcartFilmsEnv = "GRENADE_ECART_FILMS"
)

// grenEcartFilm : un film du banc et la carte qui porte ses bornes de déquantification.
type grenEcartFilm struct{ id, carte string }

// grenEcartCas : un lancer publié, avec ce qu'il faut pour le juger.
type grenEcartCas struct {
	src        string
	auteurVu   bool
	distance   float64
	candidates int
}

func TestBancEcartLancerLanceur(t *testing.T) {
	parc := os.Getenv(grenEcartParcEnv)
	films := grenEcartFilmsDeEnv(os.Getenv(grenEcartFilmsEnv))
	if parc == "" || len(films) == 0 {
		t.Skipf("banc désactivé : %s et %s requis", grenEcartParcEnv, grenEcartFilmsEnv)
	}
	for _, f := range films {
		t.Run(f.id, func(t *testing.T) { grenEcartMesureFilm(t, parc, f) })
	}
}

// grenEcartFilmsDeEnv lit « id=Carte,id=Carte » — la carte porte des espaces (« Live Fire »),
// donc le séparateur de champ est la virgule et celui de paire le signe égal.
func grenEcartFilmsDeEnv(s string) []grenEcartFilm {
	var out []grenEcartFilm
	for _, part := range strings.Split(s, ",") {
		id, carte, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok || id == "" || carte == "" {
			continue
		}
		out = append(out, grenEcartFilm{id: id, carte: carte})
	}
	return out
}

// grenEcartMesureFilm cuit un film et publie les quatre mesures.
func grenEcartMesureFilm(t *testing.T, parc string, f grenEcartFilm) {
	t.Helper()
	entry := grenEcartBornes(t, parc, f.carte)
	lu := &grenEcartLectures{}
	dir := filepath.Join(parc, "data", "cache", "film_chunks", f.id)
	doc, err := buildFromFilmDir(f.id, "halo_infinite", dir, Options{
		MapQuant: &entry,
		Observe:  lu.observe,
	})
	if err != nil {
		t.Fatalf("cuisson de %s : %v", f.id, err)
	}
	cas := grenEcartCas2(doc, lu)
	grenEcartRapport(t, f, doc, lu, cas)
}

// grenEcartLectures collecte les lectures brutes que l'observateur rend — celles dont le pont
// d'identité a besoin, et les naissances de projectile.
type grenEcartLectures struct {
	positions []filmdec.BipedPosition
	creations []filmdec.BipedCreation
	deaths    []Death
	indices   PlayerIndexTable
	fire      []filmdec.FireEvent
	throws    []filmdec.GrenadeThrow
	proj      []filmdec.ProjectileTrack
}

func (l *grenEcartLectures) observe(step string, v any) {
	// LES NOMS D'ETAPE SONT CEUX DU PAQUET (cf. equivalence_minifilm_test.go) : le paquet ne
	// garde qu'UNE ecriture de chacun, celle des litteraux `opt.observe("...")` de build.go.
	switch step {
	case etapePositions:
		l.positions, _ = v.([]filmdec.BipedPosition)
	case etapeCreationsBipede:
		l.creations, _ = v.([]filmdec.BipedCreation)
	case etapeMorts:
		l.deaths, _ = v.([]Death)
	case etapeIndicesJoueur:
		l.indices, _ = v.(PlayerIndexTable)
	case etapeFire:
		l.fire, _ = v.([]filmdec.FireEvent)
	case etapeGrenades:
		l.throws, _ = v.([]filmdec.GrenadeThrow)
	case etapeProjectiles:
		l.proj, _ = v.([]filmdec.ProjectileTrack)
	}
}

// grenEcartCas2 rejoue le rattachement sur les lectures brutes et rend un cas par lancer
// PUBLIÉ. Il ne relit pas le document : la position publiée y est arrondie au centimètre, ce
// qui est sans effet à l'échelle du mètre, mais le SLOT du lanceur n'y est pas toujours (c'est
// le défaut que le banc mesure).
func grenEcartCas2(doc ReplayDocument, l *grenEcartLectures) []grenEcartCas {
	if len(l.positions) == 0 || len(l.throws) == 0 {
		return nil
	}
	sorted := append([]filmdec.BipedPosition(nil), l.positions...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TimestampUS < sorted[j].TimestampUS })
	origin := sorted[0].TimestampUS
	step := uint64(doc.FrameIntervalMS) * 1000
	reg := BuildIdentityRegistry(IdentityInput{
		Positions: sorted, BipedCreations: l.creations,
		Deaths: l.deaths, PlayerIndices: l.indices, Fire: fireRefs(l.fire),
		Clock:   IdentityClock{OriginUS: origin, StepUS: step, FrameCount: doc.FrameCount},
		MatchID: doc.MatchID,
	})
	owner := reg.IndexParSlot()
	tracks := indexBySlot(sorted)
	births := projectileBirths(l.proj)
	var out []grenEcartCas
	for _, g := range l.throws {
		if _, known := g.Rank(); !known {
			continue
		}
		gr, _, ok := locateThrow(g, births, tracks, owner)
		if !ok {
			continue
		}
		c := grenEcartCas{src: gr.Src, candidates: len(birthsInWindow(births, g.TimestampUS))}
		if p, d := grenEcartAuteur(g, tracks, owner); d {
			c.auteurVu = true
			c.distance = planDist(gr.X, gr.Y, p.X, p.Y)
		}
		out = append(out, c)
	}
	return out
}

// grenEcartAuteur rend la position répliquée du lanceur à l'instant du lancer, quand le pont et
// le film la donnent tous les deux.
func grenEcartAuteur(g filmdec.GrenadeThrow, tracks map[uint32]slotTrack,
	owner map[uint32]int) (filmdec.BipedPosition, bool) {
	slot, reason := slotFor(tracks, owner, g.FilmIndex, g.TimestampUS)
	if reason != reasonAttached {
		return filmdec.BipedPosition{}, false
	}
	p, d := tracks[slot].at(g.TimestampUS)
	if d > shotPosToleranceUS || !p.HasWorld {
		return filmdec.BipedPosition{}, false
	}
	return p, true
}

// grenEcartRapport publie les quatre mesures. Le banc ne juge RIEN : il chiffre, et le verdict
// s'écrit dans le rapport du lot.
func grenEcartRapport(t *testing.T, f grenEcartFilm, doc ReplayDocument,
	l *grenEcartLectures, cas []grenEcartCas) {
	t.Helper()
	var parProj, auteurs, ambigus int
	var ecarts, ecartsBiped []float64
	for _, c := range cas {
		if c.candidates > 1 {
			ambigus++
		}
		if c.auteurVu {
			auteurs++
		}
		if c.src == GrenadeSrcProjectile {
			parProj++
			if c.auteurVu {
				ecarts = append(ecarts, c.distance)
			}
			continue
		}
		if c.auteurVu {
			ecartsBiped = append(ecartsBiped, c.distance)
		}
	}
	t.Logf("FILM %s (%s) : %d lancers décodés, %d situés, %d publiés ; %d trajectoires de projectile",
		f.id, f.carte, len(l.throws), len(cas), len(doc.Grenades), len(l.proj))
	t.Logf("M1 population : %d par PROJECTILE (%s), auteur résolu pour %d (%s)",
		parProj, grenEcartPct(parProj, len(cas)), auteurs, grenEcartPct(auteurs, len(cas)))
	t.Logf("M3 ambiguïté : %d lancers dans une fenêtre à 2 naissances ou plus (%s)",
		ambigus, grenEcartPct(ambigus, len(cas)))
	grenEcartDistances(t, "M2 écart lancer -> lanceur (branche PROJECTILE)", ecarts)
	grenEcartDistances(t, "M4 témoin (branche BIPED, doit valoir zéro)", ecartsBiped)
}

// grenEcartDistances publie médiane, part au-delà du rayon d'auteur et pire cas.
func grenEcartDistances(t *testing.T, titre string, d []float64) {
	t.Helper()
	if len(d) == 0 {
		t.Logf("%s : aucune mesure", titre)
		return
	}
	sort.Float64s(d)
	loin := 0
	for _, v := range d {
		if v > grenadeAuthorRadiusM {
			loin++
		}
	}
	t.Logf("%s : n=%d médiane=%.2f m p90=%.2f m max=%.2f m ; > %d m : %d (%s)",
		titre, len(d), d[len(d)/2], d[9*(len(d)-1)/10], d[len(d)-1],
		grenadeAuthorRadiusM, loin, grenEcartPct(loin, len(d)))
}

func grenEcartPct(n, total int) string {
	if total == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f %%", 100*float64(n)/float64(total))
}

// grenEcartBornes charge les bornes de déquantification de la carte depuis le catalogue
// VERSIONNÉ du parc désigné.
func grenEcartBornes(t *testing.T, parc, carte string) filmdec.MapQuantEntry {
	t.Helper()
	chemin := filepath.Join(parc, "data", "titles", "halo_infinite", "reference", "map_quant_bounds.json")
	cat, err := filmdec.LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatalf("catalogue de bornes %s : %v", chemin, err)
	}
	entry, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("carte %q absente du catalogue : %v", carte, err)
	}
	return entry
}
