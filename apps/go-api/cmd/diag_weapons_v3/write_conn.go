package main

// write_conn.go — ouverture RW SÛRE pour le -write du mode POSITIONS.
//
// Contrat (cf. tâche couche 5) : -write ne touche JAMAIS la vraie shared DB si un
// serveur la verrouille en écriture. On tente d'abord OpenReadWrite (lock exclusif
// DuckDB) ; si le lock échoue (serveur up), on travaille sur une COPIE temp de la
// DB (mktemp), qu'on supprime à la fermeture — comme le CLI objective-events l'a
// fait. La copie sert de validation du chemin d'écriture sans risque prod.
//
// La table visée est shadow/additive : écrire sur la vraie DB n'est risqué QUE par
// le lock, d'où ce garde-fou.
//
// RENOMMÉ de `weapons_write.go` le 2026-09-01 : le mode ARMES qu'il servait à
// l'origine est parti avec la table `weapon_kills_v3` (lot arme-source-unique). Le
// garde-fou, lui, sert toujours le mode POSITIONS.

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"levelup/go-api/internal/platform/duckdb"
)

// openShadowWriteConn ouvre la DB en RW pour le -write shadow. Si le lock exclusif
// échoue (serveur up), bascule sur une copie temp (supprimée à la fermeture). Le
// flag *toTemp* informe l'appelant que l'écriture est partie sur une copie jetable.
func openShadowWriteConn(dbPath string) (c *conn, toTemp bool, err error) {
	rw, rwErr := openLockedRW(dbPath)
	if rwErr == nil {
		return &conn{sqlDB: rw.SQLDb(), rwDB: rw, release: func() { _ = rw.Close() }}, false, nil
	}

	tmp, copyErr := copyDBToTemp(dbPath)
	if copyErr != nil {
		return nil, false, fmt.Errorf("RW lock échoué (%v) et copie temp impossible: %w", rwErr, copyErr)
	}
	rwTmp, tmpErr := duckdb.OpenReadWrite(tmp)
	if tmpErr != nil {
		_ = os.Remove(tmp)
		return nil, false, fmt.Errorf("open RW copie temp %s: %w", tmp, tmpErr)
	}
	return &conn{
		sqlDB: rwTmp.SQLDb(),
		rwDB:  rwTmp,
		release: func() {
			_ = rwTmp.Close()
			_ = os.Remove(tmp)
		},
	}, true, nil
}

// openLockedRW ouvre la DB en RW et FORCE l'acquisition du lock via un ping : le
// lock DuckDB est lazy (à la 1re requête), donc OpenReadWrite seul ne révèle pas un
// serveur concurrent. Le ping force l'erreur "Conflicting lock" si elle doit venir.
func openLockedRW(dbPath string) (*duckdb.DB, error) {
	rw, err := duckdb.OpenReadWrite(dbPath)
	if err != nil {
		return nil, err
	}
	if pingErr := rw.SQLDb().PingContext(context.Background()); pingErr != nil {
		_ = rw.Close()
		return nil, pingErr
	}
	// Sonde d'écriture : un SELECT trivial ne déclenche pas forcément le write-lock.
	if _, qErr := rw.SQLDb().ExecContext(context.Background(),
		`CREATE TEMP TABLE IF NOT EXISTS _diag_v3_lockprobe(x INTEGER)`); qErr != nil {
		_ = rw.Close()
		return nil, qErr
	}
	return rw, nil
}

// copyDBToTemp copie la DB vers un fichier temp unique (os.CreateTemp). Renvoie le
// chemin de la copie ; l'appelant la supprime à la fermeture.
func copyDBToTemp(dbPath string) (string, error) {
	src, err := os.Open(dbPath) //nolint:gosec // chemin fourni par l'opérateur du CLI
	if err != nil {
		return "", fmt.Errorf("open source %s: %w", dbPath, err)
	}
	defer src.Close()

	f, err := os.CreateTemp("", "shared_v3_*"+filepath.Ext(dbPath))
	if err != nil {
		return "", fmt.Errorf("createtemp: %w", err)
	}
	tmpPath := f.Name()
	if _, err := io.Copy(f, src); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("copy vers %s: %w", tmpPath, err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("close temp: %w", err)
	}
	return tmpPath, nil
}
