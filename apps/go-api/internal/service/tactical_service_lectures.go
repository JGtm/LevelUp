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
		spawnsSelonVies(sidecars, xuid, vies), zonesPures(zones))
	out := make([]domain.TacticalGrappe, 0, len(amas))
	for _, a := range amas {
		out = append(out, domain.TacticalGrappe{
			ID: a.ID, Nom: a.Nom.PickOptional(loc), X: a.X, Y: a.Y, Matchs: a.Matchs,
		})
	}
	return out
}

// zonesPures projette les zones du port vers le type du paquet d'algo, qui reste pur : nom et
// point de référence (grappes de spawn), forme et tranche (nom en jeu d'une cellule).
func zonesPures(zones []domain.ZoneNommee) []tactical.ZoneNommee {
	out := make([]tactical.ZoneNommee, 0, len(zones))
	for _, z := range zones {
		out = append(out, tactical.ZoneNommee{
			Nom: z.Nom, X: z.X, Y: z.Y,
			Polygone: z.Polygone, Parties: z.Parties, Trous: z.Trous,
			ZBas: z.ZBas, ZHaut: z.ZHaut, VolumeIndex: z.VolumeIndex,
		})
	}
	return out
}

// spawnsDuJoueur parcourt les sidecars et rend, pour le joueur, les spawns que `retenu`
// accepte. C'est la seule boucle de lecture des spawns : les deux definitions ci-dessous ne
// different que par leur predicat.
func spawnsDuJoueur(sidecars map[string]*domain.TacticalRasterSidecar, xuid string,
	retenu func(domain.TacticalRasterSpawn) bool) []tactical.PointSpawn {
	out := make([]tactical.PointSpawn, 0, len(sidecars))
	for matchID, sc := range sidecars {
		for _, j := range sc.Joueurs {
			if j.XUID != xuid {
				continue
			}
			for _, sp := range j.Spawns {
				if retenu(sp) {
					out = append(out, tactical.PointSpawn{MatchID: matchID, Frame: sp.Frame, X: sp.X, Y: sp.Y})
				}
			}
		}
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
	return spawnsDuJoueur(sidecars, xuid, func(sp domain.TacticalRasterSpawn) bool {
		return sp.PremiereVie
	})
}

// reapparitionsDuJoueur rend les points de TOUTES LES VIES du joueur : la premiere vie, chaque
// reapparition apres une mort et chaque debut de manche — toutes les fois ou le jeu l'a fait
// naitre, a plusieurs par match.
//
// UN DEBUT DE MANCHE N'EST PAS UNE REAPPARITION, mais c'est bien une apparition du joueur : il
// entre ici. Une piste qui n'est rien de tout cela (vie coupee par un trou de replication) n'y
// entre pas. Meme garde-rail que le depart : archlint/no_local_spawn_depart_test.go.
func reapparitionsDuJoueur(sidecars map[string]*domain.TacticalRasterSidecar,
	xuid string) []tactical.PointSpawn {
	return spawnsDuJoueur(sidecars, xuid, func(sp domain.TacticalRasterSpawn) bool {
		return sp.PremiereVie || sp.Reapparition || sp.DebutDeManche
	})
}

// spawnsSelonVies choisit la definition du spawn selon le parametre `vies` : `toutes` rend les
// apparitions de toutes les vies, tout autre valeur (`depart`) la premiere vie seule.
func spawnsSelonVies(sidecars map[string]*domain.TacticalRasterSidecar,
	xuid, vies string) []tactical.PointSpawn {
	if vies == domain.TacticalViesToutes {
		return reapparitionsDuJoueur(sidecars, xuid)
	}
	return spawnsDeDepart(sidecars, xuid)
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
				copie.Joueurs[i].Routes = routesDesVies(copie.Joueurs[i].Routes, vies[matchID])
			}
		}
		out[matchID] = &copie
	}
	return out
}

// routesDesVies garde les routes dont la vie commence a l'un des instants donnes.
func routesDesVies(routes []domain.TacticalRasterRoute, instants map[int]bool) []domain.TacticalRasterRoute {
	out := make([]domain.TacticalRasterRoute, 0, len(routes))
	for _, r := range routes {
		if instants[r.DebutFrame] {
			out = append(out, r)
		}
	}
	return out
}
