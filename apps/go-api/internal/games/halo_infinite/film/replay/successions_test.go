package replay

import "testing"

// successions_test.go — la fermeture par relais : chaîne à candidat unique, jamais une
// devinette. Axe : origin=0, step=100 ms -> frame N = N*100 ms de film ; offset match->film
// = 0 pour lire les fenêtres directement.

func succTrack(slot uint32, start, end int, xuid, bot string) Track {
	return Track{Slot: slot, XUID: xuid, Bot: bot, StartFrame: start, EndFrame: end}
}

func TestAttributeSuccessionsChainsUniqueCandidates(t *testing.T) {
	// Arrivée à 60 s : vie 1 anonyme naît à 61 s (fenêtre première vie), meurt à 90 s ;
	// vie 2 anonyme naît à 100 s (90+10 s, fenêtre de réapparition). Une vie nommée et une
	// vie anonyme TARDIVE (200 s, hors chaîne) ne bougent pas.
	tracks := []Track{
		succTrack(500, 0, 850, "111", ""),
		succTrack(600, 610, 900, "", ""),
		succTrack(601, 1000, 1500, "", ""),
		succTrack(602, 2000, 2100, "", ""), // caméra de fin : hors fenêtres, reste anonyme
	}
	attributeSuccessions(tracks, []Succession{{BotName: "343 Razzle [bot]", SwitchMatchMS: 60_000}},
		0, 100_000, 0, 10, nil)
	if tracks[1].Bot != "343 Razzle [bot]" || tracks[2].Bot != "343 Razzle [bot]" {
		t.Errorf("la chaîne doit nommer les deux vies du relais : %+v", tracks)
	}
	if tracks[0].Bot != "" || tracks[3].Bot != "" {
		t.Errorf("vie nommée et vie hors chaîne doivent rester intactes : %+v", tracks)
	}
}

func TestAttributeSuccessionsStopsOnContest(t *testing.T) {
	// DEUX vies anonymes naissent dans la fenêtre d'arrivée : on ne tranche pas.
	tracks := []Track{
		succTrack(600, 610, 900, "", ""),
		succTrack(601, 620, 910, "", ""),
	}
	attributeSuccessions(tracks, []Succession{{BotName: "343 Razzle [bot]", SwitchMatchMS: 60_000}},
		0, 100_000, 0, 10, nil)
	if tracks[0].Bot != "" || tracks[1].Bot != "" {
		t.Errorf("deux candidates = contesté, aucune attribution : %+v", tracks)
	}
}

func TestAttributeSuccessionsNeedsClockBridge(t *testing.T) {
	// Sans calage morts->film (0 apparié), les deux horloges ne se parlent pas : rien.
	tracks := []Track{succTrack(600, 610, 900, "", "")}
	attributeSuccessions(tracks, []Succession{{BotName: "343 Razzle [bot]", SwitchMatchMS: 60_000}},
		0, 100_000, 0, 0, nil)
	if tracks[0].Bot != "" {
		t.Errorf("sans offset apparié, aucune attribution : %+v", tracks)
	}
}

func TestAttributeSuccessionsLiftsContestByFire(t *testing.T) {
	// DEUX remplaçants simultanés : deux vies anonymes naissent dans la même fenêtre. Le
	// TIR départage — chaque vie contient un tir de l'indice de SON remplaçant, lu dans le
	// film, jamais deviné. Un seul des deux tire : sa vie est levée, l'autre chaîne
	// s'arrête contestée.
	tracks := []Track{
		succTrack(600, 610, 900, "", ""),
		succTrack(601, 620, 910, "", ""),
	}
	fire := []FireEventRef{
		// Les deux vies se chevauchent (61-90 s et 62-91 s) : seuls les tirs tombes dans
		// la partie NON PARTAGEE d une vie votent.
		{FilmIndex: 8, TimestampUS: 61_500_000}, // avant la naissance du slot 601 -> vote 600
		{FilmIndex: 9, TimestampUS: 90_500_000}, // apres la mort du slot 600 -> vote 601
	}
	attributeSuccessions(tracks, []Succession{
		{BotName: "343 A [bot]", FilmIndex: 8, SwitchMatchMS: 60_000},
		{BotName: "343 B [bot]", FilmIndex: 9, SwitchMatchMS: 60_000},
	}, 0, 100_000, 0, 10, fire)
	if tracks[0].Bot != "343 A [bot]" || tracks[1].Bot != "343 B [bot]" {
		t.Errorf("les tirs indexés doivent départager les deux relais : %+v", tracks)
	}
}

func TestAttributeSuccessionsFireCannotLieAcrossTwoCandidates(t *testing.T) {
	// L'indice du remplaçant tire dans LES DEUX candidates (fenêtres qui se chevauchent,
	// tir posthume mal daté…) : la corroboration ne tranche pas, la chaîne s'arrête.
	tracks := []Track{
		succTrack(600, 610, 900, "", ""),
		succTrack(601, 620, 910, "", ""),
	}
	fire := []FireEventRef{
		{FilmIndex: 8, TimestampUS: 61_500_000}, // vote 600
		{FilmIndex: 8, TimestampUS: 90_500_000}, // vote 601 — deux votes opposes du MEME indice
	}
	attributeSuccessions(tracks, []Succession{
		{BotName: "343 A [bot]", FilmIndex: 8, SwitchMatchMS: 60_000},
	}, 0, 100_000, 0, 10, fire)
	if tracks[0].Bot != "" || tracks[1].Bot != "" {
		t.Errorf("deux candidates tirées = contesté, aucune attribution : %+v", tracks)
	}
}
