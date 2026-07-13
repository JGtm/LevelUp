/**
 * Helpers d'affichage d'URL (revue UI 2026-07-13).
 *
 * verificationLinkLabel : libellé COURT d'un lien de vérification device-flow.
 * On n'affiche JAMAIS l'URL brute avec ses query params (code_challenge,
 * client_id, scope…) : c'est illisible, ça déborde du cadre, et ça n'apporte
 * rien à l'utilisateur. La bonne pratique anti-phishing est de montrer le
 * DOMAINE réel (l'utilisateur vérifie qu'il va bien chez Microsoft) — le host
 * + chemin suffisent, l'URL complète reste portée par le href du lien.
 */

/**
 * Retourne « host/chemin » sans protocole, sans query/hash, sans préfixe www.
 * Exemples :
 *   https://www.microsoft.com/link            → microsoft.com/link
 *   https://login.live.com/oauth20_authorize.srf?code_challenge=…
 *                                              → login.live.com/oauth20_authorize.srf
 * Entrée invalide (pas une URL absolue) : renvoyée telle quelle, tronquée du
 * protocole si présent (fallback défensif, jamais de throw).
 */
export function verificationLinkLabel(uri: string): string {
  try {
    const u = new URL(uri)
    const host = u.hostname.replace(/^www\./, '')
    const path = u.pathname === '/' ? '' : u.pathname
    return `${host}${path}`
  } catch {
    return uri.replace(/^https?:\/\//, '').split(/[?#]/)[0]
  }
}
