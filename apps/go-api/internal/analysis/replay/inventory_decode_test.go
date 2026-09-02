package replay

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// inventory_decode_test.go — CE QU'UN CHUNK ILLISIBLE PRODUIT, ET CE QUE LE LOT 1 Y A CHANGÉ.
//
// AVANT LE LOT 1 (audit AUDIT_AVAL_INVENTAIRE_2026-08-24.md, point 3), chaque balayage lisait
// les chunks LUI-MÊME : un chunk illisible était sauté par un `continue` et compté dans
// `Stats.ChunksUnread`, et le film sortait avec une FRACTION de ses images-clés — d'où ce
// compteur, réclamé par l'audit pour que la fraction se voie.
//
// DEPUIS LE LOT 1 (PLAN_CUISSON_PERF item 1.2, 2026-09-02), le film est chargé UNE fois par
// `filmsource.LoadDir` avant tout balayage, et un chunk illisible fait ÉCHOUER CE CHARGEMENT.
// La dégradation silencieuse devient donc un REFUS EXPLICITE : la cuisson ne produit plus un
// artefact amputé qui se lirait comme un film pauvre, elle s'arrête et le dit. `ChunksUnread`
// reste publié (il entre dans l'empreinte de l'étape `inventory.stats`) et vaut désormais zéro
// sur un film chargé — un chunk absent du film ne peut plus être demandé par le balayage.
//
// `TestMiniFilmDecodesTheKeyframes` (minifilm_test.go) couvre le cas nominal.

// filmDirWithBadChunk construit un répertoire de film minimal portant UN chunk illisible :
// `chunk_%02d.bin` est un RÉPERTOIRE, pas un fichier. Le glob de `filmsource.DirSource` le voit
// (c'est bien un `chunk_NN.bin`), mais `os.ReadFile` échoue dessus sur toutes les plateformes —
// c'est le geste le plus portable pour simuler une lecture disque qui échoue sans dépendre de
// permissions.
func filmDirWithBadChunk(t *testing.T, goodChunks int) string {
	t.Helper()
	dir := t.TempDir()
	for i := 1; i <= goodChunks; i++ {
		p := filepath.Join(dir, chunkFileName(i))
		// Contenu vide : WalkPackets rend une liste vide sur un chunk trop court pour un
		// en-tête de paquet — aucune image-clé, aucun panic.
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatalf("écriture %s : %v", p, err)
		}
	}
	badPath := filepath.Join(dir, chunkFileName(goodChunks+1))
	if err := os.Mkdir(badPath, 0o755); err != nil {
		t.Fatalf("création du chunk illisible %s : %v", badPath, err)
	}
	return dir
}

// chunkFileName reproduit le nommage de filmdec.ReadFilmChunk (`chunk_%02d.bin`) — non
// exporté du paquet voisin, donc redéclaré ici plutôt qu'emprunté (même règle de frontière
// que inventory_decode.go, cf. son en-tête).
func chunkFileName(n int) string {
	return fmt.Sprintf("chunk_%02d.bin", n)
}

// TestScanFilmKeyframeInventoryRefuseUnChunkIllisible : un chunk illisible AU MILIEU d'un film
// par ailleurs sain fait échouer le CHARGEMENT, donc le balayage — il n'est plus sauté.
//
// C'EST UN CHANGEMENT DE COMPORTEMENT DU LOT 1, ET IL EST VOULU : décoder une fois exige de
// charger le film une fois, et un film dont un chunk manque n'est pas le film. Le refus arrive
// tôt et bruyamment (`replaybuild.chargerFilm` le journalise, la cuisson finit sur
// `ErrNoTracks`) au lieu de produire un artefact amputé indiscernable d'un film pauvre.
func TestScanFilmKeyframeInventoryRefuseUnChunkIllisible(t *testing.T) {
	dir := filmDirWithBadChunk(t, 2)
	known := map[uint32]bool{1: true}
	inv, st, err := ScanFilmKeyframeInventory(dir, known, 0)
	if err == nil {
		t.Fatal("chunk illisible : une erreur était attendue — un film amputé n'est pas un film")
	}
	if inv != nil {
		t.Errorf("inventaire non nil malgré l'échec de chargement : %+v", inv)
	}
	if st != (KeyframeInventoryStats{}) {
		t.Errorf("Stats = %+v, attendu la valeur zéro : aucun chunk n'a été balayé", st)
	}
}

// TestScanFilmKeyframeInventoryAllChunksUnreadable : un répertoire dont AUCUN chunk n'est
// lisible échoue aussi — comportement inchangé, par une autre voie (le chargement, et non le
// compteur `ChunksUnread`).
func TestScanFilmKeyframeInventoryAllChunksUnreadable(t *testing.T) {
	dir := filmDirWithBadChunk(t, 0)
	known := map[uint32]bool{1: true}
	inv, _, err := ScanFilmKeyframeInventory(dir, known, 0)
	if err == nil {
		t.Fatal("aucun chunk lisible : une erreur était attendue")
	}
	if inv != nil {
		t.Errorf("inventaire non nil malgré l'échec total : %+v", inv)
	}
}

// TestScanKeyframeInventoryCompteLesChunks : sur un film CHARGÉ, `Stats.Chunks` compte les
// chunks de données et `ChunksUnread` reste à zéro — le dénominateur que l'audit réclamait
// continue d'être publié, sur la grandeur qui a encore un sens après le lot 1.
func TestScanKeyframeInventoryCompteLesChunks(t *testing.T) {
	dir := t.TempDir()
	for i := 1; i <= 3; i++ {
		if err := os.WriteFile(filepath.Join(dir, chunkFileName(i)), nil, 0o644); err != nil {
			t.Fatalf("écriture du chunk %d : %v", i, err)
		}
	}
	inv, st, err := ScanFilmKeyframeInventory(dir, map[uint32]bool{1: true}, 0)
	if err != nil {
		t.Fatalf("ScanFilmKeyframeInventory : %v", err)
	}
	if len(inv) != 0 {
		t.Errorf("%d inventaire(s) décodé(s) sur des chunks vides, attendu 0", len(inv))
	}
	if st.Chunks != 3 || st.ChunksUnread != 0 {
		t.Errorf("Stats = %+v, attendu Chunks=3 ChunksUnread=0", st)
	}
	if st.Keyframes != 0 || st.Records != 0 {
		t.Errorf("Stats = %+v, attendu aucune image-clé et aucun record (chunks vides)", st)
	}
}
