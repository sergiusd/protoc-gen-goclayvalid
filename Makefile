PROJECT_STRUCTURE_API=api
PROJECT_STRUCTURE_CLIENTS=output
PROJECT_STRUCTURE_IMPLEMENTATIONS=internal/app/api
FROM_SERVICES_TO_ROOT_REL=$(shell echo $(PROJECT_STRUCTURE_CLIENTS) | perl -F/ -lane 'print "../"x scalar(@F)')
IMPLEMENTATION_TYPE_NAME="Implementation"
SERVICES=$(shell ls -1 $(PROJECT_STRUCTURE_API) | grep \.proto | sed s/\.proto//)

GO_EXEC=go
LOCAL_BIN:=$(CURDIR)/bin
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m▶\033[0m")
PKGS=$(shell go list -f '{{.Dir}}' ./... | grep -v /vendor/ | grep -v /snami/pkg/api/)

PKGMAP:=Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,$\
        Mgoogle/protobuf/api.proto=github.com/gogo/protobuf/types,$\
        Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/types,$\
        Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,$\
        Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,$\
        Mgoogle/protobuf/field_mask.proto=github.com/gogo/protobuf/types,$\
        Mgoogle/protobuf/source_context.proto=github.com/gogo/protobuf/types,$\
        Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,$\
        Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,$\
        Mgoogle/protobuf/type.proto=github.com/gogo/protobuf/types,$\
        Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types

$(LOCAL_BIN):
	@mkdir -p $@
$(LOCAL_BIN)/%: ; $(info $(M) building $(REPOSITORY)…)
	$Q tmp=$$(mktemp -d); \
		(GOPATH=$$tmp GO111MODULE=off go get $(REPOSITORY) && cp $$tmp/bin/* $(LOCAL_BIN)/.) || ret=$$?; \
		rm -rf $$tmp ; exit $$ret

.PHONY: bin-deps
bin-deps: ; $(info $(M) install bin depends…)
	$(info #Installing binary dependencies...)
	GOBIN=$(LOCAL_BIN) $(GO_EXEC) install github.com/gogo/protobuf/protoc-gen-gofast
	GOBIN=$(LOCAL_BIN) $(GO_EXEC) install github.com/utrack/clay/v2/cmd/protoc-gen-goclay

.PHONY: deps
deps: ; $(info $(M) install depends…)
	$(info #Install dependencies...)
	$(GO_EXEC) mod tidy

.PHONY: generate
generate: bin-deps build ; $(info $(M) go generate…)
	$(Q) for srv in $(SERVICES); do \
	    echo "Generate $(CURDIR)/$(PROJECT_STRUCTURE_CLIENTS)/$$srv" && \
	    echo "Implementation $(FROM_SERVICES_TO_ROOT_REL)../../$(PROJECT_STRUCTURE_IMPLEMENTATIONS)/$$srv" && \
		mkdir -p $(CURDIR)/$(PROJECT_STRUCTURE_CLIENTS)/$$srv && \
		cd $(CURDIR)/$(PROJECT_STRUCTURE_CLIENTS)/$$srv && \
		protoc \
		    --plugin=protoc-gen-goclay=$(LOCAL_BIN)/protoc-gen-goclay \
			--plugin=protoc-gen-gofast=$(LOCAL_BIN)/protoc-gen-gofast \
			--plugin=protoc-gen-goclayvalid=$(LOCAL_BIN)/protoc-gen-goclayvalid \
			-I$(FROM_SERVICES_TO_ROOT_REL)../api/:$(CURDIR)/vendor.pb \
			--gofast_out=$(PKGMAP),plugins=grpc:. \
			--goclayvalid_out=. \
			--goclay_out=$(PKGMAP),impl=true,impl_service_sub_dir=false,impl_path=$(FROM_SERVICES_TO_ROOT_REL)../$(PROJECT_STRUCTURE_IMPLEMENTATIONS)/$$srv,impl_type_name_tmpl=$(IMPLEMENTATION_TYPE_NAME):. \
			$(FROM_SERVICES_TO_ROOT_REL)../$(PROJECT_STRUCTURE_API)/$$srv.proto; \
	done

.PHONY: build
build: ; $(info $(M) run build…)
	$(info #Building app...)
	$(GO_EXEC) build -o $(LOCAL_BIN)/protoc-gen-goclayvalid ./main.go
