###############################################################################
# - SERVIDOR. 
# - 09/Agosto/2024
###############################################################################
import socket
from datetime import datetime
import numpy as np
import threading
import msvcrt
import time
import tkinter as tk
from tkinter import ttk

#==============================================================================
#================================= Funciones ==================================
#==============================================================================
# » Función para enviar caracteres a un cliente.
def SendUDP(ID_SEND,Comando):
    Payload = ID_SEND + ' ' + Comando + ' ' # Ejemplo: 'OAX C+DUMMY '
    
    # Se busca en 'Matriz_Reg' el Cliente en cuestión.
    ID_Flag = 0
    for ii in range(50):
        if (Matriz_Reg[ii][0] == ID_SEND):
            ID_Flag = 1
            break
    if ID_Flag == 0:
        print('Cliente no registrado.')
        return

    # Cliente registrado en 'Matriz_Reg'.
    print('<<< Se envia mensaje: \''+ Payload + 'CKCRLF\'')
                
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
    # ATENCIÓN: XOR_SUMA es un valor enteramente decimal. 
    # Por ejemplo, si 'XOR_SUMA = 86' => XOR_Suma.to_bytes(1,'big') = b'V', 
    # valor en byte.

    Carga = Payload_bytes + XOR_Suma.to_bytes(1,'big') + b'\r\n'
    
    s.sendto(Carga,(Matriz_Reg[ii][1],int(Matriz_Reg[ii][2])))
#==============================================================================
#=================================== Hilos ====================================
#==============================================================================
#=============================== Programa Principal ===========================
#==============================================================================

# HILOS
#------------------------------------------------------------------------------
# Limpia el archivo 'Registro'.
open('Registro.txt','w').close()

s = socket.socket(family=socket.AF_INET, type=socket.SOCK_DGRAM)
s.bind(('',9095))
print("Servidor UDP esta escuchando...")
#------------------------------------------------------------------------------
# CINCUENTA CLIENTES.
# Vector auxiliar para guardar los nombres UDP.
Vector_ID = np.zeros((50,1),dtype='<U3')    # Matriz con entradas STR de hasta 3 caracteres.
# Matriz para almacenar la información de entrada de 50 clientes, exeptuando la carga útil.
Matriz_Reg = np.zeros((50,4),dtype='<U30')  # Matriz con entradas STR de hasta 30 caracteres.
# Columna 0: Nombre UDP del Cliente, tres caracteres. Ejemplo: 'OAX'.
# Columna 1: IP del Cliente.
# Columna 2: Puerto del Cliente.
# Columna 3: Hora de último reporte del Cliente.
# CADA RENGLÓN DE ESTA MATRIZ ES LA INFORMACIÓN DE UN CLIENTE.
jj = 0
Conta_Pos = 0

#------------------------------------------------------------------------------
while(True):
    try:
        Dato = s.recvfrom(10024)             # Espera hasta recibir dato desde algún Cliente.
        message = Dato[0]                   # 'message' cadena conteniendo la entrada, en decimal.
        address = Dato[1]
        now = datetime.now()
        
        # Ejemplo:  message = b'OAX 00031 e\r\n' || address = ('200.68.170.69',21177).
        #......................................................................
        # Verificación del dato mediante el «Checksum»
        message_size = len(message)

        XOR_Suma = 0
        for ii in range(message_size-3):
            XOR_Suma = XOR_Suma ^ message[ii]
        if XOR_Suma % 2 == 0:
            XOR_Suma = XOR_Suma + 1
        else:
            XOR_Suma = XOR_Suma - 1
        
        # Si el CK resultara igual a \n, se suma uno, para evitar terminar la cadena prematuramente.
        if XOR_Suma == 10:
            XOR_Suma = XOR_Suma + 1
        #......................................................................
        # Compara el Checksum generado con el recibido.
        if message[message_size - 3] == XOR_Suma:
            # » El cliente envía su KEEP ALIVE con una frecuencia definida: ~1,~2,~3,...,~9 minutos.
            # » A la recepción y validación del KEEP ALIVE, este Servidor envía el dato 'NNN C+DUMMY CKCRLF'.
            # » Si la recepción del comando por el cliente es correcta, éste responde ACK y según el comando transferido.
            if (b'ACK' not in message):
                # Dato válido. No contiene ACK así que es un KEEP ALIVE. Se actualiza 'Matriz_Reg'.

                # Sobrescribe si es un cliente existente. Genera una nueva línea si es cliente nuevo.
                ID = message[0:3]                   # Los tres primeros char de la cadena, ejemplo: b'OAX'.
                ID = str(ID)                        # "b'OAX'"
                ID = ID[2:5]                        #  012345 
                ID_Flag = 0
                for ii in range(50):
                    if (Vector_ID[ii] == ID):       # ID ya registrada.
                        ID_Flag = 1                 # 'ii' almacena el número de renglón en cuestión.
                        break
                if ID_Flag == 0:                    # ID NO registrada.
                    Vector_ID[Conta_Pos] = ID       
                    ii = Conta_Pos                  # 'ii' almacena el número de renglón en cuestión.
                    Conta_Pos = Conta_Pos + 1       # Apunta al siguiente renglón vacío.
                    if Conta_Pos == 50:             # Vector lleno, sobrescribe en la primera posición.
                        Conta_Pos = 0

                Matriz_Reg[ii][0] = ID                                  # IP en STR.
                Matriz_Reg[ii][1] = address[0]                          # IP en STR del Cliente.
                Matriz_Reg[ii][2] = str(address[1])                     # Puerto en STR del cliente.
                Matriz_Reg[ii][3] = now.strftime("%A %d/%b %H:%M:%S")   # Hora en STR.
                np.savetxt('Registro.txt',Matriz_Reg,fmt='%s',delimiter=' ') 
                
                # Convierte el «array» en «list», segun la exigencia de Combobox.
                Lista_ID = Vector_ID.tolist() 
                for ii in range(50):
                    if Lista_ID[ii] == ['']:
                        break
                Lista_ID = Lista_ID[:ii]


                # Arriba KEEP ALIVE (conteo), ejemplo: '00T 00000 CKCRLF'.
                print('\n>>> KEEP ALIVE desde el Cliente: ' + str(message[:(len(message)-3)]) + 'CKCRLF')
                print(now.strftime("%A %H:%M:%S"))
                
                SendUDP(ID,'C+MEN?')
            else:
                # Dato válido. Sí contiene ACK así que es una respuesta.
                
                if b'MEN 11' in message:
                    print('>>> Respuesta FORMATEADA desde el Cliente: ')
                    ii = 12
                    for jj in range(9):
                        print(str(message[ii:ii+13]))
                        ii = ii + 14
                elif b'STAT' in message:
                    print('>>> Respuesta FORMATEADA desde el Cliente: ')
                    # b'OAX ACK STAT NAME:OAXACA,DNS:1,KEEP:1,TIME:1,SQ:20,TER:0,VOLcom:3.3,VOLali:12.3,VOLext:24.5 <CK><CR><LF>'
                    Stat = str(message)
                    Stat = Stat[15:-7]
                    Stat = ',' + Stat + ','
                    # ',NAME:OAXACA,DNS:1,KEEP:1,TIME:1,SQ:20,TER:0,VOLcom:3.3,VOLali:12.3,VOLext:24.5,'
                    Renglon = ''
                    for ii in range(len(Stat)):
                        if Stat[ii] == ',':
                            while(1):
                                ii = ii + 1
                                if ii == len(Stat):
                                    break
                                if Stat[ii] == ',':
                                    print(Renglon)
                                    Renglon = ''
                                    break
                                Renglon = Renglon + Stat[ii]
                else:
                    print('>>> Respuesta desde el Cliente: ' + str(message[:(len(message)-3)]) + 'CKCRLF')
                
                
                if b'NAME' in message and b'STAT' not in message:
                    # Respuesta generada por 'C+NAME' y no por 'C+STAT?'.
                    
                    # Si un cambio de nombre, sobrescribe en el renglón con el nombre pasado.
                    # Ejemplo de respuesta C+STAT?: '00T ACK STAT NAME:00T,DNS...<CK><CR><LF>'
                    # Ejemplo de respuesta C+NAME:  '00T ACK NAME:OAXACA...<CK><CR><LF>'
                    #                                01234567890123456
                    # Nombre pasado. En el ejemplo: 00T.
                    ID_PAS = message[0:3]
                    ID_PAS = str(ID_PAS)
                    ID_PAS = ID_PAS[2:5]
                    # Nombre actual. En el ejemplo: OAX
                    ID = message[13:16]
                    ID = str(ID)
                    ID = ID[2:5]
                    
                    for ii in range(50):
                        if (Vector_ID[ii] == ID_PAS):
                            break
                
                    Vector_ID[ii] = ID
                    Matriz_Reg[ii][0] = ID
                    np.savetxt('Registro.txt',Matriz_Reg,fmt='%s',delimiter=' ') 

        else:
            print('Dato entrante corrupto.' + '\n')

    except Exception as error:
        print('Una excepcion ha occurrido: ', error)
        continue