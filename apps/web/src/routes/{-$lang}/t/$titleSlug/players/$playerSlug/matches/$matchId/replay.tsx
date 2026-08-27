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
import { endMatchSoundSpec } from '@/features/match-replay/endMatchSound'
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
import { replayWindow } from '@/features/match-replay/replayWindow'
import { ReplayScoreBanner } from '@/features/match-replay/ReplayScoreBanner'
import { ReplayTeams } from '@/features/match-replay/ReplayTeams'
import { ReplayVictoryOverlay } from '@/features/match-replay/ReplayVictoryOverlay'
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
  // LE CADRAGE SUR LE MATCH RÉEL, CALCULÉ UNE FOIS ICI (et nulle part ailleurs) : le film
  // déborde le match du countdown d'avant-partie et d'une queue de 5-6 s, et les deux bornes
  // demandent l'artefact ET l'en-tête — deux requêtes distinctes qui ne se rejoignent qu'à ce
  // niveau. La lecture, la frise, l'horloge, le fil et les infobulles la reçoivent ; aucun ne
  // la recalcule. `null` = pas de cadrage établi, tout le monde retombe sur le film entier.
  const playWindow = useMemo(
    () => (data ? replayWindow(data, matchView?.header) : null),
    [data, matchView?.header],
  )

  // LA FIN DE PARTIE SONORE (lot C), lue ICI comme le cadrage et pour la même raison : elle
  // croise l'en-tête (l'issue du joueur de la page) et le scoreboard (ses camps), deux données
  // qui ne se rejoignent qu'à ce niveau. C'est la MÊME lecture que l'écran de fin ci-dessous —
  // `endMatchSoundSpec` s'appuie sur `readVictory`, il ne re-décode pas `outcome_code`.
  const endMatchSound = useMemo(
    () => endMatchSoundSpec(scoreboard, matchView?.header.outcome_code, locale),
    [scoreboard, matchView?.header.outcome_code, locale],
  )

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
        /* DEUX COLONNES : LA CARTE, PUIS FICHES + FIL EMPILÉS À DROITE (demande utilisateur
           du 2026-08-24, qui remplace la rangée à trois colonnes du 16/08) : le fil passe
           SOUS les fiches, à la même largeur qu'elles, et la carte récupère toute la place
           libérée à gauche. La colonne de droite est passée de 28 à 30 rem (+7 %, « élargir
           les fiches de 5 à 10 % »).

           C'EST LA CARTE QUI IMPOSE LA HAUTEUR DE LA RANGÉE, et c'est la technique du POC qui
           l'obtient : la colonne de droite est `relative` et son contenu `absolute inset-0`
           à partir de `xl`. Un contenu absolu ne participe pas au calcul de hauteur de son
           parent — un fil de 80 kills ou un BTB à 24 fiches ne peuvent donc PAS étirer la
           ligne. Dedans, les FICHES prennent leur hauteur naturelle (bornée, défilable) et le
           FIL remplit le reste (« pour la hauteur faudrait que ça s'adapte »).

           SOUS 1280 px (`xl`) la rangée n'a plus la place : la carte reprend toute la largeur
           et fiches puis fil passent dessous, chacun borné à 60 % de la hauteur d'écran. */
        <div className="grid gap-3 xl:grid-cols-[minmax(0,1fr)_minmax(0,30rem)] xl:items-stretch">
          {/* `min-w-0` : sans lui, un contenu large ferait déborder la colonne au lieu de la
              contraindre — c'est la colonne que le ResizeObserver du canvas mesure. */}
          {/* `relative` : c'est le repère de l'ÉCRAN DE VICTOIRE, qui coiffe toute la colonne
              de la carte (bandeau, terrain, frise) à la fin du match. Il se monte ICI et non
              dans le canvas, qui est déjà au plafond de taille du dépôt. */}
          <section className="relative min-w-0">
            {/* LE BANDEAU DE SCORE COIFFE LE TERRAIN (demande utilisateur du 2026-08-20) :
                score des deux camps à l'image lue, de part et d'autre de l'horloge. Il est
                DANS la colonne du canvas, et non en frère de celle-ci : la rangée est une
                grille à trois colonnes ordonnées, où un frère supplémentaire prendrait une
                cellule et décalerait le fil, le terrain et les fiches d'un cran. Ici il
                coiffe la carte, à sa largeur exacte, sans toucher à la grille.

                IL TIQUE SANS RIEN AJOUTER : la page tient déjà `frame` (le canvas le lui
                publie toutes les 150 ms via `onFrameChange`) et `nowMs` en découle. C'est
                la même horloge que le fil et que les scores en tête des colonnes — un
                montage déclaratif suffit, aucun pilotage par ref n'est nécessaire. */}
            <ReplayScoreBanner
              doc={data}
              scoreboard={scoreboard}
              xuidMeta={xuidMeta}
              frame={frame}
              nowMs={nowMs}
              playWindow={playWindow}
              locale={locale}
            />
            <ReplayCanvas
              doc={data}
              locale={locale}
              playWindow={playWindow}
              kills={kills}
              t0Ms={t0Ms}
              onFrameChange={setFrame}
              background={mapBackground}
              callouts={callouts}
              scoreboard={scoreboard}
              xuidMeta={xuidMeta}
              marks={marks}
              endMatch={endMatchSound}
            />
            {/* L'ÉCRAN DE FIN DE MATCH, dérivé de la position de lecture (D-B5) : il apparaît
                quand la lecture atteint la borne de fin et disparaît dès qu'on remonte la
                frise. Il laisse passer les clics — la frise est dessous. */}
            <ReplayVictoryOverlay
              doc={data}
              scoreboard={scoreboard}
              xuidMeta={xuidMeta}
              outcomeCode={matchView?.header.outcome_code}
              outcomeLabel={matchView?.header.outcome_label}
              playWindow={playWindow}
              frame={frame}
              titleSlug={params.titleSlug}
              locale={locale}
            />
          </section>
          {/* FICHES AU-DESSUS, FIL EN DESSOUS, même largeur (demande du 2026-08-24) : un
              rejeu se lit en balayant du terrain vers les joueurs, puis vers l'événement.
              Les fiches gardent leur hauteur naturelle (bornée à 62 % de la colonne, elles
              défilent au-delà — BTB) ; le fil PERMANENT (verdict user 2026-08-13) remplit
              tout le reste et défile dedans. */}
          <aside className="relative min-h-[12rem] xl:min-h-0">
            <div className="flex max-h-[80vh] flex-col gap-3 xl:absolute xl:inset-0 xl:max-h-none">
              <div className="flex max-h-[60vh] min-h-0 shrink-0 flex-col overflow-hidden xl:max-h-[62%]">
                {/* PAS DE `marks` ICI (2026-08-25) : les fiches ne portent plus de glyphe
                    d'identité. La table reste servie à la CARTE (forme du point) et au FIL
                    (glyphe devant un nom), ses deux derniers lecteurs. */}
                <ReplayTeams
                  doc={data}
                  scoreboard={scoreboard}
                  frame={frame}
                  locale={locale}
                  xuidMeta={xuidMeta}
                />
              </div>
              <div className="flex min-h-0 flex-1 flex-col">
                <ReplayKillFeed
                  kills={kills}
                  medals={medalEvents}
                  t0Ms={t0Ms}
                  nowMs={nowMs}
                  playWindow={playWindow}
                  doc={data}
                  scoreboard={scoreboard}
                  xuidMeta={xuidMeta}
                  locale={locale}
                  marks={marks}
                  colorOf={colorOfTeam}
                />
              </div>
            </div>
          </aside>
        </div>
      )}
    </div>
  )
}
