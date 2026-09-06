// Package duckdb — tactical_repo.go : implementation DuckDB de
// port.TacticalRepository (onglet Tactique, plan .ai/PLAN_TACTIQUE_2026-09-06.md
// phase 2).
//
// Source : `match_registry` x `match_participants` pour l'UNIVERS (les matchs
// retenus par le filtre) ; `kill_positions_latest` x `match_kill_events_latest`
// pour les positions et le journal des morts — calque de la jointure de
// kill_distance_repo.go, memes vues `_latest` (regle ART n 2, jamais la table
// brute).
//
// ─── CE QUI DIFFERE DE KillDistanceRepo ────────────────────────────────────────
//
// Celui-la lit UN match et rend une distance par arme. Celui-ci lit une FENETRE
// de matchs (une carte, un filtre) et rend des positions brutes : pas de
// classificateur d'arme, pas de source de degat, donc pas de garde d'unanimite
// sur `source_tag`. La garde d'ambiguite reste, sous une autre forme (cf.
// QTacticalPositions).
//
// ─── POURQUOI `publishable` EST EXIGE DES DEUX COTES ───────────────────────────
//
// `publishable = FALSE` signifie : les lignes de la passe sont justes en AGREGAT
// et fausses INDIVIDUELLEMENT (bijection nom -> xuid a marge nulle, cas BTB). Or
// les deux lectures d'ici sont des attributions PAR LIGNE :
//
//   - le raster range chaque point sur l'axe « moi / escouade / adversaires »
//     d'apres l'identite de la victime ou du tueur — une identite permutee peint
//     le point du mauvais cote ;
//   - l'echange demande QUI a venge QUI — une identite permutee fabrique une
//     vengeance qui n'a pas eu lieu.
//
// Une passe non publiable est donc ECARTEE ici, comme dans KillDistanceRepo, et
// contrairement a KillSourceClassRepo (qui, lui, ne produit que des cumuls).
//
// ─── AUCUN FILTRE, AUCUN SCAN ──────────────────────────────────────────────────
//
// Toute lecture est bornee par le joueur (`mp.xuid = ?`). La lecture SPATIALE
// (KillPositions) l'est en plus par une CARTE : un xuid vide ou une carte vide y
// sont un REFUS, jamais un balayage de `shared.kill_positions` en entier.
//
// KillEvents, elle, accepte une carte VIDE depuis le 2026-09-06 (phase 3) : la
// page Escouade lit le journal des morts d'une COMPOSITION, qui n'a pas de carte,
// et le perimetre de matchs y est resserre en Go. Ecrire une seconde requete pour
// ce seul predicat aurait donne deux definitions de « le journal des morts du
// joueur » ; le SELECT est donc le meme, la carte devenant un parametre neutre
// (`? = ” OR mr.map_id = ?`). La borne reste le joueur, jamais la table entiere.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// tacticalReadTimeout borne une lecture tactique. Plus large que les 15 s de
// KillDistanceRepo : la fenetre porte sur des centaines de matchs, pas un seul.
const tacticalReadTimeout = 30 * time.Second

// TacticalRepo implemente port.TacticalRepository.
type TacticalRepo struct {
	pdb *PlayerDB

	// modeTax resout les prefixes `pair_name` d'une categorie de mode du filtre.
	// Zero-value = aucune classification : le filtre `mode` est alors IGNORE
	// (degradation gracieuse, cf. analysis.BuildNeighborsWhereClause) plutot
	// qu'une comparaison de slug.
	modeTax analysis.ModeTaxonomy
}

// NewTacticalRepo cree un TacticalRepo lie a un PlayerDB.
func NewTacticalRepo(pdb *PlayerDB) *TacticalRepo {
	return &TacticalRepo{pdb: pdb}
}

// WithModeTaxonomy injecte la taxonomie de modes du titre (chainable).
func (r *TacticalRepo) WithModeTaxonomy(t analysis.ModeTaxonomy) *TacticalRepo {
	r.modeTax = t
	return r
}

// ─── L'UNIVERS ─────────────────────────────────────────────────────────────────

// QTacticalUnivers est le SELECT des matchs RETENUS : ceux du joueur, sur la
// carte demandee quand il y en a une, qui passent le filtre.
//
// LA CARTE EST OPTIONNELLE, ET C'EST UN PARAMETRE, PAS UNE CONCATENATION (ajout
// 2026-09-06, phase 3) : `? = ” OR mr.map_id = ?` neutralise le predicat quand
// l'appelant ne vise aucune carte (page Escouade). Assembler la clause en Go
// aurait fait DEUX chaines SQL pour un seul univers — et le garde-rail structurel
// campaign_exclusion_guard_test ne balaye qu'une constante, pas un assemblage.
//
// Le fragment de filtre est produit par
// analysis.BuildNeighborsWhereClause (aliases `mr` et `mp`), source unique du
// vocabulaire de filtre du depot — et donc du fragment timezone canonique pour
// les bornes de date.
//
// LE DRAPEAU `mesure` (ajout 2026-09-06, correction G2) dit si le journal des morts
// de ce match est LISIBLE : au moins une ligne publiable dans
// `match_kill_events_latest`. Un match dont le film n'a jamais ete decode — ou dont le
// film Theater a EXPIRE cote serveur — n'est pas un match a zero mort, c'est un match
// ILLISIBLE : il ne peut alimenter aucun numerateur, et le laisser au denominateur
// « par match » ferait varier la grandeur avec la couverture de film au lieu du jeu.
//
// `publishable` est exige ICI comme dans les deux lectures : sans lui, un match dont
// toutes les lignes sont ecartees compterait comme mesure sur cette page et pas sur la
// page Escouade, qui lit le meme journal filtre pareil.
//
// PREFIXE `Q` ET TOKEN CAMPAGNE (correction R2, revue du 2026-09-06) : le
// garde-rail structurel campaign_exclusion_guard_test ne balaye QUE les constantes
// nommees `Q<...>`. Sans le prefixe, un lecteur per-player passait sous son radar ;
// sans le token, les ~287 matchs Campagne d'un joueur Halo 5 entraient dans
// l'univers des rasters alors que l'Explorateur les masque. Le token est resolu au
// call site par resolveCampaignExclusion, qui connait le titre du joueur (no-op
// pour Infinite, qui n'a aucun match Campagne au registre).
const QTacticalUnivers = `
SELECT mr.match_id, COALESCE(mp.outcome, ?) AS outcome,
       EXISTS (SELECT 1 FROM match_kill_events_latest e
               WHERE e.match_id = mr.match_id AND e.publishable) AS mesure
FROM match_registry mr
JOIN match_participants mp ON mp.match_id = mr.match_id
WHERE mp.xuid = ? AND (? = '' OR mr.map_id = ?)` + campaignExclusionToken

// universSQL assemble le SELECT de l'univers et ses arguments, token Campagne
// resolu pour le titre du joueur. `q.MapID` vide = toutes les cartes.
func (r *TacticalRepo) universSQL(q domain.TacticalQuery) (string, []any) {
	clause := analysis.BuildNeighborsWhereClause(q.Filtre, r.modeTax.Prefixes)
	args := append([]any{domain.OutcomeUnknown, q.PlayerXUID, q.MapID, q.MapID}, clause.Args...)
	return resolveCampaignExclusion(QTacticalUnivers, r.pdb.TitleSlug, "mr") + clause.SQL, args
}

// chargerUnivers lit les matchs retenus PUIS la composition de leurs equipes.
//
// Les equipes sont lues par une SECONDE requete qui re-selectionne l'univers en
// sous-requete, plutot que par une liste `IN (?, ?, ...)` construite en Go : une
// fenetre de plusieurs centaines de matchs ferait autant de parametres lies, et
// le predicat serait alors ecrit a deux endroits au lieu d'un.
func (r *TacticalRepo) chargerUnivers(ctx context.Context, db *sql.DB, q domain.TacticalQuery) (domain.TacticalUnivers, error) {
	univ := domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}

	selectSQL, args := r.universSQL(q)
	rows, err := db.QueryContext(ctx, selectSQL+" ORDER BY mr.match_id", args...)
	if err != nil {
		return univ, fmt.Errorf("univers: %w", err)
	}
	if err := scanRows(ctx, rows, "univers", func(sc rowScanner) error {
		var m domain.TacticalMatch
		if err := sc.Scan(&m.MatchID, &m.Outcome, &m.Mesure); err != nil {
			return err
		}
		univ.Matchs = append(univ.Matchs, m)
		return nil
	}); err != nil {
		return univ, err
	}
	if len(univ.Matchs) == 0 {
		return univ, nil
	}

	equipesSQL := `SELECT p.match_id, p.xuid, p.team_id FROM match_participants p
		WHERE p.match_id IN (SELECT u.match_id FROM (` + selectSQL + `) u)
		  AND p.xuid IS NOT NULL AND p.xuid <> '' AND p.team_id IS NOT NULL`
	rows, err = db.QueryContext(ctx, equipesSQL, args...)
	if err != nil {
		return univ, fmt.Errorf("equipes: %w", err)
	}
	err = scanRows(ctx, rows, "equipes", func(sc rowScanner) error {
		var matchID, xuid string
		var team int
		if err := sc.Scan(&matchID, &xuid, &team); err != nil {
			return err
		}
		parMatch := univ.Equipes[matchID]
		if parMatch == nil {
			parMatch = make(map[string]int)
			univ.Equipes[matchID] = parMatch
		}
		parMatch[xuid] = team
		return nil
	})
	return univ, err
}

// ─── LES TROIS LECTURES ────────────────────────────────────────────────────────

// QTacticalMaps : les cartes JOUEES par le joueur sous le filtre.
//
// Les codes d'issue sont des PARAMETRES LIES (domain.OutcomeWin / OutcomeLoss),
// jamais des litteraux dans la chaine SQL — un `outcome = 2` en dur est
// exactement ce que le ratchet no_raw_outcome_literal interdit.
//
// Meme token Campagne que QTacticalUnivers, et pour la meme raison : sans lui, la
// grille d'entree d'un joueur Halo 5 affichait ses cartes de Campagne a cote de
// ses cartes d'arene.
const QTacticalMaps = `
SELECT mr.map_id,
       COALESCE(mr.map_name, '') AS map_name,
       COUNT(*)                          AS matchs,
       COUNT(*) FILTER (WHERE mp.outcome = ?) AS victoires,
       COUNT(*) FILTER (WHERE mp.outcome = ?) AS defaites
FROM match_registry mr
JOIN match_participants mp ON mp.match_id = mr.match_id
WHERE mp.xuid = ? AND mr.map_id IS NOT NULL AND mr.map_id <> ''` + campaignExclusionToken

// MapsPlayed liste les cartes jouees, matchs decroissants puis map_id.
func (r *TacticalRepo) MapsPlayed(ctx context.Context, q domain.TacticalQuery) ([]domain.TacticalMapRow, error) {
	if q.PlayerXUID == "" {
		return nil, fmt.Errorf("TacticalRepo.MapsPlayed: xuid vide")
	}
	ctx, cancel := context.WithTimeout(ctx, tacticalReadTimeout)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "TacticalRepo.MapsPlayed: shared reader", "err", err)
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	clause := analysis.BuildNeighborsWhereClause(q.Filtre, r.modeTax.Prefixes)
	logIgnoredFilters(ctx, "TacticalRepo.MapsPlayed", clause.IgnoredFilters)
	args := append([]any{domain.OutcomeWin, domain.OutcomeLoss, q.PlayerXUID}, clause.Args...)
	query := resolveCampaignExclusion(QTacticalMaps, r.pdb.TitleSlug, "mr") + clause.SQL +
		` GROUP BY mr.map_id, mr.map_name ORDER BY matchs DESC, mr.map_id`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, r.degrader(ctx, "MapsPlayed", err)
	}
	out := make([]domain.TacticalMapRow, 0)
	err = scanRows(ctx, rows, "TacticalRepo.MapsPlayed", func(sc rowScanner) error {
		var row domain.TacticalMapRow
		if err := sc.Scan(&row.MapID, &row.MapName,
			&row.Matchs, &row.Victoires, &row.Defaites); err != nil {
			return err
		}
		out = append(out, row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	r.habillerNomsFR(ctx, out)
	return out, nil
}

// habillerNomsFR remplit MapNameFR depuis `metadata.asset_translations`.
//
// POURQUOI PAS `match_registry.map_name_fr` (correction R3, revue du 2026-09-06) :
// cette colonne est SYSTEMATIQUEMENT NULLE — constat deja pose deux fois dans ce
// paquet (`engagement_score_repo_queries.go`, `filters_repo_asset_names.go`). La
// lire faisait sortir toutes les cartes avec un nom FR VIDE, et le test ne le
// voyait pas parce que sa fixture semait une valeur qui n'existe pas en prod.
//
// UNE REQUETE PAR CARTE, et c'est assume : la boucle est bornee par le nombre de
// cartes DISTINCTES jouees par un joueur (quelques dizaines), chaque lecture est un
// point sur `(asset_id, asset_type)` indexe. Une variante par lot serait une
// TROISIEME implementation de la meme resolution dans ce paquet — la centralisation
// des deux copies existantes est notee au §7 du plan, hors perimetre de ce lot.
func (r *TacticalRepo) habillerNomsFR(ctx context.Context, rows []domain.TacticalMapRow) {
	if r.pdb == nil || r.pdb.Metadata == nil {
		return
	}
	for i := range rows {
		if fr, ok := mapNameFRFromAssetTranslations(ctx, r.pdb.Metadata, rows[i].MapID); ok {
			rows[i].MapNameFR = fr
		}
	}
}

// QTacticalPositions : les morts MESUREES des matchs de l'univers — position
// connue des DEUX cotes (tueur ET victime), passe publiable, et pas d'ambiguite.
//
// LA GARDE D'AMBIGUITE. `kill_positions_latest` porte UNE ligne par
// (match, tueur, instant) — elle n'a aucune colonne de victime (cf.
// steps_shared_kill_positions.go). Un double kill au meme instant donne donc DEUX
// kill-events pour UNE seule position de victime : accrocher cette position a la
// mauvaise victime rangerait le point du mauvais cote de l'axe « qui », de facon
// indetectable a l'ecran. `HAVING count(*) = 1` ecarte le groupe entier — meme
// prudence que la garde d'unanimite de Q21b / KillDistanceRepo.
//
// LES DEUX IDENTITES VIDES, ET ELLES NE SE VALENT PAS (doc corrigee le 2026-09-06,
// verifiee sur pieces cote PRODUCTEURS) :
//
//	killer_xuid vide   DEFENSIF. Le collecteur du film ne garde que les morts dont
//	                   LES DEUX identites sont resolues
//	                   (sync/killcollector/positions.go, killRefsFromDeaths) et le
//	                   persister REFUSE toute ligne sans tueur
//	                   (persist/kill_position_persister.go). La jointure ci-dessous
//	                   est en outre une EGALITE sur cette colonne. Aucun producteur
//	                   connu n'en ecrit ; le COALESCE reste pour qu'un scan douteux
//	                   ne range pas une chaine vide dans un axe par accident.
//	victim_xuid vide   REEL, et servi. Le producteur NATIF de Halo 5 ne pose que le
//	                   tueur (games/halo_5/ingest/positions.go) : sa ligne peut donc
//	                   joindre un kill-event dont la victime est un BOT
//	                   (`victim_xuid` NULL). C'est bien une position ou le joueur a
//	                   tue, et l'ecarter sous-compterait ses kills.
//
// Dans les deux cas c'est l'appelant qui tranche : une identite vide n'appartient a
// aucun axe « qui », faute d'equipe connue.
const QTacticalPositions = `
SELECT kp.match_id,
       COALESCE(kp.killer_xuid, '')     AS killer_xuid,
       COALESCE(min(e.victim_xuid), '') AS victim_xuid,
       min(kp.killer_x) AS killer_x, min(kp.killer_y) AS killer_y,
       min(kp.victim_x) AS victim_x, min(kp.victim_y) AS victim_y
FROM kill_positions_latest kp
JOIN match_kill_events_latest e
    ON e.match_id = kp.match_id
   AND e.feed_killer_xuid = kp.killer_xuid
   AND e.time_ms = kp.time_ms
WHERE kp.match_id IN (SELECT u.match_id FROM (%s) u)
  AND e.publishable
  AND kp.killer_x IS NOT NULL AND kp.killer_y IS NOT NULL
  AND kp.victim_x IS NOT NULL AND kp.victim_y IS NOT NULL
GROUP BY kp.match_id, kp.killer_xuid, kp.time_ms
HAVING count(*) = 1
ORDER BY kp.match_id, kp.time_ms`

// KillPositions rend l'univers ET les positions mesurees de ses matchs.
func (r *TacticalRepo) KillPositions(ctx context.Context, q domain.TacticalQuery) (domain.TacticalPositions, error) {
	var out domain.TacticalPositions
	// La lecture SPATIALE exige une carte : sans elle, la requete balaierait
	// `kill_positions` sur tout l'historique du joueur pour une grille qui n'a de
	// sens que carte par carte.
	if q.MapID == "" {
		return out, fmt.Errorf("TacticalRepo.KillPositions: map_id vide")
	}
	ctx, cancel := context.WithTimeout(ctx, tacticalReadTimeout)
	defer cancel()
	db, release, err := r.ouvrir(ctx, q, "KillPositions")
	if err != nil {
		return out, err
	}
	defer release()

	univ, err := r.chargerUnivers(ctx, db, q)
	if err != nil {
		return out, r.degrader(ctx, "KillPositions", err)
	}
	out.Univers = univ
	if len(univ.Matchs) == 0 {
		return out, nil
	}

	selectSQL, args := r.universSQL(q)
	rows, err := db.QueryContext(ctx, fmt.Sprintf(QTacticalPositions, selectSQL), args...)
	if err != nil {
		return out, r.degrader(ctx, "KillPositions", err)
	}
	err = scanRows(ctx, rows, "TacticalRepo.KillPositions", func(sc rowScanner) error {
		var p domain.TacticalKillPosition
		if err := sc.Scan(&p.MatchID, &p.KillerXUID, &p.VictimXUID,
			&p.KillerX, &p.KillerY, &p.VictimX, &p.VictimY); err != nil {
			return err
		}
		out.Points = append(out.Points, p)
		return nil
	})
	return out, err
}

// QTacticalEvents : le journal des morts des matchs de l'univers.
//
// Aucune jointure sur les positions : l'echange se mesure sur des INSTANTS et des
// IDENTITES, pas sur des coordonnees — exiger une position mesuree ecarterait les
// morts d'un match non decode et gonflerait le taux.
const QTacticalEvents = `
SELECT e.match_id,
       COALESCE(e.feed_killer_xuid, '') AS killer_xuid,
       COALESCE(e.victim_xuid, '')      AS victim_xuid,
       e.time_ms
FROM match_kill_events_latest e
WHERE e.match_id IN (SELECT u.match_id FROM (%s) u)
  AND e.publishable
ORDER BY e.match_id, e.time_ms, e.victim_xuid, e.feed_killer_xuid`

// KillEvents rend l'univers ET le journal des morts de ses matchs.
func (r *TacticalRepo) KillEvents(ctx context.Context, q domain.TacticalQuery) (domain.TacticalKillEvents, error) {
	var out domain.TacticalKillEvents
	ctx, cancel := context.WithTimeout(ctx, tacticalReadTimeout)
	defer cancel()
	db, release, err := r.ouvrir(ctx, q, "KillEvents")
	if err != nil {
		return out, err
	}
	defer release()

	univ, err := r.chargerUnivers(ctx, db, q)
	if err != nil {
		return out, r.degrader(ctx, "KillEvents", err)
	}
	out.Univers = univ
	if len(univ.Matchs) == 0 {
		return out, nil
	}

	selectSQL, args := r.universSQL(q)
	rows, err := db.QueryContext(ctx, fmt.Sprintf(QTacticalEvents, selectSQL), args...)
	if err != nil {
		return out, r.degrader(ctx, "KillEvents", err)
	}
	err = scanRows(ctx, rows, "TacticalRepo.KillEvents", func(sc rowScanner) error {
		var e domain.KillEvent
		if err := sc.Scan(&e.MatchID, &e.KillerXUID, &e.VictimXUID, &e.TimeMs); err != nil {
			return err
		}
		out.Events = append(out.Events, e)
		return nil
	})
	return out, err
}

// ─── HELPERS ───────────────────────────────────────────────────────────────────

// ouvrir valide la demande et prend le lecteur shared. La CARTE n'est pas exigee
// ici : seule la lecture spatiale en a besoin, et c'est elle qui la reclame
// (KillPositions) — le journal des morts se lit aussi sur toutes les cartes.
func (r *TacticalRepo) ouvrir(ctx context.Context, q domain.TacticalQuery, op string) (*sql.DB, func(), error) {
	if q.PlayerXUID == "" {
		return nil, nil, fmt.Errorf("TacticalRepo.%s: xuid vide", op)
	}
	clause := analysis.BuildNeighborsWhereClause(q.Filtre, r.modeTax.Prefixes)
	logIgnoredFilters(ctx, "TacticalRepo."+op, clause.IgnoredFilters)

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "TacticalRepo: shared reader", "op", op, "err", err)
		return nil, nil, fmt.Errorf("shared reader: %w", err)
	}
	return db, release, nil
}

// degrader traduit une table absente en ErrCapabilityNotSupported (503 propre en
// bout de chaine) et journalise tout le reste avant de le propager. Aucune erreur
// n'est avalee.
func (r *TacticalRepo) degrader(ctx context.Context, op string, err error) error {
	if isTableNotFoundErr(err) {
		slog.DebugContext(ctx, "TacticalRepo: tables du film absentes",
			"op", op, "titleSlug", r.pdb.TitleSlug, "err", err)
		return games.ErrCapabilityNotSupported
	}
	slog.ErrorContext(ctx, "TacticalRepo: requete en echec", "op", op, "err", err)
	return fmt.Errorf("TacticalRepo.%s: %w", op, err)
}

// logIgnoredFilters signale les axes de filtre ecartes (categorie de mode
// inconnue du titre, issue hors liste blanche). Ecarter en silence donnerait un
// compte de matchs inexplicable a l'ecran.
func logIgnoredFilters(ctx context.Context, op string, ignored []string) {
	if len(ignored) == 0 {
		return
	}
	slog.WarnContext(ctx, "TacticalRepo: filtres ignores",
		"op", op, "filtres", strings.Join(ignored, ","))
}

// rowScanner : la seule surface de *sql.Rows dont les lecteurs ci-dessus ont
// besoin. Permet de nommer le scan sans exposer le curseur.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanRows deroule un curseur, ferme, et rend la premiere erreur rencontree.
// Une ligne illisible est une anomalie de schema : elle est SIGNALEE puis
// propagee, jamais sautee en silence (une lecture partielle qui se presente
// comme complete fausserait tous les denominateurs).
func scanRows(ctx context.Context, rows *sql.Rows, op string, fn func(rowScanner) error) error {
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		if err := fn(rows); err != nil {
			slog.ErrorContext(ctx, "TacticalRepo: scan en echec", "op", op, "err", err)
			return fmt.Errorf("%s scan: %w", op, err)
		}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "TacticalRepo: curseur en echec", "op", op, "err", err)
		return fmt.Errorf("%s rows: %w", op, err)
	}
	return nil
}
