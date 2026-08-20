# HALO_INFINITE_CANONICAL_MAPPING.md — Mapping Halo Infinite vers le modèle canonique

> Document de préparation à l'implémentation.
> Il décrit comment le futur provider `haloinfinite` devra projeter ses payloads natifs vers le modèle canonique Halo, sans mélanger cette étape avec les contrats produit ou les analytics LevelUp.

## Rôle du document

Ce document fixe la discipline de mapping entre :

1. les payloads ou DTOs natifs Halo Infinite ;
2. les types canoniques définis dans [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md) ;
3. les limitations connues à reporter explicitement plutôt qu'à masquer.

Il sert à éviter deux dérives classiques :

1. laisser les payloads SPNKr dicter la forme du produit ;
2. faire porter au mapper Halo Infinite des calculs métier qui appartiennent à LevelUp.

## Ce document ne définit pas

1. les DTO OpenAPI côté handlers ;
2. les tables DuckDB ;
3. les calculs dérivés comme sessions, performance score, LUSR final, citations ou réconciliations finales d'armes ;
4. la gestion détaillée du cache auth ou du rate limit.

## Position dans la chaîne de transformation

```text
payloads natifs Halo Infinite
  -> DTOs provider haloinfinite
    -> mapper haloinfinite -> canonical
      -> services produit LevelUp
        -> adaptateurs bootstrap / OpenAPI
```

Le mapper `haloinfinite` produit donc des objets canoniques exploitables par le produit.
Il ne doit pas produire directement des réponses HTTP LevelUp.

## Emplacement cible recommandé

```text
apps/go-api/
  internal/
    halo/
      titles/
        haloinfinite/
          dto/
          mapper.go
          mapper_identity.go
          mapper_match.go
          mapper_career.go
          mapper_film.go
```

## Principes de mapping

1. Chaque mapper part d'un DTO provider explicite, jamais d'un `map[string]any` diffus.
2. Une valeur absente côté provider reste absente dans le canonique ; elle n'est pas synthétisée pour faire plaisir au produit.
3. Toute donnée partielle ou fragile doit être accompagnée d'une `limitation` explicite si elle change le comportement attendu.
4. Les enums et identifiants canoniques sont décidés côté canonique, pas copiés mécaniquement depuis les champs 343i.
5. Le mapper Halo Infinite ne calcule pas les dérivés métier propres à LevelUp.

## Pipeline de mapping recommandé

### Étape 1. Parse natif -> DTO provider

Le provider Halo Infinite parse les réponses HTTP vers des DTOs stables internes au titre.

Règle :

les champs au nom ou au shape fragile restent confinés à ces DTOs.

### Étape 2. Normalisation locale au titre

Avant le mapping canonique, le provider peut normaliser :

1. structures imbriquées irrégulières ;
2. timestamps ;
3. identifiants externes ;
4. regroupements mineurs nécessaires au titre.

Cette étape reste technique. Elle ne doit pas déjà créer des concepts produit.

### Étape 3. DTO Halo Infinite -> types canoniques

Le mapper produit les objets définis dans [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md).

### Étape 4. Limitations et capabilities

Quand une surface est absente ou partielle :

1. la capability map reste la source de vérité globale ;
2. le mapper peut ajouter des `limitations` au niveau objet quand le gap est contextuel à un match, un joueur ou un film précis.

## Règles transverses de projection

| Sujet | Règle |
|------|-------|
| `xuid` | transporté comme string canonique, sans imposer de conversion numérique dans le mapper |
| temps | normalisés en UTC avant création des objets canoniques |
| listes | slice vide si la collection est connue mais vide ; `null` si la surface n'est pas disponible |
| URLs | uniquement si le provider fournit une ressource utile au produit |
| labels | le mapper peut propager un label natif, mais l'i18n LevelUp reste en aval |
| score/skill | pas de calcul dérivé ; seulement projection des valeurs réellement fournies |

## Mapping par objet canonique

### 1. Title Runtime Context

Pour Halo Infinite, la projection minimale est :

1. `title_key = halo_infinite` ;
2. `provider_key = spnkr` pendant la transition, puis `native_go` si le provider Go devient autonome ;
3. `capabilities` issues de [HALO_INFINITE_CAPABILITY_MAP.md](HALO_INFINITE_CAPABILITY_MAP.md) ;
4. `limitations` réservées aux gaps utiles au consommateur produit.

Règle :

le mapper ne reconstruit pas les capabilities à partir d'une réponse HTTP ponctuelle. La map vient d'une configuration/version documentaire assumée.

### 2. Player Identity

Sources Halo Infinite possibles : lookup gamertag, lookup XUID, bootstrap profile, roster de match.

Projection canonique :

1. `xuid` obligatoire dès que l'identité est considérée comme résolue ;
2. `gamertag` repris tel qu'exposé par la surface native ;
3. `gamertag_normalized` autorisé comme normalisation locale purement technique ;
4. `service_tag`, `avatar_url`, `emblem_url` seulement si la surface provider les livre proprement ;
5. `is_bot` uniquement si l'information existe réellement.

À ne pas faire :

1. inférer un bot par heuristique fragile ;
2. injecter des alias DuckDB ou des règles produit dans le mapper titre ;
3. rendre obligatoire une image cosmétique pour considérer l'identité valide.

### 3. Asset Reference

Les assets Halo Infinite doivent converger vers un seul type canonique, quel que soit leur endpoint source.

Règles :

1. `asset_kind` est déterminé par le mapper à partir du contexte du DTO ;
2. `asset_id` reste l'identifiant source stable ;
3. `default_label` peut reprendre un label natif connu ;
4. `labels` ne sont remplis que si une vraie table de labels est disponible ;
5. `icon_url` reste optionnel.

Le mapper n'a pas à garantir une i18n complète ; il garantit une référence cohérente.

### 4. Match History Page et Match History Item

Sources : endpoints d'historique du titre.

Projection :

1. `start` et `count` reflètent la pagination demandée au provider ;
2. `items` contiennent uniquement les champs utiles à la liste produit ;
3. `playlist`, `map`, `game_variant` sont des `AssetReference` ;
4. `is_ranked` et `is_pve` ne sont renseignés que si la classification est fiable ; sinon `null`.

À exclure du mapper history :

1. agrégats de session ;
2. performance score ;
3. enrichissements issus de la DB joueur ;
4. citations ou awards calculés.

### 5. Match Detail

Le `MatchDetail` canonique Halo Infinite doit être construit par composition de surfaces natives, pas par un seul objet omniscient.

Assemblage typique :

1. détail principal du match ;
2. roster participants ;
3. blocs team/score ;
4. snapshot skill si la surface existe pour ce match ;
5. events ou hints de timeline si disponibles ;
6. film reference si disponible.

Règles :

1. l'absence du bloc skill ne doit pas invalider tout le `MatchDetail` ;
2. l'absence de timeline complète ne doit pas empêcher la restitution du scoreboard ;
3. un `MatchDetail` partiel peut être servi avec `limitations` explicites.

### 6. Match Participant

Projection par participant :

1. identité obligatoire ;
2. stats principales uniquement si la surface native les expose clairement ;
3. champs numériques laissés à `null` si absents ;
4. aucun recalcul de ratio ou de score dérivé dans le mapper.

Règle pratique :

si le provider donne une statistique sous forme agrégée et non canonique, le mapper choisit soit une projection explicite, soit l'absence. Il n'invente pas un champ canoniquement proche par approximation.

### 7. Team Snapshot

Pour Halo Infinite, `TeamSnapshot` reste léger :

1. `team_id` ;
2. `score` ;
3. `mmr` si présent nativement ;
4. `participants_xuids` si la composition d'équipe est fiable.

Le mapper ne calcule pas de métrique d'équipe composite propre à LevelUp.

### 8. Match Skill Snapshot

Sources : surfaces natives skill/MMR du match.

Règles :

1. seules les valeurs explicitement livrées sont projetées ;
2. les champs souvent manquants, comme certaines espérances secondaires, restent `null` ;
3. aucune transformation vers le LUSR produit n'est faite ici.

Si la surface est partielle :

1. le bloc `skill` peut être présent mais incomplet ;
2. une `limitation` doit rappeler que la capability est `degrade`.

### 9. Match Event Stream

Halo Infinite ne doit pas promettre plus qu'il ne sait faire proprement.

Règles :

1. les événements natifs confirmés peuvent être projetés avec une confiance forte ;
2. les événements dérivés d'une source fragile, comme certains films/chunks, doivent être marqués `best_effort` ou `derived` ;
3. le mapper n'émet pas une vérité killer/victim autoritaire si la source ne la garantit pas.

Conséquence :

la matrice killer/victim produit reste une dérivation en aval, pas un objet canonique garanti par Halo Infinite.

### 10. Career Progression

Sources : endpoints carrière + metadata de rangs.

Projection :

1. `current_rank` et `next_rank` comme `AssetReference` ;
2. `current_xp` si disponible ;
3. `history` uniquement si la surface native ou la reconstruction documentaire le permet sans fiction.

Le mapper ne reconstruit pas un historique de carrière complet à partir d'indices incomplets.

### 11. Customization Snapshot

Projection minimale utile :

1. identité joueur ;
2. `armor_core` si connu ;
3. `spartan_image_url` et `emblem_url` si exposés ;
4. `raw_components` seulement si le produit a un besoin concret de ce détail.

La granularité cosmétique fine ne doit pas contaminer tous les objets canoniques.

### 12. Film Reference

Pour Halo Infinite, la règle documentaire doit rester prudente.

Projection autorisée :

1. `film_available` ;
2. `film_id` si connu ;
3. `chunk_manifest` ou équivalent si accessible ;
4. `limitations` si l'accès film ne permet pas certaines restitutions attendues.

Projection interdite au niveau canonique :

1. présenter l'extraction d'armes via films comme une vérité native ;
2. présenter une reconstitution killer/victim comme capacité autoritaire du provider ;
3. faire dépendre la validité du match du succès de lecture d'un chunk.

## Gaps connus Halo Infinite à reporter proprement

| Surface | Décision de mapping |
|--------|---------------------|
| `match.skill.snapshot` | objet partiel autorisé + limitation si certaines métriques manquent |
| `film.weapon_extraction` | pas de promesse native ; si exposé plus tard, marquer la confiance et la fragilité |
| `match.weapon_breakdown_native` | ne pas fabriquer d'objet canonique natif correspondant |
| `combat.killer_victim_authoritative` | rester hors du mapping canonique provider |

## Ce que le mapper Halo Infinite ne doit jamais faire

1. joindre DuckDB pour compléter silencieusement une réponse provider ;
2. injecter des labels UI ou du wording OpenAPI ;
3. recalculer les analytics produit ;
4. convertir un manque de donnée en zéro silencieux ;
5. laisser des noms de champs SPNKr fuir dans les types canoniques.

## Vérifications attendues avant implémentation

1. corpus d'exemples par surface native ;
2. exemples documentés de `MatchHistoryPage`, `MatchDetail`, `CareerProgression` et `FilmReference` partiels ;
3. règle claire pour chaque champ canonique : source native, transformation admise, nullabilité, limitation éventuelle.

## Documents liés

1. [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md) pour la définition des objets cibles.
2. [HALO_INFINITE_CAPABILITY_MAP.md](HALO_INFINITE_CAPABILITY_MAP.md) pour la surface réellement supportée par `halo_infinite`.
3. [HALO_GO_TYPE_BLUEPRINT.md](HALO_GO_TYPE_BLUEPRINT.md) pour la forme des types Go canoniques.
4. [HALO_BOOTSTRAP_CONTRACT.md](HALO_BOOTSTRAP_CONTRACT.md) pour la projection produit du contexte Halo.
5. [HALO_PRODUCT_CONTRACT_ADAPTERS.md](HALO_PRODUCT_CONTRACT_ADAPTERS.md) pour la phase suivante de transformation côté produit.
6. [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md) pour le rattachement au programme.
