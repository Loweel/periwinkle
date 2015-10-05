# Set NET='' on the command line to not try to update things
NET ?= NET

# packages is the list of packages that we actually wrote and need built
packages = listener

# set deps to be a list of import strings of external packages we need to import
deps += bitbucket.org/ww/goautoneg
deps += github.com/go-sql-driver/mysql
deps += golang.org/x/crypto/bcrypt


default: all
.PHONY: default

# What directory is the Makefile in?
topdir := $(realpath $(dir $(lastword $(MAKEFILE_LIST))))

# Configuration of the C compiler for C code called from Go
CFLAGS = -std=c99 -Wall -Wextra -Werror -pedantic
CGO_CFLAGS = $(CFLAGS) -Wno-unused-parameter
CGO_ENABLED = 1
cgo_variables = CGO_ENABLED CGO_CFLAGS CGO_CPPFLAGS CGO_CXXFLAGS CGO_LDFLAGS CC CXX
export $(cgo_variables)

# A list of go source files; if any of these change, we need to rebuild
goext = c s S cc cpp cxx h hh hpp hxx
gosrc = $(shell find -L src -name '.*' -prune -o \( -type f \( $(foreach e,$(goext), -name '*.$e' ) \) -o -type d \) -print)

# Iterate over external dependencies, and create a rule to download it
$(foreach d,$(deps),$(eval src/$d: $(NET); GOPATH='$(topdir)' go get -d -u $d))

all: bin
.PHONY: all

# The rule to build the Go code.  The first line nukes the built files
# if there is a discrepancy between Make and Go's internal
# dependency tracker.
bin pkg: $(gosrc) $(addprefix src/,$(deps)) $(addprefix .var.,$(cgo_variables) packages)
	@true $(foreach f,$(filter-out .var.%,$^), && test $@ -nt $f ) || rm -rf -- bin pkg
	GOPATH='$(topdir)' go install $(packages)

# Rule to nuke everything
clean:
	rm -rf -- pkg bin src/*.*/ .var.*
.PHONY: clean

# Now, this is magic.  It stores the values of environment variables,
# so that if you change them in a way that would cause something to be
# rebuilt, then Make knows.
.var.%: FORCE
	@printf '%s' '$($*)' > .tmp$@ && { cmp -s .tmp$@ $@ && rm -f -- .tmp$@ || mv -Tf .tmp$@ $@; } || { rm -f -- .tmp$@; false; }

# Boilerplate
.SECONDARY:
.DELETE_ON_ERROR:
.PHONY: FORCE NET
