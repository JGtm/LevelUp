package service

// tactical_service_bilan.go — LE BILAN DE LA ZONE SELECTIONNEE : ce que ses morts et ses frags, pris
// ensemble, disent des coequipiers. Lu sur les contributions DEJA enrichies (equipe, riposte,
// positions) : aucune lecture nouvelle.
//
// COMPTES « N SUR M » ET MEDIANES, JAMAIS UN TAUX NU. Chaque compte voyage avec son denominateur ;
// une mediane est absente sans mesure. Les quatre statuts se partagent les morts dont l'equipe a ete
// lue (`Examinees`) ; l'ecart avec `Total` est le nombre de morts d'un match pas encore relu.
//
//	morts   les contributions de face mort (lectures morts, isole, solde) ;
//	frags   les contributions de face frag (lectures kills, solde), quand le journal a ete lu.

import (
	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
)

// bilanDeZone rend le bilan de la zone, ou nil quand elle ne porte ni mort ni frag.
func bilanDeZone(out []domain.TacticalContribution, lec lectureDeLEquipe) *domain.TacticalBilanZone {
	morts := bilanDesMorts(out, lec)
	frags := bilanDesFrags(out, lec)
	if morts == nil && frags == nil {
		return nil
	}
	return &domain.TacticalBilanZone{Morts: morts, Frags: frags}
}

// bilanDesMorts compte les statuts, range les tueurs dans la rose et mesure les distances.
func bilanDesMorts(out []domain.TacticalContribution, lec lectureDeLEquipe) *domain.TacticalBilanMorts {
	b := &domain.TacticalBilanMorts{Rose: make([]int, coordination.SecteursDeLaRose), FenetreRiposteMs: coordination.FenetreRiposteMs}
	var tueurM, procheM []float64
	for i := range out {
		c := &out[i]
		if c.Face != domain.TacticalFaceMort {
			continue
		}
		b.Total++
		compterLeStatut(b, c.Equipe)
		poserDansLaRose(b, c)
		if c.Point != nil && c.AutrePoint != nil {
			tueurM = append(tueurM, tactical.DistanceM(*c.Point, *c.AutrePoint))
		}
		if d := plusProcheVisible(c.Equipe); d != nil {
			procheM = append(procheM, *d)
		}
		if lec.tueurConnu[i] {
			b.TueurAbattu.M++
			if c.Riposte != nil {
				b.TueurAbattu.N++
			}
		}
	}
	if b.Total == 0 {
		return nil
	}
	b.MedianeTueurM, b.MedianePlusProcheM = tactical.MedianeM(tueurM), tactical.MedianeM(procheM)
	return b
}

// compterLeStatut range une mort dans l'un des quatre statuts ; sans equipe lue ou sans statut (match
// sans rayon), elle n'est pas examinee.
func compterLeStatut(b *domain.TacticalBilanMorts, eq *domain.TacticalEquipeALaMort) {
	if eq == nil {
		return
	}
	switch eq.Statut {
	case domain.TacticalStatutAccompagnee:
		b.Accompagnees++
	case domain.TacticalStatutVisibleHorsRayon:
		b.VisiblesHorsRayon++
	case domain.TacticalStatutAucunVisible:
		b.AucunVisible++
	case domain.TacticalStatutEquipeATerre:
		b.EquipeATerre++
	default:
		return
	}
	b.Examinees++
}

// poserDansLaRose range la direction victime -> tueur dans la rose, quand les deux positions sont
// mesurees et distinctes.
func poserDansLaRose(b *domain.TacticalBilanMorts, c *domain.TacticalContribution) {
	if c.Point == nil || c.AutrePoint == nil {
		return
	}
	if k, ok := coordination.RoseDepuis(*c.Point, *c.AutrePoint); ok {
		b.Rose[k]++
	}
}

// plusProcheVisible rend la plus petite distance au mort parmi les coequipiers visibles.
func plusProcheVisible(eq *domain.TacticalEquipeALaMort) *float64 {
	if eq == nil {
		return nil
	}
	var mini *float64
	for i := range eq.Coequipiers {
		if d := eq.Coequipiers[i].DistanceMortM; d != nil && (mini == nil || *d < *mini) {
			mini = d
		}
	}
	return mini
}

// bilanDesFrags compte les frags dont l'auteur a ete abattu dans la foulee ; nil sans frag, ou quand
// le journal n'a pas ete lu (le denominateur n'existe pas).
func bilanDesFrags(out []domain.TacticalContribution, lec lectureDeLEquipe) *domain.TacticalBilanFrags {
	if !lec.journalLu {
		return nil
	}
	b := domain.TacticalBilanFrags{FenetreRiposteMs: coordination.FenetreRiposteMs}
	for i := range out {
		if out[i].Face != domain.TacticalFaceFrag {
			continue
		}
		b.AbattuEnRetour.M++
		if out[i].Riposte != nil {
			b.AbattuEnRetour.N++
		}
	}
	if b.AbattuEnRetour.M == 0 {
		return nil
	}
	return &b
}
