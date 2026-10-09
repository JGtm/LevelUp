package grammar

// keyframe_etats_scan.go — UNE SEULE DISTRIBUTION POUR CE QUE LES IMAGES-CLES DISENT DES BIPEDES
// (plan `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`, D1.1.6).
//
// [ScanEtatsDesImagesCles] marche UNE fois la phase des images-cles d un film et rend, par record
// bipede, les armes portees, l inventaire et la marque de portage, avec la couverture de la marche
// d ancres ([KeyframeWalkCoverage]). Elle remplace les trois balayages de fenetres qui marchaient
// chacun la phase ([ScanKeyframeLoadoutsMarche], [ScanKeyframeInventory], [ScanCarrierMarks], qui en
// sont desormais des projections). La valeur d un record ADMIS vient de la grammaire
// ([canalDeLEtatCompletBipede], `keyframe_etat_complet_admission.go`) ; celle d un record non admis,
// des fenetres de bits DERRIERE la lecture (`keyframe_etats_fenetre.go`), marquee recuperee et
// comptee. Les comptes des trois replis sont notes sur le contexte du film a chaque appel
// ([FilmContext.NoterReplis]) : la cuisson l appelle une fois.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// EtatsDesImagesCles est ce que la phase des images-cles d un film dit de ses bipedes.
type EtatsDesImagesCles struct {
	// Loadouts : les armes portees, un element par record bipede qui en porte, dans l ordre du film.
	Loadouts []types.KeyframeLoadout
	// Marche : la couverture de la marche d ancres des images-cles.
	Marche KeyframeWalkCoverage
	// Inventaire : un inventaire par record bipede lu, dans l ordre du film ; StatsInventaire, ses
	// denominateurs.
	Inventaire      []types.KeyframeInventory
	StatsInventaire types.KeyframeInventoryStats
	// Marques : les records bipedes porteurs de la marque de portage, et les instants balayes.
	Marques CarrierMarkScan
	// Admission : ce que la regle d admission a juge, et ce que les fenetres ont recu.
	Admission ComptesDeLAdmission
	// Recuperes : les records non admis dont une fenetre a rendu une valeur, dans l ordre du film
	// (`keyframe_etats_fenetre.go`).
	Recuperes []RecordRecupere
}

// ComptesDeLAdmission compte ce que la lecture de l etat complet du bipede a rendu et refuse.
type ComptesDeLAdmission struct {
	// Bipedes : records bipedes des images-cles ; SansCorps : ceux dont la phase n a pas parcouru le
	// corps (film sans registre).
	Bipedes, SansCorps int
	// Admis : records dont la valeur est publiee par la grammaire.
	Admis int
	// RefusNiFermeNiI22, RefusT1, RefusT2 : les records refuses, par raison ([raisonDeRefus]).
	RefusNiFermeNiI22, RefusT1, RefusT2 int
	// Debordements : occurrences dont la relecture ne tient pas l etendue de la marche (attendu 0).
	Debordements int
	// CapaciteHorsDomaine : records admis dont le rang de capacite lu est hors de 16..23 (non publie).
	CapaciteHorsDomaine int
	// FenetresArmes, FenetresInventaire, FenetresMarque : records bipedes non admis donnes a chaque
	// fenetre derriere la lecture — les comptes des trois replis (`keyframe_etats_fenetre.go`).
	FenetresArmes, FenetresInventaire, FenetresMarque int
}

// ScanEtatsDesImagesCles marche la phase des images-cles de `fc` une fois. `known` est le catalogue
// des familles d arme ; `grenMax` NUL applique [DefaultGrenadeMax]. Erreur
// [ErrNoReadableFilmChunk] quand aucun chunk ne se lit.
func ScanEtatsDesImagesCles(fc *FilmContext, known map[uint32]bool, grenMax uint32) (EtatsDesImagesCles, error) {
	if grenMax == 0 {
		grenMax = DefaultGrenadeMax
	}
	c := nouveauCanalDeLEtatComplet(fc, known, grenMax)
	distribuerLesImagesClesSeules(fc, []Canal{c})
	c.out.StatsInventaire.Chunks = len(fc.ChunkNumbers())
	c.out.StatsInventaire.ChunksUnread = c.out.StatsInventaire.Chunks - c.lus
	if c.lus == 0 {
		return c.out, ErrNoReadableFilmChunk
	}
	c.out.Marche.BipedesAbsentsEncadres = bipedesAbsentsEncadres(c.bipedes)
	a := c.out.Admission
	fc.NoterReplis(ComptesDesReplis{FenetresArmesImageCle: a.FenetresArmes,
		FenetresInventaireImageCle: a.FenetresInventaire, FenetresMarqueDePortage: a.FenetresMarque})
	return c.out, nil
}

// canalDeLEtatCompletBipede lit, dans la phase des images-cles, l etat complet de chaque record
// bipede et la couverture de la marche d ancres.
type canalDeLEtatCompletBipede struct {
	fc      *FilmContext
	known   map[uint32]bool
	grenMax uint32
	// lisible : le registre du film declare l archetype bipede ; sans lui, aucun corps n est
	// parcouru et aucun record n est admis.
	lisible            bool
	arch               Archetype
	roles              map[int]roleDOccurrence
	dernierEmplacement int
	out                EtatsDesImagesCles
	// bipedes : les bipedes ancres, par image-cle, dans l ordre du film.
	bipedes []map[uint32]bool
	// lus : les chunks que la phase a pu lire.
	lus int
}

// nouveauCanalDeLEtatComplet prepare le canal ; un film sans archetype bipede lisible le laisse
// sans interet.
func nouveauCanalDeLEtatComplet(fc *FilmContext, known map[uint32]bool, grenMax uint32) *canalDeLEtatCompletBipede {
	c := &canalDeLEtatCompletBipede{fc: fc, known: known, grenMax: grenMax, dernierEmplacement: -1}
	arch, err := fc.bipedArchetype()
	if err != nil {
		return c
	}
	c.lisible, c.arch, c.roles = true, arch, rolesDuBipede(arch)
	for id := range weaponEmplacements(arch) {
		c.dernierEmplacement = max(c.dernierEmplacement, id)
	}
	return c
}

func (c *canalDeLEtatCompletBipede) Interets() []Interet {
	if !c.lisible {
		return nil
	}
	return interetsDeLEtatComplet()
}

func (c *canalDeLEtatCompletBipede) Clore(b BilanDeMarche) { c.lus = b.ChunksLus }

// ImageCle lit chaque record bipede du paquet et publie ce que la regle admet.
func (c *canalDeLEtatCompletBipede) ImageCle(p *lecture.Paquet, m *MarcheDistribuee) {
	c.out.Marche.Ajouter(m.Ancres.Stats)
	c.bipedes = append(c.bipedes, bipedesAncres(p.Records))
	c.out.StatsInventaire.Keyframes++
	c.out.Marques.KeyframeUS = append(c.out.Marques.KeyframeUS, p.TS)
	c.out.Marques.Records += len(p.Records)
	rendus := c.lireLePaquet(p)
	c.fenetresDerriereLaLecture(p, rendus)
	c.emettre(p, rendus)
}

// lireLePaquet rend, par record du paquet, ce que la grammaire publie d un record bipede ADMIS ; un
// record bipede non admis est marque pour la fenetre.
func (c *canalDeLEtatCompletBipede) lireLePaquet(p *lecture.Paquet) []renduDuRecord {
	rendus := make([]renduDuRecord, len(p.Records))
	ctx := c.fc.ContexteDeLecture()
	for i := range p.Records {
		r := &p.Records[i]
		if int(r.TI) != keyframeBipedTI {
			continue
		}
		c.out.Admission.Bipedes++
		c.out.Marques.BipedRecords++
		rendus[i].bipede = true
		if r.Desync == lecture.CorpsNonParcouru || !c.lisible {
			c.out.Admission.SansCorps++
			continue
		}
		l := lireLEtatComplet(p, r, c.arch, c.roles, ctx)
		c.out.Admission.Debordements += l.debordements
		if c.compterLeVerdict(admettre(r, &l, c.dernierEmplacement)) {
			rendus[i] = c.lu(r.Vie.Slot, &l)
		}
	}
	return rendus
}

// compterLeVerdict compte le verdict d un record et dit s il est admis.
func (c *canalDeLEtatCompletBipede) compterLeVerdict(v raisonDeRefus) bool {
	a := &c.out.Admission
	switch v {
	case refusNiFermeNiI22:
		a.RefusNiFermeNiI22++
	case refusT1:
		a.RefusT1++
	case refusT2:
		a.RefusT2++
	default:
		a.Admis++
		return true
	}
	return false
}

// lu rend ce que la grammaire publie d un record admis.
func (c *canalDeLEtatCompletBipede) lu(slot uint32, l *lectureDEtatComplet) renduDuRecord {
	armes := l.armesDe(slot)
	inv, horsDomaine := l.inventaireDe(slot)
	if horsDomaine {
		c.out.Admission.CapaciteHorsDomaine++
	}
	return renduDuRecord{bipede: true, admis: true, armes: &armes, inventaire: &inv, marque: l.porteLaMarque()}
}
