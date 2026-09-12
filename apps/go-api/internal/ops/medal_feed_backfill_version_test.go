package ops

// medal_feed_backfill_version_test.go — LA VERSION DECLAREE PAR LA SOURCE ARRIVE AU PARSEUR.
//
// # LE DEFAUT QU IL FERME
//
// Revue adversariale du 2026-09-12, constat P1-3 : le champ `version` de la doublure
// `filmSynthetique` n etait JAMAIS pose a autre chose que 0, et repasser 0 en dur au parseur dans
// la passe de rattrapage laissait la CI verte. La cause est structurelle, pas accidentelle :
// l appariement des medailles se fait sur le couple (xuid, instant), deux champs lus HORS du bloc
// de 60 octets, alors que la version ne commande que le decoupage du GAMERTAG a l interieur du
// bloc. Aucune correction ecrite par la passe n en depend — il n y avait donc rien a observer.
//
// # LE POINT D OBSERVATION
//
// [eventsDuFilm] isole le geste « parser le chunk AVEC la version que le film declare ». Ce test
// lui donne un bloc de version 40 (gamertag a l octet 12) et lit le gamertag qui en sort : avec
// la version transmise il vaut le nom attendu, avec 0 il vaut autre chose. C est la meme propriete
// que les bobines versionnees prouvent sur des octets reels pour les deux autres appelants
// (killsource et ScanDeaths) ; ici les octets sont fabriques, parce que la source de cette passe
// est une interface et qu aucun film n a a etre sur le disque.

import (
	"context"
	"encoding/binary"
	"testing"
	"unicode/utf16"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

// gamertagTemoin : le nom pose dans le bloc synthetique. Il n a rien de special, sinon d etre
// assez long pour qu un decoupage decale de douze octets ne puisse pas le rendre par hasard, et
// de tenir dans les 32 octets du champ (16 unites UTF-16 : au-dela il serait TRONQUE, et le test
// echouerait pour une raison qui n a rien a voir avec la version).
const gamertagTemoin = "Temoin Bobine 40"

// octetsEventV40 fabrique les 60 octets d un bloc event AU LAYOUT DES VERSIONS 39-40 :
// pad[0:12] | gamertag[12:44] | pad[44:47] | type_hint[47] | time_ms[48:52] big-endian |
// is_medal[55] | medal_type[59]. C est le layout documente en tete de
// `analysis/highlight_event_parser.go` — la seule difference avec [octetsEvent] (version >= 41)
// est l offset du gamertag, et c est exactement ce que ce test met sous garde.
func octetsEventV40(e evenementFilm, gamertag string) []byte {
	b := make([]byte, 60)
	unites := utf16.Encode([]rune(gamertag))
	if len(unites) > 16 {
		panic("gamertag de test trop long : le champ fait 32 octets (16 unites UTF-16)")
	}
	for i, u := range unites {
		binary.LittleEndian.PutUint16(b[12+2*i:], u)
	}
	b[47] = byte(e.typeHint)
	binary.BigEndian.PutUint32(b[48:52], uint32(e.timeMS)) //nolint:gosec // instant de test, borne
	if e.isMedal {
		b[55] = 1
	}
	b[59] = byte(e.medalType)
	return b
}

// chunkSynthetiqueV40 : l enveloppe de [chunkSynthetique], au layout 39-40.
func chunkSynthetiqueV40(e evenementFilm, gamertag string) []byte {
	var buf []byte
	buf = append(buf, make([]byte, 20)...) // rembourrage avant le marqueur de xuid
	xuidBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(xuidBytes, e.xuid)
	buf = append(buf, xuidBytes...)
	buf = append(buf, 0x2d, 0xc0)
	buf = append(buf, make([]byte, 30)...)
	buf = append(buf, octetsEventV40(e, gamertag)...)
	buf = append(buf, 0x00, 0x00, 0x2e, 0xe0) // marqueur de fin
	buf = append(buf, make([]byte, 10)...)
	return buf
}

// TestEventsDuFilmSuitLaVersionDeclaree — LE GARDE. La source declare 40, le bloc est au layout
// 39-40 : le gamertag doit sortir intact. Passer 0 au parseur lit douze octets trop tot.
func TestEventsDuFilmSuitLaVersionDeclaree(t *testing.T) {
	const matchID = "m-v40"
	ev := evenementFilm{xuid: 2535400000000001, typeHint: 50, timeMS: 12_345, isMedal: true, medalType: 26}
	source := filmSynthetique{
		parMatch: map[string][]byte{matchID: chunkSynthetiqueV40(ev, gamertagTemoin)},
		version:  40,
	}
	film, trouve, err := source.ChunkHighlight(context.Background(), matchID)
	if err != nil || !trouve {
		t.Fatalf("doublure de source : trouve=%v err=%v", trouve, err)
	}
	if film.MajorVersion == filmdec.FilmMajorVersionUnknown {
		t.Fatal("la doublure declare une version inconnue : le test ne prouverait rien")
	}
	events, err := eventsDuFilm(film)
	if err != nil {
		t.Fatalf("eventsDuFilm : %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("%d event(s) decode(s), 1 attendu — le chunk synthetique a change", len(events))
	}
	if events[0].EventType != analysis.EventTypeMedal {
		t.Fatalf("type d event %q, medaille attendue", events[0].EventType)
	}
	if events[0].Gamertag != gamertagTemoin {
		t.Fatalf(`LA VERSION DECLAREE PAR LA SOURCE N ARRIVE PLUS AU PARSEUR.

  version declaree : %d (bloc au layout 39-40, gamertag a l octet 12)
  gamertag lu      : %q
  attendu          : %q

Sur les versions 39-40 le gamertag vit a l octet 12 du bloc d event. Parser avec 0 le lit a
l octet 0 et rend du rembourrage (.ai/RAPPORT_BTB_2025_ABSTENTION_2026-09-12.md).

Verifier que eventsDuFilm passe film.MajorVersion a analysis.ParseHighlightEvents, et que la
source du CLI (cmd/levelup/cmd_backfill_medailles_feed.go) lit bien la version du registre.`,
			film.MajorVersion, events[0].Gamertag, gamertagTemoin)
	}
}

// TestEventsDuFilmVersionInconnue : la contre-epreuve. Version 0 = decoupage historique ; le meme
// bloc 39-40 ne rend PAS le nom. Sans ce cas, un parseur qui ignorerait la version rendrait le bon
// gamertag des deux cotes et le garde ci-dessus ne prouverait rien.
func TestEventsDuFilmVersionInconnue(t *testing.T) {
	ev := evenementFilm{xuid: 2535400000000001, typeHint: 50, timeMS: 12_345, isMedal: true, medalType: 26}
	film := FilmHighlight{
		Chunk:        chunkSynthetiqueV40(ev, gamertagTemoin),
		MajorVersion: filmdec.FilmMajorVersionUnknown,
	}
	events, err := eventsDuFilm(film)
	if err != nil {
		t.Fatalf("eventsDuFilm : %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("%d event(s) decode(s), 1 attendu", len(events))
	}
	if events[0].Gamertag == gamertagTemoin {
		t.Fatal("le decoupage historique rend le gamertag du layout 39-40 : les deux decoupages " +
			"se confondent, et aucun test de ce fichier ne prouve plus rien")
	}
}
