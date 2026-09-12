package analysis

// highlight_event_parser_version_test.go — LA RESOLUTION DU DECOUPAGE DU GAMERTAG QUAND LA
// VERSION DU FILM EST INCONNUE (correctif du 2026-09-12).
//
// CE QUE CES TESTS FIGENT. Trois appelants passent `filmMajorVersion = 0` faute de manifeste
// (killsource, replay/deaths_source, ops/medal_feed_backfill) ; le cache de films local pose la
// meme valeur (« legacy cache n'a pas la version »). Avant le correctif, 0 signifiait
// « decoupage en tete » (b[0:32]) : sur un film de version 39-40, le gamertag vit a b[12:44] et
// la lecture ne rendait que du rembourrage. Les tests ci-dessous verifient les deux sens :
// un flux 39-40 doit etre RESOLU, un flux >= 41 doit rester INTACT.

import (
	"encoding/binary"
	"testing"
)

// blocEventDecale : les 60 octets d'un event dont le gamertag vit a b[12:44] (versions 39-40).
//
// LES DOUZE PREMIERS OCTETS REPRODUISENT CE QUE LA LECTURE FAUTIVE RAMENAIT : un rembourrage
// identique d'un event a l'autre, termine par un u16 nul — `decodeUTF16LE` s'arrete donc au
// premier zero et rend la MEME chaine pour tous les joueurs. C'est la forme mesuree sur les
// films 2025 (2 gamertags distincts pour 24 a 27 XUID), et c'est elle qui effondrait le roster.
func blocEventDecale(gamertag string, typeHint, timeMS int) []byte {
	b := make([]byte, eventDataBytes)
	binary.LittleEndian.PutUint16(b[0:], uint16('#'))
	binary.LittleEndian.PutUint16(b[2:], uint16('#'))
	runes := []rune(gamertag)
	for i := 0; i < 16 && i < len(runes); i++ {
		binary.LittleEndian.PutUint16(b[12+i*2:], uint16(runes[i]))
	}
	b[47] = byte(typeHint)
	binary.BigEndian.PutUint32(b[48:52], uint32(timeMS))
	return b
}

// TestScanEvents_VersionInconnue_ResoutLeDecoupageDecale : un flux 39-40 lu sans version.
func TestScanEvents_VersionInconnue_ResoutLeDecoupageDecale(t *testing.T) {
	noms := []string{"AlphaUn", "BetaDeux", "GammaTrois", "DeltaQuatre"}
	var flux []byte
	for i, nom := range noms {
		xuid := uint64(2_500_000_000_000_001 + i)
		flux = append(flux, buildRawChunkWithEventBytes(xuid,
			blocEventDecale(nom, typeHintKill, 1000*(i+1)))...)
	}

	events, err := ParseHighlightEvents(zlibCompress(flux), versionUnknown)
	if err != nil {
		t.Fatalf("ParseHighlightEvents: %v", err)
	}
	if len(events) != len(noms) {
		t.Fatalf("events = %d, want %d", len(events), len(noms))
	}
	vus := map[string]bool{}
	for _, ev := range events {
		vus[ev.Gamertag] = true
	}
	if len(vus) != len(noms) {
		t.Fatalf("gamertags distincts = %d (%v), want %d — le decoupage 39-40 n'a pas ete resolu",
			len(vus), vus, len(noms))
	}
	for _, nom := range noms {
		if !vus[nom] {
			t.Errorf("gamertag %q absent : %v", nom, vus)
		}
	}
}

// TestScanEvents_VersionInconnue_NeCassePasLeDecoupageEnTete : un flux >= 41 lu sans version
// doit rendre EXACTEMENT ce qu'il rendait avant le correctif. C'est la moitie qui protege le
// parc 2026 — la resolution ne doit jamais preferer le decalage a tort.
func TestScanEvents_VersionInconnue_NeCassePasLeDecoupageEnTete(t *testing.T) {
	noms := []string{"AlphaUn", "BetaDeux", "GammaTrois", "DeltaQuatre"}
	var flux []byte
	for i, nom := range noms {
		xuid := uint64(2_500_000_000_000_001 + i)
		flux = append(flux, buildRawChunk(xuid, nom, typeHintKill, 1000*(i+1), false, 0)...)
	}
	compresse := zlibCompress(flux)

	sansVersion, err := ParseHighlightEvents(compresse, versionUnknown)
	if err != nil {
		t.Fatalf("ParseHighlightEvents(0): %v", err)
	}
	avecVersion, err := ParseHighlightEvents(compresse, versionGamertagEnTete)
	if err != nil {
		t.Fatalf("ParseHighlightEvents(41): %v", err)
	}
	if len(sansVersion) != len(avecVersion) {
		t.Fatalf("events : sans version %d, avec version %d", len(sansVersion), len(avecVersion))
	}
	for i := range sansVersion {
		if sansVersion[i].Gamertag != avecVersion[i].Gamertag {
			t.Errorf("event %d : sans version %q, avec version %q — la resolution a prefere le "+
				"decalage a tort", i, sansVersion[i].Gamertag, avecVersion[i].Gamertag)
		}
	}
}

// TestScanEvents_VersionDeclaree_NeSeResoutPas : une version DECLAREE fait foi, meme quand
// l'autre decoupage rendrait davantage de noms. Le manifeste tranche, la mesure ne le corrige
// pas — sans quoi le correctif deviendrait une heuristique appliquee partout.
func TestScanEvents_VersionDeclaree_NeSeResoutPas(t *testing.T) {
	var flux []byte
	for i, nom := range []string{"AlphaUn", "BetaDeux", "GammaTrois"} {
		xuid := uint64(2_500_000_000_000_001 + i)
		flux = append(flux, buildRawChunkWithEventBytes(xuid,
			blocEventDecale(nom, typeHintKill, 1000*(i+1)))...)
	}
	events, err := ParseHighlightEvents(zlibCompress(flux), versionGamertagEnTete)
	if err != nil {
		t.Fatalf("ParseHighlightEvents: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("aucun event")
	}
	for _, ev := range events {
		if ev.Gamertag != "##" {
			t.Fatalf("gamertag = %q, want le rembourrage \"##\" — une version declaree ne doit "+
				"PAS declencher la resolution par mesure", ev.Gamertag)
		}
	}
}
