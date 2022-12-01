#!/usr/bin/env bash

set -e

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
cd $SCRIPT_DIR
go mod download

cd $SCRIPT_DIR/cmd/golden
gitversion=$(git describe)
CGO_ENABLED=0
go build -trimpath -ldflags "-X golden/pkg/git.Version=${gitversion}"

cd $SCRIPT_DIR
mkdir -p build
mv cmd/golden/golden build/
echo "Built $SCRIPT_DIR/build/golden" 1>&2
