// Package duckdb — read_recovery.go : lectures BEST-EFFORT sur une DB partagée,
// résilientes à l'invalidation CONCURRENTE du handle (RecoveringReader).
//
// # Le trou que ce fichier ferme
//
// `OpenReadForQuery(path)` retourne un *INSTANTANÉ* : le `*sql.DB` que le
// wrapper `*DB` porte AU MOMENT de l'appel. Quand l'appel a emprunté le handle
// du cache process (`LookupCachedDB`), l'emprunt est NON POSSÉDANT — aucun
// incrément de refCount — et le release rendu est un no-op. Le propriétaire peut
// donc fermer le `*sql.DB` sous les pieds de l'emprunteur :
//
//   - B-swap RO→RW (ADR 0016) : `PlayerDB.PrepareForSharedSwap` ferme `pdb.Shared`
//     pour que le Provider puisse ouvrir le fichier en RW ;
//   - fin d'un autre `OpenReadForQuery` PROPRIÉTAIRE (celui qui a eu le cache
//     miss et détient donc refCount=1) : son release supprime l'entrée du cache
//     et ferme le `*sql.DB` — que d'autres lecteurs concurrents ont pourtant
//     emprunté entre-temps ;
//   - `EvictAndCloseCached` (purge délibérée d'une DB).
//
// L'emprunteur voit alors « sql: database is closed » PENDANT sa requête ou son
// itération de rows. Les deux familles observées en prod (2026-08-25) sont
// exactement ce cas :
//   - `ops.IndexMedia` → `loadMatchTimeWindows` sur shared_matches_v2 (~2,7 ERROR/j) ;
//   - `sync.buildCitationContext` → `loadPveStats` sur shared_pve (372 WARN/mois) :
//     le pipeline post-sync tourne en parallèle SANS limite par joueur
//     (`RunPostSync`, PostSyncParallelism = 0), chaque joueur acquiert son handle
//     shared_pve ; le premier à finir ferme le fichier pour tous les autres.
//
// # Pourquoi PAS « rendre l'emprunt possédant »
//
// Faire incrémenter le refCount par `OpenReadForQuery` supprimerait la classe
// entière… en cassant le B-swap : `pdb.Shared.Close()` deviendrait un simple
// décrément, le fichier resterait tenu en RO, et l'`OpenReadWrite` du Provider
// échouerait ("different configuration"). Le modèle assumé est « le swap gagne,
// le lecteur se répare » — d'où la reprise ci-dessous plutôt qu'une possession.
//
// # INVARIANT : jamais d'ouverture RO neuve sur un chemin géré par un provider
//
// `sharedprovider` documente (provider.go) être l'UNIQUE owner du handle DuckDB
// pour son chemin : « Jamais de sql.Open direct sur ce fichier ailleurs ».
// La raison est mécanique : `swapToRW` ferme le handle RO (provider_writer.go),
// notifie les subscribers, PUIS appelle `OpenReadWrite`. Une ré-ouverture RO
// émise dans cette fenêtre gagne parfois la course contre l'`OpenReadWrite` — le
// provider part alors en StateError (lectures shared en `ErrSwapFailed`/503,
// burst d'écriture abandonné jusqu'au `retryReopenLoop`). Un RecoveringReader
// n'est PAS visible du drain lecteurs (`readersWG`) sur lequel le swap s'appuie :
// il ne peut donc pas se contenter d'espérer.
//
// D'où `ReopenPolicy`, à choisir EXPLICITEMENT à la construction :
//
//   - `ReopenCacheOnly` — chemin géré par un provider (shared_matches_v2) : la
//     reprise n'emprunte QUE le cache process (`LookupCachedDB` ; l'entrée `rw:`
//     posée par le swap compte, une lecture marche sur un handle RW). Cache miss
//     → AUCUNE ouverture, on rend l'erreur d'origine : la lecture reste
//     best-effort exactement comme avant ce helper, et le swap est préservé.
//   - `ReopenAllowed` — chemin SANS provider (shared_pve : aucun `OpenReadWrite`
//     en régime établi, le serveur ne l'ouvre en RW qu'au boot pour les
//     migrations) : la reprise peut ouvrir un handle RO neuf.
//
// Nuance assumée : l'acquisition INITIALE passe par `OpenReadForQuery` dans les
// deux modes, donc peut encore ouvrir un RO neuf sur cache miss. C'est la parité
// stricte avec le code d'avant ce lot (les deux sites appelaient déjà
// `OpenReadForQuery`) — aucune régression introduite ici, mais la fenêtre
// théorique subsiste au PREMIER appel. La fermer demande le canal drainé du
// provider (`Provider.Get`, où le swap ATTEND le lecteur) : voie consignée au
// journal, elle exige de faire descendre `cfg.SharedProvider` jusqu'à `ops`.
//
// # Le contrat
//
// `RecoveringReader` garde le CHEMIN (pas seulement l'instantané) : si `fn`
// échoue sur une erreur d'invalidation (`IsInvalidatedError`), le handle est
// RE-RÉSOLU puis `fn` est rejouée UNE seule fois. La reprise est loguée en WARN
// (même registre que `WithReopenOnInvalidated`, db_recovery.go).
//
// Elle n'est PAS toujours possible, et le helper ne le prétend pas :
//
//   - classe « handle fermée » (« sql: database is closed ») : le `Close()` du
//     propriétaire a supprimé l'entrée de cache, la re-résolution rend un
//     `*sql.DB` VIVANT — c'est le cas nominal, celui des deux familles prod ;
//   - classe FATAL (« database has been invalidated », « Failed to delete all
//     rows from index ») : le `*sql.DB` n'est PAS fermé et le cache PAS purgé, la
//     re-résolution rend LE MÊME pointeur mort. Détecté par comparaison de
//     pointeur : aucun rejeu, erreur d'origine rendue, log dédié. Le geste
//     correctif est `(*DB).Reopen` côté PROPRIÉTAIRE du handle (db_recovery.go) —
//     hors de portée d'un emprunteur ;
//   - `ReopenCacheOnly` + cache miss : aucun rejeu (cf. invariant ci-dessus).
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"sync"
)

// ReopenPolicy décide ce qu'un RecoveringReader a le droit de faire pour
// re-résoudre son handle après une invalidation. Choix EXPLICITE à la
// construction — il n'y a pas de défaut sûr pour les deux familles de chemins
// (cf. section INVARIANT de l'en-tête).
type ReopenPolicy int

const (
	// ReopenAllowed autorise la reprise à ouvrir un handle RO neuf si rien n'est
	// en cache. Réservé aux chemins qu'AUCUN sharedprovider ne gère (shared_pve).
	ReopenAllowed ReopenPolicy = iota

	// ReopenCacheOnly interdit toute ouverture : la reprise n'emprunte que le
	// cache process. OBLIGATOIRE sur un chemin géré par un sharedprovider
	// (shared_matches_v2) — une ouverture RO dans la fenêtre de swap peut faire
	// échouer l'OpenReadWrite du provider et l'envoyer en StateError.
	ReopenCacheOnly
)

// RecoveringReader porte un handle de LECTURE sur une DB DuckDB partagée et le
// re-résout quand il est invalidé sous les pieds du lecteur (cf. en-tête).
//
// Usage : acquérir UNE fois (l'ouverture DuckDB est coûteuse — ne pas en faire
// une par ligne lue), puis passer chaque requête par Do. Le zéro-value n'est pas
// utilisable : passer par OpenRecoveringReader. Un `*RecoveringReader` nil est
// accepté par Close (no-op) pour que les callers en dégradation gracieuse
// (fichier absent) puissent différer Close sans test préalable.
//
// Sûr en usage concurrent : plusieurs goroutines peuvent partager un reader ;
// une seule re-résolution a lieu par invalidation (compteur de génération).
type RecoveringReader struct {
	path     string
	timezone []string
	policy   ReopenPolicy

	mu      sync.Mutex
	db      *sql.DB
	release func()
	gen     uint64
	closed  bool
}

// ErrReaderClosed est renvoyée par Do sur un reader nil ou déjà fermé.
var ErrReaderClosed = errors.New("duckdb: RecoveringReader fermé ou nil")

// errNoCachedHandle : re-résolution impossible en ReopenCacheOnly (rien en
// cache). Sur le chemin de REPRISE, Do rend l'erreur d'origine à sa place ;
// elle n'atteint le caller que si le reader était déjà vide AVANT l'appel (une
// re-résolution antérieure ayant échoué) — la lecture échoue alors proprement,
// ce qui reste préférable à l'ouverture RO que l'invariant interdit.
var errNoCachedHandle = errors.New("duckdb: aucun handle en cache")

// errSameDeadHandle : la re-résolution a rendu LE MÊME pointeur que celui qui
// vient d'échouer (classe FATAL : *sql.DB non fermé, cache non purgé) — rejouer
// serait un no-op trompeur. Interne : Do rend l'erreur d'origine à sa place.
var errSameDeadHandle = errors.New("duckdb: handle re-résolu identique à l'invalidé")

// OpenRecoveringReader acquiert un handle de lecture sur path (via
// OpenReadForQuery : réutilise le handle process RW/RO s'il existe, ouvre en RO
// sinon) et l'enveloppe dans un reader auto-réparant. Le caller DOIT différer
// Close (qui rend l'emprunt / ferme le handle possédé, exactement comme le
// release d'OpenReadForQuery).
//
// policy : ReopenCacheOnly pour un chemin géré par un sharedprovider,
// ReopenAllowed sinon. Cf. section INVARIANT de l'en-tête — se tromper ici est
// un risque opérationnel, pas un détail de style.
func OpenRecoveringReader(path string, policy ReopenPolicy, timezone ...string) (*RecoveringReader, error) {
	r := &RecoveringReader{path: path, policy: policy, timezone: timezone}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, _, err := r.openLocked(); err != nil {
		return nil, err
	}
	return r, nil
}

// Do exécute fn avec le handle de lecture courant. Si fn échoue sur une
// invalidation (IsInvalidatedError : « sql: database is closed », handle fermée
// par un B-swap ou par le release d'un emprunt concurrent), le handle est
// re-résolu et fn est rejouée UNE fois — jamais plus.
//
// Quand la reprise est impossible (cache miss en ReopenCacheOnly, classe FATAL),
// Do rend l'erreur D'ORIGINE, non enveloppée : le caller best-effort la traite
// exactement comme avant ce helper.
//
// Contrat de fn : elle doit être REJOUABLE — toute accumulation (slice, map)
// doit être réinitialisée en entrée de fn, sinon le retry duplique les lignes.
// fn ne doit PAS conserver le *sql.DB au-delà de son retour.
//
// Les erreurs non-invalidation (y compris sql.ErrNoRows) sont retournées telles
// quelles, sans retry ni enveloppe — `errors.Is` reste utilisable côté caller.
func (r *RecoveringReader) Do(ctx context.Context, fn func(db *sql.DB) error) error {
	if r == nil {
		return ErrReaderClosed
	}
	db, gen, err := r.current()
	if err != nil {
		return err
	}
	if err = fn(db); !IsInvalidatedError(err) {
		return err
	}

	fresh, refreshErr := r.refresh(gen, db)
	if refreshErr != nil {
		r.logUnrecoverable(ctx, err, refreshErr)
		return err
	}

	slog.WarnContext(ctx, "duckdb: handle de lecture invalidée pendant la requête — re-résolue, retry unique",
		"path", r.path, "err", err)
	retryErr := fn(fresh)
	// ERROR réservé à l'invalidation PERSISTANTE : une erreur métier au retry
	// (sql.ErrNoRows sur un match non-Firefight, cas majoritaire) est un
	// résultat, pas un incident — la remonter sans bruit, le WARN de reprise
	// ci-dessus suffit à tracer l'événement.
	if IsInvalidatedError(retryErr) {
		slog.ErrorContext(ctx, "duckdb: invalidation persistante après re-résolution du handle",
			"path", r.path, "err", retryErr)
	}
	return retryErr
}

// logUnrecoverable trace une invalidation dont la reprise est impossible, en
// distinguant les causes (elles n'appellent pas les mêmes gestes d'exploitation).
func (r *RecoveringReader) logUnrecoverable(ctx context.Context, origErr, refreshErr error) {
	switch {
	case errors.Is(refreshErr, errSameDeadHandle):
		slog.ErrorContext(ctx, "duckdb: invalidation FATALE d'un handle partagé — hors de portée du lecteur",
			"path", r.path, "err", origErr,
			"hint", "le *sql.DB n'est pas fermé et le cache pas purgé : geste correctif = (*DB).Reopen côté propriétaire du handle")
	case errors.Is(refreshErr, errNoCachedHandle):
		slog.WarnContext(ctx, "duckdb: handle invalidée et aucun handle en cache — lecture best-effort abandonnée",
			"path", r.path, "err", origErr,
			"hint", "chemin géré par un sharedprovider : ouvrir un RO neuf ici casserait le B-swap")
	default:
		slog.WarnContext(ctx, "duckdb: re-résolution du handle de lecture impossible — lecture best-effort abandonnée",
			"path", r.path, "err", origErr, "reopen_err", refreshErr)
	}
}

// Close rend le handle (no-op sur un emprunt du cache, fermeture sur un handle
// possédé). Idempotent, et sûr sur un reader nil.
func (r *RecoveringReader) Close() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.releaseLocked()
	r.closed = true
}

// current retourne le handle courant et sa génération. Si une re-résolution
// précédente a échoué et laissé le reader vide, il est re-résolu — SOUS LA
// POLICY (resolveLocked, pas openLocked) : sur un chemin géré par un provider,
// un Do ultérieur ne doit pas plus ouvrir un RO neuf que la reprise elle-même.
func (r *RecoveringReader) current() (*sql.DB, uint64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, 0, ErrReaderClosed
	}
	if r.db != nil {
		return r.db, r.gen, nil
	}
	return r.resolveLocked()
}

// refresh rend le handle mort et en re-résout un. seenGen est la génération
// observée par le Do appelant : si elle a déjà changé, une autre goroutine a
// re-résolu entre-temps — on réutilise son handle au lieu d'en acquérir un 2e.
// deadDB est le handle qui vient d'échouer : si la re-résolution rend le MÊME
// pointeur, c'est la classe FATAL et il n'y a rien à rejouer.
func (r *RecoveringReader) refresh(seenGen uint64, deadDB *sql.DB) (*sql.DB, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, ErrReaderClosed
	}
	if r.gen != seenGen && r.db != nil {
		if r.db == deadDB {
			return nil, errSameDeadHandle
		}
		return r.db, nil
	}
	// Rendre AVANT de re-résoudre : si ce reader était le PROPRIÉTAIRE de
	// l'entrée de cache, son release la supprime — sans quoi on re-emprunterait
	// le même instantané mort.
	r.releaseLocked()

	db, _, err := r.resolveLocked()
	if err != nil {
		return nil, err
	}
	if db == deadDB {
		return nil, errSameDeadHandle
	}
	return db, nil
}

// resolveLocked acquiert un handle selon la ReopenPolicy. À appeler sous r.mu.
func (r *RecoveringReader) resolveLocked() (*sql.DB, uint64, error) {
	if r.policy == ReopenCacheOnly {
		return r.borrowCachedLocked()
	}
	return r.openLocked()
}

// borrowCachedLocked emprunte le handle du cache process SANS jamais ouvrir —
// seule re-résolution autorisée sur un chemin géré par un sharedprovider.
// LookupCachedDB sert l'entrée `rw:` en priorité : pendant une fenêtre RW le
// lecteur récupère donc le handle du writer (une lecture marche sur un RW).
// À appeler sous r.mu.
func (r *RecoveringReader) borrowCachedLocked() (*sql.DB, uint64, error) {
	cached, ok := LookupCachedDB(r.path)
	if !ok {
		return nil, r.gen, errNoCachedHandle
	}
	// Emprunt non possédant (contrat de LookupCachedDB) : release no-op.
	r.db, r.release, r.gen = cached.SQLDb(), nil, r.gen+1
	return r.db, r.gen, nil
}

// openLocked acquiert un handle, en ouvrant en RO si rien n'est en cache.
// À appeler sous r.mu.
func (r *RecoveringReader) openLocked() (*sql.DB, uint64, error) {
	db, release, err := OpenReadForQuery(r.path, r.timezone...)
	if err != nil {
		return nil, r.gen, err
	}
	r.db, r.release, r.gen = db, release, r.gen+1
	return db, r.gen, nil
}

// releaseLocked rend le handle courant s'il y en a un. À appeler sous r.mu.
func (r *RecoveringReader) releaseLocked() {
	if r.release != nil {
		r.release()
	}
	r.db, r.release = nil, nil
}
