// Package squadagg — equipment_usage.go : L'ORCHESTRATION DU BLOC « SERVI OU
// GÂCHÉ » AU GRAIN PÉRIODE (étapes E5.5, E6.1 puis E6.1bis du
// PLAN_EQUIPEMENT_GACHIS_2026-09-09).
//
// UN SEUL ASSEMBLAGE POUR PLUSIEURS PAGES. La Synthèse (scope = les matchs du
// joueur sur la période filtrée) et l'Escouade/Teammates (scope = les matchs
// filtrés de la page) posent exactement la même question et lisent exactement
// les mêmes trois vues. Un builder par page aurait fait diverger deux fois la
// règle de camp et la résolution des amis — c'est le patron « helper de
// package » (jamais un service qui en appelle un autre : couplage horizontal,
// cf. skill arch-rules).
//
// POURQUOI DANS squadagg ET PAS service (E6.1bis, 2026-09-09) : le service-root
// (package service) importe déjà internal/service/teammates (synthesis_service_usage.go,
// pour teammates.FriendGamertagsResolver). Si TeammatesService avait besoin
// d'appeler une fonction du package service, ce serait un cycle service→teammates→
// service. squadagg est une FEUILLE (ne dépend ni de service ni de teammates) déjà
// importée des deux côtés — même patron que BuildSquadHeader/IntersectByMatchID,
// déplacés ici pour la même raison (K3b). Le package service continue d'appeler
// cette fonction via l'alias de squadagg_reexport.go : zéro site d'appel changé.
//
// AUCUN SQL ICI, AUCUNE REQUÊTE NEUVE AILLEURS : les trois lectures sont celles de
// la page Sessions (port.SessionUsageRepository → duckdb.SessionUsageRepo), qui
// prennent déjà un scope FERMÉ de match_id — la seule chose qui change d'une page
// à l'autre est la liste d'identifiants qu'on leur passe.
//
// BEST-EFFORT ET DIT : capability absente (repo non câblé) ou lecture en échec ⇒
// bloc présent avec une raison MACHINE, jamais un 500 ni un bloc muet. Les
// constantes de raison sont celles du bloc de session (domain.SessionUsage*) : un
// second vocabulaire pour les mêmes deux états n'aurait rien dit de plus.
package squadagg

import (
	"context"
	"log/slog"
	"strconv"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/replaylabels"
	"levelup/go-api/internal/port"
)

// EquipmentUsageQuery — tout ce que l'assemblage demande à la page appelante : la
// source (nil ⇒ titre sans film.usage_summary), le joueur de la route, le scope
// FERMÉ de matchs, et les amis (coéquipiers suivis/sélectionnés ou amis
// configurés selon la page appelante) qui pourront devenir la part « mes amis »
// des deux donuts.
type EquipmentUsageQuery struct {
	Repo            port.SessionUsageRepository
	PlayerXUID      string
	MatchIDs        []string
	FriendGamertags []string
	// RepoRoot / TitleSlug / Locale : de quoi NOMMER les armes du detail par niveau
	// (catalogue du titre). Vides = les armes s'affichent sous leur cle, ce qui est la
	// degradation ecrite partout ailleurs — jamais un nom approchant.
	RepoRoot  string
	TitleSlug string
	Locale    string
	// Lectures : les trois lectures du résumé d'usage DÉJÀ faites sur ce scope par la page
	// (LireUsage), partagées avec le bloc « formes retenues » (lot perf L2, D2.6). Nil ⇒
	// l'assemblage les fait lui-même (Synthèse, Sessions).
	Lectures *LecturesUsage
}

// LecturesUsage : les trois lectures communes aux blocs « servi ou gâché » et « formes
// retenues » sur un scope — les films, les joueurs, les participants —, erreurs comprises.
type LecturesUsage struct {
	Films        map[string]sessionusage.FilmRow
	Players      []sessionusage.PlayerRow
	Participants []sessionusage.ParticipantRow
	erreurs      []error // dans l'ordre films, joueurs, participants
}

// LireUsage fait les trois lectures communes sur le scope, toutes, même après un échec
// (comme les blocs le faisaient chacun) ; Erreur rend la première en échec.
func LireUsage(ctx context.Context, repo port.SessionUsageRepository, matchIDs []string) *LecturesUsage {
	films, filmsErr := repo.LoadUsageFilms(ctx, matchIDs)
	players, playersErr := repo.LoadUsagePlayers(ctx, matchIDs)
	participants, partErr := repo.LoadParticipants(ctx, matchIDs)
	return &LecturesUsage{
		Films: films, Players: players, Participants: participants,
		erreurs: []error{filmsErr, playersErr, partErr},
	}
}

// Erreur rend la première lecture en échec, dans l'ordre films, joueurs, participants.
func (l *LecturesUsage) Erreur() error {
	for _, err := range l.erreurs {
		if err != nil {
			return err
		}
	}
	return nil
}

// BuildEquipmentUsageBlock lit le résumé d'usage sur le scope et rend le bloc
// contractuel. Scope vide ⇒ nil (rien à publier, pas même une indisponibilité).
func BuildEquipmentUsageBlock(ctx context.Context, q EquipmentUsageQuery) *domain.EquipmentUsageBlock {
	if len(q.MatchIDs) == 0 {
		return nil
	}
	if q.Repo == nil || q.PlayerXUID == "" {
		return &domain.EquipmentUsageBlock{
			UnavailableReason: domain.SessionUsageUnsupported, MatchesTotal: len(q.MatchIDs),
		}
	}
	lu := q.Lectures
	if lu == nil {
		lu = LireUsage(ctx, q.Repo, q.MatchIDs)
	}
	if err := lu.Erreur(); err != nil {
		slog.ErrorContext(ctx, "equipment usage: lecture du résumé d'usage en échec",
			"err", err, "match_count", len(q.MatchIDs))
		return &domain.EquipmentUsageBlock{
			UnavailableReason: domain.SessionUsageLoadFailed, MatchesTotal: len(q.MatchIDs),
		}
	}
	participants := lu.Participants

	tc := sessionusage.BuildTeamContext(q.PlayerXUID, participants)
	friends := sessionusage.ResolveScopeFriends(q.PlayerXUID, participants, q.FriendGamertags)
	in := sessionusage.OverviewInput{
		PlayerXUID: q.PlayerXUID,
		Matches:    sessionusage.BuildMatchInputs(q.MatchIDs, lu.Films, lu.Players, tc),
	}
	for _, f := range friends {
		in.FriendXUIDs = append(in.FriendXUIDs, f.XUID)
	}
	block := sessionusage.ComputeUsageOverview(in)
	block.TrackedPlayers = friends
	attacherNiveauxDArmes(ctx, &block, q, tc)
	// La couverture des films n'est jamais totale : un scope entier sans match
	// mesuré est un état légitime (le bloc dit « 0/N »), mais il ne doit pas
	// passer en silence — c'est le premier symptôme d'une recuisson manquante.
	if block.MatchesMeasured == 0 {
		slog.InfoContext(ctx, "equipment usage: aucun match du scope n'est mesuré",
			"match_count", len(q.MatchIDs), "player_xuid", q.PlayerXUID)
	}
	return &block
}

// attacherNiveauxDArmes ajoute au bloc les prises de socle rangees par niveau.
//
// Best-effort et DIT : repo sans ce loader ⇒ silence ; lecture en echec ⇒ WARN, le reste du
// bloc est servi. Le calcul est CELUI DE LA PAGE SESSIONS (`sessionusage.ComputePadTiers`) :
// deux projections de la meme regle divergeraient au premier changement.
func attacherNiveauxDArmes(
	ctx context.Context, block *domain.EquipmentUsageBlock,
	q EquipmentUsageQuery, tc sessionusage.TeamContext,
) {
	rows, err := q.Repo.LoadPadTiers(ctx, q.MatchIDs)
	if err != nil {
		slog.WarnContext(ctx, "equipment usage: niveaux d'armes indisponibles", "err", err)
		return
	}
	block.PadTiers = sessionusage.ComputePadTiers(sessionusage.PadTiersInput{
		Rows:         rows,
		PlayerXUID:   q.PlayerXUID,
		MatchesTotal: len(q.MatchIDs),
		PlayerTeam:   tc.PlayerTeam,
		TeamOf:       tc.TeamOf,
	})
	nommerArmes(ctx, block.PadTiers, q)
}

// nommerArmes charge le catalogue d'armes du titre et nomme le detail par niveau.
//
// Best-effort et DIT : sans `RepoRoot`/`TitleSlug` (montage qui ne les passe pas) le bloc reste
// servi, les armes sous leur cle. Catalogue illisible ⇒ WARN, meme degradation.
func nommerArmes(ctx context.Context, block *domain.SessionUsagePadTiersBlock, q EquipmentUsageQuery) {
	if block == nil || q.RepoRoot == "" || q.TitleSlug == "" {
		return
	}
	cat, err := replaylabels.Catalogue(q.RepoRoot, q.TitleSlug)
	if err != nil {
		slog.WarnContext(ctx, "equipment usage: catalogue d'armes illisible — armes des niveaux non nommees",
			"err", err, "titleSlug", q.TitleSlug)
		return
	}
	NommerArmesDesNiveaux(block, cat.Weapons, q.Locale != "en", parseFamilleArme)
}

// parseFamilleArme rend l'identifiant 32 bits d'une cle de famille du contrat — huit
// hexadecimaux minuscules, sans « 0x » (la forme de `replay.PadWeaponFamilyKey`).
//
// DEUXIEME ET DERNIERE COPIE de `service.parseWeaponFamilyKey` : les deux paquets sont
// disjoints (squadagg ne peut pas importer service, qui l'importe). Une TROISIEME copie
// devrait etre centralisee (regle CLAUDE.md n°6).
func parseFamilleArme(key string) (uint32, bool) {
	v, err := strconv.ParseUint(key, 16, 32)
	if err != nil {
		return 0, false
	}
	return uint32(v), true
}
