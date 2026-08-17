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
- [ ] 2.2 `i59` : chercher ce qui manque au keyframe (tag/mode absent du chemin delta).
- [x] 2.3 Mesurer, republier les histogrammes.

### Phase 3 — `i9` et `i63` (SI l'ecart residuel les designe)

- [x] 3.1 `i9` : decompiler `FUN_1407d4c94` et confronter au port TLV. Si le deser est FAUX,
      corriger A LA SOURCE — **non-regression delta obligatoire** (suites `filmdec`,
      `replay`, `killsource` vertes).
- [ ] 3.2 `i63` : la limite dure (`count2` = popcount RAM) est deja etablie ; la QUANTIFIER
      sur le corpus image-cle (part des records ou `count1 > 0`), pas la re-decouvrir.

### Phase 4 — Verdict

- [ ] 4.1 C1 statue, chiffre. Si >= 95 % : dire ce que ca ouvre. Sinon : ou ca casse, chiffre.
- [ ] 4.2 Lignes de registre (dont l'AMENDEMENT de la ligne R7-a) + entree thought_log redigees.

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
