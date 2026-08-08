# Lot 5 v7.5 — Catalyst et Vagabond au catalogue, et l'oracle du containment

> Branche `feat/v75`, 2026-08-08. Repond a la piste B du `PLAN_MASTER_FILM_KILLFEED_REJEU.md`
> (etapes 5.2 et 5.3 : « c'est ici que Catalyst et Vagabond obtiennent leur fond de carte »)
> et aux items « Fragmentation Heavies » / « Highpower » du `REGISTRE_REPORTS.md`.
>
> Ce document dit ce qui est ENTRE au catalogue, ce qui a ete REFUSE et sur quelle mesure.
> Le gate humain de la piste B n'est PAS declare passe ici : il attend une validation
> ecrite de l'utilisateur, carte par carte, depuis l'artefact de revue (§6).

## 1. Le resultat en une page

| item | statut | ce qui est entre |
|---|---|---|
| L5a Catalyst | partiel | zones et bornes verifiees ; une 2e variante d'asset ajoutee ; **fond de carte refuse** |
| L5b Vagabond | fait | **3 zones de Bastion reelles** au catalogue — l'oracle du containment est disponible ; fond de carte refuse |
| L5c alias Heavies | fait | `NormalizeMapName` retire le suffixe ; **+43 matchs** retrouvent leurs bornes |
| L5d zones de Bastion sur Highpower | non traite, prouve absent | la donnee n'existe dans aucune des deux variantes de la carte |
| L5e artefact de revue | fait | page HTML autonome, en attente de validation |

Et un **defaut corrige au passage, decouvert en verifiant L5b** : le choix de la variante
`.mvar` d'une carte publiait le rack du canevas Forge au lieu de la carte jouee (§3).

## 2. Ce qui existait deja, et qu'il ne fallait pas refaire

Verifie sur pieces avant de coder :

- **Catalyst avait deja ses bornes** (`map_quant_bounds.json`, cle `catalyst`) **et ses
  zones** (`map_objectives.json`, `catalyst_map` : 3 zones de Bastion, 5 zones
  d'Extraction). Ce qui lui manquait est le FOND DE CARTE, et lui seul.
- **Vagabond avait ses bornes** (`fo08_wetland`) mais **aucune entree** au catalogue
  d'objectifs. C'etait le vrai manque, et c'est celui qui bloquait l'oracle.
- La chaine `.module -> triangles` n'existe **qu'en Python jetable, sur Cliffhanger seule**
  (`cartes/HANDOFF_GEOMETRIE_TRIANGLES.md`). Son portage en Go est le chantier
  `PLAN_BELLE_CARTE_TRIANGLES` — hors perimetre de ce lot, qui devait utiliser les outils
  existants.

## 3. Le rack du canevas — le defaut trouve en cherchant les zones de Vagabond

`cmd/mapobj-build` retenait, parmi les `.mvar` d'un asset, **celui qui declare le plus
d'objectifs**. Sur les cartes baties dans Forge, ce critere se retourne :

| fichier | objets | objectifs | emprise des objectifs | verdict |
|---|---:|---:|---|---|
| `fo08_wetland.mvar` | 100 | 20 | **8,2 m**, tous a z = 50,50 exactement | canevas livre avec le jeu : le RACK des objets de mode, un exemplaire de chaque, range hors terrain |
| `map.mvar` | 4 709 | 4 | 22,3 m | la carte reellement batie |

L'ancien critere retenait le rack. Le catalogue a donc porte, pendant la duree de ce lot,
**trois « zones de Bastion » de 1 m de rayon posees a 2 m l'une de l'autre** — un objectif
au mauvais endroit, exactement ce que l'en-tete de `fetch.go` dit vouloir eviter.

**Correction** (`cmd/mapobj-build/variant.go`) : une variante dont les objectifs tiennent
dans moins de **5 %** de l'emprise de ses propres objets est ecartee. Calibration sur les
37 cartes du catalogue : le rack est a 2,3 %, la carte la plus basse ensuite est
`corpo_map` a 15,8 %, puis 44,4 % — le seuil est entre les deux avec un facteur ~2 de
marge des deux cotes. Si TOUTES les variantes sont ecartees, la carte n'entre pas au
catalogue et l'echec est logue : une carte absente vaut mieux qu'une zone fausse.

Trois temoins de test, dont deux tournent en CI :
`vagabond_fo08_wetland.mvar` doit etre ecarte, les deux `.mvar` de Cliffhanger ne doivent
pas l'etre, et `vagabond_map.mvar` (hors depot, 882 Ko) doit rendre ses 3 zones. Les deux
mutations du seuil (0,0 et 0,9) ont ete jouees et vues rouges.

**Effet de bord traite** : deux cartes exposent desormais un fichier nomme `map.mvar`
(Vagabond et une variante de Highpower). Le mode hors ligne `--refresh-from` lisait a plat,
donc leur aurait servi le MEME fichier — les zones d'une carte publiees sous le nom d'une
autre, sans un mot. `--save-mvar` depose maintenant dans un sous-dossier par `map_id`, et
le repli a plat est REFUSE quand le nom est partage.

## 4. Les fonds de carte : refuses, et sur quel chiffre

`cmd/mapstruct-build` lit le bloc `instanced geometry instances` du tag sbsp et publie des
AABB. Mesure du 2026-08-08 (depot de jeu `C:/Program Files (x86)/Steam/.../deploy/pc`) :

| carte | emprises | couverture brute | **couverture sans les boites > 200 m²** |
|---|---:|---:|---:|
| ridgeline (Cliffhanger) — *publiee* | 10 223 | 100 % | **51,1 %** |
| sgh_streets (Streets) — *publiee* | 10 908 | 100 % | **76,6 %** |
| catalyst | 11 178 | 49,0 % | **8,6 %** |
| fo08_wetland (Vagabond) | 788 | 100 % | **12,6 %** |

- **Catalyst** : sa structure vit dans le maillage de rendu NON instancie, que ce binaire
  ne lit pas. C'est ecrit dans l'en-tete de `cmd/mapstruct-build` depuis sa creation, et la
  mesure le confirme. Figer ce fichier donnerait un fond troue qui se lirait comme une
  carte.
- **Vagabond** : ses 100 % bruts sont tautologiques (une seule boite de 312 182 m²). Et
  surtout, **le canevas n'est pas la carte** : Vagabond est batie dans Forge, ses 4 709
  objets sont dans `map.mvar`, pas dans le BSP. Aucun reglage de `mapstruct-build` ne peut
  rendre cette carte — c'est une autre source.

Aucun fichier n'a donc ete ajoute a `reference/map_structure/`.

## 5. Highpower : la donnee n'est pas la, et c'est mesure

Les deux assets de Highpower ont ete telecharges et parses ; leurs quatre variantes
declarent **zero** `strongholds_zone` :

| asset | fichier | objets | objectifs | zones de Bastion |
|---|---|---:|---:|---:|
| `c494ef7c` (84 matchs) | `btb_highpower.mvar` | 605 | 26 | 0 |
| `33c6505d` (13 matchs, AJOUTE au catalogue) | `map.mvar` | 638 | 28 | 0 |

Ce n'est pas un defaut d'extraction : une recherche murmur3 sur 45 noms de mode croises
avec 26 suffixes, plus 40 candidats explicites (`total_control_zone`, `land_grab_zone`,
`koth_hill`, `hill_include`...), ne fait retomber **aucun** hash non resolu de Highpower
sur un label de zone. Or la carte EST jouee en mode a zones : 25 matchs de Total Control
au registre. Les zones de Total Control ne sont donc pas dans la variante de CARTE. Piste
suivante, non ouverte : la variante de MODE (game variant). Consigne au registre.

Sous-produit de la recherche : 7 labels jusque-la inconnus retombent proprement
(`oddball_exclude`, `extraction_exclude`, `slayer_include`, `firefight_exclude`,
`firefight_objective`, `minigame_exclude`, `forge_exclude`). Aucun n'est un role — les
ajouter a `labelNames` ne changerait aucune zone, seulement le compte des non resolus.
Hors perimetre, consigne au registre.

## 6. L'artefact de revue — le gate n'est pas passe

Page autonome, generee hors depot :

	C:\Users\GUILLA~1\AppData\Local\Temp\claude\c--Users-Guillaume-Projects-LevelUp\
	  1d8d44ec-6f37-47fb-b05b-a7d238cb6ffd\scratchpad\revue_cartes_lot5.html

Elle porte, par carte : la vue du dessus (structure mesuree en gris, objectifs avec leur
forme et leur ORIENTATION reelles), la barre d'echelle de 10 m, la couverture publiee, le
detail metrique des zones de Bastion (position, taille pleine, hauteur au-dessus et
au-dessous, distances mutuelles) et les criteres generaux.

**Les temoins propres a chaque carte sont laisses VIDES, volontairement.** Le gate humain
de la piste B interdit a la session de choisir ses temoins apres avoir vu son rendu, et
c'est le cas ici. Ils reviennent a l'utilisateur (ou a une source externe : carte en jeu,
rendu Reclaimer). Rappel du plan maitre : l'anneau du fer a cheval et les deux ponts sont
les temoins de **Cliffhanger uniquement**, ils ne se reutilisent pas.

## 7. Ce que ce lot debloque

- **L'oracle du containment est disponible** : Vagabond porte ses 3 zones reelles, et le
  releve terrain du 2026-08-02 devient rejouable par le code du lot 4. Corpus : 3 matchs
  Strongholds sur Vagabond au registre.
- **+43 matchs** retrouvent leurs bornes de dequantification par l'alias Heavies
  (22 Fragmentation, 11 Highpower, 10 Breaker), sur les 45 que le lot 4 comptait sans
  bornes. **+13 matchs** Highpower gagnent une entree de formes (sans zone de Bastion).
- Le catalogue d'objectifs passe de **34 a 37 cartes**. Aucune entree preexistante n'a ete
  modifiee (verifie par comparaison entree par entree avant/apres).
