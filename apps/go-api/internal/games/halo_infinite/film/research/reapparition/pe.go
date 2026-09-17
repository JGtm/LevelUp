//go:build research

package reapparition

// pe.go — L'EXECUTABLE EN LECTURE SEULE : sections, adresses virtuelles, et les bornes exactes
// des fonctions.
//
// TOUT PASSE PAR L'ADRESSE VIRTUELLE (VA). Les notes du chantier ecrivent les adresses en VA
// (`0x143d084d0`, `FUN_1410d9088`) parce que c'est ce que Ghidra affiche avec la base
// `0x140000000`. Cet instrument parle la meme langue : aucune adresse de fichier ne sort d'ici.
//
// LES BORNES DE FONCTION NE SONT PAS DEVINEES. Le PE x64 porte un repertoire d'exceptions
// (`.pdata`) fait d'entrees `RUNTIME_FUNCTION` de 12 octets — `BeginAddress`, `EndAddress`,
// `UnwindInfoAddress`, toutes en RVA. Une recherche binaire y rend la fonction qui contient une
// adresse, avec sa fin. C'est exact et gratuit ; chercher un `RET` a la main ne l'est pas.

import (
	"debug/pe"
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

// Executable est l'image chargee en memoire, lisible par adresse virtuelle.
type Executable struct {
	// Base est l'ImageBase du PE (0x140000000 sur HaloInfinite.exe).
	Base uint64
	// sections porte, pour chaque section a contenu, sa VA de debut et ses octets.
	sections []section
	// funcs porte les entrees du repertoire d'exceptions, triees par debut.
	funcs []FuncRange
}

type section struct {
	nom      string
	va, fin  uint64
	contenu  []byte
	execCode bool
}

// FuncRange est une fonction du repertoire d'exceptions : ses bornes EXACTES, en VA.
type FuncRange struct {
	Debut, Fin uint64
}

// Charger ouvre l'executable en lecture seule et indexe ses sections et ses fonctions.
func Charger(chemin string) (*Executable, error) {
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin fourni par l'operateur, lecture seule
	if err != nil {
		return nil, fmt.Errorf("lecture de %s : %w", chemin, err)
	}
	f, err := pe.NewFile(newLecteurAt(brut))
	if err != nil {
		return nil, fmt.Errorf("format PE de %s : %w", chemin, err)
	}
	defer func() { _ = f.Close() }()

	oh, ok := f.OptionalHeader.(*pe.OptionalHeader64)
	if !ok {
		return nil, fmt.Errorf("%s n'est pas un PE 64 bits", chemin)
	}
	ex := &Executable{Base: oh.ImageBase}
	for _, s := range f.Sections {
		if s.Size == 0 || s.Offset == 0 {
			continue
		}
		fin := uint64(s.Offset) + uint64(s.Size)
		if fin > uint64(len(brut)) {
			fin = uint64(len(brut))
		}
		va := oh.ImageBase + uint64(s.VirtualAddress)
		contenu := brut[s.Offset:fin]
		ex.sections = append(ex.sections, section{
			nom: s.Name, va: va, fin: va + uint64(len(contenu)), contenu: contenu,
			execCode: s.Characteristics&0x20000000 != 0,
		})
	}
	ex.chargerPdata(oh)
	return ex, nil
}

// chargerPdata indexe le repertoire d'exceptions (entree 3 du tableau des repertoires).
func (e *Executable) chargerPdata(oh *pe.OptionalHeader64) {
	const repertoireException = 3
	const tailleEntree = 12
	d := oh.DataDirectory[repertoireException]
	if d.VirtualAddress == 0 || d.Size == 0 {
		return
	}
	pay := e.Lire(e.Base+uint64(d.VirtualAddress), int(d.Size))
	for off := 0; off+tailleEntree <= len(pay); off += tailleEntree {
		deb := binary.LittleEndian.Uint32(pay[off:])
		fin := binary.LittleEndian.Uint32(pay[off+4:])
		if deb == 0 || fin <= deb {
			continue
		}
		e.funcs = append(e.funcs, FuncRange{Debut: e.Base + uint64(deb), Fin: e.Base + uint64(fin)})
	}
	sort.Slice(e.funcs, func(i, j int) bool { return e.funcs[i].Debut < e.funcs[j].Debut })
}

// Lire rend au plus n octets a partir de la VA demandee, ou nil si la VA n'est dans aucune
// section a contenu. La tranche est une VUE sur l'image : ne jamais y ecrire.
func (e *Executable) Lire(va uint64, n int) []byte {
	for i := range e.sections {
		s := &e.sections[i]
		if va < s.va || va >= s.fin {
			continue
		}
		deb := va - s.va
		reste := uint64(len(s.contenu)) - deb
		if uint64(n) > reste {
			n = int(reste)
		}
		return s.contenu[deb : deb+uint64(n)]
	}
	return nil
}

// U64 lit un mot de 64 bits a la VA demandee. `ok` est faux hors section.
func (e *Executable) U64(va uint64) (uint64, bool) {
	b := e.Lire(va, 8)
	if len(b) < 8 {
		return 0, false
	}
	return binary.LittleEndian.Uint64(b), true
}

// Fonction rend les bornes exactes de la fonction qui contient la VA, depuis `.pdata`.
func (e *Executable) Fonction(va uint64) (FuncRange, bool) {
	i := sort.Search(len(e.funcs), func(i int) bool { return e.funcs[i].Debut > va })
	if i == 0 {
		return FuncRange{}, false
	}
	f := e.funcs[i-1]
	if va >= f.Debut && va < f.Fin {
		return f, true
	}
	return FuncRange{}, false
}

// Sections rend le nom et les bornes de chaque section a contenu — publie pour que le rapport
// puisse dire sur QUELLE image il a travaille.
func (e *Executable) Sections() []string {
	out := make([]string, 0, len(e.sections))
	for i := range e.sections {
		s := &e.sections[i]
		out = append(out, fmt.Sprintf("%-8s %#x..%#x (%d o)", s.nom, s.va, s.fin, len(s.contenu)))
	}
	return out
}

// lecteurAt adapte une tranche d'octets a l'interface attendue par `debug/pe`.
type lecteurAt struct{ b []byte }

func newLecteurAt(b []byte) *lecteurAt { return &lecteurAt{b: b} }

func (l *lecteurAt) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off >= int64(len(l.b)) {
		return 0, fmt.Errorf("lecture hors image a %d", off)
	}
	n := copy(p, l.b[off:])
	return n, nil
}
