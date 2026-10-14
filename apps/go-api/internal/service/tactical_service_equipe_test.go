package service

// tactical_service_equipe_test.go — L'EQUIPE A LA MORT, LA RIPOSTE ET LE BILAN du detail d'une zone :
// equipe posee sur la cle exacte du contexte, statut par le rayon du match, tueur abattu (humain et
// bot), abattu en retour, bilan en « N sur M », et les sources qui manquent ou echouent.

import (
	"errors"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// corpusEquipe : le corpus du detail (corpusDetail), complete de ses coequipiers et de son journal.
//
//	m1 / 1 000   ma mort par Rival : deux coequipiers hors de vue -> aucun visible ;
//	             Ami abat Rival 800 ms plus tard -> TUEUR ABATTU (humain).
//	m1 / 2 000   mon frag ; je suis abattu 1 500 ms plus tard par un bot -> ABATTU EN RETOUR (bot).
//	m2 / 3 000   un coequipier visible a 10 m (rayon 24 m) -> accompagnee ; personne n'abat Rival.
//	m2 / 4 000   contexte a 4 500 ms, coequipier visible a 30 m -> visible hors rayon.
func corpusEquipe() *mockTacticalRepo {
	repo, _ := corpusDetail()
	repo.contextes = []domain.ContexteDeMort{
		{MatchID: "m1", VictimXUID: tsMoi, TimeMs: 1000, Visibles: 0, HorsDeVue: 2, Total: 2},
		{MatchID: "m2", VictimXUID: tsMoi, TimeMs: 3000, PlusProcheM: m(10), Visibles: 1, Total: 1},
		{MatchID: "m2", VictimXUID: tsMoi, TimeMs: 4500, PlusProcheM: m(30), Visibles: 1, Total: 1},
	}
	repo.mates = []domain.MateDeMort{
		{MatchID: "m1", VictimXUID: tsMoi, TimeMs: 1000, MateXUID: "900", Etat: domain.TacticalMateHorsDeVue},
		{MatchID: "m1", VictimXUID: tsMoi, TimeMs: 1000, MateXUID: tsAmi, Gamertag: "Ami", Etat: domain.TacticalMateHorsDeVue},
		{MatchID: "m2", VictimXUID: tsMoi, TimeMs: 3000, MateXUID: tsAmi, Gamertag: "Ami", Etat: domain.TacticalMateVisible,
			X: m(2.3), Y: m(12.3), DistanceM: m(10)},
		{MatchID: "m2", VictimXUID: tsMoi, TimeMs: 4500, MateXUID: tsAmi, Gamertag: "Ami", Etat: domain.TacticalMateVisible,
			X: m(32.4), Y: m(2.4), DistanceM: m(30)},
	}
	repo.journal = []domain.KillDuJournal{
		{MatchID: "m1", TimeMs: 1000, VictimXUID: tsMoi, VictimGamertag: "Moi", KillerXUID: tsAdv, KillerGamertag: "Rival"},
		{MatchID: "m1", TimeMs: 1800, VictimXUID: tsAdv, VictimGamertag: "Rival", KillerXUID: tsAmi, KillerGamertag: "Ami",
			KillerX: m(5), KillerY: m(6)},
		{MatchID: "m1", TimeMs: 2000, VictimXUID: tsAdv, VictimGamertag: "Cible", KillerXUID: tsMoi, KillerGamertag: "Moi"},
		{MatchID: "m1", TimeMs: 3500, VictimXUID: tsMoi, VictimGamertag: "Moi", KillerGamertag: "Bot Elite"},
		{MatchID: "m2", TimeMs: 3000, VictimXUID: tsMoi, VictimGamertag: "Moi", KillerXUID: tsAdv, KillerGamertag: "Rival"},
		{MatchID: "m2", TimeMs: 4000, VictimXUID: tsMoi, VictimGamertag: "Moi", KillerXUID: tsAdv2, KillerGamertag: "Rival2"},
	}
	return repo
}

func detailEquipe(t *testing.T, repo *mockTacticalRepo) domain.TacticalCelluleReponse {
	t.Helper()
	_, matchs := corpusDetail()
	return celluleLue(t, sourcesCompletes(matchs).service(repo, capsCompletes()),
		celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
}

func TestEquipe_StatutsEtCoequipiers(t *testing.T) {
	got := detailEquipe(t, corpusEquipe())
	c := got.Contributions
	if len(c) != 4 {
		t.Fatalf("contributions = %d, want 4", len(c))
	}
	if eq := c[0].Equipe; eq == nil || eq.Statut != domain.TacticalStatutAucunVisible || len(eq.Coequipiers) != 2 ||
		eq.RayonM == nil || *eq.RayonM != 18 {
		t.Errorf("m1 / 1000 : equipe = %+v", eq)
	}
	eq := c[2].Equipe // m2 / 3000
	if eq == nil || eq.Statut != domain.TacticalStatutAccompagnee || len(eq.Coequipiers) != 1 {
		t.Fatalf("m2 / 3000 : equipe = %+v", eq)
	}
	ami := eq.Coequipiers[0]
	if !ami.DansLeRayon || ami.Gamertag != "Ami" || ami.Point == nil || *ami.DistanceMortM != 10 {
		t.Errorf("coequipier = %+v", ami)
	}
	// la distance au tueur est celle du coequipier (2.3, 12.3) au point du tueur (9, 9).
	if ami.DistanceTueurM == nil || *ami.DistanceTueurM != 7.47 {
		t.Errorf("distance au tueur = %v, want 7.47", ami.DistanceTueurM)
	}
	if c[3].Equipe == nil || c[3].Equipe.Statut != domain.TacticalStatutVisibleHorsRayon || c[3].Equipe.Coequipiers[0].DansLeRayon {
		t.Errorf("m2 / 4000 : equipe = %+v (30 m hors d'un rayon de 24 m)", c[3].Equipe)
	}
	if c[1].Equipe != nil {
		t.Errorf("un frag n'a pas d'equipe : %+v", c[1].Equipe)
	}
}

func TestEquipe_Ripostes(t *testing.T) {
	c := detailEquipe(t, corpusEquipe()).Contributions
	r := c[0].Riposte
	if r == nil || r.Type != domain.TacticalRiposteTueurAbattu || r.DelaiMs != 800 || r.ParGamertag != "Ami" ||
		r.ParXUID != tsAmi || r.Point == nil || r.Point.X != 5 {
		t.Errorf("tueur abattu (m1 / 1000) = %+v", r)
	}
	r = c[1].Riposte
	if r == nil || r.Type != domain.TacticalRiposteAbattuEnRetour || r.DelaiMs != 1500 || r.ParGamertag != "Bot Elite" || r.ParXUID != "" {
		t.Errorf("abattu en retour par un bot (m1 / 2000) = %+v", r)
	}
	if c[2].Riposte != nil || c[3].Riposte != nil {
		t.Errorf("m2 : personne n'abat le tueur : %+v / %+v", c[2].Riposte, c[3].Riposte)
	}
}

func TestEquipe_BilanEnNSurM(t *testing.T) {
	b := detailEquipe(t, corpusEquipe()).Bilan
	if b == nil || b.Morts == nil || b.Frags == nil {
		t.Fatalf("bilan = %+v", b)
	}
	m := b.Morts
	if m.Total != 3 || m.Examinees != 3 || m.AucunVisible != 1 || m.Accompagnees != 1 || m.VisiblesHorsRayon != 1 || m.EquipeATerre != 0 {
		t.Errorf("statuts = %+v", m)
	}
	if m.TueurAbattu != (domain.TacticalNSurM{N: 1, M: 3}) {
		t.Errorf("tueur abattu = %+v, want 1 sur 3", m.TueurAbattu)
	}
	if len(m.Rose) != 8 {
		t.Fatalf("rose = %v", m.Rose)
	}
	somme := 0
	for _, n := range m.Rose {
		somme += n
	}
	// le tueur (9, 9) est au nord-est de la victime (2.x, 2.x) : dx ~ dy -> secteur 1 ou 0 selon le cheveu.
	if somme != 3 {
		t.Errorf("rose = %v, want trois tueurs ranges", m.Rose)
	}
	if m.MedianeTueurM == nil || m.MedianePlusProcheM == nil || *m.MedianePlusProcheM != 20 {
		t.Errorf("medianes = %v / %v (plus proche : 10 et 30 -> 20)", m.MedianeTueurM, m.MedianePlusProcheM)
	}
	if m.FenetreRiposteMs != 3000 || b.Frags.FenetreRiposteMs != 3000 {
		t.Errorf("fenetre de riposte = %d / %d, want 3000 (publiee pour le web)", m.FenetreRiposteMs, b.Frags.FenetreRiposteMs)
	}
	if b.Frags.AbattuEnRetour != (domain.TacticalNSurM{N: 1, M: 1}) {
		t.Errorf("abattu en retour = %+v, want 1 sur 1", b.Frags.AbattuEnRetour)
	}
}

func TestEquipe_UneLectureParSource(t *testing.T) {
	repo := corpusEquipe()
	detailEquipe(t, repo)
	if len(repo.vuMates) != 1 || strings.Join(repo.vuMates[0].Matchs.IDs(), ",") != "m1,m2" {
		t.Errorf("coequipiers : %+v, want 1 lecture bornee a [m1 m2]", repo.vuMates)
	}
	if len(repo.vuJournal) != 1 || strings.Join(repo.vuJournal[0].Matchs.IDs(), ",") != "m1,m2" {
		t.Errorf("journal : %+v, want 1 lecture bornee a [m1 m2]", repo.vuJournal)
	}
}

func TestEquipe_SourcesAbsentesOuEnEchec(t *testing.T) {
	// coequipiers en echec : plus d'equipe, la riposte reste.
	repo := corpusEquipe()
	repo.errMates = errors.New("boum")
	got := detailEquipe(t, repo)
	if len(got.Contributions) != 4 || got.Contributions[0].Equipe != nil || got.Contributions[0].Riposte == nil {
		t.Errorf("coequipiers en echec : %+v", got.Contributions[0])
	}
	// journal en echec : plus de riposte, ni N sur M des frags ; l'equipe reste.
	repo = corpusEquipe()
	repo.errJournal = errors.New("boum")
	got = detailEquipe(t, repo)
	if got.Contributions[0].Riposte != nil || got.Contributions[0].Equipe == nil {
		t.Errorf("journal en echec : %+v", got.Contributions[0])
	}
	if got.Bilan == nil || got.Bilan.Frags != nil || got.Bilan.Morts == nil || got.Bilan.Morts.TueurAbattu.M != 0 {
		t.Errorf("journal en echec : bilan = %+v", got.Bilan)
	}
}

func TestEquipe_LignesIncompletes_PasDEquipePartielle(t *testing.T) {
	repo := corpusEquipe()
	repo.mates = repo.mates[:1] // m1 / 1000 : une ligne sur deux ; m2 : aucune
	got := detailEquipe(t, repo)
	c := got.Contributions
	if c[0].Equipe != nil || c[2].Equipe != nil || c[3].Equipe != nil {
		t.Errorf("des lignes qui ne retombent pas sur teammates_total ne font pas une equipe : %+v", c)
	}
	if b := got.Bilan.Morts; b.Total != 3 || b.Examinees != 0 {
		t.Errorf("bilan = %+v : 3 morts, aucune examinee", b)
	}
}

func TestEquipe_EquipeATerre_EtMatchSansRayon(t *testing.T) {
	repo := corpusEquipe()
	repo.contextes[0] = domain.ContexteDeMort{MatchID: "m1", VictimXUID: tsMoi, TimeMs: 1000, EnAttente: 2, Total: 2}
	repo.mates[0].Etat, repo.mates[1].Etat = domain.TacticalMateEnAttente, domain.TacticalMateEnAttente
	if eq := detailEquipe(t, repo).Contributions[0].Equipe; eq == nil || eq.Statut != domain.TacticalStatutEquipeATerre {
		t.Errorf("tous en attente : equipe = %+v", eq)
	}
	_, matchs := corpusDetail()
	svc := sourcesCompletes(matchs).service(repo, capsCompletes()).WithRadarRange(map[string]int{"Oddball:Rounds": 24})
	got := celluleLue(t, svc, celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
	if eq := got.Contributions[0].Equipe; eq == nil || eq.Statut != "" || eq.RayonM != nil || len(eq.Coequipiers) != 2 {
		t.Errorf("m1 sans rayon : ni statut ni rayon, la liste reste : %+v", eq)
	}
}
