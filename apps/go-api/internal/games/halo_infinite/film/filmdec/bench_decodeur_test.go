package filmdec

// bench_decodeur_test.go — LE BUDGET DE TEMPS DU DECODEUR (lot 0.A.5).
//
// # POURQUOI DES BANCS, ALORS QUE `replay-equiv` CHRONOMETRE DEJA
//
// `replay-equiv` rend une duree PAR FILM (cf. `cmd/replay-equiv/parent.go`, colonne `res.Dur`) :
// c'est le budget de bout en bout, et c'est lui qui dit si un pas de la trajectoire a coute cher.
// Mais il ne dit pas OU : une cuisson qui ralentit de 10 % peut venir du lecteur de bits, d'un
// balayage chaud ou du constructeur. Ces bancs isolent les trois etages que la revision (M2) va
// deplacer, pour qu'une regression se localise au lieu de se constater.
//
// # CE QU'ILS MESURENT, ET SUR QUOI
//
//	BenchmarkBitReaderReadBits    le primitif — lecture de bits en rafale, sans aucune grammaire.
//	BenchmarkTraverseEntity       la boucle de composants sur des records REELS d'image-cle.
//	BenchmarkKeyframeClosure      le balayage chaud complet d une bobine (le plus proche de la
//	                              cuisson : il traverse tous les chunks).
//
// La matiere est la mini-bobine `bcb6d393` (lot 0.A.2) : elle porte son `chunk_00`, donc son
// registre, donc une vraie grammaire — et elle est la plus legere des sept (941 Ko), ce qui garde
// le banc utilisable en boucle courte. Une bobine, pas un film du cache : un banc qui dependrait
// de `data/` ne tournerait pas en CI.
//
// # LA LIGNE DE BASE
//
//	go test -bench . -run '^$' -count 10 ./internal/games/halo_infinite/film/filmdec/ \
//	  > internal/games/halo_infinite/film/filmdec/testdata/bench_baseline.txt
//
// Comparaison a chaque cloture de M2, SUR LA MEDIANE (ce que `benchstat` compare) :
//
// LE BUDGET DE +10 % NE VAUT QUE POUR LES DEUX BANCS SERRES (`BitReaderReadBits`,
// `TraverseEntity`). `KeyframeClosure` est INFORMATIF : mesure de la revue R1, 71 % d ecart au
// sein d une meme passe et +21 % de mediane d une passe a l autre SANS changement de code. Son
// ecart suit la charge de la machine ; un budget de +10 % dessus ferait rougir des lots innocents
// et laisserait passer de vrais ralentissements. Detail : `testdata/bench_baseline.txt`.
//
//	go test -bench . -run '^$' -count 10 ./internal/games/halo_infinite/film/filmdec/ > apres.txt
//	benchstat internal/games/halo_infinite/film/filmdec/testdata/bench_baseline.txt apres.txt

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// benchBobine : la bobine de reference des bancs, relative au paquet.
const benchBobine = "../replay/testdata/minifilm_bcb6d393"

// benchReadBitsWidths : les largeurs balayees par le banc du lecteur de bits.
//
// ELLES NE SONT PAS ARBITRAIRES : 1 bit (les portes et les bits de presence), 5 (les en-tetes de
// manche du statborg), 6 (l'etat par defaut de ti=6), 16 (les quantifications d'endpoint) et 32
// (les mots de taille de l'image-cle). Un banc sur une seule largeur ne dirait rien du cout reel,
// qui melange les cinq.
func benchReadBitsWidths() []uint { return []uint{1, 5, 6, 16, 32} }

// BenchmarkBitReaderReadBits mesure le primitif, sans grammaire.
func BenchmarkBitReaderReadBits(b *testing.B) {
	// 64 Kio de matiere pseudo-aleatoire DETERMINISTE : le banc doit rendre le meme travail a
	// chaque execution, sinon la comparaison `benchstat` mesure le hasard.
	buf := make([]byte, 64<<10)
	for i := range buf {
		buf[i] = byte(i*31 + 7)
	}
	widths := benchReadBitsWidths()
	b.SetBytes(int64(len(buf)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br := NewBitReader(buf)
		for j := 0; ; j++ {
			w := widths[j%len(widths)]
			if br.BitPos()+int(w) > len(buf)*8 {
				break
			}
			_ = br.ReadBits(w)
		}
	}
}

// BenchmarkTraverseEntity mesure la boucle de composants sur des records REELS.
func BenchmarkTraverseEntity(b *testing.B) {
	pays, reg := benchKeyframePayloads(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, pay := range pays {
			for _, rec := range WalkKeyframeWorld(pay) {
				br := NewBitReader(pay)
				br.SetBitPos(rec.Bit)
				_ = TraverseEntity(br, reg, 0)
			}
		}
	}
}

// BenchmarkKeyframeClosure mesure le BALAYAGE CHAUD complet d'une bobine : tous les chunks, tous
// les paquets d'image-cle, tous les records, toute la boucle de composants.
//
// POURQUOI PAS `ScanBipedPositions`, QUE LE PLAN NOMMAIT. Il ne s'execute PAS sur une mini-bobine :
// il derive sa bande de slots des images-cles et refuse le film quand elle est vide
// (« aucun slot biped (ti=35) dans les keyframes du film », `offline_biped.go`). Les images-cles
// d'une bobine sont concatenees HORS de leur continuite — c'est ce que dit chaque PROVENANCE.txt —
// et la bande ne s'y etablit pas. Le faire tourner exigerait un film entier du cache, donc un banc
// qui depend de `data/` et ne tourne pas en CI. `KeyframeClosure` est le balayage chaud EQUIVALENT
// pour ce chantier : c'est exactement le chemin que la revision (M2) va deplacer, et il traverse
// la meme matiere.
func BenchmarkKeyframeClosure(b *testing.B) {
	film := benchFilm(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := KeyframeClosure(NewFilmContext(film)); err != nil {
			b.Fatalf("KeyframeClosure : %v", err)
		}
	}
}

// benchFilm charge la bobine des bancs UNE fois.
func benchFilm(b *testing.B) *filmsource.Film {
	b.Helper()
	film, err := filmsource.LoadDir(filepath.FromSlash(benchBobine), nil)
	if err != nil {
		b.Fatalf("bobine des bancs illisible (%s) : %v — regenerer les bobines du lot 0.A.2",
			benchBobine, err)
	}
	return film
}

// benchKeyframePayloads rend les payloads d'image-cle de la bobine et son registre.
func benchKeyframePayloads(b *testing.B) ([][]byte, *Registry) {
	b.Helper()
	film := benchFilm(b)
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		b.Fatalf("registre de la bobine : %v", err)
	}
	var pays [][]byte
	for _, num := range fc.ChunkNumbers() {
		data, packets, ok := fc.ChunkAt(num)
		if !ok {
			continue
		}
		for _, pk := range packets {
			if pk.Type == PacketTypeKeyframe {
				pays = append(pays, pk.Payload(data))
			}
		}
	}
	if len(pays) == 0 {
		b.Fatal("aucune image-cle dans la bobine des bancs : le banc ne mesurerait rien")
	}
	return pays, reg
}
