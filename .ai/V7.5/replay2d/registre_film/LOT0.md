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
