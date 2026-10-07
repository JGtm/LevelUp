package grammar

// zone_state_scan_images_cles_test.go — LA VOIE IMAGE-CLE DE ti=13, sur la mini-bobine versionnee
// (`bobineFamilles`, prefixe contigu de 000d5950 : registre et images-cles reels, en CI).

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// TestProprietesGereesImagesClesFermeesSeules : chaque record ti=13 d'image-cle se range dans
// exactement une case ; seules les lectures de records fermes sortent, toutes scalaires (mode A)
// et chainees ; la voie delta est intacte (ses lectures ne portent aucune lecture d'image-cle).
func TestProprietesGereesImagesClesFermeesSeules(t *testing.T) {
	film, err := source.LoadDir(bobineFamilles, nil)
	if err != nil {
		t.Fatalf("mini-bobine versionnee illisible (%s) : %v", bobineFamilles, err)
	}
	sc, err := ScanManagedProperties(NewFilmContext(film))
	if err != nil {
		t.Fatalf("balayage ti=13 : %v", err)
	}
	t.Logf("images-cles ti=13 : %d records, %d fermes, %d casses, %d non prouves, %d refuses, %d lectures",
		sc.KeyRecords, sc.KeyClosed, sc.KeyBroken, sc.KeyUnproven, sc.KeyRefused, len(sc.KeyReads))
	if sc.KeyRecords == 0 || sc.KeyClosed == 0 {
		t.Fatalf("aucun record ti=13 ferme aux images-cles de la mini-bobine (%d records) : la voie "+
			"image-cle ne lit rien", sc.KeyRecords)
	}
	if got := sc.KeyClosed + sc.KeyBroken + sc.KeyUnproven + sc.KeyRefused; got != sc.KeyRecords {
		t.Errorf("repartition %d != %d records : un record sans case", got, sc.KeyRecords)
	}
	if sc.KeyRefused != 0 {
		t.Errorf("%d record(s) ferme(s) refuse(s) a la relecture : la relecture a l etendue diverge de la marche",
			sc.KeyRefused)
	}
	// AU PLUS une valeur scalaire par record ferme : un record dont le second mot de taille est nul
	// n'ecrit aucun composant (cf. `consumeFullStateDefaultBlock`) et n'en porte donc aucune.
	if len(sc.KeyReads) == 0 || len(sc.KeyReads) > sc.KeyClosed {
		t.Errorf("%d lectures pour %d records fermes : attendu entre 1 et une par record ferme",
			len(sc.KeyReads), sc.KeyClosed)
	}
	// CHAQUE lecture porte le nom (`i0`) de son record, et un slot n en porte qu un : le nom est
	// l identite de la propriete, il ne change pas d une image-cle a l autre.
	noms := map[uint32]uint32{}
	for _, r := range sc.KeyReads {
		if r.Field != ManagedPropertyScalar || r.FilmIndex != -1 || !r.Chained {
			t.Fatalf("lecture d image-cle %+v : attendu scalaire, FilmIndex -1, chainee", r)
		}
		if !r.Named {
			t.Fatalf("lecture d image-cle %+v sans nom : l etat complet porte toujours `i0`", r)
		}
		if n, ok := noms[r.Slot]; ok && n != r.Name {
			t.Fatalf("slot %d : deux noms (%d puis %d)", r.Slot, n, r.Name)
		}
		noms[r.Slot] = r.Name
	}
	// La voie delta ne recolte pas `i0` : aucune de ses lectures n est nommee.
	for _, r := range sc.Reads {
		if r.Named {
			t.Fatalf("lecture delta %+v nommee : seule la voie image-cle recolte le nom", r)
		}
	}
}
