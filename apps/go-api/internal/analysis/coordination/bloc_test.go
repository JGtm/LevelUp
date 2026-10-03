package coordination

// bloc_test.go — LE BLOC COORDINATION (lot N1, 2026-09-21).
//
// Ce que ces tests cadenassent, et pourquoi chacun compte :
//   - une mort ripostée DANS la fenêtre compte, une riposte HORS fenêtre ne compte pas
//     (c'est la borne de `Echanges`, et le bloc ne la redéfinit pas) ;
//   - « je riposte » se normalise par les morts DE MON CAMP, jamais par les miennes ;
//   - un appui reçu d'un coéquipier NON SUIVI compte (réserve R2), un appui adverse non ;
//   - un match NON MESURÉ ne fournit ni numérateur ni dénominateur ;
//   - un scope sans effectif de camp (FFA) n'a pas de parité — jamais 100 % sur un 1 inventé.

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func entier(v int) *int { return &v }

// scenario — un match mesuré à quatre contre deux, où tout est vérifiable à la main :
//
//	t=1 s    E1 tue A          mort de camp, vengée par MOI a t=3 s (2 s, DANS la fenêtre)
//	t=10 s   E2 tue P (moi)    ma mort, « vengée » par A a t=20 s (10 s, HORS fenêtre)
//	t=30 s   E1 tue P (moi)    ma mort, vengée par A a t=31 s (1 s, DANS la fenêtre)
//	t=40 s   E2 tue X          X est adverse : la mort n'est pas de mon camp
func scenario() domain.CoordinationEntree {
	equipes := domain.EquipesParMatch{"m1": {"P": 0, "A": 0, "E1": 1, "E2": 1, "X": 1}}
	return domain.CoordinationEntree{
		MoiXUID: "P",
		Matchs:  []domain.CoordinationMatch{{MatchID: "m1", Mesure: true, TeamSize: entier(4)}},
		Equipes: equipes,
		Kills: []domain.KillEvent{
			{MatchID: "m1", KillerXUID: "E1", VictimXUID: "A", TimeMs: 1000},
			{MatchID: "m1", KillerXUID: "P", VictimXUID: "E1", TimeMs: 3000},
			{MatchID: "m1", KillerXUID: "E2", VictimXUID: "P", TimeMs: 10000},
			{MatchID: "m1", KillerXUID: "A", VictimXUID: "E2", TimeMs: 20000},
			{MatchID: "m1", KillerXUID: "E1", VictimXUID: "P", TimeMs: 30000},
			{MatchID: "m1", KillerXUID: "A", VictimXUID: "E1", TimeMs: 31000},
			{MatchID: "m1", KillerXUID: "E2", VictimXUID: "X", TimeMs: 40000},
		},
		Appuis: []domain.CoordinationAppuiRow{
			// A m'a préparé un frag : A n'est PAS un joueur suivi du produit, et cela ne
			// change rien (réserve R2 — le film nomme tout le monde).
			{MatchID: "m1", AssistXUID: "A", KillerXUID: "P", Nombre: 1},
			// Deux frags MESURÉS sans assistant : un fait, pas une absence de ligne.
			{MatchID: "m1", AssistXUID: "", KillerXUID: "P", Nombre: 2},
			// Un appui que JE distribue : il compte à ce que le camp a distribué, pas à
			// ce que j'ai reçu.
			{MatchID: "m1", AssistXUID: "P", KillerXUID: "A", Nombre: 1},
			// Appui adverse : hors de mon camp des deux côtés.
			{MatchID: "m1", AssistXUID: "E1", KillerXUID: "E2", Nombre: 3},
		},
	}
}

func TestBloc_RiposteFenetreEtDenominateurDeCamp(t *testing.T) {
	got := Bloc(scenario())

	if !got.Available || got.MatchesMeasured != 1 || got.MatchesTotal != 1 {
		t.Fatalf("bloc = %+v, attendu disponible sur 1/1 match", got)
	}
	r := got.Riposte
	if r.TeamDeaths != 3 {
		t.Errorf("morts de camp = %d, attendu 3 (la mort de X est adverse)", r.TeamDeaths)
	}
	if r.TeamDeathsAvenged != 2 {
		t.Errorf("morts ripostées = %d, attendu 2 : la riposte a 10 s est HORS fenêtre",
			r.TeamDeathsAvenged)
	}
	if r.JeSuisCouvert.Brut != 1 || r.JeSuisCouvert.N != 2 {
		t.Errorf("je suis couvert = %d/%d, attendu 1/2 (MES morts, pas celles du camp)",
			r.JeSuisCouvert.Brut, r.JeSuisCouvert.N)
	}
	if r.JeRiposte.Brut != 1 || r.JeRiposte.N != 3 {
		t.Errorf("je riposte = %d/%d, attendu 1/3 — le dénominateur est les morts DU CAMP : "+
			"le normaliser sur mes morts gonflerait ma part dès que le camp meurt peu",
			r.JeRiposte.Brut, r.JeRiposte.N)
	}
	if r.DelaiMedianMs == nil || *r.DelaiMedianMs != 1500 {
		t.Errorf("délai médian = %v, attendu 1500 (moyenne de 1 000 et 2 000)", r.DelaiMedianMs)
	}
	if r.ParityPct == nil || *r.ParityPct != 25 {
		t.Errorf("parité = %v, attendu 25 (100/4)", r.ParityPct)
	}
	if !r.JeSuisCouvert.EchantillonFaible {
		t.Error("deux morts : l'échantillon faible doit être posé")
	}
}

func TestBloc_AppuiRecuDeuxDenominateurs(t *testing.T) {
	a := Bloc(scenario()).Appui

	if a.OnMePrepare.Brut != 1 || a.OnMePrepare.N != 3 {
		t.Errorf("on me prépare = %d/%d, attendu 1/3 : le dénominateur est MES frags mesurés, "+
			"les deux frags sans assistant compris", a.OnMePrepare.Brut, a.OnMePrepare.N)
	}
	if a.MaPartDesAppuis.Brut != 1 || a.MaPartDesAppuis.N != 2 {
		t.Errorf("ma part des appuis = %d/%d, attendu 1/2 : l'appui que JE distribue compte au "+
			"dénominateur (ce que le camp a distribué) sans compter au numérateur, et l'appui "+
			"adverse n'entre nulle part", a.MaPartDesAppuis.Brut, a.MaPartDesAppuis.N)
	}
}

// TestBloc_MatchNonMesureNEntrePas — un film non décodé n'est pas un match à zéro riposte.
//
// Le compter au dénominateur « par match » ferait varier la grandeur avec la COUVERTURE DE
// FILM au lieu du jeu (correction G2) : 20 matchs sur 20 décodés et 2 sur 20 rendraient
// deux valeurs pour exactement le même jeu.
func TestBloc_MatchNonMesureNEntrePas(t *testing.T) {
	in := scenario()
	in.Matchs = append(in.Matchs, domain.CoordinationMatch{MatchID: "m2", Mesure: false})
	in.Kills = append(in.Kills,
		domain.KillEvent{MatchID: "m2", KillerXUID: "E1", VictimXUID: "P", TimeMs: 1000})
	in.Equipes["m2"] = map[string]int{"P": 0, "E1": 1}
	in.Appuis = append(in.Appuis,
		domain.CoordinationAppuiRow{MatchID: "m2", AssistXUID: "A", KillerXUID: "P", Nombre: 9})

	got := Bloc(in)

	if got.MatchesMeasured != 1 || got.MatchesTotal != 2 {
		t.Fatalf("couverture = %d/%d, attendu 1/2", got.MatchesMeasured, got.MatchesTotal)
	}
	if got.Riposte.JeSuisCouvert.N != 2 {
		t.Errorf("je suis couvert : N = %d, attendu 2 — la mort du match non mesuré n'entre pas",
			got.Riposte.JeSuisCouvert.N)
	}
	if got.Appui.OnMePrepare.N != 3 {
		t.Errorf("on me prépare : N = %d, attendu 3 — les 9 frags du match non mesuré n'entrent pas",
			got.Appui.OnMePrepare.N)
	}
	if len(got.PerMatch) != 1 || got.PerMatch[0].MatchID != "m1" {
		t.Errorf("cases = %+v, attendu la seule case de m1", got.PerMatch)
	}
}

// TestBloc_SansMatchMesure_IndisponibleAvecRaison — aucun bloc de zéros : « aucune donnée »
// n'est pas « zéro pour cent ».
func TestBloc_SansMatchMesure_IndisponibleAvecRaison(t *testing.T) {
	got := Bloc(domain.CoordinationEntree{
		MoiXUID: "P",
		Matchs:  []domain.CoordinationMatch{{MatchID: "m1", Mesure: false}},
	})
	if got.Available {
		t.Fatal("Available = true sans aucun match mesuré")
	}
	if got.UnavailableReason != domain.CoordinationNoMeasuredMatch {
		t.Errorf("raison = %q, attendu %q", got.UnavailableReason, domain.CoordinationNoMeasuredMatch)
	}
}

// TestBloc_FFA_AucuneParite — camp inconnu = pas de parité (réserve R1).
//
// Un effectif de 1 par défaut donnerait une parité de 100 % : le joueur serait « sous son
// tour » sur tous les matchs sans camp, quoi qu'il fasse.
func TestBloc_FFA_AucuneParite(t *testing.T) {
	in := scenario()
	in.Matchs = []domain.CoordinationMatch{{MatchID: "m1", Mesure: true}}

	got := Bloc(in)

	if got.Riposte.ParityPct != nil || got.Appui.ParityPct != nil {
		t.Fatalf("parités = %v et %v, attendu nil et nil", got.Riposte.ParityPct, got.Appui.ParityPct)
	}
	if len(got.PerMatch) != 1 || got.PerMatch[0].ParityPct != nil || got.PerMatch[0].TeamSize != nil {
		t.Fatalf("case = %+v, attendu sans effectif ni parité", got.PerMatch[0])
	}
}

// TestBloc_PariteMixtePonderee — UN SCOPE QUI MÊLE 4v4 ET BTB N'A PAS UNE PARITÉ UNIQUE.
//
// La moyenne des EFFECTIFS (100/6 = 16,7 % ici) n'est même pas la moyenne des parités : la
// référence juste est la part attendue si les événements s'étaient répartis également,
// donc chaque match pèse ce qu'il a produit.
func TestBloc_PariteMixtePonderee(t *testing.T) {
	// m1 : 4 joueurs, 3 morts de camp -> parité 25 %, poids 3.
	// m2 : 8 joueurs, 1 mort de camp  -> parité 12,5 %, poids 1.
	// Attendu : (3*25 + 1*12,5) / 4 = 21,875 %.
	in := scenario()
	in.Matchs = append(in.Matchs,
		domain.CoordinationMatch{MatchID: "m2", Mesure: true, TeamSize: entier(8)})
	in.Equipes["m2"] = map[string]int{"P": 0, "E1": 1}
	in.Kills = append(in.Kills,
		domain.KillEvent{MatchID: "m2", KillerXUID: "E1", VictimXUID: "P", TimeMs: 1000})

	got := Bloc(in).Riposte

	if got.ParityPct == nil || *got.ParityPct != 21.875 {
		t.Fatalf("parité = %v, attendu 21,875 (pondérée par les morts de camp de chaque match)",
			got.ParityPct)
	}
}

// TestRestreindre_DecoupeLUniversAvecLesEvenements — la maille SOIRÉE de la frise.
//
// Un match retenu qui ne porte aucune mort doit rester dans l'univers : il compte au
// dénominateur « par match », et le déduire des événements l'effacerait.
func TestRestreindre_DecoupeLUniversAvecLesEvenements(t *testing.T) {
	in := scenario()
	in.Matchs = append(in.Matchs,
		domain.CoordinationMatch{MatchID: "muet", Mesure: true, TeamSize: entier(4)})
	in.Matchs = append(in.Matchs,
		domain.CoordinationMatch{MatchID: "autre", Mesure: true, TeamSize: entier(4)})

	got := Restreindre(in, []string{"m1", "muet"})

	if len(got.Matchs) != 2 {
		t.Fatalf("%d matchs, attendu 2 — le match sans mort reste dans l'univers", len(got.Matchs))
	}
	if len(got.Equipes) != 1 || got.Equipes["m1"] == nil {
		t.Errorf("équipes = %+v, attendu la seule table de m1", got.Equipes)
	}
	for _, e := range got.Kills {
		if e.MatchID != "m1" {
			t.Fatalf("événement hors périmètre : %+v", e)
		}
	}
	if b := Bloc(got); b.MatchesMeasured != 2 || b.MatchesTotal != 2 {
		t.Errorf("couverture = %d/%d, attendu 2/2", b.MatchesMeasured, b.MatchesTotal)
	}
}
