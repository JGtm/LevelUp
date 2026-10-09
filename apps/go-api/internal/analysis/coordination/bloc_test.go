package coordination

// bloc_test.go — LE BLOC COORDINATION : l'appui reçu (lot N1, 2026-09-21).
//
// Ce que ces tests cadenassent, et pourquoi chacun compte :
//   - un appui reçu d'un coéquipier NON SUIVI compte (réserve R2), un appui adverse non ;
//   - « on me prépare » se normalise par MES frags, « ma part des appuis » par les appuis DU CAMP ;
//   - un match NON MESURÉ (journal illisible, ou film sans assistance lue) ne fournit ni
//     numérateur ni dénominateur ;
//   - la base de « on me prépare » est la feuille de match, jamais sous les frags lus ;
//   - les appuis impliquant des BOTS (sans xuid) comptent comme les autres ;
//   - un scope sans effectif de camp (FFA) n'a pas de parité — jamais 100 % sur un 1 inventé ;
//   - la parité d'un scope mixte est pondérée par les appuis de camp de chaque match.

import (
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

func entier(v int) *int { return &v }

// scenario — un match mesuré à quatre contre deux, où tout est vérifiable à la main.
func scenario() domain.CoordinationEntree {
	equipes := domain.EquipesParMatch{"m1": {"P": 0, "A": 0, "E1": 1, "E2": 1, "X": 1}}
	return domain.CoordinationEntree{
		MoiXUID: "P",
		Matchs:  []domain.CoordinationMatch{{MatchID: "m1", Mesure: true, TeamSize: entier(4)}},
		Equipes: equipes,
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

func TestBloc_AppuiRecuDeuxDenominateurs(t *testing.T) {
	got := Bloc(scenario())
	if !got.Available || got.MatchesMeasured != 1 || got.MatchesTotal != 1 {
		t.Fatalf("bloc = %+v, attendu disponible sur 1/1 match", got)
	}
	a := got.Appui
	if a.OnMePrepare.Brut != 1 || a.OnMePrepare.N != 3 {
		t.Errorf("on me prépare = %d/%d, attendu 1/3 : le dénominateur est MES frags mesurés, "+
			"les deux frags sans assistant compris", a.OnMePrepare.Brut, a.OnMePrepare.N)
	}
	if a.MaPartDesAppuis.Brut != 1 || a.MaPartDesAppuis.N != 2 {
		t.Errorf("ma part des appuis = %d/%d, attendu 1/2 : l'appui que JE distribue compte au "+
			"dénominateur (ce que le camp a distribué) sans compter au numérateur, et l'appui "+
			"adverse n'entre nulle part", a.MaPartDesAppuis.Brut, a.MaPartDesAppuis.N)
	}
	if a.ParityPct == nil || *a.ParityPct != 25 {
		t.Errorf("parité = %v, attendu 25 (100/4)", a.ParityPct)
	}
}

// TestBloc_MatchNonMesureNEntrePas — un film non décodé n'est pas un match à zéro appui.
//
// Le compter au dénominateur « par match » ferait varier la grandeur avec la COUVERTURE DE
// FILM au lieu du jeu (correction G2) : 20 matchs sur 20 décodés et 2 sur 20 rendraient
// deux valeurs pour exactement le même jeu.
func TestBloc_MatchNonMesureNEntrePas(t *testing.T) {
	in := scenario()
	in.Matchs = append(in.Matchs, domain.CoordinationMatch{MatchID: "m2", Mesure: false})
	in.Equipes["m2"] = map[string]int{"P": 0, "E1": 1}
	in.Appuis = append(in.Appuis,
		domain.CoordinationAppuiRow{MatchID: "m2", AssistXUID: "A", KillerXUID: "P", Nombre: 9})

	got := Bloc(in)

	if got.MatchesMeasured != 1 || got.MatchesTotal != 2 {
		t.Fatalf("couverture = %d/%d, attendu 1/2", got.MatchesMeasured, got.MatchesTotal)
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

	if got.Appui.ParityPct != nil {
		t.Fatalf("parité = %v, attendu nil", got.Appui.ParityPct)
	}
	if len(got.PerMatch) != 1 || got.PerMatch[0].ParityPct != nil || got.PerMatch[0].TeamSize != nil {
		t.Fatalf("case = %+v, attendu sans effectif ni parité", got.PerMatch[0])
	}
}

// TestBloc_PariteMixtePonderee — UN SCOPE QUI MÊLE 4v4 ET BTB N'A PAS UNE PARITÉ UNIQUE.
//
// La moyenne des EFFECTIFS n'est même pas la moyenne des parités : la référence juste est la
// part attendue si les appuis s'étaient répartis également, donc chaque match pèse ce qu'il a
// produit.
func TestBloc_PariteMixtePonderee(t *testing.T) {
	// m1 : 4 joueurs, 2 appuis de camp -> parité 25 %, poids 2.
	// m2 : 8 joueurs, 1 appui de camp  -> parité 12,5 %, poids 1.
	// Attendu : (2*25 + 1*12,5) / 3 = 20,8333… %.
	in := scenario()
	in.Matchs = append(in.Matchs,
		domain.CoordinationMatch{MatchID: "m2", Mesure: true, TeamSize: entier(8)})
	in.Equipes["m2"] = map[string]int{"P": 0, "A": 0, "E1": 1}
	in.Appuis = append(in.Appuis,
		domain.CoordinationAppuiRow{MatchID: "m2", AssistXUID: "A", KillerXUID: "P", Nombre: 1})

	got := Bloc(in).Appui

	if got.ParityPct == nil || math.Abs(*got.ParityPct-62.5/3) > 1e-9 {
		t.Fatalf("parité = %v, attendu 20,83 (pondérée par les appuis de camp de chaque match)",
			got.ParityPct)
	}
}

// TestRestreindre_DecoupeLUniversAvecLesAppuis — la maille SOIRÉE de la frise.
//
// Un match retenu qui ne porte aucun appui reste dans l'univers (il compte au total du scope),
// mais n'est pas mesuré : son film ne porte pas l'assistance.
func TestRestreindre_DecoupeLUniversAvecLesAppuis(t *testing.T) {
	in := scenario()
	in.Matchs = append(in.Matchs,
		domain.CoordinationMatch{MatchID: "muet", Mesure: true, TeamSize: entier(4)})
	in.Matchs = append(in.Matchs,
		domain.CoordinationMatch{MatchID: "autre", Mesure: true, TeamSize: entier(4)})
	in.Appuis = append(in.Appuis,
		domain.CoordinationAppuiRow{MatchID: "autre", AssistXUID: "A", KillerXUID: "P", Nombre: 1})

	got := Restreindre(in, []string{"m1", "muet"})

	if len(got.Matchs) != 2 {
		t.Fatalf("%d matchs, attendu 2 — le match sans appui reste dans l'univers", len(got.Matchs))
	}
	if len(got.Equipes) != 1 || got.Equipes["m1"] == nil {
		t.Errorf("équipes = %+v, attendu la seule table de m1", got.Equipes)
	}
	for _, a := range got.Appuis {
		if a.MatchID != "m1" {
			t.Fatalf("appui hors périmètre : %+v", a)
		}
	}
	if b := Bloc(got); b.MatchesMeasured != 1 || b.MatchesTotal != 2 {
		t.Errorf("couverture = %d/%d, attendu 1/2 : le match muet n'est pas mesuré", b.MatchesMeasured, b.MatchesTotal)
	}
}

// TestBloc_BaseOfficielle — « on me prépare » se rapporte aux frags de la feuille de match :
// un frag sur un bot dont l'assistance n'est pas lue reste dans la base sans entrer au
// numérateur ; une feuille sous les frags lus ne fait pas descendre la base.
func TestBloc_BaseOfficielle(t *testing.T) {
	in := scenario()
	in.FragsOfficiels = map[string]int{"m1": 5}
	if a := Bloc(in).Appui.OnMePrepare; a.Brut != 1 || a.N != 5 {
		t.Errorf("on me prépare = %d/%d, attendu 1/5 (base : la feuille de match)", a.Brut, a.N)
	}
	in.FragsOfficiels = map[string]int{"m1": 1}
	if a := Bloc(in).Appui.OnMePrepare; a.N != 3 {
		t.Errorf("base = %d, attendu 3 : jamais sous les frags lus par le film", a.N)
	}
}

// TestBloc_BotsComptes — un appui impliquant un BOT compte comme les autres. Le bot n'a ni
// xuid ni camp : il se range du côté de l'autre acteur de la ligne, ou de la victime quand les
// deux acteurs sont sans xuid.
func TestBloc_BotsComptes(t *testing.T) {
	in := scenario()
	in.Appuis = []domain.CoordinationAppuiRow{
		// Un bot coéquipier m'assiste : frag appuyé, appui reçu de mon camp.
		{MatchID: "m1", AssistGamertag: "343 Oscar [bot]", KillerXUID: "P", Nombre: 2},
		// J'assiste un bot coéquipier : appui distribué dans mon camp.
		{MatchID: "m1", AssistXUID: "P", Nombre: 1},
		// Un bot assiste un bot sur un adversaire : appui de mon camp (match à deux camps).
		{MatchID: "m1", AssistGamertag: "343 Cosmo [bot]", VictimXUID: "E1", Nombre: 1},
		// Un bot assiste un bot sur un coéquipier : appui adverse, hors de mon camp.
		{MatchID: "m1", AssistGamertag: "343 Ritzy [bot]", VictimXUID: "A", Nombre: 4},
		// Un adversaire assisté par un bot : hors de mon camp.
		{MatchID: "m1", AssistGamertag: "343 Hollis [bot]", KillerXUID: "E1", Nombre: 3},
	}
	a := Bloc(in).Appui
	if a.OnMePrepare.Brut != 2 || a.OnMePrepare.N != 2 {
		t.Errorf("on me prépare = %d/%d, attendu 2/2", a.OnMePrepare.Brut, a.OnMePrepare.N)
	}
	if a.MaPartDesAppuis.Brut != 2 || a.MaPartDesAppuis.N != 4 {
		t.Errorf("ma part des appuis = %d/%d, attendu 2/4", a.MaPartDesAppuis.Brut, a.MaPartDesAppuis.N)
	}
}
