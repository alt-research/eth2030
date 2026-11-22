import argparse
import json
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

import glamsterdam_soak_audit as audit
import glamsterdam_soak_supervisor as supervisor
from test_glamsterdam_soak_audit import START, operating


class SupervisorTests(unittest.TestCase):
    def test_unowned_container_is_never_stopped(self):
        with patch.object(audit, "run_json", return_value=[{"Config": {"Labels": {}}}]), \
                patch.object(supervisor.subprocess, "run") as stop:
            result = supervisor.stop_owned({"cl": "not-ours-cl", "el": "not-ours-el"})
        stop.assert_not_called()
        self.assertTrue(all(not item["clean"] for item in result.values()))

    def test_stop_command_failure_is_not_successful_cleanup(self):
        owned = {"Config": {"Labels": {"eth2030.qualification": audit.LABEL}},
                 "State": {"Running": True}}
        with patch.object(audit, "run_json", return_value=[owned]), \
                patch.object(supervisor.subprocess, "run", return_value=SimpleNamespace(returncode=1)):
            result = supervisor.stop_owned({"cl": "ours-cl", "el": "ours-el"})
        self.assertTrue(all(not item["clean"] for item in result.values()))

    def test_corrective_cleanup_does_not_erase_failed_qualification(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "run").mkdir()
            (root / "finality").mkdir()
            state, _ = operating()
            audit.write_json(root / "run/result.json", state)
            audit.write_json(root / "finality/result.json", {"status": "collecting", "windows": []})
            audit.write_json(root / "record.json", {"cl": "owned-cl", "el": "owned-el"})
            args = argparse.Namespace(output=root / "supervisor", run=root / "run",
                                      finality=root / "finality", record=root / "record.json")
            with patch.object(audit, "audit", side_effect=audit.EvidenceError("delayed sample")), \
                    patch.object(supervisor, "stop_owned", return_value={"clean": True}) as cleanup, \
                    patch.object(supervisor, "datetime") as clock:
                clock.now.return_value = START
                with self.assertRaisesRegex(audit.EvidenceError, "delayed sample"):
                    supervisor.supervise(args)
            cleanup.assert_called_once()
            result = json.loads((args.output / "result.json").read_text())
            self.assertFalse(result["passed"])
            self.assertEqual(result["status"], "failed")
            self.assertEqual(result["failure"]["message"], "delayed sample")
            self.assertTrue(result["cleanup_after_failure"]["clean"])


if __name__ == "__main__":
    unittest.main()
