package killcollector

// positions_test.go — LE CHARGEMENT DU FILM (chunks synthétiques -> `filmsource.Film`), LA
// COMPOSITION PURE, ET LE REFUS PROPRE. Aucun test ici n'ouvre de base ni ne lit de film réel —
// c'est le rôle de positions_integration_test.go (fixture réelle, gate KILLSOURCE_FIXTURES) et de
// kill_position_persister_test.go (persister, :memory:). Ce fichier verrouille exactement ce que
// G.2bis a ajouté : l'entrée des chunks, le filtrage des identités, la traduction en lignes, et
// que CHAQUE refus s'arrête AVANT toute tentative d'écriture (acquireShared panique s'il est
// appelé). Le pont disque qu'il verrouillait a disparu au lot 1 (item 1.6) : son seul contrôle
// utile, le refus d'une séquence trouée, est verrouillé ici sous son nouveau nom.

import (
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
	"errors"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/haloclient"
)

// zlibCompressForTest compresse b, pour verifier qu'un chunk COMPRESSE (la forme du cache
// herite) arrive decompresse aux balayages — c'est `filmsource` qui inflate, une seule fois.
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

// ─── FilmOf : le chargement, et refuserSequenceTrouee : le refus ───────────────────────────
//
// LE PONT DISQUE A DISPARU AU LOT 1 (PLAN_CUISSON_PERF, item 1.6) : les chunks téléchargés ne
// sont plus recopiés dans un répertoire temporaire pour être relus quatre fois. `FilmOf` les
// charge une fois (`filmsource`), et le seul contrôle qui protégeait d'une position FAUSSE — le
// refus d'une séquence trouée — est conservé tel quel, en mémoire.

func TestFilmOf_ChargeLesChunksALeurIndex(t *testing.T) {
	chunks := []haloclient.FilmChunk{
		{Index: 1, ChunkType: 2, StartMS: 100, Data: []byte("chunk-un-donnees")},
		{Index: 2, ChunkType: 2, StartMS: 200, Data: []byte("chunk-deux-donnees")},
	}
	film, err := FilmOf(chunks)
	if err != nil {
		t.Fatalf("FilmOf: %v", err)
	}
	if got := film.NumChunks(); got != 3 {
		t.Fatalf("NumChunks = %d, attendu 3 (dimensionne sur l index MAX, en-tete compris)", got)
	}
	if got := string(film.Chunk(1)); got != "chunk-un-donnees" {
		t.Errorf("chunk 1 = %q", got)
	}
	if got := string(film.Chunk(2)); got != "chunk-deux-donnees" {
		t.Errorf("chunk 2 = %q", got)
	}
	// LES METADONNEES SONT POSITIONNELLES et portent le manifeste : c'est par elles que les
	// balayages traduisent un NUMERO de chunk en position (filmdec.FilmChunkNumbers).
	meta := film.Meta()
	if len(meta) != 3 {
		t.Fatalf("Meta = %d entrees, attendu 3", len(meta))
	}
	for i, m := range meta {
		if m.Index != i {
			t.Errorf("Meta[%d].Index = %d, attendu %d (la position EST le numero ici)", i, m.Index, i)
		}
	}
	if meta[2].ChunkType != 2 || meta[2].StartMS != 200 {
		t.Errorf("Meta[2] = %+v, attendu le type et le debut du manifeste", meta[2])
	}
}

// TestFilmOf_ZlibRoundTrip — les chunks descendent COMPRESSÉS du cache hérité et CLAIRS des
// téléchargements récents. `filmsource` décompresse à la charge, une fois pour tous les lecteurs
// (avant, chacune des quatre lectures repayait cette décompression).
func TestFilmOf_ZlibRoundTrip(t *testing.T) {
	compressed := zlibCompressForTest(t, []byte("payload-compresse"))
	film, err := FilmOf([]haloclient.FilmChunk{{Index: 1, Data: compressed}})
	if err != nil {
		t.Fatalf("FilmOf: %v", err)
	}
	if got := string(film.Chunk(1)); got != "payload-compresse" {
		t.Errorf("chunk decompresse = %q, attendu %q", got, "payload-compresse")
	}
}

// TestFilmOf_NumerosDeChunksVusParLesBalayages — LE CONTRAT QUI REMPLACE LE PONT DISQUE, et le
// seul qui pouvait se perdre en route : les quatre balayages parcourent les chunks de DONNÉES par
// NUMÉRO (`filmdec.FilmChunkNumbers`), là où ils comptaient `filmdec.CountFilmChunks(dir)` — donc
// 1..N depuis chunk_01. Le film chargé en mémoire doit rendre exactement les mêmes numéros, sinon
// `ScanDeaths` (qui prend le DERNIER numéro comme chunk du kill-feed) et `ScanClockOrigin` (qui
// lit le numéro 1) changeraient de cible sans rien signaler.
func TestFilmOf_NumerosDeChunksVusParLesBalayages(t *testing.T) {
	film, err := FilmOf([]haloclient.FilmChunk{
		{Index: 0, ChunkType: 1, Data: []byte("entete")},
		{Index: 1, ChunkType: 2, Data: []byte("replication-1")},
		{Index: 2, ChunkType: 2, Data: []byte("replication-2")},
		{Index: 3, ChunkType: 3, Data: []byte("killfeed")},
	})
	if err != nil {
		t.Fatalf("FilmOf: %v", err)
	}
	nums := filmdec.FilmChunkNumbers(film)
	if len(nums) != 3 || nums[0] != 1 || nums[2] != 3 {
		t.Fatalf("numeros = %v, attendu [1 2 3] (le registre exclu, le kill-feed en dernier)", nums)
	}
	// Le NUMÉRO adresse bien la position : c'est ce que `FilmChunkAt` traduit pour les balayages.
	raw, _, ok := filmdec.FilmChunkAt(film, 3)
	if !ok || string(raw) != "killfeed" {
		t.Errorf("chunk numero 3 = %q (ok=%v), attendu le kill-feed", raw, ok)
	}
}

// TestRefuserSequenceTrouee_RefuseUnTrouDeSequence — un index manquant ferait lire les quatre
// balayages sur un film AMPUTÉ, en silence (un chunk vide ne rend aucun paquet) : le contrôle
// refuse plutôt que de laisser passer une lecture partielle plausible.
func TestRefuserSequenceTrouee_RefuseUnTrouDeSequence(t *testing.T) {
	film, err := FilmOf([]haloclient.FilmChunk{
		{Index: 1, Data: []byte("a")},
		{Index: 3, Data: []byte("c")}, // 2 absent
	})
	if err != nil {
		t.Fatalf("FilmOf: %v", err)
	}
	if err := refuserSequenceTrouee(film); err == nil {
		t.Fatal("attendu un refus (trou de sequence), obtenu nil")
	}
}

// TestRefuserSequenceTrouee_ChunkVideEstUnTrou — un chunk DÉCLARÉ mais sans octets (Data vide)
// est un trou au même titre qu'un index manquant : il ne rend aucun paquet.
func TestRefuserSequenceTrouee_ChunkVideEstUnTrou(t *testing.T) {
	film, err := FilmOf([]haloclient.FilmChunk{
		{Index: 1, Data: []byte("a")},
		{Index: 2, Data: nil},
	})
	if err != nil {
		t.Fatalf("FilmOf: %v", err)
	}
	if err := refuserSequenceTrouee(film); err == nil {
		t.Fatal("attendu un refus (chunk 2 vide = trou), obtenu nil")
	}
}

// TestRefuserSequenceTrouee_AucunChunkDeDonneesPasse — le contrôle ne juge pas l'absence de
// contenu (0 chunk de données = 0 trou) ; c'est aux LECTEURS (ScanBipedPositions, etc.) de
// refuser un film vide. Verrouille la frontière entre les deux responsabilités, comme le faisait
// le pont disque avant lui.
func TestRefuserSequenceTrouee_AucunChunkDeDonneesPasse(t *testing.T) {
	film, err := FilmOf(nil)
	if err != nil {
		t.Fatalf("FilmOf(nil): %v", err)
	}
	if err := refuserSequenceTrouee(film); err != nil {
		t.Errorf("un film sans chunk de donnees doit passer ce controle, obtenu : %v", err)
	}
}

// TestRefuserSequenceTrouee_FilmNil — le film absent est refusé ici, jamais déréférencé plus bas.
func TestRefuserSequenceTrouee_FilmNil(t *testing.T) {
	if err := refuserSequenceTrouee(nil); err == nil {
		t.Fatal("attendu un refus (film nil), obtenu nil")
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
