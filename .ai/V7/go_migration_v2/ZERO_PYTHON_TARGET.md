# ZERO_PYTHON_TARGET.md — Cible terminale du programme

> Ce document rend explicite la cible finale du chantier Go dans le corpus v2.
> Il sert de lecture rapide au-dessus de la stratégie exhaustive locale.

## Position

Le programme ne vise pas seulement moins de Python.
Il vise un produit exécutable sans runtime Python dans son chemin critique.

## Critères terminaux

| Critère | Cible |
|---------|-------|
| Runtime Python en production | `0` |
| Fichiers `.py` exécutés dans le chemin produit | `0` |
| Dépendance `pip` ou `venv` pour installer l'application | `0` |
| Bridge Python SPNKr | supprimé — client Go direct dès S11 |
| `src/ai/` | hors scope produit, hors build et hors packaging |

## Ce que cela signifie concrètement

1. L'utilisateur final n'installe ni Python, ni pip, ni environnement virtuel.
2. Le déploiement produit ne lance aucun script Python.
3. Le packaging final repose sur un binaire Go et ses artefacts web, pas sur une stack Python embarquée.
4. Les tests de parité peuvent rester en Python pendant la transition, tant qu'ils ne font pas partie du chemin d'exécution du produit.
5. `src/ai/` peut rester Python comme outillage développeur, mais ne doit jamais redevenir une dépendance du runtime produit.
6. Le remplacement de SPNKr ne doit pas durcir le produit sur Halo Infinite uniquement : le runtime Go garde un socle provider Halo générique et des adaptateurs par titre.

## Surfaces à faire sortir du chemin produit

| Famille | Sortie attendue |
|---------|-----------------|
| API Python `apps/api/` | remplacée par handlers et services Go |
| Moteur sync/backfill Python | remplacé par commandes et packages Go |
| Scripts critiques `scripts/*.py` utiles à l'ops | remplacés ou absorbés dans les sous-commandes Go |
| `launcher.py`, wrappers de lancement Python | remplacés par un point d'entrée Go |
| `spnkr` et `spnkr_pr/` | éliminés du produit final |
| Packaging Python `pyproject.toml`, `pip install`, image Docker Python | absents du packaging produit final |
| Restes Streamlit exécutables | absents du chemin produit |

## Ce qui peut encore rester en Python hors produit

1. les tests de parité historiques ;
2. les outils développeur sous `src/ai/` ;
3. des documents ou scripts archivés explicitement hors build, hors packaging et hors runbook produit.

## Règles sur le client Go Halo (ex-SPNKr)

SPNKr est remplacé directement par un client Go natif (`pkg/haloapi/`) dès le Sprint 11 — pas de bridge Python transitoire.

1. le client Go porte uniquement le transport HTTP, retry et parsing JSON ;
2. il ne porte aucune logique métier LevelUp ;
3. il ne touche aucune base DuckDB ;
4. il ne conserve aucun état applicatif ;
5. seule exception : si le weapon parser (D6) échoue en Go, un subprocess Python étroit est toléré pour cette seule fonction.

## Jalon ZP — avant la Phase 5

Le programme ne doit pas entrer en Phase 5 tant que les conditions suivantes ne sont pas vraies :

1. aucun fichier `.py` n'est exécuté dans le chemin produit ;
2. aucun `pip install` ou bootstrap `venv` n'est requis dans le packaging final ;
3. le client Go Halo est validé (tous endpoints, 3 cycles sync clean) ;
4. le build et le déploiement produit ne dépendent plus d'un artefact Python ;
5. `src/ai/` est explicitement exclu du build et du packaging produit.

## Règle de non-régression

Pendant le programme :

1. aucun nouveau `.py` ne doit être ajouté au chemin produit ;
2. le client Go Halo ne doit pas être remplacé par un bridge Python ;
3. aucun nouveau besoin d'installation Python ne doit réapparaître dans Docker, les launchers ou le runbook.

## Pour aller plus loin

Le détail module par module, y compris la stratégie d'extinction SPNKr et les correspondances de packaging, reste dans [ZERO_PYTHON_STRATEGY.md](ZERO_PYTHON_STRATEGY.md).
