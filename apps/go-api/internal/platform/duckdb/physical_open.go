package duckdb

// physical_open.go — l'ouverture PHYSIQUE d'un fichier DuckDB par le cache du paquet, et le
// soin qu'elle porte pour les bases joueur.
//
// Contrat : toute connexion *sql.DB que le cache publie (première ouverture d'une clé,
// remplacement en place après un ping en échec, Reopen après invalidation) sort de
// openPhysicalSQLDB, et de lui seul (garde-rail TestOuverturePhysiqueUnique). Quand la clé est
// en écriture (`rw:`) et que le chemin a la forme d'une base joueur (title.IsPlayerDBPath),
// les séquences servant de DEFAULT y sont alignées sur le max de leurs colonnes AVANT la
// publication du handle : aucun appelant, quel que soit son chemin (sync V1 par OpenPlayerDB,
// post-sync V2 et LUSR, persisteur de lots, pool HTTP, dépôts sous bail, CLI), n'écrit dans
// une base joueur dont une séquence rendrait un id déjà pris.
//
// Une fois par ouverture physique : un emprunt du handle en cache ne repasse pas ici. Entre
// deux ouvertures physiques, le processus est le seul écrivain du fichier (ADR 0013) et une
// séquence ne recule pas ; ce qui la met en retard (séquence recréée à START 1 par une
// reconstruction, ids posés à la main, valeurs perdues au rejeu du WAL) se produit hors du
// handle ou à sa réouverture. Le soin de schéma joueur (sync.EnsurePlayerSchema) réaligne
// lui-même quand il crée une séquence.
//
// Ni le bail d'écriture (dblease) ni le cache ne sont pris par le soin : il s'exécute sous
// openDBsMu, que tient déjà l'appelant, sur une connexion que personne d'autre ne voit encore.
// Un échec d'alignement ne bloque jamais l'ouverture (ERROR + compteur, cf.
// migration.AlignSequencesBestEffort).

import (
	"context"
	"database/sql"
	"strings"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability"
)

// rwCacheKeyPrefix — préfixe des clés de cache des handles en écriture.
const rwCacheKeyPrefix = "rw:"

// playerOpenCareCounter — compteur expvar (/debug/vars, clé "levelup") des soins appliqués
// à l'ouverture physique d'une base joueur en écriture.
const playerOpenCareCounter = "duckdb_player_open_care_total"

// openPhysicalSQLDB ouvre le fichier décrit par la configuration d'ouverture de spec (DSN,
// fuseau, limites du pool, clé de cache) — connecteur + ping —, applique les limites du pool,
// puis le soin propre aux bases joueur en écriture. Ne lit ni n'écrit le handle de spec : la
// publication reste à l'appelant. Seul appelant d'openSQLDBFor.
func openPhysicalSQLDB(spec *DB) (*sql.DB, error) {
	sqlDB, err := openSQLDBFor(spec.dsn, spec.timezone, spec.op, spec.path)
	if err != nil {
		return nil, err
	}
	applyConnLimits(sqlDB, spec.maxOpenConns, spec.maxIdleConns)
	careOnPhysicalOpen(spec.cacheKey, spec.path, sqlDB)
	return sqlDB, nil
}

// careOnPhysicalOpen aligne les séquences d'une base joueur ouverte en écriture ; no-op pour
// toute autre base (partagées, metadata, sociales : leur ouverture en écriture suit chaque
// rafale du B-swap, un balayage des max y serait payé à chaque écriture) et pour une lecture.
func careOnPhysicalOpen(key, path string, sqlDB *sql.DB) {
	if !strings.HasPrefix(key, rwCacheKeyPrefix) || !title.IsPlayerDBPath(path) {
		return
	}
	migration.AlignSequencesBestEffort(context.Background(), sqlDB, "duckdb.openPhysicalSQLDB")
	observability.IncCounter(playerOpenCareCounter)
}
