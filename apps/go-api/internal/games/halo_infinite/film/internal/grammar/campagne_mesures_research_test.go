//go:build research

package grammar

// campagne_mesures_research_test.go — MESURES CIBLEES DE LA CAMPAGNE DE GRAMMAIRE (etape 4) :
// L ENTREE. Un film a la fois, sous la sentinelle `filmproc` (4 Gio) ; quatre marches par film :
//
//	reference         la marche de la carte v2 (memes comptes) + toutes les sondes ;
//	oracle            reference + les naissances retrouvees par M1, liees apres leur paquet ;
//	tete-bloc         reference avec les candidats de tete filtres par bande OU bloc de type 1 ;
//	oracle+tete-bloc  les deux.
//
// Sorties (TSV, une ligne par film et par cle) dans CAMPAGNE_SORTIE, qui doit etre HORS de data/.

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// cmLigne : une ligne de sortie.
type cmLigne struct{ film, build, cle string }

// TestCampagneMesuresCiblees joue les mesures ciblees sur les films de CAMPAGNE_FILMS.
func TestCampagneMesuresCiblees(t *testing.T) {
	racine, sortie := os.Getenv("CAMPAGNE_RACINE"), os.Getenv("CAMPAGNE_SORTIE")
	var films []string
	for _, x := range strings.Split(os.Getenv("CAMPAGNE_FILMS"), ",") {
		if x = strings.TrimSpace(x); x != "" {
			films = append(films, x)
		}
	}
	if racine == "" || sortie == "" || len(films) == 0 {
		t.Skip("CAMPAGNE_RACINE, CAMPAGNE_FILMS et CAMPAGNE_SORTIE requis")
	}
	abs, _ := filepath.Abs(sortie)
	for _, seg := range strings.Split(filepath.ToSlash(abs), "/") {
		if strings.EqualFold(seg, "data") {
			t.Fatalf("sortie sous data/ : %s", abs)
		}
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		t.Fatal(err)
	}
	utiles := cmUtiles(t)
	tables := map[string][]string{}
	for _, id := range films {
		cmUnFilm(t, racine, id, utiles, tables)
	}
	for nom, lignes := range tables {
		sort.Strings(lignes)
		tete := "film\tbuild\tcle\tn\tpaquets\thors_cadre\tfermes\ten_jeu\n"
		if nom == "variantes" {
			tete = "film\tbuild\tvariante\tpaquets\tfermes\tutiles_fermes\tutiles_lus\thors_cadre\t" +
				"listes_non_localisees\trejets_hors_datum\tanticipations\tliaisons_oracle\toracle_sur_slot_lie\tpaquets_gagnes\tpaquets_perdus\n"
		}
		if nom == "registres" {
			tete = "film\tbuild\tmesure\tvaleur\n"
		}
		brut := tete + strings.Join(lignes, "\n") + "\n"
		if err := os.WriteFile(filepath.Join(abs, "mc_"+nom+".tsv"), []byte(brut), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// cmUnFilm mesure un film.
func cmUnFilm(t *testing.T, racine, id string, utiles UsagesProduit, tables map[string][]string) {
	garde := filmproc.Arm("campagne/mesures", 4, func(pic uint64) {
		fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
		os.Exit(3)
	})
	defer garde.Disarm()
	debut := time.Now()
	f, ok := cmOuvrir(t, racine, id, utiles)
	if !ok {
		return
	}
	b := cmLireBlocs(f)
	wr := profile.QuantRangeCEBiped
	cre, _, _ := ScanVehicleCreations(f.fc, &wr)
	col := nouveauCollecteur(f, b, cre)
	rep, obs, _ := cmMarcher(f, cmVariante{}, col)
	compter := func(m map[[2]int][]cmLiaison) int {
		n := 0
		for _, l := range m {
			n += len(l)
		}
		return n
	}
	liaisons, liaisonsBloc := compter(col.oracle), compter(col.oracleBloc)
	variante := func(nom string, r FrameClosureReport, o *Observation, occ, nl int, cmp *cmComparateur) {
		anti := 0
		for _, k := range o.LiaisonsParRepliDAnticipation {
			anti += k
		}
		tables["variantes"] = append(tables["variantes"], fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d",
			id, f.build, nom, r.Paquets, r.PaquetsFermes, r.Utiles.RecordsFermes, r.Utiles.Records,
			r.Bloquants[CauseHorsCadre].Paquets, r.ListesNonLocalisees, o.RejetsHorsDatum, anti, nl, occ,
			cmp.gagnes, cmp.perdus))
	}
	variante("reference", rep, obs, 0, 0, &cmComparateur{})
	cmp1 := &cmComparateur{ref: col.statut}
	r1, o1, occ1 := cmMarcher(f, cmVariante{oracle: col.oracle}, cmp1)
	variante("oracle-NEW", r1, o1, occ1, liaisons, cmp1)
	cmp4 := &cmComparateur{ref: col.statut}
	r4, o4, occ4 := cmMarcher(f, cmVariante{oracle: col.oracleBloc}, cmp4)
	variante("oracle-bloc", r4, o4, occ4, liaisonsBloc, cmp4)
	tb := cmTeteBloc(b)
	cmp2 := &cmComparateur{ref: col.statut}
	r2, o2, _ := cmMarcher(f, cmVariante{tete: tb}, cmp2)
	variante("tete-bloc", r2, o2, 0, 0, cmp2)
	cmp3 := &cmComparateur{ref: col.statut}
	r3, o3, occ3 := cmMarcher(f, cmVariante{oracle: col.oracleBloc, tete: tb}, cmp3)
	variante("oracle-bloc+tete-bloc", r3, o3, occ3, liaisonsBloc, cmp3)
	for nom, m := range col.t {
		for _, cle := range cmCles(m) {
			x := m[cle]
			tables[nom] = append(tables[nom], fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d",
				id, f.build, cle, x.n, x.paquets, x.horsCadre, x.fermes, x.enJeu))
		}
	}
	tables["registres"] = append(tables["registres"], cmRegistres(f, b, obs)...)
	t.Logf("%s %s : reference %d/%d fermes, oracle-NEW %d/%d (%d), oracle-bloc %d/%d (%d), tete-bloc %d/%d, les deux %d/%d ; pic %d Mio, %s",
		id, f.build, rep.PaquetsFermes, rep.Paquets, r1.PaquetsFermes, r1.Paquets, liaisons, r4.PaquetsFermes, r4.Paquets, liaisonsBloc,
		r2.PaquetsFermes, r2.Paquets, r3.PaquetsFermes, r3.Paquets, garde.Peak()>>20, time.Since(debut).Round(time.Second))
}

// cmTeteBloc : le localisateur de debut de liste dont les candidats NEW sont filtres par la bande
// de production OU par l allocation de leur eid au bloc de type 1 (nee dans le chunk, ou vivante a
// son debut).
func cmTeteBloc(b *cmBlocs) func(c int) func([]byte, *World, FrameConfig) (int, bool) {
	return func(c int) func([]byte, *World, FrameConfig) (int, bool) {
		n, aSuivant := b.suivant[c]
		alloue := func(slot, gen uint32) bool {
			g := uint8(gen) //nolint:gosec // deux bits
			cur, _ := b.entree(c, slot)
			if cur.Vivante() && cur.Gen == g {
				return true
			}
			if !aSuivant {
				return false
			}
			nx, ok := b.entree(n, slot)
			return ok && cmAlloueSous(nx, g) && !cmAlloueSous(cur, g)
		}
		return func(pay []byte, w *World, cfg FrameConfig) (int, bool) {
			cand := func(fin int) []int {
				var out []int
				for p := 0; p+woNewHeaderBits <= fin; p++ {
					slot, ti, ok := enteteNeufEn(pay, p)
					if !ok {
						continue
					}
					h := LireHandle(pay, p+woNewTypeBits)
					if w.anticipee.SlotDeLArchetype(slot, ti) || alloue(h.Slot, h.Gen) {
						out = append(out, p)
					}
				}
				return out
			}
			debut := marchLocateStrict(pay, w, cfg)
			if debut < 0 {
				return debutParFermeture(pay, cand(len(pay)*8), w, cfg)
			}
			return debutParChaine(pay, debut, cand(debut), w, cfg)
		}
	}
}

// cmRegistres : T2-3, T2-5, T4-C2, T7-3 (registre du film), T4-C3 (index de plage), T1-6
// (conflits et tetes des images-cles).
func cmRegistres(f *cmFilm, b *cmBlocs, obs *Observation) []string {
	var out []string
	add := func(m, v string) { out = append(out, fmt.Sprintf("%s\t%s\t%s\t%s", f.id, f.build, m, v)) }
	add("cadre", fmt.Sprintf("IDLowBits=%d IDBase=%d HasExtraFields=%v Region=%d",
		f.cfg.IDLowBits, f.cfg.IDBase, f.cfg.HasExtraFields, f.cfg.Profil.Mouvement.WorldObject.Region))
	groupes, multi := 0, 0
	for _, a := range f.reg.Archetypes {
		parNom := map[string]map[uint32]bool{}
		compte := map[string]int{}
		for i, nom := range a.Components {
			if parNom[nom] == nil {
				parNom[nom] = map[uint32]bool{}
			}
			parNom[nom][a.Level(i)] = true
			compte[nom]++
			if a.Index == 24 && a.Level(i) != 1 {
				add("T2-3 ti=24 niveau != 1", fmt.Sprintf("i%d %s L%d", i, nom, a.Level(i)))
			}
			if strings.Contains(nom, "waypointstate") || strings.Contains(nom, "flock-destination") {
				add("T4-C2 niveau", fmt.Sprintf("ti=%d i%d %s L%d", a.Index, i, nom, a.Level(i)))
			}
		}
		for nom, n := range compte {
			if n < 2 {
				continue
			}
			groupes++
			if len(parNom[nom]) > 1 {
				multi++
				add("T2-5 homonymes a niveaux multiples", fmt.Sprintf("ti=%d %s", a.Index, nom))
			}
		}
		if a.Index == 0 || a.Index == 2 || a.Index == 43 {
			h := fnv.New64a()
			for i, nom := range a.Components {
				fmt.Fprintf(h, "%d:%s:%d;", i, nom, a.Level(i))
			}
			add(fmt.Sprintf("T7-3 empreinte ti=%d", a.Index), fmt.Sprintf("%d entrees %016x", len(a.Components), h.Sum64()))
		}
	}
	add("T2-5 groupes d homonymes", fmt.Sprintf("%d groupes, %d a niveaux multiples", groupes, multi))
	var idx []string
	for k, v := range obs.IndexAbsolus {
		idx = append(idx, fmt.Sprintf("%d:%d", k, v))
	}
	sort.Strings(idx)
	add("T4-C3 index absolus i0", strings.Join(idx, " "))
	tetes := []string{}
	for k, v := range b.table.Tetes() {
		tetes = append(tetes, fmt.Sprintf("%d:%d", k, v))
	}
	sort.Strings(tetes)
	add("T1-6 images-cles", fmt.Sprintf("cles %d, conflits %d, tetes %s", b.table.Entrees(),
		b.table.Conflits(), strings.Join(tetes, " ")))
	return out
}

// cmComparateur compte, contre la marche de reference, les paquets qu une variante ferme en plus
// et ceux qu elle perd.
type cmComparateur struct {
	ref            map[[2]int]bool
	gagnes, perdus int
}

func (c *cmComparateur) debutDeChunk(int, []byte, []FilmPacket, *World) {}

func (c *cmComparateur) paquet(_ int, p *cmPaquet, _ *World) {
	avant := c.ref[[2]int{p.d.Chunk, p.d.Index}]
	switch {
	case p.d.Fermee && !avant:
		c.gagnes++
	case !p.d.Fermee && avant:
		c.perdus++
	}
}

func (c *cmComparateur) finDeFilm() {}
