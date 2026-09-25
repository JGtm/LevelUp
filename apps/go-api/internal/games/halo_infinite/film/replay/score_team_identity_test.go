package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// score_team_identity_test.go — LA PREUVE (a0) : LE MATCH A SENS UNIQUE (lot M5.1, retours rejeu
// du 2026-09-23).
//
// LE GABARIT EST CELUI DE `bf5ced1b` (CTF Illusion, registre 3-0) : le camp qui marque porte une
// serie de score de mode qui finit a 3, le camp qui ne marque jamais n en porte AUCUNE — son
// compteur n a jamais quitte zero, et le statborg n emet un composant qu a son CHANGEMENT. Le slot
// du camp muet existe pourtant (il porte les frags de son camp, composant 2). Avant ce lot, la
// preuve (a) exigeait un score final sur les DEUX slots et le calque sortait `unresolved`. Aucun
// identifiant de match n est ecrit dans la regle : ce fichier la teste sur des enregistrements
// CONSTRUITS.

// teamFragsRec construit une emission du total de frags d un slot d EQUIPE (composant 2, valeur A).
func teamFragsRec(timeMS, slot int, frags int64) types.StatRecord {
	return statRec(timeMS, slot, 0, map[int]types.StatValue{2: {A: frags}})
}

// oneSidedRecords : le slot 8 marque 1, 2, 3 ; le slot 6 ne marque jamais mais porte des frags.
func oneSidedRecords(final ...int64) []types.StatRecord {
	var recs []types.StatRecord
	recs = append(recs, modeRamp(8, 0, 2_000, 1_000, final...)...)
	recs = append(recs, teamFragsRec(2_200, 6, 1), teamFragsRec(3_400, 6, 2))
	recs = append(recs, teamFragsRec(2_300, 8, 1), teamFragsRec(3_300, 8, 2), teamFragsRec(4_300, 8, 3))
	return recs
}

// TestScoreTimelineTeamIdentityOneSided — PREUVE (a0) : registre 3-0, une seule serie qui finit
// EXACTEMENT a 3 -> cette serie est le camp a 3, et le slot muet est l autre camp.
func TestScoreTimelineTeamIdentityOneSided(t *testing.T) {
	for _, cas := range []struct {
		nom        string
		scores     [2]int
		campMarque int
	}{
		{"camp 0 marque (3-0)", [2]int{3, 0}, 0},
		{"camp 1 marque (0-3)", [2]int{0, 3}, 1},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			scores := cas.scores
			tl, cov := buildScoreTimeline(&ScoreInput{Records: oneSidedRecords(1, 2, 3), TeamScores: &scores},
				nil, testClock(), nil)
			if cov.TeamIdentity != ScoreIdentityFinalOneSided {
				t.Fatalf("identite = %q, attendu %q", cov.TeamIdentity, ScoreIdentityFinalOneSided)
			}
			if tl == nil || len(tl.Teams) != 1 {
				t.Fatalf("attendu UNE courbe d equipe (le camp muet n en a pas), obtenu %+v", tl)
			}
			if got := tl.Teams[0].TeamID; got == nil || *got != cas.campMarque {
				t.Fatalf("camp de la serie = %v, attendu %d", got, cas.campMarque)
			}
		})
	}
}

// TestScoreTimelineTeamIdentityOneSidedSingleSlot — le camp muet n a RIEN emis (ni score ni frags) :
// un seul slot d equipe est vu. La regle est la meme — la serie finit exactement au score non nul
// du registre, l autre vaut zero.
func TestScoreTimelineTeamIdentityOneSidedSingleSlot(t *testing.T) {
	scores := [2]int{0, 3}
	recs := modeRamp(6, 0, 2_000, 1_000, 1, 2, 3)
	tl, cov := buildScoreTimeline(&ScoreInput{Records: recs, TeamScores: &scores}, nil, testClock(), nil)
	if cov.TeamIdentity != ScoreIdentityFinalOneSided {
		t.Fatalf("identite = %q, attendu %q", cov.TeamIdentity, ScoreIdentityFinalOneSided)
	}
	if got := tl.Teams[0].TeamID; got == nil || *got != 1 {
		t.Fatalf("camp de la serie = %v, attendu 1", got)
	}
}

// TestScoreTimelineTeamIdentityOneSidedGuards — LES GARDE-FOUS : l egalite est EXACTE, et le
// score absent ne vaut zero que si le registre dit zero.
func TestScoreTimelineTeamIdentityOneSidedGuards(t *testing.T) {
	for _, cas := range []struct {
		nom    string
		scores *[2]int
		final  []int64
	}{
		// Le gabarit d `ab526724` tronque : la 3e capture manque au film, la serie finit a 2.
		{"serie a 2 contre un registre a 3", &[2]int{0, 3}, []int64{1, 2, 2}},
		// Le registre dit que le camp muet a marque : « absent vaut zero » le contredirait.
		{"registre 1-3, une seule serie a 3", &[2]int{1, 3}, []int64{1, 2, 3}},
		{"registre a egalite 0-0", &[2]int{0, 0}, []int64{1, 2, 3}},
		{"sans registre", nil, []int64{1, 2, 3}},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			tl, cov := buildScoreTimeline(&ScoreInput{Records: oneSidedRecords(cas.final...), TeamScores: cas.scores},
				nil, testClock(), nil)
			if cov.TeamIdentity != ScoreIdentityUnresolved {
				t.Fatalf("identite = %q, attendu %q", cov.TeamIdentity, ScoreIdentityUnresolved)
			}
			// La serie EST publiee : le refus porte sur son camp, pas sur son existence.
			if tl == nil || len(tl.Teams) != 1 {
				t.Fatalf("attendu UNE courbe d equipe sans camp, obtenu %+v", tl)
			}
			for i, team := range tl.Teams {
				if team.TeamID != nil {
					t.Errorf("courbe %d : camp %d publie sans preuve", i, *team.TeamID)
				}
			}
		})
	}
}

// TestScoreTimelineTeamIdentityOneSidedBeforeFrags — L ORDRE DES PREUVES EST (a), (a0), (b), et il
// est tenu (revue adverse M5, constat R4, 2026-09-24). Sur un match a sens unique ou la somme des
// frags tranche AUSSI (camps identifies des deux cotes, totaux distincts), c est `a0` qui est
// publiee : elle n emprunte rien au pont d identite des joueurs. La chronique v69 annonce ce
// basculement `b` -> `a0` sur huit documents du parc ; sans ce test, deplacer (a0) apres (b)
// laissait le paquet vert.
func TestScoreTimelineTeamIdentityOneSidedBeforeFrags(t *testing.T) {
	in := fragsIdentityInput()
	var recs []types.StatRecord
	for _, r := range in.Records {
		if _, mode := r.Comps[0]; mode && r.Slot == 6 {
			continue // le slot 6 (camp 0, 5 frags) ne marque jamais : le match est a sens unique
		}
		recs = append(recs, r)
	}
	in.Records = recs
	scores := [2]int{0, 3} // le slot 8 (camp 1, 7 frags) finit a 3
	in.TeamScores = &scores
	// Controle : sans registre de score, (b) SEULE tranche, et dans le meme sens.
	if m := identityByFrags(in.TeamByXUID, []int{6, 8},
		loadScoreSeries(recs, objectives.KillsComponent, true), loadScoreSeries(recs, objectives.KillsComponent, false),
		objectives.SlotIdentityFrom(recs, in.Lines)); m[8] != 1 || m[6] != 0 {
		t.Fatalf("CONTROLE FAUX : la somme des frags ne tranche pas ce gabarit (%v) — le test ne "+
			"prouverait pas l ordre des preuves", m)
	}
	tl, cov := buildScoreTimeline(in, nil, testClock(), nil)
	if cov.TeamIdentity != ScoreIdentityFinalOneSided {
		t.Fatalf("identite = %q, attendu %q (la preuve (a0) passe avant la somme des frags)",
			cov.TeamIdentity, ScoreIdentityFinalOneSided)
	}
	if tl == nil || len(tl.Teams) != 1 || tl.Teams[0].TeamID == nil || *tl.Teams[0].TeamID != 1 {
		t.Fatalf("attendu UNE courbe d equipe au camp 1, obtenu %+v", tl)
	}
}
