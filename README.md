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
