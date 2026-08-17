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
