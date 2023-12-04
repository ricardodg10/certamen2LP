Dentro del directorio "src" se debe escribir la siguiente linea de comando para ejecutar el código:

go run main.go A B C D

Donde:

A es la cantidad de núcleos de la simulacion (número entero mayor a cero)
B es la cantidad de instrucciones que se pueden ejecutar por cada núcleo (número entero mayor a cero)
C es la probabilidad (1/C) de que el proceso termine su ejecución de manera temprana (número entero mayor a cero)
D es el nombre del archivo para crear los procesos en un tiempo determinado (este archivo debe ser .txt y encontrarse dentro del directorio "input" del proyecto)

Ejemplo:

C:\Users\ricar\OneDrive\Escritorio\CERTAMEN2-LENGUAJES\Certamen-2-CarrascoDiazElgueta-V7\src> go run main.go 5 10 1 creacion_procesos.txt

Donde:

- Hay 5 núcleos
- Son 10 las instrucciones que se pueden ejecutar antes de que se cambie de proceso
- La probabilidad de que finalice un proceso antes es de 1/1
- El archivo donde se especifican los tiempos de creacion para los procesos se llama "creacion_procesos.txt"

Sobre los archivos .txt en el directorio "output"

Se crearán los archivos .txt con la traza de la ejecución de los procesos respecto al número de núcleos especificado en el input por consola.
Actualmente, en el directorio se entregan cinco archivos .txt con la traza de 5 núcleos. Si se vuelve a ejecutar el código, se recomienda borrar estos 
archivos antes de la ejecución para evitar posibles confusiones (cabe mencionar que si se ejecuta con estos archivos en el directorio, su contenido se 
sobrescribirá con el procedimiento de la última ejecución del código)