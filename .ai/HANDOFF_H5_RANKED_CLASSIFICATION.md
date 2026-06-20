# HANDOFF — Classification classé/non-classé (+ mode/PvE) Halo 5

> **Créé** : 2026-06-20. **Branche** : `feat/multititre-peripherie` (worktree `c:/Users/Guillaume/Downloads/Scripts/levelup-multititre`).
> **À reprendre** : pendant l'activation Halo 5 (1b), quand on aura la **data autoritative**.
> Lié : `.ai/HANDOFF_MULTITITRE_ACTIVATION.md`, mémoire `reference_ranked_playlists_source`, `reference_openspartan_import_isranked_false`.

## 0. TL;DR

La classification **classé/non-classé** (+ catégorie de mode, + PvE/Firefight) de Halo 5 est un **TODO Phase 2 documenté dans l'adapter h5**, **bloqué sur la donnée autoritative** (liste des playlists/HopperIds classés Halo 5). Ce n'est PAS un travail sur le chemin de sync Halo Infinite. Tant qu'on n'a pas la liste autoritative, `IsRanked` reste `nil` (conservateur, correct — ne mint jamais de faux CSR).

## 1. Correction de l'audit (IMPORTANT — ne pas refaire l'erreur)

L'audit multi-titre (workflow `wdpdwir92`) + les blueprints d'implémentation (workflow `wupaqbr8i`) ont **conflé deux chemins distincts** :

- **FAUX** : « `isRankedPlaylist` / `ExtractRegistry` (internal/sync) HI-hardcodé → cassé pour Halo 5 ». **OpenSpartan = une source Halo INFINITE** (pas Halo 5). `internal/sync/transforms.go::ExtractRegistry` parse le JSON de l'**API Halo Infinite** ; il n'a **aucune référence à halo_5** (vérifié). **Halo 5 ne passe JAMAIS par ce chemin.**
- **VRAI** : Halo 5 a son **propre adapter** (`internal/games/halo_5/`). Son historique de match passe par `mapMatchSummaries` → `canonical.MatchSummary`, consommé par l'ingestion `internal/games/halo_5/ingest/CollectMatchBatch` (livrée cette session). `is_ranked` h5 = `MatchSummary.IsRanked` posé par l'adapter, **pas** par `ExtractRegistry`.

→ **NE PAS** refactorer `ExtractRegistry`/`isRankedPlaylist`/`determineModeCategory` en « title-aware » : ce serait de l'infra spéculative sur le **chemin critique CSR/LUSR** (risque) pour **zéro bénéfice Halo 5**. (Le bug HI récurrent `is_ranked=false` à l'import OpenSpartan est un sujet **HI data-quality** séparé — cf. `reference_openspartan_import_isranked_false`.)

## 2. Où vit la classification h5 aujourd'hui

`internal/games/halo_5/mapping.go` :
- `mapMatchSummaries` (l.120) → `canonical.MatchSummary` avec (l.145-159) :
  - `IsRanked: nil`  // « classification ranked = taxonomie HopperId (Phase 2) »
  - `IsPvE: nil`     // « detection warzone = Phase 2 »
  - `PairMode: nil`  // « h5 n'a pas de pair_name (Phase 2) »
  - `MatchType: h5MatchType(r.Id.GameMode)` (l.149)
- `h5MatchType(gameMode int)` (l.204) : `GameMode 1 = Arena (PvP) → social` faute de taxonomie ranked. La distinction ranked/social **exige le set de playlists classées h5**.
- La playlist h5 = `r.HopperId` (mappée en `assetRef("playlist", r.HopperId)`, l.150).

Côté ingestion (livré) : `ingest/registry.go:34` fait `IsRanked: derefBool(s.IsRanked)` → nil ⇒ `false`. **Dès que l'adapter posera `IsRanked`, l'ingestion le reprend automatiquement.** Idem `IsFirefight` ← `s.IsPvE`.

## 3. La question de design (tranchée)

**Faut-il une composante de LOGIQUE inter-titres pour classé/non-classé ?** → **Non.**

- Le **concept** est déjà canonique et partagé : `canonical.Experience` (ranked/social/btb/firefight) + `canonical.MatchSummary.IsRanked`. C'est l'output commun. ✅
- La **détermination** est intrinsèquement **par-titre** (HI : `Playlist.Tags`/`experience_rules.toml` ; h5 : HopperId ∈ set classé). Inputs différents ⇒ pas de logique partageable.
- Ce qui est **uniformisable** = le **pattern config-driven** : HI a `config/titles/halo_infinite/catalog/experience_rules.toml` + un loader. h5 adopte le **même format** (set de HopperIds classés) chargé par son adapter.
- **Doctrine projet** (`reference_ranked_playlists_source`) : la liste classée vient d'une **source autoritative par titre** (HaloDotAPI/Grunt, asset_id stables) — **jamais dérivée des parties**.

## 4. Le blocage

Il faut la **liste autoritative des playlists/HopperIds CLASSÉS de Halo 5**. Pistes :
- Source de référence type HaloDotAPI/Grunt pour Halo 5 (équivalent du set HI).
- OU l'endpoint skill h5 (`GetPlaylistCsr` par playlist — la présence d'un CSR = playlist classée ; cf. la doctrine HI `GetPlaylistCsr`).
- Le sondage live Halo 5 (Phase 1a, `cmd/probe-h5`) peut aider à récupérer les HopperIds réels rencontrés par JGtm, mais **ne pas dériver « classé » des parties** — il faut le set autoritatif.

## 5. Implémentation recommandée (quand la data est là)

1. `config/titles/halo_5/catalog/ranked_hoppers.toml` (nouveau) : liste des HopperIds classés h5 (+ éventuellement warzone/firefight HopperIds pour `IsPvE`).
2. Loader (générique si possible, calqué sur le loader experience_rules HI) → un set en mémoire dans l'adapter h5.
3. `mapMatchSummaries` : poser `IsRanked = &(hopperId ∈ rankedSet)` ; `IsPvE = &(hopperId ∈ warzoneFirefightSet)` ; affiner `h5MatchType`/`PairMode` (catégorie de mode h5 : Slayer/CTF/Strongholds/Breakout/SWAT/FFA/Warzone — pas la taxo HI Assassin/Fiesta/BTB).
4. **Tests** : golden `mapMatchSummaries` (HopperId classé → IsRanked=true ; non-classé → false ; inconnu → nil conservateur). Pur, offline-testable une fois le set défini.
5. **Validation live** : passe de sync h5 sur JGtm → vérifier que les matchs classés portent `is_ranked=true` et alimentent le CSR (cf. `reference_lusr_target_levels` pour le sanity check).

Note : `canonical.MatchType` n'a PAS besoin d'être créé (les blueprints le proposaient à tort) — `canonical.Experience` existe déjà et est l'enum de référence.

## 7. Extensibilité (Halo 7 et au-delà) — « sans tout recoder »

**Objectif** : un titre futur qui réutilise une méthode de définition « classé » DÉJÀ connue → **zéro code, juste de la data** ; une méthode NOUVELLE → une petite **stratégie** ajoutée, tout le reste inchangé.

Le seam qui garantit ça :
- **Contrat de sortie STABLE** : `canonical.MatchSummary.IsRanked` / `canonical.Experience`. Ne change JAMAIS. Tout consommateur (ingestion, CSR, LUSR, UI) lit ça, agnostique du titre.
- **Détermination = STRATÉGIE config-driven choisie par l'adapter du titre**, PAS du code bespoke réécrit à chaque titre. Bâtir une petite **librairie de stratégies réutilisables** :
  1. **Appartenance à un set autoritatif de playlist-ids classés** (l'approche HopperId de h5). Un titre fournit une liste TOML d'ids classés → `IsRanked = playlistId ∈ set`. **La plus universelle** (tout Halo a des playlists + une notion autoritative de « classé »). Réutilisation = **data seule**.
  2. **Règles sur métadonnées de playlist** (l'approche `experience_rules.toml` de HI : préfixes de nom, tags). Un titre fournit ses règles. Réutilisation = **data seule**.
  3. **Flag natif par match** (si un futur titre encode « ranked » directement dans le payload de match) → petite stratégie de lecture du flag.

**Règle de conception à TENIR** (sinon retour de la dette) : quand on implémente la classif h5 (§5), la coder comme une **stratégie réutilisable et paramétrée** (ex. `RankedBySetClassifier(ids)`), PAS comme une fonction `halo_5`-spécifique. Ainsi Halo 7 :
- même méthode que h5 (set d'ids) → `config/titles/halo_7/catalog/ranked_*.toml`, **0 code** ;
- même méthode que HI (rules) → réutilise le loader `experience_rules`, **0 code** ;
- méthode inédite → 1 nouvelle stratégie (~30-50L) + l'adapter Halo 7 la sélectionne ; ingestion/CSR/LUSR/UI **inchangés** (ils lisent le canonique).

**État actuel** : le seam de sortie (canonical) existe ✅ ; la librairie de stratégies n'est PAS encore factorisée (HI a son `experience_rules` semi-spécifique, h5 = Phase 2). → À l'activation, livrer la classif h5 **en factorisant la stratégie #1 comme composant réutilisable** (c'est l'investissement qui rend Halo 7 gratuit). Ne pas la coder en dur dans `halo_5/mapping.go`.

## 6. État des « 4 gaps » de l'audit (à la fermeture)

| Gap | État |
|---|---|
| 2 — indexation média title-aware | ✅ FAIT — commit `8ced9f154` (`ctxkeys.TitleSlug(ctx)`, zéro caller touché) |
| 3 — traductions maps title-aware | ✅ FAIT — commit `e5c83dbed` (`resolveMediaTitleSlug`) |
| 1 — `ExtractRegistry` is_ranked title-aware | ❌ NON-PERTINENT (chemin HI ; voir §1) — abandonné sciemment |
| 4 — classification ranked/mode h5 | ⏸ CE HANDOFF — Phase 2 adapter, bloqué data autoritative |
