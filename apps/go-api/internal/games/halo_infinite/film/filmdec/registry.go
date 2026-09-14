package filmdec

// Component registry (ECS archetype schema) parser. The registry lives in the
// film's chunk_00 (zlib-compressed; inflates to ~1.97 MB). It is an array of
// fixed-size archetype blocks; block #N holds the ORDERED component list of
// archetype #N — exactly the order FUN_14076cb60 iterates (and the bit index the
// presence-mask FUN_1406d7610 gates). Verified empirically: block 35 = the
// BIPED/player archetype (object-position-dynamic-precision at i0, … ,
// weapon-state-type-info ×4 = HELD WEAPON at i43..46, …).
//
// LE CADRAGE EST CELUI DU JEU DEPUIS LE LOT 1.2 (2026-09-14). Le registre inflate commence par
// un EN-TETE de `registryEntryBase` octets ([u32 FilmMajorVersion][u32]), puis un tableau
// d'ENTREES de `registrySlotSize` (0x104) octets chacune :
// `[nom ASCII NUL-termine @ +0x00, 0x100 octets][u32 niveau LE @ +0x100]`. C'est la table que
// `FUN_1428e2b68` remet a `FUN_142e2c690`, laquelle deserialise l'entree `k` avec le niveau lu
// en `entree + 0x100` (chaine relue au lot R7-d, cf. keyframe_fullstate_loop.go).
// Block layout: archetypeBlockSlots entrees ; la liste de composants est la suite d'entrees
// NOMMEES en tete du bloc, le reste est du bourrage nul.
//
// CE QUE CE CADRAGE CORRIGE. Jusqu'au 2026-09-14 le parse lisait des slots de 260 octets A
// PARTIR DE L'OCTET 0, sur un layout suppose `[u32 kind][u32 flags][nom @ +8]`. Les NOMS
// tombent au meme octet dans les deux cadrages (`slot+8` == `entree+0`) : tout le dispatch par
// nom etait donc juste, et le reste ne l'etait pas. Le `flags` lu en `slot+4` etait le niveau
// du composant PRECEDENT ; le « kind » lu en `slot+0` etait la queue de bourrage du nom voisin
// — nul sur 1 066 des 1 067 slots du build de reference, le 1 067e n'etant pas un kind mais le
// FilmMajorVersion en tete de fichier (cf. FilmMajorVersionFromHeader). Mesure du lot 1.2 sur
// les sept bobines par build (registry_entree_jeu_test.go) : 173 a 189 niveaux changent, dont
// QUATRE que le dispatch consomme reellement — ti=14 i0 crew-order, ti=21 i2 flock-destination,
// ti=30 i0 tacmap-poiicon, ti=44 i0 asset-transform.

import (
	"bytes"
	"encoding/binary"
)

const (
	registrySlotSize    = 260 // 0x104 : une entree = [nom @ +0x00][u32 niveau @ +0x100]
	archetypeBlockSlots = 64
	archetypeBlockSize  = registrySlotSize * archetypeBlockSlots // 0x4100
	// registryEntryBase : l'octet ou commence le tableau d'entrees. Les huit premiers octets du
	// registre inflate sont un en-tete : [u32 FilmMajorVersion][u32] (25/24/27 selon la version
	// du film, cf. film_major_version.go). L'entree `i` du bloc `b` commence donc a
	// `registryEntryBase + b*archetypeBlockSize + i*registrySlotSize`.
	registryEntryBase = 8
	// registryEntryNameBytes : la zone de nom d'une entree, ASCII NUL-terminee.
	registryEntryNameBytes = 0x100
	// registryEntryLevelOffset : le u32 LE de niveau de precision, en queue d'entree. C'est la
	// valeur que `FUN_142e2c690` passe au deserialiseur du composant.
	registryEntryLevelOffset = 0x100
)

// Étiquettes de composant citées à plus de deux endroits du décodage. Les autres
// restent en littéral : les centraliser toutes créerait une table d'indirection sans
// lecteur. Le risque que porte une étiquette dupliquée — deux branches qui ne
// consomment pas le même nombre de bits — est déjà couvert par
// TestCaptureConsumesSameBitsAsDispatch (capture_test.go), pas par un test de grep.
const (
	compObjectBodyVitality  = "object-body-vitality-component"
	compWeaponStateTypeInfo = "weapon-state-type-info"
	compForwardUpDynPrec    = "object-forward-and-up-dynamic-precision-component"
)

// Archetype is one ECS archetype: an ordered list of component names. The slice
// index is the iterator/mask bit index used by the component loop.
type Archetype struct {
	Index      int      // block number = archetype index in the registry
	Components []string // ordered component names (mask bit i -> Components[i])
	// Levels[i] = le u32 de niveau de précision du composant i, lu en `entrée + 0x100` —
	// exactement celui que `FUN_142e2c690` passe au désérialiseur. Il sert de niveau L
	// (largeur d'axe = quantAxisWidth(L)) au traverseur générique.
	//
	// LE CHAMP S'APPELAIT `Flags` JUSQU'AU LOT 1.2 (2026-09-14), du nom du champ que
	// l'ancien cadrage lisait en `slot+4` — lequel était en réalité le niveau du composant
	// PRÉCÉDENT. Le nom est corrigé avec la lecture : il n'existe aucun champ « flags » dans
	// une entrée de registre.
	//
	// CE N'EST PAS la source des largeurs de la position absolue d'un biped (i0) : le
	// registre est BIT-À-BIT IDENTIQUE d'un film à l'autre DANS UN BUILD alors que les
	// largeurs d'i0 changent de carte en carte (13/13/14 vs 15/15/15). Le niveau d'i0 est
	// câblé au site d'appel (MOV R9D,0x10) et les largeurs dérivent des bornes du BSP de la
	// carte. Découpage réel d'i0 : DetectI0Layout (i0_layout.go), lu dans le bitstream.
	Levels []uint32
}

// Level returns the precision level of component i, or 0 if out of range.
func (a Archetype) Level(i int) uint32 {
	if i < 0 || i >= len(a.Levels) {
		return 0
	}
	return a.Levels[i]
}

// component returns the name at iterator index i, or "" if out of range.
func (a Archetype) component(i int) string {
	if i < 0 || i >= len(a.Components) {
		return ""
	}
	return a.Components[i]
}

// indicesOf returns every iterator index whose component name equals name.
func (a Archetype) indicesOf(name string) []int {
	var out []int
	for i, c := range a.Components {
		if c == name {
			out = append(out, i)
		}
	}
	return out
}

// Registry is the parsed set of archetype blocks from chunk_00.
type Registry struct {
	Archetypes []Archetype
	// fingerprint est l'empreinte FNV-1a des entrees nommees, calculee pendant la passe de
	// lecture (registry_fingerprint.go). Se lit par RegistryFingerprint.
	fingerprint uint64
	// TruncatedBytes : les octets de queue qu'aucun bloc ENTIER ne couvre, QUAND le parse a
	// epuise le tampon sans rencontrer la fin structurelle du registre. Zero sur un chunk_00
	// nominal — la lecture s'y arrete sur la section d'identification (bloc 49 ou 50), et tout
	// ce qui suit appartient aux sections 2 et 3, pas a une troncature.
	//
	// POURQUOI CE CHAMP EXISTE (revue R1 du lot 1.2). Le parse ne lit que des blocs ENTIERS : un
	// tampon qui s'arrete au milieu d'un bloc verrait sa fin ignoree EN SILENCE, et « registre
	// plus court que prevu » se lirait comme « ce build declare moins d'archetypes ». Une
	// troncature est un fait du tampon, pas une propriete du jeu : elle se compte et se nomme
	// (D14). Le champ est exporte pour qu'un appelant puisse le journaliser ; aucun ne le fait
	// encore, et c'est consigne au registre des replis du lot 1.9.0.
	TruncatedBytes int
}

// Archetype returns archetype #idx, or (zero, false) if idx is out of range.
func (r *Registry) Archetype(idx int) (Archetype, bool) {
	if idx < 0 || idx >= len(r.Archetypes) {
		return Archetype{}, false
	}
	return r.Archetypes[idx], true
}

// registryError : une erreur sentinelle CONSTANTE de ce fichier.
//
// POURQUOI UN `const` ET PAS UN `var errors.New(...)`. Le ratchet
// `archlint/TestFilmdecPackageVarsNeCroitPas` gele l'etat global de `filmdec` : ce paquet porte
// son etat de reglage dans des variables de paquet, et c'est ce qui oblige tout le decodage a
// passer sous `LockProcessDecode`. Le ratchet vise cet etat MUTABLE — une valeur constante n'en
// est pas, elle ne peut ni etre reassignee ni etre partagee entre deux decodages. La sentinelle
// est donc posee en `const` : l'intention du ratchet est respectee, pas contournee, et
// `errors.Is` fonctionne (la valeur est comparable et unique).
type registryError string

// Error implemente error.
func (e registryError) Error() string { return string(e) }

// ErrRegistryStillCompressed : le tampon remis a [ParseRegistryChunk] porte encore son en-tete
// zlib — l'appelant a saute la decompression, ou le chunk en porte DEUX couches.
//
// POURQUOI CETTE ERREUR EXISTE. Du 2026-09-02 (c17f4941f, lot 1a de PLAN_CUISSON_PERF, qui a
// retire l'inflate de cette fonction) au 2026-09-06, un tampon encore compresse rendait un
// registre VIDE et une erreur NULLE. Rien ne disait « tu ne m'as pas decompresse » : chaque
// lecteur d'archetype rendait ensuite « archetype N absent du registre », un message qui accuse
// le BUILD DU JEU d'un defaut de l'APPELANT. C'est un refus silencieux, pas une lecture.
//
// CE QUE CE COMMENTAIRE A DIT DE FAUX, ET QUI EST CORRIGE ICI (revue CTF-R1, 2026-09-06). Il
// affirmait que « la seule cuisson reelle de la CI decodait sans registre pendant quatre
// jours », a cause des deux couches zlib du fixture `film_e2e/c0a82e88`. C'EST FAUX : le
// telechargeur de l'ouvrier pele deja une couche (`cmd/replay-worker/job.go`, `downloadChunk`),
// `filmsource.Load` pele la seconde — les deux couches etaient donc absorbees par deux etages
// differents et le registre arrivait INTACT. Mesure : fixture d'origine remis, l'epreuve E2E est
// verte et rend le meme artefact de 283 260 octets. Aucun sinistre de production ni de CI n'est
// attribuable a ce defaut ; il n'a ete observe que par une sonde jetable lisant `testdata` en
// direct, chemin qu'aucun test n'emprunte. La sentinelle reste justifiee pour elle-meme : un
// registre vide rendu en silence est un piege, quel que soit l'appelant qui tombe dedans.
const ErrRegistryStillCompressed = registryError(
	"filmdec: chunk_00 (registre) encore compresse — decompresser avant ParseRegistryChunk")

// ParseRegistryChunk parses every fixed-size archetype block of an ALREADY-INFLATED chunk_00.
//
// IT NO LONGER INFLATES (lot 1 of PLAN_CUISSON_PERF, 2026-09-02). Decompression happens once per
// film, in `filmsource`: the cooking path hands over `film.Chunk(<registre>)`, and the single-chunk
// readers (research tools, tests) hand over `filmdec.ReadFilmChunk(dir, 0)`, which inflates
// through the same decompressor. A still-compressed buffer is REFUSED ([ErrRegistryStillCompressed])
// — the caller must inflate first; `internal/archlint` forbids a second `zlib.NewReader` inside
// `filmdec`, so this function DETECTE l'en-tete sans jamais le decompresser.
//
// The error return is kept: the signature is used in a dozen files, and the parse itself will grow
// error cases (a registry whose block size does not divide the buffer is already suspicious).
func ParseRegistryChunk(data []byte) (*Registry, error) {
	if looksZlib(data) {
		return nil, ErrRegistryStillCompressed
	}
	return parseRegistry(data), nil
}

// looksZlib dit si `data` commence par un EN-TETE ZLIB (RFC 1950), sans rien decompresser.
//
// LE TEST EST CELUI DE LA RFC, PAS LE SEUL PREMIER OCTET. `filmsource.inflate` se contente de
// `raw[0] == 0x78` parce qu'un faux positif y est inoffensif (le `zlib.NewReader` echoue et le
// tampon traverse tel quel) ; ici un faux positif REFUSERAIT un registre valide. Les deux octets
// de l'en-tete portent donc leurs trois conditions : methode DEFLATE (CM=8, quartet bas de
// l'octet 0), pas de dictionnaire preset (bit 5 de l'octet 1), et somme de controle
// `(octet0<<8 | octet1) % 31 == 0`.
//
// LA SOMME DE CONTROLE EST PORTEUSE, ET C'EST MESURE (revue CTF-R1, 2026-09-06, balayage des
// 1 378 `chunk_00` du cache). Un registre inflate commence par son `FilmMajorVersion` u32 LE
// (identifie le 2026-09-12 — cf. [FilmMajorVersionFromHeader] ; cette note le lisait auparavant
// comme « le kind du premier slot », le comptage etait juste et l'interpretation non) : `0x29`
// sur 1 117 films, mais aussi `0x28` (204 films), `0x27` (34), `0x25` (13), `0x26`/`0x1f`/`0x21`
// (3 chacun), `0x22` (1) — le premier octet n'est donc PAS constant, et les 204 films en `0x28`
// passent la condition CM=8 : seule la somme de controle les sauve (`0x2800 % 31 = 10`).
// « Jamais 0x78 » tient sur tout le corpus : 0 faux positif.
func looksZlib(data []byte) bool {
	if len(data) < 2 || data[0]&0x0f != 0x08 || data[1]&0x20 != 0 {
		return false
	}
	return (uint16(data[0])<<8|uint16(data[1]))%31 == 0
}

// parseRegistry lit les blocs d'archetype et S'ARRETE A LA FIN STRUCTURELLE du registre : un
// bloc de registre est une suite d'entrees nommees en tete, puis des zeros jusqu'au bout du
// bloc (bloc vide = zero entree, ex. bloc 8). Le premier bloc qui viole cette regle appartient
// a la section suivante de chunk_00 (table par type + identification du build, puis corps
// propre au match) — diviser le FICHIER ENTIER par la taille d'un bloc donnait « 118 blocs » et
// ramassait des faux positifs dans le corps (mesure lot 3 du plan « percer la trame »,
// 2026-08-30 : registre = 50 blocs sur le build de reference, verdict corpus dans
// lot3_registre_compte_research_test.go).
//
// LA REGLE DE QUEUE N'A PLUS D'EXEMPTION (lot 1.2, 2026-09-14). L'ancienne devait epargner
// quatre octets du « slot de terminaison » (0x01/0x02 mesures sur la plupart des blocs) : sous
// le cadrage a l'octet 8 ces quatre octets ne sont pas du bourrage, ce sont le NIVEAU de la
// derniere entree nommee, et l'entree de terminaison est entierement nulle. Mesure sur les sept
// bobines par build : meme compte de blocs qu'avant (49 ou 50 selon le build), queue nulle sur
// 7/7 (registry_entree_jeu_test.go).
func parseRegistry(data []byte) *Registry {
	nBlocks, queue := registryWholeBlocks(len(data))
	reg := &Registry{}
	fp := registryHasher()
	// epuise : la boucle est allee au bout des blocs ENTIERS sans rencontrer la fin structurelle
	// du registre. C'est la seule situation ou les octets de queue sont une TRONCATURE ; sur un
	// chunk_00 complet la boucle sort par `break` et la queue est la section suivante.
	epuise := true
	for b := 0; b < nBlocks; b++ {
		base := registryEntryBase + b*archetypeBlockSize
		arch := Archetype{Index: b}
		for s := 0; s < archetypeBlockSlots; s++ {
			off := base + s*registrySlotSize
			name := entryName(data, off)
			if name == "" {
				break // start of zero padding -> end of this archetype's list
			}
			arch.Components = append(arch.Components, name)
			arch.Levels = append(arch.Levels, entryLevel(data, off))
		}
		if !registryBlockTail(data, base, len(arch.Components)) {
			epuise = false
			break // fin du registre : ce bloc est le debut de la section suivante
		}
		for i, name := range arch.Components {
			fp.addEntry(data, base+i*registrySlotSize, name)
		}
		reg.Archetypes = append(reg.Archetypes, arch)
	}
	if epuise {
		reg.TruncatedBytes = queue
	}
	reg.fingerprint = fp.sum()
	warnUnknownRegistry(reg.fingerprint, len(reg.Archetypes), fp.slots)
	return reg
}

// registryWholeBlocks rend le nombre de blocs ENTIERS que porte un tampon de `n` octets, et les
// octets de queue qu'aucun d'eux ne couvre.
//
// LA BOUCLE DE BLOCS NE DOIT PARCOURIR QUE DES BLOCS ENTIERS, et c'est une CONDITION DE SURETE,
// pas une commodite : `registryBlockTail` compare la suite nommee a la fin du bloc, donc sur un
// bloc incomplet il recevrait `from > to` et `zeroTail` PANIQUERAIT (`data[from:to]`). Les deux
// appelants de production (`killcollector/hits.go`, `filmdec/film_context.go`) n'ont aucun
// `recover` : un `chunk_00` tronque ferait tomber le processus. Reproductions et non-regression :
// registry_tronque_test.go.
//
// Un tampon plus court que l'en-tete est entierement de la queue : il ne porte meme pas le
// debut du tableau d'entrees.
func registryWholeBlocks(n int) (blocs, queue int) {
	dispo := n - registryEntryBase
	if dispo < 0 {
		return 0, n
	}
	return dispo / archetypeBlockSize, dispo % archetypeBlockSize
}

// registryBlockTail dit si, apres la suite nommee de `run` entrees, le bloc n'est plus que du
// bourrage : des zeros depuis l'entree de terminaison jusqu'au bout du bloc. `base` est
// l'octet de la PREMIERE ENTREE du bloc, pas celui du bloc.
func registryBlockTail(data []byte, base, run int) bool {
	if run >= archetypeBlockSlots {
		return true // bloc plein : pas d'entree de terminaison
	}
	return zeroTail(data, base+run*registrySlotSize, base+archetypeBlockSize)
}

// zeroTail dit si data[from:to) ne contient que des octets nuls (bornes ecretees au buffer).
func zeroTail(data []byte, from, to int) bool {
	if from < 0 {
		from = 0
	}
	if to > len(data) {
		to = len(data)
	}
	for _, c := range data[from:to] {
		if c != 0 {
			return false
		}
	}
	return true
}

// entryName extracts the NUL-terminated ASCII name at the START of the entry at `off`.
//
// LE PARAMETRE EST L'OCTET DE L'ENTREE, pas celui d'un slot : la fonction s'appelait `slotName`
// et lisait le nom en `off+8` jusqu'au lot 1.2. Le renommage est deliberé — un appelant qui
// passerait l'ancien offset lirait le nom du VOISIN, en silence, et le compilateur ne pouvait
// pas le dire.
func entryName(data []byte, off int) string {
	if off < 0 || off >= len(data) {
		return ""
	}
	end := off + registryEntryNameBytes
	if end > len(data) {
		end = len(data)
	}
	raw := data[off:end]
	if z := bytes.IndexByte(raw, 0); z >= 0 {
		raw = raw[:z]
	}
	for _, c := range raw { // reject non-printable (not a real name entry)
		if c < 0x20 || c > 0x7e {
			return ""
		}
	}
	return string(raw)
}

// entryLevel rend le u32 LE de niveau de precision de l'entree a `off`, ou 0 hors du tampon.
func entryLevel(data []byte, off int) uint32 {
	p := off + registryEntryLevelOffset
	if p < 0 || p+4 > len(data) {
		return 0
	}
	return binary.LittleEndian.Uint32(data[p:])
}
