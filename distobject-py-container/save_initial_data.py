import redis
import ulid
import json
import os

def main():
    r = redis.Redis(host=os.getenv('REDIS_HOST', 'redis'), port=6379, decode_responses=True)
    
    ulid_value = ulid.new().str
    key = f"user:{ulid_value}"
    data = {
        "name": "Alice",
        "email": "alice@example.com",
        "created_at": "0",
        "updated_at": "0"
    }

    r.hset(key, mapping=data)
    r.publish("user-updates", json.dumps({"id": key, "changes": data}))

    # Save key to a shared file
    os.makedirs("/shared", exist_ok=True)
    with open("/shared/key.txt", "w") as f:
        f.write(key)

    print(f"Saved key to /shared/key.txt: {key}")

if __name__ == "__main__":
    main()
