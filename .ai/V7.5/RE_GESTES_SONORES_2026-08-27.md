# RE — LE GESTE SONORE : ce qui manquait au format, et ce que ca change

> Ecrit le 2026-08-27, suite du handoff `HANDOFF_SONS_RECONSTITUTION_2026-08-27.md`.
> Ce document porte les MESURES. Les hypotheses du handoff (H1 a H5) et ses sondes (S1 a S6)
> y sont statuees une par une, avec leur denominateur. Ce qui reste ouvert est dit comme tel.

## 0. La question du handoff, et sa reponse

**« Quelle est l'unite de GESTE dans ce format, et comment le moteur l'assemble ? »**

L'unite est l'EVENEMENT, comme on le supposait. Ce qui manquait n'etait pas l'unite : c'etait
**TROIS champs du format que le parseur ne lisait pas**, et chacun explique un symptome
signale a l'oreille par l'utilisateur.

	le DELAI de l'action        une couche ne demarre pas forcement a t = 0
	le NOMBRE DE LECTURES       un conteneur declare combien de fois il se joue, 0 = infini
	le MODE D'ENCHAINEMENT      et A QUEL RYTHME les lectures se suivent

## 1. LE DELAI DE L'ACTION — H2 CONFIRMEE, sonde S2

### 1.1 Le controle d'offset, avec son temoin negatif

`couchesDeEvent` rend UNE COUCHE PAR EVENT ACTION. Le parseur ne lisait de l'action que son
type et sa cible ; tout ce qui suit etait ignore. Layout vise (`CAkAction`, Wwise 2019+) :

	+0  u16 ulActionType | +2 u32 idExt | +6 u8 bIsBus | +7 AkPropBundle | AkPropBundle RANGED
	    puis, pour une action « jouer » : u8 byBitVector | u32 bankID  = EXACTEMENT 5 octets

Le controle est une EGALITE, pas une plausibilite floue : le decodage est juste si et
seulement si le reste vaut 5. Mesure sur `pc/globals/common-rtx-new` (mode `audit-actions`) :

	offset 6  TEMOIN NEGATIF      reste = 5 :     0 / 8 701   (0,00 %)
	offset 7  THESE               reste = 5 : 8 701 / 8 701   (100,00 %)
	offset 8  TEMOIN NEGATIF      reste = 5 :     0 / 8 701   (0,00 %)

### 1.2 Les valeurs ne sont pas des flottants

Deux identifiants de propriete seulement apparaissent sur les actions : **15 et 16**. Lues
comme des `float32` elles donnent des denormaux absurdes (2,8e-44 ; 5,6e-43 ; 1,5134e-41) ;
lues comme des `int32` — ce que l'union `AkPropValue` autorise — elles donnent **20, 400,
10 800**, des millisecondes rondes. Le test est refutable : une mauvaise interpretation ne
produirait pas des multiples de 5 ms.

	idProp 15 = DELAI de l'action   (ms)   235 occurrences non nulles, jusqu'a 10,8 s
	idProp 16 = duree de FONDU      (ms)   151 occurrences non nulles, jusqu'a 5,0 s

### 1.3 Ce que ca corrige, et c'est un symptome date

`71cb04b8` (avant l'apparition d'une nouvelle zone, Roi de la colline) porte deux actions :
la premiere sans delai, la seconde a **400 ms**. L'utilisateur, a l'ecoute du rendu du
2026-08-26, decrivait « un tres court son au debut qui me parait EN TROP ». C'etait la
seconde couche remontee de 400 ms sur la premiere. **L'evenement n'est pas un empilement,
c'est un ENCHAINEMENT.**

Distribution du nombre d'actions « jouer » par evenement (8 687 evenements de `common`) :
1 action 89,69 %, 2 actions 3,23 %, 3 actions 0,58 %, au-dela 0,1 %, et 6,40 % sans aucune
action « jouer ». **340 evenements (3,91 %) etaient sommes a t = 0 a tort.**

## 2. LE NOMBRE DE LECTURES — H1 CONFIRMEE, et mieux que par les `lsnd`

### 2.1 Ou vit la reponse, et pourquoi les `lsnd` ne sont plus necessaires

La sonde S3 proposait de passer par les 20 tags `lsnd` qui referencent la banque des zones.
Inutile : **le nombre de lectures est DANS la banque**, dans `AkRanSeqCntrInitialValues`,
juste avant les champs que `conteneurs_mode.go` localise deja :

	u16 sLoopCount | u16 sLoopModMin | u16 sLoopModMax
	f32 fTransitionTime | f32 ...ModMin | f32 ...ModMax
	u16 wAvoidRepeatCount | u8 eTransitionMode | u8 eRandomMode | u8 eMode | u8 byBitVector
	u32 ulNumChilds   <- ancre deja validee

### 2.2 Les octets, puis le temoin de coherence

Vidage sur la banque des zones (`1c609526`), 24 octets avant la liste d'enfants :

	01000000 00000000 7a440000 00000000 00000100 00000012 01000000   -> sLoopCount = 1
	00000000 00000000 7a440000 00000000 00000100 0000001a 03000000   -> sLoopCount = 0

Les deux layouts candidats sont tous deux « plausibles » sur les bornes (100 % chacun) : ce
qui les separe est le SENS. Le layout B rendrait 0 partout, c'est-a-dire « tout le jeu est
une boucle infinie ». Temoin de coherence interne, ecrit avant la mesure : une boucle infinie
et le drapeau `bIsContinuous` decrivent la meme chose, ils doivent aller ensemble.

	                        continu   pas a pas
	compteur = 0 (infini)       171          39
	compteur >= 1               107        1436

P(continu | infini) = 81,4 % contre 6,9 % pour les autres. **Le layout A est retenu.**

Repartition sur `common` : 210 boucles infinies, 1 541 lectures uniques, 2 doubles
(1 753 conteneurs a lecture plausible sur 1 827).

### 2.3 Ce que ca donne sur les cibles de l'utilisateur

	6b8081a2  base EN COURS DE CAPTURE, alliee   3 couches, dont DEUX EN BOUCLE
	5e48e1d9 / 81f3d1a3 / ca5b3fe1              meme forme — la famille « capture en cours »
	1badec8a / af31554f                          1 couche, 1 parmi 5, EN BOUCLE
	93f632c0 / dcf980a5                          1 couche a +1,50 s, EN BOUCLE

Une « capture en cours » EST une boucle. Le rendu du 2026-08-26 en servait un fragment.

## 3. LE MODE D'ENCHAINEMENT — decouvert en verifiant un « x3 » suspect

Le premier rendu donnait a `fefed142` (drapeau pris, mon equipe) une duree de **7,86 s** :
trois lectures concatenees. C'est faux, et les octets le disent — le meme bloc porte aussi
`eTransitionMode` et `fTransitionTime` :

	61007dcf/2b2a4c34  03000000 00000080 54440000 ... 05000012 01000000
	                   ^ sLoopCount = 3    ^ 850,0 ms        ^ eTransitionMode = 5

`AkTransitionMode` 5 = **cadence de declenchement** : une lecture demarre toutes les 850 ms,
elles se CHEVAUCHENT. Duree reelle du geste : **4,59 s**, pas 7,86.

Calibration de l'offset, sur les 212 conteneurs de `common` qui se repetent — valeurs BRUTES,
non ramenees (un offset faux les disperserait sur 0..255) :

	mode 0  bout a bout                    52
	mode 1  fondu enchaine (amplitude)     21
	mode 3  silence entre deux lectures    19
	mode 4  bout a bout, a l'echantillon    1
	mode 5  cadence de declenchement      119
	                                      ---
	                        hors {0,1,3,4,5} : 0 sur 212

## 4. LES PHASES SOUS CONDITION — H4 CONFIRMEE, sonde S5 RESOLUE

### 4.1 Le translocateur : la « montee en intensite » est un fondu pilote

Les DEUX orphelins de `sb_007_abl_quantum` (`532684898` = 6,22 s et `708804123` = 6,77 s,
les deux plus longs sons de la banque) ne sont pas morts. Le mode `orphelins` remonte la
chaine et nomme la cause :

	Sound 1e4dea87 <- RandomSequence 0c5dad82 <- Blend 33f7ed7c  (lien COUPE)
	   declenche par l'event 388207de

Table du `Blend`, pilotee par le parametre de jeu **3236399890** :

	enfant 265549d3 -> [290526945 627126538]  : (x=0 y=1) (x=0,797 y=1) (x=0,940 y=0)
	enfant 0c5dad82 -> [532684898 708804123]  : (x=0,797 y=0) (x=0,940 y=1) (x=1 y=1)

**C'est un fondu enchaine entre 79,7 % et 94,0 % du parametre.** Le rendu evalue les courbes
« au point de reference » (x minimal, decision assumee du rejeu 2D qui ne pilote aucun
parametre de jeu) : il ne voyait donc que la premiere moitie du geste. La description de
l'utilisateur — « c'est comme si on le chargeait, ca monte en intensite, et ensuite il est
pose » — est exactement cette courbe.

**S5 est close, et son resultat est un negatif ET un positif** : aucun evenement d'AUCUNE
banque de `globals` ni de `common` n'atteint ces deux `.wem` (recherche sur les 5 612
evenements de `arbre_all_globals.json` et les evenements de `arbre_all_common.json` :
0 occurrence hors la liste d'orphelins de leur propre banque). Ils ne sont pas joues depuis
ailleurs : ils sont joues d'ICI, sous condition.

### 4.2 Les bobines : un etat de commutation, et il est le meme quatre fois

Les QUATRE banques de bobine laissent **16 orphelins CHACUNE**. Meme cause, un `Switch` :

	Switch 3685f605, groupe 2275666646, etat 163696720, defaut 1093928064

Un compte identique quatre fois de suite n'est pas un accident : l'explosion complete de
bobine a un SECOND jeu de 16 sons que le jeu joue sous un autre etat, et que nous ne rendions
pas. L'etat n'est pas nomme (le hachage n'a pas ete tente dessus).

### 4.3 Le drapeau : deux orphelins, meme mecanique

`61007dcf` porte 2 orphelins sous les `Blend` des evenements `297b9c7b` et `6766338f`,
pilotes par le parametre **1299539600**, entree a x = 25,27 et plein a x = 74,40.

### 4.4 Recensement des parametres de fondu — ils sont SEPT dans tout `globals`

	1096541797 : 60 couches sur 30 banques      landing_magnitude   (NOM CASSE)
	 260706399 : 20 couches sur  3 banques
	2365147768 : 16 couches sur  2 banques
	1871251695 :  3 couches sur  3 banques      cluster_num_sounds  (NOM CASSE)
	1299539600 :  2 couches sur  1 banque       (propre au drapeau)
	 860368976 :  1 couche  sur  1 banque
	3236399890 :  1 couche  sur  1 banque       (propre au translocateur)

**Deux noms sont casses** par FNV-1 32 bits contre les 187 496 jetons du binaire du jeu, a
esperance de collision `187 496 x 7 / 2^32 = 0,0003` — c'est-a-dire certains. Le fait que
`1299539600` et `3236399890` ne vivent chacun que dans UNE banque interdit de les lire comme
un parametre global (la distance en serait le candidat evident) : ils sont propres a l'objet.

**NEGATIF, avec son denominateur** : les cinq autres resistent a la composition de deux
jetons (187 496 jetons x 140 affixes x 2 ordres = 52 498 880 candidats, esperance 0,0611,
sous le seuil de publication 0,10). **0 resultat.** La voie du hachage est epuisee a
discipline constante sur ces cinq.

## 5. CE QUI EST PRODUIT

	413 gestes rendus, un fichier par GESTE (et non plus par couple evenement x variante)
	 43 en boucle | 31 enchaines | 7 phases sous condition | 0 silencieux | 0 ecrete
	 24 banques   | 56 cartes portent un libelle francais, dont 45 un nom Wwise casse

Planche d'ecoute : artefact `6aadf3d5-7acf-4b98-a050-6bc88db55b7d`, republiee en place
(une seule adresse, regle utilisateur). Outillage de rendu et de planche :
`Halo Infinite - Sons v75/_outils/{rendu_geste,planche_gestes}/`.

### 5.1 Un defaut de rendu trouve et corrige DANS CETTE PASSE

`volumedetect` ne sait mesurer que des echantillons ENTIERS : sur un intermediaire en
flottant, ffmpeg convertit d'abord en 16 bits, ce qui **ECRETE** tout ce qui depasse 0 dBFS
et rapporte 0,0 dB. La normalisation qui suit n'enlevait alors qu'un decibel et le fichier
final sortait ecrete. Temoin : la capture de drapeau alliee (une couche a +15 dB) sortait a
0,0 dBFS au lieu de -1,0. Corrige par une mesure en DEUX PASSES — la seconde n'est pas
gratuite : attenuer systematiquement casse les melanges FAIBLES, qui passent sous le plancher
du 16 bits. On ne rejoue avec reserve que si la premiere mesure bute sur 0 dB.

### 5.2 Les sons deja livres dans le rejeu etaient rendus a plat

Les huit `.wav` de `static/sounds/halo_infinite/objective_*` dataient du rendu du 2026-08-26.
Quatre d'entre eux etaient FAUX au sens du format :

	objective_flag_scored_team    couche a +15 dB remontee de 100 ms sur l'autre
	objective_flag_scored_enemy   idem, 100 ms
	objective_flag_stolen_team    couche a +400 ms, jouee 3 fois toutes les 850 ms
	                              (2,49 s -> 4,59 s)
	objective_flag_stolen_enemy   couche a +400 ms (3,41 s -> 3,81 s)

Les huit sont remplaces par les rendus neufs. `objective_zone_captured_team` etait deja
identique au bit pres — controle de reproductibilite de la chaine, pas une coincidence.

## 6. CE QUI RESTE OUVERT

1. **S1 — le `hsc*` du chemin.** Non lancee. C'est la seule piste de NOMMAGE encore ouverte
   pour les reperes de zone (`d8a2fcb8`, `6b8081a2`, `71cb04b8`), qui restent designes a
   l'oreille. Le hachage y est epuise (handoff §3.3).
2. **S4 — le tag de MODE.** Non lancee.
3. **Le rendu des phases sous condition est un PREMIER SON, pas la phase entiere.** Une phase
   est un point de choix comme un autre ; on en sert le premier media et la fiche dit combien
   il y en a. Rendre le fondu lui-meme (balayer le parametre de x = 0 a x = 1) demanderait de
   connaitre la vitesse du balayage, que le format ne porte pas.
4. **L'etat de commutation des bobines n'est pas nomme** (`163696720`, groupe `2275666646`).
5. **`assault_bomb_planted_loop` ne declare aucune boucle dans son conteneur** : la repetition
   est imposee par le jeu (evenement reposte, ou paire Play/Stop), pas par la banque. Ce n'est
   pas une contradiction, c'est une facon differente de tenir une boucle — mais elle n'est pas
   reconstituable depuis la banque seule.
6. **Le RAMASSAGE SUR SOCLE reste non date** : `padPickups` publie un INTERVALLE
   `[tLow, tHigh]`, pas un instant. Le son est identifie et rendu
   (`play_007_abl_shared_pickup`), il n'a pas d'instant ou se poser. Condition de reprise deja
   au registre des reports.

## 7. OUTILLAGE NEUF — `cmd/weapon-sounds`

	audit-actions   le contenu des Event Actions : controle d'offset avec temoin negatif,
	                distribution du nombre d'actions par evenement, proprietes portees
	audit-boucles   le nombre de lectures d'un conteneur et le mode d'enchainement, avec le
	                vidage des octets, la plausibilite des deux layouts et le temoin de
	                coherence compteur/bIsContinuous
	orphelins       pourquoi un `.wem` embarque n'est atteint par aucun evenement : la remontee
	                nomme le filtre qui coupe et la condition sous laquelle le jeu le joue
	blend           la table de fondu d'un conteneur, courbe par courbe ; sans `-sbnk`,
	                RECENSEMENT des parametres de fondu de tout un module

Le mode `eqip-arbre` porte desormais, par couche : `delai_s`, `repetitions`,
`mode_enchainement`, `transition_s`, et par evenement `variantes_conditionnelles`.

---

## 8. SECONDE PASSE DU 2026-08-27 — la planche etait inutilisable, et sept noms de plus

**LE RETOUR UTILISATEUR, mot pour mot** : « la quasi totalite des sons ont des noms
inintelligibles comme 0b2a938e, comment veux-tu que je travaille avec ca ? Y en a plus de 400. »
Il a raison et c'est un defaut de LIVRAISON, pas de mesure : 413 cartes portant des
identifiants ne sont pas un inventaire, ce sont des donnees brutes servies telles quelles.

### 8.1 Trois corrections, et chacune RETIRE du travail

1. **DEDOUBLONNAGE.** 413 rendus ne font pas 413 sons. Mesure sur la banque des zones :
   **88 evenements, 49 jeux de medias distincts** — le meme geste y est declare une fois PAR
   MODE DE JEU. Une carte par SON, et la carte dit combien d'evenements le jouent.
   Total : **413 rendus -> 328 cartes**, dont 140 ouvertes (a identifier) et 188 repliees.
2. **PROPAGATION DU NOM DANS LE GROUPE.** Si un seul evenement du groupe porte un nom casse,
   tout le groupe le porte — deux evenements qui jouent le meme materiau jouent le meme son.
   `c3327c0b` = `..._strongholds_contested` nomme du meme coup ses trois jumeaux.
3. **UN NUMERO, PAS UN HACHAGE.** « Zone 07 », pas « 0b2a938e ». Et quand le son n'a pas de
   nom, le titre dit sa FORME (« Boucle, 3 couches », « Enchainement, 2 couches ») — on ne
   cherche pas de la meme facon un stinger d'une seconde et une boucle a trois couches.

Ce qui est ACQUIS (equipements, drapeau, bombe, extraction, bobines) est REPLIE derriere un
depliant, a la demande de l'utilisateur.

### 8.2 SEPT NOMS DE PLUS, et la methode qui les a rendus

La voie du hachage etait declaree epuisee (§4.4 et handoff §3.3). Elle ne l'etait pas : ce qui
etait epuise, c'etait **la composition a DEUX jetons pris dans le dictionnaire du binaire**.
La forme reelle en prend TROIS.

	play_007_abl_quantum_teleport_player_start     <- 3 jetons

Casser trois jetons parmi 187 496 mots demande 6,6e15 candidats : hors d'atteinte. Mais un
**vocabulaire CURIE de 120 a 180 mots** en demande 2 a 5 millions, soit une esperance de
quelques centiemes. **Le pari n'est pas sur la taille, il est sur le CHOIX des mots** : ceux
de la grammaire deja mesuree, plus ceux du geste que l'utilisateur decrit.

	TRANSLOCATEUR (banque dcfaa487)
	10f62776  play_007_abl_quantum_teleport_player_start   DEBUT de la teleportation
	61adc7b4  play_007_abl_quantum_ready_player_loop       BOUCLE de « pret »
	e3dc967f  play_007_abl_quantum_portal_warning          avertissement du portail (confirme)

	ZONES (banque 1c609526, espace de nommage `strongholds`)
	a880da84  play_004_mod_mp_strongholds_contested_win       base contestee, on la GAGNE
	0c6582b9  play_004_mod_mp_strongholds_contested_lose      base contestee, on la PERD
	fddf794f  play_004_mod_mp_strongholds_scoring_tick_team   tic de score, mon equipe
	9a2a8880  play_004_mod_mp_strongholds_scoring_tick_enemy  tic de score, adversaire

**ESPERANCE CUMULEE DE LA JOURNEE : 0,149** sur 7 noms publies — soit environ 2 % de chance
qu'un nom donne soit fortuit. Le dire est la moitie du travail.

**ET LE CONTROLE QUI VAUT MIEUX QUE L'ESPERANCE** : les noms sortent en PAIRES COHERENTES.
`contested_win` et `contested_lose` d'un cote, `scoring_tick_team` et `scoring_tick_enemy` de
l'autre, `teleport_player_start` et `ready_player_loop` partageant la modulation `_player`.
Une collision fortuite ne produit pas une paire `_team`/`_enemy` sur le meme radical.

**NEGATIFS, avec leur denominateur** : 19 des 23 evenements du translocateur resistent a un
vocabulaire de 183 mots (6 831 939 candidats, esperance 0,0302), et 78 des 85 evenements de
zone resistent a 141 mots (3 220 863 candidats, esperance 0,0615).

### 8.3 Ce que l'utilisateur a corrige sur le translocateur, et ce que ca elimine

Trois precisions donnees le 2026-08-27, et elles valent des mesures :

	« Y a pas vraiment de balise dans le jeu, c'est un equipement que le joueur porte au
	  poignet. La ca cree une genre de faille spatiotemporelle sur sa position exacte. »
	duree du geste : 2 a 4 secondes
	« Seulement moi, un autre joueur entendra un autre son. »

Consequences immediates, chacune ecrite comme une elimination :

1. **LA PHASE SOUS CONDITION EST ECARTEE.** Les deux plus longs sons de la banque (6,22 s et
   6,77 s), que §4.1 designait comme la « montee en intensite », durent trop. Ils restent une
   mesure exacte du format ; ils ne sont pas le geste cherche.
2. **LE GESTE PORTE LA MODULATION `_player`.** Deux des trois noms casses aujourd'hui la
   portent — la piste est la bonne famille.
3. **QUATORZE CANDIDATS**, et pas un de plus : les gestes du translocateur dont la duree tombe
   dans [1,8 ; 4,5] s. Ils sont marques CANDIDAT et places en tete de la section sur la
   planche. C'est deux minutes d'ecoute, pas quatre cents cartes.

### 8.4 Le champ de reparation : le fichier livre etait le mauvais evenement

L'utilisateur : « pour le champ de reparation je veux que le replay joue le son *Champ de
reparation — activation* quand un joueur le pose ». Mesure : le fichier livre depuis le
2026-08-18 sous le nom `repair_field_activate.wav` faisait **0,38 s** — c'est
`play_007_abl_repairfield_deploy_player` (`8ed46d21`), le « pop » de l'objet lache.
L'ACTIVATION est `play_007_abl_repairfield_activate` (`c48cf171`), **3,26 s**. Les deux
evenements ne se distinguaient pas avant le nommage des banques par hachage.

Les trois variantes livrees sont desormais celles de `c48cf171` (143632032 / 222530989 /
640887009), gain de chemin +1 dB, crete plafonnee a -1,0 dBTP.

**DEUX GARDE-RAILS ONT MORDU, ET C'EST LEUR ROLE** :

1. `SOURCES_COURTES` declarait `repair_field_activate` a 0,38 s. Entree supprimee, avec la
   raison ecrite : le fichier n'est plus le meme son.
2. `SOUND_CUT_MAX_S` valait 4,0 s et `objective_flag_stolen_team` en fait desormais 4,588
   (trois declenchements a 850 ms d'intervalle). **Releve a 5,0 s**, avec la mesure qui
   l'impose en commentaire — garder 4 s aurait tronque la troisieme alerte EN SILENCE, ce que
   le commentaire de la constante interdit nommement.

### 8.5 Le retour du drapeau etait deja identifie

Demande : « faut identifier le son du retour du drapeau aussi ». Il l'est depuis le 2026-08-26
et il est BRANCHE : `b2a0d0f0` = `play_004_mod_mp_ctf_flag_returned`, verifie par recalcul du
hachage (FNV-1 exact), rendu, livre en `objective_flag_returned.wav` (1,31 s) et cable sur
`flag_returns` sans variante d'equipe — le jeu n'en a pas. Si ce qui s'entend ne colle pas,
c'est l'ATTRIBUTION qu'il faut revoir, pas le nommage.

### 8.6 Outillage neuf

	_outils/composer            casse des noms par COMPOSITION d'un vocabulaire curie (1 a 3
	                            jetons, plus la forme <a>_<b>_<modulation>_<phase>), esperance
	                            imprimee AVANT tout resultat
	gestes/noms_evenements.json TABLE DES NOMS, source unique lue par la planche. Un nom casse
	                            apres coup entre sans re-rendre un seul fichier audio.

---

## 9. TROISIEME PASSE — l'inventaire par MODE, et trois banques qui manquaient

**Question utilisateur** : « pour les modes de jeu il nous manque quoi ? A identifier ou
cabler ? » Elle demande un inventaire par mode, pas une planche. Le voici, sur pieces.

**Et il fallait d'abord verifier que l'inventaire des BANQUES etait complet.** Il ne l'etait
pas. Un balayage de 79 noms de banque candidats (`sb_004_mod_mp_*`, `sb_004_mod_cv_*`,
`sb_007_abl_*`) contre les identifiants Wwise des 1 697 banques, esperance
`79 x 1697 / 2^32 = 3,1e-5`, rend **trois banques jamais inventoriees** :

	b0c651ea  sb_004_mod_mp_oddball       28 evenements, 53 sons   <- LE CRANE
	6a9ba454  sb_004_mod_mp_elimination   27 evenements, 40 sons
	e3ba2522  sb_004_mod_mp_vip           40 evenements, 62 sons

Le meme balayage retrouve `sb_007_abl_quantum` et `sb_007_abl_grapplinghook` a leurs banques
connues : c'est la calibration de la passe, et elle est gratuite.

### 9.1 VINGT-DEUX NOMS DE PLUS, par la meme composition curie

	CTF          flag_return_contested                                          (+1)
	ASSAUT       bomb_carrier_killed                                            (+1)
	EXTRACTION   zone_spawn_alert, arming_{start,loop,stop,complete}_{team,enemy} (+9)
	LANDGRAB     zone_spawn, contested, contested_win, contested_lose            (+4)
	ODDBALL      skull_{spawn,despawn,taken,pickup,dropped}, scoring_{team,enemy} (+7)

**Le controle est dans la REGULARITE des familles.** `arming_start` / `arming_loop` /
`arming_stop` / `arming_complete`, chacun en `_team` et `_enemy` : huit noms qui forment une
grille complete. Une collision fortuite ne remplit pas une grille.

**ESPERANCE CUMULEE DE LA JOURNEE : environ 0,34** pour 29 noms publies, soit ~1,2 % par nom.

### 9.2 L'INVENTAIRE PAR MODE — identifie, declenchable, cable

Trois colonnes, et ce sont trois problemes differents.

	MODE          SONS NOMMES   DECLENCHEUR DANS LE FILM        CABLE DANS LE REJEU
	CTF               10        flag_captures/steals/grabs/     7 sons sur 10
	                            returns (decodeur OK)           manquent : flag_spawn,
	                                                            flag_return_contested
	                                                            (aucun declencheur)
	ZONES             10        zone_captures, zone_secures     1 son sur 10
	(Bastion, CT,               + jauge + span actif + score    (capture alliee seulement)
	 KOTH, Landgrab)  +4        (tous publies, non lus)
	ASSAUT            10        AUCUN                           0
	EXTRACTION        13        AUCUN                           0
	ODDBALL            7        AUCUN                           0
	ELIMINATION / VIP  0        AUCUN                           0

### 9.3 CE QUI BLOQUE, en trois categories qui ne se traitent pas pareil

**A. Un SON manque — c'est une ecoute, pas du code.**

	capture de zone ADVERSE   la paire `zone_captures` est a moitie vide, le cote adverse est
	                          MUET par decision (jamais le son allie « faute de mieux »)
	zone_secures              la statistique est publiee par le decodeur, aucun son designe

**B. Le CABLAGE est possible tout de suite** — le son ET le declencheur existent, personne ne
les a joints. Aucun travail cote Go.

	capture en cours       `zoneStates[].gauge` : les RAMPES sont deja publiees (schema 18)
	                       son : `6b8081a2`, 3 couches dont deux EN BOUCLE
	nouvelle colline       `zoneStates[].spans[].active` bascule quand la colline change
	                       son : `71cb04b8`, 2 couches enchainees a +0,40 s
	tic de score           `scoreTimeline.teams[].total` : chaque increment est un tic
	                       sons : `..._scoring_tick_team` et `..._tick_enemy`

Le moteur sonore ne lit AUJOURD'HUI que `doc.objectives` (les evenements de statistique
nommes) : ni `zoneStates`, ni `scoreTimeline`. C'est la seule raison pour laquelle ces trois
sons ne sonnent pas.

**C. Le DECODEUR manque** — le son existe, le film ne publie rien pour l'accrocher.

	ASSAUT, EXTRACTION   `namedStatSlots` (analysis/objectiveevents/named.go) ne porte QUE
	                     `ObjectiveTypeFlag` et `ObjectiveTypeZone`. Il faut la table de slots
	                     statborg de ces deux modes — le meme travail que celui qui a produit
	                     les tables CTF et zones.
	ODDBALL              `ObjectiveTypeSkull` existe comme constante, sans table de slots. Le
	                     lot D4 a etabli l'identite du crane dans le film (mot 0x0017592C) :
	                     la detection existe, elle ne remonte pas en evenement de statistique.
	zone CONTESTEE       `ZoneSpan` ne publie pas d'etat « contestee ». A DERIVER de la jauge :
	                     une rampe qui monte puis retombe sans changement de proprietaire.
	sortie de zone       aucun canal. Il faudrait croiser les positions des joueurs avec la
	                     geometrie de `mapObjectives.zones` — un vrai lot d'analyse.

### 9.4 Le translocateur est mis en veille

L'utilisateur, apres ecoute : « dans translo y a beaucoup de sons qui n'ont rien a voir, a mon
avis c'est pas la bonne piste sur laquelle tu t'acharnes. » La section passe en REPLIEE sur la
planche. Ce qui reste acquis et ne sera pas a refaire : la banque est bien celle du
translocateur (nom casse ET chaine de tags, deux voies independantes), quatre de ses
evenements sont nommes, et les 14 candidats a la fourchette de duree sont marques. Ce qui est
REFUTE : que le geste cherche soit l'un des 32 sons de ces trois banques.

---

## 10. QUATRIEME PASSE — les zones SONNENT, et le translocateur a enfin une reference

### 10.1 Deux designations de plus, et les paires se ferment

L'utilisateur, a l'ecoute de la planche : « Zone 15 a l'air de ressembler a la capture adverse,
Zone 17 a la capture en cours adverse. »

	4ebe99d6 / 8594aef7 / 9fad450d   base capturee — ADVERSE      (2,30 s, 1 couche)
	b2af5c02 / bd7462ce              capture en cours — ADVERSE   (boucle, 2 couches)

Les deux paires du mode zones sont donc COMPLETES pour la premiere fois. La capture adverse
etait muette depuis le 2026-08-26 par decision assumee — jouer le son allie sur une capture
adverse annoncerait un gain quand on perd une base.

### 10.2 Le cablage : `zoneSound.ts`, trois regles

Le moteur ne lisait que `doc.objectives`. Il lit desormais AUSSI `doc.zoneStates`, dans un
fichier a part parce que la source et la cle de jointure different (un etat de zone n'a pas
d'auteur ; il a un proprietaire).

	CAPTURE EN COURS   une RAMPE de jauge dont la fin coincide avec un changement de
	                   proprietaire -> son du camp du NOUVEAU proprietaire, a la frame de
	                   DEBUT de la rampe. Une rampe qui retombe sans changer le proprietaire
	                   reste MUETTE : c'est le cas « contestee », et nous ne savons pas nommer
	                   celui qui a echoue.
	TIC DE SCORE       un tic PAR SECONDE tant qu'un camp tient TOUTES les zones. REGLE PRODUIT
	                   de l'utilisateur, pas une mesure. Garde double : au moins deux zones, et
	                   aucun intervalle `active` (le marqueur de colline) — sans quoi une
	                   colline tenue ferait sonner un tic par seconde tout le match.
	NOUVELLE COLLINE   chaque debut d'intervalle `active` SAUF LE PREMIER. Aucun camp.

15 tests neufs, dont quatre qui epinglent des SILENCES : camp allie non resolu, rampe sans
changement de proprietaire, tics en Roi de la colline, zone unique.

### 10.3 Le tic de score est le seul son livre TRONQUE, et il faut le dire

« Je les trouve un peu fort. » Le geste du jeu dure 3,62 s (allie) et 4,36 s (adverse) — servi
entier a raison d'un par seconde, il s'empilerait quatre fois sur lui-meme. Il est donc coupe a
1,2 s avec un fondu de 0,25 s, et attenue a **-12 dBTP** au lieu du -1 dBTP de tous les autres.
Les deux ecarts sont declares nominativement dans `SOURCES_COURTES`, avec leur raison.

### 10.4 Le plafond de duree monte une seconde fois, et la regle est ecrite

`SOUND_CUT_MAX_S` passe de 5,0 a 6,0 : `objective_zone_new` (le deplacement de la colline)
enchaine deux couches a +0,40 s et dure 5,15 s. La regle est desormais explicite dans le
commentaire de la constante : **le plafond suit la plus longue SOURCE livree, arrondie a la
seconde superieure**. Il n'est pas la pour raccourcir un geste.

### 10.5 Le translocateur : une video, et une mesure qui ECHOUE

L'utilisateur a fourni une capture video du jeu (76 s, le geste vers 3 s). Une comparaison
spectrale automatique a ete ecrite et lancee contre les rendus : mono 16 kHz, trames de 512
avec saut de 256, 20 bandes logarithmiques de 50 Hz a 7 kHz, correlation croisee normalisee
maximisee sur le decalage.

**ELLE NE DISCRIMINE PAS, et le temoin l'a dit a chaque tour** :

	1re version    normalisation par bande, recouvrement partiel autorise
	               1er 0,964, 10e 0,954 sur des banques sans rapport -> aucune separation
	2e version     normalisation par trame, recouvrement COMPLET exige
	               le classement se remplit de sons de 0,16 a 1,04 s -> biais de LONGUEUR
	3e version     plancher de duree a 1,8 s (567 candidats compares)
	               1er 0,639, mais le classement suit encore la duree des candidats

**Cause, et elle est structurelle** : la bande son de la video porte la musique et l'ambiance
PAR-DESSUS le geste. Une correlation de forme spectrale ne separe pas une couche d'un melange.
La mesure est donc REFUTEE pour cet usage, et le classement n'est pas publie comme un resultat.

**Ce qui est fait a la place** : l'extrait video (2,2 s -> 6,6 s) et les 8 premieres secondes
sont servis EN TETE DE PLANCHE, comme deux cartes de reference. La comparaison redevient ce
qu'elle aurait du rester — une ecoute, mais dans la meme page que les 14 candidats.

### 10.6 Oddball : les sept sons sont valides, le cablage ne l'est pas encore

L'utilisateur valide Oddball 01 a 07 (« tu peux les plug »). Ils ne sont PAS cablables
aujourd'hui, et la raison est exacte : `doc.objectives` vient de `IdentifyNamedEvents`, qui lit
`namedStatSlots` — une table qui ne porte que `ObjectiveTypeFlag` et `ObjectiveTypeZone`.
`ObjectiveTypeSkull` existe comme constante et `extractFromTh10` produit bien des
`ObjectiveEvent` de type `skull`, mais ce canal-la n'alimente pas le document de rejeu.

Le lot est donc le meme que celui qui a produit les tables CTF et zones : balayage
`cmd/tmp_statnames` sur un corpus de films Oddball, confrontation des valeurs finales de chaque
emplacement aux comptes de `personal_score_awards`, controle sur moities disjointes. Ce n'est
pas un cablage, c'est une MESURE.
