# Rotolog
[![License](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/catamat/rotolog/blob/master/LICENSE)
[![Build Status](https://travis-ci.org/catamat/rotolog.svg?branch=master)](https://travis-ci.org/catamat/rotolog)
[![Go Report Card](https://goreportcard.com/badge/github.com/catamat/rotolog)](https://goreportcard.com/report/github.com/catamat/rotolog)
[![Go Reference](https://pkg.go.dev/badge/github.com/catamat/rotolog.svg)](https://pkg.go.dev/github.com/catamat/rotolog)
[![Version](https://img.shields.io/github/tag/catamat/rotolog.svg?color=blue&label=version)](https://github.com/catamat/rotolog/releases)

Rotolog is a simple package for rotating logs based on the number of days or their size.

## Installation:
```
go get -u github.com/catamat/rotolog
```
## Example 1:
```golang
package main

import (
	"fmt"
	"os"
	"github.com/catamat/rotolog"

	"github.com/rs/zerolog"
)

func main() {
	myVar1 := 999999

	rotator, err := rotolog.NewFileDaysRotator("logs", 30)
	if err != nil {
		panic(err)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro
	log := zerolog.New(rotator).With().Timestamp().Caller().Logger()

	for i := 0; i < 10; i++ {
		log.Warn().Str("region", "us-west").Int("id", 2).Msg("")
		log.Info().Msg("info1")
		log.Printf("%d", myVar1)
		log.Error().Msg("asd2 \a \b \f \n \r \t \v \\ \\\\ ' \\' \\'\\' \" \"\" \\g uuuu \n\n")
		log.Debug().Msg("debug33 %d")

		time.Sleep(2 * time.Second)
	}
}

```

## Example 2:
```golang
package main

import (
	"fmt"
	"os"
	"github.com/catamat/rotolog"

	"github.com/rs/zerolog"
)

func main() {
	myVar1 := 999999

	rotator, err := rotolog.NewFileSizeRotator("logs", 1)
	if err != nil {
		panic(err)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro
	log := zerolog.New(rotator).With().Timestamp().Caller().Logger()

	for i := 0; i < 10; i++ {
		log.Warn().Str("region", "us-west").Int("id", 2).Msg("")
		log.Info().Msg("info1")
		log.Printf("%d", myVar1)
		log.Error().Msg("asd2 \a \b \f \n \r \t \v \\ \\\\ ' \\' \\'\\' \" \"\" \\g uuuu \n\n")
		log.Debug().Msg("debug33 %d")

		time.Sleep(2 * time.Second)
	}
}

```