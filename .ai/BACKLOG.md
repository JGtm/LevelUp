— Tâches et TODO centralisés

---

## 🔄 Aucune tâche en cours

---

## 📋 Backlog

---

### [prod/VPS] Constats audit post-deploy 2026-07-16 (lecture seule) — 4 suites à traiter

> Noté le 2026-07-16 (post-deploy c3f2aa6fb, audit ssh en lecture seule — rien modifié).

1. **AUCUNE sauvegarde automatique en prod** : `restic` absent de l'hôte → le scheduler backup se désactive au boot (WARN). Décider et installer une stratégie (restic + dépôt distant + secrets) — priorité haute pour un service avec données joueurs.
2. **`LEVELUP_HALOAPI_KEY` absente du VPS** → seed `team_colors` H5 impossible en prod en l'état. Options : (a) ajouter la clé au `.env.local` prod + one-shot `h5-metadata-fetch` pendant une fenêtre d'arrêt courte ; (b) régénérer `metadata-prebuilt.zip` depuis la DB locale déjà seedée et remplacer la metadata prod pendant l'arrêt (attention : `extractPrebuiltMetadataIfAbsent` ne réextrait pas si le fichier existe). Tant que non fait : la match view H5 garde le fallback d'équipes.
3. **nginx** : fichier mort `sites-enabled/levelup.bak-1781365391` (même `server_name` que le vhost actif → warnings « conflicting server name » à chaque `nginx -t`) à supprimer ; vhost réel très divergent du template `packaging/nginx/levelup.conf` (locations consolidées, `auth_basic` commenté, timeouts 3600s, commentaire « Streamlit » obsolète) — réaligner ou assumer et documenter.
4. **Observabilité media tooling** : la prod loggue en `LEVELUP_LOG_LEVEL=warn` → la ligne INFO « ffmpeg disponibles » du boot est invisible (seuls les problèmes sortiraient en WARN — le contrat d'alerte est rempli, mais le positif est silencieux). Si l'on veut la preuve positive au boot en prod : passer cette ligne en WARN-neutre ou l'exposer dans /health. Confort, pas un bug. Vérifié par ailleurs : ffmpeg Debian 5.1.9 possède tous les encodeurs/muxers requis (`check-env` tout [OK]).

**Effort** : 1 = chantier dédié (secrets/politique) ; 2 = décision + 15 min ; 3 = rapide ; 4 = cosmétique.

---

### [dette/web] squad/v2 largement mort (post-HistoryTable) — inventaire knip à purger

> Noté le 2026-07-16 (train corrections v7, salve 4). `HistoryTable` v2 et `TimeseriesCombatYield` ont été supprimés ; knip signale le reste de `squad/v2` comme inutilisé : exports de `types.ts` (`AssetReference`, `SquadHeader`, `SquadPageV2Response`…), `SquadCombatProfileRow.tsx`, `squad/v2/queries.ts`. Vérifier les consommateurs réels puis purger (règle 0 code mort — git garde l'historique). **Effort** : rapide.

---

### [dette/api] `GET /players/{slug}/battlepass` (Home) — consommateur à vérifier, supprimer si mort

> Noté le 2026-07-16 (train corrections v7, salve 4). La home lit ses défis via `/pages/palmares/season-pass` ; `/battlepass` semble sans consommateur web direct. Vérifier (web + CLI + tests métier) puis supprimer si mort — même traitement que l'ex-route `/challenges` Home (supprimée salve 4 après preuve de mort). **Effort** : rapide.

---

### [data/h5] Registre d'armes H5 — long-tail de 31 weapon_ids sans rôle

> Noté le 2026-07-16 (fix donut « Frags par type d'arme » H5). Le registre couvre 35/66 `weapon_id` réels de `v_weapon_kills` (233 274 frags répartis sur 8 rôles) ; ~31 IDs restants (variantes/UGC) restent sans rôle — dégradation gracieuse, absents du donut. Étendre `weaponRegistryH5Stock` si l'exhaustivité est voulue. Chantier data (mapping manuel). **Effort** : moyen.

---

### [ux/relations] PLAN_RELATIONS_UX_2026-07 — lots restants non exécutés

> Noté le 2026-07-16. Seul le lot H (chip « Multi-jeux ») a été livré (train corrections v7). Le reste du plan — renommage du toggle `includeFriends`, refonte des CoreCards, tri du tableau, etc. — est en attente : voir [.ai/PLAN_RELATIONS_UX_2026-07.md](.ai/PLAN_RELATIONS_UX_2026-07.md).

---

### [i18n] Tiers de rangs — duplication du mapping FR + cache des défis mono-langue

> Noté le 2026-07-16 (lot i18n du train corrections v7). Deux suites : (1) le mapping des noms de tiers existe en 3 endroits (grille `lib/skillTiers.ts`, `CSR_TIER_FR` d'`ExplorerTargetSeasonCSR`, helper `localizeTierLabel` dérivé) → centraliser en source unique + garde-rail (règle ≤2 copies, la 3e impose le helper) ; (2) les titres de défis mis en cache par le fetch background sont bakés à la langue du fetch (FR) — seul le chemin live est locale-aware ; invalider ou re-résoudre à la bascule de langue. **Effort** : rapide à moyen.

---

### [infra/release] Build de l'image en CI → pull registry (au lieu de `docker build` sur le VPS)

> Noté le 2026-07-16 (audit release, reco P3). Aujourd'hui l'image est reconstruite sur le VPS à chaque deploy : un `apt-get` cassé ne se découvre qu'en prod, pendant le déploiement. Pré-builder en CI (GHCR) attrape l'échec avant et accélère le deploy. Déjà esquissé dans `RUNBOOK_GO_LIVE.md` (§ évolutions). **Effort** : moyen (workflow + deploy.sh).

---

### [sécurité/autZ] Routes prestige : réconciliation acteur user_id ↔ chemin (BOLA résiduel) — à corriger avant exposition élargie

> Noté le 2026-06-10 (revue `feat/coach-v3-squad`). **MàJ 2026-07-16 (train corrections v7)** : l'état décrit initialement est périmé — les routes prestige sont aujourd'hui montées **SOUS** `/players/{player_slug}` (ownershipMW, ADR 0029) et tout le client passe par le chokepoint `scopedToPlayer` (challenges déplacées sous `/prestige/challenges*`, garde-rail anti-collision `route_collision_test.go`). Le fix squad-only `ActorGuard` est livré.

**Problème (résiduel)** : les handlers non-squad lisent l'acteur depuis le **body/query** (`user_id`, `created_by`) sans le réconcilier avec le `{player_slug}` du chemin. `ownershipMW` vérifie le segment d'URL, pas le payload : un utilisateur authentifié A peut passer la garde avec SON slug dans le chemin et cibler B via `user_id` dans le body (BOLA horizontal). Blast radius limité (instance invités-only + lockdown) mais incohérent avec le reste.

**Note harness** : en mode démo le bundle prestige ne se monte pas (pas de DuckDB) → le test anti-collision générique ne couvre pas prestige-vs-autres ; la paire home+prestige est couverte par un test ciblé.

**Déjà fait (squad-only, cette branche)** : helper `squadActorGuard` (`server.go`) + `PrestigeHandler.WithActorGuard` → 403 `player_forbidden` si l'acteur n'est pas un profil possédé par la session (réutilise `authz.Enforced`/`CurrentUser`/`CanAccessPlayer`, multi-profil famille). Transparent demo/auth-off. Tests httptest 403 sur les 6 routes squad.

**À faire (reste)** : étendre la même garde aux endpoints prestige pré-existants — soit (a) per-handler sur `user_id` (`CreateArc`, `ListActiveChallenges`, `GetMyPrestige`, `SuggestTemplates`, `DeleteArc`, `pilot-mode`, …), soit (b) un middleware sur le sous-groupe. Audit complet des handlers lisant `user_id`/`created_by`/`requested_by` requis. Tests 403 par endpoint.

**Note design** : « dériver de la session » = en pratique « vérifier que le slug client ∈ profils possédés (`CanAccessPlayer`) » pour gérer le multi-profil famille, pas un slug unique figé.

**Effort** : moyen (audit + N handlers + tests ; touche des features livrées → PR dédiée).

---

### [POST-V7] Housekeeping post-cutover (optionnel, non bloquant)

> Le cutover Go (la branche Go est devenue `main`) est **terminé** — cf. archive « Récemment complété ».
> Reste 2 micro-tâches optionnelles, non bloquantes :
- [ ] Documenter le default async ON dans le README utilisateur
- [ ] Tuning du janitor (24h → 12h ?) si la latence WAL le justifie en prod

---

### [Migration] Cible desktop Tauri web-first, sans réécriture Rust métier — ⏸️ GARDÉ DE CÔTÉ

> ⏸️ **Gardé de côté** (2026-06-09) : conservé pour distribution desktop néophyte future. Note : le cutover Go étant fait, le « backend Python local packagé » ci-dessous doit se lire **backend Go local** — à re-cadrer si réactivé.

**Noté le** : 2026-04-12 | **Priorité** : Moyenne (distribution simplifiée, non bloquante pour les slices MVP)

**Référence plan** : `.ai/MIGRATION_MASTER.md`, `.ai/migration/DECISIONS.md`

**Problème** : La migration React/FastAPI améliore l'UX et le déploiement web, mais ne résout pas à elle seule le cas utilisateur néophyte qui ne doit ni installer Python, ni lancer `pip`, ni manipuler un terminal. Il faut documenter une cible desktop installable qui n'abîme pas la stratégie web/VPS.

**Décision cible** : Conserver une architecture **web-first** (`apps/web` + `apps/api`) comme source de vérité produit, puis ajouter **Tauri comme coque desktop** optionnelle. Rust est explicitement **hors périmètre métier** : aucune logique de sync, auth Halo, DuckDB, filtres, agrégats, visualisations ou contrats API ne doit être réécrite en Rust.

**Solution** : Préparer un spike de packaging Tauri autour du frontend React existant et d'un backend FastAPI/Python local packagé, avec un contrat d'intégration minimal et réversible.

**Changements ciblés** :
1. Architecture : figer la règle `React navigateur d'abord`, `FastAPI canonique`, `Tauri simple shell desktop`
2. Packaging : définir comment lancer/arrêter proprement le backend Python local depuis l'app desktop, avec gestion des logs, ports, répertoires de données et erreurs de démarrage
3. Frontend : isoler les appels natifs desktop derrière une couche d'adaptation pour que l'app reste exécutable telle quelle sur navigateur et sur VPS
4. Données locales : cadrer les chemins Windows pour DuckDB, médias, cache et configuration utilisateur sans hardcoder de chemins machine
5. Distribution : évaluer installateur Windows, taille du bundle, temps de démarrage et absence de prérequis Python côté utilisateur final
6. Exploitation : préserver explicitement la cible VPS en interdisant toute dépendance produit au runtime Tauri/Rust
7. Go/no-go : définir les critères du spike (installation propre, backend embarqué stable, auth utilisable, fichiers locaux OK, perf de lancement acceptable)

**Point de vigilance** : Tauri implique mécaniquement une fine couche Rust côté shell. Ce point est acceptable uniquement comme détail d'enveloppe technique. Toute dérive vers des commandes Rust métier, un stockage canonique côté Tauri ou une divergence desktop-only dans les flux React/FastAPI doit être refusée.

---

### Kills environnementaux — catégorie dédiée (v8++)

> ⚠️ **Spec à re-écrire pour Go (2026-06-09)** : les étapes ci-dessous référencent le code Python supprimé au cutover (`constants.py`, `_weapon_kills_repo.py`, `ParticipantBits`, `GRENADE_MEDALS`). L'idée reste valable mais doit être re-spécifiée côté Go (`apps/go-api`) avant toute implémentation. Priorité très basse (barrel kills extrêmement rares).

**Contexte** : La médaille **Kong** (kill via baril projeté) est actuellement comptée dans `GRENADE_MEDALS` faute d'une meilleure catégorie. Ce classement est approximatif — il est impossible de savoir avec certitude si l'API inclut ces kills dans `GrenadeKills` ou non.

**Idée** : Créer une catégorie `environmental_kills` (ou `environmental`) pour regrouper les kills causés par l'environnement sans arme tenue :
- Baril projeté (médaille **Kong**)
- Potentiellement : chutes provoquées, explosions de véhicules, etc.

**Ce que ça impliquerait** :
1. Nouvelle colonne `environmental_kills` dans `match_participants` (migration DuckDB)
2. Nouveau bit `ParticipantBits.ENVIRONMENTAL_KILLS` dans `constants.py`
3. Retirer `Kong` de `GRENADE_MEDALS` → nouvel ensemble `ENVIRONMENTAL_MEDALS`
4. Logique de réconciliation filmshell dédiée dans `_weapon_kills_repo.py`
5. Backfill pour l'historique existant
6. Affichage UI éventuel

**Complexité estimée** : Moyenne (surtout le backfill + validation que l'API expose bien des compteurs séparés)

**Priorité** : Basse — les barrel kills sont extrêmement rares, l'impact sur les stats est négligeable. À faire uniquement si on veut une exhaustivité totale des catégories de kills.

---

## 🎮 Backlog — Coach proactif × Prestige (post-V2)

Référence : ADR 0020 — Coach proactif : pont vers Prestige. ADR 0021 — Synthèse dynamique de Template et Arc ad-hoc.

Ce backlog liste les extensions volontairement reportées **après** la livraison V2 du pont coach → Prestige. Chaque entrée doit faire l'objet d'une décision produit séparée avant ouverture d'une nouvelle branche.

### [coach/prestige] V2.1 — Plumbing `Source` → `prestige_telemetry.source` — ⏸️ PARKÉ

> ⏸️ **Parké (2026-06-09)** : **but final = tuner le coach** (mesurer si les défis proposés par le coach sont acceptés/complétés davantage que ceux créés par le joueur, et réinjecter dans `synthesis_grammar.toml`). Mais **personne n'a aujourd'hui de question de tuning concrète** → écrire la télémétrie serait « écrire pour personne ». À ne ressortir **que** quand un besoin analytics réel émerge (définir d'abord la question, puis le consommateur, puis le plumbing). Vérifié : champ `Source` existe dans `CreateChallengeRequest` mais n'est pas propagé à la table, et `prestige_telemetry` n'est lue nulle part.

**Idée** : ajouter une colonne `source` dans la table `prestige_telemetry` et propager `CreateChallengeRequest.Source` jusqu'à `EmitCreated` (puis aux EmitTransition pour suivre le devenir des challenges coach).

**Bloqueur actuel** : la table `prestige_telemetry` est écrite mais jamais lue. Aucun script, aucun handler, aucun dashboard ne l'interroge. Le commentaire dans `types.go` mentionnant `analyze_prestige_tuning.py` réfère à un script qui n'existe pas — c'était une intention V1, jamais matérialisée.

**Pré-requis avant d'implémenter** :
- Construire d'abord un consommateur : soit un endpoint `GET /diag/prestige/telemetry` qui agrège (taux acceptance par source, complétion par source), soit un analyseur CLI Go (`cmd/prestige_tuning_analyze`) lisant directement la DB.
- **PAS DE PYTHON** : analyseur en Go ou DuckDB CLI direct (`duckdb stats.duckdb -c 'SELECT ...'`).
- Définir les métriques cibles : ratio accept/reject par source, taux completion coach vs user vs pilot_mode, distribution de strength des proposals acceptées.

**Sans consommateur** : ajouter la colonne maintenant = écriture pour personne, dette de schéma qui grossit. Reporter jusqu'à ce qu'un besoin analytics concret émerge.

**Effort estimé** : rapide (ALTER TABLE + 2 lignes Go) — mais l'analyseur côté consommation est moyen (1 commit endpoint diag ou CLI).

---

### [coach/prestige] V3 — Apprentissage automatique de `synthesis_grammar.toml` — ⏸️ PARKÉ

> ⏸️ **Parké (2026-06-09)** : downstream direct de la télémétrie ci-dessus (lit `prestige_telemetry`). Bloqué tant que V2.1 est parké. À reprendre seulement après qu'un consommateur de télémétrie existe et qu'un besoin de tuning concret est posé.

**Idée** : analyser `prestige_telemetry` pour ajuster la grammaire de synthèse. Si les templates synthétisés sur metric=X ont taux de complétion < 30 % sur 50 acceptations, retirer X de la grammaire ou réduire ses windows autorisés.

**Pré-requis** :
- Job analytics qui lit `prestige_telemetry` + `coach_proposal`.
- Mécanisme de PR automatique sur `synthesis_grammar.toml` (ou ajustement runtime via override en DB metadata).
- Validation manuelle obligatoire avant application.

**Effort estimé** : lourd. Demande infra analytics.

---

### [coach/prestige] V3 — Notifications push externes — ⏸️ GARDÉ DE CÔTÉ

> ⏸️ **Gardé de côté (2026-06-09)** : conservé tel quel. Vérifié : notifications coach **in-app uniquement** aujourd'hui (`player_notifications`). Pas de plan dédié pour l'instant (bloqué sur décision produit + infra push mobile inexistante).

**Idée** : émission externe (push mobile, email, Discord) des proposals coach les plus fortes. Aujourd'hui les notifications restent in-app (`player_notifications` UI uniquement).

**Pré-requis** :
- Décision produit (sécurité / vie privée).
- Infra push mobile (non disponible aujourd'hui).
- Préférences fines par catégorie de notif.

**Effort estimé** : lourd.

---

**Ordre de priorisation suggéré (coach/prestige V3)** — voir [.ai/PLAN_COACH_V3_GENERATION.md](.ai/PLAN_COACH_V3_GENERATION.md) :

1. Phase A — Coach négatif soft (backend testable vite ; front gaté sur mini-spec UX)
2. Phase B — Coach tone (mutualisable i18n avec Phase A)
3. Phase C — Squad coach (la plus lourde : coach + coach_advisor + prestige + front squad)
4. Cross-titre arcs backend ([.ai/PLAN_CROSS_TITLE_ARCS_BACKEND.md](.ai/PLAN_CROSS_TITLE_ARCS_BACKEND.md)) — à anticiper avec l'arrivée du 2e titre
5. Parkés : V2.1 télémétrie + auto-grammar (besoin analytics à définir d'abord)

---

## ✅ Récemment complété (référence)

| Date | Item |
|------|------|
| 2026-06 | **[POST-V7] Go-live / cutover Python → Go** — la branche Go est devenue `main` (code applicatif Python `src/` retiré). Cutover terminé ; reste 2 micro-tâches optionnelles (doc async, tuning janitor) listées en backlog actif. |
| 2026-06 | **[V8/Compare] CSR + CSR ATH (re-implémentation)** — cron CSR mondial autonome (`internal/scheduler/world_leaderboard_cron.go`, snapshots append-only), champs `HighestCSR*` restaurés dans `domain/compare.go` + `compare_service.go::applyCSRSummary`, exposés au front (`highest_csr`/`csr_alltime`). Commits `1693e6e1`, `aeaaffcd` (Phase 2 livrée). |
| 2026-06 | **[auth/unification] Consolidation ADR 0023 — read-path + watcher daemon → MultiUserTokenStore** — phases 3a/3b/3c livrées (`ab0ebefa`, `9eb9b738`, `5c7d87a8`) : tous les chemins lisent le multi-user store en priorité (fallback legacy), tracker auto-découvert au boot, migration boot-time `MigrateLegacyTokens`. Couvre les items backlog « PR 2.5b watcher migration », « read-path switch » et « auth/cleanup migration ». PRs A–D du lockdown auth livrées (`2e2357db`, `00cb920c`, `58bf9ec7`, `d9fcb178`). |
| 2026-06 | **[persist/safety] Gap [D] circuit breaker** — `internal/persist/queue.go:69-79` (`consecutiveFailures` + seuil 5, `ErrDrainCircuitBreaker` fail-fast ~1s) + 5 tests verts `queue_circuit_breaker_test.go`. Gaps restants (G/E + A/B optionnels) → [.ai/PLAN_PERSIST_ROBUSTNESS.md](.ai/PLAN_PERSIST_ROBUSTNESS.md). |
| 2026-06 | **[db-concurrency] leased-writer-enforcement intégré** — `internal/platform/dblease/` (`LeasedWriter`, `AcquireWriterCtx`) mergé et utilisé dans 10+ sites sync (commits `351798cc`, `65ca246d`, `f50f5753`). Conditions de déblocage backlog (build cgo, baseline, coordination) levées par le go-live. |
| 2026-05/06 | **[frontend/nav] Nettoyage `PlayerScopeNav`** — composant mort supprimé (cleanup knip `4b73b584`). Les constantes `PLAYER_*_NAV_ITEMS` sont conservées (toujours consommées par `pageTitle.ts`, donc non mortes). |
| 2026-05/06 | **[feedback-drawer] Drawer feedback + sync labels** — feature mergée sur `main` (`apps/web/src/features/feedback-drawer/`), `.github/workflows/sync-labels.yml` + `.github/labels.yml` présents. Labels GitHub confirmés présents (2026-06-09) → item entièrement clos. |
| 2026-05 | **[Go/PR 7] Sync Engine Migration to dblease** — 17 sites `AcquireLeaseCtx` → `AcquireWriterCtx` migrés (engine.go ×10, backfill_weapons.go ×1, citations_backfill.go ×2, friends_recompute.go ×2, session_recalc.go ×2). Deprecation comment sur legacy facade. |
| 2026-05 | **[Go/PR 4-6] Leased-Writer-Enforcement Foundation** — type `LeasedWriter` + interfaces `DBExecutor`/`DBWriter`, expvar metrics `dblease_acquire_total{kind,status}`, 26 tests intégration (burst, coordination, atomicity), corrections fixtures (global schema), CI workflow updates (go-lease-enforcement, go-baseline-tests jobs), preservation 1662 tests baseline. |
| 2026-05-24 | **[auth/unification] E.v2 — Pool.AddOrUpdateSource + periodic re-scan** (commit `4508df92`) : hot-add ou refresh d'un slot, goroutine main.go 15min tick, 5 tests TDD GREEN. |
| 2026-05-24 | **[auth/unification] PR 2.5b phase 1 — RefreshLoop.WithMultiUserMirror** (commit `157d80a8`) : mirror write legacy → multi-user, 3 tests TDD GREEN. |
| 2026-04-28 | **[Multi-titre] Migration `static/` vers arborescence title-scopée** — Plan finition multi-titres Phase 6 livré (branche `feat/multi-title-static-fs-rescope`, 6 commits). Couche 2 `internal/assets/static/` (35 tests) + couche 3 `TitleAssetURLAdapter` HI + ST_B stub + bascule des 5 callers Go (A1–A5, C1, F) + frontend `apps/web/src/lib/staticAssets.ts` (D1–D2) + big bang atomique (328 fichiers `git mv` + 180 rows UPDATE DB + flag flip + fixtures D3+D4) + cleanup Phase 6.6 (suppression flag + script jetable + dead branches). H5G/HI renames vers slugs canoniques longs. |
| 2026-04-10 | **Score de forme individuel + escouade** : `compute_form_score_history()` (Polars rolling avg_14 - avg_90), `load_full_performance_history()` (DB query), `plot_form_score_history()` (Plotly multi-lignes + fill). Intégré en tête de l'onglet Résumé (Timeseries) et avant "Taux de victoires vs historique" (Teammates). st.metric + graphe historique avec points session surlignés. |
| 2026-04-06 | **Discord i18n — assets résolus par ID dans l'embed** : `fetch_last_match_info()` remonte `map_id`/`playlist_id`/`pair_id`/`game_variant_id` + libellés EN bruts ; `src/utils/_discord_embed.py` résout désormais les traductions via `asset_translations` selon `discord_lang`, avec fallback unique vers l'anglais en BDD. Les colonnes `*_fr` de `v_match_full` ne sont plus utilisées dans ce flux. Tests ciblés : 138 passés (`test_discord_notifier.py`, `test_translations.py`, `test_delta_sync.py`). |
| 2026-03-30 | **i18n — Table `asset_translations` peuplée dans `metadata.duckdb`** : 9 674 traductions (698 assets × 14 langues BCP-47). Script `populate_asset_translations.py` réécrit avec `_build_version_id_cache()` (version_id SPNKr requis, `""` → 404), parallélisme `asyncio.gather` sur les 14 langues, reprise possible. |
| 2026-03-30 | **Fix critique — `v_match_full` sans traductions en prod** : `_try_attach_meta_for_views()` cherchait `meta.maps` (table absente en v6) → toujours `None` → vue créée sans JOINs i18n. Fix : vérifier `meta.asset_translations`. `_create_v_match_full()` : suppression des 4 JOINs legacy (`meta.maps/playlists/playlist_map_mode_pairs/game_variants`), 8 JOINs `asset_translations` (en-US + fr-FR × 4 types). Vue recréée en prod : "Starboard"→"Tribord", "The Pit"→"La fosse", etc. |
| 2026-03-30 | **Docs — Renommage ARCHITECTURE_V5 → V6** : `git mv` + mise à jour contenu (titre, version 6.3.0, `shared_matches_v2.duckdb`). §6 asset_translations ajouté dans la version FR. Toutes les références mises à jour : `CLAUDE.md`, `README.md`, `README_FR.md`, `FR/README.md`, `FR/COMMANDS.md`, `.ai/project_map.md`, `.ai/START_HERE.md`. |
| 2026-03-30 | **Docs — CHANGELOG 6.3.0** : entrées EN + FR documentant `asset_translations`, refonte `v_match_full` v6, fix `_try_attach_meta_for_views`. |
| 2026-03-30 | **Normalisation des labels de modes de jeu (v6.2.1)** : `resolve_display_mode()` dans `src/analysis/mode_display.py`, colonne `canonical_category` dans `mode_prefix_names`, 29 overrides dans `mode_pair_overrides`, `translate_pair_name` délégue au resolver, fichier plat de contrôle généré et validé. |
| 2026-03-30 | **Audit KDA locaux → `efficiency` (v6.2.1)** : sémantiques séparées — `p.kda` API conservé per-match, agrégats session/carte/cumul renommés `efficiency`/`session_efficiency` ; clés i18n `efficiency`/`efficacité` ajoutées ; 6 modules `src/analysis/` mis à jour (`cumulative.py`, `stats.py`, `_performance_relative.py`, `_performance_relative_helpers.py`, `_performance_session.py`, `stats.py` domain model). |
| 2026-03-27 | **Bug — `index_media.py --force` levait `ConstraintError: Duplicate key`** : quand `force_rescan=True`, `existing` était laissé vide `{}` → toutes les entrées considérées "nouvelles" → INSERT sur des clés déjà présentes. Fix : `existing` est toujours chargé depuis la DB ; `force_rescan` contourne uniquement le filtre delta `mtime`. Ré-indexation JGtm (73 médias) exécutée avec succès après fix. |
| 2026-03-26 | **Bug critique — `mv_player_matches` recalcule le KDA au lieu de lire la valeur API** : vue recréait `(kills + assists/3)/deaths` au lieu de `COALESCE(p.kda, fallback)`. Fix : détection dynamique `has_kda_col` (même pattern `has_enemy_mmr`) + génération SQL conditionnelle. |
| 2026-03-26 | **UX — Score d'équipe supérieur aux scores individuels (En-tête Page Coéquipiers)** : carte équipe n'affichait pas les bonus collectifs. Fix : `_render_compact_team_card` calcule `bonus = score - base_avg` et affiche `"moy. X (+Y collectif)"` quand > 0. |
| 2026-03-26 | **Bug — Colonne "Dernière rencontre" incohérente (Page Match · Encounters)** : SQL `MAX(start_time)` incluait le match courant et les matchs futurs. Fix : `filter_past` CTE + `_fetch_match_start_time` helper + guard `days = max(0, delta.days)` + colonne renommée "Précédente rencontre" + "1ère rencontre" pour les nouvelles têtes. |
| 2026-03-26 | **Bug annexe — `datetime.utcnow()` déprécié dans `career_lusr.py`** : remplacé par `datetime.now(timezone.utc).replace(tzinfo=None)`. |
| 2026-03-26 | **Bug — Médias mal rattachés aux matchs (décalage fuseau horaire)** : `epoch(capture_end_utc)` → `epoch(timezone('UTC', capture_end_utc))` dans `associate_with_matches()` + EXIF naïf ignoré (heure locale caméra, pas UTC). Ré-indexation requise (faite pour JGtm le 2026-03-27). |
| 2026-03-26 | **Bug RÉCURRENT CRITIQUE — Session escouade absente du graphe "Évolution de la performance"** : root cause A (fanout ouvrait shared en R/W → conflit handle Streamlit) fixée via Phase J (`shared_read_only=True` dans `_engine_fanout.py`). Fix défensif LEFT JOIN dans `_performance_squad._join_perf_frames()`. Les deux chemins de fix documentés dans l'audit sont implémentés. |
| 2026-03-26 | **Bug — Stats coéquipiers absentes (Page Teammates)** : résolu par le fix fanout R/O (Phase J). La root cause était identique au bug session escouade — fanout silencieux → PME coéquipier non créées. À revalider sur la prochaine session de jeu. |
| 2026-03-26 | **Bug annexe — `get_sync_metadata` lit mauvaise DB** : `SELECT last_sync_at FROM meta.sync_meta WHERE xuid=?` → `SELECT value FROM sync_meta WHERE key='last_sync_at'` dans la player DB. Fix commité dans `_diagnostic_repo.py` (Phase F). |
| 2026-03-26 | **Piste — Crashes silencieux (Page Coéquipiers · Top medals)** : source principale (connexions zombies fanout R/W) supprimée par Phase J. Si non récurrent → archivé. |
| 2026-03-21 | **Bug — Frags vs. détail armes (double-comptage melee)** : melee kills filmés attribués à l'arme tenue + `melee_kills` API → double-comptage. Fix : remainder `api_total - film_kills` dans 3 fichiers + `load_total_kills_for_player()` + 2 nouveaux tests. |
| 2026-03-21 | **UI — Graphe stats/min escouade : morts sous l'axe** — `plot_per_minute_timeseries` : deaths tracées en négatif (`dpm_neg`), `customdata[5]` = valeur absolue, `hover_dpm_neg` i18n, ticks Y absolus via `build_symmetric_abs_ticks` (extrait dans `src/visualization/_permin_helpers.py`). `timeseries.py` à exactement 500L. |
| 2026-03-21 | **Maintenance — Nettoyage dossier `scripts/`** — 10 scripts investigation → `scripts/investigation/` + README ; `cleanup_legacy_tables.py` + `cleanup_player_dbs_v5.py` → `scripts/_archive/` ; `.tmp.*` supprimés. |
| 2026-03-21 | **CI — Scripts exclus par `.gitignore`** — `check_code_size.py` → `enforce_size_limits.py` ; `check_imports.py` → `validate_imports.py` ; stubs `test_page_router_smoke.py` + `test_page_router_regressions.py` créés. Références mises à jour dans `ci.yml`, `.pre-commit-config.yaml`, `test_code_quality.py`. |
| 2026-03-21 | **UI — Notation de session escouade (Page Coéquipiers)** — `compute_squad_performance_score()` dans `src/analysis/_performance_squad.py` ; `SQUAD_GRADE_THRESHOLDS` + `resolve_squad_grade()` dans `performance_config.py` ; `render_squad_session_header()` + `_render_squad_score_block()` dans `src/ui/components/performance.py` ; 7 clés i18n `squad_grade_*` dans `src/ui/i18n/pages/teammates.py` ; bloc tendance K/D remplacé dans `teammates.py` ; 18 tests unitaires. |
| 2026-03-21 | **Perf — `_MAX_CONCURRENT_CHUNKS`** : déjà à 50 en production (`weapon_extraction_service.py`). Tâche obsolète — objectif déjà atteint. |
| 2026-03-19 | **Medal definitions en BDD** — table `medal_definitions` dans `metadata.duckdb` (167 médailles, DB-first + JSON-fallback). Migration, script population, CLI `--medal-metadata`, `MedalsMixin.load_medal_definitions()` / `get_medal_label()`, UI DB-first dans `medals.py`, 16 tests unitaires + 4 intégration. Orphan `citations_{fr,en}.json` supprimés. |
| 2026-03-19 | **Phase 8 — Couche centralisée médailles** (`medal_definitions.py`) — `src/data/medal_definitions.py` source canonique unique ; `_medal_data.py` thin re-export ; `medals.py` wrapper `@st.cache_data` délégant ; `_medals_repo.py` délègue. 3 chemins DB indépendants → 1. Fallbacks JSON applicatifs supprimés de `medals.py`. JSON `static/medals/*.json` conservés (source pour `populate_medal_metadata.py`). 51 tests passent. Commit `88d5cf0`. |
| 2026-03-19 | **Migration `b5>>4`** — `scan_fire_events_b5` implémenté, `fire_seq%n_players` supprimé, `map_b2_to_player`/`group_events_by_pi`/`POV_PLAYER_INDEX` retirés, 25 nouveaux tests — 4968 tests passent. Relancer `--force-weapons --all` pour re-extraire. |
| 2026-03-19 | **Backfill enrichissement** JGtm + Madina97294 — 8 matchs du 18 mars rattrapés (performance_score, sessions, citations) |
| 2026-03-19 | **Fix 11 — Fan-out multi-joueurs** : `FanoutEnrichmentMixin` (`_engine_fanout.py`) + branchement dans `engine.py` après `_detach_shared_from_player_conn()`. Résout le manquement d'enrichissement local pour les joueurs qui ne sync pas eux-mêmes. |
| 2026-03-19 | **Fix 10 — Performance vs historique** : `performance_score` ajouté à `COLUMNS_COMMON` + JOIN `player_match_enrichment` dans `load_matches_as_polars` + `df_history` propagé dans `WinLossService` |
| 2026-03-19 | **Fix 9 — Radar escouade** : `radar_squad_ids` sauvegardé avant filtre UI ; DFs historiques séparés (`radar_me_df/f1/f2/f3`) passés à `render_trio_synergy_radar` |
| 2026-03-19 | **Fix 8 — Heatmap monochrome** : `compute_map_breakdown` lit `performance_score` depuis la colonne quand présente (fallback percentile supprimé pour les joueurs enrichis) |
| 2026-03-19 | **Fix 7 — Performance vue 1 coéquipier** : `enrich_with_performance_score` appelé pour `me_df` et `friend_df` dans `render_single_teammate_view` |
| 2026-03-19 | **Fix 6 — MediaFileStorageError icônes rang** : images rang converties en data URI base64 dans `career.py` (IDs Streamlit éphémères éliminés) |
| 2026-03-19 | **Fix 5 — Joueurs fantômes** : `_is_ghost_player` requiert la présence des clés stat + filtre appliqué uniquement dans `filter_encounter_xuids` (scoreboard non filtré — joueurs légitimes à 0 stats conservés) |
| 2026-03-19 | **Fix 4 — ratio=kda** : `ratio = pl.col("kda").alias("ratio")` dans `_finalize_polars_df` + `p.kda AS ratio` dans `_query_teammate_shared_stats` — source unique API, plus de recalcul |
| 2026-03-19 | **Fix 3 — Matrice d'impact** : `.unique(maintain_order=True)` dans `friends_impact_heatmap.py` |
| 2026-03-19 | **Fix 2 — Bots bid(33.0)** : `get_bot_name()` appelé dans `_build_encounter_rows` avant le fallback `xuid[:8]` |
| 2026-03-19 | **Fix 1 — ColumnNotFoundError map_name** : `mr.map_name` ajouté au SELECT de `load_friend_match_details` + `_FRIEND_DF_EMPTY_SCHEMA` mis à jour |
| 2026-03-19 | **Bonus — `resolve_weapon_display` fusion avant DB** : la fusion map est appliquée (étape 0) avant le lookup `weapon_labels`, évitant que M392 Bandit / Fuel Rod SPNKr contournent leur regroupement canonique |
| 2026-03-16 | Audit post-V6 : `weapon_kills` bit sync + logging, `v_gamertag_lookup` systématique, `shared_matches_v2.duckdb` production, LEGACY SyncScope supprimés, 17 nouveaux tests — 4799 tests passent |
| 2026-03-16 | Sprint refactor : splits fonctions/modules >80/500L, `_teammates_trio_helpers`, `_match_relations`, `_roster_loader` helpers, `render_trio_charts` DRY |
| 2026-03-15 | Phase 3 v6 : migration complète `duckdb_read_only` UI → repo — 7 fichiers migrés, 17 tests + 9 tests antagonistes, 4764 tests passent |
| 2026-03-15 | Phase 2 v6 : `career`, `career_lusr`, `explorer` migrés + `CareerMixin` créé |
| 2026-03-15 | Migration last_match : requêtes directes → DuckDBRepository (`load_player_match_enrichment`, `is_abandoned_match`) — 12 tests |
| 2026-03-15 | Fixes Phase 1 v6 : `player_provisioning.py` bare connect, `cache_filters.py` `_get_connection()` privé, `multiplayer.py` dead code — 6 tests |
| 2026-03-15 | Couche résolution gamertag→XUID : `lookup_xuid_for_gamertag()` dans `src/utils/xuid.py` + `GamertagResolverMixin` — 9 fichiers migrés, 11 tests |
| 2026-03-15 | **v5.8 Wave 5** : nettoyage i18n playlists/modes obsolètes → `metadata.duckdb` |
| 2026-03-15 | **v5.8 Wave 4** : suppression `highlight_events.gamertag` + helper `resolve_medal_name` |
| 2026-03-15 | **v5.8 Wave 3** : nettoyage wrappers XUID + dead code outcomes → `Outcome` enum |
| 2026-03-15 | **v5.8 Wave 2** : migration consommateurs directs (gamertags, KV pairs, assets) |
| 2026-03-15 | **v5.8 Wave 1** : vues SQL `v_gamertag_lookup`, `v_match_full`, `v_killer_victim_full` + `GamertagResolverMixin` |
| 2026-03-15 | **Fix weapon-parser** : corrélation globale — taux `fire_event` 15% → 95% |
| 2026-03-15 | **Navigation last_match** : boutons ◀/▶ entre matchs filtrés |
| 2026-03-13 | Couverture tests `migrations.py` (lacunes v5.5–v5.7) |
| 2026-03-13 | Conflit `shared_matches.duckdb` — sync depuis UI Streamlit |
| 2026-03-13 | **[UI] Heatmap performance par joueur × carte** — Page Teammates |
| 2026-03-13 | **[UI] Performance par carte vs historique** — vues escouade et joueur |
| 2026-03-08 | **Bug #0 : match invisible post-sync** — suppression `_filters_loaded_*` dans `_clear_app_caches()` |
| 2026-03-08 | **Perf UI** — vues matérialisées lazy, pagination SQL, projections fines, `@fragment_if_available` |
| 2026-03-28 | [v6.2] Badges Remontada / Débandade / Contre-Remontada — `DominanceFlag` 3-5, `comeback_analysis.py`, `comeback_backfill.py`, `--comeback-badges` CLI |
| 2026-03-28 | [v6.2] Unification vue coéquipier unique → vue escouade — `f2_xuid` optionnel, suppression `render_single_teammate_view` |
| 2026-03-28 | [v6.2] Graphe combiné Frags↑/Morts↓ — `plot_trio_kills_deaths()`, axe Y symétrique, `safe_chart_render()` |
