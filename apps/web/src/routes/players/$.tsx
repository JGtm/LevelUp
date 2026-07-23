/**
 * Splat legacy `/players/$` — redirige toute ancienne URL `/players/…` (bookmark,
 * lien externe partagé, `notif.target_route` encore émis par le backend) vers sa
 * forme title-préfixée /t/{slug}/players/… . D-5 (préservation intégrale du suffixe
 * + `?search` + `#hash`, aucun lien mort) et D-8 (mécanisme DÉCLARATIF composant,
 * JAMAIS `beforeLoad`).
 *
 * Pourquoi pas un `beforeLoad` : le bootstrap est composant-level (`__root.tsx:34-49`)
 * et `__root.tsx:175-177` rend l'`<Outlet />` NU tant que `!isBootstrapped`. Un
 * `beforeLoad` s'exécuterait AVANT hydratation du store → il verrait `currentTitleSlug`
 * = défaut et redirigerait un bookmark H5 vers /t/halo_infinite/… (trou n°1 de D-8).
 *
 * GATE `isBootstrapped` : tant que le store n'est pas hydraté, on rend `null` et on
 * ATTEND la session serveur (qui fixe le titre actif). Rediriger maintenant enverrait
 * un bookmark `/players/…` généré pour halo_5 vers le titre par défaut. Une fois
 * bootstrappé, `currentTitleSlug` est autoritaire et `buildLegacyRedirect` projette la
 * bonne cible /t/halo_5/… (et non le titre par défaut).
 *
 * PROJECTION (D-8) : la DÉCISION vit dans `buildLegacyRedirect` — PURE et 100% testée
 * (matrice table-driven `buildLegacyRedirect.test.ts`). Ce composant ne fait que la
 * PROJETER, il ne décide rien. Source de l'URL = `useLocation()` BRUT : `pathname`,
 * `searchStr` (avec `?`) et `hash` (le champ TanStack exclut le `#`, réintroduit par
 * `buildLegacyRedirect`). On prend `searchStr`, JAMAIS l'objet parsé `location.search` :
 * vérifié sur pièces que le round-trip du sérialiseur PAR DÉFAUT du routeur (le projet
 * n'en configure aucun) est byte-identique au navigateur, y compris pour l'enveloppe
 * base64 `?f=` (PR #59) — `?f=` préservé verbatim.
 *
 * FORME (D-8) : le href retourné est COMPLET (forme /t/{slug}/players/…?f=…#…), appliqué
 * via `router.history.replace` — un `<Navigate to>` typé n'accepte pas un href
 * arbitraire complet sans re-découper `?`/`#` (double-parsing) ni cast. `replace` (pas
 * `push`) : une URL legacy ne doit pas polluer l'historique. `buildLegacyRedirect` →
 * `null` (pathname hors matrice, ex. `/players` nu) → `<Navigate to="/" replace />`
 * typé (retour index).
 *
 * COURSE POST-REPLACE (critique, corrigée) : `history.replace` ne démonte PAS le splat
 * synchroniquement — il re-rend d'abord avec la NOUVELLE location /t/… . Sur cette
 * location transitoire `buildLegacyRedirect` renvoie `null` (pathname hors /players) ;
 * SANS garde, la branche finale « → index » s'exécuterait et ÉCRASERAIT la bonne
 * redirection (bug observé : suffixe + ?f= + #hash perdus, atterrissage sur home via /).
 * Le prédicat `isLegacyPath` réserve donc les DEUX branches de décision (replace ET
 * index) au SEUL pathname réellement legacy ; toute autre location (transition) → `null`,
 * on ne décide rien et on laisse le démontage arriver.
 */
import { createFileRoute, Navigate, useLocation, useRouter } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { buildLegacyRedirect } from '@/lib/title-routing'

export const Route = createFileRoute('/players/$')({
  component: LegacyPlayersRedirect,
})

function LegacyPlayersRedirect() {
  const isBootstrapped = useAppShellStore((s) => s.isBootstrapped)
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const location = useLocation()
  const router = useRouter()

  const pathname = location.pathname
  // Le splat n'est PAS démonté synchroniquement par le replace : il re-rend d'abord
  // avec la NOUVELLE location /t/… (transition en vol, cf. en-tête COURSE POST-REPLACE).
  // isLegacyPath distingue le pathname RÉELLEMENT legacy de cette location transitoire —
  // sans quoi la branche finale « → index » écraserait la redirection correcte.
  const isLegacyPath = pathname === '/players' || pathname.startsWith('/players/')

  // Décision PURE (D-5), gatée bootstrap (trou n°1 D-8) ET pathname legacy.
  const redirect =
    isBootstrapped && isLegacyPath
      ? buildLegacyRedirect(pathname, location.searchStr, location.hash, currentTitleSlug)
      : null
  const href = redirect?.href ?? null

  // Projection de la décision pure : appliquer le href complet. L'effet ne tourne
  // qu'une fois par href (deps), puis le composant démonte (nouvelle route matchée).
  useEffect(() => {
    if (href) router.history.replace(href)
  }, [href, router])

  if (!isBootstrapped) return null // pré-hydratation : attendre la session (cf. en-tête)
  if (!isLegacyPath) return null // location transitoire post-replace (/t/…) : ne rien décider
  if (href) return null // l'effet redirige — ne rien rendre
  return <Navigate to="/" replace /> // /players nu / hors matrice legacy → retour index
}
