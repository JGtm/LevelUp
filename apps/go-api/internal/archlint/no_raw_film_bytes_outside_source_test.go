package archlint

// no_raw_film_bytes_outside_source_test.go — UNE SEULE PORTE AUX OCTETS D UN FILM
// (item 2.4.3 du PLAN_DECODEUR_FILM_2026-09-13 ; ADR 0034 D-2).
//
// # POURQUOI CE RATCHET EXISTE, ET POURQUOI AVANT LE LOT 2.4
//
// ADR 0034 D-2 : seule la couche `source` touche les octets d un film. Tout le reste recoit
// des valeurs deja lues. A la pose de ce ratchet c etait faux — SEPT lecteurs de bits distincts
// vivaient dans NEUF paquets, et chacun reposait sur sa propre idee du bourrage, du
// debordement et de l ordre des bits. Le lot 2.4 les a ramenes a une facade unique
// (`film.Source`, nee dans `internal/games/halo_infinite/film/internal/source/`, decision V15 (1)).
//
// # L ALLOWLIST N EXISTE PLUS (lot 2.5.e, 2026-09-16)
//
// Les NEUF dernieres entrees etaient les deux lecteurs restes dans `internal/analysis` racine
// — `scanEvents` du parseur de temps forts, la copie de `bitAt` d `analysis/positions` — et
// leurs consommateurs d octets. Le lot 2.5.e les a fait DESCENDRE (decision V15 (4)) :
//
//	`analysis/highlight_event_parser.go`  -> `film/grammar/highlight_events.go`, zlib par
//	                                         [source.Decompresser], octet au bit par
//	                                         [source.OctetAuBit], entiers par [source.U16LE] /
//	                                         [source.U32BE]
//	`analysis/weapon_scanner.go`          -> `film/grammar/weaponscan`
//	`analysis/positions/`                 -> `film/grammar/positions`, `bitAt` par
//	                                         [source.BitAt], en-tete de bloc par [source.U16LE]
//	`analysis/weapon_data.go`             -> `games/weapons/filmshell` (hors des racines : il
//	                                         NOMME les armes, il ne lit aucun octet)
//	`sync/killcollector/shots.go`         -> `filmshell.IDFromBytes` et `bits.ReverseBytes64`
//	`cmd/diag_film`                       -> `filmshell.IDFromBytes`
//
// L allowlist s est donc videe, et LE MECANISME QUI LA PORTAIT EST PARTI AVEC ELLE — meme
// geste qu au lot 2.5.a pour la tolerance de lieu du ratchet des couches, et pour la meme
// raison : une table vide qu on garde « au cas ou » invite a la remplir, alors qu une
// tolerance neuve doit etre une DECISION ecrite. Il ne reste que les EXCLUSIONS, qui disent
// « ces octets ne sont pas ceux d un film », pas « cette violation attend son portage ».
//
// Ce ratchet avait ete pose AVANT le lot 2.4, et c est ce qui l a rendu utile : son allowlist
// ETAIT la liste de travail, elle s est videe a mesure, et elle rougissait des qu une entree
// devenait perimee. Pose apres, il n aurait rien garde pendant le seul moment ou l on casse
// des choses.
//
// # CE QU EST UNE « LECTURE D OCTETS BRUTS », MECANIQUEMENT (cinq motifs)
//
//	motifLecteur      construction ou declaration d un lecteur de bits : `NewBitReader(...)`,
//	                  `newBitReader(...)`, `&evReader{...}`, et les primitives
//	                  `bitAt` / `bits32` / `bitsN` / `bitsWide` / `wordBitsAt` /
//	                  `readBitsBE` / `readByteAtBit` / `readU64LEAtBit` / `scanEvents`.
//	                  PAS la simple mention d un type : `func decodeX(br *BitReader)` fait
//	                  circuler un lecteur construit ailleurs, il n ouvre aucune porte. Sans
//	                  cette nuance le ratchet comptait 90 fichiers de `grammar` qui ne lisent
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
//	                  source, et rien qu a lui (`source.Inflate`).
//
// # LA CIBLE
//
// La couche `source` — aujourd hui `internal/games/halo_infinite/film/internal/source/`, demain
// `film/internal/source` (lot 2.5.a) — est le SEUL lieu autorise. Elle est exclue du balayage.
// Tout autre site des racines surveillees est une violation, sans exception : l allowlist
// datee a ete supprimee avec sa derniere entree au lot 2.5.e.
//
// # POURQUOI LES FICHIERS DE PRODUCTION SEULS (PAS LES `_test.go`)
//
// La porte aux octets est une frontiere de PRODUCTION : ce qu on interdit, c est qu un chemin
// servi a l utilisateur decode des bits sans passer par la source. Les tests du decodeur, eux,
// lisent des octets par CONSTRUCTION — c est leur travail : `source/source_test.go`
// compare les deux marcheurs de paquets, les tests de `grammar` batissent des `BitReader` sur
// des chaines forgees pour prouver une largeur, et les 23 fichiers `//go:build research` des
// racines (TOUS des `_test.go`, mesure du 2026-09-17) mesurent la grammaire sur des films
// reels. Interdire ces lectures interdirait les preuves, et aurait rempli l allowlist de
// centaines d entrees que 2.4 n aurait jamais videes. La regle de D9 est l inverse (elle parse les
// tests) parce qu elle garde un GRAPHE D IMPORTS, que les tests ferment aussi surement.
//
// # L INVENTAIRE RE-MESURE, ET L ECART AVEC LA NOTE DE PREPARATION
//
// `.ai/PREPARATION_M2_PAS_4_A_6_2026-09-17.md` §1.1 annonce 6 lecteurs de bits, 9 paquets et
// environ 120 sites. Re-mesure du 2026-09-17 sur `24b67e339`, par ce balayage : **77 couples
// (fichier, motif) de production**, tous mis en allowlist ce jour-la. Les deux comptes ne mesurent
// pas la meme chose (la note compte les SITES, ce ratchet compte les couples fichier x motif,
// et il ne compte pas la circulation d un lecteur deja construit), mais deux ecarts sont
// REELS :
//
//   - un SEPTIEME lecteur de bits, absent de la note : `internal/analysis/positions/` porte sa
//     propre copie de `bitAt` (`positions.go:179`) et son propre marcheur de paquets 16 octets
//     (`positions.go:127-128`). Consigne en §4 du plan.
//   - `objectives/statborg.go` appelle `readBitsBE` 13 fois ; la note ne citait que
//     `film.go` pour ce lecteur. Le paquet en a donc deux fichiers, pas un.
//
// Trois faux positifs de la note, mecaniquement ecartes ici et non par une exception :
// `filmcache.go:116` (`s.chunks[i]` est un `[]ChunkMeta`), `offline_biped_band.go:163`
// (`chunks []int`), `registre_killsource.go:259` (`"func evBody(r *curseurEv"` est une CHAINE).
//
// # L ALLOWLIST QU IL A EUE, ET COMMENT ELLE S EST VIDEE
//
// 78 entrees a la pose : 77 le 2026-09-17, une par couple (fichier, motif), plus une le
// 2026-09-18 a la fusion du lot 2.3 (`filmdec/film_context.go`). IL N EN RESTE AUCUNE :
// soixante-neuf sont tombees aux lots 2.4.1 et 2.4.2, les NEUF dernieres au lot 2.5.e, chacune
// dans le commit qui a fait son portage. L historique qui suit reste ecrit parce qu il dit
// CE QUI A ETE DEPLACE, et ou le chercher.
//
//	2.4.1 (6)   VIDEE le 2026-09-18. `killsource.evReader` est absorbe par le lecteur canonique
//	            de la couche source ([source.Bits]) ; le type, sa structure et les
//	            primitives `bitAt` / `bits32` / `bitsN` / `bitsWide` sont SUPPRIMES. Ce que le
//	            drapeau `over` gardait reste au marcheur de chaine (`killsource.curseurEv`), qui
//	            teste `Remaining()` avant chaque lecture. Equivalence bit a bit prouvee appel
//	            par appel sur 109 168 positions reelles des dix bobines versionnees
//	            (`killsource/equivalence_lecteur_test.go`) et de bout en bout par le golden des
//	            triplets fige AVANT l absorption (`chaines_evenements_test.go`).
//	2.4.2 (63)  VIDEE le 2026-09-18. La facade EST `internal/games/halo_infinite/film/internal/source` (V15 (1)), et
//	            elle porte desormais TOUT ce qui touche un octet de film :
//	              - le lecteur canonique [source.Bits] et ses quatre conventions de bord
//	                nommees (`BitsAt`, `BitAt`, `BitsTolerants`, `BitsTronques`) ;
//	              - les entiers du film (`U16LE` / `U32LE` / `U64LE`), l octet et le u64 a
//	                offset BIT (`OctetAuBit`, `U64LEAuBit`), le balayage de motif
//	                (`ChercherMotif64`) ;
//	              - LE marcheur de paquets ([source.Paquets]) : les QUATRE copies de
//	                l en-tete de seize octets (`grammar.WalkPackets`, `weaponv3/timing.go`,
//	                `cmd/rdata_weapon_scan`) n en sont plus que des traductions, et le temoin
//	                `source.TestDeuxMarcheursDePaquetsSAccordent` oppose les deux grammaires
//	                sur des chunks reels ;
//	              - LE decompresseur, en deux contrats ecrits : `Inflate` (tolerant, un chunk
//	                peut etre deja clair) et `Decompresser` (strict, un telechargement CDN doit
//	                etre du zlib).
//	            `grammar.BitReader` / `NewBitReader` ont DISPARU : le type s appelle `Lecteur`,
//	            il EMBARQUE `*source.Bits` et n ajoute que la grammaire (profil, capture,
//	            observateur, `ReadSignedVarWidth`). Les deux anciens noms restent listes
//	            ci-dessous, en RATCHET ANTI-RESURRECTION.
//	2.5.e (9)   VIDEE le 2026-09-16, et le MECANISME retire avec elle (cf. l en-tete). Descente
//	            de la grammaire de film posee dans `internal/analysis` racine (V15 (2) et
//	            V15 (4)) : `highlight_event_parser.go`, `weapon_scanner.go`, `weapon_data.go`,
//	            `positions/positions.go`, plus leurs deux consommateurs d octets
//	            `sync/killcollector/shots.go` et `cmd/diag_film`.
//
// V15 (2) avait envisage un PERIMETRE DECLARE qui aurait rendu trois de ces fichiers
// invisibles au ratchet ; ils sont au contraire restes DANS les racines surveillees jusqu a
// leur portage. C etait le bon choix : un perimetre declare n aurait rien garde contre leur
// croissance, alors qu une entree datee disait ce qu il restait a faire et rougissait le jour
// ou c etait fait.
//
// # MUTATIONS QUI DOIVENT LE FAIRE ROUGIR
//
// Les trois suivantes ont ete JOUEES le 2026-09-17, rouges, puis retirees :
//
//   - un fichier jetable `internal/replaybuild/mutation_jetable.go` (paquet surveille, zero
//     entree d allowlist a l epoque) portant `func bitAt(d []byte, p int) int` ET
//     `type lecteurNeuf struct{ pl []byte; bp int }` : DEUX violations, `lecteur-de-bits` et
//     `type-lecteur-de-bits`. La seconde est la preuve que le motif structurel attrape un
//     lecteur invente sous un nom que personne n a liste ;
//   - une entree d allowlist sans violation reelle
//     (`internal/replaybuild/facts_file.go | lecteur-de-bits`) : « entree perimee, la
//     retirer », par le test de peremption de l allowlist ;
//   - retirer une entree d allowlist sans faire le portage : la violation correspondante
//     rougit (c est le meme chemin de code que la premiere mutation).
//
// LES TROIS ONT ETE REJOUEES LE 2026-09-18 A LA CLOTURE DU LOT 2.4 (rouges, retirees), sur la
// table REDUITE a ses neuf entrees. Les deux dernieres n ont plus d objet depuis le lot 2.5.e,
// qui a retire l allowlist ET son mecanisme : LA MUTATION QUI VAUT DESORMAIS EST LA PREMIERE,
// et elle a ete REJOUEE le 2026-09-16 sur l arborescence sans allowlist — le fichier jetable
// rougit sur ses DEUX motifs, et plus aucune ligne ne peut le taire.

import (
	"sort"
	"strings"
	"testing"
)

// Les cinq motifs. Ce sont les cles sous lesquelles une violation est rapportee.
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
	{chemin: "internal/games/halo_infinite/film/internal/source", raison: "LA COUCHE SOURCE : le seul lieu " +
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

// TestAucuneLectureDOctetsBrutsHorsDeLaSource : hors de la couche `source`, aucun site des
// racines surveillees ne lit d octets de film — sauf les couples dates ci-dessus.
func TestAucuneLectureDOctetsBrutsHorsDeLaSource(t *testing.T) {
	sites, fichiers := balayerLecturesBrutes(t)
	if fichiers < plancherFichiersOctets {
		t.Fatalf("balayage muet : %d fichiers .go de production parcourus, plancher %d. Les "+
			"racines ont bouge ou un filtre est casse — ce ratchet ne garde plus rien et doit "+
			"echouer bruyamment.", fichiers, plancherFichiersOctets)
	}
	violations := map[string]string{}
	for _, s := range sites {
		cle := cleLectureBrute(s.fichier, s.motif)
		if violations[cle] != "" {
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
		"recopier `bitAt`) est exactement ce que les lots 2.4 et 2.5.e ont supprime.\n"+
		"IL N Y A PLUS D ALLOWLIST : elle s est videe au lot 2.5.e avec sa derniere entree, et "+
		"le mecanisme qui la portait est parti avec elle. Une tolerance re-devient une "+
		"DECISION a ecrire, pas une ligne a remplir.",
		len(lignes), strings.Join(lignes, "\n  "))
}

// TestExclusionsDesLecturesBrutesSontMotiveesEtVivantes : ce ratchet N A PLUS D ALLOWLIST
// depuis le lot 2.5.e. Il ne lui reste que ses EXCLUSIONS, qui ne disent pas « cette violation
// attend son portage » mais « ces octets ne sont pas ceux d un film » — et celles-la doivent
// quand meme designer un repertoire qui EXISTE et porter leur raison : une exclusion muette,
// ou qui pointe un paquet deplace, ouvrirait une arborescence entiere sans que personne le
// voie.
func TestExclusionsDesLecturesBrutesSontMotiveesEtVivantes(t *testing.T) {
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
				"`source`).", ex.chemin)
		}
	}
}
