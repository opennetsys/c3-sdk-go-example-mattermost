# C3 Mattermost Example
This is a fork of the [Mattermost](https://www.mattermost.org/) open-source chat. You can think of it like an open-source Slack.

This fork incorporates the [c3-skd-go](https://github.com/c3systems/c3-sdk-go), and incorporates some other changes in order to make the business logic deterministic.

## Usage
In one terminal, spin up a C3 node. Be sure to first follow the [pre-install requirements](https://github.com/c3systems/c3-go#Install).
```bash
$ go get -u github.com/c3systems/c3-go
$ cd $GOPATH/github.com/c3systems/c3-go
$ c3-go generate key -o priv.pem
$ c3-go node start --pem=priv.pem --uri=/ip4/0.0.0.0/tcp/3330 --data-dir=~/.c3 --difficulty=5
```

Next, in another terminal build push this dApp to your local C3 network.
```bash
$ mkdir -p $GOPATH/src/github.com/c3systems
$ cd $GOPATH/src/github.com/c3systems
$ git clone git@github.com:c3systems/c3-sdk-go-example-mattermost.git
$ cd c3-sdk-go-example-mattermost

# build the image
$ docker build .

# push the dApp to C3
$ c3-go push $(docker images -q | grep -m1 "")

# send a genesis state
$ go run cmd/c3-frontend/main.go --image=<the docker image hash from the previous step> --peer=<peer id from the previous terminal> --genesis=true --genesisLoc=./state.tar

# run the dApp, locally
$ go run cmd/c3-frontend/main.go --image=<the docker image hash from two steps ago> --peer=<peer id from the previous terminal> --shouldNotListen=true
```

Finally navigate to `http://localhost:8065/public/channels/town-square`
