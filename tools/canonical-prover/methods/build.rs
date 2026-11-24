use risc0_build::{DockerOptionsBuilder, GuestOptionsBuilder};
use std::{collections::HashMap, env, fs, path::PathBuf};

fn main() {
    // Pin the complete build environment. Cargo's crate metadata includes path
    // identities that compiler-only prefix remapping cannot normalize reliably.
    // The SDK container builds under /src with a fixed registry and target root.
    assert!(
        env::var("RISC0_SKIP_BUILD").unwrap_or_default().is_empty(),
        "canonical proofs require the real guest image"
    );
    env::remove_var("RISC0_DOCKER_CONTAINER_TAG");
    env::set_var("DOCKER_BUILDKIT", "1");
    let manifest = PathBuf::from(env::var_os("CARGO_MANIFEST_DIR").unwrap());
    let root = manifest.parent().unwrap();
    let docker = DockerOptionsBuilder::default()
        .root_dir(root)
        .docker_container_tag(
            "r0.1.97.0@sha256:7ae0a27fa72b10fb5c1066005bd949ae43dc4ead8a284f2d6572b99c82ab4468",
        )
        .build()
        .unwrap();
    let options = GuestOptionsBuilder::default()
        .use_docker(docker)
        .build()
        .unwrap();
    let methods =
        risc0_build::embed_methods_with_options(HashMap::from([("canonical_guest", options)]));
    let actual: String = methods[0]
        .image_id
        .as_words()
        .iter()
        .flat_map(|word| word.to_le_bytes())
        .map(|byte| format!("{byte:02x}"))
        .collect();
    let expected =
        fs::read_to_string(root.join("image-id.txt")).expect("pinned canonical image ID");
    assert_eq!(
        actual,
        expected.trim(),
        "unexpected canonical image; audit the build/source before changing the trusted ID"
    );
    println!("cargo:rerun-if-changed=../image-id.txt");
}
