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
	Espera        int        //para definir cuantas instrucciones debe esperar antes de desbloquearse
	NombreProceso string     //almacena el nombre del proceso para la generacion de los archivos txt
}

// estructura de dato para CPU
type CPU struct {
	CantidadNucleos       int //cantidad de nucleos de la CPU (1 nucleo ejecuta 1 proceso)
	CantidadInstrucciones int //cantidad de instrucciones a ejecutar antes del cambio de proceso
	Probabilidad          int //probabilidad 1/P en que un proceso pueda terminar su ejecucion antes
}

// estructura de dato para el Dispatcher
type Dispatcher struct {
	ColaDeListos     []*Proceso //representa la Cola de Listos (proceso.Estado = "Listo")
	ColaDeBloqueados []*Proceso //representa la Cola de Bloqueados (proceso.Estado = "Bloqueado")
}

// estructura de dato para un Nucleo
type Nucleo struct {
	NucleoID        int      //para identificar el nucleo
	IDprocesoActual int      //para identificar el proceso que esta ejecutando el nucleo
	ArchivoSalida   string   //para almacenar el nombre del archivo de salida asociado al nucleo
	Traza           []string //para guardar la traza de las instrucciones que se ejecutan en relacion al nucleo
}

// para inicializar un proceso
// se define Espera: 0, ya que en un comienzo no se encuentra bloqueado
func crear_proceso(id_proceso int, program_counter int, nombre_archivo string, instrucciones [][]string) *Proceso {

	return &Proceso{
		ID:            id_proceso,
		Estado:        "Nuevo",
		PC:            program_counter,
		InfoES:        nombre_archivo,
		Instrucciones: instrucciones,
		Espera:        0,
		NombreProceso: strings.TrimSuffix(nombre_archivo, ".txt"), //el nombre del proceso está dado por el nombre del archivo sin la extension ".txt"
	}
}

// para inicializar una CPU
// estos valores se reciben por consola
func crear_cpu(nucleos int, instrucciones int, probabilidad int) *CPU {
	return &CPU{
		CantidadNucleos:       nucleos,
		CantidadInstrucciones: instrucciones,
		Probabilidad:          probabilidad,
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
		Traza:           make([]string, 0),
	}
}

// main
func main() {

	//obtener argumentos de la linea de comando

	argumentos := os.Args

	//si son 6 los argumentos obtenidos en la linea de comando
	n, error1 := strconv.Atoi(argumentos[1]) //cantidad de nucleos
	o, error2 := strconv.Atoi(argumentos[2]) //cantidad de instrucciones
	P, error3 := strconv.Atoi(argumentos[3]) //para calcular probabilidad

	if error1 != nil || error2 != nil || error3 != nil {
		fmt.Println("Error al obtener el valor de 'n', 'o' ó 'P'.")
	}

	archivo_creacion := argumentos[4] //nombre del archivo de creacion (debe estar dentro del directorio input)

	fmt.Printf("\nCantidad de nucleos: %d\n", n)
	fmt.Printf("Cantidad de instrucciones: %d\n", o)
	fmt.Printf("Probabilidad: 1/%d\n", P)
	fmt.Printf("Archivo de creacion: %s\n\n", archivo_creacion)

	//variables para determinar el fin de la ejecucion
	done_creacion_procesos := make(chan bool)
	done_ejecucion_dispatcher := make(chan bool)

	var nucleos []*Nucleo //para almacenar los nucleos creados

	//se instancia la cpu
	cpu := crear_cpu(n, o, P)

	// se crean la cantidad de nucleos que se especifico
	for i := 0; i < cpu.CantidadNucleos; i++ {

		nucleo := crear_nucleo(i)
		nucleos = append(nucleos, nucleo)
	}

	// se instancia el dispatcher
	dispatcher := crear_dispatcher()

	//para leer archivo de creacion
	ruta_archivo_creacion := filepath.Join("..", "input", archivo_creacion)
	procesos, cantidad_procesos := leer_creacion(ruta_archivo_creacion)

	//tiempo ejecutado
	segundos := make(chan int) //canal de tipo entero que se trasmite entre goroutines
	defer close(segundos)      //la funcion se termina de ejecutar cuando la funcion main termina su ejecucion
	go reloj(segundos)

	//se comienza la simulacion
	go comenzar_simulacion(procesos, cantidad_procesos, segundos, done_creacion_procesos, done_ejecucion_dispatcher, cpu, nucleos, dispatcher)

	//estas variables deben tener valor "true" para que la funcion main pueda terminar su ejecucion
	<-done_creacion_procesos
	<-done_ejecucion_dispatcher

	generar_trazas(nucleos)

	fmt.Println("\n[ Simulación terminada ]")

}

// para leer el archivo de creacion de procesos
func leer_creacion(ruta string) ([][]string, int) {

	var cantidad_procesos int
	cantidad_procesos = 0

	var procesos [][]string //para almacenar los tiempos con los procesos asociados a crear

	// abre el archivo
	archivo, err := os.Open(ruta)
	if err != nil {
		fmt.Println("Error al abrir el archivo:", err)
		return procesos, cantidad_procesos
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
					cantidad_procesos = cantidad_procesos + 1
				}

			}

			procesos = append(procesos, linea_array)
		}
	}
	return procesos, cantidad_procesos
}

// escribe en un archivo la traza de cada nucleo
func generar_trazas(nucleos []*Nucleo) error {
	output_directorio := "../output"

	// crear directorio de salida si no existe
	if err := os.MkdirAll(output_directorio, os.ModePerm); err != nil {
		return err
	}

	for _, nucleo := range nucleos {
		output_ruta := filepath.Join(output_directorio, nucleo.ArchivoSalida)

		archivo, err := os.Create(output_ruta)
		if err != nil {
			return err
		}
		defer archivo.Close()

		escritor := bufio.NewWriter(archivo)

		// Escribir encabezado en el archivo
		primera_linea := "# Tiempo de CPU | Tipo de Instruccion | Proceso / Despachador | Valor Program Counter"
		fmt.Fprintln(escritor, primera_linea)

		// Escribir trazas en el archivo
		for _, traza := range nucleo.Traza {
			fmt.Fprintln(escritor, traza)
		}

		escritor.Flush()
	}

	fmt.Println("¡Trazas generadas!")
	return nil
}

// para obtener id y program counter de cada proceso
func obtener_id_program_counter(nombre_archivo_proceso string) (int, int) {
	//se obtiene el id a partir del numero que hay en el nombre del archivo del proceso. Ej: proceso_1.txt → id: 1
	aux_1 := strings.Split(nombre_archivo_proceso, "_")
	aux_2 := aux_1[1]
	aux_3 := strings.TrimSuffix(aux_2, ".txt")
	id_proceso, err := strconv.Atoi(aux_3)

	if err != nil {
		fmt.Println("Error al obtener el id del proceso.")
	}

	program_counter := (id_proceso + 1) * 100 //esto para evitar que los procesos tomen las instrucciones "100" que corresponden al dispatcher. Ej: el proceso 1 se almacena en la direccion 200

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

		//se crean el o los procesos que se especifican en un tiempo determinado

		id_proceso, program_counter_inicial := obtener_id_program_counter(procesos_para_crear[i])

		instrucciones_proceso := obtener_instrucciones(procesos_para_crear[i])

		nuevo_proceso := crear_proceso(id_proceso, program_counter_inicial, procesos_para_crear[i], instrucciones_proceso)

		fmt.Printf("Proceso Creado {ID: %d, Estado: %s}.\n", nuevo_proceso.ID, nuevo_proceso.Estado)
		admision(dispatcher, nuevo_proceso)
	}
}

// para almacenar la traza ST
func st(proceso *Proceso, nucleo *Nucleo, segundosCH chan int) {

	tiempo := <-segundosCH
	nucleo.Traza = append(nucleo.Traza, strconv.Itoa(tiempo)+" ST "+proceso.NombreProceso+" Dispatcher 101")
}

// cambia a estado "Listo" y se lleva a la Cola de Listos
func admision(dispatcher *Dispatcher, proceso *Proceso) {

	proceso.Estado = "Listo"

	fmt.Printf("Proceso Admitido {ID: %d, Estado: %s}.\n", proceso.ID, proceso.Estado)

	//se define la Cola de Listos como una estructura crítica, entonces se bloque el mutex para que
	//otra rutina no pueda acceder al recurso de Cola de Listos hasata que se desbloquee
	//el mutex se desbloquea cuando se agrega el proceso a la Cola de Listos

	mutex.Lock()
	defer mutex.Unlock()

	dispatcher.ColaDeListos = append(dispatcher.ColaDeListos, proceso)
}

// saca el proceso de la Cola De Listos y lo comienza a ejecutar
func activacion(dispatcher *Dispatcher, proceso *Proceso, nucleo *Nucleo, variable_cantidad_nucleos_ejecucion *int, cantidad_instrucciones int, procesos_ejecutados *int, segundosCH chan int) {

	// para simular EXEC (se simula al comenzar la ejecucion en activacion)
	tiempo := <-segundosCH

	// formato de traza para EXEC: TiempoCPU EXEC-nombre_proceso Dispatcher ValorProgramCounter
	// supuesto: el program counter para un LOAD es 105
	exec_proceso := strconv.Itoa(tiempo) + " EXEC-" + proceso.NombreProceso + " Dispatcher 105"
	nucleo.Traza = append(nucleo.Traza, exec_proceso)

	instrucciones_ejecutadas := 0
	proceso.Estado = "Ejecutando"
	fmt.Printf("Proceso Ejecutando {ID: %d, Estado: %s, Nucleo asociado: %d}.\n", proceso.ID, proceso.Estado, nucleo.NucleoID)

	for _, instruccion := range proceso.Instrucciones {

		//para simular I / ES
		tiempo := <-segundosCH

		// formato de traza para I: TiempoCPU I nombre_proceso ValorProgramCounter
		proceso.PC = proceso.PC + 1
		var instruccion_proceso string

		if len(instruccion) == 2 {
			instruccion_proceso = strconv.Itoa(tiempo) + " " + instruccion[1] + " " + proceso.NombreProceso + " " + strconv.Itoa(proceso.PC) //correspondiente a I
		} else {
			instruccion_proceso = strconv.Itoa(tiempo) + " " + instruccion[1] + " " + instruccion[2] + " " + proceso.NombreProceso + " " + strconv.Itoa(proceso.PC) //correspondiente a ES x, donde x es la cantidad de instrucciones que debe esperar para desbloquearse
		}

		nucleo.Traza = append(nucleo.Traza, instruccion_proceso)

		fmt.Println(proceso.PC, instruccion)

		//si la instruccion es E/S
		if instruccion[1] == "ES" {
			espera, err := strconv.Atoi(instruccion[2])

			if err != nil {
				fmt.Println("Error al convertir el string.", err)
			} else {
				proceso.Instrucciones = proceso.Instrucciones[1:]
				proceso.Espera = espera
				//se bloquea
				go esperando_por_evento(dispatcher, nucleo, proceso, segundosCH)
				break
			}
		}

		//por cada instruccion "recorrida" se aumenta en uno la variable de instrucciones ejecutadas y se elimina esta instruccion en el slice del proceso
		instrucciones_ejecutadas = instrucciones_ejecutadas + 1
		proceso.Instrucciones = proceso.Instrucciones[1:]

		//para la finalizacion del proceso
		if instruccion[len(instruccion)-1] == "F" {
			proceso.Estado = "Saliente"
			fmt.Printf("Proceso Finalizado {ID: %d, Estado: %s}.\n", proceso.ID, proceso.Estado)
			*procesos_ejecutados = *procesos_ejecutados + 1
			break
		}

		//si se llega a la cantidad maxima de instrucciones a ejecutar (ingresada como parametro) se manda el proceso a la cola de listos
		if instrucciones_ejecutadas == cantidad_instrucciones {
			temporizacion(dispatcher, nucleo, proceso, segundosCH)
			break
		}

	}

	// se especifica que el nucleo queda disponible para otro proceso
	*variable_cantidad_nucleos_ejecucion = *variable_cantidad_nucleos_ejecucion - 1
	time.Sleep(1 * time.Second)
	nucleo.IDprocesoActual = 0

}

// cambia desde estado "Ejecutando" a "Listo" y los manda a la Cola de Listos
func temporizacion(dispatcher *Dispatcher, nucleo *Nucleo, proceso *Proceso, segundosCH chan int) {

	//guardar estado
	proceso.Estado = "Listo"
	st(proceso, nucleo, segundosCH)

	//PULL
	tiempo := <-segundosCH
	nucleo.Traza = append(nucleo.Traza, strconv.Itoa(tiempo)+" PUSH_Listo "+proceso.NombreProceso+" Dispatcher 103")

	fmt.Printf("Proceso Temporizado {{ID: %d, Estado: %s}.\n", proceso.ID, proceso.Estado)

	// la condicion de carrera que se puede generar es que varias rutinas intenten acceder
	// a la Cola de Listos, por lo que se utiliza mutex para agregarlos a este slice
	mutex.Lock()
	dispatcher.ColaDeListos = append(dispatcher.ColaDeListos, proceso)
	mutex.Unlock()
}

// cambia desde estado "Ejecutando" a "Bloqueado" y se lleva el proceso a la Cola de Bloqueados
func esperando_por_evento(dispatcher *Dispatcher, nucleo *Nucleo, proceso *Proceso, segundosCH chan int) {

	proceso.Estado = "Bloqueado"
	st(proceso, nucleo, segundosCH)

	fmt.Printf("Proceso Bloqueado {{ID: %d, Estado: %s}.\n", proceso.ID, proceso.Estado)

	// la condicion de carrera que se puede generar es que varias rutinas intenten acceder
	// a la Cola de Bloqueados, por lo que se utiliza mutex para agregarlos a este slice
	mutex.Lock()
	tiempo := <-segundosCH
	nucleo.Traza = append(nucleo.Traza, strconv.Itoa(tiempo)+" PUSH_Bloqueado "+proceso.NombreProceso+" Dispatcher 102")
	dispatcher.ColaDeBloqueados = append(dispatcher.ColaDeBloqueados, proceso)
	mutex.Unlock()
}

// cambia desde estado "Bloqueado" a "Listo" y se lleva a la Cola de Listos nuevamente
func evento_sucede(proceso *Proceso, i int, dispatcher *Dispatcher) {

	proceso_a_listos := proceso
	proceso_a_listos.Estado = "Listo"

	dispatcher.ColaDeBloqueados = append(dispatcher.ColaDeBloqueados[:i], dispatcher.ColaDeBloqueados[i+1:]...) //elimina el proceso de la cola de bloqueados
	dispatcher.ColaDeListos = append(dispatcher.ColaDeListos, proceso_a_listos)                                 //agrega el proceso a la cola de listos

}

// para comenzar la simulacion y llamar a la funcion de crear procesos
func comenzar_simulacion(procesos_para_crear [][]string, cantidad_procesos int, segundosCH chan int, done_creacion chan bool, done_despachador chan bool, cpu *CPU, nucleos []*Nucleo, dispatcher *Dispatcher) {

	fmt.Println("[ Comenzando simulación ]")
	fmt.Print("\n")

	go iniciar_despachador(dispatcher, nucleos, cpu, cantidad_procesos, segundosCH, done_despachador)

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

// al iniciar el despachador se tienen tres rutinas en paralelo:
// 1. monitorea las condiciones de termino
// 2. monitorea la cola de listos y manda a ejecutar los procesos
// 3. monitorea la cola de bloqueados y los desbloquea cuando se debe
func iniciar_despachador(dispatcher *Dispatcher, nucleos []*Nucleo, cpu *CPU, procesos_a_ejecutar int, segundosCH chan int, done_despachador chan bool) {

	// se obtiene la cantidad maxima de instrucciones a ejecutar antes del cambio con otro proceso
	var instrucciones int
	instrucciones = cpu.CantidadInstrucciones

	// se define una variable para llevar la cuenta de cuantos procesos se han ejecutado otalmente
	// esta variable se compara con la cantidad de procesos que se deben ejecutar para darle un
	// fin a la simulacion
	var procesos_finalizados int
	procesos_finalizados = 0

	tiempo_actual_2 := <-segundosCH

	nucleos_en_ejecucion := 0 //variable que lleva la cuenta de cuantos nucleos se encuentran activos

	//rutina para condicionar que ya no hay más procesos que ejecutar
	go func() {
		for {

			// las colas se consideran recursos críticos que pueden llevar a una condicion de carrera
			// se bloquea el mutex para obtener el tamaño de ambas colas y luego se desbloquea
			mutex.Lock()
			cantidad_procesos_listos := len(dispatcher.ColaDeListos)
			cantidad_procesos_bloqueados := len(dispatcher.ColaDeBloqueados)
			mutex.Unlock()

			if nucleos_en_ejecucion == 0 && cantidad_procesos_listos == 0 && cantidad_procesos_bloqueados == 0 && procesos_finalizados == procesos_a_ejecutar {
				done_despachador <- true
				return
			}

		}
	}()

	// rutina para monitorear la cola de listos y ejecutar procesos
	go func() {
		for {
			//tiempo_actual_2 = <-segundosCH
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

						//para simular PULL
						tiempo_actual_2 = <-segundosCH                   //se obtiene el tiempo actual al momento de sacar el proceso de la cola de listos
						proceso_a_ejecutar := dispatcher.ColaDeListos[0] //se hace una copia del proceso antes de sacarlo de la cola de listos

						// formato de traza para PULL: TiempoCPU PULL Dispatcher ValorProgramCounter
						// supuesto: el program counter para un PULL es 103
						pull_proceso := strconv.Itoa(tiempo_actual_2) + " PULL Dispatcher 103"
						nucleo.Traza = append(nucleo.Traza, pull_proceso)

						dispatcher.ColaDeListos = dispatcher.ColaDeListos[1:]

						// para simular LOAD (se simula llamando a la funcion con go)
						tiempo_actual_2 = <-segundosCH //se obtiene el tiempo actual al momento de cargar el proceso

						// formato de traza para LOAD: TiempoCPU LOAD-nombre_proceso Dispatcher ValorProgramCounter
						// supuesto: el program counter para un LOAD es 104
						load_proceso := strconv.Itoa(tiempo_actual_2) + " LOAD-" + proceso_a_ejecutar.NombreProceso + " Dispatcher 104"
						nucleo.Traza = append(nucleo.Traza, load_proceso)

						go activacion(dispatcher, proceso_a_ejecutar, nucleo, &nucleos_en_ejecucion, instrucciones, &procesos_finalizados, segundosCH)
						mutex.Unlock() // se desbloquea una vez que se lleva el proceso a ejecutar
						break

					}
				}

			}
		}

	}()

	//rutina para monitorear la cola de bloqueados
	go func() {
		for {
			//recorre la cola de bloqueados y resta 1 a la espera de cada proceso (cada un segundo), cuando la espera es cero, significa que el proceso puede ser desbloqueado
			// y llevado a la cola de listos
			time.Sleep(1 * time.Second)
			for i := 0; i < len(dispatcher.ColaDeBloqueados); i++ {
				if dispatcher.ColaDeBloqueados[i].Espera > 0 {
					dispatcher.ColaDeBloqueados[i].Espera = dispatcher.ColaDeBloqueados[i].Espera - 1
				} else {
					mutex.Lock()
					evento_sucede(dispatcher.ColaDeBloqueados[i], i, dispatcher)
					mutex.Unlock()
				}

			}
		}

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
