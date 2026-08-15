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

- `hinf_vestige_carbine` (« Carabine Vestige ») existe au registre avec le tag `3e070217`,
  mais AUCUNE des 33 armes resolues ne porte ce tag. L'arme n'est pas rattachee.
- `bt_enforcer` n'a pas de son de tir prouve, donc pas de nom. Ce n'est PAS le Déchiqueteur :
  `hinf_mangler` porte le tag `80977ba5`, deja attribue a `bt_spikerevolver`.
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
