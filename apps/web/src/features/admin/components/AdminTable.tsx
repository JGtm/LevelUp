/**
 * AdminTable / AdminTh / AdminTd — primitives des TABLES NATIVES STATIQUES du
 * dashboard admin (A8.4) : wrapper scrollable + thead caps muted + cellules.
 * Les tables INTERACTIVES (tri/filtre/actions) restent TanStack Table
 * (DetectionsPanel) — ces primitives ne portent aucune interaction.
 */
import type { ReactNode, ThHTMLAttributes, TdHTMLAttributes } from 'react'

/** Conteneur table : scroll horizontal + bordure + thead stylé. */
export function AdminTable({ head, children }: { head: ReactNode; children: ReactNode }) {
  return (
    <div className="overflow-x-auto rounded-md border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
            {head}
          </tr>
        </thead>
        <tbody>{children}</tbody>
      </table>
    </div>
  )
}

/** Cellule d'en-tête (dans le tr fourni par AdminTable). */
export function AdminTh({ children, className = '', ...rest }: ThHTMLAttributes<HTMLTableCellElement>) {
  return (
    <th className={`px-3 py-2 font-medium ${className}`} {...rest}>
      {children}
    </th>
  )
}

/** Ligne standard (hover + séparateur). */
export function AdminTr({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <tr className={`border-b last:border-b-0 hover:bg-muted/30 ${className}`}>{children}</tr>
}

/** Cellule de corps. */
export function AdminTd({ children, className = '', ...rest }: TdHTMLAttributes<HTMLTableCellElement>) {
  return (
    <td className={`px-3 py-2 ${className}`} {...rest}>
      {children}
    </td>
  )
}
