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

	"github.com/JGtm/LevelUp/libs/film/replay"
)

// cheminGoldenDeForme : le golden de forme de la brique, depuis la racine du depot.
var cheminGoldenDeForme = filepath.Join("libs", "film", "replay", "testdata", "document_shape.golden")

// TestDocumentShapeTwinsAgree : la forme STOCKEE et la forme SERVIE coincident-elles ?
//
// Elles le doivent : `domain/replaydoc` est le jumeau de fil de `replay.ReplayDocument`, meme
// noms de types, memes tags JSON, memes `omitempty` (cf. son `doc.go`). Une divergence signifie
// qu'un calque cuit n'atteint plus le client, ou qu'un champ servi n'a plus de source.
func TestDocumentShapeTwinsAgree(t *testing.T) {
	stockee := documentShapeRender(reflect.TypeFor[replay.ReplayDocument]())
	servie := documentShapeRender(reflect.TypeFor[replaydoc.ReplayDocument]())
	if stockee == servie {
		return
	}
	t.Errorf("la forme STOCKEE et la forme SERVIE divergent (empreintes %s et %s).\n%s",
		empreinteDe(stockee), empreinteDe(servie), premierEcartTexte(stockee, servie))
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
	chemin := filepath.Join(racineDuDepot(t), cheminGoldenDeForme)
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
	types := map[string]reflect.Type{}
	collecterTypes(root, types, map[reflect.Type]bool{})
	noms := make([]string, 0, len(types))
	for nom := range types {
		noms = append(noms, nom)
	}
	sort.Strings(noms)
	var b strings.Builder
	for _, nom := range noms {
		b.WriteString(nom + "\n")
		for _, ligne := range champsDe(types[nom]) {
			b.WriteString("  " + ligne + "\n")
		}
	}
	return b.String()
}

// champsDe : les champs exportes d'un type, rendus et tries par nom.
func champsDe(t reflect.Type) []string {
	out := make([]string, 0, t.NumField())
	for f := range t.Fields() {
		if f.PkgPath != "" {
			continue // champ non exporte : jamais serialise
		}
		out = append(out, fmt.Sprintf("%s json:%q %s", f.Name, f.Tag.Get("json"), nomDeType(f.Type)))
	}
	sort.Strings(out)
	return out
}

// collecterTypes recense les types struct NOMMES atteignables. La carte des types deja vus
// borne la recursion : le document porte des structures qui se referencent.
func collecterTypes(t reflect.Type, out map[string]reflect.Type, vus map[reflect.Type]bool) {
	if t == nil || vus[t] {
		return
	}
	vus[t] = true
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		collecterTypes(t.Elem(), out, vus)
	case reflect.Map:
		collecterTypes(t.Key(), out, vus)
		collecterTypes(t.Elem(), out, vus)
	case reflect.Struct:
		if nom := t.Name(); nom != "" {
			out[nom] = t
		}
		for field := range t.Fields() {
			collecterTypes(field.Type, out, vus)
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
	return "struct{" + strings.Join(champsDe(t), "; ") + "}"
}
