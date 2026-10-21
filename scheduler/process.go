package scheduler

import (
	"sync"
	"time"

	"proj1/png"
)

// processTask loads the image of the task, applies its effects using the given
// number of workers per effect, and saves the result.
func processTask(task ImageTask, workers int) {
	start := time.Now()
	img, err := png.Load(task.InPath)
	if err != nil {
		panic(err)
	}
	loadTime.add(start)

	start = time.Now()
	applyEffects(img, task.Effects, workers)
	effectsTime.add(start)

	start = time.Now()
	if err := img.Save(task.OutPath); err != nil {
		panic(err)
	}
	saveTime.add(start)
}

// applyEffects applies the effects in order. Each effect is split into
// horizontal bands that are processed by the given number of goroutines. The
// wait group makes sure that every band of one effect is finished before the
// next effect starts, because a convolution reads pixels from neighboring
// bands.
func applyEffects(img *png.Image, effects []string, workers int) {
	height := img.Height()
	if workers > height {
		workers = height
	}

	for _, effect := range effects {
		if workers <= 1 {
			img.ApplyEffect(effect, 0, height)
		} else {
			applyEffectInBands(img, effect, workers)
		}
		img.Swap()
	}
}

func applyEffectInBands(img *png.Image, effect string, workers int) {
	height := img.Height()
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		yStart, yEnd := height*i/workers, height*(i+1)/workers
		go func() {
			defer wg.Done()
			img.ApplyEffect(effect, yStart, yEnd)
		}()
	}
	wg.Wait()
}
