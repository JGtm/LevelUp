/**
 * ReplayFrameHeader — LES EN-TÊTES DES TROIS CADRES de la page du rejeu.
 *
 *  - STANDARD : trois cellules dans la page bornée : retour et titre du match sur deux lignes, bandeau de
 *    score, puis le badge de version de schéma de l'administrateur et le lien vers la fiche du match ;
 *  - CINÉMA : une ligne réduite, à l'intérieur de la page : retour, titre du match sur deux lignes et
 *    lien vers la fiche ;
 *  - PLEIN CADRE : aucune navigation de l'application (`showsAppNavigation`) ; une barre où « Quitter »
 *    (Échap) tient la place du retour, suivie du titre du match.
 * Le Cinéma et le Plein cadre prennent aussi la forme à trois cellules du Standard quand le cadre leur donne un
 * `score` (équipes dans la colonne) : la cellule du centre porte le bandeau, la barre est de 56 px.
 *
 * Aucun en-tête ne porte de sélecteur de cadre : les icônes de cadre sont à droite de la barre de
 * lecture (`ReplayFrameButtons`). « Quitter » reste actif pendant un export (la sortie du Plein cadre
 * ne touche pas le clip) et referme aussi le plein écran du système quand il est actif, comme le fait Échap.
 */
import { Link, useRouter } from '@tanstack/react-router'
import type { ReactNode } from 'react'

import { useAppShellStore } from '@/stores/appShellStore'

import { REPLAY_TEXT } from '../../i18n/i18n'
import { COTES_BANDEAU_SCORE } from '../../model/cotesBandeauScore'
import { margeCorpsDe } from '../../model/cotesCorps'
import { COTES_ENTETE_SCORE } from '../../model/cotesEnteteScore'
import { COTES_PAGE_STANDARD } from '../../model/cotesStandard'
import { showsAppNavigation, type ReplayFrame } from '../../model/replayFrame'
import { ReplaySchemaBadge } from '../ReplaySchemaBadge'
import { PRESS_GUARDS } from '../replayPressGuards'
import type { ReplayFrameChrome } from './replayFrameTypes'
import { leaveFullFrame } from './selectFrame'
import { STANDARD_PAGE_CLASS, STANDARD_PAGE_STYLE } from './standardPage'

/** Le bouton de lien vers la fiche du match : même gabarit dans les deux en-têtes qui le portent. */
const SHEET_LINK_CLASS =
  'inline-flex h-7 items-center justify-center gap-2 rounded-md border border-border bg-transparent px-2.5 text-xs font-medium transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring'

const SHEET_ROUTE = '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId'

/** Le titre du match sur deux lignes : joueur, playlist et date au-dessus, mode et carte dessous. */
function MatchTitle({ chrome }: { chrome: ReplayFrameChrome }) {
  const { header, matchLabel, params, locale } = chrome
  const small = [params.playerSlug, header?.playlist_label, header?.start_time_label].filter(Boolean).join(' · ')
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <p className="truncate text-xs text-muted-foreground">{small}</p>
      <h1 className="truncate text-sm font-semibold">{matchLabel || REPLAY_TEXT[locale].title}</h1>
    </div>
  )
}

/** Le retour de l'historique, ou l'explorateur quand la page est la première de l'onglet. */
function BackButton({ chrome }: { chrome: ReplayFrameChrome }) {
  const router = useRouter()
  const { params, locale } = chrome
  function goBack() {
    if (router.history.length > 1) {
      router.history.back()
      return
    }
    void router.navigate({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/explorer',
      params: { titleSlug: params.titleSlug, playerSlug: params.playerSlug },
    })
  }
  return (
    <button
      type="button"
      onClick={goBack}
      aria-label={REPLAY_TEXT[locale].frames.back}
      className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <svg viewBox="0 0 20 20" fill="currentColor" className="h-4 w-4" aria-hidden="true">
        <path
          fillRule="evenodd"
          d="M17 10a.75.75 0 01-.75.75H5.612l4.158 3.96a.75.75 0 11-1.04 1.08l-5.5-5.25a.75.75 0 010-1.08l5.5-5.25a.75.75 0 111.04 1.08L5.612 9.25H16.25A.75.75 0 0117 10z"
          clipRule="evenodd"
        />
      </svg>
    </button>
  )
}

/**
 * L'en-tête du Standard, en trois cellules de page bornée : à gauche le retour et le titre du match, au
 * centre le bandeau de score (`score`, que le cadre construit ; la cellule garde sa place quand il ne
 * rend rien), à droite le badge de schéma de l'administrateur et le lien vers la fiche. `score` absent
 * (`undefined`) : le corps est trop étroit pour la cellule, l'en-tête tient sur une ligne (retour, titre,
 * actions) et le score est sur la toile.
 */
function StandardHeader({ chrome, score }: { chrome: ReplayFrameChrome; score: ReactNode | undefined }) {
  const isAdmin = useAppShellStore((s) => s.isAdmin)
  const { doc, params, locale } = chrome
  const t = REPLAY_TEXT[locale]
  return (
    <div className={STANDARD_PAGE_CLASS} style={STANDARD_PAGE_STYLE} data-replay-header="standard">
      <div
        className={score === undefined ? 'flex items-center' : 'grid items-center'}
        style={{
          height: COTES_PAGE_STANDARD.enteteHauteurPx,
          columnGap: COTES_PAGE_STANDARD.enteteEcartPx,
          ...(score === undefined
            ? null
            : { gridTemplateColumns: `minmax(0,1fr) ${COTES_BANDEAU_SCORE.entete.largeurPx}px minmax(0,1fr)` }),
        }}
      >
        <div className="flex min-w-0 items-center gap-3" data-replay-header-cell="gauche">
          <BackButton chrome={chrome} />
          <MatchTitle chrome={chrome} />
        </div>
        {score !== undefined && (
          <div className="min-w-0" data-replay-header-cell="centre">
            {score}
          </div>
        )}
        <div
          className={`flex min-w-0 items-center justify-end gap-2${score === undefined ? ' ml-auto shrink-0' : ''}`}
          data-replay-header-cell="droite"
        >
          {/* Rien pour un joueur ordinaire : le badge se gate lui-même sur `isAdmin`. Le document peut
              être absent (chargement, 404) : le badge n'a alors rien à lire. */}
          {doc && (
            <ReplaySchemaBadge
              isAdmin={isAdmin}
              schemaVersion={doc.schemaVersion}
              latestSchemaVersion={doc.latestSchemaVersion}
              contractIssue={doc.contractIssue}
              cadenceStale={doc.cadenceStale}
              locale={locale}
              layers={doc.layers}
            />
          )}
          {/* « Fiche du match », PAS « retour au match » : la flèche du retour fait `history.back()`, donc
              n'importe quoi ; ce lien vise une destination fixe, qu'on en vienne ou non. */}
          <Link to={SHEET_ROUTE} params={params} aria-label={t.back} className={SHEET_LINK_CLASS}>
            {t.frames.sheet}
          </Link>
        </div>
      </div>
    </div>
  )
}

/**
 * La forme à trois cellules d'un en-tête qui porte le score (Cinéma, Plein cadre) : la cellule du centre a la
 * largeur du bandeau à l'emplacement `entete`, les deux autres se partagent le reste.
 */
function ScoreHeader({ frame, left, score, right }: { frame: ReplayFrame; left: ReactNode; score: ReactNode; right: ReactNode }) {
  return (
    <div
      className="grid shrink-0 items-center"
      style={{
        paddingInline: margeCorpsDe(frame),
        height: COTES_ENTETE_SCORE.hauteurPx,
        columnGap: COTES_ENTETE_SCORE.ecartPx,
        gridTemplateColumns: `minmax(0,1fr) ${COTES_BANDEAU_SCORE.entete.largeurPx}px minmax(0,1fr)`,
      }}
      data-replay-header={frame}
      data-replay-header-score=""
    >
      <div className="flex min-w-0 items-center gap-3" data-replay-header-cell="gauche">
        {left}
      </div>
      <div className="min-w-0" data-replay-header-cell="centre">
        {score}
      </div>
      <div className="flex min-w-0 items-center justify-end gap-2" data-replay-header-cell="droite">
        {right}
      </div>
    </div>
  )
}

/** L'en-tête du Cinéma : retour, titre du match, lien vers la fiche ; le score au centre quand le cadre le donne. */
function CinemaHeader({ chrome, score }: { chrome: ReplayFrameChrome; score: ReactNode | undefined }) {
  const t = REPLAY_TEXT[chrome.locale]
  const sheet = (
    <Link to={SHEET_ROUTE} params={chrome.params} aria-label={t.back} className={SHEET_LINK_CLASS}>
      {t.frames.sheet}
    </Link>
  )
  const left = (
    <>
      <BackButton chrome={chrome} />
      <MatchTitle chrome={chrome} />
    </>
  )
  if (score !== undefined) return <ScoreHeader frame="cinema" left={left} score={score} right={sheet} />
  return (
    <div className="flex h-[52px] items-center gap-3" style={{ paddingInline: margeCorpsDe('cinema') }}>
      {left}
      <div className="ml-auto flex shrink-0 items-center gap-2">{sheet}</div>
    </div>
  )
}

/** La barre du Plein cadre : « Quitter » (Échap) et titre du match ; le score au centre quand le cadre le donne. Aucune navigation de l'app. */
function FullFrameBar({ chrome, score }: { chrome: ReplayFrameChrome; score: ReactNode | undefined }) {
  const { frameControl, fullscreen, locale } = chrome
  const labels = REPLAY_TEXT[locale].frames
  const left = (
    <>
      <button
        type="button"
        {...PRESS_GUARDS}
        onClick={() => leaveFullFrame(frameControl, fullscreen)}
        className="inline-flex h-8 shrink-0 items-center gap-2 rounded-md border border-border bg-card px-3 text-sm transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" className="h-4 w-4" aria-hidden="true">
          <path d="M18 6L6 18M6 6l12 12" />
        </svg>
        {labels.exit}
        <kbd className="rounded border border-border px-1.5 text-xs text-muted-foreground">{labels.escapeKey}</kbd>
      </button>
      <MatchTitle chrome={chrome} />
    </>
  )
  if (score !== undefined) return <ScoreHeader frame="plein-cadre" left={left} score={score} right={null} />
  return (
    <div className="flex h-[52px] shrink-0 items-center gap-3" style={{ paddingInline: margeCorpsDe('plein-cadre') }}>
      {left}
    </div>
  )
}

interface ReplayFrameHeaderProps {
  frame: ReplayFrame
  chrome: ReplayFrameChrome
  /**
   * Le bandeau de score de la cellule centrale (`null` : la cellule garde sa place, sans bandeau). Les trois cadres
   * suivent la même règle : `undefined` = l'en-tête d'une ligne, le score est ailleurs (sur la toile, ou dans la rangée).
   */
  score?: ReactNode
}

export function ReplayFrameHeader({ frame, chrome, score }: ReplayFrameHeaderProps) {
  if (!showsAppNavigation(frame)) return <FullFrameBar chrome={chrome} score={score} />
  return frame === 'standard' ? <StandardHeader chrome={chrome} score={score} /> : <CinemaHeader chrome={chrome} score={score} />
}
