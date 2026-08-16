# RECETTE — sons d'armes Halo Infinite, de l'extraction a la livraison

> Document de reference FINAL (2026-08-16). Le travail est livre une fois ; toute reprise
> (mise a jour du jeu, arme nouvelle, erreur decouverte) REJOUE cette recette, elle ne
> l'amende pas a la main. L'historique et les preuves sont au plan
> (`PLAN_EXTRACTION_SONS_ARMES.md`) et au handoff (`HANDOFF_SONS_ARMES_2026-08-15.md`) ;
> ce fichier ne contient que le COMMENT refaire.

## 0. Ou vivent les choses

	apps/go-api/cmd/weapon-sounds/                    l'outil Go (tous les modes)
	Desktop/Halo Infinite - Sons armes/               les .wav extraits + rendus + TRIER.html
	  _outils/                                        scripts du pipeline + vgmstream + tri.html
	  _donnees/                                       lot1.json, lot2.json, coups.json,
	                                                  manifeste.json, votes-final.json
	<jeu>/Sound/win/SFX/*.pck                         sources audio (90 170 .wem, sans noms)
	<jeu>/deploy/pc/globals/globals-rtx-new.module    1305 tags sbnk (banks Wwise) — 7,24 Go
	<jeu>/deploy/any/globals/globals-rtx-new.module   tags weap/snd!/lsnd/stai — 0,62 Go

CONTRAINTE MEMOIRE ABSOLUE : `himodule.Open` lit tout le module en RAM. Ne JAMAIS ouvrir
`pc/globals` (7,24 Go) en parallele d'un autre gros chargement. Les modes s'echangent
leurs resultats par JSON precisement pour ne jamais tenir les deux modules ensemble.

Prerequis de build : CGO (`internal/ooz`, decompression Kraken) — gcc msys64 ucrt64.

## 1. Extraire les sources audio

	# .pck -> .wem (un dossier par arme)
	python _outils/akpk_unpack.py           # cf. en-tete du script pour les arguments

	# .wem -> .wav (vgmstream ; ffmpeg NE decode PAS le Vorbis Wwise)
	python _outils/conv_lot.py              # sons des packs ET embarques, nommage duree_id

Nommage des fichiers produits : `DDD.DDs_<id>.wav` (duree en secondes puis identifiant
`.wem`) — les scripts d'assemblage lisent la duree dans le NOM, ne pas le changer.

## 2. Analyser les banks (passe 1 — module 7,24 Go, SEUL)

	cd apps/go-api
	go run ./cmd/weapon-sounds -mode lot \
	  -module "pc/globals/globals-rtx-new.module" \
	  -pck "<jeu>/Sound/win/SFX" \
	  -banks "8827aa7e,09089e7e" \
	  -json <...>/lot1.json

- `-banks` recolte les banks ORPHELINES (sans `.pck` correspondant) : `8827aa7e` =
  Mutilator, `09089e7e` = Carabine Vestige. Une arme entierement embarquee n'apparait
  QUE si on la nomme ici — l'oublier fait silencieusement disparaitre ces armes.
- Sortie : `lot1.json` — par arme, les evenements avec leurs COUCHES (points de choix),
  les `.wem` candidats et le GAIN DE CHEMIN de chacun.

APRES TOUTE EVOLUTION DU PARSEUR, relancer l'inventaire et verifier qu'aucune regle
n'est invalidee (c'est la lecon la plus chere du chantier — quatre oublis de format) :

	go run ./cmd/weapon-sounds -mode audit -module "pc/globals/globals-rtx-new.module"

## 3. Chaine arme -> tir (passe 2 — module 0,62 Go)

	go run ./cmd/weapon-sounds -mode lot-tir \
	  -module "any/globals/globals-rtx-new.module" \
	  -json <...>/lot1.json -out <...>/lot2.json

- Le tir est designe par le champ NOMME « Weapon Fire Sounds » du tag `weap` (le plugin
  `weap.xml` derive du build : offsets +56 a +64 selon le champ, geres par l'outil).
- La cadence vient de `barrels -> rounds per second`. ~30 coups/s = plafond moteur, pas
  une cadence.
- Identite en jeu : par tag `weap` -> `jeu/index.json` -> `weapon_names.toml`. JAMAIS par
  le nom interne du pack (`fr_heatwave` est le Cindershot, pas le Heatwave).

## 4. La semantique d'assemblage (NE PAS REINVENTER)

Reference : en-tete de `cmd/weapon-sounds/arbre.go`, chaque regle avec sa preuve.

	Sound           joue sa source
	RandomSequence  UNE variante, uniforme (6976/6976 tables de poids egales)
	Switch          l'etat par defaut (440/445 tables validees)
	Blend a courbes les enfants audibles au point de reference (la courbe tranche)
	Blend sans courbe / ActorMixer : TOUS les enfants
	Gain d'un wem   = SOMME du chemin evenement -> Sound (>10 000 conteneurs ont un volume)
	Delais          zero mesure partout ; empilement a t=0 conforme

Statue HORS rendu, avec raison : RTPC de couche (valeur neutre absente des banks),
paquet RANGED (exporte pour l'app, jamais cuit dans les .wav), delais d'action (offset
non valide — seul trou de preuve ; symptome possible : « pan... clic », piste Vestige).

## 5. Rendre les coups et les rafales

	python _outils/coups_lot.py             # lit lot1+lot2+votes, ecrit coups.json + .wav

- Un coup = une variante par couche, gains de chemin appliques, sommes, normalisation.
- Rafale = declenchements superposes a la cadence du tag (jamais d'attente de fin).
- UN VRAI MODE DE TIR A SON PROPRE SON DE 1re PERSONNE ; les autres elements du tableau
  sont des variantes de 3e personne (dedoublonnees).
- Garde-fou de perspective : rapport de durees <= 10 (calibre sur les votes).
- **LES VOTES PRIMENT SUR TOUT CRITERE** : un vote `ev_*` (garder/favori) dans le dernier
  export `votes-sons-armes*.json` de Downloads est une DESIGNATION que le rendu respecte,
  y compris pour ajouter un evenement hors chaine de tir (garde-fou : jamais si un mode
  le revendique deja).

## 6. Trier et voter

	python _outils/manifeste2.py            # manifeste (noms, icones, votes integres)
	SP=<scratchpad> python _outils/reinjecte.py   # injecte le manifeste dans tri.html
	# copier tri.html -> Desktop/.../TRIER.html, publier l'artefact

Le marquage « a revoter » compare l'EMPREINTE (evenement + couches + gains) a la
generation effectivement votee — comparer moins que ca rate les changements de mixage.

## 7. Livraison — FUSION avec le moteur sonore du rejeu (etat final, 2026-08-16)

Le rejeu 2D possede son moteur sonore (branche v75 : `replaySound.ts` regles pures,
`replayAudio.ts` lecture, `useReplaySound.ts` couture React — tirs, kills, grenades,
melee, equipements, filtres par categorie, plafond de voix). La livraison s'y FOND :

	python _outils/livraison.py <racine du depot>

- REMPLACE les fichiers d'ARMES (`hinf_*.wav`) de `static/sounds/halo_infinite/` par les
  sons extraits et votes, tronques a 1,2 s (discipline de poids du moteur, coupe
  d'enveloppe a ~1 s) — 26 armes, dont les 4 que le pack initial n'avait pas (Bandit,
  MA5K Avenger, SPNKr a combustible, Carabine Vestige). Les sons d'EVENEMENTS du pack
  utilisateur (lancers, explosions, melee, equipements) ne sont JAMAIS touches.
- GENERE `weaponSoundVariations.ts` : fourchettes RANGED par stem, tirees a chaque
  lecture par le moteur (reglage admin « variation », 100 % = le jeu).
- La DISTANCE est une chaine gain + passe-bas posee sur le maitre par reglage admin
  (`ReplayAudioPlayer.setDistance`) — a 0 %, aucun noeud ajoute, sons purs.

Jointure : `weaponLabels[id].key` (cle canonique du registre). Les armes HORS registre
(Mutilator, tourelles, PNJ) ne sont pas livrables — leurs sons votes restent dans
l'archive Desktop, prets si le registre les nomme un jour.

Roles confirmes livres : Ravageur = rendu du tir 3 coups `bb31841b` ; Sentinelle = wem
503433748. Regles produit fermes : sons livres PURS, variation et distance IN-APP
(reglages page admin), rejeu sans gestion de hauteur/distance/environnement.

## 8. Les lecons qui ont coute le plus cher (a relire avant toute reprise)

1. CARTOGRAPHIER LE FORMAT AVANT DE CODER. Quatre oublis successifs (DIDX, tableau des
   modes, type d'Action, conteneurs) ont chacun produit du plausible-et-faux. Le mode
   `audit` enumere ce que le format contient face a ce que le parseur lit : le relancer
   apres chaque evolution.
2. NE JAMAIS changer un comportement de rendu sans la mesure qui le justifie.
3. Quand un decodage echoue : MONTRER LES OCTETS, pas essayer un offset de plus.
4. Toute mesure de similarite embarque son TEMOIN NEGATIF, sinon elle valide tout.
5. L'oreille de l'utilisateur vaut une mesure : chaque « ca ne colle pas » avait une
   cause structurelle dans le format.
