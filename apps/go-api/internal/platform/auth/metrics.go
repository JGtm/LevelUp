// Package auth — metrics.go : compteurs expvar des échanges OAuth v2
// (namespace levelup.auth.*, pattern identique à pool/metrics.go — ADR 0009).
// La brique `haloapi/xblauth` ne compte rien : elle rend une issue typée
// (`xblauth.RefreshOutcome`) que la façade comptabilise ici.
package auth

import (
	"expvar"

	"levelup/go-api/internal/haloapi/xblauth"
)

var (
	oauthRefreshTotal            = expvar.NewInt("levelup.auth.oauth_refresh_total")
	oauthRefreshFailTotal        = expvar.NewInt("levelup.auth.oauth_refresh_fail_total")
	oauthRefreshFailByClass      = expvar.NewMap("levelup.auth.oauth_refresh_fail_by_class")
	oauthRefreshRetryPublicTotal = expvar.NewInt("levelup.auth.oauth_refresh_retry_public_total")
	oauthRefreshRetryMSATotal    = expvar.NewInt("levelup.auth.oauth_refresh_retry_msa_total")
)

// recordOAuthRefreshIssue comptabilise un rafraîchissement : les reprises tentées (retry public
// AADSTS90023, repli MSA natif) puis l'issue.
func recordOAuthRefreshIssue(out xblauth.RefreshOutcome, err error) {
	if out.RetentePublique {
		oauthRefreshRetryPublicTotal.Add(1)
	}
	if out.RetenteNatif {
		oauthRefreshRetryMSATotal.Add(1)
	}
	recordOAuthRefreshOutcome(err)
}

// recordOAuthRefreshOutcome incrémente les compteurs d'issue d'un échange.
func recordOAuthRefreshOutcome(err error) {
	oauthRefreshTotal.Add(1)
	if err == nil {
		return
	}
	oauthRefreshFailTotal.Add(1)
	oauthRefreshFailByClass.Add(string(ClassifyAuthError(err)), 1)
}
