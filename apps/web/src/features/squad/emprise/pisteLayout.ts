/**
 * pisteLayout.ts — la grille des pistes de l'onglet Emprise : une colonne des noms (pastille,
 * ressource, sous-libellé ; maquette `.prow.wide.narrow`, 118 px) puis la colonne des barres.
 * Partagée par `PisteCampsForm` et « Rendement face à l'adversaire » : leurs axes tombent sous
 * leurs barres.
 */
export function pisteColumns(labelWidth = 118): string {
  return `${labelWidth}px minmax(0,1fr)`
}
