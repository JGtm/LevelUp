# Plan : Versionnement metadata.duckdb & découplage JSON

> **Date** : 2026-02-21
> **Base** : branche `feature/v5.2`
> **Statut** : En attente d'implémentation

---

## Objectif

1. **Versionner `metadata.duckdb` dans Git** — BDD 100% structurelle (0 donnée perso)
2. **Migrer les JSON structurels en tables DuckDB** dans `metadata.duckdb` — une seule source de vérité
3. **Découpler l'UI des fichiers JSON** — l'app lit tout depuis DuckDB

---

## 1. Audit — Quoi versionner, quoi ignorer

### Bases DuckDB

| Base | Taille | Contenu | Données perso ? | Git ? |
|------|--------|---------|----------------|-------|
| `data/warehouse/metadata.duckdb` | ~1 MB | `citation_mappings` (55 règles). Futur : `career_ranks`, `playlists`, `maps`, etc. | **NON** | ✅ OUI |
| `data/warehouse/shared_matches.duckdb` | ~103 MB | matchs, joueurs, médailles, kills | **OUI** | ❌ NON |
| `data/warehouse/shared_pve.duckdb` | ~268 KB | xuid, match_id, kills PVE | **OUI** | ❌ NON |
| `data/players/*/stats.duckdb` | variable | enrichissements individuels | **OUI** | ❌ NON |

### Fichiers JSON

| Fichier | Taille | Contenu | Données perso ? | Déjà Git ? | Action |
|---------|--------|---------|----------------|------------|--------|
| `data/Playlist_modes_translations.json` | 43 KB | Trad playlists EN→FR | NON | ✅ oui | Conserver tel quel (candidat migration future) |
| `data/wiki/halo5_commendations_fr.json` | 139 KB | Commendations H5G | NON | ✅ oui | **Phase 3** : découpler l'UI → lire depuis `citation_mappings` |
| `data/wiki/halo5_commendations_exclude.json` | 3 KB | Exclusions citations | NON | ✅ oui | Supprimer quand Phase 3 terminée |
| `data/cache/career_ranks_metadata.json` | 376 KB | 272 rangs officiels Halo | NON | ❌ non | **Phase 2** : migrer en table `career_ranks` |
| `data/xuid_aliases.json` | 176 KB | Mapping XUID→gamertag | **OUI** | ❌ non | ❌ Ne pas versionner |
| `data/cache/player_assets/` | 1.4 MB | Avatars joueurs | **OUI** | ❌ non | ❌ Ne pas versionner |
| `data/cache/career_ranks/*.png` | 15.8 MB | 271 icônes rangs | NON | ❌ non | ⚠️ Trop lourd pour Git, script de download |

---

## 2. Phase 1 — Versionner `metadata.duckdb`

> **Statut** : ✅ Quasi-complète — `.gitignore` déjà mis à jour, fichier déjà staged.
> Il reste uniquement à committer.

### 2.1 `.gitignore`

```gitignore
# AVANT
data/warehouse/*.duckdb

# APRÈS
data/warehouse/*.duckdb
!data/warehouse/metadata.duckdb
```

> **Fait** : l'exception `!data/warehouse/metadata.duckdb` est déjà présente dans le `.gitignore`.

### 2.2 Git add

```bash
# metadata.duckdb est déjà staged (git status: A  data/warehouse/metadata.duckdb)
git commit -m "chore(data): versionner metadata.duckdb (référentiel structurel)"
```

### 2.3 Garde-fou

`metadata.duckdb` ne doit **jamais** contenir de colonnes `xuid`, `gamertag`, `match_id`
à caractère personnel. Tables autorisées = référentiels uniquement.

### 2.4 Règle de mise à jour

Après chaque modification de `metadata.duckdb` (populate, migration, ajout de table) →
**committer le fichier**.

---

## 3. Phase 2 — Migrer `career_ranks_metadata.json` → table DuckDB

### 3.1 Contexte

Le fichier JSON contient 272 rangs officiels Halo Infinite (XP requis, titres EN,
tiers, chemins d'icônes). Aucune donnée joueur.

Actuellement lu par :
- `src/ui/career_ranks.py` → `_load_ranks_metadata()` → `json.loads()`
- `scripts/migration/migrate_metadata_to_duckdb.py` → `load_career_ranks()` ⚠️ **HORS SERVICE POST-V5.1** (migration SQLite → DuckDB archivée, cité ici pour référence de la structure JSON uniquement)

Traductions FR : **pas** dans le JSON, dans des dicts Python
(`_CAREER_RANK_TITLE_FR`, `_CAREER_RANK_TIER_FR` dans `career_ranks.py`).
Ces dicts **restent dans le code Python** après la migration — ils n'ont pas à être migrés en DuckDB.

### 3.2 Schéma table `career_ranks`

```sql
CREATE TABLE IF NOT EXISTS career_ranks (
    rank_id               INTEGER PRIMARY KEY,   -- 1 à 272
    title_en              VARCHAR NOT NULL,       -- "Recruit", "Private", "Hero"…
    subtitle_en           VARCHAR,                -- "", "Silver", "Gold"…
    tier                  VARCHAR,                -- "", "1", "2", "3"
    tier_type             VARCHAR,                -- "Bronze", "Silver", "Gold", "Platinum", "Diamond", "Onyx"
    grade                 INTEGER,                -- RankGrade (1, 2, 3)
    xp_required           INTEGER NOT NULL,       -- XP cumulé pour ce rang
    icon_path             VARCHAR,                -- RankIcon (widget petit)
    large_icon_path       VARCHAR,                -- RankLargeIcon (célébration)
    adornment_icon_path   VARCHAR                 -- RankAdornmentIcon (nameplate)
);
```

### 3.3 Script de peuplement

Option A : ajouter dans `scripts/populate_citation_mappings.py` (renommer en `populate_metadata.py`)
Option B : script séparé `scripts/populate_career_ranks.py`

```python
def populate_career_ranks(conn: duckdb.DuckDBPyConnection, json_path: Path) -> int:
    """Peuple career_ranks depuis le JSON (migration initiale)."""
    conn.execute(DDL_CAREER_RANKS)
    with open(json_path, encoding="utf-8") as f:
        data = json.load(f)
    rows = [
        (r["Rank"], r["RankTitle"]["value"],
         r.get("RankSubTitle", {}).get("value", ""),
         r.get("RankTier", {}).get("value", ""),
         r.get("TierType", ""),
         r.get("RankGrade", 1),
         r.get("XpRequiredForRank", 0),
         r.get("RankIcon", ""),
         r.get("RankLargeIcon", ""),
         r.get("RankAdornmentIcon", ""))
        for r in data["Ranks"]
    ]
    conn.executemany(
        """INSERT INTO career_ranks VALUES (?,?,?,?,?,?,?,?,?,?)
           ON CONFLICT (rank_id) DO UPDATE SET
               title_en = EXCLUDED.title_en,
               subtitle_en = EXCLUDED.subtitle_en,
               tier = EXCLUDED.tier,
               tier_type = EXCLUDED.tier_type,
               grade = EXCLUDED.grade,
               xp_required = EXCLUDED.xp_required,
               icon_path = EXCLUDED.icon_path,
               large_icon_path = EXCLUDED.large_icon_path,
               adornment_icon_path = EXCLUDED.adornment_icon_path
        """,
        rows,
    )
    return len(rows)
```

### 3.4 Refactoring `src/ui/career_ranks.py`

**Avant** (lecture JSON) :
```python
@lru_cache(maxsize=1)
def _load_ranks_metadata() -> dict[str, Any]:
    path = _get_metadata_path()  # data/cache/career_ranks_metadata.json
    return json.loads(path.read_text(encoding="utf-8"))

@lru_cache(maxsize=1)
def _build_ranks_lookup() -> dict[int, CareerRankInfo]:
    metadata = _load_ranks_metadata()
    ranks = metadata.get("Ranks", [])
    # ... parse chaque entry JSON ...
```

**Après** (lecture DuckDB) :
```python
@lru_cache(maxsize=1)
def _build_ranks_lookup() -> dict[int, CareerRankInfo]:
    """Construit le lookup depuis metadata.duckdb."""
    import duckdb
    db_path = _repo_root() / "data" / "warehouse" / "metadata.duckdb"
    conn = duckdb.connect(str(db_path), read_only=True)
    try:
        rows = conn.execute(
            "SELECT rank_id, title_en, subtitle_en, tier, xp_required, large_icon_path "
            "FROM career_ranks ORDER BY rank_id"
        ).fetchall()
    finally:
        conn.close()
    lookup = {}
    for rank_id, title, subtitle, tier, xp_req, large_icon in rows:
        lookup[rank_id] = CareerRankInfo(
            rank_number=rank_id, title=title,
            subtitle=subtitle or None, tier=tier or None,
            xp_required=xp_req, icon_path_remote=large_icon or "",
        )
    return lookup
```

**À supprimer** :
- `_load_ranks_metadata()`
- `_get_metadata_path()`
- `is_metadata_available()` (vérifie l'existence du JSON → à adapter : vérifier la table `career_ranks` dans DuckDB, ou simplement supprimer si le fallback n'est plus nécessaire)

**À conserver** :
- `_CAREER_RANK_TITLE_FR`, `_CAREER_RANK_TIER_FR` — dicts Python de traduction, inchangés
- `_GRADE_TO_ROMAN` — inchangé
- `format_career_rank_label_fr()` — inchangée
- `get_rank_icon_path()`, `get_rank_icon_url()`, `count_cached_icons()` — inchangées (gestion locale des PNG)

### 3.5 Nettoyage

- Le JSON reste sur disque comme archive (pas supprimé), mais **n'est plus lu** par l'app
- Retirer toute référence à `career_ranks_metadata.json` dans le code applicatif

---

## 4. Phase 3 — Découpler les citations du JSON H5G

### 4.1 Contexte

L'UI des citations (`src/ui/commendations.py`) lit actuellement :
1. `data/wiki/halo5_commendations_fr.json` → images, catégories, descriptions, tiers
2. `metadata.duckdb` → `citation_mappings` → règles actives

L'objectif est d'enrichir `citation_mappings` pour que le JSON ne soit plus nécessaire.

### 4.2 Colonnes à ajouter à `citation_mappings`

```sql
ALTER TABLE citation_mappings ADD COLUMN image_path VARCHAR;
ALTER TABLE citation_mappings ADD COLUMN category VARCHAR;
ALTER TABLE citation_mappings ADD COLUMN description VARCHAR;
ALTER TABLE citation_mappings ADD COLUMN tier_targets VARCHAR;  -- CSV : "10,20,30,50,100"
```

### 4.3 Peuplement

Hardcoder les 4 champs dans les tuples de `populate_citation_mappings.py`.
Référence complète : `docs/CITATIONS_REFERENCE.md` (55 citations avec tiers, images, catégories).

### 4.4 Refactoring `src/ui/commendations.py`

**Avant** : charge le JSON → build items list → match par nom normalisé avec les citations DB
**Après** : charge `citation_mappings` (avec image_path, tiers, catégorie) → build items directement

Fonctions à modifier :
- `load_h5g_commendations_json()` (⚠️ nom réel, pas `_load_h5g_commendations()`) → remplacer par requête DuckDB sur `citation_mappings`
- `_img_src()` → lire `image_path` depuis la citation (colonne DuckDB), plus depuis le JSON
- `_compute_mastery_display()` → lire `tier_targets` CSV depuis DuckDB, plus les `tiers[]` du JSON
- Supprimer le matching par nom normalisé (`_normalize_name` + filtre `enabled_norms`) — plus nécessaire, c'est la même source

Fonctions à supprimer :
- `load_h5g_commendations_json()` — plus utilisée
- `load_h5g_commendations_exclude()` — plus utilisée (enabled géré par `citation_mappings.enabled`)
- `_sanitize_item_name()` — contournement pour noms JSON corrompus, inutile avec DuckDB
- `_looks_english()`, `_prefer_parenthetical_fr()` — nettoyage de descriptions EN/FR du JSON, inutiles avec DuckDB
- `DEFAULT_H5G_JSON_PATH`, `DEFAULT_H5G_EXCLUDE_PATH` — constantes de chemin JSON

Dicts à migrer ou conserver :
- `_H5G_TITLE_OVERRIDES_FR` — décider : soit migrer les overrides dans la colonne `citation_name_display` de `citation_mappings`, soit garder le dict comme post-traitement UI
- `_H5G_DESC_OVERRIDES_FR` — idem, soit migrer dans la colonne `description` de `citation_mappings`, soit garder comme post-traitement

### 4.5 Nettoyage

- `data/wiki/halo5_commendations_fr.json` → conserver comme archive, ne plus lire
- `data/wiki/halo5_commendations_exclude.json` → supprimer (les enabled/disabled sont dans la DB via `citation_mappings.enabled`)
- `_sanitize_item_name()` → supprimer (contournement pour noms JSON corrompus)

---

## 5. Phase 4 — Migrer `Playlist_modes_translations.json` → DuckDB

### 5.1 Contexte

`data/Playlist_modes_translations.json` (43 KB, versionné Git) contient deux sections
structurellement distinctes :

| Section | Entrées | Clé naturelle | Usage actuel |
|---------|---------|--------------|--------------|
| `playlists` | 14 | `uuid` (UUID complet Halo) | `translations.py` → dict statique `PLAYLIST_FR` |
| `modes` | ~400 | `en` (nom anglais complet) | `checkbox_filter.py` → `_load_mode_categories()` (**dead code** : définie ligne 18, jamais appelée nulle part) |

**La catégorisation réelle** des modes dans l'UI se fait via `_infer_category()` +
dict statique `PREFIX_TO_CATEGORY` — indépendamment du JSON.

**Valeur réelle de la migration** :
1. `playlist_translations` → `MetadataResolver.resolve("playlist", uuid)` peut résoudre
   les noms de playlists par UUID (actuellement retourne `None` : la table `playlists` de
   `populate_metadata_from_discovery.py` n'est pas peuplée)
2. `mode_translations` → référentiel structurel EN→FR pour usage futur ; occasion de
   nettoyer la dead code dans `checkbox_filter.py`

### 5.2 Schéma des deux tables

```sql
-- 14 playlists officielles — clé = UUID Halo complet
CREATE TABLE IF NOT EXISTS playlist_translations (
    uuid    VARCHAR PRIMARY KEY,  -- ex: "2825d417-93e6-4366-98f9-839a2dc41fe4"
    name_en VARCHAR NOT NULL,     -- ex: "Big Team Battle"
    name_fr VARCHAR NOT NULL      -- ex: "Grande bataille en équipe"
);

-- ~400 modes / pairs — clé = nom anglais complet issu de l'API
CREATE TABLE IF NOT EXISTS mode_translations (
    name_en  VARCHAR PRIMARY KEY, -- ex: "Arena:Attrition on Catalyst"
    name_fr  VARCHAR NOT NULL,    -- ex: "Arène : Attrition"
    category VARCHAR NOT NULL     -- ex: "Arena", "BTB", "Ranked", "Firefight"
);
```

### 5.3 Script `scripts/populate_playlist_translations.py`

Pattern identique à `populate_citation_mappings.py` : idempotent, `--reset`, stats finales.

```python
METADATA_DB = Path("data/warehouse/metadata.duckdb")
JSON_PATH   = Path("data/Playlist_modes_translations.json")

DDL_PLAYLIST = """CREATE TABLE IF NOT EXISTS playlist_translations (
    uuid VARCHAR PRIMARY KEY, name_en VARCHAR NOT NULL, name_fr VARCHAR NOT NULL
)"""

DDL_MODE = """CREATE TABLE IF NOT EXISTS mode_translations (
    name_en VARCHAR PRIMARY KEY, name_fr VARCHAR NOT NULL, category VARCHAR NOT NULL
)"""

UPSERT_PLAYLIST = """
    INSERT INTO playlist_translations VALUES (?, ?, ?)
    ON CONFLICT (uuid) DO UPDATE SET
        name_en = EXCLUDED.name_en, name_fr = EXCLUDED.name_fr
"""

UPSERT_MODE = """
    INSERT INTO mode_translations VALUES (?, ?, ?)
    ON CONFLICT (name_en) DO UPDATE SET
        name_fr = EXCLUDED.name_fr, category = EXCLUDED.category
"""

def ensure_schema(conn) -> None:
    """Crée les deux tables si absentes (idempotent)."""

def populate(conn, data: dict) -> tuple[int, int]:
    """UPSERT depuis le JSON. Retourne (n_playlists, n_modes)."""

def cleanup_obsolete(conn, data: dict) -> tuple[int, int]:
    """Supprime les lignes absentes du JSON. Retourne (n_pl_del, n_mode_del)."""

def main() -> None:
    """argparse --reset, ouvre le JSON, appelle ensure_schema + populate + cleanup + stats."""
```

**Usage** : `python scripts/populate_playlist_translations.py [--reset]`

### 5.4 Mise à jour `MetadataResolver`

**Fichier** : `src/data/sync/metadata_resolver.py`

Ajouter `playlist_translations` comme table candidate de fallback pour le type `playlist` :

```python
# Avant
"playlist": ["playlists"],

# Après
"playlist": ["playlists", "playlist_translations"],
```

`MetadataResolver` détecte dynamiquement les colonnes :
- Colonne ID : cherche `asset_id` puis **`uuid`** → trouve `uuid` dans `playlist_translations` ✅
- Colonne nom : cherche `public_name`, **`name_fr`**, `name_en`, `name` → trouve `name_fr` ✅

Résultat : `resolver.resolve("playlist", "2825d417-93e6-4366-98f9-839a2dc41fe4")`
retourne `"Grande bataille en équipe"`.

Priorité : si la table `playlists` (peuplée par discovery API) existe, elle est
interrogée en premier. `playlist_translations` sert de fallback.

### 5.5 Nettoyage `src/ui/components/checkbox_filter.py`

Supprimer la dead code (jamais appelée) :

```python
# ── À SUPPRIMER ──────────────────────────────────────────────────────────────
_MODE_CATEGORIES_CACHE: dict[str, str] | None = None  # ligne 15

def _load_mode_categories() -> dict[str, str]:          # lignes 18-48
    ...
```

**À conserver intacts** : `_infer_category()`, `PREFIX_TO_CATEGORY`, `CATEGORY_FR`,
`_translate_category()` — ils ne lisent pas le JSON et fonctionnent parfaitement.

### 5.6 `translations.py` — pas modifié

`PLAYLIST_FR` reste un dict statique Python (14 playlists, données stables).
La table `playlist_translations` bénéficie au `MetadataResolver`, pas à `translations.py`.

### 5.7 Tests `tests/test_playlist_translations.py`

```python
import json
from pathlib import Path
import duckdb
import pytest
from src.data.sync.metadata_resolver import MetadataResolver


FAKE_JSON = {
    "playlists": [
        {"en": "Big Team Battle", "fr": "Grande bataille en équipe",
         "uuid": "2825d417-0000-0000-0000-000000000000"},
        {"en": "Quick Play", "fr": "Partie rapide",
         "uuid": "1b1691dc-0000-0000-0000-000000000000"},
    ],
    "modes": [
        {"en": "Arena:Slayer", "fr": "Arène : Assassin", "category": "Arena"},
        {"en": "BTB:CTF", "fr": "BTB : Capture du drapeau", "category": "BTB"},
    ],
}


@pytest.fixture
def fake_json(tmp_path: Path) -> Path:
    """Écrit un JSON minimal dans un répertoire temporaire."""
    p = tmp_path / "Playlist_modes_translations.json"
    p.write_text(json.dumps(FAKE_JSON), encoding="utf-8")
    return p


@pytest.fixture
def empty_metadata_db(tmp_path: Path) -> Path:
    """Base metadata.duckdb vide (sans tables)."""
    db_path = tmp_path / "metadata.duckdb"
    duckdb.connect(str(db_path)).close()
    return db_path


class TestPopulatePlaylistTranslations:

    def test_ensure_schema_idempotent(self, empty_metadata_db):
        """ensure_schema() appelé deux fois ne lève pas d'erreur."""

    def test_populate_playlists_count(self, empty_metadata_db, fake_json, monkeypatch):
        """populate() insère exactement N playlists issues du JSON."""

    def test_populate_modes_count(self, empty_metadata_db, fake_json, monkeypatch):
        """populate() insère exactement N modes issus du JSON."""

    def test_populate_playlist_values(self, empty_metadata_db, fake_json, monkeypatch):
        """Valeurs uuid, name_en, name_fr correctement insérées."""

    def test_populate_mode_values(self, empty_metadata_db, fake_json, monkeypatch):
        """Valeurs name_en, name_fr, category correctement insérées."""

    def test_upsert_idempotent(self, empty_metadata_db, fake_json, monkeypatch):
        """Double appel populate() : même nombre de lignes, pas de doublons."""

    def test_cleanup_removes_obsolete_playlists(self, empty_metadata_db, fake_json, monkeypatch):
        """cleanup_obsolete() supprime les playlists absentes du JSON."""

    def test_cleanup_removes_obsolete_modes(self, empty_metadata_db, fake_json, monkeypatch):
        """cleanup_obsolete() supprime les modes absents du JSON."""

    def test_reset_drops_and_recreates(self, empty_metadata_db, fake_json, monkeypatch):
        """--reset vide les tables puis les repeuple."""


class TestMetadataResolverWithPlaylistTranslations:

    @pytest.fixture
    def db_with_playlist_translations(self, tmp_path: Path) -> Path:
        db_path = tmp_path / "metadata.duckdb"
        conn = duckdb.connect(str(db_path))
        conn.execute(
            "CREATE TABLE playlist_translations "
            "(uuid VARCHAR PRIMARY KEY, name_en VARCHAR NOT NULL, name_fr VARCHAR NOT NULL)"
        )
        conn.execute(
            "INSERT INTO playlist_translations VALUES (?,?,?)",
            ["2825d417-0000-0000-0000-000000000000", "Big Team Battle",
             "Grande bataille en équipe"],
        )
        conn.close()
        return db_path

    def test_resolve_playlist_by_uuid(self, db_with_playlist_translations):
        """MetadataResolver résout un UUID playlist depuis playlist_translations."""
        resolver = MetadataResolver(db_with_playlist_translations)
        name = resolver.resolve("playlist", "2825d417-0000-0000-0000-000000000000")
        assert name == "Grande bataille en équipe"
        resolver.close()

    def test_resolve_unknown_uuid_returns_none(self, db_with_playlist_translations):
        """UUID inconnu retourne None sans lever d'exception."""
        resolver = MetadataResolver(db_with_playlist_translations)
        assert resolver.resolve("playlist", "ffffffff-0000-0000-0000-000000000000") is None
        resolver.close()

    def test_playlists_table_has_priority(self, tmp_path):
        """Si table playlists existe, elle est interrogée avant playlist_translations."""
        db_path = tmp_path / "metadata.duckdb"
        conn = duckdb.connect(str(db_path))
        # Créer playlists (discovery) avec valeur différente
        conn.execute(
            "CREATE TABLE playlists "
            "(asset_id VARCHAR PRIMARY KEY, version_id VARCHAR, public_name VARCHAR)"
        )
        conn.execute(
            "INSERT INTO playlists VALUES (?,?,?)",
            ["2825d417-0000-0000-0000-000000000000", "1", "BTB (Discovery)"],
        )
        # Créer playlist_translations avec valeur différente
        conn.execute(
            "CREATE TABLE playlist_translations "
            "(uuid VARCHAR PRIMARY KEY, name_en VARCHAR NOT NULL, name_fr VARCHAR NOT NULL)"
        )
        conn.execute(
            "INSERT INTO playlist_translations VALUES (?,?,?)",
            ["2825d417-0000-0000-0000-000000000000", "Big Team Battle", "Grande bataille"],
        )
        conn.close()
        resolver = MetadataResolver(db_path)
        name = resolver.resolve("playlist", "2825d417-0000-0000-0000-000000000000")
        assert name == "BTB (Discovery)"  # playlists a la priorité
        resolver.close()
```

### 5.8 Nettoyage post-migration

| Élément | Action |
|---------|--------|
| `data/Playlist_modes_translations.json` | ✅ Conserver (source de vérité versionnée) |
| `_load_mode_categories()` + `_MODE_CATEGORIES_CACHE` | 🗑️ Supprimer (dead code) |
| `import json` dans `checkbox_filter.py` | 🗑️ Supprimer si plus aucun usage |
| `translations.py` | ✅ Inchangé |

---

## 6. Tables autorisées dans `metadata.duckdb`

| Table | Lignes | Statut |
|-------|--------|--------|
| `citation_mappings` | 55 | ✅ Existe |
| `career_ranks` | 272 | Phase 2 |
| `playlist_translations` | 14 | Phase 4 |
| `mode_translations` | ~400 | Phase 4 |
| `playlists` | ~50 | Phase 5+ (discovery API) |
| `maps` | ~30 | Phase 5+ (discovery API) |
| `game_variants` | ~20 | Phase 5+ (discovery API) |
| `playlist_map_mode_pairs` | ~100 | Phase 5+ (discovery API) |

**INTERDIT** : toute table contenant `xuid`, `gamertag`, `match_id` ou données joueur.

---

## Critères de succès

- [ ] `metadata.duckdb` versionné dans Git
- [ ] `.gitignore` : exception `!data/warehouse/metadata.duckdb`
- [ ] Table `career_ranks` (272 lignes) dans `metadata.duckdb`
- [ ] `career_ranks.py` lit DuckDB, ne lit plus le JSON
- [ ] `citation_mappings` enrichie (image_path, category, description, tier_targets)
- [ ] `commendations.py` lit DuckDB, ne lit plus le JSON H5G
- [ ] Table `playlist_translations` (14 lignes) dans `metadata.duckdb`
- [ ] Table `mode_translations` (~400 lignes) dans `metadata.duckdb`
- [ ] `MetadataResolver` résout les playlists par UUID depuis `playlist_translations`
- [ ] Dead code `_load_mode_categories()` supprimé de `checkbox_filter.py`
- [ ] Suite pytest complète : OK
- [ ] Aucune régression UI sur les pages Citations, Progression et filtres de modes

---

## Ordre d'implémentation recommandé

| Étape | Phase | Effort | Risque |
|-------|-------|--------|--------|
| 1 | Phase 1 — git commit metadata.duckdb | 2 min | Nul |
| 2 | Phase 2 — `career_ranks` en BDD | 1h | Faible |
| 3 | Phase 3 — Découplage citations JSON | 2-3h | Moyen |
| 4a | Phase 4 — Script `populate_playlist_translations.py` | 30 min | Nul |
| 4b | Phase 4 — `MetadataResolver` : ajouter `playlist_translations` | 5 min | Nul |
| 4c | Phase 4 — Supprimer dead code `checkbox_filter.py` | 5 min | Nul |
| 4d | Phase 4 — Tests `test_playlist_translations.py` | 45 min | Nul |

---

**Dernière mise à jour** : 2026-02-21
**Auteur** : Revue humaine + Claude
