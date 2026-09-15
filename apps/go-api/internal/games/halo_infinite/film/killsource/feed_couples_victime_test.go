package killsource

// feed_couples_victime_test.go — LA MOITIE VICTIME DU COUPLE ECRIT DECIDE, ELLE AUSSI
// (revue de jalon M1, lentille L6 : ce que les tests ne couvrent pas).
//
// # LE TROU QUE CE FICHIER FERME, ET IL A ETE MESURE
//
// [resolveurDeCouples.chercherCoupleEcrit] exige que les DEUX indices epingles d un kill-event
// nomment exactement le couple que le feed ecrit. Mutation jouee le 2026-09-15 sur la base
// `34fa53da5` : `k == tueur && v == victime` remplace par `k == tueur && v != "" && victime != ""`
// — c est-a-dire la moitie VICTIME du predicat SUPPRIMEE — et le paquet restait `ok`. Les temoins
// de `feed_couples_test.go` ne passent qu UN SEUL `killEventRec` a la fois : avec un candidat
// unique, comparer la victime ou ne pas la comparer rend le meme enregistrement.
//
// # CE QUE LE TEMOIN CI-DESSOUS FABRIQUE, ET POURQUOI IL FAUT DEUX ENREGISTREMENTS
//
// Le meme tueur tue DEUX FOIS dans la fenetre de 2,5 s ([tolMS]) : c est exactement le cas que
// l en-tete de `feed_couples.go` nomme comme la raison d etre du premier temps. Le film ecrit
// alors DEUX kill-events au meme nom de tueur et a des victimes DIFFERENTES, et le couple que le
// feed porte au meme instant doit prendre LE SIEN — pas le premier venu.
//
// L ORDRE DES ENREGISTREMENTS EST LE MATERIAU DU TEST : celui qui ne correspond PAS vient en
// premier (il est le plus ancien, comme `scanKillEvents` les rend). Un predicat qui ne regarde
// que le tueur prend donc le MAUVAIS, et le kill orphelin voisin herite alors de la victime de
// l autre mort — la fabrication meme que le lot 1.9.3 a supprimee.
//
//	t=2000  kill de C, sans mort en face        <- l orphelin ; le film ecrit sa victime : D
//	t=2050  mort de D                           <- la mort que la lecture doit lui rattacher
//	t=2100  kill de C ET mort de E au meme instant <- le couple ECRIT, qui doit prendre SON rec
//
//	rec[0] = 2000  C -> D    (le plus ancien, celui du kill orphelin)
//	rec[1] = 2100  C -> E    (celui du couple ecrit)

import "testing"

// coupleTemoinDeuxVictimes : le decor de [coupleTemoin], dont le TROISIEME instant devient un
// couple ECRIT (kill de C ET mort de E au meme instant) au lieu d une mort isolee. Le roster et
// la table des xuids sont ceux de `coupleTemoin` — un seul decor pour un seul paquet.
func coupleTemoinDeuxVictimes() (*killFeed, *roster) {
	kf, r := coupleTemoin()
	kf.events[2].killer = "C"
	return kf, r
}

// TestLeCoupleEcritPrendLeRecordDeSaVictime — LE TEMOIN. Deux kill-events du meme tueur dans la
// fenetre : le couple ecrit (C, E) consomme celui qui nomme E, et le kill orphelin de C lit SA
// victime, D, dans ce qui reste.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : dans [resolveurDeCouples.chercherCoupleEcrit], remplacer
// `v == victime` par `v != "" && victime != ""`. Le couple ecrit consomme alors rec[0] (C -> D),
// l orphelin ne trouve plus que rec[1] (C -> E) et publie (C, E) — la victime de L AUTRE kill —,
// la mort de D reste sans tueur, et l accord devient une contradiction. Jouee et restauree par
// nom le 2026-09-15 (sorties collees au §5 du plan).
func TestLeCoupleEcritPrendLeRecordDeSaVictime(t *testing.T) {
	kf, r := coupleTemoinDeuxVictimes()
	recs := []killEventRec{coupleRec(2000, 1, 2), coupleRec(2100, 1, 3)}

	st := kf.resoudreCouples(recs, r)

	if len(kf.pairs) != 2 {
		t.Fatalf("couples = %+v, attendu 2 : celui que le feed ECRIT et celui que la lecture rend",
			kf.pairs)
	}
	lu := kf.pairs[0]
	if lu.timeMS != 2000 || lu.victim != "D" {
		t.Fatalf("couple lu = %+v, attendu (C, D) a t=2000 : le film ecrit D pour CE kill — "+
			"E appartient a l autre mort du meme tueur", lu)
	}
	if lu.victimXUID != 44 {
		t.Errorf("xuid de la victime lue = %d, attendu 44 (celui de D, pris a la mort voisine)",
			lu.victimXUID)
	}
	if st.MemeInstant != 1 || st.Lus != 1 || st.Recolles != 0 || st.Ambigu != 0 || st.Muet != 0 {
		t.Errorf("compteurs = %+v, attendu 1 meme-instant / 1 lu / 0 recolle / 0 ambigu / 0 muet", st)
	}
	if st.Accord != 1 || st.Contradiction != 0 {
		t.Errorf("controle = %d accord / %d contradiction, attendu 1 / 0 : la mort de D est bien "+
			"au voisin immediat du kill que le film lui rattache", st.Accord, st.Contradiction)
	}
	if len(kf.orphD) != 0 {
		t.Errorf("morts sans tueur = %+v, attendu AUCUNE : les deux morts sont revendiquees", kf.orphD)
	}
}

// TestLeCoupleEcritNeVolePasLeRecordDuVoisin : la meme fixture, lue par l autre bout. Le
// kill-event que le couple ECRIT consomme est celui de SA victime, donc celui de l orphelin reste
// disponible — et c est ce qui empeche l orphelin de recevoir la victime d une autre mort.
//
// Il ne double pas le temoin precedent : celui-ci porte le RESULTAT (qui est publie), celui-la
// porte la CONSOMMATION (quel enregistrement a servi). Sans le second, un predicat qui prendrait
// le bon enregistrement pour une mauvaise raison — par exemple en consommant les deux — resterait
// invisible.
func TestLeCoupleEcritNeVolePasLeRecordDuVoisin(t *testing.T) {
	kf, r := coupleTemoinDeuxVictimes()
	recs := []killEventRec{coupleRec(2000, 1, 2), coupleRec(2100, 1, 3)}
	res := &resolveurDeCouples{kf: kf, recs: recs, r: r,
		pris: make([]bool, len(recs)), prisMort: make([]bool, len(kf.events))}

	res.consommerLesCouplesDuMemeInstant()

	if res.pris[0] {
		t.Errorf("le couple ecrit a consomme rec[0] (C -> D), qui est celui du kill ORPHELIN : " +
			"la moitie victime du predicat ne decide plus")
	}
	if !res.pris[1] {
		t.Errorf("le couple ecrit (C, E) n a PAS consomme rec[1] (C -> E), le seul qui le nomme")
	}
}
