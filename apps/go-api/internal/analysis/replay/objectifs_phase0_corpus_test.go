package replay

// objectifs_phase0_corpus_test.go — PHASE 0 du plan
// `.ai/V7.5/replay2d/PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md` : LE SOCLE COMMUN des mesures.
//
// CE QUE LA PHASE 0 CHERCHE. Le drapeau de CTF et le crane d'Oddball sont, selon la
// decision 1 du plan, des `weap` TENUS EN MAIN : leur position est celle de leur PORTEUR,
// donc d'une piste bipede deja publiee. Il n'y a rien a decoder de l'objet — il n'y a qu'a
// reconnaitre, dans le loadout du porteur, la famille de l'objet porte. Reste a etablir
// QUELLE famille, et par quel canal.
//
// POURQUOI CET INSTRUMENT NE PEUT PAS SE CONTENTER DE `ScanFilmKeyframeLoadouts`. Ce
// balayage-la n'accepte que les familles du CATALOGUE d'armes (`loadoutFamilies`, derive de
// l'enum v2). Le drapeau n'y est evidemment pas : le chercher avec un predicat qui l'exclut
// par construction ne rendrait jamais que des fusils. On balaye donc les 32 bits SANS
// predicat a l'interieur de l'emprise du record bipede, et c'est la CONFRONTATION A
// L'ORACLE qui fait la selectivite — pas une liste ecrite d'avance.
//
// LES DEUX ESPACES DE SLOTS, ET POURQUOI IL FAUT DEUX PONTS. Les evenements nommes portent
// un slot d'entite STATBORG (10..24 pairs) ; les loadouts et les trajectoires portent un
// slot de BIPED. Rien ne garantit qu'ils coincident (`objectiveevents/slotidentity.go` le
// dit et refuse de le supposer). Le seul pont licite est le XUID, et il se compose de deux
// lectures independantes :
//
//	statborg -> xuid   triplet (frags, morts, assistances) contre les lignes de match
//	biped    -> xuid   le fil des morts nomme chaque vie (cf. lives.go / owners.go)
//
// LES LIGNES DE MATCH SONT UN FIXTURE GELE, pas une lecture de base. Ce paquet n'ouvre
// aucune DuckDB (regle du depot) et la phase 0 n'en ouvre pas davantage : les triplets
// ci-dessous sont releves une fois pour toutes dans l'instantane parquet
// `data/backups/staging/halo_infinite/shared_matches_v2/match_participants_20260711_090652.parquet`
// et figes ici — meme convention que l'oracle a huit joueurs d'`objectiveevents/named_test.go`.
//
// GARDE : `OBJ_FILM` porte la RACINE du cache film (le repertoire qui contient
// `film_chunks/` et `film_manifests/`). Sans elle, tous les tests de la phase 0 se sautent
// proprement — ils ne peuvent pas tourner en CI, les films ne sont pas versionnes.
//
//	CGO_ENABLED=0 OBJ_FILM=<depot>/data/cache \
//	  go test ./internal/analysis/replay/ -run Objectifs -v -timeout 30m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// objFilmEnv — la garde d'environnement de toute la phase 0.
const objFilmEnv = "OBJ_FILM"

// objBipedTI est l'archetype des records de bipede joueur dans la table d'image-cle. Meme
// valeur que `filmdec.BipedTypeIndex` ; on passe par la constante exportee du decodeur pour
// ne pas en faire une seconde definition.
const objBipedTI = int(filmdec.BipedTypeIndex)

// objPlayer est la ligne de match d'un joueur, telle que l'instantane parquet la donne.
type objPlayer struct {
	XUID                   string
	Kills, Deaths, Assists int
	Team                   int
}

// objFilm decrit un film du corpus de la phase 0.
type objFilm struct {
	// Mode est la famille d'objectif au sens d'`objectiveevents` (flag / skull).
	Mode string
	// Carte sert au rapport, jamais au decodage (aucune borne de carte n'est requise :
	// la phase 0 travaille en QUANTA, cf. objBuildBridge).
	Carte   string
	Players []objPlayer
}

// objCorpus — les films de la phase 0, avec leurs lignes de match gelees.
//
// Releve du 2026-08-18 sur `match_participants_20260711_090652.parquet` (lecture seule,
// aucune base ouverte). Les triplets sont UNIQUES a l'interieur de chaque film : c'est la
// condition que `SlotIdentity` exige pour apparier sans ambiguite, et elle est verifiee par
// `TestObjectifsPhase0Pont`.
var objCorpus = map[string]objFilm{
	// Les trois films CTF du corpus (plan §digest, ligne « Corpus »).
	"64e8adfa": {Mode: objectiveevents.ObjectiveTypeFlag, Carte: "Catalyst", Players: []objPlayer{
		{"2533274792763167", 18, 12, 5, 1},
		{"2533274808613055", 15, 15, 9, 1},
		{"2533274823110022", 20, 16, 1, 0},
		{"2535413221816250", 10, 21, 7, 0},
		{"2535449464686885", 9, 19, 4, 0},
		{"2535449963449748", 17, 12, 10, 1},
		{"2535456378021162", 24, 9, 4, 1},
		{"2535465820713037", 8, 19, 3, 0},
	}},
	"530820e5": {Mode: objectiveevents.ObjectiveTypeFlag, Carte: "Catalyst", Players: []objPlayer{
		{"2533274823110022", 11, 15, 4, 0},
		{"2533274830798809", 9, 12, 1, 1},
		{"2533274858283686", 17, 8, 5, 0},
		{"2533274933094600", 15, 14, 4, 1},
		{"2535412713182055", 8, 15, 5, 1},
		{"2535414530233939", 10, 6, 7, 0},
		{"2535435137320355", 7, 13, 3, 1},
		{"2535469190789936", 15, 11, 6, 0},
	}},
	"53ce4390": {Mode: objectiveevents.ObjectiveTypeFlag, Carte: "Behemoth", Players: []objPlayer{
		{"2533274803754807", 23, 17, 8, 1},
		{"2533274823110022", 6, 12, 3, 1},
		{"2533274830881544", 25, 13, 2, 1},
		{"2533274840602701", 16, 19, 4, 0},
		{"2533274860882060", 15, 13, 5, 0},
		{"2535430195856593", 7, 12, 2, 1},
		{"2535439497055986", 13, 17, 6, 0},
		{"2535462641971683", 10, 14, 5, 0},
	}},
	// Le film Oddball, pour la generalisation au crane (item 0.3).
	"24dbb67d": {Mode: objectiveevents.ObjectiveTypeSkull, Carte: "Recharge", Players: []objPlayer{
		{"2533274815819321", 9, 10, 4, 0},
		{"2533274822068549", 7, 11, 12, 0},
		{"2533274823110022", 5, 12, 5, 1},
		{"2533274858283686", 20, 11, 3, 1},
		{"2535425305181079", 15, 12, 4, 0},
		{"2535463185952034", 14, 10, 6, 0},
		{"2535463284114517", 10, 12, 0, 1},
		{"2535469190789936", 8, 10, 5, 1},
	}},
}

// objCTFFilms — l'ordre de passage des trois films CTF, fige pour que les sorties se
// comparent d'une session a l'autre.
var objCTFFilms = []string{"64e8adfa", "530820e5", "53ce4390"}

// objBallFilm — le film Oddball de l'item 0.3.
const objBallFilm = "24dbb67d"

// objDiskFilm est la source de film adossee au cache disque : c'est `filmcache.Source`, la
// SEULE source disque du depot (garde-rail `TestUneSeuleSourceDisqueDeFilm`). La copie locale
// que portait la phase 0 (« newDiskFilmSource n'est pas atteignable d'ici ») est retiree le
// 2026-08-18 : le paquet `replay` importe `filmcache` sans cycle, la raison n'existait pas.
type objDiskFilm = filmcache.Source

// objChunkDir rend le repertoire des chunks d'un film.
func objChunkDir(root, id string) string { return filepath.Join(root, "film_chunks", id) }

// objOpenFilm charge le manifeste d'un film ; (nil, false) si le film n'est pas en cache.
func objOpenFilm(t *testing.T, root, id string) (*objDiskFilm, bool) {
	t.Helper()
	src, ok, err := filmcache.Open(root, id)
	if err != nil {
		t.Fatalf("manifeste %s illisible : %v", id, err)
	}
	if !ok {
		return nil, false
	}
	return src, true
}

// objRequireRoot rend la racine du cache film, ou saute le test.
func objRequireRoot(t *testing.T) string {
	t.Helper()
	root := os.Getenv(objFilmEnv)
	if root == "" {
		t.Skipf("mesure non demandee : %s vide (racine du cache film)", objFilmEnv)
	}
	return root
}

// objBridge porte le pont slot de bipede -> joueur, et le calage d'horloge qui va avec.
//
// LE CALAGE EST LA PIECE QUI PERMET DE SUPERPOSER LES DEUX CHAINES : les evenements nommes
// et le fil des morts sont dates sur l'horloge du MATCH, les images-cles et les positions
// sur celle du FILM. `OffsetMS` est la difference, MESUREE par `bestDeathOffset` :
// filmMS = matchMS + OffsetMS.
type objBridge struct {
	SlotXUID   map[uint32]uint64
	OffsetMS   int64
	Deaths     []Death
	LivesTotal int
	// DeathsNamed / OffsetMatches sont les denominateurs du pont : un pont publie sans eux
	// ne se juge pas.
	DeathsNamed, OffsetMatches, Collisions int
}

// objBuildBridge construit le pont biped -> xuid par la SEULE lecture, exactement comme la
// production (`buildOwners`), mais en QUANTA : aucune borne de carte n'est fournie.
//
// POURQUOI LES QUANTA SUFFISENT, ET POURQUOI C'EST PREFERABLE. Le pont ne consomme des
// positions que leur SLOT et leur INSTANT — jamais leurs coordonnees. Exiger les bornes de
// carte ferait dependre la phase 0 du catalogue de quantification, donc echouer sur toute
// carte absente du catalogue (Behemoth, Recharge). Le decoupage en vies est identique :
// `buildLifeSpans` ne lit que `TimestampUS`.
func objBuildBridge(dir string) (objBridge, error) {
	scan := filmdec.DefaultScanFilmOptions()
	scan.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		return objBridge{}, fmt.Errorf("positions : %w", err)
	}
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		return objBridge{}, fmt.Errorf("fil des morts : %w", err)
	}
	idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
	if err != nil {
		return objBridge{}, fmt.Errorf("index de joueur : %w", err)
	}
	table, _ := injectiveOrEmpty(idx)
	sort.SliceStable(pos, func(i, j int) bool { return pos[i].TimestampUS < pos[j].TimestampUS })
	lives := buildLifeSpans(indexBySlot(pos))
	off, matched := bestDeathOffset(lives, deaths)
	named := nameLivesByDeaths(lives, deaths, off)
	_, byXUID, collisions := ownersFromLives(lives, table.ByXUID)
	return objBridge{
		SlotXUID: byXUID, OffsetMS: off, Deaths: deaths, LivesTotal: len(lives),
		DeathsNamed: named, OffsetMatches: matched, Collisions: collisions,
	}, nil
}

// objBridgeMemo memorise le pont bipede par film : sa construction balaye TOUT le film
// (positions + fil des morts + index de joueur) et plusieurs mesures de la phase 0 en ont
// besoin. Sans memo, le meme balayage serait rejoue a chaque test.
var objBridgeMemo = map[string]objBridge{}

// objBridgeOf rend le pont bipede d'un film, construit une seule fois par process.
func objBridgeOf(t *testing.T, root, id string) objBridge {
	t.Helper()
	if b, ok := objBridgeMemo[id]; ok {
		return b
	}
	b, err := objBuildBridge(objChunkDir(root, id))
	if err != nil {
		t.Fatalf("%s : pont bipede : %v", id, err)
	}
	objBridgeMemo[id] = b
	return b
}

// objRecordsMemo memorise le balayage des records de bipede d'image-cle par film.
var objRecordsMemo = map[string][]objRecord{}

// objImagesMemo retient le nombre d'images-cles balayees par film.
var objImagesMemo = map[string]int{}

// objRecordsOf rend les records de bipede d'image-cle d'un film, balayes une seule fois.
func objRecordsOf(t *testing.T, root, id string) ([]objRecord, int) {
	t.Helper()
	if r, ok := objRecordsMemo[id]; ok {
		return r, objImagesMemo[id]
	}
	recs, images, err := objScanKeyframeBipeds(objChunkDir(root, id))
	if err != nil {
		t.Fatalf("%s : balayage des images-cles : %v", id, err)
	}
	objRecordsMemo[id], objImagesMemo[id] = recs, images
	return recs, images
}

// objCheckTripletsUniques verifie que les triplets du fixture designent chacun UN joueur.
// Sans cette propriete, `SlotIdentity` s'abstient (a bon droit) et le pont par totaux se vide.
func objCheckTripletsUniques(t *testing.T, f objFilm) {
	t.Helper()
	vus := map[[3]int]int{}
	for _, p := range f.Players {
		vus[[3]int{p.Kills, p.Deaths, p.Assists}]++
	}
	for k, n := range vus {
		if n > 1 {
			t.Errorf("triplet %v porte par %d joueurs — appariement par totaux impossible", k, n)
		}
	}
}
