/**
 * TermsPage — conditions d'utilisation, route /terms.
 *
 * Accessible SANS être connecté, comme la politique de confidentialité : un visiteur doit
 * pouvoir lire ce à quoi il consent (en particulier la mise en commun des jetons) avant de
 * connecter son compte Microsoft. Le contenu vit dans `./termsI18n`.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { LegalDocument } from './LegalDocument'
import { getTermsText, TERMS_UPDATED_AT } from './termsI18n'

export function TermsPage() {
  const locale = useAppShellStore((s) => s.locale)
  return <LegalDocument text={getTermsText(locale)} updatedAt={TERMS_UPDATED_AT[locale]} />
}
