/**
 * ReplaySoundControls — l'interrupteur du son et son volume, dans la barre des calques.
 *
 * Rien ici que de l'affichage : l'état, la persistance et le déclenchement vivent dans
 * useReplaySound. Deux règles de la barre s'appliquent, celles du bouton des zones :
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
        className={sound.mutedBySpeed ? 'h-7 px-2 text-xs opacity-60' : 'h-7 px-2 text-xs'}
        title={sound.mutedBySpeed ? t.soundFastHint : t.soundHint}
        aria-pressed={sound.on}
      >
        {t.sound}
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
