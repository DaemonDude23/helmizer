# Release selection tests; no git mutations or remote calls.
import importlib.util
import unittest
from pathlib import Path
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("release_plan", Path(__file__).with_name("release-plan.py"))
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)


class ReleasePlanTests(unittest.TestCase):
    def check(self, tags, same, changed, ref, bump, expected):
        def fake(*args):
            if args[0] == "tag":
                return tags
            if args[0] == "rev-parse":
                return "head"
            if args[0] == "rev-list":
                return "head" if same else "old"
            if args[0] == "diff":
                return changed
            raise AssertionError(args)

        with patch.object(release, "git", side_effect=fake):
            self.assertEqual(release.plan(ref, bump, (0, 20, 0)), expected)

    def test_patch(self):
        self.check("v0.20.0", False, "src/go.mod", "refs/heads/main", "patch", ("v0.20.1", False))

    def test_minor(self):
        self.check("v0.20.0", False, "src/main.go", "refs/heads/main", "minor", ("v0.21.0", False))

    def test_major(self):
        self.check("v0.20.0", False, "src/main.go", "refs/heads/main", "major", ("v1.0.0", False))

    def test_documentation_only(self):
        self.check("v0.20.0", False, "", "refs/heads/main", "patch", ("", True))

    def test_retry_same_commit(self):
        self.check("v0.20.0", True, "", "refs/heads/main", "patch", ("v0.20.0", False))

    def test_first_release(self):
        self.check("", False, "", "refs/heads/main", "patch", ("v0.20.0", False))

    def test_prerelease_ignored(self):
        self.check("v0.20.0\nv0.21.0-rc1", False, "src/main.go", "refs/heads/main", "patch", ("v0.20.1", False))

    def test_older_release_rejected(self):
        with self.assertRaises(ValueError):
            self.check("v0.20.0\nv0.21.0", False, "", "refs/tags/v0.20.0", "patch", None)

    def test_older_ref_cannot_hide_newer_tag(self):
        def fake(*args):
            if args[0] == "rev-parse":
                return "old-head"
            if args == ("tag", "--merged", "HEAD"):
                return "v0.20.0"
            if args == ("tag",):
                return "v0.20.0\nv0.21.0"
            raise AssertionError(args)

        with patch.object(release, "git", side_effect=fake):
            with self.assertRaises(ValueError):
                release.plan("refs/tags/v0.20.0", "patch", (0, 20, 0))

    def test_divergent_history_cannot_promote_an_older_version(self):
        def fake(*args):
            if args[0] == "rev-parse":
                return "head"
            if args == ("tag", "--merged", "HEAD"):
                return "v0.20.0"
            if args == ("tag",):
                return "v0.20.0\nv0.21.0"
            if args[0] == "diff":
                return "src/main.go"
            raise AssertionError(args)

        with patch.object(release, "git", side_effect=fake):
            with self.assertRaises(ValueError):
                release.plan("refs/heads/main", "patch", (0, 20, 0))

    def test_version_baseline_advances_release_without_double_bumping(self):
        with patch.object(release, "git", side_effect=["head", "v0.20.0", "v0.20.0", "old", "src/VERSION"]):
            self.assertEqual(release.plan("refs/heads/main", "patch", (0, 21, 0)), ("v0.21.0", False))

    def test_noncanonical_version_is_rejected(self):
        for version in ["v0.020.1", "0.20.01", "v0.20.1-rc1"]:
            with self.subTest(version=version), self.assertRaises(ValueError):
                release.parse_version(version)

    def metadata_history(self, head="main-head", changed="", version="0.20.1", release_files="src/VERSION"):
        def fake(*args):
            if args == ("rev-parse", "HEAD"):
                return head
            if args == ("tag", "--merged", "HEAD"):
                return "v0.20.0"
            if args == ("tag",):
                return "v0.20.0\nv0.20.1"
            if args == ("diff", "--name-only", "v0.20.1^", "v0.20.1"):
                return release_files
            if args == ("show", "v0.20.1:src/VERSION"):
                return version
            if args == ("rev-parse", "v0.20.1^"):
                return "main-head"
            if args[0] == "merge-base":
                return "main-head"
            if args[0] == "diff":
                self.assertIn(":(exclude)src/VERSION", args)
                return changed
            raise AssertionError(args)

        return patch.object(release, "git", side_effect=fake)

    def test_release_metadata_commit_is_retried_on_same_source(self):
        with self.metadata_history():
            self.assertEqual(release.plan("refs/heads/main", "patch", (0, 20, 0)), ("v0.20.1", False))

    def test_release_version_difference_does_not_cause_another_release(self):
        with self.metadata_history(head="docs-only-head"):
            self.assertEqual(release.plan("refs/heads/main", "patch", (0, 20, 0)), ("", True))

    def test_dependency_change_after_metadata_commit_creates_next_patch(self):
        with self.metadata_history(head="new-main-head", changed="src/go.mod"):
            self.assertEqual(release.plan("refs/heads/main", "patch", (0, 20, 0)), ("v0.20.2", False))

    def test_explicit_baseline_bump_is_honored_without_other_changes(self):
        with self.metadata_history(head="feature-version-head"):
            self.assertEqual(release.plan("refs/heads/main", "patch", (0, 21, 0)), ("v0.21.0", False))

    def test_metadata_commit_cannot_hide_source_changes(self):
        with self.metadata_history(release_files="src/VERSION\nsrc/main.go"):
            with self.assertRaises(ValueError):
                release.plan("refs/heads/main", "patch", (0, 20, 0))

    def test_metadata_version_must_match_tag(self):
        with self.metadata_history(version="0.20.2"):
            with self.assertRaises(ValueError):
                release.plan("refs/heads/main", "patch", (0, 20, 0))

    def test_metadata_tag_is_allowed_only_atop_default_branch(self):
        responses = ["release-head", "main-head", "src/VERSION", "0.20.1", "main-head", "main-head"]
        with patch.object(release, "git", side_effect=responses):
            release.validate_history("refs/tags/v0.20.1", "origin/main")
        with patch.object(release, "git", side_effect=["release-head", "main-head", "src/main.go"]):
            with self.assertRaises(ValueError):
                release.validate_history("refs/tags/v0.20.1", "origin/main")

    def test_manual_tag_must_match_source_version(self):
        with patch.object(release, "git", side_effect=["head", "v0.20.1", "v0.20.1", "0.20.1"]):
            self.assertEqual(release.plan("refs/tags/v0.20.1", "patch", (0, 20, 1)), ("v0.20.1", False))
        for value in ["0.20.0", "v0.20.1"]:
            with patch.object(release, "git", side_effect=["head", "v0.20.1", "v0.20.1", value]):
                with self.assertRaises(ValueError):
                    release.plan("refs/tags/v0.20.1", "patch", (0, 20, 0))


if __name__ == "__main__":
    unittest.main()
