# LevelUp - Dashboard Halo Infinite

> **Analysez vos stats Halo Infinite match par match, suivez votre progression dans le temps, et comparez vos performances avec votre escouade.**

[![Version](https://img.shields.io/badge/Version-7.0.0-blue.svg)](https://github.com/JGtm/LevelUp/releases/tag/v7.0.0)
[![Python 3.12+](https://img.shields.io/badge/Python-3.12%2B-blue.svg)](https://www.python.org/downloads/)
[![React](https://img.shields.io/badge/React-19-61DAFB.svg)](https://react.dev/)
[![FastAPI](https://img.shields.io/badge/FastAPI-0.110%2B-009688.svg)](https://fastapi.tiangolo.com/)
[![DuckDB](https://img.shields.io/badge/DuckDB-1.4%2B-FEE14E.svg)](https://duckdb.org/)
[![Polars](https://img.shields.io/badge/Polars-1.38%2B-blue.svg)](https://pola.rs/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Dernières nouveautés

**v7.0 — Nouvelle app React, Mission Control & multi-titres**

Refonte majeure. LevelUp quitte Streamlit pour une app **React 19 + Go API** avec **une UX/UI entièrement repensée** et **une synchronisation désormais automatique**. Près de 400 commits de travail — voici ce que ça change pour vous :

**Une app entièrement repensée — nouvelle UX/UI**
- **Frontend React 19 + Tailwind** — navigation instantanée entre pages, URL partageables pour chaque match/session/joueur, thème clair/sombre, i18n FR/EN complète
- **Design system unifié** — palette, typographie, espacements et composants (cartes, boutons, modales, tooltips, carousels, lightbox) harmonisés sur toute l'app ; fini les pages Streamlit disparates
- **Navigation à deux niveaux** — barre L1 (Accueil / Synthèse / Explorer / Escouade / Palmarès / Médias / Aide / Paramètres) plus onglets L2 contextuels ; tout est à un clic, plus de sidebar qui défile
- **Interactions modernes** — transitions de page, deep-links (`?session=`, match précédent/suivant), survols, drawers latéraux, retours visuels sur chaque action, UI responsive
- **Backend Go API** — bascule Python → Go pour un serveur plus léger, un démarrage plus rapide et une empreinte mémoire réduite
- **Support multi-titres** — l'app gère désormais plusieurs jeux Halo (Infinite et au-delà) via un **TitleSwitcher** dans la barre de navigation ; commande CLI `levelup add-title` pour enregistrer un nouveau titre
- **Page d'aide intégrée** — Notes de version (changelog consultable dans l'app) et Glossaire des termes Halo, avec cache local 24 h pour lecture hors-ligne

**Accueil « Mission Control » — entièrement redessiné**
- **Hero banner multi-titres** — bannière dynamique par jeu avec artwork dédié
- **Panneau Battle Pass live** — carte dédiée avec artwork d'opération, progression de tier et récompense à venir, rafraîchie à chaque visite
- **Défis actifs restaurés** — cartes de challenges Mission Control avec expiry du deck, badge, titre/description localisés et votre progression `x/y` en temps réel
- **Catalogue de défis multilingue** — titres et descriptions stockés dans **26 langues** (BCP-47), avec fallback `en-US` si la langue demandée manque
- **Historisation propre** — vos défis sont snapshotés dans votre DB joueur (`challenge_snapshots`), les définitions partagées vivent dans `metadata.duckdb`
- **Live-first, failsafe** — si la base metadata est verrouillée, la Home affiche quand même les défis en live et skip la persistance proprement (pas de blocage)
- **Tuiles de match enrichies** — KDA, escouade, rang, headshots, citations, score de perf et scores des deux équipes directement sur chaque tuile
- **Carousel de sessions** — FDA coloré, playlist et mode dominants, tooltip au survol
- **Onglet médias aimés** et carousel des derniers matchs sur la Home
- **Page Synthèse** — nouvelle section L1 avec highlights hebdomadaires, rivalités top et stats synthétiques de carrière

**Médias V2 — likes, notifications Discord, upload**
- **Likes persistants** — aimez vos screenshots et clips directement depuis la grille, état conservé entre les rechargements
- **Groupage intelligent** — par favoris, par session ou par contexte solo/escouade
- **Grille allégée** — thumbnails natifs, lightbox partagée, icônes cœur pour liked / unliked
- **Upload glisser-déposer** — ajoutez vos captures manuelles directement depuis la page Médias
- **Scan non-destructif** — ré-indexation automatique en arrière-plan avec option `--captures-dir` dédiée
- **Notifications Discord pour nouveaux médias** — embed avec GIF ou miniature screenshot à chaque nouvelle capture indexée ; anti-spam (chaque fichier notifié une seule fois) ; toggle `discord_notify_new_media` dans les paramètres

**Page Match dédiée & visualisations enrichies**
- **URL propre par match** — `/players/{gamertag}/matches/{id}`, partageable, avec navigation match précédent/suivant
- **Timeline Tug-of-War** — courbe dynamique des retournements de score entre équipes
- **KD Timeline** — évolution kills/morts par phase avec moyenne mobile
- **Impact Badges** — badges narratifs (Top Killer, Silent Hero, False Brother, Comeback Champion…) calculés par match
- **Panneau Encounters** — liste des joueurs déjà croisés lors de matchs précédents
- **Combat Yield & Perfect Kills** — nouvelles métriques dans la vue match
- **Scoreboard V7** — densité d'info accrue : expected stats, skill rank, média liée, citations

**Authentification repensée**
- **Provider SISU/PoP** — nouvelle authentification Xbox avec Proof-of-Possession pour des sessions plus stables et moins de reconnexions
- **Auth locale** — mode nom d'utilisateur/mot de passe pour déploiements mono-utilisateur / LAN

**Achievements Xbox & événements de match**
- **Sync des achievements Xbox** — vos succès Xbox sont récupérés automatiquement depuis l'API Halo à chaque sync
- **Highlight events** — parseur binaire des films de match pour extraire tous les événements majeurs (medals, clutchs, spawns)
- **Backfill weapon kills** — arme utilisée par frag reconstruite depuis le film (POV ~87 %)
- **Badges Comeback pour coéquipiers** — Remontada / Collapse / Contre-Remontada calculés pour vos co-joueurs synchronisés en même temps que vous

**Palmarès & Season Pass**
- **Season Pass multilingue** — traductions des Battle Pass dans 26 langues, tier images depuis GameCMS
- **Relations / Leaderboard** — nouvelles pages palmarès : joueurs croisés, stats par joueur, micro-leaderboard de carrière
- **Compare drawer** — UI de comparaison session/joueur redessinée

**Synchronisation automatique & présence temps réel**
- **Sync 100 % automatique** — finis les `python scripts/sync.py` à lancer à la main : l'app synchronise vos matchs toute seule en arrière-plan, en continu, dès qu'une nouvelle partie est jouée
- **Déclenchement immédiat fin de partie** — dès qu'un joueur termine un match, le watcher récupère les stats sans attendre le prochain tick
- **Présence RTA Xbox + polling Steam** — détection en ligne temps réel pour savoir qui joue et synchroniser au bon moment
- **Scheduler intelligent** — cadence de sync adaptative selon l'activité des joueurs ; pas de requêtes inutiles quand personne ne joue
- **Rafraîchissement autonome des tokens** — plus d'interruptions : les tokens Halo se renouvellent tout seuls en tâche de fond
- **Reconnexion proactive** — gestion du status=3 avec refresh XSTS à la demande, reconnexion automatique au démarrage

**Paramètres & admin**
- **Auto-save des paramètres** — les réglages se sauvegardent immédiatement avec indicateur visuel éphémère
- **Page admin** — UI de supervision (auth provider, état des jobs, privacy)
- **Préférences navigateur** — joueur sélectionné, langue et filtres mémorisés entre sessions
- **API endpoints configurables** — Halo Stats, SPNKr, CMS… tous paramétrables depuis les Paramètres

**Assets & cartes**
- **Cache-aside des images de cartes** — artworks de maps téléchargés et mis en cache local, plus aucune requête externe répétée
- **CLI `populate-assets`** — commande Go pour pré-télécharger tous les assets (cartes, medals, tiers Battle Pass) avant usage hors-ligne

**v6.5 — Heatmap escouade & paramètres fiabilisés**
- **Heatmap d'intensité par joueur** (Teammates) — nouvelle visualisation : heatmap match × phase (début/milieu/fin) pour chaque membre de l'escouade. Voir qui frappe tôt, qui accélère en fin de match. Toggle pour afficher tous ensemble ou joueur par joueur
- **Notifications Discord séparées** — les alertes sync et backfill ont maintenant chacune leur propre toggle ; désactiver l'une n'affecte pas l'autre
- **Paramètres plus robustes** — les réglages sont écrits de façon sécurisée (écriture atomique + sauvegarde automatique) ; le fichier ne peut plus se corrompre en cas de crash ou d'arrêt forcé
- **Corrections** — les records (meilleure performance) ne s'affichent plus par défaut quand ils avaient été désactivés

**v6.4 — Filtres médias, CSR escouade & aides à la lecture**
- **Filtres de la médiathèque** — filtrez vos clips et screenshots par propriétaire (mes captures / coéquipiers / sans match associé), carte, mode, résultat (victoire / défaite…) et contexte solo vs escouade. Triez par date, carte, mode ou résultat en un clic
- **Aides à la lecture** — une case à cocher dans la barre latérale affiche ou masque les ~45 cartouches d'explication sur chaque page ; désactivable pour une interface plus épurée
- **Récapitulatif carrière redessiné** — 8 cartes compactes côte à côte : Matchs, Durée totale, Frags, Morts, Assists, Précision, Temps en vie, Résultats. Chaque carte compare votre valeur à votre moyenne historique (code couleur vert/or/rouge à ±8 %)
- **Win/Loss intégré dans Timeseries** — la page Win/Loss devient un onglet dans Timeseries ; onglets renommés : Résumé · Cartes & Modes · Progression · Avancé
- **CSR des coéquipiers automatique** — lors d'un sync sur un match classé, le rang de tous les co-joueurs enregistrés est récupéré et distribué automatiquement — plus besoin que chacun synchronise son propre compte
- **Badges Remontada/Collapse pour les coéquipiers** — calculés pour vos co-joueurs synchronisés en même temps que vous
- **Panneau de légende fixe** (Teammates) — un panneau flottant affiche la couleur de chaque membre de l'escouade pendant tout votre défilement dans la section squad
- **Préférences mémorisées entre sessions** — le joueur sélectionné et la langue sont mémorisés dans le navigateur ; les filtres survivent aux mises à jour

**v6.3 — Noms localisés, records escouade & détails médailles**
- **Cartes et modes dans votre langue** — noms de cartes, playlists et modes de jeu en français (ou anglais) sur toutes les pages : filtres, tableaux, graphiques et histogramme de winrate
- **Description des médailles au survol** — survolez une médaille dans le scoreboard ou la section Citations pour lire sa description
- **Records all-time de l'escouade** — la page Teammates affiche les meilleurs records en carrière pour chaque membre (K/D, kills, séries…) avec annotations colorées par joueur et vue détaillée par carte
- **Badge Top Killer** — affiché sur la timeline Impact pour le premier joueur à atteindre 10 kills dans le match
- **Histogramme temps de premier kill/mort revu** — graphique en papillon miroir avec intervalles de 15 secondes et temps réel de jeu (décompte pré-partie soustrait)
- **Dernier match amélioré** — carte et mode fusionnés ; les cartes MMR, Kills et Deaths affichent aussi le score équipe adverse avec un écart coloré ; le score de performance apparaît directement à côté du rating
- **Médailles et citations en grille 4 colonnes** — scoreboard plus lisible ; armes en grille 2 colonnes avec bonne proportion de miniatures
- **Recherche partielle par ID de match** (Explorer) — tapez 3+ caractères d'un ID de match pour filtrer instantanément la liste
- **Histogramme de cadence de kills** — nouveau graphique (onglet Combat) : kills par intervalles de 15 secondes pour vous et les ennemis, avec moyenne mobile par équipe
- **Heatmap d'intensité de match** — visualise la densité de kills par phase sur l'ensemble de vos matchs
- **Auto-indexation de la médiathèque** — vos clips sont re-scannés automatiquement en arrière-plan après chaque sync
- **Corrections** — citation Spartan Carnage corrigée ; noms corrects dans les notifications Discord ; calendar de filtres avec navigation libre entre les années

**v6.2 — Badges Comeback & vue escouade unifiée**
- **Badges Remontada / Collapse / Contre-Remontada** — l'app détecte les scénarios de comeback dans votre historique : *Remontada* (vous étiez perdants à mi-match et vous avez gagné), *Collapse* (vous meniez et avez perdu), *Contre-Remontada* (vous avez stoppé le comeback adverse)
- **Vue escouade unifiée** — les vues 1-vs-1 et escouade fusionnent ; vous obtenez les mêmes graphiques riches pour 1, 2 ou 3 amis
- **Graphique Kills ↑ / Morts ↓** — kills et morts fusionnés en un seul graphique miroir par membre, pour comparer les arcs K/D d'un coup d'œil
- **Noms de modes cohérents** — les libellés de modes de jeu sont désormais homogènes sur toutes les pages et tous les graphiques

**v6.1 — Sync plus rapide, bugs corrigés**
- **Sync ~30–40 % plus rapide** — chaque synchronisation se termine nettement plus vite
- **Noms de rangs corrects** — le rang affiché correspond désormais au vrai palier (ex. « Lance Corporal Diamond 1 »)
- Corrections : scores de performance et vues matérialisées toujours à jour après sync

---

## Fonctionnalités

### Suivez votre carrière
- **Historique des rangs** — rating LUSR et CSR par playlist dans le temps, avec votre nom de rang à chaque étape
- **Path to Hero** — graphique de projection montrant à quelle distance vous êtes du rang Hero
- **Cartes KPI de carrière** — 8 cartes en un coup d'œil : matchs joués, temps total, frags, morts, assists, précision, temps en vie, barre V/D/É/DNF — chacune colorée en fonction de votre moyenne all-time
- **Citations** — suivez vos citations Halo avec grilles de médailles et distributions par médaille
- **Progression XP** — courbe d'XP avec overlay de comparaison multi-joueurs

### Analysez vos matchs
- **Explorer** — parcourez tous vos matchs avec filtres en cascade (carte, mode, playlist, résultat, date, session), recherche partielle par ID de match et badges de rencontre
- **Dernier Match** — scoreboard complet avec K/D, médailles, armes, score de performance, impact badges, et panneau d'historique des rencontres pour les adversaires récurrents
- **Cadence de kills** — kills par intervalles de 15 secondes pour vous et l'équipe adverse, avec overlay de moyenne mobile — voyez exactement quand le rythme a basculé
- **Heatmap d'intensité de match** — densité de kills par phase de jeu (début/milieu/fin) sur l'ensemble de vos matchs d'un coup d'œil
- **Badges Comeback** — *Remontada* (vous étiez perdants et avez renversé), *Collapse* (vous meniez et avez tout gâché), *Contre-Remontada* (vous avez stoppé le comeback adverse)
- **Comparaison de sessions** — analyse côte-à-côte de deux sessions de jeu
- **Heatmap d'activité** — win rate et activité par jour de la semaine et plage horaire

### Escouade & Coéquipiers
- **Vue escouade unifiée** — mêmes graphiques riches pour 1, 2 ou 3 amis ; fonctionne pour toutes les tailles d'escouade
- **Heatmap d'intensité par joueur** — voyez le profil de kills de chaque membre d'escouade par phase de jeu sur les matchs partagés
- **Records d'escouade** — meilleurs scores de carrière pour chaque membre (K/D, kills, séries…) avec détails par carte
- **Radar de synergie** — stats par minute et complémentarité dans votre escouade
- **Cadence de kills par joueur** — tempo de kills synchronisé sur les matchs partagés
- **Timeline d'impact** — badges narratifs (Top Killer, Silent Hero, False Brother…) par match

### Clips & Médias
- **Médiathèque** — parcourez screenshots et clips vidéo liés à leur match ; filtrez par propriétaire, carte, mode, résultat ou contexte solo/escouade
- **Auto-indexation** — clips re-scannés automatiquement toutes les quelques heures et après chaque sync

### Notifications & Configuration
- **Alertes Discord** — notifications configurables après sync et après backfill, indépendamment
- **Setup en un clic** — connexion Xbox Device Code (`xbox.com/activate`) avec provisionnement joueur automatique ; pas de compte Azure requis

---

## Captures d'écran

### Vue d'ensemble

![Dashboard principal](../screenshots/main.png)

*Dashboard principal : navigation multi-pages et graphiques interactifs en temps réel.*

![Barre latérale, Temps au premier kill & Performance](../screenshots/Sidebar-first-kill-performance.png)

*Filtres avancés (type, playlist, mode, carte, session/période), distribution Time-to-First-Kill vs First Death, et score de performance par match.*

---

### Performance & Combat

| KDA | Performance cumulée & tendance |
|:-:|:-:|
| ![KDA](../screenshots/kda.png) | ![Performance cumulée & tendance](../screenshots/cumulative-perf.png) |

![Durée de vie moyenne & Skills de combat](../screenshots/avg-lifespan-perfect-kills.png)


*Ratio K/D avec tendance, score de performance cumulé, durée de vie moyenne et skills de combat.*

---

### Distributions & Corrélations

| Distributions | Corrélations |
|:-:|:-:|
| ![Distributions](../screenshots/distributions.png) | ![Corrélations](../screenshots/correlations.png) |

*Histogrammes de précision/kills/scores avec moyennes et médianes — scatter plots (temps en vie vs kills, etc.).*

---

### Activité par jour & heure

![Heatmap Top Semaine](../screenshots/heatmap-top-week.png)

*Win rate et heatmap d'activité par jour de la semaine et plage horaire.*

---

### Détails du dernier match

| Dernier match | Scoreboard |
|:-:|:-:|
| ![Résumé](../screenshots/last-match.png) | ![Scoreboard Citations](../screenshots/scoreboard.png) |
| Impact & Dominance | Antagonistes |
| ![Impact & Dominance](../screenshots/impact-dominance.png) | ![Antagonistes](../screenshots/antagonist.png) |

*Scoreboard complet pour votre dernière partie (recherchable par ID de match) — et vos rivaux les plus redoutables, MVP/LVP, scoreboard, grille de citations (inspirée de Halo 5) et distributions de médailles.*

---

### Sessions d'escouade & Coéquipiers

| Vue d'ensemble escouade | Stats de session |
|:-:|:-:|
| ![Historique session](../screenshots/history.png) | ![Complémentarité escouade](../screenshots/per-minute-complementarity.png) |
| **Performance coéquipiers** | **Classement escouade** |
| ![Performance escouade](../screenshots/performance-spree.png) | ![Classement escouade](../screenshots/teammate-heatmap.png) |

*Filtrez vos sessions par escouade : comparez vos stats quand vous jouez avec vos amis et voyez comment vous et vos coéquipiers performez sur les matchs partagés.*

---

### Progression de carrière, Rangs & Path to Hero

| Carrière | Rangs (LUSR/CSR) |
|:-:|:-:|
| ![Carrière](../screenshots/career.png) | ![Rangs](../screenshots/LUSRs.png) |
| ![Path to Hero](../screenshots/path-hero.png) | ![Matchs mémorables](../screenshots/memorable-matches.png) |

*Historique des rangs, progression vers Hero, LUSR/CSR par groupe de playlist*

---

### Explorer & Historique des rencontres

| Explorer | Historique des rencontres |
|:-:|:-:|
| ![Explorer](../screenshots/explorer.png) | ![Historique des rencontres](../screenshots/encounters.png) |

*Parcourez et filtrez tous vos matchs en détail avec l'Explorer, y compris la recherche par joueur — suivez les adversaires récurrents et les patterns de rencontres cross-match avec la vue Historique des rencontres.*

---

### Médiathèque & Citations

| Médiathèque | Citations |
|:-:|:-:|
| ![Médiathèque](../screenshots/media-library.png) | ![citations](../screenshots/commendations.png) |

*Parcourez et recherchez vos clips et screenshots liés à leurs matchs (toujours en bêta) — suivez vos citations avec grilles de médailles et distributions.*

---

## Démarrage rapide

**Prérequis** : Go 1.26+, Node.js + npm, GNU Make, et Air pour le hot-reload Go.

```bash
git clone https://github.com/JGtm/LevelUp.git
cd LevelUp
cd apps/web && npm install && cd ../..
go install github.com/air-verse/air@latest
make dev
```

Ouvrez http://localhost:5173 dans votre navigateur, puis suivez le wizard intégré.

Variantes utiles :

```bash
make go-api-dev
make web
```

**Doc détaillée** : [INSTALL.md](INSTALL.md)

**README anglais** : [../../README.md](../../README.md)

---

## Configuration

**v6 — Zéro configuration.** LevelUp embarque son propre client ID Azure.
Lancez simplement l'app, entrez votre gamertag, et authentifiez-vous via Device Code Flow
(`https://xbox.com/activate`). Pas de fichier `.env.local` ni de compte Azure requis.

### Refresh token (avancé / headless)

Si vous ne pouvez pas utiliser le wizard interactif (setup serveur/headless par exemple) :

```bash
python scripts/spnkr_get_refresh_token.py --device-code
```

La commande affiche un code à saisir sur `https://xbox.com/activate`, puis sauvegarde
automatiquement le token dans `.env.local`.

### Note pour les forks / développeurs

Le `LEVELUP_CLIENT_ID` embarqué est une Azure App Registration liée à ce projet.
**Si vous forkez LevelUp**, créez votre propre Azure App Registration gratuite
(voir [CONFIGURATION.md](CONFIGURATION.md)) et définissez :

```env
# .env.local
SPNKR_AZURE_CLIENT_ID=votre_propre_client_id
```

Cette variable d'environnement a priorité sur l'ID embarqué.

**Référence complète de configuration** : [CONFIGURATION.md](CONFIGURATION.md)

---

## Documentation

| Document | Contenu |
|----------|---------|
| [INSTALL.md](INSTALL.md) | Guide d'installation détaillé |
| [CONFIGURATION.md](CONFIGURATION.md) | Configuration des tokens et profils |
| [ARCHITECTURE_V6.md](ARCHITECTURE_V6.md) | Architecture v6 (shared matches + i18n assets) |
| [SYNC_GUIDE.md](SYNC_GUIDE.md) | Guide de synchronisation |
| [BACKUP_RESTORE.md](BACKUP_RESTORE.md) | Backup et restauration |
| [TESTING_V5.md](TESTING_V5.md) | Stratégie de tests v5 |
| [FAQ.md](FAQ.md) | Questions fréquentes |

Docs anglaises : [../](../)

Docs archivées (non traduites) : [../archive/](../archive/)

---

## Contribution

Les contributions sont les bienvenues ! Voir [CONTRIBUTING.md](../CONTRIBUTING.md) pour les guidelines.

---

## Stack technique

| Technologie | Usage |
|-------------|-------|
| **Python 3.12+** | Langage principal |
| **React 19 + Vite** | UI frontend |
| **FastAPI** | Backend API REST |
| **DuckDB 1.4** | Moteur de requêtes OLAP |
| **Polars 1.38** | DataFrames haute performance |
| **PyArrow 23** | Interopérabilité données |
| **Pydantic v2** | Validation des données |
| **Plotly** | Graphiques interactifs |
| **SPNKr** | API Halo Infinite |

---

## Limitations connues

- **API Halo** : dépend de SPNKr — certains endpoints peuvent être instables ou rate-limités. Les kills par arme sont extraits des données binaires des films de match (SPNKr), pas de l'API stats ; couverture POV ~87,5 %.

---

## Licence

Ce projet est sous licence MIT. Voir [LICENSE](../../LICENSE) pour plus de détails.

---

## Remerciements

- **Andy Curtis** ([acurtis166](https://github.com/acurtis166)) pour [SPNKr](https://github.com/acurtis166/SPNKr)
- **Den Delimarsky** ([dend](https://github.com/dend)) pour [Grunt](https://github.com/dend/grunt) et [OpenSpartan](https://github.com/OpenSpartan)

Voir aussi [ACKNOWLEDGMENTS.md](../ACKNOWLEDGMENTS.md).

---

**Fait avec passion pour la communauté Halo**
