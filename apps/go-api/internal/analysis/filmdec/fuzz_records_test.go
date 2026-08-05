package filmdec

// fuzz_records_test.go — LE HARNAIS DE FUZZ DES LECTEURS DE RECORDS.
//
// POURQUOI. Ce paquet lit du BINAIRE QU IL NE MAITRISE PAS : des chunks telecharges d un CDN,
// tronques par un telechargement partiel, produits par un build du jeu qu on n a pas vu. Ses
// lecteurs balayent bit a bit et calculent des bornes a la main (`len(pay)*8 - k`), ce qui est
// exactement la forme de code ou une lecture hors bornes se cache. Un `go test -fuzz` coute
// quasiment rien et attrape les paniques que le corpus reel ne produit jamais.
//
// CE QUE CE HARNAIS GARANTIT, ET RIEN DE PLUS : AUCUNE PANIQUE, quelle que soit l entree. Il ne
// verifie AUCUNE valeur — sur une entree aleatoire il n existe pas de resultat attendu, et un
// fuzz qui affirmerait un resultat testerait le hasard.
//
// LES GRAINES sont des payloads REELS extraits de la mini-bobine (cf.
// internal/analysis/replay/testdata/minifilm_000d5950). Elles vivent sous
// testdata/fuzz/FuzzFilmRecordReaders/ au format de corpus natif de Go, donc `go test` les
// rejoue TOUTES a chaque execution, meme sans `-fuzz`. C est le regime nominal de ce fichier :
// le fuzz long est une campagne qu on lance a la main, la non-regression est le corpus.
//
// CAMPAGNE (a la main, jamais en CI) :
//
//	go test ./internal/analysis/filmdec/ -run FuzzFilmRecordReaders -fuzz FuzzFilmRecordReaders -fuzztime 60s
//
// REGENERATION DES GRAINES :
//
//	go test ./internal/analysis/filmdec/ -run FuzzSeedsRegenerate -update

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

var updateFuzzSeeds = flag.Bool("update", false, "reecrire le corpus de graines de testdata/fuzz/")

// fuzzSeedDir est le corpus natif Go : `go test` charge automatiquement les fichiers qui s y
// trouvent comme graines de la cible du meme nom.
const fuzzSeedDir = "testdata/fuzz/FuzzFilmRecordReaders"

// miniFilmFromFilmdec : la mini-bobine vit avec le paquet qu elle sert d abord (`replay`), et
// ce paquet-ci la lit par un chemin relatif.
//
// LA DUPLIQUER AURAIT ETE PIRE : deux verites binaires a regenerer, donc deux occasions de les
// laisser diverger. Le prix est ce chemin relatif, et il est ecrit ici pour qu il se voie.
const miniFilmFromFilmdec = "../replay/testdata/minifilm_000d5950"

// fuzzMaxSeed borne la taille des graines. Une image-cle entiere pese plus de 100 Ko : la
// donner telle quelle rendrait chaque execution du corpus couteuse pour ne rien ajouter — ce
// qu on cherche ici, ce sont les bornes de lecture, pas le volume.
const fuzzMaxSeed = 4096

// FuzzFilmRecordReaders : les lecteurs de records ne paniquent sur AUCUNE entree.
//
// Les cinq lecteurs balayes sont ceux qui prennent un payload brut et calculent leurs bornes
// eux-memes. Ils sont appeles ici avec EXACTEMENT le contrat de la production — aucune garde
// de longueur ajoutee par le harnais : un harnais qui protege ce que la production ne protege
// pas ne teste rien.
func FuzzFilmRecordReaders(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0xD2})
	wr := Vec3Range{{Min: -100, Max: 100}, {Min: -100, Max: 100}, {Min: -100, Max: 100}}
	band := map[uint32]bool{}
	for s := uint32(1400); s < 1500; s++ {
		band[s] = true
	}
	f.Fuzz(func(t *testing.T, payload []byte) {
		if len(payload) > fuzzMaxSeed*8 {
			t.Skip("entree hors du domaine borne du harnais")
		}
		_ = WalkPackets(payload)
		_ = scanGrenadeThrows(payload)
		_ = scanProjectileRecords(payload, band, &wr)
		_ = WalkKeyframeWorld(payload)
		// Lecture DELIBEREMENT a cheval sur les DEUX bouts du buffer : c est la tolerance que
		// PeekBits documente. Le depart negatif n est pas un caprice du harnais — jusqu au
		// 2026-08-01 la tolerance etait a SENS UNIQUE et une position negative paniquait
		// (`index out of range [-1]`), alors que la documentation annoncait le contraire
		// (decouverte J2, alignee en J3.4). Cette ligne est ce qui interdit la rechute.
		_ = PeekBits(payload, len(payload)*8-3, 24)
		_ = PeekBits(payload, -4, 24)
		// decodeFireEvent est appele SANS garde de longueur, exactement comme le fait
		// `ScanFilmFireEvents` (qui n exige que `p.Size >= 1`). C etait la panique la plus
		// serieuse des deux : le decodeur lisait jusqu au bit 112 a offsets FIXES via
		// `readBitsAt`, qui indexe sans borne. La garde vit maintenant dans le decodeur, et la
		// graine de troncature a trois octets d un payload de tir (seed_04, produite par
		// collectFuzzSeeds) est l entree de crash conservee en regression.
		_, _ = decodeFireEvent(payload)
	})
}

// TestFuzzSeedsRegenerate : LA SEULE PORTE D ECRITURE DU CORPUS DE GRAINES.
//
// Les graines sont ecrites au format de corpus natif Go (`go test fuzz v1` puis un litteral
// []byte) : c est ce format que le moteur relit, et l ecrire depuis le code plutot qu a la main
// est ce qui rend le corpus REGENERABLE — un corpus edite a la main est un binaire sans source.
func TestFuzzSeedsRegenerate(t *testing.T) {
	if !*updateFuzzSeeds {
		t.Skip("regeneration du corpus de graines : passer -update")
	}
	seeds, err := collectFuzzSeeds()
	if err != nil {
		t.Fatalf("collecte des graines : %v", err)
	}
	if err := os.MkdirAll(fuzzSeedDir, 0o750); err != nil {
		t.Fatalf("creation de %s : %v", fuzzSeedDir, err)
	}
	for i, s := range seeds {
		path := filepath.Join(fuzzSeedDir, fmt.Sprintf("seed_%02d", i))
		body := "go test fuzz v1\n[]byte(" + strconv.Quote(string(s)) + ")\n"
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", path, err)
		}
	}
	t.Logf("%d graine(s) reecrite(s) sous %s", len(seeds), fuzzSeedDir)
}

// collectFuzzSeeds prend, dans la mini-bobine, un echantillon de payloads REELS de chaque
// nature, plus deux TRONCATURES — parce qu un payload tronque est precisement le cas que le
// corpus reel ne contient jamais et que le reseau produit.
func collectFuzzSeeds() ([][]byte, error) {
	chunk, err := ReadFilmChunk(miniFilmFromFilmdec, 1)
	if err != nil {
		return nil, err
	}
	packets := WalkPackets(chunk)
	if len(packets) == 0 {
		return nil, fmt.Errorf("mini-bobine sans paquet lisible")
	}
	var fire, keyframe, other []byte
	for _, p := range packets {
		pay := p.Payload(chunk)
		switch {
		case p.Type == PacketTypeKeyframe && keyframe == nil:
			keyframe = pay
		case p.Type == PacketTypeDelta && len(pay) > 0 &&
			int(pay[0]>>1) == FireEventType && int(pay[0])&1 == 0 && fire == nil:
			fire = pay
		case p.Type == PacketTypeDelta && other == nil:
			other = pay
		}
	}
	seeds := [][]byte{}
	for _, s := range [][]byte{fire, other, keyframe} {
		if s == nil {
			continue
		}
		seeds = append(seeds, clampSeed(s))
	}
	if len(seeds) == 0 {
		return nil, fmt.Errorf("aucun payload exploitable dans la mini-bobine")
	}
	// Les troncatures : la moitie, puis trois octets. La seconde force tous les lecteurs a
	// calculer une borne NEGATIVE, ce qui est le mode d echec qu on veut interdire.
	head := seeds[0]
	seeds = append(seeds, append([]byte(nil), head[:len(head)/2]...))
	seeds = append(seeds, append([]byte(nil), head[:min(3, len(head))]...))
	return seeds, nil
}

func clampSeed(b []byte) []byte {
	if len(b) > fuzzMaxSeed {
		b = b[:fuzzMaxSeed]
	}
	return append([]byte(nil), b...)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
