// Package knownset — l'ensemble des matchs « déjà connus » d'un joueur, LA règle qui décide,
// pour la sync Halo Infinite (moteur V1 `SyncEngine` et pipeline V2 `KnownLoader`), si un match
// de l'historique est re-récupéré ou sauté. Delta : arrêt au premier connu ; full : un connu est
// sauté. Ce paquet est la SEULE construction de cet ensemble : garde-rail
// `internal/archlint/known_set_single_source_test.go`.
//
// RÈGLE. Un match est connu SEULEMENT si la base partagée le contient pour ce joueur, au sens où
// la persistance le considère déjà écrit : sa ligne `match_registry` existe. L'ensemble vaut
//
//	match_registry ∩ (participants du xuid ∪ enrichissements du joueur)
//
// Pourquoi le registre : `persist.SharedPersister` écrit registre, participants et le reste dans
// UNE transaction, et ne saute un match que si sa ligne `match_registry` existe. Un match absent
// du registre est donc réparable par un nouveau fetch ; un match présent ne l'est pas (le fetch
// serait jeté). Les participants du xuid et les enrichissements ne servent qu'à BORNER
// l'ensemble aux matchs du joueur — le registre entier couvre tous les joueurs.
//
// Pourquoi pas les enrichissements seuls : la base joueur peut survivre à la base partagée (base
// partagée restaurée depuis une copie plus ancienne). Un enrichissement sans ligne au registre
// désignait alors un match « connu » que la base partagée n'avait pas : delta et full le
// sautaient indéfiniment. Ces enrichissements orphelins sont désormais INCONNUS (re-récupérés),
// journalisés en une ligne agrégée par appel et comptés (compteur expvar titré
// `OrphanEnrichmentsCounter`).
//
// ÉCHEC. Base partagée absente ou illisible, ou xuid vide : erreur typée
// (`ErrSharedUnreadable`, `ErrNoXUID`), jamais un ensemble partiel. Un ensemble vide ferait tout
// retélécharger, un ensemble « enrichissements seuls » ferait sauter des matchs absents ; la sync
// s'arrête et retente au cycle suivant. Les enrichissements illisibles (base joueur neuve, vue
// absente) ne bloquent pas : la règle ne dépend que de la base partagée, ils ne servent qu'au
// bornage des matchs du registre sans ligne de participant du xuid et au décompte des orphelins.
//
// Paquet à part (ADR 0027 / ratchet K3c `sync_root_freeze_test.go`) : `internal/sync` est gelé,
// et le pipeline V2 l'importe déjà ; les deux moteurs appellent `Load`.
package knownset

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
)

// ErrSharedUnreadable : la base partagée n'a pas pu être lue (connexion absente, requête ou
// lecture en échec). L'ensemble connu est indéterminable ; l'appelant arrête la sync du joueur.
var ErrSharedUnreadable = errors.New("knownset: base partagée illisible, ensemble des matchs connus indéterminable")

// ErrNoXUID : le xuid du joueur est vide ; les participants de la base partagée ne peuvent pas
// être bornés au joueur. Même conséquence que ErrSharedUnreadable.
var ErrNoXUID = errors.New("knownset: xuid vide, ensemble des matchs connus indéterminable")

// OrphanEnrichmentsCounter : compteur expvar titré (observability.AddIntT) du nombre
// d'enrichissements joueur sans ligne au registre partagé, cumulé à chaque chargement.
const OrphanEnrichmentsCounter = "sync_known_orphan_enrichments_total"

// registryChunk : nombre de match_id par requête `IN (...)` de vérification au registre.
const registryChunk = 500

// orphanSample : nombre de match_id orphelins cités dans la ligne de journal agrégée.
const orphanSample = 5

// sharedKnownSQL : participants du xuid dont le match a sa ligne au registre. Le cast texte du xuid
// (concaténation à la chaîne vide) est le prédicat commun aux lecteurs de match_participants par xuid
// (`ensurePlayerEnrichmentRows`, `countSharedMatchesMissingEnrichment`) : une dérive de type
// de la colonne (VARCHAR / UBIGINT) ne doit pas vider l'ensemble en silence.
const sharedKnownSQL = `SELECT DISTINCT mp.match_id
FROM match_participants mp
WHERE mp.xuid || '' = ?
  AND mp.match_id IS NOT NULL
  AND EXISTS (SELECT 1 FROM match_registry mr WHERE mr.match_id = mp.match_id)`

// enrichmentSQL : les matchs enrichis du joueur (vue append-only, règle ART n° 2).
const enrichmentSQL = `SELECT match_id FROM player_match_enrichment_latest`

// Load retourne l'ensemble des matchs connus du joueur `xuid` selon la règle du paquet.
//
// sharedDB est obligatoire (nil → ErrSharedUnreadable). playerDB peut être nil : l'ensemble est
// alors celui des participants du xuid présents au registre, sans décompte des orphelins.
// Lecture seule sur les deux bases ; aucune écriture.
func Load(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (map[string]bool, error) {
	if sharedDB == nil {
		return nil, fmt.Errorf("%w: aucune connexion", ErrSharedUnreadable)
	}
	xuid = strings.TrimSpace(xuid)
	if xuid == "" {
		return nil, ErrNoXUID
	}

	known, err := queryIDs(ctx, sharedDB, sharedKnownSQL, xuid)
	if err != nil {
		return nil, fmt.Errorf("%w: participants du xuid %s: %w", ErrSharedUnreadable, xuid, err)
	}
	sharedCount := len(known)

	enriched := loadEnrichments(ctx, playerDB, xuid)
	var outside []string
	for _, id := range enriched {
		if !known[id] {
			outside = append(outside, id)
		}
	}
	inRegistry, err := filterInRegistry(ctx, sharedDB, outside)
	if err != nil {
		return nil, fmt.Errorf("%w: registre: %w", ErrSharedUnreadable, err)
	}
	var orphans []string
	for _, id := range outside {
		if inRegistry[id] {
			known[id] = true
			continue
		}
		orphans = append(orphans, id)
	}
	reportOrphans(ctx, xuid, orphans)

	slog.DebugContext(ctx, "knownset: ensemble des matchs connus chargé",
		"xuid", xuid, "shared_participants", sharedCount,
		"enriched", len(enriched), "registry_without_participant", len(outside)-len(orphans),
		"orphans", len(orphans), "known", len(known))
	return known, nil
}

// loadEnrichments lit les match_id enrichis du joueur. Un échec (base joueur neuve, vue
// absente) est journalisé et rend une liste vide : la règle ne dépend que de la base partagée.
func loadEnrichments(ctx context.Context, playerDB *sql.DB, xuid string) []string {
	if playerDB == nil {
		return nil
	}
	set, err := queryIDs(ctx, playerDB, enrichmentSQL)
	if err != nil {
		slog.WarnContext(ctx, "knownset: enrichissements joueur illisibles — orphelins non décomptés, ensemble connu = base partagée",
			"xuid", xuid, "err", err)
		return nil
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	return out
}

// filterInRegistry retourne, parmi ids, ceux qui ont une ligne au registre partagé. Requêtes
// `IN (...)` par paquets de registryChunk ; ids vide → aucune requête.
func filterInRegistry(ctx context.Context, sharedDB *sql.DB, ids []string) (map[string]bool, error) {
	present := make(map[string]bool, len(ids))
	for start := 0; start < len(ids); start += registryChunk {
		end := min(start+registryChunk, len(ids))
		chunk := ids[start:end]
		args := make([]any, len(chunk))
		for i, id := range chunk {
			args[i] = id
		}
		query := "SELECT match_id FROM match_registry WHERE match_id IN (?" +
			strings.Repeat(", ?", len(chunk)-1) + ")"
		got, err := queryIDs(ctx, sharedDB, query, args...)
		if err != nil {
			return nil, err
		}
		for id := range got {
			present[id] = true
		}
	}
	return present, nil
}

// reportOrphans journalise en UNE ligne et compte les enrichissements sans ligne au registre.
func reportOrphans(ctx context.Context, xuid string, orphans []string) {
	if len(orphans) == 0 {
		return
	}
	sample := orphans[:min(orphanSample, len(orphans))]
	slog.WarnContext(ctx, "knownset: enrichissements joueur sans match au registre partagé — traités comme inconnus, re-récupérés",
		"xuid", xuid, "orphans", len(orphans), "sample", strings.Join(sample, ","))
	observability.AddIntT(ctxkeys.TitleSlug(ctx), OrphanEnrichmentsCounter, int64(len(orphans)))
}

// queryIDs exécute une requête à une colonne match_id et rend l'ensemble lu. Toute erreur
// (requête, lecture d'une ligne, itération) est rendue : un ensemble tronqué n'est jamais servi.
func queryIDs(ctx context.Context, db *sql.DB, query string, args ...any) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make(map[string]bool, 512)
	for rows.Next() {
		var id sql.NullString
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if id.Valid && id.String != "" {
			out[id.String] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
