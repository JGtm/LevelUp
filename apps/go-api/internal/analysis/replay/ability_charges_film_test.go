package replay

// ability_charges_film_test.go — LE TEST D'ACCEPTATION DU LOT P5, SUR PIÈCES ET CONTRE UNE
// VÉRITÉ TERRAIN (patron de ability_impulses_film_test.go).
//
// CE QU'IL CONTRÔLE, ET POURQUOI IL EXISTE. Le rapport R11 établit que le quartet haut
// d'i56 est un compteur de charges entières, et sa validation porte sur DEUX témoins
// indépendants : la série 4, 3, 2, 1, 0 de JGtm sur `1cd3848a`, tombée exactement sur les
// cinq usages de propulseur relevés au Theater (1:51, 1:54, 2:03, 2:05, 2:14), et
// l'appariement des baisses de grappin aux accroches `grappleLines` (36/36 sur quatre
// films, témoin décalé 2/36). Ces mesures venaient d'INSTRUMENTS de recherche ; ce test
// rejoue la CHAÎNE DE PRODUCTION ENTIÈRE (BuildFromFilm, catalogue réel du titre) et
// vérifie qu'elle rend la même chose.
//
// L'HORLOGE EST CELLE DU VISIONNEUR (temps écoulé depuis le début du film), lue dans
// `originMs` déjà publié — cf. p3FilmClockOffsetMS, réutilisé tel quel.
//
// Gardé par P5_FILM + P5_BOUNDS (film témoin propulseur) et P5_GRAPPLE_FILM (film témoin
// grappin de R11 : `f2966f08` ou `a6ae19fb`), skip par défaut, CI comprise :
//
//	CGO_ENABLED=0 P5_FILM=<depot>/data/cache/film_chunks/1cd3848a \
//	  P5_GRAPPLE_FILM=<depot>/data/cache/film_chunks/f2966f08 \
//	  P5_BOUNDS=<depot>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  go test ./internal/analysis/replay/ -run '^TestP5Charges' -v -timeout 60m

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

const (
	p5FilmEnv        = "P5_FILM"
	p5GrappleFilmEnv = "P5_GRAPPLE_FILM"
	p5BoundsEnv      = "P5_BOUNDS"
	// p5WitnessFilm : le film du relevé Theater (propulseur, JGtm).
	p5WitnessFilm = "1cd3848a"
	// p5PairToleranceMS : la tolérance d'appariement baisse <-> accroche de R11 §3.2 (±1,5 s),
	// reprise telle quelle — pas un réglage de ce test.
	p5PairToleranceMS = 1500
)

// p5WitnessSeries : la série du relevé — l'instant (secondes écoulées depuis le début du
// film, la même forme que p3WitnessSeconds) et la valeur ATTENDUE du compteur. Les instants
// sont ceux de la mesure R11 §2 (1:52, 1:55, 2:03, 2:05, 2:15), que l'utilisateur a
// confirmés au Theater à une seconde près (1:51, 1:54, 2:03, 2:05, 2:14).
var p5WitnessSeries = []struct {
	sec     int64
	charges int
}{
	{112, 4}, {115, 3}, {123, 2}, {125, 1}, {135, 0},
}

// TestP5ChargesJGtm rejoue la chaîne de production sur le film témoin et compare la série
// de charges du propulseur de JGtm au relevé Theater.
func TestP5ChargesJGtm(t *testing.T) {
	dir := os.Getenv(p5FilmEnv)
	if dir == "" {
		t.Skipf("%s absent : test d'acceptation sur pièces sauté", p5FilmEnv)
	}
	doc := p5Build(t, dir)
	cov := doc.Coverage.AbilityCharges
	if cov == nil {
		t.Fatal("aucune couverture de charges : le calque n'a pas été construit")
	}
	t.Logf("%s : %d lecture(s) -> %d publiee(s) | sansIdentite=%d familleNonMesuree=%d "+
		"attributionIndisponible=%d avantOrigine=%d sansPiste=%d composantAbsent=%t",
		filepath.Base(dir), cov.Reads, cov.Published, cov.NoIdentity, cov.OtherFamily,
		cov.NoResolver, cov.BeforeOrigin, cov.Unpublished, cov.ComponentAbsent)

	offsetMS := p3FilmClockOffsetMS(t, doc)
	got := p5ChargesOf(t, doc, p3WitnessGamertag, "thruster", offsetMS)
	for _, c := range got {
		t.Logf("  charge JGtm a %s (%d ms) : %d restante(s)", p3MMSS(c.ms), c.ms, c.charges)
	}
	if filepath.Base(dir) != p5WitnessFilm {
		t.Logf("film different du temoin %s : chiffres NON controles", p5WitnessFilm)
		return
	}
	p5CompareToWitness(t, got)
}

// p5CompareToWitness confronte la série mesurée au relevé, dans la fenêtre du relevé
// seulement (l'en-tête de p3CompareToWitness dit pourquoi) : chaque lecture attendue doit
// être rendue À SA SECONDE (une frame près) AVEC SA VALEUR, et rien d'autre ne doit sortir
// dans la fenêtre. Ce qui tombe dehors est journalisé — jamais compté ni pour ni contre.
func p5CompareToWitness(t *testing.T, all []p5Charge) {
	t.Helper()
	from, to := p5WitnessSeries[0].sec, p5WitnessSeries[len(p5WitnessSeries)-1].sec
	var got []p5Charge
	for _, c := range all {
		if sec := c.ms / 1000; sec+1 >= from && sec <= to+1 {
			got = append(got, c)
			continue
		}
		t.Logf("HORS FENETRE DU RELEVE (%s-%s), ni confirmee ni infirmee : charge %d a %s",
			p3MMSS(from*1000), p3MMSS(to*1000), c.charges, p3MMSS(c.ms))
	}
	if len(got) != len(p5WitnessSeries) {
		t.Errorf("%d lecture(s) de charge dans la fenetre %s-%s, RELEVE : %d — l'ecart n'est pas masque",
			len(got), p3MMSS(from*1000), p3MMSS(to*1000), len(p5WitnessSeries))
	}
	used := make([]bool, len(got))
	for _, want := range p5WitnessSeries {
		hit := -1
		for i, c := range got {
			if used[i] || c.charges != want.charges || !p3SameSecond(c.ms, want.sec) {
				continue
			}
			hit = i
			break
		}
		if hit < 0 {
			t.Errorf("lecture attendue a %s (charges=%d) NON RENDUE par la chaine de production",
				p3MMSS(want.sec*1000), want.charges)
			continue
		}
		used[hit] = true
	}
	for i, c := range got {
		if !used[i] {
			t.Errorf("lecture rendue a %s (charges=%d) DANS la fenetre du releve, sans attendu en face",
				p3MMSS(c.ms), c.charges)
		}
	}
}

// TestP5ChargesGrappinTemoin rejoue la chaîne sur un film témoin grappin de R11 et apparie
// les BAISSES de charge de famille grappin aux accroches `grappleLines` publiées (±1,5 s).
//
// LES DEUX DIRECTIONS SONT MESURÉES ET RAPPORTÉES, une seule est un GATE. R11 §3.2 prouve
// que chaque ACCROCHE a sa baisse (36/36, témoin décalé 2/36) : c'est l'assertion. R11 §3.3
// mesure aussi que le canal des charges a un MEILLEUR RAPPEL que `grappleLines` (42 baisses
// pour 25 accroches sur `53ce4390`) : une baisse SANS accroche en face n'est donc pas un
// faux positif — c'est un usage que `grappleLines` n'a pas su exploiter. Elles sont
// JOURNALISÉES, jamais tues, jamais comptées contre le canal.
func TestP5ChargesGrappinTemoin(t *testing.T) {
	dir := os.Getenv(p5GrappleFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : temoin grappin saute", p5GrappleFilmEnv)
	}
	doc := p5Build(t, dir)
	if doc.Coverage.AbilityCharges == nil {
		t.Fatal("aucune couverture de charges : le calque n'a pas été construit")
	}
	interval := int64(doc.FrameIntervalMS)
	// L'AXE EST UNIQUE, ET C'EST LE GATE QUI L'EXIGE : baisses ET accroches sont datées
	// `originMs + T*interval` (l'horloge du visionneur). La première version de ce test
	// n'ajoutait l'offset qu'aux baisses — chaque accroche semblait alors précéder sa baisse
	// d'exactement `originMs` (~9 s sur `f2966f08`) et l'appariement rendait 0/6.
	offset := p5OriginMS(doc)
	// Les baisses de famille grappin, tous slots : première lecture d'un slot (la première
	// valeur transmise est ce qui reste APRÈS le premier usage — R11 §2), ou valeur
	// strictement inférieure à la précédente du même slot.
	drops := p5GrappleDrops(doc)
	t.Logf("%s : %d lecture(s) grappin publiee(s) -> %d baisse(s) ; %d accroche(s) grappleLines",
		filepath.Base(dir), p5CountFamily(doc, "grapple"), len(drops), len(doc.GrappleLines))

	// GATE — chaque accroche publiée d'un slot COUVERT par le canal des charges a une baisse
	// à moins de 1,5 s. Une accroche d'un slot sans AUCUNE lecture de charge publiée (vie
	// sans rang i48 : refusée et comptée `noIdentity`) est journalisée à part — le refus
	// d'identité est le contrat, pas un raté d'appariement.
	covered := map[uint32]bool{}
	for _, c := range doc.AbilityCharges {
		if c.Family == "grapple" {
			covered[c.Slot] = true
		}
	}
	unpairedHooks := 0
	for _, l := range doc.GrappleLines {
		hookMS := offset + int64(l.T0)*interval
		if !covered[l.Slot] {
			t.Logf("  accroche slot=%d a %s : slot sans lecture de charge publiee (identite refusee)",
				l.Slot, p3MMSS(hookMS))
			continue
		}
		if !p5HasDropNear(drops, l.Slot, hookMS) {
			unpairedHooks++
			t.Errorf("accroche slot=%d a %s SANS baisse de charge a moins de %d ms",
				l.Slot, p3MMSS(hookMS), p5PairToleranceMS)
		}
	}
	// L'AUTRE DIRECTION, rapportée telle quelle : les baisses sans accroche en face sont le
	// rappel supérieur du canal (R11 §3.3), pas des faux positifs.
	unpairedDrops := 0
	for _, d := range drops {
		if !p5HasHookNear(doc.GrappleLines, d.slot, d.ms, offset, interval) {
			unpairedDrops++
			t.Logf("  baisse slot=%d a %s (charges=%d) sans accroche grappleLines en face — "+
				"usage vu par i56 seul (rappel superieur, R11 §3.3)", d.slot, p3MMSS(d.ms), d.charges)
		}
	}
	t.Logf("appariement : %d/%d accroche(s) couverte(s) appariees ; %d baisse(s) sans accroche",
		len(doc.GrappleLines)-unpairedHooks, len(doc.GrappleLines), unpairedDrops)
}

// p5Charge est une lecture datée sur l'horloge du visionneur.
type p5Charge struct {
	slot    uint32
	ms      int64
	charges int
}

// p5Build rejoue la chaîne de production entière sur un film.
func p5Build(t *testing.T, dir string) ReplayDocument {
	t.Helper()
	path := os.Getenv(p5BoundsEnv)
	if path == "" {
		t.Skipf("%s absent : sans bornes de carte le rejeu ne se construit pas", p5BoundsEnv)
	}
	entry := mapEntryFromCatalog(t, dir, path)
	doc, err := buildFromFilmDir(filepath.Base(dir), "halo_infinite", dir, Options{
		MapQuant: &entry,
		Labels:   goldenCatalog(t),
	})
	if err != nil {
		t.Fatalf("assemblage du rejeu : %v", err)
	}
	return doc
}

// p5ChargesOf rend, datées sur l'horloge du visionneur et TRIÉES, les lectures publiées de
// la famille donnée dont la vie appartient au gamertag donné.
func p5ChargesOf(t *testing.T, doc ReplayDocument, gamertag, family string,
	offsetMS int64) []p5Charge {
	t.Helper()
	xuid := ""
	for _, r := range doc.Roster {
		if equalFoldASCII(r.Name, gamertag) {
			xuid = r.XUID
			break
		}
	}
	if xuid == "" {
		t.Fatalf("%s absent du roster du film : le releve ne peut pas etre confronte", gamertag)
	}
	var out []p5Charge
	for _, c := range doc.AbilityCharges {
		if c.Family != family || !p3TrackCovers(doc.Tracks, c.Slot, c.T, xuid) {
			continue
		}
		out = append(out, p5Charge{
			slot: c.Slot, ms: offsetMS + int64(c.T)*int64(doc.FrameIntervalMS), charges: c.Charges})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ms < out[j].ms })
	return out
}

// p5OriginMS rend l'offset de l'horloge du visionneur (originMs publié, sinon zéro) — LE
// SEUL endroit du test qui le lit : baisses et accroches se datent par lui, et deux
// lectures divergentes recréeraient la faute d'axe que ce test a payée (0/6 apparié).
func p5OriginMS(doc ReplayDocument) int64 {
	if doc.OriginMs != nil {
		return *doc.OriginMs
	}
	return 0
}

// p5GrappleDrops rend les BAISSES de la famille grappin, datées sur l'horloge du
// visionneur (le même axe que les accroches dans le gate — cf. p5OriginMS).
func p5GrappleDrops(doc ReplayDocument) []p5Charge {
	offset := p5OriginMS(doc)
	last := map[uint32]int{}
	seen := map[uint32]bool{}
	var out []p5Charge
	for _, c := range doc.AbilityCharges { // déjà triées (instant, slot)
		if c.Family != "grapple" {
			continue
		}
		ms := offset + int64(c.T)*int64(doc.FrameIntervalMS)
		if !seen[c.Slot] || c.Charges < last[c.Slot] {
			out = append(out, p5Charge{slot: c.Slot, ms: ms, charges: c.Charges})
		}
		seen[c.Slot], last[c.Slot] = true, c.Charges
	}
	return out
}

func p5CountFamily(doc ReplayDocument, family string) int {
	n := 0
	for _, c := range doc.AbilityCharges {
		if c.Family == family {
			n++
		}
	}
	return n
}

func p5HasDropNear(drops []p5Charge, slot uint32, hookMS int64) bool {
	for _, d := range drops {
		if d.slot == slot && absMS(d.ms-hookMS) <= p5PairToleranceMS {
			return true
		}
	}
	return false
}

func p5HasHookNear(lines []GrappleLine, slot uint32, dropMS, offset, interval int64) bool {
	for _, l := range lines {
		if l.Slot == slot && absMS(offset+int64(l.T0)*interval-dropMS) <= p5PairToleranceMS {
			return true
		}
	}
	return false
}

func absMS(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// equalFoldASCII évite d'importer strings pour une seule comparaison de gamertag.
func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
