# Mesures AVANT, par piste (plan décodeur v1.6.0, item 0.8)

Date : 2026-11-23. Aucun film décodé pour ce fichier, aucune base ouverte hormis la copie de l'API, aucun réseau.
Aucun identifiant de joueur ni pseudo ici ; les films se désignent par leur identifiant court de match.

## Vérité de l'API (ce qui existe)

- Copie de la base partagée : `dec-v160-recup/verite/api_copie.duckdb`, octet pour octet la copie faite le 2026-11-22
  à 10:59 (`dec-v150-recup/c5/api_copie.duckdb`, vérifié par `cmp`). Elle n'est donc PAS une copie faite le 2026-11-23.
  Contenu : 2 045 matchs au registre (`match_registry`, de 2021-11-19 à 2026-11-10), 29 107 lignes de participants
  (`match_participants`, 41 colonnes dont score personnel, éliminations, morts, assistances, tirs, touches, dégâts,
  temps de jeu, présence). Les 13 films du relevé y sont tous.
- Absent de la copie : la base des référentiels (noms de cartes en anglais) ; toute colonne d'apparence des Spartans
  (armure, revêtement : piste E.4) ; toute donnée de réplique. La copie n'a jamais été rouverte en écriture (octets
  inchangés après les lectures).
- Dérivés des mêmes lectures sur le parc de la v1.4.1 : tous les chiffres de `LevelUp/.ai/annexes_decodeur_1118/avant.md`
  (jointures, score, compteurs, drapeau, zones, gestes, temps forts), pris sur 1 455 rejeux de schéma 125. Le parc n'a
  PAS été régénéré depuis (décision du 2026-11-22 : une seule régénération, après la v1.6.0) ; ces chiffres décrivent
  donc le décodeur v1.4.1, pas la v1.5.0. L'état de la v1.5.0 n'est établi que sur des échantillons (cités par piste).

## Faits des 13 films du relevé de vitesse

Export par `levelup replay-facts-export --oracle`, binaire construit depuis `LevelUp-wt-dec160`, lancé sur une racine
de données à part (`LEVELUP_DATA_DIR` = `verite/donnees`, copie de la copie, `LEVELUP_LOGS_DIR` = `verite/logs`,
`LEVELUP_RESOURCES_DIR` = `racine_essai`). 13 films, 26 fichiers (`<court>.facts.json` et `<court>.oracle.json`),
aucun refus (faits et oracle non vides pour les 13) :

| Rang | Film | Joueurs | Variante |
|---|---|---|---|
| 1 | e5adf7b2 | 28 | BTB:Fiesta Slayer |
| 2 | 03b7bbb6 | 8 | Slayer:Arena Super Fiesta |
| 3 | 221dd082 | 8 | Arena:Team Slayer |
| 4 | 1c4c63c2 | 24 | BTB:One Flag CTF |
| 5 | 51101d1d | 13 | CTF:Arena Neutral Flag |
| 6 | 50247b26 | 30 | BTB:Slayer |
| 7 | 084a804d | 26 | BTB Heavies:CTF |
| 8 | 11de8353 | 27 | BTB:Fiesta Slayer |
| 9 | a521164d | 27 | BTB Heavies:Total Control |
| 10 | 7344d24f | 8 | Strongholds:Arena |
| 11 | 01e1f945 | 8 | KOTH:Arena |
| 12 | 60ae07c4 | 8 | Ranked:Oddball |
| 13 | 9f57c612 | 12 | Assault:One Bomb |

Réserve : la base des référentiels étant absente de la racine, les noms de carte candidats sont ceux du registre brut
(ex. « Live Fire - Ranked », sans la forme canonique en anglais). Les faits (joueurs, scores, variante) sont complets ;
seul le second nom candidat de carte manque. À comparer avec le fichier de faits du corpus d'équivalence avant de
s'en servir pour un contrôle de carte (le relevé B de V0.4 ne lit pas la carte pour la vitesse).

Films : les 11 films du corpus d'équivalence sont déjà sous `LevelUp-wt-dec160/data/cache` (morceaux et manifeste) ; les
deux témoins de durée (03b7bbb6 : 25 morceaux, 221dd082 : 30 morceaux) y ont été copiés depuis l'archive du dossier
principal (copie de fichiers, octets et nombre de morceaux identiques à la source, manifestes identiques ; aucun
serveur ni processus LevelUp, rien sur le port 8000 au moment de la copie).

## Table : piste, mesure attendue, valeur avant

Légende : « parc v1.4.1 » = mesure sur les 1 455 rejeux non régénérés ; « échantillon v1.5.0 » = mesure du plan v1.5.0
sur un sous-ensemble de films. Les renvois `1118` désignent `.ai/annexes_decodeur_1118/avant.md` (sections), `P160`
le plan v1.6.0 (lignes), `P150` le plan v1.5.0 (`.ai/PLAN_DECODEUR_2026-11-18.md`, lignes).

### Piste A, score et tableau

| Item | Mesure attendue (P160) | Valeur avant | Source | Sinon |
|---|---|---|---|---|
| A.1 | cause par film : 5 Grandes batailles « incomplètes » sans rang non confirmé ni manque ; 1 « complète » à tort ; 1 film au tableau qui cesse de monter | comptes 5 / 1 / 1 posés par le plan ; cause non établie | P160 l.843-846 | à mesurer par A.1 (harnais, sur les films du parc) |
| A.3 | décalage de l'arbre prouvé par le tableau du film, sur le build qui précède HI_1_11_0 (31 lectures concordantes) et 34 Grandes batailles à drapeau sans groupe de scripts | HI_1_11_0 Grande bataille : rangs non confirmés 278 → 0, joueurs au-dessus des points publiés 41 → 0, verdict complet 12 → 18 films, sur 23 films (v1.5.0, B.3) ; les 31 lectures et les 34 films : comptes du plan | P150 l.273 (B.3) ; P160 l.848-850 | à mesurer par A.3 (build précédent : `HI_1_10_0`, égalité non testable sur trames arrêtées) |
| A.4 | points de bombe hors de l'arbre : 4 films d'Assaut | parc v1.4.1 : 4 rejeux d'Assaut parmi les 43 « complets » en écart (One Bomb 3, Neutral Bomb 1) | 1118, Piste B (groupes d'écarts par variante) ; P160 l.851-852 | publication si le film les nomme : à mesurer par A.4 |
| A.5 | accord exact des points publiés avec le score personnel de l'API en hausse ; aucun joueur exact avant ne devient inexact | parc v1.4.1, rejeux « complets » : 98,42 % (8 065 / 8 194 joueurs) ; Grande bataille complets 78,07 % (356 / 456) ; HI_1_11_0 complets 69,33 % (226 / 326) ; tous rejeux 95,29 % (11 722 / 12 302) ; 23 rejeux où tous les joueurs en écart manquent exactement 100 points (pénalité de suicide ou de trahison, courbe qui ne recule pas) | 1118, Piste B, tableaux « Contre l'API » ; P150 B.3 (cause des 100 points) | état v1.5.0 par joueur (liste des joueurs exacts, nécessaire pour la règle « aucun exact ne devient inexact ») : à mesurer par A.5 sur les films du parc et les 13 films, avec l'export `verite/faits_13` et la base copiée |
| A.6 | une ligne ajoutée à la table des réponses rend un rejeu « à redécoder » | aucune mesure chiffrée ; état avant : l'ajout d'une ligne ne change aucune révision | P160 l.856-857 ; P160 §1.4 (« A.6 ») | test d'A.6 |
| A.7 | comparateur total ; effet attendu sur les 20 films : aucun | sans objet (cas d'égalité indéterminé) | P160 l.858-861 | témoin : équivalence des 20 films, à mesurer par A.7 |
| Gate A | accord au score personnel sans baisse pour aucun build | par build, parc v1.4.1, tous rejeux : HI_1_13_0 99,48 % ; HI_1_12_0 99,27 % ; HI_1_11_0 69,84 % ; HI_1_10_0 12,70 % ; HI_1_9_0 28,57 % | 1118, Piste B, ligne « Tous, par build » | à remesurer en état v1.5.0 en début de piste (A.1/A.5) |

### Piste B, drapeau et zones

| Item | Mesure attendue (P160) | Valeur avant | Source | Sinon |
|---|---|---|---|---|
| B.1 | cause prouvée par famille pour 16 portages à « au socle » faux : porteur précédent hors des 8 premières places ou sans humain nommé (7) ; variante à un drapeau publiée avec deux drapeaux d'équipe (4) ; résidu (5) | 16 = 7 + 4 + 5 (plan) ; au-delà de 8 joueurs, 85,7 % des joueurs à 2 s de l'API | P160 l.868-870 ; P150 l.343 (E.3) | causes : à mesurer par B.1 |
| B.3 | équipe du porteur lue par le film sur une carte à 6 socles | 25 portages sur 25 d'équipe inconnue sur le film 390b1de5 | P150 l.737-738 (découvertes) ; P160 l.872-873 | lecture : à mesurer par B.3 |
| B.4 | borne par build des compteurs de zones (octet signé qui avance de 32) | pas de valeur sur les anciens builds ; sur deux films à zones (dont 7344d24f), `zone_defensive_kills` égal à l'API pour 14 joueurs sur 14 | `.ai/annexes_decodeur_1118/changements_de_sortie.md` l.62 ; P160 l.874-875 | borne par build : à mesurer par B.4 |
| B.5 | temps de porteur à 2 s de l'API pour au moins autant de joueurs au-delà de 8 qu'en deçà | parc v1.4.1 : 88,11 % tous (1 542 / 1 750) ; 91,71 % à 8 joueurs ou moins (675 / 736) ; 85,50 % au-delà de 8 (867 / 1 014). Échantillon v1.5.0 (simulation sur les 186 films) : 88,23 % tous, 85,70 % au-delà de 8 | 1118, Piste E ; P150 l.345 ; `dec-v150-recup/e/rapport.md` l.42-43 | condition de B.5 : à mesurer après B.2 et B.4 |
| B.6 | deux compteurs de couverture (lettres lues, lettres en accord avec le rang publié) | parc v1.4.1 : 77 rejeux de Bases, `coverage.zones.letters` = 3 pour les 77 (lettres lues) ; « lettres en accord avec le rang » : compteur inexistant | 1118, Piste G | accord : à mesurer par B.6 |
| B.7 | cause de la zone au propriétaire de nom inconnu (un film, carte Recharge) | un film, non chiffré davantage | P160 §1.4 (ligne « zone au propriétaire de nom inconnu ») | cause : à mesurer par B.7 |
| Gate B | temps de porteur contre l'API sur les 186 films à drapeau : aucun porteur juste avant ne devient faux ; compteurs de zones sans baisse | 186 films à drapeau (dont 94 à plus de 8 joueurs) ; 120 portages manquants au-delà de 8 joueurs | 1118, Piste E | porteurs justes avant (liste par joueur) : à capturer par B.1 |

### Piste C, gestes

| Item | Mesure attendue (P160) | Valeur avant | Source | Sinon |
|---|---|---|---|---|
| C.1 | accroupi et glissade sur les versions récentes : accord avec la vitesse et la hauteur, publication à 90 % ; vérité = les 7 rejeux qui les publient déjà | parc v1.4.1 : 1 129 accroupissements et 1 039 glissades, dans 7 rejeux seulement (1 448 à zéro) | 1118, Piste H ; P160 l.888-890 | accord à 90 % : à mesurer par C.1 (20 films) |
| C.2 | bouton de saut sur les builds 1.9 à 1.11 : accord de 50 à 61 % ; publication à 90 % sur tous les builds | accord 50 à 61 % sur ces builds (63 % en moyenne selon le backlog) ; part fermée de la vue de contrôle : HI_1_13_0 99,94 %, HI_1_11_0 99,77 %, HI_1_10_0 96,13 %, HI_1_9_0 98,68 %, HI_1_8_0 90,46 % | P160 l.891-892, §1.4 (« bouton de saut à 63 % ») ; 1118, Piste H (tableau de couverture) | disposition de la vue : à mesurer par C.2 |
| C.3 | sens des quatre booléens d'un rechargement, contre les munitions lues | aucun nom prouvé (rechargements publiés depuis la v1.5.0 ; avant la v1.5.0 : 0 publié) | 1118, Piste H, dernier paragraphe ; P160 l.893-894 | à mesurer par C.3 |
| C.4 | table « son champ, notre champ, lequel reste » (sans code) | sans objet | P160 l.895-896 | écrit par C.4 |
| Gate C | poids ajouté au document sous 1 % sur 20 films | poids compressé médian d'un rejeu du parc v1.4.1 : 1 105 992 octets (min 41 044, max 8 579 753) ; poids de référence des 20 films en v1.5.0 : non relevé ici | 1118, Piste J | à relever sur les documents des 20 films (passe de référence de l'étape 0, `dec-v160-recup/etape0/equiv_ref`) au début de la piste |

### Piste D, répliques des Spartans

| Item | Mesure attendue (P160) | Valeur avant | Source | Sinon |
|---|---|---|---|---|
| D.1 | mêmes comptes que le témoin sur 3 films pour les genres 75 et 76 de la vue A | aucun champ publié (les genres sont traversés sans valeur) | P160 l.902-904 | comptes du témoin : à mesurer par D.1 |
| D.2 | un nom prouvé par réplique, corrélation indice du film et tables de vocalisation | 5 noms résolus à ce jour ; texte retrouvé dans le jeu pour 191 sons sur 201 | P160 l.905-908 ; P160 §1.4 (ligne « Répliques ») | à mesurer par D.2 |
| D.3 | couverture : lus, publiés, sans joueur, hors du match | 0 publié (calque absent) | P160 l.909-912 | à mesurer par D.3 |
| Gate D | poids ajouté sous 1 % ; budget des fixtures de contrat relevé et daté | voir poids de la piste C | P160 l.924-926 | budget : à relever par D.3 |

### Piste E, canaux neufs

| Item | Mesure attendue (P160) | Valeur avant | Source | Sinon |
|---|---|---|---|---|
| E.1 | nombre d'éliminations d'auteur bot dans `eliminations[]` (films à bots) et dans `scoreActions` ; vérité : le tableau du film | contexte : 1 124 entrées de roster sans identifiant (bots) sur le parc v1.4.1 ; comptes demandés non établis | 1118, section Jointures ; P160 l.930-934 | à mesurer par E.1 |
| E.2 | temps en VIP, plus long temps en VIP, meilleure série en VIP, publiés en courbes | non publiés ; compteurs d'objectif `vip_kills`, `vip_assists`, `vip_kills_as_vip` publiés depuis la v1.5.0 | `changements_de_sortie.md` l.14 ; P160 l.935 | vérité API des durées VIP : absente de la copie (aucune colonne) ; à établir par E.2 |
| E.3 | nom des options de variante à côté du hachage quand le film ou le jeu le donne | un hachage seul aujourd'hui ; nombre d'options sans nom : non établi | P160 l.936-938 | à mesurer par E.3 |
| E.4 | apparence enregistrée (armure, revêtement) lue et prouvée contre la personnalisation publique de l'API, sur 3 films | absente du document ; AUCUNE vérité dans la copie de l'API (pas de colonne d'apparence) | P160 l.939-941 | vérité à obtenir hors de la copie (appel réseau autorisé par le plan seulement) : à mesurer par E.4 |
| E.5 | contrôle sur pièces (zoom, coup de mêlée sans mort) | sans objet | P160 l.942-943 | écrit par E.5 |
| E.6 | victime appariée au plus proche : compteur d'ambiguïté | non compté | P160 l.944-945 | à mesurer par E.6 |
| Gate E | éliminations et statistiques VIP contre l'API sans baisse | éliminations, morts, assistances : accord exact 99,96 % / 99,93 % / 99,92 % sur les joueurs dont le film porte la valeur (parc v1.4.1) | 1118, Piste C | à remesurer en début de piste |

### Piste F, contrôles et petites entrées

| Item | Mesure attendue (P160) | Valeur avant | Source | Sinon |
|---|---|---|---|---|
| F.1 | `go test` d'un sous-paquet du clone passe avec le `GOWORK` engendré sous Windows | échoue avec l'espace de travail engendré (barres obliques), passe avec des barres inverses | P160 §1.4 (ligne `briques-workspace.sh:38`) ; P150 l.730-732 (découvertes) | essai à rejouer par F.1 |
| F.2 | huit décisions sans effet de `config/replay_corpus.toml` retirées | 8 avertissements du gate depuis le 2026-11-18 | P160 l.953-954 | liste des huit au journal par F.2 |
| F.3 | repli du voisinage des pièces montées compté sur les films à véhicules | non compté | P160 l.955-956 | à mesurer par F.3 (harnais) |
| F.4 | recroisement des temps forts « mode » sans action de score avec les huit noms d'action de la v1.5.0 | 3 410 (Drapeau), 691 (Total Control), 303 (Extraction) ; 23 683 événements de mode, sur 527 matchs, dont 21 353 de code 10 (parc v1.4.1, base copiée) | P160 l.957-959 ; 1118, Piste I | verdict : à mesurer par F.4 sur la base copiée (le détail des 58 lignes de `evenements_mode.tsv` est dans l'export du 2026-11-18, non recopié ici) |
| F.5 | cause du décalage de 35 s entre film et temps de jeu de l'API | 35 s (un cas, non expliqué) ; temps de jeu de l'API disponible dans la copie (`time_played_seconds`) | P160 l.960-961 | à mesurer par F.5 |
| F.6 | lâchers 2 à 4 images après une mort : un fait du film nomme le propriétaire | 4 cas ; pour les poses de mur : 7 poses déployées après la fin de vie de leur poseur, écart de 1 à 3 images (médiane 1) | P160 l.962-963 ; 1118, Piste F | à mesurer par F.6 |
| F.7 | pourquoi les 5 rejeux sans build ne portent aucun compteur nommé | 5 rejeux sans build, aucun joint à un joueur de l'API, aucun compteur nommé lisible | 1118, section Jointures et Piste C (ligne « sans build ») | verdict écrit par F.7 |

### Pistes V (vitesse), pour mémoire

Non demandées ici : les mesures « avant » de vitesse sont les relevés de référence de la v1.5.0 (V0.4 pour le relevé de
bout en bout, V0.5 pour le relevé de la brique), qui lisent ces 13 films et leurs faits.

## Fichiers produits

Tout sous `C:/Users/Guillaume/Downloads/Scripts/dec-v160-recup/verite/` (hors dépôt) :

- `api_copie.duckdb` : copie de la base partagée (lecture seule), 483 143 680 octets.
- `donnees/titles/halo_infinite/warehouse/shared_matches_v2.duckdb` : seconde copie de la précédente, racine de données
  de l'export (lue en lecture seule par l'outil).
- `faits_13/` : 13 `<court>.facts.json` et 13 `<court>.oracle.json` ; CONTIENNENT des identifiants de joueurs : ne
  jamais les verser au dépôt ni les citer.
- `levelup_export.exe` : binaire de l'outil, construit depuis `LevelUp-wt-dec160`.
- `logs/` : dossier de journaux de la racine (vide).
- `avant.md` : ce fichier (sans identifiant de joueur).

Hors de `verite/` : copie de 03b7bbb6 et 221dd082 (morceaux et manifestes) sous `LevelUp-wt-dec160/data/cache`.
