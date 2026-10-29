package halo_infinite

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/domain/highlightevent"

	filmhighlight "github.com/JGtm/LevelUp/libs/film/highlightevent"
)

// TestHighlightEventDepuisFilm_MemesChamps : les deux formes de l evenement de temps fort (celle
// du decodeur, celle de LevelUp) ont les memes champs, dans le meme ordre, avec les memes types et
// les memes tags. Un champ ajoute, retire, renomme ou retype d un seul cote fait rougir ce test :
// la conversion le perdrait en silence.
func TestHighlightEventDepuisFilm_MemesChamps(t *testing.T) {
	film := reflect.TypeFor[filmhighlight.HighlightEvent]()
	levelup := reflect.TypeFor[highlightevent.HighlightEvent]()
	if film.NumField() != levelup.NumField() {
		t.Fatalf("nombre de champs : film %d, LevelUp %d", film.NumField(), levelup.NumField())
	}
	for i := range film.NumField() {
		f, l := film.Field(i), levelup.Field(i)
		if f.Name != l.Name || f.Type != l.Type || f.Tag != l.Tag {
			t.Errorf("champ %d : film %s %s %q, LevelUp %s %s %q", i, f.Name, f.Type, f.Tag, l.Name, l.Type, l.Tag)
		}
	}
}

// TestHighlightEventDepuisFilm_AllerRetour : chaque champ est recopie. La valeur source est remplie
// par reflexion (aucun champ a zero), et la valeur convertie est comparee champ par champ par nom.
func TestHighlightEventDepuisFilm_AllerRetour(t *testing.T) {
	var src filmhighlight.HighlightEvent
	v := reflect.ValueOf(&src).Elem()
	for i := range v.NumField() {
		remplirNonNul(t, v.Field(i), i)
	}
	got := reflect.ValueOf(HighlightEventDepuisFilm(src))
	for i := range v.NumField() {
		nom := v.Type().Field(i).Name
		if !reflect.DeepEqual(got.FieldByName(nom).Interface(), v.Field(i).Interface()) {
			t.Errorf("champ %s : converti %v, source %v", nom, got.FieldByName(nom).Interface(), v.Field(i).Interface())
		}
	}
}

// TestEventTypesIdentiques : les valeurs du vocabulaire `EventType*` sont les memes des deux cotes.
func TestEventTypesIdentiques(t *testing.T) {
	paires := [][2]string{
		{filmhighlight.EventTypeKill, highlightevent.EventTypeKill},
		{filmhighlight.EventTypeDeath, highlightevent.EventTypeDeath},
		{filmhighlight.EventTypeMedal, highlightevent.EventTypeMedal},
		{filmhighlight.EventTypeMode, highlightevent.EventTypeMode},
	}
	for _, p := range paires {
		if p[0] != p[1] {
			t.Errorf("EventType divergent : film %q, LevelUp %q", p[0], p[1])
		}
	}
}

// TestHighlightEventsDepuisFilm_NilEtOrdre : `nil` reste `nil`, une tranche garde longueur et ordre.
func TestHighlightEventsDepuisFilm_NilEtOrdre(t *testing.T) {
	if highlightEventsDepuisFilm(nil) != nil {
		t.Fatal("nil converti en tranche non nil")
	}
	in := []filmhighlight.HighlightEvent{{TimeMS: 2}, {TimeMS: 1}}
	out := highlightEventsDepuisFilm(in)
	if len(out) != 2 || out[0].TimeMS != 2 || out[1].TimeMS != 1 {
		t.Fatalf("ordre ou longueur perdus : %+v", out)
	}
}

// remplirNonNul pose une valeur non nulle, distincte par indice, dans un champ de type simple.
func remplirNonNul(t *testing.T, f reflect.Value, i int) {
	t.Helper()
	switch f.Kind() {
	case reflect.String:
		f.SetString("valeur" + string(rune('A'+i)))
	case reflect.Bool:
		f.SetBool(true)
	case reflect.Int, reflect.Int64, reflect.Int32:
		f.SetInt(int64(100 + i))
	case reflect.Uint64, reflect.Uint32, reflect.Uint:
		f.SetUint(uint64(1000 + i))
	default:
		t.Fatalf("type de champ non couvert par le test : %s (etendre remplirNonNul)", f.Kind())
	}
}
