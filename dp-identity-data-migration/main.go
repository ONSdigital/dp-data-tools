package main

import (
	"fmt"

	"github.com/ONSdigital/dp-data-tools@v0.0.0-20220426125527-a81d87f6fdb3/dp-identity-data-migration/groupsdataextraction"
)

func main() {
	fmt.Println("Hello World")

	groupsdataextraction.ExtractGroupData()
}
