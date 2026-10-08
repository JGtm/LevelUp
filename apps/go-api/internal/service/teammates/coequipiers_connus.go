// Package teammates — coequipiers_connus.go : LES COÉQUIPIERS CONNUS du joueur consulté, une seule
// définition (ADR 0033, décision 1) : ses amis déclarés (réglages d'amis, par joueur) et les profils
// suivis du titre (db_profiles.json, hors profils auth_only). Rien d'autre : le nombre de matchs
// joués avec quelqu'un ne fait pas de lui un connu.
//
// C'est le pool de l'option « composition stricte » : un match où un connu HORS sélection était
// dans l'équipe du joueur principal est écarté ; un inconnu (remplissage de file, bot, adversaire)
// ne casse jamais une composition, quel que soit le nombre de matchs partagés.
//
// La résolution des amis en xuids (ResolveFriendXUIDs) est aussi celle des rencontres de la
// Carrière : le registre des profils d'abord, puis UNE lecture pour les autres (ADR 0036 I4).
package teammates

import (
	"context"
	"log/slog"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

// FriendXUIDsReader résout des gamertags en xuids en UNE lecture (clé : le gamertag tel que
// demandé ; absent = non résolu) — CareerRepo.ResolveFriendXUIDs en production.
type FriendXUIDsReader func(ctx context.Context, gamertags []string) (map[string]string, error)

// ProfilsDuTitre rend les profils déclarés du titre de la page (db_profiles.json, profils
// auth_only et titres en pause compris) — config.AppConfig.LoadPlayers en production.
type ProfilsDuTitre func(ctx context.Context) ([]domain.PlayerSummary, error)

// ResolveFriendXUIDs résout des amis déclarés (gamertags) en xuids, TOUS à la fois : d'abord
// `registre` (gamertag -> xuid des profils déclarés, comparé sans casse ni espaces parasites :
// un ami suivi a un xuid connu, aucune lecture), puis UNE lecture pour les autres (`lire` ; nil =
// aucune lecture). Rend xuid -> gamertag tel que déclaré.
//
// Best-effort, jamais une erreur : la lecture en échec est journalisée (DEBUG si la requête a pris
// fin), les amis que rien ne connaît aussi, en WARN (dérive de config : un gamertag des réglages que
// rien ne nomme). `page` nomme l'appelant dans ces journaux.
func ResolveFriendXUIDs(
	ctx context.Context, page string, amis []string, registre map[string]string, lire FriendXUIDsReader,
) map[string]string {
	parNom := make(map[string]string, len(registre))
	for gt, xuid := range registre {
		if xuid != "" {
			parNom[strings.ToLower(strings.TrimSpace(gt))] = xuid
		}
	}
	out := make(map[string]string, len(amis))
	var aLire []string
	for _, gt := range amis {
		if gt = strings.TrimSpace(gt); gt == "" {
			continue
		}
		if xuid := parNom[strings.ToLower(gt)]; xuid != "" {
			out[xuid] = gt
			continue
		}
		aLire = append(aLire, gt)
	}
	if len(aLire) == 0 {
		return out
	}
	var lus map[string]string
	if lire != nil {
		var err error
		if lus, err = lire(ctx, aLire); err != nil {
			slog.Log(ctx, observability.LevelUnlessCanceled(ctx, err, slog.LevelWarn),
				"friends_xuids_read_failed", "page", page, "friends", aLire, "err", err)
		}
	}
	var nonResolus []string
	for _, gt := range aLire {
		if xuid := lus[gt]; xuid != "" {
			out[xuid] = gt
		} else {
			nonResolus = append(nonResolus, gt)
		}
	}
	if len(nonResolus) > 0 {
		slog.WarnContext(ctx, "friends_xuids_unresolved",
			"page", page, "unresolved", nonResolus, "resolved", len(out))
	}
	return out
}

// WithCoequipiersConnus injecte les deux sources des coéquipiers connus : les profils du titre et la
// lecture qui résout les amis absents de ce registre. Sans elles, aucun coéquipier connu :
// l'option « composition stricte » n'écarte alors que les matchs à équipe inconnue.
func (s *TeammatesService) WithCoequipiersConnus(profils ProfilsDuTitre, lire FriendXUIDsReader) *TeammatesService {
	s.profils = profils
	s.lireAmis = lire
	return s
}

// coequipiersConnus rend xuid -> nom des coéquipiers connus du joueur consulté (cf. l'en-tête du
// fichier) : les profils suivis (hors auth_only) et ses amis déclarés, nommés par leur clé de
// profil quand ils en ont une, sinon comme dans les réglages. Joueur principal et sélection
// compris : buildExtraPoolXUIDs les retire. Une PAUSE du sync ne fait pas d'un joueur suivi un
// inconnu (la page est une lecture : domain.SyncablePlayers ne s'y applique pas). Registre
// illisible : journalisé, les amis se résolvent alors par la seule lecture.
func (s *TeammatesService) coequipiersConnus(ctx context.Context, amis []string) map[string]string {
	connus := map[string]string{}
	registre := map[string]string{}    // gamertag -> xuid, tous profils (résolution des amis)
	nomDuProfil := map[string]string{} // xuid -> clé de profil
	if s.profils != nil {
		profils, err := s.profils(ctx)
		if err != nil {
			slog.WarnContext(ctx, "teammates_profiles_unreadable",
				"player", s.gamertag, "titleSlug", s.titleSlug, "err", err)
		}
		for _, p := range profils {
			if p.XUID == "" {
				continue
			}
			registre[p.Gamertag] = p.XUID
			nomDuProfil[p.XUID] = p.Gamertag
			if !p.AuthOnly {
				connus[p.XUID] = p.Gamertag
			}
		}
	}
	for xuid, gt := range ResolveFriendXUIDs(ctx, "teammates", amis, registre, s.lireAmis) {
		if _, deja := connus[xuid]; deja {
			continue
		}
		if nom, ok := nomDuProfil[xuid]; ok {
			gt = nom
		}
		connus[xuid] = gt
	}
	return connus
}
