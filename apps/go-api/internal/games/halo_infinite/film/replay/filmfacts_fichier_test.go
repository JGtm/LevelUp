package replay

// filmfacts_fichier_test.go — LES GARDES DU FICHIER DE FAITS (lot 4.1.1-b, 2026-09-17).
//
// AUCUN OCTET DE FILM : tous les temoins sont fabriques a la main ou relus du fixture d entrees
// versionne. Ces tests tournent partout, CI comprise.

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/killsource"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// fichierTemoin fabrique un fichier de faits dont LES CINQ SECTIONS sont non vides.
func fichierTemoin(t *testing.T) *FilmFactsFile {
	t.Helper()
	facts := loadGoldenInputs(t)
	facts.FlagMarks = grammar.CarrierMarkScan{
		Marks:      []grammar.CarrierMark{{TimestampUS: 1_000, Slot: 11}, {TimestampUS: 4_500, Slot: 12}},
		KeyframeUS: []uint64{1_000, 2_000, 4_500},
		Records:    42, BipedRecords: 17,
	}
	facts.ZoneReads = []grammar.ManagedPropertyRead{
		{Slot: 300, TimestampUS: 2_500, Field: grammar.ManagedPropertyScalar, FilmIndex: -1,
			Tag: 3, Value: 7, HasValue: true, Chained: true},
	}
	facts.ZoneScanned = true
	facts.BombReads = []types.NavpointRadialRead{{Slot: 12, TMS: -40, Q: 200, Chained: true}}
	return &FilmFactsFile{
		Coverage: DecoderCoverage{
			SourceRev: "source-x", ProfileRev: "profile-x", GrammarRev: "grammar-x",
			FactsRev: "facts-x", Build: "HI_1_13_0",
			Registry: &RegistryCoverage{Fingerprint: "0x0123456789abcdef",
				Status: RegistryStatutConnue, Blocks: 9, NamedSlots: 31},
		},
		Facts: *facts,
		Identity: &profile.FilmIdentity{Version: "v", Build: "HI_1_13_0", Flavor: "f",
			BuildID: 7, Changelist: 9, FormatVersion: 27},
		Fallbacks: []fallback.Declenchement{
			{Nom: fallback.NomLargeursAxeParDefautConservees, Declenchements: 1},
			{Nom: fallback.NomPlafondGrenadeParDefaut, Declenchements: 3},
		},
		Statborg: FilmStatborg{
			Records: []types.StatRecord{
				{TimeMS: 1_000, Slot: 10, Round: 0, Comps: map[int]types.StatValue{
					0: {A: 3, B: 4}, 22: {A: 1, B: 0, C: 2, HasC: true}, 7: {D: 9, HasD: true},
				}},
				{TimeMS: 2_000, Slot: 6, Round: 1, Comps: map[int]types.StatValue{0: {A: 50}}},
			},
			BurstMS:      []int{1_200, 8_900},
			Truncated:    true,
			ChunkStartMS: map[int]int{0: 0, 1: 30_000, 2: 60_000},
		},
		Kills: &killsource.Result{
			Kills:               []killsource.Kill{{TimeMS: 5_000, Victim: "Temoin", Diverges: true}},
			Calibration:         "temoin",
			BijectionMargin:     4,
			BijectionDetermined: true,
		},
	}
}

// TestFilmFactsFichierEstUnPointFixe : le codec du fichier ne perd rien et ne reordonne rien.
//
// DEUX MESURES EN UNE. Le point fixe OCTET POUR OCTET attrape la perte (un champ qui ne revient
// pas) ET le non-determinisme : `StatRecord.Comps` et `ChunkStartMS` sont des MAPS, et une map Go
// ne s itere pas deux fois pareil. Un encodage qui suivrait l ordre d iteration rendrait deux
// blobs differents pour les memes faits — et les faits cesseraient d etre comparables d une
// cuisson a l autre.
func TestFilmFactsFichierEstUnPointFixe(t *testing.T) {
	f := fichierTemoin(t)
	entry := goldenEntryPourTest(t)
	premier, err := EncodeFilmFactsFile(f)
	if err != nil {
		t.Fatalf("premier encodage : %v", err)
	}
	relu, err := DecodeFilmFactsFile(premier, entry)
	if err != nil {
		t.Fatalf("relecture : %v", err)
	}
	second, err := EncodeFilmFactsFile(relu)
	if err != nil {
		t.Fatalf("second encodage : %v", err)
	}
	if !bytes.Equal(premier, second) {
		t.Fatalf("le codec du fichier n est pas un point fixe : %d octets contre %d — un champ se "+
			"perd, ou une map s ecrit dans l ordre de son iteration", len(second), len(premier))
	}
	// L ordre d ecriture des maps doit tenir SUR PLUSIEURS ENCODAGES DU MEME OBJET, et pas
	// seulement sur l aller-retour : c est la mesure du determinisme, la precedente celle de la
	// fidelite.
	for i := 0; i < 8; i++ {
		encore, err := EncodeFilmFactsFile(f)
		if err != nil {
			t.Fatalf("encodage %d : %v", i, err)
		}
		if !bytes.Equal(premier, encore) {
			t.Fatalf("encodage %d different du premier : l ordre d ecriture d une map suit son "+
				"iteration", i)
		}
	}
}

// TestFilmFactsFichierPorteLesCinqSections : aucune section ne revient vide.
//
// PAR REFLEXION SUR LE TYPE, jamais sur une liste ecrite a la main : une section ajoutee a
// [FilmFactsFile] sans codec fait rougir ce test tout seul.
func TestFilmFactsFichierPorteLesCinqSections(t *testing.T) {
	f := fichierTemoin(t)
	entry := goldenEntryPourTest(t)
	blob, err := EncodeFilmFactsFile(f)
	if err != nil {
		t.Fatalf("encodage : %v", err)
	}
	relu, err := DecodeFilmFactsFile(blob, entry)
	if err != nil {
		t.Fatalf("relecture : %v", err)
	}
	champs := reflect.VisibleFields(reflect.TypeOf(FilmFactsFile{}))
	if len(champs) < 6 {
		t.Fatalf("%d champ(s) lus sur FilmFactsFile : la reflexion ne mesure plus rien", len(champs))
	}
	for _, c := range champs {
		if !c.IsExported() || strings.Contains(c.Name, ".") {
			continue
		}
		avant := reflect.ValueOf(*f).FieldByName(c.Name)
		if avant.IsZero() {
			t.Fatalf("le temoin de FilmFactsFile.%s est VIDE : le test ne mesure rien sur cette "+
				"section — remplir `fichierTemoin`.", c.Name)
		}
		if reflect.ValueOf(*relu).FieldByName(c.Name).IsZero() {
			t.Errorf("FilmFactsFile.%s revient VIDE : la section n est pas transportee. Une "+
				"section neuve s ecrit dans `EncodeFilmFactsFile` ET se relit dans `lireSection`, "+
				"avec un identifiant NEUF (jamais reutilise) et une montee de `SchemaDesFaits`.",
				c.Name)
		}
	}
}

// TestFilmFactsFichierNePerdQueLePaquetDeKillsource : LE RATCHET DE LA PERTE DECLAREE.
//
// # CE QU IL GARDE
//
// Les sections 2 a 5 passent par `encoding/json`, qui ignore les champs NON EXPORTES. Un seul est
// connu et ASSUME : `killsource.Kill.paquet` — une coordonnee interne au decodeur (ou l octet a
// ete lu), non exportee deliberement (`killsource/kill.go:69-72`), qu aucun consommateur hors de
// `killsource` ne lit. Sa perte est SANS EFFET sur le document et AVEC effet sur `digest.Of`, qui
// hache les champs exportes OU NON : c est ce qui interdit a l etape `killsource` du TSV
// d equivalence de servir d oracle a un rejeu depuis les faits.
//
// TOUT AUTRE champ non exporte apparaissant dans le graphe des cinq sections serait une perte
// SILENCIEUSE et NON DECLAREE : ce test le nomme. L allowlist est datee et se justifie en une
// ligne, comme toute tolerance du depot.
func TestFilmFactsFichierNePerdQueLePaquetDeKillsource(t *testing.T) {
	// Les champs non exportes TOLERES, `Type.champ`. 2026-09-17, lot 4.1.1-b.
	tolere := map[string]bool{
		"killsource.Kill.paquet": true, // coordonnee interne du decodeur (kill.go:69-72)
	}
	var trouves []string
	vus := map[reflect.Type]bool{}
	var marcher func(t reflect.Type)
	marcher = func(rt reflect.Type) {
		for rt.Kind() == reflect.Pointer || rt.Kind() == reflect.Slice ||
			rt.Kind() == reflect.Array || rt.Kind() == reflect.Map {
			if rt.Kind() == reflect.Map {
				marcher(rt.Key())
			}
			rt = rt.Elem()
		}
		if rt.Kind() != reflect.Struct || vus[rt] {
			return
		}
		vus[rt] = true
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if !f.IsExported() {
				cle := rt.PkgPath()
				cle = cle[strings.LastIndex(cle, "/")+1:] + "." + rt.Name() + "." + f.Name
				if !tolere[cle] {
					trouves = append(trouves, cle)
				}
				continue
			}
			marcher(f.Type)
		}
	}
	// La section 1 passe par le codec MAISON (elle ne perd rien par ce mecanisme) : les sections
	// mesurees ici sont les quatre qui passent par JSON.
	for _, rt := range []reflect.Type{
		reflect.TypeOf(&profile.FilmIdentity{}),
		reflect.TypeOf([]fallback.Declenchement{}),
		reflect.TypeOf(FilmStatborg{}),
		reflect.TypeOf(killsource.Result{}),
	} {
		marcher(rt)
	}
	if len(trouves) > 0 {
		t.Errorf("champ(s) non exporte(s) NON DECLARE(S) dans le graphe des sections JSON du "+
			"fichier de faits : %s\n"+
			"`encoding/json` les perd SANS RIEN DIRE. Soit l exporter, soit l inscrire dans la "+
			"table `tolere` de ce test AVEC sa date et la raison pour laquelle sa perte est sans "+
			"effet sur le document publie.", strings.Join(trouves, ", "))
	}
}

// TestFilmFactsEnteteSeLitSurLesPremiersOctets : la decision « decoder ou relire » ne lit pas le
// mega-octet de positions.
//
// LA MESURE EST LA GARDE : on TRONQUE le fichier juste apres son en-tete et on exige que
// [DecodeFilmFactsEntete] rende encore l en-tete complet. Un lecteur qui aurait besoin des
// sections echouerait ici.
func TestFilmFactsEnteteSeLitSurLesPremiersOctets(t *testing.T) {
	f := fichierTemoin(t)
	blob, err := EncodeFilmFactsFile(f)
	if err != nil {
		t.Fatalf("encodage : %v", err)
	}
	complet, err := DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Fatalf("en-tete du fichier complet : %v", err)
	}
	tronque, err := DecodeFilmFactsEntete(blob[:complet.corps])
	if err != nil {
		t.Fatalf("en-tete du fichier TRONQUE aux seules premieres %d octets : %v — la decision "+
			"« decoder ou relire » ne doit pas dependre des sections", complet.corps, err)
	}
	if !reflect.DeepEqual(tronque, complet) {
		t.Fatalf("l en-tete lu sur le fichier tronque differe de celui du fichier complet")
	}
	if complet.corps > 1<<12 {
		t.Errorf("l en-tete pese %d octets : la decision de fraicheur devait coster quelques "+
			"centaines d octets, pas une page", complet.corps)
	}
}

// TestFilmFactsEnteteEstLeMemeTypeQueCoverageDecoder : L INVARIANT GRATUIT.
//
// L en-tete porte [DecoderCoverage] VERBATIM — le MEME type que `coverage.decoder` de l artefact.
// Ce test epingle l aller-retour champ pour champ ; l invariant de PRODUCTION (en-tete relu ==
// `coverage.decoder` de l artefact produit) est verifie cote `replaybuild` au lot 4.1.2.
func TestFilmFactsEnteteEstLeMemeTypeQueCoverageDecoder(t *testing.T) {
	for _, cov := range []DecoderCoverage{
		{SourceRev: "s", ProfileRev: "p", GrammarRev: "g", FactsRev: "f", Build: "HI_1_13_0",
			Registry: &RegistryCoverage{Fingerprint: "0xdead", Status: RegistryStatutConnue,
				Blocks: 3, NamedSlots: 4}},
		// BUILD VIDE ET BLOC PRESENT sur un build inconnu (V15 (15)) : le cas qui doit survivre
		// a l aller-retour, sans quoi l ambiguite que D-7 interdit revient par le fichier.
		{SourceRev: "s", ProfileRev: "p", GrammarRev: "g", FactsRev: "f"},
	} {
		f := fichierTemoin(t)
		f.Coverage = cov
		blob, err := EncodeFilmFactsFile(f)
		if err != nil {
			t.Fatalf("encodage : %v", err)
		}
		e, err := DecodeFilmFactsEntete(blob)
		if err != nil {
			t.Fatalf("en-tete : %v", err)
		}
		if !reflect.DeepEqual(e.Coverage, cov) {
			t.Errorf("l en-tete relu differe : %+v contre %+v", e.Coverage, cov)
		}
	}
}

// TestFilmFactsUtilisableRefuseLesQuatreCauses : la regle de fraicheur TOUT OU RIEN, par cause —
// et ce qui NE doit PAS peser (le build, le registre : des faits du film, pas du binaire).
func TestFilmFactsUtilisableRefuseLesQuatreCauses(t *testing.T) {
	entry := goldenEntryPourTest(t)
	f := fichierTemoin(t)
	// LES REVISIONS DU TEMOIN DOIVENT ETRE CELLES DU BINAIRE : la porte de fraicheur les compare
	// aux constantes de compilation, et un temoin a revisions inventees ne mesurerait que le refus.
	f.Coverage.SourceRev = source.Rev
	f.Coverage.ProfileRev = profile.Rev
	f.Coverage.GrammarRev = grammar.Rev
	f.Coverage.FactsRev = facts.Rev
	blob, err := EncodeFilmFactsFile(f)
	if err != nil {
		t.Fatalf("encodage : %v", err)
	}
	e, err := DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Fatalf("en-tete : %v", err)
	}
	if err := e.Utilisable(entry); err != nil {
		t.Fatalf("des faits frais sont refuses : %v", err)
	}
	autre := e
	autre.Coverage.GrammarRev += "-bis"
	if err := autre.Utilisable(entry); !errorsEstRevisions(err) {
		t.Errorf("une revision differente doit rendre ErrFilmFactsRevisions, obtenu : %v", err)
	}
	// LE BUILD ET LE REGISTRE NE SONT PAS COMPARES, et c est une propriete ecrite : ce sont des
	// faits DU FILM, que la porte ne peut pas recalculer sans ouvrir le film (cf. `Utilisable`).
	autreBuild := e
	autreBuild.Coverage.Build = "HI_9_9_9"
	autreBuild.Coverage.Registry = nil
	if err := autreBuild.Utilisable(entry); err != nil {
		t.Errorf("le build et le registre ne doivent PAS peser sur la fraicheur : %v", err)
	}
	perime := e
	perime.Schema = SchemaDesFaits + 1
	if err := perime.Utilisable(entry); !errorsEstVersion(err) {
		t.Errorf("un schema inconnu doit rendre ErrFilmFactsVersion, obtenu : %v", err)
	}
	autreCarte := entry
	autreCarte.Module = "une_autre_carte"
	if err := e.Utilisable(autreCarte); !errorsEstCarte(err) {
		t.Errorf("une autre carte doit rendre ErrFilmFactsCarte, obtenu : %v", err)
	}
	decale := e
	decale.AxisW[0]++
	if err := decale.Utilisable(entry); !errorsEstDecoupage(err) {
		t.Errorf("un decoupage contredit doit rendre ErrFilmFactsDecoupage, obtenu : %v", err)
	}
	detecte := e
	detecte.LayoutDetected = true
	if err := detecte.Utilisable(entry); !errorsEstDecoupage(err) {
		t.Errorf("l AUTRE sens (faits « auto-detectes » sur une carte que le catalogue impose) "+
			"doit rendre ErrFilmFactsDecoupage, obtenu : %v", err)
	}
}

// TestFilmFactsFichierSauteUneSectionInconnue : le cadre a longueur prefixee sert a quelque chose.
func TestFilmFactsFichierSauteUneSectionInconnue(t *testing.T) {
	f := fichierTemoin(t)
	entry := goldenEntryPourTest(t)
	blob, err := EncodeFilmFactsFile(f)
	if err != nil {
		t.Fatalf("encodage : %v", err)
	}
	w := &gwriter{b: append([]byte{}, blob...)}
	ecrireSection(w, 99, []byte("une section d un schema futur"))
	relu, err := DecodeFilmFactsFile(w.b, entry)
	if err != nil {
		t.Fatalf("une section inconnue doit etre SAUTEE, pas fatale : %v", err)
	}
	if relu.Statborg.Truncated != f.Statborg.Truncated || len(relu.Statborg.Records) != len(f.Statborg.Records) {
		t.Error("les sections connues n ont pas survecu au saut de la section inconnue")
	}
}

// Les quatre causes de refus se reconnaissent PAR LEUR SENTINELLE, jamais par leur texte : un
// message se reformule, une sentinelle est un contrat.
func errorsEstVersion(err error) bool   { return errors.Is(err, ErrFilmFactsVersion) }
func errorsEstRevisions(err error) bool { return errors.Is(err, ErrFilmFactsRevisions) }
func errorsEstCarte(err error) bool     { return errors.Is(err, ErrFilmFactsCarte) }
func errorsEstDecoupage(err error) bool { return errors.Is(err, ErrFilmFactsDecoupage) }

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
