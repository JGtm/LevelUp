# Axe 1 · Réconciliation — Parité Python↔Go + Streamlit↔React

> **Statut** : ✅ Complété
> **Date de réconciliation** : `2026-04-18`
> **Profondeur Claude** : audit exhaustif (endpoints, pages, algos, schemas, scripts)
> **Profondeur ChatGPT** : audit ciblé (endpoints à risque, surfaces React, contrat OpenAPI)

## 1. Méthodologie

Claude a effectué un audit exhaustif (13 endpoints P0/P1, 16 pages UI, 7 algorithmes, 13 schémas DuckDB, 14 scripts). ChatGPT a effectué un audit ciblé (13 endpoints à risque, 15 pages UI, algorithmes non relancés). Les divergences portent essentiellement sur la profondeur d'analyse des pages UI et la classification des écarts détectés.

## 2. Convergences (Claude + ChatGPT identifient le même écart avec la même classif)

| Item | Section | Classif commune | Fichier:ligne | Note |
|------|---------|:---------------:|---------------|------|
| 13/13 endpoints P0/P1 implémentés, 0 stub 501 | A.1 | 🟢 | `internal/api/server.go:90-192` | Convergence forte |
| Exemptions contrat clôturées (contract_test:122) | A.1 | 🟢 | `api/contract_test.go:122` | ChatGPT le note explicitement |
| `ChangelogPage.tsx` — nouvelle feature React | B | 🟢 | `features/changelog/ChangelogPage.tsx:10` | Modernisation intentionnelle |
| Notify Discord présent Go | H | 🟢 | `internal/notify/` | Parité comportementale |
| Logs structurés slog middleware | H | 🟢 | `middleware/slog_logger.go` | Présent côté Go |
| `SessionContextResponse` multi-titre | I | 🟢 | `internal/domain/session.go:42-48` | Modernisation intentionnelle |
| 14 features React présentes dans les routes/tests | B | 🟢 | `apps/web/src/routes/` | Convergence |

## 3. Divergences de classification

| Item | Classif Claude | Classif ChatGPT | Classif finale retenue | Justification |
|------|:--------------:|:---------------:|:----------------------:|---------------|
| Page **Win/Loss** absente | 🟠 | ⚪ | **🟢** | Décision produit : pas de page Win/Loss dans la nouvelle architecture — non un écart |
| Page **Objectifs** absente | 🟠 | ⚪ | **🟢** | Erreur d'audit : cette page n'a jamais existé en Python (peut-être une section). Écart annulé. |
| **Timeseries** React très simplifié (5 onglets → 1) | 🟠 | 🟡 (surface présente, diff non relancée) | **🟠** | La surface est routée mais le contenu est incomplet — écart de fond confirmé par Claude |
| **i18n React** absent (14 langues Python → 0) | 🟠 | ⚪ non vérifié | **🟠** | Écart structurel confirmé : locale stockée dans Zustand mais aucun framework i18n |
| Algorithmes golden values | 🟢 (relancés, verts) | 🟡 (non relancés) | **🟢** | Claude a relancé les 7 tests avec golden values — 7/7 verts. ChatGPT n'avait pas les preuves. |
| Schémas DuckDB | 🟢 (vérifiés statiquement) | 🟡 (non revalidés en runtime) | **🟢** | Vérification statique Claude suffisante pour l'audit Sprint 50 |

## 4. Items identifiés par un seul LLM

| Item | Identifié par | Vérification manuelle | Retenu ? | Classif |
|------|:-------------:|-----------------------|:--------:|:-------:|
| Route `GET /directory/gamertags/search` conditionnelle (`gamertagSvc != nil`) — risque 503 en env dégradé | ChatGPT | `server.go:202` confirmé — enregistrement conditionnel | ✅ | 🟡 |
| Scripts `monitor_uptime.py`, `generate_thumbnails.py`, `prepare_demo_data.py` non portés | Claude | Reliquat d'une ancienne architecture — hors scope migration | ❌ | ~~🟡~~ |
| TODO `media/reset-index` sans ticket ni TTL | Claude | `handlers/settings.go:105` confirmé | ✅ | 🟡 |
| Home — médias récents, activité session embed manquants | Claude | Confirmé visuellement React vs Streamlit | ✅ | 🟡 |
| Career — projection Héros multi-joueurs manquante | Claude | Confirmé | ✅ | 🟡 |
| Squad — radars synergie, heatmaps, trios absents | Claude | Confirmé | ✅ | 🟡 |
| Match view — timeline weapon kills absente | Claude | Feature désactivée/non finalisée côté Python à l'origine — pas un écart de parité strict | ⚠️ | 🟡 *(à valider)* |

## 5. Synthèse finale de l'axe

| Niveau | Nombre d'items | Descriptions |
|--------|:--------------:|---|
| 🔴 Bloquant | 0 | — |
| 🟠 Majeur | 2 | Timeseries simplifié, i18n React absent |
| 🟡 Mineur | 5 | Home médias, Career Héros, Squad détails, MatchView timeline *(à valider)*, route gamertag conditionnelle |
| 🟢 Toléré | 9 | 13/13 P0/P1 ✅, golden values ✅, Changelog nouvelle feature, notify Discord ✅, multi-titre ✅, contrat clôturé ✅, setup wizard simplifié (voulu), Win/Loss non voulue (décision), Objectifs inexistante (erreur audit) |

## 6. Recommandation go / no-go pour l'axe 1

- [x] Aucun écart bloquant non résolu
- [ ] Écarts majeurs (Win/Loss, Objectifs, Timeseries, i18n) à ticketer pour Sprint 51+
- [x] Modernisations 🟢 toutes motivées

**Décision** : **GO conditionnel** — les 2 écarts 🟠 restants (Timeseries, i18n) sont des dettes fonctionnelles connues, non bloquantes pour la bascule technique, à planifier en Sprint 51+.
