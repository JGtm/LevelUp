# HANDOFF — SONS DU JEU : RECONSTITUTION, NOMMAGE, BRANCHEMENT

> Ecrit le 2026-08-27. Fait suite a la RE du 2026-08-26
> (`RE_BANQUES_SONORES_NOMMEES_2026-08-26.md`) et aux deux commits `3f9ffeb65` et `6e167535e`
> sur `feat/v75`. Ce document remplace toute reprise a froid : il porte ce qui est ETABLI, ce
> qui est REFUTE (pour ne pas le refaire), les hypotheses VIVANTES et les sondes ecrites.

## 0. PRIORITES, telles que l'utilisateur les a posees

1. **HAUTE — RETROUVER LES SONS RECONSTITUES, PAS ISOLES.** Poursuivre l'exploration
   (Ghidra + tags) jusqu'a ce que l'inventaire de sons soit SOLIDE. Cibles nommees : les sons
   manquants de **Bastion / Controle total / Roi de la colline**, et le **TRANSLOCATEUR
   QUANTIQUE**, que l'utilisateur cherche depuis plusieurs passes.
2. **HAUTE — les sons DEJA trouves doivent recevoir leur RECONSTITUTION et leur
   NOM / DESCRIPTION.** Un identifiant d'evenement n'est pas une livraison.
3. **NORMALE — brancher le son de RAMASSAGE SUR SOCLE** (`play_007_abl_shared_pickup`,
   confirme a l'oreille : « c'est bien pour le ramassage des armes sur socles de power up »).
4. **EN ATTENTE — le branchement des sons ISOLES deja identifies.** Ils attendent « un truc
   propre », c'est-a-dire que la reconstitution du point 1 soit faite.

## 1. LE PROBLEME CENTRAL : « reconstitue » vs « isole »

**Ce que la recette des armes produisait** (et que l'utilisateur cite comme reference) : UN
fichier par GESTE. Les couches simultanees sommees a leurs gains de chemin, une variante tiree
par couche, la fourchette RANGED exportee a part. Un coup de fusil = un son complet.

**Ce que la planche d'exploration produit aujourd'hui** : une carte par couple
(evenement x variante). C'est un INVENTAIRE de fragments, pas un catalogue de gestes. Deux
symptomes mesures le disent :

- **Le translocateur.** L'utilisateur decrit le geste : « c'est comme si on le chargeait, ca
  monte en intensite, et ensuite il est pose ». Aucun des 23 evenements de
  `sb_007_abl_quantum` ne reproduit ca ; ils durent de 0,4 a 6,8 s pris isolement.
- **`71cb04b8`** (avant apparition d'une nouvelle zone, KOTH) : « il y a un tres court son au
  debut qui me parait EN TROP ». L'evenement declare deux couches simultanees
  (`643735766` a -1 dB + `115611145` a +1 dB) sommees a t = 0. Soit la somme est fausse, soit
  les deux couches ne se jouent pas ensemble dans le jeu.

**La question a trancher est donc UNE seule** : quelle est l'unite de GESTE dans ce format, et
comment le moteur l'assemble ? Les sections 4 et 5 listent les hypotheses restantes et les
sondes qui les tranchent.

## 2. CE QUI EST ETABLI — ne pas le refaire

### 2.1 La methode de nommage

L'identifiant Wwise d'une banque (chunk `BKHD`) est le **FNV-1 32 bits de son nom de fichier
en minuscules**. Calibration : **647 banques** portent l'identifiant d'un `.pck` du disque, sur
les 1 697 des dix modules balayes. La convention est donc verifiee avant qu'un nom ne soit
casse. Seuil de publication : esperance de collision `candidats x cibles / 2^32` **< 0,10**,
imprimee AVANT chaque resultat.

### 2.2 La grammaire des noms, MESUREE

	banque      sb_<NNN>_<famille>[_<portee>]_<jetons>
	evenement   play_<nom de banque prive de « sb_ »>_<nom>_<verbe>[_<modulation>]

	  play_004_mod_mp_ctf _flag _scored _team
	  play_007_abl_repairfield    _deploy _player

17 temoins sur 17 commencent par `play_` et prennent pour base le nom de banque sans `sb_`.
Modulations observees : `_team`, `_enemy`, `_player`, `_generic`, `_loop`, `_neutral`.

### 2.3 Les banques nommees (extrait utile)

	sb_004_mod_mp_ctf          61007dcf  globals  52 wem  46 evts
	sb_004_mod_mp_assault      2b01f208  globals  38      42
	sb_004_mod_mp_extraction   156c35d5  common   33      23
	sb_004_mod_mp_landgrab     8369d532  common   39      23
	sb_004_mod_mp_shared_ui    ee694c9e  globals  33      20
	sb_004_mod_mp_shared_global 7c954e9f common   14       9
	sb_007_abl_shroud          92c830f5  globals  38      11   ECRAN OCCULTANT
	sb_007_abl_quantum         dcfaa487  globals  70      23   TRANSLOCATEUR (l'appareil)
	(non nommee)               b29ac6de  globals   8       2   TRANSLOCATEUR (la balise)
	(non nommee)               de65048f  globals   8       7   objets poses, generique
	sb_007_abl_shared          15c5b355  globals  20      15
	sb_007_abl_repairfield     5724312f  globals  31      12
	sb_007_abl_grapplinghook   385461e8  globals  69      29
	sb_007_abl_knockback       7bd0883c  globals  33      11   repulseur
	(non nommee)               1c609526  common  84      88   ZONES (Bastion/KOTH/Controle total)
	(non nommee)               7acb11cc  globals  32      16   capteur + traqueur de menaces

**46 evenements nommes** a esperance cumulee 0,0408 — detail dans
`RE_BANQUES_SONORES_NOMMEES_2026-08-26.md` §3.

### 2.4 La distinction par equipe : OUI, mesuree

Modulation terminale `_team` / `_enemy`, sur **six couples**, avec des `.wem` DIFFERENTS —
pas un meme son passe dans un interrupteur : `ctf_flag_scored`, `ctf_flag_taken`,
`ctf_flag_pickup`, `assault_bomb_taken`, `assault_bomb_pickup`,
`extraction_point_scored`. Sans variante d'equipe : `ctf_flag_returned`, les apparitions,
`assault_bomb_detonated`, les boucles d'armement et de desamorcage.

### 2.5 Les bobines (son du kill)

Un seul objet, cinq variantes d'ENERGIE, chacune sa banque NOMMEE sur le disque :
`sb_008_exp_single_small_{hardlight, kineticunsc, plasma, shock}`. Chaine deja versionnee dans
`damagetag/data/labels.tsv`, verdict d'index clos en `RE_LOG_KILLWEAPON.md` 7ter.57. Chaque
banque porte 3 evenements dont UN SEUL a **5 couches simultanees** : c'est l'explosion
complete, les deux autres sont des perspectives.

### 2.6 Le mode de lecture des conteneurs (5e oubli de format, corrige)

Le type 5 de Wwise (`CAkRanSeqCntr`) couvre DEUX comportements et le champ `eMode` dit lequel
(0 = aleatoire, les enfants sont des VARIANTES ; 1 = sequence, les enfants sont des PHASES
jouees dans l'ordre). Il n'etait **jamais lu**. Lecteur : `conteneurs_mode.go`, mode
`audit-modes`. Mesure sur `pc/globals` : **7 069 conteneurs, plausibilite de lecture 98,9 %,
96,61 % ALEATOIRES, 237 SEQUENCES (3,39 %) dont 196 CONTINUES**.

## 3. LES NEGATIFS MESURES — ne pas les refaire, chacun a son denominateur

1. **Les identifiants d'evenement ne sont PAS dans le binaire.** Recherche d'octets sur trois
   temoins (`c3327c0b`, `d8a2fcb8`, `59223651`) : **0 occurrence**. Le moteur les lit dans les
   TAGS. Ghidra ne peut donc pas fournir un identifiant, et ce n'est pas un defaut du jeu :
   c'est l'architecture normale.
2. **Le binaire ne porte que TROIS noms d'evenement Wwise en clair** sur les ~6 800 du jeu :
   `Play_002_UI_Menu_Global_TutorialPopup_Open`, `..._Close`,
   `play_002_ui_menu_forge_grabobject`. Ils confirment la grammaire, ils ne nomment rien
   d'autre.
3. **Le hachage est epuise sur les trois reperes de zone.** 162 831 744 candidats (36 noms de
   mode x 4 familles x les 141 347 identifiants du binaire x 8 modulations, esperance 0,1516)
   et seul le TEMOIN `c3327c0b` = `play_004_mod_mp_strongholds_contested` en ressort — que
   l'utilisateur identifie independamment comme « base contestee ». **Cette concordance valide
   la methode ; elle ne la sauve pas sur les autres.**
4. **`sdzg 00037692` est un REGISTRE GLOBAL, pas la table ordonnee d'un mode.** 622 `sgrp`, et
   les trois groupes identifies a l'oreille y sont aux rangs **205, 297, 516** — non contigus.
   Le rang ne porte donc pas la semantique cherchee (contrairement au precedent `gggl`).
5. **`sgrp` est 1:1 avec `snd!`** (3 mesures : `141dfb97`, `725dc61c`, `fc862992`, une seule
   dependance chacun). Le groupe sonore n'est pas l'unite de GESTE.
6. **Aucun conteneur SEQUENCE dans les banques d'equipement et de mode** : 235 conteneurs sur
   les 9 banques ciblees, **100 % aleatoires**. La « montee en charge » du translocateur n'est
   donc pas un conteneur en sequence dans sa banque.
7. **La banque propre a la BALISE du translocateur (`b29ac6de`) porte 8 `.wem`, tous entre
   0,41 et 0,48 s.** Elle ne peut pas porter la montee en charge decrite. Le geste est du cote
   de l'APPAREIL (`dcfaa487`, sons de 0,4 a 6,8 s).

## 4. LES HYPOTHESES VIVANTES pour la reconstitution, par cout croissant

**H1 — LES BOUCLES (`lsnd`).** La banque des zones est referencee par **20 `lsnd`** en plus de
ses 60 `snd!`. Une « capture en cours » et une « montee en charge » sont des BOUCLES par
nature. Le rendu actuel ne boucle pas : il joue le fichier une fois. Une boucle rendue une fois
sonne comme un fragment — c'est exactement le symptome.

**H2 — UN EVENEMENT PORTE PLUSIEURS ACTIONS.** Le parseur ne garde que les actions « jouer »
et les somme a t = 0. Si un evenement enchaine `Play A` puis `Play B` avec un delai, et que le
delai est porte par l'ACTION et non par le noeud, notre somme ecrase la sequence. Le releve de
delais (`propDelai`, AkPropID 17) vaut **0 partout sur 275 couches** — mesure faite sur les
NOEUDS ; elle ne dit rien des ACTIONS.

**H3 — LES CONTENEURS EN SEQUENCE.** Defaut PROUVE, pas une hypothese : 237 conteneurs du jeu
(dont 196 continus) sont joues DANS L'ORDRE par le moteur et rendus comme des variantes par
nous. Aucun n'est dans les banques deja ciblees, mais le rendu est faux pour eux partout
ailleurs.

**H4 — LES COUCHES PILOTEES PAR RTPC (`Blend`).** Le rendu evalue un Blend « au point de
reference » (x minimal de chaque courbe), c'est-a-dire a intensite NULLE. Une montee en charge
pilotee par un RTPC serait aplatie a son etat de repos. La recette des armes le dit deja :
« RTPC de couche : statue HORS rendu, valeur neutre absente des banks ».

**H5 — LE GESTE EST PLUSIEURS EVENEMENTS.** Le jeu poste peut-etre 2-3 evenements pour un
geste (boucle de charge + one-shot de pose). Ce groupement ne vit ni dans `sgrp` (refute, §3.5)
ni dans `sdzg` (refute, §3.4). Il reste deux endroits ou il peut vivre : les **`hsc*`
(HaloScript)** et le **tag de MODE**.

## 5. LES SONDES ECRITES — a reprendre telles quelles

**S1 — IDENTIFIER LE `hsc*` DU CHEMIN.** La remontee depuis la banque des zones
(`remonter`, module `any/globals/common-rtx-new.module`, depart `1c609526` = 476091686) rend :
niveau 1 = 60 `snd!` + 20 `lsnd` ; niveau 2 = 58 `sgrp` + 3 `effe` + **1 `hsc*`** ; niveau 3 =
6 `hsc*` + 1 `sdzg` + 1 `weap`. **Un script reference donc directement un de ces 80 sons.**
Sonde : enumerer les tags `hsc*` du module et dumper leurs dependances
(`-mode deps-ordre`) jusqu'a trouver celui qui cite un `snd!`/`lsnd` de la banque. Puis
extraire ce tag et chercher s'il porte des chaines lisibles — un script CUIT peut garder ses
noms de fonctions la ou un tag de donnees ne les garde pas. C'est la piste que l'utilisateur
designe par « faut remonter dans le code et les fonctions », et c'est la seule encore ouverte
pour le NOMMAGE.

**S2 — COMPTER LES ACTIONS PAR EVENEMENT** (teste H2). Le mode `audit` recense deja les types
d'action ; ce qui manque est la DISTRIBUTION du nombre d'actions « jouer » par evenement, et,
pour les evenements a plusieurs actions, la lecture du delai PORTE PAR L'ACTION (et non par le
noeud). Si un seul evenement des banques ciblees porte deux actions avec un delai non nul,
H2 est confirmee et le rendu doit sequencer.

**S3 — LES `lsnd` DE LA BANQUE DES ZONES** (teste H1). Les 20 `lsnd` qui referencent
`1c609526` ne sont pas dans l'arbre d'evenements produit par `eqip-arbre` (qui part des
Events). Sonde : les enumerer (`qui` sur la banque, filtre `lsnd`), lire leur corps, et voir
s'ils designent des evenements que le balayage complet a deja recenses — ou d'autres. Une
boucle de capture y est le candidat naturel.

**S4 — LE TAG DE MODE** (teste H5). Trouver le tag qui definit le mode « zones » et lire ses
dependances : s'il cite plusieurs `sgrp`/`snd!` cote a cote, l'ordre de sa table est le
groupement cherche. Point de depart : la remontee de niveau 3 rend `1 weap` (`5bd53639`) et
`1 sdzg` — le `weap` est un faux ami, mais la remontee n'a pas ete poussee au-dela de 3
niveaux faute de budget.

**S5 — LES ORPHELINS DU TRANSLOCATEUR.** `dcfaa487` porte **2 `.wem` orphelins**
(`708804123` = 6,77 s et `532684898` = 6,22 s) qu'AUCUN de ses 23 evenements n'atteint. Ce
sont les DEUX PLUS LONGS sons de la banque, et l'utilisateur cherche un son long. Sonde :
chercher quel evenement (d'une AUTRE banque) les atteint — le balayage complet
(`arbre_all_globals.json`, 5 612 evenements) permet de le faire hors module, par recherche de
l'identifiant dans les `wems` de chaque evenement.

**S6 — RENDRE EN BOUCLE ET EN SEQUENCE.** Une fois H1/H2/H3 tranchees, le rendu doit apprendre
deux choses qu'il ne sait pas faire : enchainer (sequence continue) et boucler (N repetitions
pour l'ecoute). Sans ca, aucune reconstitution ne sera fidele.

## 6. CE QUI EST BRANCHE ET COMMITE (`3f9ffeb65`)

	drapeau capture / pris / ramasse    x2 camps   doc.objectives, stat `flag_*`
	drapeau rendu                       1 son      (le jeu n'a pas de variante d'equipe)
	capture de zone ALLIEE              1 son      stat `zone_captures` — PAIRE INCOMPLETE,
	                                               le cote adverse reste MUET
	champ de reparation : pose + FIN    3+3 var.   placements `t0` et `t1`
	grappin / repulseur                 3+3 var.   sons du JEU, a stem constant
	5e categorie `objective`            FR/EN      tiroir de reglages

Tirage d'une variante A LA LECTURE (`replaySoundVariants.ts`), jamais a la construction de la
piste.

## 7. LES DECLENCHEURS DISPONIBLES — pour le jour ou les sons seront propres

	ramassage sur socle    doc.padPickups     INTERVALLE [tLow, tHigh], PAS un instant.
	                                          MAIS le film porte l'inventaire au CHANGEMENT
	                                          (voir §8.1) : le datage est un travail de
	                                          DECODEUR, pas une impossibilite.
	capture en cours       ZoneState.Gauge    « les RAMPES, montees monotones de la jauge,
	                                          c'est-a-dire les CAPTURES EN COURS »
	nouvelle zone (KOTH)   ZoneSpan.Active    bascule quand la colline change
	base contestee         AUCUN               `ZoneSpan` ne publie pas d'etat « contested ».
	                                          A deriver ou a ajouter cote Go.
	pose de l'ecran occultant  placements     BLOQUE : la famille s'appelle encore `other` au
	                                          manifeste. Nommer `shroud_screen` dans
	                                          `replay_labels.toml` ET dans la liste fermee de
	                                          `loader_replay_labels_equipment.go` — touche
	                                          aussi le DESSIN du calque.
	bombe / extraction     AUCUN               `objectiveevents/named.go` ne mappe que DEUX
	                                          familles (`ObjectiveTypeFlag`, `ObjectiveTypeZone`).
	                                          Travail de DECODEUR (table de slots), pas de son.

## 8. LES DEUX ERREURS A NE PAS REFAIRE — les miennes, corrigees ici

**8.1 « Le film ne porte pas X » alors que la mesure ne dit que « notre lecteur ne trouve pas
X ».** J'ai ecrit que l'inventaire ne venait que des images-cles. C'est faux, et
`filmdec/testdata/ecs_table.tsv` le dit lui-meme, composant **i22** :
« Le comptage en **DELTA** donnerait les grenades **a la seconde** au lieu de toutes les 20 s.
**Bloque par la derive du curseur amont, pas par la grammaire.** » Idem i30/i31 pour les
munitions. Le journal RE interdit deja cette faute nommement (`RE_LOG_KILLWEAPON.md` 7ter.58).
La meme faute vaut pour la bombe et l'extraction (§7).

**8.2 Servir a l'ecoute sans mesurer d'abord.** Trois defauts de rendu, tous trouves par
l'utilisateur : (a) intermediaire en `pcm_s16le`, ou une couche a -96 dB (convention Wwise de
mute) tombe sous le pas de quantification — **11 rendus sur 135 silencieux**, corrige en
`pcm_f32le` ; (b) une seule variante rendue (`Wems[0]`) alors que les variantes d'un geste ne
durent pas la meme chose ; (c) orphelins jamais rendus. **Regle** : mesurer duree ET niveau de
chaque rendu AVANT publication, et verifier que le contenu peut repondre a la description
donnee (la banque de la balise ne pouvait pas porter un son long : 8 fichiers, tous < 0,5 s).

## 9. OUTILLAGE — tout est dans `cmd/weapon-sounds`

	banks-noms    nomme les banques d'un module par hachage FNV-1 du BKHD ; imprime
	              CALIBRATION, puis ESPERANCE, puis resultats, dans cet ordre impose
	audit-modes   lit `eMode` des conteneurs de type 5 (aleatoire vs sequence)
	deps-ordre    dependances d'un tag DANS L'ORDRE DU FICHIER (le mode `deps` les trie, ce
	              qui detruit la donnee — precedent `gggl`)
	remonter      remonte d'un tag vers ses referents, niveau par niveau, par groupe
	qui           les referents directs d'un tag, avec leurs identifiants
	sndscan       trouve les `snd!` qui portent un identifiant d'evenement donne
	eqip-arbre    la STRUCTURE des evenements d'une banque (`-banks all` = tout le module :
	              1 645 banques et 6 819 evenements en quelques secondes)

Decodage audio : `vgmstream-cli.exe` r2117 dans `C:\Users\Guillaume\Downloads\vgmstream\`
(ffmpeg ne decode PAS le Wwise Vorbis). Rendus et fiches :
`C:\Users\Guillaume\Downloads\Halo Infinite - Sons v75\`.

**MEMOIRE** : `pc/globals` fait 7,24 Go et `himodule.Open` lit tout en RAM. Jamais deux gros
modules dans le meme processus ; les modes s'echangent leurs resultats par JSON.

## 10. ARTEFACTS

	970a7eea-5d0d-4b69-afd0-bdafe0e0fb10   planche de VALIDATION, 39 sons — VALIDEE
	6aadf3d5-7acf-4b98-a050-6bc88db55b7d   planche de TRAVAIL, 226 rendus — la seule vivante

Regle demandee par l'utilisateur : **une seule adresse**, republiee en place. Ne pas creer
d'artefact supplementaire.
