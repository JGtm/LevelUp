# NOTE — Visée du tir (type 36/105) percée au-delà des 19 % — 2026-08-31

> Suite de `NOTE_MODELE_EVENEMENTS_2026-08-30.md`. Clôt la question utilisateur « on a pas
> mieux que 19 % pour la visée ? ». Réponse : **oui — le cas modal (tir propre, 0 cible /
> 0 composante) est entièrement percé**, soit 3 à 6× plus de tirs que ce que `fire_events.go`
> couvrait. Instrument : `filmdec/lot1_visee_ghidra_research_test.go` (garde `LOT1_TRAME_FILM`).

## D'où l'on partait

`fire_events.go` lit la visée à l'offset FIXE bit 113, mais seulement sur le « chemin record
vide » (drapeaux 110==1, 111==0, 112==0) : **~19 %** des records longs. Ces offsets (attaquant
@36, arme @44/76, visée @113) sont des **atterrissages empiriques du cas canonique**. Hors de
lui, la position de la visée bouge.

## L'agent Ghidra (FUN_14080C1F8, tracé instruction par instruction)

Le record de tir/dégât est le **type 105** (`payload[0] >> 1`) ; c'est le MÊME paquet 0xD2 que
le modèle-M numérote « type 36 » (numérotation avec 2 bits de préfixe config/continuation). La
charge, dans l'ordre réel :

| Champ | Lecture | Fonction |
|---|---|---|
| a variant | R(1) | — |
| b bloc-sup | R(1) | — |
| c attaquant | R(7)+R(1) | FUN_141fcf670 |
| **d** | R(1) + **[si 0]** R(5) | FUN_1407f2034 |
| e | R(1) + [si 0] R(2) | FUN_1406d00ec |
| f arme (famille) | R(1) + [si 1] R(32) | FUN_14080d69c |
| g arme (variante) | R(32) | FUN_14080dec4 |
| i, j | R(1), R(1) | — |
| k | si bloc-sup : R(1)+R(1) | — |

Réserve honnête de l'agent (JUSTE) : les largeurs des 3 références d'en-tête viennent d'une
table (`0x1451f98d0`) **nulle dans le binaire au repos, peuplée au runtime** ; les boucles
cibles/composantes non vides restent donc de largeur runtime. On ne perce QUE le cas modal.

## Les DEUX corrections qui percent le cas modal

1. **Polarité du champ d.** Mon décodeur modèle-M sautait R(5) quand la garde vaut 1 ; le
   désassemblage dit **quand elle vaut 0**. Avec la bonne polarité, le décodage d'en-tête
   atterrit à une position **post-comptes stable de 111 sur 100 %** des paquets à visée vide,
   sur les 5 témoins.
2. **Les deux lecteurs composites (`cd5b8`, `eff64`) sont PARASITES dans le chemin modal.**
   `fire_events` place la visée à 113 = post-comptes(111) **+ 2**. Les 2 bits = les deux
   derniers drapeaux ; il n'y a AUCUNE lecture composite avant la visée. **La vraie visée est
   à post-comptes + 2.** (Les composites appartiennent au chemin non-modal, pas ici.)

## Preuve — l'oracle de concentration

Une vraie visée unitaire **sature un axe** (part de |composante| < 0.3 très haute, l'axe
vertical du repère du film) ; le bruit uniforme reste ~26 % par axe. Mesure sur les paquets que
`fire_events` NE couvre PAS (les nouveaux) :

| Témoin | fire_events (visée vide) | post-comptes+2, tout modal | GAIN (hors fire_events) | conc. axe du GAIN | conc. contrôle |
|---|---|---|---|---|---|
| 000d5950 | 33 | 210 | **177** | **97 %** | 18 % |
| 01e1f945 | 143 | 491 | **348** | **97 %** | 24 % |
| 00502e52 | 48 | 218 | **170** | **99 %** | 35 % |
| 0014603f | 102 | 256 | **154** | **100 %** | 39 % |
| 00761d27 | 148 | 639 | **491** | **99 %** | 20 % |

Le pic est **net à +2 exactement** : à +1 et +3 la concentration retombe au bruit (14-36 %),
signature d'une vraie frontière de champ. Verdict `TENU` sur les 5 témoins. Couverture : de 33
→ 210 (×6,4) et 143 → 491 (×3,4). Le plafond 19 % **tombe pour le cas modal**.

Note : le sous-ensemble déjà couvert par `fire_events` sature l'axe z (76-88 %), le sous-ensemble
gagné sature l'axe x (97-100 %) — deux sous-types de record (drapeaux 111/112 différents), même
physique (un axe horizontal). L'oracle ne juge que « directionnel vs uniforme » : les deux passent
très largement.

## Portée produit — CÂBLÉE EN PRODUCTION le 2026-08-31

`replay/shots.go` pose le cap du tir (`AimHeadingDeg` → `Shot.H`) quand `FireEvent.HasAim`. Le
chemin modal est désormais câblé :

- **`filmdec/fire_aim_modal.go`** (nouveau) : `modalAimBit` porte la grammaire Ghidra avec la
  polarité du champ d CÂBLÉE EN DUR et SANS les composites (parasites dans le chemin modal). Rend
  la position post-comptes+2 pour un record modal, `ok=false` sinon.
- **`filmdec/fire_events.go decodeFireEvent`** : chemin fixe @113 inchangé (ancre, zéro
  régression) + extension modale sous garde `!e.HasAim` (strictement additive). Lecture centralisée
  `readAimAt` partagée par les deux chemins.
- **Non-régression prouvée** : sur les records vides, post-comptes = 111 → visée à 113 = mêmes bits.
  Le diff du golden d'assemblage se réduit à UNE ligne (le compteur de cap), tout le reste inchangé.
- **Gain mesuré** : FireEvent avec `HasAim` (12 chunks témoins) 33→210, 143→491, 48→218
  (×6,4 / ×3,4 / ×4,5) ; film complet `000d5950` — tirs publiés avec un cap **90 → 401** (×4,5).

## Ce qui reste

- **Cas non-modal** (tir causant ≥ 1 dégât/cible) : la réserve « boucles de largeur runtime »
  est **RÉFUTÉE** — `NOTE_VISEE_NONMODALE_2026-08-31.md` a percé la grammaire des deux boucles
  ET des composites, ENTIÈREMENT hors ligne (param_5=0 → R(32) ; `FUN_140c1e924 = 3×W`, W d'une
  table PURE). Un décodeur non-modal peut avancer jusqu'à la visée. MAIS sa CORRECTION n'est pas
  validable par l'oracle de concentration (les tirs qui touchent visent à des élévations variées
  → vecteur ~uniforme, indiscernable du bruit ; mesure : non-modal 28-34 % vs modal 77-87 %).
  Non branché en prod. Piste : un **oracle de profondeur de trame** validerait la POSITION
  indépendamment de l'orientation (comme `TestLot1ViseeCalibration` pour le modal).
- **Départager attaquant/blessé** dans `damage_aftermath` : **RÉSOLU** — ref0 = blessé, ref1 =
  responsable (la vitalité de ref0 baisse 90-95 %, commit `5414739e4`). **Monde chronologique** :
  **INUTILE** (0 à +2 pts ; le plafond 82-89 % est l'ambiguïté de la réf, pas le temps).
- **Gate visuel** : le cap des tirs posé sur la carte du rejeu 2D — à la main de l'utilisateur.
