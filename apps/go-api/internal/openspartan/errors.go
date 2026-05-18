package openspartan

import "errors"

var (
	ErrNotOpenSpartanDB = errors.New("openspartan: file is not a recognizable OpenSpartan database")
	ErrEmptyDB          = errors.New("openspartan: database contains no matches")
	ErrReaderClosed     = errors.New("openspartan: reader is closed")
	ErrOwnerUndetected  = errors.New("openspartan: could not detect owner XUID with any heuristic")
	ErrMatchNotFound    = errors.New("openspartan: match not found")
)
