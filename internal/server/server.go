package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"dielmex-pmv-http/internal/database"
	"dielmex-pmv-http/internal/model"
)

type Server struct {
	port int

	db database.Service
}

func migrations(db database.Service) {
	dbService := db.GetDB()
	dbService.AutoMigrate(&model.Usuario{})
	dbService.AutoMigrate(&model.CatComando{})
	dbService.AutoMigrate(&model.Direccion{})
	dbService.AutoMigrate(&model.Dispositivo{})

	// Insert default commands
	model.SeedCatComando(dbService)

	// Insert default users
	root := model.Usuario{
		Nombre:      "root",
		Email:       "soporte@mail.com",
		Contrasenia: "tacos_de_asada",
		IdPerfil:    1,
	}

	root.EncriptarPassword()

	usuario := model.Usuario{
		Nombre:      "usuario",
		Email:       "usuario@mail.com",
		Contrasenia: "tacos_de_asada",
		IdPerfil:    2,
	}

	usuario.EncriptarPassword()

	dbService.Create(&root)
	dbService.Create(&usuario)
}

func NewDatabase() database.Service {
	database := database.New()
	migrations(database)
	return database
}

func NewServer() *http.Server {
	database := NewDatabase()

	port, _ := strconv.Atoi(os.Getenv("PORT"))
	NewServer := &Server{
		port: port,
		db:   database,
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
