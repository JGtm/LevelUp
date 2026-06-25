# HANDOFF — Halo 5 / multi-titre (maintenu, source unique de suivi)

> Mis à jour à chaque jalon. Branche `feat/multititre-peripherie` (worktree
> `levelup-multititre`), pas de PR. Dernière MAJ : 2026-06-21.

## DIRECTION (validée user — ne plus diverger)
- **Halo 5 STOCKE ses données** (substrat DuckDB partagé du titre), comme Infinite —
  PAS « live-only ». Le sync h5 = `internal/games/halo_5/livesync` (runner :
  fetch matchs+events+carnage → persist via `SharedPersister`), branché dans
  `cmd/server` (`livesync.HandlesTitle` / `AcquireRunner`). Idempotent (match_registry = ancre).
- **Lecture = lire le STOCKÉ.** Le live = fallback/enrichissement, jamais la source primaire.
- **Title-agnostic partout** : capability/adapter, jamais `slug == "..."`, pas de méthode par jeu.

## LIVRÉ + poussé (à re-vérifier, cf. § Vérifs)
| Réf | Commit | Quoi | Re-vérif |
|-----|--------|------|----------|
| S.1 | 7274d92c9 | slug figé `homeStaticTitleSlug` éliminé | ok |
| C0 | 89799ce9e | adapter asset-url h5 + RegisterAssetURL | ok |
| G3 | 102af277e | bornes Héros par titre | ok |
| G5 | f02925ed1 | badge CSR canonical tuiles (résolveur injecté) | ok |
| G2/F.1 | 915b65f64, b2ed57f36 | sprite médaille back+front | ok |
| V9 | daaef9bcc | images de rang carrière par titre | ok |
| G9 | 57ced69b7, 10ebc92c1 | Match View h5 (carnage→canonical) | **À VÉRIFIER : lit live, doit privilégier le stocké** |

## CARTE LIVE vs STOCKÉ (h5) — état réel
- **STOCKÉ par le sync h5** : matchs (registry, participants, médailles, kills) → shared DB ;
  LUSR (`cmd/h5-lusr-backfill` → `match_skill_rank`).
- **PAS encore stocké** : **CSR par playlist** (snapshots) → c'est **G4**.
- **Lu en LIVE aujourd'hui** (à aligner sur « lire le stocké ») :
  - `LoadCareerSnapshot` (rang SR + CSR pic) — tape l'API à chaque affichage.
  - `LoadMatchDetail` (G9) — lit la carnage live ; routage **repo-first** (si le match est
    stocké, le repo prime ; live = fallback). **À VÉRIFIER que repo-first lit bien le stocké.**

## AUDIT sujets « live » HINF — RÉSULTAT (fait, agent inventaire)
**Le socle de RÉCUPÉRATION live est déjà title-agnostic** : Battlepass, Défis, Rang XP live,
Customization passent tous par `hostFor(EndpointKey)` + `gamePrefix(ctx)` (TOML par titre) +
providers câblés `WithTitleSlug`. Routage front par header `X-LevelUp-Title`. Aucun blocage structurel du fetch.
Les points hardcodés HINF sont **en aval** (catalogues / rendu / schémas JSON), pas dans le fetch :

| Sujet | Live ? | Title-agnostic | Gap |
|-------|--------|----------------|-----|
| Battlepass (rewardtracks) | oui (live+cache 24h) | oui (fetch) | decoder JSON `OperationRewardTracks` HINF-spécifique mais isolé |
| Défis (decks) | oui (live+cache 24h) | oui (fetch) | decoder `AssignedDecks` HINF isolé |
| **Rang XP live** (careerRank1) | oui (cache 5min) | oui (fetch) / **partiel (rendu)** | **catalog de rang chargé en dur HINF au boot** (`server.go:373` `LoadRankCatalog` force DefaultSlug) → titres additionnels sans noms/images de rang (h5 = SR en chiffre). V9 a fait les images par titre ; reste les **noms** (catalog par titre). |
| Customization (emblem/banner/nameplate) | oui (cache 6h) | oui (fetch) / partiel | `spartan_nameplate_resolver.go` chemins `hi` figés + parsing `Appearance.*` HINF |
| World leaderboard | scrape cron 24h | **non** (le plus hardcodé) | scraper page Waypoint + `halo_infinite/rankedplaylists` ; UUID/saisons HINF front. Gaté par capability (vide ailleurs) mais aucune source par titre |
| Playlist ranks (home) | DB au display, live au sync | oui (rendu) / partiel (sync) | source playlists = package HINF |

**Priorité multi-titre suggérée** : (1) **rank catalog par titre** (débloque le rang XP « complet », déjà à moitié via `rankImageURLsByTitle` de V9) ; (2) decoders BP/Défis par titre SI h5 expose ces endpoints ; (3) world leaderboard (coûteux, pas de source par titre → laisser gaté/vide).
Gaps front mineurs : libellés FR en dur (`HomeChallengesList.tsx`), playlists/saisons en dur (`LeaderboardBlock.tsx`).

## PLAN RESTANT (repriorisé après le finding fondation)
0. **FONDATION — provider shared PAR TITRE** (NOUVEAU #1). Sans lui, h5 ne lit AUCUNE donnée
   shared stockée en prod (cf. § BLOQUANT). Câbler `Manager.For(SharedDBPath(titleSlug))` par
   `pdb.TitleSlug`. Débloque Match View h5 (lit le stocké, plus de live permanent) + tuiles home +
   tout read shared h5. Prudence B-swap. **Pré-requis de la vraie valeur « h5 lit le stocké ».**
1. **G4 — persistance CSR par playlist au sync h5**. Mapper `ArenaPlaylistStats[]` → snapshots
   CSR → `saveCSRSnapshots` (player DB h5, append-only). Lecture via repo. + catalogue playlists
   ranked (`isRanked`) + tier **Champion (#N)**. Note : player DB ≠ shared → moins touché par la
   fondation, MAIS confirmer que h5 a une player DB. Réf : mémoire `reference_h5_csr_model`.
2. **G8 — groupes LUSR front data-driven** + aligner capability `lusr` h5.
3. **Catalog de rang carrière PAR TITRE** (gap rang XP de l'audit live) : `LoadRankCatalog` est
   forcé HINF au boot (`server.go:373`) → titres additionnels sans noms de rang. V9 a fait les
   images ; reste les noms. (h5 = SR en chiffre, donc impact h5 faible, mais c'est le pattern.)
4. **V4** — dette documentée (pas un bug). Rien sauf demande.
5. **Prod** — copier `data/titles/halo_5/warehouse/metadata.duckdb` (5 Mo) sur le VPS (op serveur, prévenir avant).

Audit live-subjects HINF : FAIT (cf. § AUDIT). Vérif livrables : FAITE (10 commits propres).

## G4 — conception (à confirmer par la vérif storage)
Pattern Infinite à reproduire (`sync/career.go`) :
`syncPlayerCSRs(ctx, client, db, xuid, seasonID)` → fetch CSR (player-level
`GetPlayerCSRs` + complétion par playlist active via `rankedplaylists.Active()`)
→ `saveCSRSnapshots(ctx, db, csrs, seasonID)` écrit dans la **player DB** (stats.duckdb).
Lecture : `CareerRepo.GetCSRSnapshots` lit `player_csr_snapshots` (catalogue-first :
synthétise « Non classé » pour les playlists actives jamais jouées).

Source h5 (déjà dispo) : `H5ArenaStats.ArenaPlaylistStats[]` (dto.go) → par playlist :
`PlaylistId`, `MeasurementMatchesLeft` (placement), `Csr`/`HighestCsr` (`H5Csr{Tier 1-6,
DesignationId 0-5 Bronze..Onyx, Csr brut, PercentToNextTier}`). Mapping trivial vers
`PlayerPlaylistCSR` (Current=Csr, AllTime=HighestCsr, placement=MeasurementMatchesLeft).

Implémentation h5 (Phase 1, lifetime) :
1. Mapper `mapH5ArenaToPlaylistCSRs(arenaStats)` → `[]PlayerPlaylistCSR`.
2. Persister pendant le sync h5 (livesync post-sync) via `saveCSRSnapshots` dans la player DB h5.
3. Catalogue playlists ranked h5 (noms) : depuis l'API metadata officielle (flag `isRanked`,
   `cmd/h5-metadata-fetch`) → table `playlists_catalog` (comme Infinite), pour nommer + « Non classé ».
4. Tier **Champion (#N)** : étendre le resolver de paliers (DesignationId/seuils) en title-aware.
5. SeasonID : Phase 1 = un id « lifetime » fixe (le service record h5 = agrégat sans saison) ;
   Phase 2 = `seasonId` réel (calendrier saisons h5).

**BLOQUANTS à lever (vérif storage en cours)** :
- h5 a-t-il une **player DB** (stats.duckdb) où écrire `player_csr_snapshots` ? Sinon : cible alternative.
- La lecture `GetCareerCSRs` doit lire le STOCKÉ (repo), pas l'adapter live — confirmer le branchement.
- `player_csr_snapshots` doit être **append-only** (campagne ART, cf. `project_append_only_eradication_campaign`).

## ⚠️ BLOQUANT FONDATION (trouvé 2026-06-21, agent vérif) — provider shared HINF-only
**Le SharedReader injecté dans CHAQUE PlayerDB est `cfg.SharedProvider`, lié au seul chemin
Infinite** (mode B-swap = défaut prod, `main.go:355,493` ; `providerImpl` mono-path). Donc un
joueur h5 lit le `match_registry` **HINF** (vide pour h5) pour TOUTE lecture shared → la Match
View h5 tombe en **fallback live permanent**, et plus largement h5 ne peut **PAS LIRE** ses
matchs stockés en prod. La donnée EST stockée par `livesync` (`SharedDBPath(halo5.TitleSlug)`),
mais le chemin de LECTURE shared n'est pas par-titre.
- En mode kill-switch (`LEVELUP_USE_SHARED_PROVIDER=0`) la lecture irait sur le bon fichier h5 → c'est SPÉCIFIQUE au B-swap.
- Gap connu/planifié : `provider_multi_title_integration_test.go` (« commit 6 où main.go pourra itérer sur les titres » — **pas fait**).
- **FIX = provider shared PAR TITRE** : `Manager.For(SharedDBPath(titleSlug))` sélectionné selon `pdb.TitleSlug` (boot/`buildPoolConfig`/`player_resolver`). C'est LA fondation pour « h5 lit le stocké ». Prudence : machinerie B-swap délicate (mono-writer DuckDB, RO↔RW).
- Note : `player_csr_snapshots` (G4) est une table **player DB** (pas shared) → la lecture CSR n'est PAS bloquée par ce gap SI h5 a une player DB. À confirmer.

## FONDATION provider per-titre — LIVRÉE (Option A/C, vérif adversariale 2026-06-22)
Provider shared sélectionné PAR TITRE pour lecture ET écriture (le seul design sûr B-swap) :
- Lecture : `config.sharedReaderForTitle(titleSlug)` → `SharedManager.For(SharedDBPath(slug))` (player_resolver.go). HINF → même provider (byte-identique).
- Écriture h5 : `livesync.sharedProviderForPath` route le writer par le MÊME provider (PreSwap coordonne RO↔RW). Évite le « different configuration ».
- Vérif adversariale : propriété clé (read==write instance) SÛRE (cache par path, path identique, 1 Manager), HINF byte-identique, cycle de vie/concurrence SÛRS. Tests : #1 byte-identique, #2 sélection per-titre, #3 coordination read+write, #4 lifecycle.

### Résidus (non bloquants, à garder en tête)
- **Premier run h5** (shared inexistant) : **déjà mitigé** — `provisionAdditionalActiveTitles` (main.go:422→1505) provisionne le shared DB (vide + schéma via `migration.TargetShared`) de CHAQUE titre additionnel actif AU BOOT (avant `sharedMgr`/toute lecture). Donc `Manager.For(h5_path)` réussit (fichier existe) → lecture vide (pas de fallback HINF) tant que le sync n'a pas tourné. Le risque transitoire identifié par la vérif adversariale ne se présente PAS pour un titre actif (l'agent vérif ne connaissait pas ce provisioning). Réserve : un titre NON-`Active()` lu quand même n'aurait pas son shared → fallback HINF (mais les chemins excluent déjà les titres en pause).
- **`newEngineFor` (sync_handler.go:175,184)** injecte `cfg.SharedProvider` (HINF) : inoffensif (h5 passe par LiveRunner, pas SyncEngine) mais un 3e titre routé via SyncEngine hériterait du provider HINF → à corriger à l'activation 1b.
- Fallback lecture `sharedReaderForTitle` sur erreur = `cfg.SharedProvider` (HINF) : sémantiquement « lit le mauvais titre » plutôt que vide ; transitoire uniquement.

## VÉRIFS — RÉSULTAT (agent vérif, 2026-06-21)
- **10 commits = PROPRES** : zéro régression title-agnostic (slug figé supprimé, résolveurs injectés), zéro régression HINF, G9 bien conçu (repo-first/live-fallback, HINF jamais en fallback).
- **G9 lit le stocké ?** NON en prod, à cause du provider HINF-only ci-dessus (PAS un défaut de mes commits — le routage est bon, c'est la lecture shared qui n'est pas par-titre).
- Reste à confirmer : h5 a-t-il une **player DB** (pour G4) ? Saisons CSR h5 : résolution data-driven, queryabilité API 2026 non confirmée.

## RÉFÉRENCES
`.ai/PLAN_H5_ASSET_WIRING.md` (assets, 7 incréments) · `.ai/PLAN_H5_MATCH_VIEW.md` (G9) ·
mémoires `reference_h5_csr_model`, `reference_h5_metadata_official_api`, `project_convergent_sync_direction`,
`reference_bp_challenges_staleness_auth403`.
