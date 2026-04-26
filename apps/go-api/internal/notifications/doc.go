// Package notifications expose une API interne de notifications in-app
// per-player, vue comme un domaine autonome découplé du reste de l'app.
//
// Règle de dépendance :
//   - Ce package n'importe que la stdlib.
//   - L'implémentation du Repository vit dans internal/platform/duckdb/.
//   - Les modules consommateurs (sync, handlers media, etc.) reçoivent une
//     Emitter via DI ; ils ne connaissent pas l'impl, juste l'interface.
//
// Surface publique :
//   - Service       : opérations CRUD sur les notifications + préférences.
//   - Emitter       : interface réduite pour émettre une notif (côté hooks).
//   - NoopEmitter   : impl no-op pour tests / désactivation par config.
//   - Notification  : modèle exposé par l'API.
//   - Repository    : port consommé par Service.
//
// Stockage : 2 tables DuckDB per-player (stats.duckdb) :
//   - player_notifications     : flux de notifs (id snowflake-like)
//   - notification_preferences : préférences par catégorie
//
// Voir migrations/steps_player_notifications.go pour le schéma SQL.
package notifications
