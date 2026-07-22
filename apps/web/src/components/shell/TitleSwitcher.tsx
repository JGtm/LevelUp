import { useNavigate } from '@tanstack/react-router'
import { useAppShellStore, buildTitleSwitcherEntries } from '@/stores/appShellStore'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { formatMessage } from '@/lib/i18n/format'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

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
  const navigate = useNavigate()
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
    void switchTitle(slug).then(() => {
      onSwitched?.()
      // Le switch a (re-)bootstrap les joueurs du NOUVEAU titre. L'URL pointe
      // encore sur le playerSlug de l'ANCIEN titre → page morte. On navigue vers
      // l'accueil du joueur courant fraîchement résolu (lu APRÈS le switch). Si
      // le switch a échoué (rollback : titre inchangé), on ne navigue pas.
      const state = useAppShellStore.getState()
      if (state.currentTitleSlug !== slug) return
      const targetSlug = state.currentPlayer?.player_slug ?? state.availablePlayers[0]?.player_slug
      if (targetSlug) {
        void navigate({ to: '/{-$lang}/t/$titleSlug/players/$playerSlug/home', params: { titleSlug: slug, playerSlug: targetSlug } })
      } else {
        // Aucun joueur pour ce titre → retour à l'index (qui gère l'onboarding).
        void navigate({ to: '/' })
      }
    })
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
              // 3 états : actif (emphase token primary, pas de hover — déjà sélectionné)
              // / normal (hover accent) ; le disabled (coming_soon) garde son opacité.
              className={`flex items-center justify-between gap-3 rounded px-1.5 py-1 text-left text-sm disabled:cursor-not-allowed disabled:opacity-60 ${
                e.isCurrent
                  ? 'bg-primary/10 font-semibold text-primary'
                  : 'text-popover-foreground hover:bg-accent hover:text-accent-foreground'
              }`}
            >
              <span className="flex items-center gap-1.5">
                {e.isCurrent && (
                  // Pastille « actif » : indicateur non-couleur (a11y) en plus de
                  // l'emphase token, conforme à aria-checked. Couleur verte via le
                  // token sémantique `success` (jeu actif).
                  <span
                    className="size-1.5 shrink-0 rounded-full"
                    style={{ backgroundColor: tokenCssVar('success') }}
                    aria-hidden="true"
                  />
                )}
                {e.name}
              </span>
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
