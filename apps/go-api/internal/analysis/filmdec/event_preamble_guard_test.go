package filmdec

// event_preamble_guard_test.go — LE PREAMBULE D'EVENEMENT ET LA TABLE DES DOMAINES N'ONT QU'UN
// SEUL LECTEUR DE PRODUCTION (lot E, item E.3 du PLAN_V2_REJEU_FILM, 2026-09-05).
//
// # CE QUE CE FICHIER EMPECHE, ET POURQUOI IL EXISTE
//
// La grammaire du preambule de 9 bits — `[config(1)][continuation(1)][R(7) type]` — etait ecrite
// SIX fois en ligne dans ce paquet, sous DEUX conventions (`Skip(1)` + `ReadBit()` d'un cote,
// `Skip(2)` de l'autre), et la table des largeurs de reference par domaine existait en TROIS
// exemplaires de production. Deux de ces copies portaient `3: 8` la ou la mesure du siege dit 7
// (`event_list.go`, oracle du 2026-09-02 : 5/6 d'accord a 7 bits contre 0/6 a 8 bits). Rien de
// faux n'etait servi — aucun chemin de production ne lisait le domaine 3 hors de `boardRefs`,
// qui portait deja la valeur mesuree — mais le prochain decodeur qui aurait eu besoin du
// domaine 3 avait deux chances sur trois de prendre la copie perimee et de decaler d'un bit tout
// le corps de l'evenement.
//
// CLAUDE.md regle 6 : « a la 3e copie, centraliser dans un helper ET ajouter un garde-rail (test
// grep) qui interdit l'ancien litteral — une factorisation sans garde-rail re-diverge ». C'est ce
// fichier. Il ne mesure pas un comportement : il mesure que la SOURCE reste unique.
//
// # CE QU'IL NE COUVRE PAS, ET C'EST DELIBERE
//
// LES SOURCES DE TEST SONT HORS PORTEE. Deux instruments de recherche portent encore leur propre
// table (`bpkDomWidths` dans biped_pickup_research_test.go, `r7DomWidth` dans
// r7_grammaire_research_test.go), et l'un d'eux LIT le domaine 3 (type 8, `{2,3,7}`) a 8 bits.
// Les migrer changerait une largeur DANS UN INSTRUMENT DE MESURE DATE, ce que le lot E-I
// (« comportement strictement identique ») s'interdit. Le fait est consigne au journal du lot ;
// la portee du garde-rail s'etendra quand ces deux instruments seront traites.
//
// RETRAIT : jamais tant que `filmdec` porte la grammaire de la liste d'evenements. Si le
// preambule devait un jour se lire autrement selon le type d'evenement, c'est `readPacketHead`
// qui prendrait le parametre — pas une septieme copie.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestRefDomWidthEstLaSeuleTable — la table des domaines rend les largeurs mesurees, dom3 = 7
// compris, et un domaine hors table rend 0 (meme semantique que les cartes qu'elle remplace :
// une cle absente y valait le zero du type, donc `ReadBits(0)`).
func TestRefDomWidthEstLaSeuleTable(t *testing.T) {
	attendu := map[int]uint{0: 13, 1: 13, 2: 8, 3: 7, 4: 9, 5: 8, 6: 9, 7: 13, 8: 13}
	for dom, want := range attendu {
		if got := refDomWidth(dom); got != want {
			t.Errorf("refDomWidth(%d) = %d, attendu %d", dom, got, want)
		}
	}
	for _, dom := range []int{-1, 9, 12, 255} {
		if got := refDomWidth(dom); got != 0 {
			t.Errorf("refDomWidth(%d) = %d : un domaine hors table doit rendre 0 (zero bit lu)", dom, got)
		}
	}
	// Le domaine 3 est LA valeur qui distingue la table canonique des deux copies supprimees.
	if refDomWidth(3) != dom3RefWidth || dom3RefWidth != 7 {
		t.Fatalf("le domaine 3 vaut %d : la mesure du siege dit 7 (event_list.go), la prose de "+
			"l'executable disait 8 — c'est la MESURE qui fait foi", refDomWidth(3))
	}
	// Les constantes nommees et la table ne peuvent pas diverger.
	if refDomWidth(7) != dom7RefWidth || refDomWidth(2) != dom2RefWidth || refDomWidth(4) != dom4RefWidth {
		t.Fatal("refDomWidth diverge des constantes nommees qu'elle compose")
	}
	// La largeur que les instruments de translocation calculent a la compilation suit la table.
	if uint(translocRefWidth) != refDomWidth(translocRefDomain) {
		t.Errorf("translocRefWidth = %d mais refDomWidth(%d) = %d",
			translocRefWidth, translocRefDomain, refDomWidth(translocRefDomain))
	}
}

// tableDomainesRecopiee reconnait une table domaine -> largeur ecrite en litteral : au moins
// quatre paires `<chiffre>: <nombre>` separees par des virgules. C'est la forme exacte des deux
// copies de production supprimees (`lot1RefDomWidths`, `zoomRefWidth`).
var tableDomainesRecopiee = regexp.MustCompile(`(?:\b\d\s*:\s*\d{1,2}\s*,\s*){3,}\d\s*:\s*\d{1,2}`)

// TestAucuneTableDeDomainesRecopiee — aucune source de PRODUCTION du paquet ne reecrit la table
// des domaines.
func TestAucuneTableDeDomainesRecopiee(t *testing.T) {
	for _, nom := range sourcesDeProductionDuPaquet(t) {
		for i, ligne := range lignesDeCode(t, nom) {
			if ligne != "" && tableDomainesRecopiee.MatchString(ligne) {
				t.Errorf("%s:%d recopie une table domaine -> largeur :\n\t%s\n"+
					"La seule table du paquet est `refDomWidth` (event_list.go). Une copie "+
					"re-diverge : les deux precedentes portaient `3: 8` contre la valeur "+
					"MESUREE 7.", nom, i+1, ligne)
			}
		}
	}
}

// sautDePreambule reconnait le saut de tete d'un preambule d'evenement : `Skip(1)` (convention
// « bit de config saute, continuation testee ») ou `Skip(2)` (convention « les deux sautes »).
var sautDePreambule = regexp.MustCompile(`\.Skip\(\s*[12]\s*\)`)

// lectureDuType reconnait la lecture du R(7) de type.
var lectureDuType = regexp.MustCompile(`\.ReadBits\(\s*(?:7|eventTypeBits)\s*\)`)

// preambuleFenetre : nombre de lignes de code apres le saut ou la lecture du type doit tomber
// pour que la sequence soit reconnue comme un preambule. Trois suffisent : les six copies
// d'origine tenaient toutes en trois lignes (saut, test de continuation, lecture du type).
const preambuleFenetre = 3

// TestPreambuleNaQuUnSeulLecteur — la SEQUENCE du preambule (saut de tete puis R(7) de type) ne
// s'ecrit que dans `readPacketHead`. Les six copies en ligne d'avant le 2026-09-05 sont
// interdites de retour.
//
// LE MOTIF EST LA SEQUENCE, PAS LE `ReadBits(7)` SEUL : le paquet lit legitimement 7 bits a une
// vingtaine d'endroits (dequantifications, champs de composant). Ce qui est interdit, c'est de
// les lire JUSTE APRES un saut de un ou deux bits de tete.
func TestPreambuleNaQuUnSeulLecteur(t *testing.T) {
	for _, nom := range sourcesDeProductionDuPaquet(t) {
		if nom == "event_list.go" {
			continue // la declaration de readPacketHead, et elle seule
		}
		lignes := lignesDeCode(t, nom)
		for i, ligne := range lignes {
			if ligne == "" || !sautDePreambule.MatchString(ligne) {
				continue
			}
			for j := i + 1; j <= i+preambuleFenetre && j < len(lignes); j++ {
				if lectureDuType.MatchString(lignes[j]) {
					t.Errorf("%s:%d-%d recopie le preambule d'evenement :\n\t%s\n\t%s\n"+
						"Le preambule de 9 bits se lit par `readPacketHead` (event_list.go). "+
						"Six copies en ligne, sous deux conventions, ont ete ramenees a ce "+
						"lecteur unique le 2026-09-05 (lot E, item E.3).",
						nom, i+1, j+1, ligne, lignes[j])
					break
				}
			}
		}
	}
}

// sourcesDeProductionDuPaquet rend les sources Go non-test du paquet, relatives au repertoire
// courant (celui du paquet quand `go test` s'execute).
func sourcesDeProductionDuPaquet(t *testing.T) []string {
	t.Helper()
	noms, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("listage du paquet : %v", err)
	}
	var out []string
	for _, nom := range noms {
		if !strings.HasSuffix(nom, "_test.go") {
			out = append(out, nom)
		}
	}
	if len(out) == 0 {
		t.Fatal("aucune source de production trouvee : le garde-rail ne mesure plus rien")
	}
	return out
}

// lignesDeCode rend les lignes du fichier, celles de commentaire et les vides rendues comme la
// chaine vide : la prose a le droit de citer une table ou un preambule.
func lignesDeCode(t *testing.T, nom string) []string {
	t.Helper()
	src, err := os.ReadFile(nom)
	if err != nil {
		t.Fatalf("lecture de %s : %v", nom, err)
	}
	brutes := strings.Split(string(src), "\n")
	out := make([]string, len(brutes))
	for i, ligne := range brutes {
		nue := strings.TrimSpace(ligne)
		if nue == "" || strings.HasPrefix(nue, "//") {
			continue
		}
		out[i] = nue
	}
	return out
}
