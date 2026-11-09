package grammar

import "testing"

// player_table_tete_test.go — LE RECUL SUR LES VACANTS DE TETE N EST PAS NEUTRALISE PAR LE
// BOURRAGE (J10.6, GA1-5 de l audit du 2026-09-24).
//
// LE DEFAUT : `chercherDepart` essaie le depart au premier enregistrement reel (recul 0), puis
// recule d un slot vacant tant que le predicat de vacance passe. Mais la queue du tampon est faite
// de zeros : la lecture SANS recul ferme elle aussi a 32 slots — ses derniers « vacants » sont du
// bourrage — et visite les memes enregistrements. A completude egale la premiere lecture gardait
// sa place : `HeadVacant` valait 0 par construction, et le rang de chaque joueur etait faux du
// nombre de vacants de tete. La mesure « 0 sur 1 351 films » ne pouvait pas dire autre chose.
//
// LE TEMOIN : une bobine reelle, dans laquelle on INSERE `k` enregistrements vacants (tout a zero,
// de la longueur que la grammaire calcule) juste avant le slot 0. La table lue doit en porter `k`
// en tete, et chaque rang doit reculer d autant.
func TestReadPlayerTableVacantsDeTeteInseres(t *testing.T) {
	const film, k = "bcb6d393", 2
	d := bobineChunk00(t, film)
	id, err := ReadFilmIdentity(d)
	if err != nil {
		t.Fatalf("%s : identite illisible : %v", film, err)
	}
	avant, repAvant, err := ReadPlayerTable(d, id)
	if err != nil {
		t.Fatalf("%s : lecture de reference : %v", film, err)
	}
	if repAvant.HeadVacant != 0 {
		t.Fatalf("%s : la bobine porte deja %d vacant(s) de tete — choisir une bobine sans", film,
			repAvant.HeadVacant)
	}
	vide := slotVacantBits(repAvant.PersoBytes * 8)
	d2 := insererBitsNuls(d, repAvant.FirstRecordBit, k*vide)

	apres, rep, err := ReadPlayerTable(d2, id)
	if err != nil {
		t.Fatalf("%s + %d vacants de tete : %v", film, k, err)
	}
	if rep.HeadVacant != k || rep.FirstRecordBit != repAvant.FirstRecordBit {
		t.Fatalf("vacants de tete = %d (slot 0 au bit %d), attendu %d (slot 0 au bit %d) : le "+
			"recul a ete neutralise par le bourrage de queue", rep.HeadVacant, rep.FirstRecordBit, k,
			repAvant.FirstRecordBit)
	}
	if len(apres) != len(avant) {
		t.Fatalf("%d occupes lus, attendu %d", len(apres), len(avant))
	}
	for i := range apres {
		if apres[i].XUID != avant[i].XUID || apres[i].FilmIndex != avant[i].FilmIndex+k {
			t.Errorf("occupe %d : xuid %d rang %d, attendu xuid %d rang %d", i, apres[i].XUID,
				apres[i].FilmIndex, avant[i].XUID, avant[i].FilmIndex+k)
		}
	}
}

// insererBitsNuls rend `d` avec `n` bits nuls inseres au bit `at` (ordre MSB d abord, celui du
// lecteur du film). Le tampon grandit d autant : le bourrage de queue reste ce qu il etait.
func insererBitsNuls(d []byte, at, n int) []byte {
	total := len(d)*8 + n
	out := make([]byte, (total+7)/8)
	lire := func(i int) byte { return d[i>>3] >> (7 - uint(i&7)) & 1 }
	for i := 0; i < len(d)*8; i++ {
		j := i
		if i >= at {
			j = i + n
		}
		if lire(i) == 1 {
			out[j>>3] |= 1 << (7 - uint(j&7))
		}
	}
	return out
}
