# Référentiel des Citations (Commendations) — redirection

Ce référentiel a été consolidé. La source de vérité à jour (98 commendations, schéma DuckDB, seed Go) est maintenue en anglais :

- **Référence complète des données** : [../COMMENDATIONS_REFERENCE.md](../COMMENDATIONS_REFERENCE.md)
- **Guide de la feature** : [../COMMENDATIONS.md](../COMMENDATIONS.md) — voir aussi la redirection FR [CITATIONS.md](CITATIONS.md)

Le référentiel de données (IDs de médailles, clés de normalisation, tiers, masters) est désormais piloté par le **seed Go autoritatif** `internal/ops/seed_citation_data.go`. Pour reconstruire la table `citation_mappings` (`metadata.duckdb`) : `levelup seed citation-mappings`. Les fonctions custom sont dispatchées dans `internal/games/halo_infinite/citations_custom.go`.

> Note : « citations » (docs FR) et « commendations » (docs EN) désignent le même système. L'ancienne implémentation Python (`scripts/populate_citation_mappings.py`, `src/analysis/citations/`, `src/ui/commendations.py`) a été supprimée au passage à Go.
