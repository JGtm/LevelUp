package killsummary

// rev_test.go — LE GATE DE [Rev] (ADR 0039, décision 5 ; doctrine ADR 0034 D-6 : une révision
// par chose qui peut changer).
//
// Les résumés écrits en base portent `summary_rev`. Si la règle change (grain, normalisation,
// compte, bornes de tranche) sans que Rev monte, les têtes déjà écrites passent pour fraîches et
// le rattrapage ne les réécrit jamais : les pages servent des résumés calculés par l'ancienne
// règle, sans erreur. Ce fichier fige le couple (révision, empreinte) :
//
//   - l'empreinte hache le FLUX DE JETONS des sources de production du paquet (commentaires et
//     mise en page écartés : une explication réécrite ne change aucune ligne produite) et les
//     valeurs des bornes de tranche, qui vivent dans le paquet domain ;
//   - toucher le paquet ou les bornes fait rougir le test ; le remettre au vert exige de DÉCIDER
//     si la sortie change — si oui, monter Rev (et recalculer le parc, ADR 0039 décision 5),
//     puis reporter le couple ci-dessous.
//
// Le mécanisme d'empreinte de sources des couches du film (`film/revision`) ne s'applique pas
// ici : il décrit les couches du décodeur et leur fermeture d'imports, pas un paquet d'analyse.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// Le couple figé. À reporter, après décision, depuis le message d'échec.
const (
	revFigee       = "ks1"
	empreinteFigee = "4f1c69700bc8dd31a0987e3516a16b91f4f36ad7e332243dd596956a19296d32"
)

// bornesParRevision : les bornes de tranche sous lesquelles chaque révision a été écrite. Une
// borne qui bouge sans révision neuve fait rougir TestRev_BornesDeTrancheLieesALaRevision.
var bornesParRevision = map[string][2]int{
	"ks1": {25, 50},
}

// empreinteDesSources hache, dans l'ordre des noms, le flux de jetons de chaque source de
// production du dossier courant, puis les bornes de tranche.
func empreinteDesSources(t *testing.T) (string, int) {
	t.Helper()
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	var noms []string
	for _, e := range entrees {
		n := e.Name()
		if !e.IsDir() && strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go") {
			noms = append(noms, n)
		}
	}
	sort.Strings(noms)
	if len(noms) == 0 {
		t.Fatal("aucune source de production : le gate ne hache rien")
	}
	h := sha256.New()
	for _, n := range noms {
		src, err := os.ReadFile(n)
		if err != nil {
			t.Fatalf("lecture %s : %v", n, err)
		}
		fmt.Fprintf(h, "fichier:%s\n%s", n, jetonsDe(t, n, src))
	}
	fmt.Fprintf(h, "bornes:%d,%d\n", domain.AssistTierLowMaxPct, domain.AssistTierMidMaxPct)
	return hex.EncodeToString(h.Sum(nil)), len(noms)
}

// jetonsDe rend le flux de jetons d'une source, un par ligne, sans commentaires.
func jetonsDe(t *testing.T, nom string, src []byte) string {
	t.Helper()
	fset := token.NewFileSet()
	var s scanner.Scanner
	s.Init(fset.AddFile(nom, -1, len(src)), src, func(pos token.Position, msg string) {
		t.Fatalf("%s : %s", pos, msg)
	}, 0)
	var b strings.Builder
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			return b.String()
		}
		fmt.Fprintf(&b, "%s %s\n", tok, strings.ReplaceAll(lit, "\r", ""))
	}
}

// TestRev_EmpreinteDesSourcesFigee : le couple (Rev, empreinte) est celui qui est figé.
func TestRev_EmpreinteDesSourcesFigee(t *testing.T) {
	empreinte, fichiers := empreinteDesSources(t)
	if Rev != revFigee {
		t.Fatalf("Rev = %q mais le couple figé porte %q : reporter revFigee = %q et "+
			"empreinteFigee = %q (%d fichiers), et ajouter %q à bornesParRevision",
			Rev, revFigee, Rev, empreinte, fichiers, Rev)
	}
	if empreinte != empreinteFigee {
		t.Fatalf("les sources du résumé (ou les bornes de tranche) ont changé sous la révision %q.\n"+
			"Si la sortie de Summarize change pour une seule passe : monter Rev, recalculer le parc "+
			"(ADR 0039, décision 5), puis reporter le couple.\nSinon (renommage sans effet) : "+
			"reporter seulement empreinteFigee.\nEmpreinte courante : %s (%d fichiers)",
			Rev, empreinte, fichiers)
	}
}

// TestRev_BornesDeTrancheLieesALaRevision : les bornes courantes sont celles de la révision
// courante. Changer AssistTierLowMaxPct / AssistTierMidMaxPct sans monter Rev fait rougir ce
// test : les tranches déjà écrites seraient celles des anciennes bornes.
func TestRev_BornesDeTrancheLieesALaRevision(t *testing.T) {
	bornes, ok := bornesParRevision[Rev]
	if !ok {
		t.Fatalf("aucune borne enregistrée pour la révision %q", Rev)
	}
	if bornes != [2]int{domain.AssistTierLowMaxPct, domain.AssistTierMidMaxPct} {
		t.Fatalf("bornes de tranche %d/%d, la révision %q a été écrite sous %d/%d : monter Rev",
			domain.AssistTierLowMaxPct, domain.AssistTierMidMaxPct, Rev, bornes[0], bornes[1])
	}
}
