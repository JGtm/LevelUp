# PLAN — Extraction des sons de tir par arme (tags `sbnk`)

> Branche : `feat/extraction-sons-armes` — worktree `LevelUp-wt-replay2d`
> Ouvert le 2026-08-15. Contrat d'execution : skill `plan-execution` (fait foi).

## Probleme

Les `.pck` du jeu (`Sound/win/SFX/`) livrent 90 170 `.wem` sans aucun nom : les banks
Wwise n'y sont pas (0 bank sur 1645 `.pck`) et les noms de tags sont absents des modules
(`stringsSize` = 0 sur les 132 modules, 0 chemin ASCII sur 665 Mo scannes). Un pack par
arme identifie l'ARME de facon certaine, mais rien n'identifie le TIR parmi les 80 a 360
sons du pack. Le tri par duree ne generalise pas : le fusil d'assaut utilise 296 micro
echantillons de 0,08 s en round-robin, le sniper des one-shots de 4,5 s avec queue.

## Hypothese de travail (mesuree, pas postulee)

Les banks Wwise ont ete converties en tags `sbnk` dans les modules. Mesures a l'appui :

- `pc/globals/globals-rtx-new.module` contient **1305 tags `sbnk`** (et 14 228 `snd!`,
  111 `weap`) — a comparer aux 1645 `.pck` du dossier `Sound`.
- Les **40 IDs `.wem` du fusil d'assaut** cherches en clair dans les modules sont tous
  localises dans des tags `sbnk` de ce module.
- Les 1305 `sbnk` sont **tous compresses Oodle** (aucun `cs == us`) : `internal/ooz` est
  obligatoire. Verifie : `go build ./internal/ooz/` passe (msys64 ucrt64 g++).

Si le contenu decompresse est une bank Wwise (`BKHD`/`HIRC`), on remonte
Evenement -> Conteneur -> Son -> `.wem`, et le tir devient identifiable de facon certaine.

## Outillage deja disponible (ne rien reecrire)

| Piece | Emplacement | Role |
|---|---|---|
| `hmod.go` | `cmd/weapon-icons-build/` | lecteur `.module`, `dataOffset` 48 bits, blocs |
| `internal/ooz` | `apps/go-api/internal/ooz/` | decompression Kraken (CGO) |
| `hdr.go` | `cmd/weapon-icons-build/` | header `ucsh` + table de dependances typees |
| `weap.xml` | `cmd/weapon-icons-build/` | definition du tag `weap`, champs nommes |

## Etapes

### Etape 1 — Sonde : format reel du contenu d'un `sbnk`

Gate : `go run ./cmd/weapon-sounds -mode probe` statue OUI/NON sur la presence de
`BKHD`/`HIRC` et designe le `sbnk` portant les `.wem` du fusil d'assaut. PASSE.

- [x] Squelette `cmd/weapon-sounds` qui reutilise `himodule` + `ooz` (pas de copie)
- [x] Decompression des `sbnk` et recherche des signatures Wwise
- [x] Verdict ecrit au journal : bank Wwise VERBATIM

DECISION DE BIFURCATION : sans objet, `HIRC` est present. L'etape 2 est nominale.

### Etape 2 — Parser la hierarchie et produire evenement -> liste de `.wem`

Gate : dump JSON pour `sb_010_wea_un_assaultrifle`, non vide, dont les IDs appartiennent
tous au `.pck` de cette arme (controle croise). PASSE.

- [x] Parser des objets HIRC utiles (Event, Action, Random/Sequence Container, Sound)
- [x] Resolution transitive Event -> ensemble de `.wem`
- [x] Controle croise : 0 `.wem` hors du `.pck`, couverture 359/359

### Etape 3 — Designer l'evenement de TIR parmi les 22

Gate : au moins le fusil d'assaut, le sniper et le needler ont leur evenement de tir
designe, par une preuve qui ne repose pas sur la duree des sons.

Voie A — hachage FNV-1 (TENTEE, INSUFFISANTE SEULE) :

- [x] Generateur de candidats : noms de `.pck` x verbes x prefixes usuels Wwise
- [x] Hachage FNV-1 32 bits minuscule, appariement sur les IDs d'evenements
- [!] Resultat : **0 nom retrouve sur 18 evenements**. La fonction de hachage est
  PROUVEE correcte (vecteurs de reference + calibrage sur l'ID de bank, cf. journal) :
  c'est la forme des noms d'evenements qui n'est pas devinable. Voie conservee comme
  complement opportuniste, abandonnee comme moyen principal.

Voie B — le graphe de tags (MOYEN PRINCIPAL) :

- [~] Localiser les tags `snd!` qui referencent les evenements — REFUTE : ce ne sont pas
      des `snd!` mais des `lsnd`. Couvert par la remontee ci-dessous.
- [x] Remonter au tag `weap` : chaine etablie et mesuree
      `sbnk 384b727f` <- 8 `lsnd` <- 2 `weap` (`00008595`, `48c19d2d`)
- [x] Lire le champ NOMME « Weapon Fire Sound » de `weap.xml` : 3 `lsnd` sur 8 designes
      (`3fd85fcd`, `5a2b4a0e`, `625034f9`). Preuve fermee, sans aucun nom Wwise.

### Etape 12 — CARTOGRAPHIER LA HIERARCHIE AVANT DE LA CORRIGER

Demande explicite de l'utilisateur apres le quatrieme oubli de format : « ce sont des
erreurs qu'on aurait pu eviter si tu avais cherche a mieux comprendre tous les elements ».
Cette etape inverse donc l'ordre habituel : **aucun correctif tant que l'inventaire des
conteneurs n'est pas complet et statue**.

Ce qui a ete manque jusqu'ici, toujours de la meme facon — implementer le strict necessaire
a l'objectif du moment : medias embarques `DIDX`, tableau des modes de tir, type d'Action,
conteneurs `Switch`. Le mode `audit` couvrait deja les chunks, les types d'objets, les types
d'action et la charge utile des `Sound`. Il ne dit RIEN des CONTENEURS, et c'est precisement
la que se cachait le quatrieme oubli.

Gate 12a : `-mode audit` publie, pour CHAQUE type de conteneur, la taille de charge utile,
la position de la liste d'enfants trouvee, et **le nombre d'octets restants non lus apres
elle**. Un conteneur dont on ignore la moitie de la charge utile doit se voir.

- [x] Inventaire des conteneurs : effectif, taille de charge utile, octets non lus
      (`audit_conteneurs.go`, section « CONTENEURS » du mode `audit`)
- [x] Decodage de `AkSwitchPackage` (groupe, etat par defaut, enfants par etat), valide
      contre la liste d'enfants deja resolue — jamais postule (`conteneurs.go`)
- [x] Meme traitement pour `RandomSequence` et `Blend` (`conteneurs_autres.go`)
- [x] Verdict ecrit (ci-dessous)

VERDICT 12a — inventaire des conteneurs sur les 1305 banks. Les colonnes « avant » et
« apres » sont les octets de charge utile que le parseur ne lit pas, autour de la liste
d'enfants qu'il localise :

	type                   n   taille  avec enf.     avant     apres  (max apres)
	Settings           55831        8         0%         0         0
	ActorMixer          9037      189        99%       167         0
	RandomSequence      7069      141        98%        79        40  (546)
	Attenuation         4207      156         0%         0         0
	MusicTrack          4059      166         0%         0         0
	MusicSegment        3970      137        99%        33        96  (3879)
	MusicRanSeq          510      437       100%        37       365  (2529)
	Switch               445      223       100%        69       135  (970)
	Blend                303      125       100%        55        56  (373)
	Bus                  241      387         4%        52       239  (399)
	MusicSwitch           85     2030       100%        45      1953  (18928)

STATUT DE CHAQUE CONTENEUR, dans l'ordre demande par le gate :

- `Switch` — **A LIRE, et desormais LU.** 440 tables etat -> enfants sur 445 decodees ET
  validees (99 %), dont 433 ou l'etat par defaut se recoupe avec la table. Moyenne 6,1
  etats par conteneur, maximum 37. **200 etats sont declares SANS aucun enfant** : un
  `Switch` peut donc ne rien jouer, ce qu'un tirage aleatoire ne reproduit jamais.
  Le compte tombe juste sur le cas du sniper : 30 enfants pour ~6 etats, soit 5 variantes
  par etat. Le rendu piochait dans les six etats a la fois.
- `RandomSequence` — **IGNORE AVEC RAISON, mesure a l'appui.** La table de poids est lue et
  validee sur 6991 conteneurs : **6976 sur 6976 ont des poids tous egaux**. Tirer
  uniformement est donc exact, et le rester est un choix, plus un oubli.
- `Blend` — **A LIRE, portee bornee.** 303 tables sur 303 decodees. La majorite ne declare
  AUCUNE couche : « le Blend joue ses enfants tels quels » y est exact. Mais **42 (14 %)
  declarent une automation par parametre de jeu** : pour ceux-la, les couches sont fondues
  selon un etat (typiquement la distance) et les empiler a plein niveau est faux. Effet
  sur le NIVEAU relatif, pas sur la presence — donc moins grave que le `Switch`.
- `ActorMixer` — 167 octets avant la liste d'enfants, **0 apres**. Rien ne manque en aval ;
  l'amont est le bloc de parametres de noeud, dont le volume est deja lu par ailleurs.
- `Settings`, `Attenuation`, `MotionBus`, `MotionFX`, `Effect`, `Envelope` — aucun enfant,
  hors chaine de lecture des sons d'arme. **IGNORE AVEC RAISON.**
- `MusicTrack`, `MusicSegment`, `MusicRanSeq`, `MusicSwitch` — hierarchie MUSIQUE. Gros
  restes non lus (jusqu'a 18 928 octets), sans objet pour les armes. **IGNORE AVEC RAISON.**
- `Bus` — 4 % seulement portent une liste d'enfants ; 239 octets non lus en moyenne. Les bus
  portent les effets et les attenuations globales. **NON INSTRUIT** : c'est la piste a
  ouvrir si le `.wem` `195277626` partage par 20 armes (defaut 10) s'avere etre un envoi.

Un point de methode a garder : le lecteur `Blend` a echoue sur 303 conteneurs sur 303 avant
qu'on regarde les octets. Il validait `ulLayerID` contre la liste d'enfants, alors que c'est
l'identifiant PROPRE de la couche. Deux hypotheses successives ont echoue avant qu'un simple
vidage hexadecimal ne tranche en une lecture — **montrer les octets aurait du etre le
premier reflexe, pas le troisieme.**

Gate 12b : le rendu d'un coup n'utilise plus un tirage dans tout le lot d'un `Switch` mais
l'etat designe. Controle sur pieces : le sniper (couche `Switch` a 71 % du melange) et le
Needler (couche `Switch` de la supercombinaison).

- [x] Rendu par etat, avec l'etat par defaut quand aucun n'est impose
      (`bank.go` -> `resoudreSwitch`, trois issues COMPTEES : etat porteur, etat vide,
      table non decodee)
- [x] Regeneration et mesure de l'ecart avec les rendus actuels, arme par arme
- [x] Prevenir l'utilisateur des votes de 1re personne a rejouer

VERDICT 12b — 28 coups de 1re personne changent, tous dans le meme sens : le lot de
candidats se reduit d'un facteur 3 a 5, celui d'UN etat au lieu de tous.

	sniper    couche Switch  30 -> 6 candidats     evenement : 32 -> 8 wem
	shotgun   couche Switch  30 -> 6               skewer    : 32 -> 8 wem
	needler   couche Switch  40 -> 8               MA40      : 133 -> 45 wem
	tourelle gatlingmortar   122 -> 36 wem         MA5K      : 133 -> 45 wem

Le compte annonce a l'inventaire se verifie : 30 enfants pour ~6 etats, et l'etat par
defaut en porte 6.

RESULTAT NON PREVU, et c'est la meilleure preuve du correctif : **le Needler RETROUVE sa
1re personne**. Son evenement `7474d8d8` etait refuse par le garde-fou de duree (rapport
26) parce que sa couche `Switch` de 40 sons melangeait tous les etats, dont la
supercombinaison a 1,5-3,9 s. Elaguee a l'etat par defaut, la mediane redescend et
l'appariement passe. **Le defaut de perspective et le defaut `Switch` n'en faisaient
qu'un** — le garde-fou de l'etape 10 ne soignait bien que le symptome. Les refus tombent
de 11 a 10, et le Needler n'en fait plus partie.

Aucun vote orphelin : 44/44 se rattachent toujours.

FAUSSE ALERTE consignee pour memoire : la premiere regeneration rendait 53 armes au lieu de
55. Ce n'etait pas une regression du correctif mais un oubli du drapeau `-banks` a la ligne
de commande — les deux banks orphelines (Mutilator `8827aa7e`, Carabine Vestige `09089e7e`)
ne sont recoltees que si on les nomme. La commande complete :

	go run ./cmd/weapon-sounds -mode lot -module pc/globals/globals-rtx-new.module 	  -pck "<...>/Sound/win/SFX" -banks "8827aa7e,09089e7e" -json <...>/lot1.json

### Etape 4 — Generalisation aux 55 packs

Gate : un rapport par arme listant les `.wem` de tir, produit en UNE ouverture de chaque
module. PASSE.

- [x] Passe 1 (`lot`) : 55 packs en une ouverture du module de 7,24 Go — 53 armes resolues
- [x] Passe 2 (`lot-tir`) : 33 armes avec sons de tir prouves, de 8 % a 94 % du pack
- [x] Croisement avec le `weap` global tag id (cle deja etablie par `weaptags.go`) :
      23 armes rattachees a leur nom en jeu et a leur icone
- [x] Cadence de tir lue dans le tag (`barrels` -> `rounds per second`) : 22 armes
- [ ] Export audio des seuls sons de tir (non demarre)

## Journal

### 2026-08-15 — Ouverture

Prerequis verifies sur pieces avant ouverture : `ooz` compile ; 1305 `sbnk` denombres ;
IDs `.wem` du fusil d'assaut localises dans des `sbnk`. Branche creee depuis
`71bdb589f` (worktree en HEAD detache auparavant).

### 2026-08-15 — Etape 1 CLOSE : le `sbnk` est une bank Wwise verbatim

Commande : `go run ./cmd/weapon-sounds -mode probe -limite 1305`.

Mesure sur les 1305 `sbnk` de `pc/globals/globals-rtx-new.module` : **1305 decompresses,
0 echec**, en-tete `ucsh` sur 1305/1305. Signatures : `BKHD` 1299, `HIRC` 1296,
`DIDX` 694, `DATA` 694, `STMG` 3, `STID` 2. La charge utile du tag EST la bank, pas un
format maison — et elle est dans les octets du tag, pas dans le blob de ressources
(`res = 0 o` sur le tag temoin ; la sonde des ressources est restee, elle ne coute rien).

Le `sbnk` du fusil d'assaut est designe sans ambiguite : **`gid 384b727f`** (1 536 586 o)
est le SEUL des 1305 a porter les 6 `.wem` temoins.

Correctif d'ergonomie inclus : `-limite` par defaut passe de 60 a 0 (= tous). L'heuristique
initiale « une bank d'arme est petite, sonder les plus petites » est REFUTEE — la bank du
fusil d'assaut fait 1,5 Mo et etait absente des 60 plus petites, d'ou un premier verdict
« HIRC absent » qui etait un artefact d'echantillonnage.

Consequence pour l'etape 2 : `STID` n'est present que sur 2 banks. La table des noms
d'evenements est donc quasi absente — l'etape 3 (hachage FNV-1) reste bien necessaire.

### 2026-08-15 — Etape 2 CLOSE : hierarchie resolue, 359/359 couverts

Commande : `go run ./cmd/weapon-sounds -mode map -pck <...>sb_010_wea_un_assaultrifle.pck`.

Mesure : `sbnk` gid `384b727f` designe par intersection des IDs (aucun nom en jeu),
642 objets HIRC, 391 Sound resolus, 22 Events, 35 Actions. **Couverture 359/359** `.wem`
du `.pck` atteints depuis un evenement, **0 `.wem` hors du `.pck`**.

DECISION DE CONCEPTION : le parseur ne postule aucun offset. Les trois lectures ambigues
du format HIRC (dependant de la version de Wwise, non exposee) sont ESSAYEES PUIS VALIDEES
contre un ensemble connu : `sourceID` de Sound valide par l'appartenance aux IDs du `.pck` ;
liste d'enfants d'un conteneur retenue seulement si TOUS ses elements sont des objets de la
bank (on garde la plus longue) ; liste d'Actions d'un Event retenue seulement si tous ses
elements sont des objets de type Action. Le controle croise a 0 hors-pck valide l'approche :
une lecture au mauvais offset n'aurait pas survecu.

Profil des 22 evenements : 8 gros (64 a 103 `.wem`) puis une queue a 4 `.wem`. Les gros sont
les candidats tir (round-robin), les petits la mecanique. **Les departager exige les noms**
— c'est l'objet de l'etape 3, elle n'est pas contournable.

### 2026-08-15 — Etape 3 : la voie du hachage echoue, changement de moyen

CORRECTION D'UNE AFFIRMATION DE L'ETAPE 1 : le journal annoncait « STID 2, STMG 3 ». C'est
FAUX — la sonde cherchait ces quatre lettres N'IMPORTE OU dans les octets (`bytes.Contains`),
pas un vrai chunk. Le mode `noms`, qui decode reellement les chunks, ne trouve **aucun**
`STID` sur les 1305 banks. Il n'y a donc aucun nom en clair nulle part dans le jeu, ce qui
est coherent avec `stringsSize` = 0 sur les modules.

Voie A (hachage) : **0 nom retrouve sur 18 evenements**. Deux garde-fous prouvent que
l'echec ne vient ni de la fonction ni d'un bug :

1. `noms_test.go` verifie FNV-1 sur les vecteurs standards ("" -> 811c9dc5, "a" -> 050c5d7e,
   "foobar" -> 31f0b262) et l'insensibilite a la casse ;
2. calibrage sur une donnee dont le nom EST connu — l'identifiant de bank du chunk `BKHD`
   vaut `4f8f2090`, et `fnv1("sb_010_wea_un_assaultrifle")` vaut exactement `4f8f2090`.

La convention est donc bien « FNV-1 32 bits du nom complet en minuscules ». Ce qui manque,
c'est la FORME des noms d'evenements, et elle n'est pas devinable a partir du nom de bank.

DECISION : la voie principale devient le graphe de tags. Le raisonnement : le tag `weap`
porte des champs NOMMES (`weap.xml`, 84 Ko de definitions deja au depot) ; un de ces champs
designe le son de tir ; il pointe vers un `snd!` ; le `snd!` porte un identifiant d'evenement
Wwise. La preuve ne repose alors sur aucun nom Wwise ni sur aucune heuristique acoustique —
elle vient du champ nomme de la definition de tag.

### 2026-08-15 — Etape 3 voie B : la chaine arme -> bank est etablie

Trois hypotheses REFUTEES avant la bonne, chacune par la mesure :

1. « les `snd!` portent les identifiants d'evenements » — FAUX : 0 porteur sur 14 228 ;
2. « les `snd!` portent l'identifiant de la bank » — FAUX : 764 banks distinctes
   referencees par les `snd!`, et `384b727f` n'en fait PAS partie ;
3. « les `sbnk` et les `snd!` cohabitent dans un module » — FAUX : `pc/globals` ne contient
   que des assets plateforme (17 803 `bitm`, 1875 `mode`, 1305 `sbnk`, 791 `shdv`), les
   `snd!`/`weap` vivent dans `any/globals`.

La question posee A L'ENVERS a tranche : « qui depend de `384b727f` ? » rend **8 tags
`lsnd`** (sons en boucle), pas des `snd!`. Un niveau plus haut : **2 tags `weap`**
(`00008595` et `48c19d2d`) et 1 `stai`. La chaine est donc :

	sbnk 384b727f  <-  8 lsnd  <-  2 weap

Correspondance notable : 8 `lsnd` pour 8 evenements « gros » (64 a 103 `.wem`) mesures a
l'etape 2. Elle est encourageante mais N'EST PAS une preuve — l'appariement reste a faire.

Deux `weap` et non un : attendu, le fusil d'assaut a des variantes (le `.pck` du MA5K a la
meme structure et le meme compte de sons). Le global tag id d'un `weap` est deja la cle
etablie par `cmd/weapon-icons-build/weaptags.go` (32 bits hauts d'un identifiant filmshell),
donc ces deux identifiants se raccordent au referentiel d'armes du projet — c'est ce qui
donnera le NOM EN JEU, que le nom interne du `.pck` ne donne pas.

MEMOIRE (garde-fou demande) : instrumentation `rapporterMemoire` a chaque palier. Pic
mesure 1,3 Go systeme sur le module `any` (0,62 Go) en balayant les 78 174 entrees. Les
modes ne chargent JAMAIS deux modules dans le meme processus : `map` ouvre celui de 7,24 Go,
`lien`/`qui`/`remonter` celui de 0,62 Go, et l'echange se fait par le fichier JSON.

### 2026-08-15 — Etape 3 CLOSE : le champ nomme designe le tir, la chaine se recoupe

Le plugin place « Weapon Fire Sounds » a +4288 ; AUCUN tableau du tag reel n'occupe cet
offset (59 tableaux dans le bloc racine, le plus proche est a +4352). Meme derive de version
que celle documentee dans `weapon-icons-build/weapui.go`. Parade identique : identification
PAR CONTENU, avec une signature verifiable — un element de « Weapon Fire Sounds » porte, a
l'offset du sous-tableau « Variations », des entrees de 28 octets dont CHAQUE reference
pointe vers un tag de son. **Un seul tableau sur 59 la satisfait** (celui a +4352).

Resultat : 3 `lsnd` sur les 8 (`3fd85fcd`, `5a2b4a0e`, `625034f9`).

RECOUPEMENT INDEPENDANT (ce qui donne confiance) : en scannant les tags de son pour les
identifiants d'evenements du rapport `map`, les 3 `lsnd` de tir portent 4 evenements chacun
et les 5 autres 2 chacun. 3x4 + 5x2 = 22, soit EXACTEMENT le nombre d'evenements mesure a
l'etape 2 par un chemin totalement different (parsing HIRC de la bank). Les 8 evenements des
`lsnd` de tir sont precisement les 8 « gros » (64 a 103 `.wem`), les 14 autres sont les
petits (4 `.wem`). Les deux moities de la preuve concordent sans avoir ete ajustees.

Livrable mesure (mode `final`) pour `un_assaultrifle` / weap `00008595` :
**339 `.wem` de tir sur 359 (94 %)**, repartis en 8 evenements.

Ce taux est ELEVE mais coherent : le pack du fusil d'assaut est presque entierement du
coup de feu (mesure acoustique prealable : 296 echantillons de 0,08 s sur 359). Les 20
`.wem` restants sont les evenements a 4 sons — la mecanique.

RESERVE A LEVER : `evenementsDesTags` relie un `lsnd` a ses evenements par recherche de
l'identifiant en clair (`bytes.Contains`), pas par un champ structure. Sur 18 candidats et
des tags courts, un faux positif est peu probable mais PAS exclu. Le recoupement 22 = 22
ci-dessus est l'argument qui rend le resultat credible, pas la methode de recherche.

### 2026-08-15 — Etape 4 : 33 armes prouvees, cadence lue dans le tag

Passe 1 (`lot`) : 55 packs en UNE ouverture du module de 7,24 Go, deux sous-passes ou une
bank n'est jamais retenue en memoire (on ne garde qu'un index par pack, puis on re-extrait
les 55 retenues une a la fois). **Pic mesure 8,8 Go** — c'est le plafond de l'approche, il
tient dans les 16 Go libres de la machine mais ne laisse pas de marge. Passe 2 : 0,7 Go.

53 armes resolues sur 55. Quatre ratés consignes : `cv_shadeturret` (bank mal appariee,
score 1), `cv_provoker_megatron` (meme bank que `cv_provoker`), `cv_plasmapistol_overcharged`
et `un_shared_rocket` (aucune bank).

Passe 2 : **33 armes avec sons de tir prouves**, de 8 % a 94 % du pack. Les 20 sans tir sont
coherentes : melee (epee, marteau), projectiles, sifflements — pas de champ « Weapon Fire
Sound » parce qu'il n'y a pas de tir.

CADENCE. Champ `barrels` -> struct inline `firing` -> `_25 « rounds per second »` (8 octets,
deux flottants). Validee sur deux armes connues avant generalisation : MA40 AR 720 coups/min,
S7 Sniper 67 — les valeurs du jeu. Le tableau `barrels` est a +3284 alors que le plugin
annonce +3220 : **exactement la meme derive de +64** que « Weapon Fire Sounds » (+4288 ->
+4352). Deux champs independants, meme decalage — la derive est systematique, pas un hasard.

DEUX DEFAUTS TROUVES ET CORRIGES pendant cette etape :

1. `sort.Slice` n'est pas stable et plusieurs `weap` couvrent souvent le meme nombre
   d'evenements : le tag retenu CHANGEAIT d'un run a l'autre, donc la cadence affichee aussi
   (le pistolet a plasma passait de 405 a 1800 coups/min entre deux executions identiques).
   Depart au second critere sur l'identifiant. Determinisme verifie : deux runs, hachages
   identiques.
2. Le seuil de plausibilite de la cadence etait trop large. Les valeurs se repartissent en
   deux paquets separes par un vide mesure : cadences reelles de 1,11 a 20,00 coups/s, puis
   ~30,00 pile porte par des armes A UN COUP (Empaleur, canon Gauss, tourelle a rayon).
   30 est la valeur non bridee du moteur, pas un rythme. Seuil pose a 25, dans le vide entre
   les deux paquets — et non a « == 30 », la valeur float32 s'en ecartant parfois assez pour
   arrondir a 1800 sans y etre egale. Resultat : 22 cadences retenues, 11 ecartees.

### 2026-08-15 — Etape 5 : un tir est un EMPILEMENT, et la moitie des sons manquait

Deux defauts de conception, tous deux revelés par une question de l'utilisateur (« un tir,
ce serait pas la combinaison de plusieurs sons ? »), tous deux confirmés sur pieces.

**1. Le sac plat.** `wemsDeEvent` rendait l'ENSEMBLE des `.wem` atteignables depuis un
evenement, sans distinguer les deux natures de conteneur Wwise : un `RandomSequence` en mode
aleatoire joue UN enfant, un `Blend` les joue TOUS. Or un evenement porte plusieurs ACTIONS
qui se declenchent EN PARALLELE. Mesures : le tir du Skewer a 3 couches (un `Switch` de 5
branches + 2 `Sound` fixes), celui du MA40 en a 4, et l'evenement `546d8a24` du Skewer est un
`Blend` de 18 enfants. Un coup entendu est donc la SOMME d'une variante par couche — aucun
`.wem` isole ne peut sonner juste. Ajout de `couchesDeEvent` (une entree par action) et du
mode `arbre` qui affiche la hierarchie typee.

**2. Les medias embarques, ignores.** Une bank Wwise peut porter ses propres `.wem` dans ses
chunks `DIDX`/`DATA`, absents de tout `.pck`. Le validateur du parseur n'acceptait un
`sourceID` que s'il appartenait au `.pck` de l'arme : ces sons etaient donc rejetes en
silence. Mesure sur le MA40 : **359 sons dans le pack, 398 embarques dans la bank**, et deux
des quatre couches du tir ne resolvaient AUCUN son avant correctif. Sur les 53 armes :
**4642 sons dans les packs contre 5271 embarques** — plus de la moitie du contenu etait
invisible pour toute la chaine, extraction et tri compris.

FAUSSE PISTE ECARTEE EN CHEMIN, consignee parce qu'elle a coute un cycle : on a d'abord cru
que les couches vides venaient de sons PARTAGES entre armes. Un index large de tous les
packs a ete construit (15 798 identifiants, chiffre verifie independamment) — il n'a rien
change. C'est `DIDX` qui expliquait tout. L'index large est conserve : il ne nuit pas et
couvre le cas des sons partages s'il se presente.

Consequence sur le livrable : l'objet a juger n'est plus le `.wem` mais le COUP RECONSTITUE
(une variante tiree par couche, sommee). Rendu pour les 33 armes, avec sa rafale a la cadence
du tag.

NOMMAGE — DEUX POINTS A TRANCHER AVEC L'UTILISATEUR :

- `hinf_vestige_carbine` (« Carabine Vestige ») : le `weap 3e070217` EXISTE et son champ
  « Weapon Fire Sound » mene au `snd! 6eced10d`, qui depend de la **`sbnk 09089e7e`**. Ce
  qui manque est le PACK AUDIO : aucun des 55 `.pck` traites ne correspond a cette bank.
  Prochaine etape : extraire les medias embarques de `09089e7e` (elle en porte peut-etre
  la totalite) ou chercher son pack sous un autre prefixe que `sb_010_wea_`.
  Note : sur les 26 armes du registre hors grenades, seules 3 restent non rattachees —
  l'epee et le marteau (normal : pas de tir) et celle-ci.
- `bt_enforcer` = le **Mutilator**, RESOLU. L'utilisateur l'avait suppose ; verifie en
  remontant le graphe : `sbnk ff09acbd` <- 5 `snd!` <- `stai` <- `snd!` <- `weap d7915565`,
  et `jeu/index.json` designe ce tag comme `nom_jeu = mutilator` (icone `contour-37.png`).
  L'en-tete de `weapon-icons-build/weaptags.go` le documentait deja.
  POURQUOI LA CHAINE AUTOMATIQUE L'A MANQUE : `lot-tir` suit `weap -> lsnd`, or cette arme
  passe par des `snd!` avec un relais `stai`, soit quatre niveaux au lieu de deux. La
  remontee generique (`remonter`) la trouve ; la passe en lot, non. A generaliser.
  LACUNE DU REGISTRE, hors perimetre : `weapon_names.toml` n'a AUCUNE entree pour cette
  arme, et `index.json` lui met `arme = None` — d'ou l'absence de nom francais.
- CORRECTION (meme jour). J'avais qualifie `jeu/index.json` d'incoherent parce que l'entree
  `hinf_cindershot` porte `nom_jeu = heatwave`. **C'est FAUX et l'accusation est retiree.**
  L'utilisateur a indique que le nom interne « heatwave » a designe une arme en debut de
  developpement et une autre en fin ; verification faite, le fichier est FIDELE :

		pack audio `fr_heatwave` -> weap 230447b1 -> hinf_cindershot (Cremateur)
		pack audio `fr_hotrod`   -> weap 2ac9c2ff -> hinf_heatwave   (Calcineur)

  Le croisement est reel dans les donnees du jeu : le pack audio a garde le sens du debut,
  le tag porte celui de la fin. Le pipeline n'a pas ete trompe parce qu'il joint par TAG
  `weap` et jamais par nom — c'est exactement le cas que cette regle protege. A retenir :
  ne jamais deduire l'identite d'une arme du nom de son pack audio.
  Au passage : `cv_provoker` = `hinf_ravager` (Ravageur).

### 2026-08-15 — Etape 6 : relais suivis, et deux regressions attrapees en chemin

La passe en lot suivait `weap -> tag de son` sur UN niveau ; les armes a relais (`stai`)
lui echappaient. Generalisation par expansion transitive dans les classes sonores
(`snd!`, `lsnd`, `stai`), profondeur 4. Resultat : **34 armes au lieu de 33**, la nouvelle
etant `bt_enforcer` (le Mutilator) — exactement le cas qui avait ete resolu a la main.

DEUX REGRESSIONS INTRODUITES PAR CETTE GENERALISATION, mesurees puis corrigees :

1. L'expansion ECRASAIT l'association (`out[gid] = armes`, avec saut du deja-vu) : un tag
   atteint depuis deux armes revenait au premier arrive. Symptome : `bt_arczapper` rattache
   au S7 Sniper, et le Disrupteur disparu du rattachement. Corrige en ACCUMULANT les
   associations, plus rejet des tags atteints depuis plus de 3 armes (points de passage
   communs : les compter preterait a chaque arme les evenements des autres).
2. L'accumulation seule n'a PAS suffi — `bt_arczapper` restait faux. Cause reelle : les
   relais etaient traites AU MEME RANG que les liens directs, si bien qu'une chaine longue
   du sniper atteignait les evenements de l'arczapper. Corrige en faisant du relais un
   REPLI : le lien direct (le tag designe par le champ de tir) est la preuve forte, le
   suivi des relais ne sert QUE pour les armes qu'aucun lien direct ne couvre.

Controle apres correctif : Disrupteur revenu, Mutilator conserve, aucun pack ne partage
plus une cle produit avec un autre. Sur les 26 armes du registre hors grenades, seules 3
restent non rattachees : l'epee et le marteau (pas de tir, normal) et la Carabine Vestige.

VARIANTES, mesure :

	cv_provoker_megatron            weap 05b2c46c -> hinf_ravager (Ravageur)
	cv_provoker (base)              AUCUN son de tir prouve
	*_sentinelminiboss (x3), _berserk   tags distincts, ABSENTS de l'index d'icones

Lecture : les `_sentinelminiboss` et `_berserk` sont des variantes PNJ, sans entree produit
joueur — coherent. Pour `cv_provoker_megatron`, le tag est distinct mais pointe vers le
produit Ravageur de base, et c'est la variante qui porte le son de tir, pas le pack de base.
L'hypothese « variante legendaire » de l'utilisateur est COMPATIBLE avec la mesure mais
n'est pas prouvee par elle : rien dans le tag ne dit « legendaire ».

### 2026-08-15 — Etape 7 : melee, Carabine Vestige, et le nommage delie du tir

CARABINE VESTIGE, RESOLUE. Sa bank `09089e7e` porte **83 medias embarques** et AUCUN `.pck`
ne lui correspond : ses sons vivent entierement dans la bank. C'est pour cela qu'elle etait
absente de bout en bout. Le mode `embarques` accepte desormais `-sbnk` pour cibler une bank
par identifiant, sans passer par un pack.

NOMMAGE DELIE DU TIR. Une arme sans champ « Weapon Fire Sound » n'avait ni nom ni icone —
contrainte que je m'etais imposee sans raison. `rattachement.go` relie desormais une arme a
son `weap` par le seul lien `weap -> ... -> sbnk`. Gain automatique : `cv_provoker`, qui se
retrouve apparie a `cv_provoker_megatron` sur `hinf_ravager` — la variante portait le son de
tir, pas le pack de base.

MELEE : POURQUOI ON NE PASSE PAS PAR `jmad`. Mesure sur la bank de l'epee a energie :
`bank <- 21 snd!/lsnd <- 100 jmad <- 8 weap`. Les sons de melee remontent par les graphes
d'ANIMATION. Traverser `jmad` est exclu — c'est un carrefour (98 `Rani` au niveau suivant),
tout s'y rattacherait a tout. Le champ nomme « melee sound » du tag a ete implemente
(mode `melee`) et se lit sans derive sur 10 armes, MAIS il pointe vers une bank de melee
GENERIQUE (`cbce234a`), pas vers celle de l'arme : il dit « le bruit quand on frappe avec
cette arme », pas « les sons de cette arme ». Les sons propres restent dans la bank de
l'arme, et ils etaient DEJA extraits — seul le rattachement manquait. Il est donc pose a
partir des tags que `jeu/index.json` associe deja a ces armes.

GARDE-FOU DESSERRE : le plafond de `weap` candidats par bank passe de 6 a 24. A 6, l'epee
etait ecartee alors que sa bank est atteinte par 8 `weap` (3 sont bien l'epee, 2 un « skull »
de Forge, 3 inconnus). Trancher dans le Go reviendrait a deviner ; on rend tous les
candidats et la couche de nommage, qui dispose de l'index d'icones, retient celui qui resout
vers une vraie entree produit.

### 2026-08-15 — Etape 8 : AUDIT DU FORMAT, apres deux oublis de suite

L'utilisateur a releve, a juste titre, que deux specificites du format Wwise avaient ete
manquees coup sur coup : les medias embarques (`DIDX`/`DATA`) puis le tableau des modes de
tir. Le point commun n'est pas l'inattention, c'est la METHODE : le parseur n'implementait
que le strict necessaire a l'objectif du moment, et rendait un resultat plausible en
ignorant le reste — donc silencieusement faux.

Correctif de methode : nouveau mode `audit`, qui ENUMERE ce que les banks contiennent et
l'affiche en regard de ce que le parseur consomme. Un trou devient visible sans qu'il
faille le soupconner d'abord. Resultats sur les 1305 banks :

	CHUNKS      BKHD 1183 (lu) | HIRC 1180 (lu) | DIDX/DATA 578 (lus)
	            *** 989 banks portent des chunks au nom NON IMPRIMABLE — IGNORES ***
	            STID 2, PLAT 1, STMG 1, ENVS 1, INIT 1 — tous ignores
	OBJETS      Sound 62753 | Settings 55831 | Action 41464 | Event 36355 | ActorMixer 9037
	            RandomSequence 7069 | Switch 445 | Blend 303 | ... (proprietes jamais lues)
	ACTIONS     Play 38089 | Stop 1976 | Break 912 | SetLPF 122 | SetState 69 | Mute 40 ...

TROIS DEFAUTS ETABLIS PAR CET AUDIT :

1. **Le type d'Action n'etait jamais lu.** Le parseur prenait la CIBLE de toute action, y
   compris des `Stop` et des `Break` : la cible d'un `Stop` etait donc empilee comme une
   couche a jouer. CORRIGE — seules les actions de type `Play` (octet haut 0x04) alimentent
   desormais la hierarchie. C'est une cause directe de coups reconstitues moins justes
   qu'un `.wem` isole, ce que l'utilisateur avait signale a l'oreille.
2. **989 banks portent des chunks dont le nom n'est pas imprimable.** Le decoupeur en
   chunks derape quelque part. NON RESOLU, a instruire.
3. **La charge utile des objets `Sound` n'est lue qu'a hauteur de 13 octets** (pluginID,
   streamType, sourceID) alors qu'elle est bien plus longue. Volume, hauteur, filtrage,
   positionnement, effets et aleas par lecture sont ignores — donc le mixage additionne des
   couches a gain unitaire, la ou le moteur applique des gains distincts. NON RESOLU.

QUATRIEME DEFAUT, trouve par un agent : **`modes` comptait des TAGS DE SON, pas des
ELEMENTS du tableau**. Contre-epreuve : le pistolet a plasma porte un bloc de 192 octets
(2 elements, second « Variations » au champ +108 = 12 + 96) ; le fusil d'assaut un bloc de
96 octets, donc UN element, dont les « Variations » comptent 3 references. Le fusil
d'assaut a donc UN MODE A TROIS VARIANTES, et non trois modes. CORRIGE : un mode est
desormais un element de 96 octets, ses variantes sont les references de son sous-tableau.

### 2026-08-15 — Etape 8 bis : ce que les agents ont etabli

SPNKr A COMBUSTIBLE vs M41 SPNKr — le melange entendu par l'utilisateur est REEL et n'est
PAS un defaut de rattachement. La chaine `weap -> snd! -> sbnk -> event` est distincte et
correcte pour les deux armes (`9d6aaed2`/`hinf_fuel_rod_spnkr` contre `71ab0a2c`/
`hinf_m41_spnkr`, jamais croises). Mais **33 identifiants `.wem` sont communs aux deux
`.pck`, octet pour octet, soit 57 % de chaque pack**, et l'evenement `0x49b19764` du SPNKr
(noeud Blend, 15 wem) est partage a 100 % avec le fuel rod. Le jeu livre les memes fichiers
dans les deux packs : ce que l'utilisateur entend est authentique.

MUTILATOR — deux defauts cumules. (a) Ses deux modes pointent vers `sbnk 8827aa7e`, alors
que la passe 1 a apparie son `.pck` a `sbnk ff09acbd`. **`8827aa7e` n'est dans AUCUNE des
54 entrees de lot1 : ses `.wem` n'ont jamais ete moissonnes.** (b) `Modes` est construit
depuis le lien DIRECT seulement ; les armes qui entrent par le repli relais perdent leur
detail par mode. Trois armes concernees : `bt_enforcer` (2 tags -> 0 mode), `bt_voltaction`
(2 -> 1), `cv_fuelrod_hunter` (2 -> 1).

SHOCK RIFLE — toute la reconstitution repose sur un tag de son BOUCLE (`lsnd 7da2a96a`) ;
le mode `snd!` (`013968cb`) rend zero evenement car son relais `stai 3347c586` est vide
dans ce module. Les evenements retenus sont des conteneurs continus (`Blend`, `Switch`),
pas des coups discrets — coherent avec l'absence de cadence exploitable.

JUMEAUX 1P/3P — constat transverse : de nombreux evenements vont par PAIRES portant les
memes couches (perspective premiere et troisieme personne). La selection en retient un et
jette l'autre, arbitrairement. 15 des 19 evenements du Shock Rifle et 9 des 14 du Mutilator
restent non attribues.

CARABINE VESTIGE — son tag de tir designe **DEUX** evenements (`0cfa31bf` et `f5050f4e`,
ensembles disjoints, union 52 `.wem`), la ou le rendu n'en prenait qu'un. Un seul mode,
cadence reelle **285 coups/min**. Le rapport `lot2` contenait deja les deux evenements :
la perte est en aval, au moment de choisir UN evenement pour le rendu.

ERREUR DE CONVERSION DE MA PART, relevee par un agent : j'avais donne `0xd7915565` =
3616724325, ce qui vaut en realite `0xd792d565`. La bonne valeur est **3616626021**.

### 2026-08-15 — Etape 8 ter : deux defauts de plus, dont un structurel

CINQUIEME DEFAUT — **le critere de choix de l'evenement est DEGENERE**. Le rendu retient
`max(nombre de couches, nombre de wem)`. Sur l'epee a energie, les 34 evenements ont TOUS
exactement une couche : la cle se reduit au nombre de wem, et SIX evenements sont a egalite
a 5. `max()` rend alors le premier dans l'ordre du JSON. Le son presente a l'utilisateur
etait donc choisi par HASARD, pas par preuve — et `_COUP.wav` de l'epee se reduit a un seul
`.wem` mono de 0,83 s suivi de silence. Le doute de l'utilisateur etait fonde.

SIXIEME DEFAUT, et il porte sur TOUTES les armes — **la partition 1P/3P n'est pas exploitee**.
Mesure sur l'epee : 16 evenements 100 % mono, 18 evenements 100 % stereo, apparies par
durees quasi identiques. C'est la signature TROISIEME PERSONNE (mono, positionne en 3D) et
PREMIERE PERSONNE (stereo, joueur). L'epee n'a donc pas 34 sons mais ~18, chacun en deux
versions. Le meme phenomene explique les « jumeaux » releves independamment sur le Shock
Rifle (15 evenements sur 19 non attribues) et le Mutilator (9 sur 14). La selection en
retient un au hasard : elle presente donc souvent la version 3P alors que l'utilisateur
compare a ce qu'il entend EN JEU, c'est-a-dire la 1P.

REGLE A APPLIQUER : a famille egale, preferer l'evenement STEREO (1P). Sur l'epee, le bon
candidat est `110deea3` (stereo, 5 variantes, 0,83-1,14 s, -0,3 dBFS) et non `a4cdc09a`
(son jumeau mono, retenu par hasard).

L'epee illustre aussi ce que le mixage devrait faire : superposer le balayage `110deea3`,
la trainee grave `68be8dae` a -10 dB (53 % de son energie sous 200 Hz, seul evenement 1P
sans jumeau mono — donc une COUCHE de renfort, pas un son complet), l'impact `bc2b3e76`
decale d'environ 0,25 s, et par-dessus les deux boucles continues `a63d1691` (9,7 s, grave)
et `9d9ca315` (12,9 s, gresillement, -25 dB).

CONTROLE DE COMPLETUDE sur l'epee, rassurant : 34 evenements -> 94 `.wem` distincts, et
`wems_embarques` = 94. Zero orphelin dans un sens comme dans l'autre. Les 10 sons presents
uniquement en embarque appartiennent a 4 evenements — vraisemblablement non streames pour
partir sans latence.

RESERVE : cette comptabilite prouve qu'aucun SON n'echappe, pas qu'aucun EVENEMENT
n'echappe. Un `Stop`, un changement d'etat ou un evenement de switch pur ne reference aucun
wem et reste invisible dans `lot1.json`.

A NOTER pour la rafale : l'epee n'a pas de cadence, le script retombe donc sur son defaut de
400 coups/min et empile 10 coups en 2,84 s. Une arme de melee ne doit PAS avoir de rafale.

### 2026-08-15 — Etape 9 : gains appliques, perspectives separees, banks orphelines

TROIS CORRECTIFS, dans l'ordre convenu avec l'utilisateur.

**1. Les gains.** Nouveau lecteur `proprietes.go` : layout `AkPropBundle` decode depuis
l'offset 14 de la charge utile d'un `Sound`, apres `NodeInitialFxParams`. Validation par
plausibilite (nombre de proprietes borne, identifiants connus, valeurs finies dans des
plages realistes) — un layout qui aurait derive echoue au controle au lieu de rendre des
gains fantaisistes. **62 572 objets sur 62 753 se decodent, soit 100 %.** Mesures :
5 312 sons portent un volume non nul, jusqu'a **-96 dB** ; **aucun delai** (le `t=0` du
mixage etait donc correct) ; 60 sons a hauteur modifiee. Le gain est desormais applique au
mixage — additionner a gain unitaire faisait arriver au premier plan des couches de renfort
censees rester en arriere.

**2. Les perspectives.** Les evenements vont par paires mono/stereo : troisieme et premiere
personne. L'ancien critere `max(couches, wem)` en choisissait une au hasard — et etait
carrement degenere sur l'epee (34 evenements a une couche, six a egalite). On rend desormais
UN COUP PAR (MODE, PERSPECTIVE), etiquete. DECISION PRODUIT de l'utilisateur : pour le rejeu
2D, la camera n'est pas a la premiere personne, donc **la 3P est la perspective pertinente**
— la rafale la suit par defaut.

DEUX VALIDATIONS INDEPENDANTES de ce correctif, obtenues sans les viser :
- pour l'epee, la 1P retenue est `110deea3`, exactement l'evenement qu'un agent avait
  recommande par analyse acoustique separee ;
- pour le Ravageur, le mode 2 en 1P est `be684013`, exactement l'evenement que
  l'utilisateur avait vote a l'oreille.

**3. Les banks orphelines.** Une arme peut avoir PLUSIEURS banks, et celles dont tous les
sons sont embarques n'ont aucun `.pck` : l'appariement par intersection ne peut pas les
trouver. Nouveau drapeau `-banks` pour les demander explicitement. Recolte : `8827aa7e`
(16 evenements, 129 sons — la bank de tir du Mutilator, jamais moissonnee) et `09089e7e`
(8 evenements, 83 sons — la Carabine Vestige, qui n'avait plus besoin d'etre greffee a la
main). Total : **10 754 `.wav` embarques**, 55 armes en passe 1, 37 avec sons de tir.

RESTE OUVERT : les 989 banks a chunks non imprimables (decoupeur qui derape) et
l'exploitation du paquet RANGED (aleas de volume et de hauteur par lecture, qui expliquent
qu'un meme son ne sonne jamais deux fois pareil en jeu).

### 2026-08-15 — Etape 10 : le garde-fou de duree, et ce qu'il a fait remonter

Objet : defaut 5 du handoff (appariement de perspective sans garde-fou), signale a
l'oreille sur le Needler. Trois resultats, dont deux non prevus.

**1. Le seuil de 3 annonce au handoff etait FAUX, et destructeur.** Pose tel quel, il
refusait 26 appariements sur 12 armes. Or l'utilisateur avait deja valide a l'oreille des
appariements bien au-dela : skewer 0,79 s -> 4,99 s (rapport 6,3), spiker 6,2, sniper 5,9,
disrupteur 4,5, fusil a pompe 4,3. En 1re personne on entend la mecanique et la queue : la
duree DOIT etre plus grande. Le seuil est donc CALIBRE SUR LES 44 VOTES et porte a **10** :
tous les appariements valides survivent, et le Needler (rapport 26) est le SEUL refus sur
les armes votees. Lecon : un seuil annonce dans un handoff sans jeu de validation est une
hypothese, pas une mesure — ici c'est le vote de l'utilisateur qui a servi de jeu d'essai.

**2. La mesure acoustique montee pour trancher etait DEGENEREE, et son temoin l'a dit.**
Question : la couche longue et stereo est-elle le tir en 1re personne, ou autre chose ?
Instrument : energie dans 8 bandes (Goertzel) sur les 120 ms d'attaque, similarite cosinus.
Resultat : 0,994 a 1,000 entre les deux perspectives d'une meme arme... mais AUSSI 0,991 a
1,000 entre deux armes DIFFERENTES. L'instrument ne separe rien ; aucune conclusion n'en a
ete tiree. Le temoin « armes differentes » n'etait pas decoratif, c'est lui qui a evite de
publier un faux resultat. **Toute mesure de similarite doit embarquer son temoin negatif.**

**3. DECOUVERTE — les elements de « Weapon Fire Sounds » ne sont pas tous des modes.**
Constat de l'utilisateur (« pour le MA40 et le MA5K je n'ai pu me decider ») explique par
la mesure : les trois elements du MA40 partagent le MEME evenement de 1re personne
(`1046dc38`) et ne different que par la 3e. L'artefact proposait donc trois fois le meme
fichier a juger. Idem MA5K (`f622d8f9`) et magnum (`26ef9698`).

	MA40   element 1 -> 3p 47044cbf | 1p 1046dc38
	       element 2 -> 3p 70173c35 | 1p 1046dc38
	       element 3 -> 3p ca9f5ec8 | 1p 1046dc38

CRITERE QUI EN DECOULE : **un vrai mode de tir a son propre son de 1re personne** ; un
element qui partage celui d'un autre est une variante de 3e personne (distance). Le critere
se valide seul sur deux cas etablis a l'oreille : le pistolet plasma rend 2 modes (le tir
charge, element 3, a bien son propre 1P) et le Mutilator `8827aa7e` en rend 2. Applique
avant le rendu, il fait passer le MA40 de 6 groupes a 4 sans orpheliner aucun vote (le plus
petit numero de mode est conserve) — verifie : 44/44 votes toujours rattaches.

Livre aussi : une rafale PRE-RENDUE par groupe (une variante par couche a chaque coup,
gains Wwise appliques) ; les evenements refuses restent ecoutables sous un nom qui dit leur
statut, avec leur rafale, pour qu'un refus calcule puisse etre dementi a l'oreille ;
l'artefact marque « choisi » ce qui est deja tranche et se reamorce sur le dernier export
si le stockage du navigateur est vide.

RESTE : l'epee infectee choisit comme reference de 3e personne un evenement de 25,71 s de
mediane (`dabf5bc3`, retenu parce qu'il a 2 couches quand les autres en ont 1). Ses 10
refus sont donc mesures contre une reference absurde. C'est le critere de choix degenere
`max(couches, wem)`, pas le garde-fou — a traiter avec lui.

### 2026-08-15 — Etape 11 : instruire les votes sur des sons isoles, et tomber sur `Switch`

Question posee par l'utilisateur : pourquoi a-t-il vote, pour 12 groupes, un son ISOLE
plutot que le coup reconstitue de la meme arme ?

D'ABORD UNE CORRECTION DE MON PROPRE CHIFFRE. « 12 votes sur un isole plutot que sur le
coup » etait imprecis : **7 des 12 accompagnent un vote sur le coup** de la meme arme, donc
l'isole y est un complement. Seuls **5 vont a l'isole SEUL** — les quatre armes de melee et
le Needler.

METHODE. Pour chaque vote sur un isole : ou vit ce `.wem` dans la structure, et QUI DOMINE
le coup correspondant. Poids d'une couche = RMS moyen de ses candidats x facteur de gain
Wwise — c'est ce qui arrive au premier plan, pas le nombre de couches.

DEUX CAUSES, toutes deux structurelles.

**1. Les conteneurs `Switch` sont traites comme des conteneurs aleatoires.** Les couches
portent un type de noeud, et `arbre.go:27` n'en lit que le NOM. Un `RandomSequence` tire
une variante au hasard, ses candidats sont interchangeables ; un `Switch` choisit selon un
ETAT DE JEU et les siens ne le sont pas. Le parseur resout les enfants d'un `Switch` par
l'heuristique generique : tous les etats finissent dans un seul lot ou le rendu pioche.

	Types de noeud des couches des coups : RandomSequence 36 %, Blend 26 %,
	Sound 22 %, Switch 16 % (mediane 30 candidats, max 40)

	31 coups sur 107 portent une couche Switch, dont 28 en 1re personne
	UNSC_sniperrifle  M1 1p  c0 Switch 30 wem  -> 71 % du melange
	UNSC_shotgun      M1 1p  c1 Switch 30 wem  -> 38 %
	Covenant_needler  M1 1p  c2 Switch 40 wem  -> la supercombinaison

Ce defaut explique EN CASCADE trois choses deja rencontrees : les evenements 1P « trop
longs » (le garde-fou de l'etape 10 n'en soigne que le symptome), le motif « 18 wem en 3P
contre 30 en 1P » observe sur cinq armes sans lien, et la preference pour l'isole.
CONSEQUENCE IMMEDIATE : 18 des 44 votes portent sur un `_coup_m*_1p` d'une arme concernee.
Ces rendus changeront — **ne pas relancer de campagne de vote sur la 1re personne avant.**

**2. Un son unique partage par 20 armes de 4 factions entre dans les coups a -2 dB.** Le
`.wem` `195277626` (0,92 s) forme a lui seul une couche `Sound` dans 21 coups de 20 armes,
Banished, Covenant, UNSC et Forerunner confondus, ou il pese 11 a 36 % du melange. Un son
unique partage par des armes sans rapport n'est le tir d'aucune. Deux autres du meme genre :
`87187708` (0,03 s, 5 armes), `5270936` (0,06 s, 4 armes, -8 dB).

Pour les quatre armes de melee, la cause est autre et deja identifiee : le critere de choix
degenere (defaut 8) leur donne des couches de 9,70 s et 25,71 s.

LECON. La question « pourquoi ca ne sonne pas juste ? » avait une reponse dans le FORMAT,
pas dans le gout. Elle n'a ete trouvee qu'en partant des votes de l'utilisateur et en
remontant jusqu'a la structure — c'est la troisieme fois que son oreille precede l'outil.

### 2026-08-15 — Etape 13 : la piste du champ nomme de melee, RESULTAT NEGATIF

Question : puisque le critere `max(couches, wem)` ne departage rien pour les armes de melee,
un champ NOMME du tag designe-t-il le coup, comme « Weapon Fire Sound » le fait pour le tir ?

`weap.xml` repond oui sur le papier. Le tableau « melee damage parameters » porte, par
element, treize references de tag dont deux visent exactement ce qu'on cherche :

	_40 "melee damage parameters"
	    _41 "melee attack effect"        <- le coup porte
	    _41 "biped melee hit effect"     <- l'impact sur un biped

LECTURE DU TABLEAU (mode `meleefx`). Deux corrections ont ete necessaires, et toutes deux
ont ete trouvees en AFFICHANT la structure, pas en essayant un offset de plus :

1. La derive du plugin sur ce champ est de **+56**, pas +64 (annonce 2456, reel 2512).
2. L'element fait **396 octets** la ou le plugin en annonce 392 : la suite des 13 references
   de 28 octets commence a +32 et non a +28 — `32 + 13 x 28 = 396`, le compte tombe juste.
   L'en-tete d'element a grossi de 4 octets entre le plugin et le build. Les references sont
   donc adressees par leur RANG depuis la fin du bloc, ou les deux s'accordent, et plus par
   l'offset annonce.

RESULTAT, sur les 47 tags `weap` dont le tableau est lisible :

	melee attack effect  -> un tag `effe`  :  2 elements
	melee attack effect  -> vide           : 45 elements
	biped melee hit effect                 :  0 element, TOUJOURS vide

Les deux seuls porteurs sont `00007ee6` (marteau antigravite) et `841ac5e5`, et ils pointent
vers **le meme** effet `249b3cc8`. Cet `effe` a **0 dependance** : il ne mene a aucun tag de
son. L'epee a energie (`0000ae3c`) n'a ni l'un ni l'autre.

VERDICT : **le tag ne designe PAS le coup de melee.** Le champ existe, il est desormais
lisible, et le jeu ne le remplit pas. Contrairement au tir, il n'y a donc pas de designation
a exploiter — et la distinction coup touche / coup manque ne viendra pas de la non plus
(defaut 4 : elle reste a chercher du cote de `sound material effects`).

CE QUE CA CHANGE POUR LE DEFAUT 8. Le critere degenere ne peut pas etre remplace par une
designation ; il faut trancher autrement pour les cinq armes concernees (epee, epee
infectee, marteau, marteau legendaire, Needler). Deux voies, a arbitrer avec l'utilisateur :

- **Ses votes font foi.** Il a deja vote un evenement precis pour chacune (epee `44760a76`,
  epee infectee `a3935076`, marteau `d0e09f85`, marteau legendaire `f5c854f4`, Needler
  `7474d8d8`). Les epingler avec leur provenance, comme les rattachements manuels du
  generateur de manifeste. C'est de la donnee, pas une heuristique.
- **Un critere defendable pour le cas general**, a defaut de designation : preferer
  l'evenement dont la duree mediane est la plus proche de celle de l'autre perspective
  (coherence mutuelle), et refuser les durees invraisemblables — ce qui aurait ecarte
  d'office la reference de 25,71 s de l'epee infectee. A minima, un depart deterministe
  au lieu de « le premier du JSON ».

LECON DE METHODE, la meme que pour le `Blend` : deux hypotheses d'offset ont echoue avant
qu'un affichage de la structure ne tranche en une lecture. Le mode `meleefx` garde donc son
diagnostic — quand un appariement echoue, il montre ce que le plugin ANNONCE en regard de ce
que le tag CONTIENT, au lieu de laisser essayer un offset de plus.

## Decouvertes (hors perimetre — ne pas traiter ici)

- `cmd/weapon-icons-build/hmod.go` duplique volontairement `internal/himodule` (u32 vs
  48 bits). Le commentaire qui justifie cette copie est PERIME : himodule lit bien 48 bits
  + drapeaux (`module.go:265`). `cmd/weapon-sounds` utilise himodule directement, donc la
  copie de `hmod.go` n'a plus de raison d'etre — a supprimer dans un chantier dedie.
- `cmd/weapon-sounds/weapfire.go` est la 2e copie de la lecture de plugin + tables de tag
  (1re : `weapon-icons-build/weapui.go`), et `weap.xml` est donc present DEUX FOIS au depot
  (84 Ko chacun). La regle du depot tolere 2 copies, mais un fichier de DONNEES duplique
  derive silencieusement a chaque mise a jour du jeu. Promouvoir plugin + tables + `weap.xml`
  en `internal/hiweap`, et migrer les deux commandes.
- `himodule.Open` fait un `os.ReadFile` du module entier (7,24 Go pour `pc/globals`). Tient
  ici, mais un `mmap`/`ReadAt` supprimerait la contrainte.
