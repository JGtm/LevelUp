# Plan R7-d — L'ECRIVAIN de l'image-cle, par les vtables des composants

> Ecrit le 2026-08-17. Suite de `PLAN_R7B_BIPEDE_IMAGE_CLE_BIT_EXACT.md` (C1 echoue,
> 0,54 % d'atterrissage bit-exact, residu = DEFICIT DISPERSE) et de
> `PLAN_R6_FILE_PAR_ENTITE.md` (le jeu ne RELIT jamais le payload type-2 : il n'existe pas
> de « lecteur d'image-cle » a copier ; la recherche de l'ecrivain par CHAINES
> (`saved_games`, `FilmBlock*`) et par xrefs de `FUN_142f2e174` etait NEGATIVE).
> Worktree `C:/Users/Guillaume/Projects/LevelUp-wt-kfwriter`, branche `wt/kf-ecrivain-vtable`,
> base `wt/kf-biped-bit-exact` = `9b269d7f7`. Execution sous le contrat `plan-execution`.

---

## 1. Hypothese

Chaque classe de composant a une VTABLE dont R5 n'avait lu que la case de LECTURE
(`vtable[0x28]`, thunk `FUN_14076ce9c` -> `vtable[0x30]` = deser feuille). **Les cases
VOISINES portent la SERIALISATION (ecriture).** Decompiler l'ECRITURE et la confronter a la
LECTURE, composant par composant, dit exactement ou les deux grammaires divergent — et la
grammaire d'ECRITURE est celle du corps d'image-cle (un etat complet est ECRIT, pas relu).

## 2. Critere — C1, avec palier

**C1a (grammaire)** : pour >= 1 composant du bipede, la methode d'ECRITURE est identifiee
(adresse), sa grammaire est ecrite champ par champ face a la grammaire de LECTURE, et elle
EXPLIQUE un ecart mesure par R7-b.

**C1b (mesure)** : si la variante « ecriture » est portee en LECTURE pour les composants
divergents, l'atterrissage bit-exact monte de 0,54 % a **>= 50 %** (palier), cible 95 %.

**C2** : tout negatif est chiffre et publie (adresses lues, cases inspectees).

## 3. Etat des lieux — VERIFIE SUR PIECES le 2026-08-17 (pre-plan)

Acces Ghidra : instance PID 10104, **API HTTP `127.0.0.1:8089`, LECTURE SEULE** (le pont
`mcp__ghidra__*` reste HS). Methode : dump de `.rdata` (`0x143606000-0x144395200`,
14 217 728 o, 55 lectures `read_memory`) puis recherche locale du pointeur 8 o.
Le `HaloInfinite.exe` du disque est un STUB (`.text` 2 Mo contre 54 Mo dans Ghidra) : toute
lecture statique passe par Ghidra, jamais par le fichier.

| fait | preuve |
|---|---|
| Chaque deser de composant a **exactement UN** pointeur dans `.rdata` : sa case de vtable | balayage 8 o aligne, 52 composants du bipede |
| La base de vtable est confirmee par xref DATA depuis `.data` (l'objet descripteur) | `0x143d0ce00` <- `0x144747368` ; `0x143d0ce58` <- `0x1447472e0` |
| `vtable[0x08]` = getter de NOM (`lea rax,[rip+X]; ret`) | `0x141177560` -> `0x143c98cf0` = `"biped-action-component"` |
| Layout UNIFORME sur 51/52 composants : `+0x00` dtor, `+0x08` nom, `+0x10` `ret false`, **`+0x18` per-composant**, `+0x20` `int3`, `+0x28` thunk `FUN_14076ce9c`, `+0x30` **DESER**, `+0x38` `ret 1`, `+0x40` `*p=0`, `+0x48` zero-16 | sweep des 52 desers |
| **`vtable[0x18]` EST L'ECRIVAIN** | i60 : `FUN_142f04e2c` (`sub = etat+0xa48`) -> `FUN_142edd10c` = `W(1) *(sub+0x28)` puis, si non nul, la suite — MIROIR EXACT du lecteur R7-b (`R(1) -> dst+0x28`, si 0 fin). i63 : `FUN_142f05144` (`sub = etat+0xaa8`) -> `FUN_142f27a68` = `W(4) *(sub+0x18)` |
| Primitives d'ecriture identifiees | `FUN_1407edaf4(w, "nom", v)` = **W(32)** (le nom du champ est un parametre MORT, garde en retail) ; `FUN_1406d49c4(w, ., b)` = **W(1)** ; etat d'ecrivain : `+0x2c` bits ecrits, `+0x30` accumulateur, `+0x38` bits tenus, `+0x40` sortie, `+0x10` fin |
| L'archetype porte la MEME paire | vtable bipede `0x143737178` (confirmee 2x : `+0x60` = `FUN_140f44c38` deser d'etat par defaut, `+0xa0` = `FUN_14076ca20` masque par defaut). **`+0x58` = `FUN_142f14a68` = l'ECRIVAIN de l'etat par defaut**, avec des NOMS de champs (`"player-representation-name"`, `"customization-source-participant"`) |

Corpus de mesure (lecture seule) : `C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks`,
films `000d5950` / `00502e52` / `07aa428d`, 591 records `ti=35` bornes (denominateurs R7-b).

## 4. Ordre d'attaque

Trois composants, choisis par leur poids dans la derive R7-b : **`i0`** (position, 100 % des
records, 117 bits lus contre 96 attendus), **`i63`** (biped-action, le plus large, 19-23 %
des franchissements de frontiere), **`i9`** (object-multiplayer-properties, la faute de
production corrigee par R7-b). `i60` est deja lu (pre-plan) et sert de temoin de symetrie.

## 5. Phases

### Phase 0 — Plan (ce fichier)
- [x] 0.1 Hypothese, critere + palier, etat des lieux sur pieces, ordre d'attaque, gates, contrat.

### Phase 1 — LIRE l'ecrivain
- [x] 1.1 `i0` : vtable ENTIERE (toutes les cases), decompile de chaque case non triviale,
      grammaire d'ECRITURE face a la grammaire de LECTURE, champ par champ.
- [x] 1.2 `i63` : idem.
- [x] 1.3 `i9` : idem.
- [x] 1.4 Consigner dans `WALK_PORT_NOTES.md`, section « ecrivain par vtable ».

### Phase 2 — MESURER
- [ ] 2.1 Porter les divergences comme LECTURE ALTERNATIVE, gardee, dans un fichier a moi
      (`filmdec/keyframe_writer_grammar.go` + son test) — aucun fichier partage modifie.
- [ ] 2.2 Rejouer l'instrument R7-b (`TestKF35B*`) : decrochages par composant et % bit-exact
      global, AVANT / APRES, meme corpus, memes denominateurs.

### Phase 3 — Verdict
- [ ] 3.1 Statuer C1a et C1b, chiffres et denominateurs.
- [ ] 3.2 Lignes de registre + entree thought_log redigees (NON ecrites par ce lot).

## 6. Gates — commandes exactes, a chaque cloture de phase

```
CGO_ENABLED=0 go build ./internal/analysis/...
CGO_ENABLED=0 go vet   ./internal/analysis/filmdec/
gofmt -l internal/analysis/filmdec/                          (doit etre vide)
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ -run '^TestKF7D' -v        (SKIP sans garde)
CGO_ENABLED=0 KF35_ROOT=<...> go test ./internal/analysis/filmdec/ -run '^TestKF7D|^TestKF35B' -timeout 60m -v
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/
```

Toute modification d'un fichier PARTAGE ajoute `CGO_ENABLED=0 go test ./internal/analysis/... -count=1`.
**Jamais de verdict de gate lu a travers un tube** (piege R7-a §7.6) : log persistant sous `.gocache/`.

## 7. Contrat — NON NEGOCIABLE

1. Ghidra **LECTURE SEULE** : aucun rename, aucun script, aucune analyse relancee. Un lot
   JUMEAU (R7-c) lit la meme instance : lecture seule des deux cotes.
2. Rien hors du worktree n'est ecrit. `C:/Users/Guillaume/Projects/LevelUp` et les autres
   `LevelUp-wt-*` sont en LECTURE SEULE (le corpus de films y est lu, jamais ecrit).
3. Fichiers PARTAGES du decodeur (`traverse.go`, `components_*.go`, `default_state*.go`)
   modifies **uniquement** sur preuve bit-exacte ET non-regression delta (suites vertes).
4. Aucune bosse de `SchemaVersion`, aucune ecriture DuckDB, aucun rendu, aucune string UI,
   **aucune publication a l'artefact**, aucun balayage de masse (3 films, pas 951).
5. `GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-kfwriter/.gocache`, **une seule commande
   `go` a la fois**, `CGO_ENABLED=0`.
6. Bascules globales restaurees en `defer` ; `LockProcessDecode` tenu pour tout le test.
7. Pas de Python, pas d'emoji, seuils 500 L / 80 L / 5 parametres respectes.
8. Zero fix opportuniste hors perimetre : toute decouverte va au §9, non traitee.
9. Jamais `--no-verify`, jamais `git stash`, jamais `main`, jamais de merge. Commit par
   phase, `git push -u origin wt/kf-ecrivain-vtable` apres chaque phase close.
10. Pas d'ecriture dans `.ai/thought_log.md` (le superviseur s'en charge).

## 8. Borne d'arret

Si apres 3 composants lus AUCUNE methode d'ecriture n'est identifiable (vtables sans
ecrivain, serialisation par un chemin generique hors composant), c'est un NEGATIF MESURE :
l'ecrire avec les adresses lues et ARRETER. (Le pre-plan a deja leve cette borne pour `i60`
et `i63` : la case `+0x18` existe et ecrit des bits.)

## 9. Decouvertes — consignees, NON traitees

(a remplir en cours d'execution)

## 10. Journal d'execution

(a remplir en cours d'execution)

**2026-08-17 — Phase 0 CLOSE.** La reconnaissance de pre-plan a deja tranche la question de
FORME que le lot devait poser : la vtable de composant a bien une case d ECRITURE, `+0x18`,
et elle est le MIROIR de la case de LECTURE `+0x30`. La borne d arret du §8 est donc levee
avant la phase 1 ; ce que la phase 1 doit produire, c est la grammaire champ par champ.

**2026-08-17 — Phase 1 CLOSE. L hypothese est VRAIE, et elle se referme sur UNE case.**

La vtable d un descripteur de composant compte 10 cases et **exactement deux touchent le flux
de bits** : `+0x18` ECRIT, `+0x30` LIT (via le thunk `+0x28`). Les huit autres sont un getter de
nom, trois stubs constants, un `int3`, un accesseur et un zeroteur de 16 octets. Meme paire au
niveau ARCHETYPE : vtable du bipede `0x143737178`, `+0x58` ecrit l etat par defaut, `+0x60` le lit.
Methode et layout complets dans `WALK_PORT_NOTES.md`, section « L ECRIVAIN PAR VTABLE ».

Trois des quatre composants lus **confirment le lecteur porte, largeur pour largeur** :
`i60` (`FUN_142edd10c`), `i63` (`FUN_142f27a68` — et son `count2` sort du MEME popcount RAM
`FUN_1409fe718(state,0x49)` cote ecriture : la limite dure de R7-b est dans le FORMAT, pas dans
la lecture), `i9` (`FUN_1420ab570` — la polarite corrigee par R7-b et son `R(5)` de tag sont
confirmes independamment). Ces trois-la ne creent aucune derive.

`i0` DIVERGE, et son ecrivain est celui de l ETAT COMPLET (reference de base NULLE, drapeau brut
a 0) — donc exactement celui de l image-cle. Trois ecarts, tous dans le meme sens :
le 3e bit n est pas un `precHigh` qui supprime la charge utile mais la porte de la QUEUE DE
HANDLE, la charge utile est ecrite QUOI QU IL ARRIVE, et le champ de 2 bits est INCONDITIONNEL et
vient EN DERNIER. Consequence de forme : en image-cle `i0` prend le chemin QUANTIFIE, de largeur
`2 + 1 + 1 + [idxW] + 3 x axisW + [queue] + 2` — **dependante de la carte, ~46 bits sur un
decoupage 13/13/14** — la ou le lecteur porte en consomme 117 en mediane. La decouverte n2 de
R7-b (« i0 brut, 96 bits, identique sur trois cartes ») decrivait le LECTEUR, pas l ECRIVAIN.
