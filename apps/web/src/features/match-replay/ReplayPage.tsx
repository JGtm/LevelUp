/**
 * ReplayPage — la page du Rejeu 2D d'un match (vue du dessus) : chargement de l'artefact, du contexte du
 * match et du fond de carte, jointure sur le xuid, puis choix du cadre (Standard, Cinéma, Plein cadre).
 *
 * La ROUTE (`routes/.../matches/$matchId/replay.tsx`) ne porte que l'adresse, la validation de
 * `?t=&clock=` et les deux portes de titre (`matchmaking`, `replay`) posées au-dessus de cette page :
 * un titre sans rejeu ne la monte pas, donc ne paie aucune requête. Ce fichier lit ses paramètres et sa
 * recherche par `getRouteApi`, sans importer le module de route.
 *
 * LA PAGE NE DISPOSE RIEN. Elle charge, joint, et remet le résultat à `ReplayFrameLayout` : la scène (la
 * toile, montée UNE fois dans `ui/frames/ReplayStage`) et les avis sont les mêmes dans les trois cadres,
 * seuls l'en-tête, la rangée et la colonne changent de contenu. Basculer de cadre ne démonte donc pas la
 * toile.
 */
import { getRouteApi } from '@tanstack/react-router'
import { useMemo } from 'react'

import { normalizeCallouts } from '@/lib/replay/calloutsPaint'
import { endMatchSoundSpec } from '@/features/match-replay/sound/endMatchSound'
import { REPLAY_TEXT, type ReplayLocale } from '@/features/match-replay/i18n/i18n'
import { usePlaybackFrame, usePlaybackStore } from '@/features/match-replay/model/playbackStore'
import { replayLoadState, type ReplayLoadState } from '@/features/match-replay/model/replayLoadState'
import { useReplayModel } from '@/features/match-replay/model/useReplayModel'
import { resolveBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'
import { useBackgroundStyle } from '@/features/match-replay/settings/useBackgroundStyle'
import { buildMapBackgroundLayer } from '@/features/match-replay/hooks/mapBackgroundLayerLogic'
import { useFeedZones } from '@/features/match-replay/hooks/useFeedZones'
import { useFullWidthPage } from '@/components/shell/useFullWidthPage'
import { useCardRowLayout } from '@/features/match-replay/hooks/useCardRowLayout'
import { useKillSourceIcons } from '@/features/match-replay/hooks/useKillSourceIcons'
import { useReplayViewpoint } from '@/features/match-replay/hooks/useReplayViewpoint'
import { usePlayerPanel } from '@/features/match-replay/hooks/usePlayerPanel'
import { useReplayFullscreen } from '@/features/match-replay/hooks/useReplayFullscreen'
import { useReplayFrame } from '@/features/match-replay/hooks/useReplayFrame'
import {
  useMatchReplay,
  useMatchReplayContext,
  useReplayMapBackground,
  useReplayMapCallouts,
  useReplayMapImage,
} from '@/lib/replay/queries'
import { ReplayRegenBanner } from '@/components/replay-regen/ReplayRegenBanner'
import { ReplayFrameLayout } from '@/features/match-replay/ui/frames/ReplayFrameLayout'
import { ReplayStage } from '@/features/match-replay/ui/frames/ReplayStage'
import { cinemaFrameZones } from '@/features/match-replay/ui/frames/ReplayCinemaFrame'
import { standardFrameZones } from '@/features/match-replay/ui/frames/ReplayStandardFrame'
import { canvasSlotsOf } from '@/features/match-replay/ui/frames/replayFrameSlots'
import type { ReplayFrameChrome, ReplayFrameView } from '@/features/match-replay/ui/frames/replayFrameTypes'
import { frameToMs, resolveTacticalOpenAtFrame } from '@/lib/replay/replayLogic'
import { buildMatchHeadingStr } from '@/features/match-view/format'
import { useFriendGamertags } from '@/features/friends/queries'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { ReplayGenerateAction } from '@/features/_shared/replayGeneration/ReplayGenerateAction'
import { useAppShellStore } from '@/stores/appShellStore'

const routeApi = getRouteApi('/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay')

interface ReplayNoticesProps {
  loadState: ReplayLoadState
  isFetching: boolean
  refetch: () => unknown
  playerSlug: string
  matchId: string
  locale: ReplayLocale
}

/**
 * Ce que la page dit hors du rejeu, dans la zone `avis` de chaque cadre : le bandeau des rejeux en retard
 * après une mise à jour (sous la capacité de l'application de bureau seulement ; le rejeu ouvert s'affiche
 * comme avant, bandeau ou non), puis l'état de chargement, d'absence ou d'erreur.
 */
function ReplayNotices({ loadState, isFetching, refetch, playerSlug, matchId, locale }: ReplayNoticesProps) {
  const t = REPLAY_TEXT[locale]
  return (
    <>
      <ReplayRegenBanner />
      {(loadState === 'loading' || (loadState === 'error' && isFetching)) && (
        <p className="text-sm text-muted-foreground">{t.loading}</p>
      )}
      {loadState === 'empty' && (
        <ReplayGenerateAction variant="panel" playerSlug={playerSlug} matchId={matchId} emptyLabel={t.empty} />
      )}
      {loadState === 'error' && !isFetching && (
        <EmptyStateNotice
          title={t.loadErrorTitle}
          description={t.loadErrorDescription}
          actionLabel={t.loadErrorRetry}
          onAction={() => void refetch()}
        />
      )}
    </>
  )
}

export function ReplayPage() {
  // routeApi.useParams() porte lang?/titleSlug/playerSlug/matchId (typés) — même forme
  // que la cible matches/$matchId, réutilisable tel quel (préserve titre ET langue).
  const params = routeApi.useParams()
  const { playerSlug, matchId } = params
  const locale = useAppShellStore((s) => s.locale)
  const { data, isLoading, error, refetch, isFetching } = useMatchReplay(playerSlug, matchId)
  // LE LIEN TACTIQUE (`?t=&clock=`) : converti UNIQUEMENT quand le document est chargé — lui seul porte
  // l'origine du film. Avant ça `openAtFrame` reste `null` et le rejeu se comporte comme sans paramètre.
  const search = routeApi.useSearch()
  const { openAtFrame, showMissingOriginNotice } = useMemo(
    () => resolveTacticalOpenAtFrame(search, data),
    [search, data],
  )
  // DEUX SOURCES, DEUX RÔLES. Le film porte ce qui se passe et l'identifie par XUID ; la base porte qui
  // sont les gens. La jointure se fait ici, sur le xuid. La base est lue par le CONTEXTE DE MATCH du rejeu
  // (en-tête, tableau de score, fil, médias), pas par la vue complète du match.
  const { data: replayContext, isPending: replayContextPending } = useMatchReplayContext(playerSlug, matchId)
  // LA POSITION DE LECTURE VIENT DU MAGASIN : la toile écrit, cette page lit, à la cadence de PUBLICATION
  // (150 ms), pas à celle de l'écran — cf. `model/playbackStore`.
  const playbackStore = usePlaybackStore()
  const playFrame = usePlaybackFrame(playbackStore)

  // LE FOND DE CARTE, en deux temps : le CALAGE d'abord (quelques centaines d'octets), l'IMAGE ensuite
  // (jusqu'à 1,4 Mio), qu'on ne va chercher que si le calage existe.
  const { data: background } = useReplayMapBackground(playerSlug, matchId)
  // LE STYLE DU FOND (réglage « Fond » du panneau Vue) : l'image plan se superpose au rendu et n'est
  // demandée que si le fond en porte une ; sinon le rendu, sans rien dire.
  const [backgroundStyle] = useBackgroundStyle()
  const demandee = resolveBackgroundVariant(backgroundStyle, background?.planImage)
  // La variante qui compte pour les murs est celle RÉELLEMENT chargée (repli sur le rendu si le plan échoue).
  const { image: mapImage, variante } = useReplayMapImage(playerSlug, matchId, !!background, demandee)
  // L'IMAGE PLAN, SOURCE DU MASQUE DES ZONES NOMMÉES (`lib/replay/planMask`), chargée quel que soit le fond
  // affiché ; même clé de cache que le fond plan. Carte sans image plan, ou plan en échec : zones entières.
  const { image: planImage, variante: variantePlan } = useReplayMapImage(playerSlug, matchId, !!background?.planImage, 'plan')
  const mapBackground = useMemo(
    () => buildMapBackgroundLayer({ background, image: mapImage, variante, planImage, planVariante: variantePlan }),
    [background, mapImage, variante, planImage, variantePlan],
  )
  // LES ZONES NOMMÉES, normalisées UNE fois : la même liste sert le calque de la toile et la zone de la
  // victime du fil. 404 = la carte n'en a pas (Forge) — liste vide, rien.
  const { data: calloutsEntry } = useReplayMapCallouts(playerSlug, matchId)
  const callouts = useMemo(() => normalizeCallouts(calloutsEntry), [calloutsEntry])

  // LE MODÈLE DE LA PAGE, JOINT ICI ET NULLE PART AILLEURS : identité, marques, horloge, fenêtre de
  // gameplay, roster, fil recalé, médias, score final vivent dans `model/replayModel`, pure et testée sans
  // React ; ce hook ne fait que la mémoïser.
  const friendGamertags = useFriendGamertags(playerSlug)
  // PAR LES YEUX DE QUI : un seul foyer, ici ; sa valeur descend en paramètre à tout ce qui en dépend.
  // Le défaut est le joueur de la page, à CHAQUE montage (rien n'est persisté).
  const viewpoint = useReplayViewpoint(replayContext?.scoreboard)
  // Objet mémoïsé : `useReplayModel` mémoïse sur son identité ; un littéral neuf à chaque rendu (150 ms en
  // lecture) reconstruirait tout le modèle à chaque tick.
  const modelSettings = useMemo(() => ({ friendGamertags, locale }), [friendGamertags, locale])
  const model = useReplayModel(data, replayContext, modelSettings, viewpoint.xuid)
  // LE PANNEAU DE JOUEUR : son état naît avec la page et avec le match, jamais mémorisé, et NE TOUCHE PAS
  // au point de vue. Le joueur SUIVI est ce même joueur regardé (`viewpoint.xuid`).
  const playerPanel = usePlayerPanel(matchId, model.players, viewpoint.xuid)
  // LA ZONE DE LA VICTIME de chaque frag du fil : calculée UNE fois sur le fil aligné (instant brut).
  const feedZoneOf = useFeedZones(model.feed, data, callouts, locale, model.players)
  // Les pictogrammes d'arme du titre : les lignes « a détruit » du fil y retrouvent l'arme de la cause.
  const weaponIcons = useKillSourceIcons(params.titleSlug, locale)
  const nowMs = data ? frameToMs(playFrame, data) : 0
  // LA FIN DE PARTIE SONORE reste À LA PAGE : elle dépend de la LANGUE (voix d'annonceur), que le modèle ne
  // connaît pas. C'est la MÊME lecture que l'écran de fin (`readVictory`). Et c'est la SEULE surface qui ne
  // suit pas le point de vue : aucun sujet ne lui est passé, donc elle lit le résultat du JOUEUR DE LA PAGE
  // (inspecter un adversaire ne doit pas faire jouer « Défaite » sur un match gagné). L'incohérence
  // apparente avec l'écran, qui suit le point de vue, est voulue.
  const endMatchSound = useMemo(
    () => endMatchSoundSpec(model.allegiance, model.scoreboard, replayContext?.header.outcome_code, locale),
    [model.allegiance, model.scoreboard, replayContext?.header.outcome_code, locale],
  )

  const loadState = replayLoadState({
    isLoading,
    hasTracks: !!data && data.tracks.length > 0,
    error,
    identityPending: replayContextPending,
  })
  // LE CADRE ET LE PLEIN ÉCRAN DU SYSTÈME vivent ICI parce que le cadre change la mise en page de LA PAGE
  // (cf. useReplayFrame) ; la toile n'en reçoit que les commandes : touches T et F, bouton de la barre.
  // SA PORTÉE : le match, tant que la page a un rejeu à montrer (une toile, donc une barre et ses icônes de
  // cadre). Chargement, erreur et absence n'en ont pas, et le match qui change non plus : le cadre revient
  // alors au Standard (cf. useReplayFrame).
  const frameControl = useReplayFrame(data && loadState === 'ready' ? `${playerSlug}/${matchId}` : null)
  const fullscreen = useReplayFullscreen(frameControl)

  const chrome: ReplayFrameChrome = {
    locale,
    params,
    // Même libellé que la vue match (mode + carte), calculé ICI et nulle part ailleurs.
    matchLabel: buildMatchHeadingStr(replayContext?.header.map_ui, replayContext?.header.mode_ui, locale),
    header: replayContext?.header,
    doc: data,
    frameControl,
    fullscreen,
  }
  const view: ReplayFrameView | null =
    data && loadState === 'ready'
      ? {
          doc: data,
          locale,
          titleSlug: params.titleSlug,
          playerSlug,
          matchId,
          model,
          header: replayContext?.header,
          viewpoint,
          playerPanel,
          playbackStore,
          playFrame,
          nowMs,
          mapBackground,
          callouts,
          endMatchSound,
          openAtFrame,
          showMissingOriginNotice,
          feedZoneOf,
          weaponIcons,
          fullscreen,
          frameControl,
        }
      : null
  const frame = frameControl.frame
  // Le Cinéma est dessiné sur toute la fenêtre : la coquille ne borne pas son corps à 1320 px.
  useFullWidthPage(frame === 'cinema')
  const rowLayout = useCardRowLayout(frame, view)
  const zones = frame === 'standard' ? standardFrameZones(chrome, view, rowLayout) : cinemaFrameZones(frame, chrome, view, rowLayout)
  return (
    <ReplayFrameLayout
      frame={frame}
      {...zones}
      avis={
        <ReplayNotices
          loadState={loadState}
          isFetching={isFetching}
          refetch={refetch}
          playerSlug={playerSlug}
          matchId={matchId}
          locale={locale}
        />
      }
      // Le score est dans l'en-tête (Standard, ou équipes dans la colonne), ou au centre de la rangée des équipes
      // quand elle est posée ; il ne coiffe la toile que si l'en-tête est trop étroit pour le loger.
      scene={
        view && (
          <ReplayStage
            view={view}
            scorePlacement={rowLayout.scorePlacement === 'scene' ? 'scene' : null}
            slots={canvasSlotsOf(frame, rowLayout.planFeedPosed ? view : null)}
          />
        )
      }
    />
  )
}
