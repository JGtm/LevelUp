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
}

// kickoffParameters hache les nombres et l'ordre des sources que la résolution applique.
func kickoffParameters() (string, string) {
	lines := []string{
		"t0SpreadThresholdMs=" + strconv.FormatInt(t0SpreadThresholdMs, 10),
		"t0MaxPlausibleMs=" + strconv.FormatInt(t0MaxPlausibleMs, 10),
		"ordre=" + string(KickoffSourceFilm) + ">" + string(KickoffSourceAPI) + ">inconnue",
		"rejeuDansLaFenetre=[0,duree]",
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
