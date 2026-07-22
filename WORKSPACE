workspace(name = "discovery_artifact_manager")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

##############################################################################
# Common
##############################################################################
# Protobuf depends on very old version of bazel_skylib (forward compatible with the new one).
# Importing older version of bazel_skylib first (this is one of the things that protobuf_deps() call
# below will do) breaks the grpc repositories, so importing the proper version explicitly before
# everything else.
http_archive(
    name = "bazel_skylib",
    urls = ["https://github.com/bazelbuild/bazel-skylib/releases/download/0.9.0/bazel_skylib-0.9.0.tar.gz"],
)

http_archive(
    name = "com_google_protobuf",
    strip_prefix = "protobuf-fe1790ca0df67173702f70d5646b82f48f412b99",  # this is 3.11.2
    urls = ["https://github.com/protocolbuffers/protobuf/archive/fe1790ca0df67173702f70d5646b82f48f412b99.zip"],
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

http_archive(
    name = "rules_proto",
    sha256 = "f2b2eb422d010c06ae0067e297c364aa35515e095cf319867a741a32cf9cc52f",
    strip_prefix = "rules_proto-dcd61fec58ad7d9fa49a3736a2afbce29cf927c2",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_proto/archive/dcd61fec58ad7d9fa49a3736a2afbce29cf927c2.tar.gz",
        "https://github.com/bazelbuild/rules_proto/archive/dcd61fec58ad7d9fa49a3736a2afbce29cf927c2.tar.gz",
    ],
)

load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies", "rules_proto_toolchains")

rules_proto_dependencies()

rules_proto_toolchains()

# Note gapic-generator contains java-specific and common code, that is why it is imported in common
# section
http_archive(
    name = "com_google_api_codegen",
    strip_prefix = "gapic-generator-128224d3c06a23fe55a32abd5d3760515974291d",
    urls = ["https://github.com/googleapis/gapic-generator/archive/128224d3c06a23fe55a32abd5d3760515974291d.zip"],
)

##############################################################################
# Java
##############################################################################
http_archive(
    name = "com_google_api_gax_java",
    strip_prefix = "gax-java-1.54.0",
    urls = ["https://github.com/googleapis/gax-java/archive/v1.54.0.zip"],
)

load("@com_google_api_gax_java//:repository_rules.bzl", "com_google_api_gax_java_properties")

com_google_api_gax_java_properties(
    name = "com_google_api_gax_java_properties",
    file = "@com_google_api_gax_java//:dependencies.properties",
)

load("@com_google_api_gax_java//:repositories.bzl", "com_google_api_gax_java_repositories")

com_google_api_gax_java_repositories()

# gapic-generator transitive
# (goes AFTER java-gax, since both java gax and gapic-generator are written in java and may conflict)
load("@com_google_api_codegen//:repository_rules.bzl", "com_google_api_codegen_properties")

com_google_api_codegen_properties(
    name = "com_google_api_codegen_properties",
    file = "@com_google_api_codegen//:dependencies.properties",
)

load("@com_google_api_codegen//:repositories.bzl", "com_google_api_codegen_repositories")

http_archive(
    name = "com_google_protoc_java_resource_names_plugin",
    strip_prefix = "protoc-java-resource-names-plugin-7632facb0a1d9f76d9152eefe21422dc55e36524",
    urls = ["https://github.com/googleapis/protoc-java-resource-names-plugin/archive/7632facb0a1d9f76d9152eefe21422dc55e36524.zip"],
)

com_google_api_codegen_repositories()

load("@com_google_googleapis//:repository_rules.bzl", "switched_rules_by_language")

switched_rules_by_language(
    name = "com_google_googleapis_imports",
)

# protoc-java-resource-names-plugin (loaded in com_google_api_codegen_repositories())
# (required to support resource names feature in gapic generator)
load(
    "@com_google_protoc_java_resource_names_plugin//:repositories.bzl",
    "com_google_protoc_java_resource_names_plugin_repositories",
)

com_google_protoc_java_resource_names_plugin_repositories()
