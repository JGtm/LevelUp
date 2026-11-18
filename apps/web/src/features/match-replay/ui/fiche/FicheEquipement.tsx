/**
 * FicheEquipement — « ÉQUIPEMENT » : un tableau par famille, trois colonnes — pris, utilisé, perdu (lâché en
 * mourant). Les familles sont celles de la grille de la page du match, dans son ordre, et se nomment comme elle
 * (`equipmentFamilyLabel`) ; une famille dont aucun objet n'est pris jusque-là n'a pas de ligne. Un zéro d'une
 * colonne est écrit : l'objet existe, il n'a pas été utilisé ou perdu. Le rendu ne lit rien.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../../i18n/i18n'
import { COTES_FICHE_INSTANT } from '../../model/cotesFicheInstant'
import { equipmentFamilyLabel } from '../../model/equipmentUsageColumns'
import { formatCount } from '../../model/ficheFormat'
import type { EquipmentFamilyStats } from '../../model/playerEquipmentStats'
import { FicheSection } from './ficheParts'

const C = COTES_FICHE_INSTANT.tableauEquipement
const ROW_STYLE = { gridTemplateColumns: C.colonnes, columnGap: C.ecartPx, fontSize: C.taillePx }

export function FicheEquipement({ families, locale }: { families: readonly EquipmentFamilyStats[]; locale: ReplayLocale }) {
  const text = REPLAY_TEXT[locale]
  const t = text.playerInstant
  return (
    <FicheSection title={t.equipment}>
      <div role="table" aria-label={t.equipment} className="flex flex-col" style={{ rowGap: COTES_FICHE_INSTANT.section.ecartPx }}>
        <div role="row" className="grid items-center text-muted-foreground" style={{ ...ROW_STYLE, fontSize: C.tailleEntetePx }}>
          <span role="columnheader" />
          <span role="columnheader" className="text-right">{t.equipmentTaken}</span>
          <span role="columnheader" className="text-right">{t.equipmentUsed}</span>
          <span role="columnheader" className="text-right">{t.equipmentLost}</span>
        </div>
        {families.map((f) => (
          <div key={f.family} role="row" className="grid items-center text-foreground" style={ROW_STYLE}>
            <span role="cell" className="truncate">{equipmentFamilyLabel(f.family, text)}</span>
            <span role="cell" className="text-right tabular-nums">{formatCount(f.taken, locale)}</span>
            <span role="cell" className="text-right tabular-nums">{formatCount(f.used, locale)}</span>
            <span role="cell" className="text-right tabular-nums">{formatCount(f.dropped, locale)}</span>
          </div>
        ))}
      </div>
    </FicheSection>
  )
}
