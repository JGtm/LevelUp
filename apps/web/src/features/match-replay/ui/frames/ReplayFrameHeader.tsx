/**
 * ReplayFrameHeader — LES EN-TÊTES DES TROIS CADRES de la page du rejeu, et le groupe de trois boutons
 * « Cadre du lecteur » qu'ils portent tous.
 *
 *  - STANDARD : le fil d'Ariane de l'application, dont le dernier segment est le titre de la page
 *    (avec le badge de version de schéma de l'administrateur), le rappel du match, puis le groupe de
 *    cadres et le lien vers la fiche du match ;
 *  - CINÉMA : une ligne réduite, à l'intérieur de la page : retour, titre du match sur deux lignes,
 *    groupe de cadres et lien vers la fiche ;
 *  - PLEIN CADRE : aucune navigation de l'application (`showsAppNavigation`) ; une barre où « Quitter »
 *    (Échap) tient la place du retour, suivie du titre du match et du groupe de cadres.
 *
 * Le groupe reflète le cadre courant (`aria-pressed`) et envoie les gestes du modèle (`selectFrame`) ;
 * ses trois boutons sont inertes pendant un export vidéo. « Quitter », lui, reste actif pendant un export
 * (la sortie du Plein cadre ne touche pas le clip) et referme aussi le plein écran du système quand il
 * est actif, comme le fait Échap.
 */
import { Link, useRouter } from '@tanstack/react-router'

import { MatchBreadcrumb } from '@/features/match-view/MatchHeader'
import { themedIconSrc } from '@/lib/themedIcon'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

import { REPLAY_TEXT } from '../../i18n/i18n'
import { showsAppNavigation, type ReplayFrame } from '../../model/replayFrame'
import { ReplayMatchRecall } from '../ReplayMatchRecall'
import { ReplaySchemaBadge } from '../ReplaySchemaBadge'
import { PRESS_GUARDS } from '../replayPressGuards'
import type { ReplayFrameChrome } from './replayFrameTypes'
import { selectFrame } from './selectFrame'

const FRAMES: readonly ReplayFrame[] = ['standard', 'cinema', 'plein-cadre']

/** Le bouton de lien vers la fiche du match : même gabarit dans les deux en-têtes qui le portent. */
const SHEET_LINK_CLASS =
  'inline-flex h-7 items-center justify-center gap-2 rounded-md border border-border bg-transparent px-2.5 text-xs font-medium transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring'

const SHEET_ROUTE = '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId'

/** Le groupe de trois boutons qui choisit le cadre du lecteur. */
function FrameSwitch({ frame, chrome }: { frame: ReplayFrame; chrome: ReplayFrameChrome }) {
  const { frameControl, locale } = chrome
  const labels = REPLAY_TEXT[locale].frames
  return (
    <div
      role="group"
      aria-label={labels.groupLabel}
      className="inline-flex gap-0.5 rounded-lg border border-border bg-card p-0.5"
    >
      {FRAMES.map((id) => (
        <button
          key={id}
          type="button"
          {...PRESS_GUARDS}
          aria-pressed={frame === id}
          disabled={frameControl.disabled}
          onClick={() => selectFrame(frameControl, id)}
          className={`inline-flex h-7 items-center rounded-md px-2.5 text-xs transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 ${
            frame === id ? 'bg-accent font-medium text-accent-foreground' : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          {labels.frame[id]}
        </button>
      ))}
    </div>
  )
}

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

/** L'en-tête du Standard : le fil d'Ariane de l'application, le titre de la page et le badge de schéma. */
function StandardHeader({ chrome }: { chrome: ReplayFrameChrome }) {
  const isAdmin = useAppShellStore((s) => s.isAdmin)
  // Le logo du rejeu est un raster à deux variantes (noire / blanche) : le thème local choisit.
  const theme = useSettingsDraftStore((s) => s.localUiPrefs.theme)
  const { doc, header, params, locale, matchLabel } = chrome
  const t = REPLAY_TEXT[locale]
  return (
    <MatchBreadcrumb
      playerSlug={params.playerSlug}
      matchLabel={matchLabel}
      locale={locale}
      leaf={
        <h1 className="flex items-center gap-1.5 text-sm font-semibold">
          <img src={themedIconSrc('replay', theme)} alt="" aria-hidden className="h-4 w-auto" />
          {t.title}
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
        </h1>
      }
      detail={
        <ReplayMatchRecall startTimeLabel={header?.start_time_label} playlistLabel={header?.playlist_label} />
      }
      action={
        <span className="flex items-center gap-2">
          <FrameSwitch frame="standard" chrome={chrome} />
          {/* « Fiche du match », PAS « retour au match » : la flèche du fil fait `history.back()`, donc
              n'importe quoi ; ce lien vise une destination fixe, qu'on en vienne ou non. */}
          <Link to={SHEET_ROUTE} params={params} className={SHEET_LINK_CLASS}>
            {t.back}
          </Link>
        </span>
      }
    />
  )
}

/** L'en-tête du Cinéma : retour, titre du match, groupe de cadres, lien vers la fiche. */
function CinemaHeader({ chrome }: { chrome: ReplayFrameChrome }) {
  const t = REPLAY_TEXT[chrome.locale]
  return (
    <div className="flex h-[52px] items-center gap-3 px-6">
      <BackButton chrome={chrome} />
      <MatchTitle chrome={chrome} />
      <div className="ml-auto flex shrink-0 items-center gap-2">
        <FrameSwitch frame="cinema" chrome={chrome} />
        <Link to={SHEET_ROUTE} params={chrome.params} aria-label={t.back} className={SHEET_LINK_CLASS}>
          {t.frames.sheet}
        </Link>
      </div>
    </div>
  )
}

/** La barre du Plein cadre : « Quitter » (Échap), titre du match, groupe de cadres. Aucune navigation de l'app. */
function FullFrameBar({ chrome }: { chrome: ReplayFrameChrome }) {
  const { frameControl, fullscreen, locale } = chrome
  const labels = REPLAY_TEXT[locale].frames
  function quit() {
    // Comme Échap : le plein écran du système, s'il est atteint, se referme avec le cadre.
    frameControl.exit()
    if (fullscreen.active) fullscreen.toggle()
  }
  return (
    <div className="flex h-[52px] shrink-0 items-center gap-3 px-1">
      <button
        type="button"
        {...PRESS_GUARDS}
        onClick={quit}
        className="inline-flex h-8 shrink-0 items-center gap-2 rounded-md border border-border bg-card px-3 text-sm transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" className="h-4 w-4" aria-hidden="true">
          <path d="M18 6L6 18M6 6l12 12" />
        </svg>
        {labels.exit}
        <kbd className="rounded border border-border px-1.5 text-xs text-muted-foreground">{labels.escapeKey}</kbd>
      </button>
      <MatchTitle chrome={chrome} />
      <div className="ml-auto shrink-0">
        <FrameSwitch frame="plein-cadre" chrome={chrome} />
      </div>
    </div>
  )
}

export function ReplayFrameHeader({ frame, chrome }: { frame: ReplayFrame; chrome: ReplayFrameChrome }) {
  if (!showsAppNavigation(frame)) return <FullFrameBar chrome={chrome} />
  return frame === 'standard' ? <StandardHeader chrome={chrome} /> : <CinemaHeader chrome={chrome} />
}
