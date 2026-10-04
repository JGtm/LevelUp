//go:build research

package grammar

// r_loc_ls_research_test.go — CAMPAGNE DE GRAMMAIRE, CHANTIER r_loc (2026-10-02) : R-LS (D-67, la
// signature du localisateur figee sur le slot 123) et R-L1 (c) (la vue A des paquets a evenements).
// Un instrument de recherche : aucun fichier de production n est touche, aucune sortie ne change.
//
// La marche est celle de la campagne ([cmMarcher], meme monde, meme pilotage, meme classement que
// la carte v2). Seul le LOCALISATEUR de debut de liste change, par la variante `tete` :
//
//	ref        : 123 strict, puis chaine de tete / fermeture (= `debutDeLaListe`, controle d egalite)
//	ls         : 123 strict, sinon strict sur un AUTRE slot lie a un archetype « high-frequency »
//	ls+libre   : ls, sinon repli a largeur libre du slot 123 (`marchLocateFallback`)
//	libre      : 123 strict, sinon repli a largeur libre (controle : le repli seul)
//	ls+vueA    : ls, sinon l ORACLE de fin de vue A (r_loc_vuea_research_test.go)
//	ls2        : 123 strict, sinon fermeture par NEW de tete (comme ref), et SEULEMENT ENSUITE la
//	             signature high-frequency (ordre qui ne retire rien de ce que ref localise)
//
// Chaque paquet ferme passe au juge des invariants de l ecrivain ([cmContredit], decision D2).
//
// Rejouable (un film a la fois, plafond 4 Gio, sorties hors de data/ ; RLOC_VARIANTES restreint
// les marches jouees apres `ref`, par exemple `ls,ls2`) :
//
//	RLOC_RACINE=<film_chunks> RLOC_FILMS=<id,id> RLOC_SORTIE=<dir> [RLOC_CONTROLE=300] \
//	  go test -tags=research -count=1 -timeout 240m -run '^TestRLocLocalisateur$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
)

// rlocVariantes : l ordre des marches ; `ref` d abord (elle fixe les paquets de reference).
var rlocVariantes = []string{"ref", "ls", "ls+libre", "libre", "ls+vueA", "ls2"}

// rlocTrace : ce que le localisateur a decide pour le DERNIER paquet a evenements.
type rlocTrace struct {
	passe     string
	slotHF    uint32
	s123      int
	ecartProd bool
	vueA      rlocEssaiVueA
	ctl       string
}

// rlocArchetypesHF : les archetypes a un seul composant nomme `high-frequency` (lu par NOM).
func rlocArchetypesHF(reg *Registry) map[uint32]bool {
	out := map[uint32]bool{}
	for _, a := range reg.Archetypes {
		if len(a.Components) == 1 && a.Components[0] == "high-frequency" {
			out[uint32(a.Index)] = true //nolint:gosec // indice de registre
		}
	}
	return out
}

// rlocSignatures : en UNE boucle, la premiere signature stricte du slot 123 (sans test de
// generation, comme `marchLocateStrict`) et la premiere signature stricte sur un autre slot lie
// a un archetype high-frequency (generation du profil). -1 : absente.
func rlocSignatures(pay []byte, w *World, cfg FrameConfig, hf map[uint32]bool) (s123, sHF int, slotHF uint32) {
	defer cfg.Obs.neutraliserEtatsDeMouvement()()
	s123, sHF = -1, -1
	nb := len(pay) * 8
	for s, largeur := 2, largeurDeSignature(cfg); s+largeur < nb; s++ {
		if bitAvant(pay, s) != 0 {
			continue
		}
		rec, end, ok := TryDeltaAt(pay, s, w, cfg)
		if !ok || end != s+largeurDeSignature(cfg) || len(rec.Trace.Comps) != 1 {
			continue
		}
		if rec.Slot == marchSignatureSlot {
			return s, sHF, slotHF
		}
		if sHF < 0 && hf[rec.TypeIndex] && w.GenerationMatches(rec.ID, cfg.Profil.Grammaire.GenerationStricte) {
			sHF, slotHF = s, rec.Slot
		}
	}
	return -1, sHF, slotHF
}

// bitAvant lit le bit `s-1` (le terminateur de la vue A devant le premier record).
func bitAvant(pay []byte, s int) uint {
	return uint(pay[(s-1)>>3]>>(7-uint((s-1)&7))) & 1 //nolint:gosec // MSB-first, comme source.BitAt
}

// rlocLocaliser est le localisateur de la variante `v` ; il note sa decision dans `tr`.
func rlocLocaliser(v string, hf map[uint32]bool, pay []byte, w *World, cfg FrameConfig, tr *rlocTrace,
	oracle func([]byte, *World, FrameConfig) rlocEssaiVueA) (int, bool) {
	*tr = rlocTrace{s123: -1}
	s123, sHF, slotHF := rlocSignatures(pay, w, cfg, hf)
	tr.s123 = s123
	debut, passe := -1, ""
	switch {
	case s123 >= 0:
		debut, passe = s123, "strict-123"
	case v != "ref" && v != "libre" && v != "ls2" && sHF >= 0:
		debut, passe, tr.slotHF = sHF, "strict-hf", slotHF
	}
	if debut < 0 && (v == "libre" || v == "ls+libre") {
		if s := marchLocateFallback(pay, w, cfg); s >= 0 {
			debut, passe = s, "libre-123"
		}
	}
	var d int
	var ok bool
	if debut < 0 {
		d, ok = debutParFermeture(pay, candidatsDeTete(pay, len(pay)*8, w), w, cfg)
		tr.passe = "fermeture-neuf"
		if !ok {
			tr.passe = "aucune"
			if v == "ls2" && sHF >= 0 {
				tr.slotHF = slotHF
				d, ok = debutParChaine(pay, sHF, candidatsDeTete(pay, sHF, w), w, cfg)
				d, ok, tr.passe = rlocSiChaine(d, ok, "strict-hf")
				return d, ok
			}
			if v == "ls+vueA" && oracle != nil {
				tr.vueA = oracle(pay, w, cfg)
				if tr.vueA.debut >= 0 {
					d, ok, tr.passe = tr.vueA.debut, true, "oracle-vueA"
				}
			}
		}
	} else {
		d, ok = debutParChaine(pay, debut, candidatsDeTete(pay, debut, w), w, cfg)
		tr.passe = passe
		if ok {
			tr.passe = passe + "+chaine"
		}
	}
	if v == "ref" {
		dp, _ := debutDeLaListe(pay, w, cfg)
		tr.ecartProd = dp != d
	}
	return d, ok
}

// rlocRef : ce que la marche `ref` a juge de chaque paquet.
type rlocRef struct {
	ferme, sain map[[2]int]bool
	utiles      map[[2]int]int
}

// rlocMesure ecoute une marche et la juge contre `ref` (nil pendant la marche `ref`).
type rlocMesure struct {
	v                                   string
	tr                                  *rlocTrace
	ctl                                 *rlocControle
	chk                                 *cmCollecteur
	ref                                 *rlocRef
	out                                 *rlocRef
	paquets, evenements, nonLoc         int
	fermes, fermesSains                 int
	utilesFermes, utilesSains           int
	gagnes, gagnesContre, perdus        int
	perdusSains, utilesPerdusSains      int
	evFermes, evFermesSains, ecartsProd int
	passes                              map[string]int
	slotsHF                             map[uint32]int
	genres                              map[string]int
	vueA                                cmTables
}

func (m *rlocMesure) debutDeChunk(c int, _ []byte, _ []FilmPacket, _ *World) { m.chk.chunk = c }

func (m *rlocMesure) paquet(_ int, p *cmPaquet, _ *World) {
	m.paquets++
	cle := [2]int{p.d.Chunk, p.d.Index}
	ev := p.strict != -2
	if ev {
		m.evenements++
		m.passes[m.tr.passe]++
		if m.tr.ecartProd {
			m.ecartsProd++
		}
		if m.tr.passe == "strict-hf" || m.tr.passe == "strict-hf+chaine" {
			m.slotsHF[m.tr.slotHF]++
		}
		if p.debut < 0 {
			m.nonLoc++
			m.noterNonLocalise(p)
		}
		if m.tr.passe == "oracle-vueA" {
			m.vueA.un("oracle", "debut trouve · "+rlocClasseEssais(m.tr.vueA.essais))
		}
	}
	sain := false
	if p.d.Fermee {
		m.fermes++
		m.utilesFermes += p.utilesFermes
		sain = len(cmContredit(m.chk, p)) == 0
		if sain {
			m.fermesSains++
			m.utilesSains += p.utilesFermes
		}
		if ev {
			m.evFermes++
			if sain {
				m.evFermesSains++
			}
		}
	}
	if m.out != nil {
		m.out.ferme[cle], m.out.sain[cle], m.out.utiles[cle] = p.d.Fermee, sain, p.utilesFermes
	}
	if ev && m.ctl != nil && m.tr.ctl != "" {
		etat := "ref non ferme"
		switch {
		case p.d.Fermee && sain:
			etat = "ref ferme sain"
		case p.d.Fermee:
			etat = "ref ferme contredit"
		}
		m.ctl.t.un("controle", cmJoindre(m.tr.ctl, etat))
	}
	if m.ref == nil {
		return
	}
	avant := m.ref.ferme[cle]
	switch {
	case p.d.Fermee && !avant:
		m.gagnes++
		if !sain {
			m.gagnesContre++
		}
	case !p.d.Fermee && avant:
		m.perdus++
		if m.ref.sain[cle] {
			m.perdusSains++
			m.utilesPerdusSains += m.ref.utiles[cle]
		}
	}
}

// noterNonLocalise range un paquet a evenements que la variante n a pas localise : genre de tete
// de la vue A (le seul lisible sans grammaire de charge) et taille.
func (m *rlocMesure) noterNonLocalise(p *cmPaquet) {
	g, _ := PacketHeadEventType(p.pay)
	m.genres[fmt.Sprintf("%d\t%s", g, rlocStatutGenre(g))]++
	if m.tr.vueA.essais > 0 || m.v == "ls+vueA" {
		m.vueA.un("oracle", "aucun debut · "+rlocClasseEssais(m.tr.vueA.essais))
	}
}

func (m *rlocMesure) finDeFilm() {}

// rlocStatutGenre : ce que le depot sait de la charge d un genre de vue A (relu dans le code le
// 2026-10-02 : seules des TETES de six genres sont lues, jamais une charge jusqu a son terme).
func rlocStatutGenre(g int) string {
	switch g {
	case bipedBoardVehicleType, bipedPickupType, zoomEventType, EventUnitExitVehicle, TypeTirArme, translocEventType:
		return "tete lue en Go (charge partielle)"
	}
	return "aucune grammaire de charge en Go"
}

// TestRLocLocalisateur joue les variantes du localisateur sur les films de RLOC_FILMS.
func TestRLocLocalisateur(t *testing.T) {
	racine, sortie := os.Getenv("RLOC_RACINE"), os.Getenv("RLOC_SORTIE")
	var films []string
	for _, x := range strings.Split(os.Getenv("RLOC_FILMS"), ",") {
		if x = strings.TrimSpace(x); x != "" {
			films = append(films, x)
		}
	}
	if racine == "" || sortie == "" || len(films) == 0 {
		t.Skip("RLOC_RACINE, RLOC_FILMS et RLOC_SORTIE requis")
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
	controle := 300
	if v, err := strconv.Atoi(os.Getenv("RLOC_CONTROLE")); err == nil {
		controle = v
	}
	utiles := cmUtiles(t)
	tables := map[string][]string{}
	for _, id := range films {
		rlocUnFilm(t, racine, id, utiles, controle, tables)
	}
	rlocEcrire(t, abs, tables)
}

// rlocUnFilm joue les variantes sur un film.
func rlocUnFilm(t *testing.T, racine, id string, utiles UsagesProduit, controle int, tables map[string][]string) {
	garde := filmproc.Arm("campagne/r_loc", 4, func(pic uint64) {
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
	hf := rlocArchetypesHF(f.reg)
	tables["registre"] = append(tables["registre"], rlocRegistre(f, hf)...)
	ref := &rlocRef{ferme: map[[2]int]bool{}, sain: map[[2]int]bool{}, utiles: map[[2]int]int{}}
	ctl := &rlocControle{max: controle, t: cmTables{}}
	var resume []string
	for _, v := range rlocVariantes {
		if filtre := os.Getenv("RLOC_VARIANTES"); v != "ref" && filtre != "" && !strings.Contains(","+filtre+",", ","+v+",") {
			continue
		}
		tr := &rlocTrace{}
		m := &rlocMesure{v: v, tr: tr, chk: nouveauCollecteur(f, b, nil), passes: map[string]int{},
			slotsHF: map[uint32]int{}, genres: map[string]int{}, vueA: cmTables{}, ctl: ctl}
		if v == "ref" {
			m.out = ref
		} else {
			m.ref = ref
		}
		var oracle func([]byte, *World, FrameConfig) rlocEssaiVueA
		if v == "ls+vueA" {
			oracle = rlocOracleVueA
		}
		vv := v
		tete := func(int) func([]byte, *World, FrameConfig) (int, bool) {
			return func(pay []byte, w *World, cfg FrameConfig) (int, bool) {
				d, okL := rlocLocaliser(vv, hf, pay, w, cfg, tr, oracle)
				if vv == "ref" && tr.passe == "strict-123" {
					tr.ctl = ctl.essayer(pay, w, cfg, tr.s123)
				}
				return d, okL
			}
		}
		rep, _, _ := cmMarcher(f, cmVariante{tete: tete}, m)
		tables["variantes"] = append(tables["variantes"], rlocLigne(f, v, rep, m))
		tables["passes"] = append(tables["passes"], rlocCles(f, v, m.passes)...)
		for s, n := range m.slotsHF {
			tables["slots_hf"] = append(tables["slots_hf"], fmt.Sprintf("%s\t%s\t%s\t%d\t%d", f.id, f.build, v, s, n))
		}
		for g, n := range m.genres {
			tables["genres"] = append(tables["genres"], fmt.Sprintf("%s\t%s\t%s\t%s\t%d", f.id, f.build, v, g, n))
		}
		for _, k := range cmCles(m.vueA["oracle"]) {
			tables["oracle"] = append(tables["oracle"], fmt.Sprintf("%s\t%s\t%s\t%s\t%d", f.id, f.build, v, k, m.vueA["oracle"][k].n))
		}
		resume = append(resume, fmt.Sprintf("%s %d/%d nl=%d", v, rep.PaquetsFermes, rep.Paquets, m.nonLoc))
	}
	for _, k := range cmCles(ctl.t["controle"]) {
		tables["controle"] = append(tables["controle"], fmt.Sprintf("%s\t%s\t%s\t%d", f.id, f.build, k, ctl.t["controle"][k].n))
	}
	t.Logf("%s %s : %s ; pic %d Mio, %s", id, f.build, strings.Join(resume, " · "), garde.Peak()>>20,
		time.Since(debut).Round(time.Second))
}

// rlocLigne : la ligne `variantes` d une marche.
func rlocLigne(f *cmFilm, v string, r FrameClosureReport, m *rlocMesure) string {
	return fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d",
		f.id, f.build, v, r.Paquets, m.evenements, m.nonLoc, r.ListesNonLocalisees, m.fermes, m.fermesSains,
		r.Utiles.Records, m.utilesFermes, m.utilesSains, r.Bloquants[CauseHorsCadre].Paquets,
		m.gagnes, m.gagnesContre, m.perdus, m.perdusSains, m.utilesPerdusSains, m.evFermes, m.evFermesSains, m.ecartsProd)
}

// rlocCles : une table de comptes en lignes triees.
func rlocCles(f *cmFilm, v string, m map[string]int) []string {
	var out []string
	for k, n := range m {
		out = append(out, fmt.Sprintf("%s\t%s\t%s\t%s\t%d", f.id, f.build, v, k, n))
	}
	sort.Strings(out)
	return out
}

// rlocRegistre : les archetypes high-frequency du registre (par nom), les autres archetypes qui
// portent un composant de ce nom (D-49), et les slots que le monde lie a un archetype
// high-frequency, avec le nombre de chunks ou ils le sont.
func rlocRegistre(f *cmFilm, hf map[uint32]bool) []string {
	var out []string
	add := func(m, v string) { out = append(out, fmt.Sprintf("%s\t%s\t%s\t%s", f.id, f.build, m, v)) }
	var idx, autres []string
	for _, a := range f.reg.Archetypes {
		if hf[uint32(a.Index)] { //nolint:gosec // indice de registre
			idx = append(idx, fmt.Sprintf("ti=%d", a.Index))
			continue
		}
		for i, c := range a.Components {
			if c == "high-frequency" {
				autres = append(autres, fmt.Sprintf("ti=%d i%d/%d", a.Index, i, len(a.Components)))
			}
		}
	}
	add("archetypes high-frequency (seul composant)", strings.Join(idx, " "))
	add("autres archetypes portant high-frequency", strings.Join(autres, " "))
	monde := NewWorld(f.reg)
	marche := f.fc.MarcheDImageCle()
	parSlot := map[uint32]int{}
	chunks := 0
	for _, c := range f.fc.ChunkNumbers() {
		data, pks, ok := f.fc.ChunkAt(c)
		if !ok {
			continue
		}
		chunks++
		monde.PoserChunkCourant(c)
		lierLeChunkAuMonde(monde, marche, data, pks, nil)
		for s, st := range monde.slots {
			if hf[st.TypeIndex] {
				parSlot[s]++
			}
		}
	}
	var slots []string
	for s, n := range parSlot {
		slots = append(slots, fmt.Sprintf("%d:%d", s, n))
	}
	sort.Strings(slots)
	add("chunks", strconv.Itoa(chunks))
	add("slots lies high-frequency (slot:chunks)", strings.Join(slots, " "))
	return out
}

// rlocEcrire ecrit les tables.
func rlocEcrire(t *testing.T, abs string, tables map[string][]string) {
	tetes := map[string]string{
		"variantes": "film\tbuild\tvariante\tpaquets\tevenements\tnon_localises\tlistes_non_localisees_carte\tfermes\t" +
			"fermes_sains\tutiles_lus\tutiles_fermes\tutiles_fermes_sains\thors_cadre\tgagnes\tgagnes_contredits\tperdus\t" +
			"perdus_ref_sains\tutiles_perdus_ref_sains\tev_fermes\tev_fermes_sains\tecarts_ref_production",
		"passes":   "film\tbuild\tvariante\tpasse\tpaquets",
		"slots_hf": "film\tbuild\tvariante\tslot\tpaquets",
		"genres":   "film\tbuild\tvariante\tgenre_tete\tstatut_go\tpaquets",
		"oracle":   "film\tbuild\tvariante\tcle\tpaquets",
		"controle": "film\tbuild\tcle\tpaquets",
		"registre": "film\tbuild\tmesure\tvaleur",
	}
	for nom, lignes := range tables {
		sort.Strings(lignes)
		brut := tetes[nom] + "\n" + strings.Join(lignes, "\n") + "\n"
		if err := os.WriteFile(filepath.Join(abs, "rloc_"+nom+".tsv"), []byte(brut), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// rlocSiChaine rend le debut de [debutParChaine], toujours localise (le debut de la signature est
// garde quand aucune chaine de tete ne tient), et la passe qui l a trouve.
func rlocSiChaine(d int, chaine bool, passe string) (int, bool, string) {
	if chaine {
		return d, true, passe + "+chaine"
	}
	return d, false, passe
}
