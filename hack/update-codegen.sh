#!/usr/bin/env bash

# Copyright 2018, EnMasse authors.
# License: Apache License 2.0 (see the file LICENSE or http://apache.org/licenses/LICENSE-2.0.html).

set -e

SCRIPTPATH="$(cd "$(dirname "$0")" && pwd -P)"
GENERATOR_BASE=${SCRIPTPATH}/../vendor/k8s.io/code-generator

"$GENERATOR_BASE/generate-groups.sh" "deepcopy,client,informer,lister" \
    github.com/ctron/hono-qdrouter-proxy/pkg/client \
    github.com/ctron/hono-qdrouter-proxy/pkg/apis \
    iotproject:v1alpha1 \
    --go-header-file "${SCRIPTPATH}/header.txt"

