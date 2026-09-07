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
// Il hache les sources NON-TEST du paquet decodeur et compare a
// [killSourceDecoderFingerprint], figee A COTE de la revision. Toucher le decodeur fait rougir
// ce test ; le remettre au vert demande de rouvrir la ligne de la revision — donc de DECIDER si
// les lignes en base doivent etre redecodees. C est tout ce qu on lui demande.
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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

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

// TestKillSourceDecoderRevSuitLeDecodeur — LE GATE. Sources du decodeur modifiees sans bump de
// la revision = rouge, avec le geste exact a faire dans le message.
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
	if empreinte != killSourceDecoderFingerprint {
		t.Fatalf(`LE DECODEUR A CHANGE ET SA REVISION N A PAS BOUGE.

  paquet    : internal/games/halo_infinite/film/killsource (%d fichiers non-test)
  attendue  : %s
  mesuree   : %s
  revision  : KillSourceDecoderRev = %q

DEUX GESTES, DANS COLLECTOR.GO, ET LES DEUX SONT OBLIGATOIRES :

  1. BUMPER KillSourceDecoderRev (ex. "killsource-AAAA-MM-JJ") si le changement modifie les
     lignes produites — c est ce qui rend les matchs deja decodes a nouveau candidats au
     backlog (postsync.go, conditionBacklog). Si le changement ne touche PAS les lignes
     produites (commentaire, renommage interne), laisser la revision et l ecrire dans le
     commit : le choix doit etre explicite, pas implicite.
  2. RECOPIER l empreinte mesuree ci-dessus dans killSourceDecoderFingerprint.

Sans le geste 2 ce test reste rouge ; sans le geste 1 les lignes en base restent servies avec
l ancien decodage, sans compteur et sans reprise possible.`,
			n, killSourceDecoderFingerprint, empreinte, KillSourceDecoderRev)
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
