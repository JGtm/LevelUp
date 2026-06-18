import { useAppShellStore, buildTitleSwitcherEntries } from '@/stores/appShellStore'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { formatMessage } from '@/lib/i18n/format'

interface TitleSwitcherProps {
  /** Appelé après le déclenchement d'un switch (ex: fermer le menu parent). */
  onSwitched?: () => void
}

/**
 * TitleSwitcher — sélecteur de jeu (titre) dans le menu Paramètres de la NavL1
 * (PMT-8 / MT-22).
 *
 * NO-OP mono-titre : ne rend RIEN si moins de 2 titres sont listés (déploiement
 * Halo seul → aucun changement visuel). Apparaît automatiquement dès qu'un 2e
 * titre est présent dans la config (`bootstrap.available_titles`). Un titre
 * `coming_soon` est listé mais désactivé (« Bientôt disponible ») ; un titre
 * `archived` est exclu. Branche la plomberie existante (`store.switchTitle` +
 * `buildTitleSwitcherEntries`) — aucune logique nouvelle.
 *
 * Rend, quand il s'affiche, un séparateur de fin (fragment) pour s'insérer
 * proprement avant la ligne suivante du menu sans laisser de séparateur orphelin
 * en mono-titre.
 */
export function TitleSwitcher({ onSwitched }: TitleSwitcherProps) {
  const locale = useAppShellStore((s) => s.locale)
  const availableTitles = useAppShellStore((s) => s.availableTitles)
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const isTitleSwitching = useAppShellStore((s) => s.isTitleSwitching)
  const switchTitle = useAppShellStore((s) => s.switchTitle)

  const entries = buildTitleSwitcherEntries(availableTitles, currentTitleSlug)
  if (entries.length <= 1) return null

  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const onSelect = (slug: string, disabled: boolean, isCurrent: boolean) => {
    if (disabled || isCurrent || isTitleSwitching) return
    void switchTitle(slug).then(() => onSwitched?.())
  }

  return (
    <>
      <div role="group" aria-label={t('common.shell.nav_game')} className="px-3 py-1.5">
        <span className="block text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {t('common.shell.nav_game')}
        </span>
        <div className="mt-1 flex flex-col gap-0.5">
          {entries.map((e) => (
            <button
              key={e.slug}
              type="button"
              role="menuitemradio"
              aria-checked={e.isCurrent}
              disabled={e.disabled || isTitleSwitching}
              onClick={() => onSelect(e.slug, e.disabled, e.isCurrent)}
              className={`flex items-center justify-between gap-3 rounded px-1.5 py-1 text-left text-sm text-popover-foreground hover:bg-accent hover:text-accent-foreground disabled:cursor-not-allowed disabled:opacity-60 ${
                e.isCurrent ? 'font-semibold' : ''
              }`}
            >
              <span>{e.name}</span>
              {e.disabled && (
                <span className="text-xs text-muted-foreground">
                  {t('common.shell.title_coming_soon')}
                </span>
              )}
            </button>
          ))}
        </div>
      </div>
      <div role="separator" className="my-1 h-px bg-border" />
    </>
  )
}
