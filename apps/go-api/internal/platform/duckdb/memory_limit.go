package duckdb

// memory_limit.go : le plafond mémoire des bases DuckDB posé par le réglage `memory_limit_mb` (profil
// desktop, lot L22). C'est DuckDB qui consomme la mémoire du processus : le levier est le `memory_limit` de
// chaque base, pas une limite du système (aucun arrêt forcé, aucun Job object).
//
// Le plafond s'applique à CHAQUE ouverture de connexion (applyDuckSessionInit lit [duckMemoryLimit]) et,
// quand le réglage change à chaud, aux bases déjà ouvertes ([ApplyMemoryLimitToOpen]). Zéro = pas de
// plafond posé par le réglage : la valeur par défaut du serveur (LEVELUP_DUCKDB_MEMORY_LIMIT, sinon 512MB)
// reste celle de chaque connexion.

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"
)

// memoryLimitMB : le plafond du réglage, en Mo ; 0 = aucun.
var memoryLimitMB atomic.Int64

// SetMemoryLimitMB pose le plafond du réglage (0 = le défaut du serveur) pour les connexions à venir.
func SetMemoryLimitMB(mb int) { memoryLimitMB.Store(int64(mb)) }

// MemoryLimitMB rend le plafond posé par le réglage (0 = aucun).
func MemoryLimitMB() int { return int(memoryLimitMB.Load()) }

// EffectiveMemoryLimit rend le plafond que reçoit une nouvelle connexion (« 4096MiB », « 512MB »).
func EffectiveMemoryLimit() string { return duckMemoryLimit() }

// applyLimitTimeout borne le SET d'une base : il ne touche aucune donnée et rend aussitôt.
const applyLimitTimeout = 5 * time.Second

// ApplyMemoryLimitToOpen exécute `SET memory_limit` sur chaque base du cache process-wide (le plafond est un
// réglage de l'instance de base, commun à ses connexions). Un échec est journalisé et compté dans le
// rapport : il n'empêche pas les autres bases. Rend le nombre de bases mises à jour et en échec.
func ApplyMemoryLimitToOpen(ctx context.Context) (done, failed int) {
	limit := duckMemoryLimit()
	openDBsMu.Lock()
	var handles []*DB
	for _, c := range openDBs {
		if c.db != nil && !c.db.closed.Load() {
			handles = append(handles, c.db)
		}
	}
	openDBsMu.Unlock()
	for _, db := range handles {
		cctx, cancel := context.WithTimeout(ctx, applyLimitTimeout)
		_, err := db.loadSQL().ExecContext(cctx, "SET memory_limit='"+limit+"'")
		cancel()
		if err != nil {
			failed++
			slog.WarnContext(ctx, "duckdb: plafond mémoire non appliqué à une base ouverte", "path", db.path, "limit", limit, "err", err)
			continue
		}
		done++
	}
	return done, failed
}
