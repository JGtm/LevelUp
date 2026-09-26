/**
 * ReplaySoundControls — l'interrupteur du son et son volume, à la barre de lecture.
 *
 * SEUL L'HABILLAGE CHANGE (planche 2a du 2026-08-28) : bouton ROND comme le reste du transport,
 * curseur de volume habillé au lieu du `input[type=range]` nu. Les DEUX RÈGLES de l'ancien
 * fichier restent, mot pour mot :
 *
 *  - pas de commande quand il n'y a rien à commander (une piste sans un seul son ne montre pas
 *    d'interrupteur) ;
 *  - un interrupteur qui n'agit pas en ce moment le DIT (à vitesse rapide le son se tait — le
 *    bouton s'estompe et l'infobulle explique) plutôt que de laisser croire à une panne.
 *
 * ET LE CURSEUR NE S'ESCAMOTE TOUJOURS PAS quand le son est coupé (demande du 2026-08-25) : il
 * tombe à zéro, s'estompe, et son infobulle dit que le niveau réglé revient avec le son. Le
 * niveau lui-même survit à la coupure — `sound.volume` est l'état de préférence, que la bascule
 * ne touche pas ; le zéro affiché est un affichage, jamais une écriture.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { ReplaySound } from './useReplaySound'

interface ReplaySoundControlsProps {
  sound: ReplaySound
  locale: ReplayLocale
}

export function ReplaySoundControls({ sound, locale }: ReplaySoundControlsProps) {
  const t = REPLAY_TEXT[locale]
  if (!sound.available) return null
  const level = sound.on ? Math.round(sound.volume * 100) : 0
  return (
    <div className="flex items-center gap-2">
      <button
        type="button"
        onClick={sound.toggle}
        aria-label={t.sound}
        aria-pressed={sound.on}
        title={sound.mutedBySpeed ? t.soundFastHint : `${t.soundHint} (M)`}
        className={`inline-flex h-[34px] w-[34px] cursor-pointer items-center justify-center rounded-full transition-colors hover:bg-accent ${
          sound.mutedBySpeed ? 'opacity-60' : ''
        } ${sound.on ? '' : 'text-muted-foreground'}`}
      >
        <SpeakerIcon muted={!sound.on} />
      </button>
      <input
        type="range"
        min={0}
        max={100}
        step={5}
        value={level}
        disabled={!sound.on}
        onChange={(e) => sound.setVolume(Number(e.currentTarget.value) / 100)}
        aria-label={t.soundVolume}
        title={sound.on ? t.soundVolume : t.soundVolumeMutedHint}
        style={{ '--played': `${level}%` } as React.CSSProperties}
        className={`h-3 w-[58px] cursor-pointer appearance-none bg-transparent
          [&::-webkit-slider-runnable-track]:h-1 [&::-webkit-slider-runnable-track]:rounded-full
          [&::-webkit-slider-runnable-track]:bg-[linear-gradient(to_right,var(--muted-foreground)_0_var(--played),var(--input)_var(--played)_100%)]
          [&::-webkit-slider-thumb]:-mt-[4px] [&::-webkit-slider-thumb]:h-3 [&::-webkit-slider-thumb]:w-3
          [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:rounded-full
          [&::-webkit-slider-thumb]:bg-foreground
          [&::-moz-range-track]:h-1 [&::-moz-range-track]:rounded-full
          [&::-moz-range-track]:bg-[linear-gradient(to_right,var(--muted-foreground)_0_var(--played),var(--input)_var(--played)_100%)]
          [&::-moz-range-thumb]:h-3 [&::-moz-range-thumb]:w-3 [&::-moz-range-thumb]:border-0
          [&::-moz-range-thumb]:rounded-full [&::-moz-range-thumb]:bg-foreground
          ${sound.on ? '' : 'opacity-60'}`}
      />
    </div>
  )
}

/** Icône haut-parleur : ondes quand le son joue, croix quand il est coupé. */
function SpeakerIcon({ muted }: { muted: boolean }) {
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-4 w-4"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M1.5 5.6h2.6L8.3 2v12L4.1 10.4H1.5z" fill="currentColor" strokeWidth="0.8" />
      {muted ? (
        <path d="M10.6 5.7 15 10.3M15 5.7l-4.4 4.6" />
      ) : (
        <>
          <path d="M10.6 5.1a4.1 4.1 0 0 1 0 5.8" />
          <path d="M12.8 2.9a7.2 7.2 0 0 1 0 10.2" />
        </>
      )}
    </svg>
  )
}
