// Package squadagg — squad_formes.go : L'ORCHESTRATION DU BLOC « FORMES RETENUES » (cartes
// d'objectif de l'Escouade et des Séries temporelles).
//
// UN HELPER DE PACKAGE FEUILLE, appelé par la page, jamais un service qui en appelle un autre
// (couplage horizontal, skill arch-rules). Les lectures portent toutes sur le scope FERMÉ de
// matchs que la page a déjà filtré — les trois du résumé d'usage (film décodé, camps) et les
// colonnes d'objectif.
//
// BEST-EFFORT ET DIT : source non câblée (titre sans film) ⇒ bloc avec une raison MACHINE ;
// lecture en échec ⇒ idem, jamais un 500. Les colonnes d'objectif, elles, dégradent SEULES.
//
// LE CATALOGUE D'ARMES (`WeaponCatalog`, lu par l'Emprise) SE RÉSOUT ICI, À LA REQUÊTE : le
// sidecar ne stocke que la clé de famille du film, et ce qui se résout d'un catalogue du titre se
// résout au service. Le catalogue est chargé POUR LE TITRE du service : un titre sans ces tables ne
// reçoit aucun nom.
package squadagg

import (
	"context"
	"log/slog"
	"strconv"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squadformes"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/replaylabels"
	"levelup/go-api/internal/port"
)

// SquadFormesQuery — ce que la page fournit à l'assemblage.
type SquadFormesQuery struct {
	// Repo : le résumé d'usage. Nil ⇒ titre sans film.usage_summary.
	Repo port.SessionUsageRepository
	// Objectives : les colonnes d'objectif. Nil ⇒ bloc sans cartes d'objectif.
	Objectives port.SquadFormesObjectiveRepository
	PlayerXUID string
	// MainGamertag : le nom du joueur de la page, tel que la page le connaît
	// déjà. C'est la source PRIMAIRE de son libellé — les lignes de
	// `match_participants` d'un scope donné peuvent toutes avoir un gamertag
	// vide, et l'écran affichait alors son XUID (défaut mesuré le 2026-09-13 ;
	// règle du dépôt : aucune vie anonyme, aucun identifiant machine à l'écran).
	MainGamertag string
	// Metas : le scope dans l'ordre d'affichage — c'est LUI qui fait le scope.
	Metas []squadformes.MatchMeta
	// SelectedMembers : les coéquipiers sélectionnés de la page, résolus par la page (xuid) et
	// sous le nom qu'elle leur donne. Vide : le joueur de la page seul.
	SelectedMembers []domain.SessionUsageSquadPlayer
	// Lectures : les trois lectures du résumé d'usage déjà faites sur ce scope par la page,
	// partagées avec l'Emprise (ADR 0036 I4). Nil ⇒ lues ici.
	Lectures *LecturesUsage
}

// BuildSquadFormesBlock lit le scope et rend le bloc contractuel. Scope vide ⇒
// nil (rien à publier, pas même une indisponibilité).
func BuildSquadFormesBlock(ctx context.Context, q SquadFormesQuery) *domain.SquadFormesBlock {
	if len(q.Metas) == 0 {
		return nil
	}
	if q.Repo == nil || q.PlayerXUID == "" {
		return &domain.SquadFormesBlock{
			UnavailableReason: domain.SessionUsageUnsupported, MatchesTotal: len(q.Metas),
		}
	}
	matchIDs := make([]string, 0, len(q.Metas))
	for _, m := range q.Metas {
		matchIDs = append(matchIDs, m.MatchID)
	}
	lu := q.Lectures
	if lu == nil {
		lu = LireUsage(ctx, q.Repo, matchIDs)
	}
	films, players, participants := lu.Films, lu.Players, lu.Participants
	if err := lu.Erreur(); err != nil {
		slog.ErrorContext(ctx, "formes retenues: lecture du résumé d'usage en échec",
			"err", err, "match_count", len(matchIDs))
		return &domain.SquadFormesBlock{
			UnavailableReason: domain.SessionUsageLoadFailed, MatchesTotal: len(q.Metas),
		}
	}

	tc := sessionusage.BuildTeamContext(q.PlayerXUID, participants)
	in := squadformes.Input{
		PlayerXUID:   q.PlayerXUID,
		SquadPlayers: SquadPlayers(q.PlayerXUID, q.MainGamertag, participants, q.SelectedMembers),
		Metas:        q.Metas,
		Matches:      sessionusage.BuildMatchInputs(matchIDs, films, players, tc),
		Objectives:   loadFormesObjectives(ctx, q, matchIDs),
	}
	block := squadformes.Build(in)
	// Un scope entier sans film n'est pas une panne (le bloc dit « 0 sur N »),
	// mais c'est le premier symptôme d'une recuisson manquante : il se journalise.
	if block.MatchesMeasured == 0 {
		slog.InfoContext(ctx, "formes retenues: aucun match du scope n'est mesuré",
			"match_count", len(matchIDs), "player_xuid", q.PlayerXUID)
	}
	return &block
}

// SquadPlayers — le joueur de la page EN TÊTE, puis les coéquipiers
// sélectionnés alliés dans le scope (sessionusage.ResolveScopeMembers : par xuid,
// jamais par égalité de gamertag — ADR 0035).
//
// LES NOMS VIENNENT DE LA PAGE, et les participants ne sont que leur repli : sur
// un scope dont aucune ligne de participant ne porte un gamertag, l'écran
// affichait le XUID du joueur, et ses coéquipiers disparaissaient du bloc.
func SquadPlayers(
	playerXUID, mainGamertag string, participants []sessionusage.ParticipantRow, selected []domain.SessionUsageSquadPlayer,
) []domain.SessionUsageSquadPlayer {
	me := domain.SessionUsageSquadPlayer{XUID: playerXUID, Gamertag: mainGamertag}
	for _, p := range participants {
		if me.Gamertag != "" {
			break
		}
		if p.XUID == playerXUID && p.Gamertag != "" {
			me.Gamertag = p.Gamertag
		}
	}
	membres := sessionusage.ResolveScopeMembers(playerXUID, participants, selected)
	out := make([]domain.SessionUsageSquadPlayer, 0, 1+len(membres))
	out = append(out, me)
	return append(out, membres...)
}

// loadFormesObjectives — les colonnes d'objectif du scope. DÉGRADATION SEULE :
// repo non câblé ou lecture en échec ⇒ nil + log, le bloc perd ses cartes
// d'objectif et garde ses comptes.
func loadFormesObjectives(
	ctx context.Context, q SquadFormesQuery, matchIDs []string,
) []squadformes.ObjectiveColumnRow {
	if q.Objectives == nil {
		return nil
	}
	rows, err := q.Objectives.LoadObjectiveColumnRows(ctx, matchIDs)
	if err != nil {
		slog.WarnContext(ctx, "formes retenues: colonnes d'objectif illisibles — cartes d'objectif omises",
			"err", err, "match_count", len(matchIDs))
		return nil
	}
	return joindrePrisesNettes(ctx, q, matchIDs, rows)
}

// joindrePrisesNettes ajoute la grandeur lue du FILM aux lignes lues de l'API.
//
// LA JOINTURE SE FAIT ICI, ET PAS EN SQL : les deux grandeurs vivent dans deux
// tables alimentées par deux producteurs (cf. le commentaire de
// LoadFlagGrabsNet). Un LEFT JOIN aurait fait tomber toutes les colonnes
// d'objectif le jour où la vue du film manque.
//
// UN MATCH SANS PRISE LUE NE REÇOIT AUCUNE CLÉ : l'absence porte le « non
// mesuré » jusqu'à l'écran, et écrire 0 ici dirait « il n'a rien pris ».
// DÉGRADATION SEULE : lecture en échec ⇒ les lignes sortent telles quelles, le
// bloc garde ses autres colonnes.
func joindrePrisesNettes(
	ctx context.Context, q SquadFormesQuery, matchIDs []string, rows []squadformes.ObjectiveColumnRow,
) []squadformes.ObjectiveColumnRow {
	if len(rows) == 0 {
		return rows
	}
	nets, err := q.Objectives.LoadFlagGrabsNet(ctx, matchIDs)
	if err != nil {
		slog.WarnContext(ctx, "formes retenues: prises nettes illisibles — grandeur omise",
			"err", err, "match_count", len(matchIDs))
		return rows
	}
	parMatch := make(map[string]map[string]int, len(nets))
	fenetre := map[string]float64{}
	for _, n := range nets {
		if parMatch[n.MatchID] == nil {
			parMatch[n.MatchID] = map[string]int{}
		}
		parMatch[n.MatchID][n.XUID] = n.Net
		fenetre[n.MatchID] = float64(n.WindowMS) / 1000
	}
	for i := range rows {
		v, ok := parMatch[rows[i].MatchID][rows[i].XUID]
		if !ok {
			continue
		}
		if rows[i].Values == nil {
			rows[i].Values = map[string]float64{}
		}
		rows[i].Values[narrative.GrandeurFlagGrabsNet] = float64(v)
		rows[i].FlagJuggleWindowSeconds = fenetre[rows[i].MatchID]
	}
	return rows
}

// WeaponCatalog — clé de famille d'arme -> nom du titre (langue de la requête) et clé canonique du
// registre. Map vide (jamais nil-panic) quand le catalogue manque : les armes gardent alors leur
// clé à l'écran, jamais un nom approchant. Lu par l'Emprise (Escouade, Séries temporelles).
func WeaponCatalog(ctx context.Context, repoRoot, titleSlug, locale string) map[string]squadformes.WeaponInfo {
	out := map[string]squadformes.WeaponInfo{}
	if repoRoot == "" || titleSlug == "" {
		return out
	}
	cat, err := replaylabels.Catalogue(repoRoot, titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "escouade: catalogue d'armes illisible — socles non nommés",
			"err", err, "titleSlug", titleSlug)
		return out
	}
	frPreferred := locale != "en"
	for family, key := range cat.Keys {
		info := squadformes.WeaponInfo{WeaponKey: key}
		if label, ok := cat.Weapons[family]; ok {
			info.Label = label.En
			if frPreferred && label.Fr != "" {
				info.Label = label.Fr
			}
		}
		out[formesWeaponFamilyKey(family)] = info
	}
	return out
}

// formesWeaponFamilyKey rend la clé de famille dans la forme du contrat : huit
// hexadécimaux MINUSCULES sans « 0x » — celle de `replay.PadWeaponFamilyKey`,
// donc celle de `pad_pickups_json` et de `weapon_pads_json`. Le catalogue, lui,
// indexe par entier 32 bits : c'est ici, et ici seulement, que les deux espaces
// se rejoignent.
func formesWeaponFamilyKey(family uint32) string {
	s := strconv.FormatUint(uint64(family), 16)
	for len(s) < 8 {
		s = "0" + s
	}
	return s
}
