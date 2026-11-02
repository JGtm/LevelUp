/**
 * MediaNoEncoderNotice — bandeau de la page Médias quand le serveur n'a aucun encodeur H.264 utilisable
 * (`media_h264_missing` de /bootstrap : édition Windows N ou KN sans Media Foundation). Il dit pourquoi un
 * clip que le navigateur ne lit pas tels quels ne se convertit pas, et pointe le « Media Feature Pack » de
 * Microsoft. Ne rend rien quand un encodeur existe.
 */
import { Notice } from '@/components/ui/notice'
import { MEDIA_FEATURE_PACK_URL } from '@/lib/appLinks'
import { formatMessage } from '@/lib/i18n/format'
import { mediaManifest, type MediaManifestKey } from '@/lib/i18n/generated/media'
import { useAppShellStore } from '@/stores/appShellStore'

export function MediaNoEncoderNotice() {
  const missing = useAppShellStore((s) => s.mediaH264Missing)
  const locale = useAppShellStore((s) => s.locale)
  if (!missing) return null
  const t = (key: MediaManifestKey) => formatMessage(mediaManifest, key, locale)
  return (
    <div className="px-6">
      <Notice tone="warning" title={t('media.no_h264.title')} testId="media-no-h264">
        <p>{t('media.no_h264.body')}</p>
        <p className="mt-1">
          <a
            href={MEDIA_FEATURE_PACK_URL}
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary underline"
          >
            {t('media.no_h264.link')}
          </a>
        </p>
      </Notice>
    </div>
  )
}
