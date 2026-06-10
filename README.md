#  Async Job Processing System (API + Message queue + Workers)

This project demonstrates a **non-blocking, distributed job processing system** where heavy workloads are offloaded to background workers—keeping APIs fast and responsive under load.

---


- **Frontend (Node.js/Express)**  
  Renders dashboard + proxies API requests  

- **API (Go)**  
  Validates requests, creates jobs, pushes to Redis  

- **Worker (Python)**  
  Consumes jobs, processes asynchronously, updates status  

- **Redis**  
  Acts as queue + job state store  

---

##  Key Features

- Non-blocking API design (async job queue)
- Fully containerized (Docker Compose)
- Real-time job status tracking
- Decoupled services (API, worker, frontend)

##  Run Locally

```bash
### How to run locally
1. Clone the repository: git clone https://github.com/nehecodes/asyncflow.git && cd asyncflow
2. Create your .env file and copy .env.example into it: cp .env.example .env
3. Build and start containers: docker compose up --build
4. Check the containers' status: docker compose ps  

5. Open http://localhost:3000 in your browser
6. To stop running containers: docker compose down