import subprocess


# TODO: Move this to use Popen so that we can live-stream the output
def run_command(commands):
    process = subprocess.run(commands)
    return process.returncode
