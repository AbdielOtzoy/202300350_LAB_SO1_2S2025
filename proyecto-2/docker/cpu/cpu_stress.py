import time
import math

while True:
    for _ in range(2000):  # menos iteraciones que antes
        _ = [math.sqrt(i) for i in range(1000)]
    time.sleep(0.5)  # aumenta el tiempo de descanso
