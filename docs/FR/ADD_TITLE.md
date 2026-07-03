# Ajouter un nouveau jeu (titre) à LevelUp

English version: [../ADD_TITLE.md](../ADD_TITLE.md)

> **Architecture** : title-aware v7 — arborescence `data/titles/<slug>/`, registre des
> titres piloté par config (`config/titles/<slug>/`). Voir
> [ADR 0008](../adr/0008-db-schema-multi-title-and-xuid-global.md) (isolation par chemin
> FS, pas de colonne `title_id`) et
> [ADR 0025](../adr/0025-title-agnostic-minimal-viable-window.md) (refactor title-agnostic).

LevelUp supporte plusieurs jeux. Ce guide est la procédure complète, de bout en bout —
du scaffolding CLI jusqu'au câblage de l'adapter qui sert réellement les données. Pour
les fondations transverses (types canoniques, adapters, manifests i18n, wrappers ECharts)
référencées plus bas, voir [FOUNDATIONS_GUIDE.md](FOUNDATIONS_GUIDE.md).

---

## Comment un titre est enregistré (à lire en premier)

Il existe **deux mécanismes d'enregistrement**, et le canonique est piloté par config :

1. **Titre par défaut built-in** — `halo_infinite` est câblé en dur dans `NewRegistry()`
   (`apps/go-api/internal/domain/title/registry.go`). Byte-identique et robuste même
   sans config. **Vous n'y touchez pas pour un nouveau titre.**

2. **Piloté par config (le chemin de tout titre additionnel)** — au boot,
   `cmd/server/main.go` appelle :

   ```go
   title.SetDefaultRegistry(title.NewRegistryFromConfig(cfg.RepoRoot, slog.Default()))
   ```

   `NewRegistryFromConfig` construit d'abord le registre built-in, puis
   `LoadTitlesIntoRegistry` scanne `config/titles/*/` et enregistre chaque dossier
   portant un `title.toml`. **Déposer `config/titles/<slug>/title.toml` suffit à
   enregistrer un titre additionnel — zéro recompilation du registre.**

> La commande `levelup add-title` affiche un snippet Go pour `registry.go`, mais pour un
> titre **additionnel** la source de vérité est `title.toml`. Modifier `registry.go` ne
> sert qu'à déclarer le titre *par défaut*. Préférez la voie `title.toml` ; un manifeste
> invalide est logué et ignoré sans bloquer le boot du serveur.

---

## Démarrage rapide — scaffolder l'arborescence disque

`levelup` est le CLI d'ops (`apps/go-api/cmd/levelup`). Lancez-le comme binaire compilé
ou via `go run ./cmd/levelup` depuis `apps/go-api`.

```bash
# Minimum : nom du jeu uniquement (slug déduit du nom)
levelup add-title --name "Halo MCC"

# Avec toutes les options
levelup add-title \
  --name "Halo MCC" \
  --slug halo_mcc \
  --capabilities matchmaking,media,ranked \
  --xbox-id 976923 \
  --steam-id 976730
```

**Flags** (d'après `cmd_title.go`) :

| Flag             | Requis | Défaut               | Notes                                              |
|------------------|:------:|----------------------|----------------------------------------------------|
| `--name`         | **Oui**| —                    | Nom complet du jeu, ex. `"Halo MCC"`               |
| `--slug`         | Non    | déduit du `--name`   | Minuscules `[a-z][a-z0-9_]*[a-z0-9]` ; ne peut pas être `halo_infinite` |
| `--capabilities` | Non    | `matchmaking,media`  | Capabilities coarse séparées par virgule (voir plus bas) |
| `--xbox-id`      | Non    | vide                 | Xbox Title ID (présence watcher + achievements)    |
| `--steam-id`     | Non    | vide                 | Steam App ID                                       |

La commande :
1. Déduit le slug depuis le nom (`"Halo MCC"` → `halo_mcc`).
2. Crée `data/titles/<slug>/warehouse/` et `data/titles/<slug>/players/`.
3. Crée `apps/web/public/titles/<slug>/` (dossier images header frontend).
4. Crée et initialise `shared_pve.duckdb` **uniquement si** `firefight` est dans `--capabilities`.
5. Ajoute la section `"<slug>"` vide dans `db_profiles.json` (atomique, via le store dbprofiles).
6. Affiche le snippet `registry.go` et le rappel images frontend.

Elle n'écrit **pas** `config/titles/<slug>/` — vous créez le manifeste et les mappings
vous-même (Étapes 1–3). La création des fichiers DuckDB et les migrations de schéma se
font au prochain démarrage du serveur.

---

## Étapes — vue d'ensemble

| Étape | Action | Qui |
|------:|--------|-----|
| 0 | Scaffolder dirs + section `db_profiles.json` | Automatisé par `levelup add-title` |
| 1 | Écrire `config/titles/<slug>/title.toml` (descripteur) | **Manuel** |
| 2 | Déclarer les capabilities (coarse dans `title.toml`, fines dans `mappings/capabilities.toml`) | **Manuel** |
| 3 | Écrire les mappings TOML (`fields`, `assets`, `outcomes`) | **Manuel** |
| 4 | (Si le titre sert des données) Écrire un `TitleDataAdapter` + l'enregistrer | **Manuel, Go** |
| 5 | Ajouter les joueurs dans `db_profiles.json` | **Manuel** |
| 6 | Démarrer le serveur (création DB + migrations + découverte au boot) | Automatique |
| 7 | Ajouter les images du hero banner | **Manuel** |
| 8 | (Optionnel) Pré-remplir les référentiels `metadata.duckdb` | Manuel selon le titre |

---

## Étape 1 — Écrire `config/titles/<slug>/title.toml`

Ce manifeste est le descripteur parsé par `LoadTitleManifest`
(`internal/domain/title/config_loader.go`). C'est l'équivalent d'un appel `Register(...)`
dans `registry.go`, externalisé en config.

```toml
# config/titles/halo_mcc/title.toml
[meta]
title_slug     = "halo_mcc"   # doit correspondre au nom du dossier
schema_version = 1            # doit être > 0

[title]
name          = "Halo: The Master Chief Collection"
provider      = "halo_mcc"
status        = "coming_soon" # active | coming_soon | archived
icon_url      = ""
xbox_title_id = "976923"      # présence watcher + achievements (vide si N/A)
steam_app_id  = "976730"      # chaîne vide si N/A
placement_matches = 0         # nb de matchs de placement classés ; 0 = défaut consommateur
csr_season_id     = ""        # overlay saison CSR fixe ; vide = fallback global

# Capabilities coarse — voir Étape 2.
capabilities = [
  "matchmaking",
  "media",
  "ranked",
]
```

Règles de validation appliquées par `LoadTitleManifestFromBytes` :

- `[meta].title_slug`, s'il est présent, **doit être égal au nom du dossier**.
- `[meta].schema_version` doit être `> 0`.
- `[title].name` est requis ; un `provider` manquant prend le slug par défaut.
- `[title].status` doit valoir `active | coming_soon | archived`.
- `[title].is_default` est **interdit** pour un titre additionnel (réservé à `halo_infinite`).
- Chaque entrée de `capabilities` doit être une capability coarse connue (Étape 2) — une
  inconnue échoue à la validation et le titre est ignoré (log `title_manifest_invalid`).

### Effet du `status`

| Status         | Comportement                                                             |
|----------------|--------------------------------------------------------------------------|
| `active`       | Titre entièrement activé, servi, provisionné au boot, adapters câblés     |
| `coming_soon`  | Découvert + listé dans le switcher (« bientôt »), mais **non servi** — `RequireActiveTitle` retourne `503 title_unavailable` |
| `archived`     | Exclu du switcher (`NonArchived()` le filtre)                            |

`ValidateTitle(slug)` réussit pour tout titre enregistré quel que soit le statut ; le
gating runtime des routes title-scoped passe par le middleware `RequireActiveTitle`
(`IsActive()` ⇒ `Status == active`), pas par le résolveur de titre.

---

## Étape 2 — Déclarer les capabilities

Il existe **deux vocabulaires de capabilities distincts** — ne pas les confondre :

### A. Capabilities coarse (`title.Capability`) — dans `title.toml`

Déclarées dans `[title].capabilities`, validées contre `knownCapabilities`
(`config_loader.go`). Elles gouvernent des **surfaces produit / middlewares** et
alimentent le switcher de titres. Valeurs connues (miroir des constantes `Cap*` de
`registry.go`) :

| Valeur                 | Signification                                                 |
|------------------------|---------------------------------------------------------------|
| `matchmaking`          | Stats matchmaking classé/social                               |
| `firefight`            | Co-op PvE / Firefight (déclenche la création de `shared_pve.duckdb`) |
| `forge`                | Maps et modes personnalisés                                   |
| `media`                | Screenshots et clips vidéo                                    |
| `ranked`               | CSR / classement compétitif                                  |
| `career`               | Progression de rang de carrière                              |
| `season_pass`          | Progression season pass / Battlepass                          |
| `asset.images`         | Miniatures Asset Drawer (maps & armes)                        |
| `achievements`         | Page achievements Xbox                                        |
| `engagement`           | Score d'engagement intra-match + coefficients                 |
| `lusr`                 | Rating interne LevelUp (LUSR v2)                              |
| `world.leaderboard`    | Classements mondiaux                                          |
| `native_kill_mechanics`| Mécaniques de kill natives (assassinats, compétences spartiate)|

Le flag `levelup add-title --capabilities` ne reconnaît que les six premières
(`matchmaking,firefight,forge,media,ranked,career`) pour générer le snippet `registry.go` ;
déclarez l'ensemble plus riche directement dans `title.toml`.

### B. Capabilities fines (`games.CapabilityKey`) — dans `mappings/capabilities.toml`

Elles gouvernent les **méthodes `Load*` du `TitleDataAdapter`** du titre (c.-à-d.
exactement quelles surfaces de données l'adapter est câblé à servir). Les 16 clés connues
sont des constantes de `internal/games/adapter.go` ; `AllCapabilityKeys()` est dans `internal/games/capabilities.go` :

`match.history`, `match.detail.core`, `match.skill.snapshot`, `career.progression`,
`career.rank_catalog`, `pve.firefight_stats`, `analytics.timeseries`,
`match.scoreboard.extra`, `citations.engine`, `engagement.score`,
`battlepass.progression`, `challenges.surface`, `match.events.timeline`,
`match.killfeed.per_kill`, `match.events.spatial`, `commendations.native`.

Chaque valeur vaut `supported | degraded | not_exposed` :

```toml
# config/titles/halo_mcc/mappings/capabilities.toml
[meta]
title_slug     = "halo_mcc"
schema_version = 1

[capabilities]
"match.history"      = "supported"   # seulement si LoadMatchSummaries est réellement câblé
"match.detail.core"  = "supported"
"career.progression" = "supported"
"analytics.timeseries" = "not_exposed"
# ... les clés restantes par défaut à not_exposed
```

> **Règle d'or** (de `halo_5/mappings/capabilities.toml`) : une clé ne peut être
> `supported` ou `degraded` **que si sa méthode `Load*` est réellement câblée**.
> `CapabilityMap.Has()` retourne vrai pour `supported`/`degraded` ; si la méthode est un
> stub, tout appel échouerait. Les surfaces non câblées restent `not_exposed`.

`CapabilityMapFromMappings` (`internal/games/capabilities.go`) valide chaque clé et statut
au boot et agrège les erreurs — une clé inconnue ou un statut invalide est rejeté.

---

## Étape 3 — Écrire les mappings TOML (couche sémantique)

Sous `config/titles/<slug>/mappings/`, à côté de `capabilities.toml` :

### `fields.toml` — définitions des métriques canoniques, unités, libellés, ordre

```toml
[meta]
title_slug     = "halo_mcc"
schema_version = 1

[fields.kills]
labels        = { en = "Kills", fr = "Éliminations" }
storage_unit  = "count"
display_unit  = "count"
format        = "integer"
display_order = 10
group         = "combat"

[fields.accuracy]
labels        = { en = "Accuracy", fr = "Précision" }
storage_unit  = "ratio"
display_unit  = "percent"
format        = "percent_2"
display_order = 40
group         = "combat"
```

### `assets.toml` — modes, tiers, etc. (libellés bilingues + ordre d'affichage)

```toml
[meta]
title_slug     = "halo_mcc"
schema_version = 1

[assets.mode.slayer]
labels = { en = "Slayer", fr = "Massacre" }
display_order = 10

[assets.mode.ctf]
labels = { en = "Capture the Flag", fr = "Capture du drapeau" }
display_order = 20
```

Les clés (`kind/id`) sont libres ; la validation exige des valeurs non vides et aucune
collision de `display_order`.

### `outcomes.toml` — libellés win/loss/tie/dnf + tokens de couleur sémantiques

```toml
[meta]
title_slug     = "halo_mcc"
schema_version = 1

[outcomes.win]
labels = { en = "Victory", fr = "Victoire" }
color_token = "outcome.positive"

[outcomes.loss]
labels = { en = "Defeat", fr = "Défaite" }
color_token = "outcome.negative"

[outcomes.tie]
labels = { en = "Tie", fr = "Égalité" }
color_token = "outcome.neutral"

[outcomes.dnf]
labels = { en = "DNF", fr = "Abandon" }
color_token = "outcome.neutral"
```

Utilisez des **tokens** de couleur sémantiques (jamais de hex brut) — voir
[ADR 0011](../adr/0011-canonical-vs-semantic-adapter-separation.md) pour la frontière
adapter canonical (data brute) vs semantic (i18n/libellés) vs asset-URL.

---

## Étape 4 — Adapters (seulement si le titre sert des données)

La couche adapter (`internal/games/`) projette la source native d'un titre sur le schéma
canonique inter-titres (`internal/games/canonical/`). Trois interfaces adapter existent
(`adapter.go`) :

| Interface              | Ce que vous écrivez pour un nouveau titre                        |
|------------------------|------------------------------------------------------------------|
| `TitleSemanticAdapter` | **Rien.** Le `GenericSemanticAdapter` partagé (`semantic_adapter.go`) enveloppe vos TOML — aucun code semantic par titre. |
| `TitleDataAdapter`     | **Un package Go** (ex. `internal/games/halo_mcc/`) projetant votre source (API live ou DuckDB) sur `canonical.*`. C'est le vrai travail. |
| `TitleAssetURLAdapter` | Optionnel — seulement si le naming d'URL d'assets diverge du défaut. |

### Câbler le data adapter

Les titres additionnels actifs sont câblés par `registerAdditionalTitles`
(`internal/api/server_titles_additional.go`), qui itère `titleRegistry.Active()` et
dispatche par slug via la map `additionalTitleRegistrars`. Ajoutez votre titre :

```go
var additionalTitleRegistrars = map[string]additionalTitleRegistrar{
    halo5.TitleSlug:    registerHalo5Adapters,
    halo_mcc.TitleSlug: registerHaloMCCAdapters, // votre nouveau registrar
}
```

Votre `registerHaloMCCAdapters` construit le `GenericSemanticAdapter` depuis les mapping
sets, convertit `capabilities.toml` via `CapabilityMapFromMappings`, construit votre
`TitleDataAdapter`, et enregistre les deux sur le `StaticResolver`
(`RegisterSemantic` / `RegisterData`) — voir `registerHalo5Adapters` comme référence.
Ce changement Go nécessite un rebuild (`make go-api-build`).

> Si un titre est `active` mais sans registrar, le serveur logue
> `additional_title_no_adapter_registrar` et le titre n'est pas servi. Un titre
> `coming_soon` n'a pas besoin d'adapter (il n'est jamais servi).

### Écritures de données : l'architecture Collect → Persist (ADR 0019)

Un nouveau titre qui synchronise des matchs **ne doit jamais écrire de lignes
per-match avec un `ExecContext` brut**. Toute écriture per-match sur une DB
partagée ou joueur passe par la couche `internal/persist`, qui est **INSERT-only**
— jamais d'`UPSERT` / `ON CONFLICT DO UPDATE` concurrent sur les tables critiques
(c'est ce qui a corrompu les index ART DuckDB en production, ADR 0019 / 0026) :

- **Collect** : agréger le match dans un `persist.MatchBatch` via
  `persist.NewBatchBuilder(...)` (`SetMatch` / `AddParticipants` / `AddMedals` /
  `AddMatchCSRs`).
- **Persist** : l'écrire avec `persist.NewSharedPersister(db).Persist(ctx, batch)`
  (partagé, atomique, idempotent — re-persister un `match_id` existant est un no-op),
  ou un `PlayerPersister` pour les enrichissements joueur / ratings per-match.
- **Tables append-only** (`match_skill_rank`, `match_csrs`, `player_csr_snapshots`,
  `pve_match_stats`, …) : écriture INSERT-only avec un `written_at`, **lecture
  EXCLUSIVEMENT via la vue `<table>_latest`** (une lecture brute sert des lignes
  périmées, ADR 0026).

Implémentation de référence pour la sync live d'un nouveau titre :
`internal/games/halo_5/livesync/csr_match.go` (CSR/skill per-match écrits via
`PlayerPersister.PersistPerMatchRating`). Hiérarchie : `games/<slug>/client.go`
(fetch) → `games/<slug>/livesync/*` (mapping vers les entrées persist) →
`internal/persist/*` (écriture). Ne jamais court-circuiter persist depuis la couche
client ou livesync.

---

## Étape 5 — Ajouter les joueurs dans `db_profiles.json`

La section vide du titre est créée par `add-title` ; les joueurs y sont ajoutés manuellement.

```json
{
  "version": "3.0",
  "profiles": {
    "halo_infinite": { },
    "halo_mcc": {
      "MonGamertag": {
        "db_path":         "data/titles/halo_mcc/players/MonGamertag/stats.duckdb",
        "xuid":            "2533274800000000",
        "waypoint_player": "MonGamertag"
      }
    }
  }
}
```

`cfg.LoadPlayers(titleSlug)` navigue directement vers `profiles[titleSlug]`, donc la clé
**doit correspondre exactement** au slug. Le sous-répertoire `players/<gamertag>/` et
`stats.duckdb` sont créés automatiquement lors du premier sync du joueur.

---

## Étape 6 — Démarrer le serveur

Au démarrage :

1. `NewRegistryFromConfig` découvre `config/titles/<slug>/title.toml` et l'enregistre
   (log `title_registered_from_config`).
2. Les migrations de schéma s'exécutent par base. `OpenReadWrite` crée un fichier
   `.duckdb` si le **répertoire parent existe** (d'où la nécessité que
   `data/titles/<slug>/warehouse/` soit déjà présent — `add-title` le crée).

| Base de données            | Auto-créée ? | Notes                                              |
|----------------------------|:------------:|----------------------------------------------------|
| `metadata.duckdb`          | Oui          | `OpenReadWrite` crée le fichier si absent           |
| `shared_matches_v2.duckdb` | Oui          | Idem                                               |
| `shared_social.duckdb`     | Oui          | Idem                                               |
| `shared_pve.duckdb`        | **Non**      | Migrations seulement si le fichier existe déjà      |

Les migrations sont tracées dans une table `schema_migrations` et idempotentes (jamais
exécutées deux fois). `shared_pve.duckdb` est créé par `add-title` quand `firefight` est
dans `--capabilities` ; pour le bootstrapper manuellement, ouvrez-le et fermez-le une fois
pour que le fichier existe.

### Router les requêtes vers le nouveau titre

Le middleware `TitleExtractor` résout le titre actif par requête :

1. **Header `X-LevelUp-Title`** — si le slug est enregistré, il est utilisé
2. **Session courante** (`CurrentTitleSlug`) — persistée côté serveur
3. **Fallback** — `halo_infinite`

```bash
curl -H "X-LevelUp-Title: halo_mcc" \
     http://localhost:8000/api/v1/players/MonGamertag/pages/home
```

Aucune modification de routeur n'est nécessaire — toutes les routes
`/api/v1/players/{player_slug}/...` sont title-aware. Note : un titre `active` sans
adapter câblé (Étape 4) est résolu mais retourne `503 title_unavailable` sur les routes
de données via `RequireActiveTitle`.

---

## Étape 7 — Ajouter les images du hero banner

`add-title` crée `apps/web/public/titles/<slug>/`. Y déposer les visuels header
(`.webp` ou `.png`), puis les référencer dans
`apps/web/src/features/home/HomeHeroBanner.tsx` dans `HEADER_IMAGES_BY_TITLE` :

```ts
const HEADER_IMAGES_BY_TITLE: Record<string, string[]> = {
  halo_infinite: [ /* … */ ],
  halo_mcc: [
    '/titles/halo_mcc/header-1.webp',
    '/titles/halo_mcc/header-2.png',
  ],
}
```

Si aucune image n'est fournie, le banner ne s'affiche pas silencieusement (fallback
tableau vide).

---

## Étape 8 — (Optionnel) Pré-remplir les données de référence

Les tables de référence de `metadata.duckdb` (labels d'armes, rangs de carrière, noms de
cartes, etc.) démarrent vides pour un nouveau titre. L'alimentation dépend du titre —
consulter les CLI d'ops disponibles (`apps/go-api/cmd/*`) et l'adapter du titre pour ce
qu'il attend.

---

## Ce qui dégrade gracieusement via les capabilities

Les capabilities font de la donnée manquante un état de premier ordre non fatal plutôt
qu'une erreur :

- Une méthode `Load*` d'un `TitleDataAdapter` retourne `ErrCapabilityNotSupported` quand
  sa capability fine (Étape 2.B) n'est pas `supported`/`degraded`. Les callers traitent
  cela comme un signal de dégradation, pas un échec.
- Les capabilities coarse (Étape 2.A) gouvernent des surfaces produit entières. Exemples
  observés dans le code : sans `season_pass` ⇒ l'onglet career/season-pass est masqué et
  dégrade en `FeatureUnavailable` ; sans `world.leaderboard` ⇒ la page leaderboard
  retourne un `200` vide ; sans `native_kill_mechanics` ⇒ le front masque les sections
  assassinats / compétences spartiate.
- Les manques sémantiques dégradent aussi : un `RankCatalog` vide ⇒ les consommateurs
  affichent le `rank_id` brut ; `Assets()`/`Outcomes()` à `nil` ⇒ fallback gracieux.

Déclarer uniquement ce que le titre sert réellement est donc la bonne façon de livrer une
intégration incrémentale : démarrer `coming_soon` avec tout `not_exposed`, puis basculer
les clés en `supported` à mesure que chaque méthode `Load*` est câblée.

---

## Checklist de validation

```bash
# Diagnostic global de l'intégrité des bases
levelup healthcheck

# Vérification Gate (tables, vues, migrations)
levelup gate-check --gamertag MonGamertag
```

Vérification manuelle :

- [ ] `config/titles/<slug>/title.toml` existe et est valide (boot log `title_registered_from_config`, pas `title_manifest_invalid`)
- [ ] `registry.Get("<slug>")` retourne un descripteur non-nil
- [ ] `config/titles/<slug>/mappings/{fields,assets,outcomes,capabilities}.toml` présents
- [ ] `data/titles/<slug>/warehouse/` existe et contient les fichiers `.duckdb`
- [ ] `db_profiles.json` a l'entrée sous `profiles["<slug>"]`
- [ ] Les tables `schema_migrations` sont remplies dans chaque base
- [ ] `GET /api/v1/bootstrap` liste le titre dans `titles`
- [ ] Pour un titre `active` : une requête joueur avec `X-LevelUp-Title: <slug>` retourne `200` (ou `503` si l'adapter n'est pas encore câblé)
- [ ] `apps/web/public/titles/<slug>/` contient au moins une image header référencée dans `HomeHeroBanner.tsx`

---

## Référence : résolution des chemins

Tous les chemins d'un titre sont résolus par `title.NewPathResolver(repoRoot)`
(`internal/domain/title/registry.go`) :

| Méthode                        | Chemin résolu                                |
|--------------------------------|----------------------------------------------|
| `TitleDataDir(slug)`           | `data/titles/<slug>/`                        |
| `WarehouseDir(slug)`           | `data/titles/<slug>/warehouse/`              |
| `SharedDBPath(slug)`           | `…/warehouse/shared_matches_v2.duckdb`       |
| `MetadataDBPath(slug)`         | `…/warehouse/metadata.duckdb`                |
| `SharedPVEDBPath(slug)`        | `…/warehouse/shared_pve.duckdb`              |
| `SharedSocialDBPath(slug)`     | `…/warehouse/shared_social.duckdb`           |
| `PlayerDir(slug, gamertag)`    | `…/players/<gamertag>/`                      |
| `PlayerDBPath(slug, gamertag)` | `…/players/<gamertag>/stats.duckdb`          |
| `GlobalXuidAliasesDBPath()`    | `data/global/xbox_aliases.duckdb` (global)   |

La base d'alias `xuid → gamertag` est **globale**, pas par titre (c'est une identité
Microsoft/Xbox, indépendante du titre — [ADR 0008](../adr/0008-db-schema-multi-title-and-xuid-global.md)).
Aucun `filepath.Join(repoRoot, "data", ...)` n'est autorisé hors de `PathResolver` —
toujours utiliser ces méthodes.
