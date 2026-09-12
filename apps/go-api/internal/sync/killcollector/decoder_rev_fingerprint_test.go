package killcollector

// decoder_rev_fingerprint_test.go — LE GARDE-RAIL QUI TRANSFORME UNE CONSIGNE EN GATE.
//
// # Le defaut qu il ferme
//
// [KillSourceDecoderRev] porte depuis sa creation le contrat « LA FAIRE EVOLUER a chaque
// changement de decodage ». Mesure du 2026-09-05 : 14 commits sur
// `internal/games/halo_infinite/film/killsource/` depuis v7.3.0, ZERO bump. Une consigne ecrite
// dans un commentaire ne se tient pas toute seule — les lignes deja ecrites portaient la
// revision courante et etaient donc exclues A VIE du backlog de redecodage.
//
// # Ce que le garde-rail fait, exactement
//
// Il hache les sources NON-TEST du paquet decodeur et compare AU GOLDEN
// `testdata/killsource_decoder_rev.golden`, qui porte le couple (revision, empreinte). Toucher le
// decodeur fait rougir ce test ; le remettre au vert demande de rouvrir la ligne de la revision —
// donc de DECIDER si les lignes en base doivent etre redecodees. C est tout ce qu on lui demande.
//
// # Le golden porte LES DEUX valeurs, et c est le correctif du 2026-09-12
//
// Revue adversariale, constat P1-4 : tant que le test ne comparait que l EMPREINTE a une constante
// de `collector.go`, remettre [KillSourceDecoderRev] a sa valeur d avant — en gardant la nouvelle
// empreinte — restait VERT. Le gate ne tenait donc qu un des deux gestes qu il exigeait dans son
// propre message. Les deux valeurs figees ENSEMBLE rendent les deux derives visibles, avec deux
// messages distincts : « le decodeur a change » et « la revision a change sans le decodeur ».
//
// # Ce qu il ne fait pas, et pourquoi c est assume
//
// Il ne distingue pas un changement de decodage d une reformulation de commentaire : le hachage
// porte sur les OCTETS. Un garde-rail qui tenterait de ne mordre que sur les changements
// « significatifs » devrait comprendre le decodeur — il rendrait des faux negatifs, c est-a-dire
// exactement le defaut d aujourd hui. Un faux positif, lui, coute une ligne a mettre a jour.
//
// # Les fins de ligne sont NORMALISEES avant le hachage
//
// Le depot force `eol=lf` (.gitattributes) mais un checkout mal configure, un editeur ou un
// outil Windows peut poser des CRLF. Sans normalisation, l empreinte serait verte sur la CI
// (Linux) et rouge sur le poste de developpement pour une raison qui n a rien a voir avec le
// decodage. On hache donc le texte a fins de ligne LF.

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// updateDecoderRevGolden : la regeneration du golden, EXPLICITE. Sans ce drapeau le test ne
// reecrit jamais rien — un gate qui se repare tout seul ne garde rien.
var updateDecoderRevGolden = flag.Bool("update", false,
	"reecrire testdata/killsource_decoder_rev.golden (revision ET empreinte)")

// cheminGoldenDecoderRev : le golden, relatif au paquet.
const cheminGoldenDecoderRev = "testdata/killsource_decoder_rev.golden"

// lireGoldenDecoderRev rend le couple (revision, empreinte) fige. Les lignes vides et les lignes
// de commentaire (`#`) sont ignorees : le fichier porte sa propre recette et son historique.
func lireGoldenDecoderRev(t *testing.T) (revision, empreinte string) {
	t.Helper()
	blob, err := os.ReadFile(cheminGoldenDecoderRev)
	if err != nil {
		t.Fatalf("golden %s illisible : %v — il est VERSIONNE, son absence est une erreur",
			cheminGoldenDecoderRev, err)
	}
	for _, ligne := range strings.Split(strings.ReplaceAll(string(blob), "\r\n", "\n"), "\n") {
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			continue
		}
		champs := strings.Split(ligne, "\t")
		if len(champs) != 2 || champs[0] == "" || champs[1] == "" {
			t.Fatalf("golden %s : ligne de donnees malformee %q — attendu "+
				"`revision<TAB>empreinte`", cheminGoldenDecoderRev, ligne)
		}
		if revision != "" {
			t.Fatalf("golden %s : plusieurs lignes de donnees — le fichier fige UN couple",
				cheminGoldenDecoderRev)
		}
		revision, empreinte = champs[0], champs[1]
	}
	if revision == "" {
		t.Fatalf("golden %s : aucune ligne de donnees", cheminGoldenDecoderRev)
	}
	return revision, empreinte
}

// ecrireGoldenDecoderRev reecrit la SEULE ligne de donnees, en gardant l en-tete et l historique.
func ecrireGoldenDecoderRev(t *testing.T, revision, empreinte string) {
	t.Helper()
	blob, err := os.ReadFile(cheminGoldenDecoderRev)
	if err != nil {
		t.Fatalf("golden %s illisible : %v", cheminGoldenDecoderRev, err)
	}
	var sortie []string
	for _, ligne := range strings.Split(strings.ReplaceAll(string(blob), "\r\n", "\n"), "\n") {
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			sortie = append(sortie, ligne)
		}
	}
	// La derniere entree est la ligne vide finale : la ligne de donnees se glisse avant elle.
	sortie = append(sortie[:len(sortie)-1], revision+"\t"+empreinte, "")
	if err := os.WriteFile(cheminGoldenDecoderRev, []byte(strings.Join(sortie, "\n")), 0o600); err != nil {
		t.Fatalf("ecriture du golden %s : %v", cheminGoldenDecoderRev, err)
	}
	t.Logf("golden regenere : %s / %s", revision, empreinte)
}

// cheminPaquetKillsource rend le chemin du paquet decodeur depuis CE fichier de test.
//
// Il est resolu par `runtime.Caller` et pas par un chemin relatif au repertoire courant : un
// test Go tourne depuis le dossier de son paquet, mais le chemin de `killsource` change avec
// l arborescence — le jour ou le paquet demenage (ADR 0012, R.1 du plan v2), ce test doit
// ECHOUER bruyamment plutot que hacher un dossier vide.
func cheminPaquetKillsource(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	// internal/sync/killcollector -> internal/sync -> internal -> apps/go-api
	racine := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(ici))))
	return filepath.Join(racine, "internal", "games", "halo_infinite", "film", "killsource")
}

// empreinteSourcesGo hache les sources Go NON-TEST d un arbre, dans un ordre stable.
//
// Le chemin RELATIF entre dans le hachage a cote du contenu : sans lui, renommer un fichier
// sans en changer une ligne laisserait l empreinte immobile — or un renommage de fichier de
// decodeur est exactement le genre de mouvement qu on veut voir passer par la revision.
// `testdata/` est ecarte : ce sont des fixtures, pas du decodage.
func empreinteSourcesGo(racine string) (string, int, error) {
	type fichier struct {
		rel     string
		contenu []byte
	}
	var lus []fichier
	err := filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, errMarche error) error {
		if errMarche != nil {
			return errMarche
		}
		if d.IsDir() {
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		nom := d.Name()
		if !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			return nil
		}
		blob, errLire := os.ReadFile(chemin) //nolint:gosec // chemin construit depuis la racine du module
		if errLire != nil {
			return errLire
		}
		rel, errRel := filepath.Rel(racine, chemin)
		if errRel != nil {
			return errRel
		}
		lus = append(lus, fichier{rel: filepath.ToSlash(rel), contenu: blob})
		return nil
	})
	if err != nil {
		return "", 0, err
	}
	if len(lus) == 0 {
		return "", 0, fmt.Errorf("aucune source Go non-test sous %s", racine)
	}
	sort.Slice(lus, func(i, j int) bool { return lus[i].rel < lus[j].rel })

	h := sha256.New()
	for _, f := range lus {
		texte := strings.ReplaceAll(string(f.contenu), "\r\n", "\n")
		// Le nom, la longueur normalisee, puis le texte : la longueur empeche deux decoupages
		// differents des memes octets de rendre la meme empreinte.
		fmt.Fprintf(h, "%s\n%d\n", f.rel, len(texte))
		h.Write([]byte(texte))
	}
	return hex.EncodeToString(h.Sum(nil)), len(lus), nil
}

// TestKillSourceDecoderRevSuitLeDecodeur — LE GATE. Il tient les DEUX gestes que son message
// exige, et il les distingue :
//
//	empreinte differente          le decodeur a change -> decider, bumper si les lignes bougent,
//	                              puis regenerer le golden ;
//	empreinte egale, revision non la revision a bouge sans le decodeur -> geste sans effet, ou
//	                              golden non regenere apres un bump legitime.
func TestKillSourceDecoderRevSuitLeDecodeur(t *testing.T) {
	dir := cheminPaquetKillsource(t)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("paquet decodeur introuvable (%s) : %v — le paquet a demenage ? "+
			"mettre a jour cheminPaquetKillsource", dir, err)
	}
	empreinte, n, err := empreinteSourcesGo(dir)
	if err != nil {
		t.Fatalf("empreinte des sources du decodeur : %v", err)
	}
	if *updateDecoderRevGolden {
		ecrireGoldenDecoderRev(t, KillSourceDecoderRev, empreinte)
		return
	}
	revisionGolden, empreinteGolden := lireGoldenDecoderRev(t)
	if empreinte != empreinteGolden {
		t.Fatalf(`LE DECODEUR A CHANGE.

  paquet    : internal/games/halo_infinite/film/killsource (%d fichiers non-test)
  attendue  : %s
  mesuree   : %s
  revision  : KillSourceDecoderRev = %q (golden : %q)

DEUX GESTES, ET LES DEUX SONT OBLIGATOIRES :

  1. BUMPER KillSourceDecoderRev dans collector.go (ex. "killsource-AAAA-MM-JJ") si le changement
     modifie les lignes produites — c est ce qui rend les matchs deja decodes a nouveau candidats
     au backlog (postsync.go, conditionBacklog). Si le changement ne touche PAS les lignes
     produites (commentaire, renommage interne), laisser la revision et l ecrire dans le commit :
     le choix doit etre explicite, pas implicite.
  2. REGENERER le golden, qui fige le couple (revision, empreinte) :

       go test ./internal/sync/killcollector/ -run TestKillSourceDecoderRevSuitLeDecodeur -update

Sans le geste 2 ce test reste rouge ; sans le geste 1 les lignes en base restent servies avec
l ancien decodage, sans compteur et sans reprise possible.`,
			n, empreinteGolden, empreinte, KillSourceDecoderRev, revisionGolden)
	}
	if KillSourceDecoderRev != revisionGolden {
		t.Fatalf(`LA REVISION A CHANGE SANS QUE LE DECODEUR BOUGE.

  revision  : KillSourceDecoderRev = %q
  golden    : %q
  empreinte : %s (inchangee)

Deux lectures possibles, et aucune ne se regle en laissant le test vert :

  - la revision a ete bumpee pour un changement qui vit AILLEURS que dans
    internal/games/halo_infinite/film/killsource/ (le parseur d events, filmdec, une source de
    chunk). C est legitime — le gate ne hache que le decodeur, pas son amont : regenerer le
    golden avec -update, et le dire dans le commit.
  - la revision a ete modifiee par megarde, ou remise a une valeur anterieure. La remettre.

  go test ./internal/sync/killcollector/ -run TestKillSourceDecoderRevSuitLeDecodeur -update`,
			KillSourceDecoderRev, revisionGolden, empreinte)
	}
}

// TestEmpreinteSourcesGoMord — LA PREUVE QUE LE GARDE-RAIL MORD.
//
// Meme doctrine que TestBareBulkUpdateDetection_Sanity (internal/sync) : un garde-rail qui ne
// detecte jamais rien est inutile. Les quatre cas verifient les quatre proprietes sur
// lesquelles repose le gate ci-dessus.
func TestEmpreinteSourcesGoMord(t *testing.T) {
	ecrire := func(dir, nom, contenu string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, nom)), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, nom), []byte(contenu), 0o600); err != nil {
			t.Fatalf("ecrire %s: %v", nom, err)
		}
	}
	empreinte := func(dir string) string {
		t.Helper()
		e, _, err := empreinteSourcesGo(dir)
		if err != nil {
			t.Fatalf("empreinte(%s): %v", dir, err)
		}
		return e
	}

	base := t.TempDir()
	ecrire(base, "decode.go", "package p\n\nconst Largeur = 5\n")
	ecrire(base, "scan.go", "package p\n\nfunc Scan() {}\n")
	ref := empreinte(base)

	// 1. UN OCTET DE DECODAGE CHANGE -> l empreinte bouge.
	mute := t.TempDir()
	ecrire(mute, "decode.go", "package p\n\nconst Largeur = 6\n")
	ecrire(mute, "scan.go", "package p\n\nfunc Scan() {}\n")
	if empreinte(mute) == ref {
		t.Error("empreinte immobile apres un changement de source : le garde-rail est aveugle")
	}

	// 2. UN FICHIER RENOMME -> l empreinte bouge (le chemin entre dans le hachage).
	renomme := t.TempDir()
	ecrire(renomme, "decode2.go", "package p\n\nconst Largeur = 5\n")
	ecrire(renomme, "scan.go", "package p\n\nfunc Scan() {}\n")
	if empreinte(renomme) == ref {
		t.Error("empreinte immobile apres un renommage : le chemin n entre pas dans le hachage")
	}

	// 3. UN TEST OU UNE FIXTURE AJOUTE -> l empreinte NE BOUGE PAS. Sans cette exclusion, tout
	//    ajout de test du decodeur exigerait un bump de revision, ce qui viderait la revision
	//    de son sens (elle designerait des lignes produites identiques).
	avecTests := t.TempDir()
	ecrire(avecTests, "decode.go", "package p\n\nconst Largeur = 5\n")
	ecrire(avecTests, "scan.go", "package p\n\nfunc Scan() {}\n")
	ecrire(avecTests, "scan_test.go", "package p\n\nfunc TestScan() {}\n")
	ecrire(avecTests, "testdata/film.go", "package fixture\n")
	if got := empreinte(avecTests); got != ref {
		t.Errorf("empreinte modifiee par un test ou une fixture : %s != %s", got, ref)
	}

	// 4. LES MEMES SOURCES EN CRLF -> meme empreinte. C est ce qui rend le gate identique sur
	//    la CI (Linux) et sur un poste Windows.
	crlf := t.TempDir()
	ecrire(crlf, "decode.go", "package p\r\n\r\nconst Largeur = 5\r\n")
	ecrire(crlf, "scan.go", "package p\r\n\r\nfunc Scan() {}\r\n")
	if got := empreinte(crlf); got != ref {
		t.Errorf("empreinte sensible aux fins de ligne : %s != %s", got, ref)
	}
}
