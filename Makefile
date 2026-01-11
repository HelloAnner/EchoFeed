.PHONY: start restart stop ps logs

COMPOSE ?= docker compose
IMAGE ?= echofeed:local

start:
	mkdir -p data bak
	$(COMPOSE) up -d --build --remove-orphans

restart:
	mkdir -p data bak
	$(COMPOSE) down --remove-orphans
	$(COMPOSE) up -d --build --remove-orphans

stop:
	$(COMPOSE) down --remove-orphans --rmi local
	docker image rm -f $(IMAGE) || true

ps:
	$(COMPOSE) ps

logs:
	$(COMPOSE) logs -f --tail=200
