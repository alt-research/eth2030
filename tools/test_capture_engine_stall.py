import os
import subprocess
import sys
import threading
import unittest
import tempfile
import json
from pathlib import Path
from unittest.mock import patch
import capture_engine_stall as collector
from capture_engine_stall import pipe_lines


class PipeCaptureTests(unittest.TestCase):
    def test_quiet_after_burst_delivers_timeout_without_another_write(self):
        # The child keeps stdout open. EOF or a subsequent log line cannot
        # rescue a collector that left its timeout in a text decoder buffer.
        child = subprocess.Popen([sys.executable, "-c", "import os,sys;os.write(1,b'accepted block77622\\nengine_request_timeout\\n');sys.stdin.buffer.read()"],
                                 stdin=subprocess.PIPE, stdout=subprocess.PIPE, bufsize=0)
        stop = threading.Event()
        delivered = threading.Event()
        lines = []
        def read():
            for line in pipe_lines(child.stdout, stop.is_set, poll_seconds=.05):
                if line is not None:
                    lines.append(line)
                if line == b"engine_request_timeout":
                    delivered.set()
                    return
        thread = threading.Thread(target=read)
        thread.start()
        try:
            self.assertTrue(delivered.wait(3), "timeout stayed hidden until a new write/EOF")
            self.assertIsNone(child.poll(), "writer must still be alive when the timeout arrives")
            self.assertEqual(lines, [b"accepted block77622", b"engine_request_timeout"])
        finally:
            stop.set()
            child.terminate()
            child.wait(timeout=3)
            thread.join(timeout=3)
            child.stdin.close()
            child.stdout.close()
            self.assertFalse(thread.is_alive())

    def run_chunks(self, chunks, max_line=1 << 20):
        reader, writer = os.pipe()
        for chunk in chunks:
            os.write(writer, chunk)
        os.close(writer)
        with os.fdopen(reader, "rb", buffering=0) as stream:
            return list(pipe_lines(stream, max_line=max_line))

    def test_multiple_lines_and_unterminated_eof(self):
        self.assertEqual(self.run_chunks([b"a\nengine_", b"request_timeout\nlast"]),
                         [b"a", b"engine_request_timeout", b"last"])

    def test_empty_eof(self):
        self.assertEqual(self.run_chunks([]), [])

    def test_partial_line_survives_a_quiet_poll(self):
        reader, writer = os.pipe()
        try:
            with os.fdopen(reader, "rb", buffering=0) as stream:
                lines = pipe_lines(stream, poll_seconds=.01)
                os.write(writer, b"engine_request_")
                self.assertIsNone(next(lines))
                os.write(writer, b"timeout\n")
                self.assertEqual(next(lines), b"engine_request_timeout")
                lines.close()
        finally:
            os.close(writer)

    def test_failed_initialization_is_explicit_not_watching(self):
        with tempfile.TemporaryDirectory() as directory:
            out = Path(directory) / "capture"
            result = subprocess.run([sys.executable, str(Path(__file__).with_name("capture_engine_stall.py")),
                                     "--record", str(Path(directory) / "missing.json"),
                                     "--output", str(out)], capture_output=True, timeout=3)
            self.assertNotEqual(result.returncode, 0)
            value = json.loads((out / "result.json").read_text())
            self.assertEqual(value["status"], "collector_failed")
            self.assertFalse(value["actual48h_passed"])
            self.assertIn("FileNotFoundError", value["error"])

    def test_log_process_launch_failure_cannot_leave_watching_status(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            record = root / "record.json"
            record.write_text(json.dumps({"el": "el", "cl": "cl", "profile": {
                "source": "fixture", "cl_source": "fixture", "binary_sha256": "fixture",
                "cl_binary_sha256": "fixture"}}))
            output = root / "capture"
            with patch.object(sys, "argv", ["capture", "--record", str(record), "--output", str(output)]), \
                 patch.object(collector, "identity", return_value={}), \
                 patch.object(collector, "inspect", return_value={"NetworkSettings": {"Ports": {"9001/tcp": [{"HostPort": "1"}]}}}), \
                 patch.object(collector, "native_profile", return_value={}), \
                 patch.object(collector.signal, "signal"), \
                 patch.object(collector.subprocess, "Popen", side_effect=OSError("log launch failed")):
                with self.assertRaisesRegex(OSError, "log launch failed"):
                    collector.main()
            self.assertEqual(json.loads((output / "result.json").read_text())["status"], "collector_failed")

    def test_oversized_complete_and_incomplete_lines_fail(self):
        for body in [b"12345\n", b"12345"]:
            with self.subTest(body=body), self.assertRaises(ValueError):
                self.run_chunks([body], max_line=4)

    def test_stopped_reader_does_not_wait_for_a_live_writer(self):
        reader, writer = os.pipe()
        try:
            with os.fdopen(reader, "rb", buffering=0) as stream:
                self.assertEqual(list(pipe_lines(stream, stop=lambda: True)), [])
        finally:
            os.close(writer)


if __name__ == "__main__":
    unittest.main()
