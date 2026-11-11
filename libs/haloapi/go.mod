// Deprecated: ce module est publié dans son propre dépôt, github.com/JGtm/halo-api ; utiliser ce chemin de module.
module github.com/JGtm/LevelUp/libs/haloapi

go 1.26.11

require (
	golang.org/x/sync v0.24.0
	golang.org/x/time v0.16.0
)

retract v0.1.0 // la capture de classement de testdata contenait des identifiants de tiers ; remplacée dans github.com/JGtm/halo-api v1.0.1
