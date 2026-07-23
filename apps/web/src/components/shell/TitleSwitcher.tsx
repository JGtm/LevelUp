import { useNavigate } from '@tanstack/react-router'
import { useAppShellStore, buildTitleSwitcherEntries } from '@/stores/appShellStore'
import { applyActiveTitle } from '@/lib/title-routing/applyActiveTitle'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { formatMessage } from '@/lib/i18n/format'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { log } from '@/components/shell/_logger'

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
 * `archived` est exclu. NAVIGATE-FIRST (D-6, Phase 4a) : le clic NAVIGUE vers le
 * segment du titre cible ; le layout `t/$titleSlug` exécute la bascule
 * (applyActiveTitle) sur divergence segment↔store. `buildTitleSwitcherEntries`
 * construit la liste — aucune bascule pilotée depuis ce composant.
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

  const entries = buildTitleSwitcherEntries(availableTitles, currentTitleSlug)
  if (entries.length <= 1) return null

  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const onSelect = (slug: string, disabled: boolean, isCurrent: boolean) => {
    if (disabled || isCurrent || isTitleSwitching) return
    // NAVIGATE-FIRST (D-6, inversion de contrôle finale — Phase 4a). On ne bascule
    // PLUS ici : on navigue vers le SEGMENT du titre cible, et le layout
    // `t/$titleSlug` détecte la divergence segment↔store puis exécute
    // applyActiveTitle. Le playerSlug visé est celui du titre COURANT (joueur
    // courant, sinon 1er disponible) : VOULU — si ce joueur n'existe pas sur le
    // titre cible, le filet resolvePlayerFallback (PlayerLayout) re-cible au
    // fresh-load post-bascule.
    const state = useAppShellStore.getState()
    const playerSlug = state.currentPlayer?.player_slug ?? state.availablePlayers[0]?.player_slug
    if (playerSlug) {
      void navigate({
        to: '/{-$lang}/t/$titleSlug/players/$playerSlug/home',
        params: { titleSlug: slug, playerSlug },
      })
      onSwitched?.()
      return
    }
    // Cas limite : AUCUN joueur (availablePlayers vide, aucun joueur courant) → il
    // n'existe pas de route de titre nue (segment sans sous-arbre joueur) à cibler
    // par l'URL. Fallback DOCUMENTÉ :
    // basculer DIRECTEMENT via applyActiveTitle (hors chemin URL, faute de
    // destination joueur) puis renvoyer à l'index (qui gère l'onboarding et
    // re-résout le titre actif). Le menu se ferme quoi qu'il arrive.
    void (async () => {
      try {
        await applyActiveTitle(slug)
      } catch (err) {
        log.warn(
          'title_switch:direct_fallback_failed',
          'bascule directe (aucun joueur) échouée',
          err,
        )
      } finally {
        onSwitched?.()
        void navigate({ to: '/' })
      }
    })()
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
