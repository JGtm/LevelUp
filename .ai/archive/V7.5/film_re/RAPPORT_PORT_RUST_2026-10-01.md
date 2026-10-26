# Port Rust du décodeur de film (S. Imbleau) face à LevelUp — analyse et expérience (2026-10-01)

> Analyse en LECTURE SEULE. LevelUp : `feat/v75` à `8b894a677` (aucune édition, aucun commit, aucune
> base ouverte, aucune cuisson). Port : clone `halo_api` en HEAD détachée sur
> `origin/simbleau/theater-experiments` = `0aaa05b` (2026-09-30, 33 commits depuis `main`).
> Expérience : crate construit en release dans le clone, exemple écrit dans le clone
> (`examples/levelup_closure.rs`), films lus en place, un à la fois.
>
> Conventions de preuve :
> - **[R]** = clone, chemin relatif à `halo_api/` ; **[G]** = LevelUp, chemin relatif à
>   `apps/go-api/internal/games/halo_infinite/film/` sauf mention ; `fichier:ligne`.
> - **[É]** = établi (lu dans le code, ou mesuré ici) ; **[S]** = supposé (inférence, extrapolation).

---

## 0. Synthèse

1. **[É] Le port est une architecture « canonique d'abord » propre** : une seule entrée
   `Film::parse` sans heuristique ([R] `src/theater/film/mod.rs:19`), un modèle qui garde chaque
   octet, des arrêts typés à chaque niveau, une partition de couverture par paquet
   (`Packet::source_coverage`, Fields/Opaque/Padding/Unparsed), puis une résolution privée
   (`TheaterRuntime::load`) où vivent les recherches étiquetées « dérivées ». C'est très proche de
   nos principes P1, P3, P7, P8, P10.
2. **[É] Il contredit P9 et P5** : tout le film est matérialisé (deux copies de chaque chunk,
   un `String` + un `Vec<u8>` par champ lu), l'état de décodage est privé. Mesuré : 230 à 416 Mio de
   pic pour 20 à 68 Mio de film, alors que le parseur ne traverse que **9,2 %** des bits des trames
   type 0.
3. **[É] Sur nos 10 témoins v41, la fermeture « au sens LevelUp » du port est de 5,5 %** des trames
   type 0 (18 587 / 335 960), contre **66,9 %** pour notre tête (224 818 / 335 960, mêmes paquets,
   mêmes dénominateurs). Ses « trois vues complètes » (71,9 %) sont à 92 % des faux positifs.
4. **La cause n'est pas la grammaire des composants** ([É] la vue B porte exactement les mêmes noms
   que nous : 544 lignes `porte` + 51 `partiel` de `ecs_table.tsv` connues du port, aucune des 472
   autres) **mais le contexte de schéma** ([S] attribution causale, [É] faits qui suivent) : sans balayage d'ancres, le port ne lie que 2 records
   par image-clé (632 au total), ne lie rien du bloc de type 1, n'a ni table anticipée ni
   localisation de liste. 68 % des fins de vue B (198 331 sur 291 210 trames décodées) sont alors des REJETS d'en-tête, et un paquet dont
   la vue B finit sur un rejet ne ferme presque jamais (28 sur 175 447).
5. **[É] Idée directement utile pour notre cause n° 1 (« vue C : terminateur hors cadre »)** :
   chez le port, 94,2 % des paquets « trois vues complètes mais non fermés » ont une vue C VIDE
   (un seul bit 0 lu), le plus souvent après une fin de vue B par rejet. Notre instrument range
   exactement ces cas sous « terminateur hors cadre » ([G] `internal/grammar/frame_infer.go:291-294`,
   `frame_harvest.go:333-349`, `frame_closure_classement.go:94-109`) sans distinguer la forme de la
   fin de vue B ni celle de la vue C. **[S]** Une part importante de nos 264 757 « terminateurs hors
   cadre » est probablement une désynchronisation silencieuse de la vue B absorbée par une fin
   « gratuite » (rejet ou `000`) suivie d'une vue C vide (probabilité 1/2 à une position
   quelconque) — à mesurer, pas à supposer (proposition §8.1).
6. **[É] Leur oracle est notre propre code** (Go à `d61443ef5`, contextes Go injectés) : il prouve
   l'accord du lecteur Rust avec notre lecteur, pas la justesse ni la fermeture. 43 commits ont
   touché `grammar`/`profile` depuis ; la fermeture n'a presque pas bougé (J11 : +1 à +11 records
   utiles par film), donc l'oracle reste valable comme détecteur de régression de lecteur sur
   HI_1_13_0, mais il encode les 13 lecteurs « exceptions datées » qui contredisent l'écrivain
   (Ghidra).
7. La branche `origin/feat/serde-serialization` n'apporte **rien** au décodeur : 3 commits, dérivés
   serde des modèles de réponse et un remplacement `chunks_exact` → `as_chunks` dans
   `src/clients/hi/film.rs` (ancien lecteur de registre) — `git diff --stat` = 2 fichiers,
   `film.rs` (16 lignes) et `models.rs` (274 lignes).

---

## 1. Périmètre, versions, provenance

| Élément | Valeur | Preuve |
|---|---|---|
| Branche analysée | `origin/simbleau/theater-experiments`, `0aaa05b`, 2026-09-30 15:31 (−0400), S. C. Imbleau | `git log -1` dans le clone |
| Base | merge-base avec `origin/main` = `78f1b3fa`, 33 commits | `git log origin/main..HEAD` |
| Licence | `MIT OR Apache-2.0` | [R] `Cargo.toml:5` |
| Épinglage | LevelUp `43a01721e` (2026-09-22 19:45) + lecteurs choisis de `d61443ef5` (2026-09-26 19:17) | [R] `src/theater/docs/fixtures.json:4-5`, `VALIDATION.md:3-7` |
| Crédit | « Spencer Imbleau's and Guillaume Sitbon's explorations … LevelUp/tree/feat/v75 » | [R] `src/theater/docs/CREDIT.md` |
| Taille | ~16 000 lignes Rust sous `src/theater/`, dont parseur 8 528 lignes | `wc -l` |
| Corpus de validation | 32 films personnels (`experiments/films`, non versionné), 403 465 bornes de lecteur | [R] `film/tests.rs:385-393`, `VALIDATION.md:41-48` |
| Version supportée | v41 seule ; toute autre majeure = erreur | [R] `src/theater/parser/mod.rs:33-37` |
| `feat/serde-serialization` | 3 commits, rien pour le décodeur | `git diff --stat origin/main...origin/feat/serde-serialization` : `src/clients/hi/film.rs` (16 lignes), `src/clients/hi/models.rs` (274) |

Ancêtres LevelUp **[É]** : `43a01721e` est ancêtre de `d61443ef5`, lui-même ancêtre de la tête ;
`43a01721e..d61443ef5` = 270 commits, `d61443ef5..HEAD` = 277. À `43a01721e`, ni `bloc_action.go`, ni
`composants_vue_b_m4b.go`, ni `debut_de_liste.go` n'existaient (`git ls-tree`) : le port en a repris
une partie depuis `d61443ef5` (lot M4b) — `bloc_action` ([R] `components/control.rs:2`) et
`composants_vue_b_m4b` ([R] `components/m4b.rs`), mais pas `debut_de_liste`.

---

## 2. Architecture du port face à la spec

### 2.1 Le flux public

**`Film::parse(chunks)`** ([R] `film/mod.rs:19-21` → `parser/mod.rs:20-44`) **[É]** :
- exige que le premier chunk soit le SEUL registre, lit l'u32 de version dans ses 4 premiers octets,
  choisit `V41ChunkReader` (seule variante de `ChunkReader`) ;
- `PreparedChunk::new` décompresse (ou garde tel quel) une fois et **copie** : `data.into_owned()`
  ([R] `parser/transport/mod.rs:20`) ; le `Chunk<T>` garde `source: FilmChunk` (octets d'origine)
  ET `data` (décompressés) ([R] `film/chunks/models.rs:25-31`) ;
- un `DecodeContext { config, registry, state, event_gate15 }` ([R]
  `parser/v41/chunks/replication/replication_stream/mod.rs:15-20`) suit les paquets de réplication
  dans l'ordre ; `event_gate15` vaut toujours `None` en production ([R] `parser/v41/chunks/mod.rs:29`) ;
- rend `Film { registry, chunks: Vec<FilmDataChunk> }` où `FilmDataChunk` = `Replication | Summary |
  Unknown` ([R] `film/mod.rs:6-15`). Rien n'est paresseux : le film entier est en mémoire.

**`TheaterRuntime::load(film)`** ([R] `runtime/mod.rs:33`, doc `RESOLUTION.md`) **[É]** : prend le
`Film` (ou un `Arc<Film>`), construit un `ResolvedFilm` PRIVÉ (index chronologiques, monde de
composants matérialisé, points de reprise, événements de résumé typés). Les recherches heuristiques
(résumé, joueurs, tirs, visée) n'existent qu'ici, étiquetées `SummaryDerivation::GuardedV41Layout`
et `Provenance::DerivedSummary` ([R] `runtime/resolved/events.rs:91-97`, `summary.rs:121,216`).
Mesuré : sur `0797ce72`, +134 Mio et 0,13 s, 118 773 événements ; sur `4f77afc1`, 436 Mio de
working set au total, 128 499 événements, 844 événements de résumé.

### 2.2 L'état de schéma (`DecodeContext` / `ReplicationDecodeState`)

[R] `parser/v41/chunks/replication/state.rs:26-43` **[É]** :
`slots: BTreeMap<u32, DecodeBinding { archetype, full_id, soft, generation_any, view: Option<i8> }>`,
`current_view`, `keyframe_namespace`, `current_chunk`, `anticipated: Option<AnticipatedBindings>`.

| Source de liaison | Comportement du port | Preuve |
|---|---|---|
| Images-clés (type 2) | `bind_keyframe` pour chaque tentative AVEC record ; la marche de table s'arrête au premier record non complet (`KeyframeChainStop::Desync`) | [R] `.../keyframes/mod.rs:17-25`, `keyframes/table.rs:107-119` |
| Bloc de datums (type 1) | décodé (`datums::decode(payload)`, sans contexte) mais ne lie RIEN | [R] `.../replication_stream/mod.rs:29` |
| NEW | lie si arrêt `Complete` OU `Truncated`, archétype < 50 et présent au registre (même corps tronqué) | [R] `.../frame/entities.rs:136-147` |
| DELETE | délie même si le mot de 32 bits qui suit est hors source | [R] `.../frame/entities.rs:152` |
| Table anticipée | champ présent, jamais rempli hors tests | `grep` : aucun site de production ne pose `anticipated` |
| DELTA non lié / autre vue | fin de vue B `ProductionEntityEnd::Rejected{header}`, compte comme vue terminée | [R] `.../frame/entities.rs:96-104` ; `completed_views` (test) |

### 2.3 Statuts, arrêts, couverture, état par défaut, bits bruts, provenance

| Notion | Forme dans le port | Preuve |
|---|---|---|
| Statut de paquet | `PacketRead::Decoded(T)` (« un décodeur a tourné », PAS « payload complet ») / `Opaque{reason: PacketDecodeError}` | [R] `film/chunks/packet/models.rs:90-97` |
| Arrêts typés | `EventListStop` (Terminator, SourceBoundary, Truncated, UnsupportedCode/References/Body, MissingRuntimeGate15, RecordLimit, InvalidStart) ; `EntityViewStop` (…, MissingBinding, GenerationMismatch, UnsupportedComponent{index,name}, RuntimeContextUnavailable{index,name,field}) ; `ProductionEntityEnd` (Marker, Rejected{header}, Failure, Truncated, PayloadBoundary, RecordLimit) ; `FrameViewStop` (Complete, Truncated, Unsupported{reason}, RecordLimit) ; `KeyframeChainStop` ; `FrameDecodeError` (IncompleteMessageList, ConflictingMessageLayout…) | [R] `film/chunks/replication/replication_stream/models/*.rs` |
| Partition de couverture | `Packet::source_coverage(&chunk.data) -> Option<Vec<SourceRegion>>`, régions demi-ouvertes Fields / Opaque / Padding / Unparsed, dérivées du modèle (aucun second décodage), O(n log n), `None` si conflit ou hors source ; un trou n'est JAMAIS présumé bourrage | [R] `film/chunks/packet/models.rs:28-44`, `packet/coverage.rs:5-62` |
| État par défaut | `DefaultState { source: BitRange, fields, status: Complete/Opaque/Unsupported/Truncated }`, séparé des gardes et masques | [R] `.../models/default_state.rs` ; construit dans `frame/records.rs` (NEW) |
| Bits bruts | `RawBits { bit_len, bytes: Vec<u8> }` MSB d'abord, aussi pour > 64 bits ; `ComponentField { name: String, bit, width, raw }` | [R] `film/chunks/replication/components/field.rs:4-16` |
| Provenance | `PacketSource { header: ByteRange, payload: ByteRange }` dans le chunk décompressé ; bits relatifs au payload ; étendues demi-ouvertes ; au runtime `Provenance::{RecordedRead, PartialRead, PacketEnvelope, DerivedSummary…}` | [R] `packet/models.rs:121-124`, `FORMAT.md` §Source coordinates, `runtime/resolved/events.rs:91` |
| Tentative hors source | `EventFieldValue::Unavailable` : une lecture qui dépasse la source est gardée comme TENTATIVE, jamais comme zéro | [R] `.../models/kill_event_chain.rs` (enum `EventFieldValue`) |

### 2.4 Principes P1-P10

| # | Statut dans le port | Preuve et commentaire |
|---|---|---|
| P1 grammaire seule autorité | **Fait** [É] | Entrée unique, « does not search ahead for plausible record boundaries » ([R] `FORMAT.md` §Decoder scope) ; recherches de résumé/joueurs/tirs seulement au runtime. Prix payé : pas de balayage d'ancres → contexte appauvri (§7). |
| P2 trois états (interprété / délimité / infranchissable + queue opaque à cause typée) | **Partiel** [É] | Composant : `Complete / Unsupported / Truncated` ; état par défaut a `Opaque` ; événements : `Scalar / Opaque / Unavailable`. Pas de distinction « interprété » vs « délimité » au niveau composant (tout est `RawBits`). La « queue opaque » existe en deux morceaux séparés : l'arrêt typé et la région `Unparsed` de la couverture. |
| P3 forme = niveaux stables | **Fait pour v41** [É] | Film → chunks → paquets → trame (config, vue A, vue B, vue C). Pas de profil par build : un seul `V41DecodeConfig::default()` ([R] `parser/v41/context/mod.rs:86-97`). |
| P4 paramètres hors flux explicites avec provenance (lu / calibré / supposé) | **Autrement** [É] | Pas de provenance. Paramètres fixes (`id_low_bits: 13`, `DecodeProfile::default()`) ou absents → arrêt typé (`MissingRuntimeGate15`, `RuntimeContextUnavailable{field:"vehicle+0x818"}` [R] `components/m4b.rs:9`). Empreinte de registre inconnue = simple avertissement, le décodage continue ([R] `parser/v41/chunks/registry/mod.rs:23-50`) — contraire à notre D-4. |
| P5 état de lecture exposé en lecture seule | **Absent** [É] | `ReplicationDecodeState` est `pub(crate)` ; « Parser schema bindings are private decoding context » ([R] `ARCHITECTURE.md`). Une table unique avec un champ `view`, pas trois tables. |
| P6 identité `{Slot, Gen}` | **Fait, autre forme** [É] | Identifiant complet `tag<<30 | low` ([R] `frame/header.rs:47`), liaison par `full_id`, arrêt `GenerationMismatch` ([R] `frame/records.rs:176-182`). |
| P7 provenance de bout en bout | **Fait** [É] | §2.3. |
| P8 récupération en couche séparée | **Fait par retrait** [É] | « Candidate recovery and its oracle fixture have been removed » ([R] `VALIDATION.md:46-48`) ; les seules lectures heuristiques sont au runtime, étiquetées. Il n'y a pas de couche de récupération qui travaille sur les queues opaques : elle est simplement absente. |
| P9 paresseux, sans copie, en flux | **Absent** [É] | Lot complet en entrée et en sortie ; deux copies par chunk ; un `String` et un `Vec<u8>` par champ. Mesures §3. |
| P10 ordres totaux | **Fait** [É] | `BTreeMap` pour l'état et la couverture ; ordre d'entrée conservé ; seul `HashSet` = déduplication d'un avertissement de journal ([R] `registry/mod.rs:19-31`). |

### 2.5 Nœuds du modèle §4

| Nœud de la spec | Port | Preuve |
|---|---|---|
| En-tête (build, majeure, empreinte, profil, paramètres + provenance) | **Partiel** : majeure lue ; deux mots d'en-tête `FilmRegistryRead.header` ; empreinte calculée (`pub(crate)`) mais seulement pour avertir ; pas de build ; profil fixe ; aucune provenance | [R] `parser/mod.rs:31-37`, `film/chunks/registry/models.rs:76-110` |
| Registre | **Fait** : archétypes → composants ordonnés, 256 octets de nom, niveau u32, étendues de bloc et de bourrage | [R] `film/chunks/registry/models.rs:6-60` |
| Chunks[] | **Fait** (type, index, `start_ms`, position d'entrée, transport `Clear/ZlibComplete/ZlibPartial/ZlibRejected`) ; pas de « finalisé » | [R] `film/chunks/models.rs` |
| Paquets[] | **Fait** (16 octets d'en-tête, type, `unknown_2`, taille, horodatage µs, étendues d'octets) ; `PacketStream.opaque` = suffixe non cadré | [R] `packet/models.rs:13-17,83-87,128-133` |
| Trame delta : bit de configuration puis vues A, B, C dans l'ordre | **Fait** (`FramePacket { configuration, events, frame }`) | [R] `.../models/mod.rs` (`FramePacket`), `frame/mod.rs:18-38` |
| Vue A (genre R(7), étendue, état) | **Fait, grammaire killsource** : `EventRecord { start_bit, end_bit, code, body_start_bit, fields, layout_complete }` | [R] `frame/events.rs:154-246` |
| Vue B : genre NEW/DELTA/DEL/END, handle, archétype, masque, état par défaut, composants, étendues | **Fait** sauf : archétype publié pour NEW seulement ; pas d'« hypothèses[] » par composant | [R] `.../models/entity_records.rs`, `frame/records.rs` |
| Vue C : entrées {kind, index, étendue, état} | **Fait pour kind 0** (`ControlEntry`), kinds listés (`DecodedFrameView.kinds`) ; kinds 1/2 et bloc secondaire = arrêt | [R] `.../models/frame_views.rs`, `frame/controls.rs:96-171` |
| Fermeture par vue (bits consommés contre longueur) | **Absent** : `end_bit` par vue, aucune confrontation à la fin du paquet | [R] `frame/entities.rs:162-182` |
| Queue opaque {étendue, cause} | **Partiel** : cause = arrêt typé, étendue = région `Unparsed` (non reliées) | §2.3 |
| Image-clé | **Fait** (`KeyframeTable`), mais la marche s'arrête au 2e record sur nos films (§4.5) | [R] `keyframes/table.rs` |
| Autres types | type 1 **décodé en entier** (`DatumTable` : en-têtes 79 bits, masques 256 bits, 5 mots, bourrage) ; type 7 fin ; type 8 opaque ; 6, 10, 12 `UnsupportedLayout` | [R] `.../replication_stream/mod.rs:24-35`, `models/datums.rs` |
| Temps forts (chunk 3) | **Partiel** : paquet type 9 = compte déclaré u32 BE + flux opaque ; candidats au runtime | [R] `film/chunks/summary/mod.rs`, `parser/v41/chunks/summary/mod.rs:38-50` |
| État de marche (tables par vue, liaisons inférées marquées) | **Absent du modèle public** (privé) ; `soft` / `generation_any` existent en interne | [R] `state.rs:26-43` |
| Diagnostics (fermetures, queues par cause, hypothèses, récupérés) | **Partiel** : couverture + arrêts ; pas de fermeture, pas d'hypothèse (refus), pas de récupérés | — |

---

## 3. Modèle de données et mémoire

### 3.1 Types et tailles

Tailles mesurées par `std::mem::size_of` dans l'exemple **[É]** : `ComponentField` 72 o,
`EventField` 40 o, `EntityRecord` 288 o, `EntityComponentRead` 80 o. Chaque `ComponentField`
porte en plus deux allocations de tas (`name: String`, `raw.bytes: Vec<u8>`) ; les noms sont des
`format!` (`"record[3].present"`, `"actions.a[0]"`…) ([R] `frame/controls.rs:97-111`).

Sur `0797ce72` (20,5 Mio, lecture arrêtée tôt dans la plupart des paquets) **[É]** :
1 041 761 `ComponentField`, 12,6 Mo de noms, 1,35 Mo d'octets bruts, 82 991 `EventField`,
188 393 entrées de datums ; working set 25 Mio après chargement → **331 Mio après `Film::parse`** ;
`EntityComponentRead` est cloné à chaque composant (`rec.components.push(component.clone())`, puis
remplacé, et `fields: r.fields[..].to_vec()`) ([R] `frame/records.rs:267-309`).

**[S]** Coût de la représentation ≈ 130 o par champ ≈ 18 o par bit de trame type 0 traversé.
Extrapolé à une traversée complète : ~1 Go pour `bfecd02b` (57,6 Mbit de trames type 0) et
~3,5 Go pour `4f77afc1` (197,5 Mbit), avant le runtime. **Le port matérialise tout le film**, et ne
tient en mémoire aujourd'hui que parce qu'il s'arrête tôt.

### 3.2 Ce qui se transpose en Go (`film/types` + `grammar`), sans copier de Rust

| Idée | Transposition proposée | Respecte P9 ? |
|---|---|---|
| Étendue demi-ouverte relative au payload + étendue d'octets du payload dans le chunk | `types.Etendue{Chunk, Paquet, Debut, Fin}` (bits, demi-ouverte) — c'est déjà la forme §4 | oui |
| Statut de paquet `Decoded/Opaque{raison}` | `types.LecturePaquet` (énuméré + cause) | oui |
| Arrêts typés par niveau, fin de vue B qualifiée | `types.FinDeVueB{Genre: Marqueur|Rejet|Echec|FinDePayload|Plafond; Entete; Cause}` — Go a déjà `ArretVueC` ([G] `internal/grammar/frame_vue_controle.go:40-58`) | oui |
| Partition de couverture | `grammar.Couverture(paquet) []types.Region` calculée depuis la structure, invariant « tout bit classé, aucun chevauchement » (T1 bis) | oui (calcul à la demande) |
| `DefaultState` séparé des gardes et du masque | nœud `EtatParDefaut{Etendue, Statut}` | oui |
| Tentative hors source (`Unavailable`) | drapeau « lu dans le bourrage » sur la valeur, pour que `facts` ne publie pas un zéro synthétique | oui |
| Provenance `RecordedRead / PartialRead / DerivedSummary` | provenance des faits (P7) + marque « récupéré » de la couche P8 | oui |
| Bits bruts matérialisés (`RawBits`), nom par champ | **NE PAS reprendre** : étendue + index de composant dans le registre, décodage à la demande sur le tampon du chunk | — |

---

## 4. Couverture grammaticale comparée (port vs tête Go)

### 4.1 Vue A (messages)

| | Port | Go tête |
|---|---|---|
| Lecture | SÉQUENTIELLE depuis le bit 1, avec la table de présence et les corps de `killsource` (codes à taille fixe, 85 kill, 1, 15, 0, 82 ; ~28 codes sur 123) | Pas de lecture séquentielle dans la marche : localisation de la liste par recherche (`marchLocateStrict`, signature du slot 123) puis `debutDeLaListe` (NEW de tête prouvés par chaîne ou par fermeture) |
| Preuve | [R] `frame/events.rs:8-131` (table), `:347-509` (présence et corps) | [G] `internal/grammar/debut_de_liste.go:45-51`, `frame_closure.go:230-244` ; corps `internal/facts/killsource/eventbody.go:10-11,19-50` (« COUVERTURE ASSUMEE : 28 codes sur 123 ») |
| Code 15 (porte d'exécution) | `event_gate15 = None` → `MissingRuntimeGate15` ([R] `events.rs:217-218`, `chunks/mod.rs:29`) | tranché PAR FILM dans killsource : `pickGate15` essaie les deux et garde celui qui enchaîne le plus d'événements ([G] `internal/facts/killsource/assist.go:183-198`) |
| Code 0 | trame refusée (`ConflictingMessageLayout`) ([R] `frame/mod.rs:24-25`) | corps bit-exact 84-241 bits ([G] `eventbody.go:81`) |
| Mesure (10 témoins v41) | 44 750 trames refusées avant la vue B (13,3 %) : 36 775 listes non terminées (`UnsupportedBody` 29 319, `MissingRuntimeGate15` 7 422, autres 34) + 7 975 listes contenant un code 0 | « liste non localisée » : 15 613 paquets sur 9 de ces films (carte du 26/09, `fermeture_bloquants.tsv`) |

Idée utile **[É]** : `gate15` est exactement un « paramètre hors flux calibré » de P4 ; chez nous il
est choisi dans `killsource` seulement, il devrait vivre dans l'en-tête de la structure avec sa
provenance « calibré ».

### 4.2 Vue B (records, composants)

**[É] Couverture par nom identique.** Les 217 noms de composants présents comme littéraux dans
[R] `parser/v41/chunks/replication/components/*.rs` et dans `ecs_table.tsv` couvrent exactement les
lignes `porte` et `partiel` de la table, et aucune autre :

| Statut `ecs_table.tsv` (tête) | Lignes | Nom connu du port | dont `product_use` |
|---|---|---|---|
| `porte` | 544 | 544 | 42 |
| `partiel` | 51 | 51 | 1 |
| `non_porte` | 440 | 0 | 0 |
| `deser_non_cable` | 32 | 0 | 1 |

(commande : `grep -ohE '"[A-Za-z][A-Za-z0-9_-]+"'` sur les composants Rust, `comm` avec la colonne
`component` ; statuts identiques entre `d61443ef5` et la tête : 544 / 51 / 440 / 32, contre 537 / 51 /
447 / 32 à `43a01721e`.) Un nom connu ne prouve pas un lecteur identique : les 13 exceptions datées
J6/R3 (§6) ont changé ou figé des lecteurs depuis.

Différences de comportement **[É]** : le port n'a ni l'inférence de chaîne, ni la réparation de
composant non porté (`repairUnportedComponent`), ni le refus « NEW qui contredit une entité vivante »
([G] `internal/grammar/frame_infer.go:178-207`), ni la table anticipée en production, ni
`TableDeDatums` (lecture à position libre des en-têtes d'image-clé, [G]
`internal/grammar/keyframe_datums.go:1-60`).

### 4.3 Vue C (contrôle) — notre premier bloqueur

**Comment le port trouve la fin de la vue C [É]** : exactement comme nous. Boucle `présent = R(1)` ;
0 → fin ; `kind = R(2)` ; 3 → rien, on continue ; 0 → entrée de contrôle ; 1 ou 2 →
`Unsupported("control handler kind 1 or 2")` ; puis `secondary.present = R(1)` → 1 =
`Unsupported("secondary control block")` ; plafond 64 tours ([R] `frame/controls.rs:96-171`).
Même structure que [G] `internal/grammar/frame_vue_controle.go:122-205` (bloc 0xbc = « secondary
control block »). Le bloc d'action est celui de `bloc_action.go` de `d61443ef5` ([R]
`components/control.rs:2,36-110` ; [G] `internal/grammar/bloc_action.go:88-126`).

**Ce que le port n'a pas [É]** : aucun oracle de fermeture. « Complete » = un bit de présence 0 lu.
Il ne confronte jamais la fin de la vue C à la fin du paquet ([R] `frame/entities.rs:162-182` ne
borne que le dépassement). Rien sur les kinds 1/2 ni le bloc secondaire : aucune idée nouvelle là.

**Ce que l'expérience montre (10 témoins v41, §7) [É]** :

| Fin de la vue B | Trames « trois vues complètes » | Dont fermées au sens LevelUp |
|---|---|---|
| `Marker` (record END, `000`) | 65 975 | 18 559 (28,1 %) |
| `Rejected` (delta sur slot non lié ou d'une autre vue) | 175 447 | **28 (0,02 %)** |

Sur les 222 835 trames « trois vues complètes » NON fermées : **209 982 (94,2 %) ont une vue C vide**
(premier bit 0, aucun kind lu), dont 166 768 après un rejet et 43 214 après un `Marker` ; le reste du
payload après la vue C dépasse 256 bits dans l'immense majorité (ex. `4f77afc1` : 16 618 + 1 283 sur
17 920).

**Lecture [É + S]** : une fin de vue B par rejet coûte `1 + idLow + 2` bits quelconques (presque
toute position désalignée se lit comme un delta sur un slot inconnu), une fin par `Marker` coûte
trois bits nuls (1/8 au hasard), et une vue C vide coûte un bit nul (1/2). Ces trois fins sont des
oracles FAIBLES ; seule la fin du paquet (reste de 0 à 7 bits nuls) est forte. Notre code a la même
forme : le rejet clôt la vue B avec `hitEnd = true` ([G] `internal/grammar/frame_infer.go:291-294`),
la vue C est alors « atteinte » ([G] `frame_harvest.go:333-349`), et un paquet qui lit son
terminateur sans fermer est classé « terminateur hors cadre » sans autre détail ([G]
`frame_closure_classement.go:94-109`). **[S]** Une part importante de nos 264 757 « terminateurs hors
cadre » (cause n° 1, MESURES J11 §2) est probablement une désynchronisation de la vue B absorbée par
une fin faible ; la mesure à faire est au §8.1. Indice déjà établi chez nous : sur `bfecd02b`
(mesure du lot 5.21, avant M4b), **0** des 23 325 en-têtes rejetés n'était vivant dans le bloc de
type 1 de son chunk, 23 306 tombaient sur une entrée entièrement vide
(`.ai/V7.5/film_re/NOTE_5_21_BLOC_TYPE_1_2026-09-22.md:151-156`) : des lectures à position fausse.

### 4.4 Paquets de type 1 (datums)

**[É]** Les deux décodent la même grammaire (6 + 8 + 32 + 33 = 79 bits par slot, 256 bits de masque,
5 mots, bourrage) : port [R] `.../replication_stream/datums.rs`, `models/datums.rs` ; Go
[G] `internal/grammar/type1_datums.go:1-60,195` (`LireBlocDeDatums`, lu chez l'écrivain
`FUN_1429883ec`, fermé par l'arithmétique 343 019 octets). **Aucun des deux ne s'en sert pour lier en
production** (Go : « La liaison n est donc PAS cablee en production », NOTE_5_21:166 ; port :
`datums::decode(payload)` sans contexte). Mesuré dans le port : 2 588 356 entrées sur 10 films, 100 %
des bits en `Fields` (867,2 Mbit, 39 % des bits des films).

### 4.5 Paquets de type 2 (images-clés)

**[É]** Port : marche séquentielle, sans recherche ; sur les 316 images-clés des 10 films, elle rend
exactement 2 records par table (632), le second s'arrêtant sur `tacmap-mapdismissallock`
(`ti=34 i10`, `non_porte` chez nous aussi : `ecs_table.tsv:682`), puis `Desync`. Couverture type 2 :
0,2 % de bits en `Fields`. Go : `WalkKeyframeRecords` + balayeur d'ancres et `TableDeDatums`
(position libre) — p. ex. 6 460 records d'image-clé sur la mini-bobine `fb1a1a72`
(`internal/grammar/testdata/keyframe_closure.golden`, 2 514 fermés).

### 4.6 Paquets de type 8 (roster)

**[É]** Port : `RosterPacketBody` vide, payload 100 % `Unparsed` (112,9 Mbit), « personalization width »
non établie ([R] `.../replication_stream/roster.rs`, `COVERAGE.md` §Structural limits). Go :
`DecodeRoster(pay, formatVersion, persoBits)` ([G] `internal/grammar/roster_type8.go:127`, grammaire de
la table de `chunk_00`, `FUN_142987bd4`). Nous sommes en avance.

### 4.7 Chunk de type 3 (résumé / temps forts)

| | Port | Go tête |
|---|---|---|
| Canonique | cadrage en paquets ; paquet type 9 = `declared_events` (u32 BE) + flux opaque ; type 7 fin | pas de cadrage : décompression puis balayage BIT À BIT des XUID (`[64 bits XUID][0x2d|0x25][0xc0]`, fenêtre 20 000 bits, marqueur de fin) — port de SPNKr |
| Interprétation | runtime : recherche gardée (`GuardedV41Layout`), compare candidats et compte déclaré (`summary_reports().count_matches()`) | `ParseHighlightEvents` → kill-feed ; `ScanDeaths` → fil des morts (filtre `EventTypeDeath`) |
| Preuve | [R] `parser/v41/chunks/summary/mod.rs:38-50`, `RESOLUTION.md` §Summary events | [G] `internal/grammar/highlight_events.go:16-33,103-134`, `deaths_source.go:123-160` |

**[É]** Mesuré : sur `0797ce72` et `4f77afc1`, le runtime du port rend 220 et 844 événements pour
220 et 844 déclarés. Notre lecteur n'utilise PAS le compte déclaré (aucune lecture des 4 premiers
octets du payload type 9 dans `highlight_events.go`) : c'est un oracle de complétude gratuit (§8).

---

## 5. Divergences documentées par l'auteur

| Divergence (source) | Existe à la tête Go ? | Impact |
|---|---|---|
| **Lectures au-delà du payload avec bourrage à zéro** (`CANONICAL_AUDIT.md` « reference endpoint at bit 34 of a 32-bit payload » ; le port refuse les valeurs synthétiques) | **Oui** [É] : `source.Bits` rend 0 au-delà du tampon ([G] `internal/source/bits.go:18-25,90-99,128-132`) ; la vue B s'en sert (boucle `for br.BitPos() < frameLen`, corps sans borne, [G] `frame_infer.go:271-305`) ; les vues A et C sont bornées (`placeDisponible`, [G] `frame_vue_messages.go:91-96`, `frame_vue_controle.go:125-136`). **Aucune preuve Ghidra trouvée dans le dépôt** : « c est le bourrage de queue du moteur » n'est appuyé que par l'arbitrage V15 (3), qui conserve la sémantique de l'ancien `BitReader` (`.ai/V7.5/PLAN_DECODEUR_FILM_2026-09-13.md:103,3975`). Grep `bourrage` / `refill` dans `.ai/`, `docs/` et `grammar` : aucun relevé de l'écrivain/lecteur du jeu à ce sujet. | Fermeture : nul (un paquet qui déborde ne ferme pas). Données : **[S]** des valeurs lues partiellement dans le bourrage peuvent être publiées par les canaux qui ne demandent pas la fermeture ; des liaisons peuvent naître d'un corps bourré (ligne suivante). À compter : records dont la fin dépasse `frameLen` (mode borné d'instrument, §8). |
| **Déclaration NEW conservée malgré un corps tronqué** (slot 256 / ti 21 à l'octet 492 675, lu jusqu'au bit 214 d'un payload de 208 ; slot 7200 à l'octet 501 189, lu jusqu'au bit 254) | **Oui, autrement** [É] : Go lie un NEW dont la traversée BOURRÉE ne désynchronise pas, sans contrôle de fin ([G] `frame_infer.go:178-207`, `w.BindFull` l. 203) ; le port lie dès que l'archétype est lu, corps complet ou tronqué ([R] `frame/entities.rs:136-147`). Les deux décident sur des bits hors source. | Contexte des paquets suivants : une mauvaise liaison désynchronise la suite (visible en fermeture). Fréquence non mesurée sur notre corpus [S faible : paquets de 26 octets du film « controller » de l'auteur]. |
| **Métadonnée DELETE au-delà de la source** (octet 501 401) | **Oui, et le port s'est aligné** [É] : Go `br.Skip(32)` sans borne puis `w.Unbind(slot)` ([G] `frame_infer.go:303-305`) ; port : délie même tronqué ([R] `frame/entities.rs:152`). | Aucun écart résiduel entre les deux. |
| **Hypothèse `vehicle+0x818`** (ti=40 i34 `vehicle-type-physics`) | **Oui, assumée et comptée** [É] : repli nommé `repli_physique_de_type_de_vehicule_supposee`, prouvé par l'oracle de cadrage (fenêtre LAAG de `1cd3848a` : 0 → 765 paquets fermés sur 785) ([G] `internal/grammar/composants_vue_b_m4b.go:88-121`, registre `internal/facts/fallback/registre_filmdec_marche.go:50-68`) ; déclenché sur 7 témoins, 33 193 fois (MESURES J11 §3:142). Le port s'arrête (`RuntimeContextUnavailable`, [R] `components/m4b.rs:6-9`, `frame/records.rs:286-292`). | Doctrine différente, la nôtre est mieux étayée (la fermeture du paquet prouve la présence du corps). Mais ce n'est pas une largeur fixe qu'on saute : c'est une porte → la règle utilisateur du 25/09 exige Ghidra (cible de retrait déjà écrite). |
| **Divergence de liaison du contrôleur** (`natural-end/04-walk-forward/chunk-002-type-2.bin:509578` : déclaration slot 7200/ti 27 absente du monde Go ; arrêt Rust au bit 184 contre vues complètes Go au bit 183) | Cas des deux lignes précédentes, sur un film de l'auteur (non présent chez nous). Mécanismes toujours présents à la tête [É]. | Illustration : des contextes cumulés différents à cause d'une lecture tronquée en amont. L'auteur a contourné en injectant les contextes Go par trame (`reference-contexts-d61443e`, 1 241 instantanés). |

---

## 6. Fraîcheur de leur oracle

`git log --oneline d61443ef5..HEAD -- …/film/internal/grammar …/film/internal/profile` **[É]** :
**43 commits (37 hors fusions)**. Thèmes :

| Thème | Commits (exemples) | Touche les bornes de lecture ? |
|---|---|---|
| J3 révisions par fermeture des imports, empreintes insensibles aux commentaires | `6b5f76915`, `cae8cc866`, `c456deb6e` | non |
| J4 carte de fermeture (instrument), étage unique du pont, façade, lecteurs de bits via `source` | `c465f93ec`, `dd0d4dfa3`, `ba7ad4322`, `a15bfc126`, `c69d0c097` | non (déplacements, zéro différence) |
| J5 `LifeKey`, GB-1 générations vivantes, filtre de génération daté (R2) | `10574e8ad`, `9f1563ff2`, `5f65ff049`, `6883456ea`, `f924ab4ef` | non (sélection des faits) |
| **J6 portage unique de `FUN_14076e524`** (GA2-3, GA2-4, bit `precHigh`) | `56299bdc3` | **oui** |
| **Exceptions datées J6-bis / R3 / R3-bis** (13 sites gardent l'ANCIEN lecteur parce que la lecture du jeu fait baisser la fermeture) | `0fc3277b5`, `562e060ba`, `26ef60122`, `a1544498f` | **oui** — [G] `internal/grammar/lecteur_position_exceptions.go:1-20` |
| J7 carte obligatoire pour killsource ; J8 replis comptés ; J10 tris totaux, LIS des datums, GA1-5, GB-4 | `fa132074b`, `7a703beb7`, `e2a4462a4`, `3f7d672a9`, `b3a38a5a5`, `f5c223978` | marginal (`3f7d672a9` change la coïncidence de la table de datums) |
| Perf : ancres d'image-clé par motif de 64 bits, marche d'image-clé une fois par contexte | `ffa45a435`, `12ae77ba2` | non (mêmes sorties) |

**Que vaut leur oracle aujourd'hui [É + S]** :
- La fermeture n'a presque pas bougé entre le 26/09 et la tête (+1 à +11 records utiles par film,
  MESURES J11 §2) : **[S]** les bornes de lecture sur HI_1_13_0 sont très majoritairement les mêmes
  qu'à `d61443ef5`, l'oracle reste un bon détecteur de RÉGRESSION de lecteur.
- Il n'est **pas indépendant** : ses 403 465 bornes, ses contextes (`reference-contexts-d61443e`) et
  les 124 paires de médailles viennent de notre code ([R] `docs/fixtures.json`, `VALIDATION.md:77-97`).
- Il encode des lecteurs que nous savons contraires à l'écrivain (les 13 exceptions datées) et des
  hypothèses (`+0x818` côté référence) : l'accord Rust/Go n'en dit rien.
- Il porte sur un contexte RÉDUIT : le harnais lie seulement par `WalkKeyframeRecords`, démarre
  chaque paquet au bit 2 sans localiser la liste d'événements, sans table anticipée ni table de
  datums ([R] `src/theater/fixtures/reference-contexts-v41_test.go:89-121`). Il valide donc un
  lecteur sous contexte fourni, jamais une fermeture.

---

## 7. Harnais Go `reference-contexts-v41_test.go`

Symboles appelés et présence à la tête **[É]** (`grep`, aucune copie dans le dépôt) :

| Symbole | Tête | Preuve |
|---|---|---|
| `source.MemoryChunks`, `source.Load(src, meta)` | oui | [G] `internal/source/source.go:40`, `internal/source/film.go:103` |
| `types.ChunkMeta` | oui | [G] `types/source.go:11` |
| `NewFilmContext`, `(*FilmContext).CadreDeBalayage` | oui | [G] `internal/grammar/film_context.go:251,225` |
| `ParseRegistryChunk`, `NewWorld`, `ContexteParDefaut` (+ champ `Profil`) | oui | `registry.go:221`, `world.go:82`, `profil_balayage.go:343-349` |
| `(*World).PoserChunkCourant`, `(*World).BindImageCle` | oui | `world.go:108,262` |
| `WalkPackets`, `FilmPacket.{Type,Start,Payload}`, `PacketTypeKeyframe`, `PacketTypeDelta` | oui | `film_packets.go:15-36,72` |
| `WalkKeyframeRecords`, `KeyframeHeader.SansArchetype`, champs `Gen`, `Slot`, `Archetype` | oui | `keyframe_record_walk.go:71-82,183` |
| `DecodeFrameViewsCurseur(buf, w, cfg, nViews, skip) (recs, rangs, fin)` | oui | `frame_harvest.go:162-163` |
| `record.Trace.Comps[].{Name,StartBit}` | oui | `traverse.go:21-37`, `frame_records.go:154` |
| `compVehicleTypePhysics` | oui | `composants_vue_b_m4b.go:108` |
| champs non exportés `world.slots` (map de `slotState`, champs exportés → JSON) et `world.nsImageCle` | oui | `world.go:29,41,54-62` |

Verdict : **aucun symbole manquant [É]** ; compilation probable dans `package grammar` **[S]** (non
tentée : copie interdite). Adaptation nécessaire : le harnais attend `film.json` avec `file`,
`chunk_type`, `start_time_offset_ms` ; notre cache a `chunk_NN.bin` + `film_manifests/<id>.json`
(`index`, `chunk_type`, `start_ms`) → un adaptateur d'entrée. Il exige 32 films (`films != 32`).
`world.go`, `keyframe_record_walk.go`, `film_context.go` ont changé depuis `d61443ef5` (29+/48−) :
les instantanés de contexte auraient d'autres valeurs (générations vivantes J5, coïncidence J10.5).

---

## 8. EXPÉRIENCE : `Film::parse` sur nos témoins v41

### 8.0 Protocole

- Version majeure de chaque témoin de `config/replay_corpus.toml` lue en tête de `chunk_00.bin`
  (u32 LE, cf. [G] `internal/grammar/film_major_version.go:72-77`) **[É]** : v41 = `0797ce72`,
  `c75f33b8`, `f75e7053`, `bf15f7ab`, `51ebbc0f`, `bfecd02b`, `396cfc92`, `d9781168`, `fb1a1a72`,
  `4f77afc1` (tous HI_1_13_0) ; v40 `bcb6d393`, `e5adf7b2` ; v39 `084a804d`, `111fa685` ; v38
  `11de8353` ; v37 `60ae07c4` ; v33 `a349fea8`, `a521164d` ; v31 `50247b26`. Les 9 non-v41 sont
  refusés par le port (`UnsupportedVersion`), non lancés.
- Build : `cargo build --release` (cargo 1.97.1, 52 s), exemple `examples/levelup_closure.rs`
  (dans le CLONE). Entrée : `data/cache/film_chunks/<id>/chunk_NN.bin` (déjà décompressés : le port
  les accepte en `Clear`) + `data/cache/film_manifests/<id>.json` pour le type de chunk.
- Mesures : statuts de paquets, arrêts par niveau, trames dont les trois vues sont complètes,
  **fermeture au sens LevelUp appliquée à la sortie du port** (événements `Terminator`, vue B
  `Marker|Rejected`, vue C `Complete`, ET reste du payload après la vue C de 0 à 7 bits tous nuls —
  la définition de `vueCFermee`, [G] `frame_vue_controle.go:294-305`), records, records utiles
  (`product_use` de `ecs_table.tsv`), couverture, temps, pic de working set
  (`K32GetProcessMemoryInfo`) avec un garde-fou qui tue le processus au-delà de 4 Gio.
- Un film à la fois, du plus petit au plus grand ; aucun n'a approché la limite.

### 8.1 Résultats par film

| Film | Entrée | Trames t0 | Refusées (vue A) | Fin B Rejected / Marker | 3 vues complètes | **Fermées (sens LevelUp)** | **Go tête : fermés** | Records NEW+DELTA lus | Utiles lus / fermés | Bits t0 en `Fields` | Parse | Pic |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `0797ce72` | 20,5 Mio | 26 740 | 1 532 | 13 167 / 10 091 | 19 245 (72,0 %) | **709 (2,7 %)** | 19 205 (71,8 %) | 49 434 | 9 215 / 11 | 16,1 % | 0,29 s | 334 Mio |
| `c75f33b8` | 20,5 Mio | 27 800 | 2 153 | 16 928 / 7 657 | 22 207 (79,9 %) | **4 763 (17,1 %)** | 22 944 (82,5 %) | 35 499 | 2 072 / 5 | 11,6 % | 0,19 s | 262 Mio |
| `f75e7053` | 21,3 Mio | 28 502 | 2 513 | 16 359 / 7 863 | 20 210 (70,9 %) | **2 071 (7,3 %)** | 23 423 (82,2 %) | 46 555 | 7 372 / 10 | 15,5 % | 0,28 s | 324 Mio |
| `bf15f7ab` | 22,6 Mio | 31 053 | 2 855 | 15 354 / 10 335 | 22 331 (71,9 %) | **2 823 (9,1 %)** | 28 476 (91,7 %) | 49 678 | 14 396 / 7 | 12,9 % | 0,27 s | 337 Mio |
| `51ebbc0f` | 24,1 Mio | 30 666 | 3 418 | 22 432 / 3 664 | 24 286 (79,2 %) | **1 289 (4,2 %)** | 9 847 (32,1 %) | 31 098 | 5 250 / 5 | 7,7 % | 0,15 s | 230 Mio |
| `bfecd02b` | 24,4 Mio | 31 232 | 3 834 | 18 491 / 7 169 | 22 002 (70,4 %) | **1 806 (5,8 %)** | 26 403 (84,5 %) | 51 885 | 3 798 / 5 | 12,9 % | 0,29 s | 356 Mio |
| `396cfc92` | 25,1 Mio | 32 052 | 3 041 | 18 237 / 9 423 | 23 271 (72,6 %) | **80 (0,2 %)** | 22 828 (71,2 %) | 48 963 | 14 725 / 3 | 11,4 % | 0,26 s | 325 Mio |
| `d9781168` | 32,2 Mio | 43 645 | 6 515 | 26 405 / 9 399 | 32 102 (73,6 %) | **1 881 (4,3 %)** | 26 464 (60,6 %) | 47 879 | 6 279 / 10 | 10,7 % | 0,26 s | 364 Mio |
| `fb1a1a72` | 37,1 Mio | 48 771 | 5 554 | 33 085 / 7 177 | 35 855 (73,5 %) | **1 170 (2,4 %)** | 22 318 (45,8 %) | 63 180 | 21 720 / 19 | 9,0 % | 0,31 s | 416 Mio |
| `4f77afc1` | 67,6 Mio | 35 499 | 13 335 | 17 873 / 3 573 | 19 913 (56,1 %) | **1 995 (5,6 %)** | 22 910 (64,5 %) | 18 252 | 3 037 / 0 | 2,8 % | 0,16 s | 332 Mio |
| **Total** | 295 Mio | **335 960** | 44 750 | 198 331 / 76 351 | 241 422 (71,9 %) | **18 587 (5,5 %)** | **224 818 (66,9 %)** | 442 423 | 87 864 / 75 | 9,2 % | — | max 416 Mio |

Sources Go : `fermeture_films.tsv` de `.ai/V7.5/film_re/carte_fermeture_2026-09-26/` et MESURES J11
§2 (tête, `f75e7053` absent du 26/09). Les nombres de trames type 0 sont IDENTIQUES des deux côtés
pour chaque film **[É]**.

Autres totaux (10 films) **[É]** :
- Statuts : types 0, 1, 2, 7, 8 et 9 `Decoded` ; types 6, 10 (un par trame type 0 : 26 740 sur
  `0797ce72`), 12 et 7 du chunk 3 `Opaque(UnsupportedLayout)` ; aucune couverture refusée
  (`source_coverage` = `Some` sur 100 % des paquets).
- Vue A : `Terminator` 299 185, `UnsupportedBody` 29 319, `MissingRuntimeGate15` 7 422,
  `UnsupportedReferences` 30, `UnsupportedCode` 4 (somme = 335 960). Trames refusées :
  `IncompleteMessageList` 36 775, `ConflictingMessageLayout` (code 0) 7 975.
- Fin de vue B : `Rejected` 198 331, `Marker` 76 351, `Failure:UnsupportedComponent` 11 379,
  `Failure:InvalidArchetype` 3 107, `Failure:Truncated` 2 004, `Failure:RuntimeContextUnavailable` 10,
  `Truncated` 26, `PayloadBoundary` 2.
- Vue C atteinte : `Complete` 241 422 ; `kind 1 or 2` 27 525 ; `secondary control block` 5 731 ;
  `Truncated` 4 (Go, synchronisé : « kind non porté » 1 à 913 paquets par film,
  `fermeture_bloquants.tsv`).
- Arrêts de record : `Complete` 452 376, `UnsupportedComponent` 11 379, `InvalidArchetype` 3 107,
  `Truncated` 2 004, `RuntimeContextUnavailable` 10. Premiers composants bloquants (bornes basses,
  25 premiers par film) : `low-frequency` 1 616, `managed-object-navpoint` 925,
  `managed-object-networked-property` 760, `effect-state-data` 653, `forge-engine-forge-mode` 583,
  `game-engine-soft-ceilings` 554 — tous `non_porte` chez nous sauf `effect-state-data` (`partiel`).
- Images-clés : 316 tables, 632 records, 316 `Desync` sur `tacmap-mapdismissallock`.
- Datums : 2 588 356 entrées. Résumé : 10 paquets type 9, 3 179 événements déclarés.
- Partition `source_coverage` (2 234,5 Mbit) : `Fields` 41,8 %, `Opaque` 4,7 %, `Padding` 2 212 bits,
  `Unparsed` 53,5 %. Par type : t0 `Fields` 9,2 % / `Opaque` 0,2 % / `Unparsed` 90,6 % (717,9 Mbit) ;
  t1 100 % `Fields` (867,2 Mbit) ; t2 0,2 % `Fields` (433,1 Mbit) ; t8 100 % `Unparsed` (112,9 Mbit) ;
  chunk 3 100 % `Opaque` hors le compte (75,3 Mbit). Le total est gonflé par le bloc de type 1.
- Runtime (`LU_RUNTIME=1`) : `0797ce72` 0,13 s, 465 Mio, 118 773 événements, 220 résumés (220
  déclarés) ; `4f77afc1` 0,13 s, 436 Mio, 128 499 événements, 844 résumés (844 déclarés).

### 8.2 Biais de comparaison (à ne pas arrondir en faveur de l'un ou l'autre)

1. **« Fermé » n'existe pas chez le port** : j'ai appliqué notre oracle à sa sortie. Ses
   « trois vues complètes » (71,9 %) ne sont pas une fermeture : 92 % d'entre elles ne ferment pas.
2. **Contexte de schéma** : le port lie 632 records d'image-clé en tout, rien du type 1, pas
   d'anticipation, pas de localisation de liste ; Go lie par balayage d'ancres, `TableDeDatums`,
   table anticipée, `debutDeLaListe`. [S] C'est la cause principale de l'écart, pas la grammaire.
3. **Vue A** : le port lit séquentiellement et refuse (code 0, code 15 sans porte, codes non
   portés) ; Go localise par recherche. « Refusée » ≠ « liste non localisée ».
4. **Records utiles** : le port ne lit qu'une fraction des records (arrêt au premier échec) ;
   l'archétype des DELTA est reconstruit par mon instrument (liaisons NEW + images-clés, puis
   appariement nom/index au registre) ; dénominateurs incomparables (87 864 contre 2 187 577).
5. **Profil** : port = `DecodeProfile::default()`, `id_low_bits` 13 ; instrument Go = profil par
   défaut + largeurs d'axe lues dans le film (`largeurs_lues = true`), `IDLowBits` 13 par défaut
   ([G] `frame_records.go:144`).
6. **Grammaire** : port épinglé au 22/09 + lecteurs du 26/09 ; 43 commits depuis (§6) — effet sur la
   fermeture Go presque nul, donc biais faible ici.
7. **Temps et mémoire** : le port s'arrête tôt dans la plupart des paquets ; ses 0,15-0,31 s et
   230-416 Mio SOUS-ESTIMENT une traversée complète. L'instrument Go fait bien plus de travail
   (2,9 à 22,6 s, 71 à 176 Mio, `fermeture_films.tsv`) : pas de comparaison de performance honnête.

---

## 9. Ce qu'on peut leur prendre, ce qu'il ne faut pas reprendre, ce qu'on peut leur proposer

### 9.1 À prendre (idées, en Go)

| # | Idée | Gain attendu chez nous |
|---|---|---|
| 1 | **Qualifier la fin de vue B et la forme de la vue C dans la carte de fermeture** (instrument seul) : fin `Marqueur` / `Rejet` (hors datum, autre vue, par anticipation) ; vue C vide / n entrées ; reste du payload ; pour un rejet, état du slot dans le bloc de type 1 du chunk (vivant / trace / vide, comme `TestBloc521Rejets`) ; et le DERNIER composant lu avant la fin faible. | Casser la cause n° 1 (264 757 paquets, borne haute 2,7 M records utiles) en causes localisées par composant. Un rejet sur une entrée de datum vide prouve un désalignement en amont, et désigne le record qui le précède. Coût : quelques champs dans `paquetMarche`, aucune sortie de production. |
| 2 | **Partition de couverture par paquet** (Fields / Opaque / Padding / Unparsed, dérivée de la structure, refus des chevauchements) | Un T1 bis pour la spec : « tout bit classé », progression mesurable même sans fermeture, débordements visibles. |
| 3 | **Mode borné d'instrument** (tentative hors source = `Indisponible`, jamais 0) en A/B contre la sémantique « bourrage » | Compter les records/valeurs qui dépendent de bits hors payload et les NEW liés sur corps bourré ; trancher la divergence n° 1 (aucune preuve Ghidra du bourrage dans le dépôt). |
| 4 | **Compte déclaré du chunk 3** (paquet type 9, u32 BE) comparé aux événements trouvés par `ParseHighlightEvents` | Oracle de complétude gratuit du kill-feed et du fil des morts, par film (le port : 220/220, 844/844 sur les deux films mesurés). |
| 5 | **Arrêts typés riches dans la structure** (`RuntimeContextUnavailable{champ}`, `GenerationMismatch`, `Rejected{entête}`) | Forme IR naturelle de P2/P4 : un paramètre « supposé » non appliqué devient une queue opaque à cause nommée. |
| 6 | **`gate15` et `+0x818` comme paramètres d'en-tête** avec provenance (« calibré » par film pour `gate15`, « supposé » pour `+0x818`) | P4 tenu ; `pickGate15` sort de `killsource` vers la structure. |
| 7 | Validation de l'enveloppe (`Packet::payload` relit les 16 octets d'en-tête) | Mineur, ratchet de cohérence. |

### 9.2 À ne pas reprendre

- **La matérialisation** (`String` + `Vec<u8>` par champ, deux copies par chunk, film entier en
  mémoire) : contraire à P9, mesurée (§3.1).
- **L'état de décodage privé** : P5 le veut exposé ; c'est lui qui explique la fermeture.
- **Le refus sans repli** des paramètres hors flux (`gate15`, `+0x818`) : il coupe tout le reste du
  paquet ; notre doctrine (repli nommé, compté, prouvé par la fermeture, retiré sur relevé Ghidra)
  est meilleure pour le produit.
- **La clé de version majeure seule et l'empreinte inconnue tolérée** : contraire à D-3/D-4.
- **« Trois vues complètes » comme succès** : sans oracle de fin de paquet, c'est un faux positif
  dans 92 % des cas mesurés. (Notre « vue C atteinte » a le même défaut si on la lit seule.)
- Le code lui-même : le dépôt est Go/TS uniquement.

### 9.3 Oracle croisé (spec T7, question ouverte 6)

**[É]** Le port n'est pas un oracle indépendant de LECTEUR (il porte notre code et se valide contre
lui). Il ne peut pas servir d'oracle de FERMETURE tant qu'il n'a ni notion de fermeture ni notre
contexte (5,5 % mesuré). Ce qu'un échange apporterait réellement :
- pour nous : un détecteur de dérive involontaire de nos lecteurs (rejouer leur harnais à la tête
  et comparer les bornes sous contexte fourni) et un second compteur de couverture de bits ;
- pour eux (décision de l'utilisateur, Q6) : l'oracle `vueCFermee`, la carte de fermeture par
  film, le contexte de production (table anticipée, `TableDeDatums`, `debutDeLaListe`, bloc de
  type 1), `DecodeRoster` (type 8), `pickGate15`, `ecs_table.tsv`, et la liste des 13 exceptions
  datées — avec la mise en garde que ces lecteurs contredisent l'écrivain.

---

## 10. Reproduire

- Clone : `…/scratchpad/halo_api` (HEAD `0aaa05b`), exemple `examples/levelup_closure.rs`.
- Build : `CARGO_TARGET_DIR=<clone>/target cargo build --release --example levelup_closure`.
- Exécution (un film) : `target/release/examples/levelup_closure.exe <LevelUp>/data/cache/film_chunks/<id> <LevelUp>/data/cache/film_manifests/<id>.json <LevelUp>/apps/go-api/internal/games/halo_infinite/film/internal/grammar/testdata/ecs_table.tsv` ; `LU_RUNTIME=1` ajoute `TheaterRuntime::load`.
- Sorties brutes : `…/scratchpad/exp/<id>.txt` (et `<id>_rt.txt` pour le runtime) ; listes de noms :
  `exp/rust_known_names.txt`, `exp/ecs_names.txt`.
- Aucun fichier de LevelUp n'a été modifié ; aucune commande Go n'a été lancée.
