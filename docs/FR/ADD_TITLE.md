# Ajouter un nouveau jeu (titre) à LevelUp

English version: [../ADD_TITLE.md](../ADD_TITLE.md)

> **Architecture** : title-aware v7 — arborescence `data/titles/<slug>/`

LevelUp est conçu pour supporter plusieurs jeux. Ce guide couvre la procédure complète
pour intégrer un nouveau titre, de l'enregistrement dans le code jusqu'au premier
démarrage du serveur.

---

## Démarrage rapide — commande automatisée

```bash
# Minimum : nom du jeu uniquement (slug déduit)
levelup add-title --name "Halo MCC"

# Avec toutes les options
levelup add-title \
  --name "Halo MCC" \
  --slug halo_mcc \
  --capabilities matchmaking,media,ranked \
  --xbox-id 976923 \
  --steam-id 976730
```

La commande :
1. Déduit le slug depuis le nom (`"Halo MCC"` → `halo_mcc`)
2. Crée `data/titles/<slug>/warehouse/` et `data/titles/<slug>/players/`
3. Crée et initialise `shared_pve.duckdb` si `firefight` est dans `--capabilities`
4. Ajoute la section `"<slug>"` vide dans `db_profiles.json`
5. Affiche le snippet Go à coller dans `registry.go` (nécessite un `make build` ensuite)

La création des fichiers DuckDB et les migrations de schéma se font automatiquement
au prochain démarrage du serveur.

---

## Étapes — vue d'ensemble

| Étape | Action | Qui |
|------:|--------|-----|
| 1 | Enregistrer le descripteur dans `registry.go` + `make build` | **Manuel** |
| 2 | Créer les répertoires disque | Automatisé par `add-title` |
| 3 | Initialiser `shared_pve.duckdb` (si Firefight) | Automatisé par `add-title --capabilities firefight` |
| 4 | Ajouter la section dans `db_profiles.json` | Automatisé par `add-title` |
| 5 | Démarrer le serveur (création DuckDB + migrations) | Automatique au démarrage |
| 6 | (Optionnel) Pré-remplir les référentiels `metadata.duckdb` | Manuel selon le titre |

---

## Étape 1 — Enregistrer le titre dans `registry.go`

**Fichier** : `apps/go-api/internal/domain/title/registry.go`

Ajouter un appel `Register(...)` dans la fonction `NewRegistry()` :

```go
r.Register(&TitleDescriptor{
    Slug:     "halo_mcc",                  // slug unique, minuscules, underscores
    Name:     "Halo: The Master Chief Collection",
    Provider: "halo_mcc",
    Status:   StatusComingSoon,            // active | coming_soon | archived
    Capabilities: []Capability{
        CapMatchmaking, CapMedia,
    },
    IsDefault:   false,
    XboxTitleID: "976923",                 // depuis le catalogue Xbox
    SteamAppID:  "976730",                 // depuis Steam (chaîne vide si N/A)
})
```

### Capabilities disponibles

| Constante       | Signification                               |
|-----------------|---------------------------------------------|
| `CapMatchmaking`| Stats matchmaking classé/social             |
| `CapFirefight`  | Mode co-op PvE / Firefight                  |
| `CapForge`      | Support maps et modes personnalisés         |
| `CapMedia`      | Screenshots et clips vidéo                  |
| `CapRanked`     | CSR / classement compétitif                 |
| `CapCareer`     | Progression de rang de carrière             |

Ne déclarer que les capabilities réellement disponibles pour le titre — les services
vérifient `HasCapability(...)` avant d'activer les fonctionnalités associées.

### Effet du champ `Status`

| Status         | Comportement                                                             |
|----------------|--------------------------------------------------------------------------|
| `active`       | Titre entièrement activé, résolu par le middleware de routage            |
| `coming_soon`  | Enregistré mais pas encore routé (intégration en cours)                  |
| `archived`     | Accès lecture seule, aucun nouveau sync                                  |

Quel que soit le statut, le titre est **enregistré** dans le `Registry` et ses
chemins sont résolus par `PathResolver`. `ValidateTitle(slug)` retourne `nil` pour
tous les statuts.

---

## Étape 2 — Créer la structure de répertoires

**Automatisé par `levelup add-title`.**

Si vous créez le titre manuellement sans la commande :

```bash
mkdir -p data/titles/<slug>/warehouse
mkdir -p data/titles/<slug>/players
```

### Pourquoi ces répertoires sont nécessaires

`duckdb.OpenReadWrite(path)` crée un fichier `.duckdb` s'il n'existe pas, mais
uniquement si le **répertoire parent existe déjà**. Si
`data/titles/<slug>/warehouse/` est absent, le serveur échouera à ouvrir ou créer
toute base de données pour ce titre.

### Arborescence résultante

```
data/
└── titles/
    └── <slug>/
        ├── warehouse/          ← les fichiers DuckDB sont créés ici au démarrage
        │   ├── metadata.duckdb
        │   ├── shared_matches_v2.duckdb
        │   └── shared_social.duckdb
        └── players/            ← un sous-répertoire par joueur après le premier sync
```

---

## Étape 3 — Ajouter les joueurs dans `db_profiles.json`

**La section du titre est ajoutée automatiquement par `levelup add-title`.**
Les joueurs sont ensuite ajoutés manuellement dans cette section.

**Fichier** : `db_profiles.json` (racine du repo)

Ajouter une entrée joueur dans la section du titre :

```json
{
  "version": "3.0",
  "profiles": {
    "halo_infinite": { ... },
    "halo_mcc": {
      "MonGamertag": {
        "db_path":        "data/titles/halo_mcc/players/MonGamertag/stats.duckdb",
        "xuid":           "2533274800000000",
        "waypoint_player": "MonGamertag"
      }
    }
  }
}
```

`cfg.LoadPlayers(titleSlug)` navigue directement vers `profiles[titleSlug]`, donc la
clé **doit correspondre exactement** au slug enregistré à l'étape 1.

Le sous-répertoire `players/<gamertag>/` et le fichier `stats.duckdb` sont créés
automatiquement lors du premier sync du joueur via `RunPlayerMigrations`.

---

## Étape 4 — Démarrer le serveur

Au démarrage, `runMigrations` dans `cmd/server/main.go` exécute les migrations dans
l'ordre suivant :

| Base de données             | Auto-créée ? | Notes                                                            |
|-----------------------------|:------------:|------------------------------------------------------------------|
| `metadata.duckdb`           | Oui          | `OpenReadWrite` crée le fichier si absent                        |
| `shared_matches_v2.duckdb`  | Oui          | Idem                                                             |
| `shared_social.duckdb`      | Oui          | Idem                                                             |
| `shared_pve.duckdb`         | **Non**      | Les migrations ne s'appliquent que si le fichier existe (`os.Stat`) |

Chaque migration est tracée dans une table `schema_migrations` à l'intérieur de la
base cible. Une migration déjà appliquée **ne s'exécute jamais deux fois**
(idempotence garantie).

### Support PvE / Firefight

`shared_pve.duckdb` est créé automatiquement par `add-title` si `firefight` est
dans `--capabilities`. Si vous bootstrappez le titre manuellement :

```bash
# Avec le CLI DuckDB
duckdb data/titles/<slug>/warehouse/shared_pve.duckdb ".quit"
```

Le serveur appliquera les migrations PvE automatiquement au démarrage.

---

## Étape 5 — Router les requêtes vers le nouveau titre

Le middleware `TitleExtractor` résout le titre actif pour chaque requête API selon
la priorité suivante :

1. **Header `X-LevelUp-Title`** — si le slug est enregistré, il est utilisé
2. **Session courante** (`CurrentTitleSlug`) — persistée côté serveur
3. **Fallback** — `halo_infinite` (slug par défaut)

Pour cibler le nouveau titre depuis un client ou `curl` :

```bash
curl -H "X-LevelUp-Title: halo_mcc" \
     http://localhost:8000/api/v1/players/MonGamertag/pages/home
```

Aucune modification de routeur n'est nécessaire — toutes les routes
`/api/v1/players/{player_slug}/...` sont title-aware via le middleware et
`ResolvePlayer`.

---

## Étape 6 — (Optionnel) Pré-remplir les données de référence

`metadata.duckdb` contient les tables de référence (labels d'armes, rangs de
carrière, noms de cartes, etc.). Pour un nouveau titre, ces tables démarrent vides.

L'alimentation de ces référentiels dépend du titre — consulter `scripts/` pour
les importeurs disponibles et suivre la documentation propre à chaque jeu.

---

## Checklist de validation

Après avoir suivi les étapes ci-dessus, vérifier la configuration avec les commandes existantes :

```bash
# Diagnostic global de l'intégrité des bases
levelup healthcheck

# Vérification Gate (tables, vues, migrations)
levelup gate-check --gamertag MonGamertag
```

Ou inspecter manuellement chaque point :

- [ ] `registry.Get("<slug>")` retourne un descripteur non-nil (rebuild effectué)
- [ ] `data/titles/<slug>/warehouse/` existe et contient les fichiers `.duckdb`
- [ ] `db_profiles.json` a l'entrée correcte sous `profiles["<slug>"]`
- [ ] Les tables `schema_migrations` sont remplies dans chaque base de données
- [ ] `GET /api/v1/bootstrap` retourne le titre dans la liste `titles`
- [ ] Une requête joueur avec `X-LevelUp-Title: <slug>` retourne HTTP 200

---

## Référence : résolution des chemins

Tous les chemins d'un titre sont résolus par `title.NewPathResolver(repoRoot)` :

| Méthode                        | Chemin résolu                                           |
|--------------------------------|---------------------------------------------------------|
| `TitleDataDir(slug)`           | `data/titles/<slug>/`                                   |
| `WarehouseDir(slug)`           | `data/titles/<slug>/warehouse/`                         |
| `SharedDBPath(slug)`           | `…/warehouse/shared_matches_v2.duckdb`                  |
| `MetadataDBPath(slug)`         | `…/warehouse/metadata.duckdb`                           |
| `SharedPVEDBPath(slug)`        | `…/warehouse/shared_pve.duckdb`                         |
| `SharedSocialDBPath(slug)`     | `…/warehouse/shared_social.duckdb`                      |
| `PlayerDir(slug, gamertag)`    | `…/players/<gamertag>/`                                 |
| `PlayerDBPath(slug, gamertag)` | `…/players/<gamertag>/stats.duckdb`                     |

Aucun `filepath.Join(repoRoot, "data", ...)` n'est autorisé en dehors de
`PathResolver` — toujours utiliser ces méthodes.
