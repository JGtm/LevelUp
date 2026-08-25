/**
 * ReplaySoundControls — l'interrupteur du son et son volume. Il a vécu dans la barre des
 * calques (jusqu'au 16/08), puis au tiroir de réglages ; depuis le 2026-08-24 il vit à la
 * BARRE DE LECTURE (ReplayTransport) — « c'est plus simple si c'est au niveau de la
 * lecture » — et l'interrupteur est une ICÔNE haut-parleur (barrée quand le son est coupé),
 * son libellé porté par aria-label/title.
 *
 * Rien ici que de l'affichage : l'état, la persistance et le déclenchement vivent dans
 * useReplaySound. Deux règles s'appliquent, les mêmes que pour le bouton des zones :
 * pas de commande quand il n'y a rien à commander (une piste sans un seul son ne montre
 * pas d'interrupteur), et un interrupteur qui n'agit pas en ce moment le DIT (à vitesse
 * rapide le son se tait — le bouton s'estompe et l'infobulle explique) plutôt que de
 * laisser croire à une panne.
 */
import { Button } from '@/components/ui/button'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { ReplaySound } from './useReplaySound'

interface ReplaySoundControlsProps {
  sound: ReplaySound
  locale: ReplayLocale
}

export function ReplaySoundControls({ sound, locale }: ReplaySoundControlsProps) {
  const t = REPLAY_TEXT[locale]
  if (!sound.available) return null
  return (
    <>
      <Button
        variant={sound.on ? 'default' : 'ghost'}
        size="sm"
        onClick={sound.toggle}
        className={sound.mutedBySpeed ? 'h-8 w-9 opacity-60' : 'h-8 w-9'}
        title={sound.mutedBySpeed ? t.soundFastHint : t.soundHint}
        aria-label={t.sound}
        aria-pressed={sound.on}
      >
        <SpeakerIcon muted={!sound.on} />
      </Button>
      {/* Le volume n'apparaît qu'avec le son : un curseur qui ne règle rien encombre. */}
      {sound.on && (
        <input
          type="range"
          min={0}
          max={100}
          step={5}
          value={Math.round(sound.volume * 100)}
          onChange={(e) => sound.setVolume(Number(e.currentTarget.value) / 100)}
          className="h-7 w-16"
          aria-label={t.soundVolume}
          title={t.soundVolume}
        />
      )}
    </>
  )
}

/** Icône haut-parleur : ondes quand le son joue, croix quand il est coupé. */
function SpeakerIcon({ muted }: { muted: boolean }) {
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-5 w-5"
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
