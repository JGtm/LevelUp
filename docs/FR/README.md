# LevelUp - Dashboard Halo

> **Analysez vos stats de Halo 5: Guardians et de Halo Infinite match par match, suivez votre progression dans le temps, et comparez vos performances avec votre escouade.**

[![Version](https://img.shields.io/badge/Version-7.3.2-blue.svg)](https://github.com/JGtm/LevelUp/releases/tag/v7.3.2)
[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8.svg)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61DAFB.svg)](https://react.dev/)
[![DuckDB](https://img.shields.io/badge/DuckDB-1.5%2B-FEE14E.svg)](https://duckdb.org/)
[![ECharts](https://img.shields.io/badge/ECharts-5-AA344D.svg)](https://echarts.apache.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Dernières nouveautés

**v7.5 — Le rejeu 2D, le film Theater décodé & un onglet Tactique**

La plus grosse version à ce jour. Halo enregistre un film de chaque match ; jusqu'ici l'application ne l'avait jamais ouvert. C'est fait — un match se revoit désormais vu du dessus, seconde par seconde, avec tout ce que le film sait écrit autour : qui a tué qui et avec quoi, à quelle distance, qui portait le drapeau, qui a raflé l'arme de puissance, qui est mort avec un équipement qu'il n'a jamais utilisé.

**Rejeu 2D**
- **Chaque match, revu du dessus** — tous les joueurs se déplacent sur le vrai fond de la carte, sous leur nom, avec une barre de lecture, quatre pistes sur la frise (Toi, Alliés, Dominance, Médias), le réglage de vitesse et les sauts de 10 secondes
- **Un fil des éliminations qui suit le curseur** — tueur, arme, victime, médaille et assistance avec sa part de dégâts ; une mort sans tueur crédité le dit, au lieu d'inventer un coupable
- **Des calques à cocher** — visée, traînées, effets de tirs et de mort, carte de chaleur, emplacements d'armes, armes au sol, équipements posés, véhicules, zones nommées, et les objectifs vivants (drapeau, crâne, colline, bastions, bombe, couronne du VIP)
- **Le son du jeu**, coupé par défaut et filtrable par catégorie, plus une capture PNG et un enregistrement vidéo du rejeu avec sa bande-son

**Ce que le film sait, et que l'API n'a jamais dit**
- **L'arme de chaque élimination**, y compris les morts sans arme à feu (répulseur, chute, environnement), et l'endroit de la carte où l'on tue et où l'on meurt
- **La portée des engagements** — basse (p10), médiane et haute (p90) par arme, avec le dénivelé signé
- **Le niveau des armes** — les prises de socle ventilées en armes de base, de terrain et de puissance ; ce que vaut un emplacement vient de la carte, jamais du nom de l'arme
- **Les prises nettes de drapeau**, les statistiques d'Assaut, et l'équipement lu comme utilisé, gardé ou lâché en mourant
- **Les manches comptent comme des manches** — sur les modes à manches, le score affiché est le nombre de manches gagnées et perdues, parce que le score en points de l'API peut donner l'avantage au camp qui a perdu

**Onglet Tactique**
- **Les cartes que tu joues**, leur bilan, et une vue d'analyse sur le plan de la carte : où tu passes ton temps, où tu meurs, où tu tues, où tu meurs isolé, où les victoires et les défaites se séparent, et les routes que tu empruntes
- **Une cellule ouvre le rejeu à l'instant exact** où ça s'est joué

**Escouade, sessions et fiche de match**
- **« Les formes retenues »** — six lectures d'une escouade sur dix-neuf cartes, et le bloc équipement utilisé / gardé / gâché sur la Synthèse, l'Escouade et les Sessions
- **La cadence par match** partout, la distance par arme, la répartition des frags en deux niveaux, et une courbe de score dans le temps qui respecte le mode

**Cartes, médias et réparations**
- **109 fonds de carte** avec les zones Forge nommées à 100 %, les repères officiels, les images de médailles rafraîchies depuis le catalogue du jeu, les mentions J'aime par spectateur
- **La page Classement mondial est réparée** et ne peut plus se dégrader en silence ; une note LUSR d'arène Halo 5 corrompue est réparée à la source

Tout ce qui se lit dans un film est réservé à Halo Infinite — Halo 5 garde ses propres pages et n'emprunte jamais les données d'un autre titre.

## Fonctionnalités

> Tout ce qui se lit dans le film Theater — le rejeu 2D, l'onglet Tactique, l'arme de chaque frag, la portée des engagements, le niveau des armes, les usages d'équipement, les prises nettes de drapeau, les statistiques d'Assaut — est réservé à **Halo Infinite** et gardé par des capabilities fines. Halo 5 garde ses propres pages et n'emprunte jamais les données d'un autre titre.

### Revoir son match — le rejeu 2D
- **Tout le match, vu du dessus** — chaque joueur se déplace sur le vrai fond de la carte, sous son nom, du coup d'envoi à la fin, décodé depuis le film Theater que le jeu enregistre
- **Une barre de lecture à quatre pistes** — Toi, Alliés, Dominance et Médias sur une même frise, avec les pastilles de manche, les messages inter-manches, la lecture/pause, les sauts de 10 secondes, un menu de vitesse et des raccourcis clavier
- **Le fil des éliminations calé sur le curseur** — tueur, icône d'arme, victime, médaille, et l'assistance avec sa part de dégâts ; une mort sans tueur crédité le dit au lieu d'inventer un coupable, et dit *de quoi* on est mort
- **Une fiche par joueur** — armes portées, grenades, bouclier, camouflage et surbouclier en cours, grappin, temps passé mort, et un filigrane tant que le joueur porte un objectif
- **25 calques à cocher** — visée, traînées, éclairs de bouche, effets de tirs et de mort, vols de grenade et explosions, étoiles de corps à corps, carte de chaleur (temps passé ou éliminations), emplacements d'armes et socles de bonus, armes laissées au sol avec leurs munitions exactes, équipements posés (mur, faille, capteur, translocateur…), objets lâchés, véhicules, zones nommées, chevrons hors cadre
- **Les objectifs vivants** — portages et zone de retour du drapeau, crâne d'Oddball, jauge de propriété de colline en Roi de la colline, bastions A/B/C, bombe d'Assaut avec son décompte de mèche, couronne du VIP
- **Un bandeau de score** — le score de la manche en cours, la cible de victoire, et la courbe de score du mode
- **Le son du jeu** — 177 sons tirés du jeu (armes, grenades, corps à corps, équipements, objectifs, annonceur, fanfare de fin), coupés par défaut, filtrables par catégorie
- **Zoom, glissement et cadrage** — paliers, molette, clavier ou croix directionnelle ; la toile prend tout le bloc et le zoom ne recadre jamais la carte
- **Capture et enregistrement** — une image PNG en un clic, ou un export vidéo complet du rejeu avec sa bande-son mixée, encodé hors du temps réel
- **On y entre de partout** — la fiche du match, les tuiles de l'accueil, l'Explorateur, et une cellule de la Tactique qui ouvre le rejeu à la seconde exacte

### Lire un match
- **Tableau des scores complet** — F/M, médailles, armes, note de performance, repères d'impact, et un panneau d'historique des rencontres pour les adversaires récurrents
- **L'arme de chaque élimination** — lue dans la source de dégât du film au lieu d'être devinée sur le tableau des scores, y compris les morts sans arme à feu : répulseur, explosion, chute, sortie de carte
- **« Où ça se joue »** — les positions des frags et des morts sur la carte, tracées en plan quand la carte n'a pas d'image figée
- **La distance par arme**, la répartition des frags en deux niveaux, et le contrôle des armes spéciales ventilé par niveau d'arme (armes de base / de terrain / de puissance, socles de bonus)
- **Section Objectifs** — une colonne par statistique du mode joué, avec un total d'équipe, pour Capture du drapeau, Bastions, Roi de la colline, Oddball, Réserve, Extraction, VIP et Assaut
- **Les manches comptent comme des manches** — sur les modes à manches, le score affiché est le nombre de manches gagnées et perdues, le score en points de l'API restant à côté puisqu'il peut donner l'avantage au camp qui a perdu
- **Repère de prolongation** — un match allé au-delà du temps réglementaire est signalé, avec le temps de jeu supplémentaire
- **Cadence des frags** — frags par tranches de 15 secondes pour vous et l'équipe adverse avec une moyenne mobile, plus une courbe de tir à la corde et un F/M cumulé
- **Repères de retournement** — *Remontada*, *Effondrement* et *Contre-Remontada*
- **Onglets Chronologie et Médias** — les événements du match dans l'ordre, et les clips et captures qui lui sont rattachés

### Suivez votre carrière
- **Historique des rangs** — rating LUSR et CSR par playlist dans le temps, avec votre nom de rang à chaque étape
- **Path to Hero** — graphique de projection montrant à quelle distance vous êtes du rang Hero
- **Cartes de KPI de carrière** — 8 cartes en un coup d'œil : matchs joués, temps total, frags, morts, assistances, précision, temps en vie, barre V/D/É/DNF — chacune colorée en fonction de votre moyenne de toujours
- **Citations** — suivez vos citations Halo avec grilles de médailles et distributions par médaille
- **Médailles** — le catalogue complet des médailles avec vos compteurs, les images servies depuis le catalogue du jeu
- **Pass saisonnier** — progression des paliers avec le carrousel des récompenses et un résumé du contenu
- **Progression d'XP** — courbe d'XP avec superposition de comparaison multi-joueurs
- **Rivaux et rencontres** — qui vous croisez, qui vous battez, et qui vous bat
- **Matchs marquants** — vos matchs saillants, filtrables en classé / non classé

### Analysez vos matchs
- **Explorateur** — parcourez tous vos matchs avec filtres en cascade (carte, mode, playlist, résultat, date, session), recherche partielle par identifiant de match, repères de rencontre, une bande de briefing et un profil de combat ; fonctionne aussi sur **un autre joueur** pour le repérer avant un match
- **Synthèse** — graphe bipolaire, résultats par groupe, meilleures semaines, précision par arme, carte de chaleur d'activité, et la section **portée des engagements** : portée basse (p10) / médiane / haute (p90) par arme, avec le dénivelé signé
- **Séries temporelles** — tendance, densité et barres de KDA, histogrammes de distribution, progression du niveau, écart d'engagement, et premier frag / première mort sur une bande par joueur
- **Sessions** — détail par session avec frags, dégâts, profil d'intensité, ventilation par mode, placement, vies nettes, participation, haltère de MMR, XP de carrière, et le bloc des usages d'équipement
- **Comparaison de sessions** — analyse côte à côte de deux sessions de jeu
- **Carte de chaleur d'activité** — taux de victoire et activité par jour de la semaine et par plage horaire

### Équipement, armes et objectifs — utilisé, gardé ou gâché
- **Les trois issues de chaque équipement** — utilisé, gardé sans jamais l'utiliser, ou lâché en mourant, famille par famille, avec les charges restantes
- **Sur trois pages** — Synthèse, Escouade et Sessions, depuis un bloc partagé unique : variantes comptes et parts, trait de parité, bande de régularité match par match, et une piste de lobby qui montre qui ramasse réellement les armes spéciales
- **Le niveau des armes** — les prises de socle ventilées en armes de base, armes de terrain, armes de puissance et socles de bonus ; ce que vaut un emplacement vient de la carte, jamais du nom de l'arme
- **Les prises nettes de drapeau** — le jonglage replié sur une fenêtre de 1,5 seconde : reprendre son propre drapeau deux fois en une seconde compte pour une prise
- **Les objectifs par rôle et par famille** — prendre, défendre, tenir, sur le Drapeau, le Roi de la colline, les Bastions et le Crâne

### Escouade & coéquipiers
- **Vue d'escouade unifiée** — les mêmes graphes riches pour 1, 2 ou 3 amis ; fonctionne pour toutes les tailles d'escouade
- **« Les formes retenues »** — six lectures d'une escouade sur dix-neuf cartes et trois blocs (équipement, armes spéciales, objectifs), solo contre escouade, votre part face à la parité de votre équipe
- **L'échange en six cartes** — qui donne et qui reçoit, le délai, la matrice, le taux de session et le compte, avec les assistances en barres empilées
- **Carte de chaleur d'intensité par joueur** — le profil de frags de chaque membre par phase de jeu sur les matchs partagés
- **Records d'escouade** — meilleurs scores de carrière pour chaque membre (F/M, frags, séries…) avec détail par carte
- **Radar de synergie** — statistiques par minute et complémentarité dans votre escouade
- **Nuage d'isolement** — où et à quelle fréquence un membre meurt loin du groupe
- **Cadence des frags par joueur** — tempo synchronisé sur les matchs partagés
- **Frise d'impact** — repères narratifs (Meilleur tueur, Héros silencieux, Faux frère…) par match
- **Panneaux d'objectifs** — les statistiques d'objectif de l'escouade, par mode et par rôle

### Ascension — profil, objectifs, entraînement, réalisations, tactique
- **Profil** — vos axes de combat, séries, frise des records, alertes de comportement, leviers à tirer, contexte des motifs et campagnes d'amélioration
- **Objectifs** — créez des défis individuels ou d'escouade (collectifs ou compétitifs) sur n'importe quelle métrique Halo, avec des fenêtres configurables, des paliers (Normal / Heroic / Legendary / Mythic) et des arcs narratifs ; gagnez des Prestige Points (PP) à la complétion
- **Entraînement** — des propositions construites sur vos propres motifs mesurés, pas sur une liste générique
- **Réalisations** — la grille des jalons par titre
- **Tactique** — une grille des cartes que vous jouez avec leur bilan, et une vue d'analyse sur le plan de la carte qui répond à six questions : où vous passez votre temps, où vous mourez, où vous tuez, où vous mourez isolé, où les victoires et les défaites se séparent, et les routes que vous empruntez. Quatre tuiles de KPI (matchs retenus, couverture, échange, morts en isolement), une carte de coordination d'équipe dont le rayon est lu par match dans le référentiel des variantes, et un clic de cellule qui ouvre le rejeu à l'instant exact

### Communauté
- **Classements** — le classement mondial à côté du classement local, avec un état vide honnête quand un classement n'a pas pu être récupéré
- **Prestige** — classement des PP qui vous situe parmi votre escouade et vos relations ; quatre paliers avec badges colorés
- **Relations** — avec qui et contre qui vous jouez, anneaux de taux de victoire, cartes de rivalité, barres de répartition et carte de chaleur des moments
- **Face-à-face** — une comparaison en miroir de deux joueurs, statistique par statistique

### Clips & médias
- **Médiathèque** — parcourez captures et clips vidéo liés à leur match ; filtrez par propriétaire, carte, mode, résultat ou contexte solo/escouade
- **Indexation automatique** — clips rebalayés automatiquement toutes les quelques heures et après chaque synchronisation
- **Réassociation manuelle** — corrigez en un clic un clip mal associé : un sélecteur intégré propose les matchs autour de l'horodatage de capture (±15 / ±60 / ±180 min) avec vignettes des cartes, résultat et lobby complet
- **Mentions J'aime par spectateur** — votre J'aime est le vôtre, pas celui du compte
- **Dans le rejeu** — vos captures occupent leur propre piste de la frise, à la seconde où elles ont été prises

### Notifications & configuration
- **Centre de notifications intégré** — fil par joueur avec repère de non-lus, filtres par catégorie, frise groupée par jour et actions groupées ; rafraîchissement toutes les 60 secondes ; préférences par joueur
- **Alertes Discord** — notifications configurables après la synchronisation, après le rattrapage et quand les rejeux sont prêts, indépendamment
- **Configuration en un clic** — connexion Xbox par code d'appareil (`xbox.com/activate`) avec provisionnement automatique du joueur ; aucun compte Azure requis
- **Personnalisation du Spartan** — votre armure et vos couleurs, recolorées en direct
- **Multi-titre** — Halo Infinite et Halo 5: Guardians côte à côte, chacun avec son stockage, ses catalogues et ses capabilities

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

**Prérequis** : Go 1.26+, Node.js + npm, GNU Make, Air pour le rechargement à chaud du Go, et une **chaîne C** — le pilote DuckDB est en CGO, et sous Windows un `gcc` absent du `PATH` fait servir à Air un binaire périmé sans le dire (MinGW/UCRT64 ; voir [INSTALL.md](INSTALL.md)).

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

**Zéro configuration.** LevelUp embarque son propre client ID Azure.
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
| [INSTALL.md](INSTALL.md) | Guide d'installation détaillé (chaîne C comprise, pour CGO) |
| [CONFIGURATION.md](CONFIGURATION.md) | Configuration des jetons et des profils |
| [COMMANDS.md](COMMANDS.md) | Aide-mémoire des commandes courantes |
| [FAQ.md](FAQ.md) | Questions fréquentes |
| [ARCHITECTURE_V6.md](ARCHITECTURE_V6.md) | Architecture courante (matchs partagés, isolation par titre, assets i18n) |
| [FOUNDATIONS_GUIDE.md](FOUNDATIONS_GUIDE.md) | Fondations du front : mise en page, graphes, jetons |
| [adr/](../adr/) | 34 décisions d'architecture — le *pourquoi* des choix structurants (en anglais) |
| [SYNC_GUIDE.md](SYNC_GUIDE.md) | Guide de synchronisation |
| [ADD_TITLE.md](ADD_TITLE.md) | Ajouter un nouveau titre Halo |
| [BACKUP_RESTORE.md](BACKUP_RESTORE.md) | Sauvegarde et restauration |
| [RUNBOOK_*.md](../) | Runbooks d'exploitation : déploiement, ouvrier de rejeu, test de restauration, outils DuckDB (en anglais) |
| [testing.md](../testing.md) | Stratégie de tests Go (CGO, ratchet de couverture, tag `gamefiles`) |
| [WEAPONS.md](WEAPONS.md) | Référentiel des armes (clés, familles, icônes) |
| [CITATIONS.md](CITATIONS.md) · [référence](CITATIONS_REFERENCE.md) | Système de citations et référence complète |
| [CHANGELOG.md](CHANGELOG.md) · [RELEASE_NOTES.md](RELEASE_NOTES.md) | Journal technique · notes de version utilisateur |

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
| **DuckDB 1.5+** (1.5.5 embarquée) | Moteur de requêtes OLAP |
| **Client Go maison** | Services Halo : synchronisation, catalogues, téléchargement des films — aucune bibliothèque d API Halo tierce |
| **Décodeur de film maison** (Go) | Film Theater → faits → artefact de rejeu |
| **WebCodecs + mediabunny** | Export vidéo hors temps réel du rejeu |

---

## Limitations connues

- **Services Halo** : les points d'entrée ne sont pas documentés et peuvent changer ou limiter le débit sans préavis ; le client est le nôtre, le terrain sous lui ne l'est pas.
- **Le film Theater est la source de tout ce qui se mesure au-delà de l'API de statistiques** — l'arme d'un frag, les positions, la portée des engagements, le niveau des armes, les usages d'équipement, les objectifs vivants et le rejeu 2D viennent tous du film, pas d'un champ de statistiques.
- **Les films expirent du côté de Microsoft** — un match assez ancien pour avoir perdu son film garde ses statistiques d'API mais n'a ni rejeu ni mesure tirée du film ; l'application le classe absent au lieu de le réessayer sans fin.
- **Un build de jeu inconnu est un refus, pas une supposition** — la clé de déchiffrage d'un film est son build ; un film issu d'un build que le décodeur n'a jamais vu est mis de côté avec une erreur typée plutôt que décodé au jugé.
- **Les fonctionnalités tirées du film sont réservées à Halo Infinite** — Halo 5 n'expose aucun équivalent, et l'application le dit au lieu d'afficher une page vide.

---

## Licence

Ce projet est sous licence MIT. Voir [LICENSE](../../LICENSE) pour plus de détails.

---

## Remerciements

LevelUp parle aux services Halo via son propre client Go — aucune bibliothèque d'API Halo tierce. Les travaux ci-dessous sont des références dont nous avons appris, pas du code embarqué :

- **Andy Curtis** ([acurtis166](https://github.com/acurtis166)) pour [SPNKr](https://github.com/acurtis166/SPNKr) — points d'entrée et format du film Theater
- **Den Delimarsky** ([dend](https://github.com/dend)) pour [Grunt](https://github.com/dend/grunt) et [OpenSpartan](https://github.com/OpenSpartan) — documentation communautaire et voie d'import OpenSpartan
- **Gravemind2401** ([Gravemind2401](https://github.com/Gravemind2401)) pour [Reclaimer](https://github.com/Gravemind2401/Reclaimer) — formats de fichiers de cartes, référence de notre décodeur de géométrie

Voir aussi [ACKNOWLEDGMENTS.md](ACKNOWLEDGMENTS.md).

---

**Fait avec passion pour la communauté Halo**
