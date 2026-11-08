package killcollector

import (
	"context"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmrun"
	"levelup/go-api/internal/observability"
)

// TestPrendreLeVerrouDeDecodage : l'etape 1.57 decode DANS le processus du serveur ; elle prend le verrou solo de la
// machine (le meme que la cuisson apres synchronisation et la file de regeneration des rejeux). Occupe = reportee, comptee,
// jamais une erreur ; libre = prise, et le verrou se rend.
func TestPrendreLeVerrouDeDecodage(t *testing.T) {
	racine := t.TempDir()
	h := NewPostSyncHook(racine, 0)
	d := PostSyncDeps{TitleSlug: "halo_infinite", Gamertag: "GT"}
	verrou := titlePkg.NewPathResolver(racine).CacheRootDir()

	tenu, err := filmrun.AcquireSolo(verrou, "file de regeneration", "00000001")
	if err != nil {
		t.Fatal(err)
	}
	avant := observability.LoadCounter(CompteurPostSyncReporteeVerrou)
	if liberer, ok := h.prendreLeVerrouDeDecodage(context.Background(), d); ok || liberer != nil {
		t.Fatal("un decodage tient la machine : l'etape ne doit pas decoder")
	}
	if observability.LoadCounter(CompteurPostSyncReporteeVerrou) != avant+1 {
		t.Fatal("le report se compte")
	}

	tenu.Release()
	liberer, ok := h.prendreLeVerrouDeDecodage(context.Background(), d)
	if !ok {
		t.Fatal("verrou libre : l'etape decode")
	}
	// Tenu par l'etape, il refuse un second decodage...
	if second, err := filmrun.AcquireSolo(verrou, "file de regeneration", "00000002"); err == nil {
		second.Release()
		t.Fatal("deux decodages en meme temps : le verrou n'est pas tenu par l'etape")
	}
	// ... et il se rend.
	liberer()
	rendu, err := filmrun.AcquireSolo(verrou, "file de regeneration", "00000003")
	if err != nil {
		t.Fatalf("verrou non rendu : %v", err)
	}
	rendu.Release()
}
