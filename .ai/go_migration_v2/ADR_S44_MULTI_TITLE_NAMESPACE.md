# ADR_S44_MULTI_TITLE_NAMESPACE.md — Namespace par titre pour le runtime multi-titres

## Statut

**Acceptée**

Date : 2026-04-16

## Contexte

Le runtime LevelUp actuel est mono-titre par construction.

Les éléments suivants sont aujourd'hui implicites ou hardcodés autour de `halo_infinite` :

1. chemins de stockage DuckDB ;
2. configuration et profils joueurs ;
3. bootstrap et contexte session ;
4. liens publics et conventions produit.

> **Note** : l'auth (flow MSAL) est titre-agnostique et n'est pas impactée par cette ADR.
> Le projet Python LevelUp n'est plus maintenu à ce stade ; le Go est la seule baseline.
> Aucune rétrocompatibilité Python n'est requise.

L'audit consolidé a montré qu'un support multi-titres naïf casserait rapidement le modèle actuel si l'on essayait d'injecter `title_slug` directement dans toutes les tables, PK, vues SQL et jointures metadata.

## Problème

Comment introduire un support multi-titres sans :

1. réécrire tout le schéma DuckDB existant ;
2. multiplier les chemins implicites ;
3. casser Halo Infinite pendant la transition ;
4. laisser le runtime choisir son titre courant de façon ambiguë.

## Décision

Le runtime Go adopte une stratégie de **namespace par titre**.

### 1. Namespace de stockage retenu

La structure cible devient :

1. `data/titles/{title_slug}/warehouse/metadata.duckdb`
2. `data/titles/{title_slug}/warehouse/shared_matches_v2.duckdb`
3. `data/titles/{title_slug}/warehouse/shared_pve.duckdb` si pertinent
4. `data/titles/{title_slug}/players/{gamertag}/stats.duckdb`

### 1 bis. Artefacts globaux explicitement conservés

Le namespace par titre ne signifie pas que tout le runtime devient title-aware au niveau du chemin physique.

Restent globaux par design :

1. `db_profiles.json` (mais en format v3 multi-titres)
2. `app_settings.json`
3. `data/sessions/{session_id}.json`
4. `data/cache/jobs.json`

Leur invariant change : ces artefacts restent uniques à l'instance, mais portent désormais `title_slug` dans leur contenu métier quand c'est nécessaire.

### 2. `title_slug` devient une dimension runtime explicite

Le titre courant doit être porté explicitement par :

1. la configuration ;
2. la session ;
3. le bootstrap ;
4. les jobs persistants ;
5. les résolveurs de profils et de chemins (`PlayerResolver`, `PathResolver`).

Le changement de titre à runtime est préparé structurellement :

1. `POST /session/context` accepte un `title_slug` optionnel ;
2. un switch de titre invalide le joueur courant (sécurité anti-fuite cross-titre) ;
3. le backend retourne un bootstrap complet avec contexte du nouveau titre ;
4. le frontend flush ses stores et se re-hydrate entièrement ;
5. le pool DuckDB ouvre les connexions du nouveau titre en lazy (à la première requête, pas au switch).

Aucun bouton de sélection de titre n'est livré avec le Sprint 44. La plomberie est complète pour que l'ajout d'un sélecteur ne nécessite qu'un composant React appelant `switchTitle(slug)`.

> **Auth hors périmètre** : le flow MSAL est titre-agnostique (confirmé par audit).
> Aucune modification requise dans `internal/platform/auth/`.

Le runtime ne doit plus déduire implicitement le titre courant à partir d'un chemin legacy, d'un joueur courant ou d'une convention historique Halo Infinite.

### 3. Source de vérité centrale des titres

Le système introduit un registre de titres de référence, de type `TitleRegistry` ou équivalent.

Ce registre doit décrire au minimum :

1. `title_slug`
2. provider associé
3. capabilities produit visibles
4. chemins runtime dérivés
5. éventuels statuts de disponibilité et defaults

### 4. Compatibilité legacy explicite

Halo Infinite reste le titre legacy de départ.

Pendant la migration :

1. la lecture des chemins historiques HI-only reste supportée si nécessaire ;
2. l'écriture cible converge vers le namespace par titre ;
3. une migration opérable `dry-run / apply / rollback` est fournie via un **manifest JSON** (`operations.json`) traçant chaque opération `(source, dest)` — rollback = exécution inverse du manifest ;
4. pas de symlinks (problématiques sur Windows) — la migration déplace physiquement les fichiers ;
5. backup automatique avant `apply` (copie des fichiers source dans un répertoire horodaté) ;
6. aucune suppression destructive du layout legacy n'est tolérée sans backup et rollback.

### 5. Non-décision explicite

Cette ADR **rejette** l'approche consistant à rendre `title_slug` premier citoyen dans toutes les tables DuckDB existantes à court terme.

Le namespace par titre est précisément retenu pour éviter :

1. une requalification massive des PK et index ;
2. la réécriture immédiate des vues `v_*` ;
3. des migrations SQL transverses à fort risque ;
4. l'introduction d'un multi-jeux partiellement cohérent et difficile à rollback.

## Conséquences attendues

### Positives

1. le blast radius SQL est contenu ;
2. Halo Infinite peut rester stable pendant l'introduction du multi-titres ;
3. l'isolement des données par titre est naturel ;
4. la migration est plus réversible ;
5. le runtime devient compatible avec une montée en charge multi-titres future sans inheritance implicite du layout HI-only.

### Négatives

1. la config, la session, le bootstrap et les jobs doivent tous devenir title-aware ;
2. le `PlayerResolver` et le pool DuckDB (13 fichiers `*_repo.go`) doivent être refactorés pour accepter `(title_slug, gamertag)` ;
3. `db_profiles.json` doit être migré vers un format v3 title-aware avec rétrocompatibilité lecture ;
4. il faut maintenir temporairement une compatibilité legacy ;
5. la CI, les golden values, les fixtures et l'E2E doivent être enrichis ;
6. il faut créer un corpus synthétique de second titre (~0.5–1 jour dédié) si aucun deuxième titre réel n'est encore branché.

## Alternatives considérées

### Alternative A — Rester mono-titre Halo Infinite seulement

Rejetée.

Raison : la décision produit a été prise d'introduire maintenant le support multi-titres, et le Sprint 44 est précisément réservé à cette mise en place propre.

### Alternative B — Une seule famille de DB multi-jeux avec `title_slug` dans toutes les tables

Rejetée à court terme.

Raison : effort et risque disproportionnés par rapport au besoin, requalification trop large des tables, vues, migrations et jointures.

### Alternative C — Support multi-titres purement applicatif sans namespace de stockage

Rejetée.

Raison : laisse trop d'implicites dans les chemins, augmente le risque de fuite inter-titres et rend l'exploitation fragile.

## Plan de mise en oeuvre

La mise en oeuvre est portée par le Sprint 44 et détaillée dans [SPRINT_44_WORKPACKAGES.md](SPRINT_44_WORKPACKAGES.md).

Les obligations de livraison sont :

1. `TitleRegistry` / `PathResolver` centralisés ;
2. `PlayerResolver` title-aware + pool DuckDB clé `{title}:{gamertag}` ;
3. `db_profiles.json` v3 title-aware avec rétrocompatibilité lecture ;
4. matrice explicite des chemins globaux vs title-aware (`db_profiles.json`, `app_settings.json`, `data/sessions`, `data/cache/jobs.json` restent globaux) ;
5. runtime title-aware pour config, session, bootstrap, jobs, provisioning joueur et listing joueurs (auth hors périmètre) ;
6. switch titre runtime fonctionnel : `POST /session/context` + invalidation joueur + re-bootstrap ;
7. `BootstrapResponse` enrichi : `current_title`, `available_titles` (type `TitleSummary`) ;
8. `JobMeta` structuré avec `TitleSlug` obligatoire validé via `TitleRegistry` ;
9. namespace de stockage branché ;
10. migration `dry-run / apply / rollback` via manifest JSON, testée et idempotente ;
11. frontend : `appShellStore.switchTitle()` + `isTitleSwitching` + stores `reset()` + `settingsDraftStore.lastPlayerSlug` + routes TanStack/query keys/navigation/codegen/API client title-aware ;
12. golden parity Halo Infinite pré/post migration ;
13. validation d'isolement inter-titres (corpus synthétique ~0.5–1j) ;
14. logging structuré slog : `title_switched`, `legacy_session`, `bootstrap_served`, `job_created` ;
15. couverture ciblée ≥ 80% sur les modules modifiés, couverture Go globale ≥ 50% ;
16. tests quantifiés : 20 WP2 + 17 WP4 + golden + smoke E2E Playwright ;
17. runbook et rollback plan à jour.

## Critères d'acceptation

Cette ADR n'est considérée correctement mise en oeuvre que si :

1. Halo Infinite ne régresse pas fonctionnellement (golden diff = 0) ;
2. le titre courant est explicite dans le runtime ;
3. le stockage est namespacé par titre ;
4. la migration est opérable et réversible (manifest JSON + rollback testé) ;
5. l'isolement inter-titres est prouvé par les tests et les corpus de validation ;
6. le switch titre runtime est fonctionnel (`POST /session/context` + re-bootstrap + invalidation joueur) ;
7. le logging structuré est en place (`title_switched`, `legacy_session`, `bootstrap_served`) ;
8. la couverture des modules touchés est ≥ 80% ;
9. `golangci-lint run` est clean.

## Documents liés

1. [SPRINT_ROADMAP.md](SPRINT_ROADMAP.md)
2. [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md)
3. [SPRINT_44_WORKPACKAGES.md](SPRINT_44_WORKPACKAGES.md)
4. [AUDIT_CONSOLIDE.md](AUDIT_CONSOLIDE.md)
