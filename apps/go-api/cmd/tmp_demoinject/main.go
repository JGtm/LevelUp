// tmp_demoinject — injecte la STRUCTURE de carte de l'artefact de rejeu dans le bloc de
// données de la page de démonstration (scratchpad/replay_demo.html), en remplacement des
// props Forge (`objects`).
//
// La page a été éditée à la main : ce binaire ne la RÉGÉNÈRE PAS. Il remplace uniquement le
// contenu de <script id="replay-data">, et à l'intérieur uniquement les clés `objects`
// (supprimée) et `structure` (ajoutée) ; toutes les autres valeurs sont recopiées OCTET POUR
// OCTET (json.RawMessage), donc ni réarrondies ni reformatées.
//
// Usage : go run ./cmd/tmp_demoinject <artefact.json> <page.html>
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/analysis/replay"
)

// demoMargin : marge (m) autour de la boîte des positions joueur. La page cadre sur cette
// boîte ; une emprise qui n'y touche pas ne serait jamais dessinée.
const demoMargin = 6.0

const dataOpen = `<script id="replay-data" type="application/json">`
const dataClose = `</script>`

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: tmp_demoinject <artefact.json> <page.html>")
		os.Exit(2)
	}
	doc, err := loadDoc(os.Args[1])
	if err != nil {
		fmt.Println("artefact:", err)
		os.Exit(1)
	}
	page, err := os.ReadFile(os.Args[2])
	if err != nil {
		fmt.Println("page:", err)
		os.Exit(1)
	}
	html := string(page)
	i := strings.Index(html, dataOpen)
	if i < 0 {
		fmt.Println("bloc de données introuvable dans la page")
		os.Exit(1)
	}
	start := i + len(dataOpen)
	rel := strings.Index(html[start:], dataClose)
	if rel < 0 {
		fmt.Println("fin du bloc de données introuvable")
		os.Exit(1)
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(html[start:start+rel]), &fields); err != nil {
		fmt.Println("données de la page:", err)
		os.Exit(1)
	}
	kept, dropped := filterSurfaces(doc.Structure, doc.Bounds)
	blob, err := json.Marshal(kept)
	if err != nil {
		fmt.Println("sérialisation structure:", err)
		os.Exit(1)
	}
	delete(fields, "objects") // les props Forge ne sont plus le fond de carte
	fields["structure"] = blob
	out, err := json.Marshal(fields)
	if err != nil {
		fmt.Println("sérialisation données:", err)
		os.Exit(1)
	}
	page = []byte(html[:start] + string(out) + html[start+rel:])
	if err := os.WriteFile(os.Args[2], page, 0o644); err != nil {
		fmt.Println("écriture:", err)
		os.Exit(1)
	}
	fmt.Printf("structure injectée : %d emprises retenues, %d hors cadre écartées ; page %d octets\n",
		len(kept), dropped, len(page))
}

func loadDoc(path string) (replay.ReplayDocument, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return replay.ReplayDocument{}, err
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(blob, &doc); err != nil {
		return replay.ReplayDocument{}, err
	}
	if len(doc.Structure) == 0 {
		return replay.ReplayDocument{}, fmt.Errorf("l'artefact ne porte aucune structure")
	}
	return doc, nil
}

// filterSurfaces ne garde que les emprises qui touchent le cadre dessiné, et les publie en
// tableaux compacts [x0, y0, x1, y1, z_face_supérieure] — 10 000 objets JSON pèseraient
// trois fois plus dans une page déjà à 900 Ko.
func filterSurfaces(surf []replay.Surface, b replay.Bounds) ([][5]float32, int) {
	lo := [2]float32{b.MinX - demoMargin, b.MinY - demoMargin}
	hi := [2]float32{b.MaxX + demoMargin, b.MaxY + demoMargin}
	out := make([][5]float32, 0, len(surf))
	dropped := 0
	for _, s := range surf {
		if s.X1 < lo[0] || s.X0 > hi[0] || s.Y1 < lo[1] || s.Y0 > hi[1] {
			dropped++
			continue
		}
		out = append(out, [5]float32{s.X0, s.Y0, s.X1, s.Y1, s.Z})
	}
	return out, dropped
}
