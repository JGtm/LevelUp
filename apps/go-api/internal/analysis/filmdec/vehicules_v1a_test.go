package filmdec

// vehicules_v1a_test.go — INSTRUMENT DE MESURE du lot V1a (vehicules, 2026-08-31).
//
// V1.1 — QUALIFIER LE CORPUS, ET RIEN D'AUTRE. Le cadrage du 31/08 (§ 4.3) a etabli que le
// nombre de slots `ti=40` releves en image-cle ne qualifie PAS un film : des cartes d'arene
// sans vehicule conduisible en portent (Illusion, 7 slots, 17 records delta sur tout le film).
// Le critere de qualification est le nombre de RECORDS DELTA ACCEPTES, et son seuil est ecrit
// avant la mesure : >= 1 000. Le recensement des vies sort de la MEME marche d'images-cles que
// la bande de slots : il ne coute pas une seconde lecture.
//
// L'ACCEPTATION N'EST PAS REECRITE. Le comptage passe par `v0ScanPayload`
// (`vehicules_v0_composants_test.go`), qui reprend exactement le filtre de
// `scanProjectileRecords` — meme ancre, meme exigence d'i0 en tete, meme rejet de quantum
// sature, meme avance apres un record accepte. En changer changerait le denominateur du seuil.
// Les deux instruments vivent et meurent donc ensemble.
//
// RESERVE SUR LES LARGEURS D'AXE, a lire avant d'interpreter un comptage.
// `decodeWorldObjectPos` consomme `WorldObjectPrecision.AxisW`, un global de paquet dont le
// defaut est 13/13/14 (Cliffhanger). L'instrument NE L'INSTALLE PAS, exactement comme le lot V0
// dont vient le seuil : les deux chiffres restent comparables. Ce comptage est un CRITERE DE
// QUALIFICATION, pas une mesure de position — la position juste passe par la grammaire bipede
// (cadrage § 2.2), pas par ce chemin.
//
// LECTURE SEULE : aucun fichier ecrit, aucune base ouverte.
//
//	CGO_ENABLED=0 V1A_FILM_ROOT=<depot>/data/cache \
//	  V1A_FILMS="0d76e8f1:Behemoth SF,fccc61cd:Launch Site SF" \
//	  go test ./internal/analysis/filmdec/ -run TestV1aQualification -v -timeout 180m

import (
	"encoding/binary"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	// v1aRootEnv porte la racine du cache film (celle qui contient `film_chunks`).
	v1aRootEnv = "V1A_FILM_ROOT"
	// v1aFilmsEnv porte le corpus : « short8[:libelle] » separes par des virgules. Le libelle
	// vient de `match_registry` (carte + mode) et n'entre dans aucun calcul : il ne sert qu'a
	// rendre le releve lisible sans allonger un fixture a la main (cadrage § 4.6).
	v1aFilmsEnv = "V1A_FILMS"
)

// v1aSeuilRecords est LE SEUIL DE QUALIFICATION du cadrage § 4.3, ecrit avant toute mesure de
// ce lot : les quatre films mesures au cadrage rendaient 5 454 / 7 760 / 15 284 / 32 328
// records, l'arene statique 17.
const v1aSeuilRecords = 1000

// v1aFilm est une entree du corpus.
type v1aFilm struct{ ID, Libelle string }

// v1aCorpus lit la racine du cache et le corpus dans l'environnement.
func v1aCorpus(t *testing.T) (string, []v1aFilm) {
	t.Helper()
	root, liste := os.Getenv(v1aRootEnv), os.Getenv(v1aFilmsEnv)
	if root == "" || liste == "" {
		t.Skipf("mesure non demandee : %s ou %s vide", v1aRootEnv, v1aFilmsEnv)
	}
	var out []v1aFilm
	for _, s := range strings.Split(liste, ",") {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		id, lib, _ := strings.Cut(s, ":")
		out = append(out, v1aFilm{ID: strings.TrimSpace(id), Libelle: strings.TrimSpace(lib)})
	}
	return root, out
}

// TestV1aQualificationCorpus — LE RELEVE DE QUALIFICATION DU LOT V1a.
//
// Une ligne par film : chunks, images-cles, bande de slots, vies recensees, records delta
// acceptes, verdict. Trois verdicts seulement, et ils sont exclusifs :
//
//	RETENU      >= v1aSeuilRecords records — le film porte des vehicules conduits ;
//	HORS SEUIL  bande non vide mais trop peu de records — entites `ti=40` statiques ;
//	BANDE VIDE  aucun slot `ti=40` en image-cle — c'est ce qu'un temoin negatif doit rendre.
func TestV1aQualificationCorpus(t *testing.T) {
	root, films := v1aCorpus(t)
	t.Logf("V1.1 — seuil ecrit avant mesure : %d records delta ti=%d acceptes ; largeurs d'axe "+
		"du chemin objet du monde laissees au defaut %v (cf. reserve en tete de fichier)",
		v1aSeuilRecords, v0VehiculeTI, WorldObjectPrecision.AxisW)
	retenus, horsSeuil, vides, absents := 0, 0, 0, 0
	for _, f := range films {
		dir := filepath.Join(root, "film_chunks", f.ID)
		n := CountFilmChunks(dir)
		if n == 0 {
			absents++
			t.Logf("%-8s %-42s FILM ABSENT DU CACHE", f.ID, f.Libelle)
			continue
		}
		k := ScanFilmWorldObjectKeyframes(dir, v0VehiculeTI)
		rec := v1aRecordsDelta(dir, n, k.Band)
		uniques, med := v1aVies(k)
		verdict := "HORS SEUIL"
		switch {
		case len(k.Band) == 0:
			verdict, vides = "BANDE VIDE", vides+1
		case rec >= v1aSeuilRecords:
			verdict, retenus = "RETENU", retenus+1
		default:
			horsSeuil++
		}
		t.Logf("%-8s %-42s %2d chunks %2d images-cles bande %3d vies %3d (%3d vues 1 fois, "+
			"mediane %4d s) records %7d %s",
			f.ID, f.Libelle, n, len(k.TimesUS), len(k.Band), len(k.SeenUS), uniques, med, rec,
			verdict)
	}
	t.Logf("V1.1 — %d entrees : %d RETENUS, %d hors seuil, %d a bande vide, %d absents du cache",
		len(films), retenus, horsSeuil, vides, absents)
}

// v1aRecordsDelta compte les records delta de l'archetype acceptes sur TOUT le film.
func v1aRecordsDelta(dir string, n int, band map[uint32]bool) int {
	if len(band) == 0 {
		return 0 // aucun slot dans la bande : aucun record ne peut etre ancre
	}
	total := 0
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			if p.Type != PacketTypeDelta {
				continue
			}
			r, _ := v0ScanPayload(p.Payload(data), band)
			total += r
		}
	}
	return total
}

// v1aVies rend, sur le recensement d'un film, le nombre de vies vues a UNE SEULE image-cle
// (naissance et disparition non bornees des deux cotes) et la duree mediane des autres.
func v1aVies(k WorldObjectKeyframes) (uniques int, medianeS uint64) {
	var durees []uint64
	for _, vus := range k.SeenUS {
		if len(vus) == 1 {
			uniques++
			continue
		}
		durees = append(durees, (vus[len(vus)-1]-vus[0])/1_000_000)
	}
	if len(durees) == 0 {
		return uniques, 0
	}
	sort.Slice(durees, func(i, j int) bool { return durees[i] < durees[j] })
	return uniques, durees[len(durees)/2]
}

// ---------------------------------------------------------------------------------------
// V1.2 — LE GATE DE NON-REGRESSION, mesure AVANT puis APRES le refactor du point d'entree.
// ---------------------------------------------------------------------------------------

const (
	// v1aNonRegDirEnv porte le repertoire de chunks du film temoin (`000d5950`).
	v1aNonRegDirEnv = "V1A_NONREG_DIR"
	// v1aBornesEnv porte le chemin de `map_quant_bounds.json`, v1aCarteEnv le nom de la carte
	// du film temoin. Sans eux, seule la variante « quanta seuls » est mesuree — elle suffit
	// deja a verrouiller la grammaire, les bornes n'entrant que dans la dequantification.
	v1aBornesEnv = "V1A_BOUNDS"
	v1aCarteEnv  = "V1A_MAP"
)

// v1aEmpreinte condense un balayage de positions de bipede. C'est CE QUI DOIT ETRE IDENTIQUE
// avant et apres le refactor : le nombre d'echantillons, et le flux des quanta bruts dans
// l'ordre ou le decodeur les rend. Les coordonnees monde n'entrent pas dans l'empreinte —
// elles sont une fonction des quanta et des bornes, pas une lecture.
type v1aEmpreinte struct {
	N, Slots             int
	Hash                 uint64
	SommeQ               [3]uint64
	PremierUS, DernierUS uint64
}

// TestV1aNonRegressionBipede — L'EMPREINTE DU CHEMIN BIPEDE, A REJOUER APRES LE REFACTOR.
//
// POURQUOI TROIS VARIANTES ET PAS UNE. Le chemin nominal de production porte des bornes de
// carte (donc `DropTeleports`) et, pour la chaine arme-du-kill, la capture des directions
// (donc la poursuite du curseur apres i0). Un refactor du point d'entree peut casser l'une
// sans toucher aux autres : mesurer la seule variante « quanta » laisserait les deux filtres
// de post-traitement hors du gate.
func TestV1aNonRegressionBipede(t *testing.T) {
	dir := os.Getenv(v1aNonRegDirEnv)
	if dir == "" {
		t.Skipf("mesure non demandee : %s vide", v1aNonRegDirEnv)
	}
	if CountFilmChunks(dir) == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	release := LockProcessDecode()
	defer release()
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("decoupage i0 illisible dans %s : %v", dir, err)
	}
	t.Logf("V1.2 non-regression — film %s : decoupage i0 LU DANS LE FILM, gate %d bits, "+
		"axes %v, region %d", filepath.Base(dir), lay.GateBits, lay.AxisW, lay.Region)
	base := DefaultScanFilmOptions()
	quanta := base
	quanta.QuantaOnly = true
	v1aMesureEmpreinte(t, "quanta seuls", dir, quanta)
	wr, ok := v1aBornesTemoin(t)
	if !ok {
		return
	}
	monde := base
	monde.WorldRange = &wr
	v1aMesureEmpreinte(t, "coordonnees monde", dir, monde)
	avecDirs := monde
	avecDirs.CaptureDirs = true
	v1aMesureEmpreinte(t, "coordonnees monde + directions", dir, avecDirs)
}

// v1aBornesTemoin rend les bornes de la carte du film temoin, si elles sont demandees.
func v1aBornesTemoin(t *testing.T) (Vec3Range, bool) {
	t.Helper()
	chemin, carte := os.Getenv(v1aBornesEnv), os.Getenv(v1aCarteEnv)
	if chemin == "" || carte == "" {
		t.Logf("V1.2 non-regression — %s ou %s vide : variantes a bornes non mesurees",
			v1aBornesEnv, v1aCarteEnv)
		return Vec3Range{}, false
	}
	cat, err := LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	e, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("carte %q : %v", carte, err)
	}
	return e.Range(), true
}

// v1aMesureEmpreinte balaie et publie l'empreinte d'une variante.
func v1aMesureEmpreinte(t *testing.T, quoi, dir string, opt ScanFilmOptions) {
	t.Helper()
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("balayage [%s] : %v", quoi, err)
	}
	e := v1aCondense(pos)
	t.Logf("V1.2 non-regression [%-30s] %6d echantillons · %2d slots · empreinte %#016x · "+
		"sommes Q %v · fenetre %d..%d us",
		quoi, e.N, e.Slots, e.Hash, e.SommeQ, e.PremierUS, e.DernierUS)
}

// v1aCondense hache le flux d'echantillons DANS L'ORDRE OU LE DECODEUR LES REND. Le hachage
// porte sur le tuple (slot, chunk, paquet, instant, quanta) : une permutation, un echantillon
// de plus ou un quantum different changent l'empreinte. Les sommes par axe sont un second
// invariant, independant de l'ordre — utile pour distinguer « ordre change » de « valeurs
// changees » si l'empreinte devait bouger.
func v1aCondense(pos []BipedPosition) v1aEmpreinte {
	var e v1aEmpreinte
	h := fnv.New64a()
	var buf [8]byte
	ecris := func(v uint64) {
		binary.BigEndian.PutUint64(buf[:], v)
		_, _ = h.Write(buf[:])
	}
	slots := map[uint32]bool{}
	for i, p := range pos {
		ecris(uint64(p.Slot))
		ecris(uint64(p.Chunk))
		ecris(uint64(p.PacketIndex))
		ecris(p.TimestampUS)
		for ax := 0; ax < 3; ax++ {
			ecris(uint64(p.Q[ax]))
			e.SommeQ[ax] += uint64(p.Q[ax])
		}
		slots[p.Slot] = true
		if i == 0 {
			e.PremierUS = p.TimestampUS
		}
		e.DernierUS = p.TimestampUS
	}
	e.N, e.Slots, e.Hash = len(pos), len(slots), h.Sum64()
	return e
}
