package replay

// ground_weapon_pads_cluster_test.go — LA REGLE DE GRAPPE ET LA REGLE DE CYCLE, isolees de
// tout film pour qu'elles soient TESTEES et non seulement executees.
//
// POURQUOI ELLES SONT ICI ET PAS DANS L'INSTRUMENT. Les deux instruments de la phase 1 s'en
// servent — celui qui lit UN film (`ground_weapon_pads_research_test.go`) et celui qui compare
// DEUX films de la meme carte (`ground_weapon_pads_aggregate_test.go`). Deux copies d'une regle
// de seuil re-divergent au premier correctif ; une seule, testee sans garde d'environnement,
// tombe sous le gate ordinaire du depot.
//
// CE N'EST PAS DU CODE DE PRODUCTION, et ca ne doit pas le devenir tant que le critere de
// catalogue du plan (item 1.4) n'a pas tranche : un socle publie dans le contrat de rejeu est
// une donnee de REFERENCE, et une donnee de reference qui n'est pas reproductible d'un film a
// l'autre n'a rien a faire dans un catalogue versionne.

import (
	"fmt"
	"math"
	"sort"
	"testing"
)

// gwPadCluster est une GRAPPE : des apparitions de meme nature et de meme famille, a moins de
// `gwPadRadiusM` les unes des autres. Sa position est le CENTROIDE de ses apparitions.
type gwPadCluster struct {
	Kind, Family string
	X, Y, Z      float32
	// TS porte les instants des apparitions, TRIES. C'est la matiere du cycle.
	TS []uint64
}

// gwPadCycle est le cycle mesure d'une grappe. `Established` est faux quand la mesure refuse de
// conclure — jamais un chiffre instable publie comme s'il etait stable (decision 3 du plan).
type gwPadCycle struct {
	MedianS, P10S, P90S, SDS float64
	Gaps                     int
	Established              bool
}

// gwPadsCluster agglomere les apparitions par (nature, famille) et proximite.
//
// LA REGLE EST « LE PLUS PROCHE DANS LE RAYON », pas « le premier trouve » : sur une carte
// d'arene deux socles voisins peuvent etre a deux metres l'un de l'autre, et le premier trouve
// les fusionnerait selon l'ordre d'arrivee. L'ordre d'entree est rendu TOTAL avant la marche
// (instant, puis position) pour que deux executions rendent les memes grappes.
func gwPadsCluster(app []gwPadApparition, radiusM float64) []gwPadCluster {
	out, _ := gwPadsClusterAssign(app, radiusM)
	return out
}

// gwPadsClusterAssign agglomere ET rend, pour chaque apparition D'ENTREE (meme index), la
// grappe qui la porte — ou -1 si l'entree n'a pas ete agglomeree.
//
// POURQUOI L'ASSIGNATION EXISTE. La phase 2 mesure le cycle DEPUIS LE RAMASSAGE : il lui faut,
// socle par socle, quel objet est apparu et quand il a ete ramasse. Retrouver la grappe d'une
// apparition apres coup en comparant sa position au centroide serait une SECONDE regle de
// grappe, qui divergerait de celle-ci au premier correctif. `gwPadsCluster` reste le cas
// d'usage sans assignation, et il n'y a toujours qu'une seule regle.
func gwPadsClusterAssign(app []gwPadApparition, radiusM float64) ([]gwPadCluster, []int) {
	ord := make([]int, len(app))
	for i := range ord {
		ord[i] = i
	}
	sort.Slice(ord, func(i, j int) bool { return gwPadsLess(app[ord[i]], app[ord[j]]) })
	assign := make([]int, len(app))
	var out []gwPadCluster
	for _, src := range ord {
		a := app[src]
		best, bestD := -1, math.Inf(1)
		for i := range out {
			if out[i].Kind != a.Kind || out[i].Family != a.Family {
				continue
			}
			d := gwPadsDist(out[i].X, out[i].Y, out[i].Z, a.X, a.Y, a.Z)
			if d <= radiusM && d < bestD {
				best, bestD = i, d
			}
		}
		if best < 0 {
			out = append(out, gwPadCluster{
				Kind: a.Kind, Family: a.Family, X: a.X, Y: a.Y, Z: a.Z, TS: []uint64{a.TUS},
			})
			assign[src] = len(out) - 1
			continue
		}
		n := float32(len(out[best].TS))
		out[best].X = (out[best].X*n + a.X) / (n + 1)
		out[best].Y = (out[best].Y*n + a.Y) / (n + 1)
		out[best].Z = (out[best].Z*n + a.Z) / (n + 1)
		out[best].TS = append(out[best].TS, a.TUS)
		assign[src] = best
	}
	for i := range out {
		sort.Slice(out[i].TS, func(a, b int) bool { return out[i].TS[a] < out[i].TS[b] })
	}
	// Les grappes sont rendues dans un ordre TOTAL (et non celui de leur decouverte) : les
	// index d'assignation deja distribues doivent suivre la permutation, sans quoi ils
	// designeraient une grappe voisine.
	perm := make([]int, len(out))
	for i := range perm {
		perm[i] = i
	}
	sort.Slice(perm, func(i, j int) bool { return gwPadsClusterLess(out[perm[i]], out[perm[j]]) })
	inv := make([]int, len(out))
	sorted := make([]gwPadCluster, len(out))
	for newIdx, oldIdx := range perm {
		inv[oldIdx] = newIdx
		sorted[newIdx] = out[oldIdx]
	}
	for i := range assign {
		assign[i] = inv[assign[i]]
	}
	return sorted, assign
}

// gwPadsLess est l'ordre TOTAL des apparitions : instant, puis nature, famille et position.
func gwPadsLess(a, b gwPadApparition) bool {
	switch {
	case a.TUS != b.TUS:
		return a.TUS < b.TUS
	case a.Kind != b.Kind:
		return a.Kind < b.Kind
	case a.Family != b.Family:
		return a.Family < b.Family
	case a.X != b.X:
		return a.X < b.X
	case a.Y != b.Y:
		return a.Y < b.Y
	}
	return a.Z < b.Z
}

// gwPadsClusterLess est l'ordre TOTAL des grappes rendues.
func gwPadsClusterLess(a, b gwPadCluster) bool {
	switch {
	case a.Kind != b.Kind:
		return a.Kind < b.Kind
	case a.Family != b.Family:
		return a.Family < b.Family
	case a.X != b.X:
		return a.X < b.X
	case a.Y != b.Y:
		return a.Y < b.Y
	}
	return a.Z < b.Z
}

// gwPadsKeep ne garde que les grappes d'au moins `min` apparitions — les SOCLES.
func gwPadsKeep(in []gwPadCluster, min int) []gwPadCluster {
	out := make([]gwPadCluster, 0, len(in))
	for _, c := range in {
		if len(c.TS) >= min {
			out = append(out, c)
		}
	}
	return out
}

// gwPadsCycle mesure le cycle d'une grappe : les ecarts entre apparitions successives.
//
// L'ECART EST PUBLIE DES QU'IL EST MESURE, MAIS « ETABLI » EXIGE DEUX ECARTS. Un socle vu deux
// fois donne UN intervalle : c'est une mesure, et la taire rendrait un `0,0 s` qui se lirait
// comme un cycle nul. Ce n'est pourtant pas un cycle — un intervalle unique n'a pas
// d'ecart-type, donc rien ne dit qu'il se repete. Les deux faits se publient separement :
// `MedianS` porte la mesure, `Established` porte le verdict (decision 3 du plan).
func gwPadsCycle(ts []uint64) gwPadCycle {
	if len(ts) < 2 {
		return gwPadCycle{}
	}
	gaps := make([]float64, 0, len(ts)-1)
	for i := 1; i < len(ts); i++ {
		gaps = append(gaps, float64(ts[i]-ts[i-1])/1e6)
	}
	return gwPadsCycleFromGaps(gaps)
}

// gwPadsCycleFromGaps juge un cycle a partir des ECARTS deja calcules. La phase 2 mesure des
// ecarts qui ne sont PAS des differences d'instants successifs (ramassage -> reapparition
// suivante) : elle passe donc par ici, et le verdict de stabilite reste ecrit une seule fois.
func gwPadsCycleFromGaps(gaps []float64) gwPadCycle {
	if len(gaps) == 0 {
		return gwPadCycle{}
	}
	c := gwPadCycle{
		Gaps:    len(gaps),
		MedianS: gwPadsQuantile(gaps, 0.5),
		P10S:    gwPadsQuantile(gaps, 0.10),
		P90S:    gwPadsQuantile(gaps, 0.90),
		SDS:     gwPadsStdDev(gaps),
	}
	c.Established = c.Gaps >= gwPadCycleMinGaps && c.MedianS > 0 &&
		c.SDS <= gwPadCycleMaxCV*c.MedianS
	return c
}

// gwPadsQuantile rend le quantile d'ordre q, par la MEME convention d'index que les autres
// instruments du paquet (`origineQ`) : deux mesures du depot qui disent « p90 » doivent dire
// la meme chose.
func gwPadsQuantile(v []float64, q float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return s[int(q*float64(len(s)-1))]
}

// gwPadsStdDev rend l'ecart-type de POPULATION — les ecarts mesures sont la population entiere
// du cycle de ce socle, pas un echantillon tire d'une population plus large.
func gwPadsStdDev(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	var sum float64
	for _, x := range v {
		sum += x
	}
	m := sum / float64(len(v))
	var acc float64
	for _, x := range v {
		acc += (x - m) * (x - m)
	}
	return math.Sqrt(acc / float64(len(v)))
}

// gwPadsSpread resume la taille des grappes — le temoin de Notion 11 se lit la : un socle
// recurre, une grappe d'une seule apparition est un lacher isole.
func gwPadsSpread(in []gwPadCluster) string {
	if len(in) == 0 {
		return "aucune grappe"
	}
	sizes := make([]float64, 0, len(in))
	singles, maxN := 0, 0
	for _, c := range in {
		sizes = append(sizes, float64(len(c.TS)))
		if len(c.TS) == 1 {
			singles++
		}
		if len(c.TS) > maxN {
			maxN = len(c.TS)
		}
	}
	return fmt.Sprintf("grappes %d · a une seule apparition %d · mediane %.0f · max %d",
		len(in), singles, gwPadsQuantile(sizes, 0.5), maxN)
}

// gwPadsCycleSummary compte les socles dont le cycle est ETABLI, et donne la mediane des
// medianes — la seule facon honnete de resumer des cycles qui n'ont pas tous la meme stabilite.
func gwPadsCycleSummary(socles []gwPadCluster) string {
	if len(socles) == 0 {
		return "aucun socle : aucun cycle"
	}
	var med []float64
	etablis := 0
	for _, p := range socles {
		c := gwPadsCycle(p.TS)
		if !c.Established {
			continue
		}
		etablis++
		med = append(med, c.MedianS)
	}
	if etablis == 0 {
		return fmt.Sprintf("socles %d · cycle ETABLI 0 · aucun cycle publiable", len(socles))
	}
	return fmt.Sprintf("socles %d · cycle ETABLI %d (%s) · mediane des medianes %.1f s",
		len(socles), etablis, gwPadsPart(etablis, len(socles)), gwPadsQuantile(med, 0.5))
}

// --- TESTS DE LA REGLE (sans garde : ils tournent avec le paquet) ------------------------

// TestGwPadsClusterSepareDeuxSoclesVoisins : la regle « le plus proche dans le rayon » ne doit
// pas fondre deux socles distants de plus du rayon, meme quand les apparitions alternent.
func TestGwPadsClusterSepareDeuxSoclesVoisins(t *testing.T) {
	app := []gwPadApparition{
		{Kind: gwPadKindWeapon, Family: "A", X: 0, TUS: 1},
		{Kind: gwPadKindWeapon, Family: "A", X: 3, TUS: 2},
		{Kind: gwPadKindWeapon, Family: "A", X: 0.4, TUS: 3},
		{Kind: gwPadKindWeapon, Family: "A", X: 3.3, TUS: 4},
	}
	got := gwPadsCluster(app, gwPadRadiusM)
	if len(got) != 2 {
		t.Fatalf("2 grappes attendues, %d obtenues : %+v", len(got), got)
	}
	for _, c := range got {
		if len(c.TS) != 2 {
			t.Fatalf("chaque grappe doit porter 2 apparitions, %d : %+v", len(c.TS), c)
		}
	}
}

// TestGwPadsClusterNeMelangePasLesFamilles : deux armes differentes au MEME endroit sont deux
// grappes — un socle est une famille a une position, jamais une position seule.
func TestGwPadsClusterNeMelangePasLesFamilles(t *testing.T) {
	app := []gwPadApparition{
		{Kind: gwPadKindWeapon, Family: "A", TUS: 1},
		{Kind: gwPadKindWeapon, Family: "B", TUS: 2},
		{Kind: gwPadKindPowerup, Family: "A", TUS: 3},
	}
	if got := gwPadsCluster(app, gwPadRadiusM); len(got) != 3 {
		t.Fatalf("3 grappes attendues (2 familles + 1 nature), %d obtenues : %+v", len(got), got)
	}
}

// TestGwPadsClusterAssignRendLaGrappeDeChaqueEntree : l'assignation doit designer la grappe
// RENDUE (apres l'ordre total), pas celle de l'ordre de decouverte — l'erreur silencieuse que
// la permutation evite.
func TestGwPadsClusterAssignRendLaGrappeDeChaqueEntree(t *testing.T) {
	app := []gwPadApparition{
		{Kind: gwPadKindWeapon, Family: "Z", X: 10, TUS: 1},
		{Kind: gwPadKindWeapon, Family: "A", X: 0, TUS: 2},
		{Kind: gwPadKindWeapon, Family: "Z", X: 10.3, TUS: 3},
	}
	pads, assign := gwPadsClusterAssign(app, gwPadRadiusM)
	if len(pads) != 2 || len(assign) != 3 {
		t.Fatalf("2 grappes et 3 assignations attendues : %d / %d", len(pads), len(assign))
	}
	for i, a := range app {
		p := pads[assign[i]]
		if p.Family != a.Family {
			t.Fatalf("apparition %d (%s) assignee a la grappe %q", i, a.Family, p.Family)
		}
		if gwPadsDist(p.X, p.Y, p.Z, a.X, a.Y, a.Z) > gwPadRadiusM {
			t.Fatalf("apparition %d assignee a une grappe hors du rayon : %+v", i, p)
		}
	}
}

// TestGwPadsCycleRefuseDeConclureSurUnCycleInstable : le seuil de la decision 3 ne se contourne
// pas — un cycle disperse se publie « non etabli », pas arrondi.
func TestGwPadsCycleRefuseDeConclureSurUnCycleInstable(t *testing.T) {
	stable := gwPadsCycle([]uint64{0, 100_000_000, 200_000_000, 300_000_000})
	if !stable.Established || math.Abs(stable.MedianS-100) > 0.01 {
		t.Fatalf("cycle regulier de 100 s attendu etabli : %+v", stable)
	}
	instable := gwPadsCycle([]uint64{0, 10_000_000, 200_000_000, 205_000_000})
	if instable.Established {
		t.Fatalf("cycle disperse ne doit PAS etre etabli : %+v", instable)
	}
	court := gwPadsCycle([]uint64{0, 100_000_000})
	if court.Established || court.Gaps != 1 || math.Abs(court.MedianS-100) > 0.01 {
		t.Fatalf("un ecart unique se MESURE (100 s) mais n'ETABLIT rien : %+v", court)
	}
	if vide := gwPadsCycle([]uint64{42}); vide.Gaps != 0 || vide.MedianS != 0 {
		t.Fatalf("une apparition unique n'a aucun ecart : %+v", vide)
	}
}
