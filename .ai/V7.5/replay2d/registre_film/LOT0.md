# LOT 0 — Hygiene et dettes — journal d'execution

Plan maitre : `.ai/V7.5/replay2d/PLAN_EXPLOITATION_REGISTRE_FILM.md` (section « Lot 0 »).
Worktree : `LevelUp-wt-registre-film`, branche `wt/registre-film`, depuis `3fdbb8030`.
Verdicts de gate : `LOT0_gates.log` (lignes `EXIT_*`).

Regles de machine appliquees (D13/D17) : `GOCACHE` = `<worktree>/.gocache`, `CGO_ENABLED=0`
pour filmdec/replay/objectiveevents, UNE seule commande `go` a la fois en avant-plan, un film
par processus, plafond memoire surveille (`PeakWorkingSet64`), jamais de decodage pendant un
`go build`.

## Etat initial (avant toute modification)

| Gate | EXIT |
|---|---|
| `go test ./internal/analysis/{filmdec,replay,objectiveevents}/` | 0 |
| `go test ./internal/games/halo_infinite/film/killsource/ -run TestGoldenMiniBobine` | 0 |

---

## 0.1 — Test de polarite d'i9 `[x]`

**Ce qui est fait.** Deux garde-rails, l'un inconditionnel (CI), l'autre chiffre sur le corpus.

1. `apps/go-api/internal/analysis/filmdec/components_batch7_test.go` (nouveau) —
   `TestConsumeObjectMultiplayerPropertiesPolarity` : 11 flux construits a la main, un par
   branche de `FUN_1407d4c94`, avec le nombre EXACT de bits attendu. Couvre la porte
   (`bit==1` -> 1 bit ; `bit==0` -> 1+5+TLV), les six formes de corps TLV (type 7 = 4 octets,
   types 2/3/0xe = 1 octet, type 8 = 8 octets, type 1 = aucun corps, type 4 = LEB128 sur un
   octet, type 0x10 = LEB128 sur deux octets / 130 octets), les deux extensions d'octet de
   type (0xe0 = +16 bits, 0xc0 = +8 bits) et le terminateur PORTEUR d'extension.
   `TestConsumeObjectMultiplayerPropertiesGateIsExclusive` enonce la RELATION plutot qu'un
   compte : sur le meme suffixe de flux, la branche « bloc absent » consomme strictement
   moins que la branche « bloc present ». Une inversion echange les deux et ne peut pas
   passer.
2. `apps/go-api/internal/analysis/filmdec/delta_walk_witness_test.go` (nouveau) —
   `TestDeltaWalkWitness`, sous garde `DELTA_WITNESS_FILM` (un film, chemin absolu) : marche
   delta reelle (`DecodeFrameRecords`) sur les 12 premiers chunks de replication, comptes
   FIGES par film. Le monde est amorce par les images-cles du chunk courant ; c'est un temoin
   de COMPARABILITE (deterministe et sensible), pas un oracle de justesse, et le fichier le
   dit. Cet instrument est UNIQUE et sert aussi le controle `DesyncAt` de l'item 0.6.

**Comptes figes (mesure du 2026-08-17, denominateurs explicites).**

| Film | Paquets delta | Records rendus | Traversee ABOUTIE (`DesyncAt == -1`) | `ported=false` |
|---|---|---|---|---|
| `000d5950` | 14 350 | 38 860 | 30 058 (77,349 %) | 8 802 |
| `06dfe6d9` | 6 606 | 10 607 | 8 494 (80,079 %) | 2 113 |
| `64e8adfa` | 14 357 | 39 776 | 31 934 (80,285 %) | 7 842 |

Cout par film : < 3 s, `PeakWorkingSet64` 14,8 / 15,9 / 15,6 Mo — deux ordres de grandeur
sous le plafond de 3 Go. Le premier reglage a 3 chunks a ete ABANDONNE : il ne rendait que
276 records sur `06dfe6d9`, trop peu pour qu'une derive s'y voie.

**Temoin de detection, JOUE (polarite inversee localement, puis remise).**

- Unitaire : les 11 cas echouent, plus l'exclusivite. `bit==1` consomme 822 bits au lieu de 1 ;
  toutes les branches `bit==0` tombent a 1 bit au lieu de 14 a 1 078.
- Corpus (`000d5950`) : records 38 860 -> 38 848, aboutis 30 058 -> 30 052. Le golden echoue.
- Restauration verifiee : `git diff` vide sur `components_batch7.go`.

**Gates.** `gofmt -l` vide · `EXIT_0.1_test_filmdec=0` · `EXIT_0.1_witness_3films=0` (les trois
films conformes au fige).

**Aucune modification de code de production dans cet item** : `components_batch7.go` est
identique a `3fdbb8030`.

---

## 0.2 — Chemin absolu d'i0 aux vraies largeurs `[!]` STOPPE

**Verdict : NON TENU, arret propre sur la clause d'echappement du plan** (« SI ce n'est pas
possible proprement — killsource ne connait pas la carte a cet endroit — STOP sur cet item,
ecris pourquoi »). Rien n'a ete modifie : `absoluteAxisW`, `SetAbsoluteAxisW` et
`simStateComplete` sont a l'identique de `3fdbb8030`.

L'item est INDIVISIBLE, et c'est la raison de l'arret complet. Ses trois clauses sont une
seule : `simStateComplete` ne peut basculer que si la queue d'i60
(`consumeSimStateHandleTail`, `traverse.go:1155-1162`) tire ses largeurs de la carte ; elle ne
le peut que si `absAxisWFor` -> `absAxisW` cesse de retomber sur `absoluteAxisW` ; et ce
retrait est impossible sans resoudre killsource, seul consommateur de production du levier.

### Le mecanisme de largeurs de killsource, decrit avant de toucher (verifie sur pieces)

`resetGlobals` (`killsource/decode.go:86-92`) ne pose pas un DEFAUT : il remet un levier a sa
valeur d'origine avant que la CALIBRATION ne s'en serve.

1. **`SetAbsoluteAxisW` est un levier de calibration vivant, pas une constante.**
   `killsource/calibrate.go:78-101` balaie 21 largeurs d'axe x 3 largeurs d'index = 63
   configurations par film, sur un echantillon FIGE de 400 paquets, et retient celle qui
   maximise le compte de records de bipede. La preuve que le balayage MORD est dans les
   goldens : `testdata/000d5950.golden:63` axisW=**13**, `78919882.golden:60` axisW=**15**,
   `9b191a7f.golden:62` axisW=**16**, `fccc61cd.golden:61` axisW=**16**. Aucun ne vaut 14, la
   valeur du reset. Supprimer `SetAbsoluteAxisW` supprime donc la detection de largeur PAR
   FILM de killsource, et la ligne de calibration est publiee dans `Result.Calibration` —
   gelee par cinq goldens, dont un INCONDITIONNEL.
2. **killsource n'a aucune carte a cet endroit, et pas seulement « pas encore ».**
   `Decode(ctx, name, src ChunkSource, opts *Options)` ne recoit ni nom de carte ni entree de
   catalogue — c'est ecrit et mesure dans `killsource/world_precision_test.go:5-12`. Ses trois
   appelants de production sont `cmd/killsource/main.go:206`,
   `internal/replaybuild/replaybuild.go:197` et `internal/sync/killcollector/collector.go:240`.
   Le dernier passe `ChunkSourceOf(chunks)` — des chunks EN MEMOIRE, sans repertoire : ni
   `MapQuantEntry` (qui exige le nom de carte) ni `DetectI0Layout(dir)` (qui exige un chemin)
   n'y sont disponibles. Installer les largeurs de la carte demande de faire descendre une
   entree de catalogue a travers le pipeline de sync — hors du perimetre du lot 0 par nature,
   pas par prudence.
3. **Les deux leviers n'ont pas la meme FORME.** `absoluteAxisW` est UNE largeur uniforme ;
   `WorldObjectPrecision.AxisW` en porte TROIS, par axe, venues du catalogue de bornes. Un
   balayage ne produit qu'un uniforme. Les fusionner fait perdre a killsource sa detection,
   OU fait ecrire par killsource un global qu'il ne fait aujourd'hui que LIRE — sans
   restauration en sortie (`Decode` ne remet a zero qu'en ENTREE), donc avec un chemin de
   contamination vers `replay.BuildFromFilm` dans le meme process.

### Ce qui a ete ecarte, et pourquoi

Faire lire a la SEULE queue d'i60 `WorldObjectPrecision.AxisW` en direct (sans toucher a
`absAxisWFor`) donnerait bien les largeurs de la carte cote rejeu. Ecarte : cela cree une
TROISIEME source de largeur pour le meme lecteur absolu, alors que la divergence entre les deux
existantes est deja la faute documentee du paquet (« c'est leur double implementation qui les
avait laissees diverger », `position_capture.go:215-219`). De plus le temoin de detection ecrit
au registre (`0 -> 2 sources proposees`) a ete mesure par R7-b avec la queue a 14 bits par axe ;
sous 13/13/14 le compte de bits differe et le gate « toute autre variation = STOP » ne serait
plus interpretable.

### Condition de reprise (ecrite, mesurable)

Faire descendre une source de largeurs jusqu'a `killsource.Decode` — `Options.MapQuant
*filmdec.MapQuantEntry`, alimente par les trois appelants (le collecteur de sync a le
`matchID`, donc la carte) — puis, dans le meme lot : retirer le balayage d'axe de `calibrate`
au profit de ces largeurs, re-mesurer les cinq goldens killsource, et alors seulement
supprimer `absoluteAxisW` et basculer `simStateComplete`. C'est un lot a part entiere (sync +
killsource + 5 goldens), pas un item d'hygiene.

### `DesyncAt` avant / apres 0.2

Identique par construction : aucune ligne de production n'a bouge. Les comptes du temoin sont
ceux de l'item 0.1 (000d5950 38 860 / 30 058 · 06dfe6d9 10 607 / 8 494 · 64e8adfa 39 776 /
31 934), re-verifies verts apres 0.3.

---

## 0.3 — Empreinte du registre `[x]`

**Ce qui est fait.**

- `filmdec/registry_fingerprint.go` (nouveau) : `RegistryFingerprint(reg) uint64`,
  `KnownRegistryFingerprint`, et l'alerte `slog.WarnContext` dedupliquee. L'empreinte est un
  FNV-1a 64 sur `kind | flags | nom` des slots NON VIDES, dans l'ordre du chunk.
- `filmdec/registry.go` : `Registry.fingerprint` (prive) calcule PENDANT l'unique passe de
  lecture — la seule qui voie `kind`, que le parse ne retient pas (le garder dans `Archetype`
  serait un champ sans lecteur). `parseRegistry` appelle `warnUnknownRegistry`.
- Deduplication sur l'EMPREINTE (`sync.Map`) : « une alerte par grammaire inconnue rencontree »,
  et non par parse — sans quoi trois lignes identiques par film et des milliers sur une
  re-cuisson de corpus.
- `filmdec/registry_fingerprint_test.go` (nouveau) : `TestRegistryFingerprintDomain`
  (l'empreinte bouge sur `kind`, sur `flags`, sur le nom et sur l'ORDRE ; elle NE bouge PAS
  quand seul le bourrage change — sans ce dernier cas, une empreinte qui hacherait tout le bloc
  passerait le test en alertant a chaque film), `TestRegistryFingerprintWarnsOnce`,
  `TestRegistryFingerprintOnFilm` (sous garde, publie la mesure).
- Le decalage `Flags[k]` / `Flags[k+1]` (registre `:228`) N'EST PAS touche.

**L'empreinte, RECALCULEE et non recopiee.**

| Film | Blocs | Slots non vides | Archetypes porteurs | Empreinte |
|---|---|---|---|---|
| `000d5950` | 118 | 1 067 | 49 | `0x61e492dd4de7fd4e` |
| `64e8adfa` | 118 | 1 067 | 49 | `0x61e492dd4de7fd4e` |
| `06dfe6d9` | **116** | **1 031** | **48** | **`0x5827362c37d2adb3`** |

`KnownRegistryFingerprint = 0x61e492dd4de7fd4e` — le registre de REFERENCE, celui que decrit
`testdata/ecs_table.tsv` (118 blocs / 1 067 slots). La valeur `0xa413610cd08e4355` du
commentaire de `registry.go` n'a PAS ete reprise : c'est un FNV « noms + flags » (deux champs),
pas `kind | flags | nom` (trois). Deux domaines, deux valeurs.

**DECOUVERTE MAJEURE — le registre n'est PAS identique sur tous les films.** `06dfe6d9` porte
2 archetypes et 36 composants de MOINS que `000d5950` / `64e8adfa`. La stabilite mesuree au lot
table ECS (« bit-a-bit identique sur `000d5950`, `00502e52`, `07aa428d` ») vaut DANS UN BUILD,
pas ENTRE builds. L'alerte se declenche a bon droit sur `06dfe6d9` et le film reste decode.
Consequence a verifier par le lot qui touchera la table : le garde-rail **G2** (film <-> table,
niveaux compris) ne peut pas etre vert sur `06dfe6d9`.

**Cout du re-parse, chiffre (ce que le plan demande).**

- Taille : `chunk_00` fait 435 425 / 525 722 / 510 435 octets compresses pour **1 973 120 /
  1 944 960 / 1 973 120 octets INFLATES** (facteur 3,7 a 4,5).
- Parses par film sur le chemin de PRODUCTION `replaybuild.Build` : **TROIS**, pas quatre.
  `replay.BuildFromFilm` -> `ScanFilmAbilityRanks` (`ability_rank.go:158`) ;
  `replay.BuildFromFilm` -> `ScanFilmEquipmentPlacements` -> `CalibrateMPPWidths` ->
  `EquipmentArchetype` (`equipment_state.go:174`) ; `killsource.Decode` -> `newTimeline`
  (`killsource/world.go:58`). Le quatrieme site de production, `ground_weapon_creation.go:98`,
  n'a AUCUN appelant de production (seulement des tests) — le compte « >= 4 chemins » de la
  section Decouvertes du plan surestime donc d'un.
- Temps : premier parse 14,9 / 16,0 / 15,0 ms (a froid, allocation comprise) ; parses suivants
  145 a 322 us. Les trois parses de production coutent donc ~15-16 ms a froid et moins d'une
  milliseconde a chaud, contre plusieurs SECONDES pour la marche delta du meme film
  (2,2 a 6,1 s sur 12 chunks seulement, item 0.1). **Bien sous les 5 % du temps de scan : pas
  d'optimisation, conformement au plan.**

**Neutralite au bit prouvee** : les comptes du temoin de marche delta sont IDENTIQUES apres
0.3 sur les trois films.

**Gates.** `gofmt -l` vide · `go vet` 0 · `EXIT_0.3_test_filmdec_replay=0`.

---

## 0.4 — Table ECS : ti=11 i2 et i9 `[x]`

**Ce qui est fait.** Les deux lignes passent `non_porte` -> `deser_non_cable`, avec
`code_source = components_batch3.go:22` et la grammaire renseignee dans la colonne `grammar`.
La coherence avec ti=23 est faite : meme statut pour la meme situation (« grammaire ecrite,
aucun appelant »).

`deser_addr` reste VIDE, et c'est un fait ecrit en `notes` plutot qu'une case remplie au
hasard : `consumeObjectiveFormattedText` a ete porte par le workflow `port-ecs-deathchains`
sans jamais etre recoupe a une adresse `FUN_` — contrairement aux lignes ti=23, qui portent
`FUN_142ed6cec`. Inventer une adresse aurait ete pire que l'absence.

**Une modification de commentaire etait NECESSAIRE, pas cosmetique.** `checkCodeSource`
(`ecs_table_guard_test.go:271`) exige que le NOM du composant apparaisse dans le fichier
designe. Le commentaire de `components_batch3.go` ecrivait le jumeau i9 en abrege
(« son jumeau `-secondary-` »), donc le nom complet
`managed-objective-secondary-formatted-text-component` n'y figurait pas. Les deux noms sont
maintenant ecrits en entier, avec la raison.

**Temoin de detection du couplage, JOUE.** En remettant l'abreviation :
`G1 : ligne 277 : managed-objective-secondary-formatted-text-component est declare
deser_non_cable mais son nom n apparait pas dans components_batch3.go` — EXIT 1. Restaure :
EXIT 0. Le garde-rail mord bien sur ces deux lignes precises.

**Comptes de statuts apres edition** (denominateur inchange, 1 081 lignes dont 14 alias) :
`porte` 458 · `non_porte` **530** (etait 532) · `partiel` 45 · `deser_non_cable` **34**
(etait 32) · `alias` 14. La ligne `:230` du registre porte ces chiffres et est mise a jour a
l'item 0.5.

**Gates.** `EXIT_0.4_G1_G3=0` (G1 et G3 verts ; G2 saute, sans film) ·
`EXIT_0.4_test_filmdec=0` · `gofmt -l` vide.

---

## 0.5 — Registre des reports `[x]`

**Huit lignes ajoutees** (`.ai/V7.5/REGISTRE_REPORTS.md:233-240`), origine commune « artefact
*Registre du film Theater* + `replay2d/PLAN_EXPLOITATION_REGISTRE_FILM.md`, 2026-08-17 », chacune
avec sa condition de reprise nommant le lot qui la porte :

| Sujet | Etat ecrit | Lot de reprise |
|---|---|---|
| Score dans le temps (statborg ti=6) | decodeur COMPLET et mesure (283/284 vs Cheat Engine), **0 appelant** ; `objectivescore` supprime, pas parke | **A** |
| Elevation de visee | DEJA lue et DEJA stockee (`PitchRaw`), bloquee sur toute la chaine aval (accesseur, golden, JSON, cone plan) | **E** |
| ti=23 `selectable-zone-data` | NEGATIF mesure : 0 record sur 1 205 704 et 0 sur 988 752 ; les porteurs vivants sont ti=10 et ti=12 | **C** (recensement seul) |
| tacmap ti=34 / ti=30 | NEGATIF mesure : 4 records sur 988 752, et 0 pour ti=30 — c'est de la campagne | **C** (ti=12) + **F4** (ecrire le negatif) |
| ti=5 i11 `player-engine-loadout` | 8 x R(8) jetes, alors que `loadouts[]` vient d'un balayage d'images-cles espace de 20 s — le meme plafond que le ramassage | **P** |
| ti=10 `managed-object` | i0 porte en `Skip(32)` : les 32 drapeaux sont consommes sans etre lus ; i1 `boundary-color` (le CAMP) non porte | **C** |
| Medailles (`medal_type`) | AUCUNE ACTION — enonce utilisateur ; la ligne existe pour qu'aucun lot ne rouvre le sujet | **aucun** |
| Empreinte du registre `chunk_00` | **TRAITEE** par 0.3, avec sa decouverte (registre non identique entre builds) et son cout chiffre | **0** (fait) |

**Trois lignes existantes mises a jour.**

- `:220` (R7-b, polarite d'i9) — la dette « aucun test ne fige la polarite » est declaree
  TRAITEE, avec les deux instruments, le temoin joue et les chiffres du deplacement.
- `:221` (REPORT `simStateComplete`) — **PAS declaree traitee**, et re-instruite : la condition
  de reprise ecrite par R7-b (« faire tirer au chemin absolu d'i0 les largeurs de la carte »)
  etait sous-estimee d'un ordre de grandeur. Les trois obstacles de l'item 0.2 y sont ecrits
  avec leurs pieces (les quatre valeurs `axisW` retenues par le balayage, l'appelant de sync
  qui passe des chunks en memoire, la difference de forme des deux leviers), le perimetre du
  lot futur est enumere en trois etapes, et une reserve est posee sur le temoin lui-meme : il
  a ete mesure a 14 bits par axe et devra etre re-etabli sous des largeurs de carte avant de
  servir de gate.
- `:230` (Table ECS) — comptes de statuts corriges (`non_porte` 532 -> **530**,
  `deser_non_cable` 32 -> **34**) et amendement en deux points : le passage de ti=11 i2/i9 avec
  la raison du `deser_addr` vide, et l'avertissement que **G2 ne peut pas etre vert sur tous les
  films** — jouer G2 exige un film d'empreinte `0x61e492dd4de7fd4e`.

**Verification de forme.** Les 8 lignes ajoutees et les 3 amendees ont toutes exactement 6
champs de tableau (un `kind|flags|nom` litteral, qui cassait la ligne 240 en 8 champs, a ete
reecrit `kind + flags + nom`). Trois lignes ANTERIEURES du registre (`:177`, `:182`, `:201`)
portent le meme defaut ; hors perimetre, non corrigees, notees en Decouvertes.

---

## 0.6 — Plomberie de publication (D15) `[x]`

**Perimetre tenu : 23 composants, quatre hooks, `traverse.go` MAIGRIT de 1 321 a 1 297 lignes.**

| Famille | Composants deplaces | Hook | Fichier |
|---|---|---|---|
| ti=0 moteur de jeu | i2, i4, i6, i7, i8 (5) | `SetGameEngineHook` | `components_game_engine.go` (nouveau) |
| ti=5 entite joueur | i2, i3, i6, i11, i12, i14, i15, i17, i18, i19, i20 (11) | `SetPlayerStateHook` | `components_player.go` (nouveau) |
| ti=37 equipement | i26 `energy-delay-ticks-left`, i27 `charges-remaining` (2) | `SetEquipmentStateHook` (existant, +2 champs) | `equipment_state.go` |
| ti=10 objet scripte | i0 `boundary-visibility` (1) | `SetManagedObjectHook` | `components_walk_batch9.go` |
| sondes | ti=47 i0 et i1, ti=4 i0, ti=13 i0 (4) | `SetProbeHook` | `components_probe.go` (nouveau) |

**Contrat de publication, un seul et ecrit une fois** (`components_game_engine.go:78-87`) :
`values` porte les champs lus DANS L'ORDRE DU FLUX, bits de porte compris, sans aucune
dequantification ; `present` est faux quand la porte de TETE s'est fermee sans qu'aucun champ
ne suive — une porte fermee n'est pas une valeur nulle.

**Trois choix de conception, avec leur raison.**

1. **Enumerations nommees plutot que `comp int`.** Le plan ecrit `comp int` ; le MODELE qu'il
   designe (`equipment_state.go`) resout l'index par NOM et ne le cable jamais, parce qu'un
   index de composant est un numero de BUILD — et l'empreinte livree a l'item 0.3 vient de le
   prouver sur pieces (`06dfe6d9` : 116 blocs contre 118). D'ou `GameEngineField`,
   `PlayerStateField`, `ManagedObjectField`, `ProbeComponent`, chacune avec un `String()` qui
   rend l'etiquette de registre. Ce sont des types a base `int` : la signature du plan est
   respectee, sa lettre est amelioree.
2. **`SetProbeHook` recoit le `ti` DE LA TRAVERSEE**, jamais une constante
   (`publishProbe(typeIndex, ...)`). `TestProbeHookPassesRegistryTypeIndex` joue les memes
   composants sous quatre `ti` differents et exige que le hook les rende tels quels.
3. **ti=10 i0 : le `Skip(32)` devient une boucle bit a bit, pas un `ReadBits(32)`.** Le jeu pose
   le bit de l'iteration i au RANG i ; `ReadBits(32)` mettrait l'iteration 0 au rang 31. La
   CONSOMMATION est identique (32 bits), la VALEUR ne l'est pas —
   `TestManagedObjectHookFlagOrder` distingue les deux en n'allumant que les rangs 0 et 31.

**Ce qui N'A PAS ete duplique, et le garde-rail qui l'empeche.** ti=0 i5 (round-timer) et ti=5
i1 (respawn) sont deja rendus TYPES par la couche de capture (`capture.go:20-25`). Leur poser un
hook ferait une troisieme copie de la meme grammaire. `TestHookedNamesCoversMovedCases` croise
`hookedNames` avec `captureNames` et echoue si un nom apparait dans les deux.

**Garde-rails ajoutes** (`components_hooks_test.go`, nouveau) :

- `TestHooksConsumeSameBitsWithoutHook` — 23 composants x 500 tampons aleatoires x 12 niveaux :
  position du lecteur et drapeau `ported` IDENTIQUES avec et sans hook, et le hook doit avoir
  ete appele au moins une fois. Meme esprit que `TestCaptureConsumesSameBitsAsDispatch`.
- `TestHookedNamesCoversMovedCases` — la liste fait bien 23 noms, sans doublon, tous traites par
  le dispatch reel, et disjointe de `captureNames`.
- `TestGameEngineHookValues` (6 cas), `TestPlayerStateHookValues` (9 cas),
  `TestPlayerDesiredRespawnLocationHook` (4 cas : les TROIS issues des deux portes imbriquees
  plus le cas avec index lu), `TestPlayerMalleablePropertiesHook` (les 24 entrees, bits de porte
  compris), `TestEquipmentHookNewFields`, `TestManagedObjectHookFlagOrder`,
  `TestProbeHookPassesRegistryTypeIndex`, `TestProbeSplashStaticPublishesUnconditionalField`,
  `TestHookFieldStringsAreRegistryLabels`.

**Deux effets de bord assumes, mesures.**

- `consumeQuantVec3` delegue desormais a `consumeQuantVec3Values` (une seule copie de la
  grammaire) — `ok` faux quand `precHigh` est leve, ce qui evite de publier une position a
  l'origine pour un vecteur par defaut.
- `EquipmentFieldCount` passe de 4 a 6 : la marche de `ScanFilmEquipmentState` va maintenant
  jusqu'a i27 au lieu de i24. **MESURE sur `000d5950`** : i26 820 annonces / 820 lectures / 629
  vies d'objet sur 1 216, i27 883 / 883 / 599 — contre 131 / 62 / 34 / 50 annonces pour les
  quatre champs d'origine. Les deux nouveaux canaux sont bien les plus bavards de l'archetype,
  et ils TRANSITIONNENT (126 transitions sur 172 paires consecutives pour i26, 194 sur 263 pour
  i27) : c'est la these du lot D, verifiee avant lui. `equipment_state_test.go` porte desormais
  `equipAssertNewFieldsSeen` — une assertion, pas une ligne de journal.

**`DesyncAt` avant / apres 0.6 — les hooks ne changent pas un bit.**

| Film | Paquets | Records | Aboutis AVANT | Aboutis APRES | Ecart |
|---|---|---|---|---|---|
| `000d5950` | 14 350 | 38 860 | 30 058 | 30 058 | 0 (0,000 %) |
| `06dfe6d9` | 6 606 | 10 607 | 8 494 | 8 494 | 0 (0,000 %) |
| `64e8adfa` | 14 357 | 39 776 | 31 934 | 31 934 | 0 (0,000 %) |

Les trois films rendent « CONFORME au compte fige ». `PeakWorkingSet64` maximal 52,5 Mo.

**Table ECS** : les 23 lignes gardent leur statut `porte` ; seule la colonne `code_source`
bouge, vers le fichier et la ligne du nouveau deser (les quatre sondes pointent la ligne de
`publishProbe` dans `traverse.go`, puisque c'est le `case` lui-meme qui publie). G1 vert.

**Gates.** `gofmt -l` vide · `go vet` 0 · `EXIT_0.6_test_filmdec_replay_objectiveevents=0` ·
`EXIT_0.6_desync_3films=0` · instrument d'equipement sous garde `EQUIP_FILM` : EXIT 0 sur
`000d5950`.

---

## 0.7 — Gates de cloture `[x]` (avec un `[~]` nomme)

**Commandes exactes du plan (§5), jouees dans cette session, verdicts dans `LOT0_gates.log`.**

| Gate | Commande | EXIT |
|---|---|---|
| Build CGO | `go build ./...` (gcc winlibs) | **0** |
| Vet | `go vet ./internal/analysis/... ./internal/replaybuild/... ./internal/games/halo_infinite/film/... ./contracttest/...` | **0** |
| Tests sans CGO | `CGO_ENABLED=0 go test -count=1 ./internal/analysis/{filmdec,replay,objectiveevents}/` | **0** |
| Tests avec CGO | `go test -count=1 ./internal/replaybuild/ ./internal/games/halo_infinite/film/... ./contracttest/` | **0** |
| Lint du diff | `golangci-lint run --new-from-merge-base=origin/main ./...` | **0** (« 0 issues ») |
| `gofmt -l` | sur `internal/analysis/filmdec/` | vide |

**UNE issue de lint a ete trouvee et corrigee** avant la relance : `goconst` sur
`managed-object-boundary-visibility-component`, litteral en 4 endroits apres l'item 0.6. Une
constante `compManagedObjectBoundaryVisibility` a ete posee dans `components_walk_batch9.go` et
les quatre sites y renvoient — exactement la regle « a la 3e copie, centraliser ». La relance
rend 0.

**Temoin `TestGoldenMiniBobine` : INCHANGE, et c'est la conclusion attendue.** L'item 0.2 etant
stoppe, `simStateComplete` reste `false` et le temoin doit rester a `0 propose(s), 0 publiee(s)`
sur la voie MARCHE. Verifie sur pieces : `minibobine.golden:56` porte toujours
`marche, source appartenant a la victime : 0 propose(s), 0 publiee(s)`, et
`git diff HEAD~6 -- .../killsource/` est VIDE — aucun fichier du paquet killsource n'a bouge de
tout le lot.

**`[~]` — entree `.ai/thought_log.md`** : couverte par le superviseur, sur consigne explicite
(« N'edite PAS le journal §7 du plan maitre ni `.ai/thought_log.md` »). Les cases du Lot 0 du
plan maitre, elles, sont cochees par ce lot.

---

## Recapitulatif des statuts du Lot 0

| Item | Statut | En un mot |
|---|---|---|
| 0.1 Test de polarite d'i9 | `[x]` | 2 instruments, temoin joue, comptes figes sur 3 films |
| 0.2 Chemin absolu d'i0 aux vraies largeurs | `[!]` | STOP motive : killsource n'a pas de carte et `SetAbsoluteAxisW` est un levier de calibration vivant |
| 0.3 Empreinte du registre | `[x]` | `0x61e492dd4de7fd4e` ; et le registre N'EST PAS identique entre builds |
| 0.4 Table ECS ti=11 i2/i9 | `[x]` | `deser_non_cable`, G1 vert, couplage prouve rouge |
| 0.5 Registre des reports | `[x]` | 8 lignes ajoutees, 3 amendees (`:220`, `:221`, `:230`) |
| 0.6 Plomberie de publication | `[x]` | 23 `case` deplaces, 4 hooks, `DesyncAt` identique au record pres |
| 0.7 Gates de cloture | `[x]` | tous a 0 ; `thought_log` `[~]` (superviseur) |

**Gate 0 du plan** — « tout vert, `TestGoldenMiniBobine` au temoin ecrit, aucune ligne de table
sans statut coherent, chaque hook couvert par un test unitaire, `DesyncAt` identique avant/apres
0.6 » : tenu, A UNE RESERVE — le temoin ecrit du plan (`0 -> 2 sources proposees`) supposait
l'item 0.2 fait. Il ne l'est pas, donc le temoin correct est « inchange », et c'est ce qui est
mesure.

---

## Decouvertes hors perimetre (notees, NON traitees)

1. **Le registre du film n'est pas une constante de FORMAT, seulement de BUILD.** `06dfe6d9` :
   116 blocs / 1 031 slots contre 118 / 1 067. Consequence directe : le garde-rail **G2** de la
   table ECS ne peut pas etre vert sur ce film. Ecrit au registre (`:230`).
2. **Le compte « >= 4 chemins de production re-parsent le registre » de la section Decouvertes du
   plan surestime d'un.** `ground_weapon_creation.go:98` n'a AUCUN appelant de production. Le
   chemin `replaybuild.Build` en fait TROIS.
3. **Trois lignes du registre des reports sont mal formees** (`:177` 8 champs, `:182` 5, `:201`
   7, au lieu de 6) : des `|` litteraux dans le texte cassent leur rendu en tableau. Anterieures
   a ce lot, non corrigees.
4. **`equipment_state_test.go:114-116` porte une decouverte perimee** : « aucun chemin de
   production n'appelle `SetWorldObjectPrecisionFromLayout` ». C'est faux depuis le 2026-08-15 —
   `replay.installWorldObjectPrecision` le fait pour tout `BuildFromFilm`. Commentaire a
   corriger par le lot qui touchera ce fichier (lot D).
5. **`ScanFilmEquipmentState` reste sans consommateur de production** (deja au registre du plan,
   decouverte n°2). L'item 0.6 l'a ENRICHI de deux champs sans lui donner d'appelant : si le lot
   D se clot `[!]`, la question « instrument sous garde ou code mort » se posera sur six champs
   au lieu de quatre.
6. **`absPerIndexAxisW` / `SetAbsPerIndexAxisW` / `SetAbsDequantMode` / `AbsIndexHistogram`
   (`position_capture.go`)** : quatre leviers de calibration exportes dont aucun appelant de
   production n'a ete trouve pendant l'instruction de l'item 0.2. Hors perimetre du lot 0, mais
   ils appartiennent au meme nettoyage que `absoluteAxisW` — a traiter par le lot qui reprendra
   `:221`.
