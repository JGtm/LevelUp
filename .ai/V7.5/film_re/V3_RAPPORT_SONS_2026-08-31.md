# V3 — RAPPORT : sons des véhicules (chantier véhicules-tourelles)

> Exécuteur : lot V3 du `PLAN_VEHICULES_TOURELLES.md`. Worktree `LevelUp-wt-vehicules`.
> Aucun commit, aucune écriture DuckDB, aucun nouveau Python. Un seul build Go (cache isolé),
> un seul module 7,24 Go en RAM à la fois.
>
> **Révision 4 — LE DÉPLACEMENT EST CORRIGÉ SUR LE FOND.** Les rev 1-3 cherchaient le son de
> mouvement au mauvais ENDROIT (un event `snd!` de la banque `_veh_`, pris par durée puis par
> structure — refusé deux fois à l'oreille). La rev 4 suit la BONNE chaîne : **le tag `vehi`
> RÉFÉRENCE ses sons de mouvement**, et leur audio vit dans des banques **ANONYMES PARTAGÉES**,
> pas dans le pck `_veh_` (qui ne porte que tir/impacts). Hypothèse du coordinateur confirmée.

## 0. Ce qui est livré (révision 4)

**TIR** (banques `_veh_` confirmées par FNV-1) — inchangé, confirmé : **Wasp, Scorpion,
Gungoose, Warthog** (coup 3p/1p + rafale à la cadence) ; **Falcon** = la mitrailleuse partagée.

**DÉPLACEMENT** (nouvelle chaîne `vehi → snd! → events → banque anonyme`) :
- **Ghost** — 3 segments de mouvement (moteur antigrav), banque `f2b547a1`.
- **Wasp** — 2 segments, banque `09e2e7ee`.
- **Chassis `033e41df`** — 4 segments, banques `acaa4513`/`2ecd73ab` (identité châssis à
  reconnaître à l'oreille : Chopper/Banshee/Wraith probable).

Chaque segment est fourni brut + un aperçu bouclé 8 s. Manifeste : `manifeste_v3.json` (rev 4).

## 1. La découverte de fond — où vit le son de déplacement

Nouveau mode **`vehi-sons`** (lecture seule, ajouté à `cmd/weapon-sounds`) : il suit, pour
chaque tag `vehi`, ses références `lsnd`/`snd!`/`effe` lues **inline** (comme la ref `hlmt`,
`internal/himap/vehicules.go`) et par la table de dépendances, puis chaque son → sa banque
(`sbnk`) et les mots de son corps (candidats d'événement).

Sur les **13 tags `vehi`**, l'architecture du mouvement est :

    COUCHE PARTAGEE (les 13 vehi)   lsnd 06ba1096  + effe 7ff6244a / 4488f97a
                                    banques ANONYMES e793c135 / de65048f
    COUCHE SPECIFIQUE (certains)    snd! dedies :  Ghost -> f2b547a1
                                                   Wasp  -> 09e2e7ee
                                                   groupe 033e41df -> acaa4513 / 2ecd73ab

- **Le son de mouvement n'est PAS dans le pck `_veh_`** (qui porte tir/impacts) : il est dans
  des banques **anonymes** (non nommées par FNV-1), partagées ou spécifiques. C'est
  exactement l'hypothèse : « mauvais TYPE de tag, mauvais ENDROIT ».
- **Le `lsnd 06ba1096` est GÉNÉRIQUE** (le même pour les 13). Il pointe une `sndc f1327648`
  qui a **0 dépendance** (une classe de mixage, PAS un switch par type de véhicule). La
  différenciation par véhicule se fait donc dans les **`snd!` dédiés**, pas dans le lsnd.

## 2. Ce qui est reconstruit, et sa preuve

Chaque déplacement est rendu par la MÊME sémantique que le tir (`event → couches → variantes,
gains sommés`, via `coups_lot.py`). Les segments sont des **corps de boucle courts**
(0,65–2,2 s), structurellement des boucles de moteur, pas des transitoires de tir.

| Véhicule | chaîne de preuve | segments (durées) |
|---|---|---|
| **Ghost** | `vehi 00002705` (weap `0000a4bc`) → snd! `17dc7858`/`37a178d9`/`f4578eb3` → events `c5408c26`/`ae9c3fe5`/`b10e14b0` → banque anonyme `f2b547a1` → wems | 3 (1,15 / 2,18 / 1,43 s) |
| **Wasp** | `vehi b65b3b4a` (weap `11725dc4`) → snd! `50c6acf3` → events `2cdb4b0d`/`36993720` → banque anonyme `09e2e7ee` → wems | 2 (1,19 / 1,19 s) |
| **Chassis `033e41df`** | `vehi 000025aa` (weap `033e41df`) → snd! `3fb3129f`/`a5cfe3b4`/`dbde1c43`/`98cef02b` → events → banques `acaa4513`/`2ecd73ab` | 4 (1,25 / 1,18 / 1,64 / 1,26 s) |

Les 3 segments du Ghost (durées différentes) sont probablement idle / mid / haut régime, OU
start / loop / stop — **à trancher à l'oreille** (aucun nom ne survit dans les tags).

## 3. Les TIRS restent confirmés (vérifiés)

Chaîne relue sur pièces, banques `_veh_` confirmées FNV-1 : Wasp event `e22a0d32` = 3 couches
RandomSequence (+38 / +33 / +31 dB **sommés**) ; Gungoose `0a54bfb7` = 3 couches
(+10 / +10 / +9 dB). Cadences 450 / 420 / 126 cpm lues du tag. Assemblage correct. Falcon =
mitrailleuse détachable (banque `bd807a77` = `sb_010_tur_un_machinegun`).

## 4. Ce qui reste ouvert (honnête)

1. **Scorpion / Gungoose / Warthog / Falcon : PAS de `snd!` de mouvement dédié** dans leur
   `vehi` — uniquement la couche PARTAGÉE (lsnd générique + effe). Leur moteur spécifique
   n'est **pas isolable par les tags**. Deux pistes : (a) le son est le lsnd générique
   `06ba1096` (même moteur pour tous, peu probable pour un char) ; (b) il est sélectionné à
   l'exécution par un **RTPC / switch de type-véhicule** — c'est le « **Ghidra au besoin** » :
   trouver le POST de l'événement Wwise de déplacement et le paramètre de vitesse. Non fait ce
   tour (le tag ne l'expose pas ; c'est une session Ghidra dédiée).
2. **Identité du châssis `033e41df`** (3 vehi) : non présent dans lot2 → à reconnaître à
   l'oreille (Chopper/Banshee/Wraith). La chaîne son est propre ; seul le nom manque.
3. **Segmentation start/loop/stop** : les segments sont fournis bruts + bouclés ; distinguer
   « la boucle soutenue » des amorces demande l'écoute.

## 5. Découvertes hors périmètre

1. **Les banques de mouvement sont ANONYMES** (non nommées par FNV-1) : `e793c135`,
   `de65048f` (partagées), `f2b547a1`, `09e2e7ee`, `acaa4513`, `2ecd73ab` (spécifiques). Elles
   font partie des 712 banques anonymes du module — c'est pourquoi la voie « nom de pack » les
   ratait.
2. **Le `lsnd 06ba1096` + `sndc f1327648`** : contrôleur de boucle générique + classe de
   mixage. À creuser si on veut la couche commune (bruit de roulement/vent partagé).

## 6. Outillage ajouté (lecture seule, gofmt-clean)

`cmd/weapon-sounds/vehicules_sons.go` — mode `vehi-sons` : `vehi → lsnd/snd!/effe → sbnk +
mots`, avec les refs `weap` inline (identité du véhicule, croisée avec lot2). Passe 1 sur
`any/globals` (0,62 Go) ; les banques se résolvent en passe 2 par `lot -banks <sbnk>`
(`pc/globals`).

## 7. Emplacements

- **Livrables** : `.ai/V7.5/film_re/sons_v3_reconstruits/<Véhicule>/{tir,deplacement}/*.wav`
  + `manifeste_v3.json`. Déplacement RÉEL : Ghost, Wasp, Chassis_033e41df. Tir : Wasp,
  Scorpion, Gungoose, Warthog, Falcon.
- **Intermédiaires** : `<scratch>\donnees\{vehi_sons, lot1_vehbanks, lot1_veh, lot2_veh,
  banks_noms}.json`, `weapon-sounds.exe`, `emb2\`. `_outils` restauré.
