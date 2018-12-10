.PHONY: all test clean images
.DEFAULT_GOAL := all

# stolen and from https://github.com/cortexproject/cortex and modified for our needs

# Boiler plate for bulding Docker containers.
# All this must go at top of file I'm afraid.
IMAGE_PREFIX ?= grafana/devtools-
GIT_REVISION := $(shell git rev-parse HEAD)
UPTODATE := .uptodate

# Building Docker images is now automated. The convention is every directory
# with a Dockerfile in it builds an image calls quay.io/weaveworks/<dirname>.
# Dependencies (i.e. things that go in the image) still need to be explicitly
# declared.
%/$(UPTODATE): %/Dockerfile
	$(SUDO) docker build -t $(IMAGE_PREFIX)$(shell basename $(@D)) $(@D)/
	$(SUDO) docker tag $(IMAGE_PREFIX)$(shell basename $(@D)) $(IMAGE_PREFIX)$(shell basename $(@D)):$(GIT_REVISION)
	touch $@

# We don't want find to scan inside a bunch of directories, to accelerate the
# 'make: Entering directory '/go/src/github.com/weaveworks/cortex' phase.
DONT_FIND := -name tools -prune -o -name vendor -prune -o -name .git -prune -o -name .cache -prune -o -name .pkg -prune -o

# Get a list of directories containing Dockerfiles
DOCKERFILES := $(shell find . $(DONT_FIND) -type f -name 'Dockerfile' -print)
UPTODATE_FILES := $(patsubst %/Dockerfile,%/$(UPTODATE),$(DOCKERFILES))
DOCKER_IMAGE_DIRS := $(patsubst %/Dockerfile,%,$(DOCKERFILES))
IMAGE_NAMES := $(foreach dir,$(DOCKER_IMAGE_DIRS),$(patsubst %,$(IMAGE_PREFIX)%,$(shell basename $(dir))))
images:
	$(info $(IMAGE_NAMES))
	@echo > /dev/null

# Building binaries is now automated.  The convention is to build a binary
# for every directory with main.go in it, in the ./cmd directory.
MAIN_GO := $(shell find . $(DONT_FIND) -type f -name 'main.go' -print)
EXES := $(foreach exe, $(patsubst ./cmd/%/main.go, %, $(MAIN_GO)), ./cmd/$(exe)/$(exe))
GO_FILES := $(shell find . $(DONT_FIND) -name cmd -prune -o -type f -name '*.go' -print)
define dep_exe
$(dir $(1))$(UPTODATE): $(1)
endef
$(foreach exe, $(EXES), $(eval $(call dep_exe, $(exe))))

# Manually declared dependancies And what goes into each exe
all: $(UPTODATE_FILES)

# All the boiler plate for building golang follows:
SUDO := $(shell docker info >/dev/null 2>&1 || echo "sudo -E")

$(EXES):
	go build $(GO_FLAGS) -o $@ ./$(@D)

clean:
	$(SUDO) docker rmi $(IMAGE_NAMES) >/dev/null 2>&1 || true
	rm -rf $(UPTODATE_FILES) $(EXES) .cache
	go clean ./...

