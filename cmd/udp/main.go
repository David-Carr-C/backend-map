package main

import (
	"context"
	"dielmex-pmv-http/internal/database"
	"dielmex-pmv-http/internal/model"
	"dielmex-pmv-http/internal/server"
	"fmt"
	"log"
	"net"
	"os/signal"
	"strconv"
	"strings"
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

func enviarComando(idDevice string, db database.Service, idCommand int, conn *net.UDPConn) {
	dbService := db.GetDB()

	// Comando dummy
	comando := model.CatComando{}
	result := dbService.Where("id = ?", idCommand).First(&comando)
	if result.RowsAffected == 0 {
		log.Printf("Command not found: %d", idCommand)
		return
	}

	// Direccion a la que se envia el dummy
	direccion := model.Direccion{}
	result = dbService.Where("nombre = ?", idDevice).First(&direccion)
	if result.RowsAffected == 0 {
		log.Printf("Address not found: %s", idDevice)
		return
	}

	// Replace {{}} -> {{ID_UDP}}_C+DUMMY_{{CK}}\r\n
	// "{{ID_UDP}}_C+MEN?_{{CK}}\r\n"
	comandoString := strings.ReplaceAll(comando.Comando, "{{ID_UDP}}", direccion.Nombre)
	comandoString = strings.ReplaceAll(comandoString, "_", " ")
	comandoString = strings.ReplaceAll(comandoString, "{{CK}}", "")
	comandoString = strings.ReplaceAll(comandoString, "\r\n", "")

	log.Printf("Command builded: %s", comandoString)

	// Calculate checksum
	payloadSize := len(comandoString)
	payloadBytes := []byte(comandoString)

	var xorSum byte
	for i := 0; i < payloadSize; i++ {
		xorSum ^= payloadBytes[i]
	}

	if xorSum%2 == 0 {
		xorSum++
	} else {
		xorSum--
	}

	if xorSum == 10 {
		xorSum++
	}

	if xorSum == 13 {
		xorSum++
	}

	payload := append(payloadBytes, xorSum, 13, 10) // 13 = CR, 10 = LF

	log.Printf("Payload: %s", payload)
	log.Printf("Payload bin (hex): %x", payload)

	// Crear conexión udp
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%s", direccion.Direccion, direccion.Puerto))
	if err != nil {
		log.Printf("Error creating UDP address: %v", err)
		return
	}

	_, err = conn.WriteToUDP(payload, udpAddr)
	if err != nil {
		log.Printf("Error sending UDP message: %v", err)
		return
	}

	log.Printf("Command sent to %s: %s", direccion.Nombre, comandoString)

	// Wait for response
	conn.SetReadDeadline(time.Now().Add(1024 * time.Second))

}

func procesarMensaje(data []byte, addr *net.UDPAddr, db database.Service, conn *net.UDPConn) {
	dbService := db.GetDB()
	message := string(data)

	ID := string(message[0:3])
	direccion := model.Direccion{
		Nombre:             ID,
		Direccion:          addr.IP.String(),
		Puerto:             strconv.Itoa(addr.Port),
		FechaActualizacion: time.Now().In(time.Local).Format("Monday 02/Jan 15:04:05"),
	}

	result := dbService.Where("nombre = ?", ID).First(&model.Direccion{})
	if result.RowsAffected == 0 {
		dbService.Create(&direccion)
		log.Printf("New address added: %s", direccion)
	} else {
		dbService.Model(&model.Direccion{}).Where("nombre = ?", ID).Updates(direccion)
		log.Printf("Address updated: %s", direccion)
	}

	if !strings.Contains(message, "ACK") {
		enviarComando(ID, db, 7, conn)
	} else {
		log.Printf(">>> RESPONSE from client: %s", message)
	}

}

func main() {
	// Starts DB
	db := server.NewDatabase()

	// Create a UDP connection
	address := 9095
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: address})
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	log.Printf("Listening on %s", conn.LocalAddr().String())

	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)
	go gracefulShutdown(conn, done, db)

	// Main loop
	for {
		// Check if the server has been asked to shutdown
		select {
		case <-done:
			return
		default:
			// Read from the connection
			buffer := make([]byte, 1024)
			n, addr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				log.Printf("Error reading UDP connection: %v", err)
				continue
			}

			message := buffer[:n]
			// log.Printf("Received message from %s: %s", addr.String(), string(message))

			if validarChecksum(message) {
				log.Println("Checksum OK")
				procesarMensaje(message, addr, db, conn)
			} else {
				log.Println("Corrupted message")
			}
		}

	}
}
