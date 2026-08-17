package filmdec

// sonde_registre_verdicts_test.go — LES QUATRE VERDICTS des sondes du lot F. Le moteur de
// balayage est dans `sonde_registre_scan_test.go` ; ici on ne fait que lire ce qu'il a recolte.
//
//	F2  ti=47 splash-message : quelles VALEURS ses deux composants portes transmettent, quand,
//	    et lesquelles tombent (ou ne tombent jamais) autour d'un evenement d'objectif.
//	F3  ti=4  high-frequency : ce que porte son R(8) — compteur ? drapeau ? — et a quel rythme.
//	F4  tacmap ti=34 / ti=30 : combien de records en multijoueur, contre leur fantome.
//	F5  ti=13 property-name : combien de valeurs distinctes, et tombent-elles aux memes instants
//	    que ti=10 ?
//
// AUCUN CODE DE PRODUCTION (D9) : ces sondes ne publient rien, ne nomment rien, n'ajoutent aucun
// champ. Elles rendent des tables et des verdicts.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// probeFenetreMS : demi-fenetre autour d'un evenement d'objectif, ECRITE AVANT LA MESURE
// (l'enonce de F2 dit « a +/- 2 s »).
const probeFenetreMS = 2000

// TestSondesRegistre execute les quatre sondes sur UN film, en une seule passe.
func TestSondesRegistre(t *testing.T) {
	dir := probeDir(t)
	release := LockProcessDecode()
	defer release()

	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("lecture de chunk_00 (registre) : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("analyse du registre : %v", err)
	}
	t.Logf("FILM %s · registre : %d archetypes · empreinte %016x",
		filepath.Base(dir), len(reg.Archetypes), RegistryFingerprint(reg))
	n := CountFilmChunks(dir)
	debut := time.Now()
	census := probeRecenseKF(dir, n)
	t.Logf("COUT — recensement (une passe sur les images-cle) : %s",
		time.Since(debut).Round(time.Millisecond))
	archs := probeResoutArchetypes(t, reg, census)
	probeBandes(archs, census)
	probeJournaliseBandes(t, archs)

	hor, evs := probeOracleObjectif(t)
	debut = time.Now()
	m := probeBalaye(dir, n, archs, hor)
	t.Logf("COUT — passe delta : %s · %d chunks · %d paquets delta · %d emissions",
		time.Since(debut).Round(time.Millisecond), m.chunks, m.deltaPaquets, len(m.emissions))
	probeTriEmissions(m.emissions)
	probeJournaliseRecords(t, &m)

	sondeF2(t, dir, &m, evs)
	sondeF3(t, &m)
	sondeF4(t, &m)
	sondeF5(t, &m)
}

// probeJournaliseBandes publie les bandes et leurs fantomes.
func probeJournaliseBandes(t *testing.T, archs [probeRoleCount]*probeArch) {
	t.Helper()
	t.Logf("BANDES (slots vus en image-cle -> bande comblee et purgee · fantome de meme taille) :")
	for r, a := range archs {
		if a == nil {
			continue
		}
		t.Logf("  %-38s ti=%-3d  %4d vus  ->  %4d bande  ·  %4d fantome",
			probeRoleNoms[r].libelle, a.ti, len(a.slotsKF), len(a.band), len(a.ghost))
	}
}

// probeJournaliseRecords publie, par archetype, les records reconnus contre leur fantome.
func probeJournaliseRecords(t *testing.T, m *probeMoisson) {
	t.Helper()
	t.Logf("RECORDS DELTA par archetype (denominateur : %d paquets delta) :", m.deltaPaquets)
	for r, a := range m.archs {
		if a == nil {
			continue
		}
		st := m.stats[r]
		t.Logf("  %-38s ti=%-3d  %7d records · fantome %7d · rapport %s · marches abouties %d/%d",
			probeRoleNoms[r].libelle, a.ti, st.records, st.ghostRecords,
			probeRapport(st.records, st.ghostRecords), st.abouties, st.marches)
		t.Logf("      masque : %d records annoncent i%d (le composant identifiant) = %.2f %%"+
			" · %d records HORS GRAMMAIRE (index >= %d) = %.2f %%",
			st.avecIdentifiant, a.idxIdentifiant,
			100*float64(st.avecIdentifiant)/float64(maxI(1, st.records)),
			st.horsGrammaire, len(a.arch.Components),
			100*float64(st.horsGrammaire)/float64(maxI(1, st.records)))
	}
}

// probeOracleObjectif charge le manifeste (horloge du match) et les evenements d'objectif
// NOMMES. La famille d'objectif est FOURNIE, jamais devinee : appliquer la table `flag` a un
// film KOTH fabrique 267 « evenements de drapeau » sur un match qui n'a jamais vu de drapeau
// (mesure du lot C). Sans elle, F2 se limite aux valeurs et n'a pas d'oracle.
func probeOracleObjectif(t *testing.T) (probeHorloge, []objectiveevents.NamedEvent) {
	t.Helper()
	root, short := os.Getenv(probeCacheEnv), os.Getenv(probeShortEnv)
	if root == "" || short == "" {
		t.Logf("ORACLE : %s ou %s absent — pas d'horloge de match, F2 se limite aux valeurs.",
			probeCacheEnv, probeShortEnv)
		return probeHorloge{}, nil
	}
	src, ok, err := filmcache.Open(root, short)
	if err != nil || !ok {
		t.Logf("ORACLE : manifeste indisponible (%v) — F2 se limite aux valeurs.", err)
		return probeHorloge{}, nil
	}
	hor := probeHorloge{startMS: map[int]int{}}
	for _, c := range src.Chunks() {
		hor.startMS[c.Index] = c.StartMS
	}
	objType := os.Getenv(probeObjTypeEnv)
	if objType == "" {
		t.Logf("ORACLE : %s absent — horloge du match disponible, mais AUCUN evenement nomme"+
			" (une famille devinee fabrique des evenements : lecon du lot C).", probeObjTypeEnv)
		return hor, nil
	}
	all := objectiveevents.NamedEvents(src, objType)
	var evs []objectiveevents.NamedEvent
	combat := 0
	for _, e := range all {
		if e.Stat == objectiveevents.StatKills || e.Stat == objectiveevents.StatAssists {
			combat++
			continue
		}
		evs = append(evs, e)
	}
	t.Logf("ORACLE : famille %q · %d evenements d'objectif retenus · %d evenements de combat"+
		" ecartes (la question de F2 porte sur les CAPTURES, pas sur les frags)",
		objType, len(evs), combat)
	return hor, evs
}

// ---------------------------------------------------------------- F2 : ti=47 splash-message

// sondeF2 publie les valeurs des deux composants portes de ti=47, leurs instants, et leur
// densite autour des evenements d'objectif.
func sondeF2(t *testing.T, dir string, m *probeMoisson, evs []objectiveevents.NamedEvent) {
	t.Helper()
	t.Logf("=== F2 — ti=47 splash-message : QUELLES VALEURS, ET QUAND ===")
	for _, comp := range []ProbeComponent{ProbeSplashStatic, ProbeSplashDynamic} {
		es := probeFiltre(m.emissions, comp)
		if len(es) == 0 {
			t.Logf("  %s : AUCUNE emission sur ce film.", comp)
			continue
		}
		vals := probeCompteValeurs(es)
		t.Logf("  %s : %d emissions · %d valeurs distinctes", comp, len(es), len(vals))
		probeJournaliseValeurs(t, vals, len(es), 12)
		probeStructureValeurs(t, es)
		probeDensiteParValeur(t, vals, es, evs)
		probeEcrisValeursTSV(t, dir, fmt.Sprintf("F2_%d", comp), vals, len(es))
	}
}

// probeStructureValeurs cherche une STRUCTURE dans la suite des valeurs : un identifiant de
// message est un petit alphabet tire au hasard dans le temps, une horloge est une suite
// croissante a pas regulier. Les deux se distinguent sans aucune retro-ingenierie.
func probeStructureValeurs(t *testing.T, es []probeEmission) {
	t.Helper()
	if len(es) < 4 {
		return
	}
	croissant, pas := 0, make([]int64, 0, len(es))
	for i := 1; i < len(es); i++ {
		d := int64(es[i].vals[0]) - int64(es[i-1].vals[0])
		if d > 0 {
			croissant++
			pas = append(pas, d)
		}
	}
	n := float64(len(es) - 1)
	// Correlation entre la valeur et l'instant du paquet : c'est le test direct de « c'est une
	// horloge », et il ne depend d'aucune hypothese sur le pas.
	r := probeCorrelation(es)
	msg := fmt.Sprintf("  structure : %.1f %% de transitions croissantes · correlation"+
		" valeur/instant r = %.3f", 100*float64(croissant)/n, r)
	if len(pas) > 0 {
		sort.Slice(pas, func(i, j int) bool { return pas[i] < pas[j] })
		msg += fmt.Sprintf(" · pas median %d", pas[len(pas)/2])
	}
	t.Logf("%s", msg)
	switch {
	case r >= 0.90:
		t.Logf("    -> la valeur SUIT LE TEMPS : c'est une horloge ou un compteur, pas un"+
			" identifiant de message (r = %.3f).", r)
	case r <= 0.30 && r >= -0.30:
		t.Logf("    -> aucune derive temporelle : compatible avec un identifiant.")
	}
}

// probeCorrelation rend la correlation de Pearson entre la valeur et l'instant du paquet.
func probeCorrelation(es []probeEmission) float64 {
	n := float64(len(es))
	var sx, sy, sxx, syy, sxy float64
	base := es[0].usPk
	for _, e := range es {
		x := float64(e.usPk-base) / 1e6
		y := float64(e.vals[0])
		sx, sy, sxx, syy, sxy = sx+x, sy+y, sxx+x*x, syy+y*y, sxy+x*y
	}
	varX, varY := n*sxx-sx*sx, n*syy-sy*sy
	if varX <= 0 || varY <= 0 {
		return 0
	}
	return (n*sxy - sx*sy) / math.Sqrt(varX*varY)
}

// probeDensiteParValeur dit, pour chaque valeur, combien de ses emissions tombent dans une
// fenetre de +/- 2 s autour d'un evenement d'objectif, et compare cette densite a celle du
// reste du match. C'est la question de F2 : ces valeurs sont-elles les messages plein ecran ?
func probeDensiteParValeur(t *testing.T, vals []probeValeur, es []probeEmission,
	evs []objectiveevents.NamedEvent) {
	t.Helper()
	if len(evs) == 0 {
		t.Logf("    (pas d'evenement d'objectif fourni : densite non mesurable)")
		return
	}
	instants := make([]int, 0, len(evs))
	for _, e := range evs {
		instants = append(instants, e.TimeMS)
	}
	sort.Ints(instants)
	dansFenetre, hors, sansHorloge := 0, 0, 0
	parValeur := map[uint64][2]int{}
	for _, e := range es {
		if e.tMS < 0 {
			sansHorloge++
			continue
		}
		k := probeCle(e.vals)
		c := parValeur[k]
		if probeProche(instants, e.tMS) {
			dansFenetre++
			c[0]++
		} else {
			hors++
			c[1]++
		}
		parValeur[k] = c
	}
	secIn, secOut := probeSecondes(instants, es)
	t.Logf("    fenetre +/- %d ms : %d emissions dedans, %d dehors, %d sans horloge"+
		" · %.1f s en fenetre contre %.1f s hors", probeFenetreMS, dansFenetre, hors,
		sansHorloge, secIn, secOut)
	if secIn <= 0 || secOut <= 0 {
		return
	}
	t.Logf("    densite par valeur (dedans/s ÷ dehors/s ; >1 = suit les evenements) :")
	for _, v := range vals[:minI(len(vals), 12)] {
		c := parValeur[v.cle]
		d := probeDensite(c[0], c[1], secIn, secOut)
		jamais := ""
		if c[0] == 0 {
			jamais = "  JAMAIS pres d'un evenement"
		}
		t.Logf("      %-24s %6d emissions · %4d dedans · %6d dehors · %s%s",
			v.libelle, v.n, c[0], c[1], d, jamais)
	}
}

// ---------------------------------------------------------------- F3 : ti=4 high-frequency

// sondeF3 caracterise le R(8) de ti=4 : sa distribution, sa periodicite, et sa loi de succession.
func sondeF3(t *testing.T, m *probeMoisson) {
	t.Helper()
	t.Logf("=== F3 — ti=4 high-frequency : QUE PORTE SON R(8) ? ===")
	es := probeFiltre(m.emissions, ProbeHighFrequency)
	if len(es) == 0 {
		t.Logf("  AUCUNE emission sur ce film.")
		return
	}
	vals := probeCompteValeurs(es)
	t.Logf("  %d emissions · %d valeurs distinctes sur les 256 possibles", len(es), len(vals))
	probeJournaliseValeurs(t, vals, len(es), 10)

	// Loi de succession : un compteur modulo 256 se voit a ce que la valeur suivante vaille la
	// precedente + 1. Un drapeau ne prend que deux valeurs. Une donnee quelconque ne fait ni
	// l'un ni l'autre.
	inc, egal, autre := 0, 0, 0
	for i := 1; i < len(es); i++ {
		a, b := es[i-1].vals[0], es[i].vals[0]
		switch {
		case (a+1)%256 == b:
			inc++
		case a == b:
			egal++
		default:
			autre++
		}
	}
	n := float64(maxI(1, len(es)-1))
	t.Logf("  succession : +1 modulo 256 dans %.1f %% des cas · identique %.1f %% · autre %.1f %%",
		100*float64(inc)/n, 100*float64(egal)/n, 100*float64(autre)/n)

	// Periodicite : l'ecart entre deux emissions consecutives, sur l'horloge du film.
	var dts []int64
	for i := 1; i < len(es); i++ {
		if d := int64(es[i].usPk) - int64(es[i-1].usPk); d >= 0 {
			dts = append(dts, d/1000)
		}
	}
	probeJournaliseDeltas(t, dts)
	switch {
	case float64(inc)/n >= 0.90:
		t.Logf("  VERDICT : COMPTEUR modulo 256 — un tic de simulation, incremente a chaque emission.")
	case len(vals) <= 2:
		t.Logf("  VERDICT : DRAPEAU (%d valeurs seulement).", len(vals))
	default:
		t.Logf("  VERDICT : ni compteur incremental (%.1f %%) ni drapeau (%d valeurs) —"+
			" voir la distribution et les ecarts.", 100*float64(inc)/n, len(vals))
	}
}

// ---------------------------------------------------------------- F4 : tacmap ti=34 / ti=30

// sondeF4 recense les deux archetypes tacmap et les confronte a leur fantome.
func sondeF4(t *testing.T, m *probeMoisson) {
	t.Helper()
	t.Logf("=== F4 — tacmap (ti=34 waypointstate, ti=30 poiicon) : PRESENTS EN MULTIJOUEUR ? ===")
	for _, r := range []probeRole{roleTacmapWayp, roleTacmapPOI} {
		a := m.archs[r]
		if a == nil {
			t.Logf("  %-38s archetype ABSENT du registre", probeRoleNoms[r].libelle)
			continue
		}
		st := m.stats[r]
		t.Logf("  %-38s ti=%-3d · %d slots en image-cle · %d records delta · fantome %d · rapport %s",
			probeRoleNoms[r].libelle, a.ti, len(a.slotsKF), st.records, st.ghostRecords,
			probeRapport(st.records, st.ghostRecords))
		t.Logf("      dont %d annoncent le composant tacmap identifiant (i%d) = %.3f %%"+
			" · %d hors grammaire = %.1f %%",
			st.avecIdentifiant, a.idxIdentifiant,
			100*float64(st.avecIdentifiant)/float64(maxI(1, st.records)),
			st.horsGrammaire, 100*float64(st.horsGrammaire)/float64(maxI(1, st.records)))
		if st.avecIdentifiant == 0 {
			t.Logf("      -> la bande existe, mais le composant tacmap n'est JAMAIS annonce :"+
				" rien a lire dans %s sur ce film.", probeRoleNoms[r].comp)
		}
	}
}

// ---------------------------------------------------------------- F5 : ti=13 property-name

// sondeF5 publie les valeurs du R(32) de ti=13 et teste sa coincidence avec ti=10.
func sondeF5(t *testing.T, m *probeMoisson) {
	t.Helper()
	t.Logf("=== F5 — ti=13 managed-object-property-name : QUELLES VALEURS, ET LIEES A ti=10 ? ===")
	es := probeFiltre(m.emissions, ProbeManagedObjectPropertyName)
	if len(es) == 0 {
		t.Logf("  AUCUNE emission sur ce film.")
	} else {
		vals := probeCompteValeurs(es)
		t.Logf("  %d emissions · %d valeurs distinctes de R(32)", len(es), len(vals))
		probeJournaliseValeurs(t, vals, len(es), 20)
	}
	// Coincidence avec ti=10 : deux archetypes qui decrivent le MEME objet scripte devraient
	// parler dans les memes paquets. On mesure la part des instants de ti=13 qui portent aussi
	// un record de ti=10, et le temoin est la part d'instants de ti=10 dans toute la passe.
	instTI13 := m.stats[rolePropName].instants
	instTI10 := m.stats[roleManagedObj].instants
	if len(instTI13) == 0 || len(instTI10) == 0 {
		t.Logf("  COINCIDENCE : non mesurable (ti=13 %d instants, ti=10 %d instants).",
			len(instTI13), len(instTI10))
		return
	}
	communs := 0
	for us := range instTI13 {
		if instTI10[us] {
			communs++
		}
	}
	part := float64(communs) / float64(len(instTI13))
	base := float64(len(instTI10)) / float64(maxI(1, m.deltaPaquets))
	t.Logf("  COINCIDENCE : %d des %d instants de ti=13 portent aussi ti=10 = %.1f %%"+
		" · temoin (part des paquets delta portant ti=10) = %.1f %% · rapport %.2fx",
		communs, len(instTI13), 100*part, 100*base, part/math.Max(base, 1e-9))
	switch {
	case part >= 0.90 && part/math.Max(base, 1e-9) >= 1.5:
		t.Logf("  VERDICT : ti=13 parle QUAND ti=10 parle — meme objet scripte, deux canaux.")
	case part/math.Max(base, 1e-9) <= 1.1:
		t.Logf("  VERDICT : aucune coincidence au-dela du hasard — les deux canaux sont independants.")
	default:
		t.Logf("  VERDICT : coincidence PARTIELLE, ni fortuite ni systematique.")
	}
}

// ---------------------------------------------------------------- outillage commun

// probeValeur est une valeur observee et son compte.
type probeValeur struct {
	cle     uint64
	libelle string
	n       int
}

func probeFiltre(es []probeEmission, comp ProbeComponent) []probeEmission {
	var out []probeEmission
	for _, e := range es {
		if e.comp == comp {
			out = append(out, e)
		}
	}
	return out
}

// probeCle reduit un n-uplet de valeurs a une cle : les composants sondes n'en portent qu'une,
// mais la signature du hook en admet plusieurs et une cle qui ignorerait les suivantes
// confondrait deux emissions differentes.
func probeCle(vals []uint64) uint64 {
	var k uint64 = 1469598103934665603
	for _, v := range vals {
		k = (k ^ v) * 1099511628211
	}
	return k
}

func probeLibelle(vals []uint64) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, "/")
}

// probeCompteValeurs agrege les emissions par valeur, triees par frequence decroissante.
func probeCompteValeurs(es []probeEmission) []probeValeur {
	n := map[uint64]int{}
	lib := map[uint64]string{}
	for _, e := range es {
		k := probeCle(e.vals)
		n[k]++
		lib[k] = probeLibelle(e.vals)
	}
	out := make([]probeValeur, 0, len(n))
	for k, c := range n {
		out = append(out, probeValeur{cle: k, libelle: lib[k], n: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].n != out[j].n {
			return out[i].n > out[j].n
		}
		return out[i].libelle < out[j].libelle
	})
	return out
}

func probeJournaliseValeurs(t *testing.T, vals []probeValeur, total, top int) {
	t.Helper()
	for i, v := range vals {
		if i >= top {
			t.Logf("      ... et %d autres valeurs", len(vals)-top)
			break
		}
		t.Logf("      %-24s %7d  %5.2f %%", v.libelle, v.n, 100*float64(v.n)/float64(maxI(1, total)))
	}
}

func probeJournaliseDeltas(t *testing.T, dts []int64) {
	t.Helper()
	if len(dts) == 0 {
		return
	}
	sort.Slice(dts, func(i, j int) bool { return dts[i] < dts[j] })
	classes := map[int64]int{}
	for _, d := range dts {
		classes[d]++
	}
	keys := make([]int64, 0, len(classes))
	for k := range classes {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return classes[keys[i]] > classes[keys[j]] })
	t.Logf("  ecarts entre emissions (ms) : mediane %d · p95 %d · max %d · %d valeurs distinctes",
		dts[len(dts)/2], dts[minI(len(dts)-1, 95*len(dts)/100)], dts[len(dts)-1], len(classes))
	for i, k := range keys {
		if i >= 6 {
			break
		}
		t.Logf("      ecart %4d ms : %7d fois (%5.2f %%)", k, classes[k],
			100*float64(classes[k])/float64(len(dts)))
	}
}

// probeProche dit si un instant tombe dans la fenetre d'un evenement.
func probeProche(instants []int, tMS int) bool {
	i := sort.SearchInts(instants, tMS-probeFenetreMS)
	return i < len(instants) && instants[i] <= tMS+probeFenetreMS
}

// probeSecondes rend le nombre de secondes couvertes par les fenetres, et le reste du match. Les
// fenetres se RECOUVRENT : les additionner sans les fusionner gonflerait le denominateur et
// ferait passer un canal quelconque pour un canal concentre.
func probeSecondes(instants []int, es []probeEmission) (float64, float64) {
	if len(instants) == 0 || len(es) == 0 {
		return 0, 0
	}
	lo, hi := math.MaxInt, math.MinInt
	for _, e := range es {
		if e.tMS < 0 {
			continue
		}
		lo, hi = minI(lo, e.tMS), maxI(hi, e.tMS)
	}
	if lo > hi {
		return 0, 0
	}
	var couvert int
	debut, fin := instants[0]-probeFenetreMS, instants[0]+probeFenetreMS
	for _, t := range instants[1:] {
		if t-probeFenetreMS > fin {
			couvert += fin - debut
			debut, fin = t-probeFenetreMS, t+probeFenetreMS
			continue
		}
		fin = t + probeFenetreMS
	}
	couvert += fin - debut
	total := hi - lo
	return float64(couvert) / 1000, float64(maxI(0, total-couvert)) / 1000
}

func probeDensite(dedans, dehors int, secIn, secOut float64) string {
	if dehors == 0 {
		if dedans == 0 {
			return "—"
		}
		return "hors fenetre nul"
	}
	d := (float64(dedans) / secIn) / (float64(dehors) / secOut)
	return fmt.Sprintf("%.2fx", d)
}

func probeRapport(a, b int) string {
	if b == 0 {
		if a == 0 {
			return "0/0"
		}
		return "fantome NUL"
	}
	return fmt.Sprintf("%.2fx", float64(a)/float64(b))
}

// probeEcrisValeursTSV depose la table complete des valeurs d'un composant.
func probeEcrisValeursTSV(t *testing.T, dir, tag string, vals []probeValeur, total int) {
	t.Helper()
	out := os.Getenv(probeTSVEnv)
	if out == "" {
		return
	}
	var b strings.Builder
	b.WriteString("valeur\tcompte\tpart\n")
	for _, v := range vals {
		fmt.Fprintf(&b, "%s\t%d\t%.6f\n", v.libelle, v.n, float64(v.n)/float64(maxI(1, total)))
	}
	path := filepath.Join(out, filepath.Base(dir)+"_"+tag+"_valeurs.tsv")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("ecriture TSV : %v", err)
	}
	t.Logf("    TSV : %s", path)
}

func minI(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}
