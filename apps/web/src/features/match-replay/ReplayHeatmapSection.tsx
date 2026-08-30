/**
 * ReplayHeatmapSection — la section CARTE DE CHALEUR du tiroir de réglages.
 *
 * EXTRAITE DE `ReplaySettingsDrawer.tsx` LE 2026-08-18 (lot R3, item R3.7) : le tiroir a
 * gagné une section de plus — les FICHES — et franchissait le seuil de 500 lignes du dépôt.
 * C'est la carte de chaleur qui part, parce qu'elle est la seule section à porter DEUX axes
 * de choix (ce qu'on mesure, sur quelle durée) : elle pèse à elle seule autant que les
 * quatre autres, et c'est la seule dont l'affichage dépend de ce que le film porte.
 */
import { SettingsChoice, SettingsToggle } from './ReplaySettingsToggle'

import type { HeatmapMode, HeatmapSpan } from './heatmapLayer'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { HEATMAP_MODES, HEATMAP_SPANS } from './useReplaySettings'

/** Ce que le tiroir sait de la carte de chaleur : son état, et ce qu'elle peut mesurer. */
export interface ReplayHeatmapControls {
  show: boolean
  onToggle: () => void
  mode: HeatmapMode
  onSetMode: (mode: HeatmapMode) => void
  /** La PORTÉE DE TEMPS (V2, 2026-08-18) : toute la partie, ou jusqu'à l'image courante. */
  span: HeatmapSpan
  onSetSpan: (span: HeatmapSpan) => void
  /** Faux quand aucune mort du match n'a pu être localisée : la lecture « éliminations »
   *  ne commande alors rien et n'est pas proposée (même règle que le bouton Zones). */
  killsAvailable: boolean
}

/**
 * La CARTE DE CHALEUR a sa propre section : c'est un calque, mais qui porte un CHOIX de
 * lecture (ce qu'on mesure). Le noyer dans la liste des calques mettrait ce choix au même
 * rang qu'une bascule, alors qu'il change la grandeur affichée. Le choix ne s'affiche que
 * lorsque le calque est allumé — sinon il commanderait quelque chose d'invisible.
 */
export function HeatmapSection({
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
        {/* LES DEUX LECTURES SONT UN CHOIX EXCLUSIF, pas deux interrupteurs : d'ou
            `SettingsChoice` et non `SettingsToggle` depuis le 2026-08-29 (cf. l'en-tete du
            composant). Le CALQUE lui-meme, au-dessus, est bien un oui/non. */}
        {heatmap.show && modes.length > 1 && (
          <>
            <p className="pt-1 text-xs text-muted-foreground">{t.heatmapReading}</p>
            {modes.map((m) => (
              <SettingsChoice
                key={m}
                label={t.heatmapMode[m]}
                pressed={heatmap.mode === m}
                onToggle={() => heatmap.onSetMode(m)}
                hint={t.heatmapModeHint[m]}
              />
            ))}
          </>
        )}
        {/* LA PORTÉE est un second choix, distinct de la lecture : « ce qu'on mesure » et
            « sur quelle durée » sont deux questions, et les mettre en une seule liste ferait
            croire à quatre calques là où il y a deux axes. */}
        {heatmap.show && (
          <>
            <p className="pt-1 text-xs text-muted-foreground">{t.heatmapSpanTitle}</p>
            {HEATMAP_SPANS.map((s) => (
              <SettingsChoice
                key={s}
                label={t.heatmapSpan[s]}
                pressed={heatmap.span === s}
                onToggle={() => heatmap.onSetSpan(s)}
                hint={t.heatmapSpanHint[s]}
              />
            ))}
          </>
        )}
      </div>
    </section>
  )
}
