FROM golang:1.7

COPY . /go/src/github.com/cyverse-de/road-runner
RUN go install github.com/cyverse-de/road-runner

ENTRYPOINT ["road-runner"]
CMD ["--help"]

ARG git_commit=unknown
ARG version="2.9.0"
ARG descriptive_version=unknown

LABEL org.cyverse.git-ref="$git_commit"
LABEL org.cyverse.version="$version"
LABEL org.cyverse.descriptive-version="$descriptive_version"
