#!/usr/bin/env python3
"""Package an explicitly reviewed capture list and independently check every byte."""

import argparse
import gzip
import hashlib
import io
import json
import tarfile
from pathlib import Path, PurePosixPath


def member_name(value):
    path = PurePosixPath(value)
    if (
        path.is_absolute()
        or ".." in path.parts
        or str(path) != value
        or value == "MANIFEST.json"
    ):
        raise ValueError("capture names must be distinct normalized relative files")
    return value


def bundle(root, names, output):
    names = sorted(member_name(name) for name in names)
    if not names or len(names) != len(set(names)):
        raise ValueError(
            "an explicit nonempty list without duplicate captures is required"
        )
    entries = []
    for name in names:
        source = root / name
        if (
            source.is_symlink()
            or not source.resolve().is_relative_to(root.resolve())
            or not source.is_file()
        ):
            raise ValueError(
                "capture is missing or does not refer to its original owned file: "
                + name
            )
        body = source.read_bytes()
        entries.append(
            {
                "path": name,
                "size": len(body),
                "sha256": hashlib.sha256(body).hexdigest(),
            }
        )
    manifest = (
        json.dumps({"format": 1, "files": entries}, sort_keys=True, indent=2).encode()
        + b"\n"
    )

    def compressor(stream):
        if output.suffix == ".zst":
            from compression import zstd

            return zstd.ZstdFile(
                stream,
                "wb",
                options={
                    zstd.CompressionParameter.compression_level: 19,
                    zstd.CompressionParameter.window_log: 27,
                },
            )
        return gzip.GzipFile(fileobj=stream, mode="wb", filename="", mtime=0)

    with output.open("xb") as stream, compressor(stream) as compressed, tarfile.open(
        fileobj=compressed, mode="w|"
    ) as archive:
        for name, body in [("MANIFEST.json", manifest)]:
            entry = tarfile.TarInfo(name)
            entry.size, entry.mode = len(body), 0o644
            archive.addfile(entry, io.BytesIO(body))
        for entry in entries:
            body = (root / entry["path"]).read_bytes()
            if (
                len(body) != entry["size"]
                or hashlib.sha256(body).hexdigest() != entry["sha256"]
            ):
                raise RuntimeError("capture changed during packaging: " + entry["path"])
            header = tarfile.TarInfo(entry["path"])
            header.size, header.mode = len(body), 0o644
            archive.addfile(header, io.BytesIO(body))
    return verify(output)


def verify(path):
    with tarfile.open(path, "r:*") as archive:
        members = archive.getmembers()
        names = [member.name for member in members]
        if len(names) != len(set(names)) or any(
            not member.isfile() for member in members
        ):
            raise ValueError("artifact must contain original distinct regular files")
        manifest = json.load(archive.extractfile("MANIFEST.json"))
        if manifest["format"] != 1:
            raise ValueError("unknown artifact manifest version")
        entries = manifest["files"]
        expected = [member_name(entry["path"]) for entry in entries]
        if len(expected) != len(set(expected)) or sorted(names) != sorted(
            ["MANIFEST.json", *expected]
        ):
            raise ValueError("manifest does not describe the complete artifact")
        for entry in entries:
            member = archive.getmember(entry["path"])
            if member.size != entry["size"]:
                raise ValueError("captured size changed: " + entry["path"])
            body = archive.extractfile(member).read()
            if hashlib.sha256(body).hexdigest() != entry["sha256"]:
                raise ValueError("captured bytes changed: " + entry["path"])
    with path.open("rb") as captured:
        archive_hash = hashlib.file_digest(captured, "sha256").hexdigest()
    return {
        "passed": True,
        "files": len(entries),
        "uncompressed_bytes": sum(entry["size"] for entry in entries),
        "archive_bytes": path.stat().st_size,
        "sha256": archive_hash,
    }


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    create = sub.add_parser("bundle")
    create.add_argument("--root", type=Path, required=True)
    create.add_argument(
        "--files",
        type=Path,
        required=True,
        help="JSON array of explicitly reviewed relative file names",
    )
    create.add_argument("--output", type=Path, required=True)
    check = sub.add_parser("verify")
    check.add_argument("archive", type=Path)
    args = parser.parse_args()
    result = (
        verify(args.archive)
        if args.command == "verify"
        else bundle(args.root, json.loads(args.files.read_text()), args.output)
    )
    print(json.dumps(result))
