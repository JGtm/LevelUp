package replay

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// inventory_decode_test.go — LA TÉLÉMÉTRIE DE COUVERTURE DU DÉCODEUR (audit
// AUDIT_AVAL_INVENTAIRE_2026-08-24.md, point 3) : un chunk illisible ne doit plus disparaître
// sans laisser de trace. `TestMiniFilmDecodesTheKeyframes` (minifilm_test.go) couvre déjà le
// cas nominal (Stats.Records == len(inv), aucun chunk illisible) ; ce fichier couvre le cas
// dégradé qu'aucun test du paquet n'exerçait avant ce lot.

// filmDirWithBadChunk construit un répertoire de film minimal portant UN chunk illisible :
// `chunk_%02d.bin` est un RÉPERTOIRE, pas un fichier. `os.Stat` le voit (CountFilmChunks le
// compte donc), mais `os.ReadFile` échoue dessus sur toutes les plateformes — c'est le geste
// le plus portable pour simuler une lecture disque qui échoue sans dépendre de permissions.
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

// TestScanFilmKeyframeInventoryCountsUnreadableChunks : un chunk illisible AUTRE que la
// totalité du film est désormais COMPTÉ, pas seulement avalé par un `continue` nu. Le
// balayage réussit quand même (les chunks lisibles restants suffisent) — c'est la
// distinction que l'audit demandait : « le film peut sortir avec une fraction seulement de
// ses images-clés d'inventaire décodées, et rien ne le distingue d'un film sain avec moins
// de keyframes » devient mesurable.
func TestScanFilmKeyframeInventoryCountsUnreadableChunks(t *testing.T) {
	dir := filmDirWithBadChunk(t, 2)
	known := map[uint32]bool{1: true}
	inv, st, err := ScanFilmKeyframeInventory(dir, known, 0)
	if err != nil {
		t.Fatalf("ScanFilmKeyframeInventory : %v (au moins un chunk lisible, pas d'erreur attendue)", err)
	}
	if len(inv) != 0 {
		t.Errorf("%d inventaire(s) décodé(s) sur des chunks vides, attendu 0", len(inv))
	}
	if st.Chunks != 3 {
		t.Errorf("Stats.Chunks = %d, attendu 3 (2 lisibles + 1 illisible)", st.Chunks)
	}
	if st.ChunksUnread != 1 {
		t.Errorf("Stats.ChunksUnread = %d, attendu 1 — c'est exactement ce que l'audit reprochait "+
			"de ne voir NULLE PART (ni compteur, ni log)", st.ChunksUnread)
	}
	if st.Keyframes != 0 {
		t.Errorf("Stats.Keyframes = %d, attendu 0 (chunks vides, aucun paquet)", st.Keyframes)
	}
	if st.Records != len(inv) {
		t.Errorf("Stats.Records = %d, attendu %d (== len(inv), aucun filtrage dans keyframeInventories)",
			st.Records, len(inv))
	}
}

// TestScanFilmKeyframeInventoryAllChunksUnreadable : quand AUCUN chunk n'est lisible, l'erreur
// remonte (comportement inchangé), et la Stats le documente plutôt que de rester à zéro sans
// qu'on sache si c'est parce que le film est vide ou parce qu'il est corrompu.
func TestScanFilmKeyframeInventoryAllChunksUnreadable(t *testing.T) {
	dir := filmDirWithBadChunk(t, 0)
	known := map[uint32]bool{1: true}
	inv, st, err := ScanFilmKeyframeInventory(dir, known, 0)
	if err == nil {
		t.Fatal("aucun chunk lisible : une erreur était attendue")
	}
	if inv != nil {
		t.Errorf("inventaire non nil malgré l'échec total : %+v", inv)
	}
	if st.Chunks != 1 || st.ChunksUnread != 1 {
		t.Errorf("Stats = %+v, attendu Chunks=1 ChunksUnread=1", st)
	}
}
