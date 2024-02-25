package grpcbench

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"
)

var profilepath = flag.String("profilepath", "", "write cpu profile to `file`")

func SetupProfiles() func() {
	cleanup := func() {}
	if *profilepath != "" {
		if err := os.MkdirAll(*profilepath, 0700); err != nil {
			log.Fatal("could not create profile directory: ", err)
		}
		go func() {
			i := 0
			for {
				time.Sleep(5 * time.Second)
				writeMemoryProfile(filepath.Join(*profilepath, fmt.Sprintf("memory_profile_%d.bin", i)))
				i++
			}
		}()

		f, err := os.Create(filepath.Join(*profilepath, "cpu_profile.bin"))
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		cleanup = func() {
			log.Println("stopping CPU profile")
			pprof.StopCPUProfile()
			f.Close()
		}
	}
	return cleanup
}

func writeMemoryProfile(path string) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	defer f.Close()
	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
}
