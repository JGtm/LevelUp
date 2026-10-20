/**
 * DESTINATION : apps/web/src/features/match-replay/ReplaySpeedMenu.tsx
 *
 * ReplaySpeedMenu — LA VITESSE EN MENU (demande utilisateur du 2026-08-27 : « la vitesse en
 * menu déroulant plutôt que 4 boutons »). Quatre boutons côte à côte occupaient la place de
 * quatre commandes pour un seul réglage, et trois des quatre étaient toujours inutiles.
 *
 * LE DÉCLENCHEUR MONTRE LA VALEUR COURANTE (« 1× »), pas une étiquette muette : c'est ce qui
 * remplace l'information que les quatre boutons donnaient d'un coup d'œil. Le menu ne se rend
 * qu'ouvert ; fermé, il ne coûte rien.
 *
 * LES MÊMES VALEURS ET LA MÊME PERSISTANCE qu'avant (`SPEED_MULTIPLIERS`, useReplaySettings) —
 * aucun réglage n'est réinventé ici. La note « son coupé » sur 4× reprend la règle existante
 * du son (au-delà de 2× les sons se chevaucheraient : cf. useReplaySound).
 */
import { useEffect, useRef, useState } from 'react'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { SPEED_MULTIPLIERS } from './useReplaySettings'

/** Au-delà de ce multiplicateur le son se tait (règle de useReplaySound). */
const SOUND_MAX_SPEED = 2

interface ReplaySpeedMenuProps {
  speed: number
  onSetSpeed: (speed: number) => void
  locale: ReplayLocale
}

export function ReplaySpeedMenu({ speed, onSetSpeed, locale }: ReplaySpeedMenuProps) {
  const t = REPLAY_TEXT[locale]
  const [open, setOpen] = useState(false)
  const wrapRef = useRef<HTMLDivElement>(null)

  // Mêmes sorties que le tiroir de réglages : Échap et clic dehors (`pointerdown`, pas
  // `click` — le menu part au geste, pas au relâché).
  useEffect(() => {
    if (!open) return
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    function onPointerDown(e: PointerEvent) {
      const target = e.target as Node | null
      if (target && wrapRef.current?.contains(target)) return
      setOpen(false)
    }
    window.addEventListener('keydown', onKeyDown)
    document.addEventListener('pointerdown', onPointerDown)
    return () => {
      window.removeEventListener('keydown', onKeyDown)
      document.removeEventListener('pointerdown', onPointerDown)
    }
  }, [open])

  return (
    <div ref={wrapRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        aria-label={t.speed}
        title={t.speed}
        className="inline-flex h-8 cursor-pointer items-center gap-1.5 rounded-full border border-input px-3 text-[12.5px] font-medium tabular-nums transition-colors hover:bg-accent"
      >
        {fmt(speed)}
        <ChevronGlyph />
      </button>
      {open && (
        <div
          role="group"
          aria-label={t.speed}
          className="absolute bottom-[calc(100%+6px)] right-0 z-20 flex min-w-[110px] flex-col rounded-lg border border-border bg-popover p-1 shadow-xl"
        >
          {SPEED_MULTIPLIERS.map((m) => (
            <button
              key={m}
              type="button"
              onClick={() => {
                onSetSpeed(m)
                setOpen(false)
              }}
              aria-pressed={speed === m}
              className={`flex h-7 cursor-pointer items-center gap-1.5 rounded px-2 text-left text-[12.5px] tabular-nums transition-colors ${
                speed === m ? 'bg-accent text-accent-foreground' : 'hover:bg-accent'
              }`}
            >
              {fmt(m)}
              {m === 1 && <span className="text-[11px] text-muted-foreground">{t.speedNormal}</span>}
              {m > SOUND_MAX_SPEED && (
                <span className="text-[10px] text-muted-foreground">{t.speedMuted}</span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

/** Même mise en forme que les anciens boutons : 0.5× garde sa décimale, 1×/2×/4× non. */
function fmt(m: number): string {
  return m < 1 ? `${m.toFixed(1)}×` : `${m.toFixed(0)}×`
}

/** Chevron du déclencheur. Décoratif. */
function ChevronGlyph() {
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-2.5 w-2.5"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M4 6.5 8 10.5 12 6.5" />
    </svg>
  )
}
