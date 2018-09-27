# Simulate the environment used by Travis CI so that we can run local tests to
# find and resolve issues that are consistent with the Travis CI environment.
# This is helpful because Travis CI often finds issues that our own local tests
# do not.

# go vet ./...
# golint -set_exit_status `go list ./... | grep -Ev "(vendor)"`

go build ./...
GOMAXPROCS=1 CI=true ginkgo --cover core/collection/buffer \
                                    core/collection/stack  \
                                    core/vm/mul            \
                                    core/vm/open           \
                                    core/vm/rng            \
                                    core/vm
covermerge core/collection/buffer/buffer.coverprofile \
           core/collection/stack/stack.coverprofile   \
           core/vm/mul/mul.coverprofile               \
           core/vm/open/open.coverprofile             \
           core/vm/rng/rng.coverprofile               \
           core/vm/vm.coverprofile                    \
           > oro.coverprofile