# Plan R7-b — Le corps d'image-cle du BIPEDE, BIT-EXACT

> Ecrit le 2026-08-17. Suite directe de `PLAN_R7A_IMAGE_CLE_BIPEDE_ETAT_COMPLET.md`, qui a
> tranche la FORME (le corps d'un record d'image-cle est un ETAT COMPLET : 102-104 % de la
> longueur reelle, contre 12-39 % pour la lecture « record NEW ») sans atteindre le BIT
> (0,51 %). Ce lot attaque ce qui reste : la bit-exactitude des deserialiseurs.
> Worktree `C:/Users/Guillaume/Projects/LevelUp-wt-kf35`, branche `wt/kf-biped-bit-exact`,
> base `wt/kf-biped-etat-complet` = `4807b8da2` (R3+R5+R6+R7-a).
> Execution sous le contrat du skill `plan-execution`.

---

## 1. Le critere — UN seul, avec son denominateur

**C1 : >= 95 % des 591 records `ti=35` bornes des 3 films oracles atterrissent BIT-EXACT sur
la frontiere du record suivant** (variante « etat complet » = 64 leaf nus, trous neutralises,
corruption-check du mode film ALLUME). Denominateurs R7-a, remesures : 184 (`000d5950`) +
209 (`00502e52`) + 198 (`07aa428d`) = **591**. Oracle de frontiere : `WalkKeyframeWorld`
(chaine disjointe de toute lecture de corps).

**C2** : a chaque phase, publier l'histogramme des points de decrochage et des composants de
desync — c'est lui qui ordonne la phase suivante. Un negatif chiffre est un livrable.

## 2. Etat des lieux — VERIFIE SUR PIECES le 2026-08-17

| deser | port Go (fichier:ligne) | fonction EXE | ce qui manque |
|---|---|---|---|
| `i60 simulation-state-component` | dispatch `traverse.go:861-866` ; corps `traverse.go:1154 consumeSimulationState` | thunk `142f02434` -> `FUN_142ED6D88` | rend `ported=simStateComplete` (**false** par defaut). Structure lue ; seul le **handle-tail** `FUN_14076e494` manque, garde par le predicat `FUN_140501798` |
| `i57 biped-spartan-ability-component` | `components_biped_ability.go:630` | `FUN_142f02810` -> `FUN_142f268c4` | **tag==3** -> `FUN_142f262d4`, non porte (gates sur octets d'etat runtime) |
| `i59 biped-spartan-ability-non-predicted-state` | `components_biped_ability.go:334` ; corps tag==3 `components_biped_anchor.go:130` | `FUN_142f02994` -> `FUN_142f25e90` | corps porte, mais `Zero3 != 0` et `Inner` hors {1,2} -> `ported=false` ; **les largeurs de position sont celles de la CARTE** |
| `i9 object-multiplayer-properties-component` | `components_batch7.go:26 consumeObjectMultiplayerProperties` | `FUN_1407d4c94` (TLV) | suspect n°1 en corruption OFF (25 % des decrochages). **N'est PAS `consumeMultiplayerPropertiesBlock`** (`FUN_14080cfe8`, chemin etat-par-defaut) : le lot poses `160e4ea7b` accuse ce DERNIER, absent de cette branche |
| `i63 biped-action-component` | `components_biped_ability.go:580 consumeBipedAction` | `FUN_142f027f4` -> `FUN_142f26a20` | `count2 = FUN_1409fe718(state,0x49)` = **popcount d'un masque RAM 73 bits, 0 bit dans le flux** — limite dure hors ligne, deja consignee `WALK_PORT_NOTES.md` |

### 2.1 LA DECOUVERTE DE PRE-PLAN — l'instrument R7-a lisait i0 aux largeurs d'une AUTRE carte

`i0` est le PREMIER composant de 100 % des records. Sur le chemin ABSOLU (celui de l'image-cle)
sa largeur d'axe vient de deux globaux de paquet que l'instrument R7-a **n'installe jamais** :

- `absoluteAxisW = 14` **uniforme** (`position_capture.go:205`) — alors que la capture CE donne
  `3 + 1 + 1 + (13+13+14) + 2 = 47 bits` sur Cliffhanger : l'uniforme rend **49**, soit
  **+2 bits sur CHAQUE record**, et bien pire sur une autre carte ;
- `WorldObjectPrecision` (`traverse.go:172`), defaut `{13,13,14}` = l'entree `cliffhanger` du
  catalogue — lue aussi par le corps tag==3 d'`i59` (`components_biped_anchor.go:139`).

Deux des trois films oracles ne sont pas Cliffhanger (R5/R6 : `00502e52` = axes 17/17/16).
**La mesure R7-a est donc plafonnee par une largeur fausse des le premier composant**, ce qui
explique la dominante « SOUS-LECTURE » (46-50 %) de son histogramme. Le correctif est
INSTRUMENTAL (`DetectI0Layout` par film -> `SetWorldObjectPrecisionFromLayout` +
`SetAbsoluteAxisW(0)` pour retomber sur les largeurs de la carte), restaure en `defer`.

## 3. Ordre d'attaque — par contribution mesuree (R7-a §2.1)

`i0` (largeurs de carte, 100 % des records) -> `i60` (83 % des desyncs) -> `i57` (9 %) ->
`i59` (6 %) -> `i9` / `i63` si l'ecart residuel les designe.

## 4. Phases

### Phase 0 — Plan (ce fichier)

- [x] 0.1 Critere, denominateurs, etat des lieux sur pieces, ordre d'attaque, gates, contrat.

### Phase 1 — `i0` aux largeurs de la carte, puis `i60`

- [x] 1.1 Instrument R7-b (`filmdec/keyframe_biped_bitexact_test.go`, garde `KF35_ROOT`),
      reutilisant les helpers `kf35*` de R7-a. Il installe par film le decoupage lu
      (`DetectI0Layout`) et restaure. **A/B obligatoire** : largeurs par defaut vs largeurs
      de la carte, meme variante, meme corpus.
- [x] 1.2 Decompiler `FUN_142ED6D88` et son handle-tail `FUN_14076e494` (Ghidra LECTURE SEULE,
      API HTTP `127.0.0.1:8089`). Consigner dans `WALK_PORT_NOTES.md`.
- [x] 1.3 Porter `i60` bit-exact si la lecture le permet ; sinon dire ce qui bloque, chiffre.
- [x] 1.4 Mesurer : % bit-exact et nouvelle repartition des decrochages.

### Phase 2 — `i57`, `i59`

- [x] 2.1 Decompiler `FUN_142f262d4` (corps tag==3 d'`i57`) ; porter ou refuter.
- [!] 2.2 `i59` : NON TRAITE, et justifie — la mesure retire son mobile. Son cout MEDIAN est
      de 5 bits (tag R(2) + queue R(3)) ; ses branches rares (`Zero3 != 0`, `Inner` hors {1,2})
      desynchronisent proprement et leur neutralisation coute 0 bit. Chercher son tag manquant
      n aurait pas deplace un residu de 400 a 500 bits. A rouvrir SI le residu descend sous 50 bits.
- [x] 2.3 Mesurer, republier les histogrammes.

### Phase 3 — `i9` et `i63` (SI l'ecart residuel les designe)

- [x] 3.1 `i9` : decompiler `FUN_1407d4c94` et confronter au port TLV. Si le deser est FAUX,
      corriger A LA SOURCE — **non-regression delta obligatoire** (suites `filmdec`,
      `replay`, `killsource` vertes).
- [x] 3.2 `i63` : la limite dure (`count2` = popcount RAM) est deja etablie ; la QUANTIFIER
      sur le corpus image-cle (part des records ou `count1 > 0`), pas la re-decouvrir.

### Phase 4 — Verdict

(statue ci-dessous, journal du 2026-08-17)
(statue ci-dessous, journal du 2026-08-17)

## 5. Gates — commandes exactes, a chaque cloture de phase

```
CGO_ENABLED=0 go build ./internal/analysis/...
CGO_ENABLED=0 go vet   ./internal/analysis/filmdec/
gofmt -l internal/analysis/filmdec/                         (doit etre vide)
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ -run '^TestKF35B' -v      (SKIP sans la garde)
CGO_ENABLED=0 KF35_ROOT=<...> go test ./internal/analysis/filmdec/ -run '^TestKF35B' -timeout 60m -v
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/
```

Toute modification d'un fichier PARTAGE du decodeur ajoute :
`CGO_ENABLED=0 go test ./internal/analysis/... -count=1` (non-regression delta).
**Jamais de verdict de gate lu a travers un tube** (piege R7-a §7.6).

## 6. Contrat — NON NEGOCIABLE

1. Fichiers PARTAGES du decodeur (`traverse.go`, `components_biped_*.go`, `default_state*.go`)
   modifies **uniquement** pour porter `i60` ou corriger un deser PROUVE faux, chacun avec sa
   preuve : avant/apres sur l'oracle R7-a **et** non-regression delta (suites vertes).
2. Aucune bosse de `SchemaVersion`. Aucune ecriture DuckDB, aucun rendu, aucune string UI,
   **aucune publication a l'artefact**, aucun balayage de masse (3 films, pas 951).
3. Lecture seule stricte hors du worktree. Ghidra : LECTURE SEULE, aucun rename, aucun script.
4. `GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-kf35/.gocache`, **une seule commande `go`
   a la fois**, `CGO_ENABLED=0`.
5. Bascules globales restaurees en `defer` ; `LockProcessDecode` tenu pour tout le test.
6. Pas de Python, pas d'emoji, seuils 500 L / 80 L / 5 parametres respectes.
7. Zero fix opportuniste hors perimetre : toute decouverte va au §7, non traitee.
8. Jamais `--no-verify`, jamais `git stash`, jamais `main`, jamais de merge. Commit par phase,
   `git push -u origin wt/kf-biped-bit-exact` apres chaque phase close.

## 7. Decouvertes — consignees, NON traitees

(a remplir en cours d'execution)

## 8. Journal d'execution

**2026-08-17 — Phase 0 CLOSE.** Reconnaissance de pre-plan : les 5 desers cites localises sur
pieces (§2), et la decouverte §2.1 (l'instrument R7-a ne posait pas les largeurs de carte)
reordonne l'attaque — `i0` passe devant `i60`.

**2026-08-17 — Phases 1 a 3, les PORTS. Trois corrections, chacune sur piece.**

1. **`i9 object-multiplayer-properties-component` : la porte etait INVERSEE.** `FUN_1407d4c94`
   lit le bloc TLV quand le bit vaut **0** (`if (cVar1 == '\0')`), et efface le champ
   (`*(dst+0xbc) = 0`) quand il vaut 1. `FUN_1406cf008` rend bien la VALEUR du bit (MSB du
   registre, `>> 0x3f`). Le port Go faisait l'inverse. Corrige dans `components_batch7.go`.
2. **`i60 simulation-state-component` : la queue `FUN_14076e494` est RESOLUE.** Son prédicat de
   garde `FUN_140501798` est VRAI PAR CONSTRUCTION (`FUN_1406d8678` construit en `dst+0x2c` un
   vecteur unitaire PERPENDICULAIRE a la direction de `dst+0x38` ; le predicat teste
   ‖v‖²≈1 / ‖v‖²≈1 / v·v≈0, constantes 1.0 / 0.0 / 1e-3 lues dans le binaire). Queue portee
   (`consumeSimStateHandleTail`, `traverse.go`) = le lecteur absolu MOINS `precHigh` et MOINS
   le `R(2)` de queue.
3. **`i57` branche `tag == 3` : la moitie portable l'est** (`consumeSpartanAbilityTag3`). L'autre
   moitie reste irreductible : elle est gardee par `dst[2]`, un octet d'ETAT RUNTIME.

**Le drapeau `simStateComplete` RESTE a `false` en production, et pour une raison NEUVE.** Ce
n'est plus la grammaire qui manque, c'est la SOURCE DES LARGEURS d'axe de la queue : `absAxisWFor`
retombe sur `absoluteAxisW`, un uniforme 14 qui n'est la largeur d'AUCUNE carte. Basculer le
defaut fait passer `TestGoldenMiniBobine` (killsource) de 0 a 2 « source appartenant a la
victime » PROPOSEES — 0 publiee dans les deux cas. Le golden a servi d'ISOLATION : ni la queue
d'i60, ni la polarite d'i9, ni `i57` ne le bougent, seul le `ported=true`. Critere de bascule
ecrit dans le code (kill-switch date).

**Non-regression delta : VERTE.** `go test ./internal/analysis/... ./internal/games/halo_infinite/film/... ./cmd/killsource/... -count=1`
-> 19 paquets `ok`, 0 `FAIL` (dont `filmdec`, `replay`, `killsource` et son golden mini-bobine).

**Mesure AVANT / APRES (v4 etat complet, trous neutralises, 591 records) :**

| grandeur (mediane, bits) | `000d5950` | `00502e52` | `07aa428d` |
|---|---|---|---|
| longueur REELLE du record | 2 765 | 2 777 | 2 781 |
| AVANT, consommee (corr. OFF) | 2 877 (104 %) | 2 890 (104 %) | 2 830 (102 %) |
| **APRES, consommee (corr. OFF)** | **2 351 (85 %)** | **2 474 (89 %)** | **2 478 (89 %)** |
| AVANT, ecart ABSOLU median (corr. OFF) | 920 | 757 | 927 |
| **APRES, ecart ABSOLU median (corr. OFF)** | **507** | **617** | **418** |
| AVANT, ecart ABSOLU median (corr. ON) | 370 | 474 | 499 |
| APRES, ecart ABSOLU median (corr. ON) | 680 | 778 | 846 |

**DEUX RESULTATS NEUFS, ET LE SECOND CORRIGE R7-a.**

- L'erreur est presque DIVISEE PAR DEUX et **change de signe** : on ne sur-lit plus, on
  SOUS-LIT d'environ 300 a 414 bits — un deficit STABLE, donc un BLOC MANQUANT, alors que R7-a
  concluait « pas de bloc constant manquant, la derive est per-composant ». Sa conclusion etait
  vraie de sa mesure, pas du format : elle etait masquee par la sur-lecture d'`i9`.
- **Le benefice du corruption-check du mode film etait un ARTEFACT.** R7-a mesurait « ALLUME,
  l'erreur mediane est divisee par 2 » ; une fois `i9` corrige, ALLUME est SYSTEMATIQUEMENT PIRE
  (680/778/846 contre 507/617/418). Le bit par composant compensait la sur-lecture d'`i9` — il ne
  la corrigeait pas. **La decouverte n°2 de R7-a est REFUTEE.**

**L'histogramme de decrochage se vide** (corr. OFF, largeurs de carte) : `i9` tombe de 47-50
franchissements a 0-6, `i60` disparait des composants neutralises, et il ne reste qu'UN suspect
au-dessus de 10 — **`i63 biped-action-component`, 38 a 45 franchissements sur ~200 (19-23 %)**,
tous les autres composants sous 10.

**2026-08-17 — Phase 4. VERDICT : C1 ECHOUE, plafond 0,54 %.**

Les quatre lectures encore debout, x largeurs par defaut / largeurs de carte, x
corruption-check OFF / ON = **48 passes**, 591 records bornes, 3 films. Longueurs REELLES
medianes : 2 765 / 2 777 / 2 781 bits.

| lecture | consommee MEDIANE | ecart ABSOLU MEDIAN | bit-exact |
|---|---|---|---|
| v4 etat complet, corr. OFF | 2 351 / 2 474 / 2 478 (85-89 %) | **507 / 617 / 418** | 0 / 0 / 0 |
| v2 etat par defaut + leaf, corr. OFF | 2 845 / 2 859 / 2 828 (102-103 %) | 707 / 705 / 635 | **1** / 0 / 0 |
| **v3 etat par defaut + porte + leaf, corr. OFF** | **2 829 / 2 773 / 2 822 (99,9-102,3 %)** | 574 / 616 / 536 | 0 / 0 / 0 |
| v4 etat complet, corr. ON | 3 102 / 3 190 / 3 239 | 680 / 778 / 846 | 0 / 1 / 1 |
| v2, corr. ON | 3 031 / 3 064 / 2 976 | **434 / 536 / 424** | 1 / 0 / 0 |
| v3, corr. ON | 3 133 / 3 004 / 3 012 | 560 / 449 / **412** | 0 / 0 / 0 |

**Plafond : 1 record sur 184 = 0,54 %** (seuil 95 %). R7-a plafonnait a 0,51 % : le taux
d'atterrissage n'a PAS bouge. Ce qui a bouge, c'est tout le reste.

- [x] 4.1 **C1 statue : ECHEC, 0,54 % contre 95 %.** Mais l'ecart median est passe de
      920/757/927 a 412-507 bits (**-45 a -55 %**), et la longueur MEDIANE d'une lecture
      « etat par defaut + porte + 64 leaf » tombe desormais a **99,9-102,3 %** de la longueur
      reelle (2 829/2 773/2 822 contre 2 765/2 777/2 781). L'echelle n'est plus seulement
      « du bon ordre » comme chez R7-a : elle est JUSTE A 2 % PRES en mediane. Ce qui reste
      n'est pas une grammaire manquante, c'est une DISPERSION : les medianes tombent juste et
      aucun record individuel n'atterrit.
- [x] 4.2 Lignes de registre au §10, entree thought_log au §9.

**CE QUE CA OUVRE.** Le verrou nomme par R7-a (« trois desers a brancher : i60, i57, i59 »)
est LEVE pour deux d'entre eux, et il n'etait pas le bon verrou : ces trois desers pesent
1, 2 et 5 bits en mediane. La vraie faute etait `i9`, et elle est corrigee EN PRODUCTION —
au-dela de l'image-cle, sur le chemin delta de tous les titres.

**CE QUE CA FERME.** Deux pistes sont mortes, chiffrees :
1. **Les largeurs d'axe de la carte ne sont PAS le levier de l'image-cle du bipede.** Les
   trois films ont bien trois decoupages differents (13/13/14, 17/17/16, 18/18/17, lus par
   `DetectI0Layout`) et R7-a les lisait tous a 14/14/14 — mais `i0` consomme **117 bits, la
   MEME mediane sur les trois films** : en image-cle il prend le chemin brut (vec3 IEEE
   96 bits), pas le chemin quantifie. Installer les vraies largeurs deplace l'ecart median de
   moins de 4 %. Piste close.
2. **Le corruption-check du mode film n'aide pas** : la decouverte n°2 de R7-a est un
   artefact d'`i9` (cf. journal des phases 1-3). Les deux reglages restent a mesurer cote a
   cote, mais « ALLUME divise l'erreur par deux » est FAUX.

**OU CA CASSE, CHIFFRE.** Apres correction, corruption OFF, largeurs de carte : plus AUCUN
composant ne desynchronise sauf `i57` et `i59` sur leurs branches rares. L'histogramme de
franchissement de frontiere ne garde qu'un seul suspect au-dessus de 10 sur ~200 records :
**`i63 biped-action-component`, 38 a 45 franchissements (19-23 %)**. Et `i63` **ne desync
jamais** sur ce corpus (il n'apparait dans aucune liste de composants neutralises) : sa boucle
`count1` se termine toujours sur des tags connus. C'est donc le DERNIER composant, le plus
large (196-210 bits), qui ABSORBE la derive amont — pas lui qui la cree. Le residu est un
DEFICIT DE LONGUEUR disperse, pas un deserialiseur casse.

## 9. Entree thought_log (redigee, NON ecrite par ce lot)

```
### [2026-08-17] Lot R7-b — La porte d i9 etait inversee ; le verrou nomme par R7-a n etait pas le bon

Statut : Complete (C1 echoue, mais trois deserialiseurs corriges/portes et deux pistes closes)

Decision technique. R7-a designait trois desers bloquants pour l image-cle du bipede (i60,
i57, i59). Decompile Ghidra en LECTURE SEULE (API HTTP du plugin, le pont MCP restant HS) :
(1) i60 simulation-state est ENTIEREMENT resolu — sa queue FUN_14076e494 est le lecteur
absolu moins precHigh et moins le R(2) final, et son predicat de garde FUN_140501798 est
VRAI PAR CONSTRUCTION (FUN_1406d8678 fabrique un vecteur unitaire perpendiculaire a la
direction decodee ; le predicat teste norme 1 / norme 1 / produit scalaire 0, constantes
1.0 / 0.0 / 1e-3 lues dans le binaire) ; (2) i57 tag==3 est portable a moitie, l autre
moitie etant gardee par un octet d ETAT RUNTIME ; (3) surtout, i9 object-multiplayer-
properties avait sa PORTE INVERSEE dans le port Go — FUN_1407d4c94 lit son bloc TLV quand
le bit vaut ZERO. Corrige a la source (chemin de production, pas seulement l image-cle).

Resultats observes. Sur 591 records ti=35 bornes, 3 films, 48 passes : l ecart absolu
median tombe de 920/757/927 a 412-507 bits (-45 a -55 %) et CHANGE DE SIGNE ; la longueur
mediane d une lecture << etat par defaut + porte + 64 leaf >> tombe a 99,9-102,3 % de la
longueur reelle. Le taux bit-exact, lui, ne bouge pas : 0,54 % contre 0,51 % chez R7-a,
pour un seuil de 95 % — C1 ECHOUE. Deux pistes closes, chiffrees : les largeurs d axe de la
carte ne sont PAS le levier (i0 consomme 117 bits, la MEME mediane sur trois cartes aux
decoupages differents : en image-cle il prend le chemin brut 96 bits) ; et le benefice
apparent du corruption-check du mode film mesure par R7-a etait un ARTEFACT de la
sur-lecture d i9 — une fois i9 corrige, l allumer est systematiquement PIRE.

Conclusion / prochaine etape. Le verrou de R7-a n etait pas le bon : i60/i57/i59 pesent 1,
2 et 5 bits en mediane. Apres correction, plus aucun composant ne desynchronise hors
branches rares d i57/i59, et un seul suspect depasse 10 franchissements sur ~200 : i63,
qui ne desync JAMAIS sur ce corpus — dernier et plus large composant, il ABSORBE la derive
amont, il ne la cree pas. Le residu est un deficit de longueur DISPERSE (medianes justes a
2 %, aucun record individuel juste). simStateComplete reste a false en production : plus
par manque de grammaire, mais parce que la queue d i60 tirerait ses largeurs d axe de
absoluteAxisW, un uniforme 14 qui n est la largeur d aucune carte. C est le prochain
chantier concret, et il est nomme.
```

## 10. Lignes de registre (redigees, NON ecrites par ce lot)

```
| 2026-08-17 | R7-b bipede image-cle BIT-EXACT (ti=35) | MESURE, C1 ECHOUE : 0,54 %
d'atterrissage bit-exact (seuil 95 %) sur 591 records, contre 0,51 % chez R7-a. MAIS trois
deserialiseurs corriges sur piece (i9 porte INVERSEE — bug de PRODUCTION, i60 queue
FUN_14076e494 resolue, i57 tag==3 a moitie porte), ecart absolu median -45 a -55 %
(920/757/927 -> 412/507), et longueur mediane a 99,9-102,3 % du reel. Condition de reprise :
le residu est un DEFICIT DISPERSE, pas un deser casse — attaquer la dispersion (distribution
des ecarts par record), pas un composant. |
| 2026-08-17 | AMENDEMENT a la ligne R7-a « image-cle = etat complet » | DEUX de ses
conclusions sont corrigees. (1) Son verrou nomme « i60 / i57 / i59, les 3 seuls blocages » est
FAUX en importance : ces desers pesent 1, 2 et 5 bits en mediane ; le vrai fautif etait i9
object-multiplayer-properties, dont la PORTE etait inversee. (2) Sa decouverte n°2 (« le
corruption-check du mode film divise l'erreur mediane par 2, il est probablement present dans
le payload type-2 ») est REFUTEE : c'etait un artefact de la sur-lecture d'i9 ; corrige,
l'allumer est systematiquement PIRE. Sa conclusion « pas de bloc constant manquant » l'etait
aussi de sa mesure seule : apres correction on SOUS-LIT de ~300-414 bits de facon stable. |
| 2026-08-17 | REPORT : basculer `simStateComplete` a true | i60 est desormais porte EN ENTIER
(grammaire etablie). Le defaut reste false parce que la queue tirerait ses largeurs d'axe
d'`absoluteAxisW`, un uniforme 14 qui n'est la largeur d'AUCUNE carte. Condition de reprise :
faire tirer au chemin absolu d'i0 les trois largeurs de la carte du match (comme
`replay.installWorldObjectPrecision` le fait deja pour `WorldObjectPrecision`). Temoin de
detection connu : `TestGoldenMiniBobine` passe de 0 a 2 « source appartenant a la victime »
PROPOSEES, 0 publiee des deux cotes. |
```

## 11. Decouvertes — consignees, NON traitees (complete le §7)

1. **`absoluteAxisW` est un uniforme 14 sur le chemin ABSOLU d'i0, et ce n'est la largeur
   d'aucune carte.** Les trois films oracles donnent 13/13/14, 17/17/16, 18/18/17
   (`DetectI0Layout`, mesure du 2026-08-17). La production installe deja les vraies largeurs
   sur `WorldObjectPrecision` (`replay.installWorldObjectPrecision`) mais **pas** sur ce
   second reglage. C'est le blocage nomme du report `simStateComplete`. NE PAS traiter ici.
2. **En image-cle, `i0` prend le chemin BRUT** : 117 bits de mediane, identiques sur les trois
   cartes — donc un vec3 IEEE 96 bits, pas une position quantifiee. Une image-cle stocke la
   position en pleine precision. Consequence directe : les bornes de carte ne sont pas
   necessaires pour lire la position d'un joueur a une image-cle. NE PAS traiter ici.
3. **Le corpus d'image-cle exerce des branches que le delta n'exerce pas** : `i57` et `i59`
   restent neutralises sur les trois films apres correction, sur des branches (`i57` a != 0,
   `i59` `Zero3 != 0` / `Inner` hors {1,2}) que le chemin delta ne rencontrait pas. Un etat
   complet dumpe des etats qu'un flux d'evenements ne transporte jamais. NE PAS traiter ici.
4. **Piege d'instrument (nouveau).** Rediriger la sortie d'un `go test` vers un fichier PUIS
   n'exposer que son `tail` fait perdre la mesure si la session est interrompue : le fichier de
   tache ne contient que la queue. Ecrire le log dans un chemin PERSISTANT du worktree
   (`.gocache/`), jamais dans `/tmp`. NE PAS traiter ici.
