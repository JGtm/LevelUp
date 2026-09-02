# Attribution des touches explosives par la PISTE DE DETONATION (2026-09-01)

Chantier trame film / worktree `wt/trame-deto`. Instrument :
`internal/analysis/filmdec/deto_attribution_{research,helpers}_test.go`
(garde `LOT1_TRAME_FILM`, borne 16 chunks, verrou process, lecture seule). Cache Go prive.

## Thèse (utilisateur)

Le degat explosif est du SPLASH : la victime est HORS de l'axe de visee du tireur
(geo_explosifs mesure ~57 deg, angle de coincidence au hasard). On NE PEUT DONC PAS attribuer
victime -> tireur par l'alignement. MAIS le POINT DE DETONATION, lui, est SUR l'axe : le tireur
a vise LA ou ca explose. On decode donc le point de detonation, on l'attribue au tireur par
alignement (qui redevient discriminant), et on rattache les touches splash proches.

## Source du point de detonation, et sa fiabilite

Il n'existe pas d'evenement film portant une position d'impact exploitable : `projectile_detonate`
(0xC2 t5) et `projectile_impact_effect` (0xC3 t6/7) ne portent que ref0 + une `variant-name` a
~100 % distincte (cf. `lot1_projectiles_research_test.go`), pas de position deroulee.

La source FIABLE est la **derniere position repliquee de l'entite projectile (ti=41)** avant la
fin de sa vie (`ScanFilmProjectiles` / `scanProjectileRecords` + `splitLives`). Reserve ecrite
AVANT (deja dans `projectiles.go`) : pour une GRENADE ou un tir en CLOCHE la replication cesse
avant la meche, donc le dernier point n'est pas l'explosion ; pour un projectile DIRECT
(roquette / empaleur / mangler / choc) qui detone a l'impact, le dernier point EST le point
d'impact. On mesure donc separement les armes directes (`geoIsDirect`).

**Piege corrige** : les largeurs d'axe des objets du monde (ti=41) sont celles de la CARTE, pas
le defaut `13/13/14` (arene Cliffhanger). Le scan borne installe donc `WorldObjectPrecision` par
carte (`SetWorldObjectPrecisionFromLayout`, verrou tenu, valeur restauree). Sans ce correctif, un
projectile de carte a signature differente (Forge `[15,15,17]`) se decode a la mauvaise echelle :
la naissance tombait a **241 u** du tireur sur le BTB — apres correctif, **1,66 u**.

## Fiabilite de la naissance (oracle, M0)

Appariement detonation -> tir lourd par la NAISSANCE (le lien 70/70 de `projectiles.go`) : le tir
lourd dont l'instant encadre la naissance et dont le tireur est le plus proche du point de
naissance (< 3 u). Sert d'oracle a l'attribution geometrique.

| Film | detonations | appariees naissance | dist naissance<->tireur | temoin (autre deton.) |
|---|---|---|---|---|
| 000d5950 (arene Cliffhanger) | 322 | 56 (17,4 %) · 33 directes | **mediane 1,36 u** | 18,40 u |
| 4f77afc1 (BTB Forge) | 268 | 35 (13,1 %) · 18 directes | **mediane 1,73 u** | 41,81 u |

La naissance est bien celle du projectile de CE tir lourd (petite distance vs grand temoin).

## Alignement detonation vs victime vs temoin (M1 — le coeur)

Angle de la visee du tireur (i21, cap+elevation) vers le point de detonation, pour les armes
DIRECTES a tireur connu. Contraste avec un temoin (detonation d'un AUTRE tir, ~hasard).

| Film / arme | n | mediane visee -> DETONATION | temoin |
|---|---|---|---|
| 000d5950 · **M41 SPNKr** | 15 | **6,3 deg** (11/15 < 15, 0 > 30) | 85 deg |
| 000d5950 · Stalker Rifle (HITSCAN) | 17 | 109,6 deg | — |
| 4f77afc1 · **Fuel Rod SPNKr** | 8 | **4,3 deg** (tous < 15) | 61 deg |
| 4f77afc1 · **M41 SPNKr** | 7 | **8,4 deg** (5/7 < 15) | 61 deg |

**La thèse est confirmee pour les vrais lanceurs** : le point de detonation d'une roquette tombe
sur l'axe de visee du tireur (mediane 4-8 deg), alors que la victime splash est hors axe (~57 deg,
geo_explosifs) et qu'une detonation aleatoire est a ~60-85 deg. La ventilation par arme est
indispensable : le Stalker Rifle (HITSCAN, sans projectile ti=41) est faussement apparie a une
grenade voisine — son alignement 109,6 deg le rejette proprement, et sa vitesse calibree absurde
(2,7 u/s vs 14,9 pour le SPNKr) le trahit aussi. **Le Bulldog (fusil a pompe hitscan) est de meme
mal classe DIRECT** : traiter la liste `geoIsDirect` comme un sur-ensemble, pas une verite.

## Vitesses calibrees (M2)

Sur les detonations directes appariees a leur tir par la naissance,
`vitesse = dist(tireur, detonation) / (T_deto - T_tir)` :

- **M41 SPNKr** : mediane **14,9 u/s** (min 13,4 max 19,2) sur l'arene ; **16,2** (min 4,1 max 21,3)
  sur le BTB — coherent.
- **Fuel Rod SPNKr** (arc) : mediane **15,2 u/s** sur le BTB.
- Stalker Rifle : 2,7 u/s — aberration, confirme que ce n'est pas un projectile.

## Detonation -> tireur (M3 — discrimination)

Pour chaque detonation, score = alignement(visee -> detonation) + ecart de temps de vol ; gagnant
== tireur-naissance (verite). Compare au temporel-recent et a un temoin decale (+3 s).

| Film | evaluables | GEOMETRIE | TEMPOREL-recent | TEMOIN decale | ambigu (>=2 tireurs) : geometrie |
|---|---|---|---|---|---|
| 000d5950 (arene) | 55 | **98,2 %** | 92,7 % | 18,2 % | 10 : 90,0 % |
| 4f77afc1 (BTB) | 29 | **82,8 %** | 75,9 % | **6,9 %** | 14 : 71,4 % |

L'ecart GEOMETRIE - TEMOIN (98 vs 18 ; 83 vs 7) mesure le pouvoir discriminant reel de
l'alignement sur la detonation. **Le gain propre de la geometrie sur le temporel est plus net en
BTB** (sous-ensemble ambigu : 14/29 des detonations ont >= 2 tireurs lourds en vol, geometrie
juste 71,4 %) — exactement la ou plusieurs projectiles volent en meme temps, ce que le
temporel-unique ne sait pas trancher. Unicite : 42 (arene) / 33 (BTB) detonations n'ont qu'UN
seul tir aligne < 15 deg (attribution non ambigue par la seule geometrie).

## Touches splash rattachees a la detonation (M4 — reserve honnête)

Rattachement d'une touche non-bipede (`geoTouch`, victime resolue en position) a une detonation
par proximite spatiale (rayon 6 u) + temporelle, avec sweep du decalage.

- **Arene 000d5950** : au mieux **18,7 % de couverture** a un decalage de **-1 s** (unicite 100 %),
  contre 1-3 % aux decalages voisins. Le pic a decalage negatif quantifie le trou
  « derniere position repliquee <-> impact reel » (la replication precede la detonation), mais la
  couverture reste faible : **le rattachement spatial des touches splash n'est PAS fiable** avec
  le point de detonation actuel.
- **BTB 4f77afc1** : 0 touche exploitable — sur Forge, les references de degat (`ref0`/`ref1`,
  base 512) ne resolvent pas leur victime (0/19), donc pas de position de touche a rattacher.

Conclusion M4 : la voie fiable est M3 (detonation -> tireur par alignement pour les armes
directes), PAS le rattachement spatial des touches splash.

## Couverture arene vs BTB — bilan

- **Arene (000d5950)** et **BTB (4f77afc1)** : methode VALIDEE et auto-controlee (SPNKr sur axe,
  M3 geometrie >> temporel >> temoin). Le BTB est le cas ou la geometrie apporte le plus.
- **00502e52 (bazaar)** : aucun lanceur direct tire dans les 16 chunks bornes (0 apparie) —
  inconclusif par absence de donnee, pas une refutation.
- **01e1f945 (catalyst/deadlock)** : la base bipede ne resout que 29 % des tireurs sur ce film
  (probleme de detection de base propre au film) — linkage casse, inconclusif.

## Reserves

1. Le point de detonation n'est fiable que pour les armes a projectile DIRECT (roquette,
   empaleur, mangler, choc). Grenades et tirs en cloche (Fuel Rod, Ravager, Hydra en verrouillage)
   voient leur replication cesser avant la meche : le dernier point est mid-flight, pas l'impact.
2. `geoIsDirect` sur-classe : Stalker Rifle et Bulldog sont HITSCAN (pas de ti=41) et doivent etre
   ecartes par la ventilation par arme + la vitesse calibree, pas pris pour argent comptant.
3. Le rattachement des touches SPLASH a la detonation par geometrie spatiale est refute (couverture
   < 20 %, arene seule). Une voie NON exploree : lier la ref1 non-bipede d'une touche au SLOT du
   projectile ti=41 directement (lien exact plutot que geometrique) — bloquee ici car la base des
   references de degat sur Forge ne resout pas (0/19 victimes).
4. `WorldObjectPrecision` doit etre installee par carte pour tout scan projectile hors
   `BuildFromFilm` : sinon toute carte non `13/13/14` est mal decodee (piege du 241 u sur Forge).

## Gate

- `gofmt -l` : propre. `go vet ./internal/analysis/filmdec/` : vert (GOCACHE prive).
- `deto_attribution_research_test.go` 404 L · `deto_attribution_helpers_test.go` 418 L (< 500).
- Films : 000d5950 (defaut), 4f77afc1 (`LOT1_SONDE_MAP="flood gulch"`), 00502e52, 01e1f945.
