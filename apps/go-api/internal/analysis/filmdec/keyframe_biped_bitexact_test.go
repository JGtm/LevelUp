package filmdec

// keyframe_biped_bitexact_test.go — INSTRUMENT DU LOT R7-b
// (cf. .ai/V7.5/replay2d/PLAN_R7B_BIPEDE_IMAGE_CLE_BIT_EXACT.md).
//
// R7-a a tranche la FORME du corps d'un record d'image-cle : c'est un ETAT COMPLET (la
// concatenation des 64 deserialiseurs leaf consomme 102-104 % de la longueur reelle du
// record, contre 12-39 % pour la lecture « record NEW »). Il n'a PAS atteint le bit
// (0,51 %). Ce lot cherche pourquoi, composant par composant.
//
// CE QU'IL AJOUTE A L'INSTRUMENT R7-a (dont il REUTILISE tous les helpers `kf35*`) : les
// LARGEURS D'AXE DE LA CARTE. Le chemin ABSOLU d'i0 — celui que joue une image-cle — lit
// sa largeur dans deux globaux de paquet que R7-a n'installait pas :
//
//	absoluteAxisW = 14 UNIFORME (position_capture.go) ; la capture CE donne
//	  3 + 1 + 1 + (13+13+14) + 2 = 47 bits sur Cliffhanger, l'uniforme en rend 49 ;
//	WorldObjectPrecision (traverse.go), defaut {13,13,14} = l'entree `cliffhanger` du
//	  catalogue — donc FAUSSE sur toute autre carte, et lue aussi par le corps tag==3 d'i59.
//
// i0 est le PREMIER composant de 100 % des records : une largeur fausse la plafonne toute
// la marche. L'instrument lit donc le decoupage DANS le film (`DetectI0Layout`, chaine
// disjointe du catalogue) et le pose pour la duree de la passe, avec son A/B.
//
// LECTURE SEULE, garde par KF35_ROOT (meme garde que R7-a) : saute partout ailleurs, CI
// comprise. Bascules globales restaurees en defer ; LockProcessDecode tenu tout du long.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 KF35_ROOT=<repo>/data/cache/film_chunks \
//	  go test ./internal/analysis/filmdec/ -run '^TestKF35B' -timeout 60m -v

import (
	"os"
	"sort"
	"strings"
	"testing"
)

// kf35bPrecision est UNE facon de renseigner les largeurs d'axe pour une passe.
type kf35bPrecision struct {
	Label string
	// FromFilm : lire le decoupage d'i0 dans le film et l'installer. false = laisser les
	// defauts de paquet (le TEMOIN, c'est-a-dire exactement ce que R7-a mesurait).
	FromFilm bool
}

var kf35bPrecisions = []kf35bPrecision{
	{Label: "TEMOIN largeurs par defaut (absoluteAxisW=14 uniforme, Cliffhanger)"},
	{Label: "largeurs de la CARTE lues dans le film (DetectI0Layout)", FromFilm: true},
}

// kf35bDir rend le repertoire d'un film du corpus (la garde KF35_ROOT est deja passee).
func kf35bDir(name string) string {
	return strings.TrimRight(os.Getenv(kf35RootEnv), "/\\") + "/" + name
}

// kf35bInstallPrecision lit le decoupage d'i0 dans le film et l'installe sur LES DEUX
// chemins qui en dependent : `WorldObjectPrecision` (chemin world-object et corps d'i59) et
// la largeur des chemins ABSOLUS d'i0 (`absoluteAxisW`, remise a 0 pour qu'`absAxisW`
// retombe sur les largeurs de la carte au lieu de son uniforme 14). Rend la restauration.
func kf35bInstallPrecision(t *testing.T, name string) (I0Layout, func()) {
	t.Helper()
	prevW, prevAbs := WorldObjectPrecision, absoluteAxisW
	restore := func() { WorldObjectPrecision = prevW; SetAbsoluteAxisW(prevAbs) }
	lay, rep, err := DetectI0Layout(kf35bDir(name))
	if err != nil {
		t.Logf("      [%s] decoupage i0 NON detecte (%v) — largeurs par defaut conservees", name, err)
		return I0Layout{}, restore
	}
	t.Logf("      [%s] decoupage i0 lu dans le film : %s (%d paires, frontieres %v)",
		name, lay, rep.Pairs, rep.Boundaries)
	SetWorldObjectPrecisionFromLayout(lay)
	SetAbsoluteAxisW(0) // 0 => absAxisW retombe sur WorldObjectPrecision.AxisW
	return lay, restore
}

// kf35bVariants : le temoin « record NEW », l'etat complet nu, et — REMISES AU PROGRAMME —
// les lectures avec ETAT PAR DEFAUT en tete. R7-a les avait ecartees parce qu'elles
// desynchronisaient 591 fois sur 591 ; ce n'etait pas la variante qui echouait, c'etaient
// `i57`/`i59`/`i60`. Une fois ces trois-la portes ou neutralises, elles finissent — et la
// question redevient ouverte, puisque l'etat complet nu SOUS-LIT desormais d'environ 300 a
// 400 bits, l'ordre de grandeur d'un etat par defaut de bipede.
var kf35bVariants = []kf35Variant{
	{Label: "v0b TEMOIN record NEW, composants non portes sautes (0 bit)",
		Body: KeyframeBodyVariant{DefaultState: true, Gate: true, Mask: true}, Stub: true},
	{Label: "v4 ETAT COMPLET (64 leaf nus), composants non portes sautes (0 bit)",
		Body: KeyframeBodyVariant{}, Stub: true},
	{Label: "v2 ETAT PAR DEFAUT + 64 leaf, trous sautes (0 bit)",
		Body: KeyframeBodyVariant{DefaultState: true}, Stub: true},
	{Label: "v3 ETAT PAR DEFAUT + porte R(1) + 64 leaf, trous sautes (0 bit)",
		Body: KeyframeBodyVariant{DefaultState: true, Gate: true}, Stub: true},
}

// TestKF35BInventory publie, par film, le decoupage d'i0 LU DANS LE FILM et le compte de
// records ti=35 bornes. C'est lui qui montre que les trois films oracles n'ont PAS les
// memes largeurs d'axe — donc que la mesure R7-a en lisait deux sur trois a cote.
func TestKF35BInventory(t *testing.T) {
	films := kf35Films(t)
	release := LockProcessDecode()
	defer release()

	// i60 est PORTÉ EN ENTIER depuis R7-b (queue FUN_14076e494 incluse) ; seul le défaut de
	// production reste à false, faute de largeurs d'axe de carte sur le chemin absolu. Les
	// mesures d'image-clé, elles, INSTALLENT ces largeurs : elles l'activent donc.
	prevSim := simStateComplete
	SetSimStateComplete(true)
	defer SetSimStateComplete(prevSim)

	for _, f := range films {
		lay, restore := kf35bInstallPrecision(t, f.Name)
		n := 0
		var lens []int
		for _, pay := range f.Pays {
			for _, b := range kf35BoundedRecs(pay) {
				n++
				lens = append(lens, b.Want-b.Rec.Bit)
			}
		}
		t.Logf("  film %s : axes %v · %d tables · %d records ti=%d bornes · longueur REELLE mediane %d bits",
			f.Name, lay.AxisW, len(f.Pays), n, bipedDefaultStateTypeIndex, kf35Median(lens))
		restore()
	}
}

// TestKF35BBitExact est LA MESURE du lot : l'A/B des largeurs d'axe, sous le
// corruption-check du mode film eteint puis allume, sur les deux lectures encore debout.
func TestKF35BBitExact(t *testing.T) {
	films := kf35Films(t)
	release := LockProcessDecode()
	defer release()

	// i60 est PORTÉ EN ENTIER depuis R7-b (queue FUN_14076e494 incluse) ; seul le défaut de
	// production reste à false, faute de largeurs d'axe de carte sur le chemin absolu. Les
	// mesures d'image-clé, elles, INSTALLENT ces largeurs : elles l'activent donc.
	prevSim := simStateComplete
	SetSimStateComplete(true)
	defer SetSimStateComplete(prevSim)

	for _, corr := range []bool{false, true} {
		prev := filmComponentCorruptionCheck
		SetFilmComponentCorruptionCheck(corr)
		for _, p := range kf35bPrecisions {
			t.Logf("======== corruption-check=%v · %s ========", corr, p.Label)
			for _, f := range films {
				kf35bOnePass(t, f, p)
			}
		}
		SetFilmComponentCorruptionCheck(prev)
	}
}

// kf35bOnePass joue les variantes retenues sur UN film sous UN reglage de largeurs.
func kf35bOnePass(t *testing.T, f kf35Film, p kf35bPrecision) {
	t.Helper()
	if p.FromFilm {
		_, restore := kf35bInstallPrecision(t, f.Name)
		defer restore()
	}
	for _, v := range kf35bVariants {
		kf35Report(t, f, v)
	}
}

// ---------------------------------------------------------------------------
// Sonde de PROFIL : ou la marche est-elle encore juste, et ou decroche-t-elle ?
// ---------------------------------------------------------------------------

// kf35bCompStat cumule, pour UN composant, ce que la marche y a consomme. Sans ca, un
// « ecart de 300 bits » ne designe personne : il faut savoir QUEL composant lit large.
type kf35bCompStat struct {
	Name  string
	Seen  int
	Bits  []int // largeur consommee par ce composant, record par record
	Break int   // fois ou la frontiere reelle a ete franchie PENDANT ce composant
}

// kf35bProfile parcourt les records d'un film sous une variante et rend le profil par
// composant : combien de fois vu, largeur mediane consommee, et combien de fois la
// frontiere du record suivant a ete franchie a l'interieur du composant.
func kf35bProfile(f kf35Film, v kf35Variant) []kf35bCompStat {
	arch, _ := f.Reg.Archetype(bipedDefaultStateTypeIndex)
	stats := make([]kf35bCompStat, len(arch.Components))
	for i, n := range arch.Components {
		stats[i].Name = n
	}
	for _, pay := range f.Pays {
		for _, b := range kf35BoundedRecs(pay) {
			tr := WalkKeyframeBody(pay, b.Rec.Bit, f.Reg, v.Body)
			kf35bAccumulate(tr, b, stats)
		}
	}
	return stats
}

// kf35bAccumulate impute a chaque composant sa largeur consommee sur UNE marche.
func kf35bAccumulate(tr EntityTrace, b kf35Bound, stats []kf35bCompStat) {
	for k, c := range tr.Comps {
		if c.Index >= len(stats) {
			continue
		}
		end := tr.EndBit
		if k+1 < len(tr.Comps) {
			end = tr.Comps[k+1].StartBit
		}
		stats[c.Index].Seen++
		stats[c.Index].Bits = append(stats[c.Index].Bits, end-c.StartBit)
		if c.StartBit <= b.Want && b.Want < end {
			stats[c.Index].Break++
		}
	}
}

// TestKF35BProfile publie le profil par composant de la lecture « etat complet », largeurs
// de carte installees : c'est la piece qui ORDONNE la suite du plan (quel deser corriger).
func TestKF35BProfile(t *testing.T) {
	films := kf35Films(t)
	release := LockProcessDecode()
	defer release()

	// i60 est PORTÉ EN ENTIER depuis R7-b (queue FUN_14076e494 incluse) ; seul le défaut de
	// production reste à false, faute de largeurs d'axe de carte sur le chemin absolu. Les
	// mesures d'image-clé, elles, INSTALLENT ces largeurs : elles l'activent donc.
	prevSim := simStateComplete
	SetSimStateComplete(true)
	defer SetSimStateComplete(prevSim)

	prev := filmComponentCorruptionCheck
	defer SetFilmComponentCorruptionCheck(prev)
	for _, corr := range []bool{false, true} {
		SetFilmComponentCorruptionCheck(corr)
		t.Logf("======== profil par composant · corruption-check=%v ========", corr)
		for _, f := range films {
			kf35bProfileOne(t, f, corr)
		}
	}
}

// kf35bProfileOne installe les largeurs du film, neutralise les trous, et publie le profil.
func kf35bProfileOne(t *testing.T, f kf35Film, corr bool) {
	t.Helper()
	_, restore := kf35bInstallPrecision(t, f.Name)
	defer restore()
	v := kf35bVariants[len(kf35bVariants)-1] // v4 etat complet
	stubbed, unstub := kf35ApplyStubs(f, v)
	defer unstub()
	t.Logf("  [%s] corruption=%v · composants neutralises : %v", f.Name, corr, stubbed)
	for _, s := range kf35bProfile(f, v) {
		if s.Seen == 0 {
			continue
		}
		t.Logf("      i%-2d %-58s vu %4d · largeur mediane %5d · frontiere franchie ici %4d",
			kf35bIndexOf(f, s.Name), s.Name, s.Seen, kf35Median(s.Bits), s.Break)
	}
}

// kf35bIndexOf rend l'index du composant dans l'archetype bipede (-1 si absent).
func kf35bIndexOf(f kf35Film, name string) int {
	arch, _ := f.Reg.Archetype(bipedDefaultStateTypeIndex)
	for i, n := range arch.Components {
		if n == name {
			return i
		}
	}
	return -1
}

// ---------------------------------------------------------------------------
// DISPERSION — la piece qui tranche « bloc manquant » contre « derive dispersee »
// ---------------------------------------------------------------------------

// kf35bQuantile rend le quantile q (0..1) d'un echantillon. Il TRIE la tranche recue.
func kf35bQuantile(xs []int, q float64) int {
	if len(xs) == 0 {
		return 0
	}
	sort.Ints(xs)
	i := int(q * float64(len(xs)-1))
	return xs[i]
}

// TestKF35BDispersion publie la DISTRIBUTION CUMULEE de l'ecart, pas sa seule mediane.
//
// LA QUESTION QU'IL TRANCHE. Une mediane de 500 bits se lit de deux facons opposees : soit
// tous les records ratent d'environ 500 bits (un BLOC MANQUANT de largeur stable, qu'on va
// alors chercher dans le binaire), soit une part des records tombe a quelques bits pendant
// qu'une autre part part tres loin (une DERIVE, qu'aucun bloc ne corrigera). Les parts a
// 8 / 16 / 64 / 256 bits le disent ; la mediane, non.
//
// UNE SEULE configuration, la meilleure mesuree : largeurs d'axe de la CARTE installees,
// corruption-check du mode film ETEINT (allume est desormais pire, cf. plan R7-b).
func TestKF35BDispersion(t *testing.T) {
	films := kf35Films(t)
	release := LockProcessDecode()
	defer release()

	prevSim := simStateComplete
	SetSimStateComplete(true)
	defer SetSimStateComplete(prevSim)
	prevCorr := filmComponentCorruptionCheck
	SetFilmComponentCorruptionCheck(false)
	defer SetFilmComponentCorruptionCheck(prevCorr)

	for _, f := range films {
		_, restore := kf35bInstallPrecision(t, f.Name)
		for _, v := range kf35bVariants {
			kf35bDispersionOne(t, f, v)
		}
		restore()
	}
}

// kf35bDispersionOne joue une variante sur un film et publie les parts cumulees de |ecart|.
func kf35bDispersionOne(t *testing.T, f kf35Film, v kf35Variant) {
	t.Helper()
	if v.Stub {
		_, unstub := kf35ApplyStubs(f, v)
		defer unstub()
	}
	tal := kf35Pass(f, v)
	n := tal.bounded
	if n == 0 {
		return
	}
	abs := append([]int(nil), tal.absGaps...)
	t.Logf("  [%s] %s", f.Name, v.Label)
	t.Logf("      |ecart| p10 %d · p25 %d · p50 %d · p75 %d · p90 %d bits",
		kf35bQuantile(abs, 0.10), kf35bQuantile(abs, 0.25), kf35bQuantile(abs, 0.50),
		kf35bQuantile(abs, 0.75), kf35bQuantile(abs, 0.90))
	for _, s := range []int{0, 8, 16, 64, 256} {
		c := tal.exact
		for _, g := range tal.absGaps {
			if g <= s && g > 0 {
				c++
			}
		}
		t.Logf("      |ecart| <= %3d bits : %4d / %4d = %5.1f %%", s, c, n, 100*float64(c)/float64(n))
	}
}
