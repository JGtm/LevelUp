package killsource

// botmeta_equipe_decode_test.go — LA LECTURE DE L EQUIPE DES BOTS EST BRANCHEE SUR LE DECODAGE
// (revue adversariale du lot « toute entree du roster a l'equipe que le film ecrit », 2026-10-06).
//
// Les tests de `botmeta_equipe_test.go` appellent la lecture directement ; les goldens de films a bots
// sautent sans fixture, et la mini-bobine versionnee n a pas de bot. Ce test passe donc par [Decode],
// en CI, sur la mini-bobine a laquelle s ajoute UN paquet BOT_METADATA fabrique par la grammaire de
// l ecrivain (cf. [payloadEcrit]), insere avant le CHUNK_END de son premier chunk de donnees.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : dans `prepare` (decode.go), donner au roster `loadBotMeta`
// au lieu de `botsDuFilm` — le bot y entre, sans equipe.

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// enTeteDePaquet : `[u16 type][2 octets][u32 taille][u64 horodatage]`, petit-boutiste (couche source).
const enTeteDePaquet = 16

// paquetDeType12 fabrique un paquet BOT_METADATA complet, en-tete compris.
func paquetDeType12(ts uint64, payload []byte) []byte {
	p := make([]byte, enTeteDePaquet+len(payload))
	binary.LittleEndian.PutUint16(p[0:], grammar.PacketTypeBotMetadata)
	binary.LittleEndian.PutUint32(p[4:], uint32(len(payload))) //nolint:gosec // payload de test borne
	binary.LittleEndian.PutUint64(p[8:], ts)
	copy(p[enTeteDePaquet:], payload)
	return p
}

// insererAvantLaFin glisse `paquet` avant le CHUNK_END d un chunk decompresse, a l horodatage du
// dernier paquet qui le precede.
func insererAvantLaFin(t *testing.T, chunk []byte, payload []byte) []byte {
	t.Helper()
	var ts uint64
	for off := 0; off+enTeteDePaquet <= len(chunk); {
		typ := int(binary.LittleEndian.Uint16(chunk[off:]))
		taille := int(binary.LittleEndian.Uint32(chunk[off+4:]))
		if typ == 7 { // CHUNK_END, le terminateur du chunk
			out := append(append([]byte{}, chunk[:off]...), paquetDeType12(ts, payload)...)
			return append(out, chunk[off:]...)
		}
		ts = binary.LittleEndian.Uint64(chunk[off+8:])
		off += enTeteDePaquet + taille
	}
	t.Fatal("chunk sans CHUNK_END : la mini-bobine a change de forme")
	return nil
}

func TestDecodeLitLEquipeDesBots(t *testing.T) {
	noms, err := filepath.Glob(filepath.Join(miniBobineDir, "chunk_*.bin"))
	if err != nil || len(noms) != miniBobineChunks {
		t.Fatalf("mini-bobine : %d chunk(s) (%v), %d attendus", len(noms), err, miniBobineChunks)
	}
	chunks := make(source.MemoryChunks, len(noms))
	for i, nom := range noms {
		brut, err := os.ReadFile(nom) //nolint:gosec // chemin de la bobine versionnee
		if err != nil {
			t.Fatalf("%s : %v", nom, err)
		}
		chunks[i] = source.Inflate(brut)
	}
	chunks[1] = insererAvantLaFin(t, chunks[1], payloadEcrit(persoBitsHI113, sandwolf))
	src, err := source.Load(chunks, nil)
	if err != nil {
		t.Fatalf("source : %v", err)
	}
	res, err := Decode(t.Context(), miniBobineFilm, src, optionsDeLaBobine(t))
	if err != nil {
		t.Fatalf("Decode : %v", err)
	}
	var lu *BotEntry
	for i, b := range res.Roster.Bots {
		if b.Name == "343 Sandwolf" {
			lu = &res.Roster.Bots[i]
		}
	}
	if lu == nil {
		t.Fatalf("le bot fabrique n atteint pas le roster du decodage : %+v", res.Roster.Bots)
	}
	if lu.Team == nil || *lu.Team != 1 {
		t.Fatalf("equipe publiee par Decode %v, attendu 1 : la lecture n est pas branchee", lu.Team)
	}
	if res.Roster.BotEquipes != (EquipesDesBots{Lues: 1}) {
		t.Fatalf("bilan publie %+v, attendu une equipe lue et rien d autre", res.Roster.BotEquipes)
	}
}
