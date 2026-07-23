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
import { createFileRoute, Link, Outlet, useNavigate } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import { useAppShellStore } from '@/stores/appShellStore'
import { applyActiveTitle, resolveTitleGate, isKnownLocale, type TitleGate } from '@/lib/title-routing'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug')({
  component: TitleLayout,
})

function TitleLayout() {
  const navigate = useNavigate()
  const { titleSlug, lang } = Route.useParams()
  const isBootstrapped = useAppShellStore((s) => s.isBootstrapped)
  const availableTitles = useAppShellStore((s) => s.availableTitles)
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const isTitleSwitching = useAppShellStore((s) => s.isTitleSwitching)
  const locale = useAppShellStore((s) => s.locale)
  const setLocale = useAppShellStore((s) => s.setLocale)
  const [applyFailed, setApplyFailed] = useState(false)

  const gate = resolveTitleGate(titleSlug, availableTitles, isBootstrapped)
  const diverges = gate === 'valid' && titleSlug !== currentTitleSlug

  // Convergence D-6 : sur divergence segment↔store (titre valide mais ≠ courant),
  // appliquer le titre. Gardes anti-double-bascule : ni si une bascule est déjà en
  // vol (isTitleSwitching, y compris déclenchée par une ré-entrée), ni si une
  // bascule vient d'échouer SANS joueur courant (applyFailed → écran de réessai).
  // applyingRef dédoublonne les ré-entrées d'un même commit (StrictMode dev). À la
  // complétion réussie, currentTitleSlug converge → l'effet re-run et sort.
  //
  // Chemin d'erreur D-6 COMPLET (Phase 4b) : en échec d'applyActiveTitle, PAS de
  // rollback store-only (URL=cible, store=courant → divergence → boucle de
  // réconciliation). À la place :
  //  - joueur courant présent → toast + navigate REPLACE vers le titre COURANT (non
  //    basculé). Le segment redevient == store → plus de divergence → aucune
  //    re-tentative (anti-boucle). On NE pose PAS applyFailed : ce layout reste
  //    MONTÉ au changement de segment (param $titleSlug, même instance de route) —
  //    un applyFailed persistant bloquerait alors un futur switch légitime.
  //  - aucun joueur courant → pas de route joueur cible à viser : on conserve
  //    l'écran switch_failed (retry + retour manuel), fallback inchangé.
  const applyingRef = useRef(false)
  useEffect(() => {
    // Convergence ROBUSTE (course back/forward, 4c) : applyingRef ne sert qu'à
    // dédoublonner la double-invocation SYNCHRONE de StrictMode (dev). Il faut le
    // relâcher dès qu'aucune bascule n'est réellement en vol — sinon, quand une
    // bascule se termine (isTitleSwitching repasse false) ALORS qu'une divergence
    // subsiste (segment changé pendant le vol via popstate/refocus §7), l'effet
    // re-déclenché par le re-render synchrone du store verrait un applyingRef encore
    // armé (relâché seulement dans le .finally, plus tard) → la convergence CALERAIT.
    // On lit l'état LIVE du store (pas la closure) : après le 1er appel,
    // applyActiveTitle a posé isTitleSwitching=true de façon synchrone → la 2e
    // invocation StrictMode le voit true et NE relâche PAS le verrou (dédup préservée).
    if (!useAppShellStore.getState().isTitleSwitching) applyingRef.current = false
    if (!diverges || isTitleSwitching || applyFailed || applyingRef.current) return
    applyingRef.current = true
    applyActiveTitle(titleSlug)
      .catch(() => {
        const s = useAppShellStore.getState()
        const playerSlug = s.currentPlayer?.player_slug
        if (playerSlug) {
          toast.error(
            formatMessage(commonManifest, 'common.title_gate.switch_failed_toast', s.locale),
          )
          void navigate({
            to: '/{-$lang}/t/$titleSlug/players/$playerSlug/home',
            params: { titleSlug: s.currentTitleSlug, playerSlug },
            replace: true,
          })
        } else {
          setApplyFailed(true)
        }
      })
      .finally(() => {
        applyingRef.current = false
      })
  }, [diverges, titleSlug, isTitleSwitching, applyFailed, navigate])

  // Réconciliation locale←segment (D-12, Phase 5a) — effet SÉPARÉ de la convergence
  // titre (responsabilités indépendantes). Segment `lang` présent (locale connue) ET
  // ≠ locale du store → FORCE la locale via setLocale (setApiLocale pousse le header
  // X-LevelUp-Locale + set store). Segment ABSENT (lang undefined) OU déjà aligné →
  // NO-OP STRICT : la locale session/bootstrap reste inchangée (comportement actuel).
  // Un segment non-locale (ex. /xyz/t/…) est ignoré (isKnownLocale) : le gate titre
  // traite les URLs aberrantes, la locale ne bascule pas sur du bruit.
  //
  // Invalidation (audit D-12) : TOUTES les clés de payloads localisés portent `locale`
  // DANS leur clé (home, seasonPass, teammates, sessionDetail, field-mappings,
  // releaseNotes, leaderboard-catalog) → le set store re-clé et TanStack refetch tout
  // seul : la CLÉ EST l'invalidation, jamais de queryClient.clear() global ni
  // d'invalidation ciblée depuis cet effet.
  //
  // Ordre au fresh-load /en/t/… : initTitleFromLocation pose déjà le header 'en' au
  // boot ; hydrateFromBootstrap ré-écrit ensuite la locale SESSION (ex. fr) → cet
  // effet re-run post-hydratation et re-force 'en'. Convergence en un cycle (lang ==
  // locale → l'effet ressort), même patron déclaratif que la convergence titre ; une
  // ré-hydratation ultérieure (refocus) est ré-absorbée de la même façon.
  useEffect(() => {
    if (!lang || !isKnownLocale(lang) || lang === locale) return
    setLocale(lang)
  }, [lang, locale, setLocale])

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
