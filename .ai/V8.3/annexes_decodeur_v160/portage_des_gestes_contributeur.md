# Portage des gestes du contributeur dans le décodeur de film

Plan de lecture seule. Aucun code écrit, aucun test lancé, aucun film décodé. Le dépôt d'origine n'a pas été modifié.

## 0. En résumé

- Chez le contributeur, les gestes touchent **58 fichiers** : 20 de production neufs, 20 de production modifiés, 7 tests, 11 sondes de recherche. Une grande partie est inutile au portage (voir § 3).
- La **lecture** des gestes est presque déjà au dépôt d'origine. Les lecteurs des messages de la vue A (lancer amorcé, coup de mêlée) et le champ des drapeaux de l'entrée de contrôle y existent, identiques à ceux de la version 1.3.6. Il manque seulement de garder ce qu'ils lisent. Le portage de la lecture tient en une vingtaine de lignes de grammaire.
- Le contributeur déduit beaucoup : la glissade (sprint, vitesse, 6 seuils), la place de contrôle (lue dans les tirs, alors que le dépôt d'origine la tient déjà par le roster), le remplacement des sauts dérivés. Rien de cela ne se porte.
- Condition de publication : **la cadence des commandes** est le seul fait du film qui sépare les films justes des films fautifs. La version du film et la version du jeu ne les séparent pas (§ 7).
- Sort de la glissade : **hors du portage**. Ni son code ni le film n'offrent un fait plus solide que la vitesse (§ 4.2).
- Découpage : **7 items de code** et une note de crédit. Deux files parallèles : A = 1, 2, 3, 7 ; B = 4, 5, 6 (§ 8).
- Ce que son application lit change de nom pour le saut, l'accroupi, le coup, le lancer et le rechargement (§ 9). Son application ne lit pas le déplacement brut de la manette.

## 1. Suppositions et manques, en tête

- Supposé : les décisions du mainteneur du brief (bits, glissade, règles du dépôt) sont fermes. Je ne les rediscute pas.
- Supposé : « version actuelle du jeu » veut dire les films dont les commandes sont enregistrées à chaque tick de 60 Hz. C'est le sens que donne `c123.md` (mesure C.2).
- Supposé : les noms de genre `crouchCommand` et `jumpCommand` et le nom de calque `gestures` sont **proposés** : le mainteneur les confirme ou les change (§ 10).
- Manque : je n'ai pas pu rejouer une seule mesure. Tout chiffre vient de `c123.md`, d'un commentaire du contributeur ou du code lu.
- Manque : le client du contributeur n'a été lu que sur les fichiers qui touchent aux gestes (§ 9).
- La copie du contributeur a été lue en clone local à la révision `a66b791`. Rien n'y a été exécuté.

## 2. Conventions

- `O/` : dépôt d'origine, branche `feat/v1.6.0`, révision `ea8156c`. Chemins relatifs à la racine du dépôt.
- `C/` : copie du contributeur, `third_party/film-decoder/`, dépôt `llampwall/installation-08`, révision `a66b791`.
- `W/` : client du contributeur, `web/src/`, même dépôt.
- `c123.md` : mesures du mainteneur sur 22 films (builds 1.4.1 à 1.13.0).
- **vérifié** : ligne relue ou fichier comparé à la version 1.3.6 (départ du contributeur) par `git show`. **déduit** : raisonné sans preuve.
- Le départ de la copie est la version 1.3.6, la tête d'origine est la 1.6.0 en cours (schéma de document 126, schéma des faits 30) : vérifié `O/replay/document_schema.go:24`, `O/replay/filmfacts_fichier.go:263`.

## 3. Inventaire exact de son code des gestes

### 3.1 Fichiers de production neufs

| Fichier `C/` | Lignes | Ce qu'il fait | Dépend de | Sort |
|---|---|---|---|---|
| `internal/grammar/boutons_de_controle.go` | 126 | Plie les entrées de contrôle (bits accroupi et saut, couple analogique) en changements par place. Silence de 300 ms = état inconnu (`types/grammar_boutons.go:41`). | `Observation.VueControleHook`, tenu par le canal du tir continu (`tir_continu.go`) | Réécrire (item 1). Le seuil de 300 ms disparaît : le dépôt d'origine a la règle des trous. |
| `types/grammar_boutons.go` | 82 | Constantes de bits : accroupi `1<<0` (l.30), saut `1<<2` (l.32). Type `ControlInputChange` (l.48). | rien | Garder les deux bits, nommés. Le reste non porté. |
| `internal/grammar/vue_a_actions.go` | 48 | Garde les messages de la vue A des genres 0, 38, 39, 40 (`genreDAction`, l.23). | `lecture/actions.go` | Réécrire (item 4) : seuls 39 et 40 manquent, et le dépôt d'origine range le joueur, pas les références. |
| `internal/grammar/actions_du_film.go` | 80 | Balayage à part des têtes (`ScanFilmActions`, l.20), base des domaines de référence (`baseDuDomaine`, l.70). | `distribuerLesTetesSeules` | Ne pas porter : un canal de la marche des trames suffit, comme pour les rechargements (`O/internal/grammar/movement_states.go:215`). |
| `internal/grammar/lecture/actions.go` | 37 | Type du message (16 octets, gelé par `tailles_test.go`). | rien | Réécrire (item 4). |
| `types/grammar_actions.go` | 68 | Genres 0, 38, 39, 40 (l.37-39) ; lectures de l'arme désirée (hors sujet). | rien | Garder les noms des genres 39 et 40. L'arme désirée est hors périmètre. |
| `replay/document_gestes.go` | 63 | Liste `actions[]` : `t`, `ms`, `slot`, `kind` (`PlayerAction`, l.15). `buildPlayerActions` (l.39). | `fullRateTimeUnitUS` (pleine cadence) | Réécrire (item 5). |
| `replay/postures_de_controle.go` | 218 | Règle accroupi / glissade / saut, 7 constantes (l.39-58), couverture `ControlStanceCoverage`, lecture des commandes (`lireLesCommandes`, l.142). | places, vitesse, sprint, véhicules | Ne pas porter. Le repli en pressions sert d'idée à l'item 3. |
| `replay/postures_de_controle_vies.go` | 238 | Pose la commande sur la vie (`resoudre`, l.54) ; classe la glissade (`debutDeGlissade` l.153, `finDeGlissade` l.171). | `postures_cinematique.go` (vitesse du plein débit) | Ne pas porter. |
| `replay/postures_cinematique.go` | 104 | Vitesse au sol et verticale lues dans le plein débit. | pleine cadence | Ne pas porter : ne sert qu'à la glissade. |
| `replay/build_postures.go` | 181 | Passe d'assemblage : pose, et retire les sauts dérivés que la commande remplace (`filtrerLesSautsDerives`, l.122). | tout ce qui précède | Ne pas porter. Les deux sauts restent publiés (§ 5). |
| `replay/places_de_controle.go` | 247 | Lit la place de contrôle de chaque joueur dans les tirs. | `grammar.FireEvent` | Ne pas porter : le dépôt d'origine tient déjà cette règle (§ 5). |
| `replay/stick_de_deplacement.go` | 148 | Pose le couple analogique brut dans trois colonnes du plein débit (`poserLeStick` l.65, `amorceDuStick` l.104). | pleine cadence, places | Réécrire (item 7). |
| `replay/filmfacts_boutons.go` | 60 | Écrit les changements de boutons dans le fichier de faits. | codec des faits | Réécrire selon le codec d'origine (items 1, 4, 7). |
| `replay/filmfacts_actions.go` | 93 | Écrit les messages d'action et l'arme désirée. | codec des faits | Réécrire (item 4). L'arme désirée est hors périmètre. |
| `replay/filmfacts_complement.go`, `filmfacts_schema23.go` | 48, 17 | Découpage du fichier de faits pour tenir 500 lignes. | rien | Voir la règle de taille du § 8. |
| `replay/document_actions_joueur.go` | 195 | Couverture des actions (l.74-100), plus arme désirée, dégâts, causes de mort. | rien | Seule la couverture des gestes est utile (item 5). Dégâts et causes de mort font doublon avec `O/replay/document_damage.go`. |
| `internal/facts/fallback/registre_replay_postures.go` | 31 | Repli du saut dérivé. | `build_postures.go` | Ne pas porter. |
| `replay/tracks_plein_debit.go` | 130 | Colonnes de la pleine cadence. | rien | Hors du portage des gestes : autre piste (vérifié, aucun `FullRate` dans `O/replay/` hors tests, `c4_e5_recoupements.md`). |

### 3.2 Fichiers de production modifiés (20)

`replay/document.go` (57 lignes de diff avec la 1.3.6), `coverage.go` (7), `layers.go` (8), `build.go` (19), `options.go` (39), `film_inputs.go` (17), `film_scan_mouvement.go` (29), `observe.go` (11), `filmfacts_fichier.go` (51), `filmfacts_codec.go` (5), `document_stances.go` (28), `build_inventaire.go`, `document_tracks.go`, `internal/grammar/vue_a_lecture.go` (5), `lecture/paquet.go` (2), `movement_states.go` (21), `tir_continu.go` (13), `lecteur.go` (3), `distribuer_tetes.go` (1), `rev.go` (2). Vérifié par `diff` contre la 1.3.6.

Constat utile : **les lecteurs de la vue A et de la vue de contrôle n'ont pas été modifiés.** `vue_a_charges_armes.go`, `vue_a_charges_execution.go` et `frame_vue_controle.go` sont identiques à la 1.3.6 chez lui, et identiques à la 1.3.6 à la tête d'origine (diff de 0 ligne des deux côtés, vérifié). Ses gestes ne demandent aucune nouvelle lecture de bits.

### 3.3 Tests

- Neufs, utiles comme source de cas : `replay/document_gestes_test.go` (45), `replay/postures_de_controle_test.go` (157), `replay/stick_de_deplacement_test.go` (58), `internal/grammar/boutons_de_controle_test.go` (107), `vue_a_actions_test.go` (89), `actions_du_film_test.go` (80), `replay/document_actions_joueur_test.go` (83).
- Sondes de recherche (balise `research`), à lire, non à porter : `replay/postures_de_controle_research_test.go` (401), `stick_de_deplacement_research_test.go` (171), `actions_film_research_test.go` (92), `actions_masques_research_test.go` (162), `actions_registre_research_test.go` (57), `actions_vue_a_research_test.go` (296), et cinq `internal/grammar/stick_*_research_test.go`.
- `replay/actions_masques_research_test.go` (en-tête) établit que `i29` et `i62` ne sont annoncés par aucun masque de la build 1.13.0 : « zéro record sur 2,25 M ». Cela recoupe le commentaire d'origine `O/replay/document_stances.go:59-62`. Un seul film de cette build a été sondé (déduit : « 2,25 M » est un compte d'un film).

## 4. Geste par geste

### 4.1 Accroupi

- **Source** : bit de valeur 1 des drapeaux `+0x14` de l'entrée `kind 0` de la vue de contrôle (`C/types/grammar_boutons.go:30`). Le champ s'appelle `Drapeaux` dans `ChampsDeControle` (`O/internal/grammar/frame_vue_controle.go:81`). Il est déjà lu à la tête d'origine et jeté (`O/internal/grammar/commandes_de_controle.go:12-14`). Vérifié.
- **Lecture** : état du bit à chaque entrée lue. Un champ absent d'une entrée qui porte le bloc vaut zéro (`C/types/grammar_boutons.go`, en-tête). Piège : `Drapeaux` vaut -1 quand le champ est absent ; en Go, `-1 & 4` est non nul. Le contributeur teste l'absence avant le masque (`C/internal/grammar/boutons_de_controle.go:70`). Le port doit faire de même. Vérifié.
- **Déduction du contributeur** : une pression est un accroupi, sauf si un sprint lu est actif ou fini depuis au plus 250 ms (alors glissade). Constantes `C/replay/postures_de_controle.go:39-58`. Ce classement ne se porte pas.
- **Ce qui est lu, et seulement cela** : « le bouton d'accroupissement est tenu de t0 à t1 ». Le jeu n'a qu'un bouton pour l'accroupi et la glissade (commentaire `C/types/grammar_boutons.go`, `c123.md` C.1 : « même bit pour les deux »). Le nom honnête est donc une **commande** : `crouchCommand`, sur le modèle de `moveCommand` (`O/types/grammar_commandes.go:14`).
- **Forme publiée** : `stances[]`, genre `crouchCommand`, `t0`, `t1` (frames) et `ms0`, `ms1` (millisecondes depuis la frame 0, comme chez le contributeur, `C/replay/document_stances.go:56-57`).
- **Mesure** : bit posé et vitesse compatible 95,39 % (91 829 / 96 269), bit nul à vitesse debout normale 99,79 % ; sous 90 % sur un seul film des 22 (`c123.md`, C.1). Vérité physique, pas la vérité du plan : réserve dite par le mainteneur.
- **Couverture** : lus, publiés, écartés : voir item 3.
- **Tests du contributeur** : `C/replay/postures_de_controle_test.go` (pressions, relâchements, silence). Cas repris, règles réécrites.

### 4.2 Glissade : hors du portage

- Le contributeur déduit la glissade du sprint lu (`i57`) et de la vitesse au sol : début = pression pendant ou juste après un sprint (`C/replay/postures_de_controle_vies.go:153`), fin = première vitesse sous 2,45 (`C/replay/postures_de_controle.go:52`), plafond 1600 ms (l.54), atterrissage d'une pression en l'air (l.40-50).
- Mesure du mainteneur : 74 % des séries rapides atteignent le pic de vitesse, aucune vérité indépendante (`c123.md`, C.1). Sous le seuil de 90 %.
- Cherché dans son code un fait plus solide : aucun. `C/internal/grammar/components_biped_spartan.go:87-132` lit le composant `i62`, absent des builds récentes (`C/replay/actions_masques_research_test.go`, en-tête). Aucun autre bit, message ou champ ne désigne la glissade dans `C/` (recherche sur « glissade » et « slide » : sondes de recherche et lecteur d'`i62` seulement). Vérifié sur le code lu.
- Cherché dans le film : `c123.md` établit que le sprint est actif au début de 1 série sur 893 « glissade » et de 3 sur 793 « accroupi ». Le sprint ne distingue donc pas. Vérifié sur `c123.md`.
- Piste examinée et écartée : son client note que les rechargements écrits deux fois au même instant coïncident avec « 157 glissades » (`W/features/match-replay/model/reloadStarts.ts`, en-tête). C'est circulaire (les glissades sont les siennes, déduites) et non spécifique (mêlées, équipements, poussées de propulseur coïncident aussi). Déduit.
- Conséquence : le dépôt d'origine ne publie pas `slide` pour ces films. L'intervalle `crouchCommand` couvre la glissade et l'accroupi. Un client qui veut la glissade la déduit de la vitesse, à ses risques.

### 4.3 Saut

- **Source** : bit de valeur 4 (`C/types/grammar_boutons.go:32`). Tranché par le mainteneur : le contributeur a raison (`c123.md`, C.2). Il corrige aussi le commentaire d'origine (`O/internal/grammar/commandes_de_controle.go:12-14`, « bit de poids 4 »).
- **Nature** : bouton enfoncé, pas saut effectué. 34 à 78 % des séries portent un départ du sol (`c123.md`, C.2). Le saut dérivé de la vitesse verticale (`jumpDerived`, `O/types/grammar_mouvement.go:77`) est un autre fait : le décollage mesuré. **Les deux restent publiés** ; le contributeur remplaçait l'un par l'autre (`C/replay/build_postures.go:122`, compteurs `Replaced`, `Contradicted`), ce qui ne se porte pas.
- **Mesure** : 98,01 % (5 314 / 5 422) sur les films à 60 Hz, 51,6 à 61,3 % sur les 4 films à 30 Hz (`c123.md`, C.2). D'où la porte de cadence (§ 7).
- **Forme publiée** : `stances[]`, genre `jumpCommand`, avec `ms0` et `ms1`. L'instant du bouton est `ms0`. Choix : un seul mécanisme de pliage (la commande tenue, avec ses trous), et pas une seconde machine pour des instants.
- **Population mesurée** : « départs à couverture continue » (une entrée à chaque pas sur ±100 ms). Un appui dont le début suit un trou de lecture sort de la population mesurée : **à écarter et à compter** (`afterHole`). Déduit de la définition de l'accord dans `c123.md`.
- **Véhicules** : le contributeur ne pose rien pendant un épisode de véhicule (`C/replay/postures_de_controle.go:28-31`). La mesure d'accord s'appuie sur la vitesse d'un corps à pied (déduit). Reprendre la règle : écarter pendant `vehicles[].rides`, compter `inVehicle`.

### 4.4 Coup de mêlée

- **Source** : message `biped_melee_initiate`, genre 40 de la vue A (`C/types/grammar_actions.go:39`). Lecteur d'origine `O/internal/grammar/vue_a_charges_armes.go:87` (`chargeCorpsACorps`) : il lit la tête, l'arme et la variante, puis **l'index de joueur de l'auteur** (`consumeGate0R(br, 5)`) qu'il jette. Vérifié.
- **Lecture** : l'instant du paquet qui porte le message. Aucun seuil, aucune déduction. Le contributeur range la référence 0 (l'unité qui frappe, `C/internal/grammar/vue_a_actions.go`). Le dépôt d'origine range pour les rechargements l'index de joueur (`O/internal/grammar/vue_a_charges.go:204-216`) : même choix ici.
- **Forme publiée** : `gestures[]`, genre `melee`, `t`, `ms`, `slot`. Un coup part à cet instant, qu'il touche ou non (`C/replay/document_gestes.go`, en-tête de genre).
- **Mesure** : aucune. Pas de vérité posée. Voir § 10.
- **Tests du contributeur** : `C/replay/document_gestes_test.go` (projection, filtre des vies, identité de la couverture) ; cas repris.

### 4.5 Lancer amorcé

- **Source** : message `biped_throw_initiate`, genre 39 (`C/types/grammar_actions.go:37`). Lecteur d'origine `O/internal/grammar/vue_a_charges_execution.go:87` (`chargeLancerInitie`) : même schéma, l'index de joueur est lu et jeté (`consumeGate0R(br, 5)`). Vérifié.
- **Forme publiée** : `gestures[]`, genre `throwStart`. Le projectile part plus tard : il reste dans `grenades[]` (`O/replay/grenades.go`, champs `t`, `slot`). Les deux sont des faits distincts : l'amorce et la sortie.
- **Mesure** : aucune. Le délai entre l'amorce et la naissance du projectile n'est établi nulle part (§ 10).

### 4.6 Rechargement

- **Source** : message `weapon_reload`, genre 38. Déjà lu et publié à la tête d'origine (`O/replay/document_reloads.go:24-91`, `reloads[]`, version 1.5.0). Même message source que le contributeur.
- **Différence qui compte** : le dépôt d'origine **retire les copies** d'un même message dans une même trame (`O/internal/grammar/rechargements.go`, `identiteDeRechargement`) et ne les compte que dans `coverage.reloads.copies`. Le client du contributeur **lit ces copies** : un instant écrit une seule fois est un début de rechargement, un instant écrit deux fois ou plus est une coupure du rechargement en cours (`W/features/match-replay/model/reloadStarts.ts`, l.1-18 et boucle l.28-35 ; `W/features/match-replay/sound/reloadSound.ts:39`). Il chiffre : sur sept matchs, 894 des 1 289 instants uniques sont suivis d'un chargeur qui se remplit ; les instants multiples coïncident avec 81 sur 81 activations d'équipement, 391 mêlées, 157 glissades. Ces chiffres sont les siens, non rejoués (déduit).
- **Conséquence** : publier chaque fait une fois veut dire garder `reloads[]` tel qu'il est et y **ajouter le nombre de copies** de chaque ligne (`copies`) et l'instant à la milliseconde (`ms`). Le début et la coupure se lisent alors comme chez lui. Aucun fait n'est publié deux fois.
- **Booléens du message** : aucun n'est nommé (`c123.md`, C.3). Non portés.

### 4.7 Déplacement brut de la manette

- **Source** : couple analogique de l'entrée de contrôle (`ChampsDeControle.Analogique`, deux scalaires de 6 bits, 31 = zéro exact, `O/internal/grammar/frame_vue_controle.go:81-83`, `O/internal/grammar/commandes_de_controle.go:24`). Déjà lu à la tête d'origine.
- **Déjà publié** : la commande tenue (`moveCommand`), en intervalles, sans les valeurs (`O/replay/document_stances_commandes.go`, `O/FORMAT.md:386`). Le brut est un autre fait : les valeurs.
- **Lecture du contributeur** : un échantillon à chaque changement du couple ou de sa connaissance (`C/replay/stick_de_deplacement.go:48`), inconnu après 300 ms de silence, amorce à la naissance de la vie (l.104). Publié dans des colonnes différentielles du plein débit (`stickT`, `sx`, `sy`).
- **Repère** : « `sx` suit le cap (62 = plein avant), `sy` est à +90 degrés » : affirmé par un commentaire et une sonde (`C/replay/stick_de_deplacement_research_test.go`). Non rejoué (déduit).
- **Lecteur** : le client du contributeur **ne lit pas** `stickT`, `sx` ni `sy` (recherche sur tout `W/` : aucune occurrence hors lignes sans rapport). Vérifié.
- **Décision proposée** : dernier item, facultatif (item 7), en calque propre `sticks[]` sans dépendre de la pleine cadence.

## 5. Ce qui existe déjà à la tête d'origine, et qui garde quoi

| Fait | À la tête d'origine | Chez le contributeur | Reste |
|---|---|---|---|
| Rechargement | `reloads[]` : `t`, `place` ; copies retirées et comptées (`O/replay/document_reloads.go:24-91`) | `actions[]` genre `reload`, `ms`, `slot`, copies gardées | **D'origine**, avec `ms` et `copies` ajoutés (item 6). |
| Commande de déplacement | `stances[]` genre `moveCommand` en intervalles (`O/replay/document_stances_commandes.go:130-156`) | colonnes brutes `sx`, `sy` | **Les deux**, deux faits (intervalle / valeurs). |
| Saut | `jumpDerived` : intégrale de la vitesse verticale, hauteur 0,85 m (`O/replay/document_stances.go:12-15` du commentaire d'origine) | impulsion du bouton | **Les deux**, deux faits (décollage / bouton). |
| Accroupi, glissade | `crouch` et `slide` lus dans `i29` et `i62`, vides sur les films récents (`O/replay/document_stances.go:59-62`) | `crouch` et `slide` tirés du bouton | **Sous des noms distincts** : `crouch` d'origine (composant), `crouchCommand` (bouton). Pas de `slide` du bouton. |
| Place de contrôle | `tireursParPlace` : le roster, par intervalle de présence et siège (`O/replay/tirs_par_place.go:42-58`), déjà utilisé par `moveCommand` | `places_de_controle.go` : la place lue dans les tirs | **D'origine.** Le fichier de 247 lignes du contributeur ne se porte pas. |
| Coup de mêlée, lancer amorcé | lus, jetés (`O/internal/grammar/vue_a_charges_armes.go:87`, `vue_a_charges_execution.go:87`) | `actions[]` | **Neufs** (items 4 et 5). |
| Visée, zoom | `document_aim.go`, champ `s` des points (`c4_e5_recoupements.md`, tableau 2) | idem | Hors périmètre, déjà couvert. |
| Dégâts de mêlée | `damage[]`, sans étiquette de classe mêlée (`O/replay/document_damage.go`, `c4_e5_recoupements.md`) | idem | Hors périmètre. Le geste `melee` donne l'instant du coup, pas son résultat. |

## 6. Écart entre son point de départ et la tête d'origine

Écarts mesurés par `diff` entre la 1.3.6 et la tête d'origine, pour les fichiers que les gestes touchent. Vérifié.

| Fichier `O/` | Lignes de diff 1.3.6 -> tête | Lecture pour le portage |
|---|---|---|
| `internal/grammar/vue_a_charges_armes.go`, `vue_a_charges_execution.go`, `frame_vue_controle.go` | 0 | Se portent tels quels : rien à porter, ils existent. |
| `internal/grammar/tir_continu.go` | 40 | Réécrire : le collecteur porte maintenant des « natures » de bit (`natureDeBit`, l.62-71), dont la commande de déplacement. Les boutons s'y ajoutent comme deux natures de plus, sans second collecteur. |
| `internal/grammar/movement_states.go` | 158 | Réécrire : 489 lignes, plafond presque atteint. Extraire avant d'ajouter. |
| `internal/grammar/lecture/paquet.go` | 81 | Réécrire : **exactement 500 lignes**. Extraire un type dans un fichier neuf avant d'ajouter un champ. |
| `internal/grammar/vue_a_lecture.go`, `lecteur.go`, `distribuer_tetes.go` | 29, 13, 7 | Ajouter un cas (`recueillirLeMessage`, l.164-190), un champ de lecteur, une copie (`rangerLaVueA`, l.33-48). |
| `replay/document.go` | 120 | Réécrire : 460 lignes. Ajouter `Gestures` près de `Reloads` (l.297). |
| `replay/options.go`, `film_inputs.go`, `film_scan_mouvement.go`, `observe.go` | 61, 69, 30, 16 | `options.go` fait 497 lignes : extraire avant d'ajouter. |
| `replay/filmfacts_fichier.go` | 29 | 499 lignes. La constante `SchemaDesFaits` (l.263) et sa documentation y vivent : sortir la documentation du saut de schéma, comme le fait le contributeur (`C/replay/filmfacts_schema23.go`). |
| `replay/layers.go`, `coverage.go`, `build.go` | 48, 28, 53 | Ajouter des lignes de table, un champ, un appel (`O/replay/build.go:102-103`). |
| `replay/document_stances.go` | 19 | Ajouter `ms0`, `ms1` (`C/replay/document_stances.go:56-57`). Corriger le commentaire l.59-62 : « ne se lisent sur aucun film » est vrai des composants, faux de la vue de contrôle. |
| `replay/document_schema.go` | n/a (absent en 1.3.6) | La constante a changé de fichier : `O/replay/document_schema.go:24`. |

Se porte tel quel : les deux constantes de bit, le nom des genres 39 et 40, les cas de test à reprendre. Doit être réécrit : tout le reste, parce que le code a bougé (collecteur à natures, codec de faits, couverture, tailles de fichiers).

## 7. Condition de publication : la cadence des commandes

Le fait du film : **le pas des entrées de contrôle**, c'est-à-dire l'écart entre deux entrées lues du même index. Il vaut 16,7 ms à 60 Hz et 33,3 ms à 30 Hz (`c123.md`, C.2, « vérifié »).

Pourquoi pas la version du film ni la build, vérifié sur les pièces :

| Film | Build | Version du film | Format | Cadence | Accord du saut |
|---|---|---|---|---|---|
| un film de 1.8 | HI_1_8_0 | 37 | n/d | 60 Hz | 93,1 % |
| un film de 1.9 | HI_1_9_0 | 38 | n/d | 30 Hz | 61,3 % |
| `1c4c63c2` | HI_1_10_0 | 39 | 24 | 60 Hz | 95,5 % |
| `084a804d` | HI_1_10_0 | 39 | 24 | 30 Hz | 51,9 % |
| un film de 1.11 | HI_1_11_0 | 40 | n/d | 30 Hz | 51,6 % |
| 13 films de 1.12 et 1.13 | HI_1_12_0, HI_1_13_0 | n/d | n/d | 60 Hz | 98,0 à 98,5 % |

Sources : `c123.md` (cadence, accord), `builds.tsv` (builds), `films_du_releve_final.tsv` (versions 37 à 40), journaux de sonde (`format=24` pour `084a804d` et `1c4c63c2`).

Deux films de même build et de même format (`1c4c63c2` et `084a804d`) ont deux cadences. La version majeure 39 les réunit aussi. **Ni la build ni la version du film ne séparent les films justes des films fautifs : seule la cadence le fait.** Vérifié.

Règle proposée :
- Le décodeur mesure le pas des entrées pendant la marche (item 2) : l'écart le plus fréquent entre deux entrées lues du même index, sans trou entre elles.
- Les calques de commande neufs (`crouchCommand`, `jumpCommand`, `sticks`) ne sont publiés que si ce pas est au plus `PasDeCommande60HzMaxUS`. Valeur proposée : 25 000 µs. Raison écrite : milieu entre 16 667 et 33 333 µs, les deux seuls pas observés sur 22 films. À vérifier sur des films hors corpus.
- Film sans pas mesurable (vue de contrôle presque jamais lue : trois films de 1.4.1 et de versions 31 et 33 d'après `O/FORMAT.md:393`) : porte fermée, comptée.
- Le nombre minimal d'écarts pour qu'un pas soit mesurable est **un seuil à nommer et à fixer par mesure** : je n'en pose pas ici (§ 10).
- `moveCommand` n'est pas touché : il est déjà publié, sans porte.
- `melee`, `throwStart` et le rechargement sont des messages de la vue A, pas de la vue de contrôle. Aucune mesure ne les rattache à la cadence. Par défaut ils ne sont **pas** derrière la même porte. Le mainteneur peut décider de les y mettre : c'est une ligne.
- Accroupi : mesuré à 95 % sur toutes les builds, 30 Hz compris (`c123.md`, C.1). La décision du mainteneur le met derrière la porte avec les autres gestes. Coût : 4 films sur 22 perdent l'accroupi sans nécessité mesurée. Je suis la décision ; ce coût est dit.

## 8. Découpage en items de code

Règles : un item = une séance et un worktree. Un seul saut de schéma pour tous les items (document 126 -> 127, faits 30 -> 31) et une seule montée de la révision de grammaire : le premier item fusionné la pose, les autres l'ajoutent à la même version. L'empreinte de la grammaire se refige à chaque item qui change la lecture.

Règle de taille : fichiers à 500 lignes au plus. Quatre fichiers sont au plafond ou presque (`lecture/paquet.go` 500, `filmfacts_fichier.go` 499, `options.go` 497, `movement_states.go` 489) : chaque item qui y touche commence par extraire.

### Item 0 (hors code) : crédit

`NOTICE.md` du dépôt d'origine : une ligne qui crédite le contributeur, dépôt `llampwall/installation-08`, pour la découverte des bits d'accroupi et de saut, l'idée des gestes de la vue A et la mesure des copies de rechargement. `GLOSSARY.md` : `crouchCommand`, `jumpCommand`, `gestures`. À faire avec l'item 3.

### Item 1 : lire accroupi et saut comme commandes tenues (aucune sortie nouvelle)

- **Modifie** : `internal/grammar/tir_continu.go` (deux natures de plus dans `natureDeBit`, l.62-71 ; `fermer`, l.143-170), `internal/grammar/commandes_de_controle.go` (`commande`, l.37-43, généralisée aux trois natures ; garde d'absence de `Drapeaux` ; commentaire l.12-14 corrigé), `types/grammar_commandes.go` (`ControlHold.Kind`, constantes des deux genres), `replay/filmfacts_commandes.go` (écrire `Kind`), `replay/document_stances_commandes.go:77` (`publier` filtre sur la nature : sortie inchangée).
- **Ce qui change dans le document** : rien. Le fichier de faits change (`SchemaDesFaits` 31).
- **Preuve** : tests de la grammaire avec les cas de `C/internal/grammar/boutons_de_controle_test.go` (107 lignes) ; test du piège `Drapeaux = -1` ; documents d'assemblage **identiques avant et après** sur les 8 films de `O/replay/testdata/assembly_*.golden` ; comptes de `moveCommand` identiques.
- **Dépend de** : rien.

### Item 2 : mesurer la cadence des commandes

- **Modifie** : `internal/grammar/tir_continu.go` (histogramme des écarts entre entrées d'un même index, dans `fermer`), `types/grammar_commandes.go` (`ControlHoldStats.StepUS`, constante `PasDeCommande60HzMaxUS` avec sa raison), `replay/filmfacts_commandes.go`, `replay/document_stances_commandes.go` (le champ passe dans la couverture).
- **Ce qui change dans le document** : `coverage.stances.control` gagne `stepUs` et `fullCadence` (booléen). Aucun calque.
- **Preuve** : sonde jetable (balise `research`, comme `c123.md`) : le pas de chacun des 22 films doit retrouver la table du § 7 (4 films à 33 333 µs, les autres à 16 667 µs). Cette mesure fixe aussi le nombre minimal d'écarts du § 7.
- **Dépend de** : 1.

### Item 3 : publier `crouchCommand` et `jumpCommand`

- **Crée / modifie** : `replay/document_stances_commandes.go` (156 lignes ; ajoute la porte de cadence, les natures, l'écart des appuis après un trou, l'écart en véhicule ; extraire un fichier si 500 approchent), `replay/document_stances.go` (champs `ms0`, `ms1` ; commentaire l.59-62), `types/grammar_commandes.go` (noms des genres), helper commun `vieDeLaPlace(place, frame)` extrait de `publier` (`O/replay/document_stances_commandes.go:77-98`) : ce sera la deuxième copie, le troisième appel vient aux items 5 et 7, il faut donc le centraliser maintenant et poser un garde-rail.
- **Ce qui change dans le document** : `stances[].kind` prend `crouchCommand` et `jumpCommand` ; `stances[].ms0`, `ms1` (optionnels) ; `coverage.stances.control` : pour chaque genre, lus / publiés / écartés (`noTrack`, `noPlayer`, `inVehicle`, `afterHole`, `gated`). Saut de schéma 127. À poser par le mainteneur : `replay/layers.go` (la ligne `stances`, l.177, ne change pas de nom), chronique du schéma (`replay/document_chronicle.go`), `O/FORMAT.md` (réécrire le texte l.386-396 : « the jump button is not published » devient faux ; la raison des 50-61 % est la cadence), empreinte de forme (`replay/testdata/document_shape.golden`) et son jumeau côté hôte (`internal/filmcorpus/document_shape_jumeaux_test.go`, cité par `O/replay/document_shape_test.go:15-16`), goldens d'assemblage, `replay/contract_fixtures_budget_test.go`.
- **Preuve** : tests sans film (pressions, relâchements, trous, porte fermée, véhicule). Mesure d'accord sur les films récents, comme `c123.md` : accord du saut contre départs du sol (attendu >= 98 % à 60 Hz, 5 314 / 5 422 à reproduire) ; accord de l'accroupi contre la vitesse (attendu 95,4 % et 99,8 %) ; sur les 4 films à 30 Hz le calque est absent et compté.
- **Dépend de** : 1, 2.

### Item 4 : lire l'auteur des messages de mêlée et de lancer

- **Crée** : `internal/grammar/lecture/geste.go` (type du message, 16 octets, ajouté à `tailles_test.go`), `types/grammar_gestes.go` (`GestureRead`, `GestureStats`), `internal/grammar/gestes.go` (canal de la marche, même règle d'identité et mêmes copies comptées que `rechargements.go` : deuxième copie du motif, centralisation à la troisième).
- **Modifie** : `internal/grammar/vue_a_charges_execution.go:87` et `vue_a_charges_armes.go:87` (garder l'index de joueur au lieu de le jeter), `internal/grammar/lecteur.go` (champ), `vue_a_lecture.go:164-190` (cas des genres 39 et 40), `lecture/paquet.go` (500 lignes : extraire d'abord), `distribuer_tetes.go` (`rangerLaVueA`), `movement_states.go:215-248` (canal ajouté ; 489 lignes : extraire d'abord), `replay/filmfacts_*.go` (section des gestes dans le complément ; `filmfacts_rechargements.go` donne le modèle), `replay/film_inputs.go`, `options.go` (497 : extraire d'abord), `film_scan_mouvement.go`, `observe.go`.
- **Ce qui change dans le document** : rien (lecture seule). Fichier de faits : `SchemaDesFaits` 31 ; révision de grammaire montée.
- **Preuve** : tests de grammaire sur les vues A de `O/internal/grammar/testdata/vue_a_*.bin` (cas de `C/internal/grammar/vue_a_actions_test.go`, `actions_du_film_test.go`) ; décompte des copies de messages 39 et 40 par film sur le corpus (non connu aujourd'hui, § 10) ; documents d'assemblage inchangés.
- **Dépend de** : rien. **Parallèle** avec la file A.

### Item 5 : publier `gestures[]`

- **Crée** : `replay/document_gestes.go` (`Gesture` : `t`, `ms`, `slot`, `kind` `melee` ou `throwStart` ; réutilise `frameDeLAction`, `O/replay/document_score_actions.go:166`, qui sait déjà refuser avant l'origine et après la fin), couverture générique tirée de `ReloadCoverage` (mêmes champs JSON, donc `coverage.reloads` inchangé).
- **Modifie** : `replay/document.go` (champ près de `Reloads`, l.297), `coverage.go` (près de l.182), `layers.go` (une ligne dans la table des calques, près de l.178, et le prédicat de disponibilité près de l.297), `build.go:102-103` (une passe après `poserLesRechargements`).
- **Ce qui change dans le document** : calque `gestures[]` ; `coverage.gestures` {lus, copies, sans joueur, avant l'origine, après la fin, publiés, sans piste} avec l'identité `lus = copies + ... + publiés`. À poser : schéma, chronique, `FORMAT.md`, `GLOSSARY.md`, empreinte de forme et jumeau hôte, goldens d'assemblage.
- **Preuve** : tests sans film (cas de `C/replay/document_gestes_test.go`). Mesures jetables, **aucune vérité n'existe aujourd'hui** : (a) pour `melee`, fraction des coups suivis d'un dégât de mêlée dans `damage[]` à quelques dizaines de millisecondes, avec la fraction des dégâts de mêlée précédés d'un geste ; (b) pour `throwStart`, délai jusqu'à `grenades[].t` et fraction des lancers sans amorce. Les seuils de ces mesures sont à fixer par le mainteneur avant lecture.
- **Dépend de** : 4, et du helper `vieDeLaPlace` de l'item 3 (donc 3 avant 5, ou helper extrait en premier).

### Item 6 : `reloads[]` gagne `ms` et `copies` (additif)

- **Modifie** : `replay/document_reloads.go:24-30` (deux champs de `Reload`), `internal/grammar/rechargements.go` (rendre le nombre de copies par ligne, au lieu de seulement le compter), `types/grammar_rechargements.go`, `replay/filmfacts_rechargements.go`.
- **Ce qui change dans le document** : `reloads[].ms` et `reloads[].copies` (optionnels). Les lignes existantes ne changent pas.
- **Preuve** : tests existants de `O/replay/document_reloads_test.go` étendus ; sur le corpus, `somme(copies)` égale `coverage.reloads.copies` ; reprise de la mesure du contributeur : début contre coupure (894 / 1 289 suivis d'un chargeur, rapport à refaire sur le corpus du mainteneur ; « déduit » tant que non rejoué).
- **Dépend de** : 5 (convention `ms`, couverture générique). Peut se coder avant 5 si `ms` y est défini seul.

### Item 7 (facultatif) : `sticks[]`, le déplacement brut

- **Crée** : `replay/document_sticks.go` (colonnes différentielles par vie : `t` en ms, `x`, `y` ; amorce à la naissance de la vie, `C/replay/stick_de_deplacement.go:104`), `internal/grammar` : changements du couple lus par le même collecteur (valeur inconnue à l'ouverture d'un trou, sans seuil de silence).
- **Ce qui change dans le document** : calque `sticks[]` et `coverage.sticks` {échantillons, sans vie}. Même porte de cadence que l'item 3.
- **Preuve** : test de `C/replay/stick_de_deplacement_test.go` repris ; sonde d'accord du repère contre cap et vitesse (reprise de `C/replay/stick_de_deplacement_research_test.go`, non rejouée) ; **poids du document** : le nombre d'entrées lues va de 131 000 à 376 000 par film (journaux de sonde de `c123`), à mesurer sur le budget de `replay/contract_fixtures_budget_test.go`.
- **Dépend de** : 2 (porte), 3 (helper). **Pas de lecteur connu** chez le contributeur : le mainteneur peut renoncer.

### Dépendances et parallélisme

```
Fichier A : 1 -> 2 -> 3 -> 7
Fichier B : 4 -> 5 -> 6      (5 attend le helper `vieDeLaPlace` de 3)
```

Les deux fichiers touchent aux mêmes fichiers de câblage (`options.go`, `film_inputs.go`, `film_scan_mouvement.go`, `observe.go`, `movement_states.go`, `layers.go`, `coverage.go`, `document.go`, `rev.go`) : fusionner dans l'ordre 1, 4, 2, 3, 5, 6, 7 pour que les conflits restent d'une ligne. Les items 1 et 4 se codent en parallèle sans conflit de logique.

## 9. Ce qu'il faut dire au contributeur

Champs que son application lit et dont le nom ou la forme change. Lecture de `W/` limitée aux fichiers qui touchent aux gestes (vérifié) ; le reste de son client n'a pas été lu.

| Chez lui | Au dépôt d'origine | Où son application le lit |
|---|---|---|
| `actions[]` kind `reload`, `slot`, `ms` | `reloads[]` : `t`, `place`, et avec l'item 6 `ms`, `copies` | `W/features/match-replay/model/reloadStarts.ts:28`, `W/.../sound/reloadSound.ts:152`, `W/.../scene3d/filmActions3d.ts:15` |
| même couple, copies lues par `n` instants égaux | `copies` : début si 0, coupure si au moins 1 | `reloadStarts.ts` |
| `slot` du rechargement | `place` (index du roster), à joindre à la vie par le roster et sa présence | idem |
| `actions[]` kind `melee`, `throwStart` | `gestures[]` : `t`, `ms`, `slot`, `kind` (le tableau change de nom ; `kind` et `ms` gardent leur sens) | `W/.../scene3d/motion3d.ts:102,161`, `reloadSound.ts:39,152` |
| `actions[]` kind `jump` (un instant) | `stances[]` kind `jumpCommand` (un intervalle ; l'instant est `ms0`) | `motion3d.ts:102-110` |
| `stances[]` kind `crouch` tiré du bouton, avec `ms0`, `ms1` | kind `crouchCommand`. `crouch` d'origine existe encore, mais lu dans `i29`, et seulement sur les anciens films | `W/.../scene3d/posturesFilm3d.ts:7-9`, `reloadSound.ts:121` |
| `stances[]` kind `slide` | **absent** : pas de fait solide (§ 4.2). Déduction possible côté client | `posturesFilm3d.ts:7`, `motion3d.ts:58,93` |
| `coverage.playerActions.actions*` | `coverage.gestures` | commentaires de `W/lib/replay/replayNormalize.ts:176` |
| `coverage.stances.control` (ses compteurs de glissade et de saut) | nouveaux compteurs par genre (item 3) | non lu |
| `coverage.controlPlaces` | absent | non lu |
| `tracks[].fullRate.stickT`, `sx`, `sy` | `sticks[]` si l'item 7 est retenu | **non lu** |
| ses sauts dérivés filtrés (`jumpsDerived*`) | `jumpDerived` et `jumpCommand` publiés tous deux ; le client choisit | `motion3d.ts:51,102` |
| schéma 130 | 127 (ou plus, selon la version 1.6.0) | non vérifié |

À dire en une phrase : les trois instants de commande (`jump`, `melee`, `throwStart`) et le rechargement existent au dépôt d'origine, sous deux noms de tableau et avec la vie remplacée par la place pour le rechargement. L'accroupi y est une commande tenue, la glissade n'y est pas, et la pleine cadence est un autre sujet.

## 10. Ce que je n'ai pas pu établir

1. **Justesse de `melee` et de `throwStart`.** Aucune vérité mesurée, ni par lui ni par le mainteneur. Les mesures de l'item 5 sont à faire. Tant qu'elles manquent, ces deux calques ne se publient pas « au seuil de 90 % » : ils n'ont pas de seuil.
2. **Copies des messages 39 et 40.** Je ne sais pas si le jeu les écrit deux fois dans une trame comme pour le rechargement. Le contributeur ne les traite pas (`C/replay/document_gestes.go:39-60`). L'item 4 les compte.
3. **Cadence constante dans un film.** Je n'ai pas vérifié qu'un film ne change pas de cadence en cours de match. `c123.md` la donne par film.
4. **Seuil du nombre minimal d'écarts** pour qu'un pas de cadence soit mesurable : non posé.
5. **Sens du bit d'accroupi et de saut en véhicule.** Rien ne dit qu'il vaut le même geste. La règle « rien en véhicule » est reprise du contributeur sans mesure (déduit).
6. **Équivalence de la place par le roster et de la place lue dans les tirs.** Le contributeur a écrit sa lecture pour les arrivées en cours de partie (exemple de son commentaire, `C/replay/places_de_controle.go`, en-tête). Le dépôt d'origine y répond par la règle des places (`O/replay/tirs_par_place.go`, en-tête). Je n'ai pas comparé les deux sur un film.
7. **Présence du sprint `i57` sur les builds récentes.** Sans effet ici, la glissade étant hors portage.
8. **Repère du stick** : affirmé, non rejoué.
9. **Poids du calque `sticks[]`** sur le budget des fixtures : non mesuré.
10. **Sens des deux copies de rechargement du client** (« copie = coupure ») : chiffres du contributeur, non rejoués. L'item 6 publie le compte, pas l'interprétation.
11. **Aucune compilation** : je n'ai pas vérifié que `ms0`, `ms1` ajoutés à `Stance` ne cassent aucun consommateur de la tête d'origine.

## 11. Décisions attendues

- Noms : `crouchCommand`, `jumpCommand`, `gestures` : à confirmer ou à changer (§ 4.1, 4.3, 4.4).
- Rechargement : garder `reloads[]` d'origine et lui ajouter `ms` et `copies` (item 6), ou publier aussi les copies à sa façon ? Recommandé : le premier.
- Porte de cadence : l'appliquer aussi à `melee`, `throwStart` et au rechargement (non mesuré) ? Appliquer ou non l'accroupi à la porte (coût : 4 films sur 22) ? (§ 7)
- Item 7 (déplacement brut) : retenu ou abandonné ? Son client ne le lit pas.
- `slide` : confirmé hors portage ; un lot de recherche sur un fait du film reste ouvert si le mainteneur le veut.
