# Rotolog
[![License](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/catamat/rotolog/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/catamat/rotolog)](https://goreportcard.com/report/github.com/catamat/rotolog)
[![Go Reference](https://pkg.go.dev/badge/github.com/catamat/rotolog.svg)](https://pkg.go.dev/github.com/catamat/rotolog)
[![Version](https://img.shields.io/github/tag/catamat/rotolog.svg?color=blue&label=version)](https://github.com/catamat/rotolog/releases)

Rotolog is a package for rotating logs based on the number of days or their size.

## Installation

```bash
go get github.com/catamat/rotolog@latest
```

## Behavior

Both rotators are safe for concurrent `Write()` calls and expose a `Close()` method to release the active file handle when your application shuts down.
After `Close()`, further `Write()` calls return `ErrClosed`.

#### NewFileDaysRotator(folder, days)

- Creates one file per day using the format `YYYY-MM-DD.log`.
- Removes only files in `folder` that match the same naming pattern.
- Deletes files whose age is greater than or equal to `days`.
- Uses the process timezone for both file naming and retention.
- `days` must be greater than zero.ok sistema 

#### NewFileSizeRotator(folder, sizeMB)

- Writes to `half-1.log`.
- When `half-1.log` reaches about half of `sizeMB`, it is moved to `half-2.log` and a new `half-1.log` is created.
- This keeps roughly `sizeMB` megabytes of recent logs split across the two files.
- `sizeMB` must be greater than zero.

## Example: Rotate By Days

```go
package main

import (
	"github.com/catamat/rotolog"
	"github.com/rs/zerolog"
)

func main() {
	rotator, err := rotolog.NewFileDaysRotator("logs", 30)
	if err != nil {
		panic(err)
	}
	defer rotator.Close()

	log := zerolog.New(rotator).With().Timestamp().Logger()
	log.Info().Msg("application started")
}
```

## Example: Rotate By Retained Size

```go
package main

import (
	"github.com/catamat/rotolog"
	"github.com/rs/zerolog"
)

func main() {
	rotator, err := rotolog.NewFileSizeRotator("logs", 10)
	if err != nil {
		panic(err)
	}
	defer rotator.Close()

	log := zerolog.New(rotator).With().Timestamp().Logger()
	log.Info().Msg("application started")
}
```