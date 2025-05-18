## DBの初期化


```bash
docker compose down
docker volume ls 
docker volume rm sokoni_sokoni_pgadata
docker compose up -d

export DATABASE_URL="postgres://sokoni:sokoni@localhost:5432/sokoni?sslmode=disable"
migrate -path db/migrations -database "$DATABASE_URL" up
```