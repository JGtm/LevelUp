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
- [ ] 1.1 (a) La table : structure du bloc `chunk_00` contre l'entree de 0x104, ordre reel.
- [ ] 1.2 (b) L'en-tete par entite : `FUN_142e2bfd0` champ par champ, bornes de bits.
- [ ] 1.3 (c) Le controle : `FUN_14076cea8`, `FUN_1404f2b4c`, portee du drapeau film.
- [ ] 1.4 (e) `i0` : `FUN_14320678c` (ecrivain) et `FUN_14076e29c` (lecteur) face au port Go.
- [ ] 1.5 Consigner dans `WALK_PORT_NOTES.md`, section « la boucle d'etat complet, portee ».

### Phase 2 — PORTER et MESURER
- [ ] 2.1 `filmdec/keyframe_fullstate_loop.go` (+ son test) : la marche du jeu, chaque variable
      derriere une bascule. Reutiliser les deser existants et l'instrument R7-b/R7-d.
- [ ] 2.2 A/B a une seule variable, (a)..(e), 591 records, 3 films : % bit-exact, dispersion,
      histogramme des decrochages.
- [ ] 2.3 Non-regression si un fichier PARTAGE est touche.

### Phase 3 — Verdict
- [ ] 3.1 Statuer C1/C2, chiffres et denominateurs.
- [ ] 3.2 Lignes de registre + entree thought_log redigees (NON ecrites par ce lot).

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

(a remplir en cours d'execution)

## 10. Journal d'execution

(a remplir en cours d'execution)
