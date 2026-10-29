//go:build research

package grammar

// variante_taille_equipe_research_test.go — INSTRUMENT : la variante de partie de `chunk_00`,
// vidée champ par champ, pour y chercher la TAILLE D'ÉQUIPE.
//
// LA QUESTION (contrôle n° 8 des contrôles de la v8, 2026-10-21). Le rejeu estime le nombre de
// places d'une équipe ; l'utilisateur veut la taille d'équipe LUE DANS LE FILM si le film la porte.
// Le corps de `chunk_00` porte les OPTIONS DE PARTIE et, en Bond CompactBinary v2, la variante
// (`film_variante_de_partie.go`). Ce banc la vide entière (chemin d'ids de champs -> valeur
// scalaire) pour des films de tailles d'équipe différentes (2v2, 4v4, 8v8, 12v12), afin de
// chercher un champ dont la valeur suit la taille d'équipe.
//
// USAGE (lecture seule, aucun film = saut propre) :
//
//	VARTEAM_FILMS=<racine data/cache> VARTEAM_IDS='bcb6d393=4,4f77afc1=12' \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar -run TestVarianteTailleEquipe -v

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/filmcache"
)

// Les types Bond (CompactBinary v2).
const (
	vtBool = 2 + iota
	vtU8
	vtU16
	vtU32
	vtU64
	vtFloat
	vtDouble
	vtString
	vtStruct
	vtList
	vtSet
	vtMap
	vtI8
	vtI16
	vtI32
	vtI64
	vtWString
)

// vtLecteur vide un message Bond champ par champ.
type vtLecteur struct {
	l      *lecteurDuCorps
	champs map[string]string
}

func (v *vtLecteur) varint() uint64 {
	var x uint64
	for i := range uint(10) {
		o := v.l.lire(8)
		x |= (o & 0x7F) << (7 * i)
		if o&0x80 == 0 {
			return x
		}
	}
	v.l.ok = false
	return 0
}

func vtZigzag(u uint64) int64 { return int64(u>>1) ^ -int64(u&1) }

// valeur lit une valeur de type `ty` au chemin `chemin`.
func (v *vtLecteur) valeur(chemin string, ty int) {
	if !v.l.ok {
		return
	}
	switch ty {
	case vtBool, vtU8, vtI8:
		v.champs[chemin] = strconv.FormatUint(v.l.lire(8), 10)
	case vtU16, vtU32, vtU64:
		v.champs[chemin] = strconv.FormatUint(v.varint(), 10)
	case vtI16, vtI32, vtI64:
		v.champs[chemin] = strconv.FormatInt(vtZigzag(v.varint()), 10)
	case vtFloat:
		v.champs[chemin] = fmt.Sprintf("f%08x", v.l.lire(32))
	case vtDouble:
		v.champs[chemin] = fmt.Sprintf("d%016x", v.l.lire(64))
	case vtString, vtWString:
		n := int(v.varint())
		if ty == vtWString {
			n *= 2
		}
		var b strings.Builder
		for range n {
			o := byte(v.l.lire(8))
			if o >= 32 && o < 127 {
				b.WriteByte(o)
			}
		}
		v.champs[chemin] = strconv.Quote(b.String())
	case vtStruct:
		v.structure(chemin)
	case vtList, vtSet:
		o := v.l.lire(8)
		et, n := int(o&0x1F), int(o>>5)
		if n == 0 {
			n = int(v.varint())
		} else {
			n--
		}
		v.champs[chemin+"#n"] = strconv.Itoa(n)
		for i := range n {
			v.valeur(fmt.Sprintf("%s[%d]", chemin, i), et)
		}
	case vtMap:
		kt, vt := int(v.l.lire(8)), int(v.l.lire(8))
		n := int(v.varint())
		v.champs[chemin+"#n"] = strconv.Itoa(n)
		for i := range n {
			v.valeur(fmt.Sprintf("%s{%d}k", chemin, i), kt)
			v.valeur(fmt.Sprintf("%s{%d}v", chemin, i), vt)
		}
	default:
		v.l.ok = false
	}
}

// structure lit une structure v2 : sa longueur, ses champs, jusqu'à son octet de fin.
func (v *vtLecteur) structure(chemin string) {
	n := int(v.varint())
	fin := v.l.br.BitPos() + 8*n
	base := 0
	for v.l.ok && v.l.br.BitPos() < fin {
		id, ty := v.l.enTete()
		switch ty {
		case bondFin:
			return
		case bondFinDeBase:
			base++
			continue
		}
		v.valeur(fmt.Sprintf("%s.%d:%d", chemin, base, id), ty)
	}
}

// vtVariante rend les champs de la variante de partie du film, ou nil.
func vtVariante(chunk0 []byte) (map[string]string, error) {
	ident, err := ReadFilmIdentity(chunk0)
	if err != nil {
		return nil, err
	}
	l := &lecteurDuCorps{br: LecteurSur(chunk0), ok: true}
	l.br.SetBitPos(ident.BodyBit)
	champsDeTete := map[string]string{}
	for i, n := range [...]uint{3, 3, 2, 7, 64, 32, 3, 32, 32, 32} {
		champsDeTete[fmt.Sprintf("T%d", i)] = strconv.FormatUint(l.lire(n), 10)
	}
	debutOpts := l.br.BitPos()
	optsLu := &vtLecteur{l: l, champs: champsDeTete}
	optsLu.structure("O")
	l.ok = true
	l.br.SetBitPos(debutOpts)
	l.structure(0, func(int, int) bool { return false })
	l.chaine(largeurChaineDuCorps)
	champsDeTete["U0"] = strconv.FormatUint(l.lire(32), 10)
	champsDeTete["U1"] = strconv.FormatUint(l.lire(32), 10)
	if l.lire(1) == 1 {
		l.br.Skip(largeurBlocOptionnelCorps)
	}
	for i, n := range [...]uint{1, 1, 2, 1, 32, 32, 1} {
		champsDeTete[fmt.Sprintf("W%d", i)] = strconv.FormatUint(l.lire(n), 10)
	}
	if l.lire(1) != 1 {
		return nil, fmt.Errorf("variante absente")
	}
	v := &vtLecteur{l: l, champs: champsDeTete}
	v.structure("V")
	if !l.ok {
		return v.champs, fmt.Errorf("lecture arretee")
	}
	return v.champs, nil
}

func TestVarianteTailleEquipe(t *testing.T) {
	racine, liste := os.Getenv("VARTEAM_FILMS"), os.Getenv("VARTEAM_IDS")
	if racine == "" || liste == "" {
		t.Skip("VARTEAM_FILMS ou VARTEAM_IDS absent : instrument saute")
	}
	type film struct {
		id     string
		taille string
		champs map[string]string
	}
	var films []film
	for _, couple := range strings.Split(liste, ",") {
		id, taille, _ := strings.Cut(couple, "=")
		f, ok, err := filmcache.LoadFilm(racine, id)
		if err != nil || !ok {
			t.Logf("%s : film absent (%v)", id, err)
			continue
		}
		chunk0, ok := FilmRegistryChunk(f)
		if !ok {
			t.Logf("%s : pas de chunk_00", id)
			continue
		}
		champs, err := vtVariante(chunk0)
		t.Logf("%s (taille %s) : %d champs, err=%v", id, taille, len(champs), err)
		films = append(films, film{id: id, taille: taille, champs: champs})
	}
	if os.Getenv("VARTEAM_DUMP") != "" && len(films) > 0 {
		cles := make([]string, 0, len(films[0].champs))
		for k := range films[0].champs {
			cles = append(cles, k)
		}
		sort.Strings(cles)
		for _, k := range cles {
			t.Logf("  %s = %s", k, films[0].champs[k])
		}
	}
	if len(films) == 2 && os.Getenv("VARTEAM_DIFF") != "" {
		a, b := films[0].champs, films[1].champs
		var diff []string
		for k, va := range a {
			if vb, ok := b[k]; !ok || vb != va {
				diff = append(diff, k+"="+va+"|"+b[k])
			}
		}
		for k, vb := range b {
			if _, ok := a[k]; !ok {
				diff = append(diff, k+"=absent|"+vb)
			}
		}
		sort.Strings(diff)
		t.Logf("%d champs differents entre %s et %s", len(diff), films[0].id, films[1].id)
		for i, d := range diff {
			if i < 60 {
				t.Logf("  %s", d)
			}
		}
		return
	}
	if prefixe := os.Getenv("VARTEAM_PREFIXE"); prefixe != "" {
		for _, f := range films {
			var lignes []string
			for k, val := range f.champs {
				if strings.HasPrefix(k, prefixe) && !strings.Contains(k[len(prefixe):], "[") {
					lignes = append(lignes, k[len(prefixe):]+"="+val)
				}
			}
			sort.Strings(lignes)
			t.Logf("%s (taille %s) : %s", f.id, f.taille, strings.Join(lignes, " "))
		}
		return
	}
	// Les champs dont la valeur est, sur CHAQUE film, la taille d'équipe déclarée.
	tous := map[string]bool{}
	for _, f := range films {
		for k := range f.champs {
			tous[k] = true
		}
	}
	cles := make([]string, 0, len(tous))
	for k := range tous {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	// LES CHAMPS QUI SEPARENT LES CLASSES : une seule valeur (absence comprise) par taille, et des
	// valeurs distinctes d'une taille a l'autre.
	for _, k := range cles {
		parTaille := map[string]map[string]bool{}
		for _, f := range films {
			val, ok := f.champs[k]
			if !ok {
				val = "absent"
			}
			if parTaille[f.taille] == nil {
				parTaille[f.taille] = map[string]bool{}
			}
			parTaille[f.taille][val] = true
		}
		valeurs, separe := map[string]string{}, true
		for taille, vals := range parTaille {
			if len(vals) != 1 {
				separe = false
				break
			}
			for val := range vals {
				if autre, deja := valeurs[val]; deja && autre != taille {
					separe = false
				}
				valeurs[val] = taille
			}
		}
		if separe && len(parTaille) >= 3 {
			t.Logf("SEPARATEUR %s : %v", k, valeurs)
		}
	}
	if os.Getenv("VARTEAM_EGAUX") == "" {
		return
	}
	for _, k := range cles {
		egaux, vus := 0, 0
		var vals []string
		for _, f := range films {
			if val, ok := f.champs[k]; ok {
				vus++
				vals = append(vals, f.taille+":"+val)
				if val == f.taille {
					egaux++
				}
			}
		}
		if egaux >= 2 {
			t.Logf("CANDIDAT %s : %d/%d egaux a la taille (%s)", k, egaux, len(films), strings.Join(vals, " "))
		}
	}
}
