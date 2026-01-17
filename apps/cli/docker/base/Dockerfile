ARG LANGUAGE

FROM python:3.11-slim AS python
FROM node:20-slim AS node
FROM eclipse-temurin:21-jre AS java
FROM mcr.microsoft.com/dotnet/runtime:8.0 AS csharp
FROM golang:1.24-bookworm AS go

ARG LANGUAGE
FROM ${LANGUAGE} AS runtime

ARG TARGETARCH
WORKDIR /app
COPY dist/shuttl-linux-${TARGETARCH} /usr/local/bin/shuttl
RUN chmod +x /usr/local/bin/shuttl

ENTRYPOINT ["shuttl"]

