/**
 * Route /{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay — Rejeu 2D
 * du match (vue du dessus).
 *
 * Trajectoires des joueurs décodées du film, servies par l'API comme artefact
 * pré-construit (GET .../replay). La disponibilité est PAR MATCH : 404 → état vide.
 * Ce n'est donc pas un drapeau global — le stub `REJEU_2D_ENABLED` qui occupait cette
 * route est remplacé par l'implémentation réelle.
 *
 * DEUX PORTES DE TITRE, ET ELLES NE DISENT PAS LA MÊME CHOSE (2026-09-06, v2 D.14) :
 *  - `matchmaking` : sans elle, le match n'existe pas pour ce titre, donc son rejeu non plus ;
 *  - `replay` : le titre a des matchs, mais aucune chaîne de rejeu (capability title-level du
 *    lot C, servie par le bootstrap). Halo 5 est dans ce cas.
 * La porte est posée AU-DESSUS du composant : la page n'est pas montée, donc aucun de ses
 * hooks de chargement ne tourne — ni l'artefact (1,5 à 2,7 Mio), ni la vue match, ni le fond
 * de carte. Un titre sans rejeu ne paie aucune requête, et lit un état qui le dit
 * (`FeatureUnavailable`, libellé `replay` FR/EN).
 *
 * `?t=<instant_ms>&clock=match|film` OUVRE LE REJEU À L'INSTANT EXACT (lot M1b, 2026-09-08,
 * décision utilisateur ferme « corriger le décalage ») — lien posé par `TacticalCellCard`
 * depuis une cellule de la grille Tactique. La route attend le document (il porte l'offset
 * de calage), convertit avec la fonction pure `resolveTacticalReplayInstant` +
 * `msToFrames` (`lib/replay/replayLogic.ts`), et positionne `ReplayCanvas` via `openAtFrame`.
 * `clock=match` sur un artefact dont le pont n'a apparié aucune mort (schéma < 49, ou pont
 * muet même à jour) : le rejeu s'ouvre au début et affiche un avis — JAMAIS un saut
 * approximatif présenté comme exact.
 */
import { createFileRoute, Link } from '@tanstack/react-router'
import { useCallback, useMemo } from 'react'
import { z } from 'zod'

import { normalizeCallouts } from '@/features/match-replay/layers/calloutsLayer'
import { endMatchSoundSpec } from '@/features/match-replay/sound/endMatchSound'
import { REPLAY_TEXT } from '@/features/match-replay/i18n/i18n'
import { usePlaybackFrame, usePlaybackStore } from '@/features/match-replay/model/playbackStore'
import { useReplayModel } from '@/features/match-replay/model/useReplayModel'
import { useReplayViewpoint } from '@/features/match-replay/hooks/useReplayViewpoint'
import {
  useMatchReplay,
  useReplayMapBackground,
  useReplayMapCallouts,
  useReplayMapImage,
} from '@/lib/replay/queries'
import { ReplayCanvas } from '@/features/match-replay/ui/ReplayCanvas'
import { ReplayKillFeed } from '@/features/match-replay/ui/ReplayKillFeed'
import { ReplayMatchRecall } from '@/features/match-replay/ui/ReplayMatchRecall'
import { frameToMs, resolveTacticalOpenAtFrame } from '@/lib/replay/replayLogic'
import { ReplayBombCountdownOverlay } from '@/features/match-replay/ui/ReplayBombCountdownOverlay'
import { ReplayRoundBreakOverlay } from '@/features/match-replay/ui/ReplayRoundBreakOverlay'
import { ReplayScoreBanner } from '@/features/match-replay/ui/ReplayScoreBanner'
import { ReplayTeams } from '@/features/match-replay/ui/ReplayTeams'
import { ReplayVictoryOverlay } from '@/features/match-replay/ui/ReplayVictoryOverlay'
import { MatchBreadcrumb } from '@/features/match-view/MatchHeader'
import { buildMatchHeadingStr } from '@/features/match-view/format'
import { useMatchView } from '@/features/match-view/queries'
import type { TeamColorResolver } from '@/features/match-view/teamColor'
import { useSettings } from '@/features/settings/queries'
import { tokenCssVar } from '@/lib/accessibility'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'
import { themedIconSrc } from '@/lib/themedIcon'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

export const Route = createFileRoute(
  '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay',
)({
  // `t` (instant_ms) + `clock` : cf. l'en-tête. Une valeur hostile ou mal formée retombe sur
  // `undefined` (`.catch`) plutôt que de faire échouer la route — un lien cassé ouvre le
  // rejeu au début, comme sans paramètre du tout, jamais une page en erreur.
  //
  // `t` RESTE UNE CHAÎNE ICI, PAS UN NOMBRE : `FullSearchSchema` (TanStack Router) fusionne
  // les schémas de TOUTES les routes, et `HelpPage.tsx`/`SettingsPage.tsx` construisent un
  // `URLSearchParams` depuis `routerState.location.search` en supposant CHAQUE valeur déjà
  // une chaîne — un champ numérique ici casse leur build, sans rapport avec ce lot. La
  // conversion en nombre se fait au composant (`Number(search.t)`), une fois validée.
  validateSearch: z.object({
    t: z.string().optional().catch(undefined),
    clock: z.enum(['match', 'film']).optional().catch(undefined),
  }),
  component: () => (
    <RouteCapabilityGate capability="matchmaking">
      <RouteCapabilityGate capability="replay">
        <ReplayPage />
      </RouteCapabilityGate>
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
  // LE LIEN TACTIQUE (`?t=&clock=`, cf. l'en-tête) : converti UNIQUEMENT quand le document est
  // chargé — lui seul porte le calage (`coverage.bridge.deathOffsetMs`). Avant ça, `openAtFrame`
  // reste `null` et le rejeu se comporte comme sans paramètre (`ReplayCanvas` cadre au coup
  // d'envoi, cf. `useReplayPlayback`). Logique PURE et testée seule dans
  // `lib/replay/replayLogic.resolveTacticalOpenAtFrame` — cette page ne fait que l'appeler.
  const search = Route.useSearch()
  const { openAtFrame, showUncalibratedNotice } = useMemo(
    () => resolveTacticalOpenAtFrame(search, data),
    [search, data],
  )
  // DEUX SOURCES, DEUX RÔLES. Le film porte ce qui se passe et l'identifie par XUID ; la
  // base porte qui sont les gens. L'artefact ne mélange pas les deux — la jointure se fait
  // ici, sur le xuid, qui est la seule clé qui ne suppose rien (surtout pas un ordre).
  const { data: matchView } = useMatchView(playerSlug, matchId)
  // LA POSITION DE LECTURE VIENT DU MAGASIN, la page n'en garde plus de copie (2026-09-06,
  // W1) : le canvas ecrit, cette page lit. L'abonnement suit la cadence de PUBLICATION
  // (150 ms), pas celle de l'ecran — cf. `model/playbackStore`.
  const playbackStore = usePlaybackStore()
  const frame = usePlaybackFrame(playbackStore)

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

  // LE MODÈLE DE LA PAGE, JOINT ICI ET NULLE PART AILLEURS (2026-09-06, W1). L'artefact dit
  // ce qui s'est passé et l'identifie par XUID ; la vue match dit qui sont les gens. La
  // jointure des deux — identité, marques, horloge, fenêtre de gameplay, roster, fil recalé,
  // médias, score final — vit dans `model/replayModel`, pure et testée sans React ; ce hook
  // ne fait que la mémoïser. La page n'en garde que ce qui dépend de la LANGUE ou de
  // l'affichage, plus bas.
  const { data: settings } = useSettings()
  // PAR LES YEUX DE QUI (2026-09-06, L2b) : un seul foyer, ici, et sa valeur descend en
  // PARAMÈTRE à tout ce qui en dépend — modèle, calques d'objectif, écran de fin, export.
  // Le défaut est le joueur de la page, à CHAQUE montage (décision 8 : rien n'est persisté).
  // Depuis le 2026-09-07 (L3), le MENU de la première piste de la frise appelle `select` : il
  // reçoit `model.players` pour ses sections et ne touche à rien d'autre que ce foyer.
  const viewpoint = useReplayViewpoint(matchView?.team_tab.scoreboard)
  const model = useReplayModel(data, matchView, settings, viewpoint.xuid)
  const { scoreboard, identity: xuidMeta, marks, window: playWindow, feed: feedEntries } = model
  // LA PAGE PARLE D'UNE SEULE VOIX (décision D1) : sur le rejeu, les points, les titres de
  // colonnes et les noms du fil prennent les MÊMES tokens d'accessibilité — allié / adverse,
  // surchargeables par les réglages. La cascade d'identité du fil (couleur backend, puis
  // couleur officielle de l'équipe) reste celle de la Match View, qui ne change pas : c'est
  // pourquoi le résolveur est passé en prop plutôt qu'imposé dans le composant.
  const colorOfTeam = useCallback<TeamColorResolver>(
    (_teamID, ally) => tokenCssVar(ally ? 'team-ally' : 'team-enemy'),
    [],
  )
  const nowMs = data ? frameToMs(frame, data) : 0
  // LA FIN DE PARTIE SONORE (lot C) reste À LA PAGE, et pour une raison précise : elle dépend
  // de la LANGUE (voix d'annonceur), que le modèle ne connaît pas. C'est la MÊME lecture que
  // l'écran de fin ci-dessous — `endMatchSoundSpec` s'appuie sur `readVictory`, il ne
  // re-décode pas `outcome_code`.
  //
  // ET C'EST LA SEULE SURFACE QUI NE SUIT PAS LE POINT DE VUE (décision 3 du plan, 2026-09-06) :
  // aucun sujet ne lui est passé, donc elle lit le résultat du JOUEUR DE LA PAGE. Inspecter un
  // adversaire ne doit pas faire jouer « Défaite » sur un match gagné. L'écran, lui, suit ce
  // qu'on regarde — l'incohérence apparente entre les deux appels est voulue, ne pas la
  // « corriger ».
  const endMatchSound = useMemo(
    () => endMatchSoundSpec(scoreboard, matchView?.header.outcome_code, locale),
    [scoreboard, matchView?.header.outcome_code, locale],
  )

  const hasReplay = !!data && data.tracks.length > 0

  // Fil d'Ariane : même label que la vue match (mode + map). Il est calculé ICI et NULLE PART
  // AILLEURS — le rappel du match, qui le recalculait, ne porte plus que la date et la playlist.
  const matchLabel = buildMatchHeadingStr(matchView?.header.map_ui, matchView?.header.mode_ui, locale)

  // TOUTE LA TÊTE TIENT SUR LA LIGNE DU FIL D'ARIANE (demande du 2026-09-02 : « la partie sous
  // la L1 pourrait être compactée un peu »). Elle occupait ~84 px de plus en répétant ce que le
  // fil disait déjà : un `h1` sous un fil qui nomme la page, et un rappel du match qui rappelait
  // le libellé du fil. Le `h1` devient le SEGMENT FEUILLE (sémantique intacte, icône comprise),
  // le rappel devient le complément de ce segment, et le lien vers la fiche s'aligne à droite.
  //
  // POURQUOI ÇA COMPTE ICI PLUS QU'AILLEURS : sur cette page, le budget vertical est la
  // ressource rare. Le canvas est le seul élément élastique de la pile (le bloc transport est
  // incompressible, ce sont des commandes) — chaque pixel rendu par la tête devient donc un
  // pixel de terrain, via le clamp de `useReplayViewport`. Compaction et hauteur élastique ne
  // s'opposent pas : la première alimente la seconde.
  //
  // « FICHE DU MATCH », PAS « RETOUR AU MATCH » : la flèche du fil, juste à gauche, fait
  // `router.history.back()` — l'historique, donc n'importe quoi. Ce lien-ci vise une
  // destination fixe, qu'on en vienne ou non. Deux choses différentes ne portent pas le même
  // mot, et le libellé ne se conditionne PAS à l'historique : il clignoterait selon le parcours.
  return (
    <div className="flex flex-col">
      <MatchBreadcrumb
        playerSlug={playerSlug}
        matchLabel={matchLabel}
        locale={locale}
        leaf={
          <h1 className="flex items-center gap-1.5 text-sm font-semibold">
            <img src={themedIconSrc('replay', theme)} alt="" aria-hidden className="h-4 w-auto" />
            {t.title}
          </h1>
        }
        detail={
          <ReplayMatchRecall
            startTimeLabel={matchView?.header.start_time_label}
            playlistLabel={matchView?.header.playlist_label}
          />
        }
        action={
          <Link
            to="/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId"
            params={params}
            className="inline-flex h-7 items-center justify-center gap-2 rounded-md border border-border bg-transparent px-2.5 text-xs font-medium transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            {t.back}
          </Link>
        }
      />
      <div className="space-y-4 px-6 pb-3">
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
            {/* AVIS « INSTANT NON CALÉ » (lot M1b, 2026-09-08) : le lien tactique demandait un
                instant sur l'horloge du match, mais cet artefact n'a pas de calage connu
                (schéma < 49, ou pont d'identité qui n'a apparié aucune mort). Le rejeu s'ouvre
                quand même — au début, jamais sur un saut approximatif présenté comme exact. */}
            {showUncalibratedNotice && (
              <EmptyStateNotice
                className="mb-2"
                title={t.openAtUncalibratedTitle}
                description={t.openAtUncalibratedDescription}
              />
            )}
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
              playbackStore={playbackStore}
              openAtFrame={openAtFrame}
              background={mapBackground}
              callouts={callouts}
              scoreboard={scoreboard}
              xuidMeta={xuidMeta}
              marks={marks}
              endMatch={endMatchSound}
              viewpoint={viewpoint.xuid}
              players={model.players}
              onSelectViewpoint={viewpoint.select}
              outcome={{
                // LE VERDICT, POUR L'EXPORT SEUL : l'écran de fin monté juste en dessous est du
                // DOM, qu'aucun encodeur vidéo ne voit. L'export le repeint DANS la toile, et
                // c'est ici qu'il reçoit de quoi le faire — la MÊME source que l'écran affiché.
                // Le CODE seul : le MOT se résout des deux côtés depuis les mappings du titre
                // (2026-09-07), sur l'issue lue — un `outcome_label` de plus rouvrirait le
                // second vocabulaire que ce chantier a fermé.
                code: matchView?.header.outcome_code,
                finalScore: model.score,
              }}
              feedEntries={feedEntries}
              media={model.media}
              /* LES TROIS SURCOUCHES SE POSENT SUR LA CARTE, PAS SUR LE LECTEUR (2026-09-08,
                 retour utilisateur : « message et image défaite/victoire centrer sur la hauteur
                 de la map, là ça prend la hauteur du lecteur en compte, ce qui n'est pas
                 correct »).

                 Elles étaient montées ICI, sœurs de `ReplayCanvas` : leur `inset-0` couvrait
                 alors la bannière de score, la carte ET la barre de lecture, et le bloc
                 « DÉFAITE » tombait 106 px sous le centre du terrain. Les envelopper d'un
                 `relative` à ce niveau ne suffisait pas — `ReplayCanvas` rend LUI AUSSI la barre
                 de lecture. Elles descendent donc dans le conteneur de la toile, seul endroit
                 dont les bornes sont celles de la carte (cf. `ReplayCanvasProps.mapOverlays`).

                 LES TROIS, ET PAS LE SEUL ÉCRAN DE FIN : le message inter-manche partage son
                 bloc et ses styles (`replayOverlayStyles.ts`) — n'en déplacer qu'un les aurait
                 posés à deux hauteurs différentes ; et le compte à rebours de la bombe, qui se
                 veut « en haut du terrain », se posait en haut du LECTEUR. */
              mapOverlays={
                <>
                  <ReplayVictoryOverlay
                    doc={data}
                    scoreboard={scoreboard}
                    xuidMeta={xuidMeta}
                    outcomeCode={matchView?.header.outcome_code}
                    viewpoint={viewpoint.xuid}
                    finalScore={model.score}
                    playWindow={playWindow}
                    frame={frame}
                    titleSlug={params.titleSlug}
                    locale={locale}
                  />
                  <ReplayRoundBreakOverlay doc={data} frame={frame} locale={locale} />
                  <ReplayBombCountdownOverlay doc={data} frame={frame} locale={locale} />
                </>
              }
            />
          </section>
          {/* FICHES AU-DESSUS, FIL EN DESSOUS, même largeur (demande du 2026-08-24) : un
              rejeu se lit en balayant du terrain vers les joueurs, puis vers l'événement.
              Les fiches gardent leur hauteur naturelle (bornée, elles défilent au-delà — BTB) ;
              le fil PERMANENT (verdict user 2026-08-13) remplit tout le reste et défile dedans.

              LE PLAFOND DES FICHES N'EST PLUS UN POURCENTAGE DE LA CARTE (2026-09-02). Il valait
              62 % de la rangée, dont la hauteur est celle de la colonne carte — donc, depuis que
              cette carte est élastique, une promesse exprimée en fiches adossée à une hauteur
              variable. Le calcul : une fiche coûte ~100 px, un en-tête d'équipe ~30, soit 442 px
              pour un 4v4. La rangée valait 773 px (62 % = 479, il tenait à 37 px près) ; sur un
              écran contraint elle tombe à 653 (62 % = 405, il ne tient plus). Et à 5v5 (542 px)
              il ne tenait DÉJÀ pas avant — le défaut préexistait sur les gros effectifs, la
              hauteur élastique n'a fait que l'étendre au 4v4.

              LE PLAFOND EST DONC EN PIXELS, ET BORNÉ PAR LA PLACE DU FIL :
              `min(30rem, 100% - 12rem)` dit les deux choses d'un coup — au plus 30 rem (de quoi
              loger un 4v4 entier avec de la marge), et jamais au point de laisser moins de
              12 rem au fil. Sur une colonne haute c'est le premier terme qui mord, sur une
              colonne courte le second : aucun des deux ne peut être exprimé sans l'autre. */}
          <aside className="relative">
            <div className="flex max-h-[80vh] flex-col gap-3 xl:absolute xl:inset-0 xl:max-h-none">
              <div className="flex max-h-[60vh] min-h-0 flex-col overflow-hidden xl:max-h-[30rem]">
                {/* PAS DE `marks` ICI (2026-08-25) : les fiches ne portent plus de glyphe
                    d'identité. La table reste servie à la CARTE (forme du point) et au FIL
                    (glyphe devant un nom), ses deux derniers lecteurs. */}
                <ReplayTeams
                  doc={data}
                  scoreboard={scoreboard}
                  frame={frame}
                  locale={locale}
                  xuidMeta={xuidMeta}
                  header={matchView?.header}
                />
              </div>
              <div className="flex min-h-0 flex-1 flex-col xl:min-h-[12rem]">
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
