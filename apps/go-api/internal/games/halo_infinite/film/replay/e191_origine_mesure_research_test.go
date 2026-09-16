package replay

// e191_origine_mesure_research_test.go — LOT 1.9.1 : LA MESURE AVANT DE CODER (le contexte).
//
// # CE QU ELLE ETABLIT, ET POURQUOI ELLE PRECEDE TOUTE LIGNE DE PRODUCTION
//
// L origine d une pose d equipement (`deployed` / `dropped` / `unknown`) se decide aujourd hui
// par DEUX regles, dans cet ordre : le manifeste (`kind = "deployed"` -> `deployed` sans
// mesure, item H.2) puis une FENETRE TEMPORELLE de 200 ms entre la creation de l objet et la
// fin de la vie de son poseur (`equipmentOrigin`). D13 du plan dit que ce que le film ECRIT
// decide, et qu une correlation temporelle ne decide jamais a sa place. Trois signaux ECRITS
// sont candidats, et cette mesure dit ce que chacun couvre AVANT qu on ne branche quoi que ce
// soit :
//
//  1. l evenement de liste type 103 `EquipmentSpawnedObject` — « une PIECE a ete engendree » —
//     dont la deuxieme reference designe la vie de l objet cree
//     ([filmdec.ScanEquipmentSpawnEvents]) ;
//  2. la MORT ECRITE du poseur : une vie du slot poseur que le fil des morts ferme
//     (`cause == CauseVieMort`, lien d identite des lots 1.6 et 1.8) ;
//  3. la PRISE ECRITE du poseur : une emission `taken` d `equipmentChanges` sur le slot poseur
//     — le joueur ramasse un autre equipement, donc lache celui qu il portait (rapport E0,
//     question 5 : une pose `deployed` sur un objet PORTE mesure un lacher volontaire a mi-vie).
//
// # CE QU ELLE NE FAIT PAS
//
// Elle ne cuit aucun artefact, n ecrit rien, ne touche pas au parc. Elle DECODE les films par le
// meme etage de balayage que la production (`decodeFilmInputsForEntry`) et compare, pose par
// pose, ce que chaque signal dit. LA TOLERANCE D APPARIEMENT N EST PAS POSEE : la sortie donne
// la DISTRIBUTION des ecarts en millisecondes, et c est elle qui la fixera.
//
// Les tableaux vivent dans `e191_origine_rapport_research_test.go`.
//
// LECTURE SEULE, skip par defaut.
//
//	CGO_ENABLED=0 E191_ROOT=<depot>/data/cache/film_chunks \
//	  go test ./internal/games/halo_infinite/film/replay/ \
//	  -run '^TestE191OrigineMesure$' -count=1 -timeout 180m -v
//
// DEUX FILMS DE LA TABLE N ONT PAS DE FEUILLE VERSIONNEE : `0797ce72` et `4f77afc1` ne sont pas
// au corpus d equivalence, donc `testdata/equivalence/<id>.facts.json` n existe pas pour eux et
// `decodeFilmInputsForEntry` les ecarte en le DISANT (« hors mesure »). Les exporter avant la
// mesure, et les retirer apres — ce sont des donnees de parc, pas des references :
//
//	CGO_ENABLED=1 LEVELUP_REPO_ROOT=<parc> go run ./cmd/levelup replay-facts-export //	  --out internal/games/halo_infinite/film/replay/testdata/equivalence 0797ce72 4f77afc1

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

const (
	e191RootEnv = "E191_ROOT" // racine des chunks de film
	e191IDsEnv  = "E191_IDS"  // sous-ensemble de films, separes par des virgules (defaut : tous)
)

// e191SansSignal est la valeur d ecart rendue quand le signal n existe pas du tout pour ce
// poseur. Un grand nombre PLUTOT qu un NaN : les tris et les comparaisons restent totales, et
// aucune ligne ne disparait d un histogramme par accident.
const e191SansSignal = 1e12

// e191Film est un film de la mesure : son identifiant court, sa carte et son build.
//
// LES CARTES NE SE DEVINENT PAS. Les huit premieres viennent de `goldenBuilds()` (la table des
// builds du depot) ; `50247b26`, `51101d1d` et `d9781168` de `testdata/equivalence/CORPUS.txt` ;
// `0797ce72` et `4f77afc1` de `levelup replay-facts-export` joue sur le parc le 2026-09-15
// (« cartes [Live Fire] » et « cartes [Flood Gulch] »). Une carte fausse ne fausse pas seulement
// des metres : elle change le decoupage d i0 IMPOSE, donc les records acceptes.
type e191Film struct {
	Short8, Carte, Build string
}

// e191Films : les 13 films de la mesure — les 8 builds, l echantillon court du regime de gate,
// et les deux temoins d equipement demandes par le lot (`0797ce72`, `4f77afc1`).
func e191Films() []e191Film {
	return []e191Film{
		{"000d5950", "Cliffhanger", "HI_1_13_0"},
		{"a521164d", "Fragmentation Heavies", "HI_1_4_1"},
		{"60ae07c4", "Live Fire - Ranked", "HI_1_8_0"},
		{"11de8353", "Thunderhead", "HI_1_9_0"},
		{"111fa685", "Command", "HI_1_10_0"},
		{"e5adf7b2", "Fragmentation", "HI_1_11_0"},
		{"bcb6d393", "Cliffhanger", "HI_1_12_0"},
		{"fb1a1a72", "Banished Narrows", "HI_1_13_0"},
		{"50247b26", "Oasis", "v31 sans section"},
		{"51101d1d", "Fortress", "HI_1_13_0"},
		{"d9781168", "Dredge", "HI_1_13_0"},
		{"0797ce72", "Live Fire", "HI_1_13_0"},
		{"4f77afc1", "Flood Gulch", "HI_1_13_0"},
	}
}

// e191Pose est UNE pose, avec tout ce que les trois signaux en disent.
type e191Pose struct {
	Film, Famille, ID string
	// Origine est ce que la production PUBLIE aujourd hui ; Regle est ce qui la decide :
	// `manifeste` (piece engendree, H.2), `fenetre` (les 200 ms), `sans_poseur` (aucun bipede
	// contemporain a moins de 3 m).
	Origine, Regle string
	// SpawnDtMS est l ecart SIGNE, en ms, entre la creation de l objet et l evenement 103 dont
	// la reference porte la MEME cle de vie (slot, generation) et qui en est le plus proche.
	// [e191SansSignal] quand aucun evenement ne porte cette cle.
	//
	// UNE CLE NE SUFFIT PAS, ET C EST LA MESURE QUI LE DIT : la generation ne fait que 2 bits,
	// donc un slot repasse par la MEME paire plusieurs fois dans un match. Un appariement par
	// cle seule fait « designer » 83 poses par 3 evenements (mesure du 2026-09-15 sur
	// `d9781168`). L ecart de temps est donc la seconde moitie de la cle, et sa distribution
	// est ce que ce champ existe pour rendre.
	SpawnDtMS float64
	// Portee : le manifeste declare cet objet `kind = "carried"` — un appareil PORTE.
	Portee     bool
	Slot       uint32
	AvecPoseur bool
	// MortMS / PriseMS : ecart ABSOLU, en ms, au signal ECRIT le plus proche du poseur.
	// [e191SansSignal] quand le signal n existe pas.
	MortMS, PriseMS float64
	// FenetreMS est l ecart que la fenetre de 200 ms mesure aujourd hui — le CONTROLE.
	FenetreMS float64
	// OrigineF1Avant est l origine que rendait la regle D AVANT l item F.1 du 2026-09-13 — la
	// fenetre temporelle DOUBLEE d une clause de distance (`f1OrigineAvant`, le temoin conserve
	// par `f1_origine_mesure_research_test.go`). Elle sert a RETROUVER les 22 poses que F.1 a
	// requalifiees, et a les rejuger une a une par la grammaire.
	OrigineF1Avant string
	// DistM est la distance, en metres, entre la pose et la fin de la vie de poseur retenue —
	// la grandeur sur laquelle portait la clause retiree par F.1.
	DistM float64
}

func TestE191OrigineMesure(t *testing.T) {
	root := os.Getenv(e191RootEnv)
	if root == "" {
		t.Skipf("mesure 1.9.1 : definir %s (racine des chunks de film, lecture seule)", e191RootEnv)
	}
	films := e191FilmsDemandes()
	var toutes []e191Pose
	mesures := 0
	for _, f := range films {
		poses, ok := e191MesureUnFilm(t, root, f)
		if !ok {
			continue
		}
		mesures++
		toutes = append(toutes, poses...)
	}
	t.Logf("")
	t.Logf("######## PARC — %d films demandes, %d mesures, %d poses ########",
		len(films), mesures, len(toutes))
	e191Rapport(t, toutes)
}

// e191FilmsDemandes filtre la table par E191_IDS quand la variable est posee.
func e191FilmsDemandes() []e191Film {
	spec := os.Getenv(e191IDsEnv)
	if spec == "" {
		return e191Films()
	}
	garde := map[string]bool{}
	for _, id := range strings.Split(spec, ",") {
		if id = strings.TrimSpace(id); id != "" {
			garde[id] = true
		}
	}
	var out []e191Film
	for _, f := range e191Films() {
		if garde[f.Short8] {
			out = append(out, f)
		}
	}
	return out
}

// e191MesureUnFilm decode un film et rend ses poses annotees.
func e191MesureUnFilm(t *testing.T, root string, f e191Film) ([]e191Pose, bool) {
	t.Helper()
	entry, err := (goldenBuild{Short8: f.Short8, Map: f.Carte}).mapQuant()
	if err != nil {
		t.Logf("film %s : carte %q hors catalogue (%v) — hors mesure", f.Short8, f.Carte, err)
		return nil, false
	}
	dir := filepath.Join(root, f.Short8)
	g, err := decodeFilmInputsForEntry(f.Short8, dir, entry)
	if err != nil {
		t.Logf("film %s : balayage impossible (%v) — hors mesure", f.Short8, err)
		return nil, false
	}
	spawns, sst, err := e191Spawns(dir)
	if err != nil {
		t.Logf("film %s : evenements 103 illisibles (%v) — hors mesure", f.Short8, err)
		return nil, false
	}
	designees := map[filmdec.EquipmentLifeKey][]uint64{}
	for _, e := range spawns {
		if e.SpawnedValid {
			designees[e.Spawned] = append(designees[e.Spawned], e.TimestampUS)
		}
	}
	ctx := e191Contexte(g)
	ctx.familles = goldenCatalog(t).EquipmentFamilies
	t.Logf("")
	t.Logf("######## FILM %s (%s, carte %s) — %d poses ########", f.Short8, f.Build, f.Carte,
		len(g.Placements))
	t.Logf("  balayage 103 : %d chunks, %d paquets delta, %d listes non vides, %d evenements, "+
		"ref0 %d, ref1 %d, ref2 %d, %d vies distinctes designees",
		sst.Chunks, sst.Packets, sst.Lists, sst.Events, sst.WithSource, sst.WithSpawned,
		sst.Ref2, len(designees))
	t.Logf("  signaux ECRITS du film : %d morts appariees a une vie (%d slots), "+
		"%d prises `taken` (%d slots), %d changements d equipement",
		ctx.nbMorts, len(ctx.morts), ctx.nbPrises, len(ctx.prises), len(g.EquipmentChanges))

	portes := e191ObjetsPortes(t)
	out := make([]e191Pose, 0, len(g.Placements))
	for _, p := range g.Placements {
		out = append(out, e191UnePose(p, f.Short8, designees, portes, ctx))
	}
	e191Rapport(t, out)
	return out, true
}

// e191Spawns balaie les evenements 103 d un film, sous le verrou de decodage.
//
// LE CHARGEMENT EST ICI, ET C EST VOULU (revue de jalon M1, constat C4, 2026-09-16). Il vivait
// dans `filmdec.ScanFilmEquipmentSpawnEvents`, une enveloppe `dir` de PRODUCTION dont cet appel
// etait le SEUL au depot — instrument par instrument, tests compris. Regle 7 du depot (« 0 code
// mort ») : l enveloppe est supprimee, ses deux lignes vivent chez son unique appelant, et la
// production garde la seule forme qu elle emploie (`ScanEquipmentSpawnEvents(fc)`, sur un
// contexte deja ouvert, qui ne recharge rien).
func e191Spawns(dir string) ([]filmdec.EquipmentSpawnEvent, filmdec.EquipmentSpawnStats, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, filmdec.EquipmentSpawnStats{}, err
	}
	return filmdec.ScanEquipmentSpawnEvents(filmdec.NewFilmContext(film))
}

// e191Ctx porte ce qu un film donne a la mesure : le nuage trie, les vies de poseur, et les deux
// signaux ECRITS indexes par slot.
type e191Ctx struct {
	positions         []filmdec.BipedPosition
	vies              map[uint32][]equipLife
	familles          map[uint32]string
	morts, prises     map[uint32][]uint64
	nbMorts, nbPrises int
}

// e191Contexte assemble le contexte d un film : les vies de poseur (comme la production), les
// MORTS ECRITES par slot (via le registre d identite, seul producteur de liens) et les PRISES
// ECRITES par slot (`equipmentChanges` `taken`).
func e191Contexte(g *goldenInputs) e191Ctx {
	sorted := append([]filmdec.BipedPosition(nil), g.Positions...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].TimestampUS < sorted[j].TimestampUS
	})
	ctx := e191Ctx{
		positions: sorted, vies: equipmentLives(sorted),
		morts: map[uint32][]uint64{}, prises: map[uint32][]uint64{},
	}
	for _, v := range e191ViesDuRegistre(g, sorted) {
		if v.cause != CauseVieMort {
			continue
		}
		ctx.morts[v.slot] = append(ctx.morts[v.slot], uint64(v.to))
		ctx.nbMorts++
	}
	for _, c := range g.EquipmentChanges {
		if c.Kind != filmdec.EquipmentTaken {
			continue
		}
		ctx.prises[c.Slot] = append(ctx.prises[c.Slot], c.TimestampUS)
		ctx.nbPrises++
	}
	return ctx
}

// e191ViesDuRegistre construit le registre d identite EXACTEMENT comme `BuildFromPositions` (les
// memes entrees, la meme horloge) et rend ses vies : c est lui, et lui seul, qui apparie le fil
// des morts aux slots (lots 1.6 et 1.8).
func e191ViesDuRegistre(g *goldenInputs, sorted []filmdec.BipedPosition) []lifeSpan {
	if len(sorted) == 0 {
		return nil
	}
	opt := g.options()
	origin := sorted[0].TimestampUS
	step := uint64(opt.frameIntervalMS()) * 1000
	reg := BuildIdentityRegistry(IdentityInput{
		Positions: sorted, BipedCreations: g.BipedCreations,
		Deaths: g.Deaths, PlayerIndices: g.PlayerIndices, FilmTable: g.FilmTable,
		Bots: opt.Bots, Fire: fireRefs(g.Fire), RosterXUIDs: opt.RosterXUIDs,
		Participants: opt.Participants,
		Clock: IdentityClock{
			OriginUS: origin, StepUS: step, FrameCount: frameSpan(sorted, origin, step),
		},
		MatchID: g.Film,
	})
	return reg.Vies()
}

// e191ObjetsPortes lit le manifeste du titre et rend les identifiants d objet `kind = "carried"`
// — les APPAREILS PORTES, ceux dont l origine est la question du point 3 du lot.
func e191ObjetsPortes(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRootForTest(t), "config", "titles",
		"halo_infinite", "mappings", "replay_labels.toml"))
	if err != nil {
		t.Fatalf("manifeste des libelles de rejeu : %v", err)
	}
	out := map[string]bool{}
	for id, o := range manifesteObjetsEquipement(t, raw) {
		if o.Nature == "carried" {
			out[id] = true
		}
	}
	return out
}

// e191UnePose annote UNE pose avec ce que chaque signal en dit.
func e191UnePose(p filmdec.EquipmentPlacement, film string,
	designees map[filmdec.EquipmentLifeKey][]uint64, portes map[string]bool, ctx e191Ctx,
) e191Pose {
	fam := ctx.familles[p.GlobalID]
	if fam == "" {
		fam = equipmentFamilyOther
	}
	id := fmt.Sprintf("0x%08x", p.GlobalID)
	q := e191Pose{
		Film: film, Famille: fam, ID: id, Portee: portes[id],
		Origine: OriginUnknown, Regle: "sans_poseur",
		SpawnDtMS: e191PlusProcheSigne(designees[p.Life], p.T0US),
		MortMS:    e191SansSignal, PriseMS: e191SansSignal, FenetreMS: e191SansSignal,
	}
	// LE POSEUR EST CELUI DE LA PRODUCTION (`equipmentOwner`) : le mesurer autrement
	// fabriquerait des transitions qui n existent pas.
	if slot, _, ok := equipmentOwner(ctx.positions, p); ok {
		q.Slot, q.AvecPoseur = slot, true
		q.Origine, q.Regle = f1OrigineParFenetre(ctx.vies[slot], p), "fenetre"
		q.FenetreMS = e191EcartFenetre(ctx.vies[slot], p)
		q.OrigineF1Avant = f1OrigineAvant(ctx.vies[slot], p)
		if best, ok := f1VieRetenue(ctx.vies[slot], p.T0US); ok {
			q.DistM = float64(dist3([3]float32{p.X, p.Y, p.Z}, [3]float32{best.x, best.y, best.z}))
		}
		q.MortMS = e191PlusProche(ctx.morts[slot], p.T0US)
		q.PriseMS = e191PlusProche(ctx.prises[slot], p.T0US)
	}
	if equipmentIsSpawnedPiece(id) {
		q.Origine, q.Regle = OriginDeployed, "manifeste"
	}
	return q
}

// e191EcartFenetre rend l ecart, en ms, que la fenetre de 200 ms mesure : entre la creation de
// l objet et la fin de la vie de poseur RETENUE par `equipmentOrigin`.
func e191EcartFenetre(lives []equipLife, p filmdec.EquipmentPlacement) float64 {
	best, bestGap := equipLife{}, ^uint64(0)
	for _, v := range lives {
		gap := uint64(0)
		switch {
		case p.T0US < v.from:
			gap = v.from - p.T0US
		case p.T0US > v.to:
			gap = p.T0US - v.to
		}
		if gap == 0 {
			return float64(equipTimeGap(p.T0US, v.to)) / 1000
		}
		if gap < bestGap {
			best, bestGap = v, gap
		}
	}
	if bestGap == ^uint64(0) {
		return e191SansSignal
	}
	return float64(equipTimeGap(p.T0US, best.to)) / 1000
}

// e191PlusProcheSigne rend l ecart SIGNE, en ms, a l instant de la liste le plus proche en
// VALEUR ABSOLUE — un evenement qui SUIT la creation rend un nombre positif.
func e191PlusProcheSigne(instants []uint64, at uint64) float64 {
	best, abs := e191SansSignal, e191SansSignal
	for _, t := range instants {
		d := float64(equipTimeGap(t, at)) / 1000
		if d >= abs {
			continue
		}
		abs = d
		if t < at {
			d = -d
		}
		best = d
	}
	return best
}

// e191PlusProche rend l ecart ABSOLU, en ms, a l instant le plus proche de la liste.
func e191PlusProche(instants []uint64, at uint64) float64 {
	best := e191SansSignal
	for _, t := range instants {
		if d := float64(equipTimeGap(t, at)) / 1000; d < best {
			best = d
		}
	}
	return best
}
