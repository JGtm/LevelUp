package archlint

// film_sans_hote_test.go — LA BRIQUE NE CONNAIT PAS SON HOTE (ADR 0034, § "What the brick does not decide" ;
// ADR 0041).
//
// # LA REGLE
//
// La production de `libs/film` (fichiers `.go` hors `_test.go`, hors `research/` et hors fichiers sous
// `//go:build research`) ne nomme ni l'application qui l'embarque, ni sa propre adresse d'avant le module,
// ni un chemin de machine. Le test refuse, COMMENTAIRES COMPRIS (un commentaire qui cite l'hote est un
// couplage documentaire : il raconte un depot que le lecteur de la brique n'a pas), ces six litteraux :
//
//	LEVELUP_                            une variable d'environnement de l'hote ;
//	apps/go-api                         le dossier de l'hote ;
//	internal/games/halo_infinite/film   l'adresse de la brique avant qu'elle soit un module ;
//	C:\  /opt/  /home/                  un chemin absolu de poste ou de serveur.
//
// La brique recoit ce qu'il lui faut en parametre ; ce qui se decide chez l'hote se decrit chez l'hote.
//
// # ALLOWLIST (DATEE, FERMEE)
//
// Deux exceptions, toutes sous `libs/film/revision/`, et le test refuse qu'une troisieme s'ajoute sans
// raison ecrite :
//
//   - `porte.go` construit `LEVELUP_UPDATE_<NOM>` et cite `LEVELUP_CONTRACT_FIXTURES` : la porte a deux
//     verrous des goldens de revision, dont la variable est le second verrou (decision D-3 du plan de la
//     fin du decouplage). Le prefixe de variable est une decision d'outillage de la brique, lue par un
//     paquet qui ne lit pas l'environnement lui-meme.
//   - `cadre.go`, `couches.go` et `perimetre.go` portent le CADRE HISTORIQUE du hachage (ADR 0041 D8) : les
//     empreintes des cinq couches sont calculees sous l'ancien chemin de la brique, et les goldens de
//     perimetre le nomment. Le litteral y est le contrat de l'empreinte, pas une habitude.
//
// Une entree dont le litteral a disparu du fichier se retire (`TestHoteAdmisNEstPasPerime`).
//
// # LA MUTATION QUI DOIT ROUGIR
//
// Ajouter `const x = "apps/go-api/cmd"` (ou `os.Getenv("LEVELUP_X")`, ou un chemin `/home/...`) dans un fichier
// non-test de `libs/film/replay` : le test nomme le fichier, la ligne, le litteral et le remede.

import (
	"bytes"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// litterauxDeLHote : ce que la production de la brique ne contient pas.
var litterauxDeLHote = []string{
	"LEVELUP_",
	"apps/go-api",
	"internal/games/halo_infinite/film",
	`C:\`,
	"/opt/",
	"/home/",
}

// hoteAdmis : fichier (chemin de garde-rail) -> litteraux qu'il peut porter, avec la raison en tete de
// ce fichier-ci. DATEE du 2026-11-08, FERMEE : aucune entree neuve sans raison ecrite ici.
var hoteAdmis = map[string][]string{
	// 2026-11-08, decision D-3 : la porte a deux verrous des goldens de revision. Retrait : le jour ou la
	// variable est decidee par l'appelant et passee en parametre a la porte.
	racineBriqueFilm + "/revision/porte.go": {"LEVELUP_"},
	// 2026-11-08, ADR 0041 D8 : le cadre historique du hachage. Retrait : jamais tant que les empreintes
	// des cinq couches sont calculees sous l'ancien chemin ; le jour ou elles sont regenerees ensemble, a
	// revision decidee, le cadre se retire avec elles.
	racineBriqueFilm + "/revision/cadre.go":     {"apps/go-api", "internal/games/halo_infinite/film"},
	racineBriqueFilm + "/revision/couches.go":   {"internal/games/halo_infinite/film"},
	racineBriqueFilm + "/revision/perimetre.go": {"internal/games/halo_infinite/film"},
}

// plancherFichiersBalayesHote : LE PLANCHER CONTRE UN BALAYAGE MUET. 719 fichiers de production de la
// brique, hors `research/` et hors tag `research`, mesures le 2026-11-08.
const plancherFichiersBalayesHote = 650

// fautesDHote rend, pour une source, une ligne par litteral interdit et par ligne :
// `chemin:ligne : litteral`. Les litteraux admis pour ce fichier sont rendus a part, `admis`, pour que
// l'allowlist se verifie elle-meme.
func fautesDHote(rel string, blob []byte, admis map[string]bool) []string {
	var fautes []string
	for i, ligne := range bytes.Split(blob, []byte("\n")) {
		for _, lit := range litterauxDeLHote {
			if !bytes.Contains(ligne, []byte(lit)) {
				continue
			}
			if slices.Contains(hoteAdmis[rel], lit) {
				admis[rel+"\x00"+lit] = true
				continue
			}
			fautes = append(fautes, rel+":"+strconv.Itoa(i+1)+" : "+strconv.Quote(lit))
		}
	}
	return fautes
}

// releverHote balaye la production de la brique et rend les fautes, les couples (fichier, litteral
// admis) vus, et le nombre de fichiers balayes.
func releverHote(t *testing.T) ([]string, map[string]bool, int) {
	t.Helper()
	var fautes []string
	admis := map[string]bool{}
	fichiers := balayerLaProductionHorsResearch(t, []string{racineBriqueFilm}, func(rel string, blob []byte) {
		fautes = append(fautes, fautesDHote(rel, blob, admis)...)
	})
	sort.Strings(fautes)
	return fautes, admis, fichiers
}

func TestLaBriqueNeConnaitPasSonHote(t *testing.T) {
	fautes, _, fichiers := releverHote(t)
	if fichiers < plancherFichiersBalayesHote {
		t.Fatalf("balayage muet : %d fichiers de production vus sous %s, plancher %d. L'arborescence a "+
			"bouge : ce ratchet ne garde plus rien.", fichiers, racineBriqueFilm, plancherFichiersBalayesHote)
	}
	if len(fautes) > 0 {
		t.Fatalf("la production de la brique nomme son hote (%d) :\n  %s\n\n"+
			"Un litteral de l'application (variable LEVELUP_, dossier apps/go-api, ancienne adresse "+
			"internal/games/halo_infinite/film) ou un chemin absolu (C:\\, /opt/, /home/) n'a pas sa place dans "+
			"la brique, commentaires compris : la brique recoit ce qu'il lui faut en parametre, et ce qui "+
			"se decide chez l'hote se decrit chez l'hote. Reformuler le commentaire sans nommer l'hote, ou "+
			"passer la valeur en parametre. L'allowlist hoteAdmis est datee et fermee (revision/porte.go, "+
			"decision D-3 ; cadre historique du hachage, ADR 0041 D8) : y ajouter une entree N'EST PAS "+
			"une reponse.", len(fautes), strings.Join(fautes, "\n  "))
	}
}

// TestHoteAdmisNEstPasPerime : une entree de l'allowlist dont le litteral a disparu du fichier se retire.
func TestHoteAdmisNEstPasPerime(t *testing.T) {
	_, vus, _ := releverHote(t)
	var perimes []string
	for rel, lits := range hoteAdmis {
		for _, lit := range lits {
			if !vus[rel+"\x00"+lit] {
				perimes = append(perimes, fmt.Sprintf("%s : %s", rel, strconv.Quote(lit)))
			}
		}
	}
	if len(perimes) > 0 {
		sort.Strings(perimes)
		t.Fatalf("entree(s) de hoteAdmis sans objet, a retirer (le litteral n'est plus dans le fichier, "+
			"ou le fichier a bouge) :\n  %s", strings.Join(perimes, "\n  "))
	}
}

// TestLaBriqueNeConnaitPasSonHoteRougitSurSesDefauts : la regle mord sur chacun des six litteraux, dans
// le code comme dans un commentaire, et laisse passer un fichier de l'allowlist pour son litteral SEUL.
func TestLaBriqueNeConnaitPasSonHoteRougitSurSesDefauts(t *testing.T) {
	cas := []struct {
		nom, rel, source string
		veut             int
	}{
		{"variable d'environnement", racineBriqueFilm + "/replay/x.go", "package r\nvar v = os.Getenv(\"LEVELUP_X\")\n", 1},
		{"dossier de l'hote en commentaire", racineBriqueFilm + "/replay/x.go", "package r\n// voir apps/go-api/cmd\n", 1},
		{"ancienne adresse", racineBriqueFilm + "/replay/x.go", "package r\nconst c = \"internal/games/halo_infinite/film/x\"\n", 1},
		{"chemin Windows", racineBriqueFilm + "/replay/x.go", "package r\nconst c = `C:\\Halo`\n", 1},
		{"chemin serveur", racineBriqueFilm + "/replay/x.go", "package r\nconst c = \"/opt/levelup\"\n", 1},
		{"chemin personnel", racineBriqueFilm + "/replay/x.go", "package r\nconst c = \"/home/u\"\n", 1},
		{"source propre", racineBriqueFilm + "/replay/x.go", "package r\nconst c = \"levelup\"\n", 0},
		{"la porte pour son litteral", racineBriqueFilm + "/revision/porte.go", "package revision\nconst c = \"LEVELUP_UPDATE_X\"\n", 0},
		{"la porte pour un autre litteral", racineBriqueFilm + "/revision/porte.go", "package revision\n// apps/go-api\n", 1},
		{"un autre fichier de revision", racineBriqueFilm + "/revision/messages.go", "package revision\nconst c = \"LEVELUP_X\"\n", 1},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got := fautesDHote(c.rel, []byte(c.source), map[string]bool{})
			if len(got) != c.veut {
				t.Fatalf("fautes = %v, attendu %d", got, c.veut)
			}
		})
	}
}
