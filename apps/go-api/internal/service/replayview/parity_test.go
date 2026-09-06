// parity_test.go — LE DOCUMENT STOCKE ET LE DOCUMENT SERVI, CHAMP PAR CHAMP.
//
// CE QUE CE FICHIER FERME. Separer le format d'artefact du contrat public retire au
// compilateur la seule chose qui les tenait alignes : ils etaient le MEME type. Sans garde,
// un calque ajoute a la cuisson n'atteindrait plus jamais le client, et personne ne le
// verrait — le contrat resterait vert, la page resterait muette. Ces trois tests remplacent
// la contrainte de compilation par une contrainte ECRITE :
//
//  1. tout champ exporte du document STOCKE a une decision (servi, ou inscrit dans la liste
//     datee `champsNonServis` avec sa justification) ;
//  2. tout champ du document SERVI a une source dans le document stocke ;
//  3. la projection COPIE effectivement — un document stocke rempli de valeurs distinctes
//     se serialise a l'octet pres comme sa projection, une fois retires les champs de la
//     liste. Un champ oublie dans un `to*` est attrape ici, pas en production.
package replayview_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/service/replayview"
)

// champsNonServis liste les champs du document STOCKE que l'API ne publie pas, par
// `Type.Champ`, avec la DATE de la decision et son motif. La liste est vide au 2026-09-05 :
// la separation du lot B laisse la forme de fil strictement inchangee. Toute entree future
// se justifie ici, pas dans un commit qui retire silencieusement un champ du corps.
var champsNonServis = map[string]string{}

// racines : les trois corps de route derives du monde `analysis/replay`.
var racines = []struct {
	nom    string
	stocke reflect.Type
	servi  reflect.Type
}{
	{"ReplayDocument", reflect.TypeOf(replay.ReplayDocument{}), reflect.TypeOf(replaydoc.ReplayDocument{})},
	{"MapBackground", reflect.TypeOf(replay.MapBackground{}), reflect.TypeOf(replaydoc.MapBackground{})},
	{"MapCalloutsEntry", reflect.TypeOf(replay.MapCalloutsEntry{}), reflect.TypeOf(replaydoc.MapCalloutsEntry{})},
}

// TestChaqueChampStockeAUneDecision : rien ne sort du contrat par omission.
func TestChaqueChampStockeAUneDecision(t *testing.T) {
	servis := indexerTypes(racines[0].servi, racines[1].servi, racines[2].servi)
	vus := map[string]bool{}
	parcourir(racinesStockees(), func(ts reflect.Type) {
		jumeau, ok := servis[ts.Name()]
		if !ok {
			t.Errorf("type stocke %q : AUCUN jumeau servi — un calque entier sortirait du "+
				"contrat sans qu'aucun test ne le dise", ts.Name())
			return
		}
		for i := 0; i < ts.NumField(); i++ {
			f := ts.Field(i)
			if f.PkgPath != "" {
				continue // champ non exporte : jamais serialise
			}
			cle := ts.Name() + "." + f.Name
			vus[cle] = true
			if motif, exclu := champsNonServis[cle]; exclu {
				if strings.TrimSpace(motif) == "" {
					t.Errorf("%s est inscrit non servi SANS justification datee", cle)
				}
				continue
			}
			g, ok := jumeau.FieldByName(f.Name)
			if !ok {
				t.Errorf("%s est publie par l'artefact et ABSENT du document servi, sans entree "+
					"dans champsNonServis — soit le contrat le porte, soit la decision s'ecrit", cle)
				continue
			}
			if a, b := f.Tag.Get("json"), g.Tag.Get("json"); a != b {
				t.Errorf("%s : tag json %q cote stocke, %q cote servi — le client lirait un autre "+
					"nom que celui que la cuisson ecrit", cle, a, b)
			}
		}
	})
	for cle := range champsNonServis {
		if !vus[cle] {
			t.Errorf("champsNonServis cite %q, que le document stocke ne publie plus — une "+
				"exception qui survit a son champ finit par en couvrir un autre", cle)
		}
	}
}

// TestChaqueChampServiAUneSource : le contrat ne promet rien que la cuisson n'ecrive.
func TestChaqueChampServiAUneSource(t *testing.T) {
	stockes := indexerTypes(racinesStockees()...)
	parcourir([]reflect.Type{racines[0].servi, racines[1].servi, racines[2].servi}, func(ts reflect.Type) {
		source, ok := stockes[ts.Name()]
		if !ok {
			t.Errorf("type servi %q : AUCUNE source stockee — le contrat promet une forme que "+
				"rien ne remplit", ts.Name())
			return
		}
		for i := 0; i < ts.NumField(); i++ {
			f := ts.Field(i)
			if f.PkgPath != "" {
				continue
			}
			if _, ok := source.FieldByName(f.Name); !ok {
				t.Errorf("%s.%s est decrit par le contrat et n'a AUCUNE source dans l'artefact — "+
					"une promesse morte que le client lira comme optionnelle", ts.Name(), f.Name)
			}
		}
	})
}

// TestProjectionCopieChaqueChamp : la projection est exhaustive, sur les TROIS classes de
// documents. Une seule ne suffit pas, et la revue adversariale du 2026-09-05 l'a prouve en
// jouant deux mutations que le filet d'origine laissait passer :
//
//   - un champ `int` devenu `*int` cote servi (`StartFrame: &v.StartFrame`) : la valeur ZERO
//     se tait sous `omitempty` cote stocke et parle sous un pointeur cote servi. Le document
//     peuple ne porte aucun zero, il ne pouvait pas le voir ;
//   - une tranche nulle normalisee en tranche vide (`append([]T{}, sliceOf(...)...)`, ce
//     qu'ecrirait une boucle a la main) : `null` devient `[]`, visible du client sur les
//     champs sans `omitempty`. Le document peuple n'a aucune tranche nulle.
//
// D'ou les trois classes, et le BALAYAGE de la frontiere : une tranche nulle a la racine ne
// dit rien d'une tranche nulle au troisieme niveau, et c'est au troisieme niveau que la
// mutation a ete jouee.
func TestProjectionCopieChaqueChamp(t *testing.T) {
	t.Run("peuple", func(t *testing.T) { comparerLesTroisRacines(t, peuple()) })

	// Frontiere 0 = le document ZERO au sens strict : tout scalaire a zero, toute tranche,
	// toute map et tout pointeur a nil. Les frontieres suivantes descendent d'un cran a
	// chaque tour, jusqu'au squelette complet a feuilles zero.
	for d := 0; d <= profondeurMax; d++ {
		t.Run(fmt.Sprintf("nul_sous_%d", d), func(t *testing.T) {
			comparerLesTroisRacines(t, aPlat(d, conteneurNul))
		})
		t.Run(fmt.Sprintf("vide_sous_%d", d), func(t *testing.T) {
			comparerLesTroisRacines(t, aPlat(d, conteneurVide))
		})
	}
}

// comparerLesTroisRacines construit les trois corps de route avec ce remplisseur et confronte
// chacun a sa projection.
func comparerLesTroisRacines(t *testing.T, f *remplisseur) {
	t.Helper()
	doc := reflect.New(racines[0].stocke).Elem()
	f.remplir(doc, 0)
	bg := reflect.New(racines[1].stocke).Elem()
	f.remplir(bg, 0)
	co := reflect.New(racines[2].stocke).Elem()
	f.remplir(co, 0)

	stocke := doc.Interface().(replay.ReplayDocument)
	fond := bg.Interface().(replay.MapBackground)
	zones := co.Interface().(replay.MapCalloutsEntry)

	comparer(t, "ReplayDocument", stocke, replayview.FromArtifact(stocke), racines[0].stocke)
	comparer(t, "MapBackground", fond, *replayview.MapBackgroundOf(&fond), racines[1].stocke)
	comparer(t, "MapCalloutsEntry", zones, *replayview.MapCalloutsOf(&zones), racines[2].stocke)
}

// TestProjectionPreserveLaNullite : une tranche nulle reste nulle, une tranche vide reste
// vide. Sur les champs sans `omitempty` la distinction est visible du client (`null` contre
// `[]`), et une projection qui « normalise » changerait le corps sans le dire.
func TestProjectionPreserveLaNullite(t *testing.T) {
	for _, c := range []struct {
		nom    string
		tracks []replay.Track
		veut   string
	}{
		{"nulle", nil, "null"},
		{"vide", []replay.Track{}, "[]"},
	} {
		t.Run(c.nom, func(t *testing.T) {
			out := replayview.FromArtifact(replay.ReplayDocument{Tracks: c.tracks})
			raw, err := json.Marshal(out)
			if err != nil {
				t.Fatalf("serialisation: %v", err)
			}
			if !strings.Contains(string(raw), `"tracks":`+c.veut) {
				t.Errorf("tranche %s : le corps ne porte pas \"tracks\":%s — %s", c.nom, c.veut, raw)
			}
		})
	}
}

// TestContractVersionEstUneConstantePropre : les deux versions (servie et stockee) sont
// parties de la meme valeur le 2026-09-05 et ont vocation a diverger. Ce test ne verrouille
// PAS leur egalite — ce serait annuler la separation. Il verrouille deux choses : la version
// servie est un nombre POSE, et le champ `schemaVersion` du corps continue de porter la
// version de l'ARTEFACT LU, celle qui pilote la re-cuisson du parc.
func TestContractVersionEstUneConstantePropre(t *testing.T) {
	if replaydoc.ContractVersion <= 0 {
		t.Fatalf("ContractVersion = %d : une version de contrat se pose, elle ne s'improvise pas",
			replaydoc.ContractVersion)
	}
	// Le champ `schemaVersion` du corps porte la version STOCKEE de l'artefact lu, pas
	// ContractVersion : c'est elle qui dit au parc « a re-cuire ».
	out := replayview.FromArtifact(replay.ReplayDocument{SchemaVersion: 7})
	if out.SchemaVersion != 7 {
		t.Errorf("schemaVersion servi = %d, attendu 7 (la version de l'ARTEFACT LU) : le champ "+
			"pilote la re-cuisson, il ne doit jamais etre remplace par la version de contrat",
			out.SchemaVersion)
	}
}

// ---------------------------------------------------------------------------
// Parcours des deux graphes de types
// ---------------------------------------------------------------------------

func racinesStockees() []reflect.Type {
	return []reflect.Type{racines[0].stocke, racines[1].stocke, racines[2].stocke}
}

// parcourir visite chaque type STRUCT nomme atteignable depuis les racines, une seule fois.
func parcourir(racines []reflect.Type, visite func(reflect.Type)) {
	vus := map[reflect.Type]bool{}
	var walk func(t reflect.Type)
	walk = func(t reflect.Type) {
		switch t.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array:
			walk(t.Elem())
			return
		case reflect.Map:
			walk(t.Key())
			walk(t.Elem())
			return
		}
		if vus[t] || t.Kind() != reflect.Struct || t.PkgPath() == "time" {
			return
		}
		vus[t] = true
		visite(t)
		for i := 0; i < t.NumField(); i++ {
			walk(t.Field(i).Type)
		}
	}
	for _, r := range racines {
		walk(r)
	}
}

// indexerTypes nomme chaque type struct atteignable depuis les racines.
func indexerTypes(racines ...reflect.Type) map[string]reflect.Type {
	out := map[string]reflect.Type{}
	parcourir(racines, func(t reflect.Type) { out[t.Name()] = t })
	return out
}

// ---------------------------------------------------------------------------
// Comparaison des deux corps JSON
// ---------------------------------------------------------------------------

func comparer(t *testing.T, nom string, stocke, servi any, typeStocke reflect.Type) {
	t.Helper()
	a := elaguer(arbre(t, stocke), typeStocke)
	b := arbre(t, servi)
	if reflect.DeepEqual(a, b) {
		return
	}
	ja, _ := json.MarshalIndent(a, "", "  ")
	jb, _ := json.MarshalIndent(b, "", "  ")
	t.Errorf("%s : la projection ne rend pas le meme corps que l'artefact.\n"+
		"premier ecart : %s\n", nom, premierEcart(a, b, ""))
	if testing.Verbose() {
		t.Logf("stocke:\n%s\nservi:\n%s", ja, jb)
	}
}

func arbre(t *testing.T, v any) any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("serialisation: %v", err)
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("relecture: %v", err)
	}
	return out
}

// elaguer retire de l arbre JSON du document STOCKE les champs que champsNonServis exclut,
// en se guidant sur le type Go (l'arbre JSON seul ne porte pas les noms de types).
func elaguer(node any, t reflect.Type) any {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		obj, ok := node.(map[string]any)
		if !ok {
			return node
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			nom := nomJSON(f)
			if nom == "" {
				continue
			}
			if _, exclu := champsNonServis[t.Name()+"."+f.Name]; exclu {
				delete(obj, nom)
				continue
			}
			if sub, present := obj[nom]; present {
				obj[nom] = elaguer(sub, f.Type)
			}
		}
		return obj
	case reflect.Slice, reflect.Array:
		items, ok := node.([]any)
		if !ok {
			return node
		}
		for i := range items {
			items[i] = elaguer(items[i], t.Elem())
		}
		return items
	case reflect.Map:
		obj, ok := node.(map[string]any)
		if !ok {
			return node
		}
		for k, v := range obj {
			obj[k] = elaguer(v, t.Elem())
		}
		return obj
	}
	return node
}

func nomJSON(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "-" {
		return ""
	}
	if i := strings.IndexByte(tag, ','); i >= 0 {
		tag = tag[:i]
	}
	if tag == "" {
		return f.Name
	}
	return tag
}

// premierEcart nomme le premier chemin JSON qui differe — un diff de 2 000 lignes ne se lit
// pas, un chemin se corrige.
func premierEcart(a, b any, chemin string) string {
	switch va := a.(type) {
	case map[string]any:
		vb, ok := b.(map[string]any)
		if !ok {
			return chemin + " : objet d'un cote, " + fmt.Sprintf("%T", b) + " de l'autre"
		}
		cles := map[string]bool{}
		for k := range va {
			cles[k] = true
		}
		for k := range vb {
			cles[k] = true
		}
		var tri []string
		for k := range cles {
			tri = append(tri, k)
		}
		sort.Strings(tri)
		for _, k := range tri {
			xa, oka := va[k]
			xb, okb := vb[k]
			switch {
			case oka && !okb:
				return chemin + "." + k + " : present cote stocke, ABSENT cote servi"
			case !oka && okb:
				return chemin + "." + k + " : present cote servi, absent cote stocke"
			}
			if e := premierEcart(xa, xb, chemin+"."+k); e != "" {
				return e
			}
		}
		return ""
	case []any:
		vb, ok := b.([]any)
		if !ok || len(va) != len(vb) {
			return chemin + " : tranches de longueurs differentes"
		}
		for i := range va {
			if e := premierEcart(va[i], vb[i], fmt.Sprintf("%s[%d]", chemin, i)); e != "" {
				return e
			}
		}
		return ""
	}
	if !reflect.DeepEqual(a, b) {
		return fmt.Sprintf("%s : %v cote stocke, %v cote servi", chemin, a, b)
	}
	return ""
}

// ---------------------------------------------------------------------------
// Remplissage : trois classes de documents, et pourquoi il en faut trois
// ---------------------------------------------------------------------------

// etatConteneur : l'etat que prennent tranches, maps et pointeurs A LA FRONTIERE.
type etatConteneur int

const (
	conteneurPeuple etatConteneur = iota // un element (deux pour la classe peuplee)
	conteneurNul                         // nil
	conteneurVide                        // longueur 0, NON nil
)

// profondeurMax borne la recursion. Profondeur REELLE mesuree du graphe stocke : 10
// (`ReplayDocument.ScoreTimeline[].Players[].Score.Rounds[].Points[].T`) — deux niveaux de
// marge. Un calque a deux imbrications de plus rendrait le filet aveugle EN SILENCE : si la
// mesure remonte, ce nombre monte avec elle.
const profondeurMax = 12

// remplisseur construit un document de test. Deux boutons, et c'est leur COMBINAISON qui
// donne au filet sa portee :
//
//   - `zero` : les scalaires restent a leur valeur zero au lieu de porter une valeur
//     distincte. C'est ce bouton qui rend visible un `int` devenu `*int` — la valeur 0 se
//     tait sous `omitempty` cote stocke et parle sous un pointeur cote servi.
//   - `frontiere` + `bord` : jusqu'a `frontiere` les conteneurs portent `elements` entrees
//     (le parcours descend), a partir d'elle ils prennent l'etat `bord`. Balayer la
//     frontiere de 0 a profondeurMax place tour a tour CHAQUE niveau du graphe au bord :
//     c'est la seule facon de mettre une tranche nulle SOUS un conteneur qui existe, donc
//     d'attraper une normalisation `nil` -> `[]` ailleurs qu'a la racine.
type remplisseur struct {
	n         int
	zero      bool
	frontiere int
	bord      etatConteneur
	elements  int
}

// peuple rend le remplisseur de la classe historique : chaque feuille porte une valeur
// distincte, chaque conteneur deux entrees, aucun pointeur nil.
func peuple() *remplisseur {
	return &remplisseur{frontiere: profondeurMax + 1, bord: conteneurPeuple, elements: 2}
}

// aPlat rend un remplisseur a scalaires ZERO dont les conteneurs prennent `bord` a partir de
// `frontiere`.
func aPlat(frontiere int, bord etatConteneur) *remplisseur {
	return &remplisseur{zero: true, frontiere: frontiere, bord: bord, elements: 1}
}

func (r *remplisseur) suivant() int { r.n++; return r.n }

// auBord dit si un conteneur rencontre a cette profondeur prend l'etat `bord`.
func (r *remplisseur) auBord(profondeur int) bool { return profondeur >= r.frontiere }

func (r *remplisseur) remplir(v reflect.Value, profondeur int) {
	if profondeur > profondeurMax {
		return
	}
	switch v.Kind() {
	case reflect.Bool:
		if !r.zero {
			v.SetBool(true)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if !r.zero {
			v.SetInt(int64(r.suivant()))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if !r.zero {
			v.SetUint(uint64(r.suivant()))
		}
	case reflect.Float32, reflect.Float64:
		if !r.zero {
			v.SetFloat(float64(r.suivant()) + 0.5)
		}
	case reflect.String:
		if !r.zero {
			v.SetString(fmt.Sprintf("v%d", r.suivant()))
		}
	case reflect.Pointer:
		r.remplirPointeur(v, profondeur)
	case reflect.Slice:
		r.remplirTranche(v, profondeur)
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			r.remplir(v.Index(i), profondeur+1)
		}
	case reflect.Map:
		r.remplirMap(v, profondeur)
	case reflect.Struct:
		r.remplirStruct(v, profondeur)
	}
}

// remplirPointeur : nil au bord quand le bord est `nul`, alloue sinon. Un pointeur n'a pas
// d'etat « vide » — `conteneurVide` l'alloue donc sur une valeur zero, ce qui est
// precisement la forme qui distingue un scalaire nu d'un scalaire pointe.
func (r *remplisseur) remplirPointeur(v reflect.Value, profondeur int) {
	if r.auBord(profondeur) && r.bord == conteneurNul {
		return // deja nil
	}
	p := reflect.New(v.Type().Elem())
	r.remplir(p.Elem(), profondeur+1)
	v.Set(p)
}

func (r *remplisseur) remplirTranche(v reflect.Value, profondeur int) {
	n := r.elements
	if r.auBord(profondeur) {
		switch r.bord {
		case conteneurNul:
			return // laisser nil : le corps porte `null`
		case conteneurVide:
			n = 0 // longueur 0 mais NON nil : le corps porte `[]`
		case conteneurPeuple:
		}
	}
	s := reflect.MakeSlice(v.Type(), n, n)
	for i := 0; i < s.Len(); i++ {
		r.remplir(s.Index(i), profondeur+1)
	}
	v.Set(s)
}

func (r *remplisseur) remplirMap(v reflect.Value, profondeur int) {
	n := r.elements
	if r.auBord(profondeur) {
		switch r.bord {
		case conteneurNul:
			return
		case conteneurVide:
			n = 0
		case conteneurPeuple:
		}
	}
	m := reflect.MakeMap(v.Type())
	for i := 0; i < n; i++ {
		k := reflect.New(v.Type().Key()).Elem()
		r.remplir(k, profondeur+1)
		if r.zero {
			// Cles toutes vides : une map a une entree suffit, et le corps reste
			// deterministe des deux cotes.
			k.SetString(fmt.Sprintf("k%d", i))
		}
		e := reflect.New(v.Type().Elem()).Elem()
		r.remplir(e, profondeur+1)
		m.SetMapIndex(k, e)
	}
	v.Set(m)
}

func (r *remplisseur) remplirStruct(v reflect.Value, profondeur int) {
	if v.Type() == reflect.TypeOf(time.Time{}) {
		if !r.zero {
			v.Set(reflect.ValueOf(time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)))
		}
		return
	}
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).CanSet() {
			r.remplir(v.Field(i), profondeur+1)
		}
	}
}
