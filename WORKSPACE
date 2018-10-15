# IMPORTANT INFORMATION!
#
# This workspace configuraiton assumes `--experimental_enable_repo_mapping` command line argument
# on every build of anything under this workspace. The repo mapping is a recently released feature
# (bazel >= 0.16.0) and allows to solve two very nasty problems:
#     1) Allowing two or more workspaces with dependency conflicts (same dependency name but
#        different version) to coexist under same project (when one workspace imports another via
#        `*_reporitory()` workspace rules). Example: `gapic-generator` and `grpc-java` have a guava
#         dependency conflict (grpc-java uses latest java7 compatible version, while gapic-generator
#         uses fresher (java8 compatible) version)
#     2) Allowing two or more workspaces to have same dependency, but assign different names to it
#       (i.e. at least one of the workspaces does not follow naming conventions).
#
# Even if the experimental feature will be eventually cancelled, there will still be a solution
# for solving the problems desribed above, so it is relatively safe to structure workspaces and
# packages under assumption that the dependencies naming conflicts can be solved in the "outer"
# workspace (the one, which imports the "inner" workspace via `*_repository` rule).
#
# Note, the rule arguments related to the repo mapping feature (like repo_mapping arument in
# `*_repository` rules) are currently highlighted as erroneous by IDE plugins (is expected taking
# into account that the feature is new and experimental). This problem is temprorary and will go
# away once the repo mapping feature (or its "better" replacement) is stabilized.
#
# To fix enable bazel project sync in IntelliJ plugin add the following lines to .bazelproject file:
#
# build_flags:
#  --experimental_enable_repo_mapping

workspace(name = "com_google_discovery_artifact_manager")

#
# Java GAPIC (gapic-generator generated artifacts) dependencies. The dependencies may clash with
# gapic-generator and have different versions, especially taking into account that generated
# artifacts are Java 1.7 and gapic-generator is Java 1.8 compatible.
#
maven_jar(
    name = "com_google_guava_guava__com_google_api_codegen",
    artifact = "com.google.guava:guava:26.0-jre",
)

maven_jar(
    name = "com_google_api_grpc_proto_google_common_protos__com_google_api_codegen",
    artifact = "com.google.api.grpc:proto-google-common-protos:1.13.0-pre1",
)

git_repository(
    name = "com_google_api_codegen",
    remote = "https://github.com/googleapis/gapic-generator.git",
    commit = "4ae22668fb8dafbe6ecb476c0ffe83a28d2121fb",
    repo_mapping = {
        "@com_google_guava_guava": "@com_google_guava_guava__com_google_api_codegen",
        "@com_google_api_grpc_proto_google_common_protos": "@com_google_api_grpc_proto_google_common_protos__com_google_api_codegen",
    },
)

load(
    "@com_google_api_codegen//rules_gapic/java:java_gapic_pkg_repos.bzl",
    "java_gapic_direct_repositories",
    "java_gapic_gax_repositories",
)

java_gapic_direct_repositories(
    omit_junit_junit = True,
    omit_com_google_api_gax_grpc = True,
    omit_com_google_api_gax_grpc_testlib = True,
)

java_gapic_gax_repositories(
    omit_com_fasterxml_jackson_core_jackson_core = True,
)

#
# gapic-generator repository dependencies (required to compile and run gapic-generator)
#
load(
    "@com_google_api_codegen//:repositories.bzl",
    "com_google_api_codegen_repositories",
    "com_google_api_codegen_test_repositories",
    "com_google_api_codegen_tools_repositories",
)

#TODO:  Update all ommited dependencies in gapic_generator so they match the newer versions used by
#       grpc-java and gax
com_google_api_codegen_repositories(
    omit_com_google_api_api_common = True,
    omit_com_google_api_grpc_proto_google_common_protos = True,
    omit_com_google_code_findbugs_jsr305 = True,
    omit_com_google_code_gson_gson = True,
    omit_com_google_guava_guava = True,
    omit_io_grpc_grpc_core = True,
    omit_org_threeten_threetenbp = True,
)

com_google_api_codegen_test_repositories()

com_google_api_codegen_tools_repositories()

