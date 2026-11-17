package service

// tactical_service_lectures.go — LA LECTURE QUI S'AJOUTE A L'OCCUPATION : les ROUTES de
// sortie de spawn. Plus les GRAPPES de reapparition, qui ne sont pas une lecture mais un
// REPERE servi avec toutes.
//
// Fichier separe de tactical_service_rasters.go (qui porte le chargement des sidecars et
// l'occupation) : celui-la dit COMMENT on lit un sidecar, celui-ci dit CE QU'ON Y CHERCHE.
//
// ─── TOUT VIENT DES MEMES FICHIERS ─────────────────────────────────────────────
//
// Les deux lectures d'artefact (temps, routes) et les grappes se servent du MEME
// sidecar par match, charge UNE FOIS par requete. Elles partagent donc leur denominateur —
// `matchs_retenus`, les matchs dont le sidecar est present et exploitable — et l'ecart avec
// `matchs_filtres` dit la meme chose pour toutes : ce que la couverture de film ne montre
// pas.
//
import (
	"context"
	"sort"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/i18n"
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

// grappesDeLUnivers calcule les amas de reapparition depuis les spawns des sidecars charges.
//
// ELLES NE SONT PAS STOCKEES, et c'est la meme doctrine que le reste de l'onglet : une
// grappe depend de l'UNIVERS (quels matchs le filtre retient), donc la figer dans un fichier
// obligerait a l'invalider a chaque changement de filtre. Le calcul est pur et porte sur
// quelques centaines de points.
//
// QUELLES APPARITIONS COMPTENT DEPEND DU PARAMETRE `vies` (cf. domain/tactical_vies.go) :
// `depart` ne compte que la premiere vie (decision produit : les reapparitions suivantes
// dependent de l'endroit ou l'on vient de mourir, pas du placement d'ouverture), `toutes`
// compte aussi les reapparitions et les debuts de manche.
func grappesDeLUnivers(loc i18n.Locale, sidecars map[string]*domain.TacticalRasterSidecar,
	xuid string, zones []domain.ZoneNommee, vies string) []domain.TacticalGrappe {
	amas := tactical.GrappesDeSpawn(tactical.GrilleParDefaut(),
		spawnsSelonVies(sidecars, xuid, vies), tactical.ZonesDepuisLePort(zones))
	out := make([]domain.TacticalGrappe, 0, len(amas))
	for _, a := range amas {
		out = append(out, domain.TacticalGrappe{
			ID: a.ID, Nom: a.Nom.PickOptional(loc), X: a.X, Y: a.Y, Matchs: a.Matchs,
		})
	}
	return out
}

// joueurLu traduit un joueur du sidecar vers le type d'analysis qui porte la regle des spawns
// (analysis/tactical/spawns_joueurs.go). `avecInstantane` : l'instantane de chaque reapparition suit ;
// la lecture adverse le demande, celle des points de spawn non. C'est le SEUL endroit du service qui lit
// les drapeaux de vie d'un spawn (garde-rail : archlint/no_local_spawn_depart_test.go).
func joueurLu(j domain.TacticalRasterJoueur, avecInstantane bool) tactical.JoueurLu {
	spawns := make([]tactical.SpawnLu, 0, len(j.Spawns))
	for _, sp := range j.Spawns {
		lu := tactical.SpawnLu{
			Frame: sp.Frame, X: sp.X, Y: sp.Y, Z: sp.Z,
			PremiereVie: sp.PremiereVie, Reapparition: sp.Reapparition, DebutDeManche: sp.DebutDeManche,
		}
		if avecInstantane && sp.Reapparition {
			lu.Instantane = make([]tactical.Vivant, 0, len(sp.Instantane))
			for _, v := range sp.Instantane {
				lu.Instantane = append(lu.Instantane, tactical.Vivant(v))
			}
		}
		spawns = append(spawns, lu)
	}
	return tactical.JoueurLu{XUID: j.XUID, Equipe: j.Equipe, Spawns: spawns}
}

// spawnsSelonVies rend les points de spawn du joueur selon le parametre `vies` : `toutes` rend les
// apparitions de toutes les vies, tout autre valeur (`depart`) la premiere vie seule. La regle est celle
// de tactical.SpawnsSelonVies ; ici, la traduction des sidecars.
func spawnsSelonVies(sidecars map[string]*domain.TacticalRasterSidecar,
	xuid, vies string) []tactical.PointSpawn {
	matchs := make(map[string][]tactical.JoueurLu, len(sidecars))
	for matchID, sc := range sidecars {
		for _, j := range sc.Joueurs {
			if j.XUID == xuid {
				matchs[matchID] = append(matchs[matchID], joueurLu(j, false))
			}
		}
	}
	return tactical.SpawnsSelonVies(matchs, xuid, vies == domain.TacticalViesToutes)
}

// perimetreDuSpawn resout le filtre de grappe en LISTE DE MATCHS, et rend les grappes de
// l'univers ENTIER avec elle.
//
// ELLE EST APPELEE AVANT LE DISPATCH DE `Raster` (correction P1-1) : la restriction porte
// sur la liste blanche, donc elle vaut pour les lectures SQL (morts / kills / gagne / solde /
// isole) autant que pour les lectures d'artefact. Appliquee dans la seule branche des
// sidecars, elle rendait 200 sur l'univers ENTIER sous un libelle de grappe.
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
	univers, err := s.repo.Univers(ctx, s.requeteAvecRetention(s.cartesDe(ctx, carte), scope))
	if err != nil {
		s.logger.ErrorContext(ctx, "tactique: univers du filtre de spawn en echec",
			"player", s.xuid, "map_id", carte, "err", err)
		return perimetreSpawn{}, err
	}
	if len(univers.Matchs) == 0 {
		return perimetreSpawn{}, domain.ErrTacticalCarteInconnue
	}
	sidecars, _ := s.chargerSidecars(ctx, univers, carte)
	grappes := grappesDeLUnivers(i18n.FromContext(ctx), sidecars, s.xuid, s.zonesDeLaCarte(ctx, carte), scope.Vies)

	points := spawnsSelonVies(sidecars, s.xuid, scope.Vies)
	amas, ok := amasParID(points, scope.Spawn, grappes)
	if !ok {
		s.logger.InfoContext(ctx, "tactique: grappe de spawn inconnue sous ce filtre",
			"player", s.xuid, "map_id", carte, "spawn", scope.Spawn, "vies", scope.Vies, "grappes", len(grappes))
		return perimetreSpawn{}, domain.ErrTacticalSpawnInconnu
	}
	dedans := spawnsDeLAmas(points, amas)
	ids := matchsDesSpawns(dedans)
	if len(ids) == 0 {
		return perimetreSpawn{}, domain.ErrTacticalSpawnInconnu
	}
	lus := sidecars
	if scope.Vies == domain.TacticalViesToutes {
		// SOUS « TOUTES LES VIES », LE FILTRE EST UNE VIE, PAS SEULEMENT UN MATCH : un match peut
		// compter plusieurs apparitions, dont une seule part de la grappe. Les routes des autres
		// vies sortent des sidecars rendus a la lecture.
		lus = sidecarsAuxVies(sidecars, s.xuid, viesDesSpawns(dedans))
	}
	return perimetreSpawn{Grappes: grappes, MatchIDs: ids, Sidecars: lus}, nil
}

// perimetreSpawn porte ce que la resolution d'un filtre de grappe a produit : les grappes a
// publier, les matchs retenus, et LES SIDECARS DEJA LUS.
type perimetreSpawn struct {
	Grappes  []domain.TacticalGrappe
	MatchIDs []string
	Sidecars map[string]*domain.TacticalRasterSidecar
}

// amasParID retrouve l'amas COMPLET (avec ses cellules) derriere un identifiant publie, parmi
// les amas des points donnes.
//
// Les grappes publiees ne portent pas leurs cellules — le contrat n'en a pas besoin — mais
// le filtre, lui, en depend : c'est l'emprise mesuree qui definit l'appartenance, jamais un
// rayon autour du barycentre. On recalcule donc les amas, ce qui est pur et borne.
func amasParID(points []tactical.PointSpawn, spawnID string,
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
	for _, a := range tactical.GrappesDeSpawn(tactical.GrilleParDefaut(), points, nil) {
		if a.ID == spawnID {
			return a, true
		}
	}
	return tactical.GrappeSpawn{}, false
}

// spawnsDeLAmas rend ceux des points qui tombent dans l'amas.
//
// L'APPARTENANCE SE LIT SUR LES CELLULES de l'amas, pas sur une distance au barycentre :
// c'est l'emprise mesuree qui definit la grappe, et un rayon invente en changerait la forme.
func spawnsDeLAmas(points []tactical.PointSpawn, amas tactical.GrappeSpawn) []tactical.PointSpawn {
	dedans := make(map[tactical.Cellule]bool, len(amas.Cellules))
	for _, c := range amas.Cellules {
		dedans[c] = true
	}
	g := tactical.GrilleParDefaut()
	out := make([]tactical.PointSpawn, 0, len(points))
	for _, sp := range points {
		if c, ok := g.Cellule(sp.X, sp.Y); ok && dedans[c] {
			out = append(out, sp)
		}
	}
	return out
}

// matchsDesSpawns rend les match_id distincts des points, tries.
func matchsDesSpawns(points []tactical.PointSpawn) []string {
	vus := make(map[string]bool, len(points))
	out := make([]string, 0, len(points))
	for _, sp := range points {
		if !vus[sp.MatchID] {
			vus[sp.MatchID] = true
			out = append(out, sp.MatchID)
		}
	}
	sort.Strings(out)
	return out
}

// viesDesSpawns range les apparitions par match : match_id -> instants (frames) des vies.
func viesDesSpawns(points []tactical.PointSpawn) map[string]map[int]bool {
	out := make(map[string]map[int]bool, len(points))
	for _, sp := range points {
		if out[sp.MatchID] == nil {
			out[sp.MatchID] = make(map[int]bool)
		}
		out[sp.MatchID][sp.Frame] = true
	}
	return out
}

// sidecarsAuxVies rend des COPIES des sidecars où les routes du joueur sont réduites à celles
// des vies retenues (`vies` : match_id -> frames d'apparition).
//
// LA JOINTURE EST L'INSTANT : une route porte `DebutFrame`, le spawn de la meme vie porte
// `Frame`, et les deux sont le premier point de la meme piste (cf. analysis/tactical,
// `NouvelIndexVies` et `enrichirVies`). Un invariant de la cuisson le tient.
//
// SEULES LES ROUTES DU JOUEUR SONT REDUITES : « ma vie part de cette grappe » ne dit rien des
// vies des autres, dont le match est deja retenu par le filtre d'univers. Les originaux ne sont
// pas modifies (le lecteur peut les partager).
func sidecarsAuxVies(sidecars map[string]*domain.TacticalRasterSidecar, xuid string,
	vies map[string]map[int]bool) map[string]*domain.TacticalRasterSidecar {
	out := make(map[string]*domain.TacticalRasterSidecar, len(sidecars))
	for matchID, sc := range sidecars {
		copie := *sc
		copie.Joueurs = append([]domain.TacticalRasterJoueur(nil), sc.Joueurs...)
		for i := range copie.Joueurs {
			if copie.Joueurs[i].XUID == xuid {
				copie.Joueurs[i].Routes = routesAuxInstants(copie.Joueurs[i].Routes, vies[matchID])
			}
		}
		out[matchID] = &copie
	}
	return out
}

// routesAuxInstants garde les routes dont la vie commence a l'un des instants donnes (la regle est
// tactical.RoutesAuxVies).
func routesAuxInstants(routes []domain.TacticalRasterRoute, instants map[int]bool) []domain.TacticalRasterRoute {
	debuts := make([]int, 0, len(routes))
	for _, r := range routes {
		debuts = append(debuts, r.DebutFrame)
	}
	rangs := tactical.RoutesAuxVies(debuts, instants)
	out := make([]domain.TacticalRasterRoute, 0, len(rangs))
	for _, i := range rangs {
		out = append(out, routes[i])
	}
	return out
}

// reapparitionApresMort dit si le spawn est une REAPPARITION APRES UNE MORT dont la mort est connue :
// la matiere du modele de reapparition (les points que la prevision retient comme candidats). Un
// debut de manche, une premiere vie et une piste sans drapeau n'en sont pas. Meme garde-rail que la
// traduction ci-dessus : archlint/no_local_spawn_depart_test.go.
func reapparitionApresMort(sp domain.TacticalRasterSpawn) bool {
	return sp.Reapparition && sp.Mort != nil
}
