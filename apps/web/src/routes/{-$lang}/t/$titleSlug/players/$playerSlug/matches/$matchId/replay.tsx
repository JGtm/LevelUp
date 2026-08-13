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
import { useMemo, useState } from 'react'

import { REPLAY_TEXT } from '@/features/match-replay/i18n'
import {
  useMatchReplay,
  useReplayMapBackground,
  useReplayMapImage,
} from '@/features/match-replay/queries'
import { ReplayCanvas } from '@/features/match-replay/ReplayCanvas'
import { ReplayKillFeed } from '@/features/match-replay/ReplayKillFeed'
import { frameToMs } from '@/features/match-replay/replayLogic'
import { ReplayTeams } from '@/features/match-replay/ReplayTeams'
import { collectKillEvents } from '@/features/match-view/_momentum'
import { useMatchView } from '@/features/match-view/queries'
import { meXUIDOf, resolveXuidMeta } from '@/features/match-view/xuidMeta'
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

  // LE FOND DE CARTE, en deux temps assumés : le CALAGE d'abord (quelques centaines
  // d'octets, il dit si la carte a une image et où elle se pose), l'IMAGE ensuite —
  // jusqu'à 1,4 Mio, qu'on ne va chercher que si le calage existe.
  const { data: background } = useReplayMapBackground(playerSlug, matchId)
  const mapImage = useReplayMapImage(playerSlug, matchId, !!background)
  const mapBackground = useMemo(
    () => (background && mapImage ? { calibration: background.calibration, image: mapImage } : null),
    [background, mapImage],
  )

  // LE KILL FEED VIENT DE LA BASE, PAS DU FILM. Le rejeu ne porte pas les kills ; la Match
  // View, elle, les sert déjà résolus (auteur, équipe, ARME du kill avec son icône). On
  // réutilise donc sa lecture — aucun appel de plus, aucun artefact à reconstruire.
  // Mémoïsé : `?? []` fabrique un tableau neuf à chaque rendu, ce qui reconstruirait
  // l'index des joueurs et la liste des kills soixante fois par seconde de lecture.
  const scoreboard = useMemo(() => matchView?.team_tab.scoreboard ?? [], [matchView])
  const xuidMeta = useMemo(
    () => resolveXuidMeta(scoreboard, meXUIDOf(scoreboard)),
    [scoreboard],
  )
  const kills = useMemo(
    () => collectKillEvents(matchView?.combat_tab.highlight_events, xuidMeta),
    [matchView?.combat_tab.highlight_events, xuidMeta],
  )
  // Les deux horloges ne coïncident pas : cf. killFeedLogic.ts et header.t0_ms.
  const t0Ms = matchView?.header.t0_ms ?? 0
  const nowMs = data ? frameToMs(frame, data) : 0

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
          <ReplayCanvas
            doc={data}
            locale={locale}
            onFrameChange={setFrame}
            background={mapBackground}
          />
          {/* LE KILL FEED SOUS LA CARTE, jamais dessus : même règle que les fiches. Il
              n'affiche que ce qui vient de se passer à l'instant du rejeu — l'histoire
              complète du match est déjà racontée par la carte « Dominance ». */}
          <ReplayKillFeed
            kills={kills}
            victims={matchView?.combat_tab.killer_victim}
            t0Ms={t0Ms}
            nowMs={nowMs}
            scoreboard={scoreboard}
            xuidMeta={xuidMeta}
            locale={locale}
          />
          {/* Les fiches vivent À CÔTÉ de la carte, jamais dessus : un rejeu se lit en balayant
              du terrain vers le joueur, sans que l'un masque l'autre. */}
          <ReplayTeams
            doc={data}
            scoreboard={scoreboard}
            frame={frame}
            locale={locale}
          />
        </>
      )}
    </div>
  )
}
