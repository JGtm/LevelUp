package replay

// equivalence_minifilm_test.go — L'ETAGE CI DE L'EQUIVALENCE (PLAN_CUISSON_PERF §3 D4c).
//
// # CE QU'IL COUVRE, ET CE QU'IL NE PEUT PAS COUVRIR
//
// Le harnais local `cmd/replay-equiv` prouve l'equivalence sur onze VRAIS films — il exige le
// cache film (23 Go) et ne tourne donc jamais en CI. Ce test-ci est sa doublure permanente : il
// hache les balayages que la MINI-BOBINE supporte, et eux seuls.
//
// La liste est FERMEE, et elle est courte pour une raison mesurable : la mini-bobine n'a NI
// chunk_00 (registre), NI manifeste, et ses paquets sont concatenes hors de leur continuite —
// les POSITIONS de biped y sont donc sans signification, et `BuildFromFilm` la refuse
// (`world_object_precision_guard_test.go`, `PROVENANCE.txt`). Autrement dit : LA CI NE COUVRE
// NI LE REGISTRE NI LES POSITIONS. C'est le corpus local qui les couvre, et c'est ecrit.
//
// # REGENERATION
//
//	go test ./internal/analysis/replay/ -run TestEquivalenceMiniFilm -update
//
// Un digest qui bouge dans un lot de REFACTO PUR fait echouer le lot — il ne se regenere qu'aux
// lots de correction declares (3 et 4b), avec le diff des comptes au journal.
//
// Le fichier s'ouvre par sa VERSION DE GRAMMAIRE (`# digest-grammar: N`, cf.
// digest.GrammarVersion) : un changement du RENDU se signale alors comme tel, au lieu de se
// lire comme une regression du decodeur.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/digest"
	"levelup/go-api/internal/analysis/filmdec"
)

// miniFilmDigestsPath est le fichier de digests figes de la mini-bobine, a cote de ceux du
// corpus local : une seule adresse pour toutes les references d'equivalence.
func miniFilmDigestsPath() string {
	return filepath.Join(goldenDir, "equivalence", "minifilm.tsv")
}

func TestEquivalenceMiniFilm(t *testing.T) {
	lignes, err := digestsMiniBobine()
	if err != nil {
		t.Fatalf("balayages de la mini-bobine : %v", err)
	}
	path := miniFilmDigestsPath()
	if *updateGolden {
		// LA LIGNE DE GRAMMAIRE OUVRE LE FICHIER (cf. verifierGrammaireFigee).
		contenu := strings.Join(append([]string{digest.GrammarLine()}, lignes...), "\n") + "\n"
		if err := os.WriteFile(path, []byte(contenu), 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", path, err)
		}
		t.Logf("digests de la mini-bobine figes : %s (%d etapes, grammaire %d)",
			path, len(lignes), digest.GrammarVersion)
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("digests de reference illisibles (%s) : %v — les figer avec -update", path, err)
	}
	comparerDigests(t, verifierGrammaireFigee(t, lignesDe(string(raw)), path), lignes)
}

// verifierGrammaireFigee exige que le fichier fige porte la ligne de grammaire de la version
// COURANTE, et rend ses lignes de digest.
//
// SANS CE MARQUEUR, un changement du RENDU de `digest` se lit comme une regression du
// decodeur — les empreintes figees sont alors celles d'un AUTRE rendu de la MEME valeur, et
// aucune relecture du decodeur ne l'expliquera. Ce n'est pas une hypothese : le 2026-09-02, six
// des neuf fichiers du corpus local etaient restes sous la grammaire v1 sans que rien ne le
// dise. Le verdict est donc explicite : « regenerer », jamais « ECART ».
func verifierGrammaireFigee(t *testing.T, lignes []string, path string) []string {
	t.Helper()
	if len(lignes) == 0 {
		t.Fatalf("%s est vide — le figer avec -update", path)
	}
	version, ok := digest.ParseGrammarLine(lignes[0])
	if !ok {
		t.Fatalf("%s : premiere ligne %q, attendue %q — ce fichier a ete fige AVANT le marqueur "+
			"de grammaire : le regenerer avec -update", path, lignes[0], digest.GrammarLine())
	}
	if version != digest.GrammarVersion {
		t.Fatalf("%s : digests figes sous la grammaire %d, paquet digest en %d — ce n'est PAS une "+
			"regression du decodeur mais un changement du RENDU : regenerer avec -update",
			path, version, digest.GrammarVersion)
	}
	return lignes[1:]
}

// comparerDigests nomme la PREMIERE etape qui differe — c'est ce qui transforme « quelque chose
// a change » en « le balayage des projectiles a change ».
func comparerDigests(t *testing.T, attendu, obtenu []string) {
	t.Helper()
	for i := range max(len(attendu), len(obtenu)) {
		switch {
		case i >= len(obtenu):
			t.Fatalf("etape %q de la reference NON produite", attendu[i])
		case i >= len(attendu):
			t.Fatalf("etape produite en TROP (absente de la reference) : %q", obtenu[i])
		case attendu[i] != obtenu[i]:
			t.Fatalf("ECART a l'etape %d :\n  attendu %s\n  obtenu  %s\n"+
				"Un digest qui bouge dans un refacto pur fait ECHOUER le lot ; s'il s'agit d'une "+
				"correction declaree, regenerer avec -update et porter le diff au journal.",
				i+1, attendu[i], obtenu[i])
		}
	}
}

// lignesDe decoupe un fichier de digests en lignes, sans les vides de fin.
func lignesDe(contenu string) []string {
	lignes := strings.Split(strings.ReplaceAll(contenu, "\r\n", "\n"), "\n")
	for len(lignes) > 0 && strings.TrimSpace(lignes[len(lignes)-1]) == "" {
		lignes = lignes[:len(lignes)-1]
	}
	return lignes
}

// digestsMiniBobine rejoue les SEULS balayages que la mini-bobine supporte, dans un ordre fixe,
// et rend une ligne `etape\tcompte\tsha` par balayage.
func digestsMiniBobine() ([]string, error) {
	entry, err := goldenMapQuant()
	if err != nil {
		return nil, err
	}
	// MEME GESTE QUE LA PRODUCTION (cf. installWorldObjectPrecision) : les largeurs d'axe du
	// chemin world-object viennent de l'entree de catalogue. Sans ce reglage, le digest des
	// projectiles dependrait de l'etat laisse par le test precedent.
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	filmdec.SetWorldObjectPrecisionFromLayout(filmdec.I0Layout{AxisW: entry.AxisWidths})
	wr := entry.Range()
	dir := MiniFilmDir

	fire, err := filmdec.ScanFilmFireEvents(dir)
	if err != nil {
		return nil, fmt.Errorf("tirs : %w", err)
	}
	grenades, err := filmdec.ScanFilmGrenadeThrows(dir)
	if err != nil {
		return nil, fmt.Errorf("lancers de grenade : %w", err)
	}
	loadouts, err := filmdec.ScanFilmKeyframeLoadouts(dir, loadoutFamilies())
	if err != nil {
		return nil, fmt.Errorf("armes portees : %w", err)
	}
	inventory, _, err := ScanFilmKeyframeInventory(dir, loadoutFamilies(), 0)
	if err != nil {
		return nil, fmt.Errorf("inventaire d'image-cle : %w", err)
	}
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		return nil, fmt.Errorf("morts : %w", err)
	}
	indices, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
	if err != nil {
		return nil, fmt.Errorf("indices joueur : %w", err)
	}
	proj, err := filmdec.ScanFilmProjectiles(dir, &wr)
	if err != nil {
		return nil, fmt.Errorf("projectiles : %w", err)
	}
	return lignesDeDigest([]etapeMiniBobine{
		{etapeFire, fire},
		{etapeGrenades, grenades},
		{etapeLoadouts, loadouts},
		{etapeInventaire, inventory},
		{etapeMorts, deaths},
		{etapeIndicesJoueur, indices},
		{etapeProjectiles, proj},
	}), nil
}

// Noms des etapes de la mini-bobine. Ce sont, mot pour mot, ceux de BuildFromFilmSteps : le
// fichier de digests de la CI se lit avec la MEME grille que ceux du corpus local, et une
// etape renommee d'un cote sans l'autre se verrait a la comparaison.
//
// Ils sont nommes plutot qu'ecrits en toutes lettres ici parce que le paquet ne doit garder
// qu'une seule ecriture de chaque nom d'etape : celle de BuildFromFilmSteps, gardee par
// observe_test.go, qui exige que les litteraux `opt.observe("...")` de build.go restent des
// litteraux (goconst comptait sinon quatre "fire" dans le paquet).
const (
	etapeFire          = "fire"
	etapeGrenades      = "grenades"
	etapeLoadouts      = "loadouts"
	etapeInventaire    = "inventory"
	etapeMorts         = "deaths"
	etapeIndicesJoueur = "playerIndices"
	etapeProjectiles   = "projectiles"
)

// etapeMiniBobine : un balayage et sa sortie.
type etapeMiniBobine struct {
	nom string
	val any
}

// lignesDeDigest hache chaque etape.
func lignesDeDigest(etapes []etapeMiniBobine) []string {
	out := make([]string, 0, len(etapes))
	for _, e := range etapes {
		compte, sum := digest.Of(e.val)
		out = append(out, fmt.Sprintf("%s\t%d\t%s", e.nom, compte, sum))
	}
	return out
}
