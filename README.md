# semtype

Determine golang module version based on exported types.

## Overview

`semtype` is a tool that analyzes the exported types and functions in a Go
module and determines the semantic version based on changes to these exports. It
helps in maintaining versioning consistency by automatically bumping the version
when there are changes to the exported API.

## Installation

To install `semtype`, use the `go get` command:

```sh
go get github.com/jtarchie/semtype
```

## Usage

You can run `semtype` using the `go run` command. Here is an example:

```sh
go run github.com/jtarchie/semtype -dir ./path/to/your/module
```

By default, `semtype` will look for a state file named `semtype.dat` in the
specified directory. You can specify a different state file using the`-state`
flag:

```sh
go run github.com/jtarchie/semtype -dir ./path/to/your/module -state ./path/to/state/file.dat
```

## Versioning Rules

`semtype` follows semantic versioning rules to determine whether a change is a
patch, minor, or major update based on the changes to the exported types and
functions in a Go module.

### Patch Version

A patch version is incremented when backward-compatible bug fixes are made.
Examples include:

- Fixing a bug in an existing function without changing its signature.

```go
// Before
func Add(a, b int) int {
    return a + b
}

// After
func Add(a, b int) int {
    if a == 0 {
        return b
    }
    return a + b
}
```

### Minor Version

A minor version is incremented when new, backward-compatible functionality is
added. Examples include:

- Adding a new function or method.

```go
// Before
package math

// After
package math

// New function added
func Subtract(a, b int) int {
    return a - b
}
```

- Adding a new field to a struct without removing or changing existing fields.

```go
// Before
type Point struct {
    X int
    Y int
}

// After
type Point struct {
    X int
    Y int
    Z int // New field added
}
```

### Major Version

A major version is incremented when there are changes that are not
backward-compatible. Examples include:

- Changing the signature of an existing function.

```go
// Before
func Add(a, b int) int {
    return a + b
}

// After
func Add(a, b, c int) int { // Function signature changed
    return a + b + c
}
```

- Removing an existing function or method.

```go
// Before
func Multiply(a, b int) int {
    return a * b
}

// After
// Multiply function removed
```

- Changing the type of an existing field in a struct.

```go
// Before
type Point struct {
    X int
    Y int
}

// After
type Point struct {
    X float64 // Field type changed
    Y int
}
```
