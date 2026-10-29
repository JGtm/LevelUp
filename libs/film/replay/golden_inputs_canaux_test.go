package replay

// golden_inputs_canaux_test.go — L INVENTAIRE DU TYPE, TENU PAR LE COMPILATEUR ET LA REFLEXION.
//
// Le CODEC lui-meme est passe en production au lot 4.1.1-a (`filmfacts_canaux.go` et ses freres) :
// il n est plus un instrument de fixture mais la porte d ecriture et de lecture des faits
// persistes par film. Ce qui reste ici est ce qui doit rester un test — la garde qui exige que
// TOUTE feuille de [FilmInputs] soit transportee, ou nommee comme non transportee.

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// TestCodecCouvreFilmInputs : TOUTE FEUILLE de [FilmInputs] fait l aller-retour par le fichier de
// faits, A L IDENTIQUE, ou figure dans [feuillesNonTransportees] avec sa preuve.
//
// # CE QU IL FERME, ET IL A DEJA COUTE
//
// Le codec porte le type de la PRODUCTION depuis le lot 1.0 : un canal ajoute a l etage de
// balayage apparait donc tout seul dans [FilmFacts], et l assemblage le lira — mais le CODEC, lui,
// ne l apprend pas. Les faits relus rendraient alors un calque vide la ou la production en publie
// un plein, exactement le defaut que la decouverte D9 avait mesure sur sept builds.
// `TestGoldenInputsFidelite` l attrape, mais il EXIGE LE CACHE DE FILMS : il saute en CI. Ce
// test-ci, lui, ne lit aucun octet de film et tourne partout.
//
// # LE CRITERE A CHANGE DEUX FOIS, ET CHAQUE FOIS PARCE QU UN DEFAUT ETAIT PASSE
//
//   - Jusqu au 2026-09-18, le temoin ne remplissait que le PREMIER champ de chaque structure
//     imbriquee : `types.EquipmentCreation` voyageait sur sept champs sur vingt et passait.
//   - Jusqu au 2026-09-28, un champ etait « transporte » des qu UNE de ses feuilles revenait non
//     nulle : `MovementStateStats.JumpEpisodes/JumpsDerived` (publies en `coverage.stances`) sont
//     sortis du fichier par ce trou, et le sondage du meme jour en a trouve d autres
//     (`Fire[].Short/Bloc`, `InventoryDeltas[].Ammo`, `AbilityRanks[].Counter`,
//     `WeaponChanges[].Low`, l observateur du cadre des morts de vehicule).
//
// DESORMAIS : chaque feuille recoit une valeur non nulle DISTINCTE de ses voisines (un compteur
// recopie dans le suivant ne passe pas), et chaque feuille est comparee A L OCTET apres
// l aller-retour. Une feuille perdue n a qu une issue : [feuillesNonTransportees], ou chaque entree
// porte la preuve que l assemblage ne la lit pas. Deux feuilles sont ACCORDEES avant la mesure,
// parce que le codec les derive ou les borne par construction (cf. [accorderLeTemoin]).
//
// LA TABLE NE GARDE QUE CE QUI EST VRAIMENT PERDU : une exception que le temoin voit faire
// l aller-retour est morte, et ce test la refuse.
func TestCodecCouvreFilmInputs(t *testing.T) {
	entry := goldenEntryPourTest(t)
	g := &FilmFacts{Film: goldenFilm, MapModule: entry.Module, AxisW: entry.AxisWidths}
	var w temoinDeFeuilles
	w.remplir("FilmInputs", reflect.ValueOf(&g.FilmInputs).Elem(), false)
	if w.n < 600 {
		t.Fatalf("%d feuille(s) remplie(s) sur FilmInputs : la reflexion ne mesure plus rien", w.n)
	}
	for _, f := range w.nonRemplis {
		t.Errorf("feuille %s : forme que le temoin ne sait pas remplir — etendre temoinDeFeuilles", f)
	}
	accorderLeTemoin(g, entry)
	ecarts := ecartsDeLAllerRetour(g, allerRetourParLeFichier(t, g, entry))
	for _, e := range ecarts.nonNommes() {
		t.Errorf("FEUILLE NON TRANSPORTEE : %s.\n"+
			"Un artefact rejoue depuis les faits la perd si l assemblage la lit. DEUX PLACES "+
			"LEGITIMES pour la porter : le blob des entrees, a la QUEUE de la section de sa famille "+
			"(monter `filmFactsMagic` DANS LE MEME COMMIT, et re-decoder les huit fixtures de "+
			"testdata/), ou le complement de la section 1 (`encodeGardesDeMode`, monter "+
			"`SchemaDesFaits`). La nommer dans `feuillesNonTransportees` est une DECISION : releve "+
			"des usages au verificateur de types, puis S8 rejoue.", e)
	}
	exercees := ecarts.nommes()
	var mortes []string
	for c := range feuillesNonTransportees {
		if !exercees[c] {
			mortes = append(mortes, c)
		}
	}
	sort.Strings(mortes)
	if len(mortes) > 0 {
		t.Errorf("exception(s) MORTE(S) — le fichier de faits les transporte (ou le temoin ne les "+
			"remplit plus) : les retirer de feuillesNonTransportees :\n  %s", strings.Join(mortes, "\n  "))
	}
}
