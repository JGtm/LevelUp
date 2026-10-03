package service

// match_view_builders_riposte_test.go — LE BLOC RIPOSTE DE LA VUE MATCH (lot N1, D22-2).
//
// Ce que ces tests cadenassent :
//   - une mort vengée DANS la fenêtre nomme son vengeur et son délai ; hors fenêtre, non ;
//   - les deux comptes par joueur ne s'additionnent pas (subi à gauche, porté à droite) ;
//   - tout le scoreboard sort, y compris les joueurs à zéro : un absent changerait la
//     hauteur apparente des autres ;
//   - aucune ligne de journal ⇒ pas de bloc, jamais une section vide.

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func camp(v int) *int { return &v }

// matchDeTest — deux camps de deux, quatre morts :
//
//	1 s    E1 tue A         vengée par P a 3 s (2 s)
//	10 s   E2 tue P         « vengée » par A a 20 s : HORS fenêtre
//	30 s   P tue E1         vengée par E2 a 31 s (1 s)
func matchDeTest() ([]domain.KVPairRaw, []domain.ScoreboardRaw) {
	kv := []domain.KVPairRaw{
		{KillerXUID: "E1", VictimXUID: "A", VictimGT: "Kaya", KillCount: 1, TimeMS: 1000},
		{KillerXUID: "P", VictimXUID: "E1", VictimGT: "Vex", KillCount: 1, TimeMS: 3000},
		{KillerXUID: "E2", VictimXUID: "P", VictimGT: "JGtm", KillCount: 1, TimeMS: 10000},
		{KillerXUID: "A", VictimXUID: "E2", VictimGT: "Otto", KillCount: 1, TimeMS: 20000},
		{KillerXUID: "P", VictimXUID: "E1", VictimGT: "Vex", KillCount: 1, TimeMS: 30000},
		{KillerXUID: "E2", VictimXUID: "P", VictimGT: "JGtm", KillCount: 1, TimeMS: 31000},
	}
	sb := []domain.ScoreboardRaw{
		{XUID: "P", Gamertag: "JGtm", TeamID: camp(0)},
		{XUID: "A", Gamertag: "Kaya", TeamID: camp(0)},
		{XUID: "E1", Gamertag: "Vex", TeamID: camp(1)},
		{XUID: "E2", Gamertag: "Otto", TeamID: camp(1)},
	}
	return kv, sb
}

func TestBuildMatchRiposte_FenetreEtNoms(t *testing.T) {
	got := buildMatchRiposte(matchDeTest())

	if got == nil {
		t.Fatal("bloc nil alors que le match porte six morts")
	}
	if got.FenetreMs != 5000 || got.MeasuredDeaths != 6 {
		t.Fatalf("fenêtre = %d, morts = %d, attendu 5000 et 6", got.FenetreMs, got.MeasuredDeaths)
	}
	if len(got.Deaths) != 6 {
		t.Fatalf("%d morts publiées, attendu 6 — toutes, vengées ou non", len(got.Deaths))
	}
	// La mort de A a 1 s : vengée par P a 3 s.
	premiere := got.Deaths[0]
	if premiere.VictimXUID != "A" || !premiere.Avenged || premiere.AvengerXUID != "P" {
		t.Fatalf("première mort = %+v, attendu A vengée par P", premiere)
	}
	if premiere.DelaiMs == nil || *premiere.DelaiMs != 2000 {
		t.Errorf("délai = %v, attendu 2000", premiere.DelaiMs)
	}
	if premiere.AvengerGamertag != "JGtm" || premiere.VictimGamertag != "Kaya" {
		t.Errorf("noms = %q / %q, attendus depuis le scoreboard",
			premiere.VictimGamertag, premiere.AvengerGamertag)
	}
	if premiere.VictimTeamID == nil || *premiere.VictimTeamID != 0 {
		t.Errorf("camp de la victime = %v, attendu 0", premiere.VictimTeamID)
	}
	// La mort de P a 10 s : la riposte de A arrive a 20 s, HORS fenêtre.
	for _, d := range got.Deaths {
		if d.VictimXUID == "P" && d.TimeMs == 10000 {
			if d.Avenged || d.DelaiMs != nil {
				t.Errorf("mort a 10 s = %+v : la riposte a 10 s d'écart n'est pas un échange", d)
			}
		}
	}
}

// TestBuildMatchRiposte_LesDeuxComptesNeSAdditionnentPas — à gauche un événement SUBI, à
// droite un événement PORTÉ. Un solde effacerait la différence entre « beaucoup vengé,
// venge peu » et « ni l'un ni l'autre ».
func TestBuildMatchRiposte_ComptesParJoueur(t *testing.T) {
	got := buildMatchRiposte(matchDeTest())

	attendus := map[string][2]int{
		// A est vengé une fois (a 1 s), ne riposte jamais dans la fenêtre.
		"A": {1, 0},
		// P n'est jamais vengé dans la fenêtre, et riposte une fois.
		"P": {0, 1},
		// E1 est vengé une fois (mort a 30 s, vengée par E2 a 31 s).
		"E1": {1, 0},
		// E2 porte cette riposte.
		"E2": {0, 1},
	}
	if len(got.Players) != 4 {
		t.Fatalf("%d joueurs, attendu les 4 du scoreboard — un absent changerait la hauteur "+
			"apparente des autres", len(got.Players))
	}
	for _, p := range got.Players {
		want := attendus[p.XUID]
		if p.DeathsAvenged != want[0] || p.Ripostes != want[1] {
			t.Errorf("%s = %d vengé / %d riposté, attendu %d / %d",
				p.XUID, p.DeathsAvenged, p.Ripostes, want[0], want[1])
		}
	}
	if got.Players[0].TeamID == nil || *got.Players[0].TeamID != 0 {
		t.Errorf("les joueurs sortent groupés par camp : %+v", got.Players)
	}
}

// TestBuildMatchRiposte_SansJournal — sans ordre des morts il n'y a rien à dire, et
// publier un bloc vide forcerait l'écran à choisir un message là où il n'y a pas de sujet.
func TestBuildMatchRiposte_SansJournal(t *testing.T) {
	_, sb := matchDeTest()
	if got := buildMatchRiposte(nil, sb); got != nil {
		t.Fatalf("bloc = %+v, attendu nil sans aucune ligne de journal", got)
	}
}

// TestBuildMatchRiposte_SansCamp_AucunEchange — un scoreboard sans `team_id` (FFA, lignes
// incomplètes) ne permet de conclure aucune vengeance : deviner un camp en fabriquerait.
func TestBuildMatchRiposte_SansCamp_AucunEchange(t *testing.T) {
	kv, sb := matchDeTest()
	for i := range sb {
		sb[i].TeamID = nil
	}

	got := buildMatchRiposte(kv, sb)

	for _, d := range got.Deaths {
		if d.Vengeable || d.Avenged {
			t.Fatalf("mort %+v déclarée vengeable sans camps connus", d)
		}
	}
}
