//go:build research && campagne_overlay

package grammar

// campagne_bis3_ti3_research_test.go — MESURES BIS 3 DE LA CAMPAGNE DE GRAMMAIRE (2026-10-01) :
// LA REGION (iii), point 28 de la critique — les NEW `ti=3` desynchronises sur `i0 low-frequency`.
//
// CE FICHIER EXIGE LA SURCOUCHE DE RECHERCHE DES MESURES BIS 2 (tag `campagne_overlay`,
// `mesures_bis2_overlay/overlay.json`) : le composant non porte est lu par le crochet
// d interception de la copie de recherche de `capture.go` ([bis2Intercepteur]), jamais par un
// fichier de production.
//
// GRAMMAIRE, relue dans Ghidra le 2026-10-01 (lecture seule) :
//
//	nom `low-frequency` : chaine 0x143c957c8, accesseur 0x141177bd0, UNE table de composant
//	0x143d07b40 = [accesseur, 1404ab600, ecrivain 142eda938, 1411c8f80, 14076ce9c, lecteur
//	142ed4aec]. FUN_142ed4aec :
//	    FUN_1424e0e38(br, dst+0x508, 0x10)        = FUN_14076e494(.., 0x10, 0, 0, 0) [lireE494]
//	    FUN_140c5f938(br, dst+0x514, dst+0x520, 0) [consumeObjectForwardAndUp]
//	    R(16) -> +0x52c ; R(8) -> +0x52e ; R(2) -> +0x52f ; R(6) = n -> +0
//	    n x { FUN_1424d9a30 = R(3) f -> e+0x27 ;
//	          si f&1 : FUN_1424e0e38(br, e, 0x10) ; si f&2 : FUN_140c5f938(br, e+0xc, e+0x18, 0) ;
//	          R(16) -> e+0x24 ; FUN_1424ccc74 = R(5) -> e+0x26 }
//	nom `high-frequency` : chaine 0x143c95710, accesseur 0x14119d7f0, DEUX tables :
//	    0x143d06a60 -> lecteur FUN_14076d034 = R(8) (le portage de `ti=4 i0`, dispatch_item.go) ;
//	    0x143d07af0 -> lecteur FUN_142ed4880 = R(16) + R(8) + R(2), voisine IMMEDIATE de la table
//	    de `low-frequency` (0x50 octets plus bas) et memes champs (+0x52c, +0x52e, +0x52f).
//	Le Go lit `high-frequency` par son NOM : `ti=3 i1` y prend R(8). Laquelle des deux tables
//	appartient a `ti=3` n est pas lu statiquement : les deux formes sont mesurees.
//
//	go test -tags=research,campagne_overlay -overlay=<json> -count=1 -timeout 120m \
//	  -run '^TestCampagneBis3Ti3$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// b3LireBasseFrequence porte FUN_142ed4aec.
func b3LireBasseFrequence(br *Lecteur) bool {
	lireE494(br, niveauPosition)
	consumeObjectForwardAndUp(br)
	br.ReadBits(16)
	br.ReadBits(8)
	br.ReadBits(2)
	n := br.ReadBits(6)
	for k := uint64(0); k < n; k++ {
		f := br.ReadBits(3)
		if f&1 != 0 {
			lireE494(br, niveauPosition)
		}
		if f&2 != 0 {
			consumeObjectForwardAndUp(br)
		}
		br.ReadBits(16)
		br.ReadBits(5)
	}
	return true
}

// b3Ti4Temoin : le TEMOIN — `ti=4 i0 high-frequency` lu lui aussi sur 26 bits (FUN_142ed4880) ; si les
// deux tables sont deux composants distincts, ce temoin doit faire PERDRE des paquets.
var b3Ti4Temoin bool

// b3CrochetTi3 : le crochet `ti=3` ; `hf26` lit `high-frequency` par FUN_142ed4880.
func b3CrochetTi3(lf, hf26 bool) func(br *Lecteur, name string, typeIndex, level uint32) (bool, bool) {
	return func(br *Lecteur, name string, typeIndex, _ uint32) (bool, bool) {
		if b3Ti4Temoin && typeIndex == 4 && name == compHighFrequency {
			br.ReadBits(16)
			br.ReadBits(8)
			br.ReadBits(2)
			return true, true
		}
		if typeIndex != 3 {
			return false, false
		}
		switch {
		case lf && name == "low-frequency":
			return true, b3LireBasseFrequence(br)
		case hf26 && name == compHighFrequency:
			br.ReadBits(16)
			br.ReadBits(8)
			br.ReadBits(2)
			return true, true
		}
		return false, false
	}
}

// b3Ti3 compte les records `ti=3` d une marche.
type b3Ti3 struct {
	t cmTables
}

func (e *b3Ti3) debutDeChunk(int, []byte, []FilmPacket, *World) {}
func (e *b3Ti3) finDeFilm()                                     {}

func (e *b3Ti3) paquet(_ int, p *cmPaquet, _ *World) {
	for _, r := range p.recs {
		if r.TypeIndex != 3 || (r.Type != recNew && r.Type != recDelta) {
			continue
		}
		etat := "traverse"
		if r.DesyncAt >= 0 {
			etat = "desynchronise"
			if n := len(r.Trace.Comps); n > 0 {
				etat += " sur " + r.Trace.Comps[n-1].Name
			}
		}
		masque := fmt.Sprintf("masque %#x", r.Trace.Mask)
		k := cmCompte{n: 1, paquets: 1}
		if p.d.Fermee {
			k.fermes = 1
		} else if p.d.Cause == CauseHorsCadre {
			k.horsCadre = 1
		}
		e.t.add("records_ti3", cmJoindre(b3Genre(r.Type), etat), k)
		e.t.add("records_ti3_masque", cmJoindre(b3Genre(r.Type), etat, masque), k)
	}
}

// TestCampagneBis3Ti3 marche chaque film sans, puis avec la grammaire `ti=3` lue chez le jeu.
func TestCampagneBis3Ti3(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	lignes := []string{"film\tbuild\tvariante\tpaquets\tfermes\tutiles_fermes\tutiles_lus\thors_cadre\t" +
		"rejets_hors_datum\tgagnes\tperdus\tgagnes_contredits\tfermes_contredits"}
	tabs := []string{"film\tbuild\tvariante\ttable\tcle\tn\tpaquets\thors_cadre\tfermes"}
	defer func() { bis2Intercepteur = nil }()
	vs := []struct {
		nom      string
		lf, hf26 bool
	}{{"reference", false, false}, {"ti3-i0", true, false}, {"ti3-i0+i1-26", true, true}, {"ti3-i1-26", false, true}, {"ti4-i0-26 (temoin)", false, false}}
	for _, id := range films {
		garde := filmproc.Arm("campagne/bis3-ti3", 4, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
			os.Exit(3)
		})
		debut := time.Now()
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			garde.Disarm()
			continue
		}
		b := cmLireBlocs(f)
		var jref *cmJuge
		var ref map[[2]int]bool
		for _, v := range vs {
			bis2Intercepteur = nil
			b3Ti4Temoin = strings.HasPrefix(v.nom, "ti4")
			if v.lf || v.hf26 || b3Ti4Temoin {
				bis2Intercepteur = b3CrochetTi3(v.lf, v.hf26)
			}
			var j *cmJuge
			if jref == nil {
				j = cmNouveauJuge(f, b, nil)
			} else {
				j = cmNouveauJuge(f, b, ref)
				j.contreRef, j.utilesRef = jref.contreRef, jref.utilesRef
			}
			e := &b3Ti3{t: cmTables{}}
			st := &cmComparateur{ref: map[[2]int]bool{}}
			r, o, _ := cmMarcher(f, cmVariante{}, cmMux{e, j, b3Statut{st}})
			bis2Intercepteur = nil
			if jref == nil {
				jref, ref = j, st.ref
			}
			lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d", id, f.build, v.nom,
				r.Paquets, r.PaquetsFermes, r.Utiles.RecordsFermes, r.Utiles.Records, r.Bloquants[CauseHorsCadre].Paquets,
				o.RejetsHorsDatum, j.gagnes, j.perdus, j.gagnesContredits, j.fermesContre))
			for nom, m := range e.t {
				for _, cle := range cmCles(m) {
					x := m[cle]
					tabs = append(tabs, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d", id, f.build, v.nom, nom, cle,
						x.n, x.paquets, x.horsCadre, x.fermes))
				}
			}
		}
		t.Logf("%s %s : pic %d Mio, %s", id, f.build, garde.Peak()>>20, time.Since(debut).Round(time.Second))
		garde.Disarm()
	}
	b2Ecrire(t, sortie, "mb3_ti3.tsv", lignes)
	b2Ecrire(t, sortie, "mb3_ti3_records.tsv", tabs)
}

// b3Statut releve le statut ferme de chaque paquet (la reference des marches suivantes).
type b3Statut struct{ c *cmComparateur }

func (s b3Statut) debutDeChunk(int, []byte, []FilmPacket, *World) {}
func (s b3Statut) finDeFilm()                                     {}
func (s b3Statut) paquet(_ int, p *cmPaquet, _ *World) {
	s.c.ref[[2]int{p.d.Chunk, p.d.Index}] = p.d.Fermee
}

// TestCampagneBis3Sites : P3 / P4 (eid rejetes qu aucun bloc n alloue) sous la lecture du jeu des
// sites de position de la surcouche (MESURES_BIS_2 §3.1) — si le site est la cause du decalage, la
// population baisse. Sites : CAMPAGNE_SITES (noms de `bis2NomsDeSites`, separes par des virgules),
// plus « tous ».
func TestCampagneBis3Sites(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	var sites []string
	for _, s := range strings.Split(os.Getenv("CAMPAGNE_SITES"), ",") {
		if s = strings.TrimSpace(s); s != "" {
			sites = append(sites, s)
		}
	}
	lignes := []string{"film\tbuild\tvariante\tpaquets\tfermes\thors_cadre\trejets_hors_datum\tgagnes\tperdus\tgagnes_contredits"}
	tabs := []string{"film\tbuild\tvariante\ttable\tcle\tn\tpaquets\thors_cadre\tfermes\ten_jeu"}
	defer b2pVariante{}.poser()
	for _, id := range films {
		garde := filmproc.Arm("campagne/bis3-sites", 4, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
			os.Exit(3)
		})
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			garde.Disarm()
			continue
		}
		b := cmLireBlocs(f)
		vs := []b2pVariante{{nom: "reference"}}
		tous := b2pVariante{nom: "jeu:tous", waypoint: true}
		for s := range tous.sites {
			tous.sites[s] = true
		}
		for _, nom := range sites {
			for s, n := range bis2NomsDeSites {
				if n == nom {
					v := b2pVariante{nom: "jeu:" + nom}
					v.sites[s] = true
					vs = append(vs, v)
				}
			}
		}
		vs = append(vs, tous)
		var jref *cmJuge
		var ref map[[2]int]bool
		for _, v := range vs {
			v.poser()
			d := b3NouveauDiag(f, b, false)
			var j *cmJuge
			if jref == nil {
				j = cmNouveauJuge(f, b, nil)
			} else {
				j = cmNouveauJuge(f, b, ref)
				j.contreRef, j.utilesRef = jref.contreRef, jref.utilesRef
			}
			st := &cmComparateur{ref: map[[2]int]bool{}}
			r, o, _ := cmMarcher(f, cmVariante{}, cmMux{d, j, b3Statut{st}})
			b2pVariante{}.poser()
			if jref == nil {
				jref, ref = j, st.ref
			}
			lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d", id, f.build, v.nom,
				r.Paquets, r.PaquetsFermes, r.Bloquants[CauseHorsCadre].Paquets, o.RejetsHorsDatum, j.gagnes, j.perdus,
				j.gagnesContredits))
			for _, nom := range []string{"population", "dernier_composant_x_classe"} {
				m := d.t[nom]
				for _, cle := range cmCles(m) {
					x := m[cle]
					tabs = append(tabs, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d", id, f.build, v.nom, nom, cle,
						x.n, x.paquets, x.horsCadre, x.fermes, x.enJeu))
				}
			}
		}
		t.Logf("%s %s : %d marches, pic %d Mio", id, f.build, len(vs), garde.Peak()>>20)
		garde.Disarm()
	}
	b2Ecrire(t, sortie, "mb3_sites.tsv", lignes)
	b2Ecrire(t, sortie, "mb3_sites_tables.tsv", tabs)
}

// TestCampagneBis3LiveFire : P3 (« aucune allocation », D-6) sur Live Fire, contexte de production,
// sans puis avec la lecture par index de plage (MESURES_BIS_2 §3.3, T4-C3). Memes variables que
// `TestCampagneBis2PositionsProduction` (CAMPAGNE_CATALOGUE, CAMPAGNE_CARTES, CAMPAGNE_BORNES_<module>).
func TestCampagneBis3LiveFire(t *testing.T) {
	racine, sortie, films := b2Env(t)
	cat, err := profile.LoadMapQuantCatalog(os.Getenv("CAMPAGNE_CATALOGUE"))
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	cartes := map[string]string{}
	for _, x := range strings.Split(os.Getenv("CAMPAGNE_CARTES"), ";") {
		if id, carte, ok := strings.Cut(x, "="); ok {
			cartes[strings.TrimSpace(id)] = strings.TrimSpace(carte)
		}
	}
	utiles := cmUtiles(t)
	lignes := []string{"film\tbuild\tvariante\tpaquets\tfermes\thors_cadre\trejets_hors_datum\tgagnes\tperdus\tgagnes_contredits"}
	tabs := []string{"film\tbuild\tvariante\ttable\tcle\tn\tpaquets\thors_cadre\tfermes\ten_jeu"}
	defer func() { bis2C3 = nil }()
	for _, id := range films {
		entry, err := cat.Lookup(cartes[id])
		if err != nil {
			t.Logf("%s : carte %q hors catalogue (%v)", id, cartes[id], err)
			continue
		}
		f, ok := b2pOuvrirProduction(t, racine, id, entry, utiles)
		if !ok {
			continue
		}
		b := cmLireBlocs(f)
		bornes := b2pBornes(entry.Module)
		var jref *cmJuge
		var ref map[[2]int]bool
		for _, parIndex := range []bool{false, true} {
			bis2C3 = &bis2EtatC3{parIndex: parIndex, bornes: bornes}
			nom := "production"
			if parIndex {
				nom += "+lecture-par-index"
			}
			d := b3NouveauDiag(f, b, false)
			var j *cmJuge
			if jref == nil {
				j = cmNouveauJuge(f, b, nil)
			} else {
				j = cmNouveauJuge(f, b, ref)
				j.contreRef, j.utilesRef = jref.contreRef, jref.utilesRef
			}
			st := &cmComparateur{ref: map[[2]int]bool{}}
			r, o, _ := cmMarcher(f, cmVariante{}, cmMux{d, j, b3Statut{st}})
			bis2C3 = nil
			if jref == nil {
				jref, ref = j, st.ref
			}
			lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d", id, f.build, nom,
				r.Paquets, r.PaquetsFermes, r.Bloquants[CauseHorsCadre].Paquets, o.RejetsHorsDatum, j.gagnes, j.perdus,
				j.gagnesContredits))
			for _, tn := range []string{"population", "dernier_composant_x_classe"} {
				m := d.t[tn]
				for _, cle := range cmCles(m) {
					x := m[cle]
					tabs = append(tabs, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d", id, f.build, nom, tn, cle,
						x.n, x.paquets, x.horsCadre, x.fermes, x.enJeu))
				}
			}
		}
	}
	b2Ecrire(t, sortie, "mb3_livefire.tsv", lignes)
	b2Ecrire(t, sortie, "mb3_livefire_tables.tsv", tabs)
}

// b3LireTypeDeFiltre porte `spawn-filter-type-component` (ti=20 i0, FUN_142ed708c ->
// FUN_142ecf744, relu dans Ghidra le 2026-10-01) avec la branche de NIVEAU du cas 1 : niveau < 2
// -> FUN_1407f2058 (R(1) ; si 0 : R(5)), sinon FUN_142b67e34 = R(9). Le Go lit toujours la
// premiere. Les cas 0, 2, 3 restent ceux du Go.
func b3LireTypeDeFiltre(br *Lecteur, level uint32) {
	switch br.ReadBits(2) {
	case 0:
	case 1:
		if level < 2 {
			if br.ReadBits(1) == 0 {
				br.ReadBits(5)
			}
		} else {
			br.ReadBits(9)
		}
	case 2:
		br.readQuantStat(1)
		br.ReadBits(6)
	default:
		br.ReadBits(32)
		lireE494(br, niveauPosition)
		br.ReadBits(3)
		n := int(br.ReadBits(4))
		for i := 0; i < n; i++ {
			br.ReadBits(32)
		}
		if br.ReadBits(1) == 0 {
			br.ReadBits(5)
		}
	}
}

// TestCampagneBis3FiltreDeReapparition : P3 sous la branche de niveau de `spawn-filter-type`.
func TestCampagneBis3FiltreDeReapparition(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	lignes := []string{"film\tbuild\tvariante\tpaquets\tfermes\tutiles_fermes\thors_cadre\trejets_hors_datum\tgagnes\tperdus\tgagnes_contredits"}
	tabs := []string{"film\tbuild\tvariante\ttable\tcle\tn\tpaquets\thors_cadre\tfermes\ten_jeu"}
	defer func() { bis2Intercepteur = nil }()
	for _, id := range films {
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			continue
		}
		b := cmLireBlocs(f)
		var jref *cmJuge
		var ref map[[2]int]bool
		for _, v := range []string{"reference", "spawn-filter-type niveau"} {
			bis2Intercepteur = nil
			if v != "reference" {
				bis2Intercepteur = func(br *Lecteur, name string, typeIndex, level uint32) (bool, bool) {
					if name != "spawn-filter-type-component" {
						return false, false
					}
					b3LireTypeDeFiltre(br, level)
					return true, true
				}
			}
			d := b3NouveauDiag(f, b, false)
			var j *cmJuge
			if jref == nil {
				j = cmNouveauJuge(f, b, nil)
			} else {
				j = cmNouveauJuge(f, b, ref)
				j.contreRef, j.utilesRef = jref.contreRef, jref.utilesRef
			}
			st := &cmComparateur{ref: map[[2]int]bool{}}
			r, o, _ := cmMarcher(f, cmVariante{}, cmMux{d, j, b3Statut{st}})
			bis2Intercepteur = nil
			if jref == nil {
				jref, ref = j, st.ref
			}
			lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d", id, f.build, v,
				r.Paquets, r.PaquetsFermes, r.Utiles.RecordsFermes, r.Bloquants[CauseHorsCadre].Paquets, o.RejetsHorsDatum,
				j.gagnes, j.perdus, j.gagnesContredits))
			for _, tn := range []string{"population", "dernier_composant_x_classe"} {
				m := d.t[tn]
				for _, cle := range cmCles(m) {
					x := m[cle]
					tabs = append(tabs, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d", id, f.build, v, tn, cle,
						x.n, x.paquets, x.horsCadre, x.fermes, x.enJeu))
				}
			}
		}
	}
	b2Ecrire(t, sortie, "mb3_filtre.tsv", lignes)
	b2Ecrire(t, sortie, "mb3_filtre_tables.tsv", tabs)
}
