"""
This is a utility script which can exercise the score-compose CLI
to produce outputs which get stored into files. These files are then
used as expected outcomes in the end 2 end test suite.
"""

import os
import shlex
import shutil
import subprocess

sc_exec = "score-compose"

arglist = [
    "--help",
    "completion",
    "completion bash",
    "completion bash --help",
    "completion fish",
    "completion fish --help",
    "completion powershell",
    "completion powershell --help",
    "completion zsh",
    "completion zsh --help",
    "run",
    "run --help",
    "unknown",
    "--version",
    "run -f example-score.yaml",
    "run -f example-score.yaml --build test",
    "run -f example-score.yaml --overrides overrides.yaml",
    "run --verbose",
]

dir = "temp"

shutil.rmtree(dir)

isExist = os.path.exists(dir)
if not isExist:
    os.makedirs(dir)


def run_sc_cmd(arguments):
    try:
        process = subprocess.Popen(
            shlex.split(f"{sc_exec} {arguments}"),
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
        )
        output_cmd = None
        while True:
            output = process.stdout.readline().decode()
            if output == "" and process.poll() is not None:
                break
            if output:
                if output_cmd != None:
                    output_cmd = f"{output_cmd}{output}"
                else:
                    output_cmd = output
    except:
        print(f"The command >>{sc_exec} {arguments}<< failed")
        print(str(process.stdout.readline()))
    rc = process.poll()
    output_cmd = output_cmd[:-1]
    return output_cmd


def write_file(name, content):
    file = open(f"{dir}/{name}-output.txt", "a")
    file.write(content)
    file.close


for arg in arglist:
    output = run_sc_cmd(arg)
    write_file(arg, output)
