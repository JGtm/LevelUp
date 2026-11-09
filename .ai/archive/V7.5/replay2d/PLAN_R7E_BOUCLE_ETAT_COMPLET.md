# Plan R7-e — La boucle d'ETAT COMPLET du jeu, portee TELLE QUELLE sur le payload type-2

> Ecrit le 2026-08-17. Suite de `PLAN_R7D_ECRIVAIN_VTABLE.md` (C1a atteint, C1b echoue a
> 0,85 % — 5 records sur 591) et de `PLAN_R7C_ENCODAGE_DRAPEAU_IMAGE_CLE.md` (la portee
> `DAT_144e61ea0` existe ; allumee elle ANNULE l'atterrissage : la position d'image-cle est
> QUANTIFIEE aux largeurs de carte, 102/57/62 bits).
> Worktree `C:/Users/Guillaume/Projects/LevelUp-wt-kfloop`, branche `wt/kf-boucle-etat-complet`,
> base `wt/kf-ecrivain-vtable` = `9341e166a` (R3+R5+R6+R7-a/b/d).
> Execution sous le contrat du skill `plan-execution`.

---

## 1. Hypothese

R7-d a TROUVE la boucle d'etat complet du jeu (`FUN_142e2bfd0` -> `FUN_1428e2b68` ->
`FUN_142e2c690`) mais ne l'a pas PORTEE : il a mesure une largeur d'`i0` imposee sur la
lecture v4 heritee de R7-a. **H — la derive residuelle n'est pas dans les deserialiseurs de
composant (R7-d en a CONFIRME quatre sur cinq, largeur pour largeur) mais dans le CADRE que la
marche Go pose autour d'eux** : en-tete par entite, ordre des composants, mots de taille et
mots de controle. Portee telle quelle, la marche du jeu atterrit.

## 2. Critere — C1, avec palier

**C1** : la marche `FUN_142e2c690` portee (en-tete par entite + ordre de la table + controle
par composant + `i0` corrige + drapeau tranche) atterrit BIT-EXACT sur **>= 50 %** des 591
records `ti=35` bornes (palier), cible 95 %.
**C2** : tout negatif est chiffre, avec l'histogramme des decrochages et les adresses lues.
Denominateurs R7-a/R7-b/R7-c/R7-d inchanges : 184 (`000d5950`) + 209 (`00502e52`) +
198 (`07aa428d`) = **591**. Oracle de frontiere : `WalkKeyframeWorld`.

## 3. Etat des lieux — VERIFIE SUR PIECES le 2026-08-17 (pre-plan, Ghidra LECTURE SEULE)

Canal : instance PID 10104, API HTTP `127.0.0.1:8089`, `GET /decompile_function?address=0x...`
(seul endpoint fiable ; `/disassemble_function` sur une adresse sans fonction rend 200 Mo).
Le `HaloInfinite.exe` du disque est un STUB : tout passe par l'instance.

| fait relu ce jour | consequence |
|---|---|
| `FUN_142e2c690` essaie D'ABORD `comps[k]` (le meme index k que l'entree de table) et ne fait la recherche par NOM qu'en RATTRAPAGE | l'ordre de la table est CENSE etre celui du tableau de descripteurs : la decouverte n1 de R7-d se verifie au lieu de se supposer |
| entree de table = `[nom @ +0x00 .. +0x100][u32 niveau @ +0x100]`, pas de 0x104, 64 entrees, bloc a `ti*0x4100 + 8 + base` | 0x4100 = 64 x 0x104 : **exactement le bloc d'archetype de `chunk_00`** (`registry.go` : `registrySlotSize=260`, `archetypeBlockSlots=64`). `chunk_00` est la SERIALISATION de cette table |
| `FUN_142e29cf8` = **R(4)** (un seul quartet), et non « 2 x 4 bits » | l'en-tete par entite fait `32+32+32+4+8` = **108 bits** |
| en-tete complet : `R(32)` id, `R(32)` typeIndex, `R(32)`, `R(4)`, `R(8)`, puis `R(32) n1` (>0 : etat par defaut `vtable[0x60]`), `[R(32)` controle si drapeau film`]`, `R(32) n2` (>0 : `vtable[0x88]` a 0 bit, puis la boucle) | +44 bits d'en-tete et +64 bits de mots de taille par rapport a la marche Go actuelle |
| **`kfValidAnchor` n'accepte une ancre que si le mot de 32 bits a `q+32` vaut < 50** (`keyframe_world.go:70`) | sous `[id:32][field:26][ti:6]` cela veut dire `field26 == 0` ; sous l'en-tete RESEAU cela veut dire `typeIndex < 50`. **Les deux lectures sont INDISCERNABLES sur les ancres acceptees** : l'oracle ne REFUTE pas l'en-tete de 108 bits, contrairement a ce que R5 supposait |
| `corrOn = FUN_14076cea8()` rend `DAT_144c23326` si `FUN_1404f2b4c()` (mode film, valeur 2), sinon `DAT_1450e24e8` — deux globaux runtime | non decidable statiquement : c'est une VARIABLE de mesure, deja exposee (`filmComponentCorruptionCheck`) |
| le controle par composant est `si corrOn : R(1) ; si ce bit : R(32)` | c'est EXACTEMENT `consumeCorruptionCheck` — deja porte |

Deficit a combler (R7-b/R7-d) : longueur REELLE mediane 2 765 / 2 777 / 2 781 bits contre
2 350 / 2 420 / 2 456 consommes par la reference v4, soit **415 / 357 / 325 bits**. Les
`44 + 64 = 108` bits de cadre ci-dessus en couvrent le quart sans toucher a un seul deser.

## 4. Ordre d'attaque — cinq variables, allumees UNE A LA FOIS

- **(a) ORDRE de la table** : prouver que le bloc `chunk_00` EST la table de 64 x 0x104
  (offsets des champs, valeurs des niveaux) ; sinon, mesurer une permutation.
- **(b) EN-TETE par entite** : corps a `+108` au lieu de `+64`, et les deux `R(32)` de taille
  qui encadrent l'etat par defaut.
- **(c) CONTROLE `R(1)+R(32)`** par composant : A/B `corrOn` sous la NOUVELLE marche.
- **(d) `DAT_144e61ea0`** : les deux lectures portees, la mesure choisit, film par film
  (cherry-pick du port R7-c `a25e6812f`, deja non-regresse).
- **(e) `i0`** corrige selon l'ecrivain `FUN_14320678c` et le lecteur `FUN_14076e29c`.
  **Code de PRODUCTION** : correction seulement sur preuve (avant/apres sur l'oracle) ET
  non-regression delta ; sinon, lecture alternative dans MON fichier.

## 5. Phases

### Phase 0 — Plan (ce fichier)
- [x] 0.1 Hypothese, critere + palier, etat des lieux sur pieces, ordre d'attaque, gates, contrat.

### Phase 1 — LIRE
- [x] 1.1 (a) La table : structure du bloc `chunk_00` contre l'entree de 0x104, ordre reel.
- [x] 1.2 (b) L'en-tete par entite : `FUN_142e2bfd0` champ par champ, bornes de bits.
- [x] 1.3 (c) Le controle : `FUN_14076cea8`, `FUN_1404f2b4c`, portee du drapeau film.
- [x] 1.4 (e) `i0` : `FUN_14320678c` (ecrivain) et `FUN_14076e29c` (lecteur) face au port Go.
- [x] 1.5 Consigner dans `WALK_PORT_NOTES.md`, section « la boucle d'etat complet, portee ».

### Phase 2 — PORTER et MESURER
- [x] 2.1 `filmdec/keyframe_fullstate_loop.go` (+ son test) : la marche du jeu, chaque variable
      derriere une bascule. Reutiliser les deser existants et l'instrument R7-b/R7-d.
- [x] 2.2 A/B a une seule variable, (a)..(e), 591 records, 3 films : % bit-exact, dispersion,
      histogramme des decrochages.
- [x] 2.3 Non-regression si un fichier PARTAGE est touche.

### Phase 3 — Verdict
- [x] 3.1 Statuer C1/C2, chiffres et denominateurs.
- [x] 3.2 Lignes de registre + entree thought_log redigees (NON ecrites par ce lot).

## 6. Gates — commandes exactes, a chaque cloture de phase

```
CGO_ENABLED=0 go build ./internal/analysis/...
CGO_ENABLED=0 go vet   ./internal/analysis/filmdec/
gofmt -l internal/analysis/filmdec/                          (doit etre vide)
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/ -count=1
CGO_ENABLED=0 KF35_ROOT=<...> go test ./internal/analysis/filmdec/ -run '^TestKF7E' -timeout 90m -v
```
Toute modification d'un fichier PARTAGE ajoute
`CGO_ENABLED=0 go test ./internal/analysis/... ./cmd/killsource/... -count=1`.
**Jamais de verdict de gate lu a travers un tube** : log persistant sous `.gocache/`, verdict
lu sur une ligne `EXIT_*=0`.

## 7. Contrat — NON NEGOCIABLE

1. Ghidra **LECTURE SEULE** : aucun rename, aucun script, aucune analyse relancee.
2. Rien hors du worktree n'est ecrit. `C:/Users/Guillaume/Projects/LevelUp` et les autres
   `LevelUp-wt-*` sont en LECTURE SEULE (le corpus de films y est LU, jamais ecrit).
3. Fichiers PARTAGES du decodeur modifies **uniquement** sur preuve bit-exacte ET
   non-regression delta verte (20 paquets, killsource golden).
4. Aucune bosse de `SchemaVersion`, aucune ecriture DuckDB, aucun rendu, aucune string UI,
   **aucune publication a l'artefact**, aucun balayage de masse (3 films, pas 951).
5. `GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-kfloop/.gocache`, **une seule commande `go`
   a la fois**, `CGO_ENABLED=0`.
6. Bascules globales restaurees en `defer` ; `LockProcessDecode` tenu pour tout le test.
7. Pas de Python, pas d'emoji, seuils 500 L / 80 L / 5 parametres respectes.
8. Zero fix opportuniste hors perimetre : toute decouverte va au paragraphe 9, non traitee.
9. Jamais `--no-verify`, jamais `git stash`, jamais `main`, jamais de merge. Commit par phase,
   `git push -u origin wt/kf-boucle-etat-complet` apres chaque phase close.
10. Pas d'ecriture dans `.ai/thought_log.md` (le superviseur s'en charge).

## 8. Borne d'arret

Si apres les cinq variables (a)-(e) l'atterrissage reste **< 10 %**, c'est un NEGATIF MESURE :
l'ecrire avec l'histogramme des decrochages et ARRETER. **Ne pas chercher une 6e variable.**

## 9. Decouvertes — consignees, NON traitees

1. **Le niveau que `registry.go` sert est decale d'un cran** par rapport a celui que le jeu passe
   au deser (`Flags[k]` contre `Flags[k+1]`), et le champ `kind` qu'il lit en `+0` est nul sur les
   64 slots — ce n'est pas un champ. INERTE sur le chemin bipede mesure ici, mais le decodeur
   d'AUTRES archetypes pourrait, lui, dimensionner sur ce niveau. NE PAS traiter ici.
2. **`FUN_142e2bfd0` lit un `R(32)` de controle INCONDITIONNEL apres l'etat par defaut**, alors
   que le controle PAR COMPOSANT de `FUN_142e2c690` est `R(1)` puis `R(32)` conditionnel. Les
   deux sont modelises par la meme bascule `filmComponentCorruptionCheck` cote Go. NE PAS
   traiter ici.
3. **`FUN_14076cea8` depend de deux globaux runtime** (`DAT_144c23326` en mode film,
   `DAT_1450e24e8` sinon), et `FUN_1404f2b4c` teste une valeur de configuration en TLS : le
   drapeau film n'est PAS decidable statiquement. Toute conclusion sur le controle par composant
   restera une mesure, jamais une lecture. NE PAS traiter ici.
4. **La bascule `keyframeWriterI0Grammar` est une correction JUSTE mais non prouvee** (elle ne
   change aucun bit sur ce corpus). Son kill-switch porte une date cible de retrait au
   2026-10-31. NE PAS la basculer par defaut sans un corpus ou `h` vaut 1.

## 10. Journal d'execution

**2026-08-17 — Phase 0 CLOSE.** La reconnaissance de pre-plan a deja produit le fait qui ouvrait
le lot : `kfValidAnchor` n'accepte une ancre que si le mot de 32 bits a `q+32` vaut moins de 50,
ce qui veut dire `field26 == 0` sous l'en-tete de 64 bits et `typeIndex < 50` sous celui de 108.
Les deux en-tetes sont INDISCERNABLES sur les ancres acceptees : l'oracle de R3/R5 ne refutait
pas les 108 bits, contrairement a ce que R5 supposait.

**2026-08-17 — Phase 1 CLOSE. Trois relectures corrigent R7-d, et une mesure sur les octets
donne un fait neuf.**

1. **L'ordre n'est PAS une variable libre.** `FUN_142e2c690` essaie D'ABORD `comps[k]`, au MEME
   index que l'entree de table, et ne fait la recherche par NOM qu'en RATTRAPAGE. R7-d en
   deduisait « rien ne garantit que l'ordre soit celui de `arch.Components` » : c'est vrai en
   droit, mais le chemin rapide suppose l'egalite et la recherche par nom n'existe que pour
   absorber une derive de version.
2. **L'en-tete par entite fait 108 bits.** `FUN_142e29cf8` est un simple `R(4)`, pas « 2 x 4
   bits » : `R(32)` id, `R(32)` typeIndex, `R(32)`, `R(4)`, `R(8)`, puis `R(32) n1` et
   `R(32) n2` encadrant l'etat par defaut, plus un `R(32)` de controle INCONDITIONNEL (pas de
   `R(1)` de garde a cet endroit-la) quand le drapeau film est mis.
3. **LE DECALAGE DE NIVEAU.** L'entree que la boucle lit fait `0x104` octets : le NOM en `+0x00`,
   le NIVEAU en `+0x100`, le bloc a `ti*0x4100 + 8 + base` — le bloc d'archetype de `chunk_00` au
   bit pres. Les deux lectures placent les NOMS au MEME octet et le NIVEAU a des octets
   DIFFERENTS : `registry.go` sert `Flags[k]`, le jeu lit `Flags[k+1]`. Mesure sur les octets
   (`TestKF7ETableLayout`) : **les 64 slots ont `kind == 0`** — le premier `u32` que `registry.go`
   lit est toujours nul, ce qui est la queue de bourrage du nom precedent, pas un champ. Et
   **25 des 64 composants du bipede changent de niveau**, les MEMES 25 sur les trois films.
   Un niveau de 0 pour le composant de POSITION est implausible ; 1 l'est. C'est la lecture du
   jeu qui est la plus vraisemblable.

**2026-08-17 — Phase 2. LA MESURE, une variable a la fois.**

Instrument `keyframe_fullstate_loop{,_test}.go` (`WalkKeyframeFullState`), corpus ferme des 3
films oracles, 591 records `ti=35` bornes, largeurs de la carte installees, trous neutralises,
`simStateComplete` allume. Journaux persistants : `.gocache/r7e_loop2.log`,
`.gocache/r7e_layout.log`, `.gocache/r7e_levelshift.log`, `.gocache/r7e_nonreg.log`.

| lecture | `000d5950` (184) | `00502e52` (209) | `07aa428d` (198) |
|---|---|---|---|
| longueur REELLE mediane | 2 765 | 2 777 | 2 781 |
| REF (en-tete 64, rien) | 0 · med 2 350 · ecart **511** | 0 · 2 420 · **636** | 0 · 2 456 · **448** |
| (a) niveaux decales | 0 · 2 350 · 511 | 0 · 2 420 · 636 | 0 · 2 456 · 448 |
| (b1) en-tete 108 | 0 · **2 770** · 526 | 0 · 2 456 · 721 | 0 · 2 651 · 575 |
| (b2) 108 + 2 x R(32) | 0 · 3 096 · 697 | 0 · 2 901 · 695 | **1** · 2 952 · 817 |
| (b3) 108 + tailles + etat par defaut | 0 · 2 786 · 539 | 0 · 2 804 · 704 | **1** · 2 807 · 648 |
| (c) controle par composant | 0 · 3 114 · 669 | **1** · 3 196 · 778 | 0 · 3 239 · 811 |
| (d) portee `DAT_144e61ea0` | 0 · 2 396 · 543 | 0 · 2 474 · 660 | 0 · 2 458 · 526 |
| (e) i0 ecrivain | 0 · 2 350 · 511 | 0 · 2 420 · 636 | 0 · 2 456 · 448 |
| (b2+d+e) | 0 · 2 938 · 724 | **1** · 2 868 · 653 | 0 · 2 845 · 513 |
| (b3+c+e) tout | 0 · 3 491 · 1 007 | 0 · 2 969 · **424** | 0 · 3 175 · 536 |

Histogramme des decrochages, lecture de reference : sous-lecture **125 / 140 / 139** (68 / 67 /
70 %), depassement au DERNIER composant `i63` **45 / 38 / 41** (24 / 18 / 21 %), le reste
disperse. Aucun composant intermediaire ne concentre le decrochage.

**Non-regression : VERTE.** `go test ./internal/analysis/... ./internal/games/halo_infinite/film/...
./cmd/killsource/... -count=1` -> `EXIT_NONREG=0`, **20 paquets `ok`, 0 `FAIL`**. Les deux
bascules ajoutees (`keyframeWriterI0Grammar`, `keyframeBaselineScope` repris de R7-c) sont OFF par
defaut : aucun bit ne change en production.

**2026-08-17 — Phase 3. VERDICT — NEGATIF, ET LA BORNE D'ARRET EST ATTEINTE.**

- **C1 ECHOUE, largement.** Meilleure configuration : **1 record sur 591**. Toutes lectures
  confondues, **3 atterrissages bit-exact (0,51 %)** contre un palier a 50 % et une cible a 95 %.
  C'est en retrait de R7-d (5/591) et a l'iso de R7-c (3/591).
- **C2 tenu.** Chaque negatif est chiffre, avec ses adresses et son histogramme.
- **LA BORNE (section 8) EST ATTEINTE** : apres les cinq variables (a)-(e), l'atterrissage reste
  sous 10 %. **Aucune 6e variable n'a ete cherchee.**

Ce que la mesure DONNE quand meme, et qui n'etait pas connu :

1. **(a) et (e) sont INERTES au bit pres.** Le decalage de niveau, pourtant reel sur 25 des 64
   composants, ne change NI la mediane NI l'ecart : **aucun deserialiseur porte du bipede ne
   dimensionne quoi que ce soit sur le niveau du registre**. Et la correction d'`i0` selon
   l'ecrivain ne change rien parce que son bit `h` vaut 0 dans le cas dominant — la correction
   est JUSTE et sans effet sur ce corpus ; elle ne protegera que les records ou `h` vaut 1.
2. **(d) est TRANCHE, du cote de R7-c.** Portee allumee, l'ecart median monte a
   543 / 660 / 526 — **exactement les chiffres de R7-c**, obtenus par une marche differente :
   la fidelite du temoin est verifiee, et le payload type-2 est bien ecrit HORS de la portee.
   La contradiction que R7-d laissait ouverte est FERMEE.
3. **(b1) est le seul mouvement de forme qui vise juste, et sur UN seul film** : la longueur
   mediane de `000d5950` passe de 2 350 a 2 770 pour une longueur REELLE de 2 765 — cinq bits.
   Les deux autres films ne suivent pas (2 456 et 2 651 pour 2 777 et 2 781). Un en-tete de
   largeur FIXE ne peut donc pas etre la reponse complete.
4. **La derive n'est toujours pas localisee** : deux tiers de sous-lectures, un cinquieme de
   depassements au DERNIER composant, rien au milieu. Le cadre n'etait pas la cause.

## 11. Lignes de registre (redigees, NON ecrites par ce lot)

```
| 2026-08-17 | R7-e la boucle d'etat complet PORTEE | MESURE, C1 ECHOUE et la borne d'arret est
atteinte : **3 atterrissages bit-exact sur 591 (0,51 %)**, meilleure configuration 1/591, contre
un palier a 50 %. Les cinq variables du cadre ont ete portees et mesurees UNE A LA FOIS
(instrument `keyframe_fullstate_loop{,_test}.go`, `WalkKeyframeFullState`) : (a) ordre/niveaux,
(b) en-tete de 108 bits + deux `R(32)` de taille, (c) controle par composant, (d) portee
`DAT_144e61ea0`, (e) grammaire d'ecrivain d'`i0`. Aucune n'ecrase la dispersion : deux tiers des
marches sous-lisent, un cinquieme depasse au DERNIER composant (`i63`), rien au milieu. Le CADRE
n'etait pas la cause — la derive est DANS les deserialiseurs. |
| 2026-08-17 | R7-e, l'en-tete de 108 bits que l'oracle ne refutait pas | NEUF : `kfValidAnchor`
n'accepte une ancre que si le mot de 32 bits a `q+32` vaut moins de 50 — `field26 == 0` sous
`[id:32][field:26][ti:6]`, `typeIndex < 50` sous l'en-tete RESEAU de `FUN_142e2bfd0`. **Les deux
lectures sont INDISCERNABLES sur les ancres acceptees** : l'oracle valide par R3/R5 sur 249/250
entites ne refutait pas les 108 bits. Mesure : l'en-tete de 108 bits porte la longueur mediane de
`000d5950` de 2 350 a **2 770** pour un reel de **2 765** — cinq bits — mais les deux autres films
ne suivent pas (2 456 et 2 651 pour 2 777 et 2 781). Un en-tete de largeur FIXE n'est pas la
reponse complete. Corrige aussi R7-d : `FUN_142e29cf8` est un simple `R(4)`, pas 2 x 4 bits. |
| 2026-08-17 | R7-e, le NIVEAU du registre est decale d'un cran | L'entree que `FUN_142e2c690`
lit fait `0x104` octets — NOM en `+0x00`, NIVEAU en `+0x100`, bloc a `ti*0x4100 + 8 + base` : le
bloc d'archetype de `chunk_00` au bit pres. Les deux lectures placent les NOMS au MEME octet et le
NIVEAU a des octets DIFFERENTS : `registry.go` sert `Flags[k]`, le jeu lit `Flags[k+1]`. Preuve
sur les octets : **les 64 slots du bipede ont `kind == 0`** (le premier `u32` de `registry.go` est
la queue de bourrage du nom precedent, pas un champ) et **25 des 64 composants changent de
niveau**, les memes 25 sur les trois films. INERTE a la mesure — aucun deser porte du bipede ne
dimensionne sur le niveau — donc consigne, non traite. |
| 2026-08-17 | R7-e, `DAT_144e61ea0` TRANCHE et l'ordre RABATTU | Deux fils de R7-d se ferment.
(1) La portee allumee porte l'ecart median a **543 / 660 / 526**, soit EXACTEMENT les chiffres de
R7-c obtenus par une marche differente : temoin fidele, et le payload type-2 est ecrit HORS de la
portee — position QUANTIFIEE, pas de vec3 brut. (2) L'ORDRE n'est pas une variable libre :
`FUN_142e2c690` essaie D'ABORD `comps[k]` au MEME index et ne cherche par NOM qu'en rattrapage. La
decouverte n1 de R7-d est amendee. Condition de reprise : AUCUNE sur le cadre — la borne d'arret
du plan est atteinte. |
```

## 12. Entree thought_log (redigee, NON ecrite par ce lot)

```
### [2026-08-17] Lot R7-e — La boucle d etat complet portee : le cadre n etait pas la cause

Statut : Complete (C1 echoue a 0,51 %, borne d arret atteinte, negatif chiffre)

Decision technique. R7-d avait TROUVE la boucle d etat complet du jeu sans la PORTER. Ce lot la
porte entierement et mesure ses cinq variables une a la fois, sur les memes 591 records ti=35
bornes que R7-a/b/c/d. Trois relectures Ghidra corrigent R7-d au passage : FUN_142e2c690 essaie
d abord le descripteur au MEME index que l entree de table et ne cherche par nom qu en
rattrapage — l ordre n est donc pas une variable libre ; FUN_142e29cf8 est un simple R(4), ce qui
porte l en-tete par entite a 108 bits ; et la piece qui ouvrait le lot, kfValidAnchor n accepte
une ancre que si le mot de 32 bits a q+32 vaut moins de 50, ce qui veut dire field26 nul sous une
lecture et typeIndex inferieur a 50 sous l autre : les deux en-tetes sont indiscernables sur les
ancres acceptees, l oracle de R3/R5 ne refutait pas les 108 bits. Un fait neuf est sorti d une
mesure sur les octets du registre : l entree que la boucle lit place le nom en +0 et le niveau en
+0x100, la ou registry.go suppose un kind et un flags avant le nom — les noms tombent au meme
octet dans les deux lectures, le niveau non. Les 64 slots ont un kind nul, et 25 des 64
composants du bipede changent de niveau, les memes 25 sur les trois films.

Resultats observes. Trois atterrissages bit-exact sur 591 (0,51 %), meilleure configuration 1 sur
591, pour un palier a 50 %. Le decalage de niveau et la correction d i0 selon l ecrivain sont
INERTES au bit pres : aucun deserialiseur porte du bipede ne dimensionne quoi que ce soit sur le
niveau du registre, et le bit h d i0 vaut 0 dans le cas dominant. La portee DAT_144e61ea0 allumee
donne un ecart median de 543/660/526 — exactement les chiffres de R7-c, obtenus par une marche
differente : le temoin est fidele et le payload du film est bien ecrit hors de la portee. Le seul
mouvement de forme qui vise juste est l en-tete de 108 bits, et sur un seul film : la longueur
mediane de 000d5950 passe de 2 350 a 2 770 pour un reel de 2 765, cinq bits — les deux autres
films ne suivent pas. Histogramme des decrochages inchange par toutes les variables : deux tiers
de sous-lectures, un cinquieme de depassements au DERNIER composant, rien au milieu.

Conclusion / prochaine etape. La borne d arret du plan est atteinte et respectee : apres les cinq
variables, l atterrissage reste sous 10 %, aucune sixieme n a ete cherchee. Ce que ce lot ferme
est une famille entiere d hypotheses — le CADRE du record (en-tete, ordre, niveaux, mots de
taille, mots de controle, portee de precision) n est pas la cause de la derive. Ce qui reste est
ce que R7-b nommait deja et que ce lot confirme sans le resoudre : une derive DISPERSEE dans les
deserialiseurs eux-memes, que ni la lecture du jeu ni celle de l ecrivain n expliquent. Aucun
fichier partage n a change de comportement : les deux bascules ajoutees sont OFF par defaut et la
non-regression delta est verte, 20 paquets, zero FAIL.
```
