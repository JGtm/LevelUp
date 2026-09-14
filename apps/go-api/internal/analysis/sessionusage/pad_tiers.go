package sessionusage

// pad_tiers.go — LES PRISES DE SOCLE VENTILEES PAR NIVEAU D'ARME dans le bloc usage.
//
// # POURQUOI UNE VENTILATION A ELLE, ET PAS UN AJOUT A `PadFamilies`
//
// `PadFamilies` ventile les prises PAR ARME, et c'est une autre question : « avec quoi ai-je
// joué ». Le niveau répond à « qu'est-ce que j'ai contrôlé » — reprendre son fusil d'assaut
// posé sur un râtelier n'est pas rafler le lance-roquettes du socle central. Les deux
// coexistent sans se recouvrir.
//
// Surtout : les deux n'ont PAS LE MEME PERIMETRE MESURE. `PadFamilies` vient du résumé
// d'usage, produit sur tout artefact lu ; le niveau vient d'une AUTRE table
// (`match_pad_pickups_by_tier`), alimentée par une passe distincte qui a besoin de la carte du
// match. Les verser ensemble ferait compter un match non projeté comme un match sans prise —
// le faux zéro que tout ce dépôt refuse.
//
// # LES QUATRE DENOMINATEURS, ET POURQUOI ILS SONT QUATRE
//
//	MatchesMeasured           les matchs dont les socles ont été projetés. C'est LE
//	                          dénominateur de couverture : un match absent n'est pas un match
//	                          sans prise, c'est un match non mesuré ;
//	MatchesWithPads           ceux d'entre eux dont le film a publié AU MOINS UN socle. La
//	                          différence avec le précédent n'est pas une panne : un mode peut
//	                          n'allumer aucun emplacement (mesuré sur le parc — les treize
//	                          artefacts Super Fiesta sont à zéro socle) ;
//	MatchesTiersEstablished   ceux d'entre eux dont la carte est dans la référence des
//	                          emplacements. En dessous, tout tombe en « non classé », et
//	                          l'écran doit pouvoir dire que c'est un défaut de référence et
//	                          non un fait de jeu ;
//	MatchesRandomStarts       ceux d'entre eux dont le mode distribue des départs aléatoires :
//	                          le niveau « base » n'y est pas publié, et le nombre dit sur
//	                          combien de matchs il manque.
//
// # LA PART SE CALCULE SUR UN PERIMETRE UNIQUE (règle de computeMetric, usage.go)
//
// Numérateur ET dénominateur sur le sous-ensemble à camp connu. Compter le joueur sur tous les
// matchs et son camp sur les seuls matchs à camp connu ferait dépasser 100 % dès qu'un match
// du scope a un camp inconnu.
//
// PUR : zéro DB, zéro HTTP, zéro langue — les libellés d'arme sont résolus au service, comme
// pour `PadFamilies`.

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// PadTierRow — une ligne (match, joueur, niveau, arme) de
// `match_pad_pickups_by_tier_latest`.
type PadTierRow struct {
	MatchID string
	XUID    string
	// Tier : le vocabulaire ECRIT EN BASE (`persist.PadTier*`). Ce paquet ne le traduit pas :
	// il le transporte jusqu'au contrat, et c'est l'i18n du web qui le nomme.
	Tier string
	// WeaponFamily : la clé normalisée de la famille d'arme. VIDE sur une ligne
	// `aucune_prise` — le zéro mesuré d'un joueur qui n'a pris aucun socle.
	WeaponFamily string
	Pickups      int
	// PadsConfirmed / PadsTotal / RandomStarts : valeurs de MATCH, identiques sur toutes les
	// lignes du match (cf. persist.PadTiersBatch).
	PadsConfirmed int
	PadsTotal     int
	RandomStarts  bool
}

// PadTiersInput — tout ce que l'agrégat demande.
type PadTiersInput struct {
	Rows       []PadTierRow
	PlayerXUID string
	// PlayerTeam : matchID -> camp du joueur suivi (clé absente = camp inconnu).
	PlayerTeam map[string]int
	// TeamOf : matchID -> (xuid -> camp).
	TeamOf map[string]map[string]int
}

// tierSums : les accumulateurs d'UN niveau, avant projection au contrat.
type tierSums struct {
	player, team, lobby, playerTeamScope float64
	// parArme : famille d'arme -> [prises du joueur, prises du lobby]. C'est le détail servi
	// au survol ; il ne porte PAS de part — une part par arme sur un niveau ferait trois
	// dénominateurs de plus pour une lecture que personne n'a demandée.
	parArme map[string]*[2]float64
}

// ComputePadTiers agrège les niveaux du scope. nil si aucune ligne — le sous-bloc est alors
// OMIS, jamais servi à zéro : une session dont aucun match n'a été projeté n'a pas de niveaux
// à montrer, et un bloc vide se lirait « aucune prise ».
func ComputePadTiers(in PadTiersInput) *domain.SessionUsagePadTiersBlock {
	if len(in.Rows) == 0 {
		return nil
	}
	out := &domain.SessionUsagePadTiersBlock{}
	parNiveau := map[string]*tierSums{}
	mesures := map[string]bool{}
	avecSocles := map[string]bool{}
	niveauxEtablis := map[string]bool{}
	departsAleatoires := map[string]bool{}

	for i := range in.Rows {
		r := &in.Rows[i]
		mesures[r.MatchID] = true
		if r.PadsTotal > 0 {
			avecSocles[r.MatchID] = true
		}
		if r.PadsConfirmed > 0 {
			niveauxEtablis[r.MatchID] = true
		}
		if r.RandomStarts {
			departsAleatoires[r.MatchID] = true
		}
		// LE ZERO MESURE NE FAIT PAS DE LIGNE DE NIVEAU : il compte le match comme mesuré (ce
		// qu'il vient de faire ci-dessus) et rien d'autre. Lui donner un niveau inventerait une
		// catégorie « aucune prise » dans un classement qui parle de contrôle.
		if r.WeaponFamily == "" || r.Pickups == 0 {
			continue
		}
		accumulerNiveau(parNiveau, in, r)
	}

	out.MatchesMeasured = len(mesures)
	out.MatchesWithPads = len(avecSocles)
	out.MatchesTiersEstablished = len(niveauxEtablis)
	out.MatchesRandomStarts = len(departsAleatoires)
	out.Tiers = projeterNiveaux(parNiveau, out.MatchesMeasured)
	return out
}

// accumulerNiveau ajoute UNE ligne aux accumulateurs de son niveau.
//
// DEUX PRESENCES SE TESTENT, PAS UNE (même piège que `accumulerPerimetreEquipe`) : le camp du
// joueur suivi sur ce match, et le camp de CETTE ligne dans la table du match. Comparer
// `TeamOf[m][x]` sans tester la présence de la clé ferait passer un xuid ABSENT (valeur zéro
// d'une map) pour un joueur de l'équipe 0 — et zéro est un camp parfaitement légitime.
func accumulerNiveau(parNiveau map[string]*tierSums, in PadTiersInput, r *PadTierRow) {
	s := parNiveau[r.Tier]
	if s == nil {
		s = &tierSums{parArme: map[string]*[2]float64{}}
		parNiveau[r.Tier] = s
	}
	n := float64(r.Pickups)
	s.lobby += n
	arme := s.parArme[r.WeaponFamily]
	if arme == nil {
		arme = &[2]float64{}
		s.parArme[r.WeaponFamily] = arme
	}
	arme[1] += n
	if r.XUID == in.PlayerXUID {
		s.player += n
		arme[0] += n
	}
	camp, campSu := in.PlayerTeam[r.MatchID]
	if !campSu {
		return
	}
	if r.XUID == in.PlayerXUID {
		s.playerTeamScope += n
	}
	if campDeLaLigne, connu := in.TeamOf[r.MatchID][r.XUID]; connu && campDeLaLigne == camp {
		s.team += n
	}
}

// projeterNiveaux rend les niveaux du contrat, dans l'ORDRE ECRIT `domain.PadTierOrder` — pas
// par volume : un classement dont l'ordre change d'une session à l'autre ne se compare pas.
func projeterNiveaux(parNiveau map[string]*tierSums, matchsMesures int) []domain.SessionUsagePadTier {
	out := make([]domain.SessionUsagePadTier, 0, len(parNiveau))
	for _, tier := range domain.PadTierOrder {
		s := parNiveau[tier]
		if s == nil {
			continue
		}
		m := domain.SessionUsagePadTier{Tier: tier, Weapons: armesTriees(s.parArme)}
		m.PlayerTotal = s.player
		m.LobbyTotal = s.lobby
		if matchsMesures > 0 {
			pm := s.player / float64(matchsMesures)
			m.PlayerPerMatch = &pm
		}
		// PAS DE PART SUR UN DENOMINATEUR NUL, et pas de 0 % non plus : un camp qui n'a rien
		// pris à ce niveau ne donne pas « 0 % de participation », il ne donne PAS de part.
		if s.team > 0 {
			team := s.team
			m.TeamTotal = &team
			p := 100 * s.playerTeamScope / s.team
			m.PlayerShareOfTeamPct = &p
		}
		if s.lobby > 0 {
			p := 100 * s.player / s.lobby
			m.PlayerShareOfLobbyPct = &p
		}
		out = append(out, m)
	}
	return out
}

// armesTriees rend le détail par arme d'un niveau, du plus pris au moins pris (clé croissante
// à volume égal — deux relectures de la même session donnent le même survol).
func armesTriees(parArme map[string]*[2]float64) []domain.SessionUsagePadTierWeapon {
	out := make([]domain.SessionUsagePadTierWeapon, 0, len(parArme))
	for famille, v := range parArme {
		out = append(out, domain.SessionUsagePadTierWeapon{
			FamilyKey: famille, PlayerPickups: v[0], LobbyPickups: v[1],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LobbyPickups != out[j].LobbyPickups {
			return out[i].LobbyPickups > out[j].LobbyPickups
		}
		return out[i].FamilyKey < out[j].FamilyKey
	})
	return out
}
