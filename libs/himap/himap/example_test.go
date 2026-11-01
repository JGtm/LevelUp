package himap_test

import (
	"context"
	"image/png"
	"log"
	"os"

	"github.com/JGtm/LevelUp/libs/himap/himap"
)

// Render the top-down background image of a native map from its .module. The caller supplies the
// anchors (world positions of the map objectives): they frame the image and calibrate the play
// level. The map catalogue that knows those positions is not part of this module.
func ExampleCuitCarteNative() {
	deploy := "/path/to/Halo Infinite/deploy" // your install's deploy folder, or let himap.DeployRoot discover it
	r, bilan, err := himap.CuitCarteNative(context.Background(), himap.OptionsCuisson{
		RacineDeploy: deploy,
		CheminModule: deploy + "/pc/levels/multi/ridgeline/ridgeline-rtx-new.module",
		Ancres:       [][3]float64{{12.5, -40.0, 3.0}, {-60.2, 18.7, 2.5}}, // objectives, world metres
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d instances drawn, %d degradations", bilan.Dessinees, len(bilan.Degradations))

	img := himap.FondPNG(r, r.NX, r.NY, nil, himap.StyleFondParDefaut)
	out, err := os.Create("ridgeline.png")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		log.Fatal(err)
	}
	x0, y0, mpp := himap.CalageDuRendu(r) // pixel -> world: x = x0 + (px+0.5)*mpp, y = y0 - (py+0.5)*mpp
	log.Printf("origin (%.1f, %.1f), %.4f m/px", x0, y0, mpp)
}
