package main

// cadence.go — la CADENCE DE TIR reelle, lue dans le tag `weap`.
//
// A QUOI CA SERT. Enchainer des variantes de tir a une cadence inventee donne une rafale
// qui ne ressemble pas au jeu. La vraie valeur est dans le tag : tableau `barrels`, struct
// inline `firing`, champ `_25` « rounds per second » — 8 octets, soit deux flottants
// (borne min, borne max).
//
// Element de `barrels` (calcul depuis weap.xml) :
//
//	+0  _D  flags                 4 octets
//	+4  _25 rounds per second     8 octets  <- min puis max, en float32
//
// COMME AILLEURS : on n'utilise PAS l'offset du plugin pour trouver le tableau. Il derive
// (mesure : « Weapon Fire Sounds » annonce a +4288, reel a +4352). On identifie le tableau
// par son CONTENU — un candidat n'est retenu que si ses deux flottants forment une cadence
// plausible et coherente (min <= max, dans une plage d'arme reelle).

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"

	"levelup/go-api/internal/himodule"
)

// Bornes de plausibilite d'une cadence, en coups par seconde. Larges a dessein : un lance-
// roquettes tourne sous 1, une mitrailleuse lourde depasse 15.
const (
	cadenceMin = 0.3
	cadenceMax = 60.0
)

// offsetRPS est la position de « rounds per second » dans un element de `barrels`.
const offsetRPS = 4

// rpsPlafond : au-dela, ce champ n'exprime plus une cadence mais la valeur non bridee que
// le moteur pose quand rien ne limite le tir ici.
//
// LE SEUIL VIENT D'UN ECART MESURE, pas d'une intuition. Sur les 33 armes resolues, les
// valeurs se repartissent en deux paquets nettement separes :
//
//	cadences reelles ... 1,11 (S7 Sniper) a 20,00 coups/s (MA5K Avenger)
//	--- aucune valeur entre 20 et 30 ---
//	sentinelle ......... ~30,00 coups/s, portee par l'Empaleur, le canon Gauss, la
//	                     tourelle a rayon — toutes des armes A UN COUP ou a faisceau,
//	                     dont la cadence reelle est bornee ailleurs (delai de tir,
//	                     recuperation), hors de ce champ.
//
// 25 tombe dans le vide entre les deux paquets. Le test n'est pas « == 30 » : la valeur
// stockee en float32 s'en ecarte parfois assez pour arrondir a 1800 coups/min sans y etre
// egale.
const rpsPlafond = 25.0

// Significative dit si la cadence lue represente vraiment un rythme de tir.
func (c cadenceArme) Significative() bool {
	return float64(c.Max) < rpsPlafond
}

type cadenceArme struct {
	OffsetChamp int
	Min         float32
	Max         float32
	Elements    int
}

// lireCadences rend les cadences plausibles trouvees dans un tag `weap`.
func lireCadences(data []byte) []cadenceArme {
	t, err := ouvrirTagWeap(data)
	if err != nil {
		return nil
	}
	racine, err := t.blocRacine()
	if err != nil {
		return nil
	}
	var out []cadenceArme
	for offChamp, bloc := range t.enfantsDe(racine) {
		abs, taille := t.blocAbs(bloc)
		if abs < 0 || taille < offsetRPS+8 || abs+offsetRPS+8 > len(data) {
			continue
		}
		bas := math.Float32frombits(binary.LittleEndian.Uint32(data[abs+offsetRPS:]))
		haut := math.Float32frombits(binary.LittleEndian.Uint32(data[abs+offsetRPS+4:]))
		if !cadencePlausible(bas, haut) {
			continue
		}
		out = append(out, cadenceArme{OffsetChamp: offChamp, Min: bas, Max: haut, Elements: taille})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OffsetChamp < out[j].OffsetChamp })
	return out
}

// cadencePlausible filtre les couples de flottants qui ne peuvent pas etre une cadence.
func cadencePlausible(bas, haut float32) bool {
	for _, v := range []float32{bas, haut} {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return false
		}
		if v < cadenceMin || v > cadenceMax {
			return false
		}
	}
	return bas <= haut
}

// sonderCadence affiche les candidats pour un `weap`, avec l'offset annonce par le plugin.
func sonderCadence(cheminModule string, gidWeap uint32) error {
	champs, err := champsPlugin()
	if err != nil {
		return err
	}
	attendu := -1
	if c, ok := trouverChamp(champs, "barrels"); ok {
		attendu = c.off
	}
	fmt.Printf("plugin   : tableau « barrels » annonce a +%d\n", attendu)

	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")
	for _, f := range m.Files("weap") {
		if f.GlobalID != gidWeap {
			continue
		}
		data, err := m.Extract(f)
		if err != nil {
			return err
		}
		cands := lireCadences(data)
		fmt.Printf("weap %08x : %d candidat(s) de cadence\n\n", gidWeap, len(cands))
		for _, c := range cands {
			marque := ""
			if c.OffsetChamp == attendu {
				marque = "  <- offset du plugin"
			}
			note := ""
			if !c.Significative() {
				note = "  [plafond moteur, pas une cadence]"
			}
			fmt.Printf("  +%-5d  %6.2f a %6.2f coups/s  (%6.0f a %6.0f coups/min)  bloc %d o%s%s\n",
				c.OffsetChamp, c.Min, c.Max, c.Min*60, c.Max*60, c.Elements, marque, note)
		}
		return nil
	}
	return fmt.Errorf("tag weap %08x absent", gidWeap)
}
