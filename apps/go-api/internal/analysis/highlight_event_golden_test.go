package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/highlightevent"
)

// Golden de NON-REGRESSION du temps fort (item 2.5.h du PLAN_DECODEUR_FILM_2026-09-13).
//
// POURQUOI. Le type `analysis.HighlightEvent` et le vocabulaire `EventType*` quittent
// `internal/analysis` pour `internal/domain/highlightevent` : un deplacement PUR de
// declarations. « Pur » n'est pas une intention, cela se PROUVE — ce golden fige AVANT le
// mouvement les deux seules choses qu'un deplacement pourrait abimer :
//
//  1. LA FORME du type (ordre, noms, types et tags de ses champs) et la VALEUR des quatre
//     constantes de vocabulaire — un renommage, une reordonnance ou un tag perdu rougit ;
//  2. LA SORTIE du lecteur sur la fixture versionnee `testdata/v41_chunk_he.bin` (chunk
//     reel FilmMajorVersion=41) : comptes, joueurs distincts, et empreinte SHA-256 de la
//     serialisation champ a champ de TOUS les evenements, dans l'ordre rendu.
//
// Le fichier golden ne porte aucun chemin de paquet : il DOIT rester identique avant et
// apres le deplacement. L'empreinte, et non les gamertags en clair, est ce qui est fige :
// le match est reel, ses joueurs n'ont pas a etre recopies dans le depot.
const goldenHighlightEventPath = "testdata/highlight_event_golden.txt"

func TestGoldenHighlightEventFormeEtSortie(t *testing.T) {
	got := goldenHighlightEventContent(t)

	want, err := os.ReadFile(goldenHighlightEventPath)
	if err != nil {
		t.Fatalf("lecture du golden %s: %v\n--- calcule ---\n%s", goldenHighlightEventPath, err, got)
	}
	if got != string(want) {
		t.Fatalf("golden %s different.\n--- attendu ---\n%s\n--- obtenu ---\n%s",
			goldenHighlightEventPath, string(want), got)
	}
}

// goldenHighlightEventContent rend le texte fige : forme du type, vocabulaire, sortie du
// lecteur sur la fixture reelle.
func goldenHighlightEventContent(t *testing.T) string {
	t.Helper()

	var b strings.Builder
	b.WriteString("# forme du type (ordre, nom, type, tag)\n")
	rt := reflect.TypeOf(highlightevent.HighlightEvent{})
	fmt.Fprintf(&b, "champs=%d\n", rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		fmt.Fprintf(&b, "%d %s %s tag=%q\n", i, f.Name, f.Type.String(), string(f.Tag))
	}

	b.WriteString("# vocabulaire des types d'evenement\n")
	for _, kv := range [][2]string{
		{"EventTypeKill", highlightevent.EventTypeKill},
		{"EventTypeDeath", highlightevent.EventTypeDeath},
		{"EventTypeMedal", highlightevent.EventTypeMedal},
		{"EventTypeMode", highlightevent.EventTypeMode},
	} {
		fmt.Fprintf(&b, "%s=%s\n", kv[0], kv[1])
	}

	b.WriteString("# sortie du lecteur sur testdata/v41_chunk_he.bin (version 41)\n")
	if len(realV41Chunk) == 0 {
		t.Fatal("fixture testdata/v41_chunk_he.bin manquante")
	}
	events, err := ParseHighlightEvents(realV41Chunk, 41)
	if err != nil {
		t.Fatalf("ParseHighlightEvents: %v", err)
	}

	h := sha256.New()
	typeCount := map[string]int{}
	xuids := map[uint64]struct{}{}
	for _, ev := range events {
		typeCount[ev.EventType]++
		xuids[ev.XUID] = struct{}{}
		fmt.Fprintf(h, "%d|%s|%s|%d|%t|%d|%d\n",
			ev.XUID, ev.Gamertag, ev.EventType, ev.TypeHint, ev.IsMedal, ev.TimeMS, ev.MedalType)
	}
	fmt.Fprintf(&b, "evenements=%d\n", len(events))
	fmt.Fprintf(&b, "joueurs_distincts=%d\n", len(xuids))
	types := make([]string, 0, len(typeCount))
	for k := range typeCount {
		types = append(types, k)
	}
	sort.Strings(types)
	for _, k := range types {
		fmt.Fprintf(&b, "type %s=%d\n", k, typeCount[k])
	}
	fmt.Fprintf(&b, "sha256=%s\n", hex.EncodeToString(h.Sum(nil)))
	return b.String()
}
