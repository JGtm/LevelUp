// Package halo_5 implémente l'adapter Halo 5: Guardians, 2e titre du registre
// multi-titre (cf. .ai/HANDOFF_HALO5_EXPERIMENTAL.md). status=active.
//
// État (Phase 1a livrée) : le titre est SERVI. Source de données = API cryptum
// LIVE (SpartanToken résolu par requête via la SourceFactory), identité
// GAMERTAG-keyée (Player.Xuid null). Adapters câblés au boot via
// api/server_titles_additional.go (registry-driven, jamais slug littéral) :
//   - games.TitleDataAdapter (adapter_data.go) : LoadPlayerStats, LoadCareerSnapshot
//     (Spartan Rank + CSR pic), LoadMatchDetail (carnage -> canonical), LoadMatchEvents
//     (kill-feed natif). Le reste = ErrCapabilityNotSupported (capabilities.toml honnête).
//   - games.TitleSemanticAdapter (générique sur les TOML config/titles/halo_5/).
//   - games.TitleAssetURLAdapter (adapter_asset_urls.go, pur : URLs CDN officielles).
//
// Stockage : livesync/ persiste les matchs (registry/participants/médailles/kills)
// dans le shared DB du titre (persist.MatchBatch -> SharedPersister, comme Infinite)
// + CSR par playlist (player DB) + LUSR (chaîne unique h5_arena). Métadonnées
// (médailles/maps/armes/CSR) seedées par cmd/h5-metadata-fetch (API officielle).
//
// Phase 2 (différée) : LoadMatchSummaries (history = GetPlayerMatches player+page,
// pas ID-based) ; CSR par saison réelle (seasonId) ; Champion (#N) ; outcome TIE.
package halo_5
