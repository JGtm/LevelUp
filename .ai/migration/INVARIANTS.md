# INVARIANTS.md — Contrats fonctionnels à ne pas casser

> Ce fichier liste tout ce qui doit rester sémantiquement identique après migration.
> À lire AVANT d'implémenter n'importe quel slice MVP.
> Source : PLAN_MIGRATION_FASTAPI_REACT.md § Étape critique 2 + Étape critique 4

---

## 1. Contexte applicatif commun

La quasi-totalité des pages consomment un contexte commun calculé dans `streamlit_app.py`. Tant que le front React n'a pas remplacé ce contexte par des contrats API explicites, ne pas changer la sémantique côté Python.

Éléments à préserver ou migrer explicitement :

- `df` : historique complet chargé une fois
- `dff` : historique filtré courant
- `base` : clone avant filtres
- `db_path`, `xuid`, `waypoint_player`, `me_name`, `db_key`, `aliases_key`
- `base_s_ui` : DataFrame de sessions servant de source de vérité pour plusieurs pages
- `match_view_params` : callbacks et loaders injectés dans Last Match, Explorer et Match View

**Cible React** : endpoint `bootstrap` + `filters/resolve` + `PageEnvelope[T]` comme enveloppe commune.

---

## 2. Modèle de filtres et de sessions — ⚠ Nœud central, risque sous-estimé

> **C'est le contrat le plus risqué de toute la migration — pas Plotly, pas le routing.**
>
> Le système de filtres est profondément imbriqué dans Streamlit : shadow keys pour survivre aux reruns, `GAP_MINUTES_FIXED = 120` encodé dans `filters_render.py`, widgets couplés à la logique de résolution des sessions, options cascade recalculées à chaque rerun. `FilterContextInput` + `filters/resolve` semble simple sur le papier, mais implémenter toutes les combinaisons (période / sessions / cascade) correctement, avec les mêmes compteurs et options que Streamlit, est le chantier le plus susceptible de produire des régressions invisibles.
>
> **Recommandation** : faire un spike dédié sur `filters/resolve` dans Slice 0, comparer les résultats sur les 4 scopes du corpus figé avant d'avancer vers Slice 1.

Le moteur de filtres est un invariant fonctionnel, pas un détail de rendu Streamlit.

Éléments à préserver sémantiquement :

- Mode de filtre : `Période` ou `Sessions`
- Filtres temporels : `start_date_cal`, `end_date_cal`
- Scope sessions : `picked_session_label`, `picked_solo_session_label`, `picked_squad_session_label`, `picked_sessions`
- Filtres cascade : experience types, playlists, modes, maps
- Modèle de sessions : `GAP_MINUTES_FIXED = 120` dans `src/app/filters_render.py`
- Mécanisme de shadow keys pour survivre aux reruns et à `st.navigation`

**Cible React** :
- `FilterContextInput` comme source de vérité front
- Endpoint `POST /api/v1/players/{player_slug}/filters/resolve` comme arbitre serveur des options, sessions et compteurs
- `useGlobalFilterStore` pour le brouillon d'interaction local
- Synchronisation URL seulement pour les filtres devant être partageables ou restaurables

### Algorithme canonique de résolution des filtres

Sur un dataset figé, `filters/resolve` doit être déterministe.

À dataset, `player_slug`, locale et `FilterContextInput` identiques, l'API doit renvoyer le même `FilterContextResolved`.

Ordre de résolution à préserver :

1. Normaliser l'entrée : remplir les blocs absents (`period`, `sessions`, `cascade`) avec leurs valeurs neutres.
2. Déterminer le mode effectif `period` ou `sessions` et invalider les combinaisons impossibles.
3. Calculer ou relire le modèle de sessions à partir de l'historique complet du joueur avec `GAP_MINUTES_FIXED = 120`.
4. Déterminer le sous-ensemble de matchs du scope primaire : période ou sessions.
5. Recalculer les options cascade à partir du sous-ensemble courant, dans l'ordre `experience_types -> playlists -> modes -> maps`.
6. Élaguer toute valeur sélectionnée devenue indisponible et renvoyer une entrée `effective` propre, jamais une sélection fantôme.
7. Renvoyer les compteurs avant/après filtres et les options de sessions cohérents avec le scope réellement retenu.

Règles complémentaires :

- Une valeur sélectionnée mais devenue invalide est supprimée côté serveur, pas conservée dans un état ambigu côté client.
- `counts.total_matches_before_filters` correspond au volume avant cascade secondaire, pas au nombre de lignes déjà réduit par un écran particulier.
- Les options renvoyées sont des options de vérité backend. Le frontend ne déduit ni exclusions ni listes disponibles à partir de ses propres caches.

### Cycle de synchronisation URL → store → API → queries

Le shell React ne doit jamais recréer les dérives de `session_state`.

Séquence cible à préserver :

1. La route et les search params hydratent un brouillon de filtre dans `useGlobalFilterStore`.
2. Le store déclenche `filters/resolve` pour produire un état engagé et normalisé.
3. Le backend renvoie `effective`, `available_options`, `session_options` et `counts`.
4. Le store remplace son état engagé par `effective` et recalcule le `filterContextHash`.
5. Les queries de page dépendent uniquement du `filterContextHash` engagé, jamais d'un brouillon en cours d'édition.
6. L'URL n'est mise à jour qu'à partir d'un état normalisé, jamais depuis un état partiel de widget.

---

## 3. Contrat de deep links et navigation interne

Ces query params sont des contrats de navigation minimaux à conserver :

- `page`
- `match_id`
- `gamertag`
- `player`
- `stats_view`
- `session`
- `scope`

États de navigation implicites à sortir de Streamlit (remplacer par routes ou search params explicites) :

- `_pending_page`, `_pending_match_id`, `_pending_gamertag`, `_pending_player`
- `match_id_input`
- `v7_current_section`, `v7_stats_view`, `v7_profile_view`
- `_last_match_nav_index`, `_last_match_nav_total`, `_last_match_nav_session_key`

**Règle** : tout état partageable doit être reconstructible par URL seule, sans `_pending_*`.

### Formes canoniques des deep links (alignées V7)

Les routes React canoniques doivent primer sur les anciens query params Streamlit.
Les routes suivent la structure V7 — voir [SLICES.md](migration/SLICES.md) § Correspondance.

Exemples de formes cibles :

- `/setup`
- `/settings`
- `/` (→ Accueil / Home Mission Control)
- `/players/:playerSlug/profile/career`
- `/players/:playerSlug/profile/citations`
- `/players/:playerSlug/stats/history?filter_mode=period&start_date=2025-01-01&end_date=2025-03-31`
- `/players/:playerSlug/stats/timeseries`
- `/players/:playerSlug/stats/sessions`
- `/players/:playerSlug/explorer?gamertag=MonCible`
- `/players/:playerSlug/explorer/matches/:matchId?session=SessionLabel&scope=squad`
- `/players/:playerSlug/last-match?filter_mode=sessions&picked_session_label=SessionLabel`
- `/players/:playerSlug/squad`
- `/players/:playerSlug/synthesis`
- `/players/:playerSlug/media`

Règles de priorité :

1. Les route params (`playerSlug`, `matchId`, `section`) priment toujours sur les search params.
2. Les search params expriment un contexte partageable ; ils ne servent pas à transporter des clés transitoires d'UI.
3. Le query param legacy `page` ne survit que comme mécanisme de compatibilité/redirect à l'entrée. Il n'est pas une source de vérité interne du router React.
4. Un `match_id` présent hors route canonique de match est traité comme un indice de redirection, pas comme un état de page durable.
5. Toute combinaison invalide est normalisée par le routeur ou le backend vers une forme unique, jamais laissée au libre arbitre d'un composant.

---

## 4. Identité, auth et bootstrap

Le démarrage applicatif dépend de plusieurs vérités à ne pas laisser diverger :

- Browser prefs restaurées avant le reste de l'application
- `setup_wizard` bloque l'app si la configuration est incomplète
- Auth Halo/Xbox via `provider.py` et Device Code Flow
- Sélection joueur via `db_path + xuid_input + waypoint_player`
- `app_settings` persistées sur disque et relues dans la session

**Cible React** :
- Endpoint `GET /api/v1/bootstrap` comme point d'entrée unique du shell
- Session backend opaque pour les tokens et secrets
- `useAppShellStore` pour l'état shell minimal
- localStorage pour les préférences purement navigateur

**Règle absolue** : les tokens Halo, le cache MSAL, les refresh tokens et les secrets restent exclusivement côté backend. Le navigateur ne reçoit jamais de token exploitable directement.

### Machine d'état minimale du setup à préserver

Le setup n'est pas une simple page de formulaire. C'est une machine d'état produit.

États minimaux à rendre observables côté API :

- `choose_mode`
- `auth_pending`
- `player_pending`
- `smoke_test_running`
- `done`
- `failed` le cas échéant pour une tentative courante

Invariants :

- Un refresh navigateur en plein Device Code Flow ou en plein smoke test ne doit pas faire perdre l'état observable côté utilisateur.
- Un attempt expiré doit réapparaître explicitement comme expiré, pas retomber silencieusement en état initial.
- Tant que `done` n'est pas atteint, l'accès aux routes protégées reste bloqué par le bootstrap.

---

## 5. Caches et services d'arrière-plan

La migration UI ne doit pas casser la sémantique des caches/process workers existants :

- `background_media_indexing`
- `reindex_media_after_sync`
- Process cache de la home V7 pour battle pass / challenges
- Caches de repository et de sessions
- Tailscale funnel optionnel

**Cible React** : endpoints de jobs avec `AsyncJobStatus`, polling ou streaming côté front, start/progression/invalidation explicites à la fin d'un job.

---

## 6. I18n et labels métier

Le bilinguisme FR/EN et les labels métier sont des invariants de parité, pas des détails cosmétiques.

À préserver :

- Labels pages
- Labels modes/cartes/playlists
- Badges outcome / domination / humiliation
- Rangs carrière / CSR / LUSR
- Textes settings / wizard / media / citations

**Cible React** : le catalogue `src/ui/i18n/` est conservé. La langue est reçue explicitement depuis le frontend (header ou payload). Ne plus dépendre de `st.session_state["lang"]`.

### Source de vérité de la locale

Ordre de priorité à préserver dans le shell React :

1. Choix explicite de l'utilisateur dans l'UI courante
2. Préférence navigateur persistée par le front
3. Valeur persistée dans `app_settings.lang` renvoyée par `bootstrap`
4. `Accept-Language` ou préférence système si aucune préférence n'existe
5. Fallback final : `fr`

Une seule locale active doit piloter l'ensemble du shell et des appels API. Les composants ne choisissent jamais leur langue localement.

---

## 7. Modèle de permissions et de capacités

Le front doit pouvoir distinguer sans heuristiques côté client :

| Capacité | Condition requise |
|----------|------------------|
| Lecture locale des données synchronisées | Toujours |
| Actions nécessitant une auth Halo valide | sync, refresh live, battle pass, defis, device flow |
| Actions d'exploitation locale | reset index media, watcher, setup, changements de settings sensibles |
| Actions longues ou potentiellement destructrices | confirmation + suivi de job |

Ces capacités viennent du backend dans les payloads `bootstrap` ou de page — jamais d'heuristiques UI cachées dans les composants.

### Modèle d'identité et d'accès joueur

- `player_slug` dans l'URL est un identifiant de navigation. Il n'est jamais considéré comme une preuve d'autorisation.
- Le backend résout `player_slug` vers le profil local, puis vers le couple `db_path` / `xuid` réellement exploitable.
- Les routes qui lisent une player DB exigent que le profil soit connu de l'installation ou du contexte de session.
- Les lectures cross-player via Explorer, Match View ou Teammates restent ancrées au périmètre de données accessible depuis le joueur courant et les bases partagées ; elles ne transforment pas l'API en annuaire global arbitraire.
- Toute capacité affichée dans le front doit être dérivée d'un payload backend (`bootstrap` ou page), jamais d'une supposition à partir du seul `player_slug`.

---

## 8. Matrice de remplacement état Streamlit → état React

| Type de logique | Mécanisme Streamlit actuel | Cible React/FastAPI | Règle |
|---|---|---|---|
| Navigation partageable | query params + `_pending_*` | Router + search params | tout état partageable doit être reconstruisible par URL seule |
| Données serveur | cache Streamlit + rerun | TanStack Query + API | aucune donnée distante ne doit dépendre d'un rerun global |
| État UI local | `st.session_state` | Zustand ou state composant | ne promouvoir en store que ce qui traverse plusieurs composants |
| Préférences navigateur | browser storage + session_state | localStorage / IndexedDB | aucune préférence non sensible ne doit transiter par session backend sans raison |
| Auth et secrets | process cache + session Streamlit | session backend opaque | aucun token ne fuit vers le navigateur |
| Jobs longs | side effects + rerun | endpoints jobs + invalidation | la fin de job doit être visible sans recharger toute l'app |

---

## 9. Anti-patterns à éliminer explicitement

- Un rendu de composant qui écrit dans l'état global pour préparer un rerun suivant
- Un filtre dont la valeur réelle n'est connue qu'après rerun complet de la page
- Une navigation qui passe par une clé temporaire au lieu d'une URL explicite
- Une mutation serveur qui suppose un refresh total pour rendre l'écran cohérent
- Un cache sans propriétaire clair, sans politique d'expiration et sans convention d'invalidation

---

## 10. Ordre d'extraction recommandé

1. Sortir le shell, le bootstrap, le joueur courant et les query params canoniques
2. Sortir le moteur de filtres globaux derrière `FilterContextInput` et `filters/resolve`
3. Sortir setup, auth et settings des enchaînements de rerun
4. Sortir les états de page locaux dans les features MVP : Match History, Explorer, Match View, Last Match
5. Sortir les caches et jobs de fond derrière des invalidations et statuts explicites

---

## 11. Étape critique 4 — Inventaire détaillé des logiques cachées à extraire

> Source : PLAN_MIGRATION_FASTAPI_REACT.md § Étape critique 4
> Cette section est le chantier d'architecture concret. Le vrai sujet n'est pas le rendu UI — c'est extraire les mécanismes dispersés entre `st.session_state`, query params, caches Streamlit, callbacks de rendu et reruns implicites.

### Objectif opérationnel

- Identifier toute logique qui dépend aujourd'hui du moteur de rerun Streamlit plutôt que d'un contrat explicite
- Assigner à chaque catégorie d'état une cible unique : URL, store front, session backend, cache serveur ou persistence navigateur
- Supprimer les couplages implicites entre rendu, chargement de données, navigation et side effects
- Rendre chaque parcours React reproductible à partir d'une URL, d'un contexte de session backend et d'appels API explicites

### Principe directeur

**Tout état doit avoir un seul propriétaire légitime.**

| Type d'état | Propriétaire cible |
|---|---|
| Navigable et partageable | URL |
| Données serveur distantes | TanStack Query + backend |
| État UI éphémère local | Zustand ou état composant |
| Préférences navigateur non sensibles | localStorage ou IndexedDB |
| Auth, joueur courant, jobs longs, secrets | Session backend |

### Inventaire des logiques cachées par catégorie

#### Catégorie 0 — `PageContext` et callbacks injectés

Aujourd'hui certaines pages ne consomment pas seulement des données. Elles reçoivent aussi des fonctions Python injectées via `PageContext`.

Éléments à sortir :
- `MatchViewParams`
- `FilterSidebarCallbacks`
- `TeammateCallbacks`
- Les loaders/callbacks injectés comme `load_match_medals_fn`, `load_highlight_events_fn`, `load_match_rosters_fn`, `render_match_view_fn`

Remplacement cible :
- services de page explicites côté API
- payloads de page ou sous-resources lazy documentées
- disparition de l'injection de callables dans le front au profit de contrats HTTP nommés

#### Catégorie 1 — Navigation et deep links

Aujourd'hui, une partie importante de la navigation est reconstruite via `st.query_params`, redirects internes et clés temporaires `_pending_*`.

Éléments à sortir :
- `page`, `match_id`, `gamertag`, `player`, `stats_view`, `session`, `scope`
- `_pending_page`, `_pending_match_id`, `_pending_gamertag`, `_pending_player`
- `v7_current_section`, `v7_stats_view`, `v7_profile_view`
- `_last_match_nav_index`, `_last_match_nav_total`, `_last_match_nav_session_key`

Remplacement cible :
- TanStack Router pour la route et les search params
- Route params explicites pour `playerSlug`, `matchId`, `section`
- Search params uniquement pour les états partageables ou restituables au refresh
- Disparition des clés `_pending_*` au profit d'une navigation déclarative

#### Catégorie 2 — Filtres globaux et modèle de sessions

Le système actuel combine contexte chargé dans `streamlit_app.py`, shadow keys, widgets de filtre et reruns pour maintenir la cohérence du scope.

Éléments à sortir :
- `filter_mode`, `start_date_cal`, `end_date_cal`
- `picked_session_label`, `picked_solo_session_label`, `picked_squad_session_label`, `picked_sessions`
- Filtres cascade `experience_types`, `playlists`, `modes`, `maps`
- La logique de `GAP_MINUTES_FIXED = 120`
- Les shadow keys servant à survivre aux reruns ou à `st.navigation`

Remplacement cible :
- `FilterContextInput` comme source de vérité front
- Endpoint `filters/resolve` comme arbitre serveur des options, sessions et compteurs
- `useGlobalFilterStore` pour le brouillon d'interaction local
- Synchronisation URL seulement pour les filtres devant être partageables ou restaurables

#### Catégorie 3 — État de page et interaction locale

Beaucoup de comportements locaux sont aujourd'hui masqués dans `session_state` alors qu'ils relèvent d'un état purement UI ou d'une dérivée de la route.

Éléments à sortir :
- `match_id_input`, `_explorer_selected_match`
- `compare_session_a`, `compare_session_b`, `_last_picked_for_compare`
- `teammates_picked_labels`, `_cache_warning_shown`, `show_records`
- `_lb_state`, `mv2_autoplay`
- États de panneaux, d'onglets, de tri local, de lightbox et de sélection ligne

Remplacement cible :
- Zustand pour l'état local transverse à une feature
- État composant pur quand l'information ne sort pas du composant
- Route/search params seulement si l'état doit être partageable ou restaurable par URL

#### Catégorie 4 — Bootstrap, setup et préférences utilisateur

Le bootstrap actuel dépend d'un ordre de rerun : restauration des prefs navigateur, lecture des settings, vérification setup, auth, sélection joueur, puis affichage de l'app.

Éléments à sortir :
- `_setup_mode`, `_xbox_oauth_result`, `_smoke_*`
- Lecture/patch de `app_settings`
- Restauration de `lang`, `show_hints`, dernier joueur utilisé et autres prefs navigateur
- Sélection joueur via `db_path`, `xuid_input`, `waypoint_player`

Remplacement cible :
- Endpoint `bootstrap` unique pour hydrater le shell
- Endpoints `setup/*`, `auth/*`, `settings/*` pour chaque mutation critique
- `useAppShellStore` pour l'état shell minimal
- localStorage pour les préférences purement navigateur
- Session backend pour la progression du setup et les états auth sensibles

#### Catégorie 5 — Cache, invalidation et dépendances au rerun

Une partie de la cohérence de l'app vient du fait que Streamlit rerun toute la page après un clic, un changement de filtre ou une mutation de cache. Cette mécanique doit être remplacée par des invalidations explicites.

Éléments à sortir :
- `src/ui/_cache_core.py`, `src/ui/_cache_loading.py`, `src/ui/_cache_queries.py`, `src/ui/_cache_sessions.py`, `src/ui/cache.py`
- `src/app/cache_control.py` et toute invalidation basée sur refresh global
- Les dépendances implicites où un rerun recalcule l'écran entier pour obtenir un état cohérent

Remplacement cible :
- TanStack Query pour la cohérence des données côté front
- Cache backend ou process cache explicite pour les ressources coûteuses qui restent côté Python
- Invalidation par query keys, mutations et fin de job — jamais par rerun global implicite
- `freshness` et warnings exposés par l'API plutôt que déduits du comportement de cache Streamlit

#### Catégorie 6 — Jobs longs et services en arrière-plan

Les syncs, backfills, scans media et quelques caches home fonctionnent aujourd'hui en s'appuyant sur le runtime process Streamlit et des side effects au démarrage.

Éléments à sortir :
- `background_media_indexing`
- `reindex_media_after_sync`
- Smoke test, sync et backfill pilotant ensuite l'UI via rerun
- Process cache de la home V7 pour battle pass / challenges

Remplacement cible :
- Endpoints de jobs avec `AsyncJobStatus`
- Polling ou streaming côté front selon le coût réel du besoin
- Start explicite, progression explicite, invalidation explicite à la fin d'un job
- Les traitements de fond ne doivent plus supposer l'existence d'une session Streamlit vivante

### Définition of done pour l'étape 4

L'étape critique 4 est considérée comme couverte si :

- Chaque clé `session_state` encore nécessaire à un parcours MVP a été classée dans une catégorie cible explicite
- Chaque deep link utile est porté par une vraie route ou un search param stable, sans `_pending_*`
- Les filtres globaux ont un contrat d'entrée/sortie explicite et ne dépendent plus des shadow keys Streamlit
- Aucune mutation critique du MVP n'a besoin d'un rerun global pour mettre l'écran dans un état cohérent
- Les jobs longs, préférences navigateur, caches et sessions ont chacun un propriétaire unique et observable

---

## 12. DuckDB et concurrence — contrainte critique

> **DuckDB est single-writer.** Toute écriture concurrente (sync, backfill, settings write, media index reset) doit être sérialisée.

### Règles de concurrence

- **Lectures** : DuckDB supporte le multi-reader. Plusieurs requêtes de lecture en parallèle (FastAPI endpoints, Streamlit) fonctionnent sans verrou.
- **Écritures** : une seule connexion en écriture à la fois par fichier `.duckdb`. Toute tentative d'écriture concurrente bloque ou échoue.
- **Conséquence MVP** : `uvicorn --workers=1` est **obligatoire** tant que les écritures ne passent pas par un mécanisme de sérialisation explicite.
- **Conséquence Slice 0** : la cohabitation Streamlit (`:8501`) + FastAPI (`:8000`) est safe pour les lectures. Mais si les deux tentent d'écrire simultanément (sync depuis Streamlit, settings depuis React), il y aura conflit. Pendant la cohabitation, les mutations d'écriture restent côté Streamlit jusqu'à ce que le setup/settings soient migrés en canonical React.

### Si multi-worker devient nécessaire

Options, par ordre de préférence :
1. **Queue de jobs sérialisée** (processus background worker dédié aux écritures)
2. **Lock fichier advisory** (`fcntl` / `msvcrt` par plateforme)
3. **Basculer les écritures vers un service séparé** communicant via queue

### Impact sur le cache MSAL et les tokens

Le cache MSAL est aujourd'hui sérialisé dans `sync_meta` (table DuckDB dans `stats.duckdb`). En mode single-worker :
- Le `SerializableTokenCache` fonctionne tel quel — un seul worker lit et écrit.
- Le refresh token silencieux continue via `provider.py` sans changement.

En mode multi-worker futur :
- Le token MSAL ne doit pas être caché en mémoire d'un seul worker. Il faut relire `sync_meta` à chaque besoin ou utiliser un cache partagé (fichier, Redis).
- Le refresh token MSAL doit être protégé contre les races (deux workers qui refreshent simultanément = invalidation du token).

**Décision MVP** : ne pas changer le modèle tokens. Le single-worker élimine le problème. Documenter le risque pour la montée en charge future.
