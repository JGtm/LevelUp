package grammar

// lecteur_position_ratchet_test.go — LA TABLE DES SITES D APPEL DE `FUN_14076e524` ET DE LEURS
// IMMEDIATS (lot J6.3 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, DU-1).
//
// # CE QUE CE RATCHET TIENT
//
// Une fonction du jeu, UN portage (`lecteur_position.go`). Chaque site Go qui lit une position
// quantifiee appelle l un des points d entree du portage (`lireE524`, `lireE494`, `lireE420`,
// `lireE494Sur`) avec l IMMEDIAT DE NIVEAU que le jeu passe a CE site — releve dans Ghidra le
// 2026-09-27 (`.ai/V7.5/film_re/RELEVES_J6_GHIDRA_2026-09-27.md`, colonnes « appelant / CALL /
// niveau » de son §1). Le test, par l AST des sources de production du paquet :
//
//   - releve chaque appel d un point d entree, sa fonction englobante, son `case` englobant quand
//     il y en a un (les maillons de dispatch portent plusieurs sites), et la VALEUR de son niveau ;
//   - exige que l ensemble releve soit EXACTEMENT la table : un site dont l immediat change, un
//     site neuf hors table ou un site disparu rougit ;
//   - interdit, hors du portage, les formes d un lecteur local : les largeurs du profil
//     (`worldObjectPrecision().AxisW` / `.IndexW`, `traversal().AxisW` / `.IndexW`), la loi des
//     largeurs appelee a la main, la largeur `6 + uint(level)` et les noms des neuf portages
//     retires.
//
// # MUTATIONS JOUEES (2026-09-27), ROUGES, PUIS RETIREES — consignees au rapport du lot J6.3.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// fichierDuPortage : le seul fichier autorise a lire les tables du lecteur de position.
const fichierDuPortage = "lecteur_position.go"

// siteDePosition est une ligne de la table : ou le Go lit, par quelle entree, a quel niveau, et
// ce que le jeu dit du site.
type siteDePosition struct {
	fonction string // la fonction Go englobante
	cas      string // l etiquette du `case` englobant, vide hors d un switch de dispatch
	entree   string // le point d entree du portage
	niveau   int64  // l immediat de niveau du jeu
	appels   int    // nombre d appels a CE niveau dans CE site
	jeu      string // appelant et adresse d appel (releve J6.1)
}

// tableDesSitesDePosition — LA TABLE. Chaque ligne cite le site du jeu.
func tableDesSitesDePosition() []siteDePosition {
	const n10, n1e = 0x10, 0x1e
	return []siteDePosition{
		{"consumeAbsoluteWithGate", "", "lireE524", n10, 1, "FUN_1406cfe44 branche absolue, precHigh = 0, CALL 1406d009d (MOV R9D,0x10 en 1406d008a)"},
		{"consumePredictedDelta", "", "lireE524", n10, 1, "FUN_14076f3ec repli, CALL 14226a6c7 (14226a6b8)"},
		{"consumePredictedAbsolute", "", "lireE420", n10, 1, "FUN_140f7ea14 -> FUN_14076e4ec, CALL 140f7ea5c"},
		{"consumeObjectPositionDynamicPrecisionD", "", "lireE420", n10, 1, "grammaire d ecrivain d i0 : FUN_14076e29c -> FUN_14076e420, CALL 14076e2c0"},
		{"consumeMobilityActionBody", "", "lireE494", n10, 2, "i54 : FUN_1408f02c8, CALLs 1408f03c7 et 1408f0758 (EBP = 0x10)"},
		{"consumeManagedAndObjectiveComponent", "asset-transform-component", "lireE494", n1e, 1, "ti=44 i0 : FUN_142ed9530, CALL 142ed9556 (x5 par FUN_142ed3c64)"},
		{"consume14058c058", "", "lireE494", n10, 2, "unit-actor-state : FUN_14058c058, CALLs 1422cddc1 et 1422cde0e"},
		{"consumeBipedDefaultStateMediaFrame", "", "lireE494", n10, 1, "trame media : FUN_140f44c38, CALL 142451b5d"},
		{"consumeCrewFlockAndMusicComponent", "tacmap-poiiconoffset", "lireE494", n10, 1, "ti=30 i1 : FUN_142ed485c -> FUN_1424e0e38(0x10) (descripteur 143d06b00)"},
		{"consumeTacmapWaypointState", "", "lireE494", n10, 1, "ti=34 i7 : FUN_140f04d88, CALL 140f04de0 (140f04dd5)"},
		{"consumeSpawnFilterType", "", "lireE494", n10, 1, "ti=20 i0 etiquette 3 : FUN_142b6eeec, CALL 142b6ef31"},
		{"consumeSelectableZoneData", "", "lireE494", n10, 1, "selectable-zone-data : FUN_141454340, CALL 14145437e"},
		{"readTranslocVec", "", "lireE494Sur", n10, 1, "EquipmentTranslocatorTeleportEffects : FUN_140f04fb8, CALLs 140f04ff0 et 140f05023"},
		{"consumeSimulationState", "", "lireE494", n10, 1, "i60 : FUN_142ed6d88, CALL 142ed6fd5"},
		{"lireVecteur1431a0cbc", "modeVecteurQuantifie", "lireE494", n10, 1, "bloc d action : FUN_1431a0cbc, CALL 1431a0d0d"},
		{"consumeDefaultStateTI41", "", "lireE494", n10, 1, "etat par defaut ti=41 : FUN_1408efb58, CALL 1408efe11"},
		{"consume142f04664", "", "lireE494", n10, 1, "sous-lecteur ti=41 : FUN_142f04664, CALL 142f0482b"},
		{"consumeVehicleMediaFrame", "", "lireE494", n10, 1, "etat par defaut ti=40 : FUN_1410a5a74, CALL 1424a3a1e"},
		{"consumePostureTag1", "", "lireE494", n10, 1, "posture etiquette 1 : FUN_142f25a3c, CALL 142f25d46"},
		{"consumePostureTag2", "", "lireE494", n10, 1, "posture etiquette 2 : FUN_142f263ac, CALL 142f263d9"},
		{"consumePostureTag3", "", "lireE494", n10, 1, "posture etiquette 3 : FUN_142f264f4, CALL 142f26586"},
		{"consumeSpartanAbilityTag3", "", "lireE494", n10, 1, "i57 etiquette 3 : FUN_142f262d4, CALL 142f2638b"},
	}
}

// pointsDEntreeDuPortage : les noms que le releve reconnait comme un appel au portage.
var pointsDEntreeDuPortage = map[string]bool{
	"lireE524": true, "lireE524Sur": true, "lireE494": true, "lireE494Sur": true, "lireE420": true,
}

// lecteursLocauxInterdits : hors du portage, ces formes relisent la position a la main.
var lecteursLocauxInterdits = []*regexp.Regexp{
	regexp.MustCompile(`worldObjectPrecision\(\)\.(AxisW|IndexW)`),
	regexp.MustCompile(`traversal\(\)\.(AxisW|IndexW)`),
	regexp.MustCompile(`profile\.LargeursAxe(ParDefautDuBuild|DuNiveau)\(`),
	regexp.MustCompile(`6\s*\+\s*uint\(level\)`),
	regexp.MustCompile(`\b(quantAxisWidth|consumeQuantVec3|consumeQuantVec3WithGate|consumeQuantVec3Values|` +
		`consumeE524PositionBody|consumeQuat16|absAxisWFor|consumeSimStateHandleTail|consumeAbsolutePayload)\(`),
}

// fichierDesExceptions porte les sites qui gardent leur ancien lecteur ([exceptionsDuPortage]).
const fichierDesExceptions = "lecteur_position_exceptions.go"

// critereDeRetraitDesExceptions : le meme pour les onze (decision du superviseur, 2026-09-27 ; lot
// J6-bis, 2026-09-28 ; lot R3, 2026-09-29).
const critereDeRetraitDesExceptions = "la lecture du jeu fait monter la fermeture sans aucune baisse " +
	"sur les bobines, ou la grammaire dependante du build est etablie"

// exceptionDuPortage est un site du jeu que le Go lit HORS du portage unique, par decision datee.
type exceptionDuPortage struct {
	fonction string // la fonction Go, dans `fichierDesExceptions`
	niveau   int64  // l immediat du jeu, releve
	jeu      string // appelant et adresse d appel
	pourquoi string // la mesure de fermeture qui baisse
}

// exceptionsDuPortage — LES EXCEPTIONS DATEES (2026-09-27, lot J6.3, decision du superviseur).
// Critere de retrait : [critereDeRetraitDesExceptions]. La cle est celle des cas de
// `lecteur_position_sites_test.go` marques « ecart attendu ».
func exceptionsDuPortage() map[string]exceptionDuPortage {
	return map[string]exceptionDuPortage{
		"flock-position": {"consumeFlockPosition", 0x10, "ti=21 i16 : FUN_140ee7270, CALL 140ee7293 (140ee7288)",
			"GA2-2 : carte de fermeture, ks_000d5950 paquets 1823 -> 1847 mais ks_e5adf7b2 paquets 371 -> 370, " +
				"ti=21 1/60 -> 0/60, ti=4 240 -> 239"},
		"world-object-i0": {"consumeObjectPositionMonde", 0x10, "world-object i0 : FUN_14076e29c -> FUN_14076e420, CALL 14076e2c0",
			"GA2-5 : image-cle ti=38 11de8353 99 -> 83, a521164d 122 -> 119 ; ti=42 60ae07c4 5 -> 4, " +
				"11de8353 3 -> 2, 111fa685 2 -> 1 (hausses sur les builds recents)"},
		"ti38-i18": {"consumeGenericRigidBodyTransforms", 0x10, "ti=38 i18 : FUN_142f036f0, CALL 142f03837",
			"image-cle ti=38 fb1a1a72 317 -> 245, 111fa685 72 -> 30, 11de8353 99 -> 19, sans aucune hausse"},
		// Lot J6-bis (2026-09-28), meme situation, meme format.
		"flock-destination": {"consumeFlockDestination", 0x10, "ti=21 i2-i11 : FUN_140fb8af0, CALL 140fb8b3e (140fb8b33)",
			"marche des trames de 11de8353 : listes chunk 19 paquet 494 (16 entrees de controle) et chunk 9 " +
				"paquet 1146 (22 entrees) fermees -> non localisees ; hausse sur 000d5950 chunk 20 paquet 1322 (8 entrees)"},
		"respawn-location": {"consumePlayerDesiredRespawnLocation", 0x10, "ti=5 i12 : FUN_142f03ec8 (descripteur 143d0f2f8)",
			"marche des trames de e5adf7b2 : liste chunk 25 paquet 344 (14 entrees) fermee -> non localisee ; " +
				"hausses sur e5adf7b2 chunk 6 paquet 50 (4 entrees) et 111fa685 chunk 14 paquet 552 (2 entrees)"},
		"tacmap-poiicon": {"consumeTacmapPoiIcon", 0x10, "ti=30 i0 : FUN_142ed8418, CALL 142ed86d7 (thunk FUN_1424e0e38)",
			"marche des trames de 11de8353 : liste chunk 21 paquet 1032 fermee -> non localisee, sans " +
				"aucune hausse sur les huit builds"},
		// Lot R3 (2026-09-29), meme situation, meme format ; chiffres = le site seul rendu a son
		// ancien lecteur sur la tete du plan, carte de fermeture de douze films.
		"tacmap-displayasset": {"consumeTacmapDisplayAsset", 0x10, "ti=33 i0 : FUN_142ed7d38, CALL 142ed7edf (thunk FUN_1424e0e38)",
			"marche des trames de 51ebbc0f : dix paquets fermes (chunk 7 et 8, 61 entrees de controle) -> " +
				"non fermes ; hausses sur 51ebbc0f 14:42, 084a804d 46:10, 11de8353 29:208, fb1a1a72 7:2380, 60ae07c4 32:2062"},
		"tacmap-areaofinterest": {"consumeTacmapAreaOfInterest", 0x10, "ti=32 i0 : FUN_142ed7764, CALL 142ed7853 (thunk FUN_1424e0e38)",
			"marche des trames de 51ebbc0f : paquet 12:608 (6 entrees) ferme -> non ferme ; hausses sur " +
				"11de8353 19:394, fb1a1a72 38:8, 60ae07c4 3:1790"},
		"tacmap-cooptetherarea": {"consumeTacmapCoopTetherArea", 0x10, "ti=34 i11 : FUN_142ed4198, CALL 142ed41ba (thunk FUN_1424e0e38)",
			"marche des trames de c75f33b8 : liste chunk 21 paquet 1012 fermee -> non localisee, sans aucune " +
				"hausse sur les douze films"},
		"crew-order": {"consumeCrewOrder", 0x10, "ti=14 i0 : FUN_142ed9120, CALL 142ed918e",
			"marche des trames de 084a804d : liste chunk 22 paquet 538 (14 entrees) fermee -> non localisee ; " +
				"hausse sur e5adf7b2 chunk 4 paquet 900 (12 entrees)"},
		"i0-bipede-prechigh": {"consumePrecHautDuBipede", 0x10, "i0 du bipede, branche absolue precHigh = 1 : FUN_1406cfe44 " +
			"1406d0093 -> 1422f4cb7, FUN_141f85880(&DAT_143b8c6d0, 0x10)",
			"marche des trames : listes 0797ce72 9:138 (6 entrees), 084a804d 25:356 (21) et 37:22 (10) " +
				"fermees -> non localisees, sans aucune hausse sur les douze films"},
	}
}

// exemptionsDeLecteurLocal : fichier -> justification datee. Une forme interdite n y est toleree
// que parce qu elle n est PAS un portage de `FUN_14076e524`, ou parce que le site est une exception
// datee du portage.
var exemptionsDeLecteurLocal = map[string]string{
	"components_biped_anchor.go": "2026-09-27 (lot J6.3) : le corps tag==3 d i59 lit trois axes aux " +
		"largeurs de la carte selon une grammaire MESUREE ; chez l ecrivain `FUN_142f25e90`, " +
		"l appel a `FUN_14076e494(0x10)` (CALL 142f2605d) est dans les etiquettes 4 et 5, pas dans la " +
		"3 que ce port lit — ce n est donc pas un site du lecteur. Decouverte consignee au rapport J6.3.",
	fichierDesExceptions: "2026-09-27 (lot J6.3, decision du superviseur), 2026-09-28 (lot J6-bis) et " +
		"2026-09-29 (lot R3) : les onze exceptions datees du portage unique (`exceptionsDuPortage`), critere de retrait : " + critereDeRetraitDesExceptions,
}

// TestLesExceptionsDuPortageSontEnPlace : chaque exception vit dans son fichier, n appelle PAS le
// portage (sinon elle est migree et l exception doit tomber), et le fichier ne porte rien d autre.
func TestLesExceptionsDuPortageSontEnPlace(t *testing.T) {
	asts, fset := parserFichiers(t, []string{fichierDesExceptions})
	consts := constantesDuPaquet(asts)
	declarees := map[string]bool{}
	for _, d := range asts[fichierDesExceptions].Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			declarees[fd.Name.Name] = true
		}
	}
	attendues := map[string]bool{"largeurAncienneDuFlock": true, "lireVecteurAncienAuNiveauDuRegistre": true,
		"lireCorpsDeTraverseeAncien": true}
	for cle, e := range exceptionsDuPortage() {
		attendues[e.fonction] = true
		if !declarees[e.fonction] {
			t.Errorf("exception %s : %s absente de %s", cle, e.fonction, fichierDesExceptions)
		}
		if e.niveau != niveauPosition {
			t.Errorf("exception %s : immediat 0x%x, le releve dit 0x10", cle, e.niveau)
		}
	}
	for nom := range declarees {
		if !attendues[nom] {
			t.Errorf("%s declare %s, qui n est pas une exception datee du portage", fichierDesExceptions, nom)
		}
	}
	for _, a := range appelsAuPortage(t, fichierDesExceptions, asts[fichierDesExceptions], fset, consts) {
		t.Errorf("%s : le site %s appelle le portage (%s) — il est migre : retirer son exception",
			a.ligne, a.fonction, a.entree)
	}
}

// appelReleve est un appel au portage trouve dans les sources.
type appelReleve struct {
	fonction, cas, entree string
	niveau                int64
	ligne                 string
}

// TestLesSitesDePositionPassentParLePortageUnique — LE RATCHET.
func TestLesSitesDePositionPassentParLePortageUnique(t *testing.T) {
	fichiers := fichiersDeProductionDuPaquet(t)
	asts, fset := parserFichiers(t, fichiers)
	consts := constantesDuPaquet(asts)
	releves := map[string][]appelReleve{}
	for _, nom := range cles(asts) {
		if nom == fichierDuPortage {
			continue
		}
		for _, a := range appelsAuPortage(t, nom, asts[nom], fset, consts) {
			k := cleDeSite(a.fonction, a.cas, a.entree, a.niveau)
			releves[k] = append(releves[k], a)
		}
	}
	attendus := map[string]siteDePosition{}
	for _, s := range tableDesSitesDePosition() {
		attendus[cleDeSite(s.fonction, s.cas, s.entree, s.niveau)] = s
	}
	for k, s := range attendus {
		if got := len(releves[k]); got != s.appels {
			t.Errorf("site %s (%s) : %d appel(s) au portage, %d attendu(s) — le site a-t-il quitte "+
				"le portage, ou change d immediat ?", k, s.jeu, got, s.appels)
		}
	}
	for k, as := range releves {
		if _, ok := attendus[k]; !ok {
			for _, a := range as {
				t.Errorf("%s : appel au portage HORS TABLE (%s) — un site neuf, ou un immediat qui "+
					"n est pas celui du jeu. L ajouter a la table avec son appelant et son CALL "+
					"releves chez le jeu.", a.ligne, k)
			}
		}
	}
	for _, nom := range cles(asts) {
		if nom == fichierDuPortage {
			continue
		}
		verifierLecteursLocaux(t, nom)
	}
}

// verifierLecteursLocaux rougit sur une forme de lecteur local hors exemption.
func verifierLecteursLocaux(t *testing.T, nom string) {
	t.Helper()
	if _, ok := exemptionsDeLecteurLocal[nom]; ok {
		return
	}
	src, err := os.ReadFile(nom) //nolint:gosec // fichier du paquet, liste par Glob
	if err != nil {
		t.Fatal(err)
	}
	for i, ligne := range strings.Split(string(src), "\n") {
		code := ligne
		if j := strings.Index(code, "//"); j >= 0 {
			code = code[:j] // les commentaires peuvent citer l histoire
		}
		for _, re := range lecteursLocauxInterdits {
			if re.MatchString(code) {
				t.Errorf("%s:%d : lecteur de position LOCAL (%q) hors de %s — une fonction du jeu, un "+
					"portage : appeler lireE524 / lireE494 / lireE420 avec l immediat du site",
					nom, i+1, strings.TrimSpace(ligne), fichierDuPortage)
			}
		}
	}
}

// cleDeSite forme la cle d une ligne de table.
func cleDeSite(fonction, cas, entree string, niveau int64) string {
	return fonction + "|" + cas + "|" + entree + "|0x" + strconv.FormatInt(niveau, 16)
}

// fichiersDeProductionDuPaquet rend les sources non `_test.go` du paquet, triees.
func fichiersDeProductionDuPaquet(t *testing.T) []string {
	t.Helper()
	tous, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range tous {
		if !strings.HasSuffix(f, "_test.go") {
			out = append(out, f)
		}
	}
	if len(out) < 100 {
		t.Fatalf("seulement %d fichiers de production : le paquet a-t-il demenage ?", len(out))
	}
	sort.Strings(out)
	return out
}

// parserFichiers rend l AST de chaque fichier.
func parserFichiers(t *testing.T, fichiers []string) (map[string]*ast.File, *token.FileSet) {
	t.Helper()
	fset := token.NewFileSet()
	out := map[string]*ast.File{}
	for _, f := range fichiers {
		a, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatalf("parse de %s : %v", f, err)
		}
		out[f] = a
	}
	return out, fset
}

// constantesDuPaquet rend, par nom, l expression de chaque constante declaree avec une valeur.
func constantesDuPaquet(asts map[string]*ast.File) map[string]ast.Expr {
	out := map[string]ast.Expr{}
	for _, f := range asts {
		ast.Inspect(f, func(n ast.Node) bool {
			g, ok := n.(*ast.GenDecl)
			if !ok || g.Tok != token.CONST {
				return true
			}
			for _, s := range g.Specs {
				vs := s.(*ast.ValueSpec)
				for i, nom := range vs.Names {
					if i < len(vs.Values) {
						out[nom.Name] = vs.Values[i]
					}
				}
			}
			return true
		})
	}
	return out
}

// appelsAuPortage rend les appels au portage d un fichier, avec fonction, `case` et niveau.
func appelsAuPortage(t *testing.T, nom string, f *ast.File, fset *token.FileSet, consts map[string]ast.Expr) []appelReleve {
	t.Helper()
	var out []appelReleve
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		var pile []ast.Node
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			if n == nil {
				pile = pile[:len(pile)-1]
				return true
			}
			pile = append(pile, n)
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			id, ok := call.Fun.(*ast.Ident)
			if !ok || !pointsDEntreeDuPortage[id.Name] {
				return true
			}
			ligne := nom + ":" + strconv.Itoa(fset.Position(call.Pos()).Line)
			if len(call.Args) < 2 {
				t.Errorf("%s : appel a %s sans niveau", ligne, id.Name)
				return true
			}
			niveau, ok := valeurEntiere(call.Args[1], consts, 0)
			if !ok {
				t.Errorf("%s : le niveau de %s n est pas une CONSTANTE — l immediat du jeu doit se lire "+
					"dans le code", ligne, id.Name)
				return true
			}
			out = append(out, appelReleve{fonction: fd.Name.Name, cas: casEnglobant(pile),
				entree: id.Name, niveau: niveau, ligne: ligne})
			return true
		})
	}
	return out
}

// casEnglobant rend l etiquette du `case` le plus proche dans la pile : la chaine litterale, ou
// le nom de la constante.
func casEnglobant(pile []ast.Node) string {
	for i := len(pile) - 1; i >= 0; i-- {
		cc, ok := pile[i].(*ast.CaseClause)
		if !ok || len(cc.List) == 0 {
			continue
		}
		switch e := cc.List[0].(type) {
		case *ast.BasicLit:
			if s, err := strconv.Unquote(e.Value); err == nil {
				return s
			}
		case *ast.Ident:
			return e.Name
		}
		return ""
	}
	return ""
}

// valeurEntiere evalue une expression entiere de litteraux et de constantes du paquet.
func valeurEntiere(e ast.Expr, consts map[string]ast.Expr, profondeur int) (int64, bool) {
	if profondeur > 16 {
		return 0, false
	}
	switch v := e.(type) {
	case *ast.ParenExpr:
		return valeurEntiere(v.X, consts, profondeur+1)
	case *ast.BasicLit:
		if v.Kind != token.INT {
			return 0, false
		}
		n, err := strconv.ParseInt(v.Value, 0, 64)
		return n, err == nil
	case *ast.Ident:
		def, ok := consts[v.Name]
		if !ok {
			return 0, false
		}
		return valeurEntiere(def, consts, profondeur+1)
	}
	return 0, false
}

// cles rend les cles d une table d AST, triees.
func cles(m map[string]*ast.File) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
