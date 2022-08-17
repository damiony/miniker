package networks

import (
	"fmt"
	"os"
	"strconv"
)

func listNetworks() {
	maxSize := map[string]int{}
	for i := range networks {
		if maxSize["name"] < len(networks[i].Name) {
			maxSize["name"] = len(networks[i].Name)
		}
		if maxSize["ipr"] < len(networks[i].IpRange.String()) {
			maxSize["ipr"] = len(networks[i].IpRange.String())
		}
		if maxSize["driver"] < len(networks[i].Driver) {
			maxSize["dirver"] = len(networks[i].Driver)
		}
	}

	netFormat := "%-" + strconv.Itoa(maxSize["name"]) + "s\t " +
		"%-" + strconv.Itoa(maxSize["ipr"]) + "s\t " +
		"%-" + strconv.Itoa(maxSize["driver"]) + "s\n"

	fmt.Fprintf(os.Stdout, netFormat, "Name", "IpRange", "Driver")
	for i := range networks {
		fmt.Fprintf(os.Stdout, netFormat, networks[i].Name, networks[i].IpRange, networks[i].Driver)
	}
}
