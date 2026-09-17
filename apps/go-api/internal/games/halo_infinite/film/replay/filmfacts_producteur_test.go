package replay

// filmfacts_producteur_test.go — LES GARDES DU PRODUCTEUR DES FAITS, ET DE SON REJEU.
//
// Extrait de `filmfacts_fichier_test.go` le 2026-09-18 : celui-ci a franchi les 500 lignes en
// gagnant le garde-rail du trou que le gate S8 a trouve, et la table de `film_file_size_test.go`
// est DATEE ET FERMEE au 2026-09-16 — un fichier neuf ne s y inscrit pas, il se coupe.
//
// LA COUPE SUIT UNE FRONTIERE REELLE : dans `filmfacts_fichier_test.go`, le CODEC (point fixe,
// sections, pertes declarees, en-tete, fraicheur, saut d une section inconnue) ; ici, le
// PRODUCTEUR — ce que la cuisson pose dans un fichier de faits, ce qu elle laisse a l assemblage,
// et le fait qu un rejeu depuis ces faits rende le MEME document.
//
// AUCUN OCTET DE FILM dans aucun de ces tests.

import (
	"reflect"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// TestEnteteDesFaitsEgaleLaCouvertureDuDocument : L INVARIANT DU LOT 4.1.2, MESURE SANS FILM.
//
// L en-tete du fichier de faits porte [DecoderCoverage] VERBATIM, et l artefact publie le MEME
// type en `coverage.decoder`. L invariant est donc : L EN-TETE RELU == LE `coverage.decoder` DE
// L ARTEFACT PRODUIT, CHAMP POUR CHAMP.
//
// IL SE MESURE ICI SANS OUVRIR UN FILM, et c est ce qui le rend toujours actif : l assemblage est
// PUR (`BuildFromPositions`), et l identite du film est le SEUL parametre dont les deux cotes
// dependent. Les deux cas du bloc `registry` sont joues — present, et absent (film sans section
// d identification, ou l invariant V15 (15) exige « build vide, bloc present »).
func TestEnteteDesFaitsEgaleLaCouvertureDuDocument(t *testing.T) {
	entry := goldenEntryPourTest(t)
	// LES ENTREES VIENNENT DU FIXTURE VERSIONNE (aucun octet de film) : un document sans position
	// ne publie AUCUNE couverture, donc l invariant n aurait pas de cote gauche a comparer.
	g := loadGoldenInputs(t)
	for _, id := range []*profile.FilmIdentity{
		nil,
		{Build: "HI_1_13_0", FormatVersion: 27},
		{Build: "HI_1_13_0", FormatVersion: 27, RegistryFingerprint: 0x0123456789abcdef,
			RegistryBlocks: 9, RegistryNamedSlots: 31},
	} {
		opt := g.options()
		opt.MapQuant, opt.FilmIdentity = &entry, id
		doc := BuildFromPositions(goldenFilm, "halo_infinite", g.Positions, g.Fire, opt)
		if doc.Coverage == nil || doc.Coverage.Decoder == nil {
			t.Fatal("le document ne publie pas `coverage.decoder` : l invariant n a plus de " +
				"cote gauche")
		}
		blob, err := EncodeFilmFactsFile(&FilmFactsFile{
			Coverage: *couvertureDuDecodeur(id),
			Facts:    FilmFacts{Film: goldenFilm, MapModule: entry.Module, AxisW: entry.AxisWidths},
			Identity: identiteDeFaits(id),
		})
		if err != nil {
			t.Fatalf("encodage : %v", err)
		}
		e, err := DecodeFilmFactsEntete(blob)
		if err != nil {
			t.Fatalf("en-tete : %v", err)
		}
		if !reflect.DeepEqual(e.Coverage, *doc.Coverage.Decoder) {
			t.Errorf("en-tete des faits != coverage.decoder de l artefact :\n  faits    : %+v\n"+
				"  artefact : %+v", e.Coverage, *doc.Coverage.Decoder)
		}
	}
}

// TestBuildFromFactsEgaleLAssemblageDirect : LE DOCUMENT REJOUE DEPUIS UN FICHIER DE FAITS EST
// CELUI DE L ASSEMBLAGE DIRECT.
//
// C EST LA MOITIE DE S8 QUI SE MESURE SANS FILM. Le test S8 du lot 4.1.3 compare deux PASSES
// completes (film contre faits) sur le corpus ; celui-ci compare, sur le fixture versionne,
// l assemblage sur les entrees EN MEMOIRE a l assemblage sur les MEMES entrees passees par le
// fichier de faits. Un champ que le fichier perdrait se voit ici, sans decoder un film et sans
// attendre la « voie libre » de la machine.
//
// LES REPLIS DU BALAYAGE FONT PARTIE DE LA MESURE : le rapport persiste est cumule par
// `BuildFromFacts` AVANT l assemblage, et le cote gauche le pose directement sur son compteur.
// Les deux doivent publier le MEME `coverage.fallbacks`.
func TestBuildFromFactsEgaleLAssemblageDirect(t *testing.T) {
	entry := goldenEntryPourTest(t)
	g := loadGoldenInputs(t)
	id := &profile.FilmIdentity{Build: "HI_1_13_0", FormatVersion: 27}
	repliDuBalayage := []fallback.Declenchement{
		{Nom: fallback.NomPlafondGrenadeParDefaut, Declenchements: 2},
	}

	// A GAUCHE : l assemblage direct, avec le repli du balayage DEJA dans le compteur — c est
	// l etat ou `BuildFromFilmAvecFaits` laisse les options en sortant de son balayage.
	optDirect := g.options()
	optDirect.MapQuant, optDirect.FilmIdentity = &entry, id
	optDirect.Fallbacks = fallback.NouveauCompteur()
	optDirect.Fallbacks.Cumuler(repliDuBalayage)
	direct := BuildFromPositions(goldenFilm, "halo_infinite", g.Positions, g.Fire, optDirect)

	// A DROITE : le MEME etat, passe par le fichier de faits.
	blob, err := EncodeFilmFactsFile(&FilmFactsFile{
		Coverage:  *couvertureDuDecodeur(id),
		Facts:     *g,
		Identity:  identiteDeFaits(id),
		Fallbacks: repliDuBalayage,
	})
	if err != nil {
		t.Fatalf("encodage : %v", err)
	}
	f, err := DecodeFilmFactsFile(blob, entry)
	if err != nil {
		t.Fatalf("relecture : %v", err)
	}
	rejoue := BuildFromFacts(goldenFilm, "halo_infinite", f, Options{MapQuant: &entry})

	if renderAssembly(direct) != renderAssembly(rejoue) {
		t.Error("le document REJOUE DEPUIS LES FAITS differe de l assemblage direct : le fichier " +
			"de faits perd un champ que `BuildFromPositions` consomme, ou ne restitue pas " +
			"l identite / les replis du balayage. Comparer `renderAssembly` des deux cotes.")
	}
}

// sectionsLaisseesALAssemblage : les sections que [faitsDuBalayage] NE REMPLIT PAS, et pourquoi.
// 2026-09-18, lot 4.1.3 — table posee APRES que le gate S8 a trouve le trou.
var sectionsLaisseesALAssemblage = map[string]string{
	"Statborg": "les enregistrements d entite, les rafales de capture, le temoin de troncature " +
		"et l horloge des chunks du manifeste naissent dans `replaybuild.statborgDuFilm` : la " +
		"couche de PUBLICATION ne les voit jamais passer",
	"Kills": "le resultat du kill-feed nait dans `replaybuild.decodeKillSource` — meme raison",
}

// TestFaitsDuBalayageLaisseExactementDeuxSectionsALAssemblage : LA MOITIE `replay` DU GARDE-RAIL
// DU TROU QUE S8 A TROUVE.
//
// # LE DEFAUT, ET CE QU IL A COUTE
//
// `faitsDuBalayage` remplit ce que la couche de publication sait ; `Statborg` et `Kills`, elles,
// naissent dans l ASSEMBLAGE (`replaybuild`). Le 2026-09-18, PERSONNE ne les posait : le fichier
// ecrit portait deux sections vides, son en-tete restait FRAIS, et un rejeu depuis ces faits
// publiait un document sans courbe de score, sans actions d objectif, sans drapeau, sans
// couronne, sans crane et sans armement — SANS UNE LIGNE POUR LE DIRE. Le seul gate qui pouvait
// l attraper est le S8 (deux passes du meme commit, comparees a l octet).
//
// # CE QUE CE TEST FERME, ET POURQUOI IL EST ICI
//
// Il balaie PAR REFLEXION les champs de [FilmFactsFile] apres `faitsDuBalayage` et exige que les
// champs vides soient EXACTEMENT ceux de `sectionsLaisseesALAssemblage`. Une section NEUVE que
// personne ne remplirait apparait donc comme un troisieme champ vide, et le message dit quoi
// faire. Le pendant — « l assemblage pose bien ces deux-la » — est
// `replaybuild.TestCompleterLesFaitsPoseLesDEUXSectionsDeLAssemblage` : `film/internal/*` n est
// pas importable de la-bas (frontiere ADR 0034), donc chaque moitie vit ou elle est verifiable.
//
// AUCUN OCTET DE FILM : le contexte s ouvre sur un film NIL, et son decoupage vient du CATALOGUE.
func TestFaitsDuBalayageLaisseExactementDeuxSectionsALAssemblage(t *testing.T) {
	entry := goldenEntryPourTest(t)
	fc := grammar.NewFilmContextForMap(nil, &entry, nil)
	if fc.ImposedLayout() == nil {
		t.Fatal("le catalogue n impose plus de decoupage pour la carte du film de reference : ce " +
			"test ne peut plus se passer de film, le deplacer ou lui en donner un")
	}
	opt := Options{MapQuant: &entry, Fallbacks: fallback.NouveauCompteur(),
		FilmIdentity: &profile.FilmIdentity{Build: "HI_1_13_0", FormatVersion: 27}}
	opt.Fallbacks.Declenche(fallback.NomPlafondGrenadeParDefaut)
	g := loadGoldenInputs(t)

	f := faitsDuBalayage(goldenFilm, fc, opt, g.FilmInputs)
	if f == nil {
		t.Fatal("faitsDuBalayage rend nil sur un decoupage LU du catalogue : elle ne devrait " +
			"rendre nil que sur un decoupage illisible")
	}
	champs := reflect.VisibleFields(reflect.TypeOf(FilmFactsFile{}))
	if len(champs) < 6 {
		t.Fatalf("%d champ(s) lus sur FilmFactsFile : la reflexion ne mesure plus rien", len(champs))
	}
	for _, c := range champs {
		if !c.IsExported() || strings.Contains(c.Name, ".") {
			continue
		}
		vide := reflect.ValueOf(*f).FieldByName(c.Name).IsZero()
		raison, laissee := sectionsLaisseesALAssemblage[c.Name]
		switch {
		case vide && !laissee:
			t.Errorf("FilmFactsFile.%s sort VIDE de `faitsDuBalayage` et n est PAS declaree "+
				"laissee a l assemblage.\nUn fichier de faits avec cette section vide se relit "+
				"avec un en-tete FRAIS et publie un document appauvri, sans une ligne pour le "+
				"dire — c est le defaut que le gate S8 a trouve le 2026-09-18.\nDEUX REPONSES, "+
				"une seule est bonne : la remplir ICI si la publication la connait, ou "+
				"l inscrire dans `sectionsLaisseesALAssemblage` AVEC sa raison ET la poser dans "+
				"`replaybuild.completerLesFaits`.", c.Name)
		case !vide && laissee:
			t.Errorf("FilmFactsFile.%s est declaree laissee a l assemblage (%q) mais "+
				"`faitsDuBalayage` la remplit : entree perimee, la retirer — et verifier que "+
				"`replaybuild.completerLesFaits` ne l ecrase pas.", c.Name, raison)
		}
	}
}
