package main

import (
	"flag"

	"log"
	"time"

	"bytes"
	"strings"

	"io/ioutil"
	"os"

	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/coreos/etcd/client"
)

var (
	endpoint   = flag.String("etcd", "http://127.0.0.1:2379", "Specifies the etcd endpoint")
	outputDir  = flag.String("outputDir", "/run/conf", "Specifies the output dir")
	etcdPrefix = flag.String("etcdPrefix", "", "Specifies the etcd prefix")
)

func getKeyName(node *client.Node) string {
	keySplitted := strings.Split(node.Key, "/")
	return keySplitted[len(keySplitted)-1]
}

func getConfigFile(configDirectory *client.Node) []byte {
	var buffer bytes.Buffer

	for _, key := range configDirectory.Nodes {
		envVariable := getKeyName(key)

		buffer.WriteString(envVariable + "=" + key.Value + "\n")
	}
	return buffer.Bytes()
}

func traverseConfigDirectory(configDirectories *client.Node) {
	for _, configNode := range configDirectories.Nodes {
		configFileName := (*outputDir) + "/" + getKeyName(configNode)
		ioutil.WriteFile(configFileName, getConfigFile(configNode), 0644)

		log.Print("Generated config file " + configFileName)
	}
}

func generateConfig(c client.Client) {
	kapi := client.NewKeysAPI(c)

	for {
		resp, err := kapi.Get(context.Background(), *etcdPrefix, &client.GetOptions{Recursive: true})
		panicOnError(err)
		traverseConfigDirectory(resp.Node)

		watcher := kapi.Watcher(*etcdPrefix, &client.WatcherOptions{Recursive: true, AfterIndex: resp.Index})
		ctx := context.Background()

		resp, err = watcher.Next(ctx)

		panicOnError(err)
	}
}

func dirExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func panicOnError(err error) {
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
}

func main() {
	flag.Parse()

	exists, err := dirExists(*outputDir)
	panicOnError(err)

	if !exists {
		err := os.Mkdir(*outputDir, 0755)
		panicOnError(err)
	}

	cfg := client.Config{
		Endpoints: []string{*endpoint},
		Transport: client.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
	}

	c, err := client.New(cfg)
	panicOnError(err)

	generateConfig(c)
}
