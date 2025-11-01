BIN=.bin/templr

BUILDER ?= multiarch
VERSION ?= dev
IMAGE ?= kanopi/templr
PLATFORMS ?= linux/amd64,linux/arm64
BUILDER ?= multiarch
OS_LIST := linux darwin windows
# Per-OS architecture lists (skip windows/arm64)
ARCH_linux   := amd64 arm64
ARCH_darwin  := amd64 arm64
ARCH_windows := amd64

.PHONY: builder
builder:
	@docker buildx inspect $(BUILDER) >/dev/null 2>&1 || \
	docker buildx create --name $(BUILDER) --driver docker-container --use --bootstrap
	@docker run --privileged --rm tonistiigi/binfmt --install arm64,amd64

.PHONY: build test e2e golden clean

build:
	go build -o $(BIN) .

test: build
	go test ./e2e -v || true

e2e: build
	chmod +x tests/run_examples.sh
	tests/run_examples.sh

golden: build
	UPDATE_GOLDEN=1 tests/run_examples.sh

clean:
	rm -rf .bin .out

# Public phony targets we will generate:
#  - build-<os> / build_<os>         (aggregate for that OS)
#  - build-<os>-<arch> / build_<os>_<arch>
.PHONY: $(foreach os,$(OS_LIST),build-$(os) build_$(os))
.PHONY: $(foreach os,$(OS_LIST),$(foreach a,$(ARCH_$(os)),build-$(os)-$(a) build_$(os)_$(a)))

# Rule generator for a single OS/ARCH pair
define BUILD_OS_ARCH_RULE
.PHONY: build-$(1)-$(2) build_$(1)_$(2)
build-$(1)-$(2) build_$(1)_$(2):
	@mkdir -p .bin
	GOOS=$(1) GOARCH=$(2) go build -ldflags "-X main.Version=$(VERSION)" -o .bin/templr-$(1)-$(2)
endef

# Rule generator for aggregate per-OS target (depends on valid arches only)
define BUILD_OS_RULE
.PHONY: build-$(1) build_$(1)
build-$(1) build_$(1): $(foreach a,$(ARCH_$(1)),build-$(1)-$(a))
endef

# Emit per-arch rules for all OS/ARCH pairs
$(foreach os,$(OS_LIST),$(foreach a,$(ARCH_$(os)),$(eval $(call BUILD_OS_ARCH_RULE,$(os),$(a)))))

# Emit aggregate per-OS rules
$(foreach os,$(OS_LIST),$(eval $(call BUILD_OS_RULE,$(os))))

.PHONY: docker dockerbuild

docker: build-linux
	docker build --tag $(IMAGE):latest .

dockerx: docker builder
	docker buildx build --platform $(PLATFORMS) --tag $(IMAGE):latest .
