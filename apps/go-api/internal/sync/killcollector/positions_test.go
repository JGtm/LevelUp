package killcollector

// positions_test.go — LE PONT (chunks synthétiques -> répertoire), LA COMPOSITION PURE, ET LE
// REFUS PROPRE. Aucun test ici n'ouvre de base ni ne lit de film réel — c'est le rôle de
// positions_integration_test.go (fixture réelle, gate KILLSOURCE_FIXTURES) et de
// kill_position_persister_test.go (persister, :memory:). Ce fichier verrouille exactement ce que
// G.2bis a ajouté : le pont disque, le filtrage des identités, la traduction en lignes, et que
// CHAQUE refus s'arrête AVANT toute tentative d'écriture (acquireShared panique s'il est appelé).

import (
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/haloclient"
)

// zlibCompressForTest compresse b, pour verifier que le pont recopie les octets tels quels
// (compresses ou non) et que c'est ReadFilmChunk qui decompresse a la lecture.
func zlibCompressForTest(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(b); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}
	return buf.Bytes()
}

// ─── writeChunksToTempDir : le pont ────────────────────────────────────────────────────────

func TestWriteChunksToTempDir_EcritEtSeRelitParFilmdec(t *testing.T) {
	chunks := []haloclient.FilmChunk{
		{Index: 1, Data: []byte("chunk-un-donnees")},
		{Index: 2, Data: []byte("chunk-deux-donnees")},
	}
	dir, cleanup, err := writeChunksToTempDir(chunks)
	if err != nil {
		t.Fatalf("writeChunksToTempDir: %v", err)
	}
	defer cleanup()

	if got := filmdec.CountFilmChunks(dir); got != 2 {
		t.Fatalf("CountFilmChunks = %d, attendu 2", got)
	}
	got1, err := filmdec.ReadFilmChunk(dir, 1)
	if err != nil || string(got1) != "chunk-un-donnees" {
		t.Errorf("chunk 1 relu = %q, err=%v", got1, err)
	}
	got2, err := filmdec.ReadFilmChunk(dir, 2)
	if err != nil || string(got2) != "chunk-deux-donnees" {
		t.Errorf("chunk 2 relu = %q, err=%v", got2, err)
	}
}

// TestWriteChunksToTempDir_ZlibRoundTrip — le pont RECOPIE, il ne décode rien : un chunk
// COMPRESSÉ doit se relire décompressé exactement comme le cache film hérité le sert déjà
// (haloclient.LocalFilmCache), parce que c'est ReadFilmChunk qui décompresse à LA LECTURE.
func TestWriteChunksToTempDir_ZlibRoundTrip(t *testing.T) {
	compressed := zlibCompressForTest(t, []byte("payload-compresse"))
	dir, cleanup, err := writeChunksToTempDir([]haloclient.FilmChunk{{Index: 1, Data: compressed}})
	if err != nil {
		t.Fatalf("writeChunksToTempDir: %v", err)
	}
	defer cleanup()

	got, err := filmdec.ReadFilmChunk(dir, 1)
	if err != nil {
		t.Fatalf("ReadFilmChunk: %v", err)
	}
	if string(got) != "payload-compresse" {
		t.Errorf("chunk decompresse = %q, attendu %q", got, "payload-compresse")
	}
}

// TestWriteChunksToTempDir_RefuseUnTrouDeSequence — un index manquant ferait lire les quatre
// scanners disque sur un PRÉFIXE silencieux du film (filmdec.CountFilmChunks s'arrête au premier
// trou) : le pont refuse plutôt que de laisser passer une lecture partielle plausible.
func TestWriteChunksToTempDir_RefuseUnTrouDeSequence(t *testing.T) {
	chunks := []haloclient.FilmChunk{
		{Index: 1, Data: []byte("a")},
		{Index: 3, Data: []byte("c")}, // 2 absent
	}
	dir, _, err := writeChunksToTempDir(chunks)
	if err == nil {
		t.Fatalf("attendu un refus (trou de sequence), obtenu dir=%q", dir)
	}
}

// TestWriteChunksToTempDir_ChunkVideEstUnTrou — un chunk DÉCLARÉ mais absent du disque source
// (Data vide) est un trou au même titre qu'un index manquant — même règle que ChunkSourceOf.
func TestWriteChunksToTempDir_ChunkVideEstUnTrou(t *testing.T) {
	chunks := []haloclient.FilmChunk{
		{Index: 1, Data: []byte("a")},
		{Index: 2, Data: nil},
	}
	if _, _, err := writeChunksToTempDir(chunks); err == nil {
		t.Fatal("attendu un refus (chunk 2 vide = trou), obtenu nil")
	}
}

func TestWriteChunksToTempDir_CleanupSupprimeLeRepertoire(t *testing.T) {
	dir, cleanup, err := writeChunksToTempDir([]haloclient.FilmChunk{{Index: 1, Data: []byte("a")}})
	if err != nil {
		t.Fatalf("writeChunksToTempDir: %v", err)
	}
	cleanup()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("le repertoire temporaire survit au cleanup (stat err=%v)", err)
	}
}

// TestWriteChunksToTempDir_AucunChunkRendUnRepertoireVideSansErreur — le pont lui-même ne juge
// pas l'absence de contenu (0 chunks = 0 trou) ; c'est aux LECTEURS (ScanFilmBipedPositions, etc.)
// de refuser un répertoire vide. Verrouille la frontière entre les deux responsabilités.
func TestWriteChunksToTempDir_AucunChunkRendUnRepertoireVideSansErreur(t *testing.T) {
	dir, cleanup, err := writeChunksToTempDir(nil)
	if err != nil {
		t.Fatalf("writeChunksToTempDir(nil): %v", err)
	}
	defer cleanup()
	if got := filmdec.CountFilmChunks(dir); got != 0 {
		t.Errorf("CountFilmChunks = %d, attendu 0", got)
	}
}

// ─── Composition pure ──────────────────────────────────────────────────────────────────────

func TestKillRefsFromDeaths_NeGardeQueLesDeuxIdentitesResolues(t *testing.T) {
	deaths := []persist.KillEventInsert{
		{TimeMS: 1000, FeedKillerXUID: "111", VictimXUID: "222"}, // les deux resolus : garde
		{TimeMS: 2000, FeedKillerXUID: "", VictimXUID: "222"},    // tueur non resolu (bot) : ecarte
		{TimeMS: 3000, FeedKillerXUID: "111", VictimXUID: ""},    // victime non resolue (bot) : ecarte
		{TimeMS: 4000, FeedKillerXUID: "abc", VictimXUID: "222"}, // xuid non numerique : ecarte
	}
	got := killRefsFromDeaths(deaths)
	if len(got) != 1 {
		t.Fatalf("attendu 1 KillRef, obtenu %d: %+v", len(got), got)
	}
	if got[0].KillerXUID != 111 || got[0].VictimXUID != 222 || got[0].TimeMS != 1000 {
		t.Errorf("KillRef inattendu: %+v", got[0])
	}
}

func TestToKillPositionRows_PositionsPartiellesRestentNil(t *testing.T) {
	positions := []replay.KillPosition{
		{
			KillRef: replay.KillRef{KillerXUID: 111, VictimXUID: 222, TimeMS: 1000},
			Killer:  &replay.Vec3{X: 1, Y: 2, Z: 3},
			// Victim volontairement nil : position non localisee.
		},
	}
	rows := toKillPositionRows("m1", positions)
	if len(rows) != 1 {
		t.Fatalf("attendu 1 ligne, obtenu %d", len(rows))
	}
	r := rows[0]
	if r.MatchID != "m1" || r.KillerXUID != "111" || r.TimeMS != 1000 {
		t.Errorf("ligne inattendue: %+v", r)
	}
	if r.KillerX == nil || *r.KillerX != 1 || r.KillerY == nil || *r.KillerY != 2 || r.KillerZ == nil || *r.KillerZ != 3 {
		t.Errorf("position tueur inattendue: X=%v Y=%v Z=%v", r.KillerX, r.KillerY, r.KillerZ)
	}
	if r.VictimX != nil {
		t.Errorf("VictimX devait rester nil (non localisee), obtenu %v", *r.VictimX)
	}
}

func TestParseXUID(t *testing.T) {
	cases := []struct {
		in   string
		want uint64
		ok   bool
	}{
		{"111", 111, true},
		{"", 0, false},
		{"abc", 0, false},
		{"xuid(123)", 0, false}, // forme brute jamais resolue ici : MatchIdentities.Resoudre l a deja traduite
	}
	for _, c := range cases {
		got, ok := parseXUID(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("parseXUID(%q) = (%d, %v), attendu (%d, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestRosterUint64_IgnoreLesNonNumeriques(t *testing.T) {
	got := rosterUint64([]string{"111", "abc", "222", ""})
	if len(got) != 2 || got[0] != 111 || got[1] != 222 {
		t.Errorf("roster = %v, attendu [111 222]", got)
	}
}

// ─── resolveMapBounds ──────────────────────────────────────────────────────────────────────

// fakeMapNames : port.ReplayMapNameRepo minimal, sans base.
type fakeMapNames struct {
	keys port.MatchMapKeys
	err  error
}

func (f fakeMapNames) MapKeysForMatch(context.Context, string) (port.MatchMapKeys, error) {
	return f.keys, f.err
}

func testMapQuantCatalog() *filmdec.MapQuantCatalog {
	return &filmdec.MapQuantCatalog{
		SchemaVersion: filmdec.MapQuantSchemaVersion,
		Maps: map[string]filmdec.MapQuantEntry{
			"catalyst": {
				Min: [3]float32{-100, -100, -100},
				Max: [3]float32{100, 100, 100},
			},
		},
	}
}

func TestResolveMapBounds_EssaieLesCandidatsDansLOrdre(t *testing.T) {
	c := &KillSourceCollector{
		mapNames:  fakeMapNames{keys: port.MatchMapKeys{Names: []string{"Carte Forge Inconnue", "Catalyst"}}},
		mapBounds: testMapQuantCatalog(),
	}
	entry, err := c.resolveMapBounds(context.Background(), "m1")
	if err != nil {
		t.Fatalf("resolveMapBounds: %v", err)
	}
	if entry.Min[0] != -100 {
		t.Errorf("entree inattendue: %+v", entry)
	}
}

func TestResolveMapBounds_RefuseSansCandidatConnu(t *testing.T) {
	c := &KillSourceCollector{
		mapNames:  fakeMapNames{keys: port.MatchMapKeys{Names: []string{"Carte Forge Inconnue"}}},
		mapBounds: testMapQuantCatalog(),
	}
	if _, err := c.resolveMapBounds(context.Background(), "m1"); err == nil {
		t.Fatal("attendu un refus (carte hors catalogue de bornes), obtenu nil")
	}
}

func TestResolveMapBounds_PropageLErreurDeResolutionDeCarte(t *testing.T) {
	c := &KillSourceCollector{
		mapNames:  fakeMapNames{err: errors.New("base indisponible")},
		mapBounds: testMapQuantCatalog(),
	}
	if _, err := c.resolveMapBounds(context.Background(), "m1"); err == nil {
		t.Fatal("attendu la propagation de l erreur, obtenu nil")
	}
}

// ─── collectPositions : chaque refus s'arrete AVANT toute ecriture ────────────────────────

// panicWriter : la preuve qu'un gate a refuse EN AMONT de toute tentative d'ecriture. Un appel
// fait echouer le test (panic non rattrapee) — c'est la propriete recherchee, pas un detail
// d'implementation : un gate qui laisserait passer un acquireShared() ecrirait potentiellement
// une passe partielle.
var panicWriter persist.SharedWriterFn = func(context.Context) (*sql.DB, func(), error) {
	panic("acquireShared ne doit jamais etre appele : un gate aurait du arreter la passe avant")
}

func killRefValide() []persist.KillEventInsert {
	return []persist.KillEventInsert{{TimeMS: 1000, FeedKillerXUID: "111", VictimXUID: "222"}}
}

func TestCollectPositions_CapabiliteAbsenteNeTenteAucuneEcriture(t *testing.T) {
	c := &KillSourceCollector{
		caps:          games.CapabilityMap{}, // film.kill_positions absente
		mapNames:      fakeMapNames{keys: port.MatchMapKeys{Names: []string{"Catalyst"}}},
		mapBounds:     testMapQuantCatalog(),
		acquireShared: panicWriter,
	}
	c.collectPositions(context.Background(), "m1", nil, MatchIdentities{}, killRefValide())
}

func TestCollectPositions_NonCableNeTenteAucuneEcriture(t *testing.T) {
	c := &KillSourceCollector{
		caps:          games.CapabilityMap{games.CapFilmKillPositions: games.CapSupported},
		acquireShared: panicWriter,
		// mapNames / mapBounds volontairement nil : WithPositionCapture jamais appele.
	}
	c.collectPositions(context.Background(), "m1", nil, MatchIdentities{}, killRefValide())
}

func TestCollectPositions_AucuneIdentiteResolueNeTenteAucuneEcriture(t *testing.T) {
	c := &KillSourceCollector{
		caps:          games.CapabilityMap{games.CapFilmKillPositions: games.CapSupported},
		mapNames:      fakeMapNames{keys: port.MatchMapKeys{Names: []string{"Catalyst"}}},
		mapBounds:     testMapQuantCatalog(),
		acquireShared: panicWriter,
	}
	deaths := []persist.KillEventInsert{{TimeMS: 1000, FeedKillerXUID: "", VictimXUID: ""}}
	c.collectPositions(context.Background(), "m1", nil, MatchIdentities{}, deaths)
}

func TestCollectPositions_CarteHorsCatalogueNeTenteAucuneEcriture(t *testing.T) {
	c := &KillSourceCollector{
		caps:          games.CapabilityMap{games.CapFilmKillPositions: games.CapSupported},
		mapNames:      fakeMapNames{keys: port.MatchMapKeys{Names: []string{"Carte Forge Inconnue"}}},
		mapBounds:     testMapQuantCatalog(),
		acquireShared: panicWriter,
	}
	c.collectPositions(context.Background(), "m1", nil, MatchIdentities{}, killRefValide())
}

// TestCollectPositions_FilmIllisibleNeTenteAucuneEcriture — bornes resolues, morts resolues,
// mais AUCUN chunk exploitable (« film illisible ») : le pont rend un repertoire vide et les
// lectures hors ligne refusent proprement — jamais de passe partielle ecrite.
func TestCollectPositions_FilmIllisibleNeTenteAucuneEcriture(t *testing.T) {
	c := &KillSourceCollector{
		caps:          games.CapabilityMap{games.CapFilmKillPositions: games.CapSupported},
		mapNames:      fakeMapNames{keys: port.MatchMapKeys{Names: []string{"Catalyst"}}},
		mapBounds:     testMapQuantCatalog(),
		acquireShared: panicWriter,
	}
	ids := MatchIdentities{XUIDs: []string{"111", "222"}}
	c.collectPositions(context.Background(), "m1", nil, ids, killRefValide())
}
