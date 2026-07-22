/**
 * ShareLinkButton — bouton icône « Copier le lien avec les filtres ».
 *
 * Le share-link `?f=` n'est plus écrit automatiquement dans l'URL : il est
 * produit À LA DEMANDE via `buildShareUrl` (action du store) au clic, puis copié
 * dans le presse-papier. Masqué si le store n'a pas le share-link activé
 * (`buildShareUrl()` → null, ex. store escouade).
 */
import { useState } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

export interface ShareLinkButtonProps {
  /** Action du store : construit l'URL de partage du contexte courant, ou `null`
   *  si le share-link est désactivé pour ce store. */
  buildShareUrl: () => string | null
}

export function ShareLinkButton({ buildShareUrl }: ShareLinkButtonProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const [copied, setCopied] = useState(false)

  // Masqué si le store n'expose pas de share-link (buildShareUrl null → escouade).
  if (buildShareUrl() === null) return null

  const handleCopy = async () => {
    const url = buildShareUrl()
    if (!url) return
    try {
      await navigator.clipboard.writeText(url)
      setCopied(true)
      // Feedback transitoire ~2 s puis retour à l'icône « lien ».
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // Silencieux : le presse-papier peut être indisponible (permissions).
    }
  }

  const label = copied
    ? t('common.filters.share_copied')
    : t('common.filters.share_copy')

  return (
    <button
      type="button"
      onClick={handleCopy}
      aria-label={label}
      title={label}
      className="shrink-0 inline-flex items-center justify-center rounded-md border border-input bg-background px-2 py-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
    >
      {copied ? <CheckIcon /> : <LinkIcon />}
    </button>
  )
}

// Icône « lien » (heroicons mini) — inline SVG, aligné sur le bouton « Voir les
// matchs » (le projet n'utilise pas lucide-react).
function LinkIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
      fill="currentColor"
      className="h-3.5 w-3.5 opacity-70"
      aria-hidden="true"
    >
      <path
        fillRule="evenodd"
        d="M8.914 6.025a.75.75 0 0 1 1.06 0 3.5 3.5 0 0 1 0 4.95l-2 2a3.5 3.5 0 0 1-4.95-4.95l1.25-1.25a.75.75 0 1 1 1.06 1.06l-1.25 1.25a2 2 0 1 0 2.83 2.83l2-2a2 2 0 0 0 0-2.83.75.75 0 0 1 0-1.06Z"
        clipRule="evenodd"
      />
      <path
        fillRule="evenodd"
        d="M7.086 9.975a.75.75 0 0 1-1.06 0 3.5 3.5 0 0 1 0-4.95l2-2a3.5 3.5 0 0 1 4.95 4.95l-1.25 1.25a.75.75 0 1 1-1.06-1.06l1.25-1.25a2 2 0 1 0-2.83-2.83l-2 2a2 2 0 0 0 0 2.83.75.75 0 0 1 0 1.06Z"
        clipRule="evenodd"
      />
    </svg>
  )
}

// Icône « coché » pour l'état transitoire après copie.
function CheckIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
      fill="currentColor"
      className="h-3.5 w-3.5 text-primary"
      aria-hidden="true"
    >
      <path
        fillRule="evenodd"
        d="M12.416 3.376a.75.75 0 0 1 .208 1.04l-5 7.5a.75.75 0 0 1-1.154.114l-3-3a.75.75 0 0 1 1.06-1.06l2.353 2.353 4.493-6.74a.75.75 0 0 1 1.04-.207Z"
        clipRule="evenodd"
      />
    </svg>
  )
}
