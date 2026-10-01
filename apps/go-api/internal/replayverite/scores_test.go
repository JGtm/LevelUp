package replayverite

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// TestNoter_DocumentJusteNaNiFauxNiManque : le temoin synthetique concorde avec ses faits sur
// chaque oracle, et ne porte aucune violation.
func TestNoter_DocumentJusteNaNiFauxNiManque(t *testing.T) {
	b := Noter(documentJuste(), faitsJustes(), nil)
	exigerScore(t, b, ScoreKills, 4, 0, 0)
	exigerScore(t, b, ScoreMorts, 2, 0, 0)
	exigerScore(t, b, ScoreAssists, 1, 0, 0)
	exigerScore(t, b, ScoreMortsVies, 2, 0, 0)
	exigerScore(t, b, ScoreFinsDeVie, 2, 0, 0)
	exigerScore(t, b, ScoreCamps, 4, 0, 0)
	exigerScore(t, b, ScoreEquipes, 2, 0, 0)
	if _, ok := b.Scores[ScoreMarque]; ok {
		t.Errorf("O-S2 sans drapeau ni bombe doit etre absent, il vaut %+v", b.Scores[ScoreMarque])
	}
	for id, v := range b.Violations {
		if v.Total() != 0 {
			t.Errorf("%s = %v sur un document juste", id, v.Instances)
		}
	}
}

// TestKDA_FauxPositifEtFauxNegatifParJoueur : un kill publie en trop est un FP, un kill manquant un
// FN, et un xuid publie inconnu de la feuille porte tout en FP.
func TestKDA_FauxPositifEtFauxNegatifParJoueur(t *testing.T) {
	d := documentJuste()
	d.ScoreTimeline.Players[0].Kills = serie(Pas{0, 0}, Pas{50, 4}) // 4 publies, 3 officiels
	d.ScoreTimeline.Players[1].Kills = serie()                      // 0 publie, 1 officiel
	d.ScoreTimeline.Players = append(d.ScoreTimeline.Players,
		ScoreJoueur{XUID: "999", Kills: serie(Pas{10, 2})})
	b := Noter(d, faitsJustes(), nil)
	exigerScore(t, b, ScoreKills, 3, 1+2, 1)
	if e := b.Scores[ScoreKills].Ecarts["111"]; e != (Ecart{Pub: 4, Off: 3}) {
		t.Errorf("ecart 111 = %+v, veut publie 4 / officiel 3", e)
	}
}

// TestKDA_SlotNommeParLeTripletEstExclu : un joueur dont tous les slots statborg ont ete nommes par
// `triplet_feuille` a ete apparie SUR l'oracle — il n'est pas note, il est compte en Exclus.
func TestKDA_SlotNommeParLeTripletEstExclu(t *testing.T) {
	d := documentJuste()
	d.Identity.StatborgSlots[0].Link.Method = methodeCirculaire
	d.ScoreTimeline.Players[0].Kills = serie(Pas{0, 0}, Pas{50, 9}) // faux, mais circulaire
	b := Noter(d, faitsJustes(), nil)
	exigerScore(t, b, ScoreKills, 1, 0, 0)
	if b.Scores[ScoreKills].Exclus != 1 {
		t.Errorf("Exclus = %d, veut 1", b.Scores[ScoreKills].Exclus)
	}
	// Un second slot du meme joueur nomme AUTREMENT rend le joueur notable.
	d.Identity.StatborgSlots = append(d.Identity.StatborgSlots,
		SlotStatborg{Slot: 14, XUID: "111", Link: Lien{Method: "instants_de_mort"}})
	exigerScore(t, Noter(d, faitsJustes(), nil), ScoreKills, 4, 6, 0)
}

// TestVies_CoupureSansMortEtVieManquante : une vie en trop est un FP de O-V1 et de O-V2, une vie
// manquante un FN.
func TestVies_CoupureSansMortEtVieManquante(t *testing.T) {
	d := documentJuste()
	d.Tracks = append(d.Tracks, piste(516, "111", 120, 140)) // coupure : 3 vies pour 1 mort
	b := Noter(d, faitsJustes(), nil)
	exigerScore(t, b, ScoreMortsVies, 2, 1, 0)
	exigerScore(t, b, ScoreFinsDeVie, 2, 1, 0)

	d = documentJuste()
	d.Tracks = d.Tracks[:3] // 222 n'a plus qu'une vie
	b = Noter(d, faitsJustes(), nil)
	exigerScore(t, b, ScoreMortsVies, 1, 0, 1)
}

// TestCamps_CirculaireNonNoteEtMethodeBNotee : `a` / `a0` rattachent les camps par le score final,
// O-S1 n'est alors pas note ; `b` (somme des frags) est note.
func TestCamps_CirculaireNonNoteEtMethodeBNotee(t *testing.T) {
	for _, m := range []string{"a", "a0"} {
		d := documentJuste()
		d.Coverage.Score.TeamIdentity = m
		s := Noter(d, faitsJustes(), nil).Scores[ScoreCamps]
		if !s.NonNote || s.Exclus != 2 {
			t.Errorf("teamIdentity %q : %+v, veut non note avec 2 exclus", m, s)
		}
	}
	d := documentJuste()
	d.ScoreTimeline.Teams[1].Total = nil // le camp 1 marque 1 officiellement, 0 publie
	exigerScore(t, Noter(d, faitsJustes(), nil), ScoreCamps, 3, 0, 1)
}

// TestMarque_CapturesEtDetonationsParCamp : la famille se lit aux calques ; les captures du camp 0
// et les detonations du camp 1 se comparent au score officiel.
func TestMarque_CapturesEtDetonationsParCamp(t *testing.T) {
	d := documentJuste()
	d.FlagCarries = []Drapeau{{}}
	d.Objectives = []ActionObjectif{{T: 10, XUID: "111", Stat: statCaptureDrapeau}, {T: 20, XUID: "111", Stat: "flag_grabs"}}
	exigerScore(t, Noter(d, faitsJustes(), nil), ScoreMarque, 1, 0, 3) // 1 capture contre 3, 0 contre 1

	d = documentJuste()
	d.BombStats = &StatsBombe{Players: []StatsBombeJoueur{{XUID: "222", Detonations: ptr(2)}, {XUID: "111"}}}
	exigerScore(t, Noter(d, faitsJustes(), nil), ScoreMarque, 1, 1, 3)
}

// TestEquipes_ContradictionEstUnFaux : O-T1 lit accord / contradiction / silence.
func TestEquipes_ContradictionEstUnFaux(t *testing.T) {
	d := documentJuste()
	d.Coverage.Teams = &CouvEquipes{Accord: 1, Contradiction: 1, Silence: 2}
	exigerScore(t, Noter(d, faitsJustes(), nil), ScoreEquipes, 1, 1, 2)
}

// TestScorePersonnel_ContreLOracleOfficiel : O-S3 compare le score statborg final a
// `personal_score` (repli sur `score`), ignore un joueur sans colonne, et n'existe pas sans oracle.
func TestScorePersonnel_ContreLOracleOfficiel(t *testing.T) {
	d := documentJuste()
	d.ScoreTimeline.Players[0].Score = serie(Pas{0, 0}, Pas{50, 350})
	d.ScoreTimeline.Players[1].Score = serie(Pas{0, 0}, Pas{190, 100})
	oracle := &domain.MatchOracle{Players: []domain.MatchPlayerOracle{
		{XUID: "111", PersonalScore: ptr(300)},
		{XUID: "222", Score: ptr(150)}, // personal_score NULL : repli sur score
		{XUID: "333"},                  // aucune colonne : ni vrai ni faux
	}}
	exigerScore(t, Noter(d, faitsJustes(), oracle), ScorePersonnel, 300+100, 50, 50)
	if _, ok := Noter(d, faitsJustes(), nil).Scores[ScorePersonnel]; ok {
		t.Error("O-S3 sans oracle doit etre absent du bulletin")
	}
}
