# Phase 11 · Sprint 50 — Triple audit final (parité / qualité / tests)

> **Statut** : ⬜ En attente de démarrage
> **Branche** : `phase11/sprint50-triple-audit`
> **Rattachement roadmap** : [SPRINT_ROADMAP.md — Phase 11 / Sprint 50](../SPRINT_ROADMAP.md#sprint-50--triple-audit-final-parité--qualité--tests)
> **Créé le** : 2026-04-17

## Objectif

Clôturer la migration Python → Go **et** Streamlit → React avec une triple validation croisée Claude + ChatGPT sur trois axes indépendants mais complémentaires. Aucun écart bloquant ne doit rester ouvert en sortie de ce sprint.

## Portée

- **Baseline Python** : worktree principal `c:\Users\Guillaume\Downloads\Scripts\LevelUp` (dernière version sur `main` au démarrage du sprint)
- **Cible Go** : worktree `LevelUp-go-migration`, backend `apps/go-api/`
- **Cible React** : worktree `LevelUp-go-migration`, frontend `apps/web/`
- **Baseline Streamlit** : `src/ui/pages/` dans le worktree Python

## Les 3 axes

| # | Axe | Objectif | Dossier |
|---|-----|----------|---------|
| 1 | Parité fonctionnelle & technique | Écarts Python↔Go + Streamlit↔React (modernisations tolérées, classées) | [`axis1_parity_python_vs_go/`](axis1_parity_python_vs_go/) |
| 2 | Architecture & qualité code | Hexagonal, abstractions, DRY, factorisation, workarounds/fallbacks non pertinents | [`axis2_architecture_quality/`](axis2_architecture_quality/) |
| 3 | Tests & logging | Couverture unitaires, non-régression, observabilité | [`axis3_tests_and_logging/`](axis3_tests_and_logging/) |

## Protocole de double validation

Voir [PROTOCOL.md](PROTOCOL.md) pour le détail. Principe :

1. Chaque axe dispose d'un **template identique** rempli en parallèle par Claude et ChatGPT (corpus d'entrée et prompts identiques, pas de lecture croisée en cours de rédaction).
2. Les deux rapports sont réconciliés par l'humain dans un fichier `RECONCILIATION.md` par axe (les LLM-auteurs ne sont pas juges de leur propre review).
3. Un `FINAL_REPORT.md` consolide les 3 réconciliations et priorise les actions correctives.

## Classification

Grille commune aux 3 axes. Tout écart identifié **doit** recevoir une classification — un écart non classé = rapport incomplet.

| Niveau | Signification | Action |
|:------:|---------------|--------|
| 🔴 **Bloquant** | Perte de fonctionnalité critique, violation de règle structurelle, gate Phase 10 non tenu | **Empêche la bascule prod** tant que non résolu. Seuils détaillés dans [PROTOCOL.md §Seuils quantitatifs](PROTOCOL.md#seuils-quantitatifs-pour-le-niveau--bloquant). |
| 🟠 **Majeur** | Écart significatif mais contournable, dette technique importante | Doit être **ticketé** et référencé dans `FINAL_REPORT.md §4.1`. N'empêche pas la bascule si planifié. |
| 🟡 **Mineur** | Inconsistance locale, polish, nettoyage ponctuel | Fix opportuniste, listé en `FINAL_REPORT.md §4.2`. |
| 🟢 **Toléré / Modernisation** | Divergence **intentionnelle** (ex : algorithme remplacé par une version meilleure, widget remplacé par équivalent supérieur) | **Motivation écrite obligatoire** dans la review (sans motivation = reclassé 🟠). Listé en `FINAL_REPORT.md §4.3`. |

## Livrables

- [ ] [PROTOCOL.md](PROTOCOL.md) — méthodologie détaillée (rempli à l'ouverture du sprint)
- [ ] 3 × `SCOPE.md` — périmètre exact de chaque axe
- [ ] 3 × `CHECKLIST.md` — listes à cocher découpées par domaine
- [ ] 3 × `templates/axis*_template.md` — gabarit rempli à l'identique par Claude et ChatGPT
- [ ] 6 × reviews (`claude_review.md` + `chatgpt_review.md` par axe)
- [ ] 3 × `RECONCILIATION.md` — arbitrage final par axe
- [ ] [FINAL_REPORT.md](FINAL_REPORT.md) — synthèse globale + plan d'action

## Gate de sortie

Voir `SPRINT_ROADMAP.md` §Sprint 50. Critères-clés :

- Aucun écart classé **bloquant** non résolu ou non ticketé
- Tous les écarts **majeurs** tracés (ticket / commit / PR)
- Chaque axe a un rapport de réconciliation signé Claude + ChatGPT
- `FINAL_REPORT.md` committé avec date de clôture et décision go/no-go bascule finale
