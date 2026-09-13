package testutil

// replay_chronicle.go — LES VERSIONS DE SCHEMA QUE LA CHRONIQUE DU REJEU DECLARE.
//
// POURQUOI CE HELPER EXISTE, ET POURQUOI ICI. Deux garde-rails ont besoin de la MEME liste et
// vivent dans deux paquets differents : l'empreinte de forme du document
// (`games/halo_infinite/film/replay/document_shape_test.go`, qui exige une entree de chronique
// pour la version COURANTE) et la preuve par version que les anciens goldens ne se regenerent
// jamais (`replaybuild/artifact_schema_history_test.go`, qui la veut pour toutes les versions
// INFERIEURES). Ecrire deux fois le meme extracteur, c'est se garantir qu'ils divergeront sur
// la premiere forme d'en-tete non prevue — et qu'un des deux deviendra muet sans le dire.
//
// CE HELPER N'IMPORTE PAS LE PAQUET `replay`, ET CE N'EST PAS UN GOUT : les fichiers de test du
// paquet `replay` sont DANS ce paquet, donc un import inverse ferait un cycle. Il ne lit que du
// texte, et c'est tout ce dont il a besoin.
//
// LA CHRONIQUE N'A PAS UNE SEULE FORME D'EN-TETE, constate sur pieces le 2026-09-13 : trois
// formes coexistent, nees a des mois differents, et aucune n'est fautive —
//
//	// v54 (2026-09-12, lot G.2bis) — ...        v2 a v20, v40 a v50, v52 a v54
//	// SCHEMA 29 (2026-08-31) — LA LUNETTE. ...  v29
//	// CE QUE LA VERSION 22 PORTE, ET ...        v21 a v39 pour l'essentiel
//
// L'extracteur les reconnait TOUTES LES TROIS, et rien d'autre : une version citee au fil d'une
// phrase (« un artefact v7 doit se voir comme a re-cuire ») n'est pas une entree.
//
// DEUX NUMEROS N'EXISTENT PAS, ET C'EST ECRIT DANS LA CHRONIQUE ELLE-MEME : 32 et 51 ont ete
// SAUTES a la renumerotation de deux lots paralleles (cf. l'entree v33, « renumerote 33/34 au
// merge du 2026-09-01 »). La version 1, elle, est anterieure a la chronique. C'est pourquoi ce
// helper rend CE QUE LA CHRONIQUE DECLARE et jamais un intervalle 1..N : un appelant qui
// comblerait les trous affirmerait des versions qui n'ont jamais ete cuites.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

// ReplayChroniclePath : le fichier qui porte la chronique, depuis la racine du depot.
func ReplayChroniclePath(repoRoot string) string {
	return filepath.Join(repoRoot, "apps", "go-api", "internal", "games", "halo_infinite",
		"film", "replay", "document_chronicle.go")
}

// chronicleHeaders : les trois formes d'en-tete attestees (cf. en-tete du fichier). Chacune
// capture le numero de version en groupe 1.
var chronicleHeaders = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^// v(\d+) \(`),
	regexp.MustCompile(`(?m)^// SCHEMA (\d+) `),
	regexp.MustCompile(`(?m)^// CE QUE (?:LA VERSION|LE SCHEMA) (\d+) `),
}

// ReplayChronicleVersions rend, triees, les versions de schema que la chronique DECLARE.
//
// L'erreur est destinee a un t.Fatalf : une chronique illisible ou vide est un garde-rail
// mort, pas un cas nominal a ignorer.
func ReplayChronicleVersions() ([]int, error) {
	root, err := RepoRoot()
	if err != nil {
		return nil, err
	}
	path := ReplayChroniclePath(root)
	raw, err := os.ReadFile(path) //nolint:gosec // chemin deduit de la racine du depot
	if err != nil {
		return nil, fmt.Errorf("testutil: chronique du rejeu illisible (%s) : %w", path, err)
	}
	versions := chronicleVersionsFrom(string(raw))
	if len(versions) == 0 {
		return nil, fmt.Errorf("testutil: aucune entree de chronique trouvee dans %s — la forme "+
			"des en-tetes a-t-elle change ? (un extracteur muet rend tout garde-rail inerte)", path)
	}
	return versions, nil
}

// chronicleVersionsFrom extrait les versions d'un texte de chronique. Separe de la lecture
// disque pour etre eprouve sur des fixtures, decoys compris (cf. replay_chronicle_test.go).
func chronicleVersionsFrom(src string) []int {
	vues := map[int]bool{}
	for _, re := range chronicleHeaders {
		for _, m := range re.FindAllStringSubmatch(src, -1) {
			n, err := strconv.Atoi(m[1])
			if err != nil || n <= 0 {
				continue
			}
			vues[n] = true
		}
	}
	out := make([]int, 0, len(vues))
	for v := range vues {
		out = append(out, v)
	}
	sort.Ints(out)
	return out
}
