# Note — Les touches explosives dans le film Theater (2026-09-01)

Instrument : `internal/analysis/filmdec/explo_touches_research_test.go`
(+ `explo_touches_helpers_test.go`). Garde `LOT1_TRAME_FILM`, borné 12 chunks,
verrou process, lecture seule. Films : `000d5950`, `01e1f945`, `00502e52`.

## Verdict

**Les touches explosives NON FATALES sont dans le film — l'API n'est pas contredite.**
Le verdict précédent (« armes lourdes = 0 % de touches en type 0 ») était un artefact de
notre appariement, qui exige que le RESPONSABLE du dégât (`damage_aftermath` 0xC0 type 0,
réf1, domaine 1) résolve au SLOT BIPÈDE du tireur. Pour un explosif, le dégât est infligé
par le PROJECTILE : la réf1 pointe une entité qui n'est jamais un joueur. Notre appariement
la ratait ; le dégât, lui, EXISTE dans le flux.

## Ce que porte damage_aftermath (rappel, grammaire éprouvée)

En-tête : réf0 (dom1, victime/blessé), réf1 (dom1, responsable), réf2 (dom7). Charge :
un tag SOURCE 32 bits (effet jpt!, même espace que le dead-state), une magnitude R(5)
déquantifiée sur [0,16]. Le tireur n'est PAS un champ : pour un tir direct il est réf1
(le biped attaquant) ; pour un explosif réf1 est le projectile.

## Les 3 classes de responsabilité (M1), par film

| film | total 0xC0 t0 | réf1 ABSENTE | réf1 → BIPÈDE (direct) | réf1 NON-BIPÈDE (candidat projectile) |
|---|---|---|---|---|
| 000d5950 (Fiesta, 77 tirs lourds) | 190 | 62 (33 %) · mag 0.27 | 76 (40 %) · mag 1.55 | **52 (27 %) · mag 3.02** |
| 00502e52 (61 tirs lourds) | 150 | 0 | 134 (89 %) · mag 1.38 | **16 (11 %) · mag 3.03** |
| 01e1f945 (12 tirs lourds seult.) | 44 | 7 (16 %) · mag 1.56 | 28 (64 %) · mag 1.77 | 9 (20 %) · mag 0.53 |

Sur les deux films à activité explosive réelle, la classe NON-BIPÈDE a :
- une **victime réf0 résolue à ~98 %** (ce sont de vraies touches sur de vrais joueurs) ;
- une **magnitude ~2x le tir direct** (3.0 vs 1.4-1.5) ;
- une **réf1 qui ne lie à AUCUN archétype en fin de chunk** (39/52 et 12/16) — signature
  d'une entité transitoire détruite après détonation.

Sur `01e1f945` (quasi pas d'explosif joué), la petite classe non-bipède est de FAIBLE
magnitude (0.53) : « réf1 non-bipède » est un marqueur NÉCESSAIRE mais pas suffisant ; il
faut le croiser avec la magnitude (M5) et/ou la coïncidence de tir (M3/M4).

## La magnitude est le discriminant propre (M5, densité-indépendant)

| film | direct : moy / part ≥2.5 | non-bipède : moy / part ≥2.5 |
|---|---|---|
| 000d5950 | 1.55 / 17.1 % | **3.02 / 28.8 %** |
| 00502e52 | 1.38 / 18.7 % | **3.03 / 37.5 %** |

La moyenne ~2x est robuste sur les deux films actifs. (Le seuil « part ≥2.5 » est
juste-en-dessous de 2x sur Fiesta car ce mode a beaucoup d'armes directes de magnitude
moyenne ; la moyenne, elle, est nette.) Le flux de dégât ne contient QUE du firearm
(mêlée/grenades y ont zéro occurrence, cf. project_killweapon_deadstate_solved) : une
touche firearm de forte magnitude à responsable non-joueur = projectile explosif.

## Le confond « tireur mort en cours de chunk » est écarté (M6)

Une réf1 non-bipède EN FIN DE CHUNK pourrait être un tir DIRECT dont le tireur est mort
depuis (délié du monde — piège de `lot1_monde_chrono`). On résout réf1 aux deux instants
(chronologique à l'événement, et fin de chunk) :

| film | CONFOND (chrono BIPÈDE, mort depuis) | PROJECTILE authentique (jamais joueur) |
|---|---|---|
| 000d5950 | 2/52 (3.8 %) | **50/52 (96.2 %) · mag 2.92** |
| 00502e52 | 3/16 (18.8 %) | **13/16 (81.2 %) · mag 2.87** |
| 01e1f945 | 0/9 | 9/9 · mag 0.53 |

**81-96 % de la classe non-bipède ne sont JAMAIS un joueur, aux deux instants, et portent
la magnitude explosive.** Le confond est marginal. La classe non-bipède est bien une
population de dégâts distincte, sourcée par projectile.

## Attribuer une touche explosive non fatale à son tireur + arme

Le mécanisme killsource répond DÉJÀ pour les KILLS : le dead-state i11 de la victime porte
`+0x00` = tag source (effet) et `+0x08` = TUEUR (joueur, dom5) — 97,6 %, y compris 8 morts
où le kill-feed du jeu créditait le mauvais joueur (roquette tirée trop près). C'est la
preuve que le film SAIT relier un dégât explosif à son tireur, au moins à la mort.

Pour le NON FATAL, le dégât ne nomme pas le tireur (réf1 = projectile). On le récupère par
**jointure temporelle avec le TIR** (0xD2 type 36, qui porte l'attaquant réf0 dom1 + le
WeaponID 64 bits + le FilmIndex du joueur) dans la fenêtre de vol :

- M4 (fenêtre de vol 2 s) : sur les touches non-bipèdes coïncidant un tir lourd, le tireur
  est UNIQUE dans **100 % (000d5950)** et **58 % (00502e52)** des cas ; l'arme unique idem.
  Un tireur unique = attribution non ambiguë (tireur+arme = ceux du seul tir lourd de la
  fenêtre). L'ambiguïté (42 % sur 00502e52) = plusieurs tireurs lourds simultanés.
- L'ARME vient du TIR (WeaponID), PAS du tag source du dégât : ce tag est un effet jpt! qui
  ne joint pas le WeaponID sans table (réfuté, cf. lot1_sonde_precision / lot1_attrib).

Recette d'attribution proposée (non implémentée en prod ici) :
`touche_explosive := damage_aftermath{ réf1 non-bipède, magnitude élevée, victime réf0
résolue }` ; tireur+arme := l'unique tir lourd (0xD2 t36) dont `ts_tir <= ts_impact <=
ts_tir + vol`. Sinon (plusieurs candidats), désambiguïser par la trajectoire du projectile
(événements 0xC2 detonate / 0xC3 impact_effect) — non fait ici.

## Réserves honnêtes / pistes non épuisées

1. La coïncidence temporelle brute (M3) est CONFONDUE par la densité des tirs lourds sur
   Fiesta (témoin 44 % ; ratio 1,39x, sous le seuil 1,5x). Sur `00502e52` (moins dense)
   elle TIENT nettement (75 % contre témoin 25 %, 3x). Le discriminant fiable reste la
   magnitude (M5) + la non-résolution chrono (M6), densité-indépendants.
2. La jointure tir↔impact est une ATTRIBUTION PROBABILISTE (fenêtre de vol), pas un champ.
   Le lien de champ n'existe que pour les kills (dead-state). Un lien de champ non fatal
   demanderait de résoudre le projectile (réf1, handle dom1) vers son PROPRIÉTAIRE : les
   événements projectile 0xC2/0xC3 portent une réf0 dom5 dont le sens (owner vs entité
   projectile) est resté non tranché (cf. lot1_projectiles, M3 non concluant). Piste vivante.
3. Le tag source ne suffit pas seul à isoler l'explosif (peu de tags exclusifs à la classe
   non-bipède, M2) : il faut magnitude + non-résolution.

## Reproduire

```
export GOCACHE=".../.gocache-explo"
export LOT1_TRAME_FILM=".../data/cache/film_chunks/00502e52"
go test ./internal/analysis/filmdec/ -run TestExploTouches -count=1 -v
```
