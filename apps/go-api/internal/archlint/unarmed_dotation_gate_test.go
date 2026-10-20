package archlint

// unarmed_dotation_gate_test.go — UN CONSTRUCTEUR DE DOTATION NE CONSULTE JAMAIS LE CATALOGUE
// D ARMES DIRECTEMENT (retours du rejeu, revue adverse du lot M6, constat R6, 2026-09-24).
//
// POURQUOI. La regle NOMMEE des mains nues (`filmshell.IsUnarmedFamily`, decision de l utilisateur
// du 2026-09-24 : l objet n entre dans AUCUNE dotation affichee) vit dans UN passage,
// `dotationWeaponName` (`film/replay/loadouts.go`). Le garde-rail du litteral
// (`unarmed_family_literal_test.go`) interdit de DOUBLER l identifiant ; il n interdit pas de
// l OUBLIER : un constructeur neuf qui demanderait lui-meme son nom a `weaponv3.WeaponName`
// ecarterait l objet PAR ACCIDENT (le catalogue de decodage ne le connait pas aujourd hui), et
// le laisserait passer le jour ou le catalogue le connaitrait. C est exactement le cas des
// dotations de naissance du lot M3 (vague D) : elles devront passer par `dotationWeaponName`.
//
// LA REGLE : dans le paquet de publication `film/replay`, `weaponv3.WeaponName` n est cite
// (appel ou valeur de fonction) qu aux sites ci-dessous, chacun motive.

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// catalogueArmesSitesAutorises : fichier -> nombre d appels autorises, avec la raison.
var catalogueArmesSitesAutorises = map[string]int{
	// Le passage unique des dotations, derriere la regle des mains nues.
	"loadouts.go": 1,
	// Les ARMES AU SOL (socles) : un objet du monde, jamais une dotation ; le nom sert au repli des
	// alias d un meme canon. Autorise le 2026-09-24 (lot M6), revoir si le fichier devient un
	// constructeur de dotation.
	"ground_weapon_rules.go": 1,
}

func TestDotationPasseParLaRegleDesMainsNues(t *testing.T) {
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	dir := filepath.Join(filepath.Dir(filepath.Dir(ici)), "games", "halo_infinite", "film", "replay")
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", dir, err)
	}
	vus := map[string]int{}
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(dir, nom)) //nolint:gosec // source du module
		if rerr != nil {
			t.Fatal(rerr)
		}
		for _, ligne := range strings.Split(string(data), "\n") {
			code := strings.TrimSpace(ligne)
			if strings.HasPrefix(code, "//") {
				continue
			}
			vus[nom] += strings.Count(code, "weaponv3.WeaponName")
		}
	}
	var ecarts []string
	for nom, n := range vus {
		if n > 0 && n != catalogueArmesSitesAutorises[nom] {
			ecarts = append(ecarts, nom)
		}
	}
	for nom, n := range catalogueArmesSitesAutorises {
		if vus[nom] != n {
			ecarts = append(ecarts, nom+" (site autorise disparu ou deplace)")
		}
	}
	sort.Strings(ecarts)
	if len(ecarts) > 0 {
		t.Errorf("appel direct au catalogue d armes hors des sites autorises : %s — un "+
			"constructeur de dotation passe par `dotationWeaponName` (regle nommee des mains nues)",
			strings.Join(ecarts, ", "))
	}
}
