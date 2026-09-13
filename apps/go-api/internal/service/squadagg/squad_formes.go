// Package squadagg — squad_formes.go : L'ORCHESTRATION DU BLOC « FORMES
// RETENUES » de la page Escouade (artefact 2ec1b8eb, lot D2 du 2026-09-13).
//
// MÊME PATRON QUE equipment_usage.go, et pour les mêmes raisons : un helper de
// package feuille, appelé par la page, jamais un service qui en appelle un
// autre. Cinq lectures, toutes sur le scope FERMÉ de matchs que la page a déjà
// filtré (période + cascade + sessions) — les trois du résumé d'usage, le grain
// match des socles, et les colonnes d'objectif.
//
// BEST-EFFORT ET DIT : source non câblée (titre sans film) ⇒ bloc avec une
// raison MACHINE ; lecture en échec ⇒ idem, jamais un 500. Les colonnes
// d'objectif, elles, dégradent SEULES : leur absence retire cinq cartes sur
// dix-neuf, pas le bloc.
//
// LE CATALOGUE D'ARMES SE RÉSOUT ICI, À LA REQUÊTE — même arbitrage, mot pour
// mot, que `service/session_page_usage_labels.go` : le sidecar ne stocke que la
// clé de famille du film (huit hexadécimaux), et ce qui se résout d'un catalogue
// du titre se résout au service, jamais dans un artefact déjà cuit. Le catalogue
// est chargé POUR LE TITRE du service : un titre sans ces tables ne reçoit aucun
// nom, jamais le nom d'un autre titre.
package squadagg

import (
	"context"
	"log/slog"
	"strconv"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squadformes"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/halo_infinite/replaylabels"
	"levelup/go-api/internal/games/weapons"
	"levelup/go-api/internal/port"
)

// SquadFormesQuery — ce que la page fournit à l'assemblage.
type SquadFormesQuery struct {
	// Repo : le résumé d'usage + le grain match des socles. Nil ⇒ titre sans
	// film.usage_summary.
	Repo port.SquadFormesUsageRepository
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
	// SelectedGamertags : les coéquipiers sélectionnés de la page.
	SelectedGamertags []string
	// RepoRoot / TitleSlug / Locale : la résolution du catalogue d'armes.
	// RepoRoot vide ⇒ armes non nommées (montage sans cette capacité).
	RepoRoot  string
	TitleSlug string
	Locale    string
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
	films, filmsErr := q.Repo.LoadUsageFilms(ctx, matchIDs)
	players, playersErr := q.Repo.LoadUsagePlayers(ctx, matchIDs)
	participants, partErr := q.Repo.LoadParticipants(ctx, matchIDs)
	pads, padsErr := q.Repo.LoadUsageFilmPads(ctx, matchIDs)
	for _, err := range []error{filmsErr, playersErr, partErr, padsErr} {
		if err != nil {
			slog.ErrorContext(ctx, "formes retenues: lecture du résumé d'usage en échec",
				"err", err, "match_count", len(matchIDs))
			return &domain.SquadFormesBlock{
				UnavailableReason: domain.SessionUsageLoadFailed, MatchesTotal: len(q.Metas),
			}
		}
	}

	tc := sessionusage.BuildTeamContext(q.PlayerXUID, participants)
	in := squadformes.Input{
		PlayerXUID:   q.PlayerXUID,
		SquadPlayers: formesSquadPlayers(q.PlayerXUID, q.MainGamertag, participants, q.SelectedGamertags),
		Metas:        q.Metas,
		Matches:      sessionusage.BuildMatchInputs(matchIDs, films, players, tc),
		Films:        films,
		Pads:         pads,
		Gamertags:    formesGamertags(participants),
		Objectives:   loadFormesObjectives(ctx, q, matchIDs),
		Weapons:      formesWeaponCatalog(ctx, q),
		// La clé du mur voyage en PARAMÈTRE : `internal/analysis` est
		// title-agnostic (ADR 0012/0025) et ne peut pas importer le paquet du
		// titre qui nomme ses familles d'équipement.
		WallFamilyKey: replay.UsageFamilyWallKey,
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

// formesSquadPlayers — le joueur de la page EN TÊTE, puis les coéquipiers
// sélectionnés résolus contre les participants du scope (même résolution que le
// bloc d'usage : ResolveScopeFriends, insensible à la casse du gamertag).
//
// LE NOM DU JOUEUR DE LA PAGE VIENT DE LA PAGE, et les participants ne sont que
// son repli : sur un scope dont aucune ligne de participant ne porte son
// gamertag, l'écran affichait son XUID.
func formesSquadPlayers(
	playerXUID, mainGamertag string, participants []sessionusage.ParticipantRow, selected []string,
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
	friends := sessionusage.ResolveScopeFriends(playerXUID, participants, selected)
	out := make([]domain.SessionUsageSquadPlayer, 0, 1+len(friends))
	out = append(out, me)
	return append(out, friends...)
}

// formesGamertags — xuid -> gamertag sur le scope. Un xuid sans nom garde sa
// clé : le web affiche alors l'identifiant plutôt qu'un « inconnu » (décision
// utilisateur : aucune vie anonyme, tout slot est nommé à la source).
func formesGamertags(participants []sessionusage.ParticipantRow) map[string]string {
	out := make(map[string]string, len(participants))
	for _, p := range participants {
		if p.Gamertag != "" {
			out[p.XUID] = p.Gamertag
		}
	}
	return out
}

// loadFormesObjectives — les colonnes d'objectif du scope. DÉGRADATION SEULE :
// repo non câblé ou lecture en échec ⇒ nil + log, le bloc perd ses cartes
// d'objectif et garde les quatorze autres.
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
	return rows
}

// formesWeaponCatalog — clé de famille d'arme -> nom du titre et dimensions du
// registre canonique. Map vide (jamais nil-panic) quand le catalogue manque :
// les armes gardent alors leur clé à l'écran, jamais un nom approchant.
func formesWeaponCatalog(ctx context.Context, q SquadFormesQuery) map[string]squadformes.WeaponInfo {
	out := map[string]squadformes.WeaponInfo{}
	if q.RepoRoot == "" || q.TitleSlug == "" {
		return out
	}
	cat, err := replaylabels.Load(q.RepoRoot, q.TitleSlug)
	if err != nil {
		slog.WarnContext(ctx, "formes retenues: catalogue d'armes illisible — socles non nommés",
			"err", err, "titleSlug", q.TitleSlug)
		return out
	}
	frPreferred := q.Locale != "en"
	roles, classes := weapons.RolesByKey(), weapons.ClassesByKey()
	for family, key := range cat.Keys {
		info := squadformes.WeaponInfo{WeaponKey: key, Role: roles[key], Class: classes[key]}
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
