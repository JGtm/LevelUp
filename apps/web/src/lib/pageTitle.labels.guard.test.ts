// @vitest-environment node
/**
 * Garde-rail — les titres d'onglet navigateur SUIVENT le dictionnaire de la feature.
 *
 * POURQUOI (CLAUDE.md regle n°6 : une copie sans garde-rail re-diverge). La table
 * `PLAYER_SUFFIX_OVERRIDES` de `pageTitle.ts` recopie a la main les libelles FR/EN des
 * barres d'onglets des features — son propre commentaire l'assume (« reprennent
 * VERBATIM les traductions canoniques deja etablies ailleurs »). Le garde-rail voisin
 * `pageTitle.test.ts` ne verifie que l'EXHAUSTIVITE des routes (un titre non-fallback
 * existe), jamais l'EGALITE des libelles : ajouter la sous-route `squad/usages`
 * (2026-09-22) a donc exige d'ecrire « Usages » / « Usage » a deux endroits sans rien
 * qui relie les deux. Ce fichier pose le lien manquant.
 *
 * REGLE : la TABLE suit la FEATURE, jamais l'inverse. La barre d'onglets reellement
 * affichee lit `getSquadText(locale).nav.*` (`features/squad/SquadLayout.tsx`) : c'est
 * la source unique. Quand ce test rougit, on corrige `pageTitle.ts`.
 *
 * FRONTIERE : l'import de `features/` est fait ICI, dans un test. Le code de production
 * de `lib/` ne doit PAS importer `features/` — d'ou la copie de libelles qu'on se
 * contente de VERROUILLER plutot que de supprimer.
 *
 * PERIMETRE (lot XS, 2026-09-22) : les 4 sous-routes Escouade seulement. Les autres
 * barres d'onglets n'ont pas de source unique lisible en egalite stricte :
 *  - Carriere : `common.nav.tab_citations` vaut { fr: 'Citations', en: 'Commendations' }
 *    alors que la table distingue volontairement /career/citations (en: 'Citations',
 *    moteur derive Infinite) de /career/commendations (en: 'Commendations', totaux
 *    natifs H5) — la nuance est portee par la ROUTE, documentee dans `pageTitle.ts` :
 *    une egalite stricte y serait fausse.
 *  - Ascension (ajoute le 2026-09-23) : les titres de la table sont PREFIXES
 *    (« Ascension — Objectifs »), la relation est donc « prefixe + `common.nav.tab_*` »,
 *    pas l'egalite stricte. C'est la relation verrouillee ci-dessous pour les quatre
 *    sous-routes : la premiere divergence attrapee etait `tab_tactique` = 'Tactical'
 *    quand la table disait 'Tactics' — un mot que l'onglet n'a jamais porte.
 *    `/ascension` (onglet « Profil ») reste titre « Ascension » sans prefixe : c'est
 *    l'entree de la rubrique, pas un onglet de plus ; hors garde-rail.
 */
import { describe, it, expect } from 'vitest'
import { resolvePageTitle } from './pageTitle'
import { getSquadText } from '@/features/squad/i18n'
import { commonManifest } from './i18n/generated/common'
import type { Locale } from './i18n/locale'

const LOCALES: readonly Locale[] = ['fr', 'en']

/** Suffixe de route Escouade -> cle du dictionnaire `nav` de la feature. */
const SQUAD_TAB_SOURCES = [
  { suffix: '/squad/synergies', navKey: 'synergies' },
  { suffix: '/squad/contributions', navKey: 'contributions' },
  { suffix: '/squad/dynamique', navKey: 'dynamique' },
  { suffix: '/squad/usages', navKey: 'usages' },
] as const

describe('garde-rail : les titres de page des onglets Escouade suivent features/squad/i18n.ts', () => {
  it.each(SQUAD_TAB_SOURCES)('$suffix === nav.$navKey (FR + EN)', ({ suffix, navKey }) => {
    for (const locale of LOCALES) {
      const expected = getSquadText(locale).nav[navKey]
      expect(
        resolvePageTitle(`/t/halo_infinite/players/x${suffix}`, locale),
        `locale=${locale} suffix=${suffix} : la TABLE pageTitle.ts doit s'aligner sur features/squad/i18n.ts (nav.${navKey} = "${expected}"), jamais l'inverse`,
      ).toBe(`LevelUp - ${expected}`)
    }
  })

  it('le jeu de sous-routes couvertes est exhaustif vis-a-vis du dictionnaire nav', () => {
    // Une cle `nav` ajoutee a la feature sans entree ici (donc probablement sans entree
    // dans `pageTitle.ts`) fait echouer ce test.
    expect(Object.keys(getSquadText('fr').nav).sort()).toEqual(
      SQUAD_TAB_SOURCES.map((s) => s.navKey).slice().sort(),
    )
  })
})

/**
 * Suffixe de route Ascension -> cle `common.nav.*` que l'onglet L1/L2 affiche reellement
 * (`components/shell/navL1Sections.tsx`). Le titre de page est le libellé de l'onglet
 * PREFIXE par la rubrique, dans les deux langues.
 */
const ASCENSION_TITLE_PREFIX = 'Ascension — '
const ASCENSION_TAB_SOURCES = [
  { suffix: '/ascension/objectifs', navKey: 'common.nav.tab_objectives' },
  { suffix: '/ascension/coaching', navKey: 'common.nav.tab_coaching' },
  { suffix: '/ascension/realisations', navKey: 'common.nav.tab_realisations' },
  { suffix: '/ascension/tactique', navKey: 'common.nav.tab_tactique' },
] as const

describe('garde-rail : les titres de page des onglets Ascension = prefixe + common.nav.*', () => {
  it.each(ASCENSION_TAB_SOURCES)('$suffix === prefixe + $navKey (FR + EN)', ({ suffix, navKey }) => {
    for (const locale of LOCALES) {
      const expected = `${ASCENSION_TITLE_PREFIX}${commonManifest[navKey][locale]}`
      expect(
        resolvePageTitle(`/t/halo_infinite/players/x${suffix}`, locale),
        `locale=${locale} suffix=${suffix} : la TABLE pageTitle.ts doit s'aligner sur le manifeste common (${navKey} = "${commonManifest[navKey][locale]}"), jamais l'inverse`,
      ).toBe(`LevelUp - ${expected}`)
    }
  })
})
