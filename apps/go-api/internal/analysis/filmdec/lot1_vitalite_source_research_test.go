package filmdec

// lot1_vitalite_source_research_test.go — LOT 1 : QUE VAUT LA VITALITÉ DU FILM (i4 santé,
// i5 bouclier), ET EST-CE LA MÊME DONNÉE QUE CELLE DÉJÀ AFFICHÉE PAR LE REJEU 2D ?
//
// CE QUE SONT i4 ET i5. Deux composants d'un record biped, décodés dans le MÊME balayage que
// la position (ScanFilmBipedPositions, CaptureDirs) :
//
//	i4 object-body-vitality  (vitality.go, FUN_140fb8978) : quantum R(8) déquantifié sur
//	   [-1, +1] (endpointExact). HealthFraction replie la moitié négative sur 0 -> [0, 1].
//	i5 object-shield-vitality (vitality.go, FUN_140d50cbc) : quantum R(8) déquantifié sur
//	   [0, 4] (endpointExact). ShieldFraction clampe à 1 -> [0, 1] ; le surbouclier vit dans
//	   le QUANTUM BRUT (Shield.Q > OvershieldFullQ = 64).
//
// OÙ LE REJEU 2D L'UTILISE DÉJÀ. `replay.decimateTracks` (build.go) remplit, par point de
// trajectoire, `Point.Hp = p.HealthAt()` et `Point.Sh = p.ShieldAt()` (document_aim.go) —
// EXACTEMENT ces méthodes-ci de BipedPosition. La « vitalité du rejeu » N'EST donc PAS une
// autre source : c'est i4/i5. Cet instrument le CHIFFRE (couverture, plages) pour dire ce que
// la donnée vaut, sur les 3 films du corpus. Il n'invente aucune source concurrente.
//
// SEUILS / TÉMOINS ÉCRITS AVANT LA MESURE :
//
//	T1 (forme bouclier) : les quanta i5 doivent tomber dans [0, ~255] mais se concentrer sur
//	    [0, 64] (bouclier standard [0,1]) ; une part hors [0,64] = surbouclier, pas du bruit.
//	T2 (forme santé) : les valeurs i4 déquantifiées doivent rester dans [-1, +1] ; la part
//	    exploitable (fraction > 0) doit dominer (un joueur vivant a de la vie).
//	T3 (couverture) : le film ne réplique la vitalité que lorsqu'elle CHANGE — on ATTEND donc
//	    une couverture PARTIELLE, bien plus haute pour le bouclier (change souvent en combat)
//	    que pour la santé (ne bouge qu'aux dégâts qui percent le bouclier). Publier les deux
//	    parts, pas les juger : c'est la nature de la source, pas un défaut de décodage.
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule.

import (
	"os"
	"testing"
)

// lot1vMinMaxF suit min/max d'une série de float32 (init à la première valeur).
type lot1vMinMaxF struct {
	set      bool
	min, max float32
}

func (m *lot1vMinMaxF) add(v float32) {
	if !m.set || v < m.min {
		m.min = v
	}
	if !m.set || v > m.max {
		m.max = v
	}
	m.set = true
}

// lot1vMinMaxU suit min/max d'une série d'uint8.
type lot1vMinMaxU struct {
	set      bool
	min, max uint8
}

func (m *lot1vMinMaxU) add(v uint8) {
	if !m.set || v < m.min {
		m.min = v
	}
	if !m.set || v > m.max {
		m.max = v
	}
	m.set = true
}

func TestLot1VitaliteSource(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	// MÊME balayage que le rejeu 2D (build.go) : QuantaOnly (pas besoin des bornes de carte
	// pour la vitalité) + CaptureDirs (c'est CaptureDirs qui poursuit le record jusqu'à i4/i5).
	opt := DefaultScanFilmOptions()
	opt.QuantaOnly = true
	opt.CaptureDirs = true
	opt.IsolationGapMS = 0 // garder toute mesure : ne pas jeter un échantillon de vitalité isolé
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("balayage biped impossible : %v", err)
	}
	if len(pos) == 0 {
		t.Fatalf("aucune position biped décodée dans %s", dir)
	}

	var (
		nBody, nShield, nOver int
		hQ, sQ                lot1vMinMaxU
		hVal, sVal            lot1vMinMaxF // valeurs brutes déquantifiées (i4 [-1,1], i5 [0,4])
		hFrac, sFrac          lot1vMinMaxF // fractions publiées ([0,1])
		hPos                  int          // santé à fraction > 0 (moitié exploitable)
	)
	for _, p := range pos {
		if p.HasBody {
			nBody++
			hQ.add(p.Body.Q)
			hVal.add(p.Body.Health)
			f, _ := p.HealthAt()
			hFrac.add(f)
			if f > 0 {
				hPos++
			}
		}
		if p.HasShield {
			nShield++
			sQ.add(p.Shield.Q)
			sVal.add(p.Shield.Shield)
			f, _ := p.ShieldAt()
			sFrac.add(f)
			if p.Shield.Q > OvershieldFullQ {
				nOver++
			}
		}
	}

	total := len(pos)
	t.Logf("== film %s : %d records biped (positions décodées) ==", dir, total)
	t.Logf("SANTÉ i4 : couverture %d/%d (%.2f %%) · Q[%d..%d] · valeur[-1,1] observée [%.3f..%.3f] · fraction[%.3f..%.3f] · fraction>0 : %d/%d (%.1f %%)",
		nBody, total, lot1Pct(nBody, total), hQ.min, hQ.max, hVal.min, hVal.max, hFrac.min, hFrac.max,
		hPos, nBody, lot1Pct(hPos, nBody))
	t.Logf("BOUCLIER i5 : couverture %d/%d (%.2f %%) · Q[%d..%d] · valeur[0,4] observée [%.3f..%.3f] · fraction[%.3f..%.3f] · surbouclier Q>%d : %d/%d (%.1f %%)",
		nShield, total, lot1Pct(nShield, total), sQ.min, sQ.max, sVal.min, sVal.max, sFrac.min, sFrac.max,
		OvershieldFullQ, nOver, nShield, lot1Pct(nOver, nShield))

	// T1 : le quantum bouclier se concentre-t-il sur la plage standard [0,64] ?
	horsStd := nShield - nOver // Q <= 64
	t.Logf("T1 forme bouclier : dans plage standard Q<=%d : %d/%d (%.1f %%) — %s",
		OvershieldFullQ, horsStd, nShield, lot1Pct(horsStd, nShield),
		lot1Verdict(nShield > 0 && sQ.max <= 255))
	// T2 : la valeur santé reste-t-elle dans [-1, +1] (sérialisation) ?
	t.Logf("T2 forme santé : valeur dans [-1,1] : %s (min %.3f, max %.3f)",
		lot1Verdict(nBody == 0 || (hVal.min >= -1.0001 && hVal.max <= 1.0001)), hVal.min, hVal.max)

	t.Logf("VERDICT Q1 : i4/i5 sont EXACTEMENT p.HealthAt()/p.ShieldAt() (BipedPosition), " +
		"que le rejeu 2D publie déjà en Point.Hp/Point.Sh (build.go:647-651, document_aim.go). " +
		"Même donnée, PAS une source distincte ; la couverture partielle ci-dessus est la nature " +
		"du flux (réplication au changement), pas un décodage plus fin à trouver.")
}
