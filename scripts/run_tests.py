"""Run the application and isolated Docker integration test suites."""

import os
from pathlib import Path
import shutil
import subprocess
import sys
import uuid


ROOT = Path(__file__).resolve().parent.parent


def run_tests(up_flags):
    windows = os.name == "nt"
    wrapper = ROOT / "backend" / ("mvnw.cmd" if windows else "mvnw")
    npm = "npm.cmd" if windows else "npm"

    if shutil.which(npm) is None:
        raise FileNotFoundError(
            "npm was not found on PATH; install Node.js 24 with npm "
            "before running the frontend tests"
        )

    subprocess.run(
        [str(wrapper), "--batch-mode", "--no-transfer-progress", "verify"],
        cwd=ROOT / "backend", check=True,
    )
    subprocess.run([npm, "ci"], cwd=ROOT / "frontend", check=True)
    subprocess.run([npm, "test"], cwd=ROOT / "frontend", check=True)

    env = os.environ.copy()
    env.update(COMPOSE_PROJECT_NAME=f"mimir-test-{uuid.uuid4().hex}", FRONTEND_PORT="0")
    compose = ["docker", "compose"]
    try:
        subprocess.run(
            [*compose, "up", *up_flags, "--detach", "--wait", "--wait-timeout", "180"],
            cwd=ROOT, env=env, check=True,
        )
        port = subprocess.run(
            [*compose, "port", "frontend", "80"],
            cwd=ROOT, env=env, check=True, stdout=subprocess.PIPE, text=True,
        ).stdout.strip()
        env["STACK_URL"] = f"http://{port}"
        subprocess.run(
            [sys.executable, "-m", "unittest", "discover", "-s", "tests/docker", "-v"],
            cwd=ROOT, env=env, check=True,
        )
    except (OSError, subprocess.CalledProcessError):
        try:
            subprocess.run(
                [*compose, "logs", "--no-color"], cwd=ROOT, env=env, check=True,
            )
        except (OSError, subprocess.CalledProcessError) as error:
            print(f"Could not retrieve Compose logs: {error}", file=sys.stderr)
        raise
    finally:
        subprocess.run(
            [*compose, "down", "--volumes", "--remove-orphans"],
            cwd=ROOT, env=env, check=True,
        )


if __name__ == "__main__":
    try:
        run_tests(sys.argv[1:])
    except (OSError, subprocess.CalledProcessError) as error:
        print(f"Test run failed: {error}", file=sys.stderr)
        sys.exit(1)
