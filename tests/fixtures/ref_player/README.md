# tests/fixtures/ref_player/

Ce répertoire contient les **copies figées** des bases DuckDB de référence
utilisées par les tests de parité backend (`tests/parity/`).

## Contenu attendu

| Fichier | Source | Description |
|---------|--------|-------------|
| `stats.duckdb` | `data/players/{gamertag}/stats.duckdb` | DB joueur de référence |
| `shared_matches_v2.duckdb` | `data/warehouse/shared_matches_v2.duckdb` | DB partagée réduite (~500 matchs) |
| `metadata.duckdb` | `data/warehouse/metadata.duckdb` | Référentiels |

## Comment remplir ce répertoire

Utiliser le script de création du corpus :

```bash
python scripts/create_test_corpus.py --gamertag <GamertaxDuJoueurDeRef> --max-matches 500
```

Le script extrait un sous-ensemble représentatif (couvrant tous les cas de test)
dans des DBs allégées copiées ici.

## Règles

- Ces fichiers ne doivent **jamais** être écrasés par une sync réelle.
- Ne pas committer de fichiers > 100 Mo — réduire le corpus ou activer Git LFS.
- Le DEMO_MODE de l'API pointe sur ce répertoire (`LEVELUP_DEMO_FIXTURES_DIR`).
