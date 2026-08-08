// Command weapon-icons-build extrait des archives `.module` d'Halo Infinite les ICÔNES
// D'ARMES du jeu et les écrit en PNG dans static/, prêtes à être revues puis branchées.
//
// POURQUOI CE BINAIRE EXISTE, ET POURQUOI IL N'EST PAS JETABLE. Il produit un artefact
// VERSIONNÉ (les PNG de static/weapons-assets/halo_infinite/jeu/) : il tombe donc du bon
// côté du critère de purge appliqué en J3 lot A — un outil qui n'établit qu'une grammaire
// est jetable, un outil qui régénère un asset versionné reste. Il est le seul moyen de
// reproduire ces images après une mise à jour de contenu du jeu.
//
// OÙ. Sur une machine qui a Halo Infinite INSTALLÉ. Il lit les archives en LECTURE SEULE et
// n'écrit que dans le dossier de sortie. cgo requis (décompression Kraken via internal/ooz).
//
//	go run ./cmd/weapon-icons-build                       # racine du jeu auto-détectée
//	go run ./cmd/weapon-icons-build -deploy "D:/.../deploy" -out ../../static/...
//
// CE QUI EST ÉTABLI SUR PIÈCES (aucune de ces étapes n'est postulée) :
//
//  1. Les 29 `weap` du registre référencent tous DEUX bitmaps communs — mesuré sur les 29 :
//     bc17adf1 et e39747c8. Le troisième bitm varie mais se PARTAGE entre armes par groupes
//     (Mangler+Needler, MA40+MA5K, six armes Covenant ensemble) : c'est un réticule, pas une
//     icône. Les deux bitmaps communs, eux, déclarent 39 images de tailles TOUTES
//     différentes — un jeu d'icônes, et le seul candidat possible.
//
//  2. Le lien tag -> pixels : l'entrée .module porte à +0x10 un index dans la table de
//     ressources du module (0 dans la variante `ds/`, qui n'a pas les pixels ; non nul dans
//     `pc/`). Les 39 ressources qui suivent cet index correspondent une à une aux 39 images.
//
//  3. Le format : contrôle ARITHMÉTIQUE exact. Pour l'image 0 (333x117), la ressource pèse
//     53372 octets = en-tête 212 + données 72 + 53088, et 53088 = 40320 + 10080 + 2688,
//     soit mip0+mip1+mip2 en blocs 4x4 sur 16 octets. À l'octet près, sur toutes les images.
//
//  4. Le contenu : R constant à 255, G constant à 0, B = rampe verticale de teinte appliquée
//     au rendu, A = LE DESSIN. Seul l'alpha est extrait, rendu en blanc sur fond transparent
//     — même convention que static/abilities-assets.
//
//  5. QUELLE IMAGE DÉSIGNE QUELLE ARME — établi depuis, et c'est le champ `sprite index` du
//     bloc `UI display info` du tag `weap` (cf. weapui.go). 29 armes sur 29, chacune
//     auto-validée. Les fichiers restent nommés par leur INDEX d'atlas ; le weapon_key est
//     porté par index.json, ce qui évite de renommer 168 fichiers à chaque évolution.
//
// CE QUI RESTE AU GATE HUMAIN : l'atlas sandbox (véhicules, objectifs, grenades lancées,
// pictogrammes) n'a aucun lien structurel connu vers le registre, et les index de l'atlas
// d'armes qui ne correspondent à aucune arme de notre registre restent à nommer. Rien n'y est
// deviné : afficher un mauvais fusil est pire qu'un libellé.
//
// DEUX VOIES ÉCARTÉES, notées pour ne pas les re-tenter : chercher un petit entier à offset
// CONSTANT dans le corps du tag (0 candidat sur 29 — le corps est un arbre de structures),
// et l'appariement automatique par silhouette contre les icônes dessinées du dépôt (marges de
// 0,00 à 0,10 pour des scores de 0,44 à 0,90 : des armes toutes « longues et horizontales »
// se ressemblent trop une fois remplies).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// atlasTags : les deux bitmaps d'icônes communs à toutes les armes, et le style que chacun
// porte. Mesuré : même nombre d'images, mêmes dimensions index par index — ce sont les deux
// rendus du MÊME jeu d'icônes.
var atlasTags = []struct {
	ID    uint32
	Style string
}{
	{0xbc17adf1, "contour"},
	{0xe39747c8, "silhouette"},
	// Atlas « sandbox » : armes vues de plus loin (~110x38 contre ~330x117), véhicules,
	// grenades lancées et pictogrammes de mort. Les 88 images déclarées sortent toutes
	// propres depuis que scanImgs lit le tableau déclaré au lieu de chercher une signature.
	{0x0302cad3, "sandbox"},
}

// iconEntry décrit une icône extraite, telle qu'écrite dans index.json.
type iconEntry struct {
	Index      int    `json:"index"`
	Style      string `json:"style"`
	File       string `json:"file"`
	GameName   string `json:"nom_jeu,omitempty"` // nom INTERNE du jeu, craqué (cf. names.go)
	WeaponKey  string `json:"arme,omitempty"`    // le weapon_key du registre ; nomme `arme` car gitleaks voit un secret dans tout identifiant contenant « key » suivi d une chaine a forte entropie
	SourceTag  string `json:"source_tag"`
	SourceW    int    `json:"source_w"`
	SourceH    int    `json:"source_h"`
	CroppedW   int    `json:"cropped_w"`
	CroppedH   int    `json:"cropped_h"`
	BC7Format  int    `json:"bc7_format"`
	Verified   bool   `json:"align_verified"`
	Noise      string `json:"alpha_noise"`
	RebuiltPc  string `json:"bc7_rebuilt_pct"`
	OpaquePc   string `json:"bc7_opaque_pct"`
	DegradedPc string `json:"bc7_degraded_pct"`
}

func main() {
	deploy := flag.String("deploy", "", "racine `deploy` des archives du jeu (auto-détectée si vide)")
	out := flag.String("out", filepath.Join("..", "..", "static", "weapons-assets", "halo_infinite", "jeu"),
		"dossier de sortie des PNG")
	probe := flag.Int("probe", 6, "profondeur de la sonde de recalage descripteur -> ressource")
	maxIdx := flag.Int("max", 120, "nombre d'images à extraire par atlas")
	flag.Parse()
	probeWindow = *probe

	if *deploy != "" {
		if err := os.Setenv("HALO_DEPLOY", *deploy); err != nil {
			fmt.Fprintln(os.Stderr, "HALO_DEPLOY:", err)
			os.Exit(1)
		}
	}
	root := moduleRoot()
	if root == "" {
		fmt.Fprintln(os.Stderr, "archives du jeu introuvables : passer -deploy <chemin>")
		os.Exit(1)
	}
	if err := os.MkdirAll(*out, 0o750); err != nil {
		fmt.Fprintln(os.Stderr, "création du dossier de sortie:", err)
		os.Exit(1)
	}
	fmt.Printf("archives : %s\nsortie   : %s\n\n", root, *out)

	ix := buildIndex("")
	fmt.Printf("index : %d archives, %d entrées\n", ix.nMods, ix.nEntry)

	// Le lien arme -> index d'icône, lu dans les tags `weap` et auto-validé.
	icons, err := resolveWeaponIcons(ix)
	if err != nil {
		fmt.Fprintln(os.Stderr, "résolution arme -> icône:", err)
		os.Exit(1)
	}
	keyByIndex := make(map[int]string, len(icons))
	for _, w := range icons {
		keyByIndex[w.Index] = w.Key
	}
	// Les 3 manquantes sont les grenades : elles ne sont pas des `weap` (elles vivent en
	// `eqip` + `proj`, déclarés par le tag de globals `gggl`) et n'ont donc pas de bloc
	// `UI display info` à lire. C'est attendu, pas un échec.
	fmt.Printf("armes résolues : %d/%d familles (les 3 manquantes sont les grenades)\n",
		len(icons), len(weaponFamilies()))

	// Le nom INTERNE de chaque icône, craqué contre les chaînes du binaire du jeu. Bonus :
	// son absence ne fait pas échouer l'extraction.
	indexOwned := make(map[int]bool, len(icons))
	for _, w := range icons {
		indexOwned[w.Index] = true
	}
	canon := make(map[uint32]bool, len(icons))
	for _, w := range weaponFamilies() {
		canon[w.ID] = true
	}
	names, err := resolveIconNames(ix, canon, indexOwned)
	if err != nil {
		fmt.Fprintln(os.Stderr, "noms d'icônes:", err)
		os.Exit(1)
	}
	fmt.Printf("noms craqués   : %d index de l'atlas d'armes\n\n", len(names))

	var entries []iconEntry
	for _, at := range atlasTags {
		n := 0
		limit := *maxIdx
		for idx := 0; idx < limit; idx++ {
			img, st, im, err := decodeAlphaGlyph(ix, at.ID, idx)
			if err != nil {
				fmt.Printf("%-11s arrêt à #%d : %v\n", at.Style, idx, err)
				break
			}
			name := fmt.Sprintf("%s-%02d.png", at.Style, idx)
			if err := writePNG(filepath.Join(*out, name), img); err != nil {
				fmt.Fprintln(os.Stderr, "écriture:", err)
				os.Exit(1)
			}
			// Le weapon_key ne vaut que pour les deux atlas d'armes : l'atlas sandbox a
			// sa propre numérotation, sans rapport avec `sprite index`.
			key, gameName := "", ""
			if at.ID == 0xbc17adf1 || at.ID == 0xe39747c8 {
				key = keyByIndex[idx]
				// Plusieurs noms = plusieurs `weap` revendiquent l'index (variantes de
				// campagne à index périmé). Tous sont publiés : arbitrer donnerait une
				// étiquette fausse avec l'apparence d'une certitude.
				gameName = strings.Join(names[idx], " | ")
			}
			entries = append(entries, iconEntry{
				Index: idx, Style: at.Style, File: name, WeaponKey: key, GameName: gameName,
				SourceTag: fmt.Sprintf("%08x", at.ID),
				SourceW:   im.W, SourceH: im.H,
				CroppedW: img.Bounds().Dx(), CroppedH: img.Bounds().Dy(),
				BC7Format: im.Format, Verified: imageVerified(ix, at.ID, idx),
				Noise:      fmt.Sprintf("%.4f", alphaNoise(img)),
				RebuiltPc:  pct(st.Rebuilt, st.Blocks),
				OpaquePc:   pct(st.Opaque, st.Blocks),
				DegradedPc: pct(st.Degraded, st.Blocks),
			})
			n++
		}
		fmt.Printf("%-11s %d icônes écrites\n", at.Style, n)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Index != entries[j].Index {
			return entries[i].Index < entries[j].Index
		}
		return entries[i].Style < entries[j].Style
	})
	blob, err := json.MarshalIndent(entries, "", " ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "json:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*out, "index.json"), append(blob, '\n'), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "index.json:", err)
		os.Exit(1)
	}
	fmt.Printf("\n%d entrées -> %s\n", len(entries), filepath.Join(*out, "index.json"))
}

// pct rend un pourcentage lisible, ou "0.00" si le dénominateur est nul.
func pct(n, total int) string {
	if total <= 0 {
		return "0.00"
	}
	return fmt.Sprintf("%.2f", 100*float64(n)/float64(total))
}
