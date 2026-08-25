/**
 * replaySchemaLogic.ts — LA GARDE `schemaVersion`, absente jusqu'à ce lot (audit
 * AUDIT_AVAL_INVENTAIRE_2026-08-24.md, point 1, le plus grave).
 *
 * CE QUE L'AUDIT CONSTATAIT : le client ne lisait JAMAIS `doc.schemaVersion`. Un artefact
 * construit avant un bump de schéma sert une fiche AMPUTÉE — un champ ajouté par la montée
 * n'existe simplement pas sur le vieil artefact — et rien à l'écran ne le distingue d'un
 * « rien à afficher ». La reprise du backfill se fait par SchemaVersion (opérateur, commande
 * manuelle, cf. `cmd_backfill_replay.go`) : tant qu'elle n'a pas tourné pour un match donné,
 * son artefact reste dégradé SANS SIGNAL.
 *
 * CE QUE CE FICHIER AJOUTE, ET CE QU'IL N'AJOUTE PAS. Une note DISCRÈTE, jamais un blocage —
 * la fiche continue d'afficher tout ce que l'artefact porte (décision produit du lot, cf.
 * LOT2_TELEMETRIE_GARDE_SCHEMA_2026-08-25.md). Ni le REJET des lectures antérieures à l'origine
 * (point 2 de l'audit) ni le fixture `SelectedGrenadeRank` (découverte du lot 1) n'entrent
 * dans ce périmètre.
 *
 * D'OÙ VIENT `EXPECTED_REPLAY_SCHEMA_VERSION`. Le contrat généré (`lib/api/generated.ts`) type
 * `schemaVersion` en `number` — il ne PORTE aucune valeur littérale, puisque cette valeur
 * varie précisément d'un artefact à l'autre (c'est le point). La source de vérité reste donc
 * la constante Go `replay.SchemaVersion` (document.go) ; ce fichier en garde une COPIE locale,
 * documentée, avec un garde-rail de PARITÉ (`replaySchemaLogic.guard.test.ts`, même patron que
 * `placementFamily.guard.test.ts`) qui lit le fichier Go et fait échouer le test si les deux
 * divergent — une génération dédiée pour un seul entier serait disproportionnée.
 */

/**
 * EXPECTED_REPLAY_SCHEMA_VERSION — copie locale de `replay.SchemaVersion`
 * (apps/go-api/internal/analysis/replay/document.go). Le garde-rail de parité
 * (replaySchemaLogic.guard.test.ts) fait échouer la CI si cette valeur diverge de la
 * constante Go — la mettre à jour ici NE SUFFIT PAS à faire mentir le garde-rail.
 */
export const EXPECTED_REPLAY_SCHEMA_VERSION = 19

/**
 * ReplaySchemaState — les trois lectures possibles d'un `schemaVersion` reçu face à ce que ce
 * build du client sait exploiter :
 * - `current` : les deux versions concordent, rien à dire ;
 * - `stale`   : l'artefact est ANTÉRIEUR — construit avant un bump que ce client attend. C'est
 *   le cas de l'audit : un backfill n'a pas encore tourné sur ce match ;
 * - `ahead`   : l'artefact est POSTÉRIEUR — ce déploiement du client est en retard sur le
 *   format que le serveur sert désormais. Rare (suppose un serveur redéployé avant le client),
 *   mais symétrique : un client qui ignorerait silencieusement des champs qu'il ne connaît pas
 *   encore mérite la même note que l'inverse.
 */
export type ReplaySchemaState = 'current' | 'stale' | 'ahead'

/**
 * replaySchemaState compare le `schemaVersion` d'un artefact reçu à la version que CE build du
 * client sait exploiter. PURE — aucune I/O, aucun accès document : c'est ce qui la rend
 * testable sans fabriquer un ReplayDocument complet.
 */
export function replaySchemaState(schemaVersion: number): ReplaySchemaState {
  if (schemaVersion < EXPECTED_REPLAY_SCHEMA_VERSION) return 'stale'
  if (schemaVersion > EXPECTED_REPLAY_SCHEMA_VERSION) return 'ahead'
  return 'current'
}
