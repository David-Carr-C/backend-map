package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

// Constantes globales
const (
	MaxClients = 50
	Port       = ":9095"
	RecordFile = "Registro.txt"
)

// Estructuras de datos
type ClientInfo struct {
	ID       string
	IP       string
	Port     string
	LastSeen string
}

var (
	VectorID  [MaxClients]string
	MatrizReg [MaxClients]ClientInfo
	ContaPos  int
	mu        sync.Mutex
)

// SendUDP envía un mensaje a un cliente específico.
func SendUDP(ID_SEND, Comando string) {
	Payload := fmt.Sprintf("%s %s ", ID_SEND, Comando)

	// Verificar si el cliente está registrado
	clientIdx := -1
	for i, client := range MatrizReg {
		if client.ID == ID_SEND {
			clientIdx = i
			break
		}
	}
	if clientIdx == -1 {
		fmt.Println("Cliente no registrado.")
		return
	}

	// Preparar mensaje con checksum y enviar
	fmt.Printf("<<< Se envía mensaje: '%sCKCRLF'\n", Payload)

	XOR_Suma := calcularChecksum([]byte(Payload))
	Carga := append([]byte(Payload), XOR_Suma, '\r', '\n')

	// Enviar mensaje UDP
	addr := fmt.Sprintf("%s:%s", MatrizReg[clientIdx].IP, MatrizReg[clientIdx].Port)
	conn, err := net.Dial("udp", addr)
	if err != nil {
		fmt.Println("Error al conectar:", err)
		return
	}
	defer conn.Close()

	_, err = conn.Write(Carga)
	if err != nil {
		fmt.Println("Error al enviar mensaje:", err)
	}
}

// calcularChecksum calcula el checksum XOR para el payload.
func calcularChecksum(data []byte) byte {
	var xor byte
	for _, b := range data {
		xor ^= b
	}
	if xor%2 == 0 {
		xor++
	} else {
		xor--
	}
	if xor == '\n' {
		xor++
	}
	return xor
}

// actualizarRegistro guarda la información del cliente en MatrizReg.
func actualizarRegistro(id, ip, port string) {
	mu.Lock()
	defer mu.Unlock()

	for i, client := range MatrizReg {
		if client.ID == id {
			MatrizReg[i].LastSeen = time.Now().Format("Monday 02/Jan 15:04:05")
			return
		}
	}
	if ContaPos >= MaxClients {
		ContaPos = 0
	}
	MatrizReg[ContaPos] = ClientInfo{
		ID:       id,
		IP:       ip,
		Port:     port,
		LastSeen: time.Now().Format("Monday 02/Jan 15:04:05"),
	}
	ContaPos++
	guardarRegistro()
}

// guardarRegistro guarda la matriz de registros en un archivo.
func guardarRegistro() {
	file, err := os.Create(RecordFile)
	if err != nil {
		log.Fatal("Error al crear archivo:", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, client := range MatrizReg {
		if client.ID == "" {
			continue
		}
		err := writer.Write([]string{client.ID, client.IP, client.Port, client.LastSeen})
		if err != nil {
			log.Fatal("Error al escribir en archivo:", err)
		}
	}
}

// manejarConexion maneja la conexión UDP entrante.
func manejarConexion(conn *net.UDPConn) {
	buf := make([]byte, 1024)

	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error al leer desde UDP:", err)
			continue
		}

		message := buf[:n]
		fmt.Printf("\n>>> Mensaje recibido: %s\n", string(message))

		if validarChecksum(message) {
			id := string(message[:3])
			actualizarRegistro(id, addr.IP.String(), strconv.Itoa(addr.Port))

			if !bytes.Contains(message, []byte("ACK")) {
				fmt.Println(">>> KEEP ALIVE recibido.")
				SendUDP(id, "C+DUMMY")
			} else {
				fmt.Printf(">>> Respuesta desde cliente: %s\n", string(message))
			}
		} else {
			fmt.Println("Dato entrante corrupto.")
		}
	}
}

// validarChecksum valida el checksum del mensaje recibido.
func validarChecksum(data []byte) bool {
	size := len(data)
	xor := calcularChecksum(data[:size-3])
	return xor == data[size-3]
}

// iniciarServidor inicia el servidor UDP.
func iniciarServidor() {
	addr, err := net.ResolveUDPAddr("udp", Port)
	if err != nil {
		log.Fatal("Error al resolver dirección:", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal("Error al iniciar servidor UDP:", err)
	}
	defer conn.Close()

	fmt.Println("Servidor UDP escuchando en", Port)
	manejarConexion(conn)
}

// Programa principal
func main() {
	// Limpiar el archivo de registro
	os.Remove(RecordFile)

	// Iniciar servidor UDP en una goroutine
	go iniciarServidor()

	// Esperar indefinidamente
	select {}
}
