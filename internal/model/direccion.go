package model

import (
	"gorm.io/gorm"
)

type Direccion struct {
	gorm.Model
	Nombre             string `gorm:"not null" json:"nombre"`
	Direccion          string `gorm:"not null" json:"direccion"`
	Puerto             string `gorm:"not null" json:"puerto"`
	FechaActualizacion string `gorm:"not null" json:"fecha_actualizacion"`
}
