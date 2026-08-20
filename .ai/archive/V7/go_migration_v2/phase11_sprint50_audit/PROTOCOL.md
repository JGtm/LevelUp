# Protocole de double validation Claude + ChatGPT

> Ce document décrit **comment** mener l'audit pour que les deux rapports soient comparables et que la réconciliation ait du sens. À lire et appliquer **avant** de remplir le moindre template.

## Principes non négociables

1. **Mêmes entrées** — Claude et ChatGPT reçoivent le même corpus (liste de fichiers, même SHA Git, mêmes prompts). Toute divergence d'entrée invalide la comparaison.
2. **Mêmes sorties** — Les deux LLM remplissent le **même template** avec la même structure (sections fixes, tableaux avec colonnes identiques).
3. **Classification unifiée** — Chaque écart/observation est classé selon la même grille (cf. §Classification).
4. **Revue simultanée assumée** — les deux passages sont lancés **en parallèle** à partir du même corpus. On n'essaie pas de simuler une indépendance artificielle (quelle que soit la séquence, les biais LLM se chevauchent). Le bénéfice recherché n'est pas l'indépendance probabiliste mais la **couverture augmentée** : deux modèles, deux styles de lecture, deux façons de rater un point aveugle différent.
5. **Réconciliation par un tiers** — la réconciliation est menée par l'**humain** (toi), pas par un des deux LLM qui a rédigé une review. Claude peut assister (proposer un arbitrage, préparer les tableaux), mais la décision finale t'appartient. Cela évite "juge et partie".

## Classification des écarts (grille commune aux 3 axes)

Cf. [README.md](README.md#classification) pour la grille 🔴🟠🟡🟢 (source unique). Rappel de la règle d'or : **tout écart identifié DOIT recevoir une classification**. Un écart non classé = rapport incomplet.

### Seuils quantitatifs pour le niveau 🔴 bloquant

Un écart est 🔴 bloquant si **au moins une** de ces conditions est réunie :

- **Parité (axe 1)** : perte de fonctionnalité utilisateur critique, algorithme avec golden value divergente non documentée comme modernisation 🟢, schéma DuckDB divergent non migré
- **Qualité (axe 2)** : violation de règle hexagonale (import interdit entre couches), fichier > 500 L non whitelisté, fonction > 150 L sans justification, workaround/fallback sans TTL **et** sans ticket
- **Tests (axe 3)** : coverage globale Go < 70% (alignement Phase 10), package critique sous son seuil Phase 10 (handlers < 75%, middleware < 80%, sync < 70%, migration < 75%, platform/duckdb < 70%, validation < 70%), scénario de non-régression critique issu du Python sans équivalent Go/React, golden value divergente sur un des 7 algorithmes cœurs

## Processus par axe

### Étape 1 — Préparation (à faire une fois par axe)

- Vérifier que `SCOPE.md` liste exhaustivement les fichiers / endpoints / pages à analyser
- Vérifier que `CHECKLIST.md` est découpée par sous-domaine (pas un seul paquet de 500 items)
- Fixer le SHA Git de référence pour chaque worktree (Python + Go + React) et le noter dans `SCOPE.md`

### Étape 2 — Passages LLM en parallèle

Les deux passages sont lancés **en parallèle** (pas en séquence). Pour chaque axe :

1. **Claude** — ouvrir une session Claude Code sur `phase11/sprint50-triple-audit`, copier `templates/axisN_template.md` → `axisN_*/claude_review.md`, remplir section par section. Commit : `docs(phase11-audit): axisN claude review`.
2. **ChatGPT** — fournir en parallèle le même corpus (chemins absolus, SHAs figés dans `SCOPE.md`) et le même template vide. Récupérer la sortie, la poser dans `axisN_*/chatgpt_review.md`. Commit : `docs(phase11-audit): axisN chatgpt review`.

Les deux reviews peuvent être commitées dans n'importe quel ordre — ce sont des artefacts indépendants. L'essentiel est que **ni Claude ni ChatGPT ne disposent de l'autre review** au moment de produire la leur (pas de lecture croisée en cours de rédaction).

### Étape 3 — Réconciliation (par toi, pas par un LLM-auteur)

1. Ouvrir `axisN_*/RECONCILIATION.md`
2. Pour chaque item du template :
   - **Convergence** (même écart, même classif) → consigner sans débat
   - **Divergence de classification** (même écart, classifs différentes) → arbitrage **motivé** ; heuristique par défaut = classif la plus sévère, sauf justification écrite
   - **Écart identifié par un seul LLM** → lecture manuelle rapide du code concerné → retenir (avec classif) ou écarter (avec motif)
3. Tu peux t'appuyer sur une 3e session LLM *fraîche* (Claude ou autre) pour préparer les tableaux ou suggérer l'arbitrage, mais la **décision t'appartient** — pas au LLM qui a rédigé une des deux reviews (juge et partie).
4. Commit : `docs(phase11-audit): axisN reconciliation`

### Étape 4 — Rapport final

Une fois les 3 réconciliations faites, remplir `FINAL_REPORT.md` :
- Tableau consolidé des écarts bloquants / majeurs par axe
- Plan d'action priorisé avec owner et deadline
- Décision go / no-go sur la bascule Go + React
- Plan de rollback si no-go (cf. §Rollback)

## Plan de rollback si no-go

Si `FINAL_REPORT.md` conclut à un **NO-GO** (écarts bloquants non résolus) :

1. La bascule prod du backend Go et/ou du frontend React est reportée
2. Python/Streamlit reste le runtime actif en prod
3. Les écarts 🔴 bloquants sont reportés dans un nouveau sprint (Sprint 51+) avec priorité explicite
4. `FINAL_REPORT.md` documente la date de re-test ciblée et les critères à vérifier pour déclencher un nouveau Sprint 50 (on ne redéroule pas l'audit complet — seulement les items 🔴 initialement identifiés)
5. La branche `phase11/sprint50-triple-audit` reste sur son état d'audit ; le travail correctif part d'une nouvelle branche `fix/phase11-rX-<zone>`

Si **GO conditionnel** : bascule autorisée mais les conditions listées dans `FINAL_REPORT.md §5` sont des blockers pour un eventual rollback automatique (ex : si la métrique X dégrade en prod, on rollback).

## Règles d'hygiène

- **Pas de remplissage partiel** : un template à moitié rempli ne déclenche pas l'étape suivante
- **Un commit par étape** : facilite le rollback et la revue ultérieure
- **Pas de refactoring pendant l'audit** : l'objectif du Sprint 50 est de *documenter*, pas de refondre. Exception autorisée : fix trivial < 5 lignes (typo, log manquant évident, import mort) si tracé dans le `claude_review.md`/`chatgpt_review.md` de l'axe concerné avec mention « fix trivial inline ». Toute correction > 5 lignes ou qui touche la logique = sprint suivant.
- **Thought log** : une entrée par étape majeure dans `.ai/thought_log.md` du worktree Go

## Convention de commits

Tous les commits de ce sprint préfixés `docs(phase11-audit)` (cohérent avec les conventions commits `feat/fix/docs/chore` du projet). Exemples :
- `docs(phase11-audit): axis1 claude review`
- `docs(phase11-audit): axis2 reconciliation`
- `docs(phase11-audit): final report go/no-go`

## Anti-patterns à éviter

- ❌ Lecture croisée en cours de rédaction (regarder la review de l'autre LLM avant d'avoir fini la sienne)
- ❌ Réconciliation par le LLM qui a rédigé une des deux reviews (juge et partie)
- ❌ Sauter la classification sur un écart « évident »
- ❌ Remplissage vague (« globalement bonne qualité »)
- ❌ Écart sans fichier:ligne
- ❌ Modernisation 🟢 sans motivation écrite
- ❌ Seuil de gate modifié en cours d'audit pour « arranger » un résultat
