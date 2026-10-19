GO_VERSION := $(shell awk '$$1 == "go" {print $$2; exit}' go.mod)
ALPINE_VERSION := 3.23
# Preserve the source server's PostgreSQL 15.19 image during migration.
POSTGRES_VERSION := 15-alpine@sha256:f7d23353e1b15400d22ebe31189f4d314b87a4c129cc400c8c2d8d4ca127bf81
VERSION_VARS := GO_VERSION ALPINE_VERSION POSTGRES_VERSION
export $(VERSION_VARS)
