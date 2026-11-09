package i18n

import (
	"context"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		want Locale
	}{
		{"fr", "fr"},
		{"en", "en"},
		{"EN", "en"},
		{"  en  ", "en"},
		{"en-US", "en"},
		{"en_us", "en"},
		{"en-GB,fr;q=0.8", "en"},
		{"fr-FR", "fr"},
		{"", Default},
		{"   ", Default},
		{"zz", Default},
		{"de-DE", Default},
		{"-", Default},
	}
	for _, c := range cases {
		if got := Parse(c.in); got != c.want {
			t.Errorf("Parse(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLookup(t *testing.T) {
	if l, ok := Lookup(" EN-gb "); !ok || l != "en" {
		t.Errorf("Lookup(EN-gb) = %q, %v", l, ok)
	}
	for _, in := range []string{"", "zz", "de-DE"} {
		if l, ok := Lookup(in); ok || l != "" {
			t.Errorf("Lookup(%q) = %q, %v, want (\"\", false)", in, l, ok)
		}
	}
}

func TestLocalesOrderAndCopy(t *testing.T) {
	got := Locales()
	if len(got) < 2 || !reflect.DeepEqual(got[:2], []Locale{"fr", "en"}) {
		t.Fatalf("Locales() = %v", got)
	}
	got[0] = "zz"
	if Locales()[0] != "fr" {
		t.Error("Locales() doit retourner une copie")
	}
}

func TestDefaultAndFallbackAreRequired(t *testing.T) {
	if !Required(Default) || !Required(Fallback) {
		t.Errorf("Default %q et Fallback %q doivent être required", Default, Fallback)
	}
	if Required("zz") || Known("zz") {
		t.Error("une langue hors liste n'est ni connue ni required")
	}
}

func TestIntlTagAndHaloTags(t *testing.T) {
	if got := IntlTag("en"); got != "en-GB" {
		t.Errorf("IntlTag(en) = %q", got)
	}
	if got := IntlTag("fr"); got != "fr-FR" {
		t.Errorf("IntlTag(fr) = %q", got)
	}
	if got := IntlTag("zz"); got != IntlTag(Default) {
		t.Errorf("IntlTag(zz) = %q, want celle de Default", got)
	}
	tags := HaloTags("en")
	if !reflect.DeepEqual(tags, []string{"en-US"}) {
		t.Errorf("HaloTags(en) = %v", tags)
	}
	tags[0] = "zz"
	if HaloTags("en")[0] != "en-US" {
		t.Error("HaloTags doit retourner une copie")
	}
	if !reflect.DeepEqual(HaloTags("zz"), HaloTags(Default)) {
		t.Error("HaloTags(zz) doit retomber sur Default")
	}
}

func TestContext(t *testing.T) {
	if got := FromContext(context.Background()); got != Default {
		t.Errorf("contexte vide : %q, want %q", got, Default)
	}
	ctx := WithContext(context.Background(), "en")
	if got := FromContext(ctx); got != "en" {
		t.Errorf("FromContext = %q, want en", got)
	}
	if got := FromContext(WithContext(context.Background(), "zz")); got != Default {
		t.Errorf("langue hors liste : %q, want %q", got, Default)
	}
}

func TestFromHaloTag(t *testing.T) {
	for tag, want := range map[string]Locale{"fr-FR": "fr", "FR-fr": "fr", "en-US": "en", "fr": "fr", "en-GB": "en", " en ": "en"} {
		if got, ok := FromHaloTag(tag); !ok || got != want {
			t.Errorf("FromHaloTag(%q) = (%q, %v), want (%q, true)", tag, got, ok, want)
		}
	}
	for _, tag := range []string{"", "zz-YY", "zz"} {
		if got, ok := FromHaloTag(tag); ok {
			t.Errorf("FromHaloTag(%q) = (%q, true), want false", tag, got)
		}
	}
}

func TestChain(t *testing.T) {
	for in, want := range map[Locale][]Locale{"fr": {"fr", "en"}, "en": {"en", "fr"}, "zz": {"fr", "en"}, "": {"fr", "en"}} {
		got := Chain(in)
		if len(got) != len(want) {
			t.Fatalf("Chain(%q) = %v, want %v", in, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("Chain(%q) = %v, want %v", in, got, want)
			}
		}
	}
}
