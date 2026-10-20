# REVUE QUALITE — travail multi-titre Halo 5 (2026-06-22)

> Revue 5 dimensions (couches, factorisation, extensibilité, structure, front) + synthèse.
> Verdict global : **8.5/10 — niveau professionnel, globalement prod-ready.** Archi + extensibilité ~9.5 ;
> DRY des primitives ~6 (tiré par la duplication ISO8601). Branche `feat/multititre-peripherie`.

## Forces (à préserver)
- Wiring **registry-driven, zéro slug littéral** (archlint `TestNoNewSlugComparison` actif). Ajouter Halo 7 = ~5 lignes de registrar.
- Adapters 3-couches SRP stricts (Data/Semantic/AssetURL) ; `AssetURLAdapter` pur.
- Dégradation gracieuse honnête (`ErrCapabilityNotSupported`, 404→vide+warn, token absent→court-circuit). `fallbackCapabilities()` testé vs `capabilities.toml`.
- Réutilisation byte-identique du pipeline d'ingestion Infinite (`persist.MatchBatch → SharedPersister`).
- Livesync testable sans DB/réseau (Deps injectables). Discipline taille/nommage (zéro fichier >500L, func >80L).

## Dette priorisée
| Sévérité | Zone | Problème | Reco |
|---|---|---|---|
| **BLOCKER** | `halo_5/mapping.go` + `platform/halo/compare_provider.go` + `openspartan/mapper/iso8601.go` | **Triple** parseur ISO8601 avec **regex divergente** (`T` obligatoire h5 / optionnel compare) + types retour divergents → `P1D` parse d'un côté, échoue de l'autre. **Bug latent**, pas que DRY. | `internal/platform/halo/duration_utils.go` : 1 regex canonique + wrappers `...Seconds(int64)`/`...Ptr(*int)`/`...Float(*float64)` + tests (cas `P1D`). Migrer les 3 sites. |
| **Major** | `halo_5/mapping_carnage.go` + `mapping_carnage_detail.go` | 2 mappers parallèles du même DTO carnage (~75% dupliqué : xuid, KDA, durée, outcome). | Extraire `extractH5CarnageParticipantData(...) []ParticipantData` (pur) + 2 façades fines (domain/canonical). |
| **Major** | `domain/title/registry.go` + `world_player_season_stats` | `CapWorldLeaderboard` HINF-only + `DEFAULT 'halo_infinite'` figé. | Déclarer la capability dans `capabilities.toml` h5 (`not_exposed`) ; titre courant pas littéral. |
| **Major** | `config/titles/halo_5/catalog/ranked_hoppers.toml` | Catalogue ranked déclaré mais **non câblé** côté backend h5 (noms de playlists CSR — c'est le TODO laissé par G4 Phase 1). | Brancher `LoadRankedPlaylists` sur `ranked_hoppers.toml` + `playlists_catalog` clé `title_slug`. |
| Minor | `sharedprovider`/`livesync/wire.go` | Provider per-titre : `Manager==nil` → fallback RW silencieux. Non scalable à 3+ titres RW. | Gating boot fail-hard (titre actif + Manager nil). |
| Minor | `livesync/wire.go halo5ResolverFactory` | Seed PeopleHub sans dédup XUID (2 joueurs même xuid → injection silencieuse). | Validation boot fail-hard ou dédup clé=xuid. |
| Minor | `web/career/lusr-chains.ts` | `LUSR_KNOWN_GROUPS_BY_TITLE` sans clé `halo_5` → pas de placeholder « Non classé » (divergence UX vs HINF). | `halo_5: ['h5_arena']` si parité placeholder voulue (décision produit). |
| Minor | `web/capabilities/FeatureUnavailable.tsx` | Labels indispo inline FR/EN. | Externaliser i18n. |
| Minor | `halo_5/capture.go` + `ingest/medals.go` | `EventsFailed`/`CarnageFailed` best-effort silencieux (match sans participants jamais signalé). | `data_gaps` sur MatchBatch → badge `CapabilityGap` front. |
| Nit | `adapter_asset_urls.go MedalImageURL` | stub "" (sprite = chantier G2). | Doc + brancher sprite plus tard (AssetMeta a déjà les champs). |
| Nit | `halo_5/doc.go` | Docstring obsolète (`coming_soon` alors que `active`). | MAJ état réel. |
| Nit | outcome TIE non géré (nil→loss) ; `web client.ts` X-LevelUp-Title contourné par fetch directs ; `MedalDigest.tsx` hex `#fff`. | | Documenter/externaliser. |

## Plan ordonné
- **Phase 1 quick wins (~1j)** : (1) `duration_utils.go` partagé [BLOCKER+bug] ; (2) LUSR `halo_5:['h5_arena']` ; (3) doc.go + commentaires ; (4) fail-hard provider nil + dédup XUID seed.
- **Phase 2 refactors (~2-3j)** : (5) mapper carnage intermédiaire ; (6) `data_gaps` → front ; (7) labels FeatureUnavailable i18n.
- **Phase 3 finitions** : (8) `ranked_hoppers.toml` h5 + `CapWorldLeaderboard` title-aware ; (9) outcome TIE ; (10) registry de boot formel (bénéfice à 3+ titres).

## Halo 7-readiness : PRÊT (~3-4j adapter + parity tests/titre)
Archétype démontré (`synthetic_title_b` + skeleton tests). Checklist : title.toml + auth.toml + capabilities.toml ; adapters games/halo_7/* + registrar ; migrations (metadata propre) ; runnerBuilders si live-only ; skeleton + parity + vitest paramétré.
Points durs avant multi-titre RW concurrent : provider `Manager≠nil` exigé (gating), seed XUID dédup, **extraire duration_utils AVANT** (sinon Halo 7 quadruple la duplication), adapter Phase 2 = vrai refactor mappers (pas stub→impl trivial).
