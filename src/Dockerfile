FROM docker.io/golang:1.24-alpine AS base

# Install run-time and compile-time system dependencies
USER root
RUN apk add --no-cache \
    make \
    nodejs \
    npm

# Create and switch to non-root user
RUN addgroup -g 1000 runner \
    && adduser -G runner -u 1000 -D runner
USER runner

# Install run-time and compile-time local dependencies
RUN go install github.com/a-h/templ/cmd/templ@latest

# Create required folders and change to source directory
RUN mkdir /home/runner/src \
    /home/runner/data
WORKDIR /home/runner/src

# Install run-time and compile-time project dependencies
COPY --chown=runner:runner package.json package-lock.json ./
RUN npm install --omit=dev
COPY --chown=runner:runner go.mod go.sum ./
RUN go mod download

#
# Production
#

FROM base AS builder

# Copy source and build
USER runner
WORKDIR /home/runner/src
COPY --chown=runner:runner . .
RUN make ./build/server

FROM docker.io/alpine:latest AS production

# Create and switch to non-root user
RUN addgroup -g 1000 runner \
    && adduser -G runner -u 1000 -D runner
USER runner

# Create required folders and change to app directory
RUN mkdir /home/runner/app \
    /home/runner/app/pdf \
    /home/runner/data
WORKDIR /home/runner/app

# Deploy the application binary into a lean image
COPY --from=builder --chown=runner:runner /home/runner/src/build/server ./

# Expose required ports
EXPOSE 8080

# Define entrypoint
ENTRYPOINT ["/home/runner/app/server"]

#
# Development
#

FROM base AS development

# Install dev-time system dependencies
USER root
RUN apk add --no-cache \
    inotify-tools \
    bash

# Change to non-root user
USER runner

# Install dev-time local dependencies
RUN go install github.com/air-verse/air@latest

# Change to source directory
WORKDIR /home/runner/src

# Install dev-time project dependencies
RUN npm install

# Move temporally project dependencies and clean source directory
RUN mv node_modules ../ \
    && find /home/runner/src -mindepth 1 -delete

# Expose required ports
EXPOSE 8080
EXPOSE 8010

# Define entrypoint
COPY --chown=runner:runner scripts/entrypoint-dev.sh /home/runner/
ENTRYPOINT ["/home/runner/entrypoint-dev.sh"]

