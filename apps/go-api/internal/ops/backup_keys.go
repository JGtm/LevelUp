package ops

import (
	"context"
	"database/sql"
	"strings"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
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

// FinalizeRestoredDB finalise une base restaurée (duckdbbackup.FinalizeFunc) : chaque séquence
// est alignée sur les identifiants restaurés, pour ne jamais rendre une valeur déjà prise.
func FinalizeRestoredDB(ctx context.Context, _ string, db *sql.DB) error {
	_, err := migration.AlignSequencesToColumns(ctx, db)
	return err
}

// BackupAliasesKey : la clé de la base globale des alias Xbox (hors des bases par titre).
const BackupAliasesKey = "xbox_aliases"

// BackupDB : une base DuckDB que la sauvegarde en dossier protège, sa clé et son chemin de production.
type BackupDB struct {
	Key  string
	Path string
}

// BackupDatabases liste les bases à protéger, toutes celles que le PathResolver désigne et qui existent :
// les quatre bases d'entrepôt de chaque titre présent sous la racine des titres, une base par joueur, puis
// la base globale des alias quand elle existe. Aucun chemin n'est écrit à la main : la restauration
// retrouve chaque chemin par la clé ([ParseBackupKey], [BackupDBPath]).
func BackupDatabases(pr *title.PathResolver) ([]BackupDB, error) {
	targets, err := discoverLevelUpDBs(pr)
	if err != nil {
		return nil, err
	}
	out := make([]BackupDB, 0, len(targets)+1)
	for _, t := range targets {
		out = append(out, BackupDB{Key: t.Key, Path: t.Path})
	}
	if p := pr.GlobalXuidAliasesDBPath(); fileExists(p) {
		out = append(out, BackupDB{Key: BackupAliasesKey, Path: p})
	}
	return out, nil
}

// BackupDBPath rend le chemin de production de la base d'une clé de sauvegarde, "" pour une clé inconnue.
func BackupDBPath(pr *title.PathResolver, key string) string {
	if key == BackupAliasesKey {
		return pr.GlobalXuidAliasesDBPath()
	}
	if k, ok := ParseBackupKey(key); ok {
		return k.LivePath(pr)
	}
	return ""
}
