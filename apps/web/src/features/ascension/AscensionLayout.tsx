/**
 * AscensionLayout — layout parent à 4 onglets de la section Ascension.
 *
 * Architecture (parcours utilisateur : qui je suis → ce que je vise →
 * comment progresser → ce que j'ai accompli) :
 *   /ascension               → tab "Profil" (index)
 *   /ascension/objectifs     → tab "Objectifs" (couche Prestige)
 *   /ascension/coaching      → tab "Entraînement"
 *   /ascension/realisations  → tab "Réalisations"
 *   /ascension/tactique      → tab "Tactique"
 *
 * L'en-tête (titre, chapeau) et la barre d'onglets sont posés au-dessus par `SectionHeader`
 * (layout joueur), comme pour toutes les sections : les onglets viennent de `L1_SECTIONS`. Ce
 * layout ne rend que le bandeau d'astuces TipsTicker, sous l'en-tête, puis le contenu de l'onglet
 * (<Outlet />).
 *
 * L'onglet « Tactique » est derrière la capacité `replay` (déclarée sur son onglet de `L1_SECTIONS`) :
 * un titre sans décodage de film ne produit aucune position mesurée, donc aucune lecture de
 * placement ; l'onglet n'apparaît pas plutôt que de mener à une page morte. C'est la PREMIÈRE des deux
 * portes ; la seconde est sur la route elle-même (cf. `features/tactical/TacticalTab`), parce
 * qu'une URL peut être ouverte directement sans passer par la barre.
 */
import { useMemo } from 'react'
import { Outlet } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { TipsTicker } from '@/components/ui/tips-ticker'
import { buildAscensionTips } from './tips'
import { getAscensionText } from './i18n'

function LightbulbIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="12"
      height="12"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M9 18h6" />
      <path d="M10 22h4" />
      <path d="M12 2a7 7 0 0 0-4 12.7c.7.6 1 1.5 1 2.3h6c0-.8.3-1.7 1-2.3A7 7 0 0 0 12 2z" />
    </svg>
  )
}

export function AscensionLayout() {
  const locale = useAppShellStore((s) => s.locale)
  const t = getAscensionText(locale)
  const tips = useMemo(() => buildAscensionTips(locale), [locale])

  return (
    <main className="space-y-6 p-6">
      <TipsTicker
        tips={tips}
        ariaLabel={t.tipsTickerAriaLabel}
        leadingIcon={<LightbulbIcon />}
      />

      <Outlet />
    </main>
  )
}
