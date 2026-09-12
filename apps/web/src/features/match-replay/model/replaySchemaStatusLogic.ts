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
 * TROIS ÉTATS, ET PAS DEUX : un artefact antérieur à ce lot (ou une réponse dont l'en-tête
 * aurait été filtré en route) ne permet AUCUNE comparaison — le badge doit alors dire ce
 * qu'il sait (la version lue) sans jamais affirmer « à jour » ou « à recuire » sur une
 * absence.
 */

export type ReplaySchemaStatus =
  | { kind: 'unknown'; schemaVersion: number }
  | { kind: 'upToDate'; schemaVersion: number }
  | { kind: 'stale'; schemaVersion: number; latestSchemaVersion: number }

/**
 * computeReplaySchemaStatus compare la version de l'artefact lu à celle du producteur.
 *
 * `latestSchemaVersion` peut être inférieure à `schemaVersion` sur un poste de dev qui
 * lirait un artefact plus récent que son propre binaire — ce cas se traite comme « à jour »
 * (rien à recuire), pas comme une anomalie : ce module ne juge que l'égalité, pas le sens
 * de l'écart.
 */
export function computeReplaySchemaStatus(
  schemaVersion: number,
  latestSchemaVersion: number | undefined,
): ReplaySchemaStatus {
  if (latestSchemaVersion === undefined) {
    return { kind: 'unknown', schemaVersion }
  }
  if (latestSchemaVersion > schemaVersion) {
    return { kind: 'stale', schemaVersion, latestSchemaVersion }
  }
  return { kind: 'upToDate', schemaVersion }
}
