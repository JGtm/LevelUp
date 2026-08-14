package replay

// i48_ability_width_test.go — INSTRUMENT DE MESURE de l'ÉTAPE 1 du plan
// .ai/V7.5/replay2d/PLAN_CAPACITES_ACTIVES.md : « relire l'index de capacité sur 6 BITS,
// après la porte décrite par le binaire », mesure déclarée prioritaire le 2026-07-27 et
// jamais exécutée jusqu'ici.
//
// CE QU'IL MESURE, sans rien modifier au décodeur de production :
//
//  1. la LARGEUR RÉELLE du champ. La production lit 3 bits après le motif 20 bits
//     `invAbilityPattern` (cf. inventory_decode.go, règle R1). Le binaire décrit pour i48
//     `R(1) porte ; si porte==0, R(6)` (RECETTE_LOADOUT §9). Les deux lectures ne peuvent
//     pas être toutes deux exactes : l'instrument publie, pour la MÊME position, quatre
//     candidats de lecture (voir `abilityReadings`) et leurs distributions ;
//  2. la COUVERTURE de la table de noms : quels index apparaissent réellement, sur autant
//     de films que demandé — c'est le chiffre de l'étape 2 (`abilityIndexLabels` ne nomme
//     que 3/4/5/6).
//
// CONTRÔLE QUI PEUT ÉCHOUER, ÉNONCÉ AVANT (item 1.2 du plan) : la palette `sofd` de la
// famille A (RECETTE_LOADOUT §13) donne capteur=1, mur=2, grappin=4, propulseur=5. La
// vérité terrain Theater (VERITE_TERRAIN_INVENTAIRE_2026-07-27) donne sur 000d5950 les
// slots 512/513/516 = grappin, 514/518/519 = propulseur, 515 = capteur, 517 = mur. Une
// lecture correcte doit donc rendre 4 sur le premier triplet, 5 sur le second, 1 sur le
// slot 515 et 2 sur le slot 517. La lecture 3 bits actuelle rend 4 et 5 sur les triplets
// mais 6 et 3 sur les deux isolés : si aucun candidat ne redresse ces deux-là, l'hypothèse
// des 6 bits est RÉFUTÉE et il faut chercher ailleurs.
//
// LECTURE SEULE, gardé par ABILITY_FILM_ROOT, sauté partout ailleurs (CI comprise).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 ABILITY_FILM_ROOT=<repo>/data/cache/film_chunks \
//	  ABILITY_FILMS=000d5950,9e8fb31b \
//	  go test ./internal/analysis/replay/ -run '^TestAbilityIndexWidth$' -timeout 30m -v
//
//	# couverture (étape 2) : balayer les N premiers films du cache
//	CGO_ENABLED=0 ABILITY_FILM_ROOT=<repo>/data/cache/film_chunks ABILITY_FILM_LIMIT=40 \
//	  go test ./internal/analysis/replay/ -run '^TestAbilityIndexWidth$' -timeout 60m -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	abilityRootEnv  = "ABILITY_FILM_ROOT"
	abilityFilmsEnv = "ABILITY_FILMS"
	abilityLimitEnv = "ABILITY_FILM_LIMIT"
)

// abilityReading nomme une lecture candidate du champ de capacité, relative au bit `p` où
// commence le motif 20 bits. `gate` est la position du bit de porte (-1 = pas de porte dans
// ce candidat), `at`/`width` le champ lui-même.
type abilityReading struct {
	name  string
	gate  int // offset depuis p, ou -1
	at    int // offset depuis p
	width int
}

// abilityReadings énumère les lectures confrontées. Elles ne sont PAS choisies après coup :
// chacune correspond à une hypothèse écrite avant la mesure.
//
//	prod3     : la lecture de production (R1 de inventory_decode.go).
//	large6    : le §9 de la recette — « élargir la fenêtre à 6 bits, trois bits plus tôt ».
//	            La porte attendue est alors le bit p+16.
//	suite6    : l'index de production serait les 3 bits de POIDS FORT d'un champ de 6.
//	aval6     : la porte et le champ de 6 bits viennent APRÈS les 3 bits lus aujourd'hui
//	            (i48 = R(3) rotation puis R(1) porte puis R(6) identité — l'ordre littéral
//	            du binaire, cf. recette §9 : 0xA34 compteur, 0xA35 identité).
var abilityReadings = []abilityReading{
	{name: "prod3   [p+20,3]", gate: -1, at: 20, width: 3},
	{name: "large6  [p+17,6]", gate: 16, at: 17, width: 6},
	{name: "suite6  [p+20,6]", gate: -1, at: 20, width: 6},
	{name: "aval6   [p+24,6]", gate: 23, at: 24, width: 6},
}

// abilitySample est une lecture de capacité localisée dans un film.
type abilitySample struct {
	film string
	slot uint32
	patP int
	vals []uint32 // une valeur par abilityReadings
	gate []int    // bit de porte (-1 si le candidat n'en a pas)
}

func TestAbilityIndexWidth(t *testing.T) {
	root := os.Getenv(abilityRootEnv)
	if root == "" {
		t.Skipf("%s absent : instrument de mesure sauté", abilityRootEnv)
	}
	films, err := abilityFilmList(root)
	if err != nil {
		t.Fatalf("liste de films illisible : %v", err)
	}
	if len(films) == 0 {
		t.Fatalf("aucun film à mesurer sous %s", root)
	}
	t.Logf("FILMS mesurés : %d", len(films))

	var (
		samples    []abilitySample
		records    int
		uniqAnchor int
		perFilmIdx = map[string]map[uint32]int{}
	)
	for _, f := range films {
		dir := filepath.Join(root, f)
		n := filmdec.CountFilmChunks(dir)
		if n == 0 {
			t.Logf("  %s : aucun chunk, ignoré", f)
			continue
		}
		perFilmIdx[f] = map[uint32]int{}
		for c := 1; c <= n; c++ {
			chunk, err := filmdec.ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, p := range filmdec.WalkPackets(chunk) {
				if p.Type != filmdec.PacketTypeKeyframe {
					continue
				}
				pay := p.Payload(chunk)
				for _, sp := range invRecordSpans(pay) {
					if sp.ti != invBipedTI {
						continue
					}
					records++
					hits := abilityPatternHits(pay, sp.from, sp.to)
					if len(hits) != 1 {
						continue // même règle d'unicité que la production
					}
					uniqAnchor++
					s := abilitySample{film: f, slot: uint32(sp.slot), patP: hits[0]}
					for _, r := range abilityReadings {
						s.vals = append(s.vals, invBits(pay, hits[0]+r.at, r.width))
						g := -1
						if r.gate >= 0 {
							g = int(invBits(pay, hits[0]+r.gate, 1))
						}
						s.gate = append(s.gate, g)
					}
					perFilmIdx[f][s.vals[0]]++
					samples = append(samples, s)
				}
			}
		}
	}
	if len(samples) == 0 {
		t.Fatal("aucune ancre de capacité unique : rien à mesurer")
	}
	t.Logf("RECORDS biped d'image-clé %d · ancre UNIQUE %d (%.1f %%)",
		records, uniqAnchor, 100*float64(uniqAnchor)/float64(records))

	abilityLogDistributions(t, samples)
	abilityLogGroundTruth(t, samples)
	abilityLogCoverage(t, perFilmIdx)
}

// abilityLogDistributions publie, par candidat de lecture, la distribution des valeurs et
// celle du bit de porte. C'est le matériau des items 1.1 et 1.3.
func abilityLogDistributions(t *testing.T, samples []abilitySample) {
	t.Log("== DISTRIBUTIONS PAR CANDIDAT DE LECTURE ==")
	for i, r := range abilityReadings {
		hist := map[uint32]int{}
		gateHist := map[int]int{}
		for _, s := range samples {
			hist[s.vals[i]]++
			gateHist[s.gate[i]]++
		}
		keys := make([]uint32, 0, len(hist))
		for k := range hist {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(a, b int) bool { return keys[a] < keys[b] })
		parts := make([]string, 0, len(keys))
		inRange := 0
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%d:%d", k, hist[k]))
			if k < 11 {
				inRange += hist[k]
			}
		}
		t.Logf("  %s : %d valeurs distinctes · %.1f %% dans [0,11) · %s",
			r.name, len(keys), 100*float64(inRange)/float64(len(samples)), strings.Join(parts, " "))
		if r.gate >= 0 {
			t.Logf("      porte p%+d : 0 -> %d · 1 -> %d", r.gate-20, gateHist[0], gateHist[1])
		}
	}
}

// abilityGroundTruth est la vérité terrain Theater de 000d5950 (image 250), par slot :
// le RANG ATTENDU dans la palette `sofd` famille A (recette §13).
var abilityGroundTruth = map[uint32]struct {
	nom  string
	rang uint32
}{
	512: {"grappin", 4}, 513: {"grappin", 4}, 516: {"grappin", 4},
	514: {"propulseur", 5}, 518: {"propulseur", 5}, 519: {"propulseur", 5},
	515: {"capteur de menace", 1},
	517: {"mur portatif", 2},
}

// abilityLogGroundTruth confronte chaque candidat aux huit slots de la vérité terrain.
// C'est le contrôle 1.2, énoncé avant la mesure.
func abilityLogGroundTruth(t *testing.T, samples []abilitySample) {
	t.Log("== CONTRÔLE 1.2 — vérité terrain 000d5950, slots 512..519 ==")
	slots := make([]uint32, 0, len(abilityGroundTruth))
	for s := range abilityGroundTruth {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(a, b int) bool { return slots[a] < slots[b] })
	for i, r := range abilityReadings {
		ok, tot := 0, 0
		var lines []string
		for _, sl := range slots {
			hist := map[uint32]int{}
			for _, s := range samples {
				if s.film == "000d5950" && s.slot == sl {
					hist[s.vals[i]]++
				}
			}
			if len(hist) == 0 {
				continue
			}
			best, bestN := uint32(0), 0
			for v, n := range hist {
				if n > bestN {
					best, bestN = v, n
				}
			}
			tot++
			mark := "KO"
			if best == abilityGroundTruth[sl].rang {
				ok++
				mark = "OK"
			}
			lines = append(lines, fmt.Sprintf("%d=%d(att.%d %s)", sl, best, abilityGroundTruth[sl].rang, mark))
		}
		t.Logf("  %s : %d/%d · %s", r.name, ok, tot, strings.Join(lines, " "))
	}
}

// abilityLogCoverage publie la couverture de la table de noms : quels index sont observés,
// et lesquels sont nommés aujourd'hui (3/4/5/6 dans replay_labels.toml).
func abilityLogCoverage(t *testing.T, perFilm map[string]map[uint32]int) {
	t.Log("== COUVERTURE (étape 2) — index observés par la lecture de production ==")
	named := map[uint32]bool{3: true, 4: true, 5: true, 6: true}
	all := map[uint32]int{}
	for f, h := range perFilm {
		keys := make([]uint32, 0, len(h))
		for k := range h {
			keys = append(keys, k)
			all[k] += h[k]
		}
		sort.Slice(keys, func(a, b int) bool { return keys[a] < keys[b] })
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%d:%d", k, h[k]))
		}
		t.Logf("  film %s : %s", f, strings.Join(parts, " "))
	}
	keys := make([]uint32, 0, len(all))
	namedN, totalN := 0, 0
	for k := range all {
		keys = append(keys, k)
		totalN += all[k]
		if named[k] {
			namedN += all[k]
		}
	}
	sort.Slice(keys, func(a, b int) bool { return keys[a] < keys[b] })
	var missing []string
	for _, k := range keys {
		if !named[k] {
			missing = append(missing, strconv.Itoa(int(k)))
		}
	}
	t.Logf("  TOTAL : %d index distincts %v · lectures nommées %d/%d (%.1f %%) · index OBSERVÉS NON NOMMÉS : %v",
		len(keys), keys, namedN, totalN, 100*float64(namedN)/float64(totalN), missing)
}

// abilityPatternHits rend les positions du motif 20 bits (le bit `p` de son premier bit),
// selon la MÊME règle d'ancrage que invAbilityIn — dont il est le décalque, à ceci près
// qu'il publie la position du motif au lieu de l'index déjà lu.
func abilityPatternHits(pay []byte, from, to int) []int {
	var out []int
	var w uint32
	const mask28 = (uint32(1) << 28) - 1
	for b := from; b < to; b++ {
		w = ((w << 1) | invBitAt(pay, b)) & mask28
		if b-from < 27 || w != invAbilityAnchor {
			continue
		}
		for off := 0; off <= 60; off++ {
			p := b + 1 + off
			if p+20 > to {
				break
			}
			if invBits(pay, p, 20) != invAbilityPattern {
				continue
			}
			out = append(out, p)
			break
		}
	}
	return out
}

// abilityFilmList résout la liste de films : ABILITY_FILMS (liste explicite) sinon les
// ABILITY_FILM_LIMIT premiers répertoires du cache.
func abilityFilmList(root string) ([]string, error) {
	if v := strings.TrimSpace(os.Getenv(abilityFilmsEnv)); v != "" {
		var out []string
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
		return out, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	limit := 20
	if v := os.Getenv(abilityLimitEnv); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		out = append(out, e.Name())
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
