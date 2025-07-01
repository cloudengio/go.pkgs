#DG=$HOME/sdk/dev/go/bin/go

$DG test -c . && ./heap.test --test.bench=Heap --test.cpuprofile=cpu.out --test.benchtime=5s && $DG tool pprof --http=:8080 heap.test cpu.out
