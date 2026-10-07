//go:build research

package grammar

// ri27c_killsource_research_test.go — LES MESURES QUI OUVRENT 2.7.c (plan de l etape 2 de la
// representation intermediaire, item 2.7.c0), cote grammaire. killsource lit le film lui-meme ; ces
// instruments rendent en TSV ce que la marche unique lit des memes faits, et l instrument de
// killsource (`facts/killsource/ri27c_research_test.go`) les confronte a sa propre lecture. Aucun
// fichier de production n est touche.
//
//	TestRI27cMorts        les dead-states que la marche des trames lit (tous archetypes), avec leur
//	                      trame (chunk, rang, horodatage, verdict, debut de vue B) et le premier bit
//	                      du composant ; les listes non localisees recuperees comme le canal des
//	                      morts les recupere ; la bande bipede de la phase des images-cles. Deux
//	                      decoupages MPP : celui du format (le profil de killsource aujourd hui) et
//	                      celui que le film declare (la cuisson).
//	TestRI27cCalibration  les scores de la calibration de killsource (largeur d axe uniforme, mot de
//	                      poignee au triplet de la carte) sous le monde des preliminaires de la
//	                      marche — la liaison des images-cles de chaque chunk —, sur l echantillon que
//	                      killsource prend.
//
// Les mesures de la vue A (`TestRI27cVueA`, `TestRI27cVueASansQueue`) sont dans
// `ri27c_vue_a_research_test.go`.
//
// Contexte : celui de la cuisson ([ri27cContexte]) — le profil de depart de killsource (generation
// stricte, largeurs de la carte, controle de corruption du film), la largeur du mot de poignee de
// killsource quand elle est discriminee (RI27C_POIGNEE), la carte du catalogue.
//
//	RI27C_FILMS=<id,...> RI27C_RACINE=<film_chunks> RI27C_OUT=<dossier> \
//	  [RI27C_CARTES=<id=Carte;...>] [RI27C_POIGNEE=<id=iw,...>] \
//	  go test -tags=research -count=1 -run '^TestRI27c' -timeout 120m \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ri27cGenreKill est le genre du kill-event (`PlayerKilledEvent`) dans la vue A.
const ri27cGenreKill = 85

// ri27cEnv lit les variables de l instrument ; Skip sans elles.
func ri27cEnv(t *testing.T) (films []string, racine, sortie string) {
	t.Helper()
	f, racine, sortie := os.Getenv("RI27C_FILMS"), os.Getenv("RI27C_RACINE"), os.Getenv("RI27C_OUT")
	if f == "" || racine == "" || sortie == "" {
		t.Skip("instrument : RI27C_FILMS, RI27C_RACINE et RI27C_OUT requis")
	}
	if err := os.MkdirAll(sortie, 0o750); err != nil {
		t.Fatalf("sortie : %v", err)
	}
	return strings.Split(f, ","), racine, sortie
}

// ri27cCarte rend la carte d un film : RI27C_CARTES d abord, les faits d equivalence ensuite.
func ri27cCarte(t *testing.T, court string) string {
	t.Helper()
	for _, kv := range strings.Split(os.Getenv("RI27C_CARTES"), ";") {
		if k, v, ok := strings.Cut(kv, "="); ok && k == court {
			return v
		}
	}
	return ri27bCarte(t, court)
}

// ri27cPoignee rend la largeur du mot de poignee que killsource a retenue pour un film, 0 sans
// decision (l invariant).
func ri27cPoignee(court string) uint {
	for _, kv := range strings.Split(os.Getenv("RI27C_POIGNEE"), ",") {
		if k, v, ok := strings.Cut(kv, "="); ok && k == court {
			n, _ := strconv.Atoi(v)
			return uint(n) //nolint:gosec // largeur de 1 a 3
		}
	}
	return 0
}

// ri27cContexte ouvre le contexte du film comme la cuisson l ouvre apres killsource : le profil de
// depart de killsource (defaut, generation stricte, largeurs de la carte, controle de corruption du
// film, mot de poignee retenu), pose ENTIER, puis les largeurs de la carte, puis — `declare` — le
// decoupage MPP que le film declare.
func ri27cContexte(t *testing.T, dir, court string, declare bool) *FilmContext {
	t.Helper()
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s : %v", dir, err)
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	entree, err := cat.Lookup(ri27cCarte(t, court))
	if err != nil {
		t.Fatalf("carte de %s : %v", court, err)
	}
	fc := NewFilmContextForMap(film, &entree, nil)
	bal := ProfilDeBalayageParDefaut()
	bal.Grammaire.GenerationStricte = true
	_ = bal.PoserLargeursObjetDuMondeDepuisDecoupage(entree.Layout())
	bal, _ = GrammaireSousFilm(bal, film)
	if iw := ri27cPoignee(court); iw > 0 {
		bal.Mouvement.Traversal.IndexW = iw
	}
	fc.PoserProfilDeBalayage(bal)
	fc.PoserLargeursObjetDuMondeDepuisDecoupage(entree.Layout())
	if declare {
		if res := fc.ResolutionMPP(); res.Decide() {
			fc.PoserMPP(res.Widths)
		}
	}
	return fc
}

// ri27cCanal recolte les dead-states des records que la marche des trames lit, et ceux des listes
// qu elle ne localise pas, recuperees comme le canal des morts les recupere ([debutRecupere]).
type ri27cCanal struct {
	reg        *Registry
	m          *MarcheDistribuee
	court, dec string
	lignes     []string
	// bande : les slots extremes des records bipedes des images-cles.
	bandeLo, bandeHi int
	// paquets : trames a evenements, localisees par la marche, recuperees par signature, par
	// largeur libre, non localisees.
	evenements, localisees, parSignature, parLargeurLibre, nonLocalisees int
	// morts : les lignes de dead-state rangees.
	morts int
}

func (c *ri27cCanal) Interets() []Interet                          { return nouveauCanalDesMorts(c.reg).Interets() }
func (c *ri27cCanal) Clore(BilanDeMarche)                          {}
func (c *ri27cCanal) Brancher(_ *Observation, m *MarcheDistribuee) { c.m = m }

// ImageCle releve la bande bipede : les slots des records `ti=35` des images-cles.
func (c *ri27cCanal) ImageCle(p *lecture.Paquet, _ *MarcheDistribuee) {
	for i := range p.Records {
		if r := &p.Records[i]; r.TI == BipedTypeIndex {
			s := int(r.Vie.Slot)
			c.bandeLo, c.bandeHi = min(c.bandeLo, s), max(c.bandeHi, s)
		}
	}
}

// Trame range chaque record lu a dead-state `Mort`, avec sa trame et la classe de sa lecture.
func (c *ri27cCanal) Trame(p *lecture.Paquet) {
	recs, lus := c.m.recordsDeLaTrame()
	classe := "lue_par_la_marche"
	avec := listeAnnoncee(&p.VueA)
	if avec {
		c.evenements++
		if lus {
			c.localisees++
		}
	}
	if !lus && avec {
		var libre bool
		recs, lus, libre = c.m.recupererLaListe()
		switch {
		case !lus:
			classe = "non_localisee"
			c.nonLocalisees++
		case libre:
			classe = "recuperee_largeur_libre"
			c.parLargeurLibre++
		default:
			classe = "recuperee_signature"
			c.parSignature++
		}
	}
	if avec {
		c.lignes = append(c.lignes, fmt.Sprintf("T\t%s\t%s\t%d\t%d\t%d\t%s\t%d\t%d", c.court, c.dec, p.Chunk,
			p.Index, p.TS, classe, p.Fermeture.Verdict, p.Debut))
	}
	if !lus {
		return
	}
	for i := range recs {
		r := &recs[i]
		if r.Trace.Dead == nil || !r.Trace.Dead.Mort {
			continue
		}
		d := r.Trace.Dead
		bit := -1
		for _, cp := range r.Trace.Comps {
			if cp.Name == deadStateComponentName {
				bit = cp.StartBit
				break
			}
		}
		c.morts++
		c.lignes = append(c.lignes, fmt.Sprintf("M\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\t%d\t%d\t"+
			"%d\t%d\t%d\t%d\t%d\t%d", c.court, c.dec, p.Chunk, p.Index, p.TS, r.Slot, r.ID>>30, r.TypeIndex, bit,
			r.DesyncAt, ri27cIndexDuDeadState(c.reg, r.TypeIndex), classe, p.Fermeture.Verdict, p.Debut,
			d.EnumA, d.EnumB, d.Val0c, d.SrcTag0, d.GlobalID, d.Val0e))
	}
}

// ri27cIndexDuDeadState rend l index du dead-state dans l archetype `ti`, -1 s il ne le porte pas.
func ri27cIndexDuDeadState(reg *Registry, ti uint32) int {
	a, ok := reg.Archetype(int(ti))
	if !ok {
		return -1
	}
	for i, n := range a.Components {
		if n == deadStateComponentName {
			return i
		}
	}
	return -1
}

func TestRI27cMorts(t *testing.T) {
	films, racine, sortie := ri27cEnv(t)
	for _, court := range films {
		var lignes []string
		for _, declare := range []bool{false, true} {
			dec := map[bool]string{false: "format", true: "declare"}[declare]
			fc := ri27cContexte(t, filepath.Join(racine, court), court, declare)
			reg, err := fc.Registry()
			if err != nil {
				t.Fatalf("%s : registre : %v", court, err)
			}
			c := &ri27cCanal{reg: reg, court: court, dec: dec, bandeLo: 1 << 30, bandeHi: -1}
			if err := Distribuer(fc, c); err != nil {
				t.Fatalf("%s : distribution : %v", court, err)
			}
			mpp := fc.ProfilDeBalayage().MPP
			lignes = append(lignes, fmt.Sprintf("B\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\t%d", court, dec,
				c.bandeLo, c.bandeHi, c.evenements, c.localisees, c.parSignature, c.parLargeurLibre,
				c.nonLocalisees, mpp.String(), fc.EnTete().IDLowBits.Valeur))
			lignes = append(lignes, c.lignes...)
			t.Logf("%s %s : %d morts lues, bande [%d,%d], %d trames a evenements (%d localisees, %d+%d "+
				"recuperees, %d non localisees)", court, dec, c.morts, c.bandeLo, c.bandeHi,
				c.evenements, c.localisees, c.parSignature, c.parLargeurLibre, c.nonLocalisees)
			runtime.GC()
		}
		ri27cEcrire(t, filepath.Join(sortie, "morts_"+court+".tsv"), lignes)
	}
}

// ri27cEcrire ecrit les lignes d un film.
func ri27cEcrire(t *testing.T, chemin string, lignes []string) {
	t.Helper()
	if err := os.WriteFile(chemin, []byte(strings.Join(lignes, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("sortie : %v", err)
	}
}

// ri27cEchantillon rend l echantillon de calibration de killsource (`calibSample`) : les 400
// premiers paquets de replication sans evenement d au moins 400 octets, dans l ordre de la source,
// le terminateur exclu.
func ri27cEchantillon(f *source.Film) []types.Packet {
	var out []types.Packet
	for _, p := range f.AllPackets() {
		if p.Type != int(PacketTypeDelta) || len(p.Payload) < 400 || source.BitAt(p.Payload, 1) != 0 {
			continue
		}
		out = append(out, p)
		if len(out) >= 400 {
			break
		}
	}
	return out
}

// ri27cCandidat est une configuration balayee par la calibration de killsource.
type ri27cCandidat struct {
	nom string
	cfg FrameConfig
}

// ri27cCandidats rend les 21 largeurs d axe uniformes de l oracle, mot de poignee a l invariant,
// puis les trois largeurs du mot de poignee au triplet de la carte — les deux balayages de
// `calibrate.go`, dans le cadre de killsource (`DefaultFrameConfig` et le profil du contexte).
func ri27cCandidats(bal ProfilDeBalayage) []ri27cCandidat {
	lues, invariant := bal.Mouvement.WorldObject, ProfilDeBalayageParDefaut().Mouvement.Traversal
	var out []ri27cCandidat
	for aw := uint(6); aw <= 26; aw++ {
		cfg := DefaultFrameConfig()
		cfg.Profil = bal
		cfg.Profil.Mouvement.WorldObject = lues
		cfg.Profil.Mouvement.WorldObject.AxisW = [3]uint{aw, aw, aw}
		cfg.Profil.Mouvement.Traversal = invariant
		out = append(out, ri27cCandidat{fmt.Sprintf("axe=%d", aw), cfg})
	}
	for iw := uint(1); iw <= 3; iw++ {
		cfg := DefaultFrameConfig()
		cfg.Profil = bal
		cfg.Profil.Mouvement.WorldObject = lues
		cfg.Profil.Mouvement.Traversal = profile.PrecisionDescriptor{IndexW: iw, AxisW: invariant.AxisW}
		out = append(out, ri27cCandidat{fmt.Sprintf("poignee=%d", iw), cfg})
	}
	return out
}

// ri27cMarcher marche la boucle de records depuis `debut` sur `vues` vues (`walkFrom` de killsource).
func ri27cMarcher(pay []byte, w *World, cfg FrameConfig, debut, vues int) []FrameRecord {
	br := LecteurSur(pay)
	br.Skip(debut)
	var out []FrameRecord
	for v := 0; v < vues && len(pay)*8-br.BitPos() >= 8; v++ {
		recs, err := DecodeFrameRecords(br, w, cfg)
		out = append(out, recs...)
		if err != nil {
			break
		}
	}
	return out
}

// TestRI27cCalibration score les candidats de la calibration de killsource sous le monde des
// preliminaires de la marche : chunk par chunk, la table anticipee puis la liaison des images-cles
// du chunk, et chaque paquet de l echantillon marche depuis le bit 2 sur huit vues, monde restaure.
func TestRI27cCalibration(t *testing.T) {
	films, racine, sortie := ri27cEnv(t)
	var lignes []string
	for _, court := range films {
		fc := ri27cContexte(t, filepath.Join(racine, court), court, false)
		m, err := fc.nouveauMarcheurDesTrames(nil)
		if err != nil {
			t.Fatalf("%s : marche : %v", court, err)
		}
		parPos := map[int][]types.Packet{}
		ech := ri27cEchantillon(fc.Film())
		for _, p := range ech {
			parPos[p.Chunk] = append(parPos[p.Chunk], p)
		}
		cands := ri27cCandidats(fc.ProfilDeBalayage())
		scores := make([]int, len(cands))
		vus := 0
		for _, c := range m.chunks {
			m.monde.PoserChunkCourant(c)
			lierLesImagesClesDuChunk(m.monde, m.prel.liaison.rendre(c), nil)
			for _, p := range parPos[filmChunkPos(fc.Film(), c)] {
				vus++
				for k, cd := range cands {
					snap := m.monde.Snapshot()
					for _, r := range ri27cMarcher(p.Payload, m.monde, cd.cfg, 2, 8) {
						if r.DesyncAt == -1 && r.TypeIndex == BipedTypeIndex {
							scores[k]++
						}
					}
					m.monde.Restore(snap)
				}
			}
		}
		for k, cd := range cands {
			lignes = append(lignes, fmt.Sprintf("C\t%s\tpreliminaires\t%s\t%d\t%d\t%d", court, cd.nom, scores[k],
				vus, len(ech)))
		}
		t.Logf("%s : %d paquets d echantillon sur %d", court, vus, len(ech))
		runtime.GC()
	}
	ri27cEcrire(t, filepath.Join(sortie, "calibration_grammaire.tsv"), lignes)
}
