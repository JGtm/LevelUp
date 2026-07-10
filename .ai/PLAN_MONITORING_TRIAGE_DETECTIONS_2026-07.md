# PLAN — Triage et traitement des détections monitoring (2026-07)

> Statut : PRÊT (B0 exécuté le 2026-07-07 — plan RECALIBRÉ sur les mesures prod).
> Exécution sous contrat du skill `plan-execution`.
> Branche cible : `fix/monitoring-triage-2026-07` (1 branche, N commits) — SAUF B1
> (régression prod active) : hotfix depuis `origin/main` (voir DC-B6).
> À exécuter AVANT `.ai/PLAN_MONITORING_REFONTE_2026-07.md`.

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

- [ ] B1.1 Seam : `RunLUSRV2ShadowOwnerOnly` ne reçoit plus un `sharedDB *sql.DB` brut
      mais une petite interface de lease (Read + Write-burst) déclarée côté
      `sync/skill` (pas d'import de sync — éviter le cycle) ; l'engine passe un
      adaptateur sur `SharedAccess` ; les callers CLI/backfill/h5-livesync
      (`lusr_full_recompute.go:46`, `games/halo_5/livesync/wire.go:107`, cmd/*) passent
      un adaptateur trivial sur leur handle déjà RW (Write = même handle).
- [ ] B1.2 Chunking : `loadShadowMatches` sous Read ; boucle `processOneShadowMatch`
      par chunks de N matchs sous burst Write (constante nommée, même esprit que
      `postsyncEventsBurstChunk`) ; 0 match → aucun burst (cas stationnaire dominant).
- [ ] B1.3 Tests : (a) test d'intégration qui REPRODUIT le bug prod — shadow owner-only
      sur provider dont l'attach shared est read-only → doit acquérir le Write et
      réussir (rouge avant fix, vert après) ; (b) non-régression des tests shadow
      existants ; (c) audit exhaustif des AUTRES segments `shared.Read(` du post-sync
      (`engine_postsync*.go`) : vérifier sur pièces qu'aucun autre bloc classé lecture
      n'appelle un chemin qui écrit shared (events/weapons déjà OK en bursts Write ;
      scoring/intensités écrit player DB — confirmer), consigner le résultat ici.
- [ ] B1.4 Livraison hotfix (DC-B6) : branche depuis origin/main, gates
      (`go test ./...` + `go test -tags=integration -p 1 ./internal/...` persist/sync),
      PRÉVENIR le user avant push main (deploy auto).
- [ ] B1.5 Post-deploy : vérifier en prod (lecture seule) : plus d'occurrence du
      message, watermark avance (LUSR récents réapparaissent), `/health` ne produit
      plus de 503, `writer RW tenu au-delà du seuil` retombe.
- [ ] B1.6 Rattrapage : si le watermark a 4+ jours de retard, lancer le backfill LUSR
      canonique documenté (`lusr_v2_canonical_backfill --commit`, mémoire projet) —
      APRÈS le fix seulement.

**Gate B1** : B1.5 constaté sur pièces en prod + tests verts.

### B2 — Pool auth / tokens (effort : réduit après B0)

Recalibrage : les ~21 000 erreurs pool prod de juillet étaient à ~99 % la panne DNS
(éteinte). Post-reboot : 0 ERROR pool, 9 warns 429 (AIMD ok). Reste le fond local :
slots morts au boot, `refresh_token mort — reauth_required` ×45, `world-enrich skippé`
×121 (local).

- [ ] B2.1 Inventaire sur pièces des slots du pool : xuids déclarés vs RT valides
      (`data/auth/watcher_tokens/*.json`, `db_profiles.json`).
- [ ] B2.2 Retirer/blacklister les xuids aux RT morts (3 connus) — liste d'exclusion
      config, PAS de suppression des fichiers tokens (DC-B1).
- [ ] B2.3 Vérifier que les RT valides se rafraîchissent (helper canonique
      `auth.RefreshHaloTokensViaStoreFirst`, cache invalidé après rotation).
- [ ] B2.4 Soak 24-48 h local : plus de `skip slot` / `refresh_token mort` /
      `world-enrich skippé` hors comptes volontairement exclus.
- [~] B2.5 Warn legacy sync_meta → lots D1a/D2 plan audits (DC-B4).

**Gate B2** : script §Mesure post-fix = 0 occurrence des familles ci-dessus ;
`go test ./internal/platform/auth/... ./internal/sync/...` verts.

### B3 — Crons en échec (effort : réduit après B0)

Recalibrage : leaderboard prod juillet = 2 ERROR / 3 WARN seulement — l'essentiel du
bruit leaderboard était local (dev). catalog prod juillet = 0 ERROR.

- [ ] B3.1 `world_leaderboard_cron` : vérifier l'état PROD courant (dernier cycle OK ?
      snapshots insérés ?). Le bruit local (scrape échoué ×26, vide page 1 ×19,
      FetchCatalog 500 ×6) : diagnostiquer en local ; si le fix relève du reste-à-faire
      C3 du chantier leaderboard-catalogue, statuer [~] avec référence.
- [ ] B3.2 `catalog_refresh_cron` (49 E historiques locaux) + `catalog drain: markError
      failed` (×1738 W, dont 16 416 W prod historiques — vérifier fenêtre) : confirmer
      éteints ; sinon corriger.
- [ ] B3.3 `spartan_cron` verrous player DB (×245 + ×60, LOCAL uniquement — cause :
      2e writer air/worktree) : dégrader en WARN une-fois-par-cycle avec compteur
      (DC-B2) ; vérifié absent des logs prod.

**Gate B3** : un cycle complet de chaque cron sans ERROR (prod pour B3.1, local pour
le reste) ; `go test ./internal/scheduler/...` vert.

### B4 — Stock data-quality halo_infinite (effort : moyen ; mêmes BDD local/prod)

- [ ] B4.1 Assets UUID bruts (24) : `POST /admin/actions/registry-names/backfill`
      (dry-run puis réel) ; le reste = CLI testées (`populate-assets`,
      `backfill_registry_names`) — jamais de sondes throwaway.
- [ ] B4.2 XUIDs orphelins (120) : passe d'aliases (convergence PSA /
      `world-aliases-persist`). Cible : > 90 % résolus ; reste documenté [!].
- [ ] B4.3 Lying bits events (580) : dry-run puis reset (DC-B3) ; re-mesure J+2 : 0.
- [ ] B4.4 Modes non traduits : mesurer (endpoint data-quality) puis traduire via
      `POST /admin/actions/translations/mode` (FR sans anglicismes).
- [ ] B4.5 Re-relever le tableau complet (§Mesure) : HI raw_uuid = 0, lying_bits = 0 ;
      H5 re-confirmé à 0.

**Gate B4** : tableau aux cibles ; écritures uniquement via actions admin/CLI canoniques.

### B5 — Erreurs applicatives résiduelles (effort : moyen)

Liste FERMÉE (relevés 06-07 local + 07-07 prod) :

- [ ] B5.1 `duckdb: query failed` (139 + 86 after-recovery prod juillet) +
      `loadHomeSkillPeak Phase A/B` + `SharedReader unavailable` + `home: GetHomePage
      error` ×23 prod : re-mesurer APRÈS B1 (le writer-hold LUSR gonflait les fenêtres
      B-swap). Si résiduel : borne de retry/dégradation propre + compteur (DC-B2) ;
      sinon clore [x] par référence B1.
- [ ] B5.2 `general "http"` ERROR ×3 000 prod juillet (500 servis) : vérifier la
      corrélation temporelle avec la tempête DNS (échantillonner 5 lignes hors fenêtre
      d'incident) ; hors incident → investiguer ; sinon classer incident [x].
- [ ] B5.3 `augmentWithActiveRankedCSRs: GetPlaylistCsr échoué` ×7 370 prod : idem —
      part DNS vs part 401/données ; corriger le résiduel.
- [ ] B5.4 `highlight_events parse_anomaly` ×32 local : échantillonner 3 match_ids ;
      limite connue du décodeur film → [~] référence chantier filmdec + compteur.
- [ ] B5.5 `mediaStoredPathToURL: aucun mapping (path legacy)` ×896 local / ×3 106 W
      media prod (vérifier msg) : exécuter `migrate-media-paths` (dry-run d'abord),
      puis 0 récidive.
- [ ] B5.6 `configuration non sûre pour un déploiement multi-user exposé` : log
      une-fois-au-boot (DC-B2).
- [ ] B5.7 `service.log` prod : 28 987 W historiques — top messages sur fenêtre
      juillet, traiter ce qui dépasse 100/jour ou statuer.

**Gate B5** : chaque item statué ; script §Mesure 7 jours post-fix : familles éteintes
ou converties en compteurs.

### B6 — Démotions de bruit et anti-flood (effort : rapide)

Liste FERMÉE, chaque démotion = compteur expvar + première occurrence loggée (DC-B2) :

- [ ] B6.1 `metadata verrouillée, nouvelle tentative...` ×2 121 local (retry interne)
      → DEBUG + compteur.
- [ ] B6.2 `rta: subscribe refusé` ×1 348 / WebSocket ×138 (local) : cause ; dégradation
      attendue hors-jeu → WARN une-fois + compteur ; sinon corriger.
- [ ] B6.3 `endpoint_missing` ×1 015 (general local) : identifier, corriger ou dégrader.
- [ ] B6.4 Anti-flood incident réseau (leçon DNS : 67 165 W rest_poller en 7 j) :
      étouffement des logs d'échec réseau IDENTIQUES répétés (1 log / N occurrences ou
      / fenêtre, compteur exact en expvar) sur rest_poller + halo_api + oauth_refresh.
      C'est le pare-feu contre le prochain incident infra qui noie la page détections.
- [ ] B6.5 `backup: export table échoué (ignoré)` ×1 748 local (juillet = 0) : confirmer
      éteint, sinon rouvrir.

**Gate B6** : `go build ./... && go test ./...` verts ; cycle local complet : messages
démotés absents de WARN/ERROR ; test de l'étouffeur B6.4.

### B7 — Vérification finale et clôture (effort : rapide)

- [ ] B7.1 Script §Mesure complet (logs local + prod, data-quality) archivé ici.
- [ ] B7.2 Tous les items statués ; découvertes consignées ci-dessous.
- [ ] B7.3 Gate final : `cd apps/go-api && go test ./...` +
      `go test -tags=integration -p 1 ./...` ; skill `delivery-checklist` ; entrée
      `thought_log.md`.
- [ ] B7.4 Cible mensuelle : noter T0 ; à T0+30 j, re-mesurer prod (ERROR ≈ 0/jour hors
      incidents externes) — délai d'observation prescrit, pas un report.

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
