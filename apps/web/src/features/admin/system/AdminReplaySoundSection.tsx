/**
 * AdminReplaySoundSection — « Sons du rejeu » (page Admin · Système).
 *
 * DEUX CURSEURS, PAS UN DE PLUS. Les .wav du rejeu 2D sont extraits du jeu tels quels ;
 * ces réglages rejouent côté app ce que le moteur applique à chaque coup — la variation
 * (volume et hauteur, dans les fourchettes déclarées par le jeu) et la distance
 * (atténuation + son plus sourd). Le calcul vit dans features/match-replay/
 * weaponSoundLogic.ts ; cette section ne fait que servir les deux valeurs.
 *
 * RÉGLAGE D'INSTANCE, PAS DE PRÉFÉRENCE UTILISATEUR : sa place est Admin, comme la sync
 * et la sauvegarde. Même câblage autonome qu'AdminSyncSettingsSection (useSettings /
 * useUpdateSettings + getSettingsText), auto-save à chaque relâchement du curseur.
 */
import { useState } from 'react'

import { getSettingsText, normalizeSettingsLocale } from '@/features/settings/i18n'
import { useSettings, useUpdateSettings } from '@/features/settings/queries'
import { useAppShellStore } from '@/stores/appShellStore'
import type { SettingsResponse } from '@/lib/api/types'
import { SectionHeader } from '../components/SectionHeader'

/** Valeurs d'usine, alignées sur le serveur : variation du jeu telle quelle, aucune distance. */
const DEFAUT_VARIATION = 100
const DEFAUT_DISTANCE = 0

export function AdminReplaySoundSection() {
  const { data: settings, isLoading } = useSettings()
  const mutation = useUpdateSettings()
  const locale = normalizeSettingsLocale(useAppShellStore((s) => s.locale))
  const t = getSettingsText(locale)

  // Copie éditable des réglages serveur, resynchronisée quand la requête livre un nouvel
  // objet (pattern « valeur précédente », comme AdminSyncSettingsSection).
  const [local, setLocal] = useState<Partial<SettingsResponse>>({})
  const [prevSettings, setPrevSettings] = useState(settings)
  if (settings && settings !== prevSettings) {
    setPrevSettings(settings)
    setLocal(settings)
  }

  function handleChange<K extends keyof SettingsResponse>(field: K, value: SettingsResponse[K]) {
    setLocal((prev) => ({ ...prev, [field]: value }))
    mutation.mutate({ [field]: value } as Partial<SettingsResponse>)
  }

  if (isLoading) return null

  return (
    <section className="space-y-3">
      <SectionHeader title={t.replaySoundTitle} description={t.replaySoundDescription} />
      <div className="space-y-5 rounded-md border border-border bg-popover p-4">
        <PercentSlider
          label={t.replaySoundVariationLabel}
          hint={t.replaySoundVariationHint}
          minLabel={t.replaySoundOff}
          maxLabel={t.replaySoundFull}
          value={local.replay_sound_variation_percent ?? DEFAUT_VARIATION}
          onCommit={(v) => handleChange('replay_sound_variation_percent', v)}
        />
        <PercentSlider
          label={t.replaySoundDistanceLabel}
          hint={t.replaySoundDistanceHint}
          minLabel={t.replaySoundOff}
          maxLabel={t.replaySoundFull}
          value={local.replay_sound_distance_percent ?? DEFAUT_DISTANCE}
          onCommit={(v) => handleChange('replay_sound_distance_percent', v)}
        />
      </div>
    </section>
  )
}

interface PercentSliderProps {
  label: string
  hint: string
  minLabel: string
  maxLabel: string
  value: number
  /** Appelé au RELÂCHEMENT du curseur, pas à chaque pixel (cf. commentaire ci-dessous). */
  onCommit: (value: number) => void
}

/**
 * PercentSlider — un curseur 0-100 %.
 *
 * L'affichage suit le doigt (état local), mais l'enregistrement n'a lieu qu'au
 * RELÂCHEMENT : un PATCH par pixel parcouru noierait le serveur sous des dizaines de
 * requêtes pour un seul geste. `onChange` reste branché pour le clavier, où chaque flèche
 * est déjà un geste complet.
 */
function PercentSlider({ label, hint, minLabel, maxLabel, value, onCommit }: PercentSliderProps) {
  const [affiche, setAffiche] = useState(value)
  const [prevValue, setPrevValue] = useState(value)
  if (value !== prevValue) {
    setPrevValue(value)
    setAffiche(value)
  }

  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <span className="text-sm text-foreground">{label}</span>
        <span className="text-sm tabular-nums text-muted-foreground">{affiche} %</span>
      </div>
      <input
        type="range"
        min={0}
        max={100}
        step={5}
        value={affiche}
        aria-label={label}
        className="w-full accent-primary"
        onChange={(e) => setAffiche(Number(e.target.value))}
        onMouseUp={(e) => onCommit(Number((e.target as HTMLInputElement).value))}
        onTouchEnd={(e) => onCommit(Number((e.target as HTMLInputElement).value))}
        onKeyUp={(e) => onCommit(Number((e.target as HTMLInputElement).value))}
      />
      <div className="flex justify-between text-2xs text-muted-foreground">
        <span>{minLabel}</span>
        <span>{maxLabel}</span>
      </div>
      <p className="text-xs text-muted-foreground">{hint}</p>
    </div>
  )
}
