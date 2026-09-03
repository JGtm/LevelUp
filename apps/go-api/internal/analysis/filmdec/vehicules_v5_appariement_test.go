package filmdec

// vehicules_v5_appariement_test.go — LOT V5 : APPARIER CHAQUE ÉPISODE À SON VÉHICULE, puis
// chercher le lien SANS supposer un décalage stable.
//
// POURQUOI CETTE ÉTAPE. Le balayage à décalage fixe (`TestV5Balayage`) échoue, et sa faiblesse
// est identifiée : les bornes de record rendues par `WalkKeyframeWorld` ne sont pas fiables
// (le balayeur SAUTE les records dont `Field26` n'est pas nul — cf. keyframe_record_walk.go —
// donc la « fin » d'un record est parfois la fin de DEUX records), et le contenu qui précède
// un champ est de longueur variable. Un décalage fixe ne peut donc pas être exigé a priori.
//
// LA QUESTION DEVIENT SANS DÉCALAGE : le slot du véhicule occupé apparaît-il QUELQUE PART
// dans le record d'image-clé de son occupant, plus souvent que dans le record d'un piéton au
// MÊME instant ? Cette question exige de savoir QUEL véhicule — d'où l'appariement.
//
// L'APPARIEMENT SE FAIT PAR LA POSITION, et il est vérifiable : à l'instant de la SORTIE,
// l'occupant réapparaît dans le flux de position AU CONTACT du véhicule qu'il quitte (c'est
// le modèle V1a.4, déjà validé : le trou se referme à la sortie 90,7 % contre 0 % au témoin).
// Le véhicule apparié est donc le `ti=40` le plus proche du premier échantillon de l'occupant
// après le trou.
//
//	CGO_ENABLED=0 V5_ROOT=<cache> V5_FILMS=... \
//	  go test ./internal/analysis/filmdec/ -run TestV5Appariement -v -timeout 120m

import (
	"fmt"
	"math"
	"sort"
	"testing"
)

// v5ContactUS est la fenêtre temporelle dans laquelle on cherche la position d'un véhicule
// pour l'apparier à la réapparition de l'occupant (1 s : le flux véhicule est dense).
const v5ContactUS = 1_000_000

// v5EchQ est un échantillon de position en QUANTA (aucune borne de carte requise).
type v5EchQ struct {
	Slot uint32
	TS   uint64
	Q    [3]uint32
}

// v5PositionsBande décode les positions (quanta) des slots d'une bande donnée.
func v5PositionsBande(dir string, bande map[uint32]bool) ([]v5EchQ, error) {
	ps, err := ScanFilmBipedPositionsForBand(dir, bande, ScanFilmOptions{QuantaOnly: true})
	if err != nil {
		return nil, err
	}
	out := make([]v5EchQ, 0, len(ps))
	for _, p := range ps {
		out = append(out, v5EchQ{Slot: p.Slot, TS: p.TimestampUS, Q: p.Q})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TS < out[j].TS })
	return out, nil
}

// v5DistQ est la distance euclidienne en unités de quantum (monotone avec la distance réelle
// à l'échelle d'une carte : les largeurs d'axe sont constantes par film).
func v5DistQ(a, b [3]uint32) float64 {
	dx := float64(a[0]) - float64(b[0])
	dy := float64(a[1]) - float64(b[1])
	dz := float64(a[2]) - float64(b[2])
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// v5EpisodeApparie est un épisode auquel un véhicule a été attribué.
type v5EpisodeApparie struct {
	v5Episode
	Vehicule uint32
	Dist     float64
	Ok       bool
}

// v5Apparier attribue à chaque épisode le véhicule le plus proche de la réapparition de son
// occupant.
func v5Apparier(dir string, eps []v5Episode) ([]v5EpisodeApparie, error) {
	chunks := v5TousChunks(dir)
	bandeB := bipedSlotBand(dir, chunks)
	bandeV := worldObjectSlotBand(dir, CountFilmChunks(dir), v5VehiculeTI)
	// Un slot qui est dans les deux bandes ne peut pas servir de cible : on ne retient que
	// les slots véhicule PURS (les deux bandes se recouvrent partiellement).
	for s := range bandeB {
		delete(bandeV, s)
	}
	posB, err := v5PositionsBande(dir, bandeB)
	if err != nil {
		return nil, err
	}
	posV, err := v5PositionsBande(dir, bandeV)
	if err != nil {
		return nil, err
	}
	out := make([]v5EpisodeApparie, 0, len(eps))
	for _, e := range eps {
		ea := v5EpisodeApparie{v5Episode: e}
		reap, ok := v5PremierApres(posB, e.Slot, e.FinUS)
		if ok {
			best, bd := uint32(0), math.Inf(1)
			for _, v := range posV {
				if v5AbsDiff(v.TS, reap.TS) > v5ContactUS {
					continue
				}
				if d := v5DistQ(v.Q, reap.Q); d < bd {
					best, bd = v.Slot, d
				}
			}
			if !math.IsInf(bd, 1) {
				ea.Vehicule, ea.Dist, ea.Ok = best, bd, true
			}
		}
		out = append(out, ea)
	}
	return out, nil
}

// v5PremierApres rend le premier échantillon du slot à partir de `at`.
func v5PremierApres(ps []v5EchQ, slot uint32, at uint64) (v5EchQ, bool) {
	best, ok := v5EchQ{}, false
	for _, p := range ps {
		if p.Slot != slot || p.TS < at {
			continue
		}
		if !ok || p.TS < best.TS {
			best, ok = p, true
		}
	}
	return best, ok
}

// TestV5Appariement publie l'appariement épisode -> véhicule, PUIS le test de présence sans
// décalage : le slot du véhicule apparié apparaît-il dans le record d'image-clé de son
// occupant, contre le témoin des piétons au même instant ?
func TestV5Appariement(t *testing.T) {
	for _, dir := range v5Films(t) {
		v5AppariementUnFilm(t, dir)
	}
}

func v5AppariementUnFilm(t *testing.T, dir string) {
	t.Helper()
	eps, _, err := v5Episodes(dir)
	if err != nil {
		t.Logf("V5 APPARIEMENT %s : %v", dir, err)
		return
	}
	app, err := v5Apparier(dir, eps)
	if err != nil {
		t.Logf("V5 APPARIEMENT %s : %v", dir, err)
		return
	}
	n := 0
	for _, e := range app {
		if e.Ok {
			n++
		}
	}
	t.Logf("V5 APPARIEMENT %s — %d/%d épisodes appariés à un véhicule", dir, n, len(app))
	for i, e := range app {
		t.Logf("    ep%-2d slot=%-5d [%8.2f -> %8.2f] siège=%d  véhicule=%-5d dist=%.1f q  apparié=%v",
			i, e.Slot, float64(e.DebutUS)/1e6, float64(e.FinUS)/1e6, e.Seat, e.Vehicule, e.Dist, e.Ok)
	}
	v5Presence(t, dir, app)
}

// v5PresenceFenetre borne la fenêtre de bits lue dans un record d'image-clé, pour NORMALISER
// la comparaison : deux records de longueurs différentes n'ont pas la même chance de contenir
// une valeur donnée. 2 800 bits est la longueur médiane d'un record bipède (keyframe_loadout).
const v5PresenceFenetre = 2800

// v5Presence répond à la question sans décalage : le slot du véhicule apparié apparaît-il
// dans la fenêtre du record de son occupant ? Témoin : le MÊME slot cherché dans le record
// d'un piéton au MÊME instant (même valeur, même longueur de fenêtre — seul l'occupant change).
func v5Presence(t *testing.T, dir string, app []v5EpisodeApparie) {
	t.Helper()
	kfs := v5Keyframes(dir)
	type res struct{ posN, posT, negN, negT int }
	// Deux sens : occupant -> véhicule, et véhicule -> occupant.
	var oc, ve res
	for _, kf := range kfs {
		if len(kf) == 0 {
			continue
		}
		ts := kf[0].TS
		for _, e := range app {
			if !e.Ok || ts <= e.DebutUS || ts >= e.FinUS {
				continue
			}
			// Sens 1 : le record de l'occupant contient-il le slot du véhicule ?
			for _, r := range kf {
				if r.TI != v5BipedeTI {
					continue
				}
				touche := v5Contient(r, e.Vehicule)
				if uint32(r.Slot) == e.Slot {
					oc.posN++
					if touche {
						oc.posT++
					}
				} else {
					oc.negN++
					if touche {
						oc.negT++
					}
				}
			}
			// Sens 2 : le record du véhicule contient-il le slot de l'occupant ?
			for _, r := range kf {
				if r.TI != v5VehiculeTI {
					continue
				}
				touche := v5Contient(r, e.Slot)
				if uint32(r.Slot) == e.Vehicule {
					ve.posN++
					if touche {
						ve.posT++
					}
				} else {
					ve.negN++
					if touche {
						ve.negT++
					}
				}
			}
		}
	}
	t.Logf("  PRÉSENCE SANS DÉCALAGE (fenêtre %d bits, extracteurs %d)", v5PresenceFenetre, len(v5Extracteurs))
	t.Logf("    occupant -> véhicule : record de l'occupant %d/%d = %s   témoin (autres bipèdes) %d/%d = %s",
		oc.posT, oc.posN, v5Pct(oc.posT, oc.posN), oc.negT, oc.negN, v5Pct(oc.negT, oc.negN))
	t.Logf("    véhicule -> occupant : record du véhicule  %d/%d = %s   témoin (autres véhicules) %d/%d = %s",
		ve.posT, ve.posN, v5Pct(ve.posT, ve.posN), ve.negT, ve.negN, v5Pct(ve.negT, ve.negN))
}

// v5Contient dit si la valeur `cible` apparaît, sous l'un des extracteurs, à un décalage
// quelconque de la fenêtre du record.
func v5Contient(r v5KfRec, cible uint32) bool {
	long := r.Fin - r.BitStart
	if long > v5PresenceFenetre {
		long = v5PresenceFenetre
	}
	for _, ex := range v5Extracteurs {
		for d := 0; d+ex.Largeur <= long; d++ {
			if ex.Slot(kfReadBits(r.Payload, r.BitStart+d, ex.Largeur)) == cible {
				return true
			}
		}
	}
	return false
}

func v5Pct(a, b int) string {
	if b == 0 {
		return "n/a"
	}
	return formatPct(float64(a) / float64(b) * 100)
}

func formatPct(v float64) string {
	return fmt.Sprintf("%.1f %%", v)
}
