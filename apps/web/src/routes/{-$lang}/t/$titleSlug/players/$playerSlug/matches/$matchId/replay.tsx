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
import { useCallback, useMemo, useState } from 'react'

import { normalizeCallouts } from '@/features/match-replay/calloutsLayer'
import { REPLAY_TEXT } from '@/features/match-replay/i18n'
import {
  useMatchReplay,
  useReplayMapBackground,
  useReplayMapCallouts,
  useReplayMapImage,
} from '@/features/match-replay/queries'
import { collectMedalEvents } from '@/features/match-replay/killFeedLogic'
import { buildPlayerMarks } from '@/features/match-replay/playerMarks'
import { ReplayCanvas } from '@/features/match-replay/ReplayCanvas'
import { ReplayKillFeed } from '@/features/match-replay/ReplayKillFeed'
import { frameToMs } from '@/features/match-replay/replayLogic'
import { ReplayTeams } from '@/features/match-replay/ReplayTeams'
import { collectKillEvents } from '@/features/match-view/_momentum'
import { useMatchView } from '@/features/match-view/queries'
import type { TeamColorResolver } from '@/features/match-view/teamColor'
import { meXUIDOf, resolveXuidMeta } from '@/features/match-view/xuidMeta'
import { useSettings } from '@/features/settings/queries'
import { tokenCssVar } from '@/lib/accessibility'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'
import { themedIconSrc } from '@/lib/themedIcon'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

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
  // Le logo du rejeu est un raster à deux variantes (noire / blanche) : c'est le thème local
  // qui choisit, comme pour les autres icônes du dépôt (cf. lib/themedIcon.ts).
  const theme = useSettingsDraftStore((s) => s.localUiPrefs.theme)
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
  // LES ZONES NOMMÉES, normalisées UNE fois : la même liste sert le calque du canvas et
  // la zone courante des fiches. 404 = la carte n'en a pas (Forge) — liste vide, rien.
  const { data: calloutsEntry } = useReplayMapCallouts(playerSlug, matchId)
  const callouts = useMemo(() => normalizeCallouts(calloutsEntry), [calloutsEntry])

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
  // AMIS ET « MOI », UNE SEULE TABLE POUR LES TROIS PANNEAUX (décision D5). Les amis sont
  // ceux du COMPTE CONNECTÉ (`settings.friend_gamertags`), le « moi » est le joueur DE LA
  // PAGE (`is_me` du scoreboard) : deux notions distinctes, une seule grammaire à l'écran.
  // Mémoïsé sur la liste elle-même — la réponse des réglages est stable tant qu'on ne la
  // modifie pas, et la carte redessine soixante fois par seconde.
  const { data: settings } = useSettings()
  const friendGamertags = settings?.friend_gamertags
  const marks = useMemo(
    () => buildPlayerMarks(scoreboard, friendGamertags ?? []),
    [scoreboard, friendGamertags],
  )
  // LA PAGE PARLE D'UNE SEULE VOIX (décision D1) : sur le rejeu, les points, les titres de
  // colonnes et les noms du fil prennent les MÊMES tokens d'accessibilité — allié / adverse,
  // surchargeables par les réglages. La cascade d'identité du fil (couleur backend, puis
  // couleur officielle de l'équipe) reste celle de la Match View, qui ne change pas : c'est
  // pourquoi le résolveur est passé en prop plutôt qu'imposé dans le composant.
  const colorOfTeam = useCallback<TeamColorResolver>(
    (_teamID, ally) => tokenCssVar(ally ? 'team-ally' : 'team-enemy'),
    [],
  )
  // Les MÉDAILLES viennent des mêmes events (event_type `medal`), identité résolue
  // côté backend — même lecture unique, aucun appel de plus.
  const medalEvents = useMemo(
    () => collectMedalEvents(matchView?.combat_tab.highlight_events),
    [matchView?.combat_tab.highlight_events],
  )
  // Les deux horloges ne coïncident pas : cf. killFeedLogic.ts et header.t0_ms.
  const t0Ms = matchView?.header.t0_ms ?? 0
  const nowMs = data ? frameToMs(frame, data) : 0

  const hasReplay = !!data && data.tracks.length > 0

  return (
    <div className="space-y-4 p-6">
      <div className="flex items-center justify-between gap-2">
        <h1 className="flex items-center gap-2 text-lg font-semibold">
          <img src={themedIconSrc('replay', theme)} alt="" aria-hidden className="h-5 w-auto" />
          {t.title}
        </h1>
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
        /* UNE SEULE RANGÉE : LE FIL, LA CARTE, LES FICHES (demande utilisateur du 2026-08-16,
           décision D7). Rien n'est empilé — le regard balaie de l'événement (à gauche) au
           terrain (au centre) puis aux joueurs (à droite), sans qu'aucun panneau n'en masque
           un autre et sans défilement de page pour passer de l'un à l'autre.

           C'EST LA CARTE QUI IMPOSE LA HAUTEUR DE LA RANGÉE, et c'est la technique du POC qui
           l'obtient : les colonnes latérales sont `relative` et leur contenu `absolute inset-0`
           à partir de `xl`. Un contenu absolu ne participe pas au calcul de hauteur de son
           parent — un fil de 80 kills ou un BTB à 24 fiches ne peuvent donc PAS étirer la
           ligne. Ils défilent à l'intérieur de la hauteur que la carte a fixée (le défilement
           vit dans la carte, cf. 2.3 et 3.2).

           SOUS 1280 px (`xl`) la rangée n'a plus la place : la carte reprend toute la largeur
           et le couple fil + fiches passe dessous, côte à côte, chacun borné à 60 % de la
           hauteur d'écran. L'enveloppe qui les apparie devient `display: contents` à partir de
           `xl` — elle disparaît alors de la grille et rend ses deux enfants directement à la
           rangée, sans qu'un niveau de balise supplémentaire vienne s'intercaler. */
        <div className="grid gap-3 xl:grid-cols-[minmax(240px,300px)_minmax(0,1fr)_minmax(0,28rem)] xl:items-stretch">
          {/* `min-w-0` : sans lui, un contenu large ferait déborder la colonne au lieu de la
              contraindre — c'est la colonne que le ResizeObserver du canvas mesure. */}
          <section className="order-1 min-w-0 xl:order-2">
            <ReplayCanvas
              doc={data}
              locale={locale}
              kills={kills}
              t0Ms={t0Ms}
              onFrameChange={setFrame}
              background={mapBackground}
              callouts={callouts}
              scoreboard={scoreboard}
              xuidMeta={xuidMeta}
              marks={marks}
            />
          </section>
          <div className="order-2 grid grid-cols-2 gap-3 xl:contents">
            {/* LE FIL À GAUCHE DE LA CARTE (et non plus dessous). Il est PERMANENT (verdict
                user 2026-08-13) : tout ce qui est déjà survenu reste lisible, le plus récent
                en tête, et descendre va chercher les événements anciens. Le document lui donne
                le référentiel des pistes : lignes recalées sur les fins de vie (le même
                instant que le flash des fiches) et morts neutres. */}
            <aside className="relative min-h-[12rem] xl:order-1 xl:min-h-0">
              <div className="flex max-h-[60vh] flex-col xl:absolute xl:inset-0 xl:max-h-none">
                <ReplayKillFeed
                  kills={kills}
                  medals={medalEvents}
                  t0Ms={t0Ms}
                  nowMs={nowMs}
                  doc={data}
                  scoreboard={scoreboard}
                  xuidMeta={xuidMeta}
                  locale={locale}
                  marks={marks}
                  colorOf={colorOfTeam}
                />
              </div>
            </aside>
            {/* LES FICHES À DROITE DE LA CARTE, une colonne par équipe sous son nom résolu
                (Équipe Eagle / Équipe Cobra) : un rejeu se lit en balayant du terrain vers le
                joueur, sans que l'un masque l'autre. */}
            <aside className="relative min-h-[12rem] xl:order-3 xl:min-h-0">
              <div className="flex max-h-[60vh] flex-col xl:absolute xl:inset-0 xl:max-h-none">
                <ReplayTeams
                  doc={data}
                  scoreboard={scoreboard}
                  frame={frame}
                  locale={locale}
                  callouts={callouts}
                  marks={marks}
                  xuidMeta={xuidMeta}
                />
              </div>
            </aside>
          </div>
        </div>
      )}
    </div>
  )
}
