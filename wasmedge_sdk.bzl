"""Repository rule for a caller-provided WasmEdge SDK."""

def _is_absolute_path(path):
    if path.startswith("/") or path.startswith("\\\\"):
        return True
    return len(path) >= 3 and path[1] == ":" and path[2] in ("/", "\\")

def _required_path(repository_ctx, root, relative):
    path = root.get_child(relative)
    if not path.exists:
        fail(
            "WASMEDGE_SDK is missing {relative}: {path}".format(
                relative = relative,
                path = path,
            ),
        )
    return path

def _wasmedge_sdk_repository_impl(repository_ctx):
    sdk = repository_ctx.os.environ.get("WASMEDGE_SDK")
    if not sdk:
        fail(
            "WASMEDGE_SDK is required; set it to an absolute WasmEdge 0.17.1 " +
            "SDK prefix containing include/ and lib/",
        )

    if not _is_absolute_path(sdk):
        fail("WASMEDGE_SDK must be an absolute path, got: " + sdk)
    root = repository_ctx.path(sdk)

    include = _required_path(
        repository_ctx,
        root,
        "include/wasmedge/wasmedge.h",
    )
    repository_ctx.symlink(include.dirname.dirname, "include")

    os_name = repository_ctx.os.name
    if os_name == "mac os x":
        library = _required_path(
            repository_ctx,
            root,
            "lib/libwasmedge.0.dylib",
        )
        repository_ctx.symlink(library.dirname, "lib")
        import_attributes = 'shared_library = "lib/libwasmedge.0.dylib",'
    elif os_name == "linux":
        library = _required_path(
            repository_ctx,
            root,
            "lib/libwasmedge.so.0",
        )
        repository_ctx.symlink(library.dirname, "lib")
        import_attributes = 'shared_library = "lib/libwasmedge.so.0",'
    elif os_name.startswith("windows"):
        interface_library = _required_path(
            repository_ctx,
            root,
            "lib/wasmedge.lib",
        )
        shared_library = _required_path(
            repository_ctx,
            root,
            "bin/wasmedge.dll",
        )
        repository_ctx.symlink(interface_library.dirname, "lib")
        repository_ctx.symlink(shared_library.dirname, "bin")
        import_attributes = """
    interface_library = "lib/wasmedge.lib",
    shared_library = "bin/wasmedge.dll","""
    else:
        fail(
            "WASMEDGE_SDK Bazel integration supports Linux, macOS, and " +
            "Windows; got host OS: " + os_name,
        )

    repository_ctx.file(
        "BUILD.bazel",
        """
load("@rules_cc//cc:defs.bzl", "cc_import", "cc_library")

cc_import(
    name = "wasmedge_import",
    {import_attributes}
)

cc_library(
    name = "wasmedge",
    hdrs = glob([
        "include/**/*.h",
        "include/**/*.inc",
    ]),
    includes = ["include"],
    deps = [":wasmedge_import"],
    visibility = ["//visibility:public"],
)
""".format(import_attributes = import_attributes),
    )

wasmedge_sdk_repository = repository_rule(
    implementation = _wasmedge_sdk_repository_impl,
    environ = ["WASMEDGE_SDK"],
    local = True,
)
