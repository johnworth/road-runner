FROM jeanblanchard/alpine-glibc
ARG git_commit=unknown
ARG buildenv_git_commit=unknown
ARG version=unknown
COPY road-runner /bin/road-runner
CMD ["road-runner" "--help"]
