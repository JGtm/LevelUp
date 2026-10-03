# PREPARATION M2 — pas 4, 5 et 6 (lots 2.4, 2.5, 2.6)

> Note d'ANALYSE SUR PIECES, sans production. Aucun fichier Go n'a ete modifie, aucun decodage
> n'a ete joue. Objet : donner aux executeurs de 2.4, 2.5 et 2.6 les chiffres, les listes et les
> esquisses qui leur eviteraient une phase de mesure.
>
> Branche `feat/decfilm-m2prep`, base `1e246b209`. Lots 2.1, 2.2, 2.7, 2.8, 2.9 fusionnes ;
> **lot 2.3 EN COURS ailleurs** (suppression des globales, du verrou, calibration portee dans le
> profil) : traite ici comme un ACQUIS A VENIR, jamais comme un travail a refaire.
>
> Instruments : `grep -rn`, `go list -f`, `ls`, `wc -l`, `diff`, `go build ./...`.
> `go build ./...` passe sur la base (1 min 30 s, `GOCACHE` prive) : tous les comptes ci-dessous
> sont pris sur un arbre qui compile.
>
> Date : 2026-09-17.

---

## 0. L'etat mesure a l'entree

### 0.1 Le poids des paquets concernes

`ls`, puis `wc -l` sur les seuls fichiers non-test :

| Paquet | fichiers prod | fichiers `_test.go` | lignes prod |
|---|---:|---:|---:|
| `internal/games/halo_infinite/film/filmdec` | 142 | 409 | 31 011 |
| `internal/games/halo_infinite/film/replay` | 164 | 378 | 38 947 |
| `internal/games/halo_infinite/film/killsource` | 23 | 19 | 5 787 |
| `internal/analysis/objectiveevents` | 20 | 42 | 5 081 |
| `internal/analysis/filmsource` | 3 | 2 | 499 |
| `internal/games/halo_infinite/film/filmcache` | 2 | 2 | 329 |

Le pas 5 deplace `filmdec` + `killsource` + `objectiveevents` + `filmsource` : **188 fichiers de
production et 472 fichiers de test, soit 660 fichiers**. C'est l'ordre de grandeur du `git mv`.

### 0.2 Les revisions en vigueur

```
filmdec/grammar_rev.go:413                   const GrammarRev = "grammar-2026-09-15.26"
killcollector/killsource_decoder_rev.go:128  const KillSourceDecoderRev = "killsource-2026-09-16.2"
film/replay/document.go:48                   const SchemaVersion = 60
archlint/filmdec_package_vars_test.go:168    const filmdecVarsGeles = 43
```

**Le plan est perime sur un point de chiffre** : l'item 2.6.3 ecrit « montee de `SchemaVersion`
57 ». La constante vaut **60** sur la base `1e246b209` (derniere entree de chronique :
`document_chronicle.go:1506`, `// v60 (2026-09-17, vague 2 de la famille 1.9 ...)`). La montee
de 2.6.3 est donc **60 -> 61**, et le plan doit etre corrige dans le commit qui la porte.

---

## 1. Lot 2.4 (pas 4) — Une seule porte aux octets

### 1.1 (a) Inventaire EXHAUSTIF des lectures d'octets bruts de film

Motifs passes : `chunk[`, `chunks[`, `.Chunk(`, `evReader`, `bitReader` / `NewBitReader`,
`binary.LittleEndian`, `[]byte` de chunk. Racines : `internal/games/halo_infinite/film/`,
`internal/analysis/{filmsource,objectiveevents,weaponv3}/`, `internal/sync/`,
`internal/replaybuild/`, `cmd/`.

**Correction de perimetre AVANT tout** : le brief et l'item 2.4.2 citent
`internal/games/halo_infinite/weaponv3/`. Ce repertoire **n'existe pas**. Le paquet est
`internal/analysis/weaponv3/` (4 fichiers de production : `bits_word.go`, `canon.go`,
`pi_resolver.go`, `timing.go`). Consequence directe, traitee en §1.3 et en questions ouvertes :
il vit sous `analysis/`, ou le ratchet D9 interdit d'importer un paquet de titre.

#### Tableau des sites de PRODUCTION

Colonnes : fichier | ligne | ce qui est lu | via quel type.

**A. La source legitime (deviendra `film/internal/source`)**

| Fichier | Ligne | Ce qui est lu | Via quel type |
|---|---:|---|---|
| `analysis/filmsource/film.go` | 81 | `f.chunks[i]` — les octets decompresses d'un chunk | `*filmsource.Film` (accesseur `Chunk`) |
| `analysis/filmsource/film.go` | 132-137 | `src.Chunk(ch)` puis `inflate` puis `f.chunks[ch] = d` | `filmsource.Source` -> `*Film` |
| `analysis/filmsource/film.go` | 249-262 | en-tete de paquet 16 o LE : `Uint16(d[off:])`, `Uint32(d[off+4:])`, `Uint64(d[off+8:])` | `appendPackets`, `filmsource.Packet` |
| `analysis/filmsource/source.go` | (tout) | `DirSource`, `MemoryChunks`, numerotation `chunk_NN.bin` | `filmsource.Source` |

**B. Les fournisseurs d'octets bruts (entree disque / reseau)**

| Fichier | Ligne | Ce qui est lu | Via quel type |
|---|---:|---|---|
| `film/filmcache/filmcache.go` | 113-121 | `os.ReadFile(chunk_NN.bin)` — octets COMPRESSES | `filmcache.Source` (implemente `filmsource.Source`) |
| `film/filmcache/filmcache.go` | 139 | `filmsource.LoadDir(ChunkDir(...), src.Meta())` | `*filmsource.Film` |
| `sync/haloclient/halo_client_film.go` | 101 | `filmsource.Inflate(registre)` puis `FilmMajorVersionFromHeader` | `[]byte` nu |
| `sync/killcollector/bridge.go` | 131 | `filmsource.Load(filmsource.MemoryChunks(seq), meta)` | `filmsource.MemoryChunks` |

**C. `filmdec` — lectures d'octets hors lecteur de bits**

| Fichier | Ligne | Ce qui est lu | Via quel type |
|---|---:|---|---|
| `filmdec/film_packets.go` | 37 | `chunk[p.Start : p.Start+p.Size]` (`FilmPacket.Payload`) | `[]byte` nu |
| `filmdec/film_packets.go` | 69, 75, 78 | **SECOND marcheur de paquets** : `Uint32(chunk[off+4:])`, `Uint16(chunk[off:])`, `Uint64(chunk[off+8:])` | `filmdec.FilmPacket` |
| `filmdec/film_packets.go` | 54 | `filmsource.Inflate(raw)` | `[]byte` nu |
| `filmdec/film_chunks.go` | 112, 128 | `f.Chunk(pos)`, `filmPacketsOf(f.Packets(pos))` — le pont film charge / balayages | `*filmsource.Film` |
| `filmdec/film_major_version.go` | (1 `LittleEndian`) | u32 LE en tete de `chunk_00` | `[]byte` nu |
| `filmdec/film_format_version.go` | (1 `LittleEndian`) | u32 LE a `chunk_00+4` | `[]byte` nu |
| `filmdec/film_identity.go` | (4 `LittleEndian`) + 235 | section 2 de `chunk_00`, puis `NewBitReader(d)` | `filmdec.FilmIdentity` |
| `filmdec/registry.go` | (1 `LittleEndian`) | `ParseRegistryChunk(chunk_00)` | `filmdec.Registry` |
| `filmdec/offline_biped_band.go` | 163 | FAUX POSITIF : `chunks` y est un `[]int` d'indices | — |

**D. `filmdec` — le lecteur de bits canonique et ses points de construction**

`NewBitReader(...)` est appele **45 fois en production dans `filmdec`** (grep `NewBitReader` hors
`_test.go`), toujours sur un `payload` de paquet ou un `chunk_00`, plus 1 fois dans
`killsource/walk.go:61` et 1 fois dans `cmd/rdata_weapon_scan/main.go:257`.
Fichiers concernes de `filmdec` (ordre `ls`) : `ability_rank.go`, `biped_creation.go`,
`biped_pickups.go`, `bitreader.go`, `default_state.go` (x2), `equipment_creation.go`,
`equipment_state.go`, `event_list.go`, `film_identity.go`, `fire_aim_modal.go`,
`frame_chain_infer.go` (x4), `frame_harvest.go` (x5), `frame_infer.go` (x2), `frame_records.go`,
`ground_weapon_ammo.go` (x2), `keyframe_entity_queue.go`, `keyframe_fullstate_loop.go`,
`keyframe_record_walk.go`, `navpoint_radial_scan.go`, `objective_scan.go`,
`object_deaths_march.go`, `offline_aim_only.go`, `offline_biped.go`, `player_table.go`,
`player_table_record.go` (x2), `player_teams.go`, `probe_export.go` (x2), `transloc_events.go`,
`weapon_hits.go` (x3), `zone_state_scan.go`, `zoom_events.go`.

**E. `killsource` — deuxieme lecteur, et lectures par position de bit**

| Fichier | Ligne | Ce qui est lu | Via quel type |
|---|---:|---|---|
| `killsource/chunks.go` | 91 | `f.chunks[ch] = src.Chunk(ch)` | `killsource.film` (copie interne) |
| `killsource/chunks.go` | 139, 142, 150, 163, 166 | `hasEvents`, puis les trois primitives `bitAt` / `bits32` / `bitsN` | `[]byte` nu |
| `killsource/feed.go` | 95 | `f.chunks[ch]` -> `analysis.ParseHighlightEvents` | `[]byte` nu |
| `killsource/film_table.go` | 108, 111, 115 | `f.chunks[0]` -> `ReadFilmIdentity`, `ReadPlayerTable` | `[]byte` nu |
| `killsource/world.go` | 55, 58 | `f.chunks[0]` -> `filmdec.ParseRegistryChunk` | `[]byte` nu |
| `killsource/world.go` | 254, 259, 262 | `bits32(buf, q)`, `bits32(buf, q+32)`, `bitsN(buf, q+58, 6)` | `[]byte` nu |
| `killsource/scan.go` | 63, 66, 85, 88, 92, 95, 99, 186, 193 | gates Mort/tag, tag u32, victime 5 b, tueur 5 b, categorie 4 b | `[]byte` nu |
| `killsource/assist.go` | 223, 226 | `bitAt` / `bitsN(pl, x, 7)`, puis `&evReader{pl: pl, bp: x + 7}` | `killsource.evReader` |
| `killsource/walk.go` | 61, 86, 101 | `filmdec.NewBitReader(pl)` ; `bitAt(pl, s-1)` (x2) | melange des DEUX lecteurs |
| `killsource/eventchain.go` | 104-130, 140-148 | le type `evReader` et `bitsWide` | `killsource.evReader` |
| `killsource/eventbody.go` | 19, 62, 76, 86, 132, 155, 171 | 7 corps d'evenement, tous parametres par `*evReader` | `killsource.evReader` |

**F. `analysis/objectiveevents` — troisieme lecteur (le pied de film)**

| Fichier | Ligne | Ce qui est lu | Via quel type |
|---|---:|---|---|
| `objectiveevents/extract.go` | 197 | `film.Chunk(footerPos)` — le chunk de type 3 | `*filmsource.Film` |
| `objectiveevents/film.go` | 144-155 | `readBitsBE(data, bitPos, n)` — MSB-first bit a bit | `[]byte` nu |
| `objectiveevents/film.go` | 157-168 | `readByteAtBit(data, bit)` — octet a offset bit non aligne | `[]byte` nu |
| `objectiveevents/film.go` | 170-176 | `readU64LEAtBit(data, bit)` — u64 LE a offset bit | `[]byte` nu |
| `objectiveevents/film.go` | 253-280 | `scanTh10Events` : motifs `0xc0` / `0x2d` / `0x25`, bornes xuid | `FooterEvent` |
| `objectiveevents/film.go` | 286-300 | `decodeTh10Block` : marqueur `00 00 2e e0`, bloc de 60 o, octets 36/37/47/48-51 | `FooterEvent` |
| `objectiveevents/film.go` | 125-133 | `framesOf(film, pos)` -> `[]filmsource.Packet` | `filmsource.Packet` |

**G. `analysis/weaponv3` — quatrieme lecteur**

| Fichier | Ligne | Ce qui est lu | Via quel type |
|---|---:|---|---|
| `weaponv3/pi_resolver.go` | 26-32 | `type bitReader` + `newBitReader(data []byte)` | `weaponv3.bitReader` |
| `weaponv3/pi_resolver.go` | 36-57 | `bit(p)`, `readBits(bp, n)` MSB-first | idem |
| `weaponv3/pi_resolver.go` | 76-82 | `ResolveXuidToPI(rosterXuids, chunk []byte)` : motif 64 b, 5 bits devant | `[]byte` nu |
| `weaponv3/bits_word.go` | (tout) | `wordBitsAt` — **copie divergente** de `filmdec/bits_word.go` (cf. §1.2) | — |
| `weaponv3/timing.go` | (3 `LittleEndian`) | horodatages dans le flux | `[]byte` nu |

**H. `internal/analysis` (racine, title-agnostic) — cinquieme et sixieme lecteurs**

| Fichier | Ligne | Ce qui est lu | Via quel type |
|---|---:|---|---|
| `analysis/highlight_event_parser.go` | 109-128 | `ParseHighlightEvents(data, filmMajorVersion)` : inflate zlib puis `scanEvents` (396 L) | `analysis.HighlightEvent` |
| `analysis/weapon_scanner.go` | 1-377 | `FormulaAPattern` `20 00 02`, `FrameMarker` `A0 7B 42`, `ScanFireEventsB5`, `FindFramePositions`, `ScanFormulaA(NS)`, `TimestampEstimator` | `[]byte` nu |
| `analysis/weapon_data.go` | — | `binary.*` sur flux de film | `[]byte` nu |

Consommateurs de production de ces deux fichiers :
`killsource/feed.go:95`, `film/replay/deaths_source.go:80`, `ops/medal_feed_backfill.go:227`,
`cmd/levelup/cmd_backfill_medailles_feed.go`, `sync/killcollector/shots.go:84`
(`analysis.ScanFireEventsB5`), `cmd/diag_film/main.go:95-113`.

**I. `internal/sync` — septieme lecteur**

| Fichier | Ligne | Ce qui est lu | Via quel type |
|---|---:|---|---|
| `sync/killcollector/shots.go` | 76-83 | `chunks [][]byte` — les chunks REPLICATION_DATA decompresses, balayes en direct | `[][]byte` nu |
| `sync/killcollector/shots.go` | 122-132, 151 | `resolvePlayerIndices`, `binary.LittleEndian.PutUint64(le[:], xuid)` | `[][]byte` nu |
| `sync/killcollector/shots.go` | 170-196 | `chercherMotifs` / `chercherDansChunk` : fenetre glissante + prefiltre `[1<<16]bool` | `[]byte` nu |
| `sync/killcollector/shots.go` | 224 | `lireIndiceAvant(data, debutMotif)` — lecture de bits en amont du motif | `[]byte` nu |
| `sync/killcollector/positions.go` | 470 | `len(film.Chunk(i)) == 0` — controle de sequence trouee | `*filmsource.Film` |

**J. `cmd/` — outils**

| Fichier | Ligne | Ce qui est lu | Via quel type |
|---|---:|---|---|
| `cmd/killsource/main.go` | 207 | `filmsource.LoadDir(dir, nil)` | `*filmsource.Film` |
| `cmd/rdata_weapon_scan/main.go` | 257 | `filmdec.NewBitReader(p.payload)` | `*filmdec.BitReader` |
| `cmd/diag_weapons_v3/positions.go` | 73 | `src.Chunk(i)` | `filmsource.Source` |
| `cmd/levelup/cmd_backfill_medailles_feed.go` | 185 | `filmsource.Inflate(registre)` -> `HighlightProfileFromHeader` | `[]byte` nu |
| `cmd/diag_film/main.go` | 95-113 | `analysis.FindFramePositions/ScanFormulaA/ScanFireEventsB5` sur `data` | `[]byte` nu |
| `cmd/zone-attribution/measure.go` | — | `filmsource` | `*filmsource.Film` |

**HORS PERIMETRE, ecrit pour que le ratchet ne les prenne pas** : `cmd/weapon-sounds/` (19
fichiers avec `binary.LittleEndian`) et `cmd/weapon-icons-build/` (dont son propre `bitReader`
dans `bc7.go` / `bc7_mode7.go`) lisent les fichiers d'INSTALLATION DU JEU (modules, banques
Wwise, textures BC7), jamais un film. `cmd/diag_lusr_player/{loaders,main}.go` cite `chunk[%d:%d]`
dans un message de log au sujet de tranches de participants en base — faux positif.

#### Bilan chiffre

| Ce qu'on compte | Nombre |
|---|---:|
| Lecteurs de bits DISTINCTS sur des octets de film | **6** (`filmdec.BitReader` ; `killsource.evReader` ; `killsource.bitAt/bits32/bitsN/bitsWide` ; `objectiveevents.readBitsBE/readByteAtBit/readU64LEAtBit` ; `weaponv3.bitReader` ; `analysis.scanEvents`) |
| Marcheurs de paquets distincts | **2** (`filmsource.appendPackets`, `filmdec/film_packets.go:60-90`) |
| Paquets de production qui lisent des octets bruts hors `filmsource` | **9** (`filmdec`, `killsource`, `objectiveevents`, `weaponv3`, `analysis` racine, `filmcache`, `sync/killcollector`, `sync/haloclient`, `replaybuild`) |
| Sites de production distincts, tableaux A a J | **environ 120** (dont 45 `NewBitReader` de `filmdec`) |

L'ADR 0034 ecrit « six packages outside the source layer read raw chunk bytes ». La mesure du
2026-09-17 en donne **neuf**, parce que `analysis` racine (deux fichiers), `sync/haloclient` et
`sync/killcollector/shots.go` n'etaient pas comptes.

### 1.2 (b) `killsource.evReader` contre `filmdec.BitReader`, methode par methode

| Question | `filmdec.BitReader` (`bitreader.go`) | `killsource.evReader` (`eventchain.go:104`) |
|---|---|---|
| Etat | `buf []byte`, `pos int`, plus 5 champs de PROFIL (`mv`, `kf`, `mpp`, `rsp`, `rspImpose`) | `pl []byte`, `bp int`, `over bool` |
| Construction | `NewBitReader(buf)` — pose le profil herite (`herite.mouvement`, `cadreDuProfil()`, `herite.mpp`, `herite.rsp`) | litteral `&evReader{pl: pl, bp: X}` en 3 sites (`assist.go:226`, `eventchain.go:243`, `eventchain.go:291`) |
| Lecture de n bits | `ReadBits(n uint) uint64` | `rd(n int) uint64` |
| Ordre des bits | MSB-first, big-endian ; `wordBitsAt` par mot de 64 quand `pos >= 0 && n <= 64`, sinon boucle | MSB-first ; `bitsWide` — boucle bit a bit, sans chemin par mot |
| Un bit | `ReadBit() bool` | `g1() int` |
| Saut | `Skip(n int)` — avance `pos`, **sans borne** | `skip(n int)` — refuse et leve `over` si depassement |
| Position | `BitPos() int`, `SetBitPos(p int)` | `bp` en champ du paquet, ecrit directement |
| Reste | `Remaining() int` | absent |
| **Fin de flux** | bits au-dela du tampon lus comme **ZERO**, silencieusement (« matching the engine's tail padding ») | `rd` **refuse** : pose `over = true` et rend 0 sans avancer `bp` |
| Erreurs | aucune ; aucun drapeau | `over` — en-tete du type : « une lecture hors tampon rend des zeros, et sans lui une chaine desynchronisee lit des evenements parfaitement valides apres la fin du paquet » |
| Types de largeur | `n uint` | `n int`, avec garde `n <= 0 -> 0` |
| Codecs de valeur | `ReadSignedVarWidth` (selecteur 2 b, largeur `8 << sel`, extension de signe pour 8 et 16), `ReadQuantizedVec3`, `readQuantStat`, 8 accesseurs de profil | aucun |

#### Les trois ecarts qui empechent l'absorption telle quelle

1. **La convention de fin de flux est OPPOSEE, et c'est semantique, pas cosmetique.**
   `BitReader` bourre a zero (il modelise le moteur) ; `evReader` refuse et leve `over` (il
   modelise la MEFIANCE sur une chaine d'evenements potentiellement desynchronisee). Absorber
   `evReader` en appelant `ReadBits` ferait disparaitre `over`, et avec lui l'arret de chaine :
   la marche « lirait » des evenements valides apres la fin du paquet. **L'absorption doit
   laisser le drapeau quelque part**, pas le supprimer. Forme proposee : `BitReader.Remaining()`
   existe deja ; les appelants de `killsource` testent `Remaining() >= n` AVANT chaque
   `ReadBits` et portent leur propre `over` — aucune valeur lue ne change alors, et le drapeau
   reste au bon endroit (le marcheur de chaine, pas le lecteur). Variante plus lourde et non
   recommandee : un champ `strict bool` sur `BitReader`, qui ferait porter au lecteur canonique
   une regle qui n'appartient pas au moteur.
2. **`Skip` n'a pas la meme portee.** `BitReader.Skip` avance sans borne (c'est licite : la
   lecture suivante lira des zeros) ; `evReader.skip` refuse au-dela du tampon. Meme parade que
   ci-dessus : le test de bornes remonte chez l'appelant.
3. **Le lecteur canonique porte un PROFIL, `evReader` n'en porte aucun.** Depuis 2.2.a-2.2.e,
   `NewBitReader` installe cinq groupes de valeurs heritees. Construire un `BitReader` la ou
   vivait un `evReader` fait donc ENTRER le profil herite dans un chemin qui ne l'avait pas —
   et `killsource` est precisement le paquet qui CALIBRE ce profil (`calibrate.go:114`
   `filmdec.PoserMouvementHerite`). Rien ne le LIT dans la chaine d'evenements aujourd'hui (les
   corps `evBody*` n'appellent que `rd`, `g1`, `skip`), donc le risque est nul a l'octet, **mais
   il ne l'est qu'a condition que 2.3 soit deja passe** : apres 2.3 le profil ne se lit plus par
   effet de bord, et la question disparait. C'est le premier argument pour faire **2.4 APRES
   2.3** — ce que le plan prevoit deja.

Ecart secondaire a ne pas oublier : `killsource` a AUSSI trois primitives sans lecteur
(`bitAt`, `bits32`, `bitsN`, `chunks.go:142-168`) employees sur 23 sites, et `bitsWide`
(`eventchain.go:140`). `bits32` lit **40 bits puis decale** (`for k := 0; k < 5; k++` puis
`>> (8 - sh)`), ce qui n'est pas la meme boucle que `wordBitsAt` ; l'equivalence doit etre
prouvee pour elles aussi, pas seulement pour `evReader`.

#### Forme de l'equivalence bit a bit a tester sur les mini-films

Un test dans `killsource` (paquet interne, acces aux deux lecteurs), sur le corpus des 7
mini-films de V7 (au plus 1 Mio chacun, `chunk_00` + un chunk d'image-cle + le pied) :

```
TestEquivalenceBitAbitEvReaderContreBitReader
  pour chaque mini-film,
    pour chaque paquet type-0 du film,
      pour chaque position de bit d'ancrage que la chaine d'evenements visite
        (les positions REELLES, obtenues en instrumentant evStep / evPresence,
         jamais un balayage aveugle : le cout serait en heures),
        pour chaque largeur n de 1 a 64 :
          r := evReader{pl: pl, bp: bp} ; a := r.rd(n) ; overA := r.over
          b := NewBitReader(pl) ; b.SetBitPos(bp) ; vb := b.ReadBits(uint(n))
          EXIGER : overA == (bp+n > len(pl)*8)
                   si !overA : a == vb            (egalite stricte)
                   si  overA : a == 0             (contrat actuel d'evReader)
  puis, chaine entiere :
    la liste (code, bitPos de debut, bitPos de fin) produite par la marche
    d'evenements doit etre IDENTIQUE avant et apres absorption, film par film.
```

Le second bloc est le vrai gate : l'egalite par appel prouve la primitive, l'egalite des
**positions de fin de chaque evenement** prouve que la consommation de bits n'a pas bouge. Meme
forme pour `bits32` / `bitsN` / `bitsWide` (trois tests de table, positions reelles).

Le golden naturel existe deja : `killsource` ecrit `KillSourceDecoderRev` sur chaque ligne, et
`TestKillSourceDecoderRevSuitLeDecodeur` exige que la revision monte quand la source de
`killsource/` change. 2.4 fera donc monter `KillSourceDecoderRev` (source changee) **avec sortie
identique** — c'est exactement la preuve que l'item 2.4 demande.

### 1.3 (c) Esquisse de la facade `film.Source` et du ratchet

#### Le probleme d'emplacement, a trancher AVANT d'ecrire une ligne

L'item 2.4.2 demande que `objectiveevents/film.go` et `weaponv3/pi_resolver.go` « traversent »
`film.Source`. Ces deux paquets vivent sous `internal/analysis/`, ou
`archlint/no_title_package_in_analysis_test.go` **interdit d'importer
`internal/games/halo_infinite/...`** (D9, ADR 0012 / ADR 0025). Poser la facade sous `film/` et
la faire importer par `analysis/objectiveevents` ou `analysis/weaponv3` **rougit ce ratchet le
jour meme**, et ni l'un ni l'autre n'est dans l'allowlist (§2.6 : 5 entrees, aucune n'est
celle-la).

Trois issues, par ordre de preference du redacteur :

- **(i) Recommandee — la facade nait sous `internal/analysis/filmsource/`**, c'est-a-dire LA OU
  la source vit deja. `filmsource` est le seul paquet du depot dont un ratchet dedie
  (`archlint/filmsource_leaf_test.go`) garantit qu'il n'importe RIEN du depot : c'est deja la
  couche `source`, a l'emplacement pres. 2.4 l'etoffe (lecteur de bits canonique, lecture des
  quatre sections de `chunk_00`, `Film.Identity`), 2.5.a le DEPLACE en bloc sous
  `film/internal/source` par `git mv` pur. Aucun import `analysis -> games` n'apparait jamais,
  D9 reste vert, et 2.5.a reste un deplacement pur.
- (ii) Intervertir les pas : 2.5 d'abord, 2.4 ensuite. Cout : 2.5 est le lot le plus risque du
  jalon (regime COMPLET impose par V2) et il deplacerait SIX lecteurs de bits au lieu d'un.
- (iii) Allowlister temporairement `weaponv3` et `objectiveevents` dans D9. A ecarter : D9 dit
  que l'allowlist SE VIDE au pas 5, pas qu'elle grossit au pas 4.

#### Esquisse des types (emplacement (i) : paquet `filmsource`, futur `source`)

```go
// Source : la porte unique aux octets d'un film. Une implantation de production
// (le cache disque), une de test (memoire).
type Source interface {          // EXISTE DEJA (filmsource/source.go)
    NumChunks() int
    Chunk(i int) ([]byte, error) // octets BRUTS, potentiellement compresses
}

// Film : le film charge, en LECTURE SEULE apres Load.  EXISTE DEJA, a etoffer.
type Film struct { /* chunks, all, bounds, meta ... prives */ }

func (f *Film) NumChunks() int
func (f *Film) Chunk(i int) []byte          // a rendre non exporte a terme (cf. ratchet)
func (f *Film) Packets(i int) []Packet
func (f *Film) AllPackets() []Packet
func (f *Film) Meta() []ChunkMeta

// AJOUTS DU LOT 2.4 — ce qui remonte de filmdec et de killsource :
func (f *Film) Identity() (Identity, bool)  // section 2 de chunk_00 (V5 + 1.5.1)
func (f *Film) Header() Header              // versions de format et majeure (chunk_00+0, +4)
func (f *Film) Registry() (Registry, error) // memo, comme FilmContext le fait aujourd'hui

// Le lecteur de bits, UNIQUE, SANS profil (le profil est la couche du dessus) :
type Bits struct{ /* buf, pos */ }
func NewBits(buf []byte) *Bits
func (b *Bits) ReadBits(n uint) uint64      // MSB-first, bourrage a zero au-dela du tampon
func (b *Bits) ReadBit() bool
func (b *Bits) Skip(n int)
func (b *Bits) BitPos() int
func (b *Bits) SetBitPos(p int)
func (b *Bits) Remaining() int              // ce qui permet a killsource de porter son `over`
func (b *Bits) ReadSignedVarWidth() int32   // codec du moteur (FUN_140c18a1c)
```

Le `BitReader` d'aujourd'hui = `Bits` + cinq champs de profil. **La scission est le vrai travail
de 2.4** : le lecteur nu descend en `source`, les accesseurs de profil (`fullPrecision`,
`deltaHasHandleTail`, `calibratedSkip`, `deltaQuantum`, `worldPositionRange`, `absoluteAxisW`,
`traversal`, `worldObjectPrecision`, `mppWidths`, `recordStateParam`, `cadre`) restent en
`grammar` sur un type qui EMBARQUE `*source.Bits`. Dependance dans le bon sens.

Qui la traverse apres 2.4 : `filmdec` (45 sites `NewBitReader` + 6 fichiers `film_*.go`),
`killsource` (`chunks.go`, `feed.go`, `walk.go`, `world.go`, `scan.go`, `assist.go`,
`eventchain.go`, `eventbody.go`), `objectiveevents/film.go` (les trois helpers disparaissent),
`weaponv3/pi_resolver.go` (`bitReader` et `bits_word.go` disparaissent), `filmcache` (inchange :
c'est une `Source`), `analysis/highlight_event_parser.go` et `analysis/weapon_scanner.go`
(**NON traites par 2.4 tel qu'ecrit** — cf. ci-dessous et questions ouvertes).

#### Esquisse du ratchet `archlint/no_raw_film_bytes_outside_source_test.go`

```
RACINES SURVEILLEES (relatives a apps/go-api, tests COMPRIS — c'est par les tests
que la porte s'est ouverte pour D9) :
  internal/games/halo_infinite/film             (tout l'arbre)
  internal/analysis/objectiveevents
  internal/analysis/weaponv3
  internal/analysis/{highlight_event_parser,weapon_scanner,weapon_data}.go
  internal/sync/killcollector
  internal/sync/haloclient
  internal/replaybuild
  cmd/   (SAUF cmd/weapon-sounds, cmd/weapon-icons-build : fichiers du JEU, pas des films)

EXCLU DE LA REGLE : le paquet source lui-meme.

MOTIFS (analyse AST, pas grep — les chemins cites en commentaire abondent) :
  R1  declaration d'un type portant a la fois un champ []byte et un champ de position
      en BITS, plus une liste NOMMEE d'identifiants interdits : bitReader, evReader,
      readBitsBE, readByteAtBit, readU64LEAtBit, bitsWide, bits32, bitsN, bitAt,
      wordBitsAt.
  R2  indexation `chunk[...]` / `chunks[...]` sur une valeur de type []byte ou [][]byte
      dont le nom contient chunk (insensible a la casse).
  R3  `binary.LittleEndian.*` / `binary.BigEndian.*` applique a une valeur issue de
      `.Chunk(`, `.Payload`, `Inflate(` ou d'un parametre nomme chunk/payload/pl/raw.
  R4  appel a `compress/zlib` (l'inflate est au source, et rien qu'a lui).
  R5  fonction dont un parametre est []byte ET dont le corps contient un decalage
      `>> uint(7-` ou `<< sh` sur cet octet.

PLANCHER : nombre minimal de fichiers .go parcourus (modele du plancher a 300 de
no_title_package_in_analysis_test.go) — un ratchet qui ne balaye rien passe en silence.

MUTATION QUI DOIT LE FAIRE ROUGIR : reintroduire
`func bitAt(d []byte, p int) int` dans killsource.
```

**L'allowlist doit-elle etre VIDE des le premier jour ?** L'ADR 0034 D-2 dit oui (« allowlist
empty from the day it is written »), l'item 2.4.3 aussi. **La mesure dit que ce n'est realisable
que si 2.4 absorbe AUSSI trois choses que l'item ne nomme pas** :

| Ce qui reste hors source apres 2.4 tel qu'ecrit | Pourquoi |
|---|---|
| `analysis/highlight_event_parser.go` (396 L, inflate zlib + `scanEvents`) | non cite par 2.4.2 ; 5 consommateurs de production |
| `analysis/weapon_scanner.go` (377 L, motifs `20 00 02` / `A0 7B 42`) | non cite ; consomme par `killcollector/shots.go:84` et `cmd/diag_film` |
| `sync/killcollector/shots.go:76-224` (fenetre glissante sur `[][]byte`) | non cite ; c'est un balayage d'octets de film a part entiere |

Deux options, a trancher par l'executeur ou le pilote :

- **(A) allowlist vide, perimetre elargi** : 2.4 absorbe aussi ces trois-la. Cout : le lot passe
  de M a L, et il touche `analysis/` racine (paquet tres importe).
- **(B) allowlist vide sur les racines du film, ces trois-la HORS des racines surveillees, avec
  une ligne datee et une cible de retrait** (modele : `franchissementsToleres` de D9, qui porte
  date + paquet vise + portage attendu). Ce n'est pas une allowlist d'exemptions au sens de D-2 :
  c'est un perimetre declare. Recommandation du redacteur : **(B)**, en inscrivant les trois
  lignes au registre des reports avec pour cible le pas 5 (elles descendent naturellement en
  `facts` avec `killsource`), et en le disant dans l'en-tete du ratchet.

### 1.4 (d) Risques de conflit avec le lot 2.3

2.3 est declare toucher : les globales restantes de `filmdec`, les `install` / `restore`, la
double ecriture datee, `LockProcessDecode` / `decode_gate.go`, la calibration de `killsource`
portee dans le profil, et les appelants qui prenaient le verrou.

Sites mesures de `LockProcessDecode()` en production (hors `archlint` et hors tests) :

```
filmdec/decode_gate.go:31              (la definition)
killsource/decode.go:85
film/replay/build_from_film.go:58
sync/killcollector/hits.go:127
sync/killcollector/positions.go:269
cmd/rdata_weapon_scan/main.go:713
```

| Fichier | Touche par 2.3 | Touche par 2.4 | Gravite |
|---|---|---|---|
| `filmdec/bitreader.go` | OUI — `NewBitReader` cesse de lire `herite.*` (5 champs) | OUI — le lecteur nu descend en `source` | **CONFLIT FRONTAL.** Les deux lots reecrivent le meme constructeur. |
| `filmdec/profil_herite.go` (95 L) | OUI — supprime avec son kill-switch | pas directement, mais sa disparition change la signature de `NewBitReader` | fort |
| `filmdec/decode_gate.go` (34 L) | OUI — supprime | NON | nul si 2.3 passe avant |
| `killsource/calibrate.go` | OUI — `PoserMouvementHerite` (l. 114) et `SetRecordStateParam` (l. 167, 176) disparaissent au profit du profil | NON | nul |
| `killsource/walk.go` | NON | OUI — `filmdec.NewBitReader(pl)` l. 61 + `bitAt` l. 86, 101 | faible |
| `killsource/decode.go` | OUI (l. 85, verrou) | NON | nul |
| `film/replay/build_from_film.go` | OUI (l. 58, verrou) | NON | nul |
| `film/replay/world_object_precision.go` | OUI (installateur par carte, 2.2.b) | NON | nul |
| `sync/killcollector/{hits,positions}.go` | OUI (verrou) | NON (`positions.go:470` en lecture seule) | faible |
| `cmd/rdata_weapon_scan/main.go` | OUI (l. 713, verrou) | OUI (l. 257, `NewBitReader`) | faible, deux lignes distinctes |
| `archlint/decode_lock_held_test.go` | OUI — **INVERSE** par 2.3.2 | NON (2.4 ajoute un ratchet distinct) | nul |
| `archlint/filmdec_package_vars_test.go` | OUI — descendu de 43 a 0 | NON | nul |

**Conclusion : un seul conflit reel, `filmdec/bitreader.go`, et il est structurel** (2.3 vide le
constructeur, 2.4 le deplace). La regle « un seul muteur de `filmdec` » du plan suffit a
l'eviter : **2.4 ne demarre qu'apres la fusion de 2.3**, ce que le plan prevoit deja. Aucun
amenagement n'est necessaire, mais l'executeur de 2.4 doit **rouvrir `bitreader.go` sur pieces a
l'entree** (regle 4 de `plan-execution`) : la forme du constructeur aura change.

Second point de vigilance, deja nomme au §1.2 : la calibration de `killsource` fuit aujourd'hui
vers `replay.BuildFromFilm` par l'etat du processus (D1 (2.2), consigne dans 2.3.1). Si 2.4
partait AVANT 2.3, absorber `evReader` dans un `BitReader` porteur du profil herite pourrait
changer cette fuite de facon invisible.

---

## 2. Lot 2.5 (pas 5) — Les cinq couches, par deplacements purs

### 2.1 (a) Le graphe d'imports ACTUEL

`go list -f '{{.ImportPath}} {{.Imports}}'`, sortie collee ; prefixe
`levelup/go-api/internal/` abrege en `~/`, bibliotheque standard retiree :

```
~/games/halo_infinite/film/damagetag
    (aucun import du depot)
~/games/halo_infinite/film/medalname
    (aucun import du depot)
~/games/halo_infinite/film/killicon
    ~/games/halo_infinite/film/damagetag
~/games/halo_infinite/film/replay/mapvar
    (aucun import du depot)
~/games/halo_infinite/film/replay/fallback
    (aucun import du depot)
~/analysis/filmsource
    (aucun import du depot)              <- ratchet archlint/filmsource_leaf_test.go
~/games/halo_infinite/film/filmcache
    ~/analysis/filmsource
~/games/halo_infinite/film/filmdec
    ~/analysis/filmsource
~/analysis/weaponv3
    ~/analysis
~/analysis/objectiveevents
    ~/analysis/filmsource   ~/domain
~/games/halo_infinite/film/killsource
    ~/analysis   ~/analysis/filmsource   ~/games/halo_infinite/film/damagetag
    ~/games/halo_infinite/film/filmdec
~/games/halo_infinite/film/replay
    ~/analysis   ~/analysis/filmsource   ~/analysis/objectiveevents   ~/analysis/weaponv3
    ~/games/canonical   ~/games/halo_infinite/film/filmdec
    ~/games/halo_infinite/film/replay/fallback   ~/games/halo_infinite/film/replay/mapvar
    ~/observability
~/domain/replaydoc
    (aucun import du depot)
~/service/replayview
    ~/domain/replaydoc   ~/games/canonical   ~/games/halo_infinite/film/replay
~/replaybuild
    ~/analysis   ~/analysis/filmsource   ~/analysis/objectiveevents   ~/domain
    ~/domain/title   ~/games/halo_infinite   ~/games/halo_infinite/film/filmcache
    ~/games/halo_infinite/film/filmdec   ~/games/halo_infinite/film/killsource
    ~/games/halo_infinite/film/replay   ~/games/halo_infinite/film/replay/mapvar
    ~/games/halo_infinite/replayidentity   ~/games/halo_infinite/replaylabels
    ~/games/mappings   ~/observability   ~/platform/atomicfile   ~/port
~/sync/killcollector
    ~/analysis   ~/analysis/filmsource   ~/analysis/weaponv3   ~/domain/killscope
    ~/domain/title   ~/games   ~/games/halo_infinite
    ~/games/halo_infinite/film/filmcache   ~/games/halo_infinite/film/filmdec
    ~/games/halo_infinite/film/killsource   ~/games/halo_infinite/film/replay
    ~/games/halo_infinite/ingest   ~/games/halo_infinite/replayidentity   ~/migration
    ~/observability   ~/persist   ~/port   ~/sync/haloclient   ~/sync/matchflags
~/games/halo_infinite/filmprofile
    (aucun import du depot)              <- lot 3.1 volet donnees, deja livre
```

Point remarquable et favorable : **`filmdec` n'importe qu'un seul paquet du depot**
(`analysis/filmsource`). Les cinq couches sont donc deja presque en ordre a l'INTERIEUR du
decodeur ; ce qui est a l'envers est la frontiere avec `analysis/`.

### 2.2 Le graphe CIBLE, et les aretes a casser

Cible (ADR 0034 D-1), sous `internal/games/halo_infinite/film/` :

```
  internal/source   <- filmsource + lecture de chunk_00 + lecteur de bits canonique
        |
        v
  internal/profile  <- Profile, profile_table, MapQuantCatalog, controle par le film
        |
        v
  internal/grammar  <- filmdec (lecteurs, FilmContext, inference de chaines)
        |
        v
  internal/facts    <- killsource, objectiveevents, identite, equipement, vehicules,
        |              projectiles, grenades, fallback/
        v
  replay            <- publication du document versionne (EXPORTE : consomme hors film/)

  Feuilles hors couches, inchangees : damagetag, killicon, medalname, filmcache, mapvar,
  filmprofile, domain/replaydoc.
```

**Les aretes qui vont dans le mauvais sens aujourd'hui** (celles que 2.5 doit casser) :

| # | Arete actuelle | Depart -> visee | Pourquoi c'est a l'envers | Ce que 2.5 en fait |
|---:|---|---|---|---|
| 1 | `film/replay` -> `analysis/objectiveevents` | replay -> facts, mais `objectiveevents` est SOUS `analysis/` | une couche interne du decodeur vit hors du decodeur | 2.5.d : `objectiveevents` descend en `facts` ; l'arete devient replay -> facts, dans le bon sens |
| 2 | `film/killsource` -> `analysis/filmsource` | facts -> source, hors decodeur | idem | 2.5.a : devient facts -> source |
| 3 | `film/filmdec` -> `analysis/filmsource` | grammar -> source, hors decodeur | idem | 2.5.a : devient grammar -> source |
| 4 | `film/replay` -> `analysis/filmsource` | replay -> source, **saut de deux couches** | `replay` ne doit rien decoder (D-1) ; il charge pourtant le film lui-meme (`deaths_source.go:36`, `inventory_decode.go:177`, `origin.go:82`, `player_index.go:56`) | a CASSER : ces quatre sites remontent en `facts`, ou `replay` recoit un `*source.Film` deja charge |
| 5 | `film/replay` -> `analysis/weaponv3` | replay -> un 4e lecteur de bits sous `analysis/` | `weaponv3.ResolveXuidToPI` est de la GRAMMAIRE | 3 symboles seulement cote weaponv3 (`CommonWeaponSuffix`, `WeaponFusionMap`, `WeaponIDToName`) : le resolveur descend en `grammar` / `facts`, le catalogue d'armes remonte en `games/weapons` (lingua franca) |
| 6 | `film/replay` -> `analysis` (racine) | replay -> parseur d'evenements du film | `analysis.ParseHighlightEvents` est de la grammaire de film dans un paquet title-agnostic | 4 usages seulement (`ParseHighlightEvents` x3, `WeaponIDToName`, `HighlightEvent`, `EventTypeDeath`) |
| 7 | `film/killsource` -> `analysis` (racine) | facts -> meme parseur | idem | 4 symboles (`EventTypeDeath`, `EventTypeKill`, `HighlightEvent`, `ParseHighlightEvents`) |
| 8 | `analysis/weaponv3` -> `analysis` (racine) | hors decodeur, mais bloque le portage de 5 | 3 symboles (catalogue d'armes) | resolu par 5 |
| 9 | `analysis/sessionusage/usage_outcomes.go` -> `film/replay` | **SEUL franchissement de PRODUCTION de l'allowlist D9** | lit `replay.EquipmentOutcomeFamilies()`, `replay.EquipmentFamilyPowerupCamo`, `replay.EquipmentFamilyPowerupOvershield`, `replay.UsageFamilySpawnsPiece` | 2.5.e : ces types remontent en `domain/` ou `games/canonical/` (portage deja ecrit dans l'allowlist), apres quoi l'entree disparait |
| 10 | `analysis/filmsource/source_test.go` -> `film/filmdec` | test externe comparant les DEUX marcheurs de paquets | l'un des deux disparait en 2.4 | l'entree D9 tombe avec lui |
| 11 | `analysis/objectiveevents/{assaut_footer_research,extract}_test.go` -> `film/filmcache` | tests de recherche ouvrant des films reels du cache | ces tests DESCENDENT avec `objectiveevents` en 2.5.d | les deux entrees D9 tombent par le deplacement |
| 12 | `analysis/weapon_index_equivalence_test.go` -> `film/filmdec` | test d'equivalence d'index d'armes | l'index d'armes est title-agnostic (`games/weapons`) | 5e et derniere entree D9 |

**Aretes dans le BON sens, a preserver telles quelles** (ecrit pour que 2.5 ne les touche pas) :
`killsource -> filmdec` (facts -> grammar), `killicon -> damagetag`, `replay -> mapvar`,
`replay -> fallback`, `replay -> canonical`, `service/replayview -> replay`,
`replaybuild -> replay`.

### 2.3 (b) La carte des deplacements, fichier par fichier

#### `internal/analysis/filmsource/` -> `film/internal/source` (V5) — 3 fichiers, 499 L

```
doc.go  film.go  source.go
```

Les trois vont en `source`, sans exception. Symboles exportes consommes hors `film/` : **7** —
`ChunkMeta`, `Film`, `Inflate`, `Load`, `LoadDir`, `MemoryChunks`, `Packet`. Types exportes du
paquet : `ChunkMeta`, `Film`, `MemoryChunks`, `Packet`, `Source`.

#### `internal/analysis/objectiveevents/` -> `film/internal/facts` — 20 fichiers, 5 081 L

```
awards.go              extract.go             families.go            film.go
flag_grabs_net.go      flagfilm.go            helpers.go             named.go
named_bounds.go        named_series.go        rosterfit.go           round_bounds.go
rounds_decision.go     score.go               slotidentity.go        slotidentity_deaths.go
slotidentity_elimination.go                   slotidentity_residue.go
slotidentity_rounds.go statborg.go
```

Tous en `facts`, **sauf les trois helpers de lecture de bits de `film.go`** (`readBitsBE` l. 144,
`readByteAtBit` l. 157, `readU64LEAtBit` l. 170) qui sont de la couche `source` et disparaissent
en 2.4. `film.go` porte aussi `scanTh10Events` / `decodeTh10Block` (offsets 36, 37, 47, 48-51
d'un bloc de 60 o) : c'est de la **grammaire** au sens de `GrammarRev`, et le godoc du ratchet
d'empreinte le dit deja. Deux voies possibles, a trancher : bloc entier en `grammar`, ou bloc en
`facts` avec ses seules lectures de bits deleguees a `source`. Recommandation : la seconde —
`decodeTh10Block` produit un `FooterEvent`, qui est un FAIT.

Symboles exportes consommes hors `film/` : **38**.
Types traversants : **14** — `DeathInstant`, `FlagFilmSignals`, `FlagGrabsNetPlayer`, `FlagSpan`,
`FlagTrack`, `IdentifiedEvent`, `NamedEvent`, `PlayerLine`, `RoundBounds`, `RoundIdentity`,
`RoundsDecision`, `ScorePoint`, `StatComponent`, `StatRecord` (sur 25 types exportes).

#### `internal/games/halo_infinite/film/killsource/` -> `film/internal/facts` — 23 fichiers, 5 787 L

```
assist.go       bijection.go    botmeta.go      calibrate.go    chunks.go       decode.go
doc.go          eventbody.go    eventchain.go   feed.go         feed_couples.go film_table.go
health.go       hybrid.go       kill.go         label.go        match.go        options.go
paquet_identite.go              roster.go       scan.go         walk.go         world.go
```

Tous en `facts`, **sauf** : les primitives de bits de `chunks.go` (l. 142-168 : `bitAt`,
`bits32`, `bitsN`) et d'`eventchain.go` (l. 104-148 : `evReader`, `bitsWide`), qui appartiennent
a `source` et disparaissent en **2.4**, avant le deplacement. `calibrate.go` : sa calibration
remonte dans le profil en **2.3** ; ce qui reste va en `facts`.

Symboles exportes consommes hors `film/` : **33**.
Types traversants : **6** — `ApparStats`, `Assist`, `CoupleStats`, `FilmTablePinning`, `Kill`,
`Result` (sur 25 types exportes).

#### `internal/games/halo_infinite/film/filmdec/` -> `film/internal/grammar` — 142 fichiers, 31 011 L

Liste `ls` (production seule, ordre alphabetique) :

```
ability_charges.go ability_energy.go ability_impulses.go ability_rank.go ability_state_hooks.go
aim_vector.go biped_creation.go biped_pickups.go bit_leaf_readers.go bitreader.go bits_word.go
build_profile.go camo_state.go capture.go component_param4.go components_batch3.go
components_batch4.go components_batch7.go components_batch8.go components_biped_ability.go
components_biped_anchor.go components_biped_spartan.go components_cubemap.go
components_dynprec_orientation.go components_flock.go components_game_engine.go
components_managed_object.go components_managed_objective.go components_managed_property.go
components_movement.go components_object.go components_object_state.go components_player.go
components_position_i0.go components_probe.go components_team_mapping.go components_walk_batch9.go
components_world.go decode_gate.go default_state.go default_state_arch.go default_state_ti40.go
default_state_ti42.go delta_biped_walk.go dispatch_biped.go dispatch_item.go dispatch_object.go
dispatch_player.go doc.go emp_timer.go equipment_changes.go equipment_creation.go
equipment_creation_width.go equipment_placements.go equipment_recovery.go equipment_spawn_events.go
equipment_state.go event_list.go film_chunks.go film_context.go film_format_version.go
film_identity.go film_major_version.go film_packets.go fire_aim_modal.go fire_events.go
frame_chain_infer.go frame_harvest.go frame_infer.go frame_records.go grammar_rev.go
grapple_state.go grenade_events.go ground_weapon_ammo.go ground_weapon_creation.go
held_weapon_changes.go i0_layout.go inventory_delta.go inventory_delta_ammo.go
inventory_delta_stats.go keyframe_carrier_mark.go keyframe_closure.go keyframe_entity_queue.go
keyframe_fullstate_loop.go keyframe_ground_weapons.go keyframe_loadout.go keyframe_record_spans.go
keyframe_record_walk.go keyframe_scan_counters.go keyframe_world.go killhealth.go map_bounds.go
mpp_widths.go navpoint_radial_rises.go navpoint_radial_scan.go navpoint_radial_segments.go
object_deaths.go object_deaths_calibrate.go object_deaths_march.go objective_scan.go
observateur.go offline_aim.go offline_aim_only.go offline_biped.go offline_biped_band.go
offline_filters.go player_table.go player_table_control.go player_table_record.go player_teams.go
position_capture.go probe_export.go profil_herite.go profile.go profile_table.go projectiles.go
quantize.go quantize_endpoint.go registry.go registry_fingerprint.go slot_band_dense.go
slot_band_filled.go slot_band_observed.go tlv_mode2.go transloc_events.go traverse.go
traverse_precision.go unit_control.go unit_equipment_scan.go unit_ref_probe.go unit_weaponstate.go
varwidth.go vehicle_creation.go vehicle_occupancy.go vitality.go weapon_hit_distance_resolver.go
weapon_hits.go weapon_hits_decode.go world.go world_object_census.go zone_state_scan.go zoom_events.go
```

**La regle** : `filmdec` EST la couche `grammar` ; tout y reste, sauf les exceptions ci-dessous.
C'est la seule facon honnete de statuer 142 fichiers sans en inventer la nature.

Exceptions vers `source` (14 fichiers, mesures) :

| Fichier | L | Ce qu'il porte |
|---|---:|---|
| `bitreader.go` | 141 | le lecteur canonique (moins ses 5 champs de profil, qui restent en `grammar`) |
| `bits_word.go` | 72 | `wordBitsAt` |
| `varwidth.go` | 128 | les 9 categories de plage de `FUN_1406d3140` (codec de valeur) |
| `film_packets.go` | 100 | second marcheur de paquets — **a SUPPRIMER en 2.4**, pas a deplacer |
| `film_chunks.go` | 179 | pont film charge / balayages |
| `film_major_version.go` | 93 | u32 LE en tete de `chunk_00` |
| `film_format_version.go` | 213 | u32 LE a `chunk_00+4` |
| `film_identity.go` | 310 | section 2 de `chunk_00` (1.5.1) |
| `registry.go` | 381 | `ParseRegistryChunk` — registre ECS de `chunk_00` |
| `registry_fingerprint.go` | 124 | empreinte du registre |
| `player_table.go` | 420 | table des 32 joueurs, `chunk_00` (1.5.2) |
| `player_table_record.go` | 183 | un enregistrement de slot, champ par champ |
| `player_table_control.go` | 120 | controle du profil PAR le film (1.5.2) |
| `slot_band_observed.go` | 68 | importe deja `filmsource` |

`player_table*.go` : arbitrage a faire. Le plan (2.5.b) envoie un `player_table_profile.go` en
`profile` ; **ce fichier n'existe pas sous ce nom** — les trois qui existent sont ci-dessus. La
lecture (`ReadPlayerTable`) est du `source` / `grammar`, le CONTROLE
(`player_table_control.go`) est du `profile`. A trancher a l'entree du lot, sur pieces.

Exceptions vers `profile` (7 fichiers, 1 695 L) :

| Fichier | L | Ce qu'il porte |
|---|---:|---|
| `profile.go` | 321 | `Profile` a champs prives, accesseurs par valeur (2.1.1) |
| `profile_table.go` | 353 | la table `Cle / Champ / Valeur / Source / Preuve / Date` |
| `build_profile.go` | 449 | ce qui varie d'un build a l'autre |
| `map_bounds.go` | 146 | bornes de dequantification par carte (`MapQuantCatalog`) |
| `i0_layout.go` | 284 | decoupage binaire du composant i0 |
| `mpp_widths.go` | 75 | decoupage du bloc `object-multiplayer-properties` |
| `slot_band_dense.go` | 67 | bande de slots bipede, en tableau indexe |

Disparaissent avant le deplacement (aucune couche cible) :

| Fichier | L | Quand |
|---|---:|---|
| `decode_gate.go` | 34 | **2.3** (`LockProcessDecode` supprime) |
| `profil_herite.go` | 95 | **2.3** (kill-switch retire a sa date cible) |
| `film_packets.go` | 100 | **2.4** (second marcheur absorbe) |

`observateur.go` (321 L, 37 variables devenues champs au 2.2.f) : reste en `grammar`, mais son
en-tete annonce que les 29 crochets de deserialiseur « attendent le pas 5 » — c'est le seul
fichier de `filmdec` dont 2.5 changerait AUTRE CHOSE que le chemin. A statuer explicitement.

Symboles exportes consommes hors `film/` : **49**.
Types traversants : **53** vers `film/replay`, **10** vers `replaybuild` + `sync/killcollector`
(dont 3 qui ne sont pas dans les 53 : `WeaponDamage`, `WeaponHitDistanceFunc`,
`WeaponHitStats`), sur **148** types exportes. Detail au §3.2.

### 2.4 (c) Les outils `cmd/` et les consommateurs hors `film/` a re-pointer

**Le plan dit « 7 outils `cmd/` ». La mesure en donne 23 qui importent un paquet de `film/`, et
12 qui importent un paquet DEPLACE sous `internal/`** (donc devenu inaccessible).

`cmd/` important un paquet DEPLACE (`filmdec`, `killsource`, `filmsource`, `objectiveevents`) —
**12 outils**, a re-pointer sur une facade exportee :

| Outil | Paquets deplaces qu'il importe |
|---|---|
| `cmd/killsource` | `filmsource`, `filmdec`, `killsource` |
| `cmd/levelup` | `filmsource`, `filmdec` |
| `cmd/zone-attribution` | `filmsource`, `filmdec`, `objectiveevents` |
| `cmd/mapfond-build` | `filmdec` |
| `cmd/mapplafond-mesure` | `filmdec` |
| `cmd/mapquant-build` | `filmdec` |
| `cmd/mapstruct-build` | `filmdec` |
| `cmd/rdata_weapon_scan` | `filmdec` |
| `cmd/diag_weapons_v3` | `objectiveevents` |
| `cmd/oddball-terrain` | `objectiveevents` |
| `cmd/replay-equiv` | `objectiveevents` |
| `cmd/statnames-sweep` | `objectiveevents` |

`cmd/` important un paquet de `film/` qui NE bouge PAS (`replay`, `filmcache`, `damagetag`,
`medalname`) — 11 outils, aucune action : `backfill_t0_film`, `diag_deaths`, `fetch_film_chunks`,
`mapcallouts-build`, `mapfond-inventaire`, `mapfond-webp`, `mapopads-build`, `replay-build`,
`replay-corpus-gate`, `replay-worker`, `weapon-icons-build`. (Sept outils figurent dans les deux
listes.)

Consommateurs INTERNES hors `film/`, production (mesure `grep -rl` par import) :

| Paquet deplace | Consommateurs de production hors `film/` |
|---|---|
| `analysis/filmsource` | `analysis/objectiveevents`, `replaybuild`, `sync/haloclient`, `sync/killcollector` |
| `film/filmdec` | `games/halo_infinite/ingest`, `ops`, `replaybuild`, `service`, `sync/haloclient`, `sync/killcollector` |
| `film/killsource` | `games/halo_infinite/replayidentity`, `ops`, `replaybuild`, `sync/killcollector` |
| `analysis/objectiveevents` | `replaybuild`, `sync`, `sync/replayartifacts` |

Consommateurs de TEST hors `film/` (a re-pointer aussi, sinon la compilation des tests casse) :
`analysis`, `analysis/filmsource`, `api/wire`, `archlint`, `games/halo_infinite/ingest`, `ops`,
`persist`, `platform/duckdb`, `replaybuild`, `service`, `sync/killcollector`.

**Ce que la facade exportee doit porter, chiffre** : 49 + 33 + 7 + 38 = **127 symboles exportes**
sont consommes hors de `film/` aujourd'hui. C'est le cout du principe 11 (« la compilation seule
garantit la frontiere ») : soit `film/` re-exporte ces 127 symboles, soit le lot en supprime une
part en portant les appelants. Les 38 d'`objectiveevents` et les 33 de `killsource` sont
concentres (`replaybuild`, `sync/killcollector`, `sync/replayartifacts`, `ops`) ; les 49 de
`filmdec` le sont moins (6 paquets internes + 8 outils).

### 2.5 (d) Esquisse du ratchet `archlint/film_layers_deps_test.go`

```
COUCHES ET REGLES D'IMPORT (analyse AST des imports, tests COMPRIS) :

  chemin                                        peut importer
  --------------------------------------------  -----------------------------------------------
  film/internal/source                          RIEN du depot (ratchet existant a re-pointer :
                                                archlint/filmsource_leaf_test.go)
  film/internal/profile                         source, games/halo_infinite/filmprofile,
                                                games/mappings, domain/title (catalogue)
  film/internal/grammar                         source, profile
  film/internal/facts                           source, profile, grammar,
                                                film/internal/facts/fallback, domain
  film/replay                                   facts, games/canonical, observability,
                                                domain/replaydoc
  --------------------------------------------  -----------------------------------------------
  INTERDIT DANS TOUS LES SENS : l'inverse de chacune de ces lignes.
  INTERDIT : tout paquet de film/ important internal/analysis (racine) ou internal/analysis/*.
  INTERDIT : quiconque hors de source ne lit chunk[i][j] (delegue au ratchet 2.4.3, cite ici).

PLANCHER : nombre minimal de fichiers .go parcourus PAR COUCHE (une couche vide = echec bruyant).

MUTATIONS QUI DOIVENT LE FAIRE ROUGIR :
  - ajouter un import de grammar dans profile ;
  - ajouter un import de facts dans grammar ;
  - ajouter un import de internal/analysis dans facts.
```

La moitie du travail est faite par le COMPILATEUR : `film/internal/...` est inaccessible depuis
l'exterieur de `film/`. Le ratchet ne garde que ce que le compilateur ne sait pas dire : les
dependances entre couches SOEURS (ADR 0030, « the compiler first, ratchets second »).

### 2.6 (e) Les ratchets existants a re-pointer

Occurrences de chemins de paquet en dur, mesurees par `grep` dans `internal/archlint/` :

| Ratchet | Occurrences | Ce qu'il faut faire |
|---|---:|---|
| `film_file_size_test.go` | 31 | `racinesSurveilleesTaille` cite `internal/games/halo_infinite/film`, `internal/replaybuild`, `internal/sync/killcollector`, `internal/analysis/objectiveevents`. Retirer la 4e (elle entre dans la 1re) et RE-MESURER `plancherFichiersBalayesTaille` |
| `no_film_reread_test.go` | 13 | deux listes : `paquetsSansInflate` (4 entrees) et `paquetsDeProduction` (7 entrees), plus un chemin en dur l. 169 |
| `decode_lock_held_test.go` | 8 | `paquetsAttendus` (4 entrees) + `verrouDuDecodeur`. **INVERSE par 2.3.2** : si 2.3 le SUPPRIME, 2.5 n'a rien a y faire ; s'il l'inverse, 2.5 re-pointe |
| `no_recomputed_film_context_test.go` | 5 | `paquetsSansAnalyseDeRegistre` (6 entrees) + chemin en dur l. 122 |
| `no_title_package_in_analysis_test.go` | 7 | allowlist D9, collee ci-dessous |
| `filmsource_leaf_test.go` | 2 | `const filmsourceLeafPkg = "internal/analysis/filmsource"` -> `internal/games/halo_infinite/film/internal/source` |
| `filmdec_package_vars_test.go` | 2 | chemin en dur l. 172 (touche AUSSI par 2.3 : `filmdecVarsGeles` 43 -> 0) |
| `no_unregistered_fallback_test.go` | 8 | `pkgKillsource` et voisins, dont une ancre citant `eventbody.go` `func evBody(r *evReader, code int, gate15 bool) bool {` — **cette ancre casse en 2.4**, pas en 2.5 |
| `no_unregistered_fallback_sites_test.go` | 2 | idem |
| `gamefiles_tag_test.go` | 0 chemin de paquet | **RIEN A FAIRE** : les racines sont DERIVEES du systeme de fichiers depuis le 2026-09-06 (« la liste des racines est desormais DERIVEE... une racine neuve est couverte le jour ou »). Le plan le cite a re-pointer : c'est perime, a statuer `[~]` |
| autres (`no_life_cause_divergence`, `no_french_label_literal`, `tactical_pure`, `no_repo_root_walk`, `no_raw_headshot_category_literal`, `no_identity_bridge_outside_registry`, `no_analysis_type_in_http_body`, `no_unbounded_film_loop`, `no_stale_fallback_target`, `no_rewritten_slot_band`, `no_raw_kill_scope_literal`, `no_player_index_identity`, `no_objective_submode_list`, `no_local_longest_run`) | 1 a 3 chacun | citations ponctuelles, a corriger au `git grep` de fin de lot |

Hors `archlint`, deux ratchets a re-pointer aussi :
`filmdec/grammar_rev_fingerprint_test.go` (fonction `racinesGrammaire`) et
`sync/killcollector/decoder_rev_fingerprint_test.go` (l. 128-129) — voir §3.1.

#### Allowlist D9 collee (`franchissementsToleres`, 5 entrees)

```go
var franchissementsToleres = map[string]string{
	"internal/analysis/objectiveevents/assaut_footer_research_test.go": "2026-09-12 — " +
		"`games/halo_infinite/film/filmcache` : test de RECHERCHE (pied de paquet du mode " +
		"Assaut) qui ouvre des films réels du cache local. La dépendance est au CACHE DE " +
		"FILMS d'un titre, pas à l'algorithme : le portage consiste à faire passer le film " +
		"par un paramètre (fixture ou interface de source), comme le fait déjà " +
		"`analysis/filmsource`. Hors périmètre du lot E (déplacement pur).",
	"internal/analysis/objectiveevents/extract_test.go": "2026-09-12 — " +
		"`games/halo_infinite/film/filmcache` : même motif que ci-dessus (extraction des " +
		"événements d'objectif vérifiée sur films réels). Même portage attendu : la source " +
		"du film devient un paramètre du test.",
	"internal/analysis/filmsource/source_test.go": "2026-09-12, rendu visible par le " +
		"déplacement du décodeur (commit E.2) — `games/halo_infinite/film/filmdec` : le test " +
		"EXTERNE de `filmsource` compare les deux marcheurs de paquets sur un film réel " +
		"(preuve d'équivalence de la grammaire, cf. `filmsource_leaf_test.go`). Le paquet " +
		"testé, lui, reste une feuille sans aucun import du dépôt : c'est le TEST qui " +
		"franchit. Portage attendu : la preuve d'équivalence descend avec le décodeur, sous " +
		"`games/halo_infinite/film/`.",
	"internal/analysis/sessionusage/usage_outcomes.go": "2026-09-12, rendu visible par le " +
		"déplacement du décodeur (commit E.2) — `games/halo_infinite/film/replay` : SEUL " +
		"franchissement de PRODUCTION de la liste. `sessionusage` lit les types de sortie " +
		"d'usage d'équipement produits par le décodeur. Portage attendu : ces types " +
		"remontent en `domain/` (ou `games/canonical/`), comme `domain/replaydoc` l'a déjà " +
		"fait pour le document de rejeu au lot A — après quoi `sessionusage` n'importera " +
		"plus rien d'un titre.",
	"internal/analysis/weapon_index_equivalence_test.go": "2026-09-12, rendu visible par le " +
		"déplacement du décodeur (commit E.2) — `games/halo_infinite/film/filmdec` : test " +
		"d'équivalence entre l'index d'armes d'`analysis` et celui du décodeur. Portage " +
		"attendu : l'index d'armes est title-agnostic (`games/weapons`), la comparaison " +
		"descend côté décodeur.",
}
```

**L'ADR 0034 D-1 se trompe d'un chiffre** : il ecrit « the last one's single dated allowlist
entry (`analysis/sessionusage/usage_outcomes.go`) empties at step 2.5.e ». L'allowlist en porte
**cinq**. `sessionusage/usage_outcomes.go` est le seul franchissement de PRODUCTION ; les quatre
autres sont des tests. Consequence pratique : trois des cinq (les deux d'`objectiveevents`, celui
de `filmsource`) tombent MECANIQUEMENT par le deplacement, parce que le fichier quitte
`internal/analysis/`. Les deux qui demandent un vrai portage sont `sessionusage/usage_outcomes.go`
(4 symboles de `replay`) et `weapon_index_equivalence_test.go`. Le test refuse en outre toute
entree devenue sans objet (« franchissementsToleres cite %q, qui n'importe plus aucun paquet de
titre — retirer l'entree ») : **les entrees doivent partir dans le MEME commit que le
deplacement**, sans quoi le ratchet rougit.

Garde-fou a ne pas oublier : `plancherFichiersAnalysis = 300` (« 1 337 avant le deplacement du
decodeur, 399 apres »). Retirer `objectiveevents` (62 fichiers `.go`, tests compris) et
`filmsource` (5) d'`internal/analysis/` fait passer le compte a environ **332** — au-dessus du
plancher, mais de peu. Le lot doit RE-MESURER et, si le compte descend sous 300, baisser le
plancher dans le meme commit, avec sa mesure datee.

### 2.7 (f) Un ordre de commits qui compile a chaque pas

Methode du lot E : **ratchets de dependance poses AVANT le premier `git mv`** ; `git mv` sans
ajout ni suppression de ligne hors `package` et imports ; une couche par commit ; gate a chaque
couche ; `git diff --stat -M` ne doit montrer que des renommages.

| # | Commit | Contenu | Compile ? | Gate |
|---:|---|---|---|---|
| 0 | `test(2.5.0)` | `archlint/film_layers_deps_test.go` pose, couches CIBLES declarees mais TOLERANTES sur ce qui n'existe pas encore (une couche absente = skip explicite, jamais un vert muet) | oui | `go test ./internal/archlint/` |
| 1 | `refactor(2.5.a)` | `filmsource` -> couche `source` + remontee des 14 fichiers de `filmdec` du §2.3 + `Film.Identity`. Re-pointe `filmsource_leaf_test.go`, 3 entrees D9, `plancherFichiersAnalysis` | oui | `go build ./...`, `replay-equiv` court |
| 2 | `refactor(2.5.b)` | les 7 fichiers de profil -> couche `profile` | oui | idem |
| 3 | `refactor(2.5.c)` | `filmdec` -> couche `grammar` (119 fichiers prod restants + leurs tests : 142 moins les 14 de `source`, les 7 de `profile` et les 2 supprimes par 2.3). Re-pointe `filmdec_package_vars_test.go`, `no_recomputed_film_context_test.go`, `grammar_rev_fingerprint_test.go` | oui | `replay-equiv` court |
| 4 | `refactor(2.5.d.1)` | `killsource` -> `facts/killsource` + `replay/fallback` -> `facts/fallback`. Re-pointe `decoder_rev_fingerprint_test.go`, `no_unregistered_fallback*_test.go` | oui | idem |
| 5 | `refactor(2.5.d.2 + 2.5.e)` | `objectiveevents` -> `facts/objectives` **ET** facade exportee dans le MEME commit (127 symboles a arbitrer), re-pointage des 12 outils `cmd/` et des 11 paquets internes, portage de `sessionusage`, **allowlist D9 VIDEE** | oui | `go build ./...`, `go vet`, **regime COMPLET** (2.5 est marque risque par V2) |
| 6 | `test(2.5.0 bis)` | le ratchet de couches passe de tolerant a STRICT ; les 3 mutations de preuve du §2.5 | oui | `go test ./internal/archlint/` |

**Le piege d'ordonnancement, ecrit pour qu'il ne se decouvre pas a l'execution** : deplacer un
paquet sous `film/internal/` le rend inaccessible a ses consommateurs EXTERIEURS dans le meme
commit. `filmsource` (4 consommateurs internes hors `film/` + 3 outils), `filmdec` (6 + 8),
`killsource` (4 + 1), `objectiveevents` (3 + 5) : chacun de ces quatre `git mv` casse la
compilation tant que la facade n'existe pas. Trois issues, a trancher :

- **(a) Recommandee** : deplacer d'abord vers `film/<couche>` (EXPORTE, pas `internal/`), puis un
  dernier commit bascule `film/<couche>` -> `film/internal/<couche>` une fois la facade posee.
  Chaque pas compile, les `git mv` restent purs, et le dernier commit est un renommage de
  repertoire plus la facade ;
- (b) poser la facade AVANT tout `git mv` (elle delegue alors a `analysis/filmsource` etc.), puis
  deplacer. Cout : la facade change de corps deux fois ;
- (c) un seul commit geant. A ecarter : il annule la methode du lot E.

---

## 3. Lot 2.6 (pas 6) — Empreintes par couche et types de contrat

### 3.1 (a) Ce que les deux empreintes hachent aujourd'hui

#### `GrammarRev` — `filmdec/grammar_rev.go:413`, golden `filmdec/testdata/grammar_rev.golden`

Valeur : `grammar-2026-09-15.26`. Empreinte figee :
`ee09b3833c3f2264d5d08f46b4c2651c290724e75470b6fe74c04b581057b289`.

Racines hachees (`racinesGrammaire`, resolues par `runtime.Caller`, collees) :

```go
func racinesGrammaire(t *testing.T) []string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	// .../internal/games/halo_infinite/film/filmdec -> .../internal
	internalDir := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(ici)))))
	filmDir := filepath.Join(internalDir, "games", "halo_infinite", "film")
	return []string{
		filepath.Join(filmDir, "filmdec"),
		filepath.Join(filmDir, "killsource"),
		filepath.Join(internalDir, "analysis", "objectiveevents"),
	}
}
```

Ce qui est hache : toutes les sources `.go` **non-test** des trois racines ; `testdata/` ecarte ;
`grammar_rev.go` lui-meme EXCLU (`fichierHorsGrammaire`) ; fins de ligne normalisees en LF ;
chemin relatif hache a cote du contenu (un renommage bouge donc l'empreinte). Deux tests :
`TestGrammarRevSuitLaGrammaire` (couple revision / empreinte) et
`TestChroniqueCouvreLaRevisionCourante` (la revision courante a son entree dans le godoc ET dans
l'HISTORIQUE du golden).

**Le godoc de ce test prevoit deja 2.5** : « Ce paquet DEMENAGERA sous `film/` au pas 5... Le jour
ou il bougera, `racinesGrammaire` echouera bruyamment sur un dossier vide — ce qui est exactement
le comportement voulu ».

#### `KillSourceDecoderRev` — `killcollector/killsource_decoder_rev.go:128`

Valeur : `killsource-2026-09-16.2`. Racine hachee (`decoder_rev_fingerprint_test.go:128-129`) :

```go
racine := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(ici))))
return filepath.Join(racine, "internal", "games", "halo_infinite", "film", "killsource")
```

Une seule racine : `film/killsource/`, sources non-test. La constante vit dans
`sync/killcollector/` (pas dans le paquet hache) : le deplacement pur du 2026-09-16 l'a laissee
verte sans regeneration, et c'est teste.

**Le defaut que ces deux empreintes laissent passer, deja consigne dans le code** (en-tete de
`killsource_decoder_rev.go`) : « l'empreinte ci-dessous ne hache que `killsource/`. Ce correctif
a commence dans `internal/analysis/` (parseur) et dans `filmdec` — il n'aurait PAS fait sonner le
gate ». Autrement dit : `analysis/highlight_event_parser.go` et `analysis/weapon_scanner.go`, qui
sont de la grammaire de film, ne sont dans AUCUNE des deux empreintes.

#### Proposition : `grammar.Rev` et `facts.Rev`

| Revision | Vit dans | Hache (racines) | Monte quand | Consequence |
|---|---|---|---|---|
| `source.Rev` (facultative, cf. Q11) | `film/internal/source/rev.go` | `film/internal/source/` | le decoupage en chunks / paquets, l'inflate ou le lecteur de bits change | tout re-decode ; c'est la racine des deux autres |
| `profile.Rev` (facultative, cf. Q11) | `film/internal/profile/rev.go` | `film/internal/profile/` | une ligne de la table du profil change | idem |
| **`grammar.Rev`** (heritiere de `GrammarRev`) | `film/internal/grammar/rev.go` | `film/internal/grammar/`, **plus `profile/` et `source/`** si les deux precedentes n'existent pas | une largeur, un cadre, un ordre de composants, un lecteur neuf | `TestGrammarRevSuitLaGrammaire` rouge sans montee |
| **`facts.Rev`** (heritiere de `KillSourceDecoderRev`) | `film/internal/facts/rev.go` | `film/internal/facts/` (tout l'arbre : `killsource/`, `objectives/`, `fallback/`) **plus la VALEUR de `grammar.Rev`** | la SORTIE des faits peut changer | **backlog killsource** : les lignes de kill en base portent cette revision et redeviennent candidates |

Deux points de conception a ne pas manquer :

1. **`facts.Rev` doit hacher la VALEUR de `grammar.Rev`, pas seulement ses propres sources.** Le
   defaut consigne ci-dessus est exactement celui-la : une correction de grammaire qui change la
   sortie de `killsource` sans toucher un octet de `killsource/` ne faisait pas monter
   `KillSourceDecoderRev`. Hacher la chaine `grammar.Rev` dans l'empreinte de `facts` le rend
   mecanique. C'est plus strict qu'aujourd'hui, et c'est le comportement voulu : un faux positif
   coute une ligne, un faux negatif coute un parc de lignes fausses en base.
2. **La constante `facts.Rev` doit DESCENDRE dans `facts/`**, alors qu'elle vit aujourd'hui dans
   `sync/killcollector/`. Le paquet qui la publie sur chaque ligne (`killcollector`) la LIT ; il
   ne la porte plus. Cela ferme aussi le trou du 2026-09-16 (un deplacement pur de la constante
   laissait l'empreinte verte sans regeneration).

#### Ou vit la regle « montee de `facts.Rev` = backlog killsource »

- **Dans le test** : `TestFactsRevSuitLesFaits` (heritier de
  `TestKillSourceDecoderRevSuitLeDecodeur`), dont le message d'echec dit la consequence, comme le
  fait deja `TestGrammarRevSuitLaGrammaire` (« Se demander AUSSI : la sortie de killsource
  peut-elle changer (KillSourceDecoderRev, backlog de redecodage) ? »).
- **Dans `docs/SYNC_GUIDE`, FR et EN.** Mesure : les deux fichiers font **171 lignes chacun** et
  sont **alignes ligne pour ligne** (memes numeros de titre). Aucun des deux ne contient
  aujourd'hui `killsource`, ni `KillSourceDecoderRev`, ni `backlog` : la regle est a **AJOUTER**,
  pas a modifier. Sections cibles :

  | Fichier | Ligne | Titre | Ce qu'on y ajoute |
  |---|---:|---|---|
  | `docs/SYNC_GUIDE.md` | 115 | `### Backfill (local recomputes & API backfills)` | une sous-section `#### Decoder revisions and the killsource backlog`, apres le tableau des selecteurs (l. 137) |
  | `docs/fr/SYNC_GUIDE.md` | 115 | `### Backfill (recalculs locaux & backfills API)` | son miroir exact, `#### Revisions du decodeur et backlog killsource` |

  Contenu attendu, trois phrases : chaque ligne de `shared.kill_events` porte la revision des
  faits qui l'a produite ; une montee de `facts.Rev` rend candidates au redecodage toutes les
  lignes portant une revision anterieure (`conditionBacklog`,
  `sync/killcollector/postsync.go:401`) ; le redecodage part sur **signal utilisateur** (D6),
  jamais automatiquement.

  Alternative a ecarter : la section `## Sync Pipeline V2` (l. 68) decrit le cycle, pas les
  recalculs — le backlog killsource est un recalcul.

### 3.2 (b) Les types qui traversent les couches, et le paquet `film/internal/types`

Mesure : intersection entre les types EXPORTES declares par un paquet (`^type [A-Z]`) et les
identifiants `<paquet>.<X>` employes par les consommateurs, production seule.

| Producteur (couche) | Types exportes | Consommateur | Types qui traversent |
|---|---:|---|---:|
| `filmdec` (grammar) | 148 | `film/replay` | **53** |
| `filmdec` (grammar) | — | `replaybuild` + `sync/killcollector` | **10** (dont 3 nouveaux) |
| `killsource` (facts) | 25 | `replay` + `replaybuild` + `killcollector` + `ops` | **6** |
| `objectiveevents` (facts) | 25 | `replay` + `replaybuild` + `sync*` | **14** |
| `filmsource` (source) | 5 | tout le reste | **5** |

**Les 53 types de `filmdec` consommes par `film/replay`** (liste collee) :

```
AbilityCharge AbilityChargeStats AbilityImpulse AbilityImpulseStats AbilityRank AbilityRankStats
BipedAim BipedCreation BipedCreationStats BipedPickup BipedPickupStats BipedPosition CamoRead
CarrierMarkScan EquipmentChange EquipmentChangeStats EquipmentCreation EquipmentCreationStats
EquipmentLifeKey EquipmentPlacement EquipmentPlacementStats EquipmentSpawnEvent EquipmentSpawnStats
FilmContext FireEvent GrappleRead GrenadeThrow GroundWeaponAmmo HeldWeaponChange HeldWeaponChangeKind
I0Layout InventoryDelta KeyframeLoadout MPPWidths ManagedPropertyRead MapQuantCatalog MapQuantEntry
NavpointRadialRead NavpointRadialScan NavpointSegment ObjectDeath ObjectDeathStats PlayerSlot Profile
ProjectileSample ProjectileTrack ScanFilmOptions TeamScanReport TranslocatorTeleport Vec3Range
VehicleEvent WorldObjectKeyframes ZoomEvent
```

**Les 10 types de `filmdec` consommes par `replaybuild` + `sync/killcollector`** :

```
BipedCreation BipedPosition FilmContext FireEvent MapQuantCatalog MapQuantEntry ScanFilmOptions
WeaponDamage WeaponHitDistanceFunc WeaponHitStats
```

**Les 6 types de `killsource`** : `ApparStats`, `Assist`, `CoupleStats`, `FilmTablePinning`,
`Kill`, `Result`.

**Les 14 types d'`objectiveevents`** : `DeathInstant`, `FlagFilmSignals`, `FlagGrabsNetPlayer`,
`FlagSpan`, `FlagTrack`, `IdentifiedEvent`, `NamedEvent`, `PlayerLine`, `RoundBounds`,
`RoundIdentity`, `RoundsDecision`, `ScorePoint`, `StatComponent`, `StatRecord`.

**Les 5 types de `filmsource`** : `ChunkMeta`, `Film`, `MemoryChunks`, `Packet`, `Source`.

Union dedupliquee : **56 + 6 + 14 + 5 = 81 types traversent une frontiere de couche**, sur 203
types exportes par les quatre paquets (148 + 25 + 25 + 5). Soit **40 %**.

#### Proposition de contenu pour `film/internal/types`

Le paquet doit etre une FEUILLE (aucun import du depot), a la maniere de `games/canonical` et de
`domain/replaydoc` (ce dernier : 381 L, zero import du depot hors `time`). Regle proposee — et ce
n'est pas « y mettre les 81 » :

- **Y entrent** les types de DONNEES purs, sans methode qui decode : les 53 de `filmdec -> replay`
  moins `FilmContext`, `Profile`, `MapQuantCatalog`, `ScanFilmOptions` (objets de service, pas des
  faits), soit **49** ; les 6 de `killsource` ; les 14 d'`objectiveevents` ; `ChunkMeta` et
  `Packet` de `filmsource`. Ordre de grandeur : **70 types**.
- **N'y entrent PAS** : `FilmContext` (porte un memo et un profil : il appartient a `grammar`),
  `Profile` (appartient a `profile`), `MapQuantCatalog` (catalogue : `profile`),
  `ScanFilmOptions` (option d'appel : `grammar`), `Film` / `Source` / `MemoryChunks` (`source`),
  `WeaponHitDistanceFunc` (une FONCTION, pas un type de donnee).

**Un test de contrat par type, sur le modele de `document_shape_test.go`.** Ce que ce modele
apporte, mesure sur pieces (`film/replay/document_shape_test.go`, golden de 989 lignes) :

1. la forme est figee **en clair** dans un golden versionne, a cote de la version qui l'a gelee
   (`TestDocumentShapeMatchesGolden`, l. 59) ;
2. la regeneration est une **porte unique et bruyante** (`TestDocumentShapeRegenerate`, l. 124 :
   elle exige `-update` ET une variable d'environnement, et `t.Fatalf` meme quand elle reussit —
   « sans ce refus, la reponse naturelle a un golden rouge serait de le regenerer ») ;
3. le golden doit porter la version COURANTE (`...GoldenCarriesCurrentSchema`, l. 91) ;
4. la version courante doit avoir une entree de chronique (`...SchemaHasChronicleEntry`, l. 101) ;
5. les DEUX jumeaux (stocke et servi) doivent coincider (`...TwinsAgree`, l. 76).

Transposition proposee : **UN golden `types/testdata/shapes.golden`** (une section par type : nom,
champs, types, tags JSON s'il y en a), fige a cote de `grammar.Rev` / `facts.Rev`, avec une porte
de regeneration unique. Un golden par type ferait 70 fichiers pour une seule question ; le
document de rejeu, lui, en a UN seul pour tout un arbre de 989 lignes. Les points 3 et 4 se
transposent : le golden porte la revision de la couche qui produit le type, et une mutation de
forme sans montee de revision rougit.

### 3.3 (c) Ou l'artefact publie `coverage.*`, et ce qu'ajoute 2.6.3

#### L'existant, mesure

| Role | Fichier | Type / ligne |
|---|---|---|
| Producteur | `film/replay/coverage.go` (466 L) | `type Coverage struct` — 37 champs, dont 34 blocs `*XxxCoverage` en `omitempty` |
| Champ dans le document | `film/replay/document.go:429` | `Coverage *Coverage` (tag JSON `coverage,omitempty`) |
| Version de schema | `film/replay/document.go:48` | `const SchemaVersion = 60` |
| Jumeau servi | `domain/replaydoc/coverage.go` (381 L) | `type Coverage struct` — meme forme, ordre des champs different |
| Conversion | `service/replayview/convert_coverage.go` | la seule traduction producteur -> servi |
| Empreinte de forme | `film/replay/testdata/document_shape.golden` (989 L) | 3 occurrences de `coverage` : l. 95 (`BombStatsCoverage`), 436 (`IdentityCoverage`), 639 (`*Coverage`) |
| Chronique | `film/replay/document_chronicle.go:1506` | `// v60 (2026-09-17, vague 2 de la famille 1.9 ...)` |
| Contrat HTTP | `api/openapi.yaml` | 13 occurrences de `coverage` — **regeneration obligatoire** (`make openapi-gen` puis `make generate-types`, gate `TestOpenAPIYAMLIsUpToDate`) |

Precedent utile, deja dans le fichier : `Coverage.FilmMajorVersion *int` a ete ajoute en
telemetrie pure, et son godoc explique le choix du pointeur (« son absence dit : film sans
registre, ou artefact anterieur a ce lot — d'ou le pointeur plutot qu'un zero qui ressemblerait a
une version »). `coverage.decoder` est exactement la meme famille, en plus complet.

#### Ce qu'ajoute 2.6.3

```go
// DecoderCoverage dit SOUS QUELLES REVISIONS cet artefact a ete cuit.
type DecoderCoverage struct {
    GrammarRev string `json:"grammarRev"`
    FactsRev   string `json:"factsRev"`
    Build      string `json:"build"`      // le build lu en clair dans chunk_00 section 2
}

// dans Coverage :
    Decoder *DecoderCoverage `json:"decoder,omitempty"`
```

Pourquoi un pointeur et `omitempty` : meme raison que `FilmMajorVersion` — l'absence distingue
« artefact anterieur a ce lot » de « revision inconnue ». `Build` est une chaine et non un entier :
c'est la cle du profil (ADR 0034 D-3, « the build, read in clear text in `chunk_00` section 2 »),
et un film dont le build est inconnu porte `ErrUnknownBuild` — le champ doit alors etre vide, pas
zero.

Le champ existe des M2 mais ses valeurs deviennent **exploitables** a M4 (D-7 : « the document
carries a revision per layer... the presence of a layer is read in its revision, never in the
absence of a field »). 2.6.3 est donc la premiere moitie de 4.2.1, posee tot pour que la
recuisson selective par couche (4.4) ait de quoi decider.

#### La montee de `SchemaVersion` : 60 -> 61 (le plan dit 57, il est perime)

Le plan ecrit « montee de `SchemaVersion` 57 par l'empreinte de forme ». **La constante vaut 60**
sur la base `1e246b209` ; le nombre 57 date de la redaction du plan (2026-09-13), avant les
vagues de la famille 1.9. La montee est **60 -> 61**, et le plan doit etre corrige dans le meme
commit.

Checklist du commit (deduite du §2.3 « Gates communs » du plan et de la chronique) :

1. `film/replay/coverage.go` : `DecoderCoverage` + le champ `Decoder` ;
2. `domain/replaydoc/coverage.go` : le jumeau ;
3. `service/replayview/convert_coverage.go` : la conversion ;
4. `film/replay/document.go:48` : `SchemaVersion = 61` ;
5. `film/replay/document_chronicle.go` : **l'entree v61, ecrite dans CE commit** — ADR 0034,
   correction 2 : « A chronicle entry is written in the commit that raises the version, never
   afterwards » ; le trou de v51 etait reste invisible quatre jours ;
6. `film/replay/testdata/document_shape.golden` regenere par sa porte unique ;
7. `api/openapi.yaml` regenere en DERNIER (`make openapi-gen`, `make generate-types`), gate
   `go test ./internal/api/ -run TestOpenAPIYAMLIsUpToDate -count=1` (CGO) ;
8. `service/replayview/parity_test.go` verifie champ par champ (D-7).

Entree de chronique attendue, sur la forme des precedentes :

```
// v61 (2026-09-17, lot 2.6 du PLAN_DECODEUR_FILM) : COVERAGE.DECODER — l'artefact dit
// desormais SOUS QUELLES REVISIONS il a ete cuit : `grammarRev` (la grammaire de lecture),
// `factsRev` (la couche des faits, celle qui commande le backlog killsource) et `build`
// (la cle du profil, lue en clair dans chunk_00 section 2). TELEMETRIE PURE : aucun rendu
// n'en depend, aucun octet cuit ne change par ailleurs, et l'ABSENCE du bloc dit
// « artefact anterieur a ce lot ». La montee ne tient qu'a l'empreinte de FORME
// (document_shape.golden) : c'est un champ ajoute, pas un contenu modifie. Pas de
// recuisson requise ; les artefacts deja cuits restent servis tels quels et leur bloc
// `decoder` reste absent jusqu'a leur prochaine cuisson.
```

**C'est le SEUL changement de contenu de tout M2, et il doit etre le DERNIER commit du jalon.**
Raison mesurable : D4 exige zero difference d'equivalence a chaque pas structurel ; ce commit-ci
en produit une par construction, et la preuve attendue est « difference limitee au champ
`coverage.decoder` ». Le placer ailleurs qu'en derniere position melangerait cette difference
voulue avec celles, non voulues, des pas suivants.

---

## 4. Ordre conseille et duree estimee par lot

| Rang | Lot | Depend de | Regime de gate (V2) | Taille mesuree | Duree estimee |
|---:|---|---|---|---|---|
| 1 | **2.3** (en cours ailleurs) | 2.2 | court | 43 variables -> 0, 6 sites de verrou | — |
| 2 | **2.4** | 2.3 **fusionne** | court + `TestKillSourceDecoderRevSuitLeDecodeur` | 6 lecteurs a fusionner en 1, environ 120 sites, 2 marcheurs de paquets a fusionner | **M vers L** — 1 a 1,5 jour. Le plan dit M ; la mesure (6 lecteurs, pas 2) pousse vers L si l'option (A) du §1.3 est retenue |
| 3 | **2.5** | 2.4 | **COMPLET** (lot marque risque) | 660 fichiers, 6 a 7 commits, 12 outils `cmd/` + 11 paquets internes, 127 symboles de facade, 5 entrees D9, 10 ratchets a re-pointer | **L** — 2 a 3 jours. C'est le lot le plus long du jalon |
| 4 | **2.6** | 2.5 | court, sauf le dernier commit | 2 revisions -> 4, environ 70 types de contrat, 1 champ publie, 8 fichiers a la montee de schema | **M** — 1 jour |

Duree totale estimee de 2.4 + 2.5 + 2.6 : **4 a 5 jours-agent**, sequentiels (un seul muteur de
`filmdec` / `killsource` / `replay` par la regle V12).

**Ce qui peut etre prepare en parallele, sans muter les paquets du film** (aucun conflit de
fichier avec 2.4 / 2.5 / 2.6) :

- le ratchet `archlint/film_layers_deps_test.go` (item 2.5.0) : il ne touche que `archlint/` ;
- le ratchet `archlint/no_raw_film_bytes_outside_source_test.go` (2.4.3), idem ;
- les deux sous-sections de `docs/SYNC_GUIDE` FR + EN (2.6.1) ;
- l'entree de chronique v61 en brouillon.

**Ordre a NE PAS suivre** : 2.6 avant 2.5. `grammar.Rev` et `facts.Rev` sont des empreintes PAR
COUCHE ; les couches n'existent pas avant 2.5, et poser les revisions d'abord obligerait a les
re-pointer aussitot.

---

## 5. Ce qui depend de 2.3

| Ce qui depend | Pourquoi, mesure | Si 2.3 n'est pas fusionne |
|---|---|---|
| **2.4, en entier** | `filmdec/bitreader.go` : 2.3 vide le constructeur de ses 5 champs de profil herite, 2.4 deplace le lecteur. Meme fichier, meme fonction | conflit frontal ; l'executeur de 2.4 doit rouvrir `bitreader.go` sur pieces a l'entree du lot |
| **L'absorption de `evReader` (2.4.1)** | Construire un `BitReader` la ou vivait un `evReader` fait entrer le profil herite dans un chemin qui ne l'avait pas. La calibration de `killsource` (`calibrate.go:114` `PoserMouvementHerite`, l. 167 et 176 `SetRecordStateParam`) FUIT vers `replay.BuildFromFilm` par l'etat du processus (D1 (2.2)) | l'equivalence bit a bit pourrait passer alors que la cuisson du rejeu a change |
| **La suppression de `film_packets.go` (2.4)** | Il n'est plus atteignable que par le pont `film_chunks.go`, que 2.3 ne touche pas | independant |
| **`archlint/decode_lock_held_test.go`** | 2.3.2 l'INVERSE (toute prise de verrou interdite) ; 2.5 devait le re-pointer | si 2.3 le SUPPRIME plutot que l'inverser, 2.5 n'a rien a y faire : a verifier a l'entree |
| **`archlint/filmdec_package_vars_test.go`** | 2.3 le descend de 43 a 0 ; 2.5 change son chemin en dur (l. 172) | deux lots sur le meme fichier de ratchet, a 2 lignes de distance : sans risque, mais a rouvrir |
| **`filmdec/profil_herite.go` et `decode_gate.go`** | Ils DISPARAISSENT en 2.3, donc ils ne figurent dans aucune couche cible de 2.5 | si 2.3 glisse, 2.5 doit leur donner une couche (`grammar` par defaut) et la retirer ensuite |
| **Le ratchet 2.4.3 (allowlist vide)** | `LockProcessDecode` existe encore dans `cmd/rdata_weapon_scan/main.go:713`, qui est aussi un site de lecture brute (l. 257) | deux lots sur le meme fichier |
| **`TestDeuxFilmsEnParallele` (2.3.3)** | Aucun des trois lots suivants n'en depend ; il prouve 2.3 | — |

**Ce qui ne depend PAS de 2.3, et peut commencer maintenant** : toute la partie `objectiveevents`
et `filmsource` de 2.5 (aucun de ces deux paquets n'a de variable de paquet ni de verrou), les
deux ratchets (2.4.3, 2.5.0), les deux sous-sections de `docs/SYNC_GUIDE`.

---

## 6. Questions ouvertes

Une decision par ligne. Les trois premieres bloquent l'ecriture de la premiere ligne de code
de 2.4.

1. **(2.4, architecture — BLOQUANTE)** La facade `film.Source` ne peut pas etre importee par
   `analysis/objectiveevents` ni `analysis/weaponv3` sans rougir D9. Option (i) recommandee : la
   facade nait dans `internal/analysis/filmsource/` et 2.5.a la deplace par `git mv` pur.
   Option (ii) : intervertir 2.4 et 2.5. Option (iii) : allowlister — a ecarter.
2. **(2.4, perimetre — BLOQUANTE)** L'allowlist du ratchet 2.4.3 est-elle vide au prix d'un
   perimetre elargi (option A : absorber aussi `analysis/highlight_event_parser.go` 396 L,
   `analysis/weapon_scanner.go` 377 L, `sync/killcollector/shots.go:76-224`), ou au prix d'un
   perimetre declare (option B : ces trois-la hors des racines surveillees, avec date et cible de
   retrait au pas 5) ? Recommandation : B.
3. **(2.4, grammaire — BLOQUANTE pour l'equivalence)** Le contrat de fin de flux : `BitReader`
   bourre a zero, `evReader` refuse et leve `over`. Le drapeau remonte-t-il chez l'appelant
   (recommande : `Remaining()` existe deja) ou le lecteur canonique gagne-t-il un mode `strict` ?
4. **(2.4 / 2.5, architecture)** `analysis.ParseHighlightEvents` (5 consommateurs de production)
   et `analysis.ScanFireEventsB5` / `FindFramePositions` / `ScanFormulaA(NS)` /
   `TimestampEstimator` sont de la grammaire de film dans un paquet title-agnostic.
   Descendent-ils en `grammar` / `facts` au pas 5 (ce qui casse les aretes 6 et 7 du §2.2 et vide
   un trou d'empreinte connu), ou restent-ils hors du decodeur ? Le plan ne les nomme nulle part.
5. **(2.5, D9)** L'ADR 0034 D-1 dit « single dated allowlist entry » ; il y en a **cinq**. Trois
   tombent mecaniquement par le deplacement. Les deux qui demandent un portage sont
   `sessionusage/usage_outcomes.go` (4 symboles de `replay` a remonter en `domain/` ou
   `games/canonical/`) et `weapon_index_equivalence_test.go`. Le portage de `sessionusage` est-il
   dans le perimetre de 2.5.e, ou devient-il un item a lui ?
6. **(2.5, ordonnancement)** `git mv` vers `film/<couche>` exporte d'abord, puis bascule en
   `internal/` au dernier commit (recommande) — ou facade avant tout `git mv` ? Sans arbitrage,
   au moins quatre commits ne compilent pas.
7. **(2.5, perimetre)** La facade exportee doit-elle re-exporter les **127 symboles** consommes
   hors `film/` (49 `filmdec` + 33 `killsource` + 7 `filmsource` + 38 `objectiveevents`), ou le
   lot porte-t-il une partie des appelants pour reduire ce nombre ? Une facade de 127 symboles
   n'est pas une frontiere, c'est un alias.
8. **(2.5, classement)** `player_table.go` (420 L), `player_table_record.go` (183 L),
   `player_table_control.go` (120 L) : la LECTURE va-t-elle en `source` et le CONTROLE en
   `profile`, ou les trois en `grammar` ? Le plan cite un `player_table_profile.go` qui n'existe
   pas.
9. **(2.5, classement)** `objectiveevents/film.go` : `scanTh10Events` / `decodeTh10Block`
   (offsets 36, 37, 47, 48-51 d'un bloc de 60 o) sont de la grammaire au sens de `GrammarRev`.
   Bloc entier en `grammar`, ou bloc en `facts` avec ses lectures de bits deleguees a `source`
   (recommande) ?
10. **(2.5, effet de bord)** `filmdec/observateur.go` annonce que « les 29 crochets attendent le
    pas 5 » : 2.5 est-il autorise a leur donner leur parametre (ce qui n'est plus un deplacement
    pur), ou cela devient-il un item a part ?
11. **(2.6, granularite)** Combien de revisions ? Deux (`grammar.Rev` + `facts.Rev`, avec `source`
    et `profile` haches dans `grammar.Rev`), ou quatre (une par couche) ? Quatre est plus juste et
    coute trois goldens de plus.
12. **(2.6, stricte)** `facts.Rev` hache-t-elle la VALEUR de `grammar.Rev` ? Recommande oui :
    c'est la seule facon mecanique de fermer le trou consigne dans
    `killsource_decoder_rev.go` (« le gate couvre le decodeur, pas son amont »). Cout : toute
    montee de grammaire ouvre un backlog killsource, meme quand la sortie ne change pas.
13. **(2.6, contrat)** Un golden unique `types/testdata/shapes.golden` (recommande, modele
    `document_shape.golden` a 989 L) ou un fichier par type (environ 70) ?
14. **(2.6, chiffre)** Confirmer la correction du plan : `SchemaVersion` **60 -> 61**, pas 57.
15. **(2.6, contenu)** `coverage.decoder.build` : chaine vide quand `ErrUnknownBuild`, ou bloc
    `decoder` entierement absent ? Recommande : chaine vide et bloc present — l'absence doit
    signifier « artefact anterieur au lot », pas « build inconnu ».
16. **(transverse, D6)** La montee de `facts.Rev` en 2.6 ouvre mecaniquement le backlog
    killsource. Le jalon M2 se clot-il avec ce backlog OUVERT (a jouer sur signal utilisateur,
    D6), ou `facts.Rev` prend-elle la valeur de `KillSourceDecoderRev` pour eviter d'ouvrir un
    backlog qu'aucun changement de sortie ne justifie ? Recommande : reprendre la valeur — M2 est
    un jalon a zero difference de contenu, donc rien a redecoder.
