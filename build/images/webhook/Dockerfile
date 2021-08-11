FROM golang:1.16 as audit-build
WORKDIR /antrea
COPY . /antrea
RUN CGO_ENABLED=0 go build -o audit-webhook ./cmd/webhook
RUN CGO_ENABLED=0 go build -o auditctl ./cmd/cli

# TODO: Revert to distroless when auditctl out-of-cluster support is added
FROM ubuntu:20.04 
LABEL maintainer="Antrea <projectantrea-dev@googlegroups.com>"
LABEL description="The docker image for the auditing system"
ENV USER root
COPY --from=audit-build /antrea/audit-webhook /
COPY --from=audit-build /antrea/auditctl /usr/local/bin
ENTRYPOINT ["/audit-webhook"]

