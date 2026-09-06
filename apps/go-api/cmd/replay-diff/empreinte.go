package main

// empreinte.go — REDUIRE UN ARTEFACT A UN JEU DE MESURES NOMMEES.
//
// L'empreinte est la seule forme comparable entre deux epoques : deux cuissons du meme match a
// deux schemas ne sont jamais egales octet pour octet (bornes affinees, ordre des tableaux,
// champs neufs), mais « combien d'actions de capture pour ce xuid » se compare toujours.
//
// DEUX PASSES, ET LA PREMIERE EST GENERIQUE. La passe generique mesure la TAILLE de chaque
// calque de premier niveau, quel qu'il soit — y compris un calque que le code d'aujourd'hui ne
// declare plus, et y compris un calque qui n'existe pas encore. C'est ce qui rend l'outil
// insensible au vieillissement : un axe neuf entre dans le rapport sans qu'on l'y ait inscrit.
// La passe specialisee, elle, descend dans les calques dont le PRODUIT depend du detail (les
// compteurs par joueur, les actions par famille, la couverture).

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
)

// Mesure est UNE valeur d'empreinte. Numerique quand la comparaison a un SENS d'ordre (une
// perte est un « moins »), textuelle sinon (un verdict, un oracle : un changement, pas une
// perte).
type Mesure struct {
	Num    float64 `json:"num,omitempty"`
	Txt    string  `json:"txt,omitempty"`
	EstNum bool    `json:"estNum"`
}

func (m Mesure) String() string {
	if m.EstNum {
		return strconv.FormatFloat(m.Num, 'g', -1, 64)
	}
	return m.Txt
}

// Empreinte est l'ensemble des mesures d'un artefact, indexees par « axe/metrique ».
type Empreinte struct {
	Schema  int               `json:"schema"`
	MatchID string            `json:"matchId"`
	Mesures map[string]Mesure `json:"mesures"`
}

// cle assemble l'index d'une mesure. L'axe est la premiere composante : c'est lui qui groupe
// le rapport.
func cle(axe, metrique string) string { return axe + "/" + metrique }

func (e *Empreinte) num(axe, metrique string, v float64) {
	e.Mesures[cle(axe, metrique)] = Mesure{Num: v, EstNum: true}
}

func (e *Empreinte) txt(axe, metrique, v string) {
	e.Mesures[cle(axe, metrique)] = Mesure{Txt: v}
}

// incr additionne dans une mesure numerique (compteurs par famille, par joueur).
func (e *Empreinte) incr(axe, metrique string, v float64) {
	m := e.Mesures[cle(axe, metrique)]
	e.Mesures[cle(axe, metrique)] = Mesure{Num: m.Num + v, EstNum: true}
}

// lireDocument lit un artefact SANS le typer. `UseNumber` garde les entiers exacts : un
// compteur de 2 533 274 815 845 110 passe par `float64` sans perte visible, mais un xuid lu
// comme nombre flottant deviendrait une identite fausse.
func lireDocument(path string) (map[string]any, error) {
	f, err := os.Open(path) //nolint:gosec // chemin fourni par l'operateur du CLI
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	dec := json.NewDecoder(f)
	dec.UseNumber()
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%s : %w", path, err)
	}
	return doc, nil
}

// Empreindre calcule l'empreinte complete d'un artefact.
func Empreindre(doc map[string]any) Empreinte {
	e := Empreinte{Mesures: map[string]Mesure{}}
	if n, ok := nombre(doc["schemaVersion"]); ok {
		e.Schema = int(n)
	}
	if s, ok := doc["matchId"].(string); ok {
		e.MatchID = s
	}
	passeGenerique(&e, doc)
	passeSpecialisee(&e, doc)
	return e
}

// passeGenerique mesure la taille de chaque calque de premier niveau et la valeur de chaque
// scalaire. Elle ne connait AUCUN nom de champ — c'est sa raison d'etre.
func passeGenerique(e *Empreinte, doc map[string]any) {
	for k, v := range doc {
		axe := axeDe(k)
		switch t := v.(type) {
		case []any:
			e.num(axe, k+"/n", float64(len(t)))
		case map[string]any:
			e.num(axe, k+"/n", float64(len(t)))
		case json.Number:
			f, err := t.Float64()
			if err != nil {
				e.txt(axe, k, t.String())
				continue
			}
			e.num(axe, k, f)
		case string:
			e.txt(axe, k, t)
		case bool:
			e.txt(axe, k, strconv.FormatBool(t))
		}
	}
}

// --- accesseurs tolerants ---------------------------------------------------------------
//
// Ils rendent TOUJOURS une valeur utilisable : sur un artefact d'un autre schema, un champ
// peut manquer, porter un autre type, ou etre `null`. Un acces qui panique ferait perdre tout
// le balayage pour un artefact malforme.

func tableau(v any) []any {
	t, _ := v.([]any)
	return t
}

func objet(v any) map[string]any {
	o, _ := v.(map[string]any)
	return o
}

func nombre(v any) (float64, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	f, err := n.Float64()
	if err != nil {
		return 0, false
	}
	return f, true
}

func chaine(v any) string {
	s, _ := v.(string)
	return s
}

// compte additionne un champ numerique sur tous les elements d'un tableau.
func compte(items []any, champ string) float64 {
	var total float64
	for _, it := range items {
		if n, ok := nombre(objet(it)[champ]); ok {
			total += n
		}
	}
	return total
}

// compteLongueurs additionne la longueur d'un champ tableau sur tous les elements.
func compteLongueurs(items []any, champ string) float64 {
	var total float64
	for _, it := range items {
		total += float64(len(tableau(objet(it)[champ])))
	}
	return total
}

// comptePresents compte les elements dont un champ est present ET non vide.
func comptePresents(items []any, champ string) float64 {
	var total float64
	for _, it := range items {
		v, ok := objet(it)[champ]
		if !ok || v == nil {
			continue
		}
		if s, estTexte := v.(string); estTexte && s == "" {
			continue
		}
		total++
	}
	return total
}

// distincts rend les valeurs textuelles distinctes d'un champ, triees.
func distincts(items []any, champ string) []string {
	vus := map[string]bool{}
	for _, it := range items {
		if s := chaine(objet(it)[champ]); s != "" {
			vus[s] = true
		}
	}
	out := make([]string, 0, len(vus))
	for s := range vus {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// aplatir descend recursivement dans un objet et pose une mesure par FEUILLE, sous un chemin
// pointe. C'est ce qui rend `coverage` et `bombStats` comparables sans en connaitre la forme
// — et donc encore comparables quand elle change.
func aplatir(e *Empreinte, axe, prefixe string, v any, profondeur int) {
	if profondeur > 8 {
		return
	}
	switch t := v.(type) {
	case map[string]any:
		for k, sous := range t {
			aplatir(e, axe, prefixe+"."+k, sous, profondeur+1)
		}
	case []any:
		e.num(axe, prefixe+"/n", float64(len(t)))
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			e.txt(axe, prefixe, t.String())
			return
		}
		e.num(axe, prefixe, f)
	case string:
		e.txt(axe, prefixe, t)
	case bool:
		e.txt(axe, prefixe, strconv.FormatBool(t))
	}
}
