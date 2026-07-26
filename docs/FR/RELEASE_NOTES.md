## Dernières nouveautés

**v7.3 — Prolongation, premier frag / première mort & la page Réalisations réparée**

**La prolongation sur vos matchs**
- **Un badge dans l'en-tête du match** — un match allé au-delà de son temps réglementaire porte désormais la mention « Prolongation », avec le temps de jeu supplémentaire au survol
- **Une pastille dans l'Explorateur** — le même repère apparaît à côté du résultat dans votre liste de matchs : les prolongations se repèrent sans ouvrir les matchs
- **Tout votre historique, tout de suite** — le repère est déterminé au moment de la lecture : il couvre tous les matchs que vous avez joués, sans resynchronisation et sans attente
- **Mesuré, pas deviné** — le jeu ne fournit aucun signal de prolongation ; le repère compare le temps réellement joué au temps réglementaire de la variante jouée, avec une marge de sécurité
- **Halo 5 volontairement écarté** — aucun temps réglementaire n'y est encore déclaré, et une variante non déclarée n'est jamais signalée plutôt que signalée à tort

**Premier frag / première mort — un nouveau graphe**
- **Une bande par joueur** — les premiers frags de chaque match sont tracés au-dessus de la ligne et les premières morts en dessous : celui qui ouvre vite et celui qui tombe vite se distinguent d'un coup d'œil
- **Médianes et fenêtre d'avance** — chaque bande porte sa médiane de premier frag, sa médiane de première mort et la barre entre les deux : l'avance dont vous disposez habituellement
- **Trié par précocité** — les bandes sont classées par médiane de premier frag
- **Sur trois pages** — Escouade (onglet « Dynamique », après l'Intensité), Séries temporelles (onglet « Progression ») et détail de session, où il figure aussi dans le volet de comparaison
- **Il remplace les anciens histogrammes** — les graphes par tranches de 10 et de 15 secondes étaient illisibles et se contredisaient ; les deux disparaissent
- **Une échelle qui tient** — un match interminable n'écrase plus la bande de tous les autres

**Halo 5 — page Réalisations réparée**
- **La page s'ouvre de nouveau** — elle répondait par une erreur au lieu d'afficher vos jalons
- **Titre de section toujours présent** — la grille des Réalisations garde son titre, qu'elle charge, qu'elle soit vide ou en erreur

**Lecture dans votre langue**
- **Les citations suivent la bascule de langue** — leurs libellés venaient du serveur dans la langue du moment et y restaient jusqu'au rechargement de la page
- **« Tirs à la tête » est traduit** — la statistique s'affichait sous son nom de champ brut à plusieurs endroits
- **Un démarrage plus rapide** — le référentiel des champs n'est plus récupéré deux fois à l'ouverture de l'application

**Vue de match**
- **Une statistique d'objectif manquante ne casse plus le tableau des scores** — un match dont les données d'objectif ne pouvaient pas être lues s'affichait comme incomplet en entier ; seule la section Objectifs reste désormais vide, tout le reste est servi

**Démo**
- **Campagnes d'amélioration visibles** — le profil de démonstration n'affichait aucune campagne
- **Statistiques d'objectif incluses** — la démo porte désormais les données d'objectif que possède l'application réelle
- **Anonymisation renforcée** — les identifiants de joueur sont anonymisés comme le reste

**Aussi livré en amont**
- **Vue de match réorganisée** — le bloc Médias passe en dernier, seul et en pleine largeur, et les rangées de frags, de graphes et de médailles ne laissent plus une carte esseulée sur un tiers de largeur
- **Participation aux objectifs revue** — l'axe Objectifs de votre profil est désormais un indice de ce que vous avez réellement fait au regard de ce que les modes joués avaient à offrir, au lieu d'un score brut dilué par l'ensemble de vos matchs

**v7.2.1 — Modes à objectif, étanchéité entre titres & recherche de joueur instantanée**

**Trois modes à objectif de plus — Stockage, Extraction et VIP**
- **Stockage** — graines d'énergie déposées et volées, porteurs adverses éliminés, et votre temps passé à porter une graine
- **Extraction** — extractions réussies, amorçages menés à terme, balises adverses converties et conversions empêchées
- **VIP** — VIP adverses abattus, nombre de fois désigné VIP, frags réalisés en étant le VIP, temps cumulé et plus longue survie en VIP
- **Où les lire** — dans la section « Objectifs » de la vue de match, exactement comme la Capture de drapeau, les Bases, le Roi de la colline et Oddball : une colonne par statistique du mode joué, avec une ligne « Total équipe »

**Dix nouvelles citations d'objectif**
- **Autour du drapeau** — Capture du drapeau, Sécurisation du drapeau, Vol du drapeau, Chasse au rapatrieur, Porteur imparable et Rapatriement agressif
- **Zones et Oddball** — Défense de zone, Crâne intouchable, Chasse au porteur et Prise du crâne
- **Des paliers réglés sur le jeu réel** — chaque palier est calibré sur vos vraies données de match plutôt que sur une échelle générique, pour que les exploits rares (réaliser des frags en portant le drapeau ou le crâne) restent atteignables

**Explorer — toutes les saisons, plus seulement une partie**
- **« Matchs par saison » est complet** — la ventilation ne couvrait que les saisons que le jeu voulait bien renvoyer ; elle couvre désormais toutes les saisons du catalogue
- **Plus rien ne manque en silence** — une saison que vous n'avez jamais jouée et une saison qui n'a pas pu être récupérée ne s'affichent plus de la même façon

**Objectifs d'escouade & Prestige**
- **« Proposer des défis » refonctionne** — le bouton pouvait tomber en erreur définitivement jusqu'au redémarrage du serveur
- **Fini les défis dont la règle ne correspond pas** — les défis dont la règle annoncée ne correspondait pas à la façon dont ils étaient réellement évalués ne sont plus proposés
- **Prestige sans point** — ouvrir votre prestige avant d'avoir gagné le moindre point renvoyait une erreur au lieu d'un récapitulatif vide

**Des graphiques qui se lisent droit**
- **Durée de vie moyenne réelle** — le graphe de durée de vie utilise la valeur mesurée en jeu au lieu d'une estimation tirée du temps joué et des morts, et le nuage de corrélation raconte enfin la même histoire que l'histogramme
- **Taux de victoire et MMR sur leur propre axe** — la courbe de taux de victoire n'est plus écrasée en bas du graphe par l'échelle du MMR qu'elle partageait
- **Domination sur la bande de résultats** — un repère signale les matchs que votre équipe a dominés
- **Radar de synergie** — l'infobulle affiche la valeur brute à côté du score normalisé

**Modes à objectif — Capture de drapeau, Bases, Roi de la colline, Oddball**
- **Statistiques d'objectif collectées à chaque match** — captures, retours, vols de drapeau et temps en tant que porteur ; captures et sécurisations de zone et temps passé en zone ; récupérations du crâne, temps de possession et plus longue possession
- **Dans la vue de match** — une nouvelle section « Objectifs » par équipe, une colonne par statistique du mode joué, avec une ligne « Total équipe »
- **Dans vos totaux** — les chiffres d'objectif agrégés sur la Synthèse, l'Escouade et les Séries temporelles, affichés uniquement là où le titre les fournit
- **Citations d'objectif** — « À la charge », « Je te tiens ! », « Partie prenante » et « Sus au porteur du drapeau » progressent désormais sur les vrais compteurs d'objectif au lieu des médailles

**Fini les données Halo 5 qui fuyaient sur Halo Infinite**
- **Porte de titre stricte** — le bandeau Spartan (bannière, emblème, arrière-plan, indicatif de service) n'affiche plus les éléments d'un autre titre pendant le chargement d'une page ni pendant un changement de jeu
- **Changement de jeu sans données croisées** — chaque réponse du serveur indique désormais pour quel titre elle a été résolue, et l'app refuse tout ce qui ne correspond pas au titre affiché
- **Apparence par joueur et par titre** — les bannières et emblèmes Halo 5 ne sont plus partagés entre joueurs, et les couleurs d'emblème et de bannière sont conservées séparément

**Explorer — recherche instantanée et données en direct honnêtes**
- **Recherche de joueur instantanée** — les suggestions de gamertag viennent de vos données locales en 200 ms environ ; un bouton explicite « Rechercher sur Xbox » va interroger Xbox quand vous en avez besoin
- **N'importe quel joueur redevient lisible** — la carrière, l'identité, les médailles et les saisons d'un joueur recherché passent par le vivier d'identifiants valides : une cible dont les identifiants sont morts n'est plus une impasse
- **Fini la dégradation muette** — quand les données en direct ne peuvent pas être récupérées, un badge discret le dit (« Données live indisponibles (authentification) », « (erreur) », « Live partiel ») au lieu de laisser une carte vide
- **Navigation plus fluide** — sélectionner un joueur n'empile plus une entrée d'historique par clic, le graphe « Écart de frags cumulé » retrouve son axe horizontal, et un message explicite le remplace quand vous n'avez aucun match en commun

**Halo 5 — catégories de médailles**
- **215 médailles en 11 catégories** — les médailles Halo 5 sont désormais regroupées comme celles de Halo Infinite (Bases, Zone de combat, Objectif, Capture du drapeau, Oddball, séries, multi-éliminations, armes, véhicules, infection, style) sous les quatre super-sections habituelles
- **Médailles fantômes masquées** — trois médailles gagnées en partie mais absentes du catalogue officiel ne polluent plus la page Médailles (leurs données sont conservées)

**Lisibilité**
- **Infobulles de colonnes partout** — survolez n'importe quel en-tête de tableau pour obtenir la définition de la colonne
- **Légendes en pied de bloc** — les légendes de graphe sortent de la zone de dessin, ce qui rétablit au passage l'épaisseur des barres d'« Outils de destruction »
- **Grenades ventilées par type** — le sunburst des frags détaille les grenades en fragmentation / plasma / dynamo / éclats, et un double comptage Halo 5 (corps-à-corps en tenant une arme) disparaît
- **Un seul nom d'arme par titre** — les noms d'armes viennent d'un référentiel unique par titre : « SPNKr à combustible », « Grenade à fragmentation » ou « Fusil léger » s'écrivent partout pareil
- **« En placement » au lieu du vide** — une note de performance encore en calibration affiche « En placement (8/10) » dans les tableaux et les tuiles d'accueil plutôt qu'un tiret nu
- **Libellés plus clairs** — « Net score cumulé » devient « Solde frags − morts cumulé », « Lobby » devient « Partie », « OC / DR » devient « Rendement / Résistance », les playlists et modes du sélecteur d'escouade sont traduits, la description de la médaille Méganaute est réparée, et les blocs de la page Carrière sont alignés en hauteur

**Notifications & alertes**
- **Notification de médaille inédite** — vous êtes prévenu la première fois que vous décrochez une médaille
- **Alertes Discord qui se tiennent** — les alertes disque ne partent plus en rafale après chaque redémarrage, le retour à la normale est annoncé une seule fois, la notification de version porte la vraie version, et le cycle de synchronisation automatique annonce ses nouveaux matchs
- **Notifications lisibles** — les noms de rang de carrière sont traduits et les valeurs sont arrondies

**Administration & synchronisation**
- **Synchronisation initiale d'un seul joueur** — une nouvelle carte d'administration ré-importe tout l'historique d'un joueur, avec une légende expliquant la portée de chacune des quatre actions de synchronisation

**Confort d'utilisation**
- **« Dynamique » dans le menu Escouade** — l'onglet existait mais manquait dans la navigation
- **« Voir les synergies »** — un raccourci depuis une session d'escouade vers la vue Synergies
- **XP de carrière sur les Sessions** — la courbe d'XP cumulée et l'XP par match, déjà présentes sur les Séries temporelles, couvrent maintenant une session
- **Match pas encore synchronisé** — ouvrir un match absent de vos données affiche un écran dédié au lieu d'une erreur générique

**v7.1 — Escouade fiabilisée, données de combat Halo 5 & XP de carrière**

**Escouade — plus fiable**
- **Historique en composition exacte** — « Performance d'escouade par session », « Performance par carte — session vs historique » et « Taux de victoire — session vs historique » comparent désormais chaque session à votre historique avec cette composition *exacte*, au lieu d'une moyenne floue tous coéquipiers confondus
- **Compositions enregistrées corrigées** — une escouade enregistrée affiche les mêmes membres pour tout le monde ; fini les doublons ou le coéquipier manquant chez un autre joueur
- **Graphes d'escouade en ordre chronologique** — les comparaisons par carte et de taux de victoire se lisent de gauche à droite dans l'ordre d'apparition des cartes, comme la vue d'intensité
- **Axe de taux de victoire plus clair** — le compteur « (n) » de l'axe est désormais expliqué par une infobulle au survol
- **Nouvel onglet « Dynamique »** — les graphes d'intensité, de rendement/résistance et d'engagement sont regroupés dans un onglet Escouade dédié
- **Intensité en profil médian** — la heatmap match × phase laisse place à un profil médian de part de frags par phase, avec une enveloppe interquartile qui rend visible l'irrégularité
- **Rendement & Résistance** — séparés en deux graphes multi-joueurs, une couleur par joueur
- **Balance des dégâts en vies** — cumul des dégâts infligés moins subis, exprimé en vies du titre, sur Sessions et Dynamique
- **Écart d'engagement cumulé** — une nouvelle courbe d'écart d'engagement cumulé sur Séries temporelles, Dynamique et Sessions

**Objectifs d'escouade**
- **Boucle de défis d'escouade** — libellés localisés, retour au moment de rejoindre, progression par membre en direct, et cycle de vie complet (abandon, suppression, expiration)
- **Renommage** — « Cap d'escouade » devient « Objectifs d'escouade »
- **Plus d'échec silencieux** — le bouton « Proposer des défis » affiche désormais une erreur explicite au lieu de ne rien faire

**Halo 5 — données de combat réparées**
- **Véhicules détruits & Vol à la tire** — compteurs de véhicules détruits et de « Vol à la tire » sur la carte « Outils de destruction » de la Synthèse
- **Mécaniques de combat restaurées** — les assassinats, coups au sol et charges spartanes sont réparés pour les matchs dont les valeurs avaient été enregistrées à zéro
- **Scoreboard assaini** — noms d'équipe Halo 5 dédupliqués, et MVP/LVP ne sont plus départagés par les kills de mécanique
- **Colonnes de mécanique masquées hors Halo 5** — les colonnes assassinat / coup au sol / charge spartane n'affichent plus des zéros sur Halo Infinite

**Citations réparées**
- **Firefight** — les « Éliminations Firefight » comptent désormais vos victoires Firefight
- **Remaps & correctifs** — citations de grenade restaurées, « Virée sur la route » remappée sur la médaille Écrasement, et « Défenseur du drapeau » désactivée proprement (aucune source de données pour l'instant)

**Carrière**
- **XP de carrière estimée** — une courbe d'XP cumulée et l'XP par match sur les Séries temporelles, calibrées sur des données réelles
- **Path to Hero multi-titre** — progression vers le rang maximum propre au titre (corrige Halo 5 qui visait le mauvais plafond)
- **Page Médailles** — une nouvelle sous-page Carrière listant le catalogue complet des médailles du titre, y compris celles jamais obtenues, regroupées par section, avec filtres toutes / obtenues / non obtenues et tris

**Accueil & Synthèse**
- **Date du pic sur les cartes de rang** — vos cartes de meilleur LUSR / CSR affichent la date à laquelle le pic a été atteint
- **Plus longue série de défaites** — une nouvelle carte KPI à côté de votre série de victoires

**Explorer**
- **Face-à-face** — la section « Sur XX matchs joués ensemble » ajoute des donuts de taux de victoire (ensemble / face à lui) et un graphe d'écart de frags cumulé par rapport à la cible
- **Synthèse repliable** — la synthèse du mode Matchs peut être repliée, et ce choix est mémorisé

**Écart au FDA attendu**
- **Courbes d'écart attendu** — de nouveaux graphes « écart au FDA attendu » sur les Séries temporelles et les Sessions, plus un écart cumulé par membre sur les Synergies d'escouade, avec un KPI d'écart moyen par match
- **Superposition du FDA attendu** — une fine courbe de FDA attendu par match est ajoutée sur le graphe d'écart, sur le même axe

**Médias**
- **Rôles des pistes audio par joueur** — déclarez le rôle voix / jeu / autres de chaque piste audio de vos clips, par joueur, depuis une modale engrenage dans la galerie (manuel, ou automatique)

**Confort d'utilisation**
- **Langue dans l'URL** — la langue active fait désormais partie de l'adresse, donc les liens partagés conservent leur langue
- **Colonne Halo Waypoint** — une colonne optionnelle pour ouvrir n'importe quel match sur Halo Waypoint
- **Tous les tableaux triables** — cliquez sur n'importe quel en-tête de colonne pour trier, sur tous les tableaux de l'app
- **Titres d'onglet stables** — des titres d'onglet conscients de la page et de la langue au lieu d'un simple « LevelUp »
- **Identité du scoreboard** — couleurs d'identité par équipe et logos d'équipe, plus une largeur de colonne joueur cohérente
- **Formulation française** — une passe de purge des anglicismes remplacés par des termes français
- **Légendes & pourcentages** — la légende « Outils de destruction » est centrée sous le graphe, avec des étiquettes de pourcentage sur les segments et les légendes et des hauteurs de graphe alignées sur Synthèse, Sessions et Escouade

**Admin**
- **Diagnostic d'apparence Spartan** — un panneau par joueur dans l'onglet Données de l'admin explique pourquoi une bannière, un emblème, un arrière-plan ou un indicatif de service n'a pas pu se charger, avec un raccourci de reconnexion

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
