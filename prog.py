###############################################################################
# - SERVIDOR. 
# - 09/Agosto/2024
###############################################################################
import socket
from datetime import datetime
import numpy as np
import threading
import sys  # Para detectar plataforma
import time
import tkinter as tk
from tkinter import ttk

# Importación condicional según el sistema operativo
if sys.platform == "win32":
    import msvcrt
else:
    import select  # Alternativa para Linux/Unix

#==============================================================================
#================================= Funciones ==================================
#==============================================================================
def SendUDP(ID_SEND, Comando):
    Payload = ID_SEND + ' ' + Comando + ' '  # Ejemplo: 'OAX C+DUMMY '
    
    ID_Flag = 0
    for ii in range(50):
        if (Matriz_Reg[ii][0] == ID_SEND):
            ID_Flag = 1
            break
    if ID_Flag == 0:
        print('Cliente no registrado.')
        return

    print('<<< Se envia mensaje: \'' + Payload + 'CKCRLF\'')
                
    Payload_bytes = bytes(Payload, 'utf-8')
    XOR_Suma = 0
    for nn in range(len(Payload_bytes)):
        XOR_Suma ^= Payload_bytes[nn]
    XOR_Suma = XOR_Suma + 1 if XOR_Suma % 2 == 0 else XOR_Suma - 1
    if XOR_Suma == 10:
        XOR_Suma += 1

    Carga = Payload_bytes + XOR_Suma.to_bytes(1, 'big') + b'\r\n'
    s.sendto(Carga, (Matriz_Reg[ii][1], int(Matriz_Reg[ii][2])))

#==============================================================================
#=================================== Hilos ====================================
#==============================================================================
def thread2():   
    def captura():
        entrada = ID_entrada.get()
        entrada2 = CMD_entrada.get()
        print('\n')
        SendUDP(entrada, entrada2)

    def esperar_tecla():
        """Función que detecta si se presionó 'C'."""
        if sys.platform == "win32":
            return msvcrt.kbhit() and msvcrt.getwch() == 'C'
        else:
            rlist, _, _ = select.select([sys.stdin], [], [], 0.1)
            return bool(rlist) and sys.stdin.read(1) == 'C'

    while True:
        if esperar_tecla():
            ventana = tk.Tk()
            ventana.title('INGRESO DE COMANDOS VÍA UDP')
            ventana.geometry('400x200+700+100')
            ventana.minsize(width=400, height=200)

            Cuadro1 = tk.Frame(ventana)
            Rotulo_ID = tk.Label(Cuadro1, text='Elija ID:   ', font='consolas 16')
            Rotulo_ID.pack(side=tk.LEFT, padx=10, pady=10)
            Cuadro1.pack()

            Cuadro2 = tk.Frame(ventana)
            Rotulo_CMD = tk.Label(Cuadro2, text='Ingrese CMD:', font='consolas 16')
            Rotulo_CMD.pack(side=tk.LEFT, padx=10, pady=10)
            Cuadro2.pack()

            ID_entrada = tk.StringVar()
            Desplegable_entrada = ttk.Combobox(Cuadro1, width=17, values=Lista_ID,
                                               postcommand=lambda: Desplegable_entrada.configure(values=Lista_ID),
                                               state='readonly', textvariable=ID_entrada)
            Desplegable_entrada.set('')
            Desplegable_entrada.pack(side=tk.LEFT)

            CMD_entrada = tk.Entry(Cuadro2)
            CMD_entrada.pack(side=tk.LEFT)

            boton = tk.Button(ventana, text='Enviar CMD', font='consolas 16 bold',
                              bg='orange', command=captura)
            boton.pack(padx=20, pady=20)

            ventana.mainloop()

#==============================================================================
#=============================== Programa Principal ===========================
#==============================================================================
threading.Thread(target=thread2).start()

open('Registro.txt', 'w').close()

s = socket.socket(family=socket.AF_INET, type=socket.SOCK_DGRAM)
s.bind(('', 9095))
print("Servidor UDP esta escuchando...")

Vector_ID = np.zeros((50, 1), dtype='<U3')
Matriz_Reg = np.zeros((50, 4), dtype='<U30')

Conta_Pos = 0

while True:
    try:
        Dato = s.recvfrom(1024)
        message = Dato[0]
        address = Dato[1]
        now = datetime.now()

        XOR_Suma = 0
        for ii in range(len(message) - 3):
            XOR_Suma ^= message[ii]
        XOR_Suma = XOR_Suma + 1 if XOR_Suma % 2 == 0 else XOR_Suma - 1
        if XOR_Suma == 10:
            XOR_Suma += 1

        if message[len(message) - 3] == XOR_Suma:
            if b'ACK' not in message:
                ID = str(message[0:3])[2:5]
                ID_Flag = 0
                for ii in range(50):
                    if Vector_ID[ii] == ID:
                        ID_Flag = 1
                        break
                if ID_Flag == 0:
                    Vector_ID[Conta_Pos] = ID
                    ii = Conta_Pos
                    Conta_Pos = (Conta_Pos + 1) % 50

                Matriz_Reg[ii] = [ID, address[0], str(address[1]), now.strftime("%A %d/%b %H:%M:%S")]
                np.savetxt('Registro.txt', Matriz_Reg, fmt='%s', delimiter=' ')
                Lista_ID = Vector_ID.tolist()[:ii]

                print(f'\n>>> KEEP ALIVE desde el Cliente: {message[:len(message) - 3]}CKCRLF')
                print(now.strftime("%A %H:%M:%S"))

                SendUDP(ID, 'C+DUMMY')
            else:
                if b'MEN 11' in message:
                    print('>>> Respuesta FORMATEADA desde el Cliente:')
                    for jj in range(9):
                        print(str(message[12 + jj * 14:25 + jj * 14]))
                elif b'STAT' in message:
                    print('>>> Respuesta FORMATEADA desde el Cliente:')
                    Stat = ',' + str(message)[15:-7] + ','
                    Renglon = ''
                    for char in Stat:
                        if char == ',':
                            if Renglon:
                                print(Renglon)
                                Renglon = ''
                        else:
                            Renglon += char
                else:
                    print(f'>>> Respuesta desde el Cliente: {message[:len(message) - 3]}CKCRLF')

                if b'NAME' in message and b'STAT' not in message:
                    ID_PAS = str(message[0:3])[2:5]
                    ID = str(message[13:16])[2:5]
                    for ii in range(50):
                        if Vector_ID[ii] == ID_PAS:
                            Vector_ID[ii] = ID
                            Matriz_Reg[ii][0] = ID
                            np.savetxt('Registro.txt', Matriz_Reg, fmt='%s', delimiter=' ')
        else:
            print('Dato entrante corrupto.\n')

    except Exception as error:
        print('Una excepcion ha occurrido:', error)
