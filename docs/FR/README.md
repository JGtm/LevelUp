# LevelUp - Dashboard Halo

> **Analysez vos stats de Halo 5: Guardians et de Halo Infinite match par match, suivez votre progression dans le temps, et comparez vos performances avec votre escouade.**

[![Version](https://img.shields.io/badge/Version-7.2.0-blue.svg)](https://github.com/JGtm/LevelUp/releases/tag/v7.2.0)
[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8.svg)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61DAFB.svg)](https://react.dev/)
[![DuckDB](https://img.shields.io/badge/DuckDB-1.4%2B-FEE14E.svg)](https://duckdb.org/)
[![ECharts](https://img.shields.io/badge/ECharts-5-AA344D.svg)](https://echarts.apache.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Dernières nouveautés

**v7.3 — Prolongation, premier frag / première mort & la page Réalisations réparée**

Une version centrée sur la lecture d'un match : un repère de prolongation sur tous les matchs allés au-delà du temps réglementaire, un nouveau graphe premier frag / première mort sur trois pages, et une série de réparations sur Halo 5, sur la bascule de langue et sur la démo.

**Prolongation**
- **Un badge quand le match est allé au-delà du temps réglementaire** — dans l'en-tête du match et en pastille dans l'Explorateur, avec le temps de jeu supplémentaire
- **Tout votre historique d'un coup** — le repère est calculé à la lecture : tous vos matchs passés sont couverts sans attendre une resynchronisation ; Halo 5 en reste volontairement dépourvu tant que ses temps réglementaires ne sont pas déclarés

**Premier frag / première mort**
- **Un nouveau graphe qui remplace les anciens histogrammes** — une bande par joueur, les premiers frags au-dessus, les premières morts en dessous, avec la médiane de chacun et la fenêtre d'avance entre les deux
- **Sur trois pages** — Escouade (onglet « Dynamique »), Séries temporelles (onglet « Progression ») et détail de session, volet de comparaison compris

**Halo 5**
- **Page Réalisations réparée** — la page renvoyait une erreur au lieu de vos jalons, et le titre de section reste désormais en place, que la grille charge, qu'elle soit vide ou en erreur

**Lecture dans votre langue**
- **Les citations suivent la bascule de langue** — leurs libellés restaient dans la langue précédente jusqu'au rechargement de la page, et « Tirs à la tête » s'affichait sous son nom de champ brut

**Vue de match & démo**
- **Une statistique d'objectif manquante ne casse plus le tableau des scores** — le match ne se lit plus comme incomplet ; seule la section Objectifs reste vide
- **Une démo plus complète** — les campagnes d'amélioration et les statistiques d'objectif sont enfin visibles, et les identifiants de joueur sont anonymisés eux aussi

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
