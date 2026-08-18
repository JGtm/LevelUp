package replay

// zone_state_p2a_corpus_test.go — LOT C-bis PHASE 2a : LES ENTREES DE LA MESURE.
//
// Trois entrees se rejoignent ici, et chacune vient d'une source DISTINCTE :
//
//	les FORMES     `data/titles/halo_infinite/reference/map_objectives.json` (donnee versionnee) ;
//	les POSITIONS  le film, decode par `filmdec` puis assemble par `BuildFromPositions` ;
//	les INSTANTS   les evenements nommes du statborg (`objectiveevents`), identifies par xuid.
//
// AUCUNE BASE N'EST OUVERTE. Le pont slot statborg -> xuid exige les lignes de match (frags,
// morts, assistances) et l'equipe exige le roster : les deux sont GELES ci-dessous, releves une
// fois pour toutes dans l'export `registre_film/oracle_lotA_participants.tsv` (lui-meme tire de
// `match_participants`). Meme convention que la phase 0 du plan des objectifs vivants — le
// paquet `replay` n'ouvre aucune DuckDB, et la phase 2a n'en ouvre pas davantage.
//
// LES DEUX HORLOGES, ET LE FAIT QU'ELLES NE SONT PAS LA MEME :
//
//	`ti=13` et les evenements nommes   ms depuis le PREMIER PAQUET DU FILM (startMS du manifeste) ;
//	les positions du rejeu             frame depuis le PREMIER PAQUET DE POSITION.
//
// L'ecart entre ces deux zeros est publie par l'artefact sous `OriginMs` (mesure de 3,6 a 50,8 s
// selon le match) : `p2aFrameOf` le retranche. Sans cette correction, toutes les actions seraient
// posees `OriginMs` trop tard sur l'axe des positions — le defaut que `cmd/zone-attribution`
// avait deja diagnostique.

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// p2aPlayer est la ligne de match d'un joueur, telle que l'export TSV la donne.
type p2aPlayer struct {
	XUID                   string
	Kills, Deaths, Assists int
	Team                   int
}

// p2aFilm decrit un film du corpus de la phase 2a.
type p2aFilm struct {
	// Mode est le libelle du mode, pour le rapport.
	Mode string
	// ObjType est la famille d'objectif au sens d'`objectiveevents` : seul `zone` a une table
	// d'emplacements nommes. KOTH n'en a pas — c'est pourquoi CB.2a.3 se mesure sur les
	// positions et non sur un oracle d'evenements.
	ObjType string
	// Carte est la cle des BORNES de quantification (nom de carte, minuscules) ; MapID est la
	// cle des FORMES (asset UGC). Les deux catalogues ne s'indexent pas pareil.
	Carte, MapID string
	Players      []p2aPlayer
}

// p2aCorpus — les films de la phase 2a, avec leurs lignes de match gelees.
//
// Releve du 2026-08-18 sur `.ai/V7.5/replay2d/registre_film/oracle_lotA_participants.tsv`
// (lecture seule, aucune base ouverte). `1b1e380f` est HORS corpus par consigne.
var p2aCorpus = map[string]p2aFilm{
	// Les deux Strongholds de Vagabond : le coeur de CB.2a.1 et CB.2a.2 (oracle nomme).
	"7344d24f": {Mode: "Strongholds", ObjType: objectiveevents.ObjectiveTypeZone,
		Carte: "vagabond", MapID: "105f5d84-8de1-4908-af3a-1c4f3bf9d642", Players: []p2aPlayer{
			{"2533274819954312", 17, 13, 2, 0},
			{"2533274823110022", 16, 14, 7, 0},
			{"2535469190789936", 8, 16, 9, 0},
			{"2535473295198622", 17, 16, 11, 0},
			{"2533274901940205", 27, 13, 7, 1},
			{"2535433849912379", 9, 13, 9, 1},
			{"2535449981534849", 9, 14, 8, 1},
			{"2535460550991892", 14, 18, 5, 1},
		}},
	"696a9d7c": {Mode: "Strongholds", ObjType: objectiveevents.ObjectiveTypeZone,
		Carte: "vagabond", MapID: "105f5d84-8de1-4908-af3a-1c4f3bf9d642", Players: []p2aPlayer{
			{"2533274989524964", 8, 12, 8, 0},
			{"2535429028393121", 15, 9, 10, 0},
			{"2535442956677772", 15, 14, 3, 0},
			{"2535458126310341", 16, 13, 4, 0},
			{"2533274823110022", 9, 15, 7, 1},
			{"2533274858283686", 15, 11, 5, 1},
			{"2533274917930188", 15, 17, 4, 1},
			{"2535438933682278", 9, 12, 4, 1},
		}},
	// Les KOTH : CB.2a.3. `0a247154` joue sur Solitude, ABSENTE du catalogue de formes — le
	// negatif est ecrit plutot que contourne (et la phase 1 y avait deja mesure 0 rampe).
	"01e1f945": {Mode: "KOTH", ObjType: "none",
		Carte: "catalyst", MapID: "f7e8cde9-0c0a-487c-94a3-61bfa0f20465", Players: []p2aPlayer{
			{"2533274832680942", 11, 18, 7, 0},
			{"2533274969015015", 13, 15, 7, 0},
			{"2535451337001599", 14, 15, 8, 0},
			{"2535455052477469", 11, 10, 7, 0},
			{"2533274824966873", 18, 13, 4, 1},
			{"2535459782888916", 9, 11, 8, 1},
			{"2535462158176179", 23, 9, 6, 1},
			{"2535469190789936", 7, 14, 3, 1},
		}},
	"606d9844": {Mode: "KOTH", ObjType: "none",
		Carte: "chasm", MapID: "a455572d-3141-48bc-ac55-dac78d9b52c9", Players: []p2aPlayer{
			{"2533274864142980", 2, 2, 2, 0},
			{"2533275031831732", 9, 1, 2, 0},
			{"2535472156173951", 11, 2, 3, 0},
			{"2535472834247640", 5, 4, 2, 0},
			{"2533274823110022", 2, 8, 1, 1},
			{"2533274835138874", 1, 5, 0, 1},
			{"2533274897620970", 2, 6, 0, 1},
			{"2535449018082899", 3, 7, 0, 1},
		}},
	"8076f97f": {Mode: "KOTH", ObjType: "none",
		Carte: "shogun", MapID: "33075df7-01c8-40e1-8b3e-1baee0054c76", Players: []p2aPlayer{
			{"2533274857387572", 6, 7, 4, 0},
			{"2533275004520376", 10, 11, 7, 0},
			{"2535407066740278", 9, 10, 2, 0},
			{"2535435974688461", 6, 10, 6, 0},
			{"2533274823110022", 7, 7, 4, 1},
			{"2533274858283686", 12, 8, 7, 1},
			{"2535456259091650", 13, 7, 5, 1},
			{"2541316932603141", 6, 9, 6, 1},
		}},
	"0a247154": {Mode: "KOTH", ObjType: "none",
		Carte: "solitude", MapID: "4a5e5612-2b2e-4375-a0b3-9335a68815f3", Players: []p2aPlayer{
			{"2533274858283686", 16, 22, 15, 0},
			{"2535406399121492", 23, 21, 9, 0},
			{"2535455023454167", 24, 15, 13, 0},
			{"2535470442430325", 18, 20, 9, 0},
			{"2533274846684310", 18, 21, 5, 1},
			{"2533274870240621", 17, 20, 13, 1},
			{"2533274969598328", 21, 20, 16, 1},
			{"2535420289574892", 22, 20, 14, 1},
		}},
	// Le TEMOIN : un Slayer, ou aucune zone ne se capture. La phase 1 y a mesure 20 valeurs
	// d'i1 (le canal se tait) ; il sert ici de temoin de BANDE, pas de mesure d'appariement.
	"000d5950": {Mode: "Slayer (temoin)", ObjType: "none",
		Carte: "cliffhanger", MapID: "5324364b-39a8-4f93-96a6-b80a1f18ce8a", Players: []p2aPlayer{
			{"2533274823110022", 8, 14, 1, 0},
			{"2533274826120416", 8, 14, 1, 0},
			{"2533274980284321", 14, 13, 3, 0},
			{"2535467794760703", 13, 9, 1, 0},
			{"2533274815845110", 12, 10, 6, 1},
			{"2533274882097883", 14, 9, 2, 1},
			{"2535437947245250", 14, 13, 1, 1},
			{"2535444178793711", 10, 11, 2, 1},
		}},
}

// p2aFilmOf rend la fiche du film designe par la garde, ou saute la mesure.
func p2aFilmOf(t *testing.T, dir string) (string, p2aFilm) {
	t.Helper()
	short := filepath.Base(dir)
	f, ok := p2aCorpus[short]
	if !ok {
		t.Skipf("film %s hors corpus de la phase 2a", short)
	}
	return short, f
}

// p2aLines rend les lignes de match qui fondent le pont slot statborg -> xuid.
func (f p2aFilm) p2aLines() []objectiveevents.PlayerLine {
	out := make([]objectiveevents.PlayerLine, 0, len(f.Players))
	for _, p := range f.Players {
		out = append(out, objectiveevents.PlayerLine{
			XUID: p.XUID, Kills: p.Kills, Deaths: p.Deaths, Assists: p.Assists,
		})
	}
	return out
}

// p2aTeams rend le roster xuid -> team_id. C'EST LA SOURCE D'EQUIPE DE CB.2a.2 : le film ne
// publie pas la sienne (`game-engine-team-mapping` lit ses bits sans les publier, phase 1 §6.3).
func (f p2aFilm) p2aTeams() map[string]int {
	out := map[string]int{}
	for _, p := range f.Players {
		out[p.XUID] = p.Team
	}
	return out
}

// p2aSource ouvre la source disque canonique du film (`filmcache`) — jamais une implementation
// locale de `FilmSource` (le garde-rail `filmcache_guard_test.go` l'interdit, et il a raison).
func p2aSource(t *testing.T, dir string) *filmcache.Source {
	t.Helper()
	src, ok, err := filmcache.OpenChunkDir(dir)
	if err != nil || !ok {
		t.Fatalf("manifeste de film illisible (%s) : ok=%v err=%v", dir, ok, err)
	}
	return src
}

// p2aStartMS rend l'instant de depart de chaque chunk, sur l'horloge du manifeste.
func p2aStartMS(src *filmcache.Source) map[int]int {
	out := map[int]int{}
	for _, m := range src.Chunks() {
		out[m.Index] = m.StartMS
	}
	return out
}

// p2aRefDir rend le repertoire des donnees de reference versionnees du titre.
func p2aRefDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRootForTest(t), "data", "titles", title.DefaultSlug, "reference")
}

// p2aQuant rend l'entree de bornes de quantification de la carte. Sans elle, les positions
// restent des quanta et aucune distance n'a de sens : la mesure s'arrete.
func p2aQuant(t *testing.T, carte string) *filmdec.MapQuantEntry {
	t.Helper()
	cat, err := filmdec.LoadMapQuantCatalog(filepath.Join(p2aRefDir(t), "map_quant_bounds.json"))
	if err != nil {
		t.Fatalf("catalogue de bornes illisible : %v", err)
	}
	e, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("bornes de la carte %q absentes : %v", carte, err)
	}
	return &e
}

// p2aRolesZones : les roles du catalogue qui portent une ZONE SURFACIQUE.
//
// POURQUOI L'UNION, ET PAS LE SEUL ROLE DU MODE. En Strongholds, les zones capturables sont les
// `strongholds_zone` — mais le catalogue ne connait AUCUN role de colline (KOTH), et la colline
// se pose en jeu sur des emplacements que le fichier de carte declare sous d'autres roles. La
// mesure KOTH cherche donc la zone la plus proche parmi TOUTES les zones surfaciques de la
// carte ; le rapport publie le role retenu, de sorte qu'un appariement sur `extraction_zone`
// reste lisible comme tel.
var p2aRolesZones = []mapvar.Role{mapvar.RoleStrongholdZone, mapvar.RoleExtractionZone}

// p2aZones rend les zones surfaciques de la carte. `roles` restreint (Strongholds : le seul role
// du mode) ; vide = toutes celles de `p2aRolesZones`.
func p2aZones(t *testing.T, mapID string, roles ...mapvar.Role) []Zone {
	t.Helper()
	cat, err := LoadMapObjectives(filepath.Join(p2aRefDir(t), "map_objectives.json"))
	if err != nil {
		t.Fatalf("catalogue d'objectifs illisible : %v", err)
	}
	e, err := cat.Lookup(mapID)
	if err != nil {
		t.Skipf("carte %s absente du catalogue de formes : appariement geometrique IMPOSSIBLE"+
			" sur ce film (negatif ecrit)", mapID)
	}
	if len(roles) == 0 {
		roles = p2aRolesZones
	}
	var out []Zone
	for _, r := range roles {
		out = append(out, e.ZonesOfRole(r).Zones...)
	}
	// Le rang spatial est refige sur l'UNION : deux roles poses cote a cote donneraient sinon
	// deux zones de meme rang, et la table slot -> zone deviendrait ambigue.
	sortZonesSpatially(out)
	return out
}

// p2aDoc assemble le document de rejeu : positions en METRES, vies nommees, origine publiee.
//
// C'est le chemin de `BuildFromFilm` reduit a ce que la phase 2a consomme (positions, fil des
// morts, index de joueur, origine d'horloge) : ni tirs, ni armes, ni projectiles. La machine de
// l'utilisateur paie chaque balayage — on ne decode pas ce qu'on ne mesure pas.
func p2aDoc(t *testing.T, dir, short string, quant *filmdec.MapQuantEntry) ReplayDocument {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	worldRange := quant.Range()
	scan := filmdec.DefaultScanFilmOptions()
	scan.WorldRange = &worldRange
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions illisibles (%s) : %v", dir, err)
	}
	var opt Options
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("fil des morts illisible (%s) : %v", dir, err)
	}
	opt.Deaths = deaths
	if len(deaths) > 0 {
		idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
		if err != nil {
			t.Logf("index de joueur illisible : %v — les vies restent nommees par le fil des morts", err)
		}
		table, collisions := injectiveOrEmpty(idx)
		if collisions > 0 {
			t.Logf("index de joueur NON INJECTIF (%d collisions) — table ecartee", collisions)
		}
		opt.PlayerIndices = table
	}
	clockUS, err := ScanFilmClockOrigin(dir)
	if err != nil {
		t.Logf("origine d'horloge illisible : %v — les instants ne seront pas recales", err)
		clockUS = 0
	}
	opt.FilmClockOriginUS = clockUS
	return BuildFromPositions(short, title.DefaultSlug, pos, nil, opt)
}

// p2aFrameOf convertit un instant de l'horloge du MANIFESTE en index de frame du rejeu, en
// retranchant l'origine publiee. Rend false quand l'instant tombe hors de la fenetre du rejeu.
func p2aFrameOf(doc ReplayDocument, tMS int) (int, bool) {
	if doc.FrameIntervalMS <= 0 || doc.FrameCount <= 0 || doc.OriginMs == nil {
		return 0, false
	}
	rel := int64(tMS) - *doc.OriginMs
	if rel < 0 {
		return 0, false
	}
	f := int(rel / int64(doc.FrameIntervalMS))
	if f >= doc.FrameCount {
		return 0, false
	}
	return f, true
}

// p2aZoneStats : les statistiques d'objectif du mode a zones. Les frags et assistances sont
// nommes par le meme decodeur mais ne sont PAS des actions de zone.
var p2aZoneStats = map[string]bool{
	objectiveevents.StatZoneCaptures: true,
	objectiveevents.StatZoneSecures:  true,
}

// p2aCaptures rend les captures et securisations de zone, identifiees par xuid.
func p2aCaptures(src objectiveevents.FilmSource, f p2aFilm) []objectiveevents.IdentifiedEvent {
	if f.ObjType != objectiveevents.ObjectiveTypeZone {
		return nil // KOTH, Oddball, Slayer : aucun emplacement nomme (cf. named.go)
	}
	named := objectiveevents.NamedEvents(src, f.ObjType)
	identity := objectiveevents.SlotIdentity(src, f.p2aLines())
	out := make([]objectiveevents.IdentifiedEvent, 0, len(named))
	for _, e := range objectiveevents.IdentifyNamedEvents(named, identity) {
		if p2aZoneStats[e.Stat] {
			out = append(out, e)
		}
	}
	return out
}
