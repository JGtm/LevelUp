package main

import "testing"

func texteFrEn(fr, en string) map[string]string { return map[string]string{"fr": fr, "en": en} }

func TestChoisirUneVarianteParSonAvecSaVoix(t *testing.T) {
	langues := languesDeTest(t)
	joues := SonsJoues{Sons: map[string]int{"0000000a": 5, "0000000b": 2, "0000000c": 1, "0000000d": 9}}
	textes := []SonTextes{
		// 0a : trois variantes, la première sans voix française : la deuxième est retenue
		{Son: "0000000a", Banque: "b", Variantes: []VarianteTexte{
			{Evenement: "e1", Cle: "k1", Textes: texteFrEn("un", "one")},
			{Evenement: "e2", Cle: "k2", Textes: texteFrEn("deux", "two")},
			{Evenement: "e3", Cle: "k3", Textes: texteFrEn("trois", "three")},
		}},
		// 0b : un cri, une variante sans texte
		{Son: "0000000b", Variantes: []VarianteTexte{{Evenement: "e4", Cle: "k4", Textes: map[string]string{}}}},
		// 0c : pas de variante du tout
		{Son: "0000000c"},
		// 0d : texte français seulement : une langue requise manque, le son n'est pas parlé
		{Son: "0000000d", Variantes: []VarianteTexte{{Evenement: "e5", Cle: "k5", Textes: map[string]string{"fr": "seulement"}}}},
		// 0e : jamais joué
		{Son: "0000000e", Variantes: []VarianteTexte{{Evenement: "e6", Cle: "k6", Textes: texteFrEn("x", "y")}}},
	}
	banques := []SonBanque{{Son: "0000000a", Evenements: []EvenementBanque{
		{Evenement: "e1", Langues: map[string][]MediaVoix{"en": {{ID: "m1"}}}},
		{Evenement: "e2", Langues: map[string][]MediaVoix{"fr": {{ID: "m2", Paquet: "p"}, {ID: "m2bis"}}, "en": {{ID: "m2en"}}}},
	}}}
	got, bilan := choisir(joues, textes, banques, langues)
	if len(got) != 1 || got[0].Son != "0000000a" {
		t.Fatalf("choix = %+v : seul le son parlé et joué reste", got)
	}
	c := got[0]
	if c.Evenement != "e2" || c.Textes["fr"] != "deux" || c.Voix["fr"].ID != "m2" || c.Voix["en"].ID != "m2en" {
		t.Errorf("choix = %+v : la variante retenue porte son texte ET sa voix, et le premier média de l'évènement", c)
	}
	if bilan.Joues != 4 || bilan.Parles != 1 || bilan.SansTexte != 2 || bilan.SansVariant != 1 || bilan.AvecVoix["fr"] != 1 {
		t.Errorf("bilan = %+v", bilan)
	}
}

func TestChoisirGardeLaPremiereVarianteSiAucuneNAUneVoix(t *testing.T) {
	joues := SonsJoues{Sons: map[string]int{"0000000a": 1}}
	textes := []SonTextes{{Son: "0000000a", Variantes: []VarianteTexte{
		{Evenement: "e1", Cle: "k1", Textes: texteFrEn("un", "one")},
		{Evenement: "e2", Cle: "k2", Textes: texteFrEn("deux", "two")},
	}}}
	got, _ := choisir(joues, textes, []SonBanque{{Son: "0000000a", Absente: true}}, languesDeTest(t))
	if len(got) != 1 || got[0].Evenement != "e1" || len(got[0].Voix) != 0 {
		t.Errorf("choix = %+v : sans voix nulle part, la réplique reste affichable, avec sa première variante", got)
	}
}
