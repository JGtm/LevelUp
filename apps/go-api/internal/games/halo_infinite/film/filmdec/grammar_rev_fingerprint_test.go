package filmdec

// grammar_rev_fingerprint_test.go — L'EMPREINTE DU DECODEUR, ETENDUE A TOUTE LA GRAMMAIRE.
//
// # LA REGLE A TROIS ETAGES QUE CE TEST REND EXECUTOIRE
//
//	GrammarRev              monte a TOUT changement de grammaire (une largeur, un cadre, un ordre
//	                        de composants, un lecteur neuf). C'est CE test qui l'exige.
//	KillSourceDecoderRev    monte quand la SORTIE de killsource peut changer — les lignes de kill
//	                        deja en base entrent alors au backlog de redecodage. Gardee par
//	                        `internal/sync/killcollector/decoder_rev_fingerprint_test.go`, qui
//	                        reste tel quel : il pose une AUTRE question sur le meme paquet.
//	SchemaVersion           monte quand le CONTENU CUIT change — `backfill-replay` re-cuit.
//
// # CE QUE CE TEST FAIT
//
// Il hache toutes les sources `.go` hors `_test.go` des TROIS paquets qui lisent les octets du
// film — `filmdec/`, `killsource/` et `analysis/objectiveevents/` — et compare au golden
// `testdata/grammar_rev.golden`, qui fige le couple (revision, empreinte) avec son
// historique. Toucher l'une ou l'autre le fait rougir ; le remettre au vert oblige a rouvrir la
// ligne de revision — donc a DECIDER si la grammaire a change, et si les deux autres etages
// doivent monter aussi.
//
// # POURQUOI CES TROIS PAQUETS DANS UNE SEULE EMPREINTE
//
// `killsource` lit les MEMES octets que `filmdec`, avec son propre lecteur de bits (le lot 4 de
// la trajectoire les fusionne). Tant qu ils sont deux, une largeur corrigee d un cote et pas de
// l autre est exactement le genre de divergence silencieuse que ce chantier cherche a rendre
// impossible. Une empreinte commune la fait sonner.
//
// `analysis/objectiveevents` est ENTRE LE 2026-09-14 (lot 1.1.5), et il a fallu un faux negatif
// pour le voir : ce paquet porte le lecteur du PIED DE FILM (`scanTh10Events`,
// `decodeTh10Block`) — des offsets d octets dans un bloc de 60, c est-a-dire de la grammaire au
// sens exact de la ligne `GrammarRev` ci-dessus (« une largeur, un cadre, un ordre de
// composants, un lecteur »). Le lot 1.1 a deplace l equipe d un evenement de l octet 55 a
// l octet 37 et CE TEST EST RESTE VERT : la revision a du etre montee a la main. Un garde-rail
// qui laisse passer le changement qu il existe pour attraper n en est pas un.
//
// Ce paquet DEMENAGERA sous `film/` au pas 5 de la revision (decision V5 du
// PLAN_DECODEUR_FILM : `analysis/filmsource` et ses voisins passent sous `film/internal/`). Le
// jour ou il bougera, `racinesGrammaire` echouera bruyamment sur un dossier vide — ce qui est
// exactement le comportement voulu, et non une regression a contourner.
//
// # CE QU'IL NE FAIT PAS, ET C'EST ASSUME
//
// Il ne distingue pas un changement de grammaire d'une reformulation de commentaire : le hachage
// porte sur les OCTETS. Un garde-rail qui voudrait ne mordre que sur le « significatif » devrait
// comprendre le decodeur — il rendrait des faux negatifs, c'est-a-dire le defaut qu'on ferme. Un
// faux positif coute une ligne a mettre a jour.
//
// Les fins de ligne sont NORMALISEES en LF avant hachage : le depot force `eol=lf`, mais un
// checkout mal configure rendrait l'empreinte verte en CI et rouge sur le poste, pour une raison
// etrangere a la grammaire.
//
// REGENERATION :
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run GrammarRevSuitLaGrammaire -update-grammar-rev

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// cheminGoldenGrammarRev : le golden, relatif au paquet.
const cheminGoldenGrammarRev = "testdata/grammar_rev.golden"

// updateGrammarRev : LA PORTE DE REGENERATION DE CE GOLDEN, ET D AUCUN AUTRE.
//
// NOMMEE, correctif de la revue R1 (P1-1) : accrochee au `-update` du fuzz, elle laissait
// `go test ./...filmdec/ -update` (sans `-run`) refiger l empreinte d une grammaire CASSEE en
// repondant `ok`. `flag` ne panique que sur un NOM deja pris — le paquet declare deja
// `-update-golden-familles`.
var updateGrammarRev = flag.Bool("update-grammar-rev", false,
	"reecrire testdata/grammar_rev.golden (lot 0.A.4) — CE golden seulement")

// fichierHorsGrammaire : le fichier qui PORTE la revision n'est pas de la grammaire.
//
// L EXCLURE REND LA SECONDE BRANCHE DU TEST ATTEIGNABLE (revue R1, P2-3). Tant que
// `grammar_rev.go` etait hache, faire monter `GrammarRev` SEULE changeait aussi l empreinte :
// le cas « la revision a change sans que la grammaire bouge » ne pouvait jamais se produire, et
// son message etait du code mort. La constante decrit la grammaire, elle n en fait pas partie.
const fichierHorsGrammaire = "grammar_rev.go"

// TestGrammarRevSuitLaGrammaire : une source de grammaire qui change sans montee de
// [GrammarRev] fait rougir ce test.
func TestGrammarRevSuitLaGrammaire(t *testing.T) {
	empreinte, n := empreinteGrammaire(t)
	if n == 0 {
		t.Fatal("aucune source hachee : le test ne garderait rien")
	}
	if *updateGrammarRev {
		ecrireGoldenGrammarRev(t, GrammarRev, empreinte)
		// UNE PORTE DE REGENERATION NE REND JAMAIS `ok` (revue R2, C1) : `go test` jette la sortie
		// d un paquet qui passe, donc une reecriture annoncee par `t.Logf` ou sur stderr est
		// INVISIBLE avec la commande documentee (sans `-v`).
		t.Fatalf("1 reference(s) reecrite(s) : %s (revision %s, empreinte %s) ; "+
			"relancer sans -update-grammar-rev pour verifier",
			cheminGoldenGrammarRev, GrammarRev, empreinte)
	}
	revFigee, empFigee := lireGoldenGrammarRev(t)
	switch {
	case revFigee == GrammarRev && empFigee != empreinte:
		t.Errorf("LA GRAMMAIRE A CHANGE SANS MONTEE DE REVISION.\n"+
			"  revision : %s (inchangee)\n  empreinte figee   : %s\n  empreinte obtenue : %s (%d fichiers)\n"+
			"Faire monter filmdec.GrammarRev (forme grammar-AAAA-MM-JJ), puis regenerer par -update-grammar-rev.\n"+
			"Se demander AUSSI : la sortie de killsource peut-elle changer (KillSourceDecoderRev, "+
			"backlog de redecodage) ? le contenu cuit change-t-il (SchemaVersion, recuisson) ?",
			revFigee, empFigee, empreinte, n)
	case revFigee != GrammarRev && empFigee == empreinte:
		t.Errorf("LA REVISION A CHANGE SANS QUE LA GRAMMAIRE BOUGE.\n"+
			"  revision figee : %s\n  revision du code : %s\n  empreinte : %s (inchangee)\n"+
			"Une revision qui monte sans changement de source rouvre un backlog pour rien : "+
			"la remettre, ou regenerer par -update-grammar-rev si la montee est deliberee.",
			revFigee, GrammarRev, empreinte)
	case revFigee != GrammarRev:
		t.Logf("revision et empreinte ont bouge ensemble (%s -> %s) : regenerer le golden par -update-grammar-rev",
			revFigee, GrammarRev)
		t.Errorf("golden perime : revision %s, empreinte %s (%d fichiers)", GrammarRev, empreinte, n)
	}
}

// entreeChroniqueGodoc / entreeChroniqueGolden : les deux formes d'une ENTREE de chronique.
//
// Le godoc de `grammar_rev.go` ouvre chaque entree neuve sur le mot `ENTREE` suivi de la
// revision entre accents graves ; le golden porte la sienne dans sa colonne « revision » de
// l'HISTORIQUE (`#   <date>  <revision>  <texte>`). Les deux formes sont DIFFERENTES parce que
// les deux fichiers le sont — l'un est du Go, l'autre une table lisible — et il n'y a rien a
// unifier : ce qui compte est qu'aucun des deux ne puisse s'arreter a un rang depasse.
var (
	entreeChroniqueGodoc  = regexp.MustCompile("(?m)^// ENTREE `(grammar-[0-9]{4}-[0-9]{2}-[0-9]{2}(?:\\.[0-9]+)?)`")
	entreeChroniqueGolden = regexp.MustCompile(`(?m)^#\s+[0-9]{4}-[0-9]{2}-[0-9]{2}\s+(grammar-[0-9]{4}-[0-9]{2}-[0-9]{2}(?:\.[0-9]+)?)\s`)
)

// TestChroniqueCouvreLaRevisionCourante : [GrammarRev] a-t-elle son entree, des DEUX cotes ?
//
// # LE DEFAUT QUE CE TEST FERME (revue de jalon M1, ronde 2, constat F5)
//
// Trois lots de corrections partis de la meme base `.11` ont empile trois blocs annoncant
// chacun « `.11` -> `.12` », suivis de lignes « FUSION ... au rang suivant » qui racontaient une
// renumerotation que l'integration n'a jamais faite. Resultat : la chronique s'arretait a `.12`
// pendant que la constante valait `.14`, et les changements de COMPORTEMENT portes par `.13`
// (porte unique `MPPWidthsForFilm`) et `.14` n'avaient AUCUNE entree. Rien ne rougissait — le
// ratchet d'empreinte ne tient que le couple (revision, empreinte), jamais ce que la revision
// RACONTE.
//
// Modele : `replay/document_shape_test.go`, `TestDocumentShapeSchemaHasChronicleEntry`, qui
// pose la meme exigence sur `SchemaVersion`.
//
// # CE QU'IL NE FAIT PAS
//
// Il ne relit pas les entrees ANTERIEURES au 2026-09-16 : elles ont trois formes de prose nees
// a des jours differents, et normaliser le passe n'ajouterait rien. Il ne mord que sur la
// valeur COURANTE — la seule qu'une montee puisse laisser sans entree.
func TestChroniqueCouvreLaRevisionCourante(t *testing.T) {
	for _, src := range []struct {
		quoi    string
		chemin  string
		forme   *regexp.Regexp
		exemple string
	}{
		{"le godoc de grammar_rev.go", cheminGodocGrammarRev(t), entreeChroniqueGodoc,
			"// ENTREE `" + GrammarRev + "` (AAAA-MM-JJ, lot) : ..."},
		{"l'HISTORIQUE du golden", cheminGoldenGrammarRev, entreeChroniqueGolden,
			"#   AAAA-MM-JJ  " + GrammarRev + "  lot : ..."},
	} {
		blob, err := os.ReadFile(src.chemin) //nolint:gosec // chemins deduits du paquet
		if err != nil {
			t.Fatalf("%s illisible (%s) : %v", src.quoi, src.chemin, err)
		}
		var vues []string
		trouvee := false
		for _, m := range src.forme.FindAllStringSubmatch(string(blob), -1) {
			vues = append(vues, m[1])
			if m[1] == GrammarRev {
				trouvee = true
			}
		}
		if !trouvee {
			t.Errorf("GrammarRev = %s n'a AUCUNE entree dans %s (entrees declarees : %v).\n"+
				"Une montee sans entree ne dit pas ce qu'elle change, et la chronique s'arrete "+
				"a un rang que la constante a depasse (constat F5).\nForme attendue :\n  %s",
				GrammarRev, src.quoi, vues, src.exemple)
		}
	}
}

// cheminGodocGrammarRev : le fichier qui porte la constante et sa chronique, resolu par
// `runtime.Caller` — jamais un chemin relatif au repertoire courant (meme raison que
// [racinesGrammaire]).
func cheminGodocGrammarRev(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	return filepath.Join(filepath.Dir(ici), fichierHorsGrammaire)
}

// empreinteGrammaire hache les sources non-test de `filmdec` ET de `killsource`.
func empreinteGrammaire(t *testing.T) (string, int) {
	t.Helper()
	h := sha256.New()
	total := 0
	for _, racine := range racinesGrammaire(t) {
		n, err := hacherSourcesGrammaire(h, racine)
		if err != nil {
			t.Fatalf("hachage de %s : %v", racine, err)
		}
		if n == 0 {
			t.Fatalf("aucune source .go dans %s — l'arborescence a bouge, "+
				"ce test doit ECHOUER bruyamment plutot que hacher du vide", racine)
		}
		total += n
	}
	return hex.EncodeToString(h.Sum(nil)), total
}

// racinesGrammaire rend les TROIS paquets haches, resolus par `runtime.Caller`.
//
// PAS un chemin relatif au repertoire courant : le jour ou un paquet demenage (ADR 0012), ce
// test doit echouer bruyamment plutot que hacher un dossier vide.
func racinesGrammaire(t *testing.T) []string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	// .../internal/games/halo_infinite/film/filmdec -> .../internal
	internalDir := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(ici)))))
	filmDir := filepath.Join(internalDir, "games", "halo_infinite", "film")
	return []string{
		filepath.Join(filmDir, "filmdec"),
		filepath.Join(filmDir, "killsource"),
		filepath.Join(internalDir, "analysis", "objectiveevents"),
	}
}

// hacherSourcesGrammaire hache les sources Go non-test d'un arbre, dans un ordre stable, et rend
// le nombre de fichiers.
//
// Le chemin RELATIF entre dans le hachage a cote du contenu : sans lui, renommer un fichier sans
// en changer une ligne laisserait l'empreinte immobile — or un renommage de fichier de grammaire
// est exactement le mouvement qu'on veut voir passer par la revision. `testdata/` est ecarte :
// ce sont des fixtures, pas de la grammaire.
func hacherSourcesGrammaire(h interface{ Write([]byte) (int, error) }, racine string) (int, error) {
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
		if !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") ||
			nom == fichierHorsGrammaire {
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
		lus = append(lus, fichier{
			rel:     filepath.ToSlash(filepath.Join(filepath.Base(racine), rel)),
			contenu: []byte(strings.ReplaceAll(string(blob), "\r\n", "\n")),
		})
		return nil
	})
	if err != nil {
		return 0, err
	}
	sort.Slice(lus, func(i, j int) bool { return lus[i].rel < lus[j].rel })
	for _, f := range lus {
		_, _ = h.Write([]byte(f.rel))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(f.contenu)
	}
	return len(lus), nil
}

// lireGoldenGrammarRev rend le couple (revision, empreinte) fige.
func lireGoldenGrammarRev(t *testing.T) (revision, empreinte string) {
	t.Helper()
	blob, err := os.ReadFile(cheminGoldenGrammarRev)
	if err != nil {
		t.Fatalf("golden %s illisible : %v — il est VERSIONNE, son absence est une erreur",
			cheminGoldenGrammarRev, err)
	}
	for _, ligne := range strings.Split(strings.ReplaceAll(string(blob), "\r\n", "\n"), "\n") {
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			continue
		}
		champs := strings.Split(ligne, "\t")
		if len(champs) != 2 || champs[0] == "" || champs[1] == "" {
			t.Fatalf("golden %s : ligne malformee %q — attendu `revision<TAB>empreinte`",
				cheminGoldenGrammarRev, ligne)
		}
		if revision != "" {
			t.Fatalf("golden %s : plusieurs lignes de donnees — le fichier fige UN couple",
				cheminGoldenGrammarRev)
		}
		revision, empreinte = champs[0], champs[1]
	}
	if revision == "" {
		t.Fatalf("golden %s : aucune ligne de donnees", cheminGoldenGrammarRev)
	}
	return revision, empreinte
}

// ecrireGoldenGrammarRev reecrit la SEULE ligne de donnees, en gardant l'en-tete et l'historique.
func ecrireGoldenGrammarRev(t *testing.T, revision, empreinte string) {
	t.Helper()
	blob, err := os.ReadFile(cheminGoldenGrammarRev)
	if err != nil {
		t.Fatalf("golden %s illisible : %v", cheminGoldenGrammarRev, err)
	}
	var sortie []string
	for _, ligne := range strings.Split(strings.ReplaceAll(string(blob), "\r\n", "\n"), "\n") {
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			sortie = append(sortie, ligne)
		}
	}
	sortie = append(sortie[:len(sortie)-1], revision+"\t"+empreinte, "")
	if err := os.WriteFile(cheminGoldenGrammarRev, []byte(strings.Join(sortie, "\n")), 0o600); err != nil {
		t.Fatalf("ecriture du golden %s : %v", cheminGoldenGrammarRev, err)
	}
	t.Logf("golden regenere : %s / %s", revision, empreinte)
}
