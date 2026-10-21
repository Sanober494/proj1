# Project 1 Report: Parallel Image Processing System

## 1. Project description

The program applies a list of image effects (sharpen `S`, edge detection `E`,
blur `B`, grayscale `G`) to PNG images described by the JSON strings in
`data/effects.txt`. The three convolution effects use a 3x3 kernel written by
hand (no library) with zero padding at the borders. Three versions exist:

| Mode | Command | Idea |
|---|---|---|
| sequential | `go run editor.go small` | One image after the other, one thread |
| `parfiles` | `go run editor.go small parfiles 8` | One goroutine per image, images taken from a shared queue |
| `parslices` | `go run editor.go small parslices 8` | One image at a time; every effect is split into horizontal bands processed by N goroutines |

### Code structure

* `png/png.go`: `Image` holds two RGBA64 buffers, `cur` and `next`. An effect
  reads `cur` and writes `next`; `Swap()` exchanges the two pointers, so chaining
  effects never copies pixels. `Save` writes `cur`.
* `png/effects.go`: `ApplyEffect(effect, yStart, yEnd)` processes only the rows
  `[yStart, yEnd)`, so the same code serves the sequential and both parallel
  versions. Pixels are accessed directly in the `Pix` buffer (no `At`/`Set`
  calls, which allocate and are slow). Border pixels skip the kernel taps that
  fall outside the image, which is identical to zero padding.
* `scheduler/task.go`: parses `effects.txt` for every data directory in the
  `a+b+c` argument and builds `ImageTask`s (the `<dir>_` prefix is added to the
  output name).
* `scheduler/lock.go`, `queue.go`: own test-and-set lock (`atomic.SwapInt32`)
  and a FIFO queue whose `Pop` is protected by it.
* `scheduler/process.go`: `processTask` (load, effects, save) and
  `applyEffects`, shared by all three versions. With 1 worker it runs inline;
  with N workers it splits each effect into N bands and waits on a
  `sync.WaitGroup` before the next effect, because a convolution reads rows
  that neighboring bands write for the previous effect.
* `scheduler/timing.go`: with `PROJ1_TIMING=1` the program prints the time spent
  in load / effects / save to stderr; used for the Amdahl's law computation.

### Correctness

* All outputs on `small` match `data/expected` with a maximum difference of 1
  out of 255 per channel (rounding) and the mean absolute error below 0.004.
* `parfiles` and `parslices` produce pixel-identical output to the sequential
  version for several thread counts (including more threads than images and
  non-divisor thread counts), and `go build -race` shows no data races.
  The result is deterministic.

## 2. How to run

```
# on fe.ai.cs.uchicago.edu, once:
cd ~ && wget -qO- https://go.dev/dl/go1.19.13.linux-amd64.tar.gz | tar xz
# place data/ inside proj1/, edit the --chdir line of benchmark/benchmark-proj1.sh
mkdir -p proj1/benchmark/slurm/out
cd proj1 && sbatch benchmark/benchmark-proj1.sh
```

The script builds the program, runs every combination of
`{parfiles, parslices} x {small, mixture, big} x {1,2,4,6,8,12} threads` five
times (best run is used), writes `benchmark/results.csv`, the Amdahl tables
`benchmark/amdahl.md`, and the graphs `benchmark/speedup-parfiles.png` and
`benchmark/speedup-parslices.png`. Run time is well below the 2 h limit set in
the script header.

## 3. Hotspots and bottlenecks of the sequential program

Phase breakdown of one sequential run on `small` (single thread, `PROJ1_TIMING=1`):

| Phase | Time (s) | Share |
|---|---|---|
| Load (PNG decode + conversion to RGBA64) | 0.45 | 16 % |
| Effects (convolutions / grayscale) | 0.86 | 31 % |
| Save (PNG encode + write) | 1.42 | 52 % |
| Total | 2.73 | |

(`big+small` shows the same proportions: 7.6 / 14.1 / 21.3 s.)

The largest hotspot is therefore **PNG encoding**, followed by the convolutions,
and decoding. Encoding 16-bit RGB data with deflate is expensive and is done by
the standard library on one thread. I use `png.BestSpeed` to reduce it.
The effects themselves are compute-bound and independent per pixel, which makes
them ideal to parallelize, but they are only about a third of the runtime.

## 4. Amdahl's law

`S(N) = 1 / ((1 - p) + p / N)`, `p` = fraction of the 1-thread time that is
parallelized, limit for infinite threads `1 / (1 - p)`.

* `parfiles`: every image is processed completely by one goroutine, so
  load + effects + save are parallel; only parsing `effects.txt` and starting
  the goroutines are sequential, so `p` is about 1 (limited in practice by
  the number of images, 10, and by their uneven sizes).
* `parslices`: only the effects are parallel; decode and encode stay sequential,
  so `p = effects / total`.

Worked example with the numbers of section 3 (`small`, `parslices`):
`p = 0.86 / 2.73 = 0.315`

| N | 2 | 4 | 6 | 8 | 12 | infinity |
|---|---|---|---|---|---|---|
| S(N) | 1.19 | 1.31 | 1.36 | 1.38 | 1.41 | 1.46 |

**The complete tables for `small`, `mixture` and `big` with the measurements
from the peanut cluster are produced by the benchmark script in
`benchmark/amdahl.md`; paste them here:**

> RESULTS_TO_INSERT_FROM_CLUSTER (amdahl.md, speedup-parfiles.png, speedup-parslices.png)

## 5. Analysis (to be completed together with the cluster numbers)

Expected outcome from the design, to be checked against the cluster graphs:

* **Which version is faster?** `parfiles` should scale much better because
  100 % of the per-image work is parallel and there is no synchronization
  except one cheap queue pop per image. `parslices` is capped at about 1.4x to
  1.5x by Amdahl's law because decoding and encoding are sequential and make up
  roughly 2/3 of the run time.
* **Effect of problem size.** Bigger images make the parallel part of
  `parslices` larger relative to the cost of spawning goroutines and barriers
  per effect, so `big` should be closest to the Amdahl prediction. For
  `parfiles`, scaling is limited by load balance: there are only 10 images,
  so with 12 threads at most 10 goroutines run, and the largest image
  determines the finishing time (especially in `mixture`, where sizes differ a lot).
* **Distance to Amdahl.** Measurements will lie below the prediction because of
  memory bandwidth (convolutions stream through large buffers), goroutine
  creation and `WaitGroup` barriers per effect, cache effects, uneven bands,
  and, for `parfiles`, the load imbalance above and the heavier memory use of
  many images in flight.

## 6. Possible further improvements

* **Overlap I/O with computation in `parslices`:** decode the next image while
  the current one is being filtered, and encode the previous one in the
  background. This attacks the sequential 2/3 directly, and is the largest
  possible gain.
* **Parallel PNG encoding/decoding** by splitting the deflate stream
  (e.g. parallel gzip-style blocks), since compression is the biggest hotspot.
* **Dynamic work distribution in `parfiles`:** start with the largest images
  first (or split big images into bands) to reduce the tail where one thread
  works alone.
* **Fusing effects:** neighboring convolutions could be done in one pass using
  overlapping bands that shrink with each effect (the second option in the
  assignment), which removes the barrier between effects and reads each pixel
  only once. Fewer passes also reduce memory traffic.
* **Persistent worker goroutines** with a reusable barrier instead of spawning
  goroutines for every effect.
