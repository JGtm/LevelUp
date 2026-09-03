/**
 * replaySchemaLogic.ts — LA VERSION DE SCHÉMA que ce build du client attend.
 *
 * CE QU'IL RESTE DE LA GARDE `schemaVersion` (lot 2, audit AUDIT_AVAL_INVENTAIRE_2026-08-24,
 * point 1). Le lot 2 avait posé DEUX choses : la copie locale de la constante Go, et une NOTE à
 * l'écran quand l'artefact reçu n'était pas de la version attendue (« Données de rejeu d'une
 * version antérieure — certains éléments peuvent manquer »). La NOTE a été retirée le
 * 2026-08-25 sur demande utilisateur : elle s'affichait sur des rejeux parfaitement lisibles,
 * à chaque bump de schéma, jusqu'à ce que le backfill soit repassé — c'est-à-dire pendant des
 * jours, sur des matchs auxquels il ne manquait rien de visible. La fonction de comparaison qui
 * ne servait qu'à elle est partie avec (règle 0 code mort).
 *
 * CE QUI RESTE, ET POURQUOI. La copie locale de `replay.SchemaVersion` et son garde-rail de
 * PARITÉ (`replaySchemaLogic.guard.test.ts`) : le contrat généré (`lib/api/generated.ts`) type
 * `schemaVersion` en `number` sans valeur littérale — cette valeur varie d'un artefact à
 * l'autre, c'est le point. La source de vérité reste donc la constante Go
 * (`internal/analysis/replay/document.go`), et le garde-rail lit ce fichier pour interdire à la
 * copie de dériver. Il documente, dans le front, la version que le back sert aujourd'hui.
 *
 * ATTENTION SI L'ON REVIENT ICI : depuis le retrait de la note, la constante n'a plus de
 * lecteur À L'EXÉCUTION — son seul consommateur est le garde-rail de parité. La rebrancher
 * (télémétrie, garde de lecture par version) est le geste qui lui rendrait un rôle ; la
 * supprimer ferait perdre l'ancrage vérifiable entre les deux côtés.
 */

/**
 * EXPECTED_REPLAY_SCHEMA_VERSION — copie locale de `replay.SchemaVersion`
 * (apps/go-api/internal/analysis/replay/document.go). Le garde-rail de parité
 * (replaySchemaLogic.guard.test.ts) fait échouer la CI si cette valeur diverge de la
 * constante Go — la mettre à jour ici NE SUFFIT PAS à faire mentir le garde-rail.
 */
export const EXPECTED_REPLAY_SCHEMA_VERSION = 37
