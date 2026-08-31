# V3 — RAPPORT : sons des véhicules (chantier véhicules-tourelles)

> Exécuteur : lot V3 du `PLAN_VEHICULES_TOURELLES.md`. Worktree `LevelUp-wt-vehicules`.
> Aucun commit, aucune écriture DuckDB, aucun nouveau Python, aucune modif du Go de prod.
> Un seul build Go, un seul module 7,24 Go en RAM à la fois (cache Go isolé).
>
> **Révision 3.** La rev 1 avait choisi les déplacements par HEURISTIQUE DE DURÉE (les clips
> les plus longs) — erreur que la RECETTE interdit, signalée par l'utilisateur (« aucun
> déplacement, ce ne sont que des tirs isolés »). La rev 2 a reconstruit le déplacement comme
> un ÉVÉNEMENT MOTEUR. La rev 3 a vérifié l'IDENTITÉ DES BANQUES par `banks-noms` (FNV-1) et
> découvert que plusieurs de mes appariements par intersection de wems étaient FAUX — dont un
> prouvé (Wraith = le CTF). Le livrable est réduit à ce qui est CERTAIN.

## 0. Ce qui est livré (révision 3)

- **4 véhicules COMPLETS** (banque confirmée par FNV-1) : **Wasp, Scorpion, Gungoose,
  Warthog à roquettes** — tir reconstruit (coup 3p/1p + rafale à la cadence) **et** moteur
  reconstruit (événement, pas clip par durée).
- **Falcon (tourelle LMG)** : tir seul, = la mitrailleuse détachable (banque partagée) ; pas
  de banque châssis Falcon, donc pas de moteur/rotor.
- **Ghost, Banshee, Wraith, Chopper** : **rien de fiable** — leur banque n'est pas identifiée
  (voir §3). Les moteurs de la rev 2 sont RETIRÉS (celui du Wraith était un son de drapeau).

Manifeste : `sons_v3_reconstruits/manifeste_v3.json` (par véhicule : `banque_confirmee`,
et par son : rôle, event, chaîne, statut RTPC, confiance).

## 1. La chaîne, identique pour le tir ET le moteur

Tir et déplacement sont reconstruits par la MÊME sémantique (en-tête `arbre.go`) :
`événement → couches (Blend résolu au point de référence) → une variante par couche → gains
de chemin SOMMÉS`. Rendu par le renderer existant `coups_lot.py`. La rafale enchaîne 10
déclenchements à la cadence lue du tag (`barrels → rounds/s`).

Commandes (exactes) — un seul build, modules jamais simultanés :

    go build -o <sc>\weapon-sounds.exe ./cmd/weapon-sounds                       # 1 fois, CGO winlibs
    weapon-sounds.exe -mode lot      -module pc/globals/...  -pck <packs_veh renommes> ...  # 7,24 Go
    weapon-sounds.exe -mode lot-tir  -module any/globals/... -json lot1_veh.json ...        # 0,62 Go
    weapon-sounds.exe -mode banks-noms -module pc/globals/... -json banks_noms.json         # 7,24 Go (verif identite)
    python akpk_unpack.py / conv_emb.py   (extraction pck+embarques -> wav, scripts existants)
    python coups_lot.py                   (rend coup + rafale, et l'event MOTEUR fourni en lot2)

Contournement sans éditer le Go : `lot.go:51` (`motifsPck`) ne globe pas `_veh_` ; j'ai copié
les packs `_veh_` renommés `sb_010_wea_*` (IDs `.wem` identiques) dans un scratch pointé par
`-pck`.

## 2. La correction du DÉPLACEMENT — un événement moteur, pas le wem le plus long

Le moteur est un ÉVÉNEMENT de la banque, une BOUCLE pilotée par un RTPC de vitesse. Je
l'identifie par sa **STRUCTURE** (pas par la durée) :

- **Signal fort** : événement non-tir résolu à **plusieurs couches** (les enfants du Blend
  RTPC au point de référence), en **boucle continue** (`repetitions=0`, `mode_enchainement=5`)
  avec **modulation de pitch** (RTPC de vitesse). Exemple Wasp `5baca8ee` : 3 couches
  RandomSequence, 19 variantes chacune, continu, pitch ±80c — un bourdon soutenu, pas un
  transitoire de tir.
- Le marqueur `sLoopCount` est absent sur certains châssis (le jeu impose la boucle par une
  paire Play/Stop — point ouvert n°5 de `RE_GESTES_SONORES`) : pour ceux-là, le moteur est
  l'événement non-tir **à couches multiples** le plus soutenu.

Rendu : l'événement moteur est passé à `coups_lot.py` comme un « mode » (cadence 0, pas de
rafale) ; le rendu somme ses couches au point de référence. Contrôle de durée des moteurs
rendus : **0,7 s (Wasp, grain continu) à 4,7 s** — soutenus, structurellement distincts des
tirs (Wasp tir 3p = 3,13 s percussif). Un aperçu bouclé à 8 s (ffmpeg `-stream_loop`)
accompagne chaque moteur pour l'écoute.

## 3. LA DÉCOUVERTE QUI CHANGE LE LIVRABLE — l'identité des banques (banks-noms)

En PASS 1, l'appariement `pck → banque` se fait par intersection de wems. Pour les 4 UNSC
au sol, le **score** est fort (wasp 536, falcon-MG 556, scorpion 169, wargoose 146,
rockethog 156). Pour Ghost, Wraith, Chopper : **score = 1** — signal d'un FAUX appariement.

`banks-noms` (identifiant Wwise = FNV-1 du nom de pack) tranche sur pièces :

| Châssis | banque que j'avais mise | ce qu'elle EST vraiment (FNV-1) | verdict |
|---|---|---|---|
| Wasp | `4993b379` | **sb_010_veh_un_wasp** | **CONFIRMÉ** |
| Scorpion | `05a51e0a` | **sb_010_veh_un_scorpion** | **CONFIRMÉ** |
| Gungoose | `38167604` | **sb_010_veh_un_wargoose** | **CONFIRMÉ** |
| Warthog roq. | `a52af042` | **sb_010_veh_un_rockethog** | **CONFIRMÉ** |
| Falcon LMG | `bd807a77` | **sb_010_tur_un_machinegun** | banque de la MITRAILLEUSE (tourelle), pas un châssis |
| Wraith | `61007dcf` | **sb_004_mod_mp_ctf** | **FAUX — c'est le DRAPEAU** (confirme `RE_GESTES §4.3`) |
| Ghost | `01862ab3` | (anonyme) | non confirmé, score 1 |
| Banshee | `84bc4790` | (anonyme) | non confirmé, score 26 |
| Chopper | `8f5a3161` | (anonyme) | non confirmé, score 1 |

FNV-1 du nom de pack ne trouve de banque QUE pour les 4 confirmés. Pour Ghost/Banshee/Wraith/
Chopper/Falcon-châssis/Phantom/Pelican, **aucune banque nommée n'existe** et leurs wems
n'apparaissent dans le corps d'aucune banque — leur reconstruction demande d'abord de
**trouver la vraie banque** (elles sont vraisemblablement parmi les 712 banques anonymes, ou
partagées, ou nommées par un codename interne). Je ne livre donc pas de moteur pour eux :
mieux vaut rien qu'un son de drapeau étiqueté « moteur Wraith ».

## 4. Vérification des TIRS (demandée) — CONFIRMÉS

Chaîne relue sur pièces (lot1), banques confirmées :

- **Wasp** (cadence 450 cpm → intervalle rafale 0,133 s). Event `e22a0d32` = **3 couches
  RandomSequence** (6 variantes chacune), gains de chemin **+38 / +33 / +31 dB SOMMÉS** ;
  event `b0554fc9` = 2 couches (+12 dB, −3 dB).
- **Gungoose** (420 cpm → 0,143 s). Event `0a54bfb7` = 3 couches (+10 / +10 / +9 dB) ;
  `c240bac4` = 2 couches (+13, −3 dB).

`coups_lot.py` applique `10^(gain/20)` par couche puis somme et normalise à la crête. Les
couches et les gains sont donc **bien sommés**, la rafale est **à la cadence du tag**. Aucune
erreur d'assemblage trouvée. Les 4 tirs de banque confirmée (Wasp/Scorpion/Gungoose/Warthog)
sont fiables ; le tir Falcon est correct comme ARME mais provient de la banque mitrailleuse
partagée.

## 5. Table finale

| Véhicule | banque | confirmée | tir (weap → snd! → events, cadence) | moteur (event, couches) |
|---|---|---|---|---|
| **Wasp** | `4993b379` | oui | `11725dc4`→`ce8e2f81`→`b0554fc9,e22a0d32`, 450 cpm | `5baca8ee`, 3c continu RTPC (**haute**) |
| **Scorpion** | `05a51e0a` | oui | `00015cfa`→`b3adf402`→`251190f3,951f76c0`, tir unique | `0134da4e`, 2c (moyenne) |
| **Gungoose** | `38167604` | oui | `0042678e`→`276fa353`→`0a54bfb7,c240bac4`, 420 cpm | `e3082ea3`, 1c (moyenne) |
| **Warthog roq.** | `a52af042` | oui | `c7d50912`→`155f1354`→`38b83eb8,68b1a949`, 126 cpm | `28338148`, 1c (moyenne) |
| Falcon LMG | `bd807a77` | non (=MG tourelle) | `00015cd3`→`68c0807f`, 780 cpm (= mitrailleuse) | absent (pas de banque châssis) |
| Ghost / Banshee / Wraith / Chopper | — | non | non résolu | **banque non identifiée** |

Multi-armes détecté (`weap_tags`) mais non rendu (2e arme) : Scorpion coaxial `49e40d17`,
Wasp missiles `d3c407ed`, Falcon `003f5824`/`0c6fd911`.

## 6. Ce qui reste à faire (douteux / follow-up)

1. **Trouver les banques Ghost/Banshee/Wraith/Chopper** (et le châssis Falcon, Phantom,
   Pelican). Piste : leurs wems ne sont référencés dans aucun corps de banque → banque
   anonyme/partagée/codename. `banks-noms` liste 712 banques anonymes ; croiser leurs wems
   embarqués (DIDX) avec les wems du pck de chaque châssis donnerait le bon appariement,
   indépendant du score d'intersection qui a échoué ici.
2. **Confiance moyenne des moteurs Scorpion/Gungoose/Warthog** : boucle imposée par le jeu
   (Play/Stop), pas déclarée en banque → l'event moteur est le meilleur candidat structurel,
   à confirmer à l'oreille (votes priment).
3. **Armes secondaires** (coaxial Scorpion, missiles Wasp, 2 armes Falcon) : passe dédiée par
   `weap` — `lot-tir` ne retient que le weap dominant par banque.
4. **falcongrenadelauncher + pelican** : AUCUNE banque appariée en PASS 1.

## 7. Découvertes hors périmètre

1. **`61007dcf` = le CTF, pas le Wraith** : mon appariement par intersection l'a faussement
   pris pour le Wraith (score 1). Leçon : l'appariement pck→banque par intersection de wems
   est **non fiable pour les banques véhicule à wems streamés** ; `banks-noms` (FNV-1) est la
   voie autoritaire.
2. **6 packs `sb_008_exp_vehicle_{large,med,small}_{covenant,unsc}`** = explosions de véhicule
   par taille/faction — matière pour l'« état détruit » du lot V1/V2.
3. Le mode `sofa` de la consigne n'existe pas dans l'outil ; `lot-tir` donne déjà la liste des
   `weap` par châssis via `weap_tags`.

## 8. Emplacements

- **Livrables** : `.ai/V7.5/film_re/sons_v3_reconstruits/<Véhicule>/{tir,deplacement}/*.wav`
  + `manifeste_v3.json`. Contenu : 4 véhicules complets (tir + moteur), Falcon tir seul.
- **Sources extraites** (durée_id) : `Desktop\Halo Infinite - Sons armes\{UNSC_wasp, …}`.
- **Intermédiaires** : `<scratch>\donnees\{lot1_veh, lot2_veh, lot2_moteur, coups_veh,
  banks_noms}.json`, logs, `weapon-sounds.exe`. `_outils` restauré.
