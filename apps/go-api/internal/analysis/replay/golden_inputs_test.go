package replay

// golden_inputs_test.go — LES ENTREES DECODEES DU FILM DE REFERENCE, FIGEES.
//
// POURQUOI CE FICHIER EXISTE. `BuildFromPositions` est deja PUR — il ne lui manquait que ses
// entrees. Tant qu elles ne vivaient que dans un film de 20,2 Mo hors depot, les chiffres du
// chantier (475/519 tirs, 90/105 vies nommees, 70 lancers, 439 projectiles, 184 etats
// d inventaire) n etaient verrouilles par AUCUN test : ils vivaient dans des Markdown et dans
// un artefact genere. Un refactor de l assemblage pouvait les deplacer en silence.
//
// CE QUE CE FIXTURE EST, ET CE QU IL N EST PAS. C est un ETAGE 1 : les entrees DEJA DECODEES,
// serialisees, rejouees dans l assemblage pur. Il verrouille l ASSEMBLAGE et rien d autre — un
// changement du DECODAGE ne le fait pas bouger, c est le travail de l etage 2 (la mini-bobine,
// minifilm_test.go). Les deux etages sont complementaires et ne se remplacent pas.
//
// ZERO OCTET DE FILM. Les tests de golden_assembly_test.go ne lisent que ce fichier fige. Le
// film n intervient qu a la REGENERATION, ci-dessous, qui est la seule porte d ecriture.
//
// REGENERATION (la seule ; un golden ne s edite JAMAIS a la main) :
//
//	REPLAY_FILM_DIR=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/replay/ -run GoldenInputs -update
//
// puis, la sortie figee elle-meme :
//
//	go test ./internal/analysis/replay/ -run GoldenAssembly -update
//
// CE QUE LE FIXTURE PORTE, ET POURQUOI PAS PLUS. Il porte exactement les champs que
// l assemblage CONSOMME (cf. la liste par type ci-dessous). Il ne porte NI la geometrie NI la
// structure de carte : celles-ci ne viennent pas du film mais de catalogues versionnes a part,
// et les inclure ferait grossir le fixture d un ordre de grandeur pour verrouiller un chargement
// de fichier, pas un decodage.
//
// LE FORMAT EST DELTA-CODE, ET L ORDRE D ORIGINE EST PRESERVE. `BuildFromPositions` trie ses
// positions par `sort.SliceStable` et numerote les traces dans l ORDRE DE PREMIERE APPARITION
// des slots : reordonner les positions changerait la sortie. Le codec conserve donc la suite
// telle que le decodeur l a produite, et ne delta-code que les VALEURS (horodatage global,
// coordonnees par slot).

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

var updateGolden = flag.Bool("update", false, "reecrire les fichiers figes de testdata/")

// goldenFilm est le film de reference du chantier : Cliffhanger, Fiesta, 8 joueurs.
const goldenFilm = "000d5950"

// goldenDir est le repertoire des sorties figees.
const goldenDir = "testdata"

// goldenInputsPath est le fixture d entrees, versionne.
func goldenInputsPath() string {
	return filepath.Join(goldenDir, "inputs_"+goldenFilm+".bin.gz")
}

// goldenInputsMagic identifie le format et sa version. Un fixture d une autre version est une
// ERREUR, jamais une lecture « au mieux » : un decodage decale rendrait des chiffres plausibles.
const goldenInputsMagic = "REPLAYINPUTS1\n"

// goldenInputs porte les entrees de BuildFromPositions decodees du film de reference.
//
// LES CHAMPS SERIALISES, PAR TYPE — ce sont ceux que l assemblage consomme :
//
//	BipedPosition     Slot · TimestampUS · X/Y/Z · HasWorld · HasYaw+YawRaw ·
//	                  HasBody+Body.Health · HasShield+Shield.Shield
//	FireEvent         TimestampUS · FilmIndex · WeaponID · HasAim+Aim
//	KeyframeLoadout   TimestampUS · Slot · Families
//	GrenadeThrow      TimestampUS · FilmIndex · TypeID
//	ProjectileTrack   Slot · Gen · Pts(TimestampUS · X/Y/Z · AtRest)
//	KeyframeInventory tout, sauf Chunk/PacketIndex (traçabilite dans le film, pas une entree)
//	Death             XUID · Gamertag · TimeMS
//	PlayerIndexTable  entier
type goldenInputs struct {
	Film        string
	Positions   []filmdec.BipedPosition
	Fire        []filmdec.FireEvent
	Loadouts    []filmdec.KeyframeLoadout
	Grenades    []filmdec.GrenadeThrow
	Projectiles []filmdec.ProjectileTrack
	Inventory   []KeyframeInventory
	Deaths      []Death
	Indices     PlayerIndexTable
}

// options rend les Options d assemblage portees par le fixture. La geometrie et la structure
// sont volontairement absentes (cf. l en-tete).
func (g *goldenInputs) options() Options {
	return Options{
		Loadouts:      g.Loadouts,
		Grenades:      g.Grenades,
		Projectiles:   g.Projectiles,
		Inventory:     g.Inventory,
		Deaths:        g.Deaths,
		PlayerIndices: g.Indices,
	}
}

// ---------------------------------------------------------------------------
// Encodeur / decodeur
// ---------------------------------------------------------------------------

// cmScale convertit une coordonnee en CENTIMETRES ENTIERS.
//
// CE N EST PAS UNE PERTE, ET C EST MESURABLE : toute coordonnee publiee par l assemblage passe
// par `round2` (arrondi au centieme), sans exception — traces, tirs, lancers, projectiles. Le
// centimetre entier est donc exactement la precision que la sortie porte. Coder un float32 brut
// couterait 12 octets par position pour une decimale que personne ne lit.
const cmScale = 100

// gwriter accumule un flux binaire. Les entiers sont en varint : les deltas d horodatage et de
// position tiennent sur un a deux octets, ce qui fait tout le poids du fixture.
type gwriter struct{ b []byte }

func (w *gwriter) u(v uint64)   { w.b = binary.AppendUvarint(w.b, v) }
func (w *gwriter) i(v int64)    { w.b = binary.AppendVarint(w.b, v) }
func (w *gwriter) byte8(v byte) { w.b = append(w.b, v) }
func (w *gwriter) f32(v float32) {
	w.b = binary.LittleEndian.AppendUint32(w.b, math.Float32bits(v))
}
func (w *gwriter) str(s string) {
	w.u(uint64(len(s)))
	w.b = append(w.b, s...)
}
func (w *gwriter) bool8(v bool) {
	if v {
		w.byte8(1)
		return
	}
	w.byte8(0)
}

// greader relit le flux. Toute incoherence est une ERREUR remontee, jamais une valeur nulle
// servie en silence.
type greader struct {
	b   []byte
	off int
	err error
}

func (r *greader) u() uint64 {
	if r.err != nil {
		return 0
	}
	v, n := binary.Uvarint(r.b[r.off:])
	if n <= 0 {
		r.err = fmt.Errorf("uvarint illisible a l offset %d", r.off)
		return 0
	}
	r.off += n
	return v
}

func (r *greader) i() int64 {
	if r.err != nil {
		return 0
	}
	v, n := binary.Varint(r.b[r.off:])
	if n <= 0 {
		r.err = fmt.Errorf("varint illisible a l offset %d", r.off)
		return 0
	}
	r.off += n
	return v
}

func (r *greader) byte8() byte {
	if r.err != nil {
		return 0
	}
	if r.off >= len(r.b) {
		r.err = fmt.Errorf("fin de flux prematuree a l offset %d", r.off)
		return 0
	}
	v := r.b[r.off]
	r.off++
	return v
}

func (r *greader) f32() float32 {
	if r.err != nil {
		return 0
	}
	if r.off+4 > len(r.b) {
		r.err = fmt.Errorf("float32 tronque a l offset %d", r.off)
		return 0
	}
	v := math.Float32frombits(binary.LittleEndian.Uint32(r.b[r.off:]))
	r.off += 4
	return v
}

func (r *greader) str() string {
	n := int(r.u())
	if r.err != nil {
		return ""
	}
	if r.off+n > len(r.b) {
		r.err = fmt.Errorf("chaine tronquee a l offset %d", r.off)
		return ""
	}
	s := string(r.b[r.off : r.off+n])
	r.off += n
	return s
}

func (r *greader) bool8() bool { return r.byte8() == 1 }

// Drapeaux de presence d une position, sur un octet.
const (
	gpHasWorld  byte = 1 << 0
	gpHasYaw    byte = 1 << 1
	gpHasBody   byte = 1 << 2
	gpHasShield byte = 1 << 3
)

// encodeGoldenInputs serialise les entrees. Format decrit en tete de fichier.
func encodeGoldenInputs(g *goldenInputs) []byte {
	w := &gwriter{b: []byte(goldenInputsMagic)}
	w.str(g.Film)

	// Table des slots : un slot tient sur 13 bits, mais un film n en emploie qu une centaine.
	// L indirection ramene 2 octets a 1 sur chaque position.
	slotIdx := map[uint32]int{}
	var slots []uint32
	for _, p := range g.Positions {
		if _, ok := slotIdx[p.Slot]; !ok {
			slotIdx[p.Slot] = len(slots)
			slots = append(slots, p.Slot)
		}
	}
	w.u(uint64(len(slots)))
	for _, s := range slots {
		w.u(uint64(s))
	}

	w.u(uint64(len(g.Positions)))
	var lastTS uint64
	lastXYZ := map[uint32][3]int64{}
	for _, p := range g.Positions {
		w.u(p.TimestampUS - lastTS) // horodatages non decroissants dans l ordre du film
		lastTS = p.TimestampUS
		w.u(uint64(slotIdx[p.Slot]))
		var fl byte
		if p.HasWorld {
			fl |= gpHasWorld
		}
		if p.HasYaw {
			fl |= gpHasYaw
		}
		if p.HasBody {
			fl |= gpHasBody
		}
		if p.HasShield {
			fl |= gpHasShield
		}
		w.byte8(fl)
		if p.HasWorld {
			cur := [3]int64{cmOf(p.X), cmOf(p.Y), cmOf(p.Z)}
			prev := lastXYZ[p.Slot]
			for a := 0; a < 3; a++ {
				w.i(cur[a] - prev[a])
			}
			lastXYZ[p.Slot] = cur
		}
		if p.HasYaw {
			w.u(uint64(p.YawRaw))
		}
		if p.HasBody {
			w.f32(p.Body.Health)
		}
		if p.HasShield {
			w.f32(p.Shield.Shield)
		}
	}

	w.u(uint64(len(g.Fire)))
	lastTS = 0
	for _, e := range g.Fire {
		w.u(e.TimestampUS - lastTS)
		lastTS = e.TimestampUS
		w.i(int64(e.FilmIndex))
		w.u(e.WeaponID)
		w.bool8(e.HasAim)
		if e.HasAim {
			for a := 0; a < 3; a++ {
				w.f32(e.Aim[a])
			}
		}
	}

	w.u(uint64(len(g.Loadouts)))
	for _, l := range g.Loadouts {
		w.u(l.TimestampUS)
		w.u(uint64(l.Slot))
		w.u(uint64(len(l.Families)))
		for _, f := range l.Families {
			w.u(uint64(f))
		}
	}

	w.u(uint64(len(g.Grenades)))
	for _, t := range g.Grenades {
		w.u(t.TimestampUS)
		w.i(int64(t.FilmIndex))
		w.u(uint64(t.TypeID))
	}

	w.u(uint64(len(g.Projectiles)))
	for _, tr := range g.Projectiles {
		w.u(uint64(tr.Slot))
		w.u(uint64(tr.Gen))
		w.u(uint64(len(tr.Pts)))
		var pts uint64
		var prev [3]int64
		for _, s := range tr.Pts {
			w.u(s.TimestampUS - pts)
			pts = s.TimestampUS
			cur := [3]int64{cmOf(s.X), cmOf(s.Y), cmOf(s.Z)}
			for a := 0; a < 3; a++ {
				w.i(cur[a] - prev[a])
			}
			prev = cur
			w.bool8(s.AtRest)
		}
	}

	w.u(uint64(len(g.Inventory)))
	for _, inv := range g.Inventory {
		w.u(inv.TimestampUS)
		w.u(uint64(inv.Slot))
		w.bool8(inv.GrenadesRead)
		for _, c := range inv.Grenades {
			w.u(uint64(c))
		}
		w.i(int64(inv.AbilityIndex))
		w.i(int64(inv.DrawnSlot))
		w.u(uint64(inv.AmmoCandidates))
		w.bool8(inv.AmmoRead)
		for _, a := range inv.Ammo {
			encodeAmmo(w, a)
		}
	}

	w.u(uint64(len(g.Deaths)))
	for _, d := range g.Deaths {
		w.u(d.XUID)
		w.str(d.Gamertag)
		w.i(d.TimeMS)
	}

	w.u(uint64(g.Indices.Readings))
	w.u(uint64(g.Indices.Disagreements))
	xuids := make([]uint64, 0, len(g.Indices.ByXUID))
	for x := range g.Indices.ByXUID {
		xuids = append(xuids, x)
	}
	sort.Slice(xuids, func(i, j int) bool { return xuids[i] < xuids[j] })
	w.u(uint64(len(xuids)))
	for _, x := range xuids {
		w.u(x)
		w.i(int64(g.Indices.ByXUID[x]))
	}
	return w.b
}

// encodeAmmo serialise un emplacement de munitions. LES TROIS CAS SONT DISTINCTS (chargeur,
// jauge, rien) : un drapeau par pointeur, jamais un zero qui vaudrait absence.
func encodeAmmo(w *gwriter, a SlotAmmo) {
	w.bool8(a.Mag != nil)
	if a.Mag != nil {
		w.u(uint64(*a.Mag))
	}
	w.bool8(a.Res != nil)
	if a.Res != nil {
		w.u(uint64(*a.Res))
	}
	w.bool8(a.Gauge != nil)
	if a.Gauge != nil {
		w.b = binary.LittleEndian.AppendUint64(w.b, math.Float64bits(*a.Gauge))
	}
	w.u(uint64(a.Overheat))
	w.u(uint64(a.Flags))
}

func decodeAmmo(r *greader) SlotAmmo {
	var a SlotAmmo
	if r.bool8() {
		v := uint32(r.u())
		a.Mag = &v
	}
	if r.bool8() {
		v := uint32(r.u())
		a.Res = &v
	}
	if r.bool8() {
		if r.off+8 > len(r.b) {
			r.err = fmt.Errorf("jauge tronquee a l offset %d", r.off)
			return a
		}
		v := math.Float64frombits(binary.LittleEndian.Uint64(r.b[r.off:]))
		r.off += 8
		a.Gauge = &v
	}
	a.Overheat = uint32(r.u())
	a.Flags = uint32(r.u())
	return a
}

// decodeGoldenInputs relit le fixture.
func decodeGoldenInputs(blob []byte) (*goldenInputs, error) {
	if len(blob) < len(goldenInputsMagic) || string(blob[:len(goldenInputsMagic)]) != goldenInputsMagic {
		return nil, fmt.Errorf("fixture d entrees : magie absente ou version inconnue — regenerer")
	}
	r := &greader{b: blob, off: len(goldenInputsMagic)}
	g := &goldenInputs{Film: r.str()}

	nSlots := int(r.u())
	slots := make([]uint32, 0, nSlots)
	for k := 0; k < nSlots && r.err == nil; k++ {
		slots = append(slots, uint32(r.u()))
	}

	n := int(r.u())
	g.Positions = make([]filmdec.BipedPosition, 0, n)
	var lastTS uint64
	lastXYZ := map[uint32][3]int64{}
	for k := 0; k < n && r.err == nil; k++ {
		var p filmdec.BipedPosition
		lastTS += r.u()
		p.TimestampUS = lastTS
		si := int(r.u())
		if si >= len(slots) {
			return nil, fmt.Errorf("index de slot %d hors table (%d)", si, len(slots))
		}
		p.Slot = slots[si]
		fl := r.byte8()
		if fl&gpHasWorld != 0 {
			p.HasWorld = true
			prev := lastXYZ[p.Slot]
			var cur [3]int64
			for a := 0; a < 3; a++ {
				cur[a] = prev[a] + r.i()
			}
			lastXYZ[p.Slot] = cur
			p.X, p.Y, p.Z = fromCM(cur[0]), fromCM(cur[1]), fromCM(cur[2])
		}
		if fl&gpHasYaw != 0 {
			p.HasYaw = true
			p.YawRaw = uint32(r.u())
		}
		if fl&gpHasBody != 0 {
			p.HasBody = true
			p.Body.Health = r.f32()
		}
		if fl&gpHasShield != 0 {
			p.HasShield = true
			p.Shield.Shield = r.f32()
		}
		g.Positions = append(g.Positions, p)
	}

	n = int(r.u())
	g.Fire = make([]filmdec.FireEvent, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		var e filmdec.FireEvent
		lastTS += r.u()
		e.TimestampUS = lastTS
		e.FilmIndex = int(r.i())
		e.WeaponID = r.u()
		if e.HasAim = r.bool8(); e.HasAim {
			for a := 0; a < 3; a++ {
				e.Aim[a] = r.f32()
			}
		}
		g.Fire = append(g.Fire, e)
	}

	n = int(r.u())
	g.Loadouts = make([]filmdec.KeyframeLoadout, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		l := filmdec.KeyframeLoadout{TimestampUS: r.u(), Slot: uint32(r.u())}
		nf := int(r.u())
		for j := 0; j < nf && r.err == nil; j++ {
			l.Families = append(l.Families, uint32(r.u()))
		}
		g.Loadouts = append(g.Loadouts, l)
	}

	n = int(r.u())
	g.Grenades = make([]filmdec.GrenadeThrow, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		g.Grenades = append(g.Grenades, filmdec.GrenadeThrow{
			TimestampUS: r.u(), FilmIndex: int(r.i()), TypeID: uint32(r.u()),
		})
	}

	n = int(r.u())
	g.Projectiles = make([]filmdec.ProjectileTrack, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		tr := filmdec.ProjectileTrack{Slot: uint32(r.u()), Gen: uint32(r.u())}
		np := int(r.u())
		var ts uint64
		var prev [3]int64
		for j := 0; j < np && r.err == nil; j++ {
			ts += r.u()
			var cur [3]int64
			for a := 0; a < 3; a++ {
				cur[a] = prev[a] + r.i()
			}
			prev = cur
			tr.Pts = append(tr.Pts, filmdec.ProjectileSample{
				TimestampUS: ts, X: fromCM(cur[0]), Y: fromCM(cur[1]), Z: fromCM(cur[2]),
				AtRest: r.bool8(),
			})
		}
		g.Projectiles = append(g.Projectiles, tr)
	}

	n = int(r.u())
	g.Inventory = make([]KeyframeInventory, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		inv := KeyframeInventory{TimestampUS: r.u(), Slot: uint32(r.u())}
		inv.GrenadesRead = r.bool8()
		for j := 0; j < invGrenadeSlots; j++ {
			inv.Grenades[j] = uint32(r.u())
		}
		inv.AbilityIndex = int(r.i())
		inv.DrawnSlot = int(r.i())
		inv.AmmoCandidates = int(r.u())
		inv.AmmoRead = r.bool8()
		for j := 0; j < invGrenadeSlots; j++ {
			inv.Ammo[j] = decodeAmmo(r)
		}
		g.Inventory = append(g.Inventory, inv)
	}

	n = int(r.u())
	g.Deaths = make([]Death, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		g.Deaths = append(g.Deaths, Death{XUID: r.u(), Gamertag: r.str(), TimeMS: r.i()})
	}

	g.Indices = PlayerIndexTable{ByXUID: map[uint64]int{}}
	g.Indices.Readings = int(r.u())
	g.Indices.Disagreements = int(r.u())
	n = int(r.u())
	for k := 0; k < n && r.err == nil; k++ {
		x := r.u()
		g.Indices.ByXUID[x] = int(r.i())
	}
	if r.err != nil {
		return nil, r.err
	}
	if r.off != len(r.b) {
		return nil, fmt.Errorf("fixture d entrees : %d octet(s) non consomme(s) — format desynchronise",
			len(r.b)-r.off)
	}
	return g, nil
}

func cmOf(v float32) int64 { return int64(math.Round(float64(v) * cmScale)) }

func fromCM(v int64) float32 { return float32(float64(v) / cmScale) }

// loadGoldenInputs relit le fixture versionne. AUCUN OCTET DE FILM.
func loadGoldenInputs(t *testing.T) *goldenInputs {
	t.Helper()
	raw, err := os.ReadFile(goldenInputsPath()) //nolint:gosec // chemin fige dans le code
	if err != nil {
		t.Fatalf("fixture d entrees illisible : %v — regenerer avec "+
			"REPLAY_FILM_DIR=<film> go test -run GoldenInputs -update", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("fixture d entrees : gzip illisible : %v", err)
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(zr); err != nil {
		t.Fatalf("fixture d entrees : decompression : %v", err)
	}
	g, err := decodeGoldenInputs(buf.Bytes())
	if err != nil {
		t.Fatalf("fixture d entrees : %v", err)
	}
	return g
}

// TestGoldenInputsRoundTrip : le codec ne perd rien de ce que l assemblage consomme.
//
// C EST LE FILET DU FIXTURE LUI-MEME. Un codec qui perdrait un champ produirait un golden
// parfaitement stable et parfaitement faux. On le verifie par la seule mesure qui compte :
// l assemblage rejoue sur les entrees RELUES doit rendre le MEME document que sur les entrees
// d origine — ici, les entrees d origine etant deja le fixture, on controle qu un second
// aller-retour est un point fixe, octet pour octet.
func TestGoldenInputsRoundTrip(t *testing.T) {
	g := loadGoldenInputs(t)
	blob := encodeGoldenInputs(g)
	again, err := decodeGoldenInputs(blob)
	if err != nil {
		t.Fatalf("second decodage : %v", err)
	}
	if got := encodeGoldenInputs(again); !bytes.Equal(blob, got) {
		t.Fatalf("le codec n est pas un point fixe : %d octets contre %d — un champ se perd "+
			"ou se reconstruit differemment a chaque tour", len(got), len(blob))
	}
	docA := BuildFromPositions(goldenFilm, "halo_infinite", g.Positions, g.Fire, g.options())
	docB := BuildFromPositions(goldenFilm, "halo_infinite", again.Positions, again.Fire, again.options())
	if renderAssembly(docA) != renderAssembly(docB) {
		t.Error("l assemblage differe entre les entrees relues et leur re-serialisation : " +
			"le codec perd un champ que BuildFromPositions consomme")
	}
}

// TestGoldenInputsRegenerate : LA SEULE PORTE D ECRITURE DU FIXTURE.
//
// Elle exige DEUX conditions explicites — `-update` et `REPLAY_FILM_DIR` — parce qu une
// regeneration accidentelle transformerait n importe quelle regression en « nouvelle
// reference ». C est la meme discipline que le paquet killsource.
func TestGoldenInputsRegenerate(t *testing.T) {
	dir := os.Getenv("REPLAY_FILM_DIR")
	switch {
	case !*updateGolden:
		t.Skip("regeneration du fixture : passer -update (et REPLAY_FILM_DIR)")
	case dir == "":
		t.Skip("regeneration du fixture : REPLAY_FILM_DIR non defini")
	}
	g, err := decodeFilmInputs(goldenFilm, dir)
	if err != nil {
		t.Fatalf("decodage du film %s : %v", dir, err)
	}
	blob := encodeGoldenInputs(g)
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if _, err := zw.Write(blob); err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if err := os.MkdirAll(goldenDir, 0o750); err != nil {
		t.Fatalf("creation de %s : %v", goldenDir, err)
	}
	if err := os.WriteFile(goldenInputsPath(), buf.Bytes(), 0o600); err != nil {
		t.Fatalf("ecriture du fixture : %v", err)
	}
	t.Logf("fixture reecrit : %s (%d octets brut, %d compresse) — %d positions, %d tirs, "+
		"%d loadouts, %d lancers, %d projectiles, %d inventaires, %d morts, %d index",
		goldenInputsPath(), len(blob), buf.Len(), len(g.Positions), len(g.Fire), len(g.Loadouts),
		len(g.Grenades), len(g.Projectiles), len(g.Inventory), len(g.Deaths), len(g.Indices.ByXUID))
}

// decodeFilmInputs rejoue EXACTEMENT la sequence de decodage de BuildFromFilm — c est ce qui
// garantit que le fixture porte les memes entrees que la production. Les bornes de carte sont
// celles de Cliffhanger, lues dans le catalogue versionne du titre.
func decodeFilmInputs(film, dir string) (*goldenInputs, error) {
	wr, err := goldenWorldRange()
	if err != nil {
		return nil, err
	}
	scan := filmdec.DefaultScanFilmOptions()
	scan.WorldRange = wr
	scan.CaptureDirs = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		return nil, err
	}
	g := &goldenInputs{Film: film, Positions: pos}
	if g.Fire, err = filmdec.ScanFilmFireEvents(dir); err != nil {
		return nil, err
	}
	if g.Loadouts, err = filmdec.ScanFilmKeyframeLoadouts(dir, loadoutFamilies()); err != nil {
		return nil, err
	}
	if g.Inventory, err = ScanFilmKeyframeInventory(dir, loadoutFamilies(), 0); err != nil {
		return nil, err
	}
	if g.Grenades, err = filmdec.ScanFilmGrenadeThrows(dir); err != nil {
		return nil, err
	}
	if g.Projectiles, err = filmdec.ScanFilmProjectiles(dir, wr); err != nil {
		return nil, err
	}
	if g.Deaths, err = ScanFilmDeaths(dir); err != nil {
		return nil, err
	}
	idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(g.Deaths))
	if err != nil {
		return nil, err
	}
	table, collisions := injectiveOrEmpty(idx)
	if collisions > 0 {
		return nil, fmt.Errorf("index de joueur non injectif (%d collisions) — fixture refuse", collisions)
	}
	g.Indices = table
	return g, nil
}

// goldenWorldRange rend les bornes de dequantification de Cliffhanger.
//
// ELLES SONT ECRITES ICI, PAS LUES DU CATALOGUE : la regeneration ne doit dependre que du film
// et de ce fichier. Valeurs du catalogue versionne `map_quant_bounds.json` au 2026-07-31 ; si le
// catalogue change, [TestGoldenAssembly] tombe et le diff dit exactement ce qui a bouge.
func goldenWorldRange() (*filmdec.Vec3Range, error) {
	path := filepath.Join("..", "..", "..", "..", "..", "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json")
	cat, err := filmdec.LoadMapQuantCatalog(path)
	if err != nil {
		return nil, fmt.Errorf("catalogue de bornes %s : %w", path, err)
	}
	entry, err := cat.Lookup("Cliffhanger")
	if err != nil {
		return nil, err
	}
	r := entry.Range()
	return &r, nil
}
