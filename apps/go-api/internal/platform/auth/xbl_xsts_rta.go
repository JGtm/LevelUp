// Package auth — xbl_xsts_rta.go : façade d'exploitation de l'XSTS Xbox Live pour RTA (le
// protocole vit dans `internal/haloapi/xblauth/xsts_rta.go`, plan D3 lot 2). La façade ajoute
// l'identifiant d'événement du journal, le client HTTP, le créneau XBL, la provenance du `ctx`
// et, pour la variante à magasin, la lecture et la persistance de la provenance du joueur.
package auth

import (
	"context"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/haloapi/xblauth"
	"levelup/go-api/internal/observability/logging"
)

// XSTSResult contient le token XSTS + userhash nécessaires pour la connexion RTA.
type XSTSResult = xblauth.XSTSResult

// AcquireXSTSForRTAWithProvenance acquiert le XSTS RTA en posant la provenance
// persistée du joueur (préfixe RpsTicket connu → aucun 401 inutile à l'échange
// XBL), puis persiste celle qui a été MESURÉE si elle a changé.
//
// À préférer à AcquireXSTSForRTA partout où l'appelant dispose du store et du
// xuid. Sans elle, l'échange repart du préfixe par défaut « d= » et les comptes
// qui n'acceptent que « t= » encaissent un 401 puis un retry à CHAQUE boot.
//
// store ou xuid absents → dégrade proprement en AcquireXSTSForRTA (aucune
// provenance posée, aucune mesure persistée).
func AcquireXSTSForRTAWithProvenance(
	ctx context.Context, store *MultiUserTokenStore, xuid, accessToken string,
) (*XSTSResult, error) {
	if store == nil || xuid == "" {
		return AcquireXSTSForRTA(ctx, accessToken)
	}
	known := ""
	if ut, err := store.Load(xuid); err == nil && ut != nil {
		known = ut.TokenClientFamily
	}
	ctx, observer := WithTokenFamilyObserver(WithTokenClientFamily(ctx, known))

	res, err := AcquireXSTSForRTA(ctx, accessToken)
	if observed := observer.Observed(); observed != "" && observed != known {
		if uerr := store.UpdateTokenClientFamily(xuid, observed); uerr != nil {
			slog.ErrorContext(ctx, "xsts: persistance de la provenance échouée — 401 RpsTicket au prochain échange",
				"xuid", xuid, "family", observed, "err", uerr)
		}
	}
	return res, err
}

// AcquireXSTSForRTA obtient un token XSTS avec RelyingParty=http://xboxlive.com.
// Ce token est nécessaire pour la connexion WebSocket RTA.
func AcquireXSTSForRTA(ctx context.Context, accessToken string) (*XSTSResult, error) {
	// Sprint B1 commit 18 : event_id pour tracer l'acquisition XSTS RTA
	// (2 appels HTTP : User Token XBL + XSTS Xbox Live) ; porté par le ctx jusque dans
	// les journaux de la brique.
	ctx, evID := logging.WithEvent(ctx, "auth.xsts.rta")
	slog.InfoContext(ctx, "xsts: début acquisition XSTS pour RTA", "event", evID)
	return xblauth.AcquireXSTSForRTA(ctx, chaineDepuisCtx(ctx, &http.Client{Timeout: xblExchangeTimeout}), accessToken)
}
