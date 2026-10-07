package replaydiff

// polarite_ratchet_test.go — LA TABLE DES POLARITES EST TOTALE, ET ELLE LE RESTE (lot R4).
//
// L'inventaire des feuilles numeriques de la couverture est derive de la forme du document
// (`inventaire_couverture_test.go`) ; ce fichier exige, dans les deux sens :
//
//	toute feuille de l'inventaire a une polarite DECLAREE (entree exacte, `m.*` pour une map) ;
//	toute entree de la table designe une feuille de l'inventaire, ou une cle nommee d'une map
//	qu'il porte (`m.unknown`, `m.*/unknown`) ; une entree HERITEE, elle, n'y est plus ;
//	aucune feuille n'est declaree deux fois.
//
// MUTATION PAR COPIE (TestRatchetPolarite_FeuilleAjouteeNonClassee) : la forme est copiee, un
// compteur y est ajoute a la couverture, et le controle doit le nommer comme non classe.

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// bilanTable : ce que le controle reproche a la table face a un inventaire.
type bilanTable struct {
	nonClassees, orphelines, heriteesVivantes, doublons []string
}

// controlerTable confronte les lignes de table (courantes et heritees) a un inventaire.
func controlerTable(inventaire []string, courantes, heritees []blocPolarites) bilanTable {
	var b bilanTable
	idx := construireIndex(append(append([]blocPolarites{}, courantes...), heritees...))
	b.doublons = idx.doublons
	dansInventaire := map[string]bool{}
	var jokers []string
	for _, f := range inventaire {
		dansInventaire[f] = true
		if strings.Contains(f, jokerCle) {
			jokers = append(jokers, f)
		}
		if _, ok := idx.exactes[f]; !ok {
			b.nonClassees = append(b.nonClassees, f)
		}
	}
	for cle := range construireIndex(courantes).exactes {
		if !dansInventaire[cle] && !cleDeMapDeclaree(cle, jokers) {
			b.orphelines = append(b.orphelines, cle)
		}
	}
	for cle := range construireIndex(heritees).exactes {
		if dansInventaire[cle] || cleDeMapDeclaree(cle, jokers) {
			b.heriteesVivantes = append(b.heriteesVivantes, cle)
		}
	}
	sort.Strings(b.orphelines)
	sort.Strings(b.heriteesVivantes)
	return b
}

// cleDeMapDeclaree dit si `cle` nomme une cle d'une map que l'inventaire porte (`m.*`).
func cleDeMapDeclaree(cle string, jokers []string) bool {
	for _, j := range jokers {
		prefixe := strings.TrimSuffix(j, jokerCle)
		if prefixe != j && strings.HasPrefix(cle, prefixe) && len(cle) > len(prefixe) {
			return true
		}
	}
	return false
}

func inventaireOuEchec(t *testing.T, path string) []string {
	t.Helper()
	inv, err := inventaireCouverture(path)
	if err != nil {
		t.Fatalf("inventaire de la couverture : %v", err)
	}
	return inv
}

// TestRatchetPolarite_TableTotale — LE RATCHET. Une feuille ajoutee a la couverture sans
// polarite, une entree qui ne designe plus rien, une feuille heritee revenue dans la forme, ou
// une feuille declaree deux fois : ROUGE, avec la liste.
func TestRatchetPolarite_TableTotale(t *testing.T) {
	inv := inventaireOuEchec(t, cheminFormeDocument)
	if len(inv) < 400 {
		t.Fatalf("inventaire de %d feuilles : la lecture de la forme a perdu la couverture", len(inv))
	}
	b := controlerTable(inv, tablePolarites, polaritesHeritees)
	for _, f := range b.nonClassees {
		t.Errorf("feuille SANS POLARITE : %s — la classer dans polarite_table.go (echec / succes / "+
			"neutre / telemetrie)", f)
	}
	for _, f := range b.orphelines {
		t.Errorf("entree ORPHELINE de la table : %s — la forme ne la porte plus ; la retirer, ou "+
			"la deplacer dans polaritesHeritees avec sa date de retrait", f)
	}
	for _, f := range b.heriteesVivantes {
		t.Errorf("entree HERITEE encore portee par la forme : %s", f)
	}
	for _, f := range b.doublons {
		t.Errorf("feuille declaree DEUX FOIS : %s", f)
	}
}

// TestRatchetPolarite_FeuilleAjouteeNonClassee — MUTATION PAR COPIE. La forme est copiee dans
// un repertoire de test ; on y ajoute un compteur a `VehicleCoverage` et une map a
// `SeatCoverage`. Le controle doit nommer EXACTEMENT ces deux feuilles comme non classees.
func TestRatchetPolarite_FeuilleAjouteeNonClassee(t *testing.T) {
	brut, err := os.ReadFile(cheminFormeDocument)
	if err != nil {
		t.Fatal(err)
	}
	ajouts := map[string]string{
		"VehicleCoverage": `  ZzzCompteurNeuf json:"zzzCompteurNeuf" int`,
		"SeatCoverage":    `  ZzzParCause json:"zzzParCause,omitempty" map[string]int`,
	}
	var sortie []string
	for _, ligne := range strings.Split(string(brut), "\n") {
		sortie = append(sortie, ligne)
		if a, ok := ajouts[strings.TrimRight(ligne, "\r")]; ok {
			sortie = append(sortie, a)
			delete(ajouts, strings.TrimRight(ligne, "\r"))
		}
	}
	if len(ajouts) != 0 {
		t.Fatalf("types introuvables dans la forme : %v", ajouts)
	}
	copie := filepath.Join(t.TempDir(), "document_shape.golden")
	if err := os.WriteFile(copie, []byte(strings.Join(sortie, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	b := controlerTable(inventaireOuEchec(t, copie), tablePolarites, polaritesHeritees)
	sort.Strings(b.nonClassees)
	attendu := []string{"coverage.seats.zzzParCause.*", "coverage.vehicles.zzzCompteurNeuf"}
	if strings.Join(b.nonClassees, ",") != strings.Join(attendu, ",") {
		t.Fatalf("non classees : %v, attendu %v", b.nonClassees, attendu)
	}
	if len(b.orphelines)+len(b.heriteesVivantes)+len(b.doublons) != 0 {
		t.Fatalf("la mutation ne devait rien reprocher d'autre : %+v", b)
	}

	// Et l'inverse : une entree de table qui ne designe rien est orpheline.
	fantome := []blocPolarites{{Blocs: []string{"coverage.vehicles."}, Echecs: "zzzDisparu"}}
	b = controlerTable(inventaireOuEchec(t, cheminFormeDocument),
		append(append([]blocPolarites{}, tablePolarites...), fantome...), polaritesHeritees)
	if len(b.orphelines) != 1 || b.orphelines[0] != "coverage.vehicles.zzzDisparu" {
		t.Fatalf("orphelines : %v, attendu [coverage.vehicles.zzzDisparu]", b.orphelines)
	}
}

// TestRatchetPolarite_ComptesParClasse fige les comptes de la table (inventaire courant), pour
// qu'une reclassification se voie au diff du test et pas seulement a celui de la table.
func TestRatchetPolarite_ComptesParClasse(t *testing.T) {
	inv := inventaireOuEchec(t, cheminFormeDocument)
	comptes := map[Polarite]int{}
	for _, f := range inv {
		comptes[polarites().lire(f)]++
	}
	t.Logf("inventaire %d feuilles : echec=%d succes=%d neutre=%d telemetrie=%d inconnue=%d",
		len(inv), comptes[PolariteEchec], comptes[PolariteSucces], comptes[PolariteNeutre],
		comptes[PolariteTelemetrie], comptes[PolariteInconnue])
	attendu := map[Polarite]int{PolariteEchec: 177, PolariteSucces: 172, PolariteNeutre: 159,
		PolariteTelemetrie: 2}
	for p, n := range attendu {
		if comptes[p] != n {
			t.Errorf("classe %v : %d feuilles, attendu %d (reclassification ? mettre a jour ce compte "+
				"dans le meme commit que la table)", p, comptes[p], n)
		}
	}
	if comptes[PolariteInconnue] != 0 {
		t.Fatalf("%d feuilles sans polarite", comptes[PolariteInconnue])
	}
}
