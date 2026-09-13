package filmdec

// f0_103_verdicts_research_test.go — LOT F.0, QUESTIONS 1 a 3 : les references du type 103
// resolues, et ce qu elles disent du DEPLOIEMENT contre le LACHER A LA MORT.
//
// # QUESTION 1 — le 103 en liste complete, et ses references resolues
//
// Trois references gardees, domaines {0,0,7}, index de 13 bits + generation de 2 bits. La
// resolution testee est un APPARIEMENT EXACT, jamais une fenetre de temps : `(base + index,
// generation)` est-il la cle d une VIE d objet `ti=37` reellement creee dans le film
// (`EquipmentCreation.Slot/Gen`) ? Deux bases sont mesurees cote a cote — 0 et 512 (R1 avait
// etabli la base 512 sur le type 117) — et c est l ecart entre elles, contre le TEMOIN, qui
// tranche.
//
// TEMOIN NEGATIF OBLIGATOIRE : les memes resolutions sur des references TIREES AU HASARD dans
// le domaine de 13 bits, en meme nombre et sous la meme base. Un taux qui ne se detacherait
// pas du temoin ne prouverait rien — le domaine est petit et le film cree beaucoup d objets.
//
// # QUESTIONS 2 et 3 — le 103 tire-t-il a la mort ?
//
// Pour chaque pose PUBLIEE (jointure `f0Joint`), on demande si une reference d un 103 du film
// designe sa vie. La reponse est ventilee par ORIGINE (`deployed` / `dropped` / `unknown`) et
// par GlobalID. Si les `dropped` n en ont jamais et les panneaux de mur toujours, le 103 EST
// le fait « deploye », et la question 3 (les huit cas de D12) se tranche dessus.
//
// LECTURE SEULE. Gardes et commande : `f0_103_contexte_research_test.go`.

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"sort"
	"testing"
)

// f0Resolution compte, pour UNE position de reference et UNE base, ce a quoi elle se resout.
type f0Resolution struct {
	Total    int // occurrences examinees
	Presente int // porte posee
	// Resolue : `(base+index, generation)` est la cle d une creation `ti=37` du film.
	Resolue int
	// ResolueProj : la meme cle est celle d une VIE DE PROJECTILE `ti=41`. C est la seconde
	// nature d objet que le 103 engendre, et sans elle son taux de resolution se lit trop bas.
	ResolueProj int
	// IndexSeul : l index se resout, la generation non.
	IndexSeul int
	// ParTI : archetype que les images-cles voient au slot vise — UNE voix par occurrence.
	ParTI map[int]int
	// HorsCensus : le slot vise n apparait dans aucune image-cle (le cas normal d un objet
	// qui vit moins que l intervalle de 20 s entre deux images-cles).
	HorsCensus int
	// ParID : GlobalID de la creation resolue — la reponse a « QUOI est reference ».
	ParID map[uint32]int
	// Dt : ecart en ms entre l evenement et la creation resolue (evenement - creation).
	Dt []int64
}

func f0NouvelleResolution() *f0Resolution {
	return &f0Resolution{ParTI: map[int]int{}, ParID: map[uint32]int{}}
}

// f0Cible porte ce qu une cle de vie designe : son GlobalID et l instant de sa creation.
type f0Cible struct {
	ID   uint32
	TsUS uint64
}

func (r *f0Resolution) ajoute(ref r7RefVal, base uint64, atUS uint64,
	census map[uint64]map[int]int, vies map[f0CleVie]f0Cible, index map[uint64]bool,
	proj map[f0CleVie]bool) {
	r.Total++
	if !ref.Present {
		return
	}
	r.Presente++
	slot := ref.Index + base
	if tis, vu := census[slot]; vu {
		ti, best := -1, -1
		for k, n := range tis {
			if n > best {
				ti, best = k, n
			}
		}
		r.ParTI[ti]++
	} else {
		r.HorsCensus++
	}
	cle := f0CleVie{Index: slot, Gen: ref.Gen}
	switch c, ok := vies[cle]; {
	case ok:
		r.Resolue++
		r.ParID[c.ID]++
		r.Dt = append(r.Dt, (int64(atUS)-int64(c.TsUS))/1000)
	case proj[cle]:
		r.ResolueProj++
	case index[slot]:
		r.IndexSeul++
	}
}

func (r *f0Resolution) ligne(nom string) string {
	if r.Total == 0 {
		return fmt.Sprintf("%-14s aucune occurrence", nom)
	}
	return fmt.Sprintf("%-14s presente %d/%d · vie ti=37 %d (%.1f %%) · vie ti=41 %d (%.1f %%) "+
		"· TOTAL RESOLUE %.1f %% · index seul %d · hors recensement %d · dt %s · archetypes %s "+
		"· objets %s",
		nom, r.Presente, r.Total,
		r.Resolue, 100*float64(r.Resolue)/float64(max(1, r.Presente)),
		r.ResolueProj, 100*float64(r.ResolueProj)/float64(max(1, r.Presente)),
		100*float64(r.Resolue+r.ResolueProj)/float64(max(1, r.Presente)),
		r.IndexSeul, r.HorsCensus, f0Mediane(r.Dt), f0TopTI(r.ParTI), f0TopID(r.ParID))
}

func f0Mediane(v []int64) string {
	if len(v) == 0 {
		return "n/a"
	}
	s := append([]int64(nil), v...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return fmt.Sprintf("mediane %+d ms [%+d..%+d]", s[len(s)/2], s[0], s[len(s)-1])
}

func f0TopTI(m map[int]int) string {
	type kv struct{ k, v int }
	var l []kv
	for k, v := range m {
		l = append(l, kv{k, v})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].v > l[j].v })
	var parts []string
	for i, e := range l {
		if i >= 4 {
			break
		}
		parts = append(parts, fmt.Sprintf("ti=%d:%d", e.k, e.v))
	}
	return "[" + fmt.Sprint(parts) + "]"
}

func f0TopID(m map[uint32]int) string {
	type kv struct {
		k uint32
		v int
	}
	var l []kv
	for k, v := range m {
		l = append(l, kv{k, v})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].v > l[j].v })
	var parts []string
	for i, e := range l {
		if i >= 5 {
			break
		}
		parts = append(parts, fmt.Sprintf("0x%08x:%d", e.k, e.v))
	}
	return "[" + fmt.Sprint(parts) + "]"
}

// f0CensusTI recense, par slot, les archetypes que les IMAGES-CLES lui voient porter.
func f0CensusTI(dir string) map[uint64]map[int]int {
	out := map[uint64]map[int]int{}
	for c, n := 1, CountFilmChunks(dir); c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				s := uint64(r.Slot)
				if out[s] == nil {
					out[s] = map[int]int{}
				}
				out[s][r.TI]++
			}
		}
	}
	return out
}

// f0Bases : les deux bases mesurees cote a cote.
var f0Bases = []uint64{0, f0BaseRef}

// f0Agg agrege les resolutions du parc : [base][position de reference].
type f0Agg map[uint64][3]*f0Resolution

func f0NouvelAgg() f0Agg {
	out := f0Agg{}
	for _, b := range f0Bases {
		out[b] = [3]*f0Resolution{f0NouvelleResolution(), f0NouvelleResolution(), f0NouvelleResolution()}
	}
	return out
}

// TestF0Deploiement103 : questions 1 a 3 du lot F.0.
func TestF0Deploiement103(t *testing.T) {
	root, ids := f0Films(t)
	cat := f0Catalogue(t)
	cartes := f0Cartes(t, cat)
	objets := f0ManifesteObjets(t)
	t.Logf("manifeste du titre : %d objets `eqip` nommes", len(objets))

	parc, temoin := f0NouvelAgg(), f0NouvelAgg()
	parcOrigine := map[string][2]int{}
	parcIDOrigine := map[string][2]int{}
	parcVoisin := map[string][2]int{}
	var d12 []string

	for _, id := range ids {
		e, ok := cartes[id]
		if !ok {
			t.Logf("film %s : absent de %s — ignore", id, f0MapsEnv)
			continue
		}
		f := f0Charge(t, root, id, e)
		census := f0CensusTI(filepath.Join(root, id))
		vies, index := f0ViesEquipement(f)
		proj := f0ViesProjectile(f)
		t.Logf("")
		t.Logf("######## FILM %s (carte %s) ########", id, e.Module)
		t.Logf("  marche : %d listes non vides, %d fermees proprement (%.1f %%) · "+
			"%d occurrences du type 103 dont %d en tete de liste",
			f.Listes, f.ListesPropres, 100*float64(f.ListesPropres)/float64(max(1, f.Listes)),
			len(f.Ev103), f.Ev103Tete)
		t.Logf("  balayage `ti=37` : %d ancres, %d acceptees (BRUT, toutes categories) · "+
			"production : %d confirmees, %d poses · calibration MPP %s",
			f.CreStats.Anchors, f.CreStats.Accepted, f.PlaceStats.Confirmed,
			f.PlaceStats.Placements, f.PlaceStats.Calibration.Widths)

		// --- QUESTION 1 : les trois references, resolues, base par base ---
		film := f0NouvelAgg()
		for _, ev := range f.Ev103 {
			for _, b := range f0Bases {
				for i := 0; i < 3; i++ {
					film[b][i].ajoute(ev.Refs[i], b, ev.TsUS, census, vies, index, proj)
					parc[b][i].ajoute(ev.Refs[i], b, ev.TsUS, census, vies, index, proj)
				}
			}
		}
		t.Logf("  vies decodees : %d creations `ti=37` (cles distinctes %d) · %d trajectoires "+
			"`ti=41` · %d slots recenses aux images-cles",
			len(f.Creations), len(vies), len(proj), len(census))
		f0LogAgg(t, "  Q1", film)
		f0Temoin(t, f, census, vies, index, proj, temoin)

		// --- QUESTIONS 2 et 3 : le 103 designe-t-il la pose ? ---
		a, vu := f0LitArtefact(t, id)
		if !vu {
			continue
		}
		poses, orphArt, orphFilm := f0Joint(f, a)
		t.Logf("  jointure artefact <-> film : %d poses appariees · %d poses d artefact sans "+
			"correspondance · %d poses de film non publiees (schema %d)",
			len(poses), orphArt, orphFilm, a.SchemaVersion)
		refs := f0ViesReferencees(f, f0BaseRef)
		fins := f0FinDeVie(a)
		parOrigine := map[string][2]int{}
		parIDOrigine := map[string][2]int{}
		for _, p := range poses {
			vue := 0
			if refs[p.Cle] {
				vue = 1
			}
			f0Bump(parOrigine, p.Origin, vue)
			f0Bump(parcOrigine, p.Origin, vue)
			cle := fmt.Sprintf("%s/0x%08x/%s", p.Family, p.ID, p.Origin)
			f0Bump(parIDOrigine, cle, vue)
			f0Bump(parcIDOrigine, cle, vue)
			// QUESTION 3 : les cas de D12 — une pose `deployed` a l image EXACTE de la fin de
			// vie de son poseur, c est-a-dire un lacher a la mort promu par la seule clause
			// de distance de `equipmentOrigin`.
			if p.Origin == "deployed" && p.Owner >= 0 &&
				f0EstFinDeVie(fins, uint32(p.Owner), p.T0) {
				d12 = append(d12, fmt.Sprintf(
					"%s t0=%d 0x%08x %-8s poseur=%d · un 103 designe sa vie : %v",
					id, p.T0, p.ID, p.Family, p.Owner, refs[p.Cle]))
			}
		}
		f0LogCouverture(t, "  Q2 origine", parOrigine)
		f0LogCouverture(t, "  Q2 objet  ", parIDOrigine)
		f0PieceVoisine(t, poses, refs, parcVoisin)
	}

	t.Logf("")
	t.Logf("######## PARC ########")
	f0LogAgg(t, "  Q1", parc)
	f0LogAgg(t, "  Q1 TEMOIN", temoin)
	f0LogCouverture(t, "  Q2 origine", parcOrigine)
	f0LogCouverture(t, "  Q2 objet  ", parcIDOrigine)
	f0LogCouverture(t, "  Q3 piece voisine", parcVoisin)
	t.Logf("  Q3 — cas de D12 (pose `deployed` a l image exacte de la fin de vie du poseur) : %d",
		len(d12))
	for _, l := range d12 {
		t.Logf("      %s", l)
	}
}

// f0LogAgg ecrit les resolutions base par base, reference par reference.
func f0LogAgg(t *testing.T, titre string, a f0Agg) {
	t.Helper()
	bases := make([]uint64, 0, len(a))
	for b := range a {
		bases = append(bases, b)
	}
	sort.Slice(bases, func(i, j int) bool { return bases[i] < bases[j] })
	for _, b := range bases {
		for i, r := range a[b] {
			t.Logf("%s %s", titre, r.ligne(fmt.Sprintf("base %d ref%d", b, i)))
		}
	}
}

func f0Bump(m map[string][2]int, cle string, vue int) {
	c := m[cle]
	m[cle] = [2]int{c[0] + vue, c[1] + 1}
}

// f0ViesEquipement rend les vies `ti=37` du film (cle -> identite et instant) et l ensemble
// des index seuls — les deux dont la question 1 a besoin.
func f0ViesEquipement(f f0Film) (map[f0CleVie]f0Cible, map[uint64]bool) {
	vies := map[f0CleVie]f0Cible{}
	index := map[uint64]bool{}
	for _, c := range f.Creations {
		k := f0CleVie{Index: uint64(c.Slot), Gen: c.Gen}
		// A cle egale, la creation la PLUS ANCIENNE est gardee : le pool de slots reboucle et
		// la generation ne fait que 2 bits, donc une cle peut servir deux fois dans un film.
		// L ecart de temps publie dit si la resolution tombe sur la bonne occurrence.
		if prev, vu := vies[k]; !vu || c.TimestampUS < prev.TsUS {
			vies[k] = f0Cible{ID: uint32(c.MPPVal[MPPWord32]), TsUS: c.TimestampUS}
		}
		index[uint64(c.Slot)] = true
	}
	return vies, index
}

// f0ViesProjectile rend les cles de vie des trajectoires `ti=41` du film.
func f0ViesProjectile(f f0Film) map[f0CleVie]bool {
	out := map[f0CleVie]bool{}
	for _, p := range f.Projectiles {
		out[f0CleVie{Index: uint64(p.Slot), Gen: p.Gen}] = true
	}
	return out
}

// f0ViesReferencees rend les cles de vie qu AU MOINS UNE reference d un 103 designe.
func f0ViesReferencees(f f0Film, base uint64) map[f0CleVie]bool {
	out := map[f0CleVie]bool{}
	for _, ev := range f.Ev103 {
		for _, r := range ev.Refs {
			if k, ok := f0CleDe(r, base); ok {
				out[k] = true
			}
		}
	}
	return out
}

// f0Temoin tire, en MEME NOMBRE que les references presentes du film, des references au
// hasard dans le domaine de 13 bits, et leur applique les MEMES resolutions. C est le temoin
// negatif obligatoire : sans lui, un taux de resolution eleve pourrait n etre que la densite
// d occupation du domaine.
func f0Temoin(t *testing.T, f f0Film, census map[uint64]map[int]int,
	vies map[f0CleVie]f0Cible, index map[uint64]bool, proj map[f0CleVie]bool, agg f0Agg) {
	t.Helper()
	// Graine FIXE : le temoin doit etre rejouable a l identique, comme tout le reste.
	rng := rand.New(rand.NewSource(20260913))
	local := f0NouvelAgg()
	for _, ev := range f.Ev103 {
		for i, r := range ev.Refs {
			if !r.Present {
				continue
			}
			faux := r7RefVal{Present: true, Dom: r.Dom, Width: r.Width,
				Index: uint64(rng.Intn(1 << 13)), Gen: uint32(rng.Intn(4))}
			for _, b := range f0Bases {
				local[b][i].ajoute(faux, b, ev.TsUS, census, vies, index, proj)
				agg[b][i].ajoute(faux, b, ev.TsUS, census, vies, index, proj)
			}
		}
	}
	f0LogAgg(t, "  Q1 temoin", local)
}

// f0VoisinageUS est le rayon de la recherche d une PIECE ENGENDREE voisine : 5 s. Un mur
// s ouvre en moins d une seconde apres avoir ete lance (mesure E0 §1) ; 5 s laisse la marge
// d un lancer long sans jamais atteindre l ecart median de 17 s qui separe deux gestes
// distincts du meme joueur.
const f0VoisinageUS = 5_000_000

// f0PieceVoisine mesure la LECTURE INDIRECTE, celle qui interesse F.1 : une pose d APPAREIL
// PORTE (un identifiant qu aucun 103 ne designe jamais) est-elle accompagnee, dans les 5 s,
// d une pose de PIECE ENGENDREE de la meme famille (un identifiant que le 103 designe) ?
//
// Si oui pour les `deployed` et non pour les `dropped`, le film porte bien le fait
// « deploye » pour ces familles — indirectement, par la piece, et non par une regle de
// distance. Sinon, il ne le porte pas, et F.1 doit se rabattre sur le fait temporel.
//
// LES DEUX COTES SE LISENT DANS LA MESURE, PAS DANS UNE LISTE ECRITE : « piece engendree »
// est ici defini par « un 103 designe cette vie », ce que la question 2 vient d etablir.
func f0PieceVoisine(t *testing.T, poses []f0Pose, refs map[f0CleVie]bool, agg map[string][2]int) {
	t.Helper()
	pieces := map[uint32]bool{}
	for _, p := range poses {
		if refs[p.Cle] {
			pieces[p.ID] = true
		}
	}
	local := map[string][2]int{}
	for _, p := range poses {
		if pieces[p.ID] {
			continue // c est la piece elle-meme, pas l appareil
		}
		voisin := 0
		for _, q := range poses {
			if !pieces[q.ID] || q.Family != p.Family {
				continue
			}
			d := int64(q.T0US) - int64(p.T0US)
			if d < 0 {
				d = -d
			}
			if d <= int64(f0VoisinageUS) {
				voisin = 1
				break
			}
		}
		cle := p.Family + "/" + p.Origin
		f0Bump(local, cle, voisin)
		f0Bump(agg, cle, voisin)
	}
	f0LogCouverture(t, "  Q3 piece voisine", local)
}

// f0LogCouverture ecrit une table `cle -> (designee par un 103) / (total)`.
func f0LogCouverture(t *testing.T, titre string, m map[string][2]int) {
	t.Helper()
	cles := make([]string, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	for _, k := range cles {
		v := m[k]
		t.Logf("%s : %-40s %4d / %4d = %5.1f %%", titre, k, v[0], v[1],
			100*float64(v[0])/float64(max(1, v[1])))
	}
}
