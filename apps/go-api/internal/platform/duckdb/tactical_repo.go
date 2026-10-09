// Package duckdb — tactical_repo.go : implementation DuckDB de
// port.TacticalRepository (onglet Tactique, plan .ai/V7.5/PLAN_TACTIQUE_2026-09-06.md
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
//   - le journal des morts (KillEvents) rend chaque mort avec sa victime et son tueur
//     credite — une identite permutee attribue la mort au mauvais joueur.
//
// Une passe non publiable est donc ECARTEE ici, comme dans KillDistanceRepo, et
// contrairement a KillSourceClassRepo (qui, lui, ne produit que des cumuls).
//
// ─── AUCUN SCAN ────────────────────────────────────────────────────────────────
//
// Toute lecture est bornee par le joueur (`mp.xuid = ?`). La lecture SPATIALE
// (KillPositions) l'est en plus par une CARTE : un xuid vide ou une carte vide y
// sont un REFUS, jamais un balayage de `shared.kill_positions` en entier. Le
// PERIMETRE (liste blanche de match_id, composition) vient de l'appelant et se pose
// au meme endroit pour les trois lectures — cf. tactical_repo_univers.go.
//
// KillEvents, elle, accepte une carte VIDE : son appelant de production, le bloc de
// coordination des pages Sessions et Series temporelles (service/coordination_block.go),
// lit une LISTE de matchs (`RestreindreAux`), qui n'a pas de carte. Le SELECT reste le
// meme, la carte devenant un parametre neutre (`? = ” OR mr.map_id = ?`). La borne
// reste le joueur, jamais la table entiere. Une lecture SANS liste (zero-value de
// ListeBlancheMatchs) reste acceptee mais n'a pas d'appelant de production a ce jour.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// tacticalReadTimeout borne une lecture tactique. Plus large que les 15 s de
// KillDistanceRepo : la fenetre porte sur des centaines de matchs, pas un seul.
const tacticalReadTimeout = 30 * time.Second

// TacticalRepo implemente port.TacticalRepository.
//
// AUCUNE TAXONOMIE DE MODES ICI (retrait phase 4 bis, 2026-09-06) : ce lecteur ne
// classe plus rien, il applique une liste blanche de match_id. La cascade
// playlists / modes est resolue en amont, sur la base joueur, par le meme
// pipeline que l'Explorateur.
type TacticalRepo struct {
	pdb *PlayerDB
}

// NewTacticalRepo cree un TacticalRepo lie a un PlayerDB.
func NewTacticalRepo(pdb *PlayerDB) *TacticalRepo {
	return &TacticalRepo{pdb: pdb}
}

// ─── LES TROIS LECTURES ────────────────────────────────────────────────────────

// QTacticalMaps : les cartes JOUEES par le joueur dans le perimetre, UNE rangee par map_id.
//
// Les codes d'issue sont des PARAMETRES LIES (domain.OutcomeWin / OutcomeLoss),
// jamais des litteraux dans la chaine SQL — un `outcome = 2` en dur est
// exactement ce que le ratchet no_raw_outcome_literal interdit.
//
// Meme token Campagne que QTacticalUnivers, et pour la meme raison : sans lui, la
// grille d'entree d'un joueur Halo 5 affichait ses cartes de Campagne a cote de
// ses cartes d'arene.
//
// LA CLE EST LE SEUL map_id. Le nom du registre varie d'un match a l'autre pour une meme
// carte (vrai nom, NULL, ou map_id recopie quand la sync n'avait pas encore la traduction) :
// grouper aussi sur lui scinde une carte en plusieurs rangees, dont chacune porte une part
// du compte — et le plancher de matchs du service s'applique alors a une part. Le nom
// retenu est celui du match le plus recent qui en porte un VRAI (ni NULL, ni vide, ni le
// map_id) ; a defaut, vide, et le libelle vient de la traduction de l'asset
// (poserLesLibelles). Garde-rail : archlint/no_group_by_registry_name_test.go.
var QTacticalMaps = `
SELECT mr.map_id,
       ` + nomDeCarteRetenuSQL("map_name") + ` AS map_name,
       ` + nomDeCarteRetenuSQL("map_name_fr") + ` AS map_name_fr,
       COUNT(*)                          AS matchs,
       COUNT(*) FILTER (WHERE mp.outcome = ?) AS victoires,
       COUNT(*) FILTER (WHERE mp.outcome = ?) AS defaites
FROM match_registry mr
JOIN match_participants mp ON mp.match_id = mr.match_id
WHERE mp.xuid = ? AND mr.map_id IS NOT NULL AND mr.map_id <> ''` + clausePvEExclu + campaignExclusionToken

// nomDeCarteRetenuSQL rend l'agregat qui choisit, pour un groupe de matchs d'une meme carte
// (alias `mr`), le nom porte par la colonne `col` au match le plus recent (debut canonique)
// parmi ceux qui en ont un vrai. Un match sans horodatage n'entre pas dans arg_max ; le MAX
// filtre le relaie quand AUCUN match nomme n'est horodate. Jamais NULL : chaine vide a defaut.
func nomDeCarteRetenuSQL(col string) string {
	vrai := "mr." + col + " IS NOT NULL AND mr." + col + " <> '' AND mr." + col + " <> mr.map_id"
	return "COALESCE(arg_max(mr." + col + ", " + StartTimeCanonicalSQL("mr") + ") FILTER (WHERE " + vrai +
		"), MAX(mr." + col + ") FILTER (WHERE " + vrai + "), '')"
}

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

	perim, perimArgs := clausePerimetre(q)
	args := append([]any{domain.OutcomeWin, domain.OutcomeLoss, q.PlayerXUID}, perimArgs...)
	query := resolveCampaignExclusion(QTacticalMaps, r.pdb.TitleSlug, "mr") + perim +
		` GROUP BY mr.map_id ORDER BY matchs DESC, mr.map_id`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, r.degrader(ctx, "MapsPlayed", err)
	}
	out := make([]domain.TacticalMapRow, 0)
	err = scanRows(ctx, rows, "TacticalRepo.MapsPlayed", func(sc rowScanner) error {
		var row domain.TacticalMapRow
		if err := sc.Scan(&row.MapID, &row.MapName, &row.MapNameFR,
			&row.Matchs, &row.Victoires, &row.Defaites); err != nil {
			return err
		}
		out = append(out, row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	r.poserLesLibelles(ctx, out)
	return out, nil
}

// poserLesLibelles remplace le nom FR du registre de chaque carte par son LIBELLÉ CANONIQUE
// (libelleDeCarte, map_labels.go), traductions lues en une requête : la vignette et le lien vers
// l'Explorateur portent ainsi la chaîne exacte que l'Explorateur compare.
func (r *TacticalRepo) poserLesLibelles(ctx context.Context, rows []domain.TacticalMapRow) {
	if len(rows) == 0 {
		return
	}
	ids := make([]string, len(rows))
	for i := range rows {
		ids[i] = rows[i].MapID
	}
	var traductions map[string]string
	if r.pdb != nil {
		traductions = traductionsDeCartes(ctx, r.pdb.Metadata, ids)
	}
	for i := range rows {
		rows[i].MapNameFR = libelleDeCarte(rows[i].MapNameFR, rows[i].MapName, traductions[rows[i].MapID])
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
// `kp.time_ms` VOYAGE DANS LE SELECT (ajout lot M1, 2026-09-07) : c'est l'INSTANT
// CONTRIBUTEUR qu'un detail de cellule doit pouvoir citer pour ouvrir le rejeu 2D au bon
// moment (`?frame=`). Il fait partie de la clef du GROUP BY, donc le projeter ne change ni
// le nombre de lignes ni les gardes ci-dessus — seule une colonne de plus est lue.
//
// LES HAUTEURS, LES NOMS DU KILL-FEED ET LA SOURCE DE DEGAT voyagent aussi (detail d'une zone :
// nom en jeu, mini-tuile) : memes lignes, memes gardes — le groupe n'a qu'une ligne (HAVING), donc
// `min` rend la valeur. Une hauteur ou une source NULL reste NULL (jamais 0).
//
// %s = la table de positions, puis DEUX FOIS la liste des matchs de l'univers : une par vue.
// La liste posee sur `kp` ne descend pas dans `e` a travers la jointure (lot L5a, mesure :
// cf. listeDeLUnivers) — chaque vue `_latest` porte donc la sienne.
const QTacticalPositions = `
SELECT kp.match_id,
       COALESCE(kp.killer_xuid, '')     AS killer_xuid,
       COALESCE(min(e.victim_xuid), '') AS victim_xuid,
       min(kp.killer_x) AS killer_x, min(kp.killer_y) AS killer_y,
       min(kp.victim_x) AS victim_x, min(kp.victim_y) AS victim_y,
       kp.time_ms AS time_ms,
       min(kp.killer_z) AS killer_z, min(kp.victim_z) AS victim_z,
       COALESCE(min(e.feed_killer_gamertag), '') AS killer_gamertag,
       COALESCE(min(e.victim_gamertag), '')      AS victim_gamertag,
       min(e.source_tag) AS source_tag,
       COALESCE(min(e.source_category), '')      AS source_category
FROM %s kp
JOIN match_kill_events_latest e
    ON e.match_id = kp.match_id
   AND e.feed_killer_xuid = kp.killer_xuid
   AND e.time_ms = kp.time_ms
WHERE kp.match_id IN (%s)
  AND e.match_id IN (%s)
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

	liste, args := listeDeLUnivers(univ, 2)
	// Le nom de la table de positions passe par la constante de kill_measured.go, PAS par un
	// littéral ici : c'est le seul propriétaire du nom (garde-rail
	// kill_measured_guard_test.go, lot 3 v75 du 2026-09-06). Cette lecture n'emprunte PAS
	// measuredKillsQuery (pas de classificateur d'arme, pas de garde d'unanimité — cf.
	// l'en-tête du fichier) ; seul le NOM de la table est partagé.
	rows, err := db.QueryContext(ctx, fmt.Sprintf(QTacticalPositions, positionsAtKill, liste, liste), args...)
	if err != nil {
		return out, r.degrader(ctx, "KillPositions", err)
	}
	err = scanRows(ctx, rows, "TacticalRepo.KillPositions", func(sc rowScanner) error {
		var p domain.TacticalKillPosition
		var kz, vz sql.NullFloat64
		var tag sql.NullInt64
		if err := sc.Scan(&p.MatchID, &p.KillerXUID, &p.VictimXUID,
			&p.KillerX, &p.KillerY, &p.VictimX, &p.VictimY, &p.TimeMs,
			&kz, &vz, &p.KillerGamertag, &p.VictimGamertag, &tag, &p.SourceCategory); err != nil {
			return err
		}
		p.KillerZ, p.VictimZ, p.SourceTag = nullFloatPtr(kz), nullFloatPtr(vz), tagOuNil(tag)
		out.Points = append(out.Points, p)
		return nil
	})
	return out, err
}

// tagOuNil : la source de degat (UINTEGER nullable), NULL servi nil.
func tagOuNil(v sql.NullInt64) *uint32 {
	if !v.Valid {
		return nil
	}
	u := uint32(v.Int64)
	return &u
}

// QTacticalEvents : le journal des morts des matchs de l'univers.
//
// Aucune jointure sur les positions : le journal porte des INSTANTS et des IDENTITES,
// pas des coordonnees — exiger une position mesuree ecarterait les morts d'un match non
// decode.
//
// %s = la liste des matchs de l'univers (listeDeLUnivers) : une liste de constantes, que
// DuckDB pousse sous la fenetre de la vue (0,74 s -> 0,05 s pour 6 matchs, mesure lot L5a).
const QTacticalEvents = `
SELECT e.match_id,
       COALESCE(e.feed_killer_xuid, '') AS killer_xuid,
       COALESCE(e.victim_xuid, '')      AS victim_xuid,
       e.time_ms
FROM match_kill_events_latest e
WHERE e.match_id IN (%s)
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

	liste, args := listeDeLUnivers(univ, 1)
	rows, err := db.QueryContext(ctx, fmt.Sprintf(QTacticalEvents, liste), args...)
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
