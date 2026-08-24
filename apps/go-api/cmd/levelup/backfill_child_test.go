package main

// backfill_child_test.go — le PROTOCOLE parent/enfant.
//
// Ce que ces tests tiennent : un code de sortie inconnu ne doit JAMAIS passer pour un
// succes, et la ligne de protocole du pic memoire ne doit JAMAIS fuir dans le journal de
// l'operateur. Ce sont les deux proprietes dont depend la lecture du recap d'une passe.

import (
	"os"
	"strings"
	"testing"
)

func TestIssuePourCode_MappingComplet(t *testing.T) {
	cas := []struct {
		nom  string
		code int
		veut issueEnfant
	}{
		{"succes", codeEnfantOK, issueOK},
		{"carte hors catalogue", codeEnfantHorsCatalogue, issueHorsCatalogue},
		{"erreur de decodage", codeEnfantErreurDecodage, issueErreurDecodage},
		{"echec de preparation", codeEnfantPreparation, issuePreparation},
		{"plafond memoire", codeEnfantMemoire, issueMemoire},
		// Les codes que le protocole ne possede PAS. 1 = erreur rendue par main,
		// 2 = flag.ExitOnError et `fatal error` du runtime (le crash du 2026-08-20),
		// -1 = tue par l'OS. Aucun ne doit etre lu comme une issue metier.
		{"code 1 (erreur rendue par main)", 1, issueMortSubite},
		{"code 2 (fatal error du runtime)", 2, issueMortSubite},
		{"tue par l OS", -1, issueMortSubite},
		{"code inconnu", 42, issueMortSubite},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := issuePourCode(c.code); got != c.veut {
				t.Fatalf("issuePourCode(%d) = %v, veut %v", c.code, got, c.veut)
			}
		})
	}
}

// TestIssuePourCode_CodesDistincts : deux issues ne peuvent pas partager un code, sinon la
// ventilation du recap est un mensonge.
func TestIssuePourCode_CodesDistincts(t *testing.T) {
	codes := []int{codeEnfantOK, codeEnfantHorsCatalogue, codeEnfantErreurDecodage,
		codeEnfantPreparation, codeEnfantMemoire}
	vus := map[int]bool{}
	for _, c := range codes {
		if vus[c] {
			t.Fatalf("code %d attribue deux fois", c)
		}
		vus[c] = true
	}
	// Les codes du protocole ne doivent pas empieter sur ceux du runtime (1 et 2).
	for _, c := range codes[1:] {
		if c == 1 || c == 2 {
			t.Fatalf("le code %d collide avec un code du runtime/flag", c)
		}
	}
}

func TestLirePicMemoire(t *testing.T) {
	if v, ok := lirePicMemoire(marqueurPicMemoire + "123456"); !ok || v != 123456 {
		t.Fatalf("ligne de protocole = (%d, %v), veut (123456, true)", v, ok)
	}
	if _, ok := lirePicMemoire("  " + marqueurPicMemoire + "7 "); !ok {
		t.Fatal("la ligne doit etre reconnue malgre les espaces")
	}
	for _, ligne := range []string{
		"", "une ligne de journal ordinaire",
		marqueurPicMemoire + "pas-un-nombre",
		"prefixe " + marqueurPicMemoire + "12",
	} {
		if _, ok := lirePicMemoire(ligne); ok {
			t.Fatalf("ligne %q reconnue a tort comme protocole", ligne)
		}
	}
}

// TestRelayer_IntercepteLeProtocoleEtRelaieLeReste : le journal de l'enfant arrive INTACT
// chez le parent, moins les lignes de protocole.
func TestRelayer_IntercepteLeProtocoleEtRelaieLeReste(t *testing.T) {
	var journal strings.Builder
	r := &runnerEnfant{sortie: &journal}

	source := strings.NewReader(strings.Join([]string{
		"INFO decodage demarre",
		marqueurPicMemoire + "2048",
		"  abc : 21 tracks, 264225 octets (fo11_blank)",
	}, "\n") + "\n")

	pic := r.relayer(source)
	if pic != 2048 {
		t.Fatalf("pic = %d, veut 2048", pic)
	}
	got := journal.String()
	if strings.Contains(got, marqueurPicMemoire) {
		t.Fatalf("la ligne de protocole a fuit dans le journal :\n%s", got)
	}
	for _, veut := range []string{"INFO decodage demarre", "21 tracks"} {
		if !strings.Contains(got, veut) {
			t.Fatalf("le journal a perdu %q :\n%s", veut, got)
		}
	}
}

// TestEnvEnfant_ImposeLaRacine : le parent impose SA racine, une seule fois. Sans cela deux
// processus de la meme passe pourraient ecrire dans deux arborescences differentes.
func TestEnvEnfant_ImposeLaRacine(t *testing.T) {
	t.Setenv(cleRepoRoot, "C:/ancienne/racine")
	env := envEnfant("C:/racine/voulue")

	var vues []string
	for _, kv := range env {
		if strings.HasPrefix(strings.ToUpper(kv), cleRepoRoot+"=") {
			vues = append(vues, kv)
		}
	}
	if len(vues) != 1 {
		t.Fatalf("%s apparait %d fois, veut exactement 1 : %v", cleRepoRoot, len(vues), vues)
	}
	if vues[0] != cleRepoRoot+"=C:/racine/voulue" {
		t.Fatalf("racine imposee = %q", vues[0])
	}
	// Le reste de l'environnement doit survivre.
	if len(env) < len(os.Environ()) {
		t.Fatalf("l environnement a perdu des entrees : %d < %d", len(env), len(os.Environ()))
	}
}

func TestListeDrapeau_Repetable(t *testing.T) {
	var l listeDrapeau
	for _, v := range []string{"Live Fire", "Aquarius, la carte"} {
		if err := l.Set(v); err != nil {
			t.Fatalf("Set(%q): %v", v, err)
		}
	}
	if len(l) != 2 || l[0] != "Live Fire" || l[1] != "Aquarius, la carte" {
		t.Fatalf("liste = %v — la repetition doit preserver les libelles a virgule", l)
	}
}
