# ADDENDUM à `ETAT_DE_L_ART_KILLWEAPON.md` — trouvailles des 2026-07-26 et 2026-07-27

> **Destiné à la branche `feat/filmdec-killweapon`**, dont l'index `ETAT_DE_L_ART_KILLWEAPON.md`
> fait foi. Rédigé sur `feat/filmdec-continuation` et NON écrit directement dans l'autre worktree :
> celui-ci était actif au moment de la rédaction (16 fichiers modifiés, dernier commit `dbe0b7ec3`
> à 21 h 38), et écrire dedans aurait risqué d'écraser une session en cours.
>
> Format calqué sur celui de l'index : `MESURE` = chiffre reproduit · `MESURE+VERIFIE` = rejoué
> par un agent adversarial indépendant · `THEATER` = confirmé par l'utilisateur.
>
> **Ce document ne contient QUE du validé.** Ce qui est en cours de réfutation est isolé au §6 et
> ne doit pas être cité comme acquis.

---

## 1. GRAMMAIRE DU RECORD — une faute structurelle corrigée

| QUESTION | REPONSE | STATUT |
|---|---|---|
| Le payload d'un paquet type-0 commence-t-il par le premier record ? | **NON.** Il porte une **amorce de 2 bits** avant le premier record. `FUN_142987460` fait `DAT_144706104 = FUN_1406cf008(reader)` — un `R(1)` — sur un lecteur créé sur le payload EXACT (`FUN_14298816c` : memcpy de `*(param_2+4)` octets, puis `FUN_1406d5cc0(reader,3)` qui repositionne au début). | MESURE |
| Quel témoin le prouve, indépendamment de toute carte ? | Le compteur de composants tient sur 3 bits : un record épars en porte **au plus 7**. Vérité terrain (138 390 records bipèdes) : **99,86 %** dans cette plage, mode à 4. | MESURE |
| Effet mesuré | part de masques 1..7 sur les DELTA de slots bindés : **14,65 % → 84,81 %**. Le niveau du hasard, mesuré à des décalages arbitraires, vaut **10,67 %** — **l'ancien décodeur faisait donc MOINS BIEN que le hasard**. | MESURE |
| Conséquence sur les composants d'inventaire | excès de présence par rapport à la vérité : `i22` ×209 → ×63 · `i47` ×331 → ×66 · `i48` ×356 → ×86 · `i56` ×642 → ×69. | MESURE |
| Le second bit est-il établi au désassemblage ? | **NON, et il faut le dire.** Le désassemblage n'établit qu'**UN** `R(1)`. Le second est retenu par la mesure seule. Trois grammaires donnent le même en-tête total de 20 bits sur le premier record et ne se séparent qu'au second : amorce 2 + idLow 11 (**84,81 %**) · amorce 1 + idLow 12 (51,67 %) · amorce 2 + idLow 10 (5,24 %). | MESURE |
| Est-ce suffisant ? | **NON.** Après correction, `i22` lit encore **90 %** de comptes de grenades physiquement impossibles. Il reste au moins une faute dans le CORPS des records. | MESURE |

> **Leçon de méthode, la plus chère du jour.** `.ai/GRAMMAIRE_RECORD_FILM.md` §4 **mesurait déjà**
> « amorce de paquet 2 bits, en-tête 21 » depuis sa rédaction. Le constat n'avait jamais été câblé
> dans le décodeur. Une mesure juste, écrite dans un document qui « fait foi », n'a servi à rien
> pendant des semaines. **Vérifier qu'une conclusion documentée est effectivement branchée fait
> partie de la conclusion.** (Même famille que le patron E8 de l'index.)

---

## 2. PROJECTILES — un archétype entier que personne n'avait cherché

| QUESTION | REPONSE | STATUT |
|---|---|---|
| Un projectile est-il répliqué comme une entité ? | **OUI.** Le registre `chunk_00` contient un archétype dont les composants s'appellent littéralement `projectile-at-rest-state`, `projectile-tether-state`, `projectile-command_tick`, `projectile-deceleration-disabled-state`, et dont les quatre premiers sont `object-position`, `object-translational-velocity`, `forward-and-up`, `angular-velocity`. Sur ce build : **ti=41**. Nos notes le classaient « divers, mal caractérisé ». | MESURE+VERIFIE |
| Peut-on reconstituer les trajectoires ? | **OUI.** 580 vies, 13 544 positions, ~23 points par vol, durées **0,27 à 2,45 s**. | MESURE |
| Témoin | **65 des 70 lancers** de grenade voient naître une trajectoire à ±200 ms. Contrôles, mêmes lancers décalés en bloc : **+3 s → 13 · +7 s → 12 · +13 s → 11 · −5 s → 11**. | MESURE+VERIFIE |
| Témoin plus fin (autre équipe, autre portage) | distance naissance ↔ bipède lanceur **0,77 u** contre 6,4 u (instant permuté) et 33,9 u (bipède au hasard) ; le point de naissance est à **0,4 u devant et 0,64 u au-dessus** du repère — la main, à la bonne hauteur. Direction initiale ↔ cap de visée : **1,0° de médiane**. | MESURE+VERIFIE |
| L'impact est-il dans le film ? | **NON.** Aucun événement de détonation. On lit la **dernière position répliquée** ; pour une grenade à fragmentation la réplication cesse ~1,4 s après le lancer alors que la mèche court jusqu'à ~3 s. Seul `projectile-at-rest-state` (i18) CERTIFIE une fin de vol (78 fois sur 79). À l'écran : « dernière position connue », **jamais** « impact ». | MESURE |

---

## 3. `0x4C0C00` N'EST PAS UN MARQUEUR

| QUESTION | REPONSE | STATUT |
|---|---|---|
| Qu'est-ce que c'est alors ? | Le **milieu d'un record de CRÉATION d'entité** : `[5 bits bas de typeIndex=41][19 bits constants d'amorce de l'état par défaut]`. Le « marqueur » fonctionnait par accident heureux — il reconnaissait la naissance d'un projectile. | MESURE+VERIFIE |
| Que sont les 4 « identifiants de grenade » ? | Les **identifiants globaux de tag du groupe `proj`, décalés d'un bit à gauche** : `0x580B8831 << 1 == 0xB0171062` (Fragmentation), `0x6071A622 << 1 == 0xC0E34C44` (Plasma), `0x1D92B3EA << 1 == 0x3B2567D4` (Dynamo), `0x49097214 << 1 == 0x9212E428` (Spike). 4/4 exacts. | MESURE |
| Faut-il donc lire à +23 au lieu de +24 ? | **Indifférent ICI** : le bit à +23 vaut **0 dans les 1 416 marqueurs** du film, sans exception — les deux lectures reconnaissent exactement les mêmes 70 lancers. | MESURE |
| **PIÈGE** — l'index joueur se décale-t-il aussi ? | **NON, ET LE PROPAGER CASSE UN DÉCODEUR JUSTE.** Mesuré : à **+102** les 70 valeurs tombent toutes entre 16 et 19 ; à **+103** elles sont toutes dans 0..7. L'index reste à **+103**. | MESURE |
| Ce que l'identification ouvre | remplacer la liste blanche de 4 valeurs par une résolution via le **catalogue de tags `proj`** (3 086 tags). Lues ainsi, **19 valeurs récurrentes sur 19** sont des `proj`, contre 1,18 % attendus par hasard. Cela nommerait TOUS les projectiles, pas seulement les grenades. | MESURE |

---

## 4. INVENTAIRE — ce qui est tranché, et ce qui ne l'est pas

> **AMENDÉ LE 2026-07-27.** Cette section a été écrite quand l'inventaire n'était pas lu. Il
> l'est désormais : voir Partie II §10 (l'adresse exacte), §12 (i47 craqué) et §13 (i22 relu
> sans erreur). Les cellules ci-dessous portent leur correction.

| QUESTION | REPONSE | STATUT |
|---|---|---|
| `i22` est-il décodé ? | **OUI, et bit-exact.** `FUN_140f0de1c` : `count = FUN_1424d0f48(reader)` = `R(3)` **BRUT sans +1** (contrairement à `FUN_1424cd07c`), puis `count × R(8)`. Largeur vraie **35 bits fixes à 100 %** sur 259 mesures = 3 + 4×8, donc **count vaut 4**. | CONFIRMÉ 27/07 — 100 % de 249 lectures relues aux positions exactes |
| Alors pourquoi les valeurs sont-elles fausses ? | **Le problème n'est pas la recette, c'est l'adresse.** Le curseur arrive au mauvais endroit. `count != 4` dans 90 % des lectures ; des valeurs vont jusqu'à 255 alors que le jeu plafonne à 2 par emplacement. — **RÉSOLU LE 27/07.** L'adresse est désormais exacte : `position = paquet.Start*8 + curseur_moteur`, vérifiée 249/249. Relues à cette adresse, les valeurs sont IMPECCABLES : compteur à 4 dans **100 %** des cas, valeurs **0, 1 ou 2 uniquement**, jamais 255. Les « 90 % de `count != 4` » et les valeurs à 255 étaient des artefacts de NOTRE curseur, pas le contenu du film. | RÉSOLU |
| L'hypothèse « 4 emplacements sans compteur » | **RÉFUTÉE** au désassemblage : le compteur existe. | MESURE |
| Les grenades sont-elles dans les keyframes, comme les armes ? | **OUI mais ce n'est PAS l'inventaire.** 12 occurrences sur 26 keyframes (un inventaire de 8 joueurs en exigerait 8 à 16 par keyframe ; contrôle positif : le même balayage sur les 39 familles d'arme rend 16 à 26 par keyframe). Confrontées aux 70 lancers : **10/12 correspondent à un lancer du MÊME TYPE dans les 4 s**, contre **2/12** pour les lancers décalés de 7 s. **Ce sont les grenades EN VOL.** | MESURE |
| Encodage des grenades dans le flux | **identique à celui des armes** : chaque identifiant est suivi du suffixe `0x42C9679F`. Le compte 64 bits égale strictement le compte 32 bits (70 sur les delta, 12 sur les keyframes), contre 0,012 et 0,007 attendues par hasard. | MESURE |
| Correction à porter dans `keyframe_loadout.go` | son en-tête affirme « CE QUE CE DÉCODEUR NE DONNE PAS : ni les grenades ». **Faux** : il ne les donne pas parce qu'elles ne sont pas dans son catalogue de familles. | MESURE |

---

## 5. FORMAT `.module` — trois correctifs, dont un confirmé par un tiers

| QUESTION | REPONSE | STATUT |
|---|---|---|
| `dataOffset` (+0x18) est-il un `u32` ? | **NON, c'est un entier 48 BITS** ; les 2 octets de poids fort sont des DRAPEAUX. (Déjà en §8.4 de l'index ; **porté** dans `internal/himodule` le 2026-07-26.) | MESURE+VERIFIE |
| Confirmation indépendante | **Reclaimer** (`Reclaimer.Blam/Blam/HaloInfinite/ModuleItem.cs`) déclare `enum DataOffsetFlags : byte { UseHD1 = 0b00000001, ... }`. | MESURE+VERIFIE |
| Que contient le compagnon `.module_hd1` ? | Sur ridgeline : **59 entrées** sur 5 004 y pointent (`any/globals/common` : **aucune**). Base calibrée à **100 % d'extraction**. Contenu : **286 Mio de TEXTURES, zéro géométrie.** Porte fermée pour la géométrie. | MESURE |

---

## 6. GÉOMÉTRIE DE CARTE — offsets confirmés par une implémentation tierce

**Source** : `Gravemind2401/Reclaimer`, `Reclaimer.Blam/Blam/HaloInfinite/`. Implémentation C#
indépendante qui exporte des modèles ouverts dans Blender — donc validée par l'usage.

### 6.1 `sbsp` — instances de géométrie (`ScenarioStructureBspTag.cs`)

```
[Offset(420)] GeometryInstances        (bloc des instances)
[FixedSize(320)]                        = 0x140, notre stride — EXACT
  [Offset(0)]   TransformScale          <- notre "scale @0x00" : PAS vestigial
  [Offset(12)]  Matrix4x4 Transform     <- nos forward/left/up/position @0x0C..0x30
  [Offset(60)]  RuntimeGeoMeshReference <- notre meshRef @0x3C
  [Offset(116)] MeshIndex               <- notre meshIndex @0x74
  [Offset(118)] BoundsIndex             <- INEDIT : l'instance DESIGNE son jeu de bornes
```

### 6.2 `rtgo` — géométrie de rendu (`RuntimeGeoTag.cs`)

```
[Offset(16)]  PerMeshData         [Offset(64)]  Sections
[Offset(104)] BoundingBoxes       [Offset(190)] TotalVertexBufferCount
[Offset(196)] MeshResourceGroups
  RuntimeGeoPerMeshDataBlock : [FixedSize(144)] = 0x90, notre pas — EXACT
```

**Vérification sur nos tags extraits** : le bloc `PerMeshData` mesure **864 o = 6,00 × 144**
(et 1 296 = 9,00 × 144, 720 = 5,00 × 144 selon le tag) — multiple entier parfait, l'offset 16 et
le pas de 144 sont établis sans ambiguïté.

### 6.3 CORRECTION À PORTER — les bornes de déquantification

`.ai/HANDOFF_MAP_GEOMETRY_FROM_MODULES.md` §9.3 place les bornes **par maillage** à `+0x44` du
record de 0x90. **Mesure du jour** : le bloc `BoundingBoxes` (+104) fait **84 octets, identiques
sur les quatre tags examinés**, et son contenu est trois paires parfaitement symétriques —

```
+4 −2,6357  +8 +2,6357   (X)     +12 −1,7432  +16 +1,7432   (Y)
+20 −3,0514 +24 +3,0514  (Z)     +28/+36 : U et V (texture)
```

C'est une structure de **bornes de compression**, et Reclaimer l'applique en UN SEUL jeu par
modèle (`model.SetCompressionBounds`). Avec `BoundsIndex` (+118 de l'instance) pour désigner
lequel. **Ce n'est donc ni « par maillage » ni « global » : c'est indexé par instance.**

### 6.4 Statut du §9 du HANDOFF — corrigé le 2026-07-26

Il se déclarait **RÉSOLU**. Mesure : sa chaîne `Per Mesh Data` **rejette 1 908 maillages sur
2 369 (80,5 %)** sur ridgeline et rend un X à 1e33, quand les vraies bornes sont X[−41,10 ; 72,11].
Elle a été établie sur **catalyst** et jamais reproduite ailleurs. Statut ramené à
**PARTIELLEMENT RÉSOLU**.

### 6.5 EN COURS DE RÉFUTATION — ne pas citer comme acquis

Une reconstruction par les **triangles** (chaînage via `Sections`, 10 357 instances,
28,9 M triangles, 0 non résolue) produit un fond de carte que **l'utilisateur reconnaît à l'œil**,
fer à cheval et ponts sud compris — le témoin le plus fort obtenu sur ce chantier. Vide dans la
zone du fer à cheval : **12,9 m²** contre **0,00 m²** en boîtes ; désertion du disque central
reconstruite **×63,8** contre **×64** mesurée indépendamment sur les seules trajectoires ; centre
réel à **99,7 %** de 5 862 disques tirés au hasard.

**La passe de réfutation n'a pas encore tourné**, et la fois précédente trois attaques avaient
cassé une annonce identique (la règle employée creusait *partout* : un simple couloir s'y vidait
à 49 %). Deux points restent à vérifier : le témoin de non-régression est donné à **82,0 % à
moins de 25 cm** quand l'ancien fond était validé à **80,6 % à moins de 5 cm** (seuil cinq fois
plus large, nombres non comparables), et la question des bornes du §6.3.

---

## 7. PISTES MORTES — à ne pas rouvrir

| PISTE | POURQUOI ELLE EST MORTE | STATUT |
|---|---|---|
| `TraversalPrecision.AxisW` = 13/13/14 pour réparer `i0` | **DÉGRADE.** `AxisW` est la largeur des DELTAS (6 bits suffisent) ; les 13/13/14 sont celles des ABSOLUS, portées par une seconde largeur (`absAxisW`). Mesure : `i22` passe de 90,02 % à **92,83 %** de comptes impossibles. — **SUITE, 27/07** : le diagnostic de cette ligne était juste, et le vrai correctif a été applique (`absAxisW` -> 13/13/14 + le champ « fini » de 2 bits, soit 23 -> 47 bits, conforme aux 47 bits mesures a 100 %). **Il n'a produit AUCUN effet** : i22 11,971 % -> 11,979 %. Verifie a l'oracle (i0 force a la largeur CE exacte) : 12,099 %. La largeur d'i0 n'est donc PAS la cause, ni dans un sens ni dans l'autre. | CLOS |
| La calibration `axisW=14, indexW=1, rsp=3` du §5.1 appliquée à notre walk | **SANS EFFET** : 90,25 % → 89,60 %. Le test que le chantier avait posé (« si le taux chute nettement sous 91 %, i0 est la cause dominante ») **échoue**. La cause résiduelle est ailleurs. — **LA CAUSE EST TROUVÉE, 27/07** : la marche séquentielle ne décode que **2,0 records par frame** là où le moteur en traite **13,7**. On perd les onze douzièmes de chaque paquet. Aucune correction de largeur ne pouvait donc rien changer : elles ne portaient que sur les 15 % du flux qu'on atteignait. | RÉSOLU |
| Combler la bande de slots d'un objet du monde | **CONTAMINE.** Le bipède a une plage remplie à 91 %, mais le projectile à **8 %**, l'équipement à **23 %**, le corps rigide à **10 %** — et les plages de ti=37 [1462,2660] et ti=38 [1280,2641] se recouvrent presque entièrement. Symptôme : les trois archétypes rendaient des chiffres **identiques à 1 % près sur cinq critères indépendants**. | MESURE |
| S'en tenir aux slots OBSERVÉS | **RATE L'ESSENTIEL** : 57 projectiles au lieu de 580 (un projectile vit moins d'une seconde, les keyframes sont espacés de 20 s). La forme juste : combler PUIS retirer tout slot vu porter un autre archétype. | MESURE |
| Les équipements déployés ont une position lisible | **NON CONCLUANT.** Une fois la bande assainie les archétypes se séparent (projectile 2,47 m de dispersion et 12,3 % d'immobiles contre 0,28 m et 44,7 % pour l'équipement), mais une durée de vie médiane de **1,15 s** est absurde pour un mur déployable qui tient une quinzaine de secondes. | MESURE |
| Le grappin par le mouvement anormalement rapide | **AUCUNE SIGNATURE.** Les 14 candidats montent tous de **+1,12 à +1,46 m** — l'uniformité d'un SAUT, pas d'une traction. Et le seuil de 3 m/s était mauvais. | MESURE |
| La collision lue en `float32` brut dans les `scgt` | **FAUX POSITIF.** 53 % de triplets « dans les bornes de la carte » contre 14 % au hasard, mais **95 à 97 % des points sont à moins d'un mètre de l'origine** : c'est la signature d'octets quelconques lus comme des flottants. Le nuage est une croix. **Toujours DESSINER un résultat, pas seulement le compter.** | MESURE |

---

## 8. TABLE DES DÉSÉRIALISEURS DU §6 — PARTIELLEMENT PÉRIMÉE

Confrontation du §6 de l'index (daté du 3 juillet) au **code actuel** de `feat/filmdec-killweapon` :

| i | composant | doc §6 | killweapon AUJOURD'HUI | verdict |
|---|---|---|---|---|
| 3 | object-angular-velocity | `FUN_140D70998` | `FUN_140D87740` | le code a bougé |
| 42 | biped-desired-weapon-set | `FUN_14109D298` | `FUN_1406D01FC` | le code a bougé |
| 47 | biped-desired-grenade-set | `FUN_140C6A628` | `FUN_140C6A638` | le code a bougé |
| 48 | biped-desired-ability-set | `FUN_1410F8FCC` | `FUN_1406D0FF0` | le code a bougé |
| 61 | simulation-state-playback | `FUN_142F02454` | `FUN_142ED6D20` | le code a bougé |
| 25 | unit-command-tick | `FUN_1406CFB28` | **non annoté** | NON TRANCHÉ |
| 26 | unit-equipment | `FUN_1409685D8` | **non annoté** | NON TRANCHÉ |
| 27 | unit-stun | `FUN_142ED75FC` | **non annoté** | NON TRANCHÉ |

**Sur cinq divergences, c'est le document qui est périmé** — le code actuel et le nôtre
concordent. Le §6 porte d'ailleurs lui-même « ⚠ mon mapping Go était `FUN_140d87740` = FAUX » sur
`i3`, et la branche est depuis revenue à cette valeur.

**Le seul apport inédit du §6** est le trio `i25`/`i26`/`i27` : ce ne sont pas trois erreurs
indépendantes mais **les trois mêmes fonctions permutées d'un cran**. Personne ne les annote, donc
personne ne tranche. Si la permutation est réelle, elle décale le curseur AVANT `i47` et `i48` —
exactement le symptôme observé. **Départage** : `i25` doit consommer **10 bits fixes** (100 % sur
12 683 mesures directes du CSV).

> **Recommandation** : ajouter une DATE à chaque ligne du §6, ou renvoyer au code plutôt qu'aux
> adresses. Une table d'adresses non datée redevient fausse à chaque correction du décodeur, et
> elle a failli me faire modifier cinq mappings justes.

---

## 9. AUTRES MESURES UTILES

| SUJET | MESURE |
|---|---|
| Distribution des vitesses de pas d'un joueur | médiane **2,51 m/s** · p75 **2,99** · p90 3,52 · p95 4,18 · p99 **10,09** · p99,9 16,00 · p99,99 64,74. **3 m/s est le 75e centile, pas un plafond.** Au-delà de 20 m/s : téléportations de décodeur. |
| Capacités Spartan — identifiants | **12 tags `eqip` sur 108** apparaissent dans le film ; contrôle : 5 000 ids au hasard → 36 occurrences au total. Les 8 identifiants d'équipement sont non nuls dans **3/3 films Super Fiesta et 0/24 ailleurs**. |
| Capacités Spartan — nommage hors ligne | **IMPOSSIBLE par les tags** : sur 11 `eqip`, **un seul** porte une chaîne lisible (`ability_heal_blast_warmup` = Champ de réparation) et aucun libellé n'apparaît dans les 238 tables `uslg`. Cohérent avec `fileNameSize = 0` en build de release. Seule voie restante : les banques Wwise (§2.2 de l'index). — **PRÉCISION DU 27/07 : ne vaut que pour le NOMMAGE.** L'ÉTAT est acquis : `i57` porte l'interrupteur marche/arrêt (bit 0 à 48 % sur 990 lectures, Partie II §14) et `i48` porte l'identifiant de la capacité équipée (218 lectures, 13 valeurs distinctes sur 1024). On sait donc DIRE qu'une capacité est équipée et active, et LAQUELLE par son identifiant ; on ne sait toujours pas lui donner son nom lisible. |
| Zones de callout comme borne de carte | les 28 zones + 1,5 m de marge contiennent **100,0 % des 29 221 positions** de joueur, en ne couvrant que **35,6 %** de la grille. Retire 24,3 % du sol reconstruit comme superflu. |
| `i57 biped-spartan-ability` | **absent du switch `consumeByName`**, alors qu'il ne fait que **2 bits** et précède `i58`/`i59` dans **995 des 1 040** records portant `i59`. Le câbler débloque mécaniquement l'accès à l'état non-prédit de la capacité. — **CONFIRMÉ ET DÉCODÉ LE 27/07** : 990 lectures, 2 bits, et le taux de bits à 1 par position tranche la sémantique — **bit 0 à 48 %, bit 1 à 4 %**. Le premier bit EST l'interrupteur marche/arrêt ; le second est un drapeau rare. Reste à câbler dans `consumeByName`. |

---

# PARTIE II — 2026-07-27 : la capture du dispatch, et ce qu'elle tranche

Tout ce qui suit vient d'une CAPTURE CHEAT ENGINE DU DISPATCH DES COMPOSANTS (site
0x14076CD11, juste avant l'appel indirect vtable+0x28), posée par MCP sur le process vivant.
975 250 composants journalisés sur le film 9e8fb31b (Cliffhanger, Slayer:Arena Super Fiesta,
24/07 19h21). Pour chaque composant désérialisé : entité, archétype, index, CURSEUR DE BITS
AVANT LECTURE, et 16 octets bruts lus depuis le tampon du film.

Vérification préalable : 0x7FF7A655CD11 − 0x7FF7A5DF0000 = 0x76CD11, soit 0x14076CD11 en base
Ghidra. LE BUILD CORRESPOND EXACTEMENT À L'IMPORT, donc toute la table des désérialiseurs
i0..i63 est valide sur ce process.

## 10. LES DEUX IDENTITÉS QUI RENDENT TOUT LE RESTE POSSIBLE

POSITION. Le curseur de bits capturé EST la position absolue dans le payload du paquet,
décalage NUL :

    position_exacte = paquet.Start*8 + curseur_moteur

Établi par balayage de l'amorce sur 0..8 : SEUL +0 PRODUIT UN PARSE VALIDE, et il en produit
249 sur 249. Ce n'est pas un ajustement, c'est une identité — et elle vaut pour TOUT composant.

LARGEUR. La différence de curseur entre un composant et le suivant DU MÊME record EST la largeur
consommée, exactement. Plus aucune largeur n'a besoin d'être portée depuis Ghidra.

CONSÉQUENCE OUTILLAGE : cmd/tmp_comptruth localise chaque lecture d'un composant à l'octet près
(sur i22 : 249 LOCALISÉES, 0 INTROUVABLE, 0 AMBIGUË). C'est LE JUGE qui manquait — tout scanner
devient mesurable en précision ET en rappel, au lieu d'être estimé.

## 11. i0 DU BIPÈDE = 47 BITS — la contradiction 45/47 est morte

47 bits, une seule valeur distincte, 100 % de 154 158 dispatches. Le compte se ferme :

    1 bUsePred + 1 bDelta + 1 precHigh + 1 indexSel + 1 IndexW + (13+13+14) + 2 finite = 47

Le chemin ABSOLU de notre décodeur n'en consommait que 23 : absAxisW retombait sur pd.AxisW
= 6/6/6 (la largeur du chemin DELTA) au lieu des 13/13/14 de la table de région, et le champ
« fini » de 2 bits n'était jamais lu — alors que le chemin world-object le lit depuis toujours.
Corrigé. MAIS : AUCUN EFFET MESURABLE (i22 11,971 % -> 11,979 %), y compris avec i0 forcé à la
largeur CE exacte par un oracle. La largeur d'i0 n'était pas la cause du désordre.

## 12. i47 CRAQUÉ — le loadout de grenades se lit

190 lectures, DOUZE VALEURS DISTINCTES SEULEMENT :

    i47 = [6 bits : masque des types possédés][3 bits : type SÉLECTIONNÉ]

Trois vérifications indépendantes : (1) le type sélectionné appartient TOUJOURS au masque,
12 fois sur 12 ; (2) les 2 bits de poids fort du masque sont toujours nuls — 4 types de grenades
dans un champ de 6 bits ; (3) 50,5 % des i47 ont un masque VIDE, et i22 a 43 % de comptes tous
nuls — deux composants indépendants racontent le même fait.

## 13. i22 — la grammaire du projet était JUSTE

35 bits, 100 % des lectures = R(3) + 4 x R(8), donc compteur TOUJOURS à 4. Relecture aux 249
positions exactes, sans filtre : valeur maximale 2, valeurs vues 0, 1, 2 et rien d'autre. LES
BORNES DU JEU DU §6 DE GRAMMAIRE_RECORD_FILM SONT EXACTES — j'avais écrit l'inverse pendant une
heure avant de mesurer, c'était mon filtre qui rejetait à tort le cas « zéro grenade portée »
(état parfaitement normal après un lancer).

## 14. i57 — L'ÉTAT ACTIF DES CAPACITÉS

990 lectures, 2 bits. Taux de bits à 1 par position : BIT 0 = 48 %, BIT 1 = 4 %. Le premier bit
EST l'interrupteur marche/arrêt. Couplé à i48 (quelle capacité est équipée), le couple demandé
est complet.

CONFIRME LA NOTE DU §9 : i57 était absent du switch consumeByName alors qu'il ne fait que 2
bits. Le câbler débloque bien l'accès à l'état de la capacité.

## 15. L'INVENTAIRE VIT DANS LES RECORDS DENSES — avec témoins négatifs

Part de records à masque dense (plus de 7 composants) parmi les records portant chaque
composant. Référence de la population : 0,145 % (231 records denses sur 159 772).

    i43 arme tenue        208 records, 194 denses  93,27 %   x643
    i47 grenades          215 records, 182 denses  84,65 %   x584
    i22 comptes           256 records, 191 denses  74,61 %   x515
    i42 arme desiree      447 records, 198 denses  44,30 %   x306
    i48 capacite          218 records,  95 denses  43,58 %   x301
    i0  position (TEMOIN) 154 158 records           0,08 %   x0,55
    i25 tick    (TEMOIN)  158 804 records           0,13 %   x0,90

Les deux témoins font la preuve : les composants COMMUNS restent à la référence ou en dessous,
l'inventaire est enrichi de 300 à 640 fois.

## 16. EN-TÊTE DE RECORD — idLow VAUT 14, et les gros records sont des DELTA DENSES

Mesure de l'écart entre la fin d'un record et le premier composant du suivant :

    records COURTS : 27, 33, 39, 45, 51, 57 bits — ESPACÉS DE 6, la largeur d'un index
    frequences     : 45 (42,5 %), 39 (29,9 %), 51 (13,4 %), 33 (9,5 %)

Ces fréquences reproduisent la distribution des tailles de masque (mode à 4, puis 3, 5, 2) :
validation croisée complète. La base vaut 21 = 1 + 14 + 2 + 1 + 3, donc idLow = 14.

    records LONGS : 82 bits dans 82,9 % des cas — valeur UNIQUE
                    82 = 1 + 14 + 2 + 1 + 64

CE SONT DES DELTA À MASQUE DENSE, PAS DES RECORDS NEW. Mon hypothèse fondée sur param_4 est
réfutée par cette mesure.

## 17. COMPOSANTS JAMAIS DISPATCHÉS

i60, i61 (simulation-state / playback), i62 (glissade), i63 (action) : ZÉRO LECTURE sur tout le
film. À ne pas confondre avec « non décodés » — le moteur ne les envoie pas. Porter leur
grammaire serait du travail perdu.

## 18. DEUX TRANSPOSITIONS DE VOTRE RECETTE, RÉFUTÉES — avec leur cause

Votre percée (RE_LOG 7ter.73) est reprise ici : « la marche est un sous-ensemble positionnel
total du scan », 346/346. Elle vaut. Mais deux de ses composants ne se transposent pas :

LE CATALOGUE DE VALEURS — INOPÉRANT SUR DES CHAMPS ÉTROITS. Votre sélectivité vient du RAPPORT
alphabet/catalogue : 2^32 valeurs possibles pour 468 admises. Mes champs font 9 bits : 512
possibles pour 8 admises. Mesuré : 3 202 042 candidats pour 179 lectures vraies, précision
0,01 %. La sélectivité vient de la LARGEUR du champ, pas de la qualité de la liste.

LA MULTIPLICITÉ DE POSITION — ZÉRO INFORMATION ICI. Vous la donnez comme « le discriminant le
plus rentable, et il est gratuit ». Mesuré chez moi : candidats VRAIS à multiplicité 1 = 44,1 %,
candidats FAUX = 45,2 %. Distributions identiques, le rapport candidats/vrai ne descend jamais
sous 22 quel que soit le seuil. Cause : vos faux positifs viennent d'un motif de 32 bits
revenant à offsets FIXES (en-têtes, bourrage) ; les miens viennent de la DENSITÉ d'un filtre
trop lâche, donc répartis partout.

## 19. CE QUI SE TRANSPOSE : LE GABARIT RIGIDE

En relisant votre gabarit de 58 bits, l'essentiel m'avait échappé : mort, porte du tag, tag,
porte victime, victime, porte tueur, tueur, catégorie sont TOUS DES CHAMPS INTERNES AU COMPOSANT
i11. Vous ne touchez ni l'en-tête du record ni le masque.

Appliqué chez moi (cmd/tmp_gabarit), avec une contrainte gratuite que vous n'exploitez pas —
LES INDEX DU MASQUE SONT STRICTEMENT CROISSANTS, soit 4,6 bits offerts pour un masque de 4 :

    bornes du jeu seules      5 085 candidats  249 vrais    4,9 % precision  100 % rappel
    GABARIT RIGIDE                  61            59       96,7 %            23,7 %

1,03 CANDIDAT PAR LECTURE VRAIE. Et idLow se calibre tout seul : seules les valeurs 10 et 14
produisent des candidats, et ce sont la même solution (10 décalé de +4 EST 14).

Le rappel plafonne parce que ce gabarit part du DÉBUT du record et traverse tout. Ancré
localement sur i22 et prolongé vers l'avant, le compromis s'inverse : RAPPEL 97,9 %, PRÉCISION
5,5 %. Il manque un champ large et catalogué dans le voisinage — le seul candidat est i43
(forme longue de ~200 bits, co-occurrence 93 % avec i22), c'est-à-dire précisément ce que vous
savez déjà nommer.

## 20. OUTILS PRODUITS, RÉUTILISABLES TELS QUELS

    tools/ce/filmdec_full_capture.lua   capture large + signature d'ancrage (tous composants)
    cmd/tmp_cecapture                   masques, largeurs, presence, locus ; -worlddump
    cmd/tmp_comptruth                   LE JUGE : localise chaque lecture a l'octet pres
    cmd/tmp_compsweep                   la recette complete en une passe (index de hachage)
    cmd/tmp_filmmatch                   identifie le film par signature (59/60 contre 0/948)
    cmd/tmp_filmmanifest                manifeste FRAIS via /spectate (aucun chemin Go avant)
    cmd/tmp_findmatch                   identifiant complet du match par carte + date
    cmd/tmp_gabarit / tmp_gablocal      les deux gabarits et leurs compromis opposes
    cmd/tmp_recgap                      en-tete de record mesure, calibration d'idLow

NOTE D'EXPLOITATION : les entrées legacy .env.local sont VIDES depuis la migration ADR 0023.
Sur les 9 jetons du store canonique, 4 rendent AADSTS70000 (vieille app) — dont ceux de JGtm,
Madina97294 et Chocoboflor. Les 5 sains portent les xuid 2535405528935279, 2535409018618248,
2535413181053876, 2535430985184703, 2535460062932944.
