# PLAN — Profil de combat Explorer : cosmétiques + médailles

> Statut : En cours (plan validé, implémentation à venir)
> Date : 2026-05-31
> Branche cible : `fix/explorer-combat-profile-identity` (à créer depuis la branche courante)

## Critère de succès

Sur la page Explorer, l'encart « profil de combat » (`ExplorerTargetProfileCard`) affiche :
- pour une cible **suivie localement** : emblème + bannière (nameplate) + rang carrière + XP + adornment ;
- pour une cible **non-locale avec tokens Halo** : idem via fetch live ;
- pour une cible **non-locale sans bannière résolue** : une bannière de repli déterministe par joueur ;
- les **médailles** avec nom + description (tooltip), plus seulement l'ID.

---

## Contexte

L'encart « profil de combat » Explorer n'affiche **jamais** emblème / nameplate / bannière /
adornment (ni cible locale, ni non-locale), et les médailles n'affichent qu'un ID (`#1234`).
Pourtant ces données s'affichent ailleurs (Home, match-view), et 5 tentatives la veille
« prouvaient en curl » que la donnée était récupérable.

### Cause racine #1 — Sérialisation JSON d'identité (cosmétiques)

C'est l'explication des 5 échecs : un bug **invisible en curl** mais fatal au front.

- `ExplorerTargetProfile.Identity` est typé `*HomeSpartanIdentityRow` — struct **brute**
  ([domain/explorer.go:105](apps/go-api/internal/domain/explorer.go#L105)) **sans aucun tag JSON**
  ([domain/home.go:42-57](apps/go-api/internal/domain/home.go#L42)), à champs de rang **plats**.
- Le handler la sérialise telle quelle ([handlers/explorer.go:66](apps/go-api/internal/api/handlers/explorer.go#L66))
  → clés **PascalCase** : `{"BannerImageURL":…,"EmblemImageURL":…,"SpartanID":…,"RankName":…}`.
- Le front lit du **snake_case + `career_rank` imbriqué** : `identity.banner_image_url`,
  `identity.emblem_image_url`, `identity.career_rank.rank_title`
  ([ExplorerTargetIdentityBanner.tsx:41-44](apps/web/src/features/explorer/ExplorerTargetIdentityBanner.tsx#L41)).
  Le type TS déclare déjà `identity: HomeSpartanIdentity` (DTO snake_case,
  [types.ts:1230](apps/web/src/lib/api/types.ts#L1230)) — **le contrat FE/BE diverge**.

Conséquence : `identity` est non-null (pas de placeholder) mais **tous ses champs sont `undefined`**
→ rien ne s'affiche, jamais, pour local comme non-local. En curl, un humain lit les valeurs
(en PascalCase) et conclut « ça marche » ; React lisant le snake_case ne voit rien.

La page **Home** marche car elle convertit la brute en DTO via
`analysis.BuildSpartanIdentity(raw, locale, ranks)`
([home_service.go:372](apps/go-api/internal/service/home_service.go#L372) ;
impl [home_kpis.go:184](apps/go-api/internal/analysis/home_kpis.go#L184)) — qui produit le
snake_case **et** le `career_rank` imbriqué (avec `adornment_image_url`).

### Cause racine #2 — Fallback `citation_mappings` médailles manquant

Le résolveur **match-view** `lookupMedalMeta` complète les IDs absents de `medal_definitions`
via `citation_mappings.citation_name_display`
([match_view_repo_medals.go:111-132](apps/go-api/internal/platform/duckdb/match_view_repo_medals.go#L111))
→ noms OK. Les deux autres résolveurs **n'ont pas** ce fallback → labels vides :
- `MedalDefinitionsRepo.LookupByIDs` (Explorer + Squad)
  ([medal_definitions_repo.go:45](apps/go-api/internal/platform/duckdb/medal_definitions_repo.go#L45)) ;
- `resolveMedalLabels` (tuile de match Home)
  ([home_repo_medals_citations.go:165](apps/go-api/internal/platform/duckdb/home_repo_medals_citations.go#L165)).

Symptôme confirmé par l'utilisateur : noms OK en match-view, vides sur Explorer + tuile Home.

### Cause racine #3 — Parité visuelle + bannière non-locale

L'`ExplorerTargetIdentityBanner` (version compacte) ne rend pas l'adornment, contrairement au
banner Home ([HomeSpartanIdentityBanner.tsx:81-93](apps/web/src/features/home/HomeSpartanIdentityBanner.tsx#L81)).
Pour une cible non-locale, l'utilisateur veut une **bannière aléatoire** quand la vraie n'est pas
disponible. (NB : la logique random de SpartanRecord est **côté serveur Autocode/HaloDotAPI**,
non portable. Le repo possède déjà `ResolveNameplateURL`,
[spartan_nameplate_resolver.go](apps/go-api/internal/sync/spartan_nameplate_resolver.go), et le
fetch vue-publique tiers, [halo_client_career.go:64](apps/go-api/internal/sync/halo_client_career.go#L64).)

---

## Approche — 3 phases (effort croissant, chacune livrable)

### Phase 1 — Sérialiser le DTO d'identité (cœur du bug) — effort : rapide

1. `domain/explorer.go` : `Identity *HomeSpartanIdentityRow` → `Identity *HomeSpartanIdentity`.
2. `service/explorer_service.go` :
   - `fetchTargetIdentity` produit toujours la brute + `applyBannerFallback` (renommer `…Raw`).
   - Dans `buildTargetProfile`, convertir :
     `analysis.BuildSpartanIdentity(raw, ctxkeys.Locale(ctx), s.deps.Ranks)` (réutilise l'existant ;
     `nil`-safe : si `Ranks==nil`, fallback `RankName` déjà géré).
   - Ajouter `Ranks *mappings.RankCatalog` à `ExplorerTargetProfileDeps`.
3. `api/registry_pages.go` : câbler `Ranks:` depuis l'adapter sémantique du titre
   (même source que HomeService) dans `WithTargetProfileProviders`
   ([registry_pages.go:275-286](apps/go-api/internal/api/registry_pages.go#L275)).
4. Front : aucun changement de type (TS déclare déjà `HomeSpartanIdentity`).

Livre la plainte centrale : cible **locale** (données DB, sans auth) + cible **non-locale avec auth**
→ emblème + bannière + rang + XP + adornment dans le JSON, lus par le front.

### Phase 2 — Fallback `citation_mappings` médailles (DRY) — effort : rapide

Mutualiser le fallback déjà présent en match-view (helper `lookupLabelsByID` + requête
`citation_mappings`) dans les deux résolveurs qui en manquent :
- `medal_definitions_repo.go` `LookupByIDs` → corrige Explorer `top_medals` + Squad ;
- `home_repo_medals_citations.go` `resolveMedalLabels` → corrige la tuile de match Home.

Réutiliser le helper existant (règle DRY ≤2 du CLAUDE.md : pas de 3e copie de requête).

### Phase 3 — Parité visuelle + bannière aléatoire — effort : moyen

5. `ExplorerTargetIdentityBanner.tsx` : rendre l'adornment depuis
   `identity.career_rank.adornment_image_url`, en miroir du bloc Home. « Nameplate » = fond de
   bannière (déjà rendu dès que `banner_image_url` est présent ; pas de champ distinct). Le
   drop-shadow `rgba(...)` est une couleur **structurelle** (toléré, règle 20 CLAUDE.md) ; recopier
   tel quel pour cohérence visuelle avec Home.
6. Bannière aléatoire (cible non-locale sans bannière) : étendre `applyBannerFallback` côté service —
   si `BannerImageURL` ET `BackdropImageURL` vides, choisir une nameplate de façon **déterministe
   par xuid** (hash → index, pas de random non-déterministe) dans un catalogue de chemins nameplate
   valides. Catalogue **title-aware** (pas codé en dur sur le slug) — point de décision : source du
   catalogue (assets nameplate déjà connus, ou liste curée par titre).

---

## Fichiers à modifier

Backend
- `apps/go-api/internal/domain/explorer.go` — type `Identity`.
- `apps/go-api/internal/service/explorer_service.go` — conversion DTO + champ `Ranks`.
- `apps/go-api/internal/api/registry_pages.go` — câblage `Ranks`.
- `apps/go-api/internal/platform/duckdb/medal_definitions_repo.go` — fallback citation_mappings.
- `apps/go-api/internal/platform/duckdb/home_repo_medals_citations.go` — fallback citation_mappings.

Frontend
- `apps/web/src/features/explorer/ExplorerTargetIdentityBanner.tsx` — rendu adornment.

## Architecture (revue plan-review)

- Algo pur (mapping identité) : `internal/analysis` (`BuildSpartanIdentity`, réutilisé). OK.
- Types DTO : `internal/domain`. OK.
- Orchestration : `internal/service`. OK.
- Accès DuckDB : `internal/platform/duckdb` (repos médailles). OK, pas de SQL dans le service/handler.
- Title-aware : `RankCatalog` via adapter sémantique ; image médaille `/static/medals/{titleSlug}/…`
  déjà title-aware ; catalogue bannière Phase 3 à garder title-aware.
- Pas de nouveau `port`, pas de nouveau `FieldKey`, pas de nouvelle route, pas de string i18n
  (adornment = image). Logging : conserver les `slog.*Context` existants ; pas de `fmt.Println`.

## Tests

- `service/explorer_service_test.go` : `target_profile.identity` sérialise en snake_case + `career_rank`
  non-nil (test qui aurait attrapé le bug) ; cas dégradé `Ranks==nil` → titres via `RankName` ;
  cas non-local sans auth → `identity==nil` → le front rend le placeholder (pas de régression).
- `platform/duckdb/medal_definitions_repo_test.go` : dataset hétérogène (un ID dans
  `medal_definitions`, un seulement dans `citation_mappings`) → les deux résolus.
- Front : test rendu emblème/bannière/adornment de `ExplorerTargetIdentityBanner`
  (mock du DTO `HomeSpartanIdentity`).

## Vérification (bout-en-bout)

1. Build API (toolchain CGO msys64 + `CGO_ENABLED=1`), `POST`
   `/api/v1/players/{slug}/pages/explorer/player-query` pour une cible locale puis non-locale
   (avec tokens) → vérifier `target_profile.identity` = `banner_image_url`, `emblem_image_url`,
   `career_rank.rank_title`, `career_rank.adornment_image_url`.
2. App lancée (skill `run`/`verify`) : Explorer mode Joueur → emblème + bannière + rang + adornment ;
   médailles avec noms + tooltips.
3. Données médailles : `go run apps/go-api/cmd/inspect_bp/main.go data/warehouse/metadata.duckdb`
   → état `medal_definitions` vs `citation_mappings` (valide l'hypothèse fallback ; envisager
   `refresh-metadata` si `medal_definitions` doit aussi être peuplé).
4. Tests : Go (CGO) sur `internal/service` + `internal/platform/duckdb` ; vitest `apps/web`
   **hors sandbox** (`dangerouslyDisableSandbox=true`) ; typecheck + eslint.
5. `.ai/thought_log.md` : entrée obligatoire avant commit (date, tâche, décision, résultats).

## Notes d'exécution

- Branche : la branche courante porte un travail non lié + des fichiers modifiés non commités —
  créer une branche dédiée sans perturber ce travail ; demander avant tout commit.
- Phasage : Phase 1 seule résout la plainte principale ; Phases 2-3 complètent médailles, adornment
  et bannière aléatoire. Point de décision restant : source du catalogue nameplate (Phase 3).
