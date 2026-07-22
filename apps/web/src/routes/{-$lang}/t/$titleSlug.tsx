/**
 * Layout titre — parent du sous-arbre title-scoped
 * `/{-$lang}/t/$titleSlug/players/…` (D-6, D-8).
 *
 * Rôle : PROJETER le verdict pur `resolveTitleGate(slug, availableTitles,
 * isBootstrapped)` — aucune logique métier ici (la décision vit dans le résolveur
 * déjà testé, le composant ne fait que la rendre déclarativement).
 *
 * NOTE CRITIQUE (fenêtre pré-hydratation). `__root` NE rend PAS de loader tant que
 * le store n'est pas hydraté : il rend l'`<Outlet />` NU (__root.tsx:175-177). Ce
 * layout est donc le SEUL rempart qui ferme la fenêtre « fetch avec le mauvais
 * titre/clé » pour tout le sous-arbre joueur. Règle absolue : ne JAMAIS rendre
 * l'Outlet si le verdict n'est pas `valid`, si le segment diverge du titre courant,
 * ou si une bascule est en vol (`isTitleSwitching`).
 */
import { createFileRoute, Link, Outlet } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { applyActiveTitle, resolveTitleGate, type TitleGate } from '@/lib/title-routing'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug')({
  component: TitleLayout,
})

function TitleLayout() {
  const { titleSlug } = Route.useParams()
  const isBootstrapped = useAppShellStore((s) => s.isBootstrapped)
  const availableTitles = useAppShellStore((s) => s.availableTitles)
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const isTitleSwitching = useAppShellStore((s) => s.isTitleSwitching)
  const [applyFailed, setApplyFailed] = useState(false)

  const gate = resolveTitleGate(titleSlug, availableTitles, isBootstrapped)
  const diverges = gate === 'valid' && titleSlug !== currentTitleSlug

  // Convergence D-6 : sur divergence segment↔store (titre valide mais ≠ courant),
  // appliquer le titre. Gardes anti-double-bascule : ni si une bascule est déjà en
  // vol (isTitleSwitching, y compris déclenchée par le bouton), ni si une bascule
  // vient d'échouer (applyFailed → l'utilisateur réessaie explicitement).
  // applyingRef dédoublonne en plus les ré-entrées d'un même commit (StrictMode
  // dev). À la complétion réussie, currentTitleSlug converge → l'effet re-run et
  // sort ; à l'échec, applyFailed bascule l'écran gate en variante « bascule
  // impossible » (le chemin d'erreur complet — navigate retour segment précédent +
  // toast — est câblé en Phase 4b du plan ; ici : réessai in-place).
  const applyingRef = useRef(false)
  useEffect(() => {
    if (!diverges || isTitleSwitching || applyFailed || applyingRef.current) return
    applyingRef.current = true
    applyActiveTitle(titleSlug)
      .catch(() => setApplyFailed(true))
      .finally(() => {
        applyingRef.current = false
      })
  }, [diverges, titleSlug, isTitleSwitching, applyFailed])

  // Projection DÉCLARATIVE (D-8) — cf. NOTE CRITIQUE ci-dessus.
  if (gate === 'wait') return null // pré-hydratation : ne rien décider
  if (gate === 'unknown' || gate === 'coming_soon' || gate === 'archived') {
    return <TitleGateScreen verdict={gate} />
  }
  if (applyFailed) {
    return <TitleGateScreen verdict="switch_failed" onRetry={() => setApplyFailed(false)} />
  }
  // gate === 'valid' : ne rendre le sous-arbre QUE si le titre a convergé ET
  // qu'aucune bascule n'est en vol (sinon on servirait un rendu transitoire avec
  // le cache/le header du mauvais titre — la fenêtre que ce layout doit fermer).
  if (diverges || isTitleSwitching) return null
  return <Outlet />
}

/** Verdicts qui donnent lieu à un écran (les autres — wait/valid — ne rendent rien). */
type GateVerdict = Exclude<TitleGate, 'wait' | 'valid'> | 'switch_failed'

const GATE_HEADING: Record<GateVerdict, CommonManifestKey> = {
  unknown: 'common.title_gate.unknown_heading',
  coming_soon: 'common.title_gate.coming_soon_heading',
  archived: 'common.title_gate.archived_heading',
  switch_failed: 'common.title_gate.switch_failed_heading',
}

const GATE_BODY: Record<GateVerdict, CommonManifestKey> = {
  unknown: 'common.title_gate.unknown_body',
  coming_soon: 'common.title_gate.coming_soon_body',
  archived: 'common.title_gate.archived_body',
  switch_failed: 'common.title_gate.switch_failed_body',
}

/**
 * Écran d'indisponibilité title-scoped (parité RequireActiveTitle). Message par
 * verdict + retour au titre actif (home du joueur courant, ou index si aucun
 * joueur). En variante `switch_failed`, ajoute un bouton « Réessayer » (reset de
 * l'état d'échec côté layout). Strings via manifest i18n (FR + EN).
 */
function TitleGateScreen({ verdict, onRetry }: { verdict: GateVerdict; onRetry?: () => void }) {
  const locale = useAppShellStore((s) => s.locale)
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  return (
    <div className="flex h-screen items-center justify-center">
      <div className="text-center space-y-3">
        <p className="font-semibold text-foreground">{t(GATE_HEADING[verdict])}</p>
        <p className="text-sm text-muted-foreground">{t(GATE_BODY[verdict])}</p>
        <div className="flex items-center justify-center gap-4">
          {onRetry && (
            <button className="text-sm underline text-primary" onClick={onRetry}>
              {t('common.title_gate.retry')}
            </button>
          )}
          {currentPlayer?.player_slug ? (
            <Link
              className="text-sm underline text-primary"
              to="/{-$lang}/t/$titleSlug/players/$playerSlug/home"
              params={{ titleSlug: currentTitleSlug, playerSlug: currentPlayer.player_slug }}
            >
              {t('common.title_gate.back_to_active')}
            </Link>
          ) : (
            <Link className="text-sm underline text-primary" to="/">
              {t('common.title_gate.back_to_active')}
            </Link>
          )}
        </div>
      </div>
    </div>
  )
}
