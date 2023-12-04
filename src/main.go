package main

import (
	"bufio"
	"fmt" //para imprimir y mostrar info en consola
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var mutex sync.Mutex //variable global

// estructura de dato para un Proceso
type Proceso struct {
	ID            int        //identificador del proceso
	Estado        string     //Estado = ["Nuevo", "Listo", "Bloqueado", "Ejecutando", "Saliente"]
	PC            int        //program counter
	InfoES        string     //informacion (nombre_archivo)
	Instrucciones [][]string //instrucciones del proceso
}

// estructura de dato para CPU
type CPU struct {
	CantidadNucleos       int     //cantidad de nucleos de la CPU (1 nucleo ejecuta 1 proceso)
	CantidadInstrucciones int     //cantidad de instrucciones a ejecutar antes del cambio de proceso
	Probabilidad          float32 //probabilidad 1/P en que un proceso pueda terminar su ejecucion antes
}

// estructura de dato para el Dispatcher
type Dispatcher struct {
	ColaDeListos     []*Proceso //representa la Cola de Listos (proceso.Estado = "Listo")
	ColaDeBloqueados []*Proceso //representa la Cola de Bloqueados (proceso.Estado = "Bloqueado")
}

// estructura de dato para un Nucleo
type Nucleo struct {
	NucleoID        int    //para identificar el nucleo
	IDprocesoActual int    //para identificar el proceso que esta ejecutando el nucleo
	ArchivoSalida   string //para almacenar el nombre del archivo de salida asociado al nucleo
}

// para inicializar un proceso
func crear_proceso(id_proceso int, program_counter int, nombre_archivo string, instrucciones [][]string) *Proceso {

	return &Proceso{
		ID:            id_proceso,
		Estado:        "Nuevo",
		PC:            program_counter,
		InfoES:        nombre_archivo,
		Instrucciones: instrucciones,
	}
}

// para inicializar una CPU
// estos valores se reciben por consola
func crear_cpu(nucleos int, instrucciones int, probabilidad float32) *CPU {
	return &CPU{
		CantidadNucleos:       nucleos,
		CantidadInstrucciones: instrucciones,
		Probabilidad:          1 / probabilidad,
	}
}

// para inicializar un Dispatcher
func crear_dispatcher() *Dispatcher {
	return &Dispatcher{
		ColaDeListos:     make([]*Proceso, 0), //se crea slice de tamaño cero (en un comienzo)
		ColaDeBloqueados: make([]*Proceso, 0), //se crea slice de tamaño cero (en un comienzo)
	}
}

// para inicializar un Nucleo
// la variable i se recibe desde un for
func crear_nucleo(i int) *Nucleo {
	return &Nucleo{
		NucleoID:        i + 1,
		IDprocesoActual: 0,
		ArchivoSalida:   "archivo_salida_nucleo_" + strconv.Itoa(i+1) + ".txt",
	}
}

// main
func main() {

	//variables para determinar el fin de la ejecucion
	done_creacion_procesos := make(chan bool)
	done_ejecucion_dispatcher := make(chan bool)

	var nucleos []*Nucleo //para almacenar los nucleos creados

	//se instancia la cpu
	cpu := crear_cpu(2, 4, 1)

	// se crean la cantidad de nucleos que se especifico
	for i := 0; i < cpu.CantidadNucleos; i++ {

		nucleo := crear_nucleo(i)
		nucleos = append(nucleos, nucleo)
	}

	// se instancia el dispatcher
	dispatcher := crear_dispatcher()

	//para leer archivo de creacion
	ruta_archivo_creacion := filepath.Join("..", "input", "creacion_procesos.txt")
	procesos := leer_creacion(ruta_archivo_creacion)

	//tiempo ejecutado
	segundos := make(chan int) //canal de tipo entero que se trasmite entre goroutines
	defer close(segundos)      //la funcion se termina de ejecutar cuando la funcion main termina su ejecucion
	go reloj(segundos)

	//se comienza la simulacion
	go comenzar_simulacion(procesos, segundos, done_creacion_procesos, done_ejecucion_dispatcher, cpu, nucleos, dispatcher)

	//estas variables deben tener valor "true" para que la funcion main pueda terminar su ejecucion
	<-done_creacion_procesos
	<-done_ejecucion_dispatcher

	fmt.Println("\n[ Simulación terminada ]")
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

// para obtener id y program counter de cada proceso
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

// para almacenar las instrucciones del proceso del archivo txt al proceso asociado
func obtener_instrucciones(proceso_txt string) [][]string {

	ruta_archivo_creacion := filepath.Join("..", "input", proceso_txt)

	var instrucciones [][]string //para almacenar las instrucciones

	// abre el archivo
	archivo, err := os.Open(ruta_archivo_creacion)
	if err != nil {
		fmt.Println("Error al abrir el archivo:", err)
		return instrucciones
	}
	defer archivo.Close()

	// recorre el archivo
	scanner := bufio.NewScanner(archivo)
	for scanner.Scan() {
		texto := scanner.Text()
		if !strings.HasPrefix(texto, "#") {
			linea := strings.Fields(texto) //divide la linea del txt en palabras

			linea_array := make([]string, len(linea)) //se crea el slice para almacenar las instrucciones

			for i := 0; i < len(linea); i++ {

				linea_array[i] = linea[i] //para almacenar la linea con el numero de instruccion y el tipo de instruccion

			}

			instrucciones = append(instrucciones, linea_array)
		}
	}

	return instrucciones
}

// para crear uno o mas procesos cuando corresponda
func crear_procesos(procesos_para_crear []string, dispatcher *Dispatcher) {
	for i := 1; i < len(procesos_para_crear); i++ {

		id_proceso, program_counter_inicial := obtener_id_program_counter(procesos_para_crear[i])

		instrucciones_proceso := obtener_instrucciones(procesos_para_crear[i])

		nuevo_proceso := crear_proceso(id_proceso, program_counter_inicial, procesos_para_crear[i], instrucciones_proceso)

		fmt.Printf("Proceso Creado {ID: %d, Estado: %s}.\n", nuevo_proceso.ID, nuevo_proceso.Estado)
		admision(dispatcher, nuevo_proceso)
	}
}

// cambia a estado "Listo" y se lleva a la Cola de Listos
func admision(dispatcher *Dispatcher, proceso *Proceso) {

	proceso.Estado = "Listo"

	fmt.Printf("Proceso Admitido {{ID: %d, Estado: %s}.\n", proceso.ID, proceso.Estado)

	//se define la Cola de Listos como una estructura crítica, entonces se bloque el mutex para que
	//otra rutina no pueda acceder al recurso de Cola de Listos hasata que se desbloquee
	//el mutex se desbloquea cuando se agrega el proceso a la Cola de Listos

	mutex.Lock()
	defer mutex.Unlock()

	dispatcher.ColaDeListos = append(dispatcher.ColaDeListos, proceso)
}

// saca el proceso de la Cola De Listos y lo comienza a ejecutar
func activacion(dispatcher *Dispatcher, proceso *Proceso, nucleo *Nucleo, variable_cantidad_nucleos_ejecucion *int, cantidad_instrucciones int) {

	instrucciones_ejecutadas := 0
	proceso.Estado = "Ejecutando"
	fmt.Printf("Proceso Ejecutando {ID: %d, Estado: %s, Nucleo asociado: %d}.\n", proceso.ID, proceso.Estado, nucleo.NucleoID)

	for _, instruccion := range proceso.Instrucciones {
		proceso.PC = proceso.PC + 1
		fmt.Println(proceso.PC, instruccion)
		instrucciones_ejecutadas = instrucciones_ejecutadas + 1
		proceso.Instrucciones = proceso.Instrucciones[1:]

		if instrucciones_ejecutadas == cantidad_instrucciones {
			mutex.Lock()
			dispatcher.ColaDeListos = append(dispatcher.ColaDeListos, proceso)
			mutex.Unlock()
			break
		}

	}

	nucleo.IDprocesoActual = 0

	mutex.Lock()
	*variable_cantidad_nucleos_ejecucion = *variable_cantidad_nucleos_ejecucion - 1
	mutex.Unlock()

}

// cambia a estado "Bloqueado" y se lleva a la Cola de Bloqueados
func esperando_por_evento(dispatcher *Dispatcher, proceso *Proceso) {
	proceso.Estado = "Bloqueado"

	fmt.Printf("Proceso Bloqueado {{ID: %d, Estado: %s}.\n", proceso.ID, proceso.Estado)

	dispatcher.ColaDeBloqueados = append(dispatcher.ColaDeBloqueados, proceso)
}

// cambia desde estado "Bloqueado" a "Listo" y se lleva a la Cola de Listos nuevamente
func evento_sucede() {

}

// para comenzar la simulacion y llamar a la funcion de crear procesos
func comenzar_simulacion(procesos_para_crear [][]string, segundosCH chan int, done_creacion chan bool, done_despachador chan bool, cpu *CPU, nucleos []*Nucleo, dispatcher *Dispatcher) {

	fmt.Println("[ Comenzando simulación ]")
	fmt.Print("\n")

	go iniciar_despachador(dispatcher, nucleos, cpu, len(procesos_para_crear), segundosCH, done_despachador)

	tiempo_actual := <-segundosCH

	for _, procesos := range procesos_para_crear {
		tiempo_creacion, err := strconv.Atoi(procesos[0])

		if err != nil {
			fmt.Println("Error al convertir el string a int:", err)
			return
		}

		for {

			if tiempo_creacion <= tiempo_actual { //cuando el tiempo para crear los procesos sea menor o igual al tiempo actual, se crean
				crear_procesos(procesos, dispatcher)
				break
			}

			tiempo_actual = <-segundosCH
		}

	}

	done_creacion <- true
}

func iniciar_despachador(dispatcher *Dispatcher, nucleos []*Nucleo, cpu *CPU, procesos_a_ejecutar int, segundosCH chan int, done_despachador chan bool) {

	var instrucciones int
	tiempo_actual := <-segundosCH
	nucleos_en_ejecucion := 0
	instrucciones = cpu.CantidadInstrucciones

	//rutina para condicionar que ya no hay más procesos que ejecutar
	go func() {
		for {
			tiempo_actual = <-segundosCH

			// las colas se consideran recursos críticos que pueden llevar a una condicion de carrera
			// se bloquea el mutex para obtener el tamaño de ambas colas y luego se desbloquea
			mutex.Lock()
			cantidad_procesos_listos := len(dispatcher.ColaDeListos)
			cantidad_procesos_bloqueados := len(dispatcher.ColaDeBloqueados)
			mutex.Unlock()

			if nucleos_en_ejecucion == 0 && cantidad_procesos_listos == 0 && cantidad_procesos_bloqueados == 0 && tiempo_actual > 10 {
				done_despachador <- true
				return
			}

		}
	}()

	// rutina para monitorear la cola de listos y ejecutar procesos
	go func() {
		for {

			mutex.Lock()
			cantidad_procesos_listos := len(dispatcher.ColaDeListos)
			mutex.Unlock()

			// si la cantidad de nucleos es menor o igual a la cantidad de nucleos creados y existen procesos listos
			// para ejecutarse, se busca el primer nucleo del slice que no tenga un proceso asignado y se lleva junto
			// con el primer proceso de la cola de listos para su ejecucion

			if nucleos_en_ejecucion <= len(nucleos) && cantidad_procesos_listos > 0 {
				for _, nucleo := range nucleos {
					if nucleo.IDprocesoActual == 0 { //que el nucleo tenga un id de proceso cero, significa que no está ejecutando algun proceso (se encuentra disponible)
						mutex.Lock()                                           //la condicion de carrera se asocia a obtener los procesos en la cola de listos, por lo que se debe bloquear
						nucleo.IDprocesoActual = dispatcher.ColaDeListos[0].ID //se asocia el proceso al nucleo que lo ejecutara
						nucleos_en_ejecucion = nucleos_en_ejecucion + 1
						proceso_a_ejecutar := dispatcher.ColaDeListos[0] //se hace una copia del proceso antes de sacarlo de la cola de listos
						dispatcher.ColaDeListos = dispatcher.ColaDeListos[1:]
						go activacion(dispatcher, proceso_a_ejecutar, nucleo, &nucleos_en_ejecucion, instrucciones)
						mutex.Unlock() // se desbloquea una vez que se lleva el proceso a ejecutar
						break

					}
				}

			}
		}

	}()

	//rutina para monitorear la cola de bloqueados
	go func() {

	}()

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
