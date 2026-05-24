// Package testfixtures fournit des helpers d'accès aux fixtures de test
// partagées entre packages (sync, persist, analysis).
//
// Les fixtures sont stockées sous apps/go-api/internal/sync/testdata/ (chunks
// binaires + JSON API responses + WAL batches). Ce package résout le chemin
// d'accès depuis n'importe quel répertoire de test via runtime.Caller(0).
//
// Fixtures principales :
//
//   - jgtm_full_match/ : 1 match Halo Infinite complet (manifest + 30 chunks
//
//   - api_match_stats.json + api_skill.json + api_match_history_page0.json).
//     Régénéré via `go run ./cmd/gen_test_fixtures download-full-match`.
//
//   - data/wal/ : 69 batches WAL JSON capturés en production (référence:
//     racine repo data/wal/, accessible via WALDir()).
//
// Le package est consommable par tous les autres packages internes : il
// n'importe RIEN du domaine métier (pas de cycle d'import possible).
package testfixtures
