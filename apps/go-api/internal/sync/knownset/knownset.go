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
// sautaient indéfiniment. Ces enrichissements ORPHELINS sont INCONNUS, et la pagination seule ne
// les reprend pas (un delta s'arrête au premier connu, souvent plus récent qu'eux). `Set.Recover`
// en porte donc au plus `OrphanRecoveryPerCycle` par chargement, que les deux moteurs récupèrent
// PAR match_id en plus de la pagination, par leur chemin normal de persistance : la sync converge
// seule, sans sync complète. Une ligne de journal agrégée par chargement et deux compteurs expvar
// titrés disent combien sont détectés (`OrphanEnrichmentsCounter`) et combien sont demandés
// (`OrphanRecoveryRequestedCounter`).
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
	"slices"
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

// OrphanRecoveryRequestedCounter : compteur expvar titré du nombre d'orphelins placés dans
// Set.Recover (demandés à la récupération par match_id), cumulé à chaque chargement.
const OrphanRecoveryRequestedCounter = "sync_known_orphan_recovery_requested_total"

// OrphanRecoveryPerCycle borne les orphelins récupérés par match_id en UN chargement, donc en un
// cycle de sync du joueur. Pourquoi borner : une base partagée restaurée plus ancienne peut
// laisser des milliers d'orphelins, et chaque récupération coûte plusieurs appels d'API ; la borne
// garde le surcoût d'un cycle de l'ordre d'un delta chargé et rattrape N orphelins en
// N/OrphanRecoveryPerCycle cycles. Ordre lexicographique des match_id : déterministe ; un orphelin
// récupéré entre au registre et sort de la liste, le cycle suivant prend les suivants.
const OrphanRecoveryPerCycle = 50

// Set : l'ensemble connu d'un joueur et les orphelins à récupérer par match_id.
type Set struct {
	// Known : les matchs connus (règle du paquet). Non nil quand Load réussit, nil en erreur.
	Known map[string]bool
	// Recover : orphelins (enrichis côté joueur, absents du registre partagé) à récupérer PAR
	// match_id ce cycle, en plus de la pagination ; au plus OrphanRecoveryPerCycle, ordre
	// lexicographique. Vide en régime normal.
	Recover []string
}

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

// Load retourne l'ensemble des matchs connus du joueur `xuid` selon la règle du paquet, et les
// orphelins à récupérer par match_id ce cycle (Set.Recover).
//
// sharedDB est obligatoire (nil → ErrSharedUnreadable). playerDB peut être nil : l'ensemble est
// alors celui des participants du xuid présents au registre, sans orphelin.
// Lecture seule sur les deux bases ; aucune écriture. En erreur, le Set rendu est vide.
func Load(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (Set, error) {
	if sharedDB == nil {
		return Set{}, fmt.Errorf("%w: aucune connexion", ErrSharedUnreadable)
	}
	xuid = strings.TrimSpace(xuid)
	if xuid == "" {
		return Set{}, ErrNoXUID
	}

	known, err := queryIDs(ctx, sharedDB, sharedKnownSQL, xuid)
	if err != nil {
		return Set{}, fmt.Errorf("%w: participants du xuid %s: %w", ErrSharedUnreadable, xuid, err)
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
		return Set{}, fmt.Errorf("%w: registre: %w", ErrSharedUnreadable, err)
	}
	var orphans []string
	for _, id := range outside {
		if inRegistry[id] {
			known[id] = true
			continue
		}
		orphans = append(orphans, id)
	}
	toRecover := reportOrphans(ctx, xuid, orphans)

	slog.DebugContext(ctx, "knownset: ensemble des matchs connus chargé",
		"xuid", xuid, "shared_participants", sharedCount,
		"enriched", len(enriched), "registry_without_participant", len(outside)-len(orphans),
		"orphans", len(orphans), "known", len(known))
	return Set{Known: known, Recover: toRecover}, nil
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

// reportOrphans choisit les orphelins demandés ce cycle (les OrphanRecoveryPerCycle premiers en
// ordre lexicographique), les journalise en UNE ligne et les compte (détectés, demandés). Rend
// les orphelins demandés ; aucun orphelin → nil, ni journal ni compteur.
func reportOrphans(ctx context.Context, xuid string, orphans []string) []string {
	if len(orphans) == 0 {
		return nil
	}
	slices.Sort(orphans)
	toRecover := slices.Clone(orphans[:min(OrphanRecoveryPerCycle, len(orphans))])
	slog.WarnContext(ctx, "knownset: enrichissements joueur sans match au registre partagé — récupération par match_id",
		"xuid", xuid, "detected", len(orphans), "requested_this_cycle", len(toRecover),
		"left_for_next_cycles", len(orphans)-len(toRecover),
		"sample", strings.Join(toRecover[:min(orphanSample, len(toRecover))], ","))
	slug := ctxkeys.TitleSlug(ctx)
	observability.AddIntT(slug, OrphanEnrichmentsCounter, int64(len(orphans)))
	observability.AddIntT(slug, OrphanRecoveryRequestedCounter, int64(len(toRecover)))
	return toRecover
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
