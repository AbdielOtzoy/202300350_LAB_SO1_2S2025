from locust import HttpUser, TaskSet, task, between
import random
import json

class WeatherTasks(TaskSet):
    
    @task(1)
    def send_weather_data(self):
        # Municipalities (enum values from protobuf)
        # 1: mixco, 2: guatemala, 3: amatitlan, 4: chinautla
        municipalities = [1, 2, 3, 4]
        
        # Weather types (enum values from protobuf)
        # 1: sunny, 2: cloudy, 3: rainy, 4: foggy
        weather_types = [1, 2, 3, 4]
    
        # Weather data matching Rust API structure
        weather_data = {
            "municipality": random.choice(municipalities),  # 1-4
            "temperature": random.randint(15, 35),  # Temperature in Celsius (15-35°C)
            "humidity": random.randint(40, 95),  # Humidity percentage (40-95%)
            "weather": random.choice(weather_types)  # 1-4
        }
        
        # Send JSON as POST to the /clima route
        headers = {'Content-Type': 'application/json'}
        response = self.client.post("/clima", json=weather_data, headers=headers)
        
        # Log the response for debugging
        if response.status_code == 200:
            print(f"✅ Success: {weather_data}")
        else:
            print(f"❌ Failed: {response.status_code} - {response.text}")

class WeatherUser(HttpUser):
    tasks = [WeatherTasks]
    wait_time = between(1, 3)  # Wait time between tasks (1 to 3 seconds)
    
    # Set the host (can be overridden via command line)
    host = "http://34.36.85.14"