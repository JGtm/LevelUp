# Halo 5 — Registre des items DIFFÉRÉS / non faits (autoritatif)

Compilé 2026-06-26 depuis `.ai/**` (handoffs/plans/thought_log), `config/titles/halo_5/`
et `apps/go-api/internal/games/halo_5/`. Statuts : **planned** | **partial** | **blocked**
| **data-limited** | **degraded**. Contexte : « livré » (parité enrichissement) désignait
l'ingestion, PAS un zéro-reste. Plusieurs items « faits en code » ne sont que sur des
branches non mergées (cf. G9).

## (A) CSR / Skill
- **A1 — Tier Champion (DesignationId 6) non mappé** — `h5DesignationTiersEN` s'arrête à Onyx (0..5) → tier vide pour un Champion. `csr_mapper.go:32-33`. **data-limited** (jamais vu jusqu'à la sonde 2026-06-26 : un vrai 6/Rank 236 existe → désormais actionnable).
- **A2 — CSR par saison réelle (seasonId)** — Phase 1 = bucket lifetime fixe `h5-lifetime`. `title.toml:33-34`, `csr_mapper.go:57`. **planned** (queryabilité saisons H5 2026 non confirmée).
- **A3 — CSR pré/post par match côté capability** — donnée dans le carnage, `match.skill.snapshot`/`scoreboard.extra` = `not_exposed`. `capabilities.toml:24-25`. **partial** (extrait en backfill, pas via capability).
- **A4 — Leaderboard CSR mondial** — aucune source (endpoint officiel 404 partout). **blocked**.
- **A5 — Catalogue de noms de playlists classées** — `ranked_hoppers` peuplé, mais `LoadRankedPlaylists`/`playlists_catalog` non câblé. **partial**.

## (B) Armes
- **B1 — Registre taxonomy armes (BDD) P2→P4** — construit sur branche, non mergé/non branché lecteurs. **planned**.
- **B2 — UI taxonomy** — narration différée. **planned**.
- **B3 — Seed `weapon_id` (StockId) H5 effectif** — dépend de la collecte réelle des StockId. **partial**.
- **B4 — Précision par arme (carnage `WeaponStats[]` vide)** — **blocked via carnage** ; MAIS reconstructible via `WeaponDrop.ShotsFired/ShotsLanded` des events (cf. exploration). 
- **B5 — Suppression du décodeur kill-feed (weaponv3)** — cleanup d'origine jamais fait. **planned**.

## (C) Identité / Apparence
- **C1 — Champs appearance droppés** — DTO ne garde que ServiceTag+Emblem+Company ; armure/visière/couleurs/skins/stance/assassination/voiceover parsés-droppés. `dto.go:124-128`. **planned**.
- **C2 — Image badge CSR (carrière/home)** — libellés OK, image non câblée ; builder forcé HINF. `PLAN_H5_ASSETS.md`. **partial**.
- **C3 — Catalogue noms de rangs par titre** — `LoadRankCatalog` forcé HINF ; SR affiché « SR N ». **partial**.
- **C4 — Traductions FR médailles + surface sprites** — API officielle EN-only ; pas de surface d'affichage médailles. **data-limited/partial**.
- **C5 — Catalogue maps live (`maps_catalog`)** — pas d'adapter UGC ; seedé via metadata-fetch seulement. **planned**.
- **C-NEW — Bannière joueur** — voir FINDINGS_personnalisation §Bannière : pas de bannière native ; company banner retirée par vNext. **blocked (source)** → alternative à synthétiser.

## (D) Engagement / Coaching
- **D1 — Engagement score `not_exposed`** — events présents mais (a) coefficients à recalibrer ET (b) **kill/death absents de `highlight_events` H5** (vont dans killer_victim_pairs/weapon_kills). `capabilities.toml:28`. **degraded/planned**. Voir FINDINGS_engagement_enrichment.
- **D2 — Damage-model P80 résiduel (radar session-compare)** — encore const Infinite ; DR = N/A (pas de damage_taken). **partial/data-limited**.
- **D3 — `assist_frag_weight` par titre** — dur à 3 (Infinite) ; valeur H5 à confirmer. **planned**.
- **D4 — Progression V2 / Prestige / Coach capabilities non déclarées H5** — wiring livré sur `feat/h5-enrichment-parity` (non mergé) ; Prestige mono-titre. **partial/planned**.
- **D5 — Streaks/records vides H5** — fenêtre 120j ; matchs H5 anciens → 0. **data-limited**.
- **D6 — Radar awards (player_profile) par titre** — `awards.toml` chargé HINF-only. **planned (polish)**.

## (E) UI / Front
- **E1 — Timeseries `not_exposed`** (HI et H5). **planned/degraded**.
- **E2 — Moteur citations vs commendations natives** — natif câblé ; moteur dérivé `not_exposed`. **partial**.
- **E3 — Page Escouade H5** — agrégation legacy HINF ; roster H5 → fallback/`not_exposed`. **planned**.
- **E4 — Libellés `FeatureUnavailable` non i18n** ; **E5 — `data_gaps` → badge front** ; **E6 — fallback placement `?? 10`** ; **E7 — KDA nullable `0` vs `—`** ; **E8 — Outcome TIE non géré** ; **E9 — media UGC/forge exclus**. **planned (mineurs/partials)**.

## (F) Data-quality
- **F1 — `delta_mmr`/`performance_score` vides H5** (pas de source). **data-limited**.
- **F2 — `first_joined_tz` global, pas par titre** (risque reindex média). **planned (dette)**.
- **F3 — Lignes SR carrière contaminées (>152)** — backfill réparateur. **partial**.
- **F4 — Reclassement `is_ranked`/`is_pve` des matchs déjà importés** — backfill. **planned**.
- **F5 — Distinction null réel vs Tie au niveau KDA** — Phase 2. **data-limited**.
- **F6 — Backfill kill-mechanics sur matchs existants** — colonnes peuplées seulement au re-sync. **planned (op)**.

## (G) Infra / Activation / Multi-titre
- **G1 — Warzone/Firefight (PvE) exclus** (modèle ≠ HINF ; rosters 24 ; events non re-sondés). **planned/blocked**.
- **G2 — Battlepass/Challenges (n/a)** — REQ packs = feature H5 Phase 2+. **degraded (par design)**.
- **G3 — Watcher live multi-titre (Pass B.7)** — clé gamertag-only, collisions 2 titres. **planned (risqué)**.
- **G4 — SharedProvider 3e titre via SyncEngine** — à fixer en activation 1b. **planned**.
- **G5 — Gating provider par-titre au boot + dedup XUID PeopleHub**. **planned**.
- **G6 — Migration `GenericSemanticAdapter` (dette DRY)**. **planned**.
- **G7 — Durabilité events (capture-on-fetch append-only) Phase 4a** — events H5 viennent UNIQUEMENT de cryptum (fragile, irremplaçable) → persister à la lecture. **planned**.
- **G8 — Deploy metadata sur VPS** (`cmd/h5-metadata-fetch` + clé). **planned (op)**.
- **G9 — Land branches `integration`/`feat` → `main`** (auto-deploy ; plusieurs items « faits » n'arrivent qu'après merge). **partial**.

## Déjà RÉSOLU (ne pas re-signaler — docs datés périmés)
match.history/LoadMatchSummaries · commendations.native · match events/timeline/kill-feed/spatial · Career SR (X/152, « SR N ») · G8-front groupes LUSR (`h5_arena`) · refactor durée ISO8601 · classification ranked (hoppers peuplés) · damage-model Phases 0-2 + front · appearance/emblem/service-tag fetch+persist · NormalizeMedalCategory.

**Bloquant externe transverse** : la clé d'abonnement API Metadata officielle (`developer.haloapi.com`) — requise pour FR + référentiels médailles/maps/armes + images badges CSR/SR. Non automatisable par agent.
