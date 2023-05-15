import subprocess, os

BASE_PATH = os.path.dirname(os.path.abspath(__file__))
os.chdir(BASE_PATH)

for file in os.listdir("static_files"):
    retcode = subprocess.call(["aws", "s3", "cp", f"static_files/{file}", "s3://r-place-client"])
    print(f"Upload {file} complete with code {retcode}")
