/**
 * Garde-rail — LE PLAFOND DES FICHES NE REDEVIENT PAS UN POURCENTAGE DE LA CARTE.
 *
 * CE QU'IL PROTÈGE, et pourquoi un test plutôt qu'un commentaire. Le bloc des fiches était
 * plafonné à `max-h-[62%]` : 62 % de la rangée, dont la hauteur est celle de la colonne carte.
 * Tant que la carte mesurait 480 px, ce pourcentage valait une promesse stable. Depuis que la
 * carte s'adapte à l'écran (2026-09-02), le même pourcentage vaut 479 px sur un poste et 405 px
 * sur un autre — et à 405, un 4v4 (442 px de fiches) est tronqué.
 *
 * Le piège est qu'un pourcentage se relit très bien : « 62 % de la colonne » a l'air d'une
 * décision de mise en page raisonnable. Rien dans sa forme ne dit qu'il est adossé à une
 * hauteur devenue variable. C'est exactement le genre de valeur qu'une revue future
 * réintroduirait de bonne foi — d'où ce test, qui refuse la FORME et pas seulement la valeur.
 *
 * CE QU'IL N'INTERDIT PAS : les pourcentages ailleurs dans la page, ni `max-h-[60vh]` sur la
 * pile empilée sous `xl` (celle-là se mesure contre la FENÊTRE, pas contre la carte : elle ne
 * bouge pas quand le terrain change de taille).
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

const ROUTE = resolve(
  __dirname,
  '../../routes/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay.tsx',
)

describe('garde-rail : la hauteur des fiches ne dépend plus de celle de la carte', () => {
  const src = readFileSync(ROUTE, 'utf8')

  it('aucun plafond exprimé en pourcentage de la rangée sur le bloc des fiches', () => {
    // `xl:max-h-[NN%]` est la forme exacte qui a causé la régression : un pourcentage qui se
    // résout contre la colonne carte, donc contre une hauteur désormais élastique.
    expect(src).not.toMatch(/xl:max-h-\[\d+%\]/)
  })

  it('le plafond borne bien les DEUX cotés : la taille des fiches et la place laissée au fil', () => {
    // Sans le second terme, une colonne courte donnerait tout aux fiches et rien au fil ;
    // sans le premier, un BTB à 24 fiches remplirait toute la colonne.
    expect(src).toContain('xl:max-h-[30rem]')
    expect(src).toContain('xl:min-h-[12rem]')
  })

  // LA LEÇON DU 2026-09-02, ET ELLE A COÛTÉ UNE LIVRAISON POUR RIEN. La première version de ce
  // plafond s'écrivait `xl:max-h-[min(30rem,calc(100%-12rem))]` — une seule classe qui disait
  // les deux bornes. Elle est passée au typecheck, aux tests, au lint et a été livrée : le test
  // ci-dessus vérifiait la CHAÎNE, et la chaîne était bien là. Mais `calc(100%-12rem)` n'est pas
  // du CSS valide (les opérateurs + et - y exigent des espaces), Tailwind ne produisait donc
  // AUCUNE règle, et les fiches se retrouvaient sans plafond du tout.
  //
  // Un test qui grep une chaîne ne peut pas juger du CSS qu'elle produit. Ce qu'il PEUT faire,
  // c'est interdire la construction qui l'a rendu aveugle.
  it('aucun calc() dans une valeur arbitraire — la syntaxe qui a rendu le grep aveugle', () => {
    expect(src).not.toContain('calc(')
  })

  it('sur une colonne courte, c est le PLANCHER du fil qui fait céder les fiches', () => {
    // Les fiches sont rétrécissables (`min-h-0`, plus de `shrink-0`) et le fil porte un
    // plancher : c'est ce couple qui répartit. Avec `shrink-0` sur les fiches, un 4v4 dans une
    // colonne de 400 px prendrait tout et le fil n'aurait rien — le plancher ne servirait
    // alors à rien, puisque rien ne pourrait lui céder la place.
    expect(src).toMatch(/min-h-0[^"]*xl:max-h-\[30rem\]/)
    expect(src).not.toMatch(/shrink-0[^"]*xl:max-h-\[30rem\]/)
  })
})
