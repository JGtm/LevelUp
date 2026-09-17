/**
 * ExplorerBriefingWeapons — bloc « Arme favorite » du bandeau de briefing.
 *
 * L'arme favorite est celle qui a produit le PLUS DE FRAGS sur la sélection — ni la plus
 * tenue, ni la plus tirée. Le backend en sert une ou deux, déjà triées, avec le nombre de
 * frags qu'il a su rattacher à une arme nommée : la note de couverture n'apparaît que
 * lorsque ce compte reste sous le total de frags de la sélection (une source de dégât peut
 * créditer autant que l'API du titre, auquel cas il n'y a rien à dire au lecteur).
 *
 * TROIS FORMES, décidées par `favoriteWeaponSlots` (ExplorerBriefing.logic) à partir de
 * comptes de lignes, jamais d'une mesure du DOM :
 *   - deux emplacements → carte de deux armes avec leur barre ;
 *   - un emplacement    → carte d'une arme avec sa barre ;
 *   - aucun             → forme compacte : UNE ligne nue (libellé + frags, sans barre ni
 *     chrome de carte), pour que la rangée « Par… » ne gagne qu'une ligne et jamais plus.
 *
 * Le bloc est une LISTE `flex flex-col` et jamais une grille à colonnes nommées : le test
 * DP-3 vise la DERNIÈRE grille portant cette classe et tomberait sur celle-ci. Le nom du
 * fichier porte « Briefing » pour rester sous les garde-rails qui filtrent sur ce motif.
 * Couleurs : `fragClassColor` seul (barre teintée par la classe de l'arme).
 */
import { fragClassColor } from '@/lib/accessibility/scales'
import { intlLocale } from '@/lib/formatters'
import type { ExplorerBriefingWeapons, SynthesisWeaponKillEntry } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { BriefingSectionCard } from './BriefingSectionCard'

type T = (key: ExplorerManifestKey, values?: Record<string, string | number>) => string

/** Note « N frags mesurés sur M », ou null quand tout le scope est rattaché à une arme. */
function coverageNote(weapons: ExplorerBriefingWeapons, t: T): string | null {
  if (weapons.measured_kills >= weapons.scope_kills) return null
  return t('explorer.briefing.weapons_coverage', {
    n: weapons.measured_kills,
    m: weapons.scope_kills,
  })
}

/** Ligne d'arme : libellé tronqué, frags alignés à droite, barre fine teintée par classe. */
function WeaponRow({
  weapon,
  maxKills,
  locale,
}: {
  weapon: SynthesisWeaponKillEntry
  maxKills: number
  locale: Locale
}) {
  const pct = Math.round((weapon.kills / maxKills) * 100)
  return (
    <li className="flex flex-col gap-1">
      <div className="flex items-baseline justify-between gap-2">
        <span className="min-w-0 truncate text-xs font-medium text-foreground">{weapon.label}</span>
        <span className="flex-shrink-0 text-xs font-semibold tabular-nums text-foreground">
          {weapon.kills.toLocaleString(intlLocale(locale))}
        </span>
      </div>
      <div className="h-1 w-full overflow-hidden rounded-full bg-muted-foreground/15">
        <div
          className="h-full rounded-full"
          style={{ width: `${pct}%`, backgroundColor: fragClassColor(weapon.class) }}
        />
      </div>
    </li>
  )
}

/**
 * Forme compacte : une seule ligne, sans carte ni barre — le titre du bloc, l'arme, ses
 * frags, et la note de couverture à la suite. C'est le repli sûr quand la rangée n'a
 * aucune place libre : elle ne coûte qu'une ligne.
 */
function CompactLine({
  weapon,
  note,
  t,
  locale,
}: {
  weapon: SynthesisWeaponKillEntry
  note: string | null
  t: T
  locale: Locale
}) {
  return (
    <div className="flex flex-wrap items-baseline gap-x-1.5 text-xs">
      <span className="text-2xs uppercase tracking-wide text-muted-foreground">
        {t('explorer.briefing.weapons_title')}
      </span>
      <span className="min-w-0 truncate font-medium text-foreground">{weapon.label}</span>
      <span className="font-semibold tabular-nums text-foreground">
        {weapon.kills.toLocaleString(intlLocale(locale))}
      </span>
      {note != null && <span className="text-3xs text-muted-foreground">{note}</span>}
    </div>
  )
}

/**
 * Bloc « Arme favorite ». `slots` vient de `favoriteWeaponSlots` : il choisit la FORME,
 * jamais la présence — l'omission du module est décidée par le backend (aucun frag mesuré).
 */
export function FavoriteWeaponBlock({
  weapons,
  slots,
  t,
  locale,
}: {
  weapons: ExplorerBriefingWeapons
  slots: 0 | 1 | 2
  t: T
  locale: Locale
}) {
  // Une arme au minimum : la forme compacte en montre une, pas zéro.
  const entries = (weapons.entries ?? []).slice(0, Math.max(1, slots))
  if (entries.length === 0) return null
  const note = coverageNote(weapons, t)
  if (slots === 0) {
    return <CompactLine weapon={entries[0]} note={note} t={t} locale={locale} />
  }
  const maxKills = Math.max(1, ...entries.map((w) => w.kills))
  return (
    <BriefingSectionCard className="h-full" title={t('explorer.briefing.weapons_title')}>
      <ul className="flex flex-col gap-2">
        {entries.map((w, i) => (
          <WeaponRow key={`${w.label}-${i}`} weapon={w} maxKills={maxKills} locale={locale} />
        ))}
      </ul>
      {note != null && <p className="mt-2 text-3xs text-muted-foreground">{note}</p>}
    </BriefingSectionCard>
  )
}
