package main

import (
	"fmt"
)

// align.go — APPARIEMENT descripteur d'image <-> ressource de pixels, et cache par tag.
//
// Separe de extract.go (seuil de 500 lignes du depot) : ce fichier ne porte QUE la question
// « quelle ressource contient les pixels de quelle image declaree », qui est celle ou le
// risque d'erreur silencieuse est le plus eleve.

// bcSize : poids d'une image en blocs 4x4 sur 16 octets.
func bcSize(w, h int) int { return ((w + 3) / 4) * ((h + 3) / 4) * 16 }

// expectedSizes : les poids ADMISSIBLES du contenu utile d'une ressource pour une image
// donnée — les sommes cumulées de la chaîne de mips, du seul mip0 jusqu'à la chaîne
// complète. Contrôle exact et non un intervalle : une ressource est acceptée si elle pèse
// EXACTEMENT l'une de ces valeurs.
//
// Le nombre de niveaux réellement stockés n'est pas postulé. Le champ `mips` du descripteur
// le donne pour l'atlas des armes (333x117 mips=2 → 40320+10080+2688 = 53088, à l'octet
// près), mais l'exiger fait rejeter 9 images sur 39 de ce même atlas : d'autres entrées
// stockent une chaîne plus courte. On accepte donc toute troncature de la chaîne, ce qui
// reste une égalité stricte — il ne s'agit pas d'une tolérance.
func expectedSizes(im bitmImg) []int {
	var out []int
	total, w, h := 0, im.W, im.H
	for lvl := 0; lvl < 14; lvl++ {
		total += bcSize(w, h)
		out = append(out, total)
		if w == 1 && h == 1 {
			break
		}
		w, h = max1(w/2), max1(h/2)
	}
	return out
}

func max1(v int) int {
	if v < 1 {
		return 1
	}
	return v
}

// alignImages apparie les descripteurs d'images aux ressources par CONTRÔLE ARITHMÉTIQUE,
// et non par simple rang.
//
// POURQUOI. Le rang seul (ressource[base+i] pour le descripteur i) tient sur l'atlas des
// armes mais DÉRIVE sur l'atlas sandbox après ~41 images : le recensement des descripteurs
// est heuristique (signature « dimensions répétées à +0x14 ») et laisse passer des faux
// positifs, qui décalent tout ce qui suit. Un décalage silencieux servirait l'icône d'une
// autre entrée — exactement ce qu'il ne faut pas.
//
// COMMENT. Le rang reste le point de départ, mais chaque appariement est CONTRÔLÉ : le
// contenu utile de la ressource doit peser exactement une somme cumulée de la chaîne de
// mips. S'il ne colle pas, on sonde vers l'avant dans une fenêtre courte et on se recale
// sur la première ressource qui colle — c'est ce qui répare la dérive.
//
// Si aucune ne colle, l'appariement par rang est CONSERVÉ mais marqué non vérifié
// (`verified=false`) : le rejeter ferait perdre 8 icônes d'armes pourtant correctes à
// l'oeil. Le doute est reporté dans index.json, il n'est pas avalé.
// probeWindow : profondeur de la sonde de recalage. Reglable pour mesurer son effet.
var probeWindow = 6

func alignImages(m *hmod, tab []uint32, base int, imgs []bitmImg) (slots []int, verified []bool) {
	window := probeWindow
	out := make([]int, len(imgs))
	okFlags := make([]bool, len(imgs))
	cur := base
	for i, im := range imgs {
		out[i] = -1
		want := expectedSizes(im)
		for probe := cur; probe < cur+window && probe < len(tab); probe++ {
			fi := int(tab[probe])
			if fi < 0 || fi >= m.hdr.FileCount {
				continue
			}
			blob, err := m.extract(m.file(fi))
			if err != nil {
				continue
			}
			rh, ok := parseTagHeader(blob)
			if !ok {
				continue
			}
			px := int(rh.HeaderSize) + int(rh.DataSize)
			if px < 0 || px > len(blob) {
				continue
			}
			payload := len(blob) - px
			match := false
			for _, w := range want {
				if payload == w {
					match = true
					break
				}
			}
			if !match {
				continue
			}
			out[i] = probe
			okFlags[i] = true
			cur = probe + 1
			break
		}
		if out[i] < 0 {
			// Aucune ressource ne colle exactement. On garde le rang et on marque le doute —
			// MAIS seulement si la ressource au rang peut au moins CONTENIR le mip0 : sinon
			// le descripteur est un faux positif du recensement, et le retenir décalerait
			// tout ce qui suit (c'est ce qui faisait dérailler l'atlas sandbox après 41).
			if cur < len(tab) && resourcePayload(m, tab, cur) >= bcSize(im.W, im.H) {
				out[i] = cur
				cur++
			}
		}
	}
	return out, okFlags
}

// atlas : un tag `bitm` d'icônes, ses descripteurs appariés et la table de ressources de son
// archive. L'appariement est coûteux (il décompresse pour vérifier les tailles) : il est fait
// UNE FOIS par tag, puis mis en cache.
type atlas struct {
	mod   *hmod
	tab   []uint32
	imgs  []bitmImg // descripteurs retenus, dans l ordre
	verif []bool    // appariement confirme par le controle arithmetique
	slots []int     // index de ressource de chaque descripteur retenu
}

// atlasOf apparie une fois pour toutes les images d'un tag à leurs ressources.
func atlasOf(ix *tagIndex, id uint32) (*atlas, error) {
	if a, ok := ix.atlases[id]; ok {
		if a == nil {
			return nil, fmt.Errorf("tag %08x non exploitable", id)
		}
		return a, nil
	}
	build := func() (*atlas, error) {
		r, ok := pickRef(ix.byID[id])
		if !ok {
			return nil, fmt.Errorf("tag %08x absent des archives", id)
		}
		m, err := ix.open(r.Module)
		if err != nil {
			return nil, err
		}
		tab, err := resourceTable(m)
		if err != nil {
			return nil, err
		}
		data, err := ix.extract(r)
		if err != nil {
			return nil, err
		}
		h, ok := parseTagHeader(data)
		if !ok {
			return nil, fmt.Errorf("tag %08x sans en-tête ucsh", id)
		}
		all := scanImgs(data[h.HeaderSize:])
		slots, verif := alignImages(m, tab, entryResourceIndex(m, r.Entry), all)
		a := &atlas{mod: m, tab: tab}
		for i, s := range slots {
			if s >= 0 {
				a.imgs = append(a.imgs, all[i])
				a.slots = append(a.slots, s)
				a.verif = append(a.verif, verif[i])
			}
		}
		return a, nil
	}
	a, err := build()
	ix.atlases[id] = a
	return a, err
}

// resBlob rend le contenu de la ressource appariée à l'image idx, et l'offset de ses pixels.
// La ressource est elle-même un blob à en-tête `ucsh` : les pixels commencent après
// headerSize + dataSize (vérifié à l'octet près, cf. l'en-tête du paquet).
func resBlob(ix *tagIndex, id uint32, idx int) (blob []byte, px int, im bitmImg, err error) {
	a, err := atlasOf(ix, id)
	if err != nil {
		return nil, 0, im, err
	}
	if idx < 0 || idx >= len(a.imgs) {
		return nil, 0, im, fmt.Errorf("index %d hors borne (%d images appariées)", idx, len(a.imgs))
	}
	blob, err = a.mod.extract(a.mod.file(int(a.tab[a.slots[idx]])))
	if err != nil {
		return nil, 0, im, err
	}
	rh, ok := parseTagHeader(blob)
	if !ok {
		return nil, 0, im, fmt.Errorf("ressource %d sans en-tête ucsh", a.slots[idx])
	}
	return blob, int(rh.HeaderSize) + int(rh.DataSize), a.imgs[idx], nil
}

// imageVerified dit si l'appariement de l'image idx a été confirmé par le contrôle
// arithmétique (poids exact d'une chaîne de mips) ou seulement déduit du rang.
func imageVerified(ix *tagIndex, id uint32, idx int) bool {
	a, err := atlasOf(ix, id)
	if err != nil || idx < 0 || idx >= len(a.verif) {
		return false
	}
	return a.verif[idx]
}

// resourcePayload rend le poids du contenu utile d'une ressource (tout ce qui suit son
// en-tête `ucsh`), ou -1 si elle n'est pas lisible.
func resourcePayload(m *hmod, tab []uint32, slot int) int {
	if slot < 0 || slot >= len(tab) {
		return -1
	}
	fi := int(tab[slot])
	if fi < 0 || fi >= m.hdr.FileCount {
		return -1
	}
	blob, err := m.extract(m.file(fi))
	if err != nil {
		return -1
	}
	rh, ok := parseTagHeader(blob)
	if !ok {
		return -1
	}
	px := int(rh.HeaderSize) + int(rh.DataSize)
	if px < 0 || px > len(blob) {
		return -1
	}
	return len(blob) - px
}
