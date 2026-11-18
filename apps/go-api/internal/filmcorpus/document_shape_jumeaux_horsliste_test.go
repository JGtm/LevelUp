package filmcorpus

// document_shape_jumeaux_horsliste_test.go — LA REGLE DES FORMES JUMELLES MORD ENCORE HORS DE LA LISTE.
//
// `TestDocumentShapeTwinsAgree` retire de la forme SERVIE les champs et les types de la liste datee
// `testutil.ReplayChampsServisSeuls` / `ReplayTypesServisSeuls`. Ce fichier prouve, sur des types de test
// de meme nom (`Doc`, `Lot`) dans deux fonctions, que ce retrait est exact : il couvre ce que la liste
// nomme et rien d'autre.

import (
	"reflect"
	"strings"
	"testing"
)

// docStocke et docServi sont deux types `Doc` de meme nom : le rendu de forme designe un type par son nom.
// Le servi porte `Extra`, un champ de trop, et `Lots`, dont le type `Lot` n'a pas de jumeau stocke.
func docStocke() reflect.Type {
	type Doc struct {
		ID string `json:"id"`
	}
	return reflect.TypeFor[Doc]()
}

func docServi() reflect.Type {
	type Lot struct {
		N int `json:"n"`
	}
	type Doc struct {
		ID    string `json:"id"`
		Extra string `json:"extra,omitempty"`
		Lots  []Lot  `json:"lots,omitempty"`
	}
	return reflect.TypeFor[Doc]()
}

func TestFormesJumellesChampServiHorsListeEchoue(t *testing.T) {
	msg := ecartDesFormesJumelles(docStocke(), docServi(), nil, nil)
	if msg == "" {
		t.Fatal("un champ servi sans jumeau stocke et hors liste doit faire echouer la comparaison")
	}
	if !strings.Contains(msg, "divergent") {
		t.Errorf("message inattendu : %s", msg)
	}
}

func TestFormesJumellesListeNeCouvreQueSesEntrees(t *testing.T) {
	champs := map[string]string{"Doc.Extra": "2026-01-01 : test"}
	types := map[string]string{"Lot": "2026-01-01 : test"}

	t.Run("champ seul liste : le type Lot echoue encore", func(t *testing.T) {
		// `Doc.Lots` reste decrit et son type `Lot` n'a pas de jumeau : divergence.
		if ecartDesFormesJumelles(docStocke(), docServi(), champs, nil) == "" {
			t.Error("le type servi seul `Lot` non liste doit faire echouer la comparaison")
		}
	})
	t.Run("type seul liste : le champ Extra echoue encore", func(t *testing.T) {
		if ecartDesFormesJumelles(docStocke(), docServi(), nil, types) == "" {
			t.Error("le champ servi seul `Doc.Extra` non liste doit faire echouer la comparaison")
		}
	})
	t.Run("champs et types listes : jumeaux", func(t *testing.T) {
		// `Doc.Lots` est un champ servi seul aussi : sans lui, le type `Lot` retire laisserait une ligne orpheline.
		tous := map[string]string{"Doc.Extra": "2026-01-01 : test", "Doc.Lots": "2026-01-01 : test"}
		if msg := ecartDesFormesJumelles(docStocke(), docServi(), tous, types); msg != "" {
			t.Errorf("les entrees listees doivent suffire : %s", msg)
		}
	})
	t.Run("la liste ne s'applique qu'a la forme servie", func(t *testing.T) {
		// `Doc.ID` existe des deux cotes : le nommer dans la liste le retire du servi seul, et le stocke
		// qui le porte encore diverge. Une entree ne peut donc pas masquer un champ jumeau.
		tous := map[string]string{"Doc.Extra": "2026-01-01 : test", "Doc.Lots": "2026-01-01 : test", "Doc.ID": "2026-01-01 : test"}
		if ecartDesFormesJumelles(docStocke(), docServi(), tous, types) == "" {
			t.Error("nommer un champ jumeau dans la liste ne doit pas le masquer cote stocke")
		}
	})
}

// TestInventaireServiVoitLesChampsEtLesTypes : ce que le verificateur partage confronte aux listes.
func TestInventaireServiVoitLesChampsEtLesTypes(t *testing.T) {
	champs, types := inventaireServi(docServi())
	for _, want := range []string{"Doc.ID", "Doc.Extra", "Doc.Lots", "Lot.N"} {
		if !champs[want] {
			t.Errorf("champ %q absent de l'inventaire", want)
		}
	}
	for _, want := range []string{"Doc", "Lot"} {
		if !types[want] {
			t.Errorf("type %q absent de l'inventaire", want)
		}
	}
}
