package replay

// vip_periodes_test.go — LOT VIP COURONNE, E3 : LES PERIODES DE PORT DE COURONNE.
//
// LE PROTOCOLE EST COMMITE AVANT CE FICHIER (`.ai/V7.5/replay2d/registre_film/
// VIP_COURONNE_PROTOCOLE.md` §3). Ce qui suit l'applique.
//
// # LA QUESTION
//
// E2 a etabli que `comp 22 A` reproduit `TimesSelectedAsVip` EXACTEMENT par joueur (gate corrige
// TENU, `VIP_temoin_corrige.log`). Reste a savoir si ce compteur donne des EVENEMENTS DATES : la
// meme mecanique que `flag_grabs` (`incrementTimes`) convertit un compteur cumule en un instant
// par unite gagnee. Si les increments sont ETALES sur le match, chaque selection DATE l'ouverture
// d'une periode VIP, fermee par la MORT du VIP (kill feed) ou la selection suivante — patron
// `flag_carries`. La preuve se lit dans le RECOUVREMENT : si les periodes reconstruites somment,
// par joueur, a `TimeAsVipSeconds` de l'API, elles sont datees ET justes.
//
// # POURQUOI NI POSITIONS NI BORNES NI OFFSET
//
// Les evenements nommes (statborg) ET le fil des morts sont sur l'horloge du MATCH (meme horloge,
// cf. slotidentity_deaths.go : « Les deux sont sur l'horloge du MATCH »). Une periode = mort -
// selection : tout offset constant s'annule. On ne lit donc ni les images-cles ni les bornes de
// carte — seulement les records d'entite (comp 22 A) et le chunk des highlight events (morts).
//
// REGIME : gardes `ATT_FILM` (racine cache) + `VIP_FILM` (id court) + `VIP_ORACLE` (oracle fige
// JSON, film -> xuid -> colonnes VipStats), UN FILM PAR PROCESSUS, lecture seule, AUCUNE base.
//
//	$env:ATT_FILM="<cache>"; $env:VIP_FILM="00761d27"; $env:VIP_ORACLE="<V_oracle_vipstats.json>"
//	go test ./internal/analysis/replay/ -run VIPPeriodes -v

import (
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/filmproc"
)

const (
	// vipOracleEnv designe l'oracle FIGE (le meme que la confrontation E2) : on n'en lit que
	// TimeAsVipSeconds. Aucune base ouverte — l'oracle arrive en entree.
	vipOracleEnv = "VIP_ORACLE"
	// vipPeriodesSeuilRecouv : le gate PRIMAIRE (recouv >= 90 %), recopie du protocole §3.
	// Metrique de recouvrement IDENTIQUE a D6 portage. Le recouv et son temoin de couverture
	// restent IMPRIMES comme DIAGNOSTIC (la couverture ne discrimine pas une permutation quand
	// le recouv vaut 100 % — cf. §3-bis, meme piege qu'E2).
	vipPeriodesSeuilRecouv = 0.90
	// TEMOIN CORRIGE (§3-bis) : l'EXACTITUDE PAR JOUEUR est le discriminant correct d'une
	// attribution DETERMINISTE (periode = mort - selection, sans parametre libre). Un joueur est
	// EXACT si |recon - oracle| <= vipExactToleranceS. Gate : signal >= 7/8, temoin permute <= 2/8,
	// marge >= 5 joueurs, par film sur >= 2/3.
	vipExactToleranceS = 1.0
	vipExactSignalMin  = 7
	vipExactTemoinMax  = 2
	vipExactMargeMin   = 5
	// vipPeriodesGraine fixe le tirage du temoin : deux executions rendent la MEME sortie.
	vipPeriodesGraine = 20260827
)

// TestVIPPeriodes — LA MESURE. Un film par processus.
func TestVIPPeriodes(t *testing.T) {
	root := attRequireRoot(t)
	id := os.Getenv(vipFilmEnv)
	if id == "" {
		t.Skipf("mesure non demandee : %s vide", vipFilmEnv)
	}
	oracle, ok := vipTimeOracle(t, id)
	if !ok {
		return
	}
	g := filmproc.Arm("vip-periodes", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio", id, float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire observe %.2f Gio", id, float64(g.Peak())/(1<<30))
	}()

	src, okF := objOpenFilm(t, root, id)
	if !okF {
		t.Fatalf("%s : film absent du cache", id)
	}
	recs, truncated := objectiveevents.StatRecordsCtx(context.Background(), src, id)
	if truncated {
		t.Logf("%s : enregistrements TRONQUES — periodes partielles, et cela se dit", id)
	}
	deaths, err := ScanFilmDeaths(objChunkDir(root, id))
	if err != nil {
		t.Fatalf("%s : fil des morts illisible : %v", id, err)
	}
	identity := objectiveevents.SlotIdentityByDeaths(recs, deathInstantsOf(deaths))
	events := objectiveevents.NamedEventsFrom(recs, objectiveevents.ObjectiveTypeVip)
	t.Logf("%s : %d record(s), %d selection(s) VIP, %d slot(s) nomme(s), %d mort(s), oracle sur "+
		"%d joueur(s)", id, len(recs), vipCompte(events), len(identity), len(deaths), len(oracle))

	vipDatesDiag(t, id, events)

	// LA RECONSTRUCTION EST CELLE DE PRODUCTION (`vipReconstructPeriods`, vip_crown.go) : la
	// mesure et le calque partagent la MEME regle, une seule source. La mesure travaille en
	// horloge du MATCH (secondes), le build convertira en frames.
	matchEnd := vipMatchEndMS(events, deaths)
	rec := vipParJoueur(vipReconstructPeriods(events, identity, deaths, matchEnd))
	rng := rand.New(rand.NewSource(vipPeriodesGraine)) //nolint:gosec // temoin reproductible
	tem := vipParJoueur(vipReconstructPeriods(events, vipTemoinIdentity(identity, rng), deaths, matchEnd))
	vipPeriodesVerdict(t, id, rec, tem, oracle)
}

// vipTimeOracle lit TimeAsVipSeconds par xuid pour le film mesure, depuis l'oracle fige.
func vipTimeOracle(t *testing.T, id string) (map[string]float64, bool) {
	t.Helper()
	path := os.Getenv(vipOracleEnv)
	if path == "" {
		t.Skipf("mesure non demandee : %s vide (oracle VIP fige)", vipOracleEnv)
	}
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fourni par l'operateur de la mesure
	if err != nil {
		t.Fatalf("oracle illisible (%s) : %v", path, err)
	}
	var all map[string]map[string]map[string]float64
	if err := json.Unmarshal(raw, &all); err != nil {
		t.Fatalf("oracle invalide (%s) : %v", path, err)
	}
	film, ok := all[id]
	if !ok || len(film) == 0 {
		t.Logf("NON EXPLOITABLE %s : aucun releve VIP. NI POUR NI CONTRE.", id)
		return nil, false
	}
	out := map[string]float64{}
	for xuid, cols := range film {
		out[xuid] = cols["TimeAsVipSeconds"]
	}
	return out, true
}

// vipCompte compte les selections VIP parmi les evenements nommes.
func vipCompte(events []objectiveevents.NamedEvent) int {
	n := 0
	for _, e := range events {
		if e.Stat == objectiveevents.StatVipSelected {
			n++
		}
	}
	return n
}

// vipDatesDiag imprime, par slot, l'etalement des instants d'increment de comp 22 A. Si tout un
// slot a ses selections au MEME instant (span 0), le compteur ne DATE pas — le dire.
func vipDatesDiag(t *testing.T, id string, events []objectiveevents.NamedEvent) {
	t.Helper()
	bySlot := map[int][]int64{}
	for _, e := range events {
		if e.Stat == objectiveevents.StatVipSelected {
			bySlot[e.Slot] = append(bySlot[e.Slot], int64(e.TimeMS))
		}
	}
	slots := make([]int, 0, len(bySlot))
	for s := range bySlot {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	empiles := 0
	for _, s := range slots {
		ts := bySlot[s]
		sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
		distincts := 1
		for i := 1; i < len(ts); i++ {
			if ts[i] != ts[i-1] {
				distincts++
			}
		}
		span := int64(0)
		if len(ts) > 0 {
			span = ts[len(ts)-1] - ts[0]
		}
		if len(ts) >= 2 && distincts == 1 {
			empiles++
		}
		t.Logf("  DATES slot %d : %d selection(s), %d instant(s) distinct(s), etalement %d ms",
			s, len(ts), distincts, span)
	}
	if empiles > 0 {
		t.Logf("ALERTE DATES %s : %d slot(s) ont leurs selections EMPILEES sur un seul instant — "+
			"le compteur ne DATE pas ces selections.", id, empiles)
	}
}

// vipParJoueur somme les durees de periode par xuid decimal, depuis les periodes brutes de
// PRODUCTION (`vipRawPeriod`). Les slots non pontes (xuid vide) ne comptent pas — ils ne
// peuvent porter aucune couronne attribuee.
func vipParJoueur(periodes []vipRawPeriod) map[string]float64 {
	out := map[string]float64{}
	for _, p := range periodes {
		if p.xuid == "" || p.t1MS <= p.t0MS {
			continue
		}
		out[p.xuid] += float64(p.t1MS-p.t0MS) / 1000
	}
	return out
}

// vipTemoinIdentity permute l'affectation slot -> xuid : le TEMOIN du protocole (attribution
// aleatoire). Meme chaine, meme cloture, meme metrique — la comparaison est honnete.
func vipTemoinIdentity(identity map[int]string, rng *rand.Rand) map[int]string {
	slots := make([]int, 0, len(identity))
	xuids := make([]string, 0, len(identity))
	for s, x := range identity {
		if x == "" {
			continue
		}
		slots = append(slots, s)
		xuids = append(xuids, x)
	}
	sort.Ints(slots)
	sort.Strings(xuids)
	rng.Shuffle(len(xuids), func(i, j int) { xuids[i], xuids[j] = xuids[j], xuids[i] })
	out := map[int]string{}
	for i, s := range slots {
		out[s] = xuids[i]
	}
	return out
}

// vipRecouvrement rend le recouvrement `sum_x min(rec, oracle) / sum_x oracle` et le total API.
func vipRecouvrement(rec, oracle map[string]float64) (float64, float64) {
	var inter, total float64
	for x, api := range oracle {
		total += api
		if r := rec[x]; r < api {
			inter += r
		} else {
			inter += api
		}
	}
	if total <= 0 {
		return 0, 0
	}
	return inter / total, total
}

// vipExactCount compte les joueurs dont la reconstruction tombe a moins de vipExactToleranceS
// de l'oracle — le discriminant CORRECT d'une attribution deterministe (§3-bis).
func vipExactCount(rec, oracle map[string]float64) int {
	n := 0
	for x, api := range oracle {
		d := rec[x] - api
		if d < 0 {
			d = -d
		}
		if d <= vipExactToleranceS {
			n++
		}
	}
	return n
}

// vipPeriodesVerdict rend le verdict chiffre : gate PRIMAIRE (recouv) + TEMOIN CORRIGE
// (exactitude par joueur, §3-bis). La couverture reste imprimee comme diagnostic.
func vipPeriodesVerdict(t *testing.T, id string, rec, tem, oracle map[string]float64) {
	t.Helper()
	recouv, total := vipRecouvrement(rec, oracle)
	if total <= 0 {
		t.Logf("NON EXPLOITABLE %s : TimeAsVip API nul. NI POUR NI CONTRE.", id)
		return
	}
	recTem, _ := vipRecouvrement(tem, oracle)
	xuids := make([]string, 0, len(oracle))
	for x := range oracle {
		xuids = append(xuids, x)
	}
	sort.Strings(xuids)
	for _, x := range xuids {
		t.Logf("  %s : API %.1f s · reconstruit %.1f s · temoin %.1f s",
			x[len(x)-4:], oracle[x], rec[x], tem[x])
	}
	exSig, exTem := vipExactCount(rec, oracle), vipExactCount(tem, oracle)
	n := len(oracle)
	t.Logf("DIAG %s : recouvrement %.1f %% (gate primaire >= %.0f %%) ; temoin couverture %.1f %% "+
		"(inapt, cf. §3-bis)", id, 100*recouv, 100*vipPeriodesSeuilRecouv, 100*recTem)
	t.Logf("SIGNAL %s : exactitude signal %d/%d (seuil >= %d/8, tol %.1f s) ; temoin permute %d/%d "+
		"(seuil <= %d/8) ; marge %d joueur(s) (seuil >= %d)", id, exSig, n, vipExactSignalMin,
		vipExactToleranceS, exTem, n, vipExactTemoinMax, exSig-exTem, vipExactMargeMin)
	gate := recouv >= vipPeriodesSeuilRecouv && exSig >= vipExactSignalMin &&
		exTem <= vipExactTemoinMax && exSig-exTem >= vipExactMargeMin
	switch {
	case gate:
		t.Logf("VERDICT %s : GATE TENU — periodes reconstruisent TimeAsVip (recouv %.1f %%), "+
			"exactitude %d/%d contre temoin %d/%d effondre.", id, 100*recouv, exSig, n, exTem, n)
	case recouv >= vipPeriodesSeuilRecouv:
		t.Logf("VERDICT %s : recouv TENU mais exactitude/temoin sous le gate corrige.", id)
	default:
		t.Logf("VERDICT %s : recouvrement SOUS le seuil — les periodes ne reconstruisent pas.", id)
	}
}
