# RE — NOMMER LES BANQUES SONORES DU JEU, ET LES EVENEMENTS QU'ELLES PORTENT

> Ecrit le 2026-08-26. Demande utilisateur : les sons qui manquent — objets d'objectif
> (dispositif d'extraction, prise de bastion, activation de bombe, capture de drapeau, avec
> la question « y a-t-il une distinction de son par equipe ? »), bobines lancees (le son du
> KILL, precision donnee en cours de lot), equipements (translocateur quantique, traqueur de
> menaces, ecran occultant, champ de reparation).
>
> Ce document ne remplace pas `RECETTE_SONS_ARMES.md` (le COMMENT refaire une livraison) ni
> `replay2d/PLAN_EQUIPEMENTS_MANQUANTS_SONS.md` (le lot des equipements). Il porte UNE
> mesure neuve, qui leve le plafond des deux : **le nom d'une banque sonore se retrouve par
> hachage**, et avec lui le nom de ses evenements.

## 0. Le verrou qui saute

Jusqu'ici une banque ne se nommait que par RICOCHET : si un de ses `.wem` vivait dans un
`.pck` au nom explicite, elle heritait de ce nom. Le lot des equipements avait chiffre la
portee du pont : **3 banques d'equipement sur 17** touchent un pack nomme. Les banques de
MODE (drapeau, bombe, extraction) n'ont AUCUN pack : elles etaient hors d'atteinte.

La voie neuve : le chunk `BKHD` d'une banque porte son identifiant Wwise, et cet identifiant
est le **FNV-1 32 bits du nom de fichier en minuscules**. Ce n'est pas une hypothese —
`calibrerNommage` (`cmd/weapon-sounds/mapping.go`) confrontait deja les deux sur les banques
a pack. Le mode neuf `banks-noms` generalise ce controle a TOUT le module, puis attaque au
dictionnaire les banques restantes.

**CALIBRATION, ET ELLE EST GRATUITE.** Les noms de packs sont connus : 841 sur le disque.
Sur les modules balayes, **647 banques portent l'identifiant FNV-1 d'un nom de pack**
(580 dans `pc/globals`, 57 dans `common`, 10 ailleurs). La convention n'est donc pas
supposee, elle est verifiee sur 647 temoins avant qu'un seul nom ne soit casse.

**ESPERANCE DE COLLISION, imprimee AVANT les resultats.** `candidats x cibles / 2^32` doit
rester sous 0,10 (le seuil que le lot des equipements s'etait donne pour le murmur3). Les
resultats publies ici sortent de passes a **0,0708** (banques) et **0,0408** (evenements).
Ce qui a ete obtenu au-dessus du seuil est signale comme tel et n'est pas publie comme acquis.

## 1. Perimetre balaye

	module                                     banques   nommees par pck
	pc/globals/globals-rtx-new                   1305           580
	pc/globals/common-rtx-new                     340            57
	pc/globals/multiplayer-rtx-new                 17             0
	pc/globals/multiplayer_r1-rtx-new              17             0
	pc/globals/multiplayer_r2-rtx-new               1             1
	pc/globals/multiplayer_r3-rtx-new               2             2
	pc/globals/levels-rtx-new                       1             1
	pc/globals/forge-rtx-new                        0             -
	pc/globals/mainmenu-rtx-new                     0             -
	pc/compositions/.../mp_02-rtx-new              14             6
	                                     TOTAL   1697           647

Les modules par NIVEAU (`pc/levels/multi/*`) ne sont pas balayes : aucune banque de MODE n'y
vit (les quatre banques de mode trouvees sont dans `globals` et `common`).

## 2. Les banques nommees qui repondent a la demande

	banque                             sbnk        module    wem   evts   cible utilisateur
	sb_004_mod_mp_ctf                  61007dcf    globals    52     46   DRAPEAU
	sb_004_mod_mp_assault              2b01f208    globals    38     42   BOMBE
	sb_004_mod_mp_extraction           156c35d5    common     33     23   DISPOSITIF D EXTRACTION
	sb_004_mod_mp_landgrab             8369d532    common     39     23   zones (piste bastion)
	sb_004_mod_mp_shared_ui            ee694c9e    globals    33     20   sons d objectif partages
	sb_004_mod_mp_shared_global        7c954e9f    common     14      9   idem
	sb_004_mod_mp_shared_ai            586f11c8    globals    86     11   -
	sb_004_mod_mp_attrition            522404fa    common     10      6   -
	sb_004_mod_mp_escalation           d39ed58a    common      8      8   -
	sb_004_mod_mp_infection            4900a71e    common      9      7   -
	sb_004_mod_mp_academy              ea290593    common     16     14   -
	sb_007_abl_shroud                  92c830f5    globals    38     11   ECRAN OCCULTANT
	sb_007_abl_quantum                 dcfaa487    globals    70     23   TRANSLOCATEUR QUANTIQUE
	sb_007_abl_repairfield             5724312f    globals    31     12   CHAMP DE REPARATION
	sb_007_abl_grapplinghook           385461e8    globals    69     29   grappin
	sb_007_abl_knockback               7bd0883c    globals    33     11   repulseur
	sb_007_abl_activecamo              923ff9f4    globals      5      5   camouflage
	sb_007_abl_overshield              5dbd34db    globals      4      4   surbouclier
	sb_007_abl_evade                   916d040a    globals      6      2   propulseur
	sb_007_abl_shared                  15c5b355    globals     20     15   equipement, sons communs
	(non nommee)                       7acb11cc    globals     32     16   capteur + traqueur

**CE QUE CELA CLOT.** `sb_007_abl_shroud` est la banque `92c830f5` — exactement celle que le
lot du 2026-08-18 avait attribuee au rang 10 anonyme (`eqip 0x4396db42` / `0x4eebcb18`), et
dont il ecrivait : « aucun de ces 38 `.wem` ne vit dans un `.pck` nomme, elle ne nomme donc
rien ». **Elle nomme, par une AUTRE chaine** : son identifiant Wwise est
`fnv1("sb_007_abl_shroud")`. L'intuition de l'utilisateur (« ecran de dissimulation ») etait
juste ; ce qui manquait n'etait pas un argument mais un espace de hachage. Le negatif du
dictionnaire murmur3 reste valide et n'est pas contredit : il portait sur le `string_id` du
TAG, pas sur l'identifiant de la BANQUE — deux fonctions de hachage differentes, deux
espaces differents.

De meme `sb_007_abl_quantum` = `dcfaa487`, la banque que le meme lot appelait
`translocator_beacon` par la chaine de tags : le nom la confirme independamment.

## 3. Les evenements — la grammaire, puis les noms

**LA GRAMMAIRE EST MESUREE, PAS DEVINEE.** Sur les 17 premiers evenements casses, 17 sur 17
prennent la forme :

	play_<nom de banque prive de « sb_ »>_<nom>_<verbe>[_<modulation>]
	play_        007_abl_repairfield        _deploy _player
	play_        004_mod_mp_ctf      _flag  _scored _team

Cette contrainte, une fois mesuree, autorise un vocabulaire beaucoup plus large a esperance
egale. Resultat : **46 evenements nommes a esperance cumulee 0,0408**.

### 3.1 DRAPEAU — `sb_004_mod_mp_ctf`

	play_004_mod_mp_ctf_flag_scored_team     59223651   2 wem   2 couches SIMULTANEES
	play_004_mod_mp_ctf_flag_scored_enemy    be7759aa   2 wem   2 couches SIMULTANEES
	play_004_mod_mp_ctf_flag_taken_team      fefed142   2 wem   2 couches SIMULTANEES
	play_004_mod_mp_ctf_flag_taken_enemy     f4084c17   2 wem   2 couches SIMULTANEES
	play_004_mod_mp_ctf_flag_pickup_team     ff3e5807   1 wem
	play_004_mod_mp_ctf_flag_pickup_enemy    47b28508   1 wem
	play_004_mod_mp_ctf_flag_returned        b2a0d0f0   1 wem
	play_004_mod_mp_ctf_flag_spawn           58a5e7ba   1 wem

**LA CAPTURE DE DRAPEAU EST `flag_scored`**, et elle existe en DEUX exemplaires distincts.

### 3.2 BOMBE — `sb_004_mod_mp_assault`

	play_004_mod_mp_assault_bomb_planted_loop  a38d8b3e  4 wem  UN son parmi 4
	play_004_mod_mp_assault_bomb_disarm_loop   b57933f2  1 wem
	play_004_mod_mp_assault_bomb_detonated     984f65e5  1 wem
	play_004_mod_mp_assault_bomb_taken_team    458c0a64  1 wem
	play_004_mod_mp_assault_bomb_taken_enemy   db424829  1 wem
	play_004_mod_mp_assault_bomb_pickup_team   d636b2ad  2 wem
	play_004_mod_mp_assault_bomb_pickup_enemy  d0eb18a6  2 wem
	play_004_mod_mp_assault_bomb_spawn         e8ca00b8  1 wem
	play_004_mod_mp_assault_bomb_despawn       4cf90163  1 wem

**L'ACTIVATION DE LA BOMBE EST `bomb_planted_loop`** (une BOUCLE, 4 variantes) ; le desamorcage
a la sienne, la detonation est un evenement a part.

### 3.3 DISPOSITIF D'EXTRACTION — `sb_004_mod_mp_extraction`

	play_004_mod_mp_extraction_zone_spawn          2aca2a6c   1 wem
	play_004_mod_mp_extraction_zone_despawn        85b3f057   1 wem
	play_004_mod_mp_extraction_point_scored_team   cc22923b   1 wem
	play_004_mod_mp_extraction_point_scored_enemy  66df9bfc   1 wem

L'APPARITION du dispositif est `zone_spawn`. Les 19 autres evenements de la banque — dont,
tres probablement, l'initiation et la conversion (`ExtractionInitiationsCompleted` /
`ExtractionConversionsCompleted` cote statistiques) — ne sont PAS nommes a ce jour.

### 3.4 EQUIPEMENTS

	play_007_abl_shroud_deploy_player         ecd34df7   3 wem   ECRAN OCCULTANT : la pose
	play_007_abl_shroud_attach                435d741b   3 wem
	play_007_abl_shroud_enter                 d6ee1fae   3 wem
	play_007_abl_shroud_exit                  37d7abd4   3 wem
	play_007_abl_repairfield_deploy_player    8ed46d21   3 wem   champ de reparation
	play_007_abl_repairfield_activate         c48cf171   3 wem   (LIVRE : c'est celui-la)
	play_007_abl_repairfield_deactivate       6aa63330   3 wem
	play_007_abl_repairfield_attach           a80859e9   5 wem
	play_007_abl_repairfield_enter            bc9ffa2c   4 wem
	play_007_abl_repairfield_exit             d9832d6e   4 wem
	play_007_abl_repairfield_device_impact    400b2b01   4 wem
	play_007_abl_grapplinghook_deploy_player  34df0941   3 wem
	play_007_abl_grapplinghook_impact_generic 00de776e   3 wem
	play_007_abl_knockback_pulse_player       5fa136df   3 wem   repulseur
	play_007_abl_activecamo_activate_player   aa932bc4   1 wem
	play_007_abl_activecamo_deactivate_player afde714f   1 wem
	play_007_abl_overshield_start_player      09eec642   1 wem
	play_007_abl_overshield_end_player        40d46c8f   1 wem
	play_007_abl_shared_pickup                c73036e4   2 wem
	play_007_abl_shared_device_explode        9095fa10   3 wem

**LE GESTE DE POSE DE L'ECRAN OCCULTANT EST DESIGNE** (`shroud_deploy_player`), sans avoir a
soumettre 38 fichiers a l'oreille. Le TRANSLOCATEUR, lui, garde ses 23 evenements : un seul
est nomme (`play_007_abl_quantum_portal_warning`, obtenu a esperance 0,25 — SECOND RANG, a
confirmer), donc **la designation du geste de pose du translocateur reste une ECOUTE**, comme
le disait deja le lot du 18/08.

## 4. LA DISTINCTION PAR EQUIPE — REPONSE MESUREE : OUI

La modulation terminale des noms d'evenements est `_team` / `_enemy`, et les deux existent
cote a cote sur **six couples** : `ctf_flag_scored`, `ctf_flag_taken`, `ctf_flag_pickup`,
`assault_bomb_taken`, `assault_bomb_pickup`, `extraction_point_scored`. Ce sont des
evenements Wwise DISTINCTS, portant des `.wem` DIFFERENTS — pas un meme son module par un
Switch. Le jeu joue donc bien deux sons differents selon que l'action est faite par votre
equipe ou par l'adversaire.

**CE QUI N'A PAS DE VARIANTE D'EQUIPE, et c'est une information en soi** :
`ctf_flag_returned`, `ctf_flag_spawn`, `assault_bomb_detonated`, `assault_bomb_disarm_loop`,
`assault_bomb_planted_loop`, `extraction_zone_spawn` — un seul son pour tout le monde.

## 5. BOBINES — LE SON DU KILL

**LA CHAINE ETAIT DEJA FAITE ET VERSIONNEE**, elle n'avait simplement jamais ete lue comme une
reponse a cette demande : `internal/games/halo_infinite/film/damagetag/data/labels.tsv`
porte, pour la classe `OBJET_EXPLOSIF`, deux modeles destructibles (`hlmt 95b23ee5` et
`hlmt a2fb33e4`) qui exposent les MEMES cinq variantes d'energie, et la banque de chacune :

	etat de degat   flaveur         pack                                    bobine
	0/7             hardlight       sb_008_exp_single_small_hardlight       Bobine a lumiere dure
	1/7             kineticunsc     sb_008_exp_single_small_kineticunsc     Bobine explosive (Blast)
	2/7             plasma          sb_008_exp_single_small_plasma          Bobine a plasma
	3/7             (aucune banque atteinte, 4 weap)                        5e variante, NON RESOLUE
	4/7             shock           sb_008_exp_single_small_shock           Bobine a choc

Le sens de l'index est CLOS depuis `killweapon/RE_LOG_KILLWEAPON.md` 7ter.57 : c'est le TYPE
D'ENERGIE de l'objet, et la flaveur de banque le decrit (l'utilisateur avait tranche : « c'est
le meme objet, il a juste un type d'energie dedans »). Le `jpt!` lu est celui du degat qui a
TUE ; la banque qu'il atteint est donc bien le son de l'explosion qui tue.

**RESERVE A DIRE** : la chaine etablit l'ENERGIE, pas le nom commercial. Trois noms sur quatre
SONT l'energie (lumiere dure, plasma, choc) ; « Blast Coil = kineticunsc » est une deduction
par elimination, pas une chaine.

**STRUCTURE DES QUATRE BANQUES, mesuree** : chacune porte exactement **3 evenements** —
deux de la forme « 1 couche, un son tire parmi 6 a 8 » et **un seul de la forme « 5 couches
SIMULTANEES »** (4 couches de 6 a 8 variantes + 1 son). C'est ce dernier qui est l'explosion
complete ; les deux autres sont des perspectives. Aucun de ces noms d'evenements ne casse
(base probable differente de celle des modes).

	banque      sbnk        evenement a 5 couches
	hardlight   52ca7622    4429b959
	kineticunsc 3435ff42    30343586
	plasma      9a28782d    db64918e
	shock       6fd78d85    ae8da132

## 6. NEGATIFS, avec leur denominateur

1. **AUCUNE BANQUE NE S'APPELLE `sb_004_mod_mp_strongholds` — MAIS LES SONS DE BASTION
   EXISTENT, ET LEUR BANQUE EST TROUVEE.** Le negatif de nommage tient : sur les 1697 banques
   des dix modules balayes, aucune ne porte l'identifiant `fnv1("sb_004_mod_mp_strongholds")`,
   ni celui d'une variante A UN JETON tiree des identifiants du binaire (le mot `strongholds`
   en fait partie, il est present en `14381bb28`). **Ce qui etait faux, c'est la conclusion
   qu'on en tirait.** Le balayage structurel complet (`eqip-arbre -banks all` : 1 645 banques,
   **6 819 evenements**, quelques secondes) permet de chercher un NOM D'EVENEMENT dans TOUT le
   jeu sans savoir dans quelle banque il vit — et `play_004_mod_mp_strongholds_contested` est
   tombe ainsi. **La banque est `1c609526`** (module `common`, **88 evenements, 84 `.wem`**),
   et son nom de fichier n'est simplement pas celui du mode. Deux autres noms ont suivi :
   `play_004_mod_mp_strongholds_zone_exit_team` et `..._enemy` — la modulation d'equipe existe
   donc aussi sur les zones.
   **LA CAPTURE DE ZONE, ELLE, RESISTE, et voici son denominateur.** Trois passes sur ces
   88 evenements : vocabulaire curie (esperance 0,014), les 135 580 identifiants du binaire
   (0,072), puis les **142 023** obtenus en decoupant AUSSI le camelCase
   (`StrongholdCaptures` -> `stronghold` + `captures`), soit **2 982 483 candidats a esperance
   0,061**. Les trois memes noms sortent, jamais un quatrieme. La voie du hachage est epuisee
   a discipline constante ; la suivante est l'OREILLE (`RECETTE_SONS_ARMES` §5).
   A NOTER, et c'est l'utilisateur qui l'a releve : `play_004_mod_mp_attrition_enemy_captured`
   est vraisemblablement PROPRE au mode Attrition (« il me semble n'avoir jamais entendu ce
   son »). Il n'est donc pas le son de capture generique des zones.
2. **La banque `7acb11cc` (capteur de menaces + traqueur) ne se nomme pas.** Elle a resiste au
   dictionnaire complet du binaire sur le prefixe `sb_007_abl_`. Son rattachement, lui, reste
   etabli par la chaine de tags du lot du 18/08.
3. **`sb_004_mod_mp_shared_ui` (33 `.wem`, 20 evenements) ne livre aucun nom d'evenement.**
   C'est la banque la plus probable pour les stingers d'objectif generiques ; sa grammaire
   d'evenements n'est pas celle des banques de mode.
4. **Deux noms de banque sont des COLLISIONS FORTUITES ASSUMEES**, et il faut les dire parce
   qu'elles calibrent le bruit : `sb_020_prototype_weapon_hkpballandsocketconstraintdata`
   (0 `.wem`, nom absurde tire d'une classe Havok) et `sb_004_mod_cv_objective_threat_seeker`
   (0 `.wem`). Elles sortent des passes a esperance elevee, se reconnaissent a leur incoherence
   ET a leur banque VIDE, et n'entrent nulle part.

## 7. Ce qui est extrait sur disque (scratchpad de session)

	wem_globals/    517 .wem   14 banques de `pc/globals`   (ctf, assault, shared_ui, shroud,
	                                                         quantum, repairfield, ...)
	wem_common/     129 .wem    7 banques de `common`        (extraction, landgrab, ...)
	wem_coils/      248 .wem    4 packs d'explosion de bobine + repairfield

**LE DECODEUR A DU ETRE REINSTALLE, ET IL FAUT LE DIRE** : les `.wem` sont en Wwise Vorbis
(`fmt` = `0xFFFF`, verifie sur piece). `ffmpeg` (8.0.1) NE LES DECODE PAS — il rend
`Audio: none ([255][255][0][0] / 0xFFFF), unknown codec`. Le seul decodeur est
`vgmstream-cli.exe`, que la recette cite mais qui n'etait plus sur cette machine (le dossier
`Desktop/Halo Infinite - Sons armes/_outils/` n'existe pas ici). **Reinstalle le 2026-08-26**
depuis la release GitHub officielle du projet (`vgmstream/vgmstream`, `r2117`,
`vgmstream-win64.zip`) vers `C:\Users\Guillaume\Downloads\vgmstream\`. La recette
(`RECETTE_SONS_ARMES.md` §1) doit desormais pointer ce chemin.

**LES 39 EVENEMENTS SONT RENDUS** (2026-08-26) : `C:\Users\Guillaume\Downloads\Halo Infinite -
Sons v75\` porte 74 `.wav` (une a trois variantes par evenement), les `.mp3` d'ecoute, les
fiches et la planche de validation. Rendu = une variante par couche, gain de chemin applique,
somme a t = 0, puis gain LINEAIRE strict jusqu'a -1 dBTP. Decodage vgmstream r2117, mixage et
mesure de crete par ffmpeg 8.0.1.

## 8. LES PARAMETRES DE RENDU SONT DANS LE FORMAT, ET ILS SONT RELEVES

Un `.wem` isole n'est pas le son du jeu : le moteur EMPILE des couches et applique un gain de
chemin a chacune. Le mode `eqip-arbre` releve deja tout ce qu'il faut pour reconstruire un
evenement a l'identique, et l'annexe
`RE_FICHES_RENDU_SONS_2026-08-26.txt` porte les **39 fiches** des evenements cibles :

	par COUCHE     type de conteneur (Sound / RandomSequence / Switch / Blend)
	               -> RandomSequence = UNE variante tiree ; couches = SOMME a t=0
	par `.wem`     gain de chemin en dB (somme evenement -> Sound)
	par COUCHE     delai de debut en secondes
	par COUCHE     fourchette RANGED (volume dB / hauteur en cents, tiree A CHAQUE LECTURE)

**DEUX MESURES QUI CHANGENT LE RENDU, et qui distinguent ces sons de ceux des armes :**

1. **AUCUNE des couches des 39 fiches ne porte de delai** (0 sur 275 couches relevees) :
   l'empilement se fait a t = 0, sans decalage.
2. **AUCUNE ne porte de paquet RANGED** (0 sur 275). Les sons d'objectif, d'equipement et
   d'explosion de bobine se jouent donc **PURS** — le moteur ne deplace ni leur volume ni leur
   hauteur d'une lecture a l'autre. C'est l'inverse des armes, ou la fourchette RANGED est ce
   qui empeche une rafale de sonner comme une photocopie. Consequence directe : pour ces
   familles, le reglage « variation » de la page admin n'a rien a rejouer, et un fichier rendu
   une fois est fidele.

Exemple, la capture de drapeau par MON equipe : deux couches simultanees, une a +15 dB
(`757373997`), une a 0 dB (`728789050`) — additionnees, pas jouees en alternance.

Exemple, l'explosion d'une bobine a choc : **cinq couches simultanees**, quatre d'entre elles
tirant une variante parmi huit, aux gains +6 / +7 / +16 / +9 dB, plus une couche fixe a -8 dB.
Rendre un seul de ces 33 `.wem` ne donne pas l'explosion ; les additionner aux gains releves,
oui.

## 9. Outillage — ce qui est versionne

	cmd/weapon-sounds -mode banks-noms   `banks_noms.go` + `banks_dico.go`
	  1 passe par module, imprime CALIBRATION puis ESPERANCE puis resultats, dans cet ordre.
	  Sortie JSON {sbnk_gid, id_wwise, nom, provenance, wem_embarques} : tout le cassage
	  ulterieur se fait HORS module, sur ce JSON, en quelques secondes.

	go run ./cmd/weapon-sounds -mode banks-noms \
	  -module "pc/globals/globals-rtx-new.module" -etroit -json <sortie>.json

Les deux crackers d'iteration (noms de banques a partir du vocabulaire du jeu, noms
d'evenements par banque) sont restes dans le scratchpad de session : ce sont des instruments
de fouille, pas des outils de production. Ce qui merite d'etre versionne le jour ou on les
rejoue, c'est la GRAMMAIRE mesuree (section 3), pas le code jetable.
