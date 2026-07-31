// tmp_wicons — redimensionne les visuels d'armes et les encode en data: URI pour la page
// de demonstration autonome (aucune requete externe possible). THROWAWAY.
//
// La correspondance nom -> fichier est EXPLICITE : jamais de rapprochement approximatif.
// Une arme sans fichier n'a pas d'icone (repli texte) — afficher un mauvais fusil est pire
// qu'un libelle.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sort"
)

const assetsDir = "C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/static/weapons-assets/halo_infinite"
const outJSON = "C:/Users/GUILLA~1/AppData/Local/Temp/claude/C--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration--claude-worktrees-filmdec-continuation/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad/weapon_icons.json"

// targetH : hauteur cible en pixels. L'icone s'affiche a 17 px : 34 px la rend nette sur un
// ecran a densite double sans peser.
const targetH = 34

// iconFile — nom canonique d'arme (catalogue de production weaponv3) -> fichier PNG.
var iconFile = map[string]string{
	"BR75":           "BR75.png",
	"Bandit Evo":     "Bandit.png",
	"CQS48 Bulldog":  "Bulldog.png",
	"Disruptor":      "Disruptor.png",
	"Energy Sword":   "Sword.png",
	"Gravity Hammer": "Hammer.png",
	"Heatwave":       "Heatwave.png",
	"M41 SPNKr":      "M41.png",
	"MA40 AR":        "MA40.png",
	"MLRS-2 Hydra":   "Hydra.png",
	"Mangler":        "Mangler.png",
	"Mk51 Sidekick":  "Sidekick.png",
	"Mutilator":      "Mutilator.png",
	"Needler":        "Needler-1.png",
	"Plasma Pistol":  "Plasma.png",
	"Pulse Carbine":  "Carabine.png",
	"Ravager":        "Ravager.png",
	"S7 Sniper":      "Sniper-S7.png",
	"Sentinel Beam":  "Sentinel.png",
	"Shock Rifle":    "Shock-rifle.png",
	"Skewer":         "Skewer.png",
	"Stalker Rifle":  "Stalker.png",
	"VK78 Commando":  "Commando.png",
	"Frag Grenade":   "Grenade.png",
	"Plasma Grenade": "Grenade.png",
	"Dynamo Grenade": "Grenade.png",
	// SANS VISUEL, volontairement : "Cindershot" (aucun fichier ne la represente ;
	// Cremator et Mutilator sont d'autres armes), "MA5K Avenger", "Mythic Sandwich",
	// "Sandwich".
}

func main() {
	out := map[string]string{}
	total := 0
	var names []string
	for n := range iconFile {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		src, err := loadPNG(filepath.Join(assetsDir, iconFile[name]))
		if err != nil {
			fmt.Println("LU KO", name, err)
			continue
		}
		blob, w, h, err := encodeScaled(src, targetH)
		if err != nil {
			fmt.Println("ENCODE KO", name, err)
			continue
		}
		uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString(blob)
		out[name] = uri
		total += len(uri)
		fmt.Printf("%-16s %-16s %3dx%-3d  %5d o png  %5d o base64\n",
			name, iconFile[name], w, h, len(blob), len(uri))
	}
	fmt.Printf("=== %d icones, %d octets de data: URI au total ===\n", len(out), total)
	blob, _ := json.Marshal(out)
	if err := os.WriteFile(outJSON, blob, 0o644); err != nil {
		panic(err)
	}
}

func loadPNG(p string) (image.Image, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

// encodeScaled reduit l'image a la hauteur h par MOYENNE DE BOITE (chaque pixel de sortie est
// la moyenne des pixels source qu'il couvre, en alpha premultiplie pour ne pas halo les bords
// transparents), puis encode en PNG.
func encodeScaled(src image.Image, h int) ([]byte, int, int, error) {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sh <= 0 || sw <= 0 {
		return nil, 0, 0, fmt.Errorf("image vide")
	}
	w := sw * h / sh
	if w < 1 {
		w = 1
	}
	rgba := image.NewNRGBA(image.Rect(0, 0, sw, sh))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)

	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		y0, y1 := y*sh/h, (y+1)*sh/h
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < w; x++ {
			x0, x1 := x*sw/w, (x+1)*sw/w
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var sr, sg, sb, sa, n float64
			for yy := y0; yy < y1; yy++ {
				for xx := x0; xx < x1; xx++ {
					c := rgba.NRGBAAt(xx, yy)
					a := float64(c.A) / 255
					sr += float64(c.R) * a
					sg += float64(c.G) * a
					sb += float64(c.B) * a
					sa += a
					n++
				}
			}
			if n == 0 || sa == 0 {
				continue
			}
			dst.SetNRGBA(x, y, color.NRGBA{
				R: uint8(sr/sa + 0.5), G: uint8(sg/sa + 0.5), B: uint8(sb/sa + 0.5),
				A: uint8(255*sa/n + 0.5),
			})
		}
	}
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, dst); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), w, h, nil
}
