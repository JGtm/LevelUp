// Package games — frontière title-agnostic (ADR 0011/0012/0025).
//
// Définit les interfaces d'adaptation par titre (TitleDataAdapter,
// TitleSemanticAdapter, TitleAssetURLAdapter), le Resolver registry-driven, les
// CapabilityMap (branchement sur capabilities, jamais sur `slug == "..."`) et le
// damage model par titre. Aucune logique slug-littérale : les implémentations
// concrètes vivent dans games/halo_infinite/ et games/halo_5/.
package games
