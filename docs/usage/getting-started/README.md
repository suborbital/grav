# Getting Started

To get started, import `github.com/suborbital/grav/grav` and create a Grav instance:

```go
package main

import (
	"fmt"

	"github.com/suborbital/grav/grav"
)

func gettingStarted() {
	g := grav.New()

	fmt.Println(g.NodeUUID)
}
```

Every Grav instance gets a UUID so that it can can identify itself to other instances when meshing is in use. `grav.New` can take a set of [options](grav-instance-options.md), but for an in-process Grav instance, no options are needed.

