package assets

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pngBytes est un en-tête PNG suffisant pour que LookupBinary serve le fichier.
var pngBytes = []byte("\x89PNG\r\n\x1a\nsecret")

// trappedRefs : chaque composant d'une Ref qui vient d'une requête (titre, identifiant,
// variante) porté par une traversée, un séparateur Windows, un lecteur ou un octet nul.
func trappedRefs() map[string]Ref {
	up := strings.Repeat("../", 4) // remonte au-dessus de la racine du stock
	deep := "/" + up + "secret"
	return map[string]Ref{
		"variante remontante":          {Kind: KindMapImage, TitleID: "halo_infinite", ID: "abc", Variant: deep},
		"variante relative":            {Kind: KindMapImage, TitleID: "halo_infinite", ID: "abc", Variant: "x/" + up + "secret"},
		"variante antislash":           {Kind: KindMapImage, TitleID: "halo_infinite", ID: "abc", Variant: `\..\..\..\..\secret`},
		"variante point-point":         {Kind: KindMapImage, TitleID: "halo_infinite", ID: "abc", Variant: ".."},
		"variante lecteur":             {Kind: KindMapImage, TitleID: "halo_infinite", ID: "abc", Variant: "C:secret"},
		"variante octet nul":           {Kind: KindMapImage, TitleID: "halo_infinite", ID: "abc", Variant: "v\x00"},
		"identifiant remontant":        {Kind: KindMapImage, TitleID: "halo_infinite", ID: "../../../secret"},
		"identifiant antislash":        {Kind: KindMapImage, TitleID: "halo_infinite", ID: `..\..\..\secret`},
		"identifiant point-point seul": {Kind: KindMapImage, TitleID: "halo_infinite", ID: ".."},
		"titre remontant":              {Kind: KindMapImage, TitleID: "../../..", ID: "secret"},
		"titre point-point":            {Kind: KindMapImage, TitleID: "..", ID: "secret"},
		"titre à barre oblique":        {Kind: KindMapImage, TitleID: "a/b", ID: "secret"},
		"chemin de passe remontant":    {Kind: KindBPTrackImage, TitleID: "halo_infinite", ID: "tracks/" + up + "secret", Variant: "tracks"},
	}
}

// Une Ref piégée ne lit ni n'écrit jamais hors de la racine du stock, même sans proxy
// devant le serveur : Path la refuse, LookupBinary ne sert rien, PersistBinary n'écrit rien.
func TestLocalFSStore_TrappedRefNeverLeavesTheRoot(t *testing.T) {
	for name, ref := range trappedRefs() {
		t.Run(name, func(t *testing.T) {
			base := t.TempDir()
			root := filepath.Join(base, "cache")
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(base, "secret.png"), pngBytes, 0o600); err != nil {
				t.Fatal(err)
			}
			s := NewLocalFSStore(root)

			got, err := s.LookupBinary(context.Background(), ref)
			if got != nil {
				t.Fatalf("LookupBinary(%s) a servi %q : lecture hors de la racine", ref, got.Bytes)
			}
			if err == nil {
				t.Errorf("LookupBinary(%s) : erreur attendue (référence refusée), obtenu nil", ref)
			}

			_ = s.PersistBinary(context.Background(), ref, BinaryPayload{Bytes: pngBytes})
			assertNothingWrittenOutside(t, base, root)
		})
	}
}

// assertNothingWrittenOutside vérifie que base ne contient, hors de root, que secret.png.
func assertNothingWrittenOutside(t *testing.T, base, root string) {
	t.Helper()
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == base || path == root || strings.HasPrefix(path, root+string(filepath.Separator)) {
			return nil
		}
		if d.Name() == "secret.png" && filepath.Dir(path) == base {
			return nil
		}
		t.Errorf("écrit hors de la racine : %s", path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Les identifiants légitimes restent servis : numéro de médaille, UUID de carte et de
// version, chemin GameCMS (barres obliques, barre de tête), badge de défi.
func TestLocalFSStore_LegitimateRefsAreServed(t *testing.T) {
	legit := []Ref{
		{Kind: KindMedalImage, TitleID: "halo_infinite", ID: "1512363953"},
		{Kind: KindMapImage, TitleID: "halo_infinite", ID: "298bff6c-79e6-4d2f-a3b5-bd8ec5b02e25",
			Variant: "a0b1c2d3-e4f5-4a6b-8c7d-9e0f1a2b3c4d"},
		{Kind: KindBPTrackImage, TitleID: "halo_infinite", ID: "/progression/Tracks/a.png", Variant: "tracks"},
		{Kind: KindSpartanEmblem, TitleID: "halo_5", ID: "Inventory/Emblems/x y.png"},
		{Kind: KindChallengeBadge, TitleID: "halo_infinite", ID: "combat-x..y"},
	}
	root := t.TempDir()
	s := NewLocalFSStore(root)
	for _, ref := range legit {
		if err := s.PersistBinary(context.Background(), ref, BinaryPayload{Bytes: pngBytes}); err != nil {
			t.Fatalf("PersistBinary(%s) : %v", ref, err)
		}
		got, err := s.LookupBinary(context.Background(), ref)
		if err != nil || got == nil {
			t.Fatalf("LookupBinary(%s) = %v, %v ; attendu le fichier écrit", ref, got, err)
		}
	}
}
