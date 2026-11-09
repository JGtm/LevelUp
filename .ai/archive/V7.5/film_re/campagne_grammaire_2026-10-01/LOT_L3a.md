# Lot L3a — la fin du moteur de partie, lue dans le jeu (2026-10-02)

> Lot L3a du plan `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` (§6.2 L3, partie L3a seulement : le
> portage lu dans l'exécutable HI_1_13_0 ; L3b, le bit de trop des vieux builds, est HORS lot), sous
> le contrat `plan-execution` et le critère du 2026-10-02 : **corrections générales lues dans le
> jeu**, aucun réglage par film, par carte ni par version.
>
> Worktree `LevelUp-wt-cg-l3a`, branche `feat/cg-l3a`, base `af6e93e23` (L0 fusionné). Films lus en
> place, en lecture seule (20 films = 19 témoins de `config/replay_corpus.toml` + `1c4c63c2`), un à
> la fois, plafond 4 Gio. Aucune base ouverte en écriture, aucune cuisson en lot, aucune écriture dans
> le checkout principal. Ghidra : `HaloInfinite.exe`, HTTP direct `127.0.0.1:8089`, lecture seule
> (`decompile_function`, `disassemble_function`, `disassemble_bytes`, `read_memory`,
> `search_strings`, `get_xrefs_to`). Convention : **mesuré** = compté par un outil sur les films ;
> **établi** = lu dans le jeu (Ghidra) ; **supposé** = hypothèse écrite.

## 0. Statut

**[x] retenu** — après les quatre corrections du contrôle indépendant (§9), sous réserve de la
décision de l'intégrateur sur les quatre requalifications (§4.1) et sur le filet du gate de corpus
(§6.3 : rc 1, aucun oracle
touché, familles expliquées). Formule du gate 2 : **0 film en baisse nette ; 4 sains
requalifiés contredits, aucun perdu non fermé ; chacun des 4 est contredit par une règle de
l'écrivain (`FUN_142e2da44`, masque au-delà de l'archétype) sur un en-tête de tête que la marche
accepte désormais** (§4.1 : ce n'est pas une fermeture factice retirée au sens strict, c'est une
requalification nette-positive par film).

| Item | Statut |
|---|---|
| Portage `i11`, `i13`, `i14`, `i15`, `i16`, `i17` des archétypes `ti=0/1/2` | fait (§1, §2) |
| Lecteur unique de `FUN_140d580d0` / `FUN_142ba78dc`, 5 copies migrées, garde-rail | fait (§2.2) |
| Vecteurs d'après l'écrivain | fait (§3) |
| `grammar.Rev`, chronique, empreinte | fait (`grammar-2026-10-02.2`, §5) |
| `killsource.Rev`, chronique, empreinte | fait (`killsource-2026-10-02`, §5.2, correction 3 du contrôle) |
| Corrections du contrôle indépendant (4) | fait (§9) |
| Gate 1 (gofmt, vet, vet research, archlint, G-film) | vert (§6) |
| Gate 2 (carte v2, 20 films, avant / après, juge sur gains et pertes) | tenu en net (§4) |
| Gate 3 (killsource, 19 témoins) | aucune mort changée ; diagnostics seuls (§5.3) |
| `replay-equiv` (recette L0) | lot contre base : 20 / 20 différents sur 11 étapes du produit, familles expliquées (§6.2) |
| `replay-corpus-gate` | rc 1 : banc 18 / 19 ok, `60ae07c4` FAUX (un repli existant vu pour la première fois) ; 207 pertes « filet » expliquées par famille, aucun oracle touché (§6.3) |
| Mutations | 14 / 14 rouges, base verte (§3.2) |

## 1. Ce qui est lu dans le jeu

Chaîne statique : nom ASCII → xref DATA (accesseur de nom, `vtable[0x08]` = `descripteur + 0x18`) →
descripteur → lecteur à `descripteur + 0x40` (atteint par le thunk `FUN_14076ce9c`), écrivain à
`descripteur + 0x28`. Les deux côtés ont été relus pour chaque composant (lecture seule) :

| Composant (`ti=0/1/2`) | Descripteur | Lecteur | Écrivain | Grammaire (établie) |
|---|---|---|---|---|
| `i11 game-engine-soft-ceilings` | `143d0f560` | `FUN_14116d1ac` → `FUN_1406d676c(n = 0x80)` | `0x142f0688c` → `FUN_1406d60f4(n = 0x80)` | `R(128)` |
| `i13 game-engine-disabled-kill-volume-flags` | `143d0f510` | `FUN_142f03498` | `FUN_142f06530` | `n = R(13)`, puis `n × R(1)` ; le lecteur ne borne pas `n` |
| `i14 GameEngineComposerLetterboxComponent` | `143d0f740` | `FUN_142f0328c` | `FUN_142f06308` | `R(1)` + réel `R(16)` (`FUN_1406d84b4`, `[RSP+0x20] = 0x10`) + 4 × porte inversée `FUN_142efd284` (`R(1)` ; si 0 : `R(7)`) ; puis, niveau ≥ 2 : 4 × [`R(1)` ; si 1 : `R(16)`] ; niveau < 2 : `R(64)` (`FUN_1406d676c(n = 0x40)`) |
| `i15 managed-engine-timers` | `143d08ca0` | `FUN_1407ee7b8` + `FUN_1407ee87c` | `FUN_142edad74` + `FUN_142ecf980` | masque `R(64)` ; par bit `k` posé (`k` croissant) : étiquette `R(2)` ; 0 → rien ; 1 → `FUN_142ba78dc(n = 16)` ; 2, 3 → `FUN_1424cd048` = `FUN_140d580d0(n = 16)` |
| `i16 scenario-intro` | `143d08c50` | `FUN_1410d9004` → `FUN_1410d9088` | `FUN_142edd05c` → `FUN_142ed15a0` | `R(7)` (l'écrivain écrit valeur + 1) + `R(1)` |
| `i17 matchflow-isplaying-flags` | `143d08bb0` | `FUN_141101038` | `0x142edbef4` (non défini en fonction dans Ghidra, désassemblé par octets) | `R(8)` |
| lecteur de minuteur | — | `FUN_140d580d0` : `FUN_1406d84b4` × 2 (n bits) puis `FUN_1407f0354` (`+0x2c += 5`) ; `FUN_142ba78dc` : le même puis un 3e `FUN_1406d84b4` (`MOV [RSP+0x20], EBX` = n) | `FUN_142b6f76c` (`FUN_1406d22c0` × 2 puis l'octet `+0xb`) ; `FUN_142ba7c74` (le même puis `FUN_1406d22c0`) | `R(n)+R(n)+R(5)` ; `R(n)+R(n)+R(5)+R(n)` |

**Le niveau (établi).** `FUN_14076cb60` (la boucle de composants) appelle, en rejeu de film,
`FUN_1428e1b50(&DAT_144c23178, ti, nom)` : le niveau DÉCLARÉ PAR LE FILM (`R13D`), poussé en 5e
argument (`[RSP+0x20]`) ; le thunk `FUN_14076ce9c` le charge dans `R9D` (`MOV R9D, [RSP+0x28]`)
avant de sauter au lecteur. `FUN_142f0328c` teste `CMP R9D, 2 ; JC` : la branche dépend du niveau du
REGISTRE DU FILM, pas de la version du jeu. Le port reçoit le même niveau (`arch.Level(i)`, déjà
passé par `traverseComponentLoopFrom`). **Aucune branche sur la version du jeu n'est introduite** :
la forme courte n'est déclarée par aucun film du corpus (niveau 2 partout, `ecs_table.tsv`), et cet
exécutable n'a pas d'écrivain pour elle (`FUN_142f06308` n'écrit que la forme longue) ; elle est
portée d'après le LECTEUR, son vecteur le dit (§3).

**Mesures antérieures reprises (sans les refaire)** : T7 §2.6-2.7 et la note 3.7 §2 bis donnaient les
mêmes grammaires ; elles sont re-décompilées ici (`scratchpad/L3a/ghidra/`, `dec_*.txt`,
`dis_*.txt`). Ajouts de ce lot : les écrivains de `i11` (`0x142f0688c`), `i13`, `i14`, `i17`
(`0x142edbef4`) et la forme courte de `i14` relue au désassemblage.

## 2. Ce qui change

### 2.1 Production (`film/internal/grammar/`)

- `components_moteur_de_partie.go` (neuf, 138 lignes) : les six lecteurs et un maillon de la chaîne de
  dispatch, `consumeMoteurDePartie`, inséré entre `consumeNavpointComponent` et
  `consumeComposantsVueBM4b` (`dispatch_biped.go`, une ligne ; `dispatch_object.go`, la liste des
  maillons). Chaque largeur est une constante nommée qui cite l'instruction du jeu.
- `lecteur_minuteur.go` (neuf, 47 lignes) : `lireMinuteur140d580d0(br, n)` et
  `lireMinuteur142ba78dc(br, n)`, rendant les quanta bruts (`Minuteur{A, B, Queue, C}`).
- Migration des cinq copies (D-15), bits lus inchangés : `ti=5 i2` (`components_player.go`,
  `n = 5`, constante `largeurMinuteurSoftKill`), `ti=0 i5` (`vitality.go`,
  `decodeGameEngineRoundTimer`), `i6` et `i7` (`components_game_engine.go`), `i12`
  (`components_walk_batch9.go`, `Skip(37)` remplacé). Les valeurs publiées par les hooks et la
  capture restent identiques (`TestHooks*`, `TestCapture*` verts ; fixtures de contrat identiques
  hors chaîne de révision, §5).
- `testdata/ecs_table.tsv` : 15 lignes `non_porte` → `porte` (`ti=0` i11, i13-i17 ; `ti=1` i11, i13,
  i14 ; `ti=2` i11, i13-i17), adresse, grammaire, largeur, source ; 12 sources `fichier:ligne`
  recalées sur les fichiers modifiés.

### 2.2 Garde-rail (règle 6, D-15)

`lecteur_minuteur_guard_test.go` : hors de `lecteur_minuteur.go`, aucun fichier de production du
paquet ne peut porter deux `ReadBits` du même argument suivis d'un `ReadBits(5)`, ni un `Skip(37)` /
`Skip(53)`, ni (depuis la correction 2 du contrôle, §9) un `Skip` dont la ligne nomme une largeur de
minuteur (`largeurQueueMinuteur`, `roundTimerBits`, `largeurMinuteurSoftKill`) ; un saut calculé à
partir de littéraux seuls (`Skip(2*16 + 5)`) lui échappe, et il ne vérifie pas qui cite
`FUN_140d580d0` en commentaire ; le fichier hôte doit porter la séquence exactement une fois (garde contre un fantôme).
`Skip(15)` n'est pas interdit (`consumeDevicePosition` = `R(14) + R(1)`, autre forme). Le lot L2
(`ti=43 i37`, `FUN_142ba78dc`) devra appeler le lecteur unique : le garde-rail le lui impose.

### 2.3 Tests et goldens

- `components_moteur_de_partie_test.go` (neuf) : vecteurs (§3).
- `ecs_widths_guard_test.go` : 123 → 130 largeurs fixes (7 lignes neuves entrent par le haut :
  `i11` × 3, `i16` × 2, `i17` × 2), gardées inchangées (66).
- `frame_closure.golden` et `keyframe_closure.golden` re-figés (aucune ligne ne descend), historique
  écrit dans les deux en-têtes (tests ratchets).
- `grammar_rev.golden`, `killsource_rev.golden` (révision montée à `killsource-2026-10-02` par la
  correction 3, §5.2 ; la ligne `killsource-2026-09-27` garde son empreinte de la base),
  `types/testdata/shapes.golden`, 8 fixtures de contrat `replay_schema_76_*.json.gz` + `manifest.json`
  (identiques hors `grammarRev`, vérifié par décompression et substitution : 31 ou 32 occurrences
  par fixture, aucun autre octet ; puis, après la correction 3, identiques à la version du commit
  `115db0e71` hors `killsource-2026-09-27` → `killsource-2026-10-02`, 4 occurrences par fixture,
  vérifié de même sur les 8).

## 3. Vecteurs (d'après l'écrivain) et mutations

### 3.1 Vecteurs

`TestComposantsDuMoteurSuiventLEcrivain` : chaque état sérialisé par l'écrivain, écrit bit à bit,
suivi d'un fond de uns ; le dispatch doit consommer exactement les bits écrits et rendre `ported`.

| Id | Écrivain | Bits | Discrimine |
|---|---|---:|---|
| V11 | `0x142f0688c` | 128 | largeur du bloc |
| V13a, V13b | `FUN_142f06530` | 13, 16 | compte puis un bit par volume |
| V14 forme longue (niveau 2) | `FUN_142f06308` | 48 | porte inversée (index 5 puis trois −1), mot `0x1234` puis trois `0xffff` non écrits |
| V14 forme courte (niveau 1) | LECTEUR `FUN_142f0328c` (pas d'écrivain dans l'exécutable) | 92 | même tronc, `R(64)` |
| V15a | `FUN_142edad74` / `FUN_142b6f75c` | 105 | fente étiquette 2 (trois champs), fente étiquette 0 (rien) |
| V15b | `FUN_142ba7c74` | 119 | étiquette 1 : quatre champs |
| V15c | `FUN_142edad74` | 103 | fente 63 (bit de poids fort), étiquette 3 |
| V15d | — | 64 | masque vide |
| V16 | `FUN_142ed15a0` | 8 | valeur + 1 sur 7 bits |
| V17 | `0x142edbef4` | 8 | — |

Plus `TestCompteDeVolumesNonBorne` (`n = 8191` : 8 204 bits, le lecteur du jeu ne borne pas) et
`TestLecteurDeMinuteurRendLesQuanta` (`n = 5` et `n = 16`, ordre des quanta de `FUN_142b6f76c`).

### 3.2 Mutations (`l3a_tsv/mutations.sh`, résultat `l3a_tsv/mutations.txt`)

Base verte ; **14 / 14 ROUGES** :

| Mutation | Tests rouges |
|---|---|
| M1 `i11` sur 127 bits | vecteurs, G4, ratchet image-clé |
| M2 `i13` bits de volume non lus | vecteurs, compte non borné, ratchet image-clé |
| M3 `i13` compte sur 12 bits | idem |
| M4 `i14` niveau ignoré (forme longue toujours) | vecteurs (seul témoin : aucun film ne déclare le niveau 1) |
| M5 `i14` porte lue droite | vecteurs, ratchet image-clé |
| M6 `i15` étiquette 1 lue sur trois champs | vecteurs |
| M7 `i15` fente éteinte lue comme une fente | vecteurs |
| M8 `i16` sur 8 bits | vecteurs, G4, ratchet image-clé |
| M9 `i17` sur 7 bits | vecteurs, G4, ratchets image-clé et trame |
| M10 queue du minuteur sur 4 bits | vecteurs, quanta, G4, ratchets |
| M11 copie en ligne revenue (`ti=5 i2`) | garde-rail |
| M12 `Skip(37)` revenu (`i12`) | garde-rail |
| M13 maillon non chaîné | vecteurs, compte, G4, ratchets |
| M14 lecteur hôte sans la séquence | garde-rail (fantôme) |
| X16b (contrôle) saut calculé `br.Skip(int(2*roundTimerBits + largeurQueueMinuteur))` (`i12`) | garde-rail, après son élargissement (§9, correction 2) ; VERTE avant |

**Écart** : M1 à M10 et M13 par `-overlay` (copie mutée, le worktree n'est pas touché). M11, M12,
M14 visent le garde-rail, qui lit les SOURCES sur disque (`os.ReadFile`), ce que `-overlay` ne
remplace pas : mutation EN PLACE sur sauvegarde, restaurée et vérifiée à l'octet après chaque essai
(`cmp`), comme les mutations de L0.

## 4. Gate 2 — carte de fermeture v2, 20 films, avant (`af6e93e23`) / après

`cmd_fermeture -mode v2 -paquets -denominateur-fixe r_comb2_denominateurs.tsv -plafond-gib 4`, binaire
de la base contre binaire du lot, table ECS de chacun ; comparaison paquet par paquet
(`l3a_tsv/comparer.awk`). « Sain » = fermé selon L0 (au bit près ET aucune règle de l'écrivain
contredite). « Gagnés au bit » = paquets non fermés au bit avant, fermés au bit après ; le juge des
invariants de L0 (`ecrivain_invariants.go`) est joué sur eux comme sur les perdus.

| Film | Build | Sains avant | Sains après | Δ | Sains perdus (contredits / non fermés) | Gagnés au bit (contredits ; part factice) | Contredits devenus sains | Fermés au bit perdus (tous contredits avant) | Utiles sains avant | après | Δ | Utiles lus avant | après |
|---|---|---:|---:|---:|---|---|---:|---:|---:|---:|---:|---:|---:|
| `084a804d` | HI_1_10_0 | 4773 | 4803 | +30 | 0 / 0 | 41 (11 ; 26,8 %) | 0 | 0 | 88036 | 88841 | +805 | 535804 | 540525 |
| `111fa685` | HI_1_10_0 | 4012 | 4014 | +2 | 0 / 0 | 3 (1 ; 33,3 %) | 0 | 0 | 42601 | 42623 | +22 | 245556 | 245898 |
| `1c4c63c2` | HI_1_10_0 | 13333 | 13375 | +42 | 1 / 0 | 425 (390 ; 91,8 %) | 8 | 9 | 165404 | 165766 | +362 | 777075 | 780079 |
| `e5adf7b2` | HI_1_11_0 | 4123 | 4124 | +1 | 0 / 0 | 1 (0 ; 0 %) | 0 | 0 | 79457 | 79458 | +1 | 289357 | 289358 |
| `bcb6d393` | HI_1_12_0 | 5830 | 5879 | +49 | 0 / 0 | 50 (1 ; 2,0 %) | 0 | 0 | 35143 | 35492 | +349 | 121628 | 122669 |
| `0797ce72` | HI_1_13_0 | 19152 | 19156 | +4 | 0 / 0 | 7 (3 ; 42,9 %) | 0 | 0 | 159920 | 159929 | +9 | 205685 | 205703 |
| `396cfc92` | HI_1_13_0 | 22819 | 23131 | +312 | 0 / 0 | 312 (0 ; 0 %) | 0 | 0 | 166984 | 169502 | +2518 | 225495 | 227125 |
| `4f77afc1` | HI_1_13_0 | 22260 | 23254 | +994 | 3 / 0 | 1032 (46 ; 4,5 %) | 11 | 1 | 550289 | 581027 | +30738 | 738490 | 753109 |
| `51ebbc0f` | HI_1_13_0 | 9759 | 9953 | +194 | 0 / 0 | 196 (4 ; 2,0 %) | 2 | 0 | 58585 | 60186 | +1601 | 88493 | 89822 |
| `bf15f7ab` | HI_1_13_0 | 28465 | 28603 | +138 | 0 / 0 | 138 (0 ; 0 %) | 0 | 0 | 214974 | 216104 | +1130 | 227637 | 228864 |
| `bfecd02b` | HI_1_13_0 | 26380 | 26930 | +550 | 0 / 0 | 550 (0 ; 0 %) | 0 | 0 | 226529 | 232582 | +6053 | 254963 | 260624 |
| `c75f33b8` | HI_1_13_0 | 22854 | 23053 | +199 | 0 / 0 | 201 (2 ; 1,0 %) | 0 | 0 | 144430 | 146206 | +1776 | 155136 | 156237 |
| `d9781168` | HI_1_13_0 | 26210 | 26317 | +107 | 0 / 0 | 112 (5 ; 4,5 %) | 0 | 0 | 180052 | 180854 | +802 | 244872 | 245931 |
| `f75e7053` | HI_1_13_0 | 23416 | 23673 | +257 | 0 / 0 | 258 (1 ; 0,4 %) | 0 | 0 | 160042 | 162137 | +2095 | 187304 | 188919 |
| `fb1a1a72` | HI_1_13_0 | 22276 | 22713 | +437 | 0 / 0 | 438 (1 ; 0,2 %) | 0 | 0 | 161566 | 165643 | +4077 | 179254 | 181931 |
| `a521164d` | HI_1_4_1 | 692 | 692 | 0 | 0 / 0 | 1 (1 ; 100 %) | 0 | 0 | 78 | 78 | 0 | 181607 | 184473 |
| `60ae07c4` | HI_1_8_0 | 13802 | 13946 | +144 | 0 / 0 | 145 (1 ; 0,7 %) | 0 | 0 | 82730 | 83669 | +939 | 268352 | 271810 |
| `11de8353` | HI_1_9_0 | 5612 | 5622 | +10 | 0 / 0 | 12 (2 ; 16,7 %) | 0 | 0 | 65230 | 65460 | +230 | 248287 | 248837 |
| `50247b26` | version-31 | 139 | 139 | 0 | 0 / 0 | 0 | 0 | 0 | 274 | 274 | 0 | 319053 | 320107 |
| `a349fea8` | version-33 | 420 | 424 | +4 | 0 / 0 | 4 (0 ; 0 %) | 0 | 0 | 3595 | 3654 | +59 | 469451 | 479511 |
| **corpus** | | **276 327** | **279 801** | **+3 474** | **4 / 0** | **3 926 (469 ; 11,9 %)** | 21 | 10 | **2 585 919** | **2 639 485** | **+53 566** | 5 963 499 | 6 021 532 |

Lecture (mesuré) :
- **Aucun film en baisse nette**, ni en paquets sains, ni en records utiles sains. Deux films à 0
  (`a521164d`, `50247b26` : le bit de trop des vieux builds, L3b, hors lot).
- Contrôle : `sains après = sains avant − perdus + gagnés sains + contredits devenus sains` :
  276 327 − 4 + 3 457 + 21 = 279 801.
- Gains contredits par règle (`l3a_tsv/gains_contredits.tsv`) : `1c4c63c2` 231 sortie par rejet, 126
  masque au-delà de l'archétype, 24 ordre de la vue B, 8 masque épars non croissant ; `4f77afc1` 29
  masque, 15 rejet, 2 ordre ; le reste ≤ 4 par film et par règle. Ils ne comptent pas comme sains.
- Les 10 « fermés au bit perdus » étaient tous contredits avant (`1c4c63c2` 9, `4f77afc1` 1 ; règles
  masque ou sortie par rejet) et deviennent « liste d'événements non localisée » : fermetures
  factices retirées, aucun sain.
- Dénominateurs (corpus) : variable 43,36 % → 43,83 % ; fixe consolidé (7 758 290, aucune marche du
  lot ne lit plus que lui) 33,33 % → 34,02 %. HI_1_13_0 : utiles fermés 2 023 371 → 2 074 170.
- Causes d'arrêt : les lignes `ti=0/1/2 i11, i13, i14, i15, i16, i17` (avant : `ti=2 i15` 3 996
  paquets, `ti=2 i17` 85, `ti=2 i11` 67, `ti=2 i14` 42, `ti=0 i11` 34, `ti=0 i13` 28 …) disparaissent
  toutes ; aucune cause nouvelle d'un composant porté par le lot (l'oracle est le paquet suivant).
  Paraît la suite de l'archétype `ti=0` : `forge-engine-*` (`i18` à `i26`, 100 paquets, dont `i18
  forge-engine-player-roles` 61) — hors lot (D-L3a-6).
- **Oracle indépendant, images-clés** (`TestImageCleFermetureParArchetype`, 20 films,
  `l3a_tsv/images_cles_*.txt`) : `ti=2` sous l'état complet **0 / 706 → 588 / 706** ; témoin
  décalé d'un bit 0 / 706 avant comme après (la largeur est discriminée) ; `ti=1` 0 / 39 (désyncs
  25 → 0) ; aucun autre archétype ne bouge. Les 118 non fermés : les 98 records des trois vieux builds
  (L3b) et 20 autres, non instruits (R-COMP en comptait 16 sur les builds récents).
- Comparaison aux mesures de recherche (contexte différent, non une preuve) : R-COMB-2, seul, juge à
  trois règles, base pré-L0 : +3 575 / +53 521. Ici, juge de L0, base `af6e93e23`, code du lot :
  +3 474 / +53 566.

### 4.1 Les 4 sains perdus, instruits un par un

Tous **devenus contredits** (fermés au bit près avant ET après), aucun devenu non fermé. Instruits par
`TestCampagneL0Paquets` (sonde existante, `-tags=research`), joué avec le code du lot puis avec la base
par `-overlay` (`l3a_tsv/paquets_requalifies_{avant,apres}.txt`) :

| Paquet | Début de liste avant → après | Ce que lit la tête ajoutée | Règle contredite |
|---|---|---|---|
| `1c4c63c2` 11:1620 | 7693 → **7447** | NEW slot 2 `ti=2` (18 composants), masque `0x6000000018004034` (bits 27, 28, 61, 62 au-delà), puis DELTA slot 73 `ti=5` jusqu'à 7693 | masque au-delà de l'archétype |
| `4f77afc1` 25:874 | 447 → **360** | NEW slot 1024 `ti=0` (27), masque `0x10000000004000` (bit 52 au-delà) | idem |
| `4f77afc1` 37:1188 | 310 → **227** | NEW slot 2048 `ti=0`, masque `0x10000100020100` (bits 40, 52 au-delà) | idem |
| `4f77afc1` 59:682 | 872 → **131** | NEW slot 2529 `ti=0` (bit 40 au-delà), NEW slot 927 `ti=23` (tous les bits au-delà de 33), puis DELTA slot 2 `ti=2`, masque `0x8000` (`i15` seul, 637 bits) | idem + ordre de la vue B |

Mécanisme (établi sur pièces, pour les quatre) : le localisateur de tête des paquets à événements
retient le premier candidat d'où la chaîne de records se lit sans désynchronisation. Avant le lot,
ces candidats plus tôt butaient sur un composant non porté de `ti=0/2` (`i14`, `i15`) ; le lot les
lit, la chaîne se prolonge jusqu'au début d'avant, et le paquet ferme au bit près exactement comme
avant (même vue C). Mais la tête ajoutée porte un NEW dont le masque pose des bits que
`FUN_142e2da44` n'écrit jamais (`i < *(desc+0x4320)`) : la lecture depuis cette tête contredit
l'écrivain, et le juge de L0 le dit. Le paquet n'a donc pas perdu sa vue C : il a gagné une tête que
le jeu n'a pas écrite. Le traverseur MARQUE la règle (`EntityTrace.MasqueNonEcrit`) mais ne
désynchronise pas, et la marche ne la consulte pas pour choisir sa tête (D-L3a-1).

Qualification au regard du gate : chacun des quatre est contredit par une règle de l'écrivain (la
condition posée au lot) ; ce n'est pas, au sens strict de D2, « une fermeture factice retirée » (la
lecture d'avant, depuis l'ancienne tête, n'était pas factice). Le gate est NET par film (§6.0 point
2) : `1c4c63c2` +42 net, `4f77afc1` +994 net ; records utiles sains nets positifs sur les deux. La
correction générale (refuser comme tête un NEW que l'écrivain ne peut pas écrire) est une règle de
MARCHE, hors de ce lot de composant (règle 5 du contrat : pas de correctif opportuniste) ; elle est
proposée en découverte.

## 5. Révisions et gate 3

### 5.1 `grammar.Rev`

`grammar-2026-10-02` → **`grammar-2026-10-02.2`** ; entrée `rev_chronique.go` ; empreinte
régénérée par la commande du dépôt (`957023c2…`, après le renommage d'une constante interne au lot ;
première valeur `828d0246…`). **Écart** : la forme du dépôt (`revision/chronique.go`,
`formeRevision`) n'admet qu'un rang numérique ≥ 2 et exige des rangs consécutifs (`Rang.Suit`) ; le
suffixe distinctif `.l3a` est refusé. `.2` est le seul rang valide ; les lots parallèles (L2, L8)
partent de la même base et prendront vraisemblablement le même : l'intégrateur renumérotera.

### 5.2 Autres révisions

- `source.Rev`, `profile.Rev`, `objectives.Rev` : inchangées (aucune source touchée ; tests verts).
- `killsource.Rev` : **`killsource-2026-09-27` → `killsource-2026-10-02`** (correction 3 du
  contrôle, §9), par la règle écrite du plan : §6.0 point 1 (N6, « `facts.Rev` suit » ;
  `decfilm.Rev = killsource.Rev`, `decfilm.go:89-91`) et point 3 (« `killsource.Rev` est monté si la
  sortie change »), la sortie JSON changeant sur 12 témoins (§5.3). Entrée neuve dans
  `facts/killsource/rev_chronique.go` ; golden régénéré par la commande du dépôt (empreinte
  `a0a59c83…`, celle de la première version du lot, car aucune source de la couche ne change : seule
  la VALEUR de `grammar.Rev` bouge) ; la ligne `killsource-2026-09-27` reprend l'empreinte de la base
  (`3d8c497b…`), que la première version du lot avait recopiée. Suivent `types/testdata/shapes.golden`
  et les 8 fixtures de contrat (§2.3).
- `replay.SchemaVersion` : inchangée (76) ; le document ne change que par la chaîne de révision des
  calques sur les fixtures (§2.3) — à confirmer par `replay-equiv` (§6.2).

### 5.3 Gate 3 — killsource

`cmd/killsource json <film> -carte <carte> -cache <cache du parc, lecture seule> -catalogue …`, 19
témoins (`l3a_tsv/ks_diff.tsv`), binaire de la base contre binaire du lot :
- **7 / 19 identiques à l'octet** (`084a804d`, `11de8353`, `4f77afc1`, `50247b26`, `60ae07c4`,
  `a521164d`, `e5adf7b2`) ;
- **11 / 19** : seule la chaîne `calibration` change — le diagnostic d'ORACLE (score du profil plat,
  par exemple `bcb6d393` 33 → 34 ; score de `indexW_poignee`, par exemple `0797ce72` 876 → 880) ;
- **`111fa685`** : seuls deux compteurs de santé changent (`killsource_candidates_total` 226 → 227,
  `killsource_unexplained_pair` 24 → 25, d'où `gate_par_voie.sequentielle` population 205 → 206 et
  `taux_inexpliques`) ;
- **aucune mort, aucune valeur, aucune voie ne change** (aucun chemin `morts.*` dans les écarts, 191
  morts sur `111fa685` avant et après). Conforme à R-COMB-2 §6 pour L3a.

Première version du lot (commit `115db0e71`) : `killsource.Rev` ne montait pas, au motif que ces
deux sorties ne sont ni persistées dans `match_kill_events` ni lues par `decoder_rev`. Le contrôle
indépendant l'a relevé contraire à la règle écrite (§6.0 points 1 et 3), et aucune décision datée de
l'intégrateur n'y déroge : **`killsource.Rev` monte à `killsource-2026-10-02`** (§5.2, §9). Le
backlog qu'elle ouvre est de datation (aucune mort ne change), sur signal utilisateur (D6, D7).
`1c4c63c2` n'a pas de carte lisible et n'est pas joué (comme L0 et R-COMB-2).

## 6. Gates (sorties exactes)

### 6.1 Gate 1

Depuis `apps/go-api`, `GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-cg-l3a`, une commande `go`
à la fois :

| Gate | Sortie |
|---|---|
| `gofmt -l ./internal/games/halo_infinite/film/ ./cmd/` | vide |
| `go vet ./internal/games/halo_infinite/film/...` | rc 0, aucune sortie |
| `go vet -tags=research ./internal/games/halo_infinite/film/...` | rc 0, aucune sortie |
| `go test ./internal/archlint/ -count=1` | `ok … 57.758s` |
| G-film : `go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/... ./internal/sync/killcollector/... -count=1 -timeout 30m` | rc 0, 19 paquets `ok` (`grammar` 33,9 s, `replay` 30,9 s, `facts/killsource` 4,7 s, `revision`, `types` …) |
| `go test -tags=research ./internal/games/halo_infinite/film/research/cmd_fermeture/ -count=1` | `ok` |
| `golangci-lint run --new-from-rev=af6e93e23 ./internal/games/halo_infinite/film/internal/grammar/` (cache isolé) | `0 issues.` |
| Mutations | 14 / 14 rouges, base verte |

### 6.2 `replay-equiv` (recette L0)

Racine factice du lot (`scratchpad/L3a/repo` : copie de la racine de L0 ; `config/` et références
d'équivalence identiques à l'octet à celles du worktree, `diff -rq` vide ; catalogue
`data/titles/halo_infinite/reference` identique à celui du parc principal), 20 films, binaire de la
base (`af6e93e23`) puis binaire du lot, digests conservés (`-out-dir`) et comparés base contre lot
(`l3a_tsv/cmp_equiv.sh`, résultat `l3a_tsv/re_etapes_divergentes.tsv`).

- Base contre références du dépôt : `BILAN : 0 identique(s), 20 different(s)` — attendu, les
  références n'ont pas été re-figées après L0 (mêmes étapes que L0 §7.1).
- Lot contre références : `BILAN : 0 identique(s), 20 different(s), 0 ecarte(s), 0 echec(s)`.
- **Lot contre base** (ce qui est imputable au lot) : sur 61 étapes, divergent `artifact` 20,
  `continuousFire.stats` 20, `movementStates.stats` 20, `movementStates` 18, `killsource` 14,
  `vehicles` 10, `continuousFire` 8, `birthLoadouts` et `.stats` 7, `heldWeaponChanges` 3, `killRefs`
  1. Les autres étapes (positions, morts, objectifs, équipement, socles, ramassages…) sont
  identiques sur les 20 films.

Instruit sur deux films cuits un par un, base contre lot (`cmd/replay-build --facts` des références
d'équivalence, `111fa685` et `d9781168`, quatre cuissons, aucune en lot), document comparé clé par
clé :
- `111fa685` : seuls changent `coverage`, `layers` (la chaîne de révision, sur tous les calques) et
  `stances` ; `d9781168` : en plus `loadouts` (+2 dotations de naissance lues, `unconfirmed` 2 → 0)
  et `weaponChanges` (un `taken` devient `swapped`, l'arme d'avant étant désormais connue).
- États de mouvement (mesuré) : les intervalles se COUPENT là où un record désormais lu montre la fin
  de l'état (`111fa685` slot 637 : `sprint` 3089-3112 devient 3089-3094 + 3104-3112) ; des épisodes
  s'ajoutent (`intervals` 2 822 → 2 825, `reads` 7 441 → 7 446, `records` 229 785 → 230 109).
- Tir continu (mesuré) : `closed` 4 012 → 4 014 et 26 210 → 26 317, `holes` en baisse (13 350 →
  13 348 ; 17 435 → 17 328), `holesOpenViewB` en baisse (159 → 136 ; 189 → 36) et
  `holesNotClosing` en hausse (11 084 → 11 105 ; 11 503 → 11 556) : un trou passe de « vue B
  ouverte » à « vue C atteinte, non fermée » (la vue B se lit désormais jusqu'au bout).
- Étapes `killsource`, `killRefs` : la section de faits `killsource` (`killsource.Result`) porte le
  diagnostic de calibration et les compteurs de santé (§5.3) ; supposé être leur seule cause sur les
  films d'équivalence qui ne sont pas des témoins de gate 3 (non instruit film par film ; sur les
  films communs, aucune mort ne change).
- `vehicles` (10 films) : non instruit ; sur `111fa685`, la clé `vehicles` du document cuit est
  identique, la différence est donc hors du calque publié (supposé : compteurs de couverture).

Durées et pics (`l3a_tsv/replay_equiv_{avant,apres}_durees.txt`) pris machine CHARGÉE (carte, gate
de corpus et tests en parallèle) : non comparables ; pics ≤ 0,52 Gio des deux côtés. Le plafond du
gate 4 ne s'applique pas à ce lot (aucune lecture du bloc de type 1).

### 6.3 `replay-corpus-gate`

`gate.exe --reference=base --base=af6e93e23 --parc-root=<copie au scratchpad> --source-root=<worktree>
--work-root=<scratchpad> --json=…` (rapport : `l3a_tsv/corpus_gate_rapport.txt`,
`l3a_tsv/corpus_gate.json.gz`). La copie du parc (`scratchpad/L3a/parc` : films et manifestes des 19
témoins, `metadata.duckdb` et `shared_matches_v2.duckdb` identiques à l'octet au parc principal)
évite que la gate écrive son cache de base et son verrou sous le parc ; le worktree de base a été
créé puis retiré par l'outil. Sortie : **rc 1** ; statut **PERTE** sur 18 témoins, **FAUX** sur
`60ae07c4` ; 639 gains, 207 pertes, 83 changements.

Banc de vérité : **18 / 19 ok**, `60ae07c4` **FAUX** sur une seule ligne :
`R-1 repli repli_physique_de_type_de_vehicule_supposee : 0 -> 1`. P-1 en GAIN sur 17 témoins (égal à
la carte, par exemple `bcb6d393` 5 830 → 5 879). **Aucun oracle ne bouge** (kills, morts,
assistances, score, équipes, vies, violations V-1 à V-8). Le FAUX : ce repli NOMMÉ existant
(`ti=40 i34`, porte supposée) se déclenche une fois sur un film où la marche ne l'atteignait pas ;
le banc classe FAUX tout repli vu pour la première fois sur un témoin (même mécanisme que D-L0-5).
Ailleurs il monte en information (`4f77afc1` 14 117 → 14 679, `084a804d` 8 914 → 9 040,
`50247b26` 5 374 → 5 395) : plus de records `ti=40` lus. Aucun repli neuf n'est introduit par le lot.

Les 207 pertes, par famille (toutes classées `[FILET]`, « aucune mesure du banc ne couvre ce
bloc ») :
- `stances/duree-totale` (10 témoins, −0,1 % à −3,4 %) et par slot (111 lignes) : les intervalles
  coupés par les records désormais lus (mesuré sur `111fa685`, §6.2) ; les comptes d'épisodes
  montent dans le même temps (`jumpEpisodes`, `sprint` : changements en hausse sur 17 témoins).
- `coverage.continuousFire.holesNotClosing` / `holesKind` / `holesBlockBC` / `holeRuns` : la
  reclassification des trous de vue B en trous de vue C (mesuré sur deux films, §6.2).
- `weaponChanges/par-kind/taken` (3 témoins, −1 chacun) : chaque fois `swapped` +1 (changement) —
  une prise devient un échange, l'arme d'avant étant lue.
- `coverage.stances.refusedNews*`, `forgottenBindings`, `dropped`, `desyncs` : plus de NEW atteints
  et jugés par le calque (estimé : la marche lit plus de records) ; non instruit paquet par paquet.
- `coverage.fallbacks/n` (`60ae07c4`) : le repli ci-dessus.

Lecture : le gate de corpus est un filet sur le PRODUIT ; il ne distingue pas « lu autrement parce
que lu plus loin » de « perdu ». Les familles ci-dessus sont expliquées par mécanisme (mesuré sur deux
films, estimé pour le reste) ; aucune ne touche un oracle. Écart déclaré, à trancher par
l'intégrateur avec la recuisson de la vague (les références `replay-equiv` se re-figent après la
vague, plan §6.0 point 6).

## 7. Écarts au critère et à la commande

- Révision `.2` au lieu d'un suffixe propre au lot (§5.1).
- Quatre sains requalifiés contredits (§4.1) : contredits par une règle de l'écrivain, nets positifs
  par film ; non « factices retirées » au sens strict de D2.
- Mutations du garde-rail jouées en place (avec restauration vérifiée) et non par `-overlay` (§3.2).
- La forme courte de `i14` est portée d'après le LECTEUR du jeu, sans écrivain dans l'exécutable ni
  film qui la déclare : seul un vecteur la tient (M4). Elle n'introduit aucune branche de version :
  elle suit le niveau déclaré par le registre du film, comme le jeu.
- Aucune valeur « présumée par mesure » : toutes les largeurs du lot sont lues dans le jeu.
- `replay-corpus-gate` rc 1 (PERTE 18, FAUX 1) et `replay-equiv` divergent sur le produit (§6.2,
  §6.3) : expliqués par famille, mesurés sur deux films, estimés ailleurs ; aucun oracle touché.
- Une constante interne (`fenteCourte` → `fenteQuatreChamps`) et un alignement de commentaire ont
  été corrigés APRÈS la construction des binaires de carte, killsource et `replay-equiv` du lot :
  aucun bit lu ne change (vecteurs et mutations rejoués après, empreinte `957023c2…` identique après
  l'alignement : l'empreinte ignore les commentaires). Le binaire `replay-build` de la gate de corpus
  a été compilé dans la même minute que le renommage (ordre exact non établi) : sans effet sur la
  sortie pour la même raison.
- Mutations M11, M12, M14 jouées en place pendant que la gate de corpus tournait : ses binaires
  étaient déjà compilés (22:18-22:19), et ces trois mutations ne changent aucun bit lu.

## 8. Découvertes

- **D-L3a-1** Le localisateur de tête des paquets à événements accepte un NEW dont le masque contredit
  `FUN_142e2da44` (bits au-delà du dernier composant de l'archétype) : le traverseur marque
  `MasqueNonEcrit` sans désynchroniser, et la marche ne consulte pas la règle. Effet mesuré dans ce
  lot : 4 sains requalifiés contredits (§4.1). Correctif général proposé, de MARCHE (LS / LU / L1a) :
  un candidat de tête dont la chaîne contredit une règle de l'écrivain n'est pas une tête. Non traité.
- **D-L3a-2** `ecs_table.tsv` : la ligne `ti=2 i0 game-engine-team-mapping-component` porte la
  grammaire `R(16) + R(16) + R(5)`, 37 bits et le sens « l'horloge de campagne » : c'est la
  description de `i12 game-engine-campaign-timer` recopiée sur la mauvaise ligne. Non traité.
- **D-L3a-3** Plusieurs `code_source` de `ecs_table.tsv` désignent une ligne qui n'est plus celle du
  `case` (par exemple `game-engine-campaign-timer` → `dispatch_biped.go:108`, le `case` est vers la
  ligne 200) : G1 ne vérifie que l'existence de la ligne. Non traité (seules les lignes des fichiers
  modifiés par le lot ont été recalées).
- **D-L3a-4** La forme des révisions (`formeRevision`, `Rang.Suit`) interdit un suffixe par lot : des
  lots parallèles d'une même vague prennent forcément le même rang ; l'intégrateur renumérote.
- **D-L3a-5** Images-clés : `ti=1` reste 0 / 1 sur `60ae07c4` et `fb1a1a72` sans bloquant nommé, et
  20 records `ti=2` des builds récents ne ferment sous aucune grammaire portée. Non instruit.
- **D-L3a-6** Fin de `ti=0` : `forge-engine-player-roles` (`i18`) et `forge-engine-*` (`i19` à `i26`)
  deviennent la cause suivante (100 paquets, dont `i18` 61). Hors du plan ; candidat à un lot lu
  dans le jeu.
- **D-L3a-7** Les garde-rails qui lisent les sources (`os.ReadFile`) échappent à `-overlay` : une
  mutation qui les vise doit être jouée en place. Non traité.
- **D-L3a-8** L'en-tête de `dispatch_object.go` parlait encore de « SEPT maillons ». La CHAÎNE
  (chaque `default` qui rend le maillon suivant) compte **onze** maillons avec ce lot (dix avant) ;
  la LISTE de l'en-tête n'en nommait que dix (neuf avant) : elle omettait
  `consumeManagedPlayerComponent` (`dispatch_player.go:311`, maillon du lot 3.6.a entre
  `consumePlayerTailAndGameEngineComponent` et `consumeCaptureAndBipedComponent`). **Traité par la
  correction 4 du contrôle (§9)** : la liste reçoit ce maillon (onze lignes `consume*`, égal au compte
  de la chaîne) et la phrase « Cette exemption vaut pour les SEPT maillons » est réécrite (l'exemption
  vaut pour les sept maillons qui portent `//nolint:gocyclo,funlen // dette gelee` ; les quatre autres
  restent sous le seuil). Reste NON traité, même en-tête : « LES SEPT `//nolint` … » est historique
  (lot 2.6.2) et reste vrai.
- **D-L3a-9** `111fa685` : un candidat killsource de plus et un couple inexpliqué de plus ; cause non
  instruite (la marche lit plus de records). Sans effet sur les morts.

## 9. Corrections du contrôle indépendant (2026-10-02)

Contrôle sur `115db0e71` (verdict : retenable après quatre corrections mineures, aucune ne change un
bit lu). Corrections appliquées dans le worktree du lot, même branche, commit « campagne(grammaire)
L3a: corrections du contrôle ».

| # | Correction | Statut | Vérification (mesurée) |
|---|---|---|---|
| 1 | `lecteur_minuteur.go`, en-tête GARDE-RAIL : retirer « exige que tout fichier qui cite `FUN_140d580d0` ou `FUN_142ba78dc` appelle l'un de ces deux lecteurs » (faux : aucune vérification de citation ; `quantize_endpoint.go:54`, `dispatch_player.go:22`, `rev_chronique.go:450` citent sans appeler) | [x] | `grep -n 'exige que tout fichier' lecteur_minuteur.go` : vide. L'en-tête décrit désormais ce que le test vérifie et dit qu'il ne vérifie pas les citations. |
| 2 | `lecteur_minuteur_guard_test.go` : l'en-tête promettait l'interdiction d'un « saut de 2n + 5 bits » ; X16b (`br.Skip(int(2*roundTimerBits + largeurQueueMinuteur))`) restait verte | [x] (garde-rail ÉLARGI, en-tête ramené à ce qu'il vérifie) | Nouvelle expression `sautMinuteurCalcule` : un `Skip` dont la ligne nomme `largeurQueueMinuteur`, `roundTimerBits` ou `largeurMinuteurSoftKill`. Base : `TestLecteurDeMinuteurUnique` vert. X16b jouée en place puis restaurée par `git checkout` (statut vide après) : **ROUGE** (`components_walk_batch9.go : 0 lecture(s) en ligne et 1 saut(s)`). Variante sur `ti=5 i2` (`br.Skip(int(3 * largeurMinuteurSoftKill))`) : **ROUGE**. Reste hors garde-rail, écrit dans l'en-tête : un saut calculé de littéraux seuls (`Skip(2*16 + 5)`). |
| 3 | `killsource.Rev` : appliquer la règle écrite (§6.0 points 1 et 3) ou citer une décision datée de l'intégrateur | [x] (règle appliquée ; aucune décision datée n'existe) | `killsource-2026-09-27` → `killsource-2026-10-02` (`rev.go`), entrée de chronique (`rev_chronique.go`, 465 lignes), golden régénéré par la commande du dépôt puis vérifié sans drapeau (`ok`), ligne `09-27` rendue à l'empreinte de la base ; `shapes.golden` et 8 fixtures de contrat régénérés (identiques hors la chaîne, 4 occurrences par fixture). §5.2, §5.3. |
| 4 | D-L3a-8 : « onze (dix avant) » contre une liste de dix (neuf avant) ; phrase « SEPT maillons » de `dispatch_object.go:43` | [x], avec un écart sur pièces | **Le compte de la correction est faux sur pièces** : la chaîne réelle (`default` → maillon suivant) a onze maillons avec le lot, dix avant — la liste en omettait un, `consumeManagedPlayerComponent` (`dispatch_player.go:311`, lot 3.6.a). Ramener LOT_L3a.md à « dix » aurait écrit une affirmation fausse : non appliqué tel quel. Appliqué à la place (règle 17, même bloc d'en-tête) : la liste reçoit ce maillon (`grep -c '^//	consume' dispatch_object.go` = **11**, égal au compte cité), la ligne de `consumePlayerTailAndGameEngineComponent` perd « joueur géré (ti=9) », la phrase de l'exemption nomme les sept maillons `funlen` et les quatre autres (32, 40, 21 et 22 lignes, mesurées accolade à accolade), et « Chaque maillon dépasse 80 lignes » (faux pour ces quatre) devient « Les maillons exemptés sont longs ». |

Aucun code de lecture ne change : hors commentaires et tests, le seul octet de production modifié est
la constante `killsource.Rev` (`git diff -- '*.go' ':!*_test.go'`, lignes non commentaires).
`grammar.Rev` reste `grammar-2026-10-02.2` et `TestGrammarRevSuitLaGrammaire` reste vert : la
grammaire est identique au jeton près. La carte v2, la sortie `cmd/killsource json` (qui ne publie pas
la révision), `replay-equiv` et `replay-corpus-gate` ne sont donc PAS rejoués : aucun film concerné
(établi par le diff, non par une mesure sur film).

Gates rejoués (depuis `apps/go-api`, `GOCACHE=…/go-build-cg-l3a`, une commande `go` à la fois,
sorties sous `scratchpad/L3a/corr_ctl/`) :

| Gate | Sortie |
|---|---|
| `gofmt -l ./internal/games/halo_infinite/film/ ./cmd/` | vide |
| `go vet ./internal/games/halo_infinite/film/...` | rc 0 |
| `go vet -tags=research ./internal/games/halo_infinite/film/...` | rc 0 |
| G-film (`film/...`, `replaybuild/...`, `sync/killcollector/...`, `-count=1 -timeout 30m`) | rc 0, 19 paquets `ok`, 0 FAIL (`grammar` 39,5 s, `replay` 33,0 s, `facts/killsource` 5,1 s) |
| `go test ./internal/archlint/ -count=1` | `ok … 41.030s` |
| `golangci-lint run --new-from-rev=af6e93e23` sur `grammar/` et `facts/killsource/` (cache isolé) | `0 issues.` |
| Mutation X16b et variante `ti=5 i2` (garde-rail seul) | ROUGES |
| vitest `goFixtures.contract.test.ts` | non rejoué : le worktree n'a pas de `node_modules` (les fixtures ne changent que par une chaîne de révision) |

## 10. Revue adverse de la vague 1 (2026-10-03) : la montée de `killsource.Rev` est retirée

**Constat (majeur), vérifié sur pièces.** La montée `killsource-2026-09-27` -> `killsource-2026-10-02`
(correction 3 du contrôle, §9) ne repose que sur des sorties NON persistées, et la même vague applique
la règle dans l'autre sens :
- ce qui change (mesuré, `vague1_tsv/killsource_base_contre_tete.tsv`) : le diagnostic
  `Result.Calibration` (lu par `cmd/killsource` seul) et, sur `111fa685`, deux compteurs de santé
  publiés en expvar (`grammar/killhealth.go`, `Health.ExpvarPairs`, consommé par
  `sync/killcollector/collector_metrics.go` et `cmd/killsource/sante.go`) ; aucune mort, valeur ni
  voie, donc aucune ligne de `match_kill_events` ;
- L8 et L4a ont changé le même diagnostic à révision constante ; le plan déclare la règle « pas de
  montée pour un diagnostic » supposée (§6.0 point 3) ;
- conséquence d'une montée en production : `decoder_rev = killsource.Rev` (`film/decfilm/decfilm.go`)
  ; `conditionBacklog` (`sync/killcollector/postsync.go`) rend tout le parc candidat, et le hook
  post-sync, installé par défaut (`sync/engine_options.go`, `NewPostSyncHook(repoRoot, 0)`,
  `DefaultPostSyncPerCycle = 8`), le redécode de lui-même pour réécrire des lignes identiques.
  L'entrée de chronique disait « jamais automatique ».

**Correction appliquée (proposition de l'intégrateur, À CONFIRMER par l'utilisateur avant tout push,
§6.3 D23 du plan)** : `killsource.Rev` revient à `killsource-2026-09-27`, empreinte recopiée à
révision constante (précédents : L8, L4a, complément DU-7 du 2026-10-02). L'entrée
`killsource-2026-10-02` de `rev_chronique.go` devient le complément du 2026-10-03 (révision
constante) ; `killsource_rev.golden` perd la ligne `killsource-2026-10-02` (régénéré par la commande
du dépôt) ; `shapes.golden` et les 8 fixtures de contrat suivent (identiques hors chaînes de
révision). Si l'utilisateur choisit la montée, elle se reprend avec une valeur qu'aucune branche n'a
portée (pas `killsource-2026-10-02`, valeur de la branche du lot seul) et une entrée qui écrit que le
déploiement redécode le parc par le hook post-sync.

## 11. Corrections des mineurs de la revue (2026-10-03)

Deux constats mineurs de la revue adverse de la vague 1 portent sur ce lot ; tous deux vérifiés sur
pièces et VRAIS. Aucun bit lu ne change (preuve de sortie nulle commune à la vague : `LOT_L4a.md`
§14).

- **Garde-rail du lecteur de minuteur sensible à la mise en page** (constat 3). L'ancien
  `lecteur_minuteur_guard_test.go` cherchait la séquence de `FUN_140d580d0` par une expression sur
  trois lignes consécutives : `Minuteur{A: br.ReadBits(n), B: br.ReadBits(n), Queue: br.ReadBits(5)}`
  sur une ligne passait. Preuve (mutation en place de `vitality.go`, cette ligne à la place de
  l'appel du lecteur) : ancien garde-rail **VERT**, nouveau **ROUGE** (`vitality.go : 1 lecture(s)
  en ligne`). Correction : la suite des appels est lue dans l'arbre syntaxique
  (`grammar/sequence_appels_test.go`, instrument partagé avec le garde-rail du jeu d'armes de L4a) ;
  trois appels consécutifs `ReadBits(x)`, `ReadBits(x)`, `ReadBits(5 | largeurQueueMinuteur)` d'un
  MÊME bloc. La condition « même bloc » n'est pas un assouplissement : sans elle, la lecture
  syntaxique trouvait une séquence que les lignes cachaient — `consumeDeadStateAnimBlock`
  (`components_object.go`), étape 14 `R(14) R(14)` sous la porte `etatMortVitessePresente` puis
  étape 15 `R(5)` inconditionnelle, qui n'est pas un minuteur. Les sauts (`Skip(37|53)`, `Skip` dont
  l'ARGUMENT nomme une largeur de minuteur) passent aussi par l'arbre : un commentaire en fin de
  ligne n'est plus pris pour un argument. Vecteurs : `TestGardeRailMinuteurInsensibleALaMiseEnPage`
  (une ligne, trois lignes avec commentaires, largeurs différentes, lectures sous porte puis lecture
  inconditionnelle, sauts).
- **Champ mort `Minuteur.C`** (constat 5). Le troisième réel de `FUN_142ba78dc` n'était lu que par
  `TestLecteurDeMinuteurRendLesQuanta` ; son seul appelant de production (`consumeManagedEngineTimers`,
  fente `i15` à quatre champs) jette la valeur rendue. Règle 7 : `lireMinuteur142ba78dc` ne rend plus
  rien et consomme ses `3n + 5` bits (`lireMinuteur140d580d0` puis `R(n)`, mêmes lectures, même
  ordre) ; le champ `C` est retiré. Le test vérifie désormais l'arrêt à 53 bits et relit intact un
  marqueur de 3 bits qui suit.

Pièces : `campagne_grammaire_2026-10-01/vague1_tsv/` n'est pas touché ; mutations et sorties au
scratchpad de la session (`integ3/mut/`), recopiées en synthèse dans `LOT_L4a.md` §14.
