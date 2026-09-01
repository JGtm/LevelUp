package replay

// assaut_a6_armement_test.go — LA CHASSE A L'ARMEMENT DE LA BOMBE, dans les canaux jamais lus.
//
// # L'hypothese, et d'ou elle vient
//
// Le releve A0.3 (fige le 2026-08-27) conclut : « l'EXPLOSION se date, l'ARMEMENT n'a aucun
// increment propre ». Mais ce balayage n'avait regarde que DEUX canaux par composant — A et B.
// La grammaire en porte QUATRE : deux inconditionnels, puis deux drapeaux commandant chacun une
// valeur. `decodeStatComponent` lisait ces deux dernieres pour avancer le curseur et les JETAIT.
// A 28 composants, ce sont 56 emplacements que rien n'avait jamais vus.
//
// L'utilisateur, le 2026-08-31 : « pour l'armement a mon avis ca doit etre dans le statborg ».
// Les canaux C et D sont depuis ce jour decodes ([objectiveevents.StatValue]) ; cet instrument
// les balaie.
//
// # LE CRITERE, ecrit AVANT la mesure, et il se valide LUI-MEME
//
// Un compteur d'armement n'est pas reconnaissable a son ampleur — il vaut 1, 2 ou 3 comme dix
// autres compteurs. Il est reconnaissable a sa POSITION DANS LE TEMPS : chaque armement PRECEDE
// une explosion, d'un delai a peu pres CONSTANT (la meche est une constante moteur). Un
// emplacement candidat doit donc, sur les 9 films :
//
//	COUVERTURE   au moins une progression datee avant CHAQUE explosion connue du releve A0.3 ;
//	CONSTANCE    les delais (explosion - progression) resserres — dispersion <= 20 % de la
//	             mediane, le meme critere que les cycles de socle (`gwPadCycleMaxCV`) ;
//	SENS         le delai POSITIF (l'armement precede) et sous 120 s — au-dela ce n'est plus
//	             une meche, c'est une coincidence.
//
// Un emplacement qui tient les trois DONNE la meche par sa mediane. Aucun oracle exterieur n'est
// requis, et c'est ce qui rend la mesure possible : l'API ne publie RIEN pour l'Assaut.
//
// CE QUE CET INSTRUMENT NE PEUT PAS FAIRE, dit avant de lire son verdict : il cherche un
// COMPTEUR qui progresse. Si le moteur porte l'armement autrement — un booleen d'etat, une
// duree qui decompte — la progression n'existe pas et le balayage rendra zero candidat sans que
// cela prouve l'absence de la donnee. L'echec se lit donc « pas un compteur », jamais « pas la ».
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/replay/ -run AssautA6Armement -v -timeout 40m

import (
	"context"
	"fmt"
	"math"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

const (
	// a6MecheMaxMS : au-dela, un ecart n'est plus une meche.
	a6MecheMaxMS = 120_000
	// a6CVMax : dispersion maximale des delais, en part de la mediane. Recopie du critere de
	// cycle des socles d'arme (`gwPadCycleMaxCV`) — meme question, meme forme de reponse.
	a6CVMax = 0.20
	// a6ValeurMax : borne de domaine d'un compteur de geste. Un canal qui porte des dizaines de
	// milliers n'est pas un compteur d'armement (c'est un score, une duree, un identifiant).
	a6ValeurMax = 1000
)

// a6Canal designe un canal d'un composant.
type a6Canal struct {
	comp  int
	canal string // "A" | "B" | "C" | "D"
}

func (c a6Canal) String() string { return fmt.Sprintf("comp %2d %s", c.comp, c.canal) }

// a6Valeur rend la valeur d'un canal, et si elle est PRESENTE. La distinction compte : un canal
// conditionnel ABSENT et un canal a zero sont deux choses differentes.
func a6Valeur(v objectiveevents.StatValue, canal string) (int64, bool) {
	switch canal {
	case "A":
		return v.A, true
	case "B":
		return v.B, true
	case "C":
		return v.C, v.HasC
	default:
		return v.D, v.HasD
	}
}

// TestAssautA6Armement balaie les quatre canaux des 28 composants sur les 9 films d'Assaut.
func TestAssautA6Armement(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautA6Armement")()

	films := make([]string, 0, len(a5Explosions))
	for id := range a5Explosions {
		films = append(films, id)
	}
	sort.Strings(films)

	delais := map[a6Canal][]float64{}
	couverts := map[a6Canal]int{}
	presents := map[a6Canal]int{}
	totalExplosions := 0

	for _, id := range films {
		src, ok, err := filmcache.Open(cache, id)
		if err != nil || !ok {
			t.Fatalf("film %s absent du cache : %v", id, err)
		}
		recs, _ := objectiveevents.StatRecordsCtx(context.Background(), src, id)
		exps := a5Explosions[id]
		totalExplosions += len(exps)

		for comp := 0; comp <= 27; comp++ {
			for _, canal := range []string{"A", "B", "C", "D"} {
				k := a6Canal{comp: comp, canal: canal}
				prog := a6Progressions(recs, comp, canal)
				if len(prog) == 0 {
					continue
				}
				presents[k]++
				for _, ms := range exps {
					meilleur := -1
					for _, p := range prog {
						d := ms - p
						if d > 0 && d <= a6MecheMaxMS && (meilleur < 0 || d < meilleur) {
							meilleur = d
						}
					}
					if meilleur >= 0 {
						couverts[k]++
						delais[k] = append(delais[k], float64(meilleur))
					}
				}
			}
		}
	}

	type verdict struct {
		k       a6Canal
		couvert int
		mediane float64
		cv      float64
	}
	var tenus []verdict
	for k, ds := range delais {
		if couverts[k] < totalExplosions {
			continue // COUVERTURE : il faut une progression avant CHAQUE explosion
		}
		med, cv := a6MedianeEtCV(ds)
		if cv > a6CVMax {
			continue
		}
		tenus = append(tenus, verdict{k, couverts[k], med, cv})
	}
	sort.Slice(tenus, func(i, j int) bool { return tenus[i].cv < tenus[j].cv })

	t.Logf("BALAYAGE : %d explosions sur %d films, %d canaux porteurs de progressions",
		totalExplosions, len(films), len(presents))
	if len(tenus) == 0 {
		t.Logf("AUCUN CANAL NE TIENT LES TROIS CRITERES. Les meilleures couvertures, pour dire " +
			"QUELLE distance il reste :")
		a6MeilleuresCouvertures(t, couverts, delais, totalExplosions)
		return
	}
	for _, v := range tenus {
		t.Logf("CANDIDAT %s : %d/%d explosions couvertes, meche mediane %.1f s, dispersion %.0f %%",
			v.k, v.couvert, totalExplosions, v.mediane/1000, v.cv*100)
	}
}

// a6Progressions rend les instants ou le canal PROGRESSE, sur les slots de JOUEUR. Un armement
// est un geste de joueur ; les slots d'equipe portent des totaux, pas des gestes.
func a6Progressions(recs []objectiveevents.StatRecord, comp int, canal string) []int {
	type cle struct{ slot, round int }
	dernier := map[cle]int64{}
	vus := map[cle]bool{}
	var out []int
	for _, r := range recs {
		if objectiveevents.IsTeamSlot(r.Slot) {
			continue
		}
		v, ok := r.Comps[comp]
		if !ok {
			continue
		}
		val, present := a6Valeur(v, canal)
		if !present || val < 0 || val > a6ValeurMax {
			continue
		}
		k := cle{r.Slot, r.Round}
		if vus[k] && val > dernier[k] {
			out = append(out, r.TimeMS)
		}
		dernier[k], vus[k] = val, true
	}
	sort.Ints(out)
	return out
}

// a6MedianeEtCV rend la mediane et la dispersion relative d'une serie.
func a6MedianeEtCV(xs []float64) (float64, float64) {
	if len(xs) == 0 {
		return 0, math.Inf(1)
	}
	tri := append([]float64(nil), xs...)
	sort.Float64s(tri)
	med := tri[len(tri)/2]
	if med == 0 {
		return med, math.Inf(1)
	}
	var somme float64
	for _, x := range xs {
		somme += (x - med) * (x - med)
	}
	return med, math.Sqrt(somme/float64(len(xs))) / med
}

// a6MeilleuresCouvertures imprime les canaux les plus proches du critere — pour que l'echec dise
// QUELLE distance il reste, et pas seulement « non ».
func a6MeilleuresCouvertures(t *testing.T, couverts map[a6Canal]int, delais map[a6Canal][]float64, total int) {
	t.Helper()
	type ligne struct {
		k       a6Canal
		couvert int
		med, cv float64
	}
	ls := make([]ligne, 0, len(couverts))
	for k, n := range couverts {
		med, cv := a6MedianeEtCV(delais[k])
		ls = append(ls, ligne{k, n, med, cv})
	}
	sort.Slice(ls, func(i, j int) bool {
		if ls[i].couvert != ls[j].couvert {
			return ls[i].couvert > ls[j].couvert
		}
		return ls[i].cv < ls[j].cv
	})
	for i, l := range ls {
		if i >= 12 {
			break
		}
		t.Logf("  %s : %d/%d couvertes, mediane %.1f s, dispersion %.0f %%",
			l.k, l.couvert, total, l.med/1000, l.cv*100)
	}
}

// a6FenetreMecheMS : la fenetre avant une explosion ou l'on cherche une minuterie.
const a6FenetreMecheMS = 60_000

// a6RMin : correlation minimale pour tenir une minuterie. 0,95 est severe a dessein — une
// minuterie EST une droite, pas une tendance.
const a6RMin = 0.95

// TestAssautA6Minuterie — LA SECONDE FORME, et c'est celle que le premier balayage ne pouvait
// pas voir.
//
// [TestAssautA6Armement] cherche un COMPTEUR qui progresse : il rend zero candidat. Mais un
// armement peut se porter autrement, et l'utilisateur l'a nomme lui-meme — « quand la bombe
// declenche son COMPTEUR avant d'exploser ». Un compte a rebours n'est pas un compteur de
// gestes : c'est une valeur qui DECROIT vers l'explosion.
//
// LE CRITERE, ecrit avant la mesure : dans les 60 s qui precedent une explosion, un canal de
// minuterie doit etre une DROITE en fonction du temps restant — correlation >= 0,95 en valeur
// absolue. Une tendance ne suffit pas ; une minuterie est lineaire par construction.
//
// Les slots d'EQUIPE entrent ici, contrairement au balayage des compteurs : une meche appartient
// a la bombe, donc au match, pas au joueur qui l'a posee.
func TestAssautA6Minuterie(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautA6Minuterie")()

	films := make([]string, 0, len(a5Explosions))
	for id := range a5Explosions {
		films = append(films, id)
	}
	sort.Strings(films)

	// rs[canal] = les correlations mesurees, une par (explosion, slot) exploitable.
	rs := map[a6Canal][]float64{}
	for _, id := range films {
		src, ok, err := filmcache.Open(cache, id)
		if err != nil || !ok {
			t.Fatalf("film %s absent du cache : %v", id, err)
		}
		recs, _ := objectiveevents.StatRecordsCtx(context.Background(), src, id)
		for _, T := range a5Explosions[id] {
			for comp := 0; comp <= 27; comp++ {
				for _, canal := range []string{"A", "B", "C", "D"} {
					k := a6Canal{comp: comp, canal: canal}
					for _, r := range a6CorrelationsParSlot(recs, comp, canal, T) {
						rs[k] = append(rs[k], r)
					}
				}
			}
		}
	}

	type ligne struct {
		k        a6Canal
		n        int
		moyenneR float64
		fort     int
	}
	ls := make([]ligne, 0, len(rs))
	for k, v := range rs {
		var somme float64
		fort := 0
		for _, r := range v {
			somme += math.Abs(r)
			if math.Abs(r) >= a6RMin {
				fort++
			}
		}
		ls = append(ls, ligne{k, len(v), somme / float64(len(v)), fort})
	}
	sort.Slice(ls, func(i, j int) bool {
		if ls[i].fort != ls[j].fort {
			return ls[i].fort > ls[j].fort
		}
		return ls[i].moyenneR > ls[j].moyenneR
	})

	t.Logf("MINUTERIE : %d canaux mesurables dans les %d s avant une explosion",
		len(ls), a6FenetreMecheMS/1000)
	for i, l := range ls {
		if i >= 12 {
			break
		}
		t.Logf("  %s : %d serie(s), |r| moyen %.2f, %d serie(s) au-dela de %.2f",
			l.k, l.n, l.moyenneR, l.fort, a6RMin)
	}
}

// a6CorrelationsParSlot rend, pour chaque slot ayant au moins 4 emissions dans la fenetre, la
// correlation entre la valeur du canal et le TEMPS RESTANT avant l'explosion.
func a6CorrelationsParSlot(recs []objectiveevents.StatRecord, comp int, canal string, T int) []float64 {
	parSlot := map[int][][2]float64{}
	for _, r := range recs {
		if r.TimeMS >= T || r.TimeMS < T-a6FenetreMecheMS {
			continue
		}
		v, ok := r.Comps[comp]
		if !ok {
			continue
		}
		val, present := a6Valeur(v, canal)
		if !present {
			continue
		}
		parSlot[r.Slot] = append(parSlot[r.Slot], [2]float64{float64(T - r.TimeMS), float64(val)})
	}
	var out []float64
	for _, pts := range parSlot {
		if len(pts) < 4 {
			continue
		}
		if r, ok := a6Pearson(pts); ok {
			out = append(out, r)
		}
	}
	return out
}

// a6Pearson rend la correlation lineaire d'un nuage, et faux si l'une des deux series est
// constante (une valeur qui ne bouge pas n'est pas une minuterie).
func a6Pearson(pts [][2]float64) (float64, bool) {
	n := float64(len(pts))
	var sx, sy float64
	for _, p := range pts {
		sx += p[0]
		sy += p[1]
	}
	mx, my := sx/n, sy/n
	var sxy, sxx, syy float64
	for _, p := range pts {
		dx, dy := p[0]-mx, p[1]-my
		sxy += dx * dy
		sxx += dx * dx
		syy += dy * dy
	}
	if sxx == 0 || syy == 0 {
		return 0, false
	}
	return sxy / math.Sqrt(sxx*syy), true
}

// a6DecalageTemoinMS : de combien on RECULE les instants pour fabriquer le temoin. Trois minutes
// tombent en plein jeu sur tous les films du corpus, loin de toute explosion.
const a6DecalageTemoinMS = 180_000

// TestAssautA6TemoinMinuterie — LE TEMOIN SANS LEQUEL LE NEGATIF NE VAUT RIEN.
//
// [TestAssautA6Minuterie] rend des correlations elevees partout : |r| moyen de 0,72 a 0,95 sur
// les douze meilleurs canaux. C'est ATTENDU et ce n'est pas un resultat — un compteur monotone
// (frags, score personnel) correle mecaniquement avec le temps ecoule, quelle que soit la
// fenetre choisie. Sans temoin, on lirait cette correlation comme un indice.
//
// Le temoin rejoue EXACTEMENT la meme mesure sur des instants QUI NE SONT PAS des explosions :
// les memes, recules de trois minutes. Si les deux passes se ressemblent, la sonde ne distingue
// rien et le negatif se lit « la mesure ne sait pas repondre », pas « l'armement n'est pas la ».
// Si la passe reelle se detache, il y a quelque chose a creuser.
func TestAssautA6TemoinMinuterie(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautA6TemoinMinuterie")()

	films := make([]string, 0, len(a5Explosions))
	for id := range a5Explosions {
		films = append(films, id)
	}
	sort.Strings(films)

	reel := map[a6Canal][]float64{}
	temoin := map[a6Canal][]float64{}
	for _, id := range films {
		src, ok, err := filmcache.Open(cache, id)
		if err != nil || !ok {
			t.Fatalf("film %s absent du cache : %v", id, err)
		}
		recs, _ := objectiveevents.StatRecordsCtx(context.Background(), src, id)
		for _, T := range a5Explosions[id] {
			faux := T - a6DecalageTemoinMS
			for comp := 0; comp <= 27; comp++ {
				for _, canal := range []string{"A", "B", "C", "D"} {
					k := a6Canal{comp: comp, canal: canal}
					reel[k] = append(reel[k], a6CorrelationsParSlot(recs, comp, canal, T)...)
					if faux > a6FenetreMecheMS {
						temoin[k] = append(temoin[k], a6CorrelationsParSlot(recs, comp, canal, faux)...)
					}
				}
			}
		}
	}

	fortsDe := func(m map[a6Canal][]float64) (int, int) {
		forts, total := 0, 0
		for _, v := range m {
			for _, r := range v {
				total++
				if math.Abs(r) >= a6RMin {
					forts++
				}
			}
		}
		return forts, total
	}
	fr, tr := fortsDe(reel)
	ft, tt := fortsDe(temoin)
	partR := 100 * float64(fr) / math.Max(float64(tr), 1)
	partT := 100 * float64(ft) / math.Max(float64(tt), 1)
	t.Logf("AVANT UNE EXPLOSION : %d series sur %d au-dela de |r| = %.2f  (%.1f %%)", fr, tr, a6RMin, partR)
	t.Logf("TEMOIN (-%d s, aucun evenement) : %d sur %d  (%.1f %%)", a6DecalageTemoinMS/1000, ft, tt, partT)
	if partT <= 0 {
		t.Logf("le temoin ne rend rien — il ne borne donc rien, et le negatif reste indecis")
		return
	}
	t.Logf("RAPPORT reel/temoin = %.2f — au-dessous de 1,5 la sonde ne distingue PAS une "+
		"explosion d'un instant quelconque, et le negatif se lit « la mesure ne sait pas "+
		"repondre », jamais « l'armement n'est pas dans le statborg »", partR/partT)
}
