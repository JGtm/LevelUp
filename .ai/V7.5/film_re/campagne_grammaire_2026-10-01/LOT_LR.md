# Lot LR — lecteur d'état de création du jeu et second rang sans mutation du monde (2026-10-05)

> Lot de la vague 3 de la campagne de grammaire (`.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`),
> accord du pilote du 2026-10-05, issu de l'enquête de la session levelup-57 sur le lot 2.7.a0 de la
> représentation intermédiaire (`feat/ri-etape2`, locale, non poussée). Critère FERMÉ de
> l'utilisateur : corrections générales lues dans le jeu ; aucun ordre, seuil ni condition choisi à la
> mesure ; gate 2 « aucun film en baisse » jamais assoupli, D2 seulement pour une fermeture factice
> retirée instruite.
>
> Worktree `LevelUp-wt-cg3-lr`, branche `feat/cg3-lr` depuis `87cdfa761` (= `origin/feat/v75`), REBASÉE sur
> `8dfadd07e` le 2026-10-06 (§12, qui fait foi).
> Films lus en place (`LevelUp/data/cache/film_chunks`, lecture seule), aucune cuisson du parc.
> Mesures et scripts : `scratchpad/cg3-lr/` (session `f46f71fc`). GOCACHE `go-build-cg3-lr`.
> Convention : **lu dans le jeu** = relu dans Ghidra (`HaloInfinite.exe`, HTTP 127.0.0.1:8089, lecture
> seule) ; **mesuré** = compté par un outil sur les films ; **estimé** = hypothèse écrite.

## 0. Statut

| Item | Statut | En une ligne |
|---|---|---|
| LR.1 règle du lecteur MPP (compte > 4 → l'état de création échoue → le record NEW n'est pas lu), au lecteur et au juge | [x] écrit | `etat_de_creation.go`, `consumeMultiplayerPropertiesBlock` rend son verdict, `TraverseEntity` arrête le record, `InvariantEtatDeCreationIllisible` au juge ; exception de `ti=41` portée |
| LR.2 le second rang de `debutParFermeture` ne lie plus de NEW et ne délie plus de DEL | [x] écrit | `debut_non_prouve.go` ; annonce posée par `debutParFermetureRangee`, prise par la boucle de la vue B |
| Vecteurs du lecteur du jeu + cas réels cités | [x] | 5 tests unitaires + garde-rail règle 6 ; cas réels `1c4c63c2` 61:42, 17:52 / 17:172 en test de recherche (sous 8/3) |
| Mutations (`-overlay`) | [x] | 10 / 10 ROUGES ; reprise : 13 / 14 avec les mutations du contrôle, M8 équivalente (§12.2), plus le test des cas réels rouge sous la mutation « monde réintroduit » |
| **Gate 2 (carte v2, 20 films, découpage par défaut, base `87cdfa761`)** | **[!] ROUGE** | +259 sains au corpus mais **14 films en baisse** ; `e5adf7b2` perd 23 sains VRAIS par LR.2 (têtes `ti=41` mal lues à 9/5) : non admissible sous D2 (§3, §4) |
| Gate 2 sous `-mpp-declare` | [x] joué à la reprise (§12) | base `8dfadd07e` : +1 311 sains, 12 films en baisse, toutes leurs pertes de la famille A (tête illisible pour le jeu) |
| Révision | [x] | reprise : `grammar-2026-10-06.3` (§12.1) ; `killsource`, `objectives`, `source`, `profile` constantes ; `grammar-2026-10-03.6` jamais fusionné |
| Gates de code (gofmt, vet, vet research, archlint, golangci, G-film) | [x] | tous verts (§8) ; reprise : §12.5 ; G-film après régénération des goldens et fixtures à révision constante (hors `grammar`) |
| killsource 19 témoins | [x] | identique à l'octet hors du diagnostic `calibration` (7 films) (à la base `87cdfa761` comme à la reprise, §12.5) : D23, révision constante |
| replay-equiv, replay-corpus-gate | [x] joués | reprise (§12.5) : gate de corpus rc 1 = `P-1` en `MANQUE` sur les 12 films en baisse ; à la base `87cdfa761` : replay-equiv : `objectives` identique sur 20 films ; gate de corpus rc 1 = `P-1` en `MANQUE` sur 13 témoins, la baisse du gate 2 (§8) |
| **Verdict (reprise du 2026-10-06, §12)** | **RETENU sous D2, proposé au pilote** | à la base `87cdfa761` : non retenu (sections 3 à 9) ; sur `8dfadd07e` sous le découpage déclaré : gate 2 sans baisse hors fermetures factices instruites ; commit sur `feat/cg3-lr`, non poussé |

## 1. Ce qui est lu dans le jeu

**`FUN_14080cfe8`** (le bloc `object-multiplayer-properties`, relu en entier) : le compte R(3) est
comparé à 4 — `14080d238: CMP ECX,0x4 ; JA 0x14080d319`, où `XOR SIL,SIL` pose l'échec puis
`JMP 0x14080d2bb` reprend la lecture : le lecteur lit QUAND MÊME `FUN_14080d4d0` et la queue G3,
puis `14080d327: TEST SIL,SIL ; JZ 0x14080d343` → `XOR R12B,R12B` → rend 0. La liste (R(5) +
`FUN_14080d69c`) n'est lue que si le compte est accepté. Deux autres échecs existent et ne sont PAS
portés : la lecture au-delà du tampon (`*(lecteur+0x18)*8 < *(lecteur+0x2c)`) et le prédicat
`FUN_1404785a0` sur une valeur que `FUN_14080d61c` cherche dans les données du jeu (absentes du film).
Le seuil 5 (compte « 5 ou plus » échoue) est donc celui de l'instruction, pas une mesure.

**Les six appelants de `FUN_14080cfe8`** (`get_xrefs_to`, relus) :

| Lecteur d'état (`vtable+0x60`) | Archétype | Échec du bloc → | Port |
|---|---|---|---|
| `FUN_140f44c38` | bipède 35 | rend 0 (`iVar15 < +0x2c || cVar1 == 0`) | `consumeBipedDefaultState` |
| `FUN_1407f2224` | 36 ; 37 via `FUN_1407f105c` | rend 0 ; `FUN_1407f105c` rend 0 à son tour SANS lire ses deux dernières feuilles | `consumeDefaultStateTI36`, `consumeDefaultStateTI37` |
| `FUN_140fe7630` | 43 | rend 0 | même port que 36 |
| `FUN_1408f0b48` | 38, 39 | rend 0 | `consumeDefaultStateTI38` |
| `FUN_1410a5a74` | véhicule 40 | rend 0 (`cVar2 == 0`) | `consumeDefaultStateTI40` |
| `FUN_1408efb58` | projectile 41 | rend 0 SAUF drapeau 2 posé et `FUN_1406d00ec` = 0xffffffff : `bVar3 = !bVar1`, prédicat sur `dst+0x14` hors du film | `consumeDefaultStateTI41` |

**`FUN_1408f1aa4`** (lecteur de record NEW, appelé par `FUN_1406cbaa0` @1406cc252, la branche que le
jeu emprunte en rejouant un film) : `1408f1c0f: CALL [RAX+0x60]` (cinquième argument 1),
`1408f1c12: TEST AL,AL ; JZ 0x1408f210e` : sur un échec, ni `vtable+0x88`, ni la boucle de composants
`FUN_14076cb60`, ni la création `FUN_1408f2150`. Le corps du record n'est pas lu, l'entité n'est pas
créée. (Le code rendu dans ce cas vaut ESI = 0, comme un succès : l'appelant continue sur un flux
désaligné — un flux que le jeu a écrit ne contient donc pas un tel record.)

**Non lu dans le jeu** : le découpage MPP des formats ≤ 25 (le port garde 9/5 ; 2.7.a0 le lit dans la
taille d'état que le film déclare, 8/3). C'est la source de l'échec du gate 2 (§4).

## 2. Ce qui change (production)

1. **Le lecteur du bloc MPP rend son verdict** : `consumeMultiplayerPropertiesBlock(br) bool`
   (`default_state.go`), `lisible := count <= mppCompteMax` avec `mppCompteMax = 4`
   (`etat_de_creation.go`). Aucun bit lu ne change.
2. **Les lecteurs d'état propagent l'échec comme le jeu** : `lireLeBlocMPPDeLEtat` (bipède, 36/37/43,
   38/39, 40) pose `Lecteur.etatIllisible` ; `ti=41` calcule son propre verdict (exception ci-dessus).
   Garde-rail règle 6 : `TestLeBlocMPPDUnEtatPasseParSonVerdict` (un appel direct du bloc hors de
   l'hôte, de `ti=41` et de la définition rougit).
3. **`TraverseEntity`** remet le drapeau à faux avant l'état, et sur un état illisible arrête le record
   à la fin de son état : `EtatIllisible = true`, `DesyncAt = 0`, ni porte ni masque ni composant. Le
   record n'est pas lié (désynchronisé), la marche s'arrête comme sur tout record infranchissable.
4. **Le juge** (`ecrivain_invariants.go`) : règle `InvariantEtatDeCreationIllisible` (« lecteur : état
   de création illisible »), jugée par record avant le masque ; `NombreDInvariants` 11 → 12. L'en-tête
   dit désormais « la fonction du jeu qui la fonde — l'écrivain, ou le lecteur qui rejoue le film ».
5. **Le second rang** (`debut_non_prouve.go`) : `debutParFermetureRangee` annonce au monde la marche
   qui part de son second rang (`World.marquerDebutNonProuve(pay, bit)`) ; la boucle de la vue B prend
   l'annonce à son entrée (`Lecteur.entrerDansLaVueB`, qui remplace la remise à zéro de la sortie) et
   la retire dans tous les cas ; si elle désigne cette marche (même payload, même bit), la marche est
   non prouvée : `corpsDeRecordNeuf` ne lie pas le NEW (liaison `LiaisonAucune`), le DEL ne délie pas
   (`Lecteur.delier`). Les fichiers de la vue A (`marche_trames*.go`) ne sont pas touchés.

**Ce qui change exactement au second rang, et ce qui ne change pas** : les records sont lus, traversés,
rendus et comptés comme avant ; leurs états, positions (`SetPos`), crochets et le tir continu (vue C)
sont publiés comme avant ; le refus d'un NEW contre une entité vivante est compté comme avant ; la
liaison par anticipation (`rejetDeVue`, issue d'une image-clé ultérieure) reste posée ; la liaison
d'inférence d'un DELTA non lié (`BindSoft`, inactive en production : `InferenceChaine` faux et
`TablesParVue` rejette avant) n'est pas touchée. Seuls changent : le NEW n'est plus LIÉ, le DEL ne
DÉLIE plus. Le texte du repli au registre (`registre_filmdec_marche.go`) le dit.

Fichiers touchés : `grammar/{default_state,default_state_arch,default_state_ti40,default_state_ti41,
traverse,ecrivain_invariants,lecteur,world,frame_infer,debut_de_liste,rev,rev_chronique}.go`,
`grammar/testdata/grammar_rev.golden`, `facts/fallback/registre_filmdec_marche.go`, goldens à révision constante `facts/{killsource,objectives}/testdata/*_rev.golden`, `types/testdata/shapes.golden`, les 8 fixtures de contrat `apps/web/src/features/match-replay/test/fixtures/go/` et leur `manifest.json` (chaînes de révision seules) ; neufs
`grammar/etat_de_creation.go`, `grammar/debut_non_prouve.go`. Aucun fichier de la vue A ni de la
liste 2.7.a0. `debut_de_liste.go` : une ligne de code (`w.marquerDebutNonProuve`) et le commentaire
du second rang.

Tests neufs : `etat_de_creation_test.go` (vecteurs du bloc pour les comptes 0 à 7 avec le littéral 4
de l'instruction ; NEW `ti=36` au compte 5 contre son témoin au compte 4 ; quatre cas de `ti=41` ;
garde-rail), `debut_non_prouve_test.go` (la marche du second rang ne modifie pas le monde, son témoin
lie et délie ; l'annonce ne vaut que pour la marche qui suit). Sondes de recherche (tag `research`) :
`lr_research_test.go` (listes du second rang, carte par paquet sous `LT_MPP`, cas réels),
`lr_instruire_research_test.go` (origine des liaisons des paquets perdus, DEL du second rang),
`lr_tetes_research_test.go` (relecture d'un NEW sous la grammaire du lot).

## 3. Gate 2 — carte v2, 20 films, découpage par défaut (base `87cdfa761` contre la tête)

`cmd_fermeture -mode v2 -denominateur-fixe -paquets -plafond-gib 4`, binaires des deux arbres ;
`gate2.awk` de la campagne (sain = fermé au sens de L0).

| Film | Sains avant | après | Net | Perdus | Gagnés | Utiles sains avant | après | Net |
|---|---|---|---|---|---|---|---|---|
| `0797ce72` | 19156 | 19155 | -1 | 1 | 0 | 159929 | 159929 | +0 |
| `084a804d` | 4837 | 4811 | -26 | 26 | 0 | 89755 | 89309 | -446 |
| `111fa685` | 4026 | 4024 | -2 | 2 | 0 | 42896 | 42886 | -10 |
| `11de8353` | 5631 | 5628 | -3 | 3 | 0 | 65658 | 65655 | -3 |
| `1c4c63c2` | 13389 | 13236 | -153 | 154 | 1 | 165987 | 165548 | -439 |
| `396cfc92` | 23131 | 23130 | -1 | 1 | 0 | 169502 | 169502 | +0 |
| `4f77afc1` | 24106 | 24607 | +501 | 17 | 518 | 607967 | 623437 | +15470 |
| `50247b26` | 139 | 139 | +0 | 0 | 0 | 274 | 274 | +0 |
| `51ebbc0f` | 20444 | 20442 | -2 | 2 | 0 | 140632 | 140632 | +0 |
| `60ae07c4` | 13948 | 13944 | -4 | 4 | 0 | 83685 | 83678 | -7 |
| `a349fea8` | 424 | 424 | +0 | 0 | 0 | 3654 | 3654 | +0 |
| `a521164d` | 692 | 692 | +0 | 0 | 0 | 78 | 78 | +0 |
| `bcb6d393` | 5880 | 5880 | +0 | 0 | 0 | 35502 | 35502 | +0 |
| `bf15f7ab` | 28603 | 28602 | -1 | 1 | 0 | 216104 | 216098 | -6 |
| `bfecd02b` | 27668 | 27667 | -1 | 1 | 0 | 239064 | 239062 | -2 |
| `c75f33b8` | 23872 | 23867 | -5 | 5 | 0 | 153997 | 153973 | -24 |
| `d9781168` | 26317 | 26311 | -6 | 6 | 0 | 180854 | 180844 | -10 |
| `e5adf7b2` | 4149 | 4113 | -36 | 36 | 0 | 80107 | 79127 | -980 |
| `f75e7053` | 23673 | 23673 | +0 | 0 | 0 | 162137 | 162137 | +0 |
| `fb1a1a72` | 43457 | 43456 | -1 | 1 | 0 | 325815 | 325815 | +0 |
| **corpus** | **313542** | **313801** | **+259** | **260** | **519** | **2923597** | **2937140** | **+13543** |

Factices des gains au bit : 30 sur 542 (5,5 %) ; ils ne sont pas comptés sains.

**Les deux moitiés, mesurées séparément** (mêmes binaires, une moitié retirée par `-overlay`) :

| Variante | Sains corpus | Perdus | Gagnés | Utiles sains | Films en baisse |
|---|---|---|---|---|---|
| LR.1 seule (gel du second rang retiré) | -258 | 258 | 0 | -1 943 | 15 (dont `1c4c63c2` -170, `084a804d` -26, `e5adf7b2` -19, `4f77afc1` -16) |
| LR.2 seule (règle du lecteur neutralisée dans `TraverseEntity`) | +491 | 29 | 520 | +14 846 | 2 : `e5adf7b2` -23 / -747, `1c4c63c2` -4 (utiles +26) |

**Juge sur les GAGNÉS** (LR.2) : les 518 sains gagnés de `4f77afc1` (chunks 9, 39, 52) sont des
paquets qui s'arrêtaient en base sur « vue B : sortie par rejet » (423) ou « liste non localisée »
(21). Instruit sur 9:878 : en base, le slot 2048 est lié à `ti=0` par un NEW du SECOND RANG (8:736 bit
2480) ; le vrai NEW `ti=37` du slot 2048 (9:874, chaîne de tête) est alors REFUSÉ (« contredit une
entité vivante »), et les DELTA de 2048 se lisent sous `ti=0` jusqu'au rejet. Avec LR.2, 2048 n'est
pas lié à 8:736, le NEW de 9:874 se lie, les paquets ferment sans règle contredite. C'est, sur un film
HI_1_13_0 au découpage lu dans le jeu, la famille du cas 17:52 / 17:172 (§5.2).

## 4. Instruction des 260 sains perdus (découpage par défaut)

Sondes `TestLRInstruire` (base et tête : rang du début, sortie, eid rejeté, origine de chaque
liaison) puis `TestLRTetes` (relecture, sous la grammaire du lot, du NEW de tête de la base ou du NEW
qui a posé la liaison). Pièces : `scratchpad/cg3-lr/{instr_base,instr_tete,classes_relues.tsv}`.

| Classe | Paquets | Établi |
|---|---|---|
| A : la base ouvrait la liste sur un NEW (rang 3 ou 4) que la tête relit **illisible pour le jeu** | 201 | mesuré (relecture) ; 196 ont un mot de 32 bits du bloc **inconnu** du film (6 « connus » sur `1c4c63c2`) |
| A' : la base lisait, après la tête, un NEW illisible (`4f77afc1` 40:102 : NEW 6035 `ti=37` au bit 2955) | 1 | mesuré |
| B : la base lisait des DELTA d'un slot lié par un NEW que la tête relit illisible | 32 | mesuré |
| C : la base lisait des DELTA d'un slot lié par un NEW du **second rang** (LR.2) | 26 | mesuré : `e5adf7b2` 23 (NEW `ti=41` 1076 de 10:20, 1096 de 10:848), `4f77afc1` 1 (59:1006), `1c4c63c2` 2 (6:2022, 49:2242 : slot 3774 lié à `ti=0` par 6:1216 et 49:2042) |

- **A, A', B (234) : fermetures qui passent par une lecture que le jeu ne fait pas.** Sur les films
  HI_1_13_0 (`0797ce72`, `396cfc92`, `4f77afc1`, `51ebbc0f`, `bf15f7ab`, `bfecd02b`, `c75f33b8`,
  `d9781168`, `fb1a1a72`), le découpage 9/5 est celui du jeu : ces fermetures sont FACTICES au sens du
  lecteur du jeu (D2 instruite). Sur les formats anciens (`1c4c63c2`, `084a804d`, `e5adf7b2`,
  `111fa685`, `11de8353`, `60ae07c4`), le compte est lu à 9/5, découpage que le film ne déclare pas :
  148 de ces pertes sont dans les 428 pertes que l'enquête de 2.7.a0 a instruites (418 « faux sains de 9/5 »)
  (`scratchpad/agentA/pertes_tous.tsv` de la session levelup-57) ; sous 8/3, 136 des 211 NEW relus
  redeviennent lisibles (§5) — l'illisibilité y est un effet du découpage par défaut, pas un fait du
  jeu.
- **C (26) : NON factices pour `e5adf7b2`.** LT §4.2 l'a établi : les têtes `ti=41` de 10:20 et
  10:848 sont de vrais NEW dont le corps est mal lu à 9/5 ; sous 8/3, 10:848 se localise au premier
  rang. Leurs liaisons portaient 23 paquets sains vrais (10:960 à 10:1024). **C'est la perte qui rend
  le gate 2 ROUGE sous le critère.** Les deux de `1c4c63c2` reposent sur des liaisons `ti=0` du second
  rang (non instruites comme factices) ; celle de `4f77afc1` est sur un film en hausse.

Verdict du gate 2 au découpage par défaut : **ROUGE**. `e5adf7b2` est en baisse par des pertes non
factices ; les 13 autres films en baisse ne perdent que des fermetures passant par un NEW illisible
au découpage appliqué (A, A', B), dont l'admission au titre de D2 pour les formats anciens dépend du
découpage (pilote).

## 5. Sous le découpage 8/3 (MESURE DE RECHERCHE, `LT_MPP=8/3`, pas le gate officiel)

`-mpp-declare` n'existe pas sur une branche poussée : **la mesure du gate officiel sous 8/3 reste à
faire**. La sonde `TestLRCarte` rejoue la marche de la carte ([cmMarcher]) ; contrôle : au découpage
par défaut elle redonne la carte officielle, sains par film, À L'UNITÉ, en base comme en tête.
`LT_MPP` impose 8/3 à tous les builds hors HI_1_12_0 / HI_1_13_0 (y compris `a349fea8`, `50247b26`,
`a521164d`, que `-mpp-declare` traite selon leur déclaration) : c'est une approximation de 2.7.a0.

### 5.1 Gate 2 de recherche

| Film | Sains avant | après | Net | Perdus | Gagnés | Utiles avant | après | Net |
|---|---|---|---|---|---|---|---|---|
| `0797ce72` | 19156 | 19155 | -1 | 1 | 0 | 159929 | 159929 | +0 |
| `084a804d` | 23642 | 23784 | +142 | 24 | 166 | 632653 | 638098 | +5445 |
| `111fa685` | 11932 | 11955 | +23 | 2 | 25 | 244187 | 244820 | +633 |
| `11de8353` | 13839 | 13831 | -8 | 8 | 0 | 268363 | 268308 | -55 |
| `1c4c63c2` | 33999 | 34678 | +679 | 126 | 805 | 682796 | 704007 | +21211 |
| `396cfc92` | 23131 | 23130 | -1 | 1 | 0 | 169502 | 169502 | +0 |
| `4f77afc1` | 24106 | 24607 | +501 | 17 | 518 | 607967 | 623437 | +15470 |
| `50247b26` | 138 | 138 | +0 | 0 | 0 | 313 | 313 | +0 |
| `51ebbc0f` | 20444 | 20442 | -2 | 2 | 0 | 140632 | 140632 | +0 |
| `60ae07c4` | 34578 | 34574 | -4 | 4 | 0 | 253818 | 253813 | -5 |
| `a349fea8` | 428 | 427 | -1 | 1 | 0 | 3654 | 3654 | +0 |
| `a521164d` | 693 | 693 | +0 | 0 | 0 | 78 | 78 | +0 |
| `bcb6d393` | 5880 | 5880 | +0 | 0 | 0 | 35502 | 35502 | +0 |
| `bf15f7ab` | 28603 | 28602 | -1 | 1 | 0 | 216104 | 216098 | -6 |
| `bfecd02b` | 27668 | 27667 | -1 | 1 | 0 | 239064 | 239062 | -2 |
| `c75f33b8` | 23872 | 23867 | -5 | 5 | 0 | 153997 | 153973 | -24 |
| `d9781168` | 26317 | 26311 | -6 | 6 | 0 | 180854 | 180844 | -10 |
| `e5adf7b2` | 12267 | 12264 | -3 | 3 | 0 | 302245 | 302195 | -50 |
| `f75e7053` | 23673 | 23673 | +0 | 0 | 0 | 162137 | 162137 | +0 |
| `fb1a1a72` | 43457 | 43456 | -1 | 1 | 0 | 325815 | 325815 | +0 |
| **corpus** | **397823** | **399134** | **+1311** | **203** | **1514** | **4779610** | **4822217** | **+42607** |

Instruction des 203 pertes (même méthode, `scratchpad/cg3-lr/classes83_relues.tsv`) : 194 passent par
une tête NEW illisible pour le jeu (le compte relu à 8/3), 6 par une tête lisible, 3 par une liaison du
second rang (`ti=41` : `084a804d` 23:8, `1c4c63c2` 27:1138, `4f77afc1` 59:1006). **Les 9 pertes non
factices sont toutes sur des films en hausse** (`1c4c63c2` +679, `4f77afc1` +501, `084a804d` +142).
**Tous les films en baisse** (`e5adf7b2` -3, `60ae07c4` -4, `11de8353` -8, `a349fea8` -1 et les huit
HI_1_13_0 ci-dessus) **ne perdent que des fermetures passant par une tête illisible pour le jeu** :
sous 8/3 le gate 2 tiendrait avec D2 (estimé ; à confirmer par la carte officielle sous
`-mpp-declare`).

### 5.2 Les cas réels cités (`1c4c63c2`, sonde `TestLRSecondRang`, slots 526 et 3331 après chaque paquet)

| Paquet | Base, 8/3 | Tête, 8/3 |
|---|---|---|
| 17:52 (second rang, début bit 5420 : NEW 3501 `ti=37`, NEW 3331 `ti=32` au bit 5982) | 3331 lié à `ti=32` | 3331 non lié (le NEW est lu, non lié) |
| 17:172 (signature, NEW 3331 `ti=10` au bit 2) | NEW refusé, 3331 reste `ti=32` | 3331 lié à `ti=10` |
| 17:232 | non fermé au bit | **fermé** (sain retrouvé) |
| 61:42 (second rang, début bit 4607 : NEW 5665 `ti=43`, DEL 526 au bit 4871) | 526 délié (bipède vivant) | 526 reste lié à `ti=35` |

Au découpage par défaut, 17:52 n'est pas localisé et 61:42 (second rang au bit 5069) ne lit pas le DEL
de 526 : les cas n'existent que sous 8/3. Test : `TestLRCasReelsDuSecondRang` (recherche, exige
`LT_MPP=8/3`) VERT sur la tête, ROUGE sous la mutation « monde réintroduit » (les trois assertions).

### 5.3 Compteur `repli_debut_de_liste_ferme_au_bit` par film (production, `TestLTRepliParFilm`, découpage par défaut)

| Film | Base | Tête | | Film | Base | Tête |
|---|---|---|---|---|---|---|
| `0797ce72` | 20 | 20 | | `51ebbc0f` | 90 | 75 |
| `084a804d` | 472 | 366 | | `60ae07c4` | 108 | 86 |
| `111fa685` | 141 | 111 | | `a349fea8` | 44 | 39 |
| `11de8353` | 128 | 104 | | `a521164d` | 5 | 5 |
| `1c4c63c2` | 6428 | 5674 | | `bcb6d393` | 5 | 5 |
| `396cfc92` | 9 | 9 | | `bf15f7ab` | 10 | 7 |
| `4f77afc1` | 1013 | 904 | | `bfecd02b` | 22 | 16 |
| `50247b26` | 14 | 12 | | `c75f33b8` | 93 | 76 |
| `d9781168` | 261 | 204 | | `e5adf7b2` | 159 | 131 |
| `f75e7053` | 8 | 7 | | `fb1a1a72` | 39 | 33 |
| **corpus** | **9069** | **7884** | | | | |

Le repli reste nommé et compté ; il baisse de 1 185 par LR.1 (des candidats dont la tête est illisible
ne ferment plus au bit près).

## 6. Décision proposée au pilote

1. **Ne pas fusionner LR sur `87cdfa761`** : le gate 2 au découpage par défaut est rouge sur
   `e5adf7b2` par des pertes vraies (LR.2), et LR.1 appliquée à 9/5 sur les formats anciens juge un
   compte lu à un découpage que le film ne déclare pas.
2. **Rejouer le gate 2 officiel sous `-mpp-declare`** dès que 2.7.a0 est poussé (surcouche des fichiers
   de LR sur `feat/ri-etape2`, aucun fichier en commun) ; la mesure de recherche §5.1 prédit aucune
   baisse non factice.
3. **Intégrer LR avec ou après 2.7.a0** si ce gate tient. LR.1 et LR.2 ne sont PAS séparables sans
   perte : LR.1 seule ne gagne rien (-258) ; LR.2 seule fait baisser `e5adf7b2` à 9/5.

## 7. Révision

`grammar-2026-10-03.5` → **`grammar-2026-10-03.6`**. La chronique exige qu'une date neuve commence au
rang 1 (`Rang.Suit`) : `grammar-2026-10-05` (sans suffixe) est porté par `feat/ri-etape2` et par le
worktree `feat/cg3-vue-a`, `grammar-2026-10-04` par LN et la RI ; le rang suivant de la série
courante, `.6`, n'est porté par aucune branche (relevé des worktrees et branches locales). Entrée de
chronique dans `rev_chronique.go`, golden régénéré.

`killsource.Rev` (D23) : sortie `cmd/killsource json` identique à l'octet sur 19 / 19 témoins hors du
diagnostic non persisté `calibration` (scores du profil plat, 7 films : `11de8353`, `396cfc92`,
`50247b26`, `a349fea8`, `a521164d`, `c75f33b8`, `e5adf7b2`) → révision constante `killsource-2026-09-27`,
empreinte recopiée. `objectives.Rev` : section `objectives` identique sur les 20 films de
`replay-equiv` → révision constante `objectives-2026-09-27`, empreinte recopiée. `source.Rev`,
`profile.Rev` : non touchées. `replay.SchemaVersion` reste 78.

## 8. Sorties des gates

| Gate | Sortie |
|---|---|
| `gofmt -l` (film) | vide |
| `go vet ./...` | rc 0 |
| `go vet -tags=research ./internal/games/halo_infinite/film/...` | rc 0 |
| `go test ./internal/archlint/` | `ok` (49,7 s) |
| `golangci-lint run --new-from-rev=87cdfa761` (grammar, facts/fallback ; puis grammar sous `--build-tags=research`) | `0 issues.` (un `unused` de la sonde corrigé en route) |
| G-film (`film/...`, `replaybuild/...`, `sync/killcollector/...`, `-count=1 -timeout 30m`) | rc 0, 21 paquets `ok` — après régénération, À RÉVISION CONSTANTE sauf `grammar`, des goldens `killsource_rev`, `objectives_rev`, `shapes.golden` et des 8 fixtures de contrat (décompressées : identiques hors des chaînes `grammar-2026-10-03.5` → `.6`) |
| Mutations | 10 / 10 ROUGES (§9) |
| Carte v2 (gate 2) | ROUGE au découpage par défaut (§3, §4) ; recherche 8/3 §5 |
| killsource json, 19 témoins | 19 rc 0 ; identique à l'octet hors du diagnostic `calibration` (7 films) |
| replay-equiv, 20 films (base `87cdfa761` contre tête, deux passes chacun) | rc 1 des deux côtés (verdict du harnais sur `artifact`). Étapes divergentes base → tête : `artifact` 20 (révision de grammaire), `movementStates.stats` 19, `continuousFire.stats` 19, `killsource` 14 (champ non exporté `Kill.paquet`, cf. `deux_passes.go` ; `killRefs` identique), `vehicles` 10, `movementStates` 6, `continuousFire` 6, `birthLoadouts.stats` 1 (`50247b26`). **Identiques sur les 20** : `objectives`, `score`, `killRefs`, `deaths`, `positions`, `fire`, `heldWeaponChanges`, `pickups`, `placements`, `zones` et toutes les autres étapes → `objectives.Rev` constante, `replay.SchemaVersion` reste 78 |
| replay-corpus-gate `--base=87cdfa761` | **rc 1** : banc de vérité `MANQUE` sur 13 témoins, toujours et seulement par `P-1 paquets fermés` (`fb1a1a72` 43457 → 43456, `d9781168` -6, `c75f33b8` -5, `bf15f7ab` -1, `51ebbc0f` -2, `084a804d` -26, `0797ce72` -1, `111fa685` -2, `e5adf7b2` -36, `60ae07c4` -4, `11de8353` -3, `bfecd02b` -1, `396cfc92` -1 ; même instruction qu'au §4) ; `ok` sur 6 (`bcb6d393`, `a349fea8`, `a521164d`, `50247b26`, `4f77afc1`, `f75e7053`). Aucune mesure V-* ni O-* en `MANQUE` ou `FAUX`. `[info] R-1 repli_debut_de_liste_ferme_au_bit` en baisse sur tous les témoins qui le portent. 198 lignes `[FILET]` de couverture (tir continu, postures). Télémétrie : `grammarRev .5 → .6` (19) |

`P-1` du gate de corpus et la carte v2 comptent le même objet : les deux disent la même baisse, film
par film. Le rc 1 du gate de corpus est donc la conséquence directe du gate 2 rouge, et il n'est pas
admissible pour la même raison (`e5adf7b2`).

## 9. Mutations (`-overlay`, `scratchpad/cg3-lr/mut/`)

| Mutation | Résultat |
|---|---|
| M1 règle retirée au lecteur (`lisible := true`) | ROUGE : `TestLeBlocMPPEchoueAuDelaDeQuatre`, `TestUnNeufDontLEtatEchoueNEstPasLu`, `TestLEtatDuProjectileSuitSonPropreVerdict` |
| M1b règle retirée à l'état `ti=36` (appel direct du bloc) | ROUGE : `TestUnNeufDontLEtatEchoueNEstPasLu` (et le garde-rail) |
| M1c règle retirée à la traversée | ROUGE : `TestUnNeufDontLEtatEchoueNEstPasLu` |
| M1d règle retirée au juge | ROUGE : `TestUnNeufDontLEtatEchoueNEstPasLu` |
| M1e exception de `ti=41` retirée | ROUGE : `TestLEtatDuProjectileSuitSonPropreVerdict` |
| M2 seuil `mppCompteMax = 5` | ROUGE : les trois tests de la règle |
| M2b seuil strict (`count < mppCompteMax`) | ROUGE : les trois tests de la règle |
| M3 mutation du monde réintroduite (annonce retirée) | ROUGE : `TestLaMarcheDuSecondRangNeModifiePasLeMonde` ; `TestLRCasReelsDuSecondRang` (recherche, 8/3) ROUGE sur ses trois cas |
| M3b DEL délié au second rang | ROUGE : `TestLaMarcheDuSecondRangNeModifiePasLeMonde` |
| M3c NEW lié au second rang | ROUGE : `TestLaMarcheDuSecondRangNeModifiePasLeMonde` |

Note de mesure : la mutation M1 change aussi la consommation de bits (la liste est lue au-delà de 4) ;
elle sert de mutation, pas de variante de mesure. La variante « LR.2 seule » du §3 est M1c.

## 10. Écarts

- **D-LR-1** : `FUN_1407f105c` (`ti=37`) ne lit pas ses deux dernières feuilles quand l'état échoue ;
  le port les lit quand même (et publie la création d'équipement). Sans effet sur un record NEW (arrêté
  à la fin de son état) ; la marche d'image-clé et la marche de création d'équipement
  (`equipment_creation.go`, qui appelle le lecteur d'état sans `TraverseEntity`) ne consultent pas le
  verdict : la règle n'y est pas portée (hors périmètre, non mesuré).
- **D-LR-2** : les deux autres échecs de `FUN_14080cfe8` (lecture au-delà du tampon, prédicat de
  `FUN_14080d61c`) ne sont pas portés : le second dépend de données du jeu absentes du film.
- **D-LR-3** : un début de second rang égal à l'amorce du paquet (`PacketPreambleBits`) serait lu
  depuis la tête par `lireTrameParRangs` ; l'annonce ne désigne alors pas la marche et le gel ne
  s'applique pas (cas non observé, non mesuré).
- **D-LR-4** : le commentaire de `consumeDefaultStateTI37` disait que `FUN_1407f2224` ne vaut 0 qu'en
  dépassement de tampon : faux (règle 17), corrigé.
- **D-LR-5** : scripts de scratchpad écrits en partie avec `python3` (génération des surcouches de
  mutation, éditions) avant d'être repassés en awk/sed ; rien de Python n'est versionné.
- **D-LR-6** : `replay-equiv` cuit les 20 films du corpus (recette de la campagne) ; aucune cuisson du
  parc.

## 11. Fichiers d'autres chantiers touchés

Aucun fichier de la vue A (`vue_a_*.go`, `marche_trames*.go`, `distribuer_tetes.go`,
`lecture/paquet.go`, `film_context.go`, `frame_vue_messages.go`) ni de 2.7.a0 (`profile/`,
`build_profile.go`, `film_format_version.go`, `equipment_placements.go`, `keyframe_fullstate_loop.go`,
`replay/build_from_film.go`, `build_ground_weapons.go`, `facts/fallback/registre_filmdec.go`,
`entete.go`). `debut_de_liste.go` (que V2 touchera) : une ligne dans `debutParFermetureRangee` et son
commentaire. `world.go` : un champ. `frame_infer.go` : trois lignes (`corpsDeRecordNeuf`, entrée de
`decodeInferLoop`, branche DEL). Points de fusion probables avec la vue A : `rev.go`,
`rev_chronique.go`, `grammar_rev.golden`, `ecrivain_invariants.go` si V1-V3 y ajoutent des règles
(numérotation de `InvariantEcrivain`).

## 12. Reprise sur `8dfadd07e` (2026-10-06)

> Base : `origin/feat/v75` = `8dfadd07e` (2.7.a, 2.7.a0 — découpage MPP déclaré par le film, drapeau
> `-mpp-declare` de `cmd_fermeture` —, vue A V1). Rebase du commit WIP `e00beafdc` ; scratchpad de la
> reprise : `scratchpad/cg3-lr2/` (session `f46f71fc`). Les sections 0 à 11 décrivent la mesure à la
> base `87cdfa761` et restent comme trace ; CETTE section fait foi pour la reprise.

### 12.1 Rebase

Conflits sur les seuls fichiers générés et la révision : `rev.go`, `rev_chronique.go`,
`grammar_rev.golden`, `killsource_rev.golden`, `objectives_rev.golden`, `shapes.golden`, les fixtures de
contrat et leur manifeste (la base est passée au schéma 79 : les sept `replay_schema_78_*` modifiés par
le WIP sont supprimés côté base). Résolution : la base a raison sur tous les fichiers générés, puis
régénération par les commandes du dépôt. Révision : `grammar-2026-10-03.6` (jamais fusionnée) →
**`grammar-2026-10-06.3`**, le rang libre qui suit `grammar-2026-10-06.2` ; entrée de chronique
réécrite contre `.2` (`rev_chronique.go`, 491 lignes). Aucun conflit dans le code de LR.

Régénérés : `grammar_rev.golden` (`grammar-2026-10-06.3`), `killsource_rev.golden` et
`objectives_rev.golden` à révision CONSTANTE (empreinte recopiée, preuve §12.4), `shapes.golden`
(ligne des révisions), les 8 fixtures `replay_schema_79_*` et `manifest.json` (décompressées :
identiques à `8dfadd07e` une fois `grammar-2026-10-06.N` neutralisé), `testdata/000d5950.golden` de
killsource (une ligne : diagnostic `calibration`, ORACLE « médiane 96 → 95 », décision et morts
identiques ; la base passe `TestGoldenFilms`, la tête le passe après régénération).

### 12.2 Corrections du contrôle

| # | Correction | Statut | Pièce |
|---|---|---|---|
| 1 | Gate 2 officiel sous `-mpp-declare`, intégration après 2.7.a0 | [x] | §12.3 ; base `8dfadd07e` porte 2.7.a0 |
| 2 | Table des lecteurs d'état : `ti=42` (`FUN_1407f0c68`) manquait | [x] | `etat_de_creation.go` : ligne de table et paragraphe. Relu (décompilé) le 2026-10-06 : `cVar1 = FUN_1407f2224(...)`, toutes les feuilles lues, puis `if (*(lecteur+0x18)*8 < *(lecteur+0x2c) \|\| cVar1 == 0) → 0`. Le port y appliquait déjà la règle par `consumeDefaultStateTI36` ; documentation seule |
| 3 | Mutation M1 verte (marche non prouvée jugée avant la contradiction) | [x] | `TestUneMarcheNonProuveeCompteLeNeufQuiContreditUnVivant` (`debut_non_prouve_test.go`) : NEW du slot vivant 124 relu en `ti=2` dans une marche annoncée non prouvée → refus compté (`NeufsContreUnVivant` = 1, `{124 2 4}` en attente), 124 reste `ti=4`, le NEW 300 n'est pas lié. M1 ROUGE |
| 4 | Mutation M2 verte (bit ignoré par `prendreDebutNonProuve`) | [x] | `TestLAnnonceDesigneUnBitDuPayload` : même payload, annonce au bit 1, marche du bit 0 → la marche lie et délie. M2 ROUGE |
| 5 | Mutations M3 et M8 équivalentes en production | [x] | M3 (remise à faux de `etatIllisible` retirée) : commentaire dans `TraverseEntity` ET test `TestLeVerdictDEtatEstCeluiDuRecord` (un état illisible lu hors traversée, puis un NEW lisible sur le même lecteur) : M3 ROUGE. M8 (`marcheNonProuvee` remis à faux à chaque entrée) : commentaire dans `entrerDansLaVueB` (un lecteur par trame et une seule boucle de vue B sous `ClassesDeVue`, vérifié : `marche_trames.go` crée `LecteurSur(pay)` par trame, `lireTrameParRangs` appelle une fois `decodeInferLoop`) ; M8 reste VERTE, équivalente |
| 6 | D-LR-5 (Python dans le scratchpad) | [x] | rien de Python dans la reprise : surcouches de mutation en `sed` (`cg3-lr2/mut/mut.sh`), tableaux en `awk` |
| 7a | `InvariantEtatDeCreationIllisible` jamais décisive | [x] | mesuré sous `-mpp-declare`, carte de la tête : la règle apparaît dans 153 paquets, jamais en première règle, jamais sur un paquet fermé au bit près, jamais sur un paquet sain (`TraverseEntity` arrête déjà ces records). Elle documente le verdict, elle ne décide aucune fermeture |
| 7b | La marche non prouvée modifie encore le monde par l'anticipation | [x] | `debut_non_prouve.go` : un DELTA d'un slot non lié y passe par `rejetDeVue` → `World.LierParRepliDAnticipation` (liaison `Soft` déclarée par une image-clé ULTÉRIEURE), qui reste posée ; seuls NEW et DEL sont gelés |

Correction hors liste, imposée par le gate de lint : `golangci-lint` (étiquettes par défaut) signalait
sur la tête `consumeDefaultStateTI41 - param5 always receives true (unparam)` ; la base, sous les
mêmes étiquettes, ne le signalait pas, mais le signale sous `--build-tags=research` (constat : la base
porte déjà le défaut ; cause du déclenchement sur la tête non élucidée). Le paramètre est retiré : le
port n'a qu'une route, le lecteur de record NEW, qui passe 1 (`FUN_1408f1aa4`, `CALL [RAX+0x60]`
@1408f1c0f, lu au §1) ; `consume1408eff64(br, true)`. Appelants mis à jour : `traverse.go`,
`default_state_ti41_test.go` (2), `etat_de_creation_test.go`, `campagne_invariants_research_test.go`.
Aucune sortie ne change (argument constant).

Sondes de recherche : `LT_MPP` remplacé dans les sondes du lot par `LR_MPP_DECLARE=1` (`lrOuvrir`,
`lr_research_test.go`), qui pose le découpage que la grammaire résout, règle de `-mpp-declare`.
CONTRÔLE : `TestLRCarte` sous `LR_MPP_DECLARE=1` redonne la carte officielle paquet par paquet, base ET
tête (397 824 / 399 135 sains, 0 écart de verdict, 0 écart d'utiles). `TestLRCasReelsDuSecondRang`
exige désormais `LR_MPP_DECLARE=1`.

### 12.3 Gate 2 OFFICIEL — carte v2 sous `-mpp-declare`, base `8dfadd07e` contre la tête

`cmd_fermeture -mode v2 -denominateur-fixe -paquets -plafond-gib 4 -mpp-declare`, 20 films, binaires
des deux arbres (`cg3-lr2/carte.sh`), `gate2.awk` de la campagne (sain = fermé au sens de L0 : au bit
près ET aucune règle de l'écrivain contredite). Découpage retenu (`mpp_declare.tsv`, IDENTIQUE base et
tête) : 9/5 `relu` pour les onze films du format 27 ; 8/3 `presume_par_mesure` pour les huit films des
formats 20, 21, 24, 25 (0 record discordant) ; `50247b26` ne déclare rien de cohérent (173 records, 53
discordants) : non posé, profil par défaut.

| Film | Découpage | Sains avant | après | Net | Utiles sains avant | après | Net | Perdus bruts (utiles) | Gagnés | Gains au bit (factices) | Familles des pertes |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `0797ce72` | 9/5 | 19156 | 19155 | -1 | 159929 | 159929 | +0 | 1 (0) | 0 | 0 (0) | A 1 |
| `084a804d` | 8/3 | 23642 | 23784 | +142 | 632653 | 638098 | +5445 | 24 (365) | 166 | 166 (1) | A 23, C 1 |
| `111fa685` | 8/3 | 11932 | 11955 | +23 | 244187 | 244820 | +633 | 2 (11) | 25 | 25 (0) | A 2 |
| `11de8353` | 8/3 | 13839 | 13831 | -8 | 268363 | 268308 | -55 | 8 (55) | 0 | 0 (0) | A 8 |
| `1c4c63c2` | 8/3 | 33999 | 34678 | +679 | 682796 | 704007 | +21211 | 126 (565) | 805 | 837 (51) | A 120, C 6 |
| `396cfc92` | 9/5 | 23131 | 23130 | -1 | 169502 | 169502 | +0 | 1 (0) | 0 | 0 (0) | A 1 |
| `4f77afc1` | 9/5 | 24106 | 24607 | +501 | 607967 | 623437 | +15470 | 17 (118) | 518 | 518 (7) | A 15, Ap 1, C 1 |
| `50247b26` | défaut (non déclaré, non posé) | 139 | 139 | +0 | 274 | 274 | +0 | 0 (0) | 0 | 0 (0) | - |
| `51ebbc0f` | 9/5 | 20444 | 20442 | -2 | 140632 | 140632 | +0 | 2 (0) | 0 | 0 (0) | A 2 |
| `60ae07c4` | 8/3 | 34578 | 34574 | -4 | 253818 | 253813 | -5 | 4 (5) | 0 | 0 (0) | A 4 |
| `a349fea8` | 8/3 | 428 | 427 | -1 | 3654 | 3654 | +0 | 1 (0) | 0 | 0 (0) | A 1 |
| `a521164d` | 8/3 | 693 | 693 | +0 | 78 | 78 | +0 | 0 (0) | 0 | 0 (0) | - |
| `bcb6d393` | 9/5 | 5880 | 5880 | +0 | 35502 | 35502 | +0 | 0 (0) | 0 | 0 (0) | - |
| `bf15f7ab` | 9/5 | 28603 | 28602 | -1 | 216104 | 216098 | -6 | 1 (6) | 0 | 0 (0) | A 1 |
| `bfecd02b` | 9/5 | 27668 | 27667 | -1 | 239064 | 239062 | -2 | 1 (2) | 0 | 0 (0) | A 1 |
| `c75f33b8` | 9/5 | 23872 | 23867 | -5 | 153997 | 153973 | -24 | 5 (24) | 0 | 0 (0) | A 5 |
| `d9781168` | 9/5 | 26317 | 26311 | -6 | 180854 | 180844 | -10 | 6 (10) | 0 | 0 (0) | A 6 |
| `e5adf7b2` | 8/3 | 12267 | 12264 | -3 | 302245 | 302195 | -50 | 3 (50) | 0 | 1 (1) | A 3 |
| `f75e7053` | 9/5 | 23673 | 23673 | +0 | 162137 | 162137 | +0 | 0 (0) | 0 | 0 (0) | - |
| `fb1a1a72` | 9/5 | 43457 | 43456 | -1 | 325815 | 325815 | +0 | 1 (0) | 0 | 0 (0) | A 1 |
| **corpus** | | **397824** | **399135** | **+1311** | **4779571** | **4822178** | **+42607** | **203 (1211)** | **1514** | **1547 (60)** | |

Familles (§12.4) : A = tête de liste illisible pour le jeu ; Ap = NEW illisible lu plus loin dans la
liste ; C = liaison posée par un NEW du second rang. Gains au bit : paquets devenus fermés au bit près ;
« factices » = ceux qui contredisent une règle de l'écrivain, NON comptés sains.

**12 films en baisse** : `0797ce72` -1, `11de8353` -8 (utiles -55), `396cfc92` -1, `51ebbc0f` -2,
`60ae07c4` -4 (-5), `a349fea8` -1, `bf15f7ab` -1 (-6), `bfecd02b` -1 (-2), `c75f33b8` -5 (-24),
`d9781168` -6 (-10), `e5adf7b2` -3 (-50), `fb1a1a72` -1. **Toutes leurs pertes sont de la famille A**
(§12.4). Les pertes des familles Ap et C (9) sont toutes sur des films EN HAUSSE (`1c4c63c2` +679,
`4f77afc1` +501, `084a804d` +142). La mesure de recherche du §5.1 (`LT_MPP=8/3`) avait donné ces
chiffres à une unité près (397 823 / 399 134 ; l'écart est `50247b26`, non posé sous `-mpp-declare`).

Juge sur les GAGNÉS (1 514 ; sains, donc aucune règle contredite dans la tête) — ce qui les arrêtait
dans la base : « vue B : sortie par rejet » 1 226, « vue C : terminateur hors cadre » 198, « liste
d'événements non localisée » 51, « masque au-delà de l'archétype » 15, « ordre de la vue B » 1, divers
23. Juge sur les PERDUS (203) dans la tête : 144 non fermés (« liste d'événements non localisée » 141,
« vue B : sortie par rejet » 3) et 59 fermés au bit près mais contredits (« masque au-delà de
l'archétype » 36, « vue B : sortie par rejet » 21, « ordre de la vue B » 1, « masque épars non
croissant » 1). `InvariantEtatDeCreationIllisible` n'est la première règle d'AUCUN paquet perdu (7a).

### 12.4 Instruction des 203 pertes (paquet par paquet, par famille)

Méthode (sondes de recherche du lot sous `LR_MPP_DECLARE=1`, contrôlées §12.2) : `TestLRInstruire` sur
la base et la tête (rang du début, sortie, eid rejeté, origine de chaque liaison lue), `classer.awk`
(scratchpad), puis `TestLRTetes` sur la tête : relecture, AU MOMENT OÙ LA MARCHE Y PASSE, du record NEW
sur lequel la base ouvrait la liste, ou du NEW qui avait posé la liaison lue. Le verdict
`etat_illisible` ne vient que du lecteur du bloc MPP (garde-rail `TestLeBlocMPPDUnEtatPasseParSonVerdict`) :
vrai si et seulement si le compte R(3) vaut 5 ou plus sous le découpage déclaré (`ti=41` : sauf
l'exception du drapeau 2). Pièces : `cg3-lr2/{instr_base,instr_tete,tetes_tete,tetes_extra}/`,
`classes_relues.tsv`, `familles.tsv`.

| Famille | Paquets | Films | Établi |
|---|---|---|---|
| A : la base ouvrait la liste (rang 4, fermeture) sur un NEW que la tête relit illisible pour le jeu ; la tête prend la liste au second rang (58) ou ne la localise pas (136) | 194 | les 12 films en baisse (34), `084a804d` 23, `111fa685` 2, `1c4c63c2` 120, `4f77afc1` 15 | mesuré (relecture) : 194 / 194 `etat_illisible=true` ; mot de 32 bits du bloc inconnu du film pour 191 |
| Ap : la base ouvrait sur un NEW lisible, puis lisait un NEW illisible | 1 | `4f77afc1` 40:102 (NEW 6035 `ti=37` au bit 2955) | mesuré (relecture) |
| C : la base lisait un DELTA sous une liaison posée par un NEW du SECOND rang, que LR.2 ne pose plus | 8 | `084a804d` 23:946 (NEW `ti=41` de 23:8), `4f77afc1` 59:1008 (NEW `ti=41` de 59:1006), `1c4c63c2` 27:1140, 30:2018, 26:852, 36:892, 37:1424, 44:1616 | mesuré (origine `/r5/` de la liaison) ; les huit NEW d'origine sont LISIBLES pour le jeu (relus) : liaisons posées par une lecture non prouvée, pas des fermetures factices au sens du lecteur |
| Autre | 0 | | |

- **A et Ap (195) : fermetures qui passent par une lecture que le jeu ne fait pas** : le record NEW au
  compte de 5 ou plus n'est pas lu par `FUN_1408f1aa4` (§1). Sous le découpage que le film déclare, ce
  n'est plus un effet de 9/5 sur les formats anciens. Fermetures factices retirées, instruites (D2).
  Ce sont les SEULES pertes des 12 films en baisse.
- **C (8) : non factices**, toutes sur des films en hausse. La liaison venait d'un NEW lu depuis un
  début que rien ne prouve (le paquet ne fermait qu'au bit près en contredisant une règle) ; LR.2
  refuse précisément cette mutation du monde. À la base `87cdfa761` (9/5), cette famille frappait
  `e5adf7b2` (23 sains vrais, §4) ; sous le découpage déclaré, `e5adf7b2` n'en perd aucun.

**Verdict du gate 2 officiel** : aucun film en baisse hors des fermetures factices retirées et
instruites (famille A, D2) ; aucune perte inexpliquée.

### 12.5 Autres gates (code final)

| Gate | Sortie |
|---|---|
| `gofmt -l` (`internal`, `cmd`) | vide |
| `go vet ./...` | rc 0 |
| `go vet -tags=research ./internal/games/halo_infinite/film/...` | rc 0 |
| `go test ./internal/archlint/` | `ok` (37,6 s) |
| G-film (`film/...`, `replaybuild/...`, `sync/killcollector/...`, `-count=1 -timeout 30m`) | rc 0, 21 paquets `ok` |
| `TestGoldenFilms` de killsource (`KILLSOURCE_FIXTURES` = `film_chunks` du parc) | `ok` après régénération de `000d5950.golden` (une ligne de diagnostic `calibration`, §12.1) ; la base `8dfadd07e` le passe sans régénération |
| `golangci-lint run` complet, `grammar/...` et `facts/fallback/...` | `0 issues.` (après retrait de `param5`, §12.2) |
| `golangci-lint run --build-tags=research`, `grammar/...` | 28 issues, TOUTES présentes à la base (29 à la base : la 29e est l'`unparam` de `param5`, retiré) ; aucune neuve |
| Mutations (`-overlay`, `cg3-lr2/mut/mut.sh`, surcouches `sed`) | 13 / 14 ROUGES : les 10 du lot (§9) et M1, M2, M3 du contrôle ; M8 VERTE, équivalente en production (commentaire, §12.2) |
| `TestLRCasReelsDuSecondRang` (`LR_MPP_DECLARE=1`) | tête VERTE ; base `8dfadd07e` ROUGE sur ses trois cas (17:52, 17:172, 61:42) |
| killsource json, 19 témoins | 19 rc 0 ; 12 identiques à l'octet, 7 identiques hors du diagnostic `calibration` (`11de8353`, `396cfc92`, `50247b26`, `a349fea8`, `a521164d`, `c75f33b8`, `e5adf7b2`) : `killsource.Rev` constante (D23) |
| replay-equiv, 20 films, base contre tête | rc 1 des deux côtés (références non re-figées, verdict du harnais). Étapes divergentes : `artifact` 20, `movementStates.stats` 19, `continuousFire.stats` 19, `killsource` 14 (champ non exporté `Kill.paquet`, §8), `vehicleDeaths.stats` 10, `vehicleDeaths` 7, `movementStates` 7, `continuousFire` 6, `birthLoadouts.stats` 1 ; **identiques sur les 20** : `objectives`, `score`, `killRefs` et toutes les autres : `objectives.Rev` constante, `replay.SchemaVersion` reste 79 |
| replay-corpus-gate `--base=8dfadd07e` | **rc 1** : banc de vérité `MANQUE` sur 12 témoins, toujours et seulement par `P-1 paquets fermés au bit près`, exactement les 12 films en baisse du gate 2 avec les mêmes écarts (`0797ce72` 19076 → 19075 à la cuisson) ; `ok` sur 7 (`bcb6d393`, `084a804d`, `111fa685`, `a521164d`, `50247b26`, `4f77afc1`, `f75e7053`). Aucune mesure O-* ni V-* en `MANQUE` ou `FAUX`. 227 lignes `[FILET]` de couverture (tir continu, postures). Changements non bloquants : durées de posture par slot sur `084a804d` (8 slots) et `4f77afc1` (5 slots, dont 541 : 531 → 341 et 760 : 1015 → 766), `vehicles.aimRideFrames` de `084a804d` 23282 → 22825. Télémétrie : `grammarRev .2 → .3` (19) |

Le rc 1 du gate de corpus est la conséquence directe des 12 baisses du gate 2 (`P-1` et la carte
comptent le même objet), toutes de la famille A : admissible au même titre (D2), décision du pilote.

### 12.6 Compteur `repli_debut_de_liste_ferme_au_bit` par film (découpage déclaré)

Listes prises au second rang, marche de la carte (`TestLRSecondRang`, `LR_MPP_DECLARE=1`) ; la cuisson
du gate de corpus (`R-1`) donne les mêmes valeurs sur ses 19 témoins, à une unité près sur `60ae07c4`
(102 → 82 à la cuisson, 101 → 82 à la sonde).

| Film | Base | Tête | | Film | Base | Tête |
|---|---|---|---|---|---|---|
| `0797ce72` | 20 | 20 | | `60ae07c4` | 101 | 82 |
| `084a804d` | 517 | 421 | | `a349fea8` | 48 | 45 |
| `111fa685` | 169 | 144 | | `a521164d` | 3 | 3 |
| `11de8353` | 107 | 91 | | `bcb6d393` | 5 | 5 |
| `1c4c63c2` | 6628 | 6103 | | `bf15f7ab` | 10 | 7 |
| `396cfc92` | 9 | 9 | | `bfecd02b` | 22 | 16 |
| `4f77afc1` | 1013 | 904 | | `c75f33b8` | 93 | 76 |
| `50247b26` | 14 | 12 | | `d9781168` | 261 | 204 |
| `51ebbc0f` | 90 | 75 | | `e5adf7b2` | 141 | 122 |
| `f75e7053` | 8 | 7 | | `fb1a1a72` | 39 | 33 |
| **corpus** | **9 298** | **8 379** | | | | |

### 12.7 Écarts de la reprise

- **D-LR-7** : `vehicleDeaths` (étape neuve de 2.7.a) diverge sur 7 films et ses statistiques sur 10.
  Le canal des morts lit les records de la marche, que LR change (estimé : même mécanisme que
  `movementStates` et `continuousFire`) ; le nombre de morts de véhicule par film n'a pas été compté
  base contre tête.
- **D-LR-8** : la classification relit la tête de la BASE dans le monde de la TÊTE (au passage de la
  marche de la tête) ; une illisibilité qui dépendrait du monde ne se distinguerait pas. Le bloc MPP ne
  lit pas le monde : sans effet attendu (estimé).
- **D-LR-9** : carte, killsource json, `replay-equiv` et les sondes ont tourné sur des binaires
  construits avant le retrait de `param5` (argument constant `true` : aucune sortie ne change) ; le
  gate de corpus, G-film, les mutations, `TestGoldenFilms` et `TestLRCasReelsDuSecondRang` ont tourné
  sur le code final.
- **D-LR-6** reste vrai : `replay-equiv` et le gate de corpus cuisent les films du corpus (recette de
  la campagne) ; aucune cuisson du parc.
- D-LR-3 (annonce non consommée si le second rang tombe sur l'amorce et que la vue A n'est pas vide)
  reste non observé ; le cas « même payload, autre bit » est désormais testé (correction 4).

### 12.8 Verdict de la reprise

LR.1 et LR.2, portés sur `8dfadd07e` au rang `grammar-2026-10-06.3` : gate 2 officiel sous le découpage
déclaré **+1 311 sains, +42 607 utiles sains** ; les 12 films en baisse ne perdent que des fermetures
factices retirées (famille A, instruites paquet par paquet) ; 8 pertes non factices (famille C), toutes
sur des films en hausse. Proposé au pilote : **RETENU sous D2**, rc 1 du gate de corpus instruit au même
titre. Commit sur `feat/cg3-lr`, non poussé.
