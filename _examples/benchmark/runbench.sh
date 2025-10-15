#!/bin/bash
# Force all execution on to one core/thread
# 5s is necessary to allow the benchmark to stabilize properly
GOMAXPROCS=1 taskset -c 0 go test -bench=. -benchtime=5s -benchmem
goos: linux