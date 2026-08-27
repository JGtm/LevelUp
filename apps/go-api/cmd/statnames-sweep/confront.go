package main

// confront.go — la CONFRONTATION : balayage x oracle, sur MOITIES DISJOINTES.
//
// AUCUN film n'est decode ici : l'entree est le TSV d'une passe de balayage et un oracle
// JSON fige par l'operateur. La regle est celle du protocole D10 (§7) : un emplacement
// est CANDIDAT s'il replique une colonne sur la moitie de RECHERCHE (accord >= 90 %,
// >= 6 paires dont >= 3 a valeur oracle non nulle) ; le VERDICT se rend sur la moitie de
// VERIFICATION, jamais sur celle qui a elu le candidat. Un compteur nul partout ne peut
// pas elire un emplacement muet : c'est la garde anti-zero.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	// confrontAccordMin / confrontPairesMin / confrontNonNullesMin : les seuils du
	// protocole D10 §7, recopies sans modification.
	confrontAccordMin    = 0.90
	confrontPairesMin    = 6
	confrontNonNullesMin = 3
)

// confrontCols : les colonnes oracle et leurs encodages declares d'avance (protocole §7).
// "n" : egalite avec round(v) ; "s" : round(v) ; "ds" : round(10*v).
var confrontCols = []struct {
	col  string
	encs []string
}{
	{"skull_grabs", []string{"n"}},
	{"skull_scoring_ticks", []string{"n"}},
	{"skull_carriers_killed", []string{"n"}},
	{"time_as_skull_carrier_seconds", []string{"s", "ds"}},
	{"longest_time_as_skull_carrier_seconds", []string{"s", "ds"}},
}

// slotKey adresse un emplacement du statborg.
type slotKey struct {
	comp int
	side string
}

func (k slotKey) String() string { return fmt.Sprintf("comp %d %s", k.comp, k.side) }

// sweepData porte ce qu'une passe de balayage a rendu, par film.
type sweepData struct {
	// joueurs : xuids identifies par le pont, par film.
	joueurs map[string]map[string]bool
	// finals : valeur finale par film -> emplacement -> xuid.
	finals map[string]map[slotKey]map[string]int64
}

// accord est le resultat d'une confrontation sur un ensemble de films.
type accord struct {
	matches, paires, nonNulles int
}

func (a accord) taux() float64 {
	if a.paires == 0 {
		return 0
	}
	return float64(a.matches) / float64(a.paires)
}

func (a accord) String() string {
	return fmt.Sprintf("%d/%d = %.1f %% (%d paire(s) non nulle(s))",
		a.matches, a.paires, 100*a.taux(), a.nonNulles)
}

// runConfront charge les entrees et rend candidats puis verdicts.
func runConfront(a runArgs) error {
	if a.sweepPath == "" || a.oraclePath == "" || len(a.search) == 0 || len(a.verify) == 0 {
		return fmt.Errorf("-confront exige -sweep, -oracle, -search et -verify")
	}
	sw, err := loadSweep(a.sweepPath)
	if err != nil {
		return err
	}
	oracle, err := loadOracle(a.oraclePath)
	if err != nil {
		return err
	}
	fmt.Printf("CONFRONTATION : recherche %v · verification %v · seuils accord >= %.0f %%, "+
		">= %d paires dont >= %d non nulles\n",
		a.search, a.verify, 100*confrontAccordMin, confrontPairesMin, confrontNonNullesMin)
	for _, ce := range confrontCols {
		for _, enc := range ce.encs {
			confrontOne(sw, oracle, ce.col, enc, a.search, a.verify)
		}
	}
	return nil
}

// confrontOne confronte UNE colonne-encodage a tous les emplacements : meilleur accord de
// recherche publie toujours, candidats verifies sur l'autre moitie.
func confrontOne(sw sweepData, oracle oracleData, col, enc string, search, verify []string) {
	keys := allKeys(sw)
	type scored struct {
		k slotKey
		a accord
	}
	var all []scored
	for _, k := range keys {
		all = append(all, scored{k, mesureAccord(sw, oracle, k, col, enc, search)})
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].a.taux() != all[j].a.taux() {
			return all[i].a.taux() > all[j].a.taux()
		}
		if all[i].k.comp != all[j].k.comp {
			return all[i].k.comp < all[j].k.comp
		}
		return all[i].k.side < all[j].k.side
	})
	if len(all) == 0 {
		fmt.Printf("COLONNE %s [%s] : aucun emplacement balaye\n", col, enc)
		return
	}
	best := all[0]
	fmt.Printf("COLONNE %s [%s] : meilleur accord de RECHERCHE %s -> %s\n",
		col, enc, best.k, best.a)
	retenus := 0
	for _, s := range all {
		if s.a.taux() < confrontAccordMin || s.a.paires < confrontPairesMin ||
			s.a.nonNulles < confrontNonNullesMin {
			continue
		}
		retenus++
		v := mesureAccord(sw, oracle, s.k, col, enc, verify)
		verdict := "NE REPLIQUE PAS (verification sous le seuil)"
		if v.taux() >= confrontAccordMin {
			verdict = "REPLIQUE"
		}
		fmt.Printf("CANDIDAT %s [%s] : %s — recherche %s ; VERIFICATION %s -> %s\n",
			col, enc, s.k, s.a, v, verdict)
	}
	if retenus == 0 {
		fmt.Printf("NEGATIF %s [%s] : aucun candidat au seuil sur la moitie de recherche "+
			"(meilleur : %s a %s)\n", col, enc, best.k, best.a)
	}
}

// mesureAccord compte les paires (joueur, film) d'un ensemble de films pour un emplacement
// et une colonne-encodage. Un joueur identifie SANS emission sur l'emplacement vaut 0 —
// un compteur qui n'a jamais emis est un compteur a zero, pas une absence de donnee.
func mesureAccord(sw sweepData, oracle oracleData, k slotKey, col, enc string, films []string) accord {
	var out accord
	for _, film := range films {
		for xuid, cols := range oracle[film] {
			if !sw.joueurs[film][xuid] {
				continue
			}
			want, ok := encode(cols[col], enc)
			if !ok {
				continue
			}
			out.paires++
			if want != 0 {
				out.nonNulles++
			}
			if sw.finals[film][k][xuid] == want {
				out.matches++
			}
		}
	}
	return out
}

// encode traduit une valeur oracle dans l'encodage teste (protocole §7).
func encode(v float64, enc string) (int64, bool) {
	switch enc {
	case "n", "s":
		return int64(math.Round(v)), true
	case "ds":
		return int64(math.Round(10 * v)), true
	default:
		return 0, false
	}
}

// allKeys rend tous les emplacements vus par le balayage, ordonnes.
func allKeys(sw sweepData) []slotKey {
	seen := map[slotKey]bool{}
	for _, byKey := range sw.finals {
		for k := range byKey {
			seen[k] = true
		}
	}
	out := make([]slotKey, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].comp != out[j].comp {
			return out[i].comp < out[j].comp
		}
		return out[i].side < out[j].side
	})
	return out
}

// oracleData : film -> xuid -> colonne -> valeur.
type oracleData map[string]map[string]map[string]float64

// loadOracle lit l'oracle JSON fige.
func loadOracle(path string) (oracleData, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fourni par l'operateur
	if err != nil {
		return nil, fmt.Errorf("oracle : %w", err)
	}
	var out oracleData
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("oracle invalide : %w", err)
	}
	return out, nil
}

// loadSweep parse le TSV d'une passe de balayage (lignes JOUEUR et SWEEP).
func loadSweep(path string) (sweepData, error) {
	f, err := os.Open(path) //nolint:gosec // chemin fourni par l'operateur
	if err != nil {
		return sweepData{}, fmt.Errorf("balayage : %w", err)
	}
	defer f.Close()
	sw := sweepData{joueurs: map[string]map[string]bool{}, finals: map[string]map[slotKey]map[string]int64{}}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		champs := strings.Split(strings.TrimSpace(sc.Text()), "\t")
		switch {
		case champs[0] == "JOUEUR" && len(champs) == 4:
			film, xuid := champs[1], champs[3]
			if sw.joueurs[film] == nil {
				sw.joueurs[film] = map[string]bool{}
			}
			sw.joueurs[film][xuid] = true
		case champs[0] == "SWEEP" && len(champs) == 7:
			if err := sw.addSweep(champs); err != nil {
				return sweepData{}, err
			}
		}
	}
	if err := sc.Err(); err != nil {
		return sweepData{}, fmt.Errorf("balayage : %w", err)
	}
	return sw, nil
}

// addSweep enregistre une ligne SWEEP : film, slot, xuid, comp, side, final.
func (sw sweepData) addSweep(champs []string) error {
	film, xuid, side := champs[1], champs[3], champs[5]
	if xuid == "-" {
		return nil // slot non nomme : le denominateur du balayage, pas de la confrontation
	}
	comp, err := strconv.Atoi(champs[4])
	if err != nil {
		return fmt.Errorf("ligne SWEEP : composant %q", champs[4])
	}
	final, err := strconv.ParseInt(champs[6], 10, 64)
	if err != nil {
		return fmt.Errorf("ligne SWEEP : valeur %q", champs[6])
	}
	k := slotKey{comp: comp, side: side}
	if sw.finals[film] == nil {
		sw.finals[film] = map[slotKey]map[string]int64{}
	}
	if sw.finals[film][k] == nil {
		sw.finals[film][k] = map[string]int64{}
	}
	sw.finals[film][k][xuid] = final
	return nil
}
