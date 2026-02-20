# Index — Documentation Projet Éradication Legacy v5.1

> **Dernière mise à jour** : 2026-02-16  
> **Projet** : LevelUp v5.0 → v5.1 (Pure Architecture)

---

## 📚 Documents de Référence

### 🎯 Plan de Projet

| Document | Description | Lecteur cible |
|----------|-------------|---------------|
| **[SYNTHESE_EXECUTIVE_V5.1.md](.ai/SYNTHESE_EXECUTIVE_V5.1.md)** | Résumé exécutif du projet (4 pages) | Stakeholders, management |
| **[PLAN_ERADICATION_LEGACY_V5.md](.ai/PLAN_ERADICATION_LEGACY_V5.md)** | Plan détaillé complet (60+ pages) | Équipe technique, développeurs |
| **[V5.1_PURE_ARCHITECTURE.md](docs/V5.1_PURE_ARCHITECTURE.md)** | Manifeste architecture pure | Développeurs, architectes |

### 📖 Documentation Architecture Existante

| Document | Description |
|----------|-------------|
| [ARCHITECTURE_V5.md](docs/ARCHITECTURE_V5.md) | Architecture v5 détaillée (shared matches) |
| [SHARED_MATCHES_SCHEMA.md](docs/SHARED_MATCHES_SCHEMA.md) | Schéma DDL complet bases DuckDB |
| [CLEANUP_V5.md](docs/CLEANUP_V5.md) | Guide cleanup tables post-migration |
| [MIGRATION_V4_TO_V5.md](docs/MIGRATION_V4_TO_V5.md) | Guide migration v4 → v5 |
| [POLARS_MIGRATION.md](docs/POLARS_MIGRATION.md) | Guide migration Pandas → Polars |

### 📋 Audits et Rapports

| Document | Description |
|----------|-------------|
| [thought_log.md](.ai/thought_log.md) | Journal de décisions et raisonnement |
| [project_map.md](.ai/project_map.md) | Cartographie du projet |
| [data_lineage.md](.ai/data_lineage.md) | Flux de données |

### 📦 Archives v5.0

| Document | Description |
|----------|-------------|
| [v5-baseline-audit.md](.ai/archive/v5.0/v5-baseline-audit.md) | Audit baseline v5 |
| [v5-migration-report.md](.ai/archive/v5.0/v5-migration-report.md) | Rapport migration v5 |
| [PLAN_UNIFIE.md](.ai/archive/v5.0/PLAN_UNIFIE.md) | Plan unifié v5 (historique) |

---

## 🗺️ Navigation Rapide

### Pour Comprendre le Projet

1. **Démarrage rapide** : Lire [SYNTHESE_EXECUTIVE_V5.1.md](.ai/SYNTHESE_EXECUTIVE_V5.1.md) (15 min)
2. **Contexte technique** : Lire [V5.1_PURE_ARCHITECTURE.md](docs/V5.1_PURE_ARCHITECTURE.md) (30 min)
3. **Plan détaillé** : Consulter [PLAN_ERADICATION_LEGACY_V5.md](.ai/PLAN_ERADICATION_LEGACY_V5.md) (60 min)

### Pour Exécuter le Projet

1. **Phase 0** : Préparation (section Plan détaillé)
2. **Phase 1-6** : Suivre l'ordre dans [PLAN_ERADICATION_LEGACY_V5.md](.ai/PLAN_ERADICATION_LEGACY_V5.md)
3. **Validation** : Utiliser checklist de conformité ([V5.1_PURE_ARCHITECTURE.md](docs/V5.1_PURE_ARCHITECTURE.md))

### Pour Développer (Post-v5.1)

1. **Guidelines** : Lire section "Guidelines Agents IA" de [V5.1_PURE_ARCHITECTURE.md](docs/V5.1_PURE_ARCHITECTURE.md)
2. **Conventions** : Section "Conventions de Code"
3. **Anti-patterns** : Section "Anti-Patterns à Éviter"

---

## 📊 Inventaire des Vestiges Legacy

### SQLite (7 fichiers runtime)

| Fichier | Type | Action |
|---------|------|--------|
| `src/data/query/engine.py` | Fallback runtime | 🗑️ SUPPRIMER |
| `src/data/infrastructure/database/duckdb_engine.py` | Fallback runtime | 🗑️ SUPPRIMER |
| `src/utils/paths.py` | Références `.db` | 🔄 NETTOYER |
| `src/ui/sync.py` | Import `sqlite3` | 🔍 AUDITER |
| `src/ui/multiplayer.py` | Import `sqlite3` | 🔍 AUDITER |
| `src/ai/rag.py` | Import `sqlite3` | 🔍 AUDITER |
| `src/data/.cursorrules` | Doc legacy | 🔄 METTRE À JOUR |

### Pandas (7 fichiers métier)

| Fichier | Type | Effort |
|---------|------|--------|
| `src/analysis/performance_score.py` | Logique métier | 🔨 4h |
| `src/data/services/win_loss_service.py` | Service | 🔨 3h |
| `src/ui/pages/objective_analysis.py` | Page UI | 🔨 2h |
| `src/ui/pages/match_view_helpers.py` | Helpers | 🔨 1h |
| `src/ui/pages/win_loss.py` | Page UI | 🔨 1h |
| `src/ui/cache_filters.py` | Caching | 🔨 0.5h |
| `src/ui/components/duckdb_analytics.py` | Composant | 🔨 0.5h |

### Scripts Migration (5 à marquer LEGACY)

| Script | Action |
|--------|--------|
| `scripts/migration/recover_from_sqlite.py` | ✅ Déjà marqué LEGACY |
| `scripts/migration/migrate_player_to_duckdb.py` | ⏳ Ajouter bannière |
| `scripts/migration/migrate_all_to_duckdb.py` | ⏳ Ajouter bannière |
| `scripts/migration/migrate_metadata_to_duckdb.py` | ⏳ Ajouter bannière |
| `scripts/migration/migrate_player_to_shared.py` | ⏳ Ajouter bannière |

### Scripts Archive (~60 à trier)

| Catégorie | Quantité estimée | Action |
|-----------|-----------------|--------|
| R&D terminée | ~30 | 🗑️ SUPPRIMER |
| Migration/Recovery | ~10 | 📦 GARDER + DOC |
| Benchmarks | ~5 | 📦 GARDER |
| Utilitaires | ~15 | 🔍 ÉVALUER |

---

## 📅 Planning (4 jours)

```
Jour 1 (7h) : Phase 0-2
├── Préparation (2h)
├── Éradication SQLite (4h)
└── Archivage Scripts (1h)

Jour 2 (7h) : Phase 3
└── Migration Pandas (7h)

Jour 3 (7h) : Phase 3-5
├── Migration Pandas suite (5h)
├── Cleanup DBs (1h)
└── Audit Archive (1h)

Jour 4 (7h) : Phase 5-6
├── Audit Archive suite (2h)
├── Validation & Docs (4h)
└── Buffer (1h)
```

---

## ✅ Checklist d'Avancement

### Phase 0 : Préparation
- [ ] Backup complet production
- [ ] Validation architecture v5
- [ ] Snapshot état actuel
- [ ] Branche de secours créée

### Phase 1 : Éradication SQLite
- [ ] Supprimé fallback `engine.py`
- [ ] Supprimé fallback `duckdb_engine.py`
- [ ] Nettoyé références `.db`
- [ ] Supprimé imports `sqlite3` runtime
- [ ] Tests passent

### Phase 2 : Archivage Scripts
- [ ] Bannières LEGACY ajoutées (5 scripts)
- [ ] `scripts/migration/README.md` créé
- [ ] Décision `refetch_film_roster.py`

### Phase 3 : Migration Pandas
- [ ] `performance_score.py` migré
- [ ] `win_loss_service.py` migré
- [ ] `objective_analysis.py` migré
- [ ] `match_view_helpers.py` migré
- [ ] `win_loss.py` migré
- [ ] `cache_filters.py` migré
- [ ] `duckdb_analytics.py` migré
- [ ] Tests passent

### Phase 4 : Cleanup DBs
- [ ] Dry-run validé
- [ ] Backup + cleanup exécuté
- [ ] Application validée
- [ ] Tests passent

### Phase 5 : Audit Archive
- [ ] Inventaire complet
- [ ] Classification scripts
- [ ] `scripts/_archive/README.md` créé
- [ ] Scripts R&D supprimés

### Phase 6 : Validation & Docs
- [ ] Suite tests complète verte
- [ ] Couverture ≥ 80%
- [ ] Audit sécurité OK
- [ ] Docs mises à jour (5 fichiers)
- [ ] Release notes créées
- [ ] Validation finale humaine

---

## 🎯 Métriques de Succès

| KPI | État v5.0 | Cible v5.1 | Statut |
|-----|-----------|-----------|--------|
| Imports SQLite runtime | 7 | 0 | ⏳ |
| Imports Pandas métier | 7 | 0 | ⏳ |
| Taille player DB | ~30 MB | ~4 MB | ⏳ |
| Couverture tests | 75% | ≥80% | ⏳ |
| Temps chargement 100 matchs | 350ms | <200ms | ⏳ |

---

## 📞 Contact et Support

### Pour Questions Techniques

- Consulter [thought_log.md](.ai/thought_log.md) pour historique décisions
- Lire [V5.1_PURE_ARCHITECTURE.md](docs/V5.1_PURE_ARCHITECTURE.md) pour guidelines

### Pour Contribuer

1. Lire les guidelines dans [V5.1_PURE_ARCHITECTURE.md](docs/V5.1_PURE_ARCHITECTURE.md)
2. Suivre le workflow de développement
3. Respecter la checklist de conformité

---

## 📖 Ressources Externes

- [DuckDB Documentation](https://duckdb.org/docs/)
- [Polars User Guide](https://docs.pola.rs/)
- [Streamlit API Reference](https://docs.streamlit.io/)
- [Pydantic v2 Documentation](https://docs.pydantic.dev/)
- [SPNKr GitHub](https://github.com/acurtis166/SPNKr)

---

**Dernière mise à jour** : 2026-02-16 — Planification initiale complète ✅
