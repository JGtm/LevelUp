package timeline

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Empreintes des paramètres de la résolution du coup d'envoi, par révision. Une révision ne change
// jamais d'empreinte : changer un seuil impose de changer KickoffRev ET d'ajouter ici une entrée.
var kickoffParametersByRevision = map[string]string{
	"kickoff-2026-11-18": "3d7470f4241fdd55c2322cb83027cd84f9688308b2bf4743dbdef30101246313",
	// Même empreinte : le coup d'envoi publié par le rejeu est lu sur l'horloge du jeu, ce qui change la valeur
	// rangée, pas un nombre ni l'ordre des sources.
	"kickoff-2026-11-22": "3d7470f4241fdd55c2322cb83027cd84f9688308b2bf4743dbdef30101246313",
	// La fenêtre du match qui accepte un coup d'envoi du rejeu est bornée par la plus grande des deux
	// valeurs « durée du registre » et « plus long temps de jeu » (la durée du registre est parfois
	// fausse) ; un coup d'envoi du rejeu gardé est contrôlé contre cette fenêtre. Le texte haché
	// de la règle change, donc l'empreinte aussi (les empreintes anciennes décrivent l'ancienne règle).
	"kickoff-2026-11-24": "edde8db690b3d5a43be15df117db6ba0ecbb609f58a7251a5b64b77497c1159e",
	// Revue : la durée n'est tenue pour fausse qu'au-delà d'une marge sur le temps de jeu des joueurs humains
	// (DurationMismatchMarginSeconds) ; la marge entre dans le texte haché.
	"kickoff-2026-11-24b": "d04ca51bc7ebafb017e476d0cadade2b63a947da6a404015616547315f76cd54",
}

// kickoffParameters hache les nombres et l'ordre des sources que la résolution applique.
func kickoffParameters() (string, string) {
	lines := []string{
		"t0SpreadThresholdMs=" + strconv.FormatInt(t0SpreadThresholdMs, 10),
		"t0MaxPlausibleMs=" + strconv.FormatInt(t0MaxPlausibleMs, 10),
		"ordre=" + string(KickoffSourceFilm) + ">" + string(KickoffSourceAPI) + ">inconnue",
		"rejeuDansLaFenetre=[0,max(duree,tempsDeJeuMaxDesJoueurs)]",
		"dureeFausseMarge=" + strconv.Itoa(DurationMismatchMarginSeconds),
		"tempsDeJeu=" + string(TimePlayedOK),
	}
	sort.Strings(lines)
	body := strings.Join(lines, "\n")
	h := sha256.Sum256([]byte(body))
	return hex.EncodeToString(h[:]), body
}

// TestKickoffParametersMatchRevision : un seuil de l'estimation ou l'ordre des sources qui change
// sans que KickoffRev change ferait garder aux bases des coups d'envoi calculés autrement que ceux
// des matchs à venir, sans qu'aucun rattrapage ne se relance.
func TestKickoffParametersMatchRevision(t *testing.T) {
	digest, body := kickoffParameters()
	want, known := kickoffParametersByRevision[KickoffRev]
	switch {
	case !known || want == "":
		t.Fatalf("la révision %q n'a pas d'empreinte enregistrée : ajouter %q: %q\nParamètres hachés :\n%s",
			KickoffRev, KickoffRev, digest, body)
	case want != digest:
		t.Fatalf("un paramètre de la résolution a changé sans que KickoffRev change : nouvelle révision, "+
			"puis l'entrée %q: %q.\nParamètres hachés :\n%s", "<nouvelle révision>", digest, body)
	}
}

func i64(v int64) *int64 { return &v }

func TestResolveKickoff_OrdreDesSources(t *testing.T) {
	start := atMs(1_000_000)
	arrivees := []ParticipationT0Input{
		{FirstJoinedTime: atMs(1_020_000), PresentAtBeginning: true},
		{FirstJoinedTime: atMs(1_020_500), PresentAtBeginning: true},
	}
	cas := []struct {
		nom      string
		in       KickoffInput
		source   KickoffSource
		offset   int64
		estimate T0Quality
	}{
		{"le rejeu prime, l estimation n est pas jouee",
			KickoffInput{StartUTC: start, FilmT0Ms: i64(26_304), Participations: arrivees},
			KickoffSourceFilm, 26_304, ""},
		{"rejeu a zero : une valeur, pas une absence",
			KickoffInput{StartUTC: start, FilmT0Ms: i64(0), Participations: arrivees},
			KickoffSourceFilm, 0, ""},
		{"sans rejeu : l estimation acceptee",
			KickoffInput{StartUTC: start, Participations: arrivees},
			KickoffSourceAPI, 20_000, T0QualityOK},
		{"sans rejeu ni arrivee : inconnu",
			KickoffInput{StartUTC: start},
			KickoffSourceUnknown, 0, T0QualityNoData},
		{"estimation negative refusee : inconnu",
			KickoffInput{StartUTC: start, Participations: []ParticipationT0Input{
				{FirstJoinedTime: atMs(990_000), PresentAtBeginning: true}}},
			KickoffSourceUnknown, 0, T0QualityNegative},
		{"estimation au-dela du seuil refusee : inconnu",
			KickoffInput{StartUTC: start, Participations: []ParticipationT0Input{
				{FirstJoinedTime: atMs(1_000_000 + 4*3_600_000), PresentAtBeginning: true}}},
			KickoffSourceUnknown, 0, T0QualitySuspiciousHigh},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			k := ResolveKickoff(c.in)
			if k.Source != c.source || k.Estimate != c.estimate {
				t.Fatalf("source=%q estimation=%q, attendu %q/%q", k.Source, k.Estimate, c.source, c.estimate)
			}
			if k.Known() && k.OffsetMs != c.offset {
				t.Fatalf("decalage=%d, attendu %d", k.OffsetMs, c.offset)
			}
			rs := k.RealStart(start)
			if k.Known() != (rs != nil) {
				t.Fatalf("RealStart=%v pour une source %q", rs, k.Source)
			}
			if rs != nil && !rs.Equal(start.Add(time.Duration(c.offset)*time.Millisecond)) {
				t.Fatalf("RealStart=%v", rs)
			}
		})
	}
}

// Coup d'envoi inconnu : la fenêtre de jeu part du début du match (durée de jeu = durée du match).
func TestGameplayTimeline_ReglesDeLaChronologie(t *testing.T) {
	start := atMs(1_000_000)
	d := 600
	tl, ok := GameplayTimeline(start, &d, Kickoff{})
	if !ok || tl.T0Ms != 0 || tl.GameplayDurationSeconds() != 600 {
		t.Fatalf("inconnu : ok=%v t0=%d duree=%d", ok, tl.T0Ms, tl.GameplayDurationSeconds())
	}
	tl, ok = GameplayTimeline(start, &d, Kickoff{Source: KickoffSourceFilm, OffsetMs: 30_000})
	if !ok || tl.GameplayDurationSeconds() != 570 || !tl.GameplayStartUTC().Equal(atMs(1_030_000)) {
		t.Fatalf("film : ok=%v duree=%d debut=%v", ok, tl.GameplayDurationSeconds(), tl.GameplayStartUTC())
	}
	if _, ok := GameplayTimeline(start, nil, Kickoff{}); ok {
		t.Fatal("duree absente : aucune fenetre attendue")
	}
	zero := 0
	if _, ok := GameplayTimeline(start, &zero, Kickoff{}); ok {
		t.Fatal("duree nulle : aucune fenetre attendue")
	}
}

func TestPlayedSeconds_ValeurRangeeConservee(t *testing.T) {
	start := atMs(1_000_000)
	d := 600
	tl, _ := GameplayTimeline(start, &d, Kickoff{Source: KickoffSourceFilm, OffsetMs: 30_000})
	if s, ok := PlayedSeconds(TimePlayedInput{FirstJoinedTime: atMs(1_010_000)}, tl, true); !ok || s != 570 {
		t.Fatalf("present du debut a la fin : %d/%v, attendu 570", s, ok)
	}
	if _, ok := PlayedSeconds(TimePlayedInput{}, tl, true); ok {
		t.Fatal("presence inconnue : la valeur rangee doit rester")
	}
	if _, ok := PlayedSeconds(TimePlayedInput{FirstJoinedTime: atMs(1_010_000)}, tl, false); ok {
		t.Fatal("match sans fenetre : la valeur rangee doit rester")
	}
	parti := atMs(1_020_000)
	if _, ok := PlayedSeconds(TimePlayedInput{FirstJoinedTime: atMs(1_010_000), LastLeaveTime: &parti}, tl, true); ok {
		t.Fatal("parti avant le coup d envoi : la valeur rangee doit rester")
	}
}

func TestHumanXUID(t *testing.T) {
	for xuid, attendu := range map[string]bool{"2603225103667623": true, "bid(19.0)": false, "": false, "12a": false} {
		if HumanXUID(xuid) != attendu {
			t.Errorf("HumanXUID(%q) != %v", xuid, attendu)
		}
	}
}

// Un coup d'envoi du rejeu hors de la fenêtre du match (négatif, au-delà de la durée) est refusé
// comme un rejeu qui n'en publie pas : l'estimation décide, et le refus est dit.
func TestResolveKickoff_RejeuHorsDeLaFenetre(t *testing.T) {
	start := atMs(1_000_000)
	arrivees := []ParticipationT0Input{{FirstJoinedTime: atMs(1_020_000), PresentAtBeginning: true}}
	d := 600
	cas := []struct {
		nom    string
		t0     int64
		duree  *int
		source KickoffSource
		refuse bool
	}{
		{"negatif", -1, &d, KickoffSourceAPI, true},
		{"au-dela de la duree", 600_001, &d, KickoffSourceAPI, true},
		{"a la duree pile", 600_000, &d, KickoffSourceFilm, false},
		{"duree inconnue : seule la borne zero", 5_000_000, nil, KickoffSourceFilm, false},
		{"duree inconnue et negatif", -5, nil, KickoffSourceAPI, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			k := ResolveKickoff(KickoffInput{StartUTC: start, FilmT0Ms: i64(c.t0), DurationSeconds: c.duree, Participations: arrivees})
			if k.Source != c.source || k.FilmOutOfBounds != c.refuse {
				t.Fatalf("source=%q horsBornes=%v, attendu %q/%v", k.Source, k.FilmOutOfBounds, c.source, c.refuse)
			}
		})
	}
}

// Une durée du registre manifestement fausse (plus courte que le plus long temps de jeu d'un joueur)
// ne fait plus refuser un coup d'envoi juste du rejeu. Les deux cas mesurés le 2026-11-24 : des
// matchs de 7 et 11 minutes rangés à 21 s et 24 s, dont le plus long temps de jeu est 462 s et 677 s.
func TestResolveKickoff_DureeDuRegistreFausse(t *testing.T) {
	start := atMs(1_000_000)
	arrivees := []ParticipationT0Input{{FirstJoinedTime: atMs(1_020_000), PresentAtBeginning: true}}
	p := func(v int) *int { return &v }
	cas := []struct {
		nom      string
		t0       int64
		duree    *int
		plusLong *int
		source   KickoffSource
		refuse   bool
	}{
		{"match de 7 min rangé à 21 s : rejeu juste accepté", 38_000, p(21), p(462), KickoffSourceFilm, false},
		{"match de 11 min rangé à 24 s : rejeu juste accepté", 41_000, p(24), p(677), KickoffSourceFilm, false},
		{"au-delà du temps de jeu : refusé", 700_000, p(24), p(677), KickoffSourceAPI, true},
		{"durée juste, temps de jeu plus court : la durée garde la main", 590_000, p(600), p(400), KickoffSourceFilm, false},
		{"durée juste : au-delà refusé", 601_000, p(600), p(400), KickoffSourceAPI, true},
		{"durée fausse et temps de jeu inconnu : comme avant", 38_000, p(21), nil, KickoffSourceAPI, true},
		{"durée inconnue : seule la borne zéro", 5_000_000, nil, p(300), KickoffSourceFilm, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			k := ResolveKickoff(KickoffInput{StartUTC: start, FilmT0Ms: i64(c.t0), DurationSeconds: c.duree,
				LongestPlayedSeconds: c.plusLong, Participations: arrivees})
			if k.Source != c.source || k.FilmOutOfBounds != c.refuse {
				t.Fatalf("source=%q horsBornes=%v, attendu %q/%v", k.Source, k.FilmOutOfBounds, c.source, c.refuse)
			}
		})
	}
}

func TestDurationUnreliable(t *testing.T) {
	p := func(v int) *int { return &v }
	for nom, c := range map[string]struct {
		duree, plusLong *int
		faux            bool
	}{
		"plus courte que le temps de jeu": {p(21), p(462), true},
		"égale":                           {p(462), p(462), false},
		"une seconde au-dessus":           {p(600), p(601), false},
		"exactement la marge":             {p(900), p(910), false},
		"au-delà de la marge":             {p(900), p(911), true},
		"plus longue":                     {p(600), p(462), false},
		"durée inconnue":                  {nil, p(462), false},
		"durée nulle":                     {p(0), p(462), false},
		"temps de jeu inconnu":            {p(21), nil, false},
	} {
		if got := DurationUnreliable(c.duree, c.plusLong); got != c.faux {
			t.Errorf("%s : DurationUnreliable = %v, attendu %v", nom, got, c.faux)
		}
	}
}
