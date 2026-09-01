# Note — Relier un projectile a son tireur PAR UN CHAMP (owner/instigateur) — 2026-09-01

Instrument : `internal/analysis/filmdec/projectile_owner_research_test.go`
(+ `projectile_owner_helpers_test.go`). Garde `LOT1_TRAME_FILM`, borne 12 chunks, verrou
process, lecture seule. Films : `000d5950` (Fiesta), `01e1f945`, `00502e52`.

## Question

Le jeu SAIT qui a blesse sans tuer (precision par arme + assistance aux armes lourdes) : donc
l'instigateur d'un degat de projectile est trace PAR DEGAT dans la simulation. Ce lien
existe-t-il comme CHAMP dans le film (pour attribuer les touches explosives NON FATALES a leur
tireur sans jointure temporelle), ou seulement au runtime ?

## Verdict

**Le lien de champ projectile->tireur existe, mais UNIQUEMENT a la mort (dead-state tueur).
Le projectile VIVANT ne replique aucun owner dans le film.** Les touches explosives non
fatales ne sont donc PAS attribuables par un lien de champ ; la jointure temporelle
tir<->impact (note `NOTE_TOUCHES_EXPLOSIVES`, 58-100 % selon le mode) reste la seule voie.

## Ce que l'exe dit (Ghidra, `HaloInfinite.exe`, image base 140000000)

L'objet porte bien un **DamageOwner** — c'est le champ instigateur que le raisonnement
predisait. Chaines et handlers script (`hsc*`) :

| Chaine | Adresse | Sens |
|---|---|---|
| `Object_SetDamageOwnerObject` / `...Player` | 143c2c6e0 / 700 | POSER l'instigateur d'un objet (projectile) |
| `Object_GetDamageOwnerObject` / `...Player` | 143c2c720 / 740 | LIRE l'instigateur |
| `Unit_GetReceivedDamage_WeakDamageOwnerObject` / `...Player` | 143c58f78 / e10 | l'instigateur du dernier degat RECU par une unite |
| `Equipment_GetOwnerUnit` | 143ca33f8 | l'unite proprietaire d'un equipement |
| `Object_SetPreserveDamageOwner` | 1436e1f90 | conserver l'owner apres transfert |

Le handler `Object_GetDamageOwnerPlayer` (registration `FUN_141d16e20`) route vers
`FUN_1407b905c`, qui resout l'owner-joueur au RUNTIME depuis les donnees de degat recu de
l'objet (appel virtuel indirect), PAS depuis un composant replique. Le DamageOwner est donc
un etat de simulation, pas un champ de trame.

**Le dead-state EST ce DamageOwner materialise a la mort.** Le corps lourd du dead-state
(`FUN_140c1dd44`, composant i11 de la victime) porte `EnumB` (+0x08) =
**killer-absolute-participant-index** (confirme `traverse.go:1276` et le verdict R3 du
2026-06-07 : +0x04 victime, +0x08 tueur, resolus via `FUN_1407f2058`->`FUN_140e958c4`). C'est
le champ-lien deja prouve a 97,6 % pour les KILLS. Confiance : haute sur l'existence du
DamageOwner et sur EnumB=tueur ; le point-cle est qu'il n'est ecrit dans le film qu'a la mort.

## Ce que le film dit (mesure, 3 films, 12 chunks)

### M1b — censure ROBUSTE des masques ti=41 (le denominateur honnete)

Balayage de TOUS les records de projectile par le scanner dedie (`matchWorldObjectRecord`,
independant du rendement des trames propres), ancre sur i0 (position, presente sur 100 % des
records) :

| film | records ti=41 | i9 (multiplayer-properties) | i10 (parent-state) |
|---|---|---|---|
| 000d5950 | 6132 | 5 (0,1 %) | **0 (0,0 %)** |
| 00502e52 | 2824 | 0 (0,0 %) | **3 (0,1 %)** |
| 01e1f945 | 2747 | 0 (0,0 %) | **0 (0,0 %)** |

Le projectile en vol ne replique QUE du cinematique : i0 position, i1 velocite (~88 %),
i2 forward-up (~78 %), i3 angulaire, plus occasionnellement i18 at-rest / i19 tether /
i20 command-tick. Les deux seuls composants susceptibles de porter une reference d'entite —
i10 `object-parent-state` et i9 `object-multiplayer-properties` — sont **quasi absents
(<= 0,1 %)**. L'owner n'est pas dans le flux replique du projectile.

### M1 — le champ candidat i10 (parent-state), decode

`object-parent-state` (i10, `FUN_140c1e4d0`) etait le meilleur candidat : sa branche LIBRE
(objet non attache, en vol) lit derriere une porte un identifiant a largeur variable
(`FUN_1408f0ac4`->`FUN_1406d3140` = index R(13), MEME espace de handle dom1 que les bipedes).
On a publie cette valeur (`ObjectParentState.FreeID`, avant jetee) pour la tester. Resultat :
i10 n'apparait presque jamais sur ti=41 (cf. M1b), et les 2-3 ids captures (5022 ; 197 ;
6688) sont ETALES (pas <= roster) et ne resolvent a AUCUN bipede sur le jeu de bases.
`word16` de la branche attachee est de forte cardinalite (transform relatif, pas un handle).
i10 est un transform d'attachement physique, pas une reference d'owner — et de toute facon
un projectile n'est pas parente.

### M2 — ancre de verite terrain (dead-state tueur)

Sur les memes chunks, les dead-states de bipede mort donnent EnumB (tueur) de FAIBLE
cardinalite : 000d5950 5 morts / 2 tueurs (0:4, 9:1) ; 00502e52 1/1 (0) ; 01e1f945 4/1 (0).
Le champ-lien projectile->tireur existe donc bien — mais QU'A LA MORT.

### M3 — pont degat->projectile impossible en non fatal

Un damage_aftermath explosif a ref1 = le PROJECTILE (dom1, non-bipede). Sur les 3 films,
**0/50, 0/13, 0/9** de ces degats resolvent ref1 a un slot ti=41 encore VIVANT a l'instant du
degat : le projectile est transitoire (detruit a la detonation). Meme l'ENTITE projectile
n'est donc pas atteignable de facon fiable depuis un degat non fatal — a fortiori son owner.

## Validation vs verite terrain (le point demande)

Le taux « owner-projectile == tueur-deadstate » ne peut PAS etre mesure : il n'existe aucun
owner-projectile a decoder (i10/i9 absents, M1b). Le test contre la verite terrain est donc
NEGATIF par construction, et proprement : le seul champ qui porte le tueur (EnumB) n'est ecrit
qu'a la mort (M2), et le projectile vivant ne le replique jamais (M1b). C'est le resultat que
le garde-fou de la consigne anticipait.

## Reserves honnetes

1. M1b ancre sur i0-present : un hypothetique record parent-state-SANS-position serait rate.
   Mais i0 est sur 100 % des records ancres, i10 sur 0/6132 (Fiesta) : le negatif est robuste,
   et la semantique de parent-state (transform matrice+velocite) n'est pas celle d'un owner.
2. Le record de CREATION (NEW/default-state) de ti=41 n'a pas ete re-teste ici ; il est couvert
   par reference : l'`equipment-creator-component` du default-state d'objet du monde (ti=37,
   `equipment_creation_owner_test.go`) est CLOS (274/274 sans reference de createur ;
   `entity-ref-index5` porte fermee) — le default-state d'objet du monde ne transmet pas le
   poseur. Meme structure pour ti=41. [~]
3. Le DamageOwner runtime pourrait etre reconstruit hors film par la jointure de vol
   (`ScanFilmProjectiles` relie deja naissance<->tireur a 0,77 u, 70/70 sur les grenades) : ce
   serait une attribution PROBABILISTE, jamais un lien de champ.

## Edits de production (dans filmdec, sans changement de bits lus)

- `readVarWidthInt` rend desormais l'index R(13) (les appelants qui l'ignorent le jettent).
- `consume1408f0ac4[Probe]` rend `(present, id)` ; seul `consumeObjectParentState` les garde.
- `ObjectParentState` publie `HasFreeID`/`FreeID` (l'id de la branche libre, avant jete).

Aucun de ces changements ne modifie les bits consommes (tests composants/hooks verts).

## Reproduire

```
export GOCACHE=".../.gocache-owner" ; export CGO_ENABLED=0
export LOT1_TRAME_FILM=".../data/cache/film_chunks/000d5950"
go test ./internal/analysis/filmdec/ -run '^TestProjectileOwner$' -count=1 -v -timeout 30m
```
