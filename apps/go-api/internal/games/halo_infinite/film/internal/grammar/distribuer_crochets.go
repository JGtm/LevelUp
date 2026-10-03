package grammar

// distribuer_crochets.go — LES CROCHETS DES CANAUX, FONDUS DANS L OBSERVATION DE LA MARCHE.
//
// Chaque canal pose ses crochets sur une observation a lui ([Canal.Brancher]) ; le distributeur
// recopie ensuite chaque crochet pose dans l observation de la marche. Un crochet qu un second canal
// pose aussi est refuse ([ErrCrochetDejaPose]) : il ecraserait le premier sans que rien ne le dise.
// Seuls les CROCHETS (champs de fonction) se fondent — les compteurs de l observation sont ceux de
// la marche, et un canal les lit dans le bilan ([BilanDeMarche.Obs]).

import (
	"fmt"
	"reflect"
)

// ErrCrochetDejaPose : deux canaux d une marche posent le meme crochet de l observation.
const ErrCrochetDejaPose = registryError("deux canaux posent le meme crochet de l observation")

// brancherLesCanaux fait poser a chaque canal ses crochets et rend l observation de la marche qui
// les porte tous.
func brancherLesCanaux(m *MarcheDistribuee, canaux []Canal) (*Observation, error) {
	obs := NouvelleObservation()
	for i, c := range canaux {
		propre := NouvelleObservation()
		c.Brancher(propre, m)
		if err := fondreLesCrochets(obs, propre); err != nil {
			return nil, fmt.Errorf("canal %d : %w", i, err)
		}
	}
	return obs, nil
}

// fondreLesCrochets recopie dans `obs` chaque crochet pose dans `propre`, et refuse un crochet que
// `obs` porte deja.
func fondreLesCrochets(obs, propre *Observation) error {
	dst, src := reflect.ValueOf(obs).Elem(), reflect.ValueOf(propre).Elem()
	for i := range src.NumField() {
		f := src.Field(i)
		if f.Kind() != reflect.Func || f.IsNil() {
			continue
		}
		if !dst.Field(i).IsNil() {
			return fmt.Errorf("%w : %s", ErrCrochetDejaPose, src.Type().Field(i).Name)
		}
		dst.Field(i).Set(f) // tous les crochets sont exportes : `TestLesCrochetsSontExportes`
	}
	return nil
}
