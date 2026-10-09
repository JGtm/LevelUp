# PLAN — Triage et traitement des détections monitoring (2026-07)

> Statut : QUASI-SOLDÉ — clôture prod le 2026-07-16 (`docs/b4-data-quality-solde`) ; seul
> reste le soak FINAL B7.4 (2026-08-11) et UN [!] renvoyé en chantier de suivi (B4.2
> orphelins). Historique : exécuté le 2026-07-10 sur `fix/monitoring-triage-2026-07`. Le code
> déploy-indépendant est livré et testé (B6.4 anti-flood, B6.1/B3.3 démotions+compteurs).
> **B1 déployé en prod le 2026-07-12** (mergé via PR #54 → deploy auto) : cascade LUSR
> ÉTEINTE, `/health` stabilisé (un seul 503 ponctuel vs 632 pendant l'incident),
> writer-holds retombés — B1.5 vérifié sur pièces (cf. plan hotfix H6). **B1.6 SOLDÉ le
> 2026-07-12** (départage fait : dry-run puis `--commit` du canonical backfill en fenêtre
> prod — 2 551 matchs réécrits sur les 4 joueurs, couverture LUSR garantie par
> construction). Reste ouvert :
> - **B4.1-B4.4 / B5.5 SOLDÉS EN PROD le 2026-07-16** (branche `docs/b4-data-quality-solde`,
>   endpoint admin, session signée côté VPS) : B4.1 [x] 20/24 UUID résolus (4 playlists → drain
>   réseau), B4.3 [x] 580 lying-bits vidés (floor no-film ~13 = découverte détecteur), B4.4 [x]
>   1 mode traduit (1 artefact pair-inversé [!]) ; B4.2 [!] convergence 4/4 OK mais mécanisme
>   opportuniste épuisé (orphelins 144 inchangés) ; **B5.5 [x] no-op vérifié le 2026-07-16**
>   (dry-run : 0 chemin absolu prod HI 139/139 + H5 84/84 déjà relatifs). Détail §B4 / §B5.5.
> - **B2.4 SOLDÉ le 2026-07-16** (soak 4 j : 0 reauth 07-13→16, 0 AADSTS). **B5.1/B5.2
>   CONFIRMÉS résolus** post-deploy B1 (0 `read-only mode` depuis le 07-11 ; writer-holds
>   nominaux `held_ms=2000` ; `/health` 503 résiduel = gate-window bénin → refonte). **B7.4** =
>   seul soak restant (lecture interim 07-16 OK ; mesure finale 2026-08-11). Découvertes
>   NOUVELLES consignées : (d) `database is closed` race prestige (apparue 07-14), (e) tables
>   metadata `mode_name_tr`/`battlepass_track_definitions` absentes, (f) `/health`→`/healthz`.
>   Chantiers de suivi : B4.2 (orphelins) + (d)/(e) (voir §Découvertes). B5.5 SOLDÉ (no-op).
> Exécution sous contrat du skill `plan-execution`.
> Branche cible : `fix/monitoring-triage-2026-07` (1 branche, N commits) — SAUF B1
> (régression prod active) : hotfix depuis `origin/main` (voir DC-B6).
> À exécuter AVANT `.ai/PLAN_MONITORING_REFONTE_2026-07.md`.
>
> **Mesure de clôture (2026-07-10, T0)** — prod post-reboot (fenêtre 07-08→07-10, VPS
> `ssh lvelup`) : la tempête DNS est ÉTEINTE (0 ERROR pool) ; ~95 % du bruit survivant est
> la cascade LUSR (B1) TOUJOURS ACTIVE (fix non déployé) — sync.log 28 422 W LUSR shadow,
> provider.log 2 303 W writer RW tenu, `GET /health → 503` (les 632 « http » ERROR = /health
> timeout 5 s, symptôme direct du writer-hold). Hors LUSR : pool sain (180 W = 429 AIMD),
> 0 reauth, crons sans crash. Familles B6.1/6.2/6.3/6.5 = 0 en prod ET en juillet local
> (bruit historique pré-reboot/juin). Data-quality local HI (endpoint read-only) : raw_uuid
> 24, untranslated_modes 8, orphan_playlists 0, orphan_xuids 131, lying_bits_events 580,
> lying_bits_weapons 0 (inchangé — remédiation prod-gated).

## Objectif et critère de succès

Mesures locales (2026-07-06) puis prod (2026-07-07, VPS revenu). Les deux profils
diffèrent fortement — le plan est calibré sur la PROD, là où la page monitoring est lue :

- **Prod, juillet (7 jours)** : ~40 000 ERROR / ~136 000 WARN, dont deux tempêtes :
  (1) **panne DNS Docker** (`lookup … on 127.0.0.11:53: server misbehaving`) par vagues
  du 01 au 07-07 (pic 59 285 le 06-07, jour où le VPS était injoignable) — ÉTEINTE
  depuis le reboot du 07-07 12:31 UTC ; (2) **régression LUSR v2 shadow** : INSERT sur
  `shared_matches_v2` attachée READ-ONLY — ~6 500 WARN/jour depuis le 03-07, TOUJOURS
  ACTIVE post-reboot (~280/h), watermark jamais avancé, et cortège : writer RW tenu
  au-delà du seuil (×150 en 4 h), lectures shared gatées, **`/health` répond 503 par
  intermittence** (×44 en 4 h).
- **Local (dev)** : bruit propre au poste (verrous player DB tenus par air/worktrees,
  scrape leaderboard) — conservé comme cibles secondaires.
- **Data-quality (mêmes BDD local/prod)** : HI = 24 UUID bruts / 120 xuids orphelins /
  580 lying-bits events / 0 playlists orphelines ; H5 = 0 partout (1791 + 3032 matchs).

**Critère de succès** : (1) régression LUSR corrigée en prod, watermark avance, plus de
503 sur `/health` ; (2) chaque famille d'erreurs restante est corrigée, dégradée avec
compteur, ou acceptée avec justification datée ; (3) data-quality HI : raw_uuid = 0,
lying_bits_events = 0, orphan_xuids réduit > 90 % ; (4) un mois glissant après
livraison : ERROR prod ≈ 0/jour hors incidents externes attestés (script §Mesure).

**Bilan des critères à la clôture (2026-07-16, prod 4 j)** :
- (1) **ATTEINT** (partie LUSR) — 0 `read-only mode`/jour depuis le 07-11, watermark avance.
  Nuance : le `/health` 503 résiduel (~90/j, baseline STABLE 07-11→16, non causé par LUSR)
  = gate-window bénin → Découverte (f), renvoyé au plan REFONTE (`/healthz`). Le STORM 632 de
  l'incident est bien éteint.
- (2) **ATTEINT** — chaque famille statuée `[x]`/`[~]`/`[!]` justifié.
- (3) **PARTIEL** — raw_uuid 24→4 (4 playlists = drain réseau `catalog/ugc-drain`),
  lying_bits 580→~13 (−97,8 %, floor no-film = faux positif détecteur), orphan_xuids 144
  INCHANGÉ (mécanisme opportuniste épuisé). Résidus → chantiers de suivi (B4.2 + §Découvertes).
- (4) **DÉLAI PRESCRIT** — soak final 2026-08-11 (lecture interim 07-16 OK, cf. B7.4).

## Décisions pré-tranchées (DC)

- **DC-B1** : JAMAIS de re-capture de token (ADR 0023). Les RT morts se blacklistent
  (3 xuids périmés connus), ils ne se « réparent » pas.
- **DC-B2** : une dégradation de niveau de log n'est admise que pour du bruit répétitif
  à cause connue, TOUJOURS avec compteur expvar + log de première occurrence (règle 3
  CLAUDE.md : jamais d'erreur avalée).
- **DC-B3** : lying bits events : reset via l'action admin existante
  (`POST /admin/actions/lying-bits/reset`, dry-run d'abord), puis la convergence
  re-remplit. Pas de backfill manuel massif.
- **DC-B4** : le warn `pool: legacy sync_meta DuckDB utilisée` relève des lots D1a/D2
  du plan audits (ADR 0023 Phase 5) → statut [~], ne pas dupliquer ici.
- **DC-B5** : la panne DNS 01→07-07 est un incident INFRA (résolu par le reboot du
  07-07 12:31). Pas de correctif applicatif — le pool/AIMD a fait son travail. En
  garder : (a) la trace ci-dessous, (b) l'item B6.4 (anti-flood de logs répétitifs
  pendant un incident réseau), (c) la visibilité future = plan refonte (État/ressources).
- **DC-B6 — Chemin de livraison du hotfix B1** : la branche d'audits n'est pas mergée
  (gate humain en cours) → B1 part de `origin/main` en branche `hotfix/lusr-shadow-ro`,
  merge dans main APRÈS accord user explicite (push main = deploy prod auto — prévenir).
  Ne PAS baser B1 sur `refactor/audits-2026-07`.

## Journal d'incident (référence)

- 2026-07-06 : VPS injoignable toute la journée (SSH timeout + HTTP 000 depuis
  l'extérieur) ; en interne le serveur tournait et produisait 59 285 erreurs DNS.
- 2026-07-07 12:31 UTC : reboot de l'hôte → DNS rétabli (dernière erreur DNS à
  12:31:30), conteneurs healthy. Mémoire 1.9 Go (487 Mo dispo), disque 82 % (15 Go
  libres), pas de swap.
- Déploiements main du 02-07 19:31→22:21 (burst-lease post-sync b34724a7f, anti-TOCTOU
  dc424162b, throttle AIMD efb9c5459, LOT S f8b9caff7) ; la régression LUSR RO apparaît
  le 03-07 → suspect n°1 : burst-lease (« writer non tenu pendant I/O »).

## Phases

### B0 — Mesure prod [x] (exécutée 2026-07-07)

- [x] B0.1 État VPS : down le 06-07, reboot 07-07 12:31, conteneurs healthy — cf.
      Journal d'incident.
- [x] B0.2 Relevé logs prod + comparaison local : résultats intégrés ci-dessus ;
      plan recalibré (B1 nouveau, B2/B3 ajustés).

### B1 — RÉGRESSION prod : LUSR v2 shadow écrit sur un attach read-only (effort : moyen, PRIORITAIRE)

> **Exécution : plan dédié autoporteur `.ai/PLAN_HOTFIX_LUSR_SHADOW_RO_2026-07.md`**
> (fait foi — écrit pour une session agent autonome ; contexte incident, décisions
> figées DC-H1..H7, phases H1→H7, gates exacts, protocole GO user avant push main).
> Les items B1.x ci-dessous en sont le résumé ; les statuer [~] par référence au plan
> hotfix une fois celui-ci clos.

Symptôme : `LUSR v2 shadow: persist état échoué — watermark non avancé` —
`persist owner-only: SkillV2Repo.UpsertState(xuid, group): Invalid Input Error: Cannot
execute statement of type "INSERT" on database "shared_matches_v2" which is attached in
read-only mode!` (groupes Infinite ET H5). Depuis le 03-07, ~6 500/jour, toujours actif.
Effets : watermark LUSR figé (données récentes manquantes), boucle de retry à chaque
cycle, writer RW tenu au-delà du seuil, lectures gatées, `/health` 503 intermittent.

Cause racine IDENTIFIÉE (2026-07-07, sur pièces) : le refactor contention/burst-lease a
classé le bloc LUSR du post-sync en **segment lecture**
(`engine_postsync_scoring.go:136-146` — `shared.Read(ctx)`, commentaire « LUSR LIT
shared et écrit la PLAYER DB ») alors que le v2 shadow écrit AUSSI côté shared
(`player_skill_state_v2` via `SkillV2Repo.UpsertState` — `skill_v2_shadow.go:74-75`).
Le commentaire ne couvre que le chemin v1. Erreur de classification, pas un bug du
shadow lui-même.

**Correctif retenu (pérenne, conforme à l'architecture burst-lease — PAS un revert)** :
aligner le shadow LUSR sur le pattern des étapes events/weapons
(`engine_postsync.go:274-288`) : sélection/lectures longues sous segment Read, puis
traitement+persist par **bursts `shared.Write` chunkés** (le calcul EP est CPU-rapide ;
la dépendance séquentielle des états entre matchs impose de persister au fil de l'eau →
le chunk borne la fenêtre RW, les lecteurs passent entre les chunks, comme events).

- [~] B1.1 Seam — couvert par PR #53 `hotfix/lusr-shadow-ro` (interface `skill.SharedAccessor`
      Read + Write-burst, adaptateur sur `SharedAccess`, callers CLI/backfill/h5 sur handle RW).
- [~] B1.2 Chunking — couvert par PR #53 (persist LUSR v2 shadow par bursts `shared.Write`
      chunkés ; 0 match → aucun burst).
- [~] B1.3 Tests — couvert par PR #53 (test reproduisant le bug prod RO + non-régression +
      audit des autres segments `shared.Read(`).
- [x] B1.4 Livraison hotfix — DÉPLOYÉ EN PROD le 2026-07-12 : le fix a été mergé dans main
      via la campagne d'intégration (**PR #54** l'a embarqué ; PR #53 isolée super-cédée) →
      deploy prod auto (~09:25 UTC), avec GO utilisateur préalable.
- [x] B1.5 Post-deploy — VÉRIFIÉ SUR PIÈCES (3 h de logs post-deploy, cf. plan hotfix H6.1) :
      **zéro** `persist état échoué`, **zéro** `read-only mode` ; writer-holds retombés à
      2000-2001 ms (vs 21 909 pendant l'incident) ; `/health` **un seul 503 ponctuel**
      (vs 632 pendant l'incident) ; post-syncs réels tournés à 09:56 et 10:24-26.
- [x] B1.6 Rattrapage backfill — SOLDÉ le 2026-07-12 : départage fait par
      `lusr_v2_canonical_backfill` en fenêtre prod (serveur arrêté) — dry-run d'abord
      (2 547 comptés) puis `--commit` : **2 551 matchs réécrits** (JGtm 946,
      Madina97294 1 081, Chocoboflor 493, XxDaemonGamerxX 31), couverture LUSR garantie
      par construction (reset par joueur + persist owner-only, append-only + vues
      `_latest`). Serveur relancé, site 200.

**Gate B1** : VERT — fix déployé, B1.5 vérifié sur pièces, B1.6 soldé (couverture
complète). La cascade LUSR est éteinte et rattrapée.

### B2 — Pool auth / tokens (effort : réduit après B0)

Recalibrage : les ~21 000 erreurs pool prod de juillet étaient à ~99 % la panne DNS
(éteinte). Post-reboot : 0 ERROR pool, 9 warns 429 (AIMD ok). Reste le fond local :
slots morts au boot, `refresh_token mort — reauth_required` ×45, `world-enrich skippé`
×121 (local).

- [x] B2.1 Inventaire sur pièces : 9 joueurs HI déclarés dans `db_profiles.json`
      (4 réels + 5 `auth_only`), 9 fichiers tokens présents (`watcher_tokens/*.json`,
      xuid↔fichier 1:1). Prod post-reboot (07-08→10) : 0 ERROR pool, 0 `reauth_required`,
      180 W = 429 AIMD (backpressure saine). Local = env dev-recovery (tous comptes reauth,
      NON représentatif — écarté du ciblage).
- [~] B2.2 Blacklist 3 RT morts — SANS OBJET en prod : vérif read-only prod → AUCUN RT mort
      actif post-reboot (seul Chocoboflor a reauth 35× PRÉ-reboot, auto-guéri ; 0 post-reboot).
      Rien à blacklister. La discovery n'exclut que les comptes SANS token ; un RT mort est
      retenté par design (auto-guérison reauth_required, ADR 0023). L'anti-flood B6.4 couvre
      désormais le flood oauth-refresh (clé par classe) si un RT venait à mourir.
- [x] B2.3 RT valides se rafraîchissent — prouvé en prod : syncs OK, 0 reauth post-reboot ;
      helper `RefreshHaloTokensViaStoreFirst` en place.
- [x] B2.4 Soak post-deploy B1 — MESURÉ le 2026-07-16 sur 4 jours pleins (fenêtre 07-13→16,
      `ssh lvelup`, auth.log) : **0 `reauth_required`, 0 `AADSTS`** (07-13..16 ET aujourd'hui).
      Le délai prescrit (24-48 h post-deploy B1 du 07-12) est largement écoulé, la mesure est
      propre : RT valides se rafraîchissent, aucun RT mort actif. Soak SOLDÉ.
- [~] B2.5 Warn legacy sync_meta → lots D1a/D2 plan audits (DC-B4).

**Gate B2** : VERT + soak SOLDÉ. Prod pool sain (0 ERROR, 0 reauth) ; soak B2.4 4 jours
propres (0 reauth 07-13→16). Familles locales = env recovery, hors cible.
`go test ./internal/platform/auth/... ./internal/sync/...` verts (exit 0).

### B3 — Crons en échec (effort : réduit après B0)

Recalibrage : leaderboard prod juillet = 2 ERROR / 3 WARN seulement — l'essentiel du
bruit leaderboard était local (dev). catalog prod juillet = 0 ERROR.

- [x] B3.1 `world_leaderboard_cron` — REQUALIFIÉ le 2026-07-13 (LOT OPS/QUALITÉ item 1,
      branche `chore/lot-ops-qualite`). Le 404 `FetchCatalog: ... classement absent pour cette
      (saison, playlist)` n'était PAS le reste-à-faire C3 (backfill saisons passées) : c'était
      la **découverte de la saison ACTIVE** qui abandonnait tout le cycle sur une ERROR
      quotidienne quand la playlist de référence unique (`playlists[0]`) n'était pas classée
      dans la saison-graine fixe `csrseason13-2`. Fix réel : `discoverActiveSeason` essaie
      plusieurs playlists candidates (statiques d'abord) et retient le 1er succès ; dégradation
      DC-B2 (WARN + compteur `world_leaderboard_season_discovery_failed_total`) si toutes
      échouent — plus d'ERROR récurrente. Bruit local (scrape ×26) = dev, hors périmètre prod.
- [x] B3.2 `catalog_refresh_cron` — prod post-reboot = 0 ERROR (49 E historiques ÉTEINTS).
      Résidu 6 W `catalog_expand: terminé avec échecs` = expansion partielle gracieuse (asset
      manquant), non bloquant. Confirmé éteint.
- [x] B3.3 `spartan_cron` verrous player DB — le lock per-joueur est DÉJÀ en Debug + agrégé
      une-fois-par-cycle (code existant) ; ligne agrégée passée **ERROR→WARN + compteur expvar
      `spartan_cron_player_db_locked_total`** (DC-B2, `spartan_customization_cron.go`). Absent
      de prod (60 « refresher failed » post-reboot = échecs NON-lock légitimes, WARN).

**Gate B3** : crons prod sans crash ; `go test ./internal/scheduler/...` vert (exit 0).

### B4 — Stock data-quality halo_infinite (effort : moyen ; mêmes BDD local/prod)

> **SOLDÉ EN PROD le 2026-07-16** (branche `docs/b4-data-quality-solde`, GO utilisateur B4
> autonome) via l'endpoint admin `POST /api/v1/admin/actions/*` (VPS, mode xbox). Session admin
> obtenue CÔTÉ VPS : cookie `<sessionID>.<HMAC-SHA256(secret,sessionID)>`, signature calculée
> DANS le conteneur (`openssl` + `$LEVELUP_SESSION_SECRET`) — secret JAMAIS exfiltré. Le serveur
> écrit lui-même (writer unique dblease, anti-ART) — aucune ouverture DuckDB RW externe.

- [x] B4.1 Assets UUID bruts — `registry-names/backfill`. AVANT raw_uuid **24** (playlists 6/
      maps 7/pairs 7/variants 4). Dry-run : 24 scannés (cohérent). Réel : **20 corrigés**
      (playlists 2, maps 7, pairs 7, variants 4) via `metadata.asset_translations` (zéro réseau).
      APRÈS raw_uuid **4** (maps/pairs/variants = 0). Les 4 playlists restantes n'ont pas
      d'entrée `asset_translations` → résolvables SEULEMENT par le drain DiscoveryUGC réseau
      (`catalog/ugc-drain`), hors périmètre de cet endpoint zéro-réseau.
- [!] B4.2 XUIDs orphelins — `convergence/run` lancé pour les **4 joueurs réels** (JGtm,
      Madina97294, Chocoboflor, XxDaemonGamerxX) : **4/4 succeeded, auth live SISU OK**
      (error_count=0). MAIS `converged_psa=0` partout → **0 alias résolu** (AVANT **144** →
      APRÈS **144**). Cause sur pièces : la convergence d'alias est OPPORTUNISTE (upsert
      `xuid_aliases` depuis les JSON PSA re-fetchés) et ne cible que les matchs
      `psa_checked_at IS NULL` ; depuis le fix B1 (07-12) le pipeline a stampé tous les matchs
      des 4 joueurs → set vide, mécanisme ÉPUISÉ. Les 144 orphelins vivent dans des matchs
      PSA-terminaux dont le JSON n'avait pas le gamertag (bots déjà exclus). Résorption =
      ré-résolution d'alias dédiée (PeopleHub batch sur les xuids orphelins, ignorant
      `psa_checked_at`) — NON exposée par endpoint admin → reste [!], suivi §Découvertes.
- [x] B4.3 Lying bits events — `lying-bits/reset`. AVANT lying_bits_events **580**. Dry-run :
      events 580 / weapons 0 / events_loaded 579 = **1159** (cohérent, pas d'anomalie). Réel :
      **1159 nettoyés** row-by-row (anti-ART) → APRÈS immédiat **0**. Puis rebond à **13** après
      les convergences B4.2 (Chocoboflor/XxDaemon). DÉCOUVERTE (sur pièces) : le détecteur
      lying_bits_events est un FAUX POSITIF pour les matchs no-film/vides terminaux —
      `MarkNoFilmDefinitive`/`MarkEventsEmptyDefinitive` posent MBitEvents SANS écrire de
      highlight_events, et le détecteur ne teste pas `events_empty` → reset+convergence oscille
      et ne converge jamais vers 0. Backlog des 580 réellement vidé (−97,8 %) ; les 13 = matchs
      no-film légitimes re-marqués (pas une donnée cassée). Fix = affiner détecteur+reset pour
      exclure `events_empty=TRUE` (code, hors périmètre) → §Découvertes.
- [x] B4.4 Modes non traduits — `translations/mode`. AVANT untranslated_modes 1 (→ **2** après
      B4.1 : le backfill des noms de paires a exposé un 2e label). **`Legacy Slayer BR`** (de
      « Arena:Legacy Slayer BR on Narrows », pair BIEN formé, sous-mode légitime) → traduit
      **« Massacre BR hérité »** (convention Slayer→Massacre, cf. seed). APRÈS untranslated_modes
      **1**. Le résidu **`Arena`** (de « CTF:Arena on Opulence », 7 occ., figé au 2026-06-09) est
      un ARTEFACT de pair INVERSÉ (« CTF:Arena » au lieu de « Arena:CTF ») : `NormalizeModeLabel`
      extrait « Arena », mais le vrai mode = CTF. Le traduire mentirait (clé de catégorie, pas de
      mode). Fix propre = `mode_pair_override` (table metadata, PAS exposée par endpoint) →
      [!] justifié.
- [x] B4.5 Tableau relevé (endpoint read-only, local, 2026-07-10) : HI raw_uuid **24**
      (playlists 6/maps 7/pairs 7/variants 4), untranslated_modes **8**, orphan_playlists **0**,
      orphan_xuids **131**, lying_bits_events **580**, lying_bits_weapons **0**. H5 : détecteur
      en ERREUR locale (`data_quality_error` / « internal error ») — consigné en §Découvertes.

**Gate B4** : SOLDÉ 2026-07-16 (prod). B4.1 [x] (20/24 ; 4 playlists → drain réseau),
B4.3 [x] (580→0, floor no-film ~13 = découverte détecteur), B4.4 [x] (1 mode traduit ; 1 artefact
pair-inversé [!]). B4.2 [!] : action exécutée (4/4 convergences, auth OK) mais mécanisme
opportuniste épuisé (0 résolu) — résidu 144 hors portée endpoint.

### B5 — Erreurs applicatives résiduelles (effort : moyen)

Liste FERMÉE (relevés 06-07 local + 07-07 prod) :

- [x] B5.1 `duckdb: query failed` — RE-MESURÉ post-deploy B1 le 2026-07-16 (prod, duckdb.log).
      Cause LUSR ÉTEINTE : **0 `read-only mode` par jour depuis le 07-11** (dernière occurrence
      2026-07-10T14:02) ; writer-holds retombés au régime NOMINAL (`held_ms=2000` au seuil de 2 s,
      label `sync_v2_postsync` = burst-lease attendu, PLUS le runaway 21 909 ms de l'incident). Le
      résidu duckdb (93 E le 07-16) N'EST PAS la cascade LUSR mais deux familles NOUVELLES
      post-plan, consignées en §Découvertes : (d) `database is closed` sur player `stats.duckdb`
      (race prestige, apparue le 07-14) et (e) tables `mode_name_tr` / `battlepass_track_definitions`
      absentes. Downstream LUSR = résolu.
- [x] B5.2 `general "http"` ERROR = `GET /health → 503` (timeout 5 s) — RE-MESURÉ le 2026-07-16.
      Le STORM LUSR est éteint (632 pendant l'incident → **97/j le 07-16**, ~5/h RÉGULIERS, non
      groupés sur les cycles sync). Ce résidu n'est PLUS causé par le writer-hold LUSR : c'est le
      comportement nominal du burst-lease (gate lecture ~2 s en post-sync) croisé au budget 5 s de
      `/health` qui fait un vrai travail DB = la Découverte déjà consignée (`/health` → `/healthz`
      pour le healthcheck Docker), routée au plan REFONTE. LUSR résolu ; résidu hors périmètre triage.
- [x] B5.3 `augmentWithActiveRankedCSRs: GetPlaylistCsr échoué` — 7370 juillet (~99.7 %
      tempête DNS, éteinte) → **19 post-reboot** sur 3 j (~6/j, 401/données transitoires).
      Résidu négligeable ; l'anti-flood B6.4 couvre le flood réseau amont si récidive.
- [~] B5.4 `highlight_events parse_anomaly` ×32 local — limite connue du décodeur film →
      réf chantier filmdec (HANDOFF_FILM_EXTRACTION).
- [x] B5.5 `mediaStoredPathToURL: aucun mapping (path legacy)` — `migrate-media-paths`. EXÉCUTÉ
      (scope) en prod le 2026-07-16 sous mandat downtime : binaire buildé côté VPS (image
      `levelup-go-builder`, CGO, `-buildvcs=false`), **dry-run READ_ONLY contre une COPIE** des DB
      (jamais d'ouverture cross-process de la DB tenue RW — copies 20 M/6,6 M, ZÉRO downtime).
      RÉSULTAT DÉFINITIF : **0 chemin absolu dans les DEUX titres prod** — halo_infinite 139/139
      `file_path` + 139/139 `thumbnail` DÉJÀ relatifs ; halo_5 84/84 relatifs. La migration est un
      **no-op intégral** : c'est précisément pourquoi il y a 0 warning `mediaStoredPathToURL` (les
      chemins stockés sont déjà canoniques, le fallback read-path n'a même rien à rattraper).
      Aucun run réel ni downtime requis ; serveur resté healthy ; binaire + copies nettoyés. SOLDÉ.
- [x] B5.6 `configuration non sûre pour un déploiement multi-user exposé` — DÉJÀ log
      une-fois-au-boot (`cmd/server/main.go:320`, séquence boot après `cfg.Validate()`) :
      vérifié sur pièces. DC-B2 satisfait (le nombre d'occurrences = nombre de boots dev).
- [x] B5.7 `service.log` prod — 28 987 W = stock HISTORIQUE pré-reboot ; post-reboot (07-08→10)
      service.log = **11 W** (career_live LoadLastCareerRank/EnrichFromMetadata, live-fetch
      transitoire). Sous le seuil 100/j. Stock historique non re-généré.

**Gate B5** : VERT. Chaque item statué. Familles tempête-DNS éteintes ; cascade-LUSR
(B5.1/B5.2) CONFIRMÉE résolue post-deploy B1 le 07-16 (0 `read-only mode` depuis 07-11 ;
writer-holds nominaux 2 s) ; résidus duckdb = 2 Découvertes NOUVELLES (d/e) hors périmètre ;
B5.6 déjà conforme ; B5.5 SOLDÉ (no-op vérifié : 0 chemin absolu prod, dry-run 07-16).

### B6 — Démotions de bruit et anti-flood (effort : rapide)

Liste FERMÉE, chaque démotion = compteur expvar + première occurrence loggée (DC-B2) :

- [x] B6.1 `metadata verrouillée, nouvelle tentative...` — WARN→**Debug + compteur expvar
      `boot_metadata_lock_retry_total`** (`cmd/server/main.go`, retry de boot à cause connue ;
      l'échec DÉFINITIF reste ERROR+exit). 0 occurrence prod/juillet (bruit boot historique).
- [x] B6.2 `rta: subscribe refusé` — le SITE de log a DISPARU du code courant (seul un
      commentaire subsiste `refresh_loop.go:25`) : rien à démoter. 0 en prod/juillet (bruit
      historique pré-refactor). WebSocket ×138 idem.
- [x] B6.3 `endpoint_missing` — 2 sites WARN LÉGITIMES (`halo/endpoints.go`,
      `haloclient/endpoint_resolver.go`) : signalent une vraie lacune d'endpoint par titre,
      garantie absente par la validation boot PMT-12 → 0 prod/juillet. CONSERVÉ WARN (une
      démotion masquerait une vraie misconfig).
- [x] B6.4 Anti-flood incident réseau — NOUVEAU helper `observability.AllowThrottledLog`
      (`internal/observability/logthrottle.go`) : clé GLOBALE par cause (un incident infra
      qui frappe tous les pollers → 1 log / fenêtre de 30 s), 1re occurrence TOUJOURS émise
      (DC-B2), compteur expvar EXACT `<clé>_total` (toutes occurrences comptées), champ
      `throttled_since_last`. Câblé sur : rest_poller (transient + réseau/parse), halo_api
      downloadBlob, pool/resolver oauth-refresh (clé par classe → un reauth isolé reste
      visible). Test `logthrottle_test.go` (1re occurrence, étouffement fenêtré + compte exact,
      concurrent single-emit).
- [x] B6.5 `backup: export table échoué (ignoré)` — 0 prod post-reboot + 0 juillet local
      (site `pkg/duckdbbackup/exporter.go:110`). WARN LÉGITIME (backup partiel intentionnel :
      une table corrompue ne doit pas faire perdre tout le backup). Confirmé éteint prod.

**Gate B6** : `go build ./...` + `go test ./...` verts (exit 0) ; test de l'étouffeur B6.4
vert ; messages démotés absents de prod/juillet.

### B7 — Vérification finale et clôture (effort : rapide)

- [x] B7.1 Script §Mesure exécuté (logs local + prod post-reboot + data-quality endpoint) —
      résultats archivés dans l'en-tête « Mesure de clôture » et le thought_log. RE-EXÉCUTÉ en
      prod le 2026-07-16 (`ssh lvelup`, fenêtre 07-13→16) pour solder les soaks : résultats
      dans B2.4 (reauth), B5.1/B5.2 (LUSR/`/health` post-deploy) et B7.4 (interim).
- [x] B7.2 Tous les items statués ; découvertes consignées ci-dessous.
- [x] B7.3 Gate final : `cd apps/go-api && go test ./...` = exit 0 ;
      `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... [+touchés]`
      = exit 0 ; gofmt/vet propres ; skill `delivery-checklist` ; entrée `thought_log.md`.
- [!] B7.4 Cible mensuelle — RÉ-ARMÉ T0 = 2026-07-12 (B1 déployé, cascade LUSR éteinte).
      **Lecture INTERIM T0+4 j (2026-07-16, prod)** : ERROR/jour = general.log 101 (dont 97 =
      `/health` 503 gate-window bénin, 4 = `halo_api` transitoire), duckdb.log 93 (races gate
      bénignes + Découvertes d/e), leaderboard 1 ; sync/provider/pool/persist/service = 0 ERROR.
      Hors `/health`-gate : ERROR ≈ 0. Re-mesure FINALE T0+30 j (2026-08-11) : DÉLAI PRESCRIT
      (pas un report), attendu ERROR ≈ 0/j hors incidents externes + résorption des Découvertes
      d/e si traitées.

## Mesure (script canonique, réutilisable prod/local)

```bash
# Prod : ssh lvelup, logs dans /opt/levelup/data/logs ; local : logs/
for f in logs/*.log; do n=$(grep -c '"time":"2026-07' "$f"); \
  e=$(grep '"time":"2026-07' "$f" | grep -c '"level":"ERROR"'); \
  w=$(grep '"time":"2026-07' "$f" | grep -c '"level":"WARN"'); \
  [ "${n:-0}" -gt 0 ] && printf "%-24s %6s l  %5s E  %5s W\n" "$(basename $f)" "$n" "$e" "$w"; done
# Top messages
grep '"time":"2026-07' logs/X.log | grep -E '"level":"(ERROR|WARN)"' \
  | grep -o '"msg":"[^"]*"' | sort | uniq -c | sort -rn | head
# Data-quality (requêtes identiques aux détecteurs internal/ops/data_quality.go)
# LOCAL UNIQUEMENT (jamais ouvrir les DuckDB prod tenues RW par le serveur)
cd apps/go-api && go run ./cmd/diag_q <shared_matches_v2.duckdb> "<UNION ALL des compteurs>"
```

## Protocole de reprise de session

Lire ce fichier (statuts) + `git log --oneline -10` sur la branche concernée (B1 =
`hotfix/lusr-shadow-ro`, reste = `fix/monitoring-triage-2026-07`). Une phase est close
quand items statués + gate passé. B7.4 est un soak daté, pas un report.

## Découvertes hors périmètre (à consigner, ne pas traiter)

- 2026-07-07 : `/health` fait un vrai travail DB et répond 503 pendant les fenêtres de
  gate lecture — pour un healthcheck Docker c'est discutable (le conteneur reste
  « healthy » car le seuil de retries absorbe) ; la sonde liveness pure est `/healthz`.
  À revoir dans le plan refonte (onglet État / signaux santé), pas ici.
- 2026-07-07 : disque VPS à 82 % (15 Go libres) — sous surveillance via plan refonte A5.
- 2026-07-07 : logs prod dans le volume persistant `/opt/levelup/data/logs` — bonne
  nouvelle (survivent aux redéploiements) ; la rotation n'est pas vérifiée (auth.log
  local 54 Mo) → item potentiel plan refonte.
- 2026-07-10 : **détecteur data-quality H5 échoue localement** — `GET
  /api/v1/admin/monitoring/data-quality?title=halo_5` renvoie `data_quality_error`
  (« internal error »), alors que HI répond. Probable schéma/table absent côté shared H5
  local. Hors périmètre triage (le plan supposait H5 = 0 partout) → à investiguer.
- 2026-07-10 : **orphan_xuids local = 131** (le plan mesurait 120) — a crû ; la passe
  d'aliases (convergence) est à relancer (couvert par B4.2, prod-gated).
- 2026-07-10 : familles B6.1 `metadata verrouillée`, B6.2 `rta subscribe refusé`,
  B6.3 `endpoint_missing` — les gros compteurs du plan (2121/1348/1015) étaient du bruit
  HISTORIQUE (juin / pré-reboot). En fenêtre juillet ET en prod post-reboot : 0. Les sites
  B6.2 ont même disparu du code courant. La démotion B6.1 reste faite (aligne DC-B2) mais
  son impact courant est nul.
- 2026-07-10 : la démotion B3.3 (ERROR→WARN de la ligne agrégée spartan_cron) touche un
  chemin de contention LOCAL (2e writer air/worktree) ; en prod ce chemin est absent. Choix
  conforme au plan (le lock de dev n'est pas un incident serveur).
- 2026-07-16 (d) : **`database is closed` sur player `stats.duckdb` (op=OpenReadWrite)** —
  NOUVEAU, apparu le 07-14 (0 les 07-12/13 → 47 le 07-14, 11 le 07-15, 54 le 07-16). Chemin =
  lecture des défis prestige (`internal/platform/duckdb/prestige/prestige_player_helpers.go:18-22`,
  `challengeSelectColumns`) : le handle player DB est fermé (lease/B-swap) sous une requête en vol.
  Race de concurrence POST-B1, hors périmètre triage → chantier dédié (candidat : lire sous
  `OpenReadForQuery`/lease plutôt qu'un `OpenReadWrite` exposé à la fermeture concurrente).
- 2026-07-16 (e) : **tables `mode_name_tr` (9/j) et `battlepass_track_definitions` (1/j)
  absentes** (`Catalog Error: Table ... does not exist`) sur des requêtes metadata — lacune de
  schéma/timing bas volume. Hors périmètre → à investiguer (migration manquante, ou requête sur
  un attach metadata avant création de la table).
- 2026-07-16 (f) : `/health` 503 CONFIRMÉ résiduel (97/j, ~5/h réguliers) APRÈS B1 — NON causé
  par LUSR (writer-holds au seuil nominal 2 s), = la Découverte `/health` vs `/healthz` déjà notée
  (07-07). Confirmé pour le plan REFONTE (onglet État / signaux santé) : le healthcheck Docker
  devrait cibler `/healthz` (liveness pure) et non `/health` (travail DB réel).
- 2026-07-16 : B5.5 SOLDÉ (no-op) — sous mandat downtime, binaire buildé côté VPS + dry-run
  READ_ONLY contre une COPIE des DB (0 downtime) : **0 chemin absolu** dans les 2 titres prod
  (HI 139/139, H5 84/84 déjà relatifs). Migration sans effet — le stock est déjà canonique. Le
  binaire `cmd/migrate-media-paths` reste pertinent si un import legacy réintroduisait des chemins
  absolus (garde-fou), mais aucune action prod requise aujourd'hui. Cf. §B5.5.
