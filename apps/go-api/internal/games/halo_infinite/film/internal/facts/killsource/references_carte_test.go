package killsource

// references_carte_test.go — CHAQUE FILM DE REFERENCE SE DECODE SOUS SA CARTE (2026-09-27).
//
// Le banc `TestGoldenFilms` decodait sans carte depuis `f3a2f00eb` : trois des quatre films etaient
// lus aux largeurs de Cliffhanger, leur marche des morts se desynchronisait et le scan publiait a sa
// place (enquete ENQUETE_MARCHE_KILLSOURCE_2026-09-27). Les references portent donc leur carte
// (`reference.carte`), et tout decodage d une reference passe par [optionsDeReference].

import "testing"

// referenceDe rend la reference d un film, ou fait ECHOUER le test : un film qu on decode comme
// reference doit en etre une.
func referenceDe(t *testing.T, film string) reference {
	t.Helper()
	for _, r := range references {
		if r.film == film {
			return r
		}
	}
	t.Fatalf("%s n est pas un film de reference", film)
	return reference{}
}

// optionsDeReference : la configuration gelee SOUS LA CARTE du film de reference. Une reference sans
// carte fait ECHOUER le test — jamais un decodage aux largeurs par defaut.
func optionsDeReference(t *testing.T, film string) *Options {
	t.Helper()
	ref := referenceDe(t, film)
	if ref.carte == "" {
		t.Fatalf("%s : reference SANS CARTE — un film de reference ne se decode jamais aux largeurs "+
			"d une autre carte (renseigner `carte` dans `references`)", film)
	}
	carte := carteDuCatalogue(t, ref.carte)
	opts := DefaultOptions()
	opts.Carte = &carte
	return &opts
}

// TestReferencesPortentLeurCarte : chaque film de reference NOMME sa carte, et elle se resout au
// catalogue avec ses largeurs. Il tourne SANS fixture (CI comprise) : c est lui qui empeche le banc
// de redevenir, en silence, un decodage sans carte.
func TestReferencesPortentLeurCarte(t *testing.T) {
	for _, ref := range references {
		t.Run(ref.film, func(t *testing.T) {
			if opts := optionsDeReference(t, ref.film); !carteApplicable(opts.Carte) {
				t.Errorf("%s : la carte %q se resout sans largeurs d axe", ref.film, ref.carte)
			}
		})
	}
}
