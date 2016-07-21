FROM golang:1.6

ARG git_commit=unknown
LABEL org.cyverse.git-ref="$git_commit"

COPY . /go/src/github.com/cyverse-de/road-runner
RUN go install github.com/cyverse-de/road-runner

ENTRYPOINT ["road-runner"]
CMD ["--help"]
