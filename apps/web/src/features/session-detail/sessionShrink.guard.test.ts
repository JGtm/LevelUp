/**
 * Garde-rail — LA COLONNE DE GAUCHE DOIT SUIVRE SA COLONNE, DRAWER DE COMPARAISON OUVERT.
 *
 * LE BUG QUE CE TEST VERROUILLE (rapporté le 2026-09-22, deux fois). Drawer ouvert, les
 * blocs et les graphes de la colonne de gauche gardaient leur largeur de pleine page et
 * passaient sous le drawer, alors que « avant, ça réduisait bien ».
 *
 * DEUX CAUSES SUPERPOSÉES, et la seconde est la vraie.
 *
 * 1. `min-width: auto`. Depuis le lot D16 (`0fb899baa`), l'ouverture du drawer fait passer
 *    la colonne principale de BLOC à GRILLE (`xl:grid` + `xl:grid-rows-subgrid`). Un enfant
 *    de bloc a `min-width: 0` et suit son parent ; un ÉLÉMENT DE GRILLE (ou de flex) a
 *    `min-width: auto` et refuse de descendre sous la largeur intrinsèque de son contenu.
 *    Entre la grille racine et une carte de graphe, TOUT maillon qui est un élément de
 *    grille ou de flex doit donc porter `min-width: 0` — un seul maillon manquant fige la
 *    chaîne entière.
 *
 * 2. LE PALIER SE LIT SUR LA FENÊTRE, PAS SUR LA COLONNE — et c'est ce que le premier
 *    correctif avait manqué. Les cinq rangées de deux graphes décidaient leur mise en page
 *    avec `xl:grid-cols-2`. Or le contenu de l'app est plafonné à 1320 px
 *    (`.app-shell-width`, `styles/globals.css`) : dès 1280 px de fenêtre, `xl:` est vrai
 *    pour toujours et la largeur réelle ne bouge plus. Drawer ouvert, la colonne tombe à
 *    ~660 px pendant que `xl:grid-cols-2` reste vrai → deux pistes de ~294 px pour des
 *    cartes dessinées pleine page. Toutes les AUTRES sections de la colonne
 *    (`SessionPadControlCards`, `SessionUsageEquipmentCards`, `SessionCoordinationSection`,
 *    `SessionFragCard`) s'empilent déjà sur `compact` : les rangées de graphes étaient les
 *    seules à décider sur la fenêtre au lieu de décider sur l'état de la page.
 *
 * CE QUE CE TEST SURVEILLE, ET SEULEMENT ÇA : le gabarit des rangées de graphes (une seule
 * colonne en `compact`, deux sinon) et la présence des classes qui rendent la chaîne
 * rétrécissable. Il ne juge ni la mise en page ni l'espacement (aucun moteur de rendu ici :
 * jsdom ne calcule pas de largeurs).
 */
import { describe, it, expect } from 'vitest'

import { pairGridClass } from './_chartSections'

// import.meta.glob (Vite) charge chaque source comme chaîne brute — même mécanique que
// detail-section.guard.test.ts / section-card.guard.test.ts, pas de dépendance à node:fs.
const sources = import.meta.glob(
  ['/src/features/session-detail/*.tsx', '/src/components/ui/section-card.tsx'],
  { query: '?raw', import: 'default', eager: true },
) as Record<string, string>

function source(path: string): string {
  const code = sources[path]
  expect(code, `source introuvable : ${path} (chemin déplacé ? mettre à jour ce garde-rail)`).toBeTypeOf(
    'string',
  )
  return code
}

describe('garde-rail rétrécissement de la colonne de session (drawer de comparaison)', () => {
  it('le glob ne perd pas ses sources à scanner', () => {
    // Un glob qui ne matche plus rien rendrait le test vert pour de mauvaises raisons.
    expect(Object.keys(sources).length).toBeGreaterThan(30)
  })

  it('une rangée de deux graphes s’empile dès que la colonne est divisée', () => {
    // C'est le correctif du 2026-09-22 : le nombre de colonnes se décide sur `compact`
    // (état de la page), jamais sur un palier de fenêtre.
    expect(pairGridClass(true)).not.toContain('grid-cols-2')
    expect(pairGridClass(false)).toContain('xl:grid-cols-2')
  })

  it('la rangée de deux graphes laisse rétrécir la rangée ET les cartes', () => {
    for (const compact of [true, false]) {
      const cls = pairGridClass(compact)
      expect(cls).toContain('min-w-0')
      // `[&>*]:min-w-0` : les cartes sont les ÉLÉMENTS de cette grille — c'est sur elles que
      // `min-width: auto` mord, pas sur la rangée.
      expect(cls).toContain('[&>*]:min-w-0')
    }
  })

  it('aucune rangée de deux graphes ne réécrit le littéral hors du gabarit', () => {
    const code = source('/src/features/session-detail/_chartSections.tsx')
    // Le littéral d'origine, celui qui manquait les `min-w-0` ET le palier sur `compact`.
    expect(code).not.toMatch(/className="grid gap-6 xl:grid-cols-2"/)
    // Toute grille à deux colonnes de ce module passe par `pairGridClass`.
    const inlineGrids = code.match(/className="[^"]*\bgrid-cols-2\b[^"]*"/g) ?? []
    expect(
      inlineGrids,
      `Rangée de deux graphes écrite à la main : passer par pairGridClass (_chartSections.tsx).`,
    ).toEqual([])
  })

  it('la colonne principale remet à zéro le min-width de ses enfants quand elle devient une grille', () => {
    const code = source('/src/features/session-detail/SessionDetailPage.tsx')
    // La branche `drawerOpen` de la colonne principale : c'est elle qui pose `xl:grid`,
    // donc c'est elle qui doit poser la remise à zéro et le filet de débordement.
    const branch = code.match(/'overflow-x-clip[^']*xl:grid xl:grid-rows-subgrid[^']*'/)
    expect(
      branch,
      "La branche drawerOpen de la colonne principale a changé : vérifier qu'elle porte toujours [&>*]:min-w-0 et overflow-x-clip",
    ).not.toBeNull()
    expect(branch?.[0]).toContain('[&>*]:min-w-0')
    // `clip` et NON `hidden` : `hidden` établirait un conteneur de scroll qui casserait le
    // `position: sticky` du header L3 (même raisonnement que la piste du drawer).
    expect(branch?.[0]).toContain('overflow-x-clip')
    expect(branch?.[0]).not.toContain('overflow-hidden')
  })

  it('chaque rangée partagée de la comparaison peut rétrécir', () => {
    const code = source('/src/features/session-detail/SessionColumnBody.tsx')
    const row = code.match(/data-session-section=\{key\}[\s\S]{0,200}?className="([^"]*)"/)
    expect(row, 'La rangée partagée de la comparaison a changé de forme').not.toBeNull()
    expect(row?.[1]).toContain('min-w-0')
  })

  it('la carte de section partagée porte min-w-0', () => {
    // SectionCard est un ÉLÉMENT de grille dans les rangées à deux cartes des sections
    // Usages / Coordination : sans `min-width: 0` elle refuse de suivre sa piste.
    expect(source('/src/components/ui/section-card.tsx')).toMatch(/relative flex min-w-0 flex-col/)
  })

  it('toutes les sections de la colonne décident leur nombre de colonnes sur compact', () => {
    // Le défaut de fond : UNE section qui garde un palier de fenêtre suffit à faire déborder
    // la colonne, puisque `xl:` reste vrai quand le drawer la coupe en deux (contenu plafonné
    // à 1320 px par `.app-shell-width`).
    const fautifs: string[] = []
    for (const [path, code] of Object.entries(sources)) {
      if (!path.startsWith('/src/features/session-detail/')) continue
      for (const m of code.match(/className=\{?["'`][^"'`]*\b(?:sm|md|lg|xl|2xl):grid-cols-[^"'`]*["'`]/g) ??
        []) {
        fautifs.push(`${path} → ${m}`)
      }
    }
    expect(
      fautifs,
      'Palier de fenêtre figé dans une section de la colonne : le nombre de colonnes doit se décider sur `compact` (cf. pairGridClass).',
    ).toEqual([])
  })
})
