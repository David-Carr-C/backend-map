package main

import (
	"context"
	"dielmex-pmv-http/internal/database"
	"dielmex-pmv-http/internal/model"
	"dielmex-pmv-http/internal/server"
	"log"
	"net"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func gracefulShutdown(conn *net.UDPConn, done chan bool, db database.Service) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Println("shutting down gracefully, press Ctrl+C again to force")

	if err := db.Close(); err != nil {
		log.Printf("Database forced to shutdown with error: %v", err)
	}

	if err := conn.Close(); err != nil {
		log.Printf("Server forced to shutdown with error: %v", err)
	}

	log.Println("Server exiting")
	done <- true
}

func validarChecksum(data []byte) bool {
	messageSize := len(data)
	var xorSum byte

	for i := 0; i < messageSize-3; i++ {
		xorSum ^= data[i]
	}

	if xorSum%2 == 0 {
		xorSum++
	} else {
		xorSum--
	}

	if xorSum == 10 {
		xorSum++
	}

	return xorSum == data[messageSize-3]
}

func procesarMensaje(data []byte, addr *net.UDPAddr, db database.Service, con *net.UDPConn) {
	dbService := db.GetDB()
	message := string(data)

	ID := string(message[0:3])
	direccion := model.Direccion{
		Nombre:             ID,
		Direccion:          addr.IP.String(),
		Puerto:             strconv.Itoa(addr.Port),
		FechaActualizacion: time.Now().Format("Monday 02/Jan 15:04:05"),
	}

	result := dbService.Where("nombre = ?", ID).First(&model.Direccion{})
	if result.RowsAffected == 0 {
		dbService.Create(&direccion)
		log.Printf("New address added: %s", direccion)
	} else {
		dbService.Model(&model.Direccion{}).Where("nombre = ?", ID).Updates(direccion)
		log.Printf("Address updated: %s", direccion)
	}

}

func main() {
	// Starts DB
	db := server.NewDatabase()

	// Create a UDP connection
	address := 9095
	con, err := net.ListenUDP("udp", &net.UDPAddr{Port: address})
	if err != nil {
		log.Fatal(err)
	}

	defer con.Close()

	log.Printf("Listening on %s", con.LocalAddr().String())

	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)
	go gracefulShutdown(con, done, db)

	// Main loop
	for {
		// Check if the server has been asked to shutdown
		select {
		case <-done:
			return
		default:
			// Read from the connection
			buffer := make([]byte, 1024)
			n, addr, err := con.ReadFromUDP(buffer)
			if err != nil {
				log.Printf("Error reading UDP connection: %v", err)
				continue
			}

			message := buffer[:n]
			log.Printf("Received message from %s: %s", addr.String(), string(message))

			if validarChecksum(message) {
				procesarMensaje(message, addr, db, con)
			} else {
				log.Println("Corrupted message")
			}
		}

	}
}
