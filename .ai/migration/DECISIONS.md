# DECISIONS.md — Décisions d'architecture figées

> Ce fichier contient les décisions irrévocables de la migration.
> Il ne change que si une décision est explicitement révisée avec justification.
> Source : PLAN_MIGRATION_FASTAPI_REACT.md

---

## 1. Runtime backend

**Décision : garder Python 3.12 pendant toute la migration.**

Justifications détaillées :
- Toute la valeur métier est déjà là : DuckDB, Polars, Pydantic, sync, auth, repositories, analyses, scores, sessions, parser d'armes, services de pages.
- Le couple DuckDB + Polars + Pydantic est déjà très adapté au domaine.
- Les modules de visualisation Plotly sont déjà en Python et peuvent être exposés en JSON sans réécriture immédiate.
- Refaire le backend en Node, Go ou Rust obligerait à réimplémenter une énorme quantité de logique déjà stable.
- Le vrai irritant à éliminer est Streamlit, pas Python.
- Changer le runtime backend maintenant serait une migration dans la migration — deux chantiers simultanés impossibles à déboguer séparément.

Quand reconsidérer (conditions nécessaires, pas suffisantes) :
- Besoin d'une infra très orientée microservices ou edge
- Besoin de très forte concurrence réseau côté API
- Besoin de sortir une partie calcul critique dans un runtime spécialisé
- Besoin de mutualiser le backend avec un autre produit non-Python

**Ce qu'on ne fait pas maintenant : Node/NestJS, Go, Rust comme backend.** Ces options peuvent avoir du sens à long terme sur un sous-système ciblé. Elles sont mauvaises comme point de départ pour cette migration car elles détruisent le principal levier de vitesse : la réutilisation de l'existant.

---

## 2. Périmètre de la migration

**Ce qui reste strictement identique :**
- Définitions métier et calculs : performance score, sessions, skill rating, citations, stats de match, résolution gamertag/xuid, agrégats et règles de tri
- Sources de données et leur architecture : DuckDB v6, metadata/shared/player DBs, SyncScope, sync/backfill, repositories et vues SQL
- Comportements fonctionnels des parcours migrés : mêmes chiffres, mêmes filtres, mêmes regroupements, mêmes permissions, mêmes labels métier
- Contrat fonctionnel de l'auth Halo : mêmes capacités, mêmes credentials, mêmes protections
- Le bilinguisme FR/EN

**Ce qui peut être amélioré :**
- Design system, hiérarchie visuelle, layouts, responsive, transitions
- Modèle de navigation : URLs plus propres, deep links explicites, navigation plus fluide
- Expérience data : loading states, empty states, erreurs lisibles, tables web riches
- Mode de livraison technique : endpoints explicites, pagination serveur, cache HTTP, TanStack Query, Zustand, cookies httpOnly, JSON Plotly

**Explicitement hors périmètre :**
- Réécriture du runtime backend
- Refonte du schéma de données, changer DuckDB, remplacer Polars ou revoir le moteur de sync
- Ajout de nouvelles briques produit sans équivalent Streamlit actuel
- Réécriture de tous les graphes dans une autre lib dès le départ
- Refonte du modèle analytique, des scores, des rules engines ou des métriques métier
- Transformation en rebranding complet, application mobile, produit multi-tenant ou plateforme publique
- **Stats PvE Firefight (`shared_pve.duckdb`)** — la base existe mais aucune surface UI dédiée ne sera migrée dans ce chantier. Si un écran Firefight est créé plus tard, il fera l'objet d'un slice indépendant post-migration

### Clarification importante sur la cible produit V7

- `Cible produit = shell V7` signifie que la navigation, la hiérarchie des surfaces, le langage visuel et les conventions d'URL du cockpit V7 deviennent la référence finale.
- Cela ne signifie **pas** que `Home Mission Control` complet doit faire partie du MVP initial. La home V7 reste explicitement post-MVP tant que les parcours centraux ne sont pas migrés.
- Le MVP peut donc adopter un shell React inspiré de V7 tout en laissant certaines surfaces V7 riches en cohabitation Streamlit.
- En cas de doute pendant l'implémentation : la priorité va aux parcours métier MVP et à la parité fonctionnelle, pas à la reconstitution immédiate de toute la home V7.

---

## 3. Stack technique cible

### Backend
- **Framework** : FastAPI
- **Validation** : Pydantic v2 (déjà utilisé dans `src/`)
- **Base path** : `/api/v1`
- **Payloads** : snake_case (aligné sur Pydantic)
- **Dates** : ISO-8601, datetimes normalisés UTC ou explicitement étiquetés
- **Auth** : session backend opaque, cookies httpOnly — les tokens Halo ne transitent jamais vers le navigateur

#### Précisions obligatoires sur la session web

- Le stockage de session doit être **partagé entre workers/processus**. Aucune session auth utile au produit ne doit dépendre de la mémoire d'un worker FastAPI unique.
- Le cookie de session est `httpOnly`, `Secure` en production, et `SameSite=Lax` par défaut. Toute dérogation à ce profil doit être documentée explicitement.
- Toute route mutante authentifiée par cookie doit être protégée contre les requêtes cross-site non désirées via CSRF ou mécanisme équivalent.
- Le CORS est limité aux origines du frontend réellement déployées. Aucun wildcard en production.
- Le navigateur ne reçoit jamais de `spartan_token`, de `clearance`, de refresh token ou le contenu du cache MSAL. Le contrat UI se limite à des états (`missing`, `partial`, `ready`) et à des actions (`start`, `poll`, `logout`, `refresh-status`).
- Le TTL de session web est piloté côté backend et reste indépendant du TTL des tokens Halo. Une session web peut survivre à un token Halo expiré, auquel cas l'API retourne un état d'auth dégradé plutôt que de laisser le front deviner.
- `player_slug` dans l'URL reste un identifiant de navigation, pas une preuve d'autorisation. Le backend valide toujours que la ressource demandée appartient bien au contexte de profils accessibles.

### Frontend
- **Bundler** : Vite
- **Framework** : React + TypeScript
- **Routing** : TanStack Router (URL-first)
- **Data fetching** : TanStack Query
- **État UI** : Zustand (transverse feature) + état composant (local)
- **Persistence navigateur** : localStorage / IndexedDB pour préférences non sensibles
- **UI / Design** : Tailwind v4 + shadcn/ui
- **Animations** : Framer Motion
- **Tables riches** : AG Grid Community
- **Graphes** : react-plotly.js (figures JSON servies par le backend Python)
- **KPI cards / jauges de progression** : React CSS + Tailwind v4 (sans Plotly — voir `NATIVE_COMPONENTS.md`)
- **Tests unitaires / intégration** : Vitest + React Testing Library
- **Tests E2E** : Playwright
- **Mocking API** : MSW (Mock Service Worker)
- **Génération de types** : openapi-typescript depuis le schéma OpenAPI exporté par FastAPI

#### Décision de session web backend

- **Technologie retenue** : session serveur signée, stockée en fichier local (`itsdangerous` + stockage JSON/DuckDB).
- Redis serait du sur-engineering pour un projet single-user / small-scale. Le fichier session est suffisant et ne crée aucune dépendance infra.
- Le cookie de session contient uniquement un identifiant opaque signé. Le contenu de session (auth state, player context) vit côté serveur.
- **TTL de session web** : 7 jours par défaut, configurable. Indépendant du TTL des tokens Halo.
- **Nettoyage** : les sessions expirées sont purgées au démarrage de l'API et périodiquement (background task FastAPI).
- **Montée de version** : en cas de changement de structure, la session est invalidée et l'utilisateur repasse par bootstrap — pas de migration de session.
- Pour le MVP, `uvicorn --workers=1` élimine les problèmes de concurrence fichier. Si multi-worker devient nécessaire, migrer vers Redis ou un backend SQLite/DuckDB avec verrou.

#### Décision de transport pour les jobs longs

- Pour le MVP, le transport temps réel des jobs repose sur **polling HTTP explicite** via `AsyncJobStatus`.
- SSE ou WebSocket restent des optimisations futures possibles, mais ne font pas partie du socle obligatoire de Slice 0/1.

---

## 4. Structure du repo

```
apps/
  api/
    app/
      __init__.py
      main.py
      core/
        config.py
        errors.py
        logging.py
      deps/
        auth.py
        filters.py
        players.py
      routers/
        bootstrap.py
        setup.py
        settings.py
        players.py
        career.py
        match_history.py
        explorer.py
        matches.py
      schemas/
        common.py
        bootstrap.py
        filters.py
        jobs.py
        settings.py
        pages/
          career.py
          match_history.py
          explorer.py
          match_view.py
      services/
        bootstrap_service.py
        filter_service.py
        setup_service.py
        settings_service.py
        pages/
          career_service.py
          match_history_service.py
          explorer_service.py
          match_view_service.py
      jobs/
        registry.py
        smoke_test.py
        media.py
  web/
    package.json
    vite.config.ts
    tsconfig.json
    index.html
    src/
      app/
        providers/
        router/
        layout/
      routes/
        __root.tsx
        index.tsx              # Accueil (Home Mission Control)
        setup.tsx
        settings.tsx
        players/
          $playerSlug/
            profile/
              career.tsx
              citations.tsx
            stats/
              history.tsx
              timeseries.tsx
              sessions.tsx
            explorer/
              index.tsx
              matches/
                $matchId.tsx
            last-match.tsx
            squad.tsx
            synthesis.tsx
            media.tsx
      features/
        bootstrap/
        filters/
        setup/
        settings/
        profile/           # career + citations
        match-history/
        timeseries/
        session-compare/
        explorer/
        match-view/
        home/
        squad/             # teammates / escouade
        synthesis/
        media/
      components/
      stores/
      lib/
        api/
        query/
        utils/
      styles/
src/           ← noyau Python inchangé
  analysis/
  app/         ← legacy, vidé progressivement
  auth/
  data/
  ports/
  ui/          ← legacy, vidé progressivement
  utils/
  visualization/
data/
tests/
streamlit_app.py     ← legacy cohabitation
streamlit_app_v7.py  ← legacy cohabitation
pyproject.toml
Makefile
Dockerfile
```

### Règles de placement non négociables

- Pas de duplication de logique métier entre `src/` et `apps/api/`
- Pas de TypeScript ou d'assets React dans `src/ui/`
- Pas de code FastAPI dans `streamlit_app.py` ou `launcher.py`
- Pas de deuxième `pyproject.toml` tant que l'API n'a pas besoin d'isolation packaging
- Pas de package Node à la racine — tout le front vit dans `apps/web/`
- Pas de package `shared/` ou `common/` fourre-tout au premier passage

### Responsabilités par zone

**`src/`** : source de vérité métier Python. Toute logique extraite des pages Streamlit va dans `src/analysis/` ou `src/data/services/`, jamais dans `apps/api/`.

**`apps/api/`** : exposition HTTP uniquement. `routers/` = définition des routes + validation HTTP. `schemas/` = payloads FastAPI/Pydantic spécifiques au transport. `services/` = composition des appels vers `src/` et assemblage des réponses. `deps/` = injection de dépendances HTTP.

**`apps/web/`** : UI React uniquement. `routes/` = arbre de navigation URL-first. `features/` = code par vertical slice. `stores/` = stores shell et cross-feature uniquement. `lib/api/` = client HTTP centralisé.

**Zone legacy** : `streamlit_app.py`, `streamlit_app_v7.py`, `src/ui/`, `src/app/` restent en place jusqu'à décommission des slices correspondantes.

---

## 5. Stratégie Plotly

Ne pas réécrire les graphes au début.

- Backend : continue de construire les figures en Python
- Backend : expose les figures en JSON Plotly (`fig.to_plotly_json()`)
- Frontend : affiche via `react-plotly.js`

Payload de transport : `PlotlyFigurePayload { figure, config_key: "clean"|"static", revision_key }`

### Ce qui doit sortir en JSON Plotly rapidement

- Carrière
- Last Match
- Match View
- Une grande partie de Timeseries
- Teammates

### Ce qui doit sortir plutôt en données brutes

- Explorer
- Historique
- Certaines cartes de la home
- Listes récentes, tables, badges, cartes d'action

### Stratégie par page

**À migrer tôt avec figures backend** : Carrière, Dernier match, Match View

**À migrer tôt avec données brutes + composants React** : Explorer, Historique, Paramètres

**À repousser après fondations stables** : Timeseries, Teammates, Session Compare, Media

### Ce qui se migre en composants React natifs (pas en Plotly JSON)

Certains éléments rendus actuellement en HTML Streamlit brut ou via `go.Indicator` sont de mauvais candidats
pour `react-plotly.js`. Ils sont migrés en AG Grid Community (tables) ou React CSS/SVG (KPI cards, jauges).

Inventaire exhaustif, tasklists et DoD → **[NATIVE_COMPONENTS.md](NATIVE_COMPONENTS.md)**

**Tableau de choix rapide** :

| Type d'élément | Stratégie |
|----------------|-----------|
| Tableaux de données (scoreboard, match history, encounters, top matchs) | AG Grid Community |
| Jauges de progression rang (§1.1, §1.2) | CSS `conic-gradient` ou SVG — composant `<RankProgressGauge>` |
| KPI cards avec delta (expected vs actual, K/D vs nemesis, blocs rang) | React CSS + Tailwind |
| Indicateurs statistiques simples (régression, delta session) | React CSS + Tailwind |
| Grilles d'icônes (médailles, grilles fixes) | CSS `grid` |
| Expand panels multiSection (scoreboard détail) | `shadcn/ui Collapsible` |
| Tout le reste (timeseries, radars, heatmaps, KDE, subplots, scatter…) | `react-plotly.js` |

---

## 6. Règle de propriété d'état

Tout état a un seul propriétaire légitime :

| Type d'état | Propriétaire |
|-------------|--------------|
| Navigable et partageable | URL (TanStack Router) |
| Données serveur distantes | TanStack Query + backend |
| État UI éphémère local | Zustand ou état composant |
| Préférences navigateur non sensibles | localStorage / IndexedDB |
| Auth, joueur courant, jobs longs, secrets | Session backend |

---

## 7. Modèle de cohabitation

**Un seul cœur backend** pendant toute la transition : Streamlit et React consomment le même Python.

**Un propriétaire canonique par surface** — jamais deux fronts canoniques simultanément.

Séquence de bascule par surface :
1. Legacy seule (Streamlit)
2. Preview React (accès dev/flag, Streamlit reste référence)
3. Canonique React (après validation parité)
4. Décommissionnée (route Streamlit retirée)

---

## 8. Règle de décision pour tout futur ajout au plan

Une demande entre dans le périmètre si :
- elle remplace ou supporte un parcours déjà existant
- elle n'impose pas de changer la logique métier de référence
- elle est nécessaire à la parité fonctionnelle ou à l'UX minimale du nouvel écran

Une demande sort du périmètre si :
- nouveau besoin produit sans équivalent actuel
- changement de calcul, de schéma ou de source de vérité métier
- chantier transverse backend non requis pour exposer l'existant via API

---

## 9. Risques principaux

Ces risques sont à garder en tête à chaque bascule de slice.

**Risque 1 — Migration du state model** (le plus gros risque, pas Plotly)

Le passage de `session_state Streamlit` vers `URL + cache serveur + état UI local + session backend` est la source la plus probable de régressions invisibles. Chaque état implicite mal migré crée un écran incohérent ou non-partageable par URL.

**Risque 2 — Auth et cache tokens**

Le modèle actuel convient à Streamlit (process unique, cache en mémoire) mais doit être refondu pour FastAPI multi-processus. La session opaque et les cookies httpOnly doivent être traités très tôt, pas en dernier.

**Risque 3 — Pages trop mixtes**

Certaines pages mélangent encore logique métier, orchestration et rendu UI — elles doivent être découpées avant d'être migrables proprement. Voir [AUDIT_CODEBASE.md](AUDIT_CODEBASE.md) § Modules mixtes à découper.

**Risque 4 — Double produit temporaire**

Pendant une partie de la migration, Streamlit et React coexisteront. Il faut éviter toute dérive ou double maintenance inutile. Règle : une surface ne doit jamais avoir deux fronts canoniques simultanément.

---

## 10. Quick wins pour démarrer

Ces actions peuvent être faites tôt et débloquent plusieurs slices en parallèle :

- Sortir l'i18n de la dépendance implicite à Streamlit
- Définir des schémas API pour les pages Carrière et Match View
- Exposer un premier endpoint de figure Plotly JSON
- Recréer Explorer avec vraies tables web

---

## 11. Les 10 étapes critiques (check-list stratégique)

1. **Définir le périmètre exact** — décider ce qui reste identique, ce qui peut être amélioré, ce qui peut être supprimé. Sans ça, la migration dérive en refonte produit.
2. **Geler le cœur métier avant de toucher au front** — documenter chaque page. Si la logique métier, les calculs, les accès DuckDB, les règles d'auth et les agrégations bougent en même temps, impossible de savoir d'où viennent les régressions. Construire une matrice de parité écran par écran. Transformer ce cadrage en backlog de vertical slices priorisés.
3. **Extraire un vrai contrat d'API** — le point de bascule n'est pas React, c'est la création d'interfaces stables entre backend et UI : endpoints, payloads, erreurs, pagination, filtres, formats de graphes.
4. **Sortir toute la logique cachée dans Streamlit** — session state, query params, filtres, cache, navigation, dépendances implicites au rerun : c'est souvent là que la difficulté réelle est sous-estimée.
5. **Préparer la structure cible du repo** dans le worktree courant.
6. **Migrer par parcours métier, pas par couches techniques** — mieux vaut livrer un écran complet utilisable de bout en bout que 20 endpoints isolés ou 15 composants sans flux métier fini.
7. **Prévoir une phase de cohabitation** — ancien front toujours vivant, nouveau front branché sur le même backend, puis remplacement progressif écran par écran.
8. **Traiter auth, permissions et état de session très tôt** — beaucoup de migrations échouent non pas sur les graphes ou les tableaux, mais sur les sessions, les expirations de tokens, les préférences utilisateur et les flux de reconnexion.
9. **Mettre des tests de parité** — comparer ancien et nouveau résultat sur quelques écrans critiques : mêmes chiffres, mêmes filtres, mêmes règles de tri, mêmes agrégats.
10. **Mesurer la migration comme un produit** — temps de réponse, erreurs API, abandon utilisateur, usages des écrans, dette restante. Sans métriques, la migration devient une suite d'impressions.

---

## 12. Environnement de développement local

> **À mettre en place dès Slice 0, avant tout autre travail frontend.**

Si démarrer le projet dev prend plusieurs minutes ou requiert une authentification Xbox réelle à chaque redémarrage, le rythme sera brisé et la migration ralentie structurellement.

### Objectif : `make dev` en une commande

Un seul script qui lance :
- L'API FastAPI en hot-reload (`uvicorn apps/api/app/main.py --reload`)
- Le frontend Vite en hot-reload (`vite dev` dans `apps/web/`)
- Pointage sur le corpus de fixtures (`tests/fixtures/ref_player/`) en mode demo

### Mode DEMO_MODE obligatoire dès le départ

Le Device Code Flow Xbox ne peut pas être rejoué en boucle rapide pendant le dev. Il faut un flag `DEMO_MODE=true` (variable d'environnement ou `.env.local`) qui :
- Bypasse l'auth Xbox et retourne un `PlayerSummary` fictif ou de référence
- Pointe sur `tests/fixtures/ref_player/` au lieu des DBs de production
- Expose les même endpoints avec les mêmes schémas — seule la source de données change

**Règle** : le mode DEMO_MODE doit être fonctionnel avant la fin de Slice 0. Aucun travail frontend sérieux n'est possible sans lui.

### `.env.local` minimal à définir

```
DEMO_MODE=true
DEMO_PLAYER_SLUG=ref_player
REF_FIXTURES_DIR=tests/fixtures/ref_player
API_PORT=8000
VITE_API_BASE_URL=http://localhost:8000
```

### Makefile cible

```makefile
dev:
    DEMO_MODE=true uvicorn apps.api.app.main:app --reload &
    cd apps/web && npm run dev

dev-api:
    DEMO_MODE=true uvicorn apps.api.app.main:app --reload

dev-web:
    cd apps/web && npm run dev

test-parity:
    python -m pytest tests/parity/ -v
```

---

## 13. Impacts racine à anticiper

- `.gitignore` : couvrir `apps/web/node_modules`, `apps/web/dist`, caches outils frontend
- `Dockerfile` : build multi-étapes Python + frontend
- `Makefile` : exposer `run-api`, `run-web`, `run-all`
- `README.md` / `docs/INSTALL.md` : distinguer mode Streamlit legacy et mode FastAPI + React
- `tests/` : reste à la racine pour les tests Python, tests de parité et vérifications de contrats API

---

## 14. Stratégie de testing — Décisions figées

> **Les tests ne sont pas une option post-MVP. Ils font partie de chaque slice dès le premier jour.**
> Sans eux, la migration produit du code dont personne ne peut garantir la non-régression.

### Backend (Python)

| Type | Outil | Quand | Couverture cible |
|------|-------|-------|------------------|
| **Tests unitaires** | `pytest` | Chaque module `apps/api/services/` et `apps/api/routers/` | Chaque endpoint a au moins 1 test happy path + 1 test erreur |
| **Tests de parité** | `pytest` + corpus figé | Dès Slice 0 | Valeurs golden sur `filters/resolve`, career, match_history, match_view |
| **Tests de non-régression** | `pytest` + snapshots | Dès Slice 1 | Les payloads API ne changent pas silencieusement entre commits |
| **Tests de contrat OpenAPI** | `schemathesis` ou validation manuelle | Dès Slice 0 | Le schéma OpenAPI exporté est cohérent avec les réponses réelles |

Règles :
- Tout endpoint livré sans test unitaire est considéré comme non livré.
- Les tests de parité utilisent exclusivement `tests/fixtures/ref_player/`, jamais les DBs de production.
- Les snapshots de payload sont versionnés. Un changement de snapshot doit être explicitement justifié dans le commit.
- `python -m pytest tests/parity/` doit être vert avant toute bascule `preview → canonical`.

### Frontend (React/TypeScript)

| Type | Outil | Quand | Couverture cible |
|------|-------|-------|------------------|
| **Tests unitaires** | Vitest | Chaque store, chaque utilitaire `lib/` | Logique de stores, serializers, formatters |
| **Tests composant** | Vitest + React Testing Library | Chaque feature | Rendu, interactions, états loading/error/empty |
| **Tests E2E** | Playwright | Dès Slice 1 | Parcours critiques : Setup, Career, Match History, Match View |
| **Mocking API** | MSW (Mock Service Worker) | Dès Slice 0 | Handlers MSW couvrent tous les endpoints consommés |
| **Tests de non-régression visuelle** | Playwright screenshots (optionnel) | Post-MVP | Détection de dérives visuelles sur les pages stabilisées |

Règles :
- MSW est la couche de mocking unique. Aucun `jest.mock()` / `vi.mock()` d'axios ou fetch au niveau composant.
- Chaque feature livrée doit avoir au minimum : 1 test composant happy path, 1 test état erreur, 1 test état loading.
- Les tests E2E Playwright tournent contre l'API réelle en DEMO_MODE (pas de mock) pour valider le contrat bout en bout.
- Un parcours E2E cassé bloque le merge — pas de "on fixera plus tard".

### Matrice des tests par slice

| Slice | Tests backend obligatoires | Tests frontend obligatoires | E2E |
|-------|---------------------------|---------------------------|-----|
| 0a | bootstrap, players, DEMO_MODE | Shell mount, store hydration | Navigation shell |
| 0b | filters/resolve parité × 4 scopes | FilterStore sync URL | — |
| 1 | Setup machine d'état, device flow, settings CRUD | Setup wizard flow, settings form | Setup complet E2E |
| 2 | Career payload parité | Career page render + loading/error | Career E2E |
| 3 | Match history parité, pagination, export | Table pagination, sort, export | Match History E2E |
| 4 | Explorer search, matches, player query | Search + results + navigation | Explorer → Match View E2E |
| 5 | Match view payload parité, last match resolve | Tabs, scoreboard, nav prev/next | Match View E2E |

### Structure des tests

```
tests/
  unit/                   ← tests unitaires backend (services, routers)
  parity/                 ← tests de parité sur corpus figé
  snapshots/              ← snapshots de payloads API versionnés
  fixtures/
    ref_player/           ← DBs DuckDB figées
    scopes/               ← FilterContextInput de référence
    golden_values/        ← valeurs attendues par écran
apps/web/
  src/__tests__/          ← tests unitaires stores/utils
  src/features/*/__tests__/ ← tests composants par feature
  e2e/                    ← tests Playwright
  e2e/fixtures/           ← handlers MSW pour E2E
```

---

## 15. Stratégie de déploiement MVP

### Architecture de serving

- **Mode dev** : Vite dev server (`:5173`) + uvicorn API (`:8000`), proxy Vite vers `/api`
- **Mode production** : FastAPI sert les fichiers statiques React via `StaticFiles` mount + les routes API. Single container Docker.
- **Reverse proxy optionnel** : nginx devant FastAPI si besoin de TLS, rate limiting ou serving statique optimisé.
- **Pas de CDN** dans le MVP — les assets statiques sont servies depuis le même container.

### Dockerfile cible

```dockerfile
# Étape 1 : build frontend
FROM node:20-alpine AS web-build
WORKDIR /app/web
COPY apps/web/package*.json ./
RUN npm ci
COPY apps/web/ ./
RUN npm run build

# Étape 2 : runtime Python
FROM python:3.12-slim
WORKDIR /app
COPY pyproject.toml ./
RUN pip install --no-cache-dir -e .
COPY src/ src/
COPY apps/api/ apps/api/
COPY --from=web-build /app/web/dist apps/web/dist
EXPOSE 8000
CMD ["uvicorn", "apps.api.app.main:app", "--host", "0.0.0.0", "--port", "8000", "--workers", "1"]
```

### Health check

- `GET /api/v1/health` → `{ status: "ok", version: str, uptime_seconds: int }`
- Docker `HEALTHCHECK` pointe sur cet endpoint
- Le shell React consomme également `/health` pour afficher un indicateur de connectivité

### Cohabitation Streamlit pendant la migration

- Streamlit reste sur `:8501` (inchangé)
- FastAPI + React sur `:8000`
- Les deux consomment le même `src/` et les mêmes DBs DuckDB
- Aucun verrou de concurrence n'est nécessaire pour les **lectures** (DuckDB supporte le multi-reader)
- Les écritures (sync, backfill, settings) passent par un unique worker — voir § DuckDB concurrence dans INVARIANTS.md

---

## 16. Observabilité et logging

### Logging structuré

- **Backend** : `structlog` avec output JSON en production, output console formaté en dev
- **Chaque requête** : `request_id` (UUID v4) injecté via middleware, propagé dans tous les logs
- **Chaque erreur** : log structuré avec `request_id`, `endpoint`, `status_code`, `error_code`, `duration_ms`
- **Pas de PII dans les logs** : les gamertags sont OK (publics), les tokens/secrets jamais

### Métriques minimales (MVP)

- Latence par endpoint (p50, p95, p99) — mesurée par middleware
- Taux d'erreur par endpoint (4xx, 5xx)
- Compteur de requêtes par route
- Durée des jobs (sync, backfill, smoke test)

Pour le MVP, ces métriques sont loguées en JSON structuré. L'intégration avec Prometheus/Grafana ou un service tiers est post-MVP.

### Frontend

- Erreurs non attrapées envoyées vers un endpoint `POST /api/v1/telemetry/errors` (optionnel, sans dépendance externe)
- TanStack Query expose `onError` global : log console + notification utilisateur
- Pas de service de tracking tiers (analytics) dans le MVP

---

## 17. Génération de types TypeScript depuis Pydantic

**Décision : générer les types TS depuis le schéma OpenAPI exporté par FastAPI.**

Pipeline :
1. FastAPI exporte automatiquement `/openapi.json`
2. `openapi-typescript` génère les types dans `apps/web/src/lib/api/generated.ts`
3. Le script tourne dans le Makefile : `make gen-types`
4. Les types générés sont commités (pas générés au build)

Règles :
- Les types générés ne sont **jamais** édités manuellement
- Si un type généré ne correspond pas au besoin front, le schéma Pydantic est corrigé côté backend
- Le CI vérifie que les types générés sont à jour (`make check-types`)

### Versionning API

- Le front et l'API sont versionnés ensemble (monorepo). Pas de backward compat multi-consommateur requis.
- `/api/v1` est le seul préfixe. Un éventuel `/v2` n'est envisageable que si un consommateur externe apparaît.
- Un changement de schéma non rétrocompatible produit un changement de `data_version` dans `ApiMeta`, permettant au front de détecter l'incompatibilité.
