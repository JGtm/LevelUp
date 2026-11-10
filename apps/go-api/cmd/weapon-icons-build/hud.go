package main

// hud.go — les vignettes du HUD du jeu (grenades, capacités d'armure).
//
// CE QUE C'EST. Des masques blancs + alpha, que le client teint (cf. `icon` de
// config/titles/halo_infinite/mappings/replay_labels.toml, servi sous
// static/weapons-assets/halo_infinite/). Ils sortent de trois atlas d'interface du jeu : les
// capacités, les équipements et un petit atlas de grenades au même dessin au trait que les
// capacités.
// Les chemins de sortie sont ceux des fichiers déjà servis : les URL sont cuites dans les
// rejeux, les changer obligerait à recuire.
//
// POURQUOI UNE TABLE EXPLICITE. L'ordre d'un atlas n'est pas celui qu'on imagine (le
// propulseur est à l'index 0 de l'atlas des capacités, la fragmentation à l'index 9 de celui
// des équipements) : chaque ligne a été fixée en comparant le glyphe décodé à la vignette
// qu'elle remplace, jamais d'après un rang supposé. Dynamo et Pointes viennent de l'atlas de
// grenades (index 3 et 2) : le glyphe de l'atlas des équipements (index 11 et 12) est un autre
// dessin, qui ne ressemble pas à ces deux grenades. Frag et Plasma restent sur l'atlas des
// équipements, validés à l'écran.
//
// CE QUI EST EXIGÉ. L'appariement de chaque image doit être confirmé par le contrôle
// arithmétique (`align_verified`) : une image déduite du seul rang n'est pas écrite.

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
)

const (
	// abilityAtlasTag : atlas des capacités d'armure (18 images de 100x100).
	abilityAtlasTag = 0xc6b7ad41
	// equipmentAtlasTag : atlas des équipements et des grenades (21 images de 128x128).
	equipmentAtlasTag = 0x7dcd396a
	// grenadeAtlasTag : atlas de cinq grenades au trait, 128x128 (fragmentation, plasma,
	// pointes, dynamo, et un losange).
	grenadeAtlasTag = 0xa2dcd69e
)

// hudIcon relie un fichier servi à l'image d'un atlas. File est relatif à la racine
// `weapons-assets/halo_infinite`.
type hudIcon struct {
	File  string
	Atlas uint32
	Index int
}

// hudIcons est la table complète : les 15 vignettes `hud/`, seules servies en masque à teindre.
// Grenade.png et Melee.png ne sont pas ici : ils sont servis en image colorée non teintée.
var hudIcons = []hudIcon{
	{"hud/ActiveCamouflage.png", abilityAtlasTag, 5},
	{"hud/DropWall.png", abilityAtlasTag, 2},
	{"hud/Dynamo.png", grenadeAtlasTag, 3},
	{"hud/Frag.png", equipmentAtlasTag, 9},
	{"hud/Grappleshot.png", abilityAtlasTag, 1},
	{"hud/Overshield.png", abilityAtlasTag, 6},
	{"hud/Plasma.png", equipmentAtlasTag, 10},
	{"hud/QuantumTranslocator.png", abilityAtlasTag, 15},
	{"hud/RepairField.png", abilityAtlasTag, 17},
	{"hud/Repulsor.png", abilityAtlasTag, 4},
	{"hud/ShroudScreen.png", abilityAtlasTag, 9},
	{"hud/Spike.png", grenadeAtlasTag, 2},
	{"hud/ThreatSeeker.png", abilityAtlasTag, 16},
	{"hud/ThreatSensor.png", abilityAtlasTag, 3},
	{"hud/Thruster.png", abilityAtlasTag, 0},
}

// squareCanvas pose le glyphe au centre d'un canevas carré de la taille de son plus grand
// côté : les vignettes servies sont carrées, et le client les loge dans une boîte fixe.
func squareCanvas(img *image.NRGBA) *image.NRGBA {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	side := max(w, h)
	out := image.NewNRGBA(image.Rect(0, 0, side, side))
	ox, oy := (side-w)/2, (side-h)/2
	for y := 0; y < h; y++ {
		so := img.PixOffset(0, y)
		do := out.PixOffset(ox, oy+y)
		copy(out.Pix[do:do+4*w], img.Pix[so:so+4*w])
	}
	return out
}

// writeHud décode chaque ligne de hudIcons et l'écrit sous root. Une image dont
// l'appariement n'est pas vérifié, ou vide, arrête l'écriture.
func writeHud(ix *tagIndex, root string) error {
	for _, ic := range hudIcons {
		img, st, im, err := decodeAlphaGlyph(ix, ic.Atlas, ic.Index)
		if err != nil {
			return fmt.Errorf("%s (atlas %08x #%d) : %w", ic.File, ic.Atlas, ic.Index, err)
		}
		if !imageVerified(ix, ic.Atlas, ic.Index) {
			return fmt.Errorf("%s (atlas %08x #%d) : appariement non vérifié", ic.File, ic.Atlas, ic.Index)
		}
		if img.Bounds().Dx() < 2 || img.Bounds().Dy() < 2 {
			return fmt.Errorf("%s (atlas %08x #%d) : image vide", ic.File, ic.Atlas, ic.Index)
		}
		path := filepath.Join(root, filepath.FromSlash(ic.File))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return err
		}
		out := squareCanvas(img)
		if err := writePNG(path, out); err != nil {
			return fmt.Errorf("%s : %w", ic.File, err)
		}
		fmt.Printf("hud %-30s %08x #%02d  source %dx%d  écrit %dx%d  blocs dégradés %d/%d\n",
			ic.File, ic.Atlas, ic.Index, im.W, im.H, out.Bounds().Dx(), out.Bounds().Dy(), st.Degraded, st.Blocks)
	}
	return nil
}
