# Plan R7-c — L'encodage des valeurs en image-cle : un DRAPEAU DE CONTEXTE dans les lecteurs ?

> Ecrit le 2026-08-17. Suite de `PLAN_R7A_IMAGE_CLE_BIPEDE_ETAT_COMPLET.md` (la FORME) et de
> `PLAN_R7B_BIPEDE_IMAGE_CLE_BIT_EXACT.md` (les DESERIALISEURS). R7-b a corrige trois desers
> sur piece, divise l'ecart median par deux — et laisse le taux d'atterrissage bit-exact a
> **0,54 %**, avec un residu qui est une DERIVE DISPERSEE (p10 46-105 bits, p50 ~500, p90 a
> quatre ordres de grandeur, aucun palier) : chaque champ devie un peu, aucun n'est casse.
> Worktree `C:/Users/Guillaume/Projects/LevelUp-wt-kfflag`, branche `wt/kf-encodage-drapeau`,
> base `wt/kf-biped-bit-exact` = `9b269d7f7` (R3+R5+R6+R7-a+R7-b).
> Execution sous le contrat du skill `plan-execution`.

---

## 1. L'hypothese — et pourquoi elle est la suite logique de R7-b

Un residu qui n'est ni un bloc manquant ni un deser casse, mais « un peu partout », a une
cause de forme connue : **les MEMES deserialiseurs, lus sous un ENCODAGE DIFFERENT**. Le RE
externe l'ecrit noir sur blanc (`.ai/V7.5/film_re/GITHUB_RE_FINDINGS_EN.md`, section
statlines, « Caveat / what's left ») :

> *two encodings coexist — the keyframe (TYPE_2) uses a continuation varint, while the
> per-frame deltas use the 2-bit-selector encoding above.*

Et sa section positions ajoute que l'image-cle porte la position en **float32 brut** la ou le
delta la porte quantifiee — ce que R7-b a mesure independamment (`i0` = 117 bits de mediane,
la MEME sur trois cartes aux decoupages differents, donc un chemin BRUT 96 bits).

**H — Il existe un DRAPEAU DE CONTEXTE, lu par les lecteurs de valeurs eux-memes (pas un bit
du flux), qui bascule l'encodage delta vers absolu/brut.** Si oui, la grammaire d'image-cle
n'est pas une grammaire neuve : ce sont les 64 memes deser, joues avec le drapeau leve.

## 2. Le critere — C1, avec son denominateur

**C1 : un drapeau de contexte identifie dans au moins UN lecteur de valeur (adresse, offset
ou global, valeurs, encodage de chaque cote), ET le port Go du mode « baseline » fait passer
l'atterrissage bit-exact des 591 records `ti=35` bornes des 3 films oracles de 0,54 % a
>= 50 %** (palier intermediaire honnete ; cible 95 %).

Denominateurs R7-a/R7-b, inchanges : 184 (`000d5950`) + 209 (`00502e52`) + 198 (`07aa428d`)
= **591**. Oracle de frontiere : `WalkKeyframeWorld`. Variante de reference : v4 « etat
complet », trous neutralises, corruption-check du mode film ETEINT (R7-b : ALLUME est pire),
largeurs d'axe de la CARTE installees.

**C2** : un negatif chiffre est un livrable. Si aucun drapeau n'existe, ou s'il existe sans
effet mesurable, le dire avec les chiffres et nommer ou l'hypothese casse.

## 3. Etat des lieux — VERIFIE SUR PIECES le 2026-08-17

Canal Ghidra : instance PID 10104, **LECTURE SEULE**, pont MCP HS (« unknown ») ; API HTTP du
plugin `http://127.0.0.1:8089`, endpoint `GET /decompile_function?address=0x...` (le POST JSON
et le parametre `name` sont refuses — verifie ce jour). Un lot jumeau (R7-d) lit la meme
instance : aucun rename, aucun script, aucune analyse relancee.

| lecteur | ce que la decompile montre CE JOUR | drapeau de contexte ? |
|---|---|---|
| `FUN_1406cf008` (R(1)) | ne lit que `+0x2c` (compteur de bits), `+0x30` (registre), `+0x38` (bits en cache) | **non** |
| `FUN_140c18a1c` (int signe a selecteur 2 bits) | idem + `+0x10` (fin) / `+0x40` (curseur octet) | **non** |
| `FUN_14076e524` (vec3 quantifie) | globaux `DAT_144632be0` (largeur d'index) et tables `DAT_1445cc9e0` / `DAT_14462cbe0` — des TABLES, pas une bascule d'encodage | **non** |
| `FUN_1406cfe44` (`i0` position) | `cVar6 = FUN_14076f91c()` — **predicat SANS argument** : `DAT_144e61ea0 != 0` OU `DAT_145121140 == 1`. VRAI vers `FUN_1411b259c` = `FUN_1406d676c(...,0x60)` = **R(96) brut** ; FAUX vers `FUN_14076e524` = **vec3 quantifie** | **OUI** |

**Le port Go n'honore ce drapeau que sur UNE de ses deux branches.** `components_movement.go`
consulte `PositionFullPrecision` dans la branche PREDICTED-DELTA (`:197`) mais **pas** dans la
branche ABSOLUE (`:180-183`), qui appelle `consumeAbsoluteWithGate` sans condition. L'EXE, lui,
teste `FUN_14076f91c()` dans les DEUX. C'est un ecart de port etabli, pas une conjecture — et
c'est le premier fil de la phase 1.

Autres gates de contexte deja modelises cote Go (donc deja connus, a re-verifier et a
completer) : `PositionDeltaHasHandleTail` (descripteur runtime `precIndex != -1`),
`PositionFullPrecision`, `filmComponentCorruptionCheck` (mode film), `simStateComplete`.

## 4. Phases

### Phase 0 — Plan (ce fichier)

- [x] 0.1 Hypothese, critere C1 avec denominateur, etat des lieux sur pieces, phases, gates,
      contrat.

### Phase 1 — LIRE : la table des branches par CONTEXTE

- [ ] 1.1 Extraire des sources `filmdec` la liste des adresses EXE des deser du bipede et de
      leurs lecteurs de valeurs, decompiler chacune (Ghidra lecture seule, cache local).
- [ ] 1.2 Pour chacune, relever toute branche conditionnee par un OCTET ou CHAMP DE CONTEXTE
      (global `DAT_`, predicat sans argument, champ d'un parametre qui n'est pas le reader) —
      PAS par un bit du flux : adresse, offset/global, valeurs, encodage de chaque cote.
      Cible prioritaire : position, vitals, munitions, statlines, et la famille `FUN_1406cf008`.
- [ ] 1.3 Consigner la table dans `.ai/V7.5/killweapon/WALK_PORT_NOTES.md`, section neuve
      « image-cle — drapeau d'encodage ».

### Phase 2 — PORTER et MESURER

- [ ] 2.1 Ajouter au contexte Go un mode « baseline » (variable de paquet + setter, comme
      `SetFilmComponentCorruptionCheck`), **sans dupliquer les 64 deser** : le drapeau se lit
      la ou le jeu le lit.
- [ ] 2.2 Rejouer l'instrument R7-b en A/B baseline OFF/ON : pourcentage bit-exact, dispersion
      (parts a 8 / 16 / 64 / 256 bits), profil de decrochage par composant.
- [ ] 2.3 **Non-regression delta OBLIGATOIRE** : baseline OFF par defaut = comportement
      IDENTIQUE. Suites `filmdec`, `replay`, `killsource` vertes ; gradient
      `cmd/tmp_cleanframe` si le binaire existe dans cette branche.

### Phase 3 — Verdict

- [ ] 3.1 POSITIF (chiffres) : ce que ca ouvre. NEGATIF : ou l'hypothese casse, chiffre.
- [ ] 3.2 Entree thought_log redigee (NON ecrite) + lignes de registre, dont l'amendement a
      R7-b.

## 5. Gates — commandes exactes, a chaque cloture de phase

```
CGO_ENABLED=0 go build ./internal/analysis/...
CGO_ENABLED=0 go vet   ./internal/analysis/filmdec/
gofmt -l internal/analysis/filmdec/                          (doit etre vide)
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/ -count=1
CGO_ENABLED=0 KF35_ROOT=<...> go test ./internal/analysis/filmdec/ -run '^TestKF35C' -timeout 60m -v
```

Toute modification d'un fichier PARTAGE du decodeur ajoute :
`CGO_ENABLED=0 go test ./internal/analysis/... ./cmd/killsource/... -count=1`.
**Jamais de verdict de gate lu a travers un tube** (piege R7-a) : le log va dans `.gocache/`,
chemin PERSISTANT, et le verdict se lit sur une ligne `EXIT_*=0`.

## 6. Contrat — NON NEGOCIABLE

1. Fichiers PARTAGES du decodeur modifies UNIQUEMENT pour poser le drapeau la ou le jeu le
   lit, chacun avec sa preuve (adresse decompilee) ET sa non-regression delta.
2. Le mode baseline est **OFF par defaut** : aucun changement de comportement en production
   sans preuve. Aucune bosse de `SchemaVersion`, aucune ecriture DuckDB, aucun rendu, aucune
   string UI, **aucune publication a l'artefact**, aucun balayage de masse (3 films).
3. Lecture seule stricte hors du worktree (le corpus `data/cache/film_chunks` du worktree
   principal est LU, jamais ecrit). Ghidra : LECTURE SEULE, aucun rename, aucun script.
4. `GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-kfflag/.gocache`, **une seule commande `go`
   a la fois**, `CGO_ENABLED=0`.
5. Bascules globales restaurees en `defer` ; `LockProcessDecode` tenu pour tout le test.
6. Pas de Python, pas d'emoji, seuils 500 L / 80 L / 5 parametres respectes.
7. Zero fix opportuniste hors perimetre : toute decouverte va a la section 7, non traitee.
8. Jamais `--no-verify`, jamais `git stash`, jamais `main`, jamais de merge. Commit par phase,
   `git push -u origin wt/kf-encodage-drapeau` apres chaque phase close.
9. Pas d'ecriture dans `.ai/thought_log.md` (le superviseur s'en charge) ; l'entree est
   REDIGEE dans ce plan.

## 7. Decouvertes — consignees, NON traitees

(a remplir en cours d'execution)

## 8. Journal d'execution

(a remplir en cours d'execution)

**2026-08-17 — Phase 0 CLOSE.** Reconnaissance de pre-plan faite AVANT d ecrire le plan : canal
Ghidra retabli (le POST JSON du plugin refuse tout, seul `GET /decompile_function?address=0x...`
repond), quatre lecteurs de valeurs relus ligne a ligne, et UN drapeau de contexte deja trouve —
`FUN_14076f91c`, predicat sans argument sur `DAT_144e61ea0` / `DAT_145121140`, qui bascule le
lecteur de position entre vec3 QUANTIFIE et vec3 BRUT 96 bits. Le port Go ne le consulte que sur
sa branche delta. La phase 1 part de la.
