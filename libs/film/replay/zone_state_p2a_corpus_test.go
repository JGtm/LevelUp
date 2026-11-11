package replay

// zone_state_p2a_corpus_test.go — LOT C-bis PHASE 2a : LES ENTREES DE LA MESURE.
//
// Trois entrees se rejoignent ici, et chacune vient d'une source DISTINCTE :
//
//	les FORMES     `data/titles/halo_infinite/reference/map_objectives.json` (donnee versionnee) ;
//	les POSITIONS  le film, decode par `grammar` puis assemble par `BuildFromPositions` ;
//	les INSTANTS   les evenements nommes du statborg (`objectives`), identifies par xuid.
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
	"context"
	"path/filepath"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/depottest"

	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/objectives"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"

	"github.com/JGtm/halo-infinite-files-reader/mapvar"
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
	// ObjType est la famille d'objectif au sens d'`objectives` : seul `zone` a une table
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
// Releve du 2026-08-18 sur l'oracle des participants
// (lecture seule, aucune base ouverte). `1b1e380f` est HORS corpus par consigne.
var p2aCorpus = map[string]p2aFilm{
	// Les deux Strongholds de Vagabond : le coeur de CB.2a.1 et CB.2a.2 (oracle nomme).
	"7344d24f": {Mode: "Strongholds", ObjType: objectives.ObjectiveTypeZone,
		Carte: "vagabond", MapID: "105f5d84-8de1-4908-af3a-1c4f3bf9d642", Players: []p2aPlayer{
			{"2533491259474651", 17, 13, 2, 0},
			{"2603225103667623", 16, 14, 7, 0},
			{"2547408409520575", 8, 16, 9, 0},
			{"2736944460235975", 17, 16, 11, 0},
			{"2599417078913178", 27, 13, 7, 1},
			{"2626540472179150", 9, 13, 9, 1},
			{"2627844859029573", 9, 14, 8, 1},
			{"2632367878260135", 14, 18, 5, 1},
		}},
	"696a9d7c": {Mode: "Strongholds", ObjType: objectives.ObjectiveTypeZone,
		Carte: "vagabond", MapID: "105f5d84-8de1-4908-af3a-1c4f3bf9d642", Players: []p2aPlayer{
			{"2756122492253326", 8, 12, 8, 0},
			{"2617315724508307", 15, 9, 10, 0},
			{"2732838123306580", 15, 14, 3, 0},
			{"2728249988014663", 16, 13, 4, 0},
			{"2603225103667623", 9, 15, 7, 1},
			{"2623439693412443", 15, 11, 5, 1},
			{"2776634124695723", 15, 17, 4, 1},
			{"2648087613616541", 9, 12, 4, 1},
		}},
	// Le Bastion a BASES TENUES AU COUP D ENVOI (A au camp 1, C au camp 0, B neutre ; diagnostic
	// confirme au Theater le 2026-10-06), temoin de l etat initial lu aux images-cles
	// (`zone_etat_initial_temoin_test.go`). Releve du 2026-10-07 par `levelup replay-facts-export`
	// (lecture seule de la base partagee).
	"572e236b": {Mode: "Strongholds", ObjType: objectives.ObjectiveTypeZone,
		Carte: "fortress", MapID: "0d1c9255-d912-416c-befc-5f3e5e176df2", Players: []p2aPlayer{
			{"2586904587338955", 8, 1, 2, 0},
			{"2571481023573114", 10, 4, 7, 0},
			{"2658276600435201", 15, 10, 3, 0},
			{"2770776903346818", 1, 1, 0, 0},
			{"2718665968625447", 10, 6, 8, 0},
			{"bid(33.0)", 0, 0, 0, 0},
			{"2603225103667623", 7, 12, 3, 1},
			{"2558964982616721", 1, 10, 1, 1},
			{"2623439693412443", 9, 11, 3, 1},
			{"2547408409520575", 5, 11, 6, 1},
		}},
	// Le Bastion COURT (Streets, ~2 min 45, 6 captures) : temoin du proprietaire designe par le
	// NOM de la jauge (`zone_proprietaire_nom_temoin_test.go`) — deux zones n y ont qu une capture
	// concordante. Releve du 2026-10-07 par `levelup replay-facts-export` (lecture seule de la base
	// partagee).
	"114b0040": {Mode: "Strongholds", ObjType: objectives.ObjectiveTypeZone,
		Carte: "streets", MapID: "e23ea388-9bcb-4180-a0dc-fbe987751b9e", Players: []p2aPlayer{
			{"2660553501845133", 2, 5, 3, 0},
			{"2538106996528547", 3, 5, 1, 0},
			{"2545713452248078", 2, 5, 2, 0},
			{"2704752500905550", 1, 5, 2, 0},
			{"2620818869973870", 4, 3, 2, 1},
			{"2574748290730998", 7, 2, 2, 1},
			{"2583901221067092", 4, 1, 2, 1},
			{"2543518745364847", 5, 2, 1, 1},
		}},
	// Les KOTH : CB.2a.3. `0a247154` joue sur Solitude, ABSENTE du catalogue de formes — le
	// negatif est ecrit plutot que contourne (et la phase 1 y avait deja mesure 0 rampe).
	"01e1f945": {Mode: "KOTH", ObjType: "none",
		Carte: "catalyst", MapID: "f7e8cde9-0c0a-487c-94a3-61bfa0f20465", Players: []p2aPlayer{
			{"2668191463923697", 11, 18, 7, 0},
			{"2686803061848660", 13, 15, 7, 0},
			{"2581425573506704", 14, 15, 8, 0},
			{"2716501986668322", 11, 10, 7, 0},
			{"2561509825159198", 18, 13, 4, 1},
			{"2758073455829681", 9, 11, 8, 1},
			{"2705281959365014", 23, 9, 6, 1},
			{"2547408409520575", 7, 14, 3, 1},
		}},
	"606d9844": {Mode: "KOTH", ObjType: "none",
		Carte: "chasm", MapID: "a455572d-3141-48bc-ac55-dac78d9b52c9", Players: []p2aPlayer{
			{"2646267994113001", 2, 2, 2, 0},
			{"2638391743317361", 9, 1, 2, 0},
			{"2648754588523414", 11, 2, 3, 0},
			{"2550117376578726", 5, 4, 2, 0},
			{"2603225103667623", 2, 8, 1, 1},
			{"2808832712178603", 1, 5, 0, 1},
			{"2570854261076248", 2, 6, 0, 1},
			{"2707698133402110", 3, 7, 0, 1},
		}},
	"8076f97f": {Mode: "KOTH", ObjType: "none",
		Carte: "shogun", MapID: "33075df7-01c8-40e1-8b3e-1baee0054c76", Players: []p2aPlayer{
			{"2709341845069387", 6, 7, 4, 0},
			{"2779576370057794", 10, 11, 7, 0},
			{"2622964463507713", 9, 10, 2, 0},
			{"2641407264103267", 6, 10, 6, 0},
			{"2603225103667623", 7, 7, 4, 1},
			{"2623439693412443", 12, 8, 7, 1},
			{"2559025798969566", 13, 7, 5, 1},
			{"2598530222917521", 6, 9, 6, 1},
		}},
	"0a247154": {Mode: "KOTH", ObjType: "none",
		Carte: "solitude", MapID: "4a5e5612-2b2e-4375-a0b3-9335a68815f3", Players: []p2aPlayer{
			{"2623439693412443", 16, 22, 15, 0},
			{"2639945145752479", 23, 21, 9, 0},
			{"2744683300358232", 24, 15, 13, 0},
			{"2712215804940684", 18, 20, 9, 0},
			{"2740974044877510", 18, 21, 5, 1},
			{"2814303751122561", 17, 20, 13, 1},
			{"2789965669678788", 21, 20, 16, 1},
			{"2791806035335783", 22, 20, 14, 1},
		}},
	// Le TEMOIN : un Slayer, ou aucune zone ne se capture. La phase 1 y a mesure 20 valeurs
	// d'i1 (le canal se tait) ; il sert ici de temoin de BANDE, pas de mesure d'appariement.
	"000d5950": {Mode: "Slayer (temoin)", ObjType: "none",
		Carte: "cliffhanger", MapID: "5324364b-39a8-4f93-96a6-b80a1f18ce8a", Players: []p2aPlayer{
			{"2603225103667623", 8, 14, 1, 0},
			{"2633090228607708", 8, 14, 1, 0},
			{"2543372234349153", 14, 13, 3, 0},
			{"2667725237021739", 13, 9, 1, 0},
			{"2742063072864889", 12, 10, 6, 1},
			{"2603004317949713", 14, 9, 2, 1},
			{"2709083604009238", 14, 13, 1, 1},
			{"2812089420052706", 10, 11, 2, 1},
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
func (f p2aFilm) p2aLines() []types.PlayerLine {
	out := make([]types.PlayerLine, 0, len(f.Players))
	for _, p := range f.Players {
		out = append(out, types.PlayerLine{
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

// p2aBobine charge le FILM du repertoire de chunks par la porte canonique du cache
// (`filmcache`) — jamais une disposition reconstituee sur place. C'est lui que prennent les
// points d'entree d'`objectives` depuis l'item 1.5 : une decompression, pas une par
// balayage. Il remplace `p2aSource`, qui n'ouvrait que le manifeste et n'a plus d'appelant.
func p2aBobine(t *testing.T, dir string) *source.Film {
	t.Helper()
	film, ok, err := filmcache.LoadFilmDir(dir)
	if err != nil || !ok {
		t.Fatalf("film illisible (%s) : ok=%v err=%v", dir, ok, err)
	}
	return film
}

// p2aStartMS rend l'instant de depart de chaque chunk, sur l'horloge du manifeste — que le
// film charge porte deja (`Meta()`), sans rouvrir le manifeste.
func p2aStartMS(film *source.Film) map[int]int {
	out := map[int]int{}
	for _, m := range film.Meta() {
		out[m.Index] = m.StartMS
	}
	return out
}

// p2aRefDir rend le repertoire des donnees de reference versionnees du titre.
func p2aRefDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(depottest.Racine(t), "data", "titles", titreDeReference, "reference")
}

// p2aQuant rend l'entree de bornes de quantification de la carte. Sans elle, les positions
// restent des quanta et aucune distance n'a de sens : la mesure s'arrete.
func p2aQuant(t *testing.T, carte string) *profile.MapQuantEntry {
	t.Helper()
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(p2aRefDir(t), "map_quant_bounds.json"))
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
func p2aDoc(t *testing.T, dir, short string, quant *profile.MapQuantEntry) ReplayDocument {
	t.Helper()
	worldRange := quant.Range()
	scan := grammar.DefaultScanFilmOptions()
	scan.WorldRange = &worldRange
	pos, err := grammar.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions illisibles (%s) : %v", dir, err)
	}
	var opt Options
	deaths, err := grammar.ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("fil des morts illisible (%s) : %v", dir, err)
	}
	opt.Deaths = deaths
	if len(deaths) > 0 {
		idx, err := grammar.ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
		if err != nil {
			t.Logf("index de joueur illisible : %v — les vies restent nommees par le fil des morts", err)
		}
		table, collisions := injectiveOrEmpty(idx)
		if collisions > 0 {
			t.Logf("index de joueur NON INJECTIF (%d collisions) — table ecartee", collisions)
		}
		opt.PlayerIndices = table
	}
	clockUS, err := grammar.ScanFilmClockOrigin(dir)
	if err != nil {
		t.Logf("origine d'horloge illisible : %v — les instants ne seront pas recales", err)
		clockUS = 0
	}
	opt.FilmClockOriginUS = clockUS
	return BuildFromPositions(context.Background(), short, titreDeReference, pos, nil, opt)
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
	objectives.StatZoneCaptures: true,
	objectives.StatZoneSecures:  true,
}

// p2aCaptures rend les captures et securisations de zone, identifiees par xuid.
func p2aCaptures(film *source.Film, f p2aFilm) []objectives.IdentifiedEvent {
	if f.ObjType != objectives.ObjectiveTypeZone {
		return nil // KOTH, Oddball, Slayer : aucun emplacement nomme (cf. named.go)
	}
	named := objectives.NamedEvents(film, f.ObjType)
	identity := objectives.SlotIdentity(film, f.p2aLines())
	out := make([]objectives.IdentifiedEvent, 0, len(named))
	for _, e := range objectives.IdentifyNamedEvents(named, identity) {
		if p2aZoneStats[e.Stat] {
			out = append(out, e)
		}
	}
	return out
}
