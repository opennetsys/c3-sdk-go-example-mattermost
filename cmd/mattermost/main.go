// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"log"
	"os"
	"strconv"

	"github.com/mattermost/mattermost-server/cmd/mattermost/commands"
	"github.com/mattermost/mattermost-server/model"

	// Plugins
	_ "github.com/mattermost/mattermost-server/model/gitlab"

	// Enterprise Imports
	_ "github.com/mattermost/mattermost-server/imports"

	// Enterprise Deps
	_ "github.com/dgryski/dgoogauth"
	_ "github.com/go-ldap/ldap"
	_ "github.com/hako/durafmt"
	_ "github.com/hashicorp/memberlist"
	_ "github.com/mattermost/rsc/qr"
	_ "github.com/prometheus/client_golang/prometheus"
	_ "github.com/prometheus/client_golang/prometheus/promhttp"
	_ "github.com/tylerb/graceful"
	_ "gopkg.in/olivere/elastic.v5"
)

func main() {
	seqUint64 := os.Getenv("SeqUint64")
	if seqUint64 == "" {
		seqUint64 = "0"
	}
	i, err := strconv.Atoi(seqUint64)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("seqUint64 is %d", seqUint64)
	model.SeqUint64 = uint64(i)

	if err := commands.Run(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
