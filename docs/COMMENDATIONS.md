# Commendations — Architecture & Guide

French version: [FR/CITATIONS.md](FR/CITATIONS.md)

LevelUp implements a **DuckDB-first commendations system** (called “citations” in the French docs).

## Architecture

### Tables

| Table | Database | Description |
|-------|----------|-------------|
| `citation_mappings` | `metadata.duckdb` | Mapping rules (commendations definitions) |
| `match_citations` | player `stats.duckdb` | Computed values per match |

### `citation_mappings` schema

```sql
CREATE TABLE citation_mappings (
    citation_name_norm    TEXT PRIMARY KEY,
    citation_name_display TEXT NOT NULL,
    mapping_type          TEXT NOT NULL,       -- medal | stat | award | custom
    medal_id              BIGINT,
    medal_ids             TEXT,
    stat_name             TEXT,
    award_name            TEXT,
    award_category        TEXT,
    custom_function       TEXT,
    confidence            TEXT,                -- high | medium | low
    notes                 TEXT,
    enabled               BOOLEAN DEFAULT TRUE,
    created_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

Note: the `enabled` column replaces the legacy JSON exclusion file (`halo5_commendations_exclude.json`).
`load_mappings()` filters with `WHERE enabled IS NOT FALSE`.

### v5 compatibility (Shared Matches)

`CitationEngine` supports reading from `shared_matches.duckdb`:

- `shared_db_path`: auto-detected (`data/warehouse/shared_matches.duckdb`)
- `load_match_medals()`: reads `shared.medals_earned` (filtered by XUID) when available
- `load_match_stats()` / `load_match_df()`: reads `shared.match_participants` + `shared.match_registry`
- `load_match_awards()`: remains local (`personal_score_awards`)
- v4 fallback: if the shared DB does not exist, reads from local tables

### `match_citations` schema

```sql
CREATE TABLE match_citations (
    match_id           TEXT NOT NULL,
    citation_name_norm TEXT NOT NULL,
    value              INTEGER NOT NULL,
    PRIMARY KEY (match_id, citation_name_norm)
);
CREATE INDEX idx_match_citations_name ON match_citations(citation_name_norm);
```

## The 14 core commendations

| Normalized name | Type | Source | Origin |
|-----------------|------|--------|--------|
| `pilote` | medal | ID 3169118333 | existing |
| `ecrasement` | medal | ID 221693153 | existing |
| `assistant` | stat | `assists` | existing |
| `bulldozer` | custom | `compute_bulldozer` | existing |
| `victoire au drapeau` | custom | `compute_wins_ctf` | existing |
| `seul contre tous` | custom | `compute_wins_firefight` | existing |
| `victoire en assassin` | custom | `compute_wins_slayer` | existing |
| `victoire en bases` | custom | `compute_wins_strongholds` | existing |
| `defenseur du drapeau` | award | `Flag Defense` | restored |
| `je te tiens` | award | `Flag Return` | restored |
| `sus au porteur du drapeau` | award | `Flag Carrier Kill` | restored |
| `partie prenante` | award | `Zone Defense` | restored |
| `a la charge` | award | `Zone Capture` | restored |
| `annexion forcee` | custom | `compute_annexion_forcee` | restored |

## Data flow

```text
Sync match → backfill_citations() → CitationEngine.compute_and_store_for_match()
                                          ↓
                               match_citations (INSERT OR REPLACE)
                                          ↓
                               CitationEngine.aggregate_for_display()
                                          ↓
                               render_h5g_commendations_section() → UI
```

## Backfill CLI

```bash
# Incremental backfill for one player
python scripts/backfill_data.py --player YourGamertag --citations

# Force full recompute
python scripts/backfill_data.py --player YourGamertag --citations --force-citations

# All players
python scripts/backfill_data.py --all --citations
```

## Add a new commendation

1. Define the rule in `scripts/create_citation_mappings_table.py`:
   - `medal`: set `medal_id` (BIGINT)
   - `stat`: set `stat_name` (a column from match stats)
   - `award`: set `award_name` (value from `personal_score_awards.award_name`)
   - `custom`: set `custom_function` (name in `CUSTOM_FUNCTIONS`)

2. For `custom`, implement the function in `src/analysis/citations/custom_rules.py` and register it.

3. Run:

```bash
python scripts/create_citation_mappings_table.py
```

4. Backfill:

```bash
python scripts/backfill_data.py --all --citations --force-citations
```

5. Enable/disable a commendation by editing `citation_mappings.enabled` in `metadata.duckdb`.

## Python API

```python
from src.analysis.citations.engine import CitationEngine

engine = CitationEngine(db_path="data/players/YourGamertag/stats.duckdb", xuid="12345")

mappings = engine.load_mappings()
totals = engine.aggregate_for_display()
filtered = engine.aggregate_for_display(match_ids=["m1", "m2", "m3"])
engine.compute_and_store_for_match("match-uuid-123")
```

## FAQ

**How do I change an existing rule?**
Update `create_citation_mappings_table.py`, re-run it, then backfill with `--force-citations`.

**Disk impact?**
Roughly 14 rows per match in `match_citations` (one per commendation with value > 0).

**How do I diagnose issues?**

```bash
python scripts/diagnose_citations.py
```
