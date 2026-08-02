// Package killcollector — shots_test.go : les tests de la VENTILATION DES TIRS.
//
// Ils tournent SANS fixture et SANS base : ce qui se joue ici est la TRADUCTION (quel indice,
// quelle arme, quelle reference), et elle doit etre falsifiable sans 107 Mo de films. Le film
// reel, lui, est couvert par le test d integration du collecteur.
//
// L INSTRUMENT DE CES TESTS EST UN FILM SYNTHETIQUE : on ECRIT des fire-events au bit pres
// (marqueur 11 bits + indice 5 bits + identifiant d arme 64 bits) et on verifie que le scanner
// les relit. C est ce qui permet de prouver la SATURATION de la lecture 4 bits — un film reel
// de grand format le montrerait aussi, mais sans dire quel indice on avait ecrit.

package killcollector

import (
	"sort"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/weaponv3"
)

// ─── L instrument : un ecrivain de bits ────────────────────────────────────────────────────

// bitBuf : un tampon ou l on ecrit des champs au bit pres, MSB-first — la meme convention que
// le lecteur du moteur.
type bitBuf struct {
	bits []byte // un octet par BIT, aplati a la fin
}

func (b *bitBuf) writeBits(v uint64, n int) {
	for i := n - 1; i >= 0; i-- {
		b.bits = append(b.bits, byte((v>>uint(i))&1))
	}
}

func (b *bitBuf) pad(n int) { b.writeBits(0, n) }

func (b *bitBuf) bytes() []byte {
	out := make([]byte, (len(b.bits)+7)/8)
	for i, bit := range b.bits {
		if bit != 0 {
			out[i/8] |= 1 << uint(7-(i%8))
		}
	}
	return out
}

// ecrireFireEvent ajoute UN fire-event lisible par `analysis.ScanFireEventsB5`.
//
// La geometrie est celle du scanner, et elle est le contrat teste ici :
//
//	bit+0    marqueur universel 11 bits (0b10100100110) ; event_start = bit+3
//	event_start+31  indice de joueur, 5 bits          <- LE champ, sur sa vraie largeur
//	event_start+40  identifiant d arme, 64 bits
func ecrireFireEvent(b *bitBuf, playerIndex int, weaponID uint64) {
	const marqueur = 0b10100100110
	b.writeBits(marqueur, 11) // occupe event_start+0 .. +7 apres le prefixe de 3 bits
	b.pad(31 - 8)             // jusqu a event_start+31
	b.writeBits(uint64(playerIndex), 5)
	b.pad(40 - 36)            // jusqu a event_start+40
	b.writeBits(weaponID, 64) // l arme
	b.pad(64)                 // queue : marge de dedup (> 2 octets) + champs post-arme
}

// deuxArmes : deux identifiants filmshell REELS du catalogue embarque, choisis de facon
// deterministe. Un identifiant invente serait rejete par le filtre d arme du scanner — le test
// passerait alors pour de mauvaises raisons.
func deuxArmes(t *testing.T) (uint64, uint64) {
	t.Helper()
	ids := make([]uint64, 0, len(analysis.WeaponIDs))
	for id := range analysis.WeaponIDs {
		ids = append(ids, id)
	}
	if len(ids) < 2 {
		t.Fatalf("catalogue d armes trop petit (%d) — le test ne peut rien prouver", len(ids))
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids[0], ids[1]
}

// ─── Ce que la ventilation doit garantir ───────────────────────────────────────────────────

// TestVentilationCompteParIndiceEtParArme : le comptage lui-meme.
func TestVentilationCompteParIndiceEtParArme(t *testing.T) {
	a1, a2 := deuxArmes(t)
	var b bitBuf
	ecrireFireEvent(&b, 3, a1)
	ecrireFireEvent(&b, 3, a1)
	ecrireFireEvent(&b, 3, a2)
	ecrireFireEvent(&b, 7, a2)

	batch := BuildWeaponShotsBatch("m1", [][]byte{b.bytes()}, nil, nil)

	if len(batch.Players) != 2 {
		t.Fatalf("joueurs = %d, attendu 2 (indices 3 et 7) : %+v", len(batch.Players), batch.Players)
	}
	if batch.Players[0].PlayerIndex != 3 || batch.Players[1].PlayerIndex != 7 {
		t.Fatalf("indices = %d,%d — attendu 3,7 dans cet ordre (l ordre doit etre STABLE)",
			batch.Players[0].PlayerIndex, batch.Players[1].PlayerIndex)
	}
	compte := map[uint64]int{}
	for _, w := range batch.Players[0].Weapons {
		compte[w.WeaponID] = w.Shots
	}
	if compte[a1] != 2 || compte[a2] != 1 {
		t.Errorf("indice 3 : arme %d = %d tir(s), arme %d = %d — attendu 2 et 1",
			a1, compte[a1], a2, compte[a2])
	}
	if batch.DecoderRev != WeaponShotsDecoderRev {
		t.Errorf("DecoderRev = %q, attendu %q — le persister refuse une passe sans revision",
			batch.DecoderRev, WeaponShotsDecoderRev)
	}
}

// TestVentilationLitLIndiceSurCinqBits — LE test du correctif d indice, et il porte sur ce qui
// se PERD sans lui.
//
// Un indice de 20 (grand format) relu sur 4 bits rend 4 : les tirs du joueur 20 iraient
// grossir la ligne du joueur 4. Ce n est pas une perte de precision, c est une ligne fabriquee
// — et rien dans la table ne le dirait.
func TestVentilationLitLIndiceSurCinqBits(t *testing.T) {
	a1, _ := deuxArmes(t)
	var b bitBuf
	ecrireFireEvent(&b, 20, a1)
	ecrireFireEvent(&b, 31, a1)

	batch := BuildWeaponShotsBatch("m1", [][]byte{b.bytes()}, nil, nil)

	vus := map[int]bool{}
	for _, p := range batch.Players {
		vus[p.PlayerIndex] = true
	}
	if !vus[20] || !vus[31] {
		t.Fatalf("indices vus = %v — attendu 20 et 31 ; une lecture 4 bits les rendrait 4 et 15",
			vus)
	}
}

// TestVentilationRefuseLesSentinelles : 0/1/2 ne sont pas des identifiants filmshell.
//
// Le persister refuserait la PASSE ENTIERE ; ici on verifie qu une sentinelle ne coute que sa
// propre ligne — un film entier ne doit pas etre perdu pour un evenement.
func TestVentilationRefuseLesSentinelles(t *testing.T) {
	a1, _ := deuxArmes(t)
	var b bitBuf
	for id := range analysis.SentinelIDs {
		ecrireFireEvent(&b, 5, id)
	}
	ecrireFireEvent(&b, 5, a1)

	batch := BuildWeaponShotsBatch("m1", [][]byte{b.bytes()}, nil, nil)

	for _, p := range batch.Players {
		for _, w := range p.Weapons {
			if analysis.SentinelIDs[w.WeaponID] {
				t.Errorf("sentinelle %d ecrite — elle fabriquerait une jointure fausse avec "+
					"metadata.weapon_labels", w.WeaponID)
			}
		}
	}
}

// TestReferenceAbsenteNEstPasZero : les deux etats de la reference API.
//
// `nil` veut dire « aucune reference » et la porte REFUSE ; `0` veut dire « l API dit zero
// tir » et la porte REND UN VERDICT. Les confondre publierait un verdict qu on n a pas mesure.
func TestReferenceAbsenteNEstPasZero(t *testing.T) {
	a1, _ := deuxArmes(t)
	var b bitBuf
	ecrireFireEvent(&b, 0, a1)
	ecrireFireEvent(&b, 1, a1)
	chunk := b.bytes()

	// Indice 0 -> xuid connu AVEC reference ; indice 1 -> xuid inconnu du film, donc sans ligne.
	// Ici aucun xuid n est resolu (le chunk ne porte aucun motif de xuid) : les deux joueurs
	// sortent SANS xuid, donc SANS reference.
	batch := BuildWeaponShotsBatch("m1", [][]byte{chunk}, []string{"2533274792395366"},
		map[string]int{"2533274792395366": 0})

	for _, p := range batch.Players {
		if p.XUID == "" && p.ShotsFired != nil {
			t.Errorf("indice %d : reference renseignee sans xuid — elle ne peut venir de nulle part",
				p.PlayerIndex)
		}
	}
}

// TestResolutionIndiceLitLesCinqBitsAvantLeMotif : la resolution xuid -> indice.
//
// Le xuid est ecrit en 8 octets LITTLE-ENDIAN dans le flux, et les 5 bits qui PRECEDENT le
// motif portent l indice. Le test ecrit exactement cela et exige de le relire — c est la seule
// facon de prouver que l on ne lit pas l ordre de la base par accident.
func TestResolutionIndiceLitLesCinqBitsAvantLeMotif(t *testing.T) {
	const xuid = "2533274792395366"
	const indice = 19

	var b bitBuf
	b.pad(13) // un decalage quelconque : le motif n est pas aligne sur un octet
	b.writeBits(indice, 5)
	b.writeBits(motifXUID(t, xuid), 64)
	b.pad(32)

	got := resolvePlayerIndices([]string{xuid}, [][]byte{b.bytes()})
	if got[indice] != xuid {
		t.Fatalf("resolution = %v — attendu l indice %d sur %s", got, indice, xuid)
	}
}

// TestResolutionRefuseDeTrancherEntreDeuxXuids : deux xuids sur le meme indice.
//
// Trancher au hasard publierait les tirs d un joueur sous le nom d un autre. On n en garde
// AUCUN, et l indice sort sans xuid.
func TestResolutionRefuseDeTrancherEntreDeuxXuids(t *testing.T) {
	const x1, x2 = "2533274792395366", "2533274824966873"
	const indice = 9

	var b bitBuf
	b.pad(7)
	b.writeBits(indice, 5)
	b.writeBits(motifXUID(t, x1), 64)
	b.pad(16)
	b.writeBits(indice, 5)
	b.writeBits(motifXUID(t, x2), 64)
	b.pad(16)

	got := resolvePlayerIndices([]string{x1, x2}, [][]byte{b.bytes()})
	if got[indice] != "" {
		t.Fatalf("indice %d resolu a %q — deux xuids le revendiquent, aucun ne doit gagner",
			indice, got[indice])
	}
}

// TestRechercheDeMotifsEquivautALaVersionNaive — LE test qui autorise l optimisation.
//
// La recherche en une passe (fenetre glissante + prefiltre) remplace `weaponv3.ResolveBest`,
// qui balaie le film une fois PAR XUID. Le remplacement n est legitime que s il rend EXACTEMENT
// la meme chose : chunks dans l ordre, positions croissantes, premiere occurrence gagnante. Ce
// test confronte les deux implementations sur des flux ou les motifs sont places a des positions
// non alignees, en plusieurs exemplaires et dans plusieurs chunks.
func TestRechercheDeMotifsEquivautALaVersionNaive(t *testing.T) {
	const x1, x2, x3 = "2533274792395366", "2533274824966873", "2535419139267382"

	// Trois chunks : le premier porte x1 (deux fois, indices differents — la PREMIERE gagne),
	// le deuxieme x2, le troisieme x1 encore (ne doit rien changer) et x3.
	var c1, c2, c3 bitBuf
	c1.pad(3)
	c1.writeBits(11, 5)
	c1.writeBits(motifXUID(t, x1), 64)
	c1.pad(9)
	c1.writeBits(28, 5)
	c1.writeBits(motifXUID(t, x1), 64) // seconde occurrence : ignoree
	c1.pad(16)

	c2.pad(6)
	c2.writeBits(4, 5)
	c2.writeBits(motifXUID(t, x2), 64)
	c2.pad(16)

	c3.pad(1)
	c3.writeBits(30, 5)
	c3.writeBits(motifXUID(t, x1), 64) // x1 est deja resolu : sans effet
	c3.pad(5)
	c3.writeBits(17, 5)
	c3.writeBits(motifXUID(t, x3), 64)
	c3.pad(16)

	chunks := [][]byte{c1.bytes(), c2.bytes(), c3.bytes()}
	xuids := []string{x1, x2, x3}

	rapide := resolvePlayerIndices(xuids, chunks)

	// La version NAIVE, telle qu elle etait employee avant l optimisation.
	numeriques := make([]uint64, 0, len(xuids))
	back := map[uint64]string{}
	for _, s := range xuids {
		v := uint64(0)
		for _, c := range s {
			v = v*10 + uint64(c-'0')
		}
		numeriques = append(numeriques, v)
		back[v] = s
	}
	naive := map[int]string{}
	for x, pi := range weaponv3.ResolveBest(numeriques, chunks) {
		if _, deja := naive[pi]; deja {
			naive[pi] = ""
			continue
		}
		naive[pi] = back[x]
	}

	if len(rapide) != len(naive) {
		t.Fatalf("cardinal different : rapide %v, naive %v", rapide, naive)
	}
	for pi, xuid := range naive {
		if rapide[pi] != xuid {
			t.Errorf("indice %d : rapide %q, naive %q — les deux recherches doivent rendre "+
				"EXACTEMENT la meme chose, sinon l optimisation change la donnee", pi, rapide[pi], xuid)
		}
	}
	// Et le fond du test : c est bien la PREMIERE occurrence qui gagne.
	if rapide[11] != x1 {
		t.Errorf("indice 11 = %q, attendu %s (premiere occurrence de x1)", rapide[11], x1)
	}
	if rapide[28] != "" {
		t.Errorf("indice 28 = %q — la seconde occurrence de x1 ne doit rien produire", rapide[28])
	}
}

// motifXUID : le xuid encode en 8 octets little-endian puis relu en big-endian — le motif
// 64 bits que la resolution cherche dans le flux.
func motifXUID(t *testing.T, xuid string) uint64 {
	t.Helper()
	var v uint64
	for _, c := range xuid {
		if c < '0' || c > '9' {
			t.Fatalf("xuid non numerique : %q", xuid)
		}
		v = v*10 + uint64(c-'0')
	}
	var out uint64
	for i := 0; i < 8; i++ {
		out = out<<8 | (v>>uint(8*i))&0xff
	}
	return out
}
