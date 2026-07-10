// Package handlers — handlers HTTP chi + Huma de l'API LevelUp.
//
// Traduisent les requêtes en appels de services et projettent les résultats en
// DTO ; aucune logique métier ici (extraite en service/ ou analysis/). Couvrent
// career, citations, media, admin, auth Xbox, sync, achievements, etc. L'auth et
// l'ownership sont posés au MONTAGE (server.go), pas dans le handler ; la
// traduction d'une capability manquante en 503 passe par MapCapabilityError.
package handlers
