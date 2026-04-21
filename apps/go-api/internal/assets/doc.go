// Package assets fournit une couche d'abstraction unifiée pour la résolution
// de tous les assets visuels et données Halo Infinite (médailles, maps, défis, battle pass).
//
// Le pattern local-first → API-fallback → persist est implémenté une seule fois
// dans ce package. Les handlers HTTP, le HaloProvider et les CLIs passent tous
// par [Resolver] sans toucher directement le filesystem, DuckDB ou GameCMS.
//
// Architecture :
//
//	BinaryStore  — fichiers sur FS (toujours disponible, atomique)
//	IndexStore   — index DuckDB (best-effort, peut être indisponible)
//	Fetcher      — sources distantes (GameCMS, DiscoveryUGC)
//	WriteQueue   — sérialise toutes les écritures DuckDB (1 goroutine writer)
//	DefaultResolver — orchestre les 4 composants ci-dessus
//
// Contrat URL unifié :
//
//	GET /api/v1/assets/{kind}/{title}/{id}[?variant=&lang=]
//	→ 302 Found (redirect vers FS local ou CDN)
//	→ 200 OK + Content-Type + bytes (si serveur direct)
//	→ 404 Not Found (référence invalide)
//	→ 502 Bad Gateway (cache miss + upstream injoignable)
package assets
