package replay

// assaut_a10_jauge_test.go — LA JAUGE, UNE FOIS `ti=11` DECODE.
//
// # OU EN EST LA CHASSE
//
// Quatre canaux ont ete fouilles et temoines : les composants du statborg (phase A6), le pied de
// film (`th=10`), l'archetype `ti=13` (phase A7), et l'armement calcule cote client (phase A8).
// Le cinquieme est le bon, et le film le NOMME lui-meme : `managed-objective-progress-component`,
// `ti=11 i12`. Le recensement des masques du 2026-09-01 l'a trouve dans 265 records sur les neuf
// films d'Assaut ; le portage du 2026-09-01 (`components_managed_objective.go`, trente-trois
// composants sur trente-quatre) le rend lisible.
//
// Cet instrument est la premiere lecture de ce canal. Il ne suppose rien : il rend ce que le
// balayage sort, et confronte les montees de jauge aux explosions datees du releve A0.3.
//
// # LE VERDICT DE LA PREMIERE PASSE : LE CANAL N'EST PAS ENCORE LU JUSTE (2026-09-01)
//
// ZERO slot porte une montee de jauge, et `TestAssautA10Detail` dit pourquoi : la valeur de i12
// est CONSTANTE sur toute la duree d'un slot, les MEMES valeurs reviennent a l'identique dans
// trois matchs differents, et des slots consecutifs rendent des valeurs decalees d'un bit
// (0x04000003, 0x08000007, 0x1000000F). Une jauge de partie ne peut pas etre octet-pour-octet
// identique dans trois matchs : la fenetre est mal posee.
//
// CE QUI RESTE ACQUIS MALGRE CA : l'archetype est le bon (mesure du miroir), sa grammaire est
// portee, et le traverseur ne bute plus. Ce qui manque est un ORACLE DE LARGEUR — un ancrage par
// `WalkKeyframeRecords` dont la fin de marche tombe exactement sur l'en-tete suivant.
//
// # LES CRITERES, ecrits avant la mesure — LES MEMES QU'AUX PHASES A6 ET A7
//
//	COUVERTURE   au moins une montee de i12 avant CHAQUE explosion ;
//	CONSTANCE    dispersion des delais <= 20 % de la mediane ;
//	SENS         delai positif, sous 120 s.
//
// LE DECALAGE D'HORLOGE NE GENE PAS : `ti=11` est date sur l'horloge MOTEUR, les explosions sur
// celle du MANIFESTE. Un decalage constant ne change pas une dispersion — c'est la CONSTANCE qui
// designe le canal, le recalage vient apres.
//
// # LA PREMIERE PASSE A REFUTE LA VOIE DELTA, ET C'EST UN RESULTAT (2026-09-01)
//
// Le balayage ouvre deux voies. La voie DELTA rend 120 000 a 350 000 lectures par film, avec un
// chainage de 2,7 a 26 % et des valeurs de jauge uniformement reparties sur 32 bits (4 851
// valeurs distinctes pour 5 749 lectures, mediane a 2^31). Ce n'est pas une jauge, c'est du
// bruit : la bande d'ancrage de ti=11 compte jusqu'a 1 704 slots et attrape tout.
//
// Cet instrument ne garde donc que les lectures d IMAGE-CLE (`FromKeyframe`), ou l ancrage est
// structurel et la marche aboutit sur 98,4 % des records. Il REND quand meme le bilan des deux
// voies, parce qu un filtre qu on ne voit pas est un filtre qu on oublie.
//
// # LE TEMOIN, sans lequel un resultat ne vaut rien
//
// Un film de CTF (186 records `ti=11` au recensement) et un de KOTH (12 records, masques nuls).
// Si l'instrument sort autant de jauge la qu'en Assaut, ce n'est pas une jauge d'armement.
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/replay/ -run AssautA10 -v -timeout 60m

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// a10Temoins : les films d'autres modes, balayes avec le meme instrument.
var a10Temoins = []struct{ id, mode string }{
	{"cde26226", "TEMOIN CTF arene"},
	{"7f1bbf06", "TEMOIN KOTH arene"},
	{"696a9d7c", "TEMOIN Strongholds arene"},
}

// a10Serie est la suite datee des valeurs d'un champ pour UN slot d'objectif.
type a10Serie struct {
	ts  []uint64
	val []uint64
}

// TestAssautA10Jauge balaie `ti=11` sur les neuf films d'Assaut et sur trois temoins.
func TestAssautA10Jauge(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautA10Jauge")()

	for _, w := range a10Temoins {
		a10Ligne(t, cache, w.id, w.mode)
	}
	films := make([]string, 0, len(a5Explosions))
	for id := range a5Explosions {
		films = append(films, id)
	}
	sort.Strings(films)

	delais := map[uint32][]float64{}
	couverts := map[uint32]int{}
	total := 0
	for _, id := range films {
		sc := a10Ligne(t, cache, id, "Assaut")
		if sc == nil {
			continue
		}
		montees := a10MonteesParSlot(*sc)
		exps := a5Explosions[id]
		total += len(exps)
		for slot, ts := range montees {
			for _, ms := range exps {
				if d := a10PlusProche(ts, ms); d >= 0 {
					couverts[slot]++
					delais[slot] = append(delais[slot], float64(d))
				}
			}
		}
	}

	t.Logf("########## %d explosion(s) sur %d films, %d slot(s) porteurs de montees de jauge",
		total, len(films), len(delais))
	a10Verdict(t, delais, couverts, total)
}

// a10Ligne balaie UN film, imprime son bilan, et rend le balayage (nil si illisible).
func a10Ligne(t *testing.T, cache, id, mode string) *filmdec.ObjectiveScan {
	t.Helper()
	sc, err := filmdec.ScanFilmObjectives(filepath.Join(cache, "film_chunks", id))
	if err != nil {
		t.Logf("%-9s %-26s balayage impossible (%v)", id, mode, err)
		return nil
	}
	cles := a10Cles(sc.Reads)
	t.Logf("%-9s %-26s IMAGE-CLE %d records (%d marches, %d cassees, chainage %s), %d lecture(s)"+
		"   |   DELTA %d records (%d marches, chainage %s), %d lecture(s) ECARTEES",
		id, mode, sc.KeyRecords, sc.KeyWalked, sc.KeyBroken, a10Pct(sc.KeyChained, sc.KeyWalked),
		len(cles), sc.Records, sc.Walked, a10Pct(sc.Chained, sc.Walked), len(sc.Reads)-len(cles))
	t.Logf("           champs (image-cle) : %s", a10ParChamp(cles))
	if j := a10Jauges(cles); len(j) > 0 {
		t.Logf("           jauge i12 : %s", a10Plage(j))
	}
	if s := a10Seuils(cles); len(s) > 0 {
		t.Logf("           seuil i13 : %s", a10Plage(s))
	}
	return &sc
}

// a10Cles ne garde que les lectures d'image-cle — cf. l'en-tete du fichier.
func a10Cles(reads []filmdec.ObjectiveRead) []filmdec.ObjectiveRead {
	out := make([]filmdec.ObjectiveRead, 0, len(reads))
	for _, r := range reads {
		if r.FromKeyframe {
			out = append(out, r)
		}
	}
	return out
}

// a10Pct rend un pourcentage lisible, ou « n/a » si le denominateur est nul.
func a10Pct(n, total int) string {
	if total == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f %%", 100*float64(n)/float64(total))
}

// a10ParChamp compte les lectures par champ publie.
func a10ParChamp(reads []filmdec.ObjectiveRead) string {
	n := map[filmdec.ObjectiveField]int{}
	for _, r := range reads {
		n[r.Field]++
	}
	ks := make([]int, 0, len(n))
	for k := range n {
		ks = append(ks, int(k))
	}
	sort.Ints(ks)
	out := ""
	for _, k := range ks {
		if out != "" {
			out += ", "
		}
		out += fmt.Sprintf("%s %d", filmdec.ObjectiveField(k), n[filmdec.ObjectiveField(k)])
	}
	if out == "" {
		return "(aucune)"
	}
	return out
}

func a10Jauges(reads []filmdec.ObjectiveRead) []uint64 {
	return a10Valeurs(reads, filmdec.ObjectiveFieldProgress)
}

func a10Seuils(reads []filmdec.ObjectiveRead) []uint64 {
	return a10Valeurs(reads, filmdec.ObjectiveFieldRequiredProgress)
}

func a10Valeurs(reads []filmdec.ObjectiveRead, f filmdec.ObjectiveField) []uint64 {
	var out []uint64
	for _, r := range reads {
		if r.Field == f {
			out = append(out, r.Value)
		}
	}
	return out
}

// a10Plage rend la plage des valeurs, BRUTES et relues en flottant — la convention de
// `ObjectiveProgressFloat` se juge ici : des flottants dans [0, 1] ou [0, seuil] la confirment,
// des valeurs absurdes la refutent.
func a10Plage(vs []uint64) string {
	tri := append([]uint64(nil), vs...)
	sort.Slice(tri, func(i, j int) bool { return tri[i] < tri[j] })
	distinctes := 1
	for i := 1; i < len(tri); i++ {
		if tri[i] != tri[i-1] {
			distinctes++
		}
	}
	lo, hi := tri[0], tri[len(tri)-1]
	med := tri[len(tri)/2]
	return fmt.Sprintf("%d lecture(s), %d valeur(s) distincte(s), brut [%d .. %d] mediane %d ; "+
		"en flottant [%g .. %g] mediane %g",
		len(vs), distinctes, lo, hi, med,
		filmdec.ObjectiveProgressFloat(lo), filmdec.ObjectiveProgressFloat(hi),
		filmdec.ObjectiveProgressFloat(med))
}

// a10MonteesParSlot rend, par slot, les instants (ms moteur) ou la jauge a CRU.
func a10MonteesParSlot(sc filmdec.ObjectiveScan) map[uint32][]int {
	series := map[uint32]*a10Serie{}
	for _, r := range sc.Reads {
		if r.Field != filmdec.ObjectiveFieldProgress || !r.FromKeyframe {
			continue
		}
		s := series[r.Slot]
		if s == nil {
			s = &a10Serie{}
			series[r.Slot] = s
		}
		s.ts, s.val = append(s.ts, r.TimestampUS), append(s.val, r.Value)
	}
	out := map[uint32][]int{}
	for slot, s := range series {
		ord := make([]int, len(s.ts))
		for i := range ord {
			ord[i] = i
		}
		sort.SliceStable(ord, func(a, b int) bool { return s.ts[ord[a]] < s.ts[ord[b]] })
		vu := false
		var dernier uint64
		for _, i := range ord {
			if vu && s.val[i] > dernier {
				out[slot] = append(out[slot], int(s.ts[i]/1000))
			}
			dernier, vu = s.val[i], true
		}
	}
	return out
}

// a10PlusProche rend le delai (ms) entre la montee la plus proche AVANT `ms`, ou -1.
func a10PlusProche(ts []int, ms int) int {
	meilleur := -1
	for _, p := range ts {
		d := ms - p
		if d > 0 && d <= a6MecheMaxMS && (meilleur < 0 || d < meilleur) {
			meilleur = d
		}
	}
	return meilleur
}

// a10Verdict applique les trois criteres et imprime le resultat.
func a10Verdict(t *testing.T, delais map[uint32][]float64, couverts map[uint32]int, total int) {
	t.Helper()
	type v struct {
		slot    uint32
		couvert int
		med, cv float64
	}
	var tenus []v
	for slot, ds := range delais {
		if couverts[slot] < total {
			continue
		}
		if med, cv := a6MedianeEtCV(ds); cv <= a6CVMax {
			tenus = append(tenus, v{slot, couverts[slot], med, cv})
		}
	}
	sort.Slice(tenus, func(i, j int) bool { return tenus[i].cv < tenus[j].cv })
	if len(tenus) == 0 {
		t.Logf("AUCUN SLOT NE TIENT LES TROIS CRITERES. Les meilleures couvertures :")
		type l struct {
			slot    uint32
			n       int
			med, cv float64
		}
		ls := make([]l, 0, len(couverts))
		for slot, n := range couverts {
			med, cv := a6MedianeEtCV(delais[slot])
			ls = append(ls, l{slot, n, med, cv})
		}
		sort.Slice(ls, func(i, j int) bool {
			if ls[i].n != ls[j].n {
				return ls[i].n > ls[j].n
			}
			return ls[i].cv < ls[j].cv
		})
		for i, x := range ls {
			if i >= 12 {
				break
			}
			if math.IsInf(x.cv, 1) {
				continue
			}
			t.Logf("  slot %d : %d/%d couvertes, mediane %.1f s, dispersion %.0f %%",
				x.slot, x.n, total, x.med/1000, x.cv*100)
		}
		return
	}
	for _, x := range tenus {
		t.Logf("CANDIDAT slot %d : %d/%d couvertes, delai median %.1f s (meche + decalage), "+
			"dispersion %.0f %%", x.slot, x.couvert, total, x.med/1000, x.cv*100)
	}
}

// TestAssautA10Detail — CE QUE LES VALEURS DE JAUGE SONT REELLEMENT.
//
// La premiere passe a sorti 94 lectures de `i12` sur un film, mais SEULEMENT QUATRE valeurs
// distinctes, toutes de la forme « un seul bit pose » (4, 256, 512, 32768, 65536, 2097152), et un
// seuil `i13` CONSTANT a 4. Aucune serie ne monte. Deux lectures s'affrontent :
//
//	A. le canal est bien lu, et ces valeurs ne sont pas une jauge continue (drapeaux ? unites ?) ;
//	B. le canal est lu QUELQUES BITS A COTE, et ces puissances de deux sont la signature d'une
//	   fenetre qui glisse sur une zone presque nulle.
//
// Un seul instrument les departage : imprimer les records porteurs de jauge EN ENTIER — slot,
// instant, et la suite des champs publies dans l'ordre. Une valeur qui se repete a l'identique sur
// un meme slot au fil du temps dit A ; une valeur qui saute d'une puissance de deux a l'autre sans
// jamais se repeter dit B.
//
//	go test ./internal/analysis/replay/ -run AssautA10Detail -v -timeout 30m
func TestAssautA10Detail(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer amArmeSentinelle(t, "TestAssautA10Detail")()
	for _, id := range []string{"34bb3bc8", "9f57c612", "c75f33b8", "df8fcbef"} {
		sc, err := filmdec.ScanFilmObjectives(filepath.Join(cache, "film_chunks", id))
		if err != nil {
			t.Logf("%s : balayage impossible (%v)", id, err)
			continue
		}
		cles := a10Cles(sc.Reads)
		slots := map[uint32][]filmdec.ObjectiveRead{}
		for _, r := range cles {
			if r.Field == filmdec.ObjectiveFieldProgress ||
				r.Field == filmdec.ObjectiveFieldRequiredProgress ||
				r.Field == filmdec.ObjectiveFieldState {
				slots[r.Slot] = append(slots[r.Slot], r)
			}
		}
		ks := make([]int, 0, len(slots))
		for k := range slots {
			ks = append(ks, int(k))
		}
		sort.Ints(ks)
		t.Logf("########## %s — %d slot(s) porteurs de jauge/seuil/etat", id, len(ks))
		for _, k := range ks {
			rs := slots[uint32(k)]
			sort.SliceStable(rs, func(a, b int) bool { return rs[a].TimestampUS < rs[b].TimestampUS })
			t.Logf("  slot %-5d %d lecture(s) : %s", k, len(rs), a10Suite(rs))
		}
	}
}

// a10Suite rend la suite datee « instant champ=valeur » d'un slot, bornee pour rester lisible.
func a10Suite(rs []filmdec.ObjectiveRead) string {
	const max = 14
	out := ""
	for i, r := range rs {
		if i >= max {
			out += fmt.Sprintf(" … (+%d)", len(rs)-max)
			break
		}
		nom := "i12"
		switch r.Field {
		case filmdec.ObjectiveFieldRequiredProgress:
			nom = "i13"
		case filmdec.ObjectiveFieldState:
			nom = "i14"
		}
		out += fmt.Sprintf(" %.1fs:%s=%d", float64(r.TimestampUS)/1e6, nom, r.Value)
	}
	return out
}
