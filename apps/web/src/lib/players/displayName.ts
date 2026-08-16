/**
 * displayName.ts — résolveur d'affichage UNIQUE pour les identités joueur.
 *
 * Garantit qu'aucune chaîne au format xuid brut (numérique long ou "xuid(...)")
 * n'est JAMAIS rendue à l'écran, quelle que soit la donnée serveur. C'est le
 * chokepoint front de la règle « jamais de xuid, toujours un gamertag » — miroir
 * de analysis.GamertagLookupViewSQL / analysis.MaskedXuidLabelSQL côté Go.
 *
 * Tout rendu d'un nom de joueur DOIT passer par `displayPlayerName(gamertag, xuid)`
 * au lieu d'un fallback ad hoc `gamertag || xuid`.
 *
 * Note bots : les identifiants bot "bid(N.0)" ne sont PAS masqués ici — ils sont
 * résolus côté serveur (BotSQLCase → "343 …") ; un bot inconnu résiduel reste
 * lisible tel quel plutôt que d'être étiqueté à tort "Joueur".
 */

/**
 * Format xuid joueur brut : "xuid(...)" OU suite purement numérique de >= 15
 * chiffres (xuid Xbox décimal). Volontairement strict pour ne pas masquer des
 * gamertags légitimes (un gamertag peut contenir des chiffres mais pas en faire
 * 15 d'affilée).
 */
const RAW_XUID_RE = /^xuid\(|^\d{15,}$/i

/** true si `s` ressemble à un identifiant xuid joueur brut (à ne jamais afficher). */
export function isXuidLike(s: string): boolean {
  return RAW_XUID_RE.test(s.trim())
}

/**
 * Libellé masqué stable pour un xuid non résolu : "Joueur ####" (4 derniers
 * caractères). Permet de distinguer plusieurs inconnus dans un même chart sans
 * exposer l'identifiant complet.
 */
export function maskedPlayerLabel(xuid: string): string {
  const tail = xuid.slice(-4) || xuid
  return `Joueur ${tail}`
}

/**
 * Nom d'affichage d'un joueur, GARANTI sans format xuid brut :
 *  - `gamertag` non vide et non xuid-like → `gamertag`
 *  - sinon `xuid` fourni → `maskedPlayerLabel(xuid)`
 *  - sinon → "Joueur inconnu"
 */
export function displayPlayerName(
  gamertag?: string | null,
  xuid?: string | null,
): string {
  const gt = gamertag?.trim()
  if (gt && !isXuidLike(gt)) return gt
  const xu = xuid?.trim()
  if (xu) return maskedPlayerLabel(xu)
  return 'Joueur inconnu'
}

/**
 * Clé de comparaison d'un gamertag : minuscules, sans espaces de bord. C'est la clé de
 * l'appariement « ce joueur est-il un ami ? » (`settings.friend_gamertags`) — la même
 * partout où la question se pose (charts de la Match View, rejeu 2D), pour qu'un ami
 * saisi « Ma Pote » et présent comme « MA POTE » soit reconnu des deux côtés.
 */
export function normalizeGamertagKey(gt: string | null | undefined): string {
  return (gt ?? '').toLowerCase().trim()
}
