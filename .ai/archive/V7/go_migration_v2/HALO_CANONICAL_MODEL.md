# HALO_CANONICAL_MODEL.md — Modèle canonique Halo pour le portage Go

> Document de cadrage pour la Phase 0 du chantier Go.
> Il définit le contrat canonique entre un provider de titre Halo et le reste du produit LevelUp.

## Rôle du document

Ce document sert à figer un point souvent implicite dans le code Python actuel :

1. ce qui relève du provider Halo lui-même ;
2. ce qui relève du contrat produit LevelUp ;
3. ce qui relève ensuite des calculs métier, de DuckDB et des vues UI.

Le but n'est pas de recopier les payloads SPNKr ni les tables DuckDB.
Le but est de définir le plus petit modèle stable qui permette :

1. d'alimenter les parcours produit ;
2. d'isoler les spécificités Halo Infinite ;
3. d'accepter plus tard un autre titre Halo sans réécrire toute la façade produit.

## Ce que ce modèle n'est pas

Ce document ne définit pas :

1. le schéma OpenAPI final des routes LevelUp ;
2. le schéma DuckDB interne ;
3. les payloads bruts SPNKr ou Waypoint ;
4. les métriques dérivées purement produit comme sessions, performance score, LUSR final, citations, bitmask de backfill ou vues matérialisées.

## Position dans l'architecture

```text
Provider de titre Halo
  -> mapping canonique Halo
    -> services produit LevelUp
      -> endpoints LevelUp / persistance DuckDB / analytics métier
```

Le modèle canonique est donc la frontière entre :

1. les données natives du jeu ;
2. le langage métier du produit.

Tant que le chantier Go reste interne au repo, ce modèle peut vivre conceptuellement dans `go-api/internal/halo/canonical/`.
Si le socle provider Halo est ensuite extrait en package public, seule la partie réellement provider-agnostic et sans dépendance produit pourra migrer vers `pkg/haloapi/`.

## Principes de modélisation

1. Orientation produit : le modèle est organisé par concepts utiles au produit, pas par endpoints 343i.
2. Nullabilité explicite : une donnée absente ou partielle doit rester absente, jamais simulée par une valeur arbitraire.
3. Dégradation gracieuse : un objet partiel reste exploitable si le produit peut continuer à rendre proprement le reste.
4. Séparation stricte : le modèle canonique ne contient ni SQL, ni logique de cache, ni colonnes DuckDB, ni hypothèses UI spécifiques.
5. Variabilité isolée : les identifiants, endpoints, labels et règles propres à Halo Infinite restent dans le provider de titre, pas dans le contrat canonique.
6. Dérivés exclus : les calculs métier construits par LevelUp à partir des données natives restent hors du modèle canonique.

## Conventions de modélisation

| Règle | Décision |
|------|----------|
| Identifiant de joueur | `xuid` reste transporté comme chaîne canonique externe ; conversion numérique autorisée seulement en bordure d'implémentation |
| Temps | UTC explicite ; timestamps ou dates sérialisés en ISO-8601 côté contrat externe |
| Listes | liste vide quand la collection est connue mais vide ; `null` uniquement si la surface n'est pas disponible |
| Références d'assets | identifiants et labels regroupés dans des objets dédiés, pas en champs éparpillés |
| Champs dérivés | exclus tant qu'ils peuvent être recomputés côté produit |
| Erreurs de provider | représentées comme gaps ou limitations explicites, pas comme absence silencieuse |

## Modèle canonique cible

### 1. Title Runtime Context

Contexte minimal transmis au produit pour savoir sur quel titre et quel provider il tourne.

| Champ | Statut | Rôle |
|------|:------:|------|
| `title_key` | requis | identifiant stable du titre courant, ex. `halo_infinite` |
| `provider_key` | requis | identifiant stable du provider courant, ex. `spnkr`, puis `native_go` |
| `capabilities` | requis | capability map produit visible par le consommateur |
| `limitations` | optionnel | limitations connues importantes pour dégrader proprement certaines surfaces |

Ce contexte n'est pas un payload Waypoint. C'est un contrat produit pour le bootstrap et les services qui doivent connaître la portée réelle du provider.

### 2. Player Identity

Représente une identité joueur résolue de façon suffisamment stable pour bootstrap, history, explorer et setup.

| Champ | Statut | Rôle |
|------|:------:|------|
| `xuid` | requis | identifiant stable côté Xbox/Halo |
| `gamertag` | requis quand connu | libellé principal du joueur |
| `gamertag_normalized` | optionnel | forme normalisée pour recherche, matching et cache |
| `service_tag` | optionnel | donnée cosmétique si disponible |
| `emblem_url` | optionnel | ressource visuelle utilisable par le produit |
| `avatar_url` | optionnel | ressource visuelle utilisable par le produit |
| `is_bot` | optionnel | distinction produit utile sans dépendre des payloads bruts |
| `raw_refs` | optionnel | références externes minimales si une corrélation provider reste utile au debug |

### 3. Asset Reference

Objet canonique unique pour les assets Halo exposés au produit.

| Champ | Statut | Rôle |
|------|:------:|------|
| `asset_kind` | requis | `map`, `playlist`, `game_variant`, `career_rank`, `challenge_track`, etc. |
| `asset_id` | requis | identifiant source stable |
| `version_id` | optionnel | version de l'asset si le provider la distingue |
| `default_label` | optionnel | libellé par défaut utilisable sans i18n locale avancée |
| `labels` | optionnel | dictionnaire de labels localisés |
| `icon_url` | optionnel | vignette ou icône si disponible |

Le produit consomme des `AssetReference`, pas des couples ad hoc `map_id` / `map_name` disséminés partout.

### 4. Match History Page

Le contrat canonique d'historique doit être paginé dès l'origine.

| Champ | Statut | Rôle |
|------|:------:|------|
| `player` | requis | identité pour laquelle l'historique est demandé |
| `start` | requis | offset de pagination demandé |
| `count` | requis | taille de page demandée |
| `items` | requis | liste de `MatchHistoryItem` |
| `has_more` | optionnel | indicateur pratique si le provider sait le calculer proprement |
| `total_matches` | optionnel | total si connu sans coût ou ambiguïté excessive |

### 5. Match History Item

Version légère d'un match, suffisante pour historiser et décider si l'on charge ensuite le détail.

| Champ | Statut | Rôle |
|------|:------:|------|
| `match_id` | requis | identifiant du match |
| `started_at_utc` | requis | début du match |
| `duration_seconds` | optionnel | durée connue du match |
| `match_type` | optionnel | classement fonctionnel minimal |
| `playlist` | optionnel | `AssetReference` playlist |
| `map` | optionnel | `AssetReference` map |
| `game_variant` | optionnel | `AssetReference` mode/variant |
| `is_ranked` | optionnel | information directement utile au produit |
| `is_pve` | optionnel | distinction PvE / PvP |
| `outcome` | optionnel | résultat du joueur si disponible dans la surface légère |

### 6. Match Detail

Objet canonique central du provider Halo vers le produit.

| Champ | Statut | Rôle |
|------|:------:|------|
| `match_id` | requis | identifiant du match |
| `started_at_utc` | requis | début du match |
| `ended_at_utc` | optionnel | fin du match si calculable proprement |
| `playlist` | optionnel | `AssetReference` playlist |
| `map` | optionnel | `AssetReference` map |
| `game_variant` | optionnel | `AssetReference` variant |
| `is_ranked` | optionnel | sémantique produit |
| `is_pve` | optionnel | sémantique produit |
| `participants` | requis | roster canonique du match |
| `teams` | optionnel | scores ou blocs d'équipe utiles |
| `skill` | optionnel | snapshot de skill du match |
| `events` | optionnel | timeline canonique best effort |
| `film` | optionnel | accès film ou chunks si disponible |
| `limitations` | optionnel | limitations ou gaps connus sur ce match |

Le détail de match ne suppose pas qu'une timeline complète, un breakdown d'armes ou une skill line parfaite existent toujours.

### 7. Match Participant

Participant d'un match dans le contrat canonique.

| Champ | Statut | Rôle |
|------|:------:|------|
| `identity` | requis | `PlayerIdentity` |
| `team_id` | optionnel | équipe du joueur |
| `rank_in_match` | optionnel | rang dans le scoreboard |
| `outcome` | optionnel | résultat pour ce joueur |
| `score` | optionnel | score principal |
| `kills` | optionnel | stat principale |
| `deaths` | optionnel | stat principale |
| `assists` | optionnel | stat principale |
| `accuracy` | optionnel | précision globale |
| `damage_dealt` | optionnel | dégâts infligés |
| `damage_taken` | optionnel | dégâts subis |
| `shots_fired` | optionnel | tirs effectués |
| `shots_hit` | optionnel | tirs touchés |
| `headshot_kills` | optionnel | headshots totaux |
| `max_killing_spree` | optionnel | spree max |
| `personal_score` | optionnel | score personnel si surface distincte |

### 8. Team Snapshot

Vue légère d'équipe pour garder la structure canonique du match sans dupliquer la logique produit.

| Champ | Statut | Rôle |
|------|:------:|------|
| `team_id` | requis | identifiant équipe |
| `score` | optionnel | score d'équipe |
| `mmr` | optionnel | valeur skill d'équipe si connue |
| `participants_xuids` | optionnel | roster réduit |

### 9. Match Skill Snapshot

Bloc canonique minimal pour les données de skill natives du provider.

| Champ | Statut | Rôle |
|------|:------:|------|
| `player_xuid` | requis | joueur concerné |
| `team_mmr` | optionnel | MMR équipe |
| `enemy_mmr` | optionnel | MMR adverse |
| `kills_expected` | optionnel | espérance fournie par le provider |
| `kills_stddev` | optionnel | dispersion associée |
| `deaths_expected` | optionnel | espérance fournie par le provider |
| `deaths_stddev` | optionnel | dispersion associée |
| `assists_expected` | optionnel | souvent absente ; ne jamais supposer sa présence |
| `assists_stddev` | optionnel | souvent absente ; ne jamais supposer sa présence |

Ce bloc est une entrée pour la logique produit. Il n'est pas le LUSR final du produit.

### 10. Match Event Stream

Flux canonique d'événements utilisable pour la timeline et les analyses en aval.

| Champ | Statut | Rôle |
|------|:------:|------|
| `event_type` | requis | type fonctionnel d'événement |
| `time_ms` | requis | offset temporel dans le match |
| `primary_actor_xuid` | optionnel | acteur principal si connu |
| `secondary_actor_xuid` | optionnel | autre acteur utile si connu |
| `confidence` | optionnel | `confirmed`, `best_effort`, `derived` |
| `raw_hint` | optionnel | hint technique minimal utile pour le debug |

La matrice killer/victim exacte n'appartient pas au modèle canonique ; elle est dérivée en aval à partir de la timeline et d'algorithmes produit.

### 11. Career Progression

Progression carrière canonique.

| Champ | Statut | Rôle |
|------|:------:|------|
| `player` | requis | identité joueur |
| `current_rank` | optionnel | `AssetReference` rang actuel |
| `current_xp` | optionnel | XP actuelle |
| `next_rank` | optionnel | rang suivant si connu |
| `history` | optionnel | liste d'entrées de progression historisées |

### 12. Customization Snapshot

Bloc cosmétique minimal, utile pour setup, profil et home.

| Champ | Statut | Rôle |
|------|:------:|------|
| `player` | requis | identité joueur |
| `armor_core` | optionnel | asset principal |
| `spartan_image_url` | optionnel | rendu visuel exploitable |
| `emblem_url` | optionnel | emblème |
| `raw_components` | optionnel | composants détaillés si vraiment utiles au produit |

### 13. Film Reference

Référence canonique à un film ou à ses chunks.

| Champ | Statut | Rôle |
|------|:------:|------|
| `match_id` | requis | rattachement produit |
| `film_available` | requis | disponibilité connue |
| `film_id` | optionnel | identifiant natif |
| `chunk_manifest` | optionnel | URL ou descripteur de téléchargement |
| `limitations` | optionnel | limites connues sur l'exploitabilité des chunks |

Le contrat n'implique pas que l'extraction d'armes ou le parsing binaire soient fiables ; il garantit seulement que la surface film est décrite explicitement.

### 14. Capability Gap / Provider Limitation

Le modèle canonique doit savoir porter ses absences.

| Champ | Statut | Rôle |
|------|:------:|------|
| `capability_key` | requis | capability concernée |
| `reason_code` | requis | cause normalisée de l'absence ou de la dégradation |
| `severity` | requis | `info`, `warning`, `blocking` |
| `message` | optionnel | message opérable pour logs ou debug |
| `retryable` | optionnel | si l'absence peut être retentée |

## Ce qui doit explicitement rester hors du modèle canonique

Ces éléments appartiennent à la logique LevelUp construite au-dessus du provider :

1. découpage en sessions ;
2. performance score ;
3. LUSR/CSR final du produit ;
4. citations et commendations calculées ;
5. killer/victim matrix consolidée ;
6. réconciliation d'armes finale pour les vues match ;
7. write lease, bitmask de backfill, migrations et vues matérialisées.

## Règles de mapping depuis le code Python actuel

| Surface Python actuelle | Sortie canonique cible |
|-------------------------|------------------------|
| `get_user_by_gamertag()` | `PlayerIdentity` |
| `get_users_by_id()` | liste de `PlayerIdentity` |
| `get_match_history()` | `MatchHistoryPage` + `MatchHistoryItem` |
| `get_match_data()` | `MatchDetail` |
| `get_skill_stats()` | `MatchSkillSnapshot` |
| `get_asset()` | `AssetReference` |
| `get_career_rank_progression()` | `CareerProgression` |
| `get_player_customization()` | `CustomizationSnapshot` |
| `get_film_by_match_id()` / `download_film_chunk()` | `FilmReference` + accès chunks |

## Décisions de cadrage déjà actées

1. Le modèle canonique est orienté produit, pas endpoint.
2. Il est préparé à plus d'un titre, mais ne doit pas inventer des concepts absents du produit actuel.
3. Il doit rester compatible avec un cadrage mono-titre `halo_infinite` tant qu'aucune autre source fiable n'existe.
4. Toute future extension multi-titre devra d'abord essayer d'exprimer ses données via ces objets avant d'introduire une nouvelle capability ou une nouvelle limitation.

## Prochaine étape attendue

Le document compagnon de ce modèle est [HALO_INFINITE_CAPABILITY_MAP.md](HALO_INFINITE_CAPABILITY_MAP.md), qui fixe quelles parties de ce contrat sont réellement supportées aujourd'hui pour `halo_infinite`.
