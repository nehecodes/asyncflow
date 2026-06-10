import redis
import time
import os
import signal
import sys
from dotenv import load_dotenv

load_dotenv()

redis_host = os.getenv("REDIS_HOST")
redis_port_str = os.getenv("REDIS_PORT")
if not redis_host or not redis_port_str:
    raise ValueError("REDIS_HOST and REDIS_PORT environment variables must be set")
try:
    redis_port = int(redis_port_str)
except ValueError:
    raise ValueError("REDIS_PORT must be a valid integer")
try:
    with open("/run/secrets/redis_password") as handler:
        password = handler.read().strip()
    redis_conn = redis.Redis(
        host=redis_host,
        port=redis_port,
        password=password,
        decode_responses=True,
    )
    redis_conn.ping()
    print("Connected to Redis")
except Exception as e:
    print(f"Redis connection failed: {e}")
    sys.exit(1)


shutdown = False


def handle_signal(sig, frame):
    global shutdown
    print("Shutting down gracefully...")
    shutdown = True


signal.signal(signal.SIGINT, handle_signal)
signal.signal(signal.SIGTERM, handle_signal)


def process_job(jobID):
    try:
        redis_conn.hset(f"job:{jobID}", "status", "processing")
        print(f"Processing job {jobID}")
        time.sleep(2)  # simulate work
        redis_conn.hset(f"job:{jobID}", "status", "completed")
        print(f"Done: {jobID}")
    except Exception as e:
        print(f"Error processing {jobID}: {e}")
        redis_conn.hset(f"job:{jobID}", "status", "failed")


while not shutdown:
    job = redis_conn.brpop("jobs", timeout=5)

    if not job:
        continue

    _, job_id = job
    process_job(job_id)
print("Worker stopped")
sys.exit(0)
