# Plan — Calmer le flot Dependabot (15 PRs) et figer une config sobre

> Statut au 2026-06-22 : étapes 1 (fermeture 15 PRs) et 2 (réécriture config) FAITES sur la
> branche `chore/tame-dependabot-grouping`. Reste : thought_log + push + PR (merge = 1 deploy,
> à la main de l'utilisateur).

## Contexte

En mergeant PR #25 (config Dependabot mise en place dans cette session), `.github/dependabot.yml`
est arrivé sur la branche par défaut `main`. Dependabot a immédiatement scanné les 3 écosystèmes
et ouvert **15 PRs** (#26→#40 : 5 npm + 5 gomod + 5 github-actions, sa limite par défaut), chacune
déclenchant toute la matrice CI (~150 jobs). Ce n'est **pas une boucle infinie** — c'est le scan
initial plafonné — mais c'est bruyant et coûteux. La config initiale livrée n'avait **pas de
groupement** : cause directe du flot.

**Décision produit retenue** : ne PAS merger les 15 bumps en l'état. Aucun n'est un correctif de
sécurité ; plusieurs sont des majors d'actions GitHub (`checkout` v4→v7, `download-artifact` v4→v8,
`setup-go` v5→v6, `setup-buildx` v3→v4) + un bump du driver DuckDB (`duckdb-go/v2`, couche la plus
sensible) → risque CI/DB réel pour un gain nul. On tame la config et on revoit les bumps groupés à froid.

## Étapes

### 1. Stopper le bruit — FAIT
Fermeture des 15 PRs `app/dependabot` (#26→#40) via `gh pr close`, avec commentaire « superseded,
reviendront groupées ». Annule les ~150 jobs CI. Vérifié : 0 PR Dependabot ouverte.

### 2. Config sobre — FAIT (`.github/dependabot.yml`)
Pour chaque écosystème (npm `/apps/web`, gomod `/apps/go-api`, github-actions `/`) :
- `schedule.interval: monthly`, `open-pull-requests-limit: 3`
- `groups` : 1 groupe par écosystème, `patterns: ["*"]`, `update-types: [minor, patch]`
  → 1 PR groupée par écosystème au lieu de N individuelles.
- github-actions : `ignore` sur `version-update:semver-major` → plus de proposition de saut majeur.

### 3. Thought log — à faire
Entrée `.ai/thought_log.md` : cause, décision (ne pas merger, tame + fermer), config retenue.

### 4. Livraison — à faire
Push branche + PR dédiée. **Merge sur `main` = 1 deploy prod** (deploy.yml sur tout push main) →
à la main de l'utilisateur. Au merge, Dependabot re-scanne groupé → ~3 PRs groupées (revue à froid).

## Vérification
- `gh pr list --repo JGtm/LevelUp --state open` → plus de PRs `app/dependabot` individuelles.
- Après merge : prochain scan = au plus 1 PR groupée/écosystème, aucune proposition de major d'action.
- `actionlint` (pre-check deploy.yml) valide les workflows.

## Hors scope / différé (à étudier séparément)
Évaluer l'adoption des bumps différés — surtout `duckdb-go/v2` (driver DB, sensible CGO/ART) et les
majors d'actions GitHub. À traiter individuellement (branche + CI + test ciblé), pas en lot.
**Candidat à une étude ultracode dédiée** (cf. session 2026-06-22).
