# loadr

loadr is a library which extends the functionality of the standard html/template functionality by providing: 
1. *Compile time type safety* through the use of generics
2. All templates are *parsed, validated and cached on application startup*, fail-fast behaviour (there is no build step)
3. Shared data can easily be set between templates (for things such as cache busting)
4. Optional *live reload* capability (like VSCode's live server), any changes to watched files automatically refreshes the browser without needing recompilation
5. Simplifies layout, partials and component based templating
6. Thin wrapper around the  std lib html/templates
7. *Neglible performance penalty* compared to std lib, check micro-benchmarks in [_examples](_examples)


The library draws inspiration from [this article](https://philipptanlak.com/web-frontends-in-go/).

# Install

```
go get github.com/nesbyte/loadr
```

# Examples
See [_examples](_examples) for more complete and involved examples.  

# Benchmarks 
See [benchmarks](_examples/benchmark) inside examples to check some core performance metrics.

A quick extract below showing the Std vs loadr performance.
```
goos: linux
goarch: amd64
pkg: github.com/nesbyte/loadr/_examples/benchmark
cpu: AMD Ryzen 5 5600X 6-Core Processor
BenchmarkStdTemplates/Size_1             		 3239991              1878 ns/op            2096 B/op         20 allocs/op
BenchmarkStdTemplates/Size_1000           		  187657             32722 ns/op           15664 B/op         21 allocs/op
BenchmarkStdTemplates/Size_1000000           		 196          30327138 ns/op        24004792 B/op         24 allocs/op
BenchmarkLoadrInProductionMode/Size_1            3273147              1869 ns/op            2104 B/op         20 allocs/op
BenchmarkLoadrInProductionMode/Size_1000          188565             32187 ns/op           15672 B/op         21 allocs/op
BenchmarkLoadrInProductionMode/Size_1000000          198          30592559 ns/op        24004800 B/op         24 allocs/op
```

# About
The philosophy of loadr is to be robust and stable for web development with the goal of becoming "finished", introducing minimal abstractions and opinions. It builds on native Go templating, regular HTML, and the standard library's HTTP package (only for error handling).