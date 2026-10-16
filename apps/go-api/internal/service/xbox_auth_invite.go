// Package service — xbox_auth_invite.go : l'invitation portée par la session au
// login SSO Xbox (réservation, compensation, effets).
//
// Une invitation, avec ou sans groupe, lève le verrou d'instance pour un xuid inconnu
// (D3, 2026-09-15). Elle n'est valable que si son émetteur a encore un compte
// (userstore.InviteStore). Les journaux ne portent jamais le code : il vaut accès.
package service

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
)

// claimPendingInvite réserve l'invitation portée par la session, ou rend nil. Une
// invitation introuvable, expirée, déjà consommée ou dont l'émetteur n'existe plus,
// ou l'absence de store câblé, vaut absence d'invitation (login normal, soumis au
// verrou). La réservation est atomique : un code ne sert qu'une fois.
func (s *XboxSSOLinkStrategy) claimPendingInvite(ctx context.Context, sess *domain.SessionData, attempt *auth.Attempt) *domain.InviteCode {
	if s.invites == nil || sess == nil || sess.PendingInviteCode == "" {
		return nil
	}
	inv, err := s.invites.Consume(sess.PendingInviteCode, attempt.Gamertag)
	if err != nil {
		slog.WarnContext(ctx, "xbox_sso: invitation en session inutilisable, ignorée",
			"xuid", attempt.XUID, "err", err)
		return nil
	}
	return inv
}

// refusalCause précise un refus du verrou d'instance : quand la session présentait une
// invitation que claimPendingInvite n'a pas pu réserver, le refus devient
// auth.ErrInviteInvalid (toujours un auth.ErrInstanceLocked pour errors.Is). Toute autre
// erreur est rendue telle quelle.
func refusalCause(err error, sess *domain.SessionData, claimed *domain.InviteCode) error {
	if errors.Is(err, auth.ErrInstanceLocked) && claimed == nil && sess != nil && sess.PendingInviteCode != "" {
		return auth.ErrInviteInvalid
	}
	return err
}

// releaseInvite rend une invitation réservée quand le login échoue après coup.
func (s *XboxSSOLinkStrategy) releaseInvite(ctx context.Context, inv *domain.InviteCode, attempt *auth.Attempt) {
	if inv == nil {
		return
	}
	if err := s.invites.Release(inv.Code, attempt.Gamertag); err != nil {
		slog.ErrorContext(ctx, "xbox_sso: invitation réservée non rendue après un login échoué",
			"xuid", attempt.XUID, "err", err)
	}
}

// redeemInvite applique une invitation réservée : ajout au groupe SI l'invitation en
// porte un, droit de provisioning sur le compte UNIQUEMENT s'il vient d'être créé
// (`created`) — un compte existant qui présente une invitation ne gagne aucun droit
// de créer un profil.
//
// Best-effort : un échec est loggé mais ne bloque pas le login (la session est déjà
// câblée). Vide PendingInviteCode : le code est consommé.
func (s *XboxSSOLinkStrategy) redeemInvite(
	ctx context.Context, inv *domain.InviteCode, attempt *auth.Attempt,
	sess *domain.SessionData, user *domain.User, created bool,
) {
	if inv.GroupID != "" {
		if s.groups != nil {
			if err := s.groups.AddMember(inv.GroupID, attempt.XUID, attempt.Gamertag); err != nil {
				slog.ErrorContext(ctx, "xbox_sso: ajout au groupe échoué (non bloquant)",
					"group_id", inv.GroupID, "xuid", attempt.XUID, "err", err)
			} else {
				slog.InfoContext(ctx, "xbox_sso: joueur ajouté au groupe via invitation",
					"group_id", inv.GroupID, "gamertag", attempt.Gamertag)
			}
		}
	} else {
		slog.InfoContext(ctx, "xbox_sso: invitation sans groupe consommée", "gamertag", attempt.Gamertag)
	}
	if created && user != nil {
		if err := s.users.SetProvisionGrant(user.Username, inv.Code); err != nil {
			slog.ErrorContext(ctx, "xbox_sso: pose du droit de provisioning échouée (non bloquant)",
				"username", user.Username, "err", err)
		} else {
			slog.InfoContext(ctx, "xbox_sso: droit de provisioning posé pour l'invité",
				"username", user.Username)
		}
	}
	sess.PendingInviteCode = ""
}
