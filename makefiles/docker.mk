# Docker Configuration

.PHONY: up down create backup restore

# Docker Compose Management
up:
	@docker compose up -d

down:
	@docker compose down

# Docker Volume Management
create:
	@echo "Running create volume script..."
	@./scripts/manage_volume.sh create

backup:
	@echo "Running backup volume script..."
	@./scripts/manage_volume.sh backup

restore:
	@echo "Running restore volume script..."
	@./scripts/manage_volume.sh restore
