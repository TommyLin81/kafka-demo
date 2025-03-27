help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

#######################
# Docker Compose
#######################
docker-up:  ## Start the services with docker-compose
	docker-compose up --build -d

docker-down:  ## Stop and remove docker-compose services
	docker-compose down

#######################
# Linting
#######################
lint:  ## Run golangci-lint with auto-fix
	golangci-lint run --fix ./...

#######################
# Helm
#######################
docker-build:  ## Build the docker images
	docker build -t chat-server:latest -f ./docker/chat-server/dockerfile .
	docker build -t chat-web:latest -f ./docker/chat-web/dockerfile .
	docker build -t content-moderator:latest -f ./docker/content-moderator/dockerfile .

helm-deps:  ## Build the Helm dependencies
	helm dependency update ./helm/kafka-demo

helm-setup: docker-build helm-deps ## Setup the Helm environment

minikube-load-images:  ## Load the docker images to minikube
	minikube image load chat-server:latest
	minikube image load chat-web:latest
	minikube image load content-moderator:latest

helm-install:  ## Install the Helm chart
	helm install kafka-demo --namespace kafka-demo --create-namespace -f ./helm/kafka-demo/values.yaml ./helm/kafka-demo 

helm-upgrade:  ## Upgrade the Helm chart
	helm upgrade kafka-demo --namespace kafka-demo -f ./helm/kafka-demo/values.yaml ./helm/kafka-demo 

helm-uninstall:  ## Uninstall the Helm chart
	helm uninstall kafka-demo --namespace kafka-demo

port-forward:  ## Port forward the services
	@bash -c '\
		kubectl port-forward svc/kafka-demo-kafka-ui 8080:80 --namespace kafka-demo & \
		pid1=$$!; \
		kubectl port-forward svc/chat-server 12345:12345 --namespace kafka-demo & \
		pid2=$$!; \
		kubectl port-forward svc/chat-web 8081:8081 --namespace kafka-demo & \
		pid3=$$!; \
		trap "echo '\''Stopping port-forward...'\''; kill $$pid1 $$pid2 $$pid3" INT; \
		wait $$pid1 $$pid2 $$pid3; \
	'
