/**
 * ReplaySettingsDrawer — LE TIROIR DE RÉGLAGES du rejeu (décision utilisateur du 16/08) :
 * calques, son (+ filtre par catégorie), vitesse. Regroupe ce qui vivait éparpillé dans la
 * barre du canvas — AUCUN réglage n'est réinventé ici, chacun garde sa règle et sa
 * persistance (calques/vitesse : useReplaySettings ; son et catégories : useReplaySound).
 *
 * PANNEAU LATÉRAL, PAS UNE MODALE. ReplayCanvas le place en frère du canvas dans une rangée
 * flex : il POUSSE la carte au lieu de la recouvrir, pour ne jamais la masquer pendant la
 * lecture (exigence explicite du 16/08). Pas de fond assombri, pas de piège de focus : la
 * lecture continue derrière, le panneau se ferme par son bouton, le déclencheur, ou Échap.
 *
 * DÉCOUPÉ EN TROIS SECTIONS (Layers/Speed/Sound), chacune sa propre fonction : un seul
 * corps de composant pour les trois dépassait le seuil de lisibilité (CLAUDE.md n°5,
 * fonction ≤ 80 lignes) sans y gagner en clarté — trois blocs indépendants s'y prêtent mieux.
 */
import { useEffect } from 'react'

import { Button } from '@/components/ui/button'

import type { HeatmapMode } from './heatmapLayer'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { ReplaySoundControls } from './ReplaySoundControls'
import { SOUND_CATEGORIES } from './replaySound'
import { HEATMAP_MODES, SPEED_MULTIPLIERS } from './useReplaySettings'
import type { ReplaySound } from './useReplaySound'

/** Ce que le tiroir sait de la carte de chaleur : son état, et ce qu'elle peut mesurer. */
export interface ReplayHeatmapControls {
  show: boolean
  onToggle: () => void
  mode: HeatmapMode
  onSetMode: (mode: HeatmapMode) => void
  /** Faux quand aucune mort du match n'a pu être localisée : la lecture « éliminations »
   *  ne commande alors rien et n'est pas proposée (même règle que le bouton Zones). */
  killsAvailable: boolean
}

interface ReplaySettingsDrawerProps {
  locale: ReplayLocale
  onClose: () => void
  showAim: boolean
  onToggleAim: () => void
  showZones: boolean
  onToggleZones: () => void
  showNames: boolean
  onToggleNames: () => void
  /** Le calque zones n'existe que si la carte a des zones nommées (même règle que le
   *  bouton d'origine : un interrupteur qui ne commande rien tromperait plus qu'il n'informe). */
  zonesAvailable: boolean
  heatmap: ReplayHeatmapControls
  sound: ReplaySound
  speed: number
  onSetSpeed: (speed: number) => void
}

/**
 * Une ligne de bascule : même gabarit pour les calques et les catégories de son — six
 * usages dans ce seul fichier, un seul rendu plutôt que six copies presque identiques
 * (CLAUDE.md règle 6, « à la 3e copie, centraliser »).
 */
function SettingsToggle({
  label, pressed, onToggle, hint,
}: {
  label: string
  pressed: boolean
  onToggle: () => void
  hint?: string
}) {
  return (
    <Button
      type="button"
      variant={pressed ? 'default' : 'ghost'}
      size="sm"
      onClick={onToggle}
      className="h-7 justify-start px-2 text-xs"
      title={hint}
      aria-pressed={pressed}
    >
      {label}
    </Button>
  )
}

interface LayersSectionProps {
  locale: ReplayLocale
  showAim: boolean
  onToggleAim: () => void
  showZones: boolean
  onToggleZones: () => void
  showNames: boolean
  onToggleNames: () => void
  zonesAvailable: boolean
}

function LayersSection({
  locale, showAim, onToggleAim, showZones, onToggleZones, showNames, onToggleNames, zonesAvailable,
}: LayersSectionProps) {
  const t = REPLAY_TEXT[locale]
  return (
    <section className="space-y-1">
      <h3 className="text-xs font-medium text-muted-foreground">{t.layers}</h3>
      <div className="flex flex-col gap-1">
        <SettingsToggle label={t.layerAim} pressed={showAim} onToggle={onToggleAim} hint={t.layerAimHint} />
        <SettingsToggle
          label={t.layerNames}
          pressed={showNames}
          onToggle={onToggleNames}
          hint={t.layerNamesHint}
        />
        {zonesAvailable && (
          <SettingsToggle
            label={t.layerZones}
            pressed={showZones}
            onToggle={onToggleZones}
            hint={t.layerZonesHint}
          />
        )}
      </div>
    </section>
  )
}

/**
 * La CARTE DE CHALEUR a sa propre section : c'est un calque, mais qui porte un CHOIX de
 * lecture (ce qu'on mesure). Le noyer dans la liste des calques mettrait ce choix au même
 * rang qu'une bascule, alors qu'il change la grandeur affichée. Le choix ne s'affiche que
 * lorsque le calque est allumé — sinon il commanderait quelque chose d'invisible.
 */
function HeatmapSection({
  locale, heatmap,
}: {
  locale: ReplayLocale
  heatmap: ReplayHeatmapControls
}) {
  const t = REPLAY_TEXT[locale]
  const modes = heatmap.killsAvailable ? HEATMAP_MODES : HEATMAP_MODES.filter((m) => m !== 'kills')
  return (
    <section className="space-y-1">
      <h3 className="text-xs font-medium text-muted-foreground">{t.layerHeatmap}</h3>
      <div className="flex flex-col gap-1">
        <SettingsToggle
          label={t.layerHeatmap}
          pressed={heatmap.show}
          onToggle={heatmap.onToggle}
          hint={t.layerHeatmapHint}
        />
        {heatmap.show && modes.length > 1 && (
          <>
            <p className="pt-1 text-xs text-muted-foreground">{t.heatmapReading}</p>
            {modes.map((m) => (
              <SettingsToggle
                key={m}
                label={t.heatmapMode[m]}
                pressed={heatmap.mode === m}
                onToggle={() => heatmap.onSetMode(m)}
                hint={t.heatmapModeHint[m]}
              />
            ))}
          </>
        )}
      </div>
    </section>
  )
}

function SpeedSection({
  locale, speed, onSetSpeed,
}: {
  locale: ReplayLocale
  speed: number
  onSetSpeed: (speed: number) => void
}) {
  const t = REPLAY_TEXT[locale]
  return (
    <section className="space-y-1">
      <h3 className="text-xs font-medium text-muted-foreground">{t.speed}</h3>
      <div className="flex flex-wrap gap-1">
        {SPEED_MULTIPLIERS.map((m) => (
          <Button
            key={m}
            type="button"
            variant={speed === m ? 'default' : 'ghost'}
            size="sm"
            onClick={() => onSetSpeed(m)}
            className="h-7 px-2 text-xs"
            // La vitesse en cours est dite, pas seulement peinte : sans `aria-pressed` les
            // quatre boutons s'annoncent identiques a un lecteur d'ecran, alors que les
            // bascules voisines (SettingsToggle) le portent toutes.
            aria-pressed={speed === m}
          >
            {m < 1 ? `${m.toFixed(1)}×` : `${m.toFixed(0)}×`}
          </Button>
        ))}
      </div>
    </section>
  )
}

/** Le son n'apparaît qu'avec au moins un événement sonore dans ce match : même règle que
 *  partout ailleurs dans la barre — pas de commande qui ne commande rien. */
function SoundSection({ locale, sound }: { locale: ReplayLocale; sound: ReplaySound }) {
  const t = REPLAY_TEXT[locale]
  if (!sound.available) return null
  return (
    <section className="space-y-2">
      <h3 className="text-xs font-medium text-muted-foreground">{t.sound}</h3>
      <div className="flex flex-wrap items-center gap-1">
        <ReplaySoundControls sound={sound} locale={locale} />
      </div>
      <h3 className="text-xs font-medium text-muted-foreground">{t.soundCategoriesTitle}</h3>
      <div className="flex flex-col gap-1">
        {SOUND_CATEGORIES.map((category) => (
          <SettingsToggle
            key={category}
            label={t.soundCategory[category]}
            pressed={sound.categories[category]}
            onToggle={() => sound.toggleCategory(category)}
          />
        ))}
      </div>
    </section>
  )
}

export function ReplaySettingsDrawer({
  locale,
  onClose,
  showAim,
  onToggleAim,
  showZones,
  onToggleZones,
  showNames,
  onToggleNames,
  zonesAvailable,
  heatmap,
  sound,
  speed,
  onSetSpeed,
}: ReplaySettingsDrawerProps) {
  const t = REPLAY_TEXT[locale]

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [onClose])

  return (
    <div
      role="region"
      aria-label={t.settingsButton}
      className="flex w-64 shrink-0 flex-col gap-4 overflow-y-auto border-l border-border bg-card px-3 py-3 text-sm"
    >
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold">{t.settingsButton}</h2>
        <button
          type="button"
          onClick={onClose}
          className="flex h-6 w-6 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          aria-label={t.settingsClose}
        >
          ×
        </button>
      </div>

      <LayersSection
        locale={locale}
        showAim={showAim}
        onToggleAim={onToggleAim}
        showZones={showZones}
        onToggleZones={onToggleZones}
        showNames={showNames}
        onToggleNames={onToggleNames}
        zonesAvailable={zonesAvailable}
      />
      <HeatmapSection locale={locale} heatmap={heatmap} />
      <SpeedSection locale={locale} speed={speed} onSetSpeed={onSetSpeed} />
      <SoundSection locale={locale} sound={sound} />
    </div>
  )
}
