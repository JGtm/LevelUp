# L'origine d'une prise : la LEVITATION est refutee, la RECURRENCE mesure le trafic des joueurs

Date : 2026-09-01. Lot 6, RECHERCHE PURE. Fichier unique et NEUF :
`apps/go-api/internal/analysis/replay/equipment_origin_levitation_research_test.go`, sous gardes
`PICKUP_FILM` + `PICKUP_MAP` (celles de `glResolve`). Saute sans elles : aucun effet en CI.
Aucun fichier du lot de publication n'est touche (borne de revue `684712b61..3dfd782d6` gelee).
Un film par process, lecture seule, AUCUNE cuisson.

## LES DEUX IDEES, ET CE QU'ELLES ATTAQUAIENT

Le lot 5 avait laisse l'origine non publiable pour deux raisons distinctes : le juge temporel
plafonnait a 25,6 % d'injectivite, et la branche « point d'apparition de la carte » n'etait pas
testable faute de catalogue. Les deux idees de ce lot sont GEOMETRIQUES la ou le lot 5 etait
temporel, et chacune vise un des deux blocages :

- **LEVITATION** — un objet pose sur un socle FLOTTE, un objet lache REPOSE. Si la hauteur
  separe, elle donne l'origine d'un objet SANS l'apparier a quoi que ce soit : la
  non-injectivite cesse d'etre un probleme.
- **RECURRENCE** — un point d'apparition sert plusieurs fois dans un match, un lacher n'a pas de
  raison de se repeter au meme endroit. Les amas de naissances construiraient le catalogue
  manquant.

**Verdict global : les deux sont REFUTEES, chacune par son propre temoin pre-enregistre.**

## LEVITATION — REFUTEE, et cette fois avec un instrument dont le bruit est mesure

### La mesure de hauteur n'a pas besoin de connaitre le sol

La reference est la hauteur des BIPEDES qui passent au meme endroit. Que `biped.Z` designe les
pieds, le nombril ou les yeux est indifferent : c'est un decalage CONSTANT, qui se simplifie des
qu'on COMPARE deux populations d'objets a la meme reference.

    levitation(objet) = objet.Z - mediane{ bipede.Z : bipede a moins de 1,5 m en XY }

Un objet avec moins de 5 releves de bipede a proximite est ECARTE, jamais approxime.

### L'etalon vient de la CARTE, pas du film

`map_weapon_pads.json` donne les emplacements declares par le `.mvar` (precision mesuree au plan
SOCLES_MVAR : 32 positions d'oracle, 32 appariees a moins d'un metre, mediane 0,01 m). Population
SOCLE = objet `ti=42` reposant a moins de 1,0 m d'un emplacement declare ; population LACHEE =
a plus de 5,0 m de TOUT emplacement. La bande morte entre 1 et 5 m est ecartee a dessein.

### LE PREMIER FILM A DONNE UNE CALIBRATION VIDE, ET C'EST LE PIEGE A RETENIR

Premier essai sur `000d5950` (Cliffhanger) : separation 0,03 m, temoin permute 0,47 m — un
resultat qui aurait pu se lire comme une refutation. **Il ne valait rien**, et la raison est
ecrite dans le depot depuis le catalogue des socles : *« Cliffhanger porte 17 socles, en rend 10
en CTF et ZERO en Super Fiesta »*. `000d5950` EST une Super Fiesta. Le mode n'allume aucun
socle, donc mes 14 objets « sur socle » n'etaient que des armes lachees tombees pres d'un
emplacement ETEINT. La carte declare, le mode allume — confondre les deux vide l'etalon.

Le meme piege avait deja ete paye au lot 3 (« ce film rend 0 socle et 0 occupation »). Il est
consigne ici une seconde fois parce qu'il s'est represente sous un autre visage.

### La calibration VALIDE, sur un film a socles

`01e1f945` — Catalyst, KOTH:Arena, le film dont le lot 5 avait deja etabli qu'il publie 10
socles et 46 occupations.

| mesure | 01e1f945 (Catalyst) | 000d5950 (Cliffhanger, etalon VIDE) |
|---|---|---|
| emplacements declares par la carte | 11 | 18 |
| vies `ti=42` | 646 | 549 |
| population SOCLE · LACHEE | **31 · 266** | 14 · 183 (sans valeur) |
| ecartees (moins de 5 releves) | 172 | 85 |
| **levitation mediane SUR SOCLE** | **0,01 m** | 0,04 m |
| **levitation mediane LACHEE** | **0,01 m** | 0,02 m |
| **SEPARATION** | **0,00 m** | 0,03 m |
| TEMOIN etiquettes permutees (pire de 200) | **0,11 m** | 0,47 m |
| classement au seuil : socle · lachee | 64,5 % · 62,4 % | 57,1 % · 61,7 % |

**Verdicts sur la calibration valide : V1 (separation >= 0,50 m) NON TENU a 0,00 m · V2 (temoin
< 0,15 m) TENU a 0,11 m · V3 (>= 70 % dans CHAQUE population) NON TENU · V4 (levitation refutee)
TENU.**

**C'est V2 qui donne son poids a la refutation.** Sur le premier film, le bruit (0,47 m)
depassait le seuil recherche : l'instrument etait aveugle, et un « pas de separation » n'aurait
rien voulu dire. Sur `01e1f945` le plancher de bruit tombe a 0,11 m — l'instrument VOIT une
separation de 0,15 m et au-dela. Il en mesure **0,00**. Les deux medianes sont egales au
centieme.

**L'APPLICATION A L'EQUIPEMENT N'A PAS ETE FAITE, et c'est deliberé** : appliquer a une
population inconnue un seuil que l'etalon ne valide pas produirait un pourcentage sans
signification. Le test s'en abstient explicitement plutot que de publier un chiffre.

### La lecture la plus economique, donnee comme HYPOTHESE

Deux mesures independantes de ce lot pointent dans la meme direction : la levitation est nulle,
et les naissances `ti=37` se concentrent a une altitude quasi constante par carte
(z ~ -1,9 sur Cliffhanger, z ~ 22-27 sur Catalyst). **Le film semble reporter la position de
CONTACT AU SOL d'un objet, pas sa hauteur visuelle.** Ce n'est pas prouve — aucune mesure de ce
lot ne l'etablit — mais c'est la seule lecture compatible avec les deux observations, et elle
expliquerait pourquoi l'intuition de jeu (un objet sur socle flotte) ne se retrouve pas dans les
octets.

## RECURRENCE — LES AMAS EXISTENT, MAIS ILS MESURENT LE TRAFIC DES JOUEURS

Regroupement glouton des naissances `ti=37` (rayon 1,0 m, amas retenu a partir de 3 naissances),
ventile par nature grace au nommage du lot 5.

| film | nature | naissances | amas | TEMOIN uniforme | amas / naissance |
|---|---|---|---|---|---|
| 01e1f945 | equipement | 27 | **1** | 0 | 0,037 |
| 01e1f945 | **grenade** | 124 | **14** | 0 | **0,113** |
| 000d5950 | equipement | 110 | **10** | 0 | 0,091 |
| 000d5950 | **grenade** | 185 | **21** | 0 | **0,114** |

**Verdicts : R1 (au moins un amas) TENU · R2 (reel >= 3x le temoin uniforme) TENU, 15 et 31
contre 0 · R3 (CONTRASTE : l'equipement s'amasse plus que les grenades) NON TENU sur les DEUX
films.**

**R3 est le juge, et il refute.** Les grenades naissent COLLEES a un bipede qui les lance — le
plan NOMMAGE_EQIP l'a mesure (96 a 100 % a moins de 3 m d'un poseur, distance mediane 0,001 a
0,004 de l'AABB). Elles ne PEUVENT donc pas avoir de point d'apparition sur la carte. Or elles
forment **plus d'amas par naissance que l'equipement**, sur les deux films (0,113 contre 0,037 ;
0,114 contre 0,091). Ce que les amas capturent est donc l'endroit ou les joueurs se battent, pas
un point d'apparition.

Sans R3, R1 et R2 auraient ete lus comme un succes : des amas nets, tres au-dessus du hasard
(temoin uniforme a ZERO sur les deux films). C'est exactement le resultat trompeur que le temoin
pre-enregistre etait la pour intercepter.

**R4 — croisement carte : sans conclusion, par construction.** Le seul amas d'equipement de
`01e1f945` est a 6,94 m du socle declare le plus proche ; ceux de `000d5950` a 1,55-5,93 m. Mais
le catalogue ne declare que des emplacements d'ARME (`rack`, `power`, `powerup`) : il n'y a rien
a quoi comparer un point d'apparition d'equipement. C'est la meme absence de donnees qui avait
rendu O3 non testable au lot 5, et ce lot ne la leve pas.

## CE QUI EST PROUVE / PLAUSIBLE / REFUTE

- **REFUTE — la levitation comme discriminant d'origine.** Separation 0,00 m sur une calibration
  valide (31 socle contre 266 lachees) dont le plancher de bruit est mesure a 0,11 m.
- **REFUTE — la recurrence des naissances comme detecteur de points d'apparition.** Les amas
  existent et depassent largement le hasard, mais les grenades — qui ne peuvent pas en avoir —
  en forment davantage par naissance, sur les deux films.
- **PROUVE — un etalon de socle exige un film dont le MODE allume les socles.** La carte declare,
  le mode allume ; `000d5950` (Super Fiesta) declare 18 emplacements et n'en allume aucun.
- **PLAUSIBLE, non prouve** — le film reporte la position de contact au sol d'un objet et non sa
  hauteur visuelle (deux observations concordantes, aucune mesure dediee).
- **INCHANGE** — l'origine d'une prise reste NON PUBLIABLE. Le lot 5 la laissait a 25,6 %
  d'injectivite par le juge temporel ; ce lot ferme deux voies geometriques de plus.

## CE QU'IL RESTE, ET CE QU'IL NE FAUT PLUS RETENTER

Ne plus retenter, sauf idee reellement neuve : la hauteur (refutee ici), la distance seule
(refutee au lot 4, mediane 1,33 m, part sous le metre 46 %), la fin de vie (lot 5, 25,6 %), la
recurrence des naissances (refutee ici).

Ce qui reste ouvert :

1. **Une VRAIE fin d'objet dans le film.** Trois pistes ont deja echoue (record de suppression,
   queue de records, recensement d'images-cles) ; la quatrieme serait un evenement natif de
   destruction, a chercher dans le registre des 50 blocs du chantier trame — le meme chemin qui
   a livre `biped_pickup`.
2. **Un catalogue de points d'apparition d'equipement extrait des `.mvar`**, sur le modele de
   `cmd/mapopads-build`. C'est la donnee qui manque a O3 et a R4 ; elle est extractible, et son
   absence est ce qui bloque la branche « point d'apparition » depuis deux lots.

## Reproduire

```
cd apps/go-api
PICKUP_FILM=<depot>/data/cache/film_chunks/01e1f945 PICKUP_MAP=Catalyst \
  go test ./internal/analysis/replay/ -run 'TestOriginLevitation|TestOriginBirthRecurrence' -v
PICKUP_FILM=<depot>/data/cache/film_chunks/000d5950 PICKUP_MAP=Cliffhanger \
  go test ./internal/analysis/replay/ -run 'TestOriginBirthRecurrence' -v
```

`01e1f945` est le film a socles (Catalyst, KOTH:Arena) : c'est le SEUL des trois qui calibre la
levitation. Un film par process, aucune cuisson.
