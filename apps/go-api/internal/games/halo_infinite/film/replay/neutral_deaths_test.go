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
	out := keepNeutralDeathsOfPublishedTracks(in, tracks, nil)
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
		[]NeutralDeath{{XUID: "A", FeedMs: 1_000, Kind: ""}}, []Track{{XUID: "A"}}, nil)
	if out != nil {
		t.Fatalf("une mort sans type a été publiée : %+v", out)
	}
}

// TestNeutralDeathsAbsentesRendentNil : le champ est omitempty, et un tableau vide non nil
// se sérialiserait quand même en `[]`. L'absence doit rester une absence.
func TestNeutralDeathsAbsentesRendentNil(t *testing.T) {
	if out := keepNeutralDeathsOfPublishedTracks(nil, []Track{{XUID: "A"}}, nil); out != nil {
		t.Fatalf("entrée vide : attendu nil, obtenu %+v", out)
	}
}

// TestMortNeutreDunJoueurSansVieNommeeEstPubliee — MEME CORRECTIF, MEME JOUR, MEME HELPER que
// les actions d'objectif (constat P1-3 de l'audit du 2026-09-06). Ce filtre et celui des
// actions d'objectif etaient les deux SEULS des treize filtres « piste publiee » du paquet a
// cadencer sur un nom LU ; les onze autres cadencent sur le SLOT.
//
// MUTATION : revenir a l'index bati sur `tr.XUID != ""` rougit (« publiees = 0, attendu 1 »).
func TestMortNeutreDunJoueurSansVieNommeeEstPubliee(t *testing.T) {
	in := []NeutralDeath{{XUID: "42", FeedMs: 1_000, Kind: "environment", Img: "/s/e.png"}}
	tracks := []Track{{Slot: 536}} // piste PUBLIEE, nommage echoue

	out := keepNeutralDeathsOfPublishedTracks(in, tracks, map[uint32]uint64{536: 42})
	if len(out) != 1 {
		t.Fatalf("publiees = %d, attendu 1 : le pont nomme le slot 536", len(out))
	}
	// CONTRE-EPREUVE : sans pont, la mort reste ecartee — on n'invente aucun joueur.
	if out := keepNeutralDeathsOfPublishedTracks(in, tracks, map[uint32]uint64{999: 42}); out != nil {
		t.Errorf("pont sur un autre slot : %+v publiee(s), attendu aucune", out)
	}
}
