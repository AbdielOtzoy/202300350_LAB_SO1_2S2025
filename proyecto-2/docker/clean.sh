#!/bin/bash

# Buscar contenedores detenidos cuyo comando empiece con "python"
containers=$(docker ps -a --filter "status=exited" --format "table {{.ID}}\t{{.Image}}\t{{.Command}}\t{{.CreatedAt}}\t{{.Status}}\t{{.Names}}" | grep "python" | awk '{print $1}')

if [ -z "$containers" ]; then
    echo "No hay contenedores detenidos con comando 'python ...' para eliminar."
    exit 0
fi

echo "Eliminando los siguientes contenedores:"
echo "$containers"

# Eliminar cada contenedor encontrado
for id in $containers; do
    echo "Eliminando contenedor $id..."
    docker rm "$id"
done
