/**
 * columnMeta — augmentation centralisée du type `ColumnMeta` de TanStack Table
 * (V72-04) + helper de rendu d'en-tête pour les tooltips explicatifs de colonnes.
 *
 * Les 8 tableaux TanStack de l'app rendent chacun leur propre `<thead>` avec
 * `flexRender`. Pour éviter d'inventer un champ ad hoc par tableau, on augmente
 * `ColumnMeta` avec `headerTooltip?: string` : n'importe quelle `ColumnDef` peut
 * porter `meta: { headerTooltip: t('...') }`, et chaque `<thead>` enveloppe son
 * libellé dans `<HeaderLabelTooltip text={...}>`.
 *
 * L'aide est portée par le LIBELLÉ lui-même (V73-L2 2.4c) : plus d'icône ⓘ dans
 * les en-têtes, c'est le survol du libellé qui la révèle.
 *
 * NB — le piège qui imposait l'ancienne icône sœur : `InfoTooltip` rend un
 * `<button>` par défaut, et un bouton imbriqué dans le `<button>` de tri est du
 * HTML invalide (le navigateur referme le bouton externe). D'où le mode `trigger`
 * d'`InfoTooltip`, qui enveloppe le libellé dans un `<span>` sans onClick : le
 * clic reste au tri, le survol et le focus ouvrent l'aide.
 */
import { type ReactNode } from 'react'
import { type RowData } from '@tanstack/react-table'
import { InfoTooltip } from '@/components/ui/info-tooltip'

// Augmentation de type globale : ajoute `headerTooltip` à toute `ColumnMeta`.
// On augmente `@tanstack/react-table` (pattern officiel TanStack v8) ; l'interface
// est effectivement déclarée dans `@tanstack/table-core` et re-exportée.
declare module '@tanstack/react-table' {
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  interface ColumnMeta<TData extends RowData, TValue> {
    /** Texte d'aide (1 phrase, FR/EN via manifest i18n) affiché au survol du
     *  LIBELLÉ de l'en-tête. Absent = pas d'aide. */
    headerTooltip?: string
  }
}

/**
 * HeaderLabelTooltip — rend le libellé d'un en-tête de colonne porteur de son
 * aide. Sans `text`, rend `children` inchangé : appelable inconditionnellement
 * dans un `<thead>`, aucun nœud superflu quand la colonne n'a pas d'aide.
 *
 * `focusable` : à poser sur les en-têtes NON triables uniquement (le libellé y
 * est un texte passif → tabIndex=0 + curseur d'aide). Sur un en-tête triable, le
 * `<button>` enveloppé est déjà focusable et son focus ouvre l'aide tout seul.
 */
export function HeaderLabelTooltip({
  text,
  children,
  focusable,
}: {
  text?: string
  children: ReactNode
  focusable?: boolean
}): ReactNode {
  if (!text) return <>{children}</>
  return <InfoTooltip content={text} trigger={children} triggerFocusable={focusable} />
}
