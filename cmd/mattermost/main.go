// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	c3 "github.com/c3systems/c3-sdk-go"
	"github.com/c3systems/c3-sdk-go-example-mattermost/cmd/mattermost/commands"
	_ "github.com/c3systems/c3-sdk-go-example-mattermost/imports"      // Enterprise Deps
	"github.com/c3systems/c3-sdk-go-example-mattermost/model"          // Plugins
	_ "github.com/c3systems/c3-sdk-go-example-mattermost/model/gitlab" // Enterprise Imports
	"github.com/c3systems/c3-sdk-go-example-mattermost/utils"
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

var client = c3.NewC3()

//var client *c3.C3

const (
	key = "data"
)

// App ...
type App struct {
}

func (a *App) processReq(reqStr string) error {
	//// JUST FOR TESTING
	//stateBytes1, err := ioutil.ReadFile("./state.tar")
	//if err != nil {
	//	log.Printf("err reading state tar file\n%v", err)
	//	return err
	//}

	//if err := client.State().Set([]byte(key), stateBytes1); err != nil {
	//	log.Printf("err setting client state\n%v", err)
	//	return err
	//}
	//// DONE JUST FOR TESTING
	log.Println("running process req")
	prevState, found := client.State().Get([]byte(key))
	if !found {
		return errors.New("no previous state")
	}

	log.Printf("prev state\n%s", string(prevState))
	if err := os.Remove("./state.tar"); err != nil {
		log.Printf("err removing prev state.tar\n%v", err)
	}
	if err := ioutil.WriteFile("./state.tar", prevState, 0644); err != nil {
		log.Printf("err writing state.tar\n%v", err)
		return err
	}

	cmd := exec.Command("/bin/sh", "./set_state.sh")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("err running set_state\n%v\nnoutput:\n%s", err, string(out))
		return err
	}

	log.Printf("set state:\n%s", string(out))
	if err := cmd.Start(); err != nil {
		log.Printf("err executing set-state\n%v", err)
		return err
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("err waiting on set-state\n%v", err)
		return err
	}

	globalBytes, err := ioutil.ReadFile("./../../data/globals/globals.json")
	if err != nil {
		log.Printf("err reading global vars file\n%v", err)
		return err
	}
	var globals utils.Globals
	if err = json.Unmarshal(globalBytes, &globals); err != nil {
		log.Printf("err unmarshalling globals\n%v", err)
		return err
	}
	log.Printf("global vars:\n%v", globals)
	model.SeqUint64 = globals.SeqUint64
	model.SeqUint64ForPresave = globals.SeqUint64ForPresave
	model.SeqUint64ForPresaveMillis = globals.SeqUint64ForPresaveMillis

	b := []byte(reqStr)
	reqBytes := bytes.NewBuffer(b)
	dec := gob.NewDecoder(reqBytes)

	var tr utils.TransformedRequest
	if err = dec.Decode(&tr); err != nil {
		log.Printf("err decoding req:\n %v", err)
		return err
	}

	req, err := utils.UnTransformRequest(&tr)
	if err != nil {
		log.Printf("err untransforming request\n%v", err)
		return err
	}

	httpClient := http.Client{
		Timeout: time.Duration(10 * time.Second),
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("err sending req\n%v", err)
		return err
	}

	if resp.StatusCode != 200 {
		log.Printf("received non 200 status code\n%v", resp)
		return err
	}

	// write the globals to disk
	if err = os.Remove("./../../data/globals/globals.json"); err != nil {
		log.Printf("err removing old globals file\n%v", err)
	}
	globals = utils.Globals{
		SeqUint64:                 model.SeqUint64,
		SeqUint64ForPresave:       model.SeqUint64ForPresave,
		SeqUint64ForPresaveMillis: model.SeqUint64ForPresaveMillis,
	}
	d, err := json.Marshal(globals)
	if err != nil {
		log.Printf("err marshaling globals\n%v", err)
		return err
	}
	if err = ioutil.WriteFile("./../../data/globals/globals.json", d, 0644); err != nil {
		log.Printf("err writing globals file\n%v", err)
		return err
	}

	cmd = exec.Command("/bin/sh", "./get_state.sh")
	out, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("err running get_state\n%v\nnoutput:\n%s", err, string(out))
		return err
	}

	log.Println(string(out))
	//if err = cmd.Start(); err != nil {
	//	log.Printf("err getting state\n%v", err)
	//	return err
	//}

	//if err = cmd.Wait(); err != nil {
	//	log.Printf("err waiting on get-state\n%v", err)
	//	return err
	//}

	stateBytes, err := ioutil.ReadFile("./state.tar")
	if err != nil {
		log.Printf("err reading state tar file\n%v", err)
		return err
	}

	return client.State().Set([]byte(key), stateBytes)
}

func startC3() {
	data := &App{}
	if err := client.RegisterMethod("processReq", []string{"string"}, data.processReq); err != nil {
		log.Fatalf("err registering c3 method\n%v", err)
	}
	client.Serve()
}

func main() {
	seqUint64 := os.Getenv("SeqUint64")
	if seqUint64 == "" {
		seqUint64 = "0"
	}
	i, err := strconv.Atoi(seqUint64)
	if err != nil {
		log.Fatalf("err setting seqUint64\n%v", err)
	}
	model.SeqUint64 = uint64(i)

	seqUint64ForPresave := os.Getenv("SeqUint64ForPresave")
	if seqUint64ForPresave == "" {
		seqUint64ForPresave = "0"
	}
	i, err = strconv.Atoi(seqUint64ForPresave)
	if err != nil {
		log.Fatalf("err setting SeqUint64ForPresave\n%v", err)
	}
	model.SeqUint64ForPresave = uint64(i)

	seqUint64ForPresaveMillis := os.Getenv("SeqUint64ForPresaveMillis")
	if seqUint64ForPresaveMillis == "" {
		seqUint64ForPresaveMillis = "0"
	}
	i, err = strconv.Atoi(seqUint64ForPresaveMillis)
	if err != nil {
		log.Fatalf("err setting seqUint64ForPresaveMillis\n%v", err)
	}
	model.SeqUint64ForPresaveMillis = uint64(i)

	log.Printf("seqUint64 is %v", seqUint64)
	log.Printf("seqUint64ForPresave is %v", seqUint64ForPresave)
	log.Printf("seqUint64ForPresaveMillis is %v", seqUint64ForPresaveMillis)
	go startC3()
	if err := commands.Run(os.Args[1:]); err != nil {
		log.Fatalf("err running command\n%v", err)
	}
}
