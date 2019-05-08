package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/ghodss/yaml"
)

const (
	name    = "terraform-kops-get-existing-subnets"
	version = "0.1"
)

type terraformOutput struct {
	AvailabilityZones map[string]interface{} `json:"azs"`
	PrivateSubnetIDs  map[string]interface{} `json:"private_subnets"`
	PublicSubnetIDs   map[string]interface{} `json:"public_subnets"`
	NATGatewayIDs     map[string]interface{} `json:"natgw_ids"`
}

type subnetSpec struct {
	Name       string `json:"name,omitempty"`
	Zone       string `json:"zone,omitempty"`
	CIDR       string `json:"cidr,omitempty"`
	ProviderID string `json:"id,omitempty"`
	Egress     string `json:"egress,omitempty"`
	Type       string `json:"type,omitempty"`
}

type subnetSpecs struct {
	Subnets []subnetSpec `json:"subnets"`
}

func main() {
	flag.Parse()

	var tfJSON []byte
	var err error
	if flag.NArg() >= 1 {
		tfJSON, err = ioutil.ReadFile(flag.Arg(0))
	} else {
		tfJSON, err = ioutil.ReadAll(os.Stdin)
	}

	if err != nil {
		log.Fatal(err)
	}

	var tfOut terraformOutput
	err = json.Unmarshal(tfJSON, &tfOut)
	if err != nil {
		log.Fatal(err)
	}

	azs := getValues(tfOut.AvailabilityZones)
	azCount := len(azs)
	privateSubnets := getValues(tfOut.PrivateSubnetIDs)
	publicSubnets := getValues(tfOut.PublicSubnetIDs)
	natGateways := getValues(tfOut.NATGatewayIDs)

	subnets := make([]subnetSpec, azCount*2)
	for i, subnetID := range privateSubnets {
		// accept either a single natGateway
		// or one for each private subnet.
		var natGateway = natGateways[0]
		if len(natGateways) > 1 {
			natGateway = natGateways[i]
		}

		subnets[i] = subnetSpec{
			ProviderID: subnetID,
			Egress:     natGateway,
			Name:       azs[i],
			Type:       "Private",
			Zone:       azs[i],
		}
	}

	for i, subnetID := range publicSubnets {
		idx := i + azCount
		subnets[idx] = subnetSpec{
			ProviderID: subnetID,
			Name:       "utility-" + azs[i],
			Type:       "Utility",
			Zone:       azs[i],
		}
	}

	subnetSpecs := subnetSpecs{Subnets: subnets}
	data, err := yaml.Marshal(subnetSpecs)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", string(data))
}

func getValues(m map[string]interface{}) []string {
	if rawv, ok := m["value"]; ok {
		if slice, ok := rawv.([]interface{}); ok {
			vals := make([]string, len(slice))
			for i, val := range slice {
				vals[i] = val.(string)
			}
			return vals
		}
	}
	return []string{}
}
