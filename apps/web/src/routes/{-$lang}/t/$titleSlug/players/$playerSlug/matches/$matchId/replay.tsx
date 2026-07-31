/**
 * Route /{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay — Rejeu 2D
 * du match (vue du dessus).
 *
 * Trajectoires des joueurs décodées du film, servies par l'API comme artefact
 * pré-construit (GET .../replay). La disponibilité est PAR MATCH : 404 → état vide.
 * Ce n'est donc pas un drapeau global — le stub `REJEU_2D_ENABLED` qui occupait cette
 * route est remplacé par l'implémentation réelle. La garde de capability est conservée :
 * sans `matchmaking`, le match n'existe pas pour ce titre, donc son rejeu non plus.
 */
import { createFileRoute, Link } from '@tanstack/react-router'
import { useState } from 'react'

import { REPLAY_TEXT } from '@/features/match-replay/i18n'
import { useMatchReplay } from '@/features/match-replay/queries'
import { ReplayCanvas } from '@/features/match-replay/ReplayCanvas'
import { ReplayCoverageBanner } from '@/features/match-replay/ReplayCoverage'
import { ReplayTeams } from '@/features/match-replay/ReplayTeams'
import { useMatchView } from '@/features/match-view/queries'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'
import { useAppShellStore } from '@/stores/appShellStore'

export const Route = createFileRoute(
  '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay',
)({
  component: () => (
    <RouteCapabilityGate capability="matchmaking">
      <ReplayPage />
    </RouteCapabilityGate>
  ),
})

function ReplayPage() {
  // Route.useParams() porte lang?/titleSlug/playerSlug/matchId (typés) — même forme
  // que la cible matches/$matchId, réutilisable tel quel (préserve titre ET langue).
  const params = Route.useParams()
  const { playerSlug, matchId } = params
  const locale = useAppShellStore((s) => s.locale)
  const t = REPLAY_TEXT[locale]
  const { data, isLoading } = useMatchReplay(playerSlug, matchId)
  // DEUX SOURCES, DEUX RÔLES. Le film porte ce qui se passe et l'identifie par XUID ; la
  // base porte qui sont les gens. L'artefact ne mélange pas les deux — la jointure se fait
  // ici, sur le xuid, qui est la seule clé qui ne suppose rien (surtout pas un ordre).
  const { data: matchView } = useMatchView(playerSlug, matchId)
  const [frame, setFrame] = useState(0)

  const hasReplay = !!data && data.tracks.length > 0

  return (
    <div className="space-y-4 p-6">
      <div className="flex items-center justify-between gap-2">
        <h1 className="text-lg font-semibold">{t.title}</h1>
        <Link
          to="/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId"
          params={params}
          className="inline-flex h-8 items-center justify-center gap-2 rounded-md border border-border bg-transparent px-3 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          {t.back}
        </Link>
      </div>

      {isLoading && <p className="text-sm text-muted-foreground">{t.loading}</p>}

      {!isLoading && !hasReplay && (
        <div className="rounded-lg border border-border bg-card px-3 py-12 text-center">
          <p className="text-sm text-muted-foreground">{t.empty}</p>
        </div>
      )}

      {data && data.tracks.length > 0 && (
        <>
          <ReplayCanvas doc={data} locale={locale} onFrameChange={setFrame} />
          {/* Les fiches vivent À CÔTÉ de la carte, jamais dessus : un rejeu se lit en balayant
              du terrain vers le joueur, sans que l'un masque l'autre. */}
          <ReplayTeams
            doc={data}
            scoreboard={matchView?.team_tab.scoreboard ?? []}
            frame={frame}
            locale={locale}
          />
          {/* CE QUI N'EST PAS À L'ÉCRAN doit être lisible à côté de ce qui y est : le rejeu
              publie 475 tirs sur les 519 que le film porte, et l'écart a sa cause nommée. */}
          <ReplayCoverageBanner doc={data} locale={locale} />
        </>
      )}
    </div>
  )
}
