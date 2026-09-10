package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain/title"
)

// TestReadBackgroundImage_MimeSuitLeSidecar prouve, sur un sidecar SYNTHETIQUE, que le type
// MIME servi est deduit du nom de fichier porte par le champ `image` (D3 du plan fonds WebP)
// et non d'une extension supposee. Ecrit apres la revue de la vague 3 (2026-09-10) : jusque-la
// le chemin `.webp` de readBackgroundImage n'etait exerce que par les donnees reelles du
// depot — un code qui aurait toujours rendu image/png serait reste vert en CI. Le corps du
// fichier n'a pas besoin d'etre un WebP valide : le service ne decode pas, il sert des octets
// et un Content-Type, et c'est ce couple qui est verifie ici. Mutation prouvee : forcer
// mimeImagePNG dans readBackgroundImage fait rougir le cas WebP.
func TestReadBackgroundImage_MimeSuitLeSidecar(t *testing.T) {
	cas := []struct {
		nom   string
		image string
		corps string
		mime  string
	}{
		{"webp", "ridgeline.webp", "RIFF\x00\x00\x00\x00WEBPVP8Lfaux", mimeImageWebP},
		{"png", "ridgeline.png", "\x89PNG\r\n\x1a\nfaux", mimeImagePNG},
		{"extension en majuscules", "ridgeline.WEBP", "RIFF\x00\x00\x00\x00WEBPVP8Lfaux", mimeImageWebP},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			root := t.TempDir()
			res := title.NewPathResolver(root)
			ecrire(t, res.MapBackgroundMetaPath(title.DefaultSlug, "ridgeline"),
				`{"schemaVersion":1,"module":"ridgeline","mapNames":["Cliffhanger"],`+
					`"image":"`+c.image+`",`+
					`"source":"test","generatedAt":"2026-09-10T10:00:00Z","style":"jeu",`+
					`"calibration":{"metersPerPixel":0.092,"originX":0,"originY":0,`+
					`"widthPx":100,"heightPx":100,"convention":"test"},`+
					`"stats":{"anchors":1}}`)
			ecrire(t, res.MapBackgroundImageFilePath(title.DefaultSlug, c.image), c.corps)

			svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{names: []string{"Cliffhanger"}})
			blob, mime, err := svc.MapBackgroundImage(context.Background(), "m1")
			if err != nil {
				t.Fatalf("image=%q : err = %v", c.image, err)
			}
			if mime != c.mime {
				t.Errorf("image=%q : mime = %q, attendu %q", c.image, mime, c.mime)
			}
			if string(blob) != c.corps {
				t.Errorf("image=%q : le service doit servir les octets du fichier tels quels", c.image)
			}
		})
	}
}
