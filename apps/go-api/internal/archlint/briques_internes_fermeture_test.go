// briques_internes_fermeture_test.go — la fermeture de dépendances du DOMAINE ne contient pas
// l'API Halo (plan D3, lot 1, item 1.3).
//
// # POURQUOI
//
// La brique « API Halo » est au bas de la pile : `domain`, `analysis` et `games/canonical` ne la
// connaissent pas (ils décrivent ce que Halo veut dire pour LevelUp, jamais comment on lui parle).
// L'import direct est déjà impossible à écrire sans cycle pour `domain` ; la FERMETURE
// (`go list -deps -test`) est ce qui attrape l'import indirect (analysis -> A -> haloapi), que le
// parcours des imports directs ne voit pas (même méthode que
// `TestFermetureDeAnalysisSansDecodeurDeFilm`).
//
// Le paquet surveillé est la brique (`internal/haloapi`, arbre).
//
// MESURE du 2026-10-29 (HEAD b92916328) : 299 paquets dans la fermeture, AUCUN des paquets
// surveillés. Il n'y a donc pas d'allowlist : toute fuite est une faute à couper, pas une dette.
//
// Mutation qui doit le faire rougir : importer `levelup/go-api/internal/haloapi/infinite` depuis un
// fichier d'`internal/analysis/`.
package archlint

import (
	"os/exec"
	"strings"
	"testing"
)

// racinesFermetureDomaine : les paquets dont la fermeture est jugée.
var racinesFermetureDomaine = []string{"./internal/domain/...", "./internal/analysis/...", "./internal/games/canonical/..."}

// arbresHaloapiInterdits : les arbres (chemins d'import) que la fermeture du domaine ne contient pas.
var arbresHaloapiInterdits = []string{
	prefixeInterneAPI + "haloapi",
}

// fuitesHaloapi rend, triés, les paquets de la fermeture qui sont dans un arbre interdit. Fonction
// pure : l'auto-preuve l'exerce sur un listing synthétique.
func fuitesHaloapi(paquets []string) []string {
	var fuites []string
	for _, p := range paquets {
		for _, arbre := range arbresHaloapiInterdits {
			if sousArbre(p, arbre) {
				fuites = append(fuites, p)
				break
			}
		}
	}
	return fuites
}

func TestFermetureDeDomainEtAnalysisSansHaloapi(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("go introuvable dans le PATH : %v", err)
	}
	args := append([]string{"list", "-deps", "-test", "-f", "{{.ImportPath}}"}, racinesFermetureDomaine...)
	cmd := exec.Command(goBin, args...)
	cmd.Dir = apiRootDepuisIci(t)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	brut, err := cmd.Output()
	if err != nil {
		t.Fatalf("go %s : %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	paquets := paquetsDeFermeture(string(brut))
	if len(paquets) < 100 || !contientChaine(paquets, prefixeInterneAPI+"domain") ||
		!contientChaine(paquets, prefixeInterneAPI+"analysis") {
		t.Fatalf("la fermeture lue compte %d paquets et ne contient pas domain et analysis : la sortie de "+
			"go list n'est plus celle qu'on parse", len(paquets))
	}
	if fuites := fuitesHaloapi(paquets); len(fuites) > 0 {
		t.Errorf("la fermeture de domain, analysis et games/canonical contient l'API Halo (plan D3 ; "+
			"`go list -deps -test` pour retrouver le chemin) : couper l'import qui la tire — la brique est "+
			"au bas de la pile, le domaine ne la connaît pas :\n  %s", strings.Join(fuites, "\n  "))
	}
}

// TestFermetureDeDomainEtAnalysis_AutoPreuve : le contrôle pur rougit sur l'arbre de la brique (y
// compris les variantes de test et les sous-paquets) et laisse passer les voisins qui partagent
// seulement le début du nom.
func TestFermetureDeDomainEtAnalysis_AutoPreuve(t *testing.T) {
	sortie := "levelup/go-api/internal/domain\n" +
		"levelup/go-api/internal/haloapi/xblauth\n" +
		"levelup/go-api/internal/haloapi\n" +
		"levelup/go-api/internal/haloapi/infinite [levelup/go-api/internal/analysis.test]\n" +
		"levelup/go-api/internal/haloapi/xblauth.test\n" +
		"levelup/go-api/internal/platform/haloextra\n" +
		"levelup/go-api/internal/sync/haloclient_reexport\n" +
		"levelup/go-api/internal/haloapix\n" +
		"levelup/go-api/internal/platform/auth\n"
	got := strings.Join(fuitesHaloapi(paquetsDeFermeture(sortie)), ",")
	attendu := "levelup/go-api/internal/haloapi,levelup/go-api/internal/haloapi/infinite," +
		"levelup/go-api/internal/haloapi/xblauth"
	if got != attendu {
		t.Fatalf("fuites = %s, attendu %s", got, attendu)
	}
	if fuites := fuitesHaloapi(paquetsDeFermeture("levelup/go-api/internal/domain\nlevelup/go-api/internal/platform/netguard\n")); len(fuites) != 0 {
		t.Fatalf("fermeture saine refusée : %v", fuites)
	}
}
