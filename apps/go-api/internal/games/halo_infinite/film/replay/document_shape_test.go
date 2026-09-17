package replay

// document_shape_test.go — L'EMPREINTE DE FORME DU DOCUMENT, MIROIR DE CELLE DU DECODEUR.
//
// CE QU'IL ATTRAPE, ET RIEN D'AUTRE : « j'ai change un champ et oublie le numero ». La forme
// du document — noms de champs, balises JSON, `omitempty`, types, a toute profondeur — est
// hachee et figee dans `testdata/document_shape.golden` AVEC la `SchemaVersion` qui l'a gelee.
// Trois consequences, toutes voulues (architecture §12, garde-rail 4) :
//
//  1. la forme change et `SchemaVersion` ne bouge pas -> la REGENERATION est refusee, avec le
//     message qui dit quoi faire. Sans cela, le parc d'artefacts cuits deviendrait
//     silencieusement illisible : la reprise du backfill se fait par `SchemaVersion`, et un
//     artefact d'une forme ancienne portant le numero courant se lit « a jour » ;
//  2. `SchemaVersion` monte sans entree dans `document_chronicle.go` -> rouge. Une montee sans
//     chronique est une montee dont personne ne saura dire ce qu'elle a change ;
//  3. la forme STOCKEE et la forme SERVIE (`internal/domain/replaydoc`) divergent -> rouge.
//
// SUR LE POINT 3, CE FICHIER NE REFAIT PAS LE TRAVAIL D'UN AUTRE. La parite champ par champ
// entre les deux jumeaux est DEJA tenue, et mieux, par `internal/service/replayview/parity_test.go`
// (decision ecrite par champ non servi, tags JSON confrontes, projection exhaustive eprouvee sur
// les frontieres de nullite). Ce qu'on ajoute ici est la seule chose qu'elle ne fait pas : une
// empreinte UNIQUE des deux formes, qui tient dans le meme golden que la version de schema —
// de sorte qu'une divergence se voie au meme endroit que la montee de version qui l'aurait
// causee. Les deux gardes se completent ; celle-ci ne la remplace pas.
//
// REGENERATION (jamais d'edition a la main) — memes deux conditions explicites que les
// fixtures de contrat, et pour la meme raison (cf. contract_fixtures_test.go) :
//
//	REPLAY_CONTRACT_UPDATE=1 go test ./internal/games/halo_infinite/film/replay/ \
//	  -run DocumentShape -update

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/testutil"
)

// documentShapePath : le golden de forme.
func documentShapePath() string { return filepath.Join(goldenDir, "document_shape.golden") }

// shapeEnteteSchema / shapeEnteteEmpreinte / shapeEnteteServie : les trois lignes d'en-tete du
// golden. Elles sont RELUES par le test (pas seulement ecrites) : c'est par elles qu'une
// regeneration sait quelle version avait gele la forme precedente.
const (
	shapeEnteteSchema    = "schema "
	shapeEnteteEmpreinte = "empreinte-stockee "
	shapeEnteteServie    = "empreinte-servie "
)

// TestDocumentShapeMatchesGolden : la forme du document est-elle toujours celle qui est figee ?
func TestDocumentShapeMatchesGolden(t *testing.T) {
	got := documentShapeGolden()
	want, err := os.ReadFile(documentShapePath()) //nolint:gosec // chemin fige dans le code
	if err != nil {
		t.Fatalf("golden de forme absent : %v — regenerer avec %s=1 go test -run DocumentShape -update",
			err, contractFixturesEnv)
	}
	if string(want) == got {
		return
	}
	t.Errorf("la FORME du document de rejeu a change par rapport a %s.\n%s\n"+
		"Si le contenu cuit change : monter SchemaVersion, ecrire son entree dans "+
		"document_chronicle.go, puis regenerer.", documentShapePath(),
		premierEcartAssembly(string(want), got))
}

// TestDocumentShapeTwinsAgree : la forme STOCKEE et la forme SERVIE coincident-elles ?
//
// Elles le doivent : `domain/replaydoc` est le jumeau de fil de `replay.ReplayDocument`, meme
// noms de types, memes tags JSON, memes `omitempty` (cf. son `doc.go`). Une divergence signifie
// qu'un calque cuit n'atteint plus le client, ou qu'un champ servi n'a plus de source.
func TestDocumentShapeTwinsAgree(t *testing.T) {
	stockee := documentShapeRender(reflect.TypeOf(ReplayDocument{}))
	servie := documentShapeRender(reflect.TypeOf(replaydoc.ReplayDocument{}))
	if stockee == servie {
		return
	}
	t.Errorf("la forme STOCKEE et la forme SERVIE divergent (empreintes %s et %s).\n%s",
		empreinteDe(stockee), empreinteDe(servie), premierEcartAssembly(stockee, servie))
}

// TestDocumentShapeGoldenCarriesCurrentSchema : le golden a-t-il ete gele a la version
// COURANTE ? Un golden fige a une version anterieure signifie qu'une montee est passee sans
// que la forme soit re-figee — le ratchet ne garderait plus rien.
func TestDocumentShapeGoldenCarriesCurrentSchema(t *testing.T) {
	if got := shapeGoldenSchema(t); got != SchemaVersion {
		t.Errorf("le golden de forme est gele au schema %d, le producteur ecrit %d — regenerer",
			got, SchemaVersion)
	}
}

// TestDocumentShapeSchemaHasChronicleEntry : la version courante a-t-elle son entree de
// chronique ? Une montee sans chronique est une montee dont personne ne saura dire ce qu'elle
// a change — et la reprise du backfill se fait par ce numero.
func TestDocumentShapeSchemaHasChronicleEntry(t *testing.T) {
	versions, err := testutil.ReplayChronicleVersions()
	if err != nil {
		t.Fatalf("chronique illisible : %v", err)
	}
	for _, v := range versions {
		if v == SchemaVersion {
			return
		}
	}
	t.Errorf("SchemaVersion = %d n'a AUCUNE entree dans document_chronicle.go (versions "+
		"declarees : %v) — une montee sans chronique ne dit pas ce qu'elle change", SchemaVersion, versions)
}

// TestDocumentShapeRegenerate : LA SEULE PORTE D'ECRITURE du golden de forme.
//
// ELLE REFUSE LA REGENERATION QUAND LA FORME A CHANGE SANS QUE `SchemaVersion` MONTE. C'est
// tout le ratchet : sans ce refus, la reponse naturelle a un golden rouge serait de le
// regenerer, et la montee de version — la seule chose qui fasse recuire le parc — serait
// oubliee precisement quand elle est necessaire.
func TestDocumentShapeRegenerate(t *testing.T) {
	switch {
	case !*updateGolden:
		t.Skip("regeneration du golden de forme : passer -update (et " + contractFixturesEnv + "=1)")
	case os.Getenv(contractFixturesEnv) == "":
		t.Skip("regeneration du golden de forme : " + contractFixturesEnv + " non defini")
	}
	ancienSchema, _ := shapeGoldenTete()
	// LE REFUS SE LIT SUR LA FORME CUITE, jamais sur la forme entière : les calques résolus à
	// la requête ne périment aucun artefact (cf. la section « CALQUES RÉSOLUS À LA REQUÊTE »).
	ancienneEmpreinte := shapeGoldenEmpreinteCuite()
	nouvelle := empreinteDe(documentShapeRenderCuite())
	// DEUX EMPREINTES CALCULÉES PAR DEUX RÈGLES DIFFÉRENTES NE SE COMPARENT PAS. Quand la règle
	// elle-même se corrige (cf. `cuiteRegleCourante`), le refus n'a aucun sens : il porterait
	// sur un changement qui ne touche AUCUN artefact, et la seule sortie serait de monter
	// `SchemaVersion` — donc de faire lire « à re-cuire » tout le parc, pour rien. Le numéro de
	// règle rend ce cas EXPLICITE au lieu de le faire contourner à la main.
	if shapeGoldenCuiteRegle() != cuiteRegleCourante {
		t.Logf("regle de l'empreinte cuite %d -> %d : le refus ne s'applique pas (les deux "+
			"empreintes ne sont pas comparables), le golden est re-fige tel quel",
			shapeGoldenCuiteRegle(), cuiteRegleCourante)
		ancienneEmpreinte = ""
	}
	if ancienneEmpreinte != "" && ancienneEmpreinte != nouvelle && ancienSchema == SchemaVersion {
		t.Fatalf("REGENERATION REFUSEE : la forme du document a change (empreinte %s -> %s) "+
			"alors que SchemaVersion est reste a %d. Monter SchemaVersion et ecrire son entree "+
			"dans document_chronicle.go AVANT de regenerer — sinon le parc d'artefacts deja "+
			"cuits se lira « a jour » avec l'ancienne forme.", ancienneEmpreinte, nouvelle, SchemaVersion)
	}
	if err := os.MkdirAll(goldenDir, 0o750); err != nil {
		t.Fatalf("creation de %s : %v", goldenDir, err)
	}
	if err := os.WriteFile(documentShapePath(), []byte(documentShapeGolden()), 0o600); err != nil {
		t.Fatalf("ecriture de %s : %v", documentShapePath(), err)
	}
	// UNE PORTE DE REGENERATION NE REND JAMAIS `ok` (revue R1 constat R1-8, parite achevee a la
	// ronde 2) : `go test` jette la sortie d un paquet qui PASSE, donc un `t.Logf` est invisible
	// avec la commande documentee.
	t.Fatalf("golden de forme reecrit : %s (schema %d, empreinte %s) ; relancer sans -update "+
		"pour verifier", documentShapePath(), SchemaVersion, nouvelle)
}

// documentShapeGolden : le contenu complet du golden — l'en-tete, puis la forme en clair.
//
// LA FORME EN CLAIR EST DANS LE GOLDEN, et pas seulement son hachage : une empreinte qui bouge
// ne dit pas CE QUI a bouge, et la premiere question d'une revue est toujours celle-la.
func documentShapeGolden() string {
	render := documentShapeRender(reflect.TypeOf(ReplayDocument{}))
	var b strings.Builder
	b.WriteString("# EMPREINTE DE FORME DU DOCUMENT DE REJEU — fige par document_shape_test.go.\n")
	b.WriteString("# Ne s'edite JAMAIS a la main : regeneration decrite en tete de ce test.\n")
	fmt.Fprintf(&b, "%s%d\n", shapeEnteteSchema, SchemaVersion)
	fmt.Fprintf(&b, "%s%s\n", shapeEnteteEmpreinte, empreinteDe(render))
	fmt.Fprintf(&b, "%s%s\n", shapeEnteteServie,
		empreinteDe(documentShapeRender(reflect.TypeOf(replaydoc.ReplayDocument{}))))
	fmt.Fprintf(&b, "%s%s\n", shapeEnteteCuite, empreinteDe(documentShapeRenderCuite()))
	fmt.Fprintf(&b, "%s%d\n\n", shapeEnteteCuiteRegle, cuiteRegleCourante)
	b.WriteString(render)
	return b.String()
}

// shapeGoldenTete relit l'en-tete du golden en place. Rend (0, "") si le golden n'existe pas
// encore — le premier gel n'a rien a comparer.
func shapeGoldenTete() (schema int, empreinte string) {
	raw, err := os.ReadFile(documentShapePath()) //nolint:gosec // chemin fige dans le code
	if err != nil {
		return 0, ""
	}
	for _, ligne := range strings.Split(string(raw), "\n") {
		switch {
		case strings.HasPrefix(ligne, shapeEnteteSchema):
			_, _ = fmt.Sscanf(strings.TrimPrefix(ligne, shapeEnteteSchema), "%d", &schema)
		case strings.HasPrefix(ligne, shapeEnteteEmpreinte):
			empreinte = strings.TrimSpace(strings.TrimPrefix(ligne, shapeEnteteEmpreinte))
		}
	}
	return schema, empreinte
}

// shapeGoldenSchema : la version sous laquelle le golden a ete gele.
func shapeGoldenSchema(t *testing.T) int {
	t.Helper()
	schema, _ := shapeGoldenTete()
	if schema == 0 {
		t.Fatalf("golden de forme absent ou sans ligne %q", strings.TrimSpace(shapeEnteteSchema))
	}
	return schema
}

// empreinteDe : le hachage court d'une forme.
func empreinteDe(render string) string {
	sum := sha256.Sum256([]byte(render))
	return hex.EncodeToString(sum[:8])
}

// documentShapeRender rend la forme d'un type racine : tous les types struct NOMMES qu'il
// atteint, tries par nom, chacun avec ses champs exportes tries par nom eux aussi.
//
// TOUT EST TRIE, ET L'ORDRE DE DECLARATION EST DELIBEREMENT IGNORE. Deux raisons, dans cet
// ordre d'importance. (1) L'ORDRE DES CHAMPS N'EST PAS LE CONTRAT : un lecteur JSON lit des
// cles, jamais un rang — le depot le dit deja ailleurs, `replayview/parity_test.go` comparant
// les deux jumeaux par des ARBRES (`reflect.DeepEqual` sur des cartes), donc sans ordre.
// Faire rougir un deplacement de champ serait un faux positif, et un ratchet qui cri sur du
// vide finit ignore. (2) C'est ce qui rend les deux jumeaux COMPARABLES : `replay.Coverage` et
// `replaydoc.Coverage` declarent les memes champs dans un ordre different (constat du
// 2026-09-13, 903 lignes de forme de part et d'autre).
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
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
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
		for i := 0; i < t.NumField(); i++ {
			collecterTypes(t.Field(i).Type, out, vus)
		}
	default:
	}
}

// nomDeType : la forme d'un type, PAR NOM et jamais par chemin de paquet.
//
// C'EST CE QUI REND LES DEUX JUMEAUX COMPARABLES : `replay.Track` et `replaydoc.Track` sont le
// meme contrat sous deux paquets ; le chemin d'import est justement la seule chose qui doit
// differer entre eux.
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
		// LE TYPE DE BASE, JAMAIS LE NOM DECLARE : `LinkMethod` (cote stocke) et `string`
		// (cote servi) sont le MEME contrat de fil — un client lit une chaine dans les deux
		// cas. Nommer l'alias ferait rougir les jumeaux sur une difference que le JSON ne
		// porte pas (constat du 2026-09-13 : `IdentityLink.Method`).
		return t.Kind().String()
	}
}

// structAnonyme rend une structure sans nom en clair : elle n'entre pas dans la table des
// types, donc sa forme doit voyager avec le champ qui la porte.
func structAnonyme(t reflect.Type) string {
	return "struct{" + strings.Join(champsDe(t), "; ") + "}"
}

// ---------------------------------------------------------------------------------------
// LES CALQUES RÉSOLUS À LA REQUÊTE — pourquoi ils ont leur propre empreinte (2026-09-14).
//
// LE PROBLÈME, CONSTATÉ EN AJOUTANT `MapWeaponPadDTO.Family` (plan des niveaux d'armes,
// étape 1.1). Deux calques du document ne sont JAMAIS ÉCRITS PAR LA CUISSON : `mapObjectives`
// et `mapWeaponPads`. L'artefact du film ne nomme ni la carte ni le mode, donc ces deux-là se
// remplissent AU SERVICE, à chaque requête, depuis une référence versionnée (cf. les en-têtes
// de map_weapon_pads.go et map_objectives.go). Aucun `Set` de la chaîne de cuisson ne les
// touche — vérifié par grep sur tout le paquet le 2026-09-14.
//
// CONSÉQUENCE : changer LEUR forme ne périme AUCUN artefact cuit. Or le ratchet refusait la
// régénération sans montée de `SchemaVersion`, et une montée aurait fait lire « à re-cuire »
// les 77 artefacts du parc pour un champ qu'aucun d'eux ne porte — exactement le dégât que ce
// garde-fou existe pour éviter, retourné contre lui-même. C'est aussi la décision D3 du plan
// des niveaux d'armes : « aucun champ nouveau dans l'artefact, SchemaVersion inchangé ».
//
// CE QUI EST GARDÉ, ET CE QUI EST ASSOUPLI. L'empreinte `empreinte-stockee` couvre TOUJOURS
// la forme entière, ces deux calques compris : une modification reste visible et oblige
// toujours à régénérer le golden, donc à passer en revue. Seul le REFUS DE RÉGÉNÉRATION se
// lit désormais sur `empreinte-cuite`, qui ignore les types atteignables par ces seuls deux
// champs. Un champ ajouté à un calque CUIT continue d'exiger la montée de version.
// ---------------------------------------------------------------------------------------

// shapeEnteteCuite : la quatrième ligne d'en-tête du golden.
const shapeEnteteCuite = "empreinte-cuite "

// shapeEnteteCuiteRegle : la cinquième ligne — la VERSION DE LA RÈGLE de l'empreinte cuite.
const shapeEnteteCuiteRegle = "empreinte-cuite-regle "

// cuiteRegleCourante — la version de la RÈGLE de calcul de l'empreinte cuite.
//
// # POURQUOI CE NUMÉRO EXISTE
//
// L'empreinte cuite gouverne le REFUS de régénération. Tant qu'elle est calculée de la même
// façon, la comparer d'une version à l'autre a un sens. Mais le jour où la RÈGLE elle-même se
// corrige, l'ancienne et la nouvelle empreinte ne sont plus comparables — et le refus se
// déclenche sur un changement qui ne touche AUCUN artefact. Sans ce numéro, la seule sortie
// serait de monter `SchemaVersion` (qui ferait lire « à re-cuire » tout le parc, pour rien) ou
// de forcer le golden à la main (ce que l'en-tête de ce fichier interdit).
//
//	1 (2026-09-14) première version : les types atteignables par les seuls calques de requête
//	  étaient exclus, mais PAS les lignes de champ de la racine.
//	2 (2026-09-14) les lignes de champ de la racine le sont aussi. Sans ce correctif, l'AJOUT
//	  d'un calque de requête (`weaponTiers`) faisait bouger l'empreinte cuite — exactement ce
//	  qu'elle existe pour ne pas faire.
const cuiteRegleCourante = 2

// calquesALaRequete — les balises JSON des champs du document que la CUISSON N'ÉCRIT JAMAIS.
// Toute entrée ici se justifie par un grep : aucun chemin de `build*.go` ne pose le champ.
// Dernière vérification : 2026-09-17 — `vehicleLabels` y entre au schéma 62 (cf. layers.go).
var calquesALaRequete = map[string]bool{
	"mapObjectives": true, // objectives_catalog.go + service/replay_map_objectives.go
	"mapWeaponPads": true, // map_weapon_pads_catalog.go + service/replay_map_weapon_pads.go
	"weaponTiers":   true, // map_weapon_pads.go (WeaponTiersInfo) + service/replay_weapon_tiers.go
	"vehicleLabels": true, // service/replay_vehicle_labels.go (resolveVehicleLabels) — sprites de chassis
}

// documentShapeRenderCuite rend la forme du document PRIVÉE des types que seuls les calques
// résolus à la requête atteignent. Un type partagé avec un calque cuit y reste.
func documentShapeRenderCuite() string {
	root := reflect.TypeOf(ReplayDocument{})
	types := map[string]reflect.Type{root.Name(): root}
	vus := map[reflect.Type]bool{root: true}
	for i := 0; i < root.NumField(); i++ {
		f := root.Field(i)
		if f.PkgPath != "" || calquesALaRequete[baliseJSON(f)] {
			continue
		}
		collecterTypes(f.Type, types, vus)
	}
	noms := make([]string, 0, len(types))
	for nom := range types {
		noms = append(noms, nom)
	}
	sort.Strings(noms)
	var b strings.Builder
	for _, nom := range noms {
		b.WriteString(nom + "\n")
		// LA RACINE VOIT SES CHAMPS FILTRÉS, les autres types non. Sans ce filtre, l'empreinte
		// cuite bougeait à l'AJOUT d'un calque de requête — la racine déclare toujours son
		// champ — et exigeait donc une montée de `SchemaVersion` pour un champ qu'aucun
		// artefact ne porte. C'est exactement ce que cette empreinte existe pour ne PAS faire
		// (constat payé le 2026-09-14 en ajoutant `weaponTiers`). L'ORDRE reste celui du tri :
		// sortir la racine de la boucle changerait l'empreinte sans rien changer au fond.
		champs := champsDe(types[nom])
		if types[nom] == root {
			champs = champsCuitsDe(root)
		}
		for _, ligne := range champs {
			b.WriteString("  " + ligne + "\n")
		}
	}
	return b.String()
}

// champsCuitsDe : les champs exportés d'un type, PRIVÉS des calques résolus à la requête.
func champsCuitsDe(t reflect.Type) []string {
	out := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" || calquesALaRequete[baliseJSON(f)] {
			continue
		}
		out = append(out, fmt.Sprintf("%s json:%q %s", f.Name, f.Tag.Get("json"), nomDeType(f.Type)))
	}
	sort.Strings(out)
	return out
}

// baliseJSON rend le nom de clé JSON d'un champ (avant la virgule des options).
func baliseJSON(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if i := strings.IndexByte(tag, ','); i >= 0 {
		return tag[:i]
	}
	return tag
}

// shapeGoldenEmpreinteCuite relit l'empreinte cuite du golden en place ("" si absente).
func shapeGoldenEmpreinteCuite() string {
	raw, err := os.ReadFile(documentShapePath()) //nolint:gosec // chemin fige dans le code
	if err != nil {
		return ""
	}
	for _, ligne := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(ligne, shapeEnteteCuiteRegle) {
			continue // la ligne de RÈGLE commence par le même préfixe : ne pas la confondre
		}
		if strings.HasPrefix(ligne, shapeEnteteCuite) {
			return strings.TrimSpace(strings.TrimPrefix(ligne, shapeEnteteCuite))
		}
	}
	return ""
}

// shapeGoldenCuiteRegle relit la version de RÈGLE sous laquelle l'empreinte cuite du golden a
// été calculée. Rend 0 quand la ligne est absente — un golden d'avant l'introduction du numéro.
func shapeGoldenCuiteRegle() int {
	raw, err := os.ReadFile(documentShapePath()) //nolint:gosec // chemin fige dans le code
	if err != nil {
		return 0
	}
	for _, ligne := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(ligne, shapeEnteteCuiteRegle) {
			var v int
			_, _ = fmt.Sscanf(strings.TrimPrefix(ligne, shapeEnteteCuiteRegle), "%d", &v)
			return v
		}
	}
	return 0
}

// TestDocumentShapeCalquesALaRequeteRestentHorsCuisson — LE GARDE-FOU DU GARDE-FOU.
//
// L'assouplissement ci-dessus ne tient que tant que ces deux calques restent absents de la
// cuisson. Le jour où un chemin de `build*.go` en poserait un, l'artefact porterait un contenu
// dont la forme ne serait plus ratchetée — et ce test rougit AVANT.
func TestDocumentShapeCalquesALaRequeteRestentHorsCuisson(t *testing.T) {
	root := reflect.TypeOf(ReplayDocument{})
	for balise := range calquesALaRequete {
		trouve := false
		for i := 0; i < root.NumField(); i++ {
			if baliseJSON(root.Field(i)) == balise {
				trouve = true
			}
		}
		if !trouve {
			t.Errorf("le calque %q est declare a la requete mais n'existe plus au document — "+
				"retirer son entree de calquesALaRequete", balise)
		}
	}
	fichiers, err := filepath.Glob("build*.go")
	if err != nil {
		t.Fatalf("balayage des fichiers de cuisson : %v", err)
	}
	for _, f := range fichiers {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f) //nolint:gosec // chemin du paquet courant
		if err != nil {
			t.Fatalf("lecture de %s : %v", f, err)
		}
		for _, champ := range []string{"MapObjectives", "MapWeaponPads", "WeaponTiers", "VehicleLabels"} {
			if strings.Contains(string(src), "."+champ+" =") {
				t.Errorf("%s ecrit %s a la CUISSON : ce calque n'est plus resolu a la requete, "+
					"retirer son entree de calquesALaRequete et remonter SchemaVersion", f, champ)
			}
		}
	}
}
