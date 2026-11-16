package tactical

// spawns_joueurs.go — LES REGLES DE JEU SUR LES SPAWNS D'UN SIDECAR : ce qu'est le spawn de DEPART d'un
// joueur, ce que sont TOUTES SES VIES, ce qu'est une REAPPARITION ADVERSE, et quelles routes suivent les
// vies retenues. Pur : le service traduit le sidecar vers ces types, la regle se teste ici.
//
// UNE SEULE DEFINITION. Le predicat « ce spawn est celui de la premiere vie » a existe en trois
// exemplaires (les grappes, la resolution d'un amas, le filtre) avant d'etre centralise ; les trois
// drapeaux qui composent « toutes les vies » sont lus ICI et nulle part ailleurs cote lecture
// (garde-rail : archlint/no_local_spawn_depart_test.go).

// SpawnLu est un spawn d'un sidecar tel que les regles le lisent.
type SpawnLu struct {
	Frame   int
	X, Y, Z float64
	// PremiereVie : le spawn de DEPART du joueur sur le match.
	PremiereVie bool
	// Reapparition : la vie commence apres une MORT lue dans le film.
	Reapparition bool
	// DebutDeManche : la vie precedente a ete close par une fin de manche ; ce n'est pas une
	// reapparition, mais c'est une apparition du joueur.
	DebutDeManche bool
	// Instantane : les autres joueurs vivants a la frame ; renseigne pour une reapparition.
	Instantane []Vivant
}

// JoueurLu est un joueur d'un sidecar : son xuid, son equipe (negative : inconnue) et ses spawns.
type JoueurLu struct {
	XUID   string
	Equipe int
	Spawns []SpawnLu
}

// spawnsDuJoueur parcourt les matchs (match_id -> joueurs) et rend, pour le joueur, les spawns que
// `retenu` accepte. C'est la seule boucle de lecture des spawns : les definitions ci-dessous ne
// different que par leur predicat.
func spawnsDuJoueur(matchs map[string][]JoueurLu, xuid string, retenu func(SpawnLu) bool) []PointSpawn {
	out := make([]PointSpawn, 0, len(matchs))
	for matchID, joueurs := range matchs {
		for _, j := range joueurs {
			if j.XUID != xuid {
				continue
			}
			for _, sp := range j.Spawns {
				if retenu(sp) {
					out = append(out, PointSpawn{MatchID: matchID, Frame: sp.Frame, X: sp.X, Y: sp.Y})
				}
			}
		}
	}
	return out
}

// SpawnsDeDepart rend les points de PREMIERE VIE du joueur, un par match.
func SpawnsDeDepart(matchs map[string][]JoueurLu, xuid string) []PointSpawn {
	return spawnsDuJoueur(matchs, xuid, func(sp SpawnLu) bool { return sp.PremiereVie })
}

// ApparitionsDuJoueur rend les points de TOUTES LES VIES du joueur : la premiere vie, chaque
// reapparition apres une mort et chaque debut de manche — toutes les fois ou le jeu l'a fait naitre, a
// plusieurs par match.
//
// UN DEBUT DE MANCHE N'EST PAS UNE REAPPARITION, mais c'est bien une apparition du joueur : il entre
// ici. Une piste qui n'est rien de tout cela (vie coupee par un trou de replication) n'y entre pas.
func ApparitionsDuJoueur(matchs map[string][]JoueurLu, xuid string) []PointSpawn {
	return spawnsDuJoueur(matchs, xuid, func(sp SpawnLu) bool {
		return sp.PremiereVie || sp.Reapparition || sp.DebutDeManche
	})
}

// SpawnsSelonVies choisit la definition du spawn : `toutes` rend les apparitions de toutes les vies,
// sinon la premiere vie seule.
func SpawnsSelonVies(matchs map[string][]JoueurLu, xuid string, toutes bool) []PointSpawn {
	if toutes {
		return ApparitionsDuJoueur(matchs, xuid)
	}
	return SpawnsDeDepart(matchs, xuid)
}

// RoutesAuxVies rend les rangs des routes dont la vie commence a l'un des instants donnes, dans
// l'ordre. `debuts` est l'instant de debut de chaque route.
//
// LA JOINTURE EST L'INSTANT : une route porte l'instant de debut de sa vie (Route.DebutFrame), le spawn
// de la meme vie porte son Frame, et les deux sont le premier point de la meme piste (NouvelIndexVies) ;
// un invariant de la cuisson le tient.
func RoutesAuxVies(debuts []int, instants map[int]bool) []int {
	out := make([]int, 0, len(debuts))
	for i, d := range debuts {
		if instants[d] {
			out = append(out, i)
		}
	}
	return out
}

// ObservationsAdversesDuMatch rend les reapparitions APRES MORT des joueurs de l'autre equipe que celle
// du joueur `xuid`, avec l'instantane qui les accompagne, et dit si l'equipe du joueur est connue du
// match (faux : le match ne peut pas servir la lecture, aucune observation).
//
// Une reapparition est celle que le sidecar a decidee (lecture du film) : un debut de manche n'en est
// pas une, une piste sans drapeau non plus. Un adversaire dont l'equipe n'est pas nommee n'est ni
// adverse ni equipier : il est ecarte.
func ObservationsAdversesDuMatch(matchID string, joueurs []JoueurLu, xuid string) (obs []ObservationAdverse, equipeConnue bool) {
	equipe := EquipeInconnue
	for _, j := range joueurs {
		if j.XUID == xuid {
			equipe = j.Equipe
			break
		}
	}
	if equipe < 0 {
		return nil, false
	}
	for _, j := range joueurs {
		if j.XUID == xuid || j.Equipe < 0 || j.Equipe == equipe {
			continue
		}
		for _, sp := range j.Spawns {
			if !sp.Reapparition {
				continue
			}
			obs = append(obs, ObservationAdverse{
				MatchID: matchID, X: sp.X, Y: sp.Y, Z: sp.Z, EquipeActive: equipe, Instantane: sp.Instantane,
			})
		}
	}
	return obs, true
}
