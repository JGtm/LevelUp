/**
 * Liens externes du projet — source UNIQUE.
 *
 * Le slug du dépôt était dupliqué (feedback-drawer + pied de page) : il est
 * centralisé ici. Garde-rail associé : `appLinks.guard.test.ts` interdit la
 * réapparition du littéral du dépôt ailleurs dans `src/`.
 *
 * Ces URL sont des identités de compte (dépôt, mécénat) : elles ne changent pas
 * par environnement, donc pas de variable d'environnement Vite ici.
 */
import type { Locale } from '@/lib/i18n/locale'

/** Dépôt GitHub du projet, au format `owner/name`. */
export const GITHUB_REPO = 'JGtm/LevelUp'

export const GITHUB_URL = `https://github.com/${GITHUB_REPO}`

/** Base des issues — le feedback-drawer y ajoute `/new` + les paramètres. */
export const GITHUB_ISSUES_URL = `${GITHUB_URL}/issues`

export const GITHUB_LICENSE_URL = `${GITHUB_URL}/blob/main/LICENSE`

/** Profil GitHub de l'auteur du projet. */
export const GITHUB_PROFILE_URL = 'https://github.com/JGtm'

/** Mécénat récurrent (programme GitHub Sponsors du compte propriétaire). */
export const SPONSORS_URL = 'https://github.com/sponsors/JGtm'

/** Don ponctuel. */
export const PAYPAL_URL = 'https://paypal.me/gsitbon'

/**
 * CSinsight — plateforme de statistiques CS2 d'un proche du projet. Le site
 * sert sa langue par le premier segment de chemin (`/fr`, `/en`) : on suit la
 * locale de l'app plutôt que de figer une langue.
 */
const CSINSIGHT_ORIGIN = 'https://csinsight.eu'

export function csinsightUrl(locale: Locale): string {
  return `${CSINSIGHT_ORIGIN}/${locale}`
}

/**
 * Adresse de contact du projet, publiée sur la page /privacy.
 *
 * ADRESSE DE RÔLE sur le domaine du projet, jamais la boîte personnelle de
 * l'auteur. Trois propriétés qui tiennent le spam à distance :
 *   - elle se filtre par DESTINATAIRE côté messagerie : une règle « pour: cette
 *     adresse » suffit à la sortir de la boîte principale ;
 *   - elle se REMPLACE en une manipulation — nouvelle adresse chez l'hébergeur
 *     mail, une seule ligne à changer ici ;
 *   - ce n'est pas un compte : rien à perdre si elle finit sur une liste.
 *
 * Assemblée à l'exécution, pièce par pièce : le littéral complet n'apparaît ni
 * dans le HTML servi (l'app est rendue côté navigateur) ni d'un seul tenant
 * dans le bundle. Ça n'arrête pas un moissonneur qui exécute le JavaScript,
 * ça écarte tous les autres — et ça ne coûte rien.
 */
const PRIVACY_CONTACT_PARTS = ['contact', 'lvelup.info'] as const

export function privacyContactEmail(): string {
  return PRIVACY_CONTACT_PARTS.join('@')
}
