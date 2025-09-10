#!/bin/bash

# Verificar si Docker está instalado
if ! [ -x "$(command -v docker)" ]; then
  echo 'Error: Docker no está instalado.' >&2
  exit 1
fi

# Imágenes disponibles
IMAGES=("low:latest" "cpu_stress:latest" "ram:latest")

# Lista final de contenedores a crear
CONTAINERS_TO_CREATE=()

# Asegurar requisitos mínimos
CONTAINERS_TO_CREATE+=("low:latest")
CONTAINERS_TO_CREATE+=("low:latest")
CONTAINERS_TO_CREATE+=("cpu_stress:latest")
CONTAINERS_TO_CREATE+=("ram:latest")

# Llenar el resto (6) con imágenes aleatorias
for i in {1..6}; do
    random_index=$((RANDOM % ${#IMAGES[@]}))
    CONTAINERS_TO_CREATE+=("${IMAGES[$random_index]}")
done

# Crear los contenedores
for i in "${!CONTAINERS_TO_CREATE[@]}"; do
    tipo_contenedor="${CONTAINERS_TO_CREATE[$i]}"

    # Generar nombre aleatorio único
    random_name=$(cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 6 | head -n 1)
    current_time=$(date +"%H%M%S")
    container_name="${tipo_contenedor//:/-}-$((i+1))-$random_name-$current_time"

    # Ejecutar el contenedor
    docker run -d --name "$container_name" "$tipo_contenedor"

    # Mostrar información
    echo "========================================"
    echo "Contenedor NO.$((i+1))"
    echo "Nombre: $container_name"
    echo "Tipo: $tipo_contenedor"
    echo "Estado: Creado exitosamente"
    echo "========================================"
    echo
done
