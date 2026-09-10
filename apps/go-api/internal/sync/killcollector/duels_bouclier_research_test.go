package killcollector

// duels_bouclier_research_test.go — SONDE N°2 : LA MESURE QUI DECIDE DU LOT 7 (duels).
//
// LA QUESTION, ET POURQUOI ELLE N'EST PAS CELLE DE LA SONDE N°1. La sonde n°1
// (`analysis/replay/duels_sonde_*_test.go`, note `.ai/V7.5/film_re/SONDE_DUELS_2026-09-06.md`)
// a mesure la RECIPROCITE DU DEGAT et l'a rejetee : le film n'emet que 0,8 a 4,8
// `damage_aftermath` par mort, le compte des duels serait un sous-comptage de 4 a 8x. Mais
// elle s'interdisait la base, donc elle ignorait QUI etait le tueur pour 50 a 90 % des morts.
// Or le tueur est connu hors film, par le kill-feed (`match_kill_events_latest`).
//
// LA QUESTION DEVIENT DONC : pour les morts dont le kill-feed NOMME le tueur, le BOUCLIER DU
// TUEUR a-t-il chute pendant la fenetre d'engagement ? Si oui, la victime lui a rendu des
// coups, et c'est un duel. Le bouclier est replique DANS LE RECORD DE POSITION a chaque fois
// qu'il change (`BipedPosition.ShieldAt`) — deux a dix fois plus dense que le flux de degats.
//
// CE QUE CETTE SONDE MESURE (items 1.3 a 1.7 de `.ai/PLAN_DUELS_PORTEE_2026-09-06.md`) :
//
//	A  pont du tueur — part des kills du feed dont le TUEUR est localise a T par le pont de
//	   PRODUCTION (`ResolveSlotXUID` + `BuildKillPositions`). Sans lui, rien d'autre n'a de sens.
//	O  oracle victime — part des kills ou le bouclier de LA VICTIME chute dans [T-2 s, T]. Un
//	   mourant a forcement perdu son bouclier : O n'est pas un resultat, c'est LE PLAFOND DE
//	   CAPTURE du canal. Mesure a 44-76 % par la sonde n°1 sur une fenetre de 3 s.
//	B  bouclier du tueur — part des kills ou le bouclier du TUEUR chute dans [T-2 s, T].
//	T  temoin — B avec la fenetre deplacee de 37 s vers le passe. Doit s'effondrer, sinon B ne
//	   mesure qu'une densite d'evenements.
//	D  discrimination — parmi les ADVERSAIRES du tueur ayant chute dans la fenetre, la victime
//	   est-elle la seule ? L'equipe vient de `match_participants` (la sonde n°1 ne l'avait pas).
//
// LE GATE, ECRIT DANS LE PLAN AVANT LA MESURE, ET JAMAIS AJUSTE AU RESULTAT :
//
//	A >= 80 %  ET  B/O dans [0,35 ; 0,90]  ET  B/temoin >= 3  ET  D >= 60 %
//
// B se lit NORMALISE par O (decision D2 du plan) : B brut est une BORNE BASSE plafonnee par le
// taux de capture du canal, et comparer une borne basse a un seuil absolu n'aurait pas de sens.
//
// CE QU'ELLE NE FAIT PAS : aucune ecriture (base ou disque), aucun appel reseau, aucune cuisson
// d'artefact de rejeu. La base est ouverte en LECTURE par `OpenReadForQuery` (jamais
// `OpenReadOnly` force : le serveur de dev peut tenir le fichier en RW — modele mono-process,
// ADR 0013/0016).
//
// USAGE (un match par process, verrou de decodage pris) :
//
//	cd apps/go-api && GOCACHE=<worktree>/.gocache-lot1 \
//	  DUELS_DATA_ROOT=<racine principale> DUELS_MATCH=000d5950 DUELS_MAP=Cliffhanger \
//	  go test ./internal/sync/killcollector -run TestSondeDuelsBouclier -v -timeout 900s
//
// `DUELS_MATCH` accepte la forme COURTE (prefixe de 8 caracteres, cf. title.FilmShortMatchID)
// comme la forme complete : la sonde resout l'identifiant complet en base et REFUSE une
// resolution ambigue plutot que de choisir.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

const (
	duelsBMatchEnv = "DUELS_MATCH"
	duelsBMapEnv   = "DUELS_MAP"
	duelsBRootEnv  = "DUELS_DATA_ROOT"
	duelsBFilmEnv  = "DUELS_FILM"
)

// duelsBSlug : la sonde est mono-titre par construction — le film Infinite est la seule source
// de bouclier du depot. Aucune generalisation n'est feinte ici.
const duelsBSlug = "halo_infinite"

// duelsBWindowUS : la fenetre d'engagement. DEUX SECONDES, et c'est une mesure, pas un reglage :
// la sonde n°1 a trouve la reciprocite IDENTIQUE a 2, 3 et 5 s — la riposte vit dans les deux
// premieres secondes ou n'existe pas (decision D2 du plan).
const duelsBWindowUS = 2_000_000

// duelsBShiftUS : decalage du temoin. MEME valeur que la sonde n°1 (37 s) : grand devant la
// fenetre, et sans rapport simple avec les cadences de match (respawn ~8 s), pour ne pas
// retomber en phase.
const duelsBShiftUS = 37_000_000

// duelsBChuteEps : baisse minimale de fraction de bouclier retenue comme une CHUTE. MEME valeur
// que la sonde n°1 : le quantum de la source vaut 1/64 = 0,0156 ; 0,02 est juste au-dessus, donc
// aucune vraie baisse n'est perdue et aucun aller-retour de quantification n'est compte.
const duelsBChuteEps = 0.02

// duelsBLifeGapUS : trou au-dela duquel deux lectures consecutives appartiennent a DEUX VIES —
// le bouclier repart alors plein et la comparaison serait une fausse chute. MEME valeur que
// `lifeGapUS` (internal/analysis/replay/lives.go), qui decoupe les vies ; elle est recopiee ici
// parce qu'elle n'est pas exportee, et la recopie est signalee pour qu'un changement la-bas se
// voie ici.
const duelsBLifeGapUS = 5_000_000

// duelsBKill : une mort du kill-feed, ses deux identites resolues.
type duelsBKill struct {
	killer, victim uint64
	timeMS         int64
	publishable    bool
}

func TestSondeDuelsBouclier(t *testing.T) {
	saisi, carte, root := os.Getenv(duelsBMatchEnv), os.Getenv(duelsBMapEnv), os.Getenv(duelsBRootEnv)
	if saisi == "" || carte == "" || root == "" {
		t.Skipf("sonde desactivee : %s, %s et %s requis", duelsBMatchEnv, duelsBMapEnv, duelsBRootEnv)
	}

	release := filmdec.LockProcessDecode()
	defer release()

	paths := title.NewPathResolver(root)
	db, closeDB := duelsBOuvrirBase(t, paths.SharedDBPath(duelsBSlug))
	defer closeDB()

	matchID := duelsBResoudreMatchID(t, db, saisi)
	kills, ecartes, nonPubliables := duelsBLireKills(t, db, matchID)
	equipes := duelsBLireEquipes(t, db, matchID)
	if len(kills) == 0 {
		t.Fatalf("match %s : aucun kill du feed avec tueur ET victime resolus", matchID)
	}

	film, positions, origin := duelsBLireFilm(t, root, matchID, carte, paths.MapQuantBoundsPath(duelsBSlug))
	slotXUID, owners := duelsBPontIdentite(t, film, positions, equipes)

	t.Logf("MATCH %s (%s) : %d kills du feed exploitables (%d ecartes faute de xuid, "+
		"%d non publiables), %d participants avec equipe, %d positions, %d slots au pont",
		matchID, carte, len(kills), ecartes, nonPubliables, len(equipes), len(positions), len(slotXUID))
	t.Logf("PONT : %d vies, %d nommees, %d lectures d'index, %d desaccords, %d collisions de slot, "+
		"calage du fil des morts %d ms sur %d morts appariees",
		owners.ViesTotal(), owners.ViesNommeesParLaLecture(), owners.LecturesIndex(), owners.DesaccordsIndex(),
		owners.CollisionsDeSlot(), owners.DeathOffsetMS(), owners.DeathOffsetMatches())
	t.Logf("HORLOGE : origine du film %d us — les instants du feed (horloge du MATCH) sont poses "+
		"sur l'horloge du FILM par tUS = time_ms*1000 + origine, comme le fait la production "+
		"(positions.go -> BuildKillPositions)", origin)

	duelsBMesurer(t, duelsBSonde{
		kills: kills, positions: positions, slotXUID: slotXUID, registre: owners,
		equipes: equipes, originUS: int64(origin),
	})
}

// duelsBOuvrirBase ouvre la base partagee en LECTURE.
//
// `OpenReadForQuery` ET PAS `OpenReadOnly` : DuckDB refuse deux configurations differentes sur
// le meme fichier dans un meme process, et la base peut etre tenue en RW ailleurs. Item 1.1 du
// plan — la contrainte est ecrite avant la mesure parce que la violer ne rate pas, elle CORROMPT.
func duelsBOuvrirBase(t *testing.T, path string) (*sql.DB, func()) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("base partagee %s introuvable : %v", path, err)
	}
	db, release, err := duckdb.OpenReadForQuery(path)
	if err != nil {
		t.Fatalf("ouverture lecture de %s : %v", path, err)
	}
	return db, release
}

// duelsBResoudreMatchID rend l'identifiant COMPLET du match a partir de la saisie.
//
// LA FORME COURTE N'EST PAS UN SIMPLE PREFIXE, ET C'EST VERIFIE : `title.FilmShortMatchID`
// coupe au PREMIER TIRET, et se replie sur les 8 premiers caracteres seulement s'il n'y en a
// pas. Le `LIKE 'xxx%'` ci-dessous est donc une PRESELECTION ; le controle qui fait foi est
// l'egalite des formes courtes, appliquee a chaque candidat.
func duelsBResoudreMatchID(t *testing.T, db *sql.DB, saisi string) string {
	t.Helper()
	court := title.FilmShortMatchID(saisi)
	rows, err := db.QueryContext(context.Background(),
		`SELECT DISTINCT match_id FROM match_kill_events_latest WHERE match_id LIKE ? || '%'`, court)
	if err != nil {
		t.Fatalf("resolution du match %q : %v", saisi, err)
	}
	defer func() { _ = rows.Close() }()

	var candidats []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("resolution du match %q, scan : %v", saisi, err)
		}
		if title.FilmShortMatchID(id) == court {
			candidats = append(candidats, id)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("resolution du match %q, rows : %v", saisi, err)
	}
	if len(candidats) != 1 {
		t.Fatalf("resolution du match %q : %d candidats (%v) — on ne tranche pas",
			saisi, len(candidats), candidats)
	}
	return candidats[0]
}

// duelsBLireKills lit les morts du kill-feed par la vue `_latest` UNIQUEMENT (regle ART n°2 :
// une lecture brute servirait les lignes d'une passe de decodage precedente).
//
// POPULATION : les kills dont LES DEUX xuids sont resolus — exactement ce que
// `killRefsFromDeaths` (positions.go) retient en production, et la seule population qui puisse
// avoir une position. Les autres (victime bot, tueur bot, nom non resolu) sont COMPTES et
// rendus a l'appelant : un denominateur ampute en silence gonflerait tous les taux.
//
// `publishable` est LU et COMPTE mais ne filtre pas : il porte sur la publication LIGNE PAR
// LIGNE d'une passe, pas sur la validite du couple (tueur, victime, instant) que cette sonde
// mesure. Le compte est publie pour que la note dise sur quoi elle a mesure.
func duelsBLireKills(t *testing.T, db *sql.DB, matchID string) (kills []duelsBKill, ecartes, nonPubliables int) {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), `
		SELECT feed_killer_xuid, victim_xuid, time_ms, publishable
		FROM match_kill_events_latest
		WHERE match_id = ?
		ORDER BY time_ms
	`, matchID)
	if err != nil {
		t.Fatalf("lecture des kills de %s : %v", matchID, err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var killer, victim sql.NullString
		var timeMS int64
		var publishable bool
		if err := rows.Scan(&killer, &victim, &timeMS, &publishable); err != nil {
			t.Fatalf("lecture des kills de %s, scan : %v", matchID, err)
		}
		k, ok1 := duelsBXUID(killer)
		v, ok2 := duelsBXUID(victim)
		if !ok1 || !ok2 {
			ecartes++
			continue
		}
		if !publishable {
			nonPubliables++
		}
		kills = append(kills, duelsBKill{killer: k, victim: v, timeMS: timeMS, publishable: publishable})
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("lecture des kills de %s, rows : %v", matchID, err)
	}
	return kills, ecartes, nonPubliables
}

// duelsBXUID : un xuid est une suite decimale, et rien d'autre — MEME regle que `parseXUID`
// (positions.go), a laquelle cette lecture ne peut pas recourir directement parce qu'elle part
// d'une colonne NULLABLE.
func duelsBXUID(v sql.NullString) (uint64, bool) {
	if !v.Valid || v.String == "" {
		return 0, false
	}
	return duelsBParse(v.String)
}

func duelsBParse(s string) (uint64, bool) {
	x, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return x, true
}

// duelsBLireEquipes rend `xuid -> team_id` pour les participants du match. Un `team_id` NULL
// n'entre PAS dans la table : l'absence d'equipe se lit comme une absence, jamais comme une
// equipe zero — c'est ce qui empeche la discrimination (D) de compter comme adversaire un
// joueur dont on ignore le camp.
func duelsBLireEquipes(t *testing.T, db *sql.DB, matchID string) map[uint64]int64 {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), `
		SELECT xuid, team_id FROM match_participants
		WHERE match_id = ? AND xuid IS NOT NULL AND xuid <> ''
	`, matchID)
	if err != nil {
		t.Fatalf("lecture du roster de %s : %v", matchID, err)
	}
	defer func() { _ = rows.Close() }()

	out := map[uint64]int64{}
	for rows.Next() {
		var xuid string
		var team sql.NullInt64
		if err := rows.Scan(&xuid, &team); err != nil {
			t.Fatalf("lecture du roster de %s, scan : %v", matchID, err)
		}
		x, ok := duelsBParse(xuid)
		if !ok || !team.Valid {
			continue
		}
		out[x] = team.Int64
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("lecture du roster de %s, rows : %v", matchID, err)
	}
	return out
}

// duelsBLireFilm charge le film du cache local et en tire les positions AVEC le bouclier.
//
// `CaptureDirs = true` poursuit le MEME record de deux composants de plus (i4 vie, i5 bouclier) :
// c'est ce qui donne `BipedPosition.ShieldAt()`, et ca ne change aucune position emise. C'est
// exactement l'option que le lot 7 devrait activer en production s'il s'ouvre.
func duelsBLireFilm(
	t *testing.T, root, matchID, carte, bornesPath string,
) (*filmsource.Film, []filmdec.BipedPosition, uint64) {
	t.Helper()
	dir := duelsBFilmDir(root, matchID)
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s illisible : %v", dir, err)
	}
	cat, err := filmdec.LoadMapQuantCatalog(bornesPath)
	if err != nil {
		t.Fatalf("catalogue de bornes %s : %v", bornesPath, err)
	}
	entry, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("carte %q absente du catalogue : %v", carte, err)
	}
	opt := filmdec.DefaultScanFilmOptions()
	rng := entry.Range()
	opt.WorldRange = &rng
	opt.CaptureDirs = true
	positions, err := filmdec.ScanBipedPositions(film, opt)
	if err != nil {
		t.Fatalf("positions bipeds : %v", err)
	}
	origin, err := replay.ScanClockOrigin(film)
	if err != nil {
		t.Fatalf("horloge du film : %v", err)
	}
	return film, positions, origin
}

// duelsBFilmDir rend le repertoire des chunks du film dans le cache local.
//
// LE CACHE DE FILMS N'A PAS D'ACCESSEUR `PathResolver` : il est hérité (title-agnostic, hors
// `data/titles/`) et sa seule abstraction du depot est `haloclient.LocalFilmCache`, qui resout
// `<cache>/film_chunks/<court>` en interne SANS exposer le repertoire. La composition est donc
// faite ici, une fois, et `DUELS_FILM` permet de la court-circuiter entierement.
func duelsBFilmDir(root, matchID string) string {
	if dir := os.Getenv(duelsBFilmEnv); dir != "" {
		return dir
	}
	return filepath.Join(root, "data", "cache", "film_chunks", title.FilmShortMatchID(matchID))
}

// duelsBPontIdentite construit le pont slot -> xuid PAR LES FONCTIONS DE PRODUCTION
// (`ScanDeaths`, `ScanPlayerIndices`, `ResolveSlotXUID`) — jamais une resolution locale : « deux
// decodeurs du meme fait divergeraient » (killpos_bridge.go). Le roster fourni a l'index de
// joueur est celui de `match_participants`, comme en production (rosterUint64 dans positions.go).
func duelsBPontIdentite(
	t *testing.T, film *filmsource.Film, positions []filmdec.BipedPosition, equipes map[uint64]int64,
) (map[uint32]uint64, replay.IdentityRegistry) {
	t.Helper()
	deaths, err := replay.ScanDeaths(film)
	if err != nil {
		t.Fatalf("fil des morts : %v", err)
	}
	roster := make([]uint64, 0, len(equipes))
	for x := range equipes {
		roster = append(roster, x)
	}
	idx, err := replay.ScanPlayerIndices(film, roster)
	if err != nil {
		t.Fatalf("index de joueur : %v", err)
	}
	owners := replay.BuildIdentityRegistry(replay.IdentityInput{
		Positions: positions, Deaths: deaths, PlayerIndices: idx, RosterXUIDs: roster,
	})
	// LE PONT EPURE, PAS L'APLATI (lot 6.1) : la sonde s'en sert pour ses propres denominateurs
	// (chutes de bouclier par siege) ; le PLACEMENT, lui, passe le registre entier.
	slotXUID := owners.PontEpure()
	if len(slotXUID) == 0 {
		t.Fatalf("pont slot->xuid vide (vies=%d nommees=%d lectures=%d) : rien a mesurer",
			owners.ViesTotal(), owners.ViesNommeesParLaLecture(), owners.LecturesIndex())
	}
	return slotXUID, owners
}

// duelsBPct formate un taux en gardant le couple brut visible — MEME convention que la sonde
// n°1 : un pourcentage seul ne s'additionne pas d'un match a l'autre, les comptes si.
func duelsBPct(n, d int) string {
	if d == 0 {
		return fmt.Sprintf("%d/0 (n.d.)", n)
	}
	return fmt.Sprintf("%d/%d = %.1f %%", n, d, 100*float64(n)/float64(d))
}
