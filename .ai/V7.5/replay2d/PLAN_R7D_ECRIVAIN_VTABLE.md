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
- [~] 2.1 Porter les divergences comme LECTURE ALTERNATIVE, gardee, dans un fichier a moi
      (`filmdec/keyframe_writer_grammar.go` + son test) — aucun fichier partage modifie.
- [x] 2.2 Rejouer l'instrument R7-b (`TestKF35B*`) : decrochages par composant et % bit-exact
      global, AVANT / APRES, meme corpus, memes denominateurs.

### Phase 3 — Verdict
- [x] 3.1 Statuer C1a et C1b, chiffres et denominateurs.
- [x] 3.2 Lignes de registre + entree thought_log redigees (NON ecrites par ce lot).

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

**2026-08-17 — Phase 1bis. LA PISTE DU LOT JUMEAU R7-c, DEROULEE : la boucle d ETAT COMPLET du
jeu existe, et elle n a PAS de masque.**

R7-c a nomme `FUN_1428e2a04` -> `FUN_1428e2a9c` -> `FUN_142e2bfd0` (lecteur d etat complet, paquet
`FUN_142988338(..., 0x10, 0)`). Deroulee jusqu au bout : `FUN_142e2bfd0` lit un en-tete par entite
(`R(32)` id, `R(32)` typeIndex, `R(32)`, `FUN_142e29cf8`, `R(8)`, puis `R(32)` de taille), appelle
`vtable[0x60]` (etat par defaut) sous `DAT_144e61ea0 = 1`, un mot de controle `R(32)` si le drapeau
film est mis, `R(32)` de taille, `vtable[0x88]` (masque par defaut, 0 bit), puis
`FUN_1428e2b68` -> **`FUN_142e2c690`, LA BOUCLE DE COMPOSANTS D ETAT COMPLET**.

`FUN_142e2c690` (detail complet dans `WALK_PORT_NOTES.md` section 6) tranche trois questions :

1. **AUCUN MASQUE DE PRESENCE** : 64 entrees d une table FIXE de `0x104` octets par archetype,
   chacune deserialisee via `vtable[0x28]`, plus un `R(1) + R(32)` de controle par composant quand
   le drapeau film est mis. La variante « 64 leaf nus » de R7-a/R7-b est STRUCTURELLEMENT LA BONNE.
2. **L ORDRE suit la TABLE, pas l archetype** : le descripteur est retrouve PAR NOM (`vtable[0x08]`),
   par recherche lineaire. Rien ne garantit que cet ordre soit celui de `arch.Components` — source
   de derive jamais envisagee par R7-a ni R7-b.
3. **`DAT_144e61ea0` est leve pour TOUTE la boucle**, et `FUN_14076f91c()` rend
   `(DAT_144e61ea0 != 0) || (DAT_145121140 == 1)` : c est la porte de PLEINE PRECISION, cote lecture
   ET cote ecriture. **Cela REFUTE ma consequence de forme de la phase 1** (« en image-cle `i0` est
   quantifie, ~46 bits ») et RETABLIT la decouverte n2 de R7-b : dans la portee d etat complet
   `i0` ecrit et lit un vec3 BRUT de 96 bits, soit `1+1+1+96+queue+2` = 101 + queue = **117 bits**
   avec une queue de 16, ce qui est exactement la mediane mesuree sur les trois films.
   Ce n est pas une propriete d `i0`, c est la PORTEE que la boucle leve autour de lui.

Reserve : ce lecteur travaille sur un paquet `0x10` (snapshot RESEAU), pas sur le bloc type-2 du
film — R6 a etabli que ce dernier n a aucun consommateur. Et R7-c mesure sur le payload type-2 une
position QUANTIFIEE aux largeurs de carte (102/57/62 bits). **La contradiction est nommee et tient
en une variable** : `DAT_144e61ea0` decide entre 96 bits bruts et `3 x axisW` quantifies. C est
elle qu il faut trancher au lot suivant.

**2026-08-17 — Phase 2. LA MESURE.**

Instrument : `keyframe_writer_grammar_test.go` (`TestKF7DWriterI0`, `TestKF7DWriterI0Profile`),
garde `KF35_ROOT`, corpus ferme des 3 films oracles, variante v4 (etat complet, trous neutralises),
corruption-check OFF, largeurs de la CARTE installees, balayage de la largeur d `i0` par
`SetCalibratedWidth` (hook DEJA exporte). **Aucun fichier partage du decodeur n est modifie.**
Denominateurs R7-b inchanges : 184 + 209 + 198 = 591 records bornes. Journal persistant :
`.gocache/r7d_sweep.log` (2 114 s) et `.gocache/r7d_profile.log`.

Profil de depart : `i0` consomme **117 bits de mediane sur les TROIS films** (0 franchissement de
frontiere), alors que les trois decoupages lus dans les films different (13/13/14, 17/17/16,
18/18/17). Longueurs REELLES medianes (R7-b) : 2 765 / 2 777 / 2 781.

| film (bornes) | REFERENCE, lecteur porte | largeur d ecrivain QUANTIFIEE | MEILLEUR du balayage |
|---|---|---|---|
| `000d5950` (184) | 0 exact · med 2 350 · ecart 511 | w=46 : 0 exact · med 2 771 · ecart 624 | **w=81 : 1 exact (0,54 %)** · med 2 527 · ecart 416 |
| `00502e52` (209) | 0 exact · med 2 420 · ecart 636 | w=56 : 0 exact · med 2 646 · ecart 709 | **w=54 : 2 exacts (0,96 %)** · med 2 808 · ecart 387 |
| `07aa428d` (198) | 0 exact · med 2 456 · ecart 448 | w=59 : 0 exact · med 166 998 (queue lourde) | **w=62 : 2 exacts (1,01 %)** · med 2 801 · ecart 504 |

**CE QUE LA MESURE DONNE.** (1) L atterrissage bit-exact passe de **0 / 591 a 5 / 591 (0,85 %)** :
c est un mouvement REEL — la reference ne fait atterrir AUCUN record sur aucun des trois films —
mais il est a deux ordres de grandeur du seuil. (2) Les meilleures largeurs des deux derniers films
(**54** et **62**) tombent a 2 et 3 bits de la largeur QUANTIFIEE predite par l ecrivain (56 et 59),
et le **62** du troisieme film est exactement la valeur que R7-c mesure sur le payload type-2 —
deux chemins independants qui se rejoignent. (3) La longueur consommee mediane monte de
2 350/2 420/2 456 a 2 527/2 808/2 801 pour des longueurs reelles de 2 765/2 777/2 781 : la lecture
« etat complet » tombe alors juste a 2 % pres en mediane sans avoir touche a un autre composant.
(4) La mediane de l ecart absolu baisse sur deux films sur trois (511 -> 416, 636 -> 387) et monte
sur le troisieme (448 -> 504).

**CE QUE LA MESURE NE DONNE PAS.** Une largeur FIXE ne peut pas capturer les deux variables du
format (`idxW` present ou non, queue de handle presente ou non). Elle recentre la distribution,
elle ne l ecrase pas — exactement le diagnostic de dispersion de R7-b, phase 4bis.

**2026-08-17 — Phase 3. VERDICT.**

- **C1a : ATTEINT.** La methode d ECRITURE est identifiee, nommement, pour cinq composants du
  bipede (`i0` `FUN_14320678c`, `i5` `FUN_142f07d68`, `i9` `FUN_142f075d8`/`FUN_1420ab570`,
  `i60` `FUN_142f04e2c`/`FUN_142edd10c`, `i63` `FUN_142f05144`/`FUN_142f27a68`) et pour l etat par
  defaut de l archetype (`FUN_142f14a68`), sa grammaire est ecrite champ par champ face a la
  lecture, et elle EXPLIQUE des ecarts mesures : le champ de 2 bits d `i0` est inconditionnel et
  vient EN DERNIER, son 3e bit n est pas un `precHigh` qui supprime la charge utile mais un
  selecteur de plage doublant la porte de queue — ce que le lecteur `FUN_14076e29c` du jeu confirme
  contre le port Go. Bonus non prevu au plan : quatre ports sur cinq sont CONFIRMES largeur pour
  largeur (dont la correction d `i9` de R7-b, verifiee independamment) et la limite dure d `i63`
  (`count2` = popcount RAM) est confirmee COTE ECRITURE : elle est dans le format.
- **C1b : ECHOUE.** 5 records sur 591 (**0,85 %**) contre un palier a 50 % et une cible a 95 %.
- **C2 : tenu.** Tous les negatifs sont chiffres avec leurs adresses.

**LA BORNE DU LOT (section 8) EST RESPECTEE** : elle prevoyait un arret si aucune methode
d ecriture n etait identifiable. L inverse s est produit — l ecriture est une case unique et
uniforme, `vtable[0x18]`.

**Statut de l item 2.1.** Il est marque `[~]` et non `[x]` : la divergence n a PAS ete portee en
lecture alternative dans un `keyframe_writer_grammar.go` de production. Raison, ecrite avant de
mesurer : router une lecture alternative d `i0` demande de toucher `traverse.go`
(`consumeByName`), un fichier PARTAGE, et le contrat (section 7.3) l interdit sans preuve
bit-exacte. Le hook DEJA exporte `SetCalibratedWidth` donne la meme mesure sans toucher a rien.
La mesure a ensuite montre qu une largeur FIXE ne suffit pas, et la phase 1bis a montre que la
forme visee etait de toute facon la mauvaise (portee `DAT_144e61ea0`) : porter aurait ete porter
du faux.

## 9. Decouvertes — consignees, NON traitees

1. **L ordre des composants d un etat complet suit une TABLE NOMMEE, pas l archetype.**
   `FUN_142e2c690` retrouve chaque descripteur par comparaison de CHAINE sur `vtable[0x08]`, en
   parcourant une table de 64 entrees de `0x104` octets indexee par typeIndex
   (`typeIndex * 0x4100 + 8 + base`). Le decodeur Go suppose l ordre `arch.Components`. NE PAS
   traiter ici — mais c est la premiere chose a verifier au lot suivant.
2. **L en-tete par entite d un etat complet n est pas `[id:32][field:26][ti:6]`** : c est
   `R(32)` id, `R(32)` typeIndex, `R(32)`, `FUN_142e29cf8`, `R(8)`, puis deux `R(32)` de taille
   encadrant l etat par defaut et les composants. La forme 64 bits validee par R5 vaut pour le
   record NEW du chemin DELTA. NE PAS traiter ici.
3. **`FUN_1409fe718(state, 0x49)` est appele des DEUX cotes** pour le second compte d `i63` : la
   limite « non recuperable hors ligne » est dans le format, pas dans le port. Le fil peut etre
   ferme definitivement. NE PAS traiter ici.
4. **Piege d outillage Ghidra** : `/disassemble_function` sur une adresse SANS fonction definie
   desassemble jusqu au bout du segment et rend 200 Mo. Utiliser `/read_memory` sur 32-48 octets.
   Consigne dans `WALK_PORT_NOTES.md`. NE PAS traiter ici.
5. **Le `HaloInfinite.exe` du disque est un STUB** (`.text` de 2 Mo contre 54 Mo dans l image
   Ghidra) : aucun balayage statique du binaire du disque n a de sens, tout passe par l API du
   plugin. NE PAS traiter ici.

## 11. Lignes de registre (redigees, NON ecrites par ce lot)

```
| 2026-08-17 | R7-d l ECRIVAIN par vtable de composant | RESOLU sur la FORME : la vtable d un
descripteur de composant a 10 cases et **exactement deux touchent le flux** — `+0x18` ECRIT,
`+0x30` LIT (thunk `FUN_14076ce9c` en `+0x28`) ; les cases vont par paires ETAT COMPLET (`+0x18` /
`+0x28`) et DELTA (`+0x20` / `+0x30`), et seuls TROIS composants du bipede sur 52 ont une forme
delta distincte (`i0`, `i1`, `i25`). Meme paire au niveau archetype : vtable bipede `0x143737178`,
`+0x58` ecrit l etat par defaut, `+0x60` le lit. Primitives d ecriture : `FUN_1406d49c4` = W(1),
`FUN_142f1f71c` = W(2), `FUN_1407edaf4(w,"nom",v)` = W(32) — le NOM du champ est un parametre mort
conserve en retail. Methode reutilisable : dump `.rdata` par `/read_memory` puis recherche du
pointeur 8 o du deser (unique pour 52/52), base confirmee par le getter de nom `+0x08`. |
| 2026-08-17 | R7-d, ce que l ecrivain dit des ports Go | QUATRE ports sur cinq CONFIRMES largeur
pour largeur (`i5`, `i9`, `i60`, `i63`) plus l etat par defaut de l archetype — dont la correction
de polarite d `i9` de R7-b, verifiee independamment cote ecriture, et la limite dure d `i63`
(`count2` = popcount RAM `FUN_1409fe718(state,0x49)`), appelee des DEUX cotes : elle est dans le
FORMAT, le fil peut etre ferme. UNE divergence : `i0`, dont le 3e bit n est pas un `precHigh` qui
supprime la charge utile (il choisit une table de plage `DAT_143b8c6d0` = plus ou moins 100, et il
garde la queue de handle) et dont le champ de 2 bits est INCONDITIONNEL et vient EN DERNIER. Le
lecteur du jeu `FUN_14076e29c` dit la meme chose que l ecrivain, contre le port Go. |
| 2026-08-17 | R7-d, LA BOUCLE D ETAT COMPLET (piste R7-c deroulee) | NEUF et central :
`FUN_142e2bfd0` -> `FUN_1428e2b68` -> **`FUN_142e2c690`** = la boucle de composants d un ETAT
COMPLET. Elle n a **AUCUN MASQUE DE PRESENCE** : 64 entrees d une table FIXE de `0x104` octets par
archetype, chacune deserialisee par `vtable[0x28]`, plus `R(1) + R(32)` de controle par composant
quand le drapeau film est mis. La variante « 64 leaf nus » de R7-a/R7-b est donc
STRUCTURELLEMENT la bonne. Deux consequences : (a) l ORDRE suit la TABLE et le descripteur est
retrouve PAR NOM, pas par index d archetype — derive jamais envisagee ; (b) `DAT_144e61ea0` est
leve pour TOUTE la boucle et `FUN_14076f91c()` en depend, donc la position est ECRITE ET LUE en
vec3 BRUT de 96 bits : `1+1+1+96+queue+2` = 117 avec une queue de 16, exactement la mediane
mesuree. La decouverte n2 de R7-b est RETABLIE et EXPLIQUEE. |
| 2026-08-17 | R7-d, mesure | C1a ATTEINT, **C1b ECHOUE : 5 records sur 591 (0,85 %)** contre un
palier a 50 %. La reference (lecteur porte, largeurs de carte, v4) fait atterrir **0 record sur
591** ; imposer la largeur d `i0` en fait atterrir 5. Meilleures largeurs par film : 81 / 54 / 62 —
les deux dernieres a 2 et 3 bits de la largeur QUANTIFIEE predite par l ecrivain (56, 59), et le 62
est exactement la valeur mesuree par R7-c sur le payload type-2. Longueur consommee mediane
2 527 / 2 808 / 2 801 pour un reel de 2 765 / 2 777 / 2 781. Condition de reprise : trancher
`DAT_144e61ea0` sur le payload type-2 (96 bits bruts contre `3 x axisW` quantifies), puis
verifier l ORDRE des composants contre la table nommee. |
```

## 12. Entree thought_log (redigee, NON ecrite par ce lot)

```
### [2026-08-17] Lot R7-d — L ecrivain est la case 0x18 de la vtable ; et la boucle d etat complet n a pas de masque

Statut : Complete (C1a atteint, C1b echoue a 0,85 % pour un palier a 50 %)

Decision technique. Hypothese du lot : les cases voisines de la vtable d un descripteur de
composant portent la SERIALISATION. Verifiee, et refermee : la vtable a dix cases et EXACTEMENT
DEUX touchent le flux de bits — `+0x18` ecrit, `+0x30` lit (thunk `FUN_14076ce9c` en `+0x28`) ;
les autres sont un getter de nom, un predicat de filtre, des stubs constants et un `int3`. Les
cases vont par paires etat complet / delta, et seuls trois composants du bipede sur 52 ont une
forme delta distincte. Meme paire au niveau archetype (`+0x58` ecrit l etat par defaut, `+0x60`
le lit). La methode qui a permis de retrouver ces vtables sans xref est reutilisable : dumper
`.rdata` par l API HTTP du plugin Ghidra puis chercher le pointeur 8 octets du deser — il est
UNIQUE pour les 52 composants — et confirmer la base par le getter de nom en `+0x08`.

Resultats observes. Quatre ports Go sur cinq sont CONFIRMES largeur pour largeur par l ecrivain
(i5, i9, i60, i63) plus l etat par defaut de l archetype : la correction de polarite d i9 du lot
R7-b est verifiee independamment, et la limite dure d i63 (un compte issu d un popcount en RAM)
est appelee des DEUX cotes, donc elle est dans le format et le fil se ferme. Une seule
divergence, i0 : son troisieme bit n est pas un << precHigh >> qui supprime la charge utile mais
un selecteur de table de plage qui double la porte de queue, et son champ de deux bits est
inconditionnel et vient en dernier — le lecteur du jeu le dit comme l ecrivain, contre le port.
Surtout, la piste transmise par le lot jumeau R7-c a livre la boucle d ETAT COMPLET du jeu
(FUN_142e2c690) : elle n a AUCUN MASQUE DE PRESENCE, elle parcourt une table fixe de 64 entrees
nommees et retrouve chaque descripteur PAR NOM, et elle leve pour toute sa duree la portee de
pleine precision — ce qui explique enfin pourquoi la position d une image-cle fait 117 bits et
non 3 x la largeur d axe de la carte. Mesure sur 591 records bornes, 3 films : la reference ne
fait atterrir AUCUN record ; imposer la largeur d i0 en fait atterrir 5 (0,85 %), avec des
largeurs optimales de 81 / 54 / 62 dont les deux dernieres tombent a 2 et 3 bits de la prediction
de l ecrivain.

Conclusion / prochaine etape. La forme du corps d image-cle n est plus une hypothese : pas de
masque, table nommee de 64 entrees, un mot de controle par composant sous le drapeau film. Ce qui
reste n est plus << quelle grammaire >> mais deux variables nommees : l ORDRE (table nommee contre
ordre d archetype, que le decodeur Go suppose) et `DAT_144e61ea0` (96 bits bruts contre trois axes
quantifies), sur laquelle ce lot et R7-c ne concluent pas pareil selon la source. Aucun fichier
partage du decodeur n a ete modifie ; les suites filmdec et replay restent vertes.
```
