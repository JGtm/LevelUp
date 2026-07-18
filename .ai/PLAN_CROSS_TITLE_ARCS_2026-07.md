# Plan — Arcs multi-titres : indépendance stricte par jeu (2026-07)

> **Créé le** : 2026-07-18
> **Statut** : Approuvé — **Option A (retrait `arc_titles`) confirmée par l'utilisateur le
> 2026-07-18**. Exécution = branche dédiée `feat/arcs-per-title-strict` (hors train en cours),
> à ouvrir à l'anticipation du 2e titre. Aucune décision ouverte restante.
> **Priorité** : 🟡 à anticiper avec l'arrivée du 2e titre (pas de titre en pipeline aujourd'hui).
> **Origine** : décision produit utilisateur (2026-07-18) sur le backlog `[coach/prestige] V3 — Cross-titre arcs`.
> **Supersède partiellement** : `.ai/V7/PLAN_CROSS_TITLE_ARCS_BACKEND.md` (livré 2026-06-18) — voir §2.

## 0. Décision produit gouvernante (TRANCHÉE, ne pas rouvrir)

**Les arcs, challenges, valeurs et progression sont STRICTEMENT INDÉPENDANTS par titre.**
« Cross-titre » signifie ici : le système d'arcs/prestige/coach **fonctionne pour chaque
titre séparément**, chacun accumulant ses propres arcs et ses propres valeurs — **jamais** un
élément qui se croise sur deux jeux.

**Non-objectifs explicites (interdits)** :
- ⛔ Un arc (ou challenge) unique couvrant **plusieurs titres**.
- ⛔ Un **objectif partagé** entre deux jeux (« faire X en Halo 5 ET en Halo Infinite »).
- ⛔ Toute **agrégation cross-titre** : PP proraté sur plusieurs titres, valeur/rating fusionné,
  classement/leaderboard « tous titres confondus », progression combinée.
- ⛔ Tout champ/endpoint API exposant une notion multi-titres d'arc.

**Ce qui reste autorisé et attendu** :
- ✅ Un même joueur a des arcs en Halo 5 ET des arcs en Halo Infinite — **deux ensembles
  disjoints**, chacun mesuré sur son propre titre.
- ✅ L'UI peut *juxtaposer* (afficher côte à côte) les arcs par titre — juxtaposition ≠ fusion.
- ✅ Le socle multi-titre existant (isolation par chemin ADR 0008, `PathResolver`,
  capabilities, `TitleRegistry`) porte déjà cette indépendance.

## 1. État des lieux (vérifié sur pièces, 2026-07-18)

- Un `Arc` (et un `Challenge`) porte **un seul** `title_slug` — `internal/prestige/types.go`.
- Stockage **déjà isolé par titre** : `arc` vit dans `stats.duckdb` **par joueur et par titre**
  (`data/titles/{slug}/players/{gamertag}/stats.duckdb`, `PathResolver.PlayerDBPath`) ; les
  presets dans `metadata.duckdb` **par titre**. → L'indépendance par jeu est **déjà l'état natif**.
- **MAIS** le plan livré le 2026-06-18 a ajouté une sur-structure orientée VERS l'inverse :
  - table de jointure `arc_titles(arc_id, title_slug)` **many-to-many** (« un arc peut couvrir
    N titres ») + migration `create_arc_titles_join` (schéma + backfill 1 ligne/arc) ;
  - repo `prestige.ArcTitlesRepo` (`ArcTitles`/`ArcsByTitle`) + maintien de l'invariant à la création ;
  - point d'extension `creditTitlesFor(challenge) []string` (retourne `[challenge.TitleSlug]`
    aujourd'hui, **conçu pour être surchargé** en crédit multi-titres).
  Aujourd'hui c'est **inerte** (exactement 1 ligne/arc = le titre primaire), mais la **capacité**
  multi-titres est là et contredit la décision §0.

## 2. Décision TRANCHÉE : retirer la sur-structure `arc_titles` (Option A, confirmée 2026-07-18)

**Option A (retrait) RETENUE** par l'utilisateur le 2026-07-18. L'Option B (gel) ci-dessous est
conservée pour trace du raisonnement mais n'est PAS suivie.

**Option A — Retirer la sur-structure (recommandé).**
Migration `DROP TABLE arc_titles` + suppression de `ArcTitlesRepo` et du seam `creditTitlesFor`
(revenir à `arc.title_slug` comme unique source). Les consommateurs actuels lisent déjà
`WHERE title_slug = ?` (la voie `ArcsByTitle` est un sur-ensemble non basculé) → retrait à
faible risque.
- ✅ Aligné avec les règles repo : **zéro code mort** (règle 7), **pas de garde de compat « au
  cas où »** (anti-pattern #2), **pas de dead-code museum** (anti-pattern #1).
- ✅ Applique la contrainte §0 **par construction** : le schéma ne permet plus un arc bi-titre.
- ✅ Réversible (le plan d'origine notait « si abandonné, `DROP TABLE` sans toucher au schéma arcs »).
- ❌ C'est un retrait de code livré (migration de suppression + tests à retirer).

**Option B — Geler en 1:1 + garde-rail.**
Garder `arc_titles` mais poser un garde-rail (test/contrainte) interdisant > 1 titre par arc,
et figer `creditTitlesFor` au titre primaire.
- ✅ Moins de churn immédiat.
- ❌ Conserve une capacité **volontairement inutilisée** = précisément ce que les anti-patterns
  #1/#2 du repo proscrivent. Un garde-rail qui interdit d'utiliser une structure = signal qu'elle
  ne devrait pas exister.

> **À confirmer par l'utilisateur avant exécution.** Défaut retenu si pas d'objection : **A**.

## 3. Phases (backend uniquement — aucune UX cross-titre)

### Phase 0 — Décision (FAITE)
Option A (retrait) confirmée user 2026-07-18. Le reste du plan exécute **A**.

### Phase 1 — Retrait de la capacité multi-titres d'arc (si Option A)
1. Migration additive de **suppression** : `DROP TABLE IF EXISTS arc_titles` (nouveau step
   `internal/migration/`, style des steps existants ; irréversible côté données mais la donnée
   = miroir 1:1 de `arc.title_slug`, aucune information perdue). Documenter en tête du step la
   raison + la date (décision produit 2026-07-18).
2. Supprimer `prestige.ArcTitlesRepo` + son implémentation DuckDB + les câblages (`Create` qui
   maintenait l'invariant) + le seam `creditTitlesFor` (les callers reviennent à
   `[challenge.TitleSlug]` en dur / au titre primaire direct).
3. Supprimer les tests dédiés (`arc_titles` backfill/invariant/extension point). **0 code mort** :
   imports, helpers orphelins retirés dans le même commit.

### Phase 2 — Garde-rail d'indépendance par titre (le cœur du plan)
Empêcher qu'un couplage cross-titre réapparaisse **ailleurs** que dans les arcs (coach, prestige,
ratings, PP, leaderboards).
1. **Ratchet de test** : un garde-rail (grep-test / test structurel) interdisant, dans
   `internal/prestige/` et `internal/progression/`, toute agrégation qui itère sur plusieurs
   `title_slug` pour un même calcul de valeur/PP/rating (motifs interdits : boucle sur
   `TitleRegistry` pour sommer des PP, requête sans filtre `title_slug` sur une table per-titre,
   etc.). Modèle : `no_slug_comparison_test.go` / `no_art_patterns_test.go` (ratchets existants).
2. **Vérification de lecture** : auditer que tout lecteur d'arcs/challenges/PP porte un
   `title_slug` (déjà le cas via l'isolation FS — confirmer qu'aucun chemin ne lit « tous titres »).
3. Documenter la règle dans le skill `arch-rules` (ou `canonical-types`) : « prestige/arcs =
   per-titre strict ; juxtaposition UI autorisée, jamais d'agrégation cross-titre ».

### Phase 3 — Préparation propre du 2e titre (à l'arrivée réelle, hors ce plan)
Quand le 2e titre entre en pipeline : vérifier que la création/lecture d'arcs, le coach et les PP
s'instancient **par titre** via les capabilities + `PathResolver`, sans aucune ligne de code
supposant un titre unique en dur (grep `slug == "..."` déjà interdit par ratchet). **Aucun
schéma cross-titre à ajouter** — l'indépendance est l'architecture.

## 4. Garde-fous & non-objectifs

- ⛔ Ne réintroduire **aucune** table/colonne/champ liant un arc à plusieurs titres.
- ⛔ Ne pas exposer d'API « arcs tous titres » ; un endpoint reste **scopé à un titre**.
- ⛔ Pas d'agrégation PP/rating/leaderboard cross-titre (le garde-rail Phase 2 le verrouille).
- ✅ Comportement mono-titre observable **strictement inchangé** en fin de plan.
- ✅ Juxtaposition d'affichage par titre = OK (c'est de la présentation, pas de la fusion).

## 5. Gates

```
# depuis apps/go-api/
go build ./... ; go vet ./...
go test ./...
go test -tags=integration -p 1 ./internal/prestige/... ./internal/platform/duckdb/... ./internal/migration/...   # touche migration/persist
# garde-rail Phase 2 : le ratchet doit être VERT et ÉCHOUER si on ré-introduit un couplage cross-titre (prouver l'anti-no-op)
```

## 6. Branche & livraison

- Branche dédiée `feat/arcs-per-title-strict` depuis `main` à jour (ne PAS greffer sur le train
  backlog en cours). Ne jamais travailler sur `main` (push = deploy prod).
- Commits : `refactor(prestige): retire la capacite multi-titres d'arc (arc_titles)` /
  `feat(prestige): garde-rail independance per-titre` / tests.
- Entrée `.ai/thought_log.md` à chaque étape. Contrat `plan-execution`.

## 7. Références

- `.ai/V7/PLAN_CROSS_TITLE_ARCS_BACKEND.md` (livré 2026-06-18 — la brique retirée par ce plan).
- `internal/prestige/{types.go, repository.go, service.go, service_arcs_squads.go}` (arcs).
- `internal/domain/title/registry.go` (`PathResolver.PlayerDBPath` — isolation per-titre, ADR 0008).
- Ratchets modèles : `internal/sync/no_art_patterns_test.go`, `no_slug_comparison_test.go`.
- ADR 0008 (isolation par chemin), 0025 (refactor title-agnostic), 0020/0021 (coach→prestige).
