/**
 * ResourceMapGridCard — « Contrôle des ressources, carte par carte » (Séries temporelles › Usages,
 * bloc « Carte par carte » ; maquette v4, `gridColumns` / `renderGrid`).
 *
 * Une colonne par carte jouée, la plus jouée à gauche, puis « Autres cartes » qui somme le reste
 * (calcul Go). En-tête : le nom, « n cartes · » pour la colonne de repli, le nombre de matchs, le
 * bilan « x V · y D (· z A) » aux couleurs de résultat. La table (lignes, cases, râteliers repliés,
 * frags aux armes spéciales) est celle de l'Emprise (`ResourceGridTable`) ; l'infobulle d'une case
 * s'ouvre sur « Aquarius (12 matchs, 9 filmés) » et dit qui l'a prise chez moi.
 */
import { useMemo } from 'react'

import { MINUS_INK, PLUS_INK, TRACK_INK } from '@/features/squad/formes/colors'
import type { GridRow, GridWho } from '@/features/squad/emprise/emprise.logic'
import type { EmpriseText } from '@/features/squad/emprise/empriseStrings'
import { ResourceGridTable, type GridColumn } from '@/features/squad/emprise/ResourceGridTable'
import { ObjectifFrame, ObjectifLegend } from '@/features/squad/objectif/ObjectifFrame'
import { tokenCssVar } from '@/lib/accessibility'

import type { MapColumnInfo, MapGrid } from './usages.logic'
import type { UsagesCardsText } from './usagesCardsText'

interface Props {
  grid: MapGrid
  itemName: (row: GridRow) => string
  /** Le nom du joueur affiché par xuid (vide = inconnu). */
  playerName: (xuid: string) => string
  t: EmpriseText
  ut: UsagesCardsText
}

export function ResourceMapGridCard({ grid, itemName, playerName, t, ut }: Props) {
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={t.grid.title}
        items={[
          { kind: 'square', label: t.grid.more, color: `color-mix(in oklab, ${PLUS_INK} 80%, var(--muted))` },
          { kind: 'square', label: t.grid.less, color: `color-mix(in oklab, ${MINUS_INK} 80%, var(--muted))` },
          { kind: 'square', label: t.grid.nothing, color: TRACK_INK },
        ]}
      />
    ),
    [t],
  )
  const columns: GridColumn[] = grid.columns.map((c) => {
    const name = c.name || ut.maps.others
    return { key: c.key, head: <MapHead c={c} name={name} ut={ut} />, tipHead: ut.maps.tipHead(name, c.matches) }
  })
  // Moi d'abord, puis le reste du camp (maquette « Chez moi : JGtm 3, reste du camp 2 »).
  const whoText = (who: GridWho[]) =>
    [...who]
      .sort((a, b) => Number(a.xuid == null) - Number(b.xuid == null))
      .map((w) => `${(w.xuid && playerName(w.xuid)) || t.grid.restLower} ${w.taken}`)
      .join(', ')
  return (
    <ObjectifFrame title={t.grid.title} info={t.grid.info} legend={legend} testId="usages-map-grid">
      <ResourceGridTable columns={columns} sections={grid.sections} itemName={itemName} whoText={whoText} t={t} />
    </ObjectifFrame>
  )
}

function MapHead({ c, name, ut }: { c: MapColumnInfo; name: string; ut: UsagesCardsText }) {
  return (
    <div className="pb-[3px] text-center text-[11px] leading-tight text-muted-foreground" data-testid={`usages-map-head-${c.key}`}>
      <b className="block text-xs font-medium text-foreground">{name}</b>
      {c.otherMaps > 0 && ut.maps.otherMapsFmt(c.otherMaps)}
      {ut.maps.matchesFmt(c.matches)}
      <div className="tabular-nums">
        <span style={{ color: tokenCssVar('outcome-win') }}>{ut.maps.winsFmt(c.wins)}</span>
        {' · '}
        <span style={{ color: tokenCssVar('outcome-loss') }}>{ut.maps.lossesFmt(c.losses)}</span>
        {c.others > 0 && ` · ${ut.maps.othersFmt(c.others)}`}
      </div>
    </div>
  )
}
