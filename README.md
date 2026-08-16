# upupidae

Upupidae is a Go package that let's you write your utility classes directly in
your Go code, without need for additional compile step. Styles are compiled
dynamically, when you "render" your components during runtime of your Go
program.

If you still want to compile your utility classes to a single CSS file, this is
doable with the current API. You just need to write a small Go program that
pipes classes into
[upupidae.Gen](https://pkg.go.dev/github.com/thinkofher/upupidae#Gen).

I've tried to come up with some generic API to support many components
libraries, but I won't lie that this package is opinionated around
[gomponents](https://www.gomponents.com/), because they are awesome and they
don't need additional compile step.

The API will probably change, there is no versioning yet and I am 100% sure
that there are some bugs. Additionally: my ambition isn't to be 100% compatible
with Tailwind. That being said, current implementation can be already quite
useful, so I encourage you to try and if you have some time, report bugs or
inconsistencies. Merge requests are always welcome!

## Examples

I've placed them [here](https://github.com/thinkofher/upupidae-examples).
