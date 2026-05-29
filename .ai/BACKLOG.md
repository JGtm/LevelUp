— Tâches et TODO centralisés

> Mis à jour le 2026-05-29. Fusion de : `backlog`, `BACKLOG.md`, `BACKLOG_COACH_PRESTIGE.md`.

---

## 🔄 Aucune tâche en cours

---

## 📋 Backlog

---

### [POST-V7] Main Merge (blocking)

- [ ] Créer PR: `fix/theme-consistency-tokens` → `main`
- [ ] Attendre approval + CI validation
- [ ] Merger PR vers main

---

### [POST-V7] Main Merge (Phase 4 Collect→Persist) — ajouté 2026-05-24

**Branche source** : `refactor/collect-persist` (7 commits Phase 4 + 4 commits cycles précédents)

**Commits inclus** (du plus ancien au plus récent) :
- `03322560` fix(auth) : E.v1 legacy store fix
- `0e4b368f` feat(persist,p4.2) : PostSyncLUSRPersister
- `14dfd135` feat(persist,sync,p4.4) : batch INSERT-only post-sync (5 sites)
- `4ef122b7` feat(migration,p4.5) : RebuildMatchSkillRankART + smoke validé
- `f243b235` chore(p4.6,cleanup) : supprime singleflight/CHECKPOINT/auto-heal
- `82d0aa1a` feat(p4.7,closure) : BatchQueue wiring + janitor + flip defaults
- `463418af` chore(p4.8,cleanup) : revert acad4603 UPDATE-then-INSERT
- `08582b89` chore(p4.7,hygiene) : PathResolver pour data/wal + data/sync_cache
- `d3825c2f` chore(p4.9,tests) : flip async ON + fix 9 pre-existing test failures
- `4508df92` feat(auth,p-Ev2) : Pool.AddOrUpdateSource + periodic re-scan
- `157d80a8` feat(auth,p-2.5b) : RefreshLoop.WithMultiUserMirror

**Pré-requis merge** :
- [ ] CI green sur `refactor/collect-persist` (go test ./... full suite)
- [ ] Squad review du runbook `.ai/RUNBOOK_PHASE4_DEPLOY.md` (procédure deploy)
- [ ] User a-t-il déjà migré sa DB locale ? Si non : suivre runbook AVANT merge

**Procédure** :
- [ ] Créer PR: `refactor/collect-persist` → `main`
- [ ] Body PR : référencer ADR 0019, `.ai/RUNBOOK_PHASE4_DEPLOY.md`, et thought_log entries Phase 4.4→4.9
- [ ] Attendre approval + CI validation
- [ ] Merger PR vers main (squash NON recommandé — chaque commit Phase 4.x est une étape logique distincte avec entrée thought_log)

**Post-merge** :
- [ ] Tag release : `v8.0.0-phase4` (breaking si user a une DB legacy non rebuilt)
- [ ] CHANGELOG.md : section Phase 4 (incident ART résolu, runbook deploy, breaking)
- [ ] Annoncer dans channel ops : pré-requis `force_rebuild_art --all true` AVANT 1er boot sur prod

**Items reportés (post merge OK)** :
- [ ] Documenter le default async ON dans le README utilisateur
- [ ] Si CI metrics observent latence WAL : tuning du janitor (24h → 12h ?)

---

### [POST-V7] Post-Merge Validation

- [ ] Vérifier baseline tests verts post-merge
- [ ] Vérifier coverage >= 76%
- [ ] Vérifier no new warnings (golangci-lint)

---

### [POST-V7] Optional Cleanup

- [ ] Supprimer branche `refactor/leased-writer-enforcement` (si besoin)
- [ ] Supprimer branche `fix/theme-consistency-tokens` (si besoin)

---

### [POST-V7] Optional Observability

- [ ] Setup monitoring: `dblease_acquire_total{kind=player|shared_matches,status=success|timeout}`
- [ ] Setup alerting si timeouts excessifs
- [ ] Documenter les seuils d'alerte

---

### [POST-V7] Optional Documentation

- [ ] Créer `docs/adr/0013-leased-writer-enforcement.md`
  - Contexte: P1 Prestige during sync
  - Décision: DLeasedWriter type + dblease package
  - Conséquences: Unified observability, deadlock resolution

---

### [auth/unification] E.v2 — Callback push watcher → pool (optimisation latence) ✅ LIVRÉ 2026-05-24

**Livré via approche révisée** (commit `4508df92`) :
- `Pool.AddOrUpdateSource(ctx, src)` method — hot-add ou refresh d'un slot
- Periodic re-scan goroutine en main.go (15min tick) : Discovery.Scan() + Pool.AddOrUpdateSource pour chaque source
- 5 tests TDD GREEN

**Différence vs plan initial** : au lieu de callback push depuis le watcher, periodic re-scan global. Avantages :
- Couvre aussi les nouveaux env vars ajoutés en cours de session (pas seulement watcher)
- Pas de couplage watcher → pool
- Latence max 15 min (acceptable pour ce use case)

**LIMITATION documentée** : nouveau slot ajouté APRÈS boot est seulement reachable via PolicyPinnedPlayer (canal round-robin sized at boot). Pas un problème pour LevelUp (auto-sync utilise PolicyPinnedPlayer per gamertag).

---

### [auth/unification] PR 2.5b — Watcher daemon tracker migration TokenStore → MultiUserTokenStore ✅ PARTIEL 2026-05-24

**Phase 1 livrée** (commit `157d80a8`) — mirror write :
- `RefreshLoop.WithMultiUserMirror(multi)` builder
- Chaque refresh XSTS/OAuth du tracker (legacy store) est aussi écrit dans MultiUserTokenStore[XSTSXUID]
- 3 tests TDD GREEN
- Read continue via legacy store (compat user, 0 changement comportement)

**Phase 2 reportée** (read-path switch, ~2-3h) :
- Daemon principal lit depuis multi-user store au lieu de legacy
- Tracker rotation : si tracker xuid revoke, basculer sur autre user valide
- **Décision product requise** : si N users SSO valides, quel est le tracker principal ? (random ? premier connecté ? config explicite ?)

**Quand traiter Phase 2** : si on observe en prod un cas de "tracker xuid revoque → daemon mort" qui doit être fixed sans reboot. À ce jour, le fallback existing (lines 953-987 main.go) suffit au boot.

---

### [auth/unification] Read-path switch vers MultiUserTokenStore (future PR post-2.5b)

**Noté le** : 2026-05-24 | **Priorité** : 🟢 Basse — la phase 1 mirror suffit pour maintenir la cohérence.

**Contexte** : Suite à PR 2.5b phase 1 (mirror legacy → multi-user), le multi-user store est toujours up-to-date avec le tracker actuel. La prochaine étape serait que le watcher daemon **LISE** depuis multi-user au lieu de legacy.

**Bloqueur design** : si plusieurs users SSO sont enregistrés, quel xuid devient le "tracker principal" pour le watcher daemon ? Options :
- 1. Premier dans la map (non-déterministe Go map iteration)
- 2. Config explicite `watcher.tracker_xuid` dans app_settings.json
- 3. User le plus récemment loggé (UpdatedAt max)
- 4. User avec XSTS valide le plus longtemps (XSTSExpiresAt max)

**Effort** : ~2-3h une fois la décision design prise.

**Quand traiter** : pas urgent. Single-user reste le cas dominant.

---

### [auth/security] Chiffrement at-rest des watcher tokens

**Noté le** : 2026-05-24 | **Priorité** : 🟢 Basse — outil reste single-user local, threat model accepté.

**Contexte** : `data/auth/watcher_tokens.json` (et le futur `watcher_tokens/{xuid}.json` post PR 2.5b) contient le **XSTS token** + le **MSAL cache** (qui contient le refresh_token Microsoft) **en clair JSON**, protégés uniquement par les permissions fichier (0600). Aucun chiffrement at-rest (`grep -rn "Encrypt|cipher|aes" internal/platform/auth/` → 0 résultat).

**Threat model actuel** : outil desktop single-user local. Vol de fichier implique déjà compromission machine. Standard de fait dans la communauté Halo (SPNKr Python identique). Tokens TTL court (Spartan ~3h, XSTS ~12h) + refresh révocable côté Microsoft.

**Pertinence du fix** :
- 🟢 Bas si l'outil reste single-user local
- 🟡 Moyen si on distribue à des users non-techniciens (ils n'auront pas le réflexe de protéger leur HOME)
- 🔴 Haut si déploiement multi-tenant ou cloud (mais pas le cas actuel)

**Fix proposé** : wrap natif OS via package Go `github.com/zalando/go-keyring` ou équivalent :
- **Windows** : DPAPI (`CryptProtectData`)
- **macOS** : Keychain
- **Linux** : libsecret / GNOME Keyring

Effort ~3h (1 nouveau package `internal/platform/auth/secureStore` qui wrap Read/Write/Delete avec fallback en clair si pas de keychain dispo).

**Quand traiter** : avant distribution publique grand-public, OU si un incident token leak survient.

---

### [auth/cleanup] Migration watcher daemon vers MultiUserTokenStore (PR 2.5b)

**Noté le** : 2026-05-24 | **Priorité** : 🟡 Moyenne — dette technique connue, blocage partiel pour vrai multi-user.

**Contexte** : 2 stores tokens coexistent dans `internal/platform/auth/` :
- **TokenStore legacy** (`token_store.go`) — mono-user, `data/auth/watcher_tokens.json`. Utilisé par le watcher daemon historique.
- **MultiUserTokenStore** (`multi_user_token_store.go`) — multi-user, `data/auth/watcher_tokens/{xuid}.json`. Utilisé par SSO Xbox PR 2.5a.

Commentaire explicite : "Migration du watcher : différée à PR 2.5b".

**Impacts** :
- 1er user = mono-user au boot (watcher daemon ne sait gérer qu'un seul `current_user`)
- Si plusieurs utilisateurs Xbox sur la même machine, conflit potentiel
- Doublon de code (2 stores quasi-identiques à maintenir)

**Fix** : Refactor `internal/sync/watcher_daemon.go` pour utiliser `MultiUserTokenStore` au lieu de `TokenStore`. Migrer les tokens existants au boot (lit `watcher_tokens.json` → écrit `watcher_tokens/{xuid}.json` → archive l'ancien). Effort ~2h.

**Quand traiter** : avant cas d'usage multi-utilisateur réel sur même machine, OU avec le sprint auth unification (cf. `.ai/PLAN_AUTH_PROVIDER_UNIFICATION.md`).

---

### [persist/safety] Garde-fous restants post-Phase 3 Collect→Persist

**Noté le** : 2026-05-24 | **Priorité** : 🟡 Moyenne — non-bloquants pour Phase 3 (rollback via feature flag + code legacy intact)

**Contexte** : Audit safety/guards du chemin `submitMatchAsBatch` (cf. ADR 0019, RUNBOOK_PHASE3, thought_log 2026-05-23). 19 garde-fous actifs livrés ; 6 gaps identifiés et reportés ici. Le gap C (timeout par Persist call) a été fixé immédiatement car simple et utile en prod (commit 2026-05-24).

**Gaps à traiter quand pertinent** :

1. **[A] Retry avec backoff sur erreur transitoire** (`persist`)
   - **Problème** : Un Persist qui échoue (lock timeout, DB busy, IO error) reste en WAL jusqu'au prochain boot. Pas de retry dans le cycle courant.
   - **Impact** : Latence ajoutée (le batch attend le prochain reboot ou auto-sync) sans réelle perte de données.
   - **Fix** : `persist.WithRetry(maxRetries, baseDelay)` autour du Persist call dans `submitMatchAsBatch` ou dans le Worker. Backoff exponentiel : 1s, 2s, 4s, abandon.
   - **Effort** : ~1h.

2. **[B] DLQ après N retries** (`persist`)
   - **Problème** : Si un batch corrompu cause une erreur répétée, il reste en WAL indéfiniment (jusqu'à `PurgeOldWAL > 7j`). Pas de dead-letter quarantine.
   - **Impact** : Batches "poison" boucle au boot via `RecoverPending` jusqu'à expiration WAL. Logs polluent.
   - **Fix** : Compteur de retries dans le WAL filename (ex. `batch_id.attempts=N.json`) ou métadonnée dans le JSON. Au-delà de N (3-5), déplacer dans `walDir/dlq/` + alerte ERROR.
   - **Effort** : ~1h30.

3. **[D] Circuit breaker sur série d'erreurs** (`persist`)
   - **Problème** : Si Persist échoue sur 10 batches consécutifs, le sync continue à submit (potentiel cascading failure).
   - **Impact** : Surconsommation API + ressources + logs spam quand DB est en panne.
   - **Fix** : Pattern circuit breaker classique (closed/open/half-open). Si error rate > threshold sur fenêtre 1min, ouvrir → skip Submit + log WARN + métrique. Half-open recheck toutes les 30s.
   - **Effort** : ~2h.

4. **[E] Health check endpoint `/health/persist`** (`api`)
   - **Problème** : Pas d'endpoint dédié pour load balancer / monitoring externe (Datadog, Uptime Robot).
   - **Impact** : Diagnostic ops via `/debug/vars` uniquement (pas de status compact).
   - **Fix** : `GET /health/persist` → 200 OK + JSON `{wal_pending: 0, last_persist_ok_at: "...", recent_errors: 0}`. 503 si pending > seuil ou erreurs récentes > seuil.
   - **Effort** : ~45min.

5. **[F] RecoverPending wiré au boot serveur** (`cmd/server`)
   - **Problème** : Le mode async (queue + worker) a un mécanisme `RecoverPending` testé en E2E mais pas appelé au boot du serveur prod.
   - **Impact** : Si crash mid-cycle en mode async, les batches en WAL ne sont pas rejoués au boot suivant.
   - **Mitigation actuelle** : Mode async non-activé par défaut (Phase 3 active uniquement le mode sync sans WAL).
   - **Fix** : Dans `cmd/server/main.go`, au boot : `queue := persist.NewBatchQueue(...)` ; `_ = queue.RecoverPending()` ; `worker.Run(ctx)` (goroutine).
   - **Effort** : ~1h. **Dépendance** : Faire AVANT d'activer le mode async en prod.

6. **[G] Test E2E avec FATAL DuckDB injecté** (`internal/persist`)
   - **Problème** : Les tests TDD unitaires couvrent les cas isolés (atomicity, idempotence, parse error). Pas de test qui simule un FATAL DuckDB mid-batch et vérifie la recovery propre.
   - **Impact** : Confiance moindre sur le path de recovery en cas de crash réel.
   - **Fix** : Mock `txBeginner` qui retourne une erreur FATAL après N rows insérées. Vérifier que la TX rollback, le WAL reste, et RecoverPending peut rejouer.
   - **Effort** : ~1h.

**Quand traiter** : après Phase 3 activée et stabilisée (10+ cycles propres en prod). Ou plus tôt si un incident révèle un de ces gaps.

---

### [frontend/nav] Nettoyage code de navigation mort — `PlayerScopeNav` + constantes `shellNavigation`

**Noté le** : 2026-05-10 | **Priorité** : Basse — cosmétique, 0 impact fonctionnel

**Contexte** : Audit pages orphelines du 2026-05-10. Après suppression des pages mortes, il reste du code de navigation inutilisé :

1. **`apps/web/src/components/shell/PlayerScopeNav.tsx`** — composant de nav défini mais jamais importé dans aucun layout ni page. Superseded par `NavL1.tsx`. À supprimer entièrement avec son test `shellNavigation.test.ts` si les cas testés couvrent uniquement ce composant.

2. **`shellNavigation.ts` — `PLAYER_PRIMARY_NAV_ITEMS` + `PLAYER_SECONDARY_NAV_ITEMS`** — deux constantes uniquement consommées par `PlayerScopeNav` (lui-même mort). `buildPlayerDestination` reste actif (utilisé par `NavL1.tsx`) → garder le fichier, supprimer uniquement ces deux exports.

3. **`apps/web/src/lib/pageTitle.ts` lignes ~59-60`** — consomme encore `PLAYER_PRIMARY_NAV_ITEMS` et `PLAYER_SECONDARY_NAV_ITEMS` via `.map()` pour construire `ROUTE_TITLE_RULES`. Après suppression des constantes, remplacer par une liste statique inline ou vider.

**Ce qu'il faut faire** :
1. Vérifier que `shellNavigation.test.ts` ne teste que `PlayerScopeNav`/les constantes mortes — si oui, supprimer le fichier de test aussi
2. Supprimer `PlayerScopeNav.tsx`
3. Retirer `PLAYER_PRIMARY_NAV_ITEMS` + `PLAYER_SECONDARY_NAV_ITEMS` de `shellNavigation.ts`
4. Mettre à jour `pageTitle.ts` pour ne plus les consommer

**Effort** : ~30 minutes

---

### [db-concurrency] Suite et fin du refactor `leased-writer-enforcement`

**Noté le** : 2026-05-05 | **Priorité** : 🔴 Bloquant merge pour les 3 premiers points, 🟡 follow-up pour le reste

**Contexte** : la branche `refactor/leased-writer-enforcement` (9 commits, `ae03600f`→`5423da2a`) résout P1 (Prestige HTTP), P2 (Notifications), P3 (Media atomicité) en introduisant le type `*dblease.LeasedWriter`, deux patterns de configuration (Wrapper / Option), et 41 nouveaux tests + 3 benchmarks. Le code complet n'a **pas pu être buildé localement** faute de `gcc`/cgo dans l'environnement de la session — vérifié uniquement via `gofmt` et cohérence statique des signatures. Plan complet dans [.ai/V7/PLAN_DB_WRITE_CONCURRENCY.md](.ai/V7/PLAN_DB_WRITE_CONCURRENCY.md), ADR officielle dans [docs/adr/0013-leased-writer-enforcement.md](docs/adr/0013-leased-writer-enforcement.md).

**🔴 Bloquant merge — à faire sur une machine cgo-enabled** :

1. **Build complet cgo** :
   ```bash
   cd apps/go-api && make test 2>&1 | tee /tmp/post_branch_test.log
   ```
   Si rouge : la cause est probablement une typo / import / signature dans un de mes commits 2-7. Me remonter les fichiers en échec pour correctif.

2. **Vérifier le baseline test** : relancer `bash scripts/check_test_baseline.sh tests` post-build cgo. Doit confirmer que les 1662 tests de la baseline pré-migration sont toujours présents et verts (contrat de non-régression du commit 0).

3. **Valider les 2 tests d'intégration coordination** (build tag `integration`) :
   - `TestCoordination_SyncLease_BlocksHTTPWriter` ([sync/lease_test.go](apps/go-api/internal/sync/lease_test.go))
   - `TestCoordination_HTTPWriter_BlocksSyncLease`
   
   Si rouge → la coordination sync↔HTTP est cassée et P1 n'est pas réellement résolu. Faux sentiment de sécurité.

**🟡 Follow-up dans des PRs séparées** :

4. **Câbler les 2 scripts en CI** : `scripts/check_test_baseline.sh` et `apps/go-api/scripts/check_lease_enforcement.sh` doivent tourner en GitHub Actions. PR infra dédiée — pas dans le scope db-concurrency.

5. **Tests intégration concurrentiels manquants** : le plan v3 prévoyait 17 scénarios sous build tag `integration`, seuls 2 ont été ajoutés (commit 7). Manquent (cf. [.ai/V7/PLAN_DB_WRITE_CONCURRENCY.md §Tests d'intégration concurrentiels](.ai/V7/PLAN_DB_WRITE_CONCURRENCY.md)) :
   - `TestSyncVsPrestigeConcurrent` — sync long + 10 POST /challenges en parallèle.
   - `TestSyncHookNoDeadlock` — pipeline sync + RunPostSyncHook avec timeout court (échoue par timeout si deadlock).
   - `TestSyncVsNotificationsConcurrent`, `TestSyncVsMediaLikeConcurrent`, `TestSyncVsMatchFavoriteConcurrent`.
   - `TestMediaLikeRollback`, `TestMediaLikeAtomic_PanicMidTx`, `TestPrestigeEmitEventAtomicity`.
   - `TestSyncDeltaProducesSameOutput`, `TestSyncFullProducesSameOutput`, `TestSyncEngineFullPipeline` (ISO bit-à-bit).
   - `TestNotificationsBurst`, `TestSyncBurstNoLeak`, `TestPrestigeBurst`.
   - `TestProcessKillNoStaleLock`, `TestExternalProcessLock` (crash recovery).

6. **Tests `Atomic_Success` + `Atomic_RepoError_Rollback` cgo-only** : documentés comme différés au commit 6 (exigent une vraie *sql.DB DuckDB :memory: pour `BeginTx`). À ajouter dans `internal/service/media_service_test.go` avec build tag `integration`. Couvrent l'invariant central de P3 (rollback observable du `*sql.Tx` mid-transaction).

7. **Migrer sync engine vers `dblease.AcquireWriterCtx`** : 11 sites dans [sync/engine.go](apps/go-api/internal/sync/engine.go), [sync/backfill_weapons.go](apps/go-api/internal/sync/backfill_weapons.go), [sync/friends_recompute.go](apps/go-api/internal/sync/friends_recompute.go). Bénéfice : alimenter les compteurs `dblease_acquire_total{kind}` du commit 1 (visibilité observabilité). Risque : ~63 tests sync à préserver bit-à-bit. Documenté comme dette dans [sync/lease.go](apps/go-api/internal/sync/lease.go). Non bloquant pour la résolution fonctionnelle de P1/P2/P3.

**🟢 Validation manuelle pré-prod** :

8. **Suite de non-régression métier** : à valider en local après merge (cf. [.ai/V7/PLAN_DB_WRITE_CONCURRENCY.md §Suite de non-régression métier](.ai/V7/PLAN_DB_WRITE_CONCURRENCY.md)).
   - Sync delta sur joueur réel : durée + nb matchs identiques.
   - Création/édition/abandon défi via UI : flow inchangé.
   - Page Prestige `/api/v1/prestige/me` : leaderboard et PP cohérents.
   - Toggle favori match : visible immédiatement, persisté.
   - Like/unlike média : état persistant après refresh.
   - Notifications : émission, mark-read, suppression OK.
   - **Comportements concurrents observables** : pendant un sync long, un POST /challenges retourne 200 ou 503 (jamais 500), retry après 5s succède.

9. **Vérifier `PRESTIGE_ENABLED` en prod** : la variable est `true` par défaut dans le code ([prestige/sync_hook.go:32](apps/go-api/internal/prestige/sync_hook.go#L32) — false uniquement si la var vaut explicitement `0/false/no/off`). Si l'env de prod ne pose pas la var explicitement, P1 est déjà actif → la résolution apportée par cette branche est immédiatement bénéfique. Si la var est `false`, on peut activer Prestige plus tranquillement après les vérifs cgo (cf. ADR-0005).

**Conditions de déblocage** :
1. ❌ Build cgo complet vert (point 1).
2. ❌ Baseline tests préservée (point 2).
3. ❌ Tests coordination intégration verts (point 3).
4. Une fois 1-3 OK : merge de la branche vers `fix/theme-consistency-tokens` ou `main` selon la stratégie.

**Documents liés** :
- [.ai/V7/PLAN_DB_WRITE_CONCURRENCY.md](.ai/V7/PLAN_DB_WRITE_CONCURRENCY.md) — plan v3 complet (~1100 lignes)
- [docs/adr/0013-leased-writer-enforcement.md](docs/adr/0013-leased-writer-enforcement.md) — ADR officielle EN
- [docs/FR/adr/0013-leased-writer-enforcement.md](docs/FR/adr/0013-leased-writer-enforcement.md) — version FR
- [.ai/baselines/tests_pre_migration.jsonl](.ai/baselines/tests_pre_migration.jsonl) — 1662 tests baseline figés
- [.ai/thought_log.md](.ai/thought_log.md) — 9 entrées commit-par-commit (pivots documentés)

---

### [feedback-drawer] Activer la sync des labels GitHub après merge sur main

**Noté le** : 2026-05-05 | **Priorité** : Bloquante au merge de `feat/feedback-drawer`

**Contexte** : la feature drawer feedback (PR `feat/feedback-drawer`) introduit un workflow `.github/workflows/sync-labels.yml` qui crée automatiquement les labels GitHub (`feedback`, `bug`, `severity:critical`, `area:synthesis`, etc.) à partir de `.github/labels.yml`. Le workflow est trigger sur `push: branches: [main]` quand `labels.yml` change — donc au premier merge il devrait s'exécuter tout seul, **mais** la première fois c'est plus prudent de le lancer manuellement pour vérifier qu'il marche avant qu'une issue feedback réelle arrive.

**À faire — juste APRÈS le merge de la PR feedback-drawer sur main** :

1. Aller sur https://github.com/JGtm/LevelUp/actions
2. Cliquer dans la liste de gauche sur **Sync labels**
3. À droite, **Run workflow** → branch: `main` → **Run workflow**
4. Attendre ~30 secondes
5. Vérifier sur https://github.com/JGtm/LevelUp/labels que tous les labels du `.github/labels.yml` sont présents (`feedback`, `bug`, `enhancement`, `question`, `severity:*`, `area:*`, `triage:*`).

**Pourquoi ce n'est pas critique au moment du merge** : sans ce run, la première issue feedback créée via le drawer aura une URL avec des `labels=feedback,bug,severity:high,area:synthesis` qui ne pourront pas être appliqués par GitHub puisque les labels n'existent pas encore. L'issue sera créée sans labels → la GitHub Action de triage ne se déclenchera PAS (elle est filtrée sur `contains(github.event.issue.labels.*.name, 'feedback')`). Pas de drame, juste un feedback qui dort sans triage IA tant que les labels ne sont pas créés.

**Conditions de déblocage** :
1. ❌ PR `feat/feedback-drawer` mergée sur main.

**Documents liés** :
- `.github/labels.yml` — déclaration versionnée des labels
- `.github/workflows/sync-labels.yml` — workflow d'auto-sync
- `.ai/V7/FEEDBACK_DRAWER.md` — plan complet
- `.ai/thought_log.md` — entrée 2026-05-05 (drawer feedback)

---

### [V8/Compare] CSR + CSR ATH (re-implémentation)

**Noté le** : 2026-05-10 | **Priorité** : 🔵 V8 — reportée

**Contexte** : retiré de la page Face-à-face le 2026-05-10 (commit `revert(compare): supprime CSR + CSR ATH`).

**Pourquoi retiré** : appel live à `skill.svc.halowaypoint.com/hi/playlist/{id}/csrs` ne fonctionne pas pour les joueurs autres que celui logué (l'endpoint Waypoint scope la lecture du CSR au token courant). SpartanRecord contourne ça via un cron Firebase + un autocode privé qui pré-cache via un compte de service ; on n'a pas l'équivalent.

**Stratégie v8** : reproduire le modèle SpartanRecord côté Go.
- Cron background (1×/jour) qui appelle Waypoint pour la liste fermée des gamertags trackés (pool joueurs LevelUp).
- Écriture dans `stats.duckdb` table `match_skill_rank` avec `rating_type='CSR'` (la colonne existe déjà, juste personne ne la remplit côté sync).
- À l'affichage de Compare : lecture DuckDB locale (< 50ms, jamais d'appel live).
- Restaurer les champs `CSRCurrent`, `CSRBest` côté domaine + sous-requêtes dans `compare_repo.go::GetPlayerATH/GetPlayerATHFor` + métriques `csr_current`/`csr_best` dans `compare_service.go::buildMetrics`.
- Côté frontend : restaurer `csr_current`/`csr_best` dans `i18n.ts` et `ComparePage.tsx::CATEGORY_KEYS.bilan`.

**À investiguer** : l'endpoint Waypoint accepte-t-il une lecture en service-account (compte 343i partner) ou faut-il rester sur le scope user ? Si scope user uniquement → cron tourne avec les tokens du joueur logué pour récupérer les CSR de ses coéquipiers.

---

### [Multi-titre] Couche canonique `weapon_family` cross-titres

**Noté le** : 2026-04-26 | **Priorité** : Basse — bloqué par arrivée d'un second titre réel

**Contexte** : Plan complet documenté dans `.ai/PLAN_WEAPON_FAMILY_CANONICAL.md` (référentiel `weapon_families` global + colonne `family_key` sur `weapon_labels` par-titre + TOML source-de-vérité). L'audit `.ai/AUDIT_WEAPONS_2026-04-25.md` a validé la faisabilité sur HI : 42 weapon_id seedés, 88 % mappables vers ~17 familles canoniques, ~32 familles totales prévues pour couvrir Halo CE→Infinite. Effort estimé : 2.5–3j en 3 phases (référentiel global + mapping HI + adapter/endpoint).

**Ce qui doit être fait** :
1. Phase 1 — créer `data/warehouse/canonical_metadata.duckdb` avec tables `weapon_families` + `weapon_family_translations` ; créer `config/canonical/weapon_families.toml` (~32 familles) ; script `tools/seed-weapon-families.go`.
2. Phase 2 — `ALTER TABLE weapon_labels ADD COLUMN family_key VARCHAR` côté HI ; créer `config/titles/halo_infinite/mappings/weapon_families.toml` (~37 lignes) ; seeder via `tools/seed-weapon-families-mapping.go`.
3. Phase 3 — étendre `TitleSemanticAdapter` avec `WeaponFamilies()` + `WeaponFamilyOf(weaponID)` ; handler `/api/v1/weapon-families` derrière flag `WEAPON_FAMILIES_API_ENABLED=false`.

**Ajustements vs plan d'origine** (cf. AUDIT §7.1) :
- compléter l'annexe §10 du plan avec 6 familles HI manquantes (`shock_rifle`, `stalker_rifle`, `heatwave`, `sentinel_beam`, `ravager`, `mutilator`) → passe de 26 à ~32 familles ;
- expliciter `family_key = NULL` comme valide pour les sentinelles (Grenade/Melee/Vehicle) et easter-eggs ;
- relever le seuil de couverture du test CI de 60 % à 85 % (HI réel à 88 %).

**Conditions de déblocage** :
1. ✅ Phase A multi-titres terminée (commit `aaccbe12`+) ;
2. ❌ second titre réel (Halo 5, MCC, ODST…) validé en pipeline produit.

**Documents liés** :
- `.ai/PLAN_WEAPON_FAMILY_CANONICAL.md` (plan complet)
- `.ai/AUDIT_WEAPONS_2026-04-25.md` (audit du référentiel HI)

---

### [Multi-titre/O10] Store / economy tracker

**Noté le** : 2026-04-18 | **Priorité** : Basse — backlog multi-titre, hors scope Halo Infinite

**Contexte** : Opportunité O10 identifiée lors de la revue des repos externes (SpartanRecord). Non pertinente pour Halo Infinite aujourd'hui (store en fin de cycle commercial, risque d'obsolescence avant livraison). Gardée en backlog car potentiellement utile si un nouveau titre Halo dispose d'une économie de store active (cosmétiques, battle pass, rotations de boutique).

**Référence** : `.ai/go_migration_v2/HALO_EXTERNAL_OPPORTUNITIES.md` §O10

**Conditions de déblocage** :
1. Onboarding d'un nouveau titre avec économie de store active confirmée
2. Signal utilisateur explicite sur l'intérêt du tracking store pour ce titre
3. Atterrissage UI validé comme module optionnel dans `Home`, jamais comme menu prioritaire

**Périmètre si débloqué** :
- Fetcher Waypoint pour les rotations de boutique du titre concerné
- Persistance dans `metadata.duckdb` avec `title_id` comme clé de partition (déjà prévu dans l'architecture O3/O8)
- Module compact `Home` scoped au titre — pas de navigation globale
- Jamais comme sous-produit autonome hors du scope analytics de LevelUp

**Point de vigilance** : ne pas ouvrir ce chantier sur Halo Infinite même sous pression — le store y est en déclin et le risque d'obsolescence est élevé.

---

### [Multi-titre/O11] Spartan Company / social layer

**Noté le** : 2026-04-18 | **Priorité** : Basse — backlog multi-titre, hors scope Halo Infinite

**Contexte** : Opportunité O11 identifiée lors de la revue des repos externes (SpartanRecord). Non pertinente pour Halo Infinite aujourd'hui (pas de signal utilisateur, dimension groupe déjà couverte partiellement par `Squad`). Gardée en backlog car potentiellement utile si un nouveau titre Halo dispose d'une dimension clan ou groupe native établie (guildes, escouades persistantes, companies).

**Référence** : `.ai/go_migration_v2/HALO_EXTERNAL_OPPORTUNITIES.md` §O11

**Conditions de déblocage** :
1. Onboarding d'un nouveau titre avec dimension groupe native confirmée
2. Ou signal utilisateur explicite sur le besoin de gestion de groupes dans LevelUp (hors `Squad` existant)
3. Dans tous les cas : valider l'atterrissage dans `Squad` avant toute autre surface

**Périmètre si débloqué** :
- Extension de la page `Squad` existante : groupes / cohortes sauvegardées scoped au titre
- Appels Waypoint vers les endpoints `Spartan Company` ou équivalent du nouveau titre
- Jamais comme rubrique de navigation autonome ni comme sous-produit social parallèle à LevelUp
- L'architecture multi-titre (`title_id` dans `xuid_aliases`, `match_participants`, etc.) le rend naturellement extensible sans restructuration

**Point de vigilance** : toute dérive vers une surface sociale indépendante de `Squad` est à refuser — la valeur de LevelUp est analytique, pas sociale.

---

### [Migration] Cible desktop Tauri web-first, sans réécriture Rust métier

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

### Script d'analyse des kills par arme pour un match donné (v8+)

**Noté le** : 2026-03-27
**Priorité** : Basse

**Contexte** : Outil de diagnostic/exploration permettant d'analyser en détail tous les kills d'un match donné, pour un joueur donné.

**Entrée** : `match_id` + `gamertag`

**Sortie** : Tableau avec, pour chaque kill :
- `match_id`
- Paire `killer` / `victim` (gamertag ou xuid si inconnu)
- `timestamp` en format `mm:ss`
- `weapon_id` (même si inconnu / non résolu)

**Ce que ça impliquerait** :
1. Requête sur `weapon_kills` (shared_matches_v2) jointure `killer_victim_pairs` + `xuid_aliases`
2. Résolution des gamertags via `v_gamertag_lookup`
3. Conversion `timestamp_ms` → `mm:ss`
4. Affichage : script CLI + éventuellement widget UI dans la page d'un match

**Complexité estimée** : Faible (données déjà disponibles dans `weapon_kills` + vues v6)

**Priorité** : Basse — outil de debug / exploration, non bloquant pour les features v7

---

### Kills environnementaux — catégorie dédiée (v8++)

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

### [coach/prestige] V3 — Squad coach

**Idée** : étendre le pont aux escouades. Quand le coach détecte un pattern collectif (composition d'équipe orientée combat / objectif / support), il propose un `SquadChallenge` ou un pool d'arcs calibrés sur la composition.

**Pré-requis** :
- Profil agrégé d'escouade (moyenne LUSR sur les axes par membre).
- Signal coach niveau escouade (à concevoir — `coach.GenerateInput` est aujourd'hui per-user).
- Extension de `prestige.RefreshSquadPool` pour accepter un filtre coach.

**Effort estimé** : lourd. Touche `coach`, `coach_advisor`, `prestige`, front-end squad UI.

---

### [coach/prestige] V3 — Coach négatif soft

**Idée** : autoriser le coach à signaler des **tendances dégradées** (LOWESS négative soutenue, baseline en chute) **sans culpabilisation**. Reformulation positive : "tu as l'opportunité de stabiliser X" plutôt que "tu régresses sur X".

**Pré-requis** :
- Décision produit explicite (ADR 0014 §6.1 cadre aujourd'hui le coach comme strictement positif — il faut un amendement ou une option par joueur).
- Tone guidelines pour i18n (FR + EN) qui maintiennent le cadre positif.
- A/B test ou opt-in séparé pour éviter d'imposer ce mode à tous les joueurs qui ont activé le coach proactif standard.

**Effort estimé** : moyen côté code, lourd côté produit/UX.

---

### [coach/prestige] V2.1 — Plumbing `Source` → `prestige_telemetry.source`

**Idée** : ajouter une colonne `source` dans la table `prestige_telemetry` et propager `CreateChallengeRequest.Source` jusqu'à `EmitCreated` (puis aux EmitTransition pour suivre le devenir des challenges coach).

**Bloqueur actuel** : la table `prestige_telemetry` est écrite mais jamais lue. Aucun script, aucun handler, aucun dashboard ne l'interroge. Le commentaire dans `types.go` mentionnant `analyze_prestige_tuning.py` réfère à un script qui n'existe pas — c'était une intention V1, jamais matérialisée.

**Pré-requis avant d'implémenter** :
- Construire d'abord un consommateur : soit un endpoint `GET /diag/prestige/telemetry` qui agrège (taux acceptance par source, complétion par source), soit un analyseur CLI Go (`cmd/prestige_tuning_analyze`) lisant directement la DB.
- **PAS DE PYTHON** : analyseur en Go ou DuckDB CLI direct (`duckdb stats.duckdb -c 'SELECT ...'`).
- Définir les métriques cibles : ratio accept/reject par source, taux completion coach vs user vs pilot_mode, distribution de strength des proposals acceptées.

**Sans consommateur** : ajouter la colonne maintenant = écriture pour personne, dette de schéma qui grossit. Reporter jusqu'à ce qu'un besoin analytics concret émerge.

**Effort estimé** : rapide (ALTER TABLE + 2 lignes Go) — mais l'analyseur côté consommation est moyen (1 commit endpoint diag ou CLI).

---

### [coach/prestige] V2.1 — Job GC `coach_advisor_template_gc`

**Idée** : job nocturne qui supprime les templates `source='coach_synthesized'` avec `usage_count=0` et `updated_at > 90 j`.

**Pré-requis** :
- Mesure post-livraison V2 du volume de templates synthétisés.
- Si volume reste < 200 templates synthétisés total, ne rien faire.

**Effort estimé** : rapide (1 commit avec scheduler).

---

### [coach/prestige] V2.1 — Compteurs expvar

**Idée** : ajouter des compteurs `coach_proposals_generated_total{kind,origin}`, `coach_proposals_accepted_total{kind,origin}`, `coach_proposals_completed_total` pour mesurer l'efficacité du coach proactif vs user / pilot_mode.

**Pré-requis** :
- Conformité ADR 0009 (expvar stdlib).
- Décision sur les labels (kind, origin, signal_kind ?) pour ne pas exploser la cardinalité.

**Effort estimé** : rapide.

---

### [coach/prestige] V3 — Apprentissage automatique de `synthesis_grammar.toml`

**Idée** : analyser `prestige_telemetry` pour ajuster la grammaire de synthèse. Si les templates synthétisés sur metric=X ont taux de complétion < 30 % sur 50 acceptations, retirer X de la grammaire ou réduire ses windows autorisés.

**Pré-requis** :
- Job analytics qui lit `prestige_telemetry` + `coach_proposal`.
- Mécanisme de PR automatique sur `synthesis_grammar.toml` (ou ajustement runtime via override en DB metadata).
- Validation manuelle obligatoire avant application.

**Effort estimé** : lourd. Demande infra analytics.

---

### [coach/prestige] V3 — Cross-titre arcs

**Idée** : permettre un arc qui couvre deux titres (ex. progression accuracy partagée Halo Infinite + futur titre Halo MCC ou cross-game). Aujourd'hui chaque `Arc` est lié à un `title_slug` unique.

**Pré-requis** :
- Décision produit (les joueurs ont-ils ce besoin ?).
- Refonte mineure `Arc.TitleSlug` → `Arc.TitleSlugs []string` ou table `arc_titles` séparée.
- Adapter les répartitions PP par titre (un challenge cross-titre crédite-t-il les PP sur chaque titre ?).

**Effort estimé** : moyen côté backend, lourd côté UX.

---

### [coach/prestige] V3 — Coach narrative tone

**Idée** : choix par joueur du "ton" du coach (technique / motivant / humour / neutre) influençant les `labelFR` / `labelEN` synthétisés et les réasons.

**Pré-requis** :
- Banque de templates i18n × 4 tons par signal kind.
- Setting joueur `coach_tone` (extension de `user_preferences`).

**Effort estimé** : moyen (surtout côté contenu i18n).

---

### [coach/prestige] V3 — Notifications push externes

**Idée** : émission externe (push mobile, email, Discord) des proposals coach les plus fortes. Aujourd'hui les notifications restent in-app (`player_notifications` UI uniquement).

**Pré-requis** :
- Décision produit (sécurité / vie privée).
- Infra push mobile (non disponible aujourd'hui).
- Préférences fines par catégorie de notif.

**Effort estimé** : lourd.

---

**Ordre de priorisation suggéré (coach/prestige V3)** :

1. V2.1 — Compteurs expvar (mesure → décision)
2. V2.1 — Job GC (si volume justifie)
3. V3 — Squad coach (le plus aligné avec les fondations Prestige squad existantes)
4. V3 — Coach négatif soft (demande arbitrage produit explicite)
5. Le reste — décisions au cas par cas

---

## 📊 Statistiques — Leased-Writer-Enforcement (PR 4-6 + PR 7)

| Métrique | Valeur |
|----------|--------|
| Commits leased-writer-enforcement | 8 |
| Commits PR 7 migration | 1 |
| Nouveaux tests intégration | 26 |
| Sites migrés (PR 7) | 17 |
| Lignes ajoutées | ~3100 |
| Baseline tests (preserved) | 1662 |
| Branches affectées | 2 (refactor/leased-writer-enforcement, fix/theme-consistency-tokens) |

---

## ✅ Récemment complété (référence)

| Date | Item |
|------|------|
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
