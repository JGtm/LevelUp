package replay

// golden_inputs_budget_test.go — CE QUE LE JEU DE FIXTURES D ENTREES PESE, ET SON PLAFOND.
//
// # POURQUOI UN PLAFOND, ET PAS SEULEMENT UNE MESURE (revue R1 du lot 1.0, exigence R1-6)
//
// Les huit `inputs_*.bin.gz` sont VERSIONNES : chaque montee de la magie du codec en depose un
// jeu complet de plus dans l historique du depot. Une taille qu on se contente de mesurer derive
// d un lot a l autre sans que personne ne la decide — le lot 1.0 vient d en faire la
// demonstration, en passant de 10 323 769 a 11 044 446 octets compresses (+7,0 %) sans qu aucune
// porte ne sonne. Un plafond force la decision (porter moins de canaux, moins de builds) le jour
// ou il est atteint, au lieu de la decouvrir dans un `git clone` de plus en plus long.
//
// LE PLAFOND NE SE RELEVE PAS PAR REFLEXE. Il se releve par DECISION ECRITE, datee, avec la
// raison — exactement comme celui des fixtures de contrat (`contract_fixtures_budget_test.go`).
// Un test qui rougit dit qu une question se pose, pas qu il faut changer sa constante.

import (
	"os"
	"testing"
)

// goldenInputsBudget : LE PLAFOND DU JEU ENTIER, en octets compresses.
//
// POSE LE 2026-09-14 (lot 1.0) A 12 MIO, RELEVE A 24 MIO LE 2026-09-18 (cf. ci-dessous). UNE
// SEULE MESURE FAIT FOI, celle que ce test lit sur le disque : **22 099 969 octets** compresses
// pour les huit fixtures, soit 21,08 Mio. Il reste 3 054 175 octets libres, soit 12 % du plafond. (Le commentaire d origine
// citait TROIS totaux differents pour une seule mesure — un d avant regeneration, un du plan, un
// mesure : revue R2, constat R2-2.)
//
// LE CHIFFRE AVAIT DEJA DERIVE DE 4 754 OCTETS, ET LE TEST RESTAIT VERT : le commentaire
// annoncait 11 044 446 o quand le disque en portait 11 049 200 (mesure du 2026-09-17, lot 4.1.1-b
// — le test n assertit que le plafond de 12 Mio, donc rien ne relisait le total ecrit ici).
// Corrige. C est la deuxieme fois que ce commentaire derive : un total recopie a la main dans un
// commentaire n a pas de gardien, et le seul gardien possible est la commande qui le remesure
// (`go test -run GoldenInputsTiennent -v`).
//
// LE LOT 4.1.1-b N Y A PAS TOUCHE, ET C EST UNE DECISION ECRITE : les quatre canaux gardes par
// l appelant (`FlagMarks`, `ZoneReads`, `ZoneScanned`, `BombReads`) entrent au FICHIER DE FAITS
// (`filmfacts_fichier.go`, section 1) et non dans ce blob-ci, parce qu un fixture ne fournit
// JAMAIS de garde de mode : les y mettre aurait exige de re-decoder huit films pour ajouter huit
// suites de zeros. Les huit fixtures sont donc INCHANGEES a l octet.
//
// HISTORIQUE, CHAQUE CHIFFRE AVEC SA BASE : 10 849 119 o a l origine du jeu par build
// (lot 0.A.2) ; 18 656 453 o a l etape flottants du lot 0.D.3 bis, revenus a 10 337 463 o en
// quanta ; 10 323 769 o a la cloture de 0.D ; 11 044 446 o depuis que le fixture porte les six
// canaux qui manquaient et le roster de la feuille (lot 1.0). Soit +7,0 % contre la cloture de
// 0.D, et +1,8 % contre le jeu d origine.
//
// LA MARGE EST VOULUE ETROITE : un canal de plus se voit. Elle n est PAS la pour absorber un
// build supplementaire — un neuvieme build est precisement la decision que ce plafond existe
// pour rendre explicite.
// RELEVE A 16 MIO LE 2026-09-18 (lot 4.1.3), PAR DECISION ECRITE — c'est ce que ce plafond
// existe pour forcer, et la question qu'il a posee avait une bonne reponse.
//
// # CE QUI A GROSSI, ET POURQUOI ON PAIE
//
// Le codec CESSAIT DE PERDRE. La v23 porte les pistes d'objets du monde en float32 EXACT (elles
// etaient arrondies au centimetre, et le document publie `groundWeapons[].x` brut), le record de
// creation ENTIER (les munitions de l'arme au sol tombaient, avec la reference d'entite,
// l'identifiant de capacite, le masque et trois mots MPP sur quatre) et les denominateurs du
// balayage entiers. Le gate S8 a mesure la perte sur dix films avant ce lot.
//
// PRIX MESURE, EN DEUX TEMPS ET LES DEUX SONT ECRITS :
//
//	11 049 200 -> 14 072 829 (+27,4 %)  les pistes d'objets du monde en float32 exact, le record
//	                                    de creation entier, les denominateurs entiers
//	14 072 829 -> 22 099 969 (+57,0 %)  les TREIZE champs de direction d'une position (le CAP des
//	                                    vehicules en sort : 4 602 echantillons sur `11de8353`) et
//	                                    les pistes de PROJECTILE passees a l'exact elles aussi —
//	                                    le centimetre y perdait le SIGNE de tout ce qui vit sous
//	                                    le demi-centimetre (`-0` contre `0`, les trois derniers
//	                                    octets d'ecart du gate)
//
// TOTAL 11 049 200 -> 22 099 969, exactement DEUX FOIS le jeu d'origine. C'est le prix d'un codec
// dont `TestGoldenInputsFidelite` prouve, sur huit films reels, que l'ARTEFACT est identique a
// l'octet des deux cotes — ce qu'aucune version precedente ne pouvait dire.
//
// # L'ALTERNATIVE, ET POURQUOI ELLE EST ECARTEE
//
// On pourrait ne porter que ce que l'assemblage LIT. C'est ECARTE COMME REGLE, parce que c'est
// exactement le raisonnement qui a produit le defaut : le codec ne portait « que ce que
// l'assemblage consomme » d'apres un jugement fait a la main, et ce jugement etait FAUX sur
// quatre familles pendant tout le chantier.
//
// UNE SEULE OMISSION SUBSISTE, ET ELLE N'EST PAS UN JUGEMENT : `componentDirs.MaskBits`, un
// uint64 par position, coutait 7,4 Mio a lui seul (le jeu montait a 21,4 Mio avant qu'on le
// retire, contre 20,4 apres). Il est omis parce que `TestGoldenInputsFidelite` PROUVE, sur huit
// films reels et sur l'artefact serialise, que le document n'en depend pas. Si un calque venait a
// le lire, ce gate rougirait — c'est la difference entre une omission mesuree et une omission
// supposee.
//
// LA MARGE RESTE ETROITE, ET C'EST VOULU : 2 704 387 octets libres, 16 % du plafond. Un neuvieme
// build ne passe pas sans une nouvelle decision.
const goldenInputsBudget = 24 << 20

// TestGoldenInputsTiennentDansLeBudget : le jeu entier tient-il sous le plafond, et combien pese
// chaque fixture ?
//
// LA MESURE EST LA POUR ETRE LUE (`go test -run GoldenInputsTiennent -v`) : c est elle que le
// plan consigne au journal des gates.
func TestGoldenInputsTiennentDansLeBudget(t *testing.T) {
	builds := goldenBuilds()
	if len(builds) == 0 {
		t.Fatal("aucun build a la table : le budget ne mesure plus rien")
	}
	total := 0
	for _, b := range builds {
		info, err := os.Stat(b.inputsPath())
		if err != nil {
			t.Fatalf("fixture d entrees absent (%s) : %v — regenerer avec -update et "+
				miniFilmCacheEnv, b.inputsPath(), err)
		}
		total += int(info.Size())
		t.Logf("%-10s %-10s %9d octets", b.Build, b.Short8, info.Size())
	}
	t.Logf("TOTAL %d fixture(s) : %d octets (%.2f Mio), plafond %d octets (%.0f Mio)",
		len(builds), total, float64(total)/(1<<20),
		goldenInputsBudget, float64(goldenInputsBudget)/(1<<20))
	if total > goldenInputsBudget {
		t.Errorf("le jeu de fixtures d entrees pese %d octets (%.2f Mio), au-dela du plafond de "+
			"%d octets (%.0f Mio) : couper (moins de canaux portes, moins de builds) ou relever "+
			"le plafond par decision ECRITE et datee, jamais par reflexe",
			total, float64(total)/(1<<20), goldenInputsBudget,
			float64(goldenInputsBudget)/(1<<20))
	}
}
