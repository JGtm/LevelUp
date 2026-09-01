package filmdec

// navpoint_ti12_verdict_test.go — LES QUATRE PORTES DU LOT ti=12 i14, et l'oracle qu'elles
// interrogent. Le CRITERE est ecrit dans l'en-tete de `navpoint_ti12_radial_test.go` ; ce
// fichier ne fait que l'appliquer et publier les chiffres, y compris quand ils disent non.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// ti12Explosions est l'oracle des instants d'explosion, RECOPIE SANS MODIFICATION de
// `a5Explosions` (`internal/analysis/replay/assaut_a5_explosions_test.go`), lui-meme recopie du
// releve A0.3 (`A_PROTOCOLE.md` §2, commite le 2026-08-27).
//
// POURQUOI UNE COPIE. `analysis/replay` importe `analysis/filmdec` ; l'inverse ferait un cycle,
// et l'oracle ne peut donc pas etre importe d'ici. La copie est CONFINEE A UN FICHIER DE TEST et
// gardee par `TestNavpointTi12OracleFige`, qui verifie le total (28 explosions, 9 films) : une
// derive de l'oracle amont fait rougir cette garde.
var ti12Explosions = map[string][]int{
	"35b75a31": {304013, 541270, 787051},
	"69b16f5d": {154305, 278617, 310215},
	"3d58eb37": {203065, 342196, 386280},
	"34bb3bc8": {427120},
	"1c01e34f": {150546, 273787, 335637, 400853},
	"ce083875": {512505, 686401, 947537},
	"df8fcbef": {255767, 309284, 485860, 778033},
	"c75f33b8": {109549, 395724, 450833},
	"9f57c612": {83322, 298489, 353160, 469057},
}

// ti12UneBombe designe les TROIS films de la variante One Bomb du corpus.
//
// CE DECOUPAGE EST ANTERIEUR A CETTE MESURE, et c'est ce qui autorise a s'en servir : il est
// ecrit dans `.ai/V7.5/ETAT_ASSAUT_2026-08-31.md` §1.b (« sur df8fcbef, c75f33b8 et 9f57c612 »,
// les seuls films MULTI-MANCHES du corpus) et §1.e, ou les memes trois films portent les cinq
// pertes du pont d'identite. Il n'a pas ete choisi apres coup pour separer un resultat : il
// separait deja le corpus la veille.
//
// Le verdict du critere reste celui du CORPUS ENTIER ; la partition est publiee A COTE, jamais
// a la place.
var ti12UneBombe = map[string]bool{"9f57c612": true, "c75f33b8": true, "df8fcbef": true}

// TestNavpointTi12OracleFige garde la copie de l'oracle contre une derive silencieuse.
func TestNavpointTi12OracleFige(t *testing.T) {
	n := 0
	for _, v := range ti12Explosions {
		n += len(v)
	}
	if len(ti12Explosions) != 9 || n != 28 {
		t.Fatalf("oracle recopie : %d films / %d explosions, attendu 9 / 28", len(ti12Explosions), n)
	}
}

// ti12JournalFilm publie ce qu'un film a rendu.
func ti12JournalFilm(t *testing.T, b *ti12FilmBilan) {
	t.Helper()
	sc := b.sc
	t.Logf("%-9s %-26s slots observes %d (bande comblee %d) · %d vie(s) recensee(s)",
		b.id, b.mode, sc.SlotsObserves, sc.SlotsBande, sc.KeyCensus)
	t.Logf("           DELTA %d ancres, %d marches, %d cassees, %d chainees (%.1f %%) | "+
		"IMAGE-CLE %d records, %d marches, %d cassees, %d chainees",
		sc.Records, sc.Walked, sc.Broken, sc.Chained, ti11Part(sc.Chained, sc.Walked),
		sc.KeyRecords, sc.KeyWalked, sc.KeyBroken, sc.KeyChained)
	t.Logf("           LECTURES i14 : %d dont %d chainees · %d slot(s) porteur(s) · "+
		"%d valeur(s) distincte(s) · %d slot(s) a valeur variable · %d montee(s) (%d chainees)",
		b.lectures, b.chainees, b.slotsPorteurs, b.valeursDistinctes, b.slotsVariables,
		len(b.montees), len(b.monteesChainees))
	if b.lectures > 0 {
		t.Logf("           HISTOGRAMME des quanta par huitieme de plage : %v", b.histo)
	}
	if sc.Tronque {
		t.Logf("           ATTENTION : plafond de %d lectures ATTEINT — la recolte est tronquee",
			ti12MaxLectures)
	}
	if sc.PaquetsSansHorloge > 0 {
		t.Logf("           %d paquet(s) sans horloge (chunk absent du manifeste) — ecartes",
			sc.PaquetsSansHorloge)
	}
	t.Logf("           PREMIER COMPOSANT BLOQUANT : %s · balayage en %s",
		ti12Bloquants(sc.Bloque), b.duree.Round(time.Second))
	for _, e := range b.extraits {
		t.Logf("           %s", e)
	}
}

// ti12Bloquants rend la distribution des composants qui arretent la marche.
func ti12Bloquants(m map[int]int) string {
	if len(m) == 0 {
		return "(aucun — toutes les marches ont abouti)"
	}
	ks := make([]int, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(a, c int) bool { return m[ks[a]] > m[ks[c]] })
	var sb strings.Builder
	for i, k := range ks {
		if i == 6 {
			fmt.Fprintf(&sb, ", (+%d autres)", len(ks)-6)
			break
		}
		if i > 0 {
			sb.WriteString(", ")
		}
		if k < 0 {
			fmt.Fprintf(&sb, "AUCUN %d", m[k])
			continue
		}
		fmt.Fprintf(&sb, "i%d %d", k, m[k])
	}
	return sb.String()
}

// ti12Extraits rend EN CLAIR les series des trois slots les plus fournis : instant en secondes
// et quantum brut. Une jauge se reconnait a l'oeil ; un bruit uniforme aussi.
func ti12Extraits(series map[uint32][]ti12Ech) []string {
	type l struct {
		slot uint32
		n    int
	}
	var ls []l
	for s, v := range series {
		ls = append(ls, l{s, len(v)})
	}
	sort.Slice(ls, func(a, c int) bool {
		if ls[a].n != ls[c].n {
			return ls[a].n > ls[c].n
		}
		return ls[a].slot < ls[c].slot
	})
	out := make([]string, 0, 3)
	for i, x := range ls {
		if i == 3 {
			break
		}
		out = append(out, fmt.Sprintf("SERIE slot %d (%d ech.) : %s",
			x.slot, x.n, ti12EnClair(series[x.slot])))
	}
	return out
}

// ti12EnClair rend au plus 20 echantillons d'une serie, « t_s=quantum ».
func ti12EnClair(s []ti12Ech) string {
	var sb strings.Builder
	pas := 1
	if len(s) > 20 {
		pas = len(s) / 20
	}
	n := 0
	for i := 0; i < len(s) && n < 20; i += pas {
		if n > 0 {
			sb.WriteString(" ")
		}
		fmt.Fprintf(&sb, "%.1fs=%d", float64(s[i].tMS)/1000, s[i].q)
		n++
	}
	if len(s) > 20 {
		sb.WriteString(" ...")
	}
	return sb.String()
}

// ti12Gate0 — PRESENCE de l'archetype sur les films d'Assaut.
func ti12Gate0(t *testing.T, bs []*ti12FilmBilan) {
	t.Helper()
	films, records, lectures := 0, 0, 0
	for _, b := range bs {
		if strings.HasPrefix(b.mode, "TEMOIN") {
			continue
		}
		if b.sc.SlotsObserves > 0 {
			films++
		}
		records += b.sc.Records + b.sc.KeyRecords
		lectures += b.lectures
	}
	t.Logf("########## GATE 0 PRESENCE — ti=12 present sur %d film(s) d'Assaut sur 9 · "+
		"%d record(s) ancres · %d lecture(s) i14", films, records, lectures)
	if films == 0 {
		t.Logf("VERDICT GATE 0 : NEGATIF NET — l'archetype des points de navigation est absent "+
			"des films d'Assaut. Le disque radial ne peut pas dater l'armement.%s", "")
	}
}

// ti12Gate1 — VALIDITE DE L'INSTRUMENT par les temoins d'un autre mode.
func ti12Gate1(t *testing.T, bs []*ti12FilmBilan) {
	t.Helper()
	vus, total := 0, 0
	for _, b := range bs {
		if !strings.HasPrefix(b.mode, "TEMOIN") {
			continue
		}
		total++
		if b.lectures > 0 {
			vus++
		}
		t.Logf("TEMOIN %-9s %-26s %d lecture(s), %d montee(s), %d slot(s) porteur(s)",
			b.id, b.mode, b.lectures, len(b.montees), b.slotsPorteurs)
	}
	t.Logf("########## GATE 1 VALIDITE — %d temoin(s) sur %d rendent des lectures i14", vus, total)
	if vus == 0 {
		t.Logf("VERDICT GATE 1 : L'INSTRUMENT EST CASSE. La progression radiale EXISTE a l'ecran "+
			"en KOTH et en Strongholds (lot C : 74,7 %% et 93,1 %% des records ti=12) ; ne rien "+
			"y voir disqualifie tout verdict sur l'Assaut.%s", "")
	}
}

// ti12Gate2 — FORME : le canal porte-t-il une progression, ou une constante ?
func ti12Gate2(t *testing.T, bs []*ti12FilmBilan) {
	t.Helper()
	var mont, montCh, variables int
	for _, b := range bs {
		if strings.HasPrefix(b.mode, "TEMOIN") {
			continue
		}
		mont += len(b.montees)
		montCh += len(b.monteesChainees)
		variables += b.slotsVariables
	}
	t.Logf("########## GATE 2 FORME (Assaut) — %d montee(s) (%d sur lectures chainees), "+
		"%d slot(s) a valeur variable · seuils figes : >= %d ech., amplitude >= %d quanta",
		mont, montCh, variables, ti12MonteeMinEch, ti12MonteeMinAmpl)
}

// ti12Gate3 — LE CRITERE DU CHANTIER contre les 28 explosions.
func ti12Gate3(t *testing.T, bs []*ti12FilmBilan) {
	t.Helper()
	par := map[string]*ti12FilmBilan{}
	for _, b := range bs {
		par[b.id] = b
	}
	t.Logf("########## GATE 3 CRITERE — 28 explosions, delai = explosion moins fin de montee")
	ti12Passe(t, "TOUTES LECTURES (corpus entier)", par, false, nil)
	ti12Passe(t, "LECTURES CHAINEES (corpus entier)", par, true, nil)
	ti12Passe(t, "NEUTRAL BOMB — 6 films", par, false, func(id string) bool { return !ti12UneBombe[id] })
	ti12Passe(t, "ONE BOMB — 3 films", par, false, func(id string) bool { return ti12UneBombe[id] })
	ti12Detail(t, par)
	ti12ParSlot(t, par)
}

// ti12Detail imprime une ligne PAR EXPLOSION : la montee qui la precede, son slot, sa course.
// Un tableau agrege cache toujours le cas qui explique tout ; celui-la ne cache rien.
func ti12Detail(t *testing.T, par map[string]*ti12FilmBilan) {
	t.Helper()
	films := ti12FilmsOracle()
	t.Logf("--- DETAIL PAR EXPLOSION (montee la plus recente avant l'explosion, toutes lectures)")
	for _, id := range films {
		b := par[id]
		for _, ms := range ti12Explosions[id] {
			if b == nil {
				t.Logf("    %s %7d ms : film absent", id, ms)
				continue
			}
			i := sort.Search(len(b.montees), func(k int) bool { return b.montees[k].finMS > int32(ms) })
			if i == 0 {
				t.Logf("    %s %7d ms : AUCUNE montee avant", id, ms)
				continue
			}
			m := b.montees[i-1]
			t.Logf("    %s %7d ms : delai %6d ms · slot %d · %d -> %d en %d ech. (debut %d ms)",
				id, ms, ms-int(m.finMS), m.slot, m.bas, m.haut, m.n, m.debutMS)
		}
	}
}

// ti12ParSlot decompose le critere PAR SLOT — c'est la question que la mission pose (« la valeur
// monte-t-elle pour un MEME slot ? »), et les slots de navpoint sont stables d'un film d'Assaut a
// l'autre (1459, 1469, 1481 vus sur plusieurs films).
//
// LE CRITERE EST LE MEME, pas un autre : couverture 28/28, dispersion <= 20 %, sens. La
// MULTIPLICITE est publiee (nombre de slots examines) : un slot qui passe parmi quarante n'est
// une trouvaille que si sa couverture est PLEINE — une couverture partielle parmi quarante
// candidats ne prouve rien.
func ti12ParSlot(t *testing.T, par map[string]*ti12FilmBilan) {
	t.Helper()
	delais := map[uint32][]int{}
	for _, id := range ti12FilmsOracle() {
		b := par[id]
		if b == nil {
			continue
		}
		for _, ms := range ti12Explosions[id] {
			for slot, d := range ti12DelaisParSlot(b.montees, int32(ms)) {
				delais[slot] = append(delais[slot], d)
			}
		}
	}
	t.Logf("--- PAR SLOT — %d slot(s) porteurs d'au moins une montee avant une explosion "+
		"(couverture sur 28)", len(delais))
	type l struct {
		slot uint32
		d    []int
	}
	ls := make([]l, 0, len(delais))
	for s, d := range delais {
		sort.Ints(d)
		ls = append(ls, l{s, d})
	}
	sort.Slice(ls, func(a, c int) bool { return len(ls[a].d) > len(ls[c].d) })
	for i, x := range ls {
		if i == 12 {
			t.Logf("    (+%d slot(s) de couverture inferieure)", len(ls)-12)
			break
		}
		med := ti12Quantile(x.d, 0.5)
		disp := 0.0
		if med != 0 {
			disp = float64(ti12Quantile(x.d, 0.75)-ti12Quantile(x.d, 0.25)) / float64(med)
		}
		t.Logf("    slot %-5d couverture %2d/28 · mediane %7d ms · dispersion %.3f · "+
			"min %d · max %d", x.slot, len(x.d), med, disp, x.d[0], x.d[len(x.d)-1])
	}
}

// ti12DelaisParSlot rend, par slot, le delai entre sa derniere montee close avant t et t.
func ti12DelaisParSlot(montees []ti12Montee, t int32) map[uint32]int {
	out := map[uint32]int{}
	for _, m := range montees {
		if m.finMS > t || int(t-m.finMS) > ti12SensMaxMS {
			continue
		}
		if d, ok := out[m.slot]; !ok || int(t-m.finMS) < d {
			out[m.slot] = int(t - m.finMS)
		}
	}
	return out
}

// ti12FilmsOracle rend les identifiants de l'oracle, tries.
func ti12FilmsOracle() []string {
	films := make([]string, 0, len(ti12Explosions))
	for id := range ti12Explosions {
		films = append(films, id)
	}
	sort.Strings(films)
	return films
}

// ti12Passe applique le critere sur une voie (toutes lectures, ou chainees seules) et sur les
// films que `garde` retient (nil = tous).
func ti12Passe(t *testing.T, titre string, par map[string]*ti12FilmBilan, chainees bool,
	garde func(string) bool,
) {
	t.Helper()
	var delais []int
	total, couvertesLecture, horsSens := 0, 0, 0
	for _, id := range ti12FilmsOracle() {
		if garde != nil && !garde(id) {
			continue
		}
		b := par[id]
		for _, ms := range ti12Explosions[id] {
			total++
			if b == nil {
				continue
			}
			montees, instants := b.montees, b.instants
			if chainees {
				montees, instants = b.monteesChainees, b.instantsChaines
			}
			if ti12LectureAvant(instants, int32(ms)) {
				couvertesLecture++
			}
			d, ok := ti12DelaiAvant(montees, int32(ms))
			if !ok {
				continue
			}
			if d > ti12SensMaxMS {
				horsSens++
				continue
			}
			delais = append(delais, d)
		}
	}
	ti12Verdict(t, titre, total, couvertesLecture, horsSens, delais)
}

// ti12LectureAvant dit si au moins une lecture precede l'instant t.
func ti12LectureAvant(instants []int32, t int32) bool {
	i := sort.Search(len(instants), func(k int) bool { return instants[k] > t })
	return i > 0
}

// ti12DelaiAvant rend le delai entre la DERNIERE montee close avant t et t.
func ti12DelaiAvant(montees []ti12Montee, t int32) (int, bool) {
	i := sort.Search(len(montees), func(k int) bool { return montees[k].finMS > t })
	if i == 0 {
		return 0, false
	}
	return int(t - montees[i-1].finMS), true
}

// ti12Verdict publie couverture, constance et sens pour une voie.
func ti12Verdict(t *testing.T, titre string, total, couvLecture, horsSens int, delais []int) {
	t.Helper()
	sort.Ints(delais)
	t.Logf("--- %s : %d/%d explosion(s) precedees d'une MONTEE dans le sens (%d hors sens > %d ms) · "+
		"%d/%d precedees d'une simple lecture", titre, len(delais), total, horsSens,
		ti12SensMaxMS, couvLecture, total)
	if len(delais) == 0 {
		t.Logf("    COUVERTURE 0 — critere NON REMPLI, rien a mesurer sur la constance")
		return
	}
	med := ti12Quantile(delais, 0.5)
	p25, p75 := ti12Quantile(delais, 0.25), ti12Quantile(delais, 0.75)
	disp := 0.0
	if med != 0 {
		disp = float64(p75-p25) / float64(med)
	}
	t.Logf("    DELAIS (ms) : min %d · p25 %d · mediane %d · p75 %d · max %d",
		delais[0], p25, med, p75, delais[len(delais)-1])
	t.Logf("    DISPERSION (p75-p25)/mediane = %.3f (plafond %.2f) · COUVERTURE %d/%d · "+
		"SENS %d hors [0, %d] ms", disp, ti12DispersionMax, len(delais), total, horsSens, ti12SensMaxMS)
	ok := len(delais) == total && disp <= ti12DispersionMax && horsSens == 0
	verdict := "NEGATIF sur ce canal"
	if ok {
		verdict = "CANDIDAT — les trois criteres passent"
	}
	t.Logf("    VERDICT %s : %s", titre, verdict)
}

// ti12Quantile rend le quantile q d'une serie DEJA TRIEE (methode du plus proche rang).
func ti12Quantile(s []int, q float64) int {
	if len(s) == 0 {
		return 0
	}
	i := int(q * float64(len(s)-1))
	if i < 0 {
		i = 0
	}
	if i >= len(s) {
		i = len(s) - 1
	}
	return s[i]
}
