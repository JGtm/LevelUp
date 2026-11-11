/**
 * MicrosoftNotice — mention imposée par les règles d'usage du contenu de jeu de Microsoft.
 *
 * Le texte de la mention reste en anglais dans toutes les langues (libellé imposé, repris mot
 * pour mot de `lib/appLinks`), sans phrase d'introduction ; seule l'étiquette du lien vers les
 * règles suit la langue de l'interface. Rendue dans le pied de page et sur la page de
 * téléchargement : visible de quiconque voit le contenu.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { MICROSOFT_CONTENT_NOTICE, XBOX_GAME_CONTENT_RULES_URL } from '@/lib/appLinks'

const LINK_CLASS =
  'text-muted-foreground underline underline-offset-2 transition-colors hover:text-foreground'

export function MicrosoftNotice() {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  return (
    <p data-testid="microsoft-notice">
      <span lang="en">{MICROSOFT_CONTENT_NOTICE}</span>{' '}
      <a href={XBOX_GAME_CONTENT_RULES_URL} target="_blank" rel="noopener noreferrer" className={LINK_CLASS}>
        {t('common.footer.microsoft_rules_link')}
      </a>
    </p>
  )
}
