package replay

// ability_impulses_film_test.go — LE TEST D'ACCEPTATION DU LOT P3, SUR PIÈCES ET CONTRE UNE
// VÉRITÉ TERRAIN.
//
// CE QU'IL CONTRÔLE, ET POURQUOI IL EXISTE. Le rapport R8 établit que le canal d'impulsion
// mesure l'usage du propulseur, mais il le dit lui-même : le RAPPEL n'y est pas établi (par.
// 10.1, report n°2) — aucun modèle ne peut le donner, seule une observation le peut. Le
// 2026-09-03, l'utilisateur a relevé au visionneur Theater du jeu, sur le film `1cd3848a`,
// CINQ usages de propulseur : 1:51, 1:54, 2:03, 2:05, 2:14 (5 charges). Ce test rejoue la
// CHAÎNE DE PRODUCTION ENTIÈRE (BuildFromFilm, catalogue réel du titre) et vérifie qu'elle en
// rend cinq, aux mêmes instants à une seconde près.
//
// LE RELEVÉ COUVRE UNE FENÊTRE, PAS TOUT LE MATCH, et le contrôle le dit au lieu de faire
// comme si. La fiche soumise à l'utilisateur (CRENEAUX_VERIFICATION_EQUIPEMENT_2026-09-03,
// créneaux 1 et 3) lui demandait de regarder LA SALVE « 1:52, 1:55, 2:03, 2:05, 2:15 — les
// cinq valent le coup d'oeil d'affilée » ; il en a confirmé cinq. Le contrôle porte donc sur
// l'intervalle de la salve, où il vaut précision ET rappel. HORS de cet intervalle, la chaîne
// rend d'autres impulsions de JGtm sur ce film (la mesure du lot précédent en compte 14 pour
// lui, tous rangs lus confondus) : elles ne sont ni confirmées ni infirmées par ce relevé —
// le test les JOURNALISE, il ne les compte ni pour ni contre.
//
// L'HORLOGE EST CELLE DU VISIONNEUR, et c'est la seule qui compte ici : le temps ÉCOULÉ DEPUIS
// LE DÉBUT DU FILM, soit `(horodatage du paquet − horodatage du premier paquet du chunk 1)`.
// Elle a été contrôlée par trois chemins indépendants au lot R9 (écart 4-17 ms contre le
// manifeste du jeu sur 23 films). Le document, lui, date sur sa grille de frames à partir de
// la PREMIÈRE POSITION lue ; la conversion ci-dessous ajoute l'écart entre les deux origines,
// et n'invente rien.
//
// Gardé par P3_FILM + P3_BOUNDS (patron P1_FILM — les films ne sont pas versionnés), skip par
// défaut, CI comprise :
//
//	CGO_ENABLED=0 P3_FILM=<depot>/data/cache/film_chunks/1cd3848a \
//	  P3_BOUNDS=<depot>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  go test ./internal/analysis/replay/ -run '^TestP3ImpulsionsJGtm' -v -timeout 60m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	p3FilmEnv   = "P3_FILM"
	p3BoundsEnv = "P3_BOUNDS"
	// p3WitnessFilm : le film sur lequel le relevé Theater a été fait. Sur un autre film, le
	// test JOURNALISE sans contrôler — un chiffre attendu n'a de sens que pour son témoin.
	p3WitnessFilm = "1cd3848a"
	// p3WitnessGamertag : le joueur du relevé.
	p3WitnessGamertag = "JGtm"
	// p3FrameToleranceMS : « à une frame près ». L'événement précède le geste rendu à l'écran
	// de quelques dizaines de millisecondes, et la grille du rejeu est à 100 ms : exiger la
	// milliseconde exacte contrôlerait la grille, pas la lecture.
	p3FrameToleranceMS = 100
)

// p3WitnessSeconds : les CINQ instants relevés par la lecture au lot précédent (1:52, 1:55,
// 2:03, 2:05, 2:15), en secondes écoulées depuis le début du film — la forme même sous
// laquelle la fiche de créneaux les a soumis à l'utilisateur, et qu'il a confirmés au
// Theater à 1:51, 1:54, 2:03, 2:05, 2:14.
var p3WitnessSeconds = []int64{112, 115, 123, 125, 135}

// TestP3ImpulsionsJGtm rejoue la chaîne de production sur le film témoin et compare les
// impulsions attribuées au propulseur pour JGtm au relevé Theater.
func TestP3ImpulsionsJGtm(t *testing.T) {
	dir := os.Getenv(p3FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : test d'acceptation sur pièces sauté", p3FilmEnv)
	}
	entry := p3MapEntry(t, dir)
	film := filepath.Base(dir)
	doc, err := BuildFromFilm(film, "halo_infinite", dir, Options{
		MapQuant: &entry,
		Labels:   goldenCatalog(t),
	})
	if err != nil {
		t.Fatalf("assemblage du rejeu : %v", err)
	}
	cov := doc.Coverage.AbilityImpulses
	if cov == nil {
		t.Fatal("aucune couverture d'impulsions : le calque n'a pas été construit")
	}
	t.Logf("%s : %d lecture(s) -> %d geste(s) -> %d publiee(s) | sansIdentite=%d "+
		"familleNonMesuree=%d avantOrigine=%d sansPiste=%d composantAbsent=%t",
		film, cov.Reads, cov.Episodes, cov.Published, cov.NoIdentity, cov.OtherFamily,
		cov.BeforeOrigin, cov.Unpublished, cov.ComponentAbsent)

	offsetMS := p3FilmClockOffsetMS(t, doc)
	got := p3ImpulsesOf(t, doc, p3WitnessGamertag, offsetMS)
	for _, ms := range got {
		t.Logf("  impulsion JGtm a %s (%d ms depuis le debut du film)", p3MMSS(ms), ms)
	}
	if film != p3WitnessFilm {
		t.Logf("film different du temoin %s : chiffres NON controles", p3WitnessFilm)
		return
	}
	p3CompareToWitness(t, got)
}

// p3CompareToWitness confronte les instants mesurés au relevé, sans rien arrondir en douce :
// un instant vaut son témoin s'il tombe sur la même seconde à une frame près.
//
// LE CONTRÔLE EST BORNÉ À LA FENÊTRE DU RELEVÉ (cf. l'en-tête). Ce qui tombe dehors est
// journalisé et DIT — jamais compté comme un faux positif contre un relevé qui ne l'a pas
// regardé, jamais tu non plus.
func p3CompareToWitness(t *testing.T, all []int64) {
	t.Helper()
	from, to := p3WitnessSeconds[0], p3WitnessSeconds[len(p3WitnessSeconds)-1]
	var got []int64
	for _, ms := range all {
		if sec := ms / 1000; sec+1 >= from && sec <= to+1 {
			got = append(got, ms)
			continue
		}
		t.Logf("HORS FENETRE DU RELEVE (%s-%s), ni confirmee ni infirmee : impulsion a %s",
			p3MMSS(from*1000), p3MMSS(to*1000), p3MMSS(ms))
	}
	if len(got) != len(p3WitnessSeconds) {
		t.Errorf("%d impulsion(s) de propulseur pour %s dans la fenetre %s-%s, "+
			"RELEVE THEATER : %d — l'ecart n'est pas masque",
			len(got), p3WitnessGamertag, p3MMSS(from*1000), p3MMSS(to*1000), len(p3WitnessSeconds))
	}
	used := make([]bool, len(got))
	for _, want := range p3WitnessSeconds {
		hit := -1
		for i, ms := range got {
			if used[i] || !p3SameSecond(ms, want) {
				continue
			}
			hit = i
			break
		}
		if hit < 0 {
			t.Errorf("usage releve a %s NON RENDU par la chaine de production", p3MMSS(want*1000))
			continue
		}
		used[hit] = true
	}
	for i, ms := range got {
		if !used[i] {
			t.Errorf("impulsion rendue a %s (%d ms) DANS la fenetre du releve, sans usage en face",
				p3MMSS(ms), ms)
		}
	}
}

// p3SameSecond dit si un instant mesuré tombe sur la seconde attendue, à une frame près —
// la tolérance annoncée, appliquée AVANT la troncature (c'est elle qui décide, pas l'inverse).
func p3SameSecond(ms, wantSec int64) bool {
	for _, d := range []int64{-p3FrameToleranceMS, 0, p3FrameToleranceMS} {
		if (ms+d)/1000 == wantSec {
			return true
		}
	}
	return false
}

// p3ImpulsesOf rend, en millisecondes écoulées depuis le début du film et TRIÉS, les instants
// des impulsions publiées dont la vie appartient au gamertag donné.
func p3ImpulsesOf(t *testing.T, doc ReplayDocument, gamertag string, offsetMS int64) []int64 {
	t.Helper()
	xuid := ""
	for _, r := range doc.Roster {
		if strings.EqualFold(r.Name, gamertag) {
			xuid = r.XUID
			break
		}
	}
	if xuid == "" {
		t.Fatalf("%s absent du roster du film : le releve ne peut pas etre confronte", gamertag)
	}
	var out []int64
	for _, im := range doc.AbilityImpulses {
		if !p3TrackCovers(doc.Tracks, im.Slot, im.T, xuid) {
			continue
		}
		out = append(out, offsetMS+int64(im.T)*int64(doc.FrameIntervalMS))
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// p3TrackCovers dit si la VIE du slot qui couvre cette frame appartient au xuid. Le slot
// migre aux réapparitions : sans la couverture de frame, on créditerait l'occupant précédent.
func p3TrackCovers(tracks []Track, slot uint32, frame int, xuid string) bool {
	for _, tr := range tracks {
		if tr.Slot != slot || frame < tr.StartFrame || frame > tr.EndFrame {
			continue
		}
		return tr.XUID == xuid
	}
	return false
}

// p3FilmClockOffsetMS rend l'écart, en millisecondes, entre la frame 0 du document et le début
// du film. C'est EXACTEMENT ce que le document publie déjà sous `originMs` — « (premier paquet
// de position − premier paquet du chunk 1) / 1000 », cf. origin.go. On le LIT plutôt que de le
// recalculer : deux calculs du même écart divergeraient, et celui-là est déjà contrôlé par un
// témoin indépendant (le fil des morts, accord à moins de 100 ms sur quatre films).
//
// SANS ORIGINE, PAS DE CONFRONTATION : le refus est explicite. Comparer un relevé Theater à des
// frames dont le zéro n'est pas établi comparerait deux horloges différentes.
func p3FilmClockOffsetMS(t *testing.T, doc ReplayDocument) int64 {
	t.Helper()
	if doc.OriginMs == nil {
		t.Fatal("document sans origine : l'horloge du visionneur n'est pas etablie sur ce film")
	}
	if doc.FrameIntervalMS <= 0 {
		t.Fatal("document sans intervalle de frame : l'axe n'a pas d'echelle")
	}
	return *doc.OriginMs
}

// p3MapEntry résout l'entrée de catalogue de la carte du film PAR SES LARGEURS D'AXE, sans
// base de données : `DetectI0Layout` les lit dans le film, et le catalogue dit quelles cartes
// les portent (la même clé que les instruments R8/R9).
func p3MapEntry(t *testing.T, dir string) filmdec.MapQuantEntry {
	t.Helper()
	path := os.Getenv(p3BoundsEnv)
	if path == "" {
		t.Skipf("%s absent : sans bornes de carte le rejeu ne se construit pas", p3BoundsEnv)
	}
	cat, err := filmdec.LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes illisible : %v", err)
	}
	lay, _, err := filmdec.DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("decoupage i0 illisible dans %s : %v", dir, err)
	}
	var names []string
	var got []filmdec.MapQuantEntry
	for name, e := range cat.Maps {
		if e.AxisWidths != lay.AxisW || e.Region != lay.Region {
			continue
		}
		if !p3hasRange(got, e.Range()) {
			got = append(got, e)
			names = append(names, name)
		}
	}
	if len(got) == 0 {
		t.Fatalf("aucune carte du catalogue ne porte les largeurs %v (region %d) de %s",
			lay.AxisW, lay.Region, filepath.Base(dir))
	}
	sort.Strings(names)
	t.Logf("%s : largeurs i0 %v region %d -> %d AABB distincte(s), cartes %v",
		filepath.Base(dir), lay.AxisW, lay.Region, len(got), names)
	return got[0]
}

func p3hasRange(all []filmdec.MapQuantEntry, r filmdec.Vec3Range) bool {
	for _, x := range all {
		if x.Range() == r {
			return true
		}
	}
	return false
}

// p3MMSS met en forme un instant du visionneur. Les millisecondes sont TUES : un relevé se
// fait à la seconde, afficher plus serait afficher une précision que la mesure ne tient pas.
func p3MMSS(ms int64) string {
	s := ms / 1000
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}
