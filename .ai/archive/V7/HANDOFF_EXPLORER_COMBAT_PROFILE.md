# HANDOFF — Explorer profil de combat (identité + médailles + adornment)

> Date : 2026-05-31
> Branche : `fix/explorer-combat-profile-identity` (créée depuis `fix/sync-combat-completion-persist`)
> Plan source : `.ai/PLAN_EXPLORER_COMBAT_PROFILE.md`
> Auteur du code : agent IA (Claude).

## État (tests automatisés VERTS)

- **Go** : `gofmt -l` propre ; `go vet` (service/duckdb/api/domain) exit 0 ; `go build ./...` (CGO) exit 0 ; `go test ./internal/service` → `ok` ; `go test -tags=integration ./internal/platform/duckdb` (suite complète, hors 1 FAIL pré-existant cf. plus bas) → `ok` ; `go test ./internal/api/...` → `ok`. Tests médailles re-joués `-count=2..3` : stables.
- **Front** : `vitest` banner + profile card → **9/9** (EXIT 0) ; `eslint` 0 sur les fichiers touchés.
- **typecheck (`tsc -b`)** : 7 erreurs UNIQUEMENT dans `src/features/media/CoverFlowModal.tsx` (`Cannot find module 'hls.js'` + `any` implicites) — **hors scope, cause = environnement** : `hls.js` est déclaré dans `package.json` (`^1.6.16`) mais `node_modules/hls.js` n'est pas installé ici. `npm install` (ou `npm ci`) résout. Aucun de mes fichiers n'a d'erreur typecheck.
- **Reste : validation LIVE** (endpoint Explorer + visuel), potentiellement bloquée par Zscaler.

**Périmètre du diff** : l'arbre de travail ne contient QUE les changements de cette tâche (vérifié). Commit en attente d'autorisation.

## Ce qui est fait

**Phase 1 — Sérialisation DTO de l'identité (cœur du bug).** `target_profile.identity` sérialise désormais en DTO snake_case (`banner_image_url`, `emblem_image_url`, `career_rank.{rank_title,adornment_image_url}`) comme la Home, au lieu de la struct brute PascalCase que le front ne savait pas lire.

**Phase 2 — Fallback `citation_mappings` médailles (DRY).** Les libellés absents/vides de `medal_definitions` sont rattrapés par `citation_mappings.citation_name_display` dans les 3 résolveurs (avant : seul match-view l'avait). Requête centralisée dans un helper unique (règle DRY ≤2).

**Phase 3.5 — Adornment dans le banner Explorer.** Rendu en miroir de la Home, scopé à la zone hero pour ne pas chevaucher la barre XP.

**Phase 3.6 — Bannière de repli déterministe (option C « pool local », choisie par l'utilisateur).** Si une cible non-locale n'a ni bannière ni backdrop, on lui attribue une nameplate piochée de façon déterministe par xuid (hash FNV-32a) dans le pool dédupliqué des `banner_image_url` des joueurs suivis. Résolution **paresseuse** (le pool n'est construit que dans ce cas rare). Limite assumée : nameplate « empruntée » à un autre joueur ; vide si aucune bannière locale.

## Fichiers modifiés / créés

Backend (`apps/go-api/`) :
- `internal/domain/explorer.go` — `ExplorerTargetProfile.Identity` : `*HomeSpartanIdentityRow` → `*HomeSpartanIdentity`.
- `internal/service/explorer_service.go` — import `games/mappings` + `hash/fnv` ; champs `Ranks *mappings.RankCatalog` + `LocalBannerPool func(ctx) []string` dans `ExplorerTargetProfileDeps` ; `fetchTargetIdentity` → `fetchTargetIdentityRaw` ; conversion `analysis.BuildSpartanIdentity(...)` ; Phase 3.6 : `applyBannerFallbacks` + `hasBanner` + `pickDeterministicBanner`.
- `internal/api/registry_pages.go` — import `games/mappings` ; câblage `Ranks` + `LocalBannerPool` dans `ExplorerCtxWithAuth` ; méthode `newExplorerLocalBannerPool`.
- `internal/platform/duckdb/medal_citation_fallback.go` — **NOUVEAU** : const `medalCitationFallbackQuery` + helper `lookupMedalCitationLabels` (source unique du fallback) + log DEBUG.
- `internal/platform/duckdb/match_view_repo_medals.go` — `lookupMedalMeta` délègue au helper (requête inline supprimée).
- `internal/platform/duckdb/medal_definitions_repo.go` — `LookupByIDs` : ajout du fallback (Explorer top_medals + Squad).
- `internal/platform/duckdb/home_repo_medals_citations.go` — `resolveMedalLabels` : ajout du fallback (tuile de match Home).
- `internal/service/explorer_service_test.go` — `.RankNumber` → `.CareerRank.RankNumber` ; nouveaux `TestExplorerService_TargetProfile_IdentitySerializesAsDTO`, `_DeterministicBannerFallback`, `TestPickDeterministicBanner`.
- `internal/platform/duckdb/medal_definitions_repo_test.go` — **NOUVEAU** (`//go:build integration`).
- `internal/platform/duckdb/medal_citation_fallback_test.go` — **NOUVEAU** (`//go:build integration`) : helper + 3 résolveurs.

Frontend (`apps/web/`) :
- `src/features/explorer/ExplorerTargetIdentityBanner.tsx` — rendu adornment (modifié).
- `src/features/explorer/ExplorerTargetIdentityBanner.test.tsx` — **NOUVEAU**.

Docs : `.ai/thought_log.md` (modifié), `.ai/HANDOFF_EXPLORER_COMBAT_PROFILE.md` (ce fichier).

## Logging (couverture)

- `lookupMedalCitationLabels` (helper) → DEBUG `medal_citation_fallback` (`requested`, `resolved`) → **`logs/duckdb.log`** (routage par PC d'appel, cf. `observability/logging/module.go`). Diagnostic du symptôme « médailles en ID nu » : `resolved=0` avec `requested>0` = citation_mappings vide/absente. À voir sous `LEVELUP_LOG_LEVEL=debug`.
- `newExplorerLocalBannerPool` → DEBUG `explorer_banner_pool_built` (`size`) → `logs/http.log` (package api).
- `applyBannerFallbacks` → DEBUG `explorer_target_banner_pool_fallback` (`xuid`) quand le repli pool est appliqué.
- Chemin identité déjà couvert : WARN `explorer_target_identity_live_failed`, `explorer_target_live_budget_exceeded` ; DEBUG `explorer_target_identity_unavailable`.

## Tests (couverture des 3 résolveurs médailles)

`medal_citation_fallback_test.go` (NOUVEAU, `//go:build integration`) :
- `TestLookupMedalCitationLabels_Helper` — helper partagé + contrat nil-safe (db nil / ids vide → map vide).
- `TestResolveMedalLabels_CitationFallback` — tuile de match **Home** (fallback nouvellement ajouté).
- `TestLookupMedalMeta_CitationFallback` — vue **Match** (verrouille le refactor : le fallback préexistant marche toujours après extraction du helper).

`medal_definitions_repo_test.go` — `TestMedalDefinitionsRepo_LookupByIDs_CitationFallback` (**Explorer/Squad**, dataset hétérogène + cas libellé vide).

Service : `TestExplorerService_TargetProfile_IdentitySerializesAsDTO` (le test qui aurait attrapé le bug), `_DeterministicBannerFallback` (Phase 3.6), `TestPickDeterministicBanner`.

Front : `ExplorerTargetIdentityBanner.test.tsx` (emblème/bannière/rang/adornment + dégradé identity=null + absence adornment).

## FAIL pré-existant à traiter séparément (PAS cette tâche)

`go test -tags=integration ./internal/platform/duckdb` fait sortir **`TestNoUnauthorizedSharedSocialMention`** (sentinel ADR 0021). **Prouvé pré-existant** : `git stash` de mon travail → run sur la base nue → même FAIL. Il pointe `cmd/backfill-media-hls/main.go` + `internal/ops/media_hls.go` (chantier HLS, commits `3abd4537` / `20266047`). **Action** : ajouter ces 2 fichiers à `sharedSocialFilesWhitelist` dans `internal/platform/duckdb/no_attach_on_social_test.go` (avec description justifiée). Hors périmètre du profil de combat.

## Notes de transparence (erreurs de l'agent corrigées)

1. **Pas de « bug curseur »** (fausse piste, ne pas chercher de trace) : 2 tests rouges initiaux venaient d'un seed de test FR-first erroné (un faux libellé inventé) confondu avec un problème de curseur SQL. Corrigé : le seed utilise `name_en='Killjoy'` seul ; les `_ = rows.Close()` ajoutés à tort ont été retirés (code revenu au `defer rows.Close()` standard).
2. **`git stash pop` imprudent** sur un arbre untracked non commité → 2 fichiers tronqués + un `.orig` (réparés). Ne pas reproduire.

## Validation LIVE restante (à faire sur la machine de dev avec config + tokens)

> ⚠️ NON FAISABLE dans l'environnement où le code a été écrit : `db_profiles.json`, `data/players/`, `data/auth/watcher_tokens/` absents + ports HTTP qui timeoutent (Zscaler / pas de serveur). À exécuter sur ta machine habituelle.

**Checklist rapide (à cocher sur l'autre machine) :**

- [ ] `make go-api-dev` (ou équivalent) → serveur up sur `:8000`.
- [ ] **Probe API brute joueur inconnu** (répond à « récupère-t-on le Spartan ID de Nilton410 ? ») :
  ```bash
  go run ./apps/go-api/cmd/diag_appearance JGtm 2535427927026623
  # Attendu : /customization/appearance → 403 ; /customization?view=public → 200 (ServiceTag "MELG", Emblem, Backdrop)
  ```
- [ ] **Endpoint Explorer, cible NON-locale** (curl avec session/tokens) :
  ```bash
  curl -s -X POST http://localhost:8000/api/v1/players/JGtm/pages/explorer/player-query \
    -H 'Content-Type: application/json' -d '{"target_gamertag":"Nilton410"}' | jq '.target_profile.identity'
  # Attendu (clés SNAKE_CASE — c'est le fix Phase 1) :
  #   spartan_id:"MELG", emblem_image_url, banner_image_url,
  #   career_rank:{ rank_title, adornment_image_url, current_xp, xp_for_next_rank }
  # ❌ Régression si on voit des clés PascalCase (SpartanID, BannerImageURL, RankNumber)
  ```
- [ ] **Cible LOCALE** (un joueur suivi) : même curl avec son gamertag → identité depuis la DB locale.
- [ ] **Logs** : `logs/service.log` → `FetchLiveIdentity résolu` (has_service_tag=true) ; `logs/duckdb.log` → `medal_citation_fallback` (requested/resolved) sous `LEVELUP_LOG_LEVEL=debug`.
- [ ] **App (visuel)** : Explorer mode Joueur → emblème + bannière + rang + **adornment** ; médailles avec **noms + tooltips** (plus d'IDs nus) sur Explorer ET tuile de match Home.
- [ ] **Médailles data** : `go run apps/go-api/cmd/inspect_bp/main.go data/warehouse/metadata.duckdb` → comparer `medal_definitions` vs `citation_mappings`.
- [ ] **(front) `npm install` puis `npm run typecheck`** → confirmer 0 erreur (l'erreur `hls.js` de cet env disparaît une fois `node_modules` installé).

### Cas « joueur tiers rencontré » (ex. Nilton410) — flow + état de validation

Pour une cible **non suivie** (absente de `db_profiles.json`) :

- **Pré-requis dur** : la cible doit exister dans `shared.v_gamertag_lookup` (cascade `xuid_aliases ∪ match_participants`) = avoir croisé un joueur suivi dans ≥1 match. `ExplorerRepo.ResolveXUIDByGamertag` (ILIKE, bots exclus) ; sinon **erreur "gamertag inconnu" et l'endpoint échoue** (pas de profil). Donc « inconnu jamais croisé » ≠ supporté ; « adversaire déjà rencontré » = supporté.
- **Chemin** : `LocalIdentity` → nil (pas suivi) → bascule **live** `CareerLiveService.FetchLiveIdentity(xuid)` (`career_live_target.go`).
- **Customisation tierce = vue publique** : `GetSpartanCustomization` → 403 sur `/customization/appearance` (player-gated) → fallback `/customization?view=public` (HTTP 200, même bloc Appearance pour tout xuid) → **SpartanID (service tag) + emblem + backdrop + banner** (banner via `ResolveNameplateURL` si absent).

**VALIDATION RUNTIME DÉJÀ FAITE (thought_log 2026-05-30, lignes 600-627)** — exactement `JGtm → Nilton410` (xuid `2535427927026623`) via `cmd/diag_appearance` + curl serveur `:8000` :
- `/customization/appearance` → 403 ; `/customization?view=public` → **200** (ServiceTag "MELG", Emblem, Backdrop).
- Endpoint Explorer → HTTP 200, `target_profile.identity` = **SpartanID "MELG" + emblem + backdrop + banner + career rank 272 « Hero Gold »** ; `top_medals`, `season_csrs`, `matches_per_season` (S13:22), `time_played_seconds`. **Carte complète pour un joueur sans DB locale.**
- `logs/service.log` : ligne INFO `FetchLiveIdentity résolu` confirmée. HighestCSR/LUSR nuls (peaks du propriétaire, omis pour un tiers — voulu).
- ⇒ **`careerranks` n'est PAS gated** dans les faits (le rang 272 est revenu) — contrairement à ce que le commentaire d'en-tête de `career_live_target.go` suppose en cas de 403. Le code dégrade proprement (`career_rank=null`) SI un jour c'était gaté, mais empiriquement le rang s'affiche.

**Ce que MON changement (Phase 1) modifie vs cette validation** : uniquement la **couche de sérialisation** — la même donnée passe désormais par `analysis.BuildSpartanIdentity(raw, locale, ranks)` → DTO snake_case (`career_rank` imbriqué) au lieu de la struct brute PascalCase. Couvert en unitaire par `TestExplorerService_TargetProfile_IdentitySerializesAsDTO`. Le **flux de récupération live (le « est-ce qu'on récupère bien le Spartan ID de Nilton410 »)** est inchangé par ma PR — il était déjà fonctionnel.

**Sur la « bannière aléatoire »** : ⚠️ malentendu à lever. La vue publique fournit quasi toujours backdrop et/ou vraie nameplate → Nilton410 reçoit **SA vraie bannière** (cf. validation ci-dessus), PAS une aléatoire. Mon pool déterministe (Phase 3.6) n'est qu'un **filet ultime** : il ne se déclenche QUE si banner **ET** backdrop sont vides (cible sans aucun visuel public — rare). Design option C = priorité **vraie bannière > backdrop > pool**. Donc pour Nilton410 spécifiquement, le pool ne s'active pas (et c'est le comportement voulu).

**Re-validation live NON rejouable dans CET environnement** : `db_profiles.json`, `data/players/`, `data/auth/watcher_tokens/` sont **absents** ici (machine sans données runtime) ; les ports `:8000/:8080/:3000` timeoutent. `cmd/diag_appearance JGtm 2535427927026623` échoue donc sur `owner introuvable dans db_profiles`. La re-preuve live doit être faite sur la machine de dev habituelle (avec config + tokens), commande prête ci-dessus.

## Commandes pour re-valider

Depuis `apps/go-api/` (CGO, `CGO_ENABLED=1`, `CC=gcc`) :

```bash
go build ./...
go test ./internal/service/ -run TestExplorerService
go test -tags=integration ./internal/platform/duckdb/ \
  -run "TestLookupMedalCitationLabels_Helper|TestResolveMedalLabels_CitationFallback|TestLookupMedalMeta_CitationFallback|TestMedalDefinitionsRepo_LookupByIDs_CitationFallback" -count=1
# Suite complète (le FAIL TestNoUnauthorizedSharedSocialMention est pré-existant, cf. ci-dessus) :
go test -tags=integration ./internal/platform/duckdb/ -count=1
```

Depuis `apps/web/` (hors sandbox) :

```bash
# FAIT : 9/9 verts
npx vitest run src/features/explorer/ExplorerTargetIdentityBanner.test.tsx \
  src/features/explorer/ExplorerTargetProfileCard.test.tsx
# FAIT : exit 0 sur les fichiers touchés
npx eslint src/features/explorer/ExplorerTargetIdentityBanner.tsx \
  src/features/explorer/ExplorerTargetIdentityBanner.test.tsx
# npm install   # hls.js (^1.6.16) déclaré mais node_modules absent dans cet env
# npm run typecheck   # après install : 0 erreur attendue côté Explorer
```
