#!/bin/bash
#
#SBATCH --mail-user=sanober@cs.uchicago.edu
#SBATCH --mail-type=ALL
#SBATCH --job-name=proj1_benchmark
#SBATCH --output=./slurm/out/%j.%N.stdout
#SBATCH --error=./slurm/out/%j.%N.stderr
#SBATCH --chdir=ABSOLUTE_PATH_TO_PROJ1_DIRECTORY
#SBATCH --partition=peanut-cpu
#SBATCH --nodes=1
#SBATCH --ntasks=1
#SBATCH --cpus-per-task=16
#SBATCH --mem-per-cpu=900
#SBATCH --exclusive
#SBATCH --time=02:00:00
# Note: 2 modes x 3 data sets x 6 thread counts x 5 runs, including the big
# images, takes a while. Lower --time only if you also lower RUNS below.
#
# Usage (from the proj1 directory, after replacing ABSOLUTE_PATH_TO_PROJ1_DIRECTORY
# above and creating the log directory once):
#
#     mkdir -p benchmark/slurm/out   # only needed once  (see note on --output)
#     sbatch benchmark/benchmark-proj1.sh
#
# Produces in proj1/benchmark:
#     results.csv           every single run (time + per-phase breakdown)
#     amdahl.md             Amdahl's law calculation tables
#     speedup-parfiles.png  speedup graph of the "parfiles" version
#     speedup-parslices.png speedup graph of the "parslices" version

# Use the user-local Go installation (Go 1.19)
export PATH=$HOME/go/bin:$PATH

RUNS=${RUNS:-5}                       # Repetitions per experiment (best one is used)
THREADS=${THREADS:-"1 2 4 6 8 12"}    # 1 is the baseline for the speedup
DIRS=${DIRS:-"small mixture big"}
MODES=${MODES:-"parfiles parslices"}

BENCH_DIR=benchmark
RESULTS=$BENCH_DIR/results.csv

# Build once so compilation time is not part of any measurement
(cd editor && go build -o ../$BENCH_DIR/editor_bin .) || exit 1

# The editor must be run from inside the editor directory (relative paths)
cd editor
echo "mode,dir,threads,run,total,load,effects,save" > ../$RESULTS

for mode in $MODES; do
  for dir in $DIRS; do
    for threads in $THREADS; do
      for run in $(seq 1 "$RUNS"); do
        # stdout: total seconds (printed by editor.go); stderr: phase breakdown
        total=$(PROJ1_TIMING=1 ../$BENCH_DIR/editor_bin "$dir" "$mode" "$threads" 2>/tmp/phases.$$)
        phases=$(sed -E 's/(load|effects|save)=//g; s/ /,/g' /tmp/phases.$$)
        echo "$mode,$dir,$threads,$run,$total,$phases" >> ../$RESULTS
      done
    done
  done
done
rm -f /tmp/phases.$$ ../$BENCH_DIR/editor_bin
cd ..

# Install matplotlib for the plots if the node does not have it
python3 -c "import matplotlib" 2>/dev/null || python3 -m pip install --user matplotlib

python3 $BENCH_DIR/analyze.py $RESULTS $BENCH_DIR
