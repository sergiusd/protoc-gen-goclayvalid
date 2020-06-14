PATH_API=example
PATH_OUTPUT=output
SERVICES=$(shell ls -1 $(PATH_API) | grep \.proto | sed s/\.proto//)

GO_EXEC=go
LOCAL_BIN:=./bin
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m▶\033[0m")

$(LOCAL_BIN):
	@mkdir -p $@
$(LOCAL_BIN)/%: ; $(info $(M) building $(REPOSITORY)…)
	$Q tmp=$$(mktemp -d); \
		(GOPATH=$$tmp GO111MODULE=off go get $(REPOSITORY) && cp $$tmp/bin/* $(LOCAL_BIN)/.) || ret=$$?; \
		rm -rf $$tmp ; exit $$ret

.PHONY: bin-deps
bin-deps: ; $(info $(M) install bin depends…)
	$(info #Installing binary dependencies...)

.PHONY: deps
deps: ; $(info $(M) install depends…)
	$(info #Install dependencies...)
	$(GO_EXEC) mod tidy

.PHONY: generate
generate: bin-deps build ; $(info $(M) go generate…)
	$(Q) for srv in $(SERVICES); do \
	    echo "Process $(PATH_API)/$$srv.proto" && \
		mkdir -p ./$(PATH_OUTPUT) && \
		protoc \
			--plugin=protoc-gen-goclayvalid=$(LOCAL_BIN)/protoc-gen-goclayvalid \
			-I./:./example/vendor.pb \
			--goclayvalid_out=original_field_name,pretty:./$(PATH_OUTPUT) \
			./$(PATH_API)/$$srv.proto; \
	done

.PHONY: build
build: ; $(info $(M) run build…)
	$(info #Building app...)
	$(GO_EXEC) build -o $(LOCAL_BIN)/protoc-gen-goclayvalid ./main.go

.PHONY: test
test: ; $(info $(M) run test…)
	./test/prepare.sh
	$(GO_EXEC) test -v .

.PHONY: bin-deps deps generate build test
