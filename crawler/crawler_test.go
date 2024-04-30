package crawler

import (
	"sync"
	"testing"

	"github.com/KYVENetwork/trustless-api/config"
)

func Benchmark(b *testing.B) {
	config.LoadConfig("../config.yml")
	c := Create()

	var wg sync.WaitGroup
	for _, bc := range c.children {
		current := bc
		wg.Add(1)
		go func() {
			current.Start()
			wg.Done()
		}()
	}
	wg.Wait()
}
