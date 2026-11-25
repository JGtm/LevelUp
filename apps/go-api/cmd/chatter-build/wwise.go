package main

// wwise.go — LES FORMATS WWISE QUE LA CHAINE LIT : banque (BKHD + HIRC + DIDX/DATA) et paquet (AKPK).
//
// Aucun octet du jeu n'est lu ici : toutes les fonctions travaillent sur des tranches que l'appelant a
// extraites, ce qui les rend testables sur des octets fabriqués (wwise_test.go).

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
)

// objetHIRC est un objet de la hiérarchie d'une banque : son type Wwise, son identifiant, ses données.
type objetHIRC struct {
	Type byte
	ID   uint32
	Data []byte
}

// Les types d'objets HIRC dont la descente des évenements a besoin.
const (
	hircSon         = 2 // source : l'identifiant du média à +5
	hircAction      = 3 // action : l'objet visé à +2
	hircEvenement   = 4 // évènement : une liste d'actions
	hircTailleMini  = 4
	hircEnTeteObjet = 9
)

// chunksWwise découpe une banque en ses sections (`BKHD`, `HIRC`, `DIDX`, `DATA`...) : magique de quatre
// caractères, taille sur quatre octets, contenu. Un découpage qui déborde s'arrête là.
func chunksWwise(b []byte) map[string][]byte {
	out := map[string][]byte{}
	for off := 0; off+8 <= len(b); {
		magic := string(b[off : off+4])
		taille := int(binary.LittleEndian.Uint32(b[off+4:]))
		debut := off + 8
		if taille < 0 || debut+taille > len(b) {
			break
		}
		out[magic] = b[debut : debut+taille]
		off = debut + taille
	}
	return out
}

// objetsHIRC lit la liste d'objets d'une section HIRC. Une liste tronquée rend les objets lus jusque-là.
func objetsHIRC(h []byte) ([]objetHIRC, error) {
	if len(h) < 4 {
		return nil, errors.New("HIRC trop court")
	}
	n := int(binary.LittleEndian.Uint32(h))
	out := make([]objetHIRC, 0, n)
	off := 4
	for i := 0; i < n && off+hircEnTeteObjet <= len(h); i++ {
		typ := h[off]
		taille := int(binary.LittleEndian.Uint32(h[off+1:]))
		id := binary.LittleEndian.Uint32(h[off+5:])
		fin := off + 5 + taille
		if taille < hircTailleMini || fin > len(h) {
			break
		}
		out = append(out, objetHIRC{Type: typ, ID: id, Data: h[off+hircEnTeteObjet : fin]})
		off = fin
	}
	return out, nil
}

// banqueLangue est la banque d'UNE langue : une banque du jeu empile une banque par langue, chacune
// ouverte par son `BKHD` dont le champ à +8 est l'empreinte FNV-1 du nom de dossier de la langue.
type banqueLangue struct {
	Langue uint32
	Objets map[uint32]objetHIRC
	Chunks map[string][]byte
}

// separerBanques coupe une banque multilingue à chaque `BKHD`.
func separerBanques(d []byte) []banqueLangue {
	var debuts []int
	for o := 0; ; {
		i := bytes.Index(d[o:], []byte("BKHD"))
		if i < 0 {
			break
		}
		debuts = append(debuts, o+i)
		o += i + 4
	}
	var out []banqueLangue
	for k, s := range debuts {
		e := len(d)
		if k+1 < len(debuts) {
			e = debuts[k+1]
		}
		ch := chunksWwise(d[s:e])
		bk := ch["BKHD"]
		if len(bk) < 12 || ch["HIRC"] == nil {
			continue
		}
		objs, _ := objetsHIRC(ch["HIRC"])
		mp := make(map[uint32]objetHIRC, len(objs))
		for _, o := range objs {
			mp[o.ID] = o
		}
		out = append(out, banqueLangue{Langue: binary.LittleEndian.Uint32(bk[8:]), Objets: mp, Chunks: ch})
	}
	return out
}

// fnv1 est l'empreinte FNV-1 (32 bits, minuscules) que Wwise donne au nom d'une langue.
func fnv1(s string) uint32 {
	h := uint32(2166136261)
	for _, c := range []byte(s) {
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		h *= 16777619
		h ^= uint32(c)
	}
	return h
}

// listeEnfants cherche, dans les données d'un conteneur, la plus longue liste d'identifiants (précédée de
// son compte sur quatre octets) dont tous les éléments existent dans la banque : c'est la liste des enfants,
// dont l'offset varie selon le type de conteneur.
func listeEnfants(d []byte, connu func(uint32) bool) []uint32 {
	const maxEnfants = 512
	var best []uint32
	for off := 0; off+4 <= len(d); off++ {
		n := int(binary.LittleEndian.Uint32(d[off:]))
		if n < 1 || n > maxEnfants || off+4+4*n > len(d) || n <= len(best) {
			continue
		}
		ids := make([]uint32, 0, n)
		ok := true
		for i := 0; i < n; i++ {
			id := binary.LittleEndian.Uint32(d[off+4+4*i:])
			if !connu(id) {
				ok = false
				break
			}
			ids = append(ids, id)
		}
		if ok {
			best = ids
		}
	}
	return best
}

// mediasDeLEvenement descend la hiérarchie d'un évènement (évènement -> actions -> conteneurs -> sons) et
// rend les identifiants de média (`.wem`) qu'il peut jouer, triés.
func mediasDeLEvenement(b map[uint32]objetHIRC, ev uint32) []uint32 {
	connu := func(id uint32) bool { _, ok := b[id]; return ok }
	vus := map[uint32]bool{}
	set := map[uint32]bool{}
	var descendre func(id uint32)
	descendre = func(id uint32) {
		if vus[id] {
			return
		}
		vus[id] = true
		ob := b[id]
		switch ob.Type {
		case hircSon:
			if len(ob.Data) >= 9 {
				set[binary.LittleEndian.Uint32(ob.Data[5:])] = true
			}
		case hircAction:
			if len(ob.Data) >= 6 {
				if cible := binary.LittleEndian.Uint32(ob.Data[2:]); connu(cible) {
					descendre(cible)
				}
			}
		case hircEvenement:
			d := ob.Data
			if len(d) == 0 {
				return
			}
			n := int(d[0])
			for i := 0; i < n && 1+4*i+4 <= len(d); i++ {
				if a := binary.LittleEndian.Uint32(d[1+4*i:]); connu(a) {
					descendre(a)
				}
			}
		default:
			for _, e := range listeEnfants(ob.Data, connu) {
				descendre(e)
			}
		}
	}
	descendre(ev)
	out := make([]uint32, 0, len(set))
	for w := range set {
		out = append(out, w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// mediaEmbarque est un média embarqué dans une banque (sections DIDX et DATA).
type mediaEmbarque struct {
	ID    uint32
	Octet []byte
}

// mediasEmbarques lit les médias embarqués d'une banque d'une langue. Un média hors de DATA est ignoré.
func mediasEmbarques(b banqueLangue) []mediaEmbarque {
	const tailleEntree = 12
	di, da := b.Chunks["DIDX"], b.Chunks["DATA"]
	var out []mediaEmbarque
	for o := 0; o+tailleEntree <= len(di); o += tailleEntree {
		id := binary.LittleEndian.Uint32(di[o:])
		off := int(binary.LittleEndian.Uint32(di[o+4:]))
		sz := int(binary.LittleEndian.Uint32(di[o+8:]))
		if off < 0 || sz < 0 || off+sz > len(da) {
			continue
		}
		out = append(out, mediaEmbarque{ID: id, Octet: da[off : off+sz]})
	}
	return out
}

// entreePck est un fichier rangé dans un paquet AKPK.
type entreePck struct {
	ID     uint32
	Offset int64
	Taille int
}

const (
	magicAKPK         = "AKPK"
	enteteFixePck     = 0x1C
	largeurIDPck      = 4
	largeurIDLargePck = 8
	champsEntreePck   = 16
)

// lirePck lit la table des fichiers d'un paquet AKPK depuis son seul en-tête : le paquet entier n'est jamais
// chargé (certains pèsent plusieurs centaines de Mo).
func lirePck(r io.ReaderAt, taille int64) ([]entreePck, error) {
	fixe := make([]byte, enteteFixePck)
	if _, err := r.ReadAt(fixe, 0); err != nil || string(fixe[:4]) != magicAKPK {
		return nil, fmt.Errorf("pck: magic AKPK absent")
	}
	tLangues := int64(binary.LittleEndian.Uint32(fixe[0x0C:]))
	tBanks := int64(binary.LittleEndian.Uint32(fixe[0x10:]))
	tStreams := int64(binary.LittleEndian.Uint32(fixe[0x14:]))
	tExternes := int64(binary.LittleEndian.Uint32(fixe[0x18:]))
	fin := enteteFixePck + tLangues + tBanks + tStreams + tExternes
	if fin > taille {
		return nil, fmt.Errorf("pck: en-tête (%d octets) plus long que le fichier (%d)", fin, taille)
	}
	b := make([]byte, fin)
	if _, err := r.ReadAt(b, 0); err != nil {
		return nil, fmt.Errorf("pck: lecture de l'en-tête : %w", err)
	}
	debutBanks := enteteFixePck + tLangues
	debutStreams := debutBanks + tBanks
	debutExternes := debutStreams + tStreams
	var out []entreePck
	for _, t := range []struct {
		offset, taille int64
		large          bool
	}{{debutBanks, tBanks, false}, {debutStreams, tStreams, false}, {debutExternes, tExternes, true}} {
		e, err := lireTablePck(b, int(t.offset), int(t.taille), t.large)
		if err != nil {
			return nil, err
		}
		out = append(out, e...)
	}
	return out, nil
}

// lireTablePck décode une table d'entrées (identifiant, taille de bloc, taille de fichier, bloc de départ,
// langue). L'identifiant est sur huit octets pour la table des fichiers externes, quatre ailleurs.
func lireTablePck(b []byte, offset, taille int, large bool) ([]entreePck, error) {
	if taille <= 4 {
		return nil, nil
	}
	if offset+4 > len(b) {
		return nil, fmt.Errorf("pck: table hors borne (offset %d)", offset)
	}
	n := int(binary.LittleEndian.Uint32(b[offset:]))
	largeurID := largeurIDPck
	if large {
		largeurID = largeurIDLargePck
	}
	stride := largeurID + champsEntreePck
	p := offset + 4
	if p+n*stride > len(b) {
		return nil, fmt.Errorf("pck: %d entrées débordent le fichier", n)
	}
	out := make([]entreePck, 0, n)
	for i := 0; i < n; i++ {
		var id uint32
		if large {
			id = uint32(binary.LittleEndian.Uint64(b[p:]))
		} else {
			id = binary.LittleEndian.Uint32(b[p:])
		}
		q := p + largeurID
		bloc := int64(binary.LittleEndian.Uint32(b[q:]))
		if bloc == 0 {
			bloc = 1
		}
		out = append(out, entreePck{
			ID:     id,
			Taille: int(binary.LittleEndian.Uint32(b[q+4:])),
			Offset: int64(binary.LittleEndian.Uint32(b[q+8:])) * bloc,
		})
		p += stride
	}
	return out, nil
}

// lirePckFichier ouvre un paquet sur disque et rend sa table.
func lirePckFichier(chemin string) ([]entreePck, error) {
	f, err := os.Open(chemin)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return lirePck(f, st.Size())
}
