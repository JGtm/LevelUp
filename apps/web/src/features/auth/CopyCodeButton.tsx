/**
 * CopyCodeButton — bouton icône « Copier le code » de l'écran de connexion
 * Xbox (Device Code Flow). Sans texte visible : le libellé porte sur
 * aria-label/title, l'icône bascule sur une coche ~2 s après la copie.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { useCopyToClipboard } from '@/lib/clipboard/useCopyToClipboard'

export interface CopyCodeButtonProps {
  code: string
}

export function CopyCodeButton({ code }: CopyCodeButtonProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  // Mécanique « copier + coche transitoire » partagée (lib/clipboard). En cas
  // d'échec du presse-papier le code reste sélectionnable à la main (select-all).
  const { copy, copied } = useCopyToClipboard()

  const handleCopy = () => void copy(code)

  const label = copied ? t('common.auth.xbox_code_copied') : t('common.auth.xbox_copy_code')

  return (
    <button
      type="button"
      onClick={handleCopy}
      aria-label={label}
      title={label}
      className="shrink-0 inline-flex h-9 w-9 items-center justify-center rounded-md border border-input bg-background text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      {copied ? <CheckIcon /> : <ClipboardIcon />}
    </button>
  )
}

// Icône « copier » (deux feuilles superposées) — inline SVG, le projet
// n'utilise pas lucide-react.
function ClipboardIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      className="h-5 w-5"
      aria-hidden="true"
    >
      <rect x="9" y="9" width="12" height="12" rx="2" />
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
    </svg>
  )
}

function CheckIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2.5}
      strokeLinecap="round"
      strokeLinejoin="round"
      className="h-5 w-5 text-success"
      aria-hidden="true"
    >
      <path d="M20 6 9 17l-5-5" />
    </svg>
  )
}
