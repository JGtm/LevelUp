# HANDOFF — Classification classé/non-classé (+ mode/PvE) Halo 5

> **Créé** : 2026-06-20. **Branche** : `feat/multititre-peripherie` (worktree `c:/Users/Guillaume/Downloads/Scripts/levelup-multititre`).
> **À reprendre** : pendant l'activation Halo 5 (1b), quand on aura la **data autoritative**.
> Lié : `.ai/HANDOFF_MULTITITRE_ACTIVATION.md`, mémoire `reference_ranked_playlists_source`, `reference_openspartan_import_isranked_false`.

## 0. TL;DR

La classification **classé/non-classé** (+ PvE/Firefight) de Halo 5 a son **substrat livré (2026-06-20)** : package réutilisable `internal/games/classification` (stratégie #1 set-membership) + TOML `ranked_hoppers.toml` (vide) + mapper h5 câblé + tests (cf. §5). Il **reste bloqué sur la donnée autoritative** (liste des HopperIds classés Halo 5) : tant qu'elle n'est pas déposée dans le TOML, `IsRanked` reste `nil` (conservateur, correct — ne mint jamais de faux CSR). À l'activation : remplir le TOML + ~3 lignes de câblage adapter. Ce n'est PAS un travail sur le chemin de sync Halo Infinite.

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

## 5. Implémentation — SUBSTRAT LIVRÉ (2026-06-20), data + câblage live restants

> **Le substrat de classification est construit cette session** (offline, test-couvert,
> byte-identique tant que les sets sont vides). Il ne reste, à l'activation, qu'à
> **(a) récupérer la liste autoritative** (§4), **(b) la déposer dans le TOML**, et
> **(c) un câblage adapter de ~3 lignes**. ZÉRO refonte.

**FAIT (commit de cette session) :**
1. ✅ `config/titles/halo_5/catalog/ranked_hoppers.toml` — **listes vides à dessein** (`ranked_hopper_ids = []`, `pve_hopper_ids = []`). Remplir = activer, zéro code.
2. ✅ **Stratégie réutilisable** `internal/games/classification/` (LEAF, n'importe ni canonical ni titre) :
   - `RankedClassifier` (interface, contrat de sortie STABLE `*bool` ; nil = indéterminé) ;
   - `SetClassifier` (stratégie #1 set-membership : set vide → nil partout ; set peuplé réputé EXHAUSTIF → présent `&true`, absent `&false`) ;
   - `LoadSetClassifier(path)` (fichier absent → vide sans erreur ; parse/schema_version → erreur).
3. ✅ `mapMatchSummaries(resp, gamertag, classifier)` pose `IsRanked`/`IsPvE` depuis le HopperId ; `h5MatchType` priorise les verdicts autoritatifs (ranked/firefight) sinon repli heuristique. **classifier nil/vide → IsRanked/IsPvE nil = comportement conservateur byte-identique.**
4. ✅ **Tests golden** : `classification/*_test.go` (set vide/peuplé/trim/nil-receiver/loader) + `halo_5/mapping_test.go::TestMapMatchSummaries_RankedClassifier` (classé→true+ranked, hors-set→false+social, PvE→true+firefight).

**RESTE (à l'activation, quand la data §4 est là) :**
- (a) **Remplir** `ranked_hopper_ids` / `pve_hopper_ids` avec la liste autoritative.
- (c) **Câbler dans l'adapter h5** (~3 lignes) : option `WithRankedClassifier(classification.RankedClassifier)` + champ sur `DataAdapter` ; au boot (`registerHalo5Adapters`, `server_titles_additional.go`) charger `classification.LoadSetClassifier(PathResolver…/catalog/ranked_hoppers.toml)` et l'injecter ; quand `LoadMatchSummaries` sera implémenté (Phase 2), lui passer `a.classifier`. (NON fait cette session pour éviter un champ inutilisé tant que `LoadMatchSummaries` renvoie `ErrCapabilityNotSupported` — même précédent que l'ingestion offline.)
- **Affiner** `h5MatchType`/mode-catégorie h5 (Slayer/CTF/Strongholds/Breakout/SWAT/FFA/Warzone — PAS la taxo HI) si on veut la granularité mode (optionnel, hors classé/PvE).
- **Validation live** : passe de sync h5 sur JGtm → matchs classés `is_ranked=true` alimentant le CSR (`reference_lusr_target_levels` pour le sanity check).

Note : `canonical.MatchType` n'a PAS été créé (les blueprints le proposaient à tort) — `canonical.MatchType`/`canonical.Experience` existaient déjà.

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

**État actuel (2026-06-20)** : le seam de sortie (canonical `*bool`) existe ✅ ; **la stratégie #1 (set-membership) EST désormais factorisée comme composant réutilisable** `internal/games/classification` ✅ (l'investissement qui rend Halo 7 gratuit — un futur titre fournit son TOML d'ids → `classification.NewSetClassifier`, zéro code). HI garde son `experience_rules` (stratégie #2 rules-based, non lifté — éviter de toucher le chemin CSR HI). La stratégie #3 (flag natif) n'est pas écrite (pas de consommateur ; une impl `RankedClassifier` ad hoc suffira le jour venu). h5/mapping.go consomme l'interface, **PAS de classif codée en dur**.

## 6. État des « 4 gaps » de l'audit (à la fermeture)

| Gap | État |
|---|---|
| 2 — indexation média title-aware | ✅ FAIT — commit `8ced9f154` (`ctxkeys.TitleSlug(ctx)`, zéro caller touché) |
| 3 — traductions maps title-aware | ✅ FAIT — commit `e5c83dbed` (`resolveMediaTitleSlug`) |
| 1 — `ExtractRegistry` is_ranked title-aware | ❌ NON-PERTINENT (chemin HI ; voir §1) — abandonné sciemment |
| 4 — classification ranked/mode h5 | 🟡 SUBSTRAT LIVRÉ 2026-06-20 (§5 : package `classification` réutilisable + TOML vide + mapper câblé + tests). Reste = data autoritative + câblage adapter ~3L (à l'activation). |
