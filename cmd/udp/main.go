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

func enviarDummy(ID string, db database.Service) {
	dbService := db.GetDB()

	// Comando dummy
	comando := model.CatComando{}
	result := dbService.Where("id = ?", 5).First(&comando)
	if result.RowsAffected == 0 {
		log.Printf("Command not found: %d", 3)
		return
	}

	// Direccion a la que se envia el dummy
	direccion := model.Direccion{}
	ID = "OAX"
	result = dbService.Where("nombre = ?", ID).First(&direccion)
	if result.RowsAffected == 0 {
		log.Printf("Address not found: %s", ID)
		return
	}

	// replace {{}} -> {{ID_UDP}}_C+DUMMY_{{CK}}\r\n
	// "{{ID_UDP}}_C+MEN?_{{CK}}"
	// comandoString := strings.Replace(comando.Comando, "{{ID_UDP}}", direccion.Nombre, -1)
	// comandoString = strings.Replace(comandoString, "_", " ", -1)
	// comandoString = strings.Replace(comandoString, "{{CK}}", "", -1)

	// Test command and hardcoded
	// {{ID_UDP}}_C+NAME{{NOMBRE_NUEVO}}_{{CK}}\r\n
	comandoString := "OAX C+NAMEOAK "
	// comandoString := "OAX C+DUMMY "

	// from python:
	/*
			 # Adecua el dato de salida agregando a Payload la cadena CK+LF+CR.
		    Payload_size = len(Payload)
		    Payload_bytes = bytes(Payload,'utf-8')
		    # Payload.encode() = bytes(Payload,'utf-8') dan los mismos resultados.

		    XOR_Suma = 0
		    for nn in range(Payload_size):
		        XOR_Suma = XOR_Suma ^ Payload_bytes[nn]
		    if XOR_Suma % 2 == 0:
		        XOR_Suma = XOR_Suma + 1
		    else:
		        XOR_Suma = XOR_Suma - 1
		    # Si el CK resultara igual a \n, se suma uno. Evita terminar la cadena prematuramente.
		    if XOR_Suma == 10:
		        XOR_Suma = XOR_Suma + 1
	*/

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

	payload := append(payloadBytes, xorSum, 13, 10) // 13 = CR, 10 = LF

	// Crear conexión udp
	ip := direccion.Direccion
	port, _ := strconv.Atoi(direccion.Puerto)
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		log.Printf("Error resolving address: %v", err)
		return
	}

	log.Printf("Sending command to %s: %s", addr.String(), comandoString)

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Printf("Error creating connection: %v", err)
		return
	}

	defer conn.Close()

	// Enviar comando
	_, err = conn.Write(payload)
	if err != nil {
		log.Printf("Error sending command: %v", err)
		return
	}

	log.Printf("Last final")
	log.Printf("Command sent: %s", comandoString)
}

func procesarMensaje(data []byte, addr *net.UDPAddr, db database.Service, con *net.UDPConn) {
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
		log.Printf("Sending ACK to %s", addr.String())
		enviarDummy(ID, db)
	} else {
		log.Printf(">>> RESPONSE from client: %s", message)
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
				log.Println("Checksum OK")
				procesarMensaje(message, addr, db, con)
			} else {
				log.Println("Corrupted message")
			}
		}

	}
}
