// Package mapdecoupe — DÉCOUPER une zone nommée sur le décor réellement praticable.
//
// LE DÉFAUT CORRIGÉ. Les polygones de callout sont les pavés du DESIGNER, lus dans le tag
// levl : des boîtes qui débordent largement sur le vide autour de la carte. L'utilisateur
// l'a signalé au gate (« les zones dépassent sur le grand fond transparent, zone
// inaccessible sans mourir »). Le décor réel, lui, est déjà publié : le fond de carte
// `map_backgrounds/{module}.png` a un canal alpha, et **alpha 0 = pas de matière**.
//
// POURQUOI LE MASQUE PUBLIÉ ET PAS UNE RE-CUISSON. Le raster `himap.Rendu` reste la source
// de vérité de la cuisson, mais il exige les fichiers du jeu ET la chaîne ooz (GPLv3). Le
// PNG est sa projection VERSIONNÉE : découper dessus est hors ligne pur, reproductible, et
// suit automatiquement les cartes déjà publiées. Une carte sans fond publié n'est jamais
// devinée — elle garde son polygone brut.
//
// CE QUE LE MASQUE NE PORTE PAS : L'ALTITUDE. Vérifié sur pièces (fond_png.go) — le PNG
// publié est un RGBA d'habillage : au style de production `jeu`, la couleur d'un pixel est
// `TeinteNiveauDeJeu(dz, éclairement)`, deux inconnues mêlées dans trois canaux 8 bits, que
// l'arête (division par 3) et l'eau écrasent en plus. Le sidecar ne publie qu'UNE altitude
// pour toute la carte (`stats.playLevelZ`). Il n'existe donc AUCUN moyen de lire l'altitude
// praticable d'une cellule depuis les artefacts versionnés.
//
// CONSÉQUENCE, et elle est du bon côté. Le test d'étage prévu (ne rogner une zone que si
// l'altitude de la cellule tombe dans [bottom, top] du prisme) n'est pas réalisable : le
// masque dit seulement « cette colonne porte de la matière », sans dire à quelle hauteur.
// Découper là-dessus est donc CONSERVATEUR — on ne retire que les cellules où il n'y a de
// matière à AUCUNE altitude, jamais praticables pour aucun étage. Une zone haute posée
// au-dessus de l'arène garde son emprise (l'arène est de la matière), une zone basse sous
// un toit aussi (le toit est de la matière). Ce que la dégradation coûte, c'est de rogner
// MOINS que le découpage par étage du POC — le coût est mesuré, pas supposé
// (oracle_test.go, IoU contre le découpé POC de Ridgeline).
package mapdecoupe

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"

	"levelup/go-api/internal/analysis/replay"
)

// Masque : la matière praticable d'une carte, cellule par cellule, dans la grille EXACTE du
// fond publié. Le calage voyage avec — c'est lui qui relie la grille au monde.
type Masque struct {
	// Module est la clé du fond (traçabilité des mesures).
	Module string
	// Calage place la grille dans le monde (convention publiée dans le sidecar).
	Calage replay.MapBackgroundCalibration
	// NX, NY sont les dimensions de la grille, égales à celles de l'image.
	NX, NY int
	dur    []bool
}

// ChargeMasque lit le masque praticable d'une carte depuis son PNG et son sidecar.
//
// Une image dont les dimensions contredisent le calage est REFUSÉE : découper sur une
// grille mal calée déplacerait silencieusement toutes les frontières.
func ChargeMasque(pngPath, metaPath string) (*Masque, error) {
	meta, err := replay.LoadMapBackground(metaPath)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(pngPath)
	if err != nil {
		return nil, fmt.Errorf("fond de carte illisible (%s) : %w", pngPath, err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("fond de carte non décodable (%s) : %w", pngPath, err)
	}
	b := img.Bounds()
	if b.Dx() != meta.Calibration.WidthPx || b.Dy() != meta.Calibration.HeightPx {
		return nil, fmt.Errorf("fond de carte %s : image %dx%d, calage %dx%d",
			meta.Module, b.Dx(), b.Dy(), meta.Calibration.WidthPx, meta.Calibration.HeightPx)
	}
	m := &Masque{
		Module: meta.Module,
		Calage: meta.Calibration,
		NX:     b.Dx(),
		NY:     b.Dy(),
		dur:    make([]bool, b.Dx()*b.Dy()),
	}
	remplitDurete(m, img)
	return m, nil
}

// remplitDurete lit le canal alpha pixel à pixel. Le chemin rapide évite 2,6 millions
// d'appels d'interface par carte ; le repli garde le lecteur correct pour tout encodage PNG.
func remplitDurete(m *Masque, img image.Image) {
	b := img.Bounds()
	if nrgba, ok := img.(*image.NRGBA); ok {
		for j := 0; j < m.NY; j++ {
			ligne := nrgba.Pix[j*nrgba.Stride:]
			for i := 0; i < m.NX; i++ {
				m.dur[j*m.NX+i] = ligne[i*4+3] > 0
			}
		}
		return
	}
	for j := 0; j < m.NY; j++ {
		for i := 0; i < m.NX; i++ {
			_, _, _, a := img.At(b.Min.X+i, b.Min.Y+j).RGBA()
			m.dur[j*m.NX+i] = a > 0
		}
	}
}

// Praticable dit si une position monde tombe sur de la matière.
//
// La conversion monde -> pixel passe par `MondeVersPixel`, seul dépositaire de la formule
// de calage (map_background.go) : la ré-écrire ici serait exactement la faute que le
// sidecar existe pour empêcher.
func (m *Masque) Praticable(x, y float64) bool {
	px, py, ok := m.Calage.MondeVersPixel(x, y)
	if !ok {
		return false
	}
	return m.dur[py*m.NX+px]
}

// CelluleM2 est l'aire au sol d'une cellule du masque, en m².
func (m *Masque) CelluleM2() float64 {
	return m.Calage.MetersPerPixel * m.Calage.MetersPerPixel
}

// Comble bouche les TROUS du masque plus étroits que 2 x rayon : fermeture morphologique
// (dilatation puis érosion du même rayon).
//
// POURQUOI IL EN FAUT UNE. Le fond publié est une reconstruction : une passerelle ajourée,
// une grille, un joint entre deux instances laissent des cellules vides AU MILIEU d'un sol
// où l'on court réellement. Rogner dessus retirerait de l'emprise jouée — le défaut que
// l'oracle des positions est là pour interdire. La fermeture ne fait qu'AJOUTER de la
// matière (A est inclus dans sa fermeture, propriété de l'opérateur) et ne touche pas la
// frontière extérieure : le grand vide autour de la carte reste coupé, lui.
//
// Un rayon nul rend le masque tel quel — c'est la mesure de référence.
func (m *Masque) Comble(rayonM float64) *Masque {
	if rayonM <= 0 {
		return m
	}
	r := rayonM / m.Calage.MetersPerPixel
	out := *m
	out.dur = erode(dilate(m.dur, m.NX, m.NY, r), m.NX, m.NY, r)
	return &out
}

// PartDure est la part de cellules portant de la matière — de quoi CONSTATER l'effet d'une
// fermeture sans ouvrir l'image.
func (m *Masque) PartDure() float64 {
	if len(m.dur) == 0 {
		return 0
	}
	n := 0
	for _, d := range m.dur {
		if d {
			n++
		}
	}
	return float64(n) / float64(len(m.dur))
}

// dilate rend les cellules à distance <= r (en cellules) de la matière.
func dilate(bin []bool, nx, ny int, r float64) []bool {
	d := distanceCarre(bin, nx, ny)
	r2 := r * r
	out := make([]bool, len(bin))
	for k := range out {
		out[k] = d[k] <= r2
	}
	return out
}

// erode ne garde que les cellules à distance > r de tout VIDE.
//
// Le dehors du cadre compte comme PLEIN — une cellule au bord de l'image n'a pas de vide
// derrière elle. C'est ce qui garantit qu'une fermeture n'ENLÈVE jamais de matière (sans
// quoi une carte dont le décor touche le cadre y perdrait une bande de r cellules), et donc
// qu'elle ne peut pas coûter de positions jouées.
func erode(bin []bool, nx, ny int, r float64) []bool {
	inv := make([]bool, len(bin))
	for k := range inv {
		inv[k] = !bin[k]
	}
	d := distanceCarre(inv, nx, ny)
	r2 := r * r
	out := make([]bool, len(bin))
	for k := range out {
		out[k] = d[k] > r2
	}
	return out
}

// infDistance : « infini » fini du transformée de distance. Fini pour que les
// soustractions de l'algorithme restent définies ; assez grand pour dominer toute distance
// atteignable sur une grille de carte (côté < 10^4 cellules, donc carré < 10^8).
const infDistance = 1e18

// distanceCarre rend, pour chaque cellule, le CARRÉ de la distance euclidienne à la
// cellule `true` la plus proche (Felzenszwalb & Huttenlocher, deux passes séparables).
//
// Exact, et linéaire en nombre de cellules : un masque de carte fait 2,6 millions de
// cellules et la fermeture en demande deux par rayon.
func distanceCarre(bin []bool, nx, ny int) []float64 {
	f := make([]float64, nx*ny)
	for k, b := range bin {
		if !b {
			f[k] = infDistance
		}
	}
	ligne := make([]float64, nx)
	for j := 0; j < ny; j++ {
		copy(ligne, f[j*nx:(j+1)*nx])
		copy(f[j*nx:(j+1)*nx], dt1d(ligne))
	}
	col := make([]float64, ny)
	for i := 0; i < nx; i++ {
		for j := 0; j < ny; j++ {
			col[j] = f[j*nx+i]
		}
		d := dt1d(col)
		for j := 0; j < ny; j++ {
			f[j*nx+i] = d[j]
		}
	}
	return f
}

// dt1d : transformée de distance 1D d'un échantillonnage de paraboles (enveloppe
// inférieure). `f` est consommé, le résultat est un tableau neuf.
func dt1d(f []float64) []float64 {
	n := len(f)
	d := make([]float64, n)
	v := make([]int, n)
	z := make([]float64, n+1)
	k := 0
	z[0], z[1] = math.Inf(-1), math.Inf(1)
	for q := 1; q < n; q++ {
		s := croisement(f, q, v[k])
		for s <= z[k] {
			k--
			s = croisement(f, q, v[k])
		}
		k++
		v[k], z[k], z[k+1] = q, s, math.Inf(1)
	}
	k = 0
	for q := 0; q < n; q++ {
		for z[k+1] < float64(q) {
			k++
		}
		e := float64(q - v[k])
		d[q] = e*e + f[v[k]]
	}
	return d
}

// croisement rend l'abscisse où la parabole de sommet q passe sous celle de sommet p.
func croisement(f []float64, q, p int) float64 {
	return ((f[q] + float64(q*q)) - (f[p] + float64(p*p))) / float64(2*q-2*p)
}
