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
	"sort"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
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
func grappesDeLUnivers(sidecars map[string]*domain.TacticalRasterSidecar,
	xuid string, zones []domain.ZoneNommee) []domain.TacticalGrappe {
	amas := tactical.GrappesDeSpawn(tactical.GrilleParDefaut(),
		spawnsDeDepart(sidecars, xuid), zonesPures(zones))
	out := make([]domain.TacticalGrappe, 0, len(amas))
	for _, a := range amas {
		out = append(out, domain.TacticalGrappe{
			ID: a.ID, NomFR: a.NomFR, NomEN: a.NomEN, X: a.X, Y: a.Y, Matchs: a.Matchs,
		})
	}
	return out
}

// zonesPures projette les zones du port vers le type du paquet d'algo, qui reste pur.
func zonesPures(zones []domain.ZoneNommee) []tactical.ZoneNommee {
	out := make([]tactical.ZoneNommee, 0, len(zones))
	for _, z := range zones {
		out = append(out, tactical.ZoneNommee{NomFR: z.NomFR, NomEN: z.NomEN, X: z.X, Y: z.Y})
	}
	return out
}

// spawnsDeDepart rend les points de PREMIERE VIE du joueur, un par match.
//
// C'EST LE SEUL ENDROIT QUI SAIT CE QU'EST UN SPAWN DE DEPART. Le predicat a existe en
// TROIS exemplaires (les grappes, la resolution d'un amas, le filtre) : a la troisieme
// copie, la regle du depot impose un helper et un garde-rail (CLAUDE.md n 6). Garde-rail :
// archlint/no_local_spawn_depart_test.go.
func spawnsDeDepart(sidecars map[string]*domain.TacticalRasterSidecar,
	xuid string) []tactical.PointSpawn {
	out := make([]tactical.PointSpawn, 0, len(sidecars))
	for matchID, sc := range sidecars {
		for _, j := range sc.Joueurs {
			if j.XUID != xuid {
				continue
			}
			for _, sp := range j.Spawns {
				if sp.PremiereVie {
					out = append(out, tactical.PointSpawn{MatchID: matchID, X: sp.X, Y: sp.Y})
				}
			}
		}
	}
	return out
}

// perimetreDuSpawn resout le filtre de grappe en LISTE DE MATCHS, et rend les grappes de
// l'univers ENTIER avec elle.
//
// ELLE EST APPELEE AVANT LE DISPATCH DE `Raster` (correction P1-1) : la restriction porte
// sur la liste blanche, donc elle vaut pour les lectures SQL (morts / kills / gagne) et pour
// le KPI d'echange autant que pour les lectures d'artefact. Appliquee dans la seule branche
// des sidecars, elle rendait 200 sur l'univers ENTIER sous un libelle de grappe.
//
// LES GRAPPES RENDUES SONT CELLES DE L'UNIVERS NON RESTREINT : c'est la liste que la page
// propose, et la reduire a la selection courante y enfermerait l'utilisateur.
func (s *TacticalService) perimetreDuSpawn(ctx context.Context, carte string,
	scope domain.TacticalScope) ([]domain.TacticalGrappe, []string, error) {
	if !s.caps.Has(games.CapFilmReplayArtifact) || s.rasters == nil {
		// LE FILTRE NE PEUT PAS ETRE HONORE : les grappes viennent des sidecars. Le
		// silencier servirait l'univers entier sous un libelle de grappe — exactement le
		// defaut que cette correction ferme.
		s.logger.WarnContext(ctx, "tactique: filtre de spawn demande sans lecteur d'artefact",
			"player", s.xuid, "map_id", carte, "spawn", scope.Spawn)
		return nil, nil, games.ErrCapabilityNotSupported
	}
	univers, err := s.repo.Univers(ctx, requeteDuScope(s.xuid, carte, scope))
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: univers du filtre de spawn en echec",
			"player", s.xuid, "map_id", carte, "err", err)
		return nil, nil, err
	}
	if len(univers.Matchs) == 0 {
		return nil, nil, domain.ErrTacticalCarteInconnue
	}
	sidecars, _ := s.chargerSidecars(ctx, univers, carte)
	grappes := grappesDeLUnivers(sidecars, s.xuid, s.zonesDeLaCarte(ctx, carte))

	amas, ok := amasParID(sidecars, s.xuid, scope.Spawn, grappes)
	if !ok {
		s.logger.InfoContext(ctx, "tactique: grappe de spawn inconnue sous ce filtre",
			"player", s.xuid, "map_id", carte, "spawn", scope.Spawn, "grappes", len(grappes))
		return nil, nil, domain.ErrTacticalSpawnInconnu
	}
	ids := matchsDeLaGrappe(sidecars, s.xuid, amas)
	if len(ids) == 0 {
		return nil, nil, domain.ErrTacticalSpawnInconnu
	}
	return grappes, ids, nil
}

// amasParID retrouve l'amas COMPLET (avec ses cellules) derriere un identifiant publie.
//
// Les grappes publiees ne portent pas leurs cellules — le contrat n'en a pas besoin — mais
// le filtre, lui, en depend : c'est l'emprise mesuree qui definit l'appartenance, jamais un
// rayon autour du barycentre. On recalcule donc les amas, ce qui est pur et borne.
func amasParID(sidecars map[string]*domain.TacticalRasterSidecar, xuid, spawnID string,
	grappes []domain.TacticalGrappe) (tactical.GrappeSpawn, bool) {
	connu := false
	for _, gr := range grappes {
		if gr.ID == spawnID {
			connu = true
			break
		}
	}
	if !connu {
		return tactical.GrappeSpawn{}, false
	}
	for _, a := range tactical.GrappesDeSpawn(tactical.GrilleParDefaut(),
		spawnsDeDepart(sidecars, xuid), nil) {
		if a.ID == spawnID {
			return a, true
		}
	}
	return tactical.GrappeSpawn{}, false
}

// matchsDeLaGrappe rend les match_id dont la PREMIERE vie du joueur tombe dans l'amas.
//
// L'APPARTENANCE SE LIT SUR LES CELLULES de l'amas, pas sur une distance au barycentre :
// c'est l'emprise mesuree qui definit la grappe, et un rayon invente en changerait la forme.
func matchsDeLaGrappe(sidecars map[string]*domain.TacticalRasterSidecar, xuid string,
	amas tactical.GrappeSpawn) []string {
	dedans := make(map[tactical.Cellule]bool, len(amas.Cellules))
	for _, c := range amas.Cellules {
		dedans[c] = true
	}
	g := tactical.GrilleParDefaut()
	vus := make(map[string]bool)
	out := make([]string, 0, len(sidecars))
	for _, sp := range spawnsDeDepart(sidecars, xuid) {
		if vus[sp.MatchID] {
			continue
		}
		if c, ok := g.Cellule(sp.X, sp.Y); ok && dedans[c] {
			vus[sp.MatchID] = true
			out = append(out, sp.MatchID)
		}
	}
	sort.Strings(out)
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
