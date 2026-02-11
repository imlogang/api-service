#!/usr/bin/env bash
set -eu -o pipefail

# shellcheck disable=SC2034
help_helm_dep_update="Used for updating chart deps"
helm_dep_update() {
  cd ./deploy && helm dependency update && cd ..
}

# shellcheck disable=SC2034
help_run_tests_ci="Run tests in CI"
run_tests_ci() {
  cd cmd && go list ./...  | circleci tests run --command "xargs gotestsum --junitfile junit.xml --format testname -- -v"
}

help_lint="Run lint"
lint() {
  set -x
  export PATH="$(go env GOPATH)/bin:$PATH"
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b "$(go env GOPATH)/bin" v2.9.0
  golangci-lint run
}

list() {
    declare -F | awk '{print $3}'
}

# shellcheck disable=SC2034
help_help="Print help text, or detailed help for a task."
help() {
    local item
    item="${1-}"
    if [ -n "${item}" ]; then
        local help_name
        help_name="help_${item//-/_}"
        echo "${!help_name-}"
        return
    fi

    if [ -z "${DO_HELP_SKIP_INTRO-}" ]; then
        type -t help-text-intro >/dev/null && help-text-intro
    fi
    for item in $(list); do
        local help_name text
        help_name="help_${item//-/_}"
        text="${!help_name-}"
        [ -n "$text" ] && printf "%-30s\t%s\n" "$item" "$(echo "$text" | head -1)"
    done
}

case "${1-}" in
list) list ;;
"" | "help") help "${2-}" ;;
*)
    if ! declare -F "${1}" >/dev/null; then
        printf "Unknown target: %s\n\n" "${1}"
        help
        exit 1
    else
        "$@"
    fi
    ;;
esac