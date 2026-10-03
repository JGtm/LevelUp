# RECETTE — sons d'armes Halo Infinite, de l'extraction a la livraison

> Document de reference FINAL (2026-08-16, mis a jour 2026-09-06). Le travail est livre une
> fois ; toute reprise (mise a jour du jeu, arme nouvelle, erreur decouverte) REJOUE cette
> recette, elle ne l'amende pas a la main. L'historique et les preuves sont au plan
> (`PLAN_EXTRACTION_SONS_ARMES.md`) et au handoff (`HANDOFF_SONS_ARMES_2026-08-15.md`) ;
> ce fichier ne contient que le COMMENT refaire.
>
> **ETAT DES SIX SCRIPTS DE `_outils/` (lot v2 G.3, 2026-09-06)** — deux des six sont
> desormais portes en Go, dans le depot, avec une sortie prouvee octet a octet contre le
> script qu'ils remplacent (jeu d'entrees synthetique : les dossiers d'armes avec leurs
> `.wav` sources de la machine d'origine ont disparu depuis la livraison du 2026-08-16,
> seuls `_donnees/*.json` et les scripts eux-memes restent) :
>
> | Script | Etape | Etat |
> |---|---|---|
> | `akpk_unpack.py` | 1 (.pck -> .wem) | **Porte** — `weapon-sounds -mode pck-dump` (2026-09-02) |
> | `conv_lot.py` | 1 (.wem -> .wav) | hors depot (vgmstream) |
> | `coups_lot.py` | 2/3/5 (analyse banks, rendu des coups) | hors depot |
> | `manifeste2.py` | 6 (manifeste de triage) | hors depot |
> | `reinjecte.py` | 6 (injection dans tri.html) | hors depot |
> | `livraison.py` | 7 (fusion moteur sonore) | **Porte** — `weapon-sounds -mode livrer` (2026-09-06) |
>
> Quatre scripts restent hors depot : ils ne produisent pas les assets versionnes finaux
> (`static/sounds/`, `weaponSoundVariations.ts`) mais les etapes intermediaires du triage
> humain (banks -> candidats, vote, manifeste HTML) — hors perimetre du constat H2 qui visait
> la reproductibilite des SEULS assets livres.

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

	# .pck -> .wem (un dossier par arme) — PORTE EN GO (2026-09-02), preferer cette voie :
	cd apps/go-api && go run ./cmd/weapon-sounds -mode pck-dump -pck <fichier.pck> -out <dossier> [-wem id1,id2,...]
	# equivalent Python historique (hors depot), garde pour memoire :
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

## 7. Livraison — FUSION avec le moteur sonore du rejeu (etat final, 2026-09-06)

Le rejeu 2D possede son moteur sonore (branche v75 : `replaySound.ts` regles pures,
`replayAudio.ts` lecture, `useReplaySound.ts` couture React — tirs, kills, grenades,
melee, equipements, filtres par categorie, plafond de voix). La livraison s'y FOND :

	cd apps/go-api
	go run ./cmd/weapon-sounds -mode livrer -donnees <chantier>/_donnees [-sons <chantier>] [-depot <depot>]

`-sons` par defaut = le parent de `-donnees` (les dossiers d'armes et `_donnees` sont
SIBLINGS sous la racine du chantier, comme dans l'archive Desktop d'origine). `-depot` par
defaut = la racine du depot courant (`title.FindRepoRoot`). PORTAGE FIDELE de
`_outils/livraison.py` (lot v2 G.3, 2026-09-06 — detail dans `cmd/weapon-sounds/livraison.go`
et ses fichiers voisins `livraison_audio.go`/`livraison_variation.go`/
`livraison_orchestrate.go`/`livraison_mt19937.go`) : meme algorithme, memes structures de
donnees, meme generateur pseudo-aleatoire (Mersenne Twister graine 20260816, port verifie
bit a bit contre CPython) pour l'unique arme rendue par evenement plutot que copiee depuis un
fichier vote (Covenant_provoker -> hinf_ravager). Preuve octet a octet contre le script
Python sur un jeu d'entrees synthetique (les quatre chemins de decision : role rendre-evenement,
role copie-directe, vote de coup, vote d'evenement en repli), journal
`.ai/V7.5/v2/LOT_G.md` — les dossiers d'armes avec leurs `.wav` sources reels ont disparu de
ce poste depuis la livraison du 2026-08-16 : la preuve sur les VRAIS sons du jeu n'a pas pu
etre rejouee, seule la fidelite algorithmique est prouvee.

Script Python historique (hors depot, garde pour memoire — NE PLUS L'UTILISER) :

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
