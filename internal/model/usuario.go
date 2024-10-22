package model

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Usuario struct {
	gorm.Model
	Nombre      string `gorm:"not null" json:"nombre"`
	Email       string `gorm:"not null" json:"email"`
	Contrasenia string `gorm:"not null" json:"contrasenia"`
	IdPerfil    uint   `gorm:"not null" json:"id_perfil"`
}

func (u *Usuario) EncriptarPassword() error {
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Contrasenia), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Contrasenia = string(hash)
	return nil
}
