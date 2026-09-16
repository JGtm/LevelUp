package archlint

// no_raw_film_bytes_outside_source_test.go — UNE SEULE PORTE AUX OCTETS D UN FILM
// (item 2.4.3 du PLAN_DECODEUR_FILM_2026-09-13 ; ADR 0034 D-2).
//
// # POURQUOI CE RATCHET EXISTE, ET POURQUOI AVANT LE LOT 2.4
//
// ADR 0034 D-2 : seule la couche `source` touche les octets d un film. Tout le reste recoit
// des valeurs deja lues. A la pose de ce ratchet c etait faux — SEPT lecteurs de bits distincts
// vivaient dans NEUF paquets, et chacun reposait sur sa propre idee du bourrage, du
// debordement et de l ordre des bits. Le lot 2.4 les ramene a une facade unique (`film.Source`,
// qui nait dans `internal/analysis/filmsource/`, decision V15 (1)) : le lot 2.4.1 en a supprime
// DEUX (`killsource.evReader` et les primitives de position de `killsource`), et la lecture par
// mot de `filmdec` est descendue dans la couche source avec eux.
//
// Ce ratchet est pose AVANT ce lot, et il vaut DEJA : son allowlist EST la liste de travail de
// 2.4, elle se vide a mesure, et elle rougit des qu une entree devient perimee. Pose apres, il
// n aurait rien garde pendant le seul moment ou l on casse des choses.
//
// # CE QU EST UNE « LECTURE D OCTETS BRUTS », MECANIQUEMENT (cinq motifs)
//
//	motifLecteur      construction ou declaration d un lecteur de bits : `NewBitReader(...)`,
//	                  `newBitReader(...)`, `&evReader{...}`, et les primitives
//	                  `bitAt` / `bits32` / `bitsN` / `bitsWide` / `wordBitsAt` /
//	                  `readBitsBE` / `readByteAtBit` / `readU64LEAtBit` / `scanEvents`.
//	                  PAS la simple mention d un type : `func decodeX(br *BitReader)` fait
//	                  circuler un lecteur construit ailleurs, il n ouvre aucune porte. Sans
//	                  cette nuance le ratchet comptait 90 fichiers de `filmdec` qui ne lisent
//	                  rien eux-memes (mesure du 2026-09-17).
//	motifTypeLecteur  une STRUCTURE portant a la fois un champ `[]byte` et un champ de
//	                  position EN BITS (`bp`, `bitPos`, `bitOff`, ...). C est le motif
//	                  structurel : un septieme lecteur invente demain sous un nom neuf rougit
//	                  quand meme. Une position en OCTETS (`pos`) ne compte pas — sinon tout
//	                  decodeur d octets du depot (`replay/mapvar/cb2.go`) rougirait.
//	motifChunk        indexation ou tranchage d une valeur nommee `chunk` / `chunks` /
//	                  `chunk0` DONT LE TYPE DECLARE dans le paquet est `[]byte` ou
//	                  `[][]byte`. Le controle de type evite les faux positifs reels :
//	                  `filmcache.source.chunks` est un `[]ChunkMeta`,
//	                  `filmdec/offline_biped_band.go` a un `chunks []int`.
//	motifBinaire      `binary.LittleEndian` / `binary.BigEndian`. Regle DE PERIMETRE : dans
//	                  les racines ci-dessous, les seuls octets qu on lit sont ceux d un film ;
//	                  les fichiers d installation du jeu sont declares hors racines. Mesure du
//	                  2026-09-17 : aucun usage non lie au film dans le perimetre.
//	motifInflate      import de `compress/zlib` ou `compress/flate`. La decompression est au
//	                  source, et rien qu a lui (`filmsource.Inflate`).
//
// # LA CIBLE
//
// La couche `source` — aujourd hui `internal/analysis/filmsource/`, demain
// `film/internal/source` (lot 2.5.a) — est le SEUL lieu autorise. Elle est exclue du balayage.
// Tout autre site des racines surveillees est une violation, toleree seulement par une entree
// datee de `lecturesTolerees` qui nomme le lot qui la retire.
//
// # POURQUOI LES FICHIERS DE PRODUCTION SEULS (PAS LES `_test.go`)
//
// La porte aux octets est une frontiere de PRODUCTION : ce qu on interdit, c est qu un chemin
// servi a l utilisateur decode des bits sans passer par la source. Les tests du decodeur, eux,
// lisent des octets par CONSTRUCTION — c est leur travail : `filmsource/source_test.go`
// compare les deux marcheurs de paquets, les tests de `filmdec` batissent des `BitReader` sur
// des chaines forgees pour prouver une largeur, et les 23 fichiers `//go:build research` des
// racines (TOUS des `_test.go`, mesure du 2026-09-17) mesurent la grammaire sur des films
// reels. Interdire ces lectures interdirait les preuves, et remplirait l allowlist de
// centaines d entrees que 2.4 ne viderait jamais. La regle de D9 est l inverse (elle parse les
// tests) parce qu elle garde un GRAPHE D IMPORTS, que les tests ferment aussi surement.
//
// # L INVENTAIRE RE-MESURE, ET L ECART AVEC LA NOTE DE PREPARATION
//
// `.ai/PREPARATION_M2_PAS_4_A_6_2026-09-17.md` §1.1 annonce 6 lecteurs de bits, 9 paquets et
// environ 120 sites. Re-mesure du 2026-09-17 sur `24b67e339`, par ce balayage : **77 couples
// (fichier, motif) de production**, tous en allowlist ci-dessous. Les deux comptes ne mesurent
// pas la meme chose (la note compte les SITES, ce ratchet compte les couples fichier x motif,
// et il ne compte pas la circulation d un lecteur deja construit), mais deux ecarts sont
// REELS :
//
//   - un SEPTIEME lecteur de bits, absent de la note : `internal/analysis/positions/` porte sa
//     propre copie de `bitAt` (`positions.go:179`) et son propre marcheur de paquets 16 octets
//     (`positions.go:127-128`). Consigne en §4 du plan.
//   - `objectiveevents/statborg.go` appelle `readBitsBE` 13 fois ; la note ne citait que
//     `film.go` pour ce lecteur. Le paquet en a donc deux fichiers, pas un.
//
// Trois faux positifs de la note, mecaniquement ecartes ici et non par une exception :
// `filmcache.go:116` (`s.chunks[i]` est un `[]ChunkMeta`), `offline_biped_band.go:163`
// (`chunks []int`), `registre_killsource.go:259` (`"func evBody(r *curseurEv"` est une CHAINE).
//
// # L ALLOWLIST, ET COMMENT ELLE SE VIDE
//
// 78 entrees a la pose : 77 le 2026-09-17, une par couple (fichier, motif), plus une le
// 2026-09-18 a la fusion du lot 2.3 (`filmdec/film_context.go`, datee sur sa ligne), chacune
// avec le lot qui la retire. La case 2.4.3 du plan ne se coche que quand il ne reste que les
// neuf entrees du lot 2.5.c (V15 (2) et (4) : elles descendent au pas 5).
//
//	2.4.1 (6)   VIDEE le 2026-09-18. `killsource.evReader` est absorbe par le lecteur canonique
//	            de la couche source ([filmsource.Bits]) ; le type, sa structure et les
//	            primitives `bitAt` / `bits32` / `bitsN` / `bitsWide` sont SUPPRIMES. Ce que le
//	            drapeau `over` gardait reste au marcheur de chaine (`killsource.curseurEv`), qui
//	            teste `Remaining()` avant chaque lecture. Equivalence bit a bit prouvee appel
//	            par appel sur 109 168 positions reelles des dix bobines versionnees
//	            (`killsource/equivalence_lecteur_test.go`) et de bout en bout par le golden des
//	            triplets fige AVANT l absorption (`chaines_evenements_test.go`).
//	2.4.2 (59)  la facade `film.Source` : `filmdec` (declaration du lecteur, 33 sites de
//	            construction, quatre sections de `chunk_00`, second marcheur de paquets),
//	            `killsource` (chunks, feed, table de joueurs, monde, walk), `objectiveevents`
//	            (film.go, statborg.go), `weaponv3` (pi_resolver, bits_word, timing),
//	            `sync/haloclient` (inflate du blob CDN) et les cinq outils `cmd/`. QUATRE de ses
//	            63 entrees sont deja tombees au 2.4.1, parce que le lecteur canonique ne
//	            pouvait pas naitre sans elles : `filmdec/bits_word.go` (la lecture par mot, qui
//	            descend en `source` sous le nom [filmsource.BitsAt] — une seule implantation,
//	            jamais deux) et ses deux derniers appelants directs, `grenade_events.go` et
//	            `keyframe_world.go`.
//	2.5.c (9)   descente de la grammaire de film posee dans `internal/analysis` racine
//	            (V15 (2) et V15 (4)) : `highlight_event_parser.go`, `weapon_scanner.go`,
//	            `weapon_data.go`, `positions/positions.go`, plus leurs deux consommateurs
//	            d octets `sync/killcollector/shots.go` et `cmd/diag_film`.
//
// V15 (2) porte ces trois-la — `highlight_event_parser.go`, `weapon_scanner.go`,
// `killcollector/shots.go` — avec la cible « pas 5 ». Ils restent DANS les racines
// surveillees : un perimetre declare qui les rendrait invisibles ne garderait rien contre leur
// croissance, alors qu une entree datee dit ce qu il reste a faire et rougit quand c est fait.
//
// # MUTATIONS QUI DOIVENT LE FAIRE ROUGIR
//
// Les trois suivantes ont ete JOUEES le 2026-09-17, rouges, puis retirees :
//
//   - un fichier jetable `internal/replaybuild/mutation_jetable.go` (paquet surveille, zero
//     entree d allowlist) portant `func bitAt(d []byte, p int) int` ET
//     `type lecteurNeuf struct{ pl []byte; bp int }` : DEUX violations, `lecteur-de-bits` et
//     `type-lecteur-de-bits`. La seconde est la preuve que le motif structurel attrape un
//     lecteur invente sous un nom que personne n a liste ;
//   - une entree d allowlist sans violation reelle
//     (`internal/replaybuild/facts_file.go | lecteur-de-bits`) : « entree perimee, la
//     retirer », par `TestAllowlistDesLecturesBrutesNEstPasPerimee` ;
//   - retirer une entree d allowlist sans faire le portage : la violation correspondante
//     rougit (c est le meme chemin de code que la premiere mutation).

import (
	"sort"
	"strings"
	"testing"
)

// Les cinq motifs. Ce sont les valeurs du champ `motif` de l allowlist.
const (
	motifLecteur     = "lecteur-de-bits"
	motifTypeLecteur = "type-lecteur-de-bits"
	motifChunk       = "indexation-de-chunk"
	motifBinaire     = "ordre-d-octets"
	motifInflate     = "decompression"
)

// plancherFichiersOctets : LE PLANCHER CONTRE UN BALAYAGE MUET. 1 084 fichiers `.go` de
// production mesures le 2026-09-17 dans les racines, exclusions deduites. Pose a 850 (environ
// 78 %) : assez serre pour qu un parcours casse echoue, assez lache pour ne pas devenir un
// compteur a maintenir. Un ratchet qui ne balaye rien passe en silence.
const plancherFichiersOctets = 850

// poseDesLecturesTolerees : la date d inscription de TOUTES les entrees de `lecturesTolerees`
// (elles ont ete mesurees d un seul coup). Une entree ajoutee plus tard porte sa propre date,
// en commentaire sur sa ligne — et doit se justifier : cette table ne grossit pas, elle se
// vide.
const poseDesLecturesTolerees = "2026-09-17"

// racinesOctetsBruts : les racines surveillees, relatives a `apps/go-api`. Ce sont les
// arborescences ou un octet de film peut apparaitre — mesure du 2026-09-17, aucun paquet hors
// de la ne lit d octets de film.
var racinesOctetsBruts = []string{
	"internal/games/halo_infinite/film",
	"internal/analysis",
	"internal/sync",
	"internal/replaybuild",
	"cmd",
}

// exclusionOctets : un repertoire hors de la regle, avec la raison. Ce n est PAS une
// allowlist : une exclusion dit « ces octets ne sont pas ceux d un film », pas « cette
// violation attend son portage ».
type exclusionOctets struct{ chemin, raison string }

var exclusionsOctetsBruts = []exclusionOctets{
	{chemin: "internal/analysis/filmsource", raison: "LA COUCHE SOURCE : le seul lieu " +
		"autorise (ADR 0034 D-2). Passe sous `film/internal/source` au lot 2.5.a ; ce " +
		"deplacement se repercute ICI, sur cette ligne."},
	{chemin: "cmd/weapon-sounds", raison: "lit les banques Wwise de l INSTALLATION DU JEU, " +
		"jamais un film."},
	{chemin: "cmd/weapon-icons-build", raison: "lit les modules et les textures BC7 de " +
		"l INSTALLATION DU JEU (son propre `bitReader` de `bc7.go` decode un bloc de " +
		"texture), jamais un film."},
	{chemin: "internal/games/halo_infinite/film/replay/mapvar", raison: "lit les VARIANTES " +
		"DE CARTE `.mvar` (Bond CompactBinary v2, stockage UGC), jamais les chunks d un " +
		"film. Le paquet vit sous `film/` par proximite d usage, pas par format."},
}

// constructeursDeLecteurDeBits : les fonctions dont l APPEL ou la DECLARATION ouvre la porte
// aux octets. Six lecteurs distincts y sont nommes (note §1.1) plus le septieme trouve a la
// re-mesure (`analysis/positions`, qui recopie `bitAt`).
var constructeursDeLecteurDeBits = map[string]bool{
	"NewBitReader": true, "newBitReader": true, "newEvReader": true,
	"bitAt": true, "bits32": true, "bitsN": true, "bitsWide": true, "wordBitsAt": true,
	"readBitsBE": true, "readByteAtBit": true, "readU64LEAtBit": true,
	"scanEvents": true,
}

// typesDeLecteurDeBits : les types dont la DECLARATION ou la construction par litteral ouvre
// la porte. Leur simple mention en parametre ne compte pas (cf. l en-tete).
var typesDeLecteurDeBits = map[string]bool{
	"BitReader": true, "bitReader": true, "evReader": true,
}

// nomsDePositionEnBits : les noms de champ qui disent « position EN BITS ». `pos` en est
// volontairement absent : une position en octets ne fait pas un lecteur de bits.
var nomsDePositionEnBits = map[string]bool{
	"bp": true, "bitpos": true, "bitoff": true, "bitposition": true, "posbits": true,
	"bitindex": true,
}

// entiersDePosition : les types entiers qu une position peut porter.
var entiersDePosition = map[string]bool{
	"int": true, "int32": true, "int64": true, "uint": true, "uint32": true, "uint64": true,
}

// nomsDeChunk : les noms sous lesquels les octets d un chunk circulent.
var nomsDeChunk = map[string]bool{"chunk": true, "chunks": true, "chunk0": true}

var ordresDOctets = map[string]bool{"LittleEndian": true, "BigEndian": true}

var paquetsDeDecompression = map[string]bool{"compress/zlib": true, "compress/flate": true}

// lectureToleree : un couple (fichier, motif) qui viole la regle AUJOURD HUI, avec le lot qui
// le retire. Posee le `poseDesLecturesTolerees`. C est la liste de travail de 2.4, pas un
// blanc-seing : le test refuse toute entree devenue sans objet.
type lectureToleree struct {
	fichier string
	motif   string
	lot     string
}

// lecturesTolerees — LES 77 COUPLES MESURES LE 2026-09-17 sur `24b67e339`, plus le couple ne de la
// fusion du lot 2.3 (2026-09-18), MOINS les DIX retires par le lot 2.4.1 (2026-09-18) : il en
// reste 68. Chacun disparait dans le commit qui fait le portage ; la case 2.4.3 du plan se coche
// quand il ne reste que les neuf entrees du lot 2.5.c.
var lecturesTolerees = []lectureToleree{
	{fichier: "cmd/diag_film/main.go", motif: motifBinaire, lot: "2.5.c"},
	{fichier: "cmd/diag_weapons_v3/positions.go", motif: motifInflate, lot: "2.4.2"},
	{fichier: "cmd/fetch_film_chunks/main.go", motif: motifInflate, lot: "2.4.2"},
	{fichier: "cmd/rdata_weapon_scan/main.go", motif: motifInflate, lot: "2.4.2"},
	{fichier: "cmd/rdata_weapon_scan/main.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "cmd/rdata_weapon_scan/main.go", motif: motifBinaire, lot: "2.4.2"},
	{fichier: "cmd/replay-worker/job.go", motif: motifInflate, lot: "2.4.2"},
	{fichier: "internal/analysis/highlight_event_parser.go", motif: motifInflate, lot: "2.5.c"},
	{fichier: "internal/analysis/highlight_event_parser.go", motif: motifLecteur, lot: "2.5.c"},
	{fichier: "internal/analysis/highlight_event_parser.go", motif: motifBinaire, lot: "2.5.c"},
	{fichier: "internal/analysis/objectiveevents/film.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/analysis/objectiveevents/statborg.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/analysis/positions/positions.go", motif: motifLecteur, lot: "2.5.c"},
	{fichier: "internal/analysis/positions/positions.go", motif: motifBinaire, lot: "2.5.c"},
	{fichier: "internal/analysis/weapon_data.go", motif: motifBinaire, lot: "2.5.c"},
	{fichier: "internal/analysis/weapon_scanner.go", motif: motifBinaire, lot: "2.5.c"},
	{fichier: "internal/analysis/weaponv3/bits_word.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/analysis/weaponv3/bits_word.go", motif: motifBinaire, lot: "2.4.2"},
	{fichier: "internal/analysis/weaponv3/pi_resolver.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/analysis/weaponv3/pi_resolver.go", motif: motifBinaire, lot: "2.4.2"},
	{fichier: "internal/analysis/weaponv3/timing.go", motif: motifBinaire, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/ability_rank.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/biped_creation.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/biped_pickups.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/bitreader.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/default_state.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/equipment_creation.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/equipment_state.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/event_list.go", motif: motifLecteur, lot: "2.4.2"},
	// 2026-09-18, fusion du lot 2.3 : `FilmContext.NouveauLecteur` construit LE lecteur qui porte le
	// profil du contexte (plus de variable de paquet). C est la forme que la facade 2.4.2 absorbe :
	// le lecteur canonique naitra dans `film.Source`, contexte compris, et ce site disparait avec lui.
	{fichier: "internal/games/halo_infinite/film/filmdec/film_context.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/film_format_version.go", motif: motifChunk, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/film_format_version.go", motif: motifBinaire, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/film_identity.go", motif: motifChunk, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/film_identity.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/film_identity.go", motif: motifBinaire, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/film_major_version.go", motif: motifChunk, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/film_major_version.go", motif: motifBinaire, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/film_packets.go", motif: motifBinaire, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/fire_aim_modal.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/frame_chain_infer.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/frame_harvest.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/frame_infer.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/frame_records.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/ground_weapon_ammo.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/keyframe_entity_queue.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/keyframe_fullstate_loop.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/keyframe_record_walk.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/navpoint_radial_scan.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/object_deaths_march.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/objective_scan.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/offline_aim_only.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/offline_biped.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/player_table.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/player_table_record.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/player_teams.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/probe_export.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/registry.go", motif: motifBinaire, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/transloc_events.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/weapon_hits.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/zone_state_scan.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/filmdec/zoom_events.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/killsource/chunks.go", motif: motifChunk, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/killsource/feed.go", motif: motifChunk, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/killsource/film_table.go", motif: motifChunk, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/killsource/walk.go", motif: motifLecteur, lot: "2.4.2"},
	{fichier: "internal/games/halo_infinite/film/killsource/world.go", motif: motifChunk, lot: "2.4.2"},
	{fichier: "internal/sync/haloclient/halo_client_http.go", motif: motifInflate, lot: "2.4.2"},
	{fichier: "internal/sync/killcollector/shots.go", motif: motifBinaire, lot: "2.5.c"},
}

// TestAucuneLectureDOctetsBrutsHorsDeLaSource : hors de la couche `source`, aucun site des
// racines surveillees ne lit d octets de film — sauf les couples dates ci-dessus.
func TestAucuneLectureDOctetsBrutsHorsDeLaSource(t *testing.T) {
	sites, fichiers := balayerLecturesBrutes(t)
	if fichiers < plancherFichiersOctets {
		t.Fatalf("balayage muet : %d fichiers .go de production parcourus, plancher %d. Les "+
			"racines ont bouge ou un filtre est casse — ce ratchet ne garde plus rien et doit "+
			"echouer bruyamment.", fichiers, plancherFichiersOctets)
	}
	tolerees := indexDesLecturesTolerees()
	violations := map[string]string{}
	for _, s := range sites {
		cle := cleLectureBrute(s.fichier, s.motif)
		if tolerees[cle] || violations[cle] != "" {
			continue
		}
		violations[cle] = s.detail
	}
	if len(violations) == 0 {
		return
	}
	lignes := make([]string, 0, len(violations))
	for cle, detail := range violations {
		lignes = append(lignes, cle+"   vu : "+detail)
	}
	sort.Strings(lignes)
	t.Errorf("%d lecture(s) d octets bruts de film hors de la couche source (ADR 0034 D-2) :"+
		"\n  %s\n"+
		"QUOI FAIRE : passer par la facade `film.Source` — elle ouvre le film, decompresse, "+
		"decoupe les paquets et rend LE lecteur de bits canonique ; le code appelant recoit "+
		"des valeurs deja lues, jamais un `[]byte` de chunk. Ecrire un lecteur de plus (ou "+
		"recopier `bitAt`) est exactement ce que le lot 2.4 supprime.\n"+
		"Ajouter une entree a `lecturesTolerees` N EST PAS une reponse : cette table est datee "+
		"du %s, elle recense le travail que 2.4 doit faire, et elle se VIDE.",
		len(lignes), strings.Join(lignes, "\n  "), poseDesLecturesTolerees)
}

// TestAllowlistDesLecturesBrutesNEstPasPerimee : une entree qui ne decrit plus une violation
// reelle se RETIRE, dans le commit meme qui la resout. Une allowlist perimee finit par
// autoriser autre chose que ce qu elle nommait.
func TestAllowlistDesLecturesBrutesNEstPasPerimee(t *testing.T) {
	sites, _ := balayerLecturesBrutes(t)
	vivantes := map[string]bool{}
	for _, s := range sites {
		vivantes[cleLectureBrute(s.fichier, s.motif)] = true
	}
	vues := map[string]bool{}
	for _, l := range lecturesTolerees {
		cle := cleLectureBrute(l.fichier, l.motif)
		if vues[cle] {
			t.Errorf("`lecturesTolerees` cite deux fois %s : doublon, en retirer un.", cle)
		}
		vues[cle] = true
		if l.lot == "" {
			t.Errorf("`lecturesTolerees` cite %s sans lot de retrait : une tolerance sans "+
				"cible est une dette anonyme.", cle)
		}
		if !vivantes[cle] {
			t.Errorf("`lecturesTolerees` cite %s (pose %s, lot %s), qui ne lit plus d octets "+
				"bruts : entree perimee, la retirer.", cle, poseDesLecturesTolerees, l.lot)
		}
	}
	verifierExclusionsDOctets(t)
}

// verifierExclusionsDOctets : une exclusion decrit un repertoire qui existe et porte une
// raison. Une exclusion muette autoriserait une arborescence entiere sans le dire.
func verifierExclusionsDOctets(t *testing.T) {
	t.Helper()
	for _, ex := range exclusionsOctetsBruts {
		if strings.TrimSpace(ex.raison) == "" {
			t.Errorf("l exclusion %q ne dit pas POURQUOI ces octets ne sont pas ceux d un "+
				"film : une exclusion muette ouvre une arborescence entiere.", ex.chemin)
		}
		if !repertoireExisteSousAPI(t, ex.chemin) {
			t.Errorf("l exclusion %q ne designe aucun repertoire : le paquet a ete deplace ou "+
				"supprime, mettre la ligne a jour (c est le cas prevu au lot 2.5.a pour "+
				"`filmsource`).", ex.chemin)
		}
	}
}

// indexDesLecturesTolerees rend l allowlist indexee par cle (fichier, motif).
func indexDesLecturesTolerees() map[string]bool {
	out := map[string]bool{}
	for _, l := range lecturesTolerees {
		out[cleLectureBrute(l.fichier, l.motif)] = true
	}
	return out
}
