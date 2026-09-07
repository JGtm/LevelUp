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
	"math"
	"sort"
	"strings"

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

// chronologie indexe les segments de position d'un joueur pour une interrogation par
// instant.
type chronologie struct {
	// segments : les fenetres observables, dans l'ordre.
	segments []domain.TacticalRasterSegment
	// pasFrames : le pas de la chronologie, converti en frames pour ce match.
	pasFrames int
}

// positionA rend la position du joueur a une frame, si le sidecar en connait une.
//
// LA POSITION EST TENUE JUSQU'AU PAS SUIVANT : un echantillon a la frame f vaut pour
// [f, f + pas). Au-dela d'un segment, rien — c'est une absence, pas une position.
func (c chronologie) positionA(frame int) (x, y float64, ok bool) {
	for _, sg := range c.segments {
		n := len(sg.XY) / 2
		if n == 0 {
			continue
		}
		idx := (frame - sg.DebutFrame) / c.pasFrames
		if frame < sg.DebutFrame || idx >= n {
			continue
		}
		x, y = sg.XY[idx*2], sg.XY[idx*2+1]
		if math.IsNaN(x) || math.IsNaN(y) {
			// Trou DANS une fenetre observable (embarquement sans point de vehicule) : on
			// ne fabrique pas de position.
			return 0, 0, false
		}
		return x, y, true
	}
	return 0, 0, false
}

// dernierePositionAvant rend la derniere position CONNUE du joueur avant un instant.
//
// DECISION UTILISATEUR (2026-09-07) : un coequipier vivant dont le film a perdu la trace
// (vehicule non rattache) est quelque part, et sa DERNIERE position connue vaut mieux qu'un
// « inconnu » qui ne se mesure pas. On ne fabrique pas d'incertitude : on tient la mesure la
// plus recente.
func (c chronologie) dernierePositionAvant(frame int) (x, y float64, ok bool) {
	for _, sg := range c.segments {
		n := len(sg.XY) / 2
		for i := 0; i < n; i++ {
			f := sg.DebutFrame + i*c.pasFrames
			if f > frame {
				break
			}
			if px, py := sg.XY[i*2], sg.XY[i*2+1]; !math.IsNaN(px) && !math.IsNaN(py) {
				x, y, ok = px, py, true
			}
		}
	}
	return x, y, ok
}

// chronologiesDuMatch indexe la chronologie de chaque joueur nomme du sidecar.
func chronologiesDuMatch(sc *domain.TacticalRasterSidecar) map[string]chronologie {
	pasFrames := 1
	if sc.FrameIntervalMs > 0 {
		if p := tactical.PasChronologieMs / sc.FrameIntervalMs; p > 1 {
			pasFrames = p
		}
	}
	out := make(map[string]chronologie, len(sc.Joueurs))
	for _, j := range sc.Joueurs {
		out[j.XUID] = chronologie{segments: j.Chronologie, pasFrames: pasFrames}
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
		// LE NOM VIENT DE LA BASE, ET LA BASE PORTE CE QUE L'API A ENVOYE : des variantes
		// y arrivent avec un blanc de tete ou de queue. Une cle non nettoyee manque alors
		// la table, et le match sort silencieusement de l'univers mesurable — un defaut de
		// donnee deguise en « ce mode n'a pas de portee connue » (revue P2).
		metres, ok := s.radar[strings.TrimSpace(m.GameVariantName)]
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
//
// ELLE REND AUSSI LES SIDECARS QU'ELLE A LUS (revue P2) : la lecture d'artefact qui suit
// porte sur un SOUS-ENSEMBLE de cet univers, et les relire etait une seconde traversee du
// disque pour exactement les memes fichiers.
func (s *TacticalService) perimetreDuSpawn(ctx context.Context, carte string,
	scope domain.TacticalScope) (perimetreSpawn, error) {
	if !s.caps.Has(games.CapFilmReplayArtifact) || s.rasters == nil {
		// LE FILTRE NE PEUT PAS ETRE HONORE : les grappes viennent des sidecars. Le
		// silencier servirait l'univers entier sous un libelle de grappe — exactement le
		// defaut que cette correction ferme.
		s.logger.WarnContext(ctx, "tactique: filtre de spawn demande sans lecteur d'artefact",
			"player", s.xuid, "map_id", carte, "spawn", scope.Spawn)
		return perimetreSpawn{}, games.ErrCapabilityNotSupported
	}
	univers, err := s.repo.Univers(ctx, requeteDuScope(s.xuid, carte, scope))
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: univers du filtre de spawn en echec",
			"player", s.xuid, "map_id", carte, "err", err)
		return perimetreSpawn{}, err
	}
	if len(univers.Matchs) == 0 {
		return perimetreSpawn{}, domain.ErrTacticalCarteInconnue
	}
	sidecars, _ := s.chargerSidecars(ctx, univers, carte)
	grappes := grappesDeLUnivers(sidecars, s.xuid, s.zonesDeLaCarte(ctx, carte))

	amas, ok := amasParID(sidecars, s.xuid, scope.Spawn, grappes)
	if !ok {
		s.logger.InfoContext(ctx, "tactique: grappe de spawn inconnue sous ce filtre",
			"player", s.xuid, "map_id", carte, "spawn", scope.Spawn, "grappes", len(grappes))
		return perimetreSpawn{}, domain.ErrTacticalSpawnInconnu
	}
	ids := matchsDeLaGrappe(sidecars, s.xuid, amas)
	if len(ids) == 0 {
		return perimetreSpawn{}, domain.ErrTacticalSpawnInconnu
	}
	return perimetreSpawn{Grappes: grappes, MatchIDs: ids, Sidecars: sidecars}, nil
}

// perimetreSpawn porte ce que la resolution d'un filtre de grappe a produit : les grappes a
// publier, les matchs retenus, et LES SIDECARS DEJA LUS.
type perimetreSpawn struct {
	Grappes  []domain.TacticalGrappe
	MatchIDs []string
	Sidecars map[string]*domain.TacticalRasterSidecar
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

// mortsDuCamp construit les morts a examiner a partir du JOURNAL (la base), en resolvant
// la vitalite et la position de chaque coequipier dans la chronologie du sidecar.
//
// LES MORTS VIENNENT DU JOURNAL, PAS DU FILM (decision utilisateur du 2026-09-07) : c'est
// la meme source que « ou je meurs » et que l'echange, et la seule qui sache vraiment qui
// est mort quand. Le film, lui, ne sait dire que OU ETAIT CHACUN.
func (s *TacticalService) mortsDuCamp(lecture domain.TacticalKillEvents,
	sidecars map[string]*domain.TacticalRasterSidecar, dans predicatQui,
	rayons map[string]float64) []domain.MortAExaminer {
	out := make([]domain.MortAExaminer, 0, len(lecture.Events))
	for _, ev := range lecture.Events {
		if _, ok := rayons[ev.MatchID]; !ok {
			continue
		}
		sc := sidecars[ev.MatchID]
		if sc == nil || !dans(ev.MatchID, ev.VictimXUID) {
			continue
		}
		duMatch := lecture.Univers.Equipes[ev.MatchID]
		son, connu := duMatch[ev.VictimXUID]
		if !connu {
			continue
		}
		chronos := chronologiesDuMatch(sc)
		frame := frameDe(ev.TimeMs, sc.FrameIntervalMs)
		x, y, positionConnue := chronos[ev.VictimXUID].dernierePositionAvant(frame)
		if !positionConnue {
			// La mort a eu lieu, mais le film n'a jamais montre la victime : elle ne peut
			// ni se peindre ni servir de reference de distance.
			continue
		}
		m := domain.MortAExaminer{MatchID: ev.MatchID, X: x, Y: y}
		for autre, equipe := range duMatch {
			if autre == ev.VictimXUID || equipe != son {
				continue
			}
			m.Coequipiers = append(m.Coequipiers,
				etatDuCoequipier(chronos[autre], frame, x, y))
		}
		out = append(out, m)
	}
	return out
}

// etatDuCoequipier resout la presence et la distance d'UN coequipier a l'instant d'une mort.
//
// PROVISOIRE 2026-09-07 : « vivant » vaut ici « le film le montre quelque part a cet
// instant ». Le modele definitif — mort au journal, reapparition observee ou delai MESURE
// sur le match, depart lu dans la base — attend une decision produit de l'utilisateur sur
// ce que « vivant » veut dire quand le film se tait. Tant qu'elle n'est pas prise, un
// coequipier hors champ (vehicule non rattache) est traite comme absent : la mort part alors
// dans `EquipeATerre` plutot que d'etre comptee accompagnee sur une position devinee.
func etatDuCoequipier(chrono chronologie, frame int, mx, my float64) domain.EtatCoequipier {
	x, y, ok := chrono.positionA(frame)
	if !ok {
		return domain.EtatCoequipier{}
	}
	return domain.EtatCoequipier{
		Vivant: true, PositionConnue: true, DistanceM: math.Hypot(x-mx, y-my),
	}
}

// frameDe convertit un instant du match en frame de l'axe du rejeu.
func frameDe(tMs int64, intervalleMs int) int {
	if intervalleMs <= 0 {
		return 0
	}
	return int(tMs / int64(intervalleMs))
}

// mesurerIsolement assemble la lecture « ou je meurs isole ».
//
// L'UNIVERS MESURABLE EST « MESURE *ET* AYANT UN RAYON » (correction P0-2), et il est rendu
// pour que la somme des cellules soit normalisee sur LUI. `matchsSansRayon` se compte AU
// NIVEAU DU MATCH : compter au fil des morts laissait invisible un match dont la variante
// n'a pas de rayon mais ou le joueur n'est pas mort.
func (s *TacticalService) mesurerIsolement(lecture domain.TacticalKillEvents,
	sidecars map[string]*domain.TacticalRasterSidecar, univers domain.TacticalUnivers,
	dans predicatQui, mesures []string) (domain.BilanIsolement, []string) {
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
	bilan := coordination.Isolement(
		s.mortsDuCamp(lecture, sidecars, dans, rayons), rayons, len(avecRayon))
	bilan.MatchsSansRayon = sansRayon
	return bilan, avecRayon
}
