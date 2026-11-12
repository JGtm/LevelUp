// Package cronstore est la lecture des bases par les tâches de fond du scheduler : sondes du contrôle de
// santé des données et dates des matchs de la purge des artefacts de rejeu. Il implémente
// port.DataHealthStore et port.MatchDatesReader ; le scheduler n'a donc ni handle SQL ni requête.
//
// Toutes les lectures sont en lecture seule (OpenReadOnly ou OpenReadForQuery, jamais un écrivain). Le
// paquet n'est pas dans platform/duckdb parce qu'il compose persist, sync/skill et migration, que
// platform/duckdb n'importe pas.
package cronstore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sqlfrag"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/skill"
)

// SharedReaderFunc ouvre une lecture de la base partagée d'un titre et rend sa fonction release (à appeler une
// fois la lecture finie). En production : le fournisseur B-swap du titre.
type SharedReaderFunc func(ctx context.Context, titleSlug string) (*sql.DB, func(), error)

// Store lit les bases pour les tâches de fond.
type Store struct {
	sharedReader SharedReaderFunc
}

var (
	_ port.DataHealthStore  = (*Store)(nil)
	_ port.MatchDatesReader = (*Store)(nil)
)

// New construit le magasin. reader rend le lecteur de la base partagée de la jauge des passes ; nil : la
// jauge lit par duckdb.OpenReadForQuery, qui réutilise le handle tenu par le process s'il existe.
func New(reader SharedReaderFunc) *Store {
	return &Store{sharedReader: reader}
}

// OpenShared ouvre la base partagée par le cache de connexions du package duckdb (clé "ro:path") : l'audit ne
// fait que des SELECT, donc la lecture seule suffit, et partager l'instance évite le conflit « different
// configuration ». Le handle est compté par référence : Close() ne ferme pas la base si d'autres s'en servent.
func (s *Store) OpenShared(path string) (port.DataHealthShared, error) {
	db, err := duckdb.OpenReadOnly(path)
	if err != nil {
		return nil, err
	}
	return &sharedHandle{db: db}, nil
}

// CountPlayerProbe exécute une sonde de la base d'un joueur, ouverte en lecture seule le temps de la sonde.
func (s *Store) CountPlayerProbe(ctx context.Context, probe port.HealthProbe, playerDBPath string) (int, error) {
	if probe != port.ProbeGarbageBannerURLs {
		return 0, fmt.Errorf("cronstore: sonde %q inconnue pour une base joueur", probe)
	}
	db, err := duckdb.OpenReadOnly(playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", port.ErrDataHealthUnavailable, err)
	}
	defer func() { _ = db.Close() }() //nolint:errcheck // ref-count : best-effort
	var n int
	err = db.QueryRow(ctx, `
			SELECT COUNT(*) FROM career_progression
			WHERE banner_image_url LIKE '%/Waypoint/file/images/%'
			   OR emblem_image_url LIKE '%/Waypoint/file/images/%'
			   OR backdrop_image_url LIKE '%/Waypoint/file/images/%'
		`).Scan(&n)
	return n, err
}

// MeasurePasses mesure une table du registre de compaction par UNE lecture courte de la base partagée : un
// swap du sync n'attend au plus que cette table.
func (s *Store) MeasurePasses(ctx context.Context, titleSlug, sharedDBPath, table string) (migration.JaugePasses, bool, error) {
	db, release, err := s.lecteurShared(ctx, titleSlug, sharedDBPath)
	if err != nil {
		return migration.JaugePasses{Table: table}, false, fmt.Errorf("lecture de la base partagée: %w", err)
	}
	defer release()
	return migration.MesurerPasses(ctx, db, table)
}

func (s *Store) lecteurShared(ctx context.Context, titleSlug, sharedDBPath string) (*sql.DB, func(), error) {
	if s.sharedReader != nil {
		return s.sharedReader(ctx, titleSlug)
	}
	return duckdb.OpenReadForQuery(sharedDBPath)
}

// MatchDatesByShortID lit (forme courte -> start_time canonique) de tout le registre. OpenReadForQuery : la
// shared peut être tenue par le serveur (RO ou fenêtre RW B-swap), le handle process-wide est réutilisé —
// jamais un second open concurrent.
func (s *Store) MatchDatesByShortID(ctx context.Context, sharedPath string) (map[string]time.Time, error) {
	if _, err := os.Stat(sharedPath); err != nil {
		return nil, fmt.Errorf("shared introuvable: %w", err)
	}
	db, release, err := duckdb.OpenReadForQuery(sharedPath)
	if err != nil {
		return nil, fmt.Errorf("open shared (lecture): %w", err)
	}
	defer release()
	q := `SELECT match_id, ` + sqlfrag.SQLStartTimeCanonical("match_registry") + ` FROM match_registry`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("lecture match_registry: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]time.Time{}
	for rows.Next() {
		var id string
		var at sql.NullTime
		if err := rows.Scan(&id, &at); err != nil {
			return nil, fmt.Errorf("lecture match_registry (scan): %w", err)
		}
		if at.Valid {
			out[titlePkg.FilmShortMatchID(id)] = at.Time
		}
	}
	return out, rows.Err()
}

// sharedHandle : la base partagée d'un titre, ouverte en lecture seule pour un audit.
type sharedHandle struct {
	db *duckdb.DB
}

// Close rend le handle (compté par référence).
func (h *sharedHandle) Close() error { return h.db.Close() }

// Bits du bitfield match_registry.backfill_completed que les sondes « bits menteurs » contrôlent.
const (
	mbitEvents      = 1 << 16
	mbitWeaponKills = 1 << 21
	// uuidPattern : un identifiant de carte ou de paire jamais résolu en nom.
	uuidPattern = `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
)

// CountProbe exécute une sonde COUNT de la base partagée.
func (h *sharedHandle) CountProbe(ctx context.Context, probe port.HealthProbe) (int, bool, error) {
	var query string
	var args []any
	switch probe {
	case port.ProbeUUIDsMapName:
		query, args = `SELECT COUNT(*) FROM match_registry WHERE map_name ~ ?`, []any{uuidPattern}
	case port.ProbeUUIDsPairName:
		query, args = `SELECT COUNT(*) FROM match_registry WHERE pair_name ~ ?`, []any{uuidPattern}
	case port.ProbeLyingBitsEvents:
		query = fmt.Sprintf(`
		SELECT COUNT(*) FROM match_registry r
		WHERE (COALESCE(r.backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = r.match_id)
	`, mbitEvents)
	case port.ProbeLyingBitsWeapons:
		// La PREUVE du detail des armes change de table selon le titre : `weapon_kills` la porte encore la ou
		// l arme est native de l API, `match_kill_events_latest` la porte la ou elle vient du film. On sonde
		// celle que la base CONTIENT, jamais un slug.
		evidence := duckdb.WeaponEvidenceTable(ctx, h.db.SQLDb())
		if evidence == "" {
			return 0, false, nil
		}
		query = fmt.Sprintf(`
		SELECT COUNT(*) FROM match_registry r
		WHERE (COALESCE(r.backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM %s w WHERE w.match_id = r.match_id)
	`, mbitWeaponKills, evidence)
	case port.ProbeOrphanXUIDs:
		query = `
		SELECT COUNT(DISTINCT mp.xuid)
		FROM match_participants mp
		LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid
		WHERE ` + sqlfrag.SQLIsNotBotCol("mp.xuid") + `
		  AND (xa.xuid IS NULL OR xa.gamertag IS NULL OR xa.gamertag = '')
	`
	default:
		return 0, false, fmt.Errorf("cronstore: sonde %q inconnue pour la base partagée", probe)
	}
	var n int
	err := h.db.QueryRow(ctx, query, args...).Scan(&n)
	return n, true, err
}

// PendingKillSummaries compte les matchs sans résumés à jour (même texte SQL que `levelup
// backfill-match-summaries`, persist.CountPendingKillSummaries).
func (h *sharedHandle) PendingKillSummaries(ctx context.Context) (int, error) {
	return persist.CountPendingKillSummaries(ctx, h.db.SQLDb())
}

// ScanLUSRGaps ouvre la base du joueur en lecture seule le temps du scan (skill.ScanLUSRGaps).
func (h *sharedHandle) ScanLUSRGaps(ctx context.Context, playerDBPath, xuid string) (port.LUSRGaps, error) {
	pdb, err := duckdb.OpenReadOnly(playerDBPath)
	if err != nil {
		return port.LUSRGaps{}, fmt.Errorf("%w: %w", port.ErrDataHealthUnavailable, err)
	}
	defer func() { _ = pdb.Close() }() //nolint:errcheck // ref-count : best-effort
	rep, err := skill.ScanLUSRGaps(ctx, pdb.SQLDb(), h.db.SQLDb(), xuid)
	if err != nil {
		return port.LUSRGaps{}, err
	}
	return port.LUSRGaps{InteriorGaps: rep.TotalInteriorGaps, PendingRecent: rep.TotalPendingRecent}, nil
}
