package killsummary

// killsummary_test.go — LES CAS DU GRAIN ET DES COMPTES (ADR 0039, décision 3), un par propriété
// que les lecteurs de l'étape de bascule relisent : bots, assistant anonyme, cas bot-bot,
// « mesurée sans assistant », NULL gardés comme groupes, passe non publiable, tranches et vol.

import (
	"reflect"
	"testing"
)

func pct(v uint8) *uint8 { return &v }

// TestSummarize_TeteEtPairesGardentLesNull : la tête compte toute la passe, S1 garde tueur
// absent, victime bot et source inconnue comme groupes, et la somme des frags vaut les morts.
func TestSummarize_TeteEtPairesGardentLesNull(t *testing.T) {
	rows := []Row{
		{KillerXUID: "k1", VictimXUID: "v1", SourceTag: 7, AssistKnown: true},
		{KillerXUID: "k1", VictimXUID: "v1", SourceTag: 7},
		{KillerXUID: "k1", VictimXUID: "", SourceTag: 7},          // victime bot
		{KillerXUID: "", KillerGamertag: "Bot", VictimXUID: "v1"}, // tueur bot, source inconnue
		{KillerXUID: "k2", VictimXUID: "v1"},                      // source inconnue
	}
	s := Summarize(false, rows)
	want := Head{Publishable: false, Rows: 5, AssistKnown: 1, KillerXUID: 4, SourceAndKiller: 3}
	if s.Head != want {
		t.Fatalf("tête = %+v, attendu %+v", s.Head, want)
	}
	wantPairs := []Pair{
		{PairKey{"", "v1", 0}, 1},
		{PairKey{"k1", "", 7}, 1},
		{PairKey{"k1", "v1", 7}, 2},
		{PairKey{"k2", "v1", 0}, 1},
	}
	if !reflect.DeepEqual(s.Pairs, wantPairs) {
		t.Fatalf("paires = %+v, attendu %+v", s.Pairs, wantPairs)
	}
	total := 0
	for _, p := range s.Pairs {
		total += p.Kills
	}
	if total != len(rows) {
		t.Fatalf("somme des frags %d, attendu %d (une mort = un frag de S1)", total, len(rows))
	}
	if s.Rev != Rev {
		t.Fatalf("Rev = %q, attendu %q", s.Rev, Rev)
	}
}

// TestSummarize_PasseNonPubliableSansAppuis : S2 ne retient que les passes publiables ; S1 et la
// tête restent écrites (les lecteurs de paires et d'armes ne filtrent pas sur publishable).
func TestSummarize_PasseNonPubliableSansAppuis(t *testing.T) {
	rows := []Row{{KillerXUID: "k", VictimXUID: "v", AssistKnown: true, AssistXUID: "a", AssistGamertag: "A"}}
	s := Summarize(false, rows)
	if len(s.Assists) != 0 {
		t.Fatalf("appuis écrits pour une passe non publiable : %+v", s.Assists)
	}
	if len(s.Pairs) != 1 || s.Head.Rows != 1 {
		t.Fatalf("paires %+v tête %+v : S1 et S0 doivent rester écrites", s.Pairs, s.Head)
	}
}

// TestSummarize_AppuisGrainEtComptes : assistant nommé, assistant anonyme (bot), absence
// d'assistant mesurée, assistant inconnu (exclu), et le cas bot-bot qui seul garde la victime.
func TestSummarize_AppuisGrainEtComptes(t *testing.T) {
	rows := []Row{
		// assistant nommé, deux morts, victimes différentes : cumulées (victime hors grain).
		{KillerXUID: "k", KillerGamertag: "K", VictimXUID: "v1", AssistKnown: true,
			AssistXUID: "a", AssistGamertag: "A", KillerDamagePct: pct(60), AssistDamagePct: pct(39)},
		{KillerXUID: "k", KillerGamertag: "K", VictimXUID: "v2", AssistKnown: true,
			AssistXUID: "a", AssistGamertag: "A", KillerDamagePct: pct(40), AssistDamagePct: pct(59)},
		// mesurée sans assistant : groupe à assistant NULL, gardé.
		{KillerXUID: "k", KillerGamertag: "K", VictimXUID: "v1", AssistKnown: true, KillerDamagePct: pct(100)},
		// assistant inconnu : hors S2.
		{KillerXUID: "k", KillerGamertag: "K", VictimXUID: "v1"},
		// assistant bot (sans xuid) d'un tueur nommé : victime hors grain.
		{KillerXUID: "k", KillerGamertag: "K", VictimXUID: "v3", AssistKnown: true, AssistGamertag: "BotA"},
		// bot-bot : assistant ET tueur sans xuid, la victime entre dans le grain.
		{KillerGamertag: "BotK", VictimXUID: "v4", AssistKnown: true, AssistGamertag: "BotA"},
		{KillerGamertag: "BotK", VictimXUID: "v5", AssistKnown: true, AssistGamertag: "BotA"},
	}
	got := Summarize(true, rows).Assists
	want := []AssistPair{
		{AssistKey: AssistKey{KillerXUID: "k", KillerGamertag: "K"}, N: 1},
		{AssistKey: AssistKey{KillerGamertag: "BotK", AssistGamertag: "BotA", VictimXUID: "v4"}, N: 1},
		{AssistKey: AssistKey{KillerGamertag: "BotK", AssistGamertag: "BotA", VictimXUID: "v5"}, N: 1},
		{AssistKey: AssistKey{AssistGamertag: "BotA", KillerXUID: "k", KillerGamertag: "K"}, N: 1},
		{AssistKey: AssistKey{AssistXUID: "a", AssistGamertag: "A", KillerXUID: "k", KillerGamertag: "K"},
			N: 2, Stolen: 1, PctSum: 98, PctCount: 2, High: 1, Mid: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("appuis =\n%+v\nattendu\n%+v", got, want)
	}
}

// TestSummarize_TranchesEtVol : bornes des tranches (< 25, 25..50, > 50, non plafonnée), part
// absente comptée dans N et dans aucune tranche, vol seulement avec deux parts mesurées.
func TestSummarize_TranchesEtVol(t *testing.T) {
	ligne := func(killer, assist *uint8) Row {
		return Row{KillerXUID: "k", AssistKnown: true, AssistXUID: "a", AssistGamertag: "A",
			KillerDamagePct: killer, AssistDamagePct: assist}
	}
	rows := []Row{
		ligne(pct(75), pct(24)), ligne(pct(74), pct(25)), ligne(pct(49), pct(50)),
		ligne(pct(48), pct(51)), ligne(nil, pct(228)), ligne(pct(60), nil),
	}
	got := Summarize(true, rows).Assists
	if len(got) != 1 {
		t.Fatalf("%d groupes, attendu 1", len(got))
	}
	g := got[0]
	want := AssistPair{AssistKey: g.AssistKey, N: 6, Stolen: 2, PctSum: 24 + 25 + 50 + 51 + 228,
		PctCount: 5, Low: 1, Mid: 2, High: 2}
	if g != want {
		t.Fatalf("groupe = %+v, attendu %+v", g, want)
	}
}

// TestSummarize_PasseCreditPuisPasseFilm : la passe crédit (aucun assistant connu, aucune
// source) ne produit aucun appui ; la passe de film du même match, résumée seule, produit les
// siens — deux générations indépendantes, aucune n'hérite de l'autre.
func TestSummarize_PasseCreditPuisPasseFilm(t *testing.T) {
	credit := []Row{{KillerXUID: "k", VictimXUID: "v"}, {KillerXUID: "v", VictimXUID: "k"}}
	film := []Row{
		{KillerXUID: "k", VictimXUID: "v", SourceTag: 9, AssistKnown: true},
		{KillerXUID: "v", VictimXUID: "k", SourceTag: 9, AssistKnown: true, AssistXUID: "a", AssistGamertag: "A"},
	}
	sc, sf := Summarize(true, credit), Summarize(true, film)
	if len(sc.Assists) != 0 || sc.Head.SourceAndKiller != 0 || sc.Head.AssistKnown != 0 {
		t.Fatalf("passe crédit : tête %+v appuis %+v", sc.Head, sc.Assists)
	}
	if len(sf.Assists) != 2 || sf.Head.SourceAndKiller != 2 || sf.Head.AssistKnown != 2 {
		t.Fatalf("passe film : tête %+v appuis %+v", sf.Head, sf.Assists)
	}
}

// TestSummarize_PasseVide : une passe vide rend une tête à zéro et aucune ligne fille.
func TestSummarize_PasseVide(t *testing.T) {
	s := Summarize(true, nil)
	if s.Head != (Head{Publishable: true}) || len(s.Pairs) != 0 || len(s.Assists) != 0 {
		t.Fatalf("passe vide : %+v", s)
	}
}
