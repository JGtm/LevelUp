package filmdec

// Component registry (ECS archetype schema) parser. The registry lives in the
// film's chunk_00 (zlib-compressed; inflates to ~1.97 MB). It is an array of
// fixed-size archetype blocks; block #N holds the ORDERED component list of
// archetype #N — exactly the order FUN_14076cb60 iterates (and the bit index the
// presence-mask FUN_1406d7610 gates). Verified empirically: block 35 @0x08e300 =
// the BIPED/player archetype (object-position-dynamic-precision at i0, … ,
// weapon-state-type-info ×4 = HELD WEAPON at i43..46, …).
//
// Slot layout (260 bytes): [u32 kind LE][u32 flags LE][name ASCII, NUL-padded].
// Block layout: archetypeBlockSlots slots; the component list is the leading run
// of non-empty-name slots, the rest is zero padding.

import (
	"bytes"
	"encoding/binary"
)

const (
	registrySlotSize    = 260
	archetypeBlockSlots = 64
	archetypeBlockSize  = registrySlotSize * archetypeBlockSlots // 0x4100
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
	// Flags[i] = le champ flags (u32 @ slot+4) du composant i, utilisé comme niveau de
	// précision L (largeur d'axe = quantAxisWidth(L)) par le traverseur générique.
	//
	// CE N'EST PAS la source des largeurs de la position absolue d'un biped (i0) : le
	// registre est BIT-À-BIT IDENTIQUE d'un film à l'autre (FNV des 1067 slots noms+flags
	// = a413610cd08e4355 sur Cliffhanger comme sur Catalyst) alors que les largeurs d'i0
	// changent de carte en carte (13/13/14 vs 15/15/15). Le niveau d'i0 est câblé au site
	// d'appel (MOV R9D,0x10) et les largeurs dérivent des bornes du BSP de la carte.
	// Découpage réel d'i0 : DetectI0Layout (i0_layout.go), lu dans le bitstream.
	Flags []uint32
}

// Level returns the precision level (flags) of component i, or 0 if out of range.
func (a Archetype) Level(i int) uint32 {
	if i < 0 || i >= len(a.Flags) {
		return 0
	}
	return a.Flags[i]
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
	// fingerprint est l'empreinte FNV-1a des slots non vides, calculee pendant la passe de
	// lecture (registry_fingerprint.go) — la seule qui voie le champ `kind`, que le parse ne
	// retient pas. Se lit par RegistryFingerprint.
	fingerprint uint64
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
// 1 378 `chunk_00` du cache). Un registre inflate commence par le `kind` u32 LE de son premier
// slot : `0x29` sur 1 117 films, mais aussi `0x28` (204 films), `0x27` (34), `0x25` (13),
// `0x26`/`0x1f`/`0x21` (3 chacun), `0x22` (1) — le premier octet n'est donc PAS constant, et les
// 204 films en `0x28` passent la condition CM=8 : seule la somme de controle les sauve
// (`0x2800 % 31 = 10`). « Jamais 0x78 » tient sur tout le corpus : 0 faux positif.
func looksZlib(data []byte) bool {
	if len(data) < 2 || data[0]&0x0f != 0x08 || data[1]&0x20 != 0 {
		return false
	}
	return (uint16(data[0])<<8|uint16(data[1]))%31 == 0
}

// parseRegistry lit les blocs d'archetype et S'ARRETE A LA FIN STRUCTURELLE du registre : un
// bloc de registre est une suite de slots nommes en tete, puis un slot de terminaison dont
// SEUL le champ flags peut etre non nul (0x01/0x02 mesures — le niveau lu « un cran plus
// loin », meme decalage que R7-e), puis des zeros jusqu'au bout du bloc (bloc vide = zero
// slot, ex. bloc 8). Le premier bloc qui viole cette regle appartient a la section suivante de
// chunk_00 (table par type + identification du build, puis corps propre au match) — diviser le
// FICHIER ENTIER par la taille d'un bloc donnait « 118 blocs » et ramassait des faux positifs
// dans le corps (mesure lot 3 du plan « percer la trame », 2026-08-30 : registre = 50 blocs
// sur le build de reference, verdict corpus dans lot3_registre_compte_research_test.go).
func parseRegistry(data []byte) *Registry {
	reg := &Registry{}
	fp := registryHasher()
	nBlocks := len(data) / archetypeBlockSize
	for b := 0; b < nBlocks; b++ {
		base := b * archetypeBlockSize
		arch := Archetype{Index: b}
		for s := 0; s < archetypeBlockSlots; s++ {
			off := base + s*registrySlotSize
			name := slotName(data, off)
			if name == "" {
				break // start of zero padding -> end of this archetype's list
			}
			arch.Components = append(arch.Components, name)
			arch.Flags = append(arch.Flags, binary.LittleEndian.Uint32(data[off+4:])) // flags @ slot+4 = level
		}
		if !registryBlockTail(data, base, len(arch.Components)) {
			break // fin du registre : ce bloc est le debut de la section suivante
		}
		for i, name := range arch.Components {
			fp.addSlot(data, base+i*registrySlotSize, name)
		}
		reg.Archetypes = append(reg.Archetypes, arch)
	}
	reg.fingerprint = fp.sum()
	warnUnknownRegistry(reg.fingerprint, len(reg.Archetypes), fp.slots)
	return reg
}

// registryBlockTail dit si, apres la suite nommee de `run` slots, le bloc n'est que du
// bourrage de registre : kind nul et zone de nom nulle sur le slot de terminaison, zeros
// jusqu'au bout du bloc. Le champ flags du slot de terminaison (4 octets a run*260+4) est
// exempte : il porte 0x01/0x02 sur la plupart des blocs du registre de reference.
func registryBlockTail(data []byte, base, run int) bool {
	if run >= archetypeBlockSlots {
		return true // bloc plein : pas de slot de terminaison
	}
	term := base + run*registrySlotSize
	return zeroTail(data, term, term+4) && zeroTail(data, term+8, base+archetypeBlockSize)
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

// slotName extracts the NUL-terminated ASCII name at slot offset off+8.
func slotName(data []byte, off int) string {
	start := off + 8
	end := off + registrySlotSize
	if start >= len(data) {
		return ""
	}
	if end > len(data) {
		end = len(data)
	}
	raw := data[start:end]
	if z := bytes.IndexByte(raw, 0); z >= 0 {
		raw = raw[:z]
	}
	for _, c := range raw { // reject non-printable (not a real name slot)
		if c < 0x20 || c > 0x7e {
			return ""
		}
	}
	return string(raw)
}
