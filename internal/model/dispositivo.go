package model

import (
	"gorm.io/gorm"
)

type Dispositivo struct {
	gorm.Model
	Nombre   string `json:"nombre"`
	Latitud  string `json:"latitud"`
	Longitud string `json:"longitud"`
	Mensaje1 string `json:"currentMessage1"`
	Mensaje2 string `json:"currentMessage2"`
	Mensaje3 string `json:"currentMessage3"`
}
