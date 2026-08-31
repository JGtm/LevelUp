package replay

// assaut_manches_research_test.go — INSTRUMENT DE RECHERCHE : de quoi est faite une MANCHE
// REELLE, et pourquoi `RealRounds` refuse celles de One Bomb.
//
// # La question, telle que le releve A0.3 la pose
//
// `A_PROTOCOLE.md` §2 constat 2 : « une manche de One Bomb porte au plus UNE emission de
// score », donc le critere de `RealRounds` (une suite COHERENTE d'au moins
// `statMinRoundRun`=3 emissions CROISSANTES du score de mode) ne peut jamais etre tenu.
// Consequence mesuree : sur `df8fcbef`/`c75f33b8`/`9f57c612`, seule la manche 0 survit et
// 8 explosions sur 11 sont perdues. Le releve BRUT, lui, somme au score API sur 9/9 films.
//
// Cet instrument imprime les DEUX POPULATIONS a departager, pour toutes les manches brutes
// de chaque film : celles que le releve declare REELLES, et les ancrages fortuits que le
// critere existe pour ecarter. Il ne decide rien — il rend les chiffres sur lesquels un
// second critere d'admission peut etre ecrit, puis fige.
//
// # Le corpus, et pourquoi ces films-la
//
//	One Bomb        `df8fcbef` `c75f33b8` `9f57c612` — les 3 films multi-manches d'Assaut,
//	                verite connue au releve A0.3 (4, 3, 4 explosions ; manches 0..3/0..2).
//	Assaut 1 manche `35b75a31` `69b16f5d` `3d58eb37` `34bb3bc8` `1c01e34f` `ce083875` —
//	                aucune manche au-dela de 0 n'est reelle : TEMOIN NEGATIF.
//	CTF             `53ce4390` — LE contre-exemple documente : une manche fantome y portait
//	                le score d'equipe de 1 a 2 104 avant l'introduction du critere.
//	CTF             `1bc77d2e` — l'autre contre-exemple : `flag_capture_assists` 1 -> 1 569.
//	Oddball         `c88ec007` — manches reelles courtes (la plus petite du corpus, 33
//	                emissions coherentes au slot 6 manche 1).
//
// REGIME : garde `ASSAUT_CACHE` (racine du cache film). Aucune base, aucun reseau.
//
//	$env:ASSAUT_CACHE="C:/.../LevelUp-go-migration/data/cache"
//	go test ./internal/analysis/replay/ -run AssautManchesRecherche -v -timeout 30m

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// amCorpus : film -> manches REELLES connues du releve A0.3 (ou du mode pour les temoins).
// La valeur est le nombre de manches reelles, -1 quand elle n'est pas etablie.
var amCorpus = []struct {
	id      string
	libelle string
	reelles int
}{
	{"df8fcbef", "Assaut One Bomb (4 explosions, manches 0..3)", 4},
	{"c75f33b8", "Assaut One Bomb (3 explosions, manches 0..2)", 3},
	{"9f57c612", "Assaut One Bomb (4 explosions, manches 0..3)", 4},
	{"35b75a31", "Assaut Neutral Bomb (1 manche)", 1},
	{"69b16f5d", "Assaut Neutral Bomb (1 manche)", 1},
	{"3d58eb37", "Assaut Neutral Bomb (1 manche)", 1},
	{"34bb3bc8", "Assaut Neutral Bomb Squad (1 manche)", 1},
	{"1c01e34f", "Husky Raid Assaut (1 manche)", 1},
	{"ce083875", "Assaut Neutral Bomb (1 manche, film exclu du lot A)", 1},
	{"53ce4390", "CTF — contre-exemple manche fantome (score 1 -> 2104)", 1},
	{"1bc77d2e", "CTF — contre-exemple (flag_capture_assists 1 -> 1569)", 1},
	{"c88ec007", "Oddball — manches reelles courtes", -1},
}

// amComps : les canaux mesures. Le score de mode est celui du critere actuel ; les autres
// sont les candidats d'un second critere (ils bougent, eux, dans une manche de One Bomb).
var amComps = []struct {
	label string
	comp  int
	sideB bool
	maxOK int64 // borne de plausibilite, comme statMaxModeScore pour le score de mode
}{
	{"score_mode(0A)", 0, false, 250},
	{"frags(2A)", 2, false, 200},
	{"morts(2B)", 2, true, 200},
	{"assists(3A)", 3, false, 200},
	{"score_perso(1B)", 1, true, 100000},
}

// TestAssautManchesRecherche imprime les deux populations, film par film, manche par manche.
func TestAssautManchesRecherche(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautManchesRecherche")()
	var lignes []string
	entete := "film\tmanche\treelle\tenr\tslots\tslots_j\ttmin_ms\ttmax_ms\tetendue_ms" +
		"\tsuite_score\tsuite_frags\tsuite_morts\tsuite_assists\tsuite_scoreperso\tretenue_actuelle"
	lignes = append(lignes, entete)
	for _, f := range amCorpus {
		src, ok, err := filmcache.Open(cache, f.id)
		if err != nil || !ok {
			t.Logf("FILM %s ABSENT (%v) — saute", f.id, err)
			continue
		}
		recs, tronque := objectiveevents.StatRecordsCtx(context.Background(), src, f.id)
		retenues := objectiveevents.RealRounds(recs)
		t.Logf("FILM %s — %s : %d enregistrements, tronque=%v, manches retenues actuellement=%v",
			f.id, f.libelle, len(recs), tronque, amTriRetenues(retenues))
		for _, round := range amManchesBrutes(recs) {
			lignes = append(lignes, amLigne(f, recs, round, retenues))
		}
	}
	for _, l := range lignes {
		t.Log(l)
	}
}

// amLigne rend la ligne TSV d'une manche brute.
func amLigne(f struct {
	id      string
	libelle string
	reelles int
}, recs []objectiveevents.StatRecord, round int, retenues map[int]bool) string {
	var enr, enrJoueur, tmin, tmax int
	tmin = -1
	slots := map[int]bool{}
	slotsJoueur := map[int]bool{}
	for _, r := range recs {
		if r.Round != round {
			continue
		}
		enr++
		slots[r.Slot] = true
		if !objectiveevents.IsTeamSlot(r.Slot) {
			slotsJoueur[r.Slot] = true
			enrJoueur++
		}
		if tmin < 0 || r.TimeMS < tmin {
			tmin = r.TimeMS
		}
		if r.TimeMS > tmax {
			tmax = r.TimeMS
		}
	}
	reelle := "?"
	if f.reelles >= 0 {
		reelle = "non"
		if round < f.reelles {
			reelle = "OUI"
		}
	}
	cols := []string{
		f.id, fmt.Sprint(round), reelle, fmt.Sprint(enr),
		fmt.Sprint(len(slots)), fmt.Sprint(len(slotsJoueur)), fmt.Sprint(enrJoueur),
		fmt.Sprint(tmin), fmt.Sprint(tmax), fmt.Sprint(tmax - tmin),
	}
	for _, c := range amComps {
		cols = append(cols, fmt.Sprint(amSuiteMax(recs, round, c.comp, c.sideB, c.maxOK)))
	}
	cols = append(cols, fmt.Sprintf("%v", retenues[round]))
	return strings.Join(cols, "\t")
}

// amSuiteMax rend, pour une manche, la plus longue suite STRICTEMENT croissante d'un canal,
// prise au meilleur slot de JOUEUR — exactement la forme du critere de `RealRounds`.
func amSuiteMax(recs []objectiveevents.StatRecord, round, comp int, sideB bool, maxOK int64) int {
	series := map[int][]int64{}
	tempsPar := map[int][]int{}
	for _, r := range recs {
		if r.Round != round || objectiveevents.IsTeamSlot(r.Slot) {
			continue
		}
		v, ok := r.Comps[comp]
		if !ok {
			continue
		}
		val := v.A
		if sideB {
			val = v.B
		}
		if val < 0 || val > maxOK {
			continue
		}
		series[r.Slot] = append(series[r.Slot], val)
		tempsPar[r.Slot] = append(tempsPar[r.Slot], r.TimeMS)
	}
	best := 0
	for slot, vals := range series {
		ordre := amOrdreParTemps(tempsPar[slot])
		tri := make([]int64, len(vals))
		for i, idx := range ordre {
			tri[i] = vals[idx]
		}
		if n := amLIS(tri); n > best {
			best = n
		}
	}
	return best
}

// amOrdreParTemps rend les indices tries par instant croissant (tri stable).
func amOrdreParTemps(temps []int) []int {
	idx := make([]int, len(temps))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return temps[idx[a]] < temps[idx[b]] })
	return idx
}

// amLIS rend la longueur de la plus longue sous-suite STRICTEMENT croissante — la meme
// mesure que `longestRun(pts, true)` du paquet `objectiveevents`.
func amLIS(vals []int64) int {
	var tails []int64
	for _, v := range vals {
		lo, hi := 0, len(tails)
		for lo < hi {
			mid := (lo + hi) / 2
			if tails[mid] < v {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
		if lo == len(tails) {
			tails = append(tails, v)
		} else {
			tails[lo] = v
		}
	}
	return len(tails)
}

// amManchesBrutes rend les numeros de manche presents dans les enregistrements, tries.
func amManchesBrutes(recs []objectiveevents.StatRecord) []int {
	vu := map[int]bool{}
	for _, r := range recs {
		vu[r.Round] = true
	}
	out := make([]int, 0, len(vu))
	for r := range vu {
		out = append(out, r)
	}
	sort.Ints(out)
	return out
}

// amTriRetenues rend les manches retenues, triees, pour un log lisible.
func amTriRetenues(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for r, ok := range m {
		if ok {
			out = append(out, r)
		}
	}
	sort.Ints(out)
	return out
}

// TestAssautManchesControleHorsEchantillon — LE CONTROLE HORS ECHANTILLON du second critere.
//
// # Le critere, et pourquoi il est RELATIF
//
// Le corpus de 12 films de [TestAssautManchesRecherche] separe les deux populations sur le
// nombre d'enregistrements de SLOT JOUEUR d'une manche (`enr_j`) : manche reelle >= 45,
// ancrage fortuit <= 3. Mais en ABSOLU la marge est mince — le corpus libre montre une
// manche fantome a 18 enregistrements (`bfcd1175` manche 6, un film de Slayer, mode qui n'a
// pas de manche). En RELATIF a la manche la PLUS FOURNIE du meme film, la separation est
// d'un ordre de grandeur : ancrage fortuit <= 1,0 %, manche reelle >= 21 % (`df8fcbef`
// manche 1 : 45 enregistrements contre 212 a la manche 3).
//
// C'est la forme retenue : une manche est MATERIELLE si elle porte au moins
// `statMinRoundRecordShare` % des enregistrements de joueur de la manche la plus fournie du
// film. Le denominateur est toujours une manche reelle (c'est la plus fournie), donc la
// mesure ne depend d'aucune constante de duree, de cadence ni de nombre de joueurs — un FFA
// a 6 joueurs (`610363ee`) passe comme un 4v4.
//
// # LE CRITERE DE PASSAGE, ecrit avant la mesure
//
// AUCUNE manche du corpus libre ne doit tomber dans la BANDE INTERDITE 7 %..15 % — le seuil
// de production (10 %) doit se poser au milieu d.un vide, pas au bord d.une population. Une
// seule manche y tombe et le seuil n.est pas separant — le second critere devra etre ecrit
// autrement. Les films sont echantillonnes PAR MODE (deux par variante du registre), donc
// le controle couvre Slayer, Fiesta, BTB, Husky Raid, Firefight, Oddball, CTF, KOTH,
// Strongholds et les variantes communautaires — des modes que le corpus d'Assaut n'a pas.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache" ; $env:ASSAUT_FILMS_LIBRES="6d07c363,a12f267b,..."
//	go test ./internal/analysis/replay/ -run AssautManchesControle -v -timeout 60m
func TestAssautManchesControleHorsEchantillon(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	libres := os.Getenv("ASSAUT_FILMS_LIBRES")
	if cache == "" || libres == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE et ASSAUT_FILMS_LIBRES requis")
	}
	defer amArmeSentinelle(t, "TestAssautManchesControleHorsEchantillon")()
	var bande []string
	films, manches := 0, 0
	for _, id := range strings.Split(libres, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		src, ok, err := filmcache.Open(cache, id)
		if err != nil || !ok {
			t.Logf("film %s absent — saute", id)
			continue
		}
		recs, _ := objectiveevents.StatRecordsCtx(context.Background(), src, id)
		if len(recs) == 0 {
			t.Logf("film %s : 0 enregistrement — saute", id)
			continue
		}
		films++
		enrPar := amEnrJoueurParManche(recs)
		ref := 0
		for _, n := range enrPar {
			if n > ref {
				ref = n
			}
		}
		for _, round := range amManchesBrutes(recs) {
			manches++
			part := 0.0
			if ref > 0 {
				part = 100 * float64(enrPar[round]) / float64(ref)
			}
			t.Logf("LIBRE\t%s\t%d\t%d\t%d\t%.2f", id, round, enrPar[round], ref, part)
			if part >= amBandeBasse && part <= amBandeHaute {
				bande = append(bande, fmt.Sprintf("%s manche %d : %d/%d enregistrements joueur = %.2f %%",
					id, round, enrPar[round], ref, part))
			}
		}
	}
	t.Logf("CORPUS LIBRE : %d films, %d manches brutes", films, manches)
	for _, b := range bande {
		t.Logf("  BANDE INTERDITE : %s", b)
	}
	if len(bande) > 0 {
		t.Errorf("SEUIL NON SEPARANT : %d manche(s) dans la bande %.0f %%..%.0f %%",
			len(bande), amBandeBasse, amBandeHaute)
	}
}

// amBandeBasse / amBandeHaute bornent la BANDE INTERDITE du controle : le seuil de
// production (10 %) doit se poser au milieu d'un vide, pas au bord d'une population.
const (
	amBandeBasse = 7.0
	amBandeHaute = 15.0
)

// amEnrJoueurParManche compte, par manche brute, les enregistrements de slot JOUEUR.
func amEnrJoueurParManche(recs []objectiveevents.StatRecord) map[int]int {
	out := map[int]int{}
	for _, r := range recs {
		if objectiveevents.IsTeamSlot(r.Slot) {
			continue
		}
		out[r.Round]++
	}
	return out
}

// TestAssautPointsDeModeParJoueur — DIAGNOSTIC : la suite du point de mode, slot par slot et
// manche par manche, sur les films d'Assaut. Sert a confronter les evenements nommes publies
// aux explosions datees du releve A0.3.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache" ; go test ./internal/analysis/replay/ -run AssautPointsDeMode -v
func TestAssautPointsDeModeParJoueur(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautPointsDeModeParJoueur")()
	for _, f := range amCorpus[:3] { // les 3 One Bomb
		src, ok, err := filmcache.Open(cache, f.id)
		if err != nil || !ok {
			t.Logf("film %s absent — saute", f.id)
			continue
		}
		recs, _ := objectiveevents.StatRecordsCtx(context.Background(), src, f.id)
		t.Logf("=== %s (%s) — manches retenues %v", f.id, f.libelle,
			amTriRetenues(objectiveevents.RealRounds(recs)))
		byRound := objectiveevents.SeriesByRound(recs,
			objectiveevents.StatComponent{Comp: 0, SideB: false}, false)
		rounds := make([]int, 0, len(byRound))
		for r := range byRound {
			rounds = append(rounds, r)
		}
		sort.Ints(rounds)
		for _, r := range rounds {
			slots := make([]int, 0, len(byRound[r]))
			for s := range byRound[r] {
				slots = append(slots, s)
			}
			sort.Ints(slots)
			for _, s := range slots {
				pts := byRound[r][s]
				if len(pts) == 0 || pts[len(pts)-1].Value == 0 {
					continue
				}
				t.Logf("  manche %d slot %2d : final=%d, %d point(s), premier increment a %d ms",
					r, s, pts[len(pts)-1].Value, len(pts), amPremierIncrement(pts))
			}
		}
		// Ce que la production publierait.
		named := objectiveevents.NamedEventsFrom(recs, objectiveevents.ObjectiveTypeBomb)
		t.Logf("  -> %d evenement(s) nomme(s) publie(s) :", len(named))
		for _, n := range named {
			t.Logf("     slot %2d a %d ms (%s)", n.Slot, n.TimeMS, n.Stat)
		}
	}
}

// amPremierIncrement rend l'instant du premier point ou la valeur depasse la premiere.
func amPremierIncrement(pts []objectiveevents.ScorePoint) int {
	if len(pts) == 0 {
		return -1
	}
	for _, p := range pts {
		if p.Value > pts[0].Value {
			return p.TimeMS
		}
	}
	return -1
}

// TestAssautMancheSansPorteur — DIAGNOSTIC CIBLE : les 2 explosions sur 11 qui n'ont AUCUN slot
// de joueur porteur. Imprime tous les enregistrements bruts de `comp 0` de la manche visee.
func TestAssautMancheSansPorteur(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautMancheSansPorteur")()
	cas := []struct {
		id    string
		round int
		msEsp int
	}{
		{"df8fcbef", 3, 778033},
		{"c75f33b8", 1, 395724},
	}
	for _, c := range cas {
		src, ok, err := filmcache.Open(cache, c.id)
		if err != nil || !ok {
			continue
		}
		recs, _ := objectiveevents.StatRecordsCtx(context.Background(), src, c.id)
		t.Logf("=== %s manche %d (explosion attendue a %d ms)", c.id, c.round, c.msEsp)
		parSlot := map[int][]string{}
		for _, r := range recs {
			if r.Round != c.round {
				continue
			}
			v, ok := r.Comps[0]
			if !ok {
				continue
			}
			if v.A != 0 || v.B != 0 {
				parSlot[r.Slot] = append(parSlot[r.Slot],
					fmt.Sprintf("%d ms A=%d B=%d", r.TimeMS, v.A, v.B))
			}
		}
		slots := make([]int, 0, len(parSlot))
		for s := range parSlot {
			slots = append(slots, s)
		}
		sort.Ints(slots)
		for _, s := range slots {
			equipe := ""
			if objectiveevents.IsTeamSlot(s) {
				equipe = " (EQUIPE)"
			}
			t.Logf("  slot %2d%s : %s", s, equipe, strings.Join(parSlot[s], " | "))
		}
		if len(slots) == 0 {
			t.Logf("  AUCUN enregistrement comp 0 non nul dans cette manche")
		}
	}
}

// TestAssautParasiteCe083875 — DIAGNOSTIC : les 66 evenements publies au MEME instant sur
// `ce083875`, le film que le protocole du lot A avait exclu (pont d'identite a 10,6 %).
func TestAssautParasiteCe083875(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautParasiteCe083875")()
	src, ok, err := filmcache.Open(cache, "ce083875")
	if err != nil || !ok {
		t.Skip("film absent")
	}
	recs, _ := objectiveevents.StatRecordsCtx(context.Background(), src, "ce083875")
	t.Logf("manches retenues : %v", amTriRetenues(objectiveevents.RealRounds(recs)))
	for _, r := range recs {
		if r.TimeMS < 218_000 || r.TimeMS > 221_000 {
			continue
		}
		v, ok := r.Comps[0]
		if !ok {
			continue
		}
		t.Logf("  %d ms slot %2d manche %d : comp0 A=%d B=%d (%d composants)",
			r.TimeMS, r.Slot, r.Round, v.A, v.B, len(r.Comps))
	}
	named := objectiveevents.NamedEventsFrom(recs, objectiveevents.ObjectiveTypeBomb)
	t.Logf("%d evenements nommes ; les 8 premiers :", len(named))
	for i, n := range named {
		if i >= 8 {
			break
		}
		t.Logf("  slot %2d a %d ms", n.Slot, n.TimeMS)
	}
}

// TestAssautDomaineComp0 — DIAGNOSTIC : de quoi est fait un enregistrement porteur de `comp 0`,
// et ou se situe l'aberration de `ce083875` (A=66, B=16635, un seul composant).
func TestAssautDomaineComp0(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautDomaineComp0")()
	histoB := map[string]int{}
	histoN := map[int]int{}
	var horsB, total int
	ids := make([]string, 0, len(amCorpus))
	for _, f := range amCorpus {
		ids = append(ids, f.id)
	}
	for _, id := range strings.Split(os.Getenv("ASSAUT_FILMS_LIBRES"), ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	for _, f := range ids {
		src, ok, err := filmcache.Open(cache, f)
		if err != nil || !ok {
			continue
		}
		recs, _ := objectiveevents.StatRecordsCtx(context.Background(), src, f)
		for _, r := range recs {
			v, ok := r.Comps[0]
			if !ok || objectiveevents.IsTeamSlot(r.Slot) {
				continue
			}
			total++
			histoN[len(r.Comps)]++
			switch {
			case v.B == 0:
				histoB["B=0"]++
			case v.B > 0 && v.B <= 250:
				histoB["0<B<=250"]++
				t.Logf("  B PETIT NON NUL : %s slot %d manche %d a %d ms — A=%d B=%d, %d composant(s)",
					f, r.Slot, r.Round, r.TimeMS, v.A, v.B, len(r.Comps))
			default:
				histoB["B>250 ou <0"]++
				horsB++
				if horsB <= 12 {
					t.Logf("  HORS DOMAINE B : %s slot %d manche %d a %d ms — A=%d B=%d, %d composant(s)",
						f, r.Slot, r.Round, r.TimeMS, v.A, v.B, len(r.Comps))
				}
			}
		}
	}
	t.Logf("%d enregistrements joueur porteurs de comp 0", total)
	for k, n := range histoB {
		t.Logf("  %s -> %d", k, n)
	}
	tailles := make([]int, 0, len(histoN))
	for k := range histoN {
		tailles = append(tailles, k)
	}
	sort.Ints(tailles)
	for _, k := range tailles {
		t.Logf("  %d composant(s) -> %d enregistrement(s)", k, histoN[k])
	}
}

// TestAssautPontIdentite — DIAGNOSTIC : combien d'explosions NOMMEES survivent au pont
// d'identite par manche, film par film. C'est le chiffre qui arrive a l'ECRAN — le gate A5,
// lui, mesure ce que le statborg NOMME, en amont du pont.
func TestAssautPontIdentite(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautPontIdentite")()
	var nommes, identifies int
	for _, f := range amCorpus[:9] { // les 9 films d'Assaut
		src, ok, err := filmcache.Open(cache, f.id)
		if err != nil || !ok {
			continue
		}
		recs, _ := objectiveevents.StatRecordsCtx(context.Background(), src, f.id)
		deaths, err := ScanFilmDeaths(filepath.Join(cache, "film_chunks", f.id))
		if err != nil {
			t.Logf("%s : fil des morts illisible (%v)", f.id, err)
			continue
		}
		var di []objectiveevents.DeathInstant
		for _, d := range deaths {
			di = append(di, objectiveevents.DeathInstant{
				XUID: fmt.Sprint(d.XUID), TimeMS: int(d.TimeMS)})
		}
		named := objectiveevents.NamedEventsFrom(recs, objectiveevents.ObjectiveTypeBomb)
		identity := objectiveevents.ResolveRoundIdentity(recs, di)
		ident := objectiveevents.IdentifyNamedEventsByRound(named, identity)
		nomme := 0
		for _, e := range ident {
			if e.XUID != "" {
				nomme++
			}
		}
		nommes += len(named)
		identifies += nomme
		manques := ""
		for _, e := range named {
			if identity.At(e.Slot, e.TimeMS) == "" {
				manques += fmt.Sprintf(" [slot %d a %d ms SANS identite]", e.Slot, e.TimeMS)
			}
		}
		plat := objectiveevents.SlotIdentityByDeaths(recs, di)
		platOK := 0
		for _, e := range named {
			if plat[e.Slot] != "" {
				platOK++
			}
		}
		t.Logf("%s : %d nomme(s) -> %d identifie(s) par manche, %d par le pont PLAT (%d slots), %d mort(s)%s",
			f.id, len(named), nomme, platOK, len(plat), len(deaths), manques)
	}
	t.Logf("BILAN PONT : %d explosion(s) nommee(s) -> %d identifiee(s) (%.1f %%)",
		nommes, identifies, 100*float64(identifies)/float64(nommes))
}

// amArmeSentinelle arme le plafond memoire de MESURE pour un balayage de corpus, et rend la
// fonction de desarmement (a differer par l'appelant).
//
// POURQUOI CHAQUE INSTRUMENT DE CE FICHIER L'APPELLE (leçon du 2026-08-31). Ces balayages
// enchainent jusqu'a 65 films DANS UN SEUL PROCESSUS. Le decodage du statborg est borne par
// `statMaxRecordsPerFilm` et les pics mesures restent sous le dixieme de gibioctet — mais c'est
// une PROPRIETE OBSERVEE, pas une garantie, et la doctrine du depot ne fait pas d'exception :
// tout processus qui enchaine des films arme sa sentinelle (cf. `internal/filmproc`).
func amArmeSentinelle(t *testing.T, nom string) func() {
	t.Helper()
	g := filmproc.Arm(nom, filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — balayage interrompu pour proteger la machine",
			float64(peak)/(1<<30))
	})
	return func() {
		g.Disarm()
		t.Logf("pic memoire observe : %.2f Gio (plafond souple %d Gio)",
			float64(g.Peak())/(1<<30), filmproc.MeasureLimitGiB)
	}
}
