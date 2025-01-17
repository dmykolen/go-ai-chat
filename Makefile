PROJECTNAME=$(shell basename "$(PWD)")
DIR := $(abspath .)

TAGLAST=$(shell git tag -l | sort -V | tail -1)
P1=$(shell echo ${TAGLAST} | cut -d "." -f-2)
P2=$(shell echo ${TAGLAST} | cut -d "." -f3)
TAGNEW=$(shell echo ${P1}.$(shell expr ${P2} + 1))
GIT_MSG ?= ''

BUILD_FLAGS := '-ldflags "-s -w"'

print:
	@echo PROJECTNAME: ${PROJECTNAME}
	@echo Build flags: ${BUILD_FLAGS}
	@echo DIR: ${DIR}
	@echo GIT_MSG: ${GIT_MSG}
	@echo FIRST:
	@echo Makefile TaskName: $@


help: ## This help dialog.
	@grep -F -h "##" $(MAKEFILE_LIST) | grep -F -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

fe: ## Build frontend assets
	cd web && npm run build

stop-dev: ## Stop the development server if it's running
	pgrep -i go-ai && pkill -e -i go-ai || echo "Not started yet!"

dev-nohup: stop-dev ## Run the development server in the background
	nohup go run . -l_color -dev -p 8080 -l_lvl debug -l_of -l_file ~/log/go-ai.log > ~/log/go-ai-other.log 2>&1 &

run-in-docker: ## Run the application in a Docker container
	nohup ./go-ai -l_color -dev -p 8080 -l_lvl debug -l_of -l_file ~/log/go-ai.log > ~/log/go-ai-other.log 2>&1 &

dev-local: ## Run the application locally in development mode
	go run . -l_color -dev -l_lvl debug -l_oc -p 7558 -lt layouts/main.dev

dev: ## Run the application locally
	go run . -l_color -l_lvl debug -l_oc -p 7558

dev2:
	env POSTGRES_HOST=ai.dev.ict POSTGRES_PORT=5444 go run . -l_color -l_lvl trace -l_oc -p 7558 -d -dev -app-ssl-on -prxon

dev3:
	POSTGRES_HOST="ai.dev.ict" POSTGRES_PORT=5444 go run . -l_color -l_lvl trace -l_oc -p 7558 -d -dev -aidboff -force-init-roles -wv_p 8083

dev-fe: fe dev ## Build frontend assets and run the application locally

##---------
gitinf: ## pull last changes from GitLab master branch and print info about tags
	git fetch origin && git pull origin master
	@echo Old tag: ${P1} ${P2}
	@echo New tag: ${TAGNEW}
	@echo Git commit message: ${GIT_MSG}

gitnew: clean gitinf ##+ git add, commit, tag, push
ifeq ($(strip $(GIT_MSG)),)
    $(error GIT_MSG is empty)
endif
	git add . && git commit -m "${GIT_MSG}" \
	&& git tag ${TAGNEW} && git push origin ${TAGNEW} \
	&& git push -u origin master \
	&& echo "GIT success" || echo "GIT failed - $?"
##---------

gobd: ## GO build app binary 'go-ai'
	go build ${BUILD_FLAGS} -o ${PROJECTNAME} main.go
got: ## GO check and prepare dependencies
	go mod tidy -v

.PHONY: clean
clean: ## GO clean previously builded binary and temp files + remove *.log files
	go clean
	rm *.log 2>/dev/null || true

dbi: ## Generate docker image
	docker build -t $(PROJECTNAME) .

dsh: ## Run interactive shell in the container
	docker exec -it $(PROJECTNAME) /bin/bash
