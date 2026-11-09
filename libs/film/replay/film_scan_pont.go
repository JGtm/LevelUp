package replay

// film_scan_pont.go — LA PREMIERE LECTURE DE L ETAGE DE BALAYAGE : LE PONT D IDENTITE (lot J4.3,
// 2026-09-26). Sortie de `balayerPositions` (film_scan.go) au seuil de longueur du depot ; meme
// etage, meme ordre d etapes observees.
//
// `balayerPont` ET `poserLeFilDesMorts` Y REJOIGNENT LEURS DEUX APPELES le 2026-10-22 (lot
// detonations, correction CI) : la lecture de la detonation de projectile
// (`grammar.ScanProjectileDetonations`) avait fait depasser `film_scan.go` au-dela du plafond de
// 500 lignes du depot (CLAUDE.md regle 5). DEPLACEMENT PUR, aucune ligne de logique ne change ;
// `balayerPont` appelait deja [lireSiegesEtOccupants], defini juste au-dessus dans ce meme
// fichier — le deplacement les rapproche plutot que d ouvrir un troisieme fichier.

import (
	"context"
	"log/slog"

	"github.com/JGtm/LevelUp/libs/film/internal/grammar"
	"github.com/JGtm/LevelUp/libs/film/internal/source"
	"github.com/JGtm/LevelUp/libs/film/types"
)

// lirePontDIdentite lit, d'un seul appel a l'etage unique, les six lectures du pont d'identite, et
// pose les TELEPORTATIONS (premiere etape observee). Les autres lectures sont posees par les
// phases, a leur place d'avant.
func (s *filmScan) lirePontDIdentite() {
	// TÉLÉPORTATIONS DU TRANSLOCATEUR : lues AVANT les positions, parce qu'elles servent
	// deux fois — le calque `translocations` du document, et l'EXEMPTION du filtre de
	// vitesse (décision D2) : une arrivée de téléportation part à 193-1540 m/s, le filtre à
	// 100 m/s la rejetait à tort (R3 : 51/51 rejets mesurés, tous à ±200 ms d'un événement
	// 117 du même slot). Sur un film sans tête 117, la liste est vide et le filtre est
	// bit à bit identique à l'actuel — invariance prouvée par test.
	//
	// L'ENTRÉE DE CATALOGUE Y DESCEND parce que la CHARGE de l'événement porte les deux
	// positions du va-et-vient, quantifiées aux bornes de la carte (R6 §1, validé 18/18) :
	// sans elle le scanner rendrait des quanta invérifiables, donc rien. Elle est garantie
	// non nulle ici (refus en tête de BuildFromFilm).
	//
	// L'ETAGE UNIQUE DU PONT D'IDENTITE (lot J4.3, 2026-09-26) : teleportations, positions avec
	// leurs exemptions, creations de bipede, fil des morts, table d'index et origine d'horloge sont
	// lus ICI, d'un seul appel — le MEME que le collecteur killsource. Chaque lecture est POSEE ou
	// elle l'etait (etapes observees dans le meme ordre) ; seules les POLITIQUES de la cuisson
	// restent ici : le roster de l'index (fil des morts + roster de l'appelant), la capture des
	// directions (dans `s.scan`), l'injectivite et la fatalite des erreurs.
	s.pont = grammar.ScanPontDIdentite(s.fc, grammar.OptionsDuPont{
		Balayage: s.scan, Carte: s.opt.MapQuant,
		RosterDesMorts: func(d []types.Death) []uint64 { return rosterOf(d, s.opt.RosterXUIDs) },
	})
	s.in.Translocations = s.pont.Translocations
	if len(s.in.Translocations) > 0 {
		slog.InfoContext(s.ctx, "translocateur : teleportations lues", "evenements", len(s.in.Translocations))
	}
}

// siegesEtOccupants porte les deux lectures du film qui nomment un corps SANS le fil des morts :
// la table des sieges que `chunk_00` ecrit (le lien direct `index <-> xuid <-> gamertag`) et les
// occupants par entite ti=9, lus dans la meme passe que l'equipe de chaque index.
type siegesEtOccupants struct {
	table   grammar.FilmPlayerTable
	equipes map[int]int
	rapport grammar.TeamScanReport
	entites grammar.PlayerEntityScan
}

// lireSiegesEtOccupants est LA lecture de ces deux entrees du registre d'identite, et la seule :
// la cuisson ([filmScan.balayerPont]) et le document interne du sync ([PortagesAuSync]) l'appellent
// tous deux, pour que les deux registres d'un meme film nomment les memes vies. Le registre emploie
// la table du film AVANT la table d'index des chunks (identity_registry_film_table.go) et les
// entites pour le corps d'un index que plusieurs occupants se relaient
// (identity_registry_entites.go) : une lecture omise d'un cote laisse ce cote sans nom la ou
// l'autre nomme (garde-rail : `TestLeSyncEtLaCuissonLisentLesMemesSiegesEtOccupants`).
//
// Aucun refus n'est fatal : la table refusee est consignee et publiee
// ([consignerLaTableDuFilm]), une lecture d'equipes refusee porte sa cause dans son rapport.
func lireSiegesEtOccupants(ctx context.Context, film *source.Film, fc *grammar.FilmContext, matchID string) siegesEtOccupants {
	table, err := grammar.ScanFilmPlayerTable(film)
	so := siegesEtOccupants{table: consignerLaTableDuFilm(ctx, table, err, matchID)}
	so.equipes, so.rapport, so.entites = grammar.ScanPlayerTeams(fc)
	return so
}

// balayerPont lit les lancers, les projectiles, et LE PONT D'IDENTITE : le fil des morts, la
// table d'index de joueur et l'origine d'horloge.
func (s *filmScan) balayerPont() {
	// Lancers de grenade : décodés des paquets delta du MÊME film, sur la MÊME horloge.
	// Absence non fatale, comme les tirs et les armes portées.
	grenades, err := grammar.ScanGrenadeThrows(s.fc)
	if err != nil {
		slog.WarnContext(s.ctx, "paquets delta illisibles — rejeu sans lancers de grenade", "err", err, "match_id", s.matchID)
		grenades = nil
	}
	s.in.Grenades = grenades
	s.opt.observe(s.ctx, "grenades", s.in.Grenades)
	// Trajectoires de projectile : memes chunks, meme horloge. Absence non fatale.
	proj, err := grammar.ScanProjectiles(s.fc, &s.world)
	if err != nil {
		slog.WarnContext(s.ctx, "projectiles illisibles — rejeu sans trajectoires", "err", err, "match_id", s.matchID)
		proj = nil
	}
	s.in.Projectiles = proj
	s.opt.observe(s.ctx, "projectiles", s.in.Projectiles)
	// Detonations/impacts de projectile (genres 5/6 de la vue A) : lecture seule, sans chunk
	// supplementaire — jamais d'erreur, juste moins de couverture sur un film qui ne les ecrit pas.
	s.in.Detonations, _ = grammar.ScanProjectileDetonations(s.fc)
	s.opt.observe(s.ctx, "detonations", s.in.Detonations)
	// Le fil des morts NOMME les vies par le pont par morts, cale l'horloge des morts et ouvre la
	// lecture de la table d index (cf. [filmScan.poserLeFilDesMorts]). Il a ete LU par l etage du pont.
	s.poserLeFilDesMorts(s.pont.Morts, s.pont.ErrMorts)
	deaths := s.in.Deaths
	s.opt.observe(s.ctx, "deaths", s.in.Deaths)
	// LA TABLE DES JOUEURS QUE LE FILM ÉCRIT (lot 1.6) : `chunk_00` porte les 32 sièges du match avec
	// leur XUID et leur gamertag, le lien DIRECT que le registre emploie AVANT la table des chunks de
	// réplication (cf. film_player_table.go). Un refus est NOMMÉ, journalisé et publié. Elle se lit
	// avec les occupants par la lecture que le document interne du sync appelle aussi.
	so := lireSiegesEtOccupants(s.ctx, s.film, s.fc, s.matchID)
	s.in.FilmTable = so.table
	s.opt.observe(s.ctx, "filmTable", s.in.FilmTable)
	// L'EQUIPE DE CHAQUE JOUEUR (lot 1.7) : le composant i0 de ti=9 de la trame d'etat, a une
	// position DERIVEE de la grammaire. C'est la SEULE source d'equipe du document (V4) ; la
	// base ne fait que controler. Un refus est NOMME et publie (`coverage.teams.refusal`).
	// LES ENTITES SORTENT DE LA MEME PASSE (lot M2.1) : un occupant par entite, sa presence au pas
	// des images-cles et son equipe. L'etape observee reste la TABLE par index, a l'octet : elle
	// est le controle, et son empreinte inchangee prouve que la passe unique ne l'a pas touchee.
	s.in.PlayerTeams, s.in.TeamScan, s.in.PlayerEntities = so.equipes, so.rapport, so.entites
	s.opt.observe(s.ctx, "playerTeams", s.in.PlayerTeams)
	// L'index de joueur SE LIT dans le film (cf. player_index.go) : le roster vient du fil des
	// morts, et les 5 bits qui précèdent chaque xuid donnent son index. Sans cette table, un tir
	// ou un lancer reste publie des que le registre d'identite nomme son slot par ses AUTRES
	// lectures — la table des sieges du film (`FilmTable`) en tete (message corrige a la revue
	// adverse M5, constat R3, 2026-09-24 : il disait « aucun tir ni lancer n'est publie »).
	if len(deaths) > 0 {
		// LUE PAR L ETAGE DU PONT sur le roster de cette cuisson (`rosterOf`) : un fil non vide
		// l a toujours fait lire.
		idx, err := s.pont.Index, s.pont.ErrIndex
		if err != nil {
			slog.WarnContext(s.ctx, "index de joueur illisible — les tireurs ne seront nommes que par les autres "+
				"lectures du registre (table des sieges du film)", "err", err, "match_id", s.matchID)
		}
		table, collisions := injectiveOrEmpty(idx)
		if collisions > 0 {
			slog.WarnContext(s.ctx, "index de joueur NON INJECTIF — table ecartee",
				"collisions", collisions, "match_id", s.matchID)
		}
		s.in.PlayerIndices = table
	}
	s.opt.observe(s.ctx, "playerIndices", s.in.PlayerIndices)
	// L'origine d'horloge du film : deux en-têtes de paquet, aucune estimation (cf.
	// origin.go). Son absence n'est pas fatale — le document sort sans origine, et le
	// client retombe sur l'appariement.
	clockUS, err := s.pont.OrigineUS, s.pont.ErrOrigine
	if err != nil {
		slog.WarnContext(s.ctx, "origine d'horloge illisible — rejeu sans origine publiee", "err", err, "match_id", s.matchID)
		clockUS = 0
	}
	s.in.FilmClockOriginUS = clockUS
	s.opt.observe(s.ctx, "clockOrigin", s.in.FilmClockOriginUS)
}

// poserLeFilDesMorts pose le fil des morts LU par l etage du pont (lot J4.3), son VERDICT
// (`coverage.bridge.deathsFeed`, lot M5.2 des retours rejeu) et le fil lui-meme dans les entrees.
// Un fil VIDE est une mesure, un fil ILLISIBLE une panne, et le document les distingue au lieu de
// les laisser aux seuls journaux.
// Le verdict et sa cause sont des ENTREES depuis le lot M8 (2026-09-24) : ils voyagent dans les
// faits persistes (cf. fil_des_morts_verdict.go).
//
// CE QUE LE FIL ILLISIBLE NE COUPE PLUS (message corrige le 2026-09-23) : les tirs et les
// lancers. Ils sont nommes par la table des sieges que le film ecrit (`chunk_00`, lot 1.6) —
// mesure : 2 838 tirs publies sur `ab526724` quand son fil etait illisible.
//
// UNE METHODE A PART (revue adverse M5, constat R1, 2026-09-24) pour que le trajet « octets ->
// verdict -> document » se teste sur la bobine du depot, que le balayage des positions refuse.
func (s *filmScan) poserLeFilDesMorts(deaths []types.Death, err error) {
	s.in.DeathsFeed = verdictDeLaLecture(deaths, err)
	switch s.in.DeathsFeed.Verdict {
	case DeathsFeedUnreadable:
		slog.WarnContext(s.ctx, "fil des morts illisible — ni calage d horloge des morts, ni table d index, ni "+
			"pont par morts (coverage.bridge.deathsFeed = unreadable)", "err", err, "match_id", s.matchID)
		deaths = nil
	case DeathsFeedEmpty:
		slog.InfoContext(s.ctx, "fil des morts lu et VIDE — aucune mort a nommer (coverage.bridge.deathsFeed = empty)",
			"err", err, "match_id", s.matchID)
		deaths = nil
	}
	s.in.Deaths = deaths
}
