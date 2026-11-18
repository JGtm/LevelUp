/**
 * TacticalPilulesFond — LES TROIS COMMANDES DU FOND du bandeau du plan : la pilule « Fond » (Plan, Rendu, Relief), le
 * bouton « Traits du relief » et la pilule « Étage ». Elles lisent ET écrivent les réglages du rejeu 2D (le style
 * du fond et les traits, une seule clé chacun) : choisir le relief ici le choisit au rejeu, et inversement ; la coupe
 * par étage, elle, est propre à ce plan.
 *
 * Chaque commande n'est posée que si elle commande quelque chose : la pilule « Fond » exige au moins deux fonds
 * (un plan ou un relief en plus du rendu), les traits exigent un relief lu, l'étage exige un maillage (D-7).
 */
import { ReliefBackgroundPill, ReliefFloorPill, ReliefLinesButton } from '@/components/map-relief/ReliefPills'
import { RELIEF_VIEW_TEXT } from '@/components/map-relief/text'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalRelief } from './useTacticalRelief'

export function TacticalPilulesFond({ locale, relief }: { locale: Locale; relief: TacticalRelief }) {
  const text = RELIEF_VIEW_TEXT[locale]
  return (
    <>
      <ReliefBackgroundPill
        text={text}
        options={relief.options}
        selected={relief.selected}
        onSelect={relief.setStyle}
        meshPercent={relief.meshPercent}
        testId="tactical-plan-fond-pilule"
      />
      <ReliefLinesButton
        text={text}
        available={relief.linesAvailable}
        show={relief.linesOn}
        onToggle={relief.toggleLines}
        testId="tactical-plan-traits"
      />
      <ReliefFloorPill text={text} floor={relief.floor} testId="tactical-plan-etage" />
    </>
  )
}
