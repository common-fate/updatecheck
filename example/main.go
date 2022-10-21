package main

import "github.com/common-fate/updatecheck"

func main() {
	updatecheck.Check(updatecheck.GrantedCLI, "v0.2.0", false)
	updatecheck.Print()
}
