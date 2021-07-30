FROM golang:1.16 as audit-build
WORKDIR /antrea
COPY . /antrea
RUN CGO_ENABLED=0 go build -o audit-webhook ./cmd/webhook

FROM scratch
LABEL maintainer="Antrea <projectantrea-dev@googlegroups.com>"
LABEL description="The docker image for the auditing system"
ENV USER root
COPY --from=audit-build /antrea/audit-webhook /
ENTRYPOINT ["/audit-webhook"]

