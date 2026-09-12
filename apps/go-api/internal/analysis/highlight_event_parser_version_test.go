package analysis

// highlight_event_parser_version_test.go — LE DECOUPAGE DU GAMERTAG SUIT LA VERSION DECLAREE.
//
// CE QUE CES TESTS FIGENT. Le bloc d'event de 60 octets porte le gamertag a b[0:32] sur les
// versions <= 38 et >= 41, et a b[12:44] sur les versions 39-40 (films de mars a novembre 2025).
// Le parseur ne DEVINE pas laquelle s'applique : la version lui est passee, lue par l'appelant
// dans l'en-tete du registre du film (`filmdec.FilmMajorVersionFromHeader`) ou dans le manifeste
// de l'API. Les tests ci-dessous verrouillent les trois cas qui comptent : la version 39-40 rend
// les noms, la version >= 41 sur le meme flux rend le rembourrage (et c'est ce qui s'est
// produit en production tant que 0 etait passe en dur), et la version 0 — film sans registre —
// garde le comportement historique « gamertag en tete ».
//
// Cause et mesures : .ai/RAPPORT_BTB_2025_ABSTENTION_2026-09-12.md.

import (
	"encoding/binary"
	"testing"
)

// Les trois versions de reference de ces tests. Elles NOMMENT les deux decoupages plutot que de
// semer 39/41/0 dans les appels.
const (
	// versionInconnue : le film ne porte pas son registre, l'appelant n'a rien pu lire.
	versionInconnue = 0
	// versionGamertagDecale : une version 39-40, gamertag a b[12:44].
	versionGamertagDecale = 40
	// versionGamertagEnTete : une version >= 41, gamertag a b[0:32].
	versionGamertagEnTete = 41
)

// rembourrageDecale est ce que la lecture « en tete » ramene d'un bloc 39-40 : les douze premiers
// octets sont un rembourrage IDENTIQUE d'un event a l'autre, termine par un u16 nul, donc
// `decodeUTF16LE` rend la MEME chaine pour tous les joueurs. C'est la forme mesuree sur les films
// 2025 — 2 gamertags distincts pour 24 a 27 XUID — et c'est elle qui effondrait le roster humain.
const rembourrageDecale = "##"

// blocEventDecale : les 60 octets d'un event dont le gamertag vit a b[12:44] (versions 39-40).
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

// fluxDecale : un chunk de `noms` events au decoupage 39-40, compresse comme un vrai chunk.
func fluxDecale(noms []string) []byte {
	var flux []byte
	for i, nom := range noms {
		xuid := uint64(2_500_000_000_000_001 + i)
		flux = append(flux, buildRawChunkWithEventBytes(xuid,
			blocEventDecale(nom, typeHintKill, 1000*(i+1)))...)
	}
	return zlibCompress(flux)
}

// gamertagsDistincts rend l'ensemble des gamertags lus.
func gamertagsDistincts(events []HighlightEvent) map[string]bool {
	vus := map[string]bool{}
	for _, ev := range events {
		vus[ev.Gamertag] = true
	}
	return vus
}

// TestVersionDecalee_RendLesNoms : un flux 39-40 LU AVEC SA VERSION rend un nom par joueur.
func TestVersionDecalee_RendLesNoms(t *testing.T) {
	noms := []string{"AlphaUn", "BetaDeux", "GammaTrois", "DeltaQuatre"}
	events, err := ParseHighlightEvents(fluxDecale(noms), versionGamertagDecale)
	if err != nil {
		t.Fatalf("ParseHighlightEvents: %v", err)
	}
	if len(events) != len(noms) {
		t.Fatalf("events = %d, want %d", len(events), len(noms))
	}
	vus := gamertagsDistincts(events)
	if len(vus) != len(noms) {
		t.Fatalf("gamertags distincts = %d (%v), want %d", len(vus), vus, len(noms))
	}
	for _, nom := range noms {
		if !vus[nom] {
			t.Errorf("gamertag %q absent : %v", nom, vus)
		}
	}
}

// TestVersionFausseSurFluxDecale_RendLeRembourrage : le MEME flux lu avec la mauvaise version
// rend le rembourrage, pour tous les joueurs. C'est exactement ce que produisaient les appelants
// qui passaient 0 en dur, et la raison pour laquelle la version doit etre LUE.
func TestVersionFausseSurFluxDecale_RendLeRembourrage(t *testing.T) {
	noms := []string{"AlphaUn", "BetaDeux", "GammaTrois"}
	flux := fluxDecale(noms)
	for _, version := range []int{versionInconnue, versionGamertagEnTete} {
		events, err := ParseHighlightEvents(flux, version)
		if err != nil {
			t.Fatalf("ParseHighlightEvents(%d): %v", version, err)
		}
		if len(events) != len(noms) {
			t.Fatalf("version %d : events = %d, want %d", version, len(events), len(noms))
		}
		for _, ev := range events {
			if ev.Gamertag != rembourrageDecale {
				t.Fatalf("version %d : gamertag = %q, want %q", version, ev.Gamertag, rembourrageDecale)
			}
		}
	}
}

// TestVersionEnTete_InchangeeParLaVersionInconnue : sur un flux >= 41, lire sans version rend
// EXACTEMENT ce que rend la version declaree. Le parc 2026 est donc insensible a l'absence de
// registre, et le comportement historique est preserve.
func TestVersionEnTete_InchangeeParLaVersionInconnue(t *testing.T) {
	noms := []string{"AlphaUn", "BetaDeux", "GammaTrois", "DeltaQuatre"}
	var flux []byte
	for i, nom := range noms {
		xuid := uint64(2_500_000_000_000_001 + i)
		flux = append(flux, buildRawChunk(xuid, nom, typeHintKill, 1000*(i+1), false, 0)...)
	}
	compresse := zlibCompress(flux)

	sansVersion, err := ParseHighlightEvents(compresse, versionInconnue)
	if err != nil {
		t.Fatalf("ParseHighlightEvents(0): %v", err)
	}
	avecVersion, err := ParseHighlightEvents(compresse, versionGamertagEnTete)
	if err != nil {
		t.Fatalf("ParseHighlightEvents(41): %v", err)
	}
	if len(sansVersion) != len(avecVersion) || len(sansVersion) != len(noms) {
		t.Fatalf("events : sans version %d, avec version %d, want %d",
			len(sansVersion), len(avecVersion), len(noms))
	}
	for i := range sansVersion {
		if sansVersion[i].Gamertag != avecVersion[i].Gamertag {
			t.Errorf("event %d : sans version %q, avec version %q",
				i, sansVersion[i].Gamertag, avecVersion[i].Gamertag)
		}
	}
}
