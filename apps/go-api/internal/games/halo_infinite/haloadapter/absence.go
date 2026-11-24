package haloadapter

import "github.com/JGtm/halo-api/infinite"

// IsNotFound dit si `err` signale une ressource ABSENTE côté Halo (HTTP 404 ou 410), verdict TYPÉ
// de la brique : match disparu de l'API, skill ou film perdu. Une autre erreur (réseau, 5xx,
// 401/403, identifiant mal formé) n'en est pas une : la ressource peut revenir ou n'a jamais été
// demandée correctement.
func IsNotFound(err error) bool { return infinite.IsNotFound(err) }
