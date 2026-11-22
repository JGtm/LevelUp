/**
 * lib/replay/relief — LA BIBLIOTHEQUE DU FOND « RELIEF ».
 *
 * Fonctions pures et un worker, sans branchement à l'interface. Le
 * contrat de format et les seuils sont ceux de la brique `himap` (`DonneesRelief`, `CalculeRelief`).
 *
 * Parcours d'une carte :
 *   1. `parseReliefData(json)` et `readReliefImage(blob, calibration, height)` -> `ReliefData`, `ReliefHeights` ;
 *      toute `ReliefError` = repli sur le Plan ;
 *   2. `createReliefComputer()` puis `load({ image, data })` ;
 *   3. `render({ turns, cut, base, lines, masks })` -> `ReliefFrame` (image teintée, masques, traits) ;
 *   4. le suivi du joueur ouvert (`stepFollow`) et les rangs d'étage (`floorRanks`, `snapCut`) tournent
 *      sur le fil principal, sans worker.
 */
export * from './constants'
export * from './errors'
export type * from './types'
export { parseReliefData, hasNoMesh, RELIEF_SCHEMA_VERSION, decodePolylines } from './decodeData'
export { decodeReliefPixels, readReliefImage, domLoaders } from './decodeImage'
export type { RawPixels, PixelSource, ReliefImageLoaders } from './decodeImage'
export { pixelToWorld, worldToPixel, traceLineSet, emptyLineSet, lineCount } from './lineSet'
export type { PathTarget } from './lineSet'
export { bearingFloor, floorHeightAt } from './floorMesh'
export { buildFloorStack, stackHeight, stackTop } from './floorStack'
export type { FloorStack } from './floorStack'
export { lightForTurns, shadeRelief, heightGrey } from './shading'
export { buildTintTable, reliefRamp, rgbToCss, parseHexColor, tintRelief } from './tint'
export type { ReliefBaseColor, ReliefRamp, Rgb, TintedImage } from './tint'
export { computeSlice, cutWeight } from './slice'
export type { ReliefSlice, SliceInput, SliceOptions } from './slice'
export { filterLinesByKeep } from './lineFilter'
export { floorRanks, meshShareOf, nearestRank, quantizeCut, rankHeight, snapCut } from './floors'
export type { FloorRank, MeshShare } from './floors'
export { FOLLOW_INITIAL, stepFollow } from './follow'
export type { FollowContext, FollowSample, FollowState } from './follow'
export { ReliefEngine } from './engine'
export type { ReliefFrame, ReliefFrameLines, ReliefInfo, ReliefLoadInput, ReliefRenderRequest } from './engine'
export { ReliefComputer, createReliefComputer, defaultWorkerFactory } from './client'
export type { ReliefComputerOptions, ReliefWorkerLike } from './client'
