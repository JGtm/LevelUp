# Plan de migration Streamlit -> FastAPI + React

## Objectif

Remplacer la couche UI Streamlit de LevelUp par une interface web moderne basee sur :

- Backend : FastAPI + Pydantic
- Frontend : React + TypeScript + Vite
- Navigation / data fetching : TanStack Router + TanStack Query
- Etat UI local : Zustand
- UI/design : Tailwind v4 + shadcn/ui + Framer Motion
- Tables riches : AG Grid Community
- Graphes : Plotly conserve dans un premier temps via JSON de figures

Le but n'est pas de reecrire tout le produit. Le but est de remplacer la facade UI, d'ameliorer radicalement le design et l'UX, et de preserver au maximum le capital technique deja investi dans la couche Python.

## Resume executif

- La migration est faisable sans reset complet du projet.
- La couche UI actuelle est fortement couplee a Streamlit.
- Le noyau metier Python est largement reutilisable.
- Il est recommande de garder le runtime backend Python pendant la migration.
- Changer le runtime backend maintenant serait une migration dans la migration.

Ordre de grandeur observe lors de l'audit :

- Environ 185 fichiers dans la surface src/ui/
- Environ 26 fichiers dans la surface src/app/
- Environ 47 modules Plotly dans src/visualization/
- Environ 121 imports Streamlit dans src/

## Decision runtime

## Recommandation

Si par runtime tu parles du runtime backend, ma recommandation est claire :

- Garder Python 3.12 pour le backend pendant la migration
- Changer uniquement le runtime UI vers le navigateur via React / TypeScript

## Pourquoi garder Python maintenant

- Toute la valeur metier est deja la : DuckDB, Polars, Pydantic, sync, auth, repositories, analyses, scores, sessions, parser d'armes, services de pages.
- Le couple DuckDB + Polars + Pydantic est deja tres adapte a ton domaine.
- Les modules de visualisation Plotly sont deja en Python et peuvent etre exposes en JSON sans reecriture immediate.
- Refaire le backend en Node, Go ou Rust obligerait a reimplementer une enorme quantite de logique deja stable.
- Le vrai irritant que tu veux eliminer est Streamlit, pas Python.

## Quand reconsidérer le runtime backend

On pourra reouvrir le sujet plus tard si au moins une de ces conditions apparait :

- besoin d'une infra tres orientee microservices ou edge
- besoin de tres forte concurrence reseau cote API
- besoin de sortir une partie calcul critique dans un runtime specialise
- besoin de mutualiser le backend avec un autre produit non-Python

## Ce que je ne recommande pas maintenant

- Basculer le backend sur Node/NestJS maintenant
- Basculer le backend sur Go maintenant
- Basculer le backend sur Rust maintenant

Ces options peuvent avoir du sens a long terme sur un sous-systeme cible. Elles sont mauvaises comme point de depart pour cette migration, car elles detruisent le principal levier de vitesse : la reutilisation de l'existant.

## Audit exhaustif des changements a operer

## 1. Ce qu'on garde tel quel

### Couche data et repositories

Conserver telle quelle, exposee ensuite via FastAPI :

- src/data/repositories/duckdb_repo.py
- src/data/repositories/factory.py
- la majorite des mixins et helpers dans src/data/repositories/

Pourquoi :

- acces DuckDB deja structure
- logique de resolution XUID / gamertag deja en place
- connaissances schema deja encodees

### Couche d'analyse pure

Conserver telle quelle :

- src/analysis/

Modules representatifs :

- src/analysis/match_cadence.py
- src/analysis/match_intensity.py
- src/analysis/performance_score.py
- src/analysis/sessions.py
- src/analysis/skill_rating.py
- src/analysis/friends_impact.py
- src/analysis/objective_participation.py
- src/analysis/weapon_parser.py

Pourquoi :

- zero ou quasi-zero dependance UI
- bonne separation algorithmique
- forte valeur metier

### Couche sync et migrations

Conserver telle quelle :

- src/data/sync/
- scripts/sync.py
- scripts/backfill_data.py

Fichiers cles :

- src/data/sync/engine.py
- src/data/sync/migrations.py
- src/data/sync/scope.py

Pourquoi :

- aucune raison de reecrire ce moteur pour changer d'UI

### Services de page deja bien extraits

Conserver et reutiliser dans l'API :

- src/data/services/timeseries_service.py
- autres services dans src/data/services/

### Modeles et schemas

Conserver et reutiliser :

- modeles Pydantic et dataclasses metier existants
- schemas de domaine et resultats de service

## 2. Ce qu'on garde avec adaptation

### Visualisations Plotly

Conserver la logique de generation des figures, mais changer le mode de livraison.

Modules cles :

- src/visualization/
- src/visualization/theme.py

Adaptation a faire :

- retourner ou exposer fig.to_plotly_json() ou equivalent
- ne plus rendre les figures via st.plotly_chart
- creer des endpoints de figures ou d'agregats

### Authentification

Conserver la logique, adapter l'execution :

- src/auth/provider.py
- src/auth/_msal.py
- src/auth/_halo_exchange.py

Adaptation a faire :

- sortir du cache process local pour les tokens
- gerer la session cote API
- prevoir cookies httpOnly ou session backend

### i18n

Conserver le catalogue, adapter la source de verite :

- src/ui/i18n/

Adaptation a faire :

- ne plus dependre de Streamlit pour la langue courante
- recevoir explicitement la langue depuis le frontend quand necessaire

### Configuration et exploitation

Adapter :

- launcher.py
- run.sh
- Dockerfile
- pyproject.toml
- README.md

## 3. Ce qu'il faut reecrire ou retirer

### Shell applicatif Streamlit

Retirer ou remplacer :

- streamlit_app.py
- streamlit_app_v7.py
- src/app/page_router.py
- la logique de navigation basee sur st.navigation, st.switch_page et st.query_params cote Streamlit

### Etat de session Streamlit

Retirer ou remplacer :

- src/app/session_keys.py
- src/app/state.py
- les usages intensifs de st.session_state dans src/app/ et src/ui/

Remplacement cible :

- query params URL pour l'etat navigable
- TanStack Query pour l'etat serveur
- Zustand pour l'etat UI local
- localStorage pour les preferences locales

### Cache UI Streamlit

Retirer ou remplacer :

- src/ui/_cache_core.py
- src/ui/_cache_loading.py
- src/ui/_cache_queries.py
- src/ui/_cache_sessions.py
- src/ui/cache.py
- src/app/cache_control.py

Remplacement cible :

- cache HTTP et invalidation cote backend pour les ressources couteuses
- TanStack Query cote frontend

### Pages et composants fortement couples

Ces modules ne se portent pas, ils se remplacent :

- src/ui/pages/
- src/ui/layout/
- src/ui/components/ dont les composants Streamlit purs
- src/app/filters_render.py
- src/app/filters.py
- src/app/sidebar.py
- src/ui/streamlit_modern.py

### Medias et lightbox

Reecriture complete recommandee :

- src/ui/components/media_lightbox.py
- src/ui/pages/media_v2_grid.py
- src/ui/pages/media_v2.py
- src/ui/pages/media_library_render.py

### Browser storage serveur

Retirer ou refondre :

- src/ui/components/browser_storage/__init__.py

Remplacement cible :

- localStorage natif
- IndexedDB si besoin pour des donnees plus lourdes

## 4. Modules mixtes a decouper avant ou pendant la migration

Les fichiers suivants melangent encore logique metier, orchestration et rendu UI :

- src/ui/pages/timeseries.py
- src/ui/pages/teammates.py
- src/ui/pages/explorer.py
- src/ui/pages/session_compare.py
- src/ui/pages/match_view.py
- src/app/filters_render.py
- src/ui/pages/media_v2_grid.py
- src/ui/pages/home_mission_control.py

Modules deja mieux prepares pour l'extraction :

- src/ui/pages/match_view_logic.py
- src/ui/pages/session_compare_logic.py
- src/ui/pages/home_mission_control_logic.py
- src/data/services/timeseries_service.py

## 5. Impact hors code applicatif

### Build et runtime

Travaux a prevoir :

- nouveau frontend avec package.json
- nouveau build multi-etapes pour le frontend
- nouveau point d'entree backend FastAPI
- nouveau mode dev local pour lancer API + web
- refonte partielle du Dockerfile

### Documentation

Travaux a prevoir :

- documentation d'installation
- documentation de lancement local
- documentation de deploiement
- documentation des endpoints si OpenAPI non suffisante

## Architecture cible recommande

## Structure repo cible

Je recommande de ne pas deplacer src/ au debut. Il sert de noyau Python commun.

Structure conseillee :

```text
apps/
  api/
    app/
      main.py
      routers/
      schemas/
      deps/
      services/
  web/
    src/
      routes/
      components/
      features/
      stores/
      lib/
src/
  analysis/
  data/
  auth/
  visualization/
```

## Responsabilites

### Backend FastAPI

- appeler les repositories et services Python
- orchestrer l'auth
- exposer les donnees et figures
- piloter la sync et remonter son etat

### Frontend React

- gerer le shell applicatif
- gerer le routing et les deep links
- gerer la navigation, les modales, les lightboxes, les drawers, les transitions
- afficher les graphes Plotly et les tables riches

## Strategie Plotly

## Recommandation

Ne pas reecrire les graphes au debut.

Strategie conseillee :

- backend : continue de construire les figures en Python
- backend : expose les figures en JSON Plotly
- frontend : affiche ces figures via react-plotly.js

## Ce qui doit sortir en JSON Plotly rapidement

- carriere
- last match
- match view
- une grande partie de timeseries
- teammates

## Ce qui doit sortir plutot en donnees brutes

- explorer
- historique
- certaines cartes de la home
- listes recentes, tables, badges, cartes d'action

## Strategie par page

### A migrer tot avec figures backend

- Carriere
- Dernier match
- Match View

### A migrer tot avec donnees brutes + composants React

- Explorer
- Historique
- Parametres

### A repousser apres fondations stables

- Timeseries
- Teammates
- Session Compare
- Media

## Plan de migration detaille

## Phase 0 - Cadrage

Objectif : figer la cible avant d'ecrire du code.

Travaux :

- choisir la verite produit entre shell legacy et shell V7
- lister les ecrans cibles du MVP
- definir les contrats API et les schémas principaux
- decider l'URL model de l'application

Decision recommandee :

- prendre la logique V7 comme cible produit
- ne pas reconstruire deux shells en parallele

## Section dediee - Perimetre exact de la migration

Cette section tranche explicitement l'etape critique 1. Elle sert de garde-fou produit pendant toute la migration.

### Decision de perimetre

- La migration concerne le remplacement progressif de la facade UI Streamlit par un couple FastAPI + React.
- Le backend Python actuel reste la source de verite pendant toute la migration : repositories DuckDB, services, analyses, sync, auth Halo, modeles Pydantic, generation Plotly.
- La cible produit de reference est le shell V7 et ses parcours metier, pas un melange legacy/V7 reinterprete ecran par ecran.
- La migration doit etre menee en cohabitation controlee : ancien front encore vivant, nouveau front branche sur le meme coeur backend, remplacement progressif par parcours.

### Ce qui doit rester strictement identique

- Les definitions metier et les calculs existants : performance score, sessions, skill rating, citations, stats de match, resolution gamertag/xuid, agregats et regles de tri.
- Les sources de donnees et leur architecture : DuckDB v6, metadata/shared/player DBs, SyncScope, sync/backfill, repositories et vues SQL garanties.
- Les comportements fonctionnels des parcours migrés : memes chiffres, memes filtres, memes regroupements, memes permissions, memes labels metier, memes regles de pagination et de recherche a semantique equivalente.
- Le contrat fonctionnel de l'auth Halo : meme capacite a ouvrir une session, a reutiliser les credentials valides et a proteger les operations necessitant une authentification.
- Le bilinguisme FR/EN : aucune regression sur les textes exposes a l'utilisateur lors des parcours migrés.

### Ce qui peut etre ameliore sans sortir du perimetre

- Le design system, la hierarchie visuelle, les layouts, le responsive, les transitions et la qualite percue.
- Le modele de navigation : URLs plus propres, deep links explicites, navigation plus fluide, separation claire entre etat navigable et etat local.
- L'experience data : meilleurs loading states, meilleurs empty states, erreurs plus lisibles, tables web riches, drill-down plus naturel.
- Le mode de livraison technique : endpoints explicites, pagination serveur, cache HTTP, TanStack Query, Zustand, cookies httpOnly, JSON Plotly.
- Les performances front et la reduction de la dependance aux reruns implicites de Streamlit.

### Ce qui est explicitement hors perimetre de cette migration

- Reecrire le runtime backend dans une autre techno que Python.
- Refaire le schema de donnees, changer DuckDB, remplacer Polars ou revoir le moteur de sync.
- Ajouter de nouvelles briques produit qui n'existent pas deja en Streamlit, sauf micro-ajustements UX necessaires a la migration.
- Reecrire tous les graphes dans une autre librairie des le depart.
- Repenser en profondeur le modele analytique, les scores, les rules engines ou les metriques metier pendant la migration UI.
- Transformer le chantier en rebranding complet, application mobile, produit multi-tenant ou plateforme publique generalisee.

### Perimetre MVP recommande

Le MVP React/FastAPI doit couvrir un parcours complet et visible de bout en bout, sans pretendre remplacer tout Streamlit d'un coup.

Inclure dans le MVP :

- shell applicatif moderne
- auth minimale exploitable cote API
- navigation URL-first
- Carriere
- Explorer / Historique
- Match View
- Parametres minimums necessaires au fonctionnement

Laisser hors MVP initial, en cohabitation avec Streamlit :

- Home Mission Control complete
- Timeseries
- Teammates
- Session Compare
- Media
- Objective Analysis et autres ecrans analytiques denses

### Regle de decision pour tout futur ajout au plan

Une demande entre dans le perimetre si elle respecte simultanement ces conditions :

- elle remplace ou supporte un parcours deja existant
- elle n'impose pas de changer la logique metier de reference
- elle est necessaire a la parite fonctionnelle ou a l'UX minimale du nouvel ecran

Une demande sort du perimetre si elle implique au moins un de ces cas :

- nouveau besoin produit sans equivalent actuel
- changement de calcul, de schema ou de source de verite metier
- chantier transverse backend non requis pour exposer l'existant via API

### Criteres de fin pour cette etape de cadrage

L'etape critique 1 est consideree comme couverte si les decisions suivantes ne bougent plus sans nouvelle decision explicite :

- cible produit de reference = V7
- source de verite backend = Python existant
- forme de migration = progressive par vertical slices
- MVP cible = shell + Carriere + Explorer/Historique + Match View + auth minimale
- hors scope = reecriture backend, refonte metier, big bang total

## Phase 1 - Fondations backend

Objectif : rendre le noyau Python consommable via API.

Travaux :

- creer l'app FastAPI
- brancher les repositories existants
- exposer les schemas Pydantic utiles
- adapter l'auth a un modele API
- exposer un endpoint de healthcheck

Livrables :

- API runnable localement
- OpenAPI exploitable
- auth backend minimale fonctionnelle

## Phase 2 - Fondations frontend

Objectif : poser le shell moderne.

Travaux :

- initialiser Vite + React + TypeScript
- brancher Tailwind v4
- installer shadcn/ui
- installer TanStack Router, Query, Zustand, Framer Motion
- poser layout, theme, navigation, erreurs, loading states

Livrables :

- shell moderne
- routage URL-first
- patterns de page et de data fetching fixes

## Phase 3 - Premier vertical slice

Objectif : prouver le modele end-to-end.

Ordre recommande :

1. Carriere
2. Dernier match

Pourquoi :

- forte valeur demonstrative
- bon ratio impact / complexite
- peu de dependance aux composants exotiques

## Phase 4 - Tables solides

Objectif : regler le probleme Explorer / Historique.

Travaux :

- endpoints pagines et filtres serveur
- AG Grid Community ou equivalente pour tableaux riches
- tri, recherche, colonnes, selection, drill-down

Pages cible :

- Explorer
- Historique

## Phase 5 - Match View

Objectif : migrer la page detail de match, centrale dans le produit.

Travaux :

- scoreboards
- tabs/detail panels
- timelines et armes
- rencontres / nemesis

## Phase 6 - Home Mission Control

Objectif : reconstruire la home V7 avec un vrai niveau de finition UX.

Travaux :

- hero
- cartes d'action
- sections recentes
- battle pass / challenges

## Phase 7 - Media

Objectif : remplacer la partie la plus anti-Streamlit du produit.

Travaux :

- lightbox React
- grille performante
- likes persistants
- filtres media natifs

## Phase 8 - Gros chantiers analytiques

Objectif : migrer les zones les plus denses quand les fondations sont stables.

Pages cible :

- Timeseries
- Teammates
- Session Compare
- Objective Analysis

## Quick wins recommandes

- sortir l'i18n de la dependance implicite a Streamlit
- definir des schemas API pour les pages Carriere et Match View
- exposer un premier endpoint de figure Plotly JSON
- recreer Explorer avec vraies tables web

## Risques principaux

## Risque 1 - Migration du state model

Le plus gros risque n'est pas Plotly. C'est le passage de :

- session_state Streamlit

vers :

- URL + cache serveur + etat UI local + session backend

## Risque 2 - Auth et cache tokens

Le modele actuel convient a Streamlit mais doit etre refondu pour FastAPI multi-processus.

## Risque 3 - Pages trop mixtes

Certaines pages doivent etre decoupees avant d'etre migrables proprement.

## Risque 4 - Double produit temporaire

Pendant une partie de la migration, Streamlit et React coexisteront. Il faut eviter toute derive ou double maintenance inutile.

## Recommandation finale

- Oui au remplacement de Streamlit
- Oui au maintien du runtime backend Python pendant la migration
- Non a un changement de runtime backend maintenant
- Oui a une migration en tranches verticales, pas en big bang

## Premiere tranche recommandee

Si on demarre concretemenent, je recommande cette sequence :

1. figer l'architecture repo cible
2. monter FastAPI et le frontend Vite
3. sortir Carriere
4. sortir Explorer
5. sortir Match View

Ce chemin te donne tres vite une preuve visible que le produit devient plus beau, plus libre en design, et plus robuste en UX sans sacrifier ton coeur analytique.


## Les étapes vraiment critiques v7

1. Définir le périmètre exact de la migration. Il faut décider ce qui doit rester strictement identique, ce qui peut être amélioré, et ce qui peut être supprimé. Sans ça, la migration dérive en refonte produit.
2. Geler le cœur métier avant de toucher au front en documentant chaque page en détails. Si la logique métier, les calculs, les accès DuckDB, les règles d’auth et les agrégations bougent en même temps, vous ne saurez plus d’où viennent les régressions.
- Construire une matrice de parite ecran par ecran entre Streamlit et React.
- Transformer ce cadrage en backlog de vertical slices priorises.
3. Extraire un vrai contrat d’API. Le point de bascule n’est pas React, c’est la création d’interfaces stables entre backend et UI : endpoints, payloads, erreurs, pagination, filtres, formats de graphes.
4. Sortir toute la logique cachée dans Streamlit. Session state, query params, filtres, cache, navigation, dépendances implicites au rerun : c’est souvent là que la difficulté réelle est sous-estimée.
5. Préparer la structure cible du repo dans le worktree courant
6. Migrer par parcours métier, pas par couches techniques. Il vaut mieux livrer un écran complet utilisable de bout en bout que 20 endpoints isolés ou 15 composants sans flux métier fini.
7. Prévoir une phase de cohabitation. Le meilleur pattern est souvent : ancien front toujours vivant, nouveau front branché sur le même backend, puis remplacement progressif écran par écran.
8. Traiter auth, permissions et état de session très tôt. Beaucoup de migrations échouent non pas sur les graphes ou les tableaux, mais sur les sessions, les expirations de tokens, les préférences utilisateur et les flux de reconnexion.
9. Mettre des tests de parité. Il faut comparer ancien et nouveau résultat sur quelques écrans critiques : mêmes chiffres, mêmes filtres, mêmes règles de tri, mêmes agrégats.
10. Mesurer la migration comme un produit. Temps de réponse, erreurs API, abandon utilisateur, usages des écrans, dette restante. Sans métriques, la migration devient une suite d’impressions.

## Etape critique 2 detaillee - gel du coeur metier et matrice de parite

Cette section transforme l'etape 2 en livrable exploitable. Le but n'est pas d'ajouter encore une couche de planification abstraite. Le but est de figer ce qui constitue aujourd'hui la verite fonctionnelle de LevelUp avant de toucher au front React.

## Objectif operationnel

- figer les surfaces de reference reellement en service ou deja cibles par V7
- figer les contrats de donnees, de navigation, de filtres, d'auth et de cache qui conditionnent les chiffres affiches
- distinguer ce qui doit etre migre a parite stricte, ce qui peut etre recompose, et ce qui ne doit pas etre migre en 1:1
- deriver un backlog de vertical slices priorise a partir d'un inventaire ecran par ecran au lieu d'un backlog technique generique

## Methode retenue

1. Inventorier les shells de reference, pas seulement les pages isolees.
2. Identifier les surfaces hors navigation principale mais critiques pour le produit.
3. Lister pour chaque ecran : objectif metier, sources de donnees, calculs/aggregations, etat UI, navigation, dependances implicites a Streamlit.
4. Distinguer la parite fonctionnelle stricte du simple relookage UI.
5. Transformer le tout en backlog de slices livre par parcours metier.

## Shells de reference a figer

- Shell de production courant : streamlit_app.py + src/app/page_router.py.
  - Source de verite runtime pour les pages PAGE_KEYS : timeseries, session_compare, teammates, last_match, media, citations, explorer, match_history, career, settings.
- Shell cible produit V7 : streamlit_app_v7.py + src/ui/pages/v7_sections.py.
  - Source de verite cible pour les sections home, stats, squad, synthesis, explorer, media, profile, settings.
- Surfaces hors navigation mais critiques :
  - setup_wizard.py
  - setup_smoke_test.py
  - match_view.py
  - objective_analysis.py
- Surfaces legacy a absorber plutot qu'a migrer telles quelles :
  - win_loss.py : deja redistribue entre timeseries et synthesis
  - media_tab.py / media_library.py : remplaces par media_v2.py

## Invariants transverses a geler avant toute migration React

### 1. Contexte applicatif commun

La quasi-totalite des pages consomment un contexte commun calcule dans streamlit_app.py :

- df : historique complet charge une fois
- dff : historique filtre courant
- base : clone avant filtres
- db_path, xuid, waypoint_player, me_name, db_key, aliases_key
- base_s_ui : DataFrame de sessions servant de source de verite pour plusieurs pages
- match_view_params : callbacks et loaders injectes dans Last Match, Explorer et Match View

Tant que le front React n'a pas remplace ce contexte par des contrats API explicites, il ne faut pas changer la semantique de ce contexte cote Python.

### 2. Modele de filtres et de sessions

Le moteur de filtres doit etre considere comme un invariant fonctionnel, pas comme un detail de rendu Streamlit.

- Mode de filtre : Periode ou Sessions
- Filtres temporels : start_date_cal, end_date_cal
- Scope sessions : picked_session_label, picked_solo_session_label, picked_squad_session_label, picked_sessions
- Filtres cascade : experience types, playlists, modes, maps
- Modele de sessions : GAP_MINUTES_FIXED = 120 dans src/app/filters_render.py
- Mecanisme de shadow keys pour survivre aux reruns et a st.navigation

La migration React devra reproduire la meme semantique de filtres et de regroupements, meme si l'implementation UI change completement.

### 3. Contrat de deep links et navigation interne

Les query params deja supportes doivent etre traites comme contrat de navigation minimal a conserver :

- page
- match_id
- gamertag
- player
- stats_view
- session
- scope

Les etats de navigation implicites a sortir de Streamlit sont notamment :

- _pending_page
- _pending_match_id
- _pending_gamertag
- _pending_player
- match_id_input
- v7_current_section
- v7_stats_view
- v7_profile_view
- _last_match_nav_index / _last_match_nav_total / _last_match_nav_session_key

### 4. Identite, auth et bootstrap

Le demarrage applicatif depend aujourd'hui de plusieurs verites qu'il ne faut pas laisser diverger pendant la migration :

- browser prefs restaurees avant le reste de l'application
- setup_wizard bloque l'app si la configuration est incomplete
- auth Halo/Xbox via provider.py et Device Code Flow
- selection joueur via db_path + xuid_input + waypoint_player
- app_settings persistees sur disque et relues dans la session

### 5. Caches et services d'arriere-plan

La migration UI ne doit pas casser la semantique des caches/process workers existants :

- background_media_indexing
- reindex_media_after_sync
- process cache de la home V7 pour battle pass / challenges
- caches de repository et de sessions
- Tailscale funnel optionnel

### 6. I18n et labels metier

Le bilinguisme FR/EN et les labels metier doivent etre consideres comme invariants de parite, pas comme details cosmetiques.

- labels pages
- labels modes/cartes/playlists
- badges outcome / domination / humiliation
- rangs carriere / CSR / LUSR
- textes settings / wizard / media / citations

## Inventaire exhaustif des surfaces a documenter

### Surfaces de production directement navigables

- Timeseries
- Session Compare
- Teammates
- Last Match
- Explorer
- Match History
- Career
- Citations
- Media V2
- Settings

### Surfaces V7 deja codees et a prendre comme reference cible

- Home Mission Control
- Synthesis
- Sections V7 stats/profile qui recomposent des pages legacy existantes

### Surfaces hors navigation principale mais critiques

- Match View
- Setup Wizard
- Smoke Test post-installation
- Objective Analysis

### Surfaces a ne pas migrer en 1:1

- Win/Loss comme page autonome
- Media tab / media library legacy
- anciens labels de page legacy servant seulement aux redirects

## Fiches ecran - ce qui doit etre gele page par page

### Setup Wizard + Smoke Test

- Reference Streamlit : src/ui/pages/setup_wizard.py, src/ui/pages/setup_smoke_test.py, streamlit_app.py avant tout dispatch principal.
- Role metier : bloquer l'app si la configuration est incomplete, permettre le choix du mode d'auth, provisionner le joueur, lancer une sync initiale, faire un smoke test, puis seulement ouvrir le produit.
- Sources de donnees / logique : setup_wizard_logic.py, setup_wizard_xbox.py, auth provider, creation du player profile, smoke_test_logic.py.
- Etat Streamlit a externaliser : _setup_mode, _xbox_oauth_result, _smoke_gamertag, _smoke_db_path, _setup_smoke_completed, _smoke_test_done, _smoke_test_result.
- Regle de parite : les memes gates bloquent l'acces a l'app, les memes etapes sont franchies, les memes checks d'integrite passent ou avertissent.
- Strategie React : flow dedie d'onboarding avec session backend + endpoint de job status pour sync/backfill/smoke test.
- Priorite : P0 absolu.

### Settings

- Reference Streamlit : src/ui/pages/settings.py.
- Role metier : piloter la langue, timezone, options de backfill, options d'affichage, media watcher/index, notifications Discord, reset index media.
- Sources de donnees / logique : AppSettings, patch_settings, _write_settings, MediaIndexer.reset_media_tables, browser storage pour lang/show_hints.
- Etat Streamlit a externaliser : app_settings, setting_*, _hints_visible, preferences navigateur associees.
- Regle de parite : toute option modifiee doit produire le meme effet serveur qu'aujourd'hui, sans dependre d'un rerun Streamlit.
- Strategie React : formulaires typed + endpoints PATCH settings + actions explicites pour rescan/reset media.
- Priorite : P0, car bootstrap et exploitation.

### Career

- Reference Streamlit : src/ui/pages/career.py, career_data.py, career_logic.py, career_lusr.py, career_top_matches_render.py, career_encounters_render.py.
- Role metier : progression carrière, jauges XP, progression Hero, historique XP, LUSR, top matches, encounters.
- Sources de donnees / logique : career_progression, mv_player_matches, rang metadata, projections XP, logique LUSR, top matches data.
- Etat Streamlit a externaliser : quasi nul hors contexte global; attention a l'usage de app_settings dans certains sous-rendus.
- Regle de parite : meme rang courant, meme XP total, meme progression Hero, memes projections, meme LUSR, meme top matches, memes encounters.
- Strategie React : tres bon premier slice; reexposer les KPIs et figures en API sans changer les calculs Python.
- Priorite : P1.

### Match History

- Reference Streamlit : src/ui/pages/match_history.py, match_table_html.py.
- Role metier : table complete des matchs filtres, enrichie avec score, map/mode/playlist, win rate historique, performance relative, CSV export.
- Sources de donnees / logique : dff + df_full, compute_performance_series, traductions, vectorisation Polars.
- Etat Streamlit a externaliser : aucun state local critique hors filtres globaux; export CSV a recabler en HTTP.
- Regle de parite : meme nombre de lignes, meme ordre, memes valeurs calculees, meme tri/filtrage semantique, meme export.
- Strategie React : endpoint pagine/raw data + grille React riche; tres bon candidat MVP.
- Priorite : P1.

### Explorer

- Reference Streamlit : src/ui/pages/explorer.py, explorer_logic.py, explorer_data.py, explorer_results.py.
- Role metier : rechercher un joueur ou un match, filtrer des rencontres via cascade, ouvrir un detail de match, supporter les deep links.
- Sources de donnees / logique : get_all_gamertags, resolve_gamertag_to_xuid, fuzzy_search_gamertags, classify_experience_type, load_is_with_friends.
- Etat Streamlit a externaliser : _pending_gamertag, _pending_match_id, match_id_input, _explorer_selected_match, pagination locale des tables de resultats.
- Regle de parite : memes resultats de recherche floue, memes filtres cascade, memes match_ids cibles, meme comportement de deep link.
- Strategie React : route URL-first + endpoints search / lookup / filtered results; tres bon slice apres Match History.
- Priorite : P1.

### Match View

- Reference Streamlit : src/ui/pages/match_view.py + famille match_view_*.py.
- Role metier : detail complet d'un match, avec header score/rang/carte, onglets Resume/Combat/Equipe/Medias/Citations.
- Sources de donnees / logique : match_view_logic.py, cached loaders injectes via MatchViewParams, match_view_tabs.py, match_view_weapon_kills.py, scoreboards, nemesis, timeline.
- Etat Streamlit a externaliser : match_id de route, callbacks injectes aujourd'hui via match_view_params, quelques flags de debug et de navigation venant des pages parentes.
- Regle de parite : meme score, meme roster, memes medailles, memes events, memes armes, meme rang, memes citations.
- Strategie React : route /matches/:id, payload detaille compose au backend, figures JSON quand utile, pas de reimplementation metier dans le front.
- Priorite : P1.

### Last Match

- Reference Streamlit : src/ui/pages/last_match.py.
- Role metier : wrapper de Match View sur le dernier match du scope courant, avec navigation precedent/suivant.
- Sources de donnees / logique : dff filtre courant + _resolve_nav_index().
- Etat Streamlit a externaliser : _last_match_nav_index, _last_match_nav_total, _last_match_nav_session_key.
- Regle de parite : meme match courant selon le scope filtre, meme logique de reset quand les filtres changent, meme navigation prev/next.
- Strategie React : ne pas en faire une API distincte; le considerer comme une vue derivee de Match View + liste filtree.
- Priorite : P1.5, avec Match View.

### Citations

- Reference Streamlit : src/ui/pages/citations.py.
- Role metier : afficher les commendations H5G et les medailles Halo Infinite sur le scope filtre.
- Sources de donnees / logique : CitationEngine, match_citations, top_medals_fn, referentiels medailles, distribution Plotly.
- Etat Streamlit a externaliser : peu de state local; dependance surtout au scope filtre et aux caches du moteur de citations.
- Regle de parite : memes totaux de citations, memes medailles, memes deltas filtre vs complet, meme grille et meme distribution.
- Strategie React : endpoint agregat brut + eventuellement figure Plotly JSON pour la distribution.
- Priorite : P2.

### Media V2

- Reference Streamlit : src/ui/pages/media_v2.py, media_v2_filters.py, media_v2_grid.py, media_v2_likes.py.
- Role metier : bibliotheque locale de captures, groupement par auteur/date/carte/mode/session/experience/liked, lightbox, likes persistants, renvoi vers match.
- Sources de donnees / logique : MediaIndexer.load_media_for_ui, table d'enrichissement media, likes navigateur, jointure avec df_full.
- Etat Streamlit a externaliser : _lb_state, mv2_autoplay, _pending_page, _pending_match_id, etats de filtres, persistence likes.
- Regle de parite : memes medias, meme regroupement, memes filtres, meme navigation vers le match, meme persistence des likes.
- Strategie React : endpoints raw media + URLs/paths de thumbnails + persistence locale navigateur; forte refonte UI autorisee, logique metier stricte.
- Priorite : P2.

### Home Mission Control

- Reference Streamlit : src/ui/pages/home_mission_control.py, home_mission_control_logic.py, home_mission_control_api.py, battlepass/challenges modules.
- Role metier : accueil V7 compose de hero, highlights, KPIs, quick actions, resume sessions, activite recente, battle pass, defis, dernier match, medias recents.
- Sources de donnees / logique : summaries de sessions, recent matches/media, live APIs battlepass/challenges avec cache process-level, embed de Last Match.
- Etat Streamlit a externaliser : v7_current_section, _v7_pages, stats_view/session/scope/match_id en deep link, etat du navigateur battle pass, prefetch home progressions.
- Regle de parite : memes highlights, meme ordre des matchs recents, memes KPIs, meme contenu battle pass/defis, meme CTA contextuels.
- Strategie React : route composee qui agrege plusieurs endpoints; a traiter une fois Career, Match View et Media deja exposes.
- Priorite : P2.

### Timeseries

- Reference Streamlit : src/ui/pages/timeseries.py + modules _timeseries_* + win_loss helpers reutilises.
- Role metier : lecture analytique temporelle du joueur via KPIs, KDA, cumul, forme recente, intensite, distributions, corrélations, weapon kills, personal scores.
- Sources de donnees / logique : TimeseriesService, win_loss helpers, nombreux modules Plotly, downsample_for_plot.
- Etat Streamlit a externaliser : depend surtout des filtres globaux; lecture locale de filter_mode et de quelques aides UI.
- Regle de parite : memes series, memes agrégats, memes seuils de downsampling, memes annotations et memes onglets logiques.
- Strategie React : conserver les calculs et figures cote Python au debut; livrer du Plotly JSON plutot que reecrire tous les graphes.
- Priorite : P3.

### Session Compare

- Reference Streamlit : src/ui/pages/session_compare.py, session_compare_logic.py, session_compare_charts.py, _session_compare_*.
- Role metier : comparer deux sessions et les replacer dans un contexte historique similaire.
- Sources de donnees / logique : cached_compute_sessions_db, friends mapping, build_skill_series, compute_historical_context, similar sessions logic.
- Etat Streamlit a externaliser : compare_session_a, compare_session_b, _last_picked_for_compare, picked_session_label, alias/friends fallback.
- Regle de parite : memes sessions candidates, meme choix par defaut, meme session precedente comparable, memes deltas et comparaisons.
- Strategie React : apres avoir sorti proprement le modele de sessions et les filtres URL-first.
- Priorite : P3.

### Teammates

- Reference Streamlit : src/ui/pages/teammates.py + sous-modules teammates_*.
- Role metier : analyser les performances avec 1 a 3 coequipiers, synergies, impact, intensite, armes, radars et vues duo/trio.
- Sources de donnees / logique : TeammatesService, shared + eventuelles DB joueurs, enrichissement perfect kills, build_teammates_opts_map, base_s_ui.
- Etat Streamlit a externaliser : teammates_picked_labels, _cache_warning_shown, scope de sessions solo/escouade, show_records.
- Regle de parite : memes coequipiers proposes, memes stats par duo/trio, memes radars, memes enrichissements par armes et impact.
- Strategie React : slice tardif et lourd; necessite des contrats API specifiques et une clarification definitive du modele multi-joueur.
- Priorite : P3 critique en complexite.

### Synthesis

- Reference Streamlit : src/ui/pages/synthesis.py.
- Role metier : vue d'ensemble solo vs escouade, periode locale, synthese strategique en reemploi de briques analytiques existantes.
- Sources de donnees / logique : KPIStats, load_is_with_friends, win_loss helpers, comparaison solo/squad.
- Etat Streamlit a externaliser : synthesis_period.
- Regle de parite : memes fenetres temporelles, meme decoupage solo/escouade, memes agrégats et memes chartes de comparaison.
- Strategie React : bon ecran de consolidation V7 une fois le shell web stabilise.
- Priorite : P3.

### Objective Analysis

- Reference Streamlit : src/ui/pages/objective_analysis.py.
- Role metier : valoriser les awards objectifs, le profil support/slayer, les trends objectifs et correlations avec les kills.
- Sources de donnees / logique : personal_score_awards, objective_participation, objective_charts, mv_player_matches.
- Etat Streamlit a externaliser : wrapper from_session_state et dependances implicites au joueur courant.
- Regle de parite : memes points objectifs, meme ratio, meme classification support/polyvalent/slayer, memes breakdowns et trends.
- Strategie React : feature annexe ou future page d'analyse; ne pas prioriser avant les parcours centraux.
- Priorite : P4.

## Matrice de parite ecran par ecran

| Surface React cible | Reference Streamlit | Invariants a preserver | Etat / navigation a sortir de Streamlit | Type d'API initial recommande | Inclusion |
| --- | --- | --- | --- | --- | --- |
| Setup / Onboarding | setup_wizard.py + setup_smoke_test.py | gating setup, auth modes, player provisioning, smoke test | _setup_mode, _xbox_oauth_result, _smoke_* | commandes + job status + session auth | MVP P0 |
| Settings | settings.py | ecriture app_settings, hints, media, Discord, backfill | app_settings, setting_*, browser prefs | CRUD settings + actions serveur | MVP P0 |
| Career | career.py + career_* | XP, rang, Hero, LUSR, top matches, encounters | presque rien hors contexte global | mix data brute + Plotly JSON | MVP P1 |
| Match History | match_history.py | lignes, tri, score, win rate hist, perf relative, CSV | aucun state local fort | data brute paginee/exportable | MVP P1 |
| Explorer | explorer.py + explorer_* | fuzzy search, cascade, deep links, resultats | _pending_*, match_id_input, pagination locale | data brute + search endpoints | MVP P1 |
| Match View | match_view.py + match_view_* | scoreboards, tabs, rang, armes, citations, medias | route match_id + params parents | payload detaille compose | MVP P1 |
| Last Match | last_match.py | selection du dernier match du scope + prev/next | _last_match_nav_* | derive du scope filtre + match detail | MVP P1.5 |
| Citations | citations.py | commendations, top medals, distribution, grille | peu d'etat local | agregats bruts + figure JSON | Post-MVP P2 |
| Media V2 | media_v2.py + media_v2_* | index local, filtres, groupes, lightbox, likes | _lb_state, mv2_autoplay, _pending_match_id | data brute media + thumbs + prefs locales | Post-MVP P2 |
| Home Mission Control | home_mission_control.py + logic/api | highlights, actions, battle pass, defis, dernier match, medias | v7 section state, battle pass focus, deep links stats/scope | agregat multi-endpoints | Post-MVP P2 |
| Timeseries | timeseries.py + _timeseries_* | series, cumul, EWMA, intensite, distributions | depend surtout du filtre global | Plotly JSON + quelques KPIs bruts | Post-MVP P3 |
| Session Compare | session_compare.py + logic/charts | selection A/B, historique comparable, deltas | compare_session_a/b, _last_picked_for_compare | mix data brute + Plotly JSON | Post-MVP P3 |
| Teammates | teammates.py + teammates_* | selection mates, synergies, impact, armes, multi-DB/shared | teammates_picked_labels, scope sessions | APIs specifiques teammates | Post-MVP P3 |
| Synthesis | synthesis.py | comparatif solo/escouade, fenetre periode | synthesis_period | data brute + figures | Post-MVP P3 |
| Objective Analysis | objective_analysis.py | awards objectifs, ratio, profil, trends | wrapper joueur courant | data brute + figures | Post-MVP P4 |
| Win/Loss autonome | win_loss.py | absorbe dans Timeseries/Synthesis, pas de migration 1:1 | n/a | pas d'ecran dedie | Ne pas migrer |
| Media legacy | media_tab.py / media_library.py | absorbe par Media V2 | n/a | pas d'ecran dedie | Ne pas migrer |

## Backlog de vertical slices priorise

### Slice 0 - Fondations transverses

- shell React URL-first
- gestion du joueur courant, de la langue et de la session backend
- compat deep links page/match_id/gamertag/player/session/scope
- convention de schemas FastAPI + erreurs + cache HTTP
- capture des invariants de filtres avant tout ecran metier

### Slice 1 - Setup / Auth / Settings

- reproduire le gating de setup_wizard
- sortir un onboarding React exploitable avec etat de job pour smoke test
- exposer les settings critiques (lang, auth, media, Discord, backfill)

### Slice 2 - Career

- premier ecran demonstratif a forte valeur visible
- faible couplage au rerun Streamlit
- bon candidat pour valider schemas, i18n et livraisons Plotly JSON

### Slice 3 - Match History

- premier parcours data-table riche
- valide les filtres globaux, la pagination serveur et l'export
- sert de socle a Explorer et Last Match

### Slice 4 - Explorer

- valide la recherche, les deep links, les filtres cascade et la navigation vers un match
- force a stabiliser les APIs de recherche et de lookup joueur/match

### Slice 5 - Match View + Last Match

- rend le detail de match reutilisable par Explorer, Home et future navigation React
- reprend ensuite Last Match comme vue derivee du scope filtre

### Slice 6 - Citations

- bon ecran intermediaire pour roder les agrégats et la double lecture filtered vs full
- complexite raisonnable si les loaders backend sont deja exposes

### Slice 7 - Media V2

- traite un parcours a forte valeur UX et un des plus anti-Streamlit
- a lancer seulement quand auth, routes et navigation vers Match View sont solides

### Slice 8 - Home Mission Control

- a brancher apres Career, Match View et Media pour eviter un ecran compose branche sur des morceaux non migres
- excellent ecran de convergence pour la cible V7

### Slice 9 - Timeseries

- gros bloc analytique, mais tres favorable a une strategie Plotly JSON backend-first
- a faire apres validation de la couche graphs React

### Slice 10 - Session Compare

- depend fortement d'un modele de sessions explicite et stable
- doit venir apres la clarification definitive du state model React

### Slice 11 - Teammates

- ecran le plus dense cote coequipiers, multi-vues, multi-sources et enrichissements
- a traiter une fois les patterns de charts, tables et navigation detail sont rodes

### Slice 12 - Synthesis + Objective Analysis

- ecrans annexes mais utiles pour consolider la V7
- a migrer apres les parcours coeur et avant la decommission finale si leur usage le justifie

### Slice 13 - Decommission progressive Streamlit UI

- bascules de routes ecran par ecran
- redirects des anciens labels/URLs
- retrait des pages legacy absorbees
- nettoyage des dependances Streamlit devenues mortes

## Definition of done pour cette etape 2

L'etape critique 2 n'est consideree comme terminee que si :

- chaque surface de reference a une fiche metier explicite
- chaque ecran a un statut clair : MVP, post-MVP, absorbe, ou non migre
- chaque ecran a un type d'API cible decide : data brute, Plotly JSON, ou mixte
- chaque etat implicite Streamlit a une contrepartie cible decidee : URL, store front, session backend ou cache HTTP
- le backlog est ordonne par parcours metier et non par couche technique

## Jeu minimal de tests de parite a preparer ensuite

- Career : rang, XP total, progression Hero, LUSR, top matches
- Match History / Explorer : cardinalite, ordre des lignes, tri, filtres, match_id cibles
- Match View / Last Match : score, roster, tabs, armes, rang, citations, navigation prev/next
- Media V2 : cardinalite, groupements, likes, navigation vers match, lightbox
- Home : ordre des matchs recents, resumes sessions, battle pass, defis
- Timeseries / Session Compare / Teammates : memes series, memes selections, memes agrégats
- Setup / Settings : memes gates, memes side effects de configuration, meme resultat smoke test

## Conclusion de cadrage

Le vrai gel du coeur metier n'est pas un freeze total du code Python. C'est un freeze des comportements attendus ecran par ecran et des contrats invisibles qui relient aujourd'hui Streamlit, les filtres, les caches, l'auth et DuckDB.

Tant que cette matrice reste la reference, la migration FastAPI + React peut avancer sans transformer chaque ecran en projet de reinterpretation produit.

## Etape critique 3 detaillee - extraction du contrat d'API

Cette section transforme l'etape 3 en livrable de pilotage. Le but n'est pas de lister des endpoints pour le principe. Le but est de figer la frontiere entre le backend Python et la future UI React afin d'eliminer les dependances implicites a Streamlit avant de lancer l'implementation.

## Objectif operationnel

- definir une interface stable entre UI et backend pour chaque parcours MVP
- rendre explicites les formats de donnees aujourd'hui caches dans le contexte Streamlit, les query params et le session_state
- figer les conventions communes de pagination, tri, filtres, erreurs, jobs asynchrones et figures Plotly
- permettre au front React d'orchestrer les pages sans reimplementer la logique metier Python

## Decisions de contrat a figer avant implementation

### 1. Frontiere stricte backend/UI

- aucun composant React ne lit directement DuckDB, un fichier local ou une structure Python interne
- aucun endpoint n'expose de chemin de base, de nom de table, de details de repository ou de semantique liee au worktree
- toute valeur metier affichee dans le front doit venir soit d'un schema de donnees explicite, soit d'une figure Plotly serialisee

### 2. Style des contrats

- base path unique : /api/v1
- payloads JSON en snake_case pour rester alignes avec Pydantic et reduire la couche d'adaptation initiale
- dates et datetimes en ISO-8601 ; les datetimes doivent etre normalises en UTC ou explicitement etiquetes
- identite joueur portee par player_slug dans les routes metier, pas par un etat front implicite

### 3. Familles d'endpoints autorisees

- endpoints transverses : bootstrap, session, auth, setup, settings, jobs, resolution des filtres
- endpoints page-oriented : payload complet ou semi-agrege par ecran pour limiter l'orchestration front dans les slices MVP
- sous-resources specialisées seulement quand la pagination serveur, le lazy loading ou la frequence de rafraichissement l'imposent vraiment

### 4. Contrat d'erreur et d'observabilite

- toute reponse en erreur doit retourner un code stable, un message lisible, un request_id et un indicateur retryable
- les erreurs de validation doivent rester structurées via field_errors plutot qu'un message libre inutilisable cote front
- les ecrans React doivent pouvoir distinguer sans heuristique : erreur fonctionnelle, erreur de validation, absence de donnees, job en cours, authentification requise

### 5. Contrat de filtres, tri et pagination

- FilterContextInput devient la forme canonique de tous les filtres globaux du shell
- les pages qui partagent la meme semantique de scope ne redefinissent pas leurs propres variantes de filtres
- PaginationRequest et PaginatedResponse deviennent le contrat unique des tables MVP, avec tri explicite et page indexee a partir de 1
- les compteurs et options resolues doivent etre renvoyes par le backend, pas recalcules dans Zustand

### 6. Contrat de graphes et contenus riches

- les graphes conserves cote Python sont exposes via un payload Plotly JSON stable : figure, config_key, revision_key
- une figure n'embarque jamais de logique applicative implicite necessaire au reste de la page
- les cartes, badges, tableaux et listes qui n'ont pas besoin de Plotly doivent sortir en donnees brutes deja calculees

### 7. Contrat d'auth et de session

- pour le MVP, l'API reste seule proprietaire des tokens Halo, du cache MSAL et des secrets
- le front ne manipule jamais directement spartan_token, clearance, refresh token ou contenu du cache MSAL
- la cible a figer est une session backend opaque portee par cookie httpOnly ou equivalent serveur, avec endpoints explicites pour bootstrap, statut d'auth et progression du Device Code Flow

## Livrables attendus de l'etape 3

- un socle de schemas transverses partages entre les pages MVP
- une cartographie route React -> endpoint API -> module Python source de verite
- un contrat explicite pour les erreurs, les jobs longs, la pagination, les filtres et les figures
- un lot prioritaire de payloads page-oriented suffisamment stables pour lancer Slice 0 a Slice 5 sans renegocier l'API a chaque ecran

## Ce que l'etape 3 ne doit pas faire

- ne pas transformer l'API en miroir technique des repositories DuckDB
- ne pas pousser de logique metier critique dans le front sous pretexte de gagner du temps
- ne pas figer trop tot des micro-endpoints ultra-fins qui compliquent le MVP sans valeur produit immediate
- ne pas coupler les contrats FastAPI a des details du rendu Streamlit ou a des noms de widgets historiques

## Definition of done pour cette etape 3

L'etape critique 3 est consideree comme couverte si :

- chaque parcours MVP a au moins un endpoint cible, un schema d'entree et un schema de sortie explicites
- les conventions transverses de filtres, erreurs, pagination, figures et jobs sont unifiees
- l'auth et la session ont une frontiere serveur claire, sans exposition des tokens au navigateur
- les routes React prioritaires peuvent etre branchees sans acceder a PageContext, st.session_state ou aux query params Streamlit historiques
- les sections detaillees ci-dessous suffisent a lancer l'implementation de Slice 0 a Slice 5 sans rediscuter la forme de l'API

## Contrats API MVP detailles - lot prioritaire

Cette section derive les contrats API concrets a partir de la matrice de parite. Le principe retenu est volontairement pragmatique : on ne cherche pas une API ultra-theorique. On cherche une API qui permette de livrer les premiers ecrans React sans reimplementer la logique Python dans le navigateur.

## Principes de conception a figer avant implementation

### 1. Versioning et base path

- base path initial : /api/v1
- aucune route front React ne doit parler directement a DuckDB ni connaitre un chemin de DB
- les routes React sont URL-first, les endpoints sont player-scoped quand ils dependent du joueur courant

### 2. Deux familles d'endpoints

- endpoints transverses : bootstrap, auth, setup, settings, jobs, resolution des filtres
- endpoints page-oriented : payloads agreges par ecran pour minimiser l'orchestration front au debut

### 3. Regle d'aggregation

Pour le MVP, on privilegie des endpoints de page relativement gras quand :

- la page existe deja cote Python comme composition stabilisee
- la page reunit plusieurs sous-sections a valeur metier unique
- le risque de reinterpreter les calculs dans le front est eleve

On privilegie des sous-resources plus fines quand :

- une table doit etre paginee/triable cote serveur
- une partie est relancee plus souvent qu'une autre
- une sous-section est lourde et peut etre chargee a la demande

### 4. Regle de source de verite

- les calculs metier restent cote Python
- le front consomme soit de la data brute pre-calculee, soit du Plotly JSON
- aucune derivation metier critique ne doit vivre seulement dans Zustand ou dans un composant React

### 5. Contrat de contexte commun

Tout endpoint de page MVP doit pouvoir reconstruire l'equivalent des composantes aujourd'hui derivees dans PageContext :

- player context
- filter context effectif
- freshness / provenance des donnees
- data principale
- actions ou liens associes si necessaire

## Schemas transverses a partager entre endpoints

### ApiMeta

- request_id : str
- generated_at : datetime ISO-8601
- locale : fr | en
- app_version : str
- data_version : optionnel, pour invalider les caches front si les contrats changent

### ApiError

- code : str
- message : str
- details : dict | list | None
- retryable : bool
- field_errors : list[FieldError] | None

### PlayerSummary

- player_slug : str
- gamertag : str
- xuid : str
- waypoint_player : str
- is_demo : bool

### FilterContextInput

- filter_mode : period | sessions
- period : { start_date: date | null, end_date: date | null }
- sessions :
  - picked_session_label : str | null
  - picked_solo_session_label : str | null
  - picked_squad_session_label : str | null
  - picked_sessions : list[str]
  - gap_minutes : int
- cascade :
  - experience_types : list[str]
  - playlists : list[str]
  - modes : list[str]
  - maps : list[str]

### FilterContextResolved

- effective : FilterContextInput normalise
- available_options :
  - experience_types : list[LabelValue]
  - playlists : list[LabelValue]
  - modes : list[LabelValue]
  - maps : list[LabelValue]
- session_options :
  - all_labels : list[str]
  - solo_labels : list[str]
  - squad_labels : list[str]
- counts :
  - total_matches_before_filters : int
  - total_matches_after_filters : int

### PaginationRequest

- page : int >= 1
- page_size : int
- sort : list[SortSpec]

### PaginatedResponse[T]

- items : list[T]
- total : int
- page : int
- page_size : int
- total_pages : int

### PlotlyFigurePayload

- figure : dict
- config_key : clean | static
- revision_key : str | None

### AsyncJobStatus

- job_id : str
- job_type : setup_smoke_test | sync | backfill | reindex_media | other
- status : queued | running | succeeded | failed | cancelled
- progress_pct : int | None
- current_step : str | None
- started_at : datetime | None
- finished_at : datetime | None
- result : dict | None
- error : ApiError | None

### PageEnvelope[T]

- meta : ApiMeta
- player : PlayerSummary
- filters : FilterContextInput | None
- freshness :
  - source : live | cached | mixed
  - sync_status : fresh | stale | unknown
  - warnings : list[str]
- data : T

## Contrat transverse indispensable - bootstrap et resolution des filtres

Avant meme les pages metier, le shell web a besoin de deux contrats stables.

### Bootstrap shell

Route front : /

Endpoints recommandes :

| Methode | Path | Usage | Reponse principale |
| --- | --- | --- | --- |
| GET | /api/v1/bootstrap | Charger shell, joueur courant, setup status, feature flags | BootstrapResponse |
| GET | /api/v1/players | Lister les profils joueurs disponibles | PlayersListResponse |
| POST | /api/v1/session/context | Persister joueur courant et langue de session si on choisit une persistance serveur | SessionContextResponse |

Schema BootstrapResponse :

- setup_required : bool
- auth_state : missing | partial | ready
- current_player : PlayerSummary | null
- available_players : list[PlayerSummary]
- locale : fr | en
- hints_visible_default : bool
- feature_flags :
  - v7_enabled
  - media_enabled
  - demo_mode
  - discord_configured
  - tailscale_enabled
- settings_excerpt :
  - lang
  - user_timezone
  - show_records
  - normalize_mode_labels

### Resolution des filtres globaux

Route front : shell filter bar / drawers / page stores

Endpoint recommande :

| Methode | Path | Usage | Requete | Reponse |
| --- | --- | --- | --- | --- |
| POST | /api/v1/players/{player_slug}/filters/resolve | Recalculer sessions, options cascade et compteurs | FilterContextInput | FilterContextResolved |

Ce contrat est critique car il remplace le comportement implicite aujourd'hui distribue entre streamlit_app.py, filters_render.py, filter_state.py et session_state.

## Contrats detailles - Setup / Auth / Settings

### Routes front cibles

- /setup
- /settings

### Endpoints recommandes

| Methode | Path | Usage | Requete | Reponse | Source Python |
| --- | --- | --- | --- | --- | --- |
| GET | /api/v1/setup/status | Savoir si l'app doit bloquer sur le wizard | - | SetupStatusResponse | setup_wizard_logic.py |
| POST | /api/v1/auth/device-flow/start | Demarrer le Device Code Flow Xbox | DeviceFlowStartRequest | DeviceFlowStartResponse | auth/provider.py, setup_wizard_xbox.py |
| GET | /api/v1/auth/device-flow/{attempt_id} | Poll du statut d'un flow Device Code | - | DeviceFlowStatusResponse | auth/provider.py |
| POST | /api/v1/setup/players | Creer/provisionner un profil joueur | CreatePlayerProfileRequest | CreatePlayerProfileResponse | setup_wizard_logic.py, player_provisioning.py |
| POST | /api/v1/setup/smoke-test | Lancer sync + backfill + verifications d'integrite | SmokeTestStartRequest | AsyncJobStatus | setup_smoke_test_logic.py |
| GET | /api/v1/jobs/{job_id} | Suivre un job long | - | AsyncJobStatus | infrastructure jobs |
| GET | /api/v1/settings | Charger toute la configuration editable | - | SettingsResponse | AppSettings, load_settings |
| PATCH | /api/v1/settings | Mettre a jour la configuration editable | UpdateSettingsRequest | SettingsResponse | patch_settings, _write_settings |
| POST | /api/v1/settings/media/reset-index | Reset des tables media puis reindex eventuel | MediaResetRequest | AsyncJobStatus | MediaIndexer.reset_media_tables |

### Schemas de reponse a figer

SetupStatusResponse :

- needs_setup : bool
- auth :
  - has_client_id : bool
  - has_refresh_token : bool
  - has_msal_cache : bool
  - preferred_method : refresh_token | device_code | unknown
- player :
  - has_any_profile : bool
  - default_player_slug : str | null
- next_blocking_step : choose_mode | auth | player | smoke_test | done

DeviceFlowStartResponse :

- attempt_id : str
- user_code : str
- verification_uri : str
- verification_uri_complete : str | null
- expires_in_seconds : int
- poll_interval_seconds : int

DeviceFlowStatusResponse :

- attempt_id : str
- status : pending | authorized | provisioned | failed | expired
- gamertag : str | null
- xuid : str | null
- error : ApiError | null

CreatePlayerProfileRequest :

- gamertag : str
- xuid : str | null
- profile_mode : xbox | azure_manual

CreatePlayerProfileResponse :

- player : PlayerSummary
- db_created : bool
- warnings : list[str]

SmokeTestStartRequest :

- player_slug : str
- max_matches : int
- run_backfill : bool

SettingsResponse / UpdateSettingsRequest :

- lang
- user_timezone
- normalize_mode_labels
- show_records
- refresh_clears_caches
- career_top_exclude_btb
- media_captures_base_dir
- media_tolerance_minutes
- media_watcher_enabled
- media_watcher_debounce_seconds
- discord_notifications_enabled
- discord_webhook_url_present : bool
- discord_lang
- discord_notify_sync
- discord_notify_backfill
- discord_notify_new_version
- discord_notify_new_media
- spnkr_refresh_with_backfill
- spnkr_refresh_backfill_medals
- spnkr_refresh_backfill_skill
- spnkr_refresh_backfill_aliases
- spnkr_refresh_backfill_personal_scores
- spnkr_refresh_backfill_performance_scores
- spnkr_refresh_backfill_lusr
- spnkr_refresh_backfill_events
- spnkr_refresh_backfill_weapons

### Stores front et query keys recommandes

- useAppShellStore
  - currentPlayerSlug
  - locale
  - setupRequired
  - featureFlags
- useSetupFlowStore
  - selectedMode
  - currentAttemptId
  - currentJobId
- useSettingsDraftStore
  - dirtyFields
  - lastSavedAt
  - localUiPrefs : showHints, lastPlayerSlug
- Query keys
  - ['bootstrap']
  - ['players']
  - ['setup-status']
  - ['device-flow', attemptId]
  - ['job', jobId]
  - ['settings']

### Criteres de recette

- un utilisateur non configure voit uniquement le flow setup tant que next_blocking_step != done
- le Device Code Flow produit le meme resultat fonctionnel que le wizard Streamlit
- la creation de profil cree le meme slug joueur et la meme base stats.duckdb qu'aujourd'hui
- le smoke test expose les memes phases, les memes warnings et le meme resultat final
- chaque changement de settings persiste reellement, sans perte au refresh du navigateur

## Contrats detailles - Career

### Route front cible

- /players/:playerSlug/career

### Endpoints recommandes

| Methode | Path | Usage | Reponse | Source Python |
| --- | --- | --- | --- | --- |
| GET | /api/v1/players/{player_slug}/pages/career | Charger la page carriere complete | CareerPageResponse | career.py, career_data.py, career_logic.py |
| GET | /api/v1/players/{player_slug}/pages/career/top-matches | Charger la section top matches de facon lazy si besoin | CareerTopMatchesResponse | career_top_matches_* |
| GET | /api/v1/players/{player_slug}/pages/career/encounters | Charger les encounters si charges en differe | CareerEncountersResponse | career_encounters_* |

### Schema CareerPageResponse

- summary :
  - rank_number
  - rank_label
  - rank_name_raw
  - rank_tier
  - current_xp
  - xp_for_next_rank
  - xp_total
  - progress_pct
  - is_max_rank
  - recorded_at
- hero_progress :
  - xp_total_required
  - xp_remaining
  - percentage
  - current_rank
- projections :
  - xp_per_day_active
  - xp_per_day_fallback
  - estimated_hero_date
  - estimated_rank_cap_date
- charts :
  - rank_progress_gauge : PlotlyFigurePayload | null
  - hero_progress_gauge : PlotlyFigurePayload | null
  - xp_history_figure : PlotlyFigurePayload | null
  - lusr_rating_figure : PlotlyFigurePayload | null
- xp_history : list[CareerHistoryPoint]
  - recorded_at
  - rank_number
  - rank_label
  - xp_total
- lusr :
  - current_rating
  - current_tier_label
  - current_playlist_group
  - trend_label
  - checkpoints : list[CareerLusrCheckpoint]
- top_matches_preview : list[CareerTopMatch]
- encounters_preview : list[CareerEncounter]

CareerTopMatch :

- match_id
- start_time
- map_ui
- mode_ui
- playlist_label
- performance_score
- badge_type
- score_label
- outcome_label

CareerEncounter :

- encounter_key
- opponent_gamertag
- count_matches
- wins
- losses
- last_seen_at

### Stores front et query keys recommandes

- useCareerPageStore
  - expandedPanels
  - selectedTopMatchTab si necessaire
- Query keys
  - ['career', playerSlug]
  - ['career', playerSlug, 'top-matches']
  - ['career', playerSlug, 'encounters']

### Criteres de recette

- meme rang, meme XP total, meme progression Hero et meme statut max rank que Streamlit
- meme historique XP et memes projections pour un joueur donne
- meme LUSR courant et meme tendance
- meme liste des top matches et encounters a donnees equivalentes

## Contrats detailles - Match History

### Route front cible

- /players/:playerSlug/history

### Endpoints recommandes

| Methode | Path | Usage | Requete | Reponse | Source Python |
| --- | --- | --- | --- | --- | --- |
| POST | /api/v1/players/{player_slug}/pages/match-history/query | Charger le tableau pagine/triable | MatchHistoryQueryRequest | MatchHistoryPageResponse | match_history.py |
| POST | /api/v1/players/{player_slug}/pages/match-history/export | Exporter le scope courant | MatchHistoryExportRequest | FileTokenResponse | match_history.py |

### Schema MatchHistoryQueryRequest

- filters : FilterContextInput
- pagination : PaginationRequest
- columns : list[str] | null
- include_export_hint : bool

### Schema MatchHistoryPageResponse

- summary :
  - total_matches_scoped
  - total_matches_unfiltered
  - period_label
  - active_filter_mode
- table : PaginatedResponse[MatchHistoryRow]
- available_sort_fields : list[str]
- export_hint :
  - file_name
  - estimated_rows
  - token | null

MatchHistoryRow :

- match_id
- start_time
- start_time_label
- outcome_code
- outcome_label
- score_label
- map_ui
- mode_ui
- playlist_label
- team_mmr
- enemy_mmr
- delta_mmr
- win_rate_hist
- win_rate_hist_total
- performance_score_relative
- average_life_mmss
- match_url

### Stores front et query keys recommandes

- useGlobalFilterStore
  - filterContext
  - resolvedOptions
- useMatchHistoryTableStore
  - page
  - pageSize
  - sorting
  - visibleColumns
- Query keys
  - ['filters-resolve', playerSlug, filterContextHash]
  - ['match-history', playerSlug, filterContextHash, page, pageSize, sortHash]

### Criteres de recette

- meme cardinalite de lignes que la page Streamlit a scope egal
- meme ordre pour un tri equivalent
- memes valeurs calculees sur chaque ligne critique : score, win_rate_hist, performance_score_relative, delta_mmr
- export CSV representant exactement le scope courant

## Contrats detailles - Explorer

### Route front cible

- /players/:playerSlug/explorer

### Endpoints recommandes

| Methode | Path | Usage | Requete | Reponse | Source Python |
| --- | --- | --- | --- | --- | --- |
| GET | /api/v1/directory/gamertags/search | Suggestions fuzzy pour la recherche joueur | q, limit | GamertagSearchResponse | explorer_logic.py, explorer_data.py |
| POST | /api/v1/players/{player_slug}/pages/explorer/matches-query | Resultats par filtres de matchs | ExplorerMatchesQueryRequest | ExplorerMatchesQueryResponse | explorer.py |
| POST | /api/v1/players/{player_slug}/pages/explorer/player-query | Resultats orientes joueur cible | ExplorerPlayerQueryRequest | ExplorerPlayerQueryResponse | explorer_results.py |

### Schemas detailles

GamertagSearchResponse :

- query
- items : list[GamertagSuggestion]

GamertagSuggestion :

- gamertag
- xuid
- score
- exact_match : bool

ExplorerMatchesQueryRequest :

- filters : FilterContextInput
- match_filters :
  - selected_date : date | null
  - squad_scope : solo | squad | all
  - experience_type : str | null
  - playlist : str | null
  - mode : str | null
  - map : str | null
  - selected_match_id : str | null
- pagination : PaginationRequest

ExplorerMatchesQueryResponse :

- summary :
  - total_matches
  - selected_match_id
- table : PaginatedResponse[ExplorerMatchRow]

ExplorerMatchRow :

- match_id
- start_time
- start_time_label
- map_ui
- mode_ui
- playlist_label
- outcome_label
- score_label
- is_with_friends
- experience_type_label

ExplorerPlayerQueryRequest :

- target_gamertag : str
- filters : FilterContextInput | null

ExplorerPlayerQueryResponse :

- target :
  - gamertag
  - xuid
- summary :
  - matches_together
  - wins_together
  - losses_together
  - last_seen_at
- allies_table : list[ExplorerEncounterRow]
- enemies_table : list[ExplorerEncounterRow]
- common_matches : list[ExplorerMatchRow]

ExplorerEncounterRow :

- match_id
- start_time
- map_ui
- mode_ui
- same_team
- outcome_label
- score_label

### Stores front et query keys recommandes

- useExplorerStore
  - searchMode : matches | player
  - playerSearchInput
  - selectedMatchId
  - localMatchFilters
  - pagination
- Query keys
  - ['gamertag-search', q]
  - ['explorer', 'matches', playerSlug, filterContextHash, localMatchFilterHash, page, sortHash]
  - ['explorer', 'player', playerSlug, targetGamertag, filterContextHash]

### Criteres de recette

- memes suggestions fuzzy pertinentes pour une meme requete
- meme resolution gamertag -> xuid qu'en Streamlit
- memes resultats de filtres match par match
- meme comportement sur deep links match_id et gamertag
- ouverture du meme Match View a partir d'une ligne ou d'un deep link

## Contrats detailles - Match View et Last Match

### Routes front cibles

- /players/:playerSlug/matches/:matchId
- /players/:playerSlug/last-match

### Endpoints recommandes

| Methode | Path | Usage | Requete | Reponse | Source Python |
| --- | --- | --- | --- | --- | --- |
| GET | /api/v1/players/{player_slug}/matches/{match_id} | Charger toute la vue match | - | MatchViewResponse | match_view.py + match_view_* |
| POST | /api/v1/players/{player_slug}/pages/last-match/resolve | Determiner le match courant et les voisins dans le scope filtre | LastMatchResolveRequest | LastMatchResolveResponse | last_match.py |

### Schema MatchViewResponse

- header :
  - match_id
  - start_time
  - start_time_label
  - outcome_code
  - outcome_label
  - outcome_color
  - score_label
  - dominance_flag
  - had_bot_teammate
  - map_ui
  - map_id
  - mode_ui
  - playlist_label
  - performance_display
  - performance_color
- rank :
  - rating_type : CSR | LUSR | none
  - tier_label
  - numeric_value
  - delta_value
  - icon_url | null
- summary_tab :
  - kpis : dict
  - personal_result : dict
  - medals : list[MatchMedal]
  - citations : list[MatchCitation]
- combat_tab :
  - weapon_kills : list[MatchWeaponKill]
  - highlight_events : list[MatchHighlightEvent]
  - charts : list[PlotlyFigurePayload]
- team_tab :
  - roster : list[MatchRosterRow]
  - scoreboard : list[MatchScoreboardRow]
  - nemesis : list[MatchNemesisRow]
  - encounters : list[MatchEncounterRow]
- media_tab :
  - media_items : list[AssociatedMediaItem]
- citations_tab :
  - commendations : list[MatchCitation]
  - medals : list[MatchMedal]

Schemas critiques :

MatchRosterRow :

- xuid
- gamertag
- team_side
- is_me
- is_bot
- kills
- deaths
- assists
- kda
- damage_dealt
- damage_taken

MatchWeaponKill :

- weapon_id
- weapon_label
- effective_weapon_id
- kill_count

MatchHighlightEvent :

- event_time_ms
- event_type
- actor_xuid
- target_xuid
- weapon_id | null

LastMatchResolveRequest :

- filters : FilterContextInput

LastMatchResolveResponse :

- current_match_id
- total_matches_in_scope
- current_index
- previous_match_id : str | null
- next_match_id : str | null
- session_tracking_key : str

### Stores front et query keys recommandes

- useMatchViewStore
  - activeTab
  - selectedScoreboardRow
  - mediaLightboxIndex
- useLastMatchStore
  - resolvedMatchId
  - currentIndex
  - total
- Query keys
  - ['match-view', playerSlug, matchId]
  - ['last-match', playerSlug, filterContextHash]

### Criteres de recette

- meme score, meme outcome, meme dominance flag et meme rank affiches pour un match donne
- meme scoreboard, meme roster, meme set de medailles et de citations
- meme section armes et meme timeline d'evenements a donnees equivalentes
- Last Match pointe vers le meme match que Streamlit pour un scope donne
- prev/next navigue sur la meme liste ordonnee que la page Streamlit

## Etape critique 4 detaillee - sortir la logique cachee dans Streamlit

Cette section transforme l'etape 4 en chantier d'architecture explicite. Le probleme a traiter n'est pas seulement le rendu UI. Le vrai sujet est d'extraire les mecanismes aujourd'hui disperses entre `st.session_state`, query params, caches Streamlit, callbacks de rendu et reruns implicites, afin de les remplacer par des contrats et des proprietaires d'etat clairs.

## Objectif operationnel

- identifier toute logique qui depend aujourd'hui du moteur de rerun Streamlit plutot que d'un contrat explicite
- assigner a chaque categorie d'etat une cible unique : URL, store front, session backend, cache serveur ou persistence navigateur
- supprimer les couplages implicites entre rendu, chargement de donnees, navigation et side effects
- rendre chaque parcours React reproductible a partir d'une URL, d'un contexte de session backend et d'appels API explicites

## Principe directeur

Tout etat doit avoir un seul proprietaire legitime.

- etat navigable et partageable : URL
- etat serveur et donnees distantes : TanStack Query + backend
- etat UI ephemere local : Zustand ou etat composant
- preferences navigateur non sensibles : localStorage ou IndexedDB
- auth, joueur courant persiste si necessaire, jobs longs et secrets : session backend

## Inventaire des logiques cachees a extraire

### 1. Navigation et deep links

Aujourd'hui, une partie importante de la navigation est reconstruite via `st.query_params`, redirects internes et cles temporaires `_pending_*`.

Elements a sortir :

- `page`, `match_id`, `gamertag`, `player`, `stats_view`, `session`, `scope`
- `_pending_page`, `_pending_match_id`, `_pending_gamertag`, `_pending_player`
- `v7_current_section`, `v7_stats_view`, `v7_profile_view`
- `_last_match_nav_index`, `_last_match_nav_total`, `_last_match_nav_session_key`

Remplacement cible :

- TanStack Router pour la route et les search params
- route params explicites pour `playerSlug`, `matchId`, `section`
- search params uniquement pour les etats partageables ou restituables au refresh
- disparition des cles `_pending_*` au profit d'une navigation declarative

### 2. Filtres globaux et modele de sessions

Le systeme actuel combine contexte charge dans `streamlit_app.py`, shadow keys, widgets de filtre et reruns pour maintenir la coherence du scope.

Elements a sortir :

- `filter_mode`, `start_date_cal`, `end_date_cal`
- `picked_session_label`, `picked_solo_session_label`, `picked_squad_session_label`, `picked_sessions`
- filtres cascade `experience_types`, `playlists`, `modes`, `maps`
- la logique de `GAP_MINUTES_FIXED = 120`
- les shadow keys servant a survivre aux reruns ou a `st.navigation`

Remplacement cible :

- `FilterContextInput` comme source de verite front
- endpoint `filters/resolve` comme arbitre serveur des options, sessions et compteurs
- `useGlobalFilterStore` pour le brouillon d'interaction local
- synchronisation URL seulement pour les filtres devant etre partageables ou restaurables

### 3. Etat de page et interaction locale

Beaucoup de comportements locaux sont aujourd'hui masques dans `session_state` alors qu'ils relevent d'un etat purement UI ou d'une derivee de la route.

Elements a sortir :

- `match_id_input`, `_explorer_selected_match`
- `compare_session_a`, `compare_session_b`, `_last_picked_for_compare`
- `teammates_picked_labels`, `_cache_warning_shown`, `show_records`
- `_lb_state`, `mv2_autoplay`
- etats de panneaux, d'onglets, de tri local, de lightbox et de selection ligne

Remplacement cible :

- Zustand pour l'etat local transverse a une feature
- etat composant pur quand l'information ne sort pas du composant
- route/search params seulement si l'etat doit etre partageable ou restaurable par URL

### 4. Bootstrap, setup et preferences utilisateur

Le bootstrap actuel depend d'un ordre de rerun : restauration des prefs navigateur, lecture des settings, verification setup, auth, selection joueur, puis affichage de l'app.

Elements a sortir :

- `_setup_mode`, `_xbox_oauth_result`, `_smoke_*`
- lecture/patch de `app_settings`
- restauration de `lang`, `show_hints`, dernier joueur utilise et autres prefs navigateur
- selection joueur via `db_path`, `xuid_input`, `waypoint_player`

Remplacement cible :

- endpoint `bootstrap` unique pour hydrater le shell
- endpoints `setup/*`, `auth/*`, `settings/*` pour chaque mutation critique
- `useAppShellStore` pour l'etat shell minimal
- localStorage pour les preferences purement navigateur
- session backend pour la progression du setup et les etats auth sensibles

### 5. Cache, invalidation et dependances au rerun

Une partie de la coherence de l'app vient aujourd'hui du fait que Streamlit rerun toute la page apres un click, un changement de filtre ou une mutation de cache. Cette mecanique doit etre remplacee par des invalidations explicites.

Elements a sortir :

- `src/ui/_cache_core.py`, `src/ui/_cache_loading.py`, `src/ui/_cache_queries.py`, `src/ui/_cache_sessions.py`, `src/ui/cache.py`
- `src/app/cache_control.py` et toute invalidation basee sur refresh global
- les dependances implicites ou un rerun recalcule l'ecran entier pour obtenir un etat coherent

Remplacement cible :

- TanStack Query pour la coherence des donnees cote front
- cache backend ou process cache explicite pour les ressources couteuses qui restent cote Python
- invalidation par query keys, mutations et fin de job, jamais par rerun global implicite
- `freshness` et warnings exposes par l'API plutot que deduits du comportement de cache Streamlit

### 6. Jobs longs et services en arriere-plan

Les syncs, backfills, scans media et quelques caches home fonctionnent aujourd'hui en s'appuyant sur le runtime process Streamlit et des side effects au demarrage.

Elements a sortir :

- `background_media_indexing`
- `reindex_media_after_sync`
- smoke test, sync et backfill pilotant ensuite l'UI via rerun
- process cache de la home V7 pour battle pass / challenges

Remplacement cible :

- endpoints de jobs avec `AsyncJobStatus`
- polling ou streaming cote front selon le cout reel du besoin
- start explicite, progression explicite, invalidation explicite a la fin d'un job
- les traitements de fond ne doivent plus supposer l'existence d'une session Streamlit vivante

## Matrice de remplacement recommandee

| Type de logique | Mecanisme Streamlit actuel | Cible React/FastAPI | Regle de migration |
| --- | --- | --- | --- |
| Navigation partageable | query params + `_pending_*` | Router + search params | tout etat partageable doit etre reconstruisible par URL seule |
| Donnees serveur | cache Streamlit + rerun | TanStack Query + API | aucune donnee distante ne doit dependre d'un rerun global |
| Etat UI local | `st.session_state` | Zustand ou state composant | ne promouvoir en store que ce qui traverse plusieurs composants |
| Preferences navigateur | browser storage + session_state | localStorage / IndexedDB | aucune preference non sensible ne doit transiter par session backend sans raison |
| Auth et secrets | process cache + session Streamlit | session backend opaque | aucun token ne fuit vers le navigateur |
| Jobs longs | side effects + rerun | endpoints jobs + invalidation | la fin de job doit etre visible sans recharger toute l'app |

## Anti-patterns a eliminer explicitement

- un rendu de composant qui ecrit dans l'etat global pour preparer un rerun suivant
- un filtre dont la valeur reelle n'est connue qu'apres rerun complet de la page
- une navigation qui passe par une cle temporaire au lieu d'une URL explicite
- une mutation serveur qui suppose un refresh total pour rendre l'ecran coherent
- un cache sans proprietaire clair, sans politique d'expiration et sans convention d'invalidation

## Ordre d'extraction recommande

1. Sortir le shell, le bootstrap, le joueur courant et les query params canoniques.
2. Sortir le moteur de filtres globaux derriere `FilterContextInput` et `filters/resolve`.
3. Sortir setup, auth et settings des enchainements de rerun.
4. Sortir les etats de page locaux dans les features MVP : Match History, Explorer, Match View, Last Match.
5. Sortir les caches et jobs de fond derriere des invalidations et statuts explicites.

## Definition of done pour cette etape 4

L'etape critique 4 est consideree comme couverte si :

- chaque cle `session_state` encore necessaire a un parcours MVP a ete classee dans une categorie cible explicite
- chaque deep link utile est porte par une vraie route ou un search param stable, sans `_pending_*`
- les filtres globaux ont un contrat d'entree/sortie explicite et ne dependent plus des shadow keys Streamlit
- aucune mutation critique du MVP n'a besoin d'un rerun global pour mettre l'ecran dans un etat coherent
- les jobs longs, preferences navigateur, caches et sessions ont chacun un proprietaire unique et observable

## Etape critique 5 detaillee - preparer la structure cible du repo dans le worktree courant

Cette section transforme l'etape 5 en decision d'implantation concrete. Le but n'est pas de lancer un grand deplacement de code avant d'avoir une API et un front utilisables. Le but est de figer une structure cohabitable dans le worktree courant, pour que FastAPI et React puissent etre introduits sans dupliquer le coeur Python ni casser la reference Streamlit existante.

## Objectif operationnel

- definir ou vivent le noyau metier Python, la couche HTTP, le front React, les tests et les artefacts de build
- fixer les zones legacy et transitoires a conserver pendant la cohabitation
- eviter les faux bons raccourcis comme melanger TypeScript dans `src/ui/` ou mettre l'orchestration HTTP dans `src/data/`
- rendre les Slices 0 a 5 implementables sans reouvrir l'architecture du repo a chaque etape

## Decisions structurantes a figer

- `src/` reste la source de verite metier Python pendant toute la migration.
- `apps/api/` porte exclusivement l'exposition HTTP, les schemas d'entree/sortie, l'orchestration de session et les jobs backend.
- `apps/web/` porte exclusivement la nouvelle UI React/TypeScript/Vite.
- `streamlit_app.py`, `streamlit_app_v7.py`, `src/ui/` et `src/app/` restent en place tant que la decommission n'est pas terminee.
- Le runtime Python reste gere par le `pyproject.toml` racine. On n'ouvre pas un deuxieme projet Python dans `apps/api/`.
- Le runtime Node est localise dans `apps/web/` avec son propre `package.json`.
- On n'introduit pas de monorepo tool supplementaire (`turbo`, `nx`, `pnpm workspaces`) dans le premier passage. La valeur est dans la parite produit, pas dans une infra JS plus lourde.

## Structure cible a figer dans ce worktree

Arborescence cible recommandee :

```text
.ai/
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
        setup.tsx
        settings.tsx
        players/
          $playerSlug/
            career.tsx
            history.tsx
            explorer.tsx
            matches/
              $matchId.tsx
            last-match.tsx
      features/
        bootstrap/
        filters/
        setup/
        settings/
        career/
        match-history/
        explorer/
        match-view/
      components/
      stores/
      lib/
        api/
        query/
        utils/
      styles/
data/
docs/
packaging/
scripts/
spnkr_pr/
src/
  analysis/
  app/
  auth/
  data/
  ports/
  ui/
  utils/
  visualization/
static/
tests/
Dockerfile
Makefile
pyproject.toml
streamlit_app.py
streamlit_app_v7.py
```

## Repartition des responsabilites par zone

### 1. Noyau Python conserve dans `src/`

- `src/analysis/`, `src/data/`, `src/auth/`, `src/visualization/`, `src/utils/` et `src/ports/` restent la source de verite.
- Toute logique metier extraite des pages Streamlit doit aller dans `src/analysis/` ou `src/data/services/`, pas dans `apps/api/`.
- Les figures Plotly restent generees ici tant qu'elles ne sont pas remplacees.

### 2. Couche HTTP concentree dans `apps/api/`

- `routers/` ne contient que la definition des routes, la validation HTTP et le mapping vers les services.
- `schemas/` contient les payloads FastAPI/Pydantic specifiques au transport.
- `services/` compose les appels vers `src/` et assemble les reponses de page.
- `deps/` porte l'injection de dependances HTTP : joueur courant, session, auth, resolveur de filtres.
- `jobs/` porte l'orchestration des traitements longs exposes a l'UI.

### 3. Front React concentre dans `apps/web/`

- `routes/` decrit l'arbre de navigation URL-first.
- `features/` porte le code par vertical slice, pas par type technique.
- `stores/` reste reserve aux stores shell et cross-feature. Les etats locaux d'une feature restent dans `features/<feature>/`.
- `lib/api/` centralise le client HTTP, la serialisation des search params et les helpers TanStack Query.
- `components/` porte les composants vraiment partages. On evite d'y recreer un nouveau fourre-tout UI.

### 4. Zone legacy maintenue pendant la cohabitation

- `streamlit_app.py`, `streamlit_app_v7.py`, `src/ui/` et `src/app/` restent en place tant que les slices React equivalentes ne sont pas validees.
- Ces modules deviennent la reference fonctionnelle a comparer, pas l'endroit ou l'on continue d'ajouter des comportements nouveaux sans necessite forte.
- On ne deplace pas ces dossiers au debut. On les vide progressivement a mesure que les parcours sont remplaces.

## Regles de placement a ne pas negocier

- Pas de duplication de logique metier entre `src/` et `apps/api/`.
- Pas de TypeScript ou d'assets React dans `src/ui/`.
- Pas de code FastAPI transporte dans `streamlit_app.py` ou `launcher.py`.
- Pas de deuxieme `pyproject.toml` tant que l'API n'a pas besoin d'une isolation packaging reelle.
- Pas de package Node a la racine : tout le front vit dans `apps/web/`.
- Pas de nouveau package `shared/` ou `common/` fourre-tout au premier passage. Si un besoin de partage apparait, il devra etre prouve par duplication reelle.

## Mapping concret des slices vers la structure cible

- Slice 0 : creer `apps/api/app/main.py`, `apps/api/app/routers/bootstrap.py`, `apps/api/app/routers/players.py`, `apps/api/app/schemas/common.py`, `apps/api/app/schemas/filters.py`, puis `apps/web/src/app/`, `apps/web/src/routes/__root.tsx` et le shell initial.
- Slice 1 : ajouter `apps/api/app/routers/setup.py`, `apps/api/app/routers/settings.py`, `apps/api/app/jobs/`, puis `apps/web/src/routes/setup.tsx`, `apps/web/src/routes/settings.tsx` et les features associees.
- Slice 2 : ajouter `apps/api/app/routers/career.py`, `apps/api/app/schemas/pages/career.py`, `apps/api/app/services/pages/career_service.py`, puis `apps/web/src/features/career/` et la route carriere.
- Slice 3 : ajouter `apps/api/app/routers/match_history.py` et `apps/web/src/features/match-history/`.
- Slice 4 : ajouter `apps/api/app/routers/explorer.py` et `apps/web/src/features/explorer/`.
- Slice 5 : ajouter `apps/api/app/routers/matches.py` et `apps/web/src/features/match-view/`, puis la vue `last-match` comme derivee du meme contrat.

## Impacts racine a anticiper des maintenant

- `.gitignore` devra couvrir `apps/web/node_modules`, `apps/web/dist` et les caches outils frontend.
- `Dockerfile` et `docker-compose.yml` devront passer a un build multi-etapes Python + frontend.
- `Makefile` ou les scripts de dev devront exposer au minimum un run API, un run web et un run combine.
- `README.md` et `docs/INSTALL.md` devront distinguer clairement le mode Streamlit legacy et le mode FastAPI + React.
- `tests/` reste au niveau racine pour les tests Python, les tests de parite et les futures verifications de contrats API. Les tests UI web peuvent vivre sous `apps/web/` si necessaire, sans dupliquer les tests de parite backend.

## Sequence de mise en place recommandee dans le worktree courant

1. Ajouter `apps/api/` sans deplacer `src/`.
2. Ajouter `apps/web/` sans toucher a `src/ui/`.
3. Brancher le shell React sur les premiers endpoints Slice 0 et Slice 1.
4. Migrer les pages MVP par vertical slices.
5. Supprimer les surfaces Streamlit seulement quand la route React equivalente est validee en parite.

## Definition of done pour cette etape 5

L'etape critique 5 est consideree comme couverte si :

- la place de `src/`, `apps/api/`, `apps/web/` et des zones legacy est figee sans ambiguite
- aucune equipe ne peut raisonnablement hesiter sur l'endroit ou ajouter une route API, un schema HTTP, une feature React ou une extraction metier Python
- la cohabitation Streamlit/React est rendue explicite dans la structure elle-meme
- les Slices 0 a 5 peuvent etre implementes sans nouvelle discussion sur la topologie du repo
- la future decommission Streamlit est preparee, sans exiger un deplacement massif de code au debut

## Etape critique 6 detaillee - migrer par parcours metier, pas par couches techniques

Cette section transforme l'etape 6 en discipline de delivery. L'objectif n'est pas d'avoir d'abord "100 % de l'API", puis "100 % du front", puis "100 % des stores". L'objectif est de fermer des parcours utilisateurs entiers, chacun avec ses contrats, ses etats, ses ecrans et ses criteres de parite.

## Objectif operationnel

- definir l'unite de livraison minimale de la migration
- ordonner le backlog autour de routes et d'usages reels, pas autour des couches internes
- limiter la dette de cohabitation entre Streamlit et React
- rendre chaque regression attribuable a une tranche claire et demonstrable

## Unite de livraison cible

Un vertical slice acceptable doit couvrir, pour un parcours donne :

- une route React adressable par URL
- un ou plusieurs endpoints suffisants pour alimenter l'ecran sans logique metier critique cote client
- un proprietaire clair pour l'etat : URL, store front, session backend ou cache serveur
- les etats loading, empty, error et retry
- au moins un scenario de parite et au moins un signal de mesure
- une bascule ou un mode d'acces explicite dans la phase de cohabitation

## Regle de decoupage

1. Partir d'une intention utilisateur observable : consulter sa carriere, filtrer l'historique, chercher un match, ouvrir un detail.
2. Identifier le minimum de contexte necessaire : joueur courant, filtres, deep link, auth eventuelle, permissions.
3. Exposer un contrat d'API suffisant pour fermer la page, pas pour "finir la couche data" au sens abstrait.
4. Construire le rendu React complet de la route, y compris loading, empty, error et navigation.
5. Ajouter les tests de parite et l'instrumentation minimale avant de declarer le slice livrable.

## Regles de priorisation

- preferer les parcours a forte valeur visible et a faible dette de state cache
- preferer les pages dont la logique metier est deja bien extraite en services ou en modules purs
- repousser les abstractions generiques tant que deux ou trois slices ne les exigent pas vraiment
- autoriser une fondation transverse seulement si elle debloque immediatement plusieurs slices et produit une sortie tangible

## Anti-patterns a eviter

- une "phase API" qui produit des endpoints sans ecran utilisable
- une "phase composants" qui produit une bibliotheque UI sans parcours branche
- une migration des graphes, de la table system ou des stores comme chantier autonome avant qu'un premier ecran complet existe
- un epic defini comme "terminer auth front" ou "terminer tous les endpoints" sans route metier cible
- la duplication temporaire de calculs dans le front pour compenser un backend pas encore tranche

## Definition of done pour cette etape 6

L'etape critique 6 est consideree comme couverte si :

- le backlog d'execution est exprime en routes ou parcours metier, pas en couches techniques
- chaque slice prioritaire nomme sa route, ses endpoints, son proprietaire d'etat, ses criteres de parite et ses metriques minimales
- les fondations transverses restantes sont rattachees explicitement aux slices qu'elles debloquent
- aucune tranche MVP n'est planifiee comme simple travail de plomberie sans resultat demonstrable cote utilisateur

## Etape critique 7 detaillee - organiser la cohabitation Streamlit / React

Cette section transforme l'etape 7 en modele de transition operable. Le sujet n'est pas seulement de faire tourner deux fronts en meme temps. Le sujet est de savoir lequel possede quelle surface, comment on bascule le trafic, comment on garde un rollback simple et comment on evite de dupliquer la logique metier pendant des mois.

## Objectif operationnel

- permettre une migration progressive sans big bang
- maintenir une seule source de verite backend pendant toute la transition
- rendre explicite le proprietaire canonique de chaque parcours utilisateur
- garantir un rollback rapide si un ecran React regressse en production

## Modele de cohabitation recommande

### 1. Un seul coeur backend

- repositories DuckDB, services, analyses, auth Halo, sync et generation Plotly restent proprietaires des calculs
- Streamlit et React consomment le meme coeur Python, meme si les formes d'acces divergent temporairement
- aucune branche de calcul metier "speciale React" ne doit devenir une seconde source de verite

### 2. Un proprietaire canonique par surface

Une surface fonctionnelle ne doit jamais avoir deux fronts canoniques simultanement.

| Etat d'une surface | Front canonique | Front secondaire | Regle |
| --- | --- | --- | --- |
| Legacy seule | Streamlit | aucun | surface non commencee cote React |
| Preview React | Streamlit | React | acces React reserve au dev, au flag ou a une URL dediee |
| Bascule canonique | React | Streamlit | React devient l'entree principale, Streamlit reste rollback court terme |
| Decommissionnee | React | aucun | la route Streamlit est retiree ou redirigee |

### 3. Compatibilite de navigation

- les deep links utiles doivent survivre a la cohabitation : player, match_id, gamertag, session, scope
- les routes React migrantes doivent pouvoir etre appelees depuis les CTA encore rendus dans Streamlit
- les routes Streamlit restantes doivent etre accessibles depuis le shell React quand un parcours n'est pas encore migre

### 4. Bascule par feature flag ou point d'entree explicite

- en dev local : deux serveurs distincts sont acceptables si le modele de donnees et de session reste coherent
- en integration ou prod : il faut un point d'entree explicite par surface, via reverse proxy, feature flag ou route dediee
- toute bascule doit etre reversible sans migration de donnees ni changement manuel de DB

## Regles de bascule recommandees

1. Ouvrir une surface React en preview interne seulement apres validation des tests de parite critiques.
2. Passer en canonique React uniquement quand la navigation, l'instrumentation et le rollback sont prets.
3. Garder la version Streamlit seulement comme filet de securite court terme, pas comme double maintenance indefinie.
4. Retirer la version Streamlit des qu'une surface React a stabilise ses chiffres, ses usages et ses erreurs sur une periode observee.

## Anti-patterns a eviter

- faire coexister deux versions modifiables du meme ecran sans proprietaire canonique
- dupliquer des calculs ou des regles d'aggregation entre Streamlit et React "pour aller plus vite"
- basculer une page React en canonique sans route de rollback simple
- garder la cohabitation ouverte trop longtemps sur une surface deja migree, au point de figer toute evolution

## Definition of done pour cette etape 7

L'etape critique 7 est consideree comme couverte si :

- chaque surface de la matrice a un etat de cohabitation explicite : legacy, preview, canonique React ou retiree
- chaque bascule prevoit un point d'entree, un rollback et une date cible de retrait du fallback
- aucune surface MVP n'a deux fronts canoniques concurrents
- le modele de cohabitation n'impose jamais une seconde source de verite metier

## Etape critique 8 detaillee - traiter auth, permissions et etat de session tres tot

Cette section transforme l'etape 8 en frontiere d'architecture. L'auth et le state model ne sont pas des sujets "a brancher a la fin". Ils conditionnent le bootstrap, les jobs longs, les ecrans settings, le setup, les actions de sync et la cohabitation entre fronts.

## Objectif operationnel

- figer tres tot le proprietaire des secrets, de la session applicative et des preferences utilisateur
- rendre explicites les capacites requises pour chaque action critique
- definir les comportements d'expiration, de reconnexion, de reprise de session et de changement de joueur
- eviter que l'auth ou la session deviennent des dependances implicites dispersees dans les pages

## Frontieres a figer des le depart

### 1. Propriete des secrets

- les tokens Halo, le cache MSAL, les refresh tokens et les secrets restent exclusivement cote backend
- le navigateur ne recoit jamais de token exploitable directement
- le front ne manipule que des etats derives : auth_state, needs_reauth, current_player, capabilities, jobs accessibles

### 2. Session applicative minimale

La session backend ou son equivalent doit pouvoir porter au minimum :

- le joueur courant s'il est persiste cote serveur
- la langue courante ou son fallback
- l'etat d'auth effectif
- les flows de setup ou de device code en cours
- les jobs longs lies a l'utilisateur ou au shell courant

### 3. Modele de permissions et de capacites

Le plan n'a pas besoin d'un RBAC complexe pour exister, mais il a besoin d'un modele explicite de capacites. A minima, le front doit savoir distinguer :

- lecture locale des donnees deja synchronisees
- actions necessitant une auth Halo valide : sync, refresh live, battle pass, defis, device flow
- actions d'exploitation locale : reset index media, watcher, setup, changements de settings sensibles
- actions longues ou potentiellement destructrices qui demandent confirmation et suivi de job

Ces capacites doivent venir du backend dans les payloads de bootstrap ou de page, pas d'heuristiques UI cachees dans les composants.

### 4. Expiration, reconnexion et reprise

Il faut figer des maintenant :

- le comportement quand une session est absente au chargement initial
- le comportement quand une auth expire pendant une page ouverte
- la maniere de reprendre un device flow ou de relancer une auth sans perdre l'etat navigable
- la maniere de rehydrater le shell apres refresh navigateur ou changement de joueur

### 5. Preferences etat local vs etat serveur

- preferences purement locales : hints, preferences visuelles locales, etats de panneaux non partages
- preferences partagees ou d'exploitation : langue, timezone, options de sync, watcher media, notifications
- le plan doit classer chaque preference des maintenant pour eviter des aller-retours inutiles entre localStorage, Zustand et backend

## Anti-patterns a eviter

- stocker des tokens, refresh tokens ou identifiants sensibles dans localStorage
- laisser les composants UI deduire seuls qu'une action est permise ou non
- repliquer des cles `session_state` Streamlit dans des stores React sans redefinir leur contrat
- traiter la reconnexion comme un simple message d'erreur generique sans chemin de reprise clair

## Definition of done pour cette etape 8

L'etape critique 8 est consideree comme couverte si :

- le front ne voit jamais les secrets d'auth ni le contenu du cache MSAL
- le bootstrap ou les endpoints transverses exposent explicitement auth_state, current_player, preferences utiles et capacites
- les actions sensibles du MVP ont un comportement specifie pour les cas non-authentifie, expire, reauth et job en cours
- chaque preference du MVP a un proprietaire tranche : session backend, persistance disque, localStorage ou store local

## Etape critique 9 detaillee - mettre des tests de parite

Cette section transforme l'etape 9 en filet de securite concret. Le but n'est pas d'obtenir une parite pixel perfect. Le but est de demontrer que les surfaces migrees produisent les memes comportements metier et les memes chiffres critiques que la reference Streamlit sur un corpus connu.

## Objectif operationnel

- definir une base de comparaison stable entre la reference Streamlit et la cible React/FastAPI
- automatiser les verifications sur les parcours a plus fort risque metier
- faire de la parite un critere de bascule, pas une verif manuelle de fin de chantier
- isoler les regressions entre calcul metier, contrat d'API, rendu front et navigation

## Corpus de reference a figer

Le plan doit s'appuyer sur un petit corpus stable, versionne et rejouable :

- un ou plusieurs joueurs de reference
- quelques scopes de filtres figes : periode, session solo, session squad, playlist/mode/map quand utile
- une liste de match_ids de reference pour Match View et Last Match
- au moins un cas FR et un cas EN pour verifier les labels critiques
- un cas avec auth valide et, si possible, un cas sans auth pour les ecrans concernes

## Niveaux de parite a verifier

| Niveau | Ce qu'on compare | Exemples |
| --- | --- | --- |
| Parite de donnees | memes chiffres et memes agregats | XP, LUSR, compteurs, cardinalites, deltas |
| Parite de tri/filtre | meme scope et meme ordre | pagination, tri table, filtres cascade, last match scope |
| Parite de navigation | memes cibles et memes deep links | ouverture match, route joueur, prev/next |
| Parite de comportement | memes side effects observables | export, reset, setup gating, smoke test, likes, jobs |

## Strategie de test recommandee

### 1. Tests backend de reference

- comparer les payloads API a des calculs issus des services Python ou des modules de page existants
- figer des golden payloads seulement sur les zones stables, pas sur des champs purement cosmetiques ou timestamps volatils
- comparer les figures Plotly sur des invariants utiles : traces, labels, series, annotations critiques, pas sur un JSON brut fragile si cela cree trop de bruit

### 2. Tests front de parcours critiques

- verifier qu'une route React affiche les memes KPIs critiques qu'une reponse API de reference
- verifier les cas loading, empty, error et retry, qui sont souvent absents de Streamlit mais essentiels dans le web
- verifier la coherence URL-first : refresh, deep link direct, navigation prev/next, retour depuis une table

### 3. Gates de bascule

- aucun ecran ne passe en canonique React sans scenario de parite explicite
- toute regression de chiffres bloque la bascule, meme si le rendu visuel parait plus abouti
- les ecarts volontaires doivent etre documentes comme decisions produit, pas acceptes implicitement

## Anti-patterns a eviter

- ne comparer que des screenshots ou un ressenti manuel de recette
- lancer les tests sur des donnees mouvantes sans corpus de reference fige
- exiger une egalite stricte sur des champs volatils non metiers et rendre la suite inutilisable
- attendre la fin du chantier pour definir ce qui doit etre considere comme equivalent

## Definition of done pour cette etape 9

L'etape critique 9 est consideree comme couverte si :

- chaque slice prioritaire reference un scenario de parite automatise ou scriptable
- un corpus de reference stable existe pour les parcours MVP et P1
- les criteres de bascule distinguent clairement ce qui doit etre identique, ce qui peut tolerer une difference et ce qui releve d'un changement produit assume
- les regressions de chiffres, de tri, de filtre ou de navigation sont detectables avant la bascule canonique

## Etape critique 10 detaillee - mesurer la migration comme un produit

Cette section transforme l'etape 10 en pilotage observable. Une migration UI n'est pas seulement un programme technique. C'est un produit temporairement double, avec des usages, des erreurs, des temps de reponse, des abandons et des arbitrages de dette. Sans instrumentation minimale, on pilote au ressenti.

## Objectif operationnel

- disposer de signaux suffisamment fiables pour arbitrer les bascules d'ecrans
- mesurer la qualite technique, l'adoption et la dette restante pendant la cohabitation
- detecter rapidement les regressions backend, front ou auth apres chaque slice
- sortir du pilotage par anecdotes ou impressions visuelles

## Familles de metriques a suivre

| Famille | Metriques minimales | Usage |
| --- | --- | --- |
| Qualite backend | latence p50/p95 par endpoint, taux d'erreur, timeouts, jobs failed | savoir si l'API tient la charge du nouveau front |
| Experience front | temps de chargement route, frequence loading long, erreurs UI, retries | mesurer la perception reelle du shell web |
| Adoption | part d'usage Streamlit vs React par surface, utilisateurs actifs par route, retours en fallback | savoir quand une surface peut devenir canonique |
| Parite | nombre d'ecarts ouverts, regressions detectees, exceptions assumees | suivre la dette de parite restante |
| Dette de migration | nombre de surfaces encore legacy, doubles implementations, routes en preview | piloter la sortie de cohabitation |

## Tableau minimal de suivi

Pour chaque surface migree ou en cours, il faut pouvoir repondre a ces questions sans audit manuel :

- quelle est la route canonique aujourd'hui
- quel est l'etat de cohabitation : preview, canonique, rollback possible, retiree
- quels sont les endpoints critiques et leur p95
- quel est le taux d'erreur recent
- quels scenarios de parite sont verts, jaunes ou rouges
- quelle dette bloque la decommission Streamlit de cette surface

## Regles de pilotage recommandees

1. Une surface React ne devient pas canonique sur la seule base d'un rendu plus moderne.
2. Une surface React ne reste pas indefiniment en preview sans plan de bascule ou de retrait.
3. Une surface Streamlit ne doit pas etre retiree tant que les usages, les erreurs et la parite de la surface React ne sont pas observes sur une periode suffisante.
4. Les exceptions produit ou les ecarts volontaires doivent etre visibles dans le suivi, pas caches dans les changelogs.

## Anti-patterns a eviter

- mesurer uniquement les logs backend sans savoir quel ecran utilisateur est en cause
- ne suivre que des web vitals front sans lien avec la qualite des chiffres metier
- confondre "route disponible" et "route adoptee"
- laisser la dette de cohabitation hors tableau de bord jusqu'au jour de la decommission

## Definition of done pour cette etape 10

L'etape critique 10 est consideree comme couverte si :

- le programme de migration dispose d'un jeu minimal de metriques backend, front, adoption, parite et dette
- chaque surface prioritaire a un statut observable et une decision de bascule pilotable
- les ecarts ou exceptions assumes sont traces comme tels
- la decommission progressive de Streamlit peut etre decidee sur des signaux mesurables, pas sur une impression generale

## Backlog d'implementation executable - slices, schemas, endpoints, stores, recette

Cette section transforme le plan en backlog d'execution. Chaque slice doit produire quelque chose de testable en bout de chaine, pas seulement des endpoints ou des composants isoles.

## Slice 0 - Shell, bootstrap et contrat de filtres

### Backend

- creer l'app FastAPI, la convention /api/v1 et le socle Pydantic des schemas transverses
- implementer GET /bootstrap
- implementer GET /players
- implementer POST /players/{player_slug}/filters/resolve
- definir un middleware request_id + envelope d'erreurs unifie

### Frontend

- creer le shell Vite/React/Router/Query/Zustand
- poser les routes /setup, /settings, /players/:playerSlug/*
- creer useAppShellStore et useGlobalFilterStore
- brancher bootstrap + hydration du joueur courant + language

### Stores / query keys

- useAppShellStore
- useGlobalFilterStore
- ['bootstrap']
- ['players']
- ['filters-resolve', playerSlug, filterContextHash]

### Sortie tangible

- shell React monte
- joueur courant selectionnable
- filtres resolus cote API sans ecran metier complet

### Criteres de recette

- le shell React demarre avec le meme joueur courant que le shell Streamlit quand le contexte est connu
- un changement de joueur recharge proprement le contexte
- le resolveur de filtres renvoie des options et compteurs coherents avec l'etat Streamlit

### Dependances

- aucune

## Slice 1 - Setup / Auth / Settings

### Backend

- implementer /setup/status
- implementer /auth/device-flow/start et /auth/device-flow/{attempt_id}
- implementer /setup/players
- implementer /setup/smoke-test + /jobs/{job_id}
- implementer GET/PATCH /settings et POST /settings/media/reset-index

### Frontend

- ecran setup en plusieurs etapes
- polling du Device Code Flow
- ecran smoke test avec progression temps reel
- page settings avec formulaires groupes par section

### Stores / query keys

- useSetupFlowStore
- useSettingsDraftStore
- ['setup-status']
- ['device-flow', attemptId]
- ['settings']
- ['job', jobId]

### Sortie tangible

- un utilisateur neuf peut configurer l'app sans passer par Streamlit
- un utilisateur configure peut modifier ses settings depuis React

### Criteres de recette

- setup bloque l'acces aux routes protegees tant qu'il n'est pas termine
- settings persistants apres refresh navigateur
- smoke test affichant les memes conclusions que la page Streamlit

### Dependances

- Slice 0

## Slice 2 - Career

### Backend

- implementer /players/{player_slug}/pages/career
- optionnel : decouper /top-matches et /encounters si le payload devient trop gros

### Frontend

- route /players/:playerSlug/career
- cards resume, jauges Plotly, historique XP, section LUSR
- liens vers top matches et match detail

### Stores / query keys

- useCareerPageStore
- ['career', playerSlug]

### Sortie tangible

- premiere page metier React demonstrable en parite forte

### Criteres de recette

- meme rang et memes valeurs XP qu'en Streamlit
- figures chargees sans adaptation metier cote front

### Dependances

- Slices 0 et 1

## Slice 3 - Match History

### Backend

- implementer /pages/match-history/query
- implementer /pages/match-history/export

### Frontend

- table riche AG Grid ou equivalent
- pagination, tri, export, colonnes configurables
- synchronisation avec useGlobalFilterStore

### Stores / query keys

- useMatchHistoryTableStore
- ['match-history', playerSlug, filterContextHash, page, pageSize, sortHash]

### Sortie tangible

- premiere grande table web reactive branchee sur le backend Python

### Criteres de recette

- parite ligne a ligne sur un echantillon critique
- export identique au scope affiche

### Dependances

- Slice 0

## Slice 4 - Explorer

### Backend

- implementer /directory/gamertags/search
- implementer /pages/explorer/matches-query
- implementer /pages/explorer/player-query

### Frontend

- route /players/:playerSlug/explorer
- mode recherche joueur
- mode filtres match
- deep links vers un match ou un gamertag

### Stores / query keys

- useExplorerStore
- ['gamertag-search', q]
- ['explorer', 'matches', ...]
- ['explorer', 'player', ...]

### Sortie tangible

- parcours complet recherche -> resultat -> ouverture match detail

### Criteres de recette

- meme comportement de recherche et de cascade que Streamlit
- navigation correcte vers Match View

### Dependances

- Slices 0 et 3

## Slice 5 - Match View + Last Match

### Backend

- implementer GET /players/{player_slug}/matches/{match_id}
- implementer POST /pages/last-match/resolve

### Frontend

- route /players/:playerSlug/matches/:matchId
- tabs detail match
- route /players/:playerSlug/last-match ou redirection logique depuis shell

### Stores / query keys

- useMatchViewStore
- useLastMatchStore
- ['match-view', playerSlug, matchId]
- ['last-match', playerSlug, filterContextHash]

### Sortie tangible

- detail match complet dans le nouveau front
- vue last match fonctionnelle sur le meme backend

### Criteres de recette

- parite sur score, roster, armes, medailles, citations, navigation prev/next

### Dependances

- Slices 0, 3 et 4

## Slice 6 - Citations

### Backend

- implementer l'endpoint page-oriented citations avec filtered vs full

### Frontend

- page citations avec grille medailles + distribution

### Stores / query keys

- ['citations', playerSlug, filterContextHash]

### Criteres de recette

- memes totaux et memes deltas filtre vs complet

### Dependances

- Slice 0

## Slice 7 - Media V2

### Backend

- exposer l'index media, les enrichissements, les thumbs et les jobs de reset/reindex

### Frontend

- grille media, lightbox, likes locaux, navigation vers match

### Stores / query keys

- useMediaStore
- ['media', playerSlug, mediaFilterHash]

### Criteres de recette

- meme cardinalite, memes groupements, meme navigation vers match

### Dependances

- Slice 5

## Slice 8 - Home Mission Control

### Backend

- endpoint agregateur home branchant summaries sessions, recent matches, recent media, battle pass, challenges, last match

### Frontend

- page home composee branchant les slices deja sorties

### Stores / query keys

- useHomeStore
- ['home', playerSlug]

### Criteres de recette

- meme contenu battle pass/defis et memes highlights

### Dependances

- Slices 2, 5 et 7

## Slice 9 - Timeseries

### Backend

- endpoint page-oriented timeseries + figures Plotly JSON

### Frontend

- page analytics a onglets, sans recalcul metier client

### Stores / query keys

- useTimeseriesStore
- ['timeseries', playerSlug, filterContextHash]

### Criteres de recette

- meme serie et memes agrégats sur un scope donne

### Dependances

- Slice 0

## Slice 10 - Session Compare

### Backend

- endpoint page-oriented compare sessions + selection A/B + contexte historique

### Frontend

- selection A/B, radars, historiques, breakdowns

### Stores / query keys

- useSessionCompareStore
- ['session-compare', playerSlug, filterContextHash, compareStateHash]

### Criteres de recette

- meme choix par defaut et memes deltas qu'en Streamlit

### Dependances

- Slices 0 et 9

## Slice 11 - Teammates

### Backend

- endpoints teammates pour selection, overview, synergy, impact, weapons

### Frontend

- ecran multi-coequipiers avec vues single/duo/trio

### Stores / query keys

- useTeammatesStore
- ['teammates', playerSlug, filterContextHash, teammatesSelectionHash]

### Criteres de recette

- meme set de coequipiers, memes radars, memes stats d'impact et memes enrichissements armes

### Dependances

- Slices 0, 5 et 9

## Slice 12 - Synthesis + Objective Analysis

### Backend

- endpoint synthesis et endpoint objective-analysis

### Frontend

- pages annexes de consolidation analytique

### Stores / query keys

- ['synthesis', playerSlug, filterContextHash, period]
- ['objective-analysis', playerSlug, filterContextHash]

### Criteres de recette

- memes aggregats solo/escouade et memes ratios objectifs

### Dependances

- Slices 0 et 9

## Slice 13 - Decommission Streamlit UI

### Backend

- supprimer les endpoints temporaires devenus redondants si besoin
- verrouiller la compatibilite des contrats utiles restants

### Frontend / produit

- basculer les routes finales
- mettre en place redirects legacy
- retirer les pages Streamlit absorbees ou non exposees

### Criteres de recette

- aucun parcours MVP/P1/P2 ne depend encore d'un rendu Streamlit
- la documentation d'installation et de lancement est a jour

## Stores front minimaux a creer des maintenant

- useAppShellStore : joueur courant, locale, feature flags, etat bootstrap
- useGlobalFilterStore : filter context, options resolues, hash de contexte, dirty state
- useSetupFlowStore : wizard, attempt auth, jobs
- useSettingsDraftStore : brouillon settings + ui prefs locales
- useMatchHistoryTableStore : pagination, tri, colonnes
- useExplorerStore : mode de recherche, input, selectedMatchId, pagination locale
- useMatchViewStore : tab active, sous-selection UI
- useLastMatchStore : index courant et voisins

## Query keys TanStack minimales a normaliser

- ['bootstrap']
- ['players']
- ['filters-resolve', playerSlug, filterContextHash]
- ['settings']
- ['setup-status']
- ['job', jobId]
- ['career', playerSlug]
- ['match-history', playerSlug, filterContextHash, page, pageSize, sortHash]
- ['explorer', 'matches', playerSlug, filterContextHash, localMatchFilterHash, page, sortHash]
- ['explorer', 'player', playerSlug, targetGamertag, filterContextHash]
- ['match-view', playerSlug, matchId]
- ['last-match', playerSlug, filterContextHash]

## Definition of done etendue - contrats API + backlog executable

En plus de la definition de done precedente, cette phase n'est pas terminee tant que :

- les endpoints MVP prioritaires ont chacun un schema d'entree et de sortie explicite
- chaque endpoint critique est rattache a un module Python source de verite
- chaque route React prioritaire a son store et ses query keys cibles
- chaque slice a une sortie tangible et des criteres de recette observables
- les dependances entre slices sont explicites pour eviter les chantiers en parallele mal ordonnes

## Ce que je ferais dans votre cas

1. Conserver le backend Python au début. Changer Streamlit et changer le runtime backend en même temps serait une migration dans la migration.
2. Exposer l’existant via FastAPI en réutilisant les services, repositories, modèles Pydantic et logique Plotly déjà stables.
3. Livrer les graphes en JSON Plotly dans un premier temps, au lieu de les réécrire immédiatement dans une autre lib de chart.
4. Choisir un premier vertical slice simple mais visible : par exemple une page d’accueil, un explorer ou une vue session, avec navigation, filtres, data fetching et rendu complet.
5. Faire tourner ancien et nouveau front en parallèle jusqu’à obtenir la parité fonctionnelle sur les parcours les plus utilisés.
Les causes d’échec les plus fréquentes

1. Vouloir tout réécrire d’un coup.
2. Changer front, backend, auth et design system en même temps.
3. Sous-estimer la logique embarquée dans Streamlit.
4. Refaire trop tôt tous les graphes et composants au lieu de réutiliser l’existant.
5. Ne pas définir de critères de fin de migration.
