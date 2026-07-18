## Dernières nouveautés

**v7.0 — Nouvelle app React, Mission Control & multi-titres**

Refonte majeure. LevelUp quitte Streamlit pour une app **React 19 + Go API** avec **une UX/UI entièrement repensée** et **une synchronisation désormais automatique**. Près de 400 commits de travail — voici ce que ça change pour vous :

**Une app entièrement repensée — nouvelle UX/UI**
- **Frontend React 19 + Tailwind** — navigation instantanée entre pages, URL partageables pour chaque match/session/joueur, thème clair/sombre, i18n FR/EN complète
- **Design system unifié** — palette, typographie, espacements et composants (cartes, boutons, modales, tooltips, carousels, lightbox) harmonisés sur toute l'app ; fini les pages Streamlit disparates
- **Navigation à deux niveaux** — barre L1 (Accueil / Synthèse / Explorer / Escouade / Communauté / Ascension / Médias / Aide / Paramètres) plus onglets L2 contextuels ; tout est à un clic, plus de sidebar qui défile
- **Interactions modernes** — transitions de page, deep-links (`?session=`, match précédent/suivant), survols, drawers latéraux, retours visuels sur chaque action, UI responsive
- **Backend Go API** — bascule Python → Go pour un serveur plus léger, un démarrage plus rapide et une empreinte mémoire réduite
- **Support multi-titres** — l'app gère désormais plusieurs jeux Halo (Infinite et au-delà) via un **TitleSwitcher** dans la barre de navigation ; commande CLI `levelup add-title` pour enregistrer un nouveau titre
- **Page d'aide intégrée** — Notes de version (changelog consultable dans l'app) et Glossaire des termes Halo, avec cache local 24 h pour lecture hors-ligne
- **Mode démo multi-joueur** — démo publique en lecture seule avec un corpus multi-joueur anonymisé (un Spartan et deux coéquipiers), de vrais clips HLS et des fixtures de prestige, avec changement de langue côté client

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
- **Bande Rendement/Résistance** — Taux de Conversion Offensif / Résistance Défensive affichés en une bande unifiée sur Synthèse, Sessions et Escouade
- **Vraie durée de jeu** — les durées soustraient désormais le décompte pré-partie, donc le temps joué et les stats par minute reflètent le vrai temps de combat

**Médias V2 — likes, notifications Discord, upload**
- **Likes persistants** — aimez vos screenshots et clips directement depuis la grille, état conservé entre les rechargements
- **Groupage intelligent** — par favoris, par session ou par contexte solo/escouade
- **Grille allégée** — thumbnails natifs, lightbox partagée, icônes cœur pour liked / unliked
- **Upload glisser-déposer** — ajoutez vos captures manuelles directement depuis la page Médias
- **Scan non-destructif** — ré-indexation automatique en arrière-plan avec option `--captures-dir` dédiée
- **Notifications Discord pour nouveaux médias** — embed avec GIF ou miniature screenshot à chaque nouvelle capture indexée ; anti-spam (chaque fichier notifié une seule fois) ; toggle `discord_notify_new_media` dans les paramètres
- **Réassociation manuelle avec suggestions de matchs** — modale intégrée qui liste vos matchs dans une fenêtre ±15 / ±60 / ±180 min autour de la capture, avec miniature de la carte, carte · mode · playlist, heure locale + écart, badge de résultat et lobby complet par équipe ; un clic + confirmation pour corriger un média associé au mauvais match
- **Lecteur vidéo HLS** — lecture des clips dans le navigateur avec sélecteur de piste audio (jeu / voix / mix complet) et transcodage multipiste à l'ingestion (clips HEVC remixés automatiquement)

**Page Match dédiée & visualisations enrichies**
- **URL propre par match** — `/players/{gamertag}/matches/{id}`, partageable, avec navigation match précédent/suivant
- **Timeline Tug-of-War** — courbe dynamique des retournements de score entre équipes
- **KD Timeline** — évolution kills/morts par phase avec moyenne mobile
- **Impact Badges** — badges narratifs (Top Killer, Silent Hero, False Brother, Comeback Champion…) calculés par match
- **Panneau Encounters** — liste des joueurs déjà croisés lors de matchs précédents
- **Combat Yield & Perfect Kills** — nouvelles métriques dans la vue match
- **Scoreboard V7** — densité d'info accrue : expected stats, skill rank, média liée, citations
- **Comparaison de sessions** — page A/B dédiée : choisissez deux sessions et comparez KDA, score de performance, Taux de Conversion Offensif / Résistance Défensive, distribution des résultats et playlist dominante côte à côte
- **Exclusion d'un match** — retirez un match de vos stats avec recalcul complet en cascade (sessions, score de performance, citations) et garde-fou pour les matchs classés
- **Badge de rang toujours visible** — palier LUSR/CSR affiché sur le scoreboard, y compris les CSR partagés pour les joueurs non suivis

**Authentification repensée**
- **Provider SISU/PoP** — nouvelle authentification Xbox avec Proof-of-Possession pour des sessions plus stables et moins de reconnexions
- **Flux OAuth redirect** — connexion Xbox via navigateur (`/auth/xbox/login` → Microsoft → callback) comme alternative au Device Code ; configurable via `LEVELUP_OAUTH_REDIRECT_URI`
- **Auth locale** — mode nom d'utilisateur/mot de passe pour déploiements mono-utilisateur / LAN
- **Inscription par invitation** — nouvelle page `/register` : créez votre compte LevelUp uniquement via un lien d'invitation envoyé par l'administrateur ; les codes expirés ou déjà utilisés sont refusés avec un message clair
- **SISU par défaut & verrouillage d'instance** — le provider SISU est désormais l'auth Xbox par défaut ; l'instance peut être fermée et des contrôles d'ownership par joueur isolent les données de chacun (pas d'accès inter-comptes)
- **Mot de passe opt-in pour re-login rapide** — définissez un mot de passe pour rouvrir vite votre session SSO sans repasser par tout le flux Xbox à chaque fois
- **Bannière de reconnexion & détection de token mort** — une bannière in-app détecte un refresh-token Xbox mort et vous guide pour vous reconnecter avant que le sync ne casse ; le token store est la source unique des identifiants

**Achievements Xbox & événements de match**
- **Sync des achievements Xbox** — vos succès Xbox sont récupérés automatiquement depuis l'API Halo à chaque sync
- **Suivi des achievements** — parcourez votre liste complète de succès Xbox sur la page Carrière : filtrez par débloqué / en cours / non commencé, suivez votre Gamerscore (obtenu vs total) et filtrez par jeu pour le support multi-titres
- **Highlight events** — parseur binaire des films de match pour extraire tous les événements majeurs (medals, clutchs, spawns)
- **Backfill weapon kills** — arme utilisée par frag reconstruite depuis le film (POV ~87 %)
- **Badges Comeback pour coéquipiers** — Remontada / Collapse / Contre-Remontada calculés pour vos co-joueurs synchronisés en même temps que vous
- **Filtre par catégorie de succès** — filtrez vos succès Xbox par Multijoueur / Campagne / Autres, Multijoueur affiché par défaut

**Communauté — Palmarès, Relations & Face-à-face**
- **Season Pass multilingue** — traductions des Battle Pass dans 26 langues, tier images depuis GameCMS
- **Relations** — suivez tous les joueurs que vous avez croisés : stats par joueur, historique de matchs partagés, badges alliance et rivalité, micro-leaderboard de carrière
- **Face-à-face** — page de comparaison 1v1 (ou 1v1v1 en miroir) : opposez deux ou trois joueurs sur les métriques Combat, Précision et Bilan ; badges de rencontre (allié, rival, adversaire coriace) issus de votre historique partagé

**Objectifs & Prestige**
- **Objectifs** — système de défis individuels et d'escouade : fixez des objectifs personnels ou créez des défis d'escouade (collectifs ou compétitifs) sur n'importe quelle métrique Halo avec des fenêtres temporelles, des paliers et des arcs narratifs ; gagnez des Prestige Points (PP) à la complétion ; deux modes d'évaluation (seuil / cumulatif) et deux modes de création (libre / piloté)
- **Leaderboard Prestige** — classement PP dans Palmarès comparant votre score à ceux de votre escouade et de vos relations ; quatre paliers : Normal / Heroic / Legendary / Mythic
- **Arcs narratifs** — regroupez des défis en arcs avec création libre, presets prêts à l'emploi et suppression ; un bonus de complétion d'arc est crédité et affiché à l'étape finale
- **Défis pilotés par le coach** — le mode guidé propose des défis auto-calibrés sur vos axes de performance les plus faibles

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
- **Suivi de campagne** — lancez une campagne d'amélioration sur un objectif choisi et suivez votre progression dans le temps, avec un tracker dédié et une modale de démarrage

**Centre de notifications in-app**
- **Centre de notifications** — fil par joueur avec badge non-lus dans la barre de navigation, filtres par catégorie, timeline groupée par jour, actions groupées et rafraîchissement live toutes les 60 secondes ; préférences configurables par joueur dans les Paramètres

**Synchronisation automatique & présence temps réel**
- **Sync 100 % automatique** — finis les `python scripts/sync.py` à lancer à la main : l'app synchronise vos matchs toute seule en arrière-plan, en continu, dès qu'une nouvelle partie est jouée
- **Déclenchement immédiat fin de partie** — dès qu'un joueur termine un match, le watcher récupère les stats sans attendre le prochain tick
- **Présence RTA Xbox + polling Steam** — détection en ligne temps réel pour savoir qui joue et synchroniser au bon moment
- **Scheduler intelligent** — cadence de sync adaptative selon l'activité des joueurs ; pas de requêtes inutiles quand personne ne joue
- **Rafraîchissement autonome des tokens** — plus d'interruptions : les tokens Halo se renouvellent tout seuls en tâche de fond
- **Reconnexion proactive** — gestion du status=3 avec refresh XSTS à la demande, reconnexion automatique au démarrage
- **Sync convergent** — les noms d'assets (cartes, modes, playlists) se résolvent tout seuls pendant le sync, avec un filet hebdomadaire de rafraîchissement du catalogue pour la traîne
- **Déduplication cross-source** — les syncs concurrents d'un même match sont dédupliqués pour ne rien récupérer ni écrire deux fois

**Paramètres & admin**
- **Auto-save des paramètres** — les réglages se sauvegardent immédiatement avec indicateur visuel éphémère
- **Page admin** — UI de supervision (auth provider, état des jobs, privacy)
- **Préférences navigateur** — joueur sélectionné, langue et filtres mémorisés entre sessions
- **API endpoints configurables** — Halo Stats, SPNKr, CMS… tous paramétrables depuis les Paramètres
- **Sauvegarde & restauration automatiques** — snapshots restic de chaque base, avec un onglet dédié dans les Paramètres (déclenchement manuel, statut par base, contrôle d'intégrité informationnel) et restauration à une date donnée
- **Dashboard de monitoring** — supervision admin complète : cycles de sync et sparklines de tendance, convergence, invariants d'intégrité des données, santé des tokens (MSAL/XSTS/Refresh), attribution des appels API Halo par joueur, collecteur d'erreurs récurrentes, logs et performance

**Assets & cartes**
- **Cache-aside des images de cartes** — artworks de maps téléchargés et mis en cache local, plus aucune requête externe répétée
- **CLI `populate-assets`** — commande Go pour pré-télécharger tous les assets (cartes, medals, tiers Battle Pass) avant usage hors-ligne

**Accessibilité des couleurs**
- **Palette adaptée aux daltoniens** — une nouvelle palette Okabe-Ito (conçue en 2008, recommandée universellement) est disponible dans Paramètres → Accessibilité ; elle remplace toutes les couleurs de l'app — graphiques, indicateurs de performance, résultats de match, K/D — par des teintes distinguables en cas de deutéranopie, protanopie et tritanopie
- **Aperçu en direct** — la palette bascule instantanément sur toute l'app sans rechargement de page ; un aperçu en pastilles permet de comparer avant de valider
- **Préférence persistante** — votre choix est sauvegardé dans le navigateur et restauré automatiquement à chaque visite

**Page Sessions repensée**
- **Graphes enrichis** — F/D/A par match et par minute, score de performance par tier, radar F/D/A, nuage TC/RD et engagement par match, avec axes explicites et bandes de sous-palier
- **Drawer de comparaison A/B** — choisissez deux sessions et comparez-les côte à côte, avec des échelles partagées sur tous les graphes
- **Métriques en vue solo** — Taux de victoire, KDR, kills/match, précision moyenne et delta de rang affichés directement en vue simple
- **Delta de rang par match** — mouvement LUSR/CSR affiché match par match, avec une fenêtre de session adaptative

**Explorer — profils de combat & rivalités**
- **Profil de combat de n'importe quel joueur en live** — consultez le profil de combat récent d'un joueur non suivi en live (lecture seule, cache court), avec rang de carrière et grade Spartan
- **Dominance & rencontres** — métriques de dominance et rencontres issues de l'historique partagé (allié / rival / adversaire) affichées par joueur
- **Export CSV** — exportez le tableau de matchs filtré en un clic
- **Filtres en cascade** — cinq dimensions de filtre dont les options disponibles s'ajustent au fur et à mesure que vous affinez les autres
- **Matchs par saison** — barres de matchs par saison avec un badge de rang CSR
- **Briefing de recherche** — un récapitulatif compact au-dessus des résultats : matchs, taux de victoire, FDA et performance (affichés en plus bas · moyen · plus haut), temps total et pics ; répartitions par carte / mode / sélection / contexte ; progression de classement par sélection ; meilleures séries et moments forts — chaque bloc avec un tooltip de légende
- **Tableau triable & mise en valeur des extrêmes** — cliquez sur l'en-tête d'une colonne numérique pour trier l'ensemble des résultats ; les 10 % meilleures et pires valeurs de chaque colonne clé sont surlignées pour repérer d'un coup d'œil vos meilleurs et pires matchs

**Classé (CSR)**
- **CSR par match & par playlist** — CSR capturé pour chaque match classé et pour chaque playlist classée active
- **Sélecteur de saison CSR** — basculez entre les saisons CSR disponibles ; les saisons passées peuvent être backfillées
- **Seuils de placement dynamiques** — le nombre de matchs de placement est résolu par saison
- **CSR des coéquipiers automatique** — le CSR de chaque co-joueur enregistré est récupéré et distribué lors d'un sync classé
- **Référence autoritative des playlists classées** — le statut classé est lu depuis une référence stable au lieu d'être deviné depuis les matchs

**Classement mondial**
- **Classement CSR mondial** — un classement CSR mondial scrapé depuis Halo Waypoint, enrichi de stats natives par joueur (KDA, précision, dégâts) sur plusieurs saisons
- **Tendance inter-saison** — un indicateur coloré montre le mouvement de chaque joueur par rapport à la saison précédente
- **Joueurs locaux en avant** — vos joueurs suivis sont toujours remontés en tête

**Coach d'escouade**
- **Orientation d'escouade** — le coach affiche le cap actuel de l'escouade et biaise le pool de défis vers vos axes de performance les plus faibles
- **Carte « Cap du moment »** — une CoachFocusCard met en avant la seule chose la plus utile à travailler ensuite, avec un signal soft-négatif qui reste encourageant

**Précision du rating (LUSR v2)**
- **Moteur de rating refondu** — un nouveau modèle TrueSkill2 (graphe de facteurs + expectation propagation) avec pondération par temps joué, prise en compte des abandons, probabilité de victoire pré-match et protections anti-volatilité à l'affichage, pour que votre rating bouge pour les bonnes raisons

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
