import time

MAX_MB = 300   # tope de memoria en MB
CHUNK_MB = 5   # cuánto crece por ciclo
CHUNK_SIZE = 1024 * 1024 * CHUNK_MB  # bytes

data = []

while True:
    # si aún no alcanzamos el tope, agregamos un bloque
    if len(data) * CHUNK_MB < MAX_MB:
        chunk = "X" * CHUNK_SIZE  # string de CHUNK_MB MB
        data.append(chunk)
        print(f"RAM usada: {len(data) * CHUNK_MB} MB (aprox)")
    else:
        print(f"Tope alcanzado ({MAX_MB} MB) ✅")
    
    time.sleep(0.5)
    