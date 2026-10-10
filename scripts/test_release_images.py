# Exercise release stages with a fake Docker CLI; never contact a registry.
import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("release-images.sh").resolve()
FAKE_DOCKER = """
import hashlib, json, os, sys
from pathlib import Path
args = sys.argv[1:]
with open(os.environ["DOCKER_LOG"], "a") as log:
    log.write(json.dumps(args) + "\\n")
if args[:2] == ["buildx", "build"]:
    output = args[args.index("--output") + 1]
    Path(output.split("dest=", 1)[1]).write_text("saved-image")
if args[0] == "run" and os.environ.get("FAIL_SCAN"):
    sys.exit(1)
if args[:2] == ["image", "inspect"]:
    image = args[-1]
    digest = hashlib.sha256(image.encode()).hexdigest()
    print(image.rsplit(":", 1)[0] + "@sha256:" + digest)
"""


class ReleaseImageTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.bin = self.root / "bin"
        self.bin.mkdir()
        docker = self.bin / "docker"
        docker.write_text(f"#!{sys.executable}\n" + FAKE_DOCKER)
        docker.chmod(0o755)
        self.log = self.root / "commands.jsonl"
        self.images = self.root / "images"
        self.env = {
            **os.environ,
            "PATH": str(self.bin) + os.pathsep + os.environ["PATH"],
            "TAG": "v0.20.1",
            "GITHUB_REPOSITORY": "DaemonDude23/helmizer",
            "IMAGE_DIR": str(self.images),
            "TRIVY_IMAGE": "scanner:review",
            "DOCKER_LOG": str(self.log),
        }
        self.env.pop("FAIL_SCAN", None)

    def run_stage(self, stage, success=True):
        result = subprocess.run(["bash", str(SCRIPT), stage], env=self.env, capture_output=True, text=True)
        if success:
            self.assertEqual(result.returncode, 0, result.stderr)
        else:
            self.assertNotEqual(result.returncode, 0)
        return [json.loads(line) for line in self.log.read_text().splitlines()] if self.log.exists() else []

    def test_publish_loads_saved_images_without_rebuilding(self):
        calls = self.run_stage("build-scan")
        builds = [args for args in calls if args[:2] == ["buildx", "build"]]
        self.assertEqual(len(builds), 5)
        self.assertEqual(len([args for args in calls if args[0] == "run"]), 5)
        self.assertFalse(any(args[0] == "push" for args in calls))
        self.log.unlink()
        calls = self.run_stage("push")
        self.assertFalse(any(args[:2] == ["buildx", "build"] for args in calls))
        self.assertEqual(len([args for args in calls if args[0] == "load"]), 5)
        self.assertEqual(len([args for args in calls if args[0] == "push"]), 5)
        manifests = [args for args in calls if args[:3] == ["buildx", "imagetools", "create"]]
        self.assertEqual(len(manifests), 2)
        for args in manifests:
            self.assertNotIn(":latest", " ".join(args))
            self.assertTrue(all("@sha256:" in item for item in args[5:]))
        self.log.unlink()
        promoted = self.run_stage("promote")
        self.assertEqual(len(promoted), 2)
        self.assertTrue(all(args[4].endswith(":latest") for args in promoted))

    def test_failed_scan_prevents_publication_even_with_stale_markers(self):
        self.run_stage("build-scan")
        self.env["FAIL_SCAN"] = "1"
        self.run_stage("build-scan", success=False)
        self.log.unlink()
        calls = self.run_stage("push", success=False)
        self.assertEqual(calls, [])

    def test_push_requires_all_archives(self):
        self.run_stage("build-scan")
        (self.images / "helmizer-arm64.tar").unlink()
        self.log.unlink()
        self.run_stage("push", success=False)
        self.assertFalse((self.images / "pushed").exists())

    def test_modified_archive_cannot_be_published(self):
        self.run_stage("build-scan")
        (self.images / "helmizer-amd64.tar").write_text("modified-after-scan")
        self.log.unlink()
        self.assertEqual(self.run_stage("push", success=False), [])

    def test_changed_version_cannot_reuse_previous_scans(self):
        self.run_stage("build-scan")
        self.env["TAG"] = "v0.20.2"
        self.log.unlink()
        self.assertEqual(self.run_stage("push", success=False), [])

    def test_latest_requires_successful_versioned_publication(self):
        self.run_stage("build-scan")
        self.log.unlink()
        self.assertEqual(self.run_stage("promote", success=False), [])

    def test_invalid_version_is_rejected_before_docker_runs(self):
        self.env["TAG"] = "v0.020.1"
        self.assertEqual(self.run_stage("build-scan", success=False), [])


if __name__ == "__main__":
    unittest.main()
