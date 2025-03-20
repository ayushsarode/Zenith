.PHONY: help

help: 
	@echo "Available commands:"
	@echo " make build			- Build all Docker images"
	@echo " make up				- Start all services in detached mode"
	@echo " make down 			- Stop all services"
	@echo " make logs 			- Follow logs from all services"
	@echo " make restart 		- Restart all services"
	@echo " make client 		- Run client container interactively"
	@echo " make server 		- Run server container interactively" 
	@echo " make clean 			- Remove all containers, volumes, and images"


.PHONY: build 
build: 
	docker compose build 

.PHONY: up 
	docker compose up -d

.PHONY: down 
down: 
	docker compose down

.PHONY: logs
logs:
	docker compose logs -f 

.PHONY: restart
restart: 
	docker compose restart

.PHONY: client
client: 
	docker compose run --rm client

.PHONY: server
server: 
	docker compose run --rm server

.PHONY: clean
clean: 
	docker compose fown -v --rmi all


