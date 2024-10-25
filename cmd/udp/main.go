package main

import (
	"context"
	"dielmex-pmv-http/internal/database"
	"dielmex-pmv-http/internal/model"
	"dielmex-pmv-http/internal/server"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
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

func calcularChecksum(data []byte) byte {
	var xorSum byte
	for _, b := range data {
		xorSum ^= b
	}

	if xorSum%2 == 0 {
		xorSum++
	} else {
		xorSum--
	}

	// Si el CK resultara igual a '\n'
	if xorSum == 10 {
		xorSum++
	}

	return xorSum
}

func validarChecksum(data []byte) bool {
	message_size := len(data)

	var xorSum byte
	for i := 0; i < message_size-3; i++ {
		xorSum ^= data[i]
	}

	if xorSum%2 == 0 {
		xorSum++
	} else {
		xorSum--
	}

	// Si el CK resultara igual a '\n'
	if xorSum == 10 {
		xorSum++
	}

	return xorSum == data[message_size-3]
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
	payloadBytes := []byte(comandoString)
	xorSum := calcularChecksum(payloadBytes)

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
	conn.SetReadDeadline(time.Now().Add(1024 * time.Second)) // todo: remove later
}

func enviarComandoCarga(idDevice string, db database.Service, idCommand int, conn *net.UDPConn, carga1 string, cargaNum int) {
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

	// Limpiar carga, cada linea debe tener 10 caracteres, si no lo logra rellena con ....
	// Primero divide/split por renglones
	cargaList1 := strings.Split(carga1, "\n")

	// Contar cuantos caracteres tiene cada renglon
	for i, renglon := range cargaList1 {
		if len(renglon) > 10 {
			cargaList1[i] = renglon[:10]
		} else {
			cargaList1[i] = renglon + strings.Repeat(".", 10-len(renglon))
		}
	}

	carga1 = strings.Join(cargaList1, ",")

	// Replace {{}} -> {{ID_UDP}}_C+DUMMY_{{CK}}\r\n
	// "{{ID_UDP}}_C+MEN?_{{CK}}\r\n"
	cargaNumero := strconv.Itoa(cargaNum)
	messageVal := cargaNumero + "," + carga1
	comandoString := strings.ReplaceAll(comando.Comando, "{{ID_UDP}}", direccion.Nombre)
	comandoString = strings.ReplaceAll(comandoString, "_", " ")
	comandoString = strings.ReplaceAll(comandoString, "{{MENSAJE}}", messageVal)
	comandoString = strings.ReplaceAll(comandoString, "{{CK}}", "")
	comandoString = strings.ReplaceAll(comandoString, "\r\n", "")

	log.Printf("Command builded: %s", comandoString)

	// Calculate checksum
	payloadBytes := []byte(comandoString)
	xorSum := calcularChecksum(payloadBytes)

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
	conn.SetReadDeadline(time.Now().Add(1024 * time.Second)) // todo: remove later
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
		enviarComando(ID, db, 3, conn)
	} else {
		log.Printf(">>> RESPONSE from client: %s", message)
	}

}

var globalConn *net.UDPConn
var globalDB database.Service

func handleSendCommandRequest(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	commandID, err := strconv.Atoi(r.URL.Query().Get("command_id"))
	if err != nil {
		http.Error(w, "Invalid command_id", http.StatusBadRequest)
		return
	}

	enviarComando(deviceID, globalDB, commandID, globalConn)
	w.WriteHeader(http.StatusOK)
}

func handleCreateLocationRequest(w http.ResponseWriter, r *http.Request) {
	currentName := r.FormValue("nombre")
	latitud := r.FormValue("latitud")
	longitud := r.FormValue("longitud")
	currentMessage1 := r.FormValue("currentMessage1")
	currentMessage2 := r.FormValue("currentMessage2")
	currentMessage3 := r.FormValue("currentMessage3")

	log.Printf("Entrando a handleCreateLocationRequest")
	log.Printf("Name: %s, Latitud: %s, Longitud: %s, Message1: %s, Message2: %s, Message3: %s", currentName, latitud, longitud, currentMessage1, currentMessage2, currentMessage3)

	dbService := globalDB.GetDB()

	// Actualizar si es el mismo ID
	currentNameSearch := currentName[:3]
	var dispositivo model.Dispositivo
	result := dbService.Where("nombre = ?", currentNameSearch).First(&dispositivo)
	if result.RowsAffected > 0 {
		dispositivo.Latitud = latitud
		dispositivo.Longitud = longitud
		dispositivo.Mensaje1 = currentMessage1
		dispositivo.Mensaje2 = currentMessage2
		dispositivo.Mensaje3 = currentMessage3

		result = dbService.Model(&model.Dispositivo{}).Where("nombre = ?", currentNameSearch).Updates(dispositivo)
		if result.Error != nil {
			http.Error(w, result.Error.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		enviarComandoCarga(currentNameSearch, globalDB, 2, globalConn, currentMessage1, 1)
		enviarComandoCarga(currentNameSearch, globalDB, 2, globalConn, currentMessage2, 2)
		enviarComandoCarga(currentNameSearch, globalDB, 2, globalConn, currentMessage3, 3)
		return
	}

	// Dar de alta en model.Dispositivo
	dispositivo = model.Dispositivo{
		Nombre:   currentName,
		Latitud:  latitud,
		Longitud: longitud,
		Mensaje1: currentMessage1,
		Mensaje2: currentMessage2,
		Mensaje3: currentMessage3,
	}

	result = dbService.Create(&dispositivo)
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	enviarComandoCarga(currentNameSearch, globalDB, 2, globalConn, currentMessage1, 1)
	enviarComandoCarga(currentNameSearch, globalDB, 2, globalConn, currentMessage2, 2)
	enviarComandoCarga(currentNameSearch, globalDB, 2, globalConn, currentMessage3, 3)

	w.WriteHeader(http.StatusOK)
}

func handleGetLocationsRequest(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*") // Permitir todas las conexiones
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	dbService := globalDB.GetDB()

	var dispositivos []model.Dispositivo
	result := dbService.Find(&dispositivos)
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Dispositivos: %v", dispositivos)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dispositivos)
}

func handleGetLocationRequest(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	dbService := globalDB.GetDB()

	var dispositivo model.Dispositivo
	result := dbService.Where("id = ?", id).First(&dispositivo)
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dispositivo)
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

	globalConn = conn
	globalDB = db
	http.HandleFunc("/send-command", handleSendCommandRequest)
	http.HandleFunc("/create-location", handleCreateLocationRequest)
	http.HandleFunc("/get-locations", handleGetLocationsRequest)
	http.HandleFunc("/get-location", handleGetLocationRequest)
	go func() {
		http.Handle("/", http.FileServer(http.Dir("./static")))
		if err := http.ListenAndServe(":9095", nil); err != nil {
			log.Fatalf("Error starting HTTP server: %v", err)
		} else {
			log.Println("HTTP server started on port 9095")
		}
	}()

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
