# HANDOFF — LE DÉSAMORÇAGE DE LA BOMBE D'ASSAUT (2026-09-04)

> Écrit au moment où le chantier `.ai/V7.5/PLAN_ASSAUT_STATS_2026-09-04.md` met le désamorçage
> HORS LOT (décision 5). Ce document ne demande rien : il passe la main. Il dit ce qui est
> mesuré, ce qui est réfuté avec son témoin, ce qui est déduit, et à quelle condition exacte le
> sujet redevient traitable.
>
> **Mission de rédaction : lecture seule.** Aucun film décodé, aucun test lancé, aucune base
> ouverte, aucun fichier de production touché. Tous les chiffres ci-dessous sont relus dans les
> journaux et les sources du dépôt, et chacun porte sa provenance. Les quelques arithmétiques
> faites ici (délais, décomptes) sont signalées comme telles.

## AMENDEMENT DU 2026-09-04, APRÈS L'ÉTAPE E2-ter — À LIRE AVANT LE RESTE

Ce handoff a été écrit avant que l'étape E2-ter du chantier ne lève la garde One Bomb. Elle est
close et verte, et elle déplace trois choses. **Le corps du document reste valable ; ce qui
suit le corrige là où il est daté.**

**(1) La détection d'une TENUE DE DÉSAMORÇAGE est en PRODUCTION.** Ce n'était plus qu'un
instrument de recherche quand ce document a été écrit ; c'est maintenant
`filmdec/navpoint_radial_segments.go` — `NavpointSegments()`, `EndsAtSummit()` et surtout
**`IsDisarmHold()`**, avec ses seuils nommés (`NavpointPauseMaxSlopeQS = 60`,
`NavpointSummitToleranceQ = 4`). La copie qui vivait dans les tests a été SUPPRIMÉE, pas
dupliquée : les instruments délèguent désormais à la production. **La marche 0 du §6.1 se
reformule en conséquence** : il ne s'agit plus d'écrire un balayage, mais d'interroger un
prédicat qui existe.

**(2) One Bomb n'est plus une zone aveugle.** La garde par NOM DE VARIANTE
(`replaybuild.isArmableBombVariant`) est SUPPRIMÉE, avec ratchet
(`replaybuild.TestAucuneGardeParNomDeVariante` : aucun fichier de production ne peut contenir
le littéral « one bomb »). La variante où les descentes de désamorçage ont été observées est
donc désormais balayée par le chemin normal. La GARDE 2 (confrontation locale tout-ou-rien)
reste, et elle a même gagné une seconde branche (dispersion, CV <= 0,20).

**(3) La mèche est MESURÉE PAR FILM**, elle n'est plus une constante : `BombArming.FuseMS`
porte 4 987 ms (Neutral), 5 089 ms (Husky) et **16 183 ms (One Bomb, CV 0,010)** — sortis de la
même règle, sans aucun branchement sur un nom. Toute lecture de ce document qui suppose une
mèche unique est à relire avec ça en tête.

### Ce que cela change au décompte du §4 (les CINQ montées pleines sans explosion)

| candidat | statut au 2026-09-04, après E2-ter |
|---|---|
| `9f57c612` @388 080 | **RÉSOLU — c'est un ARMEMENT**, publié parmi les 5 que ce film date désormais (65 137 · 279 103 · 335 193 · 388 080 · 445 839 ms). Un candidat de moins. |
| `c75f33b8` @196 605 | NON TRANCHÉ — le film est ENTIÈREMENT retenu par la garde 2. |
| `df8fcbef` @51 901 / @168 619 / @527 367 | NON TRANCHÉ — même raison. |

**Et ce qui retient ces deux films est déjà connu, ce qui est une piste et non un mur** :
`c75f33b8` @395 724 (aucun armement) et `df8fcbef` @778 033 (délai de 27 845 ms contre les
~16 000 attendus) sont **exactement les deux entrées d'`a5SansPorteur`**, la partition mesurée
du 2026-08-31. Autrement dit : les deux seuls films encore muets le sont à cause des deux seules
explosions que personne n'explique — et une explosion sans porteur ni armement à la mèche est
précisément ce à quoi ressemblerait une partie où quelqu'un a désamorcé. **C'est là qu'il faut
regarder en premier**, et c'est une hypothèse, pas une mesure : rien ne l'établit encore.

Note de version : le document de rejeu est passé au **schéma 39**.

---

## 0. Les sources, et leur rang

| source | rang | où |
|---|---|---|
| Journaux de mesure du chantier (protocole écrit AVANT la mesure, sortie collée) | **PREMIÈRE** | `.ai/V7.5/replay2d/registre_film/B4_bombe_desamorcage.log`, `BV_onebomb_rejeu.log`, `A_PROTOCOLE.md` |
| Note moteur Ghidra + pool Lua du 2026-09-04 | **PREMIÈRE** (mesure de première main) | `.ai/V7.5/ETAT_ASSAUT_EVENEMENTS_MOTEUR.md` |
| Code de production et instruments de recherche | PREMIÈRE (pour ce qu'ils FONT) | `internal/analysis/replay/`, `internal/analysis/filmdec/` |
| Plan du chantier, section 5 | SECONDE (synthèse, et elle est ici amendée — §4) | `.ai/V7.5/PLAN_ASSAUT_STATS_2026-09-04.md` |

---

## 1. LA QUESTION OUVERTE, EN UNE PHRASE

**Peut-on dater un désamorçage de bombe dans un film Theater, et nommer le joueur qui l'a fait ?**

Elle vaut la peine pour trois raisons, dans cet ordre : (a) c'est le seul geste d'objectif de
l'Assaut qui manque encore — explosion, ramassage, portage, armement et porteurs tués sont
tous reconstruits et entrent en base ; (b) c'est le geste DÉFENSIF du mode, donc la seule
statistique d'Assaut qui parle du camp qui n'a pas la bombe ; (c) le moteur le NOMME et le
distingue à trois temps — l'obstacle n'est ni le vocabulaire ni l'outillage.

---

## 2. ACQUIS — ce qui est MESURÉ, avec ses dénominateurs

### 2.1 Le moteur distingue proprement l'abouti de l'interrompu — MESURE Ghidra + Lua

Source : `.ai/V7.5/ETAT_ASSAUT_EVENEMENTS_MOTEUR.md` §3.a, §4, §5 (mesure de première main sur
`HaloInfinite.exe`, 2026-09-04).

- Le parcel Lua `primitive_carriable_arming_base` (tag `25af9c45`, 12 559 octets) enregistre
  **six** événements : `OnInitializationStarted / Interrupted / Completed` (amorçage) et
  `OnDeactivationStarted / Interrupted / Completed` (désamorçage). Le triplet est nommé.
- `BombObjectState` a cinq membres, avec leurs valeurs en clair (`FUN_14034a0d0`) :
  `None`(0), `Unarmed`(1), `Armed`(2), **`Disarming`(3)**, `Contested`(4). Il n'y a pas d'état
  « Arming » : `Contested` EST l'amorçage en cours.
- Le script de mode (tag `a35c6ce9`) déclare **deux appareils séparés**, `ArmingDeviceTag` et
  `DefuseDeviceTag`, plus une boucle sonore propre `AssaultLoopDisarm`.
- L'état et la jauge remontent **jusqu'au HUD** : `CenterScoreboard_BombState` (`14382d300`) et
  `CenterScoreboard_BombNormalizedCaptureProgress` (`14382d160`). Un client n'affiche que ce
  qu'on lui réplique. Mais cette famille est **globale au match**, donc sans acteur.
- **Le Lua n'expose que l'ÉQUIPE.** Les seuls appels d'identité du parcel sont
  `Item_GetInventoryUnit -> Player_GetMultiplayerTeam -> Object_SetMultiplayerTeam`, et le champ
  propagé est `modeParcel.activatingTeam`. Il n'existe aucun `activatingPlayer`.
  *Statut : MESURE pour ce que le script nomme ; DÉDUIT pour l'absence* — le pool Lua déduplique
  ses chaînes et ne rend jamais un négatif (règle du dépôt, 2026-08-30).

### 2.2 Le corpus, et ses dénominateurs exacts

Source : `.ai/V7.5/replay2d/registre_film/A_PROTOCOLE.md` §1 (qualification du 2026-08-27) et
`filmdec/navpoint_ti12_verdict_test.go:23-33` (oracle figé des explosions).

**9 films d'Assaut, 8 admis** (`ce083875` exclu : pont bipède 19/180 = 10,6 %, sous le plancher
de 50 %) ; **28 explosions** au total.

| variante | films | explosions |
|---|---|---|
| Neutral Bomb | `35b75a31`, `69b16f5d`, `3d58eb37`, `ce083875` | 3 + 3 + 3 + 3 |
| Neutral Bomb Squad | `34bb3bc8` | 1 |
| Husky Raid | `1c01e34f` | 4 |
| One Bomb | `9f57c612`, `c75f33b8`, `df8fcbef` | 4 + 3 + 4 |

Deux régimes de mèche, mesurés séparément : **4,93 s fixe** sur Neutral Bomb (13/13, CV 0,016,
0/1000 tirages nuls) et Husky Raid (4/4, CV 0,016) ; **16,2 s PAUSABLE** sur One Bomb
(9/9 explosions portées, CV 0,017, 0/1000 — `BV_onebomb_rejeu.log`).

### 2.3 Ce que le canal de l'anneau montre DÉJÀ du désamorçage — et c'est le fait le plus utile

Source : `filmdec/navpoint_ti12_meche_test.go:1-40` (définitions figées avant mesure) et
`BV_onebomb_rejeu.log` (inspection structurelle du 2026-09-01).

Sur One Bomb, l'anneau `ti=12 i14` du site porte **trois figures distinctes, toutes mesurées** :

| figure | signature mesurée | ce que c'est |
|---|---|---|
| ARMEMENT | montée contiguë 131 -> 254, n=30, ~2,9 s, identique sur 10 occurrences | la pose |
| **TENUE DE DÉSARMEMENT** | **segment strictement descendant depuis ~251, pente 14 à 26 quanta/s, l'anneau REVIENT ensuite à 251** | **un défenseur tient l'appareil, puis relâche** |
| chute d'explosion | 251 -> 127 à ~138 quanta/s | la bombe saute |
| cycle RESET | 130 -> 253 -> 127 en 5,0 s (n=51), sur l'autre paire de slots | ré-apparition du marqueur |

Exemples relevés tels quels dans `BV_onebomb_rejeu.log` (film `9f57c612`, slots 1459/1471) :
`70,5s -> 72,0s q 251->213 n=16`, `81,5s -> 82,0s q 251->238 n=6`, `283,7s -> 285,8s q 251->218
n=21`, `286,6s -> 287,5s q 251->228 n=10`, `293,1s -> 293,3s q 251->246 n=3`.

**Conclusion, et elle est solide** : les tenues de désamorçage INTERROMPUES existent au corpus,
elles sont datées, et le vide de pente entre elles (14-26 q/s) et la chute d'explosion (138 q/s)
est large — le seuil `mpPenteMaxQS = 60` (`navpoint_ti12_meche_test.go:74`) est posé au milieu
de ce vide. L'anneau revient à 251 après chaque descente : **ce n'est donc pas un décompte de
mèche qui s'écoule, c'est une tenue qu'on relâche.** (Une mèche de 16,2 s qui viderait
linéairement 124 quanta descendrait à ~7,7 q/s, et ne remonterait jamais.)

Ce que la lecture en production fait de ces descentes : elle les appelle **PAUSES** et les
soustrait du délai (`mpDelai`), ce qui est exactement le comportement du jeu — un désamorçage en
cours suspend la mèche.

### 2.4 Ce qui est déjà en production, et qui servira de socle

- `bomb_armings.go` — l'armement daté par l'anneau, avec sa **garde de mode double** (§3.4).
- `bomb_arms.go` — l'armeur nommé par jointure : **le lâcher EST le geste de pose**, écart
  lâcher − armement mesuré `+247` à `+259 ms` sur 10 appariements de 4 films, étendue 12 ms.
  Couverture du gate : **9 armements attribués sur 13 (69 %)**, les 4 autres publiés SANS acteur.
  *Chiffre daté du gate E2* : l'étape E2-bis (repli « porteur actif à l'instant armé », arbitrée
  par l'utilisateur le 2026-09-04) était en cours d'écriture au moment de ce handoff et vise
  ~85 % — relire la couverture republiée par son gate avant de citer ce nombre.
- Le pont d'horloge FILM -> MATCH est écrit et testé :
  `horlogeMatch = horlogeFilm + premierPaquetDuFilmUS/1000 − deathOffsetMS`, **33 à 114 ms**
  mesurés sur les 5 films du gate, déjà calculé en production par `resolveOriginMs`
  (`origin.go`). `TestBombArmsRecalageHorloge` rougit si le recalage saute.
- Le canal des armes tenues nomme le porteur de la bombe (famille `0x3fee4fcf`,
  `held_object_carry.go`) : témoin Oddball 46/46, « bombe posée portée par personne » 27/28.

---

## 3. RÉFUTÉ AVEC TÉMOIN — à ne PAS rouvrir

Un négatif est un résultat. Chacun de ceux-ci est borné par son dénominateur et par son
plancher ; les rejouer avec de nouvelles aiguilles ne changera pas un canal muet en canal
parlant. **Pour chacun, la portée exacte sur le DÉSAMORÇAGE est dite — elle n'est pas toujours
totale, et l'étendre sans le justifier serait une faute.**

### 3.1 L'API 343 — FERMÉ, et le négatif vaut PLEINEMENT pour le désamorçage

Payloads `GetMatchStats` des 3 variantes, relevés le 2026-08-31 : le bundle `Stats` ne porte que
`CoreStats` et `PvpStats`, la chaîne « bomb » n'y apparaît nulle part (commentaire de
`ObjectiveTypeBomb`, `objectiveevents/named.go`). Il n'y a pas de `BombDefusals` à demander :
c'est le bundle entier qui est absent.

### 3.2 La famille `BombStats` du moteur — FERMÉ STRUCTURELLEMENT, et cela couvre `BombDefusals`

Les 9 statistiques (`BombDetonations`, **`BombDefusals`**, `BombPlants`, `BombCarriersKilled`,
`BombDefusersKilled`, `BombPickUps`, `BombReturns`, `KillsAsBombCarrier`, `TimeAsBombCarrier`,
adresses `14381e790`..`14381f1b8`) sont des champs Bond
`Microsoft.Halo.HaloStats.Bond.HaloInfinite.Match.Stats.BombStats` (`14373a2a0`), sérialisés par
`FUN_1434771b8` vers le service de télémétrie Microsoft. **Ce ne sont pas des composants
d'entité : le film ne peut pas les répliquer, par construction.** C'est la même cause unique que
le silence de l'API.

**Portée sur le désamorçage : TOTALE pour un COMPTEUR par joueur.** Chercher un compteur de
désamorçages répliqué dans le film est une impasse démontrée. Ce qui reste ouvert, c'est un
canal d'ÉTAT ou de PROGRESSION (§3.5), qui n'est pas de la même nature.

### 3.3 Le statborg contre l'ARMEMENT — négatif borné, mais sa portée sur le désamorçage est PARTIELLE

`objectiveevents/assaut_statborg_armement_test.go` (2026-09-04). Balayage : **928 unités
possibles** (58 composants × 4 canaux A/B/C/D × 2 familles de slot joueur|équipe × 2 genres
progression|transition), dont **133 porteuses**. Oracle : les **17 armements** des variantes à
mèche fixe (`explosion − 4 930 ms`, ± 600 ms). Plancher : **500 instants aléatoires par film**,
graine fixe `20260904`, soit 4 500 tirages. Critère : couverture pleine ET taux ≥ 3× le plancher.
**Verdict : 0 unité sur 133 tient les deux ; la meilleure sélectivité vaut 2,59× pour un seuil de 3×.**

**Ce qui se transporte au désamorçage** : l'ÉNUMÉRATION (les 928 unités ont été balayées, les
133 qui émettent sont connues) et la borne structurelle de §3.2.
**Ce qui NE se transporte PAS** : le VERDICT. Il a été rendu contre un oracle d'ARMEMENT ; un
canal qui ne réagirait qu'à un désamorçage n'émettrait rien à ces 17 instants et serait sorti
« non couvert » — il n'a jamais été jugé, et **il ne peut pas l'être tant qu'aucun instant de
désamorçage n'est connu.** C'est la circularité de tout ce sujet : l'instrument exige un oracle,
et l'oracle est précisément ce qui manque. La seule voie qui y échappe est la recherche **PAR
NOM** (§6, marche 3).

### 3.4 Le pied de film — énumération CLOSE, verdict d'armement non transposable tel quel

`objectiveevents/assaut_pied_armement_test.go` (2026-09-04). **2 490 blocs** sur les 9 films, et
l'octet 47 qu'on prenait pour un indice de type est le petit octet de la VALEUR de récompense
(u32 grand-boutien aux octets 44-47) : **2 490 sur 2 490**. L'énumération est close. Couverture
de la meilleure valeur (`+20`) : **24 % des armements pour un plancher de 22 %** — du bruit.

**Portée sur le désamorçage** : la clôture de l'énumération est TOTALE (le pied porte des
valeurs de récompense, pas des types d'événement — quel que soit le geste cherché). Le verdict
« l'armement n'est pas récompensé » ne se transporte PAS mécaniquement ; qu'un désamorçage ne
rapporte pas de points est une HYPOTHÈSE cohérente avec un mode scripté, pas une mesure.

### 3.5 Les autres canaux

| canal | verdict | portée sur le désamorçage |
|---|---|---|
| `ti=13` propriétés nommées | **FERMÉ EN ASSAUT** — 8 slots pour 12 à 51 noms distincts : bruit | totale (le canal est muet sur ces films, quel que soit l'événement) |
| `ti=11 i12/i13/i14`, voie delta | **RÉFUTÉ** — légalité 45,7 % SOUS le hasard ; densités ×0,97/×0,94/×1,14, p = 0,61/0,63/0,09 | totale. **`ti=11 i14 state` a déjà été proposé, publié et réfuté : ne pas le re-proposer comme piste neuve** |
| statborg composants 0-27 | **FERMÉ** (balayage A6, 112 canaux, C et D compris) | même réserve qu'en §3.3 |
| `murmur3("bombobjectstate")` dans `ti=13 i0` | négatif net, **mais sur une chaîne que le moteur ne hache JAMAIS** | le négatif tient ; la porte n'est pas fermée, la clé n'était pas la bonne |
| garde de mode One Bomb | `isArmableBombVariant` (`replaybuild/zones.go:165`) **exclut One Bomb par son nom** | conséquence : même une lecture juste sur One Bomb ne serait pas publiée tant que la garde 1 n'est pas levée (§7.4) |

---

## 4. CORRECTION SUR PIÈCES — le résidu n'est pas unique : il y en a CINQ

La décision 5 du plan écrit « il reste un seul résidu inexpliqué (`df8fcbef` à 527 s) ». **La
relecture des journaux en donne cinq.** Ce n'est pas une contradiction du négatif — c'est une
précision qui change le coût de la reprise, et il vaut mieux la lire ici que la redécouvrir.

**Ce qui est MESURÉ** (`B4_bombe_desamorcage.log`, `BV_onebomb_rejeu.log`) :

- Sur Neutral Bomb / Neutral Bomb Squad / Husky Raid : **D2 = 17/17 explosions couvertes** par
  une pose complète dans les 10 s, et **0 candidate D3**. Les 34 montées complètes de ces
  6 films sont exactement 17 armements × 2 miroirs (les navpoints vont par paires, écart de slot
  +12, déduplication à 500 ms — `bombPairToleranceMS`). **Aucune pose n'y est restée sans
  explosion : personne n'a désamorcé.** Le négatif est net et il tient.
- Sur les trois films One Bomb : **32 candidates D3 = 16 instants distincts × 2 miroirs**, pour
  **11 explosions**. Le journal de la mèche pausable nomme, explosion par explosion, l'armement
  qui la précède (`mpDelai` retient la DERNIÈRE fin d'armement avant la cible).

**Arithmétique faite ici** (soustractions sur des instants publiés — aucune mesure neuve) : les
11 explosions consomment 11 des 16 instants. **Cinq montées pleines ne sont l'armement d'aucune
explosion :**

| film | instant (horloge film) | explosion suivante | écart brut | ce que fait la bombe après (fenêtre 30 s) |
|---|---|---|---|---|
| `9f57c612` | 388 080 ms (6:28) | 469 057 | +80,98 s | +18,02 s PRISE par le slot 592 — aucun lâcher |
| `c75f33b8` | 196 605 ms (3:17) | 395 724 | +199,1 s | +29,25 s PRISE par le slot 547 — aucun lâcher |
| `df8fcbef` | 51 901 ms (0:52) | 255 767 | +203,9 s | rien dans les 30 s |
| `df8fcbef` | 168 619 ms (2:49) | 255 767 | +87,1 s | **+0,25 s LÂCHER par le slot 532** |
| `df8fcbef` | **527 367 ms (8:47)** | 778 033 | **+250,7 s** | **+0,25 s LÂCHER par le slot 610**, puis +25,41 s PRISE par le slot 620 |

Deux remarques, mesurées elles aussi :

1. **Le lâcher à +0,25 s est la signature de l'armement** — l'écart mesuré en production vaut
   +247 à +259 ms sur 10 appariements (`bomb_arms.go`). Les instants `168 619` et `527 367` la
   portent : ce sont donc des **armements aboutis dont la bombe n'a jamais explosé**. C'est
   exactement la forme qu'aurait un désamorçage réussi.
2. **Deux montées ne partent pas du repos** : `c75f33b8 @252 163` (q **165** -> 254) et
   `df8fcbef @527 367` (q **207** -> 254), là où les 14 autres partent de ~131. *DÉDUCTION, pas
   mesure* : l'anneau n'était pas au repos quand la tenue a commencé — figure attendue si une
   descente de désamorçage avait déjà entamé la course.

**Et le trou d'instrument, qui est le vrai enseignement de cette relecture** : le protocole
D1-D3 (`filmdec/bombe_desamorcage_research_test.go:26-37`) ne cherche que des **MONTÉES**
(« pose complète »). **Aucun instrument n'a jamais cherché une DESCENTE COMPLÈTE** — la
signature directe d'un désamorçage abouti (§2.3). `mpEstPause`
(`navpoint_ti12_meche_test.go:231`) accepte n'importe quelle descente sous 60 q/s sans
distinguer celles qui atteignent le plancher (127) de celles qui s'arrêtent en route (185..246).
Le corpus n'a donc pas été fouillé pour ce signal-là : **il n'a pas été trouvé, il n'a pas été
cherché.** Dire « aucune occurrence » sans cette nuance serait faux.

---

## 5. LE VERROU

**Il n'y a, au corpus, aucun désamorçage AVÉRÉ — et il n'existe aucun oracle capable d'en
avérer un.**

Sans détour :

1. Sur les 6 films où la lecture de l'anneau est PROUVÉE (Neutral, Husky), **17/17 des poses
   ont explosé** : il n'y a rien à trouver, quelle que soit la finesse de l'instrument.
2. Sur les 3 films One Bomb, il y a **cinq candidats sérieux** (§4) — mais c'est précisément la
   variante où la lecture simple de l'anneau a été RÉFUTÉE (CV 0,725, 87/1000 tirages nuls aussi
   bons), où la production ne publie rien (garde de mode, §3.5), et où la lecture pausable qui
   la sauve **n'est pas encore portée en production** (étape E2-ter du plan, cases non cochées
   au 2026-09-04).
3. **Aucun juge extérieur n'existe** : l'API ne publie rien (§3.1), le statborg ne peut pas
   porter le compteur (§3.2), le pied ne récompense rien d'identifiable (§3.4). Toute mesure de
   désamorçage se jugerait contre elle-même.
4. **L'utilisateur, interrogé le 2026-09-04, n'a aucun match à proposer.**

Le verrou n'est donc pas seulement « pas de corpus » : c'est **pas d'oracle**. Et les deux ne se
lèvent pas par les mêmes moyens (§6).

---

## 6. COMMENT OBTENIR UN CORPUS — OU S'EN PASSER

### 6.0 Contraintes NON NÉGOCIABLES, à relire avant de proposer quoi que ce soit

- **JAMAIS de cuisson d'artefacts en lot.** C'est une bombe RAM : la machine a été mise à genoux
  **quatre fois**. Le verrou `filmproc.AcquireSolo` est posé — **un film par processus**, et il
  n'existe aucune raison valable de le contourner.
- **Toute campagne de décodage se DEMANDE à l'utilisateur avant d'être lancée.** Sans accord
  explicite : rien.
- Toute mesure passe par la sentinelle mémoire (`filmproc.Arm`, plafond 2 Gio) et le verrou
  `LockProcessDecode` — comme tous les instruments de ce chantier.
- Le worktree dédié `LevelUp-wt-assaut-stats` **n'a pas de cache film** (`data/cache/film_chunks`
  y est vide). Le cache vit dans le worktree principal (**1 380 films en chunks, 1 383
  manifestes**, comptés le 2026-09-04). `ASSAUT_CACHE` doit donc pointer sur le cache du
  principal — sans jamais y écrire.
- Deux compilations Go concurrentes corrompent le cache de build : **une mesure à la fois**,
  jamais en parallèle d'un autre agent qui compile.

### 6.1 L'échelle des marches, de la moins chère à la plus chère

**Marche 0 — GRATUITE, et c'est par là qu'il faut commencer : les cinq candidats sont DÉJÀ au
corpus.** Aucun film à obtenir, aucun décodage neuf à demander : les instants sont publiés au
§4, les films sont en cache. Le premier travail n'est pas d'agrandir le corpus, c'est de
**regarder ce qu'il contient déjà** avec un instrument qui cherche la bonne figure (la descente
complète), et non celle qu'on cherchait jusqu'ici (la montée).
*Coût mesuré* : le protocole D1-D3 a balayé les **9 films en 170,4 s** avec un **pic mémoire de
0,02 Gio** (`B4_bombe_desamorcage.log`). Un balayage des descentes est du même ordre — c'est la
même lecture de l'anneau. Il reste une campagne de décodage : **elle se demande**.

**Marche 1 — Recenser le parc, en LECTURE SEULE, avant de parler de corpus.** L'instrument
existe et il ne décode aucun film : `go run ./apps/go-api/cmd/zone-attribution -census`
(`cmd/zone-attribution/census.go`) — il ouvre la base partagée par `duckdb.OpenReadForQuery`
(correct même si le serveur la tient en RW) et se contente de `os.Stat` sur les répertoires de
chunks. Il rend, par mode : matchs au registre / films en cache / artefacts cuits.

Ce qu'on sait DÉJÀ par le recensement figé au dépôt (`D1_recensement_modes.log`, passe du
2026-08-27, **1 940 matchs au registre**) :

| mode (`pair_name` normalisé) | matchs | films en cache | dans le corpus d'Assaut |
|---|---:|---:|---|
| Neutral Bomb | 4 | 4 | 4 (dont 1 exclu) |
| One Bomb | 3 | 3 | 3 |
| Neutral Bomb Squad | 1 | 1 | 1 |
| Husky Raid | 4 | 4 | **1** (`1c01e34f`) |
| Super Husky Raid | 11 | 11 | **0** |

**Le fait qui saute aux yeux** : la famille bombe stricte est ÉPUISÉE (8 films sur 8 déjà
étudiés), mais **14 films Husky / Super Husky Raid sont en cache et n'ont jamais été passés au
protocole d'Assaut.** Or `Husky Raid:Assault` est l'une des **4 formes de la famille bombe au
registre** (relevé du 2026-08-31, `objectiveevents/extract.go:143-151`), et le seul film Husky
du corpus (`1c01e34f`) porte 4 explosions de bombe.
*DÉDUIT, pas mesuré* : rien ne dit que les 14 autres sont des variantes d'Assaut — leur libellé
de mode est un `pair_name`, pas un `game_variant_name`. **C'est exactement ce que la marche 1
tranche, sans décoder un seul film.** La requête minimale, à passer en lecture seule :

```sql
-- lecture seule (OpenReadForQuery / ATTACH ... READ_ONLY) : ne jamais ouvrir en RW
SELECT game_variant_name, count(*) AS matchs
FROM match_registry
WHERE lower(game_variant_name) LIKE '%assault%' OR lower(game_variant_name) LIKE '%bomb%'
GROUP BY 1 ORDER BY 2 DESC;
```

**Marche 2 — Passer les films d'Assaut non étudiés au protocole (après la marche 1).** Coût :
une mesure par film, un film par processus, ~20 s et < 0,5 Gio par film aux ordres de grandeur
mesurés. Risque : nul pour la machine si le verrou est respecté ; le seul risque est de ne rien
trouver — ce qui reste un résultat, à condition d'en publier le dénominateur.

**Marche 3 — La recherche PAR NOM, qui échappe à la circularité de l'oracle.** Deux lectures,
recommandées par la note moteur §7, et **elles ne demandent AUCUN désamorçage au corpus** :
  a. **Le dictionnaire ECS du film** (`chunk_00`, en clair) : y chercher
     `bomb_icon_reader_component` (`143803848`). L'instrument existe
     (`replay/assaut_a9_interaction_test.go`, `TestAssautA9Dictionnaire`) et son motif `"bomb"`
     y était déjà — mais **son résultat n'est journalisé nulle part** (aucun log `A9_*` au
     registre du film). Coût : une lecture de `chunk_00`, aucun corps de record décodé.
  b. **Les composants 28-57 du statborg** (jamais balayés : l'archétype en compte 58, la passe
     A6 s'arrêtait à 27), avec comme aiguilles les hachages murmur3 de
     `prop_stats_mode_assault_bomb_arms` / `_detonations` / **`_disarms`** (`14380f8f0`,
     `14380fbf0`). La fabrique de hachage du dépôt (`mapvar/hash.go`) est la BONNE — vérifié le
     2026-09-04, `FUN_140748a74` et `FUN_140748d64` sont une seule fonction que Ghidra a scindée.
  *Pourquoi c'est la meilleure marche technique* : une correspondance par ÉGALITÉ DE NOM ne
  demande pas d'oracle. C'est la seule voie qui peut trancher sans qu'un humain ait vu un
  désamorçage. *Réserve* : `prop_stats_*` est un nom de propriété de mode, rien ne garantit
  qu'il soit répliqué — et §3.2 rappelle que la famille voisine `BombStats` ne l'est pas.

**Marche 4 — L'ORACLE HUMAIN : des créneaux Theater datés, sur les cinq candidats du §4.**
C'est le moyen le moins cher d'obtenir une VÉRITÉ TERRAIN, et il ne demande aucun corpus neuf.
Le précédent est établi et il a marché : lot « créneaux Theater » du 2026-09-03 (commit
`c378e5269`), où la conversion de temps a été contrôlée par trois chemins indépendants —
**écart 4 à 17 ms contre le manifeste sur 23 films**, coup d'envoi détecté à 0:26-0:40, et
9 `spent` sur 9 coïncidant à moins d'une seconde d'une impulsion mesurée.
- Ce qu'on livrerait à l'utilisateur : film, minutage (`df8fcbef` **8:47**, `df8fcbef` **2:49**,
  `9f57c612` **6:28**, `c75f33b8` **3:17**, `df8fcbef` **0:52**), et la question exacte : « la
  bombe est-elle désamorcée, et par qui ? »
- **Contrainte à vérifier AVANT de proposer les créneaux** : le Theater ne montre que les films
  où le joueur figure au roster (leçon du lot propulseur). Il faut donc contrôler, film par
  film, que l'utilisateur y est — sinon le créneau est inutilisable.
- Risque : faible. Coût : le temps de l'utilisateur, et une lecture de roster.

**Marche 5 — Faire naître des occurrences.** Le parc est épuisé pour la famille bombe stricte :
il n'y a pas de match d'Assaut supplémentaire à aller chercher, il n'y en a **que 8**. La seule
façon d'en ajouter est que des matchs d'Assaut soient JOUÉS puis synchronisés. Si cette voie est
choisie, la consigne se formule en une phrase : *jouer du One Bomb ou du Neutral Bomb en
désamorçant délibérément une bombe, et noter le minutage à la seconde* — un match dont on
connaît d'avance la vérité vaut dix films tirés au hasard, parce qu'il fournit l'oracle EN MÊME
TEMPS que l'occurrence. Coût : hors périmètre agent. Risque : aucun.

### 6.2 Ce qu'il ne faut PAS faire

- **Ne pas cuire d'artefacts en lot** pour « voir ce qu'il y a dedans » (§6.0).
- **Ne pas rouvrir** `ti=11` (delta), `ti=13`, le pied, les composants 0-27 : quatre négatifs
  mesurés, bornés, avec témoins (§3).
- **Ne pas re-proposer `ti=11 i14 state`** : déjà porté, publié et réfuté (p = 0,61 / 0,63 / 0,09).
- **Ne pas chercher un compteur de désamorçages dans le film** : §3.2 le ferme structurellement.

---

## 7. LE PROTOCOLE À APPLIQUER QUAND LE CORPUS (OU L'ORACLE) EXISTERA

Écrit pour être exécutable sans rien redécouvrir. Il se pose AVANT toute mesure, comme tous les
protocoles de ce chantier, et **aucun seuil ne se rabaisse après coup**.

### 7.1 Détection de l'événement — la descente complète de la jauge

Transposition directe de la lecture pausable (`navpoint_ti12_meche_test.go`), avec les seuils
DÉJÀ mesurés — ne pas en inventer d'autres :

```
SEGMENT      suite maximale d'échantillons d'un même slot, trous <= 500 ms
             (NavpointRiseMaxGapMS).
D'1  TENUE DE DÉSAMORÇAGE : segment strictement descendant, jamais au-dessus de son
     départ (max <= premier + mpPleinTolQ = 4), pente moyenne < mpPenteMaxQS = 60 q/s
     — mesuré 14 à 26 q/s pour les tenues, 138 q/s pour la chute d'explosion.
D'2  ABOUTIE si le dernier quantum atteint le PLANCHER de l'anneau (127 mesuré sur les
     trois films One Bomb) ; INTERROMPUE sinon (fins mesurées entre 185 et 246, l'anneau
     revenant ensuite à ~251).
D'3  Une tenue ABOUTIE doit suivre un ARMEMENT du même site sans explosion entre les deux
     — sinon ce n'est pas un désamorçage, c'est une chute d'explosion mal classée.
DÉDUPLICATION : les navpoints vont par PAIRES (écart de slot +12) et répliquent le MÊME
     anneau — deux fins à moins de 500 ms (bombPairToleranceMS) sont UN événement.
```

**Témoins obligatoires, écrits avant la mesure :**
- *Témoin négatif de mode* : rejouer D'1-D'3 sur Neutral Bomb / Husky Raid, où D2 vaut 17/17 et
  où **aucun désamorçage n'a eu lieu**. Une tenue ABOUTIE y serait un faux positif et
  invaliderait la lecture.
- *Plancher* : 500 à 1 000 instants aléatoires par film, graine fixe, même lecture. Sans
  plancher, un taux de couverture ne veut rien dire — c'est le piège n°1 de ce chantier, et il a
  mordu deux fois (24 % de couverture pour 22 % de plancher, côté pied).
- *Contrôle positif de complétude* : chaque explosion doit rester couverte par son armement après
  l'ajout du détecteur. Une lecture qui trouve des désamorçages en cassant les 13/13 Neutral est
  fausse.

### 7.2 Attribution de l'acteur — la voie POSITIONNELLE, transposée de B3

L'acteur n'est dans aucun canal (§8). La seule voie est celle que B3 a déjà validée pour
départager les désaccords d'armement (`replay/bombe_b3_desaccords_test.go:13-24`) :
**le désamorçage est une INTERACTION TENUE au site (`Device_GetInteractionHoldTime`), donc son
auteur est IMMOBILE, au site, pendant toute la tenue.**

```
P'1  CONTRÔLE POSITIF, et il est GRATUIT : sur les armements DÉJÀ attribués (9 sur 13, la
     jointure du lâcher), mesurer l'amplitude du nuage de positions de l'acteur sur
     [t−2500, t+2000] ms. Cette amplitude CALIBRE l'« immobilité d'interaction ».
     Sa position au lâcher devient un SITE de référence (b3Site), et les distances entre
     sites du même film calibrent l'échelle « au même endroit ».
P'2  CANDIDATS d'une tenue aboutie [t0, t1] : les slots VIVANTS sur la fenêtre, de l'ÉQUIPE
     ADVERSE de celle qui a armé (l'équipe est la seule identité que le moteur propage —
     activatingTeam), dont on mesure (a) l'amplitude sur [t0−500, t1+500] et (b) la distance
     au site de la pose en cours. Le porteur de la bombe est exclu par construction : un
     désamorceur ne porte rien.
P'3  VERDICT par événement : le candidat compatible avec P'1 (amplitude du même ordre que le
     contrôle, distance au site du même ordre que les sites entre eux) est le désamorceur.
     Deux candidats compatibles, ou aucun : INDÉCIS, publié SANS acteur, jamais deviné.
```

Outillage déjà écrit à réutiliser tel quel : `filmdec.ScanFilmBipedPositions` en `QuantaOnly`,
`ScanFilmClockOrigin`, et surtout **`dist3` (geometry.go) — une seule écriture de la distance 3D
au paquet, garde-rail `TestUneSeuleFormuleDeDistance3D`** : ne pas en écrire une seconde.

### 7.3 Les juges — qui arbitre, et ce qu'aucun d'eux ne peut faire

| juge | ce qu'il tranche | rang |
|---|---|---|
| **Vérité terrain Theater** (créneaux datés, §6.1 marche 4) | l'événement a-t-il eu lieu, et par qui | **le seul juge extérieur qui existe** |
| Anneau `ti=12 i14` | l'instant, et la complétude de la tenue | interne, mais gaté (0/1000 sur Neutral, 0/1000 sur One Bomb portées) |
| Armement précédent + absence d'explosion | qu'il y avait bien une bombe armée à désamorcer | interne, gratuit |
| Canal des armes tenues (`0x3fee4fcf`) | la reprise de la bombe après le reset (PRISE mesurée à +15,5 à +29,5 s sur les candidats) | corroboration, jamais preuve |
| Fil des kills | une tenue INTERROMPUE devrait souvent coïncider avec la mort du tenant — **contrôle falsifiable** | corroboration forte |
| API 343 / statborg `BombStats` | **RIEN. Jamais.** (§3.1, §3.2) | — |

### 7.4 Les trois pièges qui coûteront une journée si on les oublie

1. **Deux horloges.** L'anneau est sur l'horloge du FILM (manifeste), les positions et les
   portages sur celle du MATCH. Pont :
   `horlogeMatch = horlogeFilm + premierPaquetDuFilmUS/1000 − deathOffsetMS` (déjà en production
   dans `resolveOriginMs`). Écart mesuré 33 à 114 ms — **assez petit pour être invisible à un
   gate, assez gros pour fausser une jointure fine.** Écrire le test qui rougit quand le
   recalage saute (modèle : `TestBombArmsRecalageHorloge`, offset forcé de 40 000 ms).
2. **La garde de mode One Bomb.** `isArmableBombVariant` (`replaybuild/zones.go:165`) exclut One
   Bomb **par son nom**. Or c'est la SEULE variante où le désamorçage est visible. Toute
   livraison du désamorçage dépend donc de l'étape **E2-ter** du plan (porter la lecture pausable
   en production, faire tomber la garde 1, garder la garde 2 tout-ou-rien). **Dépendance à
   déclarer d'entrée** : tant qu'E2-ter n'est pas livrée, un désamorçage mesuré ne peut pas être
   publié.
3. **Où ça s'écrirait.** La place est déjà prévue : événements datés dans
   `match_objective_events` avec `objective_type = 'bomb'` (valeur déjà au schéma), agrégat par
   joueur dans `match_bomb_stats` (table dédiée append-only + vue `_latest`, créée en E3). Une
   sixième colonne `bomb_defusals` s'y ajoute par la recette ADR 0026 (step au NOM NEUF), jamais
   par un ALTER d'une migration déployée.

---

## 8. CE QU'ON N'AURA JAMAIS — la limite structurelle

**L'acteur du désamorçage n'est dans AUCUN canal du film. Ce n'est pas une lacune de décodage,
c'est le moteur.**

- Le parcel Lua ne propage que l'ÉQUIPE (`activatingTeam`) ; il n'existe aucun
  `activatingPlayer` (§2.1).
- Les deux propriétés qui remontent au HUD (`CenterScoreboard_BombState`,
  `CenterScoreboard_BombNormalizedCaptureProgress`) appartiennent à une famille **globale au
  match** : elles ne portent pas de joueur.
- Le compteur par joueur (`BombStats_BombDefusals`) est un champ de télémétrie Bond, jamais
  répliqué (§3.2).
- **Et contrairement à l'armement, le désamorceur ne PORTE PAS la bombe** : le canal des armes
  tenues — qui a sauvé l'attribution de l'armement, le lâcher étant le geste de pose — **ne le
  verra pas.** La jointure qui marche pour l'armeur n'a pas d'équivalent ici.

**Conséquence à accepter d'avance** : même avec un canal d'événement parfait, on aura
**l'instant et l'équipe**. Le JOUEUR ne s'obtiendra que par la voie positionnelle (§7.2), avec
son taux d'indécision, et les événements indécis se publieront SANS acteur — jamais avec un
acteur deviné.

---

## 9. CONDITION DE REPRISE (testable, en une phrase)

**Le sujet redevient traitable le jour où EXISTE au moins un désamorçage AVÉRÉ — c'est-à-dire un
couple (film, instant) dont la vérité est établie hors de la mesure qu'on veut valider : soit
par un créneau Theater vérifié par l'utilisateur sur l'un des cinq candidats du §4, soit par un
match joué exprès en désamorçant et dont le minutage est noté.**

Deux corollaires qui n'attendent PAS cette condition, parce qu'ils ne demandent aucun oracle :
- la lecture par NOM du dictionnaire ECS et des composants 28-57 (§6.1, marche 3) ;
- le recensement en lecture seule du parc et le passage des films d'Assaut non étudiés au
  protocole (§6.1, marches 1 et 2).

Les deux se DEMANDENT avant d'être lancés (règle de campagne), et aucun ne justifie une cuisson
d'artefacts en lot.

---

## 10. QUESTIONS OUVERTES — laissées telles quelles, faute de pouvoir les trancher en lecture seule

1. **Les cinq candidats du §4 portent-ils une descente complète jusqu'au plancher ?** Une seule
   mesure y répond (§7.1 sur trois films déjà en cache). Non lancée : hors périmètre de cette
   rédaction, et une campagne se demande.
2. **Les 14 films Husky / Super Husky Raid en cache sont-ils de la famille bombe ?** Répondu par
   la marche 1, sans décoder.
3. **L'utilisateur figure-t-il au roster de `df8fcbef`, `9f57c612`, `c75f33b8` ?** Sans cela, pas
   de créneau Theater exploitable.
4. **`bomb_icon_reader_component` figure-t-il au dictionnaire ECS d'un film d'Assaut ?** Question
   à une seule lecture de `chunk_00`, dont la réponse n'a jamais été journalisée.
