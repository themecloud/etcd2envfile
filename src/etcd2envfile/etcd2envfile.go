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

	"github.com/coreos/go-systemd/daemon"
	systemdutil "github.com/coreos/go-systemd/util"
)

var (
	endpoint   = flag.String("etcd", "http://127.0.0.1:2379", "Specifies the etcd endpoint")
	outputDir  = flag.String("outputDir", "/run/conf", "Specifies the output dir")
	etcdPrefix = flag.String("etcdPrefix", "/conf", "Specifies the etcd prefix")
	watch      = flag.Bool("watch", true, "Watch for new values on etcd")
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

func generateConfig(kapi client.KeysAPI) (*client.Response, error) {
	resp, err := kapi.Get(context.Background(), *etcdPrefix, &client.GetOptions{Recursive: true})
	if err != nil {
		return resp, err
	}
	traverseConfigDirectory(resp.Node)
	return resp, nil
}

func generateConfigWatcher(kapi client.KeysAPI, resp *client.Response) (*client.Response, error) {

	watcher := kapi.Watcher(*etcdPrefix, &client.WatcherOptions{Recursive: true, AfterIndex: resp.Index})
	ctx := context.Background()

	resp, err := watcher.Next(ctx)
	if err != nil {
		return resp, err
	}
	return generateConfig(kapi)

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

func main() {
	flag.Parse()

	exists, err := dirExists(*outputDir)
	if err != nil {
		log.Fatal(err)
	}

	if !exists {
		if err := os.Mkdir(*outputDir, 0755); err != nil {
			log.Fatal(err)
		}
	}

	cfg := client.Config{
		Endpoints: []string{*endpoint},
		Transport: client.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
	}

	c, err := client.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := c.Sync(context.Background()); err != nil {
		log.Fatal(err)
	}

	kapi := client.NewKeysAPI(c)

	resp, err := generateConfig(kapi)
	if err != nil {
		log.Fatal(err)
	}
	if systemdutil.IsRunningSystemd() {
		err := daemon.SdNotify("READY=1")
		if err != nil {
			log.Printf("failed to notify systemd for readiness: %v", err)
			if err == daemon.SdNotifyNoSocket {
				log.Printf("forgot to set Type=notify in systemd service file?")
			}
		}
	}
	if *watch {
		for {
			resp, err = generateConfigWatcher(kapi, resp)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	os.Exit(0)
}
