package archlint

// no_raw_film_bytes_extraction_test.go — LE MOTIF STRUCTUREL DU RATCHET « UNE SEULE PORTE AUX
// OCTETS » : EXTRAIRE UN BIT D UN TAMPON D OCTETS, SOUS N IMPORTE QUEL NOM (lot J4.6 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision DU-3 option S2, ADR 0034 D-2).
//
// # POURQUOI UN SIXIEME MOTIF
//
// Les cinq motifs d origine reperent un lecteur de bits par son NOM (`bitAt`, `bitsN`, ...) ou
// par la FORME de sa structure (`[]byte` + position en bits). Sept lecteurs de la grammaire leur
// echappaient (mesure du plan, 2026-09-25) : `readBitsAt`, `PeekBits`, `kfReadBits`,
// `kfReadBitsLoop`, `kfBitAt`, `invBitAt`, `invBits` — des FONCTIONS sans curseur, sous des noms
// que personne n avait listes. Le lot J4.6 les a remplacees par les conventions nommees de la
// couche `source` (`BitsBourres`, `BitsStricts`, `BitsTolerants`, `BitAt`). Ce motif-ci ferme la
// porte par la FORME DU CALCUL : extraire un bit d un octet, c est adresser l octet d une
// position EN BITS, ou decaler un octet d une quantite qui varie. Quel que soit le nom de la
// fonction qui le fait, elle rougit.
//
// # LES DEUX FORMES, MECANIQUEMENT
//
//	octet d une position en bits   une indexation `X[p >> 3]` ou `X[p / 8]` : c est l adresse de
//	                               l octet qui porte le bit `p`, quand `p` n est pas une
//	                               constante (`Slots[0x28/8]` adresse un mot d un tableau, pas
//	                               un bit : faux positif mesure le 2026-09-26, ecarte par cette
//	                               condition). L indice peut passer par une variable locale
//	                               affectee de cette forme (`idx := p >> 3`, `byteIdx :=
//	                               (bitPos + i) / 8`) : elle compte aussi.
//	decalage variable d un octet   un decalage `X[i] >> k` / `X[i] << k` dont la quantite `k`
//	                               n est pas une constante, sur un `X` declare `[]byte` par la
//	                               fonction elle-meme (parametre, resultat, variable) ou, a
//	                               defaut, par tout le paquet. C est le lecteur qui parcourt un
//	                               octet bit a bit (`d[i] >> (7 - j) & 1`). Un decalage CONSTANT
//	                               (`uint16(d[i]) << 8`) ou d un MULTIPLE DE HUIT (`<< (8 * i)`)
//	                               assemble des octets, il n en extrait aucun bit : il ne compte
//	                               pas.
//
// Le mot de 64 bits charge puis decale (`binary.BigEndian.Uint64(d[i:]) << sh`) est deja tenu par
// le motif `ordre-d-octets`.
//
// ANGLE MORT CONNU, ECRIT POUR NE PAS ETRE PRESENTE COMME FERME : un octet copie d abord dans une
// variable (`c := d[i]`) puis decale (`c >> (7 - j)`) n est plus une indexation au moment du
// decalage ; sans type-checker, le motif ne le suit pas. Leur nombre n a pas ete mesure a la pose.
//
// # LES SEPT SONT PARTIS ; CE QUE LE MOTIF A TROUVE EN PLUS EST DATE
//
// DU-3 fixait la condition d avance : si le passage des sept lecteurs a la source coutait plus
// de 5 % au balayage, ils seraient restes, inscrits ici comme exceptions DATEES. La mesure A/B du
// 2026-09-26 (banc `grammar.BenchmarkBalayageBitABit`, 17 paires alternees en priorite haute :
// ecart median apparie -0,9 %, dispersion 1,3 point) a tenu la condition : AUCUN des sept n est
// une exception.
//
// LE MOTIF A TROUVE PLUS QUE LES SEPT. A sa pose (2026-09-26) il a rougi sur des fonctions que
// l inventaire du plan (`PLAN_SUITE_AUDIT_DECODEUR_FILM` §5 J4, « Pieces mesurees ») ne comptait
// pas — des lecteurs artisanaux sous des noms que personne n avait listes, exactement ce que la
// regle par la forme devait attraper. Les porter n est PAS le lot J4.6 (perimetre ferme : les
// sept) ; l un d eux vit dans `sync/killcollector`, qui ne peut pas importer la couche `source`
// (paquet `internal` du decodeur) et demande un deplacement derriere la facade. Ils sont donc
// inscrits dans [exceptionsDExtraction], UNE LIGNE PAR FONCTION (une fonction neuve dans le meme
// fichier rougit quand meme), chacune avec sa date, sa raison et son CRITERE DE RETRAIT ; une
// exception qui ne correspond plus a aucune extraction rougit
// (`TestExceptionsDExtractionSontDateesEtVivantes`). C est une DECISION ecrite, pas le retour de
// l allowlist d avant.
//
// # MUTATIONS JOUEES (2026-09-26), ROUGES, PUIS RETIREES
//
//   - `invBitAt` reintroduit tel qu il etait (`grammar/inventory_decode.go`) : rouge par
//     `lecteur-de-bits` (son nom est desormais liste) ET par `extraction-de-bits`. La premiere
//     version du motif ne le voyait PAS par la forme (`buf` porte plusieurs types dans la
//     grammaire, l indice passait par `idx`) : c est ce rouge manque qui a amene la portee
//     locale ;
//   - un lecteur NEUF sous un nom que personne ne liste (`lireUnBitNeuf`, corps de `readBitsAt`)
//     dans `grammar/offline_biped.go` : rouge par `extraction-de-bits` seul — la regle ne depend
//     plus du nom ;
//   - une exception rendue perimee (ligne pour une fonction qui n extrait rien) : rouge par
//     `TestExceptionsDExtractionSontDateesEtVivantes`.

import (
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

// motifExtraction : la cle sous laquelle une extraction de bits est rapportee.
const motifExtraction = "extraction-de-bits"

// contexteDePaquetOctets : ce que le balayage sait d un paquet sans type-checker — les types
// declares par nom, et les noms de constantes.
type contexteDePaquetOctets struct {
	types  map[string]map[string]bool
	consts map[string]bool
}

// constantesDuPaquet rend les noms declares `const` dans le paquet.
func constantesDuPaquet(asts map[string]*ast.File) map[string]bool {
	out := map[string]bool{}
	for _, f := range asts {
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.CONST {
				continue
			}
			for _, s := range g.Specs {
				for _, n := range s.(*ast.ValueSpec).Names {
					out[n.Name] = true
				}
			}
		}
	}
	return out
}

// extractionsDeBits rend, fonction par fonction, les extractions de bits d un fichier.
func extractionsDeBits(rel string, f *ast.File, ctx contexteDePaquetOctets, fset *token.FileSet) []siteBrut {
	var sites []siteBrut
	for _, d := range f.Decls {
		nom := "(declaration de paquet)"
		if fd, ok := d.(*ast.FuncDecl); ok {
			nom = "func " + fd.Name.Name
		}
		p := porteeDeLaDeclaration(d, ctx.consts)
		ast.Inspect(d, func(n ast.Node) bool {
			if expr, ok := extractionDuNoeud(n, ctx, p); ok {
				sites = append(sites, siteBrut{fichier: rel, motif: motifExtraction,
					detail: nom + " : " + expr, ligne: fset.Position(n.Pos()).Line, fonction: nom})
			}
			return true
		})
	}
	return sites
}

// porteeLocale : ce qu une declaration dit d elle-meme — les noms qu elle declare `[]byte`
// (parametres, resultats, variables), et ceux qu elle AFFECTE d une position en bits divisee par
// huit (`idx := p >> 3`, `byteIdx := (bitPos + i) / 8`). Sans elle, un lecteur recopie sous un
// nom neuf echappait au motif : `buf` porte plusieurs types dans la grammaire, et l indice de
// l octet passait par une variable (mutation du 2026-09-26, `invBitAt` sous un autre nom).
type porteeLocale struct {
	octets, indicesDOctet map[string]bool
}

// porteeDeLaDeclaration rend la portee locale d une declaration.
func porteeDeLaDeclaration(d ast.Decl, consts map[string]bool) porteeLocale {
	p := porteeLocale{octets: map[string]bool{}, indicesDOctet: map[string]bool{}}
	noter := func(noms []*ast.Ident, valeurs []ast.Expr) {
		for i, n := range noms {
			if i < len(valeurs) && estPositionEnBitsVersOctet(valeurs[i], consts) {
				p.indicesDOctet[n.Name] = true
			}
		}
	}
	ast.Inspect(d, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.Field:
			if renduDuType(v.Type) == "[]byte" {
				for _, nom := range v.Names {
					p.octets[nom.Name] = true
				}
			}
		case *ast.ValueSpec:
			if v.Type != nil && renduDuType(v.Type) == "[]byte" {
				for _, nom := range v.Names {
					p.octets[nom.Name] = true
				}
			}
			noter(v.Names, v.Values)
		case *ast.AssignStmt:
			var noms []*ast.Ident
			for _, g := range v.Lhs {
				id, _ := g.(*ast.Ident)
				if id == nil {
					id = &ast.Ident{}
				}
				noms = append(noms, id)
			}
			noter(noms, v.Rhs)
		}
		return true
	})
	return p
}

// extractionDuNoeud reconnait les deux formes (cf. l en-tete) sur un noeud.
func extractionDuNoeud(n ast.Node, ctx contexteDePaquetOctets, p porteeLocale) (string, bool) {
	switch v := n.(type) {
	case *ast.IndexExpr:
		id, _ := sansParenthesesNiConversion(v.Index).(*ast.Ident)
		if estPositionEnBitsVersOctet(v.Index, ctx.consts) || (id != nil && p.indicesDOctet[id.Name]) {
			return nomDeLaFonctionAppelee(v.X) + "[position en bits -> octet]", true
		}
	case *ast.BinaryExpr:
		if v.Op != token.SHR && v.Op != token.SHL {
			return "", false
		}
		idx, ok := sansParenthesesNiConversion(v.X).(*ast.IndexExpr)
		if !ok || estConstanteSyntaxique(v.Y, ctx.consts) || estMultipleDeHuit(v.Y) {
			return "", false
		}
		nom := nomDeLaFonctionAppelee(idx.X)
		if nom != "" && (p.octets[nom] || estTrancheDOctetsPartout(ctx.types, nom)) {
			return nom + "[...] " + v.Op.String() + " (quantite variable)", true
		}
	}
	return "", false
}

// estPositionEnBitsVersOctet : `p >> 3` ou `p / 8`, `p` non constant — l adresse de l octet qui
// porte le bit `p`.
func estPositionEnBitsVersOctet(x ast.Expr, consts map[string]bool) bool {
	b, ok := sansParenthesesNiConversion(x).(*ast.BinaryExpr)
	if !ok {
		return false
	}
	lit, ok := b.Y.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return false
	}
	if estConstanteSyntaxique(b.X, consts) {
		return false
	}
	return (b.Op == token.SHR && lit.Value == "3") || (b.Op == token.QUO && lit.Value == "8")
}

// sansParenthesesNiConversion retire les parentheses et les conversions vers un type entier
// (`uint64(x)`, `int(x)`) : elles ne changent pas ce qu on lit.
func sansParenthesesNiConversion(x ast.Expr) ast.Expr {
	for {
		switch v := x.(type) {
		case *ast.ParenExpr:
			x = v.X
		case *ast.CallExpr:
			id, ok := v.Fun.(*ast.Ident)
			if !ok || len(v.Args) != 1 || !estTypeEntier(id.Name) {
				return x
			}
			x = v.Args[0]
		default:
			return x
		}
	}
}

// estTypeEntier : les noms des types entiers predeclares.
func estTypeEntier(nom string) bool {
	return entiersDePosition[nom] || strings.HasPrefix(nom, "uint") || strings.HasPrefix(nom, "int") ||
		nom == "byte"
}

// estConstanteSyntaxique : un litteral, une constante du paquet, ou une expression qui n en
// combine que (`8 * 2`, `tailleOctet`).
func estConstanteSyntaxique(x ast.Expr, consts map[string]bool) bool {
	switch v := sansParenthesesNiConversion(x).(type) {
	case *ast.BasicLit:
		return true
	case *ast.Ident:
		return consts[v.Name]
	case *ast.BinaryExpr:
		return estConstanteSyntaxique(v.X, consts) && estConstanteSyntaxique(v.Y, consts)
	case *ast.UnaryExpr:
		return estConstanteSyntaxique(v.X, consts)
	default:
		return false
	}
}

// exceptionDExtractionDatee : UNE fonction qui extrait des bits hors de la source, toleree par
// DECISION ecrite (cf. l en-tete), avec son critere de retrait.
type exceptionDExtractionDatee struct {
	fichier, fonction string // chemin relatif a apps/go-api ; `func Nom` tel que le motif le rapporte
	date, raison      string
	retrait           string // le fait mesurable qui fait retirer la ligne
}

// retraitParPortage : le critere commun aux lignes ci-dessous.
const retraitParPortage = "la fonction n extrait plus de bit elle-meme (portee sur une " +
	"convention nommee de `film/internal/source`, ou descendue dans le decodeur derriere la " +
	"facade) : le motif cesse de la voir et la ligne devient perimee, donc ROUGE — la retirer " +
	"dans le commit du portage."

// exceptionsDExtraction : les extractions de bits trouvees par le motif a sa pose (lot J4.6,
// 2026-09-26) HORS des sept lecteurs du plan. Aucune n est l un des sept.
var exceptionsDExtraction = []exceptionDExtractionDatee{
	{fichier: "internal/games/halo_infinite/film/internal/grammar/frame_vue_controle.go",
		fonction: "func vueCFermee", date: "2026-09-26", retrait: retraitParPortage,
		raison: "oracle de cadrage de la vue C (carte de fermeture J4.0, instrument sans sortie de " +
			"production) : teste que le reste du paquet est nul, bit par bit (`pay[i/8]`). Hors de " +
			"l inventaire des sept ; une position de depart negative y paniquerait la ou " +
			"`source.BitAt` rendrait 0, donc le portage n est pas neutre sans preuve."},
	{fichier: "internal/games/halo_infinite/film/internal/grammar/weaponscan/scanner.go",
		fonction: "func matchMarkerAt", date: "2026-09-26", retrait: retraitParPortage,
		raison: "marqueur universel de 11 bits du scanner d armes, bit par bit, SORTIE ANTICIPEE au " +
			"premier bit discordant : hors du tampon il rend faux au lieu de paniquer quand la " +
			"discordance precede le debordement — aucune convention de la source n a cette forme."},
	{fichier: "internal/games/halo_infinite/film/internal/grammar/weaponscan/scanner.go",
		fonction: "func readBitsUint64", date: "2026-09-26", retrait: retraitParPortage,
		raison: "lecteur du scanner d armes, indexation nue (convention `source.BitsStricts`), hors " +
			"de l inventaire des sept."},
	{fichier: "internal/games/halo_infinite/film/internal/grammar/weaponscan/scanner.go",
		fonction: "func readBitsUint8", date: "2026-09-26", retrait: retraitParPortage,
		raison: "lecteur du scanner d armes, indexation nue (convention `source.BitsStricts`, " +
			"accumulateur de 8 bits), hors de l inventaire des sept."},
	{fichier: "internal/games/halo_infinite/film/research/cmd_rdata_weapon_scan/main.go",
		fonction: "func bitsAt", date: "2026-09-26", retrait: retraitParPortage,
		raison: "outil de recherche (tag `research`) descendu sous `film/research` au lot J4.5 avec " +
			"sa copie de lecteur (convention `source.BitsBourres`) ; son nom `bitsAt` n est pas " +
			"dans la liste des noms, seule la forme l a trouve."},
	{fichier: "internal/games/halo_infinite/film/internal/facts/killsource/botmeta.go",
		fonction: "func byteAtBit", date: "2026-09-26", retrait: retraitParPortage,
		raison: "octet a une position de bit quelconque, zero hors bornes, dans la couche des faits " +
			"`killsource` ; `source.OctetAuBit` porte la meme lecture sous une autre convention de " +
			"bord — le portage demande la preuve d equivalence de `killsource` (revision " +
			"`killsource.Rev`), hors du perimetre de J4.6."},
	{fichier: "cmd/diag_film/main.go",
		fonction: "func countMarkerBits", date: "2026-09-26", retrait: retraitParPortage,
		raison: "CLI de diagnostic : compte un marqueur par fenetre de 32 bits glissee bit a bit " +
			"(`bitPos / 8`). Hors de l inventaire des sept ; `cmd/` n importe pas la couche " +
			"`source` (paquet `internal` du decodeur)."},
	{fichier: "internal/sync/killcollector/shots.go",
		fonction: "func chercherDansChunk", date: "2026-09-26", retrait: retraitParPortage,
		raison: "fenetre glissante de 64 bits du collecteur de tirs (production). `sync` ne peut pas " +
			"importer `film/internal/source` : le portage est un deplacement derriere la facade " +
			"`decfilm`, hors du perimetre de J4.6. ECART A L ADR 0034 D-2 (« personne hors de " +
			"`source` ne lit un bit »), consigne au rapport du lot."},
	{fichier: "internal/sync/killcollector/shots.go",
		fonction: "func lireIndiceAvant", date: "2026-09-26", retrait: retraitParPortage,
		raison: "les 5 bits qui precedent le motif xuid (meme fichier, meme raison que " +
			"`chercherDansChunk`)."},
}

// exceptionDExtraction rend l exception qui couvre un site du motif `extraction-de-bits`, nil
// sinon. Les autres motifs n ont AUCUNE exception.
func exceptionDExtraction(s siteBrut) *exceptionDExtractionDatee {
	if s.motif != motifExtraction {
		return nil
	}
	for i := range exceptionsDExtraction {
		if e := &exceptionsDExtraction[i]; e.fichier == s.fichier && e.fonction == s.fonction {
			return e
		}
	}
	return nil
}

// TestExceptionsDExtractionSontDateesEtVivantes : chaque exception porte sa date, sa raison et
// son critere de retrait, et couvre une extraction REELLE — une ligne perimee rougit.
func TestExceptionsDExtractionSontDateesEtVivantes(t *testing.T) {
	sites, _ := balayerLecturesBrutes(t)
	vivantes := map[string]bool{}
	for _, s := range sites {
		if e := exceptionDExtraction(s); e != nil {
			vivantes[e.fichier+" | "+e.fonction] = true
		}
	}
	for _, e := range exceptionsDExtraction {
		if strings.TrimSpace(e.date) == "" || strings.TrimSpace(e.raison) == "" ||
			strings.TrimSpace(e.retrait) == "" {
			t.Errorf("exception %s | %s : date, raison ET critere de retrait sont obligatoires",
				e.fichier, e.fonction)
		}
		if !vivantes[e.fichier+" | "+e.fonction] {
			t.Errorf("exception PERIMEE %s | %s : plus aucune extraction de bits a cet endroit. "+
				"Critere de retrait atteint — retirer la ligne.", e.fichier, e.fonction)
		}
	}
}

// estMultipleDeHuit : une quantite de decalage de la forme `8 * x`, `x * 8` ou `x << 3`. Decaler
// un octet d un multiple de huit le PLACE dans un entier (assemblage petit- ou gros-boutiste,
// `replay/filmfacts_flux.go`) ; il n en extrait aucun bit. Faux positif mesure le 2026-09-26.
func estMultipleDeHuit(x ast.Expr) bool {
	b, ok := sansParenthesesNiConversion(x).(*ast.BinaryExpr)
	if !ok {
		return false
	}
	huit := func(e ast.Expr) bool {
		lit, ok := sansParenthesesNiConversion(e).(*ast.BasicLit)
		return ok && lit.Kind == token.INT && lit.Value == "8"
	}
	switch b.Op {
	case token.MUL:
		return huit(b.X) || huit(b.Y)
	case token.SHL:
		lit, ok := sansParenthesesNiConversion(b.Y).(*ast.BasicLit)
		return ok && lit.Kind == token.INT && lit.Value == "3"
	}
	return false
}
