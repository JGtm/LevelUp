# HALO_PRODUCT_CONTRACT_ADAPTERS.md — Adaptation du canonique Halo vers les contrats produit

> Document de préparation à l'implémentation.
> Il décrit comment les objets canoniques Halo doivent être projetés vers le bootstrap produit et les contrats OpenAPI sans laisser les handlers ou le frontend reconstruire eux-mêmes la logique de transformation.

## Rôle du document

Ce document fixe la couche située entre :

1. les objets canoniques Halo ;
2. les read models ou DTOs produit par parcours ;
3. les réponses JSON exposées au frontend.

Il répond à une contrainte du chantier Go :

le modèle canonique ne doit pas devenir directement le contrat HTTP final, mais le contrat HTTP ne doit pas non plus court-circuiter le modèle canonique.

## Ce document ne définit pas

1. le modèle canonique lui-même ;
2. le mapping Halo Infinite natif ;
3. les calculs métier internes comme sessions, LUSR, citations ou enrichissements de persistance ;
4. le schéma OpenAPI définitif ligne par ligne.

## Position dans l'architecture

```text
provider de titre
  -> modèle canonique Halo
    -> adaptateurs produit par parcours
      -> DTOs OpenAPI / réponses HTTP
        -> frontend
```

## Trois couches à garder distinctes

### 1. Canonique Halo

Vérité stable du domaine Halo côté backend.

### 2. Read model produit

Forme orientée parcours utilisateur : bootstrap, history, match view, career, setup.

### 3. DTO HTTP / OpenAPI

Projection contractuelle finale, alignée sur la version d'API publique/interne retenue.

Règle :

un handler ne doit ni mapper un payload provider brut, ni manipuler directement tous les champs du canonique au fil de l'eau.

## Emplacement cible recommandé

```text
apps/go-api/
  internal/
    api/
      adapters/
        bootstrap/
        history/
        matchview/
        career/
        setup/
      openapi/
        dto/
```

## Principes d'adaptation

1. Chaque adaptateur part d'objets canoniques explicites, pas d'un provider ou d'un repository brut.
2. Une dégradation déjà exprimée dans le canonique doit rester visible dans le contrat produit.
3. Les adaptateurs produit ne reconstruisent pas des données Halo natives absentes ; ils orchestrent seulement ce qui est déjà canonique ou produit.
4. Les contrats HTTP exposent un wording stable orienté produit, pas un lexique Halo Infinite ou SPNKr.
5. Les handlers assemblent les dépendances, mais la logique de forme vit dans les adaptateurs.

## Bootstrap : adaptation recommandée

Le bootstrap produit n'est pas un dump des objets canoniques.
Il sélectionne les informations nécessaires au démarrage.

### Entrées minimales

1. `TitleRuntimeContext` canonique ;
2. identité joueur courante si disponible ;
3. settings ou contexte produit provenant d'autres couches ;
4. flags de disponibilité utiles aux parcours initiaux.

### Sortie attendue

Le bloc `halo` du bootstrap suit [HALO_BOOTSTRAP_CONTRACT.md](HALO_BOOTSTRAP_CONTRACT.md).

Règles :

1. `title`, `provider`, `capabilities`, `limitations` proviennent du `TitleRuntimeContext` ;
2. aucun détail d'endpoint ou de provider natif ne fuit dans le bootstrap ;
3. une capability absente ou `non_expose` ne doit pas nécessiter de logique de reconstruction côté frontend.

## Match History : adaptation recommandée

### Entrée

`MatchHistoryPage` canonique.

### Projection produit

1. pagination conservée explicitement ;
2. items transformés vers le shape attendu par la liste/history/explorer ;
3. labels et champs d'affichage uniquement s'ils appartiennent au contrat produit, pas au provider.

### Règles

1. un item history ne récupère pas des agrégats de session par effet de bord ;
2. `is_ranked`, `is_pve`, `outcome` restent optionnels si le canonique les a laissés absents ;
3. l'OpenAPI ne doit pas forcer artificiellement une valeur par défaut là où le canonique portait une absence.

## Match View : adaptation recommandée

### Entrée

`MatchDetail` canonique, complété si nécessaire par des enrichissements produit explicitement distincts.

### Découpage conseillé

1. bloc résumé du match ;
2. scoreboard/participants ;
3. blocs teams ;
4. bloc skill ;
5. timeline/events ;
6. bloc film ;
7. warnings/limitations.

### Règles

1. si `skill` est partiel, l'écran match view ne doit pas être bloqué ;
2. si `events` sont `best_effort`, l'adaptateur doit le rendre visible dans un bloc warnings ou limitations si le contrat le prévoit ;
3. les dérivés produit comme killer/victim consolidé ou weapon reconciliation finale restent dans une couche explicitement distincte si le produit les expose.

## Career / Setup / Profil : adaptation recommandée

### Entrées possibles

1. `CareerProgression` ;
2. `CustomizationSnapshot` ;
3. `PlayerIdentity`.

### Règles

1. les assets carrière et customization doivent être présentés avec un wording produit stable ;
2. aucune granularité cosmétique fine n'est obligatoirement remontée tant qu'elle n'est pas utile à un écran précis ;
3. les adaptateurs évitent de dupliquer partout la logique de fallback sur labels ou images.

## Limitations et warnings

Les limitations canoniques doivent être traitées comme des informations de premier rang.

### Politique recommandée

1. limitation globale de titre -> bloc bootstrap `halo.limitations` ;
2. limitation locale à un match -> bloc warnings/limitations de l'endpoint match concerné ;
3. limitation locale à un joueur/profil -> bloc warnings/limitations du parcours concerné.

### À éviter

1. convertir une limitation en simple log serveur invisible ;
2. déduire côté frontend qu'une donnée est "cassée" sans signal explicite ;
3. mélanger limitations provider et erreurs métier produit dans une même liste sans typage clair.

## Nullabilité et valeurs par défaut

| Cas | Règle |
|-----|-------|
| champ canonique absent | le contrat produit l'omet ou le rend nullable, il ne l'invente pas |
| liste connue mais vide | renvoyer une liste vide |
| capability non supportée | la masquer ou la signaler explicitement selon le contrat parcours |
| donnée dérivée non calculée | ne pas la remplacer par `0`, `false` ou `"unknown"` sans décision de contrat explicite |

## Contrats OpenAPI : discipline attendue

L'OpenAPI final doit être dérivé des read models produit, pas du provider natif et pas non plus des structs canoniques brutes.

Pipeline recommandé :

```text
canonique Halo
  -> read model produit
    -> DTO OpenAPI
      -> JSON HTTP
```

Cette étape supplémentaire évite deux pièges :

1. figer accidentellement dans l'API des détails internes du canonique ;
2. multiplier des remappings ad hoc directement dans les handlers.

## Ce que les adaptateurs produit ne doivent pas faire

1. parser des payloads Halo Infinite ;
2. appeler directement des endpoints provider ;
3. porter la logique d'auth ou de retry ;
4. recalculer les analytics lourds ;
5. exposer des noms ou shapes SPNKr au frontend.

## Vérifications attendues avant implémentation

1. exemples documentés d'un bootstrap avec capability dégradée ;
2. exemples documentés d'un match detail partiel mais servable ;
3. séparation claire entre warnings provider et erreurs API ;
4. liste explicite des champs canoniques qui ne doivent jamais sortir tels quels en HTTP sans adaptation.

## Documents liés

1. [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md) pour la source canonique.
2. [HALO_INFINITE_CANONICAL_MAPPING.md](HALO_INFINITE_CANONICAL_MAPPING.md) pour la projection Halo Infinite -> canonique.
3. [HALO_BOOTSTRAP_CONTRACT.md](HALO_BOOTSTRAP_CONTRACT.md) pour la forme cible du bloc bootstrap Halo.
4. [HALO_GO_TYPE_BLUEPRINT.md](HALO_GO_TYPE_BLUEPRINT.md) pour les structs canoniques Go.
5. [PORTING_REFERENCE.md](PORTING_REFERENCE.md) pour le cadrage technique global.
