package filmcorpus

// document_shape_jumeaux_test.go — LA FORME STOCKEE ET LA FORME SERVIE DU DOCUMENT DE REJEU, ET LA
// CHRONIQUE DE SES VERSIONS (plan D2, item 2.11, regle R1).
//
// Deux gardes deplacees du paquet `replay` de la brique, sous leur nom : elles confrontent la
// brique (`replay.ReplayDocument`, `replay.SchemaVersion`) a des morceaux de LevelUp — le document
// servi `domain/replaydoc`, l'extracteur de chronique de `testutil` — et n'utilisent que la
// surface exportee de la brique. Le reste de l'empreinte de forme (golden, regeneration, calques
// resolus a la requete) reste dans la brique : `replay/document_shape_test.go`.
//
// LE RENDU EST UNE COPIE. `documentShapeRender` et ses auxiliaires sont ceux de la brique (deux
// copies, regle 6 de CLAUDE.md) ; la derive est refusee par `TestRenduDesFormesJumeauxSuitLeGolden`,
// qui exige que CETTE copie rende, pour le document de la brique, exactement la forme en clair que
// le golden de la brique a figee.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/testutil"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// cheminGoldenDeForme : le golden de forme de la brique, relatif a la brique du decodeur (resolveur des
// briques).
var cheminGoldenDeForme = filepath.Join("replay", "testdata", "document_shape.golden")

// TestDocumentShapeTwinsAgree : la forme STOCKEE et la forme SERVIE coincident-elles, hors les champs et
// les types SERVIS SEULS de la liste datee ?
//
// Elles le doivent : `domain/replaydoc` est le jumeau de fil de `replay.ReplayDocument`, meme
// noms de types, memes tags JSON, memes `omitempty` (cf. son `doc.go`). Une divergence signifie
// qu'un calque cuit n'atteint plus le client, ou qu'un champ servi n'a plus de source.
//
// UNE SEULE EXCEPTION : les champs et les types de `testutil.ReplayChampsServisSeuls` et
// `testutil.ReplayTypesServisSeuls` (calques resolus a la requete, sans source stockee), la liste datee que
// lit aussi `service/replayview/parity_test.go`. Elle est retiree de la forme SERVIE seule : un champ servi
// hors liste, sans jumeau stocke, fait toujours echouer le test, et une entree qui ne designe plus rien dans
// le document servi aussi.
//
// ELLES SE COMPARENT SUR CE QUI SE SERIALISE. Un champ non exporte n'atteint jamais le client : le type
// qu'il est seul a porter (le dead-state qu'une vie de vehicule garde pour batir sa destruction) n'a pas
// de jumeau servi et n'en doit pas avoir. Le golden de la brique, lui, le range dans sa forme en clair
// (cf. `TestRenduDesFormesJumeauxSuitLeGolden`) : la comparaison des jumeaux suit donc les seuls champs
// exportes, et le golden la forme complete.
func TestDocumentShapeTwinsAgree(t *testing.T) {
	stocke := reflect.TypeFor[replay.ReplayDocument]()
	servi := reflect.TypeFor[replaydoc.ReplayDocument]()
	for _, pb := range testutil.ReplayServisSeulsIncoherents(inventaireServi(servi)) {
		t.Error(pb)
	}
	if msg := ecartDesFormesJumelles(stocke, servi, exceptionsDeForme{
		servisSeulsChamps: testutil.ReplayChampsServisSeuls, servisSeulsTypes: testutil.ReplayTypesServisSeuls,
		nonServisChamps: testutil.ReplayChampsNonServis, nonServisTypes: testutil.ReplayTypesNonServis,
		balisesServies: testutil.ReplayBalisesServiesDifferentes,
	}); msg != "" {
		t.Error(msg)
	}
}

// exceptionsDeForme porte les decisions ecrites qui separent les deux formes : les champs et types SERVIS SEULS
// (retires de la forme servie) et les champs et types NON SERVIS (retires de la forme stockee), par `Type.Champ`
// et par nom de type.
type exceptionsDeForme struct {
	servisSeulsChamps, servisSeulsTypes map[string]string
	nonServisChamps, nonServisTypes     map[string]string
	// balisesServies : les differences de balise JSON declarees (`Type.Champ`), rapportees a la balise stockee avant
	// comparaison.
	balisesServies map[string]testutil.BaliseServieDifferente
}

// ecartDesFormesJumelles rend le message d'ecart entre la forme serialisee de `stocke` et celle de `servi`, chacune
// privee de ses exceptions ecrites, ou "" quand elles coincident. Une exception ne retire rien de l'autre forme.
func ecartDesFormesJumelles(stocke, servi reflect.Type, ex exceptionsDeForme) string {
	a := documentShapeRenderSerialise(stocke, ex.nonServisChamps, ex.nonServisTypes)
	b := documentShapeRenderSerialise(servi, ex.servisSeulsChamps, ex.servisSeulsTypes)
	b, perimees := ramenerLesBalisesDeclarees(b, ex.balisesServies)
	if len(perimees) > 0 {
		return fmt.Sprintf("testutil.ReplayBalisesServiesDifferentes cite des balises que la forme servie n'ecrit plus "+
			"telles que declarees : %s", strings.Join(perimees, ", "))
	}
	if a == b {
		return ""
	}
	return fmt.Sprintf("la forme STOCKEE et la forme SERVIE divergent (empreintes %s et %s).\n%s",
		empreinteDe(a), empreinteDe(b), premierEcartTexte(a, b))
}

// ramenerLesBalisesDeclarees remplace, dans la forme servie `render`, la balise servie de chaque champ de `declarees`
// (`Type.Champ`) par sa balise stockee, pour que la comparaison ne voie que les differences NON declarees. Une
// declaration dont le champ n'a plus exactement la balise servie declaree est rendue dans `perimees` : une exception
// qui survit a sa raison finit par en couvrir une autre.
func ramenerLesBalisesDeclarees(render string, declarees map[string]testutil.BaliseServieDifferente) (string, []string) {
	vues := map[string]bool{}
	lignes := strings.Split(render, "\n")
	nomDuType := ""
	for i, l := range lignes {
		if !strings.HasPrefix(l, "  ") {
			nomDuType = l
			continue
		}
		champ, reste, _ := strings.Cut(strings.TrimPrefix(l, "  "), " ")
		cle := nomDuType + "." + champ
		d, ok := declarees[cle]
		if !ok {
			continue
		}
		servie := fmt.Sprintf("json:%q ", d.Servi)
		if strings.HasPrefix(reste, servie) {
			vues[cle] = true
			lignes[i] = "  " + champ + " " + fmt.Sprintf("json:%q ", d.Stocke) + strings.TrimPrefix(reste, servie)
		}
	}
	var perimees []string
	for cle := range declarees {
		if !vues[cle] {
			perimees = append(perimees, cle)
		}
	}
	sort.Strings(perimees)
	return strings.Join(lignes, "\n"), perimees
}

// inventaireServi recense, pour la forme SERVIE entiere (listes non retirees), les champs exportes par
// `Type.Champ` et les types struct nommes : ce que `testutil.ReplayServisSeulsIncoherents` confronte aux
// listes.
func inventaireServi(servi reflect.Type) (champs, types map[string]bool) {
	atteints := map[string]reflect.Type{}
	collecterTypes(servi, atteints, map[reflect.Type]bool{}, true, nil, nil)
	champs, types = map[string]bool{}, map[string]bool{}
	for nom, t := range atteints {
		types[nom] = true
		for f := range t.Fields() {
			if f.PkgPath == "" {
				champs[nom+"."+f.Name] = true
			}
		}
	}
	return champs, types
}

// TestDocumentShapeSchemaHasChronicleEntry : la version courante a-t-elle son entree de
// chronique ? Une montee sans chronique est une montee dont personne ne saura dire ce qu'elle a
// change — et la reprise du backfill se fait par ce numero.
func TestDocumentShapeSchemaHasChronicleEntry(t *testing.T) {
	versions, err := testutil.ReplayChronicleVersions()
	if err != nil {
		t.Fatalf("chronique illisible : %v", err)
	}
	if slices.Contains(versions, replay.SchemaVersion) {
		return
	}
	t.Errorf("SchemaVersion = %d n'a AUCUNE entree dans document_chronicle.go (versions "+
		"declarees : %v) — une montee sans chronique ne dit pas ce qu'elle change", replay.SchemaVersion, versions)
}

// TestRenduDesFormesJumeauxSuitLeGolden : la copie du rendu de forme de ce fichier est celle de la
// brique. Elle rend, pour le document de la brique, la forme en clair du golden et son empreinte.
func TestRenduDesFormesJumeauxSuitLeGolden(t *testing.T) {
	chemin := testutil.CheminBriqueFilm(t, cheminGoldenDeForme)
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin fige dans le code
	if err != nil {
		t.Fatalf("golden de forme de la brique illisible (%s) : %v", chemin, err)
	}
	tete, corps, ok := strings.Cut(string(brut), "\n\n")
	if !ok {
		t.Fatalf("%s : pas de ligne vide entre l'en-tete et la forme en clair", chemin)
	}
	render := documentShapeRender(reflect.TypeFor[replay.ReplayDocument]())
	if corps != render {
		t.Errorf("la copie du rendu de forme ne rend plus la forme en clair du golden.\n%s",
			premierEcartTexte(corps, render))
	}
	if want := "empreinte-stockee " + empreinteDe(render); !strings.Contains(tete, want) {
		t.Errorf("l'en-tete du golden ne porte pas %q", want)
	}
}

// empreinteDe : le hachage court d'une forme.
func empreinteDe(render string) string {
	sum := sha256.Sum256([]byte(render))
	return hex.EncodeToString(sum[:8])
}

// documentShapeRender rend la forme d'un type racine : tous les types struct NOMMES qu'il
// atteint, tries par nom, chacun avec ses champs exportes tries par nom eux aussi. L'ordre de
// declaration est deliberement ignore (l'ordre des champs n'est pas le contrat d'un lecteur JSON,
// et c'est ce qui rend les deux jumeaux comparables).
func documentShapeRender(root reflect.Type) string {
	return documentShapeRenderChamps(root, false, nil, nil)
}

// documentShapeRenderSerialise est la forme des seuls types atteints par un champ EXPORTE : ce qui
// peut atteindre un lecteur JSON, sans les champs ni les types servis seuls nommes (nil : aucun retrait).
func documentShapeRenderSerialise(root reflect.Type, champsHors, typesHors map[string]string) string {
	return documentShapeRenderChamps(root, true, champsHors, typesHors)
}

func documentShapeRenderChamps(root reflect.Type, exportesSeuls bool, champsHors, typesHors map[string]string) string {
	types := map[string]reflect.Type{}
	collecterTypes(root, types, map[reflect.Type]bool{}, exportesSeuls, champsHors, typesHors)
	noms := make([]string, 0, len(types))
	for nom := range types {
		noms = append(noms, nom)
	}
	sort.Strings(noms)
	var b strings.Builder
	for _, nom := range noms {
		b.WriteString(nom + "\n")
		for _, ligne := range champsDe(types[nom], champsHors) {
			b.WriteString("  " + ligne + "\n")
		}
	}
	return b.String()
}

// champsDe : les champs exportes d'un type, rendus et tries par nom, hors ceux que `hors` nomme
// (`Type.Champ`).
func champsDe(t reflect.Type, hors map[string]string) []string {
	out := make([]string, 0, t.NumField())
	for f := range t.Fields() {
		if f.PkgPath != "" {
			continue // champ non exporte : jamais serialise
		}
		if _, retire := hors[t.Name()+"."+f.Name]; retire {
			continue
		}
		out = append(out, fmt.Sprintf("%s json:%q %s", f.Name, f.Tag.Get("json"), nomDeType(f.Type)))
	}
	sort.Strings(out)
	return out
}

// collecterTypes recense les types struct NOMMES atteignables. La carte des types deja vus
// borne la recursion : le document porte des structures qui se referencent. Un champ de `champsHors`
// (`Type.Champ`) n'est pas suivi ; un type de `typesHors` n'est ni recense ni descendu.
func collecterTypes(t reflect.Type, out map[string]reflect.Type, vus map[reflect.Type]bool, exportesSeuls bool,
	champsHors, typesHors map[string]string) {
	if t == nil || vus[t] {
		return
	}
	vus[t] = true
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		collecterTypes(t.Elem(), out, vus, exportesSeuls, champsHors, typesHors)
	case reflect.Map:
		collecterTypes(t.Key(), out, vus, exportesSeuls, champsHors, typesHors)
		collecterTypes(t.Elem(), out, vus, exportesSeuls, champsHors, typesHors)
	case reflect.Struct:
		if _, retire := typesHors[t.Name()]; retire {
			return
		}
		if nom := t.Name(); nom != "" {
			out[nom] = t
		}
		for field := range t.Fields() {
			if exportesSeuls && field.PkgPath != "" {
				continue
			}
			if _, retire := champsHors[t.Name()+"."+field.Name]; retire {
				continue
			}
			collecterTypes(field.Type, out, vus, exportesSeuls, champsHors, typesHors)
		}
	default:
	}
}

// nomDeType : la forme d'un type, PAR NOM et jamais par chemin de paquet — `replay.Track` et
// `replaydoc.Track` sont le meme contrat sous deux paquets ; le chemin d'import est la seule chose
// qui doit differer entre eux. Un type de base se rend par son genre, jamais par son nom declare
// (`LinkMethod` cote stocke et `string` cote servi sont le meme contrat de fil).
func nomDeType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Pointer:
		return "*" + nomDeType(t.Elem())
	case reflect.Slice:
		return "[]" + nomDeType(t.Elem())
	case reflect.Array:
		return fmt.Sprintf("[%d]%s", t.Len(), nomDeType(t.Elem()))
	case reflect.Map:
		return "map[" + nomDeType(t.Key()) + "]" + nomDeType(t.Elem())
	case reflect.Struct:
		if nom := t.Name(); nom != "" {
			return nom
		}
		return structAnonyme(t)
	default:
		return t.Kind().String()
	}
}

// structAnonyme rend une structure sans nom en clair : elle n'entre pas dans la table des
// types, donc sa forme doit voyager avec le champ qui la porte.
func structAnonyme(t reflect.Type) string {
	return "struct{" + strings.Join(champsDe(t, nil), "; ") + "}"
}
