/**
 * PrivacyPage — politique de confidentialité, route /privacy.
 *
 * Accessible SANS être connecté (cf. la liste des chemins anonymes de
 * `routes/__root.tsx`) : le pied de page de l'écran de connexion y renvoie, et
 * un visiteur qui hésite à connecter son compte Microsoft doit pouvoir lire
 * cette page avant de le faire.
 *
 * Le contenu vit dans `./i18n` — s'y référer avant toute modification de fond.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { LegalDocument } from './LegalDocument'
import { getPrivacyText, PRIVACY_UPDATED_AT } from './i18n'

export function PrivacyPage() {
  const locale = useAppShellStore((s) => s.locale)
  return <LegalDocument text={getPrivacyText(locale)} updatedAt={PRIVACY_UPDATED_AT[locale]} />
}
