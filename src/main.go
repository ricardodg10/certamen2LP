package main

import (
	"bufio"
	"fmt" //para imprimir y mostrar info en consola
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// estructura de dato para un Proceso
type Proceso struct {
	ID     int         //identificador del proceso
	Estado chan string //Estado = ["Nuevo", "Listo", "Bloqueado", "Ejecutando", "Saliente"]
	PC     chan int    //program counter
	InfoES string      //informacion de estado (nombre_archivo)
}

// estructura de dato para CPU
type CPU struct {
	CantidadNucleos       int     //cantidad de nucleos de la CPU (1 nucleo ejecuta 1 proceso)
	CantidadInstrucciones int     //cantidad de instrucciones a ejecutar antes del cambio de proceso
	Probabilidad          float32 //probabilidad 1/P en que un proceso pueda terminar su ejecucion antes
}

// estructura de dato para el Dispatcher
type Dispatcher struct {
	ColaDeListos     chan []Proceso //representa la Cola de Listos (proceso.Estado = "Listo")
	ColaDeBloqueados chan []Proceso //representa la Cola de Bloqueados (proceso.Estado = "Bloqueado")
}

// estructura de dato para un Nucleo
type Nucleo struct {
	NucleoID        int      //para identificar el nucleo
	IDprocesoActual chan int //para identificar el proceso que esta ejecutando el nucleo
	ArchivoSalida   string   //para almacenar el nombre del archivo de salida asociado al nucleo
}

// main
func main() {

	//variables para determinar el fin de la ejecucion
	done_simulacion := make(chan bool)

	//se instancia la cpu
	cpu := CPU{
		CantidadNucleos:       1,
		CantidadInstrucciones: 1,
		Probabilidad:          1 / 1,
	}

	var nucleos []Nucleo //para almacenar los nucleos creados

	// se crean la cantidad de nucleos que se especifico
	for i := 0; i < cpu.CantidadNucleos; i++ {

		nucleo := Nucleo{
			NucleoID:        i + 1,
			IDprocesoActual: make(chan int),
			ArchivoSalida:   "archivo_salida_nucleo_" + strconv.Itoa(i+1) + ".txt",
		}
		nucleos = append(nucleos, nucleo)
	}

	// se instancia el dispatcher con sus colas vacias
	dispatcher := Dispatcher{
		ColaDeListos:     make(chan []Proceso),
		ColaDeBloqueados: make(chan []Proceso),
	}

	//para leer archivo de creacion
	ruta_archivo_creacion := filepath.Join("..", "input", "creacion_procesos.txt")
	procesos := leer_creacion(ruta_archivo_creacion)

	//tiempo ejecutado
	segundos := make(chan int) //canal de tipo entero que se trasmite entre goroutines
	defer close(segundos)      //la funcion se termina de ejecutar cuando la funcion main termina su ejecucion
	go reloj(segundos)

	//se comienza la simulacion
	go comenzar_simulacion(procesos, segundos, done_simulacion, cpu, nucleos, dispatcher)

	//estas variables deben tener valor "true" para que la funcion main pueda terminar su ejecucion
	<-done_simulacion

	fmt.Println("Terminado")
}

// simula el tiempo en segundos
func reloj(segundosCH chan<- int) {
	segundos := 0
	for {
		segundosCH <- segundos
		segundos++
		time.Sleep(1 * time.Second)
	}
}

// para obtener id a cada proceso
func obtener_id_program_counter(nombre_archivo_proceso string) (int, int) {
	aux_1 := strings.Split(nombre_archivo_proceso, "_")
	aux_2 := aux_1[1]
	aux_3 := strings.TrimSuffix(aux_2, ".txt")
	id_proceso, err := strconv.Atoi(aux_3)

	if err != nil {
		fmt.Println("Error al obtener el id del proceso.")
	}

	program_counter := (id_proceso + 1) * 100 //esto para evitar que los procesos tomen las instrucciones "100" que corresponden al dispatcher. Ej: el proceso 1 comienza en la direccion 200

	return id_proceso, program_counter

}

// para leer el archivo de creacion de procesos
func leer_creacion(ruta string) [][]string {

	var procesos [][]string //para almacenar los tiempos con los procesos asociados a crear

	// abre el archivo
	archivo, err := os.Open(ruta)
	if err != nil {
		fmt.Println("Error al abrir el archivo:", err)
		return procesos
	}
	defer archivo.Close()

	// recorre el archivo
	scanner := bufio.NewScanner(archivo)
	for scanner.Scan() {
		texto := scanner.Text()
		if !strings.HasPrefix(texto, "#") {
			linea := strings.Fields(texto) //divide la linea del txt en palabras

			if err != nil {
				fmt.Println("Error al convertir el primer valor a entero.")
			}

			linea_array := make([]string, len(linea)) //se crea el slice para almacenar el tiempo y los nombres de los archivos

			for i := 0; i < len(linea); i++ {

				if i == 0 {
					linea_array[i] = linea[i] //para almacenar el tiempo de creacion de procesos
				} else {
					linea_array[i] = linea[i] + ".txt" //para almacenar el nombre del archivo correspondiente al proceso
				}

			}

			procesos = append(procesos, linea_array)
		}
	}

	return procesos
}

// para comenzar la simulacion y llamar a la funcion de crear procesos
func comenzar_simulacion(procesos_para_crear [][]string, segundosCH chan int, done_simulacion chan bool, cpu CPU, nucleos []Nucleo, dispatcher Dispatcher) {

	tiempo_actual := <-segundosCH

	for _, procesos := range procesos_para_crear {
		tiempo_creacion, err := strconv.Atoi(procesos[0])

		if err != nil {
			fmt.Println("Error al convertir el string a int:", err)
			return
		}

		for {
			if tiempo_creacion <= tiempo_actual { //cuando el tiempo para crear los procesos sea menor o igual al tiempo actual, se crean
				go crear_procesos(procesos, dispatcher, nucleos, cpu)
				break
			}

			tiempo_actual = <-segundosCH
		}

	}

	done_simulacion <- true

}

// para instanciar los procesos
func crear_procesos(procesos_para_crear []string, dispatcher Dispatcher, nucleos []Nucleo, cpu CPU) {

	for i := 1; i < len(procesos_para_crear); i++ {

		id_proceso, program_counter_inicial := obtener_id_program_counter(procesos_para_crear[i])

		nuevoProceso := Proceso{
			ID:     id_proceso,
			Estado: make(chan string),
			PC:     make(chan int),
			InfoES: procesos_para_crear[i],
		}

		nuevoProceso.Estado <- "Nuevo"
		nuevoProceso.PC <- program_counter_inicial

		admision(dispatcher, nuevoProceso, nucleos, cpu)

	}

}

// admision (para comenzar a llevarlos a las respectivas colas del dispatcher)
func admision(dispatcher Dispatcher, proceso Proceso, nucleos []Nucleo, cpu CPU) {

	proceso.Estado <- "Listo"
	cola_de_listos := <-dispatcher.ColaDeListos
	cola_de_listos = append(cola_de_listos, proceso)
	dispatcher.ColaDeListos <- cola_de_listos

	go transiciones(dispatcher, nucleos, cpu)

}

// para leer los archivos correspondientes a cada proceso
func ejecutar_instrucciones() {

}

// para realizar las transiciones
func transiciones(dispatcher Dispatcher, nucleos []Nucleo, cpu CPU) {

}
