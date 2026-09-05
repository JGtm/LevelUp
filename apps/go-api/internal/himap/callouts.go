// Package himap — callouts.go : les ZONES NOMMÉES (« callouts ») d'une carte, lues dans
// le tag de scénario `levl` de son .module.
//
// PORTAGE ÉTABLI, PAS UNE DÉCOUVERTE. La chaîne et les offsets viennent des scripts de
// recherche `callouts.py` / `callouts_all.py` (LevelUp-re/scratchpad_recherche), documentés
// au champ près et mesurés sur les 31 modules installés (2026-08 : 22 cartes portent
// 816 zones, liaison nom<->volume 816/816, les canevas Forge en portent ZÉRO — c'est un
// fait de construction, pas une absence de données) :
//
// LE ZÉRO DES CANEVAS EST VÉRIFIÉ, PAS SUPPOSÉ (sonde_levl_canevas_gamefiles_test.go,
// 2026-08-27) : lire un compte à root+0x91C sur un tag de disposition différente rendrait
// zéro aussi, et ce zéro-là ne dirait rien. Le root block du tag levl mesure 3 184 octets
// sur les 31 modules installés, canevas compris, et les blocs `names`/`volumes` y sont
// présents des deux côtés. Les 8 canevas portent 12 volumes ANONYMES, tous kind=1 (une
// zone nommée est kind=6), aux mêmes string_id d'un canevas à l'autre et aux bornes de la
// boîte ±212,5/250 m : ce sont les barrières du canevas. Les callouts d'une carte Forge
// vivent dans son `.mvar`, pas ici.
//
//	levl -> root block
//	  root+0x91C : TagBlock « named locations » (stride 0x28)
//	               { i16 volume_index, char name[32], u16 pad, u32 name_string_id }
//	  root+0x3BC : TagBlock « volumes »          (stride 0xD0)
//	               { u32 string_id, ..., u8 has_shape @0x14, u8 kind @0x16,
//	                 3f pos @0x44, f32 top @0x64, f32 bottom @0x68,
//	                 u32 n_poly @0x7C, ... }
//	  bloc enfant du volume @local 0x6C : sommets du polygone au sol (3f par sommet)
//
// La navigation de struct-table est celle de sddt.go (meilleurTagInfo, liensBlocs,
// compteChamp) — la lecture d'un tag SANS plugin, déjà promue en production.
//
// DÉPÔT DE JEU : deploy/ds/. C'est la variante sur laquelle la table a été établie et
// mesurée (callouts_all.py) ; le build serveur dédié porte le scénario complet.
package himap

import (
	"fmt"
	"strings"

	"levelup/go-api/internal/himodule"
)

// Offsets et strides du tag levl — mesurés, cf. en-tête.
const (
	levlChampNames   = 0x91C
	levlChampVolumes = 0x3BC
	levlStrideName   = 0x28
	levlStrideVolume = 0xD0

	levlNameOffVolume = 0x00 // i16 volume_index
	levlNameOffText   = 0x02 // char[32]
	levlNameOffSID    = 0x24 // u32 string_id

	levlVolOffSID      = 0x00 // u32 string_id (doit égaler celui du nom : liaison 816/816)
	levlVolOffHasShape = 0x14 // u8
	levlVolOffKind     = 0x16 // u8
	levlVolOffPos      = 0x44 // 3f
	levlVolOffTop      = 0x64 // f32 — extension verticale AU-DESSUS de pos.z
	levlVolOffBottom   = 0x68 // f32 — extension verticale EN DESSOUS de pos.z
	levlVolOffPolygon  = 0x6C // bloc enfant : sommets 3f, RELATIFS à pos (cf. AABB)
	levlVolOffAABB     = 0x94 // 6f [minX maxX minY maxY minZ maxZ], relatifs à pos
)

// levlAABBToleranceM : écart admis entre l'AABB du record et la boîte recalculée sur les
// sommets. Mesuré nul (float32 identiques) sur les 22 cartes ; la marge absorbe la seule
// conversion float32 -> float64.
const levlAABBToleranceM = 0.01

// calloutNamePrefix : préfixe de conception systématique des named locations. Il ne porte
// rien (816/816 le portent) — on publie le nom SANS lui, comme le faisaient les scripts.
const calloutNamePrefix = "named location "

// Callout est une zone nommée : une entrée « named location » et le volume qu'elle désigne.
type Callout struct {
	// VolumeIndex est l'indice du volume dans le bloc volumes — la clé de jointure avec
	// callouts_i18n.csv (une ligne par named location, indexée carte+volumeIndex).
	VolumeIndex int
	// Name est le nom de CONCEPTION (char[32] du tag, préfixe « named location » retiré).
	// Possiblement tronqué à 32 octets par le format — le libellé joueur vient du StringID.
	Name string
	// StringID est la clé du libellé localisé (tag uslg, résolu hors ligne dans
	// callouts_i18n.csv — on ne re-extrait PAS uslg).
	StringID uint32
	// HasShape dit si le volume porte son propre polygone dessiné par le designer.
	HasShape bool
	// Kind est le type du volume (mesuré : 6 sur toutes les zones nommées).
	Kind int
	// Pos est le point de référence du volume, en mètres monde — le même repère que les
	// trajectoires du rejeu.
	Pos [3]float64
	// Top et Bottom sont les extensions verticales du prisme au-dessus et en dessous de
	// Pos[2] : la tranche habitée va de Pos[2]-Bottom à Pos[2]+Top.
	Top, Bottom float64
	// Polygon est le polygone au sol en MÈTRES MONDE (sommets XY ; le Z de conception de
	// chaque sommet est ignoré, la tranche verticale vit dans Top/Bottom). Vide quand le
	// volume n'a pas de forme propre.
	//
	// LE TAG STOCKE LES SOMMETS RELATIFS À POS, ET C'EST MESURÉ DEUX FOIS : (1) pos+rel
	// reproduit EXACTEMENT (écart 0,0000 m) les polygones monde du dump de référence de
	// Ridgeline sur ses 16 zones dessinées ; (2) l'AABB @0x94 du record est la boîte des
	// sommets RELATIFS ([minX maxX minY maxY minZ maxZ]) sur toutes les cartes — donc le
	// repère des sommets est le repère monde translaté, sans rotation supplémentaire
	// (une rotation appliquée après coup rendrait l'AABB fausse pour le jeu lui-même).
	// La lecture vérifie cet invariant à chaque extraction et refuse un record qui ne le
	// porte plus.
	Polygon [][2]float64
}

// ZBottom et ZTop rendent la tranche verticale absolue du volume.
func (c Callout) ZBottom() float64 { return c.Pos[2] - c.Bottom }
func (c Callout) ZTop() float64    { return c.Pos[2] + c.Top }

// ReadModuleCallouts lit les zones nommées du tag levl d'un .module de carte.
//
// ZÉRO ZONE EST UN RÉSULTAT, PAS UNE ERREUR : les canevas Forge portent un tag levl sans
// named locations (mesuré sur les 8 canevas installés). L'appelant distingue donc « carte
// sans callouts » (slice vide, nil error) d'une chaîne cassée (error).
func ReadModuleCallouts(modulePath string) ([]Callout, error) {
	m, err := himodule.Open(modulePath)
	if err != nil {
		return nil, fmt.Errorf("himap: ouvrir %s: %w", modulePath, err)
	}
	// Le module est projete en memoire : tout ce qui est rendu ci-dessous est une COPIE, la
	// projection peut donc etre relachee a la sortie (cf. himodule.Module.Close).
	defer func() { _ = m.Close() }()
	levls := m.Files("levl")
	if len(levls) != 1 {
		return nil, fmt.Errorf("himap: %d tags levl dans %s (attendu 1)", len(levls), modulePath)
	}
	tag, err := m.Extract(levls[0])
	if err != nil {
		return nil, fmt.Errorf("himap: extraire levl de %s: %w", modulePath, err)
	}
	return calloutsFromTag(tag)
}

// calloutsFromTag décode les zones nommées d'un tag levl déjà extrait.
func calloutsFromTag(tag []byte) ([]Callout, error) {
	ti, err := meilleurTagInfo(tag)
	if err != nil {
		return nil, fmt.Errorf("himap: en-tête levl: %w", err)
	}
	root, err := ti.rootBlockIndex()
	if err != nil {
		return nil, fmt.Errorf("himap: levl: %w", err)
	}
	liens := liensBlocs(ti)
	parOwnerOff := map[[2]int]lienBloc{}
	for _, l := range liens {
		parOwnerOff[[2]int{l.owner, l.fieldOff}] = l
	}

	rootAbs, rootSize := ti.blockAbs(root)
	if levlChampNames+0x14 > rootSize {
		return nil, fmt.Errorf("himap: root block levl trop court (%d o)", rootSize)
	}
	nNames := u32(ti.tag, rootAbs+levlChampNames+0x10)
	nVols := u32(ti.tag, rootAbs+levlChampVolumes+0x10)
	if nNames == 0 {
		// Cas NOMINAL des canevas Forge : un scénario sans zones nommées.
		return nil, nil
	}
	ln, okN := parOwnerOff[[2]int{root, levlChampNames}]
	lv, okV := parOwnerOff[[2]int{root, levlChampVolumes}]
	if !okN || !okV {
		return nil, fmt.Errorf("himap: levl: blocs names/volumes introuvables (%d noms, %d volumes déclarés)", nNames, nVols)
	}
	nAbs, _, err := blocVerifie(ti, ln, levlStrideName)
	if err != nil {
		return nil, fmt.Errorf("himap: levl names: %w", err)
	}
	vAbs, _, err := blocVerifie(ti, lv, levlStrideVolume)
	if err != nil {
		return nil, fmt.Errorf("himap: levl volumes: %w", err)
	}

	out := make([]Callout, 0, nNames)
	for i := 0; i < nNames; i++ {
		c, err := litCallout(ti, nAbs+i*levlStrideName, vAbs, nVols, lv, parOwnerOff)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// litCallout décode une named location et le volume qu'elle désigne.
func litCallout(ti tagInfo, o, vAbs, nVols int, lv lienBloc, parOwnerOff map[[2]int]lienBloc) (Callout, error) {
	vi := int(int16(u16(ti.tag, o+levlNameOffVolume)))
	c := Callout{
		VolumeIndex: vi,
		Name:        strings.TrimPrefix(cString(ti.tag[o+levlNameOffText:o+levlNameOffText+32]), calloutNamePrefix),
		StringID:    uint32(u32(ti.tag, o+levlNameOffSID)),
	}
	if vi < 0 || vi >= nVols {
		return Callout{}, fmt.Errorf("himap: levl: named location %q pointe le volume %d hors des %d volumes", c.Name, vi, nVols)
	}
	r := vAbs + vi*levlStrideVolume
	if volSID := uint32(u32(ti.tag, r+levlVolOffSID)); volSID != c.StringID {
		// La liaison nom<->volume est un invariant MESURÉ (816/816) : sa rupture est
		// une structure inconnue, jamais une zone à publier quand même.
		return Callout{}, fmt.Errorf("himap: levl: string_id du volume %d (%08x) != nom %q (%08x)",
			vi, volSID, c.Name, c.StringID)
	}
	c.HasShape = ti.tag[r+levlVolOffHasShape] == 1
	c.Kind = int(ti.tag[r+levlVolOffKind])
	for a := 0; a < 3; a++ {
		c.Pos[a] = f32(ti.tag, r+levlVolOffPos+4*a)
	}
	c.Top = f32(ti.tag, r+levlVolOffTop)
	c.Bottom = f32(ti.tag, r+levlVolOffBottom)
	// Le polygone n'est lu QUE sur les volumes à forme propre : les autres portent
	// une boîte par défaut (sommets ±0,5, AABB nulle — mesuré sur le balayage des
	// 22 cartes), pas une forme dessinée.
	if lp, ok := parOwnerOff[[2]int{lv.target, vi*levlStrideVolume + levlVolOffPolygon}]; ok && c.HasShape {
		pAbs, pSize := ti.blockAbs(lp.target)
		if err := verifieAABBRelative(ti, r, pAbs, pSize); err != nil {
			return Callout{}, fmt.Errorf("himap: levl volume %d (%s): %w", vi, c.Name, err)
		}
		for k := 0; k+12 <= pSize; k += 12 {
			// Sommets relatifs -> monde (invariant vérifié juste au-dessus).
			c.Polygon = append(c.Polygon, [2]float64{
				c.Pos[0] + f32(ti.tag, pAbs+k),
				c.Pos[1] + f32(ti.tag, pAbs+k+4),
			})
		}
	}
	return c, nil
}

// verifieAABBRelative confronte l'AABB @0x94 d'un volume à la boîte recalculée sur ses
// sommets relatifs — LE témoin qui établit (et garde) le repère des sommets. Un record
// qui ne le vérifie plus est une structure inconnue : on refuse, on ne translate pas au
// petit bonheur.
func verifieAABBRelative(ti tagInfo, volAbs, polyAbs, polySize int) error {
	if polySize < 12 {
		return nil
	}
	minX, maxX := f32(ti.tag, polyAbs), f32(ti.tag, polyAbs)
	minY, maxY := f32(ti.tag, polyAbs+4), f32(ti.tag, polyAbs+4)
	for k := 12; k+12 <= polySize; k += 12 {
		x, y := f32(ti.tag, polyAbs+k), f32(ti.tag, polyAbs+k+4)
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}
	for i, attendu := range []float64{minX, maxX, minY, maxY} {
		lu := f32(ti.tag, volAbs+levlVolOffAABB+4*i)
		if d := lu - attendu; d < -levlAABBToleranceM || d > levlAABBToleranceM {
			return fmt.Errorf("AABB @0x94[%d] = %.3f, boîte des sommets = %.3f — repère des sommets inconnu", i, lu, attendu)
		}
	}
	return nil
}

// cString coupe un champ char[N] au premier NUL.
func cString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}
