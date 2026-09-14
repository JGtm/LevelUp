package filmdec

// film_identity.go — LA SECTION 2 DE `chunk_00` : QUI A ECRIT CE FILM, ET QUAND (lot 1.5.1).
//
// # D'OU VIENT CETTE GRAMMAIRE
//
// Elle est LUE chez l'ecrivain, pas devinee sur les octets. `FUN_14299b198(base, writer)` et
// `FUN_14299b278(base, writer)` serialisent la totalite de `chunk_00` ; leur desassemblage donne
// la source et la largeur en bits de chaque champ (releve du 2026-09-12, Ghidra base
// `0x140000000`, lecture seule — `NOTE_SECTION3_CHUNK00_2026-09-12.md` section B, instrument
// permanent `section3_ecrivain_research_test.go`) :
//
//	base+0x000000  0x20 bits      u32 A = FilmMajorVersion (cf. film_major_version.go)
//	base+0x000004  0x20 bits      u32 B
//	base+0x000008  0x659000 bits  LE REGISTRE (50 blocs de 0x4100 ; 49 avant HI_1_12_0)
//	base+0x0CB208  0xF60 bits     LA TABLE PAR TYPE : 123 u32 sur le build de reference
//	base+0x0CB3F4  0x100 bits     version en clair  ("6.10026.19225.0")
//	base+0x0CB414  0x100 bits     build en clair    ("HI_1_13_0")
//	base+0x0CB434  0x100 bits     saveur en clair   ("release")
//	base+0x0CB454  0x20 bits      identifiant de build (DAT_144e4ef50)
//	base+0x0CB458  0x20 bits      changelist           (DAT_144e4ef54, " changelist: %d")
//	base+0x0CB45C  UN SEUL BIT    booleen (FUN_1406d49c4) — TOUT CE QUI SUIT EST DECALE D'UN BIT
//	base+0x0CB460  0x800 bits x2  deux champs de nom de 256 octets (mesures VIDES)
//	base+0x0CB660  0x20 bits      L'HORODATAGE DU MATCH (`_time64()` dans FUN_14299b674)
//	base+0x0CB664  0x20 bits x3   trois u32 mesures nuls
//	base+0x0CB670  0x8000 bits x3 trois blocs de 4 096 octets
//	base+0x0CE670  0x80 bits x2   deux blocs de 16 octets
//	base+0x0CE690  FUN_1407ec560  LE CORPS, ou vit la table des joueurs (player_table.go)
//
// # LES OFFSETS DE LA CARTE NE VALENT QUE POUR DEUX BUILDS — TOUT SE DERIVE ICI
//
// La carte ci-dessus est celle de `HI_1_12_0` / `HI_1_13_0`. Sur les builds anterieurs le
// registre compte 49 blocs et la table par type moins d'entrees, donc TOUT ce qui les suit
// recule : la chaine de build est mesuree a `0x0C7310` (`HI_1_11_0`), `0x0C730C`
// (`HI_1_10_0` / `HI_1_9_0` / `HI_1_8_0`) et `0x0C72F8` (`HI_1_4_1`) — releve du 2026-09-12,
// `NOTE_SECTION3_SLOTS_2026-09-12.md` section D. Ce lecteur ne porte donc AUCUN offset absolu :
// il derive la fin du registre du parse (`parseRegistry`, cadrage du jeu depuis le lot 1.2),
// ancre la section sur la chaine de build — l'ancre que la recherche a suivie — et deduit tout
// le reste relativement a elle.
//
// LA FERMETURE ARITHMETIQUE QUI LE PROUVE, mesuree le 2026-09-14 sur les 1 351 `chunk_00` du
// cache : `(offset de la chaine de build - 0x20) - fin du registre` vaut 492 octets sur les
// 1 269 films de `HI_1_12_0`/`HI_1_13_0` (50 blocs), 488 sur les 39 de `HI_1_11_0`, 484 sur les
// 37 de `HI_1_10_0`/`HI_1_9_0`/`HI_1_8_0` et 464 sur `HI_1_4_1` (49 blocs) — soit EXACTEMENT
// 123 / 122 / 121 / 116 entrees de table par type, sans un octet de reste. Le 123 du build
// courant est la valeur que l'ecrivain ecrit (`MOV R9D,0xf60` = 3 936 bits = 492 octets) : la
// derivation structurelle rend le compte de l'ECRIVAIN, la ou l'heuristique de `lireEntete`
// (remonter tant que la valeur tient sur 16 bits) en comptait 124 (residu G.2 de la note).
//
// # CE LECTEUR N'A PAS DE CONSOMMATEUR, ET C'EST LE PLAN
//
// Lot 1.5 : lecteurs purs. Les consommateurs viennent aux lots 1.6 (registre d'identite),
// 1.7 (equipe) et 1.8 (kill feed). Aucun octet cuit ne change tant qu'ils ne sont pas branches.
//
// # PAS DE VERROU DE DECODAGE
//
// Ces lecteurs sont des fonctions pures de (octets) : aucune variable de paquet, aucun crochet,
// aucun etat de reglage. Ils n'ont donc pas besoin de `LockProcessDecode` (D-5 d'ADR 0034), et
// le ratchet `archlint/decode_lock_held_test.go` ne les vise pas (il porte sur les familles
// `Scan*`, `DecodeFrame*`, `TraverseEntity*`).

import (
	"encoding/binary"
	"strings"
)

// chunk00Error : une erreur sentinelle CONSTANTE des lecteurs de `chunk_00`.
//
// POURQUOI UN `const` ET PAS UN `var errors.New(...)` : meme raison que `registryError`
// (registry.go) — le ratchet `archlint/TestFilmdecPackageVarsNeCroitPas` gele l'etat global
// MUTABLE de `filmdec` a 96 noms, et une sentinelle en `const` n'en est pas un : elle ne peut
// ni etre reassignee ni etre partagee entre deux decodages. `errors.Is` fonctionne (la valeur
// est comparable et unique).
type chunk00Error string

// Error implemente error.
func (e chunk00Error) Error() string { return string(e) }

// ErrNoFilmIdentity : le `chunk_00` ne porte AUCUNE section d'identification — aucune chaine de
// build entre la fin du registre et la borne de recherche.
//
// CE N'EST PAS UN CAS THEORIQUE : 5 films du cache sont dans ce cas, mesures le 2026-09-14 et
// nommes ici parce qu'un lecteur qui rencontre cette erreur doit savoir qu'elle a une
// population connue — `03af54c3`, `13b00e35`, `47d20b5d`, `50247b26`, `a349fea8`. Leur build
// reste INCONNU (leur table des slots se comporte comme celle de `HI_1_4_1`, mais se comporter
// comme n'est pas etre : D-4 d'ADR 0034 interdit de lire un film au profil du build voisin).
const ErrNoFilmIdentity = chunk00Error(
	"filmdec: chunk_00 sans section d'identification (aucune chaine de build apres le registre)")

// ErrChunk00Truncated : le tampon s'arrete avant la fin de la section que le lecteur doit lire.
//
// Un `chunk_00` coupe ne fait JAMAIS paniquer ces lecteurs et ne rend jamais une identite
// partielle en silence (lecon du lot 1.2 : `zeroTail` paniquait sur un bloc incomplet).
const ErrChunk00Truncated = chunk00Error(
	"filmdec: chunk_00 tronque avant la fin de la section d'identification")

const (
	// identBuildPrefix : l'ancre. Toutes les chaines de build mesurees commencent par `HI_`.
	identBuildPrefix = "HI_"
	// identFieldBytes : la largeur d'un champ de chaine (`0x100` bits = 32 octets), pour les
	// trois champs contigus version / build / saveur.
	identFieldBytes = 0x20
	// identBuildIDOff / identChangelistOff / identBoolOff : relatifs a la chaine de BUILD.
	// `0x0CB454 - 0x0CB414 = 0x40`, `0x0CB458 - 0x0CB414 = 0x44`, `0x0CB45C - 0x0CB414 = 0x48`.
	identBuildIDOff    = 0x40
	identChangelistOff = 0x44
	identBoolOff       = 0x48
	// identSearchSpan : la fenetre de recherche de la chaine de build, en octets APRES la fin
	// du registre et le champ de version. Elle borne la TABLE PAR TYPE, dont l'ecrivain ecrit
	// 492 octets sur le build courant (`MOV R9D,0xf60`) ; la mesure du 2026-09-14 sur les
	// 1 351 chunk_00 du cache donne 464 a 492 octets, tous builds confondus. La fenetre est
	// posee a 4 096 octets — de la place pour 1 024 entrees de table, huit fois le cardinal
	// mesure le plus grand — pour qu'un build futur a table plus longue se lise sans toucher
	// a ce fichier, et assez etroite pour qu'un `HI_` du CORPS (bit-packe, donc fortuit) ne
	// soit jamais atteint.
	identSearchSpan = 0x1000
	// identNomBytes / identBlocBytes / identPetitBlocBytes : les champs qui separent le booleen
	// d'un bit du debut du corps, largeurs lues chez l'ecrivain (`0x800`, `0x8000`, `0x80` bits).
	identNomBytes       = 0x800 / 8
	identBlocBytes      = 0x8000 / 8
	identPetitBlocBytes = 0x80 / 8
	// identApresBoolBytes : leur somme. FERMETURE : `0x0CB45C + 12 848 = 0x0CE68C`, l'offset du
	// corps que l'ecrivain donne (`FUN_1407ec560` sur `base+0x0CE690`, moins les 4 octets que
	// le booleen d'un bit occupe dans la structure sans les consommer dans le flux).
	identApresBoolBytes = 2*identNomBytes + 4*4 + 3*identBlocBytes + 2*identPetitBlocBytes
	// identDecalageBit : LE decalage d'un bit. Apres le booleen de `0x0CB45C`, tout le reste du
	// fichier est decale d'un bit — c'est l'explication mecanique de « la section 3 est
	// amorphe » du 2026-08-30, et la raison pour laquelle aucune lecture alignee sur l'octet
	// ne rendait rien apres cet offset.
	identDecalageBit = 1
)

// FilmIdentity : la section 2 de `chunk_00`, lue champ par champ.
type FilmIdentity struct {
	// Version, Build, Flavor : les trois champs de 32 octets, en clair.
	Version string
	Build   string
	Flavor  string
	// BuildID, Changelist : les deux u32 de `0x0CB454` / `0x0CB458`, recopies par
	// `FUN_14299b674` depuis la structure d'infos de build de l'executable.
	BuildID    uint32
	Changelist uint32
	// MatchStartUnix : l'horodatage du match, `_time64()` au moment ou `chunk_00` est ecrit.
	// Mesure du 2026-09-12 contre `match_registry.start_time_utc` : +19 s, +29 s, +43 s sur
	// trois films (seuil de 120 s ecrit avant la mesure), et un seul decalage de bit sur
	// dix-sept rend une valeur plausible — celui que l'ecrivain predit.
	MatchStartUnix uint32
	// TypeVersions : la table par type, les u32 qui precedent le champ de version. Le cardinal
	// depend du build (123 / 122 / 121 / 116 mesures). Leur semantique (« version de
	// serialisation par type », appel virtuel `vtable+0x30`) est APPUYEE, pas prouvee, et ce
	// lecteur ne l'interprete pas : il rend les valeurs.
	TypeVersions []uint32
	// RegistryBlocks : le nombre de blocs du registre (49 ou 50 selon le build). C'est lui qui
	// ancre la fin du registre, donc le debut de la table par type.
	RegistryBlocks int
	// BuildOffset : l'octet de la chaine de build dans le tampon inflate. Publie parce que
	// c'est l'ancre de toute la derivation, et qu'un rapport qui le porte se relit.
	BuildOffset int
	// BodyBit : le PREMIER BIT du corps (`FUN_1407ec560`), decalage d'un bit compris. C'est la
	// borne basse de tout balayage du corps — la table des joueurs en particulier.
	BodyBit int
}

// ReadFilmIdentity lit la section 2 d'un `chunk_00` DEJA DECOMPRESSE.
//
// Erreurs typees, jamais de panique ni de lecture partielle silencieuse :
// [ErrRegistryStillCompressed] (tampon encore zlib), [ErrChunk00Truncated] (tampon coupe avant
// la fin de la section), [ErrNoFilmIdentity] (le film ne porte pas de section — 5 films du
// cache).
func ReadFilmIdentity(chunk0 []byte) (FilmIdentity, error) {
	if looksZlib(chunk0) {
		return FilmIdentity{}, ErrRegistryStillCompressed
	}
	reg := parseRegistry(chunk0)
	if reg.Truncated {
		return FilmIdentity{}, ErrChunk00Truncated
	}
	finRegistre := registryEntryBase + len(reg.Archetypes)*archetypeBlockSize
	buildOff, ok := chercherChaineBuild(chunk0, finRegistre)
	if !ok {
		return FilmIdentity{}, ErrNoFilmIdentity
	}
	corpsOctet := buildOff + identBoolOff + identApresBoolBytes
	if corpsOctet >= len(chunk0) {
		return FilmIdentity{}, ErrChunk00Truncated
	}
	id := FilmIdentity{
		Version:        chaineDeChamp(chunk0, buildOff-identFieldBytes),
		Build:          chaineDeChamp(chunk0, buildOff),
		Flavor:         chaineDeChamp(chunk0, buildOff+identFieldBytes),
		BuildID:        binary.LittleEndian.Uint32(chunk0[buildOff+identBuildIDOff:]),
		Changelist:     binary.LittleEndian.Uint32(chunk0[buildOff+identChangelistOff:]),
		TypeVersions:   lireTableParType(chunk0, finRegistre, buildOff-identFieldBytes),
		RegistryBlocks: len(reg.Archetypes),
		BuildOffset:    buildOff,
		BodyBit:        corpsOctet*8 + identDecalageBit,
	}
	id.MatchStartUnix = lireHorodatage(chunk0, buildOff)
	return id, nil
}

// lireHorodatage lit les 32 bits de `_time64()` : ils suivent le booleen d'un bit et les deux
// champs de nom de 256 octets, donc au bit `(buildOff + 0x48) * 8 + 1 + 2 * 2 048`.
func lireHorodatage(d []byte, buildOff int) uint32 {
	bit := (buildOff+identBoolOff)*8 + identDecalageBit + 2*identNomBytes*8
	return u32DuFlux(d, bit)
}

// u32DuFlux lit 32 bits MSB d'abord a `bit` et les relit comme un u32 petit-boutiste.
//
// POURQUOI CETTE DOUBLE CONVENTION, ET ELLE N'EST PAS UN BRICOLAGE : l'ecrivain de bits
// `FUN_1406d60f4` pousse les octets de la SOURCE dans l'ordre des adresses, MSB d'abord. Les
// quatre octets sortis du flux sont donc l'image memoire du u32, et se relisent en LE.
func u32DuFlux(d []byte, bit int) uint32 {
	if bit < 0 || (bit+32+7)/8 > len(d) {
		return 0
	}
	br := NewBitReader(d)
	br.SetBitPos(bit)
	var tmp [4]byte
	binary.BigEndian.PutUint32(tmp[:], uint32(br.ReadBits(32)))
	return binary.LittleEndian.Uint32(tmp[:])
}

// chercherChaineBuild ancre la section d'identification sur la chaine de build.
//
// La recherche part de la fin du REGISTRE plus un champ de version, et ne porte que sur des
// offsets ALIGNES SUR 4 : la fin du registre l'est (`8 + n * 0x4100`, et `0x4100 = 4 * 4160`),
// la table par type est faite de u32, donc l'offset de la chaine l'est aussi — et l'alignement
// est ce qui rend la fermeture `(versionOff - finRegistre) % 4 == 0` vraie par construction
// plutot que verifiee apres coup.
func chercherChaineBuild(d []byte, finRegistre int) (int, bool) {
	debut := finRegistre + identFieldBytes
	fin := debut + identSearchSpan
	if borne := len(d) - identFieldBytes; fin > borne {
		fin = borne
	}
	for off := debut; off < fin; off += 4 {
		if !strings.HasPrefix(string(d[off:off+len(identBuildPrefix)]), identBuildPrefix) {
			continue
		}
		if s := chaineDeChamp(d, off); len(s) >= len(identBuildPrefix) {
			return off, true
		}
	}
	return 0, false
}

// chaineDeChamp lit un champ de chaine de 32 octets : ASCII IMPRIMABLE termine par NUL. Un
// octet non imprimable rend la chaine vide — le champ n'est alors pas une chaine.
func chaineDeChamp(d []byte, off int) string {
	if off < 0 || off+identFieldBytes > len(d) {
		return ""
	}
	var b strings.Builder
	for _, c := range d[off : off+identFieldBytes] {
		if c == 0 {
			return b.String()
		}
		if c < 0x20 || c > 0x7e {
			return ""
		}
		b.WriteByte(c)
	}
	return b.String()
}

// lireTableParType rend les u32 qui separent la fin du registre du champ de version. Leur
// cardinal EST la fermeture arithmetique : c'est lui que l'ecrivain ecrit (`0xF60` bits = 123
// u32 sur le build de reference).
func lireTableParType(d []byte, finRegistre, versionOff int) []uint32 {
	if versionOff <= finRegistre || versionOff > len(d) {
		return nil
	}
	out := make([]uint32, 0, (versionOff-finRegistre)/4)
	for off := finRegistre; off+4 <= versionOff; off += 4 {
		out = append(out, binary.LittleEndian.Uint32(d[off:]))
	}
	return out
}

// dernierOctetNonNul rend l'offset du dernier octet non nul du tampon, ou -1.
//
// C'est la borne haute de tout balayage du corps : la table des joueurs se termine avec le
// dernier octet ecrit du chunk.
func dernierOctetNonNul(d []byte) int {
	for i := len(d) - 1; i >= 0; i-- {
		if d[i] != 0 {
			return i
		}
	}
	return -1
}
