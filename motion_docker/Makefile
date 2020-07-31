IMAGE_NAME := lpr/motion:latest


build: ## Build a new image and tag it as the latest
	docker build -t $(IMAGE_NAME) .

run:
	docker run --rm --net=host -v `pwd`/conf:/etc/motion -v `pwd`/images:/var/lib/motion --name motion $(IMAGE_NAME)

shell: ## Run an interactive shell in a new container and delete it afterwards
	docker run --rm --net=host -v `pwd`/conf:/etc/motion -v `pwd`/images:/var/lib/motion -i -t --entrypoint /bin/bash $(IMAGE_NAME)


.PHONY: build run shell
