/**
 * replaySchemaStatusLogic.ts — LE STATUT DU BADGE ADMIN « version de schéma » (lot A,
 * 2026-09-11), pur et testé sans React.
 *
 * DEUX NOMBRES, JAMAIS CONFONDUS : `schemaVersion` (le corps) est celle de l'ARTEFACT LU,
 * figée à la cuisson ; `latestSchemaVersion` (l'en-tête `X-Replay-Latest-Schema-Version`,
 * posé à la frontière transport dans `lib/replay/queries.ts`) est celle que le PRODUCTEUR
 * écrirait s'il cuisait ce match maintenant (`games/halo_infinite/film/replay.SchemaVersion` côté Go). Le
 * document jumeau servi (`domain/replaydoc`) ne porte aucun numéro de version — cf. son
 * `doc.go` — cette seconde valeur ne peut donc venir que du transport, jamais du corps.
 *
 * QUATRE ÉTATS (2026-09-13, lot 0.B — le quatrième est neuf) : un artefact antérieur au lot A
 * (ou une réponse dont l'en-tête aurait été filtré en route) ne permet AUCUNE comparaison — le
 * badge doit alors dire ce qu'il sait (la version lue) sans jamais affirmer « à jour » ou
 * « à recuire » sur une absence ; et un document qui ne respecte pas le contrat se dit
 * INVALIDE, ce qui prime sur toute question de version.
 */

/**
 * MIN_RENDERABLE_SCHEMA_VERSION — LA VERSION MINIMALE QUE CE CLIENT DÉCLARE SAVOIR AFFICHER.
 *
 * D'OÙ VIENT CE NOMBRE, ET IL EST VÉRIFIABLE SUR PIÈCES. La chronique du producteur
 * (`replay/document_chronicle.go`) ne porte que DEUX montées qui RETIRENT au client un champ
 * qu'on lui avait donné : v6 (`Inventory.a`, retiré plutôt que réinterprété — il portait
 * `rang − 16`) et v27 (`weaponChanges[].until`, une durée de table refusée par l'utilisateur).
 * À partir de 27, TOUTE montée est additive ou ne change que le contenu : un artefact plus
 * ancien se rend, plus pauvre, jamais faux. En deçà, le document appartient à une génération
 * de contrat dont des champs promis n'existent plus — ce client ne prétend pas l'afficher.
 *
 * CE QUE CETTE CONSTANTE NE DIT PAS : « tout ce qui est au-dessus est à jour ». Un artefact de
 * version 30 est parfaitement affichable ET parfaitement périmé ; c'est la comparaison avec la
 * version du producteur qui tranche cela, pas celle-ci.
 *
 * QUAND LA MONTER : le jour où une montée RETIRE ou RÉINTERPRÈTE un champ que le web lit. Ce
 * jour-là, et seulement ce jour-là. Le test de contrat vérifie qu'aucune fixture produite par
 * Go n'est en dessous (`test/contract/goFixtures.contract.test.ts`).
 */
export const MIN_RENDERABLE_SCHEMA_VERSION = 27

export type ReplaySchemaStatus =
  | { kind: 'unknown'; schemaVersion: number }
  | { kind: 'upToDate'; schemaVersion: number }
  | { kind: 'stale'; schemaVersion: number; latestSchemaVersion?: number }
  | { kind: 'invalid'; schemaVersion: number; issue: string }

/**
 * computeReplaySchemaStatus compare la version de l'artefact lu à celle du producteur, et dit
 * d'abord si le document respecte seulement le contrat.
 *
 * L'ORDRE DES TROIS QUESTIONS EST LE PROPOS DE CETTE FONCTION :
 *
 *  1. le document est-il CONFORME (`contractIssue`, posé par la frontière de transport) ? Un
 *     document non conforme se signale POUR CE QU'IL EST : sa version n'apprend rien, et le
 *     rendu qui suit est au mieux partiel ;
 *  2. est-il en deçà de ce que ce client déclare savoir afficher
 *     (`MIN_RENDERABLE_SCHEMA_VERSION`) ? Alors « à recuire », MÊME SANS en-tête à comparer —
 *     c'est le seul cas où l'absence de comparaison ne doit PAS donner « inconnu » ;
 *  3. sinon, la comparaison ordinaire.
 *
 * `latestSchemaVersion` peut être inférieure à `schemaVersion` sur un poste de dev qui
 * lirait un artefact plus récent que son propre binaire — ce cas se traite comme « à jour »
 * (rien à recuire), pas comme une anomalie : ce module ne juge que l'égalité, pas le sens
 * de l'écart.
 */
export function computeReplaySchemaStatus(
  schemaVersion: number,
  latestSchemaVersion: number | undefined,
  contractIssue?: string,
): ReplaySchemaStatus {
  if (contractIssue) {
    return { kind: 'invalid', schemaVersion, issue: contractIssue }
  }
  if (schemaVersion < MIN_RENDERABLE_SCHEMA_VERSION) {
    // `latestSchemaVersion` est RECOPIÉE telle quelle, absente comprise : le badge nomme une
    // cible quand il en connaît une, et se tait sinon. Mettre le minimum à sa place ferait
    // passer un seuil de compatibilité pour la version du producteur.
    return { kind: 'stale', schemaVersion, latestSchemaVersion }
  }
  if (latestSchemaVersion === undefined) {
    return { kind: 'unknown', schemaVersion }
  }
  if (latestSchemaVersion > schemaVersion) {
    return { kind: 'stale', schemaVersion, latestSchemaVersion }
  }
  return { kind: 'upToDate', schemaVersion }
}
