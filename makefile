help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

#######################
# Docker Compose
#######################
docker-up:  ## Start the services with docker-compose
	docker-compose up --build -d

docker-down:  ## Stop and remove docker-compose services
	docker-compose down
