/**
 * ReplayPlayerPanelList — LES LISTES DU PANNEAU DE JOUEUR : précision, dégâts, détail du score,
 * objectif, armes. Une liste = un titre « sur le match » et des lignes libellé / valeur, comme le
 * tiroir du tableau de score (`PlayerDetailPanel` : libellé en encre discrète à gauche, valeur
 * mono à droite, séparées d'un filet). Pas de mini-carte, pas de courbe : des listes.
 *
 * LES VALEURS SONT DES TOTAUX DE FIN DE MATCH lus sur la feuille (`playerPanelLogic`) : le titre de
 * chaque liste le dit. Une valeur non lue est un TIRET (`NO_VALUE`), jamais un zéro ; une arme hors
 * référentiel écrit son numéro, jamais le nom d'une autre.
 */
import { WeaponIcon } from '@/components/ui/WeaponIcon'

import type { ReplayLocale } from '../i18n/i18n'
import type { PlayerPanelText } from '../i18n/playerPanel.i18n'
import type { PanelRow, PanelSection, PanelWeapon } from '../model/playerPanelLogic'
import { formatPanelValue } from '../model/playerPanelFormat'

/** Largeur et hauteur du slot d'une icône d'arme (le même que le tiroir du tableau de score). */
const WEAPON_ICON_W = 56
const WEAPON_ICON_H = 24

const ROW_CLASS = 'flex items-center justify-between gap-4 border-b border-border/60 py-1 text-xs'

function ValueRow({ row, text, locale }: { row: PanelRow; text: PlayerPanelText; locale: ReplayLocale }) {
  return (
    <li className={ROW_CLASS}>
      <span className="text-muted-foreground">{text.row[row.key]}</span>
      <span className={`font-mono tabular-nums ${row.value === null ? 'text-muted-foreground' : 'text-foreground'}`}>
        {formatPanelValue(row, locale)}
      </span>
    </li>
  )
}

function WeaponRow({ weapon, text }: { weapon: PanelWeapon; text: PlayerPanelText }) {
  const label = weapon.label ?? text.weaponFmt(weapon.weaponId)
  return (
    <li className={ROW_CLASS} title={label}>
      <span className="flex min-w-0 items-center gap-2">
        {weapon.imageUrl && (
          // Repli sur le libellé (écrit à côté) quand l'arme n'a pas d'icône : jamais celle d'une autre.
          <span style={{ width: WEAPON_ICON_W, height: WEAPON_ICON_H }} className="flex shrink-0 items-center justify-center text-foreground">
            <WeaponIcon
              imageUrl={weapon.imageUrl}
              tinted={weapon.tinted}
              label={label}
              width={WEAPON_ICON_W}
              height={WEAPON_ICON_H}
            />
          </span>
        )}
        <span className="truncate text-muted-foreground">{label}</span>
      </span>
      <span className="font-mono tabular-nums text-foreground">{`×${weapon.kills}`}</span>
    </li>
  )
}

/** Une liste du panneau : son titre (« … sur le match ») et ses lignes. */
export function PanelSectionList({
  section, text, locale,
}: { section: PanelSection; text: PlayerPanelText; locale: ReplayLocale }) {
  return (
    <section aria-label={text.section[section.id]}>
      <h4 className="mb-0.5 text-2xs font-semibold uppercase tracking-wide text-foreground">
        {text.section[section.id]}
      </h4>
      <ul>
        {section.weapons
          ? section.weapons.map((w) => <WeaponRow key={w.weaponId} weapon={w} text={text} />)
          : section.rows.map((r) => <ValueRow key={r.key} row={r} text={text} locale={locale} />)}
      </ul>
    </section>
  )
}
