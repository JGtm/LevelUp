# Corpus de test — Joueur de référence

Généré le 2026-04-13 depuis le gamertag `Chocoboflor` (limit=500).

## Contenu

| Fichier | Description |
|---------|-------------|
| `stats.duckdb` | Enrichissements joueur (`player_match_enrichment`, etc.) |
| `shared_matches_v2.duckdb` | Sous-ensemble shared matches pour ce joueur |
| `metadata.duckdb` | Référentiels complets |
| `xuid.txt` | XUID du joueur |

## Statistiques d'extraction

| Table | Lignes |
|-------|--------|
| `shared.match_registry` | 364 |
| `shared.match_participants` | 3304 |
| `shared.highlight_events` | 77959 |
| `shared.medals_earned` | 7366 |
| `shared.weapon_kills` | 30062 |
| `shared.xuid_aliases` | 15370 |
| `player.player_match_enrichment` | 364 |
| `player.personal_score_awards` | 1064 |
| `player.match_citations` | 5137 |
| `player.match_skill_rank` | 364 |
| `player.career_progression` | 56 |
| `player.sessions` | 214 |
| `player.sync_meta` | 8 |
| `metadata` | 1 |
| `match_ids` | 364 |

## Usage

Ces fichiers sont utilisés par :
- `tests/api/test_filters.py` (tests schéma filtre avec DB réelle)
- `tests/parity/` (tests de parité API vs Streamlit)

Pour régénérer :
```bash
python scripts/create_test_corpus.py --gamertag Chocoboflor --limit 500
```
