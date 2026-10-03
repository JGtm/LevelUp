# MIGRATION_MASTER.md — Document maître FastAPI + React

> Point d'entrée unique pour naviguer le chantier de migration.
> Ce fichier ne contient pas de détails — il orchestre et pointe vers les sous-docs.

---

## Décisions irrévocables

| Décision | Statut |
|----------|--------|
| Garder Python 3.12 comme runtime backend | **Figé** |
| Remplacer uniquement la façade UI (Streamlit → React) | **Figé** |
| Migration en vertical slices, pas en big bang | **Figé** |
| Cible produit = shell V7 | **Figé** |
| Stack frontend : Vite + React + TS + TanStack Router + TanStack Query + Zustand + Tailwind v4 + shadcn/ui | **Figé** |
| Base path API : `/api/v1` | **Figé** |
| Payloads en snake_case, dates en ISO-8601 UTC | **Figé** |

Référence complète → [migration/DECISIONS.md](migration/DECISIONS.md)

---

## État courant

**Phase active** : Toutes les slices MVP (0b, 1, 2, 3, 4, 5, 6, 7, 8, **9**) **100% `canonical`** ✅ — 2026-04-13

**Slice en cours** : —

**Dernière action** : Slice 9 canonical — `launcher_startup.py` ajoute `_launch_react()` (uvicorn + npm run dev), `launcher.py`/`launcher_sync.py` migrés vers le nouveau point d’entrée, `run.sh` mis à jour (sanity check sans streamlit), badges README basculés React/FastAPI v7.0.0. `_launch_streamlit` conservé comme rollback court terme.

**Prochaine étape concrète** : **Migration terminée** — DoD global satisfait (2026-04-14). `streamlit_app*.py` archivés, docs mises à jour, tests verts. Prochains travaux : polish P2/P3 ou validation FUNCTIONAL_SPECS section par section.

---

## Tableau de gel par section V7

> Mettre à jour à chaque changement de statut. Dès qu'une section est en `preview`, la colonne "Gel Streamlit" passe à ✅.
> Les sections V7 regroupent les anciennes pages Streamlit — voir [FUNCTIONAL_SPECS.md](migration/FUNCTIONAL_SPECS.md) pour le détail.

| Section V7 | Contenu regroupé | Statut | Gel Streamlit | Notes |
|------------|-----------------|--------|---------------|-------|
| Shell + Bootstrap (0a) | Layout L1/L2/KPI, routing, PageContext | `canonical` ✅ | — | ✅ Slice 0a canonique — 5 E2E Playwright, generated.ts, proxy :5173→:8000 |
| Filtres resolve (0b) | Filtres cascade, chips, scope sessions | `canonical` ✅ | — | ✅ Slice 0b canonique — 3 E2E, API contract validé |
| Setup / Onboarding | Wizard configuration | `canonical` ✅ | — | ✅ Slice 1 canonique — 4 E2E, setup/status + settings |
| **Settings** | Langue, affichage, médias, Discord, backfill | `canonical` ✅ | — | ✅ Slice 1 (idem) |
| **Accueil** | Hero, signaux, Battle Pass, challenges, timeline, dernier match, médias récents | `canonical` ✅ | — | ✅ Slice 5 canonique — 4 E2E, home page + DemoPlayer |
| **Stats** | Séries (5 onglets) + Sessions (15 composants) + Historique (17 col) | `canonical` ✅ (Phases A+B+C) | — | ✅ Phase A : 4 E2E, match-history/query · Phase B : timeseries_api_service + router + 5 tests + TimeseriesPage React + route stats/timeseries · Phase C : session_compare_service + router + 5 tests + SessionComparePage React + route stats/sessions |
| **Escouade** | Synergies + Contributions, sélecteur 3 coéquipiers, impact ranking | `canonical` ✅ (Phase A) | — | ✅ Slice 6 canonique — 4 E2E, pages/teammates |
| **Synthèse** | Solo vs Escouade, heatmap, top semaine, breakdown carte/mode | `canonical` ✅ (Phase A) | — | ✅ Slice 7 canonique — 4 E2E, pages/synthesis |
| **Explorer** | Filtres cascade, recherche joueur, Match View (4 onglets, scoreboard 19 col) | `canonical` ✅ (Phases A+B+C) | — | ✅ Phase A : 4 E2E, explorer/matches-query · Phase B : match_view_service + router explorer (GET /matches/:id) + 4 tests + MatchViewPage React + route explorer/matches/$matchId · Phase C : router last-match (POST /pages/last-match/resolve) + 3 tests + LastMatchPage React + route last-match |
| **Médias** | Galerie, filtres, groupement, lightbox, likes | `canonical` ✅ (Phase A) | — | ✅ Slice 8 canonique — 4 E2E, POST pages/media (fix GET→POST) |
| **Profil** | Carrière (rang+XP+LUSR/CSR+encounters) + Citations (commendations+médailles) | `canonical` ✅ (Phases A+B) | — | ✅ Phase A : fix xuid DEMO_MODE, 8 parité + 5 Vitest + 5 E2E · Phase B : citations_service + router career (POST /pages/citations) + 4 tests + CitationsPage React + route profile/citations |

---

## Périmètre MVP

> Le périmètre MVP est calé sur les **sections V7 réelles**, pas sur l'ancien découpage Streamlit.

Sections incluses dans le MVP React/FastAPI :
- Setup / Onboarding (P0)
- Settings (P0)
- Profil — Carrière (P1), Citations (Phase B → P2)
- Stats — Historique des parties (P1), Séries et Sessions (Phase B/C → P3)
- Explorer — Recherche + filtres + Match View + Last Match (P1)

Sections post-MVP (cohabitation Streamlit) :
- Accueil — Home Mission Control (P2) — dépend de Career, Match View et Media déjà exposés
- Escouade (P2) — complexité élevée (13 sous-modules, 2 onglets riches)
- Médias (P2) — dépendance disque local + lightbox
- Synthèse (P3)

> **Priorités discutables** : l'ordre P2 entre Accueil, Escouade et Médias peut être ajusté
> selon les priorités utilisateur. Ce qui compte c'est que le slice couvre la section V7 **entière**.
> La Home (P2) peut devenir P1 si le hero card est jugé prioritaire.

---

## Navigation vers les sous-docs

| Sujet | Fichier | Quand le consulter |
|-------|---------|-------------------|
| **Specs fonctionnelles V7 — référentiel exhaustif** | [migration/FUNCTIONAL_SPECS.md](migration/FUNCTIONAL_SPECS.md) | **En premier** — avant toute implémentation. Contient le détail de chaque section V7 avec checkpoints de lecture |
| Stack, runtime, périmètre, règles structurantes | [migration/DECISIONS.md](migration/DECISIONS.md) | Avant de démarrer un nouveau slice ou de prendre une décision d'architecture |
| Ce qu'on garde / adapte / supprime dans `src/` | [migration/AUDIT_CODEBASE.md](migration/AUDIT_CODEBASE.md) | Avant de toucher à un module existant |
| Invariants fonctionnels à ne pas casser | [migration/INVARIANTS.md](migration/INVARIANTS.md) | À chaque slice — contient filtres, auth, i18n, deep links |
| Fiches écran + matrice de parité | [migration/PARITY_MATRIX.md](migration/PARITY_MATRIX.md) | Avant d'implémenter ou de valider un écran |
| Backlog de slices + statuts + dépendances | [migration/SLICES.md](migration/SLICES.md) | Pour savoir quoi faire ensuite et dans quel ordre |
| Contrats API : schemas, endpoints, stores, query keys | [migration/API_CONTRACTS.md](migration/API_CONTRACTS.md) | Pendant l'implémentation des routers et du front |
| Tables AG Grid + KPI cards React — inventaire et tasklists | [migration/NATIVE_COMPONENTS.md](migration/NATIVE_COMPONENTS.md) | À chaque slice contenant des tables ou des indicateurs (slices 2–5 en MVP) |

---

## Lecture obligatoire par slice active

### Slice 0a — Shell, bootstrap, plomberie
- [migration/DECISIONS.md](migration/DECISIONS.md) § Structure repo cible + Testing + Déploiement + Observabilité
- [migration/API_CONTRACTS.md](migration/API_CONTRACTS.md) § Bootstrap + Contrat erreurs HTTP
- [migration/SLICES.md](migration/SLICES.md) § Slice 0a

### Slice 0b — Contrat de filtres (spike)
- [migration/INVARIANTS.md](migration/INVARIANTS.md) § Filtres globaux et modèle de sessions + DuckDB concurrence
- [migration/AUDIT_CODEBASE.md](migration/AUDIT_CODEBASE.md) § Fichiers src/app/ à lire impérativement
- [migration/API_CONTRACTS.md](migration/API_CONTRACTS.md) § FilterContextInput/Resolved
- [migration/SLICES.md](migration/SLICES.md) § Slice 0b

### Slice 1 — Setup / Auth / Settings
- [migration/INVARIANTS.md](migration/INVARIANTS.md) § Auth et bootstrap
- [migration/PARITY_MATRIX.md](migration/PARITY_MATRIX.md) § Setup Wizard + Settings
- [migration/API_CONTRACTS.md](migration/API_CONTRACTS.md) § Contrats Setup / Auth / Settings
- [migration/SLICES.md](migration/SLICES.md) § Slice 1

### Slice 2 — Profil (Carrière + Citations)
- **[migration/FUNCTIONAL_SPECS.md](migration/FUNCTIONAL_SPECS.md) § 7 Profil** — specs détaillées (rang, XP, LUSR/CSR, encounters, commendations, médailles)
- [migration/INVARIANTS.md](migration/INVARIANTS.md) § Filtres globaux
- [migration/PARITY_MATRIX.md](migration/PARITY_MATRIX.md) § Profil (Carrière Phase A + Citations Phase B)
- [migration/API_CONTRACTS.md](migration/API_CONTRACTS.md) § Contrats Slice 2 — Profil
- [migration/SLICES.md](migration/SLICES.md) § Slice 2
- [migration/NATIVE_COMPONENTS.md](migration/NATIVE_COMPONENTS.md) § C1, C2, A2, A3, A4 (jauges + tables Career)

### Slice 3 — Stats (Séries + Sessions + Historique)
- **[migration/FUNCTIONAL_SPECS.md](migration/FUNCTIONAL_SPECS.md) § 2 Stats** — 3 sous-vues, 5 onglets Séries, 15 composants Sessions, 17 colonnes Historique
- [migration/INVARIANTS.md](migration/INVARIANTS.md) § Filtres globaux
- [migration/PARITY_MATRIX.md](migration/PARITY_MATRIX.md) § Stats (Historique Phase A + Séries Phase B + Sessions Phase C)
- [migration/API_CONTRACTS.md](migration/API_CONTRACTS.md) § Contrats Slice 3 — Stats
- [migration/SLICES.md](migration/SLICES.md) § Slice 3
- [migration/NATIVE_COMPONENTS.md](migration/NATIVE_COMPONENTS.md) § A1 (table Historique)

### Slice 4 — Explorer (Filtres + Match View)
- **[migration/FUNCTIONAL_SPECS.md](migration/FUNCTIONAL_SPECS.md) § 5 Explorer** — cascade 4 dimensions, recherche joueur, Match View 4 onglets
- [migration/PARITY_MATRIX.md](migration/PARITY_MATRIX.md) § Explorer (Phase A + Match View Phase B + Last Match Phase C)
- [migration/API_CONTRACTS.md](migration/API_CONTRACTS.md) § Contrats Slice 4 — Explorer
- [migration/SLICES.md](migration/SLICES.md) § Slice 4
- [migration/NATIVE_COMPONENTS.md](migration/NATIVE_COMPONENTS.md) § A1, A5, A6, C3, C4, C5, E1 (table Explorer + scoreboard + KPI cards)

### Slice 5 — Accueil (Home Mission Control)
- **[migration/FUNCTIONAL_SPECS.md](migration/FUNCTIONAL_SPECS.md) § 1 Accueil** — Hero Card, signaux, Battle Pass, challenges, timeline, dernier match, médias récents
- [migration/PARITY_MATRIX.md](migration/PARITY_MATRIX.md) § Accueil (Home Mission Control)
- [migration/API_CONTRACTS.md](migration/API_CONTRACTS.md) § Contrats Slice 5 — Accueil
- [migration/SLICES.md](migration/SLICES.md) § Slice 5

### Post-MVP — Escouade (Slice 6)
- **[migration/FUNCTIONAL_SPECS.md](migration/FUNCTIONAL_SPECS.md) § 3 Escouade** — 13 sous-modules, 2 onglets, sélecteur 3 coéquipiers
- [migration/PARITY_MATRIX.md](migration/PARITY_MATRIX.md) § Escouade (Teammates)
- [migration/API_CONTRACTS.md](migration/API_CONTRACTS.md) § Contrats Slice 6 — Escouade (placeholder)

### Post-MVP — Synthèse (Slice 7)
- **[migration/FUNCTIONAL_SPECS.md](migration/FUNCTIONAL_SPECS.md) § 4 Synthèse** — Solo vs Escouade, heatmap, top semaine
- [migration/PARITY_MATRIX.md](migration/PARITY_MATRIX.md) § Synthèse (+ Objective Analysis absorbé)
- [migration/API_CONTRACTS.md](migration/API_CONTRACTS.md) § Contrats Slice 7 — Synthèse (placeholder)

### Post-MVP — Médias (Slice 8)
- **[migration/FUNCTIONAL_SPECS.md](migration/FUNCTIONAL_SPECS.md) § 6 Médias** — galerie, enrichissement, lightbox, likes localStorage
- [migration/PARITY_MATRIX.md](migration/PARITY_MATRIX.md) § Médias (Media V2)
- [migration/API_CONTRACTS.md](migration/API_CONTRACTS.md) § Contrats Slice 8 — Médias (placeholder)

---

## Structure repo cible (résumé)

```
apps/
  api/         ← couche HTTP FastAPI uniquement
  web/         ← front React/Vite uniquement
src/           ← noyau Python inchangé (analysis, data, auth, visualization…)
streamlit_app.py   ← legacy, maintenu pendant cohabitation
src/ui/        ← legacy, vidé progressivement
```

Détail complet → [migration/DECISIONS.md](migration/DECISIONS.md) § Structure repo

---

## Modèle de cohabitation

| État surface | Front canonique | Règle |
|---|---|---|
| Legacy seule | Streamlit | pas encore commencé côté React |
| Preview React | Streamlit (référence) + React (dev/flag) | tests de parité requis avant bascule |
| Canonique React | React | Streamlit conservé comme rollback court terme |
| Décommissionnée | React | route Streamlit retirée ou redirigée |

Règles détaillées → [migration/SLICES.md](migration/SLICES.md) § Modèle de cohabitation

---

## Définition of done — **`canonical` ✅ 2026-04-14**

> DoD global satisfait. Migration React/FastAPI terminée.

1. ✅ Toutes les sections V7 du MVP/P1 sont en état `canonical` (Setup, Settings, Accueil, Stats, Explorer, Profil)
2. ✅ Les sections P2 (Escouade, Synthèse, Médias) sont explicitement dépriorisées pour délais — décision dans ce fichier à Slice 5-8
3. ✅ Streamlit ne délivre plus aucune surface active (`streamlit_app*.py` archivés)
4. ✅ Tests de parité verts (suite hors integration)
5. ✅ Documentation à jour sans mention de Streamlit comme front principal
6. ✅ Les imports `src/ui/pages/` dans les services FastAPI sont de la logique métier réutilisée, pas du rendu Streamlit
7. ⚪ Validation FUNCTIONAL_SPECS.md — optionnel, à faire par section lors du polish P2/P3

Détail → [migration/SLICES.md](migration/SLICES.md) § Définition of done globale

---

## Causes d'échec à éviter

1. Tout réécrire d'un coup
2. Changer front, backend, auth et design system simultanément
3. Sous-estimer la logique embarquée dans Streamlit (session_state, reruns, shadow keys)
4. Refaire tous les graphes trop tôt plutôt que réutiliser le JSON Plotly
5. Patcher Streamlit sur un écran déjà en preview React au lieu de corriger dans le nouveau front
6. Ne pas constituer le corpus de référence avant d'écrire du code (tests de parité inutilisables)
7. Démarrer sans `DEMO_MODE` fonctionnel — la boucle de dev devient trop lente dès le premier jour
