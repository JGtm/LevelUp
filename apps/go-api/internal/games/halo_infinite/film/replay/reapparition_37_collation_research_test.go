//go:build research

package replay

// reapparition_37_collation_research_test.go — LOT 3.7, VOIE LIBRE : LA COLLATION.
//
// # CE QU IL FAUT COLLER, ET POURQUOI
//
// L instrument frere (`grammar/reapparition_37_bassin_film_research_test.go`) lit le BASSIN de
// minuteurs du moteur de jeu et rend, image-cle par image-cle, la duree TOTALE et le temps
// RESTANT de chaque fente. Il ne peut pas NOMMER ces minuteurs : une fente est un numero. Le
// nom vient de la COLLATION avec ce que le rejeu publie deja et qui est DATE :
//
//	flagCarries[].spans   les etats du drapeau, `dropped` compris, bornes en frames
//	vehicles[]            T0 (naissance, record de creation date a la ms), TEnd (dead-state
//	                      `ti=40 i11`, date a la ms), Spawn (position monde)
//	deaths / tracks       le reste du document, disponible si besoin
//
// Cet instrument CUIT le film par `BuildFromFilm` — donc par le chemin de production, jamais par
// une recopie — et imprime ces trois canaux avec leurs instants. La collation elle-meme se fait
// dans la note : deux instruments, une seule conclusion ecrite, chacun rejouable.
//
// # ET IL REPOND A LA QUATRIEME MESURE, CELLE DES VEHICULES
//
// `V2_SPAWNS_COOLDOWNS_2026-09-01` § 3 a conclu « cooldown NON mesurable » parce que la fin de
// vie d un vehicule etait bornee a +/-20 s par le recensement des images-cles. Le dead-state est
// lu depuis le 2026-09-05 et le rejeu publie `TEnd` DATE A LA MILLISECONDE. Le cycle est donc
// mesurable : cet instrument agglomere les naissances par EMPLACEMENT (les pads mesures a
// 0,00 m de rayon) et rend, par emplacement, les ecarts `TEnd -> naissance suivante`, avec les
// regles de stabilite de `PadCycle` (au moins deux ecarts pour parler d un cycle, et le compte
// des reapparitions dont la disparition precedente n est PAS datee).
//
// # REGIME
//
//	REAP_FILM_ROOT=<repo>/data/cache/film_chunks \
//	REAP_COLLATION="bcb6d393:Cliffhanger,fb1a1a72:Banished Narrows" \
//	  go test -tags=research ./internal/games/halo_infinite/film/replay/ \
//	    -run Reapparition37Collation -v -timeout 90m
//
// La carte est donnee EN DONNEE (le catalogue de bornes se cherche par son nom) : ce paquet ne
// resout pas un nom de carte, et surtout il n ouvre AUCUNE base. Un film a la fois.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/testutil"
)

// reap37PasAmas est le rayon d agglomeration des naissances de vehicule, en metres. Les
// naissances sont EXACTES (rayon d amas mesure 0,00 m, `V2_SPAWNS_COOLDOWNS` § 1.2) : deux
// metres laissent la place a un pad decale sans jamais fusionner deux pads voisins, mesures a
// 3,2 m au plus proche sur Behemoth.
const reap37PasAmas = 2.0

// reap37CycleMin est le nombre d ecarts en dessous duquel un cycle n est PAS etabli. Meme regle
// que `PadCycle.Gaps` : un ecart unique n a pas de dispersion et rien ne dit qu il se repete.
const reap37CycleMin = 2

func TestReapparition37Collation(t *testing.T) {
	racine := os.Getenv("REAP_FILM_ROOT")
	liste := os.Getenv("REAP_COLLATION")
	if racine == "" || liste == "" {
		t.Skip("REAP_FILM_ROOT et REAP_COLLATION requis — aucun film ouvert sans eux")
	}
	cat := reap37Catalogue(t)
	for _, spec := range strings.Split(liste, ",") {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}
		court, carte, ok := strings.Cut(spec, ":")
		if !ok {
			t.Fatalf("REAP_COLLATION : %q n est pas `court:carte`", spec)
		}
		t.Run(court, func(t *testing.T) { reap37UnFilmCollation(t, cat, racine, court, carte) })
	}
}

func reap37Catalogue(t *testing.T) *profile.MapQuantCatalog {
	t.Helper()
	// `testutil.RepoRoot` et non `title.FindRepoRoot` : le helper de PRODUCTION laisserait ce
	// test se skipper en silence en CI (ratchet `no_repo_root_walk_test.go`). Ici on echoue.
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	cat, err := profile.LoadMapQuantCatalog(title.NewPathResolver(root).MapQuantBoundsPath(title.DefaultSlug))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	return cat
}

func reap37UnFilmCollation(t *testing.T, cat *profile.MapQuantCatalog, racine, court, carte string) {
	t.Helper()
	entry, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("carte %q absente du catalogue de bornes : %v", carte, err)
	}
	film, err := source.LoadDir(filepath.Join(racine, court), nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", court, err)
	}
	doc, err := BuildFromFilm("lot-3.7-collation", "halo_infinite", film, Options{MapQuant: &entry})
	if err != nil {
		t.Fatalf("cuisson de %s : %v", court, err)
	}
	t.Logf("FILM %s (%s) — schema %d, %d frame(s), pas %d ms, duree %d ms",
		court, carte, doc.SchemaVersion, doc.FrameCount, doc.FrameIntervalMS, doc.DurationMS)
	reap37PublierDrapeaux(t, doc)
	reap37PublierVehicules(t, doc)
}

// reap37PublierDrapeaux imprime les etats du drapeau avec leurs instants EN SECONDES, pour que
// la collation avec le bassin se lise sans conversion mentale.
func reap37PublierDrapeaux(t *testing.T, doc ReplayDocument) {
	t.Helper()
	if len(doc.FlagCarries) == 0 {
		t.Logf("  DRAPEAUX : aucun (`flagCarries` absent — ce film n en publie pas)")
		return
	}
	for _, fc := range doc.FlagCarries {
		t.Logf("  DRAPEAU equipe %d — %d intervalle(s)", fc.Team, len(fc.Spans))
		for _, sp := range fc.Spans {
			if sp.State != FlagStateDropped {
				continue
			}
			t.Logf("      LACHE  t0=%8.1f s  t1=%8.1f s  (duree %6.1f s)  frames [%d, %d]",
				reap37Sec(doc, sp.T0), reap37Sec(doc, sp.T1),
				reap37Sec(doc, sp.T1)-reap37Sec(doc, sp.T0), sp.T0, sp.T1)
		}
		for _, sp := range fc.Spans {
			if sp.State == FlagStateDropped {
				continue
			}
			t.Logf("      %-12s t0=%8.1f s  t1=%8.1f s", sp.State,
				reap37Sec(doc, sp.T0), reap37Sec(doc, sp.T1))
		}
	}
}

// reap37PublierVehicules imprime les vies de vehicule, puis le CYCLE par emplacement.
func reap37PublierVehicules(t *testing.T, doc ReplayDocument) {
	t.Helper()
	if len(doc.Vehicles) == 0 {
		t.Logf("  VEHICULES : aucun")
		return
	}
	avecFin, avecSpawn := 0, 0
	for _, v := range doc.Vehicles {
		if v.TEnd != nil {
			avecFin++
		}
		if v.Spawn != nil {
			avecSpawn++
		}
	}
	t.Logf("  VEHICULES : %d vie(s), %d avec naissance situee, %d avec FIN DATEE (dead-state)",
		len(doc.Vehicles), avecSpawn, avecFin)
	reap37PublierCycles(t, doc)
}

// reap37Emplacement est un pad de naissance de vehicule, avec les vies qui y sont nees.
type reap37Emplacement struct {
	X, Y, Z float32
	Vies    []VehicleTrack
}

// reap37PublierCycles agglomere les naissances en emplacements et rend, par emplacement, les
// ecarts `fin datee -> naissance suivante`.
//
// LA REGLE EST CELLE DE `PadCycle`, ET ELLE EST STRICTE : un ecart ne compte que si la vie
// PRECEDENTE porte une fin DATEE (`TEnd`). Les reapparitions dont la disparition precedente
// n est pas datee sont comptees A PART (`manques`) — c est l autre moitie du denominateur, et
// c est le correctif de revue du 2026-08-17 applique ici d avance.
func reap37PublierCycles(t *testing.T, doc ReplayDocument) {
	t.Helper()
	empl := reap37Agglomerer(doc.Vehicles)
	t.Logf("  CYCLE DE REAPPARITION — %d emplacement(s) agglomere(s) a %.1f m",
		len(empl), reap37PasAmas)
	for _, e := range empl {
		vies := e.Vies
		sort.Slice(vies, func(a, b int) bool { return vies[a].T0 < vies[b].T0 })
		var ecarts []float64
		manques := 0
		for i := 1; i < len(vies); i++ {
			if vies[i-1].TEnd == nil {
				manques++
				continue
			}
			d := reap37Sec(doc, vies[i].T0) - reap37Sec(doc, *vies[i-1].TEnd)
			if d <= 0 {
				manques++ // vies qui se chevauchent : l ecart n a pas de sens
				continue
			}
			ecarts = append(ecarts, d)
		}
		sort.Float64s(ecarts)
		etat := fmt.Sprintf("%d ecart(s), %d manque(s)", len(ecarts), manques)
		if len(ecarts) >= reap37CycleMin {
			etat = fmt.Sprintf("CYCLE ETABLI : mediane %.1f s  p10 %.1f s  p90 %.1f s  (%s)",
				reap37Quantile(ecarts, 0.5), reap37Quantile(ecarts, 0.1),
				reap37Quantile(ecarts, 0.9), etat)
		} else {
			etat = "cycle NON etabli (" + etat + ")"
		}
		t.Logf("      (%8.1f, %8.1f, %6.1f) m : %2d vie(s) — %s", e.X, e.Y, e.Z, len(vies), etat)
		if len(ecarts) > 0 {
			t.Logf("           ecarts : %s", reap37Ecarts(ecarts))
		}
	}
}

// reap37Agglomerer regroupe les vies par position de naissance, algorithme du meneur, points
// TRIES : deterministe.
func reap37Agglomerer(vies []VehicleTrack) []reap37Emplacement {
	avec := make([]VehicleTrack, 0, len(vies))
	for _, v := range vies {
		if v.Spawn != nil {
			avec = append(avec, v)
		}
	}
	sort.Slice(avec, func(a, b int) bool {
		if avec[a].Spawn.X != avec[b].Spawn.X {
			return avec[a].Spawn.X < avec[b].Spawn.X
		}
		return avec[a].Spawn.Y < avec[b].Spawn.Y
	})
	var out []reap37Emplacement
	for _, v := range avec {
		place := -1
		for i := range out {
			if reap37Dist(out[i].X, out[i].Y, v.Spawn.X, v.Spawn.Y) <= reap37PasAmas {
				place = i
				break
			}
		}
		if place < 0 {
			out = append(out, reap37Emplacement{X: v.Spawn.X, Y: v.Spawn.Y, Z: v.Spawn.Z})
			place = len(out) - 1
		}
		out[place].Vies = append(out[place].Vies, v)
	}
	sort.Slice(out, func(a, b int) bool { return len(out[a].Vies) > len(out[b].Vies) })
	return out
}

// reap37Dist rend la distance EUCLIDIENNE en metres, dans le plan. Pas la distance au carre :
// elle se compare directement a `reap37PasAmas`, et un seuil qu il faut mettre au carre au site
// d appel est un piege.
func reap37Dist(x1, y1, x2, y2 float32) float64 {
	dx, dy := float64(x1-x2), float64(y1-y2)
	return math.Sqrt(dx*dx + dy*dy)
}

// reap37Sec convertit une frame du document en secondes de match, par le PAS REEL que le
// document publie (`frameIntervalMs`). Sans pas publie, la frame sort telle quelle et le log le
// dit : inventer une cadence ferait mentir toute la collation.
func reap37Sec(doc ReplayDocument, frame int) float64 {
	if doc.FrameIntervalMS <= 0 {
		return float64(frame)
	}
	return float64(frame) * float64(doc.FrameIntervalMS) / 1000
}

func reap37Quantile(tri []float64, q float64) float64 {
	if len(tri) == 0 {
		return 0
	}
	i := int(q * float64(len(tri)-1))
	return tri[i]
}

func reap37Ecarts(e []float64) string {
	parts := make([]string, 0, len(e))
	for _, x := range e {
		parts = append(parts, fmt.Sprintf("%.1f", x))
	}
	return strings.Join(parts, " · ")
}
