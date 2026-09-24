//go:build research

package grammar

// m3_naissance_mesures_research_test.go — LOT M3.2, LES MESURES DU CANAL DES NAISSANCES :
//
//   - `TestM3ImageCleN2` : la reparation H-DS32 vue par l oracle `n2` des images-cles — la
//     derniere feuille de l etat par defaut du bipede (`FUN_14080d69c`, `uVar10 >= 12`) lit-elle
//     son R(32) quand sa porte vaut 1, AUSSI dans le cadre d image-cle ?
//   - `TestM3NaissancesProduction` : la lecture de PRODUCTION des dotations, confrontee a
//     l oracle par catalogue de la sonde P3 ;
//   - `TestM3PremieresEmissionsContreNaissance` : les premieres emissions de meme famille que la
//     naissance, moities basses comparees.
//
// L ORACLE (`imagecle_oracle_n2_research_test.go`, lot 1.9.1 bis) : dans un record d image-cle,
// `FUN_142e2bfd0` lit `R(32) n1 | etat par defaut | R(32) n2`. `n1` et `n2` sont des TAILLES DE
// TAMPON, constantes par archetype et par build ; `n2` se lit APRES l etat par defaut, donc une
// largeur fausse le fait atterrir sur des bits quelconques. La lecture qui rend `n2` CONSTANT est
// la bonne.
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> M3_N2=1 \
//	  go test -tags=research -count=1 -v -run '^TestM3ImageCleN2$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"os"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// m3EtatParDefautSansOpt32 est la lecture de l etat par defaut du bipede AVANT le lot M3.2 : la
// derniere feuille (`uVar10 >= 12`) ne lisait que sa porte R(1), jamais son R(32). C est le
// temoin « avant » des mesures du lot, garde a l identique.
func m3EtatParDefautSansOpt32(br *Lecteur) {
	uVar10 := uint32(13)
	if br.ReadBit() {
		uVar10 = uint32(br.ReadBits(8))
	}
	if br.ReadBit() {
		br.ReadBits(32)
	}
	if int32(uVar10) > 10 {
		consumeGate0R(br, 5)
	}
	consumeMultiplayerPropertiesBlock(br)
	if br.ReadBit() {
		br.ReadBits(6)
	}
	br.ReadBit()
	consumeOpt32(br)
	br.ReadBits(19)
	if int32(uVar10) > 5 {
		br.ReadBit()
	}
	if int32(uVar10) >= 12 {
		br.ReadBit()
	}
}

// TestM3ImageCleN2 publie, pour chaque record d image-cle ti=35, `n2` lu apres l etat par defaut
// sous les deux lectures (AVANT : porte seule ; H-DS32 : porte puis R(32) quand elle vaut 1).
func TestM3ImageCleN2(t *testing.T) {
	if os.Getenv("M3_N2") == "" {
		t.Skip("M3_N2 absent")
	}
	tc := t516Cadre(t)
	ctx := tc.fc.ContexteDeLecture()
	n2Avant, n2DS32, portes := map[uint64]int{}, map[uint64]int{}, map[uint64]int{}
	n := 0
	for _, ch := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(ch)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			for _, r := range WalkKeyframeWorld(pay) {
				if r.TI != BipedTypeIndex {
					continue
				}
				n++
				for _, ds32 := range []bool{false, true} {
					br := LecteurSur(pay)
					br.PoserContexte(ctx)
					br.SetBitPos(r.Bit + br.cadre().EnTeteBits)
					mot := uint(br.cadre().MotDeTailleBits) //nolint:gosec // largeur de profil
					if int32(br.ReadBits(mot)) <= 0 {       //nolint:gosec // n1 signe
						continue
					}
					if ds32 {
						m3EtatParDefautOpt32(br)
					} else {
						m3EtatParDefautSansOpt32(br)
						portes[PeekBits(pay, br.BitPos()-1, 1)]++
					}
					if br.p.Grammaire.ControleDeCorruption {
						br.ReadBits(mot)
					}
					n2 := br.ReadBits(mot)
					if ds32 {
						n2DS32[n2]++
					} else {
						n2Avant[n2]++
					}
				}
			}
		}
	}
	t.Logf("== image-cle ti=35 : %d records ; porte de la derniere feuille %v", n, portes)
	t.Logf("   n2 AVANT (porte seule) : %d valeurs distinctes ; %v", len(n2Avant), m3Principales(n2Avant, 6))
	t.Logf("   n2 H-DS32              : %d valeurs distinctes ; %v", len(n2DS32), m3Principales(n2DS32, 6))
}

// m3Principales rend les `k` valeurs les plus frequentes d un histogramme, avec leur compte.
func m3Principales(h map[uint64]int, k int) map[uint64]int {
	out := map[uint64]int{}
	for len(out) < k && len(out) < len(h) {
		var best uint64
		bestN := -1
		for v, c := range h {
			if _, deja := out[v]; !deja && c > bestN {
				best, bestN = v, c
			}
		}
		out[best] = bestN
	}
	return out
}

// TestM3NaissancesProduction rejoue la LECTURE DE PRODUCTION des dotations de naissance
// (`ScanBirthLoadouts`) sur un film, et confronte chaque dotation rendue a l oracle par
// catalogue de la sonde P3 (familles nommees par le document du match) : la famille de chaque
// emplacement lu doit etre celle que l oracle localise, dans le meme ordre.
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> M3_DOC=<document> \
//	  go test -tags=research -count=1 -v -run '^TestM3NaissancesProduction$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/
func TestM3NaissancesProduction(t *testing.T) {
	doc := os.Getenv("M3_DOC")
	if doc == "" {
		t.Skip("M3_DOC absent")
	}
	tc := t516Cadre(t)
	cat := m3Catalogue(t, doc)
	cres, _, err := ScanBipedCreations(tc.fc)
	if err != nil {
		t.Fatalf("creations : %v", err)
	}
	births, st, err := ScanBirthLoadouts(tc.fc, cres)
	if err != nil {
		t.Fatalf("naissances : %v", err)
	}
	t.Logf("== naissances : %+v", st)
	pos := map[[2]int]BipedCreation{}
	for _, c := range cres {
		pos[[2]int{int(c.Slot), int(c.TimestampUS / 1000)}] = c
	}
	var accord, desaccord, sansOracle int
	for _, b := range births {
		c := pos[[2]int{int(b.Slot), int(b.TimestampUS / 1000)}]
		pay := m3Paquet(tc.fc, c)
		_, fams := m3OracleArmes(pay, c.BitPos, tc.cfg, cat)
		if len(fams) == 0 {
			sansOracle++
			continue
		}
		ok := true
		for i, w := range b.Weapons {
			if i < len(fams) && w.Family != noVariant && w.Family != fams[i] {
				ok = false
			}
		}
		if ok {
			accord++
		} else {
			desaccord++
			if desaccord <= 5 {
				t.Logf("   DESACCORD slot %d : lu %+v, oracle %08X", b.Slot, b.Weapons, fams)
			}
		}
	}
	t.Logf("== accord avec l oracle par catalogue : %d / %d (sans oracle %d)", accord, accord+desaccord,
		sansOracle)
}

// TestM3PremieresEmissionsContreNaissance publie, pour chaque premiere emission d un emplacement
// dont la famille EGALE celle de la naissance (le cas « re-annonce »), les deux moities basses :
// egales = la meme arme re-annoncee ; differentes = un AUTRE objet de la meme famille (un
// echange contre une arme identique, pour ses munitions).
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> M3_PREM=1 \
//	  go test -tags=research -count=1 -v -run '^TestM3PremieresEmissionsContreNaissance$' ./internal/games/halo_infinite/film/internal/grammar/
func TestM3PremieresEmissionsContreNaissance(t *testing.T) {
	if os.Getenv("M3_PREM") == "" {
		t.Skip("M3_PREM absent")
	}
	tc := t516Cadre(t)
	cres, _, err := ScanBipedCreations(tc.fc)
	if err != nil {
		t.Fatalf("creations : %v", err)
	}
	births, _, err := ScanBirthLoadouts(tc.fc, cres)
	if err != nil {
		t.Fatalf("naissances : %v", err)
	}
	type cle struct {
		slot uint32
		k    int
	}
	derniere := map[uint32]types.BirthLoadout{}
	par := map[uint32][]types.BirthLoadout{}
	for _, b := range births {
		par[b.Slot] = append(par[b.Slot], b)
	}
	changes, _, err := ScanHeldWeaponChanges(tc.fc, nil)
	if err != nil {
		t.Fatalf("changements : %v", err)
	}
	vus := map[cle]uint64{}
	var egales, differentes int
	for _, ch := range changes {
		var nee types.BirthLoadout
		ok := false
		for _, b := range par[ch.Slot] {
			if b.TimestampUS <= ch.TimestampUS {
				nee, ok = b, true
			}
		}
		if !ok {
			continue
		}
		k := cle{ch.Slot, ch.Emplacement}
		if ts, deja := vus[k]; deja && ts >= nee.TimestampUS {
			continue // pas la premiere emission de cette vie
		}
		vus[k] = ch.TimestampUS
		derniere[ch.Slot] = nee
		for _, w := range nee.Weapons {
			if w.Emplacement != ch.Emplacement || w.Family != ch.Family {
				continue
			}
			if w.Low == ch.Low {
				egales++
			} else {
				differentes++
				t.Logf("   slot %d k%d famille %08X : bas naissance %08X, bas emission %08X (%.1f s apres)",
					ch.Slot, ch.Emplacement, ch.Family, w.Low, ch.Low,
					float64(ch.TimestampUS-nee.TimestampUS)/1e6)
			}
		}
	}
	t.Logf("== premieres emissions de meme famille que la naissance : bas egaux %d, bas differents %d",
		egales, differentes)
}
