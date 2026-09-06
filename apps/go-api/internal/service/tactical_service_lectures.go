package service

// tactical_service_lectures.go — LES DEUX LECTURES QUI S'AJOUTENT A L'OCCUPATION : les
// ROUTES de sortie de spawn et les MORTS ISOLEES. Plus les GRAPPES de reapparition, qui ne
// sont pas une lecture mais un REPERE servi avec toutes.
//
// Fichier separe de tactical_service_rasters.go (qui porte le chargement des sidecars et
// l'occupation) : celui-la dit COMMENT on lit un sidecar, celui-ci dit CE QU'ON Y CHERCHE.
//
// ─── TOUT VIENT DES MEMES FICHIERS ─────────────────────────────────────────────
//
// Les trois lectures d'artefact (temps, routes, isole) et les grappes se servent du MEME
// sidecar par match, charge UNE FOIS par requete. Elles partagent donc leur denominateur —
// `matchs_retenus`, les matchs dont le sidecar est present et exploitable — et l'ecart avec
// `matchs_filtres` dit la meme chose pour toutes : ce que la couverture de film ne montre
// pas.
//
// ─── L'ISOLEMENT SE DECIDE ICI, PARCE QUE C'EST ICI QUE LES EQUIPES EXISTENT ───
//
// Le sidecar porte, pour chaque mort, la distance a chaque autre joueur nomme VIVANT — sans
// equipe, que le film ne porte pas. Ce fichier joint `Univers.Equipes`, ne garde que les
// COEQUIPIERS, applique le rayon de la variante DU MATCH, et laisse `coordination.Isolement`
// trancher. Une variante sans rayon mesure : le match sort de la lecture et se compte.

import (
	"context"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
)

// comptesDesRoutes rend les cellules de ROUTE de la cible, une occurrence par passage.
//
// UNE CELLULE PAR VIE, PAS PAR ECHANTILLON : la route est un CHEMIN (les doublons
// consecutifs ont ete fusionnes a la cuisson), si bien qu'une cellule pese autant qu'on la
// traverse de fois, jamais autant qu'on y reste. C'est ce qui distingue « par ou je sors »
// de « ou je passe mon temps » — sans quoi les deux lectures rendraient la meme carte, la
// seconde etant simplement bornee a 15 s.
func comptesDesRoutes(sc *domain.TacticalRasterSidecar, matchID string,
	dans predicatQui) []tactical.CompteCellule {
	out := make([]tactical.CompteCellule, 0, 64)
	for _, j := range sc.Joueurs {
		if !dans(matchID, j.XUID) {
			continue
		}
		for _, r := range j.Routes {
			for _, c := range r.Cases {
				out = append(out, tactical.CompteCellule{
					Cellule:     tactical.Cellule{Col: c.Col, Lig: c.Lig},
					MatchID:     matchID,
					Occurrences: 1,
				})
			}
		}
	}
	return out
}

// mortsAExaminer projette les morts de la cible en morts examinables : chaque voisin est
// garde avec son STATUT si et seulement si c'est un COEQUIPIER du mort DANS CE MATCH.
//
// L'EQUIPE SE LIT PAR MATCH, jamais globalement : les numeros d'equipe se reattribuent a
// chaque partie, et une table globale melangerait deux compositions au premier joueur ayant
// change de camp.
func mortsAExaminer(sc *domain.TacticalRasterSidecar, m domain.TacticalMatch,
	equipes domain.EquipesParMatch, dans predicatQui) []domain.MortAExaminer {
	duMatch := equipes[m.MatchID]
	out := make([]domain.MortAExaminer, 0, 32)
	for _, j := range sc.Joueurs {
		if !dans(m.MatchID, j.XUID) {
			continue
		}
		son, connu := duMatch[j.XUID]
		if !connu {
			// Un mort dont on ignore le camp n'a pas de coequipier identifiable : le
			// compter donnerait « il etait seul » a partir d'une ignorance.
			continue
		}
		for _, mort := range j.Morts {
			out = append(out, domain.MortAExaminer{
				MatchID:          m.MatchID,
				X:                mort.X,
				Y:                mort.Y,
				PositionInconnue: mort.PositionInconnue,
				Coequipiers:      statutsDesCoequipiers(mort.Voisins, duMatch, son),
			})
		}
	}
	return out
}

// statutsDesCoequipiers ne garde que les voisins du MEME camp, avec leur statut.
//
// UN ADVERSAIRE PROCHE N'ACCOMPAGNE PERSONNE : la question porte sur le soutien. Un voisin
// dont l'equipe est inconnue est ecarte lui aussi — lui en preter une serait une invention.
func statutsDesCoequipiers(voisins []domain.TacticalRasterVoisin,
	duMatch map[string]int, monEquipe int) []domain.StatutCoequipier {
	out := make([]domain.StatutCoequipier, 0, len(voisins))
	for _, v := range voisins {
		son, connu := duMatch[v.XUID]
		if !connu || son != monEquipe {
			continue
		}
		out = append(out, domain.StatutCoequipier{Statut: v.Statut, DistanceM: v.DistanceM})
	}
	return out
}

// rayonsParMatch resout la portee du radar de chaque match MESURE, par sa variante.
//
// UN MATCH DONT LA VARIANTE N'EST PAS DANS LA TABLE N'ENTRE PAS DANS LA TABLE DE SORTIE, et
// il sort donc de l'UNIVERS de la lecture — pas seulement de ses numerateurs (correction
// P0-2). Le laisser au denominateur divisait la mesure par des matchs qu'on avait refuse de
// lire : deux matchs dont un Husky Raid rendaient 0,5 mort isolee par match au lieu de 1.
// C'est la seule facon de distinguer « il est mort accompagne » de « on ne sait pas a
// quelle distance on se voit sur ce mode ».
func (s *TacticalService) rayonsParMatch(matchs []domain.TacticalMatch) map[string]float64 {
	out := make(map[string]float64, len(matchs))
	for _, m := range matchs {
		metres, ok := s.radar[m.GameVariantName]
		if !ok || metres <= 0 {
			continue
		}
		out[m.MatchID] = float64(metres)
	}
	return out
}

// comptesDesMortsIsolees rend une occurrence par mort isolee, dans sa cellule.
func comptesDesMortsIsolees(g tactical.Grille, isolees []domain.MortAExaminer) []tactical.CompteCellule {
	out := make([]tactical.CompteCellule, 0, len(isolees))
	for _, m := range isolees {
		c, ok := g.Cellule(m.X, m.Y)
		if !ok {
			continue
		}
		out = append(out, tactical.CompteCellule{Cellule: c, MatchID: m.MatchID, Occurrences: 1})
	}
	return out
}

// grappesDeLUnivers calcule les amas de reapparition depuis les spawns de DEPART des
// sidecars charges.
//
// ELLES NE SONT PAS STOCKEES, et c'est la meme doctrine que le reste de l'onglet : une
// grappe depend de l'UNIVERS (quels matchs le filtre retient), donc la figer dans un fichier
// obligerait a l'invalider a chaque changement de filtre. Le calcul est pur et porte sur
// quelques centaines de points.
//
// SEULE LA PREMIERE VIE COMPTE (decision produit) : les reapparitions suivantes dependent de
// l'endroit ou l'on vient de mourir, pas du placement d'ouverture.
func grappesDeLUnivers(ctx context.Context, sidecars map[string]*domain.TacticalRasterSidecar,
	xuid string, zones []domain.ZoneNommee) []domain.TacticalGrappe {
	_ = ctx
	points := make([]tactical.PointSpawn, 0, len(sidecars))
	for matchID, sc := range sidecars {
		for _, j := range sc.Joueurs {
			if j.XUID != xuid {
				continue
			}
			for _, sp := range j.Spawns {
				if !sp.PremiereVie {
					continue
				}
				points = append(points, tactical.PointSpawn{MatchID: matchID, X: sp.X, Y: sp.Y})
			}
		}
	}
	amas := tactical.GrappesDeSpawn(tactical.GrilleParDefaut(), points, zonesPures(zones))
	out := make([]domain.TacticalGrappe, 0, len(amas))
	for _, a := range amas {
		out = append(out, domain.TacticalGrappe{ID: a.ID, Nom: a.Nom, X: a.X, Y: a.Y, Matchs: a.Matchs})
	}
	return out
}

// zonesPures projette les zones du port vers le type du paquet d'algo, qui reste pur.
func zonesPures(zones []domain.ZoneNommee) []tactical.ZoneNommee {
	out := make([]tactical.ZoneNommee, 0, len(zones))
	for _, z := range zones {
		out = append(out, tactical.ZoneNommee{Nom: z.Nom, X: z.X, Y: z.Y})
	}
	return out
}

// matchsDeLaGrappe rend les match_id dont la PREMIERE vie du joueur tombe dans l'amas
// demande.
//
// LE FILTRE PORTE SUR L'UNIVERS, PAS SUR LES POINTS : restreindre les seuls points peints
// aurait garde au denominateur des matchs partis d'un autre spawn, et la lecture aurait
// repondu « je passe peu de temps ici » alors qu'on n'y a simplement pas commence.
//
// L'APPARTENANCE SE LIT SUR LES CELLULES de l'amas, pas sur une distance au barycentre :
// c'est l'emprise mesuree qui definit la grappe, et un rayon invente en changerait la forme.
func matchsDeLaGrappe(sidecars map[string]*domain.TacticalRasterSidecar, xuid string,
	amas tactical.GrappeSpawn) map[string]bool {
	dedans := make(map[tactical.Cellule]bool, len(amas.Cellules))
	for _, c := range amas.Cellules {
		dedans[c] = true
	}
	g := tactical.GrilleParDefaut()
	out := make(map[string]bool)
	for matchID, sc := range sidecars {
		for _, j := range sc.Joueurs {
			if j.XUID != xuid {
				continue
			}
			for _, sp := range j.Spawns {
				if !sp.PremiereVie {
					continue
				}
				if c, ok := g.Cellule(sp.X, sp.Y); ok && dedans[c] {
					out[matchID] = true
				}
			}
		}
	}
	return out
}

// mesurerIsolement assemble la lecture « ou je meurs isole ».
//
// L'UNIVERS MESURABLE EST « MESURE *ET* AYANT UN RAYON » (correction P0-2), et il est rendu
// pour que la somme des cellules soit normalisee sur LUI. `matchsSansRayon` se compte AU
// NIVEAU DU MATCH : compter au fil des morts laissait invisible un match dont la variante
// n'a pas de rayon mais ou le joueur n'est pas mort.
func (s *TacticalService) mesurerIsolement(sidecars map[string]*domain.TacticalRasterSidecar,
	univers domain.TacticalUnivers, dans predicatQui, mesures []string) (domain.BilanIsolement, []string) {
	rayons := s.rayonsParMatch(univers.Matchs)
	avecRayon := make([]string, 0, len(mesures))
	sansRayon := 0
	for _, id := range mesures {
		if _, ok := rayons[id]; ok {
			avecRayon = append(avecRayon, id)
			continue
		}
		sansRayon++
	}
	morts := make([]domain.MortAExaminer, 0, 64)
	for _, m := range univers.Matchs {
		sc := sidecars[m.MatchID]
		if sc == nil {
			continue
		}
		morts = append(morts, mortsAExaminer(sc, m, univers.Equipes, dans)...)
	}
	bilan := coordination.Isolement(morts, rayons, len(avecRayon))
	bilan.MatchsSansRayon = sansRayon
	return bilan, avecRayon
}
