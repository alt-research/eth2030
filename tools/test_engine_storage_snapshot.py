import errno
import json
import os
import subprocess
import sys
import tempfile
import time
import unittest
from pathlib import Path
from unittest.mock import patch
import engine_storage_snapshot as storage
import capture_engine_stall as collector


@unittest.skipUnless(Path("/proc/self/task").exists(), "Linux procfs required")
class StorageSnapshotTests(unittest.TestCase):
    def test_live_blocked_reader_binds_thread_fd_inode_and_device(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "owned-database-file"
            path.write_bytes(b"owned fixture")
            child = subprocess.Popen([sys.executable, "-c", "import os,sys;fd=os.open(sys.argv[1],os.O_RDONLY);print(fd,flush=True);sys.stdin.buffer.read(1)", str(path)], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
            try:
                fd = child.stdout.readline().strip()
                started = storage.start_ticks(child.pid)
                deadline = time.monotonic() + 3
                while True:
                    result = storage.snapshot(child.pid, started, [{"Type": "bind", "Source": directory, "Destination": directory}])
                    thread = next(t for t in result["threads"] if t["tid"] == str(child.pid))
                    syscall = thread["syscall"].get("text", "").split()
                    if "pipe_read" in thread["wchan"].get("text", "") and len(syscall) > 1 and syscall[1] == "0x0":
                        break
                    self.assertLess(time.monotonic(), deadline, thread)
                    time.sleep(.01)
                self.assertTrue(result["identity_matches"])
                self.assertIsNone(child.poll())
                input_fd = next(item for item in result["fds"] if item["fd"] == "0")
                self.assertTrue(input_fd["target"].startswith("pipe:"))
                self.assertTrue(input_fd["binding_stable"])
                file_fd = next(item for item in result["fds"] if item["fd"] == fd)
                self.assertEqual(file_fd["target"], str(path))
                self.assertEqual(file_fd["inode"], path.stat().st_ino)
                self.assertEqual(file_fd["device"], f"{os.major(path.stat().st_dev)}:{os.minor(path.stat().st_dev)}")
                self.assertTrue(file_fd["binding_stable"])
                self.assertEqual(file_fd["bind_mount"]["device"], file_fd["device"])
                self.assertIn("text", result["process"]["io"])
                self.assertFalse(result["original_fault_resolved"])
                destination = Path(directory) / "storage.json"
                info = {"State": {"Pid": child.pid, "StartedAt": "fixture"}, "Mounts": []}
                with patch.object(collector, "inspect", return_value=info):
                    receipt = collector.capture_storage("owned", {"host_pid": child.pid, "started": "fixture", "start_ticks": started}, destination)
                self.assertTrue(receipt["identity_matches"])
                self.assertTrue(json.loads(destination.read_text())["identity_matches"])
                with patch.object(collector, "inspect", return_value=info):
                    with self.assertRaisesRegex(ValueError, "restarted"):
                        collector.capture_storage("owned", {"host_pid": child.pid + 1, "started": "fixture"}, destination)
            finally:
                child.terminate()
                child.wait(timeout=3)
                child.stdin.close()
                child.stdout.close()

    def test_reused_file_descriptor_is_not_reported_as_a_stable_binding(self):
        with tempfile.TemporaryDirectory() as directory:
            old = Path(directory) / "old"
            new = Path(directory) / "new"
            old.write_text("before")
            new.write_text("after")
            code = "import os,sys;fd=os.open(sys.argv[1],os.O_RDONLY);print(fd,flush=True);sys.stdin.buffer.read(1);os.close(fd);next_fd=os.open(sys.argv[2],os.O_RDONLY);assert fd==next_fd;print('changed',flush=True);sys.stdin.buffer.read(1)"
            child = subprocess.Popen([sys.executable, "-c", code, str(old), str(new)], stdin=subprocess.PIPE, stdout=subprocess.PIPE)
            try:
                fd = child.stdout.readline().decode().strip()
                link = Path("/proc") / str(child.pid) / "fd" / fd
                real_readlink = os.readlink
                changed = False
                def swap_after_first_observation(path):
                    nonlocal changed
                    target = real_readlink(path)
                    if Path(path) == link and not changed:
                        changed = True
                        child.stdin.write(b"n")
                        child.stdin.flush()
                        self.assertEqual(child.stdout.readline(), b"changed\n")
                    return target
                with patch.object(storage.os, "readlink", side_effect=swap_after_first_observation):
                    result = storage.snapshot(child.pid, storage.start_ticks(child.pid))
                item = next(item for item in result["fds"] if item["fd"] == fd)
                self.assertTrue(changed)
                self.assertFalse(item["binding_stable"])
            finally:
                child.terminate()
                child.wait(timeout=3)
                child.stdin.close()
                child.stdout.close()

    def test_pid_reuse_or_missing_identity_does_not_capture_another_process(self):
        for expected in ["different-start", None]:
            result = storage.snapshot(os.getpid(), expected)
            self.assertFalse(result["identity_matches"])
            self.assertNotIn("threads", result)
            self.assertNotIn("fds", result)

    def test_permissions_and_disappeared_files_are_explicit(self):
        with patch.object(Path, "open", side_effect=PermissionError(errno.EACCES, "denied")):
            result = storage.read_text(Path("/fixture"))
        self.assertEqual(result["errno"], errno.EACCES)
        self.assertNotIn("text", result)
        with tempfile.TemporaryDirectory() as directory:
            self.assertEqual(storage.read_text(Path(directory) / "gone")["errno"], errno.ENOENT)

    def test_capture_rejects_restart_or_namespace_identity_change_after_observation(self):
        pid = os.getpid()
        expected = {"host_pid": pid, "started": "before", "start_ticks": storage.start_ticks(pid)}
        info = {"State": {"Pid": pid, "StartedAt": "before"}, "Mounts": []}
        changed = {"State": {"Pid": pid + 1, "StartedAt": "after"}, "Mounts": []}
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "frame.json"
            with patch.object(collector, "inspect", side_effect=[info, changed]):
                receipt = collector.capture_storage("owned", expected, path)
            self.assertFalse(receipt["identity_matches"])
            self.assertTrue(json.loads(path.read_text())["target_restarted_during_capture"])
            with patch.object(collector, "inspect", return_value=info), patch.object(collector, "namespace_storage", return_value={"observation": {"identity_matches": False}}):
                receipt = collector.capture_storage("owned", expected, path, "fixture")
            self.assertFalse(receipt["identity_matches"])
            self.assertFalse(json.loads(path.read_text())["namespace_identity_matches"])

    def test_namespace_helper_timeout_is_bounded_and_removed(self):
        calls = []
        def run(command, **options):
            calls.append(command)
            if command[1] == "run":
                self.assertEqual(options["timeout"], 8)
                raise subprocess.TimeoutExpired(command, 8)
            return subprocess.CompletedProcess(command, 0, b"", b"")
        with patch.object(collector.subprocess, "run", side_effect=run):
            result = collector.namespace_storage("owned-target", "123", "sha256:fixture")
        self.assertIn("TimeoutExpired", result["error"])
        self.assertEqual(result["helper_remove_exit"], 0)
        command = calls[0]
        self.assertEqual(command[command.index("--pid") + 1], "container:owned-target")
        self.assertEqual(command[command.index("--network") + 1], "none")
        self.assertIn("--read-only", command)
        self.assertEqual(calls[-1][-1], command[command.index("--name") + 1])
        self.assertEqual(collector.namespace_storage("owned-target", None, "fixture"),
                         {"error": "target process start identity is unavailable"})

    def test_large_files_and_directory_entries_are_bounded_and_marked(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)
            for number in range(3):
                (path / str(number)).write_bytes(b"abcdef")
            result = storage.read_text(path / "0", limit=4)
            self.assertEqual(result["text"], "abcd")
            self.assertTrue(result["truncated"])
            names, clipped, error = storage.entries(path, 2)
            self.assertEqual(len(names), 2)
            self.assertTrue(clipped)
            self.assertIsNone(error)


if __name__ == "__main__":
    unittest.main()
