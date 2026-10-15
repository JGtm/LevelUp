package service

// tactical_service_equipe_test.go — L'EQUIPE A LA MORT ET LE BILAN du detail d'une zone : equipe posee
// sur la cle exacte du contexte, statut par le rayon du match, bilan en « N sur M », et la source qui
// manque ou echoue.

import (
	"errors"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// corpusEquipe : le corpus du detail (corpusDetail), complete de ses coequipiers et de son journal.
//
//	m1 / 1 000   ma mort par Rival : deux coequipiers hors de vue -> aucun visible ;
//	m1 / 2 000   mon frag.
//	m2 / 3 000   un coequipier visible a 10 m (rayon 24 m) -> accompagnee.
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

func TestEquipe_BilanEnNSurM(t *testing.T) {
	b := detailEquipe(t, corpusEquipe()).Bilan
	if b == nil || b.Morts == nil {
		t.Fatalf("bilan = %+v", b)
	}
	m := b.Morts
	if m.Total != 3 || m.Examinees != 3 || m.AucunVisible != 1 || m.Accompagnees != 1 || m.VisiblesHorsRayon != 1 || m.EquipeATerre != 0 {
		t.Errorf("statuts = %+v", m)
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
}

func TestEquipe_UneLectureParSource(t *testing.T) {
	repo := corpusEquipe()
	detailEquipe(t, repo)
	if len(repo.vuMates) != 1 || idsTries(repo.vuMates[0].Matchs.IDs()) != "m1,m2" {
		t.Errorf("coequipiers : %+v, want 1 lecture bornee a [m1 m2]", repo.vuMates)
	}
}

func TestEquipe_CoequipiersEnEchec_PlusDEquipe(t *testing.T) {
	repo := corpusEquipe()
	repo.errMates = errors.New("boum")
	got := detailEquipe(t, repo)
	if len(got.Contributions) != 4 || got.Contributions[0].Equipe != nil {
		t.Errorf("coequipiers en echec : %+v", got.Contributions[0])
	}
	if got.Bilan == nil || got.Bilan.Morts == nil || got.Bilan.Morts.Examinees != 0 {
		t.Errorf("coequipiers en echec : bilan = %+v, les morts se comptent sans etre examinees", got.Bilan)
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

func TestEquipe_DoublonDeLigne_DedoublonneParCoequipier(t *testing.T) {
	repo := corpusEquipe()
	// la cle de la mort a ete ecrite deux fois (doublon du journal) : chaque coequipier a deux lignes.
	repo.mates = append(repo.mates, repo.mates...)
	c := detailEquipe(t, repo).Contributions
	if c[0].Equipe == nil || len(c[0].Equipe.Coequipiers) != 2 || c[2].Equipe == nil || len(c[2].Equipe.Coequipiers) != 1 {
		t.Fatalf("des lignes en double ne doivent pas faire perdre l'equipe : %+v / %+v", c[0].Equipe, c[2].Equipe)
	}
}

func TestEquipe_LigneEnTrop_EquipeEcarteeEtJournalisee(t *testing.T) {
	repo := corpusEquipe()
	repo.mates = append(repo.mates, domain.MateDeMort{MatchID: "m2", VictimXUID: tsMoi, TimeMs: 3000, MateXUID: "999", Etat: domain.TacticalMateHorsDeVue})
	_, matchs := corpusDetail()
	src := sourcesCompletes(matchs)
	got := celluleLue(t, src.service(repo, capsCompletes()), celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
	if got.Contributions[2].Equipe != nil {
		t.Errorf("une ligne en trop (2 pour Total=1) : pas d'equipe partielle, %+v", got.Contributions[2].Equipe)
	}
	if !strings.Contains(src.trace.String(), "lignes de coequipiers incoherentes") {
		t.Errorf("l'ecart doit etre journalise :\n%s", src.trace.String())
	}
	// une mort SANS ligne est « a relire », pas une incoherence.
	repo = corpusEquipe()
	repo.mates = nil
	src = sourcesCompletes(matchs)
	celluleLue(t, src.service(repo, capsCompletes()), celluleDemande(repo, domain.TacticalQuestionGagne, domain.TacticalQuiMoi, 4, 4))
	if strings.Contains(src.trace.String(), "incoherentes") || !strings.Contains(src.trace.String(), "pas encore lus") {
		t.Errorf("aucune ligne : journal « pas encore lus », jamais « incoherentes » :\n%s", src.trace.String())
	}
}

// idsTries rend les identifiants triés : l'ordre d'une liste blanche n'est pas un contrat.
func idsTries(ids []string) string {
	c := append([]string(nil), ids...)
	sort.Strings(c)
	return strings.Join(c, ",")
}
