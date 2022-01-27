# Copyright 2022 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Old-skool build tools.
# Simple makefile to build archeio quickly and reproducibly
#
# Common uses:
# - building: `make build`
# - cleaning up and starting over: `make clean`
#
################################################################################
# ========================== Capture Environment ===============================
# get the repo root and output path
REPO_ROOT:=${CURDIR}
OUT_DIR=$(REPO_ROOT)/bin
# record the source commit in the binary, overridable
COMMIT?=$(shell git rev-parse HEAD 2>/dev/null)
################################################################################
# ========================= Setup Go With Gimme ================================
# go version to use for build etc.
# setup correct go version with gimme
PATH:=$(shell . hack/tools/setup-go.sh && echo "$${PATH}")
# go1.9+ can autodetect GOROOT, but if some other tool sets it ...
GOROOT:=
# enable modules
GO111MODULE=on
# disable CGO by default for static binaries
CGO_ENABLED=0
export PATH GOROOT GO111MODULE CGO_ENABLED
# work around broken PATH export
SPACE:=$(subst ,, )
SHELL:=env PATH=$(subst $(SPACE),\$(SPACE),$(PATH)) $(SHELL)
################################################################################
# ============================== OPTIONS =======================================
# the output binary name, overridden when cross compiling
ARCHEIO_BINARY_NAME?=archeio
# build flags for the archeio binary
# - reproducible builds: -trimpath
# - smaller binaries: -w (trim debugger data, but not panics)
ARCHEIO_BUILD_FLAGS?=-trimpath -ldflags="-w"
################################################################################
# ================================= Building ===================================
# standard "make" target -> builds
all: build
# builds archeio, outputs to $(OUT_DIR)
archeio:
	go build -v -o "$(OUT_DIR)/$(ARCHEIO_BINARY_NAME)" $(ARCHEIO_BUILD_FLAGS) ./cmd/archeio
# alias for building archeio
build: archeio
################################################################################
# ================================= Testing ====================================
# unit tests (hermetic)
unit:
	MODE=unit hack/make-rules/test.sh
# integration tests
integration:
	MODE=integration hack/make-rules/test.sh
# all tests
test:
	hack/make-rules/test.sh
################################################################################
# ================================= Cleanup ====================================
# standard cleanup target
clean:
	rm -rf "$(OUT_DIR)/"
################################################################################
# ============================== Auto-Update ===================================
# update generated code, gofmt, etc.
update:
	hack/make-rules/update.sh
tidy:
	hack/make-rules/tidy.sh
# gofmt
gofmt:
	hack/make-rules/gofmt.sh
################################################################################
# ================================== Linting ===================================
# run linters, ensure generated code, etc.
verify:
	hack/make-rules/verify.sh
# code linters
lint:
	hack/make-rules/lint.sh
# shell linter
shellcheck:
	hack/make-rules/shellcheck.sh
#################################################################################
.PHONY: all archeio build unit integration clean update gofmt verify lint shellcheck