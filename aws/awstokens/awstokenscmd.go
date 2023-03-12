// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"cloudeng.io/aws/awstokens"
	"github.com/aws/aws-sdk-go-v2/config"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Printf("failed to get aws config: %v\n", err)
		return
	}
	for _, name := range os.Args[1:] {
		token, err := awstokens.GetSecret(ctx, cfg, name)
		if err != nil {
			fmt.Printf("%v: %v\n", name, err)
			continue
		}
		fmt.Printf("%v: %v\n", name, token)
	}
}
