# Ruta del directorio donde está el script
RUTA_SCRIPT="$(dirname "$0")"

# Cantidad aleatoria de archivos entre 1 y 4
CANTIDAD=$(( RANDOM % 4 + 1 ))

echo "Creando $CANTIDAD contenedor(es) en la ruta: $RUTA_SCRIPT"

# Generar archivos
for (( i=1; i<=CANTIDAD; i++ ))
do
    # Generar un nombre aleatorio (usamos /dev/urandom para 6 letras)
    NOMBRE_ALEATORIO=$(cat /dev/urandom | tr -dc 'a-zA-Z' | head -c 6)

    # Nombre del archivo
    ARCHIVO="contenedor_202300350_${NOMBRE_ALEATORIO}.txt"

    # Ruta completa
    RUTA_COMPLETA="$RUTA_SCRIPT/$ARCHIVO"

    # Crear archivo con su nombre como contenido
    echo "$ARCHIVO" > "$RUTA_COMPLETA"

    echo "Archivo creado: $ARCHIVO"
done

echo "Simulación completada ✅"
