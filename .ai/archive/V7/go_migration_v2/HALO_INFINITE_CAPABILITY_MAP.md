# HALO_INFINITE_CAPABILITY_MAP.md — Capability map initiale pour Halo Infinite

> Document de cadrage Phase 0.
> Cette capability map décrit uniquement la situation connue pour `halo_infinite`.
> Elle ne prédit rien sur un futur titre Halo.

## Rôle du document

Cette capability map sert à trois choses :

1. rendre explicite ce que le provider Halo Infinite sait réellement alimenter pour le produit ;
2. éviter de transformer des spécificités Halo Infinite en hypothèses implicites du contrat LevelUp ;
3. préparer un bootstrap capable d'annoncer proprement les surfaces supportées, dégradées ou absentes.

## Règles d'usage

1. Ce document est mono-titre : `halo_infinite` seulement.
2. Une capacité absente ici est inconnue ou non retenue, pas implicitement supportée.
3. Le niveau décrit est produit-orienté, pas endpoint-orienté.
4. Une capacité `degrade` peut être utilisée par le produit, mais ne doit pas devenir un invariant fort sans preuve supplémentaire.
5. Une capacité `non_expose` ne doit pas apparaître dans le contrat produit comme si elle était fiable ou native.

## Sémantique des statuts

| Statut | Sens |
|--------|------|
| `supporte` | surface disponible et suffisamment stable pour faire partie du contrat produit |
| `degrade` | surface partielle, fragile, heuristique ou dépendante d'une chaîne à haut risque |
| `non_expose` | donnée absente, non fiable ou explicitement exclue du contrat produit |
| `hors_scope` | capacité purement produit ou analytique, hors responsabilité du provider Halo |

## Capability map initiale `halo_infinite`

| Capability produit | Statut | Nature | Source actuelle | Notes de cadrage |
|-------------------|:------:|--------|-----------------|------------------|
| `identity.lookup_by_gamertag` | `supporte` | native provider | `get_user_by_gamertag()` | surface de base pour setup, bootstrap et explorer |
| `identity.bulk_lookup_by_xuid` | `supporte` | native provider | `get_users_by_id()` | utile pour résolution roster et enrichissement produit |
| `profile.customization_snapshot` | `supporte` | native provider | `get_player_customization()` | cosmétique ; ne pas la rendre bloquante pour le reste du produit |
| `match.history` | `supporte` | native provider | `get_match_history()` | capability socle P1 |
| `match.detail.core` | `supporte` | native provider | `get_match_data()` / `get_match_stats()` | socle pour match view, history, explorer |
| `match.timeline.highlights` | `supporte` | native provider | `get_highlight_events()` | timeline exploitable ; la sémantique fine reste distincte des dérivés produit |
| `match.skill.snapshot` | `degrade` | native provider | `get_skill_stats()` | usable mais partielle ; les champs assists attendus sont souvent absents |
| `career.progression` | `supporte` | native provider | `get_career_rank_progression()` | surface stable et visible côté produit |
| `assets.discovery` | `supporte` | native provider | `get_asset()` + metadata | maps, playlists, variants ; indispensable pour labels et filtres |
| `economy.challenges_and_battlepass` | `supporte` | native provider | surfaces live actuelles Halo Infinite | capability utile à Home ; dépendance live à conserver isolée |
| `pve.firefight_stats` | `supporte` | native provider + pipeline produit | pipeline PvE existant documenté dans le plan | à revalider dans le portage Go, mais fait partie du périmètre réel du produit |
| `film.raw_access` | `supporte` | native provider | `get_film_by_match_id()` / `download_film_chunk()` | accès brut aux films ou chunks, sans promesse sur tout le reste |
| `film.weapon_extraction` | `degrade` | dérivé fragile | parser spécifique + réconciliation interne | toléré comme surface à haut risque ; ne jamais en faire un invariant socle |
| `match.weapon_breakdown_native` | `non_expose` | absent du provider | voir `.ai/API_LIMITATIONS.md` | l'API réelle n'expose pas de breakdown officiel fiable par arme |
| `combat.killer_victim_authoritative` | `non_expose` | absent du provider | timeline + heuristiques seulement | la matrice killer/victim exacte reste un dérivé produit, pas une surface native |
| `analytics.sessions` | `hors_scope` | dérivé produit | `src/analysis/sessions.py` | ne doit pas être annoncé comme capability provider |
| `analytics.performance_score` | `hors_scope` | dérivé produit | `src/analysis/_performance_relative.py` | calcul LevelUp, pas capability Halo |
| `analytics.lusr_final` | `hors_scope` | dérivé produit | `src/analysis/skill_rating.py` | la capability provider n'est qu'un snapshot de skill d'entrée |
| `analytics.citations` | `hors_scope` | dérivé produit | `src/analysis/citations/*` | ne dépend pas d'une surface native unique |

## Lecture opérationnelle de la map

### Ce que le produit peut considérer comme socle stable

1. résolution d'identité ;
2. historique de matchs ;
3. détail de match cœur ;
4. progression carrière ;
5. discovery assets ;
6. economy/challenges ;
7. accès film brut ;
8. Firefight/PvE dans le périmètre courant du produit.

### Ce que le produit doit considérer comme surface fragile

1. snapshot de skill natif, car certaines dimensions ne sont pas toujours disponibles ;
2. extraction d'armes à partir des films, car elle dépend d'une chaîne fragile et spécifique au titre.

### Ce qui ne doit pas entrer dans le contrat provider

1. breakdown officiel par arme ;
2. killer/victim exact comme donnée native ;
3. sessions, performance score, LUSR final, citations.

## Projection bootstrap minimale recommandée

Le bootstrap produit n'a pas besoin d'exposer la mécanique Waypoint ou les endpoints 343i.
Il doit seulement annoncer au consommateur ce qu'il peut raisonnablement attendre du provider courant.

### Shape documentaire cible

```json
{
  "halo": {
    "title": "halo_infinite",
    "provider": "spnkr",
    "capabilities": {
      "identity.lookup_by_gamertag": "supporte",
      "identity.bulk_lookup_by_xuid": "supporte",
      "match.history": "supporte",
      "match.detail.core": "supporte",
      "match.skill.snapshot": "degrade",
      "career.progression": "supporte",
      "assets.discovery": "supporte",
      "economy.challenges_and_battlepass": "supporte",
      "pve.firefight_stats": "supporte",
      "film.raw_access": "supporte",
      "film.weapon_extraction": "degrade",
      "match.weapon_breakdown_native": "non_expose"
    }
  }
}
```

### Règles de bootstrap associées

1. exposer uniquement les capabilities qui peuvent changer le comportement du produit ;
2. ne pas exposer de noms d'endpoints Waypoint ni d'URLs externes ;
3. ne pas exposer les métriques purement dérivées LevelUp comme si elles étaient natives ;
4. garder les clés stables pour permettre la comparaison future avec un autre titre.

## Sources ayant servi au cadrage

1. [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md) pour le contrat cible ;
2. [PORTING_REFERENCE.md](PORTING_REFERENCE.md) pour le périmètre produit et les surfaces à couvrir ;
3. [.ai/API_LIMITATIONS.md](../API_LIMITATIONS.md) pour les limites connues de l'API Halo Infinite ;
4. `src/ports/api.py` pour les surfaces provider actuellement exposées côté Python ;
5. le plan de migration v2 pour le rattachement à la Phase 0 et à la Phase 1.

## Règle de maintenance

Cette capability map ne doit changer que si l'un de ces éléments change :

1. le contrat provider réel ;
2. le périmètre produit LevelUp ;
3. la confiance accordée à une surface aujourd'hui marquée `degrade` ;
4. l'introduction d'un nouveau titre Halo documenté avec des preuves réelles.
