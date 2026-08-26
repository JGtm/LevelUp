/**
 * ReplayTransport — LA BARRE DE LECTURE du rejeu : lecture/pause et recommencer en ICÔNES,
 * les multiplicateurs de vitesse, le son, l'horloge et la frise.
 *
 * EXTRAIT DE ReplayCanvas.tsx LE 2026-08-24 (septième extraction imposée par le cliquet de
 * taille), en exécutant trois demandes utilisateur du même jour : « des symboles pour la
 * lecture et recommencer », « les boutons de vitesse direct à côté des boutons lecture »,
 * « pareil pour le réglage du son, c'est plus simple si c'est au niveau de la lecture ».
 * La vitesse et l'interrupteur du son SORTENT donc du tiroir de réglages — le tiroir garde
 * les calques, les effets et le filtre de son par catégorie.
 *
 * LES ICÔNES PORTENT LEUR LIBELLÉ en aria-label/title (les mêmes clés i18n qu'avant) : un
 * symbole sans nom serait une régression d'accessibilité, pas une simplification. SVG
 * inline en currentColor, comme les autres icônes du dépôt (pas de librairie d'icônes).
 *
 * L'ÉTAT NE VIT PAS ICI : la LECTURE (état lu/pause, boucle rAF, curseur de la frise, arrêt
 * sur la dernière image) vit dans `useReplayPlayback` ; l'image courante, l'horloge, la
 * vitesse et le son restent au canvas — cette barre ne fait que les afficher et les commander.
 */
import type { ChangeEvent, ComponentProps, RefObject } from 'react'

import { Button } from '@/components/ui/button'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { ReplayLeadMarks } from './ReplayLeadMarks'
import { ReplaySoundControls } from './ReplaySoundControls'
import { SlidersIcon } from './SlidersIcon'
import { SPEED_MULTIPLIERS } from './useReplaySettings'
import type { ReplayCapture } from './useReplayCapture'
import type { ReplaySound } from './useReplaySound'

interface ReplayTransportProps {
  playing: boolean
  onTogglePlay: () => void
  onRestart: () => void
  /** L'horloge est écrite par la boucle de dessin (textContent), pas par React. */
  clockRef: RefObject<HTMLSpanElement | null>
  /** Le curseur est piloté par la boucle de dessin ; React ne le contrôle pas. */
  sliderRef: RefObject<HTMLInputElement | null>
  maxFrame: number
  onScrub: (e: ChangeEvent<HTMLInputElement>) => void
  speed: number
  onSetSpeed: (speed: number) => void
  sound: ReplaySound
  /**
   * CE QUI SORT DU REJEU (image, vidéo), en UN objet comme le son — et pour la même raison :
   * le canvas vit sous un cliquet de taille, et chaque commande ajoutée ici ne doit pas lui
   * coûter une prop de plus (cf. `useReplayCapture`).
   */
  capture: ReplayCapture
  locale: ReplayLocale
  /** Les marques de retournement, posées SUR la piste (cf. ReplayLeadMarks). */
  leadMarks: ComponentProps<typeof ReplayLeadMarks>
  /**
   * Le bouton du TIROIR DE RÉGLAGES, tout à droite de la barre (convention des lecteurs
   * vidéo : les réglages ferment la barre). L'état et le tiroir restent au canvas ; la ref
   * sert au « clic dehors » du tiroir et au retour de focus à sa fermeture. Props À PLAT
   * (pas un objet) : la règle `react-hooks/refs` prend un accès membre `x.fooRef` en rendu
   * pour une lecture de ref.
   */
  settingsOpen: boolean
  onToggleSettings: () => void
  settingsButtonRef: RefObject<HTMLButtonElement | null>
}

export function ReplayTransport({
  playing, onTogglePlay, onRestart, clockRef, sliderRef, maxFrame, onScrub,
  speed, onSetSpeed, sound, capture, locale, leadMarks,
  settingsOpen, onToggleSettings, settingsButtonRef,
}: ReplayTransportProps) {
  const t = REPLAY_TEXT[locale]
  // L'ORDRE EST CELUI D'UN LECTEUR VIDÉO (retour utilisateur du 2026-08-24 : « réorganise,
  // là c'est le bazar ») : les COMMANDES DE LECTURE à gauche (lecture, recommencer), puis
  // l'horloge, la TIMELINE au centre — c'est elle qui prend la largeur — et à droite les
  // RÉGLAGES de lecture (vitesse, son). L'œil trouve chaque commande là où tous les
  // lecteurs la mettent.
  return (
    <div className="mt-2 flex items-center gap-2">
      <span className="flex items-center gap-1">
        <Button
          variant="default"
          size="sm"
          onClick={onTogglePlay}
          className="h-8 w-9"
          aria-label={playing ? t.pause : t.play}
          title={playing ? t.pause : t.play}
        >
          {playing ? <PauseIcon /> : <PlayIcon />}
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={onRestart}
          className="h-8 w-9"
          aria-label={t.restart}
          title={t.restart}
        >
          <RestartIcon />
        </Button>
      </span>
      <span
        ref={clockRef}
        className="min-w-[5.5rem] font-mono text-xs tabular-nums text-muted-foreground"
        aria-label={t.time}
      />
      {/* LA TIMELINE AU CENTRE, seule à s'étirer ; les marques de retournement se posent
          SUR la piste (cf. ReplayLeadMarks). */}
      <span className="relative flex-1">
        <input
          ref={sliderRef}
          type="range"
          min={0}
          max={maxFrame}
          defaultValue={0}
          onChange={onScrub}
          className="block w-full"
          aria-label={t.time}
        />
        <ReplayLeadMarks {...leadMarks} />
      </span>
      {/* LA VITESSE puis LE SON, à droite : des réglages de lecture, pas des commandes. Le
          filtre de son par catégorie, plus rare, reste au tiroir. */}
      <span className="flex items-center gap-0.5" role="group" aria-label={t.speed}>
        {SPEED_MULTIPLIERS.map((m) => (
          <Button
            key={m}
            type="button"
            variant={speed === m ? 'default' : 'ghost'}
            size="sm"
            onClick={() => onSetSpeed(m)}
            className="h-7 px-1.5 text-xs"
            aria-pressed={speed === m}
          >
            {m < 1 ? `${m.toFixed(1)}×` : `${m.toFixed(0)}×`}
          </Button>
        ))}
      </span>
      <ReplaySoundControls sound={sound} locale={locale} />
      {/* CE QUI SORT DU REJEU, entre le son et les réglages : capturer l'image de la scène.
          La place n'est pas arbitraire — ce sont des commandes de SORTIE, pas des réglages de
          lecture, et un lecteur vidéo les groupe là, juste avant l'engrenage. */}
      <Button
        variant="ghost"
        size="sm"
        onClick={capture.captureImage}
        className="h-8 w-9"
        aria-label={t.captureImage}
        title={t.captureImage}
      >
        <CameraIcon />
      </Button>
      {/* LES RÉGLAGES FERMENT LA BARRE, tout à droite — là où tous les lecteurs les mettent. */}
      <Button
        ref={settingsButtonRef}
        variant={settingsOpen ? 'default' : 'ghost'}
        size="sm"
        onClick={onToggleSettings}
        className="h-8 w-9"
        aria-expanded={settingsOpen}
        aria-label={t.settingsButton}
        title={t.settingsButton}
      >
        <SlidersIcon />
      </Button>
    </div>
  )
}

/** Icône lecture : le triangle. Décorative — le libellé vit sur le bouton. */
function PlayIcon() {
  return (
    <svg viewBox="0 0 16 16" className="h-4 w-4" fill="currentColor" aria-hidden="true">
      <path d="M4.5 2.8a.8.8 0 0 1 1.2-.7l8 5.2a.8.8 0 0 1 0 1.4l-8 5.2a.8.8 0 0 1-1.2-.7z" />
    </svg>
  )
}

/** Icône pause : les deux barres. */
function PauseIcon() {
  return (
    <svg viewBox="0 0 16 16" className="h-4 w-4" fill="currentColor" aria-hidden="true">
      <rect x="3.5" y="2.5" width="3.4" height="11" rx="1" />
      <rect x="9.1" y="2.5" width="3.4" height="11" rx="1" />
    </svg>
  )
}

/** Icône appareil photo : le boîtier et son objectif. Décorative — le libellé vit sur le bouton. */
function CameraIcon() {
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-4 w-4"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M1.8 5.2h2.6l1.1-1.7h4.9l1.1 1.7h2.7v7.3H1.8z" />
      <circle cx="8" cy="8.9" r="2.4" />
    </svg>
  )
}

/** Icône recommencer : la flèche qui reboucle vers le début. */
function RestartIcon() {
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-4 w-4"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M2.5 8a5.5 5.5 0 1 0 1.6-3.9" />
      <path d="M4.1 1.6v2.9H7" />
    </svg>
  )
}
