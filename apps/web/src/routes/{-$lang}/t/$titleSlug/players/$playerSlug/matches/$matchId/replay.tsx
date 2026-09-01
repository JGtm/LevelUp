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
import { buildFeedEntries, collectMedalEvents } from '@/features/match-replay/killFeedLogic'
import { buildReplayMedia } from '@/features/match-replay/replayMediaLogic'
import { buildPlayerMarks } from '@/features/match-replay/playerMarks'
import { ReplayCanvas } from '@/features/match-replay/ReplayCanvas'
import { ReplayKillFeed } from '@/features/match-replay/ReplayKillFeed'
import { ReplayMatchRecall } from '@/features/match-replay/ReplayMatchRecall'
import { frameToMs } from '@/features/match-replay/replayLogic'
import { replayWindow } from '@/features/match-replay/replayWindow'
import { ReplayBombCountdownOverlay } from '@/features/match-replay/ReplayBombCountdownOverlay'
import { ReplayRoundBreakOverlay } from '@/features/match-replay/ReplayRoundBreakOverlay'
import { ReplayScoreBanner } from '@/features/match-replay/ReplayScoreBanner'
import { ReplayTeams } from '@/features/match-replay/ReplayTeams'
import { ReplayVictoryOverlay } from '@/features/match-replay/ReplayVictoryOverlay'
import { finalScoreFromHeader } from '@/features/match-replay/victoryLogic'
import { MatchBreadcrumb } from '@/features/match-view/MatchHeader'
import { buildMatchHeadingStr } from '@/features/match-view/format'
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
  // LE SCORE FINAL DU MATCH, quand il ne se déduit PAS du film : sur un mode à manches, le
  // calque rendrait à la borne de fin les points de la dernière manche (« 100 - 43 ») au lieu
  // du résultat (« 2 - 1 »). L'écran de fin ET son jumeau repeint dans la vidéo exportée
  // reçoivent donc la même valeur, celle que la vue match affiche déjà en tête.
  const finalScore = useMemo(
    () => finalScoreFromHeader(matchView?.header),
    [matchView?.header],
  )
  // LE FIL ALIGNÉ, ASSEMBLÉ ICI ET NULLE PART AILLEURS (planche 2a, 2026-08-28) : la colonne
  // de droite l'affiche, la frise du lecteur en tire ses pistes « Toi » et « Alliés ». Un
  // second appel côté canvas referait tout le recalage — et surtout, deux recalages menés
  // séparément peuvent diverger : la marque d'une élimination sur la frise ne serait alors
  // plus exactement la ligne qu'on lit dans le fil.
  const feedEntries = useMemo(
    () => buildFeedEntries(kills, medalEvents, t0Ms, data),
    [kills, medalEvents, t0Ms, data],
  )
  // LES MÉDIAS DU MATCH, RECALÉS ICI ET NULLE PART AILLEURS (phase 2, 2026-08-28) : ce sont
  // ceux de l'onglet médias du match, déjà en mémoire — la page monte la vue du match pour ses
  // fiches et son fil, aucun appel de plus. Leur horodatage est ABSOLU (l'heure de la capture),
  // là où le fil compte en millisecondes de match : `buildReplayMedia` fait la soustraction,
  // une seule fois, sur la même origine que le fil (`doc.originMs`).
  const replayMedia = useMemo(
    () => buildReplayMedia(matchView?.media_tab, matchView?.header, data?.originMs),
    [matchView?.media_tab, matchView?.header, data?.originMs],
  )
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

  // Fil d'Ariane : même label que la vue match (mode + map). La date est portée par la
  // ReplayMatchRecall ci-dessous, pas besoin de la répéter ici.
  const matchLabel = buildMatchHeadingStr(matchView?.header.map_ui, matchView?.header.mode_ui, locale)

  // CE QUI DÉBORDAIT N'ÉTAIT PAS DU CONTENU (retour du 2026-08-27 : « je peux descendre
  // alors qu'il n'y a rien »). Cette page est une pile RIGIDE — titre, bandeau de score, et
  // une carte dont la hauteur est constante — dans un shell où seul le <main> défile. Quand
  // le dépassement de la pile est PLUS PETIT que la marge basse de la page, défiler ne révèle
  // que du vide : c'est le fantôme, et il ne peut pas disparaître tant que la page a une
  // marge basse et une carte fixe — il peut rétrécir. Les marges verticales tombent donc de
  // 24 à 12 px (le vide défilable au pire des cas est divisé par deux), et la ligne de rappel
  // gagnée ci-dessous vit dans une hauteur RÉSERVÉE : la pile ne gagne pas un pixel sur
  // l'avant-correctif — aucune fenêtre qui tenait juste ne se met à défiler (revue R1).
  // CE QUE ÇA NE RÈGLE PAS : sous ~730 px de fenêtre (1366x768, ou 1080p à 150 %), c'est la
  // carte elle-même qui ne tient plus dans l'écran. Là, le scroll a quelque chose à montrer,
  // et le corriger demanderait une hauteur de carte qui s'adapte — hors périmètre ici (D8).
  return (
    <div className="flex flex-col">
      <MatchBreadcrumb playerSlug={playerSlug} matchLabel={matchLabel} locale={locale} />
      <div className="space-y-4 px-6 py-3">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <h1 className="flex items-center gap-2 text-lg font-semibold">
            <img src={themedIconSrc('replay', theme)} alt="" aria-hidden className="h-5 w-auto" />
            {t.title}
          </h1>
          {/* DE QUEL MATCH PARLE-T-ON ? (retour du 2026-08-27) La page match le dit en tête ;
              le rejeu montrait un terrain sans jamais le nommer. Les libellés sont déjà en
              mémoire — c'est la vue du match que la page monte pour ses fiches et son fil.
              LA HAUTEUR EST RÉSERVÉE dès le premier rendu (revue R1) : cette vue peut arriver
              APRÈS le rejeu, et une ligne qui pousse de 20 px ce qu'on est en train de lire
              est un saut — le vide réservé, lui, ne promet rien et ne fait rien bouger. */}
          <div className="mt-1 min-h-6">
            <ReplayMatchRecall
              mapUI={matchView?.header.map_ui}
              modeUI={matchView?.header.mode_ui}
              startTimeLabel={matchView?.header.start_time_label}
              playlistLabel={matchView?.header.playlist_label}
              locale={locale}
            />
          </div>
        </div>
        <Link
          to="/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId"
          params={params}
          className="inline-flex h-8 shrink-0 items-center justify-center gap-2 rounded-md border border-border bg-transparent px-3 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
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
              outcome={{
                // LE VERDICT, POUR L'EXPORT SEUL : l'écran de fin monté juste en dessous est du
                // DOM, qu'aucun encodeur vidéo ne voit. L'export le repeint DANS la toile, et
                // c'est ici qu'il reçoit de quoi le faire — la MÊME source que l'écran affiché.
                code: matchView?.header.outcome_code,
                label: matchView?.header.outcome_label,
                finalScore,
              }}
              feedEntries={feedEntries}
              media={replayMedia}
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
              finalScore={finalScore}
              playWindow={playWindow}
              frame={frame}
              titleSlug={params.titleSlug}
              locale={locale}
            />
            {/* LE MESSAGE INTER-MANCHE (Oddball et modes multi-manche), dérivé de la position
                de lecture comme l'écran de fin : « Manche N terminée » paraît brièvement à la
                bascule d'une manche à la suivante et laisse passer les clics. Sur un mode à
                manche unique, il ne se rend jamais. */}
            <ReplayRoundBreakOverlay doc={data} frame={frame} locale={locale} />
            {/* LE COMPTE À REBOURS DE LA BOMBE (Assaut, schéma 29), dérivé de la position de
                lecture comme le message inter-manche : « Bombe armée — 4,9 s » et sa barre de
                mèche, en haut du terrain, pendant les ~5 s qui précèdent l'explosion. Hors des
                variantes couvertes (One Bomb comprise), le calque est vide : rien ne se rend. */}
            <ReplayBombCountdownOverlay doc={data} frame={frame} locale={locale} />
          </section>
          {/* FICHES AU-DESSUS, FIL EN DESSOUS, même largeur (demande du 2026-08-24) : un
              rejeu se lit en balayant du terrain vers les joueurs, puis vers l'événement.
              Les fiches gardent leur hauteur naturelle (bornée à 62 % de la colonne, elles
              défilent au-delà — BTB) ; le fil PERMANENT (verdict user 2026-08-13) remplit
              tout le reste et défile dedans.

              AUCUNE HAUTEUR MINIMALE ICI : la colonne en portait une (12 rem), qui ne mordait
              que dans le trou où le rejeu est arrivé avant la vue du match — fiches et fil y
              tiennent en ~160 px, et les 32 px manquants étaient du vide sous lequel il n'y
              avait rien. C'est exactement le scroll fantôme qu'on chasse. Hors de ce trou,
              fiches + fil dépassent toujours cette borne : elle ne servait rien. */}
          <aside className="relative">
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
                  entries={feedEntries}
                  nowMs={nowMs}
                  playWindow={playWindow}
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
    </div>
  )
}
