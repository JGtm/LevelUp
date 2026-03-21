# scripts/investigation/

Scripts d'investigation, d'analyse ponctuelle et de benchmarking.
Ces scripts ne font pas partie du flux de production et ne sont pas référencés par la CI.

## Contenu

| Script | Rôle |
|--------|------|
| `analyze_match_overlap.py` | Analyse du chevauchement entre matchs de différents joueurs |
| `audit_current_data.py` | Audit de l'état courant des données DuckDB |
| `benchmark_pages.py` | Mesure de performance des pages Streamlit |
| `benchmark_query_performance.py` | Benchmarks des requêtes DuckDB |
| `benchmark_v4_vs_v5.py` | Comparaison perf architecture v4 vs v5 (référence historique) |
| `demo_regression_detection.py` | Démonstration de la détection de régressions |
| `diagnose_citations.py` | Diagnostic du moteur de citations |
| `exp_b2_pi_correlation.py` | Expérimentation corrélation B2/PI (extraction armes) |
| `exp_find_pi_offset.py` | Expérimentation recherche d'offset PI (extraction armes) |
| `_verify_weapon_kills.py` | Vérification de la qualité des données weapon_kills |

## Usage

Ces scripts s'exécutent depuis la racine du projet avec le venv activé :

```bash
source .venv/Scripts/activate  # Git Bash
.venv/Scripts/python.exe scripts/investigation/<script>.py
```

Ils peuvent nécessiter les bases de données de production montées localement.
