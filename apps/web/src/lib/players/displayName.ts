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
 *
 * SUFFIXE " [bot]" — RETIRÉ À L'AFFICHAGE (retour user 2026-09-02). Le producteur
 * killsource écrit le gamertag d'un bot avec le suffixe littéral ` [bot]`
 * (`apps/go-api/internal/games/halo_infinite/film/killsource/roster.go`,
 * `botSuffix`) et l'artefact de rejeu (schéma 36) le porte pareil : c'est un
 * MARQUEUR DE DONNÉES, pas un choix d'affichage — l'écran n'a pas à le répéter,
 * d'autant que le contexte le dit déjà (badge « Bot », style atténué). Xbox
 * interdit les crochets dans un gamertag : aucun joueur réel ne peut porter ce
 * suffixe, le retrait est donc sans risque de confusion. `stripBotSuffix` est
 * exporté pour les rares points d'affichage qui ne passent pas par
 * `displayPlayerName` (ex. un littéral déjà masqué par sa propre logique) —
 * jamais pour recopier le suffixe ailleurs.
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

/** Suffixe de DONNÉES posé par killsource sur le gamertag d'un bot — jamais un
 * choix d'affichage (cf. en-tête du fichier). Casse exacte, miroir du Go. */
const BOT_SUFFIX = ' [bot]'

/**
 * stripBotSuffix retire le suffixe ` [bot]` FINAL d'un gamertag, s'il y est —
 * jamais une occurrence en plein milieu du nom (donnée aberrante que ce helper
 * ne réinterprète pas). `displayPlayerName` l'applique déjà : n'appeler ce
 * helper directement que pour un point d'affichage qui ne passe pas par lui.
 */
export function stripBotSuffix(gamertag: string): string {
  return gamertag.endsWith(BOT_SUFFIX) ? gamertag.slice(0, -BOT_SUFFIX.length) : gamertag
}

/**
 * Nom d'affichage d'un joueur, GARANTI sans format xuid brut :
 *  - `gamertag` non vide et non xuid-like → `gamertag`, suffixe bot retiré
 *  - sinon `xuid` fourni → `maskedPlayerLabel(xuid)`
 *  - sinon → "Joueur inconnu"
 */
export function displayPlayerName(
  gamertag?: string | null,
  xuid?: string | null,
): string {
  const gt = gamertag?.trim()
  if (gt && !isXuidLike(gt)) return stripBotSuffix(gt)
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
