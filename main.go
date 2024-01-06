package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"syscall"
)

const (
	helmBinary         = "/usr/bin/helm"
	kubeconfig         = "/home/nonroot/.kube/config"
	relocationsMapping = "/cnab/app/relocation-mapping.json"
	chartName          = "martianbank"
	chartDir           = "/cnab/app/charts/" + chartName
	timeOut            = "600s"
)

type image struct {
	repo, tag string
}

func main() {

	// get action from environment variable
	cnabAction, exists := os.LookupEnv("CNAB_ACTION")
	if !exists {
		cnabAction = "install"
	}
	fmt.Printf("CNAB_ACTION: %s\n", cnabAction)

	// get parameters from environment variables
	/*
		parameters:
		  - name: namespace
		    type: string
		    env: namespace
		    default: "demo"
		  - name: http_port
		    type: string
		    env: http_port
		    default: "8080"
		  - name: storage_size
		    type: string
		    env: storage_size
		    default: "100Mi"
	*/

	namespace, exists := os.LookupEnv("namespace")
	if !exists {
		namespace = "demo"
	}
	fmt.Printf("namespace: %s\n", namespace)
	httpPort, exists := os.LookupEnv("http_port")
	if !exists {
		httpPort = "8080"
	}
	fmt.Printf("http_port: %s\n", httpPort)
	storageSize, exists := os.LookupEnv("storage_size")
	if !exists {
		storageSize = "100Mi"
	}
	fmt.Printf("storage_size: %s\n", storageSize)

	// get relocations from json file
	/*
		images:
		  accounts:
		    imageType: docker
		    repository: 192.168.14.200:5080/docker/martian-bank-demo-accounts
		    tag: "latest"
		    digest: sha256:e54fb290a66c0966c30887a796970abe080f19c8c98804c53783fd57ae8155aa
		  locator:
		    imageType: docker
		    repository: 192.168.14.200:5080/docker/martian-bank-demo-atm-locator
		    tag: "latest"
		    digest: sha256:177c89df6be15681ebaeb9b98eb0d3eaac89cffaf18b59abbbd626051359ae9a
		  auth:
		    imageType: docker
		    repository: 192.168.14.200:5080/docker/martian-bank-demo-customer-auth
		    tag: "latest"
		    digest: sha256:1415f540fb5e3aa208ef93efad9f695c87495e9c3f4caf1a7c6b8efe36d74a39
		  dashboard:
		    imageType: docker
		    repository: 192.168.14.200:5080/docker/martian-bank-demo-dashboard
		    tag: "latest"
		    digest: sha256:a2b242dbddc590de8faa1cbca1f3ae58fff8e0c53accbda567fbd931be062818
		  loan:
		    imageType: docker
		    repository: 192.168.14.200:5080/docker/martian-bank-demo-loan
		    tag: "latest"
		    digest: sha256:452553e4ab53d9b333071db04757ea8e08b99aed146b5d39d55172f357f42797
		  mongo:
		    imageType: docker
		    repository: 192.168.14.200:5080/docker/mongo
		    tag: "latest"
		    digest: sha256:9f0d0ef54799cd17e4338c5c4da75565d2816a20d0202507fa51bca078a0d593
		  nginx:
		    imageType: docker
		    repository: 192.168.14.200:5080/docker/martian-bank-demo-nginx
		    tag: "latest"
		    digest: sha256:3ebbd2de11475e4a7088dc8aff11a10fb6a6dc34e0c26bc724144e144aa626d9
		  transactions:
		    imageType: docker
		    repository: 192.168.14.200:5080/docker/martian-bank-demo-transactions
		    tag: "latest"
		    digest: sha256:fbb606011a999467550d302570dcf7f0e2c17572b26d7096ee1bf02b5ef35f8c
		  ui:
		    imageType: docker
		    repository: 192.168.14.200:5080/docker/martian-bank-demo-ui
		    tag: "latest"
		    digest: sha256:10b459e366a1c07cb93278bc688a8941e11905065308bd3a3b74a19e52ffa08c
	*/

	// images from porter manifest
	images := map[string]string{
		"accounts":     "192.168.14.200:5080/docker/martian-bank-demo-accounts@sha256:e54fb290a66c0966c30887a796970abe080f19c8c98804c53783fd57ae8155aa",
		"auth":         "192.168.14.200:5080/docker/martian-bank-demo-customer-auth@sha256:1415f540fb5e3aa208ef93efad9f695c87495e9c3f4caf1a7c6b8efe36d74a39",
		"dashboard":    "192.168.14.200:5080/docker/martian-bank-demo-dashboard@sha256:a2b242dbddc590de8faa1cbca1f3ae58fff8e0c53accbda567fbd931be062818",
		"loan":         "192.168.14.200:5080/docker/martian-bank-demo-loan@sha256:452553e4ab53d9b333071db04757ea8e08b99aed146b5d39d55172f357f42797",
		"locator":      "192.168.14.200:5080/docker/martian-bank-demo-atm-locator@sha256:177c89df6be15681ebaeb9b98eb0d3eaac89cffaf18b59abbbd626051359ae9a",
		"mongo":        "192.168.14.200:5080/docker/mongo@sha256:9f0d0ef54799cd17e4338c5c4da75565d2816a20d0202507fa51bca078a0d593",
		"nginx":        "192.168.14.200:5080/docker/martian-bank-demo-nginx@sha256:3ebbd2de11475e4a7088dc8aff11a10fb6a6dc34e0c26bc724144e144aa626d9",
		"transactions": "192.168.14.200:5080/docker/martian-bank-demo-transactions@sha256:fbb606011a999467550d302570dcf7f0e2c17572b26d7096ee1bf02b5ef35f8c",
		"ui":           "192.168.14.200:5080/docker/martian-bank-demo-ui@sha256:10b459e366a1c07cb93278bc688a8941e11905065308bd3a3b74a19e52ffa08c",
	}

	// relocations map where key is image name and value is image struct
	relocations := make(map[string]image)

	// get relocations from json file
	jsonFile, err := os.Open(relocationsMapping)
	if err != nil {
		fmt.Printf("Can't opening relocation mapping: %s", err)

		// if relocations mapping file does not exist, use images from porter manifest
		for microservice, original := range images {
			// split image name to repo and tag
			parts := strings.Split(original, "@")
			// add image to relocations map
			relocations[microservice] = image{
				repo: parts[0],
				tag:  parts[1],
			}
		}
	} else {
		// get relocations from json file
		defer jsonFile.Close()

		byteValue, err := io.ReadAll(jsonFile)
		if err != nil {
			log.Fatalf("Error reading relocation mapping: %s", err)
		}

		// convert json to map where key is original image name and value is relocated image name
		var rawmap map[string]interface{}
		err = json.Unmarshal([]byte(byteValue), &rawmap)
		if err != nil {
			log.Fatalf("Error unmarshalling relocation mapping: %s", err)
		}

		// for each image in porter manifest
		for microservice, original := range images {
			// if image is in relocations map
			if _, ok := rawmap[original]; ok {

				relocatedImage := rawmap[original].(string)
				// split image name to repo and tag
				parts := strings.Split(relocatedImage, "@")
				// add image to relocations map
				relocations[microservice] = image{
					repo: parts[0],
					tag:  parts[1],
				}
			}
		}
	}
	fmt.Printf("relocations: %+v\n", relocations)

	// generate helm command
	/*
		#/usr/local/bin/helm3 /usr/local/bin/helm3 upgrade \
		#  --install martianbank charts/martianbank \
		#  --namespace demo --wait --values charts/martianbank/values.yaml \
		#  --timeout 600s --atomic --create-namespace \
		#  --set http_port=8080 \
		#  --set images.accounts.repository=192.168.14.200:5080/docker/martian-bank-demo-accounts \
		#  --set images.accounts.tag=latest@sha256:e54fb290a66c0966c30887a796970abe080f19c8c98804c53783fd57ae8155aa \
		#  --set images.auth.repository=192.168.14.200:5080/docker/martian-bank-demo-customer-auth \
		#  --set images.auth.tag=latest@sha256:1415f540fb5e3aa208ef93efad9f695c87495e9c3f4caf1a7c6b8efe36d74a39 \
		#  --set images.dashboard.repository=192.168.14.200:5080/docker/martian-bank-demo-dashboard \
		#  --set images.dashboard.tag=latest@sha256:a2b242dbddc590de8faa1cbca1f3ae58fff8e0c53accbda567fbd931be062818 \
		#  --set images.loan.repository=192.168.14.200:5080/docker/martian-bank-demo-loan \
		#  --set images.loan.tag=latest@sha256:452553e4ab53d9b333071db04757ea8e08b99aed146b5d39d55172f357f42797 \
		#  --set images.locator.repository=192.168.14.200:5080/docker/martian-bank-demo-atm-locator \
		#  --set images.locator.tag=latest@sha256:177c89df6be15681ebaeb9b98eb0d3eaac89cffaf18b59abbbd626051359ae9a \
		#  --set images.nginx.repository=192.168.14.200:5080/docker/martian-bank-demo-nginx \
		#  --set images.nginx.tag=latest@sha256:3ebbd2de11475e4a7088dc8aff11a10fb6a6dc34e0c26bc724144e144aa626d9 \
		#  --set images.transactions.repository=192.168.14.200:5080/docker/martian-bank-demo-transactions \
		#  --set images.transactions.tag=latest@sha256:fbb606011a999467550d302570dcf7f0e2c17572b26d7096ee1bf02b5ef35f8c \
		#  --set images.ui.repository=192.168.14.200:5080/docker/martian-bank-demo-ui \
		#  --set images.ui.tag=latest@sha256:10b459e366a1c07cb93278bc688a8941e11905065308bd3a3b74a19e52ffa08c \
		#  --set storage_size=100Mi
	*/

	var args []string
	if cnabAction == "install" {
		fmt.Println("\n------helm install------")
		args = []string{
			"helm",
			"upgrade",
			//"--debug",
			"--install",
			chartName,
			chartDir,
			"--namespace",
			namespace,
			"--wait",
			"--values",
			chartDir + "/values.yaml",
			"--timeout",
			timeOut,
			"--atomic",
			"--create-namespace",
			"--set",
			"http_port=" + httpPort,
			"--set",
			"storage_size=" + storageSize,
			"--set",
			"images.accounts.repository=" + relocations["accounts"].repo,
			"--set",
			"images.accounts.tag=latest@" + relocations["accounts"].tag,
			"--set",
			"images.auth.repository=" + relocations["auth"].repo,
			"--set",
			"images.auth.tag=latest@" + relocations["auth"].tag,
			"--set",
			"images.dashboard.repository=" + relocations["dashboard"].repo,
			"--set",
			"images.dashboard.tag=latest@" + relocations["dashboard"].tag,
			"--set",
			"images.loan.repository=" + relocations["loan"].repo,
			"--set",
			"images.loan.tag=latest@" + relocations["loan"].tag,
			"--set",
			"images.locator.repository=" + relocations["locator"].repo,
			"--set",
			"images.locator.tag=latest@" + relocations["locator"].tag,
			"--set",
			"images.nginx.repository=" + relocations["nginx"].repo,
			"--set",
			"images.nginx.tag=latest@" + relocations["nginx"].tag,
			"--set",
			"images.transactions.repository=" + relocations["transactions"].repo,
			"--set",
			"images.transactions.tag=latest@" + relocations["transactions"].tag,
			"--set",
			"images.ui.repository=" + relocations["ui"].repo,
			"--set",
			"images.ui.tag=latest@" + relocations["ui"].tag,
		}
	} else if cnabAction == "uninstall" {
		fmt.Println("\n------helm uninstall------")
		args = []string{
			"helm",
			"uninstall",
			//"--debug",
			chartName,
			"--namespace",
			namespace,
			"--wait",
			"--timeout",
			timeOut,
		}
	}

	os.Setenv("KUBECONFIG", kubeconfig)
	env := os.Environ()
	execErr := syscall.Exec(helmBinary, args, env)
	if execErr != nil {
		log.Fatalf("Error executing helm command: %s", execErr)
	}

}
