# HALO_BOOTSTRAP_CONTRACT.md — Contrat bootstrap Halo côté produit

> Document de cadrage Phase 0/1.
> Il décrit la portion Halo du bootstrap produit, sans présumer du reste du payload global de l'application.

## Rôle du document

Ce document fixe le contrat minimal que le backend Go devra pouvoir exposer dans le bootstrap pour rendre la façade produit indépendante des détails du provider Halo.

Il sert à :

1. annoncer proprement le titre courant et le provider courant ;
2. exposer les capabilities produit réellement supportées ;
3. rendre visible une dégradation ou une absence de surface sans casser le reste du bootstrap ;
4. garder une forme stable si le provider change plus tard.

## Ce document ne définit pas

1. le bootstrap complet de LevelUp ;
2. la structure des blocs settings, session, player context ou jobs ;
3. les endpoints Waypoint ou leurs URLs ;
4. les payloads bruts du provider ;
5. les calculs métier dérivés LevelUp.

## Position dans l'architecture

```text
Provider Halo
  -> modèle canonique Halo
    -> bootstrap produit LevelUp
      -> façade web / consumers backend
```

Le bootstrap Halo est donc un contrat produit consommable directement.
Il n'est ni un payload natif Waypoint, ni un simple dump de metadata technique.

## Principes de conception

1. Le bootstrap reste orienté produit.
2. Les clés de capabilities sont stables et indépendantes du provider concret.
3. Les statuts permettent la dégradation gracieuse sans mensonge sur la réalité du provider.
4. Les consommateurs n'ont pas besoin de connaître les détails 343i pour décider quoi afficher, cacher ou dégrader.
5. Le bloc Halo doit rester suffisamment petit pour être lu au démarrage de l'app sans devenir un mini inventaire technique complet.

## Shape canonique recommandée

### Bloc `halo`

| Champ | Statut | Rôle |
|------|:------:|------|
| `title` | requis | identifiant stable du titre courant, ex. `halo_infinite` |
| `provider` | requis | identifiant stable du provider courant, ex. `spnkr`, `native_go` |
| `capability_schema_version` | requis | version du dictionnaire de capabilities exposé au consommateur |
| `capabilities` | requis | map `capability_key -> status` |
| `limitations` | optionnel | liste de limitations ou gaps réellement utiles au consommateur |

### Exemple documentaire minimal

```json
{
  "halo": {
    "title": "halo_infinite",
    "provider": "spnkr",
    "capability_schema_version": "v1",
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
    },
    "limitations": [
      {
        "capability_key": "match.weapon_breakdown_native",
        "reason_code": "provider_not_available",
        "severity": "warning"
      }
    ]
  }
}
```

## Sémantique des statuts côté bootstrap

| Statut | Comportement attendu côté consommateur |
|--------|----------------------------------------|
| `supporte` | surface disponible sans garde spéciale autre que les nullabilités habituelles |
| `degrade` | surface utilisable avec prudence, fallback ou UI dégradée si nécessaire |
| `non_expose` | surface à masquer ou à traiter comme indisponible |
| `hors_scope` | en principe absent du bootstrap ; utile seulement dans la documentation source |

## Règles de consommation côté frontend / API produit

1. Une capability absente du bloc bootstrap ne doit pas être interprétée comme `supporte`.
2. Une capability `degrade` ne doit pas casser un écran entier si seul un sous-module dépend de cette surface.
3. Une capability `non_expose` doit conduire à masquer la fonctionnalité dépendante ou à la remplacer par un fallback purement produit.
4. Le consommateur ne doit jamais inférer des URLs, des endpoints externes ou des heuristiques provider à partir du bootstrap.

## `limitations` : quand les exposer

La liste `limitations` ne doit contenir que des gaps réellement utiles au consommateur produit ou au diagnostic opérable.

Exemples pertinents :

1. une surface officiellement absente du provider ;
2. une dimension notoirement partielle, comme certaines stats de skill ;
3. une capacité disponible mais marquée fragile ou expérimentale.

Exemples à exclure :

1. détails internes d'implémentation du provider ;
2. noms d'endpoints Waypoint ;
3. URLs d'assets ou détails techniques sans impact produit.

## Dictionnaire recommandé des `reason_code`

| `reason_code` | Sens |
|--------------|------|
| `provider_not_available` | la surface n'existe pas côté provider |
| `provider_partial_data` | la surface existe mais avec des champs incomplets ou fragiles |
| `product_disabled_for_title` | le produit ne choisit pas d'exposer cette surface pour ce titre |
| `derived_only_not_native` | la surface n'existe pas nativement et ne doit pas être présentée comme capability provider |
| `temporarily_bridged` | la surface dépend provisoirement d'un bridge transitoire |

## Ce qui doit rester hors du bloc `halo`

1. sessions calculées LevelUp ;
2. performance score ;
3. LUSR final du produit ;
4. citations et commendations ;
5. état détaillé des jobs ;
6. configuration d'UI ou de navigation.

## Relation avec les autres documents

1. [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md) fixe les objets canoniques derrière ce bootstrap.
2. [HALO_INFINITE_CAPABILITY_MAP.md](HALO_INFINITE_CAPABILITY_MAP.md) fixe la première map mono-titre à projeter ici.
3. [HALO_INFINITE_CANONICAL_MAPPING.md](HALO_INFINITE_CANONICAL_MAPPING.md) documente la projection `halo_infinite -> canonique` dont ce bootstrap dépend indirectement.
4. [HALO_PRODUCT_CONTRACT_ADAPTERS.md](HALO_PRODUCT_CONTRACT_ADAPTERS.md) décrit la projection `canonique -> bootstrap/OpenAPI`.
5. [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md) rattache ce contrat à la Phase 0 et à la Phase 1.

## Règle de maintenance

Le contrat bootstrap Halo ne doit changer que si :

1. une capability produit pertinente change de statut ;
2. une nouvelle capability devient nécessaire au comportement du consommateur ;
3. un nouveau titre documenté impose une extension stable du bloc `halo`.
