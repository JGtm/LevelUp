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
import { createFileRoute, Link, Outlet, useLocation, useRouter } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import {
  resolveTitleGate,
  useTitleConvergence,
  isKnownLocale,
  withLangSegment,
  type TitleGate,
} from '@/lib/title-routing'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug')({
  component: TitleLayout,
})

function TitleLayout() {
  const router = useRouter()
  const location = useLocation()
  const { titleSlug, lang } = Route.useParams()
  const isBootstrapped = useAppShellStore((s) => s.isBootstrapped)
  const availableTitles = useAppShellStore((s) => s.availableTitles)
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const isTitleSwitching = useAppShellStore((s) => s.isTitleSwitching)
  const locale = useAppShellStore((s) => s.locale)
  const setLocale = useAppShellStore((s) => s.setLocale)

  const gate = resolveTitleGate(titleSlug, availableTitles, isBootstrapped)
  const diverges = gate === 'valid' && titleSlug !== currentTitleSlug

  // Convergence D-6 : sur divergence segment↔store (titre valide mais ≠ courant), appliquer le titre
  // — cf. useTitleConvergence (gardes anti-double-bascule, chemin d'erreur complet).
  const { applyFailed, retry } = useTitleConvergence({ titleSlug, diverges, isTitleSwitching })

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

  // Émission du segment de langue par défaut (I10) — CHOKEPOINT central. Quand l'URL
  // title-scoped n'a PAS de segment `lang` (bookmark nu « /t/… », lien pleine page
  // media/data-quality, ancienne URL non préfixée), on réécrit l'URL en injectant la
  // locale de SESSION : « /t/… » devient « /{locale}/t/… » (replace). Le segment posé, TOUS les
  // Links/navigate title-scoped l'HÉRITENT ensuite (langSegmentInheritance.test) → la
  // langue reste visible pour toute la navigation, sans toucher aux ~100 call-sites.
  //
  // Préservation ?search/#hash : on reconstruit l'URL depuis `useLocation()` BRUT —
  // `searchStr` porte son `?`, `hash` EXCLUT son `#` (champ TanStack, cf. splat
  // routes/players/$.tsx) → on le réintroduit. Byte-identique pour l'enveloppe base64
  // `?f=` (share-links) : mécanique PROUVÉE identique au splat legacy.
  //
  // Gaté sur un titre VALIDE + convergé (jamais un gate d'indispo, une divergence en
  // cours, une bascule en vol, ni un échec de switch) : on n'injecte pas de segment
  // dans une URL de titre invalide. IDEMPOTENT : après le replace, `lang` est défini →
  // l'effet ressort (garde `lang` + garde d'égalité href) — aucune boucle.
  useEffect(() => {
    if (lang || gate !== 'valid' || diverges || isTitleSwitching || applyFailed) return
    const hash = location.hash ? `#${location.hash}` : ''
    const current = location.pathname + location.searchStr + hash
    const href = withLangSegment(location.pathname, locale) + location.searchStr + hash
    if (href !== current) router.history.replace(href)
  }, [
    lang,
    gate,
    diverges,
    isTitleSwitching,
    applyFailed,
    locale,
    location.pathname,
    location.searchStr,
    location.hash,
    router,
  ])

  // Projection DÉCLARATIVE (D-8) — cf. NOTE CRITIQUE ci-dessus.
  if (gate === 'wait') return null // pré-hydratation : ne rien décider
  if (gate === 'unknown' || gate === 'coming_soon' || gate === 'archived') {
    return <TitleGateScreen verdict={gate} />
  }
  if (applyFailed) {
    return <TitleGateScreen verdict="switch_failed" onRetry={retry} />
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
            <Link className="text-sm underline text-primary" to="/{-$lang}">
              {t('common.title_gate.back_to_active')}
            </Link>
          )}
        </div>
      </div>
    </div>
  )
}
