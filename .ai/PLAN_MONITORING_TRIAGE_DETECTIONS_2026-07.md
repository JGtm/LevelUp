# PLAN — Triage et traitement des détections monitoring (2026-07)

> Statut : PARTIEL (exécuté le 2026-07-10 sur `fix/monitoring-triage-2026-07`). Le code
> déploy-indépendant est livré et testé (B6.4 anti-flood, B6.1/B3.3 démotions+compteurs) ;
> les items restants sont bloqués par une dépendance explicite — deploy prod de B1 (PR #53
> `hotfix/lusr-shadow-ro` OUVERTE, non mergée), écriture prod interdite (B4/B5.5),
> soak prescrit (B2.4/B7.4), ou auth live locale en recovery (B4.2). Voir statuts par item.
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
- [~] B1.4 Livraison hotfix — branche `hotfix/lusr-shadow-ro` créée depuis origin/main,
      **PR #53 OUVERTE (non mergée)**. Attend le GO explicite du user avant push main
      (deploy prod auto). NON déployée à ce jour (2026-07-10).
- [!] B1.5 Post-deploy — EN ATTENTE DEPLOY (PR #53 non mergée). Vérif prod à faire APRÈS
      merge : disparition du message, watermark avance, `/health` sans 503, writer-hold
      retombe. Symptôme /health 503 confirmé actif en prod post-reboot (échantillon relevé).
- [!] B1.6 Rattrapage backfill — EN ATTENTE DEPLOY (à lancer APRÈS le fix seulement).

**Gate B1** : [!] PARTIEL — fix implémenté et testé (PR #53), deploy prod non effectué
(protocole GO user requis, push main = deploy auto). Les B1.5/B1.6 restent [!] jusqu'au deploy.

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
- [!] B2.4 Soak 24-48 h — DÉLAI PRESCRIT + non pertinent en local (env recovery) ; à mesurer
      en prod (déjà 0 reauth post-reboot). Reste [!] jusqu'à observation prod post-deploy B1.
- [~] B2.5 Warn legacy sync_meta → lots D1a/D2 plan audits (DC-B4).

**Gate B2** : prod pool sain (0 ERROR, 0 reauth). Familles locales = env recovery, hors
cible. `go test ./internal/platform/auth/... ./internal/sync/...` verts (exit 0).

### B3 — Crons en échec (effort : réduit après B0)

Recalibrage : leaderboard prod juillet = 2 ERROR / 3 WARN seulement — l'essentiel du
bruit leaderboard était local (dev). catalog prod juillet = 0 ERROR.

- [~] B3.1 `world_leaderboard_cron` — cron PROD tourne (dernier cycle 2026-07-10 14:15,
      quotidien) et dégrade gracieusement (fallback statique). 5 E + 2 W post-reboot, TOUS =
      `FetchCatalog: statut HTTP 404: classement absent pour cette (saison, playlist)` = le
      reste-à-faire **C3** du chantier leaderboard-catalogue → réf HANDOFF_LEADERBOARD_CATALOGUE.
      Bruit local (scrape ×26) = dev, hors périmètre prod.
- [x] B3.2 `catalog_refresh_cron` — prod post-reboot = 0 ERROR (49 E historiques ÉTEINTS).
      Résidu 6 W `catalog_expand: terminé avec échecs` = expansion partielle gracieuse (asset
      manquant), non bloquant. Confirmé éteint.
- [x] B3.3 `spartan_cron` verrous player DB — le lock per-joueur est DÉJÀ en Debug + agrégé
      une-fois-par-cycle (code existant) ; ligne agrégée passée **ERROR→WARN + compteur expvar
      `spartan_cron_player_db_locked_total`** (DC-B2, `spartan_customization_cron.go`). Absent
      de prod (60 « refresher failed » post-reboot = échecs NON-lock légitimes, WARN).

**Gate B3** : crons prod sans crash ; `go test ./internal/scheduler/...` vert (exit 0).

### B4 — Stock data-quality halo_infinite (effort : moyen ; mêmes BDD local/prod)

- [!] B4.1 Assets UUID bruts (24) — remédiation = action admin d'ÉCRITURE sur données prod
      (interdite cette session). Outillage vérifié présent : endpoint data-quality répond
      (GET 200), `POST /api/v1/admin/monitoring/data-quality` registry-names-backfill câblé.
      Mesure locale = 24 (inchangé, aucune remédiation appliquée).
- [!] B4.2 XUIDs orphelins (131 local ; plan 120) — passe d'aliases nécessite auth LIVE
      (PeopleHub) : auth locale en recovery + écriture prod interdite. Deux blocages cumulés.
- [!] B4.3 Lying bits events (580) — reset via action admin sur prod (écriture prod interdite).
      Mesure locale = 580 (inchangé).
- [!] B4.4 Modes non traduits — mesuré = 8 (endpoint) ; traduction via action admin
      d'écriture (prod interdite).
- [x] B4.5 Tableau relevé (endpoint read-only, local, 2026-07-10) : HI raw_uuid **24**
      (playlists 6/maps 7/pairs 7/variants 4), untranslated_modes **8**, orphan_playlists **0**,
      orphan_xuids **131**, lying_bits_events **580**, lying_bits_weapons **0**. H5 : détecteur
      en ERREUR locale (`data_quality_error` / « internal error ») — consigné en §Découvertes.

**Gate B4** : [!] — remédiation prod-gated (écriture prod interdite) + B4.2 auth live requise ;
mesure faite, outillage admin vérifié présent.

### B5 — Erreurs applicatives résiduelles (effort : moyen)

Liste FERMÉE (relevés 06-07 local + 07-07 prod) :

- [~] B5.1 `duckdb: query failed` / `loadHomeSkillPeak` / `SharedReader unavailable` —
      downstream du writer-hold LUSR (B1). Prod post-reboot duckdb.log = 9 E, loadHomeSkillPeak
      13+6 W : gonflé par le writer-hold. À re-mesurer APRÈS deploy B1 → réf B1 (vérif post-deploy).
- [~] B5.2 `general "http"` ERROR ×632 post-reboot — PROUVÉ = `GET /health → 503` (timeout
      5 s), symptôme DIRECT du writer-hold LUSR (échantillon : 2 lignes /health 503/5000ms).
      → résolu par B1, vérif post-deploy. (Les ×3000 juillet = essentiellement tempête DNS.)
- [x] B5.3 `augmentWithActiveRankedCSRs: GetPlaylistCsr échoué` — 7370 juillet (~99.7 %
      tempête DNS, éteinte) → **19 post-reboot** sur 3 j (~6/j, 401/données transitoires).
      Résidu négligeable ; l'anti-flood B6.4 couvre le flood réseau amont si récidive.
- [~] B5.4 `highlight_events parse_anomaly` ×32 local — limite connue du décodeur film →
      réf chantier filmdec (HANDOFF_FILM_EXTRACTION).
- [!] B5.5 `mediaStoredPathToURL: aucun mapping (path legacy)` — `migrate-media-paths` remap
      des chemins DÉPENDANT de l'environnement : run réel = prod (écriture interdite) ; local
      muterait des données non versionnées sans valeur pour la cible prod. Reste [!] prod-gated.
- [x] B5.6 `configuration non sûre pour un déploiement multi-user exposé` — DÉJÀ log
      une-fois-au-boot (`cmd/server/main.go:320`, séquence boot après `cfg.Validate()`) :
      vérifié sur pièces. DC-B2 satisfait (le nombre d'occurrences = nombre de boots dev).
- [x] B5.7 `service.log` prod — 28 987 W = stock HISTORIQUE pré-reboot ; post-reboot (07-08→10)
      service.log = **11 W** (career_live LoadLastCareerRank/EnrichFromMetadata, live-fetch
      transitoire). Sous le seuil 100/j. Stock historique non re-généré.

**Gate B5** : chaque item statué. Familles tempête-DNS éteintes ; familles cascade-LUSR
en [~] B1 (vérif post-deploy) ; B5.6 déjà conforme ; B5.5 prod-gated.

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
      résultats archivés dans l'en-tête « Mesure de clôture » et le thought_log.
- [x] B7.2 Tous les items statués ; découvertes consignées ci-dessous.
- [x] B7.3 Gate final : `cd apps/go-api && go test ./...` = exit 0 ;
      `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... [+touchés]`
      = exit 0 ; gofmt/vet propres ; skill `delivery-checklist` ; entrée `thought_log.md`.
- [!] B7.4 Cible mensuelle — T0 = 2026-07-10. Re-mesure T0+30 j (2026-08-09) : DÉLAI PRESCRIT
      (pas un report) ; MAIS conditionné au deploy de B1 (« ERROR ≈ 0/jour » suppose la cascade
      LUSR éteinte). À ré-armer une fois PR #53 déployée.

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
