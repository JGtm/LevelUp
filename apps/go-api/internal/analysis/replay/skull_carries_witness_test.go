package replay

// skull_carries_witness_test.go — RE-CUISSON DU TEMOIN avec la reconstruction DE PRODUCTION.
//
// Le gate du porteur (principal 7/7 films) a ete etabli par l'instrument `cmd/oddball-terrain`.
// Ce test rejoue le MEME gate avec le code PUBLIE (`skullCarrySecondsByXUID` + `ResolveRoundIdentity`
// + `SkullTicksComponent`), sur un film reel, et confronte le PORTEUR PRINCIPAL reconstruit au
// porteur principal de l'oracle fige. Il verifie donc que la production reproduit la mesure — pas
// seulement que l'instrument de recherche le faisait.
//
// REGIME : UN FILM PAR PROCESSUS (garde RAM du corpus Oddball), lecture seule, aucune base.
//
//	$env:ATT_FILM="<repo>"; $env:ODDBALL_FILM="d9781168"
//	$env:SKULL_ORACLE="<repo>/.ai/V7.5/replay2d/registre_film/D10_oracle_objective_stats.json"
//	go test ./internal/analysis/replay/ -run SkullCarrierWitness -v

import (
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/filmproc"
)

// skullOracleEnv : chemin de l'oracle fige (film -> xuid -> stats de portage).
const skullOracleEnv = "SKULL_ORACLE"

// skullCarrySecondsByXUID reconstruit la duree de portage (s) par joueur : somme des durees des
// trains de tics sur toutes les manches. C'est la grandeur du gate oracle (porteur principal).
//
// TEST-ONLY : cette re-cuisson du temoin n'a AUCUN appelant de production (le calque publie ne
// somme pas les durees par joueur, il pose les intervalles). Elle vit donc a cote du seul gate qui
// s'en sert, hors du binaire de prod, et reutilise `skullCarryIntervals` — le coeur PUBLIE que le
// temoin doit precisement verifier.
func skullCarrySecondsByXUID(recs []objectiveevents.StatRecord, identity objectiveevents.RoundIdentity) map[string]float64 {
	out := map[string]float64{}
	for _, r := range skullCarryIntervals(recs, identity) {
		if r.xuid == "" {
			continue
		}
		out[r.xuid] += float64(r.t1MS-r.t0MS) / 1000
	}
	return out
}

// skullOracleStat porte les colonnes de l'oracle employees ici.
type skullOracleStat struct {
	Time  float64 `json:"time_as_skull_carrier_seconds"`
	Grabs int     `json:"skull_grabs"`
}

func TestSkullCarrierWitness(t *testing.T) {
	root := attRequireRoot(t)
	id := os.Getenv(d4FilmEnv)
	if id == "" {
		t.Skipf("mesure non demandee : %s vide (identifiant court du film Oddball)", d4FilmEnv)
	}
	oracle := loadSkullOracle(t, id)

	g := filmproc.Arm("skull-carrier-witness", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio — NI POUR NI CONTRE", id, float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire %.2f Gio", id, float64(g.Peak())/(1<<30))
	}()

	src, ok := objOpenFilm(t, root, id)
	if !ok {
		t.Fatalf("%s : film absent du cache", id)
	}
	recs := objectiveevents.StatRecords(src)
	deaths, err := ScanFilmDeaths(objChunkDir(root, id))
	if err != nil {
		t.Fatalf("%s : fil des morts illisible : %v", id, err)
	}
	identity := objectiveevents.ResolveRoundIdentity(recs, deathInstantsOf(deaths))
	carry := skullCarrySecondsByXUID(recs, identity)
	gamertags := skullGamertags(deaths)

	// PORTEUR PRINCIPAL reconstruit vs oracle.
	recX, recMax := argmaxFloat(carry)
	oraX, oraMax := skullOracleArgmax(oracle)
	grabsRec, grabsOra := skullGrabCount(recs), skullOracleGrabs(oracle)

	t.Logf("%s : %d manche(s) resolue(s), %d joueur(s) porteur(s) reconstruit(s) ; prises "+
		"reconstruites=%d oracle=%d", id, len(identity.Rounds()), len(carry), grabsRec, grabsOra)
	for _, x := range sortedXUIDByCarry(carry) {
		gt := gamertags[x]
		if gt == "" {
			gt = x
		}
		t.Logf("  %-16s reconstruit=%5.1fs  oracle=%5.1fs", gt, carry[x], oracle[x].Time)
	}
	t.Logf("PORTEUR PRINCIPAL %s : reconstruit=%s (%.1fs) oracle=%s (%.1fs)",
		id, gamertagOr(gamertags, recX), recMax, gamertagOr(gamertags, oraX), oraMax)

	if oraX == "" {
		t.Fatalf("%s : oracle sans porteur — verification impossible", id)
	}
	if recX != oraX {
		t.Errorf("PORTEUR PRINCIPAL FAUX %s : reconstruit=%s, oracle=%s (le gate exige l'egalite)",
			id, gamertagOr(gamertags, recX), gamertagOr(gamertags, oraX))
	} else {
		t.Logf("VERDICT %s : porteur principal CORRECT (%s).", id, gamertagOr(gamertags, recX))
	}
}

// loadSkullOracle lit l'oracle fige et rend la table xuid -> stats du film demande.
func loadSkullOracle(t *testing.T, id string) map[string]skullOracleStat {
	t.Helper()
	path := os.Getenv(skullOracleEnv)
	if path == "" {
		t.Skipf("mesure non demandee : %s vide (oracle Oddball fige)", skullOracleEnv)
	}
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fourni par l'operateur de la mesure
	if err != nil {
		t.Fatalf("oracle illisible (%s) : %v", path, err)
	}
	var all map[string]map[string]skullOracleStat
	if err := json.Unmarshal(raw, &all); err != nil {
		t.Fatalf("oracle invalide (%s) : %v", path, err)
	}
	film, ok := all[id]
	if !ok || len(film) == 0 {
		t.Fatalf("oracle sans releve pour %s", id)
	}
	return film
}

// skullGamertags rend le gamertag lu DANS le film pour chaque xuid (le fil des morts le nomme).
func skullGamertags(deaths []Death) map[string]string {
	out := map[string]string{}
	for _, d := range deaths {
		if d.Gamertag != "" {
			out[strconv.FormatUint(d.XUID, 10)] = d.Gamertag
		}
	}
	return out
}

func gamertagOr(gamertags map[string]string, xuid string) string {
	if gt := gamertags[xuid]; gt != "" {
		return gt
	}
	return xuid
}

// argmaxFloat rend la cle de valeur maximale et cette valeur (cle vide sur table vide).
func argmaxFloat(m map[string]float64) (string, float64) {
	best, bestV := "", 0.0
	for _, k := range sortedFloatKeys(m) { // ordre total : depart deterministe
		if m[k] > bestV || best == "" {
			best, bestV = k, m[k]
		}
	}
	return best, bestV
}

func skullOracleArgmax(m map[string]skullOracleStat) (string, float64) {
	best, bestV := "", 0.0
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if m[k].Time > bestV || best == "" {
			best, bestV = k, m[k].Time
		}
	}
	return best, bestV
}

func skullOracleGrabs(m map[string]skullOracleStat) int {
	n := 0
	for _, s := range m {
		n += s.Grabs
	}
	return n
}

func sortedFloatKeys(m map[string]float64) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedXUIDByCarry(m map[string]float64) []string {
	out := sortedFloatKeys(m)
	sort.SliceStable(out, func(i, j int) bool { return m[out[i]] > m[out[j]] })
	return out
}
