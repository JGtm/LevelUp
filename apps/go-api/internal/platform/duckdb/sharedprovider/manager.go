package sharedprovider

import "sync"

// Manager déduplique les Provider par chemin absolu.
//
// Cas d'usage cible (multi-titre, commit 5+) : un seul Provider par fichier
// shared_matches_v2.duckdb, peu importe combien de fois mgr.For(path) est
// appelé. En mono-titre actuel, c'est un singleton de facto.
//
// Manager est sûr pour usage concurrent.
type Manager struct {
	// providers map[string]Provider — clé = chemin absolu.
	providers sync.Map
}

// NewManager construit un Manager vide. Les Provider sont créés à la
// demande via For().
func NewManager() *Manager {
	return &Manager{}
}

// For retourne le Provider associé à path, en le créant si nécessaire.
// Eager open : si le fichier DuckDB est inaccessible, l'erreur remonte
// immédiatement (utile pour détecter les problèmes au boot du serveur).
//
// timezone (optionnel) : utilisé seulement à la première création — les
// appels ultérieurs sur le même path ignorent cet argument et retournent
// le Provider existant.
//
// Sûr pour usage concurrent : si deux goroutines appellent For() avec le
// même path simultanément, une seule conn DuckDB est ouverte.
func (m *Manager) For(path string, timezone ...string) (Provider, error) {
	if v, ok := m.providers.Load(path); ok {
		return v.(Provider), nil
	}

	tz := ""
	if len(timezone) > 0 {
		tz = timezone[0]
	}

	p, err := New(path, tz)
	if err != nil {
		return nil, err
	}

	// LoadOrStore gère la course : si une autre goroutine a créé un
	// Provider entre notre Load initial et ici, on ferme le notre et on
	// utilise celui qui a gagné la course.
	if actual, loaded := m.providers.LoadOrStore(path, p); loaded {
		_ = p.Close()
		return actual.(Provider), nil
	}
	return p, nil
}

// Close ferme tous les Providers gérés. Idempotent.
//
// Retourne la première erreur rencontrée ; les autres Close sont tentés
// quand même (ne pas court-circuiter sur le premier échec — on veut libérer
// le maximum de ressources au shutdown).
func (m *Manager) Close() error {
	var firstErr error
	m.providers.Range(func(_, v any) bool {
		if err := v.(Provider).Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		return true
	})
	return firstErr
}
