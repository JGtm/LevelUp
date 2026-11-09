package hinavmesh

// traversee_test.go — CE QUE CONTIENT LA REGION 3 (`hkaiTraversalAnnotationLibrary`),
// et surtout CE QU'ELLE NE CONTIENT PAS.
//
// Cette region a ete ouverte pour repondre a une question precise : le `navmesh.blob`
// d'une carte Forge porte-t-il les NOMS DE LIEUX (zones de callout) que le jeu affiche ?
// Une « bibliotheque d'annotations de traversee » etait le dernier endroit plausible.
//
// REPONSE : NON. La region 3 est une table de LIENS DE SAUT — les endroits ou un agent
// quitte une face du maillage pour en rejoindre une autre sans passer par une arete
// partagee (franchir un vide, descendre d'une plateforme, grimper). Aucune chaine, aucun
// StringId : la table des types du fichier, qui est sa propre reflexion, ne declare AUCUN
// type de chaine. Ces temoins figent le constat pour qu'on ne rouvre pas la question, et
// pour que le jour ou une carte livrerait autre chose, cela se VOIE.

import (
	"encoding/binary"
	"strings"
	"testing"
)

const classeBibliothequeTraversee = "hkaiTraversalAnnotationLibrary"

// Disposition de `hkaiUserEdgeUtils::UserEdgePair`, LUE dans la table des types : les
// trois hkVector4 `x`, `y` et `z` ne sont pas trois points mais trois COLONNES — `x`
// porte les quatre abscisses, `y` les quatre ordonnees, `z` les quatre cotes. Un lien
// decrit donc QUATRE points : le segment (0,1) sur `faceA`, le segment (2,3) sur `faceB`.
const (
	membrePaires    = "userEdgePairs"
	membreAnnots    = "annotations"
	membreFaceA     = "faceA"
	membreFaceB     = "faceB"
	membreDonneesA  = "userDataA"
	membreDonneesB  = "userDataB"
	lanesParPaire   = 4
	decalageOrdonne = 16
	decalageCote    = 32
	// margeEmprise : un lien s'accroche au BORD d'une face, la tolerance couvre l'epaisseur
	// du bord sans laisser passer un point manifestement etranger au maillage.
	margeEmprise = 0.5
)

// regionsTemoin decoupe le `navmesh.blob` verse en testdata.
func regionsTemoin(t *testing.T, asset string) [][]byte {
	t.Helper()
	charge, err := decompresse(chargeBlobTemoin(t, asset))
	if err != nil {
		t.Fatalf("decompresse: %v", err)
	}
	d, err := regions(charge)
	if err != nil {
		t.Fatalf("regions: %v", err)
	}
	return d
}

// regionTraversee rend le fichier-tag dont la racine est la bibliotheque d'annotations.
func regionTraversee(t *testing.T, asset string) *fichierTag {
	t.Helper()
	for _, r := range regionsTemoin(t, asset) {
		if len(r) < 8 || string(r[4:8]) != "TAG0" {
			continue
		}
		f, err := lireFichierTag(r)
		if err != nil {
			continue
		}
		rac, err := f.racine()
		if err != nil {
			continue
		}
		if f.nomType(rac.Type) == classeBibliothequeTraversee {
			return f
		}
	}
	t.Fatalf("aucune region ne porte un %s", classeBibliothequeTraversee)
	return nil
}

// TestTraverseeEstUneTableDeSauts prouve la NATURE de la region 3 par sa geometrie : tous
// les points des liens tombent dans l'emprise du maillage de navigation de la meme carte,
// et tous les indices de face y designent une face existante. Une table de noms de lieux
// n'aurait ni l'un ni l'autre.
func TestTraverseeEstUneTableDeSauts(t *testing.T) {
	for _, cas := range []struct {
		nom, asset  string
		liens, face int
	}{
		{"Isolation", "01af558d-53ab-4f05-ba68-92d805fc6260", 190, 2348},
		{"Kikenna", "df7dbf08-b8de-4ade-9d7f-1947128c9ae4", 36, 689},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			f := regionTraversee(t, cas.asset)
			rac, err := f.racine()
			if err != nil {
				t.Fatalf("racine: %v", err)
			}
			paires, err := f.tableau(rac, membrePaires)
			if err != nil {
				t.Fatalf("%s: %v", membrePaires, err)
			}
			annots, err := f.tableau(rac, membreAnnots)
			if err != nil {
				t.Fatalf("%s: %v", membreAnnots, err)
			}
			// Les deux tableaux sont indexes ENSEMBLE : une annotation par lien.
			if paires.Compte != cas.liens || annots.Compte != cas.liens {
				t.Fatalf("%d liens et %d annotations, %d attendus de chaque",
					paires.Compte, annots.Compte, cas.liens)
			}
			m := decodeTemoin(t, cas.asset)
			if len(m.Faces) != cas.face {
				t.Fatalf("%d faces au maillage, %d attendues", len(m.Faces), cas.face)
			}
			verifieLiens(t, f, paires, m)
		})
	}
}

// verifieLiens confronte chaque lien au maillage : indices de face valides, points dans
// l'emprise.
func verifieLiens(t *testing.T, f *fichierTag, paires itemHavok, m *Maillage) {
	t.Helper()
	pas := f.types.taille(paires.Type)
	brut, err := f.octets(paires, pas)
	if err != nil {
		t.Fatalf("octets: %v", err)
	}
	champA, err := f.champEntier(paires.Type, membreFaceA, pas)
	if err != nil {
		t.Fatalf("%s: %v", membreFaceA, err)
	}
	champB, err := f.champEntier(paires.Type, membreFaceB, pas)
	if err != nil {
		t.Fatalf("%s: %v", membreFaceB, err)
	}
	for i := 0; i < paires.Compte; i++ {
		o := i * pas
		for _, fi := range []int64{champA.lit(brut, o), champB.lit(brut, o)} {
			if fi < 0 || int(fi) >= len(m.Faces) {
				t.Fatalf("lien %d: indice de face %d hors des %d faces", i, fi, len(m.Faces))
			}
		}
		for lane := 0; lane < lanesParPaire; lane++ {
			x := flottant(brut, o+4*lane)
			y := flottant(brut, o+decalageOrdonne+4*lane)
			z := flottant(brut, o+decalageCote+4*lane)
			if x < m.Min.X-margeEmprise || x > m.Max.X+margeEmprise ||
				y < m.Min.Y-margeEmprise || y > m.Max.Y+margeEmprise ||
				z < m.Min.Z-margeEmprise || z > m.Max.Z+margeEmprise {
				t.Fatalf("lien %d, point %d (%.2f, %.2f, %.2f) hors de l'emprise du maillage",
					i, lane, x, y, z)
			}
		}
	}
}

// TestNavmeshNePorteAucuneChaine est LE temoin du chantier des callouts : aucune des cinq
// regions d'un `navmesh.blob` ne porte de nom de lieu. Le controle est double, et chaque
// moitie couvre l'angle mort de l'autre :
//
//  1. par la REFLEXION, pour les quatre regions qui sont des fichiers-tag. Un fichier-tag
//     Havok est INTEGRALEMENT auto-descriptif : rien ne peut vivre dans DATA sans qu'un
//     type le declare. Si la table des types ne declare aucun type de chaine, alors il n'y
//     a aucune chaine dans la region — c'est une preuve, pas un sondage.
//  2. par le CONTENU, pour la cinquieme region, qui n'est PAS un fichier-tag et n'a donc
//     pas de reflexion a interroger : on y cherche des chaines terminees par un nul.
//
// Le critere de contenu exige des DELIMITEURS NULS des deux cotes, et ce n'est pas du
// zele : un balayage ASCII naif remonte des dizaines de fausses pistes ("ttttt"), parce
// que l'octet 0x74 = 't' est frequent dans les mantisses de flottants. Une vraie chaine C
// est nul-terminee ; le critere en trouve zero sur les deux cartes, la ou le balayage naif
// en trouvait des dizaines.
//
// Si un jour une carte livre un nom de lieu dans son navmesh, ce test tombe — et c'est
// exactement le signal qu'on veut.
func TestNavmeshNePorteAucuneChaine(t *testing.T) {
	for _, asset := range []string{
		"01af558d-53ab-4f05-ba68-92d805fc6260",
		"df7dbf08-b8de-4ade-9d7f-1947128c9ae4",
	} {
		for i, r := range regionsTemoin(t, asset) {
			zone, tagfile := zoneDeDonnees(t, r, i+1, asset)
			if tagfile {
				verifieAucunTypeDeChaine(t, r, i+1, asset)
			}
			for _, s := range chainesNulTerminees(zone) {
				t.Errorf("%s, region %d: chaine inattendue a l'offset %d de la zone de donnees: %q",
					asset, i+1, s.offset, s.texte)
			}
		}
	}
}

// zoneDeDonnees rend la partie de la region ou une chaine serait une DONNEE : la section
// DATA d'un fichier-tag (les noms Havok, eux, vivent dans TST1 et FST1), ou la region
// entiere si ce n'en est pas un.
func zoneDeDonnees(t *testing.T, region []byte, rang int, asset string) ([]byte, bool) {
	t.Helper()
	if len(region) < 8 || string(region[4:8]) != "TAG0" {
		return region, false
	}
	sections := map[string][2]int{}
	if err := parcoursSections(region, 0, len(region), sections, 0); err != nil {
		t.Fatalf("%s, region %d: sections: %v", asset, rang, err)
	}
	s, ok := sections["DATA"]
	if !ok {
		t.Fatalf("%s, region %d: fichier-tag sans section DATA", asset, rang)
	}
	return region[s[0] : s[0]+s[1]], true
}

// verifieAucunTypeDeChaine interroge la reflexion du fichier-tag. `unsigned char` et
// `char` NUS sont des primitives entieres (le `hkUint8` d'un enum en derive) et ne sont
// donc pas des chaines : seuls les types de chaine de Havok comptent.
func verifieAucunTypeDeChaine(t *testing.T, region []byte, rang int, asset string) {
	t.Helper()
	f, err := lireFichierTag(region)
	if err != nil {
		t.Fatalf("%s, region %d: %v", asset, rang, err)
	}
	for _, ty := range f.types {
		suspect := strings.Contains(ty.Nom, "String") ||
			strings.Contains(ty.Nom, "char*") ||
			strings.HasSuffix(ty.Nom, "Name") ||
			strings.HasSuffix(ty.Nom, "Id")
		if suspect {
			t.Errorf("%s, region %d: la table des types declare %q — un porteur de nom possible",
				asset, rang, ty.Nom)
		}
	}
}

type suiteASCII struct {
	offset int
	texte  string
}

// chainesNulTerminees rend les suites imprimables d'au moins 5 caracteres encadrees par
// des octets nuls — la forme d'une chaine C serialisee.
func chainesNulTerminees(buf []byte) []suiteASCII {
	const longueurMin = 5
	var out []suiteASCII
	debut := -1
	for p := 0; p <= len(buf); p++ {
		if p < len(buf) && estImprimable(buf[p]) {
			if debut < 0 {
				debut = p
			}
			continue
		}
		encadree := debut > 0 && p < len(buf) && buf[debut-1] == 0 && buf[p] == 0
		if debut >= 0 && p-debut >= longueurMin && encadree {
			out = append(out, suiteASCII{offset: debut, texte: string(buf[debut:p])})
		}
		debut = -1
	}
	return out
}

func estImprimable(b byte) bool {
	return b == '_' || b == ':' || b == ' ' || b == '-' ||
		(b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

// TestTraverseeNaPasDeVocabulaire acheve la demonstration cote CHAMPS. Les seuls entiers
// de la region 3 qu'on pourrait confondre avec un identifiant de nom sont les `userDataA`
// et `userDataB` des liens. Ils sont SEQUENTIELS et IDENTIQUES d'une carte a l'autre
// (base 0x7F900000, puis +1) : c'est un compteur de serialisation, pas un condensat de
// chaine — deux cartes aux lieux differents ne pourraient pas porter la meme suite.
func TestTraverseeNaPasDeVocabulaire(t *testing.T) {
	const baseSequentielle = 0x7F900000
	const echantillon = 4
	empreintes := map[string][]int32{}
	for _, asset := range []string{
		"01af558d-53ab-4f05-ba68-92d805fc6260",
		"df7dbf08-b8de-4ade-9d7f-1947128c9ae4",
	} {
		vus := donneesUtilisateur(t, asset, echantillon)
		if len(vus) < echantillon {
			t.Fatalf("%s: %d valeurs de userData non nulles, %d attendues au moins",
				asset, len(vus), echantillon)
		}
		if vus[0] < baseSequentielle {
			t.Errorf("%s: premier userData 0x%08x sous la base sequentielle 0x%08x",
				asset, vus[0], baseSequentielle)
		}
		empreintes[asset] = vus
	}
	a := empreintes["01af558d-53ab-4f05-ba68-92d805fc6260"]
	b := empreintes["df7dbf08-b8de-4ade-9d7f-1947128c9ae4"]
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("userData %d differe entre les deux cartes (0x%08x vs 0x%08x) : "+
				"l'hypothese du compteur de serialisation est a revoir", i, a[i], b[i])
		}
	}
}

// donneesUtilisateur rend les premieres valeurs non nulles des tableaux userDataA/B.
func donneesUtilisateur(t *testing.T, asset string, max int) []int32 {
	t.Helper()
	f := regionTraversee(t, asset)
	rac, err := f.racine()
	if err != nil {
		t.Fatalf("racine: %v", err)
	}
	paires, err := f.tableau(rac, membrePaires)
	if err != nil {
		t.Fatalf("%s: %v", membrePaires, err)
	}
	pas := f.types.taille(paires.Type)
	brut, err := f.octets(paires, pas)
	if err != nil {
		t.Fatalf("octets: %v", err)
	}
	var vus []int32
	for i := 0; i < paires.Compte && len(vus) < max; i++ {
		for _, nom := range []string{membreDonneesA, membreDonneesB} {
			m, ok := f.types.membre(paires.Type, nom)
			if !ok {
				t.Fatalf("%s ne declare pas %q", f.nomType(paires.Type), nom)
			}
			// Un hkInplaceArray serialise pointe, comme un hkArray, une entree ITEM.
			idx := int(binary.LittleEndian.Uint64(brut[i*pas+m.Offset:]))
			if idx <= 0 || idx >= len(f.items) {
				continue
			}
			octets, err := f.octets(f.items[idx], 4)
			if err != nil {
				t.Fatalf("userData du lien %d: %v", i, err)
			}
			for k := 0; k*4 < len(octets); k++ {
				if v := int32(binary.LittleEndian.Uint32(octets[4*k:])); v != 0 {
					vus = append(vus, v)
				}
			}
		}
	}
	return vus
}
