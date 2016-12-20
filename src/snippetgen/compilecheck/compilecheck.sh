#!/bin/bash

# Script to generate and run the compilecheck code.
#
# Export the following environment variables to customize this
# compilecheck. Otherwise, sensible defaults are used.
#
# COMPILECHECK_GOPATH # if different from ${GOPATH}
# COMPILECHECK_DIR  # where to output the compilecheck-generated files
# COMPILECHECK_DARTMAN # path to the discovery-artifact-manager repo
#   # This sets the default value for:
#   COMPILECHECK_TOOLKIT # location of the Github toolkit repo
#     # This sets the default value for:
#     COMPILECHECK_DISCOVERY # directory of the discovery files to generate
# COMPILECHECK_LANGUAGES # which languages to compile-check
#   # This is a string of space-separated language identifiers from the
#   # following set:
#   #   java php csharp nodejs ruby go py

[[ "${BASH_SOURCE[0]}" != "${0}" ]] && { echo "Please execute this script rather than sourcing it!"; return; }

DATE="$(date '+%Y-%m-%d.%H%M%S')"
PARENT_DIR="${COMPILECHECK_DIR:-${TMPDIR:-/tmp}}/compilecheck-${DATE}"
COMPILECHECK_GOPATH="${COMPILECHECK_GOPATH:-${GOPATH}}"
COMPILECHECK_DARTMAN="${COMPILECHECK_DARTMAN:-${HOME}/discovery-artifact-manager}"
COMPILECHECK_TOOLKIT="${COMPILECHECK_TOOLKIT:-${COMPILECHECK_DARTMAN}/toolkit}"
COMPILECHECK_DISCOVERY="${COMPILECHECK_DISCOVERY:-${COMPILECHECK_TOOLKIT}/src/test/java/com/google/api/codegen/testdata/discoveries}"
COMPILECHECK_LANGUAGES="${COMPILECHECK_LANGUAGES:-java php csharp nodejs ruby go py}"
CHECK_SCRIPT="${PARENT_DIR}/checker"

cat <<EOF
$0:
Running compilecheck in $PARENT_DIR

    COMPILECHECK_TOOLKIT="${COMPILECHECK_TOOLKIT}"
   COMPILECHECK_DARTMAN="${COMPILECHECK_DARTMAN}"
  COMPILECHECK_DISCOVERY="${COMPILECHECK_DISCOVERY}"
     COMPILECHECK_GOPATH="${COMPILECHECK_GOPATH}"

EOF

function join_by { IFS="$1"; shift; echo "$*"; }
function terminate { echo "$0: Terminating ($1)" ; exit $1; }
function allyamls() {
  echo "$@"
  export FILE="$1"
  echo -n ${ALL_YAMLS} | xargs -d " " -t -IYAMLF java -jar ./build/libs/discoGen-0.0.5-SNAPSHOT.jar --discovery_doc="${FILE}" --gapic_yaml=YAMLF --overrides="${FILE}.overrides" --output="${SNIPPET_DIR}"  || terminate $?
}
export -f terminate
export -f allyamls

export SNIPPET_DIR="${PARENT_DIR}/snippet"
CHECK_DIR="${PARENT_DIR}/check"
LIB_DIR="${PARENT_DIR}/lib"
mkdir -p "${SNIPPET_DIR}" "${CHECK_DIR}" "${LIB_DIR}"

LANGUAGE_YAMLS=$(echo -n "${COMPILECHECK_LANGUAGES}" | xargs -d " " -ILANG echo "${COMPILECHECK_TOOLKIT}/src/main/resources/com/google/api/codegen/LANG/LANG_discovery.yaml")
export GOPATH="${COMPILECHECK_GOPATH}"
export ALL_YAMLS="$(join_by ' ' $(echo ${LANGUAGE_YAMLS} | sed s/py_discovery/python_discovery/))"


pushd "${COMPILECHECK_TOOLKIT}"

echo "$0: Building discoGen jar..."
 ./gradlew discoJar

echo "$0: Generating snippets..."
find "${COMPILECHECK_DISCOVERY}" -type f -regex "${COMPILECHECK_DISCOVERY}/[^/]*.json\$" | xargs -t -IFILE bash -c 'allyamls FILE'

popd  #  "${COMPILECHECK_TOOLKIT}"

pushd "${COMPILECHECK_DARTMAN}"

echo "$0: Running compilecheck binary (generating compilecheck snippets)"
go run ./src/snippetgen/compilecheck/compilecheck.go --tst "${CHECK_DIR}" --lib "${LIB_DIR}" "${SNIPPET_DIR}" > ${CHECK_SCRIPT} || terminate $?
echo "$0: Compile-checking (running the generated compilecheck snippets)"
{ . ${CHECK_SCRIPT}; } || terminate $?

popd  # "${COMPILECHECK_DARTMAN}"

echo "$0: Success."
