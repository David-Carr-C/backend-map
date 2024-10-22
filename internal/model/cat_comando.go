package model

import (
	"log"

	"gorm.io/gorm"
)

type CatComando struct {
	gorm.Model
	Comando     string `gorm:"not null" json:"comando"`
	Descripcion string `gorm:"not null" json:"descripcion"`
}

//dielmexpmv.dyndns.org
//dielmexgps.dyndns.org

var comandos = []CatComando{
	{Comando: "{{ID_UDP}}_C+NAME{{NOMBRE_NUEVO}}_{{CK}}\r\n", Descripcion: "Cambia el nombre del dispositivo"},
	{Comando: "{{ID_UDP}}_C+MEN{{MENSAJE}}_{{CK}}\r\n", Descripcion: "Mensajes a desplegar"},
	{Comando: "{{ID_UDP}}_C+DUMMY_{{CK}}\r\n", Descripcion: "Comando dummy para comprobar conexión"},
	{Comando: "{{ID_UDP}}_C+RST_{{CK}}\r\n", Descripcion: "Reinicia el dispositivo"},
	{Comando: "{{ID_UDP}}_C+MEN?_{{CK}}\r\n", Descripcion: "Consulta los mensajes a desplegar"},
	{Comando: "{{ID_UDP}}_C+KEEP_{{FRECUENCIA}}{{CK}}\r\n", Descripcion: "Modifica la frecuencia de envío de keep alive"},
	{Comando: "{{ID_UDP}}_C+STAT?_{{CK}}\r\n", Descripcion: "Consulta el estado del dispositivo"},
	{Comando: "{{ID_UDP}}_C+DNS{{DOMINIO}}_{{CK}}\r\n", Descripcion: "Configura el dominio donde se reportarán los dispositivos"},
	{Comando: "{{ID_UDP}}_C+TEST_{{CK}}\r\n", Descripcion: "Enciende todos los leds del dispositivo"},
	{Comando: "{{ID_UDP}}_C+TIME{{TIEMPO}}_{{CK}}\r\n", Descripcion: "Configura el tiempo que se muestra cada mensaje"},
	{Comando: "{{ID_UDP}}_C+TER{{ACTIVO}}_{{CK}}\r\n", Descripcion: "Activa o desactiva la terminal por BLE"},
}

func SeedCatComando(db *gorm.DB) {
	var count int64

	if err := db.Model(&CatComando{}).Count(&count).Error; err != nil {
		log.Fatalf("Error al contar registros: %v", err)
	}

	if count == 0 {
		// Poblar la tabla con los comandos iniciales
		if err := db.Create(&comandos).Error; err != nil {
			log.Fatalf("Error al poblar CatComando: %v", err)
		}
		log.Println("Catálogo 'CatComando' inicializado correctamente.")
	} else {
		log.Println("Catálogo 'CatComando' ya contiene datos.")
	}
}
