/**
 * titleWithHelp — le bandeau de titre d'une carte de la section « Portée des engagements »,
 * et son mode d'emploi derrière une aide ⓘ.
 *
 * Le détail de lecture s'écrivait SOUS le titre de carte, sur une ligne de sous-titre à lui :
 * un titre pour la carte, un second titre pour le même graphe. Depuis le 2026-09-13 chaque
 * graphe a sa carte, et l'aide vit dans le bandeau.
 *
 * DEPUIS LE 2026-09-21 ce n'est plus qu'un RÉGLAGE de l'helper canonique
 * `components/ui/title-with-info` (taille d'icône ⓘ de cette page) : le gabarit lui-même ne
 * se réécrit plus ici. Fichier à part depuis le 2026-09-22 : la carte « Dénivelé » a quitté
 * `WeaponRangeSection.tsx` (plafond de 500 lignes) et les deux cartes lisent le même réglage
 * — recopier l'appel en aurait fait deux, qui auraient divergé.
 */
import { titleWithInfo } from '@/components/ui/title-with-info'

export function titleWithHelp(help: string) {
  return titleWithInfo(help, { iconClass: 'w-3.5 h-3.5' })
}
