#!/usr/bin/env python3
# Select a release without writing git refs or contacting GitHub.
import argparse
import re
import subprocess
from pathlib import Path

STABLE_VERSION = r"v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)"


def git(*args):
    return subprocess.check_output(["git", *args], text=True).strip()


def parse_version(value):
    if not re.fullmatch(STABLE_VERSION, value):
        raise ValueError(f"Expected stable X.Y.Z version, got {value!r}")
    return tuple(map(int, value.removeprefix("v").split(".")))


def next_version(current, bump):
    major, minor, patch = current
    return {"patch": (major, minor, patch + 1), "minor": (major, minor + 1, 0), "major": (major + 1, 0, 0)}[bump]


def metadata_source(tag, history):
    """Accept a release commit only if it changes VERSION alone atop history."""
    try:
        if git("diff", "--name-only", f"{tag}^", tag) != "src/VERSION":
            return None
        if git("show", f"{tag}:src/VERSION") != tag.removeprefix("v"):
            return None
        parent = git("rev-parse", f"{tag}^")
        if git("merge-base", parent, history) == parent:
            return parent
    except (subprocess.CalledProcessError, ValueError):
        pass
    return None


def validate_history(ref, default_branch):
    head = git("rev-parse", "HEAD")
    if git("merge-base", head, default_branch) == head:
        return
    if ref.startswith("refs/tags/") and metadata_source(ref.removeprefix("refs/tags/"), default_branch):
        return
    raise ValueError("Release source must be on default-branch history or change VERSION alone atop that history")


def plan(ref, bump, baseline):
    head = git("rev-parse", "HEAD")
    merged = set(git("tag", "--merged", "HEAD").splitlines())
    all_tags = sorted(
        (parse_version(tag), tag) for tag in git("tag").splitlines() if tag.startswith("v") and re.fullmatch(STABLE_VERSION, tag)
    )
    if ref.startswith("refs/tags/"):
        tag = ref.removeprefix("refs/tags/")
        parse_version(tag)
        if not tag.startswith("v"):
            raise ValueError("Release tags must start with v")
        if all_tags and parse_version(tag) < all_tags[-1][0]:
            raise ValueError("Refusing to promote an older release to latest")
        if git("show", f"{tag}:src/VERSION") != tag.removeprefix("v"):
            raise ValueError("Tag version does not match src/VERSION; prepare the source version before creating a manual tag")
        return tag, False
    if all_tags:
        previous, previous_tag = all_tags[-1]
        source = None
        if previous_tag not in merged:
            source = metadata_source(previous_tag, "HEAD")
            if source is None:
                raise ValueError("Newest stable tag is outside this history; retry that tag or resolve divergent release history")
        if (source or git("rev-list", "-n", "1", previous_tag)) == head:
            # Resume a partial publication, or skip if already published (workflow).
            return previous_tag, False
        changed = git(
            "diff",
            "--name-only",
            previous_tag,
            "HEAD",
            "--",
            "src",
            "Dockerfile",
            "Dockerfile.helm",
            ".dockerignore",
            "action.yml",
            "flake.nix",
            "flake.lock",
            ".goreleaser.yaml",
            ":(exclude)src/VERSION",
        )
        # Ignore the release-only VERSION difference, but honor an intentional
        # development baseline bump made on main for a new feature release.
        if not changed and baseline <= previous:
            return "", True
        target = max(next_version(previous, bump), baseline)
    else:
        target = baseline
    return "v" + ".".join(map(str, target)), False


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--ref", required=True)
    parser.add_argument("--bump", choices=["patch", "minor", "major"], default="patch")
    parser.add_argument("--default-branch", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    validate_history(args.ref, args.default_branch)
    tag, skip = plan(args.ref, args.bump, parse_version(Path("src/VERSION").read_text().strip()))
    with open(args.output, "a", encoding="utf-8") as output:
        output.write(f"tag={tag}\nskip={str(skip).lower()}\n")


if __name__ == "__main__":
    main()
