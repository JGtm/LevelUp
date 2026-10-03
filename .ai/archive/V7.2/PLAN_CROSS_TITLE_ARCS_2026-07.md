# Plan — Arcs multi-titres : indépendance stricte par jeu (2026-07)

> **Créé le** : 2026-07-18
> **Statut** : **EXÉCUTÉ ET CLOS le 2026-07-25** (chantier v7.2.1, item V721-09, branche
> `feat/v7.2.1-notion-batch` — la branche dédiée `feat/arcs-per-title-strict` prévue ici n'a pas
> été ouverte : le chantier v7.2.1 était le train courant). Option A (retrait `arc_titles`)
> appliquée intégralement. Voir « État d'exécution » ci-dessous.
> **Priorité** : ~~🟡 à anticiper avec l'arrivée du 2e titre~~ — prérequis déjà satisfait
> (`halo_5` est un slug actif), plan exécuté.
> **Origine** : décision produit utilisateur (2026-07-18) sur le backlog `[coach/prestige] V3 — Cross-titre arcs`.
> **Supersède partiellement** : `.ai/V7/PLAN_CROSS_TITLE_ARCS_BACKEND.md` (livré 2026-06-18) — voir §2.

## État d'exécution (2026-07-25 — vérifié sur pièces, chantier v7.2.1 item V721-09)

| Item du plan | Statut | Preuve / justification |
|---|---|---|
| Phase 0 — Décision Option A | `[x]` | Décision du 2026-07-18 déjà actée. La ligne résiduelle « à confirmer par l'utilisateur avant exécution » (§2) contredisait le §2 lui-même → retirée. |
| Phase 1.1 — Migration de suppression | `[x]` | Step `drop_arc_titles` (`TargetPlayer`, `DROP TABLE IF EXISTS arc_titles`) dans `internal/games/halo_infinite/migrations/steps_player_base.go`, commentaire portant la raison + la date de décision (2026-07-18). Ajouté à `internal/migration/order.go` (`canonicalOrder`) juste après `create_arc_titles_join`, et déclaré dans `stepDependencies` (`order_dependency_test.go`) pour verrouiller l'ordre créateur→dropper. `create_arc_titles_join` **conservé** dans `canonicalOrder` et dans les steps (enregistrement `schema_migrations` des bases existantes ; le retirer casserait l'audit bidirectionnel `order_audit_test.go`) — commentaires « NE PAS nettoyer » posés aux deux endroits. Son `ApplyBackfill` a en revanche été retiré (il rejouerait un INSERT sur une table droppée). |
| Phase 1.2 — Retrait `ArcTitlesRepo` + invariant `Create` + seam `creditTitlesFor` | `[x]` | Interface `ArcTitlesRepo` supprimée de `internal/prestige/repository.go` ; assertion `_ prestige.ArcTitlesRepo`, méthodes `ArcTitles`/`ArcsByTitle` et l'INSERT `arc_titles` de `Create` supprimés de `internal/platform/duckdb/prestige/prestige_player_repo.go` ; `internal/prestige/cross_title.go` supprimé ; l'unique appelant (`creditCompletion`, `service_evaluate.go`) émet désormais un seul `PrestigeEvent` sur `c.TitleSlug`. Zéro référence résiduelle (grep `ArcTitles|creditTitlesFor|arc_titles` sur `apps/go-api/**/*.go` : seuls des commentaires documentant le retrait). |
| Phase 1.3 — Suppression des tests dédiés | `[x]` | `migrations/arc_titles_test.go`, `platform/duckdb/prestige/prestige_arc_titles_test.go`, `prestige/cross_title_test.go` supprimés. Helper `findPlayerStep` retiré avec son unique fichier (0 autre appelant). Entrée `"arc_titles"` retirée de `order_audit_test.go` et **remplacée par l'assertion inverse** (la table doit être ABSENTE après `RunForDB(TargetPlayer)`) → couverture bout-en-bout du dropper. |
| Phase 2.1 — Ratchet de test (cœur du plan) | `[x]` | `internal/prestige/no_cross_title_aggregation_test.go` : scan AST (`go/parser`, commentaires exclus) de `internal/prestige/` + `internal/progression/`, 4 détecteurs (boucle sur des titres · fonction produisant une liste de titres · symbole `CrossTitle` · SQL sur table per-titre sans `title_slug`), allowlist datée + test anti-pourrissement. **Morsure prouvée** par `TestCrossTitleDetectorsBite` (rejoue le code EXACT retiré) et `TestCrossTitleDetectorsSpareMonoTitle` (contrôle négatif). |
| Phase 2.2 — Audit de lecture | `[x]` avec RÉSERVE | Arcs (`ArcRepo.ListByUser(userID, titleSlug)`), défis (`buildChallengeListQuery` → `title_slug = ?`), streaks/records/milestones (`WHERE user_id = ? AND title_slug = ?`) : tous scopés ; en outre `arc`/`challenge` vivent dans la player DB **per-titre** (ADR 0008). **MAIS 2 agrégations PP cross-titre subsistent** : `PrestigeSocialRepo.GetUserPrestigeCrossTitle` (`SUM(total_pp) … WHERE user_id = ?`, atteignable par `GET /prestige/me` sans `title_slug`) et la branche `titleSlug == nil` de `GetLeaderboard` (`SUM(total_pp) GROUP BY user_id`, **zéro appelant de production**). Voir « Reste à arbitrer ». |
| Phase 2.3 — Documenter la règle | `[x]` | Section « Prestige / arcs / progression — per-titre STRICT » ajoutée à `.claude/skills/arch-rules/SKILL.md` (interdits + exemple correct/interdit + pointeur vers le garde-rail). |
| Phase 3 — Préparation du 2e titre | `[!]` hors périmètre | Explicitement « hors ce plan » (§3). L'audit a néanmoins relevé un obstacle concret à documenter : `wire.NewPrestigeBundle` épingle `titleSlug := titlePkg.DefaultSlug` pour `shared_social.duckdb` ET `metadata.duckdb` → tout le module Prestige est actuellement rattaché à Halo Infinite quel que soit le titre de la requête. À traiter dans le chantier multi-titre, pas ici. |

### Reste à arbitrer (découvertes non traitées — DÉCISION PRODUIT REQUISE)

1. **`GetUserPrestigeCrossTitle`** somme les PP tous titres confondus pour un joueur. Le
   retirer change la réponse de `GET /prestige/me` appelé sans `title_slug` (aujourd'hui :
   total agrégé ; demain : erreur 400 ou total du titre courant). Contredit le §0 de ce plan
   mais n'est PAS un résidu d'`arc_titles` — traité comme dette tracée, pas comme un fix
   opportuniste. Allowlisté (2 entrées datées) dans le garde-rail Phase 2 : le jour où la
   décision tombe, retirer le code fait tomber les entrées (test anti-pourrissement).
2. **`PrestigeRepo.GetLeaderboard`** n'a **aucun appelant de production** : code mort
   (règle 7) dont la branche « tous titres » est de surcroît interdite par le §0. Suppression
   à faire dans un lot dédié (elle touche l'interface `PrestigeRepo` et ses stubs de test).

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

> **Décision close** : Option A, confirmée par l'utilisateur le 2026-07-18. Aucune confirmation
> supplémentaire n'est attendue avant exécution (la ligne « à confirmer » qui vivait ici
> contredisait le §2 et a été retirée le 2026-07-25, chantier v7.2.1 item V721-09).

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
4. **Deux accroches OUBLIÉES par la rédaction initiale** (ajoutées le 2026-07-25, V721-09 — sans
   elles les gates cassent) :
   - `internal/games/halo_infinite/migrations/order_audit_test.go` liste `"arc_titles"` parmi les
     tables player attendues après `RunForDB(TargetPlayer)` → l'entrée doit sauter avec la table.
   - `internal/migration/order_test.go` + `order_audit_test.go` imposent que TOUTE nouvelle
     migration figure dans `canonicalOrder`, ET que `canonicalOrder` ne référence aucun nom absent
     de (registre global + steps title-owned). Conséquences : `drop_arc_titles` DOIT être ajouté à
     `canonicalOrder`, et l'entrée historique `create_arc_titles_join` DOIT y rester **avec son
     step** (c'est un enregistrement `schema_migrations` présent sur toutes les bases existantes ;
     le retirer désynchroniserait le registre et ferait échouer l'audit bidirectionnel).

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

## Journal d'exécution

- **2026-07-18** : plan rédigé, Option A (retrait de la sur-structure `arc_titles`) tranchée
  par l'utilisateur le jour même. Exécution volontairement différée « à l'arrivée du 2e
  titre ». Aucun code touché à cette date.
- **2026-07-25** (chantier v7.2.1, item V721-09 — cf. `.ai/V7.2.1/PLAN_V721_NOTION_BATCH.md`) :
  plan exécuté en une passe. Le prérequis « 2e titre » était en réalité déjà satisfait
  (`halo_5` slug actif).
  - 09.1 — contradiction interne levée : la ligne « À confirmer par l'utilisateur avant
    exécution » (§2) contredisait le statut « confirmée le 2026-07-18 » ; remplacée par une
    mention de décision close. Deux accroches oubliées par la rédaction initiale ajoutées à
    la Phase 1 (l'entrée `"arc_titles"` de `order_audit_test.go`, et l'obligation de déclarer
    toute nouvelle migration dans `canonicalOrder` sans en retirer l'entrée historique).
  - 09.2 — migration `drop_arc_titles` + `canonicalOrder` + `stepDependencies`. Décision
    explicitée en commentaire aux deux endroits : `create_arc_titles_join` RESTE (ledger
    `schema_migrations`), son `ApplyBackfill` est retiré (il taperait une table droppée).
  - 09.3 — `ArcTitlesRepo`, son implémentation, l'invariant de `Create` et le seam
    `creditTitlesFor` supprimés ; `creditCompletion` émet un unique `PrestigeEvent` sur
    `c.TitleSlug`.
  - 09.4 — 3 fichiers de tests dédiés supprimés ; `order_audit_test.go` bascule d'« arc_titles
    doit exister » à « arc_titles doit être ABSENTE » (couverture du dropper conservée).
  - 09.5 — garde-rail `internal/prestige/no_cross_title_aggregation_test.go` (Phase 2),
    morsure prouvée sur le code exact retiré.
  - 09.6 — audit de lecture : arcs/défis/streaks/records/milestones tous scopés `title_slug`
    (+ isolation FS ADR 0008) ; 2 agrégations PP cross-titre relevées et **allowlistées avec
    justification datée** au lieu d'être corrigées en douce (décision produit requise —
    cf. « Reste à arbitrer »).
  - 09.7 — règle documentée dans `.claude/skills/arch-rules/SKILL.md`.
  - 09.8 — ce plan statué item par item puis archivé sous `.ai/archive/`.
  - **Point de vigilance déploiement** : `drop_arc_titles` s'exécute au boot sur **toutes** les
    bases joueur (`data/titles/{slug}/players/{gamertag}/stats.duckdb`). Irréversible côté
    données, mais miroir 1:1 de `arc.title_slug` **vérifié sur pièces** (backfill d'origine =
    `SELECT id, title_slug FROM arc` ; unique writer applicatif = `PrestigeArcRepo.Create` avec
    `(a.ID, a.TitleSlug)`) → aucune information perdue, reconstructible depuis `arc`.
  - **Non fait faute de droits d'écriture sur ces fichiers (réservés à d'autres agents du
    chantier)** : entrée `.ai/thought_log.md` et retrait de la ligne « Arcs multi-titres »
    de `.ai/BACKLOG.md` (ligne 187) — à faire par le pilote du chantier.
