package filmdec

// keyframe_baseline_scope_test.go — INSTRUMENT DU LOT R7-c
// (cf. .ai/V7.5/replay2d/PLAN_R7C_ENCODAGE_DRAPEAU_IMAGE_CLE.md).
//
// LA QUESTION. R7-b laisse un residu DISPERSE : 0,54 % d'atterrissage bit-exact, p10 a
// 46 bits, p50 a ~500, p90 a quatre ordres de grandeur, aucun palier. Ni bloc manquant ni
// deserialiseur casse — donc, peut-etre, les MEMES deser lus sous un AUTRE encodage.
//
// CE QUE LA PHASE 1 A TROUVE DANS LE BINAIRE. `DAT_144e61ea0` n'est pas un reglage, c'est
// une PORTEE : les huit lecteurs d'etat complet du groupe `142e2*`/`142e3*` le levent a 1
// juste avant l'appel `vtable[0x60]` (etat par defaut) et le remettent a 0 juste apres.
// Pendant cette portee, `FUN_14076f91c()` est vrai et SIX lecteurs de position du moteur
// passent du quantifie au BRUT 96 bits (`FUN_1411b259c` = `FUN_1406d676c(...,0x60)`).
//
// CE QUE CET INSTRUMENT MESURE. L'A/B de cette portee (`SetKeyframeBaselineScope`) sur le
// corpus R7-b, a reglages par ailleurs IDENTIQUES : largeurs d'axe de la CARTE installees
// (`DetectI0Layout`), corruption-check du mode film ETEINT, `simStateComplete` allume,
// trous neutralises. Une seule variable bouge entre les deux passes.
//
// LECTURE SEULE, garde par KF35_ROOT : saute partout ailleurs, CI comprise. Bascules
// globales restaurees en defer ; LockProcessDecode tenu tout du long.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 KF35_ROOT=<repo>/data/cache/film_chunks \
//	  go test ./internal/analysis/filmdec/ -run '^TestKF35C' -timeout 60m -v

import "testing"

// kf35cScopes : l'A/B du lot. Le premier est la ligne de base R7-b a l'identique.
var kf35cScopes = []struct {
	Label    string
	Baseline bool
}{
	{Label: "TEMOIN portee baseline ETEINTE (= la mesure R7-b)"},
	{Label: "portee baseline ALLUMEE (DAT_144e61ea0 = 1)", Baseline: true},
}

// kf35cSetup installe les reglages communs aux deux branches de l'A/B et rend leur
// restauration. Un seul endroit, pour qu'aucune passe ne parte d'un etat different.
func kf35cSetup() func() {
	prevSim := simStateComplete
	prevCorr := filmComponentCorruptionCheck
	SetSimStateComplete(true)
	SetFilmComponentCorruptionCheck(false)
	return func() {
		SetSimStateComplete(prevSim)
		SetFilmComponentCorruptionCheck(prevCorr)
	}
}

// TestKF35CBaselineScope est LA MESURE du lot : atterrissage bit-exact, longueurs et
// ecarts, portee baseline eteinte puis allumee.
func TestKF35CBaselineScope(t *testing.T) {
	films := kf35Films(t)
	release := LockProcessDecode()
	defer release()
	defer kf35cSetup()()

	for _, s := range kf35cScopes {
		prev := SetKeyframeBaselineScope(s.Baseline)
		t.Logf("======== %s ========", s.Label)
		for _, f := range films {
			_, restore := kf35bInstallPrecision(t, f.Name)
			for _, v := range kf35bVariants {
				kf35Report(t, f, v)
			}
			restore()
		}
		SetKeyframeBaselineScope(prev)
	}
}

// TestKF35CDispersion publie la DISTRIBUTION CUMULEE de l'ecart sous les deux portees.
// C'est elle qui dit si la portee rapproche les records de la frontiere ou si elle ne
// deplace que la mediane.
func TestKF35CDispersion(t *testing.T) {
	films := kf35Films(t)
	release := LockProcessDecode()
	defer release()
	defer kf35cSetup()()

	for _, s := range kf35cScopes {
		prev := SetKeyframeBaselineScope(s.Baseline)
		t.Logf("======== dispersion · %s ========", s.Label)
		for _, f := range films {
			_, restore := kf35bInstallPrecision(t, f.Name)
			for _, v := range kf35bVariants {
				kf35bDispersionOne(t, f, v)
			}
			restore()
		}
		SetKeyframeBaselineScope(prev)
	}
}

// TestKF35CProfile publie le profil par composant sous les deux portees : quel composant
// consomme quoi, et ou la frontiere reelle est franchie. C'est lui qui designe la suite.
func TestKF35CProfile(t *testing.T) {
	films := kf35Films(t)
	release := LockProcessDecode()
	defer release()
	defer kf35cSetup()()

	for _, s := range kf35cScopes {
		prev := SetKeyframeBaselineScope(s.Baseline)
		t.Logf("======== profil par composant · %s ========", s.Label)
		for _, f := range films {
			kf35bProfileOne(t, f, false)
		}
		SetKeyframeBaselineScope(prev)
	}
}
