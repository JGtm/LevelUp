package ops

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/pkg/duckdbbackup"
)

// Format des clés de sauvegarde (une clé = un fichier DuckDB) : `<titre>:<base>` pour une base
// d'entrepôt, `<titre>:player:<gamertag>` pour une base joueur. La clé fixe aussi l'arborescence
// du staging (`<titre>/<base>/`, `<titre>/player/<gamertag>/`). Source unique du format : les
// constantes ci-dessous, lues par la découverte (discoverLevelUpDBs) et par la restauration.
const (
	backupKeySep        = ":"
	backupDBShared      = "shared_matches_v2"
	backupDBMetadata    = "metadata"
	backupDBSharedPvE   = "shared_pve"
	backupDBSharedSoc   = "shared_social"
	backupPlayerSegment = "player"
)

// backupWarehouseTargets : base d'entrepôt (segment de clé) → cible de migration.
var backupWarehouseTargets = map[string]migration.TargetDB{
	backupDBShared:    migration.TargetShared,
	backupDBMetadata:  migration.TargetMetadata,
	backupDBSharedPvE: migration.TargetSharedPvE,
	backupDBSharedSoc: migration.TargetSharedSocial,
}

// BackupKey est une clé de sauvegarde décodée.
type BackupKey struct {
	Slug     string
	Target   migration.TargetDB
	Gamertag string // base joueur seulement
}

// ParseBackupKey décode une clé de sauvegarde. ok=false pour une clé hors format (ex.
// `xbox_aliases`, base globale retirée des sauvegardes, présente dans les anciens instantanés).
func ParseBackupKey(key string) (BackupKey, bool) {
	parts := strings.Split(key, backupKeySep)
	switch {
	case len(parts) == 2 && parts[0] != "":
		target, ok := backupWarehouseTargets[parts[1]]
		return BackupKey{Slug: parts[0], Target: target}, ok
	case len(parts) == 3 && parts[0] != "" && parts[1] == backupPlayerSegment && parts[2] != "":
		return BackupKey{Slug: parts[0], Target: migration.TargetPlayer, Gamertag: parts[2]}, true
	}
	return BackupKey{}, false
}

// warehouseBackupKey construit la clé d'une base d'entrepôt.
func warehouseBackupKey(slug, db string) string { return slug + backupKeySep + db }

// playerBackupKey construit la clé d'une base joueur.
func playerBackupKey(slug, gamertag string) string {
	return slug + backupKeySep + backupPlayerSegment + backupKeySep + gamertag
}

// LivePath rend le chemin de PRODUCTION de la base que désigne la clé.
func (k BackupKey) LivePath(pr *title.PathResolver) string {
	switch k.Target {
	case migration.TargetShared:
		return pr.SharedDBPath(k.Slug)
	case migration.TargetMetadata:
		return pr.MetadataDBPath(k.Slug)
	case migration.TargetSharedPvE:
		return pr.SharedPVEDBPath(k.Slug)
	case migration.TargetSharedSocial:
		return pr.SharedSocialDBPath(k.Slug)
	case migration.TargetPlayer:
		return pr.PlayerDBPath(k.Slug, k.Gamertag)
	}
	return ""
}

// RestoreSchema est le schéma canonique d'une base restaurée (duckdbbackup.CanonicalSchema) :
// le jeu de migrations de son titre et de sa cible, celui que l'application applique à
// l'ouverture (migration.RunForTitleDB). L'appelant câble les étapes title-owned
// (titleseams.RegisterAll) avant l'import, comme le serveur.
type RestoreSchema struct{}

// Provision applique le jeu de migrations de la base à la base neuve.
func (RestoreSchema) Provision(_ context.Context, key string, db *sql.DB) error {
	k, ok := ParseBackupKey(key)
	if !ok {
		return fmt.Errorf("%w: %q", duckdbbackup.ErrNoCanonicalSchema, key)
	}
	return migration.RunForTitleDB(db, k.Slug, k.Target)
}

// Finalize aligne les séquences sur les identifiants restaurés : une séquence repartie de son
// départ rendrait des identifiants déjà pris.
func (RestoreSchema) Finalize(ctx context.Context, _ string, db *sql.DB) error {
	_, err := migration.AlignSequencesToColumns(ctx, db)
	return err
}
