# geo-cty-funcs

A Go module providing geographic, solar/astronomical, geodesic, and geometric
functions for use in [go-cty](https://github.com/zclconf/go-cty) / HCL2
evaluation contexts.

Functions are defined in [GEO-SPEC.md](https://github.com/tsarna/vinculum/blob/main/specs/GEO-SPEC.md)
and will be added incrementally.

## Installation

```
go get github.com/tsarna/geo-cty-funcs
```

## Usage

```go
import (
    geocty "github.com/tsarna/geo-cty-funcs"
    "github.com/zclconf/go-cty/cty/function"
)

// Register all functions in an HCL eval context
funcs := geocty.GetGeoFunctions()
// funcs is map[string]function.Function — merge into your eval context
```

## Status

Early development. No functions are registered yet.
