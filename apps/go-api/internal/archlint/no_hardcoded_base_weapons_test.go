package archlint

// no_hardcoded_base_weapons_test.go — AUCUNE LISTE D'ARMES « DE BASE » EN DUR.
//
// # CE QU'IL INTERDIT, ET POURQUOI
//
// Le niveau d'une arme (base / terrain / puissance) se MESURE : « base » se lit dans
// l'equipement de DEPART du film, match par match ; « terrain » et « puissance » se lisent sur
// la CARTE, par l'emplacement Forge que le socle du match confirme. Ecrire quelque part
// `[]string{"hinf_ma40_ar", "hinf_sidekick", "hinf_br75"}` remplacerait cette mesure par une
// opinion — et une opinion qui vieillit : 343 change les equipements de depart d'une playlist
// a l'autre et d'une saison a l'autre, et les modes a departs aleatoires n'ont pas d'arme de
// base du tout.
//
// MESURE QUI LE JUSTIFIE (76 artefacts, 2026-09-14) : en partie classee et rapide les trois
// armes de depart pesent 94,4 % des equipements de premiere emission ; en BTB ce ne sont PAS
// les memes (Bandit 47,6 % et MA40 45,1 %, contre MA40/Sidekick/BR75) ; en Super Fiesta la
// distribution est PLATE sur 22 cles. Une liste unique serait fausse sur deux modes sur trois.
//
// # LA REGLE, ET SA FRONTIERE
//
// Interdit : une COLLECTION de plusieurs cles d'arme du titre (`hinf_*`) dans les couches qui
// jugent (`internal/analysis/`, `apps/web/src/features/`). C'est la forme qu'une liste « de
// base » prendrait.
//
// Autorise : une cle SEULE (un temoin de test, une constante d'une regle qui parle d'UNE arme),
// et tout le paquet `internal/games/` — le REGISTRE d'armes du titre y vit, c'est sa place.
//
// Meme doctrine que `no_slug_comparison_test.go` : le ratchet ne juge pas une intention, il
// interdit une FORME.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// cleArmeRe : une cle d'arme du titre entre guillemets (« "hinf_ma40_ar" »).
var cleArmeRe = regexp.MustCompile(`["'` + "`" + `]h(?:inf|5)_[a-z0-9_]+["'` + "`" + `]`)

// seuilCollection : a partir de COMBIEN de cles sur une meme ligne (ou deux lignes
// consecutives) on parle d'une LISTE. Deux cles peuvent se comparer ; trois se recensent.
const seuilCollection = 3

// racinesSurveillees : les couches qui JUGENT. Le registre d'armes (`internal/games/`) en est
// exclu : c'est lui, la source.
var racinesSurveillees = []string{
	filepath.Join("..", "analysis"),
	// LES DEUX COUCHES QUI PROJETTENT ET QUI SERVENT (ajoutees a la revue du 2026-09-14) : le
	// ratchet ne gardait que la couche des algorithmes et le web. Or une liste d armes en dur
	// aurait tout autant sa place — et tout autant tort — dans la projection au fil de l eau ou
	// dans un service qui assemble une reponse.
	filepath.Join("..", "sync", "replayartifacts"),
	filepath.Join("..", "service"),
	filepath.Join("..", "..", "..", "web", "src", "features"),
}

func TestAucuneListeDArmesDeBaseEnDur(t *testing.T) {
	for _, racine := range racinesSurveillees {
		if _, err := os.Stat(racine); err != nil {
			t.Fatalf("racine surveillee introuvable (%s) : %v — le ratchet ne garderait rien", racine, err)
		}
		balayer(t, racine)
	}
}

func balayer(t *testing.T, racine string) {
	t.Helper()
	err := filepath.Walk(racine, func(chemin string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		switch filepath.Ext(chemin) {
		case ".go", ".ts", ".tsx":
		default:
			return nil
		}
		// LES TESTS SONT HORS PERIMETRE, et ce n est pas un trou : un temoin qui ENUMERE des
		// cles d arme prouve un tri ou un libelle — il ne JUGE aucun niveau. La regle vise le
		// code qui decide, et lui seul.
		if estFichierDeTest(chemin) {
			return nil
		}
		src, err := os.ReadFile(chemin) //nolint:gosec // chemins issus du Walk du depot
		if err != nil {
			return err
		}
		verifierFichier(t, chemin, string(src))
		return nil
	})
	if err != nil {
		t.Fatalf("balayage de %s : %v", racine, err)
	}
}

// verifierFichier cherche une COLLECTION de cles d'arme sur une FENETRE de deux lignes — la
// forme qu'une liste prend, qu'elle tienne sur une ligne ou qu'elle soit mise en forme.
func verifierFichier(t *testing.T, chemin, src string) {
	t.Helper()
	lignes := strings.Split(src, "\n")
	for i := range lignes {
		fenetre := lignes[i]
		if i+1 < len(lignes) {
			fenetre += "\n" + lignes[i+1]
		}
		if n := len(cleArmeRe.FindAllString(fenetre, -1)); n >= seuilCollection {
			t.Errorf("%s:%d — %d cles d'arme du titre groupees : une liste d'armes en dur "+
				"remplacerait la MESURE du niveau (equipement de depart du film, emplacement de "+
				"la carte) par une opinion qui vieillit. Le niveau se lit de "+
				"`internal/analysis/weapontier` cote Go, de `features/_shared/usage` et "+
				"`features/match-replay/model/weaponTier.ts` cote web.\n  %s",
				chemin, i+1, n, strings.TrimSpace(fenetre))
		}
	}
}

// estFichierDeTest reconnait les temoins des deux langages.
func estFichierDeTest(chemin string) bool {
	base := filepath.Base(chemin)
	return strings.HasSuffix(base, "_test.go") ||
		strings.Contains(base, ".test.") ||
		strings.Contains(base, ".guard.") ||
		strings.Contains(base, ".spec.")
}
