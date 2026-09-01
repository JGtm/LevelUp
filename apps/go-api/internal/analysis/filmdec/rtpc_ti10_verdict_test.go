package filmdec

// rtpc_ti10_verdict_test.go — LES PORTES 0, 1 ET 4 DU LOT ti=10 i26..i29, et l'oracle qu'elles
// interrogent. Le CRITERE est ecrit dans l'en-tete de `rtpc_ti10_assaut_test.go` ; ce fichier ne
// fait que l'appliquer et publier les chiffres, y compris quand ils disent non.
//
// L'ORACLE N'EST PAS RECOPIE UNE TROISIEME FOIS : `ti12Explosions` (28 explosions, 9 films,
// recopie du releve A0.3 fige le 2026-08-27) et sa partition `ti12UneBombe` vivent deja dans le
// paquet, gardes par `TestNavpointTi12OracleFige`. Les seuils du critere non plus :
// `ti12SensMaxMS` et `ti12DispersionMax` sont ceux du chantier, pas ceux de ti=12.

import (
	"sort"
	"strings"
	"testing"
)

// ti10Filtre porte les options d'une passe du critere (regle des 5 parametres : un objet plutot
// qu'une liste d'arguments booleens).
type ti10Filtre struct {
	// chainees restreint aux montees issues de lectures CHAINEES.
	chainees bool
	// garde retient les films (nil = tous).
	garde func(string) bool
	// id, si parID, restreint aux montees d'un seul identifiant.
	id    uint32
	parID bool
}

// ti10Gate0 — PRESENCE de l'archetype sur les films d'Assaut.
func ti10Gate0(t *testing.T, bs []*ti10FilmBilan) {
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
	t.Logf("########## GATE 0 PRESENCE — ti=10 present sur %d film(s) d'Assaut sur 9 · "+
		"%d record(s) ancres · %d lecture(s) rtpc", films, records, lectures)
	if lectures == 0 {
		t.Logf("VERDICT GATE 0 : NEGATIF NET — aucun rtpc lu sur les films d'Assaut. Le "+
			"parametre temps reel ne peut pas dater l'armement.%s", "")
	}
}

// ti10Gate1 — VALIDITE DE L'INSTRUMENT par les temoins d'un autre mode.
func ti10Gate1(t *testing.T, bs []*ti10FilmBilan) {
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
		t.Logf("TEMOIN %-9s %-26s %d lecture(s), %d identifiant(s), %d montee(s), %d serie(s)",
			b.id, b.mode, b.lectures, len(b.ids), len(b.montees), b.seriesN)
	}
	t.Logf("########## GATE 1 VALIDITE — %d temoin(s) sur %d rendent des lectures rtpc", vus, total)
	if vus == 0 {
		t.Logf("VERDICT GATE 1 : L'INSTRUMENT EST CASSE. Le lot C a mesure que ce composant "+
			"porte 17 a 53 %% des records ti=10 en Strongholds ; ne rien y voir disqualifie "+
			"tout verdict sur l'Assaut.%s", "")
	}
}

// ti10Gate4 — LE CRITERE DU CHANTIER contre les 28 explosions.
func ti10Gate4(t *testing.T, bs []*ti10FilmBilan, inv *ti10Inventaire) {
	t.Helper()
	par := map[string]*ti10FilmBilan{}
	for _, b := range bs {
		par[b.id] = b
	}
	t.Logf("########## GATE 4 CRITERE — 28 explosions, delai = explosion moins fin de montee")
	ti10Passe(t, "TOUTES LECTURES (corpus entier)", par, ti10Filtre{})
	ti10Passe(t, "LECTURES CHAINEES (corpus entier)", par, ti10Filtre{chainees: true})
	ti10Passe(t, "NEUTRAL BOMB — 6 films", par,
		ti10Filtre{garde: func(id string) bool { return !ti12UneBombe[id] }})
	ti10Passe(t, "ONE BOMB — 3 films", par,
		ti10Filtre{garde: func(id string) bool { return ti12UneBombe[id] }})
	ti10ParID(t, par, inv)
	ti10Detail(t, par)
}

// ti10ParID applique le MEME critere identifiant par identifiant — c'est la question que la
// mission pose (« la valeur d'un identifiant propre a l'Assaut monte-t-elle dans le temps ? »).
//
// LA MULTIPLICITE EST PUBLIEE : un identifiant qui passe parmi cinquante n'est une trouvaille
// que si sa couverture est PLEINE. Une couverture partielle parmi cinquante candidats ne prouve
// rien, et c'est pour cela que le nombre d'identifiants examines est imprime avant le tableau.
func ti10ParID(t *testing.T, par map[string]*ti10FilmBilan, inv *ti10Inventaire) {
	t.Helper()
	if inv == nil {
		return
	}
	cands := append(append([]uint32{}, inv.propres...), inv.communs...)
	t.Logf("--- PAR IDENTIFIANT — %d candidat(s) examine(s) (%d propres a l'Assaut, %d communs) ; "+
		"les rares sont ecartes par le filtre de robustesse", len(cands), len(inv.propres),
		len(inv.communs))
	for _, id := range cands {
		g := inv.tous[id]
		etiq := "commun"
		if len(g.filmsTemoin) == 0 {
			etiq = "PROPRE A L'ASSAUT"
		}
		ti10Passe(t, "identifiant "+ti10Hex(id)+" ("+etiq+")", par, ti10Filtre{id: id, parID: true})
	}
}

// ti10Hex rend un identifiant en hexadecimal sur huit chiffres.
func ti10Hex(id uint32) string {
	const chiffres = "0123456789abcdef"
	b := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		b[i] = chiffres[id&0xf]
		id >>= 4
	}
	return string(b)
}

// ti10Detail imprime une ligne PAR EXPLOSION : la montee qui la precede, son couple, sa course.
// Un tableau agrege cache toujours le cas qui explique tout ; celui-la ne cache rien.
func ti10Detail(t *testing.T, par map[string]*ti10FilmBilan) {
	t.Helper()
	t.Logf("--- DETAIL PAR EXPLOSION (montee la plus recente avant l'explosion, toutes lectures)")
	for _, id := range ti12FilmsOracle() {
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
			t.Logf("    %s %7d ms : delai %6d ms · slot %d id %s · %d -> %d en %d ech. (debut %d ms)",
				id, ms, ms-int(m.finMS), m.cle.slot, ti10Hex(m.cle.id), m.bas, m.haut, m.n, m.debutMS)
		}
	}
}

// ti10Passe applique le critere sur une voie et sur les films que le filtre retient.
func ti10Passe(t *testing.T, titre string, par map[string]*ti10FilmBilan, f ti10Filtre) {
	t.Helper()
	var delais []int
	total, couvLecture, horsSens := 0, 0, 0
	for _, id := range ti12FilmsOracle() {
		if f.garde != nil && !f.garde(id) {
			continue
		}
		b := par[id]
		for _, ms := range ti12Explosions[id] {
			total++
			if b == nil {
				continue
			}
			if ti10LectureAvant(b.instantsDe(f), int32(ms)) {
				couvLecture++
			}
			d, ok := ti10DelaiAvant(b.monteesDe(f), int32(ms), f)
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
	ti10Verdict(t, titre, [3]int{total, couvLecture, horsSens}, delais)
}

// monteesDe rend la voie de montees que le filtre designe.
func (b *ti10FilmBilan) monteesDe(f ti10Filtre) []ti10Montee {
	if f.chainees {
		return b.monteesChainees
	}
	return b.montees
}

// instantsDe rend les instants de lecture que le filtre designe. Sur une passe PAR IDENTIFIANT,
// la couverture faible se compte sur les seules lectures de cet identifiant.
func (b *ti10FilmBilan) instantsDe(f ti10Filtre) []int32 {
	if f.parID {
		return b.instants[f.id]
	}
	return b.tousInstants
}

// ti10LectureAvant dit si au moins une lecture precede l'instant t.
func ti10LectureAvant(instants []int32, t int32) bool {
	i := sort.Search(len(instants), func(k int) bool { return instants[k] > t })
	return i > 0
}

// ti10DelaiAvant rend le delai entre la DERNIERE montee close avant t et t.
//
// SUR UNE PASSE PAR IDENTIFIANT la recherche est LINEAIRE et non dichotomique : la liste est
// triee par instant de fin, mais le filtre sur l'identifiant la trouerait de facon quelconque.
// Mieux vaut une passe honnete qu'une dichotomie sur un predicat non monotone.
func ti10DelaiAvant(montees []ti10Montee, t int32, f ti10Filtre) (int, bool) {
	if !f.parID {
		i := sort.Search(len(montees), func(k int) bool { return montees[k].finMS > t })
		if i == 0 {
			return 0, false
		}
		return int(t - montees[i-1].finMS), true
	}
	best, ok := 0, false
	for _, m := range montees {
		if m.cle.id != f.id || m.finMS > t {
			continue
		}
		if d := int(t - m.finMS); !ok || d < best {
			best, ok = d, true
		}
	}
	return best, ok
}

// ti10Verdict publie couverture, constance et sens pour une passe. `comptes` porte, dans cet
// ordre : le nombre d'explosions examinees, celles precedees d'une simple lecture, et celles
// dont la montee tombe hors du sens.
func ti10Verdict(t *testing.T, titre string, comptes [3]int, delais []int) {
	t.Helper()
	total, couvLecture, horsSens := comptes[0], comptes[1], comptes[2]
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
		"SENS %d hors [0, %d] ms", disp, ti12DispersionMax, len(delais), total, horsSens,
		ti12SensMaxMS)
	ok := len(delais) == total && disp <= ti12DispersionMax && horsSens == 0
	verdict := "NEGATIF sur ce canal"
	if ok {
		verdict = "CANDIDAT — les trois criteres passent"
	}
	t.Logf("    VERDICT %s : %s", titre, verdict)
}
