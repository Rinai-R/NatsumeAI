API_DIR := ./app/api
API_MODULES := user checkout order product payment coupon

.PHONY: api
.FORCE:

api: $(addprefix api-,$(API_MODULES))

api-%: .FORCE
	@echo "[api] regenerating $*"
	@cd $(API_DIR)/$* && goctl api go --dir . --api $*.api


RPC_DIR := ./app/services
RPC_MODULES := user checkout auth order product payment coupon inventory

.PHONY: rpc
.FORCE:

rpc: $(addprefix rpc-,$(RPC_MODULES))

rpc-%: .FORCE
	@echo "[rpc] regenerating $*"
	@cd $(RPC_DIR)/$* && goctl rpc protoc $*.proto --go_out=. --go-grpc_out=. --zrpc_out=. --client=true


MODEL_DIR := ./manifest/sql
MODEL_MODULES := user checkout order product payment coupon inventory

.PHONY: model
.FORCE:

model: $(addprefix model-,$(MODEL_MODULES))

model-%: .FORCE
	@echo "[model] regenerating $*"
	goctl model mysql ddl --dir ./app/dal/$* --cache true --src $(MODEL_DIR)/$*/$*.sql


.PHONY: api-format
.FORCE:

api-format: $(addprefix api-format-,$(API_MODULES))

api-format-%: .FORCE
	@cd $(API_DIR)/$* && goctl api format --dir .

.PHONY: dependency

dependency:
	docker compose -f ./manifest/deploy/dependency.yaml up -d

.PHONY: app

app:
	docker compose -f ./manifest/deploy/app.yaml up -d --build

.PHONY: devops

devops:
	docker compose -f ./manifest/deploy/devops.yaml up -d

.PHONY: dependency-down

dependency-down:
	docker compose -f ./manifest/deploy/dependency.yaml down

.PHONY: app-down

app-down:
	docker compose -f ./manifest/deploy/app.yaml down
	
.PHONY: devops-down

devops-down:
	docker compose -f ./manifest/deploy/devops.yaml down
