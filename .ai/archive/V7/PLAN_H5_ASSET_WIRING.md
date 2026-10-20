# PLAN — Câblage assets title-agnostic (audit 2026-06-20)

## STATUT (2026-06-21) — 7 incréments livrés + poussés sur `feat/multititre-peripherie`

| Réf | Gap/Violation | Commit | État |
|-----|---------------|--------|------|
| S.1 | V1/V5 slug figé `homeStaticTitleSlug` → `r.titleSlug()` | 7274d92c9 | OK |
| C0 | G1 adapter `halo_5.AssetURLAdapter` + `RegisterAssetURL` (+ openapi /medals) | 89799ce9e | OK |
| G3 | V8 bornes Héros par titre (XPMax/RankMax via CareerSnapshot) | 102af277e | OK |
| G5 | V3 badge CSR canonical tuiles title-aware (résolveur injecté) | f02925ed1 | OK |
| G2/G7/D.1 | contrat sprite médaille BACKEND (`static.MedalImage` + DTO + 4 sites) | 915b65f64 | OK |
| F.1 | rendu sprite médaille FRONT (`MedalIcon` partagé, 4 surfaces) | b2ed57f36 | OK |
| V9/D.2 | images de rang carrière PAR TITRE (h5 = map vide, plus d'image HINF erronée) | daaef9bcc | OK |

**Effet** : maps (tuiles home + vue match), badges CSR (tuiles + vue match), médailles
(end-to-end : référentiel + vue match + tuiles + explorer + digest escouade), carte Héros
carrière, images de rang carrière — tous title-aware. HINF byte-identique partout
(nil/résolveur absent → comportement HINF inchangé).

### RESTANT (chacun avec une dépendance / décision / périmètre à clarifier)
- **G4/S.6 — CSR carrière h5** : `GetCareerCSRs` lit `player_csr_snapshots` (vide pour h5) →
  colonne « Non classé ». Le CSR h5 vit dans `CareerSnapshot` (RankTier/RankName, CSR UNIQUE).
  Bloquant : **décision produit** (h5 = 1 CSR courant vs structure per-playlist HINF) + méthode
  adapter CSR per-playlist (inexistante). Non-trivial, pas un simple câblage.
- **G9/S.8 — payload Match View h5** : `match.detail.core = not_exposed` → page vue-match VIDE
  pour h5. C'est une **FEATURE** (exposer `LoadMatchDetail` via carnage → participants/teams/skill),
  pas un câblage d'asset. Plus gros chantier restant ; **pré-requis de V11** (lien Waypoint).
- **G8/V10/F.2 — groupes LUSR front HINF-only** : front itère `LUSR_KNOWN_GROUPS` (HINF) dès
  capability `lusr`. h5 déclare `lusr` (backfill `cmd/h5-lusr-backfill` EN COURS) → 4 groupes HINF
  « Non classé ». Fix = dériver les groupes de la donnée résolue SANS régresser HINF (qui affiche
  ses 4 groupes même non classés). **Différé** pending clarté sur l'état LUSR h5.
- **V4 — `halo_infinite.NewAssetURLAdapter()` figé** (`registry_pages.go:415,425,442`,
  `home_repo_skill_peak.go:460`) : hygiène sur le chemin CSR **sync HINF-only** (sync_pkg = données
  HINF) ; AUCUN impact fonctionnel h5. Re-typer sur l'interface `games.TitleAssetURLAdapter`. Basse prio.
- **V11 — WaypointURL `halo-infinite/` figée** (`match_view_builders_header.go:59`) : lien externe.
  **Dépend de G9** (vue-match vide pour h5 tant que le payload n'est pas exposé). Fix propre =
  `adapter.ExternalMatchURL(matchID)` (interface, HINF → URL ; h5 → "" pas de page Waypoint h5).


> Source : workflow d'audit 7 agents (run `w5jqyzrsk`). Cartographie tous les sites
> front+back affichant médailles/armes/maps/CSR/rang XP. Objectif : un seul chemin
> générique `slug → static.URL(kind, slug, …) → titleResolver.AssetURL(slug) →
> csrBadgeResolver(slug,…)`. JAMAIS de slug littéral dans le data-path, JAMAIS
> d'`AssetURLAdapter` concret instancié en dur, JAMAIS de méthode par jeu.
> Modèle correct à répliquer : `home_repo_identity.go:183
> buildHomeIdentityAssetURL(imageType, r.titleSlug(), value)`.

## GAPS h5 (assets qui NE s'affichent PAS)

| # | Prio | Surface | Cause | Fix |
|---|------|---------|-------|-----|
| G1 | P0 | Match View map + badge CSR | aucun `RegisterAssetURL("halo_5")` → `assetURL==nil` | adapter `halo_5/adapter_asset_urls.go` + `RegisterAssetURL` |
| G2 | P0 | Médailles (toutes surfaces hors Asset Drawer) | icône h5 = SPRITE, contrat `image_url` ne le porte pas → 4 PNG en dur 404 | contrat sprite dans DTO médaille + chokepoint `assetURL.MedalImageURL` |
| G3 | P0 | Carte Héros Carrière | consts package `xpHeroTotal=9.3M`, `rankMax=272` appliqués à h5 (SR152/XP50M) | XPMax/RankMax par titre (CareerSnapshot) |
| G4 | P1 | Badge CSR Carrière | `GetCareerCSRs` lit table joueur vide + badge forgé `homeStaticTitleSlug` | router via dataAdapter + `r.titleSlug()` |
| G5 | P1 | Badge CSR tuiles + Match View | `buildCanonicalSkillBadge` forge `/static/ranks/halo_infinite/…` sans param titre | passer `titleSlug` + `csrBadgeResolver` |
| G6 | P1 | Image map tuiles Home | `loadHomeMapImageURLs` bind `title_id = homeStaticTitleSlug` → 0 ligne | binder `r.titleSlug()` |
| G7 | P1 | Médailles tuiles Home | `homeMedalIconURL` slug HINF figé | param titre + solution sprite G2 |
| G8 | P1 | Colonne LUSR Carrière | front itère `LUSR_KNOWN_GROUPS` HINF-only dès capability `lusr` | dériver groupes du titre / gater sur data résolue |
| G9 | P0 | Match View payload h5 | `match.detail.core = not_exposed`, `WithDataAdapter` inutilisé | exposer `LoadMatchDetail` via carnage (chantier séparé) |

## VIOLATIONS title-agnostic (V1–V11)

- **V1** `const homeStaticTitleSlug = "halo_infinite"` (`home_repo.go:26`) propagé dans
  `home_repo_medals_citations.go:21`, `home_repo_translations.go:127`,
  `career_repo_csr.go:146-153,175,209`, `career_repo_lusr.go:147`,
  `home_repo_skill_peak.go:254-407`, `home_repo_playlist_ranks.go:158-375`,
  `home_repo_matches.go:69`. → remplacer par `r.titleSlug()`, supprimer la const.
- **V2** Médailles : 4 compositions PNG en dur (`match_view_builders_summary.go:224`,
  `home_repo_medals_citations.go:20-22`, `explorer_target_medals.go:49`,
  `teammates_squad_charts_medal_digest.go:231`) → chokepoint `assetURL.MedalImageURL` + variante sprite.
- **V3** Badge CSR canonical (`analysis/home_canonical_skill.go:48-52,103-138`) URL HINF figée,
  signature sans `titleSlug` → ajouter `titleSlug` + `csrBadgeResolver`.
- **V4** `halo_infinite.NewAssetURLAdapter()` en dur (`registry_pages.go:415,425,442`,
  `home_repo_skill_peak.go:460`) → `titleResolver.AssetURL(slug)`, re-typer sur l'interface.
- **V5** Map registry filtré sur slug HINF (`home_repo_translations.go:127`) → `r.titleSlug()`.
- **V6** `csrBadgeResolver` = hook mono-titre (`home_repo_skill_peak.go:418` + `SetCSRBadgeResolver`,
  wiring `server.go:699` halo5-only) → map `slug → resolver` (registry par titre).
- **V7** Asset Drawer fallback `WithMapImageURL`/`WithWeaponImageURL` ignore `titleID` (`server.go:675-680`)
  → résoudre l'adapter du titre via le `titleID`.
- **V8** Bornes Héros consts package (`career_service.go:38-43`) → par titre (CareerSnapshot).
- **V9** `rankImageURLs` chargé au boot avec `DefaultSlug` injecté dans tous les titres
  (`server.go:369`, `registry_career.go:54-56`) → charger par titre via
  `LoadCareerRankImageURLs(ctx, metaDB, titleSlug)` (déjà paramétré).
- **V10** `LUSR_KNOWN_GROUPS` HINF en dur front (`lusr-chains.ts:5`) → dériver du titre.
- **V11** WaypointURL HINF figée (`match_view_builders_header.go:59`) → `adapter.ExternalMatchURL`.

## PLAN PAR COUCHE (ordre de déploiement)

COUCHE 0 (débloque tout) → DATA → ADAPTER → SERVICE/REPO → HANDLER → FRONT.
Déployer en une fois (ne pas livrer un fix partiel laissant une surface HINF-figée).
La campagne append-only en cours (`fix/metadata-art-battlepass-appendonly`) se déploie à la fin.

- **C0.1** `games/halo_5/adapter_asset_urls.go` impl `games.TitleAssetURLAdapter` (Map/Medal/CSR/CSROnyx/Weapon),
  route `/assets/{title_id}/…` ou static h5 + variante sprite médailles. Réf : `games/halo_infinite/adapter_asset_urls.go`.
- **C0.2** `resolver.RegisterAssetURL(h5AssetURL)` dans `registerHalo5Adapters` (`server_titles_additional.go`).
- **C0.3** Promouvoir `csrBadgeResolver` (hook mono) → map `slug → resolver` peuplée au boot par titre.
- **D.1** Étendre le contrat image médaille (sprite sheet+offset) dans les DTO médaille.
- **D.2** `rankImageURLs` PAR titre (`LoadCareerRankImageURLs(ctx, metaDB, titleSlug)`).
- **A.4** Exposer XPMax/RankMax par titre (`canonical.CareerSnapshot`).
- **S.1** `homeStaticTitleSlug` → `r.titleSlug()` partout, supprimer la const.
- **S.2** Médailles : 4 compositions → `assetURL.MedalImageURL(...)` + sprite.
- **S.3** `buildCanonicalSkillBadge(+titleSlug)` → `csrBadgeResolver`.
- **S.4** Supprimer `NewAssetURLAdapter()` au profit de `titleResolver.AssetURL(slug)`.
- **S.5** Injecter le bon `TitleAssetURLAdapter` selon `pdb.TitleSlug` dans `WithAssetURL`.
- **S.6** `GetCareerCSRs` via dataAdapter (capability) pour h5.
- **S.7** Bornes Héros par titre (A.4) dans `buildHeroProgress`.
- **S.8** Match View payload h5 (`LoadMatchDetail` via carnage) — chantier séparé G9.
- **F.1** Types front médaille + rendu sprite (`background-position`) sinon `<img>` PNG.
- **F.2** LUSR : dériver les groupes du titre, gater sur data résolue.

## GARDE-FOUS (anti-régression)
- Test : aucun slug littéral `"halo_infinite"` dans le data-path asset (`home_repo*`, `career_repo*`, `analysis/home_canonical_*`).
- Test : aucun `NewAssetURLAdapter()` hors `RegisterAssetURL` au boot.
- Test : pour chaque titre `RegisterData`/`RegisterSemantic`, un `RegisterAssetURL` correspondant existe.

## DÉJÀ OK pour h5 (ne pas casser)
Asset Drawer (maps/armes/médailles sprite) · bannière identité Home (emblem/banner/rang SR+XP via
`buildHomeIdentityAssetURL(…, r.titleSlug(), …)` = modèle à suivre) · rang/XP brut Carrière (SR) ·
kill-feed Match View (arme-par-kill native) · labels armes breakdown (`weapon_labels` title-scopé).
