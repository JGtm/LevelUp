/**
 * PAIR_ROW_CLASS — LA RANGÉE DE DEUX CARTES, et il n'y en a qu'une.
 *
 * Règle produit (page match, 2026-09-22 ; Escouade, Séries temporelles) : aucun bloc seul à
 * largeur partielle. Deux cartes se partagent la rangée en grand écran ; quand il n'en reste
 * qu'une (l'autre est retirée faute de donnée), elle prend toute la rangée (`only-child`).
 *
 * Cinq copies du même littéral avaient fini par exister, et la sixième — la rangée « Frags par
 * ressource | Rendement » de la Vue match — l'avait OUBLIÉ : le graphe resté seul n'occupait que
 * la moitié. `pairRow.guard.test.ts` interdit le littéral hors de ce fichier (CLAUDE.md n°6).
 */
export const PAIR_ROW_CLASS = 'grid grid-cols-1 gap-4 lg:grid-cols-2 lg:[&>*:only-child]:col-span-2'
