TEST?=$$(go list ./... | grep -v 'vendor') 
PROVIDER_NAME=marketplace
BINARY=terraform-provider-${PROVIDER_NAME}
VERSION=1.0.0
ARCH?=darwin_arm64
VAR_FILE=manifest.json
TF_LOG=DEBUG
INSTALL_LOC=~/.terraform.d/plugins/lifeomic.com/tf/${PROVIDER_NAME}/${VERSION}/${ARCH}

.PHONY: default
default: install

.PHONY: build
build:
	go build -o ${BINARY}

.PHONY: clean
clean:
	rm -f .terraform.lock.hcl
	rm -rf .terraform
	rm -rf ${INSTALL_LOC}

.PHONY: install
install: build clean
	install -d ${INSTALL_LOC}
	install ${BINARY} ${INSTALL_LOC}/${BINARY}_v${VERSION}

.PHONY: init
init: install
	terraform init

.PHONY: plan
plan: init
	TF_LOG=${TF_LOG} terraform plan -var-file=${VAR_FILE}

.PHONY: apply
apply: init
	TF_LOG=${TF_LOG} terraform apply -var-file=${VAR_FILE}

.PHONY: test
test: 
	go test -i $(TEST) || exit 1                                                   
	echo $(TEST) | xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=4                    

.PHONY: testacc
testacc: 
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 120m   

.PHONY: lint
lint:
	staticcheck ./...
	go vet ./...
