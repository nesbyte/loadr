# loadr
*Currently in development, API can still change slightly*

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
See [_examples](_examples) for more complete and involved examples

# About
The philosophy of loadr is to be robust and stable for web development with the goal of becoming "finished", introducing minimal abstractions and opinions. It builds on native Go templating, regular HTML, and the standard library's HTTP library.