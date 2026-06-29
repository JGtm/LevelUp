# LevelUp - Dashboard Halo Infinite

> **Analysez vos stats Halo Infinite match par match, suivez votre progression dans le temps, et comparez vos performances avec votre escouade.**

[![Version](https://img.shields.io/badge/Version-7.0.0-blue.svg)](https://github.com/JGtm/LevelUp/releases/tag/v7.0.0)
[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8.svg)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61DAFB.svg)](https://react.dev/)
[![DuckDB](https://img.shields.io/badge/DuckDB-1.4%2B-FEE14E.svg)](https://duckdb.org/)
[![ECharts](https://img.shields.io/badge/ECharts-5-AA344D.svg)](https://echarts.apache.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Dernières nouveautés

**v7.0 — Nouvelle app React, Mission Control & multi-titres**

Refonte majeure. LevelUp quitte Streamlit pour une app **React 19 + Go API** avec **une UX/UI entièrement repensée** et **une synchronisation désormais automatique**. Près de 400 commits de travail — voici ce que ça change pour vous :

**Une app entièrement repensée — nouvelle UX/UI**
- **Frontend React 19 + Tailwind** — navigation instantanée entre pages, URL partageables pour chaque match/session/joueur, thème clair/sombre, i18n FR/EN complète
- **Design system unifié** — palette, typographie, espacements et composants (cartes, boutons, modales, tooltips, carousels, lightbox) harmonisés sur toute l'app ; fini les pages Streamlit disparates
- **Navigation à deux niveaux** — barre L1 (Accueil / Synthèse / Explorer / Escouade / Communauté / Ascension / Médias / Aide / Paramètres) plus onglets L2 contextuels ; tout est à un clic, plus de sidebar qui défile
- **Interactions modernes** — transitions de page, deep-links (`?session=`, match précédent/suivant), survols, drawers latéraux, retours visuels sur chaque action, UI responsive
- **Backend Go API** — bascule Python → Go pour un serveur plus léger, un démarrage plus rapide et une empreinte mémoire réduite
- **Support multi-titres** — l'app gère désormais plusieurs jeux Halo (Halo 5 : Guardians, Infinite et au-delà) via un **TitleSwitcher** dans la barre de navigation ; commande CLI `levelup add-title` pour enregistrer un nouveau titre
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

**Synthèse — hub d'analytique carrière**
- **Dashboard vue d'ensemble** — KDA, précision, dégâts infligés/reçus, headshots, perfect kills et séries de kills en une grille de cartes, chacune colorée par rapport à votre moyenne historique ; filtres locaux : expérience (classé / non classé), période, saison, playlist et mode
- **Kills par arme** — décomposition complète des kills par arme avec nombre et part en pourcentage
- **Solo vs escouade** — graphique bipolaire comparant vos métriques clés en solo vs avec votre escouade
- **Meilleures semaines** — identifiez vos périodes de jeu les plus performantes d'un coup d'œil
- **Résultats par carte & mode** — répartition V/D/É pour chaque carte et chaque mode que vous avez joués
- **Heatmap d'activité** — fréquence des sessions et densité de kills par jour de la semaine et plage horaire
- **Profil de combat** — évolution du Taux de Conversion Offensif (TC) et de la Résistance Défensive (RD) dans le temps
- **Aperçu des relations** — meilleurs coéquipiers et adversaires les plus fréquents issus de votre historique de matchs

**Médias V2 — likes, notifications Discord, upload**
- **Likes persistants** — aimez vos screenshots et clips directement depuis la grille, état conservé entre les rechargements
- **Groupage intelligent** — par favoris, par session ou par contexte solo/escouade
- **Grille allégée** — thumbnails natifs, lightbox partagée, icônes cœur pour liked / unliked
- **Upload glisser-déposer** — ajoutez vos captures manuelles directement depuis la page Médias
- **Scan non-destructif** — ré-indexation automatique en arrière-plan avec option `--captures-dir` dédiée
- **Notifications Discord pour nouveaux médias** — embed avec GIF ou miniature screenshot à chaque nouvelle capture indexée ; anti-spam (chaque fichier notifié une seule fois) ; toggle `discord_notify_new_media` dans les paramètres
- **Réassociation manuelle avec suggestions de matchs** — modale intégrée qui liste vos matchs dans une fenêtre ±15 / ±60 / ±180 min autour de la capture, avec miniature de la carte, carte · mode · playlist, heure locale + écart, badge de résultat et lobby complet par équipe ; un clic + confirmation pour corriger un média associé au mauvais match

**Page Match dédiée & visualisations enrichies**
- **URL propre par match** — `/players/{gamertag}/matches/{id}`, partageable, avec navigation match précédent/suivant
- **Timeline Tug-of-War** — courbe dynamique des retournements de score entre équipes
- **KD Timeline** — évolution kills/morts par phase avec moyenne mobile
- **Impact Badges** — badges narratifs (Top Killer, Silent Hero, False Brother, Comeback Champion…) calculés par match
- **Panneau Encounters** — liste des joueurs déjà croisés lors de matchs précédents
- **Combat Yield & Perfect Kills** — nouvelles métriques dans la vue match
- **Scoreboard V7** — densité d'info accrue : expected stats, skill rank, média liée, citations
- **Comparaison de sessions** — page A/B dédiée : choisissez deux sessions et comparez KDA, score de performance, Taux de Conversion Offensif / Résistance Défensive, distribution des résultats et playlist dominante côte à côte

**Authentification**
- **Connexion Xbox (standard)** — la façon standard d'utiliser LevelUp : SSO Xbox via navigateur (`/auth/xbox/login` → Microsoft → callback), avec SISU/Proof-of-Possession pour des sessions stables. Le Device Code reste un transport alternatif, et l'URI de redirection est configurable via `LEVELUP_OAUTH_REDIRECT_URI`. Aucune inscription : votre compte Xbox est votre identité.
- **Connexion admin (mot de passe)** — l'administrateur de l'instance dispose d'un compte nom d'utilisateur/mot de passe, créé via le CLI `admin` (`create-admin` / `reset-password`). En mode Xbox, la connexion par mot de passe est réservée aux admins.
- **Connexion locale par joueur (option)** — non standard : un joueur précis peut recevoir un compte nom d'utilisateur/mot de passe (inscription par invitation via `/register`, ou mot de passe opt-in sur un compte SSO existant) pour les déploiements qui le nécessitent. Hors flux standard.

**Achievements Xbox & événements de match**
- **Sync des achievements Xbox** — vos succès Xbox sont récupérés automatiquement depuis l'API Halo à chaque sync
- **Suivi des achievements** — parcourez votre liste complète de succès Xbox sur la page Carrière : filtrez par débloqué / en cours / non commencé, suivez votre Gamerscore (obtenu vs total) et filtrez par jeu pour le support multi-titres
- **Highlight events** — parseur binaire des films de match pour extraire tous les événements majeurs (medals, clutchs, spawns)
- **Backfill weapon kills** — arme utilisée par frag reconstruite depuis le film (POV ~87 %)
- **Badges Comeback pour coéquipiers** — Remontada / Collapse / Contre-Remontada calculés pour vos co-joueurs synchronisés en même temps que vous

**Communauté — Palmarès, Relations & Face-à-face**
- **Season Pass multilingue** — traductions des Battle Pass dans 26 langues, tier images depuis GameCMS
- **Relations** — suivez tous les joueurs que vous avez croisés : stats par joueur, historique de matchs partagés, badges alliance et rivalité, micro-leaderboard de carrière
- **Face-à-face** — page de comparaison 1v1 (ou 1v1v1 en miroir) : opposez deux ou trois joueurs sur les métriques Combat, Précision et Bilan ; badges de rencontre (allié, rival, adversaire coriace) issus de votre historique partagé

**Objectifs & Prestige**
- **Objectifs** — système de défis individuels et d'escouade : fixez des objectifs personnels ou créez des défis d'escouade (collectifs ou compétitifs) sur n'importe quelle métrique Halo avec des fenêtres temporelles, des paliers et des arcs narratifs ; gagnez des Prestige Points (PP) à la complétion ; deux modes d'évaluation (seuil / cumulatif) et deux modes de création (libre / piloté)
- **Leaderboard Prestige** — classement PP dans Palmarès comparant votre score à ceux de votre escouade et de vos relations ; quatre paliers : Normal / Heroic / Legendary / Mythic

**Ascension — suivi de progression & profil de jeu**
- **Dashboard de séries** — séries de victoires, défaites et kills suivies dans le temps, avec vos records personnels all-time mis en avant
- **Records & jalons** — meilleurs scores all-time (meilleur KDA, plus de kills en un match, plus longue série de victoires…) et grille de jalons montrant à quelle distance vous êtes du prochain objectif
- **Radar profil de jeu 6 axes** — forces et faiblesses cartographiées sur six axes : Létalité, Précision, Résilience, Impact Équipe, Survie et Régularité
- **Badge de style** — classification du style de jeu calculée à partir de votre historique (Fragger, Support, Sniper…)
- **Décomposition du rating LUSR** — chaque composante de votre rating visualisée et expliquée pour savoir exactement sur quoi travailler pour progresser
- **Détection de patterns comportementaux** — détection automatique de tilt, fatigue, plateaux d'engagement et plafonds de compétence dans vos récents matchs
- **Patterns contextuels** — comment vos stats évoluent selon le mode, la carte et la composition d'escouade
- **Carte solo vs escouade** — comparaison côte-à-côte de votre style de jeu en solo vs avec des coéquipiers
- **Coach proactif** — moteur d'analyse en tâche de fond qui observe votre progression après chaque sync et envoie des alertes exclusivement positives dans le centre de notifications : nouveaux records personnels, quasi-records, palier LUSR en approche, jalons débloqués, améliorations de stats soutenues et forces contextuelles par carte, mode et type d'escouade

**Centre de notifications in-app**
- **Centre de notifications** — fil par joueur avec badge non-lus dans la barre de navigation, filtres par catégorie, timeline groupée par jour, actions groupées et rafraîchissement live toutes les 60 secondes ; préférences configurables par joueur dans les Paramètres

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

**Accessibilité des couleurs**
- **Palette adaptée aux daltoniens** — une nouvelle palette Okabe-Ito (conçue en 2008, recommandée universellement) est disponible dans Paramètres → Accessibilité ; elle remplace toutes les couleurs de l'app — graphiques, indicateurs de performance, résultats de match, K/D — par des teintes distinguables en cas de deutéranopie, protanopie et tritanopie
- **Aperçu en direct** — la palette bascule instantanément sur toute l'app sans rechargement de page ; un aperçu en pastilles permet de comparer avant de valider
- **Préférence persistante** — votre choix est sauvegardé dans le navigateur et restauré automatiquement à chaque visite

**Historique complet des versions** : [RELEASE_NOTES.md](RELEASE_NOTES.md)

---

## Fonctionnalités

### Suivez votre carrière
- **Historique des rangs** — rating LUSR et CSR par playlist dans le temps, avec votre nom de rang à chaque étape
- **Path to Hero** — graphique de projection montrant à quelle distance vous êtes du rang Hero
- **Cartes KPI de carrière** — 8 cartes en un coup d'œil : matchs joués, temps total, frags, morts, assists, précision, temps en vie, barre V/D/É/DNF — chacune colorée en fonction de votre moyenne all-time
- **Citations** — suivez vos citations Halo avec grilles de médailles et distributions par médaille
- **Progression XP** — courbe d'XP avec overlay de comparaison multi-joueurs
- **Objectifs** — créez des défis individuels ou d'escouade (collectifs ou compétitifs) sur n'importe quelle métrique Halo avec des fenêtres configurables, des paliers (Normal / Heroic / Legendary / Mythic) et des arcs narratifs ; gagnez des Prestige Points (PP) à la complétion
- **Prestige** — leaderboard PP dans Palmarès vous classant parmi votre escouade et vos relations ; quatre paliers avec badges colorés

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
- **Réassociation manuelle** — corrigez en un clic un clip mal associé : un sélecteur intégré suggère les matchs autour de l'horodatage de capture (±15 / ±60 / ±180 min) avec miniatures des cartes, résultat et lobby complet

### Notifications & Configuration
- **Centre de notifications in-app** — fil par joueur avec badge non-lus, filtres par catégorie, timeline groupée par jour et actions groupées ; rafraîchissement live toutes les 60 secondes ; préférences par joueur
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

Si vous ne pouvez pas utiliser le wizard interactif (setup serveur/headless par exemple),
ouvrez la page de connexion à `http://localhost:5173/auth/xbox/login` depuis n'importe quel
navigateur sur le même réseau et suivez le Device Code Flow. Alternativement, configurez
un URI de redirect via `LEVELUP_OAUTH_REDIRECT_URI` pour un flux entièrement navigateur.

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
| [testing.md](../testing.md) | Stratégie de tests Go (CGO, ratchet de couverture) |
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
| **Go 1.26+** | Backend API |
| **React 19 + Vite** | UI frontend |
| **TanStack Query / Router / Table** | Données, routing, tableaux |
| **ECharts 5** | Graphiques interactifs |
| **DuckDB 1.4+** | Moteur de requêtes OLAP |
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
