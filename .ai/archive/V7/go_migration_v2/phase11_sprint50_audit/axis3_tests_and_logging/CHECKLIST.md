# Axe 3 · CHECKLIST — Tests & logging

> Cocher au fur et à mesure. Chiffres mesurés obligatoires, pas d'estimations.

## Phase de préparation

- [ ] SHAs figés dans `SCOPE.md`
- [ ] `go test -coverprofile` exécuté, `coverage_report.txt` archivé dans le dossier de l'axe
- [ ] `npm run test -- --coverage` exécuté, `coverage-summary.json` archivé
- [ ] `npm run e2e` exécuté au moins une fois localement (ou CI job pointé)
- [ ] Template copié vers `claude_review.md` ou `chatgpt_review.md`

---

## Bloc 1 — Couverture Go (section A du template)

### 1.1 Coverage par package

- [ ] `internal/domain/` — mesurée + notée
- [ ] `internal/port/` — mesurée + notée
- [ ] `internal/service/` — mesurée + notée
- [ ] `internal/analysis/` — mesurée + notée
- [ ] `internal/api/handlers/` — ≥ 75 % vérifié
- [ ] `internal/api/middleware/` — ≥ 80 % vérifié
- [ ] `internal/platform/duckdb/` — ≥ 70 % vérifié
- [ ] `internal/platform/auth/` — mesurée
- [ ] `internal/platform/jobs/` — mesurée
- [ ] `internal/sync/` — ≥ 70 % vérifié
- [ ] `internal/migration/` — ≥ 75 % vérifié
- [ ] `internal/validation/` — ≥ 70 % vérifié
- [ ] `internal/ops/` — mesurée
- [ ] `internal/notify/` — mesurée
- [ ] `internal/config/` — mesurée
- [ ] Global ≥ 70 %

### 1.2 Zones à risque

- [ ] Fichiers à 0 % listés (hors gen/)
- [ ] Fonctions exportées non couvertes : liste
- [ ] Chemins d'erreur non testés : liste

### 1.3 Qualité des tests

- [ ] Tests table-driven généralisés (grep `[]struct{` dans `*_test.go`)
- [ ] `testutil/` présent avec fixtures DB réutilisables
- [ ] Tests d'intégration séparés par build tag ou package
- [ ] Mocks : approche unifiée (interfaces ou mockery)
- [ ] Concurrence : au moins un test `-race` passant pour le write lease
- [ ] Migrations idempotentes testées (apply × 2 sans erreur)
- [ ] Durée suite Go locale < 5 min
- [ ] Durée suite Go CI < 10 min

---

## Bloc 2 — Couverture React (section B du template)

### 2.1 Fichiers test par feature

- [ ] `CareerPage.test.tsx` — présent + cas couverts
- [ ] `ExplorerPage.test.tsx`
- [ ] `HomePage.test.tsx`
- [ ] `MatchHistoryPage.test.tsx`
- [ ] `MediaPage.test.tsx`
- [ ] `SettingsPage.test.tsx`
- [ ] `SetupPage.test.tsx`
- [ ] `SquadPage.test.tsx`
- [ ] `SynthesisPage.test.tsx`
- [ ] `MatchViewPage.test.tsx` — à créer ?
- [ ] `LastMatchPage.test.tsx` — à créer ?
- [ ] `SessionComparePage.test.tsx` — à créer ?
- [ ] `CitationsPage.test.tsx` — à créer ?
- [ ] `TimeseriesPage.test.tsx` — à créer ?
- [ ] `ChangelogPage.test.tsx` — à créer ?

### 2.2 Qualité

- [ ] Testing Library + Vitest configurés
- [ ] MSW (Mock Service Worker) ou équivalent pour simuler API
- [ ] Tests a11y via axe-core sur au moins les pages racines
- [ ] Snapshots : aucun > 200 lignes
- [ ] Couverture globale ≥ 60 % (seuil à affiner)

---

## Bloc 3 — Non-régression (section C du template)

### 3.1 Scénarios critiques Python → équivalent Go/React

Pour chaque scénario, chercher le test Python puis vérifier l'équivalent côté Go ou React :

- [ ] Sync delta joueur existant
- [ ] Backfill LUSR + CSR match classé
- [ ] Backfill citations PvE
- [ ] Backfill weapon_kills + bit 18
- [ ] Détection bots (`xuid LIKE 'bid(%'`)
- [ ] Write lease 2 writers concurrents
- [ ] Restore depuis backup corrompu / intégrité
- [ ] Home computation battle pass
- [ ] Career rolling LUSR window
- [ ] Session compare overlap
- [ ] Teammates trio f2_xuid optionnel
- [ ] Comeback badges narrative
- [ ] i18n Accept-Language résolution + fallback
- [ ] Media association ↔ match (incl. DST v6.2)
- [ ] PersonalScores awards
- [ ] Schéma v6 player init depuis zéro
- [ ] Schéma v6 shared init depuis zéro
- [ ] Schéma PvE init

### 3.2 Golden values

- [ ] Performance score : golden Python extraite, golden Go alignée, diff = 0
- [ ] LUSR : idem
- [ ] CSR : idem
- [ ] Sessions : idem
- [ ] Citations : idem (tous les mappings médailles testés)
- [ ] Killer/victim : idem
- [ ] Weapon parser : idem

### 3.3 E2E Playwright

- [ ] 15 specs listées + état vert/rouge + CI job
- [ ] Onboarding auth → home vert en CI

---

## Bloc 4 — Logging Go (section D du template)

### 4.1 Démarrage / shutdown

- [ ] Démarrage serveur logué (INFO, version, config résolue)
- [ ] Shutdown gracieux logué
- [ ] Erreurs de démarrage loguées en ERROR avant exit

### 4.2 Requêtes HTTP

- [ ] Middleware logue début/fin requête avec request ID + statut + durée
- [ ] Request ID généré si absent, propagé sinon
- [ ] Request ID ajouté à tous les logs du handler (via context)

### 4.3 Sync & backfill

- [ ] Début sync joueur : INFO (gamertag, scope)
- [ ] Fin sync joueur : INFO (matchs traités, durée)
- [ ] Erreur sync : ERROR avec stack/context
- [ ] Début backfill : INFO (flags actifs)
- [ ] Progression backfill : DEBUG ou INFO par batch
- [ ] Fin backfill : INFO (succès/erreurs par flag)

### 4.4 Writes DB

- [ ] `insertRegistry`, `insertParticipants`, `insertMedals`, etc. : log DEBUG succès + ERROR échec
- [ ] Conflits INSERT loggués (niveau WARN ?)

### 4.5 Write lease

- [ ] Acquisition lease : DEBUG ou INFO
- [ ] Libération : idem
- [ ] Timeout : WARN ou ERROR (selon criticité)

### 4.6 API Halo (provider)

- [ ] Appel provider : DEBUG (URL, params) sans secrets
- [ ] Erreur provider : ERROR avec taxonomie (`HALO_PROVIDER_ERROR_TAXONOMY.md`)
- [ ] Retry / backoff : DEBUG/INFO

### 4.7 Migration

- [ ] Chaque step appliqué : INFO (name, durée)
- [ ] Step déjà appliqué : DEBUG
- [ ] Erreur step : ERROR + rollback loggué

### 4.8 Auth / device code flow

- [ ] Init : INFO (device_code créé, expires_in)
- [ ] Poll : DEBUG
- [ ] Complete : INFO (user_id résolu)
- [ ] Erreurs auth : ERROR sans fuite de token

### 4.9 Format & structuration

- [ ] Logger `slog` (ou équiv.) utilisé partout (pas de `fmt.Println` résiduel)
- [ ] Format JSON en prod (configurable)
- [ ] Niveau configurable via env
- [ ] Pas de secrets dans les logs (tokens, xuid complet ?)

---

## Bloc 5 — Logging React (section D.4 du template)

- [ ] `grep -r 'console\.log' apps/web/src` : 0 hors mode dev
- [ ] `grep -r 'console\.error' apps/web/src` : vérifié que chaque occurrence remonte à une error boundary ou télémétrie
- [ ] Error boundary sur routes racines
- [ ] Tracking requête API : request ID round-trip si backend l'émet

---

## Bloc 6 — CI & qualité d'exécution (section E du template)

- [ ] Job `go-coverage` CI activé, CGO on
- [ ] Seuil CI à 70 % effectif (pas 50 % legacy)
- [ ] Ratchet positif configuré (baseline committée)
- [ ] Temps Go local < 5 min, CI < 10 min
- [ ] Temps React local < 2 min
- [ ] Playwright CI dédié (`e2e-react`)
- [ ] Flakiness : liste les tests ayant flap > 1 fois sur 10 runs

---

## Bloc 7 — Plan d'action (section G du template)

- [ ] Top 10 trous de couverture identifiés
- [ ] Chaque trou : zone, effort S/M/L, impact

---

## Validation finale de l'axe

- [ ] Template rempli à 100%
- [ ] Récap §F cohérent avec sections A-E
- [ ] `coverage_report.txt` et `coverage-summary.json` archivés dans le dossier de l'axe
- [ ] Commit sur branche `phase11/sprint50-triple-audit`
