package replay

import "testing"

// TestNeutralDeathsSuiventLesTracesPubliees : la règle commune à tous les calques — on ne
// publie que ce qui rencontrera une trajectoire. Une entrée sans piste ne serait pas fausse,
// elle serait morte : le client déduit ces lignes DE SES PISTES.
func TestNeutralDeathsSuiventLesTracesPubliees(t *testing.T) {
	tracks := []Track{{XUID: "A"}, {XUID: ""}, {XUID: "B"}}
	in := []NeutralDeath{
		{XUID: "A", FeedMs: 1_000, Kind: "environment", Img: "/s/e.png", Tinted: true},
		{XUID: "Z", FeedMs: 2_000, Kind: "suicide", Img: "/s/s.png", Tinted: true},
		{XUID: "B", FeedMs: 3_000, Kind: "suicide", Img: "/s/s.png", Tinted: true},
	}
	out := keepNeutralDeathsOfPublishedTracks(in, tracks)
	if len(out) != 2 {
		t.Fatalf("publiees = %d, attendu 2 (A et B ; Z n'a aucune trace)", len(out))
	}
	if out[0].XUID != "A" || out[1].XUID != "B" {
		t.Errorf("ordre/contenu inattendu : %+v", out)
	}
}

// TestNeutralDeathSansTypeNEntrePas verrouille LA règle dure du lot : une mort dont la nature
// n'est pas établie ne descend pas jusqu'au fil. Elle y prendrait la place du repère neutre
// sans rien dire de plus, et une entrée vide invite le client à improviser une icône.
func TestNeutralDeathSansTypeNEntrePas(t *testing.T) {
	out := keepNeutralDeathsOfPublishedTracks(
		[]NeutralDeath{{XUID: "A", FeedMs: 1_000, Kind: ""}}, []Track{{XUID: "A"}})
	if out != nil {
		t.Fatalf("une mort sans type a été publiée : %+v", out)
	}
}

// TestNeutralDeathsAbsentesRendentNil : le champ est omitempty, et un tableau vide non nil
// se sérialiserait quand même en `[]`. L'absence doit rester une absence.
func TestNeutralDeathsAbsentesRendentNil(t *testing.T) {
	if out := keepNeutralDeathsOfPublishedTracks(nil, []Track{{XUID: "A"}}); out != nil {
		t.Fatalf("entrée vide : attendu nil, obtenu %+v", out)
	}
}
