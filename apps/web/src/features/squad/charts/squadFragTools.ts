/**
 * squadFragTools — « Outils de destruction » de l'Escouade : les lignes `weapon_tools` du
 * serveur (décision D8 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26), nommées dans la
 * langue de l'interface et mises à la forme du graphe à barres groupées.
 *
 * Le serveur a déjà tout décidé : une ligne par clé d'arme (grenades par type quand le film
 * les type, sinon une ligne « Grenade » au total de la feuille), la
 * mêlée de la feuille de match, les objets explosifs et la chute d'après la catégorie de
 * source du film, le reliquat « Non attribué » en dernier. Ce module NE regroupe RIEN et ne
 * plafonne RIEN (plus de « Autres armes ») : il nomme.
 *
 *   - une arme porte son nom de registre, FR d'abord (`label`) ou EN d'abord (`label_en`)
 *     selon la locale ;
 *   - les autres natures sont nommées ici, par clé (`kind`) — le serveur n'écrit aucun
 *     libellé (ratchet `no_french_label_literal_test.go`).
 */
import type { SquadWeaponToolLine, SquadWeaponTools } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import type { SquadBarRow, SquadBarRows } from './squadWeaponKillsChart'

/** Libellés des natures sans nom de registre (i18n de l'appelant). */
export interface SquadToolKindLabels {
  melee: string
  /** Grenades de la feuille, en une ligne, quand le film ne les type pas (titre sans film). */
  grenade: string
  assassination: string
  ground_pound: string
  shoulder_bash: string
  explosive_object: string
  environment: string
  unattributed: string
}

/** Nom affiché d'une ligne. Nature inconnue du web → sa clé brute (jamais une ligne vide). */
export function toolLineLabel(line: SquadWeaponToolLine, locale: Locale, labels: SquadToolKindLabels): string {
  if (line.kind === 'weapon') {
    const fr = line.label ?? ''
    const en = line.label_en ?? ''
    return (locale === 'en' ? en || fr : fr || en) || (line.weapon_key ?? '')
  }
  return labels[line.kind as keyof SquadToolKindLabels] ?? line.kind
}

/**
 * Lignes du graphe. Le serveur les rend du plus gros total au plus petit (« Non attribué »
 * en dernier) ; le graphe pose la PREMIÈRE catégorie EN BAS, d'où l'ordre renversé : le
 * BR75 de la soirée se lit en haut, le reliquat tout en bas.
 */
export function buildSquadToolRows(
  tools: SquadWeaponTools | null | undefined,
  opts: { locale: Locale; labels: SquadToolKindLabels },
): SquadBarRows | null {
  const players = tools?.players ?? []
  const lines = tools?.lines ?? []
  if (players.length === 0 || lines.length === 0) return null
  const rows: SquadBarRow[] = lines.map((l) => ({
    key: l.kind === 'weapon' ? `weapon:${l.weapon_key ?? l.label ?? ''}` : l.kind,
    label: toolLineLabel(l, opts.locale, opts.labels),
    cls: l.class,
    killsByPlayer: l.kills_by_player,
    total: l.total_squad,
  }))
  return { players, rows: rows.reverse() }
}
