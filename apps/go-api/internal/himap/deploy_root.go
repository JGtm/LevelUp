// Package himap — deploy_root.go : ou trouver l'installation de Halo Infinite.
//
// POURQUOI CE FICHIER EXISTE. Le chemin `D:/SteamLibrary/.../deploy` etait ecrit en dur a
// TROIS endroits : les deux binaires de production d'assets et les tests `gamefiles` de ce
// package. Le 2026-08-08, ce disque n'est plus monte sur le poste de developpement — et
// rien ne l'a signale : les tests se declarent « module absent » et passent au vert, les
// binaires echouent sur un dossier introuvable. Un garde-rail qui saute en silence ne
// garde rien.
//
// La liste des racines connues vit ICI et nulle part ailleurs (garde-rail :
// deploy_root_test.go interdit le litteral partout ailleurs).
package himap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DeployRootEnv prime sur toute detection : une installation ailleurs se declare par
// l'environnement, sans toucher au code.
const DeployRootEnv = "LEVELUP_HALO_DEPLOY"

// knownDeployRoots : installations connues, essayees dans cet ordre.
var knownDeployRoots = []string{
	`C:/Program Files (x86)/Steam/steamapps/common/Halo Infinite/deploy`,
	`D:/SteamLibrary/steamapps/common/Halo Infinite/deploy`,
	`C:/XboxGames/Halo Infinite/Content/deploy`,
}

// ErrNoDeployRoot signale qu'aucune installation n'a ete trouvee. Les appelants hors ligne
// (production d'assets, tests gamefiles) doivent s'ARRETER ou se declarer absents, jamais
// continuer sur un chemin devine.
var ErrNoDeployRoot = fmt.Errorf("himap: aucune installation de Halo Infinite trouvee "+
	"(definir %s, ou installer le jeu a l'un des emplacements connus)", DeployRootEnv)

// DeployRoot rend la racine `deploy` d'une installation de Halo Infinite.
func DeployRoot() (string, error) {
	if r := strings.TrimSpace(os.Getenv(DeployRootEnv)); r != "" {
		if _, err := os.Stat(r); err != nil {
			return "", fmt.Errorf("himap: %s pointe sur %q, illisible : %w", DeployRootEnv, r, err)
		}
		return r, nil
	}
	for _, r := range knownDeployRoots {
		if _, err := os.Stat(r); err == nil {
			return r, nil
		}
	}
	return "", ErrNoDeployRoot
}

// LevelsDir rend le dossier des cartes multijoueur d'une variante de depot.
//
// Les trois variantes ne portent PAS la meme chose : `pc` a la geometrie de rendu (c'est
// elle qu'il faut pour les instances), `ds` est le build serveur dedie — depouille du
// rendu, mais c'est lui qui porte les bornes monde du BSP — et `any` porte les materiaux
// et le tag de scenario (noms de zones).
func LevelsDir(variant string) (string, error) {
	root, err := DeployRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, variant, "levels", "multi"), nil
}

// modulesGeneriques : jetons de nom de carte qui ne designent aucun dossier.
var modulesGeneriques = map[string]bool{"map": true, "ranked": true, "": true, "-": true}

// ChercheModuleInstalle fait le pont entre le nom de module du CATALOGUE et le dossier de
// l'INSTALLATION.
//
// Ils ne coincident pas : le catalogue nomme « cliffhanger_ridgeline » ce que le jeu range sous
// « ridgeline ». Le jeu PREFIXE en outre ses dossiers par le mode d'origine de la carte —
// `ctf_bazaar`, `btb_highpower`, `sgh_interlock`, `va_behemoth` — quand le catalogue, lui, dit
// « bazaar_map ». On enumere donc l'installation et on apparie sur les JETONS, plutot que de
// deviner un prefixe.
//
// Un module introuvable est SIGNALE par `false`, jamais suppose absent : plusieurs cartes du
// catalogue ne sont pas installees localement, et les confondre avec une chaine cassee ferait
// passer une absence pour un echec.
func ChercheModuleInstalle(nom string) (string, bool) {
	dir, err := LevelsDir("pc")
	if err != nil {
		return "", false
	}
	dossiers, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	var jetons []string
	for _, j := range strings.FieldsFunc(nom, func(r rune) bool { return r == '_' || r == '-' }) {
		if !modulesGeneriques[j] {
			jetons = append(jetons, j)
		}
	}
	for _, d := range dossiers {
		if !d.IsDir() {
			continue
		}
		base := d.Name()
		p := filepath.Join(dir, base, base+"-rtx-new.module")
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if base == nom {
			return p, true
		}
		for _, j := range jetons {
			// Suffixe : `ctf_bazaar` porte `bazaar`. Egalite pour les dossiers sans prefixe.
			if base == j || strings.HasSuffix(base, "_"+j) {
				return p, true
			}
		}
	}
	return "", false
}
